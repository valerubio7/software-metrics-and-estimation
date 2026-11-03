# Propuesta: generar un reporte PDF de un proyecto o Sprint (US-21)

## Intención

US-21 (#49) exige un documento consolidado y exportable con la planificación, el seguimiento y la calidad de un proyecto o Sprint. Hoy la información existe en la API y en la TUI, pero no puede obtenerse como un informe PDF de solo lectura.

## Cambios propuestos

- Ofrecer `GET` y `HEAD` en `/projects/{project_id}/report.pdf` y `/projects/{project_id}/sprints/{sprint_id}/report.pdf`.
- Componer un único snapshot de lectura con la identificación del proyecto o Sprint, sus historias, estimaciones, tareas, esfuerzo, defectos y las métricas de US-19.
- Generar el PDF con las secciones de identificación, métricas y cobertura, historias planificadas y completadas, estimaciones, esfuerzo registrado y defectos relacionados.
- Reutilizar las fórmulas de US-19 mediante el dominio de métricas, sin introducir métricas nuevas ni lecturas inconsistentes.
- Mantener la generación como operación de solo lectura: sin migraciones, escrituras ni cambios de datos.
- Documentar la API y la disponibilidad por versión de esquema.

## Capacidades

### Capacidades nuevas

- `reporte`: generación de reportes PDF de solo lectura para un proyecto o Sprint, con historias, estimaciones, esfuerzo, métricas y defectos del ámbito.

### Capacidades modificadas

Ninguna. La capacidad consume los contratos existentes de proyecto, historia, Sprint, tarea, esfuerzo, defecto y métricas sin modificarlos.

## Impacto

Se incorpora un módulo `internal/report` con dominio, aplicación, lector PostgreSQL, renderizador PDF y transporte HTTP, además de la composición en `internal/api` y `cmd/api`. Se agrega la dependencia Go para PDF y se actualiza el README. Las rutas quedan disponibles solo con el esquema limpio en versión 14 o superior, igual que métricas y defectos. Quedan fuera de alcance el Dashboard (US-20), la TUI, el cálculo de métricas distintas de US-19 y cualquier migración de datos.
