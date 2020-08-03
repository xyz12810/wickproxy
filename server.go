package main

import (
	"crypto/tls"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"
)

const defaultServer = "0.0.0.0:7890"

var (
	currentServer *http.Server
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

	server := getServer()
	log.Infoln("[server] listen at:", server)

	currentServer = &http.Server{
		Addr:    server,
		Handler: &proxyServer{}}

	if GlobalConfig.HTTP2 == false {
		currentServer.TLSConfig = &tls.Config{NextProtos: []string{"http/1.1"}}
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
		error502Handle(w, req, err)
		return
	}
	if !aclCheck(host, port) {
		error502Handle(w, req, err)
		return
	}

	// visit secureURL
	if GlobalConfig.SecureURL != "" && GlobalConfig.SecureURL == host {
		proxyPassHandle(w, req)
		return
	}

	// protocol version check
	if req.ProtoMajor != 1 && req.ProtoMajor != 2 {
		errorHandle(w, req, http.StatusHTTPVersionNotSupported, errors.New("Unsupported HTTP major version: "+strconv.Itoa(req.ProtoMajor)))
		return
	}

	// start to proxy
	log.Debugln("[proxy] user["+username+"]", req.Method, req.Host, req.URL)

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

// handle HTTP Proxy
func httpProxyHandler(w http.ResponseWriter, req *http.Request) {
	var err error

	// step 1: build request
	transport := http.DefaultTransport
	outReq := new(http.Request)
	outReq = req.Clone(req.Context())
	if outReq.URL.Scheme == "" {
		outReq.URL.Scheme = "http"
	}
	if outReq.URL.Host == "" {
		outReq.URL.Host = outReq.Host
	}
	outReq.Proto = "HTTP/1.1"
	outReq.ProtoMajor = 1
	outReq.ProtoMinor = 1
	removeHopByHop(outReq.Header)

	res, err := transport.RoundTrip(outReq)
	if err != nil {
		error500Handle(w, req, err)
		return
	}

	removeHopByHop(res.Header)
	for key, value := range res.Header {
		for _, v := range value {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
	res.Body.Close()
}

func httpsProxyHandle(w http.ResponseWriter, req *http.Request) {
	hostPort := req.URL.Host
	Port := req.URL.Port()
	if Port == "" {
		if req.URL.Scheme == "http" {
			hostPort = net.JoinHostPort(hostPort, "80")
		} else {
			hostPort = net.JoinHostPort(hostPort, "443")
		}
	}

	outbound, err := dial(hostPort)
	if err != nil {
		error502Handle(w, req, err)
		return
	}

	defer outbound.Close()

	if req.ProtoMajor == 1 {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			error500Handle(w, req, errors.New("hijacker is not supported"))
			return
		}

		clientConn, bufReader, err := hijacker.Hijack()
		if err != nil {
			error500Handle(w, req, errors.New("hijacker is not supported"))
			return
		}
		defer clientConn.Close()

		if bufReader != nil {
			// snippet borrowed from `proxy` plugin
			if n := bufReader.Reader.Buffered(); n > 0 {
				rbuf, err := bufReader.Reader.Peek(n)
				if err != nil {
					error500Handle(w, req, errors.New("buf read error"))
					return
				}
				outbound.Write(rbuf)
			}
		}

		res := &http.Response{StatusCode: http.StatusOK,
			Proto:      "HTTP/1.1",
			ProtoMajor: 1,
			ProtoMinor: 1,
			Header:     make(http.Header),
		}
		res.Header.Set("Server", fakeServer)
		err = res.Write(clientConn)
		if err != nil {
			error500Handle(w, req, errors.New("write to client error"))
			return
		}
		dualStream(outbound, clientConn, clientConn)
	} else if req.ProtoMajor == 2 {
		defer req.Body.Close()
		wFlusher, ok := w.(http.Flusher)
		if !ok {
			error500Handle(w, req, errors.New("ResponseWriter doesn't implement Flusher"))
			return
		}
		w.WriteHeader(http.StatusOK)
		wFlusher.Flush()
		dualStream(outbound, req.Body, w)
		return
	} else {
		error500Handle(w, req, errors.New("HTTP version not supported"))
		return
	}
}
