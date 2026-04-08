# Repository Conventions

## Code Changes

- Prefer minimal, targeted changes over broad refactors.
- Preserve existing package boundaries and naming style.
- Follow current dependency direction: delivery -> app -> domain, with infra implementing ports.

## Prompts

- One task file should describe one concrete outcome.
- Reference exact files to inspect first.
- State what is in scope and out of scope.
- Always include verification commands.
- Keep final response expectations short and explicit.

## Config

- When adding a config field, update:
  - YAML examples in `configs/`
  - the config struct
  - env override handling
  - validation/defaulting logic
  - documentation where relevant

## API Work

- Reuse existing response helpers and DTO style.
- Reflect API shape changes in Swagger artifacts.
- Keep handler logic thin and service-oriented.

## Logging and Secrets

- Never log passwords, raw secrets, or full sensitive payloads.
- Keep logs focused on identifiers, status, and timing.

## Validation and Verification

- Run `gofmt` on touched Go files.
- Prefer `go test ./...` when feasible.
- If tests cannot run, say why and what was still verified.
