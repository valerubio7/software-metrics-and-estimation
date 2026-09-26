# Apply Progress: Create a Sprint and define its Sprint Goal

## Status

- Change: `us-08-create-sprint`
- Artifact store: OpenSpec
- Mode: Strict TDD
- Delivery: accepted `size:exception`; one PR boundary from `feat/us-08-create-sprint` to `main`, no child branches
- Work units: Unit 1 — Sprint domain/application contracts; Unit 2 — PostgreSQL schema and persistence
- Tasks complete: 1.1–1.6, 2.1–2.4 (10 of 24 total)
- Unit 1 commit: `e9dd5ff feat(sprint): add domain and creation use case`
- Unit 1 commit review-budget impact: 283 authored lines added, 0 deleted. The maintainer-approved `size:exception` remains the delivery mode for the aggregate PR.

## TDD Cycle Evidence

| Task | Test file | Layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1 | `tests/unit/sprint/domain/sprint_test.go` | Unit | N/A (new package) | `go test ./tests/unit/sprint/domain` failed as expected because the Sprint domain package did not exist | `go test ./tests/unit/sprint/domain` passed after adding the domain constructor | Table-driven valid goals preserve surrounding whitespace and arbitrary long/punctuated content; missing and whitespace-only goals fail | Domain refactor/formatting retained behavior; package test passed |
| 1.2 | `tests/unit/sprint/domain/sprint_test.go` | Unit | N/A (new production file) | Same failing domain test written first for the absent package/constructor | Domain package tests passed after implementing `Sprint`, `ValidationError`, and `NewSprint` | Both accepted-input cases and both invalid-goal cases exercise distinct paths | Covered by task 1.3 test run |
| 1.3 | `tests/unit/sprint/domain/sprint_test.go` | Unit | N/A (new files) | Reused the preceding RED contract; no separate behavior was introduced | Existing domain tests passed | All specified domain cases remain covered | `gofmt -w internal/sprint/domain/sprint.go tests/unit/sprint/domain/sprint_test.go && go test ./tests/unit/sprint/domain` passed |
| 1.4 | `tests/unit/sprint/application/create_sprint_test.go` | Unit | N/A (new package) | `go test ./tests/unit/sprint/application` failed as expected because the Sprint application package did not exist | `go test ./tests/unit/sprint/application` passed after implementation | Success, two invalid goal forms, and repository error paths are covered | Covered by task 1.6 test run |
| 1.5 | `tests/unit/sprint/application/create_sprint_test.go` | Unit | N/A (new production file) | Application tests written first against the absent use case and contract | Application package tests passed after implementing the use case | Verifies exactly one write with generated ID and unchanged inputs, zero ID generation/writes on invalid goals, and error propagation | Covered by task 1.6 test run |
| 1.6 | `tests/unit/sprint/application/create_sprint_test.go` | Unit | N/A (new files) | Reused the preceding RED contract; no separate behavior was introduced | Combined domain/application tests passed | All specified application scenarios remain covered | `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application` passed; final verbose run also passed |

## Work Unit Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test -v ./tests/unit/sprint/domain ./tests/unit/sprint/application` — exit 0; domain 2 top-level tests plus 4 subtests passed; application 3 top-level tests plus 2 subtests passed; both packages reported `ok`. |
| Runtime harness command/scenario and exact result | N/A — this domain/application work unit has no external runtime boundary; it introduces no HTTP handler, process composition, database adapter, or service to launch. |
| Rollback boundary | Revert `internal/sprint/domain/`, `internal/sprint/application/`, `tests/unit/sprint/domain/`, and `tests/unit/sprint/application/`; this removes only the Sprint contracts/use case and their tests, with no API composition changes. |
| Commit identity | `e9dd5ff feat(sprint): add domain and creation use case` on `feat/us-08-create-sprint`; local commit only, no branch creation, push, or PR. |

## Deviations and Issues

- None. The implementation preserves the original Sprint Goal, rejects only empty/whitespace-only goals, validates before ID generation and persistence, and performs one repository write.
- Unit 2 is recorded below; Unit 3 was not performed.

## Unit 2 — PostgreSQL Schema and Persistence

- Current status: complete; Unit 2 tasks 2.1–2.4 are complete (10 of 24 total tasks complete).
- Delivery remains the maintainer-approved `size:exception` on `feat/us-08-create-sprint` targeting `main`; no child branches, push, or PR were created.
- Only disposable Testcontainers databases were migrated; no persistent user database or data was changed, and migration down was not run.

### TDD Cycle Evidence — Unit 2

| Task | Test file | Layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 2.1 | `tests/integration/sprint/postgres/repository_integration_test.go` | PostgreSQL integration | N/A (new package) | Test-first run failed during Go package setup because `internal/sprint/infrastructure/postgres` did not exist; no test body executed, so this was a structural RED, not a behavioral assertion. Docker was confirmed reachable at the authorized user socket. | After schema/repository implementation, both integration tests passed on real PostgreSQL 16. | Real database scenarios verify exact persisted values, absent-project FK, and duplicate-primary-key error not mislabeled as missing project. | No behavior change needed; fresh uncached focused run passed after gofmt. |
| 2.2 | `tests/integration/sprint/postgres/repository_integration_test.go` | PostgreSQL integration | N/A (new migration) | Test was written before schema creation; initial package setup structural RED prevented its SQL assertions from executing. | Both real PostgreSQL scenarios passed with 000003 applied; schema/FK and exact fields were verified. | Migration has only the specified three columns and named `ON DELETE RESTRICT` FK. | Up/down pair reviewed; down drops only `sprints`; 000001/000002 remain unchanged. |
| 2.3 | `tests/integration/sprint/postgres/repository_integration_test.go` | PostgreSQL integration | N/A (new repository) | Test imports the not-yet-existing repository package; initial Go setup failed structurally before test execution. | Both real PostgreSQL scenarios passed; named SQLSTATE 23503 maps to `ErrProjectNotFound`, duplicate-primary-key SQLSTATE 23505 remains a PostgreSQL error. | Test covers valid persistence, missing project and a distinct constraint failure. | Adapter follows the existing story PostgreSQL pattern and propagates other errors. |
| 2.4 | `tests/integration/sprint/postgres/repository_integration_test.go` | PostgreSQL integration | N/A (new files) | Reused the test-first integration contract; structural RED recorded above, without claiming an earlier behavior assertion. | `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 -v ./tests/integration/sprint/postgres -run 'TestSprintRepository'` passed both tests against ephemeral PostgreSQL 16 containers. | FK definition/behavior and duplicate-PK distinction were verified; unrelated errors were not misclassified. | `gofmt` applied; uncached focused runtime test passed after review. |

### Work Unit 2 Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 -v ./tests/integration/sprint/postgres -run 'TestSprintRepository'` — exit 0; both named tests passed (`ok`, 3.965s). |
| Runtime harness command/scenario and exact result | Same command connected Testcontainers to `unix:///run/user/1000/docker.sock`, started local `postgres:16-alpine`, applied migrations 000001 and 000003 to disposable test databases, and passed both persistence/FK cases. No image pull or persistent user database access occurred. |
| Rollback boundary | Revert the new sprint repository, both 000003 migration files, and the sprint PostgreSQL integration test; preserve Unit 1 files. Never run migration down automatically or delete any applied schema/data set; retain migration and persisted Sprints if a deployment has applied it. |
| Authored line count | 302 changed lines in the Unit 2 commit (298 additions, 4 checkbox replacements counted as deletions); this is below 400, while aggregate delivery remains the explicitly accepted `size:exception`. |
| Commit identity | One Conventional Commit for the proven Unit 2 behavior; local only, no remote operation. |

### Unit 2 Initial Attempt and Resolution

The initial test-first attempt established only a structural compile/setup RED: the integration file referenced the new repository package before it existed, so no test body ran. The authorized local Docker service/socket was then verified, implementation proceeded, and the uncached real-PostgreSQL tests passed. Unit 3 remains out of scope.
