# Apply progress — TUI-01 abrir TUI y navegar base

> **Reconstruido a posteriori (2026-10-13).** Durante la implementación no se persistió este archivo. El contenido sale del historial de git del PR #88 (merge `27f6c9c`, rama `feat/tui-01-open-navigate`) y de `tasks.md`. La traza RED/GREEN sí aparece en los mensajes de commit.

## Ciclo TDD por commit (orden cronológico)

| Commit | Mensaje | Fase | Tarea |
|---|---|---|---|
| `9457e57` | `test(tui): RED config API_URL con default y trim` | RED | 1.1 |
| `cb402ba` | `feat(tui): GREEN config API_URL con default localhost` | GREEN | 1.2 |
| `4467763` | `test(tui): RED navegacion base con menu y salida` | RED | 2.1 |
| `bfad098` | `feat(tui): GREEN navegacion base BubbleTea sin mutar datos` | GREEN | 2.2 (la 2.4 se infiere: el mensaje no la nombra) |
| `2d8d93d` | `test(tui): RED api-unavailable con probe y mensaje` | RED | 2.3 |
| `ca25c0d` | `feat(tui): GREEN api-unavailable con probe y mensaje` | GREEN | 2.3 |
| `cae8a21` | `feat(tui): GREEN binario cmd/tui con config y probe` | GREEN | 3.1 |
| `7f875ab` | `docs(sdd): tui-01 tasks 1.1-3.1 completas, 3.2 verificada` | Documentación | — |
| `0ddb267` | `docs(sdd): tui-01 enmienda design con lipgloss` | Documentación | — |
| `35dfcbd` | `style(tui): restyle base con lipgloss y paleta adaptativa` | Estilo | — |
| `6d154cc` | `docs(sdd): archive tui-01-open-navigate` | Archivado | — |

## Notas

- **Tareas sin commit propio:** 1.3 (REFACTOR de config), 2.5 (REFACTOR de model) y 2.4 (salida) no tienen commit separado; su evidencia es la de los commits de GREEN que las contienen.
- **Discrepancia en la tarea 3.2:** el commit `7f875ab` dice "3.2 verificada", pero `tasks.md` la deja sin marcar (`[ ]`). Se conserva sin marcar porque la verificación global incluye `openspec validate` y una ejecución manual de `go run ./cmd/tui`, y no hay registro de que ambas se hayan ejecutado. Ver `archive-report.md`.
- **Cambio de diseño no previsto en las tareas originales:** `0ddb267` y `35dfcbd` añaden el estilo con lipgloss. Quedó documentado como enmienda de `design.md`.
