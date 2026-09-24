# Progreso de aplicación: US-05

## Corte 1 — planificación base

- Base: `main` en `b99eb58`; rama local `docs/us05-plan-base`; entrega encadenada hacia `main`, sin push ni PR.
- Tarea 1: revisión de exploración, propuesta, especificación y diseño; casilla de `tasks.md` a marcar solo tras observar el commit y sus comprobaciones. Tareas 2–10 pendientes.
- Archivos del corte: `exploration.md`, `proposal.md`, `specs/historia/spec.md`, `design.md` y este progreso mínimo. `tasks.md` queda fuera del commit (corte 2).
- Coherencia: los cuatro documentos distinguen decisiones históricas de exploración/propuesta de contrato posterior; proyecto conocido, prioridad exacta, criterios no vacíos, estado `pendiente`, puntos `null`, FK sin historias huérfanas. Las notas anteriores sobre estrategia pendiente no invalidan la decisión posterior registrada en `tasks.md`.
- Referencias comprobadas: `internal/api/api.go` registra solo `POST /projects`; existen la migración `000001` y las rutas de código y pruebas de US-01 citadas.
- Comprobaciones previas al commit: estructura de documentos y rutas revisadas; `git diff --cached --check` sin errores; `git diff --cached --numstat`: cinco archivos, +241/−0 = 241 líneas (incluido este progreso), dentro del máximo de 400. No existe `POST /projects/{project_id}/stories` en `internal/api/api.go`.
- TDD estricto: RED/GREEN/TRIANGULATE/REFACTOR y `go test ./...` no aplican a este corte exclusivamente documental; no se ejecutaron pruebas de aplicación ni se inventa evidencia runtime.
- Desviaciones de diseño: ninguna. Reversión: retirar únicamente estos documentos y este progreso, sin tocar código, datos ni US-01.
- Estado nativo consumido: `gentle-ai.sdd-status` v2, cambio US-05, `applyState: ready`, `nextRecommended: apply`, sin bloqueos; `actionContext: repo-local` con raíz y ediciones autorizadas dentro de este repositorio, sin advertencias.
- Límite: detenerse después del corte 1; no ejecutar tarea 2 ni modificar fuentes.
