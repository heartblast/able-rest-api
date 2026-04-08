# Goal

Initialize or regenerate this repository as a maintainable Go REST API template while preserving the current architectural direction.

## Scope

- project skeleton and wiring
- REST API setup
- config and secret handling
- DB vendor-aware persistence structure
- Swagger support
- build and developer workflow files

## Constraints

- read the current codebase before editing
- implement, do not only describe
- preserve the layered structure already used here
- avoid framework-heavy abstractions
- keep Oracle-specific concerns isolated from the default build

## Read First

- `cmd/server/main.go`
- `internal/infra/config/config.go`
- `internal/delivery/http/router/router.go`
- `README.md`

## Deliverables

- directory structure
- core Go files and wiring
- sample config
- migrations
- build scripts
- README updates

## Verify

- `gofmt`
- `go test ./...`
- `go build ./...`

## Final Output

- implemented work
- architectural decisions
- modified files
- run/test instructions
- remaining limitations
