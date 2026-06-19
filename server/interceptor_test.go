package server

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xudefa/go-boot/tracing"
	"google.golang.org/grpc"
)

func TestNewInterceptorRegistry(t *testing.T) {
	registry := NewInterceptorRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.Tracer())
}

func TestRegisterGlobalInterceptor(t *testing.T) {
	registry := NewInterceptorRegistry()
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}

	registry.RegisterGlobal(interceptor)
	assert.True(t, len(registry.interceptors) > 0)
}

func TestRegisterServiceInterceptor(t *testing.T) {
	registry := NewInterceptorRegistry()
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}

	registry.RegisterService("TestService", interceptor)
	assert.True(t, len(registry.interceptors) > 0)
}

func TestBuildChain(t *testing.T) {
	registry := NewInterceptorRegistry()

	callOrder := make([]string, 0)
	var mu sync.Mutex

	interceptor1 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		mu.Lock()
		callOrder = append(callOrder, "interceptor1")
		mu.Unlock()
		return handler(ctx, req)
	}

	interceptor2 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		mu.Lock()
		callOrder = append(callOrder, "interceptor2")
		mu.Unlock()
		return handler(ctx, req)
	}

	registry.RegisterGlobal(interceptor1)
	registry.RegisterGlobal(interceptor2)

	chain := registry.BuildChain("/TestService/TestMethod")
	assert.NotNil(t, chain)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		mu.Lock()
		handlerCalled = true
		callOrder = append(callOrder, "handler")
		mu.Unlock()
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/TestService/TestMethod",
	}

	_, _ = chain(context.Background(), nil, info, handler)

	assert.True(t, handlerCalled)
	assert.Equal(t, []string{"interceptor1", "interceptor2", "handler"}, callOrder)
}

func TestServiceSpecificInterceptor(t *testing.T) {
	registry := NewInterceptorRegistry()

	globalCalled := false
	serviceCalled := false

	globalInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		globalCalled = true
		return handler(ctx, req)
	}

	serviceInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		serviceCalled = true
		return handler(ctx, req)
	}

	registry.RegisterGlobal(globalInterceptor)
	registry.RegisterService("/TestService", serviceInterceptor)

	chain := registry.BuildChain("/TestService/TestMethod")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/TestService/TestMethod",
	}

	_, _ = chain(context.Background(), nil, info, handler)

	assert.True(t, globalCalled)
	assert.True(t, serviceCalled)
}

func TestServiceInterceptorNotAppliedToOtherService(t *testing.T) {
	registry := NewInterceptorRegistry()

	serviceCalled := false

	serviceInterceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		serviceCalled = true
		return handler(ctx, req)
	}

	registry.RegisterService("/TestService", serviceInterceptor)

	chain := registry.BuildChain("/OtherService/TestMethod")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/OtherService/TestMethod",
	}

	_, _ = chain(context.Background(), nil, info, handler)

	assert.False(t, serviceCalled)
}

func TestSetTracer(t *testing.T) {
	registry := NewInterceptorRegistry()
	customTracer := &tracing.NoopTracer{}
	registry.SetTracer(customTracer)
	assert.Equal(t, customTracer, registry.Tracer())
}

func TestExtractServiceName(t *testing.T) {
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

func TestExtractMethodName(t *testing.T) {
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

func TestTracingInterceptor(t *testing.T) {
	tracer := tracing.GetTracer("test")
	interceptor := TracingInterceptor(tracer)

	handlerCalled := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		handlerCalled = true
		return nil, nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/TestService/TestMethod",
	}

	_, err := interceptor(context.Background(), nil, info, handler)
	assert.NoError(t, err)
	assert.True(t, handlerCalled)
}

func TestTracingInterceptorWithError(t *testing.T) {
	tracer := tracing.GetTracer("test")
	interceptor := TracingInterceptor(tracer)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, assert.AnError
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/TestService/TestMethod",
	}

	_, err := interceptor(context.Background(), nil, info, handler)
	assert.Error(t, err)
}

func TestInterceptorRegistryConcurrency(t *testing.T) {
	registry := NewInterceptorRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				return handler(ctx, req)
			}
			registry.RegisterGlobal(interceptor)
		}()
	}

	wg.Wait()
	assert.Equal(t, 100, len(registry.interceptors))
}
