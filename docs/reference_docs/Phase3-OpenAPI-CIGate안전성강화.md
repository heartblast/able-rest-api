# Phase 3 — OpenAPI CI Gate 및 변경 안전성 강화

Phase 2 기준 커밋:

```text
4cf5b5e
```

현재 `docs/openapi.yaml` 단일 계약과 runtime contract test가 구축되어 있다.

이번 단계에서는 **OpenAPI 계약 검증을 CI 필수 품질 게이트로 만들고 API 변경 시 계약 누락을 자동 차단**하라.

## 목표

CI에서 최소 다음을 필수 수행한다.

```bash
go test ./...
go vet ./...
make openapi-check
```

OpenAPI 관련 변경뿐 아니라 router/handler/DTO 변경 시에도 계약 검증이 실행되도록 구성한다.

## 추가 구현

다음을 점검하고 필요한 범위만 구현하라.

- `docs/openapi.yaml` 문법/$ref 검증
- router ↔ OpenAPI route/operation drift 검증
- runtime request/response contract 검증
- `operationId` 중복 및 누락 검증
- 실제 API route에 대응하는 OpenAPI operation 누락 시 CI 실패
- OpenAPI에만 존재하고 실제 router에는 없는 operation도 CI 실패
- API 변경 후 계약 갱신을 빠뜨리기 어렵도록 개발자용 검증 명령과 README 정리

가능하면 CI 오류 메시지에 다음이 명확히 드러나게 한다.

```text
HTTP method
path
operationId
실패한 request/response/status/schema
```

## CI 원칙

- 기존 CI가 있다면 구조를 분석해 최소 변경
- 중복 workflow 생성 금지
- 로컬과 CI 검증 명령을 최대한 동일하게 유지
- OpenAPI 검증 실패를 warning으로 처리하지 말고 필수 실패 조건으로 설정
- 외부 네트워크나 실제 DB/SMTP에 의존하지 않을 것

## 계약 품질도 함께 점검

현재 6개 operation의 다음 항목을 간략히 점검하고 명백한 누락만 보완한다.

```text
operationId
summary
tags
required
parameter schema
requestBody
response status
error response
examples
```

단, API 기능 변경이나 대규모 OpenAPI 재설계는 하지 않는다.

## 검증

최소 다음이 모두 통과해야 한다.

```bash
go test ./...
go vet ./...
make openapi-check
```

가능하면 CI workflow 자체의 문법/실행 가능성도 검증한다.

완료 후 변경사항을 커밋하라.

## 완료 보고

다음만 간략히 보고한다.

1. CI에 추가한 OpenAPI Gate
2. drift 탐지 범위
3. 보완한 계약 품질 항목
4. 변경 파일
5. 전체 검증 결과
6. Phase 4 권고사항

`docs/reference_docs/`의 기존 미추적 파일은 건드리지 말고, **tag/release는 생성하지 않는다.**