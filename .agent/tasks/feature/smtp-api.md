# Goal

Add or adjust the SMTP mail sending API while preserving the repository's existing HTTP, service, domain, and infra separation.

## Scope

- `POST /api/v1/mail/send`
- request DTO validation
- service-level mail orchestration
- SMTP config and secret resolution
- Swagger updates

## Constraints

- inspect existing mail-related files first
- preserve current response format and routing conventions
- keep the handler dependent on the service only
- keep SMTP implementation in infra
- do not log SMTP passwords or full mail content

## Read First

- `cmd/server/main.go`
- `internal/delivery/http/router/router.go`
- `internal/delivery/http/handler/mail_handler.go`
- `internal/delivery/http/dto/mail.go`
- `internal/app/service/mail_service.go`
- `internal/domain/model/mail.go`
- `internal/domain/port/mail_sender.go`
- `internal/infra/mail/smtp_sender.go`
- `internal/infra/config/config.go`

## Verify

- `gofmt`
- `go test ./...`
- Swagger artifacts updated if DTOs or docs changed

## Final Output

- implementation summary
- decisions and tradeoffs
- modified files
- verification result
- remaining limits
