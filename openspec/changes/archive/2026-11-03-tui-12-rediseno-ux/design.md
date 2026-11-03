# Diseño: rediseño de la experiencia de usuario de la TUI (TUI-12)

## Contexto

`Model` (`internal/tui/model.go`) concentra todo el estado y el flujo se decide por `Screen string`. El pie (`footer`), la ayuda (`helpRows`) y la miga (`crumbParts`) son tres `switch` paralelos sobre `Screen`. El proyecto se elige a demanda mediante `openProjectPicker(purpose)` y `selectProject`, que reconstruye el flujo de origen con `pickerReturn`/`pickerPurpose`.

## Decisiones

### D1. Se conserva un único `Model` y los identificadores de pantalla

`Screen` sigue siendo el identificador observable de la pantalla activa: lo usan los manejadores y ~200 aserciones de `tests/unit/tui`. Extraer cada pantalla a un componente con estado propio reescribiría ~7000 líneas sin cambiar lo que ve el usuario. Este cambio extrae, en cambio, el **contrato de presentación** de cada pantalla (atajos, ayuda, miga) a declaraciones únicas; el estado por pantalla puede migrarse en cambios posteriores.

### D2. Registro de atajos por pantalla

`keys.go` define:

```go
type binding struct {
    key, desc string
    group     keyGroup // keyNav | keyAction | keyGeneral
    primary   bool     // candidato al pie
}
func (m Model) bindings() []binding
```

`footer()` toma los `primary` en orden (máximo cinco) y agrega `? más` si quedan atajos sin mostrar; `viewHelp()` agrupa todos los atajos por `group`. Así se eliminan los dos `switch` paralelos y el pie nunca contradice la ayuda.

### D3. Espacio de trabajo con pestañas en lugar de pila de navegación

La jerarquía nueva es fija: `Proyectos → pestaña → pasos del flujo`. Cada paso ya conoce su pantalla padre (los `Esc` actuales), y el único retorno dinámico era el del navegador de proyectos, que desaparece. Una pila genérica no aportaría comportamiento y agregaría estado a sincronizar; se reemplaza por:

```go
type workspaceTab int // tabDashboard … tabDefects
var workspaceTabs = []workspaceTab{tabDashboard, tabBacklog, tabSprints, tabEstimation, tabEffort, tabDefects}
func (m Model) openTab(t workspaceTab) (Model, tea.Cmd)
func (m Model) tabRoot() bool // la pantalla activa es la raíz de su pestaña
```

`Update` resuelve `Tab`/`Shift+Tab`/`1`–`6`/`Esc` de raíz antes de despachar a la pantalla, con las mismas guardas que hoy usa `?` (sin filtro en edición ni campo de texto). Se eliminan `pickerReturn`, `pickerPurpose`, el menú (`Choices`, `Cursor`, `updateMenu`, `viewMenu`) y los atajos `p` y `c`.

### D4. Arranque

`NewModel` deja `Screen = "project-picker"` con la carga en curso e `Init()` devuelve el `GET /projects`. Un indicador `autoOpened` evita reabrir el proyecto único después de la primera carga.

### D5. Avisos efímeros

`Model.toast` guarda el último aviso de éxito; `View` lo dibuja al inicio del panel con `okBlock`; `Update` lo limpia al recibir una tecla. No se usan temporizadores, por lo que el comportamiento es determinista en las pruebas. Cada `apply*Result` exitoso vuelve a la pantalla de origen y recarga su colección en lugar de abrir `*-result`; los errores quedan en el formulario.

### D6. Métricas dentro del Dashboard

`dashboardState` suma el ámbito (`sprintID`, `sprintLabel`). La pantalla `dashboard-sprint-select` reutiliza `selector` con «Proyecto completo» seguido de los Sprints abiertos. La carga del Dashboard consulta `metricsClient.Project` o `metricsClient.Sprint` según el ámbito, y `d` llama a `startReport` con el ámbito correspondiente. La sección `sectionMetrics` y sus pasos (`sectionStepScope`, `sectionStepMetrics`) se eliminan.

### D7. Atajos contextuales

`P`/`E` en el backlog abren la pestaña con `openSection(kind)`, fijan `section.selected` con la historia resaltada y llaman a `beginSectionStep` del paso siguiente. Se marca `section.preselected` para que `Esc` en ese paso vuelva a la lista de historias de la pestaña y la cargue.

## Riesgos

- **Volumen de pruebas afectadas**: las pruebas de caja negra arrancan desde el menú. Se agregan helpers (`openWorkspace`, `openTab`) y se migran por pestaña dentro de cada tarea.
- **Conflicto de teclas**: `Tab` es navegación de campos en los formularios; por eso el cambio de pestaña solo aplica en pantallas raíz (D3).

## Entrega

PRs encadenados, uno por tarea de `tasks.md`, cada uno con sus pruebas y documentación.
