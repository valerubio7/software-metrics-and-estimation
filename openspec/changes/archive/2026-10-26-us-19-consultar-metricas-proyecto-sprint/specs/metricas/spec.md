# Especificación de métricas

## Propósito

Permitir evaluar avance, esfuerzo y calidad de un proyecto o Sprint mediante métricas de solo lectura, distinguiendo resultados conocidos y observaciones faltantes.

## ADDED Requirements

### Requirement: Seleccionar el ámbito mediante la API
El sistema MUST ofrecer `GET /projects/{project_id}/metrics` y `GET /projects/{project_id}/sprints/{sprint_id}/metrics`. MUST responder `200` con métricas del ámbito válido: historias únicas del proyecto, incluso sin Sprint, o historias asignadas al Sprint indicado. MUST excluir datos de otros proyectos.

#### Scenario: Proyecto sin duplicar historias
- **WHEN** se consulta un proyecto con una historia de 5 puntos asignada a dos Sprints y otra de 3 sin Sprint
- **THEN** los puntos planificados son 8, no 13, y no incluyen historias ajenas

#### Scenario: Selección de Sprint
- **WHEN** se consulta un Sprint abierto o cerrado con una historia asignada y otras historias del proyecto no asignadas a él
- **THEN** se calculan sus métricas de historias únicamente sobre la historia asignada

### Requirement: Preservar el contrato JSON
La respuesta MUST conservar en el nivel raíz `project_id`, `sprint_id`, `story_points_planned`, `story_points_completed`, `velocity`, `estimated_hours`, `actual_hours`, `deviation`, `completed_stories_percentage`, `defects_detected` y `defects_resolved`. MUST agregar `deviation_percentage` y `coverage`, sin envolver las métricas ni sustituir valores desconocidos por cero.

#### Scenario: Identificar la consulta
- **WHEN** se consulta un proyecto o un Sprint válido
- **THEN** `project_id` identifica el proyecto y `sprint_id` identifica el Sprint consultado o es `null` para proyecto; las métricas permanecen en el nivel raíz

### Requirement: Calcular puntos planificados y completados
El sistema MUST sumar puntos de todas las historias del ámbito para planificación y solo de las completadas para completados. MUST usar finalización por `completed_at`, nunca `stories.status`; en proyecto, una historia cuenta una vez si completó en algún Sprint. Cada agregado MUST ser `null` si falta algún punto relevante, informando cuántos faltan, y `0` si su conjunto está vacío.

#### Scenario: Finalización independiente del estado global
- **WHEN** una historia de 5 puntos completó en dos Sprints y otra de 3 tiene estado global completada pero ningún `completed_at`
- **THEN** el proyecto devuelve 8 puntos planificados y 5 completados; en cada Sprint solo cuenta la finalización de su asociación

#### Scenario: Estimación faltante solo en historia pendiente
- **WHEN** el ámbito tiene una historia completada de 3 puntos y una pendiente sin puntos
- **THEN** los planificados son `null` con una observación faltante y los completados son 3 con cero faltantes

#### Scenario: Estimación faltante en historia completada
- **WHEN** una historia completada del ámbito no tiene puntos
- **THEN** los puntos planificados y completados son `null` y cada contador incluye esa observación faltante

#### Scenario: Cero conocido
- **WHEN** el ámbito no tiene historias o no tiene completadas
- **THEN** el agregado del conjunto vacío es 0; una estimación explícita de 0 no cuenta como faltante

### Requirement: Calcular el porcentaje de historias completadas
El sistema MUST devolver historias completadas / historias planificadas × 100, con historias únicas en proyecto y asociaciones del Sprint en Sprint. MUST devolver `null` si no hay historias planificadas, sin depender de las estimaciones de puntos.

#### Scenario: Porcentaje sin estimaciones
- **WHEN** una de dos historias del ámbito está completada y ambas carecen de puntos
- **THEN** `completed_stories_percentage` es 50 aunque los puntos sean desconocidos

#### Scenario: Sin denominador
- **WHEN** el ámbito no tiene historias
- **THEN** `completed_stories_percentage` es `null`

### Requirement: Calcular velocidad por ámbito
La velocidad de Sprint MUST ser sus puntos completados. La de proyecto MUST ser el promedio aritmético de los puntos completados de todos sus Sprints cerrados, incluyendo ceros. MUST ser `null` sin Sprints cerrados o si alguno tiene puntos completados desconocidos; el contador de cobertura MUST contar los Sprints incompletos.

#### Scenario: Promedio incluyendo ceros
- **WHEN** hay Sprints cerrados con 8 y 0 puntos completados y un Sprint abierto con 13
- **THEN** la velocidad del proyecto es 4 y excluye el abierto; la velocidad del Sprint sin completadas es 0

#### Scenario: Promedio desconocido
- **WHEN** no hay Sprints cerrados o uno de los cerrados tiene puntos completados desconocidos
- **THEN** la velocidad del proyecto es `null` y `coverage.velocity.missing_observations` es 0 o 1 respectivamente

#### Scenario: Velocidad de Sprint incompleta
- **WHEN** un Sprint tiene dos historias completadas sin puntos
- **THEN** su velocidad es `null` y su contador de observaciones faltantes es 2

### Requirement: Calcular horas estimadas sin doble conteo
En proyecto, el sistema MUST usar todas las tareas de cada historia única; en Sprint, solo sus tareas del Sprint seleccionado. Si una historia no tiene tareas seleccionadas, MUST usar su estimación propia, sin doble conteo. Una estimación seleccionada faltante MUST producir `null` con cantidad faltante; un conjunto sin observaciones MUST producir `null`.

#### Scenario: Sustituir la estimación de historia por tareas
- **WHEN** una historia tiene estimación 10 y dos tareas de 2 y 3 horas, y otra historia sin tareas tiene estimación 4
- **THEN** las horas estimadas son 9, no 19

#### Scenario: Tareas separadas por Sprint
- **WHEN** una historia asignada a A y B tiene estimación 10, tareas de 2 horas en A y 3 en B
- **THEN** proyecto estima 5 horas, A estima 2 y B estima 3, sin sumar las 10 horas de la historia

#### Scenario: Reemplazo cuando solo hay tareas de otro Sprint
- **WHEN** una historia asignada a A y B tiene estimación 10 y una tarea de 2 horas solo en A
- **THEN** A estima 2, B usa 10 y proyecto estima 2
- **WHEN** falta la estimación de esa historia
- **THEN** B devuelve `estimated_hours=null` con una observación faltante; A y proyecto mantienen 2

#### Scenario: Estimaciones faltantes
- **WHEN** una historia tiene una tarea de 2 horas y otra sin estimación, y otra historia sin tareas tampoco está estimada
- **THEN** `estimated_hours` es `null` con dos observaciones faltantes, aunque la primera historia tenga estimación propia

#### Scenario: Ámbito vacío
- **WHEN** no hay historias en el ámbito
- **THEN** `estimated_hours` es `null` y hay cero observaciones faltantes

### Requirement: Sumar horas reales observadas
En proyecto, el sistema MUST sumar todos los registros de cada historia única una vez. En Sprint, MUST sumar solo registros ligados a tareas de ese Sprint. MUST excluir registros directos a historia del total Sprint e informarlos como no atribuibles. MUST conservar sumas parciales y contar historias sin registros atribuibles; sin registros atribuibles, `actual_hours` MUST ser `null`.

#### Scenario: Registros de historia y tarea sin multiplicación
- **WHEN** se consulta un proyecto donde una historia tiene registros de 2 horas sin tarea y 3 con tarea, y otra historia no tiene registros
- **THEN** `actual_hours` es 5 y `coverage.actual_hours.stories_without_records` es 1, independientemente del número de tareas y Sprints asociados

#### Scenario: Atribución estricta entre Sprints
- **WHEN** una historia asignada a A y B tiene un registro directo de 2 horas, uno de 3 ligado a tarea de A y otro de 4 ligado a tarea de B
- **THEN** proyecto devuelve 9 horas, A devuelve 3 y B devuelve 4; cada Sprint informa un registro directo no atribuible y cero historias sin registros atribuibles

#### Scenario: Solo registros directos o de otro Sprint
- **WHEN** una historia asignada a B tiene un registro directo y otro ligado únicamente a una tarea de A
- **THEN** B devuelve `actual_hours=null`, `stories_without_records=1` y `unattributed_story_level_records=1`; no atribuye a B el esfuerzo de A

#### Scenario: Cobertura parcial atribuible
- **WHEN** A tiene dos historias asignadas, una con un registro de 3 horas ligado a tarea de A y otra con solo un registro directo
- **THEN** A devuelve 3 horas, una historia sin registros atribuibles y un registro directo no atribuible

#### Scenario: Ausencia total
- **WHEN** dos historias no tienen registros, o el ámbito está vacío
- **THEN** `actual_hours` es `null` y `stories_without_records` es 2 o 0 respectivamente

### Requirement: Diferenciar desviación horaria y porcentual
El sistema MUST calcular `deviation` como horas reales menos estimadas cuando ambas estén disponibles. MUST calcular `deviation_percentage` como esa diferencia / horas estimadas × 100 solo con ambos operandos disponibles y estimadas mayores que cero. En los demás casos MUST devolver `null` para la métrica no calculable.

#### Scenario: Diferencia y porcentaje con signo
- **WHEN** las horas estimadas son 8 y las reales son 10 o 6
- **THEN** `deviation` es 2 o -2 y `deviation_percentage` es 25 o -25 respectivamente

#### Scenario: Operando faltante o denominador cero
- **WHEN** falta cualquiera de los operandos
- **THEN** ambas desviaciones son `null`
- **WHEN** las horas estimadas son 0 y las reales están disponibles
- **THEN** `deviation` conserva la diferencia horaria y `deviation_percentage` es `null`

### Requirement: Contar defectos según detección y resolución
El sistema MUST contar defectos detectados por su Sprint de detección y resueltos por su Sprint de resolución. En proyecto MUST incluir los del proyecto y en Sprint MUST filtrar cada contador por el Sprint correspondiente. MUST NOT inferir resolución desde otro campo. Una consulta válida sin defectos MUST devolver 0 en ambos conteos.

#### Scenario: Detección y resolución en distintos Sprints
- **WHEN** un defecto del proyecto se detectó en A y tiene resolución en B
- **THEN** A cuenta una detección y ninguna resolución por ese defecto; B cuenta una resolución y ninguna detección; proyecto cuenta una de cada una

#### Scenario: No inferir ni multiplicar
- **WHEN** un defecto no tiene Sprint de resolución y su historia tiene varias tareas y registros de esfuerzo
- **THEN** no cuenta como resuelto y cuenta una sola detección, sin exigir asignación de la historia al Sprint de detección

### Requirement: Informar cobertura explícita
`coverage` MUST incluir `story_points_planned`, `story_points_completed`, `velocity` y `estimated_hours`, cada uno con `missing_observations` entero no negativo. MUST incluir `actual_hours.stories_without_records` y `actual_hours.unattributed_story_level_records`. Los contadores MUST distinguir falta de observaciones de conjuntos vacíos; no sustituyen los valores `null`.

#### Scenario: Unidades de cobertura
- **WHEN** se devuelve cobertura
- **THEN** puntos cuentan historias relevantes sin puntos; horas estimadas cuentan tareas seleccionadas sin estimación o historias sin tareas seleccionadas ni estimación; velocidad cuenta historias completadas sin puntos en Sprint o Sprints cerrados incompletos en proyecto; `stories_without_records` cuenta historias sin registros atribuibles

#### Scenario: Registros directos no atribuibles
- **WHEN** se consulta un Sprint
- **THEN** `unattributed_story_level_records` cuenta filas con `task_id=null` de sus historias asignadas, sin incluir registros de otras historias ni registros de tareas de otros Sprints
- **WHEN** se consulta el proyecto
- **THEN** ese contador es 0 porque todos sus registros directos están incluidos en las horas reales

### Requirement: Validar referencias y fallar sin filtrar detalles
El sistema MUST validar UUID antes de leer almacenamiento y responder `422 validation_failed` con campos inválidos. MUST responder `404 project_not_found` para proyecto inexistente y `404 sprint_not_found` para Sprint inexistente o ajeno. Un fallo inesperado MUST responder `500 internal_error` genérico sin métricas parciales. Las rutas de consulta MUST admitir `GET` y `HEAD` y rechazar otros métodos con `405`.

#### Scenario: Identificadores inválidos
- **WHEN** `project_id`, `sprint_id` o ambos no son UUID válidos
- **THEN** responde `422 validation_failed` con los campos inválidos sin lecturas ni escrituras

#### Scenario: Referencias inexistentes o ajenas
- **WHEN** el proyecto no existe, o existe pero el Sprint es inexistente o ajeno
- **THEN** responde `404 project_not_found` o `404 sprint_not_found` respectivamente, sin revelar datos ajenos

#### Scenario: Método y fallo de lectura
- **WHEN** se usa `POST`, `PUT`, `PATCH` o `DELETE` sobre una ruta de métricas disponible
- **THEN** responde `405` sin mutaciones
- **WHEN** falla inesperadamente el almacenamiento durante una consulta válida
- **THEN** responde `500 internal_error` genérico sin resultados parciales ni detalles internos

### Requirement: Consultar sin modificar los datos
Toda consulta MUST ser de solo lectura y devolver resultados consistentes sin multiplicar observaciones. MUST NOT crear, modificar ni eliminar proyectos, historias, asociaciones, Sprints, tareas, esfuerzos, integrantes ni defectos. Sin escrituras intermedias, consultas repetidas MUST devolver los mismos resultados.

#### Scenario: Lecturas y rechazos sin mutación
- **WHEN** se ejecutan consultas válidas, vacías, repetidas o rechazadas
- **THEN** los datos observables y sus conteos permanecen iguales antes y después
