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

## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: Mostrar el esfuerzo registrado y las métricas del proyecto
**Reason**: la pantalla de resultado del esfuerzo se reemplaza por un aviso efímero y las métricas del proyecto viven en el Dashboard; la TUI ya no consulta `GET /projects/{pid}/metrics` después de registrar.
**Migration**: Requirement: Avisos efímeros en lugar de pantallas de resultado (escenario «Registrar esfuerzo vuelve a la pestaña Esfuerzo»), Requirement: Enviar el registro de esfuerzo y presentar los errores de la API y Requirement: Dashboard con las métricas del proyecto y de un Sprint.

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

### Requirement: Salir limpiamente de la TUI
**Reason**: el menú y las pantallas de resultado ya no existen, de modo que los escenarios «`q` sale desde el menú» y «`q` sale desde una pantalla de resultado» no tienen pantalla a la que aplicarse.
**Migration**: Requirement: Salir limpiamente desde el navegador y las pestañas, que conserva `Ctrl+C`, `q` en las listas y `q` como carácter en los campos de texto, y agrega `q` desde el navegador de proyectos.

### Requirement: Recorrer hacia atrás el flujo de esfuerzo y descartar respuestas tardías
**Reason**: la pantalla de resultado del esfuerzo y la consulta de métricas posterior al registro se eliminaron; ya no hay un `Esc` desde el resultado ni respuestas de métricas que descartar, y el selector de historias vuelve al navegador de proyectos en lugar del menú.
**Migration**: Requirement: Retroceder en el flujo de esfuerzo y descartar respuestas tardías (retrocesos, filtros conservados, cambio de proyecto y respuestas tardías de tareas, integrantes y guardado) y Requirement: Avisos efímeros en lugar de pantallas de resultado (escenario «Registrar esfuerzo vuelve a la pestaña Esfuerzo»).
