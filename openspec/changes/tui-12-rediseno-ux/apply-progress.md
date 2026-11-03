# Apply progress: rediseño de la experiencia de usuario de la TUI (TUI-12)

Este archivo registra la evidencia observada de cada Unidad de Trabajo (UT) y el commit que la contiene. Las UT siguen el orden de ejecución, que difiere del orden de `tasks.md`: el registro de atajos (tarea 1) se aplicó después de los cambios de navegación para no declarar atajos de pantallas que esos cambios eliminan.

## UT 0 — Artefactos SDD

- **Commit**: `3ed0769` `docs(tui): proponer el rediseño de la experiencia de usuario (TUI-12)`.
- Rama `feat/tui-ux-redesign` creada desde `origin/main`.
- `exploration.md` documenta el estado actual con evidencia (`model.go:150-160`, `updateMenu`, `pickerReturn`/`pickerPurpose`, pantallas de resultado, footers de 12 atajos) y tres opciones de arquitectura.
- Decisiones del usuario registradas en `proposal.md` («Decisiones resueltas»): pestañas superiores, Métricas dentro del Dashboard, avisos efímeros y refactorización interna incluida.

## UT 1 — Inicio en Proyectos y espacio de trabajo con pestañas (tarea 2, S2–S3)

- **Commit**: `8695026` `feat(tui): iniciar en Proyectos y abrir el espacio de trabajo con pestañas`.
- **RED**: `go test ./tests/unit/tui/ -run 'Startup|Leaving|EnterOnAProject|Tabs|ShiftTab|Digits|EscOnATab|EscInside|ChangingProject|BrowserEsc|OldProject'` → 12 `FAIL` de `tests/unit/tui/workspace_test.go` contra el código del menú.
- **GREEN**: `go test -count=1 ./internal/tui ./tests/unit/tui` → `ok` en ambos paquetes.
- Producción: `internal/tui/workspace.go` nuevo (`workspaceTab`, `workspaceTabs`, `openTab`, `currentTab`, `tabRoot`, `tabForKey`, `leaveWorkspace`, `workspaceHeader`); `NewModel` arranca en `project-picker` con `Init()` cargando `GET /projects`; `applyProjectsResult` abre el proyecto único en la primera carga (`autoOpen`); `selectProject` abre el Dashboard; se eliminaron el menú (`Choices`, `Cursor`, `updateMenu`, `viewMenu`), `pickerReturn`/`pickerPurpose` y los atajos `p` y `c`; la miga arranca en «Proyectos» y el `chrome` suma la línea del proyecto y la barra de pestañas; `tabActiveStyle`/`tabIdleStyle` en `styles.go`.
- Migración de pruebas delegada: helpers reescritos alrededor de `openTab(t, url, pid, digit)`; se eliminaron las pruebas del menú ya cubiertas por `workspace_test.go` y las de la sección Métricas (reintroducidas en la UT 2).
- Textos corregidos tras la migración: el pie del backlog ya no ofrece `c proyecto`, la pista vacía dice «Esc vuelve a Proyectos» y la ayuda del Dashboard «Volver a Proyectos».

## UT 2 — Métricas dentro del Dashboard (tarea 3, S4)

- **Commit**: `358299c` `feat(tui): integrar las métricas de proyecto y Sprint en el Dashboard`.
- **RED**: 9 de las 10 pruebas de `tests/unit/tui/dashboard_scope_test.go` fallan antes del cambio (`s` no abría selector y el pie no ofrecía el ámbito); `TestOldMetricsSectionIsUnreachable` es guardia de regresión y no fue roja.
- **GREEN**: `go test -count=1 ./internal/tui ./tests/unit/tui` → `ok`.
- Producción: `dashboardState` suma el ámbito (`sprintID`, `sprintLabel`, selector `scopes`, `scopesSeq`); pantalla `dashboard-sprint-select` con «Proyecto completo» más los Sprints abiertos; `loadDashboardCmd` consulta el endpoint de proyecto o de Sprint según el ámbito; `d` descarga el PDF del ámbito mostrado y `r` lo recarga; se eliminó la sección Métricas (`sectionMetrics`, `sectionStepScope`, `sectionStepMetrics`, `metricsScope`, `sectionMetricsScopeItems`, `viewMetrics` y sus ramas de pie, ayuda y aviso).
- Cobertura reintroducida: valores exactos de la API, `null` como «sin datos» y error de la API dentro de la pantalla, que antes vivían en las pruebas borradas de la sección Métricas.

## UT 3 — Avisos efímeros (tarea 4, S5)

- **Commit**: `c7dddb0` `feat(tui): reemplazar las pantallas de resultado por avisos efímeros`.
- **RED**: 16 de las 17 pruebas nuevas (`tests/unit/tui/toast_test.go`, `internal/tui/defect_toast_test.go`) fallan antes del cambio con `Screen = "project-result" … want the browser reloading` y sus equivalentes de `backlog-result`, `sprint-result`, `effort-result`, `defect-create-result` y `defect-status-result`.
- **GREEN**: `go test -count=1 ./internal/tui ./tests/unit/tui` → `ok`.
- Producción: `Model.toast` dibujado con `okBlock` y limpiado al inicio del manejo de la siguiente tecla, sin temporizadores; cada escritura exitosa vuelve a su pantalla de origen y recarga la colección (proyectos → navegador, historias → backlog, Sprint/asignación/cierre → lista de Sprints, tarea/completar → historias del Sprint, esfuerzo → selector de historias de Esfuerzo, defecto → listado, estado → detalle actualizado desde el cuerpo de la respuesta); el rechazo conserva el formulario, sus valores y el motivo de la API; indicadores en vuelo («Guardando…», «Registrando…», «Actualizando estado…») con `projectSaving`, `backlogForm.saving` y `sprintSaving`, que ignoran reenvíos; se eliminó la lectura de métricas posterior al registro de esfuerzo.
- Delta actualizado: requisito ADDED ampliado con destinos, indicador en vuelo y respuestas tardías; MODIFIED de «Crear historias mediante la TUI», «Presentar los errores de la API del backlog», «Operar los Sprints mediante la TUI», «Enviar el registro de esfuerzo…» y «Recorrer hacia atrás el flujo de esfuerzo…»; REMOVED de «Mostrar el esfuerzo registrado y las métricas del proyecto».

## UT 4 — Registro de atajos, pie acotado y ayuda agrupada (tarea 1, S1)

- **Commit**: `9fea460` `refactor(tui): declarar los atajos por pantalla y acotar el pie`.
- **RED**: `TestSprintListFooterShowsAtMostFiveShortcutsAndMore` (el pie tenía 11 pares y ningún `? más`), `TestSprintListHelpGroupsEveryShortcut` (la ayuda no tenía encabezados de grupo) y `TestTabRootHelpListsTabKeys` (faltaban las teclas de pestaña) fallan antes del cambio; `TestScreenWithFewShortcutsHasNoMore` y `TestFilterAndTextFieldFootersDoNotAdvertiseHelp` ya pasaban y protegen comportamiento que no debía cambiar.
- **GREEN**: `go test -count=1 ./internal/tui ./tests/unit/tui` → `ok`.
- Producción: `internal/tui/keys.go` nuevo con `keyGroup`, `binding` (con texto corto para el pie y largo para la ayuda), `bindings()`, `helpAvailable()`, el `footer()` de hasta cinco atajos primarios más `? más` y el `viewHelp()` agrupado en «Navegación», «Acciones» y «General»; se eliminaron `footer`, `defectSelectorFooter`, `viewHelp`, `helpRows`, `sprintsHelpRows`, `defectHelpRows`, `sectionFooter`, `sectionHelpRows`, los cuatro de Dashboard, `effortFormFooter` y `planningPokerFooter`.
- Conjunto primario por pantalla registrado en el reporte de la UT; el resto de los atajos sigue disponible en la ayuda.

## UT 5 — Atajos contextuales de estimación y esfuerzo (tarea 5, S6)

- **Commit**: `1929638` `feat(tui): estimar y registrar esfuerzo desde la historia resaltada`.
- **RED**: `TestBacklogPOpensEstimationParticipantsForHighlightedStory`, `TestBacklogEOpensEffortTasksForHighlightedStory`, `TestEscFromPreselectedSelectorReturnsToBacklog/{P,E}`, `TestDeeperEscStillGoesBackOneStep` y `TestContextualShortcutFlowOnlyReads` fallan antes del cambio con «expected a command to execute, got nil» (los atajos no existían); `TestBacklogHelpListsPAndE` falla porque el grupo «Acciones» solo listaba `Enter`, `n` y `r`.
- **GREEN**: `go test -count=1 ./internal/tui ./tests/unit/tui` → `ok`.
- Producción: `openSectionForStory` en `backlog_screens.go` abre la pestaña con la historia ya elegida y avanza al paso siguiente (`storyNextStep`); `sectionState.preselected` hace que `Esc` en ese primer selector vuelva al backlog recargándolo, y los pasos más profundos siguen retrocediendo de a uno; la vista de participantes de Planning Poker muestra la línea «Historia: …»; `keys.go` declara `P` y `E` en el grupo «Acciones», solo cuando la lista tiene historias.
- Decisión: `P` y `E` quedan en la ayuda, no en el pie; los cinco primarios del backlog (`↑/↓`, `Enter`, `n`, `/`, `Esc`) se usan más seguido y `? más` ya aparecía.
- **PRESERVE**: `TestShortcutsDoNothingOnEmptyBacklog` y `TestLowercasePAndEStillStartASearch` pasan antes y después; fijan que el backlog vacío no reacciona y que `p`/`e` minúsculas siguen iniciando la búsqueda.

## Pendiente

- Tarea 6 (S7): README, consolidación de `openspec/specs/tui/spec.md` y archivado.

## Estrategia de entrega

El cambio supera el presupuesto de 400 líneas de `openspec/config.yaml`. Se entrega como PRs encadenados, uno por UT, en este orden: UT 0+1, UT 2, UT 3, UT 4, tarea 5 y tarea 6. Cada commit de la rama es autocontenido y deja la suite en verde.
