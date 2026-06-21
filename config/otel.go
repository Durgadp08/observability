package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	Tracer = otel.Tracer(getServiceName())
	Meter  = otel.Meter(getServiceName())
)

func SetupOTelSDK(ctx context.Context, log *slog.Logger) (func(context.Context) error, error) {
	var shutdownFuncs []func(context.Context) error

	shutdown := func(ctx context.Context) error {
		var err error

		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}

		shutdownFuncs = nil
		return err
	}

	conn, err := getGrpcConn()
	if err != nil {
		log.ErrorContext(ctx, "error while get grpc connection", "err", err)
		return nil, fmt.Errorf("%w", err)
	}

	traceShutdown, err := setupTracer(ctx, conn)
	if err != nil {
		log.ErrorContext(ctx, "error while setup tracer", "err", err)
		return nil, fmt.Errorf("%w", err)
	}
	shutdownFuncs = append(shutdownFuncs, traceShutdown)

	metricShutdown, err := setupMeter(ctx, conn)
	if err != nil {
		log.ErrorContext(ctx, "error while setup meter", "err", err)
		return nil, err
	}
	shutdownFuncs = append(shutdownFuncs, metricShutdown)

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	log.InfoContext(ctx, "otel is initialized", "endpoint", getOtelEndpoint(), "service name", getServiceName())

	return shutdown, nil
}

func setupTracer(ctx context.Context, conn *grpc.ClientConn) (func(context.Context) error, error) {

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return nil, err
	}

	res := newResource()

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}

func setupMeter(ctx context.Context, conn *grpc.ClientConn) (func(context.Context) error, error) {
	exporter, err := otlpmetricgrpc.New(
		ctx,
		otlpmetricgrpc.WithGRPCConn(conn),
	)
	if err != nil {
		return nil, err
	}

	reader := sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(10*time.Second),
	)

	res := newResource()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)

	otel.SetMeterProvider(mp)

	return mp.Shutdown, nil
}

func getGrpcConn() (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(getOtelEndpoint(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func newResource() *resource.Resource {
	return resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(getServiceName()),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.DeploymentEnvironmentKey.String(getAppEnv()),
	)
}

func getOtelEndpoint() string {
	if v := os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"); v != "" {
		return v
	}
	return "localhost:4317"
}

func getServiceName() string {
	if v := os.Getenv("SERVICE_NAME"); v != "" {
		return v
	}
	return "observeblity"
}

func getAppEnv() string {
	if v := os.Getenv("APP_ENV"); v != "" {
		return v
	}
	return "development"
}
