# Goal

Rename the API/module identity from an old name to `able-rest-api` consistently across code, config, scripts, and docs.

## Scope

- Go module path
- import paths
- application name in config and scripts
- README references
- prompt and agent docs that still use the old name

## Constraints

- update all references consistently
- avoid unrelated refactors
- verify no stale references remain outside ignored/generated areas

## Read First

- `go.mod`
- `Makefile`
- `configs/app.example.yaml`
- `README.md`

## Verify

- search for the old name outside `.git` and `vendor`
- `go test ./...`

## Final Output

- renamed locations
- verification command/result
- files changed
