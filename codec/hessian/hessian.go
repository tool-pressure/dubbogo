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
	"io"
)

import (
	"github.com/AlexStocks/goext/log"
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

func (h *hessianCodec) Close() error {
	h.buf.Reset()
	return h.rwc.Close()
}

func (h *hessianCodec) String() string {
	return "hessian codec"
}

func (h *hessianCodec) Write(m *codec.Message, b interface{}) error {
	gxlog.CInfo("@m:%+v, b:%+v", m, b)
	switch m.Type {
	case codec.Request:
		h.buf.Reset()
		return jerrors.Trace(packRequest(m, b, h.rwc))
	case codec.Response:
		// return h.s.Write(m, b)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", m.Type)
	}

	return nil
}

func (h *hessianCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	h.buf.Reset()
	h.mt = mt

	gxlog.CError("get response header")

	switch mt {
	case codec.Request:
		// return h.s.ReadHeader(m)
	case codec.Response:
		// return h.c.ReadHeader(m)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", mt)
	}
	return nil
}

func (h *hessianCodec) ReadBody(b interface{}) error {
	switch h.mt {
	case codec.Request:
		// return h.s.ReadBody(b)
	case codec.Response:
		// return h.c.ReadBody(b)
	default:
		return jerrors.Errorf("Unrecognised message type: %v", h.mt)
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
