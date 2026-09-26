# Framework Phase 4 — Scheduler ↔ Mail 모듈 연계 정리

기준 커밋:

```text
867cac4
```

현재 user/mail/scheduler가 각자의 model과 port를 소유하도록 분리되어 있다.

이번 단계에서는 **비어 있는 `MailDispatchJob`의 실제 예약 메일 실행 경로를 연결하면서 scheduler와 mail 모듈 간 의존성을 올바르게 설계**하라.

## 목표

예약 작업 실행 흐름을 분석하고 다음 구조를 만든다.

```text
Scheduler
   ↓
Job/Command interface
   ↓
Mail module service
   ↓
MailSender port
   ↓
SMTP infra
```

### 구현

- `MailDispatchJob`이 실제 예약 메일 발송을 수행하도록 연결
- scheduler가 mail 내부 구현체나 SMTP infra를 직접 import하지 않도록 할 것
- mail 서비스 또는 최소한의 application port를 통해 호출
- 의존성 조립은 `cmd/server`에서 수행
- 예약 실행 결과/오류가 기존 scheduler execution 기록과 정상 연결되는지 검증
- 동일 작업의 중복 실행이나 실패 처리 방식이 기존 정책과 충돌하지 않는지 확인

## 의존성 원칙

금지:

```text
scheduler → mail handler/DTO
scheduler → SMTP infra
mail → scheduler 내부 구현
```

허용되는 방향은 최소 계약을 통한 연계로 유지한다.

필요하다면 작은 interface를 정의하되, 단일 호출을 위해 과도한 abstraction/event framework를 만들지 않는다.

## 테스트

최소 다음을 검증한다.

- 예약 메일 정상 실행 → MailSender 정확히 1회 호출
- 전달된 수신자/제목/본문/첨부파일 일치
- MailSender 오류 → scheduler 실행 실패로 기록
- 비활성화/잘못된 입력 시 불필요한 SMTP 호출 없음
- scheduler와 mail 모듈 간 package cycle 없음

기존 HTTP mail API 및 runtime/side-effect contract test는 유지한다.

## 호환성

다음은 변경하지 않는다.

```text
URI
OpenAPI
operationId
JSON response envelope
기존 즉시 메일 발송 동작
DB schema
보안 정책
```

## 검증

```bash
go test ./...
go vet ./...
go build ./...
make openapi-check
git diff --check
```

## 완료 보고

1. 예약 메일 실행 구조
2. scheduler ↔ mail 의존 방식
3. 추가한 테스트
4. 발견·수정한 문제
5. 호환성 및 전체 검증 결과
6. Framework Phase 5 권고

`docs/reference_docs/`는 수정하지 않는다.

검증 후 커밋하되 **tag/release는 생성하지 않는다.**