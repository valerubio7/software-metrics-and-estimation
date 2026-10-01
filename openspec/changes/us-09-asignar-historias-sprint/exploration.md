# Exploración: US-09 Asignar historias a un Sprint

## Conclusión ejecutiva y fuentes

El alcance confirmado de HU-09 (#37) es asociar historias ya existentes del Product Backlog a un Sprint existente del mismo proyecto, de modo que queden disponibles como trabajo planificado del Sprint. La operación puede recibir varias historias seleccionadas, pero debe ser atómica: si una es inválida, no se asigna ninguna. Las historias permanecen en el Product Backlog; la asignación agrega el vínculo al Sprint, no las retira ni inventa un nuevo estado.

En la decisión inicial, la validación del cierre se había condicionado a US-12 (#40). Esa decisión fue reemplazada después por autorización explícita del usuario: HU-09 incorpora `sprints.is_closed BOOLEAN NOT NULL DEFAULT false` como fuente de verdad y rechaza asignaciones cuando el valor es `true`. Este artefacto se reconcilia con la decisión final; no se depende de que US-12 esté implementada.

## Estado actual del contexto OpenSpec

- `openspec/config.yaml` establece Go, artefactos técnicos en español, TDD estricto y `go test ./...`.
- `openspec/specs/sprint/spec.md` especifica creación de Sprint asociado a un proyecto existente, sin consultar ni asignar historias. La asignación es trabajo separado de US-09.
- `openspec/specs/historia/spec.md` establece creación de historias en el Product Backlog y asociaciones a proyecto. El estado inicial descrito es `pendiente`; esta exploración no lo cambia ni lo equipara con asignación.
- No había cambio activo para esta historia ni artefactos de cambio existentes, según el estado SDD proporcionado.
- La rama `feat/us-09-asignar-historias-sprint` se creó desde `main` local en `3e26553`, según el contexto de sesión.

## Alcance confirmado

- Seleccionar un Sprint existente y una o varias historias existentes del Product Backlog del mismo proyecto.
- Asociar las historias seleccionadas al Sprint como trabajo planificado, conservándolas en el Product Backlog.
- Rechazar historias de otro proyecto y duplicados ya asociados al mismo Sprint.
- No permitir agregar historias a un Sprint con `is_closed = true`; HU-09 incorpora el campo con valor predeterminado `false`.
- Aplicar atomicidad al conjunto: ante cualquier selección inválida, no asociar ninguna historia de la solicitud.

## Fuera de alcance

No crear historias (US-05), crear Sprints (US-08), descomponer historias en tareas (US-10), registrar finalización (US-11) ni implementar la operación de cierre de Sprint (US-12). HU-09 sí define el indicador `is_closed` que necesita para impedir nuevas asignaciones; US-12 podrá implementar el cierre usando ese indicador. No se define aquí cómo consultar Sprints ni se asignan estados nuevos a las historias.

## Implicaciones y riesgos para fases posteriores

- La persistencia debe representar una asociación historia–Sprint sin borrar la relación historia–proyecto ni sacar la historia del Product Backlog.
- La pertenencia al mismo proyecto y la ausencia de duplicados deben verificarse para la selección completa; el fallo de una validación no debe dejar asignaciones parciales.
- La representación acordada para HU-09 es `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`; una migración reversible agrega el campo y la asignación consulta su valor transaccionalmente.
- La selección múltiple y la atomicidad deben reflejarse conjuntamente en el contrato de operación y en las pruebas; los detalles de API, almacenamiento y errores quedan para fases posteriores.
- El presupuesto máximo acordado es 400 líneas por PR; posteriormente el usuario eligió `auto-chain` con `feature-branch-chain`, sin excepción de tamaño. El plan vigente está en `tasks.md`.

## Estado de la operación SDD

El estado nativo recibido recomendó `sdd-new`, sin cambio activo y con todos los artefactos faltantes. No se proporcionó una herramienta/operación ejecutable `sdd-new` en esta sesión; por tanto no afirmo que se haya ejecutado. Se creó únicamente este artefacto de exploración permitido para la fase actual. No se escribió propuesta, especificación, diseño, tareas ni código fuente.
