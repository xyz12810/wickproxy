package main

import "runtime"

var (
	version    string = "0.1.2-beta"
	builddate  string = ""
	nameStr   = "wickproxy " + version 
	versionStr        = "Version: " + version + " build: " + builddate + " (platform: " + runtime.GOOS + "-" + runtime.GOARCH + ")."
)
