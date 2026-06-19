package client

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xudefa/go-boot/tracing"
	"google.golang.org/grpc"
)

func TestNewClientInterceptorRegistry(t *testing.T) {
	registry := NewClientInterceptorRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.Tracer())
}

func TestRegisterClientInterceptor(t *testing.T) {
	registry := NewClientInterceptorRegistry()
	interceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	registry.Register(interceptor)
	assert.True(t, len(registry.interceptors) > 0)
}

func TestBuildClientChain(t *testing.T) {
	registry := NewClientInterceptorRegistry()

	callOrder := make([]string, 0)
	var mu sync.Mutex

	interceptor1 := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		mu.Lock()
		callOrder = append(callOrder, "interceptor1")
		mu.Unlock()
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	interceptor2 := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		mu.Lock()
		callOrder = append(callOrder, "interceptor2")
		mu.Unlock()
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	registry.Register(interceptor1)
	registry.Register(interceptor2)

	chain := registry.BuildChain()
	assert.NotNil(t, chain)

	invokerCalled := false
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		mu.Lock()
		invokerCalled = true
		callOrder = append(callOrder, "invoker")
		mu.Unlock()
		return nil
	}

	err := chain(context.Background(), "/TestService/TestMethod", nil, nil, nil, invoker)
	assert.NoError(t, err)
	assert.True(t, invokerCalled)
	assert.Equal(t, []string{"interceptor1", "interceptor2", "invoker"}, callOrder)
}

func TestSetClientTracer(t *testing.T) {
	registry := NewClientInterceptorRegistry()
	customTracer := &tracing.NoopTracer{}
	registry.SetTracer(customTracer)
	assert.Equal(t, customTracer, registry.Tracer())
}

func TestClientTracingInterceptor(t *testing.T) {
	tracer := tracing.GetTracer("test")
	interceptor := ClientTracingInterceptor(tracer)

	invokerCalled := false
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		invokerCalled = true
		return nil
	}

	err := interceptor(context.Background(), "/TestService/TestMethod", nil, nil, nil, invoker)
	assert.NoError(t, err)
	assert.True(t, invokerCalled)
}

func TestClientTracingInterceptorWithError(t *testing.T) {
	tracer := tracing.GetTracer("test")
	interceptor := ClientTracingInterceptor(tracer)

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return assert.AnError
	}

	err := interceptor(context.Background(), "/TestService/TestMethod", nil, nil, nil, invoker)
	assert.Error(t, err)
}

func TestClientInterceptorRegistryConcurrency(t *testing.T) {
	registry := NewClientInterceptorRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			interceptor := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
				return invoker(ctx, method, req, reply, cc, opts...)
			}
			registry.Register(interceptor)
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, len(registry.interceptors))
}

func TestClientExtractServiceName(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		want       string
	}{
		{
			name:       "standard method",
			fullMethod: "/TestService/TestMethod",
			want:       "/TestService",
		},
		{
			name:       "empty method",
			fullMethod: "",
			want:       "",
		},
		{
			name:       "single slash",
			fullMethod: "/TestService",
			want:       "/TestService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractServiceName(tt.fullMethod)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestClientExtractMethodName(t *testing.T) {
	tests := []struct {
		name       string
		fullMethod string
		want       string
	}{
		{
			name:       "standard method",
			fullMethod: "/TestService/TestMethod",
			want:       "TestMethod",
		},
		{
			name:       "empty method",
			fullMethod: "",
			want:       "",
		},
		{
			name:       "single slash",
			fullMethod: "/TestService",
			want:       "TestService",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMethodName(tt.fullMethod)
			assert.Equal(t, tt.want, got)
		})
	}
}
