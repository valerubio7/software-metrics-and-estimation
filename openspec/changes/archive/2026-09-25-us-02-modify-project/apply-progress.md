# Apply Progress: Modify an Existing Project

## State

- Apply status: complete; all 15 tasks are verified complete.
- Mode: Strict TDD.
- Delivery: single PR with explicitly approved `size:exception`; no chained PRs.
- Authored implementation diff: 480 additions + 14 deletions = 494 changed lines across source and tests. The total including this apply report and task-artifact edits is 582 additions + 45 deletions = 627; SDD initialization changes in `openspec/config.yaml` are excluded. The maintainer-approved `size:exception` applies.
- Verification: Go is available (`go1.27.1`). Required focused and full commands passed using the authorized rootless Docker config/socket; an uncached full-suite run also executed and passed both PostgreSQL integration packages.

## Task Progress

| Task | State | Notes |
|------|-------|-------|
| 1.1 | Complete | Application replacement, equal-date, validation/no-write, repository-error, and not-found tests pass in focused unit suite. |
| 1.2 | Complete | Composed API tests cover PUT registration, malformed UUID/no repository call, unknown ID/404, and existing project/story routes; focused unit suite passes. |
| 1.3 | Complete | HTTP success, strict JSON, missing/invalid fields, date ordering, response format, and no-write cases pass in focused unit suite. |
| 1.4 | Complete | API composition test confirms PUT route and existing routes; focused unit suite passes. |
| 1.5 | Complete | PostgreSQL update/story preservation and not-found integration scenario passed in the uncached full suite. |
| 2.1 | Complete | Repository contract and not-found sentinel implemented; application tests pass. |
| 2.2 | Complete | Update use case validates through shared date parser and domain invariant; application tests pass. |
| 2.3 | Complete | Narrow three-column update and zero-row not-found mapping passed the PostgreSQL integration test. |
| 2.4 | Complete | Strict request decoding, UUID validation, response, and error mapping pass focused HTTP tests. |
| 2.5 | Complete | PUT route registered in API composition; composed route tests pass. |
| 3.1 | Complete | Extracted shared update-command/request test helpers, retained table-driven cases, ran gofmt, and reran focused tests successfully. |
| 3.2 | Complete | Static review confirms SQL assigns only the three basic fields; integration assertions check project ID and associated story values, and passed in the uncached PostgreSQL run. |
| 3.3 | Complete | Required focused unit command passes across all four requested packages. |
| 3.4 | Complete | Required full suite passed with rootless Docker enabled; uncached execution also passed project and story PostgreSQL integration packages. |
| 3.5 | Complete | Reviewed scope and counted authored implementation diff at 494 changed lines; explicit `size:exception` is approved and recorded. |

All 15 tasks are marked `[x]` in `tasks.md`. The historical first apply had no Go toolchain; this record preserves that the tests were written before production code but their RED execution was blocked. No RED run is claimed retroactively. Current GREEN and post-refactor results, including the actual PostgreSQL integration execution, are observed below.

## TDD Cycle Evidence

| Task | Test file(s) | Layer | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|------|--------------|-------|------------|-----|-------|-------------|----------|
| 1.1 / 2.1 / 2.2 | `tests/unit/project/application/update_project_test.go` | Unit | Prior attempt: blocked (`go: command not found`); continuation focused suite: passed | Tests written before production; RED execution remained blocked in prior attempt | Passed in focused suite | Different-date/equal-date success, seven invalid-input cases, repository error and not-found propagation all pass | Extracted `validUpdateCommand`; focused suite passed again |
| 1.2 / 1.4 / 2.5 | `tests/unit/cmd/api/main_test.go` | API composition | Prior attempt: blocked; continuation focused suite: passed | Initial route test was written before production; no RED execution observed. Additional invalid/unknown-ID route cases were added during resume after implementation | Passed in focused suite | Success route, malformed UUID before repository call, not-found mapping, and existing create/story routes covered | gofmt and focused suite passed; no production refactor |
| 1.3 / 2.4 | `tests/unit/project/transport/http/update_handler_test.go` | HTTP unit | Prior attempt: blocked; continuation focused suite: passed | Tests written before production; RED execution remained blocked in prior attempt | Passed in focused suite | Equal dates, malformed/trailing/unknown-field bodies, each missing field, blank name, invalid date, finish-before-start, UUID/not-found/internal errors covered | Extracted `executeUpdateRequest`; gofmt and focused suite passed |
| 1.5 / 2.3 | `tests/integration/project/postgres/repository_integration_test.go` | PostgreSQL integration | Prior attempt: blocked; continuation uncached suite executed | Test written before production; RED execution remained blocked in prior attempt | Passed in uncached full suite | Project ID, three basic fields, associated story preservation and unknown ID assertions passed against PostgreSQL | SQL statically reviewed; no production refactor was needed |
| 3.1 / 3.3 | Focused project/API unit suites | Unit | Baseline on continuation: 4 requested packages passed | Historical RED executions not observed | Passed | Application/HTTP/API cases listed above passed across all four packages | Extracted shared test helpers, ran gofmt, then focused suite passed |
| 3.4 | `go test ./...` | Full suite | N/A | N/A | Required exact command passed (Go test cache); uncached full-suite run also passed | Both PostgreSQL integration packages executed successfully in the uncached run | No further refactor was needed after green suite |

## Work Unit Evidence

| Evidence | Result |
|----------|--------|
| Focused test | `DOCKER_CONFIG="$HOME/.config/docker-rootless" DOCKER_HOST="unix:///run/user/$(id -u)/docker.sock" go test ./tests/unit/project/... ./tests/unit/cmd/api/...` — exit 0; all 4 requested packages passed. |
| Runtime harness | `DOCKER_CONFIG="$HOME/.config/docker-rootless" DOCKER_HOST="unix:///run/user/$(id -u)/docker.sock" go test ./...` — exit 0. The exact command used the test cache; an uncached `... go test -count=1 ./...` run also exited 0, with `tests/integration/project/postgres` passing in 6.493s and `tests/integration/story/postgres` passing in 25.803s. |
| Rollback boundary | Revert US-02 changes in `internal/api/api.go`, `internal/project/application/create_project.go`, `internal/project/application/update_project.go`, `internal/project/infrastructure/postgres/repository.go`, `internal/project/transport/http/handler.go`, and the related project/API tests in `tests/unit/` and `tests/integration/project/postgres/repository_integration_test.go`; no schema/data rollback is needed. |
| Diff hygiene | `git diff --check` — exit 0, no output after gofmt and the test-helper refactor. |

## Changed Files

- `internal/api/api.go`
- `internal/project/application/create_project.go`
- `internal/project/application/update_project.go`
- `internal/project/infrastructure/postgres/repository.go`
- `internal/project/transport/http/handler.go`
- `tests/unit/project/application/create_project_test.go`
- `tests/unit/project/application/update_project_test.go`
- `tests/unit/project/transport/http/handler_test.go`
- `tests/unit/project/transport/http/update_handler_test.go`
- `tests/unit/cmd/api/main_test.go`
- `tests/integration/project/postgres/repository_integration_test.go`

`openspec/config.yaml` was updated during SDD initialization after the initial clean worktree check; it was not edited during this apply.

## Historical Blocker and Current Follow-up

The first apply was unable to run tests because Go was absent; tests were written before production code but RED was not observed. In a later resume without the rootless Docker environment variables, tests failed to create a Docker provider. The user then authorized rootless Docker setup; using the isolated config and socket, the required full suite and an uncached full suite passed. No production implementation changes were needed during this continuation; only test helpers and progress evidence were updated.
