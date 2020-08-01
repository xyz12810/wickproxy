// +build darwin linux freebsd

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func runHandle() {
	if GlobalConfig.PID != 0 {
		log.Fatalln("[cmd] there is a wickproxy running. quit!")
	}

	GlobalConfig.PID = os.Getpid()
	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[cmd] write pid to config file error:", err)
	}

	// Signal Process
	c := make(chan os.Signal)
	signal.Notify(c, syscall.SIGINT, syscall.SIGKILL, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)

	go func() {
		for {
			sig := <-c
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL, syscall.SIGUSR1:
				log.Infoln("[signal] server exit!")
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
