## Exploration: US-02 — Modify an existing project (`us-02-modify-project`)

### Current State
Projects currently support creation only: `POST /projects` is assembled in `internal/api/api.go`, validated by the project domain/use case, and inserted into PostgreSQL. The `projects` table stores exactly `id`, `name`, `start_date`, and `planned_finish_date`; both domain validation and a database check require the planned finish date to be on or after the start date. There is no project lookup/list or update operation. The user confirmed that the client will already have the existing project's ID, so US-02 does not require a project-listing or interactive selection flow. Stories reference projects with `ON DELETE RESTRICT`; project basic-field updates should not involve story records.

Existing project tests cover creation validation and PostgreSQL insertion, not update behavior. The repository uses `httptest` unit tests and Testcontainers integration tests; the latter require Docker and skip if it is unavailable. OpenSpec's project spec currently captures US-01 creation, so this change should add a focused project update requirement without changing creation behavior or adding member management.

### Affected Areas
- `internal/api/api.go` — register an update route alongside `POST /projects` in the current `net/http` composition.
- `internal/project/domain/project.go` — existing basic-field invariants can be reused or factored for validated updates.
- `internal/project/application/create_project.go` — contains creation-specific repository contracts; update needs an application operation and persistence boundary.
- `internal/project/transport/http/handler.go` — currently only decodes and handles project creation requests.
- `internal/project/infrastructure/postgres/repository.go` — currently implements only `INSERT`; updates must target only the three basic-data columns.
- `tests/unit/project/{domain,application,transport/http}/` — add validation, persistence-not-called-on-invalid-input, and HTTP behavior coverage following current test patterns.
- `tests/integration/project/postgres/repository_integration_test.go` — verify PostgreSQL changes the selected project's basic fields while retaining its ID and unrelated dependent story information.
- `openspec/specs/project/spec.md` and the new change's `specs/project/spec.md` — source spec and future delta for project update behavior.

### Approaches
1. **Full basic-data replacement with `PUT /projects/{project_id}`** — identify the project by path ID and require the complete basic-data representation (`name`, `start_date`, `planned_finish_date`) in the body.
   - Pros: unambiguous validation of the complete resulting state; aligns naturally with the existing required-field and date-consistency rules; explicit SQL update can preserve every other column or related record.
   - Cons: clients must send all three fields even when changing one; project existence/error response semantics need definition.
   - Effort: Medium

2. **Partial update with `PATCH /projects/{project_id}`** — accept only changed fields and validate the merged resulting project.
   - Pros: clients can change one basic field without resending the rest.
   - Cons: requires defining omitted-vs-empty semantics and loading current values before validating date consistency; the current repository has no read operation, so the flow and test surface expand.
   - Effort: High

### Recommendation
Prefer `PUT /projects/{project_id}` as a full replacement of the project's basic-data representation. It fits the currently small project model and ensures the date invariant is checked against the complete resulting state. Implement persistence as an explicit update of only `name`, `start_date`, and `planned_finish_date`; do not alter ID, story rows, members, or any unrelated data. Treat the path ID as the way an API client selects a project; a project-listing or UI selection flow is not present and should not be added implicitly. The proposal should make the missing-project response and the required-field update contract explicit while staying within US-02.

### Risks
- The issue does not define the response when the supplied project ID does not exist; the proposal should make this API behavior explicit.
- The issue does not define update semantics for omitted fields; this affects the API contract.
- A broad repository update could accidentally alter unrelated data; constrain the SQL to the three named basic fields and cover preservation in tests.
- PostgreSQL integration verification depends on Docker/Testcontainers availability; the configured full test command is `go test ./...`.

### Ready for Proposal
Yes. The user confirmed the client will already have the project ID, so propose a full basic-data replacement by ID and define the missing-project response. Keep project listing/interactive selection, US-03 member management, and US-04 status behavior out of scope.
