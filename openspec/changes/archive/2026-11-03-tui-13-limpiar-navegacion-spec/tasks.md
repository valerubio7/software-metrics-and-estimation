# Tareas: limpiar las referencias al menú eliminado (TUI-13)

Cambio chico de una sola unidad de trabajo; la evidencia observada está en `apply-progress.md`.

- [x] 1. Prueba de que el último paso de una sección no muestra el aviso de TUI-06 (RED) y borrado del paso de aviso en `sections.go`, `keys.go` y `model.go` (GREEN). Requirement eliminado: Avisar el límite de las pantallas de navegación.
- [x] 2. Delta de `tui` con los requisitos modificados y el eliminado; `openspec validate tui-13-limpiar-navegacion-spec --strict` sin ERROR.
- [x] 3. Archivar el cambio, consolidar el delta en `openspec/specs/tui/spec.md` y validar `openspec validate tui --type spec --strict`.
