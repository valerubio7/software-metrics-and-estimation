# Apply progress: limpiar las referencias al menú eliminado (TUI-13)

Cambio de una sola unidad de trabajo, sin commit todavía.

## UT 1 — Borrar el paso de aviso de TUI-06 (tarea 1)

- **Alcance verificado**: `sectionKind` solo tiene `planning-poker`, `effort` y `defects`, y los tres definen un paso siguiente en `storyNextStep`. `openSection` solo se invoca con Estimación y Esfuerzo (`workspace.go`, `backlog_screens.go`, `effort_form.go`); Esfuerzo y Estimación terminan en el formulario y en la votación. La pestaña Defectos usa su propio flujo (`defects*`), por lo que el paso de aviso era inalcanzable desde la interfaz.
- **RED**: `internal/tui/sections_test.go` (`TestSectionLastStepDoesNotShowNotice`) fuerza el último paso de una sección (`defects`, paso Sprint) y elige una fila. `go test -count=1 -run TestSectionLastStepDoesNotShowNotice ./internal/tui` → `FAIL`: el paso pasa a `sectionStepNotice` (4 en lugar de 3) y la vista muestra «Próximamente», «TUI-09» y «No se guardó ningún dato».
- **GREEN**: se eliminaron `sectionStepNotice`, `sectionState.notice`/`noticeReturn`, `sectionNotice()`, `viewSectionNotice()`, la rama del aviso en `updateSection` y las dos de `chooseSectionItem`, la rama de `sectionBindings` en `keys.go` y la de la miga de pan en `model.go`. `storyNextStep` devuelve `sectionStepPrimary` para un tipo desconocido y `chooseSectionItem` lo trata como no-op. `go test -count=1 ./internal/tui ./tests/unit/tui` → `ok` en ambos paquetes.
- **Pruebas existentes**: ninguna cubría el aviso directamente; se conservan las guardias negativas `TestEffortFlowReachesFormWithoutTUI08Notice` y la aserción «TUI-11» del Dashboard.
- `gofmt -l internal tests` → solo `internal/tui/metrics.go` (preexistente); `go vet ./internal/tui ./tests/unit/tui` sin salida.

## UT 2 — Delta, validación, archivo y consolidación (tareas 2 y 3)

- `openspec validate tui-13-limpiar-navegacion-spec --strict` → sin ERROR; cuatro WARNING «should contain SHALL or MUST» esperados por la redacción en español.
- `openspec archive tui-13-limpiar-navegacion-spec --yes --skip-specs` → archivado como `2026-11-03-tui-13-limpiar-navegacion-spec` (avisó la falta de «## Why», igual que los cambios anteriores redactados en español).
- Consolidación manual en `openspec/specs/tui/spec.md`: se reemplazaron los cinco requisitos modificados en su lugar y se quitó «Avisar el límite de las pantallas de navegación». Requisitos: 38 → 37.
- `openspec validate tui --type spec --strict` → 0 ERROR, 53 WARNING (SHALL/MUST).
- `grep -n "menú" openspec/specs/tui/spec.md` → solo líneas `(Previously: …)`.
- `README.md` no mencionaba el menú ni los avisos de TUI-06: sin cambios.
