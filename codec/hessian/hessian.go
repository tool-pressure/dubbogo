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
	"io"
	"reflect"
)

import (
	"github.com/AlexStocks/goext/log"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/codec"
)

type hessianCodec struct {
	mt           codec.MessageType
	rwc          io.ReadWriteCloser
	rspHeaderLen int
}

func (h *hessianCodec) Close() error {
	return h.rwc.Close()
}

func (h *hessianCodec) String() string {
	return "hessian-codec"
}

func (h *hessianCodec) Write(m *codec.Message, a interface{}) error {
	gxlog.CInfo("@m:%+v, @a:%+v", m, a)
	switch m.Type {
	case codec.Request:
		return jerrors.Trace(packRequest(m, a, h.rwc))
	case codec.Response:
		return nil
	default:
		return jerrors.Errorf("Unrecognised message type: %v", m.Type)
	}

	return nil
}

func (h *hessianCodec) ReadHeader(m *codec.Message, mt codec.MessageType) error {
	h.mt = mt
	h.rspHeaderLen = 0

	switch mt {
	case codec.Request:
		return nil
	case codec.Response:
		var buf [HEADER_LENGTH]byte
		n, e := h.rwc.Read(buf[:])
		if e != nil {
			return jerrors.Trace(e)
		}
		if n < HEADER_LENGTH {
			return codec.ErrHeaderNotEnough
		}

		h.rspHeaderLen, e = UnpackResponseHeader(buf[:], m)
		return jerrors.Trace(e)

	default:
		return jerrors.Errorf("Unrecognised message type: %v", mt)
	}

	return nil
}

func (h *hessianCodec) ReadBody(b interface{}) error {
	switch h.mt {
	case codec.Request:
		return nil
	case codec.Response:
		if b == nil {
			return jerrors.Errorf("@b is nil")
		}

		buf := make([]byte, h.rspHeaderLen)
		n, e := h.rwc.Read(buf)
		if e != nil {
			return jerrors.Trace(e)
		}
		if n < h.rspHeaderLen {
			return codec.ErrBodyNotEnough
		}

		rsp, err := UnpackResponseBody(buf)
		if err != nil {
			return jerrors.Trace(err)
		}

		bv := reflect.ValueOf(b)
		switch bv.Kind() {
		case reflect.Ptr:
			rspValue := ReflectResponse(rsp, reflect.TypeOf(b).Elem())
			gxlog.CError("rspValue:%#v", rspValue)
			bv.Elem().Set(reflect.ValueOf(rspValue))

		default:
			err = jerrors.Errorf("@b:{%#v} reflection kind shoud be reflect.Ptr", b)
		}

		gxlog.CError("@b:%#v", b)

		return jerrors.Trace(err)

	default:
		return jerrors.Errorf("Unrecognised message type: %v", h.mt)
	}

	return nil
}

func NewCodec(rwc io.ReadWriteCloser) codec.Codec {
	return &hessianCodec{
		rwc: rwc,
	}
}
