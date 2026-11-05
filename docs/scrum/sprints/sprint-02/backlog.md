# Sprint 2 — Sprint Backlog

* **Iteración en el tablero:** 06/10/2026 – 18/10/2026 (13 días)
* **Ventana de ejecución observada:** 06/10/2026 – 20/10/2026
* **Sprint Goal:** dotar al software de una interfaz usable.
* **Comprometidas:** 7 · **Completadas:** 7 · **Arrastradas:** 0
* **PRs mergeadas:** 10

## Backlog del Sprint

| # | Historia | Épica | Estado | Evidencia |
|---|---|---|---|---|
| #40 | US-12 — Cerrar un Sprint | EPIC-03 | Completada | [PR #91](https://github.com/valerubio7/software-metrics-and-estimation/pull/91) · [change](../../../../openspec/changes/archive/2026-10-13-us-12-cerrar-sprint) |
| #41 | US-13 — Consultar Sprints anteriores | EPIC-03 | Completada | [PR #94](https://github.com/valerubio7/software-metrics-and-estimation/pull/94) · [change](../../../../openspec/changes/archive/2026-10-14-us-13-consultar-sprints-anteriores) |
| #83 | TUI-01 — Abrir TUI y navegar | EPIC-08 | Completada | [PR #88](https://github.com/valerubio7/software-metrics-and-estimation/pull/88) · [change](../../../../openspec/changes/archive/2026-10-10-tui-01-open-navigate) |
| #84 | TUI-02 — Gestionar proyectos desde TUI | EPIC-08 | Completada | [PR #92](https://github.com/valerubio7/software-metrics-and-estimation/pull/92) · [change](../../../../openspec/changes/archive/2026-10-13-tui-02-gestionar-proyectos) |
| #85 | TUI-03 — Gestionar backlog desde TUI | EPIC-08 | Completada | [PR #95](https://github.com/valerubio7/software-metrics-and-estimation/pull/95) · [change](../../../../openspec/changes/archive/2026-10-14-tui-03-gestionar-backlog) |
| #86 | TUI-04 — Gestionar Sprint I: operar desde TUI | EPIC-08 | Completada | [PR #96](https://github.com/valerubio7/software-metrics-and-estimation/pull/96) · [spec TUI](../../../../openspec/specs/tui/spec.md) |
| #87 | TUI-05 — Gestionar Sprint II: cerrar e historial desde TUI | EPIC-08 | Completada | [PR #97](https://github.com/valerubio7/software-metrics-and-estimation/pull/97) · [spec TUI](../../../../openspec/specs/tui/spec.md) |

## Plan y ejecución

La TUI se construyó por secciones, una PR por flujo: abrir y navegar,
proyectos, backlog, operaciones de Sprint y cierre/historial. El cierre de
Sprint y el historial se resolvieron en el backend (US-12 y US-13) y se
expusieron en la TUI.

Dos historias de backend (US-12 y US-13) y cinco de interfaz (TUI-01 a TUI-05)
entraron en el mismo Sprint. TUI-02 registró una excepción de entrega en su
change. TUI-04 y TUI-05 no tienen change propio; su especificación quedó en
`openspec/specs/tui/spec.md`.

## Resultado

El Sprint cumplió el objetivo: la TUI permite operar proyectos, backlog y
Sprints sin usar la API directamente. El rediseño de navegación de la [PR #98](https://github.com/valerubio7/software-metrics-and-estimation/pull/98)
cerró la ventana con una experiencia más coherente que la de las primeras
pantallas.

## Trazabilidad

- Product Backlog: [`../../product-backlog.md`](../../product-backlog.md)
- Acta: [`retrospective.md`](retrospective.md)

