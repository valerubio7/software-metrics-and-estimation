# Exploración: TUI-08 — Registrar esfuerzo desde la TUI (issue #101)

## Estado actual

### API

- Ruta: `POST /projects/{project_id}/stories/{story_id}/effort-records` (`internal/effort/transport/http/handler.go`, registrada en `internal/api/api.go:279`).
- Cuerpo estricto: un campo desconocido o un tipo incorrecto devuelve 400.
  - `member_id`: UUID, obligatorio.
  - `task_id`: UUID o `null`, opcional. Si falta o es `null`, el esfuerzo es de la historia.
  - `work_date`: `YYYY-MM-DD`, obligatorio, no puede ser posterior a la fecha UTC del servidor.
  - `description`: obligatoria, se recorta, de 1 a 500 caracteres, sin NUL.
  - `hours`: obligatorio, mayor que 0 y como máximo 24, con hasta 2 decimales.
- Respuesta 201 plana: `{id, project_id, story_id, task_id|null, member_id, work_date, description, hours}`.
- Errores: 400 `invalid_request`; 422 `validation_failed` (todos los `fields` juntos, con precedencia sobre los 404); 404 `project_not_found`, `story_not_found`, `member_not_found` o `task_not_found`; 500 `internal_error`.
- La ruta **nunca emite 409**. El issue lo menciona, pero el manejo genérico de `describeBacklogFailure` ya lo cubre.
- **No existe ningún GET de registros de esfuerzo**; la spec de esfuerzo lo prohíbe expresamente.

### TUI

- Esfuerzo es `sectionEffort` en `internal/tui/sections.go`. Hoy el flujo es historia (`backlogClient.List`) → tarea (`taskClient.ListByStory`) → aviso «estará disponible en TUI-08» (`sectionNotice`).
- No hay paso de integrante para Esfuerzo. `sectionStepMembers` existe, pero su vista está cableada a `viewPokerParticipants`.
- El selector compartido (`selector.go`) maneja filtro, cursor, carga y error. `r` refresca y `Esc` limpia el filtro o vuelve. `applySectionResult` descarta respuestas tardías.
- `Esc` desde cualquier paso secundario vuelve a `primary` (`sections.go` ~305). Con tres niveles se perdería el paso de tareas: hace falta un snapshot por nivel.
- Una tarea solo existe dentro de un Sprint, y una historia sin tareas admite esfuerzo igualmente. El selector actual muestra «La historia no tiene tareas», lo que bloquearía esas historias.
- `date_field.go` aporta `dateField`, `typeDigit`, `backspace` y `validate`, pero el render (`projectDateRow`) y la edición (`updateProjectDate`, `model.go:391`) dependen de `m.formField`, `m.formSegment`, `m.formStart` y `m.formFinish`; no se pueden reutilizar tal cual.
- Teclas globales `q`, `?` y `p` se interceptan salvo en pantallas listadas en `entersText`; `quitsOnQ("section")` es verdadero. El formulario **no puede vivir dentro de `Screen == "section"`**. Precedentes con pantalla propia: `sprint-task-form`, `backlog-form` y `planning-poker`.
- Precedente funcional más cercano: Planning Poker (TUI-07). Usa `StoryPointsClient`, un mensaje propio `pokerSaveResult` y `describeBacklogFailure`; tras el éxito refresca el backlog; sus campos son `pokerX` planos en `Model`.

### Tests

- Los tests de la TUI están en `tests/unit/tui/` (paquete `tui_test`) y no necesitan Docker.
- Helpers: `apply`, `run`, `press`, `stripANSI`, `sprintServer`, `recordingServer`, `onlyRequest`, `openSectionWithProject`, `effortServer`.
- Hay que reemplazar `TestEffortFlowChoosesStoryAndTaskThenShowsTheTUI08Notice` (`sections_flows_test.go:153`). `TestMenuRoutesToEachSprint3Section` y `TestSectionNavigationOnlyEmitsGetRequests` siguen vigentes.

### Specs y documentación

- `openspec/specs/tui/spec.md`:
  - «Avisar el límite de las pantallas de navegación» (l.679–688) debe dejar de cubrir Esfuerzo.
  - «Navegación de solo lectura» (l.115–131) afirma que Esfuerzo solo emite GET; contradice TUI-08 y ya está desfasado por TUI-07.
  - Las menciones de Esfuerzo en l.37, l.129 y l.674 deben revisarse.
- `openspec/specs/esfuerzo/spec.md`: el Propósito dice «no se opera desde la TUI (TUI-08)».
- `README.md:67`: la fila «Esfuerzo | historia → tarea | TUI-08» pasa a historia → tarea → integrante.

## Áreas afectadas

- `internal/tui/effort.go` (nuevo): `EffortClient.Register`, `EffortInput`, `EffortRecord`.
- `internal/tui/effort_form.go` (nuevo): estado `effortState`, update, view, footer y ayuda del formulario, validación local y pantalla de resultado.
- `internal/tui/sections.go`: fila «Toda la historia», paso de integrante para Esfuerzo, apertura del formulario, snapshot de selectores para `Esc`, retiro del aviso, vista de integrantes según el tipo de sección.
- `internal/tui/model.go`: campos `effortClient` y `effort`, cableado en `NewModel` y `Update`, listas `entersText` y `quitsOnQ`, `crumbParts`, `footer`, `helpRows`.
- `internal/tui/date_field.go` y `model.go` (`updateProjectDate`): extraer helpers genéricos de edición y render.
- `tests/unit/tui/effort_api_test.go` y `effort_test.go` (nuevos); ajustes en `sections_flows_test.go` y `effortServer`.
- Specs: delta de `tui` (nuevo requirement, más MODIFIED de «Navegación de solo lectura» y «Avisar el límite»), delta de `esfuerzo` (Propósito) y fila del `README.md`.

## Enfoques

| # | Enfoque | Pros | Contras | Esfuerzo |
|---|---------|------|---------|----------|
| 1 | Pantalla propia (`effort-form` y resultado) con estado agrupado `effortState`; los selectores reutilizan el shell | Sigue los precedentes (Poker, `sprint-task-form`); las teclas globales funcionan bien | Toca varios puntos de `model.go` | Medio |
| 2 | Formulario como `sectionStep` dentro de `Screen "section"` | Menos pantallas | `q`, `?` y `p` interfieren al escribir la actividad; exige excepciones en varios lugares | Medio, riesgo alto |
| 3 | Flujo independiente fuera del shell | Ninguno relevante | Contradice la spec de menú y selectores y el criterio de respetar el patrón navegable | Medio |

Sub-decisión de fecha: (a) reutilizar `m.formField` y `m.formSegment` acopla el formulario de esfuerzo al de proyecto; (b) extraer `dateField.applyKey(seg, key)` y `renderDateRow(label, d, focused, seg, width)` y usarlos en ambos formularios. Se recomienda (b).

Sub-decisión de estado: campos planos en `Model` (como Poker) o un struct `effortState`. Se recomienda el struct.

## Recomendación

Enfoque 1, con la opción (b) para la fecha y un struct `effortState`.

- **Flujo**: historia → tarea → integrante → formulario → POST → resultado.
  - El selector de tareas incluye una fila sintética «Toda la historia (sin tarea)» con ID vacío; con ella se omite `task_id`.
  - El formulario tiene Fecha segmentada, Actividad y Horas, con Tab, flechas y Enter como el formulario de proyecto.
- **Validación local mínima**: integrante e historia elegidos, fecha completa y válida (`dateField.validate`), y horas numéricas mayores que 0. Si falla, se muestra el error en pantalla sin emitir request. El resto (≤ 24, 2 decimales, actividad de 1 a 500 caracteres, fecha futura) lo valida la API y se muestra su motivo.
- **Cliente**: `EffortClient`, siguiendo el patrón de `StoryPointsClient`. JSON: `{member_id, task_id (omitempty), work_date, description, hours}`.
- **Anti doble envío**: un indicador `saving` que ignore Enter mientras hay un POST en vuelo, porque la API admite duplicados.
- **`Esc`**: formulario → integrantes → tareas → historias.
- **Error de la API**: permanece en el formulario con los datos intactos.
- **Éxito**: pantalla con el registro devuelto por la API, sin calcular horas reales ni desvíos.

## Preguntas de producto abiertas

1. **Esfuerzo sobre la historia sin tarea**: confirmar la fila «Toda la historia (sin tarea)». Sin ella, las historias sin tareas no admiten esfuerzo. Propuesta: sí.
2. **«Reflejar el esfuerzo» y «refrescar con `r`»**: como no hay GET de registros, (A) mostrar el registro devuelto en el 201 (y `r` refresca los selectores, como en toda la sección) o (B) además consultar `GET /projects/{id}/metrics` y mostrar `actual_hours` y `deviation` tal como los devuelve la API. Propuesta: A.
3. **Fecha inicial del formulario**: vacía, hoy UTC (coincide con la regla del servidor; requiere reloj inyectable en `Model`) u hoy local (puede dar 422 por fecha futura al este de UTC). Propuesta: hoy UTC, editable.
4. **Alcance de la validación local**: solo horas > 0 y fecha válida (lo pedido), o también actividad no vacía, ≤ 24, 2 decimales y ≤ 500. Propuesta: solo lo pedido más la fecha.
5. **Separador decimal**: ¿aceptar coma («1,5»)? Hoy `parseEstimatedHours` usa `ParseFloat` y la rechaza.
6. **Confirmación**: Enter en el último campo envía (como proyecto y tarea) o pantalla de confirmación explícita (como Poker).
7. **Después de un éxito**: volver a integrantes o tareas para cargar otro registro, o al selector de historias.
8. **Entrega**: `single-pr` con `size:exception` (como US-15) o cadena de 2 PRs.

## Riesgos

- **Duplicados**: un doble envío, o un reintento tras un timeout de 5 s con el POST ya confirmado, crea registros duplicados. El mensaje de error de conexión debe decir que no se pudo confirmar el registro.
- **Refactor de fecha**: toca el formulario de proyecto; lo cubre `project_form_test.go`.
- **`Esc` con tres niveles**: requiere un snapshot de selectores por nivel.
- **Deriva de specs**: los deltas deben partir del texto vigente, no del histórico.
- **Integrantes vacíos**: no hay UI para registrar integrantes; el estado vacío no debe apuntar a una opción inexistente del menú.
- **Tecla `r` en campos de texto**: es un carácter; el footer y la ayuda deben reflejarlo.

## Tamaño estimado (líneas autoras, adiciones más eliminaciones)

| Concepto | Líneas |
|---|---|
| `effort.go` | 60 |
| `effort_form.go` | 230 |
| `sections.go` | 60 |
| `model.go` | 40 |
| Refactor de `date_field.go` | 40 |
| **Producción** | **~430** |
| `effort_api_test.go` | 70 |
| `effort_test.go` | 350 |
| Ajustes a tests existentes | 40 |
| **Tests** | **~460** |
| Specs, README y artefactos del cambio | 120–180 |
| **Total** | **~1.000–1.100** |

Supera ampliamente el presupuesto de 400 líneas: con `single-pr` necesita `size:exception`; la alternativa es dividir en dos PRs (cliente, selectores y formulario sin envío; luego envío y resultado).

## Listo para propuesta

Sí, tras responder las preguntas 1, 2 y 8. Las demás tienen una propuesta por defecto.
