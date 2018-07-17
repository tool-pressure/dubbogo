// Copyright (c) 2015 Asim Aslam.
// Copyright (c) 2016 ~ 2018, Alex Stocks.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/AlexStocks/dubbogo/common"
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/selector"
)

const (
	CLEAN_CHANNEL_SIZE = 64
)

//////////////////////////////////////////////
// RPC Request
//////////////////////////////////////////////

type rpcRequest struct {
	group       string
	protocol    string
	version     string
	service     string
	method      string
	args        interface{}
	contentType string
	opts        RequestOptions
}

func (r *rpcRequest) ServiceConfig() registry.ServiceConfigIf {
	return &registry.ServiceConfig{
		Protocol: r.protocol,
		Service:  r.service,
		Group:    r.group,
		Version:  r.version,
	}
}

//////////////////////////////////////////////
// RPC Client
//////////////////////////////////////////////

type empty struct{}

// thread safe
type rpcClient struct {
	ID   int64
	once sync.Once
	opts Options
}

func newRPCClient(opt ...Option) Client {
	opts := newOptions(opt...)

	t := time.Now()
	rc := &rpcClient{
		ID:   int64(uint32(t.Second() * t.Nanosecond() * common.Goid())),
		opts: opts,
	}
	log.Info("client initial ID:%d", rc.ID)

	return rc
}

func (r *rpcClient) next(request Request, opts CallOptions) (selector.Next, error) {
	// return remote address
	if nil != opts.Next {
		return opts.Next, nil
	}

	// get next nodes from the selector
	return r.opts.Selector.Select(request.ServiceConfig())
}

func (c *rpcClient) call(ctx context.Context, reqID int64, service registry.ServiceURL,
	cltRequest Request, rsp interface{}, opts CallOptions) error {
	req, ok := cltRequest.(*rpcRequest)
	if !ok {
		return jerrors.New(fmt.Sprintf("@request is not of type Request", cltRequest))
	}

	reqTimeout := opts.RequestTimeout
	if len(service.Query.Get("timeout")) != 0 {
		if timeout, err := strconv.Atoi(service.Query.Get("timeout")); err == nil {
			timeoutDuration := time.Duration(timeout) * time.Millisecond
			if timeoutDuration < reqTimeout {
				reqTimeout = timeoutDuration
			}
		}
	}
	if reqTimeout <= 0 {
		reqTimeout = DefaultRequestTimeout
	}

	// create client package
	pkg := &Package{}
	pkg.Header = make(map[string]string)
	if md, ok := ctx.Value(common.DUBBOGO_CTX_KEY).(map[string]string); ok {
		for k := range md {
			pkg.Header[k] = md[k]
		}

		// set timeout in nanoseconds
		pkg.Header["Timeout"] = fmt.Sprintf("%d", reqTimeout)
		// set the content type for the request
		pkg.Header["Content-Type"] = req.protocol
		// set the accept header
		pkg.Header["Accept"] = req.contentType
	}

	codec := c.opts.newCodec()

	ch := make(chan error, 1)
	go func() {
		var (
			err    error
			rpcReq Message
			buf    []byte
		)
		defer func() {
			if panicMsg := recover(); panicMsg != nil {
				if msg, ok := panicMsg.(string); ok {
					ch <- jerrors.New(strconv.Itoa(int(reqID)) + " request error, panic msg:" + msg)
				} else {
					ch <- jerrors.New("request error")
				}
			}
		}()

		rpcReq = Message{
			Version:     req.version,
			ServicePath: strings.TrimPrefix(service.Path, "/"),
			Target:      req.ServiceConfig().(*registry.ServiceConfig).Service,
			ID:          reqID,
			Method:      req.method,
			Timeout:     reqTimeout,
			Header:      map[string]string{},
			Args:        req.args,
		}
		if pkg.Body, err = codec.Write(&rpcReq); err != nil {
			ch <- err
			return
		}

		if buf, err = httpSendRecv(service.Location, service.Path, opts.DialTimeout, pkg); err != nil {
			ch <- err
			return
		}
		ch <- codec.Read(buf, rsp)
	}()

	select {
	case err := <-ch:
		return jerrors.Trace(err)
	case <-ctx.Done():
		return jerrors.Trace(ctx.Err())
	}

	return nil
}

func (c *rpcClient) Call(ctx context.Context, request Request, response interface{}, opts ...CallOption) error {
	reqID := atomic.AddInt64(&c.ID, 1)
	// make a copy of call opts
	callOpts := c.opts.CallOptions
	for _, opt := range opts {
		opt(&callOpts)
	}

	// get next nodes selection func from the selector
	next, err := c.next(request, callOpts)
	if err != nil {
		log.Error("selector.Select(request{%#v}) = error{%#v}", request, err)
		return jerrors.Trace(err)
	}

	// check if we already have a deadline
	d, ok := ctx.Deadline()
	log.Debug("ctx:%#v, d:%#v, ok:%v", ctx, d, ok)
	if !ok {
		// no deadline so we create a new one
		log.Debug("create timeout context, timeout:%v", callOpts.RequestTimeout)
		ctx, _ = context.WithTimeout(ctx, callOpts.RequestTimeout)
	} else {
		// got a deadline so no need to setup context
		// but we need to set the timeout we pass along
		opt := WithRequestTimeout(d.Sub(time.Now()))
		log.Debug("WithRequestTimeout:%#v", d.Sub(time.Now()))
		opt(&callOpts)
	}

	select {
	case <-ctx.Done():
		return jerrors.Trace(ctx.Err())
	default:
	}

	call := func(i int) error {
		// select next node
		serviceURL, err := next(reqID)
		if err != nil {
			log.Error("selector.next(request{%#v}, reqID{%d}) = error{%#v}", request, reqID, err)
			return jerrors.Trace(err)
		}

		err = c.call(ctx, reqID, *serviceURL, request, response, callOpts)
		log.Debug("@i{%d}, call(ID{%v}, ctx{%v}, serviceURL{%s}, request{%v}, response{%v}) = err{%v}",
			i, reqID, ctx, serviceURL, request, response, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}

	var (
		gerr error
		ch   chan error
	)
	ch = make(chan error, callOpts.Retries)
	for i := 0; i < callOpts.Retries; i++ {
		var index = i
		go func() {
			ch <- call(index)
		}()

		select {
		case <-ctx.Done():
			log.Error("reqID{%d}, @i{%d}, ctx.Done(), ctx.Err:%#v", reqID, i, ctx.Err())
			return jerrors.Trace(ctx.Err())
		case err := <-ch:
			log.Debug("reqID{%d}, err:%+v", reqID, err)
			if err == nil || len(err.Error()) == 0 {
				return nil
			}
			log.Error("reqID{%d}, @i{%d}, err{%+v}", reqID, i, jerrors.ErrorStack(err))
			gerr = jerrors.Trace(err)
		}
	}

	return gerr
}

func (c *rpcClient) Options() Options {
	return c.opts
}

func (c *rpcClient) NewRequest(group, version, service, method string, args interface{}, reqOpts ...RequestOption) Request {
	codecType := c.opts.CodecType.String()

	var opts RequestOptions
	for _, o := range reqOpts {
		o(&opts)
	}

	return &rpcRequest{
		group:       group,
		protocol:    codecType,
		version:     version,
		service:     service,
		method:      method,
		args:        args,
		contentType: codec2ContentType[codecType],
		opts:        opts,
	}
}

func (c *rpcClient) String() string {
	return "dubbogo-client"
}

func (c *rpcClient) Close() {
	c.once.Do(func() {
		if c.opts.Selector != nil {
			c.opts.Selector.Close()
			c.opts.Selector = nil
		}
		if c.opts.Registry != nil {
			c.opts.Registry.Close()
			c.opts.Registry = nil
		}
	})
}
