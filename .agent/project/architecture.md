# Architecture Notes

## Layering

The repository follows a pragmatic layered design:

- Delivery handles HTTP transport concerns only.
- App services orchestrate use cases and validation.
- Domain owns core models and outbound contracts.
- Infra implements ports and process-level mechanics.

## Wiring Pattern

`cmd/*/main.go` files assemble dependencies explicitly. Avoid hiding process wiring in global state or magic containers.

## Configuration Pattern

`internal/infra/config/config.go` is the source of truth for:

- YAML structure
- env override names
- config validation defaults
- secret resolution

Any prompt that changes runtime behavior should consider config schema, env overrides, defaults, and validation together.

## HTTP Pattern

`internal/delivery/http/router/router.go` wires middleware and routes.

- Middleware is centralized in the router.
- Handlers should depend on services, not infra implementations.
- Existing JSON response conventions should be preserved.

## Scheduler Pattern

Scheduler code lives outside the HTTP server process.

- Keep execution in `cmd/scheduler`.
- Put orchestration in `internal/infra/scheduler`.
- Keep job logic in `internal/app/service` and ports/models in `internal/domain`.

## Documentation Pattern

- Swagger artifacts belong in `docs/`.
- Agent-facing implementation prompts belong in `.agent/tasks/`.
- Stable repository guidance belongs in `.agent/project/`.
