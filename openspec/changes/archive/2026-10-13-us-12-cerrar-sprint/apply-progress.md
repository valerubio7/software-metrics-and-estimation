# Apply progress — US-12 cerrar un Sprint

> **Reconstruido a posteriori (2026-10-13).** Durante la implementación no se persistió este archivo. El contenido sale del historial de git del PR #91 (merge `9f33e8d`, rama `feat/us-12-cerrar-sprint-verification`) y de `tasks.md`. No registra evidencia RED/GREEN que no esté en el historial.

## Commits del cambio

| Commit | Mensaje | Contenido | Evidencia TDD en el historial |
|---|---|---|---|
| `4a396af` | `docs(openspec): specify US-12 sprint closure` | proposal, design, spec delta, tasks | No aplica (documentación) |
| `672a4f7` | `feat(sprint): add sprint closure API` | Handler de cierre, repositorio, `api.go` y `main.go` (gate de esquema ≥ 8), y pruebas unitarias (`close_handler_test.go`, `close_sprint_test.go`, `sprint_close_test.go`) | **No demostrable.** Producción y pruebas llegaron en el mismo commit; no hay commit RED separado. |
| `5219a2e` | `test(sprint): verify closure history and concurrency` | Pruebas de integración PostgreSQL (`repository_integration_test.go`), ajuste de `assignment_integration_test.go` (prueba de US-09) y `docs/sprint-closure.md` | Llegó **después** de la producción. No hay RED previo registrado. |

## Estado de tareas (`tasks.md`)

| Tarea | Estado | Evidencia |
|---|---|---|
| 1.1 – 1.4 Caso de uso y persistencia | Marcadas | `672a4f7`, `5219a2e` |
| 2.1 – 2.3 Endpoint HTTP y gate de esquema | Marcadas | `672a4f7` |
| 3.1 Documentación del endpoint | Marcada | `5219a2e` (`docs/sprint-closure.md`) |
| 3.2 `go test ./...` | Marcada | Ver `archive-report.md`, sección de verificación |

## Ajuste posterior a la archivación

- El delta de la spec se fusionó en `openspec/specs/sprint/spec.md` con los encabezados en español del resto de la spec (`### Requisito:`, `#### Escenario:`).
