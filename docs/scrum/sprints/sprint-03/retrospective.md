# Sprint 3 — Retrospectiva

> **Documento reconstruido.** No se conservan notas originales de la ceremonia.
> Esta acta se armó el 05/11/2026 a partir del historial de Git, las PRs
> mergeadas y los cambios archivados en `openspec/changes/archive/`. El período
> cubierto se deriva de esos artefactos, no de un calendario de ceremonias.

* **Período cubierto:** 21/10/2026 – 01/11/2026
* **Objetivo del Sprint (consigna):** añadir funcionalidades clave y robustecer
  el sistema; implementar la visualización.
* **Participantes (equipo):** Valentín Rubio, Pablo Geyer, Luciano, Santiago
  Calzolari y Santiago Oses.
* **Formato:** qué mantener / qué mejorar, cierre con acciones.

## Entregado en el Sprint

| Entrega | Evidencia | Fecha |
|---|---|---|
| TUI-06 — Navegación de las secciones del Sprint 3 | PR #106, change `2026-10-25-tui-06-navegacion-sprint-3` | 21/10 |
| US-14 — Estimación con Planning Poker en la TUI | PR #107 | 23/10 |
| US-15 — Registrar esfuerzo | PR #109, change `2026-10-25-us-15-registrar-esfuerzo` | 25/10 |
| US-16 — Registrar defecto | PR #110, change `2026-10-25-us-16-registrar-defecto` | 25/10 |
| US-17 — Ciclo de vida del defecto | PRs #111, #112, #113 y #116 — **sin change archivado** | 26/10 – 27/10 |
| US-19 — Métricas de proyecto y Sprint | PRs #120, #121, #122, #123 y #125, change `2026-10-26-us-19-consultar-metricas-proyecto-sprint` | 27/10 – 30/10 |
| TUI-10 — Consultar métricas desde la TUI | PR #115, corregida en #124 | 30/10 |
| TUI-08 — Registrar esfuerzo desde la TUI | PR #128, change `2026-10-31-tui-08-registrar-esfuerzo` | 31/10 |
| Corrección de migración duplicada en pruebas de listado | PR #126, change `2026-10-30-fix-us-18-migracion-duplicada-pruebas` | 30/10 |

Veintidós PRs mergeadas en doce días.

## Qué mantener

* **US-19 se partió por unidad de trabajo, no por capricho.** Cuatro PRs
  explícitas (`1/4` dominio y cobertura, `2/4` consulta y validación, `3/4`
  snapshot PostgreSQL, `4/4` endpoints) más una de consolidación. Cada una se
  podía revisar sola y el historial dice qué se hizo en cada paso.
* **La integración se probó contra PostgreSQL real.** Los repositorios de
  métricas y defectos sumaron pruebas de integración con esquemas aislados, no
  solo dobles de prueba.
* **El archivado SDD se sostuvo en casi todas las historias.** US-15, US-16,
  US-19, TUI-06 y TUI-08 tienen su change con proposal, design, tasks y
  verificación.
* **Las métricas exponen la calidad del dato.** En lugar de rellenar con cero,
  la API devuelve `null` y un bloque `coverage` con los faltantes. Es una
  decisión de diseño defendible y quedó documentada en el README.

## Qué mejorar

* **Se mergeó código que no compilaba.** La PR #115 (TUI-10, generada por un
  agente) entró a `main` y hubo que corregirla en `040c2a8 test(tui): fix
  metrics flow test compilation` y en la PR #124. Sin CI, nada lo detectó antes
  del merge.
* **Los tests de integración quedaron rojos en `main`.** La PR #126 corrigió una
  migración duplicada en las pruebas de listado de defectos, detectada después
  del merge.
* **Un archivo de proceso quedó versionado por error.** La PR #127 lo tuvo que
  sacar del repositorio.
* **US-17 quedó fuera del archivado SDD.** Se implementó en cuatro PRs
  (#111, #112, #113, #116), pero no existe `openspec/changes/archive/*us-17*`.
  La trazabilidad Historia → SDD tiene un hueco justo en la funcionalidad que
  más cambió de estado.
* **La deuda de integración se repitió.** La PR #116 se llamó literalmente
  `fix(defect): integrate US-17 lifecycle into main`: otra vez el trabajo por
  historia no garantizó el recorrido completo.

## Acciones para Sprint 4

| Acción | Evidencia de cumplimiento | Responsable | Plazo |
|---|---|---|---|
| Agregar CI que corra `go build ./...`, `go vet ./...` y `go test ./...` en cada PR | Workflow en `.github/workflows/` y check verde en la PR | Por definir en planning | Primera PR del Sprint 4 |
| Archivar el change de OpenSpec de toda US mergeada, sin excepción | `openspec/changes/archive/` con el change de US-17 | Por definir en planning | Al cerrar cada US |
| Compilar y correr los tests del código generado por IA antes de aprobarlo | Checklist de review con el comando ejecutado y su resultado | Por definir en planning | Cada PR con código de IA |

## Estado de las acciones del Sprint 2

| Acción | Estado | Evidencia |
|---|---|---|
| Correr build y tests sobre el árbol integrado antes de mergear | **No cumplida** | PR #115 no compilaba; PR #126 dejó tests rojos en `main` |
| Validar una sección de la TUI con el cliente antes de replicar el patrón | **Sin evidencia** | No hay nota de review ni cambio de rumbo atribuible a validación |
| Registrar el orden de integración en la issue | **No cumplida** | PR #116 vuelve a integrar tarde el ciclo de vida de US-17 |
