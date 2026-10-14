# Especificación de Sprint

## Propósito

Definir la creación de un Sprint asociado a un proyecto existente, registrar su Sprint Goal e identificarlo mediante un UUID generado por el servidor. El Sprint queda disponible para futuras asignaciones de historias; su creación no requiere la consulta ni la asignación de historias.

## Requisitos

### Requirement: Crear un Sprint para un proyecto existente

El sistema MUST ofrecer `POST /projects/{project_id}/sprints` para crear un Sprint asociado al proyecto identificado por `project_id`. Ante una solicitud válida, MUST persistir exactamente un Sprint con ese proyecto y el Sprint Goal proporcionado, generar su UUID en el servidor y devolver el identificador y los datos registrados del Sprint. La operación MUST NOT requerir ni realizar la asignación de historias.

#### Scenario: Crear un Sprint válido

- GIVEN un proyecto existente cuyo identificador conoce quien solicita la creación
- AND una solicitud que proporciona un Sprint Goal
- WHEN el cliente envía `POST /projects/{project_id}/sprints` para ese proyecto
- THEN el sistema persiste exactamente un Sprint asociado al proyecto
- AND registra el Sprint Goal proporcionado
- AND devuelve una respuesta de creación exitosa con el UUID generado por el servidor y los datos registrados

#### Scenario: Crear un Sprint sin depender de historias del Product Backlog

- GIVEN un proyecto existente, tenga o no historias disponibles para asignación
- AND una solicitud con el Sprint Goal
- WHEN el cliente solicita crear un Sprint para ese proyecto
- THEN el sistema crea el Sprint sin exigir que existan historias en el Product Backlog
- AND el Sprint no contiene historias asignadas por esta operación

### Requirement: Validar la información obligatoria antes de persistir

El sistema MUST exigir el Sprint Goal para crear un Sprint y MUST rechazar una solicitud a la que le falte ese dato antes de persistir. El sistema MUST informar que falta el Sprint Goal y MUST NOT persistir un Sprint cuando se rechace la solicitud por este motivo. Este requisito no establece restricciones adicionales sobre el contenido o la longitud del Sprint Goal.

#### Scenario: Rechazar una solicitud sin Sprint Goal

- GIVEN un proyecto existente y una solicitud de creación que no proporciona Sprint Goal
- WHEN el cliente solicita crear un Sprint para ese proyecto
- THEN el sistema rechaza la solicitud e informa que falta el Sprint Goal
- AND no persiste ningún Sprint

### Requirement: Asociar el Sprint únicamente a un proyecto existente

El sistema MUST rechazar la creación cuando `project_id` no identifica un UUID válido o no corresponde a un proyecto existente. Ante el rechazo, MUST informar que el identificador o el proyecto no es válido y MUST NOT persistir un Sprint.

#### Scenario: Rechazar un identificador de proyecto con formato inválido

- GIVEN una solicitud de creación con un `project_id` que no tiene formato UUID válido
- WHEN el cliente envía la solicitud a `POST /projects/{project_id}/sprints`
- THEN el sistema rechaza la solicitud indicando que el identificador del proyecto no es válido
- AND no persiste ningún Sprint

#### Scenario: Rechazar un proyecto inexistente

- GIVEN un `project_id` con formato UUID válido que no corresponde a un proyecto existente
- AND una solicitud con Sprint Goal
- WHEN el cliente solicita crear un Sprint para ese proyecto
- THEN el sistema rechaza la creación indicando que el proyecto no existe
- AND no persiste ningún Sprint

### Requirement: Rechazar asignaciones a un Sprint cerrado (US-09)

El sistema MUST usar `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`, agregado por la migración canónica `000008`, como fuente de cierre. Sprints existentes y nuevos MUST quedar abiertos por defecto. La asignación MUST comprobar el cierre dentro de la misma transacción que valida Sprint, proyecto e historias y escribe el lote. MUST serializarse frente a cierres y asignaciones concurrentes mediante el bloqueo del Sprint; un cierre que confirma primero MUST impedir la asignación posterior. La creación de Sprints MUST conservar su gate limpio >=5; la asignación MUST exigir versión limpia >=8 y dependencia explícita.

#### Scenario: Sprint cerrado

- GIVEN un Sprint con `is_closed=true` y una selección de historias válidas del mismo proyecto
- WHEN se solicita `POST /projects/{project_id}/sprints/{sprint_id}/stories`
- THEN se responde `409 assignment_conflict`
- AND no se agrega ningún vínculo

#### Scenario: Cierre concurrente confirma primero

- GIVEN una transacción de cierre que bloquea el Sprint
- WHEN una asignación concurrente espera ese bloqueo y el cierre confirma
- THEN la asignación observa el Sprint cerrado y rechaza íntegramente el lote

La asignación no define un endpoint de cierre, tareas ni finalización, y no cambia el estado de las historias. `000007` almacena vínculos independientes; las historias permanecen en el Product Backlog. Revertir las migraciones US-09 con `down 2` desde v8 elimina asociaciones y cierre, no proyectos/historias/Sprints ni integrantes de v6; no se afirma seguridad de rollback productivo ni compatibilidad con v5/v6 alternativos desplegados desconocidos.

### Requirement: Cerrar un Sprint abierto

El sistema MUST permitir cerrar un Sprint abierto del proyecto indicado mediante `POST /projects/{project_id}/sprints/{sprint_id}/closure`, sin requerir un cuerpo. El éxito MUST responder `200` con `project_id`, `sprint_id` e `is_closed=true`.

#### Scenario: Cierre exitoso

- WHEN se solicita el cierre de un Sprint abierto perteneciente al proyecto indicado
- THEN el sistema persiste el estado cerrado y responde `200` con ambos identificadores e `is_closed=true`

#### Scenario: Ignorar el cuerpo de la solicitud

- WHEN se solicita el cierre de un Sprint abierto enviando contenido en el cuerpo
- THEN el sistema cierra el Sprint sin exigir ni interpretar ese contenido

### Requirement: Validar identificadores y pertenencia antes del cierre

El sistema MUST validar los UUID antes de consultar el almacenamiento. Debe responder `422 validation_failed` por identificadores inválidos, `404 project_not_found` si el proyecto no existe y `404 sprint_not_found` si el Sprint no existe o pertenece a otro proyecto. Los rechazos MUST NOT modificar datos.

#### Scenario: Identificadores inválidos

- WHEN uno o ambos identificadores de ruta no son UUID válidos
- THEN el sistema responde `422 validation_failed` con los campos inválidos, sin consultar ni modificar el almacenamiento

#### Scenario: Proyecto inexistente

- WHEN el proyecto indicado no existe
- THEN el sistema responde `404 project_not_found` y no modifica datos

#### Scenario: Sprint inexistente o de otro proyecto

- WHEN el Sprint indicado no existe o no pertenece al proyecto de la ruta
- THEN el sistema responde `404 sprint_not_found` y no modifica datos

### Requirement: Rechazar el cierre repetido

El sistema MUST responder `409 sprint_closed` al solicitar el cierre de un Sprint ya cerrado y MUST preservar su estado y datos relacionados.

#### Scenario: Sprint ya cerrado

- WHEN se solicita cerrar nuevamente un Sprint cerrado
- THEN el sistema responde `409 sprint_closed` y conserva todos sus datos sin cambios

### Requirement: Conservar la información histórica

Cerrar un Sprint MUST NOT eliminar ni alterar sus historias planificadas o completadas, asociaciones, tareas, esfuerzos o defectos. El Sprint cerrado MUST seguir disponible para consultas posteriores y el cierre MUST NOT modificar los datos de las historias.

#### Scenario: Preservar resultados de la iteración

- WHEN se cierra un Sprint con historias, finalizaciones y datos de trabajo relacionados
- THEN todos esos registros y los datos de las historias permanecen sin cambios

### Requirement: Serializar el cierre frente a asignaciones concurrentes

El cierre y la asignación MUST serializarse para el mismo Sprint. Si el cierre confirma primero, una asignación posterior MUST rechazar íntegramente el lote según el conflicto definido por US-09. Si la asignación confirma primero, el cierre MUST conservar sus asociaciones.

#### Scenario: El cierre confirma antes que la asignación

- WHEN una asignación concurrente obtiene el bloqueo después de confirmar el cierre
- THEN la asignación observa el Sprint cerrado, rechaza el lote y no crea asociaciones

#### Scenario: La asignación confirma antes que el cierre

- WHEN la asignación confirma antes de que el cierre obtenga el bloqueo
- THEN el cierre conserva todas las asociaciones creadas

### Requirement: Habilitar el cierre solo con el esquema requerido

La ruta MUST estar disponible únicamente con esquema limpio en versión 8 o superior. Con versión inferior, estado `dirty` o error al resolver el esquema, la ruta MUST responder `404` sin alterar la disponibilidad de otras operaciones.

#### Scenario: Esquema compatible

- WHEN la API inicia con esquema limpio versión 8 o superior
- THEN la ruta de cierre queda disponible

#### Scenario: Esquema incompatible

- WHEN la API inicia con versión inferior a 8, estado `dirty` o error de resolución
- THEN la ruta no queda registrada y responde `404`

### Requirement: Manejar métodos no admitidos y fallos de almacenamiento

La ruta MUST aceptar únicamente `POST` y responder `405` para otros métodos. Un fallo inesperado de almacenamiento MUST responder `500 internal_error` con mensaje genérico, sin efectos parciales ni detalles internos.

#### Scenario: Método no admitido

- WHEN se usa un método distinto de `POST` en la ruta de cierre
- THEN el sistema responde `405` y no modifica datos

#### Scenario: Fallo de almacenamiento

- WHEN ocurre un error inesperado al persistir el cierre
- THEN el sistema responde `500 internal_error` genérico y no deja efectos parciales
### Requirement: Consultar los Sprints anteriores de un proyecto
El sistema MUST permitir consultar los Sprints anteriores de un proyecto existente mediante `GET /projects/{project_id}/sprints`, sin parámetros de consulta ni cuerpo. "Sprint anterior" SHALL significar Sprint cerrado (`is_closed=true`); los Sprints abiertos MUST NOT listarse. El éxito MUST responder `200` con el contenedor `{"project_id": "...", "sprints": [...]}`, que incluye únicamente Sprints cerrados del proyecto indicado y ninguno de otros proyectos.

#### Scenario: Listar los Sprints cerrados de un proyecto
- **GIVEN** un proyecto existente con varios Sprints cerrados
- **WHEN** se solicita `GET /projects/{project_id}/sprints`
- **THEN** el sistema responde `200` con `project_id` y `sprints` conteniendo exactamente los Sprints cerrados de ese proyecto

#### Scenario: Excluir Sprints abiertos
- **GIVEN** un proyecto con un Sprint cerrado y un Sprint abierto
- **WHEN** se solicita `GET /projects/{project_id}/sprints`
- **THEN** `sprints` contiene solo el Sprint cerrado
- **AND** el Sprint abierto no aparece en la respuesta

#### Scenario: Excluir Sprints de otros proyectos
- **GIVEN** dos proyectos, cada uno con Sprints cerrados
- **WHEN** se solicita la lista de uno de ellos
- **THEN** la respuesta contiene solo los Sprints cerrados de ese proyecto

### Requirement: Representar cada Sprint cerrado y sus historias planificadas
Cada Sprint listado MUST exponer `id`, `project_id`, `sprint_goal`, `is_closed` (siempre `true`) y `stories`, la lista de historias planificadas asignadas al Sprint. Cada historia MUST exponer `id`, `title`, `priority`, `story_points`, `completed_at` y `completed`. `story_points` MUST ser nullable y MUST devolverse como `null` cuando la historia no tiene puntos. `completed_at` MUST tomarse de la finalización de la historia en ese Sprint, ser nullable y expresarse en ISO 8601 UTC. `completed` MUST ser un booleano que vale `true` si y solo si `completed_at` no es nulo. El sistema MUST NOT usar el estado global de la historia (`stories.status`) para determinar la finalización, MUST NOT exponer `description`, `acceptance_criteria`, `estimated_hours` ni `seq`, y MUST NOT incluir totales, porcentajes ni fechas del Sprint.

#### Scenario: Identificar Sprint y Sprint Goal
- **GIVEN** un proyecto con un Sprint cerrado cuyo Sprint Goal es "Entregar el login"
- **WHEN** se solicita la lista de Sprints
- **THEN** el Sprint incluye su `id`, el `project_id`, `sprint_goal` igual a "Entregar el login" e `is_closed=true`

#### Scenario: Historia completada en el Sprint
- **GIVEN** un Sprint cerrado con una historia cuya finalización en ese Sprint tiene una marca de tiempo
- **WHEN** se solicita la lista de Sprints
- **THEN** la historia aparece con `completed_at` igual a esa marca en ISO 8601 UTC y `completed=true`

#### Scenario: Historia planificada no completada
- **GIVEN** un Sprint cerrado con una historia asignada sin finalización en ese Sprint
- **WHEN** se solicita la lista de Sprints
- **THEN** la historia aparece con `completed_at=null` y `completed=false`

#### Scenario: Finalización independiente por Sprint
- **GIVEN** una historia asignada a dos Sprints cerrados, completada en uno y no completada en el otro
- **WHEN** se solicita la lista de Sprints
- **THEN** la historia aparece con `completed=true` en el primer Sprint y `completed=false` en el segundo
- **AND** el estado global de la historia no altera ninguno de los dos valores

#### Scenario: Story points nulos
- **GIVEN** una historia planificada sin puntos de historia
- **WHEN** se solicita la lista de Sprints
- **THEN** la historia incluye `story_points` con valor `null`

#### Scenario: Campos no expuestos
- **WHEN** se solicita la lista de Sprints
- **THEN** ni los Sprints ni las historias incluyen `description`, `acceptance_criteria`, `estimated_hours`, `seq`, totales ni fechas del Sprint

### Requirement: Ordenar el historial de Sprints
El sistema MUST ordenar los Sprints del más reciente al más antiguo según su orden de creación. Dentro de cada Sprint, las historias MUST ordenarse por su orden de creación ascendente (`stories.seq`). El orden MUST ser determinista entre consultas repetidas sobre los mismos datos.

#### Scenario: Sprints del más reciente al más antiguo
- **GIVEN** un proyecto con tres Sprints cerrados creados en secuencia
- **WHEN** se solicita la lista de Sprints
- **THEN** `sprints` aparece con el último Sprint creado primero y el primero creado al final, sin importar el orden en que se cerraron

#### Scenario: Historias por orden de creación
- **GIVEN** un Sprint cerrado con historias asignadas en un orden distinto al de su creación
- **WHEN** se solicita la lista de Sprints
- **THEN** las `stories` del Sprint aparecen en el orden de creación de las historias, de la más antigua a la más reciente

#### Scenario: Orden estable
- **GIVEN** los mismos datos sin cambios
- **WHEN** se solicita la lista de Sprints dos veces
- **THEN** ambas respuestas tienen el mismo orden de Sprints y de historias

### Requirement: Informar listas vacías sin ambigüedad
Un proyecto existente sin Sprints cerrados MUST responder `200` con `"sprints": []`, nunca `null`, incluido el caso de un proyecto cuyo único Sprint está abierto. Un Sprint cerrado sin historias asignadas MUST aparecer con `"stories": []`, nunca `null`. El texto informativo de la lista vacía es responsabilidad de la capa de presentación; la respuesta MUST NOT incluir un mensaje.

#### Scenario: Proyecto sin Sprints
- **GIVEN** un proyecto existente que nunca tuvo Sprints
- **WHEN** se solicita la lista de Sprints
- **THEN** el sistema responde `200` con `project_id` y `"sprints": []`

#### Scenario: Proyecto con solo un Sprint abierto
- **GIVEN** un proyecto existente cuyo único Sprint está abierto
- **WHEN** se solicita la lista de Sprints
- **THEN** el sistema responde `200` con `"sprints": []`
- **AND** no responde `404 project_not_found`

#### Scenario: Sprint cerrado sin historias
- **GIVEN** un Sprint cerrado sin historias asignadas
- **WHEN** se solicita la lista de Sprints
- **THEN** el Sprint aparece en `sprints` con `"stories": []`

### Requirement: Validar el proyecto antes de listar
El sistema MUST validar que `project_id` sea un UUID válido antes de consultar el almacenamiento y MUST responder `422 validation_failed` con `fields.project_id` si no lo es. Si el identificador es válido pero el proyecto no existe, MUST responder `404 project_not_found`. Los rechazos MUST NOT modificar datos.

#### Scenario: Identificador inválido
- **GIVEN** un `project_id` que no es un UUID válido
- **WHEN** se solicita `GET /projects/{project_id}/sprints`
- **THEN** el sistema responde `422 validation_failed` con `fields.project_id`
- **AND** no consulta el almacenamiento

#### Scenario: Proyecto inexistente
- **GIVEN** un `project_id` con formato UUID válido que no corresponde a ningún proyecto
- **WHEN** se solicita `GET /projects/{project_id}/sprints`
- **THEN** el sistema responde `404 project_not_found`

### Requirement: Consultar sin modificar información histórica
La consulta MUST ser de solo lectura y MUST NOT crear, actualizar ni eliminar registros de Sprints, asociaciones Sprint-historia ni finalizaciones, historias o tareas. Una consulta MUST producir una vista consistente de una única lectura, sin listas parciales.

#### Scenario: Sin mutación de datos
- **GIVEN** proyectos con Sprints cerrados y abiertos, historias, finalizaciones y tareas
- **WHEN** se ejecuta la consulta de Sprints anteriores
- **THEN** los datos de `sprints`, `sprint_stories`, `stories` y `tasks` son idénticos antes y después

#### Scenario: Consultas repetidas
- **GIVEN** un proyecto con Sprints cerrados
- **WHEN** se ejecuta la misma consulta varias veces sin otros cambios
- **THEN** cada respuesta es idéntica y no se acumulan efectos

### Requirement: Manejar métodos no admitidos y fallos de almacenamiento en la consulta
La ruta de consulta MUST aceptar `GET` y `HEAD`, y responder `405` para cualquier otro método. Un fallo inesperado de almacenamiento MUST responder `500 internal_error` con mensaje genérico, sin lista parcial ni detalles internos.

#### Scenario: Método no admitido
- **WHEN** se usa un método distinto de `GET` o `HEAD` sobre una ruta de consulta registrada
- **THEN** el sistema responde `405` y no modifica datos

#### Scenario: Fallo de almacenamiento
- **GIVEN** un error inesperado del almacenamiento durante la consulta
- **WHEN** se solicita la lista de Sprints
- **THEN** el sistema responde `500 internal_error` con mensaje genérico
- **AND** la respuesta no incluye ninguna lista parcial ni detalles internos

### Requirement: Habilitar la consulta solo con el esquema requerido
La consulta MUST estar disponible únicamente con esquema limpio en versión 11 o superior, con una condición de disponibilidad independiente de la del cierre de Sprints. Con versión inferior a 11, estado `dirty` o error al resolver el esquema, la ruta `GET` MUST NOT registrarse. Si la ruta `POST` de creación sobre la misma colección sigue registrada según su propio gate (esquema limpio en versión 5 a 10), `GET /projects/{project_id}/sprints` MUST responder `405`; si no hay ninguna ruta de Sprints registrada (versión inferior a 5, estado `dirty` o error al resolver el esquema), MUST responder `404`. La indisponibilidad de la consulta MUST NOT alterar la disponibilidad de la creación, la asignación ni el cierre.

#### Scenario: Esquema compatible
- **WHEN** la API inicia con esquema limpio versión 11 o superior
- **THEN** `GET /projects/{project_id}/sprints` queda disponible

#### Scenario: Esquema incompatible con la creación registrada
- **GIVEN** la API inicia con esquema limpio en versión 5 a 10
- **WHEN** se solicita `GET /projects/{project_id}/sprints`
- **THEN** la ruta de consulta no queda registrada y el sistema responde `405`

#### Scenario: Esquema sin rutas de Sprints
- **GIVEN** la API inicia con versión inferior a 5, estado `dirty` o error de resolución
- **WHEN** se solicita `GET /projects/{project_id}/sprints`
- **THEN** no hay ninguna ruta de Sprints registrada y el sistema responde `404`

#### Scenario: Independencia del gate de cierre
- **GIVEN** un esquema limpio en versión 8, 9 o 10
- **WHEN** la API inicia
- **THEN** el cierre de Sprints sigue disponible y la consulta de Sprints anteriores no lo está

#### Scenario: Otras operaciones no se alteran
- **GIVEN** la consulta no disponible por esquema incompatible
- **WHEN** se usan la creación, la asignación o el cierre de Sprints
- **THEN** se comportan según sus propios gates, sin cambios

### Requirement: Agregar la secuencia de creación de Sprints
La migración `000011` MUST agregar únicamente la columna `sprints.seq` (`BIGINT`, generada como identidad, `NOT NULL`) con la restricción `UNIQUE (project_id, seq)`, para fijar el orden de creación de los Sprints. La migración MUST NOT modificar otras tablas ni columnas, ni eliminar o alterar datos existentes. Su reversión (`down`) MUST eliminar la restricción y la columna sin afectar Sprints, asociaciones, finalizaciones ni historias. `seq` MUST NOT exponerse en la API. Los Sprints existentes MAY recibir `seq` en un orden físico arbitrario al aplicar la migración.

#### Scenario: Aplicar la migración
- **GIVEN** un esquema en versión 10 con Sprints existentes
- **WHEN** se aplica la migración `000011`
- **THEN** `sprints.seq` existe, cada Sprint tiene un valor único dentro de su proyecto y la única diferencia estructural es esa columna y su restricción

#### Scenario: Nuevos Sprints reciben secuencia creciente
- **GIVEN** el esquema en versión 11
- **WHEN** se crean dos Sprints consecutivos
- **THEN** el segundo recibe un `seq` mayor que el primero

#### Scenario: Revertir la migración
- **GIVEN** el esquema en versión 11 con Sprints, asociaciones y finalizaciones
- **WHEN** se revierte la migración `000011`
- **THEN** se eliminan la restricción `UNIQUE (project_id, seq)` y la columna `sprints.seq`
- **AND** los Sprints, las asociaciones, las finalizaciones y las historias permanecen sin cambios

#### Scenario: Restricción de unicidad por proyecto
- **WHEN** se intenta registrar dos Sprints del mismo proyecto con el mismo `seq`
- **THEN** el almacenamiento rechaza la segunda escritura
