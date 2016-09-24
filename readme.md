# dubbogo #
a golang micro-service framework compatible with alibaba dubbo. just using jsonrpc 2.0 protocol over http now.

## 说明 ##
---
> dubbogo 目前版本(0.1.1)支持的codec 是jsonrpc 2.0，transport protocol是http。
>
> 只要你的java程序支持jsonrpc 2.0 over http，那么dubbogo程序就能调用它。使用过程中如遇到问题，请先查看doc/question.list.txt.zip。
>
> dubbogo自己的server端也已经实现，即dubbogo既能调用java service也能调用dubbogo实现的service。相应的代码示例请参考 https://github.com/AlexStocks/dubbogo-examples。
>
> dubbogo 下一个版本(0.2.x)计划支持的codec 是dubbo(hessian)，transport protocol是tcp。
>
> QQ group: 196953050

