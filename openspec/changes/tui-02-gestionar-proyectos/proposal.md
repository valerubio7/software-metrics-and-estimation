# Propuesta: gestionar proyectos desde la TUI

## Motivo

La issue #84 solicita administrar proyectos desde la interfaz de terminal, sin recurrir directamente a la API. La TUI-01 ya proporciona el menú y la configuración de conexión; esta propuesta agrega flujos de proyectos que consumen los contratos HTTP existentes.

## Cambios

- Incorporar desde la opción «Proyectos» una pantalla para crear proyectos con nombre, fecha de inicio y fecha planificada de finalización.
- Permitir editar los datos básicos de un proyecto existente.
- Consultar y presentar el estado derivado por el servidor (`planned`, `active`, `overdue`).
- Mostrar al usuario los errores HTTP de validación (422) y de recurso inexistente (404), conservando el motivo proporcionado por la API.
- Mantener la TUI como cliente HTTP, sin replicar validaciones ni reglas de negocio.

## Capacidades

### Capacidades nuevas

- Ninguna: los flujos extienden la capacidad existente `tui`.

### Capacidades modificadas

- `tui`: agregar administración de proyectos a la navegación TUI.

## Impacto

- Cambios en `internal/tui/` y pruebas unitarias correspondientes.
- Actualización de la especificación de la capacidad `tui`.
- Sin cambios en endpoints, contratos del backend, persistencia ni reglas de dominio.
- Fuera de alcance: gestión de backlog (TUI-03), sprints (TUI-04/TUI-05) y lógica de negocio dentro de la TUI.
