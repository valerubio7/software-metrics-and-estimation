# Especificación de Defecto

## Propósito

Permitir que un proyecto registre un defecto detectado durante el desarrollo, indicando su descripción, severidad, estado, historia relacionada y Sprint de detección. El defecto queda persistido y asociado a su proyecto, a esa historia y a ese Sprint; la resolución y su Sprint correspondiente no forman parte de este cambio (US-17), y tampoco el listado de defectos (US-18), el cálculo de métricas de defectos (US-19) ni la operación desde la TUI (TUI-09). Esta capacidad solo registra.

## ADDED Requirements

### Requirement: Registrar un defecto de un proyecto

El sistema MUST ofrecer `POST /projects/{project_id}/defects` para registrar un defecto detectado durante el desarrollo dentro del proyecto identificado en la ruta. El cuerpo MUST ser un único objeto JSON con las claves `story_id`, `sprint_id`, `description`, `severity` y `status` obligatorias. `story_id` identifica la historia relacionada y `sprint_id` el Sprint de detección, ambos pertenecientes a ese proyecto. Ante una solicitud válida, el sistema MUST persistir exactamente un defecto asociado a ese proyecto, a esa historia y a ese Sprint, y MUST responder `201` con el registro creado. El cuerpo de la respuesta MUST incluir el identificador generado por el servidor, `project_id`, `story_id`, `sprint_id`, `description` ya recortada, `severity` y `status`. La respuesta MUST NOT exponer marcas internas de creación. La ruta MUST responder `405 method_not_allowed` para métodos distintos de `POST`.

#### Scenario: Registrar un defecto de forma válida

- GIVEN un proyecto existente, una historia de ese proyecto y un Sprint de ese proyecto
- AND una solicitud con `story_id` de la historia, `sprint_id` del Sprint, `description` igual a `"El login acepta contraseñas vacías"`, `severity` igual a `"alta"` y `status` igual a `"detectado"`
- WHEN se solicita registrar el defecto en ese proyecto
- THEN el sistema responde `201`
- AND persiste exactamente un defecto con `project_id` del proyecto, `story_id` del cuerpo, `sprint_id` del cuerpo, `description` `"El login acepta contraseñas vacías"`, `severity` `"alta"` y `status` `"detectado"`
- AND el cuerpo de la respuesta incluye un identificador generado por el servidor y esos mismos valores

#### Scenario: La respuesta `201` replica los datos del alta

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida con `description` con espacios sobrantes en los extremos
- WHEN se solicita registrar el defecto
- THEN el sistema responde `201`
- AND el cuerpo de la respuesta incluye `project_id`, `story_id`, `sprint_id`, `severity` y `status` tal como fueron solicitados
- AND la `description` de la respuesta no tiene espacios sobrantes en los extremos
- AND la respuesta no incluye `created_at` ni ninguna otra marca interna de creación

#### Scenario: El método distinto de `POST` no registra defectos

- GIVEN la ruta de registro de defectos
- AND una solicitud con método `GET` o `PUT`
- WHEN se solicita la ruta
- THEN el sistema responde `405 method_not_allowed`
- AND no persiste ningún defecto

### Requirement: Asignar el identificador del defecto en el servidor

El sistema MUST generar el identificador del defecto en el servidor por solicitud y MUST NOT aceptar `id` ni `created_at` como campos del cuerpo. El identificador generado MUST ser un UUID del servidor, MUST aparecer en el registro persistido y en la respuesta `201`, y MUST ser distinto para solicitudes consecutivas. El sistema conserva `created_at` en el almacenamiento como marca interna de la creación y no lo expone en el contrato.

#### Scenario: El identificador lo genera el servidor

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida que no incluye ni `id` ni `created_at`
- WHEN se solicita registrar el defecto
- THEN el sistema responde `201`
- AND el identificador del registro devuelto y persistido es un UUID generado por el servidor
- AND dos registros consecutivos obtienen identificadores distintos

#### Scenario: Un cuerpo con `id` es rechazado como campo desconocido

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida que además incluye la clave `id`
- WHEN se solicita registrar el defecto
- THEN el sistema responde `400 invalid_request`
- AND no persiste ningún defecto

### Requirement: Validar la severidad contra un catálogo cerrado

El sistema MUST aceptar solo los valores exactos `alta`, `media` y `baja` para `severity`. Un `severity` ausente, con espacios, con mayúsculas o fuera del catálogo MUST rechazarse con `422 validation_failed`, indicando el campo `severity` en `fields`, y MUST NOT persistir ningún defecto. El valor aceptado MUST persistirse sin transformarlo.

#### Scenario: Aceptar las tres severidades del catálogo

- GIVEN un proyecto, una historia y un Sprint válidos
- AND tres solicitudes válidas idénticas salvo `severity` igual a `"alta"`, `"media"` y `"baja"` respectivamente
- WHEN se solicita registrar cada defecto
- THEN el sistema responde `201` en las tres solicitudes
- AND cada registro persiste la severidad solicitada sin transformarla

#### Scenario: Rechazar una severidad fuera del catálogo

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida salvo `severity` igual a `"grave"`, `""`, `"Alta"`, `" alta"` o `"crítica"`
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `severity` como el campo inválido
- AND no se persiste ningún defecto

### Requirement: Validar el estado contra el subconjunto inicial del alta

El sistema MUST aceptar solo los valores exactos `detectado` y `en_correccion` como `status` del alta. Un `status` igual a `"resuelto"`, fuera del catálogo o ausente MUST rechazarse con `422 validation_failed`, indicando el campo `status` en `fields`, y MUST NOT persistir ningún defecto. La transición a `resuelto`, con su Sprint de resolución, pertenece a US-17: ningún registro creado por esta capacidad queda en estado `resuelto` y el catálogo completo de la base (`detectado`, `en_correccion`, `resuelto`)��ece ser válido como restricción de almacenamiento sin que el alta lo aplique por sí sola.

#### Scenario: Aceptar los dos estados iniciales del catálogo

- GIVEN un proyecto, una historia y un Sprint válidos
- AND dos solicitudes válidas idénticas salvo `status` igual a `"detectado"` y `"en_correccion"` respectivamente
- WHEN se solicita registrar cada defecto
- THEN el sistema responde `201` en las dos solicitudes
- AND cada registro persiste el estado solicitado sin transformarlo

#### Scenario: Rechazar el alta de un defecto ya resuelto

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida salvo `status` igual a `"resuelto"`
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `status` como el campo inválido
- AND no se persiste ningún defecto

#### Scenario: Rechazar un estado fuera del catálogo o ausente

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida salvo `status` igual a `"abierto"`, `""`, `" detectado"` o sin la clave
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `status` como el campo inválido
- AND no se persiste ningún defecto

### Requirement: Validar la descripción del defecto

El sistema MUST exigir una `description` presente que, tras recortar espacios de los extremos, tenga entre 1 y 500 caracteres Unicode (contados como caracteres, no como bytes). Un `description` ausente, vacío, compuesto solo por espacios, de más de 500 caracteres o que contenga el carácter NUL (`0x00`) MUST rechazarse con `422 validation_failed`, indicando el campo `description` en `fields`, y MUST NOT persistir ningún defecto. El valor persistido y devuelto MUST ser la descripción recortada, con sus caracteres internos intactos.

#### Scenario: Persistir la descripción recortada

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida con `description` igual a `"  Error al calcular métricas  "`
- WHEN se solicita registrar el defecto
- THEN el sistema responde `201`
- AND la descripción persistida y devuelta es `"Error al calcular métricas"`, sin los espacios de los extremos

#### Scenario: Rechazar una descripción vacía o ausente

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud con `description` ausente, igual a `""` o igual a `"   "`
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `description` como el campo inválido
- AND no se persiste ningún defecto

#### Scenario: Rechazar una descripción de más de 500 caracteres

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida con `description` de 501 caracteres, contando cada carácter multi-byte como un carácter Unicode
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `description` como el campo inválido
- AND no se persiste ningún defecto

#### Scenario: Rechazar una descripción que contiene NUL

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida con `description` que contiene el carácter NUL
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `description` como el campo inválido
- AND no se persiste ningún defecto

### Requirement: Validar los identificadores del cuerpo

El sistema MUST exigir `story_id` y `sprint_id` presentes y con formato de UUID válido. Un valor ausente o vacío de un campo obligatorio MUST reportar el motivo de requerido y un valor presente con formato inválido MUST reportar el motivo de UUID inválido; en ambos casos con `422 validation_failed`, indicando el campo correspondiente en `fields`, y sin persistir nada. Cuando un identificador es válido, el sistema MUST usar su forma canónica minúscula en la persistencia.

#### Scenario: Rechazar identificadores vacíos o faltantes

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud con `story_id` o `sprint_id` ausente o igual a `""`
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica cada campo vacío con el motivo de requerido
- AND no se persiste ningún defecto

#### Scenario: Rechazar identificadores con formato inválido

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud con `story_id` o `sprint_id` presente pero con un formato inválido de UUID
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica cada campo con el motivo de UUID inválido
- AND no se persiste ningún defecto

#### Scenario: Normalizar los identificadores al formato canónico

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud con `story_id` y `sprint_id` con letras mayúsculas en los caracteres hexadecimales del UUID
- WHEN se solicita registrar el defecto
- THEN el sistema responde `201`
- AND persiste el registro con la forma canónica minúscula de esos identificadores

### Requirement: Reportar todos los errores de campo juntos

El sistema MUST acumular los errores de validación de todos los campos de una misma solicitud en un único `422 validation_failed` con `fields` que reporta cada campo y su motivo, en vez de reportar solo el primer error. El sistema MUST verificar los errores del cuerpo antes de la verificación de referencias en el almacenamiento: una solicitud con errores de campo produce `422 validation_failed` y no produce `404` de referencia.

#### Scenario: Acumular errores de varios campos

- GIVEN un proyecto existente
- AND una solicitud con un `story_id` con formato inválido, `sprint_id` vacío, `description` compuesta solo por espacios y `severity` fuera del catálogo
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed`
- AND `fields` reporta los errores de `story_id`, `sprint_id`, `description` y `severity` juntos
- AND no se persiste ningún defecto

#### Scenario: Los errores del cuerpo no producen verificación de referencias

- GIVEN un proyecto para el que la verificación de referencias produciría un `404`
- AND una solicitud con un campo del cuerpo inválido
- WHEN se solicita registrar el defecto
- THEN el sistema responde `422 validation_failed` con los errores del cuerpo
- AND la respuesta no es `404` y no se persiste nada

### Requirement: Rechazar solicitudes malformadas

El sistema MUST aceptar solo un objeto JSON como cuerpo y MUST rechazar con `400 invalid_request` y sin persistir nada una solicitud cuyo cuerpo está ausente, vacío, no es JSON válido, no es un objeto, contiene contenido después del objeto, contiene claves desconocidas o contiene valores de tipo incompatible con el contrato del cuerpo. El error `400` precede al análisis de los campos del objeto rechazado y no revela detalles internos.

#### Scenario: Rechazar un cuerpo roto

- GIVEN la ruta de registro de defectos
- AND una solicitud con un cuerpo vacío, texto arbitrario, un JSON cortado a la mitad o un arreglo en lugar de un objeto
- WHEN se solicita registrar el defecto
- THEN el sistema responde `400 invalid_request`
- AND no persiste ningún defecto

#### Scenario: Rechazar contenido después del objeto

- GIVEN la ruta de registro de defectos
- AND una solicitud cuyo cuerpo es un objeto JSON válido seguido de más contenido
- WHEN se solicita registrar el defecto
- THEN el sistema responde `400 invalid_request`
- AND no persiste ningún defecto

#### Scenario: Rechazar claves desconocidas o de tipo incompatible

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida que además incluye una clave que no existe en el contrato del cuerpo, o una solicitud válida donde `description` es un número en lugar de texto
- WHEN se solicita registrar el defecto
- THEN el sistema responde `400 invalid_request`
- AND no persiste ningún defecto

### Requirement: Verificar las referencias del defecto dentro del proyecto

El sistema MUST verificar, dentro de la misma transacción que el alta y con precedencia determinista, la existencia del proyecto identificado en la ruta, la pertenencia de la historia (`story_id`) al proyecto y la pertenencia del Sprint de detección (`sprint_id`) al proyecto. Un proyecto inexistente, una historia inexistente o de otro proyecto y un Sprint inexistente o de otro proyecto MUST rechazar el registro con `404` y MUST NOT persistir nada. El orden de los errores de referencia MUST respetar la precedencia: `project_not_found` primero, `story_not_found` después y `sprint_not_found` al final, sin importar el orden de presentación de los valores en el cuerpo. La integridad exigida es por proyecto únicamente: el sistema NO DEBE exigir que la historia esté asignada al Sprint de detección, y nada de este contrato depende de esa coherencia.

#### Scenario: Rechazar un proyecto inexistente

- GIVEN una ruta con un identificador de proyecto que no existe o es inválido
- AND una solicitud válida con historia y Sprint de un proyecto existente
- WHEN se solicita registrar el defecto
- THEN el sistema responde `404 project_not_found`
- AND no persiste ningún defecto

#### Scenario: Rechazar una historia que no pertenece al proyecto

- GIVEN un proyecto existente y un Sprint de ese proyecto
- AND una solicitud válida con un `story_id` de una historia de otro proyecto o inexistente
- WHEN se solicita registrar el defecto
- THEN el sistema responde `404 story_not_found`
- AND no persiste ningún defecto

#### Scenario: Rechazar un Sprint que no pertenece al proyecto

- GIVEN un proyecto existente y una historia de ese proyecto
- AND una solicitud válida con un `sprint_id` de un Sprint de otro proyecto o inexistente
- WHEN se solicita registrar el defecto
- THEN el sistema responde `404 sprint_not_found`
- AND no persiste ningún defecto

#### Scenario: Respetar la precedencia de los errores de referencia

- GIVEN un proyecto de ruta inexistente
- AND una solicitud válida con una historia y un Sprint de otro proyecto
- WHEN se solicita registrar el defecto
- THEN el sistema responde `404 project_not_found`

- GIVEN un proyecto de ruta existente
- AND una solicitud válida cuyo `story_id` y `sprint_id` pertenecen a otro proyecto
- WHEN se solicita registrar el defecto
- THEN el sistema responde `404 story_not_found`
- AND la precedencia nunca depende del orden interno de consulta del almacenamiento

#### Scenario: La verificación de referencias y el alta comparten transacción

- GIVEN un proyecto, una historia y un Sprint válidos al iniciar la verificación
- AND una referencia que desaparece tras la verificación y antes del alta
- WHEN se solicita registrar el defecto
- THEN el sistema o registra el defecto completo o responde el `404` del error de referencia que dejó de cumplirse
- AND no persiste un defecto que referencie una historia o un Sprint que ya no cumplen las reglas del proyecto

### Requirement: Aceptar historias en cualquier situación sin alterar datos

El sistema MUST aceptar el registro de un defecto con una historia asignada a un Sprint cerrado y con una historia sin asignación a ningún Sprint (solo en el backlog). El alta MUST escribir únicamente una fila en su tabla de defectos y MUST NOT modificar `stories`, `sprints`, `sprint_stories`, `tasks`, `project_members`, `effort_records`, el cierre de ningún Sprint (US-12) ni ninguna otra relación existente.

#### Scenario: Registrar un defecto de una historia en el backlog

- GIVEN un proyecto existente con una historia que no está asignada a ningún Sprint
- AND un Sprint existente de ese proyecto
- WHEN se solicita registrar un defecto para esa historia y ese Sprint
- THEN el sistema responde `201`
- AND persiste exactamente un defecto asociado a esa historia y a ese Sprint de detección

#### Scenario: Registrar un defecto con un Sprint cerrado como detección

- GIVEN un proyecto y una historia de ese proyecto
- AND un Sprint de ese proyecto con `is_closed` igual a `true`
- AND una solicitud válida para esa historia y ese Sprint
- WHEN se solicita registrar el defecto
- THEN el sistema responde `201`
- AND persiste exactamente un defecto asociado a esa historia y a ese Sprint cerrado

#### Scenario: El alta no modifica datos ajenos

- GIVEN un proyecto, una historia y un Sprint válidos
- AND una solicitud válida
- WHEN se solicita registrar el defecto
- THEN el sistema responde `201` y persiste exactamente un defecto
- AND las historias, Sprints, asignaciones, tareas, integrantes y esfuerzos del proyecto conservan sus filas y sus valores sin cambios

### Requirement: Permitir defectos duplicados

El sistema MUST permitir registrar dos o más defectos con los mismos valores en `story_id`, `sprint_id`, `description`, `severity` y `status` dentro del mismo proyecto, sin ningún mecanismo que deduplique o rechace la solicitudes idénticas. Cada solicitud genera su registro con identificador propio y persiste un defecto independiente.

#### Scenario: Registrar dos defectos idénticos

- GIVEN un proyecto, una historia y un Sprint válidos
- AND dos solicitudes idénticas consecutivas (los mismos valores en los cinco campos del cuerpo)
- WHEN se solicita registrar el defecto dos veces
- THEN cada solicitud responde `201`
- AND persisten exactamente dos defectos con identificadores distintos generados por el servidor
- AND cada registro conserva los datos solicitados

### Requirement: Tratar el alta como una operación atómica

El sistema MUST tratar el alta como todo-o-nada. Si algún campo del cuerpo es inválido, si cualquier referencia no cumple las reglas del proyecto, si el proyecto, la historia o el Sprint no existen, o si la persistencia falla por cualquier motivo, incluido un error inesperado del almacenamiento, el sistema MUST NOT persistir ninguna fila.

#### Scenario: El registro persiste solo si el alta se completa

- GIVEN un proyecto, una historia y un Sprint válidos, y una solicitud de defecto válida
- WHEN el alta completa la persistencia
- THEN existe exactamente una fila en la tabla de defectos con ese proyecto, esa historia, ese Sprint, esa descripción, severidad y estado
- AND no queda ninguna fila parcial

#### Scenario: Un fallo inesperado del almacenamiento no persiste nada

- GIVEN un proyecto, una historia y un Sprint válidos, y un defecto válido
- AND un fallo inesperado del almacenamiento durante la escritura
- WHEN se solicita registrar el defecto
- THEN el sistema responde `500 internal_error` con un mensaje genérico
- AND no queda ninguna fila de defecto

### Requirement: Garantizar la integridad en la base de datos

La base de datos MUST garantizar, con sus propias restricciones, las reglas que el caso de uso verifica en la aplicación: `(story_id, project_id)` referencia una historia de `stories` existente en ese proyecto, `(sprint_id, project_id)` referencia un Sprint existente en ese proyecto, `severity` pertenece al catálogo `alta`/`media`/`baja`, `status` pertenece al catálogo `detectado`/`en_correccion`/`resuelto` y la descripción tiene entre 1 y 500 caracteres. Una violación directa de estas restricciones (por una ruta distinta al caso de uso) MUST fallar sin persistir y MUST NOT estructurarse como `400`/`422`/`500` del caso de uso: cuando la verificación transaccional la precedió, el sistema responde el `404` del error de dominio correspondiente. La migración `000013` MUST permitir aplicarse, revertirse y reaplicarse sin errores, y no agrega restricciones nuevas sobre tablas existentes.

#### Scenario: Una fila con referencias cruzadas de proyecto no puede persistir

- GIVEN una fila de defecto con una historia de otro proyecto o de un proyecto que no coincide con el `project_id` de la fila
- WHEN se intenta insertar directamente en la base de datos
- THEN la inserción falla con la violación de la clave foránea de la historia y ninguna fila queda persistida

#### Scenario: Un valor fuera de catálogo no puede persistir

- GIVEN una fila de defecto con una severidad o un estado fuera de los catálogos, o una descripción vacía o de más de 500 caracteres
- WHEN se intenta insertar directamente en la base de datos
- THEN la inserción falla con la violación del `CHECK` correspondiente y ninguna fila queda persistida

#### Scenario: La migración es reversible y reaplicable

- GIVEN la migración `000013` aplicada sobre un esquema en versión `12`
- WHEN se ejecuta su `down` y después su `up` de nuevo
- THEN ambas operaciones terminan sin errores
- AND los datos de las demás tablas conservan sus filas y sus valores sin cambios

### Requirement: Habilitar la capacidad solo con el esquema en la versión 13

El sistema MUST registrar la ruta de registro de defectos solo cuando el estado de migraciones es limpio (`dirty` igual a `false` con su consulta resuelta en el arranque) en la versión `>= 13`. Con una versión menor, un esquema en estado sucio (`dirty` igual a `true`) o el error de consulta del estado de migraciones al arrancar, la ruta MUST NOT registrarse: la solicitud recibe el `404` genérico de una ruta que no existe. Las rutas ya existentes MUST conservar sus gates propios (proyectos, historias, Sprints, integrantes, tareas, historia completada, cierre de Sprint, sprint history, esfuerzo y Planning Poker), y el flag de esta capacidad es independiente de los demás.

#### Scenario: La ruta existe con la migración 13 aplicada y limpia

- GIVEN un arranque con `schema_migrations` en versión `13` y no sucia
- WHEN se consulta la ruta con una solicitud de defecto válida
- THEN el sistema responde acorde a los requisitos de registro

#### Scenario: La ruta no existe con una versión menor

- GIVEN un arranque con `schema_migrations` en versión `12` o menor, limpia o sucia, o con la consulta del estado de migraciones fallando
- WHEN se solicita `POST /projects/{project_id}/defects`
- THEN el sistema responde `404` porque la ruta no está registrada
- AND las demás rutas disponibles conservan las habilitaciones actuales

#### Scenario: La ruta no existe con el esquema sucio

- GIVEN un arranque con `schema_migrations` en versión `13` pero `dirty` igual a `true`
- WHEN se solicita `POST /projects/{project_id}/defects`
- THEN el sistema responde `404` porque la ruta no está registrada
- AND el resto del API conserva las rutas disponibles que corresponden

### Requirement: Ofrecer únicamente el registro de defectos

Esta capacidad solo registra. El sistema MUST NOT ofrecer en este alcance: consultar el detalle o el listado de defectos de un proyecto (US-18), actualizar el estado de un defecto o registrar su Sprint de resolución (US-17), editar o eliminar defectos, calcular métricas de defectos (US-19) ni operar el alta desde la TUI (TUI-09). Ningún campo adicional de la tabla queda definido en este cambio para esos servicios más allá del alta.

#### Scenario: Sin rutas de consulta, edición o resolución

- GIVEN un esquema en versión `13` limpia y la aplicación en marcha
- WHEN se solicita `GET /projects/{project_id}/defects` o cualquier ruta de consulta, edición o resolución de defectos
- THEN el sistema responde `404` porque esa ruta no existe
- AND únicamente la ruta de registro forma parte del contrato público del módulo
