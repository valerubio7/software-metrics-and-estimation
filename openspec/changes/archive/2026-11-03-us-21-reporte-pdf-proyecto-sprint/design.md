# Diseño del reporte PDF de proyecto o Sprint (US-21)

## Contexto

Véanse la motivación en [proposal.md](proposal.md) y el contrato en [specs/reporte/spec.md](specs/reporte/spec.md). La API ya compone módulos opcionales mediante `HTTPDependencies` y resuelve disponibilidad con `ResolveMigrationReadiness`. Las métricas de US-19 se calculan en `internal/metrics/domain`; el esfuerzo, la finalización y los defectos viven en tablas relacionadas con `project_id`. No existe un lector que reúna todos esos datos en un solo snapshot.

## Objetivos y no objetivos

**Objetivos:** generar un PDF legible y completo para proyecto o Sprint, reutilizar las fórmulas de US-19, mantener una única lectura consistente y no modificar datos.

**No objetivos:** implementar el Dashboard o la TUI, calcular métricas nuevas, persistir reportes, agregar migraciones, cambiar los contratos de escritura o incluir gráficos.

## Decisiones

1. **Módulo `internal/report` separado.** Sigue la estructura dominio/aplicación/infraestructura/transporte del resto del repositorio. El caso de uso valida UUID, lee un snapshot, construye el reporte y lo entrega a un renderizador. Alternativa descartada: componer la respuesta con varios endpoints, que produciría lecturas inconsistentes y no cubriría el esfuerzo.

2. **Snapshot único en una transacción de solo lectura.** El lector PostgreSQL abre una transacción `RepeatableRead` y `ReadOnly`, verifica proyecto y Sprint, y lee proyecto, Sprint, historias, tareas, esfuerzo, defectos y Sprints cerrados. Si algo falla, no hay reporte parcial. El dominio deriva las métricas del mismo snapshot con `metricsdomain.Calculate`, de modo que el PDF y la API de métricas coinciden sobre el mismo estado.

3. **El dominio deriva las métricas, no las recalcula.** `report/domain.Build` arma un `metricsdomain.Input` a partir de las historias, tareas, esfuerzos, defectos y Sprints cerrados del snapshot. No duplica fórmulas. Para Sprint, el esfuerzo directo a historia se conserva en el input de métricas para que `calculateActualHours` lo cuente como no atribuible, pero se excluye de la tabla de esfuerzo del PDF; la cobertura lo informa. Alternativa descartada: llamar al lector de métricas por separado, que abriría otro snapshot.

4. **Semántica de ámbito alineada con US-19.**
   - Proyecto: todas las historias del proyecto, todas las tareas y todos los registros de esfuerzo, todos los defectos y todos los Sprints cerrados para la velocidad.
   - Sprint: historias asignadas al Sprint, tareas del Sprint, esfuerzo de historias asignadas (el dominio filtra para la tabla) y defectos detectados o resueltos en el Sprint.
   - Completada: `sprint_stories.completed_at` del ámbito; en proyecto, la primera finalización observada de la historia. `stories.status` solo se muestra como estado global, no decide la finalización.
   - Defectos del Sprint: unión de detección y resolución, sin duplicar filas.

5. **Renderizador PDF pure Go.** Se usa `github.com/go-pdf/fpdf` (fork mantenido de gofpdf, sin cgo). El renderizador dibuja secciones con tablas de ancho fijo, paginación automática, numeración de páginas y un traductor UTF-8 a cp1252 para acentos y eñes con las fuentes base. Los valores `null` se muestran como «No disponible». Alternativa descartada: construir el PDF a mano, que exige gestionar fuentes, métricas de texto y paginación.

6. **Rutas con extensión y descarga directa.** `GET /projects/{project_id}/report.pdf` y `GET /projects/{project_id}/sprints/{sprint_id}/report.pdf`, con `HEAD`. Responden `Content-Type: application/pdf` y `Content-Disposition: attachment; filename="reporte-proyecto-<uuid>.pdf"` o `reporte-sprint-<uuid>.pdf`. Alternativa descartada: una ruta genérica `/reports` que obligaría a un parámetro de ámbito y perdería la jerarquía existente.

7. **Errores y disponibilidad existentes.** Validación de UUID antes de leer (`422 validation_failed`), proyecto/Sprint inexistente o ajeno (`404 project_not_found` / `sprint_not_found`), método no admitido (`405`) y fallo genérico (`500 internal_error`). Las rutas se registran solo con esquema limpio `>= 14`, igual que métricas y defectos; con versión menor, `dirty`, error de resolución o dependencia ausente responden `404`.

8. **Contenido del PDF.** Encabezado de identificación; resumen de métricas y cobertura; tabla de historias con prioridad, estado, puntos, horas estimadas y finalización; tabla de tareas y estimaciones; tabla de esfuerzo con fecha, integrante, historia, tarea, horas y descripción; tabla de defectos con severidad, estado, historia y Sprints de detección/resolución. Las secciones vacías se muestran con un texto explícito.

## Riesgos y compensaciones

- **Textos largos y tablas.** Las descripciones pueden superar una línea; el renderizador calcula la altura de cada fila con `SplitLines` y pagina antes de dibujar. Mitigación: prueba con descripciones largas y varias páginas.
- **Acentos y eñes.** La fuente base no es Unicode; se traduce a cp1252. Mitigación: prueba con texto acentuado y verificación de contenido.
- **Consistencia entre métricas y detalle.** Una consulta mal filtrada podría incluir esfuerzo de otro Sprint. Mitigación: prueba de integración que compara el reporte con el lector de métricas y con datos cruzados entre proyectos y Sprints.
- **Costo de lectura.** El snapshot hace varias consultas; se filtran por proyecto/Sprint y se usan índices existentes. Mitigación: medir en la prueba de integración y evitar agregaciones repetidas.
- **Dependencia nueva.** `go-pdf/fpdf` es pure Go y estable; se fija una versión y se verifica `go build`, `go vet` y las pruebas. Mitigación: aislar el renderizador detrás de un puerto para poder sustituirlo.
- **Gate de esquema.** Si se registra la ruta con esquema incompleto, fallaría en tiempo de ejecución. Mitigación: extender `ResolveMigrationReadiness` y probar v13, v14, `dirty` y error de resolución.

## Migración

No hay migración ni cambio de datos. La reversión consiste en retirar el módulo, sus rutas y su cableado; los datos existentes no se tocan. La documentación de API y la matriz de disponibilidad se actualizan en el mismo cambio.
