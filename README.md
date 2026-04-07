# my-api

장기 유지보수와 멀티 DB 전환 가능성을 우선한 Go 1.26.1 REST API 템플릿입니다.

## 디렉터리 구조

```text
my-api/
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
  README.md
  go.mod
```

## 실행 방법

1. 설정 파일 생성

```bash
cp configs/app.example.yaml configs/app.yaml
```

2. 환경변수 준비

```bash
export APP_MASTER_KEY="Base64Encoded32ByteKeyHere=="
export DB_VENDOR="postgres"
```

PowerShell 예시:

```powershell
$env:APP_MASTER_KEY="Base64Encoded32ByteKeyHere=="
$env:DB_VENDOR="postgres"
```

3. 서버 실행

```bash
make run
```

또는

```bash
go run ./cmd/server configs/app.yaml
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

현재 대표 구현:

- PostgreSQL
- MySQL

현재 skeleton 확장 포인트:

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

- 인증/인가: `internal/delivery/http/middleware`
- 감사 로그: `internal/app/service` 또는 별도 이벤트 발행 계층
- 관측성: `internal/platform/logger` 확장
- 외부 시크릿 저장소: `internal/infra/security.SecretProvider` 구현 추가
- 신규 리소스: 동일한 계층 패턴으로 `domain -> service -> handler -> infra/db/<vendor>` 추가

## Oracle / HSQLDB 메모

- Oracle은 기본 빌드에 드라이버를 넣지 않았습니다.
- Oracle 전용 드라이버와 cgo 의존성은 별도 build tag 기반 어댑터로 격리하는 방향을 권장합니다.
- HSQLDB는 선택한 드라이버 전략에 맞춰 `factory` 와 `hsqldb` 저장소 구현을 채우면 됩니다.
