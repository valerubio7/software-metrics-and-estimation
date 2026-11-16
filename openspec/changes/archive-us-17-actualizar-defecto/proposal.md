# Propuesta: actualizar el estado y la resolución de un defecto (US-17)

## Intención

US-17 (#45) permite reflejar la evolución de un defecto hasta su resolución, actualizando su estado y registrando el Sprint en el que se resolvió. La implementación ya está en `main` (PRs #111, #112, #113 y #116, consolidados el 26/10/2026), y la especificación quedó pendiente. Este cambio formaliza ese contrato en SDD, reconstruido a posteriori a partir del historial de Git (commits y diffs de las PRs mencionadas) el 2026-11-16; no es un registro contemporáneo a la implementación.

## Cambios propuestos

- Ofrecer `PATCH /projects/{project_id}/defects/{defect_id}` para actualizar el `status` de un defecto y, cuando corresponde, su `resolution_sprint_id`.
- Exigir `resolution_sprint_id` presente y válido únicamente cuando `status` es `"resuelto"`, y rechazarlo si se envía con cualquier otro estado.
- Validar que el Sprint de resolución exista en el proyecto y no sea anterior, en el orden de Sprints, al Sprint de detección del defecto.
- Permitir reabrir un defecto resuelto (volver a `detectado` o `en_correccion`), limpiando `resolution_sprint_id`.
- Responder `200` con el defecto actualizado; `422 validation_failed` por estado o Sprint de resolución inválidos; `404 defect_not_found` o `404 resolution_sprint_not_found`; `409 resolution_sprint_before_detection` por incoherencia cronológica; `400 invalid_request` por cuerpo malformado; `405 method_not_allowed` para métodos distintos de `PATCH`.
- Disponer la ruta solo con el esquema limpio en la versión 14 o superior (migración `000014`, que agrega `resolution_sprint_id`), igual que el gate del listado (US-18).

## Capacidades

### Capacidades nuevas

Ninguna. La actualización extiende la capacidad `defecto` existente.

### Capacidades modificadas

- `defecto`: agrega la transición de estado y resolución (US-17) y retira de la exclusión vigente la mención a esta operación. No modifica el contrato del alta (US-16) ni del listado (US-18), no introduce métricas (US-19) ni operación desde la TUI (TUI-09).

## Impacto

Solo artefactos de especificación: propuesta, diseño, tareas y delta de `defecto`. No cambia código de producción, migraciones, rutas, composición ni pruebas; la verificación se ejecuta contra la implementación ya presente en `main` (`go build`, `go vet`, suites de pruebas y `openspec validate --strict`). El archivado del cambio y la consolidación manual del spec de `defecto` se realizan en este mismo PR, siguiendo el precedente de TUI-08 y US-18 (`--skip-specs` porque los encabezados del repo son españoles). Queda fuera de este alcance la consolidación del SDD de TUI-09.
