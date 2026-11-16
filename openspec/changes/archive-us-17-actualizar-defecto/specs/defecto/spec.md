# Delta para defecto: actualizar el estado y la resolución de un defecto (US-17)

Este cambio agrega a la capacidad `defecto` la transición de estado y el registro del Sprint de resolución de un defecto (US-17, issue #45). No modifica el contrato del alta (US-16) ni del listado (US-18): ambos conservan sus rutas, cuerpos, errores y gates propios, y este delta no incluye sus escenarios, que el archivado debe conservar intactos. La implementación de esta actualización ya está en `main` (PRs #111, #112, #113 y #116); este delta formaliza su contrato.

## Cambio en el Propósito (no es un Requirement; aplicar manualmente a la sección «Propósito» al archivar)

Reemplazar en la sección «Propósito» la oración «La transición a `resuelto` con su Sprint de resolución pertenece a US-17, el cálculo de métricas de defectos (US-19) se especifica en `metricas` y la operación desde la TUI (TUI-09) queda sujeta a la consolidación de su propio cambio; ninguna de ellas forma parte de esta especificación.» por:

«El cálculo de métricas de defectos (US-19) se especifica en `metricas` y la operación desde la TUI (TUI-09) queda sujeta a la consolidación de su propio cambio; ninguna de ellas forma parte de esta especificación.»

## MODIFIED Requirements

### Requirement: Ofrecer el registro y el listado de defectos

Esta capacidad ofrece el registro de defectos (US-16), la consulta del listado del proyecto (US-18) y, desde este cambio, la actualización del estado y la resolución de un defecto (US-17); juntas constituyen la única superficie HTTP definida aquí. El sistema MUST NOT ofrecer en esta especificación: editar o eliminar defectos más allá de la transición de estado y resolución, calcular métricas de defectos (US-19) ni operar defectos desde la TUI (TUI-09); esas operaciones se rigen por sus especificaciones propias. El registro, el listado y la actualización conservan gates independientes o compartidos según corresponda: el alta depende de la migración `000013`; el listado y la actualización dependen de la `000014` (Requirement: Ofrecer el listado solo con el esquema limpio en versión 14; Requirement: Ofrecer la actualización solo con el esquema limpio en versión 14).

#### Scenario: Sin rutas de consulta, edición o resolución

- **GIVEN** un esquema en versión `14` limpia y la aplicación en marcha
- **WHEN** se solicita `PUT` o `DELETE` sobre `/projects/{project_id}/defects` o sobre `/projects/{project_id}/defects/{defect_id}`
- **THEN** el sistema responde `405` sin realizar ninguna mutación ni lectura
- **AND** únicamente el registro (`POST`), el listado (`GET`) y la actualización de estado y resolución (`PATCH` sobre `/projects/{project_id}/defects/{defect_id}`) forman parte del contrato público del módulo definido en esta especificación

## ADDED Requirements

### Requirement: Actualizar el estado y la resolución de un defecto
El sistema MUST ofrecer `PATCH /projects/{project_id}/defects/{defect_id}` para modificar el `status` de un defecto del proyecto, con `resolution_sprint_id` condicionado según el requisito de validación siguiente. Ante una solicitud válida, MUST actualizar atómicamente el defecto y responder `200` con el defecto completo (mismas claves que el alta y el listado). MUST responder `405 method_not_allowed` para métodos distintos de `PATCH`.

#### Scenario: Resolver un defecto con su Sprint de resolución
- **GIVEN** un defecto existente del proyecto en estado `detectado` o `en_correccion`
- **AND** un Sprint de ese proyecto que no es anterior al Sprint de detección del defecto
- **WHEN** se solicita `PATCH` con `status` igual a `"resuelto"` y `resolution_sprint_id` igual al identificador de ese Sprint
- **THEN** el sistema responde `200`
- **AND** persiste el defecto con `status` `"resuelto"` y ese `resolution_sprint_id`

#### Scenario: Reabrir un defecto resuelto
- **GIVEN** un defecto existente del proyecto en estado `"resuelto"` con un `resolution_sprint_id` asignado
- **WHEN** se solicita `PATCH` con `status` igual a `"detectado"` o `"en_correccion"` y sin `resolution_sprint_id`
- **THEN** el sistema responde `200`
- **AND** persiste el defecto con ese nuevo `status` y `resolution_sprint_id` limpio (nulo)

#### Scenario: El método distinto de `PATCH` no actualiza defectos
- **GIVEN** la ruta de actualización de un defecto existente
- **AND** una solicitud con método `GET`, `PUT` o `DELETE`
- **WHEN** se solicita la ruta
- **THEN** el sistema responde `405 method_not_allowed`
- **AND** no modifica ningún defecto

### Requirement: Validar el estado y el Sprint de resolución de la actualización
El sistema MUST aceptar como `status` únicamente el catálogo completo (`detectado`, `en_correccion`, `resuelto`). Con `status` `"resuelto"`, `resolution_sprint_id` MUST ser obligatorio y UUID válido; con otro `status`, MUST rechazarse si está presente. Todo rechazo es `422 validation_failed` con el campo y motivo en `fields`, sin persistir cambios.

#### Scenario: Rechazar un estado fuera de catálogo
- **GIVEN** un defecto existente del proyecto
- **WHEN** se solicita `PATCH` con `status` igual a `"abierto"`, `""` o sin la clave
- **THEN** el sistema responde `422 validation_failed`
- **AND** `fields` identifica `status` como el campo inválido
- **AND** no se modifica el defecto

#### Scenario: Rechazar la resolución sin Sprint de resolución
- **GIVEN** un defecto existente del proyecto
- **WHEN** se solicita `PATCH` con `status` igual a `"resuelto"` sin `resolution_sprint_id`
- **THEN** el sistema responde `422 validation_failed`
- **AND** `fields` identifica `resolution_sprint_id` como requerido
- **AND** no se modifica el defecto

#### Scenario: Rechazar un Sprint de resolución con formato inválido
- **GIVEN** un defecto existente del proyecto
- **WHEN** se solicita `PATCH` con `status` igual a `"resuelto"` y `resolution_sprint_id` con un formato inválido de UUID
- **THEN** el sistema responde `422 validation_failed`
- **AND** `fields` identifica `resolution_sprint_id` con el motivo de UUID inválido
- **AND** no se modifica el defecto

#### Scenario: Rechazar un Sprint de resolución fuera de la transición a resuelto
- **GIVEN** un defecto existente del proyecto
- **WHEN** se solicita `PATCH` con `status` igual a `"detectado"` o `"en_correccion"` y `resolution_sprint_id` presente
- **THEN** el sistema responde `422 validation_failed`
- **AND** `fields` identifica `resolution_sprint_id` con el motivo de que debe omitirse
- **AND** no se modifica el defecto

### Requirement: Verificar las referencias y la coherencia cronológica de la resolución
El sistema MUST verificar, en la misma transacción, la existencia del defecto en el proyecto y la pertenencia del Sprint de resolución a ese proyecto. Un defecto inexistente rechaza con `404 defect_not_found`; un Sprint de resolución inexistente o de otro proyecto, con `404 resolution_sprint_not_found`. Un Sprint de resolución anterior, en el orden del proyecto, al de detección rechaza con `409 resolution_sprint_before_detection`. Ningún rechazo modifica el defecto.

#### Scenario: Rechazar la actualización de un defecto inexistente
- **GIVEN** un proyecto existente
- **WHEN** se solicita `PATCH` sobre un `defect_id` que no existe en ese proyecto
- **THEN** el sistema responde `404 defect_not_found`
- **AND** no se modifica ningún defecto

#### Scenario: Rechazar un Sprint de resolución de otro proyecto
- **GIVEN** un defecto existente del proyecto
- **WHEN** se solicita `PATCH` con `status` igual a `"resuelto"` y `resolution_sprint_id` de un Sprint de otro proyecto
- **THEN** el sistema responde `404 resolution_sprint_not_found`
- **AND** no se modifica el defecto

#### Scenario: Rechazar un Sprint de resolución anterior al de detección
- **GIVEN** un defecto existente del proyecto con un Sprint de detección conocido
- **AND** un Sprint de ese proyecto anterior, en el orden de Sprints, al Sprint de detección
- **WHEN** se solicita `PATCH` con `status` igual a `"resuelto"` y `resolution_sprint_id` igual al identificador de ese Sprint anterior
- **THEN** el sistema responde `409 resolution_sprint_before_detection`
- **AND** no se modifica el defecto

### Requirement: Tratar la actualización como una operación atómica
El sistema MUST tratar la actualización como todo-o-nada: si el cuerpo es inválido, si el defecto o el Sprint de resolución no cumplen las reglas del proyecto, si la coherencia cronológica no se cumple, o si la persistencia falla por cualquier motivo, el sistema MUST NOT dejar modificado ni parcialmente modificado el defecto.

#### Scenario: La actualización persiste solo si se completa por entero
- **GIVEN** un defecto existente y una solicitud `PATCH` válida
- **WHEN** la actualización completa la persistencia
- **THEN** el defecto queda con exactamente el `status` y el `resolution_sprint_id` solicitados
- **AND** no queda una actualización parcial

#### Scenario: Un fallo inesperado del almacenamiento no modifica nada
- **GIVEN** un defecto existente y una solicitud `PATCH` válida
- **AND** un fallo inesperado del almacenamiento durante la escritura
- **WHEN** se solicita la actualización
- **THEN** el sistema responde `500 internal_error` con un mensaje genérico
- **AND** el defecto conserva su estado y su Sprint de resolución previos

### Requirement: Ofrecer la actualización solo con el esquema limpio en versión 14
El sistema MUST registrar la ruta de actualización solo con el esquema limpio (`dirty` igual a `false`) en la versión 14 o superior (migración `000014`). Con versión menor, esquema sucio o error de resolución, MUST NOT registrar la ruta: una solicitud `PATCH` recibe el `404` genérico de una ruta inexistente. El gate es el mismo que el del listado (versión 14) e independiente del gate del alta (versión 13).

#### Scenario: Ruta disponible en versión 14 limpia
- **GIVEN** un arranque con `schema_migrations` en versión `14` —o posterior— limpia y la dependencia de actualización instalada
- **WHEN** se solicita `PATCH /projects/{project_id}/defects/{defect_id}` con una solicitud válida
- **THEN** el sistema responde según el contrato de la actualización
- **AND** las rutas del alta y del listado conservan sus gates propios sin cambios

#### Scenario: Ruta no disponible con versión menor, esquema sucio o error de resolución
- **GIVEN** un arranque con versión `13` limpia, o con cualquier versión en estado sucio (`dirty`), o con el error al resolver el estado de migraciones
- **WHEN** se solicita `PATCH /projects/{project_id}/defects/{defect_id}`
- **THEN** el sistema responde `404` porque la ruta no está registrada
- **AND** el resto del API conserva las rutas disponibles que corresponden
