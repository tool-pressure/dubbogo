package transport

import (
	"net"
	"time"
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

type Socket interface {
	Recv(*Package) error
	Send(*Package) error
	Reset(c net.Conn, release func())
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type Client interface {
	Recv(*Package) error
	Send(*Package) error
	Close() error
}

type Listener interface {
	Addr() string
	Close() error
	Accept(func(Socket)) error
}

// Transport is an interface which is used for communication between
// services. It uses socket send/recv semantics.
type Transport interface {
	Options() *Options
	Dial(addr string, opts ...DialOption) (Client, error)
	Listen(addr string, opts ...ListenOption) (Listener, error)
	String() string
}

type (
	Option func(*Options)

	DialOption func(*DialOptions)

	ListenOption func(*ListenOptions)

	NewTransport func(...Option) Transport
)

var (
	DefaultDialTimeout = time.Second * 5
)

// just leave here to compatible with v0.1
func NewHTTPTransport(opts ...Option) Transport {
	return newHTTPTransport(opts...)
}

func NewTCPTransport(opts ...Option) Transport {
	return newTCPTransport(opts...)
}
