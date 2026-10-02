# Propuesta: Asignar historias a un Sprint (HU-09)

## Intención

Permitir planificar en un Sprint existente una o varias historias ya presentes en el Product Backlog. La asignación agrega el vínculo de trabajo planificado sin eliminar las historias del backlog ni cambiar o inventar su estado.

## Alcance

- Asociar una selección de una o más historias existentes con un Sprint existente del mismo proyecto.
- Validar la selección completa antes de modificarla: si alguna historia no es válida, no se asigna ninguna.
- Rechazar historias que pertenezcan a otro proyecto y las ya asociadas al Sprint seleccionado.
- No permitir asignaciones a Sprints cerrados; por decisión autorizada para HU-09, `sprints.is_closed BOOLEAN NOT NULL DEFAULT false` es la fuente de verdad.

### Fuera de alcance

Crear historias o Sprints, descomponer historias en tareas, registrar su finalización, definir el cierre del Sprint o cambiar los estados de las historias. Consultas de Sprints y detalles de API, persistencia y formato de errores se reservan para fases posteriores.

## Áreas afectadas

- **Dominio de historias y Sprints:** expresar la asociación entre una historia del Product Backlog y un Sprint de su proyecto.
- **Validación de la operación:** validar pertenencia al proyecto, duplicados y elegibilidad del Sprint para todo el lote antes de persistir cualquier asociación.
- **Persistencia y pruebas:** conservar las relaciones existentes de historia con el Product Backlog y el proyecto; verificar selección múltiple, atomicidad y rechazos. Los mecanismos concretos se determinarán en las fases siguientes.

## Reglas y supuestos confirmados

- Una operación puede seleccionar varias historias.
- La operación es atómica como conjunto: cualquier elemento inválido implica cero asignaciones de esa solicitud.
- Las historias permanecen en el Product Backlog; el Sprint queda vinculado como trabajo planificado.
- La regla sobre Sprints cerrados usa el campo `is_closed` acordado por el usuario para HU-09.
- HU-09 corresponde al issue GitHub #37; se conservan los criterios y exclusiones confirmados para esta historia.

## Dependencias

La decisión del usuario para HU-09 reemplaza la dependencia previa de US-12 (#40): agregar `sprints.is_closed BOOLEAN NOT NULL DEFAULT false` mediante migración y rechazar la asignación cuando sea `true`. Los Sprints existentes quedan abiertos por el valor predeterminado.

## Riesgos y mitigaciones

- **Asignaciones parciales ante errores:** validar todas las historias antes de persistir y asegurar atomicidad transaccional; probar que cualquier rechazo deja el lote intacto.
- **Confundir planificación con cambio de estado o salida del backlog:** modelar la asignación como vínculo adicional y verificar que ambas condiciones existentes se preserven.
- **Consistencia del cierre:** la aplicación consulta `is_closed` dentro de la transacción de asignación, antes de persistir el lote.
- **Duplicados o historias de otro proyecto:** rechazar la solicitud completa cuando se detecte cualquiera, sin asignar los demás elementos.

## Reversión

Si la capacidad causa problemas, deshabilitar o revertir la operación de asignación y retirar los vínculos creados por ella, preservando historias, su pertenencia al Product Backlog y sus relaciones con el proyecto. La estrategia concreta para retirar vínculos deberá respetar el mecanismo de persistencia que se defina; no eliminar historias como efecto de rollback.

## Criterios de éxito

1. Se pueden asociar una o varias historias existentes del mismo proyecto a un Sprint existente.
2. Las historias asignadas siguen en el Product Backlog y no reciben un estado nuevo por esta operación.
3. Si la solicitud contiene una historia de otro proyecto, un duplicado o cualquier otra selección inválida, no se asigna ninguna historia de esa solicitud.
4. No se asignan historias a un Sprint con `is_closed = true`.
5. Las pruebas cubren tanto asignaciones válidas múltiples como rechazo atómico de lotes inválidos.

## Ronda de preguntas de propuesta

La ronda de preguntas de producto fue completada por el usuario antes de esta fase. Se aprobaron las recomendaciones con las reglas y límites anteriores; no se reabren decisiones confirmadas en esta propuesta.
