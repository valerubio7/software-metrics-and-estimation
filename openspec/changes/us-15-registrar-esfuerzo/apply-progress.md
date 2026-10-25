# Progreso de apply: us-15-registrar-esfuerzo

- Modo: TDD estricto (`go test ./...`), almacén `openspec` con espejo en Engram (`sdd/us-15-registrar-esfuerzo/apply-progress`).
- Estrategia de entrega: `single-pr` con `size:exception` aceptado por el usuario.
- Entorno: Docker Desktop disponible (engine 29.8.1); las pruebas de integración usan Testcontainers `postgres:16-alpine` y se ejecutaron sin `-short`.

## Resumen de avance

| UT | Descripción | Estado | Commit |
|----|-------------|--------|--------|
| 1 | Migración `000012_create_effort_records` | Completa (tareas 1.1–1.7) | `feat(db): add effort records migration` (el hash se registra en el commit de la UT 2, porque un commit no puede contener su propio hash) |
| 2 | Dominio `EffortRecord` | Pendiente | |
| 3 | Caso de uso `RegisterEffortUseCase` | Pendiente | |
| 4 | Repositorio PostgreSQL y `MapConstraintError` | Pendiente | |
| 5 | Transporte HTTP | Pendiente | |
| 6 | Cableado, integración HTTP y documentación | Pendiente | |

## UT 1: Migración `000012_create_effort_records`

### Evidencia de ciclo TDD

| Tarea | Archivo de prueba | Capa | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------|-----|-------|-------------|----------|
| 1.1 | `tests/integration/migrations/migration_files_test.go` | Archivos | N/A: el archivo se modificó en esta misma tarea; la versión previa exigía 11 pares | Falla observada | Pasa | ➖ Un solo escenario (conteo y versión canónica) | Sin cambios |
| 1.2 / 1.3 | `tests/integration/effort/postgres/migration_integration_test.go` | Integración (PostgreSQL real) | N/A (archivo nuevo) | 5 de 5 fallan | 5 de 5 pasan | Ver 1.4 | Ver 1.5 |
| 1.4 | mismo archivo | Integración | N/A | Escrito en el mismo lote RED (ver nota) | Pasan | Mutaciones del SQL (abajo) | Ver 1.5 |
| 1.5 | ambos archivos | Integración | Verde previo (1.3) | ➖ No aplica | ✅ 5 de 5 + migraciones | ➖ No aplica | ✅ Helper unificado, sin cambios de comportamiento |

### RED observado

- Comando: `go test ./tests/integration/migrations/...`
  Resultado: `FAIL`. Motivo: `canonical sequence must contain exactly twelve migration pairs` y `missing canonical migration version 000012`.
- Comando: `go test ./tests/integration/effort/postgres/... -run TestEffortMigration -count=1 -v` (con Docker).
  Resultado: `FAIL` en las cinco pruebas (`TestEffortMigrationCreatesSchema`, `TestEffortMigrationUpOverVersion11KeepsExistingData`, `TestEffortMigrationDownRemovesSchemaAndKeepsData`, `TestEffortMigrationDownIsIdempotent`, `TestEffortMigrationCanBeReapplied`). Motivo correcto: `migration 000012 up: found 0 files ([]), want exactly 1`, es decir, la migración todavía no existe.
- Nota de proceso: las tres pruebas de TRIANGULATE de la tarea 1.4 (`Down...`, `DownIsIdempotent`, `CanBeReapplied`) se escribieron en el mismo lote que las de la tarea 1.2 y fallaron todas por la misma causa antes del GREEN. No hubo un RED separado por prueba.

### GREEN observado

- Se crearon `000012_create_effort_records.up.sql` y `000012_create_effort_records.down.sql` exactamente como define `design.md`.
- `go test ./tests/integration/migrations/... -count=1`: `ok`.
- Primer intento de las pruebas de integración: 3 de 5 fallaron por un defecto de la propia prueba (no de la migración): `ERROR: operator is not unique: text || "char" (SQLSTATE 42725)` al concatenar `confmatchtype` y `confdeltype` (tipo `"char"`) con texto en la consulta de `pg_constraint`. Corrección: `confmatchtype::text || '/' || confdeltype::text`. Las dos pruebas de `down` ya pasaban porque no usan esa consulta.
- Tras la corrección, `go test ./tests/integration/effort/postgres/... -run TestEffortMigration -count=1 -v`: `PASS` en las cinco pruebas (5 de 5).

### TRIANGULATE (las pruebas pasaron al primer intento, por eso se forzó el fallo)

Como el esquema real pasó a la primera una vez corregida la consulta, se comprobó que las aserciones detectan regresiones mutando temporalmente el SQL (cada mutación se revirtió y se verificó con `diff` contra la copia original):

| Mutación temporal | Prueba que debía fallar | Resultado observado |
|-------------------|-------------------------|---------------------|
| `up`: FK de integrante con `ON DELETE CASCADE` | `TestEffortMigrationCreatesSchema` | Falla: la FK informa `ON DELETE CASCADE/s/c` en lugar de `RESTRICT/s/r` |
| `down`: `DROP TABLE` sin `IF EXISTS` | `TestEffortMigrationDownIsIdempotent` | Falla: `table "effort_records" does not exist (SQLSTATE 42P01)` en el segundo `down` |
| `down`: eliminar las `UNIQUE` antes de la tabla | `TestEffortMigrationDownRemovesSchemaAndKeepsData` | Falla: `cannot drop constraint tasks_id_story_id_project_id_key ... other objects depend on it (SQLSTATE 2BP01)` |

Escenarios cubiertos: `up` sobre v11 con datos previos (proyecto, historia, integrante y tarea intactos), `down` con datos en `effort_records`, `tasks` y `project_members` (esquema eliminado y datos conservados), `down` repetido sin error, y reaplicación (`up` tras `down`) con altas directas por SQL, una con `task_id` nulo y otra con `task_id`.

### REFACTOR

- Se simplificó `uniqueConstraintShapes` para reutilizar `constraintShapes` (se evitó una segunda consulta casi idéntica a `pg_constraint`).
- Se eliminaron del archivo de pruebas el helper y los imports que ninguna prueba usa todavía (`assertDatabaseError`, `errors`, `pgconn`); la UT 4 los incorporará cuando los necesite.
- Se revisaron nombres de restricciones y formato del SQL contra `000001`–`000011`: el `up` no usa `IF NOT EXISTS`; `IF EXISTS` solo aparece en el `down`, por requisito del cambio.
- `gofmt -l` sin salida para los archivos nuevos, `go vet ./tests/integration/effort/...` sin hallazgos. Las pruebas de 1.3 se re-ejecutaron: verde.

### Work Unit Evidence (UT 1)

| Evidencia | Valor |
|-----------|-------|
| Comando de prueba focalizado y resultado | `go test ./tests/integration/migrations/... -count=1`: `ok`. `go test ./tests/integration/effort/postgres/... -run TestEffortMigration -count=1 -v`: 5 de 5 `PASS` |
| Arnés de ejecución real | Testcontainers `postgres:16-alpine` con Docker Desktop (engine 29.8.1): `up`, `down`, `down` repetido y reaplicación contra PostgreSQL real |
| Límite de reversión | Revertir `000012_create_effort_records.{up,down}.sql`, la línea `"000012"` y el conteo `12` de `migration_files_test.go`, y `tests/integration/effort/postgres/migration_integration_test.go`; ningún otro código depende todavía de la migración |

### Suite completa

- `go test ./... -count=1` (paralelismo por defecto, con Docker): los paquetes unitarios, `tests/integration/migrations`, `tests/integration/effort/postgres`, `project`, `projectmember` y `sprint` pasan. Fallaron `tests/integration/story/postgres` (agotó el límite de 600 s) y `tests/integration/task/postgres` con `wait for PostgreSQL: ... unexpected EOF`. Son paquetes preexistentes que no se modificaron; el motivo es la contención de Docker Desktop al arrancar muchos contenedores en paralelo (mismo precedente que US-10). No se cuenta como cumplido.
- `go test -p 1 ./... -count=1` (un paquete a la vez): todos los paquetes con pruebas pasan, incluidos `tests/integration/effort/postgres` (32.0 s), `tests/integration/migrations` (0.9 s), `story/postgres` (408.5 s) y `task/postgres` (83.7 s); 0 fallos, ninguna prueba omitida.

### Defectos reales encontrados

- Ninguno en la migración. Único defecto: la consulta de catálogo de la propia prueba (`text || "char"`), corregida en GREEN.

### Desviaciones del diseño

Ninguna: el SQL coincide con la sección "Migración `000012_create_effort_records`" de `design.md`.

### Observaciones

- Los archivos nuevos se escribieron con finales de línea LF; los archivos existentes del repositorio aparecen como CRLF en el árbol de trabajo por `core.autocrlf=true`, por lo que `gofmt -l` los lista todos sin que eso provenga de este cambio.
