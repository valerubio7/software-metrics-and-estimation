# Delta de especificación

## ADDED Requirements

### Requirement: Cerrar un Sprint abierto
El sistema MUST permitir cerrar un Sprint abierto del proyecto indicado mediante `POST /projects/{project_id}/sprints/{sprint_id}/closure`, sin requerir un cuerpo. El éxito MUST responder `200` con `project_id`, `sprint_id` e `is_closed=true`.

#### Scenario: Cierre exitoso
- **WHEN** se solicita el cierre de un Sprint abierto perteneciente al proyecto indicado
- **THEN** el sistema persiste el estado cerrado y responde `200` con ambos identificadores e `is_closed=true`

#### Scenario: Ignorar el cuerpo de la solicitud
- **WHEN** se solicita el cierre de un Sprint abierto enviando contenido en el cuerpo
- **THEN** el sistema cierra el Sprint sin exigir ni interpretar ese contenido

### Requirement: Validar identificadores y pertenencia antes del cierre
El sistema MUST validar los UUID antes de consultar el almacenamiento. Debe responder `422 validation_failed` por identificadores inválidos, `404 project_not_found` si el proyecto no existe y `404 sprint_not_found` si el Sprint no existe o pertenece a otro proyecto. Los rechazos MUST NOT modificar datos.

#### Scenario: Identificadores inválidos
- **WHEN** uno o ambos identificadores de ruta no son UUID válidos
- **THEN** el sistema responde `422 validation_failed` con los campos inválidos, sin consultar ni modificar el almacenamiento

#### Scenario: Proyecto inexistente
- **WHEN** el proyecto indicado no existe
- **THEN** el sistema responde `404 project_not_found` y no modifica datos

#### Scenario: Sprint inexistente o de otro proyecto
- **WHEN** el Sprint indicado no existe o no pertenece al proyecto de la ruta
- **THEN** el sistema responde `404 sprint_not_found` y no modifica datos

### Requirement: Rechazar el cierre repetido
El sistema MUST responder `409 sprint_closed` al solicitar el cierre de un Sprint ya cerrado y MUST preservar su estado y datos relacionados.

#### Scenario: Sprint ya cerrado
- **WHEN** se solicita cerrar nuevamente un Sprint cerrado
- **THEN** el sistema responde `409 sprint_closed` y conserva todos sus datos sin cambios

### Requirement: Conservar la información histórica
Cerrar un Sprint MUST NOT eliminar ni alterar sus historias planificadas o completadas, asociaciones, tareas, esfuerzos o defectos. El Sprint cerrado MUST seguir disponible para consultas posteriores y el cierre MUST NOT modificar los datos de las historias.

#### Scenario: Preservar resultados de la iteración
- **WHEN** se cierra un Sprint con historias, finalizaciones y datos de trabajo relacionados
- **THEN** todos esos registros y los datos de las historias permanecen sin cambios

### Requirement: Serializar el cierre frente a asignaciones concurrentes
El cierre y la asignación MUST serializarse para el mismo Sprint. Si el cierre confirma primero, una asignación posterior MUST rechazar íntegramente el lote según el conflicto definido por US-09. Si la asignación confirma primero, el cierre MUST conservar sus asociaciones.

#### Scenario: El cierre confirma antes que la asignación
- **WHEN** una asignación concurrente obtiene el bloqueo después de confirmar el cierre
- **THEN** la asignación observa el Sprint cerrado, rechaza el lote y no crea asociaciones

#### Scenario: La asignación confirma antes que el cierre
- **WHEN** la asignación confirma antes de que el cierre obtenga el bloqueo
- **THEN** el cierre conserva todas las asociaciones creadas

### Requirement: Habilitar el cierre solo con el esquema requerido
La ruta MUST estar disponible únicamente con esquema limpio en versión 8 o superior. Con versión inferior, estado `dirty` o error al resolver el esquema, la ruta MUST responder `404` sin alterar la disponibilidad de otras operaciones.

#### Scenario: Esquema compatible
- **WHEN** la API inicia con esquema limpio versión 8 o superior
- **THEN** la ruta de cierre queda disponible

#### Scenario: Esquema incompatible
- **WHEN** la API inicia con versión inferior a 8, estado `dirty` o error de resolución
- **THEN** la ruta no queda registrada y responde `404`

### Requirement: Manejar métodos no admitidos y fallos de almacenamiento
La ruta MUST aceptar únicamente `POST` y responder `405` para otros métodos. Un fallo inesperado de almacenamiento MUST responder `500 internal_error` con mensaje genérico, sin efectos parciales ni detalles internos.

#### Scenario: Método no admitido
- **WHEN** se usa un método distinto de `POST` en la ruta de cierre
- **THEN** el sistema responde `405` y no modifica datos

#### Scenario: Fallo de almacenamiento
- **WHEN** ocurre un error inesperado al persistir el cierre
- **THEN** el sistema responde `500 internal_error` genérico y no deja efectos parciales
