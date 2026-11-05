# Sprint 2 — Retrospectiva

> **Documento reconstruido.** No se conservan notas originales de la ceremonia.
> Esta acta se armó el 05/11/2026 a partir del historial de Git, las PRs
> mergeadas y los cambios archivados en `openspec/changes/archive/`. El período
> cubierto se deriva de esos artefactos, no de un calendario de ceremonias.

* **Período cubierto:** 06/10/2026 – 20/10/2026
* **Objetivo del Sprint (consigna):** dotar al software de una interfaz usable.
* **Participantes (equipo):** Valentín Rubio, Pablo Geyer, Luciano, Santiago
  Calzolari y Santiago Oses.
* **Formato:** qué mantener / qué mejorar, cierre con acciones.

## Entregado en el Sprint

| Entrega | Evidencia | Fecha |
|---|---|---|
| US-12 — Cerrar un Sprint | PR #91, change `2026-10-13-us-12-cerrar-sprint` | 12/10 |
| TUI-01 — Abrir la TUI y navegar | PR #88, change `2026-10-10-tui-01-open-navigate` | 12/10 |
| TUI-02 — Gestionar proyectos | PR #92, change `2026-10-13-tui-02-gestionar-proyectos` | 12/10 |
| TUI-03 — Gestionar el backlog | PR #95, change `2026-10-14-tui-03-gestionar-backlog` | 14/10 |
| US-13 — Consultar Sprints anteriores | PR #94, change `2026-10-14-us-13-consultar-sprints-anteriores` | 14/10 |
| TUI-04 — Operar el Sprint desde la TUI | PR #96 | 15/10 |
| TUI-05 — Cerrar Sprint y ver historial | PR #97 | 15/10 |
| Rediseño de navegación y selector de proyecto | PR #98 | 20/10 |

Diez PRs mergeadas en quince días.

## Qué mantener

* **TDD visible en la TUI.** TUI-01 arrancó con el ciclo explícito en el
  historial: `4467763 test(tui): RED navegacion base con menu y salida` y
  `bfad098 feat(tui): GREEN navegacion base BubbleTea sin mutar datos`. La
  evidencia de TDD no quedó solo en los documentos.
* **Una sección de la TUI por PR con un responsable claro.** Proyectos,
  backlog, operaciones de Sprint y cierre/historial entraron por PRs separadas
  (#92, #95, #96, #97), revisables de a una.
* **US-12 y US-13 con change de OpenSpec archivado.** El recorrido
  especificación → implementación → archivo se mantuvo en las historias de
  backend, incluso en un Sprint centrado en interfaz.
* **El cierre de Sprint no se dio por hecho por los tests por historia.**
  US-12 sumó una verificación dedicada en la PR #91.

## Qué mejorar

* **La navegación del Sprint se rompió al integrar TUI-04 y TUI-05.** Se
  corrigió en `513c575 fix(tui): restore Sprint task and completion navigation`
  (14/10). Es el mismo patrón del Sprint 1: cada historia pasaba sus pruebas,
  pero el recorrido integrado no.
* **La tecla `q` global interfería con los campos de texto.** Se corrigió en
  `c1cd178 fix(tui): scope q to non-text screens and accept spaces in fields`
  (14/10), después de mergear los flujos que la exponían.
* **Se rediseñó la TUI después de construir cuatro secciones encima.**
  `e5040f8 feat(tui): redesign navigation and visuals around a project picker`
  y siete commits más llegaron el 16/10, con TUI-02 a TUI-05 ya mergeadas. No
  hubo una validación de la experiencia antes de replicarla.
* **No se cumplió la acción del Sprint 1 de anotar dependencias antes de
  mergear.** Volvió a aparecer un fix de integración (`513c575`).

## Acciones para Sprint 3

| Acción | Evidencia de cumplimiento | Responsable | Plazo |
|---|---|---|---|
| Correr `go build ./...` y `go test ./...` sobre el árbol integrado antes de mergear | PR sin tests rojos y registro del comando ejecutado | Por definir en planning | Cada PR del Sprint 3 |
| Validar una sección de la TUI con el cliente antes de replicar el patrón en las demás | Nota de review con la sección validada y los cambios pedidos | Por definir en planning | Mitad del Sprint 3 |
| Registrar en la issue el orden de integración de las PRs relacionadas | Issue con la lista de dependencias y las PRs linkeadas | Por definir en planning | Antes de abrir la primera PR del Sprint 3 |

## Estado de las acciones del Sprint 1

| Acción | Estado | Evidencia |
|---|---|---|
| Anotar orden de integración y dependencias antes de mergear | **No cumplida** | `513c575` vuelve a corregir una integración rota |
| Probar el recorrido proyecto → backlog → Sprint además de los tests por historia | **Parcial** | TUI-01 a TUI-05 tienen pruebas; el recorrido integrado falló igual |
| Dejar acta breve de cada ceremonia | **No cumplida** | Esta acta es una reconstrucción, no una nota de la ceremonia |
