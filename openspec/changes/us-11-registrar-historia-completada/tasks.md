# Tareas: Registrar una historia del Sprint como completada (US-11)

## Convención de commits

- Cada Unidad de Trabajo cierra con **un único commit** en la rama `feat/us11-registrar-historia-completada`, con Conventional Commits. Las pruebas y la documentación viajan en el mismo commit que el comportamiento que cubren.
- **Límite de la traza TDD en el commit**: el cuerpo del commit tiene como máximo 5-8 líneas, con una línea por fase (`RED`, `GREEN`, `REFACTOR`), con uno o dos nombres de prueba clave y el resultado observado. No se pega salida de comandos. Se agrega una línea sólo si hay un límite de entorno relevante (por ejemplo, Docker ausente). La evidencia detallada va a `apply-progress.md`. Formato:
  ```
  feat(story): <asunto corto>

  RED: <Test> falla (<motivo breve>)
  GREEN: <Test> pasa
  REFACTOR: sin cambios de comportamiento
  ```
- Un solo Pull Request (`delivery_strategy: single-pr`, decisión del usuario; no se divide ni se encadena). El exceso sobre el presupuesto se registra como `size:exception` (ver "Review Workload Forecast").
- TDD estricto (`strict_tdd: true`), runner `go test ./...`. Las pruebas de integración necesitan Docker/Testcontainers; si Docker falta, fallan (no se saltan) y se registra el límite de entorno, nunca se inventa un resultado.
- Si Testcontainers falla por contención de recursos, usar `go test -p 1 ./...` (precedente de US-10) y registrarlo en `apply-progress.md`.

## Review Workload Forecast

| Campo | Valor |
|-------|-------|
| Estimated changed lines | ~585 (producción ~190, pruebas ~370, documentación ~25) |
| 400-line budget risk | High (~46 % sobre el presupuesto) |
| Chained PRs recommended | No (decisión explícita del usuario: `single-pr`) |
| Suggested split | PR único con `size:exception`, revisable commit por commit |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: Yes (la excepción `size:exception` debe estar registrada y aceptada por el orquestador antes de la primera tarea de producción, WU 1)
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Desglose por Unidad de Trabajo:

| WU | Contenido | Líneas estimadas |
|----|-----------|------------------|
| 1 | Migración `000010` + `migration_files_test.go` + esquema/reversibilidad | ~75 |
| 2 | Aplicación (`complete_sprint_story.go` + prueba) | ~110 |
| 3 | Repositorio (`CompleteSprintStory` + integración de reglas y concurrencia) | ~190 |
| 4 | Handler HTTP (handler + prueba por tabla) | ~140 |
| 5 | Composición (`api.go`, `main.go`, `main_test.go`) | ~85 |
| 6 | Integración HTTP real (2 casos) | ~40 |
| 7 | Documentación | ~25 |
| | **Total** | **~665 en el peor caso; objetivo ~585** |

Dónde las pruebas pueden ser más delgadas sin perder cobertura de la spec:

- Probar cada regla una sola vez en la capa donde vive (ya fijado por el diseño): el mapeo de errores sólo en el handler unitario, las reglas y la concurrencia sólo en integración de repositorio; la composición HTTP real sólo con dos casos (200 y 409).
- En el handler unitario, una única tabla con un helper de ruta y de comparación; no repetir los mismos asserts de cabecera en cada caso (verificar `Content-Type` en `success` y un caso de error).
- En el caso de uso, una sola tabla de errores del puerto (7 filas) en lugar de pruebas separadas por centinela.
- Cubrir "Modificar el estado de la historia no altera la finalización" con una aserción de una línea (`repo.Update`) dentro de `TestCompleteSprintStoryPersistsCompletionForThatSprintOnly`, y "tareas pendientes" con la tarea ya creada en la prueba de reversibilidad, sin pruebas nuevas dedicadas.
- Reusar los helpers existentes del paquete `tests/integration/story/postgres` (`storyDatabase`, `applyStoryMigration`, `insertProject`, `validStory`); no duplicar helpers de `task`.
- No ampliar `TestMigrationReadinessSelectsRoutesIndependently` (el caso `v9` del test nuevo ya lo cubre).

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Migración `000010` + test canónico + esquema/reversibilidad | PR único (size:exception) | `go test ./tests/integration/migrations/... ./tests/integration/story/postgres/... -run "TestMigration\|TestCompletionSchema\|TestSprintStoryCompletionMigration"` | Testcontainers `postgres:16-alpine` (o `STORY_TEST_DATABASE_URL`), requiere Docker | Revertir `000010_*.sql`, el `"000010"` del test y las pruebas de esquema |
| 2 | Caso de uso `CompleteSprintStoryUseCase` | PR único | `go test ./tests/unit/story/application/...` | N/A (fake del puerto) | Eliminar `complete_sprint_story.go` (aplicación) y su test |
| 3 | `PostgresStoryRepository.CompleteSprintStory` | PR único | `go test ./tests/integration/story/postgres/... -run TestCompleteSprintStory` | Testcontainers, requiere Docker | Eliminar el método y la aserción de interfaz; la columna sigue sin uso |
| 4 | `CompleteSprintStoryHandler` | PR único | `go test ./tests/unit/story/transport/http/...` | N/A (`httptest` + fake) | Eliminar `complete_sprint_story_handler.go` y su test |
| 5 | Composición y gate `Completion` | PR único | `go test ./tests/unit/cmd/api/... ./internal/api/...` | N/A (fakes en `NewHTTPHandlerWithDependencies`) | Revertir los bloques de `api.go`, `main.go` y `main_test.go` |
| 6 | Integración HTTP real | PR único | `go test ./tests/integration/story/postgres/... -run TestCompleteSprintStoryHTTPEndToEnd` | Testcontainers + `api.NewHTTPHandlerWithDependencies` real | Eliminar la función de prueba |
| 7 | Documentación | PR único | N/A | N/A | Revertir `README.md`, `openspec/config.yaml`, `docs/scrum/sprint-1.md` |
| 8 | Verificación final | PR único | `go test ./...` (o `go test -p 1 ./...`) | Testcontainers (suite completa) | N/A (verificación) |

---

## Fase 0: Seguimiento paralelo — `apply-progress.md` (físico)

> Se crea antes de la WU 1 y se actualiza dentro del commit de **cada** WU (no en un commit aparte), en paralelo con la marcación de `tasks.md`. Debe existir como archivo físico en `openspec/changes/us-11-registrar-historia-completada/apply-progress.md`, no sólo en memoria.

- [x] 0.1 Crear `openspec/changes/us-11-registrar-historia-completada/apply-progress.md` (en español) con: encabezado del cambio, modo (`single-pr`, `strict_tdd`), tabla `WU | RED | GREEN | REFACTOR | Commit` (una fila por WU 1-8), sección "Commits y traza TDD" (la misma traza compacta de cada commit), sección "Defectos encontrados" (síntoma, causa, corrección, WU) y sección "Límites de entorno" (Docker, contención).
- [x] 0.2 Registrar en `apply-progress.md` la excepción `size:exception` aceptada y el pronóstico (~585 líneas vs 400) antes de la primera tarea de producción (WU 1).
- [ ] 0.3 Tras cada WU: completar su fila (RED observado, GREEN observado, REFACTOR) con evidencia detallada (nombre de prueba y resultado, duración si es de integración) y marcar las casillas de `tasks.md`; incluir ambos archivos en el commit de esa WU.

## Fase 1 (WU 1): Migración `000010` — columna `completed_at`

Requisitos de spec: "Persistir la finalización ... con migración reversible (esquema 10)".

- [x] 1.1 **RED**: En `tests/integration/migrations/migration_files_test.go` agregar `"000010"` al arreglo canónico y cambiar `len(pairs) != 9` por `!= 10` (mensaje `"canonical sequence must contain exactly ten migration pairs"`). Crear `tests/integration/story/postgres/completion_integration_test.go` (paquete `postgres_test`) con el helper `completionDatabase(t)` (`storyDatabase(t)` + `000009_create_tasks.up.sql` + `000010_add_sprint_story_completion.up.sql` vía `applyStoryMigration`), el fixture (proyecto A con S1/S2 abiertos, H1 asignada a S1 y S2, H2 sólo a S1, H3 sin asignar; proyecto B con Sprint e historia) y el helper `completedAt(t, pool, sprintID, storyID) *time.Time`; escribir `TestCompletionSchemaRequiresMigrationTen` (columna existe, nulable, `timestamp with time zone`; asociación previa queda en `NULL`) y `TestSprintStoryCompletionMigrationCanBeReversedAndReapplied` (con finalización y una tarea creada: `down` elimina la columna y conserva filas de `sprint_stories` y la tarea; `up` la reaplica en `NULL`). Ejecutar `go test ./tests/integration/migrations/... ./tests/integration/story/postgres/... -run "TestMigration\|TestCompletionSchema\|TestSprintStoryCompletionMigration"` y confirmar que **fallan** (no existen los archivos `000010`).
- [x] 1.2 **GREEN**: Crear `internal/project/infrastructure/postgres/migrations/000010_add_sprint_story_completion.up.sql` (`ALTER TABLE sprint_stories ADD COLUMN completed_at TIMESTAMPTZ NULL;`) y `.down.sql` (`ALTER TABLE sprint_stories DROP COLUMN completed_at;`). Repetir el comando y confirmar que **pasa**.
- [x] 1.3 **TRIANGULATE**: Confirmar que las pruebas existentes de esquema de `story`/`task` (`go test ./tests/integration/story/postgres/... -run TestTaskBlocks\|TestSprintStories` si existen, o la suite de migraciones completa) no cambian de resultado: la PK, las FKs de `000007` y `tasks_sprint_story_fkey` siguen intactas.
- [x] 1.4 **REFACTOR**: Revisar formato SQL y nombres contra `000001`-`000009`; reconfirmar que `000010` es el siguiente número libre (el subdirectorio `migrations/fixtures/` no cuenta). Sin cambios de comportamiento.
- [x] 1.5 **Commit** (WU 1): `feat(migrations): add sprint story completion column (000010)`. Traza compacta (máx. 5-8 líneas): `RED: TestMigrationVersions... espera 10 pares y falla sin 000010` / `GREEN: mismo test y TestCompletionSchemaRequiresMigrationTen pasan` / `REFACTOR: sin cambios de comportamiento`. Incluir `tasks.md` y `apply-progress.md` actualizados.

## Fase 2 (WU 2): Aplicación — `CompleteSprintStoryUseCase`

Requisitos de spec: contrato del caso de uso (delegación sin envolver errores; precedencia vive en el repositorio).

- [x] 2.1 **RED**: Crear `tests/unit/story/application/complete_sprint_story_test.go` (paquete `application_test`) con un fake manual de `SprintStoryCompleter` (cuenta llamadas, captura IDs, devuelve resultado/error configurable). `TestCompleteSprintStoryUseCaseExecute`: éxito con una llamada y los tres IDs del comando, resultado idéntico al del fake (incluido `CompletedAt`); tabla de errores (`ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound`, `ErrSprintClosed`, `ErrStoryNotInSprint`, `ErrStoryAlreadyCompleted`, error genérico) con `errors.Is` preservado y resultado cero. Ejecutar `go test ./tests/unit/story/application/...` y confirmar que **falla** (no existen los símbolos nuevos).
- [x] 2.2 **GREEN**: Crear `internal/story/application/complete_sprint_story.go` con `ErrStoryNotInSprint`, `ErrStoryAlreadyCompleted`, `SprintStoryCompletion`, `CompleteSprintStoryCommand`, `SprintStoryCompleter`, `CompleteSprintStoryUseCase`, `NewCompleteSprintStoryUseCase` y `Execute` (delegación sin envolver), según `design.md`. Reutilizar los centinelas existentes. Confirmar que **pasa**.
- [x] 2.3 **REFACTOR**: Revisar comentarios exportados (documentar la precedencia de errores del puerto) sin cambiar firmas ni comportamiento.
- [x] 2.4 **Commit** (WU 2): `feat(story): add CompleteSprintStoryUseCase`. Traza compacta: `RED: TestCompleteSprintStoryUseCaseExecute falla (símbolos inexistentes)` / `GREEN: pasa` / `REFACTOR: sin cambios de comportamiento`. Actualizar `tasks.md` y `apply-progress.md` en el mismo commit.

## Fase 3 (WU 3): Infraestructura — `PostgresStoryRepository.CompleteSprintStory`

> Depende de WU 1 y WU 2. Requiere Docker. Requisitos de spec: persistencia por Sprint, 404s, `story_not_in_sprint`, `sprint_closed`, no doble conteo (incluida concurrencia), no tocar `stories.status`, atomicidad.

- [ ] 3.1 **RED**: En `completion_integration_test.go` agregar: `TestCompleteSprintStoryPersistsCompletionForThatSprintOnly` (completar H1 en S1: IDs y `CompletedAt` entre `before`/`after` e igual al SQL; H1 en S2 sigue `NULL`; `stories.status` y campos sin cambios vía `ListByProject` antes/después y tras `repo.Update`; filas de `sprint_stories` sin cambios; luego H1 en S2 completa con su propio instante; `COUNT(completed_at)` y `SUM(story_points)` para el escenario de métricas); `TestCompleteSprintStoryRejectsMissingOrForeignResourcesWithoutWrites` (tabla: proyecto inexistente, Sprint inexistente, Sprint de B, historia inexistente, historia de B); `TestCompleteSprintStoryRejectsUnassignedStory` (H3 en S1, H2 en S2; sin asociación nueva); `TestCompleteSprintStoryRejectsClosedSprint` (sin completar y ya completada, con precedencia sobre `story_not_in_sprint` y `already_completed`); `TestCompleteSprintStoryNeverCountsTwice`; `TestCompleteSprintStoryConcurrentRequestsRecordOnce` (8 goroutines con canal de arranque: 1 éxito, 7 `ErrStoryAlreadyCompleted`, `completed_at` persistido igual al del éxito). Ejecutar `go test ./tests/integration/story/postgres/... -run TestCompleteSprintStory` y confirmar que **falla** (el método no existe).
- [ ] 3.2 **GREEN**: En `internal/story/infrastructure/postgres/repository.go` agregar `CompleteSprintStory` con una única transacción y la secuencia exacta del diseño (proyecto sin lock → `sprints FOR UPDATE` → `stories FOR KEY SHARE` → `is_closed` → `sprint_stories FOR UPDATE` → `completed_at` → `UPDATE ... WHERE completed_at IS NULL RETURNING completed_at`, cero filas → `ErrStoryAlreadyCompleted`; `defer Rollback`; errores de commit y de SQL sin reinterpretar) y la aserción `var _ application.SprintStoryCompleter = (*PostgresStoryRepository)(nil)`. Cuidar el tipo de escaneo (precedente de US-10: no escanear `SELECT 1` en `bool`; usar `int` o `*time.Time` para `completed_at`). Confirmar que **pasa**.
- [ ] 3.3 **TRIANGULATE**: Si la prueba de concurrencia pasara de forma trivial (p. ej. por serialización accidental), verificar que el `UPDATE ... AND completed_at IS NULL` y el lock `FOR UPDATE` son cada uno necesarios razonando sobre el orden de bloqueos, y repetir `go test -count=3 -run TestCompleteSprintStoryConcurrentRequestsRecordOnce` para descartar intermitencia.
- [ ] 3.4 **REFACTOR**: Revisar nombres, comentarios y mapeo de `pgx.ErrNoRows` sin alterar el orden de verificación ni los bloqueos probados.
- [ ] 3.5 **Commit** (WU 3): `feat(story): add transactional sprint story completion to repository`. Traza compacta: `RED: TestCompleteSprintStoryPersistsCompletionForThatSprintOnly falla (método inexistente)` / `GREEN: TestCompleteSprintStory* y concurrencia (1 éxito, 7 conflictos) pasan` / `REFACTOR: sin cambios de comportamiento` (+ una línea de entorno si Docker limitó algo). Actualizar `tasks.md` y `apply-progress.md`.

## Fase 4 (WU 4): Transporte HTTP — `CompleteSprintStoryHandler`

> Puede ejecutarse en paralelo con WU 3 (ambas dependen sólo de WU 2). Requisitos de spec: 200, cuerpo ignorado, 422 acumulado, 404/409/500 y mapeo de códigos.

- [ ] 4.1 **RED**: Crear `tests/unit/story/transport/http/complete_sprint_story_handler_test.go` (paquete `transporthttp_test`) con tabla (`name, path IDs, body, completerErr, wantStatus, wantError, wantFields, wantCalls`) sobre el caso de uso real con fake del puerto y un mux real: `success` (200, tres IDs, `completed_at` en UTC aunque el fake devuelva otra zona, `Content-Type: application/json`, 1 llamada); `uppercase UUIDs are canonicalized`; `body is ignored` (`{"x":1}` y `not json` → 200); `invalid project_id`/`sprint_id`/`story_id` (422, 0 llamadas); `all route IDs invalid` (tres claves); un caso por centinela (404 x3, 409 x3, mapeo exacto de la tabla del diseño); `unexpected` (`errors.New("secret db detail")` → 500 `internal_error` sin `"secret"`); `wrong method` (405). Ejecutar `go test ./tests/unit/story/transport/http/...` y confirmar que **falla**.
- [ ] 4.2 **GREEN**: Crear `internal/story/transport/http/complete_sprint_story_handler.go` con `CompleteSprintStoryHandler`, `NewCompleteSprintStoryHandler`, `sprintStoryCompletionResponse` y el mapeo de errores (reutilizando `writeJSON` y `errorResponse` del paquete); orden: método → tres UUID acumulados → caso de uso → mapeo; no leer `request.Body`; `completed_at` con `.UTC()`. Confirmar que **pasa**.
- [ ] 4.3 **REFACTOR**: Revisar mensajes y nombres contra `internal/story/transport/http` y `internal/task/transport/http` sin cambiar ningún status, código o mensaje probado.
- [ ] 4.4 **Commit** (WU 4): `feat(story): add CompleteSprintStoryHandler with error mapping`. Traza compacta: `RED: TestCompleteSprintStoryHandler falla (handler inexistente)` / `GREEN: pasa (todas las filas de la tabla)` / `REFACTOR: sin cambios de comportamiento`. Actualizar `tasks.md` y `apply-progress.md`.

## Fase 5 (WU 5): Composición — gate `Completion`

> Depende de WU 2 y WU 4 (registra el handler real; los tests usan fakes).

- [ ] 5.1 **RED**: En `tests/unit/cmd/api/main_test.go` agregar `fakeSprintStoryCompleter` (cuenta llamadas, devuelve una finalización fija) y `TestCompletionRouteRequiresCleanVersionTenAndExplicitDependency` (casos `v9` con `readiness.Tasks == true` y `Completion == false`, `v10`, `future 12`, `nil dependency`, `dirty v10`, `lookup error`, `missing migration table`; `POST .../completion` responde 200 con 1 llamada sólo cuando corresponde, 404 y 0 llamadas en el resto) y `TestCompletionRouteIsIndependentFromStories` (`Completion` con `Stories == nil` → 200; `POST /projects/{id}/stories` sigue en 404). Ejecutar `go test ./tests/unit/cmd/api/... ./internal/api/...` y confirmar que **falla** (no existen `CompletionDependencies`, `MigrationReadiness.Completion` ni la ruta).
- [ ] 5.2 **GREEN**: En `internal/api/api.go` agregar `CompletionDependencies{Completer}`, `HTTPDependencies.Completion`, `MigrationReadiness.Completion`, `readiness.Completion = version >= 10` y el registro condicional de `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion` independiente del bloque de `Stories`. En `cmd/api/main.go` agregar el bloque de gate con los logs `story completion available/unavailable ... migration 000010 ...`, sin leer ni escribir `dependencies.Stories`. Confirmar que **pasa**.
- [ ] 5.3 **REFACTOR**: Actualizar el comentario de versiones de `main.go` (líneas ~36-38) para mencionar la versión 10, sin alterar comportamientos de gates ya probados.
- [ ] 5.4 **Commit** (WU 5): `feat(api): gate sprint story completion route behind migration 000010`. Traza compacta: `RED: TestCompletionRouteRequiresCleanVersionTenAndExplicitDependency falla (símbolos inexistentes)` / `GREEN: pasa junto con TestCompletionRouteIsIndependentFromStories` / `REFACTOR: sin cambios de comportamiento`. Actualizar `tasks.md` y `apply-progress.md`.

## Fase 6 (WU 6): Integración HTTP real

> Depende de WU 1, 3 y 5. Requiere Docker.

- [ ] 6.1 **RED/verificación**: En `completion_integration_test.go` agregar `TestCompleteSprintStoryHTTPEndToEnd` componiendo `api.NewHTTPHandlerWithDependencies(projectpostgres..., api.NewProjectID, api.HTTPDependencies{Completion: &api.CompletionDependencies{Completer: storypostgres.NewPostgresStoryRepository(pool)}})`: primer `POST` → 200 con `completed_at` igual al persistido; segundo `POST` → 409 `story_already_completed`. Ejecutar `go test ./tests/integration/story/postgres/... -run TestCompleteSprintStoryHTTPEndToEnd` y registrar el resultado real (puede pasar de inmediato porque no hay producción nueva; documentarlo como "GREEN directo", sin inventar un RED).
- [ ] 6.2 **Commit** (WU 6): `test(story): add end-to-end HTTP coverage for sprint story completion`. Traza compacta: `RED: n/a, sin producción nueva (verificación de composición)` / `GREEN: TestCompleteSprintStoryHTTPEndToEnd pasa (200 y 409)`. Actualizar `tasks.md` y `apply-progress.md`.

## Fase 7 (WU 7): Documentación

- [ ] 7.1 `README.md`: agregar la sección de US-11 (ruta `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion`, sin cuerpo, respuestas 200/404/409/422/500, fuente de verdad `sprint_stories.completed_at`, divergencia deliberada con `stories.status`, Sprint cerrado rechazado) y actualizar "Migraciones y disponibilidad" (`000001`-`000010`, gate `>= 10`); documentar que `down` de `000010` elimina los registros de finalización.
- [ ] 7.2 `openspec/config.yaml`: agregar la nota `"Implementada: US-11 (registrar historia completada en el Sprint)."` en el mismo formato de las notas existentes (no corregir la nota histórica de `000003`).
- [ ] 7.3 `docs/scrum/sprint-1.md`: agregar una nota fechada de que el registro de finalización está implementado en código, sin reescribir la instantánea del tablero.
- [ ] 7.4 **Commit** (WU 7): `docs: document sprint story completion route and migration 000010`. Sin traza TDD (documentación; indicarlo en una línea). Actualizar `tasks.md` y `apply-progress.md`.

## Fase 8 (WU 8): Verificación final

- [ ] 8.1 Ejecutar `go test ./...` con Docker disponible y confirmar que todo pasa; si hay contención de recursos de Testcontainers, repetir con `go test -p 1 ./...` y registrar ambas corridas, sin saltar ni marcar `-short`.
- [ ] 8.2 Revisar los 8 criterios de éxito de `proposal.md` y los escenarios de `specs/historia/spec.md` uno por uno, con el nombre de la prueba y el resultado observado, en `apply-progress.md`.
- [ ] 8.3 Reconfirmar, antes del PR, que `000010` sigue siendo el siguiente número libre (read-only).
- [ ] 8.4 Si la verificación detecta una corrección, aplicarla con su ciclo RED -> GREEN -> REFACTOR y un commit propio con traza compacta, y registrar el defecto en `apply-progress.md`; si no, no se agrega commit.
- [ ] 8.5 Dejar `apply-progress.md` completo (tabla final, commits, defectos, límites de entorno) y todas las casillas de `tasks.md` marcadas con evidencia. Commit final sólo si quedaron cambios de seguimiento sin commitear: `docs(sdd): finalize US-11 apply progress`.
- [ ] 8.6 **Archivo (fuera de apply)**: `sdd-verify` y luego `sdd-archive` los ejecuta el orquestador; el archivado mueve el cambio a `openspec/changes/archive/` y consolida la delta en `openspec/specs/historia/`. `sdd-apply` no archiva.

---

## Pronóstico de revisión (detalle final)

El total estimado (~585 líneas) supera el presupuesto de 400 en ~46 %. El código de producción (~190 líneas) entra en el presupuesto; el exceso proviene de pruebas (~370) y documentación (~25). Decisión ya tomada: `single-pr` con `size:exception`, revisable commit por commit (una WU = un commit). `sdd-apply` no debe iniciar la WU 1 sin que la excepción esté registrada en `apply-progress.md` (tarea 0.2).
