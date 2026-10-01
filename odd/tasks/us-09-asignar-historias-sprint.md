# ODD: Asignar historias a un Sprint (HU-09)

**Estado:** HU-09 completada, verificada, commiteada y issue #37 cerrada. Los artefactos SDD fueron reconciliados con la decisión final de no depender de US-12 y usar `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`.

## Objetivo

Permitir seleccionar una o varias historias del Product Backlog del proyecto de un Sprint existente y asociarlas como trabajo planificado, con validación atómica de todo el lote.

## Reglas confirmadas

- Una operación permite seleccionar múltiples historias.
- Si cualquier historia o Sprint no es elegible, no se persiste ninguna asociación del lote.
- Las historias siguen perteneciendo al Product Backlog, mantienen su proyecto y estado; la asociación es adicional.
- No se repite una historia dentro del mismo Sprint ni se asignan historias de otro proyecto.
- No se asignan historias a Sprints cerrados. Para HU-09 la fuente de verdad es `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`.
- SDD artefactos en español; strict TDD, `go test ./...`.
- Rama feature/tracker: `feat/us-09-asignar-historias-sprint`, creada desde `main` en `3e26553`.
- Entrega originalmente prevista: `auto-chain`, estrategia `feature-branch-chain`, máximo 400 líneas por PR, sin `size:exception`.
- El usuario autorizó commit y cierre de issue. No se hizo push ni se creó PR.

## Artefactos SDD

- `openspec/changes/us-09-asignar-historias-sprint/exploration.md`
- `openspec/changes/us-09-asignar-historias-sprint/proposal.md`
- `openspec/changes/us-09-asignar-historias-sprint/specs/historia/spec.md`
- `openspec/changes/us-09-asignar-historias-sprint/specs/sprint/spec.md`
- `openspec/changes/us-09-asignar-historias-sprint/design.md`
- `openspec/changes/us-09-asignar-historias-sprint/tasks.md`
- `openspec/changes/us-09-asignar-historias-sprint/apply-progress.md`

## Cortes y evidencia

### Corte 1 — Contrato de aplicación — COMPLETO

- Tareas SDD: 1–2.
- Archivos: `internal/story/application/assign_stories.go`, `tests/unit/story/application/assign_stories_test.go`.
- Implementa lote no vacío, IDs únicos y una sola llamada al puerto de asignación atómica.
- Strict TDD RED/GREEN y triangulación registrada en apply-progress.
- `go test -count=1 ./tests/unit/story/...` pasó; 140 líneas añadidas (59 producción, 81 tests), bajo meta 200.
- En ese momento `go test ./...` estaba bloqueado por `undefined: stories` en `internal/api/api.go`; el problema se corrigió posteriormente en la tarea separada `odd/tasks/repair-api-startup-compilation.md` y la suite completa ahora pasa.
- Sin commit.

### Corte 2 — Persistencia PostgreSQL — COMPLETO

- Tareas SDD: 3–4.
- Archivos: migraciones `000005_create_sprint_stories.{up,down}.sql`, `internal/story/infrastructure/postgres/repository.go`, `tests/integration/story/postgres/assignment_integration_test.go` y ajuste del fixture de migraciones.
- Asociación `sprint_stories`, FKs compuestas `(id, project_id)`, clave única Sprint-historia, verificación y escrituras en transacción con rollback.
- Pruebas cubren asociación, preservación de backlog/proyecto/estado, rechazo atómico, error SQL, carrera concurrente y migración down/up.
- Testcontainers PostgreSQL 16 y unit tests pasaron; verificación independiente pasó. 287 líneas añadidas, debajo de meta 300.
- Sin commit.

### Corte 3 — HTTP, integración, regla de cierre y documentación — COMPLETO

- Tareas SDD: 5–9; verificación final independiente completada.
- El usuario autorizó reemplazar la dependencia US-12 (#40) con `is_closed BOOLEAN NOT NULL DEFAULT false`.
- Se añadió handler POST, wiring de API y disponibilidad desde migración 000006; PostgreSQL rechaza cerrado antes de cualquier escritura.
- Se agregaron pruebas HTTP y de integración del guardado cerrado. El primer RED fue observado: faltaban `ErrSprintClosed` y `NewAssignStoriesHandler`.
- GREEN enfocado: `go test ./tests/unit/story/transport/http ./tests/unit/story/application ./tests/unit/cmd/api` pasó.
- Commit: `3c71430` (`feat(story): assign backlog stories to sprints`). Issue #37 cerrada. Worktree limpio.

### Regresión del verificador: proyecto de ruta — COMPLETA

- Strict TDD RED/GREEN/TRIANGULATE/REFACTOR registrado en `openspec/changes/us-09-asignar-historias-sprint/apply-progress.md` y `tasks.md`.
- El endpoint valida y propaga `project_id`; el caso de uso usa el puerto project-scoped y PostgreSQL rechaza en la transacción el mismatch con el Sprint antes de escribir. Las historias también se comparan con el proyecto del Sprint.
- Pruebas HTTP y de aplicación uncached, integración PostgreSQL dirigida y `go test ./...` pasaron. No se realizaron commits ni publicaciones.

## Corrección de prueba de composición — assigner project-scoped

- Actualizada `TestAssignmentRouteCompositionRequiresAssigner` para usar un fake con `AssignStoriesForProject`, el contrato project-scoped exigido por HU-09. Se conserva que sin assigner la ruta no existe (404).
- RED uncached observado con el fake legacy: HTTP 500 y cero llamadas. GREEN: `go test -count=1 ./tests/unit/cmd/api` pasó con el fake project-scoped; también pasaron `go test ./...` y `git diff --check`.

## Decisión del usuario

El usuario autorizó sustituir la dependencia anterior de US-12 (#40) por el campo persistente `Sprint.is_closed BOOLEAN NOT NULL DEFAULT false`, y exige rechazar asignaciones cerradas. OpenSpec se reconcilió con esta decisión.
