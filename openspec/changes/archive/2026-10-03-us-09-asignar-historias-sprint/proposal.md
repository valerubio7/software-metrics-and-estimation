# Propuesta: Asignar historias a Sprint (US-09)

> **Nota:** esta propuesta formaliza retroactivamente un cambio ya implementado y fusionado en `main` mediante el PR #78 (commit de comportamiento `f544916`), conforme a la decisión del equipo de documentar todo cambio a través de SDD a partir de ahora. No se escribe ni se modifica código en este cambio; el objetivo es dejar el expediente de planificación/cierre que la entrega original omitió.

## Intención

Permitir que un cliente asigne un lote de historias existentes del Product Backlog a un Sprint abierto del mismo proyecto, sin retirarlas del backlog ni alterar su estado, estimaciones u orden. La asignación debe validarse como un todo: cualquier historia inexistente, de otro proyecto, ya asignada a ese Sprint, o un Sprint cerrado/inexistente, debe rechazar el lote completo sin persistir ningún vínculo.

## Alcance

### Incluido
- `POST /projects/{project_id}/sprints/{sprint_id}/stories` con `{"story_ids":["<uuid>",...]}` para una selección no vacía y sin IDs repetidos.
- Validación de UUIDs, de un único objeto JSON sin campos desconocidos, de existencia y pertenencia al proyecto de la ruta tanto del Sprint como de cada historia, de que el Sprint esté abierto (`is_closed=false`) y de que ninguna historia ya esté vinculada a ese Sprint.
- Persistencia atómica del lote completo (todo o nada) con protección ante condiciones de carrera (cierre concurrente, inserciones duplicadas concurrentes).
- Habilitación de la ruta únicamente cuando el esquema canónico está en versión limpia ≥8 y la dependencia de asignación está explícitamente provista.
- Preservación del Product Backlog: las historias asignadas permanecen visibles en `ListByProject`/consulta del backlog con todos sus valores intactos.

### Excluido
- Cerrar el Sprint, definir tareas o marcar historias como completadas (historias de usuario posteriores).
- Modificar el estado, las estimaciones o la pertenencia/orden de las historias en el Product Backlog.
- Retirar historias del backlog como efecto de la asignación.
- Cualquier endpoint de desasignación o reasignación.

## Capacidades

### Capacidades nuevas
- `historia`: asignación de un lote de historias existentes a un Sprint del mismo proyecto, sin retirarlas del Product Backlog.

### Capacidades modificadas
- `sprint`: se agrega la noción de Sprint cerrado (`is_closed`) como condición que bloquea nuevas asignaciones; la creación de Sprints conserva su gate de esquema limpio ≥5 sin cambios.

## Enfoque

Extender el módulo vertical `internal/story/` con un caso de uso de asignación (`AssignStoriesUseCase`) que valida selección no vacía y sin duplicados antes de delegar en un repositorio con alcance de proyecto (`ProjectScopedStorySprintAssigner`). La persistencia usa una única transacción PostgreSQL que bloquea la fila del Sprint (`SELECT ... FOR UPDATE`) para serializar frente a cierres y asignaciones concurrentes, valida la pertenencia de cada historia al proyecto, y usa integridad referencial compuesta `(id, project_id)` como defensa adicional de "mismo proyecto". La ruta solo se habilita cuando el esquema canónico alcanza la versión 8 (migraciones `000007_create_sprint_stories` y `000008_add_sprint_closed`) con la dependencia de asignación explícitamente provista en el arranque.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/story/application/assign_stories.go` | Nueva | Caso de uso de asignación, validación de lote y contratos de error. |
| `internal/story/transport/http/assign_stories_handler.go` | Nueva | Endpoint HTTP, canonicalización de UUIDs, mapeo de errores a códigos HTTP. |
| `internal/story/infrastructure/postgres/repository.go` | Modificada | Métodos `AssignStories`/`AssignStoriesForProject` con transacción, bloqueo de fila y traducción de constraints. |
| `internal/project/infrastructure/postgres/migrations/000007_create_sprint_stories.{up,down}.sql`, `000008_add_sprint_closed.{up,down}.sql` | Nueva | Esquema de asociación Sprint-historia y bandera de cierre de Sprint. |
| `internal/api/api.go`, `cmd/api/main.go` | Modificada | Gating de disponibilidad de la ruta según versión de esquema y dependencia explícita. |
| `openspec/specs/historia/spec.md`, `openspec/specs/sprint/spec.md` | Ya fusionada | Requisitos canónicos del comportamiento, incorporados directamente por el commit de implementación sin pasar por este flujo. |
| `tests/unit/story/...`, `tests/integration/story/postgres/assignment_integration_test.go` | Nueva | Cobertura unitaria y de integración PostgreSQL real. |
| `README.md` | Modificada | Documentación del endpoint, contrato y requisito de migración. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Dos comprobaciones de versión de esquema separadas (`dependencies.Stories` en `cmd/api/main.go`, no nulo ya desde `>=2`, y `readiness.Assignment` en `internal/api/api.go` con `>=8`) podrían desincronizarse en una migración futura. | Baja hoy (`>=8` implica `>=2` con margen de sobra), pero queda como fragilidad documentada. | Mantener ambas comprobaciones en revisión conjunta ante cualquier cambio de umbral de migración. |
| Condiciones de carrera entre asignación y cierre de Sprint. | Media | Bloqueo de fila `FOR UPDATE` dentro de la misma transacción que valida cierre y persiste el lote; verificado por `TestAssignStoriesWaitsForConcurrentSprintClose`. |
| Inserciones duplicadas concurrentes del mismo lote. | Media | Manejo explícito de violación de clave primaria (`23505`) como red de seguridad adicional a la comprobación `EXISTS` previa; verificado por `TestAssignStoriesConcurrentDuplicateHasSingleWinner`. |

## Plan de reversión

Revertir en conjunto el caso de uso, el handler, los cambios de repositorio y su composición/gating. Si las migraciones `000007`/`000008` ya se aplicaron, no ejecutar su `down` automáticamente: elimina asociaciones y la bandera de cierre, no las tablas base de historias/Sprints/integrantes; su seguridad en producción no está probada y requiere decisión explícita de despliegue.

## Dependencias

- Issue #37 (GitHub), ya cerrada: define el alcance aprobado de asignación de un lote de historias a un Sprint existente del mismo proyecto.
- Requiere que el esquema canónico esté en versión limpia ≥8 (migraciones `000007` y `000008` aplicadas) antes de habilitar la ruta.
- No depende de cerrar el Sprint ni de ninguna historia de usuario posterior.

## Criterios de éxito

- [x] Una selección válida persiste íntegramente todos los vínculos solicitados y responde `201` con `sprint_id` y `story_ids`.
- [x] Cualquier historia inexistente, de otro proyecto, ya asignada, o un Sprint cerrado/inexistente/de otro proyecto, rechaza el lote completo sin persistir ningún vínculo.
- [x] Las historias asignadas permanecen visibles en el Product Backlog con todos sus valores intactos.
- [x] La ruta solo está disponible con esquema canónico limpio ≥8 y dependencia de asignación explícita.
- [x] Asignaciones y cierres concurrentes no producen escrituras parciales ni duplicadas.
