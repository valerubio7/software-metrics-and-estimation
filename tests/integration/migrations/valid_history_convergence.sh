#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
MIGRATIONS="$ROOT/internal/project/infrastructure/postgres/migrations"
PG_BIN=${PG_BIN:-"$HOME/.local/opt/postgresql-18/usr/lib/postgresql/18/bin"}
PGSHAREDIR=${PGSHAREDIR:-"$HOME/.local/opt/postgresql-18/usr/share/postgresql/18"}
MIGRATE_BIN=${MIGRATE_BIN:-"$HOME/.local/bin/migrate"}
export PGSHAREDIR
export LD_LIBRARY_PATH="${LD_LIBRARY_PATH:+$LD_LIBRARY_PATH:}$HOME/.local/opt/postgresql-18/usr/lib/x86_64-linux-gnu"
for tool in initdb pg_ctl createdb psql; do [[ -x "$PG_BIN/$tool" ]] || { echo "missing $PG_BIN/$tool" >&2; exit 2; }; done
[[ -x "$MIGRATE_BIN" ]] || { echo "missing migrate binary: $MIGRATE_BIN" >&2; exit 2; }
version=$(go version -m "$MIGRATE_BIN" | awk '$1 == "mod" && $2 == "github.com/golang-migrate/migrate/v4" {print $3}')
[[ "$version" == v4.19.1 ]] || { echo "expected golang-migrate v4.19.1, got ${version:-unknown}" >&2; exit 2; }

# Isolated PostgreSQL 18 cluster; never accepts or uses a persistent database URL.
tmp=$(mktemp -d /tmp/us03-migration-matrix-XXXXXX)
port=$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')
started=0
cleanup() {
  local rc=0
  if (( started )); then
    "$PG_BIN/pg_ctl" -D "$tmp/data" -m immediate -w stop || rc=1
    "$PG_BIN/pg_ctl" -D "$tmp/data" status >/dev/null 2>&1 && rc=1 || :
  fi
  rm -rf "$tmp"
  [[ ! -e "$tmp" ]] || rc=1
  (( rc == 0 )) || echo "cleanup failed for disposable cluster $tmp" >&2
  return "$rc"
}
trap cleanup EXIT
"$PG_BIN/initdb" -D "$tmp/data" -A trust --no-locale >/dev/null
"$PG_BIN/pg_ctl" -D "$tmp/data" -o "-h 127.0.0.1 -k $tmp -p $port" -l "$tmp/postgres.log" -w start >/dev/null
started=1

url_for() { printf 'postgres://%s@127.0.0.1:%s/%s?sslmode=disable' "$(id -un)" "$port" "$1"; }
psql_for() { local db=$1; shift; "$PG_BIN/psql" "$(url_for "$db")" -v ON_ERROR_STOP=1 "$@"; }
apply_sql() { psql_for "$1" -f "$2" >/dev/null; }
seed_project_story() {
  local db=$1
  psql_for "$db" -q -c "INSERT INTO projects VALUES ('00000000-0000-0000-0000-000000000001','matrix-project','2025-01-01','2025-12-31')" \
    -c "INSERT INTO stories (id,project_id,title,description,priority,status,story_points,acceptance_criteria) VALUES ('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','matrix-story','preserve me','alta','pendiente',3,ARRAY['accept'])"
}
assert_converged() {
  local db=$1 expect_sprints=$2
  local state counts structure values
  state=$(psql_for "$db" -Atqc 'SELECT version || '\'' '\'' || dirty FROM schema_migrations')
  [[ "$state" == '6 false' ]] || { echo "$db migration state: $state" >&2; return 1; }
  local member_indexes
  member_indexes=$(psql_for "$db" -Atqc "SELECT to_regclass('public.project_members') IS NOT NULL AND (SELECT count(*)=2 FROM pg_indexes WHERE schemaname='public' AND tablename='project_members' AND indexname IN ('project_members_identity_with_email','project_members_identity_without_email'))")
  [[ "$member_indexes" == t ]] || { echo "$db project_members relation or NULL-aware unique indexes missing" >&2; return 1; }
  counts=$(psql_for "$db" -Atqc "SELECT (SELECT count(*) FROM projects)||':'||(SELECT count(*) FROM stories)||':'||(SELECT count(*) FROM sprints)")
  if [[ "$expect_sprints" == yes ]]; then [[ "$counts" == '1:1:1' ]] || { echo "$db row counts: $counts" >&2; return 1; }
  else [[ "$counts" == '1:1:0' ]] || { echo "$db row counts: $counts" >&2; return 1; }; fi
  values=$(psql_for "$db" -Atqc "SELECT p.name||':'||s.title||':'||s.description||':'||s.status||':'||s.story_points||':'||array_to_string(s.acceptance_criteria,',')||':'||coalesce(s.estimated_hours::text,'')||':'||s.seq FROM projects p JOIN stories s ON s.project_id=p.id WHERE p.id='00000000-0000-0000-0000-000000000001'")
  [[ "$values" == 'matrix-project:matrix-story:preserve me:pendiente:3:accept::1' ]] || { echo "$db seeded values changed: $values" >&2; return 1; }
  structure=$(psql_for "$db" -Atqc "SELECT (SELECT count(*)=4 FROM information_schema.columns WHERE table_schema='public' AND table_name='stories' AND column_name IN ('id','project_id','estimated_hours','seq')) AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='stories'::regclass AND conname='stories_status_check') AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='stories'::regclass AND conname='stories_estimated_hours_positive') AND (to_regclass('public.sprints') IS NOT NULL) AND EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='sprints'::regclass AND contype='f')")
  [[ "$structure" == t ]] || { echo "$db expected columns/checks/sprints structure absent" >&2; return 1; }
  local sprint_value
  sprint_value=$(psql_for "$db" -Atqc "SELECT count(*)||':'||coalesce(max(sprint_goal),'') FROM sprints")
  if [[ "$expect_sprints" == yes ]]; then [[ "$sprint_value" == '1:preserve sprint' ]] || { echo "$db sprint seed changed: $sprint_value" >&2; return 1; }
  else [[ "$sprint_value" == '0:' ]] || { echo "$db unexpected sprint data: $sprint_value" >&2; return 1; }; fi
}
assert_v5_idempotent() {
  local db=$1 before after state
  before=$(psql_for "$db" -Atqc "SELECT (SELECT count(*) FROM projects)||':'||(SELECT count(*) FROM stories)||':'||(SELECT count(*) FROM sprints)||':'||(SELECT count(*) FROM project_members)||':'||(SELECT name FROM projects WHERE id='00000000-0000-0000-0000-000000000001')||':'||(SELECT title FROM stories WHERE id='00000000-0000-0000-0000-000000000002')||':'||(SELECT sprint_goal FROM sprints WHERE id='00000000-0000-0000-0000-000000000003')")
  apply_sql "$db" "$MIGRATIONS/000005_reconcile_story_hours_and_sprints.up.sql"
  after=$(psql_for "$db" -Atqc "SELECT (SELECT count(*) FROM projects)||':'||(SELECT count(*) FROM stories)||':'||(SELECT count(*) FROM sprints)||':'||(SELECT count(*) FROM project_members)||':'||(SELECT name FROM projects WHERE id='00000000-0000-0000-0000-000000000001')||':'||(SELECT title FROM stories WHERE id='00000000-0000-0000-0000-000000000002')||':'||(SELECT sprint_goal FROM sprints WHERE id='00000000-0000-0000-0000-000000000003')")
  [[ "$after" == "$before" ]] || { echo "$db v5 rerun changed rows/values: $before -> $after" >&2; return 1; }
  state=$(psql_for "$db" -Atqc 'SELECT version || '\'' '\'' || dirty FROM schema_migrations')
  [[ "$state" == '6 false' ]] || { echo "$db v5 rerun changed migration state: $state" >&2; return 1; }
}

# (1) Fresh install: runner resolves the complete unique source and applies through v5.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" fresh
"$MIGRATE_BIN" -path "$MIGRATIONS" -database "$(url_for fresh)" up
seed_project_story fresh
psql_for fresh -q -c "INSERT INTO sprints VALUES ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001','preserve sprint')"
assert_converged fresh yes
assert_v5_idempotent fresh

# (2) Historical estimated-hours 000003: apply 001-004 as then deployed, seed valid rows, then v5.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" hours_history
for migration in 000001_create_projects.up.sql 000002_create_stories.up.sql 000003_add_story_estimated_hours.up.sql 000004_add_story_creation_sequence.up.sql; do apply_sql hours_history "$MIGRATIONS/$migration"; done
seed_project_story hours_history
psql_for hours_history -q -c 'CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)' -c 'INSERT INTO schema_migrations VALUES (4,false)'
"$MIGRATE_BIN" -path "$MIGRATIONS" -database "$(url_for hours_history)" up
assert_converged hours_history no

# (3) Weak same-named checks must fail closed before persistent v5 DDL.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" weak_checks
for migration in 000001_create_projects.up.sql 000002_create_stories.up.sql 000003_add_story_estimated_hours.up.sql 000004_add_story_creation_sequence.up.sql; do apply_sql weak_checks "$MIGRATIONS/$migration"; done
seed_project_story weak_checks
psql_for weak_checks -q -c 'ALTER TABLE stories DROP CONSTRAINT stories_estimated_hours_positive' -c 'ALTER TABLE stories ADD CONSTRAINT stories_estimated_hours_positive CHECK (estimated_hours > 0 OR estimated_hours = 0)' -c 'ALTER TABLE stories DROP CONSTRAINT stories_status_check' -c "ALTER TABLE stories ADD CONSTRAINT stories_status_check CHECK (status IN ('pendiente', 'en_progreso', 'completada') OR true)" -c 'CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)' -c 'INSERT INTO schema_migrations VALUES (4,false)'
# Capture the server's actual canonical rendering for the two valid 000003 checks.
psql_for hours_history -Atqc "SELECT conname || ':' || pg_get_constraintdef(oid) FROM pg_constraint WHERE conname IN ('stories_estimated_hours_positive','stories_status_check') ORDER BY conname"
if "$MIGRATE_BIN" -path "$MIGRATIONS" -database "$(url_for weak_checks)" up; then
  echo 'weak same-named checks unexpectedly passed v5 preflight' >&2
  exit 1
fi
weak_state=$(psql_for weak_checks -Atqc 'SELECT version || '\'' '\'' || dirty FROM schema_migrations')
[[ "$weak_state" == '5 true' ]] || { echo "weak-check dirty state: $weak_state" >&2; exit 1; }
weak_sprints=$(psql_for weak_checks -Atqc "SELECT to_regclass('public.sprints') IS NULL")
[[ "$weak_sprints" == t ]] || { echo 'weak-check preflight created sprints' >&2; exit 1; }
weak_values=$(psql_for weak_checks -Atqc "SELECT title||':'||status||':'||coalesce(estimated_hours::text,'') FROM stories WHERE id='00000000-0000-0000-0000-000000000002'")
[[ "$weak_values" == 'matrix-story:pendiente:' ]] || { echo "weak-check story values changed: $weak_values" >&2; exit 1; }

# (4) A weak status check alone must fail closed; the valid hours check remains intact.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" weak_status
for migration in 000001_create_projects.up.sql 000002_create_stories.up.sql 000003_add_story_estimated_hours.up.sql 000004_add_story_creation_sequence.up.sql; do apply_sql weak_status "$MIGRATIONS/$migration"; done
seed_project_story weak_status
psql_for weak_status -q -c "UPDATE stories SET estimated_hours = 5.25 WHERE id='00000000-0000-0000-0000-000000000002'" -c 'ALTER TABLE stories DROP CONSTRAINT stories_status_check' -c "ALTER TABLE stories ADD CONSTRAINT stories_status_check CHECK (status IN ('pendiente', 'en_progreso', 'completada') OR true)" -c 'CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)' -c 'INSERT INTO schema_migrations VALUES (4,false)'
if "$MIGRATE_BIN" -path "$MIGRATIONS" -database "$(url_for weak_status)" up; then
  echo 'weak status check unexpectedly passed v5 preflight' >&2
  exit 1
fi
weak_status_state=$(psql_for weak_status -Atqc 'SELECT version || '\'' '\'' || dirty FROM schema_migrations')
[[ "$weak_status_state" == '5 true' ]] || { echo "weak-status dirty state: $weak_status_state" >&2; exit 1; }
weak_status_sprints=$(psql_for weak_status -Atqc "SELECT to_regclass('public.sprints') IS NULL")
[[ "$weak_status_sprints" == t ]] || { echo 'weak-status preflight created sprints' >&2; exit 1; }
weak_status_values=$(psql_for weak_status -Atqc "SELECT title||':'||status||':'||coalesce(estimated_hours::text,'') FROM stories WHERE id='00000000-0000-0000-0000-000000000002'")
[[ "$weak_status_values" == 'matrix-story:pendiente:5.25' ]] || { echo "weak-status story values changed: $weak_status_values" >&2; exit 1; }

# (5) Historical sprint 000003 fixture plus canonical 001/002/004, then v5.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" sprint_history
for migration in 000001_create_projects.up.sql 000002_create_stories.up.sql; do apply_sql sprint_history "$MIGRATIONS/$migration"; done
apply_sql sprint_history "$MIGRATIONS/fixtures/000003_create_sprints.up.sql"
apply_sql sprint_history "$MIGRATIONS/000004_add_story_creation_sequence.up.sql"
seed_project_story sprint_history
psql_for sprint_history -q -c "INSERT INTO sprints VALUES ('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001','preserve sprint')" \
  -c 'CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)' -c 'INSERT INTO schema_migrations VALUES (4,false)'
"$MIGRATE_BIN" -path "$MIGRATIONS" -database "$(url_for sprint_history)" up
assert_converged sprint_history yes
assert_v5_idempotent sprint_history

# (3) A partial pre-existing sprints relation must fail before altering it or stories.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" partial_sprints
for migration in 000001_create_projects.up.sql 000002_create_stories.up.sql 000003_add_story_estimated_hours.up.sql 000004_add_story_creation_sequence.up.sql; do apply_sql partial_sprints "$MIGRATIONS/$migration"; done
seed_project_story partial_sprints
psql_for partial_sprints -q -c 'CREATE TABLE sprints (id UUID PRIMARY KEY)' -c "INSERT INTO sprints VALUES ('00000000-0000-0000-0000-000000000003')" -c 'CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY, dirty BOOLEAN NOT NULL)' -c 'INSERT INTO schema_migrations VALUES (4,false)'
if "$MIGRATE_BIN" -path "$MIGRATIONS" -database "$(url_for partial_sprints)" up; then
  echo 'partial pre-existing sprints unexpectedly passed v5 preflight' >&2
  exit 1
fi
partial_state=$(psql_for partial_sprints -Atqc 'SELECT version || '\'' '\'' || dirty FROM schema_migrations')
[[ "$partial_state" == '5 true' ]] || { echo "partial-sprints migration state: $partial_state" >&2; exit 1; }
partial_schema=$(psql_for partial_sprints -Atqc "SELECT (SELECT count(*)=1 FROM information_schema.columns WHERE table_schema='public' AND table_name='sprints') AND (SELECT count(*)=1 FROM information_schema.columns WHERE table_schema='public' AND table_name='stories' AND column_name='estimated_hours')")
[[ "$partial_schema" == t ]] || { echo 'partial-sprints preflight modified a table or added estimated_hours' >&2; exit 1; }
partial_data=$(psql_for partial_sprints -Atqc "SELECT (SELECT count(*)=1 AND min(id::text)='00000000-0000-0000-0000-000000000003' FROM sprints) AND (SELECT title||':'||status FROM stories WHERE id='00000000-0000-0000-0000-000000000002')='matrix-story:pendiente'")
[[ "$partial_data" == t ]] || { echo 'partial-sprints preflight modified existing data' >&2; exit 1; }

# Down one migration may remove schema additions, but must not delete existing tables or rows.
"$MIGRATE_BIN" -path "$MIGRATIONS" -database "$(url_for sprint_history)" down 1
state=$(psql_for sprint_history -Atqc 'SELECT version || '\'' '\'' || dirty FROM schema_migrations')
[[ "$state" == '5 false' ]] || { echo "down-one migration state: $state" >&2; exit 1; }
member_removed=$(psql_for sprint_history -Atqc "SELECT to_regclass('public.project_members') IS NULL")
[[ "$member_removed" == t ]] || { echo 'down 1 retained v6 project_members relation' >&2; exit 1; }
retained=$(psql_for sprint_history -Atqc "SELECT (to_regclass('public.projects') IS NOT NULL AND to_regclass('public.stories') IS NOT NULL AND to_regclass('public.sprints') IS NOT NULL) AND (SELECT count(*)=1 FROM projects) AND (SELECT count(*)=1 FROM stories) AND (SELECT count(*)=1 FROM sprints)")
[[ "$retained" == t ]] || { echo 'down 1 removed a data-bearing relation or row' >&2; exit 1; }
# (6) Run actual repository and HTTP suites against this disposable loopback cluster only.
suite_failures=0
run_suite() {
  local env_name=$1 package=$2 database=$3
  "$PG_BIN/createdb" -h 127.0.0.1 -p "$port" "$database"
  export "$env_name=$(url_for "$database")"
  if go test "$package" -count=1; then
    :
  else
    echo "FAIL: $package against disposable local PostgreSQL" >&2
    suite_failures=1
  fi
  unset "$env_name"
}
run_suite STORY_TEST_DATABASE_URL ./tests/integration/story/postgres story_suite
run_suite SPRINT_TEST_DATABASE_URL ./tests/integration/sprint/postgres sprint_suite
run_suite PROJECTMEMBER_TEST_DATABASE_URL ./tests/integration/projectmember/postgres member_suite
(( suite_failures == 0 )) || exit 1

# All package-specific helpers isolate each test in its own schema; sharing this
# disposable database lets the full repository suite run without Docker.
"$PG_BIN/createdb" -h 127.0.0.1 -p "$port" full_suite
full_suite_url=$(url_for full_suite)
export PROJECT_TEST_DATABASE_URL="$full_suite_url"
export STORY_TEST_DATABASE_URL="$full_suite_url"
export SPRINT_TEST_DATABASE_URL="$full_suite_url"
export PROJECTMEMBER_TEST_DATABASE_URL="$full_suite_url"
if go test ./... -count=1; then
  :
else
  echo 'FAIL: go test ./... -count=1 against disposable local PostgreSQL' >&2
  exit 1
fi

echo 'PASS: fresh install and both valid v4 histories converge at 6 false with seeded data preserved; down 1 removes only v6 members and retains project/story/sprint rows; story, sprint, project-member, and full Go test suites passed against disposable loopback PostgreSQL.'
