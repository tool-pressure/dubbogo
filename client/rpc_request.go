/******************************************************
# DESC    : rpc client request
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-30 10:45
# FILE    : rpc_request.go
******************************************************/

package client

import (
	"github.com/AlexStocks/dubbogo/registry"
)

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

func newRPCRequest(group, protocol, version, service, method string, args interface{},
	contentType string, reqOpts ...RequestOption) Request {

	var opts RequestOptions
	for _, o := range reqOpts {
		o(&opts)
	}

	return &rpcRequest{
		group:       group,
		protocol:    protocol,
		version:     version,
		service:     service,
		method:      method,
		args:        args,
		contentType: contentType,
		opts:        opts,
	}
}

func (r *rpcRequest) Protocol() string {
	return r.protocol
}

func (r *rpcRequest) Version() string {
	return r.version
}

func (r *rpcRequest) ContentType() string {
	return r.contentType
}

func (r *rpcRequest) ServiceConfig() registry.ServiceConfigIf {
	return &registry.ServiceConfig{
		Protocol: r.protocol,
		Service:  r.service,
		Group:    r.group,
		Version:  r.version,
	}
}

func (r *rpcRequest) Method() string {
	return r.method
}

func (r *rpcRequest) Args() interface{} {
	return r.args
}

func (r *rpcRequest) Stream() bool {
	return r.opts.Stream
}

func (r *rpcRequest) Options() RequestOptions {
	return r.opts
}
