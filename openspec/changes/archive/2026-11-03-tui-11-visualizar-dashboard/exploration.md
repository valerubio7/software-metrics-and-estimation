# Exploración: Dashboard de la TUI (TUI-11)

## Estado de partida

- El menú de la TUI ya tiene nueve opciones y la séptima abre `sectionDashboard` (`internal/tui/model.go`, `updateMenu`).
- `openSection` trata Dashboard como una sección sin historias: carga los Sprints abiertos, permite elegir uno y termina en `sectionNotice`, que muestra «El Dashboard estará disponible en TUI-11» (`internal/tui/sections.go`).
- TUI-06 dejó ese shell documentado; TUI-11 debe reemplazarlo por la pantalla real. No existe un aviso pendiente para Métricas desde TUI-10 ni para Defectos desde TUI-09, aunque la consolidación SDD de TUI-09 sigue pendiente.
- La TUI se construye con Bubble Tea, Lip Gloss y `ansi`; el patrón de pantalla es un `Screen` string en `Model`, un `Update` por pantalla, un `viewBody` y helpers de `chrome`, panel, footer y ayuda (`internal/tui/model.go`).
- Las pruebas de modelo viven en `tests/unit/tui` (paquete `tui_test`) y usan `httptest`, `press`, `apply`, `run`, `keyMsg`, `typeText`, `stripANSI` y `sprintServer`. Los helpers no exportados se prueban en `internal/tui` (paquete `tui`).

## Datos disponibles

- `GET /projects/{project_id}` responde `{id, name, status}`; `status` es `planned`, `active` u `overdue` y se deriva de las fechas contra la fecha actual (`internal/project/transport/http/handler.go`, `domain.DeriveStatus`).
- `GET /projects/{project_id}/metrics` responde el mismo sobre de US-19 que ya consume la pantalla de Métricas y el resultado de Esfuerzo: `story_points_planned`, `story_points_completed`, `velocity`, `estimated_hours`, `actual_hours`, `deviation`, `completed_stories_percentage`, `defects_detected` y `defects_resolved`, todos punteros para distinguir «sin datos» de cero.
- `MetricsClient.Project` y `ProjectClient.Status` ya existen (`internal/tui/metrics.go`, `internal/tui/projects.go`); no hace falta un cliente nuevo.
- `parseMetricsPayload` y `formatMetric` ya implementan el contrato de «sin datos» y el formateo de valores de US-19.
- `projectStatusBadge` traduce `planned/active/overdue` a etiquetas `planificado/en curso/atrasado` con texto además de color (`internal/tui/components.go`).

## Restricciones y convenciones

- La pantalla debe ser de solo lectura: únicamente `GET`.
- El proyecto activo se guarda en `activeProjectID`/`activeProjectName`; el navegador de proyectos ya se reutiliza con `p` y con `openProjectPicker`.
- El panel (`card`) trunca líneas largas y envuelve texto plano al ancho interior; el chrome y el footer ya se adaptan al ancho. Las pruebas deben comprobar 80 y 44 columnas.
- Los textos nuevos de interfaz usan español neutro sin voseo; identificadores, rutas y comentarios de código en inglés.
- No hay que tocar la API ni las métricas: el Dashboard solo presenta lo que US-19 devuelve.
- Presupuesto de revisión del repo: 400 líneas. El cambio estimado lo supera; se entrega como PR único con excepción de tamaño documentada (precedente TUI-08).

## Hallazgos relevantes para el diseño

- No hay un endpoint de «estado del proyecto» distinto: el estado se obtiene con la lectura de proyecto ya existente.
- El sobre de métricas no incluye nombre ni fechas del proyecto; por eso se combinan dos consultas.
- La pantalla de Métricas ya usa `formatMetric`; reutilizarlo garantiza consistencia de valores y de «sin datos».
- Las barras deben ser texto y glifos: la TUI no tiene motor gráfico y la consigna aclara que «gráfico» significa barras o sparklines ASCII/Unicode.
- A 44 columnas el ancho interior es 40; las barras deben calcular su ancho desde `innerWidth` y no asumir 80.
- Las respuestas tardías deben descartarse al cambiar de proyecto o reintentar; la TUI ya usa secuencias en Esfuerzo y Defectos.
