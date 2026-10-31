# Delta para Esfuerzo: operación desde la TUI (TUI-08)

Este cambio no modifica el contrato HTTP de esfuerzo: no agrega rutas, no cambia el cuerpo de la solicitud ni los códigos de respuesta, y no agrega ninguna consulta de registros. Solo cambia quién puede consumir el contrato: la TUI deja de estar excluida.

## Cambio en el Propósito (no es un Requirement; aplicar manualmente a la sección «Propósito» al archivar)

Reemplazar la oración final de la sección «Propósito» que dice «Esta capacidad solo registra: no calcula totales ni desvíos (US-19), no crea tareas (US-10), no estima puntos de historia (US-14), no ofrece listado ni consulta por HTTP y no se opera desde la TUI (TUI-08).» por:

«Esta capacidad solo registra: no calcula totales ni desvíos (US-19), no crea tareas (US-10), no estima puntos de historia (US-14) y no ofrece listado ni consulta por HTTP. El registro puede operarse desde la TUI (TUI-08), que consume este mismo contrato HTTP sin ampliarlo.»

(Previously: el Propósito afirmaba que la capacidad «no se opera desde la TUI (TUI-08)».) El resto del Propósito, incluida la oración sobre el estado de la tarea diferido en `tarea`, no cambia.

## ADDED Requirements

### Requirement: Permitir operar el registro desde la TUI sin ampliar el contrato HTTP

El sistema MUST seguir ofreciendo `POST /projects/{project_id}/stories/{story_id}/effort-records` como único medio HTTP de esta capacidad, y la TUI MUST consumirlo sin cambios: con las mismas claves de cuerpo (`member_id`, `work_date`, `description`, `hours` y `task_id` opcional) y con las mismas respuestas. El sistema MUST NOT agregar rutas de esfuerzo ni una consulta o listado de registros para la TUI, y la TUI MUST NOT consultar registros de esfuerzo por HTTP.

#### Scenario: La TUI registra con el contrato existente

- GIVEN un proyecto, una historia y un integrante existentes y una TUI con los datos del formulario completos y válidos
- WHEN el usuario confirma el registro desde la TUI
- THEN la API recibe un único `POST /projects/{project_id}/stories/{story_id}/effort-records`
- AND el cuerpo contiene solo claves del contrato (`member_id`, `work_date`, `description`, `hours` y, si se eligió una tarea, `task_id`)
- AND la API responde `201` con el registro creado según el requisito «Registrar esfuerzo sobre una historia» o «Registrar esfuerzo sobre una tarea de la historia»

#### Scenario: Registrar sobre la historia completa desde la TUI

- GIVEN que el usuario eligió registrar sobre la historia completa
- WHEN la TUI envía el registro
- THEN el cuerpo no contiene la clave `task_id`
- AND el registro persistido tiene `task_id` nulo, como en el requisito «Registrar esfuerzo sobre una historia»

#### Scenario: La TUI no consulta registros de esfuerzo

- GIVEN un recorrido completo de Esfuerzo en la TUI, desde la elección de la historia hasta la pantalla de resultado
- WHEN se revisan las requests emitidas
- THEN ninguna es un `GET` a una ruta de registros de esfuerzo
- AND las consultas de lectura son las de historias, tareas, integrantes y métricas del proyecto
