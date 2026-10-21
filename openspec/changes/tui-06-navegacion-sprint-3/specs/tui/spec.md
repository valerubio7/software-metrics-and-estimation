# Especificación: navegación del Sprint 3 en la TUI

## MODIFIED Requirements

### Requirement: Mostrar menú base de navegación
La TUI DEBERÁ mostrar un menú con las opciones Proyectos, Backlog, Sprints, Planning Poker, Esfuerzo, Defectos, Métricas, Dashboard y Salir. Las opciones Proyectos, Backlog y Sprints conservan su gestión respectiva; las cinco secciones nuevas abren directamente su pantalla principal, sin submenús de operaciones. La TUI DEBERÁ mostrar siempre un encabezado con el nombre del producto y el estado de conexión con la API, y un pie con las teclas disponibles para la pantalla activa.

#### Scenario: Menú inicial
- **WHEN** la TUI arranca correctamente
- **THEN** muestra las nueve opciones y permite moverse entre ellas
- **AND** muestra el encabezado con el estado de conexión y el pie con las teclas de la pantalla

#### Scenario: Backlog abre su gestión desde el menú
- **GIVEN** que el menú está activo con la selección en la primera opción
- **WHEN** el usuario presiona «Abajo» y luego «Enter»
- **THEN** la TUI presenta el flujo de gestión del backlog (el selector de proyectos si no hay proyecto activo, o la lista del backlog si ya lo hay)

#### Scenario: Las secciones nuevas abren directo su pantalla principal
- **GIVEN** que el menú está activo
- **WHEN** el usuario confirma Planning Poker, Esfuerzo, Defectos, Métricas o Dashboard
- **THEN** la TUI abre la pantalla principal de esa sección
- **AND** no muestra un submenú de operaciones

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

### Requirement: Navegación de solo lectura
La navegación entre el menú y las pantallas DEBERÁ ser de solo lectura y NO modificará ningún dato del sistema. Solo las operaciones explícitas de creación y modificación de proyectos (Requirement: Operar proyectos mediante la TUI) y de creación y modificación de historias (Requirement: Crear historias mediante la TUI y Requirement: Modificar historias desde la lista) pueden emitir requests que modifiquen datos. Las secciones Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard, sus selectores y sus pantallas de aviso DEBERÁN emitir únicamente consultas `GET`.

#### Scenario: Navegar no muta
- **WHEN** el usuario navega entre el menú y las pantallas sin confirmar una operación de proyectos o de historias
- **THEN** no se emite ninguna request que modifique datos

#### Scenario: Consultar el backlog no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario abre la lista del backlog o cambia de proyecto
- **THEN** las únicas requests emitidas son las consultas `GET` (del catálogo de proyectos y del backlog) y no se emite ningún `POST` ni `PUT`

#### Scenario: Navegar por las secciones del Sprint 3 no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario recorre Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard, elige filas y avanza entre selectores
- **THEN** todas las requests emitidas son `GET`
- **AND** no se crea, modifica ni elimina ningún dato

## ADDED Requirements

### Requirement: Abrir las secciones del Sprint 3 desde el menú
La TUI DEBERÁ exponer las secciones Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard como destinos directos del menú. Cada sección DEBERÁ abrir su pantalla principal en el primer nivel, sin un submenú de operaciones, y DEBERÁ reutilizar el panel, la barra superior, la miga de pan y el pie con wrapping del rediseño de #98. Si no hay proyecto activo, la sección DEBERÁ abrir primero el navegador de proyectos y, al confirmarlo, continuar en la sección elegida.

#### Scenario: Sección sin proyecto activo
- **GIVEN** que no hay proyecto activo en la sesión
- **WHEN** el usuario confirma una sección nueva del menú
- **THEN** la TUI abre el navegador de proyectos
- **AND** al confirmar un proyecto abre la pantalla principal de esa sección
- **AND** no pide ningún identificador a mano

#### Scenario: Sección con proyecto activo
- **GIVEN** que hay un proyecto activo en la sesión
- **WHEN** el usuario confirma una sección nueva del menú
- **THEN** la TUI abre directamente su pantalla principal y consulta la colección correspondiente
- **AND** el encabezado, la miga y el pie reflejan la sección activa

#### Scenario: API no disponible en una sección
- **GIVEN** que la API no responde al cargar una sección
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el motivo del fallo dentro de la pantalla
- **AND** permanece abierta y permite reintentar con `r` o volver con `Esc`

### Requirement: Elegir integrantes, historias, tareas y Sprints con selectores navegables
La TUI DEBERÁ ofrecer selectores de integrante, historia, tarea y Sprint como listas navegables con el mismo patrón de las colecciones existentes: la búsqueda se activa con `/` o al empezar a escribir, `↑`/`↓` mueven la selección, `Enter` ejecuta la acción primaria, `Esc` limpia el filtro y luego vuelve, `r` refresca y `?` abre la ayuda. Ningún selector DEBERÁ pedir un identificador escrito a mano. Los selectores DEBERÁN mostrar un estado de carga, un estado vacío y el error de la API sin cerrar la TUI.

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
- **GIVEN** que la API responde una colección vacía
- **WHEN** la TUI presenta el selector
- **THEN** muestra un estado vacío claro
- **AND** permite volver o refrescar sin cerrarse

#### Scenario: Los cuatro tipos de selector son navegables
- **GIVEN** que el usuario recorre Planning Poker, Esfuerzo y Defectos
- **WHEN** elige una historia, un integrante, una tarea y un Sprint
- **THEN** cada colección se presenta como una lista navegable con las teclas comunes
- **AND** la selección de una fila no emite requests que modifiquen datos

### Requirement: Avisar el límite de las pantallas de navegación
Las pantallas de navegación de TUI-06 NO implementan los flujos de negocio de Planning Poker, Esfuerzo, Defectos, Métricas ni Dashboard. Cuando el usuario completa la selección de navegación, la TUI DEBERÁ mostrar un aviso que indique en qué historia (TUI-07 a TUI-11) estará disponible el flujo, sin comunicar que la operación se guardó.

#### Scenario: Aviso al completar una selección
- **GIVEN** que el usuario eligió los elementos de navegación de una sección
- **WHEN** la TUI alcanza el final del flujo disponible
- **THEN** muestra un aviso con la historia que implementará el flujo
- **AND** no comunica éxito de una operación de negocio
- **AND** permite volver con `Enter` o `Esc`
