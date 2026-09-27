# Local CI First 개발 지침

GitHub Actions 사용량과 Artifact Storage를 줄이기 위해, Claude Code / Codex Agent는 **GitHub CI보다 로컬 검증을 우선**한다.

## 기본 원칙

1. 작업 시작 시 `.github/workflows/`를 확인해 GitHub Actions가 수행하는 검증 명령을 파악한다.
2. 가능한 검증은 모두 로컬에서 먼저 실행한다.
   - build
   - unit/integration test
   - race test
   - lint/vet/format
   - frontend build/test
   - OpenAPI/migration 검증
   - E2E/Playwright
3. 프로젝트에 `verify.sh`, `make ci`, `make verify`, `scripts/ci.sh` 등이 있으면 이를 우선 사용한다.
4. 코드 수정 후에는 다음 순서로 검증한다.

```text
관련 테스트
→ Build
→ Static Check
→ Unit Test
→ 필요 시 Race/Integration/E2E
→ Local Full CI
→ Commit/Push
```

5. 로컬 검증 실패 상태에서 CI 확인 목적으로 반복 push하지 않는다.

```text
금지:
수정 → push → CI 실패 → 수정 → push

권장:
수정 → Local Test → 수정 → Local Full CI → push → GitHub CI
```

6. GitHub Actions가 실패한 경우에도 즉시 재push하지 말고, 실패한 Step의 명령을 로컬에서 재현·수정·검증한 뒤 push한다.

7. 가능하면 Local CI와 GitHub CI가 동일한 스크립트를 사용하도록 구성한다.

```text
Local Agent ─┐
             ├─> scripts/ci.sh
GitHub CI ───┘
```

8. Local CI 스크립트가 없다면 `.github/workflows/`를 분석해 `scripts/ci.sh` 또는 이에 준하는 검증 스크립트 생성을 검토한다.

9. GitHub Actions는 다음 용도로 제한한다.
   - 최종 통합 검증
   - main/PR merge 검증
   - release/tag 검증
   - 로컬에서 재현하기 어려운 환경 검증

10. CI 비용 절감을 위해 Workflow도 함께 점검한다.
   - `paths` / `paths-ignore`
   - `concurrency` + `cancel-in-progress`
   - push/PR 중복 실행
   - 불필요한 `upload-artifact`
   - 짧은 `retention-days`
   - Fast CI / Full CI 분리

11. 사용자의 명시적 요청 없이 다음 작업을 임의로 하지 않는다.

```bash
gh workflow run
gh run rerun
git tag
git push --tags
gh release create
```

## 완료 보고

작업 완료 시 실제 수행한 검증을 명확히 보고한다.

```text
Local Validation
- build: PASS
- test: PASS
- race: PASS / NOT RUN
- frontend build: PASS
- E2E: PASS / NOT RUN

GitHub Actions
- 실행 여부 및 결과

Git
- branch
- commit
- push 여부
```

실행하지 않은 검증은 절대 PASS로 표시하지 않는다.

### 최우선 규칙

> **GitHub Actions를 개발·디버깅 반복 루프로 사용하지 않는다.**  
> **가능한 모든 검증을 로컬에서 완료한 뒤 GitHub에 push한다.**