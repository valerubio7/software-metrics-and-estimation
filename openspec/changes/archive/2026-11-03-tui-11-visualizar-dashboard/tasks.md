# Tareas: visualizar el Dashboard desde la TUI (TUI-11)

## Convención de commits y de ejecución

- Rama de trabajo: `feat/tui-11-dashboard`, creada desde `main`.
- Cada Unidad de Trabajo (UT) cierra con **un único commit** (Conventional Commits; solo el prefijo `type(scope)` va en inglés, el resto en español neutro). Cada commit contiene pruebas, código y documentación de su unidad, compila y deja en verde `go test ./internal/tui/... ./tests/unit/tui/...`.
- **TDD estricto**: RED (la prueba falla por aserción) → GREEN (código mínimo) → TRIANGULATE → REFACTOR sin cambiar comportamiento. Cuando una prueba necesita un símbolo nuevo, el esqueleto mínimo se agrega en el mismo paso RED.
- **Evidencia de apply**: al cerrar cada UT se actualiza `openspec/changes/tui-11-visualizar-dashboard/apply-progress.md` con RED/GREEN/TRIANGULATE/REFACTOR, comandos, resultados y hash del commit. El archivo se agrega al commit de su UT.
- **Estrategia de entrega**: un solo PR (`single-pr`) con `size:exception` documentada, igual que TUI-08. El flujo no tiene valor parcial entregable y el repo configura `single-pr`; el presupuesto de 400 líneas se supera por pruebas y artefactos SDD.
- Idioma: documentos y textos de interfaz en español neutro (sin voseo); identificadores, rutas y comentarios de código en inglés.
- Pruebas existentes que deben seguir en verde sin modificarse salvo las que describen el aviso retirado: `TestMenuExposesNineOptions`, `TestSectionNavigationOnlyEmitsGetRequests`, las suites de Métricas, Esfuerzo, Defectos y navegación.

## Review Workload Forecast


| Campo                   | Valor                                                                                          |
| ----------------------- | ---------------------------------------------------------------------------------------------- |
| Estimated changed lines | ~850–950 (adiciones más eliminaciones: ~230 producción, ~330 pruebas, ~300 specs/README)        |
| 400-line budget risk    | High                                                                                           |
| Chained PRs recommended | Yes                                                                                            |
| Suggested split         | PR único con `size:exception`; commits por UT revisables en orden                              |
| Delivery strategy       | single-pr                                                                                      |
| Chain strategy          | size-exception                                                                                 |


Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: size-exception
400-line budget risk: High

**Aclaración**: la estrategia `single-pr` es la configurada por el repo y el flujo no admite una porción entregable intermedia: una pantalla sin gráficos o sin estados de error no satisface TUI-11. La excepción de tamaño se documenta en el PR, como en TUI-08; el mantenedor decide la aprobación final.

### Líneas estimadas por Unidad de Trabajo


| UT  | Commit                                                               | Contenido principal                                                                 | Líneas aprox. |
| --- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------- | ------------- |
| 1   | `feat(tui): abrir el Dashboard del proyecto activo`                  | `dashboard.go` (estado, carga, vista base), menú, picker, miga, footer/ayuda; pruebas | ~330          |
| 2   | `feat(tui): graficar las métricas del Dashboard`                     | barras Unicode, reglas de «sin datos», ancho adaptable; pruebas                      | ~220          |
| 3   | `feat(tui): tolerar errores y terminales angostas en el Dashboard`   | errores por consulta, `r`, descarte de respuestas tardías, pruebas 80/44 y GET-only  | ~220          |
| 4   | `docs(tui): especificar y archivar el Dashboard (TUI-11)`            | propuesta, exploración, diseño, tareas, delta de `tui`, README y archivado           | ~300          |
|     | **Total**                                                            |                                                                                     | **~1.070**    |


### Suggested Work Units


| Unit | Goal                                                                 | Likely PR                 | Focused test command                                                                 | Runtime harness                                                      | Rollback boundary                                                                 |
| ---- | -------------------------------------------------------------------- | ------------------------- | ------------------------------------------------------------------------------------ | -------------------------------------------------------------------- | --------------------------------------------------------------------------------- |
| 1    | Pantalla Dashboard con estado y valores, menú y picker               | PR único (size:exception) | `go test ./tests/unit/tui/... -run 'Dashboard'`                                       | N/A — `httptest` y `Model.Update` sin proceso real                    | Revertir `dashboard.go`, el ruteo del menú/picker y las pruebas de dashboard       |
| 2    | Barras Unicode, «sin datos» y ancho adaptable                        | PR único (size:exception) | `go test ./tests/unit/tui/... -run 'Dashboard'`                                       | N/A — `WindowSizeMsg` de 80 y 44 y vista renderizada                  | Revertir los helpers de barra y las filas gráficas; la vista vuelve a solo texto   |
| 3    | Errores por consulta, reintento y descarte de respuestas tardías     | PR único (size:exception) | `go test ./tests/unit/tui/... -run 'Dashboard'`                                       | N/A — servidor con respuestas configurables y requests grabadas       | Revertir el manejo de errores/secuencia; la pantalla queda en el estado de la UT 2 |
| 4    | Artefactos SDD, README y archivado                                   | PR único (size:exception) | `openspec validate tui-11-visualizar-dashboard --strict` y `go test ./internal/tui/... ./tests/unit/tui/...` | N/A — documentación                                                      | Revertir el commit; no afecta código                                              |


## UT 1 — Abrir el Dashboard del proyecto activo

- [x] 1.1 RED: en `tests/unit/tui/dashboard_test.go`, probar que el menú (índice 7) abre `Screen == "dashboard"` con proyecto activo, consulta `/projects/p1` y `/projects/p1/metrics`, y muestra el estado `en curso`, el nombre del proyecto y los valores de las métricas. Ejecutar y observar el fallo por aserción (hoy abre `section` y muestra el aviso).
- [x] 1.2 RED: probar que sin proyecto activo el menú abre el navegador y, al confirmar un proyecto, continúa en Dashboard. Ejecutar y observar el fallo.
- [x] 1.3 GREEN: crear `internal/tui/dashboard.go` con `dashboardState`, `dashboardResultMsg`, `openDashboard`, `loadDashboardCmd`, `updateDashboard`, `applyDashboardResult` y `viewDashboard`; agregar el campo al `Model`, el caso de mensaje en `Update`, el `Screen` en `viewBody` y el ruteo del menú.
- [x] 1.4 GREEN: integrar `project_picker.go` (`dashboard` como propósito y retorno), `crumbParts`, `quitsOnQ` y `footer`; retirar `sectionDashboard` y su aviso de `sections.go`.
- [x] 1.5 TRIANGULATE: cubrir el cambio de proyecto con `p` y el refresco con `r`; verificar que la vista no muestra el aviso de TUI-11.
- [x] 1.6 REFACTOR: extraer la construcción de filas de métricas a un helper sin cambiar el comportamiento; ejecutar `go test ./internal/tui/... ./tests/unit/tui/...`.
- [x] 1.7 Evidencia: actualizar `apply-progress.md` (UT 1) y commit `feat(tui): abrir el Dashboard del proyecto activo`.

## UT 2 — Graficar las métricas del Dashboard

- [x] 2.1 RED: probar que las relaciones disponibles muestran barras `█/░` y que una métrica `null` muestra `sin datos` sin barra y sin `0`; incluir el porcentaje de historias y los defectos. Ejecutar y observar el fallo.
- [x] 2.2 RED: probar que a 80 y 44 columnas ninguna línea visible excede el ancho, que las barras siguen presentes y que al quitar ANSI se leen etiquetas y valores (no depende del color). Ejecutar y observar el fallo.
- [x] 2.3 GREEN: implementar `dashboardBar`, el cálculo de ancho desde `innerWidth`, las filas con barra y las reglas de `nil`/denominador cero.
- [x] 2.4 TRIANGULATE: cubrir fracciones mayores que 1 (recorte), cero real y relaciones sin datos; verificar que los valores son los de la API (sin recalcular historias ni métricas).
- [x] 2.5 REFACTOR: unificar el formateo de pares de métricas en un helper; ejecutar `go test ./internal/tui/... ./tests/unit/tui/...`.
- [x] 2.6 Evidencia: actualizar `apply-progress.md` (UT 2) y commit `feat(tui): graficar las métricas del Dashboard`.

## UT 3 — Errores, reintento y solo lectura

- [x] 3.1 RED: probar que un error de métricas muestra el motivo, mantiene `Screen == "dashboard"` y permite reintentar con `r`; probar también un error de estado de proyecto. Ejecutar y observar el fallo.
- [x] 3.2 RED: grabar las requests del recorrido y verificar que todas son `GET`; probar que `Esc` vuelve al menú y que una respuesta tardía de un proyecto anterior se descarta.
- [x] 3.3 GREEN: completar el manejo separado de errores, el reintento con `seq` y el descarte de respuestas tardías; ajustar footer/ayuda para anunciar solo `r`, `p`, `Esc`, `?`, `q` y `Ctrl+C`.
- [x] 3.4 TRIANGULATE: cubrir error de conexión y error HTTP; verificar que la pantalla nunca se cierra sola.
- [x] 3.5 REFACTOR: revisar mensajes y textos neutros; ejecutar `go test ./internal/tui/... ./tests/unit/tui/...`.
- [x] 3.6 Evidencia: actualizar `apply-progress.md` (UT 3) y commit `feat(tui): tolerar errores y terminales angostas en el Dashboard`.

## UT 4 — SDD, README y archivado

- [x] 4.1 Actualizar `README.md`: marcar Dashboard como disponible (TUI-11), describir estado, barras, «sin datos», 80/44 columnas y solo lectura.
- [x] 4.2 Completar el delta `openspec/changes/tui-11-visualizar-dashboard/specs/tui/spec.md` y validar con `openspec validate tui-11-visualizar-dashboard --strict`.
- [x] 4.3 Verificar `go build ./...`, `go vet ./...` y `go test ./...` (si Docker está disponible para integración) o las suites afectadas; registrar resultados en `verify-report.md`.
- [x] 4.4 Archivar el cambio en `openspec/changes/archive/2026-11-03-tui-11-visualizar-dashboard` y consolidar manualmente el requisito en `openspec/specs/tui/spec.md` (`--skip-specs` por los encabezados en español, precedente TUI-08/US-18).
- [x] 4.5 Escribir `archive-report.md`, actualizar `apply-progress.md` (UT 4) y commit `docs(tui): especificar y archivar el Dashboard (TUI-11)`.
- [x] 4.6 Abrir el PR único con `Closes #104`, resumen, evidencia de pruebas y nota de `size:exception`.
