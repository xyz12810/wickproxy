package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

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
