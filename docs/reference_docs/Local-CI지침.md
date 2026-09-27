# Local CI First 개발 지침

Claude Code / Codex Agent는 GitHub Actions 사용량을 줄이기 위해 **가능한 모든 검증을 로컬에서 우선 수행**한다.

## 기본 원칙

1. `.github/workflows/`를 확인해 CI 검증 명령을 파악한다.
2. 다음 검증은 가능한 경우 로컬에서 먼저 수행한다.
   - build
   - test
   - race
   - lint/vet/format
   - frontend build/test
   - integration/E2E
   - OpenAPI/migration 검증
3. `verify.sh`, `make ci`, `scripts/ci.sh` 등이 있으면 우선 사용한다.
4. 권장 순서:

```text
코드 수정
→ 관련 테스트
→ Build/Static Check
→ Unit Test
→ 필요 시 Race/Integration/E2E
→ Local Full CI
→ Commit
→ Push
```

5. 다음 방식으로 GitHub Actions를 디버깅 루프로 사용하지 않는다.

```text
금지:
수정 → push → CI 실패 → 수정 → push

권장:
수정 → Local Test → 수정 → Local Full CI → push
```

6. GitHub CI 실패 시 실패 명령을 로컬에서 재현·수정·검증한 후 다시 push한다.
7. 가능하면 Local CI와 GitHub CI가 같은 `scripts/ci.sh`를 사용하도록 구성한다.
8. 불필요한 `upload-artifact`, 중복 push/PR workflow, 장기 retention을 줄이고 `concurrency + cancel-in-progress`를 적용한다.

## Git Sandbox 처리

`.git/index.lock` 또는 `.git` 쓰기가 sandbox에 의해 차단되면:

- 즉시 포기하지 말고 사용 가능한 권한 승인/상승 실행 방법으로 commit을 재시도한다.
- 권한이 허용되고 사용자가 push까지 요청했다면 commit 후 대상 브랜치로 push한다.
- 실제로 불가능한 경우에만 아래처럼 보고한다.

```text
Commit: BLOCKED
원인: sandbox가 .git/index.lock 생성을 차단
변경 파일: 작업 트리에 유지
Push: NOT RUN
```

다음처럼 실제로 수행하지 않은 작업을 미래에 할 것처럼 표현하지 않는다.

```text
승인된 환경에서 나중에 커밋 후 푸시하겠습니다.
```

## 최우선 규칙

> GitHub Actions를 개발 반복 루프로 사용하지 않는다.  
> 가능한 검증은 로컬에서 완료한 뒤 push한다.  
> Sandbox 제한이 있으면 권한 승인 방법을 먼저 시도하고, 불가능할 때만 BLOCKED로 보고한다.