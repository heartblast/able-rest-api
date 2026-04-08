# Goal

Regenerate and verify Swagger artifacts after API route or DTO changes.

## Scope

- `docs/docs.go`
- `docs/swagger.json`
- `docs/swagger.yaml`

## Constraints

- keep generated artifacts aligned with the current handlers and annotations
- do not hand-edit generated files unless necessary and intentional

## Read First

- `cmd/server/main.go`
- relevant handler and DTO files
- `Makefile`

## Verify

- `make swag` or `swag init -g cmd/server/main.go -o docs`
- spot-check affected schemas and routes

## Final Output

- generation command used
- files updated
- any gaps or generation blockers
