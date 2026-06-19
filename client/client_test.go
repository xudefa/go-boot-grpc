package client

import (
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
			want: "localhost:50051",
		},
		{
			name: "custom address",
			opts: []Option{WithAddress("localhost:8080")},
			want: "localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New(tt.opts...)
			if c == nil {
				t.Fatal("New() returned nil")
			}
			if c.address != tt.want {
				t.Errorf("address = %q, want %q", c.address, tt.want)
			}
		})
	}
}

func TestWithAddress(t *testing.T) {
	c := New(WithAddress("localhost:9090"))
	if c.address != "localhost:9090" {
		t.Errorf("WithAddress() = %q, want %q", c.address, "localhost:9090")
	}
}

func TestWithTimeout(t *testing.T) {
	timeout := 10 * time.Second
	c := New(WithTimeout(timeout))
	if c.timeout != timeout {
		t.Errorf("WithTimeout() = %v, want %v", c.timeout, timeout)
	}
}

func TestWithDialOptions(t *testing.T) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	}
	c := New(WithDialOptions(opts...))
	if c == nil {
		t.Fatal("New() with WithDialOptions returned nil")
	}
}

func TestAddress(t *testing.T) {
	c := New(WithAddress(":8080"))
	if got := c.Address(); got != ":8080" {
		t.Errorf("Address() = %q, want %q", got, ":8080")
	}
}

func TestTimeout(t *testing.T) {
	timeout := 3 * time.Second
	c := New(WithTimeout(timeout))
	if got := c.Timeout(); got != timeout {
		t.Errorf("Timeout() = %v, want %v", got, timeout)
	}
}

func TestCloseWithoutConnect(t *testing.T) {
	c := New()
	if err := c.Close(); err != nil {
		t.Errorf("Close() on unconnected client returned error: %v", err)
	}
}

func TestConnectTwice(t *testing.T) {
	c := New(WithAddress("localhost:50051"))

	// 第一次连接（可能失败因为没有服务器）
	_ = c.Connect()
	_ = c.Close() // 确保关闭

	// 再次尝试连接
	_ = c.Connect()

	// 测试重复连接的情况
	c2 := New(WithAddress("passthrough://invalid_address"))
	err2 := c2.Connect()
	if err2 == nil {
		// 如果连接成功，测试重复连接
		err3 := c2.Connect()
		if err3 == nil || err3.Error() != "gRPC client already connected to passthrough://invalid_address" {
			t.Errorf("Expected error for double connect, got: %v", err3)
		}
		_ = c2.Close()
	}
}

func TestConnectWithInvalidAddress(t *testing.T) {
	c := New(WithAddress("invalid_address:9999"))
	err := c.Connect()
	if err == nil {
		t.Error("Expected error when connecting to invalid address")
		_ = c.Close() // Clean up if connection somehow succeeded
	}
}
