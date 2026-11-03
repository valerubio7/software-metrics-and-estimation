# Delta para tui: visualizar el Dashboard desde la TUI (TUI-11)

Este cambio agrega a la capacidad `tui` la pantalla del Dashboard del proyecto activo (TUI-11, issue #104) y retira Dashboard del aviso de flujos pendientes de TUI-06. No modifica los contratos de la API ni el cálculo de métricas: el Dashboard consume las lecturas de proyecto y de métricas ya especificadas. La consolidación del aviso de Defectos (TUI-09) sigue pendiente de su propio cambio y no forma parte de este delta.

## MODIFIED Requirements

### Requirement: Avisar el límite de las pantallas de navegación
Las pantallas de navegación de TUI-06 NO implementan los flujos de negocio de las secciones que todavía no tienen su historia de TUI integrada. Desde este cambio, Dashboard (TUI-11) tiene su propio flujo y la TUI MUST NOT mostrar el aviso de TUI-11. La consolidación de Defectos (TUI-09) sigue pendiente de su propio cambio; hasta entonces, el aviso de esa sección se conserva tal como estaba especificado. Planning Poker (TUI-07), Esfuerzo (TUI-08) y Métricas tienen su propio flujo y NO DEBERÁN mostrar este aviso. Cuando el usuario completa la selección de navegación de una sección que todavía lo requiera, la TUI DEBERÁ mostrar un aviso que indique en qué historia estará disponible el flujo, sin comunicar que la operación se guardó.
(Previously: el aviso cubría Defectos (TUI-09) y Dashboard (TUI-11); Dashboard deja de mostrarlo con TUI-11 y la mención de Defectos queda pendiente de la consolidación de TUI-09.)

#### Scenario: Aviso al completar una selección
- **GIVEN** que el usuario eligió los elementos de navegación de una sección que todavía usa el aviso de TUI-06
- **WHEN** la TUI alcanza el final del flujo disponible
- **THEN** muestra un aviso con la historia que implementará el flujo
- **AND** no comunica éxito de una operación de negocio
- **AND** permite volver con `Enter` o `Esc`

#### Scenario: Esfuerzo ya no muestra el aviso
- **GIVEN** que el usuario eligió una historia, una tarea (o «Toda la historia») y un integrante en Esfuerzo
- **WHEN** la TUI alcanza el final del flujo de selección
- **THEN** abre el formulario de esfuerzo
- **AND** no muestra ningún aviso que mencione TUI-08 ni «No se guardó ningún dato»

#### Scenario: Dashboard ya no muestra el aviso
- **GIVEN** que el usuario confirma Dashboard en el menú
- **WHEN** la TUI abre la pantalla del proyecto activo
- **THEN** no muestra ningún aviso que mencione TUI-11 ni «No se guardó ningún dato»
- **AND** presenta el estado y las métricas del proyecto

## ADDED Requirements

### Requirement: Abrir el Dashboard desde el menú
La TUI MUST ofrecer una pantalla Dashboard que se abra directamente desde el menú para el proyecto activo, sin pedir identificadores ni obligar a elegir un Sprint. Si no hay proyecto activo, DEBERÁ abrir primero el navegador de proyectos y, al confirmarlo, continuar en Dashboard. La pantalla DEBERÁ consultar el estado del proyecto mediante `GET /projects/{project_id}` y mostrar el nombre del proyecto y su estado derivado con una etiqueta textual.

#### Scenario: Abrir Dashboard con proyecto activo
- **GIVEN** que hay un proyecto activo con estado `active`
- **WHEN** el usuario confirma Dashboard en el menú
- **THEN** la TUI abre la pantalla Dashboard y consulta `GET /projects/{project_id}`
- **AND** muestra el nombre del proyecto y el estado «en curso»
- **AND** no muestra el aviso de TUI-11 ni pide un identificador escrito a mano

#### Scenario: Abrir Dashboard sin proyecto activo
- **GIVEN** que no hay proyecto activo en la sesión
- **WHEN** el usuario confirma Dashboard en el menú
- **THEN** la TUI abre el navegador de proyectos
- **AND** al confirmar un proyecto abre Dashboard y carga su estado y sus métricas
- **AND** no pide ningún identificador escrito a mano

### Requirement: Presentar los valores de las métricas sin recalcularlos
El Dashboard MUST consultar `GET /projects/{project_id}/metrics` con el sobre de US-19 y mostrar los valores exactos devueltos por la API, sin recalcular el porcentaje de historias, la velocidad ni la desviación. Una métrica ausente (`null`) DEBERÁ mostrarse como «sin datos», nunca como cero; un cero devuelto por la API se conserva como cero.

#### Scenario: Filas de métricas del Dashboard
- **GIVEN** que la API devuelve el sobre de métricas de US-19
- **WHEN** la TUI presenta el Dashboard
- **THEN** muestra Story Points planificados y completados, velocidad, horas estimadas y reales, desviación, porcentaje de historias completadas y defectos detectados y resueltos

#### Scenario: Métrica no calculable
- **GIVEN** que la API devuelve `null` en una o más métricas del proyecto
- **WHEN** la TUI presenta el Dashboard
- **THEN** muestra «sin datos» en cada métrica ausente
- **AND** no muestra cero ni una barra para esa métrica
- **AND** conserva las métricas que sí tienen valor

#### Scenario: Valores consistentes con Métricas
- **GIVEN** que la pantalla de Métricas y el Dashboard consultan el mismo proyecto
- **WHEN** el usuario compara sus valores
- **THEN** ambos muestran los mismos números devueltos por la API
- **AND** el Dashboard no recalcula el porcentaje de historias, la velocidad ni la desviación

#### Scenario: Los valores llegan desde la API
- **GIVEN** un proyecto con Story Points planificados `21` y completados `13`
- **WHEN** la TUI presenta el Dashboard
- **THEN** muestra `13 / 21` para Story Points
- **AND** no calcula esos valores a partir de historias ni de tareas

### Requirement: Graficar las métricas con barras Unicode
El Dashboard MUST incluir barras Unicode para las relaciones disponibles: Story Points, horas, historias completadas y defectos. Las barras NO DEBERÁN ser el único portador del significado: cada fila DEBERÁ conservar su etiqueta y sus valores numéricos, y la pantalla DEBERÁ seguir siendo legible sin color y en 80 y 44 columnas.

#### Scenario: Barras visibles sin color
- **GIVEN** un proyecto con métricas disponibles
- **WHEN** la TUI presenta el Dashboard y se eliminan las secuencias de color
- **THEN** las etiquetas, los valores y los glifos `█/░` de las barras siguen presentes
- **AND** el estado del proyecto se distingue por su texto, no solo por su color

#### Scenario: Terminal angosta
- **GIVEN** una terminal de 80 o 44 columnas
- **WHEN** la TUI presenta el Dashboard
- **THEN** ninguna línea visible excede el ancho de la terminal
- **AND** las etiquetas, los valores y los glifos de las barras siguen presentes

### Requirement: Mantener el Dashboard de solo lectura y tolerante a errores
El Dashboard MUST ser de solo lectura: todas sus consultas DEBERÁN ser `GET` y NO DEBERÁ modificar datos. Si la API no responde o devuelve un error, DEBERÁ mostrar el motivo dentro de la pantalla, permanecer abierta y permitir reintentar con `r` y volver con `Esc`, sin comunicar que se guardó ningún dato. Las respuestas tardías de un proyecto anterior o de un reintento anterior DEBERÁN descartarse.

#### Scenario: API no disponible en el Dashboard
- **GIVEN** que la API no responde o devuelve un error al consultar el estado o las métricas
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el motivo del fallo dentro de la pantalla
- **AND** permanece abierta y permite reintentar con `r` o volver con `Esc`
- **AND** no comunica que se guardó ningún dato

#### Scenario: Actualizar el Dashboard
- **GIVEN** que el Dashboard muestra datos del proyecto activo
- **WHEN** el usuario presiona `r`
- **THEN** la TUI vuelve a consultar el estado y las métricas del mismo proyecto
- **AND** descarta cualquier respuesta tardía de la consulta anterior

#### Scenario: Recorrido de solo lectura
- **GIVEN** que el usuario abre el Dashboard, refresca y cambia de proyecto
- **WHEN** la TUI emite sus consultas
- **THEN** todas las requests son `GET`
- **AND** no se crea, modifica ni elimina ningún dato
