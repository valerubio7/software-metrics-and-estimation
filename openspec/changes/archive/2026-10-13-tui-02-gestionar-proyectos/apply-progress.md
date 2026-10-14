# Apply progress — TUI-02 gestionar proyectos desde la TUI

> **Reconstruido a posteriori (2026-10-13).** Durante la implementación no se persistió este archivo. El contenido sale del historial de git del PR #92 (merge `111f446`, rama `feat/tui-02-proyectos`) y de `tasks.md`. No registra evidencia RED/GREEN que no esté en el historial.

## Entrega

Un único PR con `size:exception`, decidido por el usuario: diff acumulado de 723 líneas cambiadas frente al presupuesto de 400 (registrado en `tasks.md`, commit `ba3f49a`).

## Commits del cambio

| Commit | Mensaje | Contenido | Evidencia TDD en el historial |
|---|---|---|---|
| `011ca99` | `docs(openspec): specify TUI project management` | proposal, design, spec delta, tasks, README, y **cambio directo a `openspec/specs/tui/spec.md`** | No aplica. Ver nota de la spec principal abajo. |
| `37d93ca` | `feat(tui): add project API client` | `internal/tui/projects.go` y `tests/unit/tui/api_test.go` | **No demostrable.** Producción y pruebas en el mismo commit. |
| `0456f74` | `feat(tui): manage projects from terminal UI` | `internal/tui/model.go` (+321/−67) y `tests/unit/tui/model_test.go` | **No demostrable.** Producción y pruebas en el mismo commit. |
| `ba3f49a` | `docs(openspec): record TUI-02 delivery exception` | Registro de `size:exception` en `tasks.md` | No aplica |

## Estado de tareas (`tasks.md`)

| Tarea | Estado | Evidencia |
|---|---|---|
| 1.1 Spec: creación y validación 422 | Marcada | `011ca99` |
| 1.2 Spec: modificación, estado y 404 | Marcada | `011ca99` |
| 2.1 Cliente HTTP de proyectos | Marcada | `37d93ca` |
| 2.2 Formularios y navegación Bubble Tea | Marcada | `0456f74` |
| 2.3 Errores, regresiones y suite completa | Marcada | Ver `archive-report.md`, sección de verificación |

## Hallazgo de proceso

- **`011ca99` modificó `openspec/specs/tui/spec.md` antes del archivado.** La spec principal solo debe cambiar al archivar. Ese cambio eliminó el requisito de TUI-01 "Navegación de solo lectura" sin un `REMOVED` ni `MODIFIED` en el delta, y dejó un requisito con un título distinto al del delta. Se corrigió al archivar: la spec principal volvió a su versión previa (`011ca99^`) y se aplicó el delta.
