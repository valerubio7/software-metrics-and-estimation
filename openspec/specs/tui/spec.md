# tui Specification

## Purpose
Permitir operar el sistema desde consola sin curl mediante una TUI base navegable que solo consume la API HTTP.

## Requirements

### Requirement: Abrir la TUI con configuración de conexión a la API

El sistema DEBERÁ permitir abrir la TUI configurando la conexión vía `API_URL` con default `http://localhost:8080`.

#### Scenario: Apertura con API_URL explícita

- **WHEN** el usuario abre la TUI con `API_URL` válida
- **THEN** la TUI arranca y usa esa URL como base de la API

#### Scenario: Apertura sin API_URL

- **WHEN** el usuario abre la TUI sin definir `API_URL`
- **THEN** la TUI usa `http://localhost:8080` por defecto

### Requirement: Presentar el contenido en un panel con jerarquía visual
La TUI DEBERÁ mostrar el contenido de cada pantalla dentro de un panel con borde que ocupa el ancho disponible, encabezado por una barra superior sólida con la marca y el estado de conexión a la API, seguida de la miga de pan (breadcrumb). La fila enfocada DEBERÁ resaltarse como una barra que ocupa todo el ancho del panel, no solo el texto. Cuando el ancho no alcanza para la marca, la barra superior DEBERÁ priorizar el indicador de conexión y mantener el contenido dentro del ancho disponible sin desbordar.

#### Scenario: Selección como barra completa
- **WHEN** una lista o formulario muestra una fila enfocada
- **THEN** la fila enfocada se resalta con una barra que ocupa todo el ancho del panel

#### Scenario: Ancho reducido
- **WHEN** la terminal es más angosta que la marca más el indicador de conexión
- **THEN** la barra superior conserva el indicador de conexión y el resto del contenido se mantiene dentro del ancho disponible

### Requirement: Mostrar la ayuda de teclado
La TUI DEBERÁ ofrecer una ayuda de teclado accesible con `?` desde las pantallas que no son de entrada de texto, y cerrarla con `Esc`, `Enter` o `?`. Dentro de un campo de texto o de un filtro de búsqueda, `?` se trata como un carácter más. La ayuda y el pie DEBERÁN mostrar únicamente las teclas válidas para la pantalla y el paso activos; no se listarán acciones que no estén disponibles.

#### Scenario: Abrir y cerrar la ayuda
- **GIVEN** que una pantalla que no es de entrada de texto está activa
- **WHEN** el usuario presiona `?`
- **THEN** la TUI muestra la ayuda con las teclas principales
- **AND** al presionar `Esc` la ayuda se cierra y se vuelve a la pantalla previa

#### Scenario: `?` se escribe en un campo de texto
- **GIVEN** que el foco está en un campo de texto o en un filtro de búsqueda
- **WHEN** el usuario presiona `?`
- **THEN** el carácter se agrega al valor y la ayuda no se abre

#### Scenario: La ayuda de una sección no muestra teclas ajenas
- **GIVEN** que una pantalla de sección del Sprint 3 está activa
- **WHEN** el usuario presiona `?`
- **THEN** la ayuda muestra solo las teclas válidas de esa pantalla
- **AND** no muestra acciones de creación, edición, cierre ni multiselección que no existen en la sección

### Requirement: Informar API no disponible sin cierre abrupto

Si la API no está disponible la TUI DEBERÁ informar el motivo y seguir abierta.

#### Scenario: API caída

- **WHEN** la API no responde al abrir o navegar
- **THEN** la TUI muestra el motivo del fallo y no se cierra abruptamente

### Requirement: Navegación de solo lectura
La navegación entre el menú y las pantallas DEBERÁ ser de solo lectura y NO modificará ningún dato del sistema. Solo las operaciones explícitas de escritura pueden emitir requests que modifiquen datos: la creación y modificación de proyectos (Requirement: Operar proyectos mediante la TUI), la creación y modificación de historias (Requirement: Crear historias mediante la TUI y Requirement: Modificar historias desde la lista), las operaciones de los Sprints (Requirement: Operar los Sprints mediante la TUI), el guardado del Story Points acordado en Planning Poker (Requirement: Estimar historias con Planning Poker), que emite un `PUT`, y el registro de esfuerzo (Requirement: Enviar el registro de esfuerzo y presentar los errores de la API), que emite un `POST`. Las secciones Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard, sus selectores y sus pantallas de aviso DEBERÁN emitir únicamente consultas `GET`, salvo el `PUT` de Planning Poker al confirmar el Story Points y el `POST` de Esfuerzo al confirmar el formulario de registro.
(Previously: el texto afirmaba que las secciones Planning Poker y Esfuerzo emitían únicamente consultas `GET` y solo reconocía como escrituras las operaciones de proyectos e historias; ya no contemplaba el `PUT` de Planning Poker (TUI-07), las operaciones de Sprints ni el `POST` de Esfuerzo (TUI-08).)

#### Scenario: Navegar no muta
- **WHEN** el usuario navega entre el menú y las pantallas sin confirmar una operación explícita de escritura
- **THEN** no se emite ninguna request que modifique datos

#### Scenario: Consultar el backlog no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario abre la lista del backlog o cambia de proyecto
- **THEN** las únicas requests emitidas son las consultas `GET` (del catálogo de proyectos y del backlog) y no se emite ningún `POST` ni `PUT`

#### Scenario: Navegar por las secciones del Sprint 3 no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario recorre Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard, elige filas y avanza entre selectores, sin confirmar el guardado de Story Points ni el envío del formulario de esfuerzo
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

La TUI DEBERÁ permitir operar proyectos desde la opción «Proyectos», que abre directamente el navegador buscable de proyectos (Requirement: Listar y elegir proyectos con un buscador) en lugar de un submenú con un pedido manual de identificador. Desde el navegador el usuario fija el proyecto activo, crea uno nuevo y modifica el proyecto resaltado. La gestión consume la API existente y no incorpora lógica de negocio. La navegación de menú existente se conserva.

#### Scenario: La opción Proyectos abre el navegador
- **GIVEN** que la TUI muestra el menú inicial
- **WHEN** el usuario confirma la opción «Proyectos»
- **THEN** la TUI emite `GET /projects` y presenta el navegador buscable
- **AND** no pide ningún identificador a mano

#### Scenario: Crear un proyecto

- **GIVEN** que el usuario abre la creación desde el navegador (tecla `n`) y completa nombre, fecha de inicio y fecha planificada de finalización
- **WHEN** confirma la creación
- **THEN** la TUI envía esos campos al endpoint HTTP de creación y presenta el resultado de la API

#### Scenario: La API rechaza datos de creación

- **GIVEN** que la API responde HTTP 422 con detalles de validación
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el motivo informado por la API y no comunica éxito

#### Scenario: Modificar un proyecto existente

- **GIVEN** que el usuario resalta un proyecto en el navegador y abre su modificación con `e`
- **WHEN** confirma la modificación
- **THEN** la TUI envía los campos al endpoint HTTP de actualización sin duplicar validaciones de negocio

#### Scenario: Proyecto inexistente al modificar

- **GIVEN** que la API responde HTTP 404 al modificar
- **WHEN** la TUI presenta el resultado
- **THEN** la TUI informa que el proyecto no existe y presenta el detalle de la API

#### Scenario: El estado derivado se muestra en el navegador

- **GIVEN** que el catálogo informa el estado de cada proyecto (`planned`, `active` o `overdue`)
- **WHEN** la TUI presenta el navegador
- **THEN** cada fila muestra ese estado con una etiqueta en español (`planificado`, `en curso` o `atrasado`) sin recalcularlo

#### Scenario: Error de transporte

- **GIVEN** que no se puede establecer conexión con la API
- **WHEN** falla una operación de proyectos
- **THEN** la TUI informa el problema de conexión sin cerrarse abruptamente

### Requirement: Ingresar las fechas del proyecto como año, mes y día

En el formulario de proyectos, la fecha de inicio y la fecha fin planificada DEBERÁN ingresarse como tres segmentos separados (año, mes y día) en lugar de una cadena `YYYY-MM-DD` escrita a mano. Cada segmento DEBERÁ aceptar solo dígitos, avanzar solo al completar su longitud (4 para el año, 2 para el mes y el día) y poder recorrerse con las flechas izquierda y derecha. La TUI DEBERÁ rellenar con ceros los segmentos cortos y validar la fecha localmente antes de enviarla: una fecha incompleta, una fecha imposible o una fecha fin planificada anterior a la de inicio DEBERÁN informarse en el propio formulario sin emitir requests. Los valores se enviarán a la API ya normalizados como `YYYY-MM-DD`.

#### Scenario: Completar una fecha por segmentos

- **GIVEN** el formulario de creación de proyectos con el foco en «Fecha inicio»
- **WHEN** el usuario escribe los dígitos del año, el mes y el día
- **THEN** cada segmento se completa y el foco avanza solo al llenarse
- **AND** `Enter` en el último campo envía la fecha normalizada como `YYYY-MM-DD`

#### Scenario: Fecha incompleta sin request

- **GIVEN** el formulario con algún segmento de fecha sin completar
- **WHEN** el usuario intenta enviar
- **THEN** la TUI muestra el error en el formulario y NO emite ningún request

#### Scenario: Fecha fin anterior a la de inicio

- **GIVEN** el formulario con una fecha fin planificada anterior a la fecha de inicio
- **WHEN** el usuario intenta enviar
- **THEN** la TUI muestra el error de orden en el formulario y NO emite ningún request

### Requirement: Operar el backlog mediante la TUI
La TUI DEBERÁ permitir consultar, crear y modificar historias de usuario de un proyecto desde la opción «Backlog» del menú. La gestión consume la API existente (`GET /projects/{pid}/stories`, `POST /projects/{pid}/stories`, `PUT /projects/{pid}/stories/{sid}`) y NO incorpora reglas de negocio. La pantalla alcanzada desde el menú conserva el nombre `backlog`. La lista del backlog se comporta según el patrón navegable (Requirement: Listas navegables con teclas uniformes) y NO muestra filas de acción: crear, modificar, cambiar de proyecto y refrescar se ofrecen con las teclas `n`, `Enter`, `c` y `r`.

#### Scenario: La opción Backlog da acceso a la gestión
- **GIVEN** que la TUI muestra el menú inicial
- **WHEN** el usuario se mueve a la opción «Backlog» y la confirma
- **THEN** la TUI deja de mostrar el marcador de posición TUI-03 y presenta el flujo de gestión del backlog
- **AND** la pantalla activa conserva el nombre `backlog`

#### Scenario: Acceder sin proyecto activo abre el selector
- **GIVEN** que en la sesión aún no se eligió ningún proyecto
- **WHEN** el usuario confirma la opción «Backlog»
- **THEN** la TUI presenta el selector de proyectos con el catálogo de la API
- **AND** no consulta el backlog hasta que el usuario confirme un proyecto

#### Scenario: Crear y modificar historias desde el navegador
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `n`
- **THEN** la TUI abre el formulario de creación vacío
- **AND** si resalta una historia y presiona `Enter` abre el formulario de modificación precargado

#### Scenario: Cambiar de proyecto desde el navegador
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `c`
- **THEN** la TUI abre el selector de proyectos para reemplazar el proyecto activo

### Requirement: Listar y elegir proyectos con un buscador
La TUI DEBERÁ obtener el catálogo de proyectos con `GET /projects` y presentarlo en un navegador buscable, de modo que el usuario NO necesite recordar ni escribir identificadores. El navegador es la pantalla inicial de la TUI y el punto desde el que se entra al espacio de trabajo de un proyecto. DEBERÁ permitir filtrar por nombre o identificador (la búsqueda se activa con `/` y cualquier otro carácter la inicia ya escrito), navegar con las teclas de flecha, abrir el proyecto resaltado con `Enter`, abrir la creación con `n`, abrir la modificación del proyecto resaltado con `e` y refrescar con `r`. `Esc` limpia el filtro aplicado; sin filtro no hace nada. Cada fila DEBERÁ mostrar el nombre, el estado derivado con una etiqueta en español (`planificado`, `en curso` o `atrasado`) y una forma corta del identificador, y marcar con `✓` el último proyecto abierto en la sesión. El estado `active` de la API se muestra como «en curso». Si el catálogo está vacío o la búsqueda no arroja coincidencias, la TUI DEBERÁ informarlo sin cerrarse.
(Previously: el navegador se abría desde la opción «Proyectos» del menú, servía de selector a las demás secciones y `Esc` volvía a la pantalla previa.)

#### Scenario: Abrir el navegador desde Proyectos
- **GIVEN** que la TUI arranca
- **WHEN** la API responde `GET /projects`
- **THEN** la TUI muestra el navegador con el título «Proyectos»
- **AND** cada fila muestra el nombre, el estado y la forma corta del identificador

#### Scenario: Abrir el selector desde el backlog
- **GIVEN** que la pestaña Backlog muestra su lista sin filtro aplicado
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI emite `GET /projects` y muestra el navegador
- **AND** cada fila muestra el nombre, el estado y la forma corta del identificador

#### Scenario: Volver sin cambiar de proyecto
- **GIVEN** que el usuario volvió al navegador desde el espacio de trabajo de un proyecto
- **WHEN** presiona `Esc` sin filtro aplicado
- **THEN** la TUI permanece en el navegador
- **AND** ese proyecto sigue marcado con `✓` como el último abierto

#### Scenario: Filtrar el catálogo
- **GIVEN** que el navegador muestra el catálogo
- **WHEN** el usuario activa la búsqueda con `/` (o empieza a escribir) y tipea parte de un nombre o identificador
- **THEN** la lista se reduce a las coincidencias
- **AND** con `Enter` se elige la primera coincidencia

#### Scenario: Editar el proyecto resaltado
- **GIVEN** que el navegador muestra el catálogo con un proyecto resaltado
- **WHEN** el usuario presiona `e`
- **THEN** la TUI abre el formulario de modificación precargado con ese proyecto

#### Scenario: Crear un proyecto desde el navegador
- **GIVEN** que el navegador está activo
- **WHEN** el usuario presiona `n`
- **THEN** la TUI abre el formulario de creación vacío

#### Scenario: Catálogo vacío
- **GIVEN** que la API responde `GET /projects` con una lista vacía
- **WHEN** la TUI presenta el resultado
- **THEN** muestra un mensaje que indica que todavía no hay proyectos y que se crea uno con `n`
- **AND** permanece abierta

#### Scenario: Búsqueda sin coincidencias
- **GIVEN** que el usuario escribe un filtro que no coincide con ningún proyecto
- **WHEN** la TUI presenta la lista filtrada
- **THEN** muestra un mensaje de «sin coincidencias»
- **AND** el usuario puede borrar el filtro

### Requirement: Precargar la modificación de proyecto y resolver el resultado
El formulario de modificación DEBERÁ abrirse precargado con el nombre, la fecha de inicio y la fecha fin planificada del proyecto resaltado en el navegador, para no reescribirlos. Tras una creación o modificación exitosa, la TUI DEBERÁ volver al navegador recargando el catálogo y mostrar un aviso efímero de éxito. Ante un rechazo de la API, la TUI DEBERÁ permanecer en el formulario conservando los valores escritos y mostrando el motivo informado por la API.
(Previously: el éxito y el rechazo pasaban por una pantalla de resultado que había que cerrar con una tecla.)

#### Scenario: Formulario de modificación precargado
- **GIVEN** que el navegador muestra un proyecto con nombre y fechas
- **WHEN** el usuario abre su modificación con `e`
- **THEN** el formulario muestra el nombre y las dos fechas ya cargados en sus segmentos

#### Scenario: Vuelta al navegador tras el éxito
- **GIVEN** que una creación o modificación responde con éxito
- **WHEN** la TUI procesa la respuesta
- **THEN** vuelve al navegador, recarga `GET /projects` y muestra un aviso que empieza con `✓`

#### Scenario: Vuelta al formulario tras el rechazo
- **GIVEN** que la API rechaza los datos con un error
- **WHEN** la TUI procesa la respuesta
- **THEN** permanece en el formulario con los valores escritos y muestra el motivo informado por la API

### Requirement: Listar el backlog del proyecto
La TUI DEBERÁ consultar el backlog del proyecto activo y presentar las historias en el mismo orden que informa la API, sin reordenarlas. Cada historia DEBERÁ mostrar al menos su título, prioridad y estado, y `story_points` solo como lectura. Cuando `story_points` sea nulo la TUI DEBERÁ mostrar «—». La lista DEBERÁ ser seleccionable.

#### Scenario: Listar historias en el orden de la API
- **GIVEN** un proyecto activo cuya API responde HTTP 200 con varias historias en un orden determinado
- **WHEN** la TUI presenta la lista
- **THEN** muestra las historias exactamente en el orden informado por la API
- **AND** permite moverse entre ellas con las teclas de navegación

#### Scenario: `story_points` nulo se muestra como «—»
- **GIVEN** una historia cuyo `story_points` es nulo en la respuesta de la API
- **WHEN** la TUI presenta la lista
- **THEN** la historia muestra «—» en lugar de `story_points`

#### Scenario: `story_points` informado se muestra de solo lectura
- **GIVEN** una historia cuyo `story_points` tiene un valor numérico en la respuesta de la API
- **WHEN** la TUI presenta la lista y el formulario de modificación de esa historia
- **THEN** la lista muestra ese valor
- **AND** el formulario no ofrece un campo editable para `story_points`

#### Scenario: Backlog vacío
- **GIVEN** un proyecto activo cuya API responde HTTP 200 con una lista de historias vacía
- **WHEN** la TUI presenta el resultado
- **THEN** muestra un mensaje que indica que el backlog no tiene historias
- **AND** sigue ofreciendo crear una historia y cambiar de proyecto

### Requirement: Buscar y refrescar en el backlog
La lista del backlog DEBERÁ ofrecer una búsqueda por título activada con `/` y cancelable con `Esc`, y un refresco de la consulta activado con `r`. La búsqueda NO emitirá requests y solo filtrará las historias ya cargadas. Mientras la búsqueda tiene el foco, `q`, `?` y `p` se tratan como caracteres de texto.

#### Scenario: Filtrar historias por título
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `/` y escribe parte de un título
- **THEN** la lista muestra solo las historias cuyo título coincide
- **AND** no se emite ninguna request a la API

#### Scenario: Búsqueda sin coincidencias
- **GIVEN** que la búsqueda está activa con un filtro que no coincide
- **WHEN** la TUI presenta la lista
- **THEN** muestra un mensaje de «sin coincidencias»

#### Scenario: Cancelar la búsqueda
- **GIVEN** que la búsqueda está activa
- **WHEN** el usuario presiona `Esc`
- **THEN** el filtro se borra y la lista vuelve a mostrar todas las historias

#### Scenario: Refrescar el backlog
- **GIVEN** que la lista del backlog está cargada
- **WHEN** el usuario presiona `r`
- **THEN** la TUI vuelve a consultar `GET /projects/{pid}/stories`

### Requirement: Crear historias mediante la TUI
La TUI DEBERÁ permitir crear una historia con título, descripción, prioridad y criterios de aceptación, y enviarla al endpoint HTTP de creación del proyecto del espacio de trabajo. La prioridad se ingresa como texto libre y la TUI NO validará sus valores permitidos. Los criterios de aceptación se ingresan en una sola línea separada por `;`; la TUI DEBERÁ recortar los espacios de cada elemento, descartar los elementos vacíos y enviarlos como lista. El cuerpo enviado DEBERÁ contener solo las claves `title`, `description`, `priority` y `acceptance_criteria`. El éxito y el rechazo se resuelven según Requirement: Avisos efímeros en lugar de pantallas de resultado.
(Previously: el resultado de la API se presentaba en una pantalla de resultado.)

#### Scenario: Crear una historia
- **GIVEN** que el usuario completa título, descripción, prioridad y criterios de aceptación
- **WHEN** confirma la creación
- **THEN** la TUI envía `POST /projects/{pid}/stories` con `title`, `description`, `priority` y `acceptance_criteria`
- **AND** ante el éxito vuelve a la lista del backlog recargada con un aviso «✓ Historia creada» seguido del título devuelto por la API

#### Scenario: Criterios de aceptación separados por `;`
- **GIVEN** que el usuario escribe en la línea de criterios el texto `  Inicia sesión ; Cierra sesión ;; `
- **WHEN** confirma la creación
- **THEN** el cuerpo enviado contiene `acceptance_criteria` igual a la lista `["Inicia sesión", "Cierra sesión"]`

#### Scenario: Criterios de aceptación vacíos
- **GIVEN** que el usuario deja vacía la línea de criterios
- **WHEN** confirma la creación
- **THEN** el cuerpo enviado contiene `acceptance_criteria` como lista vacía (no nula ni omitida)

#### Scenario: La prioridad no se valida localmente
- **GIVEN** que el usuario escribe en prioridad un valor que el backend no admite
- **WHEN** confirma la creación
- **THEN** la TUI envía el valor tal como fue escrito
- **AND** el rechazo, si corresponde, proviene de la API

#### Scenario: Los campos aceptan espacios
- **GIVEN** que el usuario está escribiendo el título «Como usuario quiero ingresar»
- **WHEN** teclea los espacios entre palabras
- **THEN** el campo refleja los espacios y el cuerpo enviado contiene el título completo con ellos

### Requirement: Modificar historias desde la lista
La TUI DEBERÁ permitir modificar una historia seleccionándola de la lista cargada del proyecto activo. El formulario DEBERÁ abrirse precargado con los valores actuales de la historia seleccionada (`title`, `description`, `priority`, `status`, `acceptance_criteria` unidos con `;` y `estimated_hours`). Como el `PUT` es un reemplazo total, la TUI DEBERÁ enviar siempre las seis claves `title`, `description`, `priority`, `status`, `acceptance_criteria` y `estimated_hours` a `PUT /projects/{pid}/stories/{sid}`, con `estimated_hours` igual a `null` cuando el campo queda vacío. La prioridad y el estado se ingresan como texto libre y la TUI NO validará sus valores permitidos. `story_points` NO es editable y NO se envía.

#### Scenario: El formulario se precarga con la historia seleccionada
- **GIVEN** una lista cargada con una historia de título «Login», prioridad «alta», estado «pendiente», criterios `["A", "B"]` y `estimated_hours` 5
- **WHEN** el usuario selecciona esa historia y elige modificarla
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
- **WHEN** el usuario intenta modificar una historia
- **THEN** la TUI no abre el formulario de modificación
- **AND** no emite ninguna request a la API

### Requirement: Análisis numérico mínimo de `estimated_hours`
La TUI DEBERÁ interpretar `estimated_hours` como número únicamente para componer el cuerpo de la request: un valor vacío equivale a `null` y un valor numérico se envía como número. Si el valor no es numérico, la TUI DEBERÁ informar el problema sin emitir la request. La TUI NO replicará reglas de negocio sobre el rango del valor.

#### Scenario: Valor numérico entero
- **GIVEN** que el usuario escribe `8` en `estimated_hours`
- **WHEN** confirma la operación
- **THEN** el cuerpo enviado contiene `estimated_hours` como el número 8

#### Scenario: Valor numérico decimal
- **GIVEN** que el usuario escribe `2.5` en `estimated_hours`
- **WHEN** confirma la operación
- **THEN** el cuerpo enviado contiene `estimated_hours` como el número 2.5

#### Scenario: Valor no numérico
- **GIVEN** que el usuario escribe un texto no numérico (por ejemplo `abc`) en `estimated_hours`
- **WHEN** confirma la operación
- **THEN** la TUI informa que el valor debe ser numérico
- **AND** no emite ninguna request a la API
- **AND** permanece en el formulario conservando lo escrito

#### Scenario: El rango no se valida localmente
- **GIVEN** que el usuario escribe un número negativo en `estimated_hours`
- **WHEN** confirma la operación
- **THEN** la TUI envía el valor tal como fue escrito
- **AND** el rechazo, si corresponde, proviene de la API

### Requirement: Presentar los errores de la API del backlog
La TUI DEBERÁ presentar el resultado de cada operación del backlog sin cerrarse. Los errores de creación y modificación se muestran en el formulario, que conserva los valores escritos; los errores de la consulta de la lista se muestran en la lista del backlog. Ante HTTP 422 DEBERÁ mostrar el detalle de `fields` como una lista formateada. Ante HTTP 404 DEBERÁ mostrar un mensaje genérico de recurso inexistente más el detalle informado por la API. Las respuestas de error que no sean JSON (por ejemplo, HTTP 404 o 405 del enrutador cuando las rutas de historias no están registradas) DEBERÁN presentarse como error de la API con el código de estado, sin cerrar la TUI. Los errores de transporte DEBERÁN informar el problema de conexión sin cerrar la TUI. Nunca se comunicará éxito ante un error.
(Previously: los errores y los éxitos se mostraban en una pantalla de resultado desde la que se volvía al formulario, a la lista o al selector de proyectos.)

#### Scenario: HTTP 422 con campos inválidos
- **GIVEN** que la API responde HTTP 422 con un `fields` que detalla al menos dos campos inválidos
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra cada campo y su motivo como un elemento de lista formateada
- **AND** no comunica éxito

#### Scenario: HTTP 404 proyecto o historia inexistente
- **GIVEN** que la API responde HTTP 404 con un detalle (por ejemplo `project_not_found` o `story_not_found`)
- **WHEN** la TUI presenta el resultado
- **THEN** muestra un mensaje genérico de que el recurso no existe
- **AND** muestra también el detalle informado por la API

#### Scenario: HTTP 404 o 405 sin cuerpo JSON
- **GIVEN** que la API responde HTTP 404 o HTTP 405 con un cuerpo que no es JSON
- **WHEN** la TUI procesa la respuesta
- **THEN** presenta un error de la API que incluye el código de estado recibido
- **AND** no falla al interpretar el cuerpo ni se cierra

#### Scenario: HTTP 422 al consultar la lista del proyecto activo
- **GIVEN** que la API rechaza la consulta del backlog con HTTP 422
- **WHEN** la TUI presenta el resultado de la consulta
- **THEN** la lista del backlog muestra el detalle de `fields` como lista formateada
- **AND** `Esc` vuelve al navegador de proyectos

#### Scenario: Error de transporte
- **GIVEN** que no se puede establecer conexión con la API
- **WHEN** falla una operación del backlog (consulta, creación o modificación)
- **THEN** la TUI informa el problema de conexión sin cerrarse abruptamente

#### Scenario: Resultado exitoso de creación o modificación
- **GIVEN** que la API responde HTTP 201 (creación) o HTTP 200 (modificación) con la historia
- **WHEN** la TUI procesa la respuesta
- **THEN** vuelve a la lista del backlog, la recarga y muestra un aviso que empieza con `✓` y nombra la historia informada por la API

### Requirement: Los campos de texto aceptan espacios
Los campos de entrada de texto de todas las pantallas con formulario (proyectos y backlog) DEBERÁN aceptar la barra espaciadora como un carácter de espacio y reflejarlo en el valor del campo.

#### Scenario: Espacio en un campo de texto
- **GIVEN** que el foco está en un campo de texto de un formulario
- **WHEN** el usuario presiona la barra espaciadora
- **THEN** el campo agrega un carácter de espacio en la posición del cursor y lo muestra

#### Scenario: Espacio en un formulario de proyectos
- **GIVEN** que el usuario escribe el nombre «Proyecto Alfa» en el formulario de creación de proyectos
- **WHEN** confirma la creación
- **THEN** el nombre enviado a la API conserva el espacio

### Requirement: Los campos de texto borran por carácter
Los campos de entrada de texto de todas las pantallas con formulario (proyectos y backlog) DEBERÁN borrar un carácter completo (una runa Unicode) por cada pulsación de retroceso, de modo que el valor del campo sea siempre UTF-8 válido. Una pulsación de retroceso sobre un campo vacío NO tendrá efecto.

#### Scenario: Borrar un carácter acentuado
- **GIVEN** que el foco está en un campo de texto que contiene «Canción»
- **WHEN** el usuario presiona retroceso una vez
- **THEN** el campo contiene «Canció»
- **AND** el valor del campo es UTF-8 válido

#### Scenario: Borrar hasta vaciar un campo con caracteres multibyte
- **GIVEN** que el foco está en un campo de texto que contiene «añá»
- **WHEN** el usuario presiona retroceso tres veces
- **THEN** el campo queda vacío
- **AND** el valor del campo fue UTF-8 válido después de cada pulsación

#### Scenario: Retroceso en un campo vacío
- **GIVEN** que el foco está en un campo de texto vacío
- **WHEN** el usuario presiona retroceso
- **THEN** el campo permanece vacío y la TUI no falla

#### Scenario: El valor enviado conserva los caracteres acentuados
- **GIVEN** que el usuario escribe el título «Gestión añ», borra un carácter y escribe «o»
- **WHEN** confirma la operación
- **THEN** el título enviado a la API es «Gestión ao» y es UTF-8 válido

### Requirement: Listas navegables con teclas uniformes

Las pantallas que muestran colecciones (proyectos, historias y Sprints) DEBERÁN comportarse como un navegador con el mismo patrón de interacción: la búsqueda se activa con `/` y cualquier otro carácter la inicia ya escrito, salvo las teclas reservadas de la pantalla y, en las pantallas raíz del espacio de trabajo, los dígitos `1` a `6`; `↑`/`↓` mueven la selección; `Enter` ejecuta la acción primaria de la fila resaltada; `n` inicia la creación; `e` modifica el elemento resaltado en las listas donde editar no es la acción primaria (por ejemplo, Proyectos); `r` refresca el listado; `x` ejecuta una acción destructiva sobre la fila resaltada (pidiendo confirmación); `h` abre una vista relacionada de solo lectura; y `Esc` primero limpia el filtro y luego vuelve a la pantalla anterior. Cada fila DEBERÁ mostrar el contenido y sus etiquetas, con un marcador inicial cuando corresponda (por ejemplo `✓` en el proyecto abierto) y una forma corta del identificador en las listas que lo necesiten para distinguir elementos (por ejemplo, proyectos y Sprints). Ninguna lista DEBERÁ pedir un identificador escrito a mano. Las pantallas de formulario NO adoptan este patrón.
(Previously: cualquier carácter no reservado iniciaba la búsqueda, incluidos los dígitos.)

#### Scenario: Buscar en cualquier colección
- **GIVEN** que una colección está visible (proyectos, backlog, Sprints abiertos, historias de un Sprint o Sprints cerrados)
- **WHEN** el usuario presiona `/` o empieza a escribir una letra
- **THEN** la lista se reduce a las coincidencias sin emitir requests

#### Scenario: Acción primaria con Enter
- **GIVEN** que una fila está resaltada en una colección
- **WHEN** el usuario presiona `Enter`
- **THEN** la TUI ejecuta la acción primaria de esa colección sobre la fila resaltada

#### Scenario: Esc limpia el filtro antes de volver
- **GIVEN** que una lista tiene un filtro aplicado
- **WHEN** el usuario presiona `Esc`
- **THEN** el filtro se borra y la lista vuelve a mostrarlo todo
- **AND** un segundo `Esc` vuelve a la pantalla anterior

#### Scenario: Ninguna lista pide identificadores a mano
- **WHEN** el usuario opera cualquier colección
- **THEN** la TUI nunca solicita que se escriba un identificador

### Requirement: Estimar historias con Planning Poker

La TUI DEBERÁ permitir seleccionar una historia del backlog del proyecto activo y elegir uno o más integrantes del proyecto como participantes. El facilitador ingresará un voto por participante en la misma terminal usando `0, 1, 2, 3, 5, 8, 13, 20, 40, 100` o `?`. Los votos serán transitorios en memoria y NO DEBERÁN mostrarse hasta que el facilitador revele explícitamente la ronda. La revelación mostrará cada participante y su voto, e indicará si hay diferencias. Una nueva ronda DEBERÁ ocultar y reiniciar los votos. La TUI NO calculará consenso ni persistirá votos.

Luego de revisar la ronda, el facilitador DEBERÁ elegir explícitamente un Story Points numérico acordado y confirmarlo. La TUI enviará únicamente `PUT /projects/{project_id}/stories/{story_id}/story-points` con `{"story_points": <integer>}`. La validación corresponde a la API; `?` no se puede guardar. Tras el éxito se actualizará el backlog mediante una nueva consulta. Ante errores 404, 409 o 422 se mostrará el motivo de la API y no se afirmará que el cambio se guardó ni se modificará el Story Points local.

#### Scenario: Votos privados y revelación explícita
- **GIVEN** que se seleccionaron participantes y se ingresaron votos
- **WHEN** la ronda todavía no fue revelada
- **THEN** las vistas de voto no muestran ningún valor de voto
- **AND** solo la acción explícita de revelar muestra los votos e indica si difieren

#### Scenario: Reiniciar una ronda
- **GIVEN** que los votos de una ronda fueron revelados
- **WHEN** el facilitador inicia una nueva ronda
- **THEN** los votos anteriores dejan de mostrarse y cada participante vuelve a votar

#### Scenario: Guardar el estimado acordado
- **GIVEN** una historia seleccionada y una ronda revisada
- **WHEN** el facilitador elige un valor numérico y confirma explícitamente
- **THEN** la TUI envía el PUT dedicado con el cuerpo `story_points`
- **AND** tras el éxito vuelve a mostrar el backlog actualizado

#### Scenario: Fallo al guardar el estimado
- **GIVEN** que la API responde 404, 409 o 422
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el motivo informado por la API sin comunicar éxito ni alterar el Story Points local

### Requirement: Operar los Sprints mediante la TUI
La TUI DEBERÁ permitir operar los Sprints del proyecto del espacio de trabajo desde la pestaña Sprints, que muestra el navegador de Sprints abiertos. Desde el navegador el usuario abre las historias de un Sprint (`Enter`), asigna historias del backlog al Sprint resaltado (`a`), cierra el Sprint resaltado (`x`, con confirmación), crea un Sprint (`n`) y consulta los Sprints cerrados con `h`. Las pantallas de asignación (multiselección con `Space`) y de historias del Sprint reutilizan el patrón navegable (Requirement: Listas navegables con teclas uniformes) y consumen la API existente sin incorporar reglas de negocio. El éxito y el rechazo de cada escritura se resuelven según Requirement: Avisos efímeros en lugar de pantallas de resultado; los errores de las consultas se muestran en la pantalla que las pidió.
(Previously: los Sprints se abrían desde la opción «Sprints» del menú, pedían un proyecto si no había uno activo y cada operación terminaba en una pantalla de resultado.)

#### Scenario: La opción Sprints abre el navegador
- **GIVEN** que el espacio de trabajo de un proyecto está abierto
- **WHEN** el usuario abre la pestaña Sprints
- **THEN** la TUI emite `GET /projects/{pid}/sprints/open` y presenta el navegador de Sprints abiertos
- **AND** no pide ningún identificador a mano

#### Scenario: Sin proyecto activo abre el selector
- **GIVEN** que en la sesión aún no se abrió ningún proyecto y la API devuelve varios
- **WHEN** la TUI arranca
- **THEN** presenta el navegador de proyectos con el catálogo de la API
- **AND** no consulta los Sprints hasta que el usuario abra un proyecto y su pestaña Sprints

#### Scenario: Crear un Sprint vuelve a la lista con un aviso
- **GIVEN** que el usuario escribió el objetivo «Meta» en el formulario de un Sprint nuevo
- **WHEN** confirma y la API responde con éxito
- **THEN** la TUI vuelve a la lista de Sprints abiertos, la recarga y muestra «✓ Sprint creado: Meta»

#### Scenario: Asignar historias al Sprint resaltado
- **GIVEN** que el navegador de Sprints muestra un Sprint abierto resaltado
- **WHEN** el usuario presiona `a`, marca historias con `Space` y confirma con `Enter`
- **THEN** la TUI envía `POST /projects/{pid}/sprints/{sid}/stories` con los identificadores marcados
- **AND** ante el éxito vuelve a la lista de Sprints recargada con un aviso que empieza con `✓`

#### Scenario: Cerrar un Sprint pide confirmación
- **GIVEN** que el navegador de Sprints muestra un Sprint abierto resaltado
- **WHEN** el usuario presiona `x`
- **THEN** la TUI pide confirmación antes de emitir `POST /projects/{pid}/sprints/{sid}/closure`
- **AND** si el usuario cancela no se emite ningún request
- **AND** si la API rechaza el cierre la confirmación permanece con el motivo informado

#### Scenario: Crear tareas y completar historias desde el Sprint
- **GIVEN** que el usuario abrió las historias de un Sprint abierto
- **WHEN** resalta una historia y presiona `Enter` o `c`
- **THEN** `Enter` abre el formulario de tareas de esa historia
- **AND** `c` envía `POST /projects/{pid}/sprints/{sid}/stories/{story_id}/completion`
- **AND** ante el éxito de cualquiera de las dos la TUI vuelve a las historias del Sprint recargadas con un aviso que empieza con `✓`

#### Scenario: Consultar los Sprints cerrados
- **GIVEN** que el navegador de Sprints está activo
- **WHEN** el usuario presiona `h`
- **THEN** la TUI emite `GET /projects/{pid}/sprints` y presenta los Sprints cerrados con su progreso
- **AND** `Enter` despliega las historias del Sprint resaltado sin emitir requests

#### Scenario: Backlog vacío en la asignación
- **GIVEN** que el proyecto no tiene historias en el backlog
- **WHEN** el usuario intenta asignar historias a un Sprint
- **THEN** la TUI lo informa y no emite ningún request de asignación

### Requirement: Elegir integrantes, historias, tareas y Sprints con selectores navegables
La TUI DEBERÁ ofrecer selectores de integrante, historia, tarea y Sprint como listas navegables con el mismo patrón de las colecciones existentes: la búsqueda se activa con `/` o al empezar a escribir, `↑`/`↓` mueven la selección, `Enter` ejecuta la acción primaria, `Esc` limpia el filtro y luego vuelve, `r` refresca y `?` abre la ayuda. Ningún selector DEBERÁ pedir un identificador escrito a mano. Los selectores DEBERÁN mostrar un estado de carga, un estado vacío y el error de la API sin cerrar la TUI. El selector de tareas de Esfuerzo es la excepción del estado vacío: antepone la fila «Toda la historia (sin tarea)» (Requirement: Elegir historia, tarea e integrante para registrar esfuerzo), por lo que ante una historia sin tareas muestra esa fila e indica que la historia no tiene tareas, en lugar de un estado vacío bloqueante.
(Previously: todos los selectores mostraban un estado vacío ante una colección vacía, sin excepción para las tareas de Esfuerzo.)

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

#### Scenario: Los cuatro tipos de selector son navegables
- **GIVEN** que el usuario recorre Planning Poker, Esfuerzo y Defectos
- **WHEN** elige una historia, un integrante, una tarea y un Sprint
- **THEN** cada colección se presenta como una lista navegable con las teclas comunes
- **AND** la selección de una fila no emite requests que modifiquen datos

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
### Requirement: Elegir historia, tarea e integrante para registrar esfuerzo
La sección Esfuerzo DEBERÁ guiar el registro con los selectores navegables existentes, en este orden: historia del backlog del proyecto activo, tarea de esa historia (o la historia completa) e integrante del proyecto activo. Una vez cargadas las tareas de la historia elegida, el selector de tareas DEBERÁ anteponer, como primera fila, la fila sintética «Toda la historia (sin tarea)», seguida de las tareas en el orden informado por la API; la fila sintética DEBERÁ estar presente también cuando la historia no tiene tareas. Elegir una tarea registra el esfuerzo sobre esa tarea; elegir la fila sintética lo registra sobre la historia completa. Elegir cualquiera de las dos filas DEBERÁ abrir el selector de integrantes del proyecto activo, que en Esfuerzo es de selección simple: `Enter` elige el integrante resaltado y abre el formulario de esfuerzo (Requirement: Completar el formulario de esfuerzo), y `Space` no marca integrantes. Ningún paso DEBERÁ pedir un identificador escrito a mano. El avance entre selectores y la apertura del formulario NO DEBERÁN emitir requests que modifiquen datos. Si el proyecto no tiene integrantes, el selector de integrantes DEBERÁ mostrar un estado vacío propio que informe que el proyecto no tiene integrantes registrados, que NO remita a ninguna opción del menú principal y que permita volver con `Esc` y refrescar con `r`.

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
- **AND** el mensaje no remite a ninguna opción del menú principal
- **AND** `Esc` vuelve al selector de tareas y `r` vuelve a consultar los integrantes

#### Scenario: Falla la consulta de integrantes
- **GIVEN** que la API responde HTTP 500 al consultar los integrantes
- **WHEN** la TUI presenta el selector de integrantes
- **THEN** muestra el error de la API dentro de la pantalla sin cerrarse
- **AND** no abre el formulario de esfuerzo
- **AND** permite reintentar con `r` o volver con `Esc`

### Requirement: Completar el formulario de esfuerzo
El formulario de esfuerzo DEBERÁ ser una pantalla propia, distinta de la pantalla de sección, con tres campos en este orden: «Fecha» (segmentada en año, mes y día), «Actividad» (texto) y «Horas» (texto). DEBERÁ mostrar, solo como lectura, la historia, la tarea (o «Toda la historia») y el integrante elegidos. El foco inicial DEBERÁ estar en «Fecha». La fecha DEBERÁ abrirse con la fecha UTC actual según el reloj de la TUI y ser editable; el ingreso de la fecha usa las mismas reglas que el formulario de proyectos (Requirement: Ingresar las fechas del proyecto como año, mes y día): solo dígitos, avance automático al completar cada segmento, flechas izquierda y derecha entre segmentos, `-` y `/` como separadores, retroceso y relleno con ceros. `Tab` y `↓` DEBERÁN pasar al campo siguiente y `Shift+Tab` y `↑` al anterior, de forma circular; `Enter` en «Fecha» o «Actividad» DEBERÁ pasar al campo siguiente y `Enter` en «Horas» DEBERÁ confirmar el envío. Cada apertura del formulario DEBERÁ comenzar con la fecha de hoy y con «Actividad» y «Horas» vacías. Dentro del formulario `q`, `Q`, `?`, `p` y `r` DEBERÁN tratarse como caracteres de texto en «Actividad» y «Horas»: no cierran la TUI, no abren la ayuda, no cambian el proyecto activo ni refrescan nada; `Ctrl+C` sigue cerrando la TUI. Los campos de texto DEBERÁN aceptar espacios y borrar una runa Unicode completa por pulsación de retroceso. El pie y la ayuda del formulario DEBERÁN mostrar únicamente las teclas válidas en él, sin ofrecer `q`, `p`, `r` ni `?` como acciones.

#### Scenario: La fecha inicial es la fecha UTC actual
- **GIVEN** que el reloj de la TUI indica `2026-10-31T23:30:00-03:00`, equivalente a `2026-11-01T02:30:00Z`
- **WHEN** el usuario abre el formulario de esfuerzo
- **THEN** la fecha muestra año 2026, mes 11 y día 01
- **AND** «Actividad» y «Horas» están vacías
- **AND** el foco está en «Fecha»

#### Scenario: Editar la fecha por segmentos
- **GIVEN** el formulario abierto con el foco en «Fecha»
- **WHEN** el usuario borra la fecha con retroceso y escribe `20261005`
- **THEN** los segmentos muestran año 2026, mes 10 y día 05
- **AND** el foco avanzó solo al llenarse cada segmento

#### Scenario: `Enter` avanza de campo sin enviar
- **GIVEN** el formulario con el foco en «Fecha»
- **WHEN** el usuario presiona `Enter` y luego `Enter` otra vez
- **THEN** el foco pasa a «Actividad» y después a «Horas»
- **AND** no se emite ninguna request

#### Scenario: Navegar con `Tab` y flechas de forma circular
- **GIVEN** el formulario con el foco en «Horas»
- **WHEN** el usuario presiona `Tab`
- **THEN** el foco vuelve a «Fecha»
- **AND** `Shift+Tab` desde «Fecha» lleva el foco a «Horas»

#### Scenario: Las teclas globales son caracteres en los campos de texto
- **GIVEN** el formulario con el foco en «Actividad»
- **WHEN** el usuario escribe `q`, `Q`, `?`, `p` y `r`
- **THEN** «Actividad» contiene `qQ?pr`
- **AND** la TUI sigue abierta en el formulario, la ayuda no se abrió y no se emitió ninguna request
- **AND** lo mismo ocurre con esos caracteres en «Horas»

#### Scenario: Espacios y retroceso por carácter en «Actividad»
- **GIVEN** el formulario con el foco en «Actividad»
- **WHEN** el usuario escribe «Revisión de diseño» y presiona retroceso una vez
- **THEN** el campo contiene «Revisión de diseñ»
- **AND** el valor es UTF-8 válido

#### Scenario: El pie del formulario no ofrece teclas ajenas
- **GIVEN** el formulario de esfuerzo activo
- **WHEN** la TUI muestra el pie
- **THEN** el pie incluye `Tab`, `Enter` y `Esc`
- **AND** no ofrece `q` para salir, `p` para cambiar de proyecto, `r` para refrescar ni `?` para la ayuda

#### Scenario: El formulario de proyectos conserva su ingreso de fechas
- **GIVEN** el formulario de creación de proyectos
- **WHEN** el usuario completa las fechas por segmentos, deja una incompleta o ingresa una fecha fin anterior a la de inicio
- **THEN** se mantienen los comportamientos del Requirement: Ingresar las fechas del proyecto como año, mes y día, sin cambios

### Requirement: Validar localmente la fecha y las horas del esfuerzo
Antes de emitir cualquier request, la TUI DEBERÁ validar únicamente que (a) la fecha esté completa y sea una fecha real del calendario y (b) «Horas», una vez recortados los espacios de los extremos, sea un número finito escrito con punto decimal y mayor que 0. La coma decimal NO se acepta. Si alguna validación falla, la TUI DEBERÁ mostrar el motivo en el formulario, nombrando el campo, conservar todos los valores escritos y NO emitir ninguna request; si fallan ambos campos se informa primero «Fecha». El error local DEBERÁ limpiarse al editar un campo. La TUI NO DEBERÁ validar localmente el máximo de horas, la cantidad de decimales, la longitud ni la ausencia de «Actividad», ni que la fecha sea futura: esas reglas las aplica la API y la TUI muestra su motivo. El integrante y la historia siempre están elegidos al llegar al formulario porque el flujo no permite abrirlo sin ellos.

#### Scenario: Fecha incompleta
- **GIVEN** el formulario con el día de la fecha vacío y «Horas» igual a `2`
- **WHEN** el usuario confirma desde «Horas»
- **THEN** la TUI muestra en el formulario que la fecha está incompleta
- **AND** no emite ninguna request y conserva los valores escritos

#### Scenario: Fecha imposible
- **GIVEN** el formulario con la fecha 2026-02-30 y «Horas» igual a `2`
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que la fecha no es válida
- **AND** no emite ninguna request

#### Scenario: Horas vacías
- **GIVEN** el formulario con fecha válida y «Horas» vacía
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request

#### Scenario: Horas no numéricas
- **GIVEN** el formulario con fecha válida y «Horas» igual a `abc`
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request y «Horas» conserva `abc`

#### Scenario: Coma decimal rechazada
- **GIVEN** el formulario con fecha válida y «Horas» igual a `1,5`
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request

#### Scenario: Horas igual a cero o negativas
- **GIVEN** el formulario con fecha válida
- **WHEN** el usuario confirma con «Horas» igual a `0` y luego con «Horas» igual a `-2`
- **THEN** en ambos casos la TUI muestra que las horas deben ser mayores que 0
- **AND** no emite ninguna request

#### Scenario: Horas no finitas
- **GIVEN** el formulario con fecha válida
- **WHEN** el usuario confirma con «Horas» igual a `NaN` y luego con «Horas» igual a `Inf`
- **THEN** en ambos casos la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request

#### Scenario: Con fecha y horas inválidas se informa primero la fecha
- **GIVEN** el formulario con la fecha incompleta y «Horas» igual a `abc`
- **WHEN** el usuario confirma
- **THEN** el error mostrado corresponde a la fecha
- **AND** no se emite ninguna request

#### Scenario: El error local se limpia al corregir
- **GIVEN** el formulario mostrando el error de horas por «Horas» igual a `0`
- **WHEN** el usuario edita «Horas»
- **THEN** el error deja de mostrarse
- **AND** al confirmar con `2` la TUI emite la request de registro

#### Scenario: El rango y los decimales los decide la API
- **GIVEN** el formulario con fecha válida, «Actividad» con texto y «Horas» igual a `25`
- **WHEN** el usuario confirma
- **THEN** la TUI emite la request con `hours` igual al número 25 sin rechazarla localmente
- **AND** lo mismo ocurre con «Horas» igual a `1.234`

#### Scenario: La actividad vacía y la fecha futura los decide la API
- **GIVEN** el formulario con «Actividad» vacía, «Horas» igual a `2` y una fecha posterior a la fecha UTC actual
- **WHEN** el usuario confirma
- **THEN** la TUI emite la request con `description` vacía y esa fecha, sin rechazarla localmente

### Requirement: Enviar el registro de esfuerzo y presentar los errores de la API
Al confirmar con datos válidos, la TUI DEBERÁ emitir exactamente una request `POST /projects/{pid}/stories/{sid}/effort-records`, con el proyecto del espacio de trabajo y la historia elegida, y un cuerpo JSON cuyas claves son `member_id`, `work_date` (normalizada como `YYYY-MM-DD`), `description` (tal como fue escrita, sin recortar ni validar), `hours` (número JSON) y `task_id` únicamente cuando se eligió una tarea; al elegir «Toda la historia (sin tarea)» la clave `task_id` DEBERÁ omitirse por completo, no enviarse como `null`. Mientras el `POST` está en curso la TUI DEBERÁ indicar que está guardando e ignorar `Enter` y `Esc`, de modo que no se emita una segunda request ni se abandone el formulario antes de conocer el resultado. Solo la respuesta HTTP 201 comunica éxito: la TUI vuelve al selector de historias de la pestaña Esfuerzo, lo recarga y muestra un aviso con las horas y la fecha devueltas por la API (o las enviadas si el cuerpo no se puede interpretar), la historia, la tarea («toda la historia» si no se eligió una) y el integrante (Requirement: Avisos efímeros en lugar de pantallas de resultado). Ante cualquier otra respuesta de error o ante un error de transporte, la TUI DEBERÁ permanecer en el formulario con todos los valores escritos, dejar de indicar el guardado, mostrar el motivo y NO comunicar éxito. Los errores de la API (HTTP 400, 404, 422 con `fields`, 500 y cualquier otro código no exitoso, incluido 409 si alguna vez se devolviera) DEBERÁN presentarse con las reglas del Requirement: Presentar los errores de la API del backlog. Un error de transporte DEBERÁ informar el problema de conexión y que no se pudo confirmar el registro, y aconsejar verificar antes de reintentar, porque la API no es idempotente. Tras un error el usuario puede corregir y confirmar de nuevo, lo que emite una nueva request.
(Previously: el HTTP 201 abría una pantalla de resultado con el registro y las métricas del proyecto.)

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
- **AND** si la API responde HTTP 201 vuelve al selector de historias de Esfuerzo con el aviso de éxito

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

### Requirement: Iniciar en el navegador de proyectos
La TUI MUST arrancar en el navegador de proyectos y emitir `GET /projects` al iniciar, sin un menú intermedio. Si la primera carga devuelve exactamente un proyecto, la TUI DEBERÁ abrir directamente el espacio de trabajo de ese proyecto en la pestaña Dashboard. Con cero o más de un proyecto, la TUI DEBERÁ quedarse en el navegador. La apertura automática ocurre solo en la primera carga de la sesión.

#### Scenario: Arranque con varios proyectos
- **GIVEN** que la API devuelve dos o más proyectos
- **WHEN** la TUI arranca
- **THEN** muestra el navegador de proyectos con el título «Proyectos»
- **AND** no muestra un menú de secciones

#### Scenario: Arranque con un único proyecto
- **GIVEN** que la API devuelve exactamente un proyecto
- **WHEN** la TUI termina la primera carga
- **THEN** abre el espacio de trabajo de ese proyecto en la pestaña Dashboard

#### Scenario: Volver al navegador no reabre el proyecto único
- **GIVEN** que la TUI abrió automáticamente el único proyecto
- **WHEN** el usuario vuelve al navegador con `Esc` y la lista se recarga
- **THEN** la TUI permanece en el navegador

### Requirement: Espacio de trabajo del proyecto con pestañas
Elegir un proyecto en el navegador con `Enter` DEBERÁ fijarlo como proyecto de la sesión y abrir su espacio de trabajo. El espacio de trabajo DEBERÁ mostrar, debajo de la barra superior, una línea con el nombre y el estado del proyecto y una barra de pestañas con, en este orden: Dashboard, Backlog, Sprints, Estimación, Esfuerzo y Defectos, resaltando la pestaña activa. La pestaña Estimación es el flujo de Planning Poker. En la pantalla raíz de una pestaña, y sin un filtro en edición, `Tab` y `Shift+Tab` DEBERÁN pasar a la pestaña siguiente y anterior (en forma circular), y las teclas `1` a `6` DEBERÁN abrir la pestaña de esa posición. `Esc` en la pantalla raíz de una pestaña, sin filtro aplicado, DEBERÁ volver al navegador de proyectos. Todas las pestañas DEBERÁN operar sobre el proyecto del espacio de trabajo sin volver a pedirlo. La miga de pan DEBERÁ empezar con «Proyectos», seguir con el nombre del proyecto y la pestaña activa, y terminar en el paso actual del flujo.

#### Scenario: Elegir un proyecto abre el Dashboard
- **GIVEN** que el navegador muestra varios proyectos
- **WHEN** el usuario resalta uno y presiona `Enter`
- **THEN** la TUI abre el espacio de trabajo de ese proyecto en la pestaña Dashboard
- **AND** la barra de pestañas muestra Dashboard, Backlog, Sprints, Estimación, Esfuerzo y Defectos

#### Scenario: Cambiar de pestaña con Tab y con números
- **GIVEN** que la pestaña Dashboard está activa
- **WHEN** el usuario presiona `Tab`
- **THEN** se abre la pestaña Backlog y se carga su lista
- **AND** al presionar `6` se abre la pestaña Defectos
- **AND** al presionar `Shift+Tab` se abre la pestaña Esfuerzo

#### Scenario: Shift+Tab desde la primera pestaña
- **GIVEN** que la pestaña Dashboard está activa
- **WHEN** el usuario presiona `Shift+Tab`
- **THEN** se abre la pestaña Defectos

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

#### Scenario: Cambiar de proyecto
- **GIVEN** que el usuario trabaja en el proyecto A
- **WHEN** vuelve al navegador con `Esc` y elige el proyecto B
- **THEN** el espacio de trabajo muestra el proyecto B y todas las pestañas usan B

### Requirement: Dashboard con las métricas del proyecto y de un Sprint
La pestaña Dashboard DEBERÁ integrar la consulta de métricas: muestra el estado del proyecto y las métricas del ámbito elegido, que por defecto es el proyecto completo. La tecla `s` DEBERÁ abrir un selector con la fila «Proyecto completo» seguida de los Sprints abiertos del proyecto; elegir un Sprint DEBERÁ consultar `GET /projects/{project_id}/sprints/{sprint_id}/metrics` y mostrar sus métricas con las mismas barras, indicando el ámbito en el título del bloque. `Esc` en el selector DEBERÁ volver al Dashboard sin cambiar el ámbito. `d` DEBERÁ descargar el reporte PDF del ámbito mostrado (proyecto o Sprint). `r` DEBERÁ recargar el ámbito mostrado. La TUI MUST NOT ofrecer una sección Métricas separada.

#### Scenario: Ver las métricas de un Sprint en el Dashboard
- **GIVEN** que el Dashboard muestra las métricas del proyecto
- **WHEN** el usuario presiona `s` y elige un Sprint abierto
- **THEN** la TUI consulta las métricas de ese Sprint
- **AND** el Dashboard las muestra bajo el título «Métricas del Sprint» con el objetivo del Sprint

#### Scenario: Volver al proyecto completo
- **GIVEN** que el Dashboard muestra las métricas de un Sprint
- **WHEN** el usuario presiona `s` y elige «Proyecto completo»
- **THEN** el Dashboard vuelve a mostrar las métricas del proyecto

#### Scenario: PDF del ámbito mostrado
- **GIVEN** que el Dashboard muestra las métricas de un Sprint
- **WHEN** el usuario presiona `d`
- **THEN** la TUI descarga el reporte PDF de ese Sprint

### Requirement: Avisos efímeros en lugar de pantallas de resultado
Cuando una operación de escritura termina con éxito (crear o modificar un proyecto, crear o modificar una historia, crear un Sprint, asignar historias, completar una historia, crear una tarea, cerrar un Sprint, registrar esfuerzo, registrar un defecto o cambiar su estado), la TUI DEBERÁ volver a la pantalla desde la que se inició el flujo, recargar su colección y mostrar un aviso de éxito con el prefijo `✓` al inicio del panel. Los destinos son: el navegador de proyectos (recargando `GET /projects`, sin abrir automáticamente un proyecto único) para los proyectos; la lista del backlog para las historias; la lista de Sprints abiertos para crear un Sprint, asignar historias y cerrar un Sprint; las historias del Sprint para completar una historia y crear una tarea; el selector de historias de la pestaña Esfuerzo para el registro de esfuerzo, con un aviso que resume horas, fecha, historia, tarea e integrante; la lista de defectos para registrar un defecto; y el detalle del defecto, actualizado con la respuesta de la API, para el cambio de estado. El aviso DEBERÁ desaparecer con la siguiente tecla, que conserva su efecto normal; no se usan temporizadores. Mientras una escritura está en curso, la pantalla DEBERÁ mostrar un indicador de envío e ignorar las repeticiones del envío. Cuando la API rechaza la operación, la TUI DEBERÁ permanecer en el formulario o la pantalla de confirmación, conservar los valores escritos y mostrar el motivo informado por la API. Una respuesta de escritura que llega sin un envío en curso DEBERÁ descartarse sin cambiar de pantalla. La TUI MUST NOT mostrar pantallas de resultado que exijan una tecla adicional para continuar.

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

#### Scenario: Registrar esfuerzo vuelve a la pestaña Esfuerzo
- **GIVEN** que el formulario de esfuerzo de «Ada Lovelace» sobre la tarea «Diseñar» de la historia «Login» está completo
- **WHEN** la API responde HTTP 201 con `hours` 3.5 y `work_date` 2026-10-05
- **THEN** la TUI vuelve al selector de historias de la pestaña Esfuerzo y lo recarga
- **AND** muestra el aviso «✓ Esfuerzo registrado: 3.5 h el 2026-10-05 · Login · Diseñar · Ada Lovelace»
- **AND** no consulta las métricas del proyecto

#### Scenario: Cambiar el estado de un defecto vuelve a su detalle
- **GIVEN** que el usuario eligió un nuevo estado para un defecto
- **WHEN** la API responde con éxito
- **THEN** la TUI muestra el detalle de ese defecto con el estado devuelto por la API y un aviso que empieza con `✓`

#### Scenario: El envío en curso ignora repeticiones
- **GIVEN** que el usuario confirmó la creación de un Sprint
- **WHEN** presiona `Enter` de nuevo antes de que responda la API
- **THEN** la TUI no emite otra request y muestra «Guardando…»

### Requirement: Pie de teclas acotado y ayuda agrupada
El pie DEBERÁ mostrar como máximo cinco atajos, los principales de la pantalla y el paso activos, y agregar `? más` cuando la pantalla tenga más atajos disponibles y `?` abra la ayuda en ese paso. La ayuda abierta con `?` DEBERÁ listar todos los atajos válidos de la pantalla activa agrupados en «Navegación», «Acciones» y «General», omitiendo los grupos vacíos. Las teclas de cambio de pestaña DEBERÁN figurar en la ayuda de las pantallas raíz del espacio de trabajo.

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
- **THEN** la ayuda lista `Tab/Shift+Tab` y `1–6` en «Navegación»

### Requirement: Atajos contextuales de estimación y esfuerzo
En la lista del backlog, `P` DEBERÁ abrir la pestaña Estimación con la historia resaltada ya elegida, mostrando el selector de participantes, y `E` DEBERÁ abrir la pestaña Esfuerzo con la historia resaltada ya elegida, mostrando el selector de tareas. `Esc` desde ese primer selector DEBERÁ volver a la lista del backlog y recargarla. Los atajos se listan en la ayuda, grupo «Acciones», sin ocupar el pie.

#### Scenario: Estimar desde el backlog
- **GIVEN** que el backlog tiene una historia resaltada
- **WHEN** el usuario presiona `P`
- **THEN** la pestaña Estimación queda activa con esa historia elegida y muestra el selector de participantes

#### Scenario: Registrar esfuerzo desde el backlog
- **GIVEN** que el backlog tiene una historia resaltada
- **WHEN** el usuario presiona `E`
- **THEN** la pestaña Esfuerzo queda activa con esa historia elegida y muestra el selector de tareas

#### Scenario: Volver al backlog desde la historia preelegida
- **GIVEN** que el usuario abrió Estimación o Esfuerzo con `P` o `E`
- **WHEN** presiona `Esc` en el primer selector
- **THEN** la TUI vuelve a la lista del backlog y la recarga

#### Scenario: Backlog vacío
- **GIVEN** que el backlog no tiene historias
- **WHEN** el usuario presiona `P` o `E`
- **THEN** la pantalla no cambia, no se emite ninguna consulta y la ayuda no ofrece esos atajos

#### Scenario: Un paso posterior retrocede de a uno
- **GIVEN** que el usuario abrió Esfuerzo con `E` y eligió una tarea
- **WHEN** presiona `Esc` en el selector de integrantes
- **THEN** la TUI vuelve al selector de tareas de esa historia, no al backlog

#### Scenario: Los atajos solo leen
- **WHEN** el usuario recorre `P` o `E` y vuelve al backlog
- **THEN** la TUI solo emite consultas `GET`

#### Scenario: Las minúsculas siguen buscando
- **GIVEN** que la lista del backlog está activa
- **WHEN** el usuario presiona `p` o `e`
- **THEN** la TUI inicia la búsqueda con ese carácter ya escrito

### Requirement: Salir limpiamente desde el navegador y las pestañas
El usuario DEBERÁ poder salir limpiamente con `Ctrl+C` desde cualquier pantalla y con `q` desde el navegador de proyectos, las listas, los selectores, las confirmaciones y las pantallas raíz de las pestañas. Dentro de un campo de entrada de texto, `q` (y `Q`) DEBERÁ tratarse como un carácter más y NO cerrará la TUI.

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

#### Scenario: `q` en un campo de texto no sale
- **GIVEN** que el foco está en un campo de texto de un formulario o en el buscador
- **WHEN** el usuario presiona `q` o `Q`
- **THEN** la TUI permanece abierta
- **AND** el carácter se agrega al valor del campo

#### Scenario: `Ctrl+C` sale desde cualquier pantalla
- **GIVEN** que cualquier pantalla está activa, incluidos los formularios y los buscadores
- **WHEN** el usuario presiona `Ctrl+C`
- **THEN** la TUI se cierra con código 0 sin traza de error

### Requirement: Retroceder en el flujo de esfuerzo y descartar respuestas tardías
`Esc` DEBERÁ retroceder un nivel por vez en el flujo de Esfuerzo: del formulario al selector de integrantes, del selector de integrantes al de tareas, del selector de tareas al de historias y del selector de historias al navegador de proyectos. Cada retroceso DEBERÁ restaurar el selector anterior tal como estaba al elegir su fila (filas, cursor y filtro), sin emitir requests; en un selector con filtro aplicado, el primer `Esc` limpia el filtro y el siguiente retrocede. Salir del formulario con `Esc` descarta lo escrito. Tras un registro exitoso la TUI vuelve al selector de historias y el siguiente registro empieza con el formulario inicial. Cambiar de proyecto desde el navegador DEBERÁ descartar el estado del flujo (historia, tarea, integrante y formulario) y abrir Esfuerzo en el selector de historias del nuevo proyecto. La TUI DEBERÁ descartar sin efecto toda respuesta que ya no corresponda a la pantalla, el paso o la consulta activos: una respuesta de tareas o de integrantes tras haber retrocedido y un resultado de guardado sin un guardado en curso.

#### Scenario: `Esc` recorre formulario, integrantes, tareas e historias
- **GIVEN** el formulario de esfuerzo abierto desde los selectores de historia, tarea e integrante
- **WHEN** el usuario presiona `Esc` tres veces
- **THEN** la TUI pasa sucesivamente al selector de integrantes, al selector de tareas y al selector de historias
- **AND** no emite ninguna request en esos retrocesos
- **AND** un cuarto `Esc` vuelve al navegador de proyectos

#### Scenario: El filtro del selector de tareas se conserva
- **GIVEN** que el usuario filtró las tareas con `dis` y eligió «Diseñar»
- **WHEN** retrocede desde el formulario al selector de integrantes y de allí al selector de tareas
- **THEN** el selector de tareas muestra el filtro `dis` y el cursor sobre «Diseñar», sin consultar de nuevo
- **AND** el siguiente `Esc` limpia el filtro y recién el que sigue vuelve al selector de historias

#### Scenario: Un segundo registro empieza con el formulario inicial
- **GIVEN** que la TUI volvió al selector de historias tras un registro exitoso
- **WHEN** el usuario elige una historia, otra fila de tareas y un integrante
- **THEN** el formulario se abre con la fecha de hoy y «Actividad» y «Horas» vacías

#### Scenario: Respuesta tardía de integrantes tras retroceder
- **GIVEN** que el usuario eligió una tarea y la consulta de integrantes sigue en curso
- **WHEN** presiona `Esc` y después llega la respuesta de integrantes
- **THEN** la TUI sigue mostrando el selector de tareas sin cambios
- **AND** la respuesta se descarta

#### Scenario: Respuesta tardía de tareas tras retroceder
- **GIVEN** que el usuario eligió una historia y la consulta de tareas sigue en curso
- **WHEN** presiona `Esc` y después llega la respuesta de tareas
- **THEN** la TUI sigue mostrando el selector de historias sin cambios
- **AND** la respuesta se descarta

#### Scenario: Resultado de guardado sin guardado en curso
- **GIVEN** que no hay un `POST` de esfuerzo en curso
- **WHEN** llega un resultado de guardado
- **THEN** la TUI lo descarta sin cambiar de pantalla ni de contenido

#### Scenario: Cambiar el proyecto descarta el flujo
- **GIVEN** que el usuario registró un esfuerzo en el proyecto `p-1`
- **WHEN** vuelve al navegador, elige el proyecto `p-2` y abre la pestaña Esfuerzo
- **THEN** la TUI abre Esfuerzo en el selector de historias de `p-2`
- **AND** la pantalla no conserva el aviso, el registro ni las selecciones de `p-1`
