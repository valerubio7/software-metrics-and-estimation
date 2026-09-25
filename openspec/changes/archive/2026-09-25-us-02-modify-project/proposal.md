# Proposal: Modify an Existing Project

## Intent

Implement US-02 (GitHub issue #30) so a client that already knows a project ID can update its name, start date, and planned finish date. Updates must enforce the existing date invariant, persist valid changes, and leave all unrelated project and member/story data unchanged.

## Scope

### In Scope
- Add `PUT /projects/{project_id}` for replacing the complete basic-data representation (`name`, `start_date`, `planned_finish_date`) of the identified project.
- Reject invalid or incomplete data, including a planned finish date before the start date; return a reason and do not persist invalid changes.
- Persist valid changes to the selected project's three basic fields only; return not found for an unknown project ID.
- Cover validation, HTTP behavior, persistence, and preservation of unrelated data with unit and integration tests.

### Out of Scope
- Project listing, searching, or interactive selection; the caller already has the ID.
- Project creation (US-01), member management (US-03), and project status (US-04).
- Partial-update semantics (`PATCH`) or changes to project/story/member data models.

## Capabilities

### New Capabilities
- None.

### Modified Capabilities
- `project`: add the existing-project basic-data update behavior while preserving the current US-01 creation requirements. The specs phase should create a delta for `openspec/specs/project/spec.md`.

## Approach

Follow the existing Go HTTP/application/domain/PostgreSQL boundaries. Implement `PUT /projects/{project_id}` as a full replacement of the three required basic fields, validate the complete resulting state before persistence, and update only `name`, `start_date`, and `planned_finish_date` for the path ID. Use the conventional not-found response for an unknown ID. Keep error responses informative and ensure invalid input never reaches persistence. Follow strict TDD with focused unit/HTTP tests and PostgreSQL integration coverage; run `go test ./...` (integration tests require Docker/Testcontainers). Keep the implementation reviewable within the configured 400-changed-line budget in a single PR.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `internal/api/api.go` | Modified | Register the project update route. |
| `internal/project/domain/` and `internal/project/application/` | Modified | Reuse project invariants and add the update use case/repository contract. |
| `internal/project/transport/http/handler.go` | Modified | Decode and handle update requests and responses. |
| `internal/project/infrastructure/postgres/repository.go` | Modified | Update only the three basic-data columns and distinguish a missing project. |
| `tests/unit/project/` | Modified | Cover valid updates, invalid data, missing IDs, and no persistence on validation failure. |
| `tests/integration/project/postgres/repository_integration_test.go` | Modified | Verify updated fields and preservation of the project ID and related/unrelated data. |
| `openspec/specs/project/spec.md` | Modified in future specs/archive phase | Define the US-02 behavior as a delta without removing US-01 requirements. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| A broad update could overwrite unrelated data. | Medium | Restrict the update statement to the three basic fields and assert preservation in integration tests. |
| Full-replacement `PUT` requires clients to send all three fields, even for a one-field change. | Low | Make required-field/full-replacement behavior explicit in the API contract and validation errors. |
| PostgreSQL integration verification may be unavailable without Docker. | Medium | Run unit tests regardless; run and report `go test ./...` with Docker/Testcontainers when available. |
| The single-PR change could exceed the 400-line review budget. | Low | Keep scope limited to US-02 and monitor authored additions plus deletions during implementation; do not add unrelated flows. |

## Rollback Plan

Revert the single implementation PR. This change requires no schema migration; reverting the route, use case, repository update, and tests restores the previous creation-only behavior. Existing persisted project records remain valid and are not deleted by rollback.

## Dependencies

- Existing PostgreSQL project table and project date invariant; no external dependency or schema migration is expected.
- Docker is required to execute Testcontainers integration tests.

## Success Criteria

- [ ] A client can replace all three basic fields for a project by ID, and the successful response reflects the persisted values.
- [ ] Unknown IDs return not found; missing/invalid fields and `planned_finish_date < start_date` return an explanatory error and leave persisted values unchanged.
- [ ] Updating a project preserves its ID and all unrelated project, member, and story data.
- [ ] `go test ./...` passes, including PostgreSQL integration tests when Docker is available.
- [ ] The implementation is delivered in one PR with no more than 400 authored changed lines.
