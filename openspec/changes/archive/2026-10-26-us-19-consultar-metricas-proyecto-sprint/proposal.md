# Propuesta: consultar métricas por proyecto o Sprint

## Intención

US-19 (#47) permite evaluar objetivamente avance, esfuerzo y calidad con información registrada. La API debe distinguir resultados conocidos de datos insuficientes sin modificar los datos consultados.

## Cambios propuestos

- Consultar métricas mediante las rutas de proyecto y Sprint acordadas, seleccionando el ámbito por la ruta.
- Calcular puntos planificados/completados, velocidad, horas estimadas/reales, desviación, porcentaje completado y defectos detectados/resueltos.
- Aplicar las fórmulas y límites aprobados en S2-S7 de `odd/tasks/us-19-metricas.md`; informar valores desconocidos mediante `null` y cobertura explícita.
- Conservar el objeto plano consumido por PR #115; agregar `deviation_percentage` y `coverage` sin cambiar el significado horario de `deviation`.

## Capacidades

### Capacidades nuevas

- `metricas`: consulta de métricas de planificación, avance, esfuerzo y defectos, por proyecto o Sprint, de solo lectura.

### Capacidades modificadas

Ninguna. Se leyeron completas las especificaciones de proyecto, historia, Sprint, tarea, esfuerzo y defecto. Sus operaciones conservan contratos y fuentes de datos; sus exclusiones de US-19 delimitan aquellas entregas, no prohíben una capacidad nueva de consulta.

## Impacto

La implementación posterior incorporará un módulo de lectura con dominio/aplicación, repositorio PostgreSQL, transporte HTTP y composición en API/arranque. Actualizará documentación de API y matriz de rutas; reutilizará el esquema existente hasta versión 14, sin migración prevista ni dependencia nueva. TUI-10 (#103), US-20 y US-21 quedan fuera, al igual que registrar esfuerzo o registrar/resolver defectos. Esta entrega solo autoriza artefactos, no implementación ni commits.
