/******************************************************
# DESC    : client.go unit test
# AUTHOR  : Alex Stocks
# EMAIL   : alexstocks@foxmail.com
# MOD     : 2016-10-22 20:26
# FILE    : client_test.go
******************************************************/

package hessian

import (
	"bytes"
	"fmt"
	"log"
	"testing"
)

const (
	MATH_URL     = "http://localhost:8000/math"     //整数四则运算hessian测试接口
	DATATYPE_URL = "http://localhost:8000/datatype" //数据类型测试结果,传入参数，返回该参数的(响应)编码结果
)

// attention that: the server is https://github.com/AlexStocks/dubbogo-examples/tree/master/calculator/java-server

func TestHttpPost(t *testing.T) {
	log.Println("Test_request")
	data := bytes.NewBuffer([]byte{0, 1, 3, 4})
	rb, _ := httpPost(DATATYPE_URL, bytes.NewReader(data.Bytes()))
	log.Println(rb)
	log.Println(string(rb))
}

//整数 数学运算测试
func TestMath(t *testing.T) {
	fmt.Println("test math")
	res, err := Request(MATH_URL, "Add", 100, 200)
	fmt.Println("Add(100, 200) = res:", res, ", err:", err)
	res, err = Request(MATH_URL, "Sub", 100, 200)
	fmt.Println("Sub(100, 200) = res:", res, ", err:", err)
	res, err = Request(MATH_URL, "Mul", 100, 200)
	fmt.Println("Mul(100, 200) = res:", res, ", err:", err)
	res, err = Request(MATH_URL, "Div", 200, 50)
	fmt.Println("Div(200, 50) = res:", res, ", err:", err)
}

//数据类型测试
func Test_request_data_type(t *testing.T) {
	Request(DATATYPE_URL, "dataBytes", []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	Request(DATATYPE_URL, "dataBoolean", true)
	Request(DATATYPE_URL, "dataBoolean", false)
	Request(DATATYPE_URL, "dataDouble", 1989.0604)

	list := []Any{100, 10.001, "不厌其烦", []byte{0, 2, 4, 6, 8, 10}, true, nil, false}
	Request(DATATYPE_URL, "dataList", list)

	var hmap = make(map[Any]Any)
	hmap["你好"] = "哈哈哈"
	hmap[100] = "嘿嘿"
	hmap[100.1010] = 101910
	hmap[true] = true
	hmap[false] = true
	Request(DATATYPE_URL, "dataMap", hmap)

	Request(DATATYPE_URL, "dataMapNoParam")

	Request(DATATYPE_URL, "dataNull")

	Request(DATATYPE_URL, "dataString", "_BEGIN_兔兔你小姨子_END_")

	Request(DATATYPE_URL, "dataInt", 1000)

}

//异常测试,服务器已经抛出异常，但客户端看到的是200和空响应
//curl -vvv --data-binary "c\x00\x01m\x00\adataIntz" -H "Content-Type: application/binary" http://localhost:8080/HessianTest/dt
//curl -vvv --data-binary "c\x00\x01m\x00\x0EthorwExceptionz" -H "Content-Type: application/binary" http://localhost:8080/HessianTest/dt
func Test_request_exception(t *testing.T) {
	// Request(DATATYPE_URL,"dataInt")
	Request(DATATYPE_URL, "thorwException")
}
