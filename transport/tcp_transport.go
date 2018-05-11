/******************************************************
# DESC    : tcp transport
# AUTHOR  :
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2017-10-19 13:17
# FILE    : tcp_transport.go
******************************************************/

package transport

import (
	"io"
	"net"
	"runtime"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/goext/log"
)

const (
	DefaultTcpReadBufferSize  = 256 * 1024
	DefaultTcpWriteBufferSize = 128 * 1024
)

//////////////////////////////////////////////
// tcp transport socket
//////////////////////////////////////////////

type tcpTransportSocket struct {
	t       *tcpTransport
	conn    net.Conn
	timeout time.Duration
	release func()
}

func initTcpTransportSocket(t *tcpTransport, c net.Conn, release func()) *tcpTransportSocket {
	return &tcpTransportSocket{
		t:       t,
		conn:    c,
		release: release,
	}
}

func (t *tcpTransportSocket) Reset(c net.Conn, release func()) {
	t.Close()
	t.conn = c
	t.release = release
}

func (t *tcpTransportSocket) Recv(p *Package) error {
	if p == nil {
		return jerrors.Errorf("message passed in is nil")
	}

	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	t.conn.Read(p.Body)

	return nil
}

func (t *tcpTransportSocket) Send(p *Package) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	n, err := t.conn.Write(p.Body)
	if err != nil {
		return jerrors.Trace(err)
	}
	if n < len(p.Body) {
		return jerrors.Annotatef(io.ErrShortWrite, "t.conn.Write(buf len:%d) = len:%d", len(p.Body), n)
	}

	return nil
}

func (t *tcpTransportSocket) Close() error {
	return jerrors.Trace(t.conn.Close())
}

func (t *tcpTransportSocket) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *tcpTransportSocket) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
}

//////////////////////////////////////////////
// tcp transport client
//////////////////////////////////////////////

type tcpTransportClient struct {
	t    *tcpTransport
	conn net.Conn
}

func initTcpTransportClient(t *tcpTransport, conn net.Conn) *tcpTransportClient {
	return &tcpTransportClient{
		t:    t,
		conn: conn,
	}
}

func (t *tcpTransportClient) Send(p *Package) error {
	if t.t.opts.Timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.t.opts.Timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	n, err := t.conn.Write(p.Body)
	if err != nil {
		return jerrors.Trace(err)
	}
	if n < len(p.Body) {
		return jerrors.Annotatef(io.ErrShortWrite, "t.conn.Write(buf len:%d) = len:%d", len(p.Body), n)
	}

	return nil
}

func (t *tcpTransportClient) Recv(p *Package) error {
	if t.t.opts.Timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.t.opts.Timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	if p.Body == nil {
		p.Body = make([]byte, 0, 128)
	}
	buf := make([]byte, 0, 1024)
	// l, err := t.conn.Read(p.Body)
	l, err := t.conn.Read(buf)
	gxlog.CWarn("rsp return len %d, err:%v", l, err)

	return jerrors.Trace(err)
}

func (t *tcpTransportClient) Close() error {
	return jerrors.Trace(t.conn.Close())
}

//////////////////////////////////////////////
// tcp transport listener
//////////////////////////////////////////////

type tcpTransportListener struct {
	t        *tcpTransport
	listener net.Listener
	sem      chan struct{}
}

func initTcpTransportListener(t *tcpTransport, listener net.Listener) *tcpTransportListener {
	return &tcpTransportListener{
		t:        t,
		listener: listener,
		sem:      make(chan struct{}, DefaultMAXConnNum),
	}
}

func (t *tcpTransportListener) acquire() { t.sem <- struct{}{} }
func (t *tcpTransportListener) release() { <-t.sem }

func (t *tcpTransportListener) Addr() string {
	return t.listener.Addr().String()
}

func (t *tcpTransportListener) Close() error {
	return t.listener.Close()
}

func (t *tcpTransportListener) Accept(fn func(Socket)) error {
	var (
		err       error
		c         net.Conn
		ok        bool
		ne        net.Error
		tempDelay time.Duration
	)

	for {
		t.acquire() // 若connect chan已满,则会阻塞在此处
		c, err = t.listener.Accept()
		if err != nil {
			t.release()
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				if tempDelay != 0 {
					tempDelay <<= 1
				} else {
					tempDelay = 5 * time.Millisecond
				}
				if tempDelay > DefaultMaxSleepTime {
					tempDelay = DefaultMaxSleepTime
				}
				log.Info("htp: Accept error: %v; retrying in %v\n", err, tempDelay)
				time.Sleep(tempDelay)
				continue
			}
			return jerrors.Trace(err)
		}

		if tcpConn, ok := c.(*net.TCPConn); ok {
			tcpConn.SetReadBuffer(DefaultTcpReadBufferSize)
			tcpConn.SetWriteBuffer(DefaultTcpWriteBufferSize)
		}

		sock := initTcpTransportSocket(t.t, c, t.release)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					log.Error("tcp: panic serving %v: %v\n%s", c.RemoteAddr(), r, buf)
					sock.Close() // 遇到错误退出的时候保证socket fd的回收
				}
			}()

			fn(sock) // rpcServer:handlePkg 函数里面有一个defer语句段，保证了正常退出的情况下sock.Close()
		}()
	}
}

//////////////////////////////////////////////
// tcp transport
//////////////////////////////////////////////

type tcpTransport struct {
	opts Options
}

func (t *tcpTransport) Dial(addr string, opts ...DialOption) (Client, error) {
	dopts := DialOptions{
		Timeout: DefaultDialTimeout,
	}
	for _, opt := range opts {
		opt(&dopts)
	}

	gxlog.CError("connect address:%s", addr)

	conn, err := net.DialTimeout("tcp", addr, dopts.Timeout)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return initTcpTransportClient(t, conn), nil
}

func (t *tcpTransport) Listen(addr string, opts ...ListenOption) (Listener, error) {
	var options ListenOptions
	for _, o := range opts {
		o(&options)
	}

	fn := func(addr string) (net.Listener, error) {
		return net.Listen("tcp", addr)
	}
	l, err := listen(addr, fn)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	return initTcpTransportListener(t, l), nil
}

func (t *tcpTransport) String() string {
	return "tcp-transport"
}

func newTcpTransport(opts ...Option) Transport {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return &tcpTransport{opts: options}
}
