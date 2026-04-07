# 스케줄링 구조안

## 목적

이 문서는 현재 Go REST API 프로젝트에 핵심 도메인과 밀접한 스케줄링 기능을 추가할 때의 권장 구조를 정리한다.

이 프로젝트는 이미 다음 성격을 가지고 있다.

- `cmd/server` 중심의 API 서버 진입점
- `internal/app/service` 중심의 유스케이스 계층
- `internal/domain` 중심의 도메인 모델/계약
- `internal/infra` 중심의 DB, 설정, 시크릿, SMTP 등 인프라 구현

이 구조를 유지하면서 스케줄링을 넣으려면 "도메인 로직은 같은 코드베이스 안에 두고, 실행 프로세스는 API 서버와 분리"하는 방향이 가장 적절하다.

## 권장 방향

권장 구조는 다음과 같다.

- 스케줄 대상 유스케이스는 `internal/app/service` 에 둔다
- 스케줄 정의와 실행 상태 모델은 `internal/domain` 에 둔다
- 락, 실행 이력 저장, 시간 계산, 스케줄러 구현은 `internal/infra` 에 둔다
- API 서버는 `cmd/server` 로 유지한다
- 스케줄 전용 실행기는 `cmd/scheduler` 로 분리한다

즉, 같은 레포 안에서 도메인 규칙을 공유하되 프로세스 역할은 분리한다.

## 왜 API 서버와 분리해야 하는가

핵심 도메인과 관련이 깊더라도 스케줄러를 `cmd/server` 안에 직접 넣는 방식은 운영 리스크가 있다.

- API 인스턴스가 여러 대 뜨면 같은 작업이 중복 실행될 수 있다
- API 트래픽 처리와 배치성 작업의 자원 사용 패턴이 다르다
- 장애 분석 시 API 장애와 스케줄링 장애가 뒤섞인다
- 재시작, 배포, 롤백 시 작업 중복 또는 누락이 생기기 쉽다

따라서 "같은 프로젝트 안의 별도 실행기"가 가장 균형이 좋다.

## 목표 아키텍처

```text
able-rest-api/
  cmd/
    server/
      main.go
    scheduler/
      main.go
  internal/
    app/
      service/
        job_service.go
        <도메인별_스케줄_유스케이스>.go
    domain/
      model/
        scheduled_job.go
        job_execution.go
      port/
        scheduler_lock.go
        job_execution_repository.go
      repository/
        <도메인_리포지토리>.go
    infra/
      scheduler/
        runner.go
        cron_parser.go
        ticker_runner.go
        db_lock.go
        execution_repository.go
```

## 핵심 구성 요소

### 1. 도메인 모델

최소한 아래 개념이 필요하다.

- `ScheduledJob`
  - 작업 ID
  - 작업 이름
  - 활성화 여부
  - 스케줄 표현식 또는 interval
  - 타임존
  - 동시 실행 허용 여부
  - 최대 재시도 횟수
- `JobExecution`
  - 실행 ID
  - 작업 ID
  - 실행 시작 시각
  - 종료 시각
  - 상태
  - 에러 메시지
  - 실행 호스트 또는 인스턴스 식별자

### 2. 애플리케이션 서비스

`internal/app/service` 에는 다음 역할을 둔다.

- 어떤 스케줄 작업이 무엇을 하는지 정의
- 작업 실행 전 도메인 검증
- 실행 결과를 성공/실패로 정리
- 재시도 정책 판단
- 실행 이력 기록 요청

예시:

- `SendReservedMailJobService`
- `ExpirePendingOrdersJobService`
- `SyncPartnerDataJobService`

### 3. 스케줄러 러너

`internal/infra/scheduler` 에는 다음 역할을 둔다.

- 주기적으로 실행 대상 조회
- 다음 실행 시각 계산
- 락 획득
- 작업 실행
- 실행 이력 저장
- 실패 시 로그 기록

이 계층은 도메인 로직을 직접 가지지 않고, 서비스 계층을 호출하는 오케스트레이터 역할만 맡는다.

## 실행 방식 권장안

### 1안. interval 기반

가장 단순한 시작점이다.

- 예: 10초마다 due job 조회
- 현재 시각 기준으로 실행해야 할 작업만 선별

장점:

- 구현이 단순하다
- DB 기반 due job 조회와 잘 맞는다

단점:

- cron 식 표현의 유연성이 낮다

### 2안. cron 표현식 기반

도메인 요구가 복잡하면 cron 표현식 기반이 더 적합하다.

- 예: 매일 09:00
- 매주 월요일 00:05
- 매월 1일 01:00

장점:

- 운영자가 이해하기 쉽다
- 도메인 요구를 표현하기 좋다

단점:

- 파서 및 다음 실행 시각 계산이 필요하다

현재 프로젝트에는 처음에는 interval 또는 단순 cron 라이브러리 기반으로 시작하고, 이후 복잡도가 높아질 때 확장하는 접근을 추천한다.

## 중복 실행 방지 전략

스케줄러에서 가장 중요한 문제는 중복 실행 방지다.

권장 우선순위는 다음과 같다.

### 1. DB 락 기반 리더/작업 락

가장 현실적인 기본안이다.

- 작업 실행 전 DB 락 획득
- 락 획득 성공 인스턴스만 작업 수행
- 실행 종료 후 락 해제

PostgreSQL 을 기준으로는 advisory lock 같은 방식이 적합하다.

장점:

- 현재 프로젝트의 DB 의존 구조와 잘 맞는다
- 별도 Redis 같은 인프라가 없어도 된다

### 2. 실행 이력 테이블 + 상태 전이

락과 함께 보완적으로 사용한다.

- `pending -> running -> success/failed` 상태 관리
- 이미 `running` 인 작업은 중복 수행 방지

장점:

- 장애 분석과 운영 가시성이 좋아진다

### 3. 외부 분산 락

Redis, ZooKeeper 같은 외부 락은 지금 단계에서는 과하다.

## 권장 데이터 구조

최소한 아래 2개 테이블이 있으면 좋다.

### `scheduled_jobs`

- `id`
- `name`
- `job_type`
- `enabled`
- `schedule_expr`
- `timezone`
- `allow_concurrent`
- `max_retries`
- `next_run_at`
- `last_run_at`
- `updated_at`

### `job_executions`

- `id`
- `job_id`
- `status`
- `started_at`
- `finished_at`
- `error_message`
- `attempt`
- `runner_id`
- `created_at`

## 설정 구조 권장안

`configs/app.yaml` 에는 아래와 같은 `scheduler` 섹션을 추가하는 것이 좋다.

```yaml
scheduler:
  enabled: true
  poll_interval: "10s"
  timezone: "Asia/Seoul"
  runner_id: "scheduler-1"
  lock_provider: "postgres"
  max_parallel_jobs: 1
  shutdown_timeout: "30s"
```

환경변수 override 예시:

- `SCHEDULER_ENABLED`
- `SCHEDULER_POLL_INTERVAL`
- `SCHEDULER_TIMEZONE`
- `SCHEDULER_RUNNER_ID`
- `SCHEDULER_LOCK_PROVIDER`
- `SCHEDULER_MAX_PARALLEL_JOBS`
- `SCHEDULER_SHUTDOWN_TIMEOUT`

## 추천 구현 순서

### 1단계

- `scheduler` 설정 구조 추가
- `cmd/scheduler` 진입점 추가
- 기본 러너 루프 추가
- graceful shutdown 추가

### 2단계

- 스케줄 작업 도메인 모델 추가
- 실행 이력 저장 구조 추가
- 작업 인터페이스 정의

### 3단계

- DB 락 기반 중복 실행 방지 추가
- due job 조회 및 실행 추가

### 4단계

- 실제 핵심 도메인 작업 1개 연결
- 로그와 에러 처리 정리
- 테스트 보강

## 현재 프로젝트에 맞는 추천 파일 배치

### 새로 추가할 가능성이 높은 파일

- `cmd/scheduler/main.go`
- `internal/app/service/job_service.go`
- `internal/domain/model/scheduled_job.go`
- `internal/domain/model/job_execution.go`
- `internal/domain/port/job_runner.go`
- `internal/domain/port/job_execution_repository.go`
- `internal/domain/port/scheduler_lock.go`
- `internal/infra/scheduler/runner.go`
- `internal/infra/scheduler/postgres_lock.go`
- `internal/infra/scheduler/execution_repository.go`

### 수정 가능성이 높은 기존 파일

- `internal/infra/config/config.go`
- `configs/app.yaml`
- `configs/app.example.yaml`
- `README.md`

## 운영 원칙

- API 서버와 스케줄러는 별도 프로세스로 배포한다
- 실행 이력은 반드시 남긴다
- 작업 실패는 재시도 가능하도록 구조화한다
- 작업 본문이나 민감한 payload 는 로그에 남기지 않는다
- 타임존은 명시적으로 설정한다
- 다중 인스턴스 환경에서는 락 없는 스케줄링을 금지한다

## 이 프로젝트에 대한 최종 권장 결론

이 프로젝트에 스케줄링 기능을 넣는 것은 적절하다.

다만 권장 형태는 다음과 같다.

- 도메인 로직: 현재 프로젝트 내부
- 스케줄 실행기: `cmd/scheduler` 로 분리
- 중복 실행 방지: DB 락
- 실행 가시성: 실행 이력 테이블
- API 서버: 기존 역할 유지

이 방향이 현재 레포 구조, 운영 단순성, 향후 확장성 사이에서 가장 현실적인 선택이다.
