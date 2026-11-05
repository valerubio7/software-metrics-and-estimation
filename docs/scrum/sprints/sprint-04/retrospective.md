# Sprint 4 — Retrospectiva

> **Documento reconstruido.** No se conservan notas originales de la ceremonia.
> Esta acta se armó el 05/11/2026 a partir del historial de Git, las PRs
> mergeadas y los cambios archivados en `openspec/changes/archive/`. El período
> cubierto se deriva de esos artefactos, no de un calendario de ceremonias.

* **Período cubierto:** 02/11/2026 – 05/11/2026
* **Objetivo de la consigna:** pulir el producto, documentar y preparar la entrega
  final; exportar el informe del proyecto a PDF; realizar el Sprint Review
  Final.
* **Participantes (equipo):** Valentín Rubio, Pablo Geyer, Luciano, Santiago
  Calzolari y Santiago Oses.
* **Formato:** qué mantener / qué mejorar, cierre con acciones.

## Entregado en el Sprint

| Entrega | Evidencia | Fecha |
|---|---|---|
| Flujo de defectos en la TUI (listar, buscar, registrar, cambiar estado, navegar) | PRs #129 a #135 | 03/11 |
| US-18 — Consultar los defectos de un proyecto | PR #136, change `2026-11-03-us-18-consultar-defectos-proyecto` | 03/11 |
| TUI-11 — Dashboard del proyecto | PR #137, change `2026-11-03-tui-11-visualizar-dashboard` | 03/11 |
| US-21 — Reportes PDF de proyecto y Sprint | PR #138, change `2026-11-03-us-21-reporte-pdf-proyecto-sprint` | 03/11 |
| Descarga de reportes PDF desde la TUI | PR #139 | 03/11 |
| TUI-12 — Rediseño de la experiencia de usuario | PR #141, change `2026-11-03-tui-12-rediseno-ux` | 03/11 |
| Acciones desde el detalle de la historia | PR #142, change `2026-11-04-tui-refactor-acciones-desde-historia` | 04/11 |
| TUI-13 — Limpieza de la navegación en la spec | change `2026-11-03-tui-13-limpiar-navegacion-spec` | 03/11 |
| Guía de secciones y ayuda en la TUI | PR #143 (abierta al cierre de esta acta) | 05/11 |
| Informe de métricas y cobertura de pruebas | PR #144 (abierta al cierre de esta acta) | 05/11 |

Trece PRs mergeadas en cuatro días, más dos abiertas al cierre.

## Qué mantener

* **El flujo de defectos se partió en siete PRs de un objetivo cada una.**
  Listado, búsqueda, alta, cambio de estado, navegación y guía entraron por
  separado (#129 a #135). Es el mejor ejemplo del proyecto de PRs chicas y
  revisables.
* **El PDF se entregó de punta a punta.** Dominio, caso de uso, handler HTTP,
  descarga desde la TUI y change archivado (PR #138, #139). El objetivo
  explícito del Sprint 4 quedó cubierto.
* **El Dashboard cerró la visualización prometida.** TUI-11 tiene su change
  archivado y consume las métricas de US-19.
* **Se limpió la spec de navegación.** TUI-13 no agregó funcionalidad: ordenó lo
  que ya existía. Mantener ese hábito evita specs que mienten.

## Qué mejorar

* **Tres pasadas de UX en tres días.** La PR #141 rediseñó la experiencia, la
  #142 cambió las acciones del detalle y la #143 agregó guías y ayuda. Cada una
  mejoró la anterior, pero el orden muestra que la UX no se validó antes de
  empezar el rediseño.
* **La PR #143 rompió el presupuesto de review.** 2.769 líneas cambiadas en 36
  archivos, mergeada con `size:exception`. En la última semana del proyecto es
  cuando más caro sale revisar así.
* **No hay CI.** La PR #143 no tiene ningún check automático. La acción pedida
  en el Sprint 3 no se cumplió y el riesgo se concentró en el cierre.
* **El Sprint Review Final no tiene evidencia.** No hay presentación, guion de
  demo ni registro de la recorrida de trazabilidad que pide la consigna. Es el
  entregable #13 y sigue pendiente.
* **US-17 sigue sin change archivado.** El hueco de trazabilidad detectado en el
  Sprint 3 no se cerró.

## Acciones para la entrega final

| Acción | Evidencia de cumplimiento | Responsable | Plazo |
|---|---|---|---|
| Agregar un workflow mínimo de CI (build, vet, test) | `.github/workflows/` con el workflow y una PR con el check en verde | Por definir | Antes de la entrega |
| Escribir el manual breve de usuario separado del README técnico | `docs/user-manual.md` con instalación, recorrido y preguntas frecuentes | Por definir | Antes de la entrega |
| Preparar la presentación y el guion de demo con el recorrido de trazabilidad | Slides y guion con Historia → SDD → BDD → test → código de una funcionalidad | Por definir | Antes de la entrega |
| Archivar el change de US-17 | `openspec/changes/archive/` con el change de US-17 | Por definir | Antes de la entrega |

## Estado de las acciones del Sprint 3

| Acción | Estado | Evidencia |
|---|---|---|
| Agregar CI que corra build, vet y tests en cada PR | **No cumplida** | No existe `.github/`; la PR #143 no tiene checks |
| Archivar el change de OpenSpec de toda US mergeada | **No cumplida** | `openspec/changes/archive/` no tiene ningún change de US-17 |
| Compilar y correr los tests del código de IA antes de aprobarlo | **Parcial** | El flujo de defectos (#129 a #135) no necesitó correcciones posteriores, pero no hay checklist que lo demuestre |
