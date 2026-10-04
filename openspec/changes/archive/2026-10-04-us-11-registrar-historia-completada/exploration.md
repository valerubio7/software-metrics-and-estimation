# Exploración: US-11 — Registrar una historia del Sprint como completada

## Resumen del problema

US-11 (issue #39) requiere que un integrante del equipo registre una historia perteneciente a un Sprint como completada. Debe conservar la asociación con ese Sprint, no contarla dos veces, rechazar historias ajenas al Sprint y dejar el dato disponible para métricas posteriores (US-19: Story Points completados y porcentaje de historias completadas).

Hoy no existe ningún mecanismo que registre la finalización dentro de un Sprint. La pertenencia vive en `sprint_stories` (US-09). El único "estado" es `stories.status`, que es global a la historia y se edita con `PUT /projects/{project_id}/stories/{story_id}` sin contexto de Sprint.

## Hallazgos concretos

### Asociación Historia↔Sprint (US-09)

- `sprint_stories(sprint_id, story_id, project_id)` (migración `000007`): PK `(sprint_id, story_id)` y FKs compuestas `sprint_stories_sprint_project_fkey` y `sprint_stories_story_project_fkey`, ambas `ON DELETE RESTRICT`. No tiene columnas de estado ni de fecha.
- No hay UNIQUE sobre `story_id`: una misma historia puede estar en varios Sprints, así que "completada" es un hecho por Sprint, no global.
- `sprints.is_closed BOOLEAN NOT NULL DEFAULT false` (migración `000008`). No está en el struct de dominio `Sprint`; se lee con SQL crudo.
- `PostgresStoryRepository.AssignStoriesForProject` (`internal/story/infrastructure/postgres/repository.go`) es el patrón a seguir: transacción, `SELECT ... FOR UPDATE` del Sprint, `ErrProjectMismatch`, `ErrSprintClosed`, verificación de historias y errores mapeados por código/constraint de pgx.

### Estado de la historia (`internal/story/domain/story.go`)

- `StatusPending`, `StatusInProgress` y `StatusCompleted` ("completada"). No hay reglas de transición: cualquier estado puede pasar a cualquier otro vía `PUT`.
- Si US-11 solo escribiera `stories.status`: no habría vínculo con el Sprint (rompe el criterio 3), una historia en dos Sprints quedaría "completada" en ambos y un `PUT` posterior podría revertirla y alterar métricas (rompe los criterios 5 y 6).

### Referencia US-10 (la más reciente)

- Módulo `internal/task/{domain,application,infrastructure/postgres,transport/http}`.
- `CreateForSprintStory` verifica en orden proyecto, Sprint, historia y pertenencia (`SELECT 1 FROM sprint_stories ... FOR KEY SHARE`).
- La última migración es `000009_create_tasks`; la próxima sería `000010`. El directorio es `internal/project/infrastructure/postgres/migrations/`.
- Cableado en `internal/api/api.go`: dependencias opcionales, `MigrationReadiness` / `ResolveMigrationReadiness` (`Stories>=2, Sprints>=5, Members>=6, Assignment>=8, Tasks>=9`) y registro condicional de rutas. En `cmd/api/main.go` se arman las dependencias según `readiness`.
- Handler de US-09: UUIDs de ruta inválidos → `422 validation_failed`; `404 resource_not_found`; `409 assignment_conflict` (cerrado, mismatch, ya asignada); `500 internal_error`.
- Tests: `tests/unit/{story,task}/...` (fakes manuales, tabla de casos), `tests/integration/{story,task}/postgres/*` (Testcontainers, requiere Docker) y `tests/integration/migrations/migration_files_test.go`, que deberá actualizarse.
- Documentación tocada por US-10: `README.md`, `openspec/config.yaml` (lista de historias implementadas) y `docs/scrum/sprint-1.md`. Verificar en diseño `docs/traceability.md` y `docs/architecture`.

## Enfoques comparados

| Enfoque | Descripción | Pros | Contras | Esfuerzo |
|---|---|---|---|---|
| A. `completed_at TIMESTAMPTZ NULL` en `sprint_stories` (recomendado) | Migración `000010`; `UPDATE ... WHERE completed_at IS NULL` en una transacción que bloquea el Sprint | Asocia por diseño historia completada y Sprint; la pertenencia se verifica por la propia fila; no hay doble conteo; las métricas futuras son un `COUNT`/`SUM` con JOIN a `stories.story_points`; no depende de `stories.status` | Requiere migración y gate de readiness; `stories.status` puede divergir | Medio |
| B. Solo `stories.status` | Reutilizar el campo existente | Sin migración | Sin asociación al Sprint, global entre Sprints, reversible con `PUT` | Bajo (no cumple criterios 3 y 5) |
| C. Columna + sincronizar `stories.status` | Como A, más actualizar `stories.status` en la misma transacción | Coherencia visible en el listado | Acopla con la edición libre de estados; semántica ambigua con varios Sprints; amplía alcance | Medio-alto |
| D. Tabla nueva `sprint_story_completions` | PK `(sprint_id, story_id)` con FK a `sprint_stories` | Aísla el evento | Más estructura de la necesaria para un hecho 1:1; JOIN extra | Medio |

## Recomendación

Enfoque A. NULL significa no completada. `completed_at` aporta además el instante. La decisión final se confirma en `sdd-design`.

- **Endpoint candidato:** `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion`, cuerpo vacío.
- **Módulo:** `internal/story`, donde ya vive el agregado `sprint_stories`, con un puerto y un caso de uso de registro de finalización.
- **Repositorio:** transacción que bloquea el Sprint (`FOR UPDATE`), valida proyecto, verifica la fila de `sprint_stories` (`FOR UPDATE`), responde `ErrStoryNotInSprint` si no existe, trata el caso ya completada y actualiza.
- **Idempotencia:** el lock de fila más `completed_at IS NULL` evita el doble conteo bajo concurrencia.
- **Métricas:** `sprint_stories.completed_at` permite a US-19 calcular Story Points completados y porcentaje sin cambios de esquema. US-11 no implementa consultas de métricas.
- **Gating:** nuevo flag `Completion` en `MigrationReadiness` (`version >= 10`), con dependencia opcional como `Tasks`.
- **Reversión:** `000010.down` elimina la columna con pérdida de los registros de finalización; documentarlo.

## Preguntas abiertas

1. Contrato HTTP exacto (verbo, ruta y cuerpo).
2. Registrar una historia ya completada: ¿respuesta idempotente o conflicto 409?
3. ¿Se permite registrar en un Sprint cerrado? US-09 lo rechaza; por coherencia se propone rechazar con 409.
4. ¿Debe sincronizarse `stories.status`? Recomendación: no.
5. ¿Se puede deshacer? Fuera de alcance.
6. ¿Se registra quién la completó? El criterio no lo pide.
7. ¿Se exigen tareas completas (US-10)? El criterio no lo menciona; se asume que no.

## Riesgos

- `stories.status` puede divergir de `completed_at`; las métricas deben leer solo la segunda.
- El doble umbral de migraciones (`api.go` y `cmd/api/main.go`) se repite con un nuevo flag.
- Los tests de integración requieren Docker.
- El orden de locks (Sprint antes que `sprint_stories`) debe ser compatible con el futuro cierre de Sprint (US-12).
- El presupuesto de 400 líneas es ajustado pero viable con un único endpoint, sin lecturas ni métricas.
- Hay que actualizar `migration_files_test.go` y los tests de `ResolveMigrationReadiness`.
