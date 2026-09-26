# AGENT_USAGE.md 생성 — 외부 Go 프로젝트에서의 재사용 가이드

`heartblast/able-rest-api`를 분석하여, 다른 Go 프로젝트의 코딩 에이전트가 이 Framework를 **빠르게 이해하고 안전하게 재사용할 수 있도록 저장소 루트에 `AGENT_USAGE.md`를 작성**하라.

## 목표

문서는 일반 README 설명이 아니라 **에이전트용 실행 지침**으로 작성한다.

먼저 실제 소스를 확인해 다음을 정확히 파악한다.

- `go.mod` module path
- 외부 프로젝트에서 import 가능한 공개 package
- `internal/`에 남아 있는 비공개 구현
- Framework bootstrap 방식
- module 등록 방식
- dependency wiring 위치
- OpenAPI 관리 방식
- scheduler/mail 사용 방식
- 검증 명령
- version/tag 사용 방식

추측으로 API 예제를 만들지 말고 **현재 실제 코드에서 사용 가능한 API만 문서화**한다.

## 문서 구성

### 1. Framework 목적

짧게 설명:

```text
able-rest-api는 OpenAPI-first REST API Framework이며,
업무 module + HTTP + OpenAPI + contract test + scheduler 등의
공통 구조를 다른 Go 프로젝트에서 재사용하기 위한 기반이다.
```

### 2. Quick Start

외부 프로젝트에서 사용하는 최소 절차를 작성한다.

가능한 경우 실제 명령 포함:

```bash
go get <actual-module-path>@<version>
```

private repository라면 필요한 `GOPRIVATE` 설정도 설명한다.

### 3. 공개 Package Map

실제 공개 package만 표로 작성한다.

예:

```text
package | 목적 | 언제 사용하는가
```

`internal/*`는 외부 import가 불가능함을 명확히 표시한다.

### 4. 최소 사용 예제

실제 공개 API를 사용한 **컴파일 가능한 최소 예제**를 작성한다.

예제 흐름:

```text
config
→ framework/bootstrap
→ module 등록
→ dependency wiring
→ HTTP server
```

현재 공개 API가 부족해 외부 사용이 불가능한 부분은 가짜 코드를 만들지 말고 명확히 기록한다.

### 5. 신규 업무 Module 추가 절차

에이전트가 반드시 따를 순서를 짧게 명시한다.

```text
1. module 생성
2. domain/service 작성
3. DTO/handler 작성
4. route 등록
5. bootstrap에서 dependency wiring
6. OpenAPI 계약 수정
7. runtime contract case 추가
8. side-effect test 필요 여부 확인
9. 전체 검증
```

### 6. OpenAPI 규칙

반드시 명시:

- `docs/openapi.yaml`이 Single Source of Truth
- operationId 안정적으로 유지
- route 추가 시 OpenAPI 동시 수정
- request/response/error schema 정의
- runtime contract test 추가
- 기존 API 변경 시 호환성 확인

### 7. Architecture Rules

에이전트가 지켜야 할 의존 방향을 명확히 작성한다.

```text
module → port ← infra
bootstrap → module + infra wiring
platform은 업무 module 내부 구현을 알지 않는다.
```

### 8. DO / DO NOT

간결하게 작성한다.

DO:

```text
공개 package만 import
dependency wiring은 bootstrap에서 수행
module은 자신의 model/port를 소유
OpenAPI와 contract test를 함께 변경
기존 framework extension point 우선 사용
```

DO NOT:

```text
internal/* import
module 간 내부 구현 직접 import
공통 router에서 업무 handler 생성
SMTP/DB 구현을 module logic에서 직접 생성
OpenAPI 없는 route 추가
contract test 우회
```

### 9. Scheduler / Mail

현재 구현된 실제 기능만 설명한다.

- 즉시 메일
- 예약 메일
- retry
- DB claim
- Message-ID/idempotency
- migration 필요 여부

SMTP exactly-once가 보장되지 않는 등의 known limitation도 명시한다.

### 10. 검증 명령

현재 프로젝트 기준으로 최소:

```bash
go test ./...
go test -race ./...
go vet ./...
go build ./...
make openapi-check
```

외부 프로젝트에서 어떤 검증을 수행해야 하는지도 구분해 설명한다.

### 11. Agent 작업 전 체크리스트

10개 이하로 짧게 작성한다.

예:

```text
- 공개 API로 해결 가능한가?
- internal을 import하려 하고 있지 않은가?
- 신규 route인가?
- OpenAPI를 수정했는가?
- operationId를 추가했는가?
- runtime contract test가 있는가?
- module 경계를 침범하지 않았는가?
```

### 12. Known Limitations

현재 실제 제한만 기술한다.

추후 계획이나 존재하지 않는 기능을 현재 기능처럼 설명하지 않는다.

## 문서 작성 원칙

- 5\~10분 내 읽을 수 있는 분량
- 에이전트가 검색하기 쉬운 명확한 heading 사용
- 긴 설계 설명보다 규칙과 예제 우선
- 현재 코드와 다른 예제를 만들지 말 것
- README 내용을 그대로 복제하지 말 것
- 상세 설명이 이미 README/docs에 있으면 링크하고 핵심만 요약

## 추가 진단

문서 작성 과정에서 다음 문제가 발견되면 함께 보고한다.

```text
외부 import 가능한 package가 사실상 없음
go.mod module path가 원격 사용에 부적합
공개 API가 internal 구현을 누출
외부 사용에 필요한 bootstrap API 부재
```

이 경우 이번 작업에서 대규모 구조 변경은 하지 말고,
**Public Framework API 추출을 다음 작업으로 권고**한다.

## 검증

```bash
git diff --check
```

문서에 포함한 Go 예제가 실제 공개 API 기준으로 컴파일 가능한 경우 가능한 범위에서 검증한다.

## 완료 보고

1. 생성한 `AGENT_USAGE.md`
2. 확인한 공개 package
3. 외부 프로젝트 최소 사용 흐름
4. 발견한 재사용 제약
5. 다음 작업 권고
6. 검증 결과

기존 `docs/reference_docs/`는 수정하지 않는다.

이번 작업은 문서 작성만 수행하고 **tag/release는 생성하지 않는다.**
