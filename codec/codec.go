package codec

import (
	"io"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	Error MessageType = iota
	Request
	Response
	Heartbeat
)

var (
	ErrHeaderNotEnough = jerrors.Errorf("header buffer too short")
	ErrBodyNotEnough   = jerrors.Errorf("body buffer too short")
)

type MessageType int

// Takes in a connection/buffer and returns a new Codec
type NewCodec func(io.ReadWriteCloser) Codec

type Codec interface {
	ReadHeader(*Message, MessageType) error
	ReadBody(interface{}) error
	Write(m *Message, args interface{}) error
	Close() error
	String() string
}

type Message struct {
	Id          int64
	Version     string
	Type        MessageType
	ServicePath string // service path
	Target      string // Service
	Method      string
	Timeout     time.Duration // request timeout
	Error       string
	Header      map[string]string
}
