# Tareas: gestionar el backlog desde la TUI (TUI-03)

## Pronóstico de carga de revisión

| Campo | Valor |
|-------|-------|
| Líneas modificadas estimadas | 800 a 950 (pruebas aproximadamente 45 %, producción 40 %, documentación 15 %) |
| Riesgo frente al presupuesto de 400 líneas | Alto |
| PRs encadenados recomendados | No (decisión de la propuesta: un único PR) |
| División sugerida | Un único PR con etiqueta `size:exception` y nueve commits RED/GREEN separados |
| Estrategia de entrega | single-pr |
| Estrategia de encadenado | size-exception |

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Nota: con `single-pr` el orquestador debe exigir `size:exception` antes de aplicar. La propuesta ya registra el acuerdo (un único PR, 800 a 950 líneas); el orquestador debe confirmarlo y dejarlo asentado en `apply-progress.md` antes del commit 2. Mitigación de revisión: commits pequeños y temáticos, RED y GREEN separados, `backlog.go` (transporte) aislado de `backlog_screens.go` (presentación).

### Unidades de trabajo sugeridas

Todas las unidades viajan en el mismo PR; cada una es un límite de reversión independiente dentro del historial (se pueden revertir los commits de una unidad sin tocar las anteriores).

| Unidad | Objetivo | PR | Comando de prueba focalizado | Arnés de ejecución | Límite de reversión |
|--------|----------|----|------------------------------|--------------------|---------------------|
| 1 | Refactor: ayudante HTTP compartido | PR único | `go test ./tests/unit/tui/ -run 'TestProjectAPI|TestProbeAPI'` (verde antes y después) | N/A: refactor interno cubierto por `httptest` | commit 2 (`api.go`, `projects.go`) |
| 2 | Política de teclado: `q` acotada, `KeySpace`, retroceso por runa | PR único | `go test ./tests/unit/tui/ -run 'TestQuitOnQ\|TestCtrlC\|TestSpace\|TestBackspace'` | N/A: caja negra con `Model.Update` y `httptest` | commits 3 y 4 (`model_test.go`, `model.go`) |
| 3 | Cliente HTTP del backlog | PR único | `go test ./tests/unit/tui/ -run 'TestBacklogClient'` | N/A: `httptest` registra método, ruta y cuerpo crudo | commits 5 y 6 (`backlog_api_test.go`, `backlog.go`) |
| 4 | Pantallas del backlog y presentación de errores | PR único | `go test ./tests/unit/tui/ -run 'TestBacklog'` y luego `go test ./...` | N/A: los comandos `tea.Cmd` se ejecutan contra `httptest`; prueba manual opcional con `go run ./cmd/tui` si el binario existe | commits 7 y 8 (`backlog_test.go`, `backlog_screens.go`, `model.go`) |
| 5 | Registro de avance | PR único | `go test ./...` | N/A: solo documentación | commit 9 (`tasks.md`, `apply-progress.md`) |

## Contexto de ejecución

- TDD estricto (`openspec/config.yaml`: `strict_tdd: true`, `test_command: go test ./...`). Cada RED es un commit propio y su salida fallida (resumen de `go test`) se registra en `apply-progress.md` (Fase 7). Un RED válido falla por aserciones, no por compilación.
- Pruebas de caja negra en `tests/unit/tui/` con paquete `tui_test`; solo API exportada (`NewModel`, `Model.Update`, `Model.View`, `Model.Screen`, `NewBacklogClient`, `BacklogResultMsg`, `APIResponse`).
- El espacio se envía como `tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}`; la salida se verifica con `_, ok := cmd().(tea.QuitMsg)`.
- Ayudantes de prueba compartidos (se crean en el primer commit RED que los necesita y se reutilizan): `typeText(m, s)`, `press(m, keyType)`, `openBacklogWithProject(t, url, pid)`. Si `keyMsg` de `model_test.go` ya cubre el caso, se reutiliza.
- Mensajes de commit: Conventional Commits, sin atribución de IA.
- Los caminos con `(solo lectura)` no son objetivos de edición.

## Fase 1: Documentación de la especificación (commit 1)

Commit: `docs(openspec): specify TUI backlog management`
Evidencia: commit `518aadb`.

- [x] 1.1 Verificar que `openspec/changes/tui-03-gestionar-backlog/proposal.md`, `openspec/changes/tui-03-gestionar-backlog/design.md` y `openspec/changes/tui-03-gestionar-backlog/specs/tui/spec.md` incluyen el borrado por runa (requisito «Los campos de texto borran por carácter»), y que `openspec/changes/tui-03-gestionar-backlog/tasks.md` queda completo.
- [x] 1.2 Crear el commit 1 con los cuatro documentos. Sin código ni pruebas. Verificación: `go test ./...` sigue verde (sin cambios de código).

## Fase 2: Refactor del ayudante HTTP compartido (commit 2)

Commit: `refactor(tui): share JSON request helper between API clients`
Evidencia: commit `0168323`.

Requisito de la spec: ninguno nuevo; el refactor no cambia comportamiento (protege «Operar proyectos mediante la TUI» de la TUI-02).

- [x] 2.1 REFACTOR (línea base): ejecutar `go test ./tests/unit/tui/` y registrar que `tests/unit/tui/api_test.go` y el resto de las pruebas de proyectos pasan antes del cambio. Sin prueba nueva: la protección es la suite existente.
- [x] 2.2 REFACTOR: en `internal/tui/api.go` agregar `APIResponse` (movido desde `internal/tui/projects.go`), el tipo no exportado `apiClient{baseURL string; http *http.Client}` con timeout de 5 s, `newAPIClient(baseURL string) apiClient` y `(apiClient) do(ctx, method, path string, input any) (APIResponse, error)` con la lógica de serialización JSON, cabeceras `Content-Type`/`Accept` y lectura del cuerpo en éxito y error. `ProbeAPI` queda sin cambios.
- [x] 2.3 REFACTOR: en `internal/tui/projects.go` hacer que `ProjectClient` contenga `api apiClient` (campo con nombre, no incrustado), delegue en `api.do` y elimine `request` y la definición de `APIResponse`. La API pública de `ProjectClient` no cambia.
- [x] 2.4 Verificar: `gofmt -l internal/ tests/`, `go vet ./internal/tui/...` y `go test ./tests/unit/tui/` verdes con las mismas pruebas que en 2.1. Crear el commit 2 (solo `internal/tui/api.go` y `internal/tui/projects.go`).

## Fase 3: Política de teclado — RED y GREEN (commits 3 y 4)

### Commit 3 — RED

Commit: `test(tui): cover scoped quit, space and rune-aware backspace in forms`
Evidencia: commit `352965c`.

Alcance: `tests/unit/tui/model_test.go`, solo pantallas de proyectos existentes (`create-project`, `edit-project`, `project-id`, `projects`, `project-result`) y el menú. Requisitos de la spec: «Salir limpiamente de la TUI», «Los campos de texto aceptan espacios», «Los campos de texto borran por carácter».

- [x] 3.1 RED: agregar a `tests/unit/tui/model_test.go` `TestQuitOnQDoesNotExitInProjectTextFields`: tabla sobre `project-id` y `create-project`; enviar `q` y `Q` no devuelve `tea.QuitMsg`, la pantalla no cambia y el carácter aparece en el campo (vista). Escenario «`q` en un campo de texto no sale». Falla hoy porque `Model.Update` intercepta `q` antes de despachar.
- [x] 3.2 RED: agregar `TestCtrlCQuitsFromProjectForms`: `tea.KeyCtrlC` devuelve `tea.QuitMsg` desde `project-id`, `create-project` y `edit-project`. Escenario «`Ctrl+C` sale desde cualquier pantalla». Puede pasar ya; se registra como protección de regresión, no como RED.
- [x] 3.3 RED: agregar `TestQuitOnQFromMenuAndProjectLists`: `q` devuelve `tea.QuitMsg` desde `menu`, `projects` y `project-result`. Escenarios «`q` sale desde el menú / una lista / una pantalla de resultado». Protección de regresión (hoy pasa); fija el comportamiento antes de la lista permitida.
- [x] 3.4 RED: agregar `TestSpaceIsAcceptedInProjectTextFields`: `tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}` en `project-id` y `create-project` agrega un espacio visible al valor. Escenario «Espacio en un campo de texto». Falla hoy porque `tea.KeySpace` se descarta.
- [x] 3.5 RED: agregar `TestSpaceInProjectNameIsSentToAPI`: con `httptest` y `tui.NewModel(server.URL)`, escribir «Proyecto Alfa» con `KeySpace`, confirmar y verificar que el cuerpo recibido en `POST /projects` conserva el espacio en `name`. Escenario «Espacio en un formulario de proyectos».
- [x] 3.6 RED: agregar `TestBackspaceRemovesWholeRuneInProjectFields`: tabla sobre `project-id` y `create-project` con «Canción» (un retroceso → «Canció») y «añá» (tres retrocesos → vacío); tras cada pulsación el valor observable en la vista no contiene `�` y `utf8.ValidString` del texto extraído del campo es verdadero. Escenarios «Borrar un carácter acentuado» y «Borrar hasta vaciar un campo con caracteres multibyte». Falla hoy porque borra bytes.
- [x] 3.7 RED: agregar `TestBackspaceOnEmptyFieldIsNoop`: retroceso en un campo vacío no falla ni cambia el valor. Escenario «Retroceso en un campo vacío».
- [x] 3.8 RED: agregar `TestBackspaceThenTypeSendsValidUTF8Name`: con `httptest`, escribir «Gestión añ», retroceso, «o», confirmar la creación del proyecto; el `name` recibido es «Gestión ao» y `utf8.Valid`. Escenario «El valor enviado conserva los caracteres acentuados».
- [x] 3.9 Ejecutar `go test ./tests/unit/tui/` y confirmar RED por aserciones (no por compilación) en 3.1, 3.4, 3.5, 3.6 y 3.8; registrar el resumen fallido y la lista de las pruebas que ya pasan (3.2, 3.3, 3.7 si corresponde). Crear el commit 3 (solo `tests/unit/tui/model_test.go`).

### Commit 4 — GREEN

Commit: `fix(tui): scope q to non-text screens and accept spaces in fields`
Evidencia: commit `c1cd178`.

- [x] 4.1 GREEN: en `internal/tui/model.go` agregar `func quitsOnQ(screen string) bool` con la lista permitida `menu`, `projects`, `project-result`, `backlog`, `backlog-result` y `sprints`; las entradas del backlog quedan listadas aunque las pantallas lleguen en el commit 8 (lista permitida: una pantalla inexistente no causa daño).
- [x] 4.2 GREEN: en `Model.Update`, antes de despachar por pantalla: `tea.KeyCtrlC` → `tea.Quit` en cualquier pantalla; `q`/`Q` (una sola runa) → `tea.Quit` solo si `quitsOnQ(m.Screen)`; en otro caso la tecla llega a la pantalla.
- [x] 4.3 GREEN: agregar `func editText(value string, key tea.KeyMsg) (string, bool)` en `internal/tui/model.go`: `tea.KeyRunes` agrega las runas, `tea.KeySpace` agrega `" "`, `tea.KeyBackspace` elimina la última runa con `utf8.DecodeLastRuneInString` (sin efecto si el valor está vacío); devuelve `true` si la tecla fue consumida.
- [x] 4.4 GREEN: usar `editText` en `project-id`, `create-project` y `edit-project`, reemplazando el manejo previo de `tea.KeyRunes` y `value[:len(value)-1]`.
- [x] 4.5 Verificar: `go test ./tests/unit/tui/` verde (incluidas `TestProjectNavigationPreservesOtherSectionsAndQuit` y `TestQuitOnQAndCtrlC` sin modificar) y registrar la salida GREEN.
- [x] 4.6 REFACTOR: revisar `internal/tui/model.go` para eliminar manejo de teclas duplicado que haya quedado en los formularios de proyectos; `gofmt -l` y `go vet ./internal/tui/...` limpios; repetir `go test ./tests/unit/tui/`. Crear el commit 4 (solo `internal/tui/model.go`).

## Fase 4: Cliente HTTP del backlog — RED y GREEN (commits 5 y 6)

### Commit 5 — RED

Commit: `test(tui): cover backlog API client`
Evidencia: commit `0b5a329`.

Archivos: `tests/unit/tui/backlog_api_test.go` e `internal/tui/backlog.go` con el esqueleto mínimo (tipos de la sección «Interfaces y contratos» del diseño y las tres firmas que devuelven valores cero, sin comportamiento). Requisitos de la spec: «Listar el backlog del proyecto», «Crear historias mediante la TUI», «Modificar historias desde la lista», «Análisis numérico mínimo de `estimated_hours`», «Presentar los errores de la API del backlog».

- [x] 5.1 RED: crear `tests/unit/tui/backlog_api_test.go` con un ayudante que levanta un `httptest.Server`, registra método, ruta (`r.URL.EscapedPath()`) y cuerpo crudo, y responde un estado y cuerpo configurables.
- [x] 5.2 RED: `TestBacklogClientListUsesGetOnProjectStories`: `List(ctx, "p1")` emite `GET /projects/p1/stories` sin cuerpo y devuelve `StatusCode` 200 y el cuerpo de la respuesta.
- [x] 5.3 RED: `TestBacklogClientCreateSendsExactlyFourKeys`: `Create` emite `POST /projects/p1/stories`; el cuerpo decodificado a `map[string]json.RawMessage` tiene exactamente `title`, `description`, `priority`, `acceptance_criteria` y los criterios llegan como lista.
- [x] 5.4 RED: `TestBacklogClientCreateSendsEmptyCriteriaAsList`: con `AcceptanceCriteria: []string{}` el cuerpo contiene `"acceptance_criteria":[]` (ni `null` ni omitida). Escenario «Criterios de aceptación vacíos».
- [x] 5.5 RED: `TestBacklogClientUpdateSendsSixKeysWithNullHours`: `Update(ctx, "p1", "s1", input)` con `EstimatedHours: nil` emite `PUT /projects/p1/stories/s1` con exactamente las seis claves y `estimated_hours` igual al literal `null`. Escenario «`estimated_hours` vacío se envía como `null`».
- [x] 5.6 RED: `TestBacklogClientUpdateSendsHoursNumber`: tabla con 8 y 2.5 y -3 (el rango no se valida localmente); el cuerpo contiene el número tal cual. Escenarios «Valor numérico entero / decimal» y «El rango no se valida localmente».
- [x] 5.7 RED: `TestBacklogClientEscapesPathSegments`: identificadores con `/`, espacios y `?` llegan escapados con `url.PathEscape` en `List`, `Create` y `Update`.
- [x] 5.8 RED: `TestBacklogClientPreservesErrorBodies`: respuestas 422 (con `fields`), 404 JSON y 405 en texto plano devuelven `StatusCode` y `Body` intactos y `err == nil`. Escenarios «HTTP 422…», «HTTP 404…», «HTTP 404 o 405 sin cuerpo JSON».
- [x] 5.9 RED: `TestBacklogClientTransportErrorWhenServerClosed`: con el servidor cerrado, las tres operaciones devuelven `err != nil`. Escenario «Error de transporte».
- [x] 5.10 Ejecutar `go test ./tests/unit/tui/ -run 'TestBacklogClient'` y confirmar RED por aserciones (el esqueleto compila); registrar la salida. Crear el commit 5.

### Commit 6 — GREEN

Commit: `feat(tui): add backlog API client`
Evidencia: commit `af0ccf9`.

- [x] 6.1 GREEN: en `internal/tui/backlog.go` implementar `BacklogClient{api apiClient}`, `NewBacklogClient(baseURL)`, `List` (`GET`), `Create` (`POST`, `StoryCreateInput` con cuatro claves) y `Update` (`PUT`, `StoryUpdateInput` con seis claves y `EstimatedHours *float64` sin `omitempty`), con segmentos escapados con `url.PathEscape`, todo vía `api.do`.
- [x] 6.2 GREEN: completar en `internal/tui/backlog.go` `Story`, `BacklogOp` (`BacklogOpList`, `BacklogOpCreate`, `BacklogOpUpdate`) y `BacklogResultMsg{Op, Response, Err}` según el diseño.
- [x] 6.3 Verificar: `go test ./tests/unit/tui/ -run 'TestBacklogClient'` verde y `go test ./tests/unit/tui/` completo verde; registrar la salida GREEN.
- [x] 6.4 REFACTOR: revisar duplicación entre `BacklogClient` y `ProjectClient` (solo si el ahorro es claro y no cambia la API pública); `gofmt -l` y `go vet ./internal/tui/...` limpios. Crear el commit 6 (solo `internal/tui/backlog.go`).

## Fase 5: Pantallas del backlog — RED y GREEN (commits 7 y 8)

### Commit 7 — RED

Commit: `test(tui): cover backlog screens and error rendering`
Evidencia: commit `a0ae950`.

Archivo: `tests/unit/tui/backlog_test.go`. No se necesita esqueleto: las pruebas solo usan `NewModel`, `Update`, `View`, `Screen`, `BacklogResultMsg` y `NewBacklogClient`, que ya existen. Si algún símbolo no compila, agregar solo la firma mínima sin comportamiento.

- [x] 7.1 RED: crear `tests/unit/tui/backlog_test.go` con los ayudantes `typeText`, `press` y `openBacklogWithProject(t, url, pid)` (menú → «Backlog» → «Elegir proyecto» → escribir pid → Enter, ejecutando el `tea.Cmd` contra el `httptest` y aplicando el mensaje resultante), más un servidor `httptest` que registra cada request (método, ruta, cuerpo).
- [x] 7.2 RED: `TestBacklogMenuOpensBacklogScreen`: «Abajo + Enter» desde el menú deja `Screen == "backlog"` y la vista ya no contiene «TUI-03». Escenarios «La opción Backlog da acceso a la gestión» y «Backlog conserva la pantalla `backlog` desde el menú».
- [x] 7.3 RED: `TestBacklogWithoutProjectOffersChooseProjectAndEmitsNoRequest`: sin proyecto, la vista ofrece «Elegir proyecto» y «Volver», y el servidor no recibe requests. Escenario «Acceder sin proyecto conocido solicita el identificador».
- [x] 7.4 RED: `TestBacklogProjectIDEmptyStaysAndEmitsNoRequest`: Enter con identificador vacío (o solo espacios) permanece en `backlog-project` sin request. Escenario «Identificador vacío».
- [x] 7.5 RED: `TestBacklogProjectSelectionLoadsBacklogInAPIOrder`: confirmar el identificador emite `GET /projects/p1/stories` y la vista muestra «Cargando backlog…» y luego las historias en el orden exacto de la API, con «—» para `story_points` nulo y el número cuando existe. Escenarios «Listar historias en el orden de la API», «`story_points` nulo se muestra como «—»» y «`story_points` informado…» (lista).
- [x] 7.6 RED: `TestBacklogProjectIDIsKeptAcrossMenuReentry`: tras volver al menú y elegir «Backlog», no se vuelve a pedir el identificador y se emite `GET` del proyecto conservado. Escenario «El identificador se solicita una sola vez».
- [x] 7.7 RED: `TestBacklogChangeProjectPrefillsAndReplacesID`: «Cambiar proyecto» abre `backlog-project` con el identificador actual precargado; `Esc` vuelve a `backlog` sin cambiarlo; confirmar otro identificador reemplaza el conservado y las operaciones siguientes usan el nuevo. Escenario «Cambiar el proyecto activo».
- [x] 7.8 RED: `TestBacklogEmptyListShowsMessageAndKeepsActions`: lista vacía muestra «El backlog no tiene historias.» y sigue ofreciendo «Crear historia» y «Cambiar proyecto». Escenario «Backlog vacío».
- [x] 7.9 RED: `TestBacklogCreateStorySendsFourKeysAndShowsSuccess`: «Crear historia» → completar título (con espacios), descripción, prioridad y criterios → Enter en el último campo emite `POST /projects/p1/stories` con exactamente cuatro claves; la vista muestra el éxito (201) y la historia informada. Escenarios «Crear una historia», «Los campos aceptan espacios» y «Resultado exitoso de creación o modificación».
- [x] 7.10 RED: `TestBacklogCreateSplitsCriteriaBySemicolon`: criterios `  Inicia sesión ; Cierra sesión ;; ` → `["Inicia sesión","Cierra sesión"]`; vacío → `[]`. Escenarios «Criterios de aceptación separados por `;`» y «vacíos».
- [x] 7.11 RED: `TestBacklogCreateDoesNotValidatePriorityLocally`: prioridad inválida se envía tal cual. Escenario «La prioridad no se valida localmente».
- [x] 7.12 RED: `TestBacklogUpdateFormIsPrefilledFromSelectedStory`: seleccionar la historia («Login», «alta», «pendiente», `["A","B"]`, horas 5) y Enter abre `backlog-form` mostrando «Login», «alta», «pendiente», «A; B» y «5»; no hay campo editable de `story_points`. Escenarios «El formulario se precarga…» y «`story_points` informado…» (formulario).
- [x] 7.13 RED: `TestBacklogUpdateWithoutChangesSendsSixKeys`: Enter hasta el último campo sin editar emite `PUT /projects/p1/stories/s1` con las seis claves y los valores precargados. Escenario «Modificar sin cambios envía las seis claves».
- [x] 7.14 RED: `TestBacklogUpdateEmptyHoursSendsNull`: vaciar horas con retroceso envía `estimated_hours: null` sin omitir la clave. Escenario «`estimated_hours` vacío se envía como `null`».
- [x] 7.15 RED: `TestBacklogUpdateChangingTitleKeepsRest`: cambiar solo el título conserva el resto. Escenario «Modificar un campo conserva el resto».
- [x] 7.16 RED: `TestBacklogUpdateHoursNumericParsing`: `8` → 8, `2.5` → 2.5, `-3` se envía tal cual. Escenarios «Valor numérico entero / decimal» y «El rango no se valida localmente».
- [x] 7.17 RED: `TestBacklogNonNumericHoursShowsErrorWithoutRequest`: `abc` muestra «Horas estimadas debe ser un número», no emite request y conserva lo escrito en el formulario; también `NaN` e `Inf`. Escenario «Valor no numérico».
- [x] 7.18 RED: `TestBacklogSelectingStoryWithEmptyListOpensNoForm`: con lista vacía no existe fila de historia que abrir; Enter sobre las acciones no abre un formulario de modificación y no emite request. Escenario «Modificar con la lista vacía».
- [x] 7.19 RED: `TestBacklogFormEscReturnsToBacklogWithoutRequest`: `Esc` en `backlog-form` vuelve a `backlog` sin enviar.
- [x] 7.20 RED: `TestBacklogRendersValidationErrorsAsList`: `BacklogResultMsg{Op: Create, Response: 422 con fields de dos campos}` muestra «La API rechazó los datos (422)» y una línea `- campo: mensaje` por campo ordenada por clave, sin comunicar éxito. Escenario «HTTP 422 con campos inválidos».
- [x] 7.21 RED: `TestBacklogRendersNotFoundWithGenericMessageAndAPIDetail`: 404 JSON (`project_not_found`, `story_not_found`) muestra el mensaje genérico «El recurso solicitado no existe (404)» más `Detalle de la API: <error>: <message>`. Escenario «HTTP 404 proyecto o historia inexistente».
- [x] 7.22 RED: `TestBacklogRendersNonJSONErrorWithStatusCode`: 404 y 405 en texto plano muestran `Error HTTP <código>` con el cuerpo recortado, o «sin detalle» si está vacío, sin cerrarse. Escenario «HTTP 404 o 405 sin cuerpo JSON».
- [x] 7.23 RED: `TestBacklogListFailureLeadsToProjectCorrection`: 422 al listar muestra `fields` como lista y Enter/Esc lleva a `backlog-project` con el identificador precargado. Escenario «HTTP 422 al consultar la lista por identificador inválido».
- [x] 7.24 RED: `TestBacklogTransportErrorDoesNotQuit`: `BacklogResultMsg{Err: …}` para las tres operaciones muestra «Error de conexión» y no devuelve `tea.QuitMsg`. Escenario «Error de transporte».
- [x] 7.25 RED: `TestBacklogResultReturnDestinations`: tabla de la decisión «Retorno desde `backlog-result`»: éxito de `Create`/`Update` → `backlog` con `GET` de recarga; fallo de `Create`/`Update` → `backlog-form` con los valores conservados; fallo de `List` → `backlog-project`.
- [x] 7.26 RED: `TestBacklogLateListResultIsDiscarded`: un `BacklogResultMsg{Op: List}` recibido en el menú (o fuera de carga) no cambia la pantalla; `Create`/`Update` se aplican siempre y llevan a `backlog-result`.
- [x] 7.27 RED: `TestBacklogBrowsingDoesNotMutate`: abrir la lista y cambiar de proyecto solo emite `GET`, nunca `POST` ni `PUT`. Escenarios «Consultar el backlog no muta» y «Navegar no muta».
- [x] 7.28 RED: `TestQuitOnQIsScopedInBacklogScreens`: tabla; `q`/`Q` en `backlog-project` y `backlog-form` no sale y se escribe en el campo; `q` sale desde `backlog` y `backlog-result`. Escenarios «`q` sale desde una lista / una pantalla de resultado» y «`q` en un campo de texto no sale».
- [x] 7.29 RED: `TestCtrlCQuitsFromEveryBacklogScreen`: `Ctrl+C` devuelve `tea.QuitMsg` en `backlog`, `backlog-project`, `backlog-form` y `backlog-result`. Escenario «`Ctrl+C` sale desde cualquier pantalla».
- [x] 7.30 RED: `TestSpaceAndBackspaceByRuneInBacklogFields`: espacio visible en `backlog-project` y `backlog-form`; «Canción» → «Canció» y «añá» vacío tras tres retrocesos con valor UTF-8 válido; el título «Gestión añ» + retroceso + «o» se envía como «Gestión ao» en el `POST`. Escenarios de los requisitos «aceptan espacios» y «borran por carácter».
- [x] 7.31 Ejecutar `go test ./tests/unit/tui/ -run 'TestBacklog|TestQuitOnQIsScoped|TestCtrlCQuitsFromEveryBacklogScreen|TestSpaceAndBackspace'` y confirmar RED por aserciones; registrar la salida fallida y las pruebas que ya pasan. Crear el commit 7 (solo `tests/unit/tui/backlog_test.go`, más el esqueleto mínimo si fue necesario).

### Commit 8 — GREEN

Commit: `feat(tui): manage the product backlog from the terminal UI`
Evidencia: commit `ceb4ba8`.

- [x] 8.1 GREEN: en `internal/tui/backlog_screens.go` implementar `splitCriteria` (divide por `;`, recorta, descarta vacíos, devuelve slice no nulo) y `parseEstimatedHours` (vacío → `nil`; `strconv.ParseFloat`; rechaza no numéricos y no finitos). Con error local, el formulario muestra «Horas estimadas debe ser un número» sin emitir request.
- [x] 8.2 GREEN: implementar `storyForm` (alta con cuatro campos, modificación con seis), la precarga (criterios unidos con `"; "`, horas con `strconv.FormatFloat(v, 'f', -1, 64)` o vacío) y la navegación `Tab`/`↓`, `Shift+Tab`/`↑`, `Enter` avanza y envía en el último campo, `Esc` vuelve a `backlog`.
- [x] 8.3 GREEN: implementar `updateBacklog` (acciones primero, luego historias en el orden de la API; sin proyecto solo «Elegir proyecto» y «Volver»), `updateBacklogProject` (`backlogInput` separado de `backlogProjectID`, sin validar formato, `Esc` sin modificar el confirmado) y las vistas (`Backlog del proyecto <pid>`, fila `<título> · prioridad <p> · estado <s> · puntos <n|—>`, «Cargando backlog…», «El backlog no tiene historias.»).
- [x] 8.4 GREEN: implementar `listStoriesCmd`, `createStoryCmd`, `updateStoryCmd` (comandos que llaman a `BacklogClient` y devuelven `BacklogResultMsg{Op, …}`) y `applyBacklogResult` (`List` solo si `Screen == "backlog"` y `backlogLoading`; `Create`/`Update` siempre a `backlog-result`), con `backlogReturn` según la tabla del diseño.
- [x] 8.5 GREEN: implementar `describeBacklogFailure(APIResponse) string` (422 con `fields` ordenados por clave, 404 genérico más detalle, otros códigos `Error HTTP <código>: <detalle>`, cuerpo no JSON recortado o «sin detalle») y `Error de conexión: <err>` para transporte.
- [x] 8.6 GREEN: implementar `updateBacklogResult` (Enter/Esc hacia `backlogReturn`, recarga con `List` tras éxito) y usar `editText` en `backlog-project` y `backlog-form`.
- [x] 8.7 GREEN: en `internal/tui/model.go` agregar los campos de estado del backlog y `backlogClient` (`NewModel` lo construye con la misma URL que `client`), despachar las cuatro pantallas nuevas, hacer que el menú lleve siempre a `backlog` y emita `List` si hay proyecto, y eliminar el caso `TUI-03` de `placeholderRelease`.
- [x] 8.8 Verificar: `go test ./tests/unit/tui/` y `go test ./...` verdes, incluyendo las pruebas existentes (`TestProjectNavigationPreservesOtherSectionsAndQuit`, `TestQuitOnQAndCtrlC`, la prueba «Abajo + Enter → `backlog`»); registrar la salida GREEN.
- [x] 8.9 REFACTOR: revisar `internal/tui/backlog_screens.go` y `internal/tui/model.go` (nombres, duplicación entre `updateBacklogForm` y `updateProjectForm` solo si es claro y seguro, funciones largas); `gofmt -l`, `go vet ./...` y `go test ./...` verdes. Crear el commit 8 (`internal/tui/backlog_screens.go`, `internal/tui/model.go`).

## Fase 6: Cierre y registro de avance (commit 9)

Commit: `docs(openspec): record TUI-03 apply progress`
Evidencia: commit 9 (este commit).

- [x] 9.1 Crear `openspec/changes/tui-03-gestionar-backlog/apply-progress.md` con, por commit, el hash, la salida RED fallida (commits 3, 5, 7), la salida GREEN (commits 4, 6, 8), el resultado de `go test ./...` y la confirmación de `size:exception`.
- [x] 9.2 Marcar las tareas completadas en `openspec/changes/tui-03-gestionar-backlog/tasks.md` con el hash del commit como evidencia y registrar la línea final de entrega: diff acumulado frente al presupuesto de 400 líneas, con `size:exception` para el PR único.
- [x] 9.3 Verificación final: `go test ./...` verde, `gofmt -l .` sin salida, `go vet ./...` limpio. Crear el commit 9 (solo `tasks.md` y `apply-progress.md`).

## Orden y dependencias

Secuencial estricto por commit: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9. No hay tareas paralelizables: un único escritor, y cada GREEN depende de su RED. Dependencias duras: el commit 2 precede a 6 (el cliente usa `apiClient`); el commit 4 precede a 8 (`editText` y `quitsOnQ`); el commit 6 precede a 7 y 8 (las pruebas de pantallas usan `BacklogClient` y `BacklogResultMsg`).

## Trazabilidad con la spec

| Requisito de la spec | Pruebas |
|----------------------|---------|
| Operar el backlog mediante la TUI | 7.2, 7.3, 7.9 |
| Identificador de proyecto único por sesión | 7.3, 7.4, 7.6, 7.7 |
| Listar el backlog del proyecto | 5.2, 7.5, 7.8, 7.12 |
| Crear historias mediante la TUI | 5.3, 5.4, 7.9, 7.10, 7.11 |
| Modificar historias desde la lista | 5.5, 7.12 a 7.15, 7.18, 7.19 |
| Análisis numérico mínimo de `estimated_hours` | 5.6, 7.16, 7.17 |
| Presentar los errores de la API del backlog | 5.8, 5.9, 7.20 a 7.26 |
| Los campos de texto aceptan espacios | 3.4, 3.5, 7.9, 7.30 |
| Los campos de texto borran por carácter | 3.6, 3.7, 3.8, 7.30 |
| MODIFIED: Mostrar menú base de navegación | 7.2 |
| MODIFIED: Salir limpiamente de la TUI | 3.1 a 3.3, 7.28, 7.29 |
| MODIFIED: Navegación de solo lectura | 7.27 |

## Cierre de entrega

- Estrategia: `single-pr` con `size:exception` (aceptado por el usuario antes de aplicar; ver `apply-progress.md`).
- Diff acumulado de los commits 1 a 8 frente a `main`: 3194 líneas añadidas y 68 eliminadas (3262 en total): producción `internal/tui/` 716 (648+68), pruebas `tests/unit/tui/` 1643, documentación de la especificación `openspec/changes/tui-03-gestionar-backlog/` 903. El commit 9 suma solo `tasks.md` y `apply-progress.md`.
- Frente al presupuesto de 400 líneas y al pronóstico de 800 a 950: el pronóstico se superó de forma material (aproximadamente 3,4 veces). Las causas principales son las pruebas de pantalla de caja negra (1067 líneas en `backlog_test.go`, 30 pruebas con tablas por escenario de la spec) y el tamaño de `backlog_screens.go` (438 líneas). No se recortaron pruebas, comentarios ni documentación para ajustar el número.
- Etiqueta requerida en el PR: `size:exception`. Los nueve commits son límites de reversión independientes.
- Nota: las tareas con salida RED/GREEN y los hallazgos están en `apply-progress.md`.

## Tarea 7 — estilo visual del backlog (commits 10 a 12)

Refinamiento visual solicitado por el usuario tras la prueba manual de la TUI. No hay cambio de especificación: el comportamiento, el manejo de teclas y los textos no cambian (`openspec/specs/` no se toca). Las pantallas del backlog adoptan los estilos `lipgloss` de `internal/tui/styles.go` que ya usa el menú principal.

Commits: `fcf9e0e` (RED), `281018e` (GREEN) y el commit de documentación (este).

- [x] 7.1 RED: crear `tests/unit/tui/backlog_style_test.go` (8 pruebas) que fuerza `termenv.TrueColor` (restaurado con `t.Cleanup`) y verifica ANSI en título, fila seleccionada (fondo de `selectedStyle`), filas sin seleccionar (sin fondo), mensajes de carga/vacío, pista de teclas, banner de error (borde) y estilo de éxito, y que el texto sin ANSI conserva el cuerpo plano actual. `termenv` pasa de indirecta a directa en `go.mod`. Evidencia: commit `fcf9e0e`; las 8 pruebas nuevas fallan por falta de ANSI y los 62 tests previos siguen verdes.
- [x] 7.2 GREEN: en `internal/tui/backlog_screens.go`, títulos con `screenTitleStyle`; fila seleccionada `cursorStyle("▸") + selectedStyle`, no seleccionada `"  " + itemStyle`; pistas con `hintKeyStyle` en las teclas y `hintStyle` en el resto; carga y vacío en `hintStyle`; errores del formulario y de la API en `errorBannerStyle`; éxito en `statusOkStyle`. Se elimina `cursorPrefix`. Evidencia: commit `281018e`.
- [x] 7.3 Verificar: `go test ./tests/unit/tui/` con 70 pruebas de nivel superior en verde (62 previas sin modificar + 8 nuevas) y `go test` de los paquetes no de integración en verde.
