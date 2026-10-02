# Apply progress — US-03 register project members

## TDD Cycle Evidence

| Work slice | RED evidence | GREEN evidence | TRIANGULATE / REFACTOR evidence |
|---|---|---|---|
| Migration chain and reconciliation | The original active migration directory caused local golang-migrate v4.19.1 to reject duplicate `000003` sources before SQL; incompatible `stories.status`, weakened checks, and partial-schema fixtures were recorded as expected fail-closed cases. | Disposable PostgreSQL 18.6 `valid_history_convergence.sh` now migrates a clean schema and both valid v4 histories through v6; `legacy_status_fail_closed.sh` passes. | Full matrix verifies dirty state, schema/data preservation, partial-sprints preflight, v5 reapplication idempotency, and that `down 1` removes only the v6 members table while retaining earlier data. |
| Member domain and HTTP contract | Domain/application/handler tests were authored before implementation and observed failing in the earlier apply cycles; failures are recorded in the session/apply history. | `go test ./tests/unit/projectmember/... ./tests/unit/cmd/api/...` passes. | Unit and PostgreSQL HTTP tests cover strict JSON/UUID handling, exact and NULL duplicate identity, batch success, 201/409/404 mapping, and readiness gates. |
| Member PostgreSQL persistence and readiness | Persistence, rollback, conflict, and readiness acceptance tests preceded their implementation; the earlier apply cycles record the RED results. | Member repository and HTTP integration suites pass in the disposable migration harness. | Tests cover project existence, atomic inserts/rollback, injected second-insert failure, concurrency, project-field preservation, and route availability only on clean schema >=6. |
| Shared-schema catalog defect and integration refactor | The shared-database full suite reproducibly failed with `incompatible partial story estimated-hours schema` because v5 catalog checks matched same-named relations in other schemas. | Namespace-qualified v5 checks and the shared loopback schema helper pass the complete `go test ./... -count=1` through `valid_history_convergence.sh`. | Independent verification reran the matrix and legacy negative case, confirmed all four DSNs/callers and cleanup, and checked shell syntax plus `git diff --check`. |

## Cumulative task/evidence reconciliation

Reconciled every task 1.1–9.2 against the complete proposal, both specs, design, cumulative history, repository evidence, and the parent-provided independent read-only verifier report. The prior 5/21 checkbox state was stale: all 21 implementation tasks are now marked complete in `tasks.md` because each task's clauses have implementation/test/documentation evidence in the current cumulative record and independent verification. This is local acceptance only, not production rollout approval.

### Evidence by task group

- **1.1–3.2, migration reconciliation:** `tests/integration/migrations/README.md`, `valid_history_convergence.sh`, `legacy_status_fail_closed.sh`, migration fixtures and disposable local PostgreSQL 18.6 / golang-migrate v4.19.1 evidence cover fresh install, both valid historical v4 states, status-incompatible fail-closed behavior, idempotent v5, data/schema preservation, dirty/readiness behavior, and the local-only runner caveat. The exact README-shaped local command is documented. Production runner/process, version, checksum, path and applied history remain unknown and block rollout; no production compatibility is claimed.
- **4.1–5.3, domain and HTTP:** prior strict RED evidence is recorded in the cumulative history; unit/domain and HTTP contracts cover validation, duplicates including NULL, strict JSON/UUID handling, response mapping and readiness. `go test ./tests/unit/projectmember/... ./tests/unit/cmd/api/...` passed in the independent report.
- **6.1–7.3, persistence/composition:** `tests/integration/projectmember/postgres/` covers project association/existence, NULL and duplicate identity, batch rollback/fault injection, concurrent conflict, and preservation of project fields. Member `23505` mapping is restricted to named constraints. API readiness closes for missing/dirty/error and opens only at the clean final version; HTTP integration asserts success/conflict/not-found. Independent verifier inspected member implementation and API composition.
- **8.1–8.2, triangulation:** domain boundaries and both spec contracts are covered by unit tests; disposable migration matrix and full tests inspect preservation, fail-closed routing, and project integrity. The parent verifier confirmed fresh and both v4 histories, partial-sprints fail-closed, idempotent v5, and four PostgreSQL integration suites.
- **9.1–9.2, refactor and final verification:** `tests/integration/testpostgres/schema.go` is shared by all four PostgreSQL setup paths while preserving each exact DSN environment variable and Testcontainers fallback. Repository already has named `isMemberUniqueViolation`; parent independently inspected helper and all callers, down migrations and README. The down evidence is explicitly not a production-safe rollback claim and does not assert deletion of historical data. No production compatibility is inferred.

### Verification evidence (independent verifier report)

All PASS:

- `tests/integration/migrations/valid_history_convergence.sh` — disposable PostgreSQL 18.6 / golang-migrate v4.19.1; fresh install and both valid v4 histories; partial-sprints fail-closed; idempotent v5; four PostgreSQL integration suites; `go test ./... -count=1`; temporary server stopped.
- `tests/integration/migrations/legacy_status_fail_closed.sh` — GREEN; temporary server stopped.
- `go test ./tests/integration/migrations -count=1`
- `go test ./tests/unit/projectmember/... ./tests/unit/cmd/api/...`
- `bash -n tests/integration/migrations/*.sh`
- `git diff --check`

No tests were run by this artifact-only reconciliation executor; all commands above are reported from the independent verifier evidence supplied by the parent. No DB access, source edits, commit, or push occurred in this phase.

### Persisted task artifact

Updated `openspec/changes/us-03-register-project-members/tasks.md` from stale 5/21 to 21/21 checkboxes. Re-read task artifact after update and confirmed every task 1.1–9.2 is `[x]`; no unchecked task lines remain. Native status consumed from parent: schema v2, `nextRecommended=apply`, `applyState=ready`, repo-local scope authorized, no blockers. No fresh status was produced by this artifact-only executor. `actionContext` warnings: none.

### Remaining tasks / risks

No unchecked implementation tasks. **Rollout blocker (not an unchecked local implementation task):** identify and validate the actual production migration runner/process, version, checksum, migration path and applied schema history on each target environment before deployment. Local disposable tests do not prove production compatibility. This work did not access databases/environments or alter source code.
