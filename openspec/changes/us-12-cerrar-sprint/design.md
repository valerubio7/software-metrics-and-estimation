# Diseño

## Contexto

La migración `000008` ya agrega `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`. La asignación en `internal/story/infrastructure/postgres/repository.go` bloquea el Sprint con `FOR UPDATE`, lee ese indicador y rechaza asignaciones cerradas. El endpoint de finalización de historias usa el mismo bloqueo. La API resuelve disponibilidad de rutas a partir de versión limpia del esquema.

## Objetivos y exclusiones

**Objetivos:** cerrar un Sprint de forma atómica, compatible con la serialización existente y sin tocar sus datos asociados; exponer la operación HTTP con errores consistentes y gate por esquema.

**No objetivos:** nuevas migraciones, interfaz TUI, consulta del historial (US-13), edición/eliminación del cierre, cierre automático ni restricciones sobre creación de tareas en Sprints cerrados.

## Decisiones

### Reutilizar el módulo `sprint`

La operación se implementa en `internal/sprint` (aplicación, transporte y persistencia), porque el cierre cambia el ciclo de vida del Sprint, no la asignación ni la finalización de historias. El repositorio de Sprint recibe un método de alcance de proyecto y la capa de aplicación traduce el comando a errores tipados. Se evita ampliar `story` o duplicar sus contratos.

### Actualizar el estado bajo bloqueo de fila y transacción

El repositorio verifica el proyecto, bloquea la fila de Sprint con `SELECT ... FOR UPDATE`, valida pertenencia y estado abierto, actualiza `is_closed=true` y confirma en una sola transacción. Este modo de bloqueo coincide con asignación y finalización, serializa las operaciones sobre el Sprint y hace que una asignación que espera vea el cierre confirmado. La transacción solo actualiza la fila `sprints`; no escribe ni elimina datos asociados.

### Endpoint y contrato HTTP

Se añade `POST /projects/{project_id}/sprints/{sprint_id}/closure`, sin cuerpo obligatorio, siguiendo la estructura de endpoints del módulo Sprint. La ruta valida UUID antes de invocar aplicación; mapea proyecto ausente a `404 project_not_found`, Sprint ausente o ajeno a `404 sprint_not_found`, cierre repetido a `409 sprint_closed`, entrada inválida a `422 validation_failed` y fallos inesperados a `500 internal_error`. El éxito es `200` con `project_id`, `sprint_id` e `is_closed`. Solo se registra si el esquema está limpio en versión >=8, sin cambiar gates existentes.

## Riesgos y compensaciones

- **Carrera con asignaciones:** una implementación sin el mismo bloqueo podría aceptar historias después del cierre → usar `FOR UPDATE` y prueba de integración concurrente con PostgreSQL.
- **Pérdida de historia por escrituras ajenas al cierre:** una actualización accidental de tablas relacionadas infringiría el requisito → probar preservación antes/después y limitar la mutación al indicador del Sprint.
- **Esquema parcial:** exponer la ruta antes de `000008` produciría fallos en tiempo de solicitud → gate independiente con pruebas de disponibilidad por versión.

## Plan de migración

No se requiere migración de base de datos. Desplegar la aplicación habilita la ruta solo en esquemas limpios versión 8 o superior. Revertir la aplicación retira la capacidad; el valor persistido puede permanecer y no debe revertirse automáticamente.
