# Informe de verificación: tui-refactor-acciones-desde-historia

## Resultado

**CRITICAL 0 · WARNING 5 · SUGGESTION 7.** La implementación (commits `b3c10ca`, `84850d9`, `a3d9f52`, `7e5b074`, `cd7979c`, `e0cf312`) compila, pasa `go vet`, está formateada y todas las pruebas sin Docker pasan (361/361 de primer nivel en `internal/tui` y `tests/unit/tui`). No se encontró ningún requisito roto ni ausente. Quedan sin ejecutar el recorrido manual con la API real (tarea 7.4: no hay daemon de Docker ni PostgreSQL) y las pruebas de integración (necesitan Docker). El supuesto S1 sigue **sin confirmar por el usuario**. Este informe es diagnóstico y no condiciona el archivo.

Fecha: 2026-11-04 · Rama `feat/tui-refactor-acciones-desde-historia` · Modo: openspec · Strict TDD: activo · Skill aplicada: `go-testing` (registro `.atl/skill-registry.md`).

## Alcance y estado de las tareas

Artefactos leídos: `proposal.md`, `specs/tui/spec.md` (16 requisitos: 2 ADDED, 13 MODIFIED, 1 REMOVED; 135 escenarios), `design.md`, `tasks.md`, `apply-progress.md`. `openspec/config.yaml` (modificado sin commitear, ajeno al cambio) se ignoró; no define `rules.verify`.

- Tareas 1.1 a 6.3: `[x]` (UT 1 a 6 con commit). Se observó, no se reescribió.
- Fase 7: `[ ]` 7.1 a 7.7 y los 12 criterios de cierre quedan sin marcar. Estado observado: 7.1 (gofmt/build/vet/test) y 7.2/7.3 (revisión y `rg`) se ejecutaron en esta verificación; **7.4 NO EJECUTADA**; 7.5 no hizo falta (sin correcciones); 7.6 cumple (`apply-progress.md` físico y `git status` solo con `openspec/config.yaml`); 7.7 es posterior al merge.

## Comandos ejecutados y resultados

| # | Comando | Resultado |
|---|---------|-----------|
| 1 | `go build ./...` | exit 0, sin salida |
| 2 | `go vet ./...` | exit 0, sin salida |
| 3 | `gofmt -l` (con CR eliminado) por cada uno de los 25 `.go` cambiados desde `main` (`git diff --name-only --diff-filter=d main...HEAD -- '*.go'`) | sin salida: 25/25 formateados |
| 4 | `go test -count=1 -v ./internal/tui/... ./tests/unit/tui/...` | exit 0. `ok internal/tui 7.1s`, `ok tests/unit/tui 5.2s`. **361 `--- PASS` de primer nivel, 0 `--- FAIL`; 328 subpruebas PASS, 0 FAIL; 0 SKIP**. Los 361 coinciden con las 361 funciones `Test*` |
| 5 | `go test -count=1 ./internal/... ./tests/unit/... ./cmd/...` | exit 0: 31 paquetes `ok`, 0 `FAIL` |
| 6 | `go test -count=1 ./tests/integration/...` | **NO EJECUTADO en la práctica** (necesita Docker): 9 paquetes `FAIL` (`task`, `story`, `sprint`, `report`, `projectmember`, `project`, `metrics`, `effort`, `defect`; 132 `--- FAIL` de primer nivel) con `rootless Docker is not supported on Windows, failed to create Docker provider`; `migrations` `ok` (no usa contenedores). No se cuenta como regresión; **no se confirmó contra `main`** (sin cambiar de rama). El cambio no toca `tests/integration` |
| 7 | `go test -count=1 -coverpkg=./internal/tui -coverprofile ... ./internal/tui/... ./tests/unit/tui/...` | `ok` ambos (cobertura informativa, ver más abajo) |
| 8 | `git grep -h "^func Test"` por commit (`main`, `b3c10ca`, `84850d9`, `a3d9f52`, `7e5b074`, `cd7979c`, `e0cf312`) | 297, 320, 350, 366, 363, 361, 361: **coinciden con los conteos de `apply-progress.md`** |

Docker: `docker --version` 29.8.1 pero `docker info` falla (`open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified`); no hay `psql`, `pg_ctl` ni `postgres` en el PATH; `cmd/api` exige `DATABASE_URL` (`internal/api/api.go:38`). No se levantó ningún servicio.

## Matriz requisito → escenario → prueba → código

Leyenda de archivos: `SD` = `tests/unit/tui/story_detail_test.go`; `SF` = `internal/tui/story_flow_test.go`; `DFS` = `internal/tui/defect_from_story_test.go`; `PP` = `tests/unit/tui/planning_poker_test.go`; `ET`/`EF`/`ER` = `tests/unit/tui/effort_test.go`/`effort_form_test.go`/`effort_result_test.go`; `BK` = `backlog_test.go`; `WS` = `workspace_test.go`; `KY` = `keys_test.go`; `TS` = `toast_test.go`; `SC` = `sections_test.go`; `SFL` = `sections_flows_test.go` (todos bajo `tests/unit/tui/` salvo `SF` y `DFS`). El número tras `:` es la línea de la función `Test*`. Código: `story_detail.go` (`openStoryDetail`:69, `detailStory`:80, `storyActionsAvailable`:91, `updateStoryDetail`:103, `openStoryFlow`:136, `backToStoryDetail`:158, `storyDetailBindings`:168, `viewStoryDetail`:182), `backlog_screens.go`:198-208, `sections.go` (`firstStep`:36, `applySectionResult`:106, `updateSection`:127), `workspace.go` (`currentTab`:60, `inStoryFlow`:84, `tabRoot`:96), `model.go` (`crumbParts`:733, `storyFlowCrumb`:821, `quitsOnQ`:327), `keys.go`:91/255, `project_picker.go`:156, `planning_poker.go`:177, `effort_form.go`:219, `defect_creation.go`:37/50/150/253, `defect_submission.go`:67.

### 1. ADDED Detalle de la historia con sus acciones (21 escenarios)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Enter abre el detalle con los campos | `SD:49` | COMPLIANT |
| 2 | Valores faltantes como «—» | `SD:78` | COMPLIANT |
| 3 | No muestra tareas, defectos ni esfuerzo | `SD:97` | COMPLIANT |
| 4 | Enter con la lista vacía | `SD:125` | COMPLIANT |
| 5 | Enter con filtro abre la resaltada | `SD:140` (2 casos) | COMPLIANT |
| 6 | Enter mientras se escribe el filtro | `SD:172` | COMPLIANT |
| 7 | Acciones como filas en orden | `SD:385`, `SD:400` | COMPLIANT |
| 8 | Enter ejecuta la acción resaltada | `SD:436` (Estimar y Esfuerzo), `DFS:165` (Defecto) | COMPLIANT |
| 9 | Cada atajo inicia su acción | `SD:712` (`p`,`t`), `DFS:133` (`d`) | COMPLIANT |
| 10 | Estimar y volver con Esc | `SD:480` | COMPLIANT |
| 11 | Story Points releídos tras estimar | `PP:261` | COMPLIANT |
| 12 | Mientras se recarga indica la carga | `PP:278`, `SF:25/62/139/178` | **PARTIAL** (W1) |
| 13 | La historia ya no está | `PP:310`, `SF:25/62` | **PARTIAL** (W1) |
| 14 | Falla la recarga | `PP:292`, `SF:62` | **PARTIAL** (W1) |
| 15 | Esc conserva filtro y cursor | `SD:218`, `SF:25` | COMPLIANT |
| 16 | No ofrece modificar | `SD:200` | COMPLIANT |
| 17 | Error de API en el paso lanzado | `SD:498` (500 en `/members` y `/tasks`, sin fila sintética, Esc al detalle) | COMPLIANT |
| 18 | Resalta la pestaña Backlog | `SD:308` (miga), `SF:280`/`SF:319` (`currentTab`) | COMPLIANT (S1; ver S-b) |
| 19 | Tab y dígitos no cambian de pestaña | `SD:257` | COMPLIANT |
| 20 | Pie y ayuda | `SD:669`, `SD:682` | COMPLIANT |
| 21 | Cambiar de proyecto descarta el detalle | `SD:601`, `SF:240` | COMPLIANT |

### 2. ADDED Reportar un defecto desde la historia (15 escenarios)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Formulario con historia precargada | `DFS:133` | COMPLIANT |
| 2 | Enter en el último campo abre Sprints | `DFS:150` | COMPLIANT |
| 3 | Esc en Sprints vuelve al formulario | `DFS:234` | COMPLIANT |
| 4 | Registrar envía la historia precargada | `DFS:303` | COMPLIANT |
| 5 | Éxito vuelve al detalle con aviso | `DFS:279` | COMPLIANT |
| 6 | Cancelar la confirmación | `DFS:262` | COMPLIANT |
| 7 | Esc en la confirmación | `DFS:325` | COMPLIANT |
| 8 | Esc en el formulario descarta el borrador | `DFS:201` | COMPLIANT |
| 9 | La API rechaza el registro | `DFS:340` (422 y conexión) | COMPLIANT |
| 10 | Doble envío | `DFS:381` | COMPLIANT |
| 11 | Falla la consulta de Sprints | `DFS:455` | COMPLIANT |
| 12 | Respuesta tardía de Sprints | `DFS:399` (subprueba «Sprints tras Esc») | COMPLIANT |
| 13 | Respuesta de un proyecto anterior | `DFS:399` (subprueba proyecto anterior; solo la respuesta del `POST`) | COMPLIANT |
| 14 | Pestaña según el origen del alta | `SF:280` (3 filas detalle → Backlog, 4 filas Defectos → Defectos) | COMPLIANT |
| 15 | El alta desde Defectos conserva su flujo | `DFS:563`, `DFS:506`, `defect_create_test.go`, `defect_submission_test.go`, `defect_toast_test.go` | COMPLIANT |

### 3. MODIFIED Navegación de solo lectura (6)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Navegar no muta | `BK:963`, `dashboard_test.go` (`TestDashboardOnlyEmitsGETRequests`), `defects_test.go:62` | COMPLIANT |
| 2 | Consultar el backlog no muta | `BK:963`, `BK:340` | COMPLIANT |
| 3 | Pestañas y acciones de la historia no mutan | `SFL:121` (detalle + 3 acciones, y exige que se hayan pedido members/tasks/sprints-open); las pestañas, por las pruebas del punto 1 | COMPLIANT (compuesta) |
| 4 | Abrir y completar el formulario de esfuerzo no muta | `ET:538` | COMPLIANT |
| 5 | Solo la confirmación emite el `POST` | `ET:565` | COMPLIANT |
| 6 | Planning Poker solo escribe al confirmar | `PP:17`, `PP:235` | COMPLIANT |

### 4. MODIFIED Operar el backlog mediante la TUI (7)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | La opción Backlog da acceso | `WS:187` | COMPLIANT |
| 2 | Sin proyecto activo | `WS:98`, `BK:255` | COMPLIANT |
| 3 | Crear, abrir y modificar | `SD:49`, `BK:1110`, `BK:427` | COMPLIANT |
| 4 | `e` modifica y Esc vuelve | `BK:710` | COMPLIANT |
| 5 | `P` y `E` ya no inician flujos | `BK:1191`, `BK:1241` | COMPLIANT |
| 6 | Letras no reservadas inician la búsqueda | `BK:1223`, `BK:1130` | COMPLIANT |
| 7 | Cambiar de proyecto desde el navegador | `BK:340` | COMPLIANT |

### 5. MODIFIED Modificar historias desde la lista (7)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1-4 | Precarga, seis claves, `null`, conserva el resto | `BK:520`, `BK:560`, `BK:599`, `BK:616` | COMPLIANT (4) |
| 5 | Lista vacía | `BK:1141`, `BK:691` | COMPLIANT |
| 6 | Filtro aplicado | `BK:1156` | COMPLIANT |
| 7 | `e` dentro del filtro es texto | `BK:1175` | COMPLIANT |

### 6. MODIFIED Listas navegables con teclas uniformes (6)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Buscar en cualquier colección | `BK:384`, `SC:149`, `internal/tui/selector_test.go` | COMPLIANT |
| 2 | Acción primaria con Enter (Backlog = detalle) | `SD:49` | COMPLIANT |
| 3 | `e` modifica en Backlog y Proyectos | `BK:1110`, `project_form_test.go:206` | COMPLIANT |
| 4 | Dígitos 1 a 4 reservados; 5 y 6 no | `WS:187/204/218` | COMPLIANT |
| 5 | Esc limpia el filtro antes de volver | `SC:149`, `SD:524` | COMPLIANT |
| 6 | Ninguna lista pide identificadores | ninguna prueba lo afirma (escenario preexistente, no tocado por este cambio) | **UNTESTED** (S-a) |

### 7. MODIFIED Estimar historias con Planning Poker (7)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1-2 | Votos privados; reiniciar ronda | `SFL:83` | COMPLIANT (2) |
| 3 | Estimar empieza en los participantes | `SD:436`, `SD:712` | COMPLIANT |
| 4 | Guardar el estimado | `PP:17`, `PP:235` | COMPLIANT |
| 5 | Fallo al guardar | `PP:96` (404, 409, 422) | COMPLIANT |
| 6 | Esc desde los participantes (y filtro) | `SD:480`, `SD:524` | COMPLIANT |
| 7 | Respuesta tardía de integrantes tras volver al detalle | sin prueba de Poker; solo las análogas de Esfuerzo `ET:471`, `ET:489` ejercen la misma guarda | **PARTIAL** (W2) |

### 8. MODIFIED Selectores navegables (7)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Buscar en un selector | `SC:149`, `SD:524` | COMPLIANT |
| 2 | Elegir una fila | `EF:57`, `ET:176` | COMPLIANT |
| 3 | Refrescar conserva el filtro | `SC:99`, `SC:149` | COMPLIANT |
| 4 | Selector vacío | `SC:137`, `PP:73` | COMPLIANT |
| 5 | El de tareas de Esfuerzo no queda vacío | `ET:176` | COMPLIANT |
| 6 | Los tipos de selector son navegables | `SFL:121`, `DFS:563` | COMPLIANT (compuesta) |
| 7 | Estimar y Esfuerzo sin selector de historias | `SD:436` (verifica que «Segunda historia» no aparece) | COMPLIANT |

### 9. MODIFIED Elegir historia, tarea e integrante para esfuerzo (8)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Flujo completo hasta el formulario | `ET:508`, `EF:233`, `ET:538` | COMPLIANT |
| 2 | La fila «Toda la historia» encabeza | `ET:159` | COMPLIANT |
| 3 | Historia sin tareas | `ET:176` | COMPLIANT |
| 4 | Refrescar conserva la fila | `ET:200` | COMPLIANT |
| 5 | Falla de tareas: sin fila sintética, Esc al detalle | `ET:218`, `SD:498` | COMPLIANT |
| 6 | Selección simple de integrante | `ET:240`, `EF:57` | COMPLIANT |
| 7 | Proyecto sin integrantes | `ET:272` | COMPLIANT |
| 8 | Falla de integrantes | `ET:299` | COMPLIANT |

### 10. MODIFIED Enviar el registro de esfuerzo (12)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Sobre una tarea | `EF:335` | COMPLIANT |
| 2 | Historia completa omite `task_id` | `EF:350` | COMPLIANT |
| 3 | Sin recortar | `EF:364` | COMPLIANT |
| 4 | Doble Enter | `EF:536` | COMPLIANT |
| 5 | Esc ignorado mientras guarda | `EF:556` | COMPLIANT |
| 6-10 | HTTP 422, 400, 404, 500, otro código | `EF:620` | COMPLIANT (5) |
| 11 | Error de conexión | `EF:663` | COMPLIANT |
| 12 | Corregir y reenviar → detalle con aviso | `EF:684`, `ER:96` | COMPLIANT |

### 11. MODIFIED Retroceder en el flujo de esfuerzo (8)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Esc recorre formulario, integrantes, tareas, detalle | `ET:391` | COMPLIANT |
| 2 | El filtro del selector de tareas se conserva | `ET:418`, `ET:323` | COMPLIANT |
| 3 | Segundo registro con formulario inicial | `ER:119` | COMPLIANT |
| 4 | Tardía de integrantes tras retroceder | `ET:471` (compara la vista completa) | COMPLIANT |
| 5 | Tardía de tareas tras volver al detalle | `ET:489` | COMPLIANT (con matiz, W2) |
| 6 | Tardía de tareas de otra historia | `SD:552`, `SF:214`, `SF:227` | COMPLIANT |
| 7 | Guardado sin guardado en curso | `EF:718` | COMPLIANT |
| 8 | Cambiar de proyecto descarta flujo, detalle y destino | `ER:144`, `ER:203`, `SD:601`, `SF:240` | COMPLIANT |

### 12. MODIFIED Espacio de trabajo con pestañas (11)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Elegir un proyecto abre el Dashboard con 4 pestañas | `WS:143`, `WS:170` | COMPLIANT |
| 2 | Tab y números | `WS:187`, `WS:204`, `WS:253` | COMPLIANT |
| 3 | Tab desde la última | `WS:230` | COMPLIANT |
| 4 | Shift+Tab desde la primera | `WS:244` | COMPLIANT |
| 5 | Los dígitos 5 y 6 | `WS:218` | COMPLIANT |
| 6 | Números dentro de un filtro | `WS:267` | COMPLIANT |
| 7 | Esc en la raíz | `WS:283` | COMPLIANT |
| 8 | Esc dentro de un flujo | `WS:292` | COMPLIANT |
| 9 | El detalle y los flujos resaltan Backlog | `SD:624` (6 pantallas), `SF:280` | COMPLIANT (S1; ver S-b) |
| 10 | Defectos resalta solo su flujo | `SF:280` (`defects`, `defect-detail`, alta con `returnDefault`) | COMPLIANT |
| 11 | Cambiar de proyecto | `WS:305` | COMPLIANT |

### 13. MODIFIED Avisos efímeros (8)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Crear historia vuelve al backlog | `TS:116` | COMPLIANT |
| 2 | Un rechazo conserva el formulario | `TS:90`, `TS:132` | COMPLIANT |
| 3 | Registrar esfuerzo vuelve al detalle | `ER:31`, `ER:96` | COMPLIANT |
| 4 | Defecto desde la historia vuelve al detalle | `DFS:279` (incluye «la siguiente tecla limpia el aviso») | COMPLIANT |
| 5 | Defecto desde Defectos vuelve a la lista | `DFS:563`, `defect_toast_test.go:11` | COMPLIANT |
| 6 | Estimar vuelve al detalle sin aviso | `PP:235` (sin `✓`) | COMPLIANT |
| 7 | Cambiar estado de un defecto vuelve a su detalle | `defect_toast_test.go:44` | COMPLIANT |
| 8 | El envío en curso ignora repeticiones | `TS:105`, `TS:164` («Guardando…») | COMPLIANT |

### 14. MODIFIED Pie de teclas acotado y ayuda agrupada (6)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Pie de la lista de Sprints | `KY:44` | COMPLIANT |
| 2 | La ayuda lista todos los atajos | `KY:59` | COMPLIANT |
| 3 | Pantalla con pocos atajos | `KY:91` | COMPLIANT |
| 4 | Filtro o campo de texto activo | `KY:108` | COMPLIANT |
| 5 | Teclas de pestaña en la ayuda (`1–4`) | `KY:78` | COMPLIANT |
| 6 | La ayuda del Backlog refleja sus teclas | `BK:1241`, `SD:340` | COMPLIANT |

### 15. MODIFIED Salir limpiamente (6)

| # | Escenario | Prueba(s) | Estado |
|---|-----------|-----------|--------|
| 1 | Salida limpia | `model_test.go:199` | COMPLIANT |
| 2 | `q` desde el navegador | `WS:321`, `model_test.go:258` | COMPLIANT |
| 3 | `q` desde una lista | `BK:1001` | COMPLIANT |
| 4 | `q` desde el detalle | `SD:282` | COMPLIANT |
| 5 | `q` en un campo de texto no sale | `BK:1001`, `model_test.go:213`, `EF:179` | COMPLIANT |
| 6 | `Ctrl+C` desde cualquier pantalla | `BK:1030`, `model_test.go:199/234` (el detalle no está en la lista de `BK:1030`, ver S-g) | COMPLIANT |

### 16. REMOVED Atajos contextuales de estimación y esfuerzo

`P`/`E` sin efecto: `BK:1191`; pestañas retiradas: `WS:170`; `contextual_shortcuts_test.go` e `internal/tui/sections_test.go` eliminados; símbolos retirados sin referencias (ver «Referencias sueltas»). **COMPLIANT**.

### Totales de la matriz

136 ítems (135 escenarios + el requisito REMOVED): **COMPLIANT 131 · PARTIAL 4 · UNTESTED 1 · MISSING 0**.

## Referencias sueltas

- Búsqueda de los símbolos retirados (`sectionDefects`, `sectionStepPrimary`, `sectionStepSprints`, `preselected`, `openSectionForStory`, `tabEstimation`, `tabEffort` y las llamadas a `openSection`) en `internal`, `tests`, `cmd` y `README.md` → **sin coincidencias** (exit 1).
- `README.md`: búsqueda de `Estimaci`, `Esfuerzo`, `P/E`, `1–6`, `1-6` y `seis pesta` → sin coincidencias. Describe 4 pestañas, `Enter` → detalle, `e`, `p`/`t`/`d` y vuelta por niveles.
- `internal/tui` (no prueba): «Estimación» y «Esfuerzo» solo como títulos de flujo (`sections.go:25,27`) y en el aviso «Esfuerzo registrado» (`effort_form.go:252`); no hay nombres de pestaña. No quedan cadenas `1–6` ni `P`/`E` en la ayuda. En pruebas, `keys_test.go:86` usa `1–6` como aserción negativa.
- Delta de spec: 14 coincidencias del patrón de 6.2, todas históricas (`Previously`/`Reason`/`Migration`: 269, 397, 487, 544, 620, 673, 737, 787, 858, 859) o redacción negativa vigente (268, 295, 297, 819). Sin referencias sueltas.
- Voseo: ninguna cadena nueva lo usa; sobrevive el preexistente (`defect_creation.go:260` «Usá»), documentado en UT 3 (desviación 4). Ver S-f.

## Coherencia con el diseño

| Decisión | Real | Desviación | ¿Documentada en apply-progress? |
|----------|------|------------|---------------------------------|
| D1 `returnTo` tipado, 2 valores, asignación explícita | `story_detail.go:14-19`; asignado en `beginDefectCreation`, `openStoryFlow`, `beginDefectCreationForStory`; reinicio en `openTab` (`workspace.go:44`) y `selectProject` | ninguna | n/a |
| D2 tabla `storyActions` + cursor propio | `story_detail.go:35-39`, `updateStoryDetail` | ninguna | n/a |
| D3 atajos `p`/`t`/`d`; pie ≤ 5 | `↑/↓`, `Enter elegir acción`, `Esc backlog`, `? más` (`SD:669`); `p/t/d` solo en la ayuda | etiquetas de ayuda en minúscula (`estimar`, `registrar esfuerzo`, `reportar defecto`) frente a `esfuerzo`/`defecto` del diseño; irrelevante | no (trivial) |
| D4 `sectionState`/`sectionKind` recortados, `sectionStepNone` | hecho; `sectionStepNone` solo lo usan las pruebas internas | ninguna funcional | sí (UT 5, desv. 5) |
| D5 relee por ID; recarga in situ; guarda acepta `story-detail` | `detailStory`, `reloadBacklogInPlace` (delega en `startBacklogLoad`), `backlog_screens.go:310` | `reloadBacklogInPlace` delega | sí (UT 2, desv. 4) |
| D6 cursor filtrado; Esc sin recargar | `openStoryDetail` recibe la lista visible; `Esc` solo asigna `Screen` | ninguna | n/a |
| D7 alta desde la historia reutiliza el flujo | `beginDefectCreationForStory`; `cancelDefectCreation` unifica `Esc` y `n` | helper `cancelDefectCreation` no figura en el diseño | sí (UT 3, REFACTOR) |
| D8 pestaña y miga por origen | `inStoryFlow()` con lista explícita de las 3 pantallas `defect-*` (el diseño usaba el prefijo `defect-`) | lista explícita en lugar de prefijo; `defect-detail` y selectores de estado no heredan `returnTo` | sí (UT 3, desv. 2) |
| D9 `SectionResultMsg.StoryID` | `sections.go:107-110` | ninguna | n/a |
| D10 limpieza al cambiar de proyecto | `project_picker.go:156-158` y reinicio de `effort`/`pokerVotes` en `openStoryFlow:141-142` | ninguna | n/a |
| 4 pestañas | `workspace.go:17-25` | ninguna | n/a |
| Switches paralelos | `Update`, `viewBody`, `crumbParts`, `screenBindings`, `quitsOnQ`, `currentTab` actualizados; `entersText`, `filtering` y `tabRoot` no se tocan, como pide el diseño | ninguna | n/a |
| `backlogHint` | omite «r refresca» (ancho de 80 columnas) | sí, respecto del diseño | sí (UT 1, desv. 2) |
| Binding de búsqueda del Backlog pasa a la ayuda | `keys.go:109-113` | sí, respecto de las tareas | sí (UT 1, desv. 3) |
| Esqueleto `beginDefectCreationForStory` en el RED de 3.1 | no se agregó | sí, respecto de las tareas | sí (UT 3, desv. 1) |
| Orden de 2.4/2.6 (bindings) | movidos a 2.6 | sí, respecto de las tareas | sí (UT 2, desv. 1) |

No hay desviaciones sin documentar de peso. Planning Poker conserva su doble envío en la confirmación (`applyPokerSave` sin guarda, `planning_poker.go:177`); el diff de ese archivo solo toca el destino del éxito.

## Alcance

- **Guarda del doble envío de Planning Poker: NO agregada** (el único hunk de `planning_poker.go` es el destino del éxito).
- **Sin requests nuevas**: el diff `main...HEAD` de `internal/tui/api.go` y de los archivos `*client*.go` está vacío; el diff de producción no agrega rutas ni `http.NewRequest`. Las requests del cambio son las ya existentes (`GET /stories`, `/members`, `/stories/{sid}/tasks`, `/sprints/open`; `PUT story-points`; `POST effort-records`; `POST defects`). Las pruebas lo confirman (`SD:97`, `ET:565`, `DFS:279`).
- **Sin consultas de registros de esfuerzo**: `ET:565` falla ante cualquier `GET` a `effort-records`.
- **Mensajes de commit**: los 6 llevan prefijo `type(scope)` en inglés y el resto en español; `git log main..HEAD --format=%B` filtrado por `co-authored|claude|anthropic|generated` → sin coincidencias.

## Tamaño

`git diff --shortstat main...HEAD`: **34 archivos, 5.181 inserciones, 749 eliminaciones (5.930 líneas)**. Excluyendo `openspec/` (artefactos SDD, 2.355 líneas, todas adiciones): **3.575 líneas (2.826 + 749)**: producción 616 (369 + 247), pruebas 2.914 (2.428 + 486), README 45. Frente a la estimación de ~1.770 (509 / 1.230 / 30): **~2,0 veces el total, ~2,4 veces en pruebas, ~1,2 veces en producción**. `size:exception` aplica (aceptada por el usuario).

| Commit | Real (sin `openspec/`) | Estimado | Documentado |
|--------|------------------------|----------|-------------|
| `b3c10ca` | 730 (712 + 18) | ~380 | no |
| `84850d9` | 1.085 (1.070 + 15) | ~335 | no |
| `a3d9f52` | 717 (701 + 16) | ~230 | sí (UT 3, desv. 6) |
| `7e5b074` | 525 (245 + 280) | ~300 | no |
| `cd7979c` | 839 (252 + 587) | ~420 | no |
| `e0cf312` | 45 (+ 2.355 de `openspec/`) | ~30 | no |

## TDD estricto

`apply-progress.md` no usa la tabla literal «TDD Cycle Evidence», pero cada UT trae «Ciclo TDD observado» (RED/GREEN con comando y resultado), triangulación, línea base y resultado final; se considera evidencia equivalente (S-c).

| Verificación | Resultado | Detalle |
|--------------|-----------|---------|
| Evidencia TDD presente | OK | UT 1 a 6, una sección cada una |
| Archivos de prueba existen (RED) | OK | los archivos citados existen; `contextual_shortcuts_test.go` e `internal/tui/sections_test.go` eliminados como se declara |
| GREEN confirmado | OK | 361/361 PASS ahora; conteos intermedios 297/320/350/366/363/361 corroborados con `git grep` |
| Triangulación | OK | tablas de 2 a 5 filas, `returnStoryDetail` frente a `returnDefault`, 422/conexión; floja solo en los escenarios 12/13/14 del requisito 1 (W1) |
| Red de seguridad | OK | línea base explícita en UT 1 (297), UT 4 (366), UT 5 (363); UT 2 y 3 parten de la suma anterior |
| Pruebas de caracterización | Declaradas y coherentes | UT 1: lista vacía y `e` en el detalle; UT 2: estados cargando/error/ausente, `StoryID` coincidente, 5 filas `returnDefault`, `ChangingProject`; UT 3: Esc de la confirmación, rechazo, doble envío, tardías, `SprintQueryFailure`, 4 filas `returnDefault`, alta desde Defectos; UT 5: 6 pruebas. Por lectura del código, las nacidas en verde lo son por comportamiento ya existente |
| Pruebas de migración (UT 4) | Sin RED propio, como pide la regla | su validez se apoya en la tabla de mutaciones (no re-ejecutable) |

**Afirmaciones que no se pudieron corroborar** (verificación de solo lectura, sin editar fuentes ni cambiar de rama): las salidas de los RED históricos; la tabla de mutaciones de UT 3 y UT 4 (p. ej. «71 pruebas fallan»); el RED retroactivo por `git stash` de UT 1 (1.5), declarado por el propio apply como no previo. Por lectura sí se corroboró que la mutación «quitar `returnTo = returnDefault`» rompe `DFS:506` y que la de `Esc` del primer paso rompe `SD:480`, `SD:524` y `ET:391`.

**Auditoría de calidad de aserciones** (`story_detail_test.go`, `story_flow_test.go`, `defect_from_story_test.go` y las pruebas nuevas de `planning_poker_test.go`): sin tautologías, sin bucles fantasma (los `for` recorren tablas o mapas fijos no vacíos; `SD:107` recorre una colección que debe estar vacía y se acompaña de `cmd != nil`), sin pruebas de solo-render, sin dobles de comportamiento (`httptest` real). Hallazgos: ver W1, W2, W4 y S-b.

**Distribución por capa**: unitarias con `Model.Update` y `httptest` (equivalentes a integración ligera), 0 E2E, 0 `teatest`. Herramientas de cobertura disponibles.

**Cobertura de funciones tocadas** (`-coverpkg=./internal/tui`, ambos paquetes): `story_detail.go` 87,5–100 % (`openStoryFlow` 87,5 %), `workspace.go` 83,3–100 % (`title` 83,3 %), `defect_creation.go` `updateDefectCreateForm` 89,3 % y `updateDefectSprintSelect` 100 %, `defect_submission.go` 90–100 %, `sections.go` `updateSection` 89,5 % y `applySectionResult` 81,8 %, `model.go` `crumbParts` 97,8 %, `effort_form.go` `applyEffortSave` 100 %, `planning_poker.go` `applyPokerSave` 57,1 % (ramas preexistentes de error de conexión y cuerpo ilegible, sin tocar).

## Hallazgos

### CRITICAL (0)

Ninguno.

### WARNING (5)

- **W1: acciones no verificadas como ausentes en los estados cargando, historia ausente y error de recarga.** Las pruebas verifican que `p`/`t`/`d`/`Enter` se ignoran (`tests/unit/tui/planning_poker_test.go:226-233`), que el pie solo ofrece `Esc` (`internal/tui/story_flow_test.go:178-198`) y el contenido del estado (`story_flow_test.go:62-97`), pero ninguna aserta que las filas «Estimar», «Registrar esfuerzo» y «Reportar defecto» no se dibujan (`viewStoryDetail` retorna antes, `internal/tui/story_detail.go:188-199`). Si alguien dibujara las filas en esos estados, ninguna prueba fallaría. Afecta a los escenarios 12, 13 y 14 del requisito 1 (PARTIAL).
- **W2: respuesta tardía de integrantes de Estimar sin prueba propia, y la prueba de tareas es inerte frente a la guarda de pantalla.** El escenario de `spec.md:433` (Estimar) solo tiene pruebas análogas de Esfuerzo (`tests/unit/tui/effort_test.go:471`, `:489`). Además `effort_test.go:489` no falla si se elimina `m.Screen != "section"` de `applySectionResult` (`internal/tui/sections.go:107`): la respuesta poblaría un selector oculto y la vista del detalle no cambia (límite ya reconocido en `apply-progress.md`, UT 4). No hay daño observable, pero la guarda no tiene una prueba que la proteja.
- **W3: desvío de tamaño no documentado.** El cambio real es ~2,0 veces la estimación (3.575 frente a ~1.770 líneas sin `openspec/`); solo UT 3 declara su exceso (`apply-progress.md:254`). UT 1 (730 frente a ~380), UT 2 (1.085 frente a ~335), UT 4 (525 frente a ~300) y UT 5 (839 frente a ~420) no lo mencionan. `size:exception` ya cubre el PR único, pero no se avisó de que la cifra aceptada (~1.770) se duplicó.
- **W4: evidencia TDD parcialmente no corroborable o declarada débil.** (a) RED y mutaciones históricas no reproducibles en esta verificación; (b) UT 3 reconoce que `SprintQueryFailure…`, las filas `returnDefault` y las cinco salidas del alta desde Defectos **no se mutaron**, por lo que no se sabe si pueden fallar; (c) `TestChangingProjectDiscardsDetailAndReturnTarget` (`tests/unit/tui/story_detail_test.go:601`) nace en verde y no distingue el reinicio de `selectProject`, porque abrir un detalle o un flujo sobrescribe el estado de todos modos; lo protege `internal/tui/story_flow_test.go:240`; (d) `DefectFormHelpDependsOnWhereTheStoryComesFrom` (`defect_from_story_test.go:480`) y los bindings del Backlog tienen RED solo retroactivo.
- **W5: verificación manual no ejecutada.** La tarea 7.4 (recorrido con la API real) no se ejecutó: no hay daemon de Docker ni PostgreSQL. Todas las conclusiones de comportamiento provienen de `Model.Update` con `httptest`, no de la TUI real. Pasos para el usuario más abajo.

### SUGGESTION (7)

- **S-a**: el escenario «Ninguna lista pide identificadores a mano» (requisito 6) no tiene prueba que falle si una lista lo pidiera; preexistente.
- **S-b**: el resaltado visual de la pestaña no se afirma en la vista (los estilos ANSI se eliminan); `story_detail_test.go:315-316` y `:660` comprueban «2 Backlog», texto que siempre está en la barra. La regla S1 se prueba por `currentTab()` (`story_flow_test.go:280`), que es la única fuente del resaltado (`workspace.go:150-159`); basta, pero la aserción de la barra es decorativa.
- **S-c**: añadir en `apply-progress.md` la tabla «TDD Cycle Evidence» con columnas Safety Net y Triangulate por tarea facilitaría la verificación; hoy es narrativa por UT.
- **S-d**: `sectionStepNone` no se usa en producción (solo en pruebas internas); documentado, sin acción.
- **S-e**: tres commits tienen cuerpo de menos de 5 líneas (`cd7979c` 3, `7e5b074` 4, `e0cf312` 4) frente a la regla «5 a 8»; ninguno incumple el máximo ni lleva atribución.
- **S-f**: coexisten voseo (`internal/tui/defect_creation.go:260` «Usá», preexistente) y neutro (`:258` «Use») en la misma pantalla según el origen del alta; documentado por el apply. Unificar en una limpieza posterior.
- **S-g**: `TestCtrlCQuitsFromEveryBacklogScreen` (`tests/unit/tui/backlog_test.go:1030`) no incluye `story-detail`; `Ctrl+C` es global y `model_test.go:234` lo cubre en otras pantallas.

### Punto abierto para el usuario

**S1 (sin confirmar):** que la barra de pestañas resalte **Backlog** en el detalle y en los flujos que parten de él (Estimar, Registrar esfuerzo y el alta de defecto desde la historia), y Defectos solo en su propio flujo. Está implementado en un único lugar (`currentTab` e `inStoryFlow`, `internal/tui/workspace.go:60-95`) y se revierte con pocas líneas y las filas de `internal/tui/story_flow_test.go:280`. Hasta que el usuario lo confirme, el archivo y el delta de spec lo registran como supuesto.

## Recorrido manual pendiente (tarea 7.4: NO EJECUTADO)

Requisitos: Docker Desktop en marcha, PostgreSQL migrado, `DATABASE_URL` para `go run ./cmd/api` y `API_URL` para `go run ./cmd/tui`. Datos: 2 proyectos; el primero con 2 historias (una completa con story points, horas y criterios; otra con todo nulo), una con tareas, 2 integrantes y un Sprint abierto.

1. Barra: cuatro pestañas; `1`–`4` las abren; `5` y `6` no hacen nada (Espacio de trabajo 1 y 5; Listas 4).
2. Backlog: `Enter` abre el detalle con los siete campos y «—» en los nulos; `e` abre la modificación; `P` y `E` inician el filtro; `Esc` del detalle vuelve con filtro y cursor (Detalle 1, 2, 15, 16; Backlog 5).
3. En el detalle: `Tab`, `Shift+Tab` y los dígitos no cambian de pestaña; pie con 5 atajos como máximo; `?` lista `p`/`t`/`d`; `q` sale (Detalle 19 y 20; Salir 4).
4. `p` (y luego `Enter` sobre la fila): `Esc` desde participantes vuelve al detalle; completar la ronda y guardar muestra los Story Points nuevos tras la recarga, sin aviso (Detalle 10 y 11; Estimar 4; Avisos 6).
5. `t`: tareas con «Toda la historia» primera, integrante y formulario; `Esc` tres veces hasta el detalle; registrar con 201 vuelve al detalle con el aviso `✓`, que desaparece con la siguiente tecla (Esfuerzo 1; Retroceder 1; Avisos 3).
6. `d`: «Historia relacionada» precargada; `Enter` en el último campo abre Sprints; `Esc` vuelve al formulario con el borrador; en la confirmación `n` vuelve al detalle; `s` muestra el aviso en el detalle, sin lista de defectos (Defecto 1 a 8).
7. Durante los pasos 4 a 6 la pestaña resaltada es Backlog (S1; Espacio de trabajo 9).
8. Pestaña Defectos: `n`, selector de historias, Sprint y confirmación; `Esc`, `n` y el éxito vuelven a la lista (Defecto 15; Avisos 5).
9. Cambiar de proyecto con `Esc`: abrir el segundo proyecto y su Backlog muestra la lista, no el detalle; iniciar y cancelar una acción vuelve al detalle de su historia (Detalle 21; Retroceder 8).

## Resumen y siguiente paso

Verificado: compilación, `vet`, formato, 361/361 pruebas de la TUI y 31/31 paquetes sin Docker; 131 de 136 ítems de la matriz cumplen, 4 son parciales y 1 no tiene prueba; sin requests nuevas ni guarda de Planning Poker; sin referencias sueltas. Desconocido o no ejecutado: recorrido manual, pruebas de integración y S1. Siguiente paso recomendado: `sdd-archive` (las observaciones son diagnósticas; si se desea cubrir W1 y W2 bastan dos pruebas pequeñas con ciclo RED, GREEN y un commit `test(tui): …` antes de archivar).
