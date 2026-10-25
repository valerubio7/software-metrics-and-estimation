# Especificación de Esfuerzo

## Propósito

Permitir que un integrante del proyecto registre el esfuerzo real que dedicó a una historia o a una tarea de esa historia, indicando quién trabajó, en qué fecha, qué hizo y cuántas horas. Todo registro de esfuerzo queda persistido y asociado a la historia seleccionada, y también a la tarea cuando se indica una: el `task_id` precisa el destino pero no mueve el registro fuera de su historia, de modo que una agregación posterior por historia (US-19) cuente cada registro exactamente una vez. Si falta un dato obligatorio o un dato es inválido, el esfuerzo no se registra y el sistema informa el motivo. Esta capacidad solo registra: no calcula totales ni desvíos (US-19), no crea tareas (US-10), no estima puntos de historia (US-14), no ofrece listado ni consulta por HTTP y no se opera desde la TUI (TUI-08). El estado de la tarea, diferido en `tarea` a "US-15", sigue diferido: el issue #43 no lo pide y el contrato de `tarea` no cambia.

## Requirements

### Requirement: Registrar esfuerzo sobre una historia

El sistema MUST ofrecer `POST /projects/{project_id}/stories/{story_id}/effort-records` para registrar esfuerzo realizado sobre la historia identificada en la ruta. El cuerpo MUST ser un único objeto JSON con las claves `member_id`, `work_date`, `description` y `hours` obligatorias, y `task_id` opcional. Ante una solicitud válida, el sistema MUST persistir exactamente un registro de esfuerzo asociado a ese proyecto, a esa historia y al integrante indicado, con `task_id` nulo, y MUST responder `201` con el registro creado. El cuerpo de la respuesta MUST incluir el identificador generado por el servidor, `project_id`, `story_id`, `task_id` (`null` si no se indicó), `member_id`, `work_date` en formato `YYYY-MM-DD`, `description` ya recortada y `hours`. La respuesta MUST NOT exponer marcas internas de creación.

#### Scenario: Registrar esfuerzo sobre una historia

- GIVEN un proyecto existente, una historia de ese proyecto y un integrante de ese proyecto
- AND un reloj del servidor que indica una fecha UTC igual o posterior a `2026-03-10`
- AND una solicitud con `member_id` del integrante, `work_date` igual a `"2026-03-10"`, `description` igual a `"Implementar login"` y `hours` igual a `3.5`, sin `task_id`
- WHEN se solicita registrar el esfuerzo sobre esa historia
- THEN el sistema responde `201`
- AND persiste exactamente un registro con `project_id`, `story_id` de la ruta, `member_id`, `work_date` `2026-03-10`, `description` `"Implementar login"`, `hours` `3.5` y `task_id` nulo
- AND el cuerpo de la respuesta incluye un identificador generado por el servidor, esos mismos valores, y `task_id` igual a `null`

#### Scenario: El identificador y la marca de creación los asigna el servidor

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud válida que además no incluye ni `id` ni `created_at`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `201`
- AND el identificador del registro devuelto y persistido es generado por el servidor y no es nulo

#### Scenario: Un `task_id` igual a `null` equivale a no indicar tarea

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud válida con `task_id` igual a `null`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `201`
- AND el registro persistido y devuelto tiene `task_id` nulo y queda asociado a la historia de la ruta

### Requirement: Registrar esfuerzo sobre una tarea de la historia

Cuando el cuerpo incluye un `task_id`, el sistema MUST registrar el esfuerzo sobre esa tarea siempre que la tarea pertenezca a la historia de la ruta dentro del proyecto de la ruta. El registro MUST quedar asociado a la tarea (`task_id`) y, a la vez, a la historia de la ruta (`story_id`) y al proyecto (`project_id`). Un registro con `task_id` MUST NOT perder su asociación a la historia: `story_id` siempre es la historia de la ruta. El sistema MUST persistir exactamente un registro por solicitud, sin generar un segundo registro para la historia.

#### Scenario: Registrar esfuerzo sobre una tarea

- GIVEN un proyecto, una historia de ese proyecto con una tarea y un integrante de ese proyecto
- AND una solicitud válida con `task_id` igual al identificador de esa tarea
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `201`
- AND persiste exactamente un registro con `task_id` de esa tarea y `story_id` de la historia de la ruta
- AND el cuerpo de la respuesta incluye ese `task_id` y ese `story_id`

#### Scenario: El esfuerzo de una tarea cuenta una sola vez para su historia

- GIVEN una historia con dos registros de esfuerzo, uno sin `task_id` y otro con `task_id` de una tarea de esa historia
- WHEN se consultan en el almacenamiento los registros de esa historia por `story_id`
- THEN se obtienen exactamente dos filas, una por registro
- AND ninguna fila adicional representa el registro de la tarea como esfuerzo de la historia

### Requirement: Aceptar el esfuerzo del integrante cuando pertenece al proyecto

El sistema MUST aceptar como autor del esfuerzo únicamente a un integrante del proyecto de la ruta. Un `member_id` que no corresponda a ningún integrante o que corresponda a un integrante de otro proyecto MUST rechazarse con `404` y el código `member_not_found`, sin persistir ningún registro.

#### Scenario: Integrante del proyecto

- GIVEN un proyecto con un integrante, y una historia de ese proyecto
- WHEN se registra esfuerzo válido indicando el `member_id` de ese integrante
- THEN el sistema responde `201`
- AND el registro persistido tiene ese `member_id`

#### Scenario: Integrante inexistente

- GIVEN un proyecto y una historia existentes
- AND un `member_id` con formato UUID válido que no corresponde a ningún integrante
- WHEN se solicita registrar esfuerzo válido con ese `member_id`
- THEN el sistema responde `404` con el código `member_not_found`
- AND no se persiste ningún registro

#### Scenario: Integrante de otro proyecto

- GIVEN dos proyectos A y B, una historia del proyecto A y un integrante que pertenece solo al proyecto B
- WHEN se solicita registrar esfuerzo válido sobre la historia del proyecto A indicando el `member_id` de ese integrante
- THEN el sistema responde `404` con el código `member_not_found`
- AND no se persiste ningún registro

### Requirement: Aceptar una tarea solo si pertenece a la historia de la ruta

Cuando se indica `task_id`, el sistema MUST rechazar con `404` y el código `task_not_found` una tarea que no exista, que pertenezca a otra historia (aunque sea del mismo proyecto) o que pertenezca a una historia de otro proyecto. En esos casos el sistema MUST NOT persistir ningún registro.

#### Scenario: Tarea inexistente

- GIVEN un proyecto, una historia y un integrante válidos
- AND un `task_id` con formato UUID válido que no corresponde a ninguna tarea
- WHEN se solicita registrar esfuerzo válido con ese `task_id`
- THEN el sistema responde `404` con el código `task_not_found`
- AND no se persiste ningún registro

#### Scenario: Tarea de otra historia del mismo proyecto

- GIVEN un proyecto con dos historias H1 y H2, una tarea de H2 y un integrante del proyecto
- WHEN se solicita registrar esfuerzo válido sobre H1 con el `task_id` de la tarea de H2
- THEN el sistema responde `404` con el código `task_not_found`
- AND no se persiste ningún registro

#### Scenario: Tarea de una historia de otro proyecto

- GIVEN dos proyectos A y B, una historia del proyecto A, una tarea de una historia del proyecto B y un integrante del proyecto A
- WHEN se solicita registrar esfuerzo válido sobre la historia del proyecto A con el `task_id` de la tarea del proyecto B
- THEN el sistema responde `404` con el código `task_not_found`
- AND no se persiste ningún registro

### Requirement: Rechazar un proyecto o una historia inexistente o ajena

El sistema MUST responder `404` con el código `project_not_found` cuando el `project_id` de la ruta no corresponda a ningún proyecto existente, y `404` con el código `story_not_found` cuando el `story_id` no corresponda a ninguna historia existente o corresponda a una historia de otro proyecto. En ambos casos el sistema MUST NOT persistir ningún registro. Cuando coexistan varias referencias inexistentes, el sistema MUST informar la primera en este orden: proyecto, historia, integrante, tarea.

#### Scenario: Proyecto inexistente

- GIVEN un `project_id` con formato UUID válido que no corresponde a ningún proyecto
- WHEN se solicita registrar esfuerzo con una solicitud de cuerpo válido bajo ese proyecto
- THEN el sistema responde `404` con el código `project_not_found`
- AND no se persiste ningún registro

#### Scenario: Historia inexistente

- GIVEN un proyecto existente y un `story_id` con formato UUID válido que no corresponde a ninguna historia
- WHEN se solicita registrar esfuerzo con una solicitud de cuerpo válido sobre esa historia
- THEN el sistema responde `404` con el código `story_not_found`
- AND no se persiste ningún registro

#### Scenario: Historia de otro proyecto

- GIVEN dos proyectos A y B, una historia que pertenece solo al proyecto B y un integrante del proyecto A
- WHEN se solicita registrar esfuerzo válido bajo la ruta del proyecto A con el `story_id` de la historia del proyecto B
- THEN el sistema responde `404` con el código `story_not_found`
- AND no se persiste ningún registro

#### Scenario: Varias referencias inexistentes informan la primera en orden

- GIVEN un proyecto existente y una historia existente
- AND una solicitud válida cuyo `member_id` y cuyo `task_id` no existen
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `404` con el código `member_not_found`
- AND no se persiste ningún registro

### Requirement: Validar la fecha del trabajo

El sistema MUST exigir `work_date` como una cadena con formato estricto `YYYY-MM-DD` que represente una fecha de calendario real; una cadena que no sobreviva a un parseo y reformateo idéntico (por ejemplo `"2026-02-30"`, `"2026-3-1"` o `"10/03/2026"`) MUST rechazarse. El sistema MUST rechazar una `work_date` posterior a la fecha UTC actual del servidor, obtenida de un reloj inyectable; la fecha de hoy y las fechas pasadas MUST aceptarse. Toda violación MUST responderse con `422 validation_failed` indicando `work_date` en `fields`, sin persistir ningún registro. La fecha MUST persistirse y devolverse como fecha de calendario, sin componente horario ni zona.

#### Scenario: Aceptar la fecha de hoy según el reloj UTC

- GIVEN un proyecto, una historia y un integrante válidos
- AND un reloj del servidor fijo en `2026-03-10T23:59:59Z`
- WHEN se solicita registrar esfuerzo con `work_date` igual a `"2026-03-10"`
- THEN el sistema responde `201`
- AND el registro persistido tiene `work_date` `2026-03-10`

#### Scenario: Aceptar una fecha pasada

- GIVEN un proyecto, una historia y un integrante válidos
- AND un reloj del servidor fijo en `2026-03-10T08:00:00Z`
- WHEN se solicita registrar esfuerzo con `work_date` igual a `"2026-01-05"`
- THEN el sistema responde `201`

#### Scenario: Rechazar una fecha futura

- GIVEN un proyecto, una historia y un integrante válidos
- AND un reloj del servidor fijo en `2026-03-10T23:59:59Z`
- WHEN se solicita registrar esfuerzo con `work_date` igual a `"2026-03-11"`
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `work_date`
- AND no se persiste ningún registro

#### Scenario: La fecha de comparación es la fecha UTC, no la local

- GIVEN un reloj del servidor fijo en un instante cuya fecha UTC es `2026-03-10` aunque en una zona horaria local ya sea `2026-03-11`
- WHEN se solicita registrar esfuerzo con `work_date` igual a `"2026-03-11"`
- THEN el sistema responde `422 validation_failed` con `work_date` en `fields`
- AND no se persiste ningún registro

#### Scenario: Rechazar una fecha faltante o con formato inválido

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud sin `work_date`, o con `work_date` igual a `""`, `"2026-02-30"`, `"2026-3-1"`, `"10/03/2026"` o `"2026-03-10T00:00:00Z"`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `work_date`
- AND no se persiste ningún registro

### Requirement: Validar las horas trabajadas

El sistema MUST exigir `hours` como número y MUST aceptarlo únicamente si es estrictamente mayor que `0`, menor o igual que `24` y tiene como máximo dos decimales. Un valor ausente, igual o menor que `0`, mayor que `24` o con más de dos decimales MUST rechazarse con `422 validation_failed` indicando `hours` en `fields`, sin persistir ningún registro. Un valor numérico válido MUST persistirse y devolverse sin pérdida de precisión.

#### Scenario: Aceptar valores válidos de horas

- GIVEN un proyecto, una historia y un integrante válidos
- AND solicitudes con `hours` igual a `0.01`, `0.5`, `8`, `7.25` o `24`
- WHEN se solicita registrar cada esfuerzo
- THEN el sistema responde `201` en cada caso
- AND cada registro persistido y devuelto conserva exactamente el valor enviado

#### Scenario: Rechazar horas iguales a cero o negativas

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con `hours` igual a `0`, `-1` o `-0.01`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `hours`
- AND no se persiste ningún registro

#### Scenario: Rechazar horas mayores que 24

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con `hours` igual a `24.01` o `100`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `hours`
- AND no se persiste ningún registro

#### Scenario: Rechazar horas con más de dos decimales

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con `hours` igual a `1.005` o `0.001`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `hours`
- AND no se persiste ningún registro

#### Scenario: Rechazar horas faltantes

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud sin la clave `hours` o con `hours` igual a `null`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `hours`
- AND no se persiste ningún registro

### Requirement: Validar la descripción de la actividad

El sistema MUST exigir `description` como cadena que, una vez recortados los espacios iniciales y finales, no esté vacía y tenga como máximo 500 caracteres (se cuentan caracteres, no bytes). El sistema MUST persistir y devolver la descripción ya recortada. Una descripción ausente, vacía, compuesta solo por espacios, mayor que 500 caracteres o que contenga el carácter NUL (`U+0000`) MUST rechazarse con `422 validation_failed` indicando `description` en `fields`, sin persistir ningún registro.

#### Scenario: La descripción se recorta antes de persistir

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con `description` igual a `"  Revisión de código  "`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `201`
- AND el registro persistido y la respuesta tienen `description` igual a `"Revisión de código"`

#### Scenario: Aceptar una descripción de exactamente 500 caracteres

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con una `description` de 500 caracteres, incluso si algunos son caracteres multibyte
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `201`
- AND la descripción se persiste completa

#### Scenario: Rechazar una descripción de más de 500 caracteres

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con una `description` de 501 caracteres tras recortarla
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `description`
- AND no se persiste ningún registro

#### Scenario: Rechazar una descripción faltante o vacía

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con `description` ausente, igual a `""` o igual a `"   "`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `description`
- AND no se persiste ningún registro

#### Scenario: Rechazar una descripción con el carácter NUL

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con una `description` que contiene el carácter NUL (`U+0000`), por ejemplo `"a\u0000b"` en el JSON
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `description`
- AND no se persiste ningún registro
- AND no se consulta ni se escribe en el almacenamiento

### Requirement: Validar los identificadores del cuerpo y de la ruta

El sistema MUST exigir `member_id` y, cuando se indique, `task_id` como UUID con formato válido. Un `member_id` ausente, vacío o con formato inválido MUST rechazarse con `422 validation_failed` indicando `member_id` en `fields`; un `task_id` presente con formato inválido MUST rechazarse con `422 validation_failed` indicando `task_id` en `fields`. Un `project_id` o un `story_id` de la ruta con formato UUID inválido MUST rechazarse con `422 validation_failed` indicando el parámetro en `fields`, conforme a la convención de las rutas existentes. En todos los casos el sistema MUST NOT persistir ningún registro.

#### Scenario: `member_id` faltante

- GIVEN un proyecto, una historia y una solicitud válida sin la clave `member_id`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `member_id`
- AND no se persiste ningún registro

#### Scenario: `member_id` con formato inválido

- GIVEN un proyecto y una historia existentes
- AND una solicitud válida con `member_id` igual a `"abc"`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `member_id`
- AND no se persiste ningún registro

#### Scenario: `task_id` con formato inválido

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud válida con `task_id` igual a `"abc"`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `task_id`
- AND no se persiste ningún registro

#### Scenario: Parámetros de ruta con formato inválido

- GIVEN una solicitud con cuerpo válido
- WHEN se solicita registrar esfuerzo con `project_id` igual a `"abc"`, o con `story_id` igual a `"abc"`
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye el parámetro inválido (`project_id` o `story_id`)
- AND no se persiste ningún registro

### Requirement: Informar juntos todos los errores de campo

Cuando el cuerpo tiene más de un campo inválido, el sistema MUST responder una única vez `422 validation_failed` con un objeto `fields` que incluya todos los campos inválidos, cada uno con un motivo legible. La validación de campos MUST resolverse antes de consultar el almacenamiento: una solicitud con campos inválidos MUST responder `422 validation_failed` aunque además referencie un integrante, una tarea, una historia o un proyecto inexistentes.

#### Scenario: Varios campos inválidos en una misma solicitud

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con `hours` igual a `0`, `description` igual a `"   "` y `work_date` igual a `"2026-02-30"`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `hours`, `description` y `work_date`, cada uno con un motivo
- AND no se persiste ningún registro

#### Scenario: Los errores de campo tienen precedencia sobre los errores de referencia

- GIVEN un proyecto y una historia existentes
- AND una solicitud con `hours` igual a `0` y con un `member_id` que no corresponde a ningún integrante
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `422 validation_failed` con `hours` en `fields`
- AND no responde `404 member_not_found`
- AND no se persiste ningún registro

### Requirement: Rechazar solicitudes malformadas sin persistir registros

El sistema MUST responder `400 invalid_request` cuando el cuerpo no sea un único objeto JSON válido, cuando contenga una clave desconocida, cuando contenga contenido adicional tras el objeto, o cuando algún campo tenga un tipo JSON incorrecto (por ejemplo `hours` como cadena, `description` numérica, `work_date` numérica, `member_id` numérico o `task_id` numérico). El sistema MUST NOT persistir ningún registro en estos casos.

#### Scenario: JSON malformado

- GIVEN un proyecto, una historia y un integrante válidos
- AND un cuerpo que no es JSON válido, por ejemplo `{"member_id":`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ningún registro

#### Scenario: Campo desconocido

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud válida que además incluye una clave desconocida, por ejemplo `"status": "done"` o `"story_id"`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ningún registro

#### Scenario: Tipo incorrecto en un campo

- GIVEN un proyecto, una historia y un integrante válidos
- AND una solicitud con `hours` igual a la cadena `"8"`, o con `description` igual al número `5`
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ningún registro

#### Scenario: Contenido adicional tras el objeto JSON

- GIVEN un proyecto, una historia y un integrante válidos
- AND un cuerpo con un objeto JSON válido seguido de un segundo objeto JSON
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `400 invalid_request`
- AND no se persiste ningún registro

### Requirement: Permitir registrar esfuerzo con independencia del Sprint de la historia

El sistema MUST permitir registrar esfuerzo sobre una historia (o una tarea de ella) cuyo Sprint esté cerrado (`is_closed=true`) y sobre una historia que no esté asignada a ningún Sprint (backlog), sin aplicar ninguna restricción adicional por esos estados. El sistema MUST NOT exigir ni modificar la pertenencia de la historia a un Sprint al registrar esfuerzo.

#### Scenario: Un Sprint cerrado no impide registrar esfuerzo

- GIVEN un proyecto, un Sprint con `is_closed=true`, una historia asignada a ese Sprint y un integrante del proyecto
- WHEN se solicita registrar esfuerzo válido sobre esa historia
- THEN el sistema responde `201`
- AND persiste el registro igual que si el Sprint estuviera abierto

#### Scenario: Registrar esfuerzo sobre una tarea de un Sprint cerrado

- GIVEN un proyecto, un Sprint con `is_closed=true`, una historia asignada a ese Sprint con una tarea, y un integrante del proyecto
- WHEN se solicita registrar esfuerzo válido con el `task_id` de esa tarea
- THEN el sistema responde `201`
- AND persiste el registro asociado a la tarea y a su historia

#### Scenario: Una historia del backlog admite esfuerzo

- GIVEN un proyecto, una historia sin ningún Sprint asignado y un integrante del proyecto
- WHEN se solicita registrar esfuerzo válido sobre esa historia
- THEN el sistema responde `201`
- AND persiste el registro, y la historia sigue sin Sprint asignado

### Requirement: Permitir registros de esfuerzo duplicados

El sistema MUST aceptar y persistir como registros independientes dos solicitudes con idénticos `member_id`, `task_id`, `work_date`, `description` y `hours`. El sistema MUST NOT detectar, rechazar ni fusionar duplicados, y MUST NOT limitar las horas acumuladas por integrante y día. Cada registro MUST recibir un identificador distinto.

#### Scenario: Dos solicitudes idénticas generan dos registros

- GIVEN un proyecto, una historia y un integrante válidos
- AND dos solicitudes idénticas y válidas
- WHEN se solicita registrar el esfuerzo dos veces
- THEN el sistema responde `201` en ambas
- AND persiste exactamente dos registros con identificadores distintos

#### Scenario: Las horas acumuladas del día no se limitan

- GIVEN un integrante con registros del mismo día que suman 24 horas
- WHEN se registra un nuevo esfuerzo válido de 8 horas para ese integrante y esa fecha
- THEN el sistema responde `201`
- AND persiste el registro

### Requirement: Persistir el registro completo y asociado a su historia

El sistema MUST persistir cada registro de esfuerzo aceptado con, como mínimo, estos datos: identificador generado por el servidor, `project_id`, `story_id`, `task_id` (nulo cuando no se indicó tarea), `member_id`, `work_date` (fecha de calendario), `description` (recortada), `hours` (decimal exacto con dos decimales) y la marca de creación. `project_id` y `story_id` MUST ser no nulos en todo registro. Los registros persistidos MUST permanecer disponibles en el almacenamiento para que una funcionalidad posterior (US-19) calcule el esfuerzo real; esta capacidad MUST NOT calcular totales, desvíos ni comparaciones con la estimación, y MUST NOT exponer una ruta de listado, consulta, edición ni eliminación.

#### Scenario: Todos los datos del registro quedan persistidos

- GIVEN un proyecto, una historia con una tarea y un integrante
- WHEN se registra esfuerzo válido con `task_id`, `work_date` `2026-03-10`, `description` `"Escribir pruebas"` y `hours` `2.5`
- THEN el almacenamiento contiene exactamente una fila de esfuerzo con el identificador devuelto en la respuesta
- AND esa fila tiene `project_id`, `story_id`, `task_id`, `member_id`, `work_date` `2026-03-10`, `description` `"Escribir pruebas"`, `hours` `2.50` y una marca de creación no nula

#### Scenario: Un registro persiste solo si todo el alta se completa

- GIVEN un proyecto, una historia y un integrante válidos y una solicitud válida
- AND un fallo inesperado del almacenamiento durante la escritura
- WHEN se solicita registrar el esfuerzo
- THEN el sistema responde `500 internal_error` con un mensaje genérico
- AND no queda persistido ningún registro

#### Scenario: La respuesta coincide con lo persistido

- GIVEN un proyecto, una historia y un integrante válidos
- WHEN se registra esfuerzo válido y se recibe `201`
- THEN los valores de `id`, `project_id`, `story_id`, `task_id`, `member_id`, `work_date`, `description` y `hours` de la respuesta coinciden con la fila almacenada

### Requirement: Garantizar la integridad referencial del esfuerzo en el almacenamiento

El almacenamiento MUST impedir, con independencia de las verificaciones de la aplicación, los registros de esfuerzo cuyo integrante, historia o tarea no pertenezcan al mismo proyecto, y los registros con una tarea que no pertenezca a su historia. Cuando `task_id` es nulo, MUST NOT evaluarse la restricción de tarea. El almacenamiento MUST rechazar además horas fuera de `(0, 24]` y descripciones vacías o de más de 500 caracteres. Si una violación de integridad referencial llega al almacenamiento pese a la verificación previa (por ejemplo, por concurrencia), el sistema MUST traducirla al error de referencia correspondiente (`404 member_not_found`, `404 task_not_found` o `404 story_not_found`) y MUST NOT responder `500`. La inexistencia del proyecto (`404 project_not_found`) no proviene de una clave foránea de proyecto, sino de la verificación de existencia que se ejecuta dentro de la misma transacción de la inserción.

#### Scenario: El almacenamiento rechaza un integrante de otro proyecto

- GIVEN dos proyectos A y B, una historia del proyecto A y un integrante del proyecto B
- WHEN se intenta insertar directamente un registro con `project_id` de A, esa historia y el integrante de B
- THEN el almacenamiento rechaza la inserción por violación de clave foránea

#### Scenario: El almacenamiento rechaza una historia de otro proyecto

- GIVEN dos proyectos A y B, una historia del proyecto B y un integrante del proyecto A
- WHEN se intenta insertar directamente un registro con `project_id` de A y la historia de B
- THEN el almacenamiento rechaza la inserción por violación de clave foránea

#### Scenario: El almacenamiento rechaza una tarea de otra historia

- GIVEN un proyecto con dos historias H1 y H2 y una tarea de H2
- WHEN se intenta insertar directamente un registro con `story_id` de H1 y el `task_id` de la tarea de H2
- THEN el almacenamiento rechaza la inserción por violación de clave foránea

#### Scenario: Un registro sin tarea no evalúa la restricción de tarea

- GIVEN un proyecto, una historia y un integrante válidos
- WHEN se inserta directamente un registro con `task_id` nulo
- THEN el almacenamiento acepta la inserción

#### Scenario: El almacenamiento rechaza horas y descripciones inválidas

- GIVEN un proyecto, una historia y un integrante válidos
- WHEN se intenta insertar directamente un registro con `hours` igual a `0` o `24.01`, o con `description` vacía o de 501 caracteres
- THEN el almacenamiento rechaza cada inserción por violación de restricción de verificación

#### Scenario: Una violación de clave foránea no produce `500`

- GIVEN un repositorio de esfuerzo cuyo alta recibe una violación de clave foránea de integrante, de tarea o de historia
- WHEN la aplicación procesa ese error
- THEN lo traduce al error de referencia correspondiente (`member_not_found`, `task_not_found` o `story_not_found`)
- AND la respuesta HTTP es `404` con ese código y no `500`

### Requirement: Versionar la migración de esfuerzo de forma reversible

El sistema MUST incorporar la migración `000012_create_effort_records` con su par `up` y `down`. El `up` MUST crear la tabla `effort_records` con las restricciones descritas y agregar `UNIQUE(id, project_id)` a `project_members` y `UNIQUE(id, story_id, project_id)` a `tasks`. El `down` MUST eliminar primero `effort_records` y después esas dos restricciones `UNIQUE`. La migración MUST poder aplicarse, revertirse y reaplicarse sin errores, sin alterar los datos existentes de las tablas previas, y el directorio de migraciones MUST contener exactamente 12 pares.

#### Scenario: Aplicar la migración sobre un esquema en versión 11 con datos

- GIVEN un esquema en versión 11 con proyectos, historias, integrantes y tareas existentes
- WHEN se aplica la migración `000012`
- THEN se completa sin errores
- AND existen la tabla `effort_records` y las dos restricciones `UNIQUE` nuevas
- AND los datos previos de `project_members` y `tasks` permanecen sin cambios

#### Scenario: Revertir y reaplicar la migración

- GIVEN un esquema en versión 12
- WHEN se aplica el `down` de `000012` y luego se aplica de nuevo el `up`
- THEN ambas operaciones se completan sin errores
- AND tras el `down` no existen `effort_records` ni las dos restricciones `UNIQUE` agregadas
- AND tras la reaplicación vuelven a existir

#### Scenario: El directorio de migraciones contiene 12 pares

- GIVEN el directorio de migraciones del proyecto
- WHEN se verifican sus archivos
- THEN contiene exactamente 12 pares `up`/`down`, el último llamado `000012_create_effort_records`

### Requirement: Habilitar la ruta de esfuerzo solo con el esquema en versión 12 o superior

El sistema MUST exponer la ruta de registro de esfuerzo únicamente cuando el esquema de almacenamiento esté en versión 12 o superior y sin estado inconsistente (`dirty`), mediante un flag `Effort` nuevo en la resolución de disponibilidad de migraciones, independiente de los flags existentes. Con el esquema en una versión inferior a 12, o en estado `dirty`, la ruta MUST NOT existir y una solicitud sobre ella MUST responder `404`. Esta condición MUST NOT alterar la disponibilidad de las rutas existentes de proyecto, historia, Sprint, integrantes, tareas ni Planning Poker.

#### Scenario: La ruta está disponible con el esquema en versión 12

- GIVEN el esquema de almacenamiento en versión 12 y sin estado inconsistente
- WHEN el sistema arranca y se solicita registrar esfuerzo válido
- THEN el sistema responde `201`

#### Scenario: La ruta no existe con el esquema en versión 11

- GIVEN el esquema de almacenamiento en versión 11 y sin estado inconsistente
- WHEN se solicita `POST` sobre la ruta de registro de esfuerzo
- THEN el sistema responde `404` porque la ruta no está registrada
- AND no se persiste ningún registro

#### Scenario: El esquema inconsistente deja la ruta no disponible

- GIVEN el esquema de almacenamiento en versión 12 o superior con estado inconsistente (`dirty`)
- WHEN el sistema arranca
- THEN la ruta de registro de esfuerzo no queda expuesta

#### Scenario: El flag de esfuerzo se resuelve por versión

- GIVEN la resolución de disponibilidad de migraciones
- WHEN se evalúa con versión 11, con versión 12, con versión 13 y con versión 12 en estado `dirty`
- THEN `Effort` es falso, verdadero, verdadero y falso respectivamente

#### Scenario: Las rutas existentes no cambian

- GIVEN el esquema de almacenamiento en versión 12 y sin estado inconsistente
- WHEN se solicitan las operaciones existentes de proyecto, historia, Sprint, integrantes, tareas y Planning Poker
- THEN todas responden igual que antes de habilitar la ruta de esfuerzo
