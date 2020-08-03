// +build darwin linux freebsd

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

var (
	restartSig = make(chan bool)
)

func runHandle() {

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
				configWriter(*configFlag)
				logFile.Close()
				os.Exit(0)
			case syscall.SIGUSR2:
				// reload configuration file
				log.Infoln("[signal] reload configuration file")
				err := configReader(*configFlag)
				if err != nil {
					log.Infoln("[cmd] reload configuration file found error:", err)
					return
				}

				// reload logging
				logInit(runCmd.FullCommand())

				// restart server
				if currentServer != nil {
					log.Infoln("[signal] shotdown server ...")
					ctx := new(context.Context)
					if err := currentServer.Shutdown(*ctx); err != nil {
						log.Fatalln("[signal] shutdown server error,", err)
					}
				}
			}
		}
	}()

	for {
		serverHandle()
	}
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
