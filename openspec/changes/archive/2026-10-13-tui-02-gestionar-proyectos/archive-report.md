# Reporte de archivo — TUI-02 gestionar proyectos desde la TUI

**Fecha de archivo**: 2026-10-13
**Estado**: implementación completa (5/5 tareas). Artefactos SDD incompletos, ver tabla.
**Implementación**: PR #92, rama `feat/tui-02-proyectos`, merge `111f446`. Issue #84.
**Corrección de archivo**: rama `docs/sdd-archive-us12-tui02`.

**Escrito para**: revisores del equipo y futuros archivos del proyecto.

## Resumen

TUI-02 agrega a la TUI la creación, modificación y consulta de estado de proyectos, consumiendo la API existente sin lógica de negocio. El PR se mergeó sin archivar el cambio, lo que incumple la regla de archivar al terminar.

## Entrega

Un único PR con `size:exception`, decidido por el usuario. El diff acumulado fue de 723 líneas cambiadas frente al presupuesto de 400 (registrado en `tasks.md`, commit `ba3f49a`).

## Tareas

5/5 marcadas en `tasks.md`: 1.1–1.2 (especificación), 2.1 (cliente HTTP), 2.2 (formularios y navegación), 2.3 (errores, regresiones y suite completa).

## Specs fusionadas

Sobre la spec principal de TUI, estado previo a este cambio:

- **ADDED**: "Operar proyectos mediante la TUI", con 7 escenarios (crear, rechazo 422, modificar, 404 al modificar, consultar estado, 404 al consultar, error de transporte).
- **MODIFIED**: "Mostrar menú base de navegación". La opción proyectos da acceso a la gestión; backlog y sprints siguen sin pantallas.
- **MODIFIED**: "Navegación de solo lectura". La navegación no muta datos; solo la creación y modificación de proyectos pueden hacerlo.

**Corrección de proceso:** el commit `011ca99` había editado la spec principal durante la fase de especificación. Esa edición eliminó el requisito de TUI-01 "Navegación de solo lectura" sin un `MODIFIED` ni `REMOVED` en el delta, y dejó un requisito con un título distinto ("Gestionar proyectos desde la TUI"). Se corrigió restaurando la spec a su versión previa (`011ca99^`) y aplicando el delta correctamente.

**Decisión de contenido:** el texto de los dos `MODIFIED` es una lectura propia del cambio, no algo que el usuario haya validado. Si no coincide con la intención, se ajusta en el delta.

## Artefactos SDD

| Artefacto | Estado |
|---|---|
| `proposal.md` | Presente |
| `design.md` | Presente |
| `specs/tui/spec.md` (delta) | Presente. Se corrigió en esta corrección (escenarios en `GIVEN/WHEN/THEN`, `MODIFIED` agregados). |
| `tasks.md` | Presente |
| `apply-progress.md` | Reconstruido a posteriori desde git. No se persistió durante la implementación. |
| `exploration.md` | **Faltante.** No existe en el repo ni en Engram. |
| `verify-report.md` | **Faltante.** No se ejecutó `sdd-verify`. |
| `archive-report.md` | Este documento. |

Retirados en esta corrección para seguir el formato de US-01 a US-11:

- `.openspec.yaml`: `schema: spec-driven`, sin fecha de creación.
- `README.md`: "Cambio OpenSpec para implementar la issue #84" y la nota de artefactos en español.

## Verificación

Corrida del 2026-10-14 con Docker 29.8.1, `go test -p 1 -count=1 -timeout 30m ./...`:

- `go vet ./...`: sin hallazgos.
- `tests/unit/tui`: ok.
- Resto de la suite: ok, salvo la carrera de readiness documentada en el reporte de US-12 (`TestUS09SchemaRequiresCompleteMigration`, que pasa aislado).

Una corrida anterior del mismo día no es válida: el motor de Docker estaba caído.
