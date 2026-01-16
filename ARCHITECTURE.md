# Architecture Documentation

## 📐 Visión General

La **AWS Lambda OTEL Layer** es una solución de instrumentación transparente que añade observabilidad distribuida a funciones AWS Lambda sin modificar el código de la aplicación.

## 🎯 Principios de Diseño

### 1. Fail-Open (No Bloqueante)

El sistema está diseñado para **nunca** interrumpir la ejecución del handler original:

```
┌─────────────────────────────────────────┐
│ OTEL Initialization                     │
│ ┌─────────────┐                         │
│ │ Try Init    │──────┐                  │
│ └─────────────┘      │                  │
│                      ▼                  │
│              ┌───────────────┐          │
│              │ Success?      │          │
│              └───┬───────┬───┘          │
│                  │       │              │
│             YES  │       │ NO           │
│                  ▼       ▼              │
│          ┌────────┐  ┌────────────┐    │
│          │ Use    │  │ Log Warn + │    │
│          │ OTEL   │  │ Continue   │    │
│          └────────┘  └────────────┘    │
│                  │       │              │
│                  └───┬───┘              │
│                      ▼                  │
│            Continue Execution           │
└─────────────────────────────────────────┘
```

### 2. Opt-In (Kill-Switch)

Control explícito del comportamiento:

```mermaid
graph TD
    A[Lambda Start] --> B{OBS_ENABLED == "1"?}
    B -->|NO| C[Passthrough Mode]
    B -->|YES| D[Instrumented Mode]
    C --> E[Execute Original Handler]
    D --> F[Initialize OTEL]
    F --> G[Create Instrumented Handler]
    G --> H[Execute with Tracing]
    E --> I[Return Response]
    H --> I
```

### 3. Idempotencia

La inicialización usa `sync.Once` para garantizar:
- Una sola inicialización por proceso Lambda
- Reutilización del TracerProvider en invocaciones warm
- Comportamiento predecible en concurrencia

```go
var (
    initOnce       sync.Once
    initErr        error
    tracerProvider *sdktrace.TracerProvider
)

func Initialize(ctx context.Context) (ShutdownFunc, error) {
    initOnce.Do(func() {
        initErr = doInitialize(ctx)
    })
    // ...
}
```

## 🏗️ Componentes

### 1. Wrapper Entrypoint (`cmd/wrapper/main.go`)

**Responsabilidades:**
- Verificar kill-switch (`OBS_ENABLED`)
- Inicializar OpenTelemetry (fail-open)
- Crear handler apropiado (instrumented vs passthrough)
- Gestionar lifecycle (startup + shutdown)

**Flujo de Decisión:**

```
START
  │
  ├─► Check OBS_ENABLED
  │    │
  │    ├─► != "1" ──► Create PassthroughHandler ──► lambda.Start()
  │    │
  │    └─► == "1" ──► Initialize OTEL (fail-open)
  │                    │
  │                    ├─► Success: Use OTEL
  │                    └─► Failure: Continue without OTEL
  │                         │
  │                         └─► Create InstrumentedHandler ──► lambda.Start()
  │
END
```

### 2. OTEL Initialization (`internal/otel/init.go`)

**Características:**
- **Timeout corto** en conexión a collector (configurable, default 5s)
- **Batch export asíncrono** (no agrega latencia)
- **Context propagation** (W3C TraceContext + Baggage)
- **Sampling configurable** (AlwaysSample, NeverSample, TraceIDRatioBased)

**Componentes OpenTelemetry:**

```
┌─────────────────────────────────────────────────────┐
│ TracerProvider                                      │
│ ┌─────────────────────────────────────────────────┐ │
│ │ Resource (Service Name, Lambda Metadata)        │ │
│ ├─────────────────────────────────────────────────┤ │
│ │ Sampler (Rate-based or Always/Never)            │ │
│ ├─────────────────────────────────────────────────┤ │
│ │ BatchSpanProcessor                              │ │
│ │ ┌─────────────────────────────────────────────┐ │ │
│ │ │ OTLP gRPC Exporter                          │ │ │
│ │ │ - Endpoint: configurable                    │ │ │
│ │ │ - Timeout: 5s default                       │ │ │
│ │ │ - Insecure: for dev (TLS in prod)           │ │ │
│ │ └─────────────────────────────────────────────┘ │ │
│ │ - BatchTimeout: 500ms                           │ │
│ │ - MaxExportBatchSize: 512                       │ │
│ │ - MaxQueueSize: 2048                            │ │
│ └─────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

### 3. Configuration (`internal/otel/config.go`)

Carga configuración desde environment variables con defaults sensibles:

| Config Field | Env Var | Default | Validación |
|--------------|---------|---------|------------|
| `ServiceName` | `OTEL_SERVICE_NAME` | `"unknown-service"` | String no vacío |
| `OTLPEndpoint` | `OTEL_EXPORTER_OTLP_ENDPOINT` | `"localhost:4317"` | Host:Port |
| `ExporterTimeout` | `OTEL_EXPORTER_OTLP_TIMEOUT` | `5000` | > 0 ms |
| `LogLevel` | `OTEL_LOG_LEVEL` | `"error"` | debug/info/error |
| `SamplingRate` | `OTEL_SAMPLING_RATE` | `1.0` | 0.0 - 1.0 |

### 4. Instrumented Handler (`internal/invoke/handler.go`)

**Responsabilidades:**
- Crear span raíz por invocación
- Añadir metadata de AWS Lambda como atributos
- Medir duración de ejecución
- Recuperar de panics (fail-open)
- Registrar errores en spans

**Span Attributes:**

```yaml
span:
  name: "lambda.invoke"
  kind: SERVER
  attributes:
    - faas.execution: "${_X_AMZN_TRACE_ID}"
    - faas.handler: "${ORIGINAL_HANDLER}"
    - faas.name: "${AWS_LAMBDA_FUNCTION_NAME}"
    - faas.version: "${AWS_LAMBDA_FUNCTION_VERSION}"
    - cloud.provider: "aws"
    - cloud.region: "${AWS_REGION}"
    - lambda.duration_ms: <computed>
  status:
    - OK (success)
    - ERROR (handler error or panic)
```

**Panic Recovery Flow:**

```
┌──────────────────────────────────────┐
│ InstrumentedHandler.Invoke()         │
│ ┌──────────────────────────────────┐ │
│ │ defer func() {                   │ │
│ │   if r := recover() {            │ │
│ │     span.RecordError(panicErr)   │ │
│ │     span.SetStatus(ERROR)        │ │
│ │     log panic                    │ │
│ │     handlerErr = panicErr        │ │
│ │   }                              │ │
│ │ }()                              │ │
│ │                                  │ │
│ │ // Execute original handler      │ │
│ │ result, err = execute(...)       │ │
│ └──────────────────────────────────┘ │
│                                      │
│ // Handler sempre retorna (fail-open)│
└──────────────────────────────────────┘
```

### 5. Recovery Utilities (`internal/invoke/recovery.go`)

Funciones auxiliares para recuperación de panics:

- `RecoverPanic()`: Recupera panic y registra stack trace
- `SafeExecute(fn)`: Wrapper genérico con recuperación

## 📊 Data Flow

### Span Export Pipeline

```
┌─────────────────────────────────────────────────────────────┐
│ Lambda Invocation                                           │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ Tracer.Start()        │
         │ Create Span           │
         └───────────┬───────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ Execute Handler       │
         │ (with span context)   │
         └───────────┬───────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ Span.End()            │
         └───────────┬───────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │ BatchSpanProcessor        │
         │ - Queue span              │
         │ - Batch if threshold met  │
         └───────────┬───────────────┘
                     │ (async)
                     ▼
         ┌───────────────────────────┐
         │ OTLP Exporter             │
         │ - Serialize to protobuf   │
         │ - Send via gRPC           │
         └───────────┬───────────────┘
                     │
                     ▼
         ┌───────────────────────────┐
         │ OTEL Collector            │
         │ (external)                │
         └───────────────────────────┘
```

**Notas importantes:**
- Export es **asíncrono** (no bloquea handler)
- Batch timeout: 500ms (flush automático)
- Shutdown con timeout: 2s (evita bloqueo en termination)

## 🔄 Lambda Lifecycle

### Cold Start

```
┌──────────────────────────────────────────────────────────┐
│ Lambda Container Init                                    │
├──────────────────────────────────────────────────────────┤
│ 1. Load wrapper binary                                   │
│ 2. Parse environment variables                           │
│ 3. Check OBS_ENABLED                                     │
│ 4. Initialize OTEL (if enabled)                          │
│    - Connect to collector (with timeout)                 │
│    - Create TracerProvider                               │
│    - Register propagators                                │
│ 5. Create handler                                        │
│ 6. Register with Lambda Runtime API                      │
└────────────────────┬─────────────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ Wait for Invocation   │
         └───────────────────────┘
```

### Warm Invocation

```
┌──────────────────────────────────────────────────────────┐
│ Lambda Invocation (Warm)                                 │
├──────────────────────────────────────────────────────────┤
│ 1. Reuse existing TracerProvider (idempotent)            │
│ 2. Create new span for invocation                        │
│ 3. Execute handler with tracing                          │
│ 4. End span (queued for batch export)                    │
│ 5. Return response immediately                           │
│                                                          │
│ Background:                                              │
│ - BatchProcessor flushes periodically                    │
└──────────────────────────────────────────────────────────┘
```

### Shutdown

```
┌──────────────────────────────────────────────────────────┐
│ Lambda Container Termination                             │
├──────────────────────────────────────────────────────────┤
│ 1. Receive SIGTERM from Lambda                           │
│ 2. Call shutdown() with 2s timeout                       │
│    - Flush pending spans                                 │
│    - Close exporter connections                          │
│    - Best-effort (fail-open if timeout)                  │
│ 3. Exit process                                          │
└──────────────────────────────────────────────────────────┘
```

## ⚡ Performance Considerations

### Latencia Agregada

| Fase | Overhead Estimado | Notas |
|------|-------------------|-------|
| **Cold Start** | +100-300ms | Inicialización OTEL (una vez) |
| **Warm Start** | < 1ms | Creación de span (síncrono) |
| **Export** | 0ms | Asíncrono, no bloquea |
| **Shutdown** | +50-200ms | Flush final (best-effort) |

**Total overhead típico: < 1ms por invocación warm**

### Memoria

- Baseline (sin OTEL): ~20MB
- Con OTEL: ~25-30MB
- Span buffer: ~2-5MB (2048 spans máx)

### Optimizaciones

1. **Batch Processing**: Reduce overhead de red
2. **Async Export**: No bloquea handler
3. **Timeout corto**: Evita bloqueos prolongados
4. **Sampling**: Reduce volumen en producción

## 🛠️ Build & Deployment

### Multi-Architecture Build

```bash
# AMD64 (x86_64)
GOOS=linux GOARCH=amd64 go build -o dist/otel-wrapper-amd64 ./cmd/wrapper

# ARM64 (Graviton2)
GOOS=linux GOARCH=arm64 go build -o dist/otel-wrapper-arm64 ./cmd/wrapper
```

### Layer Structure

```
layer/
└── bin/
    └── otel-wrapper     # Binary ejecutable

# Lambda busca binarios en:
# 1. /opt/bin/
# 2. /opt/
# 3. Runtime default path
```

### Deployment Options

**1. Lambda Layer:**
```bash
aws lambda publish-layer-version \
  --layer-name otel-observability \
  --zip-file fileb://otel-layer-amd64.zip
```

**2. Container Image:**
```dockerfile
COPY otel-wrapper /usr/local/bin/
ENV _HANDLER=bin/otel-wrapper
```

## 🧪 Testing Strategy

### Unit Tests

- `TestOTELInitIdempotent`: Verifica inicialización única
- `TestFailOpenWithInvalidEndpoint`: Valida fail-open
- `TestConfigLoadFromEnv`: Comprueba parsing de config
- `TestInstrumentedHandlerCreation`: Valida construcción de handler

### Integration Tests

- Mock OTEL Collector
- Validar propagación de contexto
- Verificar atributos de span
- Comprobar panic recovery

### E2E Tests (Manual)

1. Deploy Lambda con layer
2. Invocar y verificar traces en backend
3. Simular fallos de collector
4. Verificar logs de fail-open

## 🔐 Security Considerations

### 1. Secrets Management

❌ **NO** hardcodear endpoints o credenciales  
✅ Usar AWS Secrets Manager o Parameter Store

### 2. Network Security

- Usar TLS en producción (`WithTLSCredentials`)
- Restringir egress a collector endpoint
- Validar certificados de collector

### 3. Data Privacy

- Sanitizar payloads antes de span attributes
- Aplicar sampling para datos sensibles
- Revisar logs de stderr (pueden contener metadata)

## 📈 Monitoring the Monitor

### Wrapper Metrics (via Logs)

```bash
# Inicialización exitosa
INFO: OpenTelemetry initialized successfully

# Fail-open activado
WARN: OTEL initialization failed (continuing without observability): <error>

# Panic recuperado
PANIC in handler: <panic_value>
Stack trace: <stack>
```

### Collector Metrics

Monitorear:
- `otelcol_receiver_accepted_spans`
- `otelcol_receiver_refused_spans`
- `otelcol_exporter_sent_spans`

## 🎓 Best Practices

### 1. Configuración

```bash
# Desarrollo
OBS_ENABLED=1
OTEL_LOG_LEVEL=debug
OTEL_SAMPLING_RATE=1.0

# Producción
OBS_ENABLED=1
OTEL_LOG_LEVEL=error
OTEL_SAMPLING_RATE=0.1  # Sample 10%
```

### 2. Troubleshooting

**Síntoma:** No aparecen traces  
**Solución:**
1. Verificar `OBS_ENABLED=1`
2. Comprobar conectividad a collector
3. Revisar logs de Lambda (CloudWatch)
4. Validar sampling rate

**Síntoma:** Latencia aumentada  
**Solución:**
1. Verificar export asíncrono
2. Ajustar batch size/timeout
3. Aumentar sampling (reducir volumen)

### 3. Rollback Plan

En caso de problemas:
1. Set `OBS_ENABLED=0` (rollback inmediato)
2. O remover layer de Lambda
3. O revertir a versión anterior de layer

---

## 📚 Referencias

- [OpenTelemetry Specification](https://opentelemetry.io/docs/specs/otel/)
- [AWS Lambda Execution Environment](https://docs.aws.amazon.com/lambda/latest/dg/lambda-runtime-environment.html)
- [OTLP Protocol](https://github.com/open-telemetry/opentelemetry-proto)
- [Go OpenTelemetry SDK](https://pkg.go.dev/go.opentelemetry.io/otel)

---

**Última actualización:** 2026-01-16  
**Versión:** 1.0.0
