// Package client 提供 gRPC 客户端拦截器。
//
// 包含客户端追踪拦截器和 OpenTelemetry 拦截器实现，
// 以及拦截器注册表（ClientInterceptorRegistry）用于管理拦截器链。
package client

import (
	"context"
	"sync"

	"github.com/xudefa/go-boot/tracing"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
)

// ClientInterceptorRegistry gRPC 客户端拦截器注册表
//
// 管理客户端一元拦截器链，支持动态注册和构建拦截器链。
type ClientInterceptorRegistry struct {
	mu           sync.RWMutex                  // 保护 interceptors 的并发访问
	interceptors []grpc.UnaryClientInterceptor // 已注册的拦截器列表
	tracer       tracing.Tracer                // 链路追踪器
}

// NewClientInterceptorRegistry 创建客户端拦截器注册表
func NewClientInterceptorRegistry() *ClientInterceptorRegistry {
	return &ClientInterceptorRegistry{
		interceptors: make([]grpc.UnaryClientInterceptor, 0),
		tracer:       tracing.GetTracer("grpc-client"),
	}
}

// Register 注册客户端拦截器
func (r *ClientInterceptorRegistry) Register(interceptor grpc.UnaryClientInterceptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.interceptors = append(r.interceptors, interceptor)
}

// BuildChain 构建拦截器链，将所有已注册的拦截器组合为一个
func (r *ClientInterceptorRegistry) BuildChain() grpc.UnaryClientInterceptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.interceptors) == 0 {
		return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	chain := make([]grpc.UnaryClientInterceptor, len(r.interceptors))
	copy(chain, r.interceptors)

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		currentInvoker := invoker
		for i := len(chain) - 1; i >= 0; i-- {
			interceptor := chain[i]
			nextInvoker := currentInvoker
			currentInvoker = func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
				return interceptor(ctx, method, req, reply, cc, nextInvoker, opts...)
			}
		}
		return currentInvoker(ctx, method, req, reply, cc, opts...)
	}
}

// SetTracer 设置链路追踪器
func (r *ClientInterceptorRegistry) SetTracer(tracer tracing.Tracer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracer = tracer
}

// Tracer 返回当前链路追踪器
func (r *ClientInterceptorRegistry) Tracer() tracing.Tracer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tracer
}

// ClientTracingInterceptor 创建 gRPC 客户端链路追踪拦截器
func ClientTracingInterceptor(tracer tracing.Tracer) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if _, ok := tracer.(*tracing.NoopTracer); ok {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		spanName := method
		ctx, span := tracer.Start(ctx, spanName,
			tracing.WithSpanKind(tracing.SpanKindClient),
			tracing.WithAttribute("rpc.system", "grpc"),
			tracing.WithAttribute("rpc.service", extractServiceName(method)),
			tracing.WithAttribute("rpc.method", extractMethodName(method)),
			tracing.WithAttribute("net.peer.name", cc.Target()),
		)
		defer span.End()

		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			span.SetError(err)
			span.SetStatus(tracing.SpanStatusError)
		} else {
			span.SetStatus(tracing.SpanStatusOK)
		}

		return err
	}
}

// ClientOpenTelemetryTracingInterceptor 创建基于 OpenTelemetry 的 gRPC 客户端追踪拦截器
func ClientOpenTelemetryTracingInterceptor() grpc.UnaryClientInterceptor {
	tracer := otel.Tracer("grpc-client")
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		spanName := method
		attrs := []attribute.KeyValue{
			semconv.RPCSystemKey.String("grpc"),
			semconv.RPCServiceKey.String(extractServiceName(method)),
			semconv.RPCMethodKey.String(extractMethodName(method)),
			semconv.NetPeerNameKey.String(cc.Target()),
		}

		ctx, span := tracer.Start(
			ctx,
			spanName,
			trace.WithAttributes(attrs...),
			trace.WithSpanKind(trace.SpanKindClient),
		)
		defer span.End()

		err := invoker(ctx, method, req, reply, cc, opts...)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return err
	}
}

func extractServiceName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return ""
	}
	for i := 1; i < len(fullMethod); i++ {
		if fullMethod[i] == '/' && fullMethod[i-1] != '/' {
			return fullMethod[:i]
		}
	}
	return fullMethod
}

func extractMethodName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return ""
	}
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}
