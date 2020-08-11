package main

import (
	"errors"
	oldlog "log"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
	"strconv"
	"time"

	midlog "github.com/improbable-eng/go-httpwares/logging/logrus"
	"github.com/sirupsen/logrus"
)

const defaultServer = "0.0.0.0:7890"

var (
	currentServer *http.Server
	loggerAdapter *oldlog.Logger
)

type proxyServer struct{}

func (p *proxyServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	defaultServerHandler(w, req)
}

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

	// lock PID lock
	if GlobalConfig.PID != 0 {
		log.Fatalln("[cmd] there is a wickproxy running. quit! If wickproxy has quit by accident, use `wickproxy unlock`.")
	}

	GlobalConfig.PID = os.Getpid()
	err := configWriter(*configFlag)
	if err != nil {
		log.Fatalln("[cmd] write pid to config file error:", err)
	}

	// pool initial
	poolInit()

	//logger adapter initial
	ent := logrus.NewEntry(log)
	loggerAdapter = midlog.AsHttpLogger(ent)

	// reverse proxy server init
	reverseProxyHandlerInit()

	server := getServer()
	log.Infoln("[server] listen at:", server)

	currentServer = &http.Server{
		Addr:         server,
		Handler:      &proxyServer{},
		ErrorLog:     loggerAdapter,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if GlobalConfig.TLS.Certificate != "" && GlobalConfig.TLS.CertificateKey != "" {
		err = currentServer.ListenAndServeTLS(GlobalConfig.TLS.Certificate, GlobalConfig.TLS.CertificateKey)
	} else {
		err = currentServer.ListenAndServe()
	}

	if err != nil {
		log.Infoln("[server] server closed:", err)
	}

	// unlock PID
	GlobalConfig.PID = 0
	err = configWriter(*configFlag)
	if err != nil {
		log.Infoln("[server] failed to unlock PID")
	}
}

func defaultServerHandler(w http.ResponseWriter, req *http.Request) {

	var host, port, hostport string
	hostport = req.URL.Host
	if hostport == "" {
		hostport = req.Host
	}
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
		if req.URL.Scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	// Authenticate
	ret, emptyAuth, username := authenticate(w, req)
	if emptyAuth {
		if checkWhiteList(host) {
			proxy407Handle(w, req)
			return
		}
		errorHandle(w, req, http.StatusNotFound, errors.New("Empty Proxy-Authorization"))
		return
	}

	// Authenticate Failed
	if ret == false {
		if checkWhiteList(host) {
			error403Handle(w, req, errors.New("Authenticate Failed"))
			return
		}
		errorHandle(w, req, http.StatusNotFound, errors.New("Authenticate Failed"))
		return
	}

	// Access Control List check
	if !aclPrivateCheck(host) {
		error403Handle(w, req, err)
		return
	}
	if !aclCheck(host, port) {
		error403Handle(w, req, err)
		return
	}

	// start to proxy
	log.Debugln("[proxy] user["+username+"]", req.Method, host, req.URL)

	// visit secureURL
	if GlobalConfig.SecureHost != "" && GlobalConfig.SecureHost == host {
		proxyPassHandle(w, req)
		return
	}

	// protocol version check
	if req.ProtoMajor != 1 && req.ProtoMajor != 2 {
		errorHandle(w, req, http.StatusHTTPVersionNotSupported, errors.New("Unsupported HTTP major version: "+strconv.Itoa(req.ProtoMajor)))
		return
	}

	// For http proxy
	if req.Method != http.MethodConnect {
		httpProxyHandler(w, req)
		return
	}

	// For http(s)(2) proxy
	httpsProxyHandle(w, req)
	return
}

func authenticate(w http.ResponseWriter, req *http.Request) (ret, emptyAuth bool, username string) {
	if len(GlobalConfig.Users) == 0 {
		return true, false, ""
	}

	reqUsername, reqPassword, ok := proxyAuth(req)
	if !ok {
		return false, true, ""
	}

	for _, u := range GlobalConfig.Users {
		if u.Username == reqUsername && u.Password == reqPassword {
			return true, false, reqUsername
		}
	}
	return false, false, ""
}

func httpProxyHandler(w http.ResponseWriter, req *http.Request) {
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	if req.URL.Host == "" {
		req.URL.Host = req.Host
	}

	tmpRPHandler := httputil.ReverseProxy{
		ErrorLog:     loggerAdapter,
		ErrorHandler: error404Handle,
		Director: func(req *http.Request) {
			removeHopByHop(req.Header)
		},
		ModifyResponse: func(w *http.Response) error {
			removeHopByHop(w.Header)
			return nil
		},
	}

	tmpRPHandler.ServeHTTP(w, req)
}

func httpsProxyHandle(w http.ResponseWriter, req *http.Request) {

	// hostPort
	hostPort := req.URL.Host
	if hostPort == "" {
		hostPort = req.Host
	}
	_, _, err := net.SplitHostPort(hostPort)
	if err != nil {
		if req.URL.Scheme == "http" {
			hostPort = net.JoinHostPort(hostPort, "80")
		} else {
			hostPort = net.JoinHostPort(hostPort, "443")
		}
	}

	// outbound
	outbound, err := dial(hostPort)
	if err != nil {
		error502Handle(w, req, err)
		return
	}
	defer outbound.Close()

	// Return 200
	wFlusher, ok := w.(http.Flusher)
	if !ok {
		error500Handle(w, req, errors.New("Server doesn't implement Flusher"))
	}
	responsePadding(w)
	w.Header().Set("Server", fakeServer)
	w.WriteHeader(http.StatusOK)
	wFlusher.Flush()

	switch req.ProtoMajor {
	case 1:
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			error500Handle(w, req, errors.New("Server does not implement Hijacker"))
			return
		}
		clientConn, bufReader, err := hijacker.Hijack()
		if err != nil {
			error500Handle(w, req, errors.New("failed to hijack: "+err.Error()))
			return
		}
		defer clientConn.Close()

		if bufReader != nil {
			// snippet borrowed from `proxy` plugin
			if n := bufReader.Reader.Buffered(); n > 0 {
				rbuf, err := bufReader.Reader.Peek(n)
				if err != nil {
					error500Handle(w, req, err)
					return
				}
				outbound.Write(rbuf)
			}
		}
		dualStream(outbound, clientConn, clientConn)
	case 2:
		defer req.Body.Close()
		dualStream(outbound, req.Body, w)
	default:
		error500Handle(w, req, errors.New("HTTP protocol verion error"))
	}
}
