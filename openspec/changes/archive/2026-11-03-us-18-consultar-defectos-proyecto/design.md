# Diseño de la consulta de defectos del proyecto (US-18)

## Contexto

Véanse la motivación en [proposal.md](proposal.md) y el contrato en [specs/defecto/spec.md](specs/defecto/spec.md). La implementación ya está en `main` (PRs #117, #118 y corregida por PR #126; commit `3035264`): dominio y caso de uso en `internal/defect/domain` y `internal/defect/application` (`DefectLister`, `ListDefectsUseCase`), transporte HTTP en `internal/defect/transport/http/list_handler.go`, repositorio PostgreSQL en `internal/defect/infrastructure/postgres/repository.go` (`ListByProject`) y composición/cableado en `internal/api/api.go` bajo `HTTPDependencies` con `ResolveMigrationReadiness` (`DefectList`: esquema limpio ≥ 14). La TUI la consume en `internal/tui/defects.go` (TUI-09). Este documento registra las decisiones tomadas en esa implementación; no introduce una nueva.

## Objetivos

**Objetivos:** especificar con fidelidad el contrato implementado, mantener la consulta de solo lectura, delimitar la lectura al proyecto de la ruta y dejar trazabilidad verificable entre escenarios y pruebas existentes.

**No objetivos:** no se modifican código, migraciones, rutas ni composición; no se especifican la resolución de estado con Sprint de resolución (US-17), la operación desde la TUI (TUI-09, pendiente de consolidar su propio SDD) ni los filtros o la paginación del listado.

## Decisiones registradas (tal como quedaron implementadas)

1. **Sobre en lugar de arreglo pelado.** La respuesta es `{"project_id", "defects"}`: la ruta se identifica a sí misma y el consumidor (TUI-09) decodifica un objeto determinado. `project_id` es el UUID canónico en minúsculas de la ruta; el arreglo vacío se garantiza en las tres capas (repositorio `make([]domain.Defect, 0)`, caso de uso `nil → []`, handler `make([]defectListItem, 0, len(...))`). Alternativa descartada: arreglo JSON pelado, que obligaría a encabezados o parámetros para identificar el proyecto.
2. **Siete claves por elemento.** `id`, `description`, `severity`, `status`, `story_id`, `sprint_id`, `resolution_sprint_id`; la última es `null` mientras no exista Sprint de resolución. La consulta MUST NOT exponer `created_at` ni el resto de columnas internas del almacenamiento.
3. **Orden de registro (`seq`) sin filtros ni paginación.** La tabla usa `seq` (migración `000013`, índice `(project_id, seq)`); `ORDER BY seq` da un orden estable e independiente de UUID o fechas. El filtrado es superficie de presentación: la TUI filtra en memoria con `/`, sin rutas nuevas.
4. **Lectura no transaccional y delimitada al proyecto.** El repositorio verifica la existencia del proyecto (una consulta) y luego ejecuta una única sentencia `SELECT ... WHERE project_id=$1 ORDER BY seq`; sin bloqueos ni escrituras. La delimitación por `project_id` en el `WHERE` excluye por diseño los defectos de otros proyectos.
5. **Validación antes de leer y errores existentes.** El caso de uso valida el UUID y lo canoniza antes del repositorio (422 sin lecturas); `404 project_not_found` para proyecto inexistente; `500 internal_error` genérico sin filtrar detalles; `405` con cero lecturas para métodos no admitidos. Se reutilizan los códigos y el cuerpo de error del módulo de defecto (`errorResponse` con `error`, `message` y `fields` opcionales).
6. **Gate por versión 14, independiente del alta.** `DefectList = esquema limpio && versión >= 14`: la migración `000014` agrega `resolution_sprint_id`. Con versión menor, `dirty`, fallo de resolución o dependencia ausente no se registra la ruta (observable: `405` en `GET` por patrón ausente; `404` genérico si el módulo entero no está instalado). El alta conserva su gate por versión 13.
7. **Contrato aditivo.** Nada del alta cambia: ruta, cuerpo, códigos, catálogos y gate se conservan íntegros; la única modificación del spec base es retirar la exclusión del listado y el renombrado correspondiente del requisito «Ofrecer únicamente el registro de defectos».

## Riesgos y notas de verificación

- **Escenario con contenido obsoleto en el spec base.** El requisito «Ofrecer únicamente el registro de defectos» y su escenario «Sin rutas de consulta, edición o resolución» afirmaban que la ruta de consulta no existe. El delta los modifica (renombrado del requisito, escenario conservado con el mismo título y pasos actualizados), y `openspec validate --strict` exige copiar todos los escenarios vigentes para que el archivado no los descarte.
- **Orden sin prueba dedicada.** El orden por registro no tiene una prueba específica; se verifica implícitamente en las pruebas de integración de listado y en la estabilidad de consultas repetidas de la TUI. Se documenta como límite de cobertura, no como requisito no cumplido.
- **`HEAD` sin prueba dedicada.** El guard del handler acepta `GET` y `HEAD`; no existe prueba explícita de `HEAD`. Queda documentado en el delta como límite.
- **Consolidaciones pendientes fuera de este cambio.** TUI-09 y US-17 carecen de delta propio; el propósito renombrado los referencia como pendientes para no dejar contradicciones en el spec consolidado.

## Migración

Ninguna: el listado reutiliza el esquema vigente (versión 14) sin agregar columnas ni tablas. La reversión consiste en retirar la ruta del cableado; no afecta datos.
