# Progreso de apply: tui-refactor-acciones-desde-historia

**Modo**: Strict TDD (`go test ./...` como referencia; iteración con `go test ./internal/tui/... ./tests/unit/tui/...`)
**Rama**: `feat/tui-refactor-acciones-desde-historia`
**Entrega**: PR único con `size:exception` (aceptado por el usuario). No se hizo push ni se abrió PR.
**Skills aplicadas**: `go-testing` (pruebas por tabla, `Model.Update` directo, `httptest`, sin `teatest`) y `work-unit-commits` (un commit por UT con sus pruebas).

## Resumen de lotes

| UT | Commit | Estado |
| -- | ------ | ------ |
| 1 | `b3c10ca` `feat(tui): abrir el detalle de la historia desde el Backlog` | Completa (tareas 1.1 a 1.11) |
| 2 | `84850d9` `feat(tui): estimar y registrar esfuerzo desde el detalle` | Completa (tareas 2.1 a 2.14) |
| 3 | `a3d9f52` `feat(tui): reportar un defecto desde el detalle con la historia precargada` | Completa (tareas 3.1 a 3.8) |
| 4 | `7e5b074` `test(tui): recorrer los flujos de estimación y esfuerzo desde el detalle` | Completa (tareas 4.1 a 4.8) `[migración]` |
| 5 | `cd7979c1ee0b9ad9ce2911040024eac07e78854f` `refactor(tui): retirar las pestañas Estimación y Esfuerzo y el código muerto de secciones` | Completa (tareas 5.1 a 5.10) |
| 6 | `docs(tui): actualizar el README para las acciones desde la historia` (el hash queda en el historial de git) | Completa (tareas 6.1 a 6.3) `[solo documentación]` |

## UT 1 — Detalle de la historia desde el Backlog (commit 1) — COMPLETA

**Commit**: `b3c10ca2d8fe247e5e55cb19ce0b0a6ad799de46` — `feat(tui): abrir el detalle de la historia desde el Backlog`

### Estado por tarea

| Tarea | Estado | Evidencia resumida |
| ----- | ------ | ------------------ |
| 1.1 Línea base | [x] | Rama correcta; `go test ./internal/tui/... ./tests/unit/tui/...` en verde antes de tocar código (297 `--- PASS`, 0 `--- FAIL`) |
| 1.2 RED apertura del detalle | [x] | 5 de 6 pruebas fallan por aserción (`Screen = "backlog-form", want story-detail`); `TestBacklogEnterOnEmptyListDoesNotOpenDetail` nace en verde `[caracterización]` |
| 1.3 GREEN pantalla y apertura | [x] | `go test ./tests/unit/tui/... -run 'StoryDetail\|BacklogEnter'` → `ok` |
| 1.4 RED `e` modifica | [x] | 4 pruebas nuevas fallan por aserción; `TestBacklogEKeyInsideFilterIsText` y `TestStoryDetailEKeyDoesNothing` nacen en verde `[caracterización]`; las ~10 pruebas de modificación existentes (vía `selectStory` con `e`) fallan (incluido un `nil pointer` de `cmd()` en una de ellas) |
| 1.5 GREEN `e` y bindings | [x] | Ver detalle abajo |
| 1.6 RED estados y guarda | [x] | Fallan por aserción `TestStoryDetailStatesGateAvailability/listo`, 4 filas de `TestStoryDetailViewIndicatesLoadingErrorAndMissing` y `TestApplyBacklogResultAcceptsStoryDetailScreen/detalle_de_la_historia` (esqueleto `storyActionsAvailable()` con valor cero en el mismo paso) |
| 1.7 GREEN estados y guarda | [x] | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` |
| 1.8 RED registro en los switch | [x] | Fallan `TestStoryDetailQuitsOnQ`, `TestStoryDetailHighlightsBacklogTabAndCrumb`, `TestStoryDetailFooterOffersEscAndHelp` y `TestCurrentTabOfStoryDetailIsBacklog`; `Tab`/dígitos y `?` nacen en verde `[caracterización]` |
| 1.9 GREEN registro | [x] | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` (incluye `Footer\|Help\|UX\|Workspace`) |
| 1.10 REFACTOR | [x] | `gofmt` (normalizando CRLF), `go vet ./internal/tui/...` y `go test` en verde; ver notas |
| 1.11 Evidencia y commit | [x] | Este archivo, Engram y el commit `b3c10ca` |

### Ciclo TDD observado

| Paso | RED observado (comando y resultado) | GREEN observado |
| ---- | ----------------------------------- | --------------- |
| 1.2/1.3 | `go test ./tests/unit/tui/... -run 'StoryDetail\|BacklogEnter'` → 5 `--- FAIL` (`Screen = "backlog-form", want story-detail`) | Mismo comando → `ok` tras `story_detail.go` + `case "story-detail"` en `Update`/`viewBody` + `Enter` → `openStoryDetail` |
| 1.4/1.5 | `go test ./tests/unit/tui/... -run 'TestBacklogEKey\|TestStoryDetailEKey' -v` → 4 `--- FAIL` (`Screen = "backlog", want backlog-form`; `e` abría el filtro `Buscar: e_`), 2 `--- PASS` `[caracterización]` | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` tras `case "e"` y bindings |
| 1.5 (bindings) | RED retroactivo honesto: las pruebas `TestBacklogFooterOffersDetailAndEditWithoutExceedingTheLimit` y `TestBacklogHintAnnouncesDetailAndEdit` se escribieron **después** del cambio de `keys.go`/`backlogHint` (el cambio no tenía prueba que lo fallara); se verificó su RED revirtiendo temporalmente `keys.go` (`git stash`) y la cadena de `backlogHint`: ambas fallaron por aserción (`footer "… Enter editar … / buscar …" lacks "Enter detalle"`) y luego se restauró el código | Con el código restaurado → `ok` |
| 1.6/1.7 | `go test ./internal/tui -run 'StoryDetailStates\|StoryDetailView\|ApplyBacklogResult' -v` → fallos de aserción listados arriba | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` |
| 1.8/1.9 | `go test ./tests/unit/tui/... ./internal/tui -run 'StoryDetail\|CurrentTabOfStoryDetail' -v` → 4 `--- FAIL` (`q` no salía; sin cabecera de pestañas/miga; pie `Esc volver`; `currentTab() = tabNone`) | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` |

**Triangulación**: apertura con filtro (tabla de 2 filas), valores presentes vs. faltantes, cuatro estados + precedencia «carga sobre error», guarda de `applyBacklogResult` con tres pantallas (`story-detail`, `backlog`, otra descartada).

**Resultado final de la UT**: `go test -count=1 ./internal/tui/... ./tests/unit/tui/...` → `ok` en ambos paquetes; `go test -v` → **320 `--- PASS`, 0 `--- FAIL`** (297 de la línea base + 23 pruebas nuevas de primer nivel). `go build ./...` y `go vet ./internal/tui/... ./tests/unit/tui/...` sin salida.

### Work Unit Evidence

| Evidencia | Valor |
| --------- | ----- |
| Prueba focal y resultado | `go test ./internal/tui/... ./tests/unit/tui/... -run 'StoryDetail\|StoryFlow\|Backlog'` y la suite completa de ambos paquetes: `ok` / `ok`, 320 PASS, 0 FAIL |
| Harness de ejecución | N/A: `Model.Update` con `httptest`, sin frontera de proceso; recorrido manual pendiente en la fase 7.4 |
| Límite de rollback | Revertir `internal/tui/story_detail.go`, los casos de `model.go`/`keys.go`/`workspace.go`/`backlog_screens.go` de esta UT y las pruebas nuevas; `Enter` vuelve a modificar |

### Archivos tocados

| Archivo | Acción | Qué se hizo |
| ------- | ------ | ----------- |
| `internal/tui/story_detail.go` | Creado | `storyDetailState`, `openStoryDetail`, `detailStory`, `storyActionsAvailable`, `updateStoryDetail` (solo `Esc`), `viewStoryDetail` (cuatro estados y siete campos), `storyDetailBindings`, `detailField` |
| `internal/tui/model.go` | Modificado | Campo `storyDetail`; `story-detail` en `Update`, `viewBody`, `quitsOnQ` y `crumbParts` (`Backlog › Detalle`) |
| `internal/tui/backlog_screens.go` | Modificado | `Enter` → `openStoryDetail`; `e` → `editSelectedStory` (reservada); guarda de `applyBacklogResult` acepta `story-detail`; `backlogHint` |
| `internal/tui/keys.go` | Modificado | Bindings del Backlog (`Enter detalle`, `e editar`, `/` pasa a la ayuda para no superar `footerLimit`) y `case "story-detail"` |
| `internal/tui/workspace.go` | Modificado | `currentTab` devuelve `tabBacklog` para `story-detail` (S1); `tabRoot` sin cambios |
| `tests/unit/tui/story_detail_test.go` | Creado | 14 pruebas de primer nivel: apertura, valores faltantes, sin requests, lista vacía, filtro, `e`, `Tab`/dígitos, `q`, `?`, pestaña/miga, pie, pie y hint del Backlog |
| `tests/unit/tui/backlog_test.go` | Modificado | `selectStory` usa `e`; prueba de filtro y de lista vacía migradas a `e`; 5 pruebas `BacklogEKey*` |
| `tests/unit/tui/contextual_shortcuts_test.go` | Modificado | `TestLowercasePAndEStillStartASearch` pasa a `TestLowercasePStillStartsASearch` (`e` ya no inicia la búsqueda); el archivo se elimina en la UT 5 |
| `internal/tui/story_flow_test.go` | Creado | Estados del detalle, vista de los cuatro estados, guarda de la lista, `currentTab`/`tabRoot` del detalle |

### Desviaciones respecto de las tareas y el diseño

1. **`returnTo`/`returnTarget` y `storyDetailState.cursor` no se agregaron en la UT 1** (la tarea 1.3 los menciona). Ninguna prueba de la UT 1 los usa y el código sin prueba que lo falle contradice TDD estricto; la tarea 2.2 ya define `returnTarget` y el cursor de filas, y ahí tienen su RED. **La UT 2 debe agregarlos** (`Model.returnTo`, `returnDefault`, `returnStoryDetail`, `storyDetailState.cursor`).
2. **`backlogHint` sin «r refresca»**: la cadena exacta del diseño («Enter abre el detalle · e edita · n crea · r refresca · / busca · Esc vuelve a Proyectos.», 89 caracteres) supera el ancho interno de 76 columnas del panel a 80 columnas, se parte en dos líneas y rompe la aserción existente `Esc vuelve a Proyectos` (`TestBacklogEmptyListShowsMessageAndKeepsNavigatorKeys`). Se acortó a «Enter abre el detalle · e edita · n crea · / busca · Esc vuelve a Proyectos.» (76 caracteres); `r` sigue en la ayuda.
3. **Binding `/ buscar` pasa a no primario** en el Backlog: con `Enter detalle` y `e editar` habría seis atajos primarios y el pie recortaría `Esc proyectos`. Coincide con la tabla de bindings del diseño (`/ buscar` solo en la ayuda). `P`/`E` se mantienen en la ayuda hasta la UT 5.
4. **Dos pruebas existentes se modificaron** por el cambio de comportamiento especificado: `TestBacklogSearchFiltersStories` y `TestBacklogSelectingStoryWithEmptyListOpensNoForm` (de `Enter` a `e`) y `TestLowercasePAndEStillStartASearch` (ver archivos).
5. **El RED del cambio de bindings del Backlog** (`keys.go`, `backlogHint`) no se observó antes de escribirlo; se observó retroactivamente revirtiendo el cambio (ver tabla). Se registra para no inventar evidencia.
6. **Normalización de fin de línea**: el árbol de trabajo usa CRLF (`core.autocrlf=true`) y `gofmt -l` marca como «sin formato» todo archivo CRLF preexistente. La verificación de formato se hizo con `tr -d '\r' < archivo | gofmt -l` sobre los archivos tocados: sin salida.
7. **`TestStoryDetailHighlightsBacklogTabAndCrumb`** verifica la cabecera de pestañas y la miga; el resaltado exacto de Backlog se prueba con `currentTab()` en `internal/tui/story_flow_test.go` (los estilos ANSI no se pueden distinguir en la salida de prueba).

### Notas para la UT 2

- `openStoryFlow` debe usar `storyActionsAvailable()` (ya existe, hoy solo usada por pruebas) y `m.row(selected, text)` para las filas.
- El detalle hoy solo maneja `Esc`; `storyDetailBindings` tiene `Esc backlog` como única primaria y `?`/`q`/`Ctrl+C` salen de `bindings()`. La UT 2 debe agregar `↑/↓`, `Enter`, `p`, `t`.
- `currentTab` ya mapea `story-detail` → Backlog con una rama incondicional; las ramas por origen (`returnTo`) para `section`/`planning-poker`/`effort-form` van en la tarea 2.12.
- Helper de pruebas disponibles: `openStoryDetailWithProject`, `detailServer`, `detailStories`, `footerOf` (en `story_detail_test.go`) y `detailModel` (en `internal/tui/story_flow_test.go`).
- El archivo `.go` nuevo y los de prueba se crearon con LF; el árbol de trabajo del resto usa CRLF. Git normaliza al commitear (aviso `LF will be replaced by CRLF`), sin efecto en el historial.

## UT 2 — Estimar y registrar esfuerzo desde el detalle (commit 2) — COMPLETA

**Commit**: `84850d9fb5b1ae44e2fe7ba533f2babc78223040` — `feat(tui): estimar y registrar esfuerzo desde el detalle`

### Estado por tarea

| Tarea | Estado | Evidencia resumida |
| ----- | ------ | ------------------ |
| 2.1 RED acciones del detalle | [x] | 5 pruebas nuevas fallan por aserción (filas ausentes, `Enter`/`p`/`t` sin comando); en `internal/tui` los casos «listo» de `TestStoryDetailActionsDoNothingWhileLoading` fallan y los de cargando/error/ausente nacen en verde `[caracterización]` |
| 2.2 GREEN tipos y apertura | [x] | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` / `ok` |
| 2.3 RED `Esc` del primer paso | [x] | 5 pruebas fallan por aserción (el `Esc` caía en el selector primario vacío en vez del detalle) |
| 2.4 GREEN `Esc` del primer paso | [x] | Rama `selectorBack` en `updateSection`; suite en verde. Los bindings se movieron a 2.6 (ver desviación 1) |
| 2.5 RED pie y ayuda | [x] | `TestStoryDetailFooterShowsAtMostFiveKeysIncludingEnterAndEsc`, `TestStoryDetailHelpListsActionShortcutsAndNotTabs` y el caso «listo» de `TestStoryDetailStatesShowOnlyEscInFooter` fallan; los tres estados sin acciones nacen en verde `[caracterización]` (el pie de la UT 1 ya era solo `Esc backlog`) |
| 2.6 GREEN/TRIANGULATE pie y ayuda | [x] | `storyDetailBindings` con `↑/↓`, `Enter elegir acción`, `Esc backlog` y `p`/`t` solo en la ayuda; suite en verde |
| 2.7 RED guarda `StoryID` | [x] | `TestSectionResultFromAnotherStoryIsDropped` y `TestLateTasksResponseOfAnotherStoryDoesNotPopulateTheSelector` fallan por aserción (esqueleto `StoryID string` en `SectionResultMsg`); `TestSectionResultWithMatchingStoryIDStillApplies` nace en verde `[caracterización]` |
| 2.8 GREEN `StoryID` | [x] | `loadSectionStepCmd` puebla `StoryID`; `applySectionResult` lo compara; suite en verde |
| 2.9 RED éxito de Poker y estados del detalle | [x] | 5 pruebas fallan por aserción (`Screen = "backlog"`): `PokerSaveFromDetail…`, `ShowsStoryPointsReleidosAfterReload`, `ShowsLoadingWhileBacklogReloads`, `ShowsAPIErrorWhenReloadFails`, `ReportsMissingStoryAfterReload`. `TestPlanningPokerExplicitConfirmationSavesThenRefreshesBacklog` (`returnDefault`) se mantiene sin cambios `[caracterización]` |
| 2.10 GREEN recarga in situ | [x] | `reloadBacklogInPlace` y rama `returnStoryDetail` de `applyPokerSave`; sin guarda nueva (el doble envío queda fuera de alcance) |
| 2.11 RED/GREEN éxito de Esfuerzo | [x] | 3 pruebas fallan por aserción (`Screen = "section"`, recarga emitida, sin detalle); GREEN en `applyEffortSave` |
| 2.12 RED/GREEN `selectProject`, `openTab`, pestaña y miga | [x] | Fallan `TestSelectProjectResetsStoryDetailAndReturnTo`, `TestOpenTabResetsReturnTo` (6 pestañas), `TestBacklogShortcutsResetAResidualReturnTo` (P/E) y 5 filas de `TestCurrentTabAndCrumbPerScreenAndOrigin`; las 5 filas `returnDefault` y `TestChangingProjectDiscardsDetailAndReturnTarget` nacen en verde `[caracterización]`; el RED de `TestFlowScreensHighlightBacklogTab` se verificó mutando `storyFlowCrumb` |
| 2.13 REFACTOR | [x] | `reloadBacklogInPlace` delega en `startBacklogLoad` (cuerpo idéntico); `openStoryFlow` y `openSection` no comparten código útil (`openSection` se elimina en la UT 5); `gofmt`, `go vet` y `go test` en verde |
| 2.14 Evidencia y commit | [x] | Este archivo, Engram y el commit `84850d9` |

### Ciclo TDD observado

| Paso | RED observado (comando y resultado) | GREEN observado |
| ---- | ----------------------------------- | --------------- |
| 2.1/2.2 | `go test ./tests/unit/tui/... -run 'StoryDetailAction\|StoryDetailArrows\|StoryDetailEnter\|StoryDetailShortcuts'` → 5 `--- FAIL` (filas ausentes; `p`/`t`/`Enter` sin comando); `go test ./internal/tui -run StoryDetailActionsDoNothing -v` → 3 `FAIL` («listo») y 9 `PASS` `[caracterización]` | `go test -count=1 ./internal/tui/... ./tests/unit/tui/...` → `ok` / `ok` |
| 2.3/2.4 | `go test ./tests/unit/tui/... -run 'EscFrom\|EffortEscChain'` → 5 `--- FAIL` por aserción (pantalla `section` con el selector primario vacío «No hay historias en el backlog») | Suite → `ok` / `ok` |
| 2.5/2.6 | `go test ./tests/unit/tui/... -run 'StoryDetailFooter\|StoryDetailHelp'` → 2 `FAIL` (pie `Esc backlog · ? más`; ayuda sin grupo «Acciones»); `go test ./internal/tui -run StoryDetailStatesShowOnlyEsc -v` → `listo` falla, 3 estados pasan | Suite → `ok` / `ok` (tras corregir un defecto de la propia prueba: el borde `│` era el primer campo de la línea de ayuda) |
| 2.7/2.8 | `go test ./internal/tui ./tests/unit/tui/... -run 'SectionResultFromAnotherStory\|SectionResultWithMatching\|LateTasksResponseOfAnotherStory' -v` → 2 `FAIL` (el selector de B quedó poblado con las tareas de A), 1 `PASS` `[caracterización]` | Suite → `ok` / `ok` |
| 2.9/2.10 | `go test ./tests/unit/tui/... -run 'PokerSaveFromDetail\|StoryDetailShows\|StoryDetailReports'` → 5 `FAIL` (`Screen = "backlog"`; el detalle no se muestra) | Suite → `ok` / `ok` |
| 2.11 | `go test ./tests/unit/tui/... -run 'EffortSuccessFromDetail\|EffortSecondRegistrationFromDetail\|EffortToast'` → 3 `FAIL` (`Screen = "section"`; el éxito recargaba las historias) | Suite → `ok` / `ok` |
| 2.12 | `go test ./internal/tui -run 'SelectProject\|OpenTab\|CurrentTabAndCrumb\|BacklogShortcutsReset' -v` → 4 pruebas `FAIL`: `returnTo`/`storyDetail` sin reinicio y pestaña/miga de `section`/`planning-poker`/`effort-form` sin ramas por origen. RED de `TestFlowScreensHighlightBacklogTab` observado mutando temporalmente `storyFlowCrumb` (5 subpruebas `FAIL`) y restaurando después | Suite → `ok` / `ok` |

**Triangulación**: filas y atajos con dos acciones (`Enter` en cada fila, `p`/`t`); `Esc` del primer paso con éxito, con error 500 y con filtro, para Poker y Esfuerzo; guarda `StoryID` con ID distinto y con ID coincidente; recarga de Poker con puntos, con error 500 y con historia ausente; origen `returnStoryDetail` frente a `returnDefault` en cinco pantallas.

**Resultado final de la UT**: `go test -count=1 ./internal/tui/... ./tests/unit/tui/...` → `ok` en ambos paquetes; `go test -v` → **350 `--- PASS` de primer nivel (más 312 subpruebas), 0 `--- FAIL`** (320 de la UT 1 + 30 pruebas nuevas de primer nivel). `go build ./...`, `go vet ./internal/tui/... ./tests/unit/tui/...` y `gofmt` (normalizando CRLF por archivo) sin salida. `go test ./...` completo: los paquetes de integración fallan solo por falta de Docker (`rootless Docker is not supported on Windows`), ajeno a este cambio; el resto de los paquetes está en `ok`.

### Work Unit Evidence

| Evidencia | Valor |
| --------- | ----- |
| Prueba focal y resultado | `go test -count=1 ./internal/tui/... ./tests/unit/tui/...` (suite completa de ambos paquetes): `ok` / `ok`, 350 PASS de primer nivel, 0 FAIL |
| Harness de ejecución | N/A: `Model.Update` con `httptest`, sin frontera de proceso; recorrido manual pendiente en la fase 7.4 |
| Límite de rollback | Revertir `returnTarget`, `storyActions`, `openStoryFlow`, `backToStoryDetail` y `reloadBacklogInPlace` (`story_detail.go`), `StoryID` y `firstStep` (`sections.go`), las ramas `returnStoryDetail` de `applyPokerSave`/`applyEffortSave`, `inStoryFlow`/`storyFlowCrumb` y el reinicio en `selectProject`/`openTab`/`openSection`, más las pruebas nuevas; las pestañas viejas siguen operando con `returnDefault` |

### Archivos tocados

| Archivo | Acción | Qué se hizo |
| ------- | ------ | ----------- |
| `internal/tui/story_detail.go` | Modificado | `returnTarget`, `storyDetailState.cursor`, `storyActions` (`p`, `t`), `openPokerForStory`/`openEffortForStory`, `updateStoryDetail` con `↑`/`↓`/`Enter`/atajos (solo con el detalle listo), `openStoryFlow`, `backToStoryDetail`, `reloadBacklogInPlace`, filas en `viewStoryDetail`, `storyDetailBindings` |
| `internal/tui/sections.go` | Modificado | `firstStep()`, `StoryID` en `SectionResultMsg`, `loadSectionStepCmd` y `applySectionResult`; `Esc` del primer paso → `backToStoryDetail`; `openSection` reinicia `returnTo` |
| `internal/tui/model.go` | Modificado | Campo `returnTo`; `storyFlowCrumb()` y prefijo `Backlog › Detalle` en `crumbParts` de `section`/`planning-poker`/`effort-form` |
| `internal/tui/workspace.go` | Modificado | `inStoryFlow()`, rama de `currentTab` hacia Backlog (S1), `openTab` reinicia `returnTo` |
| `internal/tui/planning_poker.go` | Modificado | `applyPokerSave`: con `returnStoryDetail` vuelve al detalle y recarga in situ |
| `internal/tui/effort_form.go` | Modificado | `applyEffortSave`: con `returnStoryDetail` vuelve al detalle con el aviso, sin recargar |
| `internal/tui/project_picker.go` | Modificado | `selectProject` reinicia `storyDetail` y `returnTo` |
| `internal/tui/story_flow_test.go` | Modificado | Gating de acciones por estado, pie por estado, guarda `StoryID`, reinicios, tabla de pestaña y miga por origen |
| `tests/unit/tui/story_detail_test.go` | Modificado | Filas, flechas, `Enter`, atajos, `Esc` del primer paso, pie, ayuda, respuesta tardía de otra historia, cambio de proyecto, pestaña de los flujos |
| `tests/unit/tui/sections_test.go` | Modificado | Helper aditivo `openStoryFlowWithProject` (N1) |
| `tests/unit/tui/effort_test.go` | Modificado | Helper `openEffortFormFromDetail`; `Esc` de tareas y cadena de `Esc` hasta el detalle |
| `tests/unit/tui/effort_result_test.go` | Modificado | Éxito de Esfuerzo desde el detalle y segundo registro |
| `tests/unit/tui/toast_test.go` | Modificado | Aviso que desaparece con la siguiente tecla en el detalle |
| `tests/unit/tui/planning_poker_test.go` | Modificado | Servidor, helpers y cinco pruebas del éxito de Poker y de los estados del detalle tras recargar |

### Desviaciones respecto de las tareas y el diseño

1. **Orden de 2.4 y 2.6**: la tarea 2.4 incluye los bindings del detalle, pero la 2.5 (RED del pie y la ayuda) exige que las pruebas fallen antes. Para conservar un RED real, 2.4 implementó solo el `Esc` del primer paso y `storyDetailBindings` se escribió en 2.6 tras observar el RED de 2.5.
2. **`TestStoryDetailActionsDoNothingWhileLoading` y `TestStoryDetailStatesShowOnlyEscInFooter` viven en `internal/tui/story_flow_test.go`** (no en `tests/unit/tui/story_detail_test.go`): necesitan fabricar `backlogLoading`, `backlogMessage` y una historia ausente, cosa que el paquete externo no puede hacer. El caso de carga de punta a punta sí está en `planning_poker_test.go` (`TestStoryDetailShowsLoadingWhileBacklogReloads`: Poker → guardado → recarga en curso).
3. **`openSection` reinicia `returnTo`** (no figura en las tareas): `P`/`E` en el Backlog siguen entrando por `openSection`; sin el reinicio, un `returnTo = returnStoryDetail` residual (tras volver del detalle al Backlog) haría que esos flujos volvieran al detalle. Lo cubre `TestBacklogShortcutsResetAResidualReturnTo`; el reinicio desaparece con `openSection` en la UT 5.
4. **`reloadBacklogInPlace` delega en `startBacklogLoad`**: ambas asignan los mismos cuatro campos y emiten el mismo comando; se evita duplicar.
5. **Datos de prueba**: las pruebas usan `effortFlowServer` (historias «Primera historia» y «Segunda historia»), no «Login»; los escenarios de la spec se verifican con esos datos. La prueba de respuesta tardía de otra historia navega `Esc`, `Esc`, `↓`, `Enter`, `t` (la secuencia de la tarea omite el `Esc` al Backlog, necesario para llegar al detalle de la otra historia).
6. **Normalización de fin de línea**: igual que en la UT 1 (`tr -d '\r' < archivo | gofmt -l` por archivo, sin salida). Algunos archivos editados con la herramienta quedaron con LF en el árbol de trabajo; Git los normaliza al commitear.
7. **Fuera de alcance, respetado**: no se agregó guarda al doble envío de Planning Poker.

### Notas de caracterización (RED no alcanzable)

- `TestStoryDetailActionsDoNothingWhileLoading` (casos cargando/error/ausente), `TestStoryDetailStatesShowOnlyEscInFooter` (tres estados sin acciones), `TestSectionResultWithMatchingStoryIDStillApplies`, las cinco filas `returnDefault` de `TestCurrentTabAndCrumbPerScreenAndOrigin` y `TestChangingProjectDiscardsDetailAndReturnTarget` nacen en verde: el comportamiento ya existía o no depende del código nuevo (abrir un detalle o un flujo sobrescribe el estado anterior). Su valor es fijar la regla; el RED del reinicio lo aporta `TestSelectProjectResetsStoryDetailAndReturnTo`.
- Supuesto S1 (sin confirmar por el usuario): vive solo en `inStoryFlow()` y en la rama correspondiente de `currentTab` (`workspace.go`), marcado con un comentario en el código; mientras existan las pestañas viejas es condicional a `returnTo == returnStoryDetail` (N3) y se simplifica en la tarea 5.8.

### Notas para la UT 3

- `storyActions` ya es la tabla única: agregar `{key: "d", label: "Reportar defecto", help: "Reportar un defecto de la historia", open: openDefectForStory}` hace que la fila, el atajo y la ayuda aparezcan solos (`storyDetailBindings` recorre la tabla). Hay que **extender** `TestStoryDetailActionRowsInOrder` (tercera fila), `TestStoryDetailArrowsMoveHighlightWithinBounds` (el tope pasa de «Registrar esfuerzo» a «Reportar defecto») y `TestStoryDetailHelpListsActionShortcutsAndNotTabs` (agregar `d`).
- `returnTo`, `returnStoryDetail`, `backToStoryDetail()` y el cursor ya existen. `beginDefectCreation()` debe asignar `returnTo = returnDefault` de forma explícita (hoy no lo toca; `openTab` y `selectProject` lo reinician).
- `inStoryFlow()` cubre `section`, `planning-poker` y `effort-form`; las pantallas `defect-*` necesitan su propia rama (`strings.HasPrefix(m.Screen, "defect-") && m.returnTo == returnStoryDetail` → Backlog, evaluada **antes** de la de Defectos) y el prefijo `Backlog › Detalle` en `crumbParts`.
- Helpers de prueba nuevos: `requestLines`, `openStoryFlowWithProject` (solo `p`/`t`), `openEffortFormFromDetail`, `openBacklogOfProject`, `pokerDetailServer`, `pokerConfirmFromDetail`, `assertDetailIgnoresActions`; en `internal/tui`: `flowScreenModel` y `effortTasksStep`.
- `applyPokerSave` y `applyEffortSave` son los únicos puntos con rama `returnStoryDetail`; sus ramas `returnDefault` se eliminan en la UT 5.

### Estado del entregable

- `apply-progress.md` y `tasks.md` quedan **sin commitear** (van en el commit 6 con el resto del directorio del cambio). `openspec/config.yaml` queda fuera de todo commit.

## UT 3 — Reportar un defecto desde el detalle (commit 3) — COMPLETA

**Commit**: `a3d9f52a716cb3abbbdc284d925a76e50ceb5c15` — `feat(tui): reportar un defecto desde el detalle con la historia precargada`

### Estado por tarea

| Tarea | Estado | Evidencia resumida |
| ----- | ------ | ------------------ |
| 3.1 RED precarga y salto del selector | [x] | `TestDefectFromStoryOpensFormWithPreloadedStory`, `…EnterOnLastFieldOpensSprintSelector` y `…RowEnterAndShortcutAreEquivalent` fallan por aserción (`Screen = "story-detail"`, `d` y la tercera fila no abren el alta); en `tests/unit/tui` fallan `TestStoryDetailActionRowsInOrder`, `…ArrowsMoveHighlightWithinBounds` y `…HelpListsActionShortcutsAndNotTabs` (extendidas con la tercera fila y `d`) |
| 3.2 GREEN precarga | [x] | Tercera entrada de `storyActions`, `openDefectForStory`, `beginDefectCreationForStory`, `returnTo = returnDefault` explícito en `beginDefectCreation`; las seis pruebas anteriores en verde |
| 3.3 RED destinos por `returnTo` | [x] | Fallan por aserción 5 de las 13 `DefectFromStory*`: `EscOnFormReturnsToDetailAndDiscardsDraft` (`Screen = "defects"`), `EscOnSprintSelectorReturnsToFormKeepingDraft` (`defect-story-select`), `CancelWithNReturnsToDetailWithoutPost` (`defects`), `SuccessReturnsToDetailWithToast` (`defects` con recarga) y `LateResponsesAreDiscarded/Sprints_tras_Esc` (la precondición `Esc` caía en `defect-story-select`). Nacen en verde `[caracterización]`: `EscOnConfirmReturnsToFormKeepingDraft`, `RejectionKeepsConfirmationWithValues` (422 y conexión), `DoubleSubmitEmitsSingleRequest`, `LateResponsesAreDiscarded/respuesta_de_un_proyecto_anterior`, `SprintQueryFailureShowsErrorAndAllowsRetry` (escenario «Falla la consulta de Sprints», no listado en la tarea) y `SendsPreloadedStoryIDInBody` (la precarga ya la daba 3.2) |
| 3.4 GREEN destinos y ayuda | [x] | `Esc` del formulario y `n` de la confirmación → `cancelDefectCreation`; `Esc` del selector de Sprints → formulario; éxito → aviso + `backToStoryDetail()` sin recargar; ayuda del formulario por historia precargada. 16/16 pruebas nuevas en verde |
| 3.5 RED/GREEN alta desde Defectos y `returnTo` residual | [x] | `TestNInDefectsResetsResidualReturnTo` y `[caracterización]` `TestDefectCreationFromDefectsTabKeepsStoryPickerAndReturnsToList` (5 subpruebas); su RED por mutación (ver abajo). Las pruebas existentes `defect_create_test.go`, `defect_submission_test.go` y `defect_toast_test.go` pasan sin modificarse |
| 3.6 RED/GREEN pestaña y miga `defect-*` por origen (S1) | [x] | Tabla de `TestCurrentTabAndCrumbPerScreenAndOrigin` con 3 filas `returnStoryDetail` (fallan: `currentTab() = Defectos`, miga «Defectos › …») y 4 filas `returnDefault` `[caracterización]`; `TestFlowScreensHighlightBacklogTab` con el caso «formulario de defecto» (falla antes, pasa después) |
| 3.7 REFACTOR | [x] | `cancelDefectCreation` unifica `Esc` del formulario y `n` de la confirmación; `gofmt` (normalizando CRLF por archivo), `go vet` y suite en verde |
| 3.8 Evidencia y commit | [x] | Este archivo, Engram y el commit `a3d9f52` |

### Ciclo TDD observado

| Paso | RED observado (comando y resultado) | GREEN observado |
| ---- | ----------------------------------- | --------------- |
| 3.1/3.2 | `go test -count=1 ./internal/tui -run DefectFromStory` → 3 `--- FAIL` (`Screen = "story-detail", cmd nil = true, want the defect form without a command`); `go test -count=1 ./tests/unit/tui/... -run 'StoryDetailActionRows\|StoryDetailArrows\|StoryDetailHelp'` → 3 `--- FAIL` (sin «Reportar defecto»; el resaltado se queda en «Registrar esfuerzo»; la ayuda no lista `d`) | Mismos comandos → `ok` / `ok` |
| 3.3/3.4 | `go test -count=1 ./internal/tui -run DefectFromStory -v` → 5 `--- FAIL` por aserción (`Screen = "defects"` ×3, `defect-story-select` ×2) y 8 `--- PASS` (caracterizaciones y precarga) | Mismo comando → `ok`; luego tabla completa de 3.5 en verde |
| 3.5 | Por mutación (ver abajo): quitar `m.returnTo = returnDefault` de `beginDefectCreation` hace fallar `TestNInDefectsResetsResidualReturnTo` | Con el código restaurado → `ok` |
| 3.6 | `go test -count=1 ./internal/tui -run CurrentTabAndCrumb` → 3 `--- FAIL` (`currentTab() = 6 ("Defectos"), want 2 ("Backlog")`; miga «Proyectos › Alpha › Defectos › Registrar defecto»); `go test ./tests/unit/tui/... -run FlowScreensHighlight` → 1 `--- FAIL` (formulario de defecto) | Ambos → `ok` tras ampliar `inStoryFlow()` y el prefijo de `crumbParts` |

**RED por mutación** (código de producción alterado de forma temporal y restaurado desde copia; el resultado se observó antes de restaurar):

| Mutación | Pruebas que fallan |
| -------- | ------------------ |
| Quitar `m.returnTo = returnDefault` de `beginDefectCreation` | `TestNInDefectsResetsResidualReturnTo` |
| Quitar la rama de ayuda del formulario con historia precargada | `TestDefectFormHelpDependsOnWhereTheStoryComesFrom/historia_precargada_desde_el_detalle` |
| Asignar un `storyID` fijo distinto en `beginDefectCreationForStory` | `TestDefectFromStorySendsPreloadedStoryIDInBody` (y las que dependen de la precarga) |
| Quitar la guarda `defectCreateLoading` de `updateDefectCreateConfirm` y cambiar el destino de `Esc` | `…DoubleSubmitEmitsSingleRequest`, `…EscOnConfirmReturnsToFormKeepingDraft` |
| Quitar la guarda de proyecto/pantalla/`requestID` de `applyDefectCreateResult` y alterar los mensajes de rechazo | `…LateResponsesAreDiscarded/respuesta_de_un_proyecto_anterior`, `…RejectionKeepsConfirmationWithValues` (422 y conexión) |

**Triangulación**: destinos del alta con origen detalle frente a origen Defectos (mismas pantallas, dos `returnTo`); ayuda con y sin historia precargada; rechazo por HTTP 422 y por error de conexión; respuesta tardía por pantalla y por proyecto; cinco salidas del alta desde Defectos.

**Resultado final de la UT**: `go test -count=1 ./internal/tui/... ./tests/unit/tui/...` → `ok` en ambos paquetes; `go test -v` → **366 `--- PASS` de primer nivel, 0 `--- FAIL`** (350 de la UT 2 + 16 pruebas nuevas de primer nivel: 13 `DefectFromStory*`, `DefectFormHelpDependsOnWhereTheStoryComesFrom`, `NInDefectsResetsResidualReturnTo` y `DefectCreationFromDefectsTabKeepsStoryPickerAndReturnsToList`). `go build ./...`, `go vet ./internal/tui/... ./tests/unit/tui/...` y `gofmt` (normalizando CRLF por archivo) sin salida.

### Work Unit Evidence

| Evidencia | Valor |
| --------- | ----- |
| Prueba focal y resultado | `go test -count=1 ./internal/tui/... ./tests/unit/tui/...` (suite completa de ambos paquetes): `ok` / `ok`, 366 PASS de primer nivel, 0 FAIL |
| Harness de ejecución | N/A: `Model.Update` con `httptest`, sin frontera de proceso; recorrido manual pendiente en la fase 7.4 |
| Límite de rollback | Revertir `beginDefectCreationForStory`, la acción `d` y `openDefectForStory` (`story_detail.go`), `cancelDefectCreation`, las ramas `returnStoryDetail` y la ayuda de `defect_creation.go`/`defect_submission.go`, la ampliación de `inStoryFlow()` (`workspace.go`) y el prefijo de la miga (`model.go`), más las pruebas nuevas; el alta desde Defectos no cambia |

### Archivos tocados

| Archivo | Acción | Qué se hizo |
| ------- | ------ | ----------- |
| `internal/tui/story_detail.go` | Modificado | Tercera entrada de `storyActions` (`d`), `openDefectForStory`, `beginDefectCreationForStory` |
| `internal/tui/defect_creation.go` | Modificado | `returnTo = returnDefault` explícito, `cancelDefectCreation`, `Esc` del formulario y del selector de Sprints por `returnTo`, ayuda con historia precargada |
| `internal/tui/defect_submission.go` | Modificado | `n` de la confirmación vía `cancelDefectCreation`; el éxito vuelve al detalle con aviso y sin recargar |
| `internal/tui/workspace.go` | Modificado | `inStoryFlow()` incluye `defect-create-form`, `defect-sprint-select` y `defect-create-confirm` (S1) |
| `internal/tui/model.go` | Modificado | `crumbParts` de `defect-*` usa el prefijo `Backlog › Detalle` cuando `storyFlowCrumb()` lo da |
| `internal/tui/defect_from_story_test.go` | Creado | API simulada con `httptest`, helpers del alta y 16 pruebas de primer nivel |
| `internal/tui/story_flow_test.go` | Modificado | 3 filas `returnStoryDetail` y 4 filas `returnDefault` `[caracterización]` en la tabla de pestaña y miga |
| `tests/unit/tui/story_detail_test.go` | Modificado | Filas, flechas y ayuda con la tercera acción; caso «formulario de defecto» en `TestFlowScreensHighlightBacklogTab` |

### Desviaciones respecto de las tareas y el diseño

1. **Sin esqueleto previo de `beginDefectCreationForStory`** (la tarea 3.1 lo pedía): las pruebas de 3.1 recorren el alta con la tecla `d`, no llaman al símbolo, así que un esqueleto habría sido código muerto. El RED fue por aserción de pantalla.
2. **Pantallas `defect-*` en `inStoryFlow()` por lista explícita** (`defect-create-form`, `defect-sprint-select`, `defect-create-confirm`) en lugar del prefijo `strings.HasPrefix(m.Screen, "defect-")`: solo esas tres pantallas pertenecen al alta; así `defect-detail` y los selectores de estado no pueden heredar un `returnTo` residual. Se reutiliza la rama `inStoryFlow()` que ya iba antes de la de Defectos en `currentTab`, en vez de una rama nueva; la regla S1 sigue en un único lugar.
3. **El éxito del alta usa `backToStoryDetail()`**, no una llamada literal a `openDefects()`: el diseño ya indicaba `openDefects()` en `defect_submission.go:79` solo para `returnDefault`, que se conserva sin cambios.
4. **Texto de ayuda en neutro** («Use ←/→ …») para la variante nueva, según la regla de interfaz del cambio; el texto original con voseo («Usá …») del alta desde Defectos no se toca.
5. **Dos pruebas extra** no listadas: `DefectFromStorySprintQueryFailureShowsErrorAndAllowsRetry` (escenario «Falla la consulta de Sprints» de la spec) y `DefectFormHelpDependsOnWhereTheStoryComesFrom` (tarea 3.4). La segunda se escribió después del código de ayuda; su RED se observó por mutación (tabla de arriba), no antes.
6. **Tamaño**: el commit trae 701 inserciones y 16 eliminaciones frente a las ~230 líneas estimadas; \~480 líneas son `defect_from_story_test.go` (API simulada, helpers y 16 pruebas). Queda dentro de la excepción `size:exception` aceptada; no se recortó nada.
7. **Normalización de fin de línea**: igual que en UT 1 y UT 2 (`tr -d '\r' < archivo | gofmt -l` por archivo, sin salida).
8. **Fuera de alcance, respetado**: no se agregó la guarda del doble envío de Planning Poker; las pestañas viejas y `P`/`E` siguen hasta la UT 5.

### Notas de caracterización (RED no alcanzable de forma directa)

- `EscOnConfirmReturnsToFormKeepingDraft`, `RejectionKeepsConfirmationWithValues`, `DoubleSubmitEmitsSingleRequest`, `LateResponsesAreDiscarded/respuesta_de_un_proyecto_anterior`, `SprintQueryFailureShowsErrorAndAllowsRetry`, las cuatro filas `returnDefault` de `defect-*` y `DefectCreationFromDefectsTabKeepsStoryPickerAndReturnsToList` nacen en verde: fijan comportamiento que ya existía. Las cuatro primeras se verificaron con mutación (tabla de arriba); `SprintQueryFailure…`, las filas `returnDefault` y las cinco salidas del alta desde Defectos no se mutaron.
- Supuesto S1 (sin confirmar por el usuario): sigue aislado en `inStoryFlow()` y su rama de `currentTab` (`workspace.go`); ahora cubre también el alta de defecto iniciada desde el detalle. Se simplifica en la tarea 5.8.

### Notas para la UT 4

- `openStoryFlowWithProject` (aditivo) sigue limitado a `p`/`t`; para la navegación sin mutación de 4.4 (Backlog → detalle → Reportar defecto → `Esc`), `d` no emite ninguna request: el recorrido llega al formulario precargado sin `GET`, y solo `Enter` en el último campo emite `GET /projects/{pid}/sprints/open`.
- Helpers nuevos en `internal/tui/defect_from_story_test.go`: `defectStoryAPI`/`newDefectStoryAPI` (acepta solo Sprints abiertos y el `POST`; `serveDefectsTab()` agrega historias y lista de defectos), `defectDetailModel`, `storyStep`, `storyType`, `openDefectForm`, `defectFormToSprints`, `defectConfirmation`, `defectDraftToConfirmation`, `submitDefect`, `defectsTabDraft`. La UT 4 no los necesita (es migración de Poker y Esfuerzo).
- `returnTo` ya vale `returnStoryDetail` en las tres acciones; en la UT 5 se eliminan las ramas `returnDefault` de Poker y Esfuerzo, pero **no** las de defectos: el alta desde la pestaña Defectos las conserva (`cancelDefectCreation`, `Esc` del selector de Sprints y el éxito con `openDefects()`).

### Estado del entregable

- `apply-progress.md` y `tasks.md` siguen **sin commitear** (van en el commit 6). `openspec/config.yaml` queda fuera de todo commit.

## UT 4 — Migración de pruebas al camino Backlog → detalle → acción (commit 4) — COMPLETA `[migración]`

**Commit**: `7e5b07424e31692a2c2192d8e728081d98ffd533` — `test(tui): recorrer los flujos de estimación y esfuerzo desde el detalle`

Solo pruebas: **ningún archivo de producción cambió** (`git diff --stat HEAD~1 HEAD` toca 9 archivos `_test.go`; 245 inserciones y 280 eliminaciones, saldo neto de 35 líneas menos). Las pestañas Estimación y Esfuerzo, `P`/`E` y `contextual_shortcuts_test.go` siguen intactos hasta la UT 5.

### Estado por tarea

| Tarea | Estado | Evidencia resumida |
| ----- | ------ | ------------------ |
| 4.1 Línea base `[migración]` | [x] | `go test -count=1 -v ./internal/tui/... ./tests/unit/tui/...` → `ok` / `ok`, **366 `--- PASS` de primer nivel (331 subpruebas), 0 `--- FAIL`** antes de tocar nada. `rg -n "openSectionWithProject"` listó **13 sitios de llamada directos** (no ≈ 25; ver desviación 1) más los que llegaban por `openEffortTasks` |
| 4.2 Helper definitivo `[migración]` | [x] | `openSectionWithProject` eliminado; `openStoryFlowWithProject` (Backlog → detalle → `p`/`t` → primer paso aplicado) es el único camino. `openEffortTasks` conserva su firma y delega en él, de modo que `openEffortMembers`, `openEffortFormFor` y las pruebas de Esfuerzo que los usan pasan al camino nuevo sin cambiar de firma |
| 4.3 Pruebas del paso `primary` `[migración]` | [x] | Reescritas sobre el primer paso en `sections_test.go`: `TestSectionAPIErrorStaysOpenAndRefreshes` (500 en `GET /members` y en `GET /stories/s1/tasks`, `r` reintenta), `TestSectionEmptyCollectionShowsAnEmptyState` (proyecto sin integrantes), `TestSectionFilterRefreshAndEscape` (filtro en tareas, `r` lo conserva, primer `Esc` lo limpia y el segundo vuelve al **detalle**, no a Proyectos) y `TestSectionFooterAndHelpOnlyShowValidKeys` (pie y ayuda de Participantes y de Tareas, sin «elegir historia»). Las otras migraciones del paso primario: `TestPlanningPokerParticipantSelectorShowsLoadingAndAPIError`, `TestPlanningPokerFootersAdvertiseContextualSelectionAndVoteKeys`, `TestPlanningPokerCollectsHiddenVotesRevealsResetsAndSaves`, `TestPlanningPokerEmptyMemberListHasClearStateAndCannotStartVoting` y `TestEffortLateTasksResponseAfterEscIsDropped` |
| 4.4 Navegación sin mutación `[migración]` | [x] | `TestSectionNavigationOnlyEmitsGetRequests` recorre Backlog → detalle → Estimar (participantes, voto, `Esc` ×2) → Registrar esfuerzo (tareas, integrantes, formulario, `Esc` ×3) → Reportar defecto (formulario, tres `Enter` hasta el selector de Sprints, `Esc` ×2) y exige que toda request sea `GET` **y** que se hayan pedido `members`, `tasks` y `sprints/open` (sin esa segunda condición un recorrido que no avanzara pasaría en vacío) |
| 4.5 Poker, esfuerzo y avisos `[migración]` | [x] | `planning_poker_test.go` (el helper `pokerConfirmModel` desaparece; `pokerConfirmFromDetail` es el único; destino `story-detail`), `effort_form_test.go` (miga `Backlog › Detalle › Esfuerzo › Registrar`; cuatro destinos de éxito `section` → `story-detail`; el aviso se verifica con `wideView`), `effort_result_test.go` (destino; pruebas de cambio de proyecto reescritas completas con `switchProjectToBacklog`), `effort_test.go` (las tres cadenas de `Esc` terminan en el detalle) y `toast_test.go` |
| 4.6 Prueba interna `[migración]` | [x] | `internal/tui/effort_form_test.go`: `TestOpenSectionResetsEffortFlowKeepingSeq` → `TestOpenStoryFlowResetsEffortFlowKeepingSeq` (reinicia `effort` conservando `seq`) y ahora también comprueba que `pokerVotes` se vacía |
| 4.7 Verificación | [x] | `rg -n "openSectionWithProject" tests internal` → sin coincidencias (rg sale con 1); `go build ./...` y `go vet ./...` sin salida; `gofmt` por archivo (`tr -d '\r' \| gofmt -l`) sin salida; suite en verde (ver abajo). `contextual_shortcuts_test.go` y las pruebas de pestañas viejas **no se tocaron** y siguen en verde |
| 4.8 Evidencia y commit | [x] | Este archivo, Engram y el commit `7e5b074` |

### Prueba de que las pruebas migradas recorren el camino nuevo (RED por mutación)

Las tareas `[migración]` no tienen RED propio (el de las acciones `p`/`t`/`d` se observó en los commits 1 a 3), así que se probó que las pruebas migradas ejercen el camino nuevo alterando de forma temporal el código de producción (script en el directorio temporal de la sesión; cada mutación se restauró desde copia y `git status` confirma que `internal/tui/*.go` no tiene diferencias):

| Mutación temporal | Pruebas de primer nivel que fallan |
| ----------------- | ---------------------------------- |
| Los atajos `p` y `t` del detalle dejan de abrir el flujo (solo `d` funciona) | **71**: todas las de Esfuerzo y Poker migradas (`TestEffortDoubleEnterEmitsSingleRequest`, `TestEffortEscChainEndsAtStoryDetail`, `TestSection…`, `TestPlanningPoker…`, …) más las de la UT 2 |
| El éxito de Esfuerzo vuelve a la raíz vieja (`if false` en `applyEffortSave`) | 10: `TestEffortDoubleEnterEmitsSingleRequest`, `TestEffortEscIgnoredWhileSaving`, `TestEffortFormCorrectAndResubmitAfterRejection`, `TestEffortOnlyExplicitSubmitEmitsThePost`, `TestEffortSaveResultWithoutPendingSaveIsDiscarded`, `TestEffortSecondRegistrationFromDetailStartsWithInitialForm`, `TestEffortSuccessFromDetail…`, `TestEffortToast…` ×3 |
| El éxito de Poker vuelve al Backlog en lugar de al detalle (`if false` en `applyPokerSave`) | 6: `TestPlanningPokerExplicitConfirmationSavesThenRefreshesBacklog`, `TestPokerSaveFromDetail…`, `TestStoryDetailShowsStoryPointsReleidosAfterReload`, `…ShowsLoadingWhileBacklogReloads`, `…ShowsAPIErrorWhenReloadFails`, `…ReportsMissingStoryAfterReload` |
| `Esc` del primer paso ignora el detalle (se borra la rama de `updateSection`) | 13: `TestSectionFilterRefreshAndEscape`, `TestSectionNavigationOnlyEmitsGetRequests`, `TestEffortEscChainEndsAtStoryDetail`, `TestEffortEscFromMembersRestoresTasksSnapshotWithoutRequests`, `TestEffortTaskFilterIsKeptWhenWalkingBackFromForm`, `TestEffortLateTasksResponseAfterEscIsDropped`, `TestEffortProjectSwitchAfterRegistrationOpensBacklogAndDiscardsFlow`, `TestEscFrom…` ×4, … |
| `openStoryFlow` no reinicia `effort` | 1: `TestOpenStoryFlowResetsEffortFlowKeepingSeq` |
| `openStoryFlow` no vacía `pokerVotes` | 1: `TestOpenStoryFlowResetsEffortFlowKeepingSeq` (la aserción nueva) |
| La miga del flujo pierde `Backlog › Detalle` (`storyFlowCrumb`) | 3: `TestEffortFormCrumbAndTitle`, `TestCurrentTabAndCrumbPerScreenAndOrigin`, `TestFlowScreensHighlightBacklogTab` |
| `loadDefectOpenSprints` no emite la request | 12, entre ellas `TestSectionNavigationOnlyEmitsGetRequests` |

**Límite honesto de la mutación**: `TestEffortLateTasksResponseAfterEscIsDropped` no detecta la ausencia de la guarda `m.Screen != "section"` de `applySectionResult`. Tras `Esc` la pantalla es el detalle y una respuesta tardía que poblara el selector oculto no cambiaría la vista; lo que sí protege la prueba es que `Esc` vuelva al detalle (la mutación del `Esc` la rompe). La protección contra la respuesta tardía de **otra historia** es la guarda `StoryID`, cubierta por `TestLateTasksResponseOfAnotherStoryDoesNotPopulateTheSelector` (UT 2).

### Resultado de la suite y comparación de cobertura de pruebas

| Medida | Antes (línea base, 4.1) | Después |
| ------ | ----------------------- | ------- |
| `--- PASS` de primer nivel | 366 | **363** |
| `--- PASS` de subpruebas | 331 | **335** (+4) |
| `--- FAIL` | 0 | 0 |

Comandos: `go test -count=1 -v ./internal/tui/... ./tests/unit/tui/...` → `ok` en ambos paquetes (los nombres se compararon con `comm` entre la salida de antes y la de después); `go build ./...`, `go vet ./...` y `gofmt` sin salida.

Diferencia de 3 pruebas de primer nivel (−5 +2), una por una:

| Prueba | Tratamiento | Motivo |
| ------ | ----------- | ------ |
| `TestEffortEscWalksBackThroughThreeLevels` | **Fusionada** en `TestEffortEscChainEndsAtStoryDetail` | Era la misma cadena formulario → integrantes → tareas → raíz, sin requests. Con el camino nuevo la raíz es el detalle, que ya recorría la prueba de la UT 2; se le agregó la `Esc` final (detalle → Backlog), equivalente de la cuarta `Esc` de la prueba vieja |
| `TestEffortRegistrationReturnsToEffortRootWithToast` | **Eliminada** | Fijaba el destino viejo (raíz de Esfuerzo con recarga de historias). Su escenario vigente (aviso, sin `GET /stories` ni `GET /metrics`, el aviso desaparece con la siguiente tecla) lo cubren `TestEffortSuccessFromDetailReturnsToStoryDetailWithToast`, `TestEffortToastDisappearsOnNextKeyInDetail` y `TestEffortToastFallsBackToFormValuesOnUnreadableBody` |
| `TestEffortSecondRegistrationStartsWithInitialForm` | **Eliminada** | Idéntica a `TestEffortSecondRegistrationFromDetailStartsWithInitialForm` (UT 2) una vez migrado el helper: mismo recorrido, mismas aserciones |
| `TestEffortProjectSwitchAfterRegistrationReopensStoriesAndDiscardsFlow` → `…OpensBacklogAndDiscardsFlow` | **Renombrada y reescrita** | Reabre el **Backlog** del proyecto nuevo (no Esfuerzo); conserva las aserciones de lo que no debe sobrevivir y del formulario limpio, y agrega que cancelar vuelve al detalle de la historia de Beta |
| `TestOpenSectionResetsEffortFlowKeepingSeq` → `TestOpenStoryFlowResetsEffortFlowKeepingSeq` | **Renombrada y extendida** | Ejerce `openStoryFlow`; suma la aserción de `pokerVotes` |

Subpruebas nuevas (+4): `TestSectionAPIErrorStaysOpenAndRefreshes` y `TestSectionFooterAndHelpOnlyShowValidKeys` pasan de un caso a dos (Estimar y Registrar esfuerzo), que triangulan los dos primeros pasos.

### Work Unit Evidence

| Evidencia | Valor |
| --------- | ----- |
| Prueba focal y resultado | `go test -count=1 ./internal/tui/... ./tests/unit/tui/...`: `ok` / `ok`, 363 PASS de primer nivel y 335 de subpruebas, 0 FAIL |
| Harness de ejecución | N/A: solo pruebas con `Model.Update` y `httptest`, sin frontera de proceso; recorrido manual pendiente en 7.4 |
| Límite de rollback | Revertir los 9 archivos de prueba del commit (`git revert 7e5b074`); el código de producción no cambia en esta UT |

### Archivos tocados

| Archivo | Acción | Qué se hizo |
| ------- | ------ | ----------- |
| `tests/unit/tui/sections_test.go` | Modificado | Elimina `openSectionWithProject`; `TestTabsRouteToEachSprint3Section` abre la pestaña en línea (se borra en la UT 5); reescribe las cuatro pruebas del primer paso |
| `tests/unit/tui/sections_flows_test.go` | Modificado | Cuatro llamadas migradas; `TestSectionNavigationOnlyEmitsGetRequests` recorre las tres acciones |
| `tests/unit/tui/planning_poker_test.go` | Modificado | Elimina `pokerConfirmModel`; la confirmación explícita vuelve al detalle y relee los Story Points |
| `tests/unit/tui/effort_test.go` | Modificado | `openEffortTasks` delega en `openStoryFlowWithProject`; elimina `openEffortFormFromDetail`; destinos y `Esc` al detalle |
| `tests/unit/tui/effort_form_test.go` | Modificado | Miga con el prefijo del detalle; destinos de éxito en `story-detail` |
| `tests/unit/tui/effort_result_test.go` | Modificado | Elimina `registered`; cambio de proyecto reescrito (`switchProjectToBacklog`) |
| `tests/unit/tui/toast_test.go` | Modificado | Elimina la prueba del destino viejo de Esfuerzo |
| `tests/unit/tui/story_detail_test.go` | Modificado | Una línea: `openEffortFormFor` reemplaza al helper eliminado |
| `internal/tui/effort_form_test.go` | Modificado | La prueba interna ejerce `openStoryFlow` |

### Desviaciones respecto de las tareas y el diseño

1. **Sitios de llamada**: la tarea 4.1 estimaba ≈ 25; hay **13 directos** (5 en `sections_test.go`, 4 en `sections_flows_test.go`, 2 en `planning_poker_test.go`, 2 en `effort_test.go`) y el resto llega por `openEffortTasks`/`openEffortMembers`/`openEffortFormFor`. Todos migrados.
2. **`TestTabsRouteToEachSprint3Section`** (prueba de las pestañas viejas, que se borra en 5.6) ya no puede usar el helper eliminado (4.7 exige cero coincidencias): abre la pestaña con `openTab` + `run` en línea y conserva sus aserciones.
3. **Helpers de la UT 2 que quedaron redundantes**: `openEffortFormFromDetail` (idéntico a `openEffortFormFor` tras la migración) se elimina y sus tres llamadas pasan al segundo; `registered` y `pokerConfirmModel` también se eliminan (los reemplazan `registeredFromDetail` y `pokerConfirmFromDetail`); `switchEffortProject` se renombra `switchProjectToBacklog`.
4. **Pruebas eliminadas o fusionadas**: las dos eliminadas y la fusionada de la tabla de arriba no admitían una versión migrada que no duplicara a una prueba existente (el escenario viejo desaparece con el destino viejo). No se debilitó ninguna aserción: las que fijaban el destino viejo pasaron a fijar el nuevo.
5. **`TestPlanningPokerExplicitConfirmationSavesThenRefreshesBacklog` conserva su nombre** pero ahora verifica el detalle (Story points «8» leídos del backlog recargado) en lugar de «puntos 8» en la lista del Backlog; la secuencia de requests es la misma (`GET stories`, `GET members`, `PUT`, `GET stories`).
6. **Dos sitios más migrados que el texto de la tarea**: `TestPlanningPokerEmptyMemberListHasClearStateAndCannotStartVoting` y `TestPlanningPokerParticipantSelectorShowsLoadingAndAPIError` (listados en 4.3 como «las dos restantes del paso primary» sin nombre; se interpretó así).
7. **Datos de prueba**: el recorrido de la navegación sin mutación usa el servidor de Esfuerzo (`effortMembersBody`, `effortTasksBody`) en lugar de un servidor propio; los escenarios se verifican con «Primera historia».
8. **Normalización de fin de línea**: igual que en UT 1 a 3 (`tr -d '\r' < archivo | gofmt -l` por archivo, sin salida). Se reformateó `effort_result_test.go` (alineación de comentarios) con `gofmt` antes de confirmar.
9. **Fuera de alcance, respetado**: no se tocó el doble envío de Planning Poker ni código de producción.

### Notas de caracterización

- No se agregó ninguna prueba `[caracterización]` nueva. Las migradas son `[migración]`: su RED lo aportaron los commits 1 a 3 y la tabla de mutaciones de arriba.

### Notas para la UT 5

- **Ramas `returnDefault` sin cobertura de pruebas desde este commit** (perfil de cobertura de `go test -coverpkg=./internal/tui`): `applyEffortSave` → `return m.openSection(sectionEffort)` (`effort_form.go:237`) y `applyPokerSave` → destino Backlog con recarga (`planning_poker.go:197-200`) ya no las recorre ninguna prueba; la tarea 5.5 las elimina, así que no hace falta caracterizarlas. Entre los commits 4 y 5 solo son alcanzables con `P`/`E` o las pestañas viejas.
- **No eliminar** las ramas `returnDefault` de defectos (`cancelDefectCreation`, `Esc` del selector de Sprints, éxito con `openDefects()`): el alta desde la pestaña Defectos las usa y sus pruebas (`defect_create_test.go`, `defect_submission_test.go`, `defect_toast_test.go`, `TestDefectCreationFromDefectsTabKeepsStoryPickerAndReturnsToList`) siguen vivas.
- **Qué pruebas siguen pasando por el camino viejo y se borran/reescriben en la UT 5**: `contextual_shortcuts_test.go` (completo, 5.6), `TestTabsRouteToEachSprint3Section` (5.6), `internal/tui/sections_test.go` (5.6) y las de pestañas en `workspace_test.go`, `keys_test.go`, `dashboard_scope_test.go` (5.1). Las constantes `tabEstimation`/`tabEffort` de `tests/unit/tui/sections_test.go` las siguen usando `workspace_test.go` y `TestTabsRoute…`; se renumeran en 5.1 (`tabDefects = "4"`).
- `internal/tui/story_flow_test.go` conserva filas `returnDefault` de `section`/`planning-poker`/`effort-form` (`flowScreenModel(..., sectionStepPrimary, returnDefault)`) que fijan la pestaña y la miga viejas; 5.8 las simplifica junto con `currentTab`.
- Helpers de prueba vigentes: `openStoryFlowWithProject` (solo `p`/`t`), `openEffortTasks`/`openEffortMembers`/`openEffortFormFor`, `registerEffort`/`registeredFromDetail`, `pokerConfirmFromDetail`/`pokerConfirmFrom`/`savedPoker`, `switchProjectToBacklog` y `stepRunning`.

### Estado del entregable

- `apply-progress.md` y `tasks.md` siguen **sin commitear** (van en el commit 6). `openspec/config.yaml` queda fuera de todo commit.

## UT 5 — Retiro de pestañas y código muerto (commit 5) — COMPLETA

**Commit**: `cd7979c1ee0b9ad9ce2911040024eac07e78854f` — `refactor(tui): retirar las pestañas Estimación y Esfuerzo y el código muerto de secciones`

### Estado por tarea

| Tarea | Estado | Evidencia resumida |
| ----- | ------ | ------------------ |
| 5.1 RED cuatro pestañas | [x] | Fallan por aserción `TestTabBarShowsFourTabsWithoutEstimationOrEffort` (barra con seis pestañas), `TestDigitFourOpensDefects` (`4` abría `section`), `TestDigitsFiveAndSixDoNothing` (`5` → `section`, `6` → `defects`), `TestTabWrapsFromDefectsToDashboard`, `TestShiftTabFromDefectsOpensSprints`, `TestRootHelpListsOneToFourAndTabKeys` (ayuda `1–6`) y `TestDefectsTabOpensProjectDefectList` (con `tabDefects = "4"`). `TestTabsSwitchWithTabAndDigits` y `TestShiftTabFromDashboardOpensDefects` nacen en verde `[caracterización]` |
| 5.2 RED `P`/`E` | [x] | Fallan 4 subpruebas de `TestBacklogUppercasePAndEDoNotStartFlows` (`P`/`E` abrían `section` con comando, con y sin historias) y `TestBacklogHelpAndFooterDoNotOfferPAndE` (la ayuda listaba `P`/`E`); `TestBacklogLowercasePStartsTheSearch` nace en verde `[caracterización]` |
| 5.3 GREEN pestañas y teclas | [x] | `workspaceTabs` con cuatro entradas (el `1–%d` del pie y `tabForKey` derivan de `len`), `openTab` sin los dos casos, `P`/`E` y `openSectionForStory` fuera de `backlog_screens.go`, bindings sin `P`/`E` |
| 5.4 Código muerto de `sections.go` | [x] | Ver «Símbolos eliminados» |
| 5.5 Destinos | [x] | `applyPokerSave` y `applyEffortSave` van siempre al detalle; el `Esc` del primer paso es incondicional; `returnDefault` queda solo para el alta de defecto desde Defectos |
| 5.6 Pruebas obsoletas | [x] | Borrados `contextual_shortcuts_test.go`, `internal/tui/sections_test.go` y `TestTabsRouteToEachSprint3Section`; ver «Mapeo de pruebas» |
| 5.7 Referencias sueltas | [x] | `rg -n "sectionDefects\|sectionStepPrimary\|sectionStepSprints\|preselected\|openSectionForStory\|primaryUsesStories\|storyNextStep\|openSection\(\|tabEstimation\|tabEffort\|primaryHint\|primaryEmpty\|section\.primary" internal tests cmd` → sin coincidencias (rg sale con 1); `rg -n "Estimaci\|Esfuerzo" internal/tui/workspace.go` → sin coincidencias |
| 5.8 `currentTab`, `tabRoot` y miga | [x] | `currentTab` sin ramas de Estimación/Esfuerzo; `inStoryFlow()` incondicional para `section`/`planning-poker`/`effort-form` y condicional a `returnTo` solo para las tres pantallas `defect-*` compartidas con Defectos; `tabRoot` sin `section`; `crumbParts` usa `tabDefects.title()` en lugar de `sectionDefects.title()`. S1 sigue en un único lugar (`currentTab`) |
| 5.9 REFACTOR | [x] | Comentarios actualizados (`returnTarget`, `storyFlowCrumb`, `inStoryFlow`, `tabForKey`); `gofmt` por archivo, `go vet ./...`, `go build ./...` y suite en verde |
| 5.10 Evidencia y commit | [x] | Este archivo, Engram y el commit |

### Ciclo TDD observado

| Paso | RED observado (comando y resultado) | GREEN observado |
| ---- | ----------------------------------- | --------------- |
| 5.1/5.3 (pestañas) | `go test ./tests/unit/tui/... -run 'TabBar\|TabsSwitch\|DigitFour\|DigitsFive\|TabWraps\|ShiftTab\|RootHelp\|DefectsTabOpens\|EnterOnAProject'` → 7 `--- FAIL` por aserción (ver tabla de tareas) | Mismo comando y la suite de ambos paquetes → `ok` / `ok` tras `workspace.go` |
| 5.2/5.3 (`P`/`E`) | `go test ./tests/unit/tui/... -run 'BacklogUppercase\|BacklogLowercase\|BacklogHelpAndFooter'` → 5 `--- FAIL` (`P: Screen = "section", command = true, want the backlog without a command`; la ayuda aún ofrecía «Estimar la historia resaltada», `│   P   ` y `│   E   `) | Mismo comando → `ok` tras quitar `case "P"`/`case "E"` y los bindings |
| 5.4–5.8 (eliminación) | Sin RED propio: es eliminación verificable; la red de seguridad es la suite existente más las pruebas de 5.1/5.2 | `go build ./...`, `go vet ./...` y suite → `ok`; `rg` sin referencias (5.7) |

**Corrección del propio RED**: `TestShiftTabFromDefectsOpensSprints` nació en verde por casualidad (con `4` el código viejo abría Estimación y `Shift+Tab` desde ella lleva a Sprints). Se le agregó la precondición `4 → defects` (y lo mismo a `TestTabWrapsFromDefectsToDashboard`) y entonces falla por aserción (`4: Screen = "section", want defects`).

**Pruebas `[caracterización]` (RED no alcanzable)**: `TestTabsSwitchWithTabAndDigits` (`Tab` y `3` ya funcionaban), `TestShiftTabFromDashboardOpensDefects` (con seis pestañas Defectos ya era la última), `TestBacklogLowercasePStartsTheSearch` (la `p` minúscula ya iniciaba la búsqueda), `TestStoryDetailEscReturnsToBacklogKeepingFilterAndCursorWithoutRequests` (cubre un hueco: ninguna prueba fijaba de punta a punta que `Esc` del detalle conserva filtro y cursor sin requests), y las dos filas nuevas de `TestCurrentTabAndCrumbPerScreenAndOrigin` (`defects` y `defect-detail` con miga «Defectos»).

### Resultado de la suite y comparación de pruebas

| Medida | Antes (línea base) | Después |
| ------ | ------------------ | ------- |
| `--- PASS` de primer nivel | 363 | **361** |
| `--- PASS` de subpruebas | 335 | **328** |
| `--- FAIL` | 0 | 0 |

Comandos: `go test -count=1 -v ./internal/tui/... ./tests/unit/tui/...` → `ok` / `ok` antes y después (nombres comparados con `comm`). `go build ./...`, `go vet ./...` y `gofmt` por archivo (`tr -d '\r' < archivo | gofmt -l`) sin salida. `go test -count=1 ./internal/... ./tests/unit/... ./cmd/...` → todos los paquetes `ok`; las pruebas de integración (`tests/integration/...`, Testcontainers) no se ejecutaron (requieren Docker).

Diferencia de −2 pruebas de primer nivel (−14 +12):

| Prueba | Tratamiento | Cubierta por |
| ------ | ----------- | ------------ |
| `TestBacklogPOpensEstimationParticipantsForHighlightedStory` | **Eliminada** (borrado de `contextual_shortcuts_test.go`) | `TestStoryDetailShortcutsStartActions/p` (`GET /members`, «Participantes»), `TestStoryDetailEnterRunsHighlightedAction` (Estimar abre los participantes de la misma historia, sin selector de historias) y `TestStoryDetailOpensTheHighlightedStoryUnderFilter` (el detalle abre la historia resaltada); `P` en la lista: `TestBacklogUppercasePAndEDoNotStartFlows` |
| `TestBacklogEOpensEffortTasksForHighlightedStory` | **Eliminada** | `TestStoryDetailShortcutsStartActions/t` (`GET /stories/s1/tasks`, «Tareas»), `TestStoryDetailEnterRunsHighlightedAction`, y `TestLateTasksResponseOfAnotherStoryDoesNotPopulateTheSelector` (tareas de la segunda historia, «Segunda historia»); `E` en la lista: `TestBacklogUppercasePAndEDoNotStartFlows` |
| `TestEscFromPreselectedSelectorReturnsToBacklog` | **Eliminada** | `TestEscFromPokerMembersReturnsToStoryDetailWithoutRequests`, `TestEscFromEffortTasksReturnsToStoryDetailWithoutRequests`, `TestEscFromFirstStepAfterAPIErrorReturnsToStoryDetail`, y la nueva `TestStoryDetailEscReturnsToBacklogKeepingFilterAndCursorWithoutRequests` (el segundo `Esc`). **Cambio de comportamiento especificado**: la prueba vieja exigía una recarga del backlog al volver; el detalle ya no recarga (Esc sin requests) |
| `TestDeeperEscStillGoesBackOneStep` | **Eliminada** | `TestEffortEscChainEndsAtStoryDetail` (formulario → integrantes → tareas → detalle → Backlog) y `TestEffortEscFromMembersRestoresTasksSnapshotWithoutRequests` |
| `TestShortcutsDoNothingOnEmptyBacklog` | **Eliminada** | `TestBacklogUppercasePAndEDoNotStartFlows/lista_vacía`, `TestBacklogEnterOnEmptyListDoesNotOpenDetail`, `TestBacklogHelpAndFooterDoNotOfferPAndE` y `TestStoryDetailActionsDoNothingWhileLoading` (internal) |
| `TestLowercasePStillStartsASearch` | **Renombrada y movida** a `backlog_test.go` | `TestBacklogLowercasePStartsTheSearch` (mismas aserciones) |
| `TestContextualShortcutFlowOnlyReads` | **Eliminada** | `TestSectionNavigationOnlyEmitsGetRequests` (Backlog → detalle → las tres acciones → `Esc`, solo `GET`) |
| `TestBacklogHelpListsPAndE` | **Invertida** | `TestBacklogHelpAndFooterDoNotOfferPAndE` (ya no ofrece `P`/`E`); la ayuda positiva del detalle: `TestStoryDetailHelpListsActionShortcutsAndNotTabs` |
| `TestBacklogShortcutsResetAResidualReturnTo` (internal) | **Eliminada** | Existía solo para que `P`/`E` no heredaran un `returnTo` residual; sin `P`/`E` no hay camino. El reinicio sigue cubierto por `TestOpenTabResetsReturnTo` (4 pestañas), `TestSelectProjectResetsStoryDetailAndReturnTo` y `TestNInDefectsResetsResidualReturnTo` |
| `TestSectionLastStepDoesNotShowNotice` (`internal/tui/sections_test.go`) | **Eliminada** | Probaba el paso de Sprints de `sectionDefects`; el escenario («elegir en un paso sin siguiente no emite comando ni muestra avisos») desapareció con el código: los pasos que quedan (integrantes, tareas) siempre tienen siguiente |
| `TestTabsRouteToEachSprint3Section` | **Eliminada** | Fijaba las pestañas retiradas; las pestañas vigentes las cubren `TestTabsSwitchWithTabAndDigits`, `TestDigitFourOpensDefects` y `TestDefectsTabOpensProjectDefectList` |
| `TestTabsSwitchWithTabDigitsAndShiftTab` | **Dividida** | `TestTabsSwitchWithTabAndDigits`, `TestDigitFourOpensDefects` y `TestShiftTabFromDefectsOpensSprints` |
| `TestShiftTabFromDashboardWrapsToDefects` | **Renombrada** | `TestShiftTabFromDashboardOpensDefects` (mismo cuerpo) |
| `TestTabRootHelpListsTabKeys` | **Reescrita** | `TestRootHelpListsOneToFourAndTabKeys` (exige `1–4` y rechaza `1–6`) |

Subpruebas: se eliminan 5 filas de `TestCurrentTabAndCrumbPerScreenAndOrigin` (las de las pestañas viejas, que ya no son alcanzables), 2 de `TestOpenTabResetsReturnTo` (las pestañas Estimación y Esfuerzo), 2 de `TestTabsRouteToEachSprint3Section`, 2 de `TestEscFromPreselectedSelectorReturnsToBacklog` y 2 de `TestBacklogShortcutsResetAResidualReturnTo`; se agregan 4 de `TestBacklogUppercasePAndEDoNotStartFlows` y 2 filas de `TestCurrentTabAndCrumbPerScreenAndOrigin`.

### Cobertura de las funciones tocadas (`go test -coverpkg=./internal/tui`)

`applyEffortSave` 100 %, `currentTab`, `inStoryFlow`, `tabRoot`, `tabForKey` y `openTab` 100 %, `sectionBindings` 100 %, `updateSection` 89,5 %, `crumbParts` 97,8 %. La rama `returnDefault` de éxito de `applyEffortSave` y de `applyPokerSave` (sin pruebas desde la UT 4) desapareció con el código. `applyPokerSave` queda en 57,1 % por ramas **preexistentes** sin prueba (error de conexión y respuesta no interpretable, líneas 179–182 y 189–192); no se tocan: fuera de alcance.

### Símbolos eliminados (prueba de que no tienen referencias)

Prueba: `rg -n "<símbolo>" internal tests cmd` sin coincidencias, `go build ./...`, `go vet ./...` y suite en verde.

| Archivo | Símbolos |
| ------- | -------- |
| `internal/tui/sections.go` | `sectionDefects` (y su rama de `title`), `sectionStepSprints`, `sectionStepPrimary` (reemplazado por el centinela `sectionStepNone`), `sectionKind.primaryUsesStories`, `storyNextStep`, `primaryHint`, `primaryEmptyTitle`, `primaryEmptyHint`, `sectionState.primary`, `sectionState.preselected`, `Model.openSection`, las ramas primaria y de Sprints de `loadSectionStepCmd`, `sectionItems` y `viewSection`, y las asignaciones de `primary` en `applySectionResult`, `updateSection` y `chooseSectionItem` |
| `internal/tui/workspace.go` | `tabEstimation`, `tabEffort` (constantes, `title`, `openTab`, `currentTab`) y el caso `section` de `tabRoot` |
| `internal/tui/backlog_screens.go` | `case "P"`, `case "E"` y `openSectionForStory` |
| `internal/tui/keys.go` | Filas `P`/`E` del Backlog y las ramas primaria/Sprints de `sectionBindings` |
| `internal/tui/model.go` | La rama `sectionStepSprints` de `crumbParts`; `sectionDefects.title()` pasa a `tabDefects.title()` |
| `internal/tui/planning_poker.go`, `effort_form.go` | Ramas `returnDefault` de éxito (`applyPokerSave` → Backlog con recarga; `applyEffortSave` → `openSection(sectionEffort)`) |
| Pruebas | `tests/unit/tui/contextual_shortcuts_test.go`, `internal/tui/sections_test.go` (archivos); `TestTabsRouteToEachSprint3Section`; constantes `tabEstimation`/`tabEffort` de `tests/unit/tui/sections_test.go` |

**No se eliminó** (lo sigue usando otra pieza): `sectionState`, `sectionKind`, `storySelectorItem` (alta de defecto desde Defectos y `openStoryFlow`), `sprintSelectorItems` (Dashboard), `viewSelector` (Dashboard y Defectos), `leaveWorkspace` y `listStoriesCmd`, y las ramas `returnDefault` del alta de defecto (`cancelDefectCreation`, `Esc` del selector de Sprints, éxito con `openDefects()`).

### Work Unit Evidence

| Evidencia | Valor |
| --------- | ----- |
| Prueba focal y resultado | `go test -count=1 ./internal/tui/... ./tests/unit/tui/...`: `ok` / `ok`, 361 PASS de primer nivel y 328 de subpruebas, 0 FAIL |
| Harness de ejecución | N/A: `Model.Update` con `httptest`, sin frontera de proceso; recorrido manual pendiente en 7.4 |
| Límite de rollback | `git revert` del commit: vuelven las seis pestañas, `P`/`E`, los pasos `primary` y las ramas `returnDefault`; el detalle y sus acciones (UT 1 a 3) siguen funcionando |

### Archivos tocados

| Archivo | Acción | Qué se hizo |
| ------- | ------ | ----------- |
| `internal/tui/workspace.go` | Modificado | Cuatro pestañas; `openTab`, `currentTab`, `inStoryFlow`, `tabRoot` simplificados |
| `internal/tui/sections.go` | Modificado | Retiro de `sectionDefects`, pasos `primary` y de Sprints, `preselected`, `openSection`; `sectionStepNone` |
| `internal/tui/backlog_screens.go` | Modificado | Sin `P`/`E` ni `openSectionForStory` |
| `internal/tui/keys.go` | Modificado | Bindings del Backlog y de `section` sin las filas retiradas |
| `internal/tui/model.go` | Modificado | `crumbParts` sin Sprints y con `tabDefects.title()`; comentario de `storyFlowCrumb` |
| `internal/tui/planning_poker.go`, `internal/tui/effort_form.go` | Modificado | Éxito siempre al detalle |
| `internal/tui/story_detail.go` | Modificado | Comentario de `returnTarget` |
| `internal/tui/story_flow_test.go` | Modificado | Sin filas de pestañas viejas ni prueba de `P`/`E`; filas de `defects`/`defect-detail`; `sectionStepNone` |
| `internal/tui/sections_test.go`, `tests/unit/tui/contextual_shortcuts_test.go` | Eliminados | Ver mapeo |
| `tests/unit/tui/workspace_test.go`, `keys_test.go`, `sections_test.go` | Modificado | Cuatro pestañas, `tabDefects = "4"`, dígitos, Tab/Shift+Tab, ayuda `1–4`; sin constantes ni prueba de las pestañas viejas |
| `tests/unit/tui/backlog_test.go` | Modificado | `P`/`E` sin efecto, `p` busca, ayuda y pie sin `P`/`E` |
| `tests/unit/tui/story_detail_test.go` | Modificado | `Esc` del detalle conserva filtro y cursor sin requests |

### Desviaciones respecto de las tareas y el diseño

1. **`tests/unit/tui/dashboard_scope_test.go` no se modificó** (la tarea 5.1 lo menciona): `TestOldMetricsSectionIsUnreachable` recorre los dígitos `1` a `6` para probar que ninguno abre la sección Métricas; con cuatro pestañas `5` y `6` simplemente no hacen nada y la aserción sigue siendo válida (y triangula esa regla).
2. **`TestRootHelpListsOneToFourAndTabKeys` reutiliza el ejemplo de Sprints** (`openSprintList`), como la prueba que reemplaza, y además rechaza `1–6`.
3. **`TestEnterOnAProjectOpensItsWorkspaceTabs` conserva su nombre**: ahora exige las cuatro pestañas y la miga del Dashboard; la ausencia de Estimación y Esfuerzo y el orden con dígitos los fija la prueba nueva `TestTabBarShowsFourTabsWithoutEstimationOrEffort`.
4. **`inStoryFlow()` se mantiene** (el diseño D8 la muestra inline en `currentTab`): la usan `currentTab` y `storyFlowCrumb`, y concentra la regla S1 en un único lugar. Es incondicional para `section`/`planning-poker`/`effort-form` y condicional a `returnTo` solo para las tres pantallas `defect-*` que comparte con Defectos, como pide el diseño.
5. **`sectionStepNone` no tiene uso en código de producción**, solo en las pruebas internas (`flowScreenModel`): es el centinela de valor cero que exige el diseño (D4) para que un `sectionState` sin iniciar no se confunda con el paso de integrantes.
6. **El texto de la tarea decía `1–%d` en `keys.go` ≈ línea 73**: ya derivaba de `len(workspaceTabs)`, por lo que cambió solo con el slice.
7. **Normalización de fin de línea**: igual que en UT 1 a 4 (`tr -d '\r' < archivo | gofmt -l` por archivo, sin salida); se editó con scripts que conservan CRLF.
8. **Fuera de alcance, respetado**: no se tocó el doble envío de Planning Poker ni `README.md` (UT 6).

### Notas para la UT 6 y la fase 7

- La UT 6 debe documentar: cuatro pestañas (`1`–`4`), `Enter` abre el detalle desde el Backlog, `e` modifica, `P`/`E` ya no son atajos (`p` minúscula inicia la búsqueda), acciones del detalle `p` Estimar, `t` Registrar esfuerzo y `d` Reportar defecto, y `Esc` vuelve de a un nivel.
- El código ya no contiene ninguna referencia a las pestañas retiradas; `rg -n "Estimaci|Esfuerzo" internal/tui/workspace.go` está vacío. «Estimación»/«Esfuerzo» siguen apareciendo solo como títulos de los flujos del detalle (`sectionKind.title()`) y en las migas `Backlog › Detalle › Estimación|Esfuerzo › …`.
- Recorrido manual (7.4): no ejecutado; requiere la API y PostgreSQL (Docker). La suite de integración tampoco corre en este entorno.
- `apply-progress.md` y `tasks.md` siguen **sin commitear** (van en el commit 6). `openspec/config.yaml` queda fuera de todo commit.

## UT 6 — README y artefactos del cambio (commit 6) — COMPLETA `[solo documentación]`

**Commit**: `docs(tui): actualizar el README para las acciones desde la historia` (hash en el historial de git; el commit incluye este archivo, por eso no puede citarse aquí).

**TDD estricto**: la unidad no contiene código de producción ni de prueba (solo `README.md` y artefactos SDD), así que no hay RED/GREEN/REFACTOR aplicable y **no se inventa evidencia de RED**. La verificación es documental (`rg`) más `go build ./...` para confirmar que nada de código cambió.

### Estado por tarea

| Tarea | Estado | Evidencia resumida |
| ----- | ------ | ------------------ |
| 6.1 README | [x] | Ver «Cambios en `README.md`» y «Verificación». El README ya no menciona pestañas Estimación/Esfuerzo, `1`–`6` ni `P`/`E` |
| 6.2 Delta de spec (read-only) | [x] | El `rg` devuelve solo menciones permitidas (ver «Verificación»); no se corrigió nada |
| 6.3 Cierre documental y commit | [x] | Este archivo, Engram y el commit; se incorporan `README.md` y todo `openspec/changes/tui-refactor-acciones-desde-historia/`, sin `openspec/config.yaml` |

### Cambios en `README.md` (verificados contra `internal/tui`)

| Zona (líneas originales) | Antes | Después |
| ------------------------ | ----- | ------- |
| Barra de pestañas (45) | `1 Dashboard` … `4 Estimación` · `5 Esfuerzo` · `6 Defectos` | `1 Dashboard` · `2 Backlog` · `3 Sprints` · `4 Defectos` |
| Tabla de pestañas (50) | `1`–`6` | `1`–`4` |
| Tabla de listas (66, 73) | `e` Editar (Proyectos); fila `P` / `E` | `e` Editar (Proyectos y Backlog); fila `P` / `E` eliminada |
| Aviso de éxito (84-85) | «vuelve a la pantalla de origen, recarga su lista…» | Se agrega «(las acciones del detalle de una historia vuelven al detalle, ver «Secciones»)» |
| Tabla «Secciones» (95-98) | Backlog «editar con `Enter`»; filas Estimación y Esfuerzo | Backlog: lista → detalle de la historia → Estimar, Registrar esfuerzo o Reportar defecto (crear con `n`, editar con `e`); filas Estimación y Esfuerzo eliminadas |
| Párrafo de Defectos (103) | «Las pestañas que encadenan selectores…» | «Los flujos que encadenan selectores…» |
| Párrafos de `P`/`E` y Esfuerzo (106-113) | `P`/`E` abren las pestañas Estimación/Esfuerzo; Esfuerzo vuelve al selector de historias | Párrafo del Backlog (`Enter` detalle, `e` editar, `n` crear, `/` y letras no reservadas —incluidas `p`, `P` y `E`— inician la búsqueda); tabla de acciones del detalle (`p`, `t`, `d`, también con `↑`/`↓` + `Enter`; inactivas mientras el backlog se recarga); párrafo de `Esc` por niveles y vuelta al detalle (Poker relee el backlog; esfuerzo y defecto muestran el aviso `✓`); Defectos sigue listando/gestionando y `n` conserva el selector de historias; Registrar esfuerzo vuelve al detalle con el aviso |

Hechos contrastados con el código: `Enter`/`e`/`n`/`/`/`r` y la rama `default` que inicia el filtro con cualquier otra letra (`backlog_screens.go` líneas 197-215); bindings del Backlog (`keys.go` líneas 104-111); acciones `p`/`t`/`d` y su desactivación durante la carga (`story_detail.go`, `storyActionsAvailable`); volver al Backlog con filtro y cursor (`TestStoryDetailEscReturnsToBacklogKeepingFilterAndCursorWithoutRequests`). No se creó ninguna sección nueva ni se reestructuró el README.

### Verificación (solo documental)

| Comando | Resultado |
| ------- | --------- |
| `rg -n "1–6\|1-6\|P/E\|seis pesta\|pestaña Estimaci\|pestaña Esfuerzo\|Estimaci" README.md` | Sin coincidencias (rg sale con 1) |
| `rg -n "Estimaci\|P/E\|1–6\|seis pesta" README.md` (comando de verificación de la tabla de UT) | Sin coincidencias (rg sale con 1) |
| `rg -n "Esfuerzo" README.md` | Sin coincidencias: «esfuerzo» queda en minúscula como flujo («Registrar esfuerzo») y en la sección de la API |
| `go build ./...` | Sin salida (compila; ningún `.go` cambió) |
| `rg -n "pestaña Estimación\|pestaña Esfuerzo\|\`P\`\|\`E\`\|1–6\|seis pestañas" …/specs/tui/spec.md` (6.2) | 14 líneas. 10 están dentro de `(Previously: …)`, `(Reason: …)` o `(Migration: …)` (269, 397, 487, 544, 620, 673, 737, 787, 858, 859). Las otras 4 (268, 295, 297, 819) son redacción **vigente y negativa**: «`P` y `E` mayúsculas ya no inician ningún flujo», el escenario «`P` y `E` ya no inician flujos» y «no lista `P` ni `E`». Describen el comportamiento nuevo, no una referencia suelta a lo retirado; no se corrigen |

`git status --short` tras el commit: solo ` M openspec/config.yaml`.

### Desviaciones respecto de las tareas

1. **Sin estados «TUI-07»/«disponible (TUI-08)»**: al eliminar las filas Estimación y Esfuerzo desaparecen esas marcas de estado; el Backlog queda «disponible» (no se asignó un identificador de historia nuevo).
2. **No se usó `P` de la tarea 6.1**: la expresión daría falsos positivos con cualquier `P` suelta (por ejemplo en «PDF» o «P»/«E» como letras de la ayuda); se verificó con las expresiones de la tabla de arriba y leyendo el diff completo.
3. **Fila `e` de la tabla de listas** pasó a «Proyectos y Backlog» y el párrafo del aviso de éxito recibió una aclaración breve: ambos cambios no estaban enumerados en la tarea pero son necesarios para que el README no contradiga el comportamiento (la spec «Listas navegables» ya exige `e` en el Backlog).
4. **Fin de línea**: `README.md` está en CRLF en el árbol de trabajo; se editó con un script que conserva CRLF. Git lo normaliza al commitear.
5. **Los artefactos SDD (exploración, propuesta, delta de spec, diseño, tareas y este archivo) viajan en este commit**, como indican la tarea 6.3 y la nota N5.

### Notas para la fase 7

- 7.1: ejecutar `gofmt -l internal tests cmd` (en este árbol CRLF, por archivo con `tr -d '\r' < archivo | gofmt -l`), `go build ./...`, `go vet ./...` y `go test ./...`; las pruebas de integración necesitan Docker (en este equipo fallan con `rootless Docker is not supported on Windows`); registrar qué no se ejecutó.
- 7.3: el comando de búsqueda sobre el README y el delta de spec arrojará las mismas coincidencias del delta que esta unidad ya clasificó (menciones `Previously`/`Reason`/`Migration` y las cuatro líneas negativas vigentes: 268, 295, 297 y 819); el README no tiene ninguna.
- 7.4: el recorrido manual con la API real sigue sin ejecutarse (requiere PostgreSQL/Docker); registrar el resultado o la razón.
- Tras 6, el árbol debe quedar solo con ` M openspec/config.yaml` (ajeno al cambio).
