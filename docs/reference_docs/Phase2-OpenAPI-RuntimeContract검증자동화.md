# Phase 2 — OpenAPI Runtime Contract 검증 자동화

Phase 1에서 구축한 `docs/openapi.yaml` 단일 계약을 기준으로, **실제 HTTP 실행 결과와 OpenAPI 스펙이 지속적으로 일치하는지 자동 검증하는 통합 테스트**를 구현하라.

기준 커밋:

```text
d8c9551 docs: serve validated OpenAPI 3 contract
```

## 목표

현재 6개 API에 대해 실제 요청을 실행하고 다음 항목을 OpenAPI 계약과 자동 대조한다.

```text
GET  /health
GET  /ready
GET  /api/v1/users
POST /api/v1/users
GET  /api/v1/users/{id}
POST /api/v1/mail/send
```

최소 검증 항목:

- 실제 HTTP status가 OpenAPI `responses`에 정의되어 있는지
- `Content-Type` 일치 여부
- 성공/실패 JSON 응답이 해당 response schema를 만족하는지
- path/query parameter 타입 및 required 조건
- request body schema 검증
- `request_id`, success/error envelope 구조
- 정의되지 않은 응답 형태 발생 시 테스트 실패

## 구현 방향

가능하면 OpenAPI 3 validator 라이브러리를 사용해 **실제 request/response를 계약에 직접 검증**하라.

기존의 정적 검사:

```text
OpenAPI 문법
$ref
operationId
route 존재 여부
DTO 주요 필드
```

는 유지하고, 이번 Phase에서는 그 위에 **runtime contract test**를 추가한다.

테스트는 외부 서비스 없이 재현 가능하게 구성하고 DB/SMTP 등 의존성은 기존 테스트 구조 또는 mock/fake를 활용한다.

## 추가 요구

CI 및 로컬에서 한 번에 검증할 수 있도록 다음과 같은 명령을 제공하라.

```bash
make openapi-check
```

가능하면 내부적으로 다음을 모두 수행하게 한다.

```text
OpenAPI 정적 validation
router ↔ spec drift 검사
runtime request/response contract 검사
```

OpenAPI 검증 실패 시 어느 API의 어떤 status/schema가 다른지 명확한 오류 메시지를 출력하라.

## 제약

- 기존 API 동작 변경 금지
- URI 및 response envelope 호환성 유지
- OpenAPI 문서를 테스트에 맞춰 임의 완화하지 말 것
- `docs/openapi.yaml`을 계속 Single Source of Truth로 유지
- 불필요한 대규모 리팩터링 금지
- `docs/reference_docs/`의 기존 미추적 파일은 건드리지 말 것

## 검증

최소 수행:

```bash
go test ./...
go vet ./...
make openapi-check
```

모두 통과한 뒤 변경사항을 커밋하라.

완료 보고에는 다음만 간략히 포함한다.

1. 추가한 runtime contract 검증 방식
2. 검증 대상 API
3. 발견·수정한 계약 불일치
4. 변경 파일
5. 검증 결과
6. Phase 3 권고사항

**tag/release는 생성하지 않는다.**