package main

import (
	"encoding/base64"
	"net/http"
	"strings"
)

func proxyAuth(r *http.Request) (username, password string, ok bool) {
	auth := r.Header.Get("Proxy-Authorization")
	if auth == "" {
		if r.URL.User.Username() != "" {
			user := r.URL.User.Username()
			pass, isPass := r.URL.User.Password()
			if isPass {
				return user, pass, true
			}
			return user, "", true
		}
		return
	}
	return parseBasicAuth(auth)
}

func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}

var hopByHopHeaders = []string{
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Upgrade",
	"Connection",
	"Proxy-Connection",
	"Te",
	"Trailer",
	"Transfer-Encoding",
}

func removeHopByHop(header http.Header) {
	connectionHeaders := header.Get("Connection")
	for _, h := range strings.Split(connectionHeaders, ",") {
		header.Del(strings.TrimSpace(h))
	}
	for _, h := range hopByHopHeaders {
		header.Del(h)
	}
}

func checkWhiteList(host string) (ret bool) {
	if GlobalConfig.SecureURL != "" && GlobalConfig.SecureURL == host {
		return true
	}

	wurls := strings.Split(GlobalConfig.WhiteListURL, " ")
	for _, wurl := range wurls {
		if host == wurl {
			return true
		}
	}
	return false
}