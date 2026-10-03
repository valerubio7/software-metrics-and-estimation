# Delta para Historia

> Este delta documenta, de forma retroactiva, requisitos que ya fueron fusionados directamente en `openspec/specs/historia/spec.md` por el commit `f544916` (PR #78), sin pasar por la fase `sdd-spec`. El texto se copia textual del contenido canónico vigente (líneas 843–889), no se parafrasea.

## ADDED Requirements

### Requirement: Asignar un lote de historias sin retirarlas del Product Backlog (US-09)

El sistema MUST ofrecer `POST /projects/{project_id}/sprints/{sprint_id}/stories` con `{"story_ids":["<uuid>"]}` para una selección no vacía y sin IDs repetidos. MUST validar UUIDs y un único objeto JSON sin campos desconocidos. Una selección válida MUST persistirse íntegramente y responder `201` con `sprint_id` y `story_ids`. Todas las historias y el Sprint MUST existir y pertenecer al proyecto de la ruta; ninguna historia MUST estar ya asociada a ese mismo Sprint, y el Sprint MUST estar abierto. La asociación MUST NOT modificar historias, su estado, estimaciones ni pertenencia/orden en el Product Backlog.

#### Scenario: Consultar el backlog después de asignar varias historias

- GIVEN un proyecto con dos historias y un Sprint abierto del mismo proyecto
- WHEN se asignan ambas historias en una solicitud válida
- THEN se persisten ambos vínculos y la respuesta es `201`
- AND `ListByProject` y la consulta del Product Backlog siguen devolviendo ambas historias con todos sus valores intactos

#### Scenario: Rechazar un lote completo

- GIVEN una selección con una historia inexistente, de otro proyecto o ya vinculada al Sprint
- WHEN se intenta asignar el lote
- THEN no se agrega ningún vínculo de esa solicitud
- AND Sprint/historia inexistente responde `404 resource_not_found`; incompatibilidad de proyecto o asociación previa responde `409 assignment_conflict`
- AND selección vacía o repetida responde `422 validation_failed`, JSON malformado/campos desconocidos `400 invalid_request`, y un fallo inesperado `500 internal_error` sin detalles internos

#### Scenario: Proyecto de ruta incorrecto

- GIVEN un Sprint existente y un proyecto de ruta distinto, incluido un UUID de proyecto inexistente
- WHEN se intenta asignar historias al Sprint desde esa ruta
- THEN se responde `409 assignment_conflict` sin agregar vínculos

#### Scenario: Asignaciones concurrentes y error de inserción

- GIVEN dos solicitudes concurrentes con las mismas historias para el mismo Sprint
- WHEN compiten por la asignación
- THEN una gana y la otra rechaza el lote sin duplicados ni inserciones parciales
- AND un fallo en cualquier inserción revierte todas las inserciones de su lote

### Requirement: Habilitar asignación solo con esquema completo y dependencia explícita

El arranque MUST habilitar asignación únicamente con versión canónica limpia >=8 y dependencia de asignación explícita. MUST preservar gates de historias, creación de Sprints >=5 e integrantes >=6. `000007` agrega `sprint_stories` con unicidad y FKs compuestas por proyecto; `000008` agrega `sprints.is_closed`. Esta compatibilidad cubre instalación nueva y upgrade de main canónico v6, no despliegues desconocidos de v5/v6 alternativos.

#### Scenario: Esquema incompleto o no verificable

- GIVEN versión v6/v7, estado `dirty`, error al leer migraciones o dependencia ausente
- WHEN arranca la API y se solicita la asignación
- THEN la ruta no existe y responde `404` sin escrituras

#### Scenario: Esquema completo

- GIVEN versión v8 o futura limpia y dependencia explícita
- WHEN arranca la API
- THEN la ruta de asignación queda disponible junto con las rutas previas
