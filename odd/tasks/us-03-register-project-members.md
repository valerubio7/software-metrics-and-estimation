# ODD task record: US-03 project members

## Goal
Complete US-03 under OpenSpec change `us-03-register-project-members`: register members against an existing project ID; require full name, allow optional email; atomic batches; exact duplicate rejection; 404 for missing project; preserve project basics.

## Constraints and decisions
- OpenSpec SDD selected; strict TDD runner is `go test ./...`.
- User authorized tests only against disposable loopback PostgreSQL 18.6 and local golang-migrate v4.19.1; this does not establish production compatibility.
- Production runner/version/checksum/path/applied history remain unknown; no persistent services or deployment credentials may be used and no rollout safety claim is allowed.
- One-PR `size:exception` explicitly accepted. User explicitly authorized Git commit(s); push/PR/merge are not authorized.

## Reconciled progress
1. Member domain/application/HTTP behavior — **done**; unit coverage includes validation, exact `(full_name,email)` duplicates including NULL, atomic rejection, malformed method/JSON/UUID and error mapping.
2. Migration reconciliation — **done for disposable histories**; duplicate active 000003 resolved by retaining historical sprint SQL under fixtures and using forward-only v5/v6. Fresh and both valid v4 histories converge; incompatible status/weak checks and malformed partial sprints fail closed; v5 reapplication is idempotent; `down 1` removes only the v6 member table while preserving earlier data.
3. PostgreSQL persistence — **done locally**; transactions check project existence, NULL-aware partial unique indexes, duplicate mapping, rollback, injected insert failure, concurrency and project-field preservation verified on disposable PostgreSQL.
4. Route/readiness — **done locally**; `POST /projects/{project_id}/members` requires member dependency and clean migration >=6. Unit readiness coverage includes old/clean/dirty/error/missing table/row; PostgreSQL HTTP integration confirms batch 201, duplicate 409 and missing-project 404.
5. Shared test helper/refactor — **done**; `tests/integration/testpostgres/schema.go` is integrated by all four PostgreSQL integration setup paths, keeps each DSN env name and Testcontainers fallback, and enforces loopback-only URLs. Existing `isMemberUniqueViolation` isolates SQLSTATE/named-index mapping.
6. Full regression / SDD Apply / Verify / Archive — **done**; all OpenSpec implementation tasks are reconciled to **21/21**. Independent verification passed the disposable matrix and focused checks. Native archive completed at `openspec/changes/archive/2026-10-01-us-03-register-project-members/`; canonical project spec received one additive requirement. Migration/harness work-unit commit `f2866ee6cb8db38a22e5680bfc095fb23fb7f0e0` is complete; project-member feature commit remains.

## Evidence
- `tests/integration/migrations/valid_history_convergence.sh` — PASS on a disposable loopback PostgreSQL 18.6 cluster with local golang-migrate v4.19.1; includes clean and both valid v4 histories, fail-closed partial schema, v5 idempotency, all PostgreSQL integration packages, and `go test ./... -count=1`. Cleanup stopped the server and removed its temporary directory.
- `tests/integration/migrations/legacy_status_fail_closed.sh` — PASS with disposable cluster cleanup.
- `go test ./tests/integration/migrations -count=1` — PASS.
- `go test ./tests/unit/projectmember/... ./tests/unit/cmd/api/...` — PASS.
- `bash -n tests/integration/migrations/*.sh` and `git diff --check` — PASS.
- Bare `go test ./... -count=1` without disposable DSNs fails before database tests because Testcontainers cannot create a Docker provider; all test packages were exercised through the local harness instead. Docker is not installed.
- No persistent DB, production credential, or production migration runner was used. Production runner/version/checksum/path/applied history remain unknown and block rollout.

## Finalization
- SDD Apply, Verify and Archive are complete; archived path: `openspec/changes/archive/2026-10-01-us-03-register-project-members/`.
- Full local disposable PostgreSQL matrix and independent verification passed as recorded above. Production rollout remains blocked pending the actual runner/version/checksum/path/applied-history evidence.
- User authorized local work-unit commit(s) on `feat/us-03-register-project-members`; push/PR/merge are not authorized. Git author identity is configured repository-locally from the user's exact values only. Migration/harness work-unit commit `f2866ee6cb8db38a22e5680bfc095fb23fb7f0e0` was created. Stage and verify the project-member feature work unit, create its commit, and leave the branch unpushed.
