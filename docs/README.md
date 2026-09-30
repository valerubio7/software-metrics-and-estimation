# Documentación del proyecto

Estado al 2026-09-30: MVP parcial de backend Go/PostgreSQL. Están implementadas US-01, US-02, US-05, US-06, US-07 y US-08; esto no equivale al producto completo requerido por la consigna.

## Recorrido de lectura

| Documento | Qué permite comprobar |
|---|---|
| [Consigna original](assignment.md) | Requisitos académicos, metodología y entregables; contenido conservado sin cambios. |
| [Arquitectura actual](architecture/overview.md) | Módulos, rutas HTTP y condiciones de disponibilidad. |
| [Trazabilidad](traceability.md) | Recorrido de las seis US entre issue, especificación, escenarios, pruebas y Go. |
| [Equipo](scrum/team.md) | Integrantes y roles confirmados, sin atribuciones exclusivas. |
| [Sprint 1](scrum/sprint-1.md) | Instantánea del tablero, propuesta de objetivo y decisiones pendientes. |
| [Uso de IA](ai-usage.md) | Evidencia disponible y registro futuro de validación humana. |

## Fuentes y límites

- [Repositorio](https://github.com/valerubio7/software-metrics-and-estimation).
- [Tablero GitHub Projects #4](https://github.com/users/valerubio7/projects/4).
- [Especificaciones canónicas](../openspec/specs/) y [cambios archivados](../openspec/changes/archive/).
- Los escenarios Given–When–Then están documentados en Markdown; las pruebas actuales son Go, no una ejecución nativa de Gherkin.
- El estado del tablero es una instantánea, no una certificación de despliegue ni de todos los criterios de aceptación.

## Documentación diferida

No se incluyen todavía README raíz, manual de usuario, guía o informe de pruebas/cobertura, actas de review ni retrospectivas. La consigna los requiere como entregables; su ausencia no se presenta como trabajo terminado.

Las propuestas y los pendientes se identifican expresamente. Actualizar cada documento cuando exista evidencia nueva, conservando la fecha de la instantánea y los enlaces que la sustentan.
