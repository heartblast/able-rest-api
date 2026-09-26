# Phase 6 — Runtime Side Effect Contract 검증

기준 커밋:

```text
b000de8
```

현재 OpenAPI 3.x 계약, router drift, runtime schema/status coverage, 응답 의미 검증까지 구축되어 있다.

이번 단계에서는 **HTTP 응답뿐 아니라 API 실행으로 발생하는 저장소·메일 발송 부작용이 요청 계약과 일치하는지 통합 검증**하라.

## 목표

기존 `runtimeContractCases`와 테스트 대역을 활용하여 주요 write operation의 side effect를 검증한다.

### `createUser`

성공 시:

- repository에 사용자 1건이 실제 저장됨
- 저장된 name/email이 정규화된 응답 값과 일치
- 예상하지 않은 추가 저장 없음

실패 시:

- validation 오류에서는 repository write가 발생하지 않음

### `sendMail`

성공 시:

- mail sender가 정확히 1회 호출됨
- To/CC/BCC/Subject/Body/IsHTML/Attachment 값이 요청 의미와 일치
- 중복 recipient 처리 결과와 `accepted_recipients`가 실제 발송 대상과 일치

실패 시:

- invalid request에서는 SMTP 호출 없음
- disabled/error 경로의 호출 여부와 응답 status/error code가 실제 서비스 정책과 일치

## 구현 원칙

- 실제 외부 DB/SMTP 사용 금지
- 기존 fake/mock/test repository를 확장해 호출 횟수와 전달 값을 관측
- handler 내부 구현 세부사항이 아니라 **외부에 관찰 가능한 부작용 계약**을 검증
- 기존 `runtimeContractCases` 구조를 가능한 한 재사용
- 과도한 mocking framework 도입 금지
- read-only API에는 불필요한 side-effect 테스트를 추가하지 말 것

필요하면 case에 다음 정도의 hook을 추가한다.

```text
assertSideEffect
```

공통 검증과 operation별 side-effect assertion은 분리한다.

## 회귀 검증

다음 오류를 의도적으로 만들었을 때 테스트가 실패하는지 확인한다.

- repository write 누락/중복
- SMTP 호출 누락/중복
- 전달된 주요 필드 값 불일치
- 실패 요청에서 불필요한 side effect 발생

실패 메시지에는 가능하면 `operationId`와 불일치 항목을 표시한다.

## 검증

```bash
go test ./...
go vet ./...
make openapi-check
```

기존 OpenAPI/schema/semantic contract 검증을 모두 유지한다.

## 완료 보고

1. 추가한 side-effect 검증
2. operation별 검증 항목
3. 발견·수정한 구현/계약 불일치
4. 변경 파일
5. 전체 검증 결과
6. Phase 7 권고

`docs/reference_docs/`는 수정하지 말고, 검증 후 커밋하되 **tag/release는 생성하지 않는다.**