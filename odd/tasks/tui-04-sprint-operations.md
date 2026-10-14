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
| TUI04-03 | Build Sprint-operation navigation and creation — **CLOSED** | Add a navigable Sprint workflow in `internal/tui/*` that collects a project and Sprint goal, submits creation to the API, and presents success or API errors. Verify keyboard/update/view behavior and client request behavior with focused TUI tests. |
| TUI04-04 | Add batch story assignment and Sprint story selection — **CLOSED** | Allow selection of an open Sprint, show its project's backlog for choosing multiple stories, submit batch assignment, and show API outcomes. Do not locally reimplement project-membership validation. Focused tests cover navigation, request payload, and server error rendering. |
| TUI04-05 | Add task decomposition for assigned stories — **CLOSED** | Reopened because the task workflow was inaccessible from its Sprint menu entry; closed after correcting navigation and verifying menu-to-project-to-open-Sprint-to-assigned-story-to-task-form behavior. From a selected Sprint story, submit one or more titled tasks with estimated hours optional. Preserve the existing US-10 task-creation contract and let the API determine validity. Focused tests cover navigation, optional/omitted estimated hours, request construction, and response/error rendering. |
| TUI04-06 | Add Sprint-scoped story completion and conflict rendering — **CLOSED** | Reopened because the completion workflow was inaccessible from its Sprint menu entry; closed after correcting navigation and verifying menu-to-project-to-open-Sprint-to-assigned-story-to-completion-request behavior. Complete only through the existing Sprint-scoped API operation. Surface a closed-Sprint `409` and `409 story_not_in_sprint` as server outcomes. Repeated completion must not double count; rely on API semantics and present its outcome without a TUI-side counter or duplicate policy. Focused tests cover navigation, responses, and repeated-operation behavior as observable through the API/TUI boundary. |
| TUI04-07 | Verify, document, and prepare the candidate — **PARTIAL** | API documentation and behavior changes are recorded with tests. Focused API/TUI tests pass. Full `go test ./...` was attempted; Docker/Testcontainers could not initialize providers, so PostgreSQL-backed verification and candidate readiness remain blocked. No push or PR was created. |

## Initial authored-lines forecast

Rough forecast: **900–1,400 additions and 60–120 deletions** across implementation, tests, and API documentation. Generated files are excluded. This is a planning estimate, not a limit; revise it against the measured diff before candidate review and explain material variance.

## Progress and verification

- **Status:** TUI04-01, TUI04-03, TUI04-04, TUI04-05, and TUI04-06 closed; TUI04-02 and TUI04-07 partial pending database-backed verification. TUI04-05/TUI04-06 were reopened for inaccessible menu navigation, then closed after the regression fix and focused tests. TUI04-06 invokes Sprint-scoped completion and surfaces API conflicts without local counting.
- **Baseline evidence:** current branch is `feat/tui-04-sprint-operations`, tracking `origin/main`; working tree was clean before this document. The strict TDD flag and runner were verified in `openspec/config.yaml`.
- **Relevant verified code context:** TUI model/navigation is in `internal/tui/model.go`; existing TUI clients/screens are under `internal/tui/`; the closed-Sprint listing handler is `internal/sprint/transport/http/list_handler.go`; API route composition is `internal/api/api.go`; TUI unit tests are under `tests/unit/tui/`.
- **Implemented routes:** `GET /projects/{project_id}/sprints/open` lists open Sprints; `GET /projects/{project_id}/sprints/{sprint_id}/stories` lists only assigned stories in the project-scoped Sprint. The existing `GET /projects/{project_id}/sprints` closed-history route remains unchanged.
- **TDD evidence:** the new API route test first failed to compile because `OpenLister`, `AssignedLister`, and the assigned-story read model did not exist (RED). After implementation, `go test ./tests/unit/cmd/api` passed (GREEN); later focused package verification also passed.
- **Focused verification:** `go test ./tests/unit/sprint/... ./tests/unit/story/... ./tests/unit/cmd/api` — PASS (all listed packages).
- **Full verification:** `go test ./...` — FAIL: PostgreSQL integration packages (project, projectmember, sprint, story, task) could not create Docker providers (`checked path: $XDG_RUNTIME_DIR, failed to create Docker provider`). Migration-file tests and all unit packages passed in that invocation; PostgreSQL/Testcontainers integration tests did not run.
- **Source-code changes:** API route composition/startup, Sprint and story read ports/use cases/queries/HTTP handlers, focused API route tests, and `docs/sprint-history.md`. No TUI or task-creation/closed-Sprint policy behavior changed.
- **Measured work-unit diff:** 535 additions and 6 deletions, including task tracking; this exceeds the advisory ~400 authored-line forecast. It remains one cohesive API discovery work unit with both routes, tests, documentation, and migration-backed composition; no content was cut to meet the advisory size.
- **Commit:** `3c7192a feat(api): add Sprint discovery read routes`.
- **TUI04-03 TDD:** focused test first failed to compile because `SprintResultMsg` and the Sprint workflow did not exist (RED); after implementing Sprint navigation, request client, form, and API response handling, `go test ./tests/unit/tui/...` passed (GREEN).
- **TUI04-03 verification:** `go test ./tests/unit/tui/...` — PASS.
- **TUI04-03 files:** `internal/tui/model.go`, `internal/tui/sprints.go`, `tests/unit/tui/sprint_test.go`, this tracker.
- **TUI04-03 commit:** `cd51909 feat(tui): add Sprint creation workflow`; Sprint create request uses POST `/projects/{project_id}/sprints` with `sprint_goal` and preserves API error details.
- **TUI04-04 TDD:** the focused assignment test first failed because selecting the Sprint menu's assignment entry routed to the unrelated projects screen (RED). After implementing open-Sprint discovery, backlog multiselect, ordered selected-ID payload construction, and API response handling, `go test ./tests/unit/tui/...` passed (GREEN).
- **TUI04-04 verification:** `go test ./tests/unit/tui/...` — PASS.
- **TUI04-04 files:** `internal/tui/model.go`, `internal/tui/sprints.go`, `tests/unit/tui/sprint_test.go`, this tracker.
- **TUI04-04 commit:** `e63e082 feat(tui): add batch Sprint story assignment`; POST `/projects/{project_id}/sprints/{sprint_id}/stories` receives only the explicitly selected story IDs.
- **TUI04-05 TDD:** the focused client contract test first failed to compile because `NewTaskClient` and `TaskInput` did not exist (RED). After implementing task client and assigned-story task navigation/form, `go test ./tests/unit/tui/...` passed (GREEN).
- **TUI04-05 verification:** `go test ./tests/unit/tui/...` — PASS.
- **TUI04-05 files:** `internal/tui/model.go`, `internal/tui/sprints.go`, `tests/unit/tui/sprint_test.go`, this tracker.
- **TUI04-05 contract:** POST `/projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks` sends `tasks` entries; `estimated_hours` is omitted for empty hours and included for supplied values. API conflict details render without local policy.
- **TUI04-05 commit:** `3a330d9 feat(tui): add Sprint story task creation`.
- **TUI04-06 TDD:** focused client test first failed to compile because `SprintClient.Complete` was absent (RED). After adding the Sprint-scoped request and completion result rendering, `go test ./tests/unit/tui/...` passed (GREEN).
- **TUI04-06 verification:** `go test ./tests/unit/tui/...` — PASS.
- **TUI04-06 files:** `internal/tui/model.go`, `internal/tui/sprints.go`, `tests/unit/tui/sprint_test.go`, this tracker.
- **TUI04-06 behavior:** completion POST uses `/projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion`; `sprint_closed`, `story_not_in_sprint`, and `story_already_completed` conflicts are shown as API responses. No local completion counter or policy was added.
- **TUI04-06 commit:** `c1a0f8c feat(tui): add Sprint story completion workflow`.
- **Final TUI verification:** `go test ./tests/unit/tui/...` — PASS. Repeated-completion and conflict rendering are unit/API-boundary tested; live database behavior remains unverified because Docker provider initialization fails.
- **Navigation regression TDD:** Added a table-driven end-to-end TUI update test for task creation and completion menu paths. RED observed before implementation: both cases failed because entering the project ID returned no open-Sprint request. The test follows the open-Sprint response and selected-Sprint request, verifies assigned stories load, then proves task selection reaches the task form and completion sends the Sprint-scoped completion POST. The initial test harness was corrected to execute the open-Sprint selection command before asserting the assigned-story request.
- **Navigation regression fix:** `updateSprintForm` now routes `assign`, `tasks`, and `complete` through open-Sprint discovery; existing `updateOpenSprints` dispatch then loads assigned Sprint stories for tasks/completion. No API business logic changed.
- **Navigation regression verification:** `go test ./tests/unit/tui -run TestSprintTaskAndCompletionMenusNavigateThroughAssignedStories -count=1` — PASS; `go test ./tests/unit/tui/...` — PASS; `git diff --check` — PASS. Regression tests and fix are included in work-unit commit `fix(tui): restore Sprint task and completion navigation`; exact Git commit hash is assigned when committed.
- **Navigation regression files:** `internal/tui/sprints.go`, `tests/unit/tui/sprint_test.go`, this tracker.
- **Navigation regression work-unit commit:** `fix(tui): restore Sprint task and completion navigation` (one commit contains the fix, regression tests, and tracker update).
- **Candidate review:** before this navigation fix, local branch `feat/tui-04-sprint-operations` was seven commits ahead of `origin/main`; this fix adds one local work-unit commit. No remote operation, push, or PR creation was performed. Candidate preparation remains partial pending PostgreSQL-backed verification.

## Next step

When Docker/Testcontainers are available, rerun `go test ./...` to complete TUI04-02 and TUI04-07 verification, then review candidate readiness. TUI04-07 remains partial until that Docker-backed integration verification passes. No remote action is authorized by this execution.
