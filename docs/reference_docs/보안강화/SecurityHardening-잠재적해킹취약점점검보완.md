# Security Hardening — 잠재적 해킹 취약점 점검 및 보완

`heartblast/able-rest-api` 전체 소스를 대상으로 **외부 공격자가 악용할 수 있는 잠재적 보안 취약점을 진단하고, 재현 가능한 항목은 직접 수정**하라.

## 우선 점검 영역

OWASP API Security 관점에서 최소 다음을 확인한다.

- 인증·인가 누락 및 BOLA/IDOR
- 관리자/민감 API 무인증 노출
- 입력 검증 부족, mass assignment, parameter pollution
- SQL Injection 및 동적 SQL
- SSRF, command/path/template injection
- SMTP header injection 및 메일 악용 가능성
- 과도한 request body / attachment / Base64로 인한 DoS·메모리 고갈
- rate limit / timeout / connection 제한 부재
- CORS, Host/Header 신뢰 문제
- HTTP request smuggling 관련 잘못된 header 처리
- 민감정보·credential·stack trace·내부 오류 노출
- 로그를 통한 secret/개인정보 유출 및 log injection
- `/openapi.json`, `/swagger/*`, `/health`, `/ready` 노출 위험
- Content-Type 및 JSON parsing 우회
- dependency 취약점과 위험한 설정 기본값
- 암호화/secret 처리 및 config file permission
- DB credential 및 TLS 검증
- 예측 가능한 ID나 enumeration 가능성
- 안전하지 않은 기본 설정 및 production fail-open

## 작업 방식

각 발견 항목을 다음 기준으로 분류한다.

```text
Critical / High / Medium / Low
공격 조건
영향
재현 경로
현재 방어 여부
조치
```

단순 이론적 가능성이 아니라 **현재 코드에서 실제 도달 가능한 공격 경로**를 우선한다.

Critical/High 및 수정 영향이 작은 Medium은 즉시 보완한다.

## 구현 원칙

- 기존 API 계약과 정상 동작을 최대한 유지
- 인증/인가가 없다면 임의의 새 인증 체계를 대규모 도입하지 말고 위험과 최소 안전조치를 구분
- 실패 시 secure-by-default / fail-closed
- 사용자에게 내부 오류·SQL·stack trace·secret을 반환하지 말 것
- 입력 크기, 배열 개수, attachment 크기 등 무제한 입력을 방치하지 말 것
- 보안 제한값은 config 가능하게 하되 안전한 기본값 사용
- 보안 조치를 우회하는 compatibility fallback 금지

## 회귀 테스트

수정한 취약점마다 공격 재현 테스트를 추가한다.

예:

```text
oversized body → 거부
비정상 JSON/content-type → 거부
메일 header injection → 거부
허용 범위를 넘는 attachment → 거부
잘못된 ID/query → 안전한 4xx
민감 내부 오류 → 일반화된 오류 응답
```

OpenAPI 계약에 새로운 제한이나 오류 status가 실제로 추가되면 `docs/openapi.yaml`도 동기화한다.

## 검증

최소 실행:

```bash
go test ./...
go vet ./...
make openapi-check
```

가능하면 Go dependency vulnerability scan도 수행하고 결과를 구분해서 보고한다.

## 완료 보고

1. 발견 취약점 목록과 위험도
2. 실제 공격 가능한 Critical/High 상세
3. 수정 내용
4. 추가한 보안 회귀 테스트
5. 남은 위험 및 운영 설정 권고
6. 전체 검증 결과

`docs/reference_docs/`는 수정하지 않는다.

검증 완료 후 변경사항을 커밋하되 **tag/release는 생성하지 않는다.**