# Framework Phase 3 — Domain 소유권 및 Port 경계 정리

기준 커밋:

```text
109b852
```

현재 `user`와 `mail`의 service·DTO·handler는 각 module로 이동했고, 공통 router와 bootstrap도 분리되어 있다.

이번 단계에서는 **공유 `internal/domain`의 model/interface를 실제 사용처 기준으로 재분류해 module 소유권을 명확히 한다.**

## 목표

현재 `internal/domain` 전체를 분석해 다음으로 분류한다.

```text
user 전용
mail 전용
scheduler 전용
여러 module이 공유하는 진짜 공통 domain
infrastructure port
```

전용 model/interface는 가능하면 해당 module 내부로 이동한다.

예:

```text
internal/modules/user/
  model.go
  repository.go

internal/modules/mail/
  model.go
  sender.go

internal/modules/scheduler/
  ...

internal/domain/
  실제 공통 개념만 유지
```

## 원칙

- 단일 module에서만 사용하는 model/interface는 해당 module이 소유
- module 간 model 공유를 위해 무리하게 공통 domain으로 올리지 말 것
- module 간 직접 import 의존은 만들지 말 것
- DB/SMTP/JobRunner 등 외부 경계는 interface/port로 유지
- 구현체는 `infra`에 유지
- dependency direction은 `module → port ← infra` 형태를 유지
- package cycle 금지
- 불필요한 추상화나 generic framework 추가 금지

## 추가 점검

다음을 특히 확인한다.

- `User` model과 repository interface의 실제 소유권
- `MailMessage`, attachment, sender interface의 소유권
- scheduled job / execution / runner / lock 관련 타입의 책임
- 공통 `app/service`에 남은 코드가 실제 framework 공통인지 여부
- persistence repository aggregation이 module 구조와 과도하게 결합되어 있는지

필요하면 최소 범위에서 package와 wiring을 정리한다.

## 반드시 유지

```text
URI
JSON response envelope
operationId
OpenAPI 계약
DB/mail/scheduler 동작
runtime/side-effect contract test
security 동작
```

기능 변경은 하지 않는다.

## 검증

```bash
go test ./...
go vet ./...
go build ./...
make openapi-check
git diff --check
```

## 완료 보고

1. 기존 shared domain의 문제점
2. module별로 이동한 model/port
3. 공통 domain에 남긴 항목과 이유
4. dependency direction 변화
5. 호환성 유지 결과
6. 전체 검증 결과
7. Framework Phase 4 권고

`docs/reference_docs/`는 수정하지 않는다.

검증 후 커밋하되 **tag/release는 생성하지 않는다.**