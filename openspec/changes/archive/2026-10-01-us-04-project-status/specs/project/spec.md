# Delta for Proyecto

## REQUISITOS AÑADIDOS

### Requirement: Consultar y derivar el estado actual de un proyecto

El sistema MUST permitir consultar un proyecto identificado y devolver una respuesta que identifique inequívocamente el proyecto (incluyendo ID y nombre) junto con exactamente uno de estos estados derivados de fechas date-only: `planned`, `active` o `overdue`. La fecha de referencia MUST ser la fecha actual de calendario. El estado MUST ser `planned` si la fecha actual es anterior a `start_date`, `active` si `start_date <= fecha actual <= planned_finish_date`, y `overdue` si la fecha actual es posterior a `planned_finish_date`. El sistema MUST NOT ofrecer `completed` como resultado de esta capacidad.

#### Scenario: Proyecto aún no iniciado

- GIVEN un proyecto cuya `start_date` es posterior a la fecha actual
- WHEN el cliente consulta ese proyecto
- THEN la respuesta identifica el proyecto y su estado es `planned`

#### Scenario: Día de inicio incluido en el período activo

- GIVEN un proyecto cuya `start_date` coincide con la fecha actual y cuya fecha planificada de finalización no es anterior a ella
- WHEN el cliente consulta ese proyecto
- THEN la respuesta identifica el proyecto y su estado es `active`

#### Scenario: Día de finalización planificada incluido en el período activo

- GIVEN un proyecto cuya fecha actual coincide con `planned_finish_date` y no es anterior a `start_date`
- WHEN el cliente consulta ese proyecto
- THEN la respuesta identifica el proyecto y su estado es `active`

#### Scenario: Fecha actual posterior a la finalización planificada

- GIVEN un proyecto cuya `planned_finish_date` es anterior a la fecha actual
- WHEN el cliente consulta ese proyecto
- THEN la respuesta identifica el proyecto y su estado es `overdue`, no `completed`

### Requirement: Informar la ausencia del proyecto consultado

El sistema MUST informar inequívocamente cuando el identificador consultado no corresponda a un proyecto existente y MUST NOT presentar un resultado de estado inventado ni un éxito vacío.

#### Scenario: Consultar un proyecto inexistente

- GIVEN que el identificador solicitado no corresponde a ningún proyecto existente
- WHEN el cliente consulta ese identificador
- THEN el sistema devuelve información explícita de que el proyecto no existe
- AND no devuelve un estado de proyecto

### Requirement: La consulta de estado no modifica datos del proyecto

La consulta MUST ser de solo lectura: MUST NOT persistir un estado de ciclo de vida, modificar los datos del proyecto ni requerir o invocar US-03. El estado se deriva para la respuesta y MUST NOT convertirse en un campo persistido por esta capacidad.

#### Scenario: Consultar un proyecto sin mutar sus datos

- GIVEN un proyecto existente con sus datos básicos almacenados
- WHEN el cliente consulta el estado del proyecto
- THEN el sistema devuelve su identidad y estado derivado
- AND los datos almacenados del proyecto permanecen sin cambios
- AND la consulta no requiere datos ni comportamiento de US-03
