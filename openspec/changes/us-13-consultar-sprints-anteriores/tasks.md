# Tareas: US-13 — Consultar Sprints anteriores

TDD estricto (`strict_tdd: true`, `test_command: go test ./...`). Cada slice es un commit de unidad de trabajo (Conventional Commits) con código, pruebas y documentación juntos, y una traza breve RED -> GREEN -> TRIANGULATE -> REFACTOR en el cuerpo del mensaje. Una prueba de integración saltada por falta de Docker se registra como **saltada**; la unidad no se declara verificada sin contenedor.

Leyenda: **[DOCKER]** = la tarea necesita Docker (Testcontainers `postgres:16-alpine`); el orquestador debe pedir al usuario que la ejecute y reportar el resultado real (ejecutada, fallida o saltada).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1140 (rango 950-1250): producción ~260, pruebas ~845, docs ~37 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Un solo PR con `size:exception` (5 commits), o PR 1 (slices 1-2, ~320) -> PR 2 (slice 3, ~335) -> PR 3 (slice 4, ~265) -> PR 4 (slice 5, ~225) |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: pending
400-line budget risk: High

La decisión entre `size:exception` para un PR único o PR encadenados (`stacked-to-main` / `feature-branch-chain`) corresponde al usuario y **no** está tomada en este documento. `sdd-apply` no debe empezar hasta que se resuelva. Cualquiera sea la elección, el orden y el contenido de los 5 commits no cambia.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Migración `000011` (`sprints.seq` + `UNIQUE (project_id, seq)`) | PR 1 | `go test ./tests/integration/migrations/...` (sin Docker); `go test ./tests/integration/sprint/postgres/... -run Migration000011` **[DOCKER]** | Testcontainers: `up`, `seq` creciente, `23505` con `OVERRIDING SYSTEM VALUE`, `down` y `up` de nuevo | Revertir el commit 1: quita los dos `.sql` y los cambios de listas de prueba; la columna `seq` es transparente para `Create` y `CloseSprint` |
| 2 | Dominio de lectura y caso de uso `ListSprintsUseCase` con puerto `SprintHistoryReader` | PR 1 (base del PR 1 = rama tracker si `feature-branch-chain`) | `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application` | N/A: tipos puros y fake del puerto; ninguna ruta expuesta todavía | Revertir el commit 2: elimina tipos de lectura y `list_sprints.go`; nada más depende de ellos |
| 3 | `ListClosedByProject` sobre PostgreSQL (consulta única, filtro en `ON`, agrupación en Go) | PR 2 (base = PR 1 si `feature-branch-chain`) | `go test ./tests/integration/sprint/postgres/... -run ListClosedByProject` **[DOCKER]** | Testcontainers con migraciones `000001`-`000011`; instantánea `xmin` de cuatro tablas antes/después | Revertir el commit 3: elimina el método y la aserción del puerto; sin efecto en producción (aún no está compuesto) |
| 4 | Handler HTTP `ListSprintsHandler` con DTOs propios y mapa de errores | PR 3 (base = PR 2 si `feature-branch-chain`) | `go test ./tests/unit/sprint/transport/http` | N/A: `httptest` con caso de uso real sobre lector falso; sin registro en el router | Revertir el commit 4: elimina `list_handler.go` y su prueba; sin ruta registrada, sin efecto |
| 5 | Composición condicionada por `version >= 11`, `main.go`, documentación y config | PR 4 (base = PR 3 si `feature-branch-chain`) | `go test ./tests/unit/cmd/api/...`; `go test ./tests/integration/sprint/postgres/... -run SprintHistoryHTTP` **[DOCKER]** | `api.NewHTTPHandlerWithDependencies` con repositorio real (Docker) y `GET` de punta a punta | Revertir el commit 5 (o arrancar con esquema `< 11`) quita la ruta `GET`; creación, asignación y cierre quedan intactos |

## Slice 1: Migración `000011` (commit 1)

Commit: `feat(sprint): add the sprint creation sequence migration`

- [x] 1.1 RED (sin Docker): en `tests/integration/migrations/migration_files_test.go` agregar `"000011"` a la lista canónica y cambiar `len(pairs) != 11` con el mensaje "exactly eleven migration pairs"; ejecutar `go test ./tests/integration/migrations/...` y observar el fallo (faltan los archivos `000011`).
- [x] 1.2 GREEN: crear `internal/project/infrastructure/postgres/migrations/000011_add_sprint_creation_sequence.up.sql` (`ALTER TABLE sprints ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY;` y `ADD CONSTRAINT sprints_project_id_seq_key UNIQUE (project_id, seq);`) y `internal/project/infrastructure/postgres/migrations/000011_add_sprint_creation_sequence.down.sql` (drop de la constraint y luego de la columna, simétrico con `000004`); repetir `go test ./tests/integration/migrations/...` en verde.
- [ ] 1.3 [DOCKER] RED: en `tests/integration/sprint/postgres/repository_integration_test.go` agregar `000011_add_sprint_creation_sequence.up.sql` a `sprintMigrationNames` y escribir los tests de migración que deben fallar o quedar saltados sin contenedor: (a) `sprints.seq` existe como identidad `ALWAYS` y `NOT NULL` (`information_schema.columns`); (b) `sprints_project_id_seq_key` existe con la definición `UNIQUE (project_id, seq)` (`pg_constraint`); (c) dos `Create` consecutivos dan `seq` creciente; (d) `INSERT ... OVERRIDING SYSTEM VALUE` con `seq` repetido en el mismo proyecto falla con `23505` sobre `sprints_project_id_seq_key` (spec "Restricción de unicidad por proyecto"); (e) un `INSERT` con `seq` explícito sin `OVERRIDING` falla con `428C9`. _Nota: escrito, pendiente de ejecución con Docker._
- [ ] 1.4 [DOCKER] TRIANGULATE: agregar el test del `down` sobre una base con Sprints, asociaciones y finalizaciones: elimina columna y constraint y deja los datos de negocio idénticos; aplicar `up` de nuevo asigna `seq` único por proyecto a todas las filas (spec "Revertir la migración" y "Aplicar la migración"). _Nota: escrito, pendiente de ejecución con Docker._
- [ ] 1.5 [DOCKER] Verificar que los tests de integración existentes de Sprint (`Create`, asignación, finalización, cierre) siguen en verde con `seq` presente; ejecutar `go test ./tests/integration/sprint/postgres/...` y registrar el resultado real. _Nota: pendiente de ejecución con Docker; no requiere código nuevo (solo ejecutar la suite existente con `seq` presente)._
- [x] 1.6 REFACTOR: revisar duplicación en los helpers de catálogo de los tests nuevos; extraer solo si hay repetición real. Ejecutar `go test ./...` (con las pruebas Docker reportadas como ejecutadas o saltadas) y crear el commit 1 con la traza TDD.

## Slice 2: Dominio de lectura y caso de uso (commit 2)

Commit: `feat(sprint): add the list sprints use case`

- [x] 2.1 RED: en `tests/unit/sprint/domain/sprint_test.go` agregar una tabla para `PlannedStory.Completed()`: `CompletedAt` nil -> `false`; `CompletedAt` con marca -> `true`. Ejecutar `go test ./tests/unit/sprint/domain` y observar el fallo de compilación/ejecución.
- [x] 2.2 GREEN: en `internal/sprint/domain/sprint.go` agregar `ClosedSprint`, `PlannedStory` con `Completed()` y `SprintHistory` (sin `IsClosed`, sin `seq`, sin campos prohibidos; `Sprint`, `ValidationError` y `NewSprint` sin cambios); verificar `go test ./tests/unit/sprint/domain`.
- [x] 2.3 RED: crear `tests/unit/sprint/application/list_sprints_test.go` con un fake de `SprintHistoryReader` (solo lectura, contador de llamadas y `projectID` capturado) y los casos mínimos: UUID inválido -> `*domain.ValidationError` con `fields.project_id` y `calls == 0`; UUID válido -> una llamada y `SprintHistory{ProjectID canónico, Sprints}`. Ejecutar `go test ./tests/unit/sprint/application` y observar el fallo.
- [x] 2.4 GREEN: crear `internal/sprint/application/list_sprints.go` con `ListSprintsQuery`, la interfaz `SprintHistoryReader`, `ListSprintsUseCase`, `NewListSprintsUseCase` y `Execute` (parsea con `uuid.Parse` antes del puerto, usa el identificador canónico, devuelve el historial sin reordenar); reutilizar `ErrProjectNotFound` de `create_sprint.go`, sin modificarlo. Verificar las pruebas del paso 2.3.
- [x] 2.5 TRIANGULATE (RED y luego GREEN en el mismo archivo): ampliar `tests/unit/sprint/application/list_sprints_test.go` con tabla de identificadores hostiles (`"no-es-uuid"`, `"abc"`, `"123"`, UUID truncado, `""`) con cero llamadas; UUID en mayúsculas -> el puerto y el resultado reciben el canónico en minúsculas; el orden del fake se preserva; `ErrProjectNotFound` y error inesperado se propagan con `errors.Is`; si el fake devuelve Sprints junto con el error, `Execute` devuelve `SprintHistory{}` (nunca lista parcial). Ajustar `list_sprints.go` solo si algún caso falla.
- [x] 2.6 REFACTOR: solo si los tests muestran duplicación. Ejecutar `go test ./...` (sin Docker en esta unidad) y crear el commit 2 con la traza TDD.

## Slice 3: Almacenamiento PostgreSQL (commit 3) — requiere Docker

Commit: `feat(sprint): list closed sprints with planned stories`

Todas las tareas de este slice son **[DOCKER]**: el orquestador debe pedir al usuario que las ejecute y reportar el resultado real. Los tests se agregan en `tests/integration/sprint/postgres/repository_integration_test.go` y se escriben **antes** de `ListClosedByProject`, en el orden indicado (el primero fija el riesgo del `ON`).

- [ ] 3.1 [DOCKER] RED (cobertura de existencia y filtro, amenaza "Exposición de datos de otros proyectos o de Sprints abiertos"): escribir los tests (1) proyecto cuyo único Sprint está abierto -> slice vacío y `nil` (no `ErrProjectNotFound`); (2) proyecto sin Sprints -> slice vacío y `nil`; (3) proyecto inexistente -> `ErrProjectNotFound`. Ejecutar `go test ./tests/integration/sprint/postgres/... -run ListClosedByProject` y observar el fallo de compilación (método inexistente).
- [ ] 3.2 [DOCKER] RED (orden): (4) tres Sprints creados A, B, C y cerrados en el orden B, C, A -> devueltos C, B, A; (5) historias creadas S1, S2, S3 y asignadas en el orden S3, S1, S2 -> devueltas S1, S2, S3.
- [ ] 3.3 [DOCKER] RED (finalización, forma y aislamiento): (6) una historia en dos Sprints cerrados, completada solo en uno, con `stories.status='completada'` -> `CompletedAt` presente en uno y `nil` en el otro; (7) Sprint cerrado sin historias aparece con `Stories` vacío junto a otro con historias; (8) dos proyectos con Sprints cerrados -> solo los del consultado; (9) mapeo: `title`, `priority`, `story_points` `NULL` -> `nil` y `5` sembrado por SQL -> `5`, `completed_at` igual al instante sembrado.
- [ ] 3.4 [DOCKER] RED (solo lectura y error): (10) instantánea de solo lectura idéntica antes y después de consultar (proyecto con Sprints cerrados y abiertos, proyecto sin Sprints, proyecto inexistente, varias veces), usando `SELECT coalesce(string_agg(t.xmin::text || ' ' || to_jsonb(t)::text, E'\n' ORDER BY to_jsonb(t)::text), '') FROM <tabla> t` sobre `sprints`, `sprint_stories`, `stories` y `tasks`; (11) error no reinterpretado: con `sprints.seq` renombrada la consulta falla, el error no es `ErrProjectNotFound` y no se devuelven Sprints.
- [ ] 3.5 [DOCKER] GREEN: en `internal/sprint/infrastructure/postgres/repository.go` agregar la estructura privada `historyRow` con destinos anulables (`*string`, `*int`, `*time.Time`), `ListClosedByProject` con la sentencia única `projects p LEFT JOIN sprints sp ON sp.project_id = p.id AND sp.is_closed LEFT JOIN sprint_stories ss ... LEFT JOIN stories st ...` (`WHERE p.id = $1 ORDER BY sp.seq DESC, st.seq ASC`, con `is_closed` en el `ON`), agrupación por cambio de `sp.id`, cero filas -> `ErrProjectNotFound`, `rows.Err()` consultado y error sin reinterpretar devolviendo `nil`, y la aserción `var _ application.SprintHistoryReader = (*PostgresSprintRepository)(nil)`. Verificar los 11 tests en verde.
- [ ] 3.6 [DOCKER] TRIANGULATE: agregar un test de consultas repetidas sobre los mismos datos con resultados idénticos (spec "Orden estable" y "Consultas repetidas").
- [ ] 3.7 REFACTOR: revisar `ListClosedByProject` y los helpers de siembra de las pruebas por duplicación; mantener las firmas de `Create` y `CloseSprint` sin cambios. Ejecutar `go test ./...` y crear el commit 3 con la traza TDD.

## Slice 4: Handler HTTP (commit 4)

Commit: `feat(sprint): serve the sprint history over HTTP`

- [x] 4.1 RED (contrato base): crear `tests/unit/sprint/transport/http/list_handler_test.go` (`httptest` + `SetPathValue`, caso de uso **real** sobre un lector falso) con: `200` con contenedor `{"project_id","sprints"}` y orden del lector; `"sprints":[]` con un fake que devuelve `nil`; `"stories":[]` con un Sprint de `Stories: nil` (ambos comprobados sobre el cuerpo crudo); `422 validation_failed` con `fields.project_id` y cero llamadas al lector; `404 project_not_found` sin la clave `sprints`; `500 internal_error` con mensaje genérico `an unexpected error occurred` sin el texto del error interno. Ejecutar `go test ./tests/unit/sprint/transport/http` y observar el fallo.
- [x] 4.2 GREEN: crear `internal/sprint/transport/http/list_handler.go` con `ListSprintsHandler`, `NewListSprintsHandler`, los DTOs `sprintHistoryResponse`, `closedSprintResponse` y `plannedStoryResponse` (sin `omitempty` en `story_points` ni `completed_at`) y el mapa de errores (`errors.As` antes que `errors.Is`); reutilizar `errorResponse` y `writeJSON` de `handler.go` sin modificarlo; construir listas con `make(..., 0, n)`. Verificar las pruebas del paso 4.1.
- [x] 4.3 RED (representación y exposición): ampliar `list_handler_test.go` con claves exactas por Sprint (5: `id`, `project_id`, `sprint_goal`, `is_closed`, `stories`) y por historia (6: `id`, `title`, `priority`, `story_points`, `completed_at`, `completed`) decodificando a `map[string]json.RawMessage`, y ausencia de `seq`, `description`, `acceptance_criteria`, `estimated_hours` y `status` (spec "Campos no expuestos"); `is_closed: true`; `story_points` y `completed_at` `null` con `completed: false`; `completed_at` en UTC (`Z`) a partir de un `time.Time` en otra zona con `completed: true`; `Content-Type: application/json`.
- [x] 4.4 RED (métodos, amenaza "Routing HTTP"): `HEAD` aceptado con `200`; `POST` (u otro método) invocado directamente sobre el handler -> `405 method_not_allowed` con mensaje `only GET is supported` y cero llamadas al lector. Si algún test del 4.3 o 4.4 no pasa con el handler del 4.2, ajustar `list_handler.go` (GREEN) hasta que todos pasen.
- [x] 4.5 REFACTOR: ninguno previsto; los helpers de `handler.go` se reutilizan sin cambios. Ejecutar `go test ./...` y crear el commit 4 con la traza TDD. La ruta aún no está registrada en `internal/api/api.go`.

## Slice 5: Composición, readiness y documentación (commit 5)

Commit: `feat(api): expose the sprint history behind schema version 11`

- [ ] 5.1 RED unitario: crear `tests/unit/cmd/api/sprint_history_test.go` (estilo `sprint_close_test.go`, sin base de datos) con: `ResolveMigrationReadiness` -> `SprintHistory` verdadero solo con esquema limpio `>= 11` (v10 falso, v11 y v12 verdadero, `dirty` v11 falso, error de lectura falso) y `SprintClose` intacto en v8-v10; con `Lister` y creación: `GET` colección `200`/`404`/`422`, `HEAD` `200`, `POST` `201`, `PUT`/`DELETE` `405`, cero lecturas en los métodos no admitidos; sin `Lister` y con creación: `GET` `405`; sin dependencias de Sprint: `GET` `404`; versiones 5-10 -> `405` y `<5`/`dirty`/error -> `404`; la independencia del gate de cierre (v8-v10 mantiene el cierre y no la consulta). Ejecutar `go test ./tests/unit/cmd/api/...` y observar el fallo de compilación.
- [ ] 5.2 GREEN: en `internal/api/api.go` agregar `SprintDependencies.Lister sprintapplication.SprintHistoryReader`, `MigrationReadiness.SprintHistory` (`version >= 11` después del corte por `lookupErr`/`dirty`, sin tocar los demás flags) y el registro condicional de `GET /projects/{project_id}/sprints` dentro del bloque `Sprints != nil`, solo si `Lister != nil` e independiente de `Closer`; actualizar los comentarios para mencionar `000011`. Verificar `go test ./tests/unit/cmd/api/...`.
- [ ] 5.3 GREEN (cableado): en `cmd/api/main.go`, dentro de la rama `readiness.Sprints`, agregar `if readiness.SprintHistory { sprintDependencies.Lister = repository; log.Printf("sprint history available (schema version=%d)", version) }` reutilizando la misma instancia de `*PostgresSprintRepository` y actualizando el comentario del gate. En `tests/unit/cmd/api/main_test.go` solo agregar el comentario de una línea en el bucle `GET/PUT/DELETE -> 405` (línea 87) que explique que representa una composición sin consulta; no cambiar aserciones. Verificar que `fakeSprintRepository` sigue compilando.
- [ ] 5.4 [DOCKER] RED de integración en proceso: en `tests/integration/sprint/postgres/http_integration_test.go` agregar el flujo de punta a punta con `api.NewHTTPHandlerWithDependencies`, repositorio real y `Lister`: crear Sprints por `POST`, cerrarlos por `POST .../closure`, sembrar historias, asignaciones y finalizaciones por SQL; `GET` -> orden, agrupación, `completed` y `completed_at` UTC; Sprint abierto excluido; Sprint cerrado sin historias con `"stories":[]`; proyecto sin Sprints cerrados con `"sprints":[]`; proyecto inexistente `404` sin `sprints`; `project_id` inválido `422`; dos `GET` consecutivos con cuerpos idénticos. Si el cableado del 5.2 ya lo satisface, registrar el resultado observado; no fabricar un fallo previo.
- [ ] 5.5 [DOCKER] Ejecutar `go test ./tests/integration/sprint/postgres/... -run SprintHistoryHTTP` y reportar el resultado real (ejecutado, fallido o saltado).
- [ ] 5.6 Documentación: crear `docs/sprint-history.md` (precedente `docs/sprint-closure.md`) con contrato `GET`/`HEAD`, forma de la respuesta y ejemplo, orden (Sprints por creación descendente, historias por creación ascendente), listas vacías, errores (`404`, `422`, `405`, `500`), gate de esquema `>= 11` con `405`/`404` por debajo (5-10 / sin rutas), `story_points` siempre `null` por ahora, ausencia de paginación y limitación del `seq` de Sprints previos a la migración; verificar que los ejemplos coinciden con el handler. En `openspec/config.yaml` agregar la nota "Implementada: US-13 (consultar Sprints anteriores)."
- [ ] 5.7 REFACTOR y cierre: revisar duplicación en las pruebas de composición. Ejecutar `go test ./...` completo (Docker reportado como ejecutado o saltado) y revisar el diff (`git diff main --stat`) para confirmar solo lectura, ausencia de cambios en creación, asignación y cierre, y que `handler.go` y `create_sprint.go` no cambiaron. Crear el commit 5 con la traza TDD.

## Matriz de amenazas: cobertura planificada

| Amenaza (diseño) | Tarea RED |
|------------------|-----------|
| Routing HTTP: solapamiento y métodos | 4.4, 5.1 |
| Identificador de ruta hostil | 2.3, 2.5, 4.1 |
| Exposición de datos de otros proyectos o de Sprints abiertos | 3.1, 3.3, 5.4 |
| Solo lectura | 2.5 (fake solo de lectura), 3.4 |
| Fuga de información en errores | 3.4, 4.1 |
| Respuesta sin límite | Riesgo aceptado; documentado en 5.6 |

## Orden y dependencias

Orden estricto 1 -> 2 -> 3 -> 4 -> 5; cada commit compila y deja `go test ./...` en verde por sí solo, y ninguno expone la ruta hasta el slice 5. Los slices 2 y 4 no dependen de Docker y 2 podría hacerse antes que 1 técnicamente, pero se conserva el orden del diseño para que la migración exista antes de la consulta (slice 3 depende de 1 y 2; slice 4 depende de 2; slice 5 depende de 2, 3 y 4). Tareas con Docker: 1.3, 1.4, 1.5, 3.1-3.7 y 5.4-5.5.
