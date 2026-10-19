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

### Requirement: Mostrar menú base de navegación
La TUI DEBERÁ mostrar un menú con las opciones proyectos, backlog, sprints y salir. Las opciones proyectos, backlog y sprints dan acceso a su gestión respectiva. La TUI DEBERÁ mostrar siempre un encabezado con el nombre del producto y el estado de conexión con la API, y un pie con las teclas disponibles para la pantalla activa.

#### Scenario: Menú inicial
- **WHEN** la TUI arranca correctamente
- **THEN** muestra las 4 opciones y permite moverse entre ellas
- **AND** muestra el encabezado con el estado de conexión y el pie con las teclas de la pantalla

#### Scenario: Backlog abre su gestión desde el menú
- **GIVEN** que el menú está activo con la selección en la primera opción
- **WHEN** el usuario presiona «Abajo» y luego «Enter»
- **THEN** la TUI presenta el flujo de gestión del backlog (el selector de proyectos si no hay proyecto activo, o la lista del backlog si ya lo hay)

### Requirement: Presentar el contenido en un panel con jerarquía visual
La TUI DEBERÁ mostrar el contenido de cada pantalla dentro de un panel con borde que ocupa el ancho disponible, encabezado por una barra superior sólida con la marca y el estado de conexión a la API, seguida de la miga de pan (breadcrumb). La fila enfocada DEBERÁ resaltarse como una barra que ocupa todo el ancho del panel, no solo el texto. Cuando el ancho no alcanza para la marca, la barra superior DEBERÁ priorizar el indicador de conexión y mantener el contenido dentro del ancho disponible sin desbordar.

#### Scenario: Selección como barra completa
- **WHEN** una lista o formulario muestra una fila enfocada
- **THEN** la fila enfocada se resalta con una barra que ocupa todo el ancho del panel

#### Scenario: Ancho reducido
- **WHEN** la terminal es más angosta que la marca más el indicador de conexión
- **THEN** la barra superior conserva el indicador de conexión y el resto del contenido se mantiene dentro del ancho disponible

### Requirement: Mostrar la ayuda de teclado
La TUI DEBERÁ ofrecer una ayuda de teclado accesible con `?` desde las pantallas que no son de entrada de texto, y cerrarla con `Esc`, `Enter` o `?`. Dentro de un campo de texto `?` se trata como un carácter más.

#### Scenario: Abrir y cerrar la ayuda
- **GIVEN** que una pantalla que no es de entrada de texto está activa
- **WHEN** el usuario presiona `?`
- **THEN** la TUI muestra la ayuda con las teclas principales
- **AND** al presionar `Esc` la ayuda se cierra y se vuelve a la pantalla previa

#### Scenario: `?` se escribe en un campo de texto
- **GIVEN** que el foco está en un campo de texto
- **WHEN** el usuario presiona `?`
- **THEN** el carácter se agrega al campo y la ayuda no se abre

### Requirement: Informar API no disponible sin cierre abrupto

Si la API no está disponible la TUI DEBERÁ informar el motivo y seguir abierta.

#### Scenario: API caída

- **WHEN** la API no responde al abrir o navegar
- **THEN** la TUI muestra el motivo del fallo y no se cierra abruptamente

### Requirement: Salir limpiamente de la TUI
El usuario DEBERÁ poder salir limpiamente con `Ctrl+C` desde cualquier pantalla, con `q` únicamente desde el menú, las listas y las pantallas de resultado, o con la opción salir del menú. Dentro de un campo de entrada de texto, `q` (y `Q`) DEBERÁ tratarse como un carácter más y NO cerrará la TUI.
(Previously: `q` / `Ctrl+C` / opción salir cerraban la TUI sin distinguir la pantalla, por lo que `q` también cerraba la TUI al escribir en un formulario.)

#### Scenario: Salida limpia
- **WHEN** el usuario pide salir
- **THEN** la TUI termina con código 0 sin traza de error

#### Scenario: `q` sale desde el menú
- **GIVEN** que el menú está activo
- **WHEN** el usuario presiona `q`
- **THEN** la TUI se cierra con código 0

#### Scenario: `q` sale desde una lista
- **GIVEN** que una lista seleccionable está activa (el navegador de proyectos o el backlog)
- **WHEN** el usuario presiona `q`
- **THEN** la TUI se cierra con código 0

#### Scenario: `q` sale desde una pantalla de resultado
- **GIVEN** que una pantalla de resultado está activa (proyectos o backlog)
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

### Requirement: Navegación de solo lectura
La navegación entre el menú y las pantallas DEBERÁ ser de solo lectura y NO modificará ningún dato del sistema. Solo las operaciones explícitas de creación y modificación de proyectos (Requirement: Operar proyectos mediante la TUI) y de creación y modificación de historias (Requirement: Crear historias mediante la TUI y Requirement: Modificar historias desde la lista) pueden emitir requests que modifiquen datos.
(Previously: solo las operaciones explícitas de creación y modificación de proyectos podían emitir requests que modifiquen datos.)

#### Scenario: Navegar no muta
- **WHEN** el usuario navega entre el menú y las pantallas sin confirmar una operación de proyectos o de historias
- **THEN** no se emite ninguna request que modifique datos

#### Scenario: Consultar el backlog no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario abre la lista del backlog o cambia de proyecto
- **THEN** las únicas requests emitidas son las consultas `GET` (del catálogo de proyectos y del backlog) y no se emite ningún `POST` ni `PUT`
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
La TUI DEBERÁ obtener el catálogo de proyectos con `GET /projects` y presentarlo en un navegador buscable, de modo que el usuario NO necesite recordar ni escribir identificadores. El navegador se abre desde la opción «Proyectos» del menú y también sirve de selector a Backlog y Sprints. DEBERÁ permitir filtrar por nombre o identificador (la búsqueda se activa con `/` y cualquier otro carácter la inicia ya escrito), navegar con las teclas de flecha, fijar el proyecto activo con `Enter`, abrir la creación con `n`, abrir la modificación del proyecto resaltado con `e`, y volver con `Esc` sin cambiar el proyecto. Cada fila DEBERÁ mostrar el nombre, el estado derivado con una etiqueta en español (`planificado`, `en curso` o `atrasado`), una forma corta del identificador, y `✓ activo` en el proyecto de la sesión. La palabra «activo» queda reservada al proyecto de la sesión; el estado `active` de la API se muestra como «en curso». Si el catálogo está vacío o la búsqueda no arroja coincidencias, la TUI DEBERÁ informarlo sin cerrarse.

#### Scenario: Abrir el navegador desde Proyectos
- **GIVEN** que el menú está activo en la opción «Proyectos»
- **WHEN** el usuario la confirma
- **THEN** la TUI emite `GET /projects` y muestra el navegador
- **AND** cada fila muestra el nombre, el estado y la forma corta del identificador

#### Scenario: Abrir el selector desde el backlog
- **GIVEN** que no hay proyecto activo
- **WHEN** el usuario confirma «Backlog»
- **THEN** la TUI emite `GET /projects` y muestra el selector
- **AND** cada fila muestra el nombre, el estado y la forma corta del identificador

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

#### Scenario: Volver sin cambiar de proyecto
- **GIVEN** que el selector está abierto sobre un proyecto ya activo
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI vuelve a la pantalla previa conservando el proyecto activo

#### Scenario: Catálogo vacío
- **GIVEN** que la API responde `GET /projects` con una lista vacía
- **WHEN** la TUI presenta el resultado
- **THEN** muestra un mensaje que indica que todavía no hay proyectos
- **AND** permanece abierta

#### Scenario: Búsqueda sin coincidencias
- **GIVEN** que el usuario escribe un filtro que no coincide con ningún proyecto
- **WHEN** la TUI presenta la lista filtrada
- **THEN** muestra un mensaje de «sin coincidencias»
- **AND** el usuario puede borrar el filtro

### Requirement: Precargar la modificación de proyecto y resolver el resultado
El formulario de modificación DEBERÁ abrirse precargado con el nombre, la fecha de inicio y la fecha fin planificada del proyecto resaltado en el navegador, para no reescribirlos. Tras una creación o modificación exitosa, la TUI DEBERÁ volver al navegador recargando el catálogo. Ante un rechazo de la API, la TUI DEBERÁ volver al formulario conservando los valores escritos y mostrando el motivo informado por la API.

#### Scenario: Formulario de modificación precargado
- **GIVEN** que el navegador muestra un proyecto con nombre y fechas
- **WHEN** el usuario abre su modificación con `e`
- **THEN** el formulario muestra el nombre y las dos fechas ya cargados en sus segmentos

#### Scenario: Vuelta al navegador tras el éxito
- **GIVEN** que una creación o modificación responde con éxito
- **WHEN** el usuario cierra la pantalla de resultado
- **THEN** la TUI vuelve al navegador y recarga `GET /projects`

#### Scenario: Vuelta al formulario tras el rechazo
- **GIVEN** que la API rechaza los datos con un error
- **WHEN** el usuario cierra la pantalla de resultado
- **THEN** la TUI vuelve al formulario con los valores escritos y muestra el motivo informado por la API

### Requirement: Proyecto activo único por sesión
La TUI DEBERÁ fijar el proyecto elegido en el selector como proyecto activo de la sesión, conservarlo para todas las operaciones del backlog y los sprints, y ofrecer la tecla `c` (desde las listas) y el atajo `p` (global) para reemplazarlo. La TUI NO validará localmente el identificador; la validación queda en la API.

#### Scenario: El proyecto se elige una sola vez
- **GIVEN** que el usuario ya eligió un proyecto en la sesión
- **WHEN** vuelve al menú, elige nuevamente «Backlog» y accede a la gestión
- **THEN** la TUI no vuelve a abrir el selector
- **AND** consulta el backlog del proyecto conservado

#### Scenario: Cambiar el proyecto activo
- **GIVEN** que el usuario está en la lista del backlog con un proyecto activo
- **WHEN** presiona `c` y confirma otro proyecto en el selector
- **THEN** la TUI reemplaza el proyecto activo por el nuevo
- **AND** las operaciones siguientes usan el nuevo proyecto

#### Scenario: El proyecto activo se refleja en la navegación
- **GIVEN** que hay un proyecto activo
- **WHEN** el usuario navega entre el menú y las pantallas del backlog
- **THEN** el encabezado de ruta y el menú muestran el nombre del proyecto activo

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
La TUI DEBERÁ permitir crear una historia con título, descripción, prioridad y criterios de aceptación, y enviarla al endpoint HTTP de creación del proyecto activo. La prioridad se ingresa como texto libre y la TUI NO validará sus valores permitidos. Los criterios de aceptación se ingresan en una sola línea separada por `;`; la TUI DEBERÁ recortar los espacios de cada elemento, descartar los elementos vacíos y enviarlos como lista. El cuerpo enviado DEBERÁ contener solo las claves `title`, `description`, `priority` y `acceptance_criteria`.

#### Scenario: Crear una historia
- **GIVEN** que el usuario completa título, descripción, prioridad y criterios de aceptación
- **WHEN** confirma la creación
- **THEN** la TUI envía `POST /projects/{pid}/stories` con `title`, `description`, `priority` y `acceptance_criteria`
- **AND** presenta el resultado de la API en la pantalla de resultado

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
La TUI DEBERÁ presentar el resultado de cada operación del backlog sin cerrarse. Ante HTTP 422 DEBERÁ mostrar el detalle de `fields` como una lista formateada. Ante HTTP 404 DEBERÁ mostrar un mensaje genérico de recurso inexistente más el detalle informado por la API. Las respuestas de error que no sean JSON (por ejemplo, HTTP 404 o 405 del enrutador cuando las rutas de historias no están registradas) DEBERÁN presentarse como error de la API con el código de estado, sin cerrar la TUI. Los errores de transporte DEBERÁN informar el problema de conexión sin cerrar la TUI. Nunca se comunicará éxito ante un error.

#### Scenario: HTTP 422 con campos inválidos
- **GIVEN** que la API responde HTTP 422 con un `fields` que detalla al menos dos campos inválidos
- **WHEN** la TUI presenta el resultado
- **THEN** muestra cada campo y su motivo como un elemento de lista formateada
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
- **THEN** muestra el detalle de `fields` como lista formateada
- **AND** permite volver al selector para cambiar de proyecto

#### Scenario: Error de transporte
- **GIVEN** que no se puede establecer conexión con la API
- **WHEN** falla una operación del backlog (consulta, creación o modificación)
- **THEN** la TUI informa el problema de conexión sin cerrarse abruptamente

#### Scenario: Resultado exitoso de creación o modificación
- **GIVEN** que la API responde HTTP 201 (creación) o HTTP 200 (modificación) con la historia
- **WHEN** la TUI presenta el resultado
- **THEN** comunica el éxito mostrando la historia informada por la API
- **AND** permite volver a la lista y al menú

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

Las pantallas que muestran colecciones (proyectos, historias y Sprints) DEBERÁN comportarse como un navegador con el mismo patrón de interacción: la búsqueda se activa con `/` y cualquier otro carácter la inicia ya escrito; `↑`/`↓` mueven la selección; `Enter` ejecuta la acción primaria de la fila resaltada; `n` inicia la creación; `e` modifica el elemento resaltado cuando la pantalla lo admite; `r` refresca el listado; `x` ejecuta una acción destructiva sobre la fila resaltada (pidiendo confirmación); `h` abre una vista relacionada de solo lectura; y `Esc` primero limpia el filtro y luego vuelve a la pantalla anterior. Cada fila DEBERÁ mostrar el contenido, sus etiquetas y una forma corta del identificador, con un marcador inicial cuando corresponda (por ejemplo `✓` en el proyecto activo de la sesión). Ninguna lista DEBERÁ pedir un identificador escrito a mano. Las pantallas de formulario y de resultado NO adoptan este patrón.

#### Scenario: Buscar en cualquier colección
- **GIVEN** que una colección está visible (proyectos, backlog, Sprints abiertos, historias de un Sprint o Sprints cerrados)
- **WHEN** el usuario presiona `/` o empieza a escribir
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

### Requirement: Operar los Sprints mediante la TUI

La TUI DEBERÁ permitir operar los Sprints del proyecto activo desde la opción «Sprints», que abre directamente el navegador de Sprints abiertos en lugar de un menú de operaciones. Desde el navegador el usuario abre las historias de un Sprint (`Enter`), asigna historias del backlog al Sprint resaltado (`a`), cierra el Sprint resaltado (`x`, con confirmación), crea un Sprint (`n`) y consulta los Sprints cerrados con `h`. Las pantallas de asignación (multiselección con `Space`), de historias del Sprint, de creación de tareas y de resultado reutilizan el patrón navegable (Requirement: Listas navegables con teclas uniformes) y consumen la API existente sin incorporar reglas de negocio.

#### Scenario: La opción Sprints abre el navegador
- **GIVEN** que hay un proyecto activo
- **WHEN** el usuario confirma la opción «Sprints»
- **THEN** la TUI emite `GET /projects/{pid}/sprints/open` y presenta el navegador de Sprints abiertos
- **AND** no pide ningún identificador a mano

#### Scenario: Sin proyecto activo abre el selector
- **GIVEN** que en la sesión aún no se eligió ningún proyecto
- **WHEN** el usuario confirma la opción «Sprints»
- **THEN** la TUI presenta el selector de proyectos con el catálogo de la API
- **AND** no consulta los Sprints hasta que el usuario confirme un proyecto

#### Scenario: Asignar historias al Sprint resaltado
- **GIVEN** que el navegador de Sprints muestra un Sprint abierto resaltado
- **WHEN** el usuario presiona `a`, marca historias con `Space` y confirma con `Enter`
- **THEN** la TUI envía `POST /projects/{pid}/sprints/{sid}/stories` con los identificadores marcados
- **AND** presenta el resultado de la API sin cerrarse

#### Scenario: Cerrar un Sprint pide confirmación
- **GIVEN** que el navegador de Sprints muestra un Sprint abierto resaltado
- **WHEN** el usuario presiona `x`
- **THEN** la TUI pide confirmación antes de emitir `POST /projects/{pid}/sprints/{sid}/closure`
- **AND** si el usuario cancela no se emite ningún request

#### Scenario: Crear tareas y completar historias desde el Sprint
- **GIVEN** que el usuario abrió las historias de un Sprint abierto
- **WHEN** resalta una historia y presiona `Enter` o `c`
- **THEN** `Enter` abre el formulario de tareas de esa historia
- **AND** `c` envía `POST /projects/{pid}/sprints/{sid}/stories/{story_id}/completion`

#### Scenario: Consultar los Sprints cerrados
- **GIVEN** que el navegador de Sprints está activo
- **WHEN** el usuario presiona `h`
- **THEN** la TUI emite `GET /projects/{pid}/sprints` y presenta los Sprints cerrados con su progreso
- **AND** `Enter` despliega las historias del Sprint resaltado sin emitir requests

#### Scenario: Backlog vacío en la asignación
- **GIVEN** que el proyecto activo no tiene historias en el backlog
- **WHEN** el usuario intenta asignar historias a un Sprint
- **THEN** la TUI lo informa y no emite ningún request de asignación

