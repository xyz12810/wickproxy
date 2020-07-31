package main

import (
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
)

var bufferPool sync.Pool

func poolInit() {
	makeBuffer := func() interface{} { return make([]byte, 0, 32*1024) }
	bufferPool = sync.Pool{New: makeBuffer}
}

func flushingIoCopy(dst io.Writer, src io.Reader, buf []byte) (written int64, err error) {
	flusher, ok := dst.(http.Flusher)
	if !ok {
		return io.CopyBuffer(dst, src, buf)
	}
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			flusher.Flush()
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return
}

func dualStream(target net.Conn, clientReader io.ReadCloser, clientWriter io.Writer) error {
	stream := func(w io.Writer, r io.Reader) error {
		// copy bytes from r to w
		buf := bufferPool.Get().([]byte)
		buf = buf[0:cap(buf)]
		_, _err := flushingIoCopy(w, r, buf)
		if closeWriter, ok := w.(interface {
			CloseWrite() error
		}); ok {
			closeWriter.CloseWrite()
		}
		return _err
	}

	go stream(target, clientReader)
	return stream(clientWriter, target)
}

func serveHijack(w http.ResponseWriter, targetConn net.Conn) (int, error) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		return http.StatusInternalServerError, errors.New("ResponseWriter does not implement Hijacker")
	}
	clientConn, bufReader, err := hijacker.Hijack()
	if err != nil {
		return http.StatusInternalServerError, errors.New("failed to hijack: " + err.Error())
	}
	defer clientConn.Close()
	// bufReader may contain unprocessed buffered data from the client.
	if bufReader != nil {
		// snippet borrowed from `proxy` plugin
		if n := bufReader.Reader.Buffered(); n > 0 {
			rbuf, err := bufReader.Reader.Peek(n)
			if err != nil {
				return http.StatusBadGateway, err
			}
			targetConn.Write(rbuf)
		}
	}
	// Since we hijacked the connection, we lost the ability to write and flush headers via w.
	// Let's handcraft the response and send it manually.
	res := &http.Response{StatusCode: http.StatusOK,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
	}
	res.Header.Set("Server", fakeServer)

	err = res.Write(clientConn)
	if err != nil {
		return http.StatusInternalServerError, errors.New("failed to send response to client: " + err.Error())
	}

	return 0, dualStream(targetConn, clientConn, clientConn)
}

func dial(hostport string) (outbound net.Conn, err error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		return nil, err
	}

	if !aclPrivateCheck(host) {
		return nil, errors.New("ACL failed, private IP address: " + host)
	}

	if !aclCheck(host, port) {
		return nil, errors.New("ACL failed, host invalid: " + hostport)
	}
	

	if GlobalConfig.Timeout > 0 {
		outbound, err = net.DialTimeout("tcp", hostport, GlobalConfig.Timeout)
	} else {
		outbound, err = net.Dial("tcp", hostport)
	}
	return
}
