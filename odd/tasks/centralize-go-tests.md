# Centralize Go tests

## Objective
Move every Go test into `tests/unit/` or `tests/integration/`, preserving behavior, `go test ./...`, and prior TDD history.

## Tasks
- [x] Extract command bootstrap helpers into an importable internal package so external unit tests can exercise them. Evidence: `1a6c6cb`.
- [x] Relocate all unit suites into `tests/unit/` and adapt command tests. Evidence: `f15e3e9`.
- [x] Relocate PostgreSQL tests into `tests/integration/` and fix migration lookup. Evidence: `a6cec35`.
- [x] Verify all tests, including actual PostgreSQL container execution without skips; update test instructions and commit each work unit. Evidence: `422fdd8` documents commands; independent verification confirmed `go test -count=1 ./...` and both real PostgreSQL integration tests passed without skips.

## Scope
Only test organization and minimal bootstrap extraction; no new product behavior.
