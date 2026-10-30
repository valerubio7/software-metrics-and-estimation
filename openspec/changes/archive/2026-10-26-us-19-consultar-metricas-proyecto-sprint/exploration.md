# Exploración de US-19

La capacidad nueva es `metricas`: consulta API de solo lectura. Las fórmulas aprobadas en S1-S8 de `odd/tasks/us-19-metricas.md` y sus decisiones hasta L12 son la fuente de decisiones; no se implementa TUI-10 (#103), US-20 ni US-21.

## Evidencia y reutilización

| Fuente leída | Conclusión |
|---|---|
| Issue #47, consultada mediante `gh issue view 47 --json title,body` | Exige seleccionar ámbito, nueve métricas, información registrada, insuficiencia explícita y ausencia de mutaciones. La TUI pertenece a otra entrega. |
| `openspec/specs/{project,historia,sprint,tarea,esfuerzo,defecto}/spec.md`, leídas completas | No existe capacidad de agregación equivalente; las exclusiones de US-19 delimitan cambios históricos. No se modifican contratos de escritura. |
| `internal/api/api.go` | Composición opcional y gates centralizados; reutilizar `ResolveMigrationReadiness` y `NewHTTPHandlerWithDependencies`. Las consultas GET existentes admiten HEAD por el mux. |
| `internal/task/infrastructure/postgres/repository.go` | Tareas conservan proyecto, historia y Sprint; puede haber múltiples tareas con estimaciones nulas. |
| `internal/effort/infrastructure/postgres/repository.go` | Cada registro conserva `story_id` y opcionalmente `task_id`; contar cada fila una vez, no sumar historia y tarea por separado. |
| `internal/project/infrastructure/postgres/migrations/000014_add_defect_resolution_sprint.up.sql` | La columna de resolución ya existe y referencia un Sprint del proyecto; no se prevé migración nueva. |
| Especificaciones de historia y Sprint | `completed_at` es la fuente de finalización y `is_closed` la de cierre; no usar estado global. |

## Criterios exactos de aceptación de #47

- El usuario debe poder seleccionar el ámbito sobre el cual desea consultar las métricas.
- El sistema debe calcular los **Story Points planificados**.
- El sistema debe calcular los **Story Points completados**.
- El sistema debe calcular la **velocidad del equipo**.
- El sistema debe calcular las **horas estimadas**.
- El sistema debe calcular las **horas reales**.
- El sistema debe calcular la **desviación entre el esfuerzo estimado y el esfuerzo real**.
- El sistema debe calcular el **porcentaje de historias completadas**.
- El sistema debe calcular la **cantidad de defectos detectados**.
- El sistema debe calcular la **cantidad de defectos resueltos**.
- Las métricas deben calcularse utilizando la información registrada en el sistema.
- Si no existen datos suficientes para calcular una determinada métrica, el sistema debe indicar la situación en lugar de presentar un valor incorrecto.
- La consulta de métricas no debe modificar los datos utilizados para su cálculo.

## Decisiones confirmadas

Proyecto deduplica historias; Sprint usa historias asignadas. Puntos requieren estimaciones completas del subconjunto pertinente. Velocidad de proyecto promedia todos los Sprints cerrados, incluyendo ceros, y es desconocida si alguno es incompleto. Proyecto suma todos los registros de cada historia única y estima con todas sus tareas, reemplazadas por la estimación de historia solo cuando no tiene tareas. Sprint suma únicamente esfuerzo vinculado a tareas de ese Sprint y estima con sus tareas; si una historia asignada no tiene tareas en ese Sprint, usa su estimación propia. Las sumas reales parciales se conservan y se cuentan historias sin registros atribuibles. Defectos separan detección y resolución.

L9 resuelve los bordes: puntos y velocidad de Sprint vacíos son 0; porcentaje sin denominador, horas sin observaciones y promedio sin Sprints cerrados son `null`. El porcentaje de desviación es `null` sin operandos o con estimación cero. Se agrega `coverage` y se conserva `deviation` como diferencia horaria.

L12 reemplaza la atribución inicial por historia: tras la indicación “hace lo mas recomendable”, se adopta atribución estricta para no repetir esfuerzo entre Sprints. Los registros directos a historia no tienen Sprint y se excluyen del total Sprint; `coverage.actual_hours.unattributed_story_level_records` cuenta esas filas de las historias asignadas. En proyecto el contador es 0, pues las incluye en su total. No se asigna esfuerzo por fecha ni se incorpora una migración.

## Trazabilidad y comprobación futura

| Especificación | Evidencia prevista |
|---|---|
| S1 | Propuesta, alcance, artefactos españoles y seguimiento de entrega/archivo |
| S2-S3 | Pruebas públicas por proyecto/Sprint, deduplicación, finalización y puntos faltantes |
| S4 | Promedio cerrado con cero, sin Sprints e incompleto |
| S5 | Historia compartida entre Sprints, tareas seleccionadas y reemplazo por historia, esfuerzo atribuible parcial/ausente, diferencias y porcentajes |
| S6 | Detección en A y resolución en B; conteos sin defectos |
| S7 | Contrato plano, nulabilidad, historias sin registros atribuibles y registros directos no atribuibles |
| S8 | Validaciones, referencias ajenas/inexistentes, errores, no mutación y no regresión |

No se ejecutan pruebas de comportamiento en esta fase documental. RED/GREEN no aplica a autoría pasiva; las tareas posteriores exigen pruebas antes de implementación. La validación OpenSpec verifica estructura, no demuestra que la API esté implementada.
