#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
MIGRATIONS="$ROOT/internal/project/infrastructure/postgres/migrations"
PG_BIN=${PG_BIN:-"$HOME/.local/opt/postgresql-18/usr/lib/postgresql/18/bin"}
PGSHAREDIR=${PGSHAREDIR:-"$HOME/.local/opt/postgresql-18/usr/share/postgresql/18"}
MIGRATE_BIN=${MIGRATE_BIN:-"$HOME/.local/bin/migrate"}
export PGSHAREDIR
export LD_LIBRARY_PATH="${LD_LIBRARY_PATH:+$LD_LIBRARY_PATH:}$HOME/.local/opt/postgresql-18/usr/lib/x86_64-linux-gnu"

if [[ -n "${DATABASE_URL:-}" ]]; then
  case "$DATABASE_URL" in
    postgres://127.0.0.1:*|postgresql://127.0.0.1:*|postgres://localhost:*|postgresql://localhost:*) ;;
    *) echo "DATABASE_URL must target loopback" >&2; exit 2 ;;
  esac
fi
for tool in initdb pg_ctl psql postgres; do [[ -x "$PG_BIN/$tool" ]] || { echo "missing $PG_BIN/$tool" >&2; exit 2; }; done
[[ -x "$MIGRATE_BIN" ]] || { echo "missing migrate binary: $MIGRATE_BIN" >&2; exit 2; }

# Use a disposable cluster; this harness never connects to an external server.
tmp=$(mktemp -d /tmp/us03-migration-XXXXXX)
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
started=0
cleanup() {
  rc=0
  if (( started )); then
    "$PG_BIN/pg_ctl" -D "$tmp/data" -m immediate -w stop || rc=1
    "$PG_BIN/pg_ctl" -D "$tmp/data" status >/dev/null 2>&1 && rc=1 || :
  fi
  rm -rf "$tmp"
  [[ ! -e "$tmp" ]] || rc=1
  (( rc == 0 )) || echo "cleanup failed; inspect disposable cluster $tmp" >&2
  return "$rc"
}
trap cleanup EXIT
"$PG_BIN/initdb" -D "$tmp/data" -A trust --no-locale >/dev/null
"$PG_BIN/pg_ctl" -D "$tmp/data" -o "-h 127.0.0.1 -k $tmp -p $port" -l "$tmp/postgres.log" -w start >/dev/null
started=1
# DATABASE_URL is only a loopback guard; all work always targets this disposable cluster.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" us03_test
url="postgres://$(id -un)@127.0.0.1:$port/us03_test?sslmode=disable"
psql_cmd=("$PG_BIN/psql" "$url" -v ON_ERROR_STOP=1)

# Build the historical v4 sprint branch and inject its incompatible legacy row.
"${psql_cmd[@]}" -q -c 'CREATE TABLE projects (id UUID PRIMARY KEY, name TEXT NOT NULL, start_date DATE NOT NULL, planned_finish_date DATE NOT NULL, CHECK (planned_finish_date >= start_date))' \
  -c 'CREATE TABLE stories (id UUID PRIMARY KEY, project_id UUID NOT NULL REFERENCES projects(id) ON DELETE RESTRICT, title TEXT NOT NULL, description TEXT NOT NULL, priority TEXT NOT NULL CHECK (priority IN ('\''alta'\'', '\''media'\'', '\''baja'\'')), status TEXT NOT NULL, story_points INTEGER NULL, acceptance_criteria TEXT[] NOT NULL, CHECK (cardinality(acceptance_criteria) > 0), CHECK (array_position(acceptance_criteria, NULL) IS NULL))' \
  -c "INSERT INTO projects VALUES ('00000000-0000-0000-0000-000000000001','fixture','2025-01-01','2025-12-31')" \
  -c "INSERT INTO stories (id,project_id,title,description,priority,status,acceptance_criteria) VALUES ('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','legacy','fixture','alta','legacy',ARRAY['done'])" \
  -c 'CREATE TABLE sprints (id UUID PRIMARY KEY, project_id UUID NOT NULL REFERENCES projects(id) ON DELETE RESTRICT, sprint_goal TEXT NOT NULL)'
"${psql_cmd[@]}" -q -c 'CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)' -c 'INSERT INTO schema_migrations VALUES (5, false)'

set +e
"$MIGRATE_BIN" -path "$MIGRATIONS" -database "$url" force 4 >/dev/null 2>&1
"$MIGRATE_BIN" -path "$MIGRATIONS" -database "$url" up >/dev/null 2>&1
migration_rc=$?
set -e
[[ $migration_rc -ne 0 ]] || { echo 'expected migration to fail for legacy status' >&2; exit 1; }

# Failure must preserve original schema/data and close readiness via dirty migration state.
preservation=$("${psql_cmd[@]}" -Atqc "SELECT status='legacy' AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='stories' AND column_name='estimated_hours') AND NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='stories_status_check') AND to_regclass('public.sprints') IS NOT NULL FROM stories WHERE id='00000000-0000-0000-0000-000000000002'")
migration_state=$("${psql_cmd[@]}" -Atqc 'SELECT version || '\'' '\'' || dirty FROM schema_migrations')
if [[ "$preservation" != t || "$migration_state" != '5 true' ]]; then
  echo "RED observed: preservation=$preservation schema_migrations=$migration_state; migration exit=$migration_rc" >&2
  "${psql_cmd[@]}" -Atqc "SELECT status FROM stories WHERE id='00000000-0000-0000-0000-000000000002'; SELECT table_name FROM information_schema.tables WHERE table_schema=current_schema(); SELECT column_name FROM information_schema.columns WHERE table_schema=current_schema() AND table_name='stories'; SELECT conname FROM pg_constraint WHERE conrelid='stories'::regclass" >&2
  exit 1
fi
echo 'GREEN: legacy-status failure preserved schema and data; migration version is dirty.'
