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
