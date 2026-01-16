package test

import (
	"context"
	"os"
	"testing"

	"github.com/Apkahym/aws_otel_lawer/internal/otel"
)

func TestOTELInitIdempotent(t *testing.T) {
	// Setup
	t.Setenv("OTEL_SERVICE_NAME", "test-service")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "100")

	ctx := context.Background()

	// Primera inicialización
	shutdown1, err1 := otel.Initialize(ctx)
	if err1 != nil {
		t.Logf("First init failed (expected in test env): %v", err1)
	}

	// Segunda inicialización (debe ser noop)
	shutdown2, err2 := otel.Initialize(ctx)
	if err2 != nil && err1 != nil {
		// Ambos fallan es OK (sin collector)
		if err2.Error() != err1.Error() {
			t.Errorf("Expected same error on idempotent call, got different: %v vs %v", err1, err2)
		}
	}

	// Cleanup
	if shutdown1 != nil {
		_ = shutdown1(ctx)
	}
	if shutdown2 != nil {
		_ = shutdown2(ctx)
	}
}

func TestFailOpenWithInvalidEndpoint(t *testing.T) {
	// Setup con endpoint inválido
	t.Setenv("OTEL_SERVICE_NAME", "test-service")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "invalid-host:9999")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "100")

	ctx := context.Background()

	// La inicialización debe fallar pero no hacer panic
	shutdown, err := otel.Initialize(ctx)

	if err == nil {
		t.Log("OTEL init succeeded unexpectedly (collector might be running)")
	} else {
		t.Logf("OTEL init failed as expected: %v", err)
	}

	// Debe retornar un shutdown válido (noop)
	if shutdown == nil {
		t.Error("Expected non-nil shutdown function")
	}

	// Shutdown no debe hacer panic
	if shutdown != nil {
		if err := shutdown(ctx); err != nil {
			t.Logf("Shutdown returned error (expected): %v", err)
		}
	}
}

func TestConfigLoadFromEnv(t *testing.T) {
	// Setup
	t.Setenv("OTEL_SERVICE_NAME", "my-service")
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "collector:4317")
	t.Setenv("OTEL_EXPORTER_OTLP_TIMEOUT", "3000")
	t.Setenv("OTEL_LOG_LEVEL", "debug")
	t.Setenv("OTEL_SAMPLING_RATE", "0.5")

	cfg := otel.LoadConfig()

	// Verificar
	if cfg.ServiceName != "my-service" {
		t.Errorf("Expected service name 'my-service', got '%s'", cfg.ServiceName)
	}
	if cfg.OTLPEndpoint != "collector:4317" {
		t.Errorf("Expected endpoint 'collector:4317', got '%s'", cfg.OTLPEndpoint)
	}
	if cfg.ExporterTimeout != 3000 {
		t.Errorf("Expected timeout 3000, got %d", cfg.ExporterTimeout)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("Expected log level 'debug', got '%s'", cfg.LogLevel)
	}
	if cfg.SamplingRate != 0.5 {
		t.Errorf("Expected sampling rate 0.5, got %f", cfg.SamplingRate)
	}
}

func TestConfigDefaults(t *testing.T) {
	// Limpiar env vars
	os.Clearenv()

	cfg := otel.LoadConfig()

	// Verificar defaults
	if cfg.ServiceName != "unknown-service" {
		t.Errorf("Expected default service name 'unknown-service', got '%s'", cfg.ServiceName)
	}
	if cfg.ExporterTimeout != 5000 {
		t.Errorf("Expected default timeout 5000, got %d", cfg.ExporterTimeout)
	}
	if cfg.SamplingRate != 1.0 {
		t.Errorf("Expected default sampling rate 1.0, got %f", cfg.SamplingRate)
	}
}
