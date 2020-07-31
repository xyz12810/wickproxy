package main

import (
	"errors"
	"net"
	"net/http"
	"strconv"
)

const defaultServer = "localhost:7890"

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
	var err error
	err = configReader(*config)
	if err != nil {
		log.Fatalln("[set] read config error:", err)
	}

	poolInit()

	server := getServer()
	log.Infoln("[server] listen at:", server)

	if GlobalConfig.TLS.Certificate != "" && GlobalConfig.TLS.CertificateKey != "" {
		err = http.ListenAndServeTLS(server, GlobalConfig.TLS.Certificate, GlobalConfig.TLS.CertificateKey, &proxyServer{})
	} else {
		err = http.ListenAndServe(server, &proxyServer{})
	}
	log.Fatalln("[server] server fatal:", err)
}

func defaultServerHandler(w http.ResponseWriter, req *http.Request) {

	// Check method == CONNECT
	if req.Method != http.MethodConnect {
		errorHandle(w, req, http.StatusNotFound, errors.New("Method not allowed"))
		return
	}

	// Authenticate
	ret, emptyAuth, username := authenticate(w, req)
	if emptyAuth {
		hostport := req.URL.Host
		host, _, err := net.SplitHostPort(hostport)
		if err != nil {
			host = hostport
		}

		if GlobalConfig.SecureURL == "" || (GlobalConfig.SecureURL == host) {
			error407Handle(w, req)
			return
		}
		errorHandle(w, req, http.StatusNotFound, errors.New("Empty Proxy-Authorization"))
		return
	}

	// Authenticate Failed
	if ret == false {
		errorHandle(w, req, http.StatusForbidden, errors.New("Authenticate Failed"))
		return
	}

	// visit secureURL
	if GlobalConfig.SecureURL != "" && GlobalConfig.SecureURL == req.URL.Host {
		errorPassHandle(w, req)
		return
	}

	// protocol version check
	if req.ProtoMajor != 1 && req.ProtoMajor != 2 {
		errorHandle(w, req, http.StatusHTTPVersionNotSupported, errors.New("Unsupported HTTP major version: "+strconv.Itoa(req.ProtoMajor)))
		return
	}

	// start to proxy
	log.Infoln("[transfer] user", username, ":", req.Method, req.URL)

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
		errorHandle(w, req, http.StatusBadGateway, err)
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

	return
}

func authenticate(w http.ResponseWriter, req *http.Request) (ret, emptyAuth bool, username string) {
	if len(GlobalConfig.Users) == 0 {
		return true, true, ""
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
