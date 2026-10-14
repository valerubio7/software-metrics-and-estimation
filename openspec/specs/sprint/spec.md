# Especificación de Sprint

## Propósito

Definir la creación de un Sprint asociado a un proyecto existente, registrar su Sprint Goal e identificarlo mediante un UUID generado por el servidor. El Sprint queda disponible para futuras asignaciones de historias; su creación no requiere la consulta ni la asignación de historias.

## Requisitos

### Requisito: Crear un Sprint para un proyecto existente

El sistema MUST ofrecer `POST /projects/{project_id}/sprints` para crear un Sprint asociado al proyecto identificado por `project_id`. Ante una solicitud válida, MUST persistir exactamente un Sprint con ese proyecto y el Sprint Goal proporcionado, generar su UUID en el servidor y devolver el identificador y los datos registrados del Sprint. La operación MUST NOT requerir ni realizar la asignación de historias.

#### Escenario: Crear un Sprint válido

- GIVEN un proyecto existente cuyo identificador conoce quien solicita la creación
- AND una solicitud que proporciona un Sprint Goal
- WHEN el cliente envía `POST /projects/{project_id}/sprints` para ese proyecto
- THEN el sistema persiste exactamente un Sprint asociado al proyecto
- AND registra el Sprint Goal proporcionado
- AND devuelve una respuesta de creación exitosa con el UUID generado por el servidor y los datos registrados

#### Escenario: Crear un Sprint sin depender de historias del Product Backlog

- GIVEN un proyecto existente, tenga o no historias disponibles para asignación
- AND una solicitud con el Sprint Goal
- WHEN el cliente solicita crear un Sprint para ese proyecto
- THEN el sistema crea el Sprint sin exigir que existan historias en el Product Backlog
- AND el Sprint no contiene historias asignadas por esta operación

### Requisito: Validar la información obligatoria antes de persistir

El sistema MUST exigir el Sprint Goal para crear un Sprint y MUST rechazar una solicitud a la que le falte ese dato antes de persistir. El sistema MUST informar que falta el Sprint Goal y MUST NOT persistir un Sprint cuando se rechace la solicitud por este motivo. Este requisito no establece restricciones adicionales sobre el contenido o la longitud del Sprint Goal.

#### Escenario: Rechazar una solicitud sin Sprint Goal

- GIVEN un proyecto existente y una solicitud de creación que no proporciona Sprint Goal
- WHEN el cliente solicita crear un Sprint para ese proyecto
- THEN el sistema rechaza la solicitud e informa que falta el Sprint Goal
- AND no persiste ningún Sprint

### Requisito: Asociar el Sprint únicamente a un proyecto existente

El sistema MUST rechazar la creación cuando `project_id` no identifica un UUID válido o no corresponde a un proyecto existente. Ante el rechazo, MUST informar que el identificador o el proyecto no es válido y MUST NOT persistir un Sprint.

#### Escenario: Rechazar un identificador de proyecto con formato inválido

- GIVEN una solicitud de creación con un `project_id` que no tiene formato UUID válido
- WHEN el cliente envía la solicitud a `POST /projects/{project_id}/sprints`
- THEN el sistema rechaza la solicitud indicando que el identificador del proyecto no es válido
- AND no persiste ningún Sprint

#### Escenario: Rechazar un proyecto inexistente

- GIVEN un `project_id` con formato UUID válido que no corresponde a un proyecto existente
- AND una solicitud con Sprint Goal
- WHEN el cliente solicita crear un Sprint para ese proyecto
- THEN el sistema rechaza la creación indicando que el proyecto no existe
- AND no persiste ningún Sprint

### Requisito: Rechazar asignaciones a un Sprint cerrado (US-09)

El sistema MUST usar `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`, agregado por la migración canónica `000008`, como fuente de cierre. Sprints existentes y nuevos MUST quedar abiertos por defecto. La asignación MUST comprobar el cierre dentro de la misma transacción que valida Sprint, proyecto e historias y escribe el lote. MUST serializarse frente a cierres y asignaciones concurrentes mediante el bloqueo del Sprint; un cierre que confirma primero MUST impedir la asignación posterior. La creación de Sprints MUST conservar su gate limpio >=5; la asignación MUST exigir versión limpia >=8 y dependencia explícita.

#### Escenario: Sprint cerrado

- GIVEN un Sprint con `is_closed=true` y una selección de historias válidas del mismo proyecto
- WHEN se solicita `POST /projects/{project_id}/sprints/{sprint_id}/stories`
- THEN se responde `409 assignment_conflict`
- AND no se agrega ningún vínculo

#### Escenario: Cierre concurrente confirma primero

- GIVEN una transacción de cierre que bloquea el Sprint
- WHEN una asignación concurrente espera ese bloqueo y el cierre confirma
- THEN la asignación observa el Sprint cerrado y rechaza íntegramente el lote

La asignación no define un endpoint de cierre, tareas ni finalización, y no cambia el estado de las historias. `000007` almacena vínculos independientes; las historias permanecen en el Product Backlog. Revertir las migraciones US-09 con `down 2` desde v8 elimina asociaciones y cierre, no proyectos/historias/Sprints ni integrantes de v6; no se afirma seguridad de rollback productivo ni compatibilidad con v5/v6 alternativos desplegados desconocidos.
