# Especificación de Tarea

## Propósito

Permitir que un integrante del equipo descomponga una historia ya asignada a un Sprint en una o más tareas, con título obligatorio y una estimación de esfuerzo en horas opcional. La restricción central es que no se pueden crear tareas para una historia que no pertenezca al Sprint seleccionado. Las tareas creadas quedan persistidas y asociadas a su historia, a su Sprint y a su proyecto, sin campo de estado propio en este alcance.

## Requirements

### Requirement: Crear un lote de tareas para una historia asignada a un Sprint

El sistema MUST ofrecer `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks` para crear una o más tareas asociadas a la historia identificada en la ruta, siempre que esa historia esté asignada al Sprint indicado. El cuerpo MUST contener la clave `tasks` con una lista no vacía de objetos, cada uno con `title` obligatorio y `estimated_hours` opcional. Ante una solicitud válida, el sistema MUST persistir exactamente una tarea por cada elemento del lote, cada una asociada a esa historia, a ese Sprint y a ese proyecto, y MUST responder `201` con un contenedor `{"tasks": [...]}` donde cada elemento incluye el identificador generado por el servidor, `project_id`, `sprint_id`, `story_id`, `title` y `estimated_hours`.

#### Scenario: Crear una tarea para una historia asignada al Sprint

- GIVEN un proyecto existente, un Sprint de ese proyecto y una historia de ese proyecto asignada a ese Sprint
- AND una solicitud con `tasks` igual a `[{"title": "Implementar validación", "estimated_hours": 4.5}]`
- WHEN se solicita crear las tareas para esa historia dentro de ese Sprint
- THEN el sistema responde `201`
- AND persiste exactamente una tarea asociada a esa historia, a ese Sprint y a ese proyecto, con el título y la estimación enviados
- AND el cuerpo de la respuesta incluye el identificador generado para la tarea creada

#### Scenario: Crear varias tareas en un mismo lote

- GIVEN un proyecto existente, un Sprint de ese proyecto y una historia asignada a ese Sprint
- AND una solicitud con `tasks` igual a tres objetos válidos, algunos con `estimated_hours` y otros sin ella
- WHEN se solicita crear el lote
- THEN el sistema responde `201`
- AND persiste exactamente tres tareas, todas asociadas a la misma historia, Sprint y proyecto
- AND cada tarea devuelta conserva el título y la estimación (o su ausencia) que le correspondía en el lote enviado

### Requirement: Tratar el lote de tareas como una operación atómica

El sistema MUST tratar la creación del lote como todo-o-nada. Si cualquier tarea del lote es inválida, si la historia no pertenece al Sprint indicado, si el proyecto, el Sprint o la historia no existen, o si la persistencia falla por cualquier motivo, incluido un error inesperado del almacenamiento, el sistema MUST NOT persistir ninguna tarea de ese lote.

#### Scenario: Ninguna tarea persiste si una tarea del lote es inválida

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una solicitud con `tasks` igual a dos objetos, el primero válido y el segundo con título vacío
- WHEN se solicita crear el lote
- THEN el sistema rechaza la solicitud
- AND no persiste ninguna de las dos tareas, incluida la primera que era válida por sí sola

#### Scenario: Ninguna tarea persiste ante un fallo inesperado del almacenamiento

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí, y un lote de tareas válido
- AND un fallo inesperado del almacenamiento durante la escritura del lote
- WHEN se solicita crear el lote
- THEN el sistema responde `500 internal_error` con un mensaje genérico
- AND no queda persistida ninguna tarea de ese lote

### Requirement: Validar el título y la estimación de horas de cada tarea del lote

El sistema MUST exigir que cada tarea del lote tenga un `title` no vacío ni compuesto solo por espacios tras recortarlos. El sistema MAY recibir `estimated_hours` ausente o `null`, lo que indica que la tarea no tiene estimación. Cuando `estimated_hours` esté presente y sea un número, el sistema MUST rechazar valores que no sean estrictamente mayores que `0` (incluido `0`), valores con más de dos decimales y valores mayores que `99999.99` — las mismas reglas que `Story.EstimatedHours` (`internal/story/domain/story.go`). Toda violación de estas reglas MUST rechazarse con `422 validation_failed`, indicando en `fields` la tarea y el campo inválidos dentro del lote, y MUST NOT persistir ninguna tarea del lote.

#### Scenario: Rechazar un título vacío o faltante

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una tarea del lote con `title` ausente, `""` o `"   "`
- WHEN se solicita crear el lote
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica la tarea y el campo `title` como inválidos
- AND no se persiste ninguna tarea del lote

#### Scenario: Rechazar una estimación de horas negativa o igual a cero

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una tarea del lote con `estimated_hours` igual a `-1`, `-0.01` o `0`
- WHEN se solicita crear el lote
- THEN el sistema responde `422 validation_failed` con `fields` indicando la tarea y `estimated_hours`
- AND no se persiste ninguna tarea del lote

#### Scenario: Rechazar una estimación con más de dos decimales

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una tarea del lote con `estimated_hours` igual a `1.001` o `0.005`
- WHEN se solicita crear el lote
- THEN el sistema responde `422 validation_failed` con `fields` indicando la tarea y `estimated_hours`
- AND no se persiste ninguna tarea del lote

#### Scenario: Rechazar una estimación mayor que el máximo permitido

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una tarea del lote con `estimated_hours` igual a `100000` o `99999.991`
- WHEN se solicita crear el lote
- THEN el sistema responde `422 validation_failed` con `fields` indicando la tarea y `estimated_hours`
- AND no se persiste ninguna tarea del lote

#### Scenario: Aceptar una tarea sin estimación de horas

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una tarea del lote con `estimated_hours` ausente o `null`, y título válido
- WHEN se solicita crear el lote
- THEN el sistema responde `201`
- AND la tarea persistida y devuelta tiene `estimated_hours` `null`

### Requirement: Rechazar solicitudes malformadas sin persistir tareas

El sistema MUST responder `400 invalid_request` cuando el cuerpo no sea un único objeto JSON válido, cuando falte la clave `tasks`, cuando `tasks` sea una lista vacía, cuando el cuerpo o alguna tarea del lote contenga una clave desconocida, o cuando algún campo tenga un tipo incorrecto (por ejemplo, `title` numérico o `estimated_hours` como cadena). El sistema MUST NOT persistir ninguna tarea del lote en estos casos.

#### Scenario: JSON malformado

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND un cuerpo que no es JSON válido, por ejemplo `{"tasks":`
- WHEN se solicita crear el lote
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ninguna tarea

#### Scenario: Lote vacío

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una solicitud con `tasks` igual a `[]`
- WHEN se solicita crear el lote
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ninguna tarea

#### Scenario: Campo desconocido

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una solicitud válida que además incluye una clave desconocida en el cuerpo o en alguna tarea del lote
- WHEN se solicita crear el lote
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ninguna tarea

#### Scenario: Tipo incorrecto en un campo

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una tarea del lote con `title` numérico o con `estimated_hours` igual a la cadena `"8"`
- WHEN se solicita crear el lote
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ninguna tarea

### Requirement: Rechazar un proyecto, Sprint o historia inexistente o de otro proyecto

El sistema MUST responder `404` con el código `project_not_found` cuando el `project_id` de la ruta no corresponda a ningún proyecto existente. MUST responder `404` con el código `sprint_not_found` cuando el `sprint_id` no corresponda a ningún Sprint existente o corresponda a un Sprint de otro proyecto. MUST responder `404` con el código `story_not_found` cuando el `story_id` no corresponda a ninguna historia existente o corresponda a una historia de otro proyecto. En cualquiera de estos casos el sistema MUST NOT persistir ninguna tarea del lote.

#### Scenario: Proyecto inexistente

- GIVEN un `project_id` con formato válido que no corresponde a ningún proyecto existente
- WHEN se solicita crear un lote de tareas usando ese `project_id`
- THEN el sistema responde `404` con el código `project_not_found`
- AND no se persiste ninguna tarea

#### Scenario: Sprint inexistente o de otro proyecto

- GIVEN un proyecto existente y un `sprint_id` que no corresponde a ningún Sprint existente, o que corresponde a un Sprint de otro proyecto
- WHEN se solicita crear un lote de tareas usando ese `sprint_id` bajo la ruta de ese proyecto
- THEN el sistema responde `404` con el código `sprint_not_found`
- AND no se persiste ninguna tarea

#### Scenario: Historia inexistente o de otro proyecto

- GIVEN un proyecto existente, un Sprint de ese proyecto y un `story_id` que no corresponde a ninguna historia existente, o que corresponde a una historia de otro proyecto
- WHEN se solicita crear un lote de tareas usando ese `story_id` bajo esa ruta
- THEN el sistema responde `404` con el código `story_not_found`
- AND no se persiste ninguna tarea

### Requirement: Rechazar una historia que no está asignada al Sprint indicado

El sistema MUST responder `409` con el código `story_not_in_sprint` cuando la historia identificada en la ruta exista en el proyecto indicado pero no esté asociada a ese Sprint. El sistema MUST NOT persistir ninguna tarea del lote en ese caso. Esta verificación de pertenencia MUST resolverse dentro de la misma operación que crea las tareas, sin depender de una comprobación previa separada.

#### Scenario: Historia del proyecto pero no asignada a ese Sprint

- GIVEN un proyecto existente, un Sprint de ese proyecto y una historia del mismo proyecto que no está asociada a ese Sprint
- WHEN se solicita crear un lote de tareas válido para esa historia dentro de ese Sprint
- THEN el sistema responde `409` con el código `story_not_in_sprint`
- AND no se persiste ninguna tarea

#### Scenario: Historia asignada a otro Sprint del mismo proyecto

- GIVEN un proyecto existente con dos Sprints, A y B, y una historia asignada únicamente a A
- WHEN se solicita crear un lote de tareas para esa historia usando el Sprint B
- THEN el sistema responde `409` con el código `story_not_in_sprint`
- AND no se persiste ninguna tarea

### Requirement: Permitir crear tareas aunque el Sprint esté cerrado

El sistema MUST permitir la creación de tareas para una historia asignada a un Sprint cuyo `is_closed` sea `true`, sin aplicar ninguna restricción adicional por ese estado. El cierre del Sprint MUST NOT impedir ni condicionar esta operación.

#### Scenario: Un Sprint cerrado no impide crear tareas

- GIVEN un proyecto existente, un Sprint de ese proyecto con `is_closed=true` y una historia de ese proyecto asignada a ese Sprint
- WHEN se solicita crear un lote de tareas válido para esa historia
- THEN el sistema responde `201`
- AND persiste las tareas del lote, igual que si el Sprint estuviera abierto

### Requirement: Las tareas creadas no tienen campo de estado

El sistema MUST NOT incluir un campo de estado en la tarea creada, ni interpretar ningún dato de estado para la tarea. Un campo de estado para la tarea queda fuera de este alcance y se difiere a una operación futura (US-15). Si el cuerpo de la solicitud incluye una clave de estado para alguna tarea, el sistema MUST rechazarla como campo desconocido con `400 invalid_request`.

#### Scenario: La tarea creada no expone un campo de estado

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí, y un lote de tareas válido
- WHEN se crea el lote
- THEN cada tarea devuelta y cada tarea almacenada carece de cualquier clave de estado

#### Scenario: Rechazar un campo de estado en la solicitud

- GIVEN un proyecto, Sprint e historia válidos y asignados entre sí
- AND una tarea del lote que además incluye una clave de estado, con cualquier valor
- WHEN se solicita crear el lote
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ninguna tarea del lote

### Requirement: Habilitar la ruta de creación de tareas solo con el esquema en versión 9 o superior

El sistema MUST exponer la ruta de creación de tareas únicamente cuando el esquema de almacenamiento esté en versión 9 o superior y sin estado inconsistente (`dirty`), mediante un flag `Tasks` nuevo en la resolución de disponibilidad de migraciones. Con el esquema en una versión inferior a 9, o en estado `dirty`, la ruta MUST NOT existir y una solicitud sobre ella MUST responder `404`. Esta condición MUST NOT alterar la disponibilidad de las rutas existentes de proyecto, historia, Sprint, integrantes ni asignación de historias a Sprint.

#### Scenario: La ruta está disponible con el esquema en versión 9

- GIVEN el esquema de almacenamiento en versión 9 y sin estado inconsistente
- WHEN el sistema arranca y se solicita crear un lote de tareas válido
- THEN el sistema responde `201`

#### Scenario: La ruta no existe con el esquema en versión 8

- GIVEN el esquema de almacenamiento en versión 8 y sin estado inconsistente
- WHEN se solicita `POST` sobre la ruta de creación de tareas
- THEN el sistema responde `404` porque la ruta no está registrada
- AND no se persiste ninguna tarea

#### Scenario: El esquema inconsistente deja la ruta no disponible

- GIVEN el esquema de almacenamiento en versión 9 o superior con estado inconsistente (`dirty`)
- WHEN el sistema arranca
- THEN la ruta de creación de tareas no queda expuesta

#### Scenario: Las rutas existentes no cambian

- GIVEN el esquema de almacenamiento en versión 9 y sin estado inconsistente
- WHEN se solicitan las operaciones existentes de proyecto, historia, Sprint, integrantes y asignación de historias a Sprint
- THEN todas responden igual que antes de habilitar la ruta de tareas
