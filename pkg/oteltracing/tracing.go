package oteltracing

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

// InitTracerProvider 初始化 OpenTelemetry TracerProvider
// 创建 OTLP gRPC exporter，配置资源属性，并设置为全局 tracer provider
// serviceName: 服务名称，用于标识当前服务
// endpoint: OTLP collector 地址，为空时默认使用 "localhost:4317"
// 返回的 TracerProvider 可用于手动关闭资源（如 Shutdown）
func InitTracerProvider(serviceName, endpoint string) (*sdktrace.TracerProvider, error) {
	ctx := context.Background()

	// 如果 endpoint 为空，使用默认地址（Collector 的 OTLP gRPC 端口）
	if endpoint == "" {
		endpoint = "localhost:4319"
	}

	// 创建 OTLP gRPC exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC exporter: %w", err)
	}

	// 创建资源，包含服务名属性
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建 TracerProvider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// 设置为全局 TracerProvider
	otel.SetTracerProvider(tp)

	return tp, nil
}
