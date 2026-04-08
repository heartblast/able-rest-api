# Goal

Extend the existing SMTP mail API to support JSON-based attachments without changing the existing behavior for messages with no attachments.

## Scope

- attachment DTO/model fields
- validation limits and base64 checks
- service-level normalization and validation
- MIME multipart message generation
- tests and Swagger sync

## Constraints

- preserve compatibility for non-attachment mail sends
- do not add filesystem-path upload behavior
- do not switch the endpoint to `multipart/form-data`
- do not log attachment payloads or base64 data
- keep changes minimal and consistent with current mail structure

## Read First

- `internal/delivery/http/handler/mail_handler.go`
- `internal/delivery/http/dto/mail.go`
- `internal/app/service/mail_service.go`
- `internal/app/service/mail_service_test.go`
- `internal/domain/model/mail.go`
- `internal/domain/port/mail_sender.go`
- `internal/infra/mail/smtp_sender.go`
- `docs/swagger.yaml`

## Suggested Limits

- max attachments: 5
- per attachment size: 5 MB
- total attachment size: 20 MB

## Verify

- `gofmt`
- `go test ./...`
- Swagger artifacts regenerated if request schema changed

## Final Output

- implementation summary
- validation and MIME decisions
- modified files
- verification result
- remaining limits
