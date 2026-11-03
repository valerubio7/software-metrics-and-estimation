# Informe de verificación: tui-11-visualizar-dashboard

- Commit verificado: `1ec3ff9`, rama `feat/tui-11-dashboard` (3 commits sobre `origin/main` `7f3604d`).
- Modo: TDD estricto, almacén `openspec`. Esta verificación cumple la tarea 4.3 de `tasks.md`; no se modificó código ni pruebas durante la verificación.
- Issue contrastado: **#104 (TUI-11 — Visualizar el Dashboard desde la TUI)**.
- **Veredicto: PASS** (0 CRITICAL, 0 WARNING, 2 SUGGESTION).

## 1. Ejecución real

| Comando | Resultado |
|---|---|
| `go build ./...` | exit 0, sin salida |
| `go vet ./...` | exit 0, sin hallazgos |
| `gofmt -l internal tests cmd` | `internal/tui/metrics.go` (preexistente en `main`, no modificado por este cambio) |
| `go test ./internal/tui/... ./tests/unit/tui/...` | `ok` en ambos paquetes |
| `go test ./...` | exit 0, 37 paquetes `ok`, 0 `FAIL`; Docker disponible y Testcontainers ejecutado |
| `openspec validate tui-11-visualizar-dashboard --strict` | `Change 'tui-11-visualizar-dashboard' is valid` |

La primera corrida de `go test ./...` registró un fallo ajeno al código en `tests/integration/story/postgres`: `testing: can't write /tmp/go-build.../testlog.txt: disk quota exceeded` (505 s). El paquete se reejecutó solo y quedó `ok` (473 s); la corrida completa posterior terminó con exit 0 y 37 `ok`. No se tocó código para resolverlo.

## 2. Contraste del issue: criterio por criterio

| Criterio de aceptación del issue #104 | Estado | Evidencia |
|---|---|---|
| Abrir el Dashboard del proyecto activo desde el menú, sin UUIDs | ✔ | `TestDashboardMenuOpensProjectDashboard`, `TestDashboardAsksForAProjectWhenNoneIsActive`; el picker continúa en Dashboard con el propósito `dashboard` |
| Mostrar el estado actual del proyecto | ✔ | `GET /projects/{id}`; vista con badge `EN CURSO` y `TestDashboardSwitchesProjectWithP` (`planificado`) |
| Incluir al menos una representación gráfica de terminal | ✔ | Barras Unicode `█/░`; `TestDashboardShowsBarsAndNoData`, `TestDashboardBarsStayInsideThePanel`, `TestDashboardBarClampsFractions` |
| Valores consistentes con Métricas (TUI-10/US-19); no recalcular | ✔ | Se reutilizan `parseMetricsPayload` y `formatMetric`; `TestDashboardShowsBarsAndNoData` verifica pares exactos de la API y el cero real |
| Métrica no calculable → «sin datos», no cero | ✔ | `TestDashboardAllNullMetricsShowNoBars` (≥6 filas «sin datos», ninguna barra) y fila Horas de `TestDashboardShowsBarsAndNoData` |
| Solo lectura | ✔ | `TestDashboardOnlyEmitsGETRequests` graba el recorrido abrir → refrescar → cambiar proyecto; todas las requests son `GET` |
| Legible a 80 y 44 columnas, sin depender solo del color | ✔ | `TestDashboardFitsNarrowTerminals` (ninguna línea excede el ancho) y `TestDashboardFooterAndHelpOnlyShowValidKeys`; `stripANSI` deja etiquetas, valores y glifos |
| Error de API con motivo, sin cerrar la TUI | ✔ | `TestDashboardShowsMetricErrorsWithoutClosingTheScreen` (503 + reintento con `r`) y `TestDashboardShowsProjectStatusErrorsAndKeepsMetrics` (500 conservando métricas) |

## 3. Trazabilidad spec ↔ pruebas

| Requirement (delta `tui`) | Pruebas principales |
|---|---|
| Abrir el Dashboard desde el menú | `TestDashboardMenuOpensProjectDashboard`, `TestDashboardAsksForAProjectWhenNoneIsActive` |
| Presentar los valores de las métricas sin recalcularlos | `TestDashboardShowsBarsAndNoData`, `TestDashboardAllNullMetricsShowNoBars` |
| Graficar las métricas con barras Unicode | `TestDashboardShowsBarsAndNoData`, `TestDashboardFitsNarrowTerminals`, `TestDashboardBarWidthFitsThePanel`, `TestDashboardBarClampsFractions` |
| Mantener el Dashboard de solo lectura y tolerante a errores | `TestDashboardOnlyEmitsGETRequests`, `TestDashboardShowsMetricErrorsWithoutClosingTheScreen`, `TestDashboardShowsProjectStatusErrorsAndKeepsMetrics`, `TestDashboardDiscardsStaleAndForeignResults` |
| Avisar el límite de las pantallas de navegación (MODIFIED) | `TestDashboardMenuOpensProjectDashboard` comprueba que no reaparece el aviso TUI-11; el resto de las secciones conserva sus pruebas |

## 4. Calidad del proceso

- RED/GREEN/TRIANGULATE/REFACTOR registrados por unidad en `apply-progress.md`, con el comando y el fallo observado.
- Tres commits separan pantalla base, gráficos y cobertura de errores/solo lectura; cada uno deja la suite de la TUI en verde.
- Las pruebas de modelo usan `httptest` y `Model.Update`, sin proceso real; las de helpers viven en el paquete `tui` para no exportar símbolos solo por prueba.
- La UT 3 no requirió cambios de producción: el manejo de errores y la secuencia ya se habían implementado en la UT 1 para no dejar la pantalla a medias. Se documentó así en `apply-progress.md`.

## 5. Sugerencias (no bloqueantes)

1. **SUGGESTION**: la pantalla ejecuta dos lecturas secuenciales (estado y métricas) dentro de un solo `tea.Cmd`. Si la primera falla, la segunda igual se intenta, lo que permite mostrar las métricas aunque el estado no esté disponible. Es el comportamiento buscado, pero no hay prueba de latencia real con la API en vivo.
2. **SUGGESTION**: `internal/tui/metrics.go` sigue sin `gofmt` en `main` y este cambio no lo tocó. Corregirlo en un cambio dedicado evitaría que futuras verificaciones arrastren la misma excepción.

## 6. Veredicto

**PASS.** El issue #104 queda satisfecho: Dashboard se abre desde el menú para el proyecto activo, muestra estado y métricas de US-19 sin recalcular, grafica con barras Unicode, distingue «sin datos» de cero, es de solo lectura, se adapta a 80/44 columnas y tolera errores sin cerrarse. El cambio queda listo para archivar y consolidar su delta en `openspec/specs/tui/spec.md`.
