# Progreso de apply: us-15-registrar-esfuerzo

- Modo: TDD estricto (`go test ./...`), almacén `openspec` con espejo en Engram (`sdd/us-15-registrar-esfuerzo/apply-progress`).
- Estrategia de entrega: `single-pr` con `size:exception` aceptado por el usuario.
- Entorno: Docker Desktop disponible (engine 29.8.1); las pruebas de integración usan Testcontainers `postgres:16-alpine` y se ejecutaron sin `-short`.

## Resumen de avance

| UT | Descripción | Estado | Commit |
|----|-------------|--------|--------|
| 1 | Migración `000012_create_effort_records` | Completa (tareas 1.1–1.7) | `af25b91` `feat(db): add effort records migration` |
| 2 | Dominio `EffortRecord` | Completa (tareas 2.1–2.6) | `3e06cef` `feat(effort): add effort record domain` |
| 3 | Caso de uso `RegisterEffortUseCase` | Completa (tareas 3.1–3.6) | `d2353e6` `feat(effort): add register effort use case` |
| 4 | Repositorio PostgreSQL y `MapConstraintError` | Completa (tareas 4.1–4.7) | `cf4d223` `feat(effort): persist effort records in PostgreSQL` |
| 5 | Transporte HTTP `RegisterEffortHandler` | Completa (tareas 5.1–5.6) | `a8cf4d4` `feat(effort): agregar handler HTTP de registro de esfuerzo` |
| 6 | Cableado, integración HTTP y documentación | Completa (tareas 6.1–6.11) | `8ea4329` `feat(effort): exponer la ruta de registro de esfuerzo` |
| 7 | Verificación final | 7.1 a 7.4 cumplidas; 7.5 (archivado) fuera de `sdd-apply` | `fix(effort): rechazar el carácter NUL en la descripción` (tarea 7.4) |

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

## UT 2: Dominio `EffortRecord`

### Evidencia de ciclo TDD

| Tarea | Archivo de prueba | Capa | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------|-----|-------|-------------|----------|
| 2.1 / 2.2 | `tests/unit/effort/domain/effort_record_test.go` | Unitaria (dominio puro) | N/A (archivos nuevos) | Falla observada (paquete inexistente) | 8 de 8 subpruebas pasan | Ver 2.3 | Ver 2.4 |
| 2.3 | mismo archivo | Unitaria | Verde previo (8/8) | Pasaron al primer intento (ver nota) | 16 de 16 subpruebas pasan | 8 subpruebas nuevas + mutaciones | Ver 2.4 |
| 2.4 | `internal/effort/domain/effort_record.go` | Unitaria | Verde previo (16/16) | ➖ No aplica | 16 de 16 pasan | ➖ No aplica | Validaciones extraídas a funciones auxiliares, sin cambio de comportamiento |

### RED observado

- Comando: `go test ./tests/unit/effort/domain/...` (solo con las pruebas de la tarea 2.1).
  Resultado: `FAIL ... [setup failed]`. Motivo correcto: `no required module provides package .../internal/effort/domain`, es decir, el paquete de dominio todavía no existe.

### GREEN observado

- Se creó `internal/effort/domain/effort_record.go` con `EffortRecord`, `NewEffortRecordParams`, `ValidationError`, las constantes `MaxHours`, `maxHoursDecimals`, `MaxDescriptionLength` y los mensajes, `decimalPlaces` (copia local de la técnica de `internal/task/domain/task.go`, decisión D14) y `NewEffortRecord(params, today)` según la sección "Capa `domain`" de `design.md`.
- `go test ./tests/unit/effort/domain/... -count=1 -v`: `PASS` en las 8 subpruebas de `TestNewEffortRecord` (registro válido sobre historia y sobre tarea, descripción requerida, descripción recortada, horas requeridas, horas inválidas con `0`, `-1`, `-0.01`, `24.01`, `100`, `1.005`, `0.001`, `NaN` y `+Inf`, fecha futura rechazada, fecha igual o anterior aceptada).

### TRIANGULATE (las pruebas pasaron al primer intento, por eso se forzó el fallo)

Se agregaron 8 subpruebas: descripción de exactamente 500 runas multibyte (aceptada), 501 runas (rechazada contando runas y no bytes, incluyendo relleno de espacios que se recorta), relleno que no cuenta para el límite, horas válidas `0.01`, `0.5`, `7.25`, `8` y `24`, horas `2.555` rechazadas, tres campos inválidos a la vez (tres claves en `Fields` y mensaje de `Error()`), `WorkDate` cero sin error de `work_date`, y `TaskID` sin aliasing. Como la implementación ya generalizaba, pasaron todas al primer intento; para no inventar evidencia se mutó temporalmente el código (cada mutación se revirtió y se verificó con `diff` contra la copia original):

| Mutación temporal | Prueba que debía fallar | Resultado observado |
|-------------------|-------------------------|---------------------|
| Contar bytes (`len`) en lugar de runas | `description of exactly 500 multibyte characters is accepted` | Falla |
| Copiar el puntero de `TaskID` sin duplicarlo | `task ID is copied without aliasing the caller's pointer` | Falla: `TaskID aliases the caller's pointer` |
| Eliminar la comprobación de decimales de `hours` | `invalid hours` y `hours with more than two decimals are rejected...` | Fallan ambas |

### REFACTOR

- Las reglas de `description`, `hours` y `work_date` se extrajeron a `validateDescription`, `validateHours` y `validateWorkDate`, con la misma estructura que `internal/task/domain/task.go`; mensajes y semántica sin cambios.
- `gofmt -l internal/effort tests/unit/effort`: sin salida. `go vet ./internal/effort/... ./tests/unit/effort/...`: sin hallazgos. `go test ./tests/unit/effort/domain/... -count=1`: `ok`, 16 de 16 subpruebas.

### Work Unit Evidence (UT 2)

| Evidencia | Valor |
|-----------|-------|
| Comando de prueba focalizado y resultado | `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok` (`tests/unit/effort/domain`), 16 subpruebas de `TestNewEffortRecord` en verde |
| Arnés de ejecución real | N/A: dominio puro sin dependencias externas ni frontera de ejecución |
| Límite de reversión | Eliminar `internal/effort/domain/` y `tests/unit/effort/domain/`; nada más los importa todavía |

### Suite completa

No se ejecutó `go test ./...` en esta UT por instrucción del orquestador (tarda ~10 min con Docker); la verificación final corresponde a la Fase 7.

### Defectos reales encontrados

- Ninguno.

### Desviaciones del diseño

Ninguna: el dominio coincide con la sección "Capa `domain`" de `design.md`. Se agregó, solo como refactor interno, la extracción en `validateDescription`/`validateHours`/`validateWorkDate`.

## UT 3: Caso de uso `RegisterEffortUseCase`

### Evidencia de ciclo TDD

| Tarea | Archivo de prueba | Capa | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------|-----|-------|-------------|----------|
| 3.1 / 3.2 | `tests/unit/effort/application/register_effort_test.go` | Unitaria (caso de uso con fake de repositorio, generador contador y reloj fijo) | N/A (archivos nuevos) | Falla observada (paquete inexistente) | 20 de 20 subpruebas pasan | Ver 3.3 | Ver 3.4 |
| 3.3 | mismo archivo | Unitaria | Verde previo (20/20) | Pasaron al primer intento (ver nota) | 34 subpruebas en total (4 pruebas de primer nivel) pasan | 10 bordes de reloj UTC, 4 grafías de UUID, duplicados + 7 mutaciones | Ver 3.4 |
| 3.4 | `internal/effort/application/register_effort.go` | Unitaria | Verde previo (34 subpruebas) | ➖ No aplica | 34 subpruebas pasan | ➖ No aplica | Fusión de errores de dominio extraída a `mergeDomainFields`; sentinelas documentados; sin cambio de comportamiento |

### RED observado

- Comando: `go test ./tests/unit/effort/application/... -count=1` (solo con las pruebas de la tarea 3.1).
  Resultado: `FAIL ... [setup failed]`. Motivo correcto: `no required module provides package .../internal/effort/application`, es decir, el paquete de aplicación todavía no existe.

### GREEN observado

- Se creó `internal/effort/application/register_effort.go` con los cuatro sentinelas, `RegisterEffortCommand`, `EffortRepository`, `IDGenerator`, `Clock`, `RegisterEffortUseCase`, `NewRegisterEffortUseCase` y `Execute` según el flujo de 8 pasos de `design.md`: UUID de `member_id` (obligatorio, `is required` si vacío) y de `task_id` (opcional, la cadena vacía es `must be a valid UUID`) normalizados con `uuid.Parse(...).String()`; `work_date` con `time.Parse` y comparación de ida y vuelta; `today` como fecha civil UTC derivada de `now().UTC()`; fusión de los campos de `*domain.ValidationError` con los de la capa de aplicación; ID generado y repositorio invocados solo si no hay errores.
- `go test ./tests/unit/effort/application/... -count=1 -v`: 20 de 20 subpruebas de `TestRegisterEffortUseCaseExecute` en verde (éxito sobre historia y sobre tarea, normalización a minúsculas, 12 casos de validación, 5 errores del repositorio propagados sin envolver).

### TRIANGULATE (las pruebas pasaron al primer intento, por eso se forzó el fallo)

Se agregaron `TestRegisterEffortUseCaseUTCClockBoundaries` (10 bordes: reloj `2026-10-24T22:30:00-03:00` con `2026-10-25` aceptada y `2026-10-26` rechazada, `2026-10-24T23:59:59Z`, `2026-10-25T00:00:00Z`, fecha pasada `2026-01-05`, y dos casos de "fecha UTC y no local" en ambos sentidos de huso), `TestRegisterEffortUseCaseNormalizesAlternativeUUIDSpellings` (4 grafías: sin guiones, `urn:uuid:`, entre llaves y ya canónica) y `TestRegisterEffortUseCaseAllowsExactDuplicates` (dos ejecuciones idénticas → dos IDs distintos y dos llamadas al repositorio). Pasaron al primer intento; para no inventar evidencia se mutó temporalmente `register_effort.go` (cada mutación se revirtió y se verificó con `diff` contra la copia original):

| Mutación temporal | Prueba que debía fallar | Resultado observado |
|-------------------|-------------------------|---------------------|
| `instant.UTC().Date()` → `instant.Date()` (fecha local en vez de UTC) | `TestRegisterEffortUseCaseUTCClockBoundaries` | Fallan 3 bordes (`...next UTC day: that day is accepted`, `...not the local one`, `...local time is ahead of UTC`) |
| Generar el ID antes de validar | `TestRegisterEffortUseCaseExecute` (casos de validación y éxito) | Fallan todos los casos de 422 (contador del generador distinto de 0) y el éxito (ID `generated-id-2`) |
| No normalizar el UUID (devolver el valor recibido) | `uppercase member and task IDs are normalized...` | Falla |
| Devolver una fecha futura cuando `work_date` no se puede interpretar | casos de formato de fecha y de acumulación | Fallan 5 casos (se agregaría un error `must not be in the future` además del de formato) |
| ID constante en lugar de `u.generateID()` | `TestRegisterEffortUseCaseAllowsExactDuplicates` y el éxito sobre historia | Fallan |
| Envolver el error del repositorio con `fmt.Errorf("%w")` | `repository error propagates unwrapped: ...` | Fallan los 5 casos (se compara por identidad) |
| Quitar la comparación de ida y vuelta de `work_date` | (ninguna) | **Sobrevive**: no falla ninguna prueba |

Sobre la última mutación: se comprobó con un programa de sondeo que, para el diseño `"2006-01-02"`, `time.Parse` de Go ya rechaza por sí mismo `2026-3-10`, `2026-03-1`, espacios laterales, `02026-03-10`, `+2026-03-10`, `2026-03-10Z` y dígitos no ASCII; ninguna entrada probada hace que la comparación de ida y vuelta cambie el resultado. Se conserva porque el diseño y la tarea 3.2 la exigen como defensa ante cambios de comportamiento del parser, pero no hay entrada observable que la distinga; queda como observación para `sdd-verify` (código defensivo sin prueba que lo aísle).

### REFACTOR

- La fusión de errores de dominio se extrajo a `mergeDomainFields` y los sentinelas quedaron documentados; `truncateToUTCDate` y `parseWorkDate` se revisaron sin cambios. Firma de `EffortRepository` y comportamiento de `Execute` sin cambios.
- `gofmt -l internal/effort tests/unit/effort`: sin salida. `go vet ./internal/effort/... ./tests/unit/effort/...`: sin hallazgos. `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok` en `application` y `domain`.

### Work Unit Evidence (UT 3)

| Evidencia | Valor |
|-----------|-------|
| Comando de prueba focalizado y resultado | `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok` (`tests/unit/effort/application` y `tests/unit/effort/domain`); 34 subpruebas en 4 pruebas de primer nivel para la aplicación |
| Arnés de ejecución real | N/A: caso de uso probado con fake de repositorio, generador contador y reloj fijo; no hay frontera de ejecución real hasta la UT 4 |
| Límite de reversión | Eliminar `internal/effort/application/` y `tests/unit/effort/application/`; depende solo de la UT 2 y nada más lo importa todavía |

### Suite completa

No se ejecutó `go test ./...` en esta UT por instrucción del orquestador; la verificación final corresponde a la Fase 7.

### Defectos reales encontrados

- Ninguno.

### Desviaciones del diseño

Ninguna: el flujo coincide con la sección "Capa `application`" de `design.md`. Detalle de implementación: `normalizeUUID` distingue campo obligatorio (`member_id`, vacío → `is required`) de opcional (`task_id`, vacío → `must be a valid UUID`), tal como fijan los pasos 2 y 3 del diseño. La dependencia `github.com/google/uuid` ya estaba en `go.mod`.

## UT 4: Repositorio PostgreSQL y `MapConstraintError`

### Evidencia de ciclo TDD

| Tarea | Archivo de prueba | Capa | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------|-----|-------|-------------|----------|
| 4.1 | `tests/unit/effort/infrastructure/postgres/constraint_error_test.go` | Unitaria (errores `*pgconn.PgError` sintéticos) | N/A (archivo nuevo) | Falla observada (paquete inexistente) | 8 de 8 subpruebas pasan | 8 casos cubren las tres FKs, envoltura, otra FK, `23514`, error ajeno y `nil` + mutación | Ver 4.5 |
| 4.2 / 4.3 | `tests/integration/effort/postgres/repository_integration_test.go` | Integración (PostgreSQL real, Testcontainers) | Verde previo: 5 de 5 `TestEffortMigration*` | Falla observada (paquete inexistente) | 4 pruebas de primer nivel (17 subpruebas) pasan | Ver 4.4 | Ver 4.5 |
| 4.4 | mismo archivo | Integración | Verde previo (4.3) | Pasaron al primer intento salvo un defecto de la propia prueba (ver abajo) | 11 pruebas de primer nivel en total, todas pasan | 7 pruebas nuevas + 7 mutaciones forzadas | Ver 4.5 |
| 4.5 | `internal/effort/infrastructure/postgres/repository.go` | Unitaria + integración | Verde previo (4.4) | ➖ No aplica | Todo verde tras el refactor | ➖ No aplica | `requireRow` y constantes de SQL y de restricciones; sin cambio de comportamiento |

### RED observado

- Comando: `go test ./tests/unit/effort/infrastructure/postgres/... -count=1` (solo con las pruebas de la tarea 4.1).
  Resultado: `FAIL ... [setup failed]`. Motivo correcto: `no required module provides package .../internal/effort/infrastructure/postgres`, es decir, el paquete del repositorio todavía no existe.
- Comando: `go test ./tests/integration/effort/postgres/... -run TestEffortRepository -count=1` (tarea 4.2, con Docker disponible).
  Resultado: `FAIL ... [setup failed]` por el mismo motivo.

### GREEN observado

- Se creó `internal/effort/infrastructure/postgres/repository.go` con `PostgresEffortRepository`, `NewPostgresEffortRepository`, `Create` y `MapConstraintError` según la sección "Capa `infrastructure/postgres`" de `design.md`: una transacción (`defer tx.Rollback`), `SELECT EXISTS` para el proyecto, `SELECT 1 ... FOR KEY SHARE` para historia (`id, project_id`), integrante (`id, project_id`) y tarea (`id, story_id, project_id`, solo si `TaskID != nil`), `INSERT` con `work_date` como `time.Time`, `hours` como `float64` y `task_id` como `*string`, y `Commit`. El proyecto se lee con `SELECT EXISTS` (no se escanea `SELECT 1` en un `bool`, defecto histórico de `internal/task`).
- `go test ./tests/unit/effort/infrastructure/postgres/... -count=1 -v`: 8 de 8 subpruebas de `TestMapConstraintError` en verde.
- `go test -p 1 ./tests/integration/effort/postgres/... -run TestEffortRepository -count=1 -v`: `PASS` en `TestEffortRepositoryPersistsRecordOverStory`, `...OverTask`, `...CountsTaskEffortOnceForItsStory` y `...RejectsMissingOrForeignReferencesWithoutWrites` (11 subpruebas: proyecto, historia e integrante inexistentes o ajenos, tarea inexistente, de otra historia y de otro proyecto, y tres casos de precedencia), con cero filas tras cada rechazo.

### TRIANGULATE

Se agregaron siete pruebas de integración: `TestEffortRepositoryAcceptsClosedSprintAndBacklogStory` (historia y tarea de un Sprint con `is_closed = true`, e historia fuera de todo Sprint), `TestEffortRepositoryAllowsExactDuplicates` (dos altas idénticas, dos filas), `TestEffortRepositoryDoesNotCapDailyHours` (cuatro registros de 8 h el mismo día, suma `32.00`), `TestEffortRecordsTableRejectsCrossProjectReferences` (inserciones SQL directas: `23503` con el `ConstraintName` de cada FK), `TestEffortRecordsTableSkipsTaskConstraintWhenTaskIsNull` (`MATCH SIMPLE`: `task_id` nulo se acepta y una tarea inexistente se rechaza), `TestEffortRecordsTableEnforcesChecks` (`23514` con `hours` `0` y `24.01`, descripción vacía, de 501 caracteres y de 501 multibyte; aceptadas `24`, `0.01` y 500 caracteres multibyte) y `TestEffortRecordsRestrictDeletesOfReferencedParents` (`DELETE` de tarea, integrante e historia referenciados → `23503` con la FK correspondiente).

- Defecto real de la propia prueba (no del código de producción): el primer intento de `TestEffortRecordsTableEnforcesChecks` generó identificadores con letras fuera de `a-f` (`...00g`) y PostgreSQL respondió `invalid input syntax for type uuid (SQLSTATE 22P02)` en dos subpruebas. Corrección: identificadores con `fmt.Sprintf("...%012d", index+1)`. Después pasó completa (8 de 8).
- Las demás pruebas pasaron al primer intento; para no inventar evidencia se mutó temporalmente el código y el SQL (cada mutación se revirtió; `git status` sin cambios en archivos rastreados y `diff` contra la copia original):

| Mutación temporal | Prueba que debía fallar | Resultado observado |
|-------------------|-------------------------|---------------------|
| `MapConstraintError` devuelve siempre el error original | `TestMapConstraintError` | Fallan 4 casos (las tres FKs y el error envuelto) |
| La verificación de proyecto nunca rechaza | `...RejectsMissingOrForeignReferencesWithoutWrites` | Fallan `missing project` y `everything missing reports the project first` |
| El `INSERT` ignora `TaskID` (siempre `NULL`) | `TestEffortRepositoryPersistsRecordOverTask` | Falla |
| Se verifica el integrante antes que la historia | `...RejectsMissingOrForeignReferencesWithoutWrites` | Falla `story, member and task missing reports the story first` |
| SQL: FK de integrante con `ON DELETE CASCADE` | `TestEffortRecordsRestrictDeletesOfReferencedParents` | Fallan `member with effort` y `story with effort` (la cascada elimina las filas de esfuerzo) |
| SQL: `CHECK` de horas con `<= 25` | `TestEffortRecordsTableEnforcesChecks` | Falla `hours above 24` |
| SQL: `CHECK` de descripción con `octet_length` en vez de `char_length` | `TestEffortRecordsTableEnforcesChecks` | Falla `description of 500 multibyte characters` |

Mutaciones que **sobreviven**, por diseño y registradas con honestidad:

- Deshabilitar la verificación transaccional de la tarea, o hacer que la consulta del integrante ignore el proyecto, no hace fallar `...RejectsMissingOrForeignReferencesWithoutWrites`: el `INSERT` choca entonces con la FK compuesta y `MapConstraintError` devuelve el mismo sentinela (`ErrTaskNotFound`, `ErrMemberNotFound`). Son mutantes equivalentes en comportamiento observable (defensa en profundidad entre las verificaciones y las FKs); lo que distingue a las verificaciones es la precedencia y el bloqueo, y la precedencia sí está cubierta (mutación del orden).
- Quitar `FOR KEY SHARE` no hace fallar ninguna prueba: la carrera (eliminación concurrente entre la verificación y el `INSERT`) no es reproducible de forma determinista. Por eso la defensa de `23503` se prueba de forma sintética en `TestMapConstraintError`, tal como prevé el diseño, y el bloqueo queda sin prueba que lo aísle (observación para `sdd-verify`).

### REFACTOR

- Las tres consultas de verificación repetidas se extrajeron a `requireRow(ctx, tx, missing, query, args...)`; el SQL y los nombres de restricción pasaron a constantes. Se conservan el orden de verificación, los bloqueos `FOR KEY SHARE` y el mapeo sin cambios.
- `gofmt -l internal/effort tests/unit/effort tests/integration/effort`: sin salida. `go vet ./internal/effort/... ./tests/unit/effort/... ./tests/integration/effort/...`: sin hallazgos. `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok`.

### Work Unit Evidence (UT 4)

| Evidencia | Valor |
|-----------|-------|
| Comando de prueba focalizado y resultado | `go test -p 1 ./internal/effort/... ./tests/unit/effort/... ./tests/integration/effort/... -count=1`: `ok` en `tests/unit/effort/{application,domain,infrastructure/postgres}` y `tests/integration/effort/postgres` (77.6 s, 5 pruebas de migración y 11 de repositorio, sin omitidas) |
| Arnés de ejecución real | Testcontainers `postgres:16-alpine` con Docker Desktop (engine 29.8.1), migraciones `000001`–`000012` aplicadas desde archivos; transacciones, `FOR KEY SHARE`, FKs compuestas, `CHECK` y `RESTRICT` ejercitados contra PostgreSQL real |
| Límite de reversión | Eliminar `internal/effort/infrastructure/`, `tests/unit/effort/infrastructure/` y `tests/integration/effort/postgres/repository_integration_test.go`; la tabla `effort_records` (UT 1) queda sin código que la use |

### Suite completa

No se ejecutó `go test ./...` en esta UT por instrucción del orquestador (la contención de Docker en paralelo produce `EOF` y límites de tiempo); la verificación final corresponde a la Fase 7.

### Defectos reales encontrados

- Ninguno en el repositorio ni en el esquema. Único defecto: el generador de identificadores de la propia prueba de `CHECK` (letras fuera de `a-f`), corregido tras observar `SQLSTATE 22P02`.

### Desviaciones del diseño

Ninguna: el flujo transaccional y el mapeo coinciden con la sección "Capa `infrastructure/postgres`" de `design.md`. Detalle de implementación: las verificaciones de historia, integrante y tarea comparten el auxiliar `requireRow`, y la discrepancia 1 de `tasks.md` se mantiene (`ErrProjectNotFound` lo cubre solo la verificación transaccional, sin FK de proyecto que mapear). El helper `assertDatabaseError` volvió al archivo de pruebas, como anticipaba la UT 1.

## UT 5: Transporte HTTP `RegisterEffortHandler`

### Evidencia de ciclo TDD

| Tarea | Archivo de prueba | Capa | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------|-----|-------|-------------|----------|
| 5.1 / 5.2 | `tests/unit/effort/transport/http/handler_test.go` | Unitaria (`httptest` sobre `http.ServeMux` local, caso de uso real, fake de repositorio, reloj fijo y generador fijo) | N/A (archivos nuevos) | Falla observada (paquete inexistente) | 4 pruebas de primer nivel (38 subpruebas) pasan | Ver 5.3 | Ver 5.4 |
| 5.3 | mismo archivo | Unitaria | Verde previo (38 subpruebas) | Pasaron al primer intento (ver nota) | 5 pruebas de primer nivel (48 subpruebas) pasan | 10 subpruebas nuevas + 13 mutaciones | Ver 5.4 |
| 5.4 | `internal/effort/transport/http/handler.go` | Unitaria | Verde previo (48 subpruebas) | ➖ No aplica | 48 subpruebas pasan | ➖ No aplica | Constante `invalidUUID` renombrada a `invalidUUIDMessage` (convención de `application`); sin cambio de comportamiento |

### RED observado

- Comando: `go test ./tests/unit/effort/transport/http/... -count=1` (solo con las pruebas de la tarea 5.1).
  Resultado: `FAIL ... [setup failed]`. Motivo correcto: `no required module provides package .../internal/effort/transport/http`, es decir, el paquete del handler todavía no existe.

### GREEN observado

- Se creó `internal/effort/transport/http/handler.go` (paquete `transporthttp`) con `RegisterEffortHandler`, `NewRegisterEffortHandler`, los DTOs `registerEffortRequest`, `effortRecordResponse` (`task_id *string` siempre presente, sin `created_at`) y `errorResponse`, `decodeEffortObject` (primer byte `{`, sin datos posteriores, `DisallowUnknownFields`) y el orden de evaluación método → cuerpo → `project_id` → `story_id` → caso de uso. El mapeo de errores está aislado en `writeUseCaseError` (`*domain.ValidationError` con `errors.As`, sentinelas con `errors.Is`, cualquier otro error → `500 internal_error` genérico).
- `go test ./tests/unit/effort/transport/http/... -count=1 -v`: `PASS` en `TestRegisterEffortHandler` (26 casos: 3 de éxito, 14 de `400`, 2 de ruta, 2 de `422` de cuerpo y precedencia sobre `404`, 5 de `404`/`500`), `TestRegisterEffortHandlerErrorMessages` (5), `TestRegisterEffortHandlerSuccessBodyShape` (3, clave por clave: sin `created_at` ni `seq`, `task_id` `null` o UUID) y `TestRegisterEffortHandlerRejectsOtherMethods` (4 métodos, `405 method_not_allowed` con `Content-Type` JSON y cero llamadas).

### TRIANGULATE (las pruebas pasaron al primer intento, por eso se forzó el fallo)

Se agregó `TestRegisterEffortHandlerEdgeCases` (10 subpruebas): horas `0.01`, `7.25` y `24` devueltas sin pérdida de precisión; descripción de 500 caracteres multibyte aceptada e íntegra y de 501 rechazada; `task_id` y IDs de ruta en mayúsculas devueltos en minúsculas canónicas; `member_id` ausente y `task_id` `"abc"` informados en `fields`; `project_id` y `story_id` inválidos a la vez informan solo `project_id`; `400` del cuerpo con precedencia sobre el `422` de ruta; y fecha futura rechazada contra el reloj inyectado. Como la implementación ya generalizaba, pasaron al primer intento. Para no inventar evidencia se mutó temporalmente `handler.go` con un programa de mutaciones (cada una se revirtió y se verificó con `git status` y una re-ejecución en verde):

| Mutación temporal | Pruebas que debían fallar | Resultado observado |
|-------------------|---------------------------|---------------------|
| Sin `DisallowUnknownFields` | claves desconocidas (`status`, `story_id`, `id`, `created_at`) | Falla (5 pruebas) |
| Sin la comprobación de contenido posterior | `two objects`, `trailing content` | Falla (3 pruebas) |
| Sin la comprobación del primer byte `{` | `null body`, `array body` | Falla (2 pruebas) |
| `project_id` leído del valor de `story_id` (origen de la validación de ruta) | rutas inválidas y precedencia de ruta | Falla (9 pruebas) |
| Sin mapeo de `ErrMemberNotFound` | `member not found`, mensajes | Falla (4 pruebas) |
| Sin mapeo de `ErrTaskNotFound` | `task not found`, mensajes | Falla (4 pruebas) |
| `500` con `err.Error()` como mensaje | `unexpected error` (fuga de `secret db detail`) | Falla (4 pruebas) |
| `work_date` formateada como RFC 3339 | forma del `201`, precisión de horas | Falla (6 pruebas) |
| `task_id` con `omitempty` | forma del `201` (clave siempre presente) | Falla (3 pruebas) |
| Sin `Content-Type` | todas las respuestas | Falla (36 pruebas) |
| Sin comprobación de método | `RejectsOtherMethods` | Falla (5 pruebas) |
| `fields` omitidos en el `422` | casos de `422` de cuerpo | Falla (7 pruebas) |
| `TaskID` siempre `nil` en el comando | éxito sobre tarea, `task_id` en mayúsculas | Falla (5 pruebas) |

Las 13 mutaciones fueron detectadas; ninguna sobrevive.

### REFACTOR

- Constante `invalidUUID` renombrada a `invalidUUIDMessage` para alinearse con `internal/effort/application`; revisión de DTOs y mensajes contra `internal/task/transport/http/handler.go` (mismo mensaje de `400`, mismas claves de `error`). Ningún status, código ni mensaje probado cambió. `writeUseCaseError` mantiene `ServeHTTP` lineal.
- `gofmt -l internal/effort tests/unit/effort`: sin salida. `go vet ./internal/effort/... ./tests/unit/effort/...`: sin hallazgos. `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok` en `application`, `domain`, `infrastructure/postgres` y `transport/http` (5 pruebas de primer nivel y 48 subpruebas en el paquete HTTP).

### Work Unit Evidence (UT 5)

| Evidencia | Valor |
|-----------|-------|
| Comando de prueba focalizado y resultado | `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok` (4 paquetes de prueba); `tests/unit/effort/transport/http`: 5 pruebas de primer nivel y 48 subpruebas en verde |
| Arnés de ejecución real | N/A: el handler se prueba con `httptest` sobre un `http.ServeMux` local con el caso de uso real y un fake de repositorio; no hay frontera de ejecución real hasta el cableado y la integración HTTP de la UT 6 |
| Límite de reversión | Eliminar `internal/effort/transport/` y `tests/unit/effort/transport/`; depende solo de la UT 3 y nada más lo importa todavía |

### Suite completa

No se ejecutó `go test ./...` en esta UT por instrucción del orquestador; la verificación final corresponde a la Fase 7.

### Defectos reales encontrados

- Ninguno.

### Desviaciones del diseño

Ninguna: el orden de evaluación y el mapeo de errores coinciden con la sección "Capa `transport/http`" de `design.md`. Detalle de implementación: `decodeEffortObject` devuelve solo `error` (no necesita las claves crudas, a diferencia de `decodeTaskObject`). La prueba de método incorrecto invoca el handler directamente, porque el `ServeMux` de Go responde `405` por sí mismo con cuerpo de texto antes de llegar al handler. La discrepancia 1 de `tasks.md` queda cubierta: los cuatro sentinelas (incluido `ErrProjectNotFound`) producen `404` con su código y mensaje.

## UT 6: Cableado, gate de esquema, integración HTTP y documentación

### Evidencia de ciclo TDD

| Tarea | Archivo de prueba | Capa | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------|-----|-------|-------------|----------|
| 6.1 / 6.2 | `tests/unit/cmd/api/effort_route_test.go` | Unitaria (composición con `api.NewHTTPHandlerWithDependencies`, fake de repositorio) | Verde previo de `tests/unit/cmd/api` | Falla observada (no compila: faltan `EffortDependencies`, `HTTPDependencies.Effort`, `MigrationReadiness.Effort`) | `TestMigrationReadinessResolvesEffortByVersion` (6 subpruebas) y `TestEffortRouteRequiresExplicitDependency` (3) pasan | Ver 6.3 | Ver 6.7 |
| 6.3 | mismo archivo | Unitaria | Verde previo (9 subpruebas) | Pasaron al primer intento (ver nota) | 4 pruebas de primer nivel nuevas pasan | Reloj inyectado, reloj por defecto, independencia, coexistencia + 5 mutaciones | Ver 6.7 |
| 6.4 / 6.5 | `tests/integration/effort/postgres/http_integration_test.go` | Integración (PostgreSQL real, Testcontainers, `NewHTTPHandlerWithDependencies` + `PostgresEffortRepository`) | Verde previo: integración de esfuerzo de las UT 1 y 4 | Pasó al primer intento; RED obtenido forzando el fallo (ver nota) | `TestEffortHTTPEndToEnd` (14 subpruebas) pasa | Ver 6.6 | Ver 6.7 |
| 6.6 | mismo archivo | Integración | Verde previo (6.5) | Pasaron al primer intento; fallo forzado (ver nota) | `TestEffortHTTPStorageFailureRespondsInternalErrorWithoutRows` y `TestEffortHTTPRouteUnavailableBelowVersion12` pasan | 2 pruebas nuevas | Ver 6.7 |
| 6.7 | `internal/api/api.go`, `cmd/api/main.go` | Unitaria + integración | Verde previo (6.6) | ➖ No aplica | Todo verde tras el refactor | ➖ No aplica | Comentario de cabecera de migraciones de `main.go` actualizado a la versión 12; sin cambio de comportamiento |

### RED observado

- Comando: `go test ./tests/unit/cmd/api/... ./internal/api/...` (solo con las pruebas de 6.1).
  Resultado: `FAIL ... [build failed]`. Motivo correcto: `unknown field Effort in struct literal of type api.HTTPDependencies`, `undefined: api.EffortDependencies` y `readiness.Effort undefined (type api.MigrationReadiness has no field or method Effort)`, es decir, el cableado todavía no existe.

### GREEN observado

- `internal/api/api.go`: se agregaron los imports `effortapplication` y `efforthttp`, el tipo `EffortDependencies{Repository, GenerateID, Now}`, el campo `HTTPDependencies.Effort`, el campo `MigrationReadiness.Effort`, `readiness.Effort = version >= 12` (tras la salida temprana existente para `dirty` y error de lectura, por lo que ambos casos dejan `Effort` en falso) y el bloque condicional que registra `POST /projects/{project_id}/stories/{story_id}/effort-records` solo si `dependencies.Effort != nil`, con reloj por defecto `time.Now().UTC()` cuando `Now` es nil. El bloque es independiente de `Stories`, `Members` y `Tasks`.
- `cmd/api/main.go`: bloque `if readiness := api.ResolveMigrationReadiness(...); readiness.Effort { ... }` que construye `EffortDependencies` con `effortpostgres.NewPostgresEffortRepository(pool)`, `api.NewProjectID` y el reloj UTC, con mensajes de log para disponible y no disponible; no lee `dependencies.Stories`, `Members` ni `Tasks`.
- `go test ./tests/unit/cmd/api/... -count=1`: `ok`.

### TRIANGULATE (las pruebas pasaron al primer intento, por eso se forzó el fallo)

Se agregaron `TestEffortRouteUsesInjectedClock` (con el reloj `2026-03-10`, esa fecha se acepta y `2026-03-11` da `422` con `work_date`), `TestEffortRouteDefaultsToCurrentUTCClockWhenNowIsNil` (`Now` nil: `2000-01-01` se acepta y `2999-12-31` se rechaza), `TestEffortRouteIsIndependentFromOtherModules` (`HTTPDependencies{Effort}` sin `Stories`, `Members` ni `Tasks` responde `201` y `POST /projects/{id}/stories` sigue en `404`) y `TestEffortRouteCoexistsWithExistingStoryAndTaskRoutes` (con `Stories` y `Tasks` presentes, `PUT .../story-points`, `PUT .../stories/{story_id}` y `GET .../tasks` responden como antes, `POST .../effort-records` da `201` y `GET` sobre esa ruta da `405`). El test no importa `internal/tui`. Como el código ya generalizaba, pasaron al primer intento; para no inventar evidencia se mutó temporalmente `internal/api/api.go` (cada mutación se revirtió restaurando una copia y se verificó con `git diff --stat`: 23 inserciones, solo el cableado):

| Mutación temporal | Prueba que debía fallar | Resultado observado |
|-------------------|-------------------------|---------------------|
| `readiness.Effort = version >= 13` | `TestMigrationReadinessResolvesEffortByVersion` | Falla `v12` |
| `Effort: true` en el valor inicial de `MigrationReadiness` (antes de la salida temprana) | `TestMigrationReadinessResolvesEffortByVersion` | Fallan `dirty v12`, `lookup error` y `missing migration table` |
| Ignorar `Effort.Now` y usar siempre el reloj real | `TestEffortRouteUsesInjectedClock` | Falla |
| Eliminar el reloj por defecto cuando `Now` es nil | `TestEffortRouteDefaultsToCurrentUTCClockWhenNowIsNil` | Falla con `nil pointer dereference` |
| Registrar la ruta solo si además `Stories != nil` | `TestEffortRouteRequiresExplicitDependency`, `...UsesInjectedClock`, `...DefaultsToCurrentUTCClock...`, `...IsIndependentFromOtherModules` | Fallan 5 pruebas |

Las 5 mutaciones fueron detectadas; ninguna sobrevive.

### Integración HTTP real (6.4, 6.5, 6.6)

- Comando: `go test -p 1 ./tests/integration/effort/postgres/... -run TestEffortHTTP -count=1 -v` (Docker Desktop, Testcontainers `postgres:16-alpine`).
- `TestEffortHTTPEndToEnd` (14 subpruebas) pasó al primer intento: `201` sobre historia (respuesta con `task_id: null`, sin `created_at` ni `seq`, descripción recortada, fila almacenada igual a la respuesta), `201` sobre tarea (`task_id` en mayúsculas devuelto en minúsculas canónicas, `8.00` almacenado), `201` sobre historia de Sprint cerrado y de backlog, dos solicitudes idénticas con IDs distintos, y sin filas tras `404` (`member_not_found` con integrante inexistente y de otro proyecto, `task_not_found` con tarea de otra historia y de otro proyecto, `story_not_found` con historia inexistente y de otro proyecto, `project_not_found`), `422` con varios campos (`hours`, `description`, `work_date`) y precedencia del `422` sobre un integrante inexistente, y `400 invalid_request` por clave desconocida. El aislamiento entre proyectos queda cubierto por los casos de integrante, tarea e historia de otro proyecto.
- `TestEffortHTTPStorageFailureRespondsInternalErrorWithoutRows` crea un `TRIGGER BEFORE INSERT` que lanza una excepción: `POST` válido da `500 internal_error`, el cuerpo no contiene `secret internal detail` ni `forced storage failure` y `effort_records` queda con 0 filas.
- `TestEffortHTTPRouteUnavailableBelowVersion12` aplica solo `000001`–`000011`, compone sin `Effort` y observa `404` con la tabla `effort_records` inexistente.
- Resultado: `ok ... 84.477s` con las 16 pruebas y subpruebas en verde.

Como las pruebas pasaron al primer intento, el RED se obtuvo forzando el fallo (cada cambio se revirtió restaurando una copia previa del archivo):

| Cambio temporal en la prueba | Prueba que debía fallar | Resultado observado |
|------------------------------|-------------------------|---------------------|
| Componer el handler sin `Effort` (`api.HTTPDependencies{}`) | `TestEffortHTTPEndToEnd/...201...`, `...unknown_member...`, `...several_invalid_fields...` | Fallan las tres: `status = 404, want 201: 404 page not found` (y `404 page not found` en lugar de `member_not_found` y de `422`) |
| `WHEN (false)` en el trigger, que lo vuelve inofensivo | `TestEffortHTTPStorageFailureRespondsInternalErrorWithoutRows` | Falla: `status = 201, want 500` |

### Defectos reales encontrados

- Ninguno en el repositorio, el transporte ni el cableado. Único error propio de la prueba, corregido antes de ejecutar: el código del `422` es `validation_failed` (no `validation_error`).

### REFACTOR

- Comentario de cabecera del bloque de migraciones de `cmd/api/main.go` actualizado para mencionar la versión 12 (`effort_records`). Nombres y mensajes de log del cableado alineados con los de Tasks y Completion; el comportamiento de los gates existentes no cambió.
- `gofmt -l` sin salida sobre el contenido de `api.go`, `main.go` y las dos pruebas nuevas (los archivos existentes aparecen como CRLF en el árbol de trabajo por `core.autocrlf=true`, ver UT 1); `go build ./...` y `go vet ./internal/... ./cmd/... ./tests/unit/cmd/... ./tests/unit/effort/... ./tests/integration/effort/...` sin hallazgos.

### Documentación

- `README.md`: subsección `## API de esfuerzo` antes de `## Pruebas` con la ruta, el cuerpo, las respuestas (`201`, `400`, `404`, `422`, `500`) y la nota de que la ruta solo existe con el esquema limpio en versión `>= 12`; no se modificó la tabla de lecturas de la TUI.
- `openspec/config.yaml`: se reemplazó la nota obsoleta de los dos `000003` por una nota vigente (12 pares `000001`–`000012`, sin duplicados, despliegue externo) y se agregó "Implementada: US-15". No hay un analizador YAML disponible en el entorno (el intérprete no incluye `yaml`); la validez se verificó por inspección de indentación y comillas contra las líneas vecinas.

### Work Unit Evidence (UT 6)

| Evidencia | Valor |
|-----------|-------|
| Comando de prueba focalizado y resultado | `go test -p 1 ./internal/... ./tests/unit/cmd/... ./tests/unit/effort/... ./tests/integration/effort/... -count=1`: `ok` en `internal/tui`, `tests/unit/cmd/api`, los cuatro paquetes de `tests/unit/effort` y `tests/integration/effort/postgres` (149.8 s) |
| Arnés de ejecución real | Testcontainers `postgres:16-alpine` con Docker Desktop (engine 29.8.1): `api.NewHTTPHandlerWithDependencies` con `PostgresEffortRepository` real, migraciones `000001`–`000012` desde archivos, `TRIGGER` forzando un `500` |
| Límite de reversión | Revertir los bloques de `internal/api/api.go` y `cmd/api/main.go`, la sección del `README.md`, las líneas de `openspec/config.yaml` y eliminar `tests/unit/cmd/api/effort_route_test.go` y `tests/integration/effort/postgres/http_integration_test.go`; sin ellos la ruta nunca se registra |

### Suite completa

No se ejecutó `go test ./...` en esta UT por instrucción del orquestador (la contención de Docker en paralelo produce `EOF` y límites de tiempo); la verificación final corresponde a la Fase 7.

### Desviaciones del diseño

Ninguna: el cableado coincide con la sección de composición de `design.md`. Detalle: el asunto del commit es `feat(effort): exponer la ruta de registro de esfuerzo` (en español, por instrucción del orquestador) en lugar de `feat(api): gate effort registration on migration 12` de la tarea 6.11.

## Fase 7: Verificación final

Commit verificado en ambas corridas: `8ea4329` (sin cambios entre una corrida y la otra).

### 7.1 Compilación, formato y suite completa

- `go build ./...` y `go vet ./...`: sin salida de error.
- `gofmt -l internal tests cmd` lista archivos preexistentes del proyecto porque el árbol de trabajo está en CRLF por `core.autocrlf=true`. Se comprobó el contenido realmente guardado en Git: `git show HEAD:<archivo> | gofmt -l` sobre los 15 archivos `.go` que cambió la rama devuelve vacío (0 sin formato).
- No se usó `-short` y no hubo pruebas omitidas.
- La contención de Docker Desktop (precedente de US-10 y de la UT 1) obligó a ejecutar `go test -p 1 ./... -count=1`. Se hicieron dos corridas completas sobre el mismo commit:

| Paquete | Corrida 1 | Corrida 2 |
|---------|-----------|-----------|
| `tests/integration/effort/postgres` (nuevo) | FAIL: `TestEffortHTTPEndToEnd/effort_over_a_story_responds_201_and_persists_the_same_data`, `wait for PostgreSQL ... unexpected EOF` (182.7 s) | ok (144.5 s) |
| `tests/integration/story/postgres` (preexistente) | ok (436.6 s) | FAIL: `TestAPIStartupRoutesFollowMigrationState/dirty_version_four`, `wait for PostgreSQL ... unexpected EOF` (399.2 s) |
| Resto de paquetes con pruebas (27) | ok | ok |

- Motivo: ambos fallos ocurren en `waitForPostgres`, al conectar al contenedor recién arrancado, antes de que se ejecute cualquier código de la aplicación, y cada paquete que falló en una corrida pasó en la otra con el mismo commit. Es el mismo `unexpected EOF` por contención de Docker de la UT 1 y de US-10. No hay ninguna aserción de negocio fallida.
- Evidencia adicional: `go test -p 1 ./tests/integration/effort/... -count=1` pasó de forma aislada (176.0 s) y el agente de la UT 6 lo había visto pasar (149.8 s).
- **Límite:** no hay una única corrida completa en verde. La evidencia es la unión de dos corridas sobre el mismo commit, con cada paquete en `ok` al menos una vez.

### 7.3 Número de migración

`internal/project/infrastructure/postgres/migrations/` contiene 12 pares `000001`–`000012` sin duplicados, y `tests/integration/migrations` pasa. En `origin/main` (tras `git fetch`) la última migración es `000011`, por lo que `000012` sigue libre.

### 7.4 Correcciones

`sdd-verify` (tarea 7.2) dejó una sospecha sin confirmar: una descripción con el carácter NUL (`U+0000`) llegaría a PostgreSQL, que rechaza `0x00` en `text` (SQLSTATE 22021), y la API respondería `500 internal_error` en lugar de un error de validación. Se corrigió con el ciclo completo de TDD estricto y un commit adicional.

#### Evidencia de ciclo TDD (7.4)

| Archivo de prueba | Capa | Safety net | RED | GREEN | TRIANGULATE | REFACTOR |
|-------------------|------|------------|-----|-------|-------------|----------|
| `tests/integration/effort/postgres/http_integration_test.go` | Integración HTTP real (PostgreSQL) | Corrida previa de la fase 7 en verde | Falla observada: `500`, se esperaba `422` | Pasa: `422` y 0 filas | Cubierto por las pruebas de dominio y de handler | Sin cambios |
| `tests/unit/effort/domain/effort_record_test.go` | Unitaria | `ok` en `tests/unit/effort/...` | Falla observada: ocho posiciones de NUL aceptadas | Pasa | Ocho posiciones de NUL más cuatro caracteres de control que deben seguir aceptándose | Comentario único en `validateDescription` |
| `tests/unit/effort/transport/http/handler_test.go` | Unitaria (handler) | `ok` en `tests/unit/effort/...` | Falla observada: `201` con 1 llamada al repositorio, se esperaba `422` con 0 | Pasa | Tres posiciones (medio, inicio, fin) | Sin cambios |

#### RED observado

- Primero a nivel real: la subprueba `TestEffortHTTPEndToEnd/description_with_a_NUL_character_responds_422_validation_failed_without_writes` (`go test -p 1 ./tests/integration/effort/postgres/ -run 'TestEffortHTTPEndToEnd/description_with_a_NUL' -count=1`) falló con `status = 500, want 422: {"error":"internal_error",...}`. La hipótesis quedó confirmada: el defecto existía.
- Después a nivel unitario: `TestNewEffortRecord/description_containing_a_NUL_character_is_rejected_wherever_the_NUL_is` falló (`NewEffortRecord("abc\x00   ") error = <nil>`) y `TestRegisterEffortHandlerEdgeCases/description_with_a_NUL_character_is_rejected_without_calling_the_repository` falló con `status/calls = 201/1, want 422/0`.
- Un primer intento del handler envió un NUL crudo en el cuerpo y obtuvo `400` por JSON inválido; fue un error de la prueba, no evidencia de RED. Se corrigió enviando el escape JSON `\u0000`.

#### GREEN observado

- `internal/effort/domain/effort_record.go`: nueva constante `nulMessage = "must not contain NUL characters"` y un caso `strings.ContainsRune(description, 0)` en `validateDescription`, antes del límite de longitud. Se evalúa sobre la descripción recortada: `TrimSpace` no elimina NUL, por eso un NUL entre espacios también se rechaza.
- `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok` en los cuatro paquetes con pruebas.
- La subprueba real pasó con `422` y 0 filas. Una primera ejecución falló por infraestructura antes de llegar al código (no se capturó el mensaje, patrón conocido de `wait for PostgreSQL`); la repetición aislada pasó.
- `go test -p 1 ./tests/integration/effort/... -count=1 -v`: `ok` (172,6 s), 63 líneas `--- PASS` y ninguna falla, sin `unexpected EOF` en esta corrida.

#### TRIANGULATE y mutaciones

Se mutó temporalmente `validateDescription` y se revirtió cada vez (la última comprobación fue `git diff --stat` sin cambios inesperados y el paquete en verde):

| Mutación | Resultado observado |
|----------|---------------------|
| `strings.IndexRune(description, 0) > 0` (no detecta NUL en la posición 0) | Fallan dominio (`wherever_the_NUL_is`) y handler |
| Rechazar cualquier carácter de control (`unicode.IsControl`) | Falla `description_with_other_control_characters_is_still_accepted` |
| `strings.HasPrefix(description, "\x00")` (solo al inicio) | Fallan dominio y handler (`in_the_middle`, `at_the_end`) |

La mutación de eliminar la regla ya estaba demostrada por el RED. Cada mutación falló en al menos una prueba, de modo que las pruebas pueden fallar.

#### Otras rutas de texto libre

Solo `description` es texto libre. `project_id`, `story_id`, `member_id` y `task_id` se validan como UUID y se normalizan antes de llegar al repositorio, `work_date` se analiza con `time.Parse` en formato `YYYY-MM-DD` y `hours` es numérico, así que ninguno puede transportar un NUL hasta PostgreSQL.

#### REFACTOR

Se unificó la explicación del NUL en el comentario de `validateDescription` (se eliminó un comentario duplicado dentro del `case`) y se actualizó el comentario de `NewEffortRecord`. Sin cambios de comportamiento; pruebas en verde.

#### Spec

`specs/esfuerzo/spec.md`: R8 incluye el carácter NUL entre las descripciones rechazadas con un escenario nuevo (`422`, `fields.description`, nada persistido). Además se corrigió el hallazgo WARNING 2 del informe de verificación: el requisito y el escenario de R15 ya no nombran una FK de proyecto (no existe); `project_not_found` proviene de la verificación de existencia dentro de la transacción.

#### Work Unit Evidence (7.4)

| Evidencia | Valor |
|-----------|-------|
| Prueba enfocada | `go test ./internal/effort/... ./tests/unit/effort/... -count=1`: `ok` en los cuatro paquetes con pruebas |
| Arnés de ejecución real | `go test -p 1 ./tests/integration/effort/... -count=1 -v`: `ok` (172,6 s), subprueba NUL en `422` con 0 filas |
| Punto de reversión | `internal/effort/domain/effort_record.go`, las tres pruebas citadas, `specs/esfuerzo/spec.md`, `tasks.md` y este archivo; revertir el commit no afecta al resto de la rama |

#### Cierre

Con la 7.4 aplicada, solo queda la 7.5 (`sdd-archive` y consolidación del spec), que corresponde al orquestador. La advertencia de estabilidad del paquete de integración (WARNING 1) y el gate sin prueba ejecutable de `cmd/api/main.go` (WARNING 3) siguen siendo observaciones de entorno y de alcance, no defectos de negocio.
