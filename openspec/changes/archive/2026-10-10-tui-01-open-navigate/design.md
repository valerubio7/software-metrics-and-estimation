# Design

## Context

Ver `proposal.md` para motivación. Estado actual: API sin health-check dedicado; la TUI solo necesita detectar disponibilidad sin mutar datos. El repo no tiene `cmd/tui` ni dependencia TUI. Tests del proyecto: `go test ./...`, unitarios sin Docker.

## Goals / Non-Goals

**Goals:**

- Binario `cmd/tui` mínimo y navegable.
- Config de conexión vía `API_URL`.
- Modelo BubbleTea testeable sin TTY real.
- Error de conexión visible sin cierre abrupto.

**Non-Goals:**

- Pantallas funcionales (TUI-02 a TUI-05).
- Cliente API completo, persistencia o validaciones de negocio en la TUI.

## Decisions

- Estructura `internal/tui/config.go` (LoadConfig) + `internal/tui/model.go` (Model/Update/View puros) + `cmd/tui/main.go` fino (solo compone y corre `tea.NewProgram`).
  - Alternativa `cmd/tui` monolítico descartada: acopla IO y dificulta tests unitarios.
- `API_URL` con default `http://localhost:8080` y trim de `/` final; chequeo lazy con timeout 2s.
  - El mux actual no tiene `GET /` ni `/health`; cualquier respuesta HTTP se trata como disponible, solo error de red/timeout como no disponible (no muta datos).
- Única dependencia nueva: `charmbracelet/bubbletea`. Sin `bubbles` en TUI-01.
- Estilo con `charmbracelet/lipgloss` en `internal/tui/styles.go`: paleta adaptativa (claro/oscuro) con roles acento/texto/atenuado/ok/error/ámbar; cada estado lleva cue de texto además de color. Enmienda posterior a TUI-01: no cambia comportamiento ni spec, solo `View`.
- Placeholders: un estado por opción (proyectos/backlog/sprints) que solo muestra "Disponible en TUI-0X", sin requests mutantes.

## Risks / Trade-offs

- [Sin endpoint health real → falso negativo si el mux responde 404] → Mitigación: tratar cualquier respuesta HTTP como disponible.
- [Tests de TTY frágiles] → Mitigación: testear solo `Update`/`View` puros + `httptest.Server`, sin PTY.

## Migration Plan

Sin migración. Rollback: borrar `cmd/tui`, `internal/tui`, revertir `go.mod`/`go.sum` y `tests/unit/tui`.

## Open Questions

- Ninguna que bloquee specs, enfoque o tareas.
