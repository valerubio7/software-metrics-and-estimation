# TUI-04 — Sprint Operations

## Objective

Deliver the TUI-04 Sprint-operation workflows so users can create a Sprint with a goal, batch-assign same-project stories, create titled tasks with optional estimated hours for an assigned story, and mark an assigned story complete. Use the existing API for policy and business validation; the TUI must render and route API outcomes rather than duplicate those rules.

## Problem and rationale

The current TUI has project and backlog screens but no Sprint-operation screens. The API already supports Sprint creation, batch story assignment, task creation, and Sprint-scoped completion, including validation and conflict responses. However, it does not expose the read-only discovery needed to navigate coherently: there is no contract to list open Sprints for a project or to list stories assigned to a selected Sprint. The existing Sprint list handler serves closed Sprint history, so its meaning must remain intact.

Add only the minimal read-only discovery contracts for open Sprints and their assigned stories. This lets the TUI select a project, find a Sprint, inspect its assigned stories, and then invoke the already-existing operations without introducing business policy in the presentation layer.

## Scope and constraints

### In scope

- Add and document read-only API contracts to list open Sprints for a project and stories assigned to a selected Sprint.
- Implement TUI navigation and screens for creating a Sprint with a goal, batch-assigning same-project stories, decomposing an assigned story into titled tasks with optional estimated hours, and completing an assigned story.
- Surface API outcomes, including closed-Sprint `409`, `409 story_not_in_sprint`, and repeat completion, without double counting.
- Add focused API and TUI unit/integration tests as applicable to the existing architecture, and run the strict project test command.

### Explicitly out of scope

- Sprint closure, closed-Sprint history changes, metrics, and other unrelated workflows.
- Duplicating API membership, project, Sprint-state, or completion business rules in TUI code.
- Changing the existing US-10 task-creation permission/contract. Task creation remains permitted according to that contract; the issue only forbids assignment/completion in a closed Sprint.
- Introducing OpenSpec change artifacts. This is an authorized direct implementation route, not SDD.
- Unrequested API or TUI features beyond the discovery contracts required for navigation and the stated TUI-04 operations.

## Implementation route and authorization

- **Route:** delegated direct implementation, with a single writer. The work spans several nontrivial API and TUI files and has an API-first dependency, which is the trigger for delegation. Keep implementation on this feature branch and do not create parallel writers.
- **Order:** API discovery contracts and their tests/docs first; then Sprint TUI workflows and focused TUI tests; then full verification and candidate review.
- **Authorization:** implementation of TUI-04 is authorized on `feat/tui-04-sprint-operations`, based on `origin/main`. A new branch, pulling, and one PR to `main` are authorized. Do not push or create the PR until checks pass and the candidate is ready. Use conventional commits and keep tests/docs with the behavior in reviewable work-unit commits.
- **Issue boundary:** GitHub issue #86 (TUI-04) defines the operation scope described above. Do not infer additional issue requirements.

## Strict TDD contract

- **Mode:** strict TDD, verified from `openspec/config.yaml` (`sdd.strict_tdd: true`).
- **Runner:** `go test ./...`, configured in `openspec/config.yaml` as `sdd.test_command` and `testing.projects[0].test_command`.
- **Sequence:** for each behavior, add a focused test and capture observable RED evidence before the implementation; implement the smallest change to reach GREEN; then REFACTOR while keeping the tests green. Never claim or invent RED/GREEN evidence that was not observed.
- **Verification:** run focused tests for touched API and TUI packages during work, then `go test ./...` at closure. Run Docker-backed integration tests only if Docker/Testcontainers prerequisites are available, and record the actual outcome (including unavailable prerequisites).

## Stable task IDs and acceptance checks

| ID | Work item | Acceptance criteria and checks |
|---|---|---|
| TUI04-01 | Specify the discovery contracts — **CLOSED** | Defined the smallest read-only API results needed: project-scoped open Sprints and selected-Sprint assigned stories. Preserved closed-history semantics. API route tests cover empty/populated results and invalid/missing scopes; API documentation describes response shapes. Captured RED before implementation. |
| TUI04-02 | Implement Sprint discovery in the API — **PARTIAL** | Composed both read contracts through existing API/query architecture with project/Sprint scoping and no write/business-policy behavior. Focused unit tests pass. Database-backed verification remains blocked because Docker/Testcontainers cannot start in this environment. |
| TUI04-03 | Build Sprint-operation navigation and creation | Add a navigable Sprint workflow in `internal/tui/*` that collects a project and Sprint goal, submits creation to the API, and presents success or API errors. Verify keyboard/update/view behavior and client request behavior with focused TUI tests. |
| TUI04-04 | Add batch story assignment and Sprint story selection | Allow selection of an open Sprint, show its project's backlog for choosing multiple stories, submit batch assignment, and show API outcomes. Do not locally reimplement project-membership validation. Focused tests cover navigation, request payload, and server error rendering. |
| TUI04-05 | Add task decomposition for assigned stories | From a selected Sprint story, submit one or more titled tasks with estimated hours optional. Preserve the existing US-10 task-creation contract and let the API determine validity. Focused tests cover optional/omitted estimated hours, request construction, and response/error rendering. |
| TUI04-06 | Add Sprint-scoped story completion and conflict rendering | Complete only through the existing Sprint-scoped API operation. Surface a closed-Sprint `409` and `409 story_not_in_sprint` as server outcomes. Repeated completion must not double count; rely on API semantics and present its outcome without a TUI-side counter or duplicate policy. Focused tests cover responses and repeated-operation behavior as observable through the API/TUI boundary. |
| TUI04-07 | Verify, document, and prepare the candidate | Update relevant API documentation for discovery contracts and keep behavior changes with tests. Run focused API/TUI tests and `go test ./...`; record Docker/Testcontainers availability and actual integration results. Review the diff for scope, generated files, and authored-line forecast, then prepare the authorized single PR only when checks and candidate are ready. |

## Initial authored-lines forecast

Rough forecast: **900–1,400 additions and 60–120 deletions** across implementation, tests, and API documentation. Generated files are excluded. This is a planning estimate, not a limit; revise it against the measured diff before candidate review and explain material variance.

## Progress and verification

- **Status:** TUI04-01 closed; TUI04-02 implementation complete but partial pending database-backed verification. No TUI workflows were included in this work unit.
- **Baseline evidence:** current branch is `feat/tui-04-sprint-operations`, tracking `origin/main`; working tree was clean before this document. The strict TDD flag and runner were verified in `openspec/config.yaml`.
- **Relevant verified code context:** TUI model/navigation is in `internal/tui/model.go`; existing TUI clients/screens are under `internal/tui/`; the closed-Sprint listing handler is `internal/sprint/transport/http/list_handler.go`; API route composition is `internal/api/api.go`; TUI unit tests are under `tests/unit/tui/`.
- **Implemented routes:** `GET /projects/{project_id}/sprints/open` lists open Sprints; `GET /projects/{project_id}/sprints/{sprint_id}/stories` lists only assigned stories in the project-scoped Sprint. The existing `GET /projects/{project_id}/sprints` closed-history route remains unchanged.
- **TDD evidence:** the new API route test first failed to compile because `OpenLister`, `AssignedLister`, and the assigned-story read model did not exist (RED). After implementation, `go test ./tests/unit/cmd/api` passed (GREEN); later focused package verification also passed.
- **Focused verification:** `go test ./tests/unit/sprint/... ./tests/unit/story/... ./tests/unit/cmd/api` — PASS (all listed packages).
- **Full verification:** `go test ./...` — BLOCKED/FAIL: integration packages could not create Docker providers (`checked path: $XDG_RUNTIME_DIR, failed to create Docker provider`). Unit packages and migration-file tests passed in that invocation; PostgreSQL/Testcontainers integration tests did not run.
- **Source-code changes:** API route composition/startup, Sprint and story read ports/use cases/queries/HTTP handlers, focused API route tests, and `docs/sprint-history.md`. No TUI or task-creation/closed-Sprint policy behavior changed.
- **Measured work-unit diff:** 535 additions and 6 deletions, including task tracking; this exceeds the advisory ~400 authored-line forecast. It remains one cohesive API discovery work unit with both routes, tests, documentation, and migration-backed composition; no content was cut to meet the advisory size.
- **Commit:** `3c7192a feat(api): add Sprint discovery read routes`.

## Next step

Re-run the database-backed verification for TUI04-02 when Docker/Testcontainers are available, then implement TUI04-03 and subsequent TUI workflows in dependency order.
