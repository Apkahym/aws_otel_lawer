package otel

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	initOnce       sync.Once
	initErr        error
	tracerProvider *sdktrace.TracerProvider
)

// ShutdownFunc es una función para cerrar el TracerProvider
type ShutdownFunc func(context.Context) error

// Initialize inicializa OpenTelemetry de forma idempotente y fail-open
func Initialize(ctx context.Context) (ShutdownFunc, error) {
	initOnce.Do(func() {
		initErr = doInitialize(ctx)
	})

	if initErr != nil {
		// Retornar función noop para shutdown
		return func(context.Context) error { return nil }, initErr
	}

	return shutdown, nil
}

func doInitialize(ctx context.Context) error {
	// Cargar configuración desde variables de entorno
	cfg := LoadConfig()

	// Crear Resource con metadata de AWS Lambda
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.CloudProviderAWS,
			semconv.FaaSName(os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
			semconv.FaaSVersion(os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")),
			semconv.CloudRegion(os.Getenv("AWS_REGION")),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Crear contexto con timeout corto para inicialización
	initCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.ExporterTimeout)*time.Millisecond)
	defer cancel()

	// Configurar exporter OTLP/gRPC con opciones fail-open
	exporter, err := otlptracegrpc.New(initCtx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(), // Cambiar a WithTLSCredentials en producción
		otlptracegrpc.WithDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
		otlptracegrpc.WithTimeout(time.Duration(cfg.ExporterTimeout)*time.Millisecond),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Crear TracerProvider con BatchSpanProcessor (asíncrono)
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(500*time.Millisecond),
			sdktrace.WithMaxExportBatchSize(512),
			sdktrace.WithMaxQueueSize(2048),
		),
		sdktrace.WithSampler(createSampler(cfg.SamplingRate)),
	)

	// Registrar TracerProvider globalmente
	otel.SetTracerProvider(tracerProvider)

	// Configurar propagadores de contexto
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	if cfg.LogLevel == "debug" {
		fmt.Fprintln(os.Stderr, "INFO: OpenTelemetry initialized successfully")
	}

	return nil
}

func shutdown(ctx context.Context) error {
	if tracerProvider == nil {
		return nil
	}

	// Timeout corto para evitar bloqueo en shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	return tracerProvider.Shutdown(shutdownCtx)
}

func createSampler(rate float64) sdktrace.Sampler {
	if rate >= 1.0 {
		return sdktrace.AlwaysSample()
	}
	if rate <= 0.0 {
		return sdktrace.NeverSample()
	}
	return sdktrace.TraceIDRatioBased(rate)
}
