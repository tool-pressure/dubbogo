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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	MAX_JSONRPC_ID = 0x7FFFFFFF
	VERSION        = "2.0"
)

//////////////////////////////////////////
// codec type
//////////////////////////////////////////

type CodecType int

const (
	CODECTYPE_UNKNOWN CodecType = iota
	CODECTYPE_JSONRPC
)

var codecTypeStrings = [...]string{
	"unknown",
	"jsonrpc",
}

func (c CodecType) String() string {
	typ := CODECTYPE_UNKNOWN
	switch c {
	case CODECTYPE_JSONRPC:
		typ = c
	}

	return codecTypeStrings[typ]
}

func GetCodecType(t string) CodecType {
	var typ = CODECTYPE_UNKNOWN

	switch t {
	case codecTypeStrings[CODECTYPE_JSONRPC]:
		typ = CODECTYPE_JSONRPC
	}

	return typ
}

type Codec interface {
	Read([]byte, interface{}) error
	Write(m *Message) ([]byte, error)
}

type NewCodec func() Codec

type Message struct {
	ID          int64
	Version     string
	ServicePath string // service path
	Target      string // Service
	Method      string
	Timeout     time.Duration // request timeout
	Header      map[string]string
	Args        interface{}
	BodyLen     int
	Error       string
}

var (
	// Actual returned error may have different message.
	errInternal    = NewError(-32603, "Internal error")
	errServerError = NewError(-32001, "jsonrpc2.Error: json.Marshal failed")
)

// Error represent JSON-RPC 2.0 "Error object".
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewError returns an Error with given code and message.
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// Error returns JSON representation of Error.
func (e *Error) Error() string {
	buf, err := json.Marshal(e)
	if err != nil {
		msg, err := json.Marshal(err.Error())
		if err != nil {
			msg = []byte(`"` + errServerError.Message + `"`)
		}
		return fmt.Sprintf(`{"code":%d,"message":%s}`, errServerError.Code, string(msg))
	}
	return string(buf)
}

type jsonClientCodec struct {
	// temporary work space
	req  clientRequest
	resp clientResponse

	pending map[int64]string
}

type clientRequest struct {
	Version string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
	ID      int64       `json:"id"`
}

type clientResponse struct {
	Version string           `json:"jsonrpc"`
	ID      int64            `json:"id"`
	Result  *json.RawMessage `json:"result,omitempty"`
	Error   *Error           `json:"error,omitempty"`
}

func (r *clientResponse) reset() {
	r.Version = ""
	r.ID = 0
	r.Result = nil
	r.Error = nil
}

func newJsonClientCodec() Codec {
	return &jsonClientCodec{
		pending: make(map[int64]string),
	}
}

func (c *jsonClientCodec) Write(m *Message) ([]byte, error) {
	// If return error: it will be returned as is for this call.
	// Allow param to be only Array, Slice, Map or Struct.
	// When param is nil or uninitialized Map or Slice - omit "params".
	param := m.Args
	if param != nil {
		switch k := reflect.TypeOf(param).Kind(); k {
		case reflect.Map:
			if reflect.TypeOf(param).Key().Kind() == reflect.String {
				if reflect.ValueOf(param).IsNil() {
					param = nil
				}
			}
		case reflect.Slice:
			if reflect.ValueOf(param).IsNil() {
				param = nil
			}
		case reflect.Array, reflect.Struct:
		case reflect.Ptr:
			switch k := reflect.TypeOf(param).Elem().Kind(); k {
			case reflect.Map:
				if reflect.TypeOf(param).Elem().Key().Kind() == reflect.String {
					if reflect.ValueOf(param).Elem().IsNil() {
						param = nil
					}
				}
			case reflect.Slice:
				if reflect.ValueOf(param).Elem().IsNil() {
					param = nil
				}
			case reflect.Array, reflect.Struct:
			default:
				return nil, NewError(errInternal.Code, "unsupported param type: Ptr to "+k.String())
			}
		default:
			return nil, NewError(errInternal.Code, "unsupported param type: "+k.String())
		}
	}

	c.req.Version = VERSION
	c.req.Method = m.Method
	c.req.Params = param
	c.req.ID = m.ID & MAX_JSONRPC_ID
	// c.pending[m.ID] = m.Method // 此处如果用m.ID会导致error: can not find method of response id 280698512
	c.pending[c.req.ID] = m.Method

	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(&c.req); err != nil {
		return nil, jerrors.Trace(err)
	}

	return buf.Bytes(), nil
}

func (c *jsonClientCodec) Read(streamBytes []byte, x interface{}) error {
	c.resp.reset()

	buf := bytes.NewBuffer(streamBytes)
	dec := json.NewDecoder(buf)
	if err := dec.Decode(&c.resp); err != nil {
		if err != io.EOF {
			err = NewError(errInternal.Code, err.Error())
		}
		return err
	}

	var (
		ok  bool
		err error
		m   Message
	)
	m.Method, ok = c.pending[c.resp.ID]
	if !ok {
		err := jerrors.Errorf("can not find method of response id %v, response error:%v", c.resp.ID, c.resp.Error)
		return err
	}
	delete(c.pending, c.resp.ID)

	m.Error = ""
	m.ID = c.resp.ID
	if c.resp.Error != nil {
		m.Error = c.resp.Error.Error()
		err = jerrors.New(m.Error)
	} else {
		err = json.Unmarshal(*c.resp.Result, x)
	}

	return err
}
