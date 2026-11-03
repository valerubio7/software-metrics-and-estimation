# Delta para tui: limpiar las referencias al menú eliminado (TUI-13)

TUI-12 reemplazó el menú de nueve opciones por el navegador de proyectos y el espacio de trabajo con pestañas, pero cinco requisitos que no estaban en su delta siguen describiendo el menú, las pantallas de resultado o la tecla `c`. Este cambio los reescribe según el comportamiento implementado y elimina el requisito de los avisos de TUI-06, cuyo paso ya no es alcanzable. No cambia ningún contrato de la API.

## MODIFIED Requirements

### Requirement: Navegación de solo lectura
Navegar entre el navegador de proyectos, las pestañas y sus pantallas DEBERÁ ser de solo lectura. Solo las operaciones explícitas de escritura emiten requests que modifican datos: proyectos, historias, Sprints, el `PUT` del Story Points de Estimación, el `POST` de esfuerzo y las operaciones de defectos, cada una definida en su propio requisito. Las demás pantallas y selectores DEBERÁN emitir únicamente consultas `GET`.
(Previously: la navegación partía del menú, enumeraba las secciones Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard e incluía sus pantallas de aviso.)

#### Scenario: Navegar no muta
- **WHEN** el usuario navega entre el navegador de proyectos, las pestañas y sus pantallas sin confirmar una operación explícita de escritura
- **THEN** no se emite ninguna request que modifique datos

#### Scenario: Consultar el backlog no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario abre la lista del backlog o cambia de proyecto
- **THEN** las únicas requests emitidas son las consultas `GET` (del catálogo de proyectos y del backlog) y no se emite ningún `POST` ni `PUT`

#### Scenario: Navegar por las secciones del Sprint 3 no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario recorre las pestañas Dashboard, Estimación, Esfuerzo y Defectos, elige filas y avanza entre selectores, sin confirmar el guardado de Story Points ni el envío del formulario de esfuerzo
- **THEN** todas las requests emitidas son `GET`
- **AND** no se crea, modifica ni elimina ningún dato

#### Scenario: Abrir y completar el formulario de esfuerzo no muta hasta confirmar
- **GIVEN** que el usuario llegó al formulario de esfuerzo desde los selectores de historia, tarea e integrante
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

### Requirement: Operar proyectos mediante la TUI
La TUI DEBERÁ operar proyectos desde el navegador de proyectos, que es la pantalla inicial (Requirement: Listar y elegir proyectos con un buscador): abrir el proyecto resaltado, crear uno nuevo con `n` y modificar el resaltado con `e`, sin pedir identificadores a mano. La gestión consume la API existente y no incorpora lógica de negocio. El éxito vuelve al navegador con un aviso `✓` (Requirement: Precargar la modificación de proyecto y resolver el resultado).
(Previously: los proyectos se operaban desde la opción «Proyectos» del menú, que se conservaba, y el resultado de la API se presentaba en una pantalla de resultado.)

#### Scenario: La opción Proyectos abre el navegador
- **GIVEN** que la API devuelve dos o más proyectos
- **WHEN** la TUI arranca o el usuario vuelve con `Esc` desde la raíz de una pestaña
- **THEN** la TUI emite `GET /projects` y presenta el navegador buscable
- **AND** no pide ningún identificador a mano

#### Scenario: Crear un proyecto

- **GIVEN** que el usuario abre la creación desde el navegador (tecla `n`) y completa nombre, fecha de inicio y fecha planificada de finalización
- **WHEN** confirma la creación
- **THEN** la TUI envía esos campos al endpoint HTTP de creación
- **AND** ante el éxito vuelve al navegador recargado con un aviso que empieza con `✓`

#### Scenario: La API rechaza datos de creación

- **GIVEN** que la API responde HTTP 422 con detalles de validación
- **WHEN** la TUI recibe la respuesta
- **THEN** permanece en el formulario, muestra el motivo informado por la API y no comunica éxito

#### Scenario: Modificar un proyecto existente

- **GIVEN** que el usuario resalta un proyecto en el navegador y abre su modificación con `e`
- **WHEN** confirma la modificación
- **THEN** la TUI envía los campos al endpoint HTTP de actualización sin duplicar validaciones de negocio

#### Scenario: Proyecto inexistente al modificar

- **GIVEN** que la API responde HTTP 404 al modificar
- **WHEN** la TUI recibe la respuesta
- **THEN** la TUI permanece en el formulario, informa que el proyecto no existe y presenta el detalle de la API

#### Scenario: El estado derivado se muestra en el navegador

- **GIVEN** que el catálogo informa el estado de cada proyecto (`planned`, `active` o `overdue`)
- **WHEN** la TUI presenta el navegador
- **THEN** cada fila muestra ese estado con una etiqueta en español (`planificado`, `en curso` o `atrasado`) sin recalcularlo

#### Scenario: Error de transporte

- **GIVEN** que no se puede establecer conexión con la API
- **WHEN** falla una operación de proyectos
- **THEN** la TUI informa el problema de conexión sin cerrarse abruptamente

### Requirement: Operar el backlog mediante la TUI
La TUI DEBERÁ permitir consultar, crear y modificar historias del proyecto del espacio de trabajo desde la pestaña Backlog, con la API existente (`GET`, `POST` y `PUT` de `/projects/{pid}/stories`) y sin reglas de negocio. La pantalla conserva el nombre `backlog` y se comporta según el patrón navegable (Requirement: Listas navegables con teclas uniformes), sin filas de acción: `n` crea, `Enter` modifica y `r` refresca. Cambiar de proyecto se hace volviendo a Proyectos con `Esc`.
(Previously: el backlog se abría desde la opción «Backlog» del menú, pedía un proyecto si no había uno activo y la tecla `c` cambiaba de proyecto.)

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

#### Scenario: Crear y modificar historias desde el navegador
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `n`
- **THEN** la TUI abre el formulario de creación vacío
- **AND** si resalta una historia y presiona `Enter` abre el formulario de modificación precargado

#### Scenario: Cambiar de proyecto desde el navegador
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `Esc` sin filtro aplicado
- **THEN** la TUI vuelve al navegador de proyectos, donde puede abrir otro proyecto
- **AND** `c` no cambia de proyecto

### Requirement: Elegir historia, tarea e integrante para registrar esfuerzo
La sección Esfuerzo DEBERÁ guiar el registro con los selectores navegables existentes, en este orden: historia del backlog del proyecto activo, tarea de esa historia (o la historia completa) e integrante del proyecto activo. Una vez cargadas las tareas de la historia elegida, el selector de tareas DEBERÁ anteponer, como primera fila, la fila sintética «Toda la historia (sin tarea)», seguida de las tareas en el orden informado por la API; la fila sintética DEBERÁ estar presente también cuando la historia no tiene tareas. Elegir una tarea registra el esfuerzo sobre esa tarea; elegir la fila sintética lo registra sobre la historia completa. Elegir cualquiera de las dos filas DEBERÁ abrir el selector de integrantes del proyecto activo, que en Esfuerzo es de selección simple: `Enter` elige el integrante resaltado y abre el formulario de esfuerzo (Requirement: Completar el formulario de esfuerzo), y `Space` no marca integrantes. Ningún paso DEBERÁ pedir un identificador escrito a mano. El avance entre selectores y la apertura del formulario NO DEBERÁN emitir requests que modifiquen datos. Si el proyecto no tiene integrantes, el selector de integrantes DEBERÁ mostrar un estado vacío propio que informe que el proyecto no tiene integrantes registrados, que indique que los integrantes se dan de alta mediante la API y que permita volver con `Esc` y refrescar con `r`.
(Previously: el estado vacío de integrantes se definía como un mensaje que NO remitía a ninguna opción del menú principal.)

#### Scenario: Flujo completo hasta el formulario
- **GIVEN** un proyecto activo con la historia «Login», que tiene la tarea «Diseñar», y el integrante «Ana Pérez»
- **WHEN** el usuario abre Esfuerzo, elige «Login», elige «Diseñar» y elige «Ana Pérez» con `Enter`
- **THEN** la TUI abre el formulario de esfuerzo mostrando la historia «Login», la tarea «Diseñar» y el integrante «Ana Pérez»
- **AND** todas las requests emitidas hasta ese punto son `GET` (historias, tareas e integrantes)
- **AND** la TUI no pidió ningún identificador

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

### Requirement: Iniciar en el navegador de proyectos
La TUI MUST arrancar en el navegador de proyectos y emitir `GET /projects` al iniciar, sin una pantalla intermedia. Si la primera carga devuelve exactamente un proyecto, la TUI DEBERÁ abrir directamente el espacio de trabajo de ese proyecto en la pestaña Dashboard. Con cero o más de un proyecto, la TUI DEBERÁ quedarse en el navegador. La apertura automática ocurre solo en la primera carga de la sesión.
(Previously: el requisito decía «sin un menú intermedio»; solo cambia la redacción.)

#### Scenario: Arranque con varios proyectos
- **GIVEN** que la API devuelve dos o más proyectos
- **WHEN** la TUI arranca
- **THEN** muestra el navegador de proyectos con el título «Proyectos»
- **AND** no muestra una lista de secciones previa a los proyectos

#### Scenario: Arranque con un único proyecto
- **GIVEN** que la API devuelve exactamente un proyecto
- **WHEN** la TUI termina la primera carga
- **THEN** abre el espacio de trabajo de ese proyecto en la pestaña Dashboard

#### Scenario: Volver al navegador no reabre el proyecto único
- **GIVEN** que la TUI abrió automáticamente el único proyecto
- **WHEN** el usuario vuelve al navegador con `Esc` y la lista se recarga
- **THEN** la TUI permanece en el navegador

## REMOVED Requirements

### Requirement: Avisar el límite de las pantallas de navegación
**Reason**: todas las pestañas tienen su propio flujo y el paso de aviso de TUI-06 quedó inalcanzable; se elimina del código (`sectionStepNotice`, `sectionNotice`, `viewSectionNotice`).
**Migration**: Estimación (TUI-07): Requirement: Estimar historias con Planning Poker; Esfuerzo (TUI-08): Requirement: Elegir historia, tarea e integrante para registrar esfuerzo; Defectos (TUI-09): su propia pestaña con la lista de defectos; Dashboard (TUI-11), que integra las métricas: Requirement: Dashboard con las métricas del proyecto y de un Sprint.
