# Diseño: registrar esfuerzo desde la TUI (TUI-08)

## Enfoque técnico

Se implementa el enfoque 1 de la propuesta: los selectores de la sección Esfuerzo siguen viviendo en el shell `Screen == "section"` y el formulario y el resultado pasan a pantallas propias (`effort-form` y `effort-result`). El estado del flujo se agrupa en un struct `effortState`, el cliente HTTP replica a `StoryPointsClient` y las métricas del resultado reutilizan `MetricsClient.Project`, `parseMetricsPayload` y `formatMetric` sin cálculo local.

El cambio es exclusivamente de cliente TUI, specs y documentación. No toca la API ni el esquema: el `POST .../effort-records` (gate `readiness.Effort = version >= 12`, `internal/api/api.go:177`) y el `GET /projects/{id}/metrics` (gate `readiness.Metrics = version >= 14`, `internal/api/api.go:181`) ya existen. Como los gates difieren, una API con esquema 12 o 13 acepta el registro y responde `404` en métricas; el diseño lo trata como un estado normal del resultado, no como un error de registro.

Las decisiones confirmadas (fila «Toda la historia», opción B de métricas con `r`, PR único) y los supuestos S1 a S6 de la propuesta se toman como restricciones cerradas.

## Contexto verificado en el código

| Punto | Estado actual | Archivo |
|---|---|---|
| Paso de tareas de Esfuerzo | `chooseSectionItem` → `beginSectionStep(sectionStepTasks)`; al elegir tarea, aviso TUI-08 | `internal/tui/sections.go:346-376`, `:396-397` |
| Paso de integrantes | `viewSection` cablea `sectionStepMembers` a `viewPokerParticipants` sin mirar `kind` | `sections.go:433-434` |
| `Esc` en pasos secundarios | Toda acción `selectorBack` fuera de `primary`/`scope` vuelve a `primary` | `sections.go:305-306` |
| Descarte de respuestas tardías | `applySectionResult` exige `Screen == "section"`, mismo `kind`, mismo `step` y `loading` | `sections.go:204-207` |
| Edición de fecha | `updateProjectDate` lee y escribe `m.formField`, `m.formSegment`, `m.formStart`, `m.formFinish` | `internal/tui/model.go:391-431` |
| Render de fecha | `projectDateRow` depende de `m.formField` y `m.formSegment` | `internal/tui/date_field.go:169-191` |
| Teclas globales | `?` se intercepta salvo `entersText`; `p` y `q` solo en `quitsOnQ` | `model.go:157-166`, `:223-239` |
| Reanudación tras el selector de proyectos | `selectProject` reabre la sección solo si `returnTo == "section"`; cualquier otra pantalla vuelve tal cual | `internal/tui/project_picker.go:159-186` |
| Formato de errores | `describeBacklogFailure` (422 con `fields`, 404, resto) | `internal/tui/backlog_screens.go:368-403` |
| Cliente modelo | `StoryPointsClient.Set` sobre `apiClient.do` (timeout 5 s) | `internal/tui/backlog.go:67-77`, `internal/tui/api.go:25-62` |
| Contrato HTTP | Cuerpo estricto (`DisallowUnknownFields`), `task_id` puntero, `201` plano | `internal/effort/transport/http/handler.go:34-53`, `:143-160` |

## Flujo y máquina de estados

```
menu ─Enter(Esfuerzo)─▶ section/primary (historias, GET /stories)
                           │ Enter: selected = historia, primary = snapshot
                           ▼
                        section/tasks (GET /stories/{sid}/tasks + fila sintética)
                           │ Enter: detail = tarea (ID "" si «Toda la historia»), tasks = snapshot
                           ▼
                        section/members (GET /members, vista propia de Esfuerzo)
                           │ Enter: openEffortForm(integrante)
                           ▼
                        effort-form (Fecha · Actividad · Horas)
                           │ Enter en Horas: validación local → POST (saving)
                           ├── error ──▶ effort-form con err y datos intactos
                           ▼ 201
                        effort-result (registro + métricas: loading → ready | unavailable)
                           │ r: GET /metrics de nuevo
                           └ Esc/Enter ──▶ section/tasks (snapshot de la misma historia)

Esc: effort-form → section/members → section/tasks → section/primary → menu
```

| Nivel | Dónde vive | Entrada | `Esc` |
|---|---|---|---|
| Historias | `Screen "section"`, `sectionStepPrimary` | `openSection(sectionEffort)` | Menú (sin cambios) |
| Tareas | `Screen "section"`, `sectionStepTasks` | `beginSectionStep(sectionStepTasks)` | `step = primary`, `selector = primary` (sin cambios) |
| Integrantes | `Screen "section"`, `sectionStepMembers` | `beginSectionStep(sectionStepMembers)` | **Nuevo**: `step = tasks`, `selector = tasks` |
| Formulario | `Screen "effort-form"` | `openEffortForm(member)` | `Screen = "section"`; `step` y `selector` siguen siendo los de integrantes |
| Resultado | `Screen "effort-result"` | `applyEffortSave` con `201` | `Screen = "section"`, `step = tasks`, `selector = tasks` |

### Snapshot por nivel

`sectionState` ya conserva `primary` para restaurar historias sin otra request. Se agrega un segundo snapshot con el mismo patrón:

```go
type sectionState struct {
    // ...campos actuales...
    tasks selector // snapshot del selector de tareas al elegir una fila (solo Esfuerzo)
}
```

- `chooseSectionItem`, en `kind == sectionEffort && step == sectionStepTasks`: `m.section.detail = item`, `m.section.tasks = m.section.selector` y `beginSectionStep(sectionStepMembers)`. El snapshot conserva ítems, filtro y cursor, como `primary`.
- `updateSection`, rama `selectorBack`: antes de la vuelta genérica a `primary`, `if kind == sectionEffort && step == sectionStepMembers { step = sectionStepTasks; selector = tasks; return }`.
- El nivel de integrantes no necesita snapshot: mientras el formulario está abierto, `m.section.selector` sigue conteniendo la lista de integrantes (igual que Planning Poker, `planning_poker.go:48-51`).
- `openSection` reinicia `m.section.tasks = selector{}` junto con el resto del estado.

### Generalización de `sectionStepMembers` por tipo de sección

- `viewSection`: `case sectionStepMembers` delega en `viewPokerParticipants()` solo si `kind == sectionPlanningPoker`; si no, usa `viewSelector` con título «Esfuerzo · Integrante», ayuda «Integrante que realizó el trabajo.» y estado vacío propio: «El proyecto no tiene integrantes.» / «Los integrantes se dan de alta mediante la API; r vuelve a consultar.» (no remite a una opción de menú inexistente).
- La tecla `Space` y el `Enter` de confirmación de participantes ya están condicionados a `kind == sectionPlanningPoker` (`sections.go:274`, `:361`); no cambian.
- `sectionFooter` y `sectionHelpRows` ya devuelven «elegir integrante» para el caso genérico; no cambian.
- `crumbParts`: `sectionStepMembers` muestra «Participantes» para Planning Poker e «Integrante» para Esfuerzo.

### Fila sintética «Toda la historia (sin tarea)»

`sectionItems`, rama `sectionStepTasks`: cuando `result.Kind == sectionEffort`, antepone `selectorItem{ID: "", Label: "Toda la historia (sin tarea)", Meta: "esfuerzo sobre la historia"}` a las tareas. Como la proyección es pura sobre la respuesta, la fila aparece en la carga inicial y en cada refresco con `r`, y una historia sin tareas muestra igualmente una fila elegible, así que el estado vacío «La historia no tiene tareas» deja de ser alcanzable para Esfuerzo. Si el `GET` de tareas falla, el selector muestra el error con `r` para reintentar y la fila no se ofrece (no hay respuesta que proyectar); se considera aceptable porque la historia elegida podría no existir.

La ayuda del paso de tareas para Esfuerzo pasa a «Tarea de <historia> o la historia completa.».

`sectionNotice` pierde el caso `sectionEffort`, que deja de ser alcanzable; Defectos, Métricas y Dashboard no cambian.

## Estado del formulario y del resultado

```go
// internal/tui/effort_form.go

type effortField int

const (
    effortFieldDate effortField = iota
    effortFieldDescription
    effortFieldHours
    effortFormFields
)

type effortMetricsStatus int

const (
    effortMetricsIdle effortMetricsStatus = iota
    effortMetricsLoading
    effortMetricsReady
    effortMetricsUnavailable
)

// effortState groups the effort form and result screens.
type effortState struct {
    story, task, member selectorItem // task.ID == "" means the whole story
    date                dateField
    description, hours  string
    field               effortField
    segment             int
    err                 string
    saving              bool
    seq                 int // monotonic request token; survives form resets
    record              EffortRecord
    metricsStatus       effortMetricsStatus
    metrics             metricsPayload
    metricsErr          string
}

type effortSaveResult struct {
    seq      int
    response APIResponse
    err      error
}

type effortMetricsResult struct {
    seq      int
    response APIResponse
    err      error
}
```

`Model` agrega `effortClient *EffortClient`, `effort effortState` y `now func() time.Time`.

### Funciones de `effort_form.go`

| Función | Responsabilidad |
|---|---|
| `openEffortForm(member selectorItem)` | `m.effort = effortState{seq: m.effort.seq, story: section.selected, task: section.detail, member, date: dateFieldFromISO(m.today())}`; `Screen = "effort-form"`. |
| `updateEffortForm(key)` | Con `saving`, ignora toda tecla (incluidos `Enter` y `Esc`). `Esc`: limpia `err` y vuelve a `section/members`. Fecha: `date.applyKey`; Actividad y Horas: `editText`; el resto, `navigateEffortForm`. |
| `navigateEffortForm(key)` | `Tab`/`↓` siguiente campo, `Shift+Tab`/`↑` anterior (con `segment = 0`); `Enter` avanza y en Horas llama a `submitEffortForm`. |
| `submitEffortForm()` | `validateEffortInput()`; si hay error, lo guarda en `err` y no emite comando. Si no, `saving = true`, `seq++`, `registerEffortCmd`. |
| `validateEffortInput()` | Devuelve `(EffortInput, string)` según la tabla de validación local. |
| `parseEffortHours(s)` | `strconv.ParseFloat` sobre `TrimSpace`, rechaza vacío, NaN, ±Inf y `<= 0`. Sin coma decimal (S3). |
| `applyEffortSave(result)` | Guarda: `Screen == "effort-form" && saving && result.seq == seq`. Error → `err`, `saving = false`, datos intactos. `2xx` → `record`, `Screen = "effort-result"`, `metricsStatus = loading`, `seq++`, `effortMetricsCmd`. |
| `effortMetricsCmd()` | `MetricsClient.Project(projectID)` → `effortMetricsResult{seq}`. |
| `applyEffortMetrics(result)` | Guarda: `Screen == "effort-result" && metricsStatus == loading && result.seq == seq`. Éxito y payload válido → `ready`; cualquier otro caso → `unavailable` con `metricsErr`. |
| `updateEffortResult(key)` | `r` (si no está `loading`): `loading`, `seq++`, `effortMetricsCmd`. `Esc`/`Enter`: `leaveEffortResult`. |
| `leaveEffortResult()` | `Screen = "section"`, `step = sectionStepTasks`, `selector = section.tasks`, `effort = effortState{seq: effort.seq}`. Sin request. |
| `viewEffortForm()` / `viewEffortResult()` | Render; ver «Presentación». |
| `effortFormFooter()` / `effortResultFooter()` / `effortResultHelpRows()` | Pie y ayuda de las pantallas nuevas. |

El token `seq` se incrementa en cada comando emitido (guardado y métricas) y sobrevive a los reinicios de `effortState`, así que una respuesta de un formulario o resultado anterior nunca coincide con el actual. Con `Esc` bloqueado durante `saving`, la respuesta del `POST` siempre llega al formulario que la emitió y el usuario siempre ve el desenlace de un `POST` potencialmente aplicado.

### Cableado en `model.go`

- `NewModel`: `effortClient: NewEffortClient(apiURL)`, `now: time.Now`.
- `WithClock(now func() time.Time) Model` (exportado): devuelve una copia con el reloj reemplazado. Es la costura de prueba; `cmd` no la usa.
- `today() string`: `m.now` (o `time.Now` si es `nil`, para el valor cero de `Model`) → `.UTC().Format("2006-01-02")`.
- `Update`: antes del manejo de teclas, `effortSaveResult` → `applyEffortSave` y `effortMetricsResult` → `applyEffortMetrics`; en el `switch m.Screen`, `"effort-form"` → `updateEffortForm` y `"effort-result"` → `updateEffortResult`.
- `viewBody`, `footer`: casos `"effort-form"` y `"effort-result"`.
- `helpRows`: `Screen == "effort-result"` → `effortResultHelpRows()`. El formulario no ofrece ayuda porque `?` es un carácter, como en los demás formularios.
- `entersText`: agrega `"effort-form"`. `quitsOnQ`: agrega `"effort-result"` y **no** `"effort-form"`.
- `crumbParts`: `"effort-form"` → `Esfuerzo › Registrar`; `"effort-result"` → `Esfuerzo › Resultado`; ambos con `Proyecto: <nombre>`.
- `project_picker.go`, `selectProject`: la condición `returnTo == "section" && m.section.kind != ""` pasa a aceptar también `returnTo == "effort-result"`, de modo que `p` desde el resultado reabre Esfuerzo con el nuevo proyecto en lugar de volver a un resultado de otro proyecto.

### Teclas por pantalla

| Tecla | `effort-form` | `effort-result` |
|---|---|---|
| Carácter imprimible (incluye `q`, `?`, `p`, `r`) | Se escribe en Actividad u Horas; en Fecha solo cuentan dígitos y `-`/`/` | `q` sale, `?` ayuda, `p` proyecto, `r` refresca métricas |
| `Tab` / `↓` | Campo siguiente | — |
| `Shift+Tab` / `↑` | Campo anterior | — |
| `←` / `→` | Segmento de la fecha | — |
| `Backspace` | Borra (en Fecha retrocede de segmento) | — |
| `Enter` | Avanza; en Horas envía; ignorado con `saving` | Vuelve a tareas |
| `Esc` | Vuelve a integrantes; ignorado con `saving` | Vuelve a tareas |
| `Ctrl+C` | Sale siempre | Sale siempre |

`q`, `?` y `p` son caracteres en el formulario porque `effort-form` está en `entersText` y fuera de `quitsOnQ` (las dos condiciones de `model.go:157-166`). `r` nunca se intercepta globalmente: solo el selector y el resultado lo interpretan.

## Campo de fecha compartido

Se extraen dos helpers de `updateProjectDate` y `projectDateRow` sin cambiar comportamiento:

```go
// internal/tui/date_field.go

// applyKey applies an editing key to the focused segment. It reports false for
// keys the caller must handle (Tab, arrows up/down, Enter, Esc...).
func (d dateField) applyKey(seg int, key tea.KeyMsg) (dateField, int, bool)

// dateRow renders the segmented year/month/day editor with a padded label.
func (m Model) dateRow(label string, d dateField, focused bool, seg, labelWidth int) string

// alignedTextRow renders a labelled text field padded to labelWidth.
func (m Model) alignedTextRow(label, value string, focused bool, labelWidth int) string
```

- `applyKey` contiene el `switch` actual de `updateProjectDate` (`←`, `→`, `Backspace`, dígitos y separadores). El llamador limpia su error cuando la tecla es `Backspace` o `KeyRunes`, igual que hoy.
- `updateProjectDate` queda como: elegir `formStart`/`formFinish`, `applyKey`, si no la consumió `navigateProjectForm`, si la consumió escribir de vuelta y limpiar `formErr` según la tecla.
- `projectDateRow(field, label, d)` pasa a `m.dateRow(label, d, m.formField == field, m.formSegment, m.projectFieldWidth())`; `projectTextRow` delega en `alignedTextRow`.
- El formulario de esfuerzo usa `dateRow` y `alignedTextRow` con `effortLabelWidth = len("Actividad")`, de modo que las filas alinean la columna de valores y entran en 44 columnas.
- `project_form_test.go` debe seguir en verde sin modificaciones; es la red de seguridad del refactor (commit propio, ver más abajo).

## Cliente `EffortClient`

```go
// internal/tui/effort.go

// EffortInput is the body of POST /projects/{pid}/stories/{sid}/effort-records.
// An empty TaskID registers effort over the whole story and is omitted.
type EffortInput struct {
    MemberID    string  `json:"member_id"`
    TaskID      string  `json:"task_id,omitempty"`
    WorkDate    string  `json:"work_date"`
    Description string  `json:"description"`
    Hours       float64 `json:"hours"`
}

// EffortRecord is the flat 201 response.
type EffortRecord struct {
    ID          string  `json:"id"`
    ProjectID   string  `json:"project_id"`
    StoryID     string  `json:"story_id"`
    TaskID      *string `json:"task_id"`
    MemberID    string  `json:"member_id"`
    WorkDate    string  `json:"work_date"`
    Description string  `json:"description"`
    Hours       float64 `json:"hours"`
}

type EffortClient struct{ api apiClient }

func NewEffortClient(baseURL string) *EffortClient

// Register sends one effort record; the API is not idempotent.
func (c *EffortClient) Register(ctx context.Context, projectID, storyID string, input EffortInput) (APIResponse, error) {
    return c.api.do(ctx, http.MethodPost, storiesPath(projectID)+"/"+url.PathEscape(storyID)+"/effort-records", input)
}
```

El cuerpo contiene exactamente las claves que acepta el decodificador estricto del handler; `task_id` se omite (no se envía `null`) para «Toda la historia». `hours` no lleva `omitempty` porque la validación local garantiza que es mayor que 0.

### Mapeo de errores

| Respuesta | Mensaje en el formulario | `saving` / datos |
|---|---|---|
| Error de transporte (incluye timeout de 5 s) | «No se pudo confirmar el registro: error de conexión (<detalle>). Es posible que el esfuerzo se haya guardado; conviene revisar las horas reales en Métricas antes de reintentar.» | `false` / intactos |
| `400 invalid_request` | `describeBacklogFailure` → «Error HTTP 400: invalid_request: …» | `false` / intactos |
| `422 validation_failed` | `describeBacklogFailure` → «La API rechazó los datos (422): …» con cada `field` | `false` / intactos |
| `404 *_not_found` | `describeBacklogFailure` → «El recurso solicitado no existe (404). Detalle de la API: story_not_found: …» | `false` / intactos |
| `500` y otros no-2xx | `describeBacklogFailure` → «Error HTTP 500: …» | `false` / intactos |
| `2xx` con cuerpo ilegible | Se considera registrado; el resultado muestra los datos enviados y la nota «No se pudo interpretar la respuesta de la API.» y consulta métricas | — |

El mensaje de transporte no usa `sectionFailure` porque debe advertir que el `POST` pudo aplicarse.

## Validación local

| Regla | Mensaje | Request |
|---|---|---|
| Historia elegida (`story.ID != ""`) | «Falta elegir la historia.» | Ninguna |
| Integrante elegido (`member.ID != ""`) | «Falta elegir el integrante.» | Ninguna |
| Fecha completa | «La fecha de trabajo está incompleta.» (`dateField.validate`) | Ninguna |
| Fecha válida | «La fecha de trabajo no es una fecha válida.» | Ninguna |
| Horas presentes | «Las horas son obligatorias.» | Ninguna |
| Horas numéricas (punto decimal) | «Las horas deben ser un número; el separador decimal es el punto.» | Ninguna |
| Horas `> 0` | «Las horas deben ser mayores que 0.» | Ninguna |

Las dos primeras reglas son defensivas: el flujo solo abre el formulario tras elegir ambas. Quedan para la API y se muestran con su motivo: horas `<= 24` y con hasta 2 decimales, actividad de 1 a 500 caracteres (se envía sin recortar; la API recorta), fecha no posterior a la UTC del servidor, existencia de proyecto, historia, tarea e integrante. El texto definitivo de los mensajes lo fija la spec; el diseño evita imperativos para no mezclar registros con la copia existente.

## Presentación

**Formulario** (`viewEffortForm`): título «Registrar esfuerzo»; ayuda con `Historia: <título>` y `Tarea: <título>` o `Tarea: toda la historia`, e `Integrante: <nombre>`; filas Fecha (segmentada, con la de hoy UTC), Actividad y Horas; `errorBlock(err)` si hay error; con `saving`, `statusMutedStyle` «Registrando…». Pie: `Tab/↑/↓ campo · ←/→ fecha · Enter siguiente/enviar · Esc volver · Ctrl+C salir`; con `saving`, solo `Ctrl+C salir`.

**Resultado** (`viewEffortResult`): `okBlock` «Esfuerzo registrado.» con Historia, Tarea o «toda la historia», Integrante, Fecha (`record.WorkDate`), Actividad (`record.Description`), Horas (`record.Hours`) e ID corto, todos tomados de la respuesta. Bloque «Métricas del proyecto» según `metricsStatus`:

| Estado | Contenido |
|---|---|
| `loading` | «Cargando métricas…» |
| `ready` | `Horas reales: formatMetric(ActualHours, " h")` y `Desviación: formatMetric(Deviation, "")`, mismas etiquetas que `viewMetrics` |
| `unavailable` | «El esfuerzo quedó registrado; las métricas del proyecto no están disponibles.» + `metricsErr` (de `sectionFailure` o «No se pudo interpretar la respuesta de la API.») + «r para reintentar.» con estilo de aviso, nunca `errorBlock` de registro |

Pie: `r refrescar métricas · Enter/Esc otra carga · p proyecto · ? ayuda · q salir`. Ayuda: `r`, `Enter / Esc`, `p`, `?`, `q`, `Ctrl+C`.

## Flujo de datos

```
selector (section)          effort-form                       API
──────────────────          ───────────                       ───
historia ── GET /stories ──────────────────────────────────▶  200
tarea ───── GET /stories/{sid}/tasks ──────────────────────▶  200 (+ fila sintética local)
integrante  GET /members ──────────────────────────────────▶  200
            openEffortForm(today UTC)
            Enter ─ validateEffortInput ─ saving, seq=n
                    POST /stories/{sid}/effort-records ────▶  201 | 4xx | 5xx | transporte
            applyEffortSave(seq==n)
                 ├─ error → err, saving=false (formulario)
                 └─ 201 → effort-result, seq=n+1
                          GET /projects/{pid}/metrics ─────▶  200 | 404 (esquema 12-13) | 5xx
                          applyEffortMetrics(seq==n+1) → ready | unavailable
                          r → seq=n+2, GET /metrics de nuevo
```

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/tui/effort.go` | Crear | `EffortInput`, `EffortRecord`, `EffortClient`, `NewEffortClient`, `Register`. |
| `internal/tui/effort_form.go` | Crear | `effortState`, mensajes, apertura, edición, validación, envío, resultado, métricas, vistas, pie y ayuda. |
| `internal/tui/sections.go` | Modificar | Campo `tasks`, fila sintética en `sectionItems`, transición tareas → integrantes → formulario, `Esc` de integrantes a tareas, vista de integrantes por `kind`, ayuda de tareas, retiro del caso Esfuerzo de `sectionNotice`. |
| `internal/tui/model.go` | Modificar | Campos `effortClient`, `effort`, `now`; `WithClock`, `today`; cableado en `NewModel`, `Update`, `viewBody`, `footer`, `helpRows`, `crumbParts`, `entersText`, `quitsOnQ`; `updateProjectDate` sobre `applyKey`. |
| `internal/tui/date_field.go` | Modificar | `applyKey`, `dateRow`, `alignedTextRow`; `projectDateRow` y `projectTextRow` delegan. |
| `internal/tui/project_picker.go` | Modificar | `selectProject` reanuda Esfuerzo desde `effort-result`. |
| `internal/tui/metrics.go` | Sin cambios | Reutilizado. |
| `tests/unit/tui/effort_api_test.go` | Crear | Contrato del cliente. |
| `tests/unit/tui/effort_test.go` | Crear | Flujo, validación, errores, anti doble envío, resultado, métricas, navegación y teclas. |
| `tests/unit/tui/sections_flows_test.go` | Modificar | Elimina `TestEffortFlowChoosesStoryAndTaskThenShowsTheTUI08Notice`; `effortServer` pasa a `effort_test.go` ampliado. |
| `openspec/changes/tui-08-registrar-esfuerzo/specs/**` | Crear (fase spec) | Deltas de `tui` y `esfuerzo`. |
| `README.md` | Modificar | Fila Esfuerzo: historia → tarea o historia completa → integrante → registro. |

## Estrategia de pruebas

TDD estricto: cada comportamiento empieza con una prueba en ROJO observada con `go test ./tests/unit/tui/...` y cierra con `go test ./...` en verde. Las pruebas viven en `tests/unit/tui/` (paquete `tui_test`), usan `httptest` y no requieren Docker.

### Fixtures y helpers

- `fixedClock = func() time.Time { return time.Date(2026, 10, 31, 23, 30, 0, 0, time.FixedZone("UTC-3", -3*3600)) }`: en UTC es `2026-11-01`, lo que prueba que la fecha inicial usa UTC y no la zona local.
- `effortFlowServer(t, opts)` (sobre `sprintServer`): sirve historias (`s1` con tareas `t1`, `t2`; `s2` sin tareas), integrantes (`m1`, `m2`, o vacío según `opts`), registra cada request (método, ruta, cuerpo) y responde el `POST` y el `GET /metrics` con el estado y cuerpo configurados. Debe declarar `/projects/p1/metrics` explícitamente: el *fallback* de `sprintServer` responde `200 {"ok":true}`, que `parseMetricsPayload` aceptaría con todos los valores `nil`.
- `openEffortFormFor(t, server, storyIndex, taskIndex, memberIndex)`: `openSectionWithProject(..., menuEffort)`, `WithClock(fixedClock)` y la secuencia de `Enter`/`↓` con `run` para cada `GET`.
- Se reutilizan `apply`, `run`, `press`, `typeText`, `stripANSI`, `recordingServer`, `onlyRequest`, `bodyKeys`.

### `tests/unit/tui/effort_api_test.go`

| Prueba | Comportamiento |
|---|---|
| `TestEffortClientRegistersTaskEffortWithExactContract` | `POST /projects/p1/stories/s1/effort-records`, claves exactas `member_id`, `task_id`, `work_date`, `description`, `hours` y valores. |
| `TestEffortClientOmitsTaskIDForWholeStory` | Con `TaskID == ""`, el cuerpo no contiene `task_id`. |
| `TestEffortClientEscapesPathSegments` | IDs con caracteres reservados se escapan. |
| `TestEffortClientReturnsHTTPErrorsAsResponses` | `422` y `404` vuelven como `APIResponse` sin `error`. |

### `tests/unit/tui/effort_test.go`

| Grupo | Comportamiento |
|---|---|
| Selector de tareas | La fila «Toda la historia (sin tarea)» aparece primera; una historia sin tareas la ofrece y no muestra «La historia no tiene tareas»; el aviso TUI-08 ya no aparece (reemplaza a `TestEffortFlowChoosesStoryAndTaskThenShowsTheTUI08Notice`). |
| Integrantes | Elegir una tarea carga integrantes con vista de Esfuerzo (sin `[ ]` ni `Space`); proyecto sin integrantes muestra el estado vacío propio sin remitir al menú; error de la API visible con `r`. |
| Formulario | Abre con la fecha UTC del reloj fijo (`2026 11 01`), Historia, Tarea e Integrante visibles; `Tab`/`↑`/`↓`/`←`/`→` y `Backspace` en la fecha. |
| Teclas como caracteres | `q`, `?`, `p` y `r` escritos en Actividad y Horas aparecen en el campo; la pantalla sigue siendo `effort-form`, no se abre ayuda ni selector de proyectos y el comando no es `tea.Quit`. |
| Validación local | Fecha incompleta, fecha inválida, horas vacías, no numéricas, `1,5`, `0` y negativas: mensaje visible y **cero** requests de escritura. |
| Envío | Cuerpo exacto con tarea; sin `task_id` para «Toda la historia»; `work_date` normalizado. |
| Anti doble envío | `Enter` → comando; segundo `Enter` → `nil`; tras `run`, un único `POST`. `Esc` durante `saving` no cambia de pantalla. |
| Errores de la API | `400`, `422` con `fields`, `404 member_not_found`, `500`: motivo visible, campos conservados, sin «registrado». Transporte (servidor cerrado): «No se pudo confirmar el registro». |
| Resultado | Datos de la respuesta (descripción recortada por la API), `Horas reales: 30.5 h` y `Desviación: -4` sin cálculo local. |
| Métricas no disponibles | `404` y `500` en `/metrics`: «El esfuerzo quedó registrado» y métricas no disponibles, sin mensaje de error de registro. |
| `r` | Emite un nuevo `GET /projects/p1/metrics` y ningún `POST`; actualiza los valores. |
| Descarte tardío | Comando de métricas ejecutado después de `Esc`: la pantalla queda en el selector de tareas sin cambios. |
| `Esc` de tres niveles | Formulario → integrantes → tareas (con el filtro conservado) → historias, sin requests nuevas. |
| Tras el éxito | `Esc` y `Enter` vuelven al selector de tareas de la misma historia; un segundo registro abre un formulario limpio. |
| `p` desde el resultado | Al elegir otro proyecto se reabre Esfuerzo en el selector de historias del nuevo proyecto. |
| Navegación sin mutación | Recorrer historia → tarea → integrante → formulario y volver con `Esc` emite solo `GET`; el único `POST` es el del envío explícito. |
| Miga, pie y ayuda | `Esfuerzo › Integrante`, `Esfuerzo › Registrar`, `Esfuerzo › Resultado`; pie del resultado anuncia `r refrescar métricas`. |

### Pruebas existentes que deben seguir en verde

- `TestMenuRoutesToEachSprint3Section`: solo abre el paso principal; no cambia.
- `TestSectionNavigationOnlyEmitsGetRequests`: recorre Planning Poker y termina sin `POST`; no cambia. La cobertura de Esfuerzo se agrega como prueba nueva en `effort_test.go` en lugar de ampliar esta, para no mezclar flujos.
- `project_form_test.go` completo: protege el refactor de fecha.
- `planning_poker_test.go` y `TestPlanningPokerParticipantSelectorShowsLoadingAndAPIError`: protegen la vista de participantes ahora condicionada por `kind`.
- `sections_api_test.go`, `footer_test.go`, `ux_test.go`, `internal/tui/selector_test.go`.

## Matriz de amenazas

N/A: el cambio agrega pantallas y un cliente HTTP dentro del proceso de la TUI contra la `API_URL` ya configurada; no hay comandos de shell, subprocesos, automatización de VCS/PR, clasificación de archivos ejecutables ni integración de procesos.

## Decisiones de arquitectura

| # | Decisión | Elección | Alternativas descartadas | Razón |
|---|---|---|---|---|
| D1 | Ubicación del formulario y el resultado | Pantallas propias `effort-form` y `effort-result` | Un `sectionStep` dentro de `Screen "section"`; flujo fuera del shell | `section` está en `quitsOnQ` y no en `entersText`: `q`, `?` y `p` romperían la escritura de la actividad. Sigue el precedente de `planning-poker` y `sprint-task-form`. |
| D2 | `Esc` de tres niveles | Snapshot nombrado `sectionState.tasks`, junto a `primary` | Pila genérica de selectores; recargar la lista con un `GET` al volver | Solo Esfuerzo tiene tres niveles; un campo explícito sigue el patrón vigente sin generalizar la navegación de las demás secciones, y restaurar sin request conserva filtro y cursor. |
| D3 | Paso de integrantes | Reutilizar `sectionStepMembers` con vista elegida por `kind` | Paso nuevo `sectionStepEffortMembers`; seleccionar el integrante dentro del formulario | Carga, error, refresco y descarte tardío ya funcionan para ese paso; solo cambian vista, miga y estado vacío. |
| D4 | Fila «Toda la historia» | Ítem sintético con `ID ""` antepuesto en `sectionItems` | Paso previo «¿tarea o historia?»; fila agregada solo en la vista | La proyección pura se aplica igual en la carga y en `r`, y el `ID` vacío se traduce directamente en `task_id` omitido. |
| D5 | Estado | Struct `effortState` | Campos planos en `Model` (como Poker) | Agrupa 15 campos de un flujo, permite reiniciarlo en una asignación y evita crecer `Model` sin estructura. |
| D6 | Respuestas tardías | Token `seq` monotónico más guarda de pantalla y estado | Solo guarda de pantalla (como `applyPokerSave`) | La guarda de pantalla no distingue dos consultas de métricas sucesivas ni un resultado de un formulario anterior. |
| D7 | `Esc` durante el envío | Bloqueado mientras `saving` | Permitir salir y descartar la respuesta | La API no es idempotente: descartar la respuesta de un `POST` aplicado ocultaría el registro y empujaría a duplicarlo. La espera está acotada por el timeout de 5 s. |
| D8 | Error de transporte | Mensaje propio «No se pudo confirmar el registro…» | `sectionFailure` («Error de conexión: …») | El timeout puede ocurrir con el `POST` ya confirmado; el mensaje debe advertirlo. |
| D9 | Métricas en el resultado | Estado independiente `metricsStatus` con `unavailable` | Tratar el fallo de métricas como error del registro; ocultar el bloque | Los gates de esquema difieren (12 frente a 14); el registro es exitoso aunque las métricas fallen. |
| D10 | Fecha compartida | `dateField.applyKey`, `dateRow`, `alignedTextRow` | Reutilizar `m.formField`/`m.formSegment`; duplicar la lógica | Desacopla los formularios sin duplicar la edición segmentada; el refactor queda cubierto por `project_form_test.go`. |
| D11 | Reloj | Campo `now` con `time.Now` por defecto y `WithClock` exportado | Campo exportado `Now`; variable de paquete | Las pruebas están en `tui_test` y necesitan una costura exportada; un método evita exponer estado mutable y no es global. |
| D12 | `p` desde el resultado | `selectProject` reabre Esfuerzo si `returnTo == "effort-result"` | Excluir `p` del resultado | `p` es un atajo global anunciado en todas las pantallas de navegación; volver a un resultado de otro proyecto sería incoherente. No se cambia el caso equivalente de Planning Poker (fuera de alcance). |

## Migración y despliegue

Sin migraciones. El registro necesita esquema `>= 12`; las métricas del resultado, `>= 14`. Con esquema 12 o 13, el resultado informa que las métricas no están disponibles. Revertir el PR restaura el aviso TUI-08 y la implementación previa del campo de fecha; los registros creados siguen siendo válidos.

## Tamaño estimado y unidades de trabajo

Estrategia `single-pr` con `size:exception` (decisión 3). Líneas autoras estimadas (adiciones más eliminaciones):

| Archivo | Líneas |
|---|---|
| `internal/tui/effort.go` | ~55 |
| `internal/tui/effort_form.go` | ~270 |
| `internal/tui/sections.go` | ~50 |
| `internal/tui/model.go` | ~70 (incluye ~35 del refactor de fecha) |
| `internal/tui/date_field.go` | ~60 |
| `internal/tui/project_picker.go` | ~3 |
| **Producción** | **~510** |
| `tests/unit/tui/effort_api_test.go` | ~90 |
| `tests/unit/tui/effort_test.go` | ~430 |
| `tests/unit/tui/sections_flows_test.go` | ~40 (eliminación y traslado de `effortServer`) |
| **Pruebas** | **~560** |
| Specs y `README.md` | ~150 |
| **Total** | **~1.200** |

El total queda en el borde superior de la estimación de la propuesta; ningún archivo supera por sí solo el presupuesto de 400 líneas salvo `effort_test.go`, que puede dividirse en `effort_form_test.go` y `effort_result_test.go` si resulta más legible.

Commits por unidad de trabajo (mensajes en español con prefijo `type(scope)` en inglés). Cada uno incluye sus pruebas, compila y deja `go test ./...` en verde.

| # | Commit | Contenido | Líneas aprox. |
|---|---|---|---|
| 1 | `refactor(tui): extraer helpers genéricos del campo de fecha` | `applyKey`, `dateRow`, `alignedTextRow`; formulario de proyecto delega; `project_form_test.go` en verde antes y después | ~95 |
| 2 | `feat(tui): agregar cliente de registro de esfuerzo` | `effort.go` + `effort_api_test.go` | ~145 |
| 3 | `feat(tui): elegir tarea o historia completa e integrante en Esfuerzo` | Fila sintética, paso de integrantes, snapshot `tasks`, `Esc` de niveles, formulario con edición, reloj, teclas como caracteres, miga y pie; retiro del aviso TUI-08 y de su prueba | ~380 |
| 4 | `feat(tui): registrar esfuerzo con validación local y anti doble envío` | Validación, `POST`, `saving`, `seq`, mapeo de errores | ~260 |
| 5 | `feat(tui): mostrar el registro y las métricas del proyecto tras registrar` | Resultado, métricas, `r`, descarte tardío, vuelta a tareas, `p` | ~240 |
| 6 | `docs(tui): actualizar specs y README para TUI-08` | Deltas de `tui` y `esfuerzo`, fila del `README.md` | ~150 |

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Duplicados por doble `Enter` o reintento tras timeout | `saving` ignora `Enter` y `Esc`; mensaje de transporte que advierte el posible registro; prueba de una sola request. |
| El refactor de fecha altera el formulario de proyecto | Commit 1 aislado con `project_form_test.go` sin cambios. |
| La vista de participantes de Poker cambia por la condición de `kind` | Pruebas de Planning Poker existentes en verde. |
| El *fallback* `{"ok":true}` de `sprintServer` oculta un fallo de métricas en pruebas | El fixture declara `/projects/p1/metrics` explícitamente en cada caso. |
| La fecha UTC puede ser «mañana» para un usuario al oeste de UTC | Supuesto S1 cerrado; la fecha es editable y el reloj fijo de las pruebas cubre el cruce de día. |
| Commit 3 cerca del presupuesto de 400 líneas | Aceptable dentro de `size:exception`; si crece, separar «selectores» de «formulario» en dos commits. |

## Preguntas abiertas

Ninguna bloqueante. La spec fija el texto exacto de los mensajes de validación, del estado vacío de integrantes y del aviso de métricas no disponibles; el diseño propone redacciones sin imperativos.
