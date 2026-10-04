# Especificación de Historia

## Purpose

Registrar una historia en el Product Backlog de un proyecto existente, con criterios de aceptación y sin estimación inicial.

## Requirements

### Requirement: Crear una historia asociada a un proyecto conocido

El sistema MUST ofrecer una operación de creación que reciba el identificador de un proyecto conocido y los datos de la historia, sin requerir listado de proyectos ni una interfaz de selección. Ante una creación válida, MUST registrar exactamente una historia vinculada a ese proyecto y devolver su identificador generado junto con los datos de la historia creada.

#### Scenario: Registrar una historia en un proyecto existente

- GIVEN un proyecto existente cuyo identificador conoce quien solicita la creación
- AND se proporcionan título, descripción, prioridad `alta`, `media` o `baja` y una lista válida de criterios de aceptación
- WHEN se solicita crear la historia para ese identificador
- THEN el sistema registra exactamente una historia asociada a ese proyecto en su Product Backlog
- AND devuelve el identificador generado y los datos registrados de la historia

### Requirement: Validar los datos obligatorios de la historia

El sistema MUST exigir título y descripción no vacíos ni compuestos solo por espacios, una prioridad elegida entre `alta`, `media` y `baja`, y una lista de criterios de aceptación no vacía sin entradas vacías ni compuestas solo por espacios. MUST rechazar la creación sin persistir ninguna historia si falta o es inválido alguno de esos datos.

#### Scenario: Rechazar un campo obligatorio ausente o en blanco

- GIVEN un proyecto existente y una solicitud donde falta el título, la descripción o la prioridad, o donde alguno de esos campos está en blanco
- WHEN se solicita crear la historia
- THEN el sistema comunica el dato inválido
- AND no persiste ninguna historia

#### Scenario: Rechazar una prioridad no permitida

- GIVEN un proyecto existente y una solicitud cuya prioridad no es `alta`, `media` ni `baja`
- WHEN se solicita crear la historia
- THEN el sistema comunica que la prioridad es inválida
- AND no persiste ninguna historia

#### Scenario: Rechazar criterios de aceptación inválidos

- GIVEN un proyecto existente y una solicitud sin lista de criterios, con lista vacía o con al menos una entrada vacía o compuesta solo por espacios
- WHEN se solicita crear la historia
- THEN el sistema comunica que los criterios de aceptación son inválidos
- AND no persiste ninguna historia

### Requirement: Inicializar la historia sin estimación

El sistema MUST asignar `pendiente` como estado inicial, `null` como Story Points iniciales, nunca cero, y `null` como horas estimadas iniciales, nunca cero. La creación MUST conservar la prioridad elegida y cada criterio de aceptación proporcionado; MUST NOT aceptar una estimación de Story Points ni de horas como parte de esta operación. La estimación en Story Points corresponde a US-14 y la estimación de horas se registra únicamente mediante la operación de modificación. La historia registrada y el resultado de creación MUST incluir `estimated_hours` con valor `null`.
(Previously: solo fijaba estado `pendiente` y Story Points `null`; no existía la estimación de horas y el resultado de creación no la incluía.)

#### Scenario: Crear una historia pendiente de estimación

- GIVEN un proyecto existente y datos válidos con prioridad `media` y dos criterios de aceptación
- WHEN se crea la historia sin estimación
- THEN la historia registrada y el resultado de creación contienen prioridad `media`, estado `pendiente`, Story Points `null` y `estimated_hours` `null`
- AND se conservan los dos criterios de aceptación proporcionados

#### Scenario: No admitir una estimación inicial

- GIVEN un proyecto existente y datos de historia válidos que incluyen una estimación inicial de Story Points, incluso cero
- WHEN se solicita crear la historia
- THEN el sistema rechaza la estimación inicial
- AND no persiste ninguna historia estimada por esta operación

#### Scenario: No admitir horas estimadas en la creación

- GIVEN un proyecto existente y datos de historia válidos que incluyen la clave `estimated_hours`, con cualquier valor incluido `null`
- WHEN se solicita crear la historia
- THEN el sistema rechaza la solicitud como campo desconocido
- AND no persiste ninguna historia

#### Scenario: Una historia recién creada puede modificarse

- GIVEN una historia recién creada con `estimated_hours` `null`
- WHEN se modifica con `estimated_hours` `2`
- THEN el sistema responde `200` con `estimated_hours` `2`

### Requirement: Impedir historias sin proyecto existente

El sistema MUST rechazar la creación para un identificador de proyecto inexistente y MUST preservar la asociación de toda historia persistida con un proyecto existente. Si la asociación no puede asegurarse, incluso por un fallo o cambio concurrente durante la creación, MUST rechazar la operación sin persistir una historia huérfana. El rechazo MUST ser observable sin divulgar detalles internos.

#### Scenario: Rechazar un proyecto inexistente

- GIVEN un identificador de proyecto que no corresponde a ningún proyecto existente y datos de historia válidos
- WHEN se solicita crear la historia
- THEN el sistema rechaza la creación indicando que el proyecto no existe
- AND no persiste ninguna historia

#### Scenario: Fallar sin dejar historias huérfanas

- GIVEN un proyecto cuyo vínculo deja de poder asegurarse durante la creación por un fallo o cambio concurrente
- WHEN se intenta registrar una historia para ese proyecto
- THEN el sistema rechaza la operación sin divulgar detalles internos
- AND no queda persistida ninguna historia sin un proyecto existente
### Requirement: Modificar una historia existente de un proyecto existente

El sistema MUST ofrecer una operación de modificación, `PUT /projects/{project_id}/stories/{story_id}`, que reemplace por completo el contenido editable de una historia existente que pertenezca al proyecto indicado. Ante una modificación válida, MUST persistir exactamente los seis campos editables recibidos y MUST responder `200` con la historia completa tal como quedó almacenada: `id`, `project_id`, `title`, `description`, `priority`, `status`, `story_points`, `acceptance_criteria` y `estimated_hours`. La operación MUST preservar el orden de los criterios de aceptación y MUST reemplazar el valor almacenado de cada campo editable, sin fusionarlo con el anterior.

#### Scenario: Modificar los seis campos de una historia existente

- GIVEN un proyecto existente y una historia suya con título "Login", descripción "Iniciar sesión", prioridad `baja`, estado `pendiente`, dos criterios de aceptación y sin horas estimadas
- WHEN se solicita `PUT` sobre la ruta de esa historia con título "Login con email", descripción "Iniciar sesión con email y contraseña", prioridad `alta`, estado `en_progreso`, criterios `["Valida el email", "Rechaza contraseña incorrecta", "Bloquea tras 5 intentos"]` y `estimated_hours` `8.5`
- THEN el sistema responde `200`
- AND el cuerpo contiene la historia completa con los seis valores enviados y los criterios en el mismo orden en que se enviaron
- AND una lectura posterior de la historia almacenada devuelve esos mismos valores

#### Scenario: La respuesta incluye los campos no editables intactos

- GIVEN una historia existente con `story_points` `5`
- WHEN se modifica con datos válidos
- THEN la respuesta `200` incluye el mismo `id`, el mismo `project_id` y `story_points` `5`, sin cambios

#### Scenario: Reemplazo completo de los criterios de aceptación

- GIVEN una historia existente con tres criterios de aceptación
- WHEN se modifica enviando un único criterio distinto
- THEN la historia almacenada tiene exactamente ese único criterio
- AND los tres criterios anteriores ya no existen

#### Scenario: Modificar sin cambiar ningún valor

- GIVEN una historia existente
- WHEN se solicita `PUT` con exactamente los mismos valores que ya tiene almacenados
- THEN el sistema responde `200` con la misma historia
- AND el contenido almacenado no cambia

#### Scenario: Modificar dos veces seguidas conserva la última escritura

- GIVEN una historia existente
- WHEN se envían dos modificaciones válidas consecutivas con contenido distinto
- THEN la historia almacenada refleja íntegramente el contenido de la segunda modificación
- AND el sistema no ofrece control de concurrencia ni versionado: ante modificaciones simultáneas gana la última escritura

### Requirement: Exigir los seis campos editables en la modificación

El sistema MUST exigir que el cuerpo de la modificación contenga las seis claves `title`, `description`, `priority`, `status`, `acceptance_criteria` y `estimated_hours`. La clave `estimated_hours` MUST estar presente, pero su valor MAY ser `null`. Una clave faltante MUST rechazarse con `422 validation_failed` indicando en `fields` la clave ausente, y MUST NOT interpretarse como "conservar el valor actual".

#### Scenario: Rechazar la ausencia de una clave obligatoria

- GIVEN una historia existente y un cuerpo que omite exactamente una de las claves `title`, `description`, `priority`, `status` o `acceptance_criteria`
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed`
- AND `fields` contiene la clave omitida con el motivo `is required`
- AND la historia almacenada no cambia

#### Scenario: Rechazar la ausencia de `estimated_hours`

- GIVEN una historia existente con `estimated_hours` `4` y un cuerpo con las otras cinco claves válidas pero sin la clave `estimated_hours`
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed` con `fields.estimated_hours` igual a `is required`
- AND la historia conserva `estimated_hours` `4`

#### Scenario: Aceptar `estimated_hours` presente con valor nulo

- GIVEN una historia existente y un cuerpo con las seis claves, donde `estimated_hours` es `null`
- WHEN se solicita la modificación
- THEN el sistema responde `200`
- AND la historia devuelta tiene `estimated_hours` `null`

#### Scenario: Reportar todas las claves faltantes a la vez

- GIVEN una historia existente y un cuerpo `{}` (objeto JSON vacío)
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed`
- AND `fields` contiene las seis claves con el motivo `is required`
- AND la historia almacenada no cambia

### Requirement: Reutilizar las reglas de validación de contenido de la creación

El sistema MUST aplicar en la modificación las mismas reglas de contenido que rigen la creación: `title` y `description` no vacíos ni compuestos solo por espacios, `priority` exactamente `alta`, `media` o `baja`, y `acceptance_criteria` una lista no vacía sin entradas vacías ni compuestas solo por espacios. Cualquier violación MUST rechazarse con `422 validation_failed` e indicar en `fields` cada dato inválido, sin escribir en el almacenamiento. Las reglas MUST evaluarse antes de cualquier acceso al almacenamiento, de modo que una solicitud inválida no produzca ninguna escritura ni ninguna consulta de existencia.

#### Scenario: Rechazar título o descripción vacíos o en blanco

- GIVEN una historia existente y un cuerpo con `title` `""` o `"   "` (o `description` con esos valores) y el resto de los datos válidos
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed` con `fields.title` (o `fields.description`) indicando que no puede estar vacío
- AND la historia almacenada no cambia

#### Scenario: Rechazar todos los campos de contenido inválidos juntos

- GIVEN una historia existente y un cuerpo con título vacío, descripción en blanco, prioridad `urgente` y criterios `[]`
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed`
- AND `fields` contiene entradas para `title`, `description`, `priority` y `acceptance_criteria`

#### Scenario: Rechazar una prioridad no permitida

- GIVEN una historia existente y un cuerpo con `priority` `"urgente"`, `"ALTA"` o `""`
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed` con `fields.priority`
- AND la historia almacenada no cambia

#### Scenario: Rechazar criterios de aceptación vacíos

- GIVEN una historia existente y un cuerpo con `acceptance_criteria` igual a `[]`
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed` con `fields.acceptance_criteria`
- AND la historia conserva sus criterios anteriores

#### Scenario: Rechazar un criterio vacío o en blanco

- GIVEN una historia existente y un cuerpo con `acceptance_criteria` igual a `["Criterio válido", "   "]` o `["Criterio válido", ""]`
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed` con `fields.acceptance_criteria`
- AND la historia conserva sus criterios anteriores

#### Scenario: Una solicitud inválida no llega al almacenamiento

- GIVEN cualquier cuerpo que viole una regla de validación de contenido, de estado o de estimación
- WHEN se solicita la modificación
- THEN el sistema rechaza la solicitud antes de consultar o escribir en el almacenamiento
- AND no se ejecuta ninguna escritura sobre ninguna historia

#### Scenario: Conservar el texto enviado sin alterarlo

- GIVEN una historia existente y un cuerpo válido cuyo título contiene espacios internos y caracteres Unicode
- WHEN se solicita la modificación
- THEN el título devuelto y almacenado coincide con el enviado

### Requirement: Restringir el estado a un conjunto cerrado sin reglas de transición

El sistema MUST aceptar como `status` únicamente `pendiente`, `en_progreso` o `completada`, con esa escritura exacta. Cualquier otro valor, incluida la cadena vacía, MUST rechazarse con `422 validation_failed` e indicar `fields.status`, sin escribir. Cualquier valor del conjunto MUST poder reemplazar a cualquier otro, incluido el mismo valor: el sistema MUST NOT imponer reglas de transición entre estados. El conjunto cerrado MUST garantizarse también en el almacenamiento, de modo que no pueda persistirse un estado fuera de él.

#### Scenario: Aceptar cada estado permitido

- GIVEN una historia existente en estado `pendiente`
- WHEN se modifica con `status` `en_progreso`, luego con `completada` y luego con `pendiente`
- THEN cada modificación responde `200` con el estado enviado
- AND el estado almacenado tras cada una coincide con el enviado

#### Scenario: Permitir una transición retrógrada

- GIVEN una historia en estado `completada`
- WHEN se modifica con `status` `pendiente`
- THEN el sistema responde `200`
- AND la historia queda en estado `pendiente`

#### Scenario: Permitir saltar directamente entre estados

- GIVEN una historia en estado `pendiente`
- WHEN se modifica con `status` `completada` sin pasar por `en_progreso`
- THEN el sistema responde `200` con estado `completada`

#### Scenario: Mantener el mismo estado

- GIVEN una historia en estado `en_progreso`
- WHEN se modifica con `status` `en_progreso`
- THEN el sistema responde `200` con estado `en_progreso`

#### Scenario: Rechazar un estado desconocido

- GIVEN una historia existente y un cuerpo con `status` `"cancelada"`, `"Completada"`, `"en progreso"` o `""`
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed` con `fields.status`
- AND la historia almacenada conserva su estado anterior

#### Scenario: El almacenamiento rechaza un estado fuera del conjunto

- GIVEN el almacenamiento con el esquema en versión 3
- WHEN se intenta persistir directamente una historia con un estado fuera de `pendiente`, `en_progreso` o `completada`
- THEN el almacenamiento rechaza la escritura
- AND no queda persistido el estado inválido

### Requirement: Estimar las horas de una historia con un valor opcional y positivo

El sistema MUST permitir registrar en una historia una estimación de esfuerzo `estimated_hours`, distinta e independiente de los Story Points, que MAY estar ausente (`null`). Cuando esté presente MUST ser un número JSON mayor que `0`, con como máximo dos decimales y menor o igual que `99999.99`. El valor `0`, los negativos, los valores con más de dos decimales, los valores mayores que `99999.99` y los valores no numéricos MUST rechazarse sin escribir: los valores numéricos fuera de rango o precisión con `422 validation_failed` e indicando `fields.estimated_hours`, y los tipos incorrectos con `400 invalid_request`. Enviar `null` MUST borrar la estimación almacenada. La estimación MUST recuperarse con el mismo valor con que se guardó, incluidos los bordes `0.01` y `99999.99`. La modificación de horas MUST NOT depender de los Story Points de la historia ni alterarlos.

#### Scenario: Registrar una estimación válida

- GIVEN una historia existente sin horas estimadas
- WHEN se modifica con `estimated_hours` `12.5`
- THEN el sistema responde `200` con `estimated_hours` `12.5`
- AND el valor almacenado es `12.5`

#### Scenario: Aceptar el valor mínimo permitido

- GIVEN una historia existente
- WHEN se modifica con `estimated_hours` `0.01`
- THEN el sistema responde `200` con `estimated_hours` `0.01`
- AND una lectura posterior del valor almacenado devuelve exactamente `0.01`

#### Scenario: Aceptar el valor máximo permitido

- GIVEN una historia existente
- WHEN se modifica con `estimated_hours` `99999.99`
- THEN el sistema responde `200` con `estimated_hours` `99999.99`
- AND una lectura posterior del valor almacenado devuelve exactamente `99999.99`

#### Scenario: Aceptar un entero y un valor con un decimal

- GIVEN una historia existente
- WHEN se modifica con `estimated_hours` `3` y luego con `2.5`
- THEN ambas modificaciones responden `200` con el valor enviado

#### Scenario: Rechazar cero

- GIVEN una historia existente con `estimated_hours` `5`
- WHEN se modifica con `estimated_hours` `0`
- THEN el sistema responde `422 validation_failed` con `fields.estimated_hours`
- AND la historia conserva `estimated_hours` `5`

#### Scenario: Rechazar valores negativos

- GIVEN una historia existente
- WHEN se modifica con `estimated_hours` `-1` o `-0.01`
- THEN el sistema responde `422 validation_failed` con `fields.estimated_hours`
- AND la historia almacenada no cambia

#### Scenario: Rechazar más de dos decimales

- GIVEN una historia existente
- WHEN se modifica con `estimated_hours` `1.001` o `0.005`
- THEN el sistema responde `422 validation_failed` con `fields.estimated_hours`
- AND la historia almacenada no cambia

#### Scenario: Rechazar un valor por encima del máximo

- GIVEN una historia existente
- WHEN se modifica con `estimated_hours` `100000` o `99999.991`
- THEN el sistema responde `422 validation_failed` con `fields.estimated_hours`
- AND la historia almacenada no cambia

#### Scenario: Rechazar un tipo no numérico

- GIVEN una historia existente
- WHEN se modifica con `estimated_hours` igual a la cadena `"8"`, a `true` o a un objeto
- THEN el sistema responde `400 invalid_request`
- AND la historia almacenada no cambia

#### Scenario: Borrar la estimación con `null`

- GIVEN una historia existente con `estimated_hours` `8`
- WHEN se modifica con `estimated_hours` `null` y el resto de los datos válidos
- THEN el sistema responde `200` con `estimated_hours` `null`
- AND el valor almacenado pasa a ser `null`, no `0`

#### Scenario: Las horas son independientes de los Story Points

- GIVEN una historia con `story_points` `8`
- WHEN se modifica con `estimated_hours` `1.5`
- THEN el sistema responde `200` con `estimated_hours` `1.5` y `story_points` `8`
- AND no se aplica ninguna regla que relacione las horas con los Story Points

### Requirement: Preservar la identidad y los Story Points en la modificación

El sistema MUST NOT modificar `id`, `project_id` ni `story_points` en la operación de modificación. La operación MUST rechazar con `400 invalid_request` un cuerpo que contenga cualquiera de esas tres claves, como campos desconocidos, sin escribir. `id` y `project_id` MUST tomarse exclusivamente de la ruta. La estimación en Story Points corresponde a US-14 y MUST NOT aceptarse aquí.

#### Scenario: Rechazar `id` en el cuerpo

- GIVEN una historia existente y un cuerpo válido que además incluye la clave `id`
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia almacenada no cambia

#### Scenario: Rechazar `project_id` en el cuerpo

- GIVEN una historia existente y un cuerpo válido que además incluye la clave `project_id`, incluso con el identificador del propio proyecto o de otro proyecto existente
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia permanece asociada a su proyecto original

#### Scenario: Rechazar `story_points` en el cuerpo

- GIVEN una historia existente y un cuerpo válido que además incluye la clave `story_points`, con un entero, con cero o con `null`
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia conserva su valor previo de `story_points`

#### Scenario: Una modificación válida deja intactos identidad y Story Points

- GIVEN una historia existente con `story_points` `13` (asignados fuera de esta operación) y un `id` y `project_id` conocidos
- WHEN se modifica con seis campos válidos
- THEN la historia almacenada conserva el mismo `id`, el mismo `project_id` y `story_points` `13`

### Requirement: Rechazar la modificación de una historia inexistente o de otro proyecto

El sistema MUST responder `404` con el código `story_not_found` cuando `story_id` no corresponde a ninguna historia, o cuando corresponde a una historia que pertenece a un proyecto distinto del indicado en la ruta. Ambos casos MUST ser indistinguibles para quien consulta, MUST NOT divulgar detalles internos y MUST NOT modificar ninguna fila. El sistema MUST asegurar la pertenencia de la historia al proyecto en la misma operación de escritura, sin una verificación previa separada.

#### Scenario: Historia inexistente

- GIVEN un proyecto existente y un `story_id` con formato válido que no corresponde a ninguna historia
- WHEN se solicita la modificación con un cuerpo válido
- THEN el sistema responde `404` con el código `story_not_found`
- AND no se modifica ninguna historia

#### Scenario: Historia de otro proyecto

- GIVEN dos proyectos A y B, y una historia S que pertenece al proyecto B
- WHEN se solicita modificar S usando la ruta del proyecto A con un cuerpo válido
- THEN el sistema responde `404` con el código `story_not_found`
- AND la historia S conserva íntegramente su contenido y su asociación al proyecto B

#### Scenario: Proyecto inexistente

- GIVEN un `project_id` con formato válido que no corresponde a ningún proyecto y un `story_id` con formato válido
- WHEN se solicita la modificación con un cuerpo válido
- THEN el sistema responde `404` con el código `story_not_found`
- AND no se modifica ninguna historia

#### Scenario: La respuesta no revela si la historia existe en otro proyecto

- GIVEN una historia S del proyecto B y una historia inexistente
- WHEN se solicita modificar cada una usando la ruta del proyecto A
- THEN ambas respuestas tienen el mismo código HTTP y el mismo código de error `story_not_found`

### Requirement: Rechazar identificadores inválidos y datos inválidos con detalle de campos

El sistema MUST responder `422` con el código `validation_failed` y el detalle `fields` para todo dato inválido, y MUST NOT escribir en el almacenamiento en ninguno de esos casos. Esto incluye un `project_id` o un `story_id` de la ruta que no sean UUID válidos, una clave obligatoria faltante y toda violación de las reglas de contenido, estado y estimación. Un identificador inválido en la ruta MUST reportarse con `fields.project_id` o `fields.story_id` según corresponda, y MUST resolverse antes de consultar la existencia de la historia.

#### Scenario: `story_id` no es un UUID válido

- GIVEN un proyecto existente y un `story_id` de ruta igual a `"abc"`
- WHEN se solicita la modificación con un cuerpo válido
- THEN el sistema responde `422 validation_failed` con `fields.story_id`
- AND no se escribe nada

#### Scenario: `project_id` no es un UUID válido

- GIVEN un `project_id` de ruta igual a `"no-es-uuid"` y un `story_id` con formato válido
- WHEN se solicita la modificación con un cuerpo válido
- THEN el sistema responde `422 validation_failed` con `fields.project_id`
- AND no se escribe nada

#### Scenario: Ambos identificadores inválidos

- GIVEN un `project_id` y un `story_id` de ruta que no son UUID válidos
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed`
- AND `fields` contiene `project_id` y `story_id`

#### Scenario: Identificador inválido y cuerpo inválido a la vez

- GIVEN un `story_id` inválido y un cuerpo con título vacío
- WHEN se solicita la modificación
- THEN el sistema responde `422 validation_failed`
- AND `fields` incluye `story_id` y `title`

#### Scenario: Identificadores válidos pero inexistentes no son un error de validación

- GIVEN un `story_id` con formato UUID válido que no existe
- WHEN se solicita la modificación con un cuerpo válido
- THEN el sistema responde `404 story_not_found` y no `422`

### Requirement: Rechazar solicitudes malformadas de modificación

El sistema MUST responder `400` con el código `invalid_request` cuando el cuerpo no sea un único objeto JSON válido: JSON malformado, cuerpo vacío, un valor JSON que no sea objeto, varios valores JSON en el mismo cuerpo, claves desconocidas, o tipos incorrectos en algún campo (por ejemplo `title` numérico, `acceptance_criteria` que no sea una lista de cadenas, o `null` dentro de la lista de criterios). MUST NOT escribir en el almacenamiento y MUST NOT divulgar detalles internos.

#### Scenario: JSON malformado

- GIVEN una historia existente y un cuerpo que no es JSON válido, por ejemplo `{"title":`
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia almacenada no cambia

#### Scenario: Cuerpo vacío o no objeto

- GIVEN una historia existente y un cuerpo vacío, o un valor JSON que no es un objeto, por ejemplo `[]` o `"texto"`
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`

#### Scenario: Varios valores JSON en el cuerpo

- GIVEN una historia existente y un cuerpo con dos objetos JSON consecutivos
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia almacenada no cambia

#### Scenario: Campo desconocido

- GIVEN una historia existente y un cuerpo válido que además incluye la clave `owner`
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia almacenada no cambia

#### Scenario: Tipo incorrecto en un campo

- GIVEN una historia existente y un cuerpo con `title` numérico, o con `acceptance_criteria` igual a la cadena `"criterio"`
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia almacenada no cambia

#### Scenario: Valor nulo dentro de los criterios de aceptación

- GIVEN una historia existente y un cuerpo con `acceptance_criteria` igual a `["Criterio", null]`
- WHEN se solicita la modificación
- THEN el sistema responde `400 invalid_request`
- AND la historia conserva sus criterios anteriores

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

### Requirement: Fallar sin divulgar detalles internos ni modificar datos ante errores del almacenamiento

El sistema MUST responder `500` con el código `internal_error` y un mensaje genérico cuando la modificación no pueda completarse por un fallo inesperado del almacenamiento. MUST NOT confundir un fallo inesperado con una historia no encontrada, ni divulgar detalles internos. Un fallo MUST NOT dejar la historia parcialmente modificada.

#### Scenario: Fallo inesperado del almacenamiento

- GIVEN una historia existente y un fallo inesperado del almacenamiento durante la escritura
- WHEN se solicita una modificación válida
- THEN el sistema responde `500` con el código `internal_error` y un mensaje genérico
- AND la respuesta no contiene detalles internos del fallo

#### Scenario: Un fallo no se reporta como historia inexistente

- GIVEN una historia existente y un error de almacenamiento distinto de "sin filas afectadas"
- WHEN se solicita una modificación válida
- THEN el sistema no responde `404 story_not_found`

#### Scenario: La modificación es atómica

- GIVEN una historia existente y una modificación válida
- WHEN la modificación se completa
- THEN todos los campos editables reflejan los nuevos valores a la vez
- AND si la operación falla, la historia conserva íntegramente su contenido anterior

### Requirement: Disponibilidad de la modificación condicionada a la versión del esquema

El sistema MUST exponer la operación de modificación únicamente cuando el esquema de almacenamiento esté en versión 3 o superior y sin estado inconsistente (`dirty`). Con el esquema en versión 2, la ruta de modificación MUST NOT existir (responde `404`), y la creación de historias MUST seguir funcionando sin cambios. La creación de historias MUST mantener su condición de disponibilidad de versión 2 o superior.

#### Scenario: La modificación está disponible con el esquema en versión 3

- GIVEN el esquema de almacenamiento en versión 3 y sin estado inconsistente
- WHEN el sistema arranca y se solicita una modificación válida de una historia existente
- THEN el sistema responde `200`

#### Scenario: La modificación no existe con el esquema en versión 2

- GIVEN el esquema de almacenamiento en versión 2
- WHEN se solicita `PUT` sobre `/projects/{project_id}/stories/{story_id}`
- THEN el sistema responde `404` porque la ruta no está registrada
- AND ninguna historia se modifica

#### Scenario: La creación sigue funcionando con el esquema en versión 2

- GIVEN el esquema de almacenamiento en versión 2 y un proyecto existente
- WHEN se solicita crear una historia válida
- THEN el sistema registra la historia y responde con éxito

#### Scenario: Esquema en estado inconsistente

- GIVEN el esquema de almacenamiento en versión 3 con estado inconsistente (`dirty`)
- WHEN el sistema arranca
- THEN la ruta de modificación no queda expuesta
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
