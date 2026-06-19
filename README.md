# go-boot-grpc

[![Go Version](https://img.shields.io/github/go-mod/go-version/xudefa/go-boot-grpc)](https://go.dev/) [![License](https://img.shields.io/github/license/xudefa/go-boot-grpc)](./LICENSE) [![Build Status](https://img.shields.io/github/actions/workflow/status/xudefa/go-boot-grpc/test.yml?branch=master)](https://github.com/xudefa/go-boot-grpc/actions) [![Go Reference](https://pkg.go.dev/badge/github.com/xudefa/go-boot-grpc.svg)](https://pkg.go.dev/github.com/xudefa/go-boot-grpc) [![Go Report Card](https://goreportcard.com/badge/github.com/xudefa/go-boot-grpc)](https://goreportcard.com/report/github.com/xudefa/go-boot-grpc)

基于 [go-boot](https://github.com/xudefa/go-boot) 的 gRPC 框架集成模块。将 gRPC 无缝集成到 go-boot 的 IoC 容器和自动配置体系中,提供声明式的 gRPC 服务器和客户端能力。

> 设计理念:遵循 go-boot 的开发规范,将 gRPC Server 作为 `net.Server` 接口的实现,通过自动配置实现零代码启动 gRPC 服务。

## 整体架构

```
┌───────────────────────────────────────────────────────────────────────┐
│                    go-boot ApplicationContext                         │
│  ┌───────────┐ ┌──────────────┐ ┌───────────┐ ┌───────────┐           │
│  │ Container │ │  Environment │ │ Lifecycle │ │ EventBus  │           │
│  └───────────┘ └──────────────┘ └───────────┘ └───────────┘           │
│                       ┌─────────────────────┐                         │
│                       │ AutoConfig Registry │                         │
│                       └─────────────────────┘                         │
└───────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
                    ┌───────────────────────────────┐
                    │     go-boot-grpc Starter      │
                    │  ┌─────────────────────────┐  │
                    │  │ gRPC Server Bean        │  │
                    │  │ gRPC Client Bean        │  │
                    │  │ Service Registration    │  │
                    │  │ Tracing Interceptor     │  │
                    │  └─────────────────────────┘  │
                    └───────────────────────────────┘
```

## 目录

- [快速开始](#快速开始)
- [功能特性](#功能特性)
- [gRPC 服务器](#grpc-服务器)
- [gRPC 客户端](#grpc-客户端)
- [分布式追踪](#分布式追踪)
- [配置选项](#配置选项)
- [项目结构](#项目结构)
- [开发指南](#开发指南)
- [贡献](#贡献)
- [许可证](#许可证)

## 快速开始

### 安装

```bash
# 安装核心框架
go get github.com/xudefa/go-boot

# 安装 gRPC 集成模块
go get github.com/xudefa/go-boot-grpc
```

### 最小示例

```go
package main

import (
    "context"

    "github.com/xudefa/go-boot/boot"
    "github.com/xudefa/go-boot/core"
    grpcserver "github.com/xudefa/go-boot-grpc/server"
    pb "your/protobuf/package"
)

func main() {
    app, err := boot.NewApplication(
        boot.WithAppName("my-grpc-app"),
        boot.WithVersion("1.0.0"),
        boot.WithProperty("grpc.server.enabled", "true"),
        boot.WithProperty("grpc.server.address", ":50051"),
    )
    if err != nil {
        panic(err)
    }
    defer app.Stop()

    // 注册 gRPC 服务
    server := app.Container().Get("grpcServer").(*grpcserver.Server)
    pb.RegisterGreeterServer(server.GrpcServer(), &greeterServer{})

    // 启动应用（自动启动 gRPC 服务器）
    app.Start()

    // 等待终止信号
    app.WaitForSignal()
}

// gRPC 服务实现
type greeterServer struct {
    pb.UnimplementedGreeterServer
}

func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
    return &pb.HelloResponse{Message: "Hello " + req.Name}, nil
}
```

## 功能特性

| 特性 | 说明 |
|------|------|
| gRPC 集成 | 将 gRPC Server/Client 注册为 Bean,支持依赖注入 |
| net.Server 实现 | gRPC Server 实现 go-boot 的 `net.Server` 接口 |
| 自动配置 | 通过 `grpc.server.enabled=true` 自动启动 gRPC 服务 |
| 优雅启停 | 支持优雅关闭和生命周期管理 |
| 分布式追踪 | 内置 OpenTelemetry 追踪拦截器 |
| 声明式服务 | 支持通过 Proto 文件生成服务并注册 |
| 配置驱动 | 地址、超时、TLS 等均可通过配置控制 |

## gRPC 服务器

### 基本服务器

```go
import grpcserver "github.com/xudefa/go-boot-grpc/server"

// 创建 gRPC 服务器
server := grpcserver.New(
    grpcserver.WithAddress(":50051"),
)

// 注册服务
pb.RegisterGreeterServer(server.GrpcServer(), &greeterServer{})
```

### 带追踪的服务器

```go
import (
    grpcserver "github.com/xudefa/go-boot-grpc/server"
    "github.com/xudefa/go-boot/tracing"
)

tracer := tracing.GetTracer("grpc")
server := grpcserver.New(
    grpcserver.WithAddress(":50051"),
    grpcserver.WithTracing(tracer),
)
```

### 服务实现示例

```go
type UserService struct {
    pb.UnimplementedUserServiceServer
}

func (s *UserService) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.User, error) {
    return &pb.User{
        Id:    req.Id,
        Name:  "John Doe",
        Email: "john@example.com",
    }, nil
}

func (s *UserService) ListUsers(ctx context.Context, req *pb.ListUsersRequest) (*pb.ListUsersResponse, error) {
    return &pb.ListUsersResponse{
        Users: []*pb.User{
            {Id: 1, Name: "Alice", Email: "alice@example.com"},
            {Id: 2, Name: "Bob", Email: "bob@example.com"},
        },
    }, nil
}
```

## gRPC 客户端

### 基本客户端

```go
import grpcclient "github.com/xudefa/go-boot-grpc/client"

// 创建 gRPC 客户端
client := grpcclient.New(
    grpcclient.WithAddress("localhost:50051"),
    grpcclient.WithTimeout(5*time.Second),
)

// 获取连接
conn, err := client.Connection()
if err != nil {
    panic(err)
}

// 创建服务客户端
greeterClient := pb.NewGreeterClient(conn)
resp, err := greeterClient.SayHello(context.Background(), &pb.HelloRequest{
    Name: "World",
})
```

### 带追踪的客户端

```go
import (
    grpcclient "github.com/xudefa/go-boot-grpc/client"
    "github.com/xudefa/go-boot/tracing"
)

tracer := tracing.GetTracer("grpc")
client := grpcclient.New(
    grpcclient.WithAddress("localhost:50051"),
    grpcclient.WithTimeout(5*time.Second),
    grpcclient.WithTracing(tracer),
)
```

## 分布式追踪

go-boot-grpc 内置 OpenTelemetry 追踪支持:

```go
import (
    grpcserver "github.com/xudefa/go-boot-grpc/server"
    grpcclient "github.com/xudefa/go-boot-grpc/client"
    "github.com/xudefa/go-boot/tracing"
)

// 创建 Tracer
tracer := tracing.GetTracer("grpc")

// 服务器端追踪
server := grpcserver.New(
    grpcserver.WithAddress(":50051"),
    grpcserver.WithTracing(tracer),
)

// 客户端追踪
client := grpcclient.New(
    grpcclient.WithAddress("localhost:50051"),
    grpcclient.WithTracing(tracer),
)
```

## 配置选项

通过 `boot.WithProperty()` 或配置文件设置:

| 配置项 | 默认值 | 说明 |
|--------|--------|------|
| `grpc.server.enabled` | `false` | 是否启用 gRPC 服务器 |
| `grpc.server.address` | `:50051` | 服务器监听地址 |
| `grpc.client.address` | `localhost:50051` | 客户端连接地址 |
| `grpc.client.timeout` | `5` | 客户端超时(秒) |

### 示例配置

```yaml
# application.yml
grpc:
  server:
    enabled: true
    address: ":50051"
  client:
    address: "localhost:50051"
    timeout: 5
```

## 项目结构

```
go-boot-grpc/
├── autoconfig.go           # 自动配置注册
├── client/                 # gRPC 客户端
│   ├── client.go           # gRPC 客户端实现
│   └── options.go          # 客户端选项配置
├── server/                 # gRPC 服务器
│   ├── server.go           # gRPC 服务器实现
│   ├── options.go          # 服务器选项配置
│   └── interceptors/       # 拦截器
│       ├── tracing.go      # 分布式追踪拦截器
│       └── logging.go      # 日志拦截器
├── README.md
├── LICENSE
└── go.mod
```

## 开发指南

### 构建

```bash
go build ./...
```

### 测试

```bash
go test ./...
go test -cover ./...       # 带覆盖率
go test -race ./...        # 数据竞争检测
```

### 代码规范

```bash
go fmt ./...
golangci-lint run
```

## 贡献

欢迎提交 Issue 和 Pull Request!详细贡献指南请参阅 [CONTRIBUTING.md](./CONTRIBUTING.md)。

## 许可证

本项目采用 MIT 许可证 — 详情请参阅 [LICENSE](./LICENSE) 文件。