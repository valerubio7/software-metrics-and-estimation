# Proposal

## Why

TUI-10 (#103) quedó parcialmente preparado con la selección de ámbito, pero todavía no consulta ni presenta métricas. Quienes usan la TUI necesitan evaluar avance, esfuerzo y calidad desde el proyecto activo o un Sprint, consumiendo los valores calculados por US-19 sin duplicar sus fórmulas.

## What Changes

- Completar la pantalla de Métricas para consultar los ámbitos de proyecto y Sprint mediante los endpoints GET de US-19.
- Mostrar los valores y estados faltantes que devuelve la API, además de refrescar y presentar errores sin cerrar la TUI.
- Mantener la pantalla de solo lectura, conservar el selector navegable existente y adaptar su presentación a terminales de 80 y 44 columnas.
- Cubrir el flujo con pruebas unitarias de cliente, navegación, refresco, errores y presentación; actualizar la especificación TUI y la documentación pertinente.

## Capabilities

### New Capabilities

Ninguna.

### Modified Capabilities

- `tui`: especificar consulta y presentación de métricas de proyecto/Sprint desde la TUI, estados sin datos, actualización, errores, solo lectura y legibilidad en terminales angostas.

## Impact

- TUI: `internal/tui/sections.go`, `internal/tui/model.go` y un cliente/presentador de métricas en `internal/tui/`.
- Pruebas: `tests/unit/tui/`.
- Especificación y documentación: delta para `openspec/specs/tui/spec.md` y README.
- Dependencia contractual: US-19 define `GET /projects/{project_id}/metrics` y `GET /projects/{project_id}/sprints/{sprint_id}/metrics`; la TUI consume la respuesta JSON plana, incluyendo `coverage` y `deviation_percentage`, y no implementa cálculos.
