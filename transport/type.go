/******************************************************
# DESC    : transport type
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache License 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2017-10-19 18:06
# FILE    : type.go
******************************************************/

package transport

import (
	"strconv"
)

type Type int32

const (
	TCP  Type = 0
	UDP  Type = 1
	HTTP Type = 2
)

var Type_name = map[int32]string{
	0: "TCP",
	1: "UDP",
	2: "HTTP",
}

var Type_value = map[string]int32{
	"TCP":  0,
	"UDP":  1,
	"HTTP": 2,
}

func (x Type) String() string {
	s, ok := Type_name[int32(x)]
	if ok {
		return s
	}

	return strconv.Itoa(int(x))
}
