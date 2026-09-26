# Framework Phase 6 — 예약 메일 멱등성 보완 및 최종 마무리

기준 커밋:

```text
9b03c9c
```

현재 영속 예약 메일, DB claim, retry, claim timeout, 동시 worker 중복 방지까지 구현되어 있다.

이번 단계에서는 **SMTP가 메일을 수락한 직후 프로세스가 종료되어 `SENT` 기록이 남지 않는 경우의 중복 발송 위험을 최소화하고 Framework 1차 개발을 최종 점검**하라.

## 핵심 목표

예약 메일마다 안정적인 `message_id` 또는 `idempotency_key`를 부여하고 동일 예약의 재시도에서도 값이 유지되도록 한다.

검토 및 구현:

- 예약 생성 시 고유하고 영속적인 message ID 생성
- 모든 retry/복구 발송에서 동일 ID 사용
- SMTP `Message-ID` header에 안정적으로 반영
- claim token과 message ID 역할을 혼동하지 말 것
- application 재시작 후에도 동일 예약은 동일 ID 유지
- 로그/실행 이력에서 예약 ID와 message ID 추적 가능
- 가능하면 downstream에서 중복 식별 가능한 구조 제공

SMTP 자체가 exactly-once를 보장하지 않는다는 점은 명확히 문서화한다.

## 장애 경계 테스트

최소 다음을 검증한다.

```text
SMTP 성공 → SENT 기록
SMTP 성공 직후 SENT 기록 실패 → 재claim 가능하지만 동일 Message-ID 유지
retry → 동일 Message-ID 유지
프로세스 재시작 → 동일 Message-ID 유지
다른 예약 → 서로 다른 Message-ID
```

실제 SMTP 서버에 exactly-once를 가정하거나 복잡한 분산 트랜잭션/2PC를 도입하지 않는다.

## 최종 Framework 점검

현재 구조를 전체적으로 점검해 다음을 확인한다.

- platform ↔ module ↔ infra 의존 방향
- user/mail/scheduler 모듈 독립성
- 신규 module 추가 절차
- OpenAPI/contract/security test 유지
- DB migration 절차
- scheduler 운영 방법
- 안전한 기본 설정
- 불필요하게 남은 dead code/구 구조

명백한 문제만 최소 수정하고 대규모 리팩터링은 하지 않는다.

## 문서 마무리

README에 최소 다음을 정리한다.

```text
Framework 구조
신규 module 추가 방법
OpenAPI 개발 절차
즉시/예약 메일 동작
retry/claim/idempotency 정책
migration 적용 방법
known limitation: SMTP exactly-once 미보장
```

## 검증

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
make openapi-check
git diff --check
```

가능하면 PostgreSQL 기존 통합 검증을 유지하고 MySQL은 실행 환경이 없으면 미검증 사실을 명확히 기록한다.

## 완료 보고

1. Message-ID/멱등성 설계
2. crash-window 대응 방식
3. 남아 있는 exactly-once 한계
4. Framework 구조 최종 점검 결과
5. 변경 파일
6. 전체 검증 결과
7. **Framework 1차 개발 완료 여부**

`docs/reference_docs/`는 수정하지 않는다.

검증 후 필요한 변경을 커밋하되 **tag/release는 생성하지 않는다.**