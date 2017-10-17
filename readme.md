# dubbogo #
a golang micro-service framework compatible with alibaba dubbo. just using jsonrpc 2.0 protocol over http now.

## feature ##
---
- 1 基于TCP or HTTP的分布式的RPC(√)
- 2 支持多种编解码协议，如JsonRPC(√),Hessian(ing),ProtoBuf,Thrift等
- 3 服务发现：服务发布(√)、订阅(√)、通知(√)等，支持多种发现方式如ZooKeeper(√)、Redis(x)等
- 4 高可用策略：失败重试(Failover,√)、快速失败(Failfast,√)
- 5 负载均衡：支持随机请求(√)、轮询(√)、基于权重(x)等
- 6 其他，如调用统计、访问日志、身份验证等(x)

## 说明 ##
---
> dubbogo 目前版本(0.1.1)支持的codec 是jsonrpc 2.0，transport protocol是http。
>
> 只要你的java程序支持jsonrpc 2.0 over http，那么dubbogo程序就能调用它。使用过程中如遇到问题，请先查看doc/question.list.txt.zip。
>
> dubbogo自己的server端也已经实现，即dubbogo既能调用java service也能调用dubbogo实现的service。相应的代码示例请参考[dubbogo-examples](https://github.com/AlexStocks/dubbogo-examples) 。
>
> dubbogo 下一个版本(0.2.x)计划支持的codec 是dubbo(hessian)，transport protocol是tcp。
>
> QQ group: 196953050

