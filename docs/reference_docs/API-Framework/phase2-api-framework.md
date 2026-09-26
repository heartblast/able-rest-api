# Framework Phase 2 — 업무 모듈 응집도 강화

기준 커밋:

```text
77150fe
```

Phase 1에서 공통 라우터와 `user`/`mail` 모듈의 route registration을 분리했다.

이번 단계에서는 **업무별 service, DTO, handler를 각 module 내부로 이동하여 공통 패키지와 업무 코드의 결합을 더 줄인다.**

## 목표

현재 공용 위치에 남아 있는 업무별 코드를 점검하고 가능하면 다음 구조로 정리한다.

```text
internal/modules/
  user/
    service.go
    dto.go
    handler.go
    routes.go

  mail/
    service.go
    dto.go
    handler.go
    routes.go
```

필요하면 domain model/port는 재사용성과 의존 방향을 기준으로 module 내부 또는 공통 domain에 유지한다.

## 원칙

- user/mail 전용 타입과 로직은 해당 module이 소유
- `platform`과 공통 router가 업무 DTO/service/handler를 import하지 않도록 유지
- module 간 직접 의존 금지
- DB/SMTP 같은 외부 구현은 interface를 통해 주입
- composition/wiring은 계속 `cmd/server`에서 수행
- package cycle 금지
- 단순 파일 이동만 하지 말고 package 책임이 명확해지도록 정리

## 반드시 유지

```text
기존 URI
JSON response envelope
operationId
OpenAPI 계약
DB/mail/scheduler 동작
runtime contract test
security 동작
```

기능 변경은 하지 않는다.

## 추가 점검

신규 module 추가 시 최소한 다음 파일/책임만 추가하면 되는 구조인지 확인한다.

```text
service/domain logic
DTO
handler
routes
OpenAPI contract
contract test
```

공통 router나 platform 코드를 수정해야 하는 부분이 있다면 그 이유를 줄이고 최소화한다.

README의 module 개발 절차도 실제 구조에 맞게 갱신한다.

## 검증

```bash
go test ./...
go vet ./...
go build ./...
make openapi-check
git diff --check
```

## 완료 보고

1. 이동/정리한 업무 코드
2. module별 책임
3. 공통 패키지에서 제거된 업무 의존성
4. 신규 module 추가 절차
5. 호환성 유지 결과
6. 전체 검증 결과
7. Framework Phase 3 권고

`docs/reference_docs/`는 수정하지 않는다.

검증 후 커밋하되 **tag/release는 생성하지 않는다.**