# Sprint 1 — Instantánea y planificación pendiente

Al 2026-09-30, el [Project #4](https://github.com/users/valerubio7/projects/4) contiene once historias del Sprint 1: seis Done, una In progress, una Ready y tres Backlog. Es una instantánea del tablero, no un acta de cierre.

## Datos observados

| Dato | Valor |
|---|---|
| Iteración | Sprint 1 |
| Inicio | 2026-09-15 |
| Duración configurada | 21 días |
| Alcance observado | US-01 a US-11 |
| Sprint Goal específico | No confirmado. |
| Capacidad y estimaciones | No confirmadas; no se calcula velocidad con estos datos. |

## Historias y estado

| Historia | Estado al 2026-09-30 | Responsable pendiente de trabajo |
|---|---|---|
| [US-01 — Crear proyecto](https://github.com/valerubio7/software-metrics-and-estimation/issues/29) | Done | No se atribuye autoría desde el estado. |
| [US-02 — Modificar proyecto](https://github.com/valerubio7/software-metrics-and-estimation/issues/30) | Done | No se atribuye autoría desde el estado. |
| [US-03](https://github.com/valerubio7/software-metrics-and-estimation/issues/31) | Ready | Sin asignar. |
| [US-04](https://github.com/valerubio7/software-metrics-and-estimation/issues/32) | Backlog | Sin asignar. |
| [US-05 — Crear historia](https://github.com/valerubio7/software-metrics-and-estimation/issues/33) | Done | No se atribuye autoría desde el estado. |
| [US-06 — Modificar historia](https://github.com/valerubio7/software-metrics-and-estimation/issues/34) | Done | No se atribuye autoría desde el estado. |
| [US-07 — Consultar Product Backlog](https://github.com/valerubio7/software-metrics-and-estimation/issues/35) | Done | No se atribuye autoría desde el estado. |
| [US-08 — Crear Sprint](https://github.com/valerubio7/software-metrics-and-estimation/issues/36) | Done | No se atribuye autoría desde el estado. |
| [US-09](https://github.com/valerubio7/software-metrics-and-estimation/issues/37) | In progress | Pablo Geyer (`PabloGeyer`). |
| [US-10](https://github.com/valerubio7/software-metrics-and-estimation/issues/38) | Backlog | Luciano (`Lucho-cas`). |
| [US-11](https://github.com/valerubio7/software-metrics-and-estimation/issues/39) | Backlog | Sin asignar. |

Las seis historias Done tienen un recorrido de [trazabilidad](../traceability.md). Done no significa que toda la consigna esté implementada ni que la secuencia externa de migraciones haya sido validada para despliegue.

## Sprint Goal — PROPUESTA para aprobación

> Permitir planificar y seguir una iteración: crear un proyecto, administrar su backlog, crear un Sprint, asignarle historias y registrar su finalización.

Esta formulación no es un acuerdo confirmado. La API actual cubre proyectos, backlog y creación de Sprint; asignación y registro de finalización no se presentan aquí como implementados. El objetivo general de «MVP» de la consigna tampoco reemplaza un Sprint Goal aprobado por el equipo.

## Notas posteriores

- 2026-10-04: el registro de finalización de una historia en el Sprint (US-11) está implementado en código (`POST .../completion`, migración `000010`). Esta nota no modifica la instantánea del tablero anterior.

## Decisiones pendientes

- Confirmar o ajustar la propuesta de Sprint Goal con el equipo y Product Architect.
- Acordar capacidad y estimaciones antes de comprometer alcance adicional.
- Asignar responsables a las historias aún sin asignar.
- Evaluar el riesgo de migraciones señalado en la [arquitectura](../architecture/overview.md#disponibilidad-al-arrancar).
- **Recomendación, no decisión:** considerar diferir US-10 si no contribuye al objetivo aprobado o excede la capacidad. US-10 permanece en el alcance observado; este documento no modifica el tablero.

## Evidencia de actividades

La consigna exige planificación, seguimiento, review y retrospectiva. No se cuenta aquí con actas, fechas, participantes ni acuerdos que permitan afirmar su realización. Deben registrarse cuando exista evidencia; no se reconstruyen ceremonias retrospectivamente a partir de estados de issues.
