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

### Requisito: Cerrar un Sprint abierto

El sistema MUST permitir cerrar un Sprint abierto del proyecto indicado mediante `POST /projects/{project_id}/sprints/{sprint_id}/closure`, sin requerir un cuerpo. El éxito MUST responder `200` con `project_id`, `sprint_id` e `is_closed=true`.

#### Escenario: Cierre exitoso

- WHEN se solicita el cierre de un Sprint abierto perteneciente al proyecto indicado
- THEN el sistema persiste el estado cerrado y responde `200` con ambos identificadores e `is_closed=true`

#### Escenario: Ignorar el cuerpo de la solicitud

- WHEN se solicita el cierre de un Sprint abierto enviando contenido en el cuerpo
- THEN el sistema cierra el Sprint sin exigir ni interpretar ese contenido

### Requisito: Validar identificadores y pertenencia antes del cierre

El sistema MUST validar los UUID antes de consultar el almacenamiento. Debe responder `422 validation_failed` por identificadores inválidos, `404 project_not_found` si el proyecto no existe y `404 sprint_not_found` si el Sprint no existe o pertenece a otro proyecto. Los rechazos MUST NOT modificar datos.

#### Escenario: Identificadores inválidos

- WHEN uno o ambos identificadores de ruta no son UUID válidos
- THEN el sistema responde `422 validation_failed` con los campos inválidos, sin consultar ni modificar el almacenamiento

#### Escenario: Proyecto inexistente

- WHEN el proyecto indicado no existe
- THEN el sistema responde `404 project_not_found` y no modifica datos

#### Escenario: Sprint inexistente o de otro proyecto

- WHEN el Sprint indicado no existe o no pertenece al proyecto de la ruta
- THEN el sistema responde `404 sprint_not_found` y no modifica datos

### Requisito: Rechazar el cierre repetido

El sistema MUST responder `409 sprint_closed` al solicitar el cierre de un Sprint ya cerrado y MUST preservar su estado y datos relacionados.

#### Escenario: Sprint ya cerrado

- WHEN se solicita cerrar nuevamente un Sprint cerrado
- THEN el sistema responde `409 sprint_closed` y conserva todos sus datos sin cambios

### Requisito: Conservar la información histórica

Cerrar un Sprint MUST NOT eliminar ni alterar sus historias planificadas o completadas, asociaciones, tareas, esfuerzos o defectos. El Sprint cerrado MUST seguir disponible para consultas posteriores y el cierre MUST NOT modificar los datos de las historias.

#### Escenario: Preservar resultados de la iteración

- WHEN se cierra un Sprint con historias, finalizaciones y datos de trabajo relacionados
- THEN todos esos registros y los datos de las historias permanecen sin cambios

### Requisito: Serializar el cierre frente a asignaciones concurrentes

El cierre y la asignación MUST serializarse para el mismo Sprint. Si el cierre confirma primero, una asignación posterior MUST rechazar íntegramente el lote según el conflicto definido por US-09. Si la asignación confirma primero, el cierre MUST conservar sus asociaciones.

#### Escenario: El cierre confirma antes que la asignación

- WHEN una asignación concurrente obtiene el bloqueo después de confirmar el cierre
- THEN la asignación observa el Sprint cerrado, rechaza el lote y no crea asociaciones

#### Escenario: La asignación confirma antes que el cierre

- WHEN la asignación confirma antes de que el cierre obtenga el bloqueo
- THEN el cierre conserva todas las asociaciones creadas

### Requisito: Habilitar el cierre solo con el esquema requerido

La ruta MUST estar disponible únicamente con esquema limpio en versión 8 o superior. Con versión inferior, estado `dirty` o error al resolver el esquema, la ruta MUST responder `404` sin alterar la disponibilidad de otras operaciones.

#### Escenario: Esquema compatible

- WHEN la API inicia con esquema limpio versión 8 o superior
- THEN la ruta de cierre queda disponible

#### Escenario: Esquema incompatible

- WHEN la API inicia con versión inferior a 8, estado `dirty` o error de resolución
- THEN la ruta no queda registrada y responde `404`

### Requisito: Manejar métodos no admitidos y fallos de almacenamiento

La ruta MUST aceptar únicamente `POST` y responder `405` para otros métodos. Un fallo inesperado de almacenamiento MUST responder `500 internal_error` con mensaje genérico, sin efectos parciales ni detalles internos.

#### Escenario: Método no admitido

- WHEN se usa un método distinto de `POST` en la ruta de cierre
- THEN el sistema responde `405` y no modifica datos

#### Escenario: Fallo de almacenamiento

- WHEN ocurre un error inesperado al persistir el cierre
- THEN el sistema responde `500 internal_error` genérico y no deja efectos parciales
