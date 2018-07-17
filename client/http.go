package client

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

import (
	"github.com/AlexStocks/goext/log"
	jerrors "github.com/juju/errors"
	"github.com/valyala/fasthttp"
)

const (
	PathPrefix = byte('/')
)

type Package struct {
	Header map[string]string
	Body   []byte
}

//////////////////////////////////////////////
// http transport client
//////////////////////////////////////////////

func SetNetConnTimeout(conn net.Conn, timeout time.Duration) {
	t := time.Time{}
	if timeout > time.Duration(0) {
		t = time.Now().Add(timeout)
	}

	conn.SetReadDeadline(t)
}

type bufer struct {
	io.ReadWriter
}

func (b *bufer) Close() error {
	return nil
}

type httpClient struct {
	conn    net.Conn
	addr    string
	path    string
	timeout time.Duration
	req     *http.Request
}

func initHTTPClient(addr, path string, timeout time.Duration) (*httpClient, error) {
	tcpConn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return &httpClient{
		conn:    tcpConn,
		addr:    addr,
		path:    path,
		timeout: timeout,
	}, nil
}

func (h *httpClient) Send(pkg *Package) error {
	header := make(http.Header)

	for k, v := range pkg.Header {
		header.Set(k, v)
	}

	reqB := bytes.NewBuffer(pkg.Body)
	defer reqB.Reset()
	buf := &bufer{
		reqB,
	}

	h.req = &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   h.addr,
			Path:   h.path,
		},
		Header:        header,
		Body:          buf,
		ContentLength: int64(reqB.Len()),
		Host:          h.addr,
	}

	if h.timeout > time.Duration(0) {
		SetNetConnTimeout(h.conn, h.timeout)
		defer SetNetConnTimeout(h.conn, 0)
	}

	reqBuf := bytes.NewBuffer(make([]byte, 0))
	err := h.req.Write(reqBuf)
	if err == nil {
		_, err = reqBuf.WriteTo(h.conn)
	}

	return jerrors.Trace(err)
}

func (h *httpClient) SendRecv(pkg *Package) ([]byte, error) {
	var (
		err      error
		rspBytes []byte
		req      *fasthttp.Request
		rsp      *fasthttp.Response
	)
	c := &fasthttp.HostClient{
		//Addr: h.addr,
		//Dial: func(addr string) (net.Conn, error) {
		//	return net.DialTimeout("tcp", addr, h.timeout)
		//},
		//ReadTimeout:  h.timeout,
		//WriteTimeout: h.timeout,
	}
	req = fasthttp.AcquireRequest()
	req.Header.SetMethod("POST")
	for k, v := range pkg.Header {
		req.Header.Set(k, v)
	}
	req.SetHost(h.addr)
	req.SetRequestURI(h.path)
	req.SetBody(pkg.Body)
	gxlog.CError("request header:%s", req.Header.String())
	rsp = fasthttp.AcquireResponse()
	defer func() {
		fasthttp.ReleaseResponse(rsp)
		fasthttp.ReleaseRequest(req)
	}()

	if err = c.Do(req, rsp); err != nil {
		return nil, jerrors.Trace(err)
	}
	if rsp.StatusCode() != 200 {
		return nil, jerrors.New(http.StatusText(rsp.StatusCode()) + ": " + string(rsp.Body()))
	}

	return append(rspBytes, rsp.Body()...), nil
}

func (h *httpClient) Recv(p *Package) error {
	// set timeout if its greater than 0
	if h.timeout > time.Duration(0) {
		SetNetConnTimeout(h.conn, h.timeout)
		defer SetNetConnTimeout(h.conn, 0)
	}

	rsp, err := http.ReadResponse(bufio.NewReader(h.conn), h.req)
	if err != nil {
		return jerrors.Trace(err)
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return jerrors.Trace(err)
	}

	if rsp.StatusCode != 200 {
		return jerrors.New(rsp.Status + ": " + string(b))
	}

	mr := &Package{
		Header: make(map[string]string),
		Body:   b,
	}

	for k, v := range rsp.Header {
		if len(v) > 0 {
			mr.Header[k] = v[0]
		} else {
			mr.Header[k] = ""
		}
	}

	*p = *mr
	return nil
}

func (h *httpClient) Close() error {
	return jerrors.Trace(h.conn.Close())
}
