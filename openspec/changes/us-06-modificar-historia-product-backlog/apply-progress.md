# Progreso de aplicación: US-06 — Modificar una historia del Product Backlog

Modo: TDD estricto (`go test ./...`). Estrategia de entrega: `ask-on-risk`, cadena `stacked-to-main`.
Ramas de los lotes: `feat/us06-domain-story-update` (base `main`, lote 1), `feat/us06-application-update-story` (lote 2), `feat/us06-storage-estimated-hours` (lote 3) y `feat/us06-http-update-handler` (lote 4), cada una apilada sobre la anterior. No se hizo push ni PR.

## Estado acumulado de tareas

| Slice | Tareas | Estado |
|-------|--------|--------|
| 0. Planificación | 0.1, 0.2 | Completas (commit `46392a5`) |
| 1. Dominio | 1.1 a 1.8 | Completas (commit `fab506d`) |
| 2. Aplicación | 2.1 a 2.8 | Completas (commit `d50d432`, rama `feat/us06-application-update-story`) |
| 3. Almacenamiento | 3.1 a 3.16 | Completas (commit `2bc1986`, rama `feat/us06-storage-estimated-hours`; Docker disponible, integración corrida) |
| 4. Handler HTTP | 4.1 a 4.11 | Completas (commit `dc42020`, rama `feat/us06-http-update-handler`) |
| 5. Composición y docs | 5.1 a 5.11 | Pendientes |

## Lote 1: Slice 0 y Slice 1 (dominio)

### Línea base (tarea 0.1)

`go test ./...` sobre `main` limpio, antes de tocar nada: los paquetes unitarios pasan
(`tests/unit/cmd/api`, `project/*`, `story/*`). Los paquetes de integración
(`tests/integration/project/postgres`, `tests/integration/story/postgres`) **FALLAN** (no se saltan) con
`rootless Docker is not supported on Windows, failed to create Docker provider`: no hay Docker en esta
máquina. Es una falla ambiental preexistente y **no es evidencia de persistencia**.

### Evidencia del ciclo TDD

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 1.1 | `tests/unit/story/domain/story_test.go` (`TestNewStoryUpdate`) | Unitario | 2/2 tests de `NewStory` en verde | Escrito; falla observada: `undefined: domain.NewStoryUpdate` (error de compilación) | Pase observado | Ver 1.5 | Ver 1.6 |
| 1.2 | idem (`TestNewStoryUpdateEstimatedHours`) | Unitario | N/A (nuevo) | Escrito; misma falla de compilación observada | Pase observado | Ver 1.5 | Ver 1.6 |
| 1.3 | idem (`TestNewStoryUpdatePreservesTextOrderAndIdentity`, `TestNewStoryUpdateCopiesCriteria`, `TestNewStoryInitializesWithoutEstimatedHours`, `TestAllowedStatusesReturnsTheClosedSetAsACopy`) | Unitario | N/A (nuevo) | Escrito; falla de compilación: `Story.EstimatedHours`, `domain.StatusPending`, `domain.AllowedStatuses` indefinidos | Pase observado | Ver 1.5 | Ver 1.6 |
| 1.4 | `internal/story/domain/story.go` | Unitario | N/A | Cubierto por 1.1-1.3 | `go test ./tests/unit/story/domain/...` PASS | — | — |
| 1.5 | idem | Unitario | — | — | — | Bordes `0.01`, `99999.99`, `3`, `2.5`, `0.07`, blancos Unicode en criterios, copia de la estimación y acumulación de seis campos inválidos: pasaron de inmediato (caracterización; la lógica de 1.4 ya era general, no hubo RED nuevo) | — |
| 1.6 | `internal/story/domain/story.go` | Unitario | Suite de `NewStory` verde antes y después | — | — | — | `validateStoryContent` extraída y compartida; `"pendiente"` sustituido por `StatusPending`; `go test ./tests/unit/story/domain/...` PASS |

Resumen de tests: 12 funciones de test de nivel superior en el paquete de dominio (51 subtests/casos con
`-v`), todas en verde. Capa usada: solo unitaria. Funciones puras creadas: `NewStoryUpdate`,
`validateStatus`, `validateEstimatedHours`, `decimalPlaces`, `validateStoryContent`, `AllowedStatuses`.

### Evidencia del work unit

| Evidencia | Valor observado |
|-----------|-----------------|
| Comando de test focalizado | `go test ./tests/unit/story/domain/...` -> `ok` (todas las funciones PASS) |
| Harness de runtime | N/A: funciones puras de dominio sin frontera de proceso; evidencia = suite unitaria + `go test ./...` |
| `go vet ./...` | Limpio |
| `gofmt` | Limpio en los archivos tocados. `gofmt -l .` lista casi todos los `.go` del repo porque en esta copia de trabajo de Windows (`core.autocrlf=true`) los archivos están en CRLF; es preexistente y no es un problema de formato |
| `go test ./...` completo | Paquetes unitarios `ok`; paquetes de integración FALLAN por falta de Docker, idéntico a la línea base de `main` |
| Rollback | Revertir `fab506d` (`internal/story/domain/story.go`, `tests/unit/story/domain/story_test.go`); el commit de planificación `46392a5` se revierte por separado |

### Commits

- `46392a5` `docs(sdd): add US-06 planning artifacts for story modification` (solo `openspec/changes/us-06-.../`, 5 archivos, 2000 líneas).
- `fab506d` `feat(story): add domain validation for story updates` (2 archivos, +326/-14 = 340 líneas autoradas, dentro del presupuesto de 400).

### Desviaciones del diseño

Ninguna en el comportamiento. Detalle menor: se agregaron las constantes no exportadas `maxEstimatedHours`,
`maxEstimatedHoursDecimal` y `estimatedHoursMessage` y el helper `decimalPlaces` para no repetir literales;
`NewStoryUpdate` copia el puntero de la estimación para no aliasar el valor del llamador (cubierto por
`TestNewStoryUpdateCopiesEstimatedHours`).

### Problemas encontrados

- Los tests de integración fallan (no se saltan) sin Docker; ver la línea base. El slice 3 y el arranque real del slice 5 seguirán sin poder verificarse en esta máquina.
- Los archivos `tasks.md` y `apply-progress.md` quedaron **sin commitear** en este lote: `tasks.md` ya está incluido sin marcas en el commit de planificación y la plantilla de commits del slice no pide incluir las marcas `[x]`. Se reportan al orquestador para decidir si se agrupan en un commit `docs(sdd)` posterior.

### Tareas restantes (tras el lote 1)

Slices 2 a 5 (ramas `feat/us06-application-update-story`, `feat/us06-storage-estimated-hours`,
`feat/us06-http-update-handler`, `feat/us06-wire-update-route-docs`).

## Lote 2: Slice 2 (aplicación)

Rama `feat/us06-application-update-story`, creada desde `feat/us06-domain-story-update` (apilada,
`stacked-to-main`). No se hizo push ni PR.

### Línea base (tarea 2.1)

`go test ./...` en la rama nueva antes de tocar nada: los paquetes unitarios pasan; los de integración
FALLAN con `rootless Docker is not supported on Windows` (preexistente, ambiental; no es evidencia).

### Evidencia del ciclo TDD

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 2.2 | `tests/unit/story/application/update_story_test.go` (`TestUpdateStoryWritesOnceWithCanonicalIdentifiersAndAllSixFields`, `TestUpdateStoryDoesNotWriteInvalidInput`, `TestUpdateStoryPropagatesPortErrorsWithoutClaimingSuccess`, `TestUpdateStoryErrStoryNotFoundIsDistinctFromProjectNotFound`) | Unitario | N/A (archivo nuevo); paquete `story/application` en verde | Falla observada: `undefined: application.UpdateStoryCommand`, `NewUpdateStoryUseCase`, `ErrStoryNotFound` (error de compilación) | Pase observado con un `Execute` mínimo (sin `MissingFields`) | Ver 2.4 | Ver 2.6 |
| 2.3 | `internal/story/application/update_story.go` | Unitario | N/A | Cubierto por 2.2 | `go test ./tests/unit/story/application/...` PASS | — | — |
| 2.4 | idem (`TestUpdateStoryReportsEverySuppliedMissingFieldAsRequired`, `...ReportsASingleMissingFieldOnly` (6 subtests), `...MergesInvalidIdentifiersWithContentErrors` (3 casos), `...MissingKeyMessageWinsOverZeroValueContentMessage`, `...DistinguishesAbsentEstimatedHoursFromExplicitNull` (2 subtests)) | Unitario | — | — | — | RED real observado contra el `Execute` mínimo (ignoraba `MissingFields`, devolvía `error = <nil>` y mensajes de dominio en lugar de `is required`); tras generalizar `Execute` (sembrar `fields` con `MissingFields` y no sobrescribir claves existentes) PASS | — |
| 2.5 | `tests/unit/story/application/create_story_test.go` | Unitario | Suite de creación en verde | Prueba de caracterización: pasó de inmediato (el slice 1 ya deja `EstimatedHours == nil`); no es un RED | PASS | — | — |
| 2.6 | `internal/story/application/update_story.go` | Unitario | Verde antes y después | — | — | — | Sin cambios necesarios: no hay duplicación real en `Execute` |

Resumen de tests: 9 funciones de test de nivel superior nuevas más 1 aserción aditiva; todas en verde.
Capa usada: solo unitaria. Fake de `StoryUpdater` escrito a mano (`storyUpdater`). Sin funciones puras nuevas
(el caso de uso es orquestación; las reglas viven en el dominio).

### Evidencia del work unit

| Evidencia | Valor observado |
|-----------|-----------------|
| Comando de test focalizado | `go test ./tests/unit/story/application/...` -> `ok` |
| Harness de runtime | N/A: caso de uso sobre fake; sin frontera de proceso ni de red |
| `go vet ./...` | Limpio |
| `gofmt` | Los archivos nuevos no aparecen en `gofmt -l`; `create_story.go` y `create_story_test.go` aparecen solo por CRLF de la copia de trabajo de Windows (preexistente; el diff de git es de 3 líneas) |
| `go test ./...` completo | Paquetes unitarios `ok` (`cmd/api`, `project/*`, `story/application`, `story/domain`, `story/transport/http`); integración FALLA por falta de Docker (**no verificado**) |
| Compilación de tests ajenos | `create_story_test.go`, `transport/http/handler_test.go` y `cmd/api/main_test.go` compilan y pasan sin cambios (el puerto es separado de `StoryRepository`) |
| Rollback | Revertir `d50d432` (`internal/story/application/update_story.go`, `tests/unit/story/application/update_story_test.go` y la aserción aditiva de `create_story_test.go`) |

### Commit

- `d50d432` `feat(story): add update story use case and port` (3 archivos, +327 líneas autoradas, dentro del presupuesto de 400).

### Desviaciones del diseño

Ninguna en el comportamiento. Nota de proceso: la tarea 2.3 describe el `Execute` completo, pero se
implementó primero una versión mínima (sin `MissingFields`) para obtener un RED real en la triangulación 2.4
y se generalizó después; el resultado final coincide con el diseño (Decisión 4).

### Problemas encontrados

- Los tests de integración siguen fallando sin Docker (ver la línea base del lote 1).
- `tasks.md` y `apply-progress.md` quedan sin commitear a propósito, para el commit `docs(sdd)` del orquestador.

### Tareas restantes

Slices 3 a 5 (ramas `feat/us06-storage-estimated-hours`, `feat/us06-http-update-handler`,
`feat/us06-wire-update-route-docs`). El slice 3 requiere Docker.

## Lote 3: Slice 3 (almacenamiento)

Rama `feat/us06-storage-estimated-hours`, creada desde `feat/us06-application-update-story` (apilada,
`stacked-to-main`). No se hizo push ni PR.

### Línea base (tarea 3.1)

Docker Desktop 28.0.4 disponible (Testcontainers `postgres:16-alpine` levanta contenedores). Los tests de
integración de `tests/integration/project/...` y los de repositorio de `tests/integration/story/postgres`
pasan. Única falla de línea base: `TestAPIStartupRoutesFollowMigrationState`, preexistente en Windows (el
test compila el binario como `api` sin `.exe`: `executable file not found in %PATH%`); fuera de alcance,
se aborda en el slice 5.

### Evidencia del ciclo TDD

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 3.2 | `tests/integration/story/postgres/repository_integration_test.go` (lista de migraciones de `storyDatabase`, helper `seedStory`) | Integración | Tests de creación existentes en verde | Falla observada en `TestStoryRepositoryStoresLinkedUnestimatedStory`: `read migration 000003_add_story_estimated_hours.up.sql: ... The system cannot find the file specified.` | Pase con la migración creada | — | — |
| 3.3 | idem (`TestStoryRepositoryUpdateRoundTripsEstimatedHours`: 0.01, 99999.99, 8, 2.5, 0.07) | Integración | N/A (nuevo) | Falla de compilación observada: `Update undefined (type *PostgresStoryRepository has no field or method Update)` | PASS con el paso 1 de la escalera | Ver 3.13 | — |
| 3.4 a 3.9 | idem (`...UpdateReplacesAllEditableFields`, `...UpdateWithNilHoursClearsEstimateToNull`, `...UpdateKeepsIdentityAndStoryPoints`, `...UpdateReportsNotFoundWithoutTouchingData`, `...EnforcesNamedStoryConstraints`, `...UpdateTreatsHostileTextAsData`) | Integración | N/A (nuevos) | Misma falla de compilación (`Update` indefinido); son un único RED por compilación, no siete fallas independientes | PASS | Ver 3.13 | — |
| 3.10 | `internal/project/infrastructure/postgres/migrations/000003_add_story_estimated_hours.{up,down}.sql` | Integración | — | Cubierto por 3.2 | `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` PASS | — | — |
| 3.11 | `internal/story/infrastructure/postgres/repository.go` (`Update` + aserciones de compilación) | Integración | — | Cubierto por 3.3-3.9 | PASS (mismo comando) | — | — |
| 3.12 | idem | Integración | — | — | Paso 1 de la escalera suficiente: parámetro `*float64` y `RETURNING` en `&story.EstimatedHours`; `story_points` NULL se escanea directo en `&story.StoryPoints` (`*int`). Sin casteos ni intermediarios `pgtype`; la columna sigue `NUMERIC(7,2)` | — | — |
| 3.13 | idem (`TestStoryRepositoryUpdateTriangulation`: idempotencia, gana la última, los tres estados; puntos 5 y 13 preservados con horas 1.5; segundo proyecto; contexto cancelado y `CHECK` violado no se etiquetan como no encontrada; rechazos directos de `status`, horas 0, -1 y 100000 -> `22003`) | Integración | — | — | — | PASS de inmediato (la lógica de 3.11 ya era general); son pruebas de caracterización, sin RED nuevo. Comprobación extra de mutación (no commiteada): cambiar el `SET` a `estimated_hours = $8 + 0.01` hizo fallar `lower_bound` e `integer`, y se revirtió | — |
| 3.14 | `internal/story/infrastructure/postgres/repository.go` | Integración | Suite de almacenamiento verde antes y después | — | — | — | Sin cambios necesarios: `Update` es el único escaneo de columnas nullables, no justifica un helper |

Resumen de tests: 8 funciones de test de nivel superior nuevas en el paquete de integración (más 4 helpers:
`readRow`, `seedStory`, `show` y punteros auxiliares). Capa usada: solo integración (PostgreSQL real).
Funciones puras nuevas: ninguna en producción.

### Evidencia del work unit

| Evidencia | Valor observado |
|-----------|-----------------|
| Comando de test focalizado | `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` -> `ok` (todas PASS; 114 s) |
| Harness de runtime | PostgreSQL 16 real vía Testcontainers con migraciones `000001`-`000003`; Docker Desktop 28.0.4; ningún test saltado |
| `go vet ./...` | Limpio |
| `gofmt` | Verificado sobre el contenido sin CR (`tr -d '\r' \| gofmt -l`): limpio. `gofmt -l .` sigue listando archivos por CRLF de la copia de trabajo de Windows (preexistente) |
| `go test ./...` completo | Todos los paquetes unitarios e `integration/project/postgres` `ok`; en `integration/story/postgres` falla SOLO `TestAPIStartupRoutesFollowMigrationState` (preexistente en Windows, sin `.exe`; no causada por este cambio) |
| Rollback | Revertir `2bc1986` (`000003_*.sql`, `Update` en `repository.go` y los tests de integración). Si `000003` ya se aplicó en un entorno, seguir el plan de rollback de esquema de `proposal.md` (el `down` es destructivo) |

### Commit

- `2bc1986` `feat(story): persist story updates and estimated hours` (4 archivos, +430/-1 = 431 líneas autoradas: ~31 de código, 10 de SQL y ~390 de tests de integración).

### Desviaciones del diseño

Ninguna en el comportamiento: escalera en el paso 1, `Update` con una sola sentencia acotada por `id` y
`project_id`, constraints nombradas.

### Problemas encontrados

- **Presupuesto de revisión**: el slice suma 431 líneas autoradas, por encima del pronóstico (~300-350) y de las 400 del presupuesto. La diferencia viene de los tests de integración (~390 líneas; ocho funciones con subtests, cada uno con su contenedor). No se recortaron tests ni comentarios para ajustar la cifra; se recomienda `size:exception` para el PR 3 (o una decisión del usuario), dado que el slice ya es una unidad de trabajo cohesiva.
- `TestAPIStartupRoutesFollowMigrationState` falla en Windows por el nombre del binario (`api` sin `.exe`): preexistente, fuera de alcance del slice 3; queda para el slice 5.
- Los tests de integración tardan ~5 s por contenedor (uno por test/subtest); el paquete completo ronda 2-3 minutos.
- `tasks.md` y `apply-progress.md` quedan sin commitear a propósito, para el commit `docs(sdd)` del orquestador.

### Tareas restantes

Slices 4 y 5 (ramas `feat/us06-http-update-handler`, `feat/us06-wire-update-route-docs`). El arranque real del
slice 5 depende de resolver el nombre del binario en Windows.

## Lote 4: Slice 4 (handler HTTP)

Rama `feat/us06-http-update-handler`, creada desde `feat/us06-storage-estimated-hours` (apilada,
`stacked-to-main`). No se hizo push ni PR.

### Nota de proceso (reanudación)

Un intento anterior de este lote dejó en el árbol de trabajo, sin commitear y sin registro, los tests
(4.2-4.5 y 4.7) y una implementación GREEN con decodificación duplicada. Como no había evidencia de RED
registrada, se **reprodujo** el RED de forma real: se guardaron copias, se revirtió `handler.go` y se
apartó `update_handler.go`, se observaron las fallas y se restauró la implementación. Ninguna evidencia
es inventada; los resultados de abajo son los observados en esta sesión.

### Línea base (tarea 4.1)

`go test ./...` en el estado de la rama del slice 3 (con `git stash -u`): todos los paquetes unitarios e
`integration/project/postgres` `ok`; en `integration/story/postgres` falla solo
`TestAPIStartupRoutesFollowMigrationState` (preexistente en Windows: binario `api` sin `.exe`).

### Evidencia del ciclo TDD

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 4.2 | `tests/unit/story/transport/http/handler_test.go` (`TestCreateStoryReturnsPendingUnestimatedStory`) | Unitario | Suite de creación en verde | Falla observada: `estimated_hours is missing from the response, want null` (`handler_test.go:86`) | Pase con `EstimatedHours *float64` en `storyResponse` | — | — |
| 4.3 | `tests/unit/story/transport/http/update_handler_test.go` (`TestUpdateStoryReturnsTheFullStoredStory`, `...PassesCanonicalIdentifiersAndAllSixFieldsToTheUpdater`, `...MapsOnlyTypedMissingStoryTo404`, `...HidesUnexpectedErrorsBehindAGeneric500`, `...RejectsUnsupportedMethods` (4 subtests)) | Unitario (`httptest`, caso de uso real sobre updater falso) | N/A (archivo nuevo) | Falla de compilación observada: `undefined: transporthttp.NewUpdateStoryHandler` (un único RED por compilación para 4.3-4.5) | PASS | Ver 4.7 | Ver 4.8 |
| 4.4 | idem (`TestUpdateStoryRejectsInvalidJSONShape`, 20 subtests: JSON malformado, vacío, `[]`, `"texto"`, dos objetos, `owner`, `id`/`project_id`/`story_points` con entero, `0` y `null`, tipos incorrectos, `null` en criterios) | Unitario | N/A | Misma falla de compilación | PASS; siempre `calls == 0` | — | — |
| 4.5 | idem (`TestUpdateStoryReportsValidationFieldsBeforeWriting`, 14 subtests: UUID inválidos, cada clave ausente, `{}`, contenido/estado/estimación, id inválido más título vacío) | Unitario | N/A | Misma falla de compilación | PASS; siempre `calls == 0` | — | — |
| 4.6 | `internal/story/transport/http/update_handler.go`, `handler.go` (`storyResponse.EstimatedHours`) | Unitario | — | Cubierto por 4.2-4.5 | `go test ./tests/unit/story/transport/http/...` PASS (14 funciones de test) | — | — |
| 4.7 | idem (`TestUpdateStoryDistinguishesNullFromAbsentEstimatedHours`, `...RespondsFromTheStoredStoryAndPreservesUnicode`, `...PreservesUnicodeSentByTheClient`; mayúsculas normalizadas en `...PassesCanonicalIdentifiers...`; `Content-Type` verificado por los helpers `decode`/`assertError`) | Unitario | — | — | — | Pasaron de inmediato (pruebas de caracterización: la lógica de 4.6 ya distinguía `null` de ausente por el mapa de claves crudas); no hubo RED nuevo | — |
| 4.8 | `handler.go`, `update_handler.go` | Unitario | Suite de creación y de modificación en verde antes y después | — | — | — | `decodeCreateStoryRequest` y `decodeUpdateStoryRequest` unificadas en `decodeStoryObject(body, target) (map[string]json.RawMessage, error)`; `newStoryResponse` usado por ambos handlers; `go test ./tests/unit/...` PASS |

Resumen de tests: 10 funciones de test nuevas en `update_handler_test.go` y 1 aserción ampliada en
`handler_test.go`. Capa usada: solo unitaria. Funciones puras nuevas: `decodeStoryObject`,
`newStoryResponse`.

### Evidencia del work unit

| Evidencia | Valor observado |
|-----------|-----------------|
| Comando de test focalizado | `go test -count=1 ./tests/unit/story/transport/http/...` -> `ok` (14 funciones PASS) |
| Harness de runtime | N/A: `httptest` con el caso de uso real sobre un updater falso; el proceso real se ejerce en el slice 5 (la ruta aún no está registrada) |
| `go vet ./...` | Limpio |
| `gofmt` | Verificado sobre el contenido sin CR (`tr -d '\r' \| gofmt -l`): limpio en `handler.go`, `update_handler.go` y `update_handler_test.go`. `handler_test.go` aparece por un desalineado preexistente en un literal de mapa (línea ~106, no tocada por este slice) |
| `go test -count=1 ./...` completo | Todos los paquetes unitarios e `integration/project/postgres` `ok`; en `integration/story/postgres` falla SOLO `TestAPIStartupRoutesFollowMigrationState` (subtests `version_one`, `version_two`, `dirty`, `lookup_error`): preexistente en Windows (sin `.exe`), fuera de alcance, pendiente para el slice 5. Docker corrió, ningún test saltado |
| Rollback | Revertir `dc42020` (`update_handler.go`, helpers de `handler.go`, sus tests y el ejemplo de creación del README); la ruta no está expuesta hasta el slice 5 |

### Commit

- `dc42020` `feat(story): add update story http handler` (5 archivos, +423/-18 = 441 líneas autoradas: ~122 de código, ~317 de tests, 2 de README).

### Desviaciones del diseño

Ninguna. Decisiones 4, 11 y 12 implementadas tal como están: claves ausentes = seis obligatorias menos las
presentes en el mapa crudo; `errors.As` de `ValidationError` antes que `errors.Is` de `ErrStoryNotFound`;
`405` defensivo en el handler.

### Problemas encontrados

- **Presupuesto de revisión**: el slice suma 441 líneas autoradas, por encima del pronóstico (~300) y de
  las 400 del presupuesto (los tests aportan ~317 líneas: tablas de 20 y 14 subtests que fija la spec). No se
  recortaron tests ni comentarios; se recomienda `size:exception` para el PR 4 (o una decisión del usuario).
- `TestAPIStartupRoutesFollowMigrationState` sigue fallando en Windows (binario sin `.exe`): preexistente; queda para el slice 5.
- `tasks.md` y `apply-progress.md` quedan sin commitear a propósito, para el commit `docs(sdd)` del orquestador.

### Tareas restantes

Slice 5 (rama `feat/us06-wire-update-route-docs`, base `feat/us06-http-update-handler`): tareas 5.1 a 5.11.
