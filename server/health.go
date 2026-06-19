// Package server 提供 gRPC 健康检查服务实现。
//
// 实现 grpc.health.v1.Health 服务接口，将 go-boot health.Indicator
// 适配为 gRPC 标准健康检查协议。
package server

import (
	"context"
	"log"

	grpc_health_v1 "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/xudefa/go-boot/health"
)

// healthServer gRPC 健康检查服务器
//
// 实现 grpc_health_v1.HealthServer 接口，
// 将 health.Indicator 的检查结果映射为 gRPC 健康状态。
type healthServer struct {
	grpc_health_v1.UnimplementedHealthServer
	indicator health.Indicator // 健康指标
	logger    *log.Logger      // 日志记录器
}

// newHealthServer 创建 gRPC 健康检查服务器
func newHealthServer(indicator health.Indicator, logger *log.Logger) *healthServer {
	return &healthServer{
		indicator: indicator,
		logger:    logger,
	}
}

// defaultHealthIndicator 默认健康指标，始终返回 UP 状态
type defaultHealthIndicator struct{}

// Name 返回健康指标名称
func (d *defaultHealthIndicator) Name() string {
	return "grpc"
}

// Health 执行健康检查
func (d *defaultHealthIndicator) Health(ctx context.Context) health.Health {
	return health.Health{
		Status: health.StatusUp,
		Details: map[string]any{
			"service": "grpc",
		},
	}
}

// mapHealthStatus 将 health.Status 映射为 gRPC 健康检查响应状态
func mapHealthStatus(status health.Status) grpc_health_v1.HealthCheckResponse_ServingStatus {
	switch status {
	case health.StatusUp, health.StatusDegraded:
		return grpc_health_v1.HealthCheckResponse_SERVING
	case health.StatusDown, health.StatusOutage:
		return grpc_health_v1.HealthCheckResponse_NOT_SERVING
	default:
		return grpc_health_v1.HealthCheckResponse_UNKNOWN
	}
}

// Check 实现 gRPC 健康检查的一元 RPC 方法
func (h *healthServer) Check(ctx context.Context, req *grpc_health_v1.HealthCheckRequest) (*grpc_health_v1.HealthCheckResponse, error) {
	if req.Service != "" && req.Service != "grpc" {
		return &grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		}, nil
	}

	healthResult := h.indicator.Health(ctx)

	if healthResult.Error != nil {
		h.logger.Printf("health check error: %v", healthResult.Error)
	}

	status := mapHealthStatus(healthResult.Status)

	return &grpc_health_v1.HealthCheckResponse{
		Status: status,
	}, nil
}

// Watch 实现 gRPC 健康检查的服务端流式 RPC 方法
func (h *healthServer) Watch(req *grpc_health_v1.HealthCheckRequest, stream grpc_health_v1.Health_WatchServer) error {
	if req.Service != "" && req.Service != "grpc" {
		return stream.Send(&grpc_health_v1.HealthCheckResponse{
			Status: grpc_health_v1.HealthCheckResponse_NOT_SERVING,
		})
	}

	healthResult := h.indicator.Health(stream.Context())
	status := mapHealthStatus(healthResult.Status)

	return stream.Send(&grpc_health_v1.HealthCheckResponse{
		Status: status,
	})
}
