# Delta para Historia

Este delta modifica la capacidad `historia` (US-07, issue #35). Agrega la consulta de solo lectura del Product Backlog de un proyecto existente, `GET /projects/{project_id}/stories`, con un orden determinista (prioridad `alta` > `media` > `baja` y, a igual prioridad, orden de creación), la distinción entre proyecto vacío y proyecto inexistente, y su disponibilidad condicionada a la versión 4 del esquema. Los requisitos de creación (US-05) y de modificación (US-06) siguen vigentes sin cambios de comportamiento; solo cambia un escenario de la ruta de colección, porque `GET` deja de responder `405`.

Convenciones de los escenarios: los identificadores técnicos (`project_id`, `stories`, `id`, `title`, `description`, `priority`, `status`, `story_points`, `acceptance_criteria`, `estimated_hours`, `seq`), las rutas, los códigos de error (`validation_failed`, `project_not_found`, `internal_error`) y los códigos HTTP se mantienen tal cual. Los escenarios de error `422` indican los campos en `fields`. Los escenarios que describen el orden de creación suponen historias creadas mediante la operación de creación, una tras otra, después de que la consulta está disponible.

## ADDED Requirements

### Requirement: Consultar el Product Backlog de un proyecto existente

El sistema MUST ofrecer una operación de consulta de solo lectura, `GET /projects/{project_id}/stories`, que reciba el identificador de un proyecto existente en la ruta, sin requerir un listado de proyectos ni parámetros de consulta. Ante una consulta válida sobre un proyecto existente con historias, MUST responder `200` con un contenedor `{"project_id": "...", "stories": [...]}`, donde `project_id` es el identificador consultado y `stories` contiene todas las historias de ese proyecto y únicamente las de ese proyecto. Cada elemento de `stories` MUST usar la representación completa de la historia: `id`, `project_id`, `title`, `description`, `priority`, `status`, `story_points`, `acceptance_criteria` y `estimated_hours`, de modo que cada historia quede identificada de forma unívoca por su `id` y muestre como mínimo título, prioridad, estado y Story Points. Los valores ausentes (`story_points` y `estimated_hours`) MUST representarse como `null`, nunca como cero. La respuesta MUST NOT exponer ningún dato de secuencia interna de creación (`seq`) ni ningún otro campo fuera de la representación completa. La operación MUST NOT paginar ni filtrar: devuelve todas las historias del proyecto.

#### Scenario: Consultar el backlog de un proyecto con historias

- GIVEN un proyecto existente con tres historias registradas, cada una con título, descripción, prioridad, estado y criterios de aceptación
- WHEN se solicita `GET /projects/{project_id}/stories` con el identificador de ese proyecto
- THEN el sistema responde `200`
- AND el cuerpo es un objeto con `project_id` igual al identificador consultado y `stories` con exactamente tres elementos
- AND cada elemento contiene `id`, `project_id`, `title`, `description`, `priority`, `status`, `story_points`, `acceptance_criteria` y `estimated_hours`

#### Scenario: Cada historia queda identificada de forma unívoca

- GIVEN un proyecto existente con dos historias que tienen el mismo título
- WHEN se consulta su Product Backlog
- THEN los dos elementos devueltos tienen valores de `id` distintos
- AND cada `id` coincide con el que el sistema devolvió al crear esa historia

#### Scenario: Los valores ausentes se representan como nulos

- GIVEN un proyecto existente con una historia recién creada, sin Story Points ni horas estimadas
- WHEN se consulta su Product Backlog
- THEN el elemento devuelto tiene `status` `pendiente`, `story_points` `null` y `estimated_hours` `null`

#### Scenario: La consulta refleja el contenido más reciente de una historia modificada

- GIVEN un proyecto existente con una historia que fue modificada con título "Login con email", prioridad `alta`, estado `en_progreso` y `estimated_hours` `8.5`
- WHEN se consulta su Product Backlog
- THEN el elemento devuelto contiene exactamente esos valores modificados
- AND los criterios de aceptación aparecen en el mismo orden en que fueron almacenados

#### Scenario: Solo se devuelven historias del proyecto consultado

- GIVEN dos proyectos existentes A y B, con dos historias en A y tres historias en B
- WHEN se consulta el Product Backlog del proyecto A
- THEN `stories` contiene exactamente las dos historias del proyecto A
- AND ninguna historia del proyecto B aparece en la respuesta
- AND todos los elementos devueltos tienen `project_id` igual al del proyecto A

#### Scenario: La respuesta no expone la secuencia de creación

- GIVEN un proyecto existente con al menos una historia
- WHEN se consulta su Product Backlog
- THEN ningún elemento de `stories` ni el contenedor incluyen la clave `seq`
- AND cada elemento contiene únicamente las nueve claves de la representación completa de la historia

#### Scenario: El backlog incluye historias en todos los estados

- GIVEN un proyecto existente con una historia `pendiente`, una `en_progreso` y una `completada`
- WHEN se consulta su Product Backlog
- THEN las tres historias aparecen en `stories`, cada una con su estado
- AND el sistema no excluye ninguna historia por su estado

### Requirement: Ordenar el backlog por prioridad y, a igual prioridad, por orden de creación

El sistema MUST devolver las historias de `stories` ordenadas primero por prioridad, con la precedencia `alta` > `media` > `baja`, y, dentro de una misma prioridad, por orden de creación ascendente: la historia creada antes MUST aparecer antes. El orden MUST ser determinista: dos consultas consecutivas sin cambios intermedios MUST devolver las historias en el mismo orden, incluso ante empates de prioridad. El orden de creación MUST determinarse por el momento en que la operación de creación registró cada historia, no por el valor de su `id`. La modificación de una historia MUST NOT alterar su posición de creación: cambiar el título, el estado o las horas estimadas no la mueve dentro de su prioridad, y cambiar su prioridad la reubica en el grupo de la nueva prioridad según su orden de creación original. Para las historias registradas antes de habilitar la consulta, el orden de creación relativo no está garantizado; el orden por prioridad rige igualmente para todas.

#### Scenario: Ordenar por prioridad alta, media y baja

- GIVEN un proyecto existente con historias creadas en este orden: una `baja`, una `alta` y una `media`
- WHEN se consulta su Product Backlog
- THEN `stories` devuelve primero la historia `alta`, luego la `media` y por último la `baja`

#### Scenario: Desempatar por orden de creación con prioridades repetidas

- GIVEN un proyecto existente con cinco historias creadas en este orden: S1 `media`, S2 `alta`, S3 `media`, S4 `baja`, S5 `alta`
- WHEN se consulta su Product Backlog
- THEN `stories` devuelve, en este orden, S2, S5, S1, S3 y S4
- AND S2 aparece antes que S5 porque fue creada antes, y S1 antes que S3 por la misma razón

#### Scenario: Todas las historias con la misma prioridad conservan el orden de creación

- GIVEN un proyecto existente con tres historias de prioridad `media` creadas en este orden: S1, S2 y S3
- WHEN se consulta su Product Backlog
- THEN `stories` devuelve S1, S2 y S3 en ese orden

#### Scenario: El orden no depende del valor del identificador

- GIVEN un proyecto existente con dos historias de la misma prioridad, donde la historia creada primero tiene un `id` que ordena alfabéticamente después del `id` de la historia creada en segundo lugar
- WHEN se consulta su Product Backlog
- THEN la historia creada primero aparece antes que la creada en segundo lugar

#### Scenario: El orden es estable entre consultas consecutivas

- GIVEN un proyecto existente con varias historias con prioridades repetidas y sin ninguna modificación entre consultas
- WHEN se solicita el Product Backlog dos veces seguidas
- THEN ambas respuestas contienen las mismas historias exactamente en el mismo orden

#### Scenario: Modificar el contenido no altera la posición de creación

- GIVEN un proyecto existente con dos historias `media` creadas en este orden: S1 y S2
- WHEN se modifica S1 cambiando su título y su estado sin cambiar su prioridad y luego se consulta el Product Backlog
- THEN `stories` devuelve S1 antes que S2

#### Scenario: Cambiar la prioridad reubica la historia por su orden de creación original

- GIVEN un proyecto existente con tres historias creadas en este orden: S1 `media`, S2 `alta`, S3 `media`
- WHEN se modifica S3 cambiando su prioridad a `alta` y luego se consulta el Product Backlog
- THEN `stories` devuelve, en este orden, S2, S3 y S1
- AND S3 aparece después de S2 porque S2 fue creada antes

### Requirement: Informar la ausencia de historias con una lista vacía

El sistema MUST responder `200` con el contenedor `{"project_id": "...", "stories": []}` cuando el proyecto existe y no tiene historias. El valor de `stories` MUST ser siempre una lista JSON, y MUST NOT ser `null` ni omitirse. Esa lista vacía es la forma en que el sistema informa que no existen elementos en el Product Backlog; cualquier mensaje textual para quien consulta corresponde a la capa de presentación. La respuesta a un proyecto sin historias MUST distinguirse de la respuesta a un proyecto inexistente.

#### Scenario: Proyecto existente sin historias

- GIVEN un proyecto existente que no tiene ninguna historia
- WHEN se solicita `GET /projects/{project_id}/stories`
- THEN el sistema responde `200`
- AND el cuerpo contiene `project_id` igual al identificador consultado y `stories` igual a `[]`

#### Scenario: La lista vacía se serializa como lista y no como nulo

- GIVEN un proyecto existente que no tiene ninguna historia
- WHEN se consulta su Product Backlog
- THEN el valor JSON de `stories` es una lista vacía `[]`
- AND el valor JSON de `stories` no es `null`

#### Scenario: Un proyecto vacío se distingue de uno inexistente

- GIVEN un proyecto existente sin historias y un identificador de proyecto con formato UUID válido que no corresponde a ningún proyecto
- WHEN se consulta el Product Backlog de cada uno
- THEN el proyecto existente responde `200` con `stories` igual a `[]`
- AND el identificador inexistente responde `404` con el código `project_not_found`

#### Scenario: Un proyecto con historias de otro proyecto sigue vacío

- GIVEN dos proyectos existentes A y B, donde A no tiene historias y B tiene dos
- WHEN se consulta el Product Backlog del proyecto A
- THEN el sistema responde `200` con `stories` igual a `[]`

### Requirement: Rechazar la consulta de un proyecto inexistente

El sistema MUST responder `404` con el código `project_not_found` cuando el `project_id` de la ruta es un UUID válido que no corresponde a ningún proyecto. La respuesta MUST NOT divulgar detalles internos y MUST NOT devolver un contenedor con `stories`. La distinción entre proyecto inexistente y proyecto sin historias MUST resolverse de forma consistente en una única consulta al almacenamiento, de modo que no existan resultados contradictorios por cambios concurrentes entre dos lecturas separadas.

#### Scenario: Proyecto inexistente

- GIVEN un `project_id` con formato UUID válido que no corresponde a ningún proyecto
- WHEN se solicita `GET /projects/{project_id}/stories`
- THEN el sistema responde `404` con el código `project_not_found`
- AND el cuerpo no contiene la clave `stories`

#### Scenario: El rechazo no divulga detalles internos

- GIVEN un `project_id` con formato UUID válido que no corresponde a ningún proyecto
- WHEN se solicita la consulta
- THEN el mensaje de error es genérico
- AND no contiene detalles del almacenamiento ni de la consulta interna

#### Scenario: Un proyecto que existe no responde `404`

- GIVEN un proyecto existente, con o sin historias
- WHEN se solicita la consulta
- THEN el sistema no responde `404 project_not_found`

### Requirement: Rechazar un identificador de proyecto inválido en la consulta

El sistema MUST responder `422` con el código `validation_failed` y `fields.project_id` cuando el `project_id` de la ruta no es un UUID válido. Esta validación MUST resolverse antes de cualquier acceso al almacenamiento: una consulta con un identificador inválido MUST NOT ejecutar ninguna lectura ni ninguna escritura sobre el almacenamiento.

#### Scenario: `project_id` no es un UUID válido

- GIVEN un `project_id` de ruta igual a `"no-es-uuid"`
- WHEN se solicita `GET /projects/no-es-uuid/stories`
- THEN el sistema responde `422 validation_failed`
- AND `fields` contiene `project_id`

#### Scenario: Un identificador inválido no llega al almacenamiento

- GIVEN un `project_id` de ruta que no es un UUID válido, por ejemplo `"abc"`, `"123"` o una cadena con formato UUID incompleto
- WHEN se solicita la consulta
- THEN el sistema rechaza la solicitud antes de consultar el almacenamiento
- AND no se ejecuta ninguna lectura ni escritura sobre el almacenamiento

#### Scenario: Un UUID válido pero inexistente no es un error de validación

- GIVEN un `project_id` con formato UUID válido que no corresponde a ningún proyecto
- WHEN se solicita la consulta
- THEN el sistema responde `404 project_not_found` y no `422`

### Requirement: La consulta MUST NOT modificar el Product Backlog

La consulta del Product Backlog MUST ser de solo lectura: MUST NOT crear, modificar ni eliminar ninguna historia ni ningún proyecto, ni alterar ningún valor almacenado, ni cambiar el orden de creación de las historias. Consultar reiteradamente el mismo proyecto MUST devolver el mismo contenido mientras no haya otras operaciones de escritura.

#### Scenario: La consulta no altera las historias almacenadas

- GIVEN un proyecto existente con varias historias y una instantánea del contenido completo de la tabla de historias tomada antes de consultar
- WHEN se consulta el Product Backlog de ese proyecto una o más veces
- THEN la instantánea de la tabla tomada después es idéntica a la anterior en todas sus filas y valores

#### Scenario: Consultar un proyecto vacío o inexistente no escribe nada

- GIVEN un proyecto existente sin historias y un identificador de proyecto inexistente, con una instantánea de la tabla de historias tomada antes de consultar
- WHEN se consulta el Product Backlog de cada uno
- THEN la tabla de historias no cambia
- AND no se crea ninguna historia ni ningún proyecto

#### Scenario: Consultas repetidas devuelven el mismo contenido

- GIVEN un proyecto existente con historias y sin operaciones de escritura entre consultas
- WHEN se solicita el Product Backlog tres veces seguidas
- THEN las tres respuestas son idénticas

### Requirement: Fallar sin divulgar detalles internos ante errores del almacenamiento en la consulta

El sistema MUST responder `500` con el código `internal_error` y un mensaje genérico cuando la consulta no pueda completarse por un fallo inesperado del almacenamiento. MUST NOT confundir un fallo inesperado con un proyecto inexistente ni con un proyecto sin historias, MUST NOT devolver una lista parcial y MUST NOT divulgar detalles internos del fallo.

#### Scenario: Fallo inesperado del almacenamiento

- GIVEN un proyecto existente y un fallo inesperado del almacenamiento durante la lectura
- WHEN se solicita `GET /projects/{project_id}/stories`
- THEN el sistema responde `500` con el código `internal_error` y un mensaje genérico
- AND la respuesta no contiene detalles internos del fallo

#### Scenario: Un fallo no se reporta como proyecto inexistente

- GIVEN un error de almacenamiento distinto de "proyecto no encontrado"
- WHEN se solicita la consulta
- THEN el sistema no responde `404 project_not_found`

#### Scenario: Un fallo no se reporta como backlog vacío

- GIVEN un error de almacenamiento durante la lectura de un proyecto que tiene historias
- WHEN se solicita la consulta
- THEN el sistema no responde `200` con `stories` igual a `[]`

### Requirement: Disponibilidad de la consulta condicionada a la versión del esquema

El sistema MUST exponer la operación de consulta del Product Backlog únicamente cuando el esquema de almacenamiento esté en versión 4 o superior y sin estado inconsistente (`dirty`). Con el esquema en una versión inferior a 4, o en estado `dirty`, la consulta MUST NOT estar disponible y `GET` sobre la ruta de colección MUST NOT devolver el Product Backlog. Con el esquema en versión 2 o 3 y sin estado `dirty`, esa ruta está registrada únicamente para la creación y `GET` MUST responder `405`. Con el esquema en versión 1, en estado `dirty` o ante un error al leer la versión, no existe ninguna ruta de historias y `GET` MUST responder `404`. La creación de historias MUST mantener su condición de disponibilidad de versión 2 o superior y la modificación su condición de versión 3 o superior, y ambas MUST seguir funcionando sin cambios con el esquema en versión 4. Con el esquema en versión 4 o superior y sin estado `dirty`, creación, modificación y consulta MUST estar disponibles a la vez.

#### Scenario: La consulta está disponible con el esquema en versión 4

- GIVEN el esquema de almacenamiento en versión 4 y sin estado inconsistente
- WHEN el sistema arranca y se solicita `GET /projects/{project_id}/stories` de un proyecto existente
- THEN el sistema responde `200` con el contenedor del Product Backlog

#### Scenario: La consulta no existe con el esquema en versión 3

- GIVEN el esquema de almacenamiento en versión 3 y sin estado inconsistente
- WHEN se solicita `GET /projects/{project_id}/stories` de un proyecto existente
- THEN el sistema responde `405` y no devuelve el Product Backlog
- AND ninguna historia se modifica

#### Scenario: Creación y modificación siguen funcionando con el esquema en versión 3

- GIVEN el esquema de almacenamiento en versión 3 y un proyecto existente con una historia
- WHEN se solicita crear una historia válida y modificar la existente con datos válidos
- THEN ambas operaciones responden con éxito

#### Scenario: Creación y modificación siguen funcionando con el esquema en versión 4

- GIVEN el esquema de almacenamiento en versión 4 y un proyecto existente
- WHEN se crea una historia válida, se la modifica con datos válidos y se consulta el Product Backlog
- THEN la creación y la modificación responden con éxito
- AND la consulta devuelve la historia con los valores modificados

#### Scenario: Esquema en versión 4 con estado inconsistente

- GIVEN el esquema de almacenamiento en versión 4 con estado inconsistente (`dirty`)
- WHEN el sistema arranca y se solicita `GET /projects/{project_id}/stories`
- THEN la consulta no queda expuesta y no devuelve el Product Backlog

## MODIFIED Requirements

### Requirement: Rechazar métodos no admitidos sobre la ruta de la historia

El sistema MUST admitir únicamente `PUT` sobre la ruta `/projects/{project_id}/stories/{story_id}` en esta operación. Cualquier otro método (por ejemplo `GET`, `POST`, `PATCH` o `DELETE`) MUST responder `405` y MUST NOT modificar ninguna historia. La lectura individual de una historia queda fuera de este cambio. Sobre la ruta de colección `/projects/{project_id}/stories`, el sistema MUST admitir `POST` (creación) y, cuando el esquema está en versión 4 o superior sin estado `dirty`, `GET` (consulta del Product Backlog); cualquier otro método sobre la ruta de colección, en particular `PUT` y `DELETE`, MUST responder `405` y MUST NOT modificar ninguna historia.
(Previously: `GET` sobre la ruta de colección respondía `405`, igual que `PUT` y `DELETE`; ahora `GET` es la consulta del Product Backlog y solo `PUT` y `DELETE` siguen respondiendo `405`.)

#### Scenario: Método no admitido

- GIVEN una historia existente y la ruta de su ítem disponible
- WHEN se solicita `PATCH`, `DELETE`, `POST` o `GET` sobre `/projects/{project_id}/stories/{story_id}`
- THEN el sistema responde `405`
- AND la historia almacenada no cambia

#### Scenario: La ruta de colección conserva su comportamiento

- GIVEN la ruta de colección `/projects/{project_id}/stories` con la creación y la consulta disponibles (esquema en versión 4 y sin estado `dirty`)
- WHEN se solicita `PUT` o `DELETE` sobre ella
- THEN el sistema sigue respondiendo `405`, como antes de este cambio
- AND ninguna historia se modifica

#### Scenario: `GET` sobre la ruta de colección ya no responde `405`

- GIVEN la ruta de colección `/projects/{project_id}/stories` con la consulta disponible (esquema en versión 4 y sin estado `dirty`)
- WHEN se solicita `GET` sobre ella con el identificador de un proyecto existente
- THEN el sistema responde `200` con el contenedor del Product Backlog
- AND para un `project_id` con formato UUID válido que no corresponde a ningún proyecto, responde `404` con el código `project_not_found`

#### Scenario: `POST` sobre la ruta de colección sigue siendo la creación

- GIVEN la ruta de colección `/projects/{project_id}/stories` con la consulta disponible y un proyecto existente
- WHEN se solicita `POST` con datos de historia válidos
- THEN el sistema registra la historia y responde con éxito, sin cambios respecto de la creación existente

#### Scenario: La ruta del ítem sigue sin admitir `GET`

- GIVEN el esquema en versión 4, la consulta del Product Backlog disponible y una historia existente
- WHEN se solicita `GET` sobre `/projects/{project_id}/stories/{story_id}`
- THEN el sistema responde `405`
- AND el sistema no devuelve la historia individual

## REMOVED Requirements

Ninguno.
