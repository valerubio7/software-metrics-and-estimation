# Tareas: Asignar historias a Sprint (US-09)

> Este cambio es retroactivo: el trabajo ya fue implementado, probado y fusionado antes de escribirse este documento. Las cifras de esta sección son las entregadas realmente, no un pronóstico.

## Carga de revisión entregada (real, no pronóstico)

| Campo | Valor |
|-------|-------|
| Archivos modificados | 23 |
| Líneas agregadas | 1158 |
| Líneas eliminadas | 45 |
| Líneas autoradas totales | 1203 |
| PRs utilizados | 1 (PR #78, consolidación directa a `main`) |
| Cadena de PRs hijos | No se usó la cadena planificada originalmente (#72–77); quedó abandonada y cerrada. |
| Estrategia de entrega efectiva | Único commit de comportamiento (`f544916`) + commit de cierre ODD (`3ad5433`), fusionados por `63a6527`. |

Confirmado con `git show --stat f544916`: 23 archivos, 1158 inserciones, 45 eliminaciones.

## Fase 1: Contrato de dominio/aplicación

- [x] 1.1 **RED — Aplicación:** `tests/unit/story/application/assign_stories_test.go` cubre selección vacía, duplicados y delegación con alcance de proyecto antes de que existiera `AssignStoriesUseCase`.
- [x] 1.2 **GREEN — Aplicación:** `internal/story/application/assign_stories.go` implementa `AssignStoriesCommand`, errores tipados (`ErrEmptyStorySelection`, `ErrDuplicateStoryID`, `ErrSprintNotFound`, `ErrSprintClosed`, `ErrProjectMismatch`, `ErrStoryAlreadyAssigned`, `ErrProjectScopedAssignerRequired`) y `AssignStoriesUseCase.Execute`.
- [x] 1.3 **REFACTOR — Aplicación:** interfaces pequeñas (`StorySprintAssigner`, `ProjectScopedStorySprintAssigner`); `go test ./tests/unit/story/application` en verde.

## Fase 2: Esquema y persistencia PostgreSQL

- [x] 2.1 **RED — Repositorio real:** `tests/integration/story/postgres/assignment_integration_test.go` (277 líneas) ejercita atomicidad, bloqueo, condiciones de carrera y preservación del backlog antes del repositorio final.
- [x] 2.2 **GREEN — Migraciones:** `000007_create_sprint_stories.{up,down}.sql` (tabla de asociación con FKs compuestas por proyecto) y `000008_add_sprint_closed.{up,down}.sql` (`sprints.is_closed`).
- [x] 2.3 **GREEN — Repositorio:** `AssignStories`/`AssignStoriesForProject` en `internal/story/infrastructure/postgres/repository.go` con transacción, `FOR UPDATE`, validación de pertenencia, `EXISTS` y manejo de `23505`.
- [x] 2.4 **REFACTOR — Persistencia:** `tests/integration/sprint/postgres/repository_integration_test.go` y `tests/integration/migrations/valid_history_convergence.sh` ampliados para la convergencia hasta v8.

## Fase 3: Contrato HTTP y gating

- [x] 3.1 **RED — Handler:** `tests/unit/story/transport/http/assign_stories_test.go` (159 líneas) cubre método, UUID, JSON, selección vacía/duplicada, canonicalización y mapeo de errores antes del handler.
- [x] 3.2 **GREEN — Handler:** `internal/story/transport/http/assign_stories_handler.go` con decodificación estricta, canonicalización de UUIDs y mapeo `201/400/404/409/422/500`.
- [x] 3.3 **RED/GREEN — Composición y readiness:** `tests/unit/cmd/api/main_test.go` ampliado (59 líneas) para `ResolveMigrationReadiness.Assignment: version >= 8`; `internal/api/api.go` y `cmd/api/main.go` componen el asignador solo con esa condición.
- [x] 3.4 **REFACTOR — HTTP/composición:** verificación de que las rutas preexistentes (proyectos, historias, Sprints, integrantes) no cambian de comportamiento.

## Fase 4: Integración, regresión y documentación

- [x] 4.1 **GREEN — Integración completa:** `tests/integration/story/postgres/http_integration_test.go` ampliado (74 líneas) para el endpoint end-to-end sobre PostgreSQL real.
- [x] 4.2 **REFACTOR — Verificación global (histórica, al momento de la fusión):** según el registro de entrega ya eliminado (`odd/tasks/us09-consolidation.md`), la suite completa reportó 183 tests + 346 subtests = 529 PASS, 0 FAIL/SKIP.
- [x] 4.3 **Documentación:** `README.md` actualizado (30 líneas) con el endpoint, contrato de error y requisito de esquema ≥8.

## Verificación re-ejecutada hoy (2026-10-03), para este expediente retroactivo

- [x] `go build ./...` — exit 0.
- [x] `go vet ./...` — exit 0, sin advertencias.
- [x] `go test ./tests/unit/...` — todos los paquetes `ok`, incluidos `tests/unit/story/application`, `tests/unit/story/transport/http` y `tests/unit/cmd/api`.
- [ ] Pruebas de integración (`tests/integration/story/postgres/assignment_integration_test.go` y demás, Testcontainers/Docker) — **omitidas por entorno**: Docker no está disponible en esta sesión. No es un fallo de código; es una limitación del entorno de verificación de hoy. La evidencia histórica de ejecución exitosa de estas pruebas está registrada en el commit `f544916` y en el registro de entrega ya eliminado.
- [x] Revisión manual de contratos contra `openspec/specs/historia/spec.md` (líneas 843–889) y `openspec/specs/sprint/spec.md` (líneas 60–77): sin inconsistencias encontradas.
