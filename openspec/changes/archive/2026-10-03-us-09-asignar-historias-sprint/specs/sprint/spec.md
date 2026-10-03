# Delta para Sprint

> Este delta documenta, de forma retroactiva, el requisito que ya fue fusionado directamente en `openspec/specs/sprint/spec.md` por el commit `f544916` (PR #78), sin pasar por la fase `sdd-spec`. El texto se copia textual del contenido canónico vigente (líneas 60–77), no se parafrasea.

## ADDED Requirements

### Requirement: Rechazar asignaciones a un Sprint cerrado (US-09)

El sistema MUST usar `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`, agregado por la migración canónica `000008`, como fuente de cierre. Sprints existentes y nuevos MUST quedar abiertos por defecto. La asignación MUST comprobar el cierre dentro de la misma transacción que valida Sprint, proyecto e historias y escribe el lote. MUST serializarse frente a cierres y asignaciones concurrentes mediante el bloqueo del Sprint; un cierre que confirma primero MUST impedir la asignación posterior. La creación de Sprints MUST conservar su gate limpio >=5; la asignación MUST exigir versión limpia >=8 y dependencia explícita.

#### Scenario: Sprint cerrado

- GIVEN un Sprint con `is_closed=true` y una selección de historias válidas del mismo proyecto
- WHEN se solicita `POST /projects/{project_id}/sprints/{sprint_id}/stories`
- THEN se responde `409 assignment_conflict`
- AND no se agrega ningún vínculo

#### Scenario: Cierre concurrente confirma primero

- GIVEN una transacción de cierre que bloquea el Sprint
- WHEN una asignación concurrente espera ese bloqueo y el cierre confirma
- THEN la asignación observa el Sprint cerrado y rechaza íntegramente el lote

La asignación no define un endpoint de cierre, tareas ni finalización, y no cambia el estado de las historias. `000007` almacena vínculos independientes; las historias permanecen en el Product Backlog. Revertir las migraciones US-09 con `down 2` desde v8 elimina asociaciones y cierre, no proyectos/historias/Sprints ni integrantes de v6; no se afirma seguridad de rollback productivo ni compatibilidad con v5/v6 alternativos desplegados desconocidos.
