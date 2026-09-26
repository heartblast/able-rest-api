# Phase 4 — 신규 API 추가 시 Contract Test 강제

기준 커밋:

```text
b440f87
```

현재 OpenAPI 3.x 단일 계약, runtime contract test, CI Gate까지 구축되어 있다.

이번 단계에서는 **새 API operation 추가 시 OpenAPI 계약과 정상·오류 런타임 테스트가 함께 추가되지 않으면 CI가 실패하도록 개발 규칙을 강화**하라.

## 목표

다음을 구현한다.

- 실제 router의 모든 operation이 OpenAPI에 존재하는지 검증
- OpenAPI의 모든 operation이 runtime contract test 대상인지 검증
- 각 operation마다 최소 1개 정상 응답과 주요 오류 응답 테스트 존재 여부 확인
- 신규 operation 추가 후 계약/테스트 누락 시 `make openapi-check` 실패
- 실패 메시지에 `method`, `path`, `operationId`, 누락 항목을 명확히 표시

## 구현 원칙

기존 테스트 구조를 최대한 재사용하고 별도 테스트 프레임워크는 만들지 않는다.

가능하면 runtime contract case를 구조화하여 다음 정보를 한 곳에서 관리한다.

```text
method
path
operationId
request
expected status
success/error case
```

이 정보를 기준으로 OpenAPI의 declared response status와 실제 테스트 coverage를 자동 비교한다.

단순히 테스트 파일 이름이나 함수명을 문자열 검색하는 방식은 피하고, 유지보수 가능한 구조로 구현한다.

## 추가 점검

6개 기존 operation에 대해 정상·오류 시나리오 coverage를 확인하고 누락된 주요 사례만 보완한다.

비즈니스 로직, URI, response envelope, OpenAPI 계약 의미는 변경하지 않는다.

## 검증

```bash
go test ./...
go vet ./...
make openapi-check
```

추가로 의도적으로 operation 또는 contract case를 누락했을 때 검사가 실패하는지도 테스트한다.

GitHub Actions에서 CI가 실제 통과하는지 확인 가능하면 확인한다.

## 완료 보고

1. 신규 API 누락 방지 방식
2. runtime contract case 관리 구조
3. 보완한 테스트
4. 변경 파일
5. 검증 및 CI 결과
6. Phase 5 권고사항

`docs/reference_docs/`는 건드리지 말고, 검증 완료 후 커밋하되 **tag/release는 생성하지 않는다.**