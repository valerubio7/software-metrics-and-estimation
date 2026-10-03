# Apply progress — US-10 descomponer una historia del Sprint en tareas

## Resumen

33/33 tareas completas en la rama `feat/us10-descomponer-historia-tareas`, en dos corridas de `sdd-apply`. La primera corrida (22/33) quedó bloqueada en las Fases 4, 7 y 9 porque Docker Desktop no tenía el motor corriendo en ese momento; la segunda corrida, con Docker ya disponible, cerró esas tres fases y encontró y corrigió dos defectos reales que la primera corrida no pudo detectar por falta de ejecución contra PostgreSQL real.

## Ciclo TDD por unidad de trabajo

| Unidad de trabajo | RED | GREEN | REFACTOR |
|---|---|---|---|
| 1. Migración `000009_create_tasks` | `tests/integration/migrations/migration_files_test.go` modificado para exigir 9 pares (antes 8); falla porque `000009_create_tasks.{up,down}.sql` no existían. | Se crean ambos archivos de migración con la tabla `tasks`, sus 3 FKs compuestas (`RESTRICT`) y el índice; el mismo test pasa. | Revisión de nombres de constraints y formato SQL contra `000001`-`000008`, sin cambiar esquema ni comportamiento. |
| 2. Dominio `Task` | `tests/unit/task/domain/task_test.go` falla: el paquete `internal/task/domain` no existe. | `task.go` implementa `Task`, `ValidationError`, `NewTask`, validación de título y horas (mismo patrón que `Story`); el test pasa. | Revisión de duplicación interna sin alterar mensajes de error. |
| 3. Aplicación `CreateTasksUseCase` | `tests/unit/task/application/create_tasks_test.go` falla: paquete `internal/task/application` no existe. | `create_tasks.go` implementa errores centinela, `TaskRepository`, `CreateTasksUseCase.Execute` con validación acumulada por índice; el test pasa. | Revisión de nombres sin alterar la firma del puerto. |
| 4. Infraestructura `PostgresTaskRepository` | `tests/integration/task/postgres/repository_integration_test.go` falla: paquete no existe. En la segunda corrida, con Docker arriba, el código de la primera corrida **falló de verdad** contra PostgreSQL real: `cannot scan int4 (OID 23) in binary format into *bool` en las tres verificaciones de pertenencia, más un `EOF` intermitente en el primer `apply` de migración por falta de espera de disponibilidad del contenedor. | Corrección: los destinos de `Scan` pasan de `bool` a `int` (el literal `SELECT 1` es `int4`, no `bool`, y el protocolo binario de pgx rechaza esa conversión cuando la fila existe); se agrega `waitForTaskDatabaseReady` (bucle de `Ping`, igual que `sprint`/`story`/`project`). Las 10 pruebas de `task/postgres` pasan. | Sin cambios de comportamiento adicionales; confirmado con una segunda corrida independiente del orquestador (75.5s, 10/10 verde). |
| 5. Transporte HTTP `CreateTasksHandler` | `tests/unit/task/transport/http/handler_test.go` falla: paquete no existe. | `handler.go` implementa decodificación estricta, DTOs y mapeo de errores (405/400/422/404×3/409/500); el test pasa. | Revisión de DTOs y mensajes contra la convención de `story/transport/http`. |
| 6. Composición `api.go` + `main.go` | `tests/unit/cmd/api/main_test.go` falla: no existen `TaskDependencies`, `MigrationReadiness.Tasks` ni la ruta. | Se agrega el gate `version >= 9` y el registro condicional de la ruta, independiente de `dependencies.Stories`; el test pasa. | Comentario de `main.go` actualizado para mencionar la versión 9. |
| 7. Integración HTTP end-to-end | `tests/integration/task/postgres/http_integration_test.go` escrito contra el stack real. En la primera corrida (sin Docker) el código quedó escrito pero sin confirmar. En la segunda corrida, con Docker arriba, pasó tal cual una vez corregido el bug de la Fase 4 (del que dependía). | 3 casos (201 con dos tareas, 409 sin pertenencia, 422 con elemento inválido) en verde contra PostgreSQL real. | Sin cambios de producción adicionales (archivo de test únicamente). |
| 8. Documentación `README.md` | No aplica (cambio de documentación). | `README.md` actualizado con la ruta nueva y el umbral `000009`. | No aplica. |
| 9. Verificación final | `go test ./...` con paralelismo por defecto mostró fallas de contención de recursos de Docker Desktop (incluso en el módulo preexistente `project`, no tocado por este cambio) — ambiental, no un defecto de US-10. | `go test -p 1 ./...` (serializa binarios de test por paquete, no cambia qué se ejecuta ni sus aserciones): **0 fallas, 0 saltos**, en los 29 paquetes con tests, incluidas las 5 suites de integración con Testcontainers. Repetido de forma independiente por el orquestador sobre `task/postgres` (75.5s, verde). | Revisión uno a uno de los 7 criterios de éxito de `proposal.md` (solo lectura) contra el comportamiento implementado; reconfirmación de que `000009` sigue siendo el número libre (`TestMigrationVersionsAreUnique` pasa). |

## Commits en `feat/us10-descomponer-historia-tareas`

| Commit | Mensaje | Traza TDD |
|---|---|---|
| `c394d08` | `feat(migrations): add tasks table migration 000009` | RED (`migration_files_test` espera 9 pares, falla sin `000009`) → GREEN (pasa con `000009` creada) → REFACTOR (sin cambios de comportamiento) |
| `99dea80` | `feat(task): add Task domain validation` | RED (`TestNewTask` falla, paquete inexistente) → GREEN (pasa) → REFACTOR (sin cambios de comportamiento) |
| `6b46ec5` | `feat(task): add CreateTasksUseCase with batch validation` | RED (`TestCreateTasksUseCaseExecute` falla, paquete inexistente) → GREEN (pasa) → REFACTOR (sin cambios de comportamiento) |
| `93fcb05` | `feat(task): add PostgresTaskRepository with sprint-story membership check` | Escrito contra fakes/compilación en la primera corrida (Docker caído); confirmado GREEN real recién en `47ce4c1` |
| `761a8cf` | `feat(task): add CreateTasksHandler with strict decoding and error mapping` | RED (`TestCreateTasksHandler` falla, paquete inexistente) → GREEN (pasa) → REFACTOR (sin cambios de comportamiento) |
| `81b1576` | `feat(api): gate task creation route behind migration 000009` | RED (`TestTaskRouteRequiresCleanVersionNineAndExplicitDependency` y `TestMigrationReadinessSelectsRoutesIndependently` fallan) → GREEN (ambos pasan) → REFACTOR (sin cambios de comportamiento) |
| `1b5abc1` | `test(task): add end-to-end HTTP integration coverage` | Escrito contra el stack real en la primera corrida (Docker caído); confirmado GREEN real recién en la segunda corrida |
| `191bb3a` | `docs: document task creation route and migration 000009` | Sin traza TDD (documentación) |
| `2b3a0c4` | `docs(sdd): record final verification evidence for US-10 (Docker-blocked)` | Cierre de la primera corrida: 22/33, bloqueo documentado |
| `47ce4c1` | `fix(task): fix membership scan type mismatch and container readiness race` | RED real (falla contra Postgres real: `int4→bool` scan error + `EOF` intermitente) → GREEN real (10/10 `task/postgres` pasa) → REFACTOR (sin cambios de comportamiento adicionales) |
| `3e7361b` | `docs(sdd): confirm final verification for US-10 with Docker available` | Cierre de la segunda corrida: 33/33, suite completa verde |

## Defectos reales encontrados durante `apply` (no en diseño ni en specs)

1. **Scan `int4→bool` en `PostgresTaskRepository`**: las tres verificaciones de pertenencia (`SELECT 1 FROM sprints/stories/sprint_stories ...`) escaneaban el literal `SELECT 1` (tipo `int4` en Postgres) directamente a una variable `bool`. El protocolo binario de pgx rechaza esa conversión cuando la fila existe — sólo el camino `ErrNoRows` funcionaba antes de la corrección. Un test unitario con fakes no podía detectar esto; sólo apareció en la ejecución real contra PostgreSQL. Corregido cambiando los destinos de `Scan` a `int`.
2. **Carrera de disponibilidad del contenedor de test**: `taskDatabase(t)` en `repository_integration_test.go` no esperaba a que PostgreSQL aceptara conexiones después de que Testcontainers reportara "ready", a diferencia de los helpers hermanos de `sprint`/`story`/`project`. Causaba un `EOF` intermitente en el primer `apply` de migración. Corregido agregando `waitForTaskDatabaseReady` (bucle de `Ping`, mismo patrón que los módulos existentes).

Ambos quedaron ocultos durante la primera corrida porque, sin Docker, sólo se pudo confirmar `go vet`/`go build` (compila), nunca ejecución real — una lección ya registrada en memoria de la sesión: "compila" no es lo mismo que "verificado".

## Verificación independiente del orquestador

Antes de reportar esto como completo, el orquestador (no el subagente) verificó por su cuenta:
- `git log` confirma los 11 commits listados arriba, en orden, en `feat/us10-descomponer-historia-tareas`.
- `git status` limpio, sin cambios sueltos.
- `grep` sobre `tasks.md` confirma 0 checkboxes `[ ]` sin marcar.
- Re-ejecución independiente de `go test -p 1 ./tests/integration/task/postgres/... -v`: **PASS, 75.573s**, incluidas `TestTasksTableEnforcesConstraints`, `TestTaskBlocksUnassigningItsStory` y `TestTasksMigrationCanBeReversedAndReapplied`.
- `docker info` confirmado dos veces: primero mostrando el error real de motor caído (bloqueo genuino, no inventado), después mostrando info real del servidor `docker-desktop` tras que el usuario lo levantó.

## Estado

`sdd-apply` para US-10 está **completo** (33/33 tareas, no parcial). Listo para `sdd-verify` (opcional) o `sdd-archive`.
