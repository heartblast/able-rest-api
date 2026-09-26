# able-rest-api

현재 릴리스: **v1.1.0**. 변경 내용과 운영 시 제한사항은 [v1.1.0 릴리스 노트](docs/releases/v1.1.0.md)를 참고하세요.

유지보수와 DB 전환 가능성을 고려해 구성한 Go 1.26.1 REST API 샘플 프로젝트입니다.

## 디렉터리 구조

```text
able-rest-api/
  cmd/
    server/             # HTTP 의존성 조립
    scheduler/          # 예약 작업 의존성 조립
    migrate/
    secretenc/
  docs/
    openapi.yaml        # OpenAPI 계약 원본
    openapi.go
  internal/
    modules/
      user/             # 서비스, DTO, 핸들러, 경로 등록
      mail/             # 서비스, DTO, 핸들러, 경로 등록, 예약 메일 작업
    domain/             # DB·메일·스케줄러가 공유하는 모델과 인터페이스
    app/service/        # 공통 예약 작업 서비스
    delivery/http/
      middleware/       # 공통 HTTP 미들웨어
      router/           # 공통 라우터와 계약 테스트
    platform/
      http/             # 상태 확인과 공통 응답 형식
      logger/
    infra/              # 설정, DB, SMTP, 보안, 스케줄러 구현
  migrations/
  configs/
    app.example.yaml
  Makefile
  build.sh
  README.md
  go.mod
```

## Framework와 업무 모듈 경계

- `internal/modules/user`는 사용자 모델·저장소 계약·서비스·DTO·핸들러·경로를 소유합니다. `internal/modules/mail`은 메일 모델·발송 계약·서비스·DTO·핸들러·경로를 소유합니다.
- `internal/modules/scheduler`는 작업 정의·실행 이력·러너와 락의 계약·작업 서비스를 소유합니다. 예약 메일 작업은 메일 모듈의 명령만 호출합니다. DB 저장소, SMTP 발송기, 스케줄러 구현은 각 모듈의 계약을 따릅니다.
- `internal/delivery/http/router`는 인증이 적용된 `/api/v1` 라우터와 모듈 경로 등록 함수를 조합합니다. 공통 미들웨어와 `internal/platform/http`는 업무 DTO·서비스·핸들러에 의존하지 않습니다.
- `cmd/server`가 DB·SMTP·서비스를 생성하고 각 모듈의 `Routes`를 공통 라우터에 전달합니다. 별도 프로세스인 `cmd/scheduler`는 메일 서비스와 예약 작업을 조립합니다. PostgreSQL/MySQL 저장소와 SMTP 구현은 `internal/infra`에 있습니다.

`scheduler.mail_dispatch.enabled`를 켜면 `cmd/scheduler`가 설정한 `interval`마다 `to`/`cc`/`bcc`, `subject`, `body`, `is_html`, `attachments`로 메일을 발송합니다. 첨부파일은 `filename`, `content_type`, `content_base64`를 사용합니다. 이 설정은 기존 반복 발송용입니다. PostgreSQL 락은 여러 스케줄러 프로세스의 동일 작업 동시 실행을 막습니다. `lock_provider: none`에서는 반복 작업의 프로세스 사이 중복 실행을 막지 않습니다.

개별 예약 메일은 PostgreSQL 또는 MySQL의 `scheduled_mails` migration을 적용한 뒤 `scheduler.scheduled_mail.enabled`를 켜서 처리합니다. `go run ./cmd/schedulemail -config configs/app.yaml -input request.json`으로 한 건을 예약합니다. 입력 JSON은 `{"scheduled_at":"2026-10-01T09:00:00+09:00","mail":{"to":["user@example.com"],"subject":"안내","body":"내용"}}` 형식입니다. 명령은 생성된 예약 ID를 출력합니다. 즉시 발송 API는 요청마다 SMTP로 바로 전송하고 영속 예약이나 자동 재시도를 만들지 않습니다.

예약 시 `<예약 ID@scheduled.able-rest-api.invalid>` 형식의 고유한 `message_id`를 DB에 저장합니다. 발송, retry, lease 만료 후 재claim 및 프로세스 재시작 시 같은 값을 SMTP `Message-ID` 헤더로 보냅니다. claim token은 각 claim의 상태 변경을 보호하는 별도 값이며 재claim마다 바뀝니다. 예약 ID와 Message-ID는 예약 발송 로그와 DB 행에서 추적할 수 있습니다. `max_attempts`에는 최초 발송도 포함되며, 일시적 실패는 `retry_delay`부터 최대 1시간까지 지수 지연으로 재시도합니다. `PROCESSING` 상태가 `lease_duration`을 넘기면 재claim하고, 시도 한도를 넘으면 `FAILED`로 종료합니다. SMTP가 수락한 뒤 `SENT` 기록 전에 프로세스가 종료되거나 DB 기록이 실패하면 동일 Message-ID로 재발송될 수 있습니다. 수신 시스템은 이 헤더를 중복 식별에 사용할 수 있지만, SMTP와 수신 서버는 Message-ID에 따른 중복 제거 또는 exactly-once 전달을 보장하지 않습니다.

### DB migration 적용

`cmd/migrate`는 대상 디렉터리만 확인하며 SQL을 실행하지 않습니다. 배포 시 사용하는 migration 도구 또는 DB 클라이언트로 `migrations/postgres/` 또는 `migrations/mysql/`의 SQL을 번호 순서대로 적용하고 적용 이력을 관리하세요. 기존 설치에는 PostgreSQL `000004_add_scheduled_mail_message_id.sql`, MySQL `000003_add_scheduled_mail_message_id.sql`을 스케줄러 재시작 전에 적용해야 합니다. 두 migration은 기존 예약의 ID에서 Message-ID를 채웁니다. 새 설치에도 앞선 테이블 생성 migration을 먼저 적용합니다.

새 업무 모듈은 `internal/modules/<이름>/`에 모델과 외부 경계 계약, 서비스의 업무 규칙, DTO, HTTP 핸들러, `routes.go`를 추가합니다. 외부 구현이 필요하면 해당 모듈의 인터페이스를 따르는 `internal/infra` 구현을 `cmd/server`에서 주입하고 서비스와 `Routes`를 조립합니다. 공통 라우터는 경로 등록 함수만 받으므로 일반적인 모듈 추가에는 수정할 필요가 없습니다. `docs/openapi.yaml`에 요청·응답과 operationId를 정의하고 `internal/delivery/http/router/runtime_contract_test.go`에 선언된 상태별 정상·오류 사례를 추가한 뒤 `go test ./...`, `go vet ./...`, `go build ./...`, `make openapi-check`를 실행합니다. 기존 URI와 응답 형식은 유지합니다.

## 실행 준비

1. 설정 파일 생성

```bash
cp configs/app.example.yaml configs/app.yaml
```

PowerShell:

```powershell
Copy-Item configs/app.example.yaml configs/app.yaml
```

2. 환경 변수 설정

```bash
export APP_MASTER_KEY="Base64Encoded32ByteKeyHere=="
export DB_VENDOR="postgres"
```

PowerShell:

```powershell
$env:APP_MASTER_KEY="Base64Encoded32ByteKeyHere=="
$env:DB_VENDOR="postgres"
```

## 빌드 및 서버 실행 방법

### 1. Makefile 사용

서버 실행:

```bash
make run
```

빌드:

```bash
make build
```

테스트:

```bash
make test
```

### 2. PowerShell 스크립트 사용 (`build.ps1`)

서버 실행:

```powershell
.\build.ps1 run -Config configs/app.yaml
```

빌드:

```powershell
.\build.ps1 build
```

테스트:

```powershell
.\build.ps1 test
```

비밀번호 암호화:

```powershell
.\build.ps1 secretenc -Value "my-db-password" -KeyEnv APP_MASTER_KEY
```

### 3. Bash 스크립트 사용 (`build.sh`)

Linux/macOS에서 실행 권한 부여:

```bash
chmod +x build.sh
```

서버 실행:

```bash
./build.sh run --config configs/app.yaml
```

빌드:

```bash
./build.sh build
```

테스트:

```bash
./build.sh test
```

비밀번호 암호화:

```bash
./build.sh secretenc --value "my-db-password" --key-env APP_MASTER_KEY
```

기본 Bash 빌드는 Go 모듈을 사용하므로 `vendor/`가 없어도 실행됩니다. 네트워크 없이 빌드하려면 온라인 환경에서 `go mod vendor`를 실행한 뒤 `./build.sh build --offline`을 사용하세요.

### 4. 오프라인 PowerShell 스크립트 사용 (`build-offline.ps1`)

`vendor/` 디렉터리와 `vendor/modules.txt`가 준비되어 있으면 네트워크 없이 실행할 수 있습니다.

서버 실행:

```powershell
.\build-offline.ps1 run -Config configs/app.yaml
```

빌드:

```powershell
.\build-offline.ps1 build
```

테스트:

```powershell
.\build-offline.ps1 test
```

비밀번호 암호화:

```powershell
.\build-offline.ps1 secretenc -Value "my-db-password" -KeyEnv APP_MASTER_KEY
```

### 5. Go 명령 직접 사용

서버 실행:

```bash
go run ./cmd/server configs/app.yaml
```

빌드:

```bash
go build ./...
```

## OpenAPI 계약

로컬·개발·테스트 환경은 기본적으로 `127.0.0.1`에만 바인딩됩니다. 외부 접속이 필요한 경우 `app.host` 또는 `APP_HOST`를 명시하세요. 운영 환경은 기본적으로 `0.0.0.0`에 바인딩됩니다.

운영 환경에서는 `APP_API_KEY`에 32자 이상의 무작위 값을 설정하고 모든 `/api/v1/*` 요청에 `X-API-Key` 헤더를 전달해야 합니다. `security.api_key_env`로 환경 변수 이름을 바꿀 수 있습니다. 로컬·개발·테스트 환경에서도 해당 환경 변수에 값이 있으면 API 키 검사가 적용됩니다. `/openapi.json`과 활성화된 `/swagger/*`도 키로 보호되며 `/health`, `/ready`는 공개 상태 확인 경로입니다. Swagger UI는 보호된 OpenAPI 문서를 자동으로 읽지 못할 수 있으므로 운영 환경에서는 `swagger.enabled: false`를 권장합니다.

JSON 요청 본문은 기본 30 MiB로 제한하며 `security.max_request_body_bytes` 또는 `SECURITY_MAX_REQUEST_BODY_BYTES`로 1~100 MiB 범위에서 조절할 수 있습니다. 운영 환경의 설정 파일은 Unix에서 소유자 전용 권한(예: `chmod 600 configs/app.yaml`)을 사용해야 합니다. PostgreSQL은 `db.sslmode: verify-full`, MySQL은 `db.sslmode: "true"`가 필요하고, SMTP를 사용하면 TLS 또는 STARTTLS가 필요합니다. API 키는 HTTPS 종단 프록시 또는 HTTPS 연결을 통해서만 전달하세요. 키 교체 후 서버를 재시작해야 새 값이 적용됩니다.

`docs/openapi.yaml`이 OpenAPI 3.0 계약의 단일 원본입니다. 서버는 이 파일을 포함하고 JSON으로 변환하여 `GET /openapi.json`에서 제공합니다. 기존 `docs/docs.go`, `docs/swagger.json`, `docs/swagger.yaml`은 Swagger 2.0 생성 산출물이었으며 제거했습니다. Handler의 Swagger 주석을 다시 생성하지 않습니다.

API의 router, handler, DTO 또는 `docs/openapi.yaml`을 변경한 뒤 다음 명령을 실행합니다. CI도 push와 pull request마다 같은 세 명령을 필수 실행합니다.

```bash
go test ./...
go vet ./...
make openapi-check
```

`openapi-check`는 문법과 `$ref`, 실제 router와 OpenAPI 양방향 route/HTTP method 일치, `operationId` 누락·중복, DTO 속성, 실제 HTTP request/response의 status와 schema를 검증합니다. `/openapi.json`과 `/swagger/*`는 문서 제공 경로라 비교에서 제외합니다. 테스트는 가짜 DB 연결과 메일 발송기를 사용하므로 실제 DB·SMTP가 필요하지 않습니다.

새 API route를 추가할 때는 `docs/openapi.yaml`에 operation과 응답 status를 선언하고, `internal/delivery/http/router/runtime_contract_test.go`의 `runtimeContractCases`에 요청·기대 status·정상/오류 구분을 추가합니다. 각 operation의 모든 선언된 응답 status와 최소 한 개의 정상 사례를 실행해야 하며, 오류 응답을 선언한 경우 오류 사례도 필요합니다. 누락되면 `make openapi-check`가 method, path, operationId와 누락 항목을 출력하며 실패합니다.

Swagger UI는 `swagger.enabled: true`일 때 아래 경로에서 `/openapi.json`을 읽습니다.

```text
/swagger/index.html
```

## DB 비밀번호 암호화

```bash
export APP_MASTER_KEY="Base64Encoded32ByteKeyHere=="
go run ./cmd/secretenc --value "my-db-password" --key-env APP_MASTER_KEY
```

출력된 `ENC(...)` 값을 `db.password`에 넣으면 됩니다.

## DB 전환 방법

`db.vendor` 또는 `DB_VENDOR` 값만 변경하면 됩니다.

- `postgres`
- `mysql`
- `oracle`
- `hsqldb`

현재 실제 구현:

- PostgreSQL
- MySQL

현재 skeleton 확장 대상:

- Oracle
- HSQLDB

## 주요 엔드포인트

- `GET /health`
- `GET /ready`
- `GET /api/v1/users/{id}`
- `POST /api/v1/users`
- `GET /api/v1/users`
- `POST /api/v1/mail/send`
- `GET /openapi.json`
- `GET /swagger/*`

## Makefile 타깃

- `make run`
- `make build`
- `make test`
- `make openapi-check`
- `make secretenc VALUE=my-password KEY_ENV=APP_MASTER_KEY`

## 확장 포인트

- 인증/권한: `internal/delivery/http/middleware`
- 감사 로그: 해당 업무 모듈 또는 별도 이벤트 발행 계층
- 관측성: `internal/platform/logger` 확장
- 외부 시크릿 저장소: `internal/infra/security.SecretProvider` 구현 추가
- 신규 리소스: `internal/modules/<이름>`에 서비스·DTO·핸들러·경로를 만들고 필요한 도메인 인터페이스와 `infra` 구현을 연결

## Oracle / HSQLDB 메모

- Oracle은 기본 빌드에 드라이버를 넣지 않았습니다.
- Oracle 전용 드라이버는 cgo 의존성이나 별도 build tag 기반 파일로 분리하는 방향을 권장합니다.
- HSQLDB는 선택한 드라이버 전략에 맞춰 `factory` 와 `hsqldb` 저장소 구현을 채우면 됩니다.
