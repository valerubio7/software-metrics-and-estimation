# Progreso de aplicación: US-06 — Modificar una historia del Product Backlog

Modo: TDD estricto (`go test ./...`). Estrategia de entrega: `ask-on-risk`, cadena `stacked-to-main`.
Rama del lote: `feat/us06-domain-story-update` (base `main`). No se hizo push ni PR.

## Estado acumulado de tareas

| Slice | Tareas | Estado |
|-------|--------|--------|
| 0. Planificación | 0.1, 0.2 | Completas (commit `46392a5`) |
| 1. Dominio | 1.1 a 1.8 | Completas (commit `fab506d`) |
| 2. Aplicación | 2.1 a 2.8 | Pendientes |
| 3. Almacenamiento | 3.1 a 3.16 | Pendientes (requiere Docker) |
| 4. Handler HTTP | 4.1 a 4.11 | Pendientes |
| 5. Composición y docs | 5.1 a 5.11 | Pendientes |

## Lote 1: Slice 0 y Slice 1 (dominio)

### Línea base (tarea 0.1)

`go test ./...` sobre `main` limpio, antes de tocar nada: los paquetes unitarios pasan
(`tests/unit/cmd/api`, `project/*`, `story/*`). Los paquetes de integración
(`tests/integration/project/postgres`, `tests/integration/story/postgres`) **FALLAN** (no se saltan) con
`rootless Docker is not supported on Windows, failed to create Docker provider`: no hay Docker en esta
máquina. Es una falla ambiental preexistente y **no es evidencia de persistencia**.

### Evidencia del ciclo TDD

| Tarea | Archivo de test | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-----------------|------|------------------|-----|-------|-------------|----------|
| 1.1 | `tests/unit/story/domain/story_test.go` (`TestNewStoryUpdate`) | Unitario | 2/2 tests de `NewStory` en verde | Escrito; falla observada: `undefined: domain.NewStoryUpdate` (error de compilación) | Pase observado | Ver 1.5 | Ver 1.6 |
| 1.2 | idem (`TestNewStoryUpdateEstimatedHours`) | Unitario | N/A (nuevo) | Escrito; misma falla de compilación observada | Pase observado | Ver 1.5 | Ver 1.6 |
| 1.3 | idem (`TestNewStoryUpdatePreservesTextOrderAndIdentity`, `TestNewStoryUpdateCopiesCriteria`, `TestNewStoryInitializesWithoutEstimatedHours`, `TestAllowedStatusesReturnsTheClosedSetAsACopy`) | Unitario | N/A (nuevo) | Escrito; falla de compilación: `Story.EstimatedHours`, `domain.StatusPending`, `domain.AllowedStatuses` indefinidos | Pase observado | Ver 1.5 | Ver 1.6 |
| 1.4 | `internal/story/domain/story.go` | Unitario | N/A | Cubierto por 1.1-1.3 | `go test ./tests/unit/story/domain/...` PASS | — | — |
| 1.5 | idem | Unitario | — | — | — | Bordes `0.01`, `99999.99`, `3`, `2.5`, `0.07`, blancos Unicode en criterios, copia de la estimación y acumulación de seis campos inválidos: pasaron de inmediato (caracterización; la lógica de 1.4 ya era general, no hubo RED nuevo) | — |
| 1.6 | `internal/story/domain/story.go` | Unitario | Suite de `NewStory` verde antes y después | — | — | — | `validateStoryContent` extraída y compartida; `"pendiente"` sustituido por `StatusPending`; `go test ./tests/unit/story/domain/...` PASS |

Resumen de tests: 12 funciones de test de nivel superior en el paquete de dominio (51 subtests/casos con
`-v`), todas en verde. Capa usada: solo unitaria. Funciones puras creadas: `NewStoryUpdate`,
`validateStatus`, `validateEstimatedHours`, `decimalPlaces`, `validateStoryContent`, `AllowedStatuses`.

### Evidencia del work unit

| Evidencia | Valor observado |
|-----------|-----------------|
| Comando de test focalizado | `go test ./tests/unit/story/domain/...` -> `ok` (todas las funciones PASS) |
| Harness de runtime | N/A: funciones puras de dominio sin frontera de proceso; evidencia = suite unitaria + `go test ./...` |
| `go vet ./...` | Limpio |
| `gofmt` | Limpio en los archivos tocados. `gofmt -l .` lista casi todos los `.go` del repo porque en esta copia de trabajo de Windows (`core.autocrlf=true`) los archivos están en CRLF; es preexistente y no es un problema de formato |
| `go test ./...` completo | Paquetes unitarios `ok`; paquetes de integración FALLAN por falta de Docker, idéntico a la línea base de `main` |
| Rollback | Revertir `fab506d` (`internal/story/domain/story.go`, `tests/unit/story/domain/story_test.go`); el commit de planificación `46392a5` se revierte por separado |

### Commits

- `46392a5` `docs(sdd): add US-06 planning artifacts for story modification` (solo `openspec/changes/us-06-.../`, 5 archivos, 2000 líneas).
- `fab506d` `feat(story): add domain validation for story updates` (2 archivos, +326/-14 = 340 líneas autoradas, dentro del presupuesto de 400).

### Desviaciones del diseño

Ninguna en el comportamiento. Detalle menor: se agregaron las constantes no exportadas `maxEstimatedHours`,
`maxEstimatedHoursDecimal` y `estimatedHoursMessage` y el helper `decimalPlaces` para no repetir literales;
`NewStoryUpdate` copia el puntero de la estimación para no aliasar el valor del llamador (cubierto por
`TestNewStoryUpdateCopiesEstimatedHours`).

### Problemas encontrados

- Los tests de integración fallan (no se saltan) sin Docker; ver la línea base. El slice 3 y el arranque real del slice 5 seguirán sin poder verificarse en esta máquina.
- Los archivos `tasks.md` y `apply-progress.md` quedaron **sin commitear** en este lote: `tasks.md` ya está incluido sin marcas en el commit de planificación y la plantilla de commits del slice no pide incluir las marcas `[x]`. Se reportan al orquestador para decidir si se agrupan en un commit `docs(sdd)` posterior.

### Tareas restantes

Slices 2 a 5 (ramas `feat/us06-application-update-story`, `feat/us06-storage-estimated-hours`,
`feat/us06-http-update-handler`, `feat/us06-wire-update-route-docs`).
