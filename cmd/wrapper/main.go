package main

import (
	"context"
	"fmt"
	"os"

	"github.com/Apkahym/aws_otel_lawer/internal/invoke"
	"github.com/Apkahym/aws_otel_lawer/internal/otel"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	// Kill-switch: verificar si observabilidad está habilitada
	if os.Getenv("OBS_ENABLED") != "1" {
		fmt.Fprintln(os.Stderr, "INFO: Observability disabled (OBS_ENABLED != 1)")
		originalHandler := os.Getenv("ORIGINAL_HANDLER")
		if originalHandler == "" {
			fmt.Fprintln(os.Stderr, "ERROR: ORIGINAL_HANDLER not set")
			os.Exit(1)
		}
		// Modo bypass: ejecutar handler sin instrumentación
		lambda.Start(invoke.NewPassthroughHandler(originalHandler))
		return
	}

	// Inicializar OpenTelemetry (idempotente, fail-open)
	ctx := context.Background()
	shutdown, err := otel.Initialize(ctx)
	if err != nil {
		// FAIL-OPEN: continuar sin observabilidad
		fmt.Fprintf(os.Stderr, "WARN: OTEL initialization failed (continuing without observability): %v\n", err)
	}

	// Registrar shutdown para flush antes de terminar
	if shutdown != nil {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*1e9) // 2 segundos
			defer cancel()
			if err := shutdown(shutdownCtx); err != nil {
				fmt.Fprintf(os.Stderr, "WARN: OTEL shutdown failed: %v\n", err)
			}
		}()
	}

	// Crear handler instrumentado
	handler, err := invoke.NewInstrumentedHandler()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Failed to create instrumented handler: %v\n", err)
		os.Exit(1)
	}

	// Iniciar Lambda Runtime
	lambda.Start(handler.Invoke)
}
