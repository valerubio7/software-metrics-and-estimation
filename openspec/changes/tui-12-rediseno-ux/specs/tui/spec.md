# Delta para tui: rediseño de la experiencia de usuario (TUI-12)

Este cambio reorganiza la navegación de la TUI alrededor del proyecto: la TUI arranca en el navegador de proyectos, elegir un proyecto abre su espacio de trabajo con pestañas y el Dashboard es la pestaña inicial. La sección Métricas se integra en el Dashboard, las pantallas de resultado se reemplazan por avisos efímeros y el pie de teclas se acota. No cambia ningún contrato de la API.

## ADDED Requirements

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
Cuando una operación de escritura termina con éxito (crear o modificar un proyecto, crear o modificar una historia, crear un Sprint, asignar historias, completar una historia, crear una tarea, cerrar un Sprint, registrar esfuerzo, registrar un defecto o cambiar su estado), la TUI DEBERÁ volver a la pantalla desde la que se inició el flujo, recargar su colección y mostrar un aviso de éxito con el prefijo `✓` en el panel. El aviso DEBERÁ desaparecer con la siguiente tecla. Cuando la API rechaza la operación, la TUI DEBERÁ permanecer en el formulario o la pantalla de confirmación, conservar los valores escritos y mostrar el motivo informado por la API. La TUI MUST NOT mostrar pantallas de resultado que exijan una tecla adicional para continuar.

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

### Requirement: Pie de teclas acotado y ayuda agrupada
El pie DEBERÁ mostrar como máximo cinco atajos, los principales de la pantalla y el paso activos, y agregar `? más` cuando la pantalla tenga más atajos disponibles. La ayuda abierta con `?` DEBERÁ listar todos los atajos válidos de la pantalla activa agrupados en «Navegación», «Acciones» y «General», omitiendo los grupos vacíos. Las teclas de cambio de pestaña DEBERÁN figurar en la ayuda de las pantallas raíz del espacio de trabajo.

#### Scenario: Pie de la lista de Sprints
- **GIVEN** que la pestaña Sprints muestra su lista
- **WHEN** la TUI dibuja el pie
- **THEN** muestra como máximo cinco atajos y `? más`

#### Scenario: La ayuda lista todos los atajos agrupados
- **GIVEN** que la pestaña Sprints muestra su lista
- **WHEN** el usuario presiona `?`
- **THEN** la ayuda muestra los grupos «Navegación», «Acciones» y «General»
- **AND** incluye los atajos de cerrar Sprint, ver los cerrados y descargar el PDF

### Requirement: Atajos contextuales de estimación y esfuerzo
En la lista del backlog, `P` DEBERÁ abrir la pestaña Estimación con la historia resaltada ya elegida, mostrando el selector de participantes, y `E` DEBERÁ abrir la pestaña Esfuerzo con la historia resaltada ya elegida, mostrando el selector de tareas. `Esc` desde ese primer selector DEBERÁ volver a la lista de historias de la pestaña abierta. Los atajos solo emiten consultas `GET`.

#### Scenario: Estimar desde el backlog
- **GIVEN** que el backlog tiene una historia resaltada
- **WHEN** el usuario presiona `P`
- **THEN** la pestaña Estimación queda activa con esa historia elegida y muestra el selector de participantes

#### Scenario: Registrar esfuerzo desde el backlog
- **GIVEN** que el backlog tiene una historia resaltada
- **WHEN** el usuario presiona `E`
- **THEN** la pestaña Esfuerzo queda activa con esa historia elegida y muestra el selector de tareas

## MODIFIED Requirements

### Requirement: Listar y elegir proyectos con un buscador
La TUI DEBERÁ obtener el catálogo de proyectos con `GET /projects` y presentarlo en un navegador buscable, de modo que el usuario NO necesite recordar ni escribir identificadores. El navegador es la pantalla inicial de la TUI y el punto desde el que se entra al espacio de trabajo de un proyecto. DEBERÁ permitir filtrar por nombre o identificador (la búsqueda se activa con `/` y cualquier otro carácter la inicia ya escrito), navegar con las teclas de flecha, abrir el proyecto resaltado con `Enter`, abrir la creación con `n`, abrir la modificación del proyecto resaltado con `e` y refrescar con `r`. `Esc` limpia el filtro aplicado; sin filtro no hace nada. Cada fila DEBERÁ mostrar el nombre, el estado derivado con una etiqueta en español (`planificado`, `en curso` o `atrasado`) y una forma corta del identificador, y marcar con `✓` el último proyecto abierto en la sesión. El estado `active` de la API se muestra como «en curso». Si el catálogo está vacío o la búsqueda no arroja coincidencias, la TUI DEBERÁ informarlo sin cerrarse.
(Previously: el navegador se abría desde la opción «Proyectos» del menú, servía de selector a las demás secciones y `Esc` volvía a la pantalla previa.)

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

#### Scenario: Permanecer en el formulario tras el rechazo
- **GIVEN** que la API rechaza los datos con un error
- **WHEN** la TUI procesa la respuesta
- **THEN** permanece en el formulario con los valores escritos y muestra el motivo informado por la API

### Requirement: Salir limpiamente de la TUI
El usuario DEBERÁ poder salir limpiamente con `Ctrl+C` desde cualquier pantalla y con `q` desde el navegador de proyectos, las listas y las pantallas raíz de las pestañas. Dentro de un campo de entrada de texto, `q` (y `Q`) DEBERÁ tratarse como un carácter más y NO cerrará la TUI.
(Previously: `q` también salía desde el menú y desde las pantallas de resultado, y existía la opción Salir del menú.)

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

## REMOVED Requirements

### Requirement: Mostrar menú base de navegación
**Reason**: el menú plano de nueve opciones se reemplaza por el navegador de proyectos inicial y el espacio de trabajo con pestañas.
**Migration**: Requirement: Iniciar en el navegador de proyectos y Requirement: Espacio de trabajo del proyecto con pestañas. El encabezado con el estado de conexión y el pie con teclas se conservan en Requirement: Presentar el contenido en un panel con jerarquía visual y Requirement: Pie de teclas acotado y ayuda agrupada.

### Requirement: Proyecto activo único por sesión
**Reason**: el proyecto deja de ser un contexto global que se cambia con `c` o `p`; es el espacio de trabajo abierto desde el navegador.
**Migration**: Requirement: Espacio de trabajo del proyecto con pestañas (escenario «Cambiar de proyecto»).

### Requirement: Abrir las secciones del Sprint 3 desde el menú
**Reason**: las secciones son pestañas del espacio de trabajo y siempre tienen un proyecto; ya no se abre el navegador a mitad del flujo.
**Migration**: Requirement: Espacio de trabajo del proyecto con pestañas. El manejo de errores de la API dentro de la pantalla se conserva en Requirement: Elegir integrantes, historias, tareas y Sprints con selectores navegables.

### Requirement: Abrir el Dashboard desde el menú
**Reason**: el Dashboard es la pestaña inicial del espacio de trabajo e integra la sección Métricas.
**Migration**: Requirement: Espacio de trabajo del proyecto con pestañas y Requirement: Dashboard con las métricas del proyecto y de un Sprint.
