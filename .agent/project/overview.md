# Project Overview

## Summary

`able-rest-api` is a Go REST API template with layered boundaries and infrastructure that can vary by DB vendor. The repository already includes HTTP delivery, config loading with env overrides, secret resolution, Swagger generation, SMTP integration, and scheduler-related components.

## Primary Entry Points

- `cmd/server/main.go`: HTTP API bootstrap
- `cmd/migrate/main.go`: migration entry point
- `cmd/secretenc/main.go`: secret encryption utility
- `cmd/scheduler/main.go`: scheduler process bootstrap

## Core Layers

- `internal/delivery/http`: router, handlers, DTOs, middleware
- `internal/app/service`: application use cases
- `internal/domain/model`: domain entities and execution records
- `internal/domain/port`: ports for outbound behavior
- `internal/infra`: config, DB, mail, scheduler, security, persistence
- `internal/platform`: shared platform concerns such as logging

## Operational Characteristics

- Config is YAML-first with environment variable overrides.
- Secrets are resolved via `internal/infra/security`.
- Swagger docs are generated into `docs/`.
- DB vendor support is abstracted behind factory and vendor-specific packages.
- The API server and scheduler should remain separate processes.
