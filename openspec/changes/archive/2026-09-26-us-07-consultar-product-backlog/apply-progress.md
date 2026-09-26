# Progreso de aplicación: US-07 — Consultar el Product Backlog

Cambio: `us-07-consultar-product-backlog` · Rama: `feat/us07-consultar-product-backlog` · Modo: TDD estricto
(`go test ./...`) · Estrategia de entrega: `single-pr` con excepción `size:exception`.

## Lote 1: Unidades 0, 1 y 2

### Unidad 0: compuerta y línea base

#### 0.1 Excepción `size:exception` (compuerta bloqueante)

El usuario **aceptó explícitamente** la excepción `size:exception` para el PR único (~1080 líneas frente al
presupuesto de 400) el 2026-09-26, con `chain_strategy=size-exception` y `delivery_strategy=single-pr`. La
confirmación proviene de la instrucción de lanzamiento del lote 1 y satisface la tarea 0.1. El PR final debe
declarar `size:exception` en su descripción.

#### 0.2 Rama y línea base

- `git branch --show-current` → `feat/us07-consultar-product-backlog` (correcta).
- Docker: `docker info` responde con el cliente y el servidor (Docker Desktop 28.0.4, contexto `desktop-linux`);
  la imagen `postgres:16-alpine` ya estaba disponible y Testcontainers pudo crear contenedores.
- `go test ./...` **antes de tocar código** (con Docker disponible, `-count=1`):
  - Unitarios: `ok` en `tests/unit/cmd/api`, `tests/unit/project/{application,domain,transport/http}` y
    `tests/unit/story/{application,domain,transport/http}`.
  - Integración: **los tests SÍ corrieron (no se saltaron) pero hay fallas preexistentes de entorno**:
    - `tests/integration/project/postgres`: `TestPostgresProjectRepositoryCreatePersistsProject`,
      `TestPostgresProjectRepositoryUpdateChangesBasicFieldsOnly` y
      `TestProjectsMigrationRejectsFinishDateBeforeStartDate` fallan con
      `wait for PostgreSQL test container: failed to connect ... [::1]:<puerto>: unexpected EOF`
      (conexión al contenedor por `localhost`/IPv6 con Docker Desktop recién iniciado).
    - `tests/integration/story/postgres`: `TestAPIStartupRoutesFollowMigrationState` falla en los subtests
      `version_one`, `version_two`, `version_three`, `dirty` y `dirty_version_three` (pasa `lookup_error`);
      es la falla preexistente de Windows ya conocida desde US-06 (binario `api` sin `.exe`) y/o la misma
      conexión inestable.
  - Estas fallas ocurren en el árbol limpio, antes de cualquier cambio de US-07: son **límites de entorno
    preexistentes**, no resultado de este trabajo. Se reportan tal cual y no se corrigen (fuera de alcance).
- Notas de entorno relevantes:
  - `gofmt -l .` lista casi todos los archivos `.go` porque el árbol de trabajo está en CRLF
    (`core.autocrlf=true`; el repositorio guarda LF). Es ruido preexistente. La verificación de formato de
    este lote se hace normalizando el fin de línea (`tr -d '\r' | gofmt -l`) sobre los archivos tocados.
  - `go vet ./...` limpio.

#### 0.3 Commit de planificación

Commit `8159ccf` `docs(sdd): add US-07 planning artifacts for backlog query` (solo archivos bajo `openspec/changes/us-07-consultar-product-backlog/`).

### Unidad 1: Dominio (precedencia de prioridad y `Backlog`)

Ciclo TDD observado (todo ejecutado de verdad):

- **Red de seguridad** (archivos existentes tocados: `story.go`): `go test ./tests/unit/story/domain/...` antes
  de tocar nada → `ok`, 12 tests de nivel superior pasando.
- **RED**: `tests/unit/story/domain/backlog_test.go` con `TestNewBacklogOrdersByPriority`,
  `TestNewBacklogIsStableForEqualPriorities`, `TestNewBacklogDoesNotMutateInput`,
  `TestNewBacklogPreservesProjectID`, `TestNewBacklogHandlesNilAndEmpty` y
  `TestAllowedPrioritiesReturnsOrderedCopy`. Falla observada (error de compilación): `undefined:
  domain.NewBacklog` y `undefined: domain.AllowedPriorities` → `FAIL ... [build failed]`.
- **GREEN**: constantes `PriorityHigh/Medium/Low` y `AllowedPriorities()` en `story.go`; `backlog.go` con
  `Backlog`, `NewBacklog` (`slices.Clone` + `slices.SortStableFunc`) y `priorityRank`. Resultado: `ok`, los seis
  tests pasan (incluido el escenario S1…S5 → S2, S5, S1, S3, S4).
- **TRIANGULATE**: `TestNewBacklogTriangulatesSpecScenarios` con misma prioridad (tres `media`), reubicación por
  prioridad (S1 `media`, S2 `alta`, S3 `alta` → S2, S3, S1), prioridad desconocida al final conservando el
  orden de entrada (`urgente`, `critica`) y una sola historia. Resultado: `ok` sin cambios de producción
  (la implementación ya generalizaba).
- **REFACTOR**: `validateStoryContent` valida ahora con `slices.Contains(AllowedPriorities(), priority)` y arma
  el mismo mensaje `must be alta, media or baja` con las constantes. Suite de dominio verde antes (12/12) y
  después; el resto de `go test ./tests/unit/...` verde.
- Verificación: `go vet ./...` limpio; formato normalizado sin diferencias; `go test ./tests/unit/...` verde.
  La integración quedó como en la línea base (fallas de entorno preexistentes descritas arriba); la unidad no
  toca almacenamiento ni composición.

Work Unit Evidence (Unidad 1):

| Evidencia | Valor |
|---|---|
| Comando focalizado y resultado | `go test ./tests/unit/story/domain/... -count=1` → `ok` (12 tests previos + 7 nuevos de nivel superior) |
| Arnés de ejecución | N/A: funciones puras sin frontera de proceso; la evidencia es la suite unitaria |
| Frontera de rollback | `internal/story/domain/story.go`, `internal/story/domain/backlog.go`, `tests/unit/story/domain/backlog_test.go` |

Commit de la Unidad 1: `5b49131` `feat(story): order the product backlog by priority`.

### Unidad 2: Aplicación (`ListStoriesUseCase`)

Ciclo TDD observado:

- **Red de seguridad** (no se modifican archivos existentes; solo se agregan): `go test
  ./tests/unit/story/application/...` sin el archivo nuevo → `ok`, 13 tests de nivel superior pasando.
- **RED**: `tests/unit/story/application/list_stories_test.go` con un fake de `StoryLister` escrito a mano
  (contador de llamadas, `projectID` capturado, solo lectura). Falla observada (error de compilación):
  `undefined: application.NewListStoriesUseCase` y `undefined: application.ListStoriesQuery` → `FAIL ... [build
  failed]`. Desvío menor de proceso: el archivo se escribió de una vez con los casos base (2.1) **y** los de
  triangulación (2.3, 2.4); el RED observado cubre todo el archivo, y el GREEN se limitó a la implementación
  mínima del diseño, sin agregar lógica que los casos base no exigieran.
- **GREEN**: `internal/story/application/list_stories.go` con `ListStoriesQuery`, `StoryLister`,
  `ListStoriesUseCase`, `NewListStoriesUseCase` y `Execute` (`uuid.Parse` antes del puerto; identificador
  canónico al puerto; error del puerto con `domain.Backlog{}`; `domain.NewBacklog` sobre el resultado).
  `ErrProjectNotFound` se reutiliza sin modificar `create_story.go`. Resultado: los seis tests nuevos pasan.
- **TRIANGULATE**: tabla de identificadores hostiles (`no-es-uuid`, `abc`, `123`, UUID truncado, vacío) con
  cero llamadas; UUID en mayúsculas → una llamada con el canónico y `Backlog.ProjectID` canónico;
  `ErrProjectNotFound` y error inesperado propagados con `errors.Is` y `Backlog{}` aunque el fake devuelva
  historias junto al error; proyecto sin historias (`nil`) sin error. Sin cambios de producción necesarios.
- **Prueba de mutación manual** (para verificar que las aserciones muerden): se cambió temporalmente el retorno
  de error para devolver el `Backlog` parcial → `TestListStoriesPropagatesPortErrorsWithoutPartialBacklog`
  falló en sus dos subtests; se restauró el código y volvió a `ok`.
- **REFACTOR**: sin cambios necesarios (una sola función corta, sin duplicación real). `go test
  ./tests/unit/story/application/...` verde antes y después; `go test ./tests/unit/...` verde.
- Verificación: `go vet ./...` limpio; formato normalizado sin diferencias; `create_story_test.go`,
  `handler_test.go` y `main_test.go` siguen compilando y pasando sin modificaciones (el puerto es independiente
  de `StoryRepository`). La integración quedó como en la línea base (fallas de entorno preexistentes).

Work Unit Evidence (Unidad 2):

| Evidencia | Valor |
|---|---|
| Comando focalizado y resultado | `go test ./tests/unit/story/application/... -count=1` → `ok` (13 tests previos + 6 nuevos de nivel superior) |
| Arnés de ejecución | N/A: caso de uso sobre fake, sin frontera de proceso ni de red |
| Frontera de rollback | `internal/story/application/list_stories.go`, `tests/unit/story/application/list_stories_test.go` |

## Tabla de evidencia TDD (lote 1)

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 1.1–1.2 | `tests/unit/story/domain/backlog_test.go` | Unitario | 12/12 | Escrito; build failed (símbolos indefinidos) | Pasó | ver 1.4 | ver 1.5 |
| 1.3–1.4 | `tests/unit/story/domain/backlog_test.go` | Unitario | 12/12 | (ver arriba) | Pasó (7 tests) | 4 casos extra, pasaron | — |
| 1.5 | `tests/unit/story/domain/story_test.go` (existente) | Unitario | 12/12 antes | N/A (refactor) | N/A | N/A | `validateStoryContent` sobre `AllowedPriorities()`, 12/12 después |
| 2.1–2.2 | `tests/unit/story/application/list_stories_test.go` | Unitario | 13/13 | Escrito; build failed | Pasó (6 tests) | — | — |
| 2.3–2.4 | `tests/unit/story/application/list_stories_test.go` | Unitario | 13/13 | (escritos junto a 2.1) | Pasaron sin cambiar producción | 5 ids hostiles, mayúsculas, 2 errores, vacío | Sin cambios necesarios |

## Estado del lote 1

Tareas completadas: 0.1–0.3, 1.1–1.7 y 2.1–2.7 (17 de las 17 asignadas en las unidades 0–2). Pendientes: unidades 3–6
(la unidad 3 y el arranque real de la 5 requieren Docker; en la línea base la integración muestra fallas de
entorno preexistentes, ver 0.2).

## Lote 2: Unidades 3 y 4

Continuidad: el lote 1 (Unidades 0-2, commits `8159ccf`, `5b49131`, `a39f3ae`) se leyó y se conserva sin cambios.
Este lote agrega las Unidades 3 y 4. La excepción `size:exception` (0.1) sigue vigente para el PR único.

### Unidad 3: Almacenamiento y migración `000004` (requiere Docker)

#### 3.1 Docker

`docker info` responde (servidor 28.0.4, Docker Desktop). Los tests de integración **corrieron de verdad**
(Testcontainers `postgres:16-alpine`); ninguno se saltó. Las fallas transitorias del lote 1 (arranque de
Docker) no reaparecieron.

#### Ciclo TDD observado

- **Red de seguridad** (se modifica `repository_integration_test.go` y `repository.go`):
  `go test -count=1 ./tests/integration/story/postgres/... -run "TestStoryRepository|TestStories"` antes de
  tocar nada → `ok` (114 s), con los tests de creación, FK, concurrencia y modificación en verde.
- **RED 3.2 (migraciones)**: se agregó `000004_add_story_creation_sequence.up.sql` a la lista de
  `storyDatabase` y se ejecutó `TestStoryRepositoryStoresLinkedUnestimatedStory` → `FAIL`: `read migration
  000004_add_story_creation_sequence.up.sql: ... The system cannot find the file specified` (el contenedor
  arrancó y la falla es la del archivo inexistente, no del entorno). Se extrajo el helper de test
  `applyStoryMigration` (sin cambio de comportamiento) para reutilizarlo en la prueba de `down`/`up`.
- **RED 3.3–3.11**: se escribieron los tests del grupo (proyecto vacío —el riesgo del `NULL` con pgx—,
  inexistente, orden de creación con `id` y prioridad desalineados, aislamiento entre proyectos, mapeo
  completo con `NULL`, `Update` no altera `seq`, instantánea de solo lectura con `xmin`, catálogo
  `stories_project_id_seq_key` + identidad `ALWAYS` + `428C9`, y error no reinterpretado con `seq`
  renombrada). Falla observada: error de compilación `PostgresStoryRepository does not implement
  application.StoryLister (missing method ListByProject)` → `FAIL ... [build failed]`. Desvío menor de
  proceso: como el paquete de integración es uno solo, el RED de todo el grupo se observó como un único fallo
  de compilación (no una falla de ejecución por test).
- **GREEN 3.12–3.13**: migraciones `000004` `up` (columna `seq BIGINT GENERATED ALWAYS AS IDENTITY` y
  `stories_project_id_seq_key UNIQUE (project_id, seq)`) y `down` (constraint y luego columna);
  `ListByProject` con la sentencia única `projects LEFT JOIN stories ... ORDER BY s.seq`, fila privada
  `listedStory` con destinos anulables, `rows.Err()` antes de devolver, `nil` ante cualquier error y la
  aserción de compilación `application.StoryLister`. Resultado: los 11 tests del grupo pasan a la primera
  (`-v`: todos `PASS`, 50 s).
- **3.14 Escalera de fallback**: **paso 1 suficiente**. El escaneo de `NULL` del `LEFT JOIN` en destinos
  `*string` funcionó con pgx sin necesidad de `s.id::text` ni de `pgtype`; `TestStoryRepositoryListByProject
  ReturnsEmptyForProjectWithoutStories` (el primer riesgo técnico) pasó tal cual.
- **TRIANGULATE 3.15–3.16**: `TestStoriesCreationSequenceMigrationRoundTripsWithoutLosingData` (`down` sobre una
  base con dos historias elimina columna y constraint sin perder filas ni valores; `up` de nuevo asigna un
  `seq` distinto a cada fila) y `TestStoryRepositoryListByProjectKeepsCreationOrderWithinEqualPriority` (cuatro
  historias de igual prioridad creadas por el repositorio real conservan el orden de creación; dos consultas
  consecutivas idénticas). Pasaron sin cambios de producción.
- **Prueba de mutación manual (hallazgo)**: al quitar temporalmente `ORDER BY s.seq`, los tests de orden
  **seguían pasando**: el índice `UNIQUE (project_id, seq)` devuelve por sí solo las filas en orden de `seq`,
  de modo que la sentencia parecía correcta sin su propio `ORDER BY`. Se agregó
  `TestStoryRepositoryListByProjectOrdersBySequenceItself`: siembra filas con `seq` explícito
  (`OVERRIDING SYSTEM VALUE`) cuyo orden físico es el inverso de su `seq` y consulta por un pool con
  `enable_indexscan`, `enable_indexonlyscan` y `enable_bitmapscan` desactivados. Con la mutación el test
  **falla**; con `ORDER BY s.seq` restaurado, **pasa**. El código de producción quedó idéntico al original.
- **REFACTOR 3.17**: sin cambios necesarios. `Update` escanea 9 columnas no anulables directo sobre
  `domain.Story` y `ListByProject` usa destinos anulables por la fila de relleno del `LEFT JOIN`; un helper
  compartido no reduciría duplicación real. La suite completa quedó verde antes y después.
- **Verificación 3.18**: `go vet ./...` limpio; formato normalizado (`tr -d '\r' | gofmt -l`) sin
  diferencias; `go test ./tests/unit/...` verde; `go test -count=1 -v ./tests/integration/...` con Docker:
  `ok` en `project/postgres` (29 s) y `ok` en `story/postgres` (284 s), **31 tests de nivel superior y 29
  subtests en `PASS`, 0 `FAIL`, 0 `SKIP`**, incluido `TestAPIStartupRoutesFollowMigrationState` con sus seis
  subtests (que en el lote 1 mostraba fallas transitorias de Docker).

Work Unit Evidence (Unidad 3):

| Evidencia | Valor |
|---|---|
| Comando focalizado y resultado | `go test -count=1 ./tests/integration/story/postgres/... -run "TestStoryRepositoryListByProject\|TestStoriesCreationSequence\|TestStoryRepositoryUpdateDoesNotChangeCreationSequence" -v` → `PASS` los 12 tests nuevos (el 12.º, `...OrdersBySequenceItself`, ejecutado aparte) |
| Arnés de ejecución | PostgreSQL real vía Testcontainers `postgres:16-alpine` con migraciones `000001`–`000004`; suite completa de integración `PASS`, sin saltos |
| Frontera de rollback | migraciones `000004` (`up`/`down`), `ListByProject` en `internal/story/infrastructure/postgres/repository.go` y los tests de `tests/integration/story/postgres/repository_integration_test.go`. Si `000004` ya se aplicó en un entorno, ejecutar el `down` (solo se pierde la secuencia de desempate) |

### Unidad 4: Handler HTTP `ListStoriesHandler` (sin Docker)

Ciclo TDD observado:

- **Red de seguridad**: no se modifica ningún archivo existente (`handler.go` y `update_handler.go` quedan
  intactos; solo se agregan `list_handler.go` y su test). La suite unitaria del transporte de historias estaba
  verde antes (`ok`, 14 tests de nivel superior).
- **RED 4.1–4.3**: `tests/unit/story/transport/http/list_handler_test.go` con un `fakeLister` de solo lectura y
  el caso de uso **real** (`application.NewListStoriesUseCase`): `TestListStoriesReturnsTheOrderedBacklogIn
  sideTheContainer` (200, `project_id` canónico en minúsculas, orden `alta, media, baja` a partir de un fake en
  orden de creación, `null` en los anulables), `TestListStoriesRespondsWithAnEmptyArrayNeverNull` (fake que
  devuelve `nil`; el cuerpo crudo contiene `"stories":[]` y no `null`), `...RejectsInvalidProjectIdentifiers
  BeforeReading` (`no-es-uuid` y `abc` → 422 con `fields.project_id` y cero lecturas), `...MapsOnlyTheTypedMissing
  ProjectTo404` (sin clave `stories`) y `...HidesUnexpectedErrorsBehindAGeneric500`. Falla observada (error de
  compilación): `undefined: transporthttp.NewListStoriesHandler` → `FAIL ... [build failed]`.
- **GREEN 4.4**: `internal/story/transport/http/list_handler.go` con `NewListStoriesHandler`,
  `ListStoriesHandler`, `backlogResponse` y el mapa de errores del diseño (`errors.As` de `ValidationError`
  antes que `errors.Is` de `ErrProjectNotFound`; respuesta con `make([]storyResponse, 0, len(backlog.Stories))`);
  reutiliza sin cambios `storyResponse`, `newStoryResponse`, `errorResponse` y `writeJSON`. Resultado: los cinco
  tests pasan.
- **TRIANGULATE 4.5**: nueve claves exactas por historia y contenedor de exactamente dos claves sin `seq`;
  identificador canónico entregado al lector (una lectura, en minúsculas); `HEAD` aceptado con una lectura y
  `POST`/`PUT`/`DELETE` directos al handler → `405 method_not_allowed` con cero lecturas; un lector que devuelve
  historias **junto con** un error produce `500` sin clave `stories`; `Content-Type: application/json` en `200`,
  `404` y `500`. Pasaron sin cambios de producción.
- **Prueba de mutación manual**: cambiar temporalmente `make([]storyResponse, 0, ...)` por `var stories
  []storyResponse` hace fallar `TestListStoriesRespondsWithAnEmptyArrayNeverNull` (el cuerpo pasa a `null`);
  se restauró el código y volvió a `ok`.
- **REFACTOR 4.6**: sin cambios necesarios (los helpers de `handler.go` se reutilizan sin modificarlos; no hay
  duplicación real). Suite de transporte verde antes y después.
- **Verificación 4.7**: `go vet ./...` limpio; formato normalizado sin diferencias; `go test ./tests/unit/...`
  verde (transporte de historias: 24 tests de nivel superior, 10 nuevos); `internal/story/transport/http/
  handler.go` sin modificar (`git diff` vacío).
- **`go test -count=1 ./...` completo con Docker (cierre de la Unidad 4)**: unitarios todos `ok`;
  `tests/integration/story/postgres` `ok` (251 s, sin `SKIP`); `tests/integration/project/postgres` falló una
  vez en `TestPostgresProjectRepositoryUpdateChangesBasicFieldsOnly` con `wait for PostgreSQL test container:
  failed to connect ... unexpected EOF` (la misma **falla transitoria de conexión al contenedor** ya vista en
  la línea base del lote 1; es un paquete que esta unidad no toca). Se re-ejecutó
  `go test -count=1 ./tests/integration/project/...` → `ok` (16 s). Límite de entorno, no resultado de US-07.

Work Unit Evidence (Unidad 4):

| Evidencia | Valor |
|---|---|
| Comando focalizado y resultado | `go test -count=1 ./tests/unit/story/transport/http/...` → `ok` (14 tests previos + 10 nuevos de nivel superior) |
| Arnés de ejecución | N/A: `httptest` con el caso de uso real sobre un lister falso; el proceso real se ejerce en la Unidad 5 (la ruta no está expuesta hasta entonces) |
| Frontera de rollback | `internal/story/transport/http/list_handler.go` y `tests/unit/story/transport/http/list_handler_test.go` |

## Tabla de evidencia TDD (lote 2)

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 3.2 | `tests/integration/story/postgres/repository_integration_test.go` | Integración | `-run "TestStoryRepository\|TestStories"` `ok` | `read migration ... cannot find the file` | Pasó con la migración | — | Helper `applyStoryMigration` extraído (test) |
| 3.3–3.11 | ídem | Integración | ídem | Escritos; `build failed` (`ListByProject` indefinido) | Pasaron 11 tests | ver 3.15–3.16 | — |
| 3.12–3.14 | ídem (migraciones y `repository.go`) | Integración | ídem | (ver arriba) | Pasó con el paso 1 de la escalera | — | — |
| 3.15–3.16 | ídem | Integración | ídem | Escritos junto al grupo | Pasaron sin cambiar producción | `down`/`up`, igual prioridad, consultas repetidas; **mutación de `ORDER BY`** detectó un hueco y se agregó `OrdersBySequenceItself` | Sin cambios necesarios (3.17) |
| 4.1–4.3 | `tests/unit/story/transport/http/list_handler_test.go` | Unitario | N/A (archivo nuevo; 14/14 previos) | Escrito; `build failed` (`NewListStoriesHandler` indefinido) | Pasó (5 tests) | — | — |
| 4.5 | ídem | Unitario | ídem | (escritos tras el GREEN de 4.4) | Pasaron sin cambiar producción | 9 claves, `seq` ausente, `HEAD`, `405`, error con historias, `Content-Type` | Sin cambios necesarios (4.6); mutación de `make(...)` detectada |

## Estado acumulado (lotes 1 y 2)

Tareas completadas: 0.1–0.3, 1.1–1.7, 2.1–2.7, 3.1–3.19 y 4.1–4.8. Pendientes: Unidad 5 (composición, arranque y
README; requiere Docker) y Unidad 6 (cierre). Commits de unidad: `5b49131` (1), `a39f3ae` (2), `0a9ebe9` (3) y
`386d8ab` (4).

## Lote 3: Unidades 5 y 6

Continuidad: los lotes 1 y 2 (commits `8159ccf`, `5b49131`, `a39f3ae`, `0a9ebe9`, `386d8ab`) se leyeron y se
conservan sin cambios. La excepción `size:exception` (0.1) sigue vigente para el PR único.

### Unidad 5: Composición, arranque y documentación (requiere Docker)

Ciclo TDD observado:

- **Red de seguridad** (se modifican `api.go`, `main.go` y `main_test.go`): `go test -count=1
  ./tests/unit/cmd/api/...` antes de tocar nada → `ok`, 13 tests de nivel superior en verde.
- **RED 5.1–5.2 (unitario)**: en `tests/unit/cmd/api/main_test.go` se agregaron `fakeStoryLister`,
  `newBacklogHandler` (sin tocar `newStoryHandler`) y cinco tests: `TestBacklogCompositionServesTheOrderedContainer
  OnCollectionGet` (200, `project_id` canónico aunque la ruta lleve mayúsculas, orden `alta, media, baja`),
  `...MapsCollectionGetErrors` (404 sin `stories`, 422 con cero lecturas, 500 sin el texto interno),
  `...KeepsTheRouteBoundaries` (`HEAD` 200, `POST` 201, `PUT`/`DELETE`/`PATCH` colección 405, `GET` ítem 405,
  `/stories/` 404, cero lecturas donde no corresponde), `...KeepsStoryUpdateWorking` y
  `TestCompositionWithoutListerDoesNotExposeTheBacklog` (sin `Lister`: `GET`/`HEAD` 405 con y sin `Updater`; solo
  proyectos: `GET` 404). Se agregó un comentario de una línea en los tests existentes (líneas ~117 y ~173) que
  explica que representan composiciones sin consulta; **sus aserciones no cambiaron** (Decisión 8). Falla
  observada (error de compilación): `unknown field Lister in struct literal of type api.StoryDependencies` →
  `FAIL ... [build failed]`. Nota de proceso: los tests de 5.2 (sin `Lister`) ya se cumplían por diseño; su RED
  es el mismo error de compilación del archivo y actúan como guarda de regresión, no como comportamiento nuevo.
- **RED 5.3–5.4 (integración)**: en `tests/integration/story/postgres/http_integration_test.go` se agregaron los
  escenarios `version four` y `dirty version four`, el log esperado de `version three` pasó a `story backlog
  unavailable`, `testAPIStartupRoutes` recibió el parámetro `lists` y se escribió `assertBacklogRoute`, más el
  test de punta a punta `TestBacklogHTTPOrdersByPriorityThenCreationEndToEnd`. Falla observada primero como error
  de compilación (`unknown field Lister`).
- **GREEN 5.5 (`api.go`)**: campo aditivo `Lister` en `StoryDependencies` y registro condicional de `GET
  /projects/{project_id}/stories` con `NewListStoriesHandler`; comentarios actualizados con `000004`. Resultado:
  `go test -count=1 ./tests/unit/cmd/api/...` → `ok`, 18 tests de nivel superior (13 previos + 5 nuevos).
- **RED de comportamiento real (entre 5.5 y 5.6)**: con `api.go` listo y `main.go` **sin** actualizar, `go test
  -count=1 ./tests/integration/story/postgres/... -run "TestAPIStartup|TestBacklogHTTP" -v` mostró la falla
  esperada: `version_three` (log `story creation and update available` en lugar de `story backlog unavailable`)
  y `version_four` (log sin `backlog` y `GET collection = 405`) en `FAIL`; `version_one`, `version_two`, `dirty`,
  `dirty_version_three`, `dirty_version_four` y `lookup_error` en `PASS`, y el test de punta a punta en `PASS`
  (el handler en proceso ya funcionaba con `Lister`).
- **GREEN 5.6 (`main.go`)**: rama `err == nil && !dirty && version >= 4` con el mismo `*PostgresStoryRepository`
  como `Repository`, `Updater` y `Lister` y log `story creation, update and backlog available (schema
  version=%d)`; el log de la rama `>= 3` pasó a `story backlog unavailable until migration 000004 is clean
  (version=%d)`; comentario del gate actualizado. Resultado: **PASS real** del arranque:
  `TestAPIStartupRoutesFollowMigrationState` con sus ocho subtests (`version_one`, `version_two`,
  `version_three`, `version_four`, `dirty`, `dirty_version_three`, `dirty_version_four`, `lookup_error`) y
  `TestBacklogHTTPOrdersByPriorityThenCreationEndToEnd`, sin `SKIP` (71 s).
- **TRIANGULATE 5.7**: cubierto por los escenarios anteriores y por `assertBacklogRoute`: con la consulta
  disponible, `GET` `200` con la historia modificada (`Modificada`, `completada`) y `project_id` del proyecto,
  `HEAD` `200` sin cuerpo, `PUT`/`DELETE` colección `405`, `/stories/` `404`, proyecto desconocido `404
  project_not_found` sin `stories`, identificador inválido `422`; `GET` del ítem `405` (ya en
  `assertStoryUpdateRoute`); creación y modificación siguen funcionando con esquema 3 y 4; versión 4 `dirty` no
  expone rutas de historia; el test de punta a punta crea un proyecto ajeno con una historia y comprueba que su
  backlog contiene solo su historia y que no se filtra al del proyecto consultado. Orden de punta a punta: S1…S5
  → `S2, S5, S1, S3, S4`; S3 a `alta` → `S2, S3, S5, S1, S4`; renombrar S1 y pasarla a `completada` no cambia su
  posición; proyecto vacío → `"stories":[]`; inexistente → 404 sin `stories`; inválido → 422; dos `GET`
  consecutivos con cuerpos idénticos.
- **5.8 README**: nueva sección "Consultar el Product Backlog" (operación y ejemplo, orden, lista vacía, errores,
  ausencia de paginación y filtros y respuesta sin límite, migración `000004` y gate `>= 4`, orden de
  despliegue, nota del bloqueo `ACCESS EXCLUSIVE` al agregar la columna de identidad, limitación del orden de
  las filas previas y reversión); se actualizaron la introducción, los requisitos previos y el paso de
  migraciones (ahora incluyen `000004` y el caso `405` con esquema 2 o 3).
- **5.9 delta de la spec**: el delta describe lo implementado y no se modificó. **Precisión de redacción
  pendiente para el archivado** (sin cambiar el comportamiento): el requisito de disponibilidad dice que con
  versión inferior a 4 o `dirty` `GET` sobre la colección responde `405`; en realidad responde `405` solo con
  versión 2 o 3 sin `dirty`, y `404` sin ninguna ruta de historias (versión 1, `dirty` o error de lectura). La
  sugerencia de redacción quedó anotada en `tasks.md` (5.9).
- **REFACTOR 5.10**: sin cambios necesarios. Las cuatro ramas de `main.go` son la tabla explícita del diseño
  (Decisión 7); unificarlas con `if` anidados sobre la versión ocultaría el gate y solo ahorraría un literal de
  `StoryDependencies`. Suite unitaria de `cmd/api` verde antes y después.
- **Verificación 5.11**: `go vet ./...` limpio; formato normalizado (`tr -d '\r' | gofmt -l`) sin diferencias en
  los cuatro archivos `.go` tocados; `go test -count=1 ./...` con Docker todo `ok` (ver Unidad 6).

Work Unit Evidence (Unidad 5):

| Evidencia | Valor |
|---|---|
| Comando focalizado y resultado | `go test -count=1 ./tests/unit/cmd/api/...` → `ok` (13 tests previos + 5 nuevos de nivel superior) |
| Arnés de ejecución | Binario compilado por el propio test contra PostgreSQL real (Testcontainers `postgres:16-alpine`): `go test -count=1 ./tests/integration/story/postgres/... -run "TestAPIStartup\|TestBacklogHTTP" -v` → `PASS` (8 subtests de arranque + 1 test de punta a punta, sin `SKIP`) |
| Frontera de rollback | `internal/api/api.go`, `cmd/api/main.go`, `tests/unit/cmd/api/main_test.go`, `tests/integration/story/postgres/http_integration_test.go` y `README.md`. Revertir el commit quita la ruta y deja creación y modificación intactas; el esquema puede quedar en versión 4 |

## Tabla de evidencia TDD (lote 3)

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 5.1–5.2 | `tests/unit/cmd/api/main_test.go` | Unitario | 13/13 | Escritos; `build failed` (`Lister` indefinido) | Pasaron (5 nuevos, 18/18) con `api.go` | Errores 404/422/500, métodos, `HEAD`, `/stories/`, ítem, sin `Lister` con y sin `Updater` | Sin cambios necesarios (5.10) |
| 5.3–5.4 | `tests/integration/story/postgres/http_integration_test.go` | Integración | N/A (escenarios nuevos; los existentes no cambian de aserción) | `build failed`; luego `version_three` y `version_four` en `FAIL` con `main.go` sin actualizar | 8 subtests de arranque y el test de punta a punta en `PASS` con `main.go` | Proyecto ajeno, vacío, inexistente, inválido, consultas idénticas, S3 sube, S1 renombrada | Sin cambios necesarios |
| 5.5–5.6 | `api.go`, `main.go` | — | 13/13 | (ver arriba) | Ver arriba | — | — |
| 5.8–5.9 | `README.md`, `tasks.md` | Docs | N/A | N/A | N/A | N/A | N/A |

### Unidad 6: Cierre y verificación completa

- **6.1 README**: revisado; **README completo**. La sección "Consultar el Product Backlog" documenta la consulta y su
  ejemplo, el orden, la lista vacía `[]`, los errores (`404`, `422`, `500`), la ausencia de paginación y filtros
  (y de límite en la respuesta), la migración `000004` y su gate `>= 4`, el orden de despliegue, el bloqueo
  `ACCESS EXCLUSIVE`, la limitación del orden de las filas previas y la reversión. No hizo falta un commit
  `docs(readme)`.
- **6.2 Verificación completa** (con Docker, sobre el árbol del commit `ca3cdec`):
  - `go vet ./...` → sin salida, código de salida 0.
  - `go test -count=1 ./...` → **todos `ok`**: `tests/integration/project/postgres` (29 s),
    `tests/integration/story/postgres` (246 s), `tests/unit/cmd/api`, `tests/unit/project/{application,domain,
    transport/http}` y `tests/unit/story/{application,domain,transport/http}`; los paquetes `internal/...` y
    `cmd/api` no tienen archivos de prueba.
  - Una segunda corrida completa con `-v` para contar resultados: `--- PASS` 384 (tests y subtests), `--- SKIP`
    **0**, `--- FAIL` 1. La única falla fue `TestPostgresProjectRepositoryUpdateChangesBasicFieldsOnly` con
    `failed to receive message: unexpected EOF` al conectar al contenedor (la falla transitoria de entorno ya
    conocida, en un paquete que US-07 no toca); `go test -count=1 ./tests/integration/project/...` → `ok` (17 s)
    al reintentar. **Límite de entorno**, no resultado de US-07. Ninguna prueba se saltó.
  - Formato: `tr -d '\r' | gofmt -l` sobre todos los `.go` versionados solo lista
    `tests/unit/project/domain/project_test.go` y `tests/unit/story/transport/http/handler_test.go`, ambos
    **sin cambios respecto de `main`** (deriva de formato preexistente, fuera de alcance; los archivos tocados por
    US-07 están limpios).
- **6.3 Criterios de éxito de `proposal.md` contra la evidencia observada**:

| Criterio | Evidencia |
|---|---|
| `200` con el contenedor y la representación completa | `TestListStoriesReturnsTheOrderedBacklogInsideTheContainer` (Unidad 4), nueve claves exactas; `assertBacklogRoute` y `TestBacklogHTTPOrdersByPriorityThenCreationEndToEnd` contra PostgreSQL |
| Orden por prioridad y creación, contra PostgreSQL | Unidad 1 (dominio, S1…S5), Unidad 3 (`ORDER BY s.seq`, con la mutación detectada) y Unidad 5 (`S2, S5, S1, S3, S4` y `S2, S3, S5, S1, S4` de punta a punta) |
| Proyecto vacío → `"stories":[]` | `TestListStoriesRespondsWithAnEmptyArrayNeverNull` (fake `nil`) y el test de punta a punta (proyecto vacío real) |
| `404` / `422` / `500` | Unidades 2 y 4 (cero lecturas ante `422`, `500` sin el texto interno), Unidad 3 (error de base no reinterpretado) y Unidad 5 (`404` sin `stories`, `422`) |
| Solo lectura | Unidad 3: instantánea con `xmin` y `to_jsonb` idéntica antes y después; fake de solo lectura en las Unidades 2 y 4 |
| Aislamiento entre proyectos | Unidad 3 (dos proyectos) y Unidad 5 (proyecto ajeno con historia; su backlog contiene solo la suya) |
| Gate `>= 4` y rutas vecinas | `TestAPIStartupRoutesFollowMigrationState` (`version_three` sin consulta con `405`, `version_four` con consulta, `dirty_version_four` sin rutas), `PUT`/`DELETE` colección `405` y `GET` ítem `405`, creación y modificación con esquema 3 y 4 |

  `git log` de la rama: `8159ccf` (planificación), `5b49131`, `a39f3ae`, `0a9ebe9`, `386d8ab` y `ca3cdec`
  (unidades 1 a 5). Cada commit de código tiene un cuerpo de 6 líneas no vacías (título `TDD trace`, 4 viñetas y
  `Refs: #35`), el de planificación 3, y **ninguno** contiene `Co-Authored-By` ni atribución de IA.

### Riesgos y notas para el archivado

- **Tamaño**: el commit de la Unidad 5 suma 483 inserciones y 31 eliminaciones (514 líneas) frente a las ~260
  estimadas, porque `assertBacklogRoute`, el test de punta a punta y la sección del README crecieron más que el
  pronóstico; la rama completa frente a `main` es de ~3640 líneas incluyendo los documentos de planificación.
  Cubierto por la excepción `size:exception` aceptada en 0.1; el PR debe declararla.
- **Redacción de la spec**: precisión pendiente para el archivado (ver 5.9): con versión 1, `dirty` o error de
  lectura la ruta de colección responde `404` (no hay ruta de historias) y solo con versión 2 o 3 sin `dirty`
  responde `405`.
- **Límites de entorno**: falla transitoria `unexpected EOF` al conectar al contenedor en `project/postgres` (pasa
  al reintentar); CRLF en el árbol de trabajo (`core.autocrlf=true`), por lo que el formato se verifica con
  `tr -d '\r' | gofmt -l`.

## Estado final (lotes 1, 2 y 3)

Tareas completadas: **todas** (0.1–0.3, 1.1–1.7, 2.1–2.7, 3.1–3.19, 4.1–4.8, 5.1–5.12 y 6.1–6.3). Sin tareas
pendientes. Siguiente fase recomendada: `sdd-archive` (la verificación es opcional).
