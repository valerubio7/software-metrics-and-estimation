# ODD Work Record: US-04 Project Status

Change: `us-04-project-status`
Branch: `feat/us-04-project-status`
Archived SDD change: `openspec/changes/archive/2026-10-01-us-04-project-status/`

## Confirmed behavior

- Derive `planned` when current date < `start_date`.
- Derive `active` from `start_date` through `planned_finish_date`, inclusive.
- Derive `overdue` after `planned_finish_date`; never infer `completed`.
- Read-only project lookup returns identity and status; missing project is reported.
- No persisted status, schema migration, writes, or US-03 dependency.
- Strict TDD configured with `go test ./...`; full suite blocked by denied Docker socket.
- Delivery: single PR; user explicitly authorized `size:exception` if needed.
- SDD change archived; native status reports archived.

## Implementation tasks

- [x] 1. Add RED domain classification tests for date boundaries and fixed reference dates.
- [x] 2. Implement GREEN pure date-only status classification and pass domain tests.
- [x] 3. Add lookup coverage for found/missing project and read-only behavior.
- [x] 4. Implement project lookup use case and PostgreSQL SELECT.
- [x] 5. Add HTTP/API tests for success, invalid UUID, missing project, and internal errors.
- [x] 6. Implement GET route, response DTO, and production date source.
- [ ] 7. **Pending environment:** Obtain a passing full-suite run; `go test ./...` is blocked by permission denied on `/var/run/docker.sock`. Focused project/unit/API checks pass. Strict-TDD ordering deviation on partial app/HTTP prototypes is documented.
- [x] 8. Reconcile canonical spec under user approval and archive SDD.

## Delivery tasks

- [x] 9. Run final formatting/diff checks and create the requested Conventional Commit.
- [ ] 10. Open the PR after issue #32 has its required `status:approved` label. The issue is currently OPEN with no labels; do not self-apply approval.

## Commit evidence

- `01fc617c53aaba03aebc0c3e49391a584954db98` — `feat(project): expose derived project status` (19 files, 771 insertions; includes implementation, tests, canonical spec, archived SDD artifacts, and this work record).
- Authored by SantiagoMO3 <santiaguistico@gmail.com>.

## Current evidence

- `go test -count=1 ./tests/unit/...`: passed (10 packages; test-case count not emitted).
- Earlier full `go test ./...`: failed because existing container-based suites cannot access Docker; project PostgreSQL integration skipped.
- User approved replacing three abbreviated canonical status requirements with full delta blocks; archive report records the composition.
- Commit/PR not yet created. Repo-local Git author is SantiagoMO3 <santiaguistico@gmail.com>.
