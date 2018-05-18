/******************************************************
# DESC    : dubbogo constants
# AUTHOR  : Alex Stocks
# VERSION : 1.0
# LICENCE : Apache Licence 2.0
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-06-22 21:30
# FILE    : misc.go
******************************************************/

package common

import (
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	DUBBOGO_CTX_KEY = "dubbogo-ctx"
)

var (
	src = rand.NewSource(time.Now().UnixNano())
)

func TimeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}

func Contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}

	return false
}

func Future(sec int, f func()) {
	time.AfterFunc(TimeSecondDuration(sec), f)
}

func TrimPrefix(s string, prefix string) string {
	if strings.HasPrefix(s, prefix) {
		s = s[len(prefix):]
	}
	return s
}

func TrimSuffix(s string, suffix string) string {
	if strings.HasSuffix(s, suffix) {
		s = s[:len(s)-len(suffix)]
	}
	return s
}

func Goid() int {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}
