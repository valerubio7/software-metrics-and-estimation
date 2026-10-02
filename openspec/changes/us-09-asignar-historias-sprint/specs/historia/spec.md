# Delta para Historia

## Requisitos añadidos

### Requisito: Asignar en lote historias existentes a un Sprint de su proyecto

El sistema MUST permitir asociar una o más historias existentes del Product Backlog con un Sprint existente del mismo proyecto como trabajo planificado. La asignación MUST conservar la pertenencia de cada historia al Product Backlog y al proyecto, así como su estado actual; MUST NOT crear historias ni modificar sus estados. La solicitud MUST validarse como un lote completo antes de efectuar asociaciones: si cualquier elemento o el Sprint no cumple las condiciones de elegibilidad, MUST rechazarse la operación y no asignarse ninguna historia de esa solicitud. MUST rechazarse una historia que pertenezca a otro proyecto, que ya esté asociada al Sprint seleccionado o que aparezca más de una vez en la selección. La asignación a un Sprint con `is_closed = true` MUST rechazarse atómicamente. La columna `is_closed BOOLEAN NOT NULL DEFAULT false` es la fuente de verdad para este requisito.

#### Escenario: Asignar varias historias elegibles

- GIVEN un Sprint existente y varias historias existentes del mismo proyecto que siguen en el Product Backlog y no están ya asociadas a ese Sprint
- WHEN se solicita asignarlas juntas al Sprint
- THEN el sistema asocia todas las historias seleccionadas al Sprint como trabajo planificado
- AND todas permanecen en el Product Backlog, en su proyecto y con sus estados sin cambios

#### Escenario: Asignar una sola historia elegible

- GIVEN un Sprint existente y una historia existente del mismo proyecto que no está ya asociada a ese Sprint
- WHEN se solicita asignar la historia al Sprint
- THEN el sistema la asocia al Sprint como trabajo planificado
- AND la historia permanece en el Product Backlog y su estado no cambia

#### Escenario: Rechazar atómicamente una selección con historia de otro proyecto

- GIVEN un Sprint y una selección que incluye historias del proyecto del Sprint y al menos una historia de otro proyecto
- WHEN se solicita la asignación del lote
- THEN el sistema rechaza la solicitud
- AND no asocia al Sprint ninguna historia de la selección

#### Escenario: Rechazar atómicamente una selección con asociación duplicada

- GIVEN un Sprint y una selección que incluye una historia ya asociada a ese Sprint junto con otra historia elegible
- WHEN se solicita la asignación del lote
- THEN el sistema rechaza la solicitud
- AND no asocia ninguna historia de la selección que todavía no estuviera asociada

#### Escenario: Rechazar una historia repetida en la selección

- GIVEN una selección que contiene más de una vez la misma historia
- WHEN se solicita asignar la selección a un Sprint
- THEN el sistema rechaza la solicitud
- AND no asocia ninguna historia de esa solicitud

#### Escenario: Rechazar atómicamente una selección con historia inexistente

- GIVEN una selección con al menos un identificador que no corresponde a una historia existente y otras historias elegibles
- WHEN se solicita la asignación del lote
- THEN el sistema rechaza la solicitud
- AND no asocia ninguna historia de la selección

#### Escenario: Rechazar la asignación a un Sprint cerrado

- GIVEN un Sprint con `is_closed = true` y una o más historias elegibles
- WHEN se solicita asignar las historias a ese Sprint
- THEN el sistema rechaza la solicitud
- AND no asocia ninguna historia
- AND determina el cierre leyendo `is_closed`
