# Delta para Sprint

## Requisitos añadidos

### Requisito: Aceptar asignaciones planificadas solo en un Sprint existente del mismo proyecto

El sistema MUST permitir que una operación de asignación vincule historias existentes únicamente a un Sprint existente y perteneciente al mismo proyecto que esas historias. MUST rechazar una solicitud cuyo Sprint no exista o cuya selección incluya historias de otro proyecto, sin efectuar ninguna asociación del lote. Un Sprint MUST tener `is_closed BOOLEAN NOT NULL DEFAULT false`, y el sistema MUST rechazar atómicamente cualquier asignación a un Sprint con `is_closed = true`.

#### Escenario: Rechazar la asignación a un Sprint inexistente

- GIVEN historias existentes y un identificador que no corresponde a un Sprint existente
- WHEN se solicita asignar las historias a ese Sprint
- THEN el sistema rechaza la solicitud
- AND no asocia ninguna historia

#### Escenario: Rechazar atómicamente un lote de otro proyecto

- GIVEN un Sprint existente y una selección que contiene al menos una historia perteneciente a otro proyecto
- WHEN se solicita asignar la selección al Sprint
- THEN el sistema rechaza la solicitud
- AND no asocia ninguna historia del lote

#### Escenario: Rechazar la asignación a un Sprint cerrado

- GIVEN un Sprint existente con `is_closed = true` y una o más historias elegibles
- WHEN se solicita asignar las historias al Sprint
- THEN el sistema rechaza la solicitud
- AND no asocia ninguna historia
