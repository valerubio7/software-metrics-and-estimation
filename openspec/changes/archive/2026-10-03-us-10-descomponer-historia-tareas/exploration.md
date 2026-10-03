# Exploración: US-10 — Descomponer una historia del Sprint en tareas

## Resumen del problema

US-10 (issue #38) requiere permitir crear una o más tareas asociadas a una historia que ya está asignada a un Sprint, con información suficiente para identificar el trabajo y una estimación opcional de horas. La restricción dura es: no se debe poder crear una tarea para una historia que no pertenezca al Sprint seleccionado.

Esa pertenencia hoy se modela exclusivamente en la tabla `sprint_stories` creada por US-09 (migración `000007`), sin un caso de uso de lectura reutilizable que la verifique — hay que construir esa verificación desde cero, replicando el patrón SQL ya usado en la asignación, no el código en sí.

## Hallazgos concretos

### Dominio Historia (`internal/story/domain/story.go`)

- `Story` struct (línea 45): `ID, ProjectID, Title, Description, Priority, Status, StoryPoints, AcceptanceCriteria, EstimatedHours *float64`.
- Estados cerrados en español: `StatusPending="pendiente"`, `StatusInProgress="en_progreso"`, `StatusCompleted="completada"` (líneas 10-15).
- Patrón de validación de horas estimadas, reutilizable para la estimación de esfuerzo de la tarea: `maxEstimatedHours=99999.99`, `maxEstimatedHoursDecimal=2`, `estimatedHoursMessage` (líneas 28-30); `validateEstimatedHours` (líneas 123-134) cuenta decimales sobre la representación mínima del `float64` (`decimalPlaces`, líneas 136-142). `*float64` nil = "sin estimar" es un patrón válido y directo para la estimación "cuando corresponda" que pide el AC.

### Dominio Sprint (`internal/sprint/domain/sprint.go`)

- `Sprint` struct (línea 6): solo `ID, ProjectID, SprintGoal`.
- El flag `is_closed` (migración `000008`) **no está mapeado en el struct de dominio** — se consulta con SQL crudo directamente en el repositorio (`internal/story/infrastructure/postgres/repository.go:47`). Es una inconsistencia de diseño preexistente, relevante si la nueva validación necesita el estado del Sprint.

### Asociación Historia↔Sprint (US-09)

- Tabla `sprint_stories(sprint_id, story_id, project_id)` (`internal/project/infrastructure/postgres/migrations/000007_create_sprint_stories.up.sql`): PK compuesta `(sprint_id, story_id)`, FKs compuestas `(sprint_id, project_id)→sprints(id,project_id)` y `(story_id, project_id)→stories(id,project_id)`, ambas `ON DELETE RESTRICT`.
- `sprints.is_closed BOOLEAN NOT NULL DEFAULT false` (`000008_add_sprint_closed.up.sql`).
- Toda la lógica de "pertenencia" vive dentro de `PostgresStoryRepository.AssignStoriesForProject` (`internal/story/infrastructure/postgres/repository.go:38-107`): transacción, `SELECT ... FOR UPDATE` sobre el sprint, valida proyecto/cierre, valida existencia/pertenencia de cada historia, valida preexistencia de la asociación (`SELECT EXISTS(SELECT 1 FROM sprint_stories WHERE sprint_id=$1 AND story_id=ANY($2))`), inserta.
- **No existe un puerto/caso de uso de solo lectura "¿esta historia pertenece a este Sprint?"** — hay que construirlo para US-10.
- Gating de disponibilidad: `api.ResolveMigrationReadiness` (`internal/api/api.go:87-97`): `Stories>=2`, `Sprints>=5`, `Members>=6`, `Assignment>=8`. US-10 necesitará un flag nuevo (p. ej. `Tasks>=9`) coordinado entre `internal/api/api.go` y `cmd/api/main.go` — US-09 ya documentó esta doble-comprobación como fragilidad a vigilar, no como defecto a resolver acá.

### Patrón arquitectónico (ejemplo `projectmember`, replicado en `story`/`sprint`)

- 4 capas: `domain/` (entidad + `ValidationError{Fields map[string]string}`), `application/` (comando + puerto + caso de uso), `infrastructure/postgres/` (implementación del puerto), `transport/http/` (handler: decodifica JSON con `DisallowUnknownFields()` + verificación de "un solo objeto JSON" vía `io.EOF`, mapea errores con `errors.Is/As` a `{error, message, fields}` + status HTTP).
- Nombres: identificadores Go en inglés (`Story`, `Sprint`, `Member`); valores de vocabulario de negocio en español (`"alta"/"media"/"baja"`, `"pendiente"/"en_progreso"/"completada"`).
- Errores HTTP: 404 `*_not_found`, 409 `assignment_conflict`, 422 `validation_failed` (con `Fields`), 400 `invalid_request`, 405 `method_not_allowed`, 500 `internal_error`.
- Migraciones: directorio único `internal/project/infrastructure/postgres/migrations/`, prefijo secuencial de 6 dígitos; cabeza real actual = `000008`. El conflicto histórico de numeración `000003` ya documentado en `openspec/config.yaml` no debe reproducirse; sólo hay que confirmar el número más alto real al momento de aplicar.

### Tests a replicar

- Unit de aplicación: fake/mock manual implementando el puerto (`tests/unit/story/application/assign_stories_test.go`), tabla de casos, asserts sobre `calls`/args/`errors.Is`.
- Unit HTTP: `httptest.NewRequest/NewRecorder` + `SetPathValue`, tabla de casos por código de estado.
- Integración: `tests/integration/story/postgres/assignment_integration_test.go` con Testcontainers (`storyDatabase(t)`), fixtures (`insertProject`, `validStory`), aplica/revierte migraciones (`applyStoryMigration`), verifica atomicidad y violaciones de constraint (`assertDatabaseError` por código pg + nombre de constraint).

## Patrones a reutilizar

- Arquitectura de 4 capas de `projectmember`/`story` para el nuevo recurso `Task`.
- Patrón de validación de horas estimadas de `Story` (`*float64` nil = sin estimar) para el campo de estimación de la tarea.
- Patrón SQL de verificación de pertenencia usado dentro de `AssignStoriesForProject`, extraído a un caso de uso de solo lectura nuevo (no existe hoy).
- Convención de nombres: identificador Go en inglés (`Task`), vocabulario de negocio en español.
- Suite de tests: unit de aplicación (mocks), unit HTTP (tabla de casos), integración con Testcontainers.

## Preguntas abiertas / decisiones pendientes

1. Contrato HTTP exacto para "seleccionar una historia asignada a un Sprint": el AC no define el endpoint — ¿anidado bajo sprint+historia (`POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks`) o bajo historia con `sprint_id` en el body?
2. ¿Qué campos son "información suficiente para identificar el trabajo"? ¿Solo título, o título + descripción (como `Story`)?
3. ¿La tarea necesita un campo de estado propio ya en US-10, o queda completamente fuera de alcance hasta US-15 (registro de horas)?
4. ¿Hay restricción de duplicados o cantidad máxima de tareas por historia? El AC no lo menciona; US-09 sí definía reglas de duplicados para historias.

## Riesgos y observaciones

- No existe hoy mecanismo reutilizable para verificar "historia pertenece al Sprint seleccionado"; debe construirse en este cambio.
- La fragilidad de doble-umbral de gating de migraciones (dos archivos coordinados a mano) documentada en US-09 se repetirá con el nuevo flag de Tasks.
- El conflicto histórico de numeración `000003` sigue pendiente de reconciliación según `openspec/config.yaml`; no tocarlo, pero verificar el número más alto real antes de crear la nueva migración.
- `sprints.is_closed` no está en el dominio Go, solo en SQL crudo — si la validación de pertenencia necesita el estado del Sprint, deberá leerlo del mismo modo (decisión de diseño a confirmar en `sdd-design`).

## Fuente

Exploración realizada por el agente `sdd-explore` (CodeGraph + lectura de código), respaldada también en Engram bajo el topic `sdd/us-10-descomponer-historia-tareas/explore` (observación #16).
