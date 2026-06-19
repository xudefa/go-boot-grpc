// Package grpc 提供 gRPC 客户端和服务器的自动配置。
//
// 当 grpc.server.enabled=true 时自动启用，从 Environment 中读取 grpc.server.address、grpc.client.address、
// grpc.client.timeout 等配置项，
// 创建并注册 gRPC Server Bean（Bean ID: grpcServer）和 gRPC Client Bean（Bean ID: grpcClient）到 IoC 容器中。
package grpc

import (
	"time"

	"github.com/xudefa/go-boot-grpc/client"
	"github.com/xudefa/go-boot-grpc/server"

	"github.com/xudefa/go-boot/boot"
	"github.com/xudefa/go-boot/condition"
	"github.com/xudefa/go-boot/constants"
	"github.com/xudefa/go-boot/core"
	"github.com/xudefa/go-boot/tracing"
)

// GrpcAutoConfiguration gRPC 客户端和服务器的自动配置
//
// 从 Environment 中读取 grpc.server.address、grpc.client.address、grpc.client.timeout 等配置项，
// 创建 gRPC Server 和 gRPC Client 实例并注册到 IoC 容器中。
// 启用条件：grpc.server.enabled=true
type GrpcAutoConfiguration struct{}

// init 注册 gRPC 自动配置，由 grpc.server.enabled=true 条件控制
func init() {
	boot.RegisterAutoConfig(&GrpcAutoConfiguration{},
		condition.OnProperty(constants.GRPCServerEnabled, constants.ConditionTrue),
	)
}

// Configure 执行自动配置逻辑，创建 gRPC Server 和 Client 并注册为 Bean
func (g *GrpcAutoConfiguration) Configure(ctx boot.ApplicationContext) error {
	env := ctx.Environment()
	tracer := tracing.GetTracer("grpc")

	if addr := env.GetString(constants.GRPCServerAddress, ""); addr != "" {
		serverOpts := []server.Option{
			server.WithAddress(addr),
		}

		if env.GetBool("tracing.enabled", false) {
			serverOpts = append(serverOpts, server.WithTracing(tracer))
		}

		srv := server.New(serverOpts...)
		if err := ctx.Register(constants.GRPCServerBeanID,
			core.Bean(srv),
			core.Singleton(),
		); err != nil {
			return err
		}
	}

	if addr := env.GetString(constants.GRPCClientAddress, ""); addr != "" {
		timeout := env.GetInt(constants.GRPCClientTimeout, constants.DefaultGRPCClientTimeout)
		clientOpts := []client.Option{
			client.WithAddress(addr),
			client.WithTimeout(time.Duration(timeout) * time.Second),
		}

		if env.GetBool("tracing.enabled", false) {
			clientOpts = append(clientOpts, client.WithTracing(tracer))
		}

		cli := client.New(clientOpts...)
		if err := ctx.Register(constants.GRPCClientBeanID,
			core.Bean(cli),
			core.Singleton(),
		); err != nil {
			return err
		}
	}

	return nil
}
