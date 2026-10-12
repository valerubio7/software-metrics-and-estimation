# Proposal

## Why

Sprint 2 La Interfaz necesita operar el sistema sin curl. TUI-01 es la base navegable sobre la que colgarán TUI-02 a TUI-05.

## What Changes

- Nuevo binario `cmd/tui` con BubbleTea.
- Config de conexión vía `API_URL` (default `http://localhost:8080`).
- Menú base: proyectos, backlog, sprints, salir (placeholders no funcionales).
- Si la API no está disponible: muestra el motivo sin cerrarse de forma abrupta.
- Salida limpia con `q` / `Ctrl+C`.
- Navegación 100% solo-lectura, sin mutar datos.

## Capabilities

### New Capabilities

- `tui`: navegación base y configuración de conexión de la TUI.

### Modified Capabilities

- Ninguna (no cambia `project`, `historia`, `sprint` ni `tarea`).

## Impact

- Nuevo: `cmd/tui/`, `internal/tui/`, `tests/unit/tui/`, dependencia `bubbletea`.
- No cambia API backend ni migraciones.
- Alcance issue #83: excluye pantallas TUI-02 a TUI-05 y lógica de negocio en la TUI.
