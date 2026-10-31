# Propuesta: Registrar esfuerzo desde la TUI (TUI-08)

## Intención

Implementar TUI-08 (issue #101) para que un usuario pueda registrar desde la terminal el esfuerzo que un integrante dedicó a una historia o a una tarea de esa historia. La API ya lo permite desde US-15 (`POST /projects/{project_id}/stories/{story_id}/effort-records`), pero la sección Esfuerzo de la TUI solo recorre historia → tarea y termina en el aviso «estará disponible en TUI-08». Además, el selector de tareas bloquea las historias sin tareas, aunque la API admite esfuerzo sobre la historia completa.

El cambio completa el flujo historia → tarea → integrante → formulario → envío → resultado y, tras registrar, muestra el registro devuelto por la API junto con las métricas del proyecto que lo reflejan (`actual_hours`, `deviation`), sin que la TUI calcule horas ni desvíos.

## Decisiones confirmadas por el usuario

Estas decisiones están cerradas y son restricciones de este cambio; no se reabren en las fases siguientes.

| # | Tema | Decisión |
|---|------|----------|
| 1 | Esfuerzo sobre la historia sin tarea | El selector de tareas incluye una fila sintética «Toda la historia (sin tarea)». Al elegirla, el cuerpo omite `task_id`. Las historias sin tareas admiten esfuerzo por esa fila. |
| 2 | Reflejar el esfuerzo y refrescar con `r` (opción B) | Tras el `201`, la pantalla de resultado muestra el registro devuelto por la API y las métricas del proyecto leídas de `GET /projects/{id}/metrics` (`actual_hours`, `deviation`) tal como las devuelve la API, reutilizando `MetricsClient.Project`, `metricsPayload` y `formatMetric` de `internal/tui/metrics.go`. `r` en la pantalla de resultado vuelve a consultar las métricas. Si el `GET` de métricas falla, el esfuerzo sigue registrado: la pantalla lo indica y no muestra un error de registro. La TUI nunca calcula horas ni desvíos. |
| 3 | Entrega | Un solo PR con `size:exception` (estrategia `single-pr`). |

## Supuestos a confirmar

Se adoptan como valores por defecto. La fase de spec los fija como requisitos salvo que el usuario indique lo contrario.

| # | Tema | Supuesto |
|---|------|----------|
| S1 | Fecha inicial | El formulario abre con la fecha de hoy en UTC, editable. Requiere un reloj inyectable en `Model` para que los tests sean deterministas. Coincide con la regla del servidor (rechaza fechas posteriores a la fecha UTC). |
| S2 | Validación local | Solo lo que pide el issue: integrante e historia elegidos, fecha completa y válida, horas numéricas y mayores que 0. Si falla, se muestra el error en pantalla y no se emite ninguna request. El resto (≤ 24 horas, hasta 2 decimales, actividad de 1 a 500 caracteres, fecha futura) lo valida la API y la TUI muestra su motivo. |
| S3 | Separador decimal | No se acepta la coma decimal («1,5»); se conserva el comportamiento de `ParseFloat` usado en `parseEstimatedHours`. |
| S4 | Confirmación | `Enter` en el último campo envía, como en los formularios de proyecto y de tarea; no hay pantalla de confirmación aparte. Un indicador `saving` ignora `Enter` mientras hay un `POST` en curso, porque la API no es idempotente. |
| S5 | Después de un éxito | `Esc` (volver) desde el resultado regresa al selector de tareas de la misma historia, para cargar otro registro. |
| S6 | Dirección de diseño | Pantallas propias (formulario y resultado) fuera de `Screen == "section"`; estado agrupado en un struct `effortState`; helpers genéricos de edición y render del campo de fecha compartidos con el formulario de proyecto; `EffortClient` modelado sobre `StoryPointsClient`; snapshot por nivel de selector para que `Esc` recorra formulario → integrantes → tareas → historias. |

## Alcance

### Incluido
- Cliente `EffortClient.Register` con `EffortInput` (`member_id`, `task_id` con `omitempty`, `work_date`, `description`, `hours`) y `EffortRecord` (respuesta `201`).
- Fila sintética «Toda la historia (sin tarea)» en el selector de tareas de Esfuerzo, también cuando la historia no tiene tareas.
- Paso de selección de integrante para Esfuerzo, con su estado vacío propio (sin remitir a una opción de menú inexistente).
- Formulario de esfuerzo con Fecha segmentada (año, mes, día), Actividad y Horas; navegación con `Tab`, flechas y `Enter` como en el formulario de proyecto; validación local según S2; envío con protección contra doble envío (S4).
- Presentación de errores de la API (`400`, `422` con `fields`, `404 *_not_found`, `500`, error de conexión) reutilizando `describeBacklogFailure`; el formulario conserva los datos. El error de conexión indica que no se pudo confirmar el registro.
- Pantalla de resultado con el registro devuelto y las métricas del proyecto (`actual_hours`, `deviation`) leídas de la API; `r` refresca las métricas; un fallo de métricas no se presenta como fallo de registro.
- Navegación `Esc` de tres niveles con snapshot por selector; retiro del aviso TUI-08 para Esfuerzo.
- Registro de las pantallas nuevas en `entersText`, `quitsOnQ`, miga de pan, pie y ayuda, de modo que `q`, `?`, `p` y `r` sean caracteres en los campos de texto.
- Extracción de helpers genéricos de edición y render de `dateField`, usados por el formulario de proyecto y por el de esfuerzo, sin cambio de comportamiento en proyectos.
- Pruebas unitarias en `tests/unit/tui/` con TDD estricto (sin Docker).
- Deltas de spec de `tui` y `esfuerzo`, y fila de Esfuerzo en el `README.md`.

### Fuera de alcance
- Crear tareas (US-10) o integrantes desde la sección Esfuerzo.
- Calcular horas reales, totales o desvíos en la TUI (US-19); solo se muestran los valores que devuelve la API.
- Editar, eliminar o listar registros de esfuerzo.
- Cualquier endpoint nuevo de la API, incluido un `GET` de registros de esfuerzo (la spec de `esfuerzo` lo excluye).
- Aceptar la coma decimal (S3) o validaciones locales adicionales a S2.
- Flujos de Defectos y Dashboard (TUI-09 a TUI-11).

## Capacidades

### Capacidades nuevas
- Ninguna.

### Capacidades modificadas
- `tui`:
  - ADDED: requirement nuevo para registrar esfuerzo desde la TUI (flujo historia → tarea o «Toda la historia» → integrante → formulario → envío; validación local S2; anti doble envío; errores de la API; pantalla de resultado con registro y métricas del proyecto; `r` refresca métricas; fallo de métricas tras un `201`; `Esc` de tres niveles; vuelta al selector de tareas tras el éxito; teclas como caracteres en campos de texto).
  - MODIFIED «Navegación de solo lectura»: el registro explícito de esfuerzo pasa a ser una operación que emite `POST`; la navegación por la sección y sus selectores sigue emitiendo solo `GET`. El delta debe partir del texto vigente, que hoy afirma que Esfuerzo solo emite `GET` y que ya está desfasado respecto de TUI-07 (Planning Poker emite `PUT`).
  - MODIFIED «Avisar el límite de las pantallas de navegación»: deja de cubrir Esfuerzo.
  - Revisar las menciones de Esfuerzo en «Mostrar menú base de navegación» (escenario de las secciones) y en el escenario «Los cuatro tipos de selector son navegables», que siguen vigentes en lo que respecta a la selección sin mutación.
- `esfuerzo`: MODIFIED del Propósito, que hoy dice «no se opera desde la TUI (TUI-08)»; pasa a indicar que puede operarse desde la TUI. No cambia ningún requisito del contrato HTTP.
- Documentación: la fila «Esfuerzo» del `README.md` pasa de «historia → tarea | TUI-08» a historia → tarea → integrante → registro.

## Enfoque propuesto

Enfoque 1 de la exploración: pantallas propias con estado agrupado, reutilizando el shell de selectores de la sección.

1. **Cliente**: `internal/tui/effort.go` con `EffortClient.Register(ctx, projectID, storyID, input)`, siguiendo `StoryPointsClient` y el `apiClient` compartido (timeout de 5 s). `task_id` con `omitempty` cuando se elige «Toda la historia».
2. **Selectores**: en `sections.go`, el paso de tareas de Esfuerzo antepone la fila sintética con ID vacío; al elegir una fila se abre el paso de integrantes (`MemberClient`) con una vista propia para Esfuerzo (la actual está cableada a Planning Poker). Cada nivel guarda un snapshot del selector para que `Esc` restaure el nivel anterior sin volver a `primary`.
3. **Formulario y resultado**: `internal/tui/effort_form.go` con `effortState` (historia, tarea opcional, integrante, fecha, actividad, horas, campo y segmento enfocados, error, `saving`, registro y métricas). Pantallas propias fuera de `Screen == "section"`, porque `q`, `?` y `p` se interceptan en `section`. Mensajes asíncronos propios para el guardado y para la consulta de métricas, que descartan respuestas tardías.
4. **Fecha**: extraer de `updateProjectDate` y `projectDateRow` helpers genéricos (`applyKey` sobre `dateField` y un render de fila de fecha) y usarlos en ambos formularios. El formulario de proyecto conserva su comportamiento, cubierto por `project_form_test.go`.
5. **Resultado**: tras el `201`, se emite `MetricsClient.Project` y se muestran `actual_hours` y `deviation` con `formatMetric`. Si la consulta falla (por ejemplo, `404` porque la ruta de métricas exige el esquema `>= 14` mientras la de esfuerzo exige `>= 12`, o un error de conexión), la pantalla indica que el esfuerzo quedó registrado y que las métricas no están disponibles, con `r` para reintentar.
6. **Reloj**: `Model` recibe un reloj inyectable (`func() time.Time`) con valor por defecto `time.Now`, usado solo para la fecha inicial en UTC.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/tui/effort.go` | Nuevo | `EffortClient`, `EffortInput`, `EffortRecord`. |
| `internal/tui/effort_form.go` | Nuevo | `effortState`, update, view, pie y ayuda del formulario y del resultado, validación local, mensajes de guardado y de métricas. |
| `internal/tui/sections.go` | Modificado | Fila «Toda la historia», paso de integrante para Esfuerzo, snapshots por nivel, apertura del formulario, retiro del aviso TUI-08. |
| `internal/tui/model.go` | Modificado | Campos `effortClient`, `effort` y reloj; cableado en `NewModel` y `Update`; `entersText`, `quitsOnQ`, `crumbParts`, `footer`, `helpRows`; uso de los helpers de fecha en el formulario de proyecto. |
| `internal/tui/date_field.go` | Modificado | Helpers genéricos de edición y render de fecha. |
| `internal/tui/metrics.go` | Sin cambios (reutilizado) | `MetricsClient.Project`, `metricsPayload`, `formatMetric`. |
| `tests/unit/tui/effort_api_test.go`, `tests/unit/tui/effort_test.go` | Nuevo | Cliente, flujo, validación, errores, resultado, métricas y navegación. |
| `tests/unit/tui/sections_flows_test.go` y helpers (`effortServer`) | Modificado | Reemplazo de `TestEffortFlowChoosesStoryAndTaskThenShowsTheTUI08Notice`; ajustes al servidor de prueba. |
| `openspec/changes/tui-08-registrar-esfuerzo/specs/tui/spec.md`, `.../specs/esfuerzo/spec.md` | Nuevo (delta) | Deltas descritos en Capacidades. |
| `README.md` | Modificado | Fila de Esfuerzo. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Registros duplicados por doble `Enter` o por reintentar tras un timeout con el `POST` ya confirmado (la API no es idempotente). | Media | Indicador `saving` que ignora `Enter` en vuelo; el mensaje de error de conexión dice que no se pudo confirmar el registro y sugiere verificar antes de reintentar; test de doble `Enter` con una sola request. |
| La extracción de helpers de fecha altera el formulario de proyecto. | Media | Refactor en commit propio con `project_form_test.go` en verde antes y después; sin cambios de comportamiento observables. |
| `Esc` con tres niveles pierde el paso de tareas o el filtro (hoy todo paso secundario vuelve a `primary`). | Media | Snapshot por nivel de selector; tests de `Esc` formulario → integrantes → tareas → historias, incluido el filtro conservado. |
| Deriva de specs: «Navegación de solo lectura» y «Avisar el límite» ya están desfasadas respecto de TUI-07. | Media | Los deltas parten del texto vigente de `openspec/specs/tui/spec.md`; el MODIFIED reescribe el requirement completo y la fase de spec decide si alinea también la mención de Planning Poker sin cambiar su comportamiento. |
| Proyecto sin integrantes: no existe UI para registrarlos. | Media | Estado vacío propio que explica la falta de integrantes sin remitir a una opción de menú inexistente; permite volver y refrescar. |
| `r`, `q`, `?` y `p` son caracteres en Actividad y Horas, pero atajos en selectores y resultado. | Media | Pantallas incluidas en `entersText` y excluidas de `quitsOnQ` donde hay texto; pie y ayuda acordes; tests de escritura de esos caracteres. |
| El `GET` de métricas falla después de un `POST` exitoso (`404` por esquema `12`–`13`, error de conexión, `500`). | Media | Estado de resultado independiente para métricas: el registro se muestra como exitoso y las métricas como no disponibles, con `r` para reintentar; nunca se muestra un error de registro. |
| Fecha inicial en UTC confunde a un usuario en una zona horaria al oeste de UTC (puede ver «mañana» localmente). | Baja | Fecha editable; supuesto S1 explícito para confirmar. |
| El PR único supera el presupuesto de 400 líneas (~1.100–1.200 líneas autoras). | Alta | `size:exception` aprobado por el mantenedor; commits por unidad de trabajo para facilitar la revisión. |

## Plan de reversión

Revertir el PR único. El cambio es exclusivamente de cliente TUI, specs y documentación: no agrega migraciones, rutas ni datos de esquema. Al revertir, la sección Esfuerzo vuelve al flujo historia → tarea con el aviso TUI-08 y el formulario de proyecto vuelve a su implementación de fecha anterior. Los registros de esfuerzo creados desde la TUI mientras el cambio estuvo activo permanecen en la base de datos y son válidos para la API; no requieren limpieza.

## Dependencias

- US-15 integrada en `main` (ruta `POST .../effort-records`, esquema `>= 12`).
- US-19 integrada en `main` (`GET /projects/{id}/metrics`, esquema `>= 14`) para mostrar las métricas; si no está habilitada, el registro funciona y las métricas se muestran como no disponibles.
- TUI-06 y TUI-07 en `main` (shell de secciones, selectores, `MemberClient`, `MetricsClient`, `describeBacklogFailure`).

## Plan de entrega y carga de revisión

Un solo PR hacia `main` con la etiqueta `size:exception` (estrategia `single-pr`, decisión 3), aprobada por el mantenedor. Estimación: ~1.100–1.200 líneas autoras (adiciones más eliminaciones), frente al presupuesto de 400: ~430 de producción, ~460 de pruebas y el resto en specs, README y artefactos. Justificación: el flujo no tiene valor parcial entregable para el usuario; un formulario sin envío o un envío sin resultado no satisface el issue, y casi la mitad del volumen son pruebas. Dentro del PR se mantienen commits por unidad de trabajo con mensajes Conventional Commits en español (prefijo `type(scope)` en inglés): refactor de fecha, cliente, selectores y navegación, formulario y envío, resultado y métricas, specs y README.

## Decisiones abiertas para el orquestador

Ninguna bloqueante. Los supuestos S1 a S6 quedan para confirmación del usuario; la fase de spec los toma como valores por defecto.

## Criterios de éxito

- [ ] Desde Esfuerzo, el usuario elige historia, tarea o «Toda la historia (sin tarea)» e integrante, completa fecha, actividad y horas, y la TUI emite un único `POST /projects/{pid}/stories/{sid}/effort-records` con el cuerpo esperado (`task_id` omitido para la historia completa).
- [ ] Una historia sin tareas admite el registro mediante la fila «Toda la historia».
- [ ] Con integrante o historia sin elegir, fecha incompleta o inválida, u horas no numéricas o ≤ 0, la TUI muestra el error y no emite ninguna request.
- [ ] Los errores `400`, `422`, `404` y `500` y los de conexión se muestran con su motivo, el formulario conserva los datos y no se comunica éxito.
- [ ] Un segundo `Enter` con un `POST` en vuelo no emite otra request.
- [ ] Tras el `201`, la pantalla muestra el registro devuelto y `actual_hours` y `deviation` tal como los devuelve `GET /projects/{id}/metrics`; `r` vuelve a consultarlas; si la consulta falla, se indica que el esfuerzo quedó registrado y no se muestra un error de registro.
- [ ] `Esc` recorre formulario → integrantes → tareas → historias; tras un éxito, volver lleva al selector de tareas de la misma historia.
- [ ] `q`, `?`, `p` y `r` se escriben como caracteres en Actividad y Horas.
- [ ] El formulario de proyecto conserva su comportamiento (tests existentes en verde).
- [ ] Ya no aparece el aviso TUI-08; los deltas de `tui` y `esfuerzo` y la fila del `README.md` están actualizados.
- [ ] `go test ./...` pasa.
