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
	PathPrefix              = byte('/')
	lastStreamResponseError = "EOS"
)

// errShutdown holds the specific error for closing/closed connections
var (
	errShutdown = jerrors.New("connection is shut down")
)

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

type response struct {
	ServiceMethod string // echoes that of the Request
	Seq           int64  // echoes that of the request
	Error         string // error, if any.
}

func (rwc *readWriteCloser) Read(p []byte) (n int, err error) {
	return rwc.rbuf.Read(p)
}

func (rwc *readWriteCloser) Write(p []byte) (n int, err error) {
	return rwc.wbuf.Write(p)
}

func (rwc *readWriteCloser) Close() error {
	rwc.rbuf.Reset()
	rwc.wbuf.Reset()
	return nil
}

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

	codec  Codec
	pkg    *Package
	buf    *readWriteCloser
	closed chan empty
}

func initHTTPClient(
	addr string,
	path string,
	timeout time.Duration,
	pkg *Package,
	newCodec NewCodec,
) (*httpClient, error) {

	tcpConn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	rwc := &readWriteCloser{
		wbuf: bytes.NewBuffer(nil),
		rbuf: bytes.NewBuffer(nil),
	}

	return &httpClient{
		conn:    tcpConn,
		addr:    addr,
		path:    path,
		timeout: timeout,
		buff:    bufio.NewReader(tcpConn),
		r:       make(chan *http.Request, 1),

		buf:    rwc,
		codec:  newCodec(rwc),
		pkg:    pkg,
		closed: make(chan empty),
	}, nil
}

func (h *httpClient) isClosed() bool {
	select {
	case <-h.closed:
		return true
	default:
		return false
	}
}

func (h *httpClient) WriteRequest(msg Message, args interface{}) error {
	if h.isClosed() {
		return errShutdown
	}

	h.buf.wbuf.Reset()

	// Serialization
	if err := h.codec.Write(&msg, args); err != nil {
		return jerrors.Trace(err)
	}
	// get binary stream
	h.pkg.Body = h.buf.wbuf.Bytes()
	// tcp 层不使用 transport.Package.Header, codec.Write 调用之后其所有内容已经序列化进 transport.Package.Body
	if h.pkg.Header != nil {
		for k, v := range msg.Header {
			h.pkg.Header[k] = v
		}
	}

	return jerrors.Trace(h.Send())
}

func (h *httpClient) Send() error {
	header := make(http.Header)

	for k, v := range h.pkg.Header {
		header.Set(k, v)
	}

	reqB := bytes.NewBuffer(h.pkg.Body)
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

func (h *httpClient) ReadResponseHeader(r *response) error {
	var (
		err error
		p   Package
		cm  Message
	)

	if h.isClosed() {
		return errShutdown
	}

	h.buf.rbuf.Reset()
	err = h.Recv(&p)
	if err != nil {
		return jerrors.Trace(err)
	}
	h.buf.rbuf.Write(p.Body)
	err = h.codec.ReadHeader(&cm)

	r.ServiceMethod = cm.Method
	r.Seq = cm.ID
	r.Error = cm.Error

	return jerrors.Trace(err)
}

func (h *httpClient) ReadResponseBody(b interface{}) error {
	if h.isClosed() {
		return errShutdown
	}

	return jerrors.Trace(h.codec.ReadBody(b))
}

func (h *httpClient) Close() error {
	var err error

	select {
	case <-h.closed:
		return nil
	default:
		h.buf.Close()
		h.codec.Close()
		h.once.Do(func() {
			h.Lock()
			h.buff.Reset(nil)
			h.buff = nil
			h.Unlock()
			close(h.r)
			err = h.conn.Close()
		})
	}

	return jerrors.Trace(err)
}
