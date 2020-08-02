package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

const fakeServer = "nginx/1.16.0 (Ubuntu)"

const padding = "<!-- a padding to disable MSIE and Chrome friendly error page -->\n<!-- a padding to disable MSIE and Chrome friendly error page -->\n<!-- a padding to disable MSIE and Chrome friendly error page -->\n<!-- a padding to disable MSIE and Chrome friendly error page -->\n<!-- a padding to disable MSIE and Chrome friendly error page -->\n<!-- a padding to disable MSIE and Chrome friendly error page -->\n"

var fakeBody = "<html>\n<head><title>%v %v</title></head>\n<body bgcolor=\"white\">\n<center><h1>%v %v</h1></center>\n<hr><center>%v</center>\n</body>\n</html>\n" + padding

func errorCoreHandle(w http.ResponseWriter, req *http.Request, code int) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")

	statusText := http.StatusText(code)
	fb := fmt.Sprintf(fakeBody, code, statusText, code, statusText, fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}

func error403Handle(w http.ResponseWriter, req *http.Request, err error) {
	log.Errorln("[server] error", http.StatusForbidden, err)
	errorCoreHandle(w, req, http.StatusForbidden)
}

func error502Handle(w http.ResponseWriter, req *http.Request, err error) {
	log.Errorln("[server] error", http.StatusBadGateway, err)
	errorCoreHandle(w, req, http.StatusBadGateway)
}

func error500Handle(w http.ResponseWriter, req *http.Request, err error) {
	log.Errorln("[server] error", http.StatusInternalServerError, err)
	errorCoreHandle(w, req, http.StatusInternalServerError)
}

func errorHandle(w http.ResponseWriter, req *http.Request, code int, err error) {

	if code == 0 {
		code = http.StatusNotFound
	}

	log.Errorln("[server] error:", code, err)
	if GlobalConfig.FallbackURL != "" {
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
	fb := fmt.Sprintf(fakeBody, code, "Authenticate Successful", code, "Authenticate Successful!", fakeServer)

	w.WriteHeader(code)
	w.Write([]byte(fb))
}

func reverseProxyHandler(w http.ResponseWriter, req *http.Request) {

	// get where to connect
	var err error
	var target string
	var targetURL *url.URL

	if !strings.HasPrefix(GlobalConfig.FallbackURL, "http") {
		targetURL, err = url.Parse("http://" + GlobalConfig.FallbackURL)
	} else {
		targetURL, err = url.Parse(GlobalConfig.FallbackURL)
	}
	if err != nil {
		log.Errorln("[fallback] parse fallback url error:", err)
		errorCoreHandle(w, req, http.StatusNotFound)
		return
	}

	if targetURL.Port() == "" {
		if targetURL.Scheme == "https" {
			target = net.JoinHostPort(targetURL.Host, "443")
		} else {
			target = net.JoinHostPort(targetURL.Host, "80")
		}
	} else {
		target = targetURL.Host
	}

	log.Debugln("[fallback] proxy to", targetURL.Scheme, target)

	// connect to the next hop
	var outbound net.Conn
	if targetURL.Scheme == "https" {
		cfg := tls.Config{}
		outbound, err = tls.Dial("tcp", target, &cfg)
	} else if GlobalConfig.Timeout > 0 {
		outbound, err = net.DialTimeout("tcp", target, GlobalConfig.Timeout*time.Second)
	} else {
		outbound, err = net.Dial("tcp", target)
	}

	// dump this request
	ProtoMajor := req.ProtoMajor
	req.Host = targetURL.Host
	req.ProtoMajor = 1
	req.ProtoMinor = 1
	req.Proto = "HTTP/1.1"
	if req.URL.Host == "" {
		req.URL.Host = req.Host
	}
	if req.URL.Scheme == "" {
		req.URL.Scheme = "http"
	}
	dumpReq, err := httputil.DumpRequest(req, true)
	if err != nil {
		log.Errorln("[fallback] dump request failed:", err)
		errorCoreHandle(w, req, http.StatusNotFound)
		return
	}
	log.Debugln("[fallback] ", string(dumpReq))

	// rewrite requests to next hop
	_, err = outbound.Write(dumpReq)
	if err != nil {
		log.Errorln("[fallback] rewrite request:", err)
		errorCoreHandle(w, req, http.StatusNotFound)
	}

	if ProtoMajor == 2 {
		log.Errorln("[fallback] fallback is not support HTTP2 now. Set http2 to false")
		errorCoreHandle(w, req, http.StatusNotFound)
		return
	}

	// hijacker
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		log.Errorln("[fallback] server do not support hijacker")
		errorCoreHandle(w, req, http.StatusNotFound)
		return
	}

	clientConn, bufReader, err := hijacker.Hijack()
	if err != nil {
		log.Errorln("[fallback] server do not support hijacker:", err)
		errorCoreHandle(w, req, http.StatusNotFound)
		return
	}
	defer clientConn.Close()

	if bufReader != nil {
		// snippet borrowed from `proxy` plugin
		if n := bufReader.Reader.Buffered(); n > 0 {
			rbuf, err := bufReader.Reader.Peek(n)
			if err != nil {
				log.Errorln("[fallback] bufReader error:", err)
				errorCoreHandle(w, req, http.StatusNotFound)
				return
			}
			outbound.Write(rbuf)
		}
	}
	dualStream(outbound, clientConn, clientConn)
}

// transport := http.DefaultTransport
// outReq := new(http.Request)
// outReq = req.Clone(req.Context())
// outReq.URL = target
// outReq.Host = target.Host
// outReq.URL.User = req.URL.User
// removeHopByHop(req.Header)

// log.Debugln("[reverse] reverse proxy to:", outReq.URL.Scheme, outReq.URL.Host)
// if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
// 	if prior, ok := outReq.Header["X-Forwarded-For"]; ok {
// 		clientIP = strings.Join(prior, ", ") + ", " + clientIP
// 	}
// 	outReq.Header.Set("X-Forwarded-For", clientIP)
// }

// res, err := transport.RoundTrip(outReq)
// if err != nil {
// 	error502Handle(w, req, err)
// 	return
// }

// removeHopByHop(res.Header)
// for key, value := range res.Header {
// 	for _, v := range value {
// 		w.Header().Add(key, v)
// 	}
// }

// w.WriteHeader(res.StatusCode)
// io.Copy(w, res.Body)
// res.Body.Close()
//}
