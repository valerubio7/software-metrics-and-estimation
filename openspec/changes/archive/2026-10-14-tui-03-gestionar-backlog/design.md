# Diseño: gestión del backlog en la TUI

## Contexto

La TUI usa Bubble Tea (`github.com/charmbracelet/bubbletea` v1.3.10). La opción «Backlog» del menú lleva a la pantalla `backlog`, que hoy es un marcador de posición (`placeholderRelease` devuelve `TUI-03`). La TUI-02 dejó el patrón vigente: `internal/tui/projects.go` contiene un `ProjectClient` con un método privado `request` que serializa JSON, ejecuta la petición y devuelve `APIResponse{StatusCode, Body}`; `internal/tui/model.go` enruta por `Model.Screen` y recibe `ProjectResultMsg` de forma asíncrona.

El backend ya expone los contratos de historias (verificados en `internal/story/transport/http`):

| Operación | Ruta | Cuerpo | Éxito | Errores |
|---|---|---|---|---|
| Consultar backlog | `GET /projects/{pid}/stories` | — | 200 `{project_id, stories[]}` ordenado por la API | 422 (UUID inválido), 404 `project_not_found` |
| Crear historia | `POST /projects/{pid}/stories` | `title`, `description`, `priority`, `acceptance_criteria[]` (JSON estricto: clave desconocida → 400) | 201 historia | 422 `validation_failed` con `fields`, 404 `project_not_found` |
| Modificar historia | `PUT /projects/{pid}/stories/{sid}` | las seis claves `title`, `description`, `priority`, `status`, `acceptance_criteria`, `estimated_hours` (clave obligatoria, `null` válido) | 200 historia | 422 con `fields`, 404 `story_not_found` |

La historia se serializa como `{id, project_id, title, description, priority, status, story_points|null, acceptance_criteria[], estimated_hours|null}` y los errores como `{error, message, fields?}` con `fields` de tipo `map[string]string`. Las rutas dependen de migraciones aplicadas; si faltan, el mux responde 404/405 en texto plano.

Defectos heredados de la TUI-02 que este cambio corrige:

- `Model.Update` intercepta `q`/`Q` como salida **antes** de despachar a la pantalla, por lo que escribir `q` en un formulario cierra la TUI.
- Los formularios solo manejan `tea.KeyRunes`; Bubble Tea v1.3.10 emite `tea.KeySpace` para la barra espaciadora, que se descarta.
- El borrado (`value[:len(value)-1]`) recorta bytes, no runas: borrar después de «á» o «ñ» deja UTF-8 inválido.

## Enfoque

- La TUI sigue siendo un cliente HTTP sin reglas de negocio: no valida prioridad, estado ni criterios; solo hace el análisis numérico mínimo de `estimated_hours` y el armado de la lista de criterios.
- Se extrae el método `request` de `ProjectClient` a un ayudante compartido no exportado en `internal/tui/api.go`. `ProjectClient` y el nuevo `BacklogClient` lo usan sin duplicar código; la API pública de `ProjectClient` no cambia.
- Nuevo `BacklogClient` (`List`, `Create`, `Update`) y mensaje asíncrono `BacklogResultMsg` etiquetado con `Op`, siguiendo el patrón de `ProjectResultMsg`.
- Cuatro pantallas nuevas o reemplazadas: `backlog` (centro con acciones y lista seleccionable), `backlog-project` (identificador del proyecto), `backlog-form` (alta y modificación con precarga) y `backlog-result`.
- La política de teclado se centraliza en `model.go`: `Ctrl+C` sale siempre; `q`/`Q` sale solo en pantallas de menú, lista y resultado; un único ayudante de edición de texto acepta runas, `tea.KeySpace` y borrado por runa, y lo usan todos los campos (proyectos y backlog).

## Decisiones de arquitectura

### Decisión: ayudante HTTP compartido en lugar de copiar `request`

**Elección**: mover `APIResponse` y la lógica de `ProjectClient.request` a `internal/tui/api.go` como un tipo no exportado `apiClient` con `newAPIClient(baseURL)` y `do(ctx, method, path, input)`. `ProjectClient` y `BacklogClient` lo contienen como campo con nombre (`api apiClient`), no por incrustación.
**Alternativas consideradas**: copiar `request` en `backlog.go` (duplica serialización, cabeceras, timeout y lectura del cuerpo); un único `APIClient` con todos los endpoints (crece como objeto dios y contradice el `BacklogClient` de la propuesta); una función libre `doJSON(ctx, *http.Client, baseURL, …)` (obliga a cada cliente a repetir la construcción de `http.Client` y el recorte de la URL base).
**Justificación**: un solo lugar para timeout (5 s), cabeceras `Content-Type`/`Accept` y lectura del cuerpo en éxito y error. El campo con nombre evita promover métodos y mantiene el acoplamiento explícito. Es un refactor sin cambio de comportamiento protegido por `tests/unit/tui/api_test.go` existente.

### Decisión: `BacklogResultMsg` único etiquetado con `Op`

**Elección**: `BacklogResultMsg{Op BacklogOp, Response APIResponse, Err error}` con `BacklogOp` exportado y constantes `BacklogOpList`, `BacklogOpCreate`, `BacklogOpUpdate`.
**Alternativas consideradas**: un tipo de mensaje por operación (tres tipos casi idénticos y tres ramas en `Update`); reutilizar `ProjectResultMsg` (mezcla flujos y destinos de pantalla).
**Justificación**: un único punto de entrada en `Update` y un único `applyBacklogResult` que decide por `Op`. Al ser exportado, las pruebas de caja negra inyectan resultados sin servidor.

### Decisión: aplicación de resultados según operación

**Elección**:
- `List` se aplica solo si `Screen == "backlog"` y `backlogLoading` es verdadero; si no, se descarta (resultado tardío tras `Esc` o tras cambiar de proyecto).
- `Create` y `Update` se aplican siempre y llevan a `backlog-result`, porque representan una mutación ya ocurrida en el servidor que el usuario debe ver.
**Alternativas consideradas**: agregar `ProjectID` al mensaje para descartar respuestas obsoletas (más estado y más campos en las pruebas inyectadas); aplicar todo incondicionalmente (un listado tardío arrastraría al usuario fuera del menú).
**Justificación**: la guarda por pantalla y bandera de carga cubre la carrera real con el mínimo de estado. Al cambiar de proyecto se vacía `backlogStories` y se vuelve a cargar, de modo que solo la respuesta en curso queda aplicable; una respuesta anterior que llegue durante la nueva carga sí se aplicaría (riesgo residual, documentado abajo).

### Decisión: pantalla `backlog` como centro con acciones y lista

**Elección**: el menú lleva siempre a `backlog` (se conserva la prueba «Abajo + Enter → `backlog`»). La pantalla presenta primero las acciones y luego las historias:
- Sin proyecto en sesión: acciones «Elegir proyecto» y «Volver».
- Con proyecto: acciones «Crear historia», «Cambiar proyecto», «Volver», seguidas de las historias en el orden de la API. `Enter` sobre una historia abre la modificación precargada.
Al entrar desde el menú con un proyecto ya conservado, se emite `List` y se muestra «Cargando backlog…».
**Alternativas consideradas**: pedir el identificador directamente en la pantalla `backlog` (mezcla un campo de texto con una lista y entra en conflicto con `q`); historias primero y acciones al final (la posición de «Crear historia» dependería del tamaño del backlog).
**Justificación**: el cursor 0 es estable (primera acción), la navegación es previsible y coincide con la pantalla de proyectos. Las pruebas pueden navegar por teclas sin conocer la cantidad de historias.

### Decisión: identificador de proyecto conservado en la sesión

**Elección**: `backlogProjectID` (confirmado) y `backlogInput` (texto en edición) separados. `Enter` en `backlog-project` con texto recortado no vacío confirma el identificador, vacía la lista, activa la carga y emite `List`. «Cambiar proyecto» abre `backlog-project` con el identificador actual precargado. `Esc` en `backlog-project` vuelve a `backlog` sin modificar el identificador confirmado. El valor persiste al volver al menú; no se persiste en disco.
**Alternativas consideradas**: reutilizar `projectID` de la TUI-02 (acopla los flujos de proyectos y backlog); persistir en archivo de configuración (fuera de alcance).
**Justificación**: la propuesta pide solicitarlo una vez por sesión; separar entrada y valor confirmado evita que una edición cancelada altere el proyecto activo.

### Decisión: formulario con estado propio y precarga

**Elección**: tipo no exportado `storyForm` con campos de texto con nombre y el campo activo:

| Modo | Campos en orden |
|---|---|
| Alta (`storyID == ""`) | Título, Descripción, Prioridad, Criterios de aceptación (separados por `;`) |
| Modificación | Título, Descripción, Prioridad, Estado, Criterios de aceptación, Horas estimadas |

La precarga copia la historia seleccionada: criterios unidos con `"; "`, horas con `strconv.FormatFloat(v, 'f', -1, 64)` o vacío si es `null`. `story_points` no se edita ni se envía. Navegación igual a la TUI-02: `Tab`/`↓` y `Shift+Tab`/`↑` cambian de campo, `Enter` avanza y en el último campo envía, `Esc` vuelve a `backlog` sin enviar.
**Alternativas consideradas**: reutilizar `ProjectInput`/`formField` (otro conjunto de campos y modos); componentes `bubbles/textinput` (nueva dependencia y cambio de estilo frente a la TUI-02).
**Justificación**: mismo patrón de punteros a campos de `updateProjectForm`, sin dependencias nuevas y con un único lugar que construye el cuerpo.

### Decisión: armado de cuerpos sin reglas de negocio

**Elección**:
- `splitCriteria(s)`: divide por `;`, recorta espacios, descarta elementos vacíos y devuelve siempre un slice no nulo (`[]` y no `null`).
- `parseEstimatedHours(s)`: texto recortado vacío → `nil` (se envía `null`); si no, `strconv.ParseFloat`; se rechazan localmente los valores no numéricos y los no finitos (`NaN`, `Inf`), que `encoding/json` no puede serializar. Los negativos o el cero se envían tal cual: los valida el backend.
- Error local de horas: el formulario muestra «Horas estimadas debe ser un número» y no emite petición.
- `StoryCreateInput` tiene exactamente cuatro claves (el `POST` rechaza claves desconocidas con 400). `StoryUpdateInput` tiene seis claves y `EstimatedHours *float64` **sin** `omitempty`, para que `null` viaje explícitamente.
**Alternativas consideradas**: validar prioridad/estado contra listas locales (duplica reglas de dominio); omitir `estimated_hours` cuando está vacío (el `PUT` de reemplazo total lo rechaza); aceptar coma decimal (ambigüedad regional; el backend recibe números JSON).
**Justificación**: cumple el contrato del reemplazo total y deja la autoridad de validación en el servidor.

### Decisión: política de teclado centralizada

**Elección**: en `Model.Update`, antes de despachar por pantalla:
1. `tea.KeyCtrlC` → `tea.Quit` en cualquier pantalla.
2. `q`/`Q` (una sola runa) → `tea.Quit` solo si `quitsOnQ(m.Screen)`; lista permitida: `menu`, `projects`, `project-result`, `backlog`, `backlog-result` y la pantalla de marcador `sprints`.
3. En el resto, la tecla llega a la pantalla, donde `q` es un carácter.

Un ayudante `editText(value string, key tea.KeyMsg) (string, bool)` maneja `tea.KeyRunes` (agrega las runas), `tea.KeySpace` (agrega `" "`) y `tea.KeyBackspace` (elimina la última runa). Lo usan `project-id`, `create-project`, `edit-project`, `backlog-project` y `backlog-form`.
**Alternativas consideradas**: lista de pantallas de texto donde `q` no sale (una pantalla de texto nueva olvidada reproduciría el defecto y perdería datos); manejar `q` en cada pantalla (dispersa la política).
**Justificación**: la lista permitida falla de forma segura: si se olvida una pantalla, `q` no sale pero `Ctrl+C` sigue funcionando y no se pierde texto. El borrado por runa se incluye porque está en el mismo ayudante y evita enviar UTF-8 inválido (títulos con tildes); tiene su propia prueba RED.

### Decisión: presentación de errores del backlog

**Elección**: función `describeBacklogFailure(APIResponse) string` en `backlog_screens.go`, aplicada a respuestas fuera de 2xx:
- Se intenta decodificar `{error, message, fields}`; si falla (por ejemplo, 404 en texto plano del mux), el detalle es el cuerpo recortado, o «sin detalle» si está vacío.
- 422: `La API rechazó los datos (422): <message>` seguido de una línea por campo, ordenada por clave: `- <campo>: <mensaje>`.
- 404: `El recurso solicitado no existe (404). Detalle de la API: <error>: <message>` (genérico, porque puede ser `project_not_found`, `story_not_found` o el mux).
- Otros códigos: `Error HTTP <código>: <detalle>`.
- Error de transporte (`Err != nil`): `Error de conexión: <err>`; la TUI no se cierra.
La presentación de proyectos de la TUI-02 no cambia.
**Alternativas consideradas**: unificar con `applyProjectResult` (cambia textos que verifican pruebas existentes y amplía el alcance); mostrar el JSON crudo (la propuesta exige la lista formateada de `fields`).
**Justificación**: orden determinista para pruebas y lectura, sin acoplar la regresión de proyectos.

### Decisión: retorno desde `backlog-result`

**Elección**: el campo `backlogReturn` define el destino de `Enter`/`Esc`:

| Resultado | Destino |
|---|---|
| `Create`/`Update` exitoso | `backlog`, con recarga (`List`) |
| `Create`/`Update` fallido (422, 404, otro, transporte) | `backlog-form` con los valores escritos conservados |
| `List` fallido | `backlog-project` con el identificador precargado para corregirlo |

**Alternativas consideradas**: volver siempre a `backlog` (pierde lo escrito ante un 422 y obliga a reescribir).
**Justificación**: permite corregir un 422 sin reescribir el formulario; un 404 de proyecto lleva directo a corregir el identificador.

### Decisión: distribución de archivos

**Elección**: `internal/tui/backlog.go` contiene solo el cliente y los tipos de transporte (como `projects.go`); `internal/tui/backlog_screens.go` contiene los métodos de `Model` para las cuatro pantallas, los comandos y la vista. `model.go` queda como enrutador y aloja la política de teclado y `editText`.
**Alternativas consideradas**: todo en `backlog.go` (archivo de más de 400 líneas que mezcla HTTP y UI); todo en `model.go` (ya tiene 335 líneas).
**Justificación**: separa transporte de presentación para la revisión del PR con `size:exception`. Es un refinamiento del «`backlog.go` aislado» de la propuesta, sin cambio de alcance.

## Flujo de datos

Máquina de estados de pantallas del backlog:

```
menu ──Enter «Backlog»──▶ backlog ◀──────────────────────────────┐
                           │  (si hay proyecto: List, cargando)    │
        Esc ◀──────────────┤                                       │
                           ├─«Elegir/Cambiar proyecto»─▶ backlog-project
                           │                              │ Enter(id) → List, cargando
                           │                              │ Esc → backlog
                           ├─«Crear historia»──▶ backlog-form (alta, vacío)
                           └─Enter en historia─▶ backlog-form (modificación, precargado)
                                                   │ Esc → backlog
                                                   │ Enter en último campo → Create/Update
                                                   ▼
BacklogResultMsg{Op} ──▶ applyBacklogResult
   List ok (si cargando en backlog) ─▶ backlog con historias
   List fallido ───────────────────▶ backlog-result ─Enter/Esc▶ backlog-project
   Create/Update ok ───────────────▶ backlog-result ─Enter/Esc▶ backlog + List
   Create/Update fallido ──────────▶ backlog-result ─Enter/Esc▶ backlog-form
```

Camino de una petición:

```
tecla ─▶ Model.Update ─▶ updateBacklogForm ─▶ updateStoryCmd(pid, sid, input)
                                                   │ tea.Cmd (función)
                                                   ▼
                       BacklogClient.Update ─▶ apiClient.do ─▶ PUT /projects/{pid}/stories/{sid}
                                                   │
                       BacklogResultMsg{Op: BacklogOpUpdate, Response|Err}
                                                   ▼
                       Model.Update ─▶ applyBacklogResult ─▶ backlog-result
```

Los segmentos `{pid}` y `{sid}` se escapan con `url.PathEscape`, igual que en `ProjectClient`.

Vista de la lista (`backlog`): encabezado `Backlog del proyecto <pid>`, acciones y luego una fila por historia con `<título> · prioridad <p> · estado <s> · puntos <n|—>`. `story_points` nulo se muestra como «—». Backlog vacío: «El backlog no tiene historias.».

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/tui/api.go` | Modificar | Agrega `APIResponse` (movido desde `projects.go`), `apiClient`, `newAPIClient` y `do`. `ProbeAPI` sin cambios. |
| `internal/tui/projects.go` | Modificar | `ProjectClient` delega en `apiClient`; se elimina `request`. API pública sin cambios. |
| `internal/tui/backlog.go` | Crear | `Story`, `StoryCreateInput`, `StoryUpdateInput`, `BacklogClient`, `NewBacklogClient`, `List`, `Create`, `Update`, `BacklogOp` y `BacklogResultMsg`. |
| `internal/tui/backlog_screens.go` | Crear | `storyForm`, `updateBacklog`, `updateBacklogProject`, `updateBacklogForm`, `updateBacklogResult`, comandos `listStoriesCmd`/`createStoryCmd`/`updateStoryCmd`, `applyBacklogResult`, `describeBacklogFailure`, `splitCriteria`, `parseEstimatedHours` y las vistas. |
| `internal/tui/model.go` | Modificar | Campos de estado del backlog y `backlogClient`; política de `Ctrl+C`/`q` con `quitsOnQ`; `editText` usado por los formularios de proyectos; despacho de las pantallas nuevas; el menú lleva a `backlog` y recarga si hay proyecto; se elimina el caso `TUI-03` de `placeholderRelease`. |
| `internal/tui/styles.go` | Sin cambios previstos | Solo si la vista necesita un estilo nuevo; se reutilizan los existentes. |
| `tests/unit/tui/model_test.go` | Modificar | Pruebas de teclado: `q` y espacio en formularios, `Ctrl+C` en formularios, `q` en menú/listas/resultados, borrado por runa. |
| `tests/unit/tui/backlog_api_test.go` | Crear | Pruebas del `BacklogClient` con `httptest`. |
| `tests/unit/tui/backlog_test.go` | Crear | Pruebas de flujo de pantallas por teclas, comandos ejecutados contra `httptest` y mensajes inyectados. |

Sin cambios en endpoints, backend, persistencia ni `openspec/specs/tui/spec.md` (la spec principal solo cambia al archivar).

## Interfaces y contratos

```go
// internal/tui/api.go
type APIResponse struct {
	StatusCode int
	Body       []byte
}

type apiClient struct {
	baseURL string
	http    *http.Client // Timeout: 5 * time.Second
}

func newAPIClient(baseURL string) apiClient
// do serializa input como JSON cuando no es nil, fija Content-Type/Accept,
// lee el cuerpo en éxito y error, y devuelve error solo ante fallos de transporte o serialización.
func (c apiClient) do(ctx context.Context, method, path string, input any) (APIResponse, error)
```

```go
// internal/tui/backlog.go
type Story struct {
	ID                 string   `json:"id"`
	ProjectID          string   `json:"project_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	StoryPoints        *int     `json:"story_points"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"`
}

type StoryCreateInput struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
}

type StoryUpdateInput struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"` // sin omitempty: null explícito
}

type BacklogClient struct{ api apiClient }

func NewBacklogClient(baseURL string) *BacklogClient
func (c *BacklogClient) List(ctx context.Context, projectID string) (APIResponse, error)
func (c *BacklogClient) Create(ctx context.Context, projectID string, input StoryCreateInput) (APIResponse, error)
func (c *BacklogClient) Update(ctx context.Context, projectID, storyID string, input StoryUpdateInput) (APIResponse, error)

type BacklogOp string

const (
	BacklogOpList   BacklogOp = "list"
	BacklogOpCreate BacklogOp = "create"
	BacklogOpUpdate BacklogOp = "update"
)

type BacklogResultMsg struct {
	Op       BacklogOp
	Response APIResponse
	Err      error
}
```

```go
// internal/tui/model.go (estado agregado, no exportado)
backlogProjectID string   // proyecto confirmado en la sesión
backlogInput     string   // texto en backlog-project
backlogCursor    int
backlogStories   []Story
backlogLoading   bool
backlogForm      storyForm
backlogMessage   string   // texto de backlog-result
backlogReturn    string   // destino al salir de backlog-result
backlogClient    *BacklogClient

func quitsOnQ(screen string) bool
func editText(value string, key tea.KeyMsg) (string, bool)
```

`NewModel(apiURL)` construye `backlogClient` con la misma URL que `client`; no se introduce una interfaz de cliente.

## Estrategia de pruebas

TDD estricto (`openspec/config.yaml`: `strict_tdd: true`, `go test ./...`). Todas las pruebas están en `tests/unit/tui/` con paquete `tui_test` (caja negra): solo usan la API exportada (`NewModel`, `Model.Update`, `Model.View`, `Model.Screen`, `NewBacklogClient`, `BacklogResultMsg`, `APIResponse`). No se agrega interfaz de cliente ni se exporta estado interno.

Técnicas:
- **Servidor `httptest`**: registra método, ruta y cuerpo crudo; decodifica el cuerpo a `map[string]json.RawMessage` para verificar el conjunto exacto de claves y el `null` literal de `estimated_hours`.
- **Comandos ejecutados**: el modelo se crea con `tui.NewModel(server.URL)`; el `tea.Cmd` devuelto por `Update` es una función síncrona, así que la prueba lo invoca y pasa el mensaje resultante a `Update`. Esto recorre teclas → comando → HTTP → mensaje → vista sin dobles de prueba.
- **Mensajes inyectados**: para 422, 404 y transporte se construye `tui.BacklogResultMsg{Op: …, Response: …}` o `{Err: …}` y se aplica sobre un modelo llevado por teclas a la pantalla correspondiente.
- **Teclas reales**: el espacio se envía como `tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}`, como lo emite Bubble Tea v1.3.10; la salida se verifica con `_, ok := cmd().(tea.QuitMsg)`.
- Ayudantes de prueba: `typeText(m, s)` (runas y `KeySpace`), `press(m, keyType)`, `openBacklogWithProject(t, url, pid)`.

| Capa | Qué se prueba | Enfoque |
|---|---|---|
| Unitaria: cliente | `List` → `GET /projects/p1/stories`; `Create` → `POST` con exactamente cuatro claves y criterios como lista; `Update` → `PUT /projects/p1/stories/s1` con exactamente seis claves y `estimated_hours: null`; escape de segmentos; cuerpos 422/404 preservados; error de transporte con servidor cerrado | `backlog_api_test.go` con `httptest` |
| Unitaria: refactor | `ProjectClient` mantiene métodos, rutas, cuerpos y errores | `api_test.go` existente sin cambios, verde antes y después |
| Unitaria: teclado | `q` en `create-project` y `project-id` no sale y aparece en el campo; espacio aparece en el campo; `Ctrl+C` sale desde un formulario; `q` sale desde `menu` y `projects`; borrar tras «á» deja el texto válido. Los mismos casos para `backlog-project`, `backlog-form`, `backlog` y `backlog-result` van en `backlog_test.go`, porque esas pantallas aparecen en el paso 8 | `model_test.go` |
| Unitaria: flujo backlog | `q` y espacio en `backlog-project` y `backlog-form` se escriben; `q` sale desde `backlog` y `backlog-result`; menú → `backlog`; elegir proyecto; identificador conservado tras volver al menú (reentrada emite `List`); «Cambiar proyecto» precargado; lista en orden de la API con «—» para puntos nulos; alta envía cuatro claves y muestra éxito; modificación precarga y envía seis claves; horas no numéricas no emiten petición; 422 con lista `- campo: mensaje`; 404 con «no existe» y el código de la API; transporte sin salir; retorno de `backlog-result` según la tabla; listado tardío descartado | `backlog_test.go` |
| Regresión | `TestProjectNavigationPreservesOtherSectionsAndQuit`, `TestQuitOnQAndCtrlC` y las pruebas de proyectos siguen pasando | Suite existente sin cambios |
| Integración / E2E | No aplica: sin cambios de backend | `go test ./...` completo al cierre |

### Orden de commits RED/GREEN

La TUI-02 no pudo demostrar RED porque pruebas y producción quedaron en el mismo commit (`apply-progress.md` archivado). En este cambio cada RED es un commit propio y su salida fallida se registra en `apply-progress.md`:

| # | Commit | Contenido | Estado esperado |
|---|---|---|---|
| 1 | `docs(openspec): specify TUI backlog management` | propuesta, spec delta, diseño y tareas | — |
| 2 | `refactor(tui): share JSON request helper between API clients` | `api.go`, `projects.go` | Verde antes y después (`api_test.go`) |
| 3 | `test(tui): cover scoped quit, space and rune-aware backspace in forms` | `model_test.go` (solo pantallas de proyectos existentes) | RED por aserciones |
| 4 | `fix(tui): scope q to non-text screens and accept spaces in fields` | `model.go` | GREEN |
| 5 | `test(tui): cover backlog API client` | `backlog_api_test.go` y esqueleto mínimo de `backlog.go` (firmas que devuelven valores cero) | RED por aserciones, no por compilación |
| 6 | `feat(tui): add backlog API client` | `backlog.go` | GREEN |
| 7 | `test(tui): cover backlog screens and error rendering` | `backlog_test.go` y esqueleto mínimo si hace falta para compilar | RED por aserciones |
| 8 | `feat(tui): manage the product backlog from the terminal UI` | `backlog_screens.go`, `model.go` | GREEN; `go test ./...` verde |
| 9 | `docs(openspec): record TUI-03 apply progress` | `tasks.md`, `apply-progress.md` | — |

El esqueleto de los pasos 5 y 7 existe solo para que el RED falle por aserciones y no por compilación; no contiene comportamiento.

## Matriz de amenazas

N/A: no hay enrutamiento de agentes, comandos de shell, subprocesos, automatización de VCS/PR, clasificación de ejecutables ni integración de procesos. La única entrada del usuario que llega a una URL son los identificadores de proyecto e historia, que se escapan con `url.PathEscape` como en la TUI-02.

## Migración y despliegue

No requiere migración. Entrega en un único PR con `size:exception` (pronóstico de 800 a 950 líneas). Reversión: revertir el commit de merge; el cambio se limita a `internal/tui/` y `tests/unit/tui/`.

## Riesgos

- **Respuesta obsoleta de `List`**: si el usuario cambia de proyecto mientras una carga anterior sigue en curso, esa respuesta puede aplicarse a la nueva carga. Probabilidad baja (timeout de 5 s, uso interactivo); se acepta para no agregar `ProjectID` al mensaje.
- **Criterios con `;`**: no se pueden representar; limitación documentada en la vista del formulario.
- **Cambio de política de `q`**: altera el comportamiento de los formularios de proyectos; cubierto por pruebas RED por pantalla y por las pruebas de salida existentes.
- **Alineación con la spec**: el retorno desde `backlog-result` al formulario, el centro `backlog` con acciones y el borrado por runa son decisiones de diseño que la spec delta debe reflejar o no contradecir.

## Preguntas abiertas

- [x] Resuelta: el borrado por runa está dentro del alcance (decisión del usuario) y la spec delta lo recoge en el requisito «Los campos de texto borran por carácter».
