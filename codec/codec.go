package codec

import (
	"io"
	"time"
)

const (
	Error MessageType = iota
	Request
	Response
	Heartbeat
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
