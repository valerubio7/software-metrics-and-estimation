# Especificación: consulta de tareas de una historia

## ADDED Requirements

### Requirement: Consultar las tareas de una historia
El sistema MUST ofrecer una operación de solo lectura, `GET /projects/{project_id}/stories/{story_id}/tasks`, que reciba el identificador de un proyecto y el de una historia de ese proyecto. Ante una consulta válida, MUST responder `200` con un contenedor `{"project_id": "...", "story_id": "...", "tasks": [...]}`, donde ambos identificadores son los consultados y `tasks` contiene todas las tareas de esa historia y únicamente las de esa historia. Cada tarea MUST incluir `id`, `project_id`, `sprint_id`, `story_id`, `title` y `estimated_hours`; `estimated_hours` MUST ser `null` cuando la tarea no tiene estimación. La lista MUST ordenarse por la secuencia de creación ascendente (`seq`), de modo que la tarea creada antes aparezca antes, y el orden MUST ser determinista. Una historia sin tareas MUST responder `200` con `tasks` igual a `[]`, nunca `null`. La operación MUST NOT modificar datos.

#### Scenario: Consultar una historia con tareas
- GIVEN un proyecto existente con una historia que tiene dos tareas creadas
- WHEN se solicita `GET /projects/{project_id}/stories/{story_id}/tasks`
- THEN el sistema responde `200`
- AND el cuerpo contiene `project_id` y `story_id` iguales a los consultados
- AND `tasks` contiene exactamente dos elementos con `id`, `project_id`, `sprint_id`, `story_id`, `title` y `estimated_hours`

#### Scenario: Historia sin tareas
- GIVEN un proyecto existente con una historia sin tareas
- WHEN se solicita la colección de tareas de esa historia
- THEN el sistema responde `200`
- AND `tasks` es una lista vacía `[]`, no `null`

#### Scenario: Orden por secuencia de creación
- GIVEN una historia con tres tareas creadas en un orden conocido
- WHEN se consulta la colección
- THEN las tareas aparecen en el mismo orden en que fueron creadas
- AND dos consultas consecutivas sin cambios devuelven el mismo orden

#### Scenario: Tarea sin estimación
- GIVEN una historia con una tarea creada sin `estimated_hours`
- WHEN se consulta la colección
- THEN esa tarea incluye la clave `estimated_hours` con valor `null`

#### Scenario: Proyecto inexistente
- GIVEN un `project_id` con formato válido que no corresponde a ningún proyecto existente
- WHEN se solicita consultar las tareas de una historia
- THEN el sistema responde `404` con el código `project_not_found`
- AND no modifica datos

#### Scenario: Historia inexistente o de otro proyecto
- GIVEN un proyecto existente y un `story_id` que no corresponde a ninguna historia, o que corresponde a una historia de otro proyecto
- WHEN se solicita consultar sus tareas bajo el proyecto indicado
- THEN el sistema responde `404` con el código `story_not_found`
- AND no modifica datos

#### Scenario: Identificadores inválidos
- GIVEN un `project_id` o un `story_id` que no es un UUID válido
- WHEN se solicita consultar las tareas
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica el identificador inválido
- AND no modifica datos

#### Scenario: Método no admitido
- GIVEN la ruta `/projects/{project_id}/stories/{story_id}/tasks`
- WHEN se solicita con un método distinto de `GET` o `HEAD`
- THEN el sistema responde `405 method_not_allowed`
- AND no modifica datos

#### Scenario: Error inesperado del almacenamiento
- GIVEN un fallo inesperado al leer las tareas
- WHEN se procesa la consulta
- THEN el sistema responde `500 internal_error` con un mensaje genérico
- AND no divulga detalles internos
- AND no modifica datos
