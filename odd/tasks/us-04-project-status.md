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

- [x] 9. Run final formatting/diff checks and create the requested Conventional Commits.
- [x] 10. Add issue approval label under explicit user authorization, push the branch, and open the PR.
- [ ] 11. Monitor automated checks and await the user's PR approval.

## Commit and PR evidence

- `01fc617c53aaba03aebc0c3e49391a584954db98` — `feat(project): expose derived project status`.
- `91522bc9411bf2574e8b7a1388a102a5c000b1ab` — `docs(odd): record US-04 commit evidence`.
- Both authored by SantiagoMO3 <santiaguistico@gmail.com>.
- PR #70: https://github.com/valerubio7/software-metrics-and-estimation/pull/70; base `main`, `Closes #32`, one `type:feature` label. Issue #32 received `status:approved` only after the user explicitly authorized that label.
- PR currently has no reported status checks. The user retains approval/merge authority.

## Verification evidence

- `go test -count=1 ./tests/unit/...`: passed, 10 packages.
- Focused Go project/unit/API package checks: passed; Docker-dependent PostgreSQL integration skipped by its helper.
- Full `go test ./...`: blocked by permission denied on `/var/run/docker.sock` for existing container-based suites.
- `gofmt -d` on changed Go files and `git diff --check`: passed.
- The partial application/HTTP prototype preceded tests; strict-TDD ordering deviation is documented in the archived apply-progress.
- The initial feature commit contains 19 files and 771 insertions, above the 400-line reference; the user explicitly accepted single-PR `size:exception`.
- No commit has been merged; wait for user review.
