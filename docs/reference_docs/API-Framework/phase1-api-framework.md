# Framework Phase 1 — 공통 플랫폼과 업무 모듈 경계 정리

`heartblast/able-rest-api`를 향후 `able-security-api`, `able-ops-api` 등에서 재사용 가능한 **OpenAPI-first 사내 REST API Framework**로 발전시키기 위한 1단계 리팩터링을 수행하라.

## 목표

현재 기능을 유지하면서 **공통 플랫폼 코드와 업무 도메인 코드를 명확히 분리**한다.

현재 주요 기능:

```text
user
mail
scheduler
DB abstraction
config
logging
HTTP middleware
OpenAPI
contract tests
```

## 우선 분석

전체 소스를 진단해 다음을 분류한다.

```text
Framework 공통 기능
업무 Domain 기능
Infrastructure
HTTP Transport
재사용하기 어려운 결합
```

특히 `internal/app`, `delivery/http`, `infra`, `domain` 사이의 의존 방향을 확인한다.

## 이번 Phase 구현 범위

대규모 디렉터리 이동보다 **경계와 의존성 정리**를 우선한다.

가능하면 다음 구조를 목표로 한다.

```text
internal/
  platform/
    config/
    logging/
    http/
    openapi/
    security/
  modules/
    user/
    mail/
  infra/
    db/
    persistence/
```

단, 현재 구조에서 무리한 이동이 오히려 변경량을 키우면 최소 변경으로 동일한 경계를 만든다.

### 반드시 수행

- user/mail을 업무 모듈로 명확히 구분
- config/logging/OpenAPI/common middleware/error response를 공통 플랫폼으로 분류
- framework code가 user/mail 같은 구체 업무 코드에 의존하지 않도록 정리
- 신규 module 추가 시 기존 framework 수정이 최소화되도록 구성
- composition/wiring은 `cmd/server` 또는 명확한 bootstrap 계층에서 수행
- package cycle 및 전역 상태 증가 금지

## 확장 규칙

신규 업무 기능은 기본적으로 다음 흐름을 따르게 한다.

```text
module
 → domain/service
 → HTTP handler
 → route registration
 → OpenAPI contract
 → runtime contract test
```

가능하면 module별 route registration 패턴을 만들어 `router.go`가 계속 비대해지지 않도록 한다.

단, 과도한 plugin framework나 reflection 기반 자동 등록은 만들지 않는다.

## 호환성

다음은 변경하지 않는다.

```text
기존 URI
JSON response envelope
operationId
OpenAPI 계약 의미
DB 동작
mail 동작
scheduler 동작
```

기존 OpenAPI/보안/contract test를 모두 유지한다.

## 문서

README 또는 개발 문서에 간략히 추가한다.

- Framework 공통 영역
- 업무 module 영역
- 신규 module 추가 절차

## 검증

```bash
go test ./...
go vet ./...
make openapi-check
```

기존 security hardening 검사가 있다면 함께 통과시킨다.

## 완료 보고

1. 기존 구조의 결합 문제
2. Framework와 Domain 경계
3. 변경한 package/파일
4. 신규 module 추가 방식
5. 호환성 유지 여부
6. 전체 검증 결과
7. Framework Phase 2 권고

`docs/reference_docs/`는 수정하지 않는다.

검증 후 커밋하되 **tag/release는 생성하지 않는다.**