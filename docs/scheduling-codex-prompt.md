# 스케줄링 구현용 Codex 프롬프트

아래 프롬프트를 Codex에 그대로 입력해서 사용한다.

```text
현재 워크스페이스는 Go REST API 프로젝트다. 이 코드베이스를 직접 읽고, 기존 구조와 스타일을 유지하면서 핵심 도메인용 스케줄링 기능의 기반 구조를 구현해줘.

반드시 지킬 것:
- 먼저 현재 코드베이스를 읽고 구조를 파악한 뒤 수정해라.
- 설명만 하지 말고 실제로 코드를 수정해라.
- 최소 범위로 변경하되, 서버에서 실제로 동작 가능한 형태로 끝까지 연결해라.
- 기존 사용자 변경사항을 되돌리거나 불필요한 리팩터링을 하지 마라.
- 기존 계층 구조, config 로딩 방식, logger 사용 방식, graceful shutdown 패턴을 최대한 유지해라.

현재 프로젝트의 특징:
- `cmd/server/main.go` 중심의 API 서버 구조를 사용한다.
- `internal/app/service` 에 유스케이스가 있다.
- `internal/domain` 은 모델과 계약을 둔다.
- `internal/infra/config/config.go` 에 설정 로딩과 env override 가 있다.
- DB, SMTP, secret provider 등은 이미 infra 계층에 있다.

구현 목표:
이 프로젝트에 스케줄러 전용 실행기와 기본 스케줄링 인프라를 추가해라.

이번 작업의 핵심 범위:
1. API 서버와 별도로 동작하는 `cmd/scheduler/main.go` 를 추가해라.
2. `scheduler` 설정 섹션을 `configs/app.yaml`, `configs/app.example.yaml`, `internal/infra/config/config.go` 에 추가해라.
3. 최소한 아래 설정을 지원해라.
   - `enabled`
   - `poll_interval`
   - `timezone`
   - `runner_id`
   - `lock_provider`
   - `max_parallel_jobs`
   - `shutdown_timeout`
4. 스케줄러 러너를 `internal/infra/scheduler` 아래에 구현해라.
5. 러너는 interval 기반 polling 으로 due job 을 실행하는 구조를 가져라.
6. 작업 인터페이스를 `internal/domain/port` 또는 프로젝트 구조에 맞는 위치에 정의해라.
7. 작업 실행 이력 모델을 `internal/domain/model` 에 추가해라.
8. 실행 이력 저장 인터페이스를 두고, 초기 구현은 메모리 대신 DB 확장을 고려할 수 있는 형태로 만들어라.
9. 중복 실행 방지를 위한 락 인터페이스를 도입해라.
10. 현재 DB vendor 가 postgres 인 경우를 우선 고려하고, 락 구현은 최소한 확장 가능한 구조로 만들어라.

설계 요구사항:
1. 실제 도메인 작업을 실행하는 유스케이스와 스케줄러 오케스트레이션을 분리해라.
2. API 서버인 `cmd/server` 는 스케줄러를 직접 실행하지 않게 유지해라.
3. 스케줄러 프로세스는 graceful shutdown 을 지원해라.
4. 로그에는 작업명, 상태, 실행 시간 정도만 남기고 민감정보는 남기지 마라.
5. 향후 작업 종류가 늘어나도 구조가 버틸 수 있게 해라.

구체 요구사항:
1. 스케줄 작업 정의는 최소한 아래 속성을 고려해라.
   - job id
   - job name
   - enabled
   - next run time 또는 schedule
2. 작업 실행 결과는 최소한 아래 속성을 고려해라.
   - execution id
   - job id
   - started at
   - finished at
   - status
   - error message
3. 러너는 poll interval 마다 실행 가능 작업을 조회하고 실행해라.
4. 초기 단계에서는 cron 파서보다 interval polling 기반이 더 단순하면 그 방식을 선택해도 된다.
5. 다만 구조는 이후 cron 기반으로 확장 가능하게 유지해라.

DB 및 락 관련 요구사항:
1. 락은 인터페이스로 분리해라.
2. postgres 기준 advisory lock 또는 유사한 DB 기반 락 구조를 염두에 두고 설계해라.
3. 당장 완전한 모든 vendor 지원은 필요 없지만, vendor 분기 구조가 현재 프로젝트 스타일과 맞게 들어가야 한다.
4. 락을 획득하지 못한 경우 작업을 스킵하는 흐름을 가져라.

추천 파일 방향:
- `cmd/scheduler/main.go`
- `internal/infra/config/config.go`
- `internal/domain/model/scheduled_job.go`
- `internal/domain/model/job_execution.go`
- `internal/domain/port/job_runner.go`
- `internal/domain/port/job_execution_repository.go`
- `internal/domain/port/scheduler_lock.go`
- `internal/app/service/job_service.go`
- `internal/infra/scheduler/runner.go`
- `internal/infra/scheduler/postgres_lock.go`
- `configs/app.yaml`
- `configs/app.example.yaml`

작업 방식:
1. 먼저 현재 코드 패턴을 읽고 어떤 파일을 추가/수정할지 판단해라.
2. 그 다음 실제 코드를 수정해라.
3. 구현 후 `gofmt` 와 가능한 검증을 실행해라.
4. 마지막 응답에서는 아래만 간단히 정리해라.
   - 구현한 내용
   - 핵심 설계 포인트
   - 수정한 파일
   - 실행/테스트 방법
   - 남은 제한사항

주의:
- 불필요한 대규모 리팩터링 금지
- 기존 코드 스타일 최대한 유지
- 지금 단계에서는 스케줄링 기반 구조를 우선 만들고, 실제 도메인 작업은 1개 정도 연결 가능한 형태면 충분하다
- 설명보다 구현 우선
```
