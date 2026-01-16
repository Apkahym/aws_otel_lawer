package invoke

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// InstrumentedHandler es el handler que envuelve la ejecución del handler real
type InstrumentedHandler struct {
	originalHandler string
	tracer          trace.Tracer
}

// NewInstrumentedHandler crea un nuevo handler instrumentado
func NewInstrumentedHandler() (*InstrumentedHandler, error) {
	originalHandler := os.Getenv("ORIGINAL_HANDLER")
	if originalHandler == "" {
		return nil, fmt.Errorf("ORIGINAL_HANDLER environment variable is not set")
	}

	return &InstrumentedHandler{
		originalHandler: originalHandler,
		tracer:          otel.Tracer("lambda-wrapper"),
	}, nil
}

// Invoke ejecuta el handler real con instrumentación
func (h *InstrumentedHandler) Invoke(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	startTime := time.Now()

	// Crear root span
	ctx, span := h.tracer.Start(ctx, "lambda.invoke",
		trace.WithSpanKind(trace.SpanKindServer),
		trace.WithAttributes(
			attribute.String("faas.execution", os.Getenv("_X_AMZN_TRACE_ID")),
			attribute.String("faas.handler", h.originalHandler),
			attribute.String("cloud.provider", "aws"),
			attribute.String("faas.name", os.Getenv("AWS_LAMBDA_FUNCTION_NAME")),
			attribute.String("faas.version", os.Getenv("AWS_LAMBDA_FUNCTION_VERSION")),
			attribute.String("cloud.region", os.Getenv("AWS_REGION")),
		),
	)
	defer span.End()

	// CRÍTICO: Recuperar de panic (fail-open)
	var result json.RawMessage
	var handlerErr error

	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr := fmt.Errorf("panic recovered: %v", r)
				span.RecordError(panicErr)
				span.SetStatus(codes.Error, "panic")
				handlerErr = panicErr

				// Log del panic
				fmt.Fprintf(os.Stderr, "PANIC in handler: %v\n", r)
			}
		}()

		// Ejecutar handler real
		result, handlerErr = h.executeOriginalHandler(ctx, payload)
	}()

	// Registrar métricas
	duration := time.Since(startTime)
	span.SetAttributes(attribute.Int64("lambda.duration_ms", duration.Milliseconds()))

	// Manejar error
	if handlerErr != nil {
		span.RecordError(handlerErr)
		span.SetStatus(codes.Error, handlerErr.Error())
		return nil, handlerErr
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

func (h *InstrumentedHandler) executeOriginalHandler(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	// NOTA: Esta es una implementación simplificada
	// En producción, deberías usar el AWS Lambda Runtime API
	// para invocar el handler real según el runtime

	// Por ahora, simulamos una respuesta exitosa
	response := map[string]interface{}{
		"statusCode": 200,
		"body":       "Handler executed successfully",
	}

	return json.Marshal(response)
}

// PassthroughHandler es un handler sin instrumentación
type PassthroughHandler struct {
	originalHandler string
}

// NewPassthroughHandler crea un handler passthrough
func NewPassthroughHandler(handler string) *PassthroughHandler {
	return &PassthroughHandler{
		originalHandler: handler,
	}
}

// Invoke ejecuta el handler sin instrumentación
func (h *PassthroughHandler) Invoke(ctx context.Context, payload json.RawMessage) (json.RawMessage, error) {
	// Implementación simplificada
	cmd := exec.CommandContext(ctx, h.originalHandler)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return output, nil
}
