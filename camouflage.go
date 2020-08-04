package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const fakeServer = "nginx/1.16.0 (Ubuntu)"

const fakeBody = `<html>
<head><title>%v</title></head>
<body bgcolor="white">
<center><h1>%v</h1></center>
<hr><center>%v</center>
</body>
</html>
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->`

const proxyBody = `<html>
<head><title>Wickproxy Proxy Server</title></head>
<body bgcolor="white">
<center><h1>Wickproxy Proxy Server</h1></center>
<\br>
<center><p>%v</p></center>
<hr><center>%v</center>
</body
</html>
`

var rpHandler *httputil.ReverseProxy

// Error Handlers
func errorCoreHandle(w http.ResponseWriter, req *http.Request, code int, err error) {
	w.Header().Add("server", fakeServer)
	w.Header().Add("content-type", "text/html")

	log.Errorf("[server] error(%v): %v\n", code, err)
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

	if GlobalConfig.FallbackURL != "" {
		log.Errorf("[server] error(%v): %v\n", code, err)
		reverseProxyHandler2(w, req)
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

// func reverseProxyHandler(w http.ResponseWriter, req *http.Request) {
// 	// get where to connect
// 	var err error
// 	var target string
// 	var targetURL *url.URL

// 	if !strings.HasPrefix(GlobalConfig.FallbackURL, "http") {
// 		targetURL, err = url.Parse("http://" + GlobalConfig.FallbackURL)
// 	} else {
// 		targetURL, err = url.Parse(GlobalConfig.FallbackURL)
// 	}
// 	if err != nil {
// 		error404Handle(w, req, errors.New("[fallback] parse fallback url error:"+err.Error()))
// 		return
// 	}

// 	if targetURL.Port() == "" {
// 		if targetURL.Scheme == "https" {
// 			target = net.JoinHostPort(targetURL.Host, "443")
// 		} else {
// 			target = net.JoinHostPort(targetURL.Host, "80")
// 		}
// 	} else {
// 		target = targetURL.Host
// 	}

// 	log.Debugln("[fallback] proxy to", targetURL.Scheme, target)

// 	// connect to the next hop
// 	var outbound net.Conn
// 	if targetURL.Scheme == "https" {
// 		cfg := tls.Config{}
// 		outbound, err = tls.Dial("tcp", target, &cfg)
// 	} else if GlobalConfig.Timeout > 0 {
// 		outbound, err = net.DialTimeout("tcp", target, GlobalConfig.Timeout*time.Second)
// 	} else {
// 		outbound, err = net.Dial("tcp", target)
// 	}

// 	// dump this request
// 	ProtoMajor := req.ProtoMajor
// 	req.Host = targetURL.Host
// 	req.ProtoMajor = 1
// 	req.ProtoMinor = 1
// 	req.Proto = "HTTP/1.1"
// 	if req.URL.Host == "" {
// 		req.URL.Host = req.Host
// 	}
// 	if req.URL.Scheme == "" {
// 		req.URL.Scheme = "http"
// 	}
// 	dumpReq, err := httputil.DumpRequest(req, true)
// 	if err != nil {
// 		error404Handle(w, req, errors.New("[fallback] dump request failed: "+err.Error()))
// 		return
// 	}
// 	log.Debugln("[fallback] ", string(dumpReq))

// 	// rewrite requests to next hop
// 	_, err = outbound.Write(dumpReq)
// 	if err != nil {
// 		error404Handle(w, req, errors.New("[fallback] rewrite request: "+err.Error()))
// 		return
// 	}

// 	if ProtoMajor == 2 {
// 		error404Handle(w, req, errors.New("[fallback] fallback is not support HTTP2 now. Set http2 to false"))
// 		return
// 	}

// 	// hijacker
// 	hijacker, ok := w.(http.Hijacker)
// 	if !ok {
// 		error404Handle(w, req, errors.New("[fallback] server do not support hijacker"))
// 		return
// 	}

// 	clientConn, bufReader, err := hijacker.Hijack()
// 	if err != nil {
// 		error404Handle(w, req, errors.New("[fallback] server do not support hijacker: "+err.Error()))
// 		return
// 	}
// 	defer clientConn.Close()

// 	if bufReader != nil {
// 		// snippet borrowed from `proxy` plugin
// 		if n := bufReader.Reader.Buffered(); n > 0 {
// 			rbuf, err := bufReader.Reader.Peek(n)
// 			if err != nil {
// 				error404Handle(w, req, errors.New("[fallback] bufReader error: "+err.Error()))
// 				return
// 			}
// 			outbound.Write(rbuf)
// 		}
// 	}
// 	dualStream(outbound, clientConn, clientConn)
// }

func reverseProxyHandler2Init() {
	rpURL, err := url.Parse(GlobalConfig.FallbackURL)
	if err != nil {
		log.Fatalln("[fallback] init reverse proxy server error:", err)
	}
	rpHandler = httputil.NewSingleHostReverseProxy(rpURL)
	rpHandler.ErrorLog = loggerAdapter
	rpHandler.Director = func(req *http.Request) {
		req.URL.Scheme = rpURL.Scheme
		req.URL.Host = rpURL.Host
		req.Host = rpURL.Host
		removeHopByHop(req.Header)
	}
	rpHandler.ModifyResponse = func(w *http.Response) error {
		removeHopByHop(w.Header)
		return nil
	}

	rpHandler.ErrorHandler = error404Handle
}

func reverseProxyHandler2(w http.ResponseWriter, req *http.Request) {
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
