package main

import (
	"errors"
	"net/http"
)

const defaultServer = "localhost:7890"

func getServer() string {
	if *runServer != "" {
		return *runServer
	}
	if GlobalConfig.Server != "" {
		return GlobalConfig.Server
	}
	return defaultServer
}

func serverHandle() {

	var err error
	err = configReader(*config)
	if err != nil {
		log.Fatalln("[set] read config error:", err)
	}

	server := getServer()
	log.Infoln("[server] listen at:", server)
	http.HandleFunc("/", defaultServerHandler)

	if GlobalConfig.TLS.Certificate != "" && GlobalConfig.TLS.CertificateKey != "" {
		err = http.ListenAndServeTLS(server, GlobalConfig.TLS.Certificate, GlobalConfig.TLS.CertificateKey, nil)
	} else {
		err = http.ListenAndServe(server, nil)
	}
	log.Fatalln("[server] server fatal:", err)
}

func defaultServerHandler(w http.ResponseWriter, req *http.Request) {
	log.Debugln("[server] read request from ["+req.RemoteAddr+"]", req.Method, req.Host, req.URL)

	if req.Method != http.MethodConnect {
		errorHandle(w, req, 0, errors.New("Method not allowed"))
	}

	return
}
