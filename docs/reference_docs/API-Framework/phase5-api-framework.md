# Framework Phase 5 — 영속 예약 메일·재시도·중복 방지

기준 커밋:

```text
e08232a
```

현재 `MailDispatchJob → Dispatch 계약 → MailService → MailSender → SMTP` 흐름까지 연결되어 있다.

이번 단계에서는 반복 실행 방식에서 확장해 **개별 메일을 영속적으로 1회 예약하고, 실패 재시도와 프로세스 간 중복 발송을 방지하는 구조**를 구현하라.

## 목표

최소 다음 상태를 관리한다.

```text
PENDING
PROCESSING
SENT
RETRY
FAILED
```

예약 메일은 최소 다음 정보를 저장한다.

```text
id
scheduled_at
payload
status
attempt_count
next_retry_at
last_error
created_at
updated_at
```

## 핵심 요구

- 예약 시간이 된 메일만 claim하여 발송
- 성공 시 `SENT`
- 일시적 실패는 제한된 횟수로 재시도
- 영구 실패 또는 최대 재시도 초과는 `FAILED`
- 여러 scheduler 프로세스가 동시에 실행돼도 동일 메일이 중복 발송되지 않도록 DB 기반 claim/lock 적용
- 프로세스가 발송 중 종료돼 `PROCESSING`에 고착되는 경우 복구 가능한 방식 검토
- transaction 범위와 SMTP 호출 경계를 명확히 설계

DB vendor별 차이는 기존 abstraction을 유지하고, 과도한 분산락/메시지큐 시스템은 도입하지 않는다.

## 의존성

```text
scheduler
  → scheduled-mail port
  → repository

scheduler
  → mail Dispatch interface
  → mail service
  → SMTP
```

scheduler가 SMTP 구현이나 mail HTTP 계층에 직접 의존하지 않게 유지한다.

## 테스트

최소 검증:

- 미래 예약은 발송하지 않음
- due mail은 정확히 1회 발송
- 성공 후 재실행해도 재발송 없음
- SMTP 실패 후 retry 상태/횟수/다음 실행시간 기록
- 최대 재시도 초과 → FAILED
- 동시 worker 2개가 동일 예약을 claim하지 못함
- PROCESSING 고착 복구 정책 검증
- DB 오류 시 fail-closed

가능하면 race/concurrency 테스트도 추가한다.

## 호환성

기존 즉시 메일 API, OpenAPI, URI, response envelope, 기존 scheduler 기능은 유지한다.

필요한 migration은 현재 지원 DB 정책에 맞춰 추가한다.

## 검증

```bash
go test ./...
go vet ./...
go build ./...
make openapi-check
git diff --check
```

## 완료 보고

1. 예약 메일 상태 모델
2. claim/중복 방지 방식
3. retry 및 복구 정책
4. DB/migration 변경
5. 동시성 테스트 결과
6. 전체 검증 결과
7. Framework Phase 6 최종 마무리 권고

`docs/reference_docs/`는 수정하지 않는다.

검증 후 커밋하되 **tag/release는 생성하지 않는다.**