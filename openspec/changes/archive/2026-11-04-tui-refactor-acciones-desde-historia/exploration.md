# Exploración: acciones desde la historia en la TUI (tui-refactor-acciones-desde-historia)

## Pedido

Las pestañas Estimación, Esfuerzo y Defectos se sienten duplicadas. Decisión del usuario a evaluar: la historia como centro; desde Backlog se elige una historia y en su detalle se ofrecen Estimar, Registrar esfuerzo y Reportar defecto; las pestañas quedan para Dashboard, Backlog y Sprints. No es una US nueva (TUI-01..TUI-13 archivadas): es un rediseño de navegación.

## Estado actual (evidencia)

### 1. Mapa de comportamiento

- `workspace.go:27` `workspaceTabs` = Dashboard, Backlog, Sprints, Estimación, Esfuerzo, Defectos; dígito = posición+1 (`tabForKey`, `workspace.go:106`). `openTab` (`workspace.go:49`): Estimación -> `openSection(sectionPlanningPoker)`; Esfuerzo -> `openSection(sectionEffort)`; Defectos -> `openDefects()`.
- **Hallazgo que corrige la premisa:** la pestaña Defectos NO es un selector de historias. `openDefects()` (`defects.go:74`) abre el listado propio de defectos del proyecto (`GET /projects/{pid}/defects`) con búsqueda, detalle (Enter), cambio de estado (`u`, con Sprint de resolución si queda resuelto) y alta (`n`). `sectionDefects` (`sections.go`) es código muerto salvo título/miga (`model.go:781`) y un test interno (`internal/tui/sections_test.go:17`). Solo Estimación y Esfuerzo son selector de historia + flujo.
- Enrutamiento (`model.go:237`): si `tabRoot() && !filtering()`, Tab/Shift+Tab/1-6 llaman `openTab`. `tabRoot()` (`workspace.go:95`) es true en dashboard, backlog, sprints, defects y en section con step primary para poker/esfuerzo. `currentTab()` (`workspace.go:68`) deriva la pestaña de `Screen` (prefijo backlog, `defect-*`, planning-poker, effort-form, section según kind). `inWorkspace()` = proyecto activo y pestaña != `tabNone`: de eso dependen la barra de pestañas y la miga (`model.go:696`, `801`).
- Backlog (`backlog_screens.go`): Enter abre `backlog-form` de edición directo; `n` alta; `/` buscar; `r` refrescar; `P`/`E` (mayúsculas) abren Estimación/Esfuerzo con historia preelegida. Esc limpia filtro y luego `leaveWorkspace`. Guardar -> backlog recargado + aviso.
- Estimación: section/primary (historias) -> section/members (multi-participante, Space) -> planning-poker (vote/reveal/revealed/estimate/confirm) -> `PUT story-points`. Esc: primary -> proyectos; luego un nivel atrás; con `preselected` y primer paso -> `openBacklog()` (recarga, resetea cursor/filtro). Éxito (`applyPokerSave`, `planning_poker.go:177`) -> backlog recargado.
- Esfuerzo: primary -> tasks (fila "Toda la historia" + tareas) -> members (simple) -> effort-form -> `POST effort-records`. effort-form Esc -> section. Éxito (`applyEffortSave`, `effort_form.go:219`) -> `openSection(sectionEffort)` = selector de historias de Esfuerzo, incluso si se llegó con `E` desde el backlog.
- Defectos: defects -> defect-detail (`u`) / `n` -> defect-create-form -> defect-story-select -> defect-sprint-select -> defect-create-confirm (s/n) -> `POST defects`. Todos los destinos hard-codeados a "defects" (`defect_creation.go:47`, `defect_submission.go:36` y `:79`).
- `openSectionForStory` (`backlog_screens.go:235`): `openSection(kind)`, `section.selected`, `section.preselected=true`, `beginSectionStep` (members para poker, tasks para esfuerzo). `preselected` solo afecta el Esc del primer paso (`sections.go:239`). `openSection` se llama desde `openTab`, `openSectionForStory` y `applyEffortSave`.
- Pie y ayuda (`keys.go`): `bindings()` es fuente única; `screenBindings` tiene un case por pantalla; en raíz de pestaña agrega Tab/Shift+Tab y 1-N con `len(workspaceTabs)` (`keys.go:71`). `footerLimit=5`.
- Cuatro switch paralelos que toda pantalla nueva debe tocar: `Update` (`model.go:249`), `viewBody` (`model.go:593`), `crumbParts` (`model.go:727`), `screenBindings` (`keys.go:91`), más `quitsOnQ`, `entersText`, `filtering` (`model.go:311-347`) y `currentTab`.

### 2. Detalle de historia hoy

No existe. Enter en backlog abre el formulario de edición. No hay `GET /projects/{pid}/stories/{sid}` (`api.go:230-241` solo POST, PUT y GET de colección). `Story` (`backlog.go:10`) trae Title, Description, Priority, Status, StoryPoints, AcceptanceCriteria, EstimatedHours: un detalle se construye sin requests nuevos desde `m.backlogStories`. Tareas: `GET .../stories/{sid}/tasks` existe. Defectos de la historia: `GET defects` + filtro cliente por `story_id`. Registros de esfuerzo: NO se puede (`esfuerzo/spec.md:552` prohíbe consultarlos).

Necesario para alojar las 3 acciones:

1. Pantalla `story-detail` con la historia referenciada por ID y releída de `backlogStories`.
2. Lista de acciones (selector reutilizado) y/o atajos.
3. Generalizar `preselected` en un destino de retorno `returnTo` (Esc de primeros pasos, éxito de esfuerzo y poker, cancelar/confirmar/éxito del alta de defecto).
4. Alta de defecto con historia precargada que salte `defect-story-select`.
5. Mapeo de pestaña por origen (`defect-*` hoy resalta Defectos por prefijo).

### 3. Specs afectadas

Deltas concentrados en `openspec/specs/tui/spec.md`. `esfuerzo/spec.md:550-575` solo exige no ampliar el contrato HTTP y no consultar registros (su escenario 571 dice "pantalla de resultado", desactualizado, no bloqueante). `reporte/spec.md:150` habla de Dashboard y lista de Sprints (sin cambio). `defecto/spec.md` sin requisitos de navegación TUI.

MODIFIED:

1. Espacio de trabajo del proyecto con pestañas (lista, teclas 1-6, escenarios Elegir proyecto abre Dashboard, Cambiar de pestaña con Tab y números, Shift+Tab desde la primera).
2. Atajos contextuales de estimación y esfuerzo (P/E; probablemente REMOVED+ADDED).
3. Retroceder en el flujo de esfuerzo y descartar respuestas tardías.
4. Elegir historia, tarea e integrante para registrar esfuerzo.
5. Enviar el registro de esfuerzo y presentar los errores de la API (destino del 201, escenario Corregir y reenviar).
6. Avisos efímeros en lugar de pantallas de resultado (destinos esfuerzo/defecto/poker; escenario Registrar esfuerzo vuelve a la pestaña Esfuerzo).
7. Estimar historias con Planning Poker.
8. Navegación de solo lectura (escenario Navegar por las secciones del Sprint 3 no muta).
9. Operar el backlog mediante la TUI (Enter modifica).
10. Modificar historias desde la lista (redacción).
11. Listas navegables con teclas uniformes (dígitos 1-6; `e` modifica donde editar no es primaria).
12. Elegir integrantes, historias, tareas y Sprints con selectores navegables (escenario Los cuatro tipos de selector).
13. Pie de teclas acotado y ayuda agrupada (escenario Teclas de pestaña en la ayuda 1-6).
14. Salir limpiamente (opcional).

ADDED: Detalle de la historia con sus acciones; Reportar un defecto desde la historia. El flujo de Defectos no tiene requisito propio en `tui/spec.md` (solo menciones en líneas 63, 619, 1042).

### 4. Tests afectados

- `tests/unit/tui/sections_test.go` (232): constantes `tabEstimation=4`/`tabEffort=5`/`tabDefects=6`; helper `openSectionWithProject` concentra ~25 sitios de llamada (sections_flows, planning_poker, effort, effort_result, effort_form, sections): cambiar el helper (backlog -> detalle -> acción) arregla casi todos. Reescribir `TestTabsRouteToEachSprint3Section` y `TestDefectsTabOpensProjectDefectList`.
- `tests/unit/tui/workspace_test.go` (267): Tab/6/Shift+Tab, miga Esfuerzo, barra con 6 nombres; ~4 tests a reescribir.
- `tests/unit/tui/contextual_shortcuts_test.go` (212): reescribir entero.
- `tests/unit/tui/backlog_test.go` (1107, 32 refs backlog-form/KeyEnter): si Enter pasa al detalle, ~10-15 tests necesitan paso extra (estimado, no verificado).
- toast_test, dashboard*_test, sprint_test (dígitos 1-3): sin cambio.
- `internal/tui/defects_test.go` y `defect_*_test.go`: sin cambio si Defectos sigue como pestaña; adaptar destinos de retorno y sumar tests de precarga.
- `internal/tui/sections_test.go` (sectionDefects): se elimina. `internal/tui/effort_form_test.go` y `tests/unit/tui/effort_result_test.go`: destino del éxito.
- `README.md` (líneas 42-55, 97-106, 124) documenta pestañas, 1-6 y P/E.

## Opciones

| Opción | Qué hace | Líneas estimadas |
|---|---|---|
| A | Quitar las tres pestañas. Pros: barra mínima, máxima coherencia con la decisión. Contras: pierde/reubica listado y estado de defectos, que no es duplicado. | 950-1200 |
| A' | Quitar Estimación y Esfuerzo; Defectos queda como pestaña de vista propia y además se reporta desde el detalle. Pros: elimina la duplicación real sin perder funcionalidad; dígitos 1-4. Contras: 4 pestañas, no 3; alta de defecto con dos entradas. | 750-950 |
| B | Tableros de trabajo pendiente. Inviable sin backend (no hay listado de esfuerzo, prohibido consultarlo; N+1). | no viable |
| C | Híbrido aditivo, pestañas intactas. Pros: bajo riesgo, incremental. Contras: no resuelve la queja, dos caminos para lo mismo. | 450-550 |

Desglose A' (estimado, producción/tests): detalle ~130/120; backlog Enter/e y retiro P/E ~40/80; `returnTo` y quitar primary/sectionDefects ~70 (+~100 borradas)/100; alta defecto precargada ~50/70; quitar pestañas ~30/60; reescritura de tests existentes ~250; spec+README ~140. Total ~320 prod (+100 borradas) + ~820 tests/docs.

Presupuesto: 400/PR superado; la config tiene `delivery_strategy: single-pr`. Requiere `size:exception` o cadena. Corte natural: (1) detalle + acciones sobre secciones existentes; (2) retorno al detalle desde esfuerzo/poker y alta de defecto precargada; (3) retirada de pestañas, código muerto, spec y README.

## Recomendación

A', entregada en cadena de PR, con el detalle como pantalla propia y lista de acciones seleccionables. Quita exactamente lo duplicado (selectores de historia de Estimación/Esfuerzo, sin vista propia), no pierde funcionalidad y reutiliza `preselected`/`openSectionForStory`/selectores/guardas ya probados. Si el usuario insiste en sacar las tres, A es viable tras decidir dónde viven los defectos.

## Riesgos

- Enter en backlog cambia de significado (hoy edita): rompe memoria muscular y parte de `backlog_test.go`; alternativa Enter=detalle y `e`=editar (consistente con Proyectos).
- Destinos hard-codeados (`defects`, `openSection(sectionEffort)`, `backlog`).
- Estado viejo del detalle tras estimar/editar: releer por ID; con filtro el cursor es índice sobre lista filtrada.
- `currentTab`/miga: story-detail, section, planning-poker, effort-form deben mapear a Backlog; `defect-*` por prefijo resalta Defectos aun entrando desde la historia.
- Cuatro switch paralelos más quitsOnQ/entersText/filtering.
- Respuestas tardías: conservar guardas (seq de esfuerzo, poker, requestID de defecto).
- Cambio de proyecto debe descartar detalle y `returnTo`.
- Más pasos para registrar esfuerzo que el atajo `E` directo.
- Presupuesto 400 superado con single-pr.
- ~15 requisitos de la spec tui tocados; riesgo de referencias sueltas a "pestaña Estimación/Esfuerzo".

## Preguntas de producto abiertas

1. Defectos: ¿se mantiene la pestaña (listado, detalle, estado) además de Reportar defecto en el detalle, o se elimina y reubica? (la premisa de que es otro selector de historias no es exacta).
2. Acciones del detalle: ¿filas seleccionables con Enter, atajos de una tecla, o ambos?
3. ¿Enter en backlog abre el detalle y `e` edita, o Enter sigue editando?
4. ¿Se conservan P y E en la lista del backlog?
5. ¿Qué muestra el detalle además de los campos: tareas, defectos asociados, ninguno? (esfuerzo no se puede).
6. Tras registrar esfuerzo/estimar/reportar defecto, ¿se vuelve al detalle (recomendado) o al backlog?
7. ¿La barra resalta Backlog en el detalle y en los flujos de acción?
8. Entrega: ¿cadena de PR (recomendado) o un PR con `size:exception`?
9. ¿Se agrega a la spec un requisito propio del flujo de Defectos?

## Listo para propuesta

Sí. Resolver primero la pregunta 1 (Defectos) y la 8 (entrega).
