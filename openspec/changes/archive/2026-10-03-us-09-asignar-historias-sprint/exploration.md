# Exploración: US-09 — Asignar historias a Sprint

## Nota de origen

Esta exploración es retroactiva. US-09 ya está implementada y fusionada en `main` (PR #78, commit de funcionalidad `f544916`, cierre de issue #37); esta investigación documenta el estado real del código y de las pruebas para completar el expediente SDD que nunca se generó durante la entrega original.

## Alcance y fuente

- Issue #37 (GitHub), ya cerrada: asignar un lote de historias existentes del Product Backlog a un Sprint del mismo proyecto, sin retirarlas del backlog ni modificar su estado/estimaciones.
- La entrega real no siguió la cadena de PRs hijos planificada originalmente (`feat/us-09-assignment-01-core`..`06-progress`, tracker PR #71, PRs #72–77): esa cadena fue construida contra ramas intermedias y nunca llegó a fusionarse en `main`. La integración efectiva se rehízo directamente sobre el esquema canónico real de `main`, en un único commit de consolidación (`fix/us09-consolidation`, PR #78), portando el comportamiento desde `feat/us-09-assignment-05-sdd` commit `e4ddb66`.
- Las especificaciones canónicas ya contienen el texto de requisito fusionado directamente por el commit `f544916`, sin pasar por la fase `sdd-spec` de este cambio: `openspec/specs/historia/spec.md` líneas 843–889 (`### Requirement: Asignar un lote de historias sin retirarlas del Product Backlog (US-09)` y `### Requirement: Habilitar asignación solo con esquema completo y dependencia explícita`) y `openspec/specs/sprint/spec.md` líneas 60–77 (`### Requisito: Rechazar asignaciones a un Sprint cerrado (US-09)`).

## Estado actual del código

- Endpoint: `POST /projects/{project_id}/sprints/{sprint_id}/stories`, body `{"story_ids":["<uuid>",...]}`. Handler en `internal/story/transport/http/assign_stories_handler.go`.
- Caso de uso `AssignStoriesUseCase.Execute` en `internal/story/application/assign_stories.go:44`: rechaza selección vacía (`ErrEmptyStorySelection`) y duplicados (`ErrDuplicateStoryID`) antes de delegar en `ProjectScopedStorySprintAssigner.AssignStoriesForProject`.
- Repositorio `PostgresStoryRepository.AssignStoriesForProject` en `internal/story/infrastructure/postgres/repository.go:38`: abre una transacción, bloquea la fila del Sprint con `SELECT ... FOR UPDATE`, valida coincidencia de proyecto, estado cerrado, existencia/pertenencia de cada historia, preexistencia de la asociación (`EXISTS`), inserta una por una con manejo de `23505` (`sprint_stories_pkey`) como red de seguridad ante condiciones de carrera, y confirma solo si todo el lote tiene éxito.
- Migración `000007_create_sprint_stories.up.sql`: crea `sprint_stories(sprint_id, story_id, project_id)` con PK compuesta `(sprint_id, story_id)` y dos FKs compuestas `(sprint_id, project_id)` → `sprints(id, project_id)` y `(story_id, project_id)` → `stories(id, project_id)`, ambas `ON DELETE RESTRICT`; agrega además `UNIQUE(id, project_id)` en `stories` y `sprints` para habilitar esas FKs compuestas. Esta FK compuesta sobre `project_id` compartido es la autoridad a nivel de base de datos para "mismo proyecto", en defensa en profundidad junto con la comprobación a nivel de aplicación en el repositorio.
- Migración `000008_add_sprint_closed.up.sql`: agrega `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`.
- Gating de arranque: `api.ResolveMigrationReadiness` (`internal/api/api.go:95`) fija `Assignment: version >= 8`; `cmd/api/main.go` solo compone el asignador cuando esa condición se cumple.
- Canonicalización de UUIDs: el handler aplica `uuid.Parse(...).String()` sobre cada `story_id` **antes** de la deduplicación/validación del caso de uso, de modo que alias de un mismo UUID con distinta capitalización colapsan y se rechazan como duplicado (422), en vez de fusionarse silenciosamente o insertarse dos veces.
- `ListByProject` no filtra historias ya asignadas: permanecen visibles en el Product Backlog (verificado por prueba, igualdad byte a byte antes/después).

## Fragilidad de diseño detectada

`dependencies.Stories` se compone de forma no nula ya desde `version >= 2` (`cmd/api/main.go:53`, la rama más baja del switch de tres ramas `>=4`/`>=3`/`>=2`) y `readiness.Assignment` (`version >= 8`, en `internal/api/api.go:95`) son dos comprobaciones de versión separadas en dos archivos distintos que deben mantenerse coherentes. Hoy no es un defecto porque `>=8` implica `>=2` con margen de sobra, pero es una fragilidad a vigilar en futuras migraciones: un cambio futuro a uno de los dos umbrales sin ajustar el otro podría habilitar la ruta de asignación sin su dependencia de historias, o viceversa.

## Cobertura de pruebas

Pruebas unitarias y de integración existentes que verifican los contratos:

- `TestAssignStoriesRejectsCrossProjectBatchWithoutPartialWrites`
- `TestAssignStoriesRejectsMismatchedRouteProjectWithoutWrites`
- `TestAssignStoriesConcurrentDuplicateHasSingleWinner`
- `TestAssignStoriesWaitsForConcurrentSprintClose`
- `TestAssignStoriesRollsBackWhenAnInsertFails`
- `TestAssignStoriesAtomicallyPreservesBacklogAndStoryFields`
- `TestCON01UUIDAliasesAreDuplicate`
- `TestAssignStoriesHandlerCanonicalizesWithoutChangingSelectionOrder`

Ubicadas en `tests/unit/story/application/assign_stories_test.go`, `tests/unit/story/transport/http/assign_stories_test.go` y `tests/integration/story/postgres/assignment_integration_test.go`.

## Verificación ejecutada hoy (2026-10-03)

- `go build ./...`: exit 0.
- `go vet ./...`: exit 0, sin advertencias.
- `go test ./tests/unit/...`: todos los paquetes `ok`, incluidos `tests/unit/story/application`, `tests/unit/story/transport/http` y `tests/unit/cmd/api`.
- Pruebas de integración (Testcontainers/Docker): **no se pudieron ejecutar** porque Docker no está disponible en este entorno. Esto es una limitación del entorno, explícitamente **no** un defecto de código.
- No se encontraron inconsistencias, errores ni validaciones faltantes contra ninguno de los contratos declarados (coincidencia de proyecto, no duplicados, rechazo de Sprint cerrado, atomicidad, visibilidad en backlog, canonicalización de UUID).

## Evidencia de entrega previa

- Commit de comportamiento: `f544916 feat(story): integrar US-09 sobre migraciones canónicas` — 23 archivos, 1158 inserciones, 45 eliminaciones (confirmado con `git show --stat f544916`).
- Commit de cierre ODD (ahora reemplazado por esta documentación SDD): `3ad5433 docs(odd): cerrar verificación y entrega de US-09`.
- Commit de fusión: `63a6527 Merge pull request #78 from valerubio7/fix/us09-consolidation`.
- Según el registro previo (ya eliminado, `odd/tasks/us09-consolidation.md`): la suite completa reportó 183 tests + 346 subtests = 529 PASS, 0 FAIL/SKIP al momento de la fusión.

## Listo para propuesta

Sí. El código, las especificaciones canónicas y las pruebas ya existen y están fusionados; esta exploración solo documenta ese estado para habilitar el resto del expediente SDD retroactivo.
