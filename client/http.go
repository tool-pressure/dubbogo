package client

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	PathPrefix = byte('/')
)

type Package struct {
	Header map[string]string
	Body   []byte
}

func (m *Package) Reset() {
	m.Body = m.Body[:0]
	for key := range m.Header {
		delete(m.Header, key)
	}
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

type buffer struct {
	io.ReadWriter
}

func (b *buffer) Close() error {
	return nil
}

type httpClient struct {
	conn    net.Conn
	addr    string
	path    string
	timeout time.Duration
	once    sync.Once

	sync.Mutex
	r    chan *http.Request
	bl   []*http.Request
	buff *bufio.Reader
}

func initHTTPClient(
	addr string,
	path string,
	timeout time.Duration,
) (*httpClient, error) {

	tcpConn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return &httpClient{
		conn:    tcpConn,
		addr:    addr,
		path:    path,
		timeout: timeout,
		buff:    bufio.NewReader(tcpConn),
		r:       make(chan *http.Request, 1),
	}, nil
}

func (h *httpClient) Send(p *Package) error {
	header := make(http.Header)

	for k, v := range p.Header {
		header.Set(k, v)
	}

	reqB := bytes.NewBuffer(p.Body)
	defer reqB.Reset()
	buf := &buffer{
		reqB,
	}

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Host:   h.addr,
			Path:   h.path,
		},
		Header:        header, // p.header
		Body:          buf,    // p.body
		ContentLength: int64(reqB.Len()),
		Host:          h.addr,
	}

	h.Lock()
	h.bl = append(h.bl, req)
	select {
	case h.r <- h.bl[0]:
		h.bl = h.bl[1:]
	default:
	}
	h.Unlock()

	if h.timeout > time.Duration(0) {
		SetNetConnTimeout(h.conn, h.timeout)
		defer SetNetConnTimeout(h.conn, 0)
	}

	reqBuf := bytes.NewBuffer(make([]byte, 0))
	err := req.Write(reqBuf)
	if err == nil {
		_, err = reqBuf.WriteTo(h.conn)
	}

	return jerrors.Trace(err)
}

func (h *httpClient) Recv(p *Package) error {
	var r *http.Request
	rc, ok := <-h.r
	if !ok {
		return io.EOF
	}
	r = rc

	h.Lock()
	defer h.Unlock()
	if h.buff == nil {
		return io.EOF
	}

	// set timeout if its greater than 0
	if h.timeout > time.Duration(0) {
		SetNetConnTimeout(h.conn, h.timeout)
		defer SetNetConnTimeout(h.conn, 0)
	}

	rsp, err := http.ReadResponse(h.buff, r)
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
	var err error
	h.once.Do(func() {
		h.Lock()
		h.buff.Reset(nil)
		h.buff = nil
		h.Unlock()
		close(h.r)
		err = h.conn.Close()
	})
	return jerrors.Trace(err)
}
