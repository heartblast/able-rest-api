# Phase 7 — MCP·SDK 친화적 OpenAPI 품질 강화

기준 커밋:

```text
43fa04c
```

현재 OpenAPI 3.x 단일 계약, CI Gate, runtime schema/status/semantic/side-effect 검증까지 완료되어 있다.

이번 단계에서는 **Dynamic MCP, SDK Generator, 범용 API Client가 안정적으로 해석할 수 있도록 `docs/openapi.yaml`의 계약 품질을 강화**하라.

## 목표

현재 6개 operation을 점검하고 다음 항목을 보완한다.

- `operationId` 안정성 및 명명 일관성
- `summary`와 `description`
- parameter/requestBody 설명
- request/response schema의 `required`
- 실제 구현이 보장하는 `format`, `minimum`, `minLength`, `minItems`, `maxLength` 등
- 의미 있는 request/response example
- 공통 error response/schema 재사용
- error code가 가능한 경우 enum 또는 문서화된 값으로 명확히 표현
- nullable/optional 필드 의미 명확화

## MCP/SDK 관점 점검

각 operation이 자동 Tool/SDK 생성 시 다음 조건을 만족하는지 확인한다.

```text
operationId만으로 기능 식별 가능
입력 필드 의미가 description만으로 이해 가능
필수/선택 입력 구분 가능
response 구조가 명확함
오류 status와 error.code 의미 식별 가능
모호한 free-form object 사용 최소화
```

필요하면 `x-` 확장 필드를 검토할 수 있으나, 특정 MCP 구현에 종속되는 metadata는 이번 Phase에서 추가하지 않는다.

## 원칙

- 실제 구현이 보장하지 않는 제약을 OpenAPI에 추가하지 말 것
- URI, 비즈니스 동작, response envelope 변경 금지
- 기존 operationId는 특별한 이유가 없으면 변경하지 말 것
- 중복 schema는 `components`로 재사용
- OpenAPI를 설명문서가 아니라 **실행 가능한 API 계약**으로 유지
- 기존 runtime contract/side-effect 테스트 모두 유지

## 검증

```bash
go test ./...
go vet ./...
make openapi-check
```

추가로 모든 operation에 대해 최소 다음 metadata 존재 여부를 자동 검사하도록 검토한다.

```text
operationId
summary
description
tags
responses
```

requestBody/parameter가 있는 operation은 해당 description과 schema도 검사한다.

## 완료 보고

1. 보완한 OpenAPI metadata/schema
2. MCP·SDK 친화성 개선 내용
3. 추가한 자동 품질 검사
4. 변경 파일
5. 전체 검증 결과
6. Phase 8 최종 마무리 권고

`docs/reference_docs/`는 수정하지 말고, 검증 후 커밋하되 **tag/release는 생성하지 않는다.**