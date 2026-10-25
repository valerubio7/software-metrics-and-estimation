# Progreso de apply: us-15-registrar-esfuerzo

- Modo: TDD estricto (`go test ./...`), almacén `openspec` con espejo en Engram (`sdd/us-15-registrar-esfuerzo/apply-progress`).
- Estrategia de entrega: `single-pr` con `size:exception` aceptado por el usuario.
- Entorno: Docker Desktop disponible (engine 29.8.1); las pruebas de integración usan Testcontainers `postgres:16-alpine` y se ejecutaron sin `-short`.

## Resumen de avance

| UT | Descripción | Estado | Commit |
|----|-------------|--------|--------|
| 1 | Migración `000012_create_effort_records` | Completa (tareas 1.1–1.7) | `af25b91` `feat(db): add effort records migration` |
| 2 | Dominio `EffortRecord` | Completa (tareas 2.1–2.6) | `3e06cef` `feat(effort): add effort record domain` |
| 3 | Caso de uso `RegisterEffortUseCase` | Completa (tareas 3.1–3.6) | `feat(effort): add register effort use case` (el hash se registra en el commit de la UT 4, porque un commit no puede contener su propio hash) |
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
