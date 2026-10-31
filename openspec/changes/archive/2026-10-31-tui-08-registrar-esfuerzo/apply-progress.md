# Progreso de apply: Registrar esfuerzo desde la TUI (TUI-08)

- **Cambio**: `tui-08-registrar-esfuerzo` (issue #101)
- **Modo**: TDD estricto (`go test ./...`), entrega `single-pr` con `size:exception` aceptado
- **Rama**: `feat/tui-08-registrar-esfuerzo`
- **Skills aplicadas**: `work-unit-commits`, `go-testing`
- **Lote 1**: Fase 1 (UT 1) y Fase 2 (UT 2).
- **Lote 2**: Fase 3 (UT 3), partida en dos commits (selectores / formulario).
- **Lote 3**: Fase 4 (UT 4), un commit.
- **Lote 4**: Fase 5 (UT 5), un commit.
- **Lote 5**: Fase 6 (UT 6), un commit solo documental. La Fase 7 (verificación final y archivo) sigue pendiente.

## Estado de tareas

| Fase | Tareas | Estado |
|------|--------|--------|
| 1 (UT 1) | 1.1 a 1.6 | Completa (commit `342b618`) |
| 2 (UT 2) | 2.1 a 2.5 | Completa (commit `77ac04d`) |
| 3 (UT 3) | 3.1 a 3.15 | Completa (commits `6f1a24f` selectores y `d83c4a0` formulario) |
| 4 (UT 4) | 4.1 a 4.11 | Completa (commit `46a6beb`) |
| 5 (UT 5) | 5.1 a 5.12 | Completa (commit `acb64f2`) |
| 6 (UT 6) | 6.1 a 6.3 | Completa (commit documental; el hash figura en el historial de la rama: `docs(tui): actualizar specs y README para TUI-08`) |
| 7 | 7.1 a 7.5 | Pendiente (verificación final, recorrido manual opcional y archivo) |

## Límites del entorno (aplican a todos los lotes)

- `gofmt -l internal tests` lista **todos** los archivos del repositorio (más de 150), incluidos los que este cambio no toca: el árbol de trabajo está en CRLF (`core.autocrlf=true`, índice en LF) y `gofmt` trata CRLF como no formateado. No es un fallo propio ni se corrige (tocaría todo el repositorio). Sustituto verificable usado: `tr -d '\r' < archivo | gofmt -l` para los archivos modificados (sin salida) y `gofmt -l` directo sobre los archivos nuevos escritos en LF (sin salida).
- Git avisa «LF will be replaced by CRLF» al agregar archivos nuevos; es solo el aviso de `autocrlf`.
- Docker está disponible, pero las Fases 1 y 2 no lo necesitan (pruebas de `Model.Update` y `httptest`).

## UT 1: helpers genéricos del campo de fecha

Commit: `342b618` `refactor(tui): extraer helpers genéricos del campo de fecha`

### Línea base (1.1)

Antes de tocar nada: `go test ./internal/tui/... ./tests/unit/tui/...` en verde (`ok internal/tui`, `ok tests/unit/tui`), incluido `project_form_test.go` sin modificar. Rama `feat/tui-08-registrar-esfuerzo` ya creada desde `main`.

### Ciclo TDD

| Tarea | Archivo de prueba | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------------|-----|-------|-------------|----------|
| 1.2 / 1.3 | `internal/tui/date_field_test.go` | Unitaria (paquete `tui`) | Línea base en verde | Observado: `go test ./internal/tui -run 'DateField\|DateRow\|AlignedTextRow'` falla por aserción (33 líneas `--- FAIL`: 4 pruebas y 29 subcasos) con los esqueletos de valores cero | `go test ./internal/tui -run 'DateField\|DateRow\|AlignedTextRow'` → `ok` | 15 casos de teclas de edición + 6 de teclas no consumidas + 4 filas de fecha + 4 filas de texto | n/a (ver 1.5) |
| 1.4 | `tests/unit/tui/project_form_test.go` (sin modificar) | Unitaria (red de seguridad) | Verde antes y después | No aplica: refactor puro; la red de seguridad ya existía y debía seguir en verde | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok`; `git diff --stat -- tests/unit/tui/project_form_test.go` vacío | n/a | n/a |
| 1.5 | (mismas) | | | | | | `projectDateRow` y `projectTextRow` quedan como una línea que delega; sin duplicación residual; `go vet ./internal/tui/...` limpio; comando focalizado de la UT con 17 pruebas en verde |

Detalle del RED honesto:

- Los esqueletos mínimos (`applyKey`, `dateRow`, `alignedTextRow` con valores cero) se agregaron en el mismo paso que las pruebas, sin commit aislado.
- `TestDateFieldApplyKeyReportsUnhandledKeys` usa un campo con valor (`2026/10/5`) y segmento `mes`: el esqueleto devuelve `dateField{}` y segmento 0, así que falla por aserción y no por azar.
- `TestDateRowMatchesProjectDateRow` y `TestAlignedTextRowMatchesProjectTextRow` comparan con la salida de `projectDateRow`/`projectTextRow` **y** con cadenas literales capturadas del código previo al refactor (prueba de aprobación). Cuando `projectDateRow` pasó a delegar, la comparación entre ambos se volvió tautológica, pero las cadenas literales siguen verificando contra el render original.
- Salida de color: bajo `go test` no hay terminal, así que lipgloss no emite secuencias ANSI y las cadenas literales son texto plano; el ancho de la barra enfocada (76) se afirma aparte.

### Evidencia de unidad de trabajo

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado y resultado | `go test ./internal/tui/... ./tests/unit/tui/... -run 'DateField\|DateRow\|AlignedTextRow\|ProjectForm\|ProjectDate\|EditProject\|ProjectResult'` → `ok` en ambos paquetes (17 pruebas de nivel superior en verde) |
| Comando de la unidad completa | `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` ambos |
| Arnés de ejecución | N/A: pruebas de `Model.Update` y de render sin proceso real; la red de seguridad es `project_form_test.go` sin modificar |
| Límite de reversión | Revertir `342b618` (`date_field.go`, `updateProjectDate` en `model.go` y `date_field_test.go`); nada más depende todavía de los helpers |

### Carga de revisión

`git diff --cached --numstat` del commit: `date_field.go` +73/-27, `date_field_test.go` +168/-0, `model.go` +6/-27. Total **301 líneas** autoras (estimación del plan: ~95; la diferencia es el volumen de pruebas por tabla, 168 líneas, que se mantiene porque protege un refactor sobre código ya en uso).

## UT 2: cliente `EffortClient`

Commit: `77ac04d` `feat(tui): agregar cliente de registro de esfuerzo`

### Ciclo TDD

| Tarea | Archivo de prueba | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------------|-----|-------|-------------|----------|
| 2.1 / 2.2 | `tests/unit/tui/effort_api_test.go` | Unitaria con `httptest` | N/A (archivos nuevos) | Observado: `go test ./tests/unit/tui/... -run EffortClient` → `--- FAIL` en las dos pruebas con `requests = 0 ([]), want exactly 1` (esqueleto con `Register` de valores cero) | Mismo comando → `ok` (2 pruebas) | Ver abajo | n/a |
| 2.3 | `tests/unit/tui/effort_api_test.go` | Unitaria con `httptest` | n/a | No aplica (ver nota) | `-run 'EffortClient\|EffortRecord'` → `ok` (6 pruebas de nivel superior) | 4 pruebas nuevas: `TestEffortClientEscapesPathSegments`, `TestEffortClientReturnsHTTPErrorsAsResponses` (422 y 404), `TestEffortClientReturnsTransportErrors`, `TestEffortRecordDecodesFlatCreatedResponse` (`task_id` nulo y con valor) | |
| 2.4 | (mismo) | | | | | | Comentario de `Register` documenta que la API no es idempotente; `gofmt` (archivos nuevos en LF) sin salida; `go vet ./internal/tui/... ./tests/unit/tui/` limpio |

Notas honestas:

- Las cuatro pruebas de triangulación **nacieron en verde**: la implementación de 2.2 ya construye la ruta con `url.PathEscape`, reenvía cualquier estado HTTP como `APIResponse` y no interpreta el cuerpo, así que no hubo RED alcanzable sin escribir una implementación falsa artificial. En vez de inventar evidencia, se hicieron **pruebas de mutación** manuales y se restauró el archivo (verificado con `diff`):
  - Mutante A (sin `url.PathEscape(storyID)`): falla `TestEffortClientEscapesPathSegments`.
  - Mutante B (`task_id` sin `omitempty`): falla `TestEffortClientOmitsTaskIDForWholeStory`.
- `TestEffortClientReturnsTransportErrors` y `TestEffortRecordDecodesFlatCreatedResponse` no se mutaron (el primero depende del comportamiento de `http.Client` y el segundo de las etiquetas JSON de `EffortRecord`, que el esqueleto del RED ya contenía).
- El comentario de `Register` sobre no-idempotencia se escribió ya en el paso GREEN; la tarea 2.4 lo prevé como parte del REFACTOR y no cambió después.
- `ptr` (genérico) es un helper de prueba nuevo en el paquete `tui_test`; sin colisiones (`rg` confirma una sola definición).

### Evidencia de unidad de trabajo

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado y resultado | `go test ./tests/unit/tui/... -run 'EffortClient\|EffortRecord'` → `ok` (6 pruebas de nivel superior y 4 subcasos) |
| Comando de la unidad completa | `go test ./tests/unit/tui/... ./internal/tui/...` → `ok` ambos |
| Arnés de ejecución | N/A: `httptest` con servidor grabador; no hay frontera de proceso real |
| Límite de reversión | Eliminar `internal/tui/effort.go` y `tests/unit/tui/effort_api_test.go`; nada más los importa todavía |

### Carga de revisión

`git diff --cached --numstat` del commit: `effort.go` +44/-0, `effort_api_test.go` +147/-0. Total **191 líneas** autoras (estimación: ~145; las pruebas de triangulación por tabla explican el exceso).

## UT 3: selectores, integrantes y formulario editable (sin envío)

La UT se partió en **dos commits** sin cambiar la estrategia (`single-pr`, `size:exception`), tal como prevé la cabecera de `tasks.md`. Motivo: la UT real midió ~1.050 líneas autoras frente a las ~380 estimadas (unas 850 son pruebas y helpers de prueba), y un único commit de ese tamaño no es revisable.

| Commit | Alcance | Tareas | Líneas autoras (adiciones + eliminaciones) |
|--------|---------|--------|--------------------------------------------|
| `6f1a24f` `feat(tui): elegir tarea o historia completa e integrante en Esfuerzo` | Fixtures, fila sintética, integrantes, snapshot `tasks`, `Esc` de integrantes a tareas, retiro de la prueba obsoleta | 3.1 a 3.5 | 435 (`sections.go` +43/-5, `model.go` +5/-1, `effort_test.go` +338, `sections_flows_test.go` -43) |
| `d83c4a0` `feat(tui): abrir y editar el formulario de esfuerzo sin enviar` | Reloj, `effort-form`, edición, teclas como texto, pie, miga, `Esc` del formulario, retiro del caso Esfuerzo de `sectionNotice` | 3.6 a 3.14 | 622 (`effort_form.go` +117, `effort_form_test.go` internal +24, `model.go` +29/-1, `sections.go` +4/-2, `effort_form_test.go` externo +270, `effort_test.go` +175) |

La suma de producción es ~210 líneas y la de pruebas ~850 (incluye fixtures y la prueba eliminada); la desviación frente a la estimación se debe casi por completo a las pruebas guiadas por tabla de comportamiento visible y a los helpers del servidor de prueba.

### Línea base (seguridad previa)

Antes de tocar `sections.go`/`model.go`: `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` ambos (en verde).

### Ciclo TDD

| Tarea | Archivo de prueba | Capa | RED observado | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|---------------|-------|-------------|----------|
| 3.1 | `tests/unit/tui/effort_test.go` (fixtures) | Soporte de pruebas | n/a (infraestructura; `go vet ./tests/unit/tui/` limpio) | n/a | n/a | n/a |
| 3.2 / 3.3 | `effort_test.go` (`EffortTaskSelector*`) | Unitaria (`Model.Update` + `httptest`) | 4 pruebas fallan por aserción: «rows must be whole story, Diseñar, Probar…», «an empty story should still offer…», «whole-story row should stay first…», «the row should appear once the retry succeeds» | 3 de 4 en verde tras 3.3; `OffersWholeStoryWhenStoryHasNoTasks` quedó rojo a propósito en su último paso («Enter … should load the members») hasta 3.5 | Historia con tareas, historia sin tareas, refresco y fallo `500` + reintento | Ver 3.5 |
| 3.4 / 3.5 | `effort_test.go` (`EffortMember*`, `EffortEscFromMembers*`) | Unitaria | 4 pruebas fallan: «expected a command to execute, got nil» (elegir tarea todavía no cargaba integrantes); falla en el helper de apertura, no en una aserción de texto | 8 pruebas (4 de 3.2 + 4 de 3.4) en verde | Lista con integrantes, vacía, con error `500` (reintentar y volver) y filtro + cursor conservados | Mutantes (ver abajo); `gofmt` aplicado a `effort_test.go` |
| 3.6 / 3.7 | `effort_form_test.go` externo e interno | Unitaria | 5 fallos por aserción: `Screen = "section"` en lugar de `effort-form` (el flujo mostraba el aviso `TUI-08`) y `today() = ""` (esqueletos de `WithClock`/`today` con valor cero) | `TestEffortFormOpensWithUTCDateFromInjectedClock`, `…ShowsWholeStoryLabel`, `…SingleSelectionOpensFormForSecondMember`, `TestToday*` en verde | Historia completa vs tarea; segundo integrante; reloj nulo vs inyectado | n/a |
| 3.8 / 3.9 | `effort_form_test.go` externo | Unitaria | 7 de 7 pruebas fallan por aserción (p. ej. `Esc`/`Tab` sin efecto, `Actividad = ""`, pie con `? ayuda`, miga sin «Esfuerzo › Registrar», `?` abría la ayuda) | `go test … -run EffortForm` y `Footer\|UX\|ProjectForm` en verde | Fecha por segmentos, avance con `Enter`, `Tab`/`Shift+Tab`/flechas circulares, texto con espacios y retroceso por runa, `q Q ? p r` como texto en dos campos | n/a |
| 3.10 / 3.11 | `effort_test.go` (`EffortEsc*`, `EffortTaskFilter*`, `EffortLate*`) | Unitaria | 3 fallan por aserción (`Esc` del formulario no salía de la pantalla) | Las 5 en verde | Tres niveles, filtro conservado, descarte de lo escrito | n/a |
| 3.12 | `effort_test.go` | Unitaria | `[caracterización]` ver abajo | n/a | n/a | n/a |
| 3.13 | (las anteriores) | | | `sectionNotice` sin el caso Esfuerzo; `go test ./tests/unit/tui/... ./internal/tui/...` en verde | | |
| 3.14 | | | | | | Pruebas restantes; `gofmt` (LF) sin salida; `go vet ./...` limpio; sin duplicación nueva |

### Pruebas nacidas en verde `[caracterización]` y su respaldo

- `TestEffortLateMembersResponseAfterEscIsDropped` y `TestEffortLateTasksResponseAfterEscIsDropped`: la guarda de `applySectionResult` ya descartaba esas respuestas (pasos distintos y selector sin `loading` tras volver). Su RED no es alcanzable. Respaldo con mutantes: quitar solo la guarda de paso o solo la de `loading` **no** las mata (cada guarda protege por sí sola, defensa en profundidad); quitar **ambas** las hace fallar. Se restauró el archivo.
- `TestEffortFlowReachesFormWithoutTUI08Notice`: nació en verde porque 3.7 ya abría el formulario. El RED real del retiro del aviso se observó en 3.6: antes de 3.7, `Screen == "section"` y la vista contenía «TUI-08» (comprobado con la salida de `TestEffortFormShowsWholeStoryLabel`). Mutante (la elección del integrante deja de abrir el formulario) la mata.
- `TestEffortNavigationEmitsOnlyGetUntilExplicitSubmit`: guarda de regresión; en esta UT no existe ninguna escritura, así que **no hay mutante distintivo** posible hasta que la UT 4 agregue el `POST`.
- `TestEffortFormAlignsTheValueColumn` (agregada, no figuraba en `tasks.md`): nació en verde porque se escribió ya con el ancho correcto; el mutante `effortLabelWidth = len("Actividad")` la mata (Actividad 15, Horas 14 en lugar de 15/15).
- Mutantes de selectores: sin `m.section.tasks = m.section.selector` fallan 3 pruebas (`EmptyState`, `ShowsAPIError`, `EscFromMembersRestoresTasksSnapshot…`); la vista de Poker para todo `kind` hace fallar 2 (`UsesSingleSelectionView`, `EmptyState…`).

### Evidencia de unidad de trabajo

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado y resultado | `go test ./internal/tui/... ./tests/unit/tui/... -run 'Effort\|Section\|PlanningPoker\|Menu\|Today'` → `ok` en ambos paquetes |
| Comando de la unidad completa | `go test ./internal/tui/... ./tests/unit/...` → `ok` en todos los paquetes (tras cada commit) |
| Arnés de ejecución | N/A: pruebas de `Model.Update` con `httptest`; el recorrido manual opcional queda para 7.3 |
| Límite de reversión | `d83c4a0`: `effort_form.go`, pruebas nuevas, reloj/cableado de `model.go` y el caso Esfuerzo de `sectionNotice` (restaura el aviso para integrantes). `6f1a24f`: fila sintética, paso de integrantes y snapshot en `sections.go` (restaura el aviso al elegir tarea); la prueba obsoleta eliminada reaparece al revertir |

### Intermedio entre los dos commits

Entre `6f1a24f` y `d83c4a0`, elegir un integrante todavía cae en la ruta genérica del aviso `TUI-08` (con el integrante mostrado como «Tarea»). Es un estado transitorio dentro del mismo PR, sin prueba que lo fije; `d83c4a0` lo reemplaza por el formulario. No se agregó código temporal para ocultarlo.

### Carga de revisión

Ver tabla de commits arriba. Total UT 3: **1.057 líneas** (435 + 622 según `git show --numstat`; adiciones más eliminaciones). Ambos commits superan o rozan el presupuesto de 400; cubierto por `size:exception` aceptado.

## UT 4: validación local, envío y errores de la API

Commit: `46a6beb` `feat(tui): registrar esfuerzo con validación local y anti doble envío`

### Línea base (seguridad previa)

Antes de tocar `effort_form.go` y `model.go`: `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` ambos (en verde).

### Ciclo TDD

| Tarea | Archivo de prueba | Capa | RED observado | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|---------------|-------|-------------|----------|
| 4.1 / 4.2 | `effort_form_test.go` (`EffortSubmit*`) | Unitaria (`Model.Update` + `httptest`) | `go test ./tests/unit/tui/... -run EffortSubmit` → 4 de 4 `--- FAIL` por aserción: «Enter on Horas should submit the effort» (Enter en «Horas» no emitía comando) | 4 de 4 en verde tras `submitEffortForm`, `registerEffortCmd`, `effortSaveResult`, `applyEffortSave` mínimo y el marcador `effort-result`. La primera ejecución del GREEN siguió roja por un defecto del **helper** de prueba (ver «Defectos reales») | Tarea con integrante 1, historia completa con integrante 2 (sin `task_id`), descripción sin recortar y fecha 2026-3-7 → `2026-03-07` | n/a |
| 4.3 / 4.4 | `effort_form_test.go` (`EffortFormLocalValidation*`, `EffortFormLocalError*`, `EffortFormConfirmsAfter*`) | Unitaria | 3 pruebas rojas por aserción: 11 de 11 subcasos de validación («an invalid form must not emit a command»), 3 de 3 subcasos de limpieza al editar («setup: expected the zero-hours error») y la corrección con reenvío («zero hours must not emit a command») | Las 3 en verde con `validateEffortInput`, `parseEffortHours`, `err` y su limpieza al editar | 11 casos: día vacío, 2026-02-30, horas vacías, `abc`, `1,5`, `NaN`, `Inf`, `+Inf`, `0`, `-2` y fecha más horas inválidas (solo fecha); limpieza editando horas, actividad y fecha | n/a |
| 4.5 / 4.6 | `effort_form_test.go` (`EffortDouble*`, `EffortEscIgnored*`, `EffortOtherKeys*`) | Unitaria | 3 de 3 rojas por aserción: sin «Registrando…», `Esc` salía a `section` y las teclas cambiaban el formulario | Las 3 en verde: `saving` ignora toda tecla en `updateEffortForm`, indicador `Registrando…` y pie `Ctrl+C salir` | Doble `Enter`, `Esc` con respuesta posterior, 8 teclas distintas más `Ctrl+C` | n/a |
| 4.7 / 4.8 | `effort_form_test.go` (`EffortFormShowsAPI*`, `EffortFormTransport*`, `EffortFormCorrect*`, `EffortSaveResult*`) | Unitaria | 4 pruebas rojas por aserción (5 filas de errores de la API, transporte, corregir y reenviar, y 2 subcasos de `EffortSaveResult`). Con el mapeo de errores pero sin guarda quedaron rojos 3 subcasos de `EffortSaveResult`; con la guarda por `saving` y `seq`, 1 (el `seq` no se conservaba al reabrir) | En verde con el mapeo de errores, la guarda y la conservación de `seq` en `openEffortForm` | 422 con `fields`, 400, 404, 500, 409 y filas `200`/`204` (D5); error de transporte; corregir y reenviar; 4 escenarios de respuesta sin guardado pendiente | n/a |
| 4.9 | `effort_test.go` (`EffortOnlyExplicitSubmitEmitsThePost`) | Unitaria | `[caracterización]`, ver abajo | n/a | n/a | n/a |
| 4.10 | (todas) | | | | | Mensajes en constantes (`effortHoursNotNumberMessage`, `effortHoursNotPositiveMessage`, `effortTransportMessage`) y armado de `EffortInput` dentro de `validateEffortInput`; `go vet ./...` limpio; `gofmt` (LF) sin salida |

Detalle del RED honesto:

- **Sin esqueletos en el RED**: ninguna prueba necesitó un símbolo exportado nuevo, así que todos los RED fallan contra el código real de la UT 3 (por ejemplo, `Enter` en «Horas» sin envío).
- **Orden de 4.2 y 4.8 ajustado para conservar un RED real** (deriva mecánica, sin cambio de comportamiento final): `tasks.md` pide en 4.2 la guarda de `applyEffortSave` y la validación de fecha con `dateField.validate`. Con eso las pruebas de 4.3 y de `EffortSaveResult` habrían nacido en verde. En 4.2 se usó `date.value()` y `strconv.ParseFloat` ignorando el error (mínimo para las pruebas de 4.1); `validate` y los mensajes llegaron con el RED de 4.3. La guarda llegó en 4.8, después de ver rojas las pruebas. El resultado final es el que piden las tareas.
- **Segunda ronda de RED en 4.7/4.8**: se implementó primero el mapeo de errores sin guarda y fallaron por aserción los subcasos «un resultado ya entregado no se aplica otra vez», «una respuesta vieja no toca un guardado nuevo» y «una respuesta de un formulario reabierto». Tras la guarda por `saving` y `seq`, el tercero seguía rojo: `openEffortForm` reiniciaba `seq` a 0 y la respuesta del formulario anterior (seq 1) coincidía con el nuevo guardado (seq 1). Se corrigió conservando `seq` en `openEffortForm`, tal como pedía la tarea 3.7 («reinicia el estado conservando `seq`»), que la UT 3 había omitido por no tener una prueba que lo exigiera.

### Pruebas nacidas en verde `[caracterización]` y su respaldo (mutantes manuales; archivo restaurado y verificado con `diff`)

- `TestEffortFormLeavesRangeDecimalsAndDescriptionToAPI` (3 subcasos): la UT 4 solo rechaza fecha y horas, así que `25`, `1.234` y actividad vacía con fecha 2099 no tenían RED alcanzable. Mutantes: rechazar `hours > 24` mata `hours_above_a_day`; rechazar más de 2 decimales mata `many_decimals`; rechazar actividad vacía mata `empty_activity_and_future_date`.
- Filas `200` y `204` de `TestEffortFormShowsAPIErrorsKeepingData` (D5): nacieron en verde porque `applyEffortSave` ya usa `!= 201`. Mutante (tratar `200` y `204` como éxito) las mata a ambas.
- `TestEffortFormEnterOnEarlierFieldsNeverSendsAValidForm` (agregada, no figuraba en `tasks.md`): nació en verde porque `Enter` solo envía en el último campo. Mutante (`Enter` envía también en campos previos) la mata. Las pruebas de `Enter` de 3.8 usan «Horas» vacío, así que ese mutante les era invisible.
- Subcaso «an answer after success» de `TestEffortSaveResultWithoutPendingSaveIsDiscarded`: nace en verde; la guarda por `saving` ya lo cubre. Mutantes de la guarda: quitar `saving` mata «ya entregado»; quitar `seq` mata «respuesta vieja» y «formulario reabierto»; quitar la conservación de `seq` en `openEffortForm` mata «formulario reabierto».
- `TestEffortNavigationEmitsOnlyGetUntilExplicitSubmit` (UT 3 pedía un mutante distintivo cuando existiera el `POST`): **hallazgo**, la versión del lote 2 usaba `press`, que descarta el comando devuelto, así que **no podía detectar** ninguna escritura. Se reescribió con `stepRunning` (aplica la tecla y ejecuta el comando). Mutante (`Esc` del formulario emite `registerEffortCmd`) → la prueba falla; con la versión anterior habría seguido en verde.
- `TestEffortOnlyExplicitSubmitEmitsThePost` (4.9): guarda de regresión; recorrido completo con una sola confirmación y exactamente un `POST /projects/p1/stories/s1/effort-records`, sin `PUT` ni lecturas de registros. Mutante (`Register` con `PUT` en lugar de `POST`) → falla.
- Mutante que **sobrevive**: quitar la condición `m.Screen != "effort-form"` de la guarda no rompe ninguna prueba (con `saving` y `seq` es inalcanzable en la UT 4), así que la condición se eliminó (ver «Desviaciones de esta UT»).

### Defectos reales

Ninguno en el código de producción previo. Dos hallazgos propios de pruebas: (1) el helper `fillEffortForm` escribió el día en el mes (un segmento completo ya avanza solo y el `→` extra lo saltaba); se detectó en el GREEN de 4.2 por `work_date` vacío y se corrigió en el helper, no en producción; (2) `TestEffortNavigationEmitsOnlyGetUntilExplicitSubmit` no ejecutaba los comandos (ver arriba).

### Evidencia de unidad de trabajo

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado y resultado | `go test ./tests/unit/tui/... -run 'EffortForm\|EffortSubmit\|EffortDouble\|EffortSave\|EffortOnly\|EffortEscIgnored\|EffortOtherKeys' -count=1` → `ok` (27 pruebas de nivel superior en verde, con sus subcasos) |
| Comando de la unidad completa | `go test ./internal/tui/... ./tests/unit/...` → `ok` en todos los paquetes tras el commit |
| Arnés de ejecución | N/A: `Model.Update` con `httptest`, incluida una conexión realmente cerrada (`server.Close()`) para el error de transporte; no hay frontera de proceso real |
| Límite de reversión | `46a6beb`: `submitEffortForm`, `validateEffortInput`, `parseEffortHours`, `applyEffortSave`, `effortSaveResult` en `effort_form.go`; cableado (`effortClient`, `Update`, `viewBody`) en `model.go`; pruebas nuevas. El formulario vuelve a no enviar (estado de la UT 3) |

### Carga de revisión

`git show --numstat 46a6beb`: `effort_form.go` +131/-4, `model.go` +9/-2, `effort_form_test.go` +538, `effort_test.go` +55/-7. Total **746 líneas** autoras (producción ~146, pruebas ~600; estimación del plan: ~260). Rama acumulada: aproximadamente 2.300 líneas autoras. Cubierto por `size:exception` (PR único). Las pruebas por tabla son la mayor parte; no se recortaron.

### Desviaciones de esta UT

- **Guarda de `applyEffortSave` sin la condición `Screen == "effort-form"`** (4.2 la pedía): solo `saving && seq`. El mutante sin esa condición sobrevive y es inalcanzable en la UT 4. Si la UT 5 dejara `saving` verdadero fuera del formulario, se revisa con su propia prueba.
- **Orden de 4.2/4.8** descrito en «Detalle del RED honesto». Sin cambio de comportamiento final.
- **`openEffortForm` conserva `seq`** (la UT 3 lo había omitido; lo pedía 3.7).
- **Pruebas adicionales** respecto de `tasks.md`: `TestEffortFormEnterOnEarlierFieldsNeverSendsAValidForm`, filas `200`/`204` (D5) y subcasos de `EffortSaveResult`. Se reescribió `TestEffortNavigationEmitsOnlyGetUntilExplicitSubmit` (de la UT 3) para ejecutar los comandos.
- **Fixture**: `effortFlowOptions.postFailures` (los primeros N `POST` responden con `postStatus`; los siguientes, `201`) para probar «corregir y reenviar». Nuevas ayudas de prueba: `fillEffortForm`, `effortPosts`, `submitEffort`, `startSaving`, `saveMessage`, `stepRunning`.
- **Texto de la fecha**: «La fecha de trabajo está incompleta.» y «La fecha de trabajo no es una fecha válida.» (D15: las pruebas fijan «fecha» más «incompleta» o «válida»).
- **D2, D3, D4, D5, D9 y D13** aplicados como en la tabla de `tasks.md` (dos mensajes de horas; sin validación defensiva de historia o integrante; limpieza al editar fecha, actividad u horas; solo `201` es éxito; «verificar» en el aviso de transporte; `saving` ignora toda tecla salvo `Ctrl+C`).
- **Estado transitorio de la pantalla de resultado**: `effort-result` es un marcador mínimo («Esfuerzo registrado.»). Su pie es el genérico (`Esc volver · ? ayuda · q salir`) aunque `?` y `q` aún no están enlazados en esa pantalla (solo `Esc`, que cae al menú por la ruta por defecto); la UT 5 lo reemplaza. No se agregó código temporal para ocultarlo.
- **`parseEffortHours` acepta lo que `strconv.ParseFloat` acepta como finito** (por ejemplo `1e2` o `0x1p-2`); la spec exige «número finito con punto decimal mayor que 0». No se endureció por no estar en la spec ni en `tasks.md`; se informa para verificación.

## UT 5: pantalla de resultado, métricas del proyecto, `r`, descarte tardío y `p`

Commit: `acb64f2` `feat(tui): mostrar el registro y las métricas del proyecto tras registrar`

### Línea base (seguridad previa)

Antes de tocar `effort_form.go`, `model.go`, `project_picker.go` y `sections.go`: `go test ./internal/tui/... ./tests/unit/tui/... -count=1` → `ok` ambos (en verde).

### Ciclo TDD

| Tarea | Archivo de prueba | Capa | RED observado | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|---------------|-------|-------------|----------|
| 5.1 / 5.2 | `tests/unit/tui/effort_result_test.go` (`EffortResultShowsRecord*`, `ShowsLoadingThenMetrics`, `DoesNotComputeMetrics`, `NullMetricsShowSinDatos`, `CrumbAndTitle`) | Unitaria (`Model.Update` + `httptest`) | `-run EffortResult` → 4 de 4 `--- FAIL` por aserción o por el helper: la vista solo traía «Esfuerzo registrado.» (sin fecha, horas, actividad ni selección) y `registerEffort` no recibía comando de métricas («expected a command to execute, got nil»). La prueba de miga falló en el mismo helper, no en el texto de la miga | Todas en verde con `effortMetricsStatus`, `startEffortMetrics`, `applyEffortMetrics`, `viewEffortResult` y la miga «Esfuerzo › Resultado» | Tarea con integrante y «Toda la historia» (D12: el servidor responde `2026-10-06`, `3.75` y `Implementar el login`, distinto de lo enviado); métricas `30.5/-4`, `10/4` (sin sumas) y `null` | Ver 5.11 |
| 5.3 / 5.4 | (misma) (`EffortResultRefreshesMetricsWithR`, `IgnoresRWhileMetricsLoading`) | Unitaria | `Refreshes`: «r should query the metrics again». `IgnoresR` nació en verde (sin `r` enlazada); se implementó `r` **sin** la guarda de `loading` a propósito y entonces falló por aserción («r while a query is in flight must not emit another one») | `r` con guarda `metricsStatus != loading` | 7.5 → 9 h; un solo `GET /metrics` con la consulta en curso | n/a |
| 5.5 / 5.6 | (misma) (`EffortResultMetricsFailure*`, `RetryAfterMetricsFailure*`, `MetricsBodyNotInterpretable*`) | Unitaria | 3 de 4 rojas por aserción: con `404`, `500` y conexión cortada (`Hijacker`) la pantalla mostraba «Horas reales: sin datos» (un `404` con cuerpo JSON se interpreta como carga útil con todo nulo) o seguía en «Cargando métricas…»; sin «no están disponibles», motivo ni «r para reintentar» | En verde con `effortMetricsUnavailable`, `metricsErr` (de `sectionFailure` o «No se pudo interpretar la respuesta de la API.») y aviso con `statusWarnStyle` | Tres causas de fallo, reintento tras `404` y cuerpo `[` | n/a |
| 5.7 / 5.8 | (misma) (`EffortResultKeys`, `EffortResultFooterAndHelp*`, `EffortResultEscReturns*`, `EffortSecondRegistration*`) | Unitaria | Fallan por aserción: `q` no salía, `p` no abría el selector, `Esc` no salía de la pantalla (`Screen = "effort-result"`), pie sin `r`/`p`, ayuda genérica con `Tab` y `Enter`, y sin selector de tareas al volver | En verde con `updateEffortResult`, `leaveEffortResult`, `effortResultFooter`, `effortResultHelpRows` y `"effort-result"` en `quitsOnQ` | `Esc` conserva fila sintética, tareas y cursor sin requests; segundo `Esc` a historias; segundo registro con formulario inicial | n/a |
| 5.9 / 5.10 | (misma) + `internal/tui/effort_form_test.go` | Unitaria (caja negra y una de caja blanca) | `EffortProjectSwitchFromResult…`: «choosing the project should load its stories» (`selectProject` cae en el `default` y se queda en `effort-result`). `EffortResultMetricsAnsweredWhileProjectPickerIsOpenAreKept`: con la guarda por pantalla la respuesta que llega detrás del selector se descarta y la vista queda en «Cargando métricas…». `TestOpenSectionResetsEffortFlowKeepingSeq` (interna): `m.effort` no se reiniciaba | En verde con `returnTo == "effort-result"` en `selectProject`, guarda sin condición de pantalla (ver «Desviaciones de esta UT») y `m.effort = effortState{seq: …}` en `openSection` | Cambio de proyecto desde el resultado y desde el selector; respuesta tras el selector | n/a |
| 5.7 (extra) | `EffortResultUnreadableCreatedBodyShowsRawBody` | Unitaria | La etiqueta nueva «Respuesta de la API: no-json» se exigió en la prueba y falló antes de agregarla | Etiqueta agregada | n/a | n/a |
| 5.11 | (todas) | | | | | `viewEffortMetrics` extraído de `viewEffortResult`; `go vet ./...` limpio; `gofmt` (LF) sin salida; `go test ./internal/tui/... ./tests/unit/...` en verde |

### Pruebas nacidas en verde `[caracterización]` y su respaldo (mutantes manuales; archivos restaurados y verificados con `diff`)

- `TestEffortResultUnreadableCreatedBodyShowsRawBody` (D6): la rama de cuerpo crudo se escribió en el GREEN de 5.2 por adelantado (desvío de TDD admitido: el código existía antes que su prueba); solo la etiqueta tuvo RED. Mutante (no imprimir el cuerpo crudo) → falla.
- `TestEffortStaleMetricsFromPreviousRegistrationNeverShown` (2 subcasos): la guarda por `seq` se escribió en 5.2. Mutante (guarda sin `seq`) → falla «old answer arrives first»; «old answer arrives last» **sobrevive** a ese mutante (lo cubre `metricsStatus`). Mutante (guarda sin la condición `metricsStatus == loading`) → **sobrevive** en toda la suite: es redundante con `seq` en los caminos alcanzables (al salir, `effort` se reinicia y la pantalla ya no muestra el estado). Se mantiene como defensa en profundidad.
- `TestEffortLateMetricsAfterLeavingResultIsDropped`: guarda de regresión del escenario de la spec. **No hay mutante distintivo en caja negra**: una vez fuera del resultado el estado de métricas no se renderiza, así que ni siquiera quitar toda la guarda cambia la vista.
- `TestEffortProjectSwitchFromSelectorDiscardsFlow`: la rama `returnTo == "section"` ya reabría la sección. Mutante (eliminar esa rama de `selectProject`) → falla.
- `TestEffortResultKeys/Enter_does_nothing` y `/question_mark_opens_the_help`: nacieron en verde (sin enlace de `Enter`; `?` ya abría la ayuda). Mutantes: `Enter` también vuelve → falla `Enter_does_nothing`; `"effort-result"` en `entersText` → fallan `Keys` y `FooterAndHelp`.
- `TestEffortResultCrumbAndTitle`: falló en el helper (sin comando de métricas), no en la miga. Mutante (miga vacía) → falla.
- Mutantes extra que sí mataron: guarda de métricas con exigencia de pantalla → `…AnsweredWhileProjectPickerIsOpenAreKept`; `r` sin guarda de `loading` → `IgnoresRWhileMetricsLoading`; `openSection` sin reinicio de `m.effort` → `TestOpenSectionResetsEffortFlowKeepingSeq`.
- Incidente de proceso: el primer intento del mutante «sin `seq`» tocó por error la guarda de `applyEffortSave` (mismo texto); se detectó porque la prueba seguía en verde, se restauró y se repitió sobre la guarda correcta.

### Defectos reales

Ninguno en producción previa. Hallazgos de la propia UT: (1) un `404` de métricas con cuerpo JSON se interpreta como carga útil válida con todo nulo y mostraría «Horas reales: sin datos» como si fuera un éxito (observado en el RED de 5.5; lo evita comprobar `sectionFailure` antes que el cuerpo); (2) la guarda de `applyEffortMetrics` pedida por 5.2 (`Screen == "effort-result" && …`) deja la pantalla en «Cargando métricas…» para siempre si el usuario abre `p` con la consulta en curso y vuelve con `Esc` (la respuesta se descarta y `r` se ignora mientras `loading`); observado en el RED de 5.9.

### Evidencia de unidad de trabajo

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado y resultado | `go test ./tests/unit/tui/... -run 'EffortResult\|EffortLate\|EffortStale\|EffortProjectSwitch\|EffortSecond' -count=1` → `ok` (22 pruebas de nivel superior en verde, con sus subcasos) y `go test ./internal/tui -run TestOpenSectionResets` → `ok` |
| Comando de la unidad completa | `go build ./...`, `go vet ./...`, `go test ./internal/tui/... ./tests/unit/...` → `ok` en todos los paquetes tras el commit |
| Arnés de ejecución | N/A: `Model.Update` con `httptest` (incluida una conexión de métricas realmente cortada con `Hijacker`); el recorrido manual opcional queda para 7.3 |
| Límite de reversión | `acb64f2`: `effort-result` completo en `effort_form.go` (estado, `startEffortMetrics`, `applyEffortMetrics`, teclas, pie, ayuda, vistas), cableado en `model.go`, `selectProject` en `project_picker.go`, reinicio de `m.effort` en `openSection`, `statusWarnStyle` y las pruebas nuevas. El `201` vuelve al marcador mínimo de la UT 4 |

### Carga de revisión

`git diff --cached --numstat` del commit: `effort_form.go` +148/-4, `model.go` +13/-1, `project_picker.go` +4, `sections.go` +1, `styles.go` +3, `effort_form_test.go` (interno) +25, `effort_test.go` +23/-6, `effort_result_test.go` +567. Total **795 líneas** autoras (producción ~174, pruebas ~620; estimación del plan: ~240). Rama acumulada contra `main` (solo archivos versionados): **3.044 líneas** (+2.937/-107); la Fase 6 suma ~150 más de artefactos y README. Cubierto por `size:exception` (PR único). El exceso es casi todo de pruebas por tabla y del fixture; no se recortó nada.

### Desviaciones de esta UT

- **Guarda de `applyEffortMetrics` sin condición de pantalla** (5.2 pedía `Screen == "effort-result" && metricsStatus == loading && seq`): solo `metricsStatus == loading && seq`. Motivo: defecto 2 de arriba. Descartar sigue funcionando porque salir del resultado reinicia `effort`. Prueba: `TestEffortResultMetricsAnsweredWhileProjectPickerIsOpenAreKept`.
- **Orden de TDD en 5.2**: la rama del cuerpo crudo (D6) y la guarda por `seq` y `metricsStatus` se escribieron en el GREEN de 5.2, antes de las pruebas de 5.5 y 5.9; esas pruebas se marcaron `[caracterización]` con mutantes. El resto de los RED de 5.5, 5.7 y 5.9 se observó contra el código real.
- **D6**: el cuerpo recibido se muestra tal cual con la etiqueta «Respuesta de la API: …» (no pedida; evita una línea suelta ininteligible).
- **D7**: pie «r refrescar métricas · Esc volver · p proyecto · ? ayuda · q salir» y ayuda con solo esas cinco teclas (sin `Ctrl+C`, que la spec no lista). `Enter` no se enlaza.
- **D8**: `selectProject` acepta `returnTo == "effort-result"` y abre siempre Esfuerzo en historias; `openSection` reinicia `m.effort` conservando `seq`. Como el reinicio no tiene efecto visible en caja negra (el formulario ya se rearma en `openEffortForm`), se verifica con la prueba interna `TestOpenSectionResetsEffortFlowKeepingSeq`.
- **D12**: la fila de tarea muestra `task.Label` (para la fila sintética, «Toda la historia (sin tarea)»); el formulario dice «Tarea: toda la historia». Texto distinto, ambos contienen la frase de la spec.
- **Estilo nuevo `statusWarnStyle`** (3 líneas en `styles.go`, color `warn` existente) para el aviso de métricas no disponibles; `tasks.md` solo pedía «estilo de aviso».
- **Fixture**: `effortFlowOptions.metricsDrop` y `metricsFailures`; `metricsStatus` ahora aplica a las respuestas «fallidas» con el cuerpo de error `metricsErrorBody`; rutas del proyecto `p2` (`/stories/s9/tasks` vacía y `/members`) para el cambio de proyecto. Nuevas ayudas: `registerEffort`, `resultWithMetrics`, `metricsRequests`, `flatView`, `footerOf`, `registerAgain`.
- **Miga del selector de proyectos abierto desde el resultado**: muestra «Proyectos › Elegir proyecto» (rama `default`), no «Esfuerzo»; sin requisito en la spec ni prueba. Se informa por si se quiere unificar.
- **Frases largas**: el panel parte las líneas largas, así que las pruebas con frases enteras usan `flatView`.

## UT 6: specs, README y artefactos (solo documentación)

Commit: `docs(tui): actualizar specs y README para TUI-08` (incorpora al control de versiones el directorio del cambio y la fila de Esfuerzo del `README.md`).

### Ciclo TDD

**No aplica ciclo RED/GREEN/TRIANGULATE/REFACTOR**: la unidad no cambia código de producción ni pruebas, solo documentación (`README.md` y artefactos del cambio). No se escribió ninguna prueba nueva y no se inventa evidencia de TDD. La verificación sustituta son las comprobaciones con `rg` que pide `tasks.md` (6.1 y 6.2), más `go build ./...` como comprobación de cordura.

### 6.1 Encabezados de las specs delta

- El comando literal de `tasks.md` (`rg -n "^### Requirement: (…)$" openspec/specs/tui/spec.md`) **no devuelve líneas** en este árbol de trabajo (`rg -c` → sin coincidencias, código de salida 1). Causa: `openspec/specs/tui/spec.md` está en CRLF en el árbol de trabajo (`core.autocrlf=true`, índice en LF) y el ancla `$` no coincide antes del `\r`. No es un desajuste real de nombres.
- Con `rg --crlf` (mismo patrón) devuelve las tres líneas esperadas: `115:### Requirement: Navegación de solo lectura`, `646:### Requirement: Elegir integrantes, historias, tareas y Sprints con selectores navegables`, `679:### Requirement: Avisar el límite de las pantallas de navegación`.
- Los nombres de los tres MODIFIED del delta (`specs/tui/spec.md`, líneas 395, 431 y 471) coinciden exactamente con los de la spec vigente.
- Referencias cruzadas en la spec vigente (con `--crlf`): `179:### Requirement: Ingresar las fechas del proyecto como año, mes y día`, `451:### Requirement: Presentar los errores de la API del backlog`, `551:### Requirement: Estimar historias con Planning Poker`.
- Revisión extra de **todas** las referencias `Requirement: …` de los deltas: existen en la spec vigente (`Operar proyectos mediante la TUI`, `Crear historias mediante la TUI`, `Modificar historias desde la lista`, `Operar los Sprints mediante la TUI`) o en el propio delta (los requisitos ADDED nombrados entre sí y `Permitir operar el registro desde la TUI sin ampliar el contrato HTTP` de `esfuerzo`).
- Las specs delta **no se modificaron** (no hubo una corrección real). `openspec/specs/**` no se tocó.

### 6.2 README

- Fila de Esfuerzo: `| Esfuerzo       | historia → tarea o historia completa → integrante → registro | disponible (TUI-08) |` (antes: `historia → tarea` y `TUI-08` como flujo futuro).
- Se agregó un párrafo de una frase tras el párrafo explicativo de las *shells*: «Esfuerzo es la excepción: registra el esfuerzo mediante `POST` y muestra las métricas del proyecto devueltas por la API, sin calcular horas ni desvíos.». Va después de ese párrafo (y no entre la tabla y él) para no separar «Estas pantallas…» de la tabla que referencia.
- `rg -n "TUI-08|Esfuerzo" README.md` → línea 4 (introducción, previa), 61 (menú, previa), 67 (fila nueva) y 78 (frase nueva). No se tocaron las demás filas ni la sección «API de esfuerzo».
- Alineación: la celda nueva excede el ancho de la columna, así que esa fila queda desalineada en el texto fuente (se renderiza bien). No se realineó toda la tabla para mantener el diff mínimo (4 adiciones y 1 eliminación según `git diff --stat`).
- Finales de línea: `README.md` sigue en CRLF en el árbol de trabajo (`file`: «with CRLF line terminators»; 317 de 317 líneas con `\r`); el índice sigue en LF.
- Límite conocido: el párrafo previo del README todavía dice que las pantallas «terminan con un aviso» y «no crean, modifican ni eliminan datos»; la frase nueva lo matiza para Esfuerzo. No se reescribió ese párrafo (fuera del alcance de 6.2). La fila de Planning Poker sigue como `TUI-07` por la misma razón.

### Evidencia de unidad de trabajo

| Evidencia | Valor |
|-----------|-------|
| Comando focalizado y resultado | `rg --crlf -n` sobre los encabezados (6.1) → 3 de 3 MODIFIED y 3 de 3 referencias cruzadas; `rg -n "TUI-08\|Esfuerzo" README.md` (6.2) → fila nueva y frase nueva; `go build ./...` → limpio |
| Arnés de ejecución | N/A: documentación, sin frontera de ejecución |
| Límite de reversión | Revertir el commit de la UT 6: la fila y la frase del `README.md` y la incorporación del directorio del cambio; no afecta código |

### Carga de revisión

Adiciones más eliminaciones: `README.md` +4/-1 y los artefactos del cambio (propuesta, exploración, specs delta, diseño, tareas y este archivo), que entran al control de versiones por primera vez; ver el recuento del commit en el informe del lote. Cubierto por `size:exception` (PR único).

## Desviaciones respecto de tasks.md, diseño y spec

- 1.6 y 2.5 piden agregar `apply-progress.md` al commit de cada UT. Se siguió la instrucción vigente del orquestador: el directorio del cambio estaba sin versionar y se agrega en la Fase 6, así que `apply-progress.md` y `tasks.md` quedaron en el árbol de trabajo sin commit hasta el commit de la UT 6.
- **UT 6 (6.1)**: el comando `rg` literal de `tasks.md` necesita `--crlf` en este árbol de trabajo (CRLF); sin la opción no encuentra nada aunque los nombres coinciden.
- **UT 6 (6.2)**: la frase sobre `POST` y métricas se colocó después del párrafo de las *shells* y no entre la tabla y ese párrafo; el texto de la celda final de la fila es `disponible (TUI-08)`.
- `gofmt -l internal tests` no es utilizable tal cual (ver «Límites del entorno»).
- **UT 3 (D-ninguno, mecánicas)**: la UT se partió en dos commits (permitido por la cabecera de `tasks.md`); la prueba obsoleta `TestEffortFlowChoosesStoryAndTaskThenShowsTheTUI08Notice` y `effortServer` se eliminaron en el primer commit (3.12 las ubicaba en el segundo) porque la fila sintética ya la rompía desde 3.3 (el cursor cae en «Toda la historia», no en «Diseñar») y la dejaba en rojo.
- **UT 3**: `effortLabelWidth = len("Actividad:")` (con los dos puntos) en lugar de `len("Actividad")` de 3.7. Con el valor literal de `tasks.md` «Horas:» queda una celda a la izquierda de «Actividad:» (se observó 14 frente a 15). El formulario de proyectos conserva su desalineación previa (`projectLabelWidth` sin dos puntos); no se tocó. Prueba agregada: `TestEffortFormAlignsTheValueColumn`.
- **UT 3**: `effortState` no incluye todavía `seq`, `err` ni `saving`, y `NewModel` no crea `effortClient` ni fija `now`: ningún comportamiento de esta UT los observa (TDD: sin código sin prueba). `today()` ya cae a `time.Now` cuando `now` es nulo (`TestTodayFallsBackToTimeNowForZeroClock`), de modo que `NewModel` no necesita fijarlo. La UT 4 agrega `seq`/`err`/`saving`/`effortClient` con sus pruebas; `Esc` de 3.11 ya no tiene `err` que limpiar.
- **UT 3**: el pie del formulario ya anuncia `Enter siguiente/enviar` (texto fijado en `tasks.md`) aunque `Enter` en «Horas» todavía no envía; el envío llega en la UT 4.
- **UT 3 (D1/D10 confirmados)**: la línea «La historia no tiene tareas.» se muestra bajo la lista cuando solo existe la fila sintética (D1); la ayuda del formulario no existe porque `?` es un carácter y el pie cumple el requisito (D10).
- **UT 3**: el RED de 3.4 falla en el helper de apertura (`expected a command to execute, got nil`) y no en una aserción de texto, porque elegir una tarea no cargaba integrantes. Es un fallo de comportamiento, pero no "por aserción" literal.
- **UT 3**: `OffersWholeStoryWhenStoryHasNoTasks` incluye `Enter` → carga de integrantes, que solo se cumple con 3.5; por eso 3.3 deja 3 de 4 pruebas en verde y la cuarta cierra en 3.5.

## Defectos reales encontrados

Ninguno en el código existente durante el lote 1 ni el lote 2. Nota del lote 2: el fixture `sprintServer` de las pruebas existentes responde `200 {"ok":true}` a rutas no declaradas (riesgo conocido); el nuevo `effortFlowServer` responde `404`, así que las pruebas de Esfuerzo no dependen de ese fallback.

## Suite completa

Cierre del lote 1, con ambas UT ya confirmadas en commit:

- `go build ./...` y `go vet ./internal/tui/...`: limpios.
- `go test ./...` (paralelo, con Docker): todas las pruebas unitarias en verde (`internal/tui`, `tests/unit/**` incluido `tests/unit/tui`). **Falló `tests/integration/task/postgres`** (`TestTaskBlocksUnassigningItsStory` y `TestTasksMigrationCanBeReversedAndReapplied`) con `wait for PostgreSQL: ... failed to receive message: unexpected EOF`. La salida capturada solo conservó las últimas 50 líneas, así que no se pudo ver si otros paquetes de integración fallaron en esa corrida paralela.
- Atribución: el fallo **no es propio**. Los paquetes de integración no dependen de `internal/tui` (`go list -deps -test ./tests/integration/...` no contiene `internal/tui`) y el error ocurre al conectar con el contenedor de Testcontainers, antes de ejecutar lógica de la aplicación.
- Prueba de que es de entorno: `go test ./tests/integration/... -p 1` (secuencial) pasa **completo**: `defect` 126 s, `effort` 154 s, `metrics` 6 s, `migrations` (caché), `project` 20 s, `projectmember` 24 s, `sprint` 104 s, `story` 377 s y `task/postgres` 89 s, todos `ok`. La corrida paralela de contenedores en Windows/Docker Desktop es inestable por carga.
- Recomendación para los lotes siguientes: usar `go test ./internal/tui/... ./tests/unit/...` por unidad y reservar `go test ./... -p 1` para el cierre.

### Cierre del lote 2 (UT 3, ambos commits ya confirmados)

- `go build ./...`, `go vet ./...` (todo el repositorio) y `go vet ./internal/tui/...`: limpios.
- `go test ./internal/tui/... ./tests/unit/...`: todo `ok`.
- `go test ./... -p 1` (secuencial, con Docker): **exit 0**, sin ningún paquete en fallo. Los paquetes de integración figuran `(cached)`: Go reutiliza el resultado del lote 1 (`-p 1` completo y en verde) porque sus entradas no cambiaron y no dependen de `internal/tui`.
- `git diff --stat -- tests/unit/tui/project_form_test.go` vacío; `TestMenuRoutesToEachSprint3Section` y `TestSectionNavigationOnlyEmitsGetRequests` sin editar y en verde. `sections_flows_test.go` solo perdió la prueba obsoleta y `effortServer`.
- `gofmt` (con `tr -d '\r'`) sin salida en todos los archivos tocados. `sections_flows_test.go` ya figuraba sin formato en `HEAD` (alineación del mapa de `metricsServer`); no es propio y no se tocó.

### Cierre del lote 3 (UT 4, commit `46a6beb`)

- `go build ./...`, `go vet ./...` y `go vet ./internal/tui/...`: limpios.
- `go test ./internal/tui/... ./tests/unit/...`: todo `ok`.
- `go test ./... -p 1` (secuencial, con Docker): **exit 0**, ningún paquete en fallo. Los paquetes de integración figuran `(cached)` (sus entradas no cambiaron y no dependen de `internal/tui`, como en el lote 2); `internal/tui` y `tests/unit/tui` se ejecutaron con el código del commit y quedaron en verde. No hubo fallos previos ni propios.
- `git diff --stat -- tests/unit/tui/project_form_test.go tests/unit/tui/sections_flows_test.go` vacío; `TestMenuRoutesToEachSprint3Section` y `TestSectionNavigationOnlyEmitsGetRequests` sin editar y en verde.
- `gofmt` con `tr -d '\r'` sin salida en los cuatro archivos tocados.

### Cierre del lote 4 (UT 5, commit `acb64f2`)

- `go build ./...`, `go vet ./...` y `go vet ./internal/tui/...`: limpios.
- `go test ./internal/tui/... ./tests/unit/...`: todo `ok`.
- `go test ./... -p 1` (secuencial, con Docker): **exit 0**, 37 paquetes `ok`, ninguno en fallo. Los paquetes de integración figuran `(cached)` (entradas sin cambios y sin dependencia de `internal/tui`, como en los lotes 2 y 3); `internal/tui` y `tests/unit/tui` se ejecutaron con el código del commit. No hubo fallos previos ni propios.
- `git diff --stat -- tests/unit/tui/project_form_test.go tests/unit/tui/sections_flows_test.go` vacío; `TestMenuRoutesToEachSprint3Section` y `TestSectionNavigationOnlyEmitsGetRequests` sin editar y en verde.
- `gofmt` con `tr -d '\r'` sin salida en los archivos tocados (`effort_form.go` y `effort_result_test.go` están en LF y pasan `gofmt -l` directo).

## Fase 7: verificación final y cierre

> **Estado al cierre**: la fila «7 | 7.1 a 7.5» de «Estado de tareas» y la línea del «Lote 5» describen el momento en que se escribieron. Esta sección registra el estado final de la fase, con los hechos observados por `sdd-verify` (Engram `sdd/tui-08-registrar-esfuerzo/verify-report`, observación #63) y lo ejecutado durante el archivado.

### Hechos observados por sdd-verify

- `go build ./...` y `go vet ./...`: exit 0.
- `go test ./... -p 1 -count=1`: exit 0; 37 paquetes `ok`, ninguno en fallo ni en caché. Docker disponible: los paquetes de integración corrieron contra contenedores reales (`defect` 147 s, `effort` 153 s, `metrics` 5,5 s, `migrations` 0,9 s, `project` 18 s, `projectmember` 31 s, `sprint` 106 s, `story` 371 s, `task` 76 s; `internal/tui` 2,5 s; `tests/unit/tui` 2,8 s).
- `go test ./...` en paralelo: exit 0 y 37 `ok`, pero 36 resultados vinieron de caché. La inestabilidad de Testcontainers (`unexpected EOF`) vista una vez durante apply no se reprodujo ni se descartó; la referencia fiable es `-p 1`.
- Pruebas focalizadas (`-count=1`, `Effort|DateField|DateRow|AlignedTextRow|Today|OpenSectionResets`): 77 pruebas de nivel superior y 77 subpruebas en verde. Cobertura unitaria de `effort.go` y `effort_form.go`: 100 %.
- Pruebas de red de seguridad: sin cambios respecto de `main` y en verde (`project_form_test.go`, `planning_poker_test.go`, `sections_api_test.go`, `footer_test.go`, `ux_test.go`, `selector_test.go`).
- Criterios: 9 de cierre y 11 de `proposal.md`, cumplidos. Escenarios: 78 (75 de `tui` y 3 de `esfuerzo`) en 10 requisitos, todos con al menos una prueba que los cubre.
- Discrepancias D1–D16: confirmadas; en cada caso el código sigue a la spec.

### 7.1 Build, vet, pruebas y gofmt

- Estado: **parcial**. La casilla queda sin marcar.
- `go build`, `go vet` y `go test -p 1 ./...`: en verde, con la evidencia anterior.
- `gofmt -l internal tests cmd` no está vacío por dos causas. (1) El árbol de trabajo está en CRLF (`core.autocrlf=true`) y gofmt reporta casi todo el repositorio. (2) Sobre el contenido del índice, dos archivos ya estaban sin formato en `main` y no se corrigen aquí: `tests/unit/tui/sections_flows_test.go` (tocado por este cambio, con el mapa `metricsServer` sin alinear) e `internal/tui/metrics.go` (sin cambios en este cambio). Comprobado con `git show main:<ruta> | gofmt -l`.
- Si el usuario acepta esa excepción, la casilla puede marcarse.

### 7.2 Criterios y discrepancias

- Cumplida. Evidencia: pruebas nombradas en la observación #63, por ejemplo `TestEffortSubmitSendsExactBodyForTask`, `TestEffortOnlyExplicitSubmitEmitsThePost`, `TestEffortFormShowsAPIErrorsKeepingData` y `TestEffortResultMetricsFailureKeepsRegistrationSuccess`.
- Discrepancias D1–D16: confirmadas (tabla «Discrepancias entre diseño y spec» de `tasks.md`).

### 7.3 Recorrido manual: no ejecutado

- Motivo: requiere una TUI interactiva contra API y PostgreSQL en vivo, que no se levantaron en esta fase.
- Evidencia parcial a nivel API: `internal/api/api.go:177` (`readiness.Effort`, versión >= 12) y `:181` (`readiness.Metrics`, versión >= 14), con `TestMigrationReadinessResolvesEffortByVersion` y `TestMetricsReadinessGatesRoutesWithoutChangingExistingReadiness` en verde.
- No ejercido: render real en terminal; aviso de métricas no disponibles contra una API en esquema 12–13; `r` actualizando «Horas reales» de extremo a extremo.

### 7.4 Corrección: no requerida

- No se agregó ningún commit `fix(tui)`. La verificación no exigió corrección; sus advertencias quedan registradas como decisiones del usuario en el reporte de archivo.

### 7.5 Archivo

- Ejecutado por `sdd-archive`: merge de `tui` (27 → 33 requisitos) y de `esfuerzo` (17 → 18) con `gentle-ai sdd-archive-compose`; edición manual del Propósito de `esfuerzo`; movimiento con `git mv` a `openspec/changes/archive/2026-10-31-tui-08-registrar-esfuerzo/`; commit local `docs(sdd): archivar tui-08-registrar-esfuerzo`.
- No se empujó ni se abrió PR, por instrucción del usuario.

### Notas de proceso

- La UT 3 se dividió en dos commits. La prueba de solo `GET` de la Fase 3 (3.12) se reescribió en la UT 4, porque su helper `press` descartaba el comando devuelto.
- Un `404` de métricas con cuerpo JSON se interpretó como carga válida con todos los valores nulos hasta comprobar antes el estado HTTP (UT 5).
