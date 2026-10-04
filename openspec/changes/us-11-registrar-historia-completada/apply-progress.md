# Apply progress — US-11 registrar una historia del Sprint como completada

## Resumen

Rama `feat/us11-registrar-historia-completada`. Modo: `single-pr`, `strict_tdd: true`, runner `go test ./...`. Corrida 1 de 2: Fase 0 y WU 1-4. Corrida 2 de 2: WU 5-8.

## Excepción de tamaño (`size:exception`)

Pronóstico: ~585 líneas cambiadas (producción ~190, pruebas ~370, documentación ~25) contra un presupuesto de 400 (~46 % sobre el límite). El usuario exige un único Pull Request (`delivery_strategy: single-pr`), por lo que el orquestador acepta `size:exception`. El PR es revisable commit por commit (una WU = un commit). Registrada antes de la WU 1 (tarea 0.2).

## Ciclo TDD por unidad de trabajo

| WU | RED | GREEN | REFACTOR | Commit |
|---|---|---|---|---|
| 1 | `TestMigrationVersionsAreUnique` falla ("exactly ten migration pairs", "missing canonical migration version 000010"); `TestCompletionSchemaRequiresMigrationTen` y `TestSprintStoryCompletionMigrationCanBeReversedAndReapplied` fallan (no existe `000010_*.up.sql`). | Se crean `000010_add_sprint_story_completion.{up,down}.sql`; las tres pruebas pasan (migraciones 1.2 s; story/postgres 14.2 s). Triangulación 1.3: suites completas de `story/postgres` (376 s), `task/postgres` (80 s) y `migrations` en verde con `-p 1`: PK, FKs de `000007` y `tasks_sprint_story_fkey` intactas. | Formato y nombres revisados contra `000001`-`000009`; `000010` es el siguiente número libre. Sin cambios de comportamiento. | `5a3934d` `feat(migrations): add sprint story completion column (000010)` |
| 2 | `TestCompleteSprintStoryUseCaseExecute` no compila: `SprintStoryCompletion`, `CompleteSprintStoryCommand`, `NewCompleteSprintStoryUseCase`, `ErrStoryNotInSprint`, `ErrStoryAlreadyCompleted` indefinidos. | Se crea `complete_sprint_story.go`; pasan las 8 filas (éxito con una llamada e IDs del comando + 7 errores del puerto sin envolver, resultado cero). Triangulación: éxito y 7 errores distintos. | Comentarios exportados con la precedencia de errores del puerto; `go vet` limpio. Sin cambios de comportamiento. | `620e303` `feat(story): add CompleteSprintStoryUseCase` |
| 3 | Las 6 pruebas `TestCompleteSprintStory*` no compilan: `repo.CompleteSprintStory` indefinido (`go vet`). | Se agrega `CompleteSprintStory` con la secuencia exacta del diseño; pasan persistencia por Sprint (8.6 s), recursos ausentes/ajenos (5 filas), historia no asignada (2 filas), Sprint cerrado (precedencia sobre `story_not_in_sprint` y `already_completed`), no doble conteo y concurrencia de 8 goroutines (1 éxito, 7 conflictos): 32 s en total. Triangulación 3.3: `-count=3` de la concurrencia en verde (17 s); el `FOR UPDATE` del Sprint ya serializa a los competidores (como en US-09), así que el `AND completed_at IS NULL` queda como defensa en profundidad documentada en el código. | Nombres, comentarios y mapeo de `pgx.ErrNoRows` revisados sin alterar orden de verificación ni bloqueos. Escaneo con `int` y `*time.Time` (lección de US-10). | `2b0a49c` `feat(story): add transactional sprint story completion to repository` |
| 4 | `TestCompleteSprintStoryHandler` no compila: `storyhttp.NewCompleteSprintStoryHandler` indefinido. | Se crea `complete_sprint_story_handler.go`; pasan las 16 filas: éxito (200, `completed_at` en UTC desde una zona UTC-3, `Content-Type`), UUID en mayúsculas canonicalizados, cuerpo ignorado (`{"x":1}` y `not json`), 422 por cada ID y acumulado de los tres, 404 x3, 409 x3, 500 sin filtrar `secret`, 405. Triangulación: tabla con casos de éxito, validación y cada centinela. | Mensajes y nombres alineados con `story/transport/http` y `task/transport/http`; ningún status, código o mensaje probado cambió. | `beaf023` `feat(story): add CompleteSprintStoryHandler with error mapping` |
| 5 | `TestCompletionRouteRequiresCleanVersionTenAndExplicitDependency` y `TestCompletionRouteIsIndependentFromStories` no compilan: `MigrationReadiness.Completion`, `HTTPDependencies.Completion` y `api.CompletionDependencies` indefinidos. | Se agregan `CompletionDependencies`, `HTTPDependencies.Completion`, `readiness.Completion = version >= 10` y el registro condicional de la ruta en `api.go`, más el bloque de gate con logs en `main.go`; pasan las 7 filas (v9 con `Tasks` listo, v10, future 12, dependencia nula, sucio, error de consulta, tabla ausente) y la independencia respecto de `Stories` (0.9 s). Riesgo abierto del diseño: `TestAPIStartupRoutesFollowMigrationState` (caso "future version", fila 10 sobre esquema v8) sigue en verde (105 s, `-p 1`). | Comentario de versiones de `main.go` actualizado (versión 10); sin cambios de comportamiento. | `7a35ff5` `feat(api): gate sprint story completion route behind migration 000010` |
| 6 | N/A: no hay producción nueva (verificación de composición), se evita inventar un RED. | GREEN directo: `TestCompleteSprintStoryHTTPEndToEnd` pasa al primer intento (7.0 s): primer `POST` 200 con `completed_at` igual al persistido; segundo `POST` 409 `story_already_completed`. | Sin cambios. | ver `git log` (`test(story): add end-to-end HTTP coverage for sprint story completion`) |
| 7-8 | pendiente | pendiente | pendiente | pendiente |

### Evidencia de unidad de trabajo

| WU | Prueba enfocada | Harness real | Límite de reversión |
|---|---|---|---|
| 1 | `go test ./tests/integration/migrations/... ./tests/integration/story/postgres/... -run "TestMigration\|TestCompletionSchema\|TestSprintStoryCompletionMigration"` → ok | Testcontainers `postgres:16-alpine` (Docker 29.8.1) | Revertir `000010_*.sql`, el `"000010"` del test y las pruebas de esquema |
| 2 | `go test ./tests/unit/story/application/... -run TestCompleteSprintStoryUseCaseExecute` → ok (8 subpruebas) | N/A: fake del puerto, sin frontera de runtime | Eliminar `complete_sprint_story.go` (aplicación) y su test |
| 3 | `go test ./tests/integration/story/postgres/... -run TestCompleteSprintStory` → ok (32.0 s) | Testcontainers `postgres:16-alpine` | Eliminar el método y la aserción de interfaz; la columna sigue sin uso |
| 4 | `go test ./tests/unit/story/transport/http/... -run TestCompleteSprintStoryHandler` → ok (16 subpruebas) | N/A: `httptest` con mux real y fake del puerto, sin frontera de runtime | Eliminar `complete_sprint_story_handler.go` y su test |
| 5 | `go test ./tests/unit/cmd/api/... ./internal/api/...` → ok (0.9 s) | N/A: fakes en `NewHTTPHandlerWithDependencies`, sin frontera de runtime | Revertir los bloques de `api.go`, `main.go` y `main_test.go` |
| 6 | `go test -p 1 ./tests/integration/story/postgres/... -run TestCompleteSprintStoryHTTPEndToEnd` → ok (7.0 s) | Testcontainers `postgres:16-alpine` + `api.NewHTTPHandlerWithDependencies` real | Eliminar la función de prueba |

## Commits y traza TDD

Los hashes se registran en el commit de la WU siguiente (un commit no puede contener su propio hash).

- WU 1: `feat(migrations): add sprint story completion column (000010)` — RED: `TestMigrationVersionsAreUnique` falla sin `000010`; GREEN: pasa junto con `TestCompletionSchemaRequiresMigrationTen`; REFACTOR: sin cambios de comportamiento.
- WU 2: `feat(story): add CompleteSprintStoryUseCase` — RED: `TestCompleteSprintStoryUseCaseExecute` no compila (símbolos inexistentes); GREEN: pasa (8 subpruebas); REFACTOR: sin cambios de comportamiento.
- WU 3: `feat(story): add transactional sprint story completion to repository` — RED: `TestCompleteSprintStory*` no compila (método inexistente); GREEN: persistencia, reglas y concurrencia (1 éxito, 7 conflictos) pasan; REFACTOR: sin cambios de comportamiento.
- WU 4: `feat(story): add CompleteSprintStoryHandler with error mapping` — RED: `TestCompleteSprintStoryHandler` no compila (handler inexistente); GREEN: pasa (16 subpruebas); REFACTOR: sin cambios de comportamiento.
- WU 5: `feat(api): gate sprint story completion route behind migration 000010` — RED: `TestCompletionRouteRequiresCleanVersionTenAndExplicitDependency` no compila (símbolos inexistentes); GREEN: pasa junto con `TestCompletionRouteIsIndependentFromStories`; REFACTOR: sin cambios de comportamiento.
- WU 6: `test(story): add end-to-end HTTP coverage for sprint story completion` — RED: n/a, sin producción nueva; GREEN: `TestCompleteSprintStoryHTTPEndToEnd` pasa (200 y 409).

## Defectos encontrados

Ninguno hasta ahora.

## Límites de entorno

Docker disponible (servidor 29.8.1). Las pruebas de integración usan Testcontainers `postgres:16-alpine` reales. Si hay contención de recursos se usa `go test -p 1`.
