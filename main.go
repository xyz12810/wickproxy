package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	version = "0.0.1"
)

var (
	log        = logrus.New()
	versionStr = "Version: " + version + " (platform: " + runtime.GOOS + "-" + runtime.GOARCH + ")"

	debug  = kingpin.Flag("debug", "set log level to 'debug'").Short('d').Default("false").Bool()
	config = kingpin.Flag("config", "set config database").Short('c').Default("config.json").String()

	initCmd   = kingpin.Command("init", "create a initial copy of config file")
	initForce = initCmd.Flag("force", "force to overwrite config file").Short('f').Default("false").Bool()

	showCmd = kingpin.Command("show", "show the config file")

	setCmd   = kingpin.Command("set", "change settings in config file")
	setKey   = setCmd.Arg("key", "keywords could be one of `server`, `timeout`, `secure_url`, `fallback_url`, `cert` and `key`").String()
	setValue = setCmd.Arg("value", "").String()

	userAddCmd      = kingpin.Command("user-add", "create a new user")
	userAddUsername = userAddCmd.Arg("username", "username").Required().String()
	userAddPassword = userAddCmd.Arg("password", "password").Required().String()
	userAddQuota    = userAddCmd.Arg("quota", "quota ( 0 for no limitation.").Default("0").Int()
	userAddForce    = userAddCmd.Flag("force", "force to add or update a user").Short('f').Default("false").Bool()

	userDelCmd      = kingpin.Command("user-del", "delete user(s)")
	userDelUsername = userDelCmd.Arg("username", "username").String()
	userDelAll      = userDelCmd.Flag("all", "use `-a` to clear the Users list.").Short('a').Default("false").Bool()

	aclAddCmd        = kingpin.Command("acl-add", "insert a new rule into ACL")
	aclAddCmdIndex   = aclAddCmd.Flag("index", "index number to insert a new rule").Short('i').Default("-1").Int()
	aclAddCmdContext = aclAddCmd.Arg("context", "domain:port or IP:port or CIDR. Example: `192.168.0.0/16`, `google.com`, `1.2.3.4:443` or `baidu.com:443`").Required().String()
	aclAddCmdAction  = aclAddCmd.Arg("action", "one of the 'allow' or 'deny'").Required().String()

	aclDelCmd      = kingpin.Command("acl-del", "delete a rule from ACL")
	aclDelCmdAll   = aclDelCmd.Flag("all", "clear all rules in ACL").Short('a').Default("false").Bool()
	aclDelCmdIndex = aclDelCmd.Arg("index", "index number to delete a rule. Default is to delete the last rule").Default("-1").Int()

	aclListCmd = kingpin.Command("acl-list", "show the ACL list")

	runCmd    = kingpin.Command("run", "run the wickproxy server")
	runServer = runCmd.Arg("server", "server address.").String()

	versionCMD = kingpin.Command("version", "print the version")
)

func logInit(loglevel logrus.Level) {
	log.SetLevel(loglevel)
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.TextFormatter{
		ForceColors:               true,
		TimestampFormat:           "2006-01-02 15:04:05",
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		// DisableLevelTruncation:true,
	})
}

func cmdInit() {
	kingpin.CommandLine.Help = "Wickproxy is a security HTTP(s) proxy for all platforms. " + versionStr
	kingpin.CommandLine.Name = "Wickproxy"
	kingpin.CommandLine.Version(versionStr)
}

func main() {
	cmdInit()
	cmd := kingpin.Parse()
	if *debug == true {
		logInit(logrus.DebugLevel)
	} else {
		logInit(logrus.InfoLevel)
	}

	switch cmd {
	case "init":
		initHandle()
	case "show":
		showHandle()
	case "set":
		setHandle()
	case "user-add":
		useraddHandle()
	case "user-del":
		userdelHandle()
	case "acl-add":
		acladdHandle()
	case "acl-del":
		acldelHandle()
	case "acl-list":
		acllist()
	case "run":
		serverHandle()
	case "version":
		versionHandler()
	}
}

func versionHandler() {
	fmt.Println(versionStr)
}

func initHandle() {
	log.Debug("[init] Init config from:", *config)

	if configExists(*config) && !*initForce {
		log.Errorln("[init] configuration file exists. Use --force true to overwrite it.")
		return
	}

	err := configWriter(*config)
	if err != nil {
		log.Fatal("[init] write to file error:", err)
	}
}

func showHandle() {
	log.Debugln("[show] show configuation file:", *config)
	err := configReader(*config)
	if err != nil {
		log.Fatal("[show] read config error:", err)
	}
	err = configPrint()
	if err != nil {
		log.Fatalln("[show] read config error:", err)
	}
}

func setHandle() {
	err := configReader(*config)
	if err != nil {
		log.Fatalln("[set] read config error:", err)
	}

	switch *setKey {
	case "server":
		GlobalConfig.Server = *setValue
	case "timeout":
		t, err := strconv.ParseInt(*setValue, 10, 64)
		if err != nil {
			log.Fatalln("[cmd] timeout invalid!", *setKey)
		}
		GlobalConfig.Timeout = time.Duration(t)
	case "secure_url":
		GlobalConfig.SecureURL = *setValue
	case "fallback_url":
		GlobalConfig.FallbackURL = *setValue
	case "cert":
		GlobalConfig.TLS.Certificate = *setValue
	case "key":
		GlobalConfig.TLS.CertificateKey = *setValue
	default:
		log.Fatalln("[cmd] no such key:", *setKey)
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
		log.Fatalln("[user] read config error:", err)
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

		log.Debugln("[user] update a existed user")
	} else {
		GlobalConfig.Users = append(GlobalConfig.Users,
			userConfig{
				Username: *userAddUsername,
				Password: *userAddPassword,
				Quato:    *userAddQuota})
		log.Debugln("[user] add a new user")
	}

	err = configWriter(*config)
	if err != nil {
		log.Fatalln("[user] write to file error:", err)
	}
}

func userdelHandle() {
	err := configReader(*config)
	if err != nil {
		log.Fatalln("[user] read config error:", err)
	}

	if *userDelAll {
		GlobalConfig.Users = make([]userConfig, 0)
		err = configWriter(*config)
		if err != nil {
			log.Fatal("[user] write to file error:", err)
		}
		log.Debugln("[user] delete all users")
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
		log.Fatalln("[user] write to file error:", err)
	}
	log.Debugln("[user] delete user:", *userDelUsername)
}

func acladdHandle() {
	err := configReader(*config)
	if err != nil {
		log.Fatalln("[user] read config error:", err)
	}

	idx := *aclAddCmdIndex
	context := *aclAddCmdContext
	action := *aclAddCmdAction

	aclAdd(idx, context, action)

	err = configWriter(*config)
	if err != nil {
		log.Fatalln("[user] write to file error:", err)
	}
}

func acldelHandle() {
	err := configReader(*config)
	if err != nil {
		log.Fatalln("[user] read config error:", err)
	}

	idx := *aclDelCmdIndex
	all := *aclDelCmdAll
	aclDel(idx, all)

	err = configWriter(*config)
	if err != nil {
		log.Fatalln("[user] write to file error:", err)
	}
}
