// Package server 提供了一个通用的 gRPC 服务器框架，内置中间件支持。
//
// # 快速开始
//
// 使用默认选项创建 gRPC 服务器：
//
//	srv := server.New()
//	pb.RegisterYourServiceServer(srv.GRPCServer(), &yourService{})
//	if err := srv.Start(); err != nil {
//	    log.Fatal(err)
//	}
//
// # 自定义配置
//
// 使用函数式选项自定义服务器：
//
//	srv := server.New(
//	    server.WithAddress(":9090"),
//	    server.WithLogger(customLogger),
//	)
//
// # 中间件
//
// 服务器默认配备了恢复和日志拦截器。
// 添加自定义拦截器：
//
//	srv := server.New(
//	    server.WithUnaryInterceptor(yourInterceptor),
//	)
//
// # 优雅关闭
//
//	srv.Stop() // 优雅停止
package server

import (
	"context"
	"log"
	"net"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/xudefa/go-boot/health"
	"github.com/xudefa/go-boot/tracing"
	"google.golang.org/grpc"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
)

// Option 是配置 Server 的函数式选项。
type Option func(*Server)

// Server 封装了 gRPC 服务器，提供中间件和生命周期管理。
type Server struct {
	grpcServer          *grpc.Server
	lis                 net.Listener
	address             string
	logger              *log.Logger
	healthEnabled       bool
	healthIndicator     health.Indicator
	interceptorRegistry *InterceptorRegistry
	tracingEnabled      bool
}

// WithAddress 设置服务器监听地址。
func WithAddress(addr string) Option {
	return func(s *Server) {
		s.address = addr
	}
}

// WithLogger 设置服务器日志器。
func WithLogger(logger *log.Logger) Option {
	return func(s *Server) {
		s.logger = logger
	}
}

// WithUnaryInterceptor 设置自定义一元拦截器，会追加到默认的中间件链中。
func WithUnaryInterceptor(interceptor grpc.UnaryServerInterceptor) Option {
	return func(s *Server) {
		if s.interceptorRegistry == nil {
			s.interceptorRegistry = NewInterceptorRegistry()
		}
		s.interceptorRegistry.RegisterGlobal(interceptor)
	}
}

// WithServerOptions 设置自定义 gRPC 服务器选项，会替换默认的中间件。
func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(s *Server) {
		s.grpcServer = grpc.NewServer(opts...)
	}
}

func WithHealthCheck() Option {
	return func(s *Server) {
		s.healthEnabled = true
		s.healthIndicator = &defaultHealthIndicator{}
	}
}

func WithHealthIndicator(indicator health.Indicator) Option {
	return func(s *Server) {
		s.healthIndicator = indicator
	}
}

func WithInterceptorRegistry(registry *InterceptorRegistry) Option {
	return func(s *Server) {
		s.interceptorRegistry = registry
	}
}

func WithTracing(tracer tracing.Tracer) Option {
	return func(s *Server) {
		s.tracingEnabled = true
		if s.interceptorRegistry == nil {
			s.interceptorRegistry = NewInterceptorRegistry()
		}
		s.interceptorRegistry.SetTracer(tracer)
	}
}

func WithGlobalInterceptor(interceptor grpc.UnaryServerInterceptor) Option {
	return func(s *Server) {
		if s.interceptorRegistry == nil {
			s.interceptorRegistry = NewInterceptorRegistry()
		}
		s.interceptorRegistry.RegisterGlobal(interceptor)
	}
}

func WithServiceInterceptor(serviceName string, interceptor grpc.UnaryServerInterceptor) Option {
	return func(s *Server) {
		if s.interceptorRegistry == nil {
			s.interceptorRegistry = NewInterceptorRegistry()
		}
		s.interceptorRegistry.RegisterService(serviceName, interceptor)
	}
}

func New(opts ...Option) *Server {
	s := &Server{
		address:             ":50051",
		logger:              log.Default(),
		interceptorRegistry: NewInterceptorRegistry(),
	}

	defaultInterceptors := []grpc.UnaryServerInterceptor{
		recovery.UnaryServerInterceptor(),
		logging.UnaryServerInterceptor(logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
			s.logger.Printf("[%v] %s %v", lvl, msg, fields)
		})),
	}

	for _, interceptor := range defaultInterceptors {
		s.interceptorRegistry.RegisterGlobal(interceptor)
	}

	for _, opt := range opts {
		opt(s)
	}

	if s.tracingEnabled {
		tracingInterceptor := TracingInterceptor(s.interceptorRegistry.Tracer())
		s.interceptorRegistry.RegisterGlobal(tracingInterceptor)
	}

	s.grpcServer = grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				return s.interceptorRegistry.BuildChain(info.FullMethod)(ctx, req, info, handler)
			},
		),
	)

	if s.healthEnabled {
		healthServer := newHealthServer(s.healthIndicator, s.logger)
		grpc_health_v1.RegisterHealthServer(s.grpcServer, healthServer)
	}

	return s
}

// GRPCServer 返回底层的 gRPC 服务器实例，用于注册服务。
func (s *Server) GRPCServer() *grpc.Server {
	return s.grpcServer
}

// Start 启动 gRPC 服务器，开始监听并处理请求。
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.address)
	if err != nil {
		return err
	}
	s.lis = lis

	s.logger.Printf("gRPC server listening on %s", s.address)
	return s.grpcServer.Serve(lis)
}

// Stop 优雅停止 gRPC 服务器，等待现有请求完成。
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

// Address 返回服务器监听地址，如果已启动则返回实际监听地址。
func (s *Server) Address() string {
	if s.lis != nil {
		return s.lis.Addr().String()
	}
	return s.address
}

func (s *Server) InterceptorRegistry() *InterceptorRegistry {
	return s.interceptorRegistry
}
