# able-rest-api

유지보수와 DB 전환 가능성을 고려해 구성한 Go 1.26.1 REST API 샘플 프로젝트입니다.

## 디렉터리 구조

```text
able-rest-api/
  cmd/
    server/
      main.go
    migrate/
      main.go
    secretenc/
      main.go
  docs/
    docs.go
  internal/
    app/
      service/
        user_service.go
    domain/
      model/
        user.go
      repository/
        user_repository.go
    delivery/
      http/
        dto/
          common.go
          user.go
        handler/
          health_handler.go
          response.go
          user_handler.go
        middleware/
          context.go
          json.go
          logging.go
          request_id.go
        router/
          router.go
    infra/
      config/
        config.go
      db/
        factory/
          factory.go
        dialect/
          dialect.go
          postgres.go
          mysql.go
          oracle.go
          hsqldb.go
        postgres/
          user_repository.go
        mysql/
          user_repository.go
        oracle/
          driver_stub.go
          user_repository.go
        hsqldb/
          user_repository.go
      security/
        provider.go
      persistence/
        repositories.go
    platform/
      logger/
        logger.go
  migrations/
    common/
      000001_baseline.sql
    postgres/
      000001_create_users.sql
    mysql/
      000001_create_users.sql
    oracle/
      000001_create_users.sql
    hsqldb/
      000001_create_users.sql
  configs/
    app.example.yaml
  Makefile
  build.ps1
  build-offline.ps1
  build.sh
  README.md
  go.mod
```

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

## Swagger 생성

```bash
make swag
```

또는

```bash
swag init -g cmd/server/main.go -o docs
```

Swagger UI 경로:

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
- `GET /swagger/*`

## Makefile 타깃

- `make run`
- `make build`
- `make test`
- `make swag`
- `make secretenc VALUE=my-password KEY_ENV=APP_MASTER_KEY`

## 확장 포인트

- 인증/권한: `internal/delivery/http/middleware`
- 감사 로그: `internal/app/service` 또는 별도 이벤트 발행 계층
- 관측성: `internal/platform/logger` 확장
- 외부 시크릿 저장소: `internal/infra/security.SecretProvider` 구현 추가
- 신규 리소스: 동일한 계층 패턴으로 `domain -> service -> handler -> infra/db/<vendor>` 추가

## Oracle / HSQLDB 메모

- Oracle은 기본 빌드에 드라이버를 넣지 않았습니다.
- Oracle 전용 드라이버는 cgo 의존성이나 별도 build tag 기반 파일로 분리하는 방향을 권장합니다.
- HSQLDB는 선택한 드라이버 전략에 맞춰 `factory` 와 `hsqldb` 저장소 구현을 채우면 됩니다.
