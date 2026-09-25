# Tasks: Modify an Existing Project

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–550 authored changed lines |
| 400-line budget risk | High |
| Chained PRs recommended | Yes (risk signal only; delivery remains one PR) |
| Suggested split | Single PR; no chained slices planned; obtain `size:exception` approval before apply |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

The design rates the review-budget risk as high. This plan honors the parent-confirmed single-PR strategy and does not create a chained-PR plan. The maintainer explicitly approved `size:exception` before apply; no chained PRs are planned.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Deliver the complete US-02 replacement endpoint, validation, narrow PostgreSQL update, and associated unit/integration coverage as one reviewable feature | Single PR (`size:exception` approved) | `go test ./tests/unit/project/... ./tests/unit/cmd/api/...` | Run `go test ./...` with Docker available to exercise Testcontainers-backed PostgreSQL update and story-preservation scenarios; without Docker, record integration skips | Revert the US-02 update use case, route/handler wiring, repository update, and their tests; no schema or persisted-data rollback is needed |

## Phase 1: RED — Specify Failing Behavior Tests

- [x] 1.1 Add table-driven RED use-case tests in `tests/unit/project/application/update_project_test.go` for complete valid replacement, equal start/finish dates, missing/blank name, missing/invalid dates, finish-before-start, repository error, and `application.ErrProjectNotFound`; assert invalid commands make zero repository calls.
- [x] 1.2 Add RED composed-handler routing tests in `tests/unit/project/transport/http/handler_test.go` proving `PUT /projects/{project_id}` is registered, a malformed UUID is rejected before any repository write, a well-formed unknown UUID returns 404 without creating or modifying data, and existing project/story routes remain available.
- [x] 1.3 Add RED HTTP request/response tests in `tests/unit/project/transport/http/handler_test.go` for a successful full replacement and its returned ID/basic fields; reject malformed JSON, trailing JSON values, unknown JSON fields, each missing field, blank name, malformed dates, and finish-before-start with the existing JSON error format and no write.
- [x] 1.4 Add RED API composition coverage in `tests/unit/cmd/api/main_test.go` confirming the composed API exposes the update route while existing routes continue to work; extend test fakes only as needed for the new repository method.
- [x] 1.5 Add RED PostgreSQL cases in `tests/integration/project/postgres/repository_integration_test.go` for updating exactly the three basic columns, retaining the project ID, returning not found for an unknown ID, and preserving associated story data after applying migrations `000001` and `000002`.

## Phase 2: GREEN — Implement Application and Persistence Behavior

- [x] 2.1 Extend `ProjectRepository` in `internal/project/application/create_project.go` with `Update(context.Context, domain.Project) error`, and add `ErrProjectNotFound` in the application package; make the tests from 1.1 compile and pass for sentinel propagation.
- [x] 2.2 Create `internal/project/application/update_project.go` with `UpdateProjectCommand` and `UpdateProjectUseCase`; parse both ISO dates with the existing `parseDate`, construct the full replacement with `domain.NewProject`, and call `Update` only after all validation succeeds.
- [x] 2.3 Implement the repository update in `internal/project/infrastructure/postgres/repository.go` using a parameterized `UPDATE` assigning only `name`, `start_date`, and `planned_finish_date` for the supplied ID; map zero affected rows to `application.ErrProjectNotFound` and preserve other database errors.
- [x] 2.4 Implement update request handling in `internal/project/transport/http/handler.go`: strictly decode one complete request object, parse the route `project_id` UUID, execute the use case, return HTTP 200 with the existing project response shape, map malformed body to 400, malformed UUID/validation to 422, not found to 404, and unexpected persistence errors to 500 without leaking repository details.
- [x] 2.5 Wire the update use case and `PUT /projects/{project_id}` registration in `internal/api/api.go`; make the routing RED cases from 1.2 and composition case from 1.4 pass without changing existing POST or story routes.

## Phase 3: REFACTOR — Verify Contracts and Integration

- [x] 3.1 Refactor the new tests in `tests/unit/project/application/update_project_test.go` and `tests/unit/project/transport/http/handler_test.go` to share existing test helpers and use table-driven cases where behavior is common, retaining explicit assertions for validation errors and zero persistence calls.
- [x] 3.2 Review `internal/project/infrastructure/postgres/repository.go` and its integration assertions in `tests/integration/project/postgres/repository_integration_test.go` to ensure the update SQL touches only the three specified columns and tests verify project identity and related story preservation.
- [x] 3.3 Run `go test ./tests/unit/project/... ./tests/unit/cmd/api/...` and resolve all focused unit failures.
- [x] 3.4 Run `go test ./...`; verify Testcontainers-backed integration scenarios with Docker available, or document the Docker-dependent skips if Docker is unavailable.
- [x] 3.5 Review the final authored additions plus deletions against the 400-line budget before implementation completion; retain the complete test coverage and scope rather than compressing or deleting tests, and do not proceed with this single-PR delivery unless `size:exception` has explicit maintainer approval.
