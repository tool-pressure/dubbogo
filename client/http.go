package client

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

const (
	PathPrefix = byte('/')
)

type Package struct {
	Header map[string]string
	Body   []byte
}

//////////////////////////////////////////////
// http transport client
//////////////////////////////////////////////

func SetNetConnTimeout(conn net.Conn, timeout time.Duration) {
	t := time.Time{}
	if timeout > time.Duration(0) {
		t = time.Now().Add(timeout)
	}

	conn.SetReadDeadline(t)
}

// !!The high level of complexity and the likelihood that the fasthttp client has not been extensively used
// in production means that you would need to expect a very large benefit to justify the adoption of fasthttp today.
// from: http://big-elephants.com/2016-12/fasthttp-client/
func httpSendRecv(addr, path string, timeout time.Duration, pkg *Package) ([]byte, error) {
	tcpConn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	defer tcpConn.Close()

	header := make(http.Header)

	for k, v := range pkg.Header {
		header.Set(k, v)
	}

	req := &http.Request{
		Method: "POST",
		URL: &url.URL{
			Scheme: "http",
			Path:   path,
		},
		Header:        header,
		Body:          ioutil.NopCloser(bytes.NewReader(pkg.Body)),
		ContentLength: int64(len(pkg.Body)),
	}

	if timeout > time.Duration(0) {
		SetNetConnTimeout(tcpConn, timeout)
		defer SetNetConnTimeout(tcpConn, 0)
	}

	reqBuf := bytes.NewBuffer(make([]byte, 0))
	if err := req.Write(reqBuf); err != nil {
		return nil, jerrors.Trace(err)
	}

	if _, err := reqBuf.WriteTo(tcpConn); err != nil {
		return nil, jerrors.Trace(err)
	}

	rsp, err := http.ReadResponse(bufio.NewReader(tcpConn), req)
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	defer rsp.Body.Close()

	b, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	if rsp.StatusCode != 200 {
		return nil, jerrors.New(rsp.Status + ": " + string(b))
	}

	var rspBytes []byte

	return append(rspBytes, b...), nil
}
