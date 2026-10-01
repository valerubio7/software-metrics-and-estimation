# Exploración: US-09 Asignar historias a un Sprint

## Conclusión ejecutiva y fuentes

El alcance confirmado de HU-09 (#37) es asociar historias ya existentes del Product Backlog a un Sprint existente del mismo proyecto, de modo que queden disponibles como trabajo planificado del Sprint. La operación puede recibir varias historias seleccionadas, pero debe ser atómica: si una es inválida, no se asigna ninguna. Las historias permanecen en el Product Backlog; la asignación agrega el vínculo al Sprint, no las retira ni inventa un nuevo estado.

La validación de Sprint cerrado depende explícitamente de US-12 (#40). No hay que inventar ni anticipar una representación del estado cerrado. Las decisiones y criterios aquí reflejan contexto autorizado del usuario e issue; el esquema OpenSpec existente aporta las convenciones de historias y Sprint. No se consultó GitHub ni la red durante esta exploración.

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
- No permitir agregar historias a un Sprint cerrado; la comprobación depende de la capacidad/estado que defina US-12.
- Aplicar atomicidad al conjunto: ante cualquier selección inválida, no asociar ninguna historia de la solicitud.

## Fuera de alcance

No crear historias (US-05), crear Sprints (US-08), descomponer historias en tareas (US-10), registrar finalización (US-11), ni definir cierre o representación de estado del Sprint (US-12). No se define aquí cómo consultar Sprints ni se asignan estados nuevos a las historias.

## Implicaciones y riesgos para fases posteriores

- La persistencia debe representar una asociación historia–Sprint sin borrar la relación historia–proyecto ni sacar la historia del Product Backlog.
- La pertenencia al mismo proyecto y la ausencia de duplicados deben verificarse para la selección completa; el fallo de una validación no debe dejar asignaciones parciales.
- La regla de Sprint cerrado queda condicionada a US-12. Hasta que exista esa definición, no asumir nombre de campo, enum, endpoint o comportamiento técnico para determinar cierre.
- La selección múltiple y la atomicidad deben reflejarse conjuntamente en el contrato de operación y en las pruebas; los detalles de API, almacenamiento y errores quedan para fases posteriores.
- El presupuesto de revisión es 400 líneas y la estrategia de entrega confirmada es `single-pr`; cualquier riesgo sobre el presupuesto debe resolverse según el preflight antes de implementar.

## Estado de la operación SDD

El estado nativo recibido recomendó `sdd-new`, sin cambio activo y con todos los artefactos faltantes. No se proporcionó una herramienta/operación ejecutable `sdd-new` en esta sesión; por tanto no afirmo que se haya ejecutado. Se creó únicamente este artefacto de exploración permitido para la fase actual. No se escribió propuesta, especificación, diseño, tareas ni código fuente.
