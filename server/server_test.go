package server

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/xudefa/go-boot/health"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/test/bufconn"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name string
		opts []Option
		want string
	}{
		{
			name: "default address",
			opts: nil,
			want: ":50051",
		},
		{
			name: "custom address",
			opts: []Option{WithAddress(":8080")},
			want: ":8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New(tt.opts...)
			if s == nil {
				t.Fatal("New() returned nil")
			}
			if s.address != tt.want {
				t.Errorf("address = %q, want %q", s.address, tt.want)
			}
		})
	}
}

func TestWithAddress(t *testing.T) {
	s := New(WithAddress(":9090"))
	if s.address != ":9090" {
		t.Errorf("WithAddress() = %q, want %q", s.address, ":9090")
	}
}

func TestWithServerOptions(t *testing.T) {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(1024),
	}
	s := New(WithServerOptions(opts...))
	if s == nil {
		t.Fatal("New() with WithServerOptions returned nil")
	}
}

func TestStartAndStop(t *testing.T) {
	s := New(WithAddress(":0"))
	go func() {
		if err := s.Start(); err != nil {
			t.Errorf("Start() failed: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	addr := s.Address()
	if addr == "" {
		t.Error("Address() returned empty string after Start()")
	}

	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("failed to parse address %q: %v", addr, err)
	}
	if port == "0" {
		t.Error("port should not be 0 after Start()")
	}

	s.Stop()
}

func TestGRPCServer(t *testing.T) {
	s := New()
	grpcSrv := s.GRPCServer()
	if grpcSrv == nil {
		t.Error("GRPCServer() returned nil")
	}
}

func TestWithUnaryInterceptor(t *testing.T) {
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}
	s := New(WithUnaryInterceptor(interceptor))
	if s == nil {
		t.Fatal("New() with WithUnaryInterceptor returned nil")
	}
}

func TestServerWithHealthCheck(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)

	srv := New(
		WithAddress("bufnet"),
		WithHealthCheck(),
	)

	done := make(chan error, 1)
	go func() {
		err := srv.grpcServer.Serve(lis)
		if err != nil && err != grpc.ErrServerStopped {
			done <- err
		} else {
			done <- nil
		}
	}()
	defer srv.Stop()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial("bufnet", //nolint:staticcheck // grpc.Dial is deprecated but still supported in 1.x
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	healthClient := grpc_health_v1.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "grpc"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	assert.Equal(t, grpc_health_v1.HealthCheckResponse_SERVING, resp.Status)

	// Stop the server to allow it to shut down gracefully
	srv.Stop()
}

type customHealthIndicator struct {
	status health.Status
}

func (c *customHealthIndicator) Name() string {
	return "custom"
}

func (c *customHealthIndicator) Health(ctx context.Context) health.Health {
	return health.Health{
		Status: c.status,
	}
}

func TestServerWithCustomHealthIndicator(t *testing.T) {
	lis := bufconn.Listen(1024 * 1024)

	indicator := &customHealthIndicator{status: health.StatusDown}

	srv := New(
		WithAddress("bufnet"),
		WithHealthCheck(),
		WithHealthIndicator(indicator),
	)

	done := make(chan error, 1)
	go func() {
		err := srv.grpcServer.Serve(lis)
		if err != nil && err != grpc.ErrServerStopped {
			done <- err
		} else {
			done <- nil
		}
	}()
	defer srv.Stop()

	// Give server a moment to start
	time.Sleep(100 * time.Millisecond)

	conn, err := grpc.Dial("bufnet", //nolint:staticcheck // grpc.Dial is deprecated but still supported in 1.x
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	healthClient := grpc_health_v1.NewHealthClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resp, err := healthClient.Check(ctx, &grpc_health_v1.HealthCheckRequest{Service: "grpc"})
	if err != nil {
		t.Fatalf("Check() error = %v", err)
	}

	assert.Equal(t, grpc_health_v1.HealthCheckResponse_NOT_SERVING, resp.Status)

	// Stop the server to allow it to shut down gracefully
	srv.Stop()
}

func TestServerWithTracing(t *testing.T) {
	srv := New(
		WithAddress(":0"),
		WithTracing(nil), // Using nil tracer for test
	)
	assert.NotNil(t, srv)
	assert.True(t, srv.tracingEnabled)
}

func TestServerWithGlobalInterceptor(t *testing.T) {
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}

	srv := New(
		WithAddress(":0"),
		WithGlobalInterceptor(interceptor),
	)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.InterceptorRegistry())
}

func TestServerWithServiceInterceptor(t *testing.T) {
	interceptor := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	}

	srv := New(
		WithAddress(":0"),
		WithServiceInterceptor("test.service", interceptor),
	)
	assert.NotNil(t, srv)
	assert.NotNil(t, srv.InterceptorRegistry())
}

func TestServerAddress(t *testing.T) {
	srv := New(WithAddress(":8080"))
	assert.Equal(t, ":8080", srv.Address())
}
