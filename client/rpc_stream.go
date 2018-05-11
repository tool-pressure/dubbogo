/******************************************************
# DESC    : invoke dubbogo.codec & dubbogo.transport to send app req & recv provider rsp
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-30 10:46
# FILE    : rpc_stream.go
******************************************************/

package client

import (
	"context"
	"io"
	"strings"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/registry"
)

// Implements the streamer interface
type rpcStream struct {
	sync.RWMutex
	seq        int64
	closed     chan struct{}
	err        error
	serviceURL registry.ServiceURL
	request    Request
	codec      clientCodec
	context    context.Context
}

func (r *rpcStream) isClosed() bool {
	select {
	case <-r.closed:
		return true
	default:
		return false
	}
}

func (r *rpcStream) Context() context.Context {
	return r.context
}

func (r *rpcStream) Request() Request {
	return r.request
}

// 调用rpcStream.clientCodec.WriteRequest函数
func (r *rpcStream) Send(args interface{}, timeout time.Duration) error {
	r.Lock()

	if r.isClosed() {
		r.err = errShutdown
		r.Unlock()
		return errShutdown
	}

	seq := r.seq
	r.seq++
	r.Unlock()

	req := request{
		Version:       r.request.Version(),
		ServicePath:   strings.TrimPrefix(r.serviceURL.Path, "/"),
		Service:       r.request.ServiceConfig().(*registry.ServiceConfig).Service,
		Seq:           seq,
		ServiceMethod: r.request.Method(),
		Timeout:       timeout,
	}

	if err := r.codec.WriteRequest(&req, args); err != nil {
		r.err = err
		return jerrors.Trace(err)
	}

	return nil
}

func (r *rpcStream) Recv(msg interface{}) error {
	r.Lock()
	defer r.Unlock()

	if r.isClosed() {
		r.err = errShutdown
		return errShutdown
	}

	var resp response
	if err := r.codec.ReadResponseHeader(&resp); err != nil {
		if err == io.EOF && !r.isClosed() {
			r.err = io.ErrUnexpectedEOF
			return io.ErrUnexpectedEOF
		}
		log.Warn("msg{%v}, err{%#v}", msg, err)
		r.err = err
		return jerrors.Trace(err)
	}

	switch {
	case len(resp.Error) > 0:
		// We've got an error response. Give this to the request;
		// any subsequent requests will get the ReadResponseBody
		// error if there is one.
		if resp.Error != lastStreamResponseError {
			r.err = serverError(resp.Error)
		} else {
			r.err = io.EOF
		}
		if err := r.codec.ReadResponseBody(nil); err != nil {
			r.err = jerrors.Annotate(err, "reading error payload")
		}
	default:
		if err := r.codec.ReadResponseBody(msg); err != nil {
			r.err = jerrors.Annotate(err, "reading body")
		}
	}

	return r.err
}

func (r *rpcStream) Error() error {
	r.RLock()
	defer r.RUnlock()
	return r.err
}

func (r *rpcStream) Close() error {
	select {
	case <-r.closed:
		return nil
	default:
		close(r.closed)
		return r.codec.Close()
	}
}
