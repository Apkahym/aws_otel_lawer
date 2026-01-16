# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.0] - 2026-01-16

### Added
- Initial implementation of AWS Lambda OpenTelemetry Observability Layer
- Go wrapper with fail-open guarantee
- OpenTelemetry SDK integration with idempotent initialization
- Instrumented handler with panic recovery
- Support for all Lambda runtimes (Node.js, Python, Java, .NET, Go)
- Multi-architecture build (amd64, arm64)
- CI/CD pipeline with GitHub Actions
- Comprehensive test suite (unit + integration)
- Complete documentation (README, ARCHITECTURE)
- Build and package scripts

### Features
- ✅ Fail-open absolute: observability never breaks Lambda execution
- ✅ Runtime-agnostic: works with any Lambda runtime
- ✅ Non-blocking: async span export
- ✅ Opt-in: enable with `OBS_ENABLED=1`
- ✅ Cold-start optimized: minimal overhead
- ✅ Idempotent initialization: single setup per container
- ✅ Panic recovery: mandatory capture without breaking execution

### Infrastructure
- Makefile for build automation
- Scripts for multi-arch compilation
- Lambda Layer packaging for AWS
- golangci-lint configuration
- GitHub Actions workflow

### Documentation
- README with installation guide and examples
- ARCHITECTURE.md with technical details and Mermaid diagrams
- Environment variables reference
- Troubleshooting guide

[0.1.0]: https://github.com/Apkahym/aws_otel_lawer/releases/tag/v0.1.0
