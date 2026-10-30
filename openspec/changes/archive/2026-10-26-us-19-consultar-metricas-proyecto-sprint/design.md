# Diseño de la consulta de métricas

## Context

Véase la motivación en [proposal.md](proposal.md) y el contrato en [specs/metricas/spec.md](specs/metricas/spec.md). La API compone módulos opcionales mediante `HTTPDependencies` y resuelve disponibilidad con `ResolveMigrationReadiness` (`internal/api/api.go`). La migración 14 incorpora `defects.resolution_sprint_id`; los esfuerzos ya conservan `story_id` aunque tengan tarea. No se necesitan escrituras para consultar.

## Objetivos y exclusiones

**Objetivos:** aislar el cálculo y la lectura, mantener respuestas consistentes y evitar multiplicar observaciones. Expresar ausencia mediante valores anulables, sin omitir claves.

**No objetivos:** persistir métricas, congelar una línea base histórica, introducir migraciones o dependencias, cambiar operaciones de escritura o implementar TUI, gráficos o reportes.

## Decisiones

1. **Módulo de lectura separado.** Seguir dominio/aplicación, infraestructura PostgreSQL y transporte HTTP existentes. El caso de uso valida identificadores y solicita una lectura por ámbito. Reutilizar las convenciones de validación y traducción de errores existentes; no invocar casos de uso de escritura ni generar identificadores. Alternativa descartada: encadenar endpoints existentes, que no entregan todas las observaciones y producirían lecturas inconsistentes.
2. **Conjuntos base únicos y agregaciones independientes.** Proyecto parte de historias únicas por proyecto; Sprint parte de sus asociaciones. Preagregar tareas y esfuerzo por historia para proyecto y por historia/Sprint de la tarea para Sprint, luego combinar resultados; filtrar las tareas seleccionadas antes de decidir el reemplazo por estimación de historia. Detecciones y resoluciones se agregan separadamente, sin unirlas al producto tareas × esfuerzos × asociaciones. No usar `SUM(DISTINCT hours)`: registros diferentes con iguales horas son observaciones independientes.
3. **Atribución estricta de esfuerzo (S5/S7, L12).** Proyecto usa todas las tareas y registros de cada historia única, una vez por fila. Sprint estima solo con tareas del Sprint seleccionado; para cada historia asignada sin tareas en ese Sprint, usa la estimación de historia aunque tenga tareas en otro Sprint. Una tarea seleccionada sin estimación no habilita ese reemplazo. Horas reales de Sprint incluyen solo registros cuyo `task_id` referencia una tarea de ese Sprint; no inferir atribución por pertenencia de la historia, fecha ni estado. Contar por separado los registros directos (`task_id=null`) de sus historias asignadas en `coverage.actual_hours.unattributed_story_level_records`. En proyecto ese contador es 0: todos los registros directos ya son atribuibles al proyecto. `stories_without_records` cuenta historias sin registros atribuibles al ámbito; los registros directos no evitan que una historia cuente como faltante en Sprint. Alternativa descartada: repetir todo el esfuerzo de la historia en cada Sprint, que duplicaría su atribución.
4. **Finalización y velocidad separadas.** Proyecto deduplica finalización por historia; velocidad promedia totales de cada Sprint cerrado sin deduplicarlos entre Sprints. Se incluyen Sprints cerrados vacíos. `stories.status` no participa. Alternativa descartada: dividir puntos completados únicos del proyecto por número de Sprints, que no respeta S4.
5. **Lectura consistente.** Resolver existencia, pertenencia y agregados en una única sentencia con preagregaciones o en una transacción de solo lectura con instantánea consistente. No construir una respuesta con agregados de momentos distintos. Un fallo impide toda respuesta parcial.
6. **Contrato aditivo.** Mantener claves raíz; `sprint_id` es `null` para proyecto. Añadir `deviation_percentage` y este objeto de cobertura, sin convertir valores ausentes a cero:

```json
{
  "coverage": {
    "story_points_planned": {"missing_observations": 0},
    "story_points_completed": {"missing_observations": 0},
    "velocity": {"missing_observations": 0},
    "estimated_hours": {"missing_observations": 0},
    "actual_hours": {"stories_without_records": 0, "unattributed_story_level_records": 0}
  }
}
```

Los ceros del ejemplo son ilustrativos, no sustituyen los conteos calculados. Sin Sprints cerrados, velocidad es `null` con cero faltantes: no existe promedio, no hay un Sprint incompleto. Las horas estimadas cuentan unidades seleccionadas faltantes (tarea seleccionada o historia sin tareas en el ámbito seleccionado), no historias afectadas. Mantener decimales de horas sin truncamiento; porcentajes y promedios deben admitir fracciones, sin imponer aquí un formato visual ni redondeo de presentación.
7. **Disponibilidad independiente.** Extender el mecanismo existente de resolución de esquema y composición, no duplicarlo. Se propone esquema limpio >=14 y dependencia de lectura explícita, pues la resolución de defectos necesita esa columna. Con versión inferior, `dirty`, fallo de resolución o dependencia ausente, no registrar las rutas nuevas; conservar todos los gates anteriores. Verificar la matriz de rutas al implementar.

## Riesgos y alternativas

- Multiplicación por JOIN → pruebas con múltiples Sprints, tareas, esfuerzos y defectos, incluyendo filas con iguales valores.
- Registros directos sin Sprint → informar su cantidad, excluirlos de horas reales de Sprint y conservarlos en proyecto; probar una historia compartida entre dos Sprints.
- Confundir desconocido y cero → probar conjuntos vacíos, faltantes parciales y cero explícito por separado.
- Consumidor PR #115 → probar claves raíz y semántica de `deviation`; cobertura solo aditiva. No modificar el consumidor en este cambio.
- Specs históricas con marcadores no canónicos o referencias a entregas antiguas → no repararlas en US-19; preservar sus contratos y validar este cambio por nombre.
- Coste de agregación → filtrar por proyecto/ámbito antes de agregar; evaluar consultas reales antes de proponer índices nuevos.

## Plan de migración

No se prevé migración. Revalidar esquema 14 y cablear el módulo detrás de la disponibilidad existente. La reversión consiste en retirar el módulo de consulta, sin revertir ni eliminar datos. Actualizar documentación de API y matriz de disponibilidad junto con el cableado; probar versiones compatibles e incompatibles. Archivar únicamente al finalizar implementación y verificación, bajo control del orquestador.
