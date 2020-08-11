package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const fakeServer = "nginx/1.14.2 (Ubuntu)"

const fakeBody = `<html>
<head><title>%v</title></head>
<body bgcolor="white">
<center><h1>%v</h1></center>
<hr><center>%v</center>
</body>
</html>`

const proxyBody = `<html>
<head><title>Wickproxy Proxy Server</title></head>
<body bgcolor="white">
<center><h1>Wickproxy Proxy Server</h1></center>
<center><p>%v</p></center>
<hr><center>%v</center>
</body>
</html>
`

var rpHandler *httputil.ReverseProxy

// Error Handlers
func errorCoreHandle(w http.ResponseWriter, req *http.Request, code int, err error) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")

	log.Debugf("[server] error(%v): %v\n", code, err)
	statusText := fmt.Sprintf("%v %v", code, http.StatusText(code))
	fb := fmt.Sprintf(fakeBody, statusText, statusText, fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}

func error403Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusForbidden, err)
}

func error404Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusNotFound, err)
}

func error502Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusBadGateway, err)
}

func error500Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusInternalServerError, err)
}

func errorHandle(w http.ResponseWriter, req *http.Request, code int, err error) {
	if code == 0 {
		code = http.StatusNotFound
	}

	if GlobalConfig.Fallback != "" {
		log.Debugf("[server] error(%v): %v\n", code, err)
		reverseProxyHandler(w, req)
		return
	}
	errorCoreHandle(w, req, code, err)
}

// Authenticate Proxy Handlers
func proxy407Handle(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")
	w.Header().Add("Proxy-Authenticate", "Basic realm=\"Wickproxy Secure Proxy\"")
	responsePadding(w)
	code := http.StatusProxyAuthRequired
	fb := fmt.Sprintf(proxyBody, "Need to authenticate.", fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}

func proxyPassHandle(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")
	w.Header().Add("Proxy-Authenticate", "Basic realm=\"Wickproxy Secure Proxy\"")
	responsePadding(w)
	code := http.StatusOK
	fb := fmt.Sprintf(proxyBody, "Authenticate Successfully", fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}

func reverseProxyHandlerInit() {

	rpURL, err := url.Parse(GlobalConfig.Fallback)
	if err != nil {
		log.Fatalln("[fallback] init reverse proxy server error:", err)
	}
	rpHandler = httputil.NewSingleHostReverseProxy(rpURL)
	rpHandler.ErrorLog = loggerAdapter
	rpHandler.Director = func(req *http.Request) {
		req.URL.Scheme = rpURL.Scheme
		req.URL.Host = rpURL.Host
		// req.Host = rpURL.Host
		removeHopByHop(req.Header)
	}
	rpHandler.ModifyResponse = func(w *http.Response) error {
		removeHopByHop(w.Header)
		return nil
	}
	rpHandler.ErrorHandler = error404Handle
}

func reverseProxyHandler(w http.ResponseWriter, req *http.Request) {
	host := req.URL.Host
	if host == "" {
		host = req.Host
	}

	rpHandler.ServeHTTP(w, req)
}

// Fork from klzgrad/forwardprox
func responsePadding(w http.ResponseWriter) {
	paddingLen := rand.Intn(32) + 30
	padding := make([]byte, paddingLen)
	bits := rand.Uint64()
	for i := 0; i < 16; i++ {
		// Codes that won't be Huffman coded.
		padding[i] = "!#$()+<>?@[]^`{}"[bits&15]
		bits >>= 4
	}
	for i := 16; i < paddingLen; i++ {
		padding[i] = '~'
	}
	w.Header().Set("Padding", string(padding))
	w.Header().Add("server", fakeServer)
}
