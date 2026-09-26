# Phase 5 — Runtime Contract 의미 검증 강화

기준 커밋:

```text
d579079
```

현재 OpenAPI 3.x 계약, router drift 검사, runtime contract coverage, CI Gate까지 구축되어 있다.

이번 단계에서는 **schema/type 검증을 넘어 실제 응답 값이 API 의미와도 일치하는지 contract test를 강화**하라.

## 목표

`runtimeContractCases`를 확장해 필요한 operation부터 다음을 검증한다.

- 성공 응답의 `success == true`
- 오류 응답의 `success == false`
- `request_id`가 비어 있지 않음
- 오류 응답의 `error.code`가 기대한 값과 일치
- 생성/조회 API의 주요 필드 값이 요청 또는 fixture와 일치
- 목록 API의 `count`와 `items` 정합성
- mail API의 `accepted_recipients` 값 검증
- health/readiness의 상태 값 검증

단순 JSON 문자열 비교는 피하고, 구조화된 assertion 방식으로 구현한다.

## 구현 방향

기존 `runtimeContractCases` 구조를 재사용하고 필요하면 다음 정도의 assertion 정보를 추가한다.

```text
expectedErrorCode
assertResponse
```

공통 검증은 helper로 중복을 줄이고, operation별 의미 검증만 case에 남긴다.

OpenAPI schema와 실제 구현이 충돌할 경우 **실제 API 의미를 먼저 확인한 뒤 계약 또는 구현 중 잘못된 쪽만 수정**한다.

## 추가 점검

OpenAPI schema에 명백히 표현 가능한 제약도 함께 점검한다.

예:

```text
required
format
minimum
minLength
minItems
enum
```

단, 실제 구현에서 보장하지 않는 제약을 문서에 임의 추가하지 않는다.

## 검증

```bash
go test ./...
go vet ./...
make openapi-check
```

의미 검증을 의도적으로 깨뜨렸을 때 해당 operation과 assertion 원인이 명확히 출력되는지도 확인한다.

가능하면 `d579079` 이후 GitHub Actions 실행 상태도 확인하되, CI 확인 실패 자체 때문에 코드 작업을 막지는 않는다.

## 완료 보고

1. 추가한 의미 검증
2. operation별 주요 assertion
3. OpenAPI 제약 보완 여부
4. 발견·수정한 불일치
5. 전체 검증/CI 결과
6. Phase 6 권고

`docs/reference_docs/`는 수정하지 말고, 검증 후 커밋하되 **tag/release는 생성하지 않는다.**