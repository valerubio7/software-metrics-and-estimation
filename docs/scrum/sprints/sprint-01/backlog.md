# Sprint 1 — Sprint Backlog

* **Iteración en el tablero:** 15/09/2026 – 05/10/2026 (21 días)
* **Ventana de ejecución observada:** 22/09/2026 – 05/10/2026
* **Sprint Goal:** entregar una versión funcional básica (MVP).
* **Comprometidas:** 11 · **Completadas:** 11 · **Arrastradas:** 0
* **PRs mergeadas:** 31

## Backlog del Sprint

| # | Historia | Épica | Estado | Evidencia |
|---|---|---|---|---|
| #29 | US-01 — Crear proyecto | EPIC-01 | Completada | [change](../../../../openspec/changes/archive/2026-09-23-us-01-create-project) |
| #30 | US-02 — Modificar un proyecto | EPIC-01 | Completada | [PR #57](https://github.com/valerubio7/software-metrics-and-estimation/pull/57) · [change](../../../../openspec/changes/archive/2026-09-25-us-02-modify-project) |
| #31 | US-03 — Registrar integrantes en un proyecto | EPIC-01 | Completada | [PR #69](https://github.com/valerubio7/software-metrics-and-estimation/pull/69) · [change](../../../../openspec/changes/archive/2026-10-01-us-03-register-project-members) |
| #32 | US-04 — Consultar el estado de un proyecto | EPIC-01 | Completada | [PR #70](https://github.com/valerubio7/software-metrics-and-estimation/pull/70) · [change](../../../../openspec/changes/archive/2026-10-01-us-04-project-status) |
| #33 | US-05 — Crear una historia en el Product Backlog | EPIC-02 | Completada | [PR #56](https://github.com/valerubio7/software-metrics-and-estimation/pull/56) · [change](../../../../openspec/changes/archive/2026-09-24-us-05-crear-historia-product-backlog) |
| #34 | US-06 — Modificar una historia del Product Backlog | EPIC-02 | Completada | [PR #63](https://github.com/valerubio7/software-metrics-and-estimation/pull/63) · [change](../../../../openspec/changes/archive/2026-09-25-us-06-modificar-historia-product-backlog) |
| #35 | US-07 — Consultar el Product Backlog | EPIC-02 | Completada | [PR #64](https://github.com/valerubio7/software-metrics-and-estimation/pull/64) · [change](../../../../openspec/changes/archive/2026-09-26-us-07-consultar-product-backlog) |
| #36 | US-08 — Crear un Sprint y definir su Sprint Goal | EPIC-03 | Completada | [PR #65](https://github.com/valerubio7/software-metrics-and-estimation/pull/65) · [change](../../../../openspec/changes/archive/2026-09-27-us-08-create-sprint) |
| #37 | US-09 — Asignar historias a un Sprint | EPIC-03 | Completada | [PR #78](https://github.com/valerubio7/software-metrics-and-estimation/pull/78) · [change](../../../../openspec/changes/archive/2026-10-03-us-09-asignar-historias-sprint) |
| #38 | US-10 — Descomponer una historia del Sprint en tareas | EPIC-03 | Completada | [PR #80](https://github.com/valerubio7/software-metrics-and-estimation/pull/80) · [change](../../../../openspec/changes/archive/2026-10-03-us-10-descomponer-historia-tareas) |
| #39 | US-11 — Registrar una historia del Sprint como completada | EPIC-03 | Completada | [PR #81](https://github.com/valerubio7/software-metrics-and-estimation/pull/81) · [change](../../../../openspec/changes/archive/2026-10-04-us-11-registrar-historia-completada) |

## Plan y ejecución

El Sprint figura en el tablero desde el 15/09, pero el repositorio arranca el
22/09: la ventana observada empieza con el primer commit.

No hay un desglose de tareas por historia en el repositorio: la unidad de
trabajo fue la historia, con su change de OpenSpec y sus PRs. Cada corte fue
vertical (dominio, aplicación, persistencia y HTTP) y se cerró con pruebas
unitarias y de integración.

El acta del Sprint 1 registra los dos incidentes de integración que costaron
retrabajo: el merge `08e324b` que rompió el wiring US-07/US-08 y la
consolidación posterior de US-09.

## Resultado

El MVP quedó completo: proyectos, backlog, Sprints, tareas y finalización de
historias, con API HTTP y esquema migrado. Las 11 historias comprometidas se
completaron dentro de la ventana del Sprint.

## Trazabilidad

- Product Backlog: [`../../product-backlog.md`](../../product-backlog.md)
- Acta: [`retrospective.md`](retrospective.md)

