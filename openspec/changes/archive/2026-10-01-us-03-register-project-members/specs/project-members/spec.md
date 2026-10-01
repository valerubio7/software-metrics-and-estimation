# Especificación de Integrantes de Proyecto

## Propósito

Definir el registro de uno o más integrantes asociados a un proyecto existente, sin incorporar operaciones de consulta o gestión fuera de ese registro.

## Requisitos

### Requirement: Registrar integrantes para un proyecto existente

El sistema MUST permitir que un cliente registre uno o más integrantes indicando el ID del proyecto. Cada integrante MUST incluir nombre completo y MAY incluir email. Si el lote es válido, el sistema MUST asociar todos sus integrantes al proyecto indicado.

#### Scenario: Registrar un integrante con email

- GIVEN un proyecto existente y un integrante con nombre completo y email
- WHEN el cliente solicita registrar el integrante para el ID de ese proyecto
- THEN el sistema registra el integrante asociado al proyecto

#### Scenario: Registrar integrantes con email opcional

- GIVEN un proyecto existente y un lote de integrantes con nombre completo, algunos sin email
- WHEN el cliente solicita registrar el lote
- THEN el sistema registra todos los integrantes asociados al proyecto
- AND la ausencia de email no impide el registro

### Requirement: Rechazar solicitudes de integrantes inválidas sin escrituras

El sistema MUST exigir un nombre completo para cada integrante. Si cualquier integrante del lote carece de nombre completo o contiene datos inválidos, el sistema MUST rechazar la solicitud, MUST explicar el motivo y MUST NOT registrar ningún integrante del lote.

#### Scenario: Rechazar un integrante sin nombre completo

- GIVEN un lote que contiene al menos un integrante sin nombre completo
- WHEN el cliente solicita registrar el lote
- THEN el sistema devuelve un error que identifica el motivo de validación
- AND no registra ningún integrante del lote

### Requirement: Registrar lotes de forma atómica

El sistema MUST tratar el registro de un lote como una operación todo-o-nada. Si el registro de cualquier integrante del lote falla, el sistema MUST NOT dejar integrantes de ese lote parcialmente registrados.

#### Scenario: No dejar registros parciales ante un fallo del lote

- GIVEN un lote válido cuya operación de registro falla antes de completarse
- WHEN el sistema procesa el lote
- THEN ningún integrante de ese lote queda registrado

### Requirement: Rechazar integrantes duplicados para el proyecto

El sistema MUST rechazar un lote que contenga integrantes duplicados entre sí o que incluya integrantes ya registrados para el mismo proyecto. El rechazo MUST NOT registrar ningún integrante del lote.

#### Scenario: Rechazar duplicados dentro del lote

- GIVEN un lote que contiene integrantes duplicados según el criterio de duplicidad aplicable
- WHEN el cliente solicita registrar el lote
- THEN el sistema rechaza la solicitud e informa el motivo
- AND no registra ningún integrante del lote

#### Scenario: Rechazar un integrante ya registrado para el proyecto

- GIVEN un integrante duplicado según el criterio de duplicidad aplicable ya está registrado para el proyecto
- WHEN el cliente solicita registrar un lote que contiene ese integrante
- THEN el sistema rechaza la solicitud e informa el motivo
- AND no registra ningún integrante del lote

### Requirement: Rechazar el registro para un proyecto inexistente

El sistema MUST devolver una respuesta HTTP 404 cuando el ID indicado no corresponda a un proyecto existente y MUST NOT registrar integrantes.

#### Scenario: Solicitar registro para un proyecto inexistente

- GIVEN un ID que no corresponde a ningún proyecto
- WHEN el cliente solicita registrar uno o más integrantes para ese ID
- THEN el sistema devuelve HTTP 404
- AND no registra ningún integrante

### Requirement: No ofrecer registro parcial ni listado como parte del registro

El sistema MUST aceptar o rechazar el lote completo y MUST NOT ofrecer resultados de éxito parcial como resultado de esta operación. Este requisito no añade un endpoint para listar o buscar proyectos o integrantes.

#### Scenario: Solicitar un lote con un integrante inválido

- GIVEN un lote con al menos un integrante inválido
- WHEN el cliente solicita registrar el lote
- THEN el sistema rechaza el lote completo
- AND no registra los integrantes válidos restantes
