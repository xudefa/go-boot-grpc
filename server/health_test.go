package server

import (
	"context"
	"log"
	"testing"

	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/xudefa/go-boot/health"
)

func TestMapHealthStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   health.Status
		expected grpc_health_v1.HealthCheckResponse_ServingStatus
	}{
		{"UP maps to SERVING", health.StatusUp, grpc_health_v1.HealthCheckResponse_SERVING},
		{"DEGRADED maps to SERVING", health.StatusDegraded, grpc_health_v1.HealthCheckResponse_SERVING},
		{"DOWN maps to NOT_SERVING", health.StatusDown, grpc_health_v1.HealthCheckResponse_NOT_SERVING},
		{"OUTAGE maps to NOT_SERVING", health.StatusOutage, grpc_health_v1.HealthCheckResponse_NOT_SERVING},
		{"UNKNOWN maps to UNKNOWN", health.StatusUnknown, grpc_health_v1.HealthCheckResponse_UNKNOWN},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapHealthStatus(tt.status)
			if result != tt.expected {
				t.Errorf("mapHealthStatus(%v) = %v, want %v", tt.status, result, tt.expected)
			}
		})
	}
}

func TestDefaultHealthIndicator(t *testing.T) {
	indicator := &defaultHealthIndicator{}

	if indicator.Name() != "grpc" {
		t.Errorf("Name() = %v, want grpc", indicator.Name())
	}

	h := indicator.Health(context.TODO())
	if h.Status != health.StatusUp {
		t.Errorf("Health().Status = %v, want UP", h.Status)
	}

	if h.Details["service"] != "grpc" {
		t.Errorf("Health().Details[\"service\"] = %v, want grpc", h.Details["service"])
	}
}

type mockIndicator struct {
	name   string
	status health.Status
	err    error
}

func (m *mockIndicator) Name() string {
	return m.name
}

func (m *mockIndicator) Health(ctx context.Context) health.Health {
	return health.Health{
		Status: m.status,
		Error:  m.err,
	}
}

func TestHealthServerCheck(t *testing.T) {
	logger := log.Default()

	tests := []struct {
		name      string
		service   string
		indicator health.Indicator
		expected  grpc_health_v1.HealthCheckResponse_ServingStatus
	}{
		{
			name:      "empty service returns SERVING",
			service:   "",
			indicator: &mockIndicator{name: "grpc", status: health.StatusUp},
			expected:  grpc_health_v1.HealthCheckResponse_SERVING,
		},
		{
			name:      "grpc service returns SERVING",
			service:   "grpc",
			indicator: &mockIndicator{name: "grpc", status: health.StatusUp},
			expected:  grpc_health_v1.HealthCheckResponse_SERVING,
		},
		{
			name:      "non-grpc service returns NOT_SERVING",
			service:   "other",
			indicator: &mockIndicator{name: "grpc", status: health.StatusUp},
			expected:  grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		},
		{
			name:      "DOWN status returns NOT_SERVING",
			service:   "grpc",
			indicator: &mockIndicator{name: "grpc", status: health.StatusDown},
			expected:  grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		},
		{
			name:      "UNKNOWN status returns UNKNOWN",
			service:   "grpc",
			indicator: &mockIndicator{name: "grpc", status: health.StatusUnknown},
			expected:  grpc_health_v1.HealthCheckResponse_UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hs := newHealthServer(tt.indicator, logger)
			req := &grpc_health_v1.HealthCheckRequest{Service: tt.service}
			resp, err := hs.Check(context.TODO(), req)

			if err != nil {
				t.Fatalf("Check() error = %v", err)
			}

			if resp.Status != tt.expected {
				t.Errorf("Check().Status = %v, want %v", resp.Status, tt.expected)
			}
		})
	}
}
