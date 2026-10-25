# Reporte de archivo — TUI-06 navegación del Sprint 3 en la TUI

**Fecha de archivo**: 2026-10-25
**Estado al cierre**: implementación completa y fusionada en `main` (PR #106, merge `820bfac`, 2026-10-21; issue #105 cerrado). Verificación no ejecutada en este cierre: no existe `verify-report.md`. Sesión manual interactiva de la TUI no realizada.
**Implementación**: rama `feat/tui-06-navigation-sections`, 10 commits (RED/GREEN por unidad, especificación y documentación) integrados mediante el merge `820bfac`.

**Escrito para**: revisores del equipo y futuros archivos del proyecto.

## Resumen

TUI-06 es la historia habilitante del Sprint 3. Amplía el menú de la TUI de cuatro a nueve opciones y rutea las secciones nuevas (Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard) a su pantalla principal de navegación. Incorpora un selector compartido con filtro, cursor y estados de carga, error y vacío, y agrega dos lecturas de solo lectura en la API: `GET /projects/{project_id}/members` y `GET /projects/{project_id}/stories/{story_id}/tasks`. No implementa los flujos de negocio de las secciones, que quedan para TUI-07 a TUI-11, y no emite requests que modifiquen datos durante la navegación.

## Entrega

- Un único PR (#106) con `size:exception`, aprobado por el usuario antes de aplicar (`delivery_strategy: single-pr`).
- Pronóstico de `tasks.md`: 1.200 a 1.600 líneas modificadas. Diff medido del merge frente a su base `64cbd93`: 34 archivos, 3.552 líneas añadidas y 25 eliminadas (3.577 en total). Supera el presupuesto de 400 líneas; se mantiene la excepción aprobada y no se recortaron pruebas ni documentación.
- Sin migraciones nuevas: las tablas `project_members` (000006) y `tasks` (000009) ya existían.

Commits de la rama, en orden cronológico:

| # | Hash | Mensaje |
|---|---|---|
| 1 | `3dc4a8f` | docs(openspec): specify Sprint 3 TUI navigation |
| 2 | `5fdec00` | test(api): cover project member listing |
| 3 | `78285cc` | feat(api): list project members |
| 4 | `f6fc85d` | test(api): cover story task listing |
| 5 | `8adc53b` | feat(api): list tasks for a story |
| 6 | `98fadf6` | test(tui): cover shared selector and section clients |
| 7 | `3a10d4d` | feat(tui): add shared selector and section clients |
| 8 | `42da4ad` | test(tui): cover Sprint 3 menu and section routing |
| 9 | `230919c` | feat(tui): navigate Sprint 3 section shells |
| 10 | `b7be8a4` | docs(tui): document Sprint 3 navigation |

Merge: `820bfac` Merge pull request #106 from valerubio7/feat/tui-06-navigation-sections.

## Alcance entregado

- Menú de nueve opciones: Proyectos, Backlog, Sprints, Planning Poker, Esfuerzo, Defectos, Métricas, Dashboard y Salir.
- Ruteo directo de cada sección nueva a su pantalla principal, sin submenús de operaciones. Al elegir un elemento se muestra el selector siguiente y, al completar el recorrido, un aviso de que el flujo de negocio llega en TUI-07 a TUI-11.
- Selector compartido para integrantes, historias, tareas y Sprints, con `/`, `↑/↓`, `Enter`, `Esc`, `r` y `?`.
- Sin proyecto activo, la sección abre el navegador de proyectos existente y vuelve a la sección al confirmar.
- Lecturas nuevas de la API: integrantes (orden por nombre; `404 project_not_found`, `422` para UUID inválido) y tareas (orden por secuencia; `404 project_not_found` y `story_not_found`, `422` para UUID inválido). Ambas rutas se componen solo si su dependencia está presente.
- Clientes TUI: `MemberClient.List` y `TaskClient.ListByStory`.

## Specs fusionadas

Composición con `gentle-ai sdd-archive-compose` (v3.7.0), una invocación por delta. Las tres terminaron con salida 0; la escritura fue atómica mediante `.compose-tmp` seguido de `mv`.

| Dominio | Requisitos antes | Delta | Requisitos después |
|---|---|---|---|
| `tui` | 24 | 3 MODIFIED, 3 ADDED | 27 |
| `project` | 16 | 1 ADDED | 17 |
| `tarea` | 9 | 1 ADDED | 10 |

- `tui` MODIFIED: «Mostrar menú base de navegación», «Mostrar la ayuda de teclado», «Navegación de solo lectura».
- `tui` ADDED: «Abrir las secciones del Sprint 3 desde el menú», «Elegir integrantes, historias, tareas y Sprints con selectores navegables», «Avisar el límite de las pantallas de navegación».
- `project` ADDED: «Consultar los integrantes de un proyecto».
- `tarea` ADDED: «Consultar las tareas de una historia».

Readback contra la instantánea previa a la composición: en `tui`, las 7 líneas eliminadas corresponden únicamente al texto anterior de los tres requisitos MODIFIED; el resto son adiciones. En `project` y `tarea` solo hubo adiciones. Los requisitos no mencionados en los deltas se conservaron.

## Artefactos del cambio

| Artefacto | Estado |
|---|---|
| `proposal.md` | Presente |
| `specs/tui/spec.md`, `specs/project/spec.md`, `specs/tarea/spec.md` (deltas) | Presentes; fusionados en las specs principales |
| `design.md` | Presente |
| `tasks.md` | Presente; 62 casillas `[x]` y 0 `[ ]`. Bytes originales preservados |
| `apply-progress.md` | Presente; escrito durante la implementación |
| `verify-report.md` | Ausente. `sdd-verify` no se ejecutó |
| `archive-report.md` | Este documento |

## Verificación

- **Automática**: no ejecutada en este cierre. El `apply-progress.md`, escrito durante la implementación, registra `go test ./...` con 48 paquetes `ok` y 0 `FAIL`, y la suite unitaria en verde. Esas cifras corresponden al momento de su escritura y no se volvieron a ejecutar al archivar.
- **Manual**: no realizada. `apply-progress.md` indica que no se hizo una sesión manual interactiva de la TUI; no se encontró evidencia posterior.

## Auditoría mecánica

- Composición: tres invocaciones de `gentle-ai sdd-archive-compose --canonical <spec> --delta <delta> --output <spec>.compose-tmp && mv`, con salida 0.
- Antes de componer: no hay bytes CR en las tres specs principales ni en los archivos del cambio.
- Movimiento: `git mv openspec/changes/tui-06-navegacion-sprint-3 openspec/changes/archive/2026-10-25-tui-06-navegacion-sprint-3`.
- `diff -r` contra la instantánea previa al movimiento: salida vacía.
- `tasks.md` archivado: idéntico byte a byte a la versión registrada en `HEAD`.
- `openspec/config.yaml`: no modificado.

## Notas residuales

1. **Prueba manual pendiente**: no hay registro de una sesión manual interactiva de la TUI.
2. **Etiquetas de commits en `tasks.md`**: la fase 5 nombra los commits 9 (`test(tui): cover Sprint 3 selector flows`) y 10 (`feat(tui): add Sprint 3 selector flows`), que no existen en el historial. La nota de esa fase indica que los flujos se cubrieron en `42da4ad` y `230919c`. `apply-progress.md` numera los commits de otra forma (9 = `230919c`, 10 = `b7be8a4`). Ambas numeraciones se registran sin resolver; `tasks.md` no se modificó.
3. **Casillas de la propuesta**: `proposal.md` conserva 9 casillas `- [ ]` sin marcar en «Criterios de éxito». No se repararon, porque el archivo no reescribe casillas.
4. **Cambio posterior no relacionado**: TUI-07 (Planning Poker) se integró en `main` por PR #107 (merge `8307895`, 2026-10-23, commit `dda9f43`). Es un cambio distinto y no forma parte del alcance de este reporte.
5. **Excedente de presupuesto**: el diff supera las 400 líneas; la excepción `size:exception` se aprobó antes de aplicar.
