# dubbo 分析 #
---
*written by AlexStocks*

## dubbo & zk ##
---

dubbo.xml 相关内容如下:

  <bean id="demoService" class="com.ikurento.user.UserProviderImpl" />
  <bean id="otherService" class="com.ikurento.user.UserProviderAnotherImpl"/>

  <dubbo:service interface="com.ikurento.user.UserProvider" ref="demoService"/>
  <dubbo:service interface="com.ikurento.user.UserProvider" ref="otherService" version="2.0"/>

  <dubbo:protocol id="dubbo" name="dubbo" port="20000" />
  <dubbo:protocol id="jsonrpc" name="jsonrpc" port="10000" />

  zk path(/dubbo/com.ikurento.user.UserProvider/providers)下的url列表如下：

   jsonrpc://192.168.35.1:10000/com.ikurento.user.UserProvider2?anyhost=true&application=user-info-server&default.timeout=10000&dubbo=2.4.10&environment=product&interface=com.ikurento.user.UserProvider&methods=getUser,isLimit,queryAll,queryUser,GetUser&owner=AlexStocks&pid=8604&revision=0.1.0&side=provider&timestamp=1476890290246&version=2.0

   jsonrpc://192.168.35.1:10000/com.ikurento.user.UserProvider3?anyhost=true&application=user-info-server&default.timeout=10000&dubbo=2.4.10&environment=product&group=as&interface=com.ikurento.user.UserProvider&methods=getUser,isLimit,queryAll,queryUser,GetUser&owner=AlexStocks&pid=8604&revision=0.1.0&side=provider&timestamp=1476890290293&version=2.0

   jsonrpc://192.168.35.1:10000/com.ikurento.user.UserProvider?anyhost=true&application=user-info-server&default.timeout=10000&dubbo=2.4.10&environment=product&interface=com.ikurento.user.UserProvider&methods=getUser,isLimit,queryAll,queryUser,GetUser&owner=AlexStocks&pid=8604&revision=0.1.0&side=provider&timestamp=1476890289997

   dubbo://192.168.35.1:20000/com.ikurento.user.UserProvider2?anyhost=true&application=user-info-server&default.timeout=10000&dubbo=2.4.10&environment=product&interface=com.ikurento.user.UserProvider&methods=getUser,isLimit,queryAll,queryUser,GetUser&owner=AlexStocks&pid=8604&revision=0.1.0&side=provider&timestamp=1476890290221&version=2.0

   dubbo://192.168.35.1:20000/com.ikurento.user.UserProvider?anyhost=true&application=user-info-server&default.timeout=10000&dubbo=2.4.10&environment=product&interface=com.ikurento.user.UserProvider&methods=getUser,isLimit,queryAll,queryUser,GetUser&owner=AlexStocks&pid=8604&revision=0.1.0&side=provider&timestamp=1476890289330

   dubbo://192.168.35.1:20000/com.ikurento.user.UserProvider3?anyhost=true&application=user-info-server&default.timeout=10000&dubbo=2.4.10&environment=product&group=as&interface=com.ikurento.user.UserProvider&methods=getUser,isLimit,queryAll,queryUser,GetUser&owner=AlexStocks&pid=8604&revision=0.1.0&side=provider&timestamp=1476890290272&version=2.0

  从以上生成结果来看:
  - 1 url path = protocol://host:port/interface + index
  - 2 protocol + interface + group + version唯一的对应一个service(goang的struct)

## hessian & dubbo ##
---

/* http://dubbo.io/User+Guide-zh.htm#UserGuide-zh-hessian%253A%252F%252F */

### hessian version ###
---
Serialization
dubbo, hessian2, java, json

[在Dubbo中使用高效的Java序列化（Kryo和FST）](http://dangdangdotcom.github.io/dubbox/serialization.html)一文中有这么一句话"hessian是一种跨语言的高效二进制序列化方式。但这里实际不是原生的hessian2序列化，而是阿里修改过的hessian lite，它是dubbo RPC默认启用的序列化方式"。

从这句话猜测它用的是hessian2。另外，dubbo可能对hessian协议做了扩展，具体细节可以参考"http://dubbo.io/Developer+Guide-zh.htm"一文中的“协议头约定”一图。

解释
协议格式 header body data

协议头是16字节的定长数据，参见上图dubbo,16*8=128,地址范围0~127

2字节magic字符串0xdabb,0-7高位，8-15低位
1字节的消息标志位。16-20序列id,21 event,22 two way,23请求或响应标识
1字节状态。当消息类型为响应时，设置响应状态。24-31位。
8字节，消息ID,long类型，32-95位。
4字节，消息长度，96-127位
源代码参考:

	// package: com.alibaba.dubbo.remoting.exchange.codec.ExchangeCodec

	protected void encodeRequest(Channel channel, ChannelBuffer buffer, Request req) throws IOException {
		Serialization serialization = getSerialization(channel);
		// header.
		byte[] header = new byte[HEADER_LENGTH];
		// set magic number.
		Bytes.short2bytes(MAGIC, header);

		// set request and serialization flag.
		header[2] = (byte) (FLAG_REQUEST | serialization.getContentTypeId());

		if (req.isTwoWay()) header[2] |= FLAG_TWOWAY;
		if (req.isEvent()) header[2] |= FLAG_EVENT;

		// set request id.
		Bytes.long2bytes(req.getId(), header, 4);

		// encode request data.
		int savedWriteIndex = buffer.writerIndex();
		buffer.writerIndex(savedWriteIndex + HEADER_LENGTH);
		ChannelBufferOutputStream bos = new ChannelBufferOutputStream(buffer);
		ObjectOutput out = serialization.serialize(channel.getUrl(), bos);
		if (req.isEvent()) {
			encodeEventData(channel, out, req.getData());
		} else {
			encodeRequestData(channel, out, req.getData());
		}
		out.flushBuffer();
		bos.flush();
		bos.close();
		int len = bos.writtenBytes();
		checkPayload(channel, len);
		Bytes.int2bytes(len, header, 12);

		// write
		buffer.writerIndex(savedWriteIndex);
		buffer.writeBytes(header); // write header.
		buffer.writerIndex(savedWriteIndex + HEADER_LENGTH + len);
	}

	CollectionSerializer.java:
	public void writeObject(Object obj, AbstractHessianOutput out) throws IOException {
	  if (out.addRef(obj))
		return;

	  Collection list = (Collection) obj;

	  Class cl = obj.getClass();
	  boolean hasEnd;

	  if (obj instanceof Set
			  || !_sendJavaType
			  || !Serializable.class.isAssignableFrom(cl))
		hasEnd = out.writeListBegin(list.size(), null);
	  else
		hasEnd = out.writeListBegin(list.size(), obj.getClass().getName());

	  Iterator iter = list.iterator();
	  while (iter.hasNext()) {
		Object value = iter.next();

		out.writeObject(value);
	  }

	  if (hasEnd)
		out.writeListEnd();
	}

### dubbo conf###
---

Dubbo协议缺省每服务每提供者每消费者使用单一长连接，如果数据量较大，可以使用多个连接。
    <dubbo:protocol name="dubbo" connections="2" />
    <dubbo:service connections=”0”>或<dubbo:reference connections=”0”>表示该服务使用JVM共享长连接。(缺省)
    <dubbo:service connections=”1”>或<dubbo:reference connections=”1”>表示该服务使用独立长连接。
    <dubbo:service connections=”2”>或<dubbo:reference connections=”2”>表示该服务使用独立两条长连接。
为防止被大量连接撑挂，可在服务提供方限制大接收连接数，以实现服务提供方自我保护。
    <dubbo:protocol name="dubbo" accepts="1000" />
    缺省协议，使用基于mina1.1.7+hessian3.2.1的tbremoting交互。

    - 连接个数：单连接
    - 连接方式：长连接
    - 传输协议：TCP
    - 传输方式：NIO异步传输
    - 序列化：Hessian二进制序列化
    - 适用范围：传入传出参数数据包较小（建议小于100K），消费者比提供者个数多，单一消费者无法压满提供者，尽量不要用dubbo协议传输大文件或超大字符串。
    - 适用场景：常规远程服务方法调用

另外，dubbo的配置中可以配置check参数，如下示例:
<dubbo:reference id="userService" interface="com.service.UserService" check="false" async="false" protocol="thrift"></dubbo:reference>
check的意思是client调用服务时检查provider是否启动成功。一般设置为false，因为client先于provider启动了，如果check设置为true则client此时会抛出一个异常.

### dubbo QA ###
---

Q: 为什么要消费者比提供者个数多：
A: 因dubbo协议采用单一长连接，假设网络为千兆网卡(1024Mbit=128MByte)，根据测试经验数据每条连接最多只能压满7MByte(不同的环境可能不一样，供参考)，理论上1个服务提供者需要20个服务消费者才能压满网卡。

Q: 为什么不能传大包：
A: 因dubbo协议采用单一长连接，
如果每次请求的数据包大小为500KByte，假设网络为千兆网卡(1024Mbit=128MByte)，每条连接最大7MByte(不同的环境可能不一样，供参考)，单个服务提供者的TPS(每秒处理事务数)最大为：128MByte / 500KByte = 262。单个消费者调用单个服务提供者的TPS(每秒处理事务数)最大为：7MByte / 500KByte = 14。如果能接受，可以考虑使用，否则网络将成为瓶颈。

Q: 为什么采用异步单一长连接：
A: 因为服务的现状大都是服务提供者少，通常只有几台机器，而服务的消费者多，可能整个网站都在访问该服务，比如Morgan的提供者只有6台提供者，却有上百台消费者，每天有1.5亿次调用，如果采用常规的hessian服务，服务提供者很容易就被压跨，通过单一连接，保证单一消费者不会压死提供者，长连接，减少连接握手验证等，并使用异步IO，复用线程池，防止C10K问题。

(1) 约束：
    - 参数及返回值需实现Serializable接口
    - 参数及返回值不能自定义实现List, Map, Number, Date, Calendar等接口，只能用JDK自带的实现，因为hessian会做特殊处理，自定义实现类中的属性值都会丢失。()
    - Hessian序列化，只传成员属性值和值的类型，不传方法或静态变量，兼容情况：(由吴亚军提供)
    数据通讯	情况	结果
    A->B	类A多一种 属性（或者说类B少一种 属性）	不抛异常，A多的那 个属性的值，B没有， 其他正常
    A->B	枚举A多一种 枚举（或者说B少一种 枚举），A使用多 出来的枚举进行传输	抛异常
    A->B	枚举A多一种 枚举（或者说B少一种 枚举），A不使用 多出来的枚举进行传输	不抛异常，B正常接 收数据
    A->B	A和B的属性 名相同，但类型不相同	抛异常
    A->B	serialId 不相同	正常传输
    总结：会抛异常的情况：枚 举值一边多一种，一边少一种，正好使用了差别的那种，或者属性名相同，类型不同
接口增加方法，对客户端无影响，如果该方法不是客户端需要的，客户端不需要重新部署；
输入参数和结果集中增加属性，对客户端无影响，如果客户端并不需要新属性，不用重新
部署；
输入参数和结果集属性名变化，对客户端序列化无影响，但是如果客户端不重新部署，不管输入还是输出，属性名变化的属性值是获取不到的。
总结：服务器端和客户端对领域对象并不需要完全一致，而是按照最大匹配原则。

(2) 配置：
    dubbo.properties：
    dubbo.service.protocol=dubbo

## hessian v1.2 readme ##
---

/* http://hessian.caucho.com/doc/hessian-1.0-spec.xtp#Metadatanon-normative */

### Design Goals ###
---

它自身是一个轻量二进制协议，用于替换XML-based web服务协议。虽然使用了二进制，Hessian具有自描述和跨语言特性。为web服务的连接协议应该对应用程序写者来说不可见，不需要额外的shema或者IDL去描述它。基于EJB环境，Hessian协议有以下特性：
    * 必须支持XML作为第一个类对象
    * 不需要额外的schema or IDL，对写者不可见
    * 能够序列化Java对象
    * 支持EJB
    * 支持非Java客户端使用web服务
    * 支持以Servlet形式部署web服务
    * 足够简单，方便测试
    * 序列化速度足够快
    * 支持事务

### Serialization ###
---

Hessian有以下九种基本类型：
    * boolean
    * 32-bit int
    * 64-bit long
    * 64-bit double
    * 64-bit date
    * UTF8-encoded string
    * UTF8-encoded xml
    * raw binary data
    * remote objects

Hessian有两种复合类型:
    * list for lists and arrays
    * map for objects and hash tables.

Hessian有两种特殊的结构体：
    * null for null values
    * ref for shared and circular object references.

#### null ####
---
Null代表一个空指针。一个字节的'N'也可代表空指针。任何string, xml, binary, list, map, remote类型变量都可以赋值为NULL。

null ::= N

#### BOOLEAN ####
---
一个字节的'F'代表false，一个字节的'T'代表true。

boolean ::= T
        ::= F

#### INT ####
---

INT是32-bit有符号整数。一个数字后缀是'I'，代表其是4字节大端序整数。

int ::= I b32 b24 b16 b8
integer 300
I x00 x00 x01 x2c

#### LONG ####
---

long是64-bit有符号整数。一个数字后缀是'L'，代表其是8字节大端序整数。

long ::= L b64 b56 b48 b40 b32 b24 b16 b8
long 300
L x00 x00 x00 x00 x00 x00 x01 x2c

#### DOUBLE ####
---

长度是64-bit，且规格复合IEEE浮点数定义。

double ::= D b64 b56 b48 b40 b32 b24 b16 b8
double 12.25
D x40 x28 x80 x00 x00 x00 x00 x00

#### DATE ####
---

Date是64-bit长度整数，其值为从公元纪元以来的毫秒数。

date ::= d b64 b56 b48 b40 b32 b24 b16 b8
2:51:31 May 8, 1998
d x00 x00 x00 xd0 x4b x92 x84 xb8

#### STRING ####
---

字符串以UTF-8编码，其最前面是16-bit unicode字符(代表长度)。其以chunk形式组织起来，'s'代表第一个chunk，'S'代表最后一个chunk，chunk开始处有16-bit长度值，这个值是字符的个数，可能与总字节长度不等。

string ::= (s b16 b8 utf-8-data)* S b16 b8 utf-8-data
"Hello" string
S x00 x05 hello

#### XML ####
---

每个XML文档以UTF-8编码，其最前面是16-bit unicode字符(代表长度)。其以chunk形式组织起来，'x'代表第一个chunk，'X'代表最后一个chunk，chunk开始处有16-bit长度值，这个值是字符的个数，可能与总字节长度不等。

xml ::= (x b16 b8 utf-8-data)* X b16 b8 utf-8-data
trivial XML document
X x00 x10 <top>hello</top>

注意：本文没有定义language mapping, 具体实现的时候读到一个xml对象时字符串解释由用户自己定义。

#### BINARY ####
---

Binary是二进制值。其以chunk形式组织起来，'b'代表第一个chunk，'B'代表最后一个chunk，chunk开始处有16-bit长度值。

binary ::= (b b16 b8 binary-data)* B b16 b8 binary-data

#### LIST ####
---

List是类似于array的有序列表。List由类型字符串(type string),长度值以及list中的对象列表和一个结束字符'z'。类型字符串(type string)可以是由服务定义的任意UTF-8字符串(通常是一个Java class name，但也可以做其他解释)。长度值可以是-1，表示list长度可变。

list ::= V type? length? object* z
Each list item is added to the reference list to handle shared and circular elements. See the ref element.

任何list都可以是空Any parser expecting a list must also accept a null or a shared ref.

Java数组 int[] = {0, 1}序列化后结果：
V t x00 x04 [int
  l x00 x00 x00 x02
  I x00 x00 x00 x00
  I x00 x00 x00 x01
  z

不指定长度不指定类型list = {0, "foobar"}序列化后结果：
V I x00 x00 x00 x00
  S x00 x06 foobar
  z

注意：The valid values of type are not specified in this document and may depend on the specific application. For example, a Java EJB server which exposes an Hessian interface can use the type information to instantiate the specific array type. On the other hand, a Perl server would likely ignore the contents of type entirely and create a generic array.
map

#### MAP ####
---

代表一个对象或Maps，type字段指定map具体指代的对象。如果Object用map表示，则field name表示key，其值就是value，type则是object class name。

map ::= M t b16 b8 type-string (object, object)* z
type可以为空，例如长度为零。当type为空的时候，parser自己选择一个type，对于对象类型，不能识别的key则不解析。

每个map都会被添加进引用链表，parser要支持null map或者null ref。

一个Java对象序列化为一个Map：
public class Car implements Serializable {
  String model = "Beetle";
  String color = "aquamarine";
  int mileage = 65536;
}
M t x00 x13 com.caucho.test.Car
  S x00 x05 model
  S x00 x06 Beetle

  S x00 x05 color
  S x00 x0a aquamarine

  S x00 x07 mileage
  I x00 x01 x00 x00
  z

A sparse array

map = new HashMap();
map.put(new Integer(1), "fee");
map.put(new Integer(16), "fie");
map.put(new Integer(256), "foe");
M I x00 x00 x00 x01
  S x00 x03 fee

  I x00 x00 x00 x10
  S x00 x03 fie

  I x00 x00 x01 x00
  S x00 x03 foe

  z

注意：The type is chosen by the service. Often it may be the Java classname describing the service.

#### REF ####
---

一个整数，用以指代前面的list 或者 map。输入流中每个对象有序，第一个map & list被赋值0，下一个是1，依次赋值之，这个数字就可以认为是这个对象的ID，后面引用这个对象时候使用其ID即可。写者无需理解ID具体指代的对象，但是parser则必须理解。

ref ::= R b32 b24 b16 b8

ref可以使用一个还未读取完整的对象。例如一个环形链表虽然没有被完全读完，但是可以引用其第一个element。

parser解析list or map的时候可以把他们存入array，引用解析时数字就是index。为了支持环形结构，可以每解析完一个element就立即放入array。

circular list
list = new LinkedList();
list.head = 1;
list.tail = list;
M t x00 x0a LinkedList
  S x00 x04 head
  I x00 x00 x00 x01
  S x00 x04 tail
  R x00 x00 x00 x00
  z

注意：
ref只能指代一个list 和 map元素，string 和 binary对象只能以list or map形式被引用。

#### REMOTE ####
---

描述非本地的远程对象，由一个type字段和utf-8编码的URL字符串构成。

remote ::= r t b16 b8 type-name S b16 b8 url
EJB Session Reference
r t x00 x0c test.TestObj
  S x00 x24 http://slytherin/ejbhome?id=69Xm8-zW

### CALL ###
---

Hessian协议的调用由method以及一个参数链表描述的对象构成。对象由容器具体定义，例如一个HTTP请求，它代表一个HTTP URL，参数也采用Hessian协议序列化之。

call ::= c x01 x00 header* m b16 b8 method-string (object)* z
obj.add2(2,3) call
c x01 x00
  m x00 x04 add2
  I x00 x00 x00 x02
  I x00 x00 x00 x03
  z
obj.add2(2,3) reply
r x01 x00
  I x00 x00 x00 x05
  z

#### OBJECT NAMING(NON-NORMATIVE) ####
---

URL既可以用来标识一个对象，也可以用来标识服务地址，URL能否唯一地标识一个Hessian对象，所以不需要额外的参数和方法就可以可用来描述面向对象的服务，如 naming services, entity beans, or session beans。

有人用URL给对象命名时常依照query传统给URL带上"?id=XXX"以说明服务的对象，hessian中依旧推荐这个传统但并非必须。

例如一个股票报价服务接口为http://foo.com/stock，一个股票对象则用http://foo.com?id=PEET。

例如一个EJB使用下面的URL：
http://hostname/hessian/ejb-name?id=object-id
http://hostname/hessian代表一个EJB container，而在Resin-EJB中它代表EJB Servlet。"/hessian"是(url-pattern)前缀，HTTP在此是非必须的。

/ejb-name是请求path信息，代表EJB名称，指定home interface。EJB container中可以有众多entity和session beans。object-id代表特定对象，Home interfaces没有像";ejbid=..."这样的组成部分。

Entity Home Identifier
http://localhost/hessian/my-entity-bean

Entity Bean Identifier
http://localhost/hessian/my-entity-bean?ejbid=slytherin

Session Home Identifier
http://localhost/hessian/my-session-bean

Session Bean Identifier
http://localhost/hessian/my-session-bean?ejbid=M9Zs1Zm

#### METHODS AND OVERLOADING ####
---

Method名称须是唯一。method须支持两种形式的重载：参数个数不同或参数类型不同。编码的时候通过把参数类型放入method名称中来区别重载的method，参数列表参数的类型就不必再放进去了。Method名称以_hessian_字符串开头。

Servers要既能支持mangled method name也能支持unmangled method name。客户端发送的方法名称则须是mangled method name。

java函数名称编码例子如下：
add(int a, int b)
add_int_int
add(double a, double b)
add_double_double
add(shopping.Cart cart, shopping.Item item)
add_shopping.Cart_shopping.Item

#### ARGUMENTS ####
---

参数紧跟在method后面，以Hessian形式编码，参数可以是引用形式。

remote.eq(bean, bean)
bean = new qa.Bean("foo", 13);

System.out.println(remote.eq(bean, bean));
c x01 x00
  m x00 x02 eq
  M t x00 x07 qa.Bean
    S x00 x03 foo
    I x00 x00 x00 x0d
    z
  R x00 x00 x00 x00
  z

参数的个数以及类型由远端方法确定，禁止客户端使用变参方法。

#### HEADERS ####
---

Headers以(string, object)形式存在，在参数列表前面。Header的值可以是任意形式的序列化对象，如一个请求可以把事物的context放入header中。

Call with Distributed Transaction Context
c x01 x00
  H x00 x0b transaction
  r t x00 x28 com.caucho.hessian.xa.TransactionManager
    S x00 x23 http://hostname/xa?ejbid=01b8e19a77
  m x00 x05 debug
  I x00 x03 x01 xcb
  z

#### VERSIONING ####
---

请求和相应tag中都应该包含版本信息，其由major和minor两部分构成，当前version值为1.0。

### REPLY ###
---

正常响应
valid-reply ::= r x01 x00 header* object z
带有错误信息的响应
fault-reply ::= r x01 x00 header* fault z

#### VALUE ####
---

如果处理请求成功，其reply应该回复一个成功值以及一些头部信息。

integer 5 result
r x01 x00
  I x00 x00 x00 x05
  z

#### FAULTS ####
---

错误的请求应该回复错误信息。

每个fault可以由一连串的信息field组织起来，以Map形式表述，而field又包含code, message, and detail。code相当于error code，message则是人可读信息，detail则是机器可识别的exception对象。Java的exception是可序列化的。

远程调用返回FileNotFoundException：

r x01 x00
  f
  S x00 x04 code
  S x00 x10 ServiceException

  S x00 x07 message
  S x00 x0e File Not Found

  S x00 x06 detail
  M t x00 x1d java.io.FileNotFoundException
    z
  z

Hessian预定义好的exception：

ProtocolException	The Hessian request has some sort of syntactic error.
NoSuchObjectException	The requested object does not exist.
NoSuchMethodException	The requested method does not exist.
RequireHeaderException	A required header was not understood by the server.
ServiceException	The called method threw an exception.

#### METADATA(NON-NORMATIVE) ####
---

一些以_hessian_开头的method可能需要metadata。

_hessian_getAttribute(String key)返回一个字符串，下面是由本标准定义的一些metadata：

ATTRIBUTE	MEANING
java.api.class	Java interface for this URL
java.home.class	Java interface for this service
java.object.class	Java interface for a service object
java.ejb.primary.key.class	Java EJB primary key class
"java.api.class" returns the client proxy's Java API class for the current URL. "java.home.class" returns the API class for the factory URL, i.e. without any "?id=XXX" query string. "java.object.class" returns the API class for object instances.

In the case of services with no object instances, i.e. non-factory services, all three attributes will return the same class name.

过时的metadata：
ATTRIBUTE	MEANING
home-class	Java class for the home interface.
remote-class	Java class for the object interface.
primary-key-class	Java class for the primary key.

#### MICRO HESSIAN ####
---

"Micro Hessian"实现不支持double浮点数。

#### FORMAL DEFINITIONS ####
---

top     ::= call
        ::= replycall    ::= c x01 x00 header* methodobject* z

reply   ::= r x01 x00 header* object z
        ::= r x01 x00 header* fault z

object  ::= null
        ::= boolean
        ::= int
        ::= long
        ::= double
        ::= date
        ::= string
        ::= xml
        ::= binary
        ::= remote
        ::= ref
        ::= list
        ::= mapheader  ::= H b16 b8 header-string objectmethod  ::= m b16 b8 method-string

fault   ::= f (objectobject)* z

list    ::= V type? length? object* z
map     ::= M type? (objectobject)* z
remote  ::= r type? stringtype    ::= t b16 b8 type-string
length  ::= l b32 b24 b16 b8

null    ::= N
boolean ::= T
        ::= F
int     ::= I b32 b24 b16 b8
long    ::= L b64 b56 b48 b40 b32 b24 b16 b8
double  ::= D b64 b56 b48 b40 b32 b24 b16 b8
date    ::= d b64 b56 b48 b40 b32 b24 b16 b8
string  ::= (s b16 b8 string-data)* S b16 b8 string-data
xml     ::= (x b16 b8 xml-data)* X b16 b8 xml-data
binary  ::= (b b16 b8 binary-data)* B b16 b8 binary-data
ref     ::= R b32 b24 b16 b8

## hessian v2 readme ##
---

/* http://hessian.caucho.com/doc/hessian-serialization.html */

3.  Hessian Grammar


RPC/Messaging Grammar

top       ::= version content
          ::= call-1.0
          ::= reply-1.0

          # RPC-style call
call      ::= 'C' string int value*

call-1.0  ::= 'c' x01 x00 <hessian-1.0-call>

content   ::= call       # rpc call
          ::= fault      # rpc fault reply
          ::= reply      # rpc value reply
          ::= packet+    # streaming packet data
          ::= envelope+  # envelope wrapping content

envelope  ::= 'E' string env-chunk* 'Z'
env-chunk ::= int (string value)* packet int (string value)*

          # RPC fault
fault     ::= 'F' (value value)* 'Z'

          # message/streaming message
packet    ::= (x4f b1 b0 <data>)* packet
          ::= 'P' b1 b0 <data>
          ::= [x70 - x7f] <data>
          ::= [x80 - xff] <data>

          # RPC reply
reply     ::= 'R' value

reply-1.0 ::= 'r' x01 x00 <hessian-1.0-reply>

version   ::= 'H' x02 x00

4.  Messages and Envelopes

Hessian message syntax organizes serialized data for messaging and RPC applications. The envelope syntax enables compression, encryption, signatures, and any routing or context headers to wrap a Hessian message.

Call ('C'): contains a Hessian RPC call, with a method name and arguments.
Envelope ('E'): wraps a Hessian message for compression, encryption, etc. Envelopes can be nested.
Hessian ('H'): introduces a Hessian stream and indicates its version.
Packet ('P'): contains a sequence of serialized Hessian objects.
Reply ('R'): contains a reply to a Hessian RPC call.
Fault ('F'): contains a reply to a failed Hessian RPC call.
Hessian 1.0 compatibility call ('c'): is a Hessian 1.0 call.
Hessian 1.0 compatibility reply ('cr): is a Hessian 2.0 call.

4.1.  Call


Call Grammar

call ::= C string int value*
 Figure 2
A Hessian call invokes a method on an object with an argument list. The object is specified by the container, e.g. for a HTTP request, it's the HTTP URL. The arguments are specified by Hessian serialization.


 TOC
4.1.1.  Methods and Overloading

Method names must be unique. Two styles of overloading are supported: overloading by number of argumetns and overloading by argument types. Overloading is permitted by encoding the argument types in the method names. The types of the actual arguments must not be used to select the methods.

Method names beginning with _hessian_ are reserved.

Servers should accept calls with either the mangled method name or the unmangled method name. Clients should send the mangled method name.


 TOC
4.1.1.1.  Overloading examples


 add(int a, int b)  ->  add_int_int

add(double a, double b)  ->  add_double_double

add(shopping.Cart cart, shopping.Item item)  ->
  add_shopping.Cart_shopping.Item
 Figure 3

 TOC
4.1.2.  Arguments

Arguments immediately follow the method in positional order. Argument values use Hessian's serialization.

All arguments share references, i.e. the reference list starts with the first argument and continues for all other arguments. This lets two arguments share values.


 TOC
4.1.2.1.  Arguments example


 bean = new qa.Bean("foo", 13);

System.out.println(remote.eq(bean, bean));

---
H x02 x00
C
  x02 eq        # method name = "eq"
  x92           # two arguments
  M x07 qa.Bean # first argument
    x03 foo
    x9d
    Z
  Q x00         # second argument (ref to first)
 Figure 4
The number and type of arguments are fixed by the remote method. Variable length arguments are forbidden. Implementations may take advantage of the expected type to improve performance.


 TOC
4.1.3.  Call examples


obj.add2(2,3) call

H x02 x00         # Hessian 2.0
C                 # RPC call
  x04 add2        # method "add2"
  x92             # two arguments
  x92             # 2 - argument 1
  x93             # 3 - argument 2
 Figure 5

obj.add2(2,3) reply

H x02 x00        # Hessian 2.0
R                # reply
  x95            # int 5
 Figure 6

 TOC
4.2.  Envelope


Envelope Grammar

envelope ::= E string env-chunk* Z

env-chunk ::= int (string value)* packet int (string value)*
 Figure 7
A Hessian envelope wraps a Hessian message, adding headers and footers and possibly compressing or encrypting the wrapped message. The envelope type is identified by a method string, e.g. "com.caucho.hessian.io.Deflation" or "com.caucho.hessian.security.X509Encryption".

Some envelopes may chunk the data, providing multiple header/footer chunks. For example, a signature envelope might chunk a large streaming message to reduce the amount of buffering required to validate the signatures.


 TOC
4.2.1.  Envelope examples


Identity Envelope

H x02 x00              # Hessian 2.0
E                      # envelope
  x06 Header           # "Header" adds headers, body is identity
  x90                  # no headers
  x87                  # final packet (7 bytes)
    R                  # RPC reply
    x05 hello          # "hello"
  x90                  # no footers
  Z                    # end of envelope
 Figure 8

Chunked Identity Envelope

H x02 x00              # Hessian 2.0
E
  x06 Header           # "Header" envelope does nothing to the body
  x90                  # no headers
  x87                  # final packet (7 bytes)
    C                  # RPC call
    x05 hello          # hello()
    x91                # one arg
  x90                  # no footers

  x90                  # no headers
  x8d                  # final packet (13 bytes)
    x05 hello, world   # hello, world
  x90                  # no footers
  Z                    # end of envelope
 Figure 9

Compression Envelope

H x02 x00
E
  x09 Deflation        # "Deflation" envelope compresses the body
  x90                  # no headers
  P x10 x00            # single chunk (4096)
    x78 x9c x4b...     # compressed message
  x90                  # no footers
  Z                    # end of envelope
 Figure 10

 TOC
4.3.  packet


packet Grammar

packet   ::= (x4f b1 b0 <data>) packet
         ::= 'P' b1 b0 <data>
         ::= [x70-x7f] b0 <data>
         ::= [x80-xff] <data>
 Figure 11
A Hessian message contains a sequence of Hessian serialized objects. Messages can be used for multihop data transfer or simply for storing serialized data.


 TOC
4.4.  Reply


Reply Grammar

reply       ::= R value
fault       ::= F map
 Figure 12

 TOC
4.4.1.  Value

A successful reply returns a single value and possibly some header information.


Integer 5 Envelope

H x02 x00
R
  x95
 Figure 13

 TOC
4.4.2.  Faults

Failed calls return a fault.

Each fault has a number of informative fields, expressed like <map> entries. The defined fields are code, message, and detail. code is one of a short list of strings defined below. message is a user-readable message. detail is an object representing the exception.


Remote Call throws FileNotFoundException

F
  H
  x04 code
  x10 ServiceException

  x07 message
  x0e File Not Found

  x06 detail
  M x1d java.io.FileNotFoundException
    Z
  Z
 Figure 14
ProtocolException:
The Hessian request has some sort of syntactic error.
NoSuchObjectException:
The requested object does not exist.
NoSuchMethodException:
The requested method does not exist.
RequireHeaderException:
A required header was not understood by the server.
ServiceException:
The called method threw an exception.

 TOC
4.5.  Versioning

The call and response tags include a major and minor byte. The current version is 2.0.


 TOC
5.  Service Location (URLs)

Hessian services are identified by URLs. Typically, these will be HTTP URLs, although protocols would be possible as well.


 TOC
5.1.  Object Naming (non-normative)

URLs are flexible enough to encode object instances as well as simple static service locations. The URL uniquely identifies the Hessian object. Thus, Hessian can support object-oriented services, e.g. naming services, entity beans, or session beans, specified by the URL without requiring extra method parameters or headers.

Object naming may use the query string convention that "?id=XXX" names the object "XXX" in the given service. This convention is recommented, but not required.

For example, a stock quote service might have a factory interface like http://foo.com/stock and object instances like http://foo.com?id=PEET. The factory interface would return valid object references through the factory methods.


 TOC
5.2.  Object naming (non-normative) Example

As an example, the following format is used for EJB:


 http://hostname/hessian
http://hostname/hessian
http://hostname/hessian
 Figure 15
http://hostname/hessian identifies the EJB container. In Resin-EJB, this will refer to the EJB Servlet. "/hessian" is the servlet prefix (url-pattern.) HTTP is just used as an example; Hessian does not require the use of HTTP.

/ejb-name, the path info of the request, identifies the EJB name, specifically the home interface. EJB containers can contain several entity and session beans, each with its own EJB home. The ejb-name corresponds to the ejb-name in the deployment descriptor.

object-id identifies the specific object. For entity beans, the object-id encodes the primary key. For session beans, the object-id encodes a unique session identifier. Home interfaces have no ";ejbid=..." portion.


 # Example Entity Home Identifier
http://localhost/hessian/my-entity-bean

# Example Entity Bean Identifier
http://localhost/hessian/my-entity-bean?ejbid=slytherin

# Example Session Home Identifier
http://localhost/hessian/my-session-bean

# Example Session Bean Identifier
http://localhost/hessian/my-session-bean?ejbid=M9Zs1Zm
 Figure 16

 TOC
6.  Bytecode map

Hessian is organized as a bytecode protocol. A Hessian reader is essentially a switch statement on the initial octet.

Bytecode Encoding

x00 - x42    # reserved
x43          # rpc call ('C')
x44          # reserved
x45          # envelope ('E')
x46          # fault ('F')
x47          # reserved
x48          # hessian version ('H')
x49 - x4f    # reserved
x4f          # packet chunk ('O')
x50          # packet end ('P')
x51          # reserved
x52          # rpc result ('R')
x53 - x59    # reserved
x5a          # terminator ('Z')
x5b - x5f    # reserved
x70 - x7f    # final packet (0 - 4096)
x80 - xff    # final packet for envelope (0 - 127)

## micro services ##
- 1 http://duanple.blog.163.com/blog/static/70971767201329113141336/
- 2 http://tonybai.com/2015/06/17/appdash-distributed-systems-tracing-in-go/
- 3 http://www.zenlife.tk/distributed-tracing.md

