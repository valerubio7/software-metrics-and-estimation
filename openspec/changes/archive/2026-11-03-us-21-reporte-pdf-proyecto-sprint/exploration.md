# Exploración: reporte PDF de proyecto o Sprint (US-21)

## Evidencia del requerimiento

- Issue #49: el reporte debe cubrir un proyecto o Sprint, identificar el ámbito, incluir historias planificadas y completadas, estimaciones, esfuerzo registrado, métricas y defectos, ser PDF y no modificar datos.
- `docs/assignment.md` (sección 9 y Sprint 4) repite el mismo alcance y exige exportar un informe del proyecto en PDF.
- US-19 definió las métricas y sus fórmulas; US-18 definió el listado de defectos; US-15 registró el esfuerzo; US-11 define la finalización por `sprint_stories.completed_at`.

## Estado del código relevante

- `internal/metrics` ya calcula todas las métricas en `domain.Calculate`; su lector PostgreSQL construye un `domain.Input` consistente en una transacción de solo lectura.
- `internal/defect` expone el listado por proyecto; los defectos guardan Sprint de detección y Sprint de resolución.
- `internal/effort` solo registra; no existe un lector de esfuerzo fuera del lector de métricas.
- `internal/story` lista el backlog por proyecto y las historias asignadas a un Sprint, pero no incluye finalización ni tareas.
- `internal/sprint` lista Sprints abiertos y el historial de cerrados; el historial incluye historias planificadas y `completed_at`.
- La composición opcional vive en `internal/api` con `HTTPDependencies` y `ResolveMigrationReadiness`; el arranque cablea módulos en `cmd/api/main.go`.
- No existe un módulo de reportes ni una dependencia PDF en `go.mod`. `internal/report` está vacío.

## Semántica que debe conservarse

- Proyecto: todas las historias del proyecto, incluidas las que no están en un Sprint; todas las tareas y todos los esfuerzos; todos los defectos; velocidad promedio de Sprints cerrados.
- Sprint: solo historias asignadas; solo tareas del Sprint; solo esfuerzo atribuible a tareas del Sprint; defectos detectados o resueltos en el Sprint; velocidad igual a puntos completados.
- Completada: `sprint_stories.completed_at` no nulo, nunca `stories.status`.
- Métricas: mismas fórmulas y cobertura de US-19; `null` no se convierte en cero.
- Defectos: detección por `sprint_id`, resolución por `resolution_sprint_id`; no inferir resolución desde `status`.

## Alternativas consideradas

1. **Componer el reporte con los endpoints existentes.** Descartada: obliga a varias lecturas no consistentes, no existe un lector de esfuerzo y el PDF podría mezclar estados distintos.
2. **Reutilizar el lector de métricas y un lector de detalles separado.** Descartada: dos snapshots pueden discrepar; el reporte debe ser consistente con sus propias métricas.
3. **Snapshot único de reporte y cálculo con el dominio de métricas.** Elegida: una transacción de solo lectura devuelve detalles y observaciones; el dominio arma el reporte y llama a `metrics/domain.Calculate`, sin duplicar fórmulas.
4. **Generar el PDF a mano.** Descartada: fuentes, tablas, paginación y codificación de acentos son propensas a errores; una biblioteca pure Go reduce el riesgo.
5. **Rutas con extensión frente a rutas REST sin extensión.** Elegida `/report.pdf`: expresa el formato solicitado y evita ambigüedad con futuros reportes de otro tipo.

## Riesgos iniciales

- La codificación de acentos y eñes en PDF con fuentes base exige un traductor a cp1252.
- Las tablas con textos largos deben paginar sin perder filas.
- El esfuerzo directo a historia no es atribuible a un Sprint; debe excluirse de la tabla y quedar en la cobertura, igual que en US-19.
- El reporte de Sprint debe incluir un defecto detectado en A y resuelto en B en ambos reportes sin duplicarlo dentro de uno.
- La dependencia PDF debe ser pure Go y no alterar el build ni las pruebas existentes.

## Decisiones abiertas para el diseño

- Forma exacta del snapshot y del renderizador.
- Contrato de errores y disponibilidad.
- Contenido y orden de las secciones del PDF.
- Estrategia de pruebas unitarias, de integración y de contrato HTTP.
