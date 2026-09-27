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
| Commit identity | `ed8aac9 feat(sprint): persist sprints in PostgreSQL` on `feat/us-08-create-sprint`; local only, no remote operation. |

### Unit 2 Initial Attempt and Resolution

The initial test-first attempt established only a structural compile/setup RED: the integration file referenced the new repository package before it existed, so no test body ran. The authorized local Docker service/socket was then verified, implementation proceeded, and the uncached real-PostgreSQL tests passed.

## Unit 3 — HTTP Contract, Compatible Composition, and Readiness

- Current status: complete; Unit 3 tasks 3.1–3.11 and Unit 4 tasks 4.1–4.3 are complete (24 of 24 total tasks complete).
- Delivery: accepted `size:exception`; one PR boundary from `feat/us-08-create-sprint` directly to `main`. No child branches, push, GitHub, or PR operation was performed.
- No migration down was run. All database tests used disposable local Testcontainers instances and the authorized Docker user socket.

### TDD Cycle Evidence — Unit 3

| Task | Test file | Layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 3.1 | `tests/unit/sprint/transport/http/handler_test.go` | HTTP unit | N/A (new handler tests) | Written first; initial focused run failed structurally because `internal/sprint/transport/http` did not exist, so no handler assertions ran. | Handler tests passed with 405 for non-POST and 422 for invalid UUID; both verify no repository write. | Other handler scenarios also exercise valid and invalid paths. | `go test ./tests/unit/sprint/transport/http ./tests/unit/cmd/api` passed after final formatting. |
| 3.2 | `tests/unit/sprint/transport/http/handler_test.go` | HTTP unit | N/A (new handler tests) | Written first; structural package failure as above, not reported as a behavioral assertion. | Malformed JSON, extra JSON value, unknown/client-owned fields return 400; absent and whitespace-only goals return 422 before writes. | Valid input preserves whitespace in the goal; invalid body and domain validation follow distinct response paths. | Focused unit command passed after final formatting. |
| 3.3 | `tests/unit/sprint/transport/http/handler_test.go` | HTTP unit | N/A (new handler tests) | Written first; structural package failure as above. | Valid request returns 201 with generated ID, canonical UUID, and exact goal; missing project maps to 404; unexpected failure maps to generic 500 without leaking its internal message. | Success, domain-invalid, missing-project, and unexpected-error branches are covered. | Focused unit command passed after final formatting. |
| 3.4 | `tests/unit/cmd/api/main_test.go` | API composition unit | Baseline passed: `go test ./tests/unit/cmd/api` | New composition assertions were written first; initial run failed to compile because `HTTPDependencies`, `SprintDependencies`, and the compatible composition function did not exist. | Route is absent without Sprint dependencies, POST registers with dependencies, GET/PUT/DELETE do not write, and existing project/story paths still work. | Project-only and story composition are additionally covered by the pre-existing tests. | Focused unit command passed. |
| 3.5 | `tests/unit/cmd/api/main_test.go` | Readiness unit | Baseline passed: `go test ./tests/unit/cmd/api` | Readiness cases were written before the resolver; initial run failed structurally because `ResolveMigrationReadiness` did not exist. | Clean v2 enables stories only; clean v3 enables both; dirty/lookup error enable neither migration-backed route; projects remain enabled in all cases. | Four schema states are table-driven and assert independent route flags. | Focused unit command passed. |
| 3.6 | `tests/integration/sprint/postgres/http_integration_test.go` | HTTP + PostgreSQL integration | N/A (new Sprint HTTP test) | Written before composition changes; initial integration compile failed because the Sprint composition API did not exist, so no DB assertion ran. | Docker-backed test passed 201 and exact HTTP/database field equality; missing project returns 404; rejected request adds no row. | The success path persists one Sprint; missing-project and validation failures preserve that count. | `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres` passed. |
| 3.7 | `tests/integration/story/postgres/http_integration_test.go` | Process + PostgreSQL integration | Baseline passed before edits: `DOCKER_HOST=unix:///run/user/1000/docker.sock go test ./tests/unit/cmd/api ./tests/integration/story/postgres` | Versioned process scenarios were extended before production composition; initial focused RED was a structural compile failure from absent dependency composition, not a behavioral test failure. | Real process harness passed v1, clean v2, clean v3, dirty, and lookup-error states; project route remained available throughout. | v2 yielded stories 201/Sprint 404; v3 yielded both 201; dirty and lookup error yielded both 404. | Docker-backed story integration package passed uncached. |
| 3.8 | `tests/unit/sprint/transport/http/handler_test.go` | HTTP unit | N/A (new handler) | All handler contracts preceded `handler.go`; structural RED noted above. | Minimal handler implementation passed focused tests for status, JSON, UUID, goal, repository errors, and response privacy. | Every handler branch has valid and alternate inputs; no speculative fields are accepted. | Refactored decoder/response handling remains strict and focused tests pass. |
| 3.9 | `tests/unit/cmd/api/main_test.go`; Sprint/story HTTP integration tests | API + integration | Unit and story integration baselines passed before edits. | Composition and integration tests first failed compilation on absent optional-dependency API. | Added explicit dependency composition while preserving `NewHTTPHandler` signature; optional Sprint route and prior project/story routes pass tests. | Tests exercise composition with and without Sprint and with retained story dependencies. | Focused unit and both Docker-backed integration packages pass. |
| 3.10 | `tests/unit/cmd/api/main_test.go`; `tests/integration/story/postgres/http_integration_test.go` | Readiness unit + process integration | Story process baseline passed before edits. | Version-state cases and process route expectations were written before readiness code; structural compile RED as above. | Single migration lookup is resolved independently: story >=2, Sprint >=3, only when clean; project composition is unconditional. | Process harness verifies v1/v2/v3, dirty, and lookup failure against PostgreSQL. | Full process readiness test passed against PostgreSQL 16 containers. |
| 3.11 | Handler/composition unit tests | HTTP/API regression | Focused baseline established for existing API/unit package before modifying it. | Reuses the test-first contract rows 3.1–3.5; no uncovered behavior was introduced. | `go test ./tests/unit/sprint/transport/http ./tests/unit/cmd/api` passed. | Existing project/story tests plus new Sprint scenarios all pass. | `gofmt` and the focused command passed after final test and implementation review. |
| 4.1 | `tests/integration/sprint/postgres/http_integration_test.go`; `tests/integration/story/postgres/http_integration_test.go` | PostgreSQL HTTP + process integration | Existing story integration baseline passed before changes. | Both integration contracts were added before production wiring; structural compile RED is recorded in 3.6–3.7. | API-to-PostgreSQL creation/404/no-write and process startup version scenarios passed. | Sprint creation leaves story count zero; existing story/project paths continue to work. | `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres ./tests/integration/story/postgres` passed. |
| 4.2 | Full Go suite | Repository regression | Existing relevant unit and integration baselines passed before implementation. | Not a separate behavior task; relies on the preceding test-first contracts. | `DOCKER_HOST=unix:///run/user/1000/docker.sock go test ./...` passed; no integration package was skipped. | All project, Sprint, and story unit/integration packages ran. | Full suite passed after final changes. |
| 4.3 | `README.md` | User-facing documentation | N/A (documentation change) | Not applicable: documentation has no executable behavior; acceptance was reviewed against the HTTP contract and migration design. | Documents external migration ordering, route, request/response, validation/errors, readiness, and destructive down risk. | N/A — documentation-only task, covered by integration-tested behavior. | Markdown diff reviewed; `git diff --check` passed. |

### Work Unit 3 Evidence

| Evidence | Result |
|---|---|
| Focused test command and exact result | `go test ./tests/unit/sprint/transport/http ./tests/unit/cmd/api` — exit 0; both packages passed. |
| Runtime harness command/scenario and exact result | `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres ./tests/integration/story/postgres` — exit 0; both packages passed against disposable PostgreSQL 16 containers. Covered exact Sprint HTTP/database persistence, absent-project 404, no write on rejected goal, no story assignment, and API process readiness for v1/v2/v3/dirty/lookup-error. The local image was used; no image pull or remote access occurred. |
| Full suite command and exact result | `DOCKER_HOST=unix:///run/user/1000/docker.sock go test ./...` — exit 0; all unit and integration packages passed; integration packages were not skipped. |
| Rollback boundary | Revert `internal/sprint/transport/http/`, `internal/api/api.go`, `cmd/api/main.go`, `tests/unit/sprint/transport/`, the Unit 3 changes in `tests/unit/cmd/api/main_test.go`, `tests/integration/sprint/postgres/http_integration_test.go`, the readiness/Sprint scenarios in `tests/integration/story/postgres/http_integration_test.go`, and the new Sprint README section/migration wording. Preserve `000003_create_sprints` and any persisted Sprint data; do not run migration down automatically. |
| Authored changed-line impact | 576 authored changed lines: 540 additions and 36 deletions, counting tracked diff plus the three new handler/test files. `size:exception` was explicitly accepted for one PR to `main`; no content was compressed or omitted to meet the default 400-line budget. |
| Commit identity | This evidence is included in the single Unit 3 Conventional Commit; its verified short ID is returned with the implementation report. No remote operations were authorized or performed. |

### Deviations and Issues — Unit 3

- None from the approved behavior/design. The unit-level RED executions initially failed structurally because the new Sprint handler/composition API did not exist; no behavioral assertion was claimed until the package compiled. One first GREEN integration attempt used a fixed generated Sprint UUID for both create and missing-project requests, which correctly triggered the primary-key constraint before the project FK; the integration harness was corrected to generate unique UUIDs per request, then passed.
- Migration readiness selection is a pure helper in `internal/api` and is exercised at both unit and real process/PostgreSQL integration layers. The process integration tests construct the required v3 schema directly for controlled readiness states; Sprint persistence integration applies the versioned 000001–000003 SQL migrations.
