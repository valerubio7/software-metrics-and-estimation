# Propuesta: consultar los defectos de un proyecto (US-18)

## Intención

US-18 (#46) permite conocer los problemas detectados y su situación actual mediante la consulta de solo lectura del listado de defectos de un proyecto. La implementación ya está en `main` (PRs #117 y #118, commit `3035264`), y la especificación quedó pendiente, como registró el cambio `2026-10-30-fix-us-18-migracion-duplicada-pruebas` en su propuesta. Este cambio formaliza ese contrato en SDD.

## Cambios propuestos

- Ofrecer la consulta de solo lectura del listado de defectos del proyecto mediante `GET /projects/{project_id}/defects`, con soporte `HEAD`.
- Responder `200` con el sobre `{"project_id", "defects"}`; cada elemento con `id`, `description`, `severity`, `status`, `story_id`, `sprint_id` (Sprint de detección) y `resolution_sprint_id` (`null` si no tiene Sprint de resolución), sin exponer marcas internas como `created_at`.
- Conservar el orden de registro (`seq`), devolver un arreglo vacío explícito cuando el proyecto no tiene defectos y no ofrecer filtros ni paginación en esta ruta.
- Validar el UUID del proyecto antes de leer y traducir los errores con las convenciones existentes: `422 validation_failed`, `404 project_not_found`, `500 internal_error` genérico y `405 method_not_allowed` para los métodos no admitidos, sin lecturas.
- Disponer la ruta solo con el esquema limpio en la versión 14 o superior (migración `000014`), independiente del gate del alta (versión 13).

## Capacidades

### Capacidades nuevas

Ninguna. El listado extiende la capacidad `defecto` existente.

### Capacidades modificadas

- `defecto`: agrega la consulta del listado (US-18) y retira del alcance la exclusión del listado que el alta enunció en US-16. No modifica el contrato del alta ni introduce especificación de resolución de estado (US-17), edición o eliminación, métricas (US-19) ni operación desde la TUI (TUI-09).

## Impacto

Solo artefactos de especificación: propuesta, diseño, tareas y delta de `defecto`. No cambia código de producción, migraciones, rutas, composición ni pruebas; la verificación se ejecuta contra la implementación ya presente en `main` (`go build`, `go vet`, suites de pruebas y `openspec validate --strict`). También se comprueba el README, que ya documenta esta consulta en la sección «API de defectos». El archivado del cambio y la consolidación manual del spec de `defecto` (precedente TUI-08: `--skip-specs` porque los encabezados del repo son españoles) se realizan dentro de este mismo PR a `main` con `Closes #46`. Quedan fuera de este alcance la consolidación del SDD de TUI-09 y del updater de estado (US-17), ambas pendientes de sus propios cambios.
