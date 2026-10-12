# Tasks

## 1. Config API_URL (TDD)

- [x] 1.1 RED: crear `tests/unit/tui/config_test.go` con default `http://localhost:8080`, env explícita y trim de `/` final; confirmar que falla sin `internal/tui/config.go` y verificar con `go test ./tests/unit/tui/`.
- [x] 1.2 GREEN: crear `internal/tui/config.go` con `LoadConfig(getenv)` que aplica default y trim; verificar con `go test ./tests/unit/tui/`.
- [x] 1.3 REFACTOR: limpiar config sin cambiar contrato y verificar con `go test ./tests/unit/tui/`.

## 2. Navegación base (TDD)

- [x] 2.1 RED: crear `tests/unit/tui/model_test.go` con menú de 4 opciones, mover cursor, entrar a placeholder y volver; confirmar que falla sin model y verificar con `go test ./tests/unit/tui/`.
- [x] 2.2 GREEN: crear `internal/tui/model.go` con BubbleTea Model/Update/View puro y placeholders no mutantes; verificar con `go test ./tests/unit/tui/`.
- [x] 2.3 RED+GREEN api-unavailable: ampliar tests con `httptest.Server` ok vs servidor caído; View muestra el motivo sin crash; verificar con `go test ./tests/unit/tui/`.
- [x] 2.4 RED+GREEN salida: cubrir `q`/`ctrl+c`/opción salir hacia `tea.Quit`; verificar con `go test ./tests/unit/tui/`.
- [x] 2.5 REFACTOR: ajustar model manteniendo tests verdes y verificar con `go test ./tests/unit/tui/` y `go vet ./internal/tui/`.

## 3. Binario e integración

- [x] 3.1 GREEN: crear `cmd/tui/main.go` que compone config+model con `tea.NewProgram` y agregar `bubbletea` a `go.mod`; verificar con `go build ./...`.
- [ ] 3.2 Verificación global: ejecutar `go test ./...`, `go vet ./...`, `openspec validate tui-01-open-navigate` y manual `API_URL=... go run ./cmd/tui`; verificar sin regresiones.

## Workflow follow-up

- Abrir PR `feat/tui-01-open-navigate` → `main`, link `Closes #83`.
- Archivar el change tras el merge.
