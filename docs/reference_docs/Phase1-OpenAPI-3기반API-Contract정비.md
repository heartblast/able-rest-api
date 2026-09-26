# Phase 1 — OpenAPI 3.x 기반 API Contract 정비

이 프로젝트를 분석하고 현재 Swagger 2.0 기반 API 문서를 **OpenAPI 3.0 이상을 기준으로 서비스할 수 있는 구조**로 개선하라.

## 목표

기존 API 동작을 변경하지 않고 다음 기반을 우선 구축한다.

1. 실제 `chi` 라우터/Handler/DTO와 현재 `docs/swagger.*` 간 불일치를 전수 점검
2. Swagger 2.0 의존 구조를 분석하고 OpenAPI 3.x 전환 방식을 결정
3. 현재 모든 HTTP API를 표현하는 OpenAPI 3.x 계약 작성
4. 서비스에서 최소한 다음 endpoint 제공
   - `GET /openapi.json`
   - 기존 Swagger UI는 가능하면 유지하되 OpenAPI 문서를 사용하도록 연결
5. 모든 operation에 안정적인 `operationId` 부여
6. request/response/path/query/status code/error schema를 실제 구현과 일치시킬 것

현재 확인된 API:

```text
GET  /health
GET  /ready
GET  /api/v1/users
POST /api/v1/users
GET  /api/v1/users/{id}
POST /api/v1/mail/send
```

## OpenAPI 작성 기준

가능하면 OpenAPI 문서를 **단일 Source of Truth**로 만들고 생성 산출물을 수동 편집하지 않는다.

각 operation에는 최소 다음을 정의한다.

```text
operationId
tags
summary
parameters
requestBody
responses
content: application/json
schema
required
examples
```

공통 오류 응답과 `request_id`, 성공 envelope도 components/schemas로 재사용하라.

Dynamic MCP/OpenAPI client에서 안정적으로 사용할 수 있도록 `operationId`는 의미 있고 중복되지 않게 설계한다.

예:

```text
getHealth
getReadiness
listUsers
getUser
createUser
sendMail
```

## 구현 원칙

- 기존 REST API URI 및 응답 호환성을 깨지 말 것
- 불필요한 프레임워크 교체 금지
- `chi` 유지
- Swagger 2.0과 OpenAPI 3.x를 이중으로 수동 관리하지 말 것
- 생성 도구를 도입한다면 재현 가능한 명령을 Makefile에 추가
- `docs/docs.go`, `swagger.json`, `swagger.yaml`의 향후 역할을 명확히 정리
- OpenAPI spec validation을 자동화할 수 있게 구성
- 기존 테스트는 모두 유지

## 검증

최소 다음을 수행한다.

```bash
go test ./...
go vet ./...
```

그리고 생성된 OpenAPI 문서가 유효한지 검증하고 다음을 확인한다.

```text
/openapi.json → HTTP 200
OpenAPI version → 3.x
6개 실제 API route 모두 존재
operationId 중복 없음
$ref 오류 없음
request/response schema가 실제 DTO와 일치
```

가능하면 router와 OpenAPI 간 drift를 탐지하는 테스트도 추가한다.

## 작업 방식

먼저 현재 구조를 분석해 **전환 전략과 문제점을 짧게 보고한 뒤 즉시 구현**하라.

이번 Phase에서는 비즈니스 기능 변경이나 대규모 리팩터링은 하지 않는다.

완료 후 다음 형식으로 보고한다.

1. 기존 Swagger/OpenAPI 구조 진단
2. 선택한 OpenAPI 3.x 구현 방식과 이유
3. 변경 파일
4. 제공 endpoint
5. operationId 목록
6. 검증 결과
7. 남은 Phase 2 과제

검증이 모두 통과하면 변경사항을 적절한 커밋 메시지로 커밋하되 **tag/release는 생성하지 않는다.**
