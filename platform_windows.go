// +build windows

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

	c := make(chan os.Signal)
	signal.Notify(c, os.Kill)

	go func() {
		for {
			sig := <-c
			switch sig {
			case syscall.SIGINT:
				log.Infoln("[signal] server exit!")
				GlobalConfig.PID = 0
				configWriter(*config)
				os.Exit(0)
			}
		}
	}()

	serverHandle()
}

func signHandle(cmd string) {
	log.Fatalln("[cmd] only usable for unix/linux systems")
}
