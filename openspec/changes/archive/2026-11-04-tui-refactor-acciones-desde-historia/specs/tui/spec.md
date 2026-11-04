# Delta para tui: acciones desde la historia (tui-refactor-acciones-desde-historia)

Rediseño de navegación de la TUI: la historia pasa a ser el centro de las acciones. Se retiran las pestañas Estimación y Esfuerzo (la barra queda con Dashboard, Backlog, Sprints y Defectos, dígitos `1` a `4`); desde la lista del backlog, `Enter` abre el detalle de la historia y desde allí se estima, se registra esfuerzo o se reporta un defecto. No cambia la API ni el contrato HTTP.

Teclas del detalle (decisión de esta fase, verificada contra `internal/tui/keys.go`): `p` Estimar, `t` Registrar esfuerzo, `d` Reportar defecto. Teclas descartadas por colisión: `e` (editar en Backlog y Proyectos), `r` (refrescar), `n` (crear en listas y cancelar en la confirmación del defecto), `h` (vista relacionada de solo lectura), `u` (cambiar estado de un defecto), `s`/`y` (confirmaciones). `d` significa «PDF» en Dashboard y Sprints, pero esas pantallas no son el detalle y el detalle no ofrece PDF.

## ADDED Requirements

### Requirement: Detalle de la historia con sus acciones
Desde la lista del backlog, `Enter` sobre la historia resaltada DEBERÁ abrir el detalle de esa historia (pantalla `story-detail`) sin emitir ninguna request. El detalle DEBERÁ mostrar únicamente los campos de la historia: título, descripción, prioridad, estado, story points, criterios de aceptación y horas estimadas, con «—» cuando un valor es nulo o está vacío. El detalle NO DEBERÁ mostrar tareas, defectos asociados ni registros de esfuerzo, ni emitir requests para obtenerlos o para obtener la historia por separado: la TUI conserva solo el identificador de la historia elegida y DEBERÁ releerla de la lista del backlog cargada cada vez que dibuja el detalle, de modo que nunca muestre datos anteriores a una estimación, una modificación o una recarga.

El detalle DEBERÁ ofrecer tres acciones como filas seleccionables, en este orden: «Estimar» (Planning Poker), «Registrar esfuerzo» y «Reportar defecto». `↑`/`↓` mueven la fila resaltada (inicialmente la primera) y `Enter` ejecuta la acción resaltada. Cada acción DEBERÁ tener además un atajo de una tecla, sin importar la fila resaltada: `p` Estimar, `t` Registrar esfuerzo y `d` Reportar defecto. Las acciones se inician con la historia ya elegida: Estimar abre directamente el selector de participantes, Registrar esfuerzo abre directamente el selector de tareas y Reportar defecto abre el formulario de alta con la historia precargada (Requirement: Reportar un defecto desde la historia). El detalle NO DEBERÁ ofrecer modificar la historia; eso se hace con `e` desde la lista.

Todo flujo iniciado desde el detalle DEBERÁ tomar el detalle de esa historia como destino de retorno: al completarse con éxito y al cancelar con `Esc` desde su primer paso, la TUI vuelve al detalle. Los pasos posteriores retroceden de a un nivel, según el requisito de cada flujo. `Esc` en el detalle DEBERÁ volver a la lista del backlog sin emitir requests, con el filtro y el cursor que tenía al abrir el detalle; si el backlog se recargó mientras tanto (por ejemplo, tras estimar), la lista se comporta como tras cualquier recarga del backlog.

Cuando el backlog se está recargando, el detalle DEBERÁ indicar la carga y no ofrecer acciones. Si la historia ya no está en el backlog cargado (por ejemplo, fue quitada antes de una recarga), el detalle DEBERÁ informarlo, no ofrecer acciones y permitir volver con `Esc`. Si la recarga falla, el detalle DEBERÁ mostrar el motivo dentro de la pantalla sin cerrarse, no ofrecer acciones y permitir volver con `Esc`. Cambiar de proyecto DEBERÁ descartar el detalle, el identificador de la historia y el destino de retorno. El detalle no es una pantalla raíz de pestaña: `Tab`, `Shift+Tab` y los dígitos no cambian de pestaña. La barra de pestañas DEBERÁ resaltar Backlog en el detalle (Requirement: Espacio de trabajo del proyecto con pestañas).

#### Scenario: Enter abre el detalle con los campos de la historia
- **GIVEN** una lista de backlog con la historia «Login», prioridad «alta», estado «pendiente», 5 story points, criterios `["A", "B"]`, 8 horas estimadas y la descripción «Ingreso de usuarios»
- **WHEN** el usuario resalta «Login» y presiona `Enter`
- **THEN** la TUI abre la pantalla `story-detail` mostrando título, descripción, prioridad, estado, story points, criterios de aceptación y horas estimadas de «Login»
- **AND** no se emite ninguna request

#### Scenario: Los valores faltantes se muestran como «—»
- **GIVEN** una historia con `story_points` nulo, `estimated_hours` nulo, descripción vacía y sin criterios de aceptación
- **WHEN** el usuario abre su detalle
- **THEN** cada uno de esos campos muestra «—»

#### Scenario: El detalle no muestra tareas, defectos ni esfuerzo
- **GIVEN** una historia con tareas, defectos y registros de esfuerzo en la API
- **WHEN** el usuario abre su detalle
- **THEN** la pantalla muestra solo los siete campos de la historia
- **AND** no se emite ninguna request, incluida cualquier consulta de tareas, de defectos o de registros de esfuerzo

#### Scenario: Enter con la lista vacía no abre el detalle
- **GIVEN** que el backlog del proyecto no tiene historias
- **WHEN** el usuario presiona `Enter`
- **THEN** la pantalla sigue siendo la lista del backlog
- **AND** no se emite ninguna request

#### Scenario: Enter con un filtro aplicado abre la historia resaltada
- **GIVEN** un backlog con «Login», «Pago» y «Perfil» y un filtro aplicado que deja visible solo «Pago»
- **WHEN** el usuario presiona `Enter` con «Pago» resaltada
- **THEN** el detalle muestra «Pago», no la historia que ocupa esa posición en la lista completa

#### Scenario: Enter mientras se escribe el filtro no abre el detalle
- **GIVEN** que el usuario está escribiendo un filtro en la lista del backlog
- **WHEN** presiona `Enter`
- **THEN** el filtro deja de editarse y la pantalla sigue siendo la lista del backlog
- **AND** un `Enter` posterior sobre la fila resaltada abre su detalle

#### Scenario: Las acciones se ofrecen como filas en orden
- **GIVEN** el detalle de una historia abierto
- **WHEN** la TUI lo dibuja
- **THEN** muestra las filas «Estimar», «Registrar esfuerzo» y «Reportar defecto», en ese orden, con la primera resaltada
- **AND** `↓` resalta la siguiente fila y `↑` la anterior

#### Scenario: Enter ejecuta la acción resaltada
- **GIVEN** el detalle de la historia «Login» con la fila «Registrar esfuerzo» resaltada
- **WHEN** el usuario presiona `Enter`
- **THEN** la TUI abre el selector de tareas de «Login», con la historia ya elegida y sin pasar por ningún selector de historias

#### Scenario: Cada atajo inicia su acción
- **GIVEN** el detalle de la historia «Login» con la primera fila resaltada
- **WHEN** el usuario presiona `p`, y en otra ocasión `t`, y en otra `d`
- **THEN** `p` abre el selector de participantes de Planning Poker para «Login»
- **AND** `t` abre el selector de tareas de «Login»
- **AND** `d` abre el formulario de alta de defecto con «Login» precargada

#### Scenario: Estimar desde el detalle y volver con Esc
- **GIVEN** que el usuario abrió Estimar desde el detalle de «Login» y está en el selector de participantes
- **WHEN** presiona `Esc`
- **THEN** la TUI vuelve al detalle de «Login» sin emitir requests

#### Scenario: El detalle refleja los Story Points tras estimar
- **GIVEN** el detalle de «Login» con story points «—» y una estimación de 8 confirmada con éxito
- **WHEN** la TUI vuelve al detalle y recarga el backlog
- **THEN** el detalle muestra 8 story points de «Login», releídos del backlog recargado y no de la copia anterior

#### Scenario: Mientras el backlog se recarga el detalle indica la carga
- **GIVEN** que una estimación se guardó con éxito y la recarga del backlog sigue en curso
- **WHEN** la TUI muestra el detalle de la historia
- **THEN** indica que está cargando y no muestra el mensaje de historia inexistente
- **AND** no ofrece acciones ni responde a `p`, `t` ni `d`

#### Scenario: La historia ya no está tras la recarga
- **GIVEN** el detalle de «Login» abierto y un backlog recargado que ya no contiene esa historia
- **WHEN** la TUI dibuja el detalle
- **THEN** informa que la historia ya no está en el backlog y no ofrece acciones
- **AND** `p`, `t` y `d` no abren ningún flujo
- **AND** `Esc` vuelve a la lista del backlog

#### Scenario: Falla la recarga del backlog
- **GIVEN** que tras una acción exitosa la recarga del backlog responde HTTP 500
- **WHEN** la TUI presenta el detalle
- **THEN** muestra el error de la API dentro de la pantalla sin cerrarse y sin ofrecer acciones
- **AND** `Esc` vuelve a la lista del backlog

#### Scenario: Esc vuelve al backlog con el filtro y el cursor
- **GIVEN** una lista con el filtro «pa» aplicado y la segunda fila resaltada, de la que se abrió el detalle
- **WHEN** el usuario presiona `Esc` en el detalle, sin que el backlog se haya recargado
- **THEN** la TUI vuelve a la lista con el filtro «pa» y la segunda fila resaltada
- **AND** no emite ninguna request

#### Scenario: El detalle no ofrece modificar la historia
- **GIVEN** el detalle de una historia
- **WHEN** el usuario presiona `e`
- **THEN** la TUI no abre el formulario de modificación y la pantalla no cambia

#### Scenario: Error de la API en un paso de la acción lanzada
- **GIVEN** que el usuario eligió Registrar esfuerzo en el detalle y la API responde HTTP 500 a la consulta de tareas
- **WHEN** la TUI presenta el selector de tareas
- **THEN** muestra el error dentro del selector sin cerrarse y sin ofrecer la fila «Toda la historia (sin tarea)»
- **AND** `Esc` vuelve al detalle de la misma historia

#### Scenario: El detalle resalta la pestaña Backlog
- **GIVEN** el detalle de una historia abierto
- **WHEN** la TUI dibuja la barra de pestañas y la miga de pan
- **THEN** la pestaña resaltada es Backlog
- **AND** la miga empieza con «Proyectos», el nombre del proyecto y Backlog, y termina en el detalle de la historia

#### Scenario: Tab y los dígitos no cambian de pestaña en el detalle
- **GIVEN** el detalle de una historia abierto
- **WHEN** el usuario presiona `Tab`, `Shift+Tab` o un dígito de `1` a `4`
- **THEN** la TUI permanece en el detalle y no emite requests

#### Scenario: Pie y ayuda del detalle
- **GIVEN** el detalle de una historia abierto
- **WHEN** la TUI dibuja el pie y el usuario abre la ayuda con `?`
- **THEN** el pie muestra como máximo cinco atajos, entre ellos `Enter` y `Esc`, y `? más` si hay otros
- **AND** la ayuda lista en «Acciones» `p` (estimar), `t` (registrar esfuerzo) y `d` (reportar defecto)
- **AND** la ayuda no lista `Tab/Shift+Tab` ni `1–4`

#### Scenario: Cambiar de proyecto descarta el detalle y el destino de retorno
- **GIVEN** el detalle de «Login» del proyecto `p-1` abierto
- **WHEN** el usuario vuelve a Proyectos con `Esc` (desde la lista del backlog), abre el proyecto `p-2` y abre su pestaña Backlog
- **THEN** la TUI muestra la lista del backlog de `p-2`, no el detalle de «Login»
- **AND** iniciar y cancelar una acción desde una historia de `p-2` vuelve al detalle de esa historia y no al de `p-1`

### Requirement: Reportar un defecto desde la historia
La acción «Reportar defecto» del detalle de la historia DEBERÁ abrir el formulario de alta de defecto con la historia del detalle ya elegida y mostrada como «Historia relacionada», omitiendo el selector de historias del alta (`defect-story-select`). La historia NO DEBERÁ poder cambiarse en este flujo. Con la historia precargada, `Enter` en el último campo del formulario DEBERÁ abrir directamente el selector de Sprints abiertos; al elegir un Sprint se vuelve al formulario y `Enter` abre la confirmación, que envía `POST /projects/{pid}/defects` con el identificador de la historia precargada. `Esc` en el selector de Sprints DEBERÁ volver al formulario conservando el borrador, y `Esc` en la confirmación DEBERÁ volver al formulario.

El destino de retorno de este flujo es el detalle de la historia: `Esc` en el formulario (que descarta el borrador), cancelar la confirmación con `n` y el éxito del registro DEBERÁN volver al detalle. El éxito DEBERÁ mostrar en el detalle el aviso «✓ Defecto registrado: » seguido de la descripción, sin abrir la lista de defectos. Se conservan sin cambios el indicador y el bloqueo de repeticiones mientras el `POST` está en curso (la confirmación ignora `s`, `n` y `Esc`, de modo que no se emite una segunda request), la presentación del rechazo de la API en la confirmación con los valores conservados, y el descarte de toda respuesta (de Sprints o del registro) que ya no corresponda a la pantalla, a la consulta o al proyecto activos. Mientras el alta se hace desde la historia, la barra de pestañas DEBERÁ resaltar Backlog y no Defectos (Requirement: Espacio de trabajo del proyecto con pestañas). El alta iniciada con `n` desde la pestaña Defectos DEBERÁ conservar su flujo actual, con selector de historias, y su destino de retorno sigue siendo la lista de defectos.

#### Scenario: El formulario se abre con la historia precargada
- **GIVEN** el detalle de la historia «Login»
- **WHEN** el usuario elige «Reportar defecto» con `Enter` o con `d`
- **THEN** la TUI abre el formulario de alta mostrando «Historia relacionada: Login»
- **AND** no se muestra el selector de historias ni se emite una consulta de historias

#### Scenario: Enter en el último campo abre el selector de Sprints
- **GIVEN** el formulario de alta iniciado desde el detalle, con descripción escrita y el foco en el último campo
- **WHEN** el usuario presiona `Enter`
- **THEN** la TUI emite `GET /projects/{pid}/sprints/open` y muestra el selector de Sprints abiertos
- **AND** no pasa por `defect-story-select`

#### Scenario: Esc en el selector de Sprints vuelve al formulario
- **GIVEN** el selector de Sprints abierto desde un alta iniciada en el detalle
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI vuelve al formulario con la descripción, la severidad, el estado y la historia conservados
- **AND** no vuelve a ningún selector de historias

#### Scenario: Registrar el defecto envía la historia precargada
- **GIVEN** la confirmación de un defecto iniciado desde el detalle de «Login» (`story_id` `s-1`) con Sprint `sp-1`
- **WHEN** el usuario presiona `s`
- **THEN** la TUI emite una única request `POST /projects/p-1/defects` cuyo cuerpo contiene `story_id` `s-1`, `sprint_id` `sp-1`, la descripción, la severidad y el estado escritos

#### Scenario: El éxito vuelve al detalle con un aviso
- **GIVEN** la confirmación de un defecto con descripción «No guarda» iniciado desde el detalle de «Login»
- **WHEN** la API responde con éxito
- **THEN** la TUI muestra el detalle de «Login» con el aviso «✓ Defecto registrado: No guarda»
- **AND** no muestra la lista de defectos

#### Scenario: Cancelar la confirmación vuelve al detalle
- **GIVEN** la confirmación de un defecto iniciado desde el detalle de «Login»
- **WHEN** el usuario presiona `n`
- **THEN** la TUI vuelve al detalle de «Login» sin emitir ninguna request de registro

#### Scenario: Esc en la confirmación vuelve al borrador
- **GIVEN** la confirmación de un defecto iniciado desde el detalle y sin registro en curso
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI vuelve al formulario con el borrador conservado

#### Scenario: Esc en el formulario vuelve al detalle y descarta el borrador
- **GIVEN** el formulario de alta iniciado desde el detalle de «Login» con la descripción «Borrador» escrita
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI vuelve al detalle de «Login»
- **AND** al volver a elegir «Reportar defecto» el formulario se abre con la descripción vacía y «Login» precargada

#### Scenario: La API rechaza el registro
- **GIVEN** la confirmación de un defecto iniciado desde el detalle
- **WHEN** la API responde HTTP 422 o falla la conexión
- **THEN** la TUI permanece en la confirmación con los valores conservados y muestra el motivo informado, sin comunicar éxito
- **AND** el usuario puede reintentar con `s` o volver al borrador con `Esc`

#### Scenario: Un doble envío emite una sola request
- **GIVEN** la confirmación con el registro ya enviado y sin respuesta
- **WHEN** el usuario presiona `s` de nuevo, o `n` o `Esc`
- **THEN** la TUI no emite otra request, no sale de la confirmación y mantiene el indicador de registro

#### Scenario: Falla la consulta de Sprints
- **GIVEN** un alta iniciada desde el detalle y la API responde HTTP 500 a la consulta de Sprints abiertos
- **WHEN** la TUI presenta el selector de Sprints
- **THEN** muestra el error dentro del selector sin cerrarse
- **AND** permite reintentar con `r` o volver al formulario con `Esc`

#### Scenario: Respuesta tardía de Sprints tras volver al formulario
- **GIVEN** que la consulta de Sprints del alta sigue en curso
- **WHEN** el usuario presiona `Esc` y después llega la respuesta
- **THEN** la TUI sigue mostrando el formulario sin cambios y la respuesta se descarta

#### Scenario: Respuesta de un proyecto anterior
- **GIVEN** un alta iniciada en el proyecto `p-1` cuya respuesta llega después de abrir el proyecto `p-2`
- **WHEN** la TUI recibe la respuesta
- **THEN** la descarta sin cambiar de pantalla ni mostrar avisos de `p-1`

#### Scenario: La pestaña resaltada depende del origen del alta
- **GIVEN** un alta de defecto en curso
- **WHEN** la TUI dibuja la barra de pestañas
- **THEN** si se inició desde el detalle de una historia resalta Backlog en el formulario, el selector de Sprints y la confirmación
- **AND** si se inició con `n` desde la pestaña Defectos resalta Defectos

#### Scenario: El alta desde la pestaña Defectos conserva su flujo
- **GIVEN** la lista de defectos y el usuario presiona `n`
- **WHEN** completa el formulario y presiona `Enter` en el último campo
- **THEN** la TUI abre el selector de historias del alta
- **AND** `Esc`, cancelar con `n` y el éxito del registro vuelven a la lista de defectos, como antes de este cambio

## MODIFIED Requirements

### Requirement: Navegación de solo lectura
Navegar entre el navegador de proyectos, las pestañas y sus pantallas DEBERÁ ser de solo lectura. Solo las operaciones explícitas de escritura emiten requests que modifican datos: proyectos, historias, Sprints, el `PUT` del Story Points de Planning Poker, el `POST` de esfuerzo y las operaciones de defectos, cada una definida en su propio requisito. Las demás pantallas y selectores, incluido el detalle de la historia, DEBERÁN emitir únicamente consultas `GET`.
(Previously: nombraba el `PUT` del Story Points de «Estimación» y el escenario del Sprint 3 recorría las pestañas Dashboard, Estimación, Esfuerzo y Defectos.)

#### Scenario: Navegar no muta
- **WHEN** el usuario navega entre el navegador de proyectos, las pestañas y sus pantallas sin confirmar una operación explícita de escritura
- **THEN** no se emite ninguna request que modifique datos

#### Scenario: Consultar el backlog no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario abre la lista del backlog o cambia de proyecto
- **THEN** las únicas requests emitidas son las consultas `GET` (del catálogo de proyectos y del backlog) y no se emite ningún `POST` ni `PUT`

#### Scenario: Navegar por las pestañas y las acciones de la historia no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario recorre las pestañas Dashboard, Backlog, Sprints y Defectos, abre el detalle de una historia, elige filas y avanza entre los selectores de Estimar, Registrar esfuerzo y Reportar defecto, sin confirmar el guardado de Story Points, el envío del formulario de esfuerzo ni el registro del defecto
- **THEN** todas las requests emitidas son `GET`
- **AND** no se crea, modifica ni elimina ningún dato

#### Scenario: Abrir y completar el formulario de esfuerzo no muta hasta confirmar
- **GIVEN** que el usuario llegó al formulario de esfuerzo desde el detalle de una historia y los selectores de tarea e integrante
- **WHEN** edita los campos, navega entre ellos y retrocede con `Esc` sin confirmar desde «Horas»
- **THEN** no se emite ningún `POST` ni `PUT`

#### Scenario: Solo la confirmación del formulario emite el `POST` de esfuerzo
- **GIVEN** el formulario de esfuerzo con datos válidos
- **WHEN** el usuario confirma desde «Horas»
- **THEN** la única request que modifica datos es un `POST` a `/projects/{pid}/stories/{sid}/effort-records`
- **AND** las demás requests de la sección, incluida la consulta de métricas posterior, son `GET`

#### Scenario: Planning Poker solo escribe al confirmar el Story Points
- **GIVEN** una ronda de Planning Poker con participantes y votos
- **WHEN** el facilitador vota, revela y elige un valor sin confirmar el guardado
- **THEN** no se emite ninguna request que modifique datos
- **AND** al confirmar se emite únicamente el `PUT /projects/{pid}/stories/{sid}/story-points`

### Requirement: Operar el backlog mediante la TUI
La TUI DEBERÁ permitir consultar, crear, modificar y abrir en detalle las historias del proyecto del espacio de trabajo desde la pestaña Backlog, con la API existente (`GET`, `POST` y `PUT` de `/projects/{pid}/stories`) y sin reglas de negocio. La pantalla conserva el nombre `backlog` y se comporta según el patrón navegable (Requirement: Listas navegables con teclas uniformes), sin filas de acción: `Enter` abre el detalle de la historia resaltada (Requirement: Detalle de la historia con sus acciones), `e` modifica la historia resaltada, `n` crea y `r` refresca. Las teclas `P` y `E` mayúsculas ya no inician ningún flujo desde la lista. Cambiar de proyecto se hace volviendo a Proyectos con `Esc`.
(Previously: `Enter` modificaba la historia resaltada; `P` y `E` iniciaban Estimación y Esfuerzo desde la lista.)

#### Scenario: La opción Backlog da acceso a la gestión
- **GIVEN** que el espacio de trabajo de un proyecto está abierto
- **WHEN** el usuario abre la pestaña Backlog con `2` o `Tab`
- **THEN** la TUI consulta `GET /projects/{pid}/stories` y presenta la lista del backlog
- **AND** la pantalla activa conserva el nombre `backlog`

#### Scenario: Acceder sin proyecto activo abre el selector
- **GIVEN** que la TUI muestra el navegador de proyectos y todavía no se abrió ningún proyecto
- **WHEN** el usuario mira la pantalla
- **THEN** la pestaña Backlog no está disponible y no se consulta ningún backlog
- **AND** la TUI consulta el backlog solo después de que el usuario abre un proyecto con `Enter`

#### Scenario: Crear, abrir y modificar historias desde el navegador
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `n`
- **THEN** la TUI abre el formulario de creación vacío
- **AND** si resalta una historia y presiona `Enter` abre su detalle, sin abrir el formulario de modificación
- **AND** si resalta una historia y presiona `e` abre el formulario de modificación precargado

#### Scenario: `e` modifica y `Esc` vuelve a la lista
- **GIVEN** el formulario de modificación abierto con `e` desde la lista
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI vuelve a la lista del backlog

#### Scenario: `P` y `E` ya no inician flujos
- **GIVEN** que la lista del backlog tiene una historia resaltada
- **WHEN** el usuario presiona `P` o `E`
- **THEN** la TUI no abre Planning Poker, Esfuerzo ni ningún flujo y no emite ninguna request
- **AND** la ayuda de la lista no ofrece esos atajos

#### Scenario: Las letras no reservadas siguen iniciando la búsqueda
- **GIVEN** que la lista del backlog está activa
- **WHEN** el usuario presiona `p`
- **THEN** la TUI inicia la búsqueda con ese carácter ya escrito
- **AND** `e` no inicia la búsqueda sino la modificación; para filtrar por texto con `e` se empieza con `/`

#### Scenario: Cambiar de proyecto desde el navegador
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `Esc` sin filtro aplicado
- **THEN** la TUI vuelve al navegador de proyectos, donde puede abrir otro proyecto
- **AND** `c` no cambia de proyecto

### Requirement: Modificar historias desde la lista
La TUI DEBERÁ permitir modificar una historia resaltándola en la lista cargada del proyecto activo y presionando `e`. El formulario DEBERÁ abrirse precargado con los valores actuales de la historia resaltada (`title`, `description`, `priority`, `status`, `acceptance_criteria` unidos con `;` y `estimated_hours`). Como el `PUT` es un reemplazo total, la TUI DEBERÁ enviar siempre las seis claves `title`, `description`, `priority`, `status`, `acceptance_criteria` y `estimated_hours` a `PUT /projects/{pid}/stories/{sid}`, con `estimated_hours` igual a `null` cuando el campo queda vacío. La prioridad y el estado se ingresan como texto libre y la TUI NO validará sus valores permitidos. `story_points` NO es editable y NO se envía. Mientras se escribe un filtro, `e` es un carácter de texto y no abre la modificación.
(Previously: la historia se seleccionaba de la lista y el formulario se abría con `Enter`.)

#### Scenario: El formulario se precarga con la historia seleccionada
- **GIVEN** una lista cargada con una historia de título «Login», prioridad «alta», estado «pendiente», criterios `["A", "B"]` y `estimated_hours` 5
- **WHEN** el usuario resalta esa historia y presiona `e`
- **THEN** el formulario muestra «Login», «alta», «pendiente», «A; B» (criterios unidos con `;`) y «5» en los campos correspondientes

#### Scenario: Modificar sin cambios envía las seis claves
- **GIVEN** que el formulario está precargado con una historia y el usuario no cambia ningún campo
- **WHEN** confirma la modificación
- **THEN** la TUI envía `PUT /projects/{pid}/stories/{sid}` con el identificador de la historia seleccionada
- **AND** el cuerpo contiene exactamente las claves `title`, `description`, `priority`, `status`, `acceptance_criteria` y `estimated_hours` con los valores precargados

#### Scenario: `estimated_hours` vacío se envía como `null`
- **GIVEN** que el usuario vacía el campo `estimated_hours` del formulario de modificación
- **WHEN** confirma la modificación
- **THEN** el cuerpo enviado contiene la clave `estimated_hours` con valor `null`
- **AND** la clave no se omite

#### Scenario: Modificar un campo conserva el resto
- **GIVEN** que el formulario está precargado y el usuario cambia solo el título
- **WHEN** confirma la modificación
- **THEN** el cuerpo enviado contiene el título nuevo y los demás valores precargados sin alteración

#### Scenario: Modificar con la lista vacía
- **GIVEN** que el backlog del proyecto activo está vacío
- **WHEN** el usuario presiona `e`
- **THEN** la TUI no abre el formulario de modificación
- **AND** no emite ninguna request a la API

#### Scenario: Modificar con un filtro aplicado
- **GIVEN** un backlog con «Login» y «Pago» y un filtro aplicado que deja visible solo «Pago»
- **WHEN** el usuario presiona `e` con «Pago» resaltada
- **THEN** el formulario se abre precargado con «Pago»

#### Scenario: `e` dentro del filtro es texto
- **GIVEN** que el usuario está escribiendo un filtro en la lista del backlog
- **WHEN** presiona `e`
- **THEN** el carácter se agrega al filtro y no se abre el formulario de modificación

### Requirement: Listas navegables con teclas uniformes

Las pantallas que muestran colecciones (proyectos, historias y Sprints) DEBERÁN comportarse como un navegador con el mismo patrón de interacción: la búsqueda se activa con `/` y cualquier otro carácter la inicia ya escrito, salvo las teclas reservadas de la pantalla y, en las pantallas raíz del espacio de trabajo, los dígitos `1` a `4`; `↑`/`↓` mueven la selección; `Enter` ejecuta la acción primaria de la fila resaltada; `n` inicia la creación; `e` modifica el elemento resaltado (por ejemplo, en Proyectos y en el Backlog, donde la acción primaria de la fila no es editar); `r` refresca el listado; `x` ejecuta una acción destructiva sobre la fila resaltada (pidiendo confirmación); `h` abre una vista relacionada de solo lectura; y `Esc` primero limpia el filtro y luego vuelve a la pantalla anterior. Cada fila DEBERÁ mostrar el contenido y sus etiquetas, con un marcador inicial cuando corresponda (por ejemplo `✓` en el proyecto abierto) y una forma corta del identificador en las listas que lo necesiten para distinguir elementos (por ejemplo, proyectos y Sprints). Ninguna lista DEBERÁ pedir un identificador escrito a mano. Las pantallas de formulario NO adoptan este patrón.
(Previously: los dígitos reservados eran `1` a `6` y `e` modificaba solo donde editar no era la acción primaria porque `Enter` ya modificaba en el Backlog.)

#### Scenario: Buscar en cualquier colección
- **GIVEN** que una colección está visible (proyectos, backlog, Sprints abiertos, historias de un Sprint o Sprints cerrados)
- **WHEN** el usuario presiona `/` o empieza a escribir una letra
- **THEN** la lista se reduce a las coincidencias sin emitir requests

#### Scenario: Acción primaria con Enter
- **GIVEN** que una fila está resaltada en una colección
- **WHEN** el usuario presiona `Enter`
- **THEN** la TUI ejecuta la acción primaria de esa colección sobre la fila resaltada
- **AND** en el Backlog esa acción es abrir el detalle de la historia

#### Scenario: `e` modifica en el Backlog y en Proyectos
- **GIVEN** que una fila está resaltada en la lista del backlog o en el navegador de proyectos
- **WHEN** el usuario presiona `e`
- **THEN** la TUI abre el formulario de modificación de esa fila

#### Scenario: Los dígitos 1 a 4 están reservados en las pantallas raíz
- **GIVEN** que una pantalla raíz del espacio de trabajo muestra una colección sin filtro en edición
- **WHEN** el usuario presiona un dígito de `1` a `4`
- **THEN** la TUI abre la pestaña de esa posición y no inicia la búsqueda
- **AND** `5` y `6` no cambian de pestaña

#### Scenario: Esc limpia el filtro antes de volver
- **GIVEN** que una lista tiene un filtro aplicado
- **WHEN** el usuario presiona `Esc`
- **THEN** el filtro se borra y la lista vuelve a mostrarlo todo
- **AND** un segundo `Esc` vuelve a la pantalla anterior

#### Scenario: Ninguna lista pide identificadores a mano
- **WHEN** el usuario opera cualquier colección
- **THEN** la TUI nunca solicita que se escriba un identificador

### Requirement: Estimar historias con Planning Poker

La TUI DEBERÁ permitir estimar una historia del backlog del proyecto activo desde la acción «Estimar» de su detalle (Requirement: Detalle de la historia con sus acciones), con la historia ya elegida, y elegir uno o más integrantes del proyecto como participantes. El facilitador ingresará un voto por participante en la misma terminal usando `0, 1, 2, 3, 5, 8, 13, 20, 40, 100` o `?`. Los votos serán transitorios en memoria y NO DEBERÁN mostrarse hasta que el facilitador revele explícitamente la ronda. La revelación mostrará cada participante y su voto, e indicará si hay diferencias. Una nueva ronda DEBERÁ ocultar y reiniciar los votos. La TUI NO calculará consenso ni persistirá votos.

Luego de revisar la ronda, el facilitador DEBERÁ elegir explícitamente un Story Points numérico acordado y confirmarlo. La TUI enviará únicamente `PUT /projects/{project_id}/stories/{story_id}/story-points` con `{"story_points": <integer>}`. La validación corresponde a la API; `?` no se puede guardar. Tras el éxito se actualizará el backlog mediante una nueva consulta y la TUI volverá al detalle de la historia, que mostrará el Story Points guardado. `Esc` desde el selector de participantes, primer paso del flujo, DEBERÁ volver al detalle de la historia sin emitir requests. Ante errores 404, 409 o 422 se mostrará el motivo de la API y no se afirmará que el cambio se guardó ni se modificará el Story Points local. La TUI DEBERÁ descartar sin efecto la respuesta de integrantes que llegue cuando ya no corresponda al paso activo.
(Previously: la historia se elegía de un selector de la pestaña Estimación o con el atajo `P`, y tras el éxito se mostraba el backlog actualizado.)

#### Scenario: Votos privados y revelación explícita
- **GIVEN** que se seleccionaron participantes y se ingresaron votos
- **WHEN** la ronda todavía no fue revelada
- **THEN** las vistas de voto no muestran ningún valor de voto
- **AND** solo la acción explícita de revelar muestra los votos e indica si difieren

#### Scenario: Reiniciar una ronda
- **GIVEN** que los votos de una ronda fueron revelados
- **WHEN** el facilitador inicia una nueva ronda
- **THEN** los votos anteriores dejan de mostrarse y cada participante vuelve a votar

#### Scenario: Estimar empieza en los participantes con la historia elegida
- **GIVEN** el detalle de la historia «Login»
- **WHEN** el usuario elige «Estimar» con `Enter` o con `p`
- **THEN** la TUI muestra el selector de participantes de «Login» sin pasar por ningún selector de historias
- **AND** las requests emitidas hasta ese punto son `GET`

#### Scenario: Guardar el estimado acordado
- **GIVEN** una historia seleccionada y una ronda revisada
- **WHEN** el facilitador elige un valor numérico y confirma explícitamente
- **THEN** la TUI envía el PUT dedicado con el cuerpo `story_points`
- **AND** tras el éxito vuelve al detalle de la historia y recarga el backlog, de modo que el detalle muestra el Story Points guardado

#### Scenario: Fallo al guardar el estimado
- **GIVEN** que la API responde 404, 409 o 422
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el motivo informado por la API sin comunicar éxito ni alterar el Story Points local

#### Scenario: Esc desde los participantes vuelve al detalle
- **GIVEN** el selector de participantes abierto desde el detalle de «Login»
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI vuelve al detalle de «Login» sin emitir requests
- **AND** un `Esc` con un filtro aplicado limpia primero el filtro

#### Scenario: Respuesta tardía de integrantes tras volver al detalle
- **GIVEN** que el usuario eligió «Estimar» y la consulta de integrantes sigue en curso
- **WHEN** presiona `Esc` y después llega la respuesta
- **THEN** la TUI sigue mostrando el detalle sin cambios
- **AND** la respuesta se descarta

### Requirement: Elegir integrantes, historias, tareas y Sprints con selectores navegables
La TUI DEBERÁ ofrecer selectores de integrante, historia, tarea y Sprint como listas navegables con el mismo patrón de las colecciones existentes: la búsqueda se activa con `/` o al empezar a escribir, `↑`/`↓` mueven la selección, `Enter` ejecuta la acción primaria, `Esc` limpia el filtro y luego vuelve, `r` refresca y `?` abre la ayuda. Ningún selector DEBERÁ pedir un identificador escrito a mano. Los selectores DEBERÁN mostrar un estado de carga, un estado vacío y el error de la API sin cerrar la TUI. El selector de tareas de Esfuerzo es la excepción del estado vacío: antepone la fila «Toda la historia (sin tarea)» (Requirement: Elegir historia, tarea e integrante para registrar esfuerzo), por lo que ante una historia sin tareas muestra esa fila e indica que la historia no tiene tareas, en lugar de un estado vacío bloqueante. Los selectores de historia subsisten en la asignación de historias a un Sprint y en el alta de defecto iniciada desde la pestaña Defectos; Estimar, Registrar esfuerzo y Reportar defecto desde el detalle ya parten de la historia elegida y no usan un selector de historias.
(Previously: la lista de selectores incluía los selectores de historia de las pestañas Estimación y Esfuerzo, que se retiran.)

#### Scenario: Buscar en un selector
- **GIVEN** que un selector muestra una colección
- **WHEN** el usuario presiona `/` o empieza a escribir
- **THEN** la lista se reduce a las coincidencias sin emitir requests
- **AND** `Esc` limpia el filtro antes de volver

#### Scenario: Elegir una fila
- **GIVEN** que un selector tiene una fila resaltada
- **WHEN** el usuario presiona `Enter`
- **THEN** la TUI ejecuta la acción primaria de esa fila
- **AND** no solicita ningún UUID

#### Scenario: Refrescar un selector
- **GIVEN** que un selector muestra datos de la API
- **WHEN** el usuario presiona `r`
- **THEN** la TUI vuelve a consultar la colección del paso actual
- **AND** conserva el filtro escrito

#### Scenario: Selector vacío
- **GIVEN** que la API responde una colección vacía en un selector distinto del de tareas de Esfuerzo (por ejemplo, el de integrantes)
- **WHEN** la TUI presenta el selector
- **THEN** muestra un estado vacío claro
- **AND** permite volver o refrescar sin cerrarse

#### Scenario: El selector de tareas de Esfuerzo no queda vacío
- **GIVEN** que la API responde una lista de tareas vacía para la historia elegida en Esfuerzo
- **WHEN** la TUI presenta el selector de tareas
- **THEN** muestra la fila «Toda la historia (sin tarea)» e indica que la historia no tiene tareas
- **AND** permite volver o refrescar sin cerrarse

#### Scenario: Los tipos de selector son navegables
- **GIVEN** que el usuario recorre Estimar y Registrar esfuerzo desde el detalle de una historia, y el alta de defecto desde la pestaña Defectos
- **WHEN** elige un integrante, una tarea, una historia (en el alta de defecto desde Defectos) y un Sprint
- **THEN** cada colección se presenta como una lista navegable con las teclas comunes
- **AND** la selección de una fila no emite requests que modifiquen datos

#### Scenario: Estimar y Registrar esfuerzo no muestran un selector de historias
- **GIVEN** el detalle de una historia
- **WHEN** el usuario inicia Estimar o Registrar esfuerzo
- **THEN** el primer selector mostrado es el de participantes o el de tareas, respectivamente
- **AND** la TUI no presenta ningún selector de historias en esos flujos

### Requirement: Elegir historia, tarea e integrante para registrar esfuerzo
El registro de esfuerzo DEBERÁ iniciarse desde la acción «Registrar esfuerzo» del detalle de la historia (Requirement: Detalle de la historia con sus acciones), que deja la historia ya elegida y abre directamente el selector de tareas. El flujo DEBERÁ guiarse con los selectores navegables existentes, en este orden: tarea de esa historia (o la historia completa) e integrante del proyecto activo; no existe un selector de historias propio de Esfuerzo. Una vez cargadas las tareas de la historia elegida, el selector de tareas DEBERÁ anteponer, como primera fila, la fila sintética «Toda la historia (sin tarea)», seguida de las tareas en el orden informado por la API; la fila sintética DEBERÁ estar presente también cuando la historia no tiene tareas. Elegir una tarea registra el esfuerzo sobre esa tarea; elegir la fila sintética lo registra sobre la historia completa. Elegir cualquiera de las dos filas DEBERÁ abrir el selector de integrantes del proyecto activo, que en Esfuerzo es de selección simple: `Enter` elige el integrante resaltado y abre el formulario de esfuerzo (Requirement: Completar el formulario de esfuerzo), y `Space` no marca integrantes. Ningún paso DEBERÁ pedir un identificador escrito a mano. El avance entre selectores y la apertura del formulario NO DEBERÁN emitir requests que modifiquen datos. Si el proyecto no tiene integrantes, el selector de integrantes DEBERÁ mostrar un estado vacío propio que informe que el proyecto no tiene integrantes registrados, que indique que los integrantes se dan de alta mediante la API y que permita volver con `Esc` y refrescar con `r`.
(Previously: el flujo comenzaba en un selector de historias de la pestaña Esfuerzo, o en el selector de tareas con la historia preelegida mediante `E`.)

#### Scenario: Flujo completo hasta el formulario
- **GIVEN** un proyecto activo con la historia «Login», que tiene la tarea «Diseñar», y el integrante «Ana Pérez»
- **WHEN** el usuario abre el detalle de «Login», elige «Registrar esfuerzo», elige «Diseñar» y elige «Ana Pérez» con `Enter`
- **THEN** la TUI abre el formulario de esfuerzo mostrando la historia «Login», la tarea «Diseñar» y el integrante «Ana Pérez»
- **AND** todas las requests emitidas hasta ese punto son `GET` (tareas e integrantes, además de las del backlog ya cargado)
- **AND** la TUI no pidió ningún identificador ni mostró un selector de historias

#### Scenario: La fila «Toda la historia» encabeza las tareas
- **GIVEN** que la historia elegida tiene las tareas «Diseñar» y «Probar», en ese orden
- **WHEN** la TUI presenta el selector de tareas de Esfuerzo
- **THEN** las filas son, en orden, «Toda la historia (sin tarea)», «Diseñar» y «Probar»

#### Scenario: Una historia sin tareas admite esfuerzo
- **GIVEN** que la API responde HTTP 200 con una lista de tareas vacía para la historia elegida
- **WHEN** la TUI presenta el selector de tareas de Esfuerzo
- **THEN** muestra la fila «Toda la historia (sin tarea)» y no la sustituye por un estado vacío bloqueante
- **AND** `Enter` sobre esa fila abre el selector de integrantes

#### Scenario: Refrescar las tareas conserva la fila sintética
- **GIVEN** el selector de tareas de Esfuerzo cargado
- **WHEN** el usuario presiona `r` y la API responde con las tareas
- **THEN** la TUI vuelve a consultar las tareas de la historia
- **AND** la fila «Toda la historia (sin tarea)» sigue siendo la primera fila

#### Scenario: Si falla la consulta de tareas no se ofrece la fila sintética
- **GIVEN** que la API responde HTTP 500 al consultar las tareas de la historia elegida
- **WHEN** la TUI presenta el selector de tareas
- **THEN** muestra el error de la API dentro de la pantalla sin cerrarse
- **AND** no ofrece la fila «Toda la historia (sin tarea)» ni abre el selector de integrantes
- **AND** `r` vuelve a consultar y, si la API responde, la fila aparece
- **AND** `Esc` vuelve al detalle de la historia

#### Scenario: Selección simple de integrante
- **GIVEN** el selector de integrantes de Esfuerzo con dos integrantes
- **WHEN** el usuario presiona `Space` y luego `Enter` sobre el segundo integrante
- **THEN** `Space` no marca ninguna fila
- **AND** `Enter` abre el formulario de esfuerzo para el segundo integrante
- **AND** el pie y la ayuda del paso no ofrecen multiselección ni confirmación de participantes

#### Scenario: Proyecto sin integrantes
- **GIVEN** que la API responde HTTP 200 con una lista de integrantes vacía
- **WHEN** la TUI presenta el selector de integrantes de Esfuerzo
- **THEN** muestra un mensaje que indica que el proyecto no tiene integrantes registrados
- **AND** el mensaje indica que los integrantes se dan de alta mediante la API y no remite a otra pantalla
- **AND** `Esc` vuelve al selector de tareas y `r` vuelve a consultar los integrantes

#### Scenario: Falla la consulta de integrantes
- **GIVEN** que la API responde HTTP 500 al consultar los integrantes
- **WHEN** la TUI presenta el selector de integrantes
- **THEN** muestra el error de la API dentro de la pantalla sin cerrarse
- **AND** no abre el formulario de esfuerzo
- **AND** permite reintentar con `r` o volver con `Esc`

### Requirement: Enviar el registro de esfuerzo y presentar los errores de la API
Al confirmar con datos válidos, la TUI DEBERÁ emitir exactamente una request `POST /projects/{pid}/stories/{sid}/effort-records`, con el proyecto del espacio de trabajo y la historia elegida, y un cuerpo JSON cuyas claves son `member_id`, `work_date` (normalizada como `YYYY-MM-DD`), `description` (tal como fue escrita, sin recortar ni validar), `hours` (número JSON) y `task_id` únicamente cuando se eligió una tarea; al elegir «Toda la historia (sin tarea)» la clave `task_id` DEBERÁ omitirse por completo, no enviarse como `null`. Mientras el `POST` está en curso la TUI DEBERÁ indicar que está guardando e ignorar `Enter` y `Esc`, de modo que no se emita una segunda request ni se abandone el formulario antes de conocer el resultado. Solo la respuesta HTTP 201 comunica éxito: la TUI vuelve al detalle de la historia desde la que se inició el registro y muestra un aviso con las horas y la fecha devueltas por la API (o las enviadas si el cuerpo no se puede interpretar), la historia, la tarea («toda la historia» si no se eligió una) y el integrante (Requirement: Avisos efímeros en lugar de pantallas de resultado), sin consultar las métricas del proyecto ni emitir otra request. Ante cualquier otra respuesta de error o ante un error de transporte, la TUI DEBERÁ permanecer en el formulario con todos los valores escritos, dejar de indicar el guardado, mostrar el motivo y NO comunicar éxito. Los errores de la API (HTTP 400, 404, 422 con `fields`, 500 y cualquier otro código no exitoso, incluido 409 si alguna vez se devolviera) DEBERÁN presentarse con las reglas del Requirement: Presentar los errores de la API del backlog. Un error de transporte DEBERÁ informar el problema de conexión y que no se pudo confirmar el registro, y aconsejar verificar antes de reintentar, porque la API no es idempotente. Tras un error el usuario puede corregir y confirmar de nuevo, lo que emite una nueva request.
(Previously: el HTTP 201 volvía al selector de historias de la pestaña Esfuerzo, lo recargaba y mostraba el aviso.)

#### Scenario: Registrar esfuerzo sobre una tarea
- **GIVEN** el formulario del integrante «Ana Pérez» (`member_id` `m-1`) para la tarea `t-1` de la historia `s-1` del proyecto `p-1`, con fecha 2026-10-05, «Actividad» «Implementar login» y «Horas» `3.5`
- **WHEN** el usuario confirma desde «Horas»
- **THEN** la TUI emite una única request `POST /projects/p-1/stories/s-1/effort-records`
- **AND** el cuerpo contiene exactamente `member_id` `m-1`, `task_id` `t-1`, `work_date` `2026-10-05`, `description` `Implementar login` y `hours` igual al número 3.5

#### Scenario: Registrar esfuerzo sobre la historia completa omite `task_id`
- **GIVEN** que el usuario eligió la fila «Toda la historia (sin tarea)» y completó el formulario con datos válidos
- **WHEN** confirma
- **THEN** el cuerpo enviado contiene `member_id`, `work_date`, `description` y `hours`
- **AND** el cuerpo no contiene la clave `task_id`

#### Scenario: La actividad se envía sin recortar
- **GIVEN** el formulario con «Actividad» igual a `  Implementar login  ` y datos válidos
- **WHEN** el usuario confirma
- **THEN** el cuerpo enviado contiene `description` igual a `  Implementar login  `

#### Scenario: El doble `Enter` emite una sola request
- **GIVEN** el formulario con datos válidos y el foco en «Horas»
- **WHEN** el usuario presiona `Enter` dos veces antes de que llegue la respuesta del `POST`
- **THEN** la TUI emite exactamente una request de registro
- **AND** mientras espera muestra un indicador de guardado

#### Scenario: `Esc` se ignora mientras se guarda
- **GIVEN** un `POST` de esfuerzo en curso
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI permanece en el formulario con el indicador de guardado
- **AND** al llegar la respuesta se aplica su resultado

#### Scenario: HTTP 422 con campos inválidos
- **GIVEN** que la API responde HTTP 422 con `fields` que detalla `hours` y `description`
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra cada campo y su motivo como un elemento de lista
- **AND** conserva fecha, «Actividad» y «Horas» tal como se escribieron
- **AND** no comunica éxito ni deja el guardado activo

#### Scenario: HTTP 400 por solicitud inválida
- **GIVEN** que la API responde HTTP 400 con `invalid_request` y un mensaje
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra el código HTTP y el detalle de la API
- **AND** conserva los valores escritos y no comunica éxito

#### Scenario: HTTP 404 por recurso inexistente
- **GIVEN** que la API responde HTTP 404 con `member_not_found` (o `task_not_found`, `story_not_found`, `project_not_found`)
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra un mensaje genérico de recurso inexistente y el detalle informado por la API
- **AND** conserva los valores escritos y no comunica éxito

#### Scenario: HTTP 500
- **GIVEN** que la API responde HTTP 500 con `internal_error`
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra el código HTTP y el detalle de la API
- **AND** conserva los valores escritos y no comunica éxito

#### Scenario: Otro código no exitoso se presenta de forma genérica
- **GIVEN** que la API responde HTTP 409 con un cuerpo JSON de error
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra el código 409 y el detalle de la API, sin comunicar éxito

#### Scenario: Error de conexión
- **GIVEN** que no se puede establecer conexión con la API o que la request excede el tiempo de espera
- **WHEN** falla el `POST` de esfuerzo
- **THEN** el formulario informa el problema de conexión
- **AND** indica que no se pudo confirmar el registro y aconseja verificar antes de reintentar
- **AND** conserva los valores escritos y la TUI sigue abierta

#### Scenario: Corregir y reenviar tras un rechazo
- **GIVEN** el formulario mostrando el error de un HTTP 422 con los valores conservados
- **WHEN** el usuario corrige «Horas» y confirma de nuevo
- **THEN** la TUI emite una nueva request de registro con los valores corregidos
- **AND** si la API responde HTTP 201 vuelve al detalle de la historia con el aviso de éxito

### Requirement: Retroceder en el flujo de esfuerzo y descartar respuestas tardías
`Esc` DEBERÁ retroceder un nivel por vez en el flujo de Esfuerzo iniciado desde el detalle de una historia: del formulario al selector de integrantes, del selector de integrantes al de tareas y del selector de tareas, primer paso del flujo, al detalle de la historia. Cada retroceso DEBERÁ restaurar el paso anterior tal como estaba al elegir su fila (filas, cursor y filtro), sin emitir requests; en un selector con filtro aplicado, el primer `Esc` limpia el filtro y el siguiente retrocede. Salir del formulario con `Esc` descarta lo escrito. Tras un registro exitoso la TUI vuelve al detalle de la historia y el siguiente registro empieza con el formulario inicial. Cambiar de proyecto desde el navegador DEBERÁ descartar el estado del flujo (historia, tarea, integrante y formulario), el detalle y su destino de retorno, y abrir el espacio de trabajo del nuevo proyecto en el Dashboard. La TUI DEBERÁ descartar sin efecto toda respuesta que ya no corresponda a la pantalla, el paso o la consulta activos: una respuesta de tareas o de integrantes tras haber retrocedido y un resultado de guardado sin un guardado en curso. Estas guardas se conservan sin cambios respecto del flujo anterior.
(Previously: el primer paso era el selector de historias de la pestaña Esfuerzo, el último `Esc` volvía al navegador de proyectos, y un registro exitoso volvía a ese selector; con la historia preelegida mediante `E`, el `Esc` del primer paso volvía a la lista del backlog.)

#### Scenario: `Esc` recorre formulario, integrantes y tareas hasta el detalle
- **GIVEN** el formulario de esfuerzo abierto desde el detalle de «Login» a través de los selectores de tarea e integrante
- **WHEN** el usuario presiona `Esc` tres veces
- **THEN** la TUI pasa sucesivamente al selector de integrantes, al selector de tareas y al detalle de «Login»
- **AND** no emite ninguna request en esos retrocesos
- **AND** el detalle sigue mostrando «Login» sin recargar el backlog

#### Scenario: El filtro del selector de tareas se conserva
- **GIVEN** que el usuario filtró las tareas con `dis` y eligió «Diseñar»
- **WHEN** retrocede desde el formulario al selector de integrantes y de allí al selector de tareas
- **THEN** el selector de tareas muestra el filtro `dis` y el cursor sobre «Diseñar», sin consultar de nuevo
- **AND** el siguiente `Esc` limpia el filtro y recién el que sigue vuelve al detalle de la historia

#### Scenario: Un segundo registro empieza con el formulario inicial
- **GIVEN** que la TUI volvió al detalle tras un registro exitoso
- **WHEN** el usuario elige otra vez «Registrar esfuerzo», otra fila de tareas y un integrante
- **THEN** el formulario se abre con la fecha de hoy y «Actividad» y «Horas» vacías

#### Scenario: Respuesta tardía de integrantes tras retroceder
- **GIVEN** que el usuario eligió una tarea y la consulta de integrantes sigue en curso
- **WHEN** presiona `Esc` y después llega la respuesta de integrantes
- **THEN** la TUI sigue mostrando el selector de tareas sin cambios
- **AND** la respuesta se descarta

#### Scenario: Respuesta tardía de tareas tras volver al detalle
- **GIVEN** que el usuario eligió «Registrar esfuerzo» y la consulta de tareas sigue en curso
- **WHEN** presiona `Esc` y después llega la respuesta de tareas
- **THEN** la TUI sigue mostrando el detalle de la historia sin cambios
- **AND** la respuesta se descarta

#### Scenario: Respuesta tardía de tareas de otra historia
- **GIVEN** que el usuario abrió «Registrar esfuerzo» para la historia A desde su detalle y la consulta de tareas de A sigue en curso
- **WHEN** presiona `Esc` para volver al detalle, abre «Registrar esfuerzo» para la historia B y después llega la lista de tareas de A
- **THEN** la lista de tareas de A se descarta y no puebla el selector de tareas de B
- **AND** el selector de B sigue mostrando su propio estado de carga o las tareas de B

#### Scenario: Resultado de guardado sin guardado en curso
- **GIVEN** que no hay un `POST` de esfuerzo en curso
- **WHEN** llega un resultado de guardado
- **THEN** la TUI lo descarta sin cambiar de pantalla ni de contenido

#### Scenario: Cambiar el proyecto descarta el flujo, el detalle y el destino
- **GIVEN** que el usuario registró un esfuerzo en el proyecto `p-1` y está en el detalle de «Login»
- **WHEN** vuelve al navegador, elige el proyecto `p-2` y abre su pestaña Backlog
- **THEN** la TUI abre el espacio de trabajo de `p-2` en el Dashboard y muestra la lista del backlog de `p-2`, no un detalle
- **AND** la pantalla no conserva el aviso, el registro ni las selecciones de `p-1`

### Requirement: Espacio de trabajo del proyecto con pestañas
Elegir un proyecto en el navegador con `Enter` DEBERÁ fijarlo como proyecto de la sesión y abrir su espacio de trabajo. El espacio de trabajo DEBERÁ mostrar, debajo de la barra superior, una línea con el nombre y el estado del proyecto y una barra de pestañas con, en este orden: Dashboard, Backlog, Sprints y Defectos, resaltando la pestaña activa. No existen las pestañas Estimación ni Esfuerzo: Planning Poker y el registro de esfuerzo se inician desde el detalle de una historia (Requirement: Detalle de la historia con sus acciones). En la pantalla raíz de una pestaña, y sin un filtro en edición, `Tab` y `Shift+Tab` DEBERÁN pasar a la pestaña siguiente y anterior (en forma circular), y las teclas `1` a `4` DEBERÁN abrir la pestaña de esa posición; el detalle de la historia y los pasos de los flujos de acción no son pantallas raíz y no responden a estas teclas. `Esc` en la pantalla raíz de una pestaña, sin filtro aplicado, DEBERÁ volver al navegador de proyectos. Todas las pestañas DEBERÁN operar sobre el proyecto del espacio de trabajo sin volver a pedirlo. La miga de pan DEBERÁ empezar con «Proyectos», seguir con el nombre del proyecto y la pestaña activa, y terminar en el paso actual del flujo.

La pestaña activa se determina por el origen de la pantalla. Supuesto S1 de la propuesta, no consultado con el usuario y válido por defecto: la barra DEBERÁ resaltar Backlog en el detalle de la historia y en los flujos de acción que parten de él (selector de participantes, Planning Poker, selector de tareas, selectores de integrantes, formulario de esfuerzo y alta de defecto iniciada desde la historia), y Defectos solo en la lista de defectos, su detalle, el cambio de estado y el alta iniciada desde la pestaña Defectos.
(Previously: la barra tenía seis pestañas (Dashboard, Backlog, Sprints, Estimación, Esfuerzo y Defectos), los dígitos eran `1` a `6`, y las pantallas `defect-*` resaltaban siempre Defectos.)

#### Scenario: Elegir un proyecto abre el Dashboard
- **GIVEN** que el navegador muestra varios proyectos
- **WHEN** el usuario resalta uno y presiona `Enter`
- **THEN** la TUI abre el espacio de trabajo de ese proyecto en la pestaña Dashboard
- **AND** la barra de pestañas muestra Dashboard, Backlog, Sprints y Defectos
- **AND** la barra no muestra Estimación ni Esfuerzo

#### Scenario: Cambiar de pestaña con Tab y con números
- **GIVEN** que la pestaña Dashboard está activa
- **WHEN** el usuario presiona `Tab`
- **THEN** se abre la pestaña Backlog y se carga su lista
- **AND** al presionar `4` se abre la pestaña Defectos
- **AND** al presionar `Shift+Tab` se abre la pestaña Sprints

#### Scenario: Tab desde la última pestaña
- **GIVEN** que la pestaña Defectos está activa
- **WHEN** el usuario presiona `Tab`
- **THEN** se abre la pestaña Dashboard

#### Scenario: Shift+Tab desde la primera pestaña
- **GIVEN** que la pestaña Dashboard está activa
- **WHEN** el usuario presiona `Shift+Tab`
- **THEN** se abre la pestaña Defectos

#### Scenario: Los dígitos 5 y 6 ya no cambian de pestaña
- **GIVEN** que la pestaña Dashboard está activa
- **WHEN** el usuario presiona `5` o `6`
- **THEN** la pestaña activa no cambia y no se emite ninguna request

#### Scenario: Los números se escriben dentro de un filtro
- **GIVEN** que el usuario está escribiendo un filtro en la lista del backlog
- **WHEN** presiona `3`
- **THEN** el carácter se agrega al filtro y la pestaña no cambia

#### Scenario: Esc en la raíz vuelve a Proyectos
- **GIVEN** que una pestaña muestra su pantalla raíz sin filtro aplicado
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI vuelve al navegador de proyectos y recarga el catálogo

#### Scenario: Esc dentro de un flujo retrocede un paso
- **GIVEN** que el usuario abrió el formulario de una historia desde la pestaña Backlog
- **WHEN** presiona `Esc`
- **THEN** la TUI vuelve a la lista del backlog sin salir del espacio de trabajo

#### Scenario: El detalle y los flujos de acción resaltan Backlog
- **GIVEN** el detalle de una historia, y a partir de él cada una de las acciones Estimar, Registrar esfuerzo y Reportar defecto
- **WHEN** la TUI dibuja la barra de pestañas y la miga de pan en cada pantalla de esos flujos
- **THEN** la pestaña resaltada es siempre Backlog
- **AND** la miga empieza con «Proyectos», el nombre del proyecto y Backlog, y termina en el paso actual

#### Scenario: Defectos resalta su pestaña solo en su propio flujo
- **GIVEN** que el usuario abre la pestaña Defectos
- **WHEN** recorre la lista, el detalle de un defecto y el alta iniciada con `n`
- **THEN** la pestaña resaltada es Defectos en todas esas pantallas

#### Scenario: Cambiar de proyecto
- **GIVEN** que el usuario trabaja en el proyecto A
- **WHEN** vuelve al navegador con `Esc` y elige el proyecto B
- **THEN** el espacio de trabajo muestra el proyecto B y todas las pestañas usan B

### Requirement: Avisos efímeros en lugar de pantallas de resultado
Cuando una operación de escritura termina con éxito (crear o modificar un proyecto, crear o modificar una historia, crear un Sprint, asignar historias, completar una historia, crear una tarea, cerrar un Sprint, registrar esfuerzo, registrar un defecto o cambiar su estado), la TUI DEBERÁ volver a la pantalla desde la que se inició el flujo, recargar su colección cuando esa pantalla muestra una y mostrar un aviso de éxito con el prefijo `✓` al inicio del panel. Los destinos son: el navegador de proyectos (recargando `GET /projects`, sin abrir automáticamente un proyecto único) para los proyectos; la lista del backlog para las historias; la lista de Sprints abiertos para crear un Sprint, asignar historias y cerrar un Sprint; las historias del Sprint para completar una historia y crear una tarea; el detalle de la historia desde la que se inició el flujo para el registro de esfuerzo, con un aviso que resume horas, fecha, historia, tarea e integrante, y para el registro de un defecto iniciado desde la historia; la lista de defectos (recargada) para registrar un defecto iniciado desde la pestaña Defectos; y el detalle del defecto, actualizado con la respuesta de la API, para el cambio de estado. El guardado del Story Points de Planning Poker no emite aviso, pero vuelve igualmente al detalle de la historia con el backlog recargado (Requirement: Estimar historias con Planning Poker). El aviso DEBERÁ desaparecer con la siguiente tecla, que conserva su efecto normal; no se usan temporizadores. Mientras una escritura está en curso, la pantalla DEBERÁ mostrar un indicador de envío e ignorar las repeticiones del envío. Cuando la API rechaza la operación, la TUI DEBERÁ permanecer en el formulario o la pantalla de confirmación, conservar los valores escritos y mostrar el motivo informado por la API. Una respuesta de escritura que llega sin un envío en curso DEBERÁ descartarse sin cambiar de pantalla. La TUI MUST NOT mostrar pantallas de resultado que exijan una tecla adicional para continuar.
(Previously: el registro de esfuerzo volvía al selector de historias de la pestaña Esfuerzo y todo registro de defecto volvía a la lista de defectos.)

#### Scenario: Crear una historia vuelve al backlog con un aviso
- **GIVEN** que el formulario de una historia nueva está completo
- **WHEN** la API responde la creación con éxito
- **THEN** la TUI vuelve a la lista del backlog, la recarga y muestra un aviso que empieza con `✓`
- **AND** el aviso desaparece con la siguiente tecla

#### Scenario: Un rechazo conserva el formulario
- **GIVEN** que el formulario de un proyecto tiene valores escritos
- **WHEN** la API rechaza la operación con un error
- **THEN** la TUI permanece en el formulario con los valores escritos
- **AND** muestra el motivo informado por la API

#### Scenario: Registrar esfuerzo vuelve al detalle de la historia
- **GIVEN** que el formulario de esfuerzo de «Ada Lovelace» sobre la tarea «Diseñar» de la historia «Login» está completo, iniciado desde el detalle de «Login»
- **WHEN** la API responde HTTP 201 con `hours` 3.5 y `work_date` 2026-10-05
- **THEN** la TUI vuelve al detalle de «Login»
- **AND** muestra el aviso «✓ Esfuerzo registrado: 3.5 h el 2026-10-05 · Login · Diseñar · Ada Lovelace»
- **AND** no consulta las métricas del proyecto

#### Scenario: Registrar un defecto desde la historia vuelve al detalle
- **GIVEN** la confirmación de un defecto con descripción «No guarda» iniciado desde el detalle de «Login»
- **WHEN** la API responde con éxito
- **THEN** la TUI vuelve al detalle de «Login» con el aviso «✓ Defecto registrado: No guarda»
- **AND** el aviso desaparece con la siguiente tecla, que conserva su efecto normal

#### Scenario: Registrar un defecto desde la pestaña Defectos vuelve a la lista
- **GIVEN** la confirmación de un defecto iniciado con `n` desde la pestaña Defectos
- **WHEN** la API responde con éxito
- **THEN** la TUI vuelve a la lista de defectos recargada con un aviso que empieza con `✓`

#### Scenario: Estimar vuelve al detalle sin aviso
- **GIVEN** una estimación confirmada desde el detalle de «Login»
- **WHEN** la API responde con éxito
- **THEN** la TUI vuelve al detalle de «Login» y recarga el backlog
- **AND** el detalle muestra el Story Points guardado

#### Scenario: Cambiar el estado de un defecto vuelve a su detalle
- **GIVEN** que el usuario eligió un nuevo estado para un defecto
- **WHEN** la API responde con éxito
- **THEN** la TUI muestra el detalle de ese defecto con el estado devuelto por la API y un aviso que empieza con `✓`

#### Scenario: El envío en curso ignora repeticiones
- **GIVEN** que el usuario confirmó la creación de un Sprint
- **WHEN** presiona `Enter` de nuevo antes de que responda la API
- **THEN** la TUI no emite otra request y muestra «Guardando…»

### Requirement: Pie de teclas acotado y ayuda agrupada
El pie DEBERÁ mostrar como máximo cinco atajos, los principales de la pantalla y el paso activos, y agregar `? más` cuando la pantalla tenga más atajos disponibles y `?` abra la ayuda en ese paso. La ayuda abierta con `?` DEBERÁ listar todos los atajos válidos de la pantalla activa agrupados en «Navegación», «Acciones» y «General», omitiendo los grupos vacíos. Las teclas de cambio de pestaña DEBERÁN figurar en la ayuda de las pantallas raíz del espacio de trabajo, con los dígitos `1` a `4`, y no en las demás pantallas.
(Previously: la ayuda de las pantallas raíz listaba los dígitos `1–6`.)

#### Scenario: Pie de la lista de Sprints
- **GIVEN** que la pestaña Sprints muestra su lista
- **WHEN** la TUI dibuja el pie
- **THEN** muestra como máximo cinco atajos y `? más`

#### Scenario: La ayuda lista todos los atajos agrupados
- **GIVEN** que la pestaña Sprints muestra su lista
- **WHEN** el usuario presiona `?`
- **THEN** la ayuda muestra los grupos «Navegación», «Acciones» y «General»
- **AND** incluye los atajos de cerrar Sprint, ver los cerrados y descargar el PDF

#### Scenario: Pantalla con pocos atajos
- **GIVEN** que el formulario del objetivo del Sprint está abierto
- **WHEN** la TUI dibuja el pie
- **THEN** muestra todos sus atajos y no agrega `? más`

#### Scenario: Filtro o campo de texto activo
- **GIVEN** que el usuario escribe en un filtro o en un campo de texto
- **WHEN** la TUI dibuja el pie
- **THEN** el pie no ofrece `?`, porque ahí `?` es un carácter

#### Scenario: Teclas de pestaña en la ayuda
- **GIVEN** que la pantalla activa es la raíz de una pestaña del espacio de trabajo
- **WHEN** el usuario presiona `?`
- **THEN** la ayuda lista `Tab/Shift+Tab` y `1–4` en «Navegación»

#### Scenario: La ayuda del Backlog refleja sus teclas
- **GIVEN** que la lista del backlog tiene una historia resaltada
- **WHEN** el usuario presiona `?`
- **THEN** la ayuda lista `Enter` para abrir el detalle y `e` para modificar
- **AND** no lista `P` ni `E`

### Requirement: Salir limpiamente desde el navegador y las pestañas
El usuario DEBERÁ poder salir limpiamente con `Ctrl+C` desde cualquier pantalla y con `q` desde el navegador de proyectos, las listas, los selectores, las confirmaciones, el detalle de la historia y las pantallas raíz de las pestañas. Dentro de un campo de entrada de texto, `q` (y `Q`) DEBERÁ tratarse como un carácter más y NO cerrará la TUI.
(Previously: la enumeración de pantallas desde las que `q` sale no incluía el detalle de la historia.)

#### Scenario: Salida limpia
- **WHEN** el usuario pide salir
- **THEN** la TUI termina con código 0 sin traza de error

#### Scenario: `q` sale desde el navegador de proyectos
- **GIVEN** que el navegador de proyectos está activo
- **WHEN** el usuario presiona `q`
- **THEN** la TUI se cierra con código 0

#### Scenario: `q` sale desde una lista
- **GIVEN** que una lista seleccionable está activa (por ejemplo, el backlog)
- **WHEN** el usuario presiona `q`
- **THEN** la TUI se cierra con código 0

#### Scenario: `q` sale desde el detalle de la historia
- **GIVEN** que el detalle de una historia está activo
- **WHEN** el usuario presiona `q`
- **THEN** la TUI se cierra con código 0

#### Scenario: `q` en un campo de texto no sale
- **GIVEN** que el foco está en un campo de texto de un formulario o en el buscador
- **WHEN** el usuario presiona `q` o `Q`
- **THEN** la TUI permanece abierta
- **AND** el carácter se agrega al valor del campo

#### Scenario: `Ctrl+C` sale desde cualquier pantalla
- **GIVEN** que cualquier pantalla está activa, incluidos los formularios y los buscadores
- **WHEN** el usuario presiona `Ctrl+C`
- **THEN** la TUI se cierra con código 0 sin traza de error

## REMOVED Requirements

### Requirement: Atajos contextuales de estimación y esfuerzo
(Reason: los atajos `P` y `E` de la lista del backlog dependían de las pestañas Estimación y Esfuerzo y de la historia «preelegida»; ambas pestañas y ese camino se retiran porque duplicaban la elección de historia que ya ofrece el Backlog.)
(Migration: desde la lista del backlog se presiona `Enter` sobre la historia para abrir su detalle y desde allí se elige la acción «Estimar» o «Registrar esfuerzo» con `Enter` sobre su fila o con su atajo (`p` y `t`); el contenido vigente se reemplaza por Requirement: Detalle de la historia con sus acciones. `P` y `E` mayúsculas dejan de tener efecto en la lista, y `Esc` desde el primer selector de esos flujos vuelve al detalle de la historia, no a la lista recargada.)
