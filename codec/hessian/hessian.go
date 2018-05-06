/******************************************************
# DESC    : hessian codec
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2017-10-23 13:05
# FILE    : hessian.go
******************************************************/

package hessian

import (
	"bytes"
	"fmt"
	"io"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
)

type hessianCodec struct {
	buf *bytes.Buffer
	mt  codec.MessageType
	rwc io.ReadWriteCloser
	// c   *clientCodec
	// s   *serverCodec
}

func (j *hessianCodec) Close() error {
	j.buf.Reset()
	return j.rwc.Close()
}

func (j *hessianCodec) String() string {
	return "hessian"
}

func (j *hessianCodec) Write(m *codec.Message, b interface{}) error {
	switch m.Type {
	case codec.Request:
		// return j.c.Write(m, b)
	case codec.Response:
		// return j.s.Write(m, b)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", m.Type)
	}

	return nil
}

func (j *hessianCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	j.buf.Reset()
	j.mt = mt

	switch mt {
	case codec.Request:
		// return j.s.ReadHeader(m)
	case codec.Response:
		// return j.c.ReadHeader(m)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", mt)
	}
	return nil
}

func (j *hessianCodec) ReadBody(b interface{}) error {
	switch j.mt {
	case codec.Request:
		// return j.s.ReadBody(b)
	case codec.Response:
		// return j.c.ReadBody(b)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", j.mt)
	}
	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &hessianCodec{
		buf: bytes.NewBuffer(nil),
		rwc: rwc,
		// c:   newClientCodec(rwc),
		// s:   newServerCodec(rwc, nil),
	}
}
