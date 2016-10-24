/******************************************************
# DESC    : hessian encode
# AUTHOR  : Alex Stocks
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-10-22 20:24
# FILE    : encode.go
******************************************************/

package hessian

import (
	"bytes"
	"log"
	"runtime"
	"time"
	"unicode/utf8"
)

import (
	"github.com/AlexStocks/goext/strings"
)

// interface{} 的别名
type Any interface{}

/*
nil bool int8 int32 int64 float64 time.Time
string []byte []interface{} map[interface{}]interface{}
*/

type Encoder struct {
}

const (
	CHUNK_SIZE    = 0x8000
	ENCODER_DEBUG = false
)

func init() {
	_, filename, _, _ := runtime.Caller(1)
	if ENCODER_DEBUG {
		log.SetPrefix(filename + "\n")
	}
}

func Encode(v interface{}, b []byte) []byte {
	switch v.(type) {
	case nil:
		return encNull(b)

	case bool:
		b = encBool(v.(bool), b)

	case int:
		if v.(int) >= -2147483648 && v.(int) <= 2147483647 {
			b = encInt32(int32(v.(int)), b)
		} else {
			b = encInt64(int64(v.(int)), b)
		}

	case int32:
		b = encInt32(v.(int32), b)

	case int64:
		b = encInt64(v.(int64), b)

	case time.Time:
		b = encDate(v.(time.Time), b)

	case float64:
		b = encFloat(v.(float64), b)

	case string:
		b = encString(v.(string), b)

	case []byte:
		b = encBinary(v.([]byte), b)

	case []Any:
		b = encList(v.([]Any), b)

	case map[Any]Any:
		b = encMap(v.(map[Any]Any), b)

	default:
		panic("unknow type")
	}

	if ENCODER_DEBUG {
		log.Println(SprintHex(b))
	}

	return b
}

//=====================================
//对各种数据类型的编码
//=====================================

// null
func encNull(b []byte) []byte {
	return append(b, 'N')
}

// boolean
func encBool(v bool, b []byte) []byte {
	var c byte = 'F'
	if v == true {
		c = 'T'
	}

	return append(b, c)
}

// int
func encInt32(v int32, b []byte) []byte {
	b = append(b, 'I')
	// return PackInt32(v, b)
	return append(b, PackInt32(v)...)
}

// long
func encInt64(v int64, b []byte) []byte {
	b = append(b, 'L')
	// return PackInt64(v, b)
	return append(b, PackInt64(v)...)
}

// date
func encDate(v time.Time, b []byte) []byte {
	b = append(b, 'd')
	// return PackInt64(v.UnixNano()/1e6, b)
	return append(b, PackInt64(v.UnixNano()/1e6)...)
}

// double
func encFloat(v float64, b []byte) []byte {
	b = append(b, 'D')
	// return PackFloat64(v, b)
	return append(b, PackFloat64(v)...)
}

// string
func encString(v string, b []byte) []byte {
	var (
		vBuf = *bytes.NewBufferString(v)
		vLen = utf8.RuneCountInString(v)

		vChunk = func(length int) {
			for i := 0; i < length; i++ {
				if r, s, err := vBuf.ReadRune(); s > 0 && err == nil {
					// b = append(b, []byte(string(r))...)
					b = append(b, gxstrings.Slice(string(r))...)
				}
			}
		}
	)

	if v == "" {
		b = append(b, 'S')
		// b = PackUint16(uint16(vLen), b)
		b = append(b, PackUint16(uint16(vLen))...)
		b = append(b, []byte{}...)
		return b
	}

	for {
		vLen = utf8.RuneCount(vBuf.Bytes())
		if vLen == 0 {
			break
		}
		if vLen > CHUNK_SIZE {
			b = append(b, 's')
			// b = PackUint16(uint16(CHUNK_SIZE), b)
			b = append(b, PackUint16(uint16(CHUNK_SIZE))...)
			vChunk(CHUNK_SIZE)
		} else {
			b = append(b, 'S')
			// b = PackUint16(uint16(vLen), b)
			b = append(b, PackUint16(uint16(vLen))...)
			vChunk(vLen)
		}
	}

	return b
}

// binary
func encBinary(v []byte, b []byte) []byte {
	var (
		tag     byte
		length  uint16
		vLength int
	)

	if len(v) == 0 {
		b = append(b, 'B')
		// b = PackUint16(0, b)
		b = append(b, PackUint16(0)...)
		return b
	}

	// vBuf := *bytes.NewBuffer(v)
	// for vBuf.Len() > 0 {
	vLength = len(v)
	for vLength > 0 {
		// if vBuf.Len() > CHUNK_SIZE {
		if len(v) > CHUNK_SIZE {
			tag = 'b'
			length = uint16(CHUNK_SIZE)
		} else {
			tag = 'B'
			// length = uint16(vBuf.Len())
			length = uint16(len(v))
		}

		b = append(b, tag)
		// b = PackUint16(length, b)
		b = append(b, PackUint16(length)...)
		// b = append(b, vBuf.Next(length)...)
		b = append(b, v[:length]...)
		v = v[length:]
		vLength = len(v)
	}

	return b
}

// list
func encList(v []Any, b []byte) []byte {
	b = append(b, 'V')

	b = append(b, 'l')
	// b = PackInt32(int32(len(v)), b)
	b = append(b, PackInt32(int32(len(v)))...)

	for _, a := range v {
		b = Encode(a, b)
	}

	b = append(b, 'z')

	return b
}

// map
func encMap(v map[Any]Any, b []byte) []byte {
	b = append(b, 'M')

	for k, v := range v {
		b = Encode(k, b)
		b = Encode(v, b)
	}

	b = append(b, 'z')

	return b
}
