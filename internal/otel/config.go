package otel

import (
	"os"
	"strconv"
)

// Config contiene la configuración de OpenTelemetry
type Config struct {
	ServiceName     string
	OTLPEndpoint    string
	ExporterTimeout int
	LogLevel        string
	SamplingRate    float64
}

// LoadConfig carga la configuración desde variables de entorno
func LoadConfig() Config {
	timeout := 5000 // default 5s
	if t := os.Getenv("OTEL_EXPORTER_OTLP_TIMEOUT"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			timeout = parsed
		}
	}

	samplingRate := 1.0 // default: sample everything
	if s := os.Getenv("OTEL_SAMPLING_RATE"); s != "" {
		if parsed, err := strconv.ParseFloat(s, 64); err == nil {
			samplingRate = parsed
		}
	}

	return Config{
		ServiceName:     getEnvOrDefault("OTEL_SERVICE_NAME", "unknown-service"),
		OTLPEndpoint:    getEnvOrDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317"),
		ExporterTimeout: timeout,
		LogLevel:        getEnvOrDefault("OTEL_LOG_LEVEL", "error"),
		SamplingRate:    samplingRate,
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
