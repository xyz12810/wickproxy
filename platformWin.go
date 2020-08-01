// +build windows

package main

import "os"

func runHandle() {
	if GlobalConfig.PID != 0 {
		log.Fatalln("[cmd] there is a wickproxy running. quit!")
	}

	GlobalConfig.PID = os.Getpid()
	err := configWriter(*config)
	if err != nil {
		log.Fatalln("[cmd] write pid to config file error:", err)
	}

	serverHandle()
}

func signHandle(cmd string) {
	log.Println("[cmd] only usable for unix/linux systems")
}
