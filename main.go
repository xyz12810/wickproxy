package main

import (
	"os"

	log "github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	version = "0.0.1"
)

var (
	debug  = kingpin.Flag("debug", "enable debug mode").Default("false").Bool()
	config = kingpin.Flag("config", "config database").Default("config.json").String()

	initCmd   = kingpin.Command("init", "initial the config database")
	initForce = initCmd.Flag("force", "force to overwrite config database").Default("false").Bool()

	showCmd = kingpin.Command("show", "show the configruation database")

	setCmd   = kingpin.Command("set", "set configuration")
	setKey   = setCmd.Arg("key", "").String()
	setValue = setCmd.Arg("Value", "").String()

	userAddCmd      = kingpin.Command("useradd", "add new users")
	userAddUsername = userAddCmd.Arg("username", "username").Required().String()
	userAddPassword = userAddCmd.Arg("password", "password").Required().String()
	userAddQuota    = userAddCmd.Arg("quota", "quota").Default("0").Int()
	userAddForce    = userAddCmd.Flag("force", "force to add or update a user").Default("false").Bool()

	userDelCmd      = kingpin.Command("userdel", "delete uses")
	userDelUsername = userDelCmd.Arg("username", "username").String()
	userDelAll      = userDelCmd.Flag("all", "delete all users").Default("false").Bool()

	runCmd    = kingpin.Command("run", "run the wickproxy server")
	runServer = runCmd.Arg("server", "server address").String()
)

func logInit(loglevel log.Level) {
	log.SetLevel(loglevel)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		ForceColors:               true,
		TimestampFormat:           "2006-01-02 15:04:05",
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		// DisableLevelTruncation:true,
	})
}

func main() {
	cmd := kingpin.Parse()
	if *debug == true {
		logInit(log.DebugLevel)
	} else {
		logInit(log.InfoLevel)
	}

	switch cmd {
	case "init":
		initHandle()
	case "show":
		showHandle()
	case "set":
		setHandle()
	case "useradd":
		useraddHandle()
	case "userdel":
		userdelHandle()
	case "run":
	}
}

func initHandle() {
	log.Debug("[init] Init configuation file:", *config)

	if configExists(*config) && !*initForce {
		log.Info("[init] configuration file exists. Use --force true to overwrite it.")
		return
	}

	err := configWriter(*config)
	if err != nil {
		log.Fatal("[init] write to file error:", err)
	}
}

func showHandle() {
	log.Debug("[show] show configuation file:", *config)
	err := configReader(*config)
	if err != nil {
		log.Fatal("[show] read config error:", err)
	}
	err = configPrint()
	if err != nil {
		log.Fatal("[show] read config error:", err)
	}
}

func setHandle() {
	err := configReader(*config)
	if err != nil {
		log.Fatal("[set] read config error:", err)
	}

	switch *setKey {
	case "server":
		GlobalConfig.Server = *setValue
	case "hide_url":
		GlobalConfig.HideURL = *setValue
	case "fail_url":
		GlobalConfig.FailURL = *setValue
	case "cert":
		GlobalConfig.TLS.Certificate = *setValue
	case "key":
		GlobalConfig.TLS.CertificateKey = *setValue
	default:
		log.Fatal("no such key:", *setKey)
	}

	err = configWriter(*config)
	if err != nil {
		log.Fatal("[set] write to file error:", err)
	}
	log.Debugf("[set] config changed: set %v = %v\n", *setKey, *setValue)
}

func useraddHandle() {
	err := configReader(*config)
	if err != nil {
		log.Fatal("[user] read config error:", err)
	}

	tmpIdx := -1
	for i, v := range GlobalConfig.Users {
		if v.Username == *userAddUsername && !*userAddForce {
			log.Fatal("[user] username duplicated. use --foce to overwrite.")
		} else if v.Username == *userAddUsername {
			tmpIdx = i
		}
	}

	if tmpIdx != -1 {
		GlobalConfig.Users[tmpIdx] = userConfig{
			Username: *userAddUsername,
			Password: *userAddPassword,
			Quato:    *userAddQuota}

		log.Debug("[user] update a existed user")
	} else {
		GlobalConfig.Users = append(GlobalConfig.Users,
			userConfig{
				Username: *userAddUsername,
				Password: *userAddPassword,
				Quato:    *userAddQuota})
		log.Debug("[user] add a new user")
	}

	err = configWriter(*config)
	if err != nil {
		log.Fatal("[user] write to file error:", err)
	}
}

func userdelHandle() {
	err := configReader(*config)
	if err != nil {
		log.Fatal("[user] read config error:", err)
	}

	if *userDelAll {
		GlobalConfig.Users = make([]userConfig, 0)
		err = configWriter(*config)
		if err != nil {
			log.Fatal("[user] write to file error:", err)
		}
		log.Debug("[user] delete all users")
		return
	}

	tmpUsers := make([]userConfig, 0)
	for _, v := range GlobalConfig.Users {
		if v.Username != *userDelUsername {
			tmpUsers = append(tmpUsers, v)
		}
	}
	GlobalConfig.Users = tmpUsers

	err = configWriter(*config)
	if err != nil {
		log.Fatal("[user] write to file error:", err)
	}
	log.Debug("[user] delete user:", *userDelUsername)
}
