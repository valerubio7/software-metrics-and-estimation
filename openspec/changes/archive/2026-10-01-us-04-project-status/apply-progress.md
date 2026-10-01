# Apply progress: us-04-project-status

## Status and scope

Consumed the supplied native `gentle-ai.sdd-status` v2: `applyState=ready`, `nextRecommended=apply`, no blockers, repo-local workspace and authorized repository root. No fresh status RPC was needed because valid native status was supplied. Delivery forecast was High; proceeded under the user's explicit `single-pr` `size:exception` approval. No PR chain and no commit.

Strict TDD is active (`go test ./...`). Loaded global strict-TDD guidance from `/home/santioses/.pi/agent/npm/node_modules/gentle-pi/assets/support/strict-tdd.md`; no project override. Skill resolution: `fallback-path` (general Gentle AI skill; no injected phase path).

## Completed tasks

- [x] 1. RED — Domain table tests in `tests/unit/project/domain/status_test.go` cover start/end boundaries, single-day projects, and date-only comparison. Initial test run failed to compile because the expected API did not exist.
- [x] 2. GREEN — Pure three-value derivation in `internal/project/domain/status.go`; focused domain tests pass.
- [x] 3. RED — Added a PostgreSQL repository test for ID lookup, missing ID, and unchanged persisted values; test package passed but its container-dependent case was skipped because Docker is unavailable. Existing partial `GetByID` implementation was already present, so this added coverage did not demonstrate a RED failure. This is part of the documented strict-TDD ordering deviation, not compliant RED evidence.
- [x] 4. GREEN — Existing partial application reader/use case and PostgreSQL lookup/missing mapping reviewed and retained. Application unit behavior passes; PostgreSQL-backed test is present but runtime skipped due Docker denial.
- [x] 5. RED — Existing HTTP tests cover success, invalid UUID with zero reads, missing project, internal error, and the exact three-field success body. Added API composition coverage verifying GET route registration for a readable repository and preserving legacy behavior without one. Because the pre-existing prototype already implemented these behaviors, added tests passed immediately; this does not constitute a compliant RED phase. Historical TDD deviation retained.
- [x] 6. GREEN — Existing GET handler/DTO, API route registration, UTC date source, and repository read path retained; focused unit tests pass.
- [x] 7. TRIANGULATE — Domain date boundary cases, use-case missing case, HTTP error/success cases, and API route composition verified. Read path uses SELECT only; handler responds with identity/status only; no US-03 dependency. Repository unchanged-value integration assertion added; cannot execute against PostgreSQL without Docker.
- [x] 8. REFACTOR — Reviewed implementation without behavioral restyling; updated `openspec/specs/project/spec.md` with the approved status, not-found and read-only requirements. No stable API-documentation location beyond the specification was identified, so no separate API doc was created. Confirmed no migration or persisted `completed` state was added.

Persisted task artifact `tasks.md` now marks tasks 1–8 complete. Re-read it before return; every task line is `[x]` and no unchecked implementation task remains.

## Files changed

- `internal/project/domain/status.go` — derived status type and pure date-only classifier.
- `internal/project/application/get_project_status.go` — read-only reader port/use case.
- `internal/project/infrastructure/postgres/repository.go` — parameterized SELECT and `pgx.ErrNoRows` mapping.
- `internal/project/transport/http/handler.go` — GET handler, validation and response DTO.
- `internal/api/api.go` — register GET route when repository supports reads; use UTC clock.
- `tests/unit/project/domain/status_test.go`
- `tests/unit/project/application/get_project_status_test.go`
- `tests/unit/project/transport/http/status_handler_test.go`
- `tests/unit/cmd/api/main_test.go` — API composition behavior.
- `tests/integration/project/postgres/repository_integration_test.go` — lookup, absent ID, and unchanged stored value.
- `openspec/specs/project/spec.md`
- `openspec/changes/us-04-project-status/tasks.md`
- `openspec/changes/us-04-project-status/apply-progress.md`

Pre-existing/untracked user files were preserved, including `odd/tasks/us-04-project-status.md`.

## Verification

- Safety net before new test edits: `go test ./tests/unit/project/... ./internal/api ./tests/integration/project/postgres` passed.
- New composition and repository coverage was run. Initial composition assertion expected overdue despite changing the fixture to relative-to-current dates; observed `active`, corrected the expected result to the correct fixture behavior, then reran successfully.
- `go test ./tests/unit/project/... ./tests/unit/cmd/api ./tests/integration/project/postgres -count=1`: PASS. PostgreSQL Testcontainers case skipped by the repository's helper because Docker access is denied.
- `go test ./...`: FAIL only in container-based Sprint, Story, and API startup/integration suites because connection to `/var/run/docker.sock` is permission denied. The project PostgreSQL package exits successfully with its container-dependent case skipped; all unit packages pass.
- No database migration was added; no persisted project status or `completed` value was introduced.

## TDD Cycle Evidence

| Task | Test file | Layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1–2 | `tests/unit/project/domain/status_test.go` | Unit | Focused baseline passed | ✅ Initial compile failure on absent status API | ✅ Focused domain tests passed | ✅ Date boundaries, single-day and date-only cases | ✅ Reviewed; no needless changes |
| 3–4 | `tests/unit/project/application/get_project_status_test.go`; `tests/integration/project/postgres/repository_integration_test.go` | Unit + PostgreSQL integration | Focused package baseline passed | ⚠️ Coverage test added after prototype existed; no test-first RED failure. PostgreSQL case skipped without Docker. | ✅ Application test passes; repository implementation present | ✅ Missing ID and stored value assertions included (container runtime unavailable) | ✅ Reviewed read-only SELECT |
| 5–6 | `tests/unit/project/transport/http/status_handler_test.go`; `tests/unit/cmd/api/main_test.go` | Unit/composition | Project unit baseline passed | ⚠️ Existing HTTP prototype predated tests; new composition test passed immediately, not compliant RED. | ✅ HTTP and API composition tests pass | ✅ Invalid ID/no-read, missing, internal, exact response shape and readable/legacy route behavior | ✅ Reviewed route wiring |
| 7–8 | Above tests; `openspec/specs/project/spec.md` | Unit + integration coverage + spec | Focused checks pass | N/A — verification/documentation tasks | ✅ Focused checks pass | ✅ No write API/US-03 dependency; SQL persistence assertion added but Docker unavailable | ✅ Scope and schema reviewed |

Strict-TDD ordering deviation is explicit: the previous partial application/HTTP prototype preceded its tests. This application does not claim those cycles were strict-TDD compliant. New integration/composition test code was added before its final checks, but existing functionality meant those tests did not produce a RED failure.

## Workload / PR boundary

Single PR under the explicit `size:exception` approval. No chain. Narrow implementation/test/spec scope. Final added implementation, test, and spec lines total **360** (151 tracked additions plus 209 lines in new source/test files; this excludes task/progress bookkeeping). The completed source/test/spec slice is below 400 added lines by this count; initial forecast risk remains disclosed and no content was compressed to fit. No commit created.

## Remaining tasks

None. Tasks 1–8 are checked in the persisted task artifact. Native status consumed before work recommends `apply`; fresh native status was not obtained after implementation, so parent/orchestrator should refresh status before selecting the next phase. Task completion is persisted; do not infer archive/verify routing locally.
