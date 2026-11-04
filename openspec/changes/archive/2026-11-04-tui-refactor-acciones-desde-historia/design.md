# Diseño: acciones desde la historia en la TUI (tui-refactor-acciones-desde-historia)

## Enfoque técnico

Se implementa el enfoque A' de la propuesta. Se agrega una pantalla propia `story-detail` que guarda solo el ID de la historia y la relee de `m.backlogStories` en cada render. Desde ella se lanzan los tres flujos existentes con la historia ya elegida: Planning Poker y Esfuerzo siguen viviendo en el shell `Screen == "section"`, pero empiezan en su primer paso secundario (integrantes o tareas). El alta de defecto empieza en `defect-create-form` con la historia precargada. Un único campo tipado `returnTo` decide a dónde vuelven los flujos al cancelar desde su primer paso, al cancelar el alta y al completarse con éxito.

Una vez que el detalle y `returnTo` existen, se retiran las pestañas Estimación y Esfuerzo, los pasos `primary` de las secciones, el paso `sectionStepSprints` y la sección muerta `sectionDefects`.

El cambio es exclusivamente de cliente TUI, pruebas y documentación: no agrega endpoints ni tipos de request. Las únicas requests que se emiten son las que ya existen (`GET /stories`, `GET /members`, `GET /stories/{sid}/tasks`, `GET /sprints/open`, `PUT story-points`, `POST effort-records`, `POST defects`).

Las decisiones 1 a 11 de la propuesta son restricciones cerradas. El supuesto S1 (Backlog resaltado en el detalle y en los flujos que parten de él) se toma como valor por defecto, igual que en el delta de spec.

## Contexto verificado en el código

Se verificaron contra el código actual las referencias `archivo:línea` de la exploración. Se corrigieron tres:

| Punto | Estado actual | Archivo |
|---|---|---|
| Pestañas | `workspaceTabs` con seis entradas; `tabForKey` y el texto `1–%d` del pie derivan de `len(workspaceTabs)` | `internal/tui/workspace.go:27`, `:106-124`; `internal/tui/keys.go:73` |
| `openTab` | Estimación → `openSection(sectionPlanningPoker)`; Esfuerzo → `openSection(sectionEffort)` | `workspace.go:49-64` |
| `currentTab` | `defect-*` mapea a Defectos por prefijo; `planning-poker`, `effort-form` y `section` mapean por `kind` | `workspace.go:68-91` |
| `tabRoot` | Incluye `section` en `sectionStepPrimary` para Poker y Esfuerzo | `workspace.go:95-103` |
| Backlog | `Enter` → `editSelectedStory`; `P`/`E` → `openSectionForStory`; cualquier otra runa inicia el filtro | `internal/tui/backlog_screens.go:197-217` |
| `openSectionForStory` | `openSection(kind)` (descarta el `GET` primario), `selected`, `preselected = true`, `beginSectionStep` | `backlog_screens.go:235-246` |
| `Esc` del primer paso | Con `preselected` y el paso siguiente a la historia, abre `openBacklog()` | `internal/tui/sections.go:239-241` |
| `sectionDefects` | **Corrección:** no es solo código muerto. `crumbParts` usa `sectionDefects.title()` como etiqueta de todas las pantallas `defect-*` | `internal/tui/model.go:781`; `sections.go:18`, `:28`, `:38`, `:52`, `:64` |
| `sectionStepSprints` | Solo es alcanzable desde `sectionDefects` (`storyNextStep`) y la rama de Sprints de `loadSectionStepCmd`/`sectionItems` | `sections.go:52`, `:159-163`, `:176-181`, `:408`, `:446-447` |
| Éxito de Poker | `applyPokerSave` no tiene guarda de pantalla, de estado ni de proyecto; `Enter` en `confirm` puede emitir el `PUT` dos veces (defecto preexistente, fuera de este cambio: ver Riesgos) | `internal/tui/planning_poker.go:87-88`, `:100-102`, `:177-197` |
| Éxito de Esfuerzo | `applyEffortSave` → `openSection(sectionEffort)` | `internal/tui/effort_form.go:231` |
| Alta de defecto | `Esc` del formulario → `"defects"` (`defect_creation.go:47`); `n` en la confirmación → `"defects"` (`defect_submission.go:36`); **corrección:** el éxito es `m.openDefects()` (recarga), no un literal (`defect_submission.go:79`) | |
| `Esc` en el selector de Sprints del alta | Vuelve a `defect-story-select` | `defect_creation.go:139-140` |
| Guarda de la lista del backlog | Solo acepta la respuesta con `Screen == "backlog" && backlogLoading` | `backlog_screens.go:328-331` |
| Selector compartido | Cualquier runa inicia el filtro y `r` refresca | `internal/tui/selector.go:138-149` |
| Cambio de proyecto | **Corrección:** ya no existe el atajo global `p`. Se cambia de proyecto con `Esc` → `openProjectPicker` → `selectProject` → `openDashboard` | `internal/tui/project_picker.go:151-157` |

## Decisiones de arquitectura

### D1. Destino de retorno como enumeración tipada

**Elección:** `returnTo returnTarget` en `Model`, con dos valores: `returnDefault` (valor cero) y `returnStoryDetail`. Cada punto de entrada a un flujo asigna `returnTo` de forma explícita, nunca lo hereda. Los sitios de retorno llaman a un único helper `backToStoryDetail()` cuando el valor es `returnStoryDetail` y conservan su destino actual cuando es `returnDefault`.

**Alternativas descartadas:**
- Guardar el nombre de la pantalla de destino (`returnTo string`, como `formReturn`). Volver no es solo asignar `Screen`: Defectos recarga la lista y el detalle no recarga nada. Un string abierto obligaría a un `switch` con efectos secundarios en cada sitio y admitiría valores inválidos.
- Guardar un closure `func(Model) (Model, tea.Cmd)` en el modelo. No es comparable, no se puede inspeccionar en pruebas y captura una copia vieja del `Model`.
- Un flag por flujo (`pokerFromStory`, `defectFromStory`). Repite `preselected`, que es justamente lo que se quiere eliminar.

**Razón:** dos valores alcanzan para todos los orígenes (pestaña Defectos o detalle). La enumeración permite una prueba de tabla por sitio de retorno, y el valor cero conserva el comportamiento actual del alta desde Defectos sin tocar sus pruebas.

### D2. El detalle no reutiliza `selector` para las acciones

**Elección:** la lista de acciones es una tabla fija `storyActions` (tecla, etiqueta, ayuda, apertura) con su propio cursor `storyDetail.cursor`. `updateStoryDetail` maneja `↑`/`↓`, `Enter`, `p`/`t`/`d` y `Esc` directamente. El render reutiliza `m.row(selected, text)`.

**Alternativas descartadas:**
- Reutilizar `selector`. Su `update` convierte cualquier runa en filtro y `r` en refresco (`selector.go:138-149`), así que se comería los atajos. Además, una lista fija de tres filas no necesita filtro, carga ni error.
- Usar solo atajos, sin filas. La decisión 4 exige ambos.

**Razón:** la misma tabla alimenta las filas, los atajos, las `bindings` del pie y la ayuda, de modo que no pueden desalinearse. Es el mismo principio de fuente única que ya sigue `bindings()`.

### D3. Atajos `p`, `t` y `d`, y pie acotado

**Elección:** `p` Estimar (Planning Poker), `t` Registrar esfuerzo (tiempo trabajado) y `d` Reportar defecto. Coinciden con el delta de spec.

Conflictos revisados en `keys.go`, los `update*` de cada pantalla y la regla de «Listas navegables con teclas uniformes»:

| Tecla | Uso actual | Veredicto |
|---|---|---|
| `e` | Editar (Proyectos y, desde este cambio, Backlog) | Descartada: el detalle no edita (decisión 6 de la spec) |
| `r`, `n`, `x`, `h` | Refrescar, crear, destructiva, vista relacionada (regla uniforme) | Descartadas |
| `s`, `y`, `u`, `a`, `c` | Confirmaciones, estado de defecto, asignar, completar | Descartadas |
| `p` | Sin uso desde TUI-12 (se retiró el atajo global de proyecto) | **Elegida** |
| `t` | Sin uso | **Elegida** |
| `d` | PDF en Dashboard y en Sprints (otras pantallas) | **Elegida**: el detalle no ofrece PDF |
| `1`–`4` | Pestañas solo en raíces (`tabRoot`) | Libres en el detalle, pero se ignoran |

El pie (`footerLimit = 5`) marca como primarias `↑/↓ mover`, `Enter elegir acción` y `Esc backlog`. `p`, `t` y `d` se agregan como no primarias en el grupo «Acciones» de la ayuda y se ven en cada fila (`p  Estimar`). Con `q`, `?` y Ctrl+C ocultos, el pie termina en `? más`.

**Alternativa descartada:** marcar los seis atajos como primarios. Supera el límite de cinco y el pie ocultaría `Esc`, que es la única salida de la pantalla.

**Razón:** cumple el escenario «Pie y ayuda del detalle» (como máximo cinco atajos, incluidos `Enter` y `Esc`, y la ayuda lista `p`, `t` y `d`). Además, la tecla se ve donde está la acción.

### D4. Se conservan `sectionState` y `sectionKind`, recortados

**Elección:** Poker (paso de integrantes) y Esfuerzo (pasos de tareas e integrantes) siguen en `Screen "section"` con `sectionState`. Esos pasos ya resuelven carga, error, refresco, filtro y descarte tardío. Se recorta todo lo que solo servía al selector de historias y a `sectionDefects`:

| Se elimina | Se reemplaza por |
|---|---|
| `sectionDefects`, `sectionStepSprints` y sus ramas en `title`, `loadSectionStepCmd`, `sectionItems`, `viewSection`, `sectionBindings` y `crumbParts` | — |
| `sectionStepPrimary`, `primaryUsesStories`, `primaryHint`, `primaryEmptyTitle`, `primaryEmptyHint`, `sectionState.primary`, rama primaria de `loadSectionStepCmd`, `sectionItems` y `viewSection` | `sectionStepNone` como valor cero centinela |
| `storyNextStep` | `firstStep()` (Poker → `sectionStepMembers`; Esfuerzo → `sectionStepTasks`) |
| `sectionState.preselected` y `openSection` | `openStoryFlow(kind)` y `returnTo` (D1) |
| `sectionDefects.title()` en `crumbParts` | El literal `tabDefects.title()` («Defectos») |
| Caso `section` de `tabRoot` | — (ninguna pantalla `section` es raíz de pestaña) |

`storySelectorItem` se conserva: lo usan el selector de historias del alta desde Defectos y `openStoryFlow`.

**Alternativas descartadas:**
- Mover Poker y Esfuerzo a pantallas propias (`poker-members`, `effort-tasks`). Duplicaría la carga, el error y la guarda de `applySectionResult` sin beneficio funcional y agregaría unas 150 líneas.
- Conservar los pasos `primary` sin uso. Deja ramas inalcanzables que el lector tiene que descartar a mano y que confunden `tabRoot`.

**Razón:** recorta unas 115 líneas de producción y deja en `sections.go` solo los pasos alcanzables.

### D5. El detalle relee por ID y la estimación recarga el backlog

**Elección:**
- `storyDetailState{storyID string; cursor int}`. `detailStory()` busca el ID en `m.backlogStories` en cada render y en cada tecla.
- El éxito de Poker vuelve al detalle y llama a `reloadBacklogInPlace()`: `backlogStories = nil`, `backlogLoading = true`, `backlogCursor = 0`, `backlogMessage = ""` y `listStoriesCmd`, **sin cambiar `Screen`**.
- La guarda de la lista en `applyBacklogResult` pasa a aceptar `Screen ∈ {"backlog", "story-detail"}`.
- El éxito de Esfuerzo y el del alta de defecto vuelven al detalle sin recargar: esos flujos no cambian ninguno de los siete campos mostrados.

El detalle tiene cuatro estados, evaluados en este orden:

| Estado | Condición | Acciones (`Enter`, `p`, `t`, `d`) | `Esc` |
|---|---|---|---|
| Cargando | `backlogLoading` | Ignoradas | Backlog |
| Error | `backlogMessage != ""` | Ignoradas | Backlog |
| Historia ausente | `detailStory()` no la encuentra | Ignoradas | Backlog |
| Listo | Historia encontrada | Disponibles | Backlog |

**Alternativas descartadas:**
- Parchear la historia en `m.backlogStories` con el cuerpo del `PUT story-points` (que devuelve la historia completa, `internal/story/transport/http/set_story_points_handler.go:45`). Ahorra un `GET`, pero contradice los escenarios «El detalle refleja los Story Points tras estimar», «Mientras el backlog se recarga…» y «Falla la recarga del backlog» del delta de spec. También cambia el contrato actual «`PUT` seguido de `GET /stories`» que fija `planning_poker_test.go:61`.
- Copiar la `Story` dentro del estado del detalle. Mostraría datos viejos tras la estimación, que es lo que la decisión 6 prohíbe.
- Una request individual de historia. No existe y está fuera de alcance.

**Razón:** releer por ID hace que el detalle nunca muestre datos anteriores. Recargar solo tras estimar reutiliza el comando y la guarda existentes, sin requests nuevas.

### D6. El cursor filtrado y la vuelta al backlog

**Elección:** `Enter` en el Backlog toma `stories := m.visibleStories()` y guarda `stories[m.backlogCursor].ID`, es decir, el índice sobre la lista **filtrada**. `Esc` en el detalle solo asigna `Screen = "backlog"`: no llama a `openBacklog()` ni a `startBacklogLoad()`, así que el filtro y el cursor se conservan. Si hubo una recarga (tras estimar), `reloadBacklogInPlace` ya dejó el cursor en 0 y conservó el filtro. Eso coincide con el comportamiento de cualquier recarga del backlog que pide el delta de spec.

**Alternativa descartada:** recalcular el cursor buscando el ID en la lista visible al volver. Cambia el comportamiento de recarga estándar que pide la spec y agrega lógica sin un escenario que la exija.

### D7. El alta de defecto desde la historia reutiliza el flujo actual

**Elección:** `beginDefectCreationForStory(story)` llama a `beginDefectCreation()` y después asigna `storyID`, `storyLabel` y `returnTo = returnStoryDetail`. `beginDefectCreation()` (que usa `n` en Defectos) asigna `returnTo = returnDefault` de forma explícita. Como `updateDefectCreateForm` ya salta el selector cuando `storyID != ""` (`defect_creation.go:62-67`), el formulario precargado va directo a `loadDefectOpenSprints`. Cambios puntuales:

| Sitio | `returnDefault` (pestaña Defectos) | `returnStoryDetail` |
|---|---|---|
| `Esc` en `defect-create-form` (`defect_creation.go:46-48`) | `"defects"` (sin cambios) | `backToStoryDetail()` (descarta el borrador) |
| `Esc` en `defect-sprint-select` (`:139-140`) | `"defect-story-select"` (sin cambios) | `"defect-create-form"` (conserva el borrador) |
| `n` en `defect-create-confirm` (`defect_submission.go:35-36`) | `"defects"` (sin cambios) | `backToStoryDetail()` |
| Éxito (`defect_submission.go:78-79`) | `openDefects()` con aviso (sin cambios) | Aviso «Defecto registrado: …» y `backToStoryDetail()`, sin recargar defectos |
| Ayuda del formulario (`viewDefectCreateForm`) | «… Enter en el último campo elige la historia.» | «… Enter en el último campo elige el Sprint de detección.» |

Las guardas `projectID`, `Screen` y `requestID` de `applyDefectOptionsResult` y `applyDefectCreateResult` no cambian.

**Alternativa descartada:** una pantalla de alta separada para el origen historia. Duplicaría formulario, confirmación y guardas.

### D8. Pestaña y miga por origen

**Elección:** `currentTab` evalúa primero las pantallas del flujo de historia:

```go
case m.Screen == "story-detail", m.Screen == "section",
	m.Screen == "planning-poker", m.Screen == "effort-form":
	return tabBacklog
case strings.HasPrefix(m.Screen, "defect-") && m.returnTo == returnStoryDetail:
	return tabBacklog
case m.Screen == "defects" || strings.HasPrefix(m.Screen, "defect-"):
	return tabDefects
```

`crumbParts` antepone `Backlog › Detalle` a esas pantallas:

| Pantalla | Miga (tras `Proyectos › <proyecto> ›`) |
|---|---|
| `story-detail` | `Backlog › Detalle` |
| `section` (Poker, integrantes) y `planning-poker` | `Backlog › Detalle › Estimación › Participantes` |
| `section` (Esfuerzo, tareas) | `Backlog › Detalle › Esfuerzo › Tareas` |
| `section` (Esfuerzo, integrantes) | `Backlog › Detalle › Esfuerzo › Integrante` |
| `effort-form` | `Backlog › Detalle › Esfuerzo › Registrar` |
| `defect-create-form` / `defect-sprint-select` / `defect-create-confirm` con `returnStoryDetail` | `Backlog › Detalle › Registrar defecto` / `… › Elegir Sprint de detección` / `… › Confirmar registro` |
| Pantallas `defect-*` con `returnDefault` | `Defectos › …` (sin cambios) |

**Razón:** «Detalle» sigue el precedente de `Defectos › Detalle` y mantiene acotado el ancho de la miga. El título de la historia ya aparece en el cuerpo. El delta de spec exige que la miga empiece con «Proyectos», el proyecto y «Backlog»; el texto exacto del último tramo queda en el diseño.

### D9. Guardas de respuestas tardías: se conservan y se refuerza una

| Respuesta | Guarda actual | Cambio |
|---|---|---|
| `SectionResultMsg` | `Screen == "section"`, mismo `kind`, mismo `step`, `loading` | **Se agrega `StoryID`** y se compara con `m.section.selected.ID`. Con el detalle, pasar de una historia a otra cuesta `Esc`, `↓`, `Enter`, `t`. Sin esta comparación, una respuesta tardía de las tareas de la historia A se aplicaría al selector de la historia B (mismo `kind`, `step` y `loading`), y se registraría esfuerzo de B con una tarea de A |
| `pokerSaveResult` | Ninguna | Sin cambios. El doble envío de Planning Poker en la confirmación es un defecto preexistente, fuera de este cambio (ver Riesgos) |
| `effortSaveResult` | `saving && seq` | Sin cambios |
| `defectOptionsResult` / `defectCreateResult` | `projectID`, `Screen`, `requestID` | Sin cambios |
| `BacklogResultMsg` (lista) | `Screen == "backlog" && backlogLoading` | Acepta también `"story-detail"` (D5) |

### D10. Limpieza al cambiar de proyecto

**Elección:** `selectProject` reinicia `m.storyDetail = storyDetailState{}` y `m.returnTo = returnDefault`. `openTab` también reinicia `returnTo` como defensa, porque es la única otra manera de entrar a una raíz. `openStoryFlow` reinicia `m.effort = effortState{seq: m.effort.seq}` (antes lo hacía `openSection`) y `m.pokerVotes = map[string]string{}`, para que no queden marcas `Space` de un flujo abandonado en otra historia.

**Razón:** todo cambio de proyecto pasa por `selectProject` (no hay otro camino desde que se retiró `p`). Reiniciar en un único punto cubre el escenario «Cambiar de proyecto descarta el detalle y el destino de retorno».

## Flujo y máquina de estados

```
backlog ──Enter──▶ story-detail (storyID; relee backlogStories)
   ▲  e ─▶ backlog-form (sin cambios)      │
   └──────────── Esc (sin recargar) ───────┤
                                           │ p / Enter «Estimar»
                                           ├──▶ section/members (Poker) ──Enter──▶ planning-poker
                                           │       Esc ─▶ story-detail          Esc ─▶ section/members
                                           │                                    confirm: PUT
                                           │                                    éxito ─▶ story-detail + GET /stories
                                           │ t / Enter «Registrar esfuerzo»
                                           ├──▶ section/tasks ──Enter──▶ section/members ──Enter──▶ effort-form
                                           │       Esc ─▶ story-detail   Esc ─▶ section/tasks     Esc ─▶ section/members
                                           │                                                   201 ─▶ story-detail + aviso
                                           │ d / Enter «Reportar defecto»
                                           └──▶ defect-create-form (historia precargada)
                                                   Esc ─▶ story-detail
                                                   Enter (último campo) ─▶ defect-sprint-select ── Esc ─▶ defect-create-form
                                                                              elegir ─▶ defect-create-form ─Enter─▶ defect-create-confirm
                                                   confirm: n ─▶ story-detail · Esc ─▶ form · s ─▶ POST ─▶ éxito: story-detail + aviso

defects ──n──▶ defect-create-form (returnDefault): flujo actual sin cambios, vuelve a defects
```

### Interfaces

```go
// internal/tui/story_detail.go

// returnTarget is where a flow goes back to when it is cancelled from its first
// step or completes. The zero value keeps each flow's own destination.
type returnTarget int

const (
	returnDefault returnTarget = iota
	returnStoryDetail
)

// storyDetailState references the story by ID; the story itself is always read
// from m.backlogStories so the detail never shows stale data.
type storyDetailState struct {
	storyID string
	cursor  int // highlighted action row
}

// storyAction is one action of the detail: its row, its shortcut and its help
// come from this single table.
type storyAction struct {
	key, label, help string
	open             func(Model, Story) (Model, tea.Cmd)
}

var storyActions = []storyAction{
	{key: "p", label: "Estimar", help: "Estimar la historia con Planning Poker", open: openPokerForStory},
	{key: "t", label: "Registrar esfuerzo", help: "Registrar esfuerzo de la historia", open: openEffortForStory},
	{key: "d", label: "Reportar defecto", help: "Reportar un defecto de la historia", open: openDefectForStory},
}

func (m Model) openStoryDetail(stories []Story) (Model, tea.Cmd) // from the backlog; no command
func (m Model) detailStory() (Story, bool)
func (m Model) storyActionsAvailable() bool                     // not loading, no error, story found
func (m Model) updateStoryDetail(key tea.KeyMsg) (Model, tea.Cmd)
func (m Model) viewStoryDetail() string
func (m Model) storyDetailBindings(l *keyList)
func (m Model) backToStoryDetail() (Model, tea.Cmd)             // Screen = "story-detail"; keeps cursor
func (m Model) openStoryFlow(kind sectionKind, story Story) (Model, tea.Cmd)
func (m Model) beginDefectCreationForStory(story Story) (Model, tea.Cmd)
func (m Model) reloadBacklogInPlace() (Model, tea.Cmd)          // used by applyPokerSave

// Model gains: storyDetail storyDetailState; returnTo returnTarget.
// SectionResultMsg gains: StoryID string.
// pokerSaveResult is unchanged.
```

`openStoryFlow`: si `!storyActionsAvailable()`, no hace nada. Si no, asigna `m.section = sectionState{kind: kind, selected: storySelectorItem(story)}`, reinicia `effort` y `pokerVotes`, asigna `returnTo = returnStoryDetail` y `Screen = "section"`, y llama a `beginSectionStep(kind.firstStep())`. `updateSection`, rama `selectorBack`: si `step == kind.firstStep()`, ejecuta `backToStoryDetail()`; si es Esfuerzo en `sectionStepMembers`, vuelve a tareas (sin cambios). La vuelta genérica a `primary` desaparece.

### Registro de la pantalla nueva (lista de verificación)

| Punto | Archivo | Cambio para `story-detail` |
|---|---|---|
| `Update` (`switch m.Screen`) | `model.go:249` | `case "story-detail": return m.updateStoryDetail(key)` |
| `viewBody` | `model.go:593` | `case "story-detail": return m.viewStoryDetail()` |
| `crumbParts` | `model.go:727` | Caso propio y prefijo `Backlog › Detalle` para los flujos (D8) |
| `screenBindings` | `keys.go:91` | `case "story-detail": m.storyDetailBindings(l)` (D3) |
| `quitsOnQ` | `model.go:323` | **Agregar** `"story-detail"` |
| `entersText` | `model.go:313` | No agregar (sin campos de texto; `?` abre la ayuda) |
| `filtering` | `model.go:337` | No agregar (sin filtro) |
| `currentTab` | `workspace.go:68` | `tabBacklog` (D8) |
| `tabRoot` | `workspace.go:95` | No agregar (Tab y dígitos no actúan) |
| `helpAvailable` | `keys.go:54` | Sin cambios (true en el detalle) |

Bindings que cambian además del detalle:

| Pantalla | Primarias (pie, máximo 5) | Solo en la ayuda |
|---|---|---|
| `backlog` | `↑/↓ mover`, `Enter detalle`, `e editar`, `n crear`, `Esc proyectos` | `/ buscar`, `r refrescar` (se retiran `P` y `E`) |
| `story-detail` (listo) | `↑/↓ mover`, `Enter elegir acción`, `Esc backlog` | `p estimar`, `t esfuerzo`, `d defecto`, `?`, `q`, Ctrl+C |
| `story-detail` (cargando, error o ausente) | `Esc backlog` | `?`, `q`, Ctrl+C |
| `section` | Se retiran las ramas `primary` y `sprints` de `sectionBindings` | — |
| Raíces de pestaña | `1–4` automático (`len(workspaceTabs)`) | — |

`backlogHint` pasa a «Enter abre el detalle · e edita · n crea · r refresca · / busca · Esc vuelve a Proyectos.».

Teclas del Backlog: `e` deja de iniciar el filtro (pasa a ser tecla reservada, como `n` y `r`). `P` y `E` mayúsculas dejan de ser atajos e inician el filtro como cualquier otro carácter, según la regla uniforme.

## Flujo de datos

```
backlogStories (GET /stories, ya cargado) ──ID──▶ storyDetail.storyID
       ▲                                              │ detailStory() en cada render/tecla
       │                                              ▼
       │                                  story-detail (campos + acciones)
       │                                              │ openStoryFlow / beginDefectCreationForStory
       │                                              ▼  returnTo = returnStoryDetail
       │        section ──GET /members | /stories/{sid}/tasks (StoryID en la guarda)
       │        planning-poker ──PUT /stories/{sid}/story-points
       │              └ 200 ─▶ backToStoryDetail + reloadBacklogInPlace ──GET /stories──┘
       │        effort-form ──POST /stories/{sid}/effort-records (seq, saving)
       │              └ 201 ─▶ backToStoryDetail + aviso (sin recargar)
       │        defect-* ──GET /sprints/open, POST /defects (projectID, requestID)
       │              └ 2xx ─▶ backToStoryDetail + aviso (sin recargar)
selectProject ─▶ storyDetail = {}, returnTo = returnDefault
```

## Cambios de archivos

Las líneas autoras estimadas son adiciones más eliminaciones. Todas las cifras son estimaciones.

| Archivo | Acción | Descripción | Líneas |
|---|---|---|---|
| `internal/tui/story_detail.go` | Crear | `returnTarget`, `storyDetailState`, `storyActions`, apertura, relectura, update, vista con sus cuatro estados, bindings, `backToStoryDetail`, `openStoryFlow`, `beginDefectCreationForStory`, `reloadBacklogInPlace` | ~170 |
| `internal/tui/sections.go` | Modificar | Retiro de `sectionDefects`, `sectionStepSprints`, pasos `primary`, `preselected` y `openSection`; `sectionStepNone`; `firstStep`; `Esc` del primer paso → `backToStoryDetail`; `StoryID` en `SectionResultMsg` y en la guarda | ~130 (≈115 eliminadas) |
| `internal/tui/workspace.go` | Modificar | Cuatro pestañas, `openTab` sin Estimación ni Esfuerzo y reinicio de `returnTo`, `currentTab` por origen, `tabRoot` sin `section` | ~35 |
| `internal/tui/backlog_screens.go` | Modificar | `Enter` → detalle, `e` → modificar, retiro de `P`/`E` y de `openSectionForStory`, guarda de lista con `story-detail`, `backlogHint` | ~45 |
| `internal/tui/planning_poker.go` | Modificar | Éxito → detalle + recarga (`applyPokerSave` sin guarda nueva) | ~18 |
| `internal/tui/effort_form.go` | Modificar | Éxito → `backToStoryDetail` | ~8 |
| `internal/tui/defect_creation.go` | Modificar | `returnTo` en `beginDefectCreation`, `Esc` del formulario y del selector de Sprints, ayuda | ~20 |
| `internal/tui/defect_submission.go` | Modificar | `n` y éxito por `returnTo` | ~10 |
| `internal/tui/model.go` | Modificar | Campos nuevos; casos en `Update`, `viewBody`, `crumbParts` y `quitsOnQ` | ~35 |
| `internal/tui/keys.go` | Modificar | Bindings del Backlog y del detalle, `sectionBindings` sin `primary`/`sprints` | ~36 |
| `internal/tui/project_picker.go` | Modificar | Limpieza en `selectProject` | ~4 |
| **Producción** | | | **~509 (≈180 eliminadas)** |
| `tests/unit/tui/story_detail_test.go` | Crear | Detalle, acciones, atajos, estados, `Esc`, filtro, pie, ayuda, pestaña, miga, cambio de proyecto | ~300 |
| `tests/unit/tui/contextual_shortcuts_test.go` | Eliminar | Reemplazado por `story_detail_test.go` | ~212 |
| `tests/unit/tui/sections_test.go` | Modificar | Constantes 1–4; helper `openStoryFlowWithProject`; reescritura de las seis pruebas del paso `primary` sobre el primer paso | ~140 |
| `tests/unit/tui/sections_flows_test.go` | Modificar | 4 llamadas al helper; cadena de `Esc` | ~35 |
| `tests/unit/tui/planning_poker_test.go` | Modificar | Éxito → detalle + `GET /stories`; carga, error y ausencia en el detalle | ~50 |
| `tests/unit/tui/effort_test.go`, `effort_form_test.go`, `effort_result_test.go`, `toast_test.go` | Modificar | Helpers sin selector de historias; destino `story-detail`; reescritura de las pruebas de cambio de proyecto | ~160 |
| `tests/unit/tui/backlog_test.go` | Modificar | `selectStory` con `e`; prueba de lista vacía con `e`; `Enter` → detalle | ~25 |
| `tests/unit/tui/workspace_test.go`, `keys_test.go`, `dashboard_scope_test.go` | Modificar | Cuatro pestañas, `4`, Shift+Tab, `1–4` | ~45 |
| `internal/tui/sections_test.go` | Eliminar | Probaba `sectionDefects` | ~30 |
| `internal/tui/effort_form_test.go` | Modificar | `TestOpenSectionResetsEffortFlowKeepingSeq` → `openStoryFlow` | ~15 |
| `internal/tui/defect_from_story_test.go` | Crear | Precarga, salto del selector, destinos por `returnTo`, guardas, pestaña y miga | ~150 |
| `internal/tui/story_flow_test.go` | Crear | Tabla de `currentTab` y `crumbParts` por pantalla y origen; guarda `StoryID`; limpieza en `selectProject` | ~70 |
| **Pruebas** | | | **~1.230 (≈330 eliminadas)** |
| `README.md` | Modificar | Pestañas 1–4, fila `P`/`E` → detalle, tabla de secciones, párrafo de líneas 106-111 | ~30 |
| **Código, pruebas y README** | | | **~1.769 (≈1.770, estimado)** |

### Conciliación con la estimación de la propuesta

El rango de 750 a 950 líneas de la propuesta no se sostiene. El propio desglose de la exploración suma unas 1.240 líneas (420 de producción, 680 de pruebas y 140 de documentación). Esta estimación llega a unas 1.770 (cifra estimada) por tres motivos:

1. Cuenta las eliminaciones de pruebas (`contextual_shortcuts_test.go` completo e `internal/tui/sections_test.go`, unas 240 líneas).
2. Incluye las pruebas de los estados del detalle que agregó el delta de spec: carga, error, ausencia y guardas.
3. Las pruebas de cambio de proyecto de `effort_result_test.go` se reescriben completas.

Fuera de esa cuenta quedan los artefactos SDD que viajan en el mismo PR: el delta `specs/tui/spec.md` (unas 850 líneas), la propuesta, la exploración, este diseño y las tareas, además de la fusión en `openspec/specs/tui/spec.md` al archivar. Son documentación de proceso, no código para revisar línea a línea. El PR único con `size:exception` ya fue aceptado por el usuario.

## Estrategia de pruebas (TDD estricto)

Cada comportamiento empieza con una prueba en ROJO observada con `go test ./tests/unit/tui/... ./internal/tui/...` y cierra con `go test ./...` en verde. Las pruebas de flujo viven en `tests/unit/tui/` (paquete `tui_test`, `httptest`, sin Docker). Las que necesitan estado interno (`returnTo`, `currentTab`, `crumbParts`, guardas con mensajes construidos a mano) viven en `internal/tui/`, como las pruebas de defectos actuales.

### Helpers

- **`openStoryDetailWithProject(t, url, pid, storyIndex)`** (nuevo, `story_detail_test.go`): `openBacklogWithProject` + `↓` × `storyIndex` + `Enter`. Exige `Screen == "story-detail"` y comando `nil`.
- **`openStoryFlowWithProject(t, url, pid, storyIndex, actionKey)`** (reemplaza a `openSectionWithProject` en `sections_test.go`): el anterior más la tecla de acción, y `run` del comando del primer paso. Exige `Screen == "section"`. Los sitios que llamaban a `openSectionWithProject` y después elegían una historia con `↓`/`Enter` pierden esos pasos. `openEffortTasks(t, url, storyIndex)` y `openEffortMembers` conservan su firma y delegan en el helper nuevo, así que la mayoría de las pruebas de Esfuerzo no cambian.
- **`selectStory`** (`backlog_test.go:219`): pasa de `Enter` a `typeText(m, "e")`. Corrige de una vez todas las pruebas de modificación.
- Se reutilizan `apply`, `run`, `press`, `typeText`, `stripANSI`, `keyMsg`, `recorder`, `requestedPath`, `requestCount`, `sectionServer`, `sectionStories`, `storiesBodyForSection`, `effortFlowServer`, `fillEffortForm`, `saveMessage`, `confirmProjectInPicker` y `switchTab`.
- `switchEffortProject` (`effort_result_test.go:153`) abre la pestaña Backlog (`tabBacklog`) del proyecto nuevo, en lugar de Esfuerzo.

### Pruebas por pieza (ROJO primero)

| Pieza | Prueba en ROJO | Dónde |
|---|---|---|
| Apertura del detalle | `Enter` abre `story-detail` sin comando ni request; los siete campos se ven y los nulos muestran «—»; la lista vacía no abre; con filtro abre la historia visible; `Enter` mientras se filtra no abre | `story_detail_test.go` |
| `e` modifica | `e` abre `backlog-form` precargado; `e` no se escribe en el filtro | `backlog_test.go`, `story_detail_test.go` |
| `Esc` del detalle | Vuelve a `backlog` con el filtro y el cursor, sin request | `story_detail_test.go` |
| Acciones | Tabla `{tecla o Enter en fila} × {Poker, Esfuerzo, Defecto}` → pantalla, primer `GET` y título esperados; filas en orden; `↑`/`↓` acotados; `e` no hace nada | `story_detail_test.go` |
| Estados del detalle | Con `backlogLoading`, `backlogMessage` o una historia ausente, `p`/`t`/`d`/`Enter` no emiten comando y `Esc` vuelve | `story_detail_test.go` |
| `Esc` del primer paso | Integrantes de Poker y tareas de Esfuerzo vuelven a `story-detail`, también tras un error 500 | `story_detail_test.go`, `effort_test.go` |
| Éxito de Poker | `PUT` → `story-detail` + `GET /stories`; el detalle muestra los puntos nuevos | `planning_poker_test.go` |
| Éxito de Esfuerzo | 201 → `story-detail` con el aviso actual y sin `GET /stories` | `effort_form_test.go`, `effort_result_test.go`, `toast_test.go` |
| Alta desde la historia | `d` abre el formulario con la historia y sin `GET /stories`; `Enter` final → `GET /sprints/open`; `Esc` del selector de Sprints → formulario; `n` → detalle; `Esc` del formulario → detalle con el borrador descartado; éxito → detalle con aviso y sin `GET /defects`; `story_id` en el cuerpo | `internal/tui/defect_from_story_test.go` |
| Alta desde Defectos | Las pruebas existentes (`defect_create_test.go`, `defect_submission_test.go`, `defect_toast_test.go`) quedan sin cambios como red de regresión de `returnDefault`; se agrega una prueba de que `n` en Defectos reinicia un `returnTo` residual | `internal/tui/defect_from_story_test.go` |
| Pestaña y miga | Tabla de pantalla × origen → `currentTab` y miga | `internal/tui/story_flow_test.go` |
| Guarda `StoryID` | Un `SectionResultMsg` de tareas con otro `StoryID` no altera el selector | `internal/tui/story_flow_test.go` |
| Cambio de proyecto | Tras `selectProject`, `storyDetail` y `returnTo` vuelven al valor cero; flujo de punta a punta en `story_detail_test.go` (escenario de la spec) | Ambos |
| Pestañas | Barra con cuatro nombres; `4` → Defectos; Shift+Tab desde Defectos → Sprints; desde Dashboard → Defectos; ayuda con `1–4`; `5`/`6` no hacen nada | `workspace_test.go`, `keys_test.go` |
| Pie y ayuda | Pie del detalle ≤ 5 con `Enter` y `Esc` y `? más`; la ayuda lista `p`/`t`/`d` y no `Tab`/`1–4`; `q` sale y `?` abre la ayuda; el pie del Backlog no ofrece `P`/`E` | `story_detail_test.go` |
| Navegación sin mutación | Backlog → detalle → cada acción → `Esc` emite solo `GET` | `sections_flows_test.go` (`TestSectionNavigationOnlyEmitsGetRequests`) |

### Pruebas que se eliminan o reescriben

- **Se eliminan:** `contextual_shortcuts_test.go` completo, `internal/tui/sections_test.go`, `TestTabsRouteToEachSprint3Section` y `TestDefectsTabOpensProjectDefectList`. La segunda se reescribe con `tabDefects = "4"`.
- **Se reescriben sobre el primer paso:** `TestSectionAPIErrorStaysOpenAndRefreshes` (error y `r` en `GET /members`), `TestSectionEmptyCollectionShowsAnEmptyState` (proyecto sin integrantes), `TestSectionFilterRefreshAndEscape` (filtro en tareas; el segundo `Esc` vuelve al detalle, no a Proyectos) y `TestSectionFooterAndHelpOnlyShowValidKeys`.
- **Se ajustan:** `effort_test.go:400` (la cadena de `Esc` termina en el detalle), `effort_form_test.go:238` (miga `Backlog › Detalle › Esfuerzo › Registrar`) y los destinos de éxito en `effort_form_test.go:551`/`:703`/`:751`/`:782`, `effort_result_test.go:57` y `toast_test.go:206`.

## Matriz de amenazas

N/A: el cambio reorganiza pantallas dentro del proceso de la TUI contra la `API_URL` ya configurada. No hay comandos de shell, subprocesos, automatización de VCS/PR, clasificación de archivos ejecutables ni integración de procesos.

## Migración y despliegue

Sin migraciones. Revertir el PR restaura las seis pestañas, `Enter` = modificar, `P`/`E` y los destinos anteriores. Los datos creados con el cambio activo usan las mismas requests y siguen siendo válidos.

## Secuencia de implementación (commits por unidad de trabajo)

La entrega es UN solo PR con `size:exception`, aceptado por el usuario (decisión final). El PR no se abre todavía. Cada commit compila, incluye sus pruebas y deja `go test ./...` en verde. Los mensajes van en español, con el prefijo `type(scope)` en inglés y un cuerpo breve con la traza TDD. El orden mantiene las pestañas viejas hasta que el camino nuevo está completo, así que ningún commit deja acciones inaccesibles.

| # | Commit | Contenido | Líneas aprox. |
|---|---|---|---|
| 1 | `feat(tui): abrir el detalle de la historia desde el Backlog` | `story-detail` (campos, cuatro estados, `Esc`), registro en los switch paralelos, `Enter` → detalle, `e` → modificar, bindings y miga del detalle. `P`/`E` y las pestañas siguen. `selectStory` con `e`. | ~380 |
| 2 | `feat(tui): estimar y registrar esfuerzo desde el detalle` | `returnTo`, `storyActions` (`p`, `t`), `openStoryFlow`, `Esc` del primer paso, éxito de Poker (con recarga) y de Esfuerzo → detalle con `returnStoryDetail`; `returnDefault` conserva temporalmente los destinos de las pestañas; guarda `StoryID`; limpieza en `selectProject`; `currentTab`/miga por origen para `section`, `planning-poker` y `effort-form` | ~335 |
| 3 | `feat(tui): reportar un defecto desde el detalle con la historia precargada` | Acción `d`, `beginDefectCreationForStory`, destinos del alta por `returnTo`, `Esc` del selector de Sprints, ayuda del formulario, pestaña y miga `defect-*` por origen; `defect_from_story_test.go` | ~230 |
| 4 | `test(tui): recorrer los flujos de estimación y esfuerzo desde el detalle` | `openStoryFlowWithProject` y la migración de todas las llamadas a `openSectionWithProject`, con las pestañas viejas todavía presentes. Las pruebas pasan antes y después del commit 5 | ~300 |
| 5 | `refactor(tui): retirar las pestañas Estimación y Esfuerzo y el código muerto de secciones` | Cuatro pestañas; retiro de `openSection`, pasos `primary`, `sectionStepSprints`, `sectionDefects`, `preselected`, `openSectionForStory`, `P`/`E` y ramas `returnDefault` de Poker y Esfuerzo; `tabRoot`; eliminación de `contextual_shortcuts_test.go` e `internal/tui/sections_test.go`; pruebas de pestañas | ~420 |
| 6 | `docs(tui): actualizar el README para las acciones desde la historia` | README (pestañas 1–4, detalle y acciones) | ~30 |

Después vienen la verificación y el archivo SDD (fusión del delta en `openspec/specs/tui/spec.md`) en un commit `docs(tui)` propio, como en TUI-12.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| Una pantalla queda fuera de algún switch paralelo, de `quitsOnQ` o de `currentTab` | Lista de verificación de este diseño; prueba de tabla en `story_flow_test.go`; pruebas de `q`, `?` y pie en el detalle |
| `returnTo` residual hace que un alta desde Defectos vuelva al detalle | Asignación explícita en cada punto de entrada (`beginDefectCreation`, `openStoryFlow`, `beginDefectCreationForStory`), reinicio en `openTab` y `selectProject`; prueba dedicada |
| Doble envío de Planning Poker en la confirmación | Doble envío de Planning Poker en la confirmación: defecto preexistente, fuera de este cambio; se trata como arreglo aparte. |
| La recarga tras estimar falla o tarda | El detalle muestra los estados de carga y de error, y deja volver con `Esc` (escenarios de la spec) |
| El commit 5 concentra eliminaciones de producción y de pruebas | El commit 4 deja migradas las pruebas antes de borrar código, así que el commit 5 es mayormente eliminación verificable |
| Una respuesta tardía de la lista de otro proyecto se aplica al backlog nuevo (la respuesta no lleva `projectID`) | Riesgo anterior a este cambio y no agravado: la guarda solo suma `story-detail`. Queda fuera de alcance |

## Preguntas abiertas

- [ ] **S1 sin confirmar por el usuario:** la pestaña Backlog resaltada en el detalle y en los flujos que parten de él. El diseño lo implementa en un único lugar (`currentTab`), así que revertirlo es un cambio de pocas líneas.
