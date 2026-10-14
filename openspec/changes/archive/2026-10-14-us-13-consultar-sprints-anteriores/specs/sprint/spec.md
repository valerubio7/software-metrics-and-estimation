# Delta de especificación

## ADDED Requirements

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
