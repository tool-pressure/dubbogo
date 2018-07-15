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
	"time"
)

import (
	"github.com/AlexStocks/dubbogo/registry"
	"github.com/AlexStocks/dubbogo/registry/zk"
	"github.com/AlexStocks/dubbogo/selector"
	"github.com/AlexStocks/dubbogo/selector/cache"
)

// Client is the interface used to make requests to services.
// It supports Request/Response via Transport and Publishing via the Broker.
// It also supports bidirectional streaming of requests.
type Client interface {
	Options() Options
	NewRequest(group, version, service, method string, args interface{}, reqOpts ...RequestOption) Request
	Call(ctx context.Context, req Request, rsp interface{}, opts ...CallOption) error
	String() string
	Close()
}

type Request interface {
	ServiceConfig() registry.ServiceConfigIf
}

type (
	// Option used by the Client
	Option func(*Options)
	// CallOption used by Call or Stream
	CallOption func(*CallOptions)
	// RequestOption used by NewRequest
	RequestOption func(*RequestOptions)
)

type (
	dubbogoClientConfig struct {
		codecType CodecType
		newCodec  NewCodec
	}
)

var (
	// DefaultRetries is the default number of times a request is tried
	DefaultRetries = 1
	// DefaultRequestTimeout is the default request timeout
	DefaultRequestTimeout = time.Second * 5

	contentType2Codec = map[string]NewCodec{
		"application/json":    newJsonClientCodec,
		"application/jsonrpc": newJsonClientCodec,
	}

	codec2ContentType = map[string]string{
		"jsonrpc": "application/json",
	}

	dubbogoClientConfigMap = map[CodecType]dubbogoClientConfig{
		CODECTYPE_JSONRPC: dubbogoClientConfig{
			codecType: CODECTYPE_JSONRPC,
			newCodec:  newJsonClientCodec,
		},
	}

	DefaultRegistries = map[string]registry.NewRegistry{
		"zookeeper": zookeeper.NewConsumerZookeeperRegistry,
	}

	DefaultSelectors = map[string]selector.NewSelector{
		"cache": cache.NewSelector,
	}
)

// creates a new client with the options passed in
func NewClient(opt ...Option) Client {
	return newRPCClient(opt...)
}
