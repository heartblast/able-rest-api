# Prompt Constraints

These constraints should be reused in most implementation prompts for this repository.

## Must Preserve

- existing layered structure
- explicit process wiring in `cmd/*`
- YAML + env override config model
- current response shape and routing style where applicable
- separation between API server and scheduler process

## Avoid

- unnecessary large-scale refactors
- new global state
- tight coupling between handler and infra packages
- logging secret values or raw confidential payloads
- silent behavior changes without config and docs updates

## Default Verification

- `gofmt`
- `go test ./...`
- Swagger sync when API DTOs or routes change
- config example updates when config schema changes
