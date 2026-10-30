# Spec Delta

## MODIFIED Requirements

### Requirement: Avisar el límite de las pantallas de navegación
Las pantallas de navegación de TUI-06 NO implementan los flujos de negocio de Planning Poker, Esfuerzo, Defectos ni Dashboard. Métricas es la excepción: TUI-10 consulta y presenta valores de la API, sin recalcularlos ni modificarlos. Cuando el usuario completa la selección de navegación de las otras secciones, la TUI DEBERÁ mostrar un aviso que indique en qué historia estará disponible el flujo, sin comunicar que la operación se guardó.

#### Scenario: Aviso al completar una selección
- **GIVEN** que el usuario eligió los elementos de navegación de Planning Poker, Esfuerzo, Defectos o Dashboard
- **WHEN** la TUI alcanza el final del flujo disponible
- **THEN** muestra un aviso con la historia que implementará el flujo
- **AND** no comunica éxito de una operación de negocio
- **AND** permite volver con `Enter` o `Esc`

## ADDED Requirements

### Requirement: Consultar métricas para el ámbito seleccionado
La TUI DEBERÁ consultar las métricas del proyecto activo o del Sprint elegido usando las rutas GET de métricas de la API. La TUI NO DEBERÁ recalcular fórmulas ni solicitar identificadores escritos a mano.

#### Scenario: Consultar métricas del proyecto
- **GIVEN** que hay un proyecto activo y se selecciona el ámbito «Proyecto»
- **WHEN** la pantalla de Métricas carga sus datos
- **THEN** consulta `GET /projects/{project_id}/metrics`
- **AND** muestra las métricas devueltas por la API

#### Scenario: Consultar métricas de un Sprint
- **GIVEN** que hay un proyecto activo y el usuario selecciona un Sprint
- **WHEN** la pantalla de Métricas carga sus datos
- **THEN** consulta `GET /projects/{project_id}/sprints/{sprint_id}/metrics`
- **AND** muestra las métricas devueltas por la API

### Requirement: Presentar las métricas de la API
La TUI DEBERÁ presentar los campos devueltos por la API: puntos planificados y completados, velocidad, horas estimadas y reales, desviación horaria y porcentual, porcentaje de historias completadas, defectos detectados y resueltos, y cobertura. La presentación NO DEBERÁ inferir ni sustituir valores.

#### Scenario: Mostrar todos los valores
- **GIVEN** que la API devuelve métricas para un proyecto o Sprint
- **WHEN** la TUI presenta el resultado
- **THEN** muestra cada valor y cobertura correspondiente sin volver a calcularlos

### Requirement: Diferenciar métricas sin datos
Cuando una métrica de la respuesta sea `null`, la TUI DEBERÁ mostrar «sin datos» y NO DEBERÁ presentarla como cero. Los valores cero devueltos por la API DEBERÁN seguir mostrándose como cero.

#### Scenario: Métrica no calculable
- **GIVEN** que una métrica devuelta por la API es `null`
- **WHEN** la TUI presenta el resultado
- **THEN** muestra «sin datos» para esa métrica
- **AND** no muestra cero en su lugar

#### Scenario: Cero informado por la API
- **GIVEN** que la API devuelve cero para una métrica
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el valor cero

### Requirement: Refrescar y manejar errores de métricas
La TUI DEBERÁ permitir volver a consultar el ámbito activo con `r`. Si la API o la respuesta falla, la TUI DEBERÁ mostrar el motivo y permanecer abierta, permitiendo reintentar o volver.

#### Scenario: Refrescar métricas
- **GIVEN** que se muestran las métricas del ámbito activo
- **WHEN** el usuario presiona `r`
- **THEN** la TUI vuelve a consultar la misma ruta GET
- **AND** actualiza la pantalla con la nueva respuesta

#### Scenario: Error de API
- **GIVEN** que la API no responde o rechaza la consulta de métricas
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el motivo disponible
- **AND** permanece abierta y permite reintentar o volver

#### Scenario: Respuesta JSON malformada
- **GIVEN** que la API responde con un cuerpo JSON que no coincide con el contrato de métricas
- **WHEN** la TUI no puede interpretar la respuesta
- **THEN** muestra el motivo de decodificación disponible
- **AND** permanece abierta y permite reintentar o volver

#### Scenario: La respuesta tardía no reemplaza el refresco
- **GIVEN** que hay una consulta de métricas activa y el usuario inicia un refresco
- **WHEN** la respuesta de la consulta anterior llega después de la respuesta del refresco
- **THEN** la TUI conserva los valores del refresco más reciente

### Requirement: Mantener métricas legibles en terminales angostas
La pantalla de Métricas DEBERÁ mantenerse legible a 80 y 44 columnas, sin desbordar el panel y sin depender exclusivamente del color para comunicar valores, estados o errores.

#### Scenario: Presentación en anchos requeridos
- **GIVEN** que se presentan métricas en una terminal de 80 o 44 columnas
- **WHEN** la TUI renderiza los valores, etiquetas y cobertura
- **THEN** el contenido permanece dentro del ancho disponible y sigue siendo identificable sin color

### Requirement: Consultar métricas sin modificar datos
La pantalla de Métricas DEBERÁ ser de solo lectura y emitir únicamente requests `GET`, incluidas las acciones de seleccionar ámbito y refrescar.

#### Scenario: Seleccionar y refrescar solo emite GET
- **GIVEN** que el usuario consulta o refresca métricas de proyecto o Sprint
- **WHEN** la TUI completa las consultas
- **THEN** todas las requests emitidas son `GET`
- **AND** ningún dato del sistema es creado, modificado o eliminado
