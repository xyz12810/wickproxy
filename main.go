package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	version = "v0.1.1-beta"
)

var (
	log        = logrus.New()
	versionStr = "Version: " + version + " (platform: " + runtime.GOOS + "-" + runtime.GOARCH + ")."
	logFile    os.File

	debug  = kingpin.Flag("debug", "Set log level to 'debug'.").Short('d').Default("false").Bool()
	config = kingpin.Flag("config", "special configuration file.").Short('c').Default("config.json").String()
	logf   = kingpin.Flag("log", "logging file").Short('l').String()

	initCmd   = kingpin.Command("init", "Create a initial copy of configuration file.")
	initForce = initCmd.Flag("force", "Force to overwrite configuration file.").Short('f').Default("false").Bool()

	showCmd = kingpin.Command("show", "Show the configuration file.")

	setCmd   = kingpin.Command("set", "Change settings in configuration file.")
	setKey   = setCmd.Arg("key", "Keywords could be one of `server`, `timeout`, `secure_url`, `fallback_url`, `cert` and `key`.").String()
	setValue = setCmd.Arg("value", "").String()

	userAddCmd      = kingpin.Command("user-add", "Create a new user.")
	userAddUsername = userAddCmd.Arg("username", "Username.").Required().String()
	userAddPassword = userAddCmd.Arg("password", "Password.").Required().String()
	userAddQuota    = userAddCmd.Arg("quota", "Quota ( 0 for no limitation.").Default("0").Int()
	userAddForce    = userAddCmd.Flag("force", "Force to add or update a user.").Short('f').Default("false").Bool()

	userDelCmd      = kingpin.Command("user-del", "Delete user(s).")
	userDelUsername = userDelCmd.Arg("username", "Username.").String()
	userDelAll      = userDelCmd.Flag("all", "Use `-a` to clear the Users list.").Short('a').Default("false").Bool()

	aclAddCmd        = kingpin.Command("acl-add", "Insert a new rule into ACL.")
	aclAddCmdIndex   = aclAddCmd.Flag("index", "Index number to insert a new rule.").Short('i').Default("-1").Int()
	aclAddCmdContext = aclAddCmd.Arg("context", "Rule details. domain:port or IP:port or CIDR. Example: `192.168.0.0/16`, `google.com`, `1.2.3.4:443` or `baidu.com:443`.").Required().String()
	aclAddCmdAction  = aclAddCmd.Arg("action", "One of the 'allow' or 'deny'.").Required().String()

	aclDelCmd      = kingpin.Command("acl-del", "Delete a rule from ACL.")
	aclDelCmdAll   = aclDelCmd.Flag("all", "Clear all rules in ACL.").Short('a').Default("false").Bool()
	aclDelCmdIndex = aclDelCmd.Arg("index", "Index number to delete a rule. Default is to delete the last rule.").Default("-1").Int()

	aclListCmd = kingpin.Command("acl-list", "Show the ACL list.")

	runCmd    = kingpin.Command("run", "Run the wickproxy server.")
	runServer = runCmd.Arg("server", "Server address. Example: `0.0.0.0:7890`.").String()

	startCmd   = kingpin.Command("start", "Run the wickproxy server as daemon service.")
	stopCmd    = kingpin.Command("stop", "stop a wickproxy  daemon  service")
	reloadCmd  = kingpin.Command("reload", "reload configuration file")
	versionCMD = kingpin.Command("version", "Print the version and platforms.")
)

func logInit(loglevel logrus.Level, logf string, cmd string) {
	log.SetLevel(loglevel)

	if logf == "" && cmd == runCmd.FullCommand() {
		logf = GlobalConfig.Logging
	}

	if logf == "" {
		log.SetOutput(os.Stdout)
		log.SetFormatter(&logrus.TextFormatter{
			ForceColors:               true,
			TimestampFormat:           "2006-01-02 15:04:05",
			EnvironmentOverrideColors: true,
			FullTimestamp:             true,
		})
	} else {
		logFile, err := os.Create(logf)
		if err != nil {
			log.Fatalln("[log] can not open(create) file", logf)
		}
		log.SetOutput(logFile)
		log.SetFormatter(&logrus.TextFormatter{
			TimestampFormat:           "2006-01-02 15:04:05",
			EnvironmentOverrideColors: true,
			FullTimestamp:             true,
		})
	}
}

func cmdInit() {
	kingpin.CommandLine.Help = "Wickproxy is a security HTTP(s) proxy for all platforms. " + versionStr
	kingpin.CommandLine.Name = "Wickproxy"
	kingpin.CommandLine.Version(versionStr)
}

func main() {
	// Command parse
	cmdInit()
	cmd := kingpin.Parse()

	// Read config
	err := configReader(*config)
	if err != nil {
		log.Println("[cmd] no configuration file found, create one.")
		initHandle()
		return
	}

	// Logging initial
	if *debug == true {
		logInit(logrus.DebugLevel, *logf, cmd)
	} else {
		logInit(logrus.InfoLevel, *logf, cmd)
	}
	defer logFile.Close()

	switch cmd {
	case initCmd.FullCommand():
		initHandle()
	case showCmd.FullCommand():
		showHandle()
	case setCmd.FullCommand():
		setHandle()
	case userAddCmd.FullCommand():
		useraddHandle()
	case userDelCmd.FullCommand():
		userdelHandle()
	case aclAddCmd.FullCommand():
		acladdHandle()
	case aclDelCmd.FullCommand():
		acldelHandle()
	case aclListCmd.FullCommand():
		acllist()
	case runCmd.FullCommand():
		runHandle()
	case startCmd.FullCommand():
		startHandle()
	case stopCmd.FullCommand(), reloadCmd.FullCommand():
		signHandle(cmd)
	case versionCMD.FullCommand():
		versionHandler()
	}
}

func versionHandler() {
	fmt.Println(versionStr)
}

func startHandle() {
	newArgs := make([]string, 0)
	for _, arg := range os.Args {
		if arg == startCmd.FullCommand() {
			newArgs = append(newArgs, "run")
		} else {
			newArgs = append(newArgs, arg)
		}
	}

	cmd := exec.Command(newArgs[0], newArgs[1:]...)
	if err := cmd.Start(); err != nil {
		log.Fatalf("[cmd] start %s failed, error: %v\n", newArgs[0], err)
	}

	fmt.Printf("[cmd] %s [PID] %d running...\n", newArgs[0], cmd.Process.Pid)
}

func runHandle() {
	GlobalConfig.PID = os.Getpid()
	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[cmd] write pid to config file error:", err)
	}

	// Signal Process
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		for {
			sig := <-c
			switch sig {
			case syscall.SIGINT, syscall.SIGKILL, syscall.SIGUSR1:
				log.Infoln("[signal] server quit!")
				GlobalConfig.PID = 0
				configWriter(*config)
				os.Exit(0)
			case syscall.SIGUSR2:
				log.Infoln("[signal] reload configuration file")
				err := configReader(*config)
				if err != nil {
					log.Infoln("[cmd] reload configuration file found error:", err)
					return
				}
			}
		}
	}()

	serverHandle()
}

func signHandle(cmd string) {
	if GlobalConfig.PID == 0 {
		log.Fatalln("[cmd] no server is running in the background")
	}
	if cmd == stopCmd.FullCommand() {
		syscall.Kill(GlobalConfig.PID, syscall.SIGUSR1)
		return
	} else if cmd == reloadCmd.FullCommand() {
		syscall.Kill(GlobalConfig.PID, syscall.SIGUSR2)
		return
	}
	log.Fatalln("[cmd] internal error:", cmd)
}

func initHandle() {
	log.Debug("[init] Init config from:", *config)

	if configExists(*config) && !*initForce {
		log.Panicln("[init] configuration file exists. Use --force true to overwrite it.")
		return
	}

	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[init] write to file error:", err)
	}
}

func showHandle() {
	err := configPrint()
	if err != nil {
		log.Fatalln("[show] read config error:", err)
	}
}

func setHandle() {

	switch *setKey {
	case "server":
		GlobalConfig.Server = *setValue
	case "logging":
		GlobalConfig.Logging = *setValue
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
	case "tls_cert":
		GlobalConfig.TLS.Certificate = *setValue
	case "tls_key":
		GlobalConfig.TLS.CertificateKey = *setValue
	default:
		log.Fatalln("[cmd] no such key:", *setKey)
	}

	err := configWriter(*config)
	if err != nil {
		log.Fatal("[set] write to file error:", err)
	}
	log.Debugf("[set] config changed: set %v = %v\n", *setKey, *setValue)
}

func useraddHandle() {
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

	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[user] write to file error:", err)
	}
}

func userdelHandle() {
	if *userDelAll {
		GlobalConfig.Users = make([]userConfig, 0)
		err := configWriter(*config)
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

	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[user] write to file error:", err)
	}
	log.Debugln("[user] delete user:", *userDelUsername)
}

func acladdHandle() {
	idx := *aclAddCmdIndex
	context := *aclAddCmdContext
	action := *aclAddCmdAction

	aclAdd(idx, context, action)

	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[user] write to file error:", err)
	}
}

func acldelHandle() {
	idx := *aclDelCmdIndex
	all := *aclDelCmdAll
	aclDel(idx, all)

	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[user] write to file error:", err)
	}
}
