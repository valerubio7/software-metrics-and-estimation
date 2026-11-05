# Sprint 3 — Sprint Backlog

* **Iteración en el tablero:** 19/10/2026 – 17/11/2026 (29 días)
* **Ventana de ejecución observada:** 21/10/2026 – 01/11/2026
* **Sprint Goal:** añadir funcionalidades clave y robustecer el sistema; implementar la visualización.
* **Comprometidas:** 13 · **Completadas en la ventana:** 9 · **Arrastradas al Sprint 4:** 4
* **PRs mergeadas:** 21

## Backlog del Sprint

| # | Historia | Épica | Estado | Evidencia |
|---|---|---|---|---|
| #42 | US-14 — Estimar una historia mediante Planning Poker | EPIC-04 | Completada en el Sprint | [PR #107](https://github.com/valerubio7/software-metrics-and-estimation/pull/107) |
| #43 | US-15 — Registrar el esfuerzo realizado en una historia o tarea | EPIC-05 | Completada en el Sprint | [PR #109](https://github.com/valerubio7/software-metrics-and-estimation/pull/109) · [change](../../../../openspec/changes/archive/2026-10-25-us-15-registrar-esfuerzo) |
| #44 | US-16 — Registrar un defecto | EPIC-06 | Completada en el Sprint | [PR #110](https://github.com/valerubio7/software-metrics-and-estimation/pull/110) · [change](../../../../openspec/changes/archive/2026-10-25-us-16-registrar-defecto) |
| #45 | US-17 — Actualizar el estado y resolución de un defecto | EPIC-06 | Completada en el Sprint | [PR #11](https://github.com/valerubio7/software-metrics-and-estimation/pull/11)1–#113, #116 |
| #46 | US-18 — Consultar los defectos de un proyecto | EPIC-06 | Arrastrada al Sprint 4 (03/11) | [PR #136](https://github.com/valerubio7/software-metrics-and-estimation/pull/136) · [change](../../../../openspec/changes/archive/2026-11-03-us-18-consultar-defectos-proyecto) |
| #47 | US-19 — Consultar las métricas de un proyecto o Sprint | EPIC-07 | Completada en el Sprint | [PR #12](https://github.com/valerubio7/software-metrics-and-estimation/pull/12)0–#123, #125 · [change](../../../../openspec/changes/archive/2026-10-26-us-19-consultar-metricas-proyecto-sprint) |
| #48 | US-20 — Visualizar el Dashboard de un proyecto | EPIC-08 | Arrastrada al Sprint 4 (03/11) | [PR #137](https://github.com/valerubio7/software-metrics-and-estimation/pull/137) |
| #100 | TUI-07 — Estimar historias con Planning Poker desde la TUI | EPIC-04 | Completada en el Sprint | [PR #107](https://github.com/valerubio7/software-metrics-and-estimation/pull/107) · [spec TUI](../../../../openspec/specs/tui/spec.md) |
| #101 | TUI-08 — Registrar esfuerzo desde la TUI | EPIC-05 | Completada en el Sprint | [PR #128](https://github.com/valerubio7/software-metrics-and-estimation/pull/128) · [change](../../../../openspec/changes/archive/2026-10-31-tui-08-registrar-esfuerzo) |
| #102 | TUI-09 — Gestionar defectos desde la TUI | EPIC-06 | Arrastrada al Sprint 4 (03/11) | [PR #12](https://github.com/valerubio7/software-metrics-and-estimation/pull/12)9–#135 · [spec TUI](../../../../openspec/specs/tui/spec.md) |
| #103 | TUI-10 — Consultar métricas desde la TUI | EPIC-07 | Completada en el Sprint | [PR #115](https://github.com/valerubio7/software-metrics-and-estimation/pull/115) (fix #124) · [spec TUI](../../../../openspec/specs/tui/spec.md) |
| #104 | TUI-11 — Visualizar el Dashboard desde la TUI | EPIC-08 | Arrastrada al Sprint 4 (03/11) | [PR #137](https://github.com/valerubio7/software-metrics-and-estimation/pull/137) · [change](../../../../openspec/changes/archive/2026-11-03-tui-11-visualizar-dashboard) |
| #105 | TUI-06 — Extender la navegación a las secciones del Sprint 3 | EPIC-08 | Completada en el Sprint | [PR #106](https://github.com/valerubio7/software-metrics-and-estimation/pull/106) · [change](../../../../openspec/changes/archive/2026-10-25-tui-06-navegacion-sprint-3) |

## Plan y ejecución

US-19 (métricas) se partió en cuatro PRs por unidad de trabajo: `1/4` dominio
y cobertura, `2/4` consulta y validación, `3/4` snapshot PostgreSQL y `4/4`
endpoints, más una consolidación. US-17 (ciclo de vida del defecto) se
implementó en cuatro PRs, pero no tiene change archivado.

Cuatro historias comprometidas se completaron el 03/11, después del cierre
observado del Sprint (01/11): US-18, US-20, TUI-09 y TUI-11. Se registran como
arrastre al Sprint 4 para no contarlas dos veces.

## Resultado

El Sprint entregó estimación, esfuerzo, defectos y métricas, más la navegación
de la TUI a esas secciones. La deuda principal fue de proceso: código que no
compilaba en la [PR #115](https://github.com/valerubio7/software-metrics-and-estimation/pull/115), tests rojos en `main` por la [PR #126](https://github.com/valerubio7/software-metrics-and-estimation/pull/126) y un change sin
archivar (US-17).

## Trazabilidad

- Product Backlog: [`../../product-backlog.md`](../../product-backlog.md)
- Acta: [`retrospective.md`](retrospective.md)

