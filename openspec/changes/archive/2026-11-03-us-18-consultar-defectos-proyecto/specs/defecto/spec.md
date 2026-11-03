# Delta para defecto: consultar los defectos de un proyecto (US-18)

Este cambio agrega a la capacidad `defecto` la consulta de solo lectura del listado de defectos de un proyecto (US-18, issue #46). No modifica el contrato HTTP del alta: el registro conserva su ruta, su cuerpo, sus errores y su gate por versión 13, y este delta no incluye los escenarios de ese requisito, que el archivado debe conservar intactos. La implementación de esta consulta ya está en `main` (PRs #117 y #118); este delta formaliza su contrato.

## Cambio en el Propósito (no es un Requirement; aplicar manualmente a la sección «Propósito» al archivar)

Reemplazar en la sección «Propósito» la oración «El defecto queda persistido y asociado a su proyecto, a esa historia y a ese Sprint; la resolución y su Sprint correspondiente no forman parte de este cambio (US-17), y tampoco el listado de defectos (US-18), el cálculo de métricas de defectos (US-19) ni la operación desde la TUI (TUI-09). Esta capacidad solo registra.» por:

«El defecto queda persistido y asociado a su proyecto, a esa historia y a ese Sprint. Esta capacidad ofrece el alta (US-16) y, desde US-18, la consulta de solo lectura del listado de defectos del proyecto. La transición a `resuelto` con su Sprint de resolución pertenece a US-17, el cálculo de métricas de defectos (US-19) se especifica en `metricas` y la operación desde la TUI (TUI-09) queda sujeta a la consolidación de su propio cambio; ninguna de ellas forma parte de esta especificación.»

## RENAMED Requirements

- FROM: `### Requirement: Ofrecer únicamente el registro de defectos`
- TO: `### Requirement: Ofrecer el registro y el listado de defectos`

## MODIFIED Requirements

### Requirement: Ofrecer el registro y el listado de defectos

Esta capacidad ofrece el registro de defectos (US-16) y, desde este cambio, la consulta del listado del proyecto (US-18); juntas constituyen la única superficie HTTP definida aquí. El listado MUST NOT crear, modificar ni eliminar ningún dato. El sistema MUST NOT ofrecer en esta especificación: actualizar el estado de un defecto o registrar su Sprint de resolución (US-17), editar o eliminar defectos, calcular métricas de defectos (US-19) ni operar defectos desde la TUI (TUI-09); esas operaciones se rigen por sus especificaciones propias. El registro y el listado conservan gates independientes: el alta es dependiente de la migración `000013` y el listado de la `000014` (Requirement: Ofrecer el listado solo con el esquema limpio en versión 14).

#### Scenario: Sin rutas de consulta, edición o resolución

- GIVEN un esquema en versión `14` limpia y la aplicación en marcha
- WHEN se solicita `PUT`, `PATCH` o `DELETE` sobre `/projects/{project_id}/defects`
- THEN el sistema responde `405` sin realizar ninguna mutación ni lectura
- AND únicamente el registro (`POST`) y el listado (`GET`) forman parte del contrato público del módulo definido en esta especificación

## ADDED Requirements

### Requirement: Consultar el listado de defectos de un proyecto
El sistema MUST ofrecer `GET /projects/{project_id}/defects` y aceptar también `HEAD`, y MUST NOT aceptar ningún otro método en la ruta. MUST responder `200` con un objeto JSON con las claves `project_id` y `defects`, con `defects` siempre como arreglo, vacío si no hay defectos. La respuesta MUST NOT exponer marcas internas como `created_at`; la ruta MUST NOT ofrecer filtros ni paginación, y el arreglo MUST conservar el orden de registro de los defectos.

#### Scenario: Listar los defectos del proyecto con su Sprint de resolución
- **GIVEN** un proyecto con dos defectos, uno con Sprint de resolución asignado y otro sin él, ambos pertenecientes al proyecto
- **WHEN** se solicita `GET /projects/{project_id}/defects`
- **THEN** responde `200` con el sobre `project_id`/`defects`
- **AND** cada elemento incluye exactamente `id`, `description`, `severity` dentro de `alta`, `media` y `baja`, `status` entre `detectado`, `en_correccion` y `resuelto` según el almacenamiento, `story_id`, `sprint_id` con el Sprint de detección y `resolution_sprint_id` con el Sprint de resolución o `null` si no lo tiene
- **AND** ningún elemento expone `created_at` ni datos de otros proyectos

#### Scenario: Listado vacío explícito
- **GIVEN** un proyecto existente sin defectos registrados
- **WHEN** se solicita `GET /projects/{project_id}/defects`
- **THEN** responde `200` con `"defects"` como arreglo vacío, no `null`

#### Scenario: Orden por orden de registro
- **GIVEN** un proyecto con defectos registrados en un orden conocido
- **WHEN** se consulta el listado
- **THEN** los elementos aparecen en ese orden de registro y no en otro derivado de identificadores o fechas

#### Scenario: UUID canónico en minúsculas
- **GIVEN** una ruta con `project_id` válido escrito con mayúsculas
- **WHEN** se solicita el listado
- **THEN** la clave `project_id` del sobre devuelve el mismo UUID en minúsculas y el arreglo correspondiente a ese proyecto

#### Scenario: Sin filtros ni paginación
- **GIVEN** la ruta de listado disponible
- **WHEN** se consulta el listado
- **THEN** la ruta no ofrece parámetros de consulta para filtrar por severidad, estado, historia o Sprint, ni paginación
- **AND** cada consulta devuelve el listado completo del proyecto en el arreglo `defects`

### Requirement: Validar referencias y fallar sin filtrar detalles del listado
El sistema MUST validar el UUID de la ruta antes de leer y responder `422 validation_failed` con `fields.project_id` si es inválido, sin ninguna lectura. Para un proyecto inexistente MUST responder `404 project_not_found`, y la consulta MUST NOT incluir defectos ajenos al proyecto de la ruta. Ante un fallo inesperado del almacenamiento MUST responder `500 internal_error` genérico. Los métodos distintos de `GET` y `HEAD` recibirán `405 method_not_allowed` sin lecturas.

#### Scenario: Identificador de proyecto inválido
- **WHEN** `project_id` no es un UUID válido
- **THEN** responde `422 validation_failed` con `fields.project_id`
- **AND** no se realiza ninguna lectura de almacenamiento

#### Scenario: Proyecto inexistente
- **GIVEN** un `project_id` con formato válido que no corresponde a ningún proyecto
- **WHEN** se solicita el listado
- **THEN** responde `404 project_not_found` sin datos de ningún proyecto

#### Scenario: Aislamiento entre proyectos
- **GIVEN** dos proyectos con defectos registrados cada uno
- **WHEN** se consulta el listado de uno de ellos
- **THEN** el sobre es su `project_id` y el arreglo contiene exclusivamente sus defectos, sin defectos del otro proyecto

#### Scenario: Método no admitido sin lecturas
- **GIVEN** la ruta de listado disponible
- **WHEN** se solicita con `POST`, `PUT`, `PATCH` o `DELETE`
- **THEN** responde `405 method_not_allowed` sin realizar ninguna operación sobre el almacenamiento

#### Scenario: Fallo de almacenamiento sin detalles
- **GIVEN** una solicitud válida de `GET` con el proyecto existente
- **WHEN** falla inesperadamente el almacenamiento durante la lectura
- **THEN** responde `500 internal_error` genérico, sin defectos parciales ni detalles internos

### Requirement: Consultar el listado sin modificar los datos
El listado MUST ser de solo lectura y consistente: sin transacción de escritura y sin bloqueos exclusivos en el camino de la lectura. MUST NOT crear, modificar ni eliminar defectos, historias, Sprints, tareas, registros de esfuerzo, integrantes ni asociaciones. Consultas repetidas sobre los mismos datos MUST devolver los mismos resultados sin multiplicar elementos ni consumir contadores.

#### Scenario: Consultas repetidas consistentes
- **GIVEN** un proyecto con defectos registrados
- **WHEN** se realiza la misma consulta del listado más de una vez
- **THEN** cada respuesta coincide en `project_id`, tamaño del arreglo y contenido y orden de los elementos

#### Scenario: Lectura sin alteración de datos
- **GIVEN** un proyecto con defectos registrados y un conteo y contenido conocidos de la tabla de defectos
- **WHEN** se consultan listados válidos, vacíos, repetidos o rechazados por validación
- **THEN** la tabla de defectos y las relaciones observables permanecen iguales antes y después

### Requirement: Ofrecer el listado solo con el esquema limpio en versión 14
El sistema MUST registrar la ruta de listado solo con el estado de migraciones limpio (`dirty` igual a `false`) en la versión 14 o superior (migración `000014`). Con versión menor, esquema sucio o error de resolución al arrancar, MUST NOT registrar la ruta: una solicitud `GET` recibe el `405` por patrón ausente en esa ruta. Con el módulo de defecto completo ausente, las solicitudes sobre la ruta reciben el `404` genérico. El gate del listado es independiente del gate del alta (versión 13).

#### Scenario: Ruta disponible en versión 14 limpia
- **GIVEN** un arranque con `schema_migrations` en versión `14` —o posterior— limpia y la dependencia del listado instalada
- **WHEN** se solicita `GET /projects/{project_id}/defects`
- **THEN** el sistema responde según el contrato del listado
- **AND** la ruta del alta conserva su gate propio (versión 13) sin cambios

#### Scenario: Ruta no disponible con versión menor, esquema sucio o error de resolución
- **GIVEN** un arranque con versión `13` limpia, o con cualquier versión en estado sucio (`dirty`), o con el error al resolver el estado de migraciones
- **WHEN** se solicita `GET /projects/{project_id}/defects`
- **THEN** el sistema responde `405` porque la ruta con métodos de lectura no está registrada
- **AND** el resto del API conserva las rutas disponibles que corresponden

#### Scenario: Módulo de defecto ausente
- **GIVEN** un arranque sin la dependencia del módulo de defecto completa
- **WHEN** se solicita `GET` o `POST` sobre `/projects/{project_id}/defects`
- **THEN** el sistema responde `404` genérico de una ruta que no existe
