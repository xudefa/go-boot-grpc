// Package server 提供 gRPC 服务器拦截器。
//
// 包含服务器追踪拦截器和 OpenTelemetry 拦截器实现，
// 以及拦截器注册表（InterceptorRegistry）用于管理全局和服务级拦截器链。
package server

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

// interceptorType 拦截器类型
type interceptorType int

const (
	interceptorTypeGlobal  interceptorType = iota // 全局拦截器
	interceptorTypeService                        // 服务级拦截器
)

// interceptorEntry 拦截器注册条目
type interceptorEntry struct {
	interceptor grpc.UnaryServerInterceptor // 拦截器函数
	itype       interceptorType             // 拦截器类型
	serviceName string                      // 服务名称（仅服务级拦截器有效）
}

// InterceptorRegistry gRPC 服务器拦截器注册表
//
// 管理全局和服务级一元拦截器，支持动态注册和按方法构建拦截器链。
type InterceptorRegistry struct {
	mu           sync.RWMutex       // 保护 interceptors 的并发访问
	interceptors []interceptorEntry // 已注册的拦截器列表
	tracer       tracing.Tracer     // 链路追踪器
}

// NewInterceptorRegistry 创建服务器拦截器注册表
func NewInterceptorRegistry() *InterceptorRegistry {
	return &InterceptorRegistry{
		interceptors: make([]interceptorEntry, 0),
		tracer:       tracing.GetTracer("grpc-server"),
	}
}

// RegisterGlobal 注册全局拦截器
func (r *InterceptorRegistry) RegisterGlobal(interceptor grpc.UnaryServerInterceptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.interceptors = append(r.interceptors, interceptorEntry{
		interceptor: interceptor,
		itype:       interceptorTypeGlobal,
	})
}

// RegisterService 注册服务级拦截器，仅对指定服务生效
func (r *InterceptorRegistry) RegisterService(serviceName string, interceptor grpc.UnaryServerInterceptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.interceptors = append(r.interceptors, interceptorEntry{
		interceptor: interceptor,
		itype:       interceptorTypeService,
		serviceName: serviceName,
	})
}

// BuildChain 根据方法全限定名构建拦截器链
//
// 全局拦截器对所有方法生效，服务级拦截器仅对匹配的服务生效。
func (r *InterceptorRegistry) BuildChain(fullMethod string) grpc.UnaryServerInterceptor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.interceptors) == 0 {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	serviceName := extractServiceName(fullMethod)
	chain := make([]grpc.UnaryServerInterceptor, 0, len(r.interceptors))

	for _, entry := range r.interceptors {
		if entry.itype == interceptorTypeGlobal {
			chain = append(chain, entry.interceptor)
		} else if entry.itype == interceptorTypeService && entry.serviceName == serviceName {
			chain = append(chain, entry.interceptor)
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		currentHandler := handler
		for i := len(chain) - 1; i >= 0; i-- {
			interceptor := chain[i]
			nextHandler := currentHandler
			currentHandler = func(ctx context.Context, req interface{}) (interface{}, error) {
				return interceptor(ctx, req, info, nextHandler)
			}
		}
		return currentHandler(ctx, req)
	}
}

// SetTracer 设置链路追踪器
func (r *InterceptorRegistry) SetTracer(tracer tracing.Tracer) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tracer = tracer
}

// Tracer 返回当前链路追踪器
func (r *InterceptorRegistry) Tracer() tracing.Tracer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.tracer
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

// TracingInterceptor 创建 gRPC 服务器链路追踪拦截器
func TracingInterceptor(tracer tracing.Tracer) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if _, ok := tracer.(*tracing.NoopTracer); ok {
			return handler(ctx, req)
		}

		spanName := info.FullMethod
		ctx, span := tracer.Start(ctx, spanName,
			tracing.WithSpanKind(tracing.SpanKindServer),
			tracing.WithAttribute("rpc.system", "grpc"),
			tracing.WithAttribute("rpc.service", extractServiceName(info.FullMethod)),
			tracing.WithAttribute("rpc.method", extractMethodName(info.FullMethod)),
		)
		defer span.End()

		resp, err := handler(ctx, req)

		if err != nil {
			span.SetError(err)
			span.SetStatus(tracing.SpanStatusError)
		} else {
			span.SetStatus(tracing.SpanStatusOK)
		}

		return resp, err
	}
}

// OpenTelemetryTracingInterceptor 创建基于 OpenTelemetry 的 gRPC 服务器追踪拦截器
func OpenTelemetryTracingInterceptor() grpc.UnaryServerInterceptor {
	tracer := otel.Tracer("grpc-server")
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		spanName := info.FullMethod
		attrs := []attribute.KeyValue{
			semconv.RPCSystemKey.String("grpc"),
			semconv.RPCServiceKey.String(extractServiceName(info.FullMethod)),
			semconv.RPCMethodKey.String(extractMethodName(info.FullMethod)),
		}

		ctx, span := tracer.Start(
			ctx,
			spanName,
			trace.WithAttributes(attrs...),
			trace.WithSpanKind(trace.SpanKindServer),
		)
		defer span.End()

		resp, err := handler(ctx, req)

		if err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
		} else {
			span.SetStatus(codes.Ok, "")
		}

		return resp, err
	}
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
