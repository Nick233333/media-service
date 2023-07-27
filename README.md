## 媒体服务


## 技术栈

- go-zero
- kafka
- ffmpeg

## 运行项目命令

```go
go run consumer.go -f etc/consumer.yaml

go run media.go -f etc/media-api.yaml
```

goctl 命令

> 注意执行命令所在的目录
```shell
goctl api new api #创建 api 模块

goctl rpc new rpc #创建 rpc 模块

go run media.go -f etc/media-api.yaml #运行项目

goctl api go -api media.api -dir . #生成接口

goctl api -o shorturl.api #指定api文件名称
```

## api 文件语法版本差异
在 go-zero 框架中，不同版本的 API 文件语法存在一些区别，主要涉及语法、语义、约束等方面。以下是一些常见的不同之处：

1. 语法差异：不同版本的 API 文件语法可能存在不同的关键字、格式要求等。例如，在 `v1` 版本中，使用 `service` 来定义服务；而在 `v2` 版本中，使用 `type` 来定义服务。

2. 特性差异：不同版本的 API 文件可能支持不同的特性或功能。例如，`v2` 版本新增了支持 `HTTP PUT` 请求的语法，而 `v1` 版本中则没有。

3. 约束差异：不同版本的 API 文件可能存在一些不同的约束或限制。例如，`v1` 版本中要求请求参数和响应参数都需要定义为 message 类型；而在 `v2` 版本中，可以使用任意的结构体类型来定义参数。

4. 易用性差异：不同版本的 API 文件可能对开发者的易用性存在一些差异。例如，在 `v2` 版本中新增了对多语言的支持，可以更方便地生成多种语言的客户端代码。

总的来说，不同版本的 API 文件语法存在差异主要是为了适应不同的开发需求和发展趋势，开发者可以根据自己的实际情况选择适合自己的版本。在使用不同版本的 API 文件时，需要注意其语法和语义的差异，以便正确地定义和使用 API 接口。

## 中间件使用

不支持针对单个接口添加中间件，只支持路由组
```go

@server(
    prefix: v1
)

service api-api {
    @handler ApiHandler
    post /a(TransformReq) returns (TransformReply)

    @handler ApiHandler
    post /b(TransformReq) returns (TransformReply)
}

@server(
    prefix: v1
    middleware: xxxMiddleware
)

service api-api {
    @handler ApiHandler
    post /xx(TransformReq) returns (TransformReply)
}
```

## Kafka 命令

```
kafka-topics -bootstrap-server localhost:9092 --create --topic nick  --partitions 1 --replication-factor 1 #创建主题命令

kafka-topics -bootstrap-server localhost:9092 --delete --topic nick #删除主题命令

kafka-topics -bootstrap-server localhost:9092 --list #查看所有主题命令

kafka-topics -bootstrap-server localhost:9092 --topic test --describe #查看指定主题详情命令

kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic media-topic --time -1 #查看消息数量

kafka-console-producer --broker-list localhost:9092 --topic media-topic #生成数据

kafka-console-consumer --topic media-topic  --bootstrap-server localhost:9092 #消费者命令

kafka-console-consumer --topic media-topic  --bootstrap-server localhost:9092 --from-beginning  #消费者命令 --from-beginning 表示从最初的未过期的 offset 处开始消费数据。不加该参数，表示从最新 offset 处开始消费数据。

kafka-consumer-groups --bootstrap-server localhost:9092 --list #查看消费组
```

参考项目 https://github.com/zhoushuguang/lebron/blob/main/apps/seckill/rmq/internal/service/service.go
