/******************************************************
# DESC    : hessian decode
# AUTHOR  : Alex Stocks
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-10-22 20:25
# FILE    : decode.go
******************************************************/

package hessian

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"reflect"
	"time"
)

const (
	PARSE_DEBUG = true
)

type Hessian struct {
	reader *bufio.Reader
	refs   []Any
}

var (
	ErrNotEnoughBuf    = fmt.Errorf("not enough buf")
	ErrIllegalRefIndex = fmt.Errorf("illegal ref index")
	typeReg            = make(map[string]reflect.Type)
)

// func NewHessian(r io.Reader) *Hessian {
// 	return &Hessian{reader: bufio.NewReader(r)}
// }

// func NewHessianWithBuf(b []byte) *Hessian {
// 	return NewHessian(bytes.NewReader(b))
// }

func NewHessian(b []byte) *Hessian {
	return &Hessian{reader: bufio.NewReader(bytes.NewReader(b))}
}

//读取当前字节,指针不前移
func (this *Hessian) peekByte() byte {
	return this.peek(1)[0]
}

//添加引用
func (this *Hessian) appendRefs(v interface{}) {
	this.refs = append(this.refs, v)
}

//获取缓冲长度
func (this *Hessian) len() int {
	this.peek(1) //需要先读一下资源才能得到已缓冲的长度
	return this.reader.Buffered()
}

//读取 Hessian 结构中的一个字节,并后移一个字节
func (this *Hessian) readByte() (byte, error) {
	return this.reader.ReadByte()
}

//读取指定长度的字节,并后移len(b)个字节
func (this *Hessian) next(b []byte) (int, error) {
	return this.reader.Read(b)
}

//读取指定长度字节,指针不后移
// func (this *Hessian) peek(n int) ([]byte, error) {
func (this *Hessian) peek(n int) []byte {
	// return this.reader.Peek(n)
	b, _ := this.reader.Peek(n)
	return b
}

//读取len(s)的 utf8 字符
func (this *Hessian) nextRune(s []rune) []rune {
	var (
		n  int
		i  int
		r  rune
		ri int
		e  error
	)

	n = len(s)
	s = s[:0]
	for i = 0; i < n; i++ {
		if r, ri, e = this.reader.ReadRune(); e == nil && ri > 0 {
			s = append(s, r)
		}
	}

	return s
}

//读取数据类型描述,用于 list 和 map
func (this *Hessian) readType() string {
	if this.peekByte() != byte('t') {
		return ""
	}

	var tLen = UnpackInt16(this.peek(3)[1:3]) // 取类型字符串长度
	var b = make([]rune, 3+tLen)
	return string(this.nextRune(b)[3:]) //取类型名称
}

// 解析struct
func showReg() {
	for k, v := range typeReg {
		fmt.Println("-->> show Registered types <<----")
		fmt.Println(k, v)
	}

}

func Reg(s string, t reflect.Type) {
	typeReg[s] = t
}

//checkExists
func hasReg(s string) bool {
	if len(s) == 0 {
		return false
	}
	_, exists := typeReg[s]
	return exists
}

//gen a new Instance of s type
func gen(s string) interface{} {
	var ret interface{}
	t, ok := typeReg[s]
	if !ok {
		return nil
	}
	ret = reflect.New(t).Interface()
	fmt.Println("gen", t)
	return ret
}

//解析 hessian 数据包
func (this *Hessian) Parse() (interface{}, error) {
	var (
		err error
		t   byte
		l   int
		a   []byte
		s   []byte
	)

	a = make([]byte, 16)
	t, err = this.readByte()
	if err == io.EOF {
		return nil, err
	}
	switch t {
	case 'N': //null
		return nil, nil

	case 'T': //true
		return true, nil

	case 'F': //false
		return false, nil

	case 'I': //int
		s = a[:4]
		l, err = this.next(s)
		if err != nil {
			return nil, err
		}
		if l != 4 {
			return nil, ErrNotEnoughBuf
		}
		return UnpackInt32(s), nil

	case 'L': //long
		s = a[:8]
		l, err = this.next(s)
		if err != nil {
			return nil, err
		}
		if l != 8 {
			return nil, ErrNotEnoughBuf
		}
		return UnpackInt64(s), nil

	case 'd': //date
		s = a[:8]
		l, err = this.next(s)
		if err != nil {
			return nil, err
		}
		if l != 8 {
			return nil, ErrNotEnoughBuf
		}
		var ms = UnpackInt64(s)
		return time.Unix(ms/1000, ms%1000*10e5), nil

	case 'D': //double
		s = a[:8]
		l, err = this.next(s)
		if err != nil {
			return nil, err
		}
		if l != 8 {
			return nil, ErrNotEnoughBuf
		}
		return UnpackFloat64(s), nil

	case 'S', 's', 'X', 'x': //string,xml
		var (
			rBuf   []rune
			chunks []rune
		)
		rBuf = make([]rune, CHUNK_SIZE)
		for { //避免递归读取 Chunks
			s = a[:2]
			l, err = this.next(s)
			if err != nil {
				return nil, err
			}
			if l != 2 {
				return nil, ErrNotEnoughBuf
			}
			l = int(UnpackInt16(s))
			chunks = append(chunks, this.nextRune(rBuf[:l])...)
			if t == 'S' || t == 'X' {
				break
			}
			if t, err = this.readByte(); err != nil {
				return nil, err
			}
		}
		return string(chunks), nil

	case 'B', 'b': //binary
		var (
			buf    []byte
			chunks []byte //等同于 []uint8,在 反射判断类型的时候，会得到 []uint8
		)
		buf = make([]byte, CHUNK_SIZE)
		for { //避免递归读取 Chunks
			s = a[:2]
			l, err = this.next(s)
			if err != nil {
				return nil, err
			}
			if l != 2 {
				return nil, ErrNotEnoughBuf
			}
			l = int(UnpackInt16(s))
			if l, err = this.next(buf[:l]); err != nil {
				return nil, err
			}
			chunks = append(chunks, buf[:l]...)
			if t == 'B' {
				break
			}
			if t, err = this.readByte(); err != nil {
				return nil, err
			}
		}

		return chunks, nil

	case 'V': //list
		var (
			v      Any
			chunks []Any
		)
		this.readType() // 忽略
		if this.peekByte() == byte('l') {
			this.next(a[:5])
		}
		for this.peekByte() != byte('z') {
			if v, err = this.Parse(); err != nil {
				return nil, err
			} else {
				chunks = append(chunks, v)
			}
		}
		this.readByte()
		this.appendRefs(&chunks)
		return chunks, nil

	case 'M': //map
		var (
			k          Any
			v          Any
			t          string
			key        interface{}
			value      interface{}
			tmpV       interface{}
			chunks     map[Any]Any
			keyName    string
			methodName string
			nv         reflect.Value
			args       []reflect.Value
		)

		t = this.readType()
		if !hasReg(t) {
			chunks = make(map[Any]Any)
			// this.readType() // 忽略
			for this.peekByte() != byte('z') {
				k, err = this.Parse()
				if err != nil {
					return nil, err
				}
				v, err = this.Parse()
				if err != nil {
					return nil, err
				}
				chunks[k] = v
			}
			this.readByte()
			this.appendRefs(&chunks)
			return chunks, nil
		} else {
			tmpV = gen(t)
			for this.peekByte() != 'z' {
				key, err = this.Parse()
				if err != nil {
					return nil, err
				}
				value, err = this.Parse()
				if err != nil {
					return nil, err
				}
				//set value of the struct     Zero will be passed
				if nv = reflect.ValueOf(value); nv.IsValid() {
					keyName = key.(string)
					if keyName[0] >= 'a' { //convert to Upper
						methodName = "Set" + string(keyName[0]-32) + keyName[1:]
					} else {
						methodName = "Set" + keyName
					}

					args = args[:0]
					args = append(args, reflect.ValueOf(value))
					reflect.ValueOf(tmpV).MethodByName(methodName).Call(args)
				}
			}
			// v = tmpV
			this.appendRefs(&tmpV)
			return tmpV, nil
		}

	case 'f': //fault
		this.Parse() //drop "code"
		code, _ := this.Parse()
		this.Parse() //drop "message"
		message, _ := this.Parse()
		return nil, fmt.Errorf("%s : %s", code, message)

	case 'r': //reply
		// valid-reply ::= r x01 x00 header* object z
		// fault-reply ::= r x01 x00 header* fault z
		this.next(a[:2])
		return this.Parse()

	case 'R': //ref, 一个整数，用以指代前面的list 或者 map
		s = a[:4]
		l, err = this.next(s)
		if err != nil {
			return nil, err
		}
		if l != 4 {
			return nil, ErrNotEnoughBuf
		}
		l = int(UnpackInt32(s)) // ref index

		if len(this.refs) <= l {
			return nil, ErrIllegalRefIndex
		}
		return &this.refs[l], nil

	default:
		return nil, fmt.Errorf("Invalid type: %v,>>%v<<<", string(t), this.peek(this.len()))
	}
}
