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
	config = kingpin.Flag("config", "configuration database").Default("config.json").String()

	initCmd   = kingpin.Command("init", "initial the config database")
	initForce = initCmd.Flag("force", "force to overwrite config database").Default("false").Bool()

	runCmd    = kingpin.Command("run", "run the wickproxy server")
	runServer = runCmd.Arg("server", "server address").String()

	showCmd = kingpin.Command("show", "show the configruation database")
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
	case "run":
	}
}

func initHandle() {
	log.Debug("Init configuation file")

	if configExists(*config) && ! *initForce{
		log.Info("[init] configuration file exists. Use --force true to overwrite it.")
		return
	}

	err := configWriter(*config)
	if err != nil {
		log.Panic("[init] write to file error:", err)
	}
}

func showHandle() {
	err := configReader(*config)
	if err != nil {
		log.Panic("[get] read config error:", err)
	}
	err = configPrint()
	if err != nil {
		log.Panic("[get] read config error:", err)
	}
}
