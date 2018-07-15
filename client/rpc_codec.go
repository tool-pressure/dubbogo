package client

import (
	"bytes"
	"errors"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	lastStreamResponseError = "EOS"
)

// serverError represents an error that has been returned from
// the remote side of the RPC connection.
type serverError string

func (e serverError) Error() string {
	return string(e)
}

// errShutdown holds the specific error for closing/closed connections
var (
	errShutdown = errors.New("connection is shut down")
)

type rpcCodec struct {
	client *httpClient
	codec  Codec
	pkg    *Package
	buf    *readWriteCloser
}

type readWriteCloser struct {
	wbuf *bytes.Buffer
	rbuf *bytes.Buffer
}

type clientCodec interface {
	WriteRequest(req *request, args interface{}) error
	ReadResponseHeader(*response) error
	ReadResponseBody(interface{}) error

	Close() error
}

type request struct {
	Version       string
	ServicePath   string
	Service       string
	ServiceMethod string // format: "Service.Method"
	Seq           int64  // sequence number chosen by client
	Timeout       time.Duration
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

func newRPCCodec(req *Package, client *httpClient, newCodec NewCodec) *rpcCodec {
	rwc := &readWriteCloser{
		wbuf: bytes.NewBuffer(nil),
		rbuf: bytes.NewBuffer(nil),
	}

	return &rpcCodec{
		buf:    rwc,
		client: client,
		codec:  newCodec(rwc),
		pkg:    req,
	}
}

func (c *rpcCodec) WriteRequest(req *request, args interface{}) error {
	c.buf.wbuf.Reset()
	m := &Message{
		ID:          req.Seq,
		Version:     req.Version,
		ServicePath: req.ServicePath,
		Target:      req.Service,
		Method:      req.ServiceMethod,
		Timeout:     req.Timeout,
		Header:      map[string]string{},
	}
	// Serialization
	if err := c.codec.Write(m, args); err != nil {
		return jerrors.Trace(err)
	}
	// get binary stream
	c.pkg.Body = c.buf.wbuf.Bytes()
	// tcp 层不使用 transport.Package.Header, codec.Write 调用之后其所有内容已经序列化进 transport.Package.Body
	if c.pkg.Header != nil {
		for k, v := range m.Header {
			c.pkg.Header[k] = v
		}
	}
	return jerrors.Trace(c.client.Send(c.pkg))
}

func (c *rpcCodec) ReadResponseHeader(r *response) error {
	var (
		err error
		p   Package
		cm  Message
	)

	c.buf.rbuf.Reset()
	err = c.client.Recv(&p)
	if err != nil {
		return jerrors.Trace(err)
	}
	c.buf.rbuf.Write(p.Body)
	err = c.codec.ReadHeader(&cm)

	r.ServiceMethod = cm.Method
	r.Seq = cm.ID
	r.Error = cm.Error

	return jerrors.Trace(err)
}

func (c *rpcCodec) ReadResponseBody(b interface{}) error {
	return jerrors.Trace(c.codec.ReadBody(b))
}

func (c *rpcCodec) Close() error {
	c.buf.Close()
	c.codec.Close()
	return c.client.Close()
}
