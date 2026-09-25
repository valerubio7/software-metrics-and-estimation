# Design: Modify an Existing Project

Add a full-replacement project update endpoint using the existing HTTP, application, domain, and PostgreSQL boundaries. The design implements the `project` delta requirements while leaving project creation, story creation, and project status behavior unchanged.

## Technical Approach

Register `PUT /projects/{project_id}` in the existing `http.ServeMux`. The handler strictly decodes the complete basic-data JSON object and obtains the target ID from the route. The update use case parses ISO dates using the existing application date parser, validates the resulting complete project through `domain.NewProject`, then performs one repository update. Validation happens before the repository call. PostgreSQL updates only `name`, `start_date`, and `planned_finish_date`; zero affected rows maps to a project-not-found error. Successful updates return HTTP 200 with the existing project response shape and the route ID.

This follows the proposal’s full-replacement contract and the delta scenarios for equal dates, actionable validation failures, missing IDs, and preservation of data beyond the three basic fields. No schema change is needed: the existing projects table already stores these fields and enforces the date-order constraint.

## Architecture Decisions

### Decision: Reuse project invariants and date parsing

**Choice**: Add an `UpdateProjectUseCase` in `internal/project/application/update_project.go`. Reuse the package-level `parseDate` helper and `domain.NewProject` to validate the complete replacement before persistence.

**Alternatives considered**: Put parsing and validation in the HTTP handler, or introduce a separate update-only domain validator.

**Rationale**: Validation belongs at the application/domain boundary, not in transport. `NewProject` already enforces nonblank names, required dates, and finish-date ordering for the same three-field representation. Reusing it avoids divergent create/update rules and avoids unnecessary domain API changes.

### Decision: Extend the project repository contract with update and not-found semantics

**Choice**: Add `Update(context.Context, domain.Project) error` to `application.ProjectRepository` and define `application.ErrProjectNotFound`. The PostgreSQL implementation checks `RowsAffected()` and returns that sentinel when the ID matches no row.

**Alternatives considered**: Add a second repository interface and inject it separately, or treat an update affecting zero rows as success.

**Rationale**: The existing API composition already receives one project repository for all project operations. One extended contract keeps construction straightforward and lets use-case and handler tests inject one fake. Returning not found from the persistence boundary distinguishes a missing row from a database failure without a read-before-write race.

### Decision: Use the route ID as the update identity and keep the update statement narrow

**Choice**: Parse `project_id` as a UUID at the HTTP boundary, pass its canonical string to the use case, and use it as the `WHERE id = $1` parameter. The SQL statement assigns only the three basic-data columns.

**Alternatives considered**: Accept an ID in the body, load and replace an entire project record, or update additional columns in anticipation of later user stories.

**Rationale**: The endpoint is explicitly addressed by ID and the caller already knows that ID. A narrow parameterized `UPDATE` cannot replace the primary key or touch story/member data. This also avoids adding listing or selection behavior, which is out of scope.

### Decision: Match existing HTTP error and response conventions

**Choice**: Return `200 OK` with the existing project response fields on success; return `400` for malformed JSON, `422` for malformed UUIDs or domain validation failures, `404` for a well-formed but unknown UUID, and `500` for unexpected persistence errors. Reuse the existing JSON error shape and avoid exposing repository details.

**Alternatives considered**: Return `204 No Content`, or introduce a new error format/status convention.

**Rationale**: Returning the updated representation makes the replacement outcome explicit and aligns with existing project response serialization. Existing handlers already use JSON error objects and `422` field errors; the story handler provides the UUID path parsing and not-found mapping precedent.

## Data Flow

```text
PUT /projects/{project_id}
  → ServeMux route
  → UpdateProjectHandler: strict JSON decode + UUID parse
  → UpdateProjectUseCase: parse dates + validate complete Project
  → ProjectRepository.Update
  → PostgreSQL UPDATE of name/start_date/planned_finish_date only
  ← updated project representation (200), or mapped validation/not-found/error response
```

Malformed body or UUID and invalid project data stop before persistence. The update operation is one SQL statement; an unknown UUID produces zero affected rows and is reported as not found. No project is created as a fallback.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/api/api.go` | Modify | Construct the update use case and register `PUT /projects/{project_id}` alongside the existing project route. |
| `internal/project/application/create_project.go` | Modify | Extend the shared `ProjectRepository` contract with the update operation. |
| `internal/project/application/update_project.go` | Create | Define the replacement command, use case, and project-not-found sentinel; parse dates, validate via the existing domain constructor, and persist only valid projects. |
| `internal/project/transport/http/handler.go` | Modify | Add the update handler/request flow, UUID parsing, result/error mapping, and reuse the existing response and JSON helpers. |
| `internal/project/infrastructure/postgres/repository.go` | Modify | Implement a parameterized update limited to three basic columns and map zero affected rows to not found. |
| `tests/unit/project/application/update_project_test.go` | Create | Table-test successful replacement, equal dates, validation cases with no write, and repository error/not-found propagation. |
| `tests/unit/project/transport/http/handler_test.go` | Modify | Test route-level request validation, success/error status and payloads, and that invalid requests do not reach persistence. |
| `tests/unit/cmd/api/main_test.go` | Modify | Extend project fakes and verify API composition registers the update route without disturbing existing routes. |
| `tests/integration/project/postgres/repository_integration_test.go` | Modify | Verify update values, unchanged project ID and associated story data, and not-found behavior against PostgreSQL. Apply the existing stories migration in the preservation scenario. |

No production changes are planned for `internal/project/domain/project.go`: its current constructor already represents the invariant required for updates. No migration file changes are planned.

## Interfaces / Contracts

The repository contract remains in the project application package and gains one operation:

```go
type ProjectRepository interface {
    Create(ctx context.Context, project domain.Project) error
    Update(ctx context.Context, project domain.Project) error
}

var ErrProjectNotFound = errors.New("project not found")
```

The update command carries the path identity and all required replacement fields:

```go
type UpdateProjectCommand struct {
    ProjectID         string
    Name              string
    StartDate         string
    PlannedFinishDate string
}
```

HTTP request and success representation:

```http
PUT /projects/{project_id}
Content-Type: application/json

{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}
```

```json
{"id":"<project_id>","name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}
```

The request is a complete replacement, not a patch. All three JSON fields are required by validation; unknown fields, malformed JSON, and trailing JSON values are rejected by the existing strict decoder pattern. Missing/blank names and missing/invalid dates produce field-specific validation errors. Equal dates are valid. Repository update errors must not leak database details through HTTP.

## Testing Strategy

Strict TDD applies: write the focused failing tests first, implement to green, then refactor. Tests should be scenario-oriented and table-driven where cases share the same behavior.

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Domain/application unit | Full replacement validation, equal dates, each missing/blank/invalid field, finish-before-start, and no repository write on invalid values | Use the existing domain constructor and a recording fake repository; assert field errors and exact call count. |
| HTTP/route unit | `PUT` registration, success response, malformed body/extra fields, missing fields, invalid UUID, unknown ID, and persistence/internal errors | `httptest` against the composed handler; assert status, JSON fields, and that rejected input performs no write. Retain existing POST and story route coverage. |
| PostgreSQL integration | Three columns change, ID remains fixed, unknown ID is not found, and existing story association/data is unchanged | Extend the Testcontainers repository test, apply migrations 000001 and 000002 for the story-preservation case, then query persisted project and story rows. Docker is required; current helpers skip when Docker is unavailable. |
| Full suite | Cross-package compatibility after extending the repository interface | Run `go test ./...`; report Docker-dependent integration tests as skipped if Docker is unavailable. |

The expected authored scope includes a use case, transport and repository changes, and tests at three boundaries. This is a realistic **high risk** to the fixed 400 changed-line budget because the existing tests are explicit and a story-preservation integration scenario adds setup/assertions. Keep tests table-driven and focused on the specified contract; do not add unrelated flows. The delivery remains one PR as directed.

## Threat Matrix

This change adds HTTP routing, so routing applicability is addressed below. It does not add shell commands, subprocesses, VCS/PR automation, executable-file classification, or process integration. The supplied matrix’s rows are explicit N/A because their listed boundaries are not present; do not create tests or tasks for those N/A rows.

| Boundary | Applicability | Design response / safe and failure behavior | Planned RED tests |
|---|---|---|---|
| HTTP route registration and path identity (change-specific routing boundary) | Applicable | Match only `PUT /projects/{project_id}`; parse the path as UUID and reject malformed IDs before repository access; a valid but absent ID returns 404 without creating or modifying data. | Composed-handler tests prove the route is registered, invalid UUID has no write, unknown UUID returns 404/no write, and other existing routes remain intact. |
| Documentation-like paths | N/A — no file classification or execution boundary changes. | No design response required. | None. |
| Git repository selection | N/A — no Git command or repository-selection behavior changes. | No design response required. | None. |
| Commit state | N/A — no commit/index automation changes. | No design response required. | None. |
| Push state | N/A — no push or refspec automation changes. | No design response required. | None. |
| PR commands | N/A — no PR command construction or execution changes. | No design response required. | None. |

## Migration / Rollout

No migration required. The projects table already contains all fields and its date check constraint permits equal dates. The route is available whenever the existing API handler is composed; no feature flag or staged rollout is needed. Rollback is removal/revert of the application, route, repository operation, and tests; existing rows remain valid.

## Open Questions

None. The 400-line budget is a delivery risk to be monitored during task planning and implementation; it does not block the technical design.
