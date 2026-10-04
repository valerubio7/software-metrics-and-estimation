# Delta para Historia

Delta de la capacidad `historia` para US-11 (issue #39): registrar como completada una historia asignada a un Sprint. Los requisitos existentes de `historia`, `sprint`, `tarea` y `project` no cambian; todo lo siguiente es ADDED. La fuente de verdad de la finalización por Sprint es `sprint_stories.completed_at`; el campo `stories.status` es independiente.

## ADDED Requirements

### Requirement: Registrar una historia asignada a un Sprint como completada (US-11)

El sistema MUST ofrecer `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion`, sin cuerpo, para registrar como completada la historia identificada en la ruta dentro del Sprint indicado, siempre que el proyecto exista, el Sprint y la historia pertenezcan a ese proyecto, la historia esté asignada a ese Sprint, el Sprint no esté cerrado y la historia no haya sido registrada antes como completada en ese Sprint. Ante una solicitud válida, el sistema MUST persistir el instante de finalización (`completed_at`) en la asociación historia-Sprint de ese Sprint y MUST responder `200` con un cuerpo que incluya `project_id`, `sprint_id`, `story_id` y `completed_at`, donde `completed_at` es el mismo valor persistido. El sistema MUST NOT exigir ni interpretar un cuerpo de solicitud, y MUST ignorar cualquier contenido enviado en él.

#### Scenario: Registrar una historia asignada como completada

- GIVEN un proyecto existente, un Sprint abierto de ese proyecto y una historia de ese proyecto asignada a ese Sprint, sin finalización registrada
- WHEN se solicita `POST` sobre la ruta de finalización de esa historia en ese Sprint, sin cuerpo
- THEN el sistema responde `200`
- AND el cuerpo incluye `project_id`, `sprint_id`, `story_id` y un `completed_at` con un instante válido
- AND la asociación de esa historia con ese Sprint queda persistida con ese mismo `completed_at`

#### Scenario: La finalización queda ligada al Sprint indicado

- GIVEN un proyecto con dos Sprints abiertos, A y B, y una historia asignada a ambos
- WHEN se registra la historia como completada en el Sprint A
- THEN la asociación de la historia con A tiene `completed_at` definido
- AND la asociación de la historia con B conserva `completed_at` sin definir (`NULL`)

#### Scenario: El instante de finalización queda disponible para métricas

- GIVEN un Sprint con tres historias asignadas, de 3, 5 y 8 Story Points, y dos de ellas registradas como completadas
- WHEN se consultan las asociaciones del Sprint directamente en el almacenamiento
- THEN exactamente dos asociaciones tienen `completed_at` definido
- AND con ese dato se puede obtener la suma de Story Points completados y el porcentaje de historias completadas del Sprint sin otro cambio de esquema

#### Scenario: El cuerpo de la solicitud no se interpreta

- GIVEN un proyecto, un Sprint abierto y una historia asignada, sin finalización registrada
- WHEN se solicita `POST` sobre la ruta de finalización con un cuerpo no vacío (por ejemplo, `{"x":1}`)
- THEN el sistema responde `200` igual que sin cuerpo
- AND la finalización queda registrada una sola vez

### Requirement: Rechazar identificadores de ruta inválidos con detalle de campos

El sistema MUST responder `422` con el código `validation_failed` cuando `project_id`, `sprint_id` o `story_id` de la ruta de finalización no sean UUID válidos. La respuesta MUST incluir `fields` indicando cada identificador inválido. El sistema MUST NOT modificar ninguna asociación historia-Sprint en este caso.

#### Scenario: `project_id` inválido

- GIVEN un `project_id` que no es un UUID, un `sprint_id` y un `story_id` con formato válido
- WHEN se solicita registrar la finalización
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `project_id` como inválido
- AND no se modifica ninguna asociación historia-Sprint

#### Scenario: `sprint_id` inválido

- GIVEN un `sprint_id` que no es un UUID, un `project_id` y un `story_id` con formato válido
- WHEN se solicita registrar la finalización
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `sprint_id` como inválido
- AND no se modifica ninguna asociación historia-Sprint

#### Scenario: `story_id` inválido

- GIVEN un `story_id` que no es un UUID, un `project_id` y un `sprint_id` con formato válido
- WHEN se solicita registrar la finalización
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `story_id` como inválido
- AND no se modifica ninguna asociación historia-Sprint

#### Scenario: Varios identificadores inválidos a la vez

- GIVEN `project_id`, `sprint_id` y `story_id` que no son UUID
- WHEN se solicita registrar la finalización
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica los tres identificadores como inválidos

### Requirement: Rechazar un proyecto, Sprint o historia inexistente o de otro proyecto en la finalización

El sistema MUST responder `404` con el código `project_not_found` cuando el `project_id` no corresponda a ningún proyecto existente. MUST responder `404` con el código `sprint_not_found` cuando el `sprint_id` no corresponda a ningún Sprint existente o corresponda a un Sprint de otro proyecto. MUST responder `404` con el código `story_not_found` cuando el `story_id` no corresponda a ninguna historia existente o corresponda a una historia de otro proyecto. En cualquiera de estos casos el sistema MUST NOT modificar ninguna asociación historia-Sprint.

#### Scenario: Proyecto inexistente

- GIVEN un `project_id` con formato válido que no corresponde a ningún proyecto existente
- WHEN se solicita registrar la finalización usando ese `project_id`
- THEN el sistema responde `404` con el código `project_not_found`
- AND no se modifica ninguna asociación historia-Sprint

#### Scenario: Sprint inexistente

- GIVEN un proyecto existente y un `sprint_id` con formato válido que no corresponde a ningún Sprint
- WHEN se solicita registrar la finalización usando ese `sprint_id`
- THEN el sistema responde `404` con el código `sprint_not_found`
- AND no se modifica ninguna asociación historia-Sprint

#### Scenario: Sprint de otro proyecto

- GIVEN dos proyectos, P1 y P2, y un Sprint que pertenece a P2
- WHEN se solicita registrar la finalización bajo la ruta de P1 usando ese Sprint
- THEN el sistema responde `404` con el código `sprint_not_found`
- AND no se modifica ninguna asociación historia-Sprint

#### Scenario: Historia inexistente

- GIVEN un proyecto existente, un Sprint de ese proyecto y un `story_id` con formato válido que no corresponde a ninguna historia
- WHEN se solicita registrar la finalización usando ese `story_id`
- THEN el sistema responde `404` con el código `story_not_found`
- AND no se modifica ninguna asociación historia-Sprint

#### Scenario: Historia de otro proyecto

- GIVEN dos proyectos, P1 y P2, un Sprint de P1 y una historia que pertenece a P2
- WHEN se solicita registrar la finalización bajo la ruta de P1 usando esa historia
- THEN el sistema responde `404` con el código `story_not_found`
- AND no se modifica ninguna asociación historia-Sprint

### Requirement: Rechazar una historia que no está asignada al Sprint indicado en la finalización

El sistema MUST responder `409` con el código `story_not_in_sprint` cuando la historia exista en el proyecto indicado pero no esté asociada al Sprint de la ruta, incluido el caso de una historia asignada únicamente a otro Sprint del mismo proyecto. El sistema MUST NOT crear ni modificar ninguna asociación historia-Sprint en ese caso: registrar la finalización MUST NOT asignar la historia al Sprint. Esta verificación de pertenencia MUST resolverse dentro de la misma operación que registra la finalización, sin depender de una comprobación previa separada.

#### Scenario: Historia del proyecto pero no asignada a ese Sprint

- GIVEN un proyecto existente, un Sprint abierto de ese proyecto y una historia del mismo proyecto que no está asociada a ese Sprint
- WHEN se solicita registrar la finalización de esa historia en ese Sprint
- THEN el sistema responde `409` con el código `story_not_in_sprint`
- AND no existe ninguna asociación entre esa historia y ese Sprint

#### Scenario: Historia asignada solo a otro Sprint del mismo proyecto

- GIVEN un proyecto con dos Sprints abiertos, A y B, y una historia asignada únicamente a A
- WHEN se solicita registrar la finalización de esa historia usando el Sprint B
- THEN el sistema responde `409` con el código `story_not_in_sprint`
- AND la asociación con A conserva `completed_at` sin definir
- AND no existe ninguna asociación entre la historia y B

### Requirement: Rechazar el registro en un Sprint cerrado

El sistema MUST responder `409` con el código `sprint_closed` cuando el Sprint indicado tenga `is_closed` igual a `true`, aunque la historia esté asignada a él. El sistema MUST NOT modificar ninguna asociación historia-Sprint en ese caso. Esta regla difiere deliberadamente de la creación de tareas (US-10), que sí admite Sprints cerrados.

#### Scenario: Sprint cerrado con una historia asignada sin completar

- GIVEN un proyecto existente, un Sprint con `is_closed=true` y una historia asignada a ese Sprint sin finalización registrada
- WHEN se solicita registrar la finalización de esa historia
- THEN el sistema responde `409` con el código `sprint_closed`
- AND la asociación conserva `completed_at` sin definir

#### Scenario: Sprint cerrado con una historia ya completada

- GIVEN un Sprint con `is_closed=true` y una historia asignada que ya tenía `completed_at` definido
- WHEN se solicita registrar la finalización de esa historia
- THEN el sistema responde `409` con el código `sprint_closed`
- AND `completed_at` conserva el valor original

### Requirement: No contar dos veces una historia completada en el mismo Sprint

El sistema MUST responder `409` con el código `story_already_completed` cuando la historia ya tenga una finalización registrada en ese Sprint. El sistema MUST conservar sin modificarse el registro original (el `completed_at` previo) y MUST NOT crear un segundo registro ni actualizar el instante. Una historia MUST contarse como completada, como máximo, una vez por Sprint, incluso ante solicitudes concurrentes: si varias solicitudes simultáneas registran la misma historia en el mismo Sprint, exactamente una MUST responder `200` y las demás MUST responder `409 story_already_completed`. La finalización en un Sprint MUST NOT afectar a la misma historia asignada a otros Sprints.

Cuando varias condiciones de rechazo coincidan, el orden de precedencia MUST ser: identificadores inválidos (`422`), proyecto, Sprint e historia inexistentes (`404`), Sprint cerrado (`409 sprint_closed`), historia no asignada (`409 story_not_in_sprint`) y, por último, historia ya completada (`409 story_already_completed`).

#### Scenario: Segundo registro secuencial de la misma historia

- GIVEN un proyecto, un Sprint abierto y una historia asignada que ya fue registrada como completada con un `completed_at` T1
- WHEN se solicita nuevamente registrar la finalización de esa historia en ese Sprint
- THEN el sistema responde `409` con el código `story_already_completed`
- AND `completed_at` sigue siendo T1
- AND el Sprint cuenta esa historia como completada una sola vez

#### Scenario: Registros concurrentes de la misma historia

- GIVEN un proyecto, un Sprint abierto y una historia asignada sin finalización registrada
- WHEN se envían simultáneamente varias solicitudes de finalización de esa historia en ese Sprint
- THEN exactamente una solicitud responde `200`
- AND todas las demás responden `409` con el código `story_already_completed`
- AND la asociación tiene un único `completed_at`, igual al devuelto por la solicitud exitosa
- AND el Sprint cuenta esa historia como completada una sola vez

#### Scenario: Completar la misma historia en otro Sprint no es un doble conteo

- GIVEN una historia asignada a los Sprints abiertos A y B, ya completada en A
- WHEN se solicita registrar su finalización en B
- THEN el sistema responde `200`
- AND ambas asociaciones tienen su propio `completed_at` definido

#### Scenario: Una historia ya completada se rechaza como ya completada, no como no asignada

- GIVEN una historia asignada a un Sprint abierto, ya registrada como completada
- WHEN se solicita registrar su finalización otra vez
- THEN el sistema responde `409 story_already_completed` y no `story_not_in_sprint` ni `sprint_closed`

### Requirement: No depender de ni modificar el estado de la historia

El registro de finalización MUST NOT leer, validar ni modificar `stories.status` ni ningún otro campo de la historia. Una historia MUST poder registrarse como completada en el Sprint con cualquier valor de `status` (`pendiente`, `en progreso` o `completada`). La finalización por Sprint MUST NOT depender de `stories.status` ni exigir que las tareas de la historia (US-10) estén completas, y el sistema MUST NOT registrar quién completó la historia. Este cambio MUST NOT ofrecer una operación para deshacer una finalización ni consultas de métricas (US-19).

#### Scenario: `stories.status` no cambia al registrar la finalización

- GIVEN una historia asignada a un Sprint abierto con `status` `pendiente`
- WHEN se registra su finalización en ese Sprint
- THEN el sistema responde `200`
- AND `stories.status` sigue siendo `pendiente` y los demás campos de la historia permanecen iguales

#### Scenario: Una historia con `status` completada igualmente puede registrarse

- GIVEN una historia asignada a un Sprint abierto con `status` `completada` y sin finalización registrada en ese Sprint
- WHEN se solicita registrar su finalización
- THEN el sistema responde `200`
- AND `stories.status` sigue siendo `completada`

#### Scenario: Modificar el estado de la historia no altera la finalización

- GIVEN una historia con `completed_at` definido en un Sprint
- WHEN la historia se modifica con `PUT` cambiando su `status` a `pendiente`
- THEN `completed_at` de la asociación con ese Sprint permanece sin cambios

#### Scenario: Se permite completar con tareas pendientes

- GIVEN una historia asignada a un Sprint abierto que tiene tareas creadas
- WHEN se solicita registrar su finalización
- THEN el sistema responde `200`
- AND las tareas permanecen sin cambios

### Requirement: Fallar sin divulgar detalles internos ni modificar datos ante errores del almacenamiento en la finalización

El sistema MUST responder `500` con el código `internal_error` y un mensaje genérico ante un error inesperado del almacenamiento durante el registro de finalización. La respuesta MUST NOT divulgar detalles internos (consultas, nombres de restricciones ni mensajes del controlador) y la operación MUST NOT dejar persistida una finalización parcial.

#### Scenario: Fallo inesperado del almacenamiento

- GIVEN un proyecto, un Sprint abierto y una historia asignada sin finalización registrada
- AND un fallo inesperado del almacenamiento durante el registro
- WHEN se solicita registrar la finalización
- THEN el sistema responde `500` con el código `internal_error` y un mensaje genérico
- AND la respuesta no contiene detalles internos del almacenamiento
- AND la asociación conserva `completed_at` sin definir

### Requirement: Persistir la finalización en la asociación historia-Sprint con migración reversible (esquema 10)

El esquema de almacenamiento MUST incorporar, mediante la migración `000010`, una columna `completed_at` de tipo instante con zona horaria, nulable, en la asociación historia-Sprint (`sprint_stories`), donde `NULL` significa "no completada en este Sprint". Las asociaciones existentes MUST conservarse con `completed_at` en `NULL` tras aplicar la migración. La migración inversa MUST eliminar la columna `completed_at` y, con ella, todos los registros de finalización persistidos, sin alterar el resto de `sprint_stories` (pertenencia, claves ni restricciones previas); esta pérdida de datos MUST documentarse. El sistema MUST mantener la numeración de migraciones consistente, de modo que `000010` siga a la última migración existente (`000009`).

#### Scenario: Aplicar la migración conserva las asignaciones existentes

- GIVEN un esquema en versión 9 con asociaciones historia-Sprint existentes
- WHEN se aplica la migración `000010`
- THEN la columna `completed_at` existe en `sprint_stories` y admite `NULL`
- AND todas las asociaciones previas siguen presentes con `completed_at` en `NULL`

#### Scenario: Revertir la migración elimina la columna y los registros de finalización

- GIVEN un esquema en versión 10 con asociaciones, algunas con `completed_at` definido
- WHEN se aplica la migración inversa de `000010`
- THEN la columna `completed_at` ya no existe
- AND las asociaciones historia-Sprint (`sprint_id`, `story_id`, `project_id`) siguen presentes y sus restricciones previas se mantienen

#### Scenario: El conjunto de migraciones reconoce la versión 10

- GIVEN el directorio de migraciones con los archivos `000010` de subida y bajada
- WHEN se ejecuta la verificación de archivos de migración
- THEN la verificación reconoce `000010` como la última migración, sin huecos ni duplicados nuevos

### Requirement: Habilitar la ruta de finalización solo con el esquema en versión 10 o superior

El sistema MUST exponer la ruta de finalización únicamente cuando el esquema de almacenamiento esté en versión 10 o superior y sin estado inconsistente (`dirty`), mediante un flag nuevo (`Completion`) en la resolución de disponibilidad de migraciones y una dependencia opcional propia, independiente de las demás. Con el esquema en una versión inferior a 10, o en estado `dirty`, la ruta MUST NOT existir y una solicitud sobre ella MUST responder `404`. Esta condición MUST NOT alterar la disponibilidad de las rutas existentes de proyecto, historia, Sprint, integrantes, asignación de historias a Sprint ni tareas, y las versiones 9 e inferiores MUST conservar su disponibilidad actual.

#### Scenario: La ruta está disponible con el esquema en versión 10

- GIVEN el esquema de almacenamiento en versión 10 y sin estado inconsistente
- WHEN el sistema arranca y se solicita registrar la finalización de una historia asignada a un Sprint abierto
- THEN el sistema responde `200`

#### Scenario: La ruta no existe con el esquema en versión 9

- GIVEN el esquema de almacenamiento en versión 9 y sin estado inconsistente
- WHEN se solicita `POST` sobre la ruta de finalización
- THEN el sistema responde `404` porque la ruta no está registrada
- AND la resolución de disponibilidad reporta `Completion` deshabilitado y `Tasks` habilitado

#### Scenario: El esquema inconsistente deja la ruta no disponible

- GIVEN el esquema de almacenamiento en versión 10 o superior con estado inconsistente (`dirty`)
- WHEN el sistema arranca
- THEN la ruta de finalización no queda expuesta

#### Scenario: Las rutas existentes no cambian

- GIVEN el esquema de almacenamiento en versión 10 y sin estado inconsistente
- WHEN se solicitan las operaciones existentes de proyecto, historia, Sprint, integrantes, asignación de historias a Sprint y tareas
- THEN todas responden igual que antes de habilitar la ruta de finalización
