# Exploración: US-15 — Registrar el esfuerzo realizado en una historia o tarea

Issue: #43. La operación desde la TUI es TUI-08 (#101) y queda fuera de este cambio.

## Estado actual (verificado)

- No existe ninguna tabla, módulo ni ruta de esfuerzo.
- Las migraciones llegan hasta `000011` (`000010_add_sprint_story_completion`, `000011_add_sprint_creation_sequence`). La siguiente es `000012`.
- No hay una migración `000003` duplicada: la nota correspondiente de `openspec/config.yaml` está desactualizada.
- `tests/integration/migrations/migration_files_test.go` exige exactamente 11 pares de migraciones; hay que actualizarlo a 12.
- Los módulos siguen cuatro capas (`domain`, `application`, `infrastructure/postgres`, `transport/http`). `internal/task` es el patrón más reciente: `400 invalid_request`, `422 validation_failed` con `fields`, `404 *_not_found` y `500 internal_error`.
- `internal/api/api.go` registra rutas opcionales mediante punteros de dependencias y `MigrationReadiness`; esfuerzo exigiría `version >= 12`.
- `stories` y `sprints` tienen `UNIQUE(id, project_id)`; `tasks` y `project_members` no.
- Una tarea solo existe dentro de un Sprint (FK a `sprint_stories`); una historia puede estar solo en el backlog.
- Fechas: formato `YYYY-MM-DD`, parseo con round-trip en la capa de aplicación y reloj inyectado `func() time.Time`.
- Los IDs se generan en el servidor con `api.NewProjectID`.
- Lecturas necesarias para la TUI (`GET /projects/{id}/members` y `GET /projects/{id}/stories/{id}/tasks`) ya existen.

## Enfoque recomendado

Módulo nuevo `internal/effort` con la tabla `effort_records`:

- `story_id` obligatorio y `task_id` opcional. Todo registro cuelga de una historia, de modo que US-19 agrega con un `GROUP BY story_id`.
- Claves foráneas compuestas que garantizan que integrante, historia y tarea pertenecen al mismo proyecto y que la tarea es de esa historia.
- Requiere `UNIQUE(id, project_id)` en `project_members` y `UNIQUE(id, story_id, project_id)` en `tasks`.
- Alternativas descartadas: `story_id`/`task_id` anulables con CHECK XOR (obliga a unir con `tasks` para agregar) y dos tablas (duplica código y lecturas).

## Decisiones del usuario

| # | Tema | Decisión |
|---|------|----------|
| 1 | Sprint cerrado | Se permite registrar esfuerzo. |
| 2 | Historia sin Sprint (backlog) | Se permite. |
| 3 | Fecha futura | Se rechaza, comparando con la fecha UTC del servidor y un reloj inyectado. |
| 4 | Ruta | `POST /projects/{project_id}/stories/{story_id}/effort-records`; cuerpo `{member_id, task_id?, work_date, description, hours}`; responde `201`. |
| 5 | Errores de referencia | `404 member_not_found` y `404 task_not_found` (alcance de proyecto/historia); `422 validation_failed` solo para campos del cuerpo. |
| 6 | Listado | Sin `GET` en esta US. |
| 7 | Límites | Horas > 0 y ≤ 24 con 2 decimales (`NUMERIC(5,2)`); descripción obligatoria, recortada, hasta 500 caracteres; duplicados permitidos. |
| 8 | Entrega | Un solo PR con `size:exception`. |

## Áreas afectadas

- `internal/project/infrastructure/postgres/migrations/000012_create_effort_records.{up,down}.sql` (nuevo).
- `tests/integration/migrations/migration_files_test.go` (12 pares).
- `internal/effort/{domain,application,infrastructure/postgres,transport/http}` (nuevo).
- `internal/api/api.go` y `cmd/api/main.go` (dependencias, bandera `Effort`, ruta).
- `tests/unit/effort/...`, `tests/unit/cmd/api/...`, `tests/integration/effort/postgres/...` (nuevo).
- `openspec/specs/esfuerzo/spec.md` (nuevo, al archivar), fila nueva en la tabla de rutas del `README.md` y corrección de la nota de `openspec/config.yaml`.

## Tamaño estimado

Aproximadamente 1.300–1.500 líneas con tests (migración 40, dominio 90, aplicación 130, repositorio 140, HTTP 190, cableado 50, tests unitarios 450, integración 300, spec y ajustes 100). Supera el presupuesto de 400 líneas; se acepta `size:exception` con un PR único.

## Riesgos

- Los `UNIQUE` adicionales en `tasks` y `project_members` tocan tablas existentes: hay que probar el `down` y la reaplicación.
- La spec debe dejar explícito que el esfuerzo de una tarea también pertenece a su historia, para que US-19 no cuente doble.
- La validación de fecha depende del reloj y de la zona horaria: el reloj se inyecta para tests deterministas.
- Los tests de integración requieren Docker.
- `openspec/specs/tarea/spec.md` menciona un estado de tarea "diferido a US-15" que el issue #43 no incluye: queda fuera de alcance y se anota en la propuesta.

## Listo para propuesta

Sí.
