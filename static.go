package main

import (
	"net/http"
	"net/http/httputil"
)

var (
	rpHandler   *httputil.ReverseProxy
	rpTransport *http.Transport
)

// Connect Successful
const (
	StatusProxySuccess = 1000
)

// StatusText return HTTP STATUS code including 1000
func StatusText(code int) string {
	if code == StatusProxySuccess {
		return "Connect OK"
	}
	return http.StatusText(code)
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

// Error Handlers
func errorCoreHandle(w http.ResponseWriter, req *http.Request, code int, err error) {

	if err != nil {
		log.Debugf("[server] error(%v): %v\n", code, err)
	}

	w.Header().Add("Content-Type", "text/plain; charset=utf-8")
	w.Header().Add("X-Content-Type-Options", "nosniff")
	rawText := StatusText(code)

	// for 200, 407 or 1000
	if code == http.StatusOK || code == http.StatusProxyAuthRequired || code == StatusProxySuccess {
		responsePadding(w)
		w.Header().Add("server", nameStr)
	}
	if code == http.StatusOK {
		rawText = ""
	}

	if code == http.StatusProxyAuthRequired {
		w.Header().Add("Proxy-Authenticate", "Basic realm=\"Wickproxy Secure Proxy\"")
	}
	if code == StatusProxySuccess {
		rawText = rawText + ". Authenticate Successfully! Feel free to browse!"
		code = http.StatusOK
	}

	w.WriteHeader(code)
	w.Write([]byte(rawText))
}

func error403Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusForbidden, err)
}

func error404Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusNotFound, err)
}

func error500Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusInternalServerError, err)
}

func error502Handle(w http.ResponseWriter, req *http.Request, err error) {
	errorCoreHandle(w, req, http.StatusBadGateway, err)
}

func proxy407Handle(w http.ResponseWriter, req *http.Request) {
	errorCoreHandle(w, req, http.StatusProxyAuthRequired, nil)
}

func proxyPassHandle(w http.ResponseWriter, req *http.Request) {
	errorCoreHandle(w, req, StatusProxySuccess, nil)
}

func proxy200Handle(w http.ResponseWriter, req *http.Request) {
	errorCoreHandle(w, req, http.StatusOK, nil)
}
