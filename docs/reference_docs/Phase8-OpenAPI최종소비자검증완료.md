# Phase 8 — OpenAPI 최종 소비자 검증 및 완료

기준 커밋:

```text
732a83d
```

현재 OpenAPI 3.x 단일 계약, CI Gate, runtime schema/status/semantic/side-effect 검증, MCP·SDK 친화적 계약 정비까지 완료되어 있다.

이번 Phase는 **실제 OpenAPI 소비자 관점의 최종 검증과 프로젝트 마무리**를 수행한다.

## 목표

`docs/openapi.yaml`을 실제 도구가 소비한다고 가정하고 6개 operation을 최종 검증한다.

```text
getHealth
getReadiness
listUsers
createUser
getUser
sendMail
```

### 1. MCP Tool 생성 관점 검증

OpenAPI를 기반으로 각 operation을 Tool로 변환했을 때 다음을 확인한다.

- operationId → 안정적인 tool name 생성 가능
- path/query/body 입력 schema가 정확히 생성됨
- required/optional 구분 유지
- description이 LLM이 이해하기 충분함
- 중첩 request/response schema 및 `$ref` 정상 해석
- free-form/ambiguous schema 없음
- 오류 응답 때문에 tool input schema가 오염되지 않음

가능하면 현재 프로젝트 내에서 재현 가능한 간단한 검증 테스트를 추가한다.

특정 MCP 제품에 대한 런타임 의존성은 추가하지 않는다.

### 2. SDK 생성 관점 검증

범용 OpenAPI generator가 해석할 수 있는지 점검한다.

최소 확인:

```text
operationId 중복 없음
schema name 충돌 없음
request/response type 생성 가능
잘못된/순환 $ref 없음
모호한 object 없음
unsupported construct 없음
```

실제 SDK 생성 도구를 사용할 수 있으면 임시 생성 후 compile/parse 수준까지만 확인하고 생성 결과물은 저장소에 커밋하지 않는다.

### 3. 최종 회귀 검증

기존 검증을 모두 실행한다.

```bash
go test ./...
go vet ./...
make openapi-check
```

가능하면 GitHub Actions의 최종 CI 상태도 확인한다.

## 문서 마무리

README에 현재 OpenAPI 사용 방법이 충분한지 최종 점검한다.

최소 포함:

```text
GET /openapi.json
/swagger/index.html
make openapi-check
신규 API 추가 시 계약 + runtime case 추가 절차
OpenAPI가 Single Source of Truth라는 원칙
```

이미 충분하면 불필요하게 수정하지 않는다.

## 완료 기준

다음을 모두 만족하면 OpenAPI 개선 작업을 **완료 상태**로 판단한다.

- 실제 route ↔ OpenAPI 일치
- request/response runtime contract 일치
- semantic/side-effect 검증 통과
- CI Gate 존재
- MCP/SDK 자동 소비에 구조적 문제 없음
- 문서 및 개발 절차 정리 완료

새 기능이나 대규모 리팩터링은 하지 않는다.

## 완료 보고

1. MCP 소비자 검증 결과
2. SDK 소비자 검증 결과
3. 발견·수정한 최종 문제
4. 변경 파일
5. 전체 테스트/CI 결과
6. OpenAPI 개선 작업 최종 완료 여부

`docs/reference_docs/`는 수정하지 않는다.

검증 완료 후 필요한 변경만 커밋하되 **tag/release는 생성하지 않는다.**