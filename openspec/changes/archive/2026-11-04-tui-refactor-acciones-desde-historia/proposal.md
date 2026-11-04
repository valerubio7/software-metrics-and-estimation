# Propuesta: Acciones desde la historia en la TUI (rediseño de navegación)

## Intención

Las pestañas Estimación y Esfuerzo duplican lo que ya ofrece Backlog: ambas abren un selector de historias que solo sirve como puerta de entrada a un flujo, sin vista propia. El usuario percibe la barra de seis pestañas como redundante y tiene que elegir la misma historia por caminos distintos según la acción.

Este cambio pone la historia en el centro: desde Backlog se abre el detalle de una historia y desde ahí se ofrecen Estimar, Registrar esfuerzo y Reportar defecto. No es una historia de usuario nueva (TUI-01 a TUI-13 están archivadas): es un rediseño de navegación de una capacidad terminada, sin cambios en la API ni en el contrato HTTP.

La exploración corrigió la premisa inicial: la pestaña Defectos no es un selector de historias, sino una vista propia (listado, detalle, cambio de estado y alta). Por eso se conserva.

## Decisiones confirmadas por el usuario

Estas decisiones están cerradas y son restricciones de este cambio; no se reabren en las fases siguientes.

| # | Tema | Decisión |
|---|------|----------|
| 1 | Enfoque (A') | Se eliminan las pestañas Estimación y Esfuerzo. Defectos se mantiene como pestaña con su vista propia. La barra pasa de 6 a 4 pestañas: Dashboard, Backlog, Sprints, Defectos (dígitos 1–4). |
| 2 | Detalle de la historia | Pantalla nueva con las acciones Estimar, Registrar esfuerzo y Reportar defecto. |
| 3 | Teclas del Backlog | `Enter` sobre una fila abre el detalle; `e` modifica la historia (reemplaza el `Enter` = modificar actual). |
| 4 | Acciones del detalle | Filas seleccionables con `Enter` y, además, un atajo de una tecla por acción. |
| 5 | Atajos `P` y `E` | Se eliminan de la lista del Backlog, junto con el camino `preselected`, que se reemplaza por un destino de retorno. |
| 6 | Contenido del detalle | Solo los campos de la historia: título, descripción, prioridad, estado, story points, criterios de aceptación y horas estimadas. Sin tareas, defectos ni registros de esfuerzo. Sin requests ni endpoints nuevos: el detalle se construye desde la lista del backlog ya cargada y se relee por ID de historia. |
| 7 | Destino tras cada acción | Tras estimar, registrar esfuerzo o reportar un defecto, y al cancelar (`Esc`) esos flujos, el usuario vuelve al detalle de la historia. Los destinos hoy fijos se generalizan en un destino de retorno. |
| 8 | Alta de defecto desde la historia | Llega con la historia precargada y omite `defect-story-select`. |
| 9 | Alcance de la spec de Defectos | Solo se documenta el alta de defecto desde la historia; no se escribe un requisito completo del flujo de Defectos. |
| 10 | Entrega | Un solo PR con `size:exception` (estrategia `single-pr`, excepción aceptada por el usuario). El PR no se abre todavía. |
| 11 | Documentación | `README.md` se actualiza en el mismo cambio; no se crea una carpeta de documentación nueva. |

## Supuestos a confirmar

| # | Tema | Supuesto |
|---|------|----------|
| S1 | Pestaña resaltada | La barra de pestañas resalta Backlog mientras el usuario está en el detalle de la historia y en los flujos de acción que parten de él (Planning Poker, esfuerzo y alta de defecto iniciada desde la historia). Este supuesto no fue consultado al usuario; la fase de spec lo toma como valor por defecto salvo indicación contraria. |

## Alcance

### Incluido
- Pantalla `story-detail` con los campos de la decisión 6, releída de `backlogStories` por ID de historia, y lista de acciones seleccionables con atajos de una tecla.
- Backlog: `Enter` abre el detalle; `e` abre el formulario de modificación; se retiran `P` y `E`.
- Destino de retorno (`returnTo`) que reemplaza a `preselected` y a los destinos fijos: `Esc` del primer paso de estimación y esfuerzo, éxito de Planning Poker y de esfuerzo, y cancelar, confirmar o éxito del alta de defecto.
- Alta de defecto con la historia precargada desde el detalle, saltando `defect-story-select`; el alta desde la pestaña Defectos conserva su flujo actual.
- Retiro de las pestañas Estimación y Esfuerzo, de los pasos `primary` (selector de historias) de ambos flujos y del código muerto `sectionDefects`.
- Mapeo de pestaña por origen (S1) en `currentTab`, la miga de pan y la barra.
- Registro de la pantalla nueva en los switch paralelos (`Update`, `viewBody`, `crumbParts`, `screenBindings`) y en `quitsOnQ`, `entersText` y `filtering`; pie y ayuda con dígitos 1–4.
- Descarte del detalle y del destino de retorno al cambiar de proyecto; conservación de las guardas de respuestas tardías existentes.
- Reescritura de las pruebas afectadas y pruebas nuevas con TDD estricto en `tests/unit/tui/` e `internal/tui/` (sin Docker).
- Deltas en `openspec/specs/tui/spec.md` y actualización del `README.md`.

### Fuera de alcance
- Cualquier endpoint o request HTTP nuevo, incluido un `GET` de historia individual o de registros de esfuerzo (la spec de `esfuerzo` prohíbe consultarlos).
- Mostrar tareas, defectos asociados o esfuerzo en el detalle de la historia.
- Cambios en la vista propia de Defectos (listado, detalle, cambio de estado, alta desde la pestaña), salvo la generalización de sus destinos de retorno.
- Un requisito completo del flujo de Defectos en la spec `tui`.
- Cambios en Dashboard, Sprints o en el contrato de las specs `esfuerzo`, `defecto` y `reporte`.
- Abrir el PR (se decide por separado).

## Capacidades

### Capacidades nuevas
- Ninguna.

### Capacidades modificadas
- `tui` (`openspec/specs/tui/spec.md`):
  - ADDED «Detalle de la historia con sus acciones»: apertura desde Backlog con `Enter`, campos mostrados, acciones seleccionables y atajos, relectura por ID, vuelta al detalle tras cada acción y al cancelar, pestaña resaltada (S1), descarte al cambiar de proyecto.
  - ADDED «Reportar un defecto desde la historia»: alta con historia precargada que omite el selector de historias; cancelar, confirmar y éxito vuelven al detalle.
  - REMOVED «Atajos contextuales de estimación y esfuerzo» (`P`/`E`); su contenido vigente se reemplaza por los requisitos ADDED.
  - MODIFIED «Espacio de trabajo del proyecto con pestañas»: cuatro pestañas, dígitos 1–4; escenarios «Elegir proyecto abre Dashboard», «Cambiar de pestaña con Tab y números» y «Shift+Tab desde la primera».
  - MODIFIED «Operar el backlog mediante la TUI» y «Modificar historias desde la lista»: `Enter` abre el detalle y `e` modifica.
  - MODIFIED «Listas navegables con teclas uniformes»: dígitos 1–4; `e` modifica en Backlog.
  - MODIFIED «Estimar historias con Planning Poker»: se inicia desde el detalle; éxito y `Esc` del primer paso vuelven al detalle.
  - MODIFIED «Elegir historia, tarea e integrante para registrar esfuerzo»: la historia llega desde el detalle; desaparece el selector de historias.
  - MODIFIED «Retroceder en el flujo de esfuerzo y descartar respuestas tardías»: el `Esc` del primer paso vuelve al detalle; se conservan las guardas.
  - MODIFIED «Enviar el registro de esfuerzo y presentar los errores de la API»: el `201` vuelve al detalle; escenario «Corregir y reenviar».
  - MODIFIED «Avisos efímeros en lugar de pantallas de resultado»: destinos de esfuerzo, Planning Poker y alta de defecto desde la historia; el escenario que vuelve a la pestaña Esfuerzo pasa a volver al detalle.
  - MODIFIED «Navegación de solo lectura»: el escenario de navegación por las secciones deja de nombrar las pestañas Estimación y Esfuerzo.
  - MODIFIED «Elegir integrantes, historias, tareas y Sprints con selectores navegables»: escenario de los cuatro tipos de selector sin el selector de historias de Estimación y Esfuerzo.
  - MODIFIED «Pie de teclas acotado y ayuda agrupada»: escenario de teclas de pestaña en la ayuda con 1–4.
  - Revisar (opcional) «Salir limpiamente desde el navegador y las pestañas» por referencias a las pestañas retiradas.
- `esfuerzo`, `defecto`, `reporte`: sin cambios de requisitos. El escenario de `esfuerzo` que menciona una «pantalla de resultado» ya está desactualizado y no bloquea este cambio.
- Documentación: `README.md` (pestañas, dígitos 1–6 y atajos `P`/`E`).

## Enfoque propuesto

Enfoque A' de la exploración, reutilizando el shell de secciones, los selectores y las guardas ya probados.

1. **Detalle**: pantalla propia `story-detail` que guarda solo el ID de la historia y la relee de `m.backlogStories` al renderizar, para no mostrar datos viejos tras estimar o modificar. Las acciones usan el selector existente y cada una tiene un atajo de una tecla.
2. **Destino de retorno**: un campo `returnTo` reemplaza a `preselected` y a los destinos fijos (`openSection(sectionEffort)`, `openBacklog()`, `"defects"`). Desde el detalle vale «detalle de la historia»; desde la pestaña Defectos conserva el comportamiento actual.
3. **Entrada a los flujos**: `openSectionForStory` evoluciona para abrir Planning Poker (paso de integrantes) y Esfuerzo (paso de tareas) desde el detalle; se eliminan los pasos `primary` de ambas secciones y `sectionDefects`.
4. **Alta de defecto precargada**: el alta recibe la historia elegida y salta `defect-story-select`; conserva el `requestID` contra respuestas tardías.
5. **Pestañas**: `workspaceTabs` queda en cuatro; `currentTab` mapea `story-detail`, `section`, `planning-poker`, `effort-form` y el alta de defecto iniciada desde la historia a Backlog (S1).
6. **Pruebas primero**: el helper `openSectionWithProject` pasa a recorrer Backlog → detalle → acción, lo que corrige la mayoría de sus ~25 sitios de llamada.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/tui/story_detail.go` (nombre a definir en diseño) | Nuevo | Pantalla de detalle, acciones, atajos, vista, pie y ayuda. |
| `internal/tui/workspace.go` | Modificado | Cuatro pestañas, `openTab`, `currentTab`, `tabRoot`. |
| `internal/tui/backlog_screens.go` | Modificado | `Enter` = detalle, `e` = modificar, retiro de `P`/`E`, `openSectionForStory`. |
| `internal/tui/sections.go` | Modificado | Retiro de pasos `primary` y `sectionDefects`; `returnTo` en el `Esc` del primer paso. |
| `internal/tui/planning_poker.go`, `internal/tui/effort_form.go` | Modificado | Destino del éxito vía `returnTo`. |
| `internal/tui/defect_creation.go`, `internal/tui/defect_submission.go` | Modificado | Historia precargada y destinos vía `returnTo`. |
| `internal/tui/model.go`, `internal/tui/keys.go` | Modificado | Switch paralelos, `quitsOnQ`, `entersText`, `filtering`, miga, pie y ayuda. |
| `tests/unit/tui/*`, `internal/tui/*_test.go` | Modificado / Nuevo | Pruebas reescritas y nuevas (ver abajo). |
| `openspec/changes/tui-refactor-acciones-desde-historia/specs/tui/spec.md` | Nuevo (delta) | Deltas descritos en Capacidades. |
| `README.md` | Modificado | Pestañas, dígitos y acciones desde la historia. |

### Pruebas que se espera reescribir
- `tests/unit/tui/sections_test.go`: constantes de pestaña, helper `openSectionWithProject`, `TestTabsRouteToEachSprint3Section`, `TestDefectsTabOpensProjectDefectList`.
- `tests/unit/tui/workspace_test.go`: Tab, `6`, Shift+Tab, miga de Esfuerzo y barra con seis nombres (~4 tests).
- `tests/unit/tui/contextual_shortcuts_test.go`: se reescribe completo para el detalle y sus acciones.
- `tests/unit/tui/backlog_test.go`: pruebas que usan `Enter` para modificar (~10–15, estimado) pasan a `e`.
- `tests/unit/tui/effort_result_test.go` e `internal/tui/effort_form_test.go`: destino del éxito.
- `internal/tui/defect_*_test.go`: destinos de retorno y pruebas nuevas de precarga.
- `internal/tui/sections_test.go`: se elimina la prueba de `sectionDefects`.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| `Enter` en Backlog cambia de significado y rompe la memoria muscular y parte de `backlog_test.go`. | Alta | `e` = modificar, coherente con Proyectos; pie y ayuda actualizados; pruebas migradas explícitamente. |
| Destinos fijos que quedan sin generalizar (`defects`, `openSection(sectionEffort)`, `backlog`). | Media | Un único `returnTo`; pruebas de éxito y cancelación por cada flujo y por cada origen. |
| Detalle con datos viejos tras estimar o modificar; con filtro, el cursor es un índice sobre la lista filtrada. | Media | Guardar el ID y releer de `backlogStories`; pruebas tras estimar y con filtro activo. |
| Pestaña o miga incorrecta: `defect-*` resalta Defectos por prefijo aunque se entre desde la historia. | Media | Mapeo por origen en `currentTab` (S1); pruebas de barra y miga en cada pantalla. |
| Olvidar una pantalla en alguno de los switch paralelos o en `quitsOnQ`/`entersText`/`filtering`. | Media | Lista de verificación en diseño; pruebas de `q`, `?` y pie por pantalla. |
| Pérdida de las guardas de respuestas tardías (seq de esfuerzo, Planning Poker, `requestID` de defecto). | Baja | Se conservan sin cambios; las pruebas existentes siguen en verde. |
| Detalle o `returnTo` residuales tras cambiar de proyecto. | Baja | Descarte explícito al cambiar de proyecto; prueba dedicada. |
| Registrar esfuerzo requiere un paso más que el atajo `E`. | Media | Aceptado por el usuario; el atajo de una tecla en el detalle reduce el costo. |
| Referencias sueltas a «pestaña Estimación/Esfuerzo» en ~15 requisitos de la spec y en el README. | Media | Búsqueda de referencias en la fase de spec y en la verificación. |
| El PR único supera el presupuesto de 400 líneas. | Alta | `size:exception` aceptado; commits por unidad de trabajo. |

## Plan de reversión

Revertir el PR único. El cambio es exclusivamente de cliente TUI, pruebas, spec y documentación: no agrega migraciones, rutas, endpoints ni datos. Al revertir vuelven las seis pestañas, `Enter` = modificar en Backlog, los atajos `P`/`E` y los destinos anteriores. Las estimaciones, registros de esfuerzo y defectos creados mientras el cambio estuvo activo usan las mismas requests de siempre y siguen siendo válidos; no requieren limpieza.

## Dependencias

- TUI-01 a TUI-13 integradas en `main` (pestañas, Backlog, secciones, Planning Poker, esfuerzo, Defectos, avisos efímeros, pie acotado).
- Ninguna dependencia de backend.

## Plan de entrega y carga de revisión

Un solo PR hacia `main` con la etiqueta `size:exception` (estrategia `single-pr`), excepción aceptada por el usuario. El PR no se abre en esta fase; su apertura es una decisión posterior del usuario.

Estimación: ~750–950 líneas autoras (adiciones más eliminaciones), frente al presupuesto de 400. La producción ronda ~320 líneas (más ~100 eliminadas); el resto son pruebas reescritas y nuevas, spec y README (~140). La fase de tareas recalibra el pronóstico, porque el desglose por partes de la exploración suma por encima del rango total. Justificación: el retiro de las pestañas sin el detalle deja las acciones inaccesibles, y el detalle sin el retiro mantiene la duplicación que motiva el cambio; un estado intermedio deja la navegación incoherente. Más de la mitad del volumen son pruebas y documentación. Dentro del PR se mantienen commits por unidad de trabajo con Conventional Commits en español (prefijo `type(scope)` en inglés): detalle y acciones, destino de retorno, alta de defecto precargada, retiro de pestañas y código muerto, spec y README.

## Decisiones abiertas para el orquestador

Ninguna bloqueante. El supuesto S1 queda para confirmación del usuario; la fase de spec lo toma como valor por defecto.

## Criterios de éxito

- [ ] La barra muestra cuatro pestañas (Dashboard, Backlog, Sprints, Defectos) y los dígitos 1–4 las abren; ya no existen Estimación ni Esfuerzo.
- [ ] `Enter` sobre una historia del Backlog abre su detalle con título, descripción, prioridad, estado, story points, criterios de aceptación y horas estimadas; `e` abre la modificación; `P` y `E` no hacen nada.
- [ ] Desde el detalle, Estimar, Registrar esfuerzo y Reportar defecto se abren tanto con `Enter` sobre la fila como con su atajo.
- [ ] Tras estimar, registrar esfuerzo o reportar un defecto, y al cancelar con `Esc`, el usuario vuelve al detalle de la misma historia con los datos releídos.
- [ ] El alta de defecto desde el detalle no muestra el selector de historias; el alta desde la pestaña Defectos conserva su flujo.
- [ ] La barra resalta Backlog en el detalle y en los flujos de acción iniciados desde él (S1).
- [ ] El cambio no emite requests HTTP nuevas ni usa endpoints nuevos.
- [ ] Cambiar de proyecto descarta el detalle y el destino de retorno.
- [ ] Los deltas de `tui` y el `README.md` no contienen referencias a las pestañas retiradas ni a `P`/`E`.
- [ ] `go test ./...` pasa.
