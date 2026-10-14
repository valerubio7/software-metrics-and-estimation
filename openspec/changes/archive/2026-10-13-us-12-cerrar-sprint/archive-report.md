# Reporte de archivo — US-12 cerrar un Sprint

**Fecha de archivo**: 2026-10-13
**Estado**: implementación completa (9/9 tareas). Artefactos SDD incompletos, ver tabla.
**Implementación**: PR #91, rama `feat/us-12-cerrar-sprint-verification`, merge `9f33e8d`.
**Corrección de archivo**: rama `docs/sdd-archive-us12-tui02`.

**Escrito para**: revisores del equipo y futuros archivos del proyecto.

## Resumen

US-12 permite cerrar un Sprint abierto mediante `POST /projects/{project_id}/sprints/{sprint_id}/closure`, conservando su historial. El PR se mergeó sin archivar el cambio, lo que incumple la regla de archivar al terminar. Este reporte regulariza ese paso.

## Tareas

9/9 marcadas en `tasks.md`: 1.1–1.4 (caso de uso y persistencia), 2.1–2.3 (endpoint HTTP y gate de esquema), 3.1–3.2 (documentación y verificación). El detalle por commit está en `apply-progress.md`.

## Specs fusionadas

Se agregaron 7 requisitos (ADDED) a `openspec/specs/sprint/spec.md`, con los encabezados en español de esa spec (`### Requisito:`, `#### Escenario:`):

1. Cerrar un Sprint abierto
2. Validar identificadores y pertenencia antes del cierre
3. Rechazar el cierre repetido
4. Conservar la información histórica
5. Serializar el cierre frente a asignaciones concurrentes
6. Habilitar el cierre solo con el esquema requerido
7. Manejar métodos no admitidos y fallos de almacenamiento

## Artefactos SDD

| Artefacto | Estado |
|---|---|
| `proposal.md` | Presente |
| `design.md` | Presente |
| `specs/sprint/spec.md` (delta) | Presente |
| `tasks.md` | Presente |
| `apply-progress.md` | Reconstruido a posteriori desde git. No se persistió durante la implementación. |
| `exploration.md` | **Faltante.** No existe en el repo ni en Engram. |
| `verify-report.md` | **Faltante.** No se ejecutó `sdd-verify`. |
| `archive-report.md` | Este documento. |

Retirados en esta corrección para seguir el formato de US-01 a US-11:

- `.openspec.yaml`: `schema: spec-driven`, creado 2026-10-12, objetivo "Permitir finalizar formalmente un Sprint sin alterar sus resultados históricos."
- `README.md`: "Cerrar un Sprint conservando su historial y bloqueando nuevas incorporaciones".

## Verificación

Corrida del 2026-10-14 con Docker 29.8.1, `go test -p 1 -count=1 -timeout 30m ./...`:

- `go vet ./...`: sin hallazgos.
- Paquetes de US-12 (`tests/unit/sprint/*`, `tests/unit/cmd/api`, `tests/integration/sprint/postgres`): ok.
- `tests/integration/story/postgres`: falla `TestUS09SchemaRequiresCompleteMigration` con `unexpected EOF` al conectar tras "ready". Es la carrera de readiness del contenedor, no una aserción. Al correrlo aislado pasa (6.78 s).

Una corrida anterior del mismo día no es válida: el motor de Docker estaba caído (`500 Internal Server Error`), y no se usa como evidencia.

## Hallazgos

- `672a4f7` incluye producción y pruebas unitarias en el mismo commit. Por eso no hay un commit RED separado en el historial.
- `5219a2e` modifica `assignment_integration_test.go`, una prueba de US-09. Conviene revisar en el próximo cambio que su intención se mantuvo.
