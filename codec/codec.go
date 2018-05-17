package codec

import (
	"io"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	Error     MessageType = 0x01
	Request               = 0x02
	Response              = 0x04
	Heartbeat             = 0x08
)

var (
	ErrHeaderNotEnough = jerrors.Errorf("header buffer too short")
	ErrBodyNotEnough   = jerrors.Errorf("body buffer too short")
	ErrIllegalPackage  = jerrors.Errorf("illegal pacakge!")
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
	ID          int64
	Version     string
	Type        MessageType
	ServicePath string // service path
	Target      string // Service
	Method      string
	Timeout     time.Duration // request timeout
	Error       string
	Header      map[string]string
	BodyLen     int
}
