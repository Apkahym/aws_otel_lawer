package test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Apkahym/aws_otel_lawer/internal/invoke"
)

func TestInstrumentedHandlerCreation(t *testing.T) {
	// Sin ORIGINAL_HANDLER debe fallar
	handler, err := invoke.NewInstrumentedHandler()
	if err == nil {
		t.Error("Expected error when ORIGINAL_HANDLER not set")
	}
	if handler != nil {
		t.Error("Expected nil handler when creation fails")
	}

	// Con ORIGINAL_HANDLER debe funcionar
	t.Setenv("ORIGINAL_HANDLER", "index.handler")
	handler, err = invoke.NewInstrumentedHandler()
	if err != nil {
		t.Errorf("Unexpected error with ORIGINAL_HANDLER set: %v", err)
	}
	if handler == nil {
		t.Error("Expected non-nil handler")
	}
}

func TestPassthroughHandler(t *testing.T) {
	handler := invoke.NewPassthroughHandler("echo")
	if handler == nil {
		t.Fatal("Expected non-nil passthrough handler")
	}

	// El passthrough debe poder invocarse (aunque falle sin setup real)
	ctx := context.Background()
	payload := json.RawMessage(`{"test": "data"}`)

	// Esta invocación fallará en test sin Lambda runtime real, pero no debe panic
	_, err := handler.Invoke(ctx, payload)
	t.Logf("Passthrough invoke result (expected to fail in test): %v", err)
}
