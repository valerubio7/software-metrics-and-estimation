# Especificación de Historia

## Purpose

Registrar una historia en el Product Backlog de un proyecto existente, con criterios de aceptación y sin estimación inicial.

## Requirements

### Requirement: Crear una historia asociada a un proyecto conocido

El sistema MUST ofrecer una operación de creación que reciba el identificador de un proyecto conocido y los datos de la historia, sin requerir listado de proyectos ni una interfaz de selección. Ante una creación válida, MUST registrar exactamente una historia vinculada a ese proyecto y devolver su identificador generado junto con los datos de la historia creada.

#### Scenario: Registrar una historia en un proyecto existente

- GIVEN un proyecto existente cuyo identificador conoce quien solicita la creación
- AND se proporcionan título, descripción, prioridad `alta`, `media` o `baja` y una lista válida de criterios de aceptación
- WHEN se solicita crear la historia para ese identificador
- THEN el sistema registra exactamente una historia asociada a ese proyecto en su Product Backlog
- AND devuelve el identificador generado y los datos registrados de la historia

### Requirement: Validar los datos obligatorios de la historia

El sistema MUST exigir título y descripción no vacíos ni compuestos solo por espacios, una prioridad elegida entre `alta`, `media` y `baja`, y una lista de criterios de aceptación no vacía sin entradas vacías ni compuestas solo por espacios. MUST rechazar la creación sin persistir ninguna historia si falta o es inválido alguno de esos datos.

#### Scenario: Rechazar un campo obligatorio ausente o en blanco

- GIVEN un proyecto existente y una solicitud donde falta el título, la descripción o la prioridad, o donde alguno de esos campos está en blanco
- WHEN se solicita crear la historia
- THEN el sistema comunica el dato inválido
- AND no persiste ninguna historia

#### Scenario: Rechazar una prioridad no permitida

- GIVEN un proyecto existente y una solicitud cuya prioridad no es `alta`, `media` ni `baja`
- WHEN se solicita crear la historia
- THEN el sistema comunica que la prioridad es inválida
- AND no persiste ninguna historia

#### Scenario: Rechazar criterios de aceptación inválidos

- GIVEN un proyecto existente y una solicitud sin lista de criterios, con lista vacía o con al menos una entrada vacía o compuesta solo por espacios
- WHEN se solicita crear la historia
- THEN el sistema comunica que los criterios de aceptación son inválidos
- AND no persiste ninguna historia

### Requirement: Inicializar la historia sin estimación

El sistema MUST asignar `pendiente` como estado inicial y `null` como Story Points iniciales, nunca cero. La creación MUST conservar la prioridad elegida y cada criterio de aceptación proporcionado; MUST NOT aceptar una estimación de Story Points como parte de esta operación. La estimación corresponde a US-14.

#### Scenario: Crear una historia pendiente de estimación

- GIVEN un proyecto existente y datos válidos con prioridad `media` y dos criterios de aceptación
- WHEN se crea la historia sin estimación
- THEN la historia registrada y el resultado de creación contienen prioridad `media`, estado `pendiente` y Story Points `null`
- AND se conservan los dos criterios de aceptación proporcionados

#### Scenario: No admitir una estimación inicial

- GIVEN un proyecto existente y datos de historia válidos que incluyen una estimación inicial de Story Points, incluso cero
- WHEN se solicita crear la historia
- THEN el sistema rechaza la estimación inicial
- AND no persiste ninguna historia estimada por esta operación

### Requirement: Impedir historias sin proyecto existente

El sistema MUST rechazar la creación para un identificador de proyecto inexistente y MUST preservar la asociación de toda historia persistida con un proyecto existente. Si la asociación no puede asegurarse, incluso por un fallo o cambio concurrente durante la creación, MUST rechazar la operación sin persistir una historia huérfana. El rechazo MUST ser observable sin divulgar detalles internos.

#### Scenario: Rechazar un proyecto inexistente

- GIVEN un identificador de proyecto que no corresponde a ningún proyecto existente y datos de historia válidos
- WHEN se solicita crear la historia
- THEN el sistema rechaza la creación indicando que el proyecto no existe
- AND no persiste ninguna historia

#### Scenario: Fallar sin dejar historias huérfanas

- GIVEN un proyecto cuyo vínculo deja de poder asegurarse durante la creación por un fallo o cambio concurrente
- WHEN se intenta registrar una historia para ese proyecto
- THEN el sistema rechaza la operación sin divulgar detalles internos
- AND no queda persistida ninguna historia sin un proyecto existente
