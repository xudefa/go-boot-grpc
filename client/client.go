// Package client 提供了一个通用的 gRPC 客户端框架。
//
// # 快速开始
//
// 创建客户端并连接：
//
//	cli := client.New()
//	if err := cli.Connect(); err != nil {
//	    log.Fatal(err)
//	}
//	defer cli.Close()
//	// 使用 cli.Conn() 创建服务客户端：
//	grpcClient := pb.NewYourServiceClient(cli.Conn())
//
// # 自定义配置
//
// 使用函数式选项自定义客户端：
//
//	cli := client.New(
//	    client.WithAddress("localhost:9090"),
//	    client.WithTimeout(10*time.Second),
//	)
//
// # Dial 选项
//
// 添加自定义 gRPC dial 选项：
//
//	cli := client.New(
//	    client.WithDialOptions(
//	        grpc.WithPerRPCCredentials(yourCreds),
//	    ),
//	)
package client

import (
	"context"
	"fmt"
	"time"

	"github.com/xudefa/go-boot/log"
	"github.com/xudefa/go-boot/tracing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// Option 是配置 Client 的函数式选项。
type Option func(*Client)

// Client 封装了 gRPC 客户端连接和相关配置。
type Client struct {
	conn                *grpc.ClientConn
	address             string
	timeout             time.Duration
	logger              log.Logger
	opts                []grpc.DialOption
	interceptorRegistry *ClientInterceptorRegistry
	tracingEnabled      bool
}

// WithAddress 设置客户端连接地址。
func WithAddress(addr string) Option {
	return func(c *Client) {
		c.address = addr
	}
}

// WithTimeout 设置连接超时时间。
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.timeout = timeout
	}
}

// WithLogger 设置客户端日志器。
func WithLogger(logger log.Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithDialOptions 添加自定义 gRPC dial 选项。
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(c *Client) {
		c.opts = append(c.opts, opts...)
	}
}

func WithInterceptorRegistry(registry *ClientInterceptorRegistry) Option {
	return func(c *Client) {
		c.interceptorRegistry = registry
	}
}

func WithTracing(tracer tracing.Tracer) Option {
	return func(c *Client) {
		c.tracingEnabled = true
		if c.interceptorRegistry == nil {
			c.interceptorRegistry = NewClientInterceptorRegistry()
		}
		c.interceptorRegistry.SetTracer(tracer)
	}
}

func WithClientInterceptor(interceptor grpc.UnaryClientInterceptor) Option {
	return func(c *Client) {
		if c.interceptorRegistry == nil {
			c.interceptorRegistry = NewClientInterceptorRegistry()
		}
		c.interceptorRegistry.Register(interceptor)
	}
}

// New 创建一个新的 gRPC 客户端实例，可传入函数式选项进行配置。
func New(opts ...Option) *Client {
	c := &Client{
		address:             "localhost:50051",
		timeout:             5 * time.Second,
		logger:              log.NewSlogLogger(),
		interceptorRegistry: NewClientInterceptorRegistry(),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Connect 连接到 gRPC 服务器。
func (c *Client) Connect() error {
	if c.conn != nil {
		if c.conn.GetState() != connectivity.Shutdown {
			return fmt.Errorf("gRPC client already connected to %s", c.address)
		}
	}

	ctx := context.Background()
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}

	if c.tracingEnabled {
		tracingInterceptor := ClientTracingInterceptor(c.interceptorRegistry.Tracer())
		c.interceptorRegistry.Register(tracingInterceptor)
	}

	dialOpts = append(dialOpts, grpc.WithChainUnaryInterceptor(c.interceptorRegistry.BuildChain()))

	if c.timeout > 0 {
		dialOpts = append(dialOpts, grpc.WithConnectParams(grpc.ConnectParams{
			MinConnectTimeout: c.timeout,
		}))
	}
	dialOpts = append(dialOpts, c.opts...)

	conn, err := grpc.NewClient(c.address, dialOpts...)
	if err != nil {
		c.logger.Error(ctx, "Failed to create gRPC client connection",
			log.KeyValue{Key: "address", Value: c.address},
			log.KeyValue{Key: "error", Value: err},
		)
		return fmt.Errorf("failed to create gRPC client connection to %s: %w", c.address, err)
	}

	c.conn = conn
	if c.timeout > 0 {
		timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
		defer cancel()
		for state := conn.GetState(); state != connectivity.Ready; state = conn.GetState() {
			if !conn.WaitForStateChange(timeoutCtx, state) {
				c.logger.Warn(ctx, "gRPC client connection timed out",
					log.KeyValue{Key: "address", Value: c.address},
					log.KeyValue{Key: "timeout", Value: c.timeout},
				)
				return fmt.Errorf("gRPC client connection to %s timed out after %v", c.address, c.timeout)
			}
		}
	}
	c.logger.Info(ctx, "gRPC client connected successfully",
		log.KeyValue{Key: "address", Value: c.address})
	return nil
}

// Timeout 返回客户端配置的超时时间。
func (c *Client) Timeout() time.Duration {
	return c.timeout
}

// Conn 返回底层的 gRPC 连接，用于创建服务客户端。
func (c *Client) Conn() *grpc.ClientConn {
	return c.conn
}

// Close 关闭 gRPC 客户端连接。
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// Address 返回客户端连接地址。
func (c *Client) Address() string {
	return c.address
}
