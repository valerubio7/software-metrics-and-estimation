# Especificación de reportes

## Purpose

Permitir obtener un reporte PDF consolidado y de solo lectura de un proyecto o Sprint, con sus historias, estimaciones, esfuerzo, métricas y defectos.

## ADDED Requirements

### Requirement: Ofrecer reportes PDF de proyecto y Sprint
El sistema MUST ofrecer `GET /projects/{project_id}/report.pdf` y `GET /projects/{project_id}/sprints/{sprint_id}/report.pdf`, y MUST aceptar `GET` y `HEAD`. Una consulta válida MUST responder `200` con `Content-Type: application/pdf`, un cuerpo PDF en `GET` y sin cuerpo en `HEAD`, identificando el ámbito seleccionado.

#### Scenario: Reporte de proyecto
- **WHEN** se solicita `GET /projects/{project_id}/report.pdf` para un proyecto existente
- **THEN** responde `200` con `Content-Type: application/pdf` y un PDF del proyecto

#### Scenario: Reporte de Sprint
- **WHEN** se solicita `GET /projects/{project_id}/sprints/{sprint_id}/report.pdf` para un Sprint existente del proyecto
- **THEN** responde `200` con `Content-Type: application/pdf` y un PDF del Sprint

#### Scenario: Encabezado sin cuerpo
- **WHEN** se solicita `HEAD` sobre cualquiera de las dos rutas válidas
- **THEN** responde `200` con los mismos encabezados, sin cuerpo y sin modificaciones

### Requirement: Identificar el proyecto o Sprint
El reporte MUST identificar el proyecto por nombre e identificador, con sus fechas de inicio y fin planificado. En ámbito Sprint MUST identificar además el Sprint por identificador, objetivo y estado abierto o cerrado, sin confundirlo con el proyecto.

#### Scenario: Identificación de proyecto
- **WHEN** se genera un reporte de proyecto
- **THEN** el PDF muestra el nombre, el identificador y las fechas del proyecto

#### Scenario: Identificación de Sprint
- **WHEN** se genera un reporte de Sprint
- **THEN** el PDF muestra el proyecto y el Sprint con su objetivo y estado

### Requirement: Incluir historias planificadas y completadas
En proyecto, el reporte MUST incluir todas las historias del proyecto; en Sprint, solo las asignadas a él. MUST identificar las completadas usando `sprint_stories.completed_at` del ámbito, no `stories.status`, e incluir título, prioridad y estado. Sin historias, MUST incluir una sección vacía explícita.

#### Scenario: Historias del proyecto
- **WHEN** el proyecto tiene historias asignadas y no asignadas a Sprints
- **THEN** el reporte de proyecto las incluye todas una sola vez

#### Scenario: Historias del Sprint
- **WHEN** el Sprint tiene historias asignadas y el proyecto tiene otras historias fuera de él
- **THEN** el reporte de Sprint incluye solo las asignadas al Sprint

#### Scenario: Completadas por asociación
- **WHEN** una historia tiene estado global `completada` pero su asociación al Sprint no tiene `completed_at`
- **THEN** el reporte no la marca completada en ese Sprint

#### Scenario: Ámbito sin historias
- **WHEN** el proyecto o Sprint no tiene historias
- **THEN** el reporte incluye una sección de historias vacía explícita

### Requirement: Incluir estimaciones
El reporte MUST incluir los Story Points y las horas estimadas de cada historia, y las horas estimadas de cada tarea del ámbito. MUST representar una estimación ausente como no disponible, sin convertirla en cero.

#### Scenario: Estimaciones de historia y tarea
- **WHEN** una historia tiene Story Points, horas estimadas y tareas con horas estimadas
- **THEN** el reporte muestra las tres estimaciones registradas

#### Scenario: Estimación ausente
- **WHEN** una historia o tarea no tiene estimación
- **THEN** el reporte la muestra como no disponible, sin inventar un cero

### Requirement: Incluir esfuerzo registrado
En proyecto, el reporte MUST incluir todos los registros de esfuerzo. En Sprint, MUST incluir solo los ligados a tareas de ese Sprint. Cada registro MUST mostrar fecha, integrante, historia, tarea si existe, horas y descripción. Los registros directos a historia excluidos del Sprint MUST informarse en la cobertura de métricas. Sin registros, la sección MUST ser vacía explícita.

#### Scenario: Esfuerzo del proyecto
- **WHEN** el proyecto tiene registros directos a historia y registros ligados a tareas
- **THEN** el reporte de proyecto incluye ambos una sola vez

#### Scenario: Atribución estricta en Sprint
- **WHEN** una historia asignada al Sprint tiene un registro directo y otro ligado a una tarea de otro Sprint
- **THEN** el reporte del Sprint no incluye esos registros en la tabla de esfuerzo y la cobertura informa el registro directo no atribuible

#### Scenario: Ámbito sin esfuerzo
- **WHEN** no hay registros de esfuerzo atribuibles al ámbito
- **THEN** el reporte incluye una sección de esfuerzo vacía explícita

### Requirement: Incluir métricas disponibles
El reporte MUST incluir las métricas de US-19 del mismo ámbito: puntos planificados y completados, velocidad, horas estimadas y reales, desviación, desviación porcentual, porcentaje de historias completadas y defectos detectados y resueltos. MUST usar los mismos valores y fórmulas, representar `null` como no disponible e incluir la cobertura de observaciones.

#### Scenario: Métricas del ámbito
- **WHEN** se genera un reporte de proyecto o Sprint con datos suficientes
- **THEN** el PDF muestra todas las métricas calculadas para ese ámbito

#### Scenario: Valores desconocidos y cobertura
- **WHEN** falta una estimación o no hay Sprints cerrados para la velocidad del proyecto
- **THEN** el reporte muestra la métrica como no disponible y su contador de cobertura correspondiente

#### Scenario: Consistencia con la API de métricas
- **WHEN** se genera un reporte y se consulta `GET` de métricas sobre los mismos datos
- **THEN** ambos coinciden en los valores y en la cobertura

### Requirement: Incluir defectos relacionados
En proyecto, el reporte MUST incluir todos los defectos del proyecto. En Sprint, MUST incluir los detectados en el Sprint o resueltos en él. Cada defecto MUST mostrar descripción, severidad, estado, historia, Sprint de detección y Sprint de resolución si existe. Sin defectos, la sección MUST ser vacía explícita.

#### Scenario: Defectos del proyecto
- **WHEN** el proyecto tiene defectos con y sin Sprint de resolución
- **THEN** el reporte de proyecto los incluye todos una sola vez

#### Scenario: Defectos del Sprint
- **WHEN** un defecto se detectó en el Sprint A y se resolvió en el Sprint B
- **THEN** aparece en los reportes de A y B, mostrando detección y resolución

#### Scenario: Ámbito sin defectos
- **WHEN** no hay defectos relacionados con el ámbito
- **THEN** el reporte incluye una sección de defectos vacía explícita

### Requirement: Generar un PDF válido
El cuerpo de `GET` MUST ser un PDF válido, con la firma `%PDF-` al inicio y contenido legible. El documento MUST paginarse cuando exceda una página e incluir las secciones de identificación, métricas, historias, estimaciones, esfuerzo y defectos.

#### Scenario: PDF legible y paginado
- **WHEN** el reporte tiene contenido suficiente para más de una página
- **THEN** el archivo es un PDF válido con varias páginas y las secciones identificables

### Requirement: Mantener la generación de solo lectura
El reporte MUST construirse desde una lectura consistente de los datos y MUST NOT crear, modificar ni eliminar proyectos, historias, Sprints, tareas, esfuerzos, integrantes ni defectos. Los datos de reportes sucesivos MUST coincidir mientras la fuente no cambie; solo puede variar la fecha de generación.

#### Scenario: Sin mutaciones
- **WHEN** se generan reportes válidos, vacíos o rechazados
- **THEN** los datos y sus conteos permanecen iguales antes y después

#### Scenario: Reportes repetidos
- **WHEN** se genera dos veces el mismo reporte sin cambios en la fuente
- **THEN** las secciones de datos son iguales

### Requirement: Validar referencias y errores
El sistema MUST validar los UUID antes de leer y responder `422 validation_failed` con los campos inválidos. MUST responder `404 project_not_found` o `404 sprint_not_found` para referencias inexistentes o ajenas, `405` para métodos no admitidos y `500 internal_error` genérico ante fallos. Las rutas MUST registrarse solo con esquema limpio en versión 14 o superior.

#### Scenario: Identificadores inválidos
- **WHEN** `project_id`, `sprint_id` o ambos no son UUID válidos
- **THEN** responde `422 validation_failed` con los campos inválidos, sin leer ni generar el PDF

#### Scenario: Referencias inexistentes o ajenas
- **WHEN** el proyecto no existe, o el Sprint no existe o pertenece a otro proyecto
- **THEN** responde `404 project_not_found` o `404 sprint_not_found` sin revelar datos ajenos

#### Scenario: Método no admitido
- **WHEN** se usa `POST`, `PUT`, `PATCH` o `DELETE` sobre una ruta disponible
- **THEN** responde `405` sin escrituras ni generación

#### Scenario: Fallo inesperado
- **WHEN** falla la lectura o la generación del PDF
- **THEN** responde `500 internal_error` genérico sin detalles internos ni PDF parcial

#### Scenario: Esquema incompatible
- **WHEN** el esquema no está limpio o su versión es menor que 14
- **THEN** las rutas no se registran y responden `404`
