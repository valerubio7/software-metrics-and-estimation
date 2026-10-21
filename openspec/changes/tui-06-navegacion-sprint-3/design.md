# Diseño: navegación del Sprint 3 en la TUI (TUI-06)

## Contexto

La TUI usa Bubble Tea (`github.com/charmbracelet/bubbletea` v1.3.10) y consume exclusivamente la API HTTP. El rediseño #98 introdujo:

- `Model` con `activeProjectID`/`activeProjectName` como contexto de sesión.
- Chrome persistente: `chrome()` (barra superior + miga + regla), `card()` (panel) y `footer()`.
- Ayuda global con `?`.
- Navegador buscable de proyectos (`project-picker`) reutilizado por Backlog y Sprints.
- Patrón de listas navegables: `/`, `↑/↓`, `Enter`, `Esc`, `r`, `?`.

El menú actual tiene cuatro opciones y `updateMenu` solo rutea Proyectos, Backlog y Sprints. La especificación `tui` describe ese menú de cuatro opciones.

Las lecturas existentes alcanzan para historias (`GET /projects/{pid}/stories`) y Sprints abiertos (`GET /projects/{pid}/sprints/open`), pero no para integrantes ni tareas:

- `project_members` solo tiene `POST /projects/{pid}/members`.
- `tasks` solo tiene `POST /projects/{pid}/sprints/{sid}/stories/{sid}/tasks`.

TUI-06 necesita ambos listados para que los selectores de integrante y tarea sean reales y no componentes aislados sin datos.

## Enfoque

### 1. Selector compartido

Nuevo `internal/tui/selector.go` con un tipo no exportado `selector`:

```go
type selectorItem struct {
    ID    string
    Label string
    Meta  string
}

type selector struct {
    items     []selectorItem
    filter    string
    filtering bool
    cursor    int
    loading   bool
    err       string
}
```

Responsabilidades:

- `visible()` filtra por `Label`, `Meta` e `ID` sin distinguir mayúsculas.
- `update(key)` implementa el patrón común y devuelve una acción:
  - `selectorIdle`: la tecla fue consumida (filtro, movimiento, etc.).
  - `selectorChosen`: `Enter` sobre una fila.
  - `selectorRefresh`: `r`.
  - `selectorBack`: `Esc` sin filtro.
- `startLoading()`, `fail(message)`, `setItems(items)`.
- El filtro se activa con `/` o con cualquier carácter imprimible; `Esc` lo limpia antes de volver.
- Mientras `filtering` es verdadero, `?`, `q` y `p` se tratan como texto. El `Model` ya centraliza esa decisión en `filtering()`.

La vista se implementa como método del `Model` (`viewSelector`) para reutilizar `row()`, `searchLine()`, `emptyState()` y `errorBlock()`. No se refactorizan las listas existentes: el componente se usa en las pantallas nuevas y queda disponible para TUI-07..11.

### 2. Shells de sección

Cinco secciones nuevas, todas con la misma estructura:

| Sección | Colección principal | Segundo paso | Aviso final |
|---------|---------------------|--------------|-------------|
| Planning Poker | Historias del backlog | Integrantes del proyecto | TUI-07 |
| Esfuerzo | Historias del backlog | Tareas de la historia | TUI-08 |
| Defectos | Historias del backlog | Sprints abiertos | TUI-09 |
| Métricas | Sprints abiertos | — | TUI-10 |
| Dashboard | Sprints abiertos | — | TUI-11 |

Estados:

```go
type sectionKind string // planning-poker, effort, defects, metrics, dashboard

type sectionStep int // primary, members, tasks, sprints, notice

type sectionState struct {
    kind     sectionKind
    step     sectionStep
    selector selector
    primary  selector
    selected selectorItem
    notice   string
}
```

Flujo:

1. El usuario elige una opción del menú.
2. Si no hay proyecto activo, se abre `project-picker` con propósito `section:<kind>`; al confirmar un proyecto se abre la sección.
3. La sección carga su colección principal con GET y muestra el selector.
4. `Enter` sobre la fila resaltada avanza al segundo paso cuando corresponde; si no, muestra el aviso de la historia futura.
5. `Esc` vuelve un nivel (aviso → lista, segundo paso → principal, principal → menú), limpiando primero el filtro.
6. `r` refresca el paso actual.
7. `p` abre el navegador de proyectos y, al confirmar, vuelve a la misma sección.

Los avisos son de solo lectura: no hay botones de creación, edición, cierre ni envío. El texto indica explícitamente qué historia implementará el flujo.

### 3. Ruteo y menú

`Model.Choices` pasa a:

```go
[]string{
    "Proyectos", "Backlog", "Sprints", "Planning Poker",
    "Esfuerzo", "Defectos", "Métricas", "Dashboard", "Salir",
}
```

`updateMenu` rutea por índice:

| Índice | Acción |
|--------|--------|
| 0 | `openProjectPicker("projects")` |
| 1 | `openBacklog()` |
| 2 | `enterSprints()` |
| 3 | `openSection(sectionPlanningPoker)` |
| 4 | `openSection(sectionEffort)` |
| 5 | `openSection(sectionDefects)` |
| 6 | `openSection(sectionMetrics)` |
| 7 | `openSection(sectionDashboard)` |
| 8 | `tea.Quit` |

`selectProject` reconoce `purpose` con prefijo `section:` y abre la sección correspondiente después de fijar el proyecto activo.

La pantalla nueva se llama `section`; la sección concreta vive en `Model.section.kind`. Esto evita cinco ramas de ruteo casi idénticas y permite que el pie, la miga y la ayuda se calculen por `kind`/`step`.

### 4. Mensajes y concurrencia

Nuevo mensaje:

```go
type SectionResultMsg struct {
    Kind     sectionKind
    Step     sectionStep
    Response APIResponse
    Err      error
}
```

`applySectionResult` solo aplica el resultado si:

- `Screen == "section"`.
- `section.kind == result.Kind`.
- `section.step == result.Step`.
- El selector está en `loading`.

Así se descartan respuestas tardías después de volver al menú, cambiar de paso o cambiar de proyecto. El error de transporte y el error HTTP se muestran dentro del selector (`selector.err`) sin cerrar la TUI, reutilizando `describeBacklogFailure` para el formato 422/404/otros.

### 5. Lecturas nuevas de la API

#### Integrantes

`GET /projects/{project_id}/members`

Respuesta `200`:

```json
{
  "project_id": "uuid",
  "members": [
    {"id": "uuid", "project_id": "uuid", "full_name": "Ada Lovelace", "email": "ada@example.com"}
  ]
}
```

- Lista vacía: `{"project_id":"...","members":[]}`.
- UUID inválido: `422 validation_failed` con `fields.project_id`.
- Proyecto inexistente: `404 project_not_found`.
- Error inesperado: `500 internal_error` sin detalle interno.
- Solo `GET`/`HEAD`; otro método: `405 method_not_allowed`.

Cadena: `ListMembersUseCase` → `MemberLister` → `PostgresMemberRepository.ListByProject` → `ListMembersHandler` → ruta compuesta en `internal/api`.

#### Tareas de una historia

`GET /projects/{project_id}/stories/{story_id}/tasks`

Respuesta `200`:

```json
{
  "project_id": "uuid",
  "story_id": "uuid",
  "tasks": [
    {"id": "uuid", "project_id": "uuid", "sprint_id": "uuid", "story_id": "uuid", "title": "Diseñar", "estimated_hours": 4.5}
  ]
}
```

- Orden: `tasks.seq` ascendente (orden de creación).
- Lista vacía: `tasks: []`.
- UUID inválido: `422 validation_failed` con `fields.project_id` o `fields.story_id`.
- Proyecto inexistente: `404 project_not_found`.
- Historia inexistente o de otro proyecto: `404 story_not_found`.
- Error inesperado: `500 internal_error` sin detalle interno.
- Solo `GET`/`HEAD`; otro método: `405 method_not_allowed`.

Cadena: `ListTasksUseCase` → `TaskLister` → `PostgresTaskRepository.ListByProjectStory` → `ListTasksHandler` → ruta compuesta en `internal/api`.

No se agregan migraciones: `project_members` (000006) y `tasks` (000009) ya existen. La composición sigue el patrón actual: `MemberDependencies.Lister` y `TaskDependencies.Lister` son opcionales; si son `nil`, la ruta no se registra.

### 6. Presentación y ergonomía

- **Título**: nombre de la sección.
- **Subtítulo**: proyecto activo y propósito del paso.
- **Búsqueda**: `searchLine` existente.
- **Filas**: `m.row()` para que la selección ocupe todo el ancho.
- **Vacío**: `emptyState` con un mensaje específico por colección.
- **Error**: `m.errorBlock` con el detalle de la API.
- **Aviso final**: bloque de texto con el nombre de la historia futura, sin acciones.
- **Pie**: `m.keyHints` con pares válidos por paso; el wrapping de #98 se conserva.
- **Ayuda**: `viewHelp` devuelve filas específicas para `section`; no muestra `n`, `e`, `x`, `h` ni `Space` porque no aplican.
- **Miga**: `Inicio › <Sección> › <Paso> › Proyecto: <nombre>`.
- **Ancho**: se usan `contentWidth`, `innerWidth`, `card` y `keyHints` existentes; no se agregan layouts propios que puedan desbordar en 80 o 44 columnas.

### 7. Teclado

| Tecla | Comportamiento |
|-------|----------------|
| `/` o carácter imprimible | Activa el filtro (el carácter inicial se conserva) |
| `↑` / `↓` | Mueve el cursor dentro de la lista visible |
| `Enter` | Acción primaria de la fila |
| `Esc` | Limpia el filtro; si no hay filtro, vuelve un nivel |
| `r` | Refresca el paso actual |
| `?` | Abre/cierra la ayuda (solo fuera del filtro) |
| `p` | Cambia el proyecto activo |
| `q` | Sale (solo fuera del filtro) |
| `Ctrl+C` | Sale siempre |

`quitsOnQ` incorpora `section`. `filtering()` incorpora el selector de sección. `entersText` no cambia: la sección no tiene campos de texto propios.

## Decisiones de arquitectura

### Decisión: selector no exportado y reutilizado por composición

**Elección**: `selector` vive en el paquete `tui`, no se exporta y se compone dentro de `sectionState`.
**Alternativas**: exportar un componente público (API innecesaria para un paquete interno); refactorizar todas las listas existentes (riesgo alto y fuera de alcance).
**Justificación**: el paquete ya contiene toda la presentación; el componente se prueba de forma interna y las pantallas nuevas lo reutilizan sin romper Backlog/Sprints.

### Decisión: una pantalla `section` con `kind` y `step`

**Elección**: una sola pantalla y un estado con discriminador.
**Alternativas**: cinco pantallas y cinco ramas de update/view (duplicación); un submenú por sección (contradice el issue).
**Justificación**: el 80 % del comportamiento es común; el `kind` solo cambia títulos, fuente de datos y textos de aviso. El `step` modela los selectores encadenados.

### Decisión: incluir las lecturas de integrantes y tareas en TUI-06

**Elección**: agregar los dos GET en esta historia.
**Alternativas**: dejar los selectores sin datos hasta TUI-08 (incumple el criterio de aceptación y obliga a retrabajo).
**Justificación**: son lecturas de navegación, no rutas de negocio de US-14/US-15; desbloquean TUI-07 y TUI-08 y permiten probar los selectores de punta a punta.

### Decisión: shells con aviso en lugar de flujos funcionales

**Elección**: los segundos pasos terminan en un aviso “disponible en TUI-0X”.
**Alternativas**: implementar ya la votación, el registro de esfuerzo o el alta de defectos (fuera de alcance y duplicaría TUI-07..11).
**Justificación**: TUI-06 debe demostrar navegación y selectores, no reglas de negocio. El aviso deja explícito el límite y evita que un usuario crea que la operación quedó guardada.

## Contratos de prueba

### API

- `tests/unit/projectmember/application/list_members_test.go`: UUID inválido, lista vacía, error del puerto, orden delegado.
- `tests/unit/projectmember/http/list_handler_test.go`: 405, 422, 404, 200 con `[]`, error interno sin detalle.
- `tests/unit/task/application/list_tasks_test.go`: UUIDs inválidos, lista vacía, error del puerto.
- `tests/unit/task/transport/http/list_handler_test.go`: 405, 422, 404 proyecto/historia, 200 con `[]`, error interno.
- `tests/unit/cmd/api/*`: composición opcional de ambas rutas.

### TUI

- `tests/unit/tui/sections_test.go`: menú de nueve opciones, ruteo, carga de historias/Sprints, errores de API, estados vacíos, `Esc`/`r`/`p`, sin mutaciones.
- `tests/unit/tui/selectors_test.go`: filtro, cursor, `Enter`, `Esc`, `r`, `?` y los cuatro tipos de elemento.
- `tests/unit/tui/section_flows_test.go`: Planning Poker (historia → integrante → aviso), Esfuerzo (historia → tarea → aviso), Defectos (historia → Sprint → aviso), Métricas/Dashboard (Sprint → aviso).
- `tests/unit/tui/footer_test.go`: pie y ayuda específicos de sección.
- Pruebas internas `internal/tui/selector_test.go` si hace falta cubrir el componente sin pasar por `Model`.

## Riesgos y mitigaciones

| Riesgo | Mitigación |
|--------|------------|
| El selector duplica el comportamiento de listas existentes | Se limita a las pantallas nuevas y se prueba por comportamiento |
| Los shells parecen funcionales | Aviso explícito + ausencia de requests POST/PUT |
| Respuestas tardías | `Kind`/`Step` + `loading` en `applySectionResult` |
| Rutas nuevas sin migración | Composición opcional y error visible |
| PR grande | `size:exception`, commits RED/GREEN por unidad |

## Plan de reversión

Revertir el merge del PR. No hay migraciones ni cambios de datos. Las rutas GET desaparecen y el menú vuelve a cuatro opciones.
