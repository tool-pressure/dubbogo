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
	"bufio"
	"encoding/gob"
	"errors"
	"net"
	"runtime"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/AlexStocks/dubbogo/common"
)

const (
	DefaultTcpReadBufferSize  = 256 * 1024
	DefaultTcpWriteBufferSize = 128 * 1024
)

//////////////////////////////////////////////
// tcp transport client
//////////////////////////////////////////////

type tcpTransportClient struct {
	t        *tcpTransport
	dialOpts DialOptions
	conn     net.Conn
	enc      *gob.Encoder
	dec      *gob.Decoder
	encBuf   *bufio.Writer
}

func initTcpTransportClient(t *tcpTransport, conn net.Conn, opts DialOptions) *tcpTransportClient {
	encBuf := bufio.NewWriter(conn)
	return &tcpTransportClient{
		t:        t,
		dialOpts: opts,
		conn:     conn,
		encBuf:   encBuf,
		enc:      gob.NewEncoder(encBuf),
		dec:      gob.NewDecoder(conn),
	}
}

func (t *tcpTransportClient) Send(m *Message) error {
	if t.t.opts.Timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.t.opts.Timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}
	if err := t.enc.Encode(m); err != nil {
		return err
	}

	return t.encBuf.Flush()
}

func (t *tcpTransportClient) Recv(m *Message) error {
	if t.t.opts.Timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.t.opts.Timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	return t.dec.Decode(&m)
}

func (t *tcpTransportClient) Close() error {
	return t.conn.Close()
}

//////////////////////////////////////////////
// tcp transport socket
//////////////////////////////////////////////

type tcpTransportSocket struct {
	t       *tcpTransport
	conn    net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	encBuf  *bufio.Writer
	timeout time.Duration
	release func()
}

func initTcpTransportSocket(t *tcpTransport, c net.Conn, release func()) *tcpTransportSocket {
	encBuf := bufio.NewWriter(c)
	return &tcpTransportSocket{
		t:       t,
		conn:    c,
		encBuf:  encBuf,
		enc:     gob.NewEncoder(encBuf),
		dec:     gob.NewDecoder(c),
		release: release,
	}
}

func (t *tcpTransportSocket) Reset(c net.Conn, release func()) {
	t.Close()
	t.conn = c
	t.release = release
}

func (t *tcpTransportSocket) Recv(m *Message) error {
	if m == nil {
		return errors.New("message passed in is nil")
	}

	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}

	return t.dec.Decode(&m)
}

func (t *tcpTransportSocket) Send(m *Message) error {
	// set timeout if its greater than 0
	if t.timeout > time.Duration(0) {
		common.SetNetConnTimeout(t.conn, t.timeout)
		defer common.SetNetConnTimeout(t.conn, 0)
	}
	if err := t.enc.Encode(m); err != nil {
		return err
	}
	return t.encBuf.Flush()
}

func (t *tcpTransportSocket) Close() error {
	return t.conn.Close()
}

func (t *tcpTransportSocket) LocalAddr() net.Addr {
	return t.conn.LocalAddr()
}

func (t *tcpTransportSocket) RemoteAddr() net.Addr {
	return t.conn.RemoteAddr()
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
			return err
		}

		if tcpConn, ok := c.(*net.TCPConn); ok {
			tcpConn.SetReadBuffer(DefaultTcpReadBufferSize)
			tcpConn.SetWriteBuffer(DefaultTcpWriteBufferSize)
		}

		sock := initTcpTransportSocket(t.t, c, t.release)

		common.GoroutinePool.Go(func() {
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
		})
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

	conn, err := net.DialTimeout("tcp", addr, dopts.Timeout)
	if err != nil {
		return nil, err
	}

	return initTcpTransportClient(t, conn, dopts), nil
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
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return initTcpTransportListener(t, l), nil
}

func (t *tcpTransport) String() string {
	return "tcp"
}

func newTcpTransport(opts ...Option) Transport {
	var options Options
	for _, o := range opts {
		o(&options)
	}
	return &tcpTransport{opts: options}
}
