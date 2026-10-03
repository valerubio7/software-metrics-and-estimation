# Tareas: Descomponer una historia del Sprint en tareas (US-10)

## Convención de commits

- Cada tarea (o grupo chico de tareas estrechamente relacionadas, marcado como una sola Unidad de Trabajo) cierra con **un único commit** en la rama de la feature, usando Conventional Commits.
- El mensaje de commit es corto (línea de asunto descriptiva). La evidencia de TDD se registra como una traza compacta en el cuerpo del commit, con el formato:
  `TDD: RED (<Test> falla) -> GREEN (<Test> pasa) -> REFACTOR (sin cambios de comportamiento)`
- No se mezclan tests y producción en una sola tarea indivisible: cada ciclo RED → GREEN → REFACTOR queda explícito en tareas separadas dentro de la misma Unidad de Trabajo, pero el commit final las integra como una sola entrega revisable.
- El cambio completo se entrega en **un único Pull Request** (`delivery_strategy: single-pr`, decisión ya tomada por el usuario). No se diseñan PRs encadenados ni se pide dividir el trabajo; el exceso sobre el presupuesto de revisión se documenta en el "Pronóstico de revisión" al final de este archivo con `size:exception`.

## Review Workload Forecast

| Campo | Valor |
|-------|-------|
| Estimated changed lines | ~1600–1700 (ver desglose por grupo al final del documento) |
| 400-line budget risk | High |
| Chained PRs recommended | No (decisión explícita del usuario: `single-pr`) |
| Suggested split | PR único con `size:exception` |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Migración `000009_create_tasks` + ajuste del test canónico de migraciones | PR único (size:exception) | `go test ./tests/integration/migrations/...` | N/A — no hay servidor HTTP involucrado en esta unidad; la verificación es sobre archivos y, en Fase 9, sobre el esquema aplicado | Revertir `000009_create_tasks.{up,down}.sql` y el `"000009"` agregado en `migration_files_test.go` |
| 2 | Dominio `Task` (`internal/task/domain/task.go`) | PR único (size:exception) | `go test ./tests/unit/task/domain/...` | N/A — módulo de dominio sin dependencias externas | Eliminar `internal/task/domain/` completo; nada más lo importa todavía |
| 3 | Aplicación `CreateTasksUseCase` (`internal/task/application/create_tasks.go`) | PR único (size:exception) | `go test ./tests/unit/task/application/...` | N/A — caso de uso probado con fake del puerto, sin servidor real | Eliminar `internal/task/application/`; depende solo de `internal/task/domain` (Unidad 2) |
| 4 | Infraestructura `PostgresTaskRepository` (`internal/task/infrastructure/postgres/repository.go`) | PR único (size:exception) | `go test ./tests/integration/task/postgres/... -run TestTaskSchemaRequiresMigrationNine\|TestCreateTasks\|TestTasksTable\|TestTaskBlocks\|TestTasksMigration` | Testcontainers `postgres:16-alpine` (o `TASK_TEST_DATABASE_URL`) — requiere Docker | Eliminar `internal/task/infrastructure/postgres/`; la tabla `tasks` sigue existiendo (migración de la Unidad 1) sin código que la use |
| 5 | Transporte HTTP `CreateTasksHandler` (`internal/task/transport/http/handler.go`) | PR único (size:exception) | `go test ./tests/unit/task/transport/http/...` | N/A — probado con `httptest` y fake del puerto, sin servidor real levantado | Eliminar `internal/task/transport/http/`; depende solo de `internal/task/application` (Unidad 3) |
| 6 | Composición (`internal/api/api.go`, `cmd/api/main.go`) | PR único (size:exception) | `go test ./tests/unit/cmd/api/... ./internal/api/...` | N/A — el gate se prueba con fakes inyectados en `api.NewHTTPHandlerWithDependencies`, sin arrancar el binario `cmd/api` | Revertir los bloques agregados en `api.go` y `main.go`; sin ellos la ruta nunca se registra (equivalente a no tener la Unidad 6) |
| 7 | Integración HTTP real (`tests/integration/task/postgres/http_integration_test.go`) | PR único (size:exception) | `go test ./tests/integration/task/postgres/... -run TestCreateTasksHTTP` (o el nombre que agrupe los casos HTTP reales) | Testcontainers `postgres:16-alpine` + `api.NewHTTPHandlerWithDependencies` real — requiere Docker | Eliminar el archivo de test; no toca producción |
| 8 | Documentación (`README.md`) | PR único (size:exception) | N/A — cambio de documentación, no ejecuta tests | N/A — no aplica | Revertir el bloque agregado en `README.md` |
| 9 | Verificación final de todo el cambio | PR único (size:exception) | `go test ./...` | Testcontainers (suite completa) — requiere Docker | N/A — tarea de verificación, no agrega código propio |

---

## Fase 1: Migración de esquema — tabla `tasks` (000009)

- [x] 1.1 **RED**: Modificar `tests/integration/migrations/migration_files_test.go` agregando `"000009"` al arreglo de versiones canónicas esperadas y cambiando la aserción `len(pairs) != 8` por `!= 9` (mensaje `"...exactly nine migration pairs"`). Ejecutar `go test ./tests/integration/migrations/...` y confirmar que **falla** (los archivos `000009_create_tasks.{up,down}.sql` todavía no existen).
- [x] 1.2 **GREEN**: Crear `internal/project/infrastructure/postgres/migrations/000009_create_tasks.up.sql` con la tabla `tasks` (`id`, `project_id`, `sprint_id`, `story_id`, `title`, `estimated_hours NUMERIC(7,2)` con `CHECK (> 0)`, `seq BIGINT GENERATED ALWAYS AS IDENTITY`, `created_at TIMESTAMPTZ DEFAULT now()`), las tres FKs compuestas `tasks_sprint_story_fkey`, `tasks_sprint_project_fkey`, `tasks_story_project_fkey` (todas `ON DELETE RESTRICT`) y el índice `tasks_sprint_story_idx`, exactamente como define `design.md` (sección "Migración — `000009_create_tasks.up.sql`"). Crear `internal/project/infrastructure/postgres/migrations/000009_create_tasks.down.sql` con `DROP TABLE tasks;`. Volver a ejecutar `go test ./tests/integration/migrations/...` y confirmar que **pasa**.
- [x] 1.3 **REFACTOR**: Revisar nombres de constraints, formato SQL y comentarios contra la convención de las migraciones `000001`–`000008` existentes, sin cambiar el esquema resultante ni el comportamiento ya verificado.
- [x] 1.4 **Commit** (Unidad de Trabajo 1): `feat(migrations): add tasks table migration 000009`. Traza TDD: `RED (TestMigrationFiles... espera 9 pares, falla sin 000009) -> GREEN (mismo test pasa con 000009 creada) -> REFACTOR (sin cambios de comportamiento)`.

## Fase 2: Dominio `Task`

- [x] 2.1 **RED**: Crear `tests/unit/task/domain/task_test.go` (paquete `domain_test`) con una función de prueba por tabla (`TestNewTask`, subtests vía `t.Run`) cubriendo: tarea válida que conserva IDs y título sin recortar (`"  Diseñar  "`) con estimación `nil`; estimaciones válidas `0.01`, `1.5`, `4.25`, `99999.99` verificando que el puntero devuelto es una copia (mutar el original no cambia la tarea); título inválido `""`, `"   "`, `"\t\n"` → `Fields["title"] == "is required"`; estimación inválida `0`, `-1`, `100000`, `99999.991`, `1.234`, `NaN`, `+Inf` → `Fields["estimated_hours"]` con el mensaje exacto `"must be greater than 0, at most 99999.99 and have at most 2 decimals"`; ambos campos inválidos a la vez → dos claves en `Fields`, `errors.As` contra `*domain.ValidationError` funciona, `Error() == "task validation failed"`. Ejecutar `go test ./tests/unit/task/domain/...` y confirmar que **falla** (el paquete `internal/task/domain` no existe).
- [x] 2.2 **GREEN**: Crear `internal/task/domain/task.go` con el tipo `Task` (`ID`, `ProjectID`, `SprintID`, `StoryID`, `Title`, `EstimatedHours *float64`), `ValidationError` (`Fields map[string]string`, método `Error()`), las constantes privadas `maxEstimatedHours = 99999.99` y `maxEstimatedHoursDecimal = 2`, las funciones privadas `validateEstimatedHours` y `decimalPlaces`, y `NewTask(id, projectID, sprintID, storyID, title string, estimatedHours *float64) (Task, error)` exactamente según la interfaz de `design.md` (sección "Dominio — `internal/task/domain/task.go`"). Ejecutar el mismo test y confirmar que **pasa**.
- [x] 2.3 **REFACTOR**: Revisar duplicación interna y comentarios exportados sin alterar los mensajes de error ni la semántica validada por el test.
- [x] 2.4 **Commit** (Unidad de Trabajo 2): `feat(task): add Task domain validation`. Traza TDD: `RED (TestNewTask falla, paquete inexistente) -> GREEN (TestNewTask pasa) -> REFACTOR (sin cambios de comportamiento)`.

## Fase 3: Aplicación `CreateTasksUseCase`

- [x] 3.1 **RED**: Crear `tests/unit/task/application/create_tasks_test.go` (paquete `application_test`) con un `fakeTaskRepository` (cuenta llamadas, captura `projectID`/`sprintID`/`storyID`/`tasks`, devuelve un error configurable) y un generador de IDs contador. Escribir `TestCreateTasksUseCaseExecute` (tabla) cubriendo: éxito con 3 tareas (una llamada al repositorio, IDs asignados en orden, título y estimación preservados, resultado igual a lo enviado); lote `nil` y `[]` → `ValidationError{"tasks": "must contain at least one task"}` con 0 llamadas al repositorio y al generador; un elemento inválido entre válidos → clave `tasks[1].title`, 0 llamadas; varios elementos inválidos → todas las claves (`tasks[0].title`, `tasks[2].estimated_hours`) sin cortar en el primero; errores del repositorio `ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound`, `ErrStoryNotInSprint` y un error genérico → `errors.Is` se preserva y el resultado es `nil`; aislamiento — mutar el `*float64` del comando después de `Execute` no altera las tareas devueltas. Ejecutar `go test ./tests/unit/task/application/...` y confirmar que **falla** (el paquete `internal/task/application` no existe).
- [x] 3.2 **GREEN**: Crear `internal/task/application/create_tasks.go` con los errores centinela `ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound`, `ErrStoryNotInSprint`; los tipos `TaskInput`, `CreateTasksCommand`; la interfaz `TaskRepository` con `CreateForSprintStory(ctx, projectID, sprintID, storyID string, tasks []domain.Task) error`; el tipo `IDGenerator`; `CreateTasksUseCase` y `NewCreateTasksUseCase`; y `Execute` siguiendo el algoritmo exacto de `design.md` (validar lote completo acumulando errores indexados `tasks[i].campo`, sin llamar al generador ni al repositorio si hay errores de validación o lote vacío; asignar IDs en orden; delegar en el repositorio; devolver el error sin envolver). Ejecutar el mismo test y confirmar que **pasa**.
- [x] 3.3 **REFACTOR**: Revisar nombres y comentarios exportados sin alterar la firma de `TaskRepository` ni el comportamiento de `Execute`.
- [x] 3.4 **Commit** (Unidad de Trabajo 3): `feat(task): add CreateTasksUseCase with batch validation`. Traza TDD: `RED (TestCreateTasksUseCaseExecute falla, paquete inexistente) -> GREEN (mismo test pasa) -> REFACTOR (sin cambios de comportamiento)`.

## Fase 4: Infraestructura PostgreSQL — `PostgresTaskRepository`

> Puede ejecutarse en paralelo con la Fase 5 (ambas dependen solo de las Fases 2 y 3, no entre sí). Requiere Docker para `go test` sin `-short`.

> **BLOQUEADO (sdd-apply)**: Docker Desktop está instalado en este entorno pero el motor no está corriendo (`docker info` → `rootless Docker is not supported on Windows, failed to create Docker provider`). Se crearon `internal/task/infrastructure/postgres/repository.go` y `tests/integration/task/postgres/repository_integration_test.go` siguiendo exactamente la interfaz y el algoritmo de `design.md`, y se confirmó que compilan (`go vet ./internal/task/... ./tests/integration/task/...` sin errores). **No se pudo confirmar GREEN por ejecución real** contra PostgreSQL: el intento de `go test ./tests/integration/task/postgres/...` falla por un error de infraestructura (arranque de Testcontainers), no por lógica de negocio. Las tareas 4.1–4.4 quedan **sin marcar** hasta que se pueda re-ejecutar la suite con Docker disponible (o `TASK_TEST_DATABASE_URL` apuntando a un PostgreSQL local).

- [ ] 4.1 **RED**: Crear los helpers de prueba propios del módulo (no compartidos con `story`): `taskDatabase(t)` (Testcontainers o `TASK_TEST_DATABASE_URL` vía `testpostgres.OpenIsolated(t, dsn, "TASK_TEST_DATABASE_URL", "task_test")`), `applyTaskMigration`, `taskModuleRoot`, `insertProject`, `insertSprint`, `insertStory` (SQL directo), `assignStory` (`INSERT` directo en `sprint_stories`), `taskCount`, `assertDatabaseError` (código Postgres + nombre de restricción) y el fixture `taskFixture(t)` (proyecto A con Sprint S1 y S2, historia H1 asignada a S1, historia H2 sin asignar; proyecto B con Sprint e historia propios). Crear `tests/integration/task/postgres/repository_integration_test.go` (paquete `postgres_test`) con: `TestTaskSchemaRequiresMigrationNine`; `TestCreateTasksPersistsBatchLinkedToStorySprintAndProject`; `TestCreateTasksRejectsMissingOrForeignResourcesWithoutWrites` (tabla: proyecto inexistente, Sprint inexistente, Sprint de otro proyecto, historia inexistente, historia de otro proyecto); `TestCreateTasksRejectsStoryNotAssignedToSprint`; `TestCreateTasksAllowsClosedSprint`; `TestCreateTasksRollsBackWhenAnInsertFails`; `TestTasksTableEnforcesConstraints`; `TestTaskBlocksUnassigningItsStory`; `TestTasksMigrationCanBeReversedAndReapplied`, exactamente con las aserciones descritas en `design.md` (sección "`tests/integration/task/postgres/repository_integration_test.go`"). Ejecutar `go test ./tests/integration/task/postgres/...` (con Docker disponible) y confirmar que **falla** (el paquete `internal/task/infrastructure/postgres` no existe).
- [ ] 4.2 **GREEN**: Crear `internal/task/infrastructure/postgres/repository.go` con `PostgresTaskRepository`, `NewPostgresTaskRepository(pool)`, y `CreateForSprintStory(ctx, projectID, sprintID, storyID string, tasks []domain.Task) error` implementando, dentro de una sola transacción (`tx.Begin` + `defer Rollback`), la secuencia SQL exacta del diseño: existencia de proyecto → `ErrProjectNotFound`; `sprints` por `(id, project_id)` con `FOR KEY SHARE` → `ErrSprintNotFound`; `stories` por `(id, project_id)` con `FOR KEY SHARE` → `ErrStoryNotFound`; `sprint_stories` por `(sprint_id, story_id)` con `FOR KEY SHARE` → `ErrStoryNotInSprint`; `INSERT INTO tasks` una sentencia por tarea en orden (con mapeo defensivo de `23503` sobre `tasks_sprint_story_fkey` a `ErrStoryNotInSprint`); `COMMIT`. Agregar la aserción `var _ application.TaskRepository = (*PostgresTaskRepository)(nil)`. Ejecutar el mismo comando y confirmar que **pasa**.
- [ ] 4.3 **REFACTOR**: Revisar nombres de parámetros, orden de las consultas y comentarios sin alterar el bloqueo (`FOR KEY SHARE`), el orden de verificación ni el mapeo de errores ya probado.
- [ ] 4.4 **Commit** (Unidad de Trabajo 4): `feat(task): add PostgresTaskRepository with sprint-story membership check`. Traza TDD: `RED (suite de integración de task/postgres falla, paquete inexistente) -> GREEN (misma suite pasa) -> REFACTOR (sin cambios de comportamiento)`.

## Fase 5: Transporte HTTP — `CreateTasksHandler`

> Puede ejecutarse en paralelo con la Fase 4 (ambas dependen solo de las Fases 2 y 3, no entre sí).

- [x] 5.1 **RED**: Crear `tests/unit/task/transport/http/handler_test.go` (paquete `transporthttp_test`) con un fake de `application.TaskRepository` inyectado en un `CreateTasksUseCase` real (patrón de `assign_stories_test.go`) y una tabla de casos (`name, body, path overrides, repoErr, wantStatus, wantError, wantFields, wantCalls`) cubriendo exactamente los casos de `design.md`: `success`, `success without estimate key`, `invalid project route`, `invalid sprint route`, `invalid story route`, `empty tasks`, `missing tasks`, `null tasks`, `blank title`, `invalid estimate`, `multiple invalid elements`, `unknown root field`, `unknown task field`, `trailing JSON`, `null body`, `array body`, `malformed`, `null task element`, `estimate as string`, `project not found`, `sprint not found`, `story not found`, `story not in sprint`, `unexpected` (error genérico, verificar que el cuerpo 500 no incluye el detalle), `wrong method`, y la verificación de `Content-Type: application/json` en todas las respuestas. Ejecutar `go test ./tests/unit/task/transport/http/...` y confirmar que **falla** (el paquete `internal/task/transport/http` no existe).
- [x] 5.2 **GREEN**: Crear `internal/task/transport/http/handler.go` (paquete `transporthttp`) con `CreateTasksHandler`, `NewCreateTasksHandler`, los DTOs `createTasksRequest`, `taskRequest`, `taskResponse`, `createTasksResponse`, `errorResponse`; `decodeTaskObject` (réplica de la técnica de `decodeStoryObject`: `json.RawMessage`, exige objeto único, `DisallowUnknownFields`, rechaza elementos `null` en `tasks`); el orden de evaluación en `ServeHTTP` (método → cuerpo → `project_id` → `sprint_id` → `story_id` → caso de uso); y el mapeo de errores exacto de la tabla de `design.md` (405/400/422/404×3/409/500), detectando `*domain.ValidationError` con `errors.As` y los centinelas con `errors.Is`. Ejecutar el mismo comando y confirmar que **pasa**.
- [x] 5.3 **REFACTOR**: Revisar nombres de DTOs y mensajes de error contra la convención de `internal/story/transport/http/handler.go` sin alterar ningún status, código o mensaje ya probado.
- [x] 5.4 **Commit** (Unidad de Trabajo 5): `feat(task): add CreateTasksHandler with strict decoding and error mapping`. Traza TDD: `RED (TestCreateTasksHandler falla, paquete inexistente) -> GREEN (mismo test pasa) -> REFACTOR (sin cambios de comportamiento)`.

> **DESVIACIÓN DE DISEÑO (sdd-apply)**: `design.md` (tabla "Mapeo de errores") pedía `tasks` ausente/`null`/`[]` → `422 validation_failed {"tasks":"must contain at least one task"}` vía el caso de uso. Pero `specs/tarea/spec.md` (Requirement "Rechazar solicitudes malformadas sin persistir tareas", escenario "Lote vacío") exige `400 invalid_request` para ese mismo caso, agrupado con los demás problemas estructurales del cuerpo. Como la especificación es el criterio de aceptación, se implementó el chequeo estructural `len(input.Tasks) == 0 → 400` en el handler HTTP, **antes** de llamar al caso de uso (0 llamadas, consistente con `tasks.md`). El chequeo `422` de `CreateTasksUseCase.Execute` (Fase 3) permanece intacto como defensa en profundidad para cualquier llamador que no pase por HTTP — ambos niveles se probaron explícitamente y conviven sin conflicto en la práctica (el handler nunca deja pasar un lote vacío al caso de uso).

## Fase 6: Composición — gate de esquema y registro de la ruta

> Depende de las Fases 4 y 5 (el gate inyecta el repositorio real y registra el handler real).

- [x] 6.1 **RED**: Modificar `tests/unit/cmd/api/main_test.go` agregando un `fakeTaskRepository` (cuenta llamadas, devuelve `nil`); el test `TestTaskRouteRequiresCleanVersionNineAndExplicitDependency` con los casos `v7`, `v8` (deshabilitado), `v9`, `future 12` (habilitado), `nil dependency`, `dirty v9`, `lookup error`, `missing migration table` (verificando `readiness.Tasks` y que un `POST /projects/{id}/sprints/{id}/stories/{id}/tasks` con `{"tasks":[{"title":"A"}]}` responde `201` y una escritura solo cuando corresponde, `404` y cero escrituras en el resto); el caso de independencia (`HTTPDependencies{Tasks: ...}` con `Stories == nil` registra la ruta de tareas y responde `201`, mientras `POST /projects/{id}/stories` sigue sin registrarse); y la extensión de `TestMigrationReadinessSelectsRoutesIndependently` agregando la columna `tasks` (clean v8 → `false`; clean v9/future → `true`; estados sucios y errores de lectura → `false`). Ejecutar `go test ./tests/unit/cmd/api/... ./internal/api/...` y confirmar que **falla** (no existen `TaskDependencies`, `MigrationReadiness.Tasks` ni la ruta).
- [x] 6.2 **GREEN**: Modificar `internal/api/api.go` agregando el tipo `TaskDependencies{Repository taskapplication.TaskRepository; GenerateID taskapplication.IDGenerator}`, el campo `Tasks *TaskDependencies` en `HTTPDependencies`, el campo `Tasks bool` en `MigrationReadiness`, la línea `readiness.Tasks = version >= 9` en `ResolveMigrationReadiness` (después de `Assignment`), y en `NewHTTPHandlerWithDependencies` el registro condicional `if dependencies.Tasks != nil { ... mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks", taskhttp.NewCreateTasksHandler(createTasksUseCase)) }`, independiente del bloque de `Stories`. Modificar `cmd/api/main.go` agregando el bloque `if readiness := api.ResolveMigrationReadiness(...); readiness.Tasks { dependencies.Tasks = &api.TaskDependencies{Repository: taskpostgres.NewPostgresTaskRepository(pool), GenerateID: api.NewProjectID}; log.Printf("task creation available (schema version=%d)", version) } else { log.Printf("task creation unavailable until migration 000009 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr) }`, sin leer ni escribir `dependencies.Stories`. Ejecutar el mismo comando y confirmar que **pasa**.
- [x] 6.3 **REFACTOR**: Revisar el comentario existente de las líneas 35-37 de `cmd/api/main.go` para mencionar la versión 9, sin alterar el comportamiento de los gates ya probados.
- [x] 6.4 **Commit** (Unidad de Trabajo 6): `feat(api): gate task creation route behind migration 000009`. Traza TDD: `RED (TestTaskRouteRequiresCleanVersionNineAndExplicitDependency y TestMigrationReadinessSelectsRoutesIndependently fallan) -> GREEN (ambos pasan) -> REFACTOR (sin cambios de comportamiento)`.

## Fase 7: Integración HTTP real (end-to-end sobre PostgreSQL)

> Depende de las Fases 1 y 6 completas (necesita el esquema migrado y la composición real).

> **BLOQUEADO (sdd-apply)**: mismo motivo que la Fase 4 — Docker Desktop no tiene el motor corriendo en este entorno. Se creó `tests/integration/task/postgres/http_integration_test.go` componiendo el handler real (`api.NewHTTPHandlerWithDependencies` con `TaskDependencies` real) sobre el fixture compartido, y se confirmó que compila (`go vet` sin errores). La ejecución real falla con el mismo error de infraestructura de Testcontainers (`rootless Docker is not supported on Windows`), no con un fallo de lógica de negocio. Las tareas 7.1–7.3 quedan **sin marcar** hasta poder re-ejecutar con Docker disponible.

- [ ] 7.1 **RED/verificación**: Crear `tests/integration/task/postgres/http_integration_test.go` componiendo `api.NewHTTPHandlerWithDependencies(projectpostgres..., api.NewProjectID, api.HTTPDependencies{Tasks: &api.TaskDependencies{Repository: taskpostgres.NewPostgresTaskRepository(pool), GenerateID: api.NewProjectID}})` sobre el fixture compartido, con los casos: POST válido → `201`, dos tareas con UUIDs generados y filas presentes en `tasks`; POST para historia no asignada → `409 story_not_in_sprint`, 0 filas; POST con un elemento inválido → `422` con clave indexada, 0 filas. Ejecutar `go test ./tests/integration/task/postgres/...` (con Docker) y confirmar el resultado: si el stack de las Fases 1–6 está completo y correcto el test debe pasar al primer intento; si falla, el defecto está en la infraestructura o el transporte (Fases 4/5), no en este test — corregir el código de producción correspondiente y volver a ejecutar hasta que **pase**.
- [ ] 7.2 Confirmar que la suite completa de integración de tareas (`repository_integration_test.go` + `http_integration_test.go`) pasa junto en una sola corrida: `go test ./tests/integration/task/postgres/...`.
- [ ] 7.3 **Commit** (Unidad de Trabajo 7): `test(task): add end-to-end HTTP integration coverage`. Traza TDD: `RED/GREEN (TestCreateTasksHTTP... verificado contra el stack real) -> sin REFACTOR de producción (no se tocó código fuera de tests)`.

## Fase 8: Documentación

- [x] 8.1 Actualizar `README.md` donde hoy se listan rutas y migraciones (línea ~142) documentando la ruta nueva `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks` y el umbral de esquema `000009` que la habilita, siguiendo el mismo formato usado para las rutas existentes.
- [x] 8.2 **Commit** (Unidad de Trabajo 8): `docs: document task creation route and migration 000009`. Sin traza TDD (cambio de documentación, no requiere RED/GREEN).

## Fase 9: Verificación final

- [ ] 9.1 Ejecutar la suite completa `go test ./...` (con Docker disponible para las pruebas de integración con Testcontainers) y confirmar que todo pasa.
- [ ] 9.2 Revisar uno por uno los criterios de éxito de `proposal.md` (read-only) contra el comportamiento implementado y marcar cada uno como cumplido o pendiente con evidencia (comando ejecutado y resultado observado).
- [ ] 9.3 Reconfirmar, antes de abrir el PR, que `000009` sigue siendo el siguiente número libre dentro de `internal/project/infrastructure/postgres/migrations/` (read-only en esta verificación; riesgo ya documentado en la propuesta: "Otro cambio en curso toma el número `000009` antes de integrar").
- [ ] 9.4 Si la verificación detecta una corrección necesaria, aplicarla siguiendo el mismo ciclo RED → GREEN → REFACTOR de la fase correspondiente y agregar un commit adicional con su propia traza TDD; si no se detecta ninguna corrección, no se agrega commit nuevo en esta fase.

---

## Pronóstico de revisión (detalle final)

Estimación de líneas por grupo de tareas (agregadas + eliminadas, excluyendo archivos generados; `README.md` y migraciones SQL se contabilizan como texto autoral):

| Grupo (Unidad de Trabajo) | Archivos principales | Líneas estimadas |
|---|---|---|
| 1. Migración 000009 | `000009_create_tasks.up.sql`, `.down.sql`, `migration_files_test.go` (ajuste) | ~30 |
| 2. Dominio `Task` | `task.go`, `task_test.go` | ~170 |
| 3. Aplicación `CreateTasksUseCase` | `create_tasks.go`, `create_tasks_test.go` | ~240 |
| 4. Infraestructura `PostgresTaskRepository` | `repository.go`, `repository_integration_test.go` (+ helpers propios) | ~480 |
| 5. Transporte HTTP `CreateTasksHandler` | `handler.go`, `handler_test.go` (~20 casos de tabla) | ~440 |
| 6. Composición (`api.go`, `main.go`, `main_test.go`) | 3 archivos modificados | ~160 |
| 7. Integración HTTP real | `http_integration_test.go` | ~110 |
| 8. Documentación | `README.md` | ~10 |
| **Total estimado** | | **~1640** |

**Riesgo de presupuesto de 400 líneas**: Alto. La estimación total (~1640 líneas) supera el presupuesto en más de 4 veces.

**Decisión ya tomada**: el usuario eligió explícitamente `delivery_strategy: single-pr` y pidió no diseñar esto como una cadena de PRs. Se mantiene **un único Pull Request** para todo el cambio, con `size:exception` registrado y aceptado por el mantenedor antes de abrir el PR. Los commits dentro de ese PR siguen la división por Unidad de Trabajo de la tabla "Suggested Work Units" (una unidad = un commit, con su propio ciclo RED → GREEN → REFACTOR), de modo que el PR sea revisable commit por commit aunque el diff total exceda el presupuesto estándar.

`sdd-apply` **no debe** iniciar la implementación oversized sin que esta excepción (`size:exception`) esté explícitamente aceptada antes de la primera tarea de producción, conforme al guardrail de `delivery_strategy: single-pr` (`Decision needed before apply: Yes`).
