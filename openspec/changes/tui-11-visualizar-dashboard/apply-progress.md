# Apply progress: visualizar el Dashboard desde la TUI (TUI-11)

Este archivo registra la evidencia observada de cada Unidad de Trabajo. Los hashes definitivos se completan en la UT 4, cuando el historial de la rama ya es estable.

## UT 1 — Abrir el Dashboard del proyecto activo

- **RED**: `go test ./tests/unit/tui/... -run Dashboard` → `FAIL`, `TestDashboardMenuOpensProjectDashboard` y `TestDashboardAsksForAProjectWhenNoneIsActive` terminan con `Screen = "section" after confirming the project, want dashboard`. La prueba falla por aserción: el menú todavía abre el shell de TUI-06.
- **GREEN**: `go test ./internal/tui/... ./tests/unit/tui/...` → `ok` en ambos paquetes. Se agregó `internal/tui/dashboard.go` con `dashboardState`, `dashboardResultMsg`, `openDashboard`, `loadDashboardCmd`, `applyDashboardResult`, `updateDashboard` y `viewDashboard`; el menú rutea a `openDashboard`, el picker acepta el propósito `dashboard`, la miga/footer/ayuda/quit incluyen la pantalla y se retiró `sectionDashboard` del shell de secciones.
- **TRIANGULATE**: las dos pruebas cubren proyecto activo y flujo sin proyecto activo (navegador → confirmación → Dashboard). La vista comprueba estado `EN CURSO`, nombre y valores `13 / 21`, `30.5 / 34.5 h`, `61.9%` y `1 / 3`, y la ausencia del aviso TUI-11.
- **REFACTOR**: la construcción de filas quedó en `viewDashboardMetrics` y el par de valores en `dashboardRatio`, sin cambiar el comportamiento; la suite completa de la TUI sigue en verde.
- **Commit**: `feat(tui): abrir el Dashboard del proyecto activo`.
