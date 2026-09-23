# Software Metrics and Estimation

This service currently implements US-01: create a project through an HTTP API backed by PostgreSQL.

## Prerequisites

- Go (the version declared in `go.mod`)
- PostgreSQL, with a database available to the application
- The [`golang-migrate`](https://github.com/golang-migrate/migrate) CLI installed and available as `migrate`

The application does **not** run migrations automatically, and this repository does not install a migration tool for you. Apply the versioned SQL migration before starting the API.

## Run locally

1. Create a PostgreSQL database and set its connection URL. Adapt the credentials, host, port, and database name to your local PostgreSQL setup:

   ```sh
   export DATABASE_URL='postgres://postgres:postgres@localhost:5432/projects?sslmode=disable'
   ```

2. Apply the versioned SQL migrations with the `golang-migrate` CLI:

   ```sh
   migrate -path internal/project/infrastructure/postgres/migrations -database "$DATABASE_URL" up
   ```

   This applies `000001_create_projects.up.sql`, which creates the `projects` table. Migration execution is an explicit deployment or local-environment prerequisite, not runtime API behavior.

3. Optionally choose the HTTP listening address. It defaults to `:8080` when `HTTP_ADDR` is unset:

   ```sh
   export HTTP_ADDR=':8080'
   ```

4. Start the API:

   ```sh
   go run ./cmd/api
   ```

   `DATABASE_URL` is required. At startup, the API connects to and pings PostgreSQL before listening for requests.

## Create a project

With the API running, create a project with:

```sh
curl -i -X POST http://localhost:8080/projects \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Metrics portal",
    "start_date": "2026-03-01",
    "planned_finish_date": "2026-06-30"
  }'
```

A valid request returns `201 Created` with `id`, `name`, `start_date`, and `planned_finish_date`. Dates must use `YYYY-MM-DD`, and `planned_finish_date` must be on or after `start_date`.

## Test

All tests live under `tests/unit/` and `tests/integration/`. Run them from the repository root:

```sh
go test -count=1 ./...
```

To run only the unit tests:

```sh
go test -count=1 ./tests/unit/...
```

To run the PostgreSQL integration tests with verbose output:

```sh
go test -count=1 -v ./tests/integration/...
```

The integration tests use Testcontainers and require Docker to be available.
