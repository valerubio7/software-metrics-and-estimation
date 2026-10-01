# Exploration: US-04 — Consultar el estado de un proyecto

## Evidencia y decisiones confirmadas

US-04 corresponde a “Consultar el estado de un proyecto” (issue #32, docs/scrum/sprint-1.md); US-01 difirió explícitamente el estado de proyecto a US-04. El cuerpo de GitHub issue #32, consultado durante esta sesión, requiere consultar e identificar un proyecto, mostrar su estado registrado o determinado por reglas del sistema, informar si no existe y no modificar datos. Excluye explícitamente US-03. La taxonomía y reglas del estado no están definidas por la issue; para esta historia el usuario confirmó la regla derivada de fechas documentada abajo.

La decisión de producto confirmada es un estado calculado de solo lectura a partir de fechas de calendario (date-only):
- `planned` cuando la fecha actual es anterior a `start_date`.
- `active` cuando `start_date <= fecha actual <= planned_finish_date` (ambos límites inclusivos).
- `overdue` cuando la fecha actual es posterior a `planned_finish_date`.
- No existe estado `completed`: no se dispone de una fecha real de finalización.

La consulta debe identificar inequívocamente el proyecto; ID y nombre son adecuados. No se añade campo de ciclo de vida persistido, no hay operación de escritura y no hay dependencia de US-03. No se ha establecido requisito de zona horaria, así que no se inventa uno; la fuente/interpretación concreta de “fecha actual” queda como supuesto abierto para diseño.

## Comportamiento y arquitectura existentes

- El módulo de proyecto representa `ID`, `Name`, `StartDate` y `PlannedFinishDate` (`internal/project/domain/project.go`). No existe atributo ni consulta de estado.
- Ya están implementados `POST /projects` y `PUT /projects/{project_id}`. La actualización persiste los datos básicos del proyecto (`internal/api/api.go`, `internal/project/application/update_project.go`, `internal/project/infrastructure/postgres/repository.go`).
- No existe endpoint de lectura individual ni listado de proyectos. El identificador UUID se usa como parámetro de ruta en otras capacidades.
- La tabla de proyectos persiste nombre y fechas planificadas. La restricción vigente exige que planned finish no sea anterior al inicio. No hace falta una migración para el estado derivado.
- La spec vigente `openspec/specs/project/spec.md` cubre creación y modificación; US-04 agrega lectura sin alterar esas capacidades.

## Alcance técnico probable

La propuesta debe cubrir consulta HTTP identificada por proyecto, lectura desde el repositorio, cálculo puro de estado con los límites indicados y respuesta que incluya como mínimo ID, nombre y estado. Debe definir una respuesta de proyecto inexistente. Las áreas afectadas previsibles son dominio/aplicación de `internal/project`, transporte HTTP, composición de rutas, puerto/repositorio PostgreSQL, pruebas y spec/documentación. No se debe asumir un endpoint concreto hasta especificarlo. Sin nuevas columnas ni operaciones de escritura.

## Riesgos y supuestos abiertos

- Aún debe fijarse en diseño cómo obtiene/inyecta el sistema la fecha actual para que el cálculo sea determinista y testeable. No hay evidencia para imponer zona horaria o regla de conversión adicional.
- No se establece aquí el código/forma exacta de error de proyecto inexistente ni el nombre definitivo del endpoint; la consulta debe reportar inequívocamente la ausencia.
- Fechas date-only tienen semántica inclusiva confirmada; convertirlas a instantes o desplazar límites podría causar errores de un día.
- La ausencia de `completed` es intencional, no equivale a inferir finalización por fecha planificada.
- Una especificación accidental de US-03, estados de historias, miembros o sprints ampliaría el alcance contra la decisión confirmada.

## Revisión de alcance

Referencia de revisión: 400 líneas modificadas. Este cambio se limita a consulta derivada y no introduce persistencia de estado ni mutaciones. Si una estimación posterior excediera el umbral, se debe resolver por la autoridad de entrega; no es autorización implícita para excederlo.

## Skill resolution

No se inyectaron rutas de skill de fase/proyecto. Se utilizó fallback al skill general `gentle-ai`; no se encontró una skill específica de propuesta SDD en las ubicaciones provistas. `skill_resolution: fallback-path`.
