# AWS Lambda OpenTelemetry Layer

[![CI/CD Pipeline](https://github.com/Apkahym/aws_otel_lawer/actions/workflows/ci.yaml/badge.svg)](https://github.com/Apkahym/aws_otel_lawer/actions/workflows/ci.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/Apkahym/aws_otel_lawer)](https://goreportcard.com/report/github.com/Apkahym/aws_otel_lawer)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/Apkahym/aws_otel_lawer)](https://github.com/Apkahym/aws_otel_lawer/releases/latest)

Una **AWS Lambda Layer** completa escrita en **Go** que provee instrumentación **OpenTelemetry** con modelo **fail-open, no bloqueante y opt-in**.

## 🎯 Características

- ✅ **Fail-Open**: Nunca bloquea la ejecución de tu Lambda
- ✅ **Opt-In**: Control total con kill-switch (`OBS_ENABLED`)
- ✅ **Idempotente**: Inicialización segura y predecible
- ✅ **Panic Recovery**: Recuperación automática de errores críticos
- ✅ **Multi-Arquitectura**: Soporta `amd64` y `arm64`
- ✅ **Configuración Flexible**: 100% vía variables de entorno
- ✅ **Async Export**: No agrega latencia perceptible

## 📦 Instalación

### Opción 1: Usar Layer Pre-Compilado

1. Descarga el layer desde [Releases](https://github.com/Apkahym/aws_otel_lawer/releases)
2. Publica como Lambda Layer en tu cuenta AWS:

```bash
aws lambda publish-layer-version \
  --layer-name otel-observability \
  --description "OpenTelemetry Observability Layer" \
  --zip-file fileb://otel-layer-amd64.zip \
  --compatible-runtimes provided.al2 \
  --compatible-architectures x86_64
```

### Opción 2: Build desde Fuente

```bash
git clone https://github.com/Apkahym/aws_otel_lawer.git
cd aws_otel_lawer
go mod download
make build
make package
```

## 🚀 Uso

### 1. Configurar Lambda Function

Añade las siguientes variables de entorno a tu función Lambda:

```bash
# REQUERIDO: Kill-switch para habilitar observabilidad
OBS_ENABLED=1

# REQUERIDO: Handler original de tu función
ORIGINAL_HANDLER=index.handler

# REQUERIDO: Configuración OTEL
OTEL_SERVICE_NAME=mi-servicio-lambda
OTEL_EXPORTER_OTLP_ENDPOINT=collector.example.com:4317

# OPCIONAL: Configuración avanzada
OTEL_EXPORTER_OTLP_TIMEOUT=5000
OTEL_LOG_LEVEL=error
OTEL_SAMPLING_RATE=1.0
```

### 2. Configurar Handler

Cambia el handler de tu Lambda para apuntar al wrapper:

```
Handler: bin/otel-wrapper
```

### 3. Añadir Layer

Asocia el layer a tu función:

```bash
aws lambda update-function-configuration \
  --function-name mi-funcion \
  --layers arn:aws:lambda:us-east-1:123456789012:layer:otel-observability:1
```

## ⚙️ Configuración

### Variables de Entorno

| Variable | Requerido | Default | Descripción |
|----------|-----------|---------|-------------|
| `OBS_ENABLED` | ✅ | - | Kill-switch: `1` = habilitado, otro valor = deshabilitado |
| `ORIGINAL_HANDLER` | ✅ | - | Handler original (ej: `index.handler`) |
| `OTEL_SERVICE_NAME` | ✅ | `unknown-service` | Nombre del servicio en traces |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | ✅ | `localhost:4317` | Endpoint del collector OTLP |
| `OTEL_EXPORTER_OTLP_TIMEOUT` | ❌ | `5000` | Timeout en ms para exporter |
| `OTEL_LOG_LEVEL` | ❌ | `error` | Nivel de log (`debug`, `info`, `error`) |
| `OTEL_SAMPLING_RATE` | ❌ | `1.0` | Tasa de muestreo (0.0 a 1.0) |

### Modo Bypass (Sin Observabilidad)

Para deshabilitar temporalmente la observabilidad sin modificar el layer:

```bash
export OBS_ENABLED=0
```

El wrapper ejecutará el handler original sin ninguna instrumentación.

## 🔍 Arquitectura

Consulta [ARCHITECTURE.md](./ARCHITECTURE.md) para detalles técnicos completos.

### Flujo de Ejecución

```
┌─────────────────────────────────────────────────────────────┐
│ Lambda Invocation                                           │
└────────────────────┬────────────────────────────────────────┘
                     │
                     ▼
         ┌───────────────────────┐
         │ OBS_ENABLED == "1"?   │
         └───────┬───────────────┘
                 │
        ┌────────┴────────┐
        │ NO              │ YES
        ▼                 ▼
┌─────────────┐   ┌──────────────────┐
│ Passthrough │   │ Initialize OTEL  │
│ Handler     │   │ (idempotent)     │
└─────────────┘   └────────┬─────────┘
                           │
                           ▼ (fail-open on error)
                  ┌─────────────────────┐
                  │ Create Root Span    │
                  │ + Metadata          │
                  └────────┬────────────┘
                           │
                           ▼
                  ┌─────────────────────┐
                  │ Execute Original    │
                  │ Handler (w/recovery)│
                  └────────┬────────────┘
                           │
                           ▼
                  ┌─────────────────────┐
                  │ Export Spans        │
                  │ (async batch)       │
                  └────────┬────────────┘
                           │
                           ▼
                     Return Result
```

## 🧪 Testing

```bash
# Tests unitarios
make test

# Linting
make lint

# Build completo
make build

# Pipeline completo
make all
```

## 📊 Observabilidad

### Spans Generados

Cada invocación Lambda genera un span con:

- **Nombre**: `lambda.invoke`
- **Tipo**: `SpanKindServer`
- **Atributos**:
  - `faas.execution`: X-Ray trace ID
  - `faas.handler`: Handler original
  - `faas.name`: Nombre de la función
  - `faas.version`: Versión de la función
  - `cloud.provider`: `aws`
  - `cloud.region`: Región AWS
  - `lambda.duration_ms`: Duración en milisegundos

### Propagación de Contexto

El layer propaga contexto usando:
- W3C Trace Context
- W3C Baggage

Esto permite trazabilidad end-to-end en arquitecturas distribuidas.

## 🛡️ Fail-Open Garantizado

El layer está diseñado para **nunca** interrumpir la ejecución de tu Lambda:

1. **Inicialización fallida**: Continúa sin observabilidad
2. **Export fallido**: Descarta spans silenciosamente
3. **Panic en handler**: Recupera y registra el error
4. **Timeout en shutdown**: Usa timeout corto (2s) para no bloquear

## 🤝 Contribución

Las contribuciones son bienvenidas. Por favor:

1. Fork el repositorio
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📝 Licencia

Distribuido bajo la licencia MIT. Ver [LICENSE](./LICENSE) para más información.

## 🙏 Agradecimientos

- [OpenTelemetry](https://opentelemetry.io/)
- [AWS Lambda Go SDK](https://github.com/aws/aws-lambda-go)

## 📞 Soporte

- 🐛 Reporta bugs en [Issues](https://github.com/Apkahym/aws_otel_lawer/issues)
- 💬 Discusiones en [Discussions](https://github.com/Apkahym/aws_otel_lawer/discussions)

---

**Hecho con ❤️ por [Apkahym](https://github.com/Apkahym)**
