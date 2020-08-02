package main

import "runtime"

var (
	version    string = "0.1.2-beta"
	builddate  string = ""
	versionStr        = "Version: " + version + " build: " + builddate + " (platform: " + runtime.GOOS + "-" + runtime.GOARCH + ")."
)
