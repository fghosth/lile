
Lile是可以帮助您快速创建基于gRPC通讯，或者可以通过[gateway](https://github.com/grpc-ecosystem/grpc-gateway)创建通讯结构

Lile主要是用于过创建基本结构，测试示例，Dockerfile，Makefile等基础骨架。

Lile也是一个简单的服务生成器，扩展了基本的gRPC服务器，包括诸如指标（如[Prometheus](prometheus.io)）。

### 功能如下：
* 根据proto文件自动生成服务端代码及单元测试1
* tls模式下同一端口https，grpc可同时访问
* 如果有定义grpc-gateway相关，可生成swagger文档
* 链路追踪




lile 改编自https://github.com/lileio/lile

改进如下：
1. 去除google的消息订阅，这个国内没法用
2. 增加tls访问，可以同一端口grpc，https自动协议判断。
3. 增加swagger文档生成
4. 增加jaeger链路追踪 （待实现）
5. proto插件bug修正

### 安装

安装Lile很容易，使用`go get`便可以安装Lile的命令行工具来生成新的服务和所需的库。

```
$ go get -u github.com/fghosth/lile/...
```

您还需要安装Google的[Protocol Buffers](https://developers.google.com/protocol-buffers/)。

### 入门

Lilek可以自动根据`username/service`生成一个完整的路径。

```
$ lile new --name users
```

# 指南

- [安装](#安装)
- [创建服务](#创建服务)
- [服务定义](#服务定义)
- [生成RPC方法](#生成RPC方法)
- [编写并运行测试](#编写并运行测试)
- [使用生成的命令行](#使用生成的命令行)
- [自定义命令行](#自定义命令行)
- [暴露Prometheus指标](#暴露Prometheus采集指标)
- [追踪](#追踪) 待实现

## 安装
* mac
```
brew install fghosth/lile
```
> 安装lile和protoc-gen-lile-server 插件

首先，你需要确保您在您安装Lile之前已经安装了Go。

安装Lile很容易，使用`go get`便可以安装Lile的命令行工具来生成新的服务和所需的库。

```
$ go get github.com/fghosth/lile/...
cd lile
go build
cp lile /usr/local/bin
```

您还需要安装Google的[Protocol Buffers][Protocol Buffers](https://developers.google.com/protocol-buffers/)。

在MacOS你可以使用`brew install protobuf`来安装。

## 安装 protoc-gen-lile-server插件
```bash
 cd protoc-gen-lile-server
 go build
 cp protoc-gen-lile-server /usr/local/bin
```
### 初始化项目
```bash
make init
```
## 生成pb文件
```bash
make proto
```
## 创建服务

Lile使用生成器来快速生成新的Lile服务。

Lile遵循Go关于$GOPATH的约定（参见[如何写Go](https://golang.org/doc/code.html#Workspaces)），并且自动解析您的新服务的名称，以在正确的位置创建服务。

如果您的Github用户名是lileio，并且您想创建一个新的服务为了发布消息到Slack，您可以使用如下命令：

```
lile new --name slack
```

这将创建一个项目到`$GOPATH/src/github.com/lileio/slack`

## 服务定义

Lile服务主要使用gRPC，因此使用[protocol buffers](https://developers.google.com/protocol-buffers/)作为接口定义语言（IDL），用于描述有效负载消息的服务接口和结构。 如果需要，可以使用其他替代品。

我强烈建议您先阅读[Google API设计](https://cloud.google.com/apis/design/)文档，以获得有关RPC方法和消息的一般命名的好建议，以及如果需要，可以将其转换为REST/JSON。

您可以在Lile中发现一个简单的例子[`account_service`](https://github.com/fghosth/account_service)

``` protobuf
service AccountService {
  rpc List (ListAccountsRequest) returns (ListAccountsResponse) {}
  rpc GetById (GetByIdRequest) returns (Account) {}
  rpc GetByEmail (GetByEmailRequest) returns (Account) {}
  rpc AuthenticateByEmail (AuthenticateByEmailRequest) returns (Account) {}
  rpc GeneratePasswordToken (GeneratePasswordTokenRequest) returns (GeneratePasswordTokenResponse) {}
  rpc ResetPassword (ResetPasswordRequest) returns (Account) {}
  rpc ConfirmAccount (ConfirmAccountRequest) returns (Account) {}
  rpc Create (CreateAccountRequest) returns (Account) {}
  rpc Update (UpdateAccountRequest) returns (Account) {}
  rpc Delete (DeleteAccountRequest) returns (google.protobuf.Empty) {}
}
```

## 生成RPC方法

默认情况下，Lile将创建一个RPC方法和一个简单的请求和响应消息。

``` protobuf
syntax = "proto3";
option go_package = "github.com/fghosth/slack";
package slack;

message Request {
  string id = 1;
}

message Response {
  string id = 1;
}

service Slack {
  rpc Read (Request) returns (Response) {}
}
```

我们来修改一下使它能够提供真正的服务，并添加自己的方法。

我们来创建一个`Announce`方法向Slack发布消息。

我们假设Slack团队和身份验证已经由服务配置来处理，所以我们服务的用户只需要提供一个房间和他们的消息。 该服务将发送特殊的空响应，因为我们只需要知道是否发生错误，也不需要知道其他任何内容。

现在我们的`proto`文件看起来像这样：

``` protobuf
syntax = "proto3";
option go_package = "github.com/fghosth/slack";
import "google/protobuf/empty.proto";
package slack;

message AnnounceRequest {
  string channel = 1;
  string msg = 2;
}

service Slack {
  rpc Announce (AnnounceRequest) returns (google.protobuf.Empty) {}
}
```

现在我们运行`protoc`工具我们的文件，以及Lile生成器插件。

```
protoc -I . slack.proto --lile-server_out=. --go_out=plugins=grpc:$GOPATH/src
```

Lile提供了一个`Makefile`，每个项目都有一个已经配置的`proto`构建步骤。 所以我们可以运行它。

```
make proto
```

我们可以看到，Lile将在`server`目录中为我们创建两个文件。

```
$ make proto
protoc -I . slack.proto --lile-server_out=. --go_out=plugins=grpc:$GOPATH/src
2017/07/12 15:44:01 [Creating] server/announce.go
2017/07/12 15:44:01 [Creating test] server/announce_test.go
```

我们来看看Lile为我们创建的`announce.go`文件。

``` go
package server

import (
    "errors"

    "github.com/golang/protobuf/ptypes/empty"
    "github.com/lileio/slack"
    context "golang.org/x/net/context"
)

func (s SlackServer) Announce(ctx context.Context, r *slack.AnnounceRequest) (*empty.Empty, error) {
  return nil, errors.New("not yet implemented")
}
```

接下来我们实现这个生成的方法，让我们从测试开始吧！


## 编写并运行测试

当您使用Lile生成RPC方法时，也会创建一个对应的测试文件。例如，给定我们的`announce.go`文件，Lile将在同一目录中创建`announce_test.go`

看起来如下:

``` go
package server

import (
	"testing"

	"github.com/fghosth/slack"
	"github.com/stretchr/testify/assert"
	context "golang.org/x/net/context"
)

func TestAnnounce(t *testing.T) {
	ctx := context.Background()
	req := &slack.AnnounceRequest{}

	res, err := cli.Announce(ctx, req)
	assert.Nil(t, err)
	assert.NotNil(t, res)
}

```

您现在可以使用`Makefile`运行测试，并运行`make test`命令

```
$ make test
=== RUN   TestAnnounce
--- FAIL: TestAnnounce (0.00s)
        Error Trace:    announce_test.go:16
        Error:          Expected nil, but got: &status.statusError{Code:2, Message:"not yet implemented", Details:[]*any.Any(nil)}
        Error Trace:    announce_test.go:17
        Error:          Expected value not to be nil.
FAIL
coverage: 100.0% of statements
FAIL    github.com/lileio/slack/server  0.011s
make: *** [test] Error 2

```

我们的测试失败了，因为我们还没有实现我们的方法，在我们的方法中返回一个“未实现”的错误。

让我们在`announce.go`中实现`Announce`方法，这里是一个使用`nlopes`的[slack library](https://github.com/nlopes/slack)的例子。

``` go
package server

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/golang/protobuf/ptypes/empty"
	"github.com/fghosth/slack"
	sl "github.com/nlopes/slack"
	context "golang.org/x/net/context"
)

var api = sl.New(os.Getenv("SLACK_TOKEN"))

func (s SlackServer) Announce(ctx context.Context, r *slack.AnnounceRequest) (*empty.Empty, error) {
	_, _, err := api.PostMessage(r.Channel, r.Msg, sl.PostMessageParameters{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, err.Error())
	}
   
	return &empty.Empty{}, nil
}
```

我们再次修改我们的测试用力，然后再次运行我们的测试

``` go
package server

import (
	"testing"

	"github.com/fghosth/slack"
	"github.com/stretchr/testify/assert"
	context "golang.org/x/net/context"
)

func TestAnnounce(t *testing.T) {
	ctx := context.Background()
	req := &slack.AnnounceRequest{
		Channel: "@alex",
		Msg:     "hellooo",
	}

	res, err := cli.Announce(ctx, req)
	assert.Nil(t, err)
	assert.NotNil(t, res)
}
```
现在如果我使用我的Slack令牌作为环境变量运行测试，我应该看到通过测试！

```
$ alex@slack: SLACK_TOKEN=zbxkkausdkasugdk make test
go test -v ./... -cover
?       github.com/lileio/slack [no test files]
=== RUN   TestAnnounce
--- PASS: TestAnnounce (0.32s)
PASS
coverage: 75.0% of statements
ok      github.com/lileio/slack/server  0.331s  coverage: 75.0% of statements
?       github.com/lileio/slack/slack   [no test files]
?       github.com/lileio/slack/slack/cmd       [no test files]
?       github.com/lileio/slack/subscribers     [no test files]
```

## 使用生成的命令行

生成您的服务时，Lile生成一个命令行应用程序。 您可以使用自己的命令行扩展应用程序或使用内置的命令行来运行服务。

运行没有任何参数的命令行应用程序将打印生成的帮助。

例如`go run orders/main.go`

### 服务

运行`serve`将运行RPC服务。


### up

运行`up`将同时运行RPC服务器和发布订阅的订阅者。

## 自定义命令行

要添加您自己的命令行，您可以使用[cobra](https://github.com/spf13/cobra)，它是Lile的内置的命令行生成器。

``` bash
$ cd orders
$ cobra add import
```

您现在可以编辑生成的文件，以创建您的命令行，`cobra`会自动将命令行的名称添加到帮助中。 

## 暴露Prometheus采集指标

默认情况下，Lile将[Prometheus](prometheus.io)的采集指标暴露在`:9000/metrics`。

如果您的服务正在运行，您可以使用cURL来预览Prometheus指标。

```
$ curl :9000/metrics
```

你应该看到如下的一些输出：

```
# HELP go_gc_duration_seconds A summary of the GC invocation durations.
# TYPE go_gc_duration_seconds summary
go_gc_duration_seconds{quantile="0"} 0
go_gc_duration_seconds{quantile="0.25"} 0
go_gc_duration_seconds{quantile="0.5"} 0
go_gc_duration_seconds{quantile="0.75"} 0
go_gc_duration_seconds{quantile="1"} 0
go_gc_duration_seconds_sum 0
go_gc_duration_seconds_count 0
...
...
```

Lile的Prometheus指标实现使用go-grpc-promesheus的拦截器将其内置到自身的gPRC中，提供如以下的指标：

```
grpc_server_started_total
grpc_server_msg_received_total
grpc_server_msg_sent_total
grpc_server_handling_seconds_bucket
```

有关使用Prometheus的更多信息，收集和绘制这些指标，请参阅[Prometheus入门](https://prometheus.io/docs/introduction/getting_started/)

有关gRPC Prometheus查询的示例，请参阅[查询示例](https://github.com/grpc-ecosystem/go-grpc-prometheus#useful-query-examples)。

Protobuf消息自动解码。

## 追踪(未实现)

Lile已经建立了跟踪，将[opentracing](http://opentracing.io/) 兼容的跟踪器设置为`GlobalTracer`。


