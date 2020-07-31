package main

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const fakeServer = "nginx/1.14.0 (Ubuntu)"
const fakeBody = "<html>\n<head><title>%v %v</title></head>\n<body bgcolor=\"white\">\n<center><h1>%v %v</h1></center>\n<hr><center>%v</center>\n</body>\n</html>\n"

func errorCoreHandle(w http.ResponseWriter, req *http.Request, code int) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")

	statusText := http.StatusText(code)
	fb := fmt.Sprintf(fakeBody, code, statusText, code, statusText, fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}

func error502Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusBadGateway)
}

func error500Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusInternalServerError)
}

func reverseProxyHandler(w http.ResponseWriter, req *http.Request) {

	log.Infoln("[reverse] proxy to", GlobalConfig.FailbackURL)
	var err error
	target, err := url.Parse(GlobalConfig.FailbackURL)
	if err != nil {
		log.Errorln("[reverse] url parse error:", err)
		return
	}

	transport := http.DefaultTransport
	outReq := new(http.Request)
	outReq = req.Clone(req.Context())
	*outReq.URL = *target
	outReq.Host = target.Host

	log.Debugln("[reverse] reverse proxy to:", outReq.URL.Scheme, outReq.URL.Host)
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if prior, ok := outReq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		outReq.Header.Set("X-Forwarded-For", clientIP)
	}

	res, err := transport.RoundTrip(outReq)
	if err != nil {
		error502Handle(w, req, err)
		return
	}

	for key, value := range res.Header {
		for _, v := range value {
			w.Header().Add(key, v)
		}
	}

	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
	res.Body.Close()
}

func errorHandle(w http.ResponseWriter, req *http.Request, code int, err error) {

	if code == 0 {
		code = http.StatusNotFound
	}

	log.Errorln("[server] error:", code, err)
	if GlobalConfig.FailbackURL != "" {
		reverseProxyHandler(w, req)
		return
	}
	errorCoreHandle(w, req, code)
}

func error407Handle(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")
	w.Header().Add("Proxy-Authenticate", "Basic realm=\"Wickproxy Secure Proxy\"")

	code := http.StatusProxyAuthRequired
	statusText := http.StatusText(code)
	fb := fmt.Sprintf(fakeBody, code, statusText, code, statusText, fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}

func errorPassHandle(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")

	code := http.StatusOK
	fb := fmt.Sprintf(fakeBody, code, "Authenticate Successful", code, "Congratulations, you are successfully authenticated to the proxy! Go browse all the things!", fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}
