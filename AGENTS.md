# Agent instructions for able-rest-api

이 파일은 저장소 전체에 적용한다. 사용자의 현재 요청과 기존 변경사항을 먼저 확인하고, 관련 없는 파일은 수정하거나 커밋하지 않는다. 필요한 코드를 직접 고치고 검증한 뒤 결과와 남은 위험을 간결하게 보고한다.

## 먼저 확인할 것

- 작업 시작 시 `git status --short`로 기존 변경사항을 확인한다. 사용자가 만든 미추적 파일도 보존한다.
- 경로와 현재 동작은 `rg`와 코드로 확인한다. `README.md`는 사용법 참고 자료이며, 코드와 차이가 있으면 실제 구현을 기준으로 판단한다.
- Go 버전은 `go.mod`를 따른다. 표준 Go 명령은 `go test ./...`, `go vet ./...`, `go build ./...`이다.
- `./build.sh`는 기본적으로 Go 모듈을 사용한다. 오프라인 빌드가 필요한 경우에만 `go mod vendor` 후 `./build.sh build --offline`을 사용한다. 직접 실행할 스크립트의 실행 권한도 확인한다.

## 코드 위치와 경계

- `cmd/server`, `cmd/scheduler`, `cmd/migrate`, `cmd/secretenc`: 실행 진입점. 현재 `cmd/migrate`는 마이그레이션 디렉터리만 확인하며 SQL을 실행하지 않는다.
- `internal/domain`: 모델·포트·저장소 계약. `internal/app/service`: 입력 검증과 유스케이스. `internal/delivery/http`: DTO·핸들러·미들웨어·라우터. `internal/infra`: DB·SMTP·설정·스케줄러 구현.
- 새 HTTP 기능은 서비스에 업무 규칙을 두고 핸들러는 요청/응답 변환에 집중한다. DB 접근과 외부 통신은 infra 구현 및 포트를 통해 연결한다.
- 소스 코드에 새로 작성하거나 수정하는 주석은 한국어로 쓴다. Go 문서 주석, 인라인 주석, `TODO`에도 적용하며 식별자와 표준 기술 용어는 원문을 유지해도 된다. 작업과 무관한 기존 주석은 일괄 변경하지 않는다.
- 실제 DB 어댑터는 PostgreSQL과 MySQL이다. Oracle과 HSQLDB는 현재 스텁이므로 완성된 지원으로 가정하거나 안내하지 않는다.
- 운영 설정은 `internal/infra/config/config.go`에서 검증한다. 예시를 바꾸면 `configs/app.example.yaml`과 필요한 사용법 문서도 맞춘다. 실제 `configs/app.yaml`, `.env`, 키, 암호문 원문은 커밋하지 않는다.

## HTTP API와 OpenAPI 계약

- `docs/openapi.yaml`이 OpenAPI 계약의 단일 원본이다. `docs/openapi.go`가 이를 포함해 JSON으로 제공한다. Swagger 생성 파일이나 핸들러 주석을 원본으로 사용하지 않는다.
- 엔드포인트, DTO, 요청 제한, 응답 상태 또는 오류 코드를 변경하면 YAML과 런타임 동작을 함께 갱신한다.
- 새 operation 또는 응답 상태를 추가하면 `internal/delivery/http/router/runtime_contract_test.go`의 `runtimeContractCases`에도 정상·오류 사례를 추가한다. 계약 테스트는 모든 선언된 상태와 주요 부작용을 확인한다.
- API 변경 중에는 관련 패키지 테스트를 먼저 실행하고, 완료 전 `go test ./...`, `go vet ./...`, `make openapi-check`를 실행한다. 실제 DB나 SMTP 없이 계약 테스트가 돌아간다.

## 보안과 호환성

- `/api/v1/*`, `/openapi.json`, 활성화된 `/swagger/*`의 API 키 보호를 유지한다. `/health`와 `/ready`는 공개 상태 확인 경로다. 공유 API 키에는 사용자별 인가가 없으므로 이를 권한 검사의 대체물로 취급하지 않는다.
- JSON Content-Type 검사, 본문 크기 제한, 엄격한 디코딩, 메일 헤더·수신자·첨부파일 검증을 우회하지 않는다. 새 입력에도 크기와 개수의 상한을 둔다.
- 운영 환경에서 설정된 TLS·암호화·설정 파일 권한 검사를 약화하지 않는다. 클라이언트 응답이나 로그에 DB 오류 세부사항, 인증값, 비밀번호, 메일 내용 등을 노출하지 않는다.
- 보안 제한이나 API 상태가 바뀌면 회귀 테스트를 추가하고 OpenAPI를 동기화한다. 동작 변경은 기존 클라이언트에 미치는 영향을 확인한다.

## 변경 완료 기준

- 변경 범위에 맞는 테스트를 실행한다. API 변경은 위의 세 검증 명령을 모두 실행하고, 의존성 변경이나 보안 수정은 가능하면 `govulncheck ./...`도 실행한다.
- `git diff --check`와 `git status --short`를 확인한다. 실패한 검증은 숨기지 말고 원인과 남은 작업을 보고한다.
- 커밋할 때는 사용자 요청 범위에 포함된 파일만 담는다. 태그와 릴리스는 명시적 요청이 있을 때만 만든다.
