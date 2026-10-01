# ODD: Reparar compilación de composición API

**Estado:** Reparación implementada y verificada; `go test ./...` pasó.

## Objetivo

Corregir los errores de compilación en la composición de `internal/api` y el arranque de `cmd/api` que impiden ejecutar `go test ./...`, preservando el comportamiento de proyectos, historias y Sprints según las migraciones disponibles.

## Alcance acotado

- Reparar el uso de dependencias en `internal/api/api.go`.
- Consolidar en `cmd/api/main.go` la lectura de estado de migraciones y la composición de rutas según readiness, eliminando referencias sin declarar y declaraciones duplicadas.
- Ajustar pruebas unitarias de composición/readiness y las pruebas afectadas únicamente si la regresión lo requiere.
- Ejecutar checks focalizados y `go test ./...`; registrar resultados sin ocultar omisiones/errores de infraestructura.
- Actualizar el registro ODD de HU-09 para quitar o mantener el bloqueo según evidencia.

## No incluido

- Implementar la ruta HU-09 ni modificar el modelo de cierre de Sprint/US-12.
- Cambiar lógica de dominio o esquema Sprint/story fuera de lo necesario para compilar y componer módulos existentes.
- Commits, push, PR, merge o migraciones en base persistente sin autorización explícita.

## Evidencia inicial

- Rama actual: `feat/us-09-asignar-historias-sprint`, creada desde `main` `3e26553`.
- Error ya observado por el SDD apply: `internal/api/api.go:103-111: undefined: stories`.
- CodeGraph además muestra `cmd/api/main.go` usando `readiness` y `dependencies` sin declarar y redeclarando `handler`.
- `ResolveMigrationReadiness` existente actualmente distingue Projects/Stories/Sprints; preservar su política y probarla.

## Tareas

- [x] Reproducir la falla en rama correcta y registrar error exacto: `go test ./...` exit 1; `internal/api/api.go:103-111` reporta seis referencias `undefined: stories`. El baseline se probó en `feat/us-09-asignar-historias-sprint` (HEAD `3e26553`).
- [x] Corregir la composición de dependencias/readiness con cambios mínimos y cobertura focalizada. Archivos: `internal/api/api.go`, `cmd/api/main.go`, `tests/unit/cmd/api/main_test.go`; la prueba de integración de arranque también se ajustó para respetar FK de `sprint_stories` al preparar esquema v1.
- [x] Correr verificación focalizada y suite `go test ./...`; sin fallos ni omisiones reportadas. `git diff --check` pasó.
- [x] Actualizar `odd/tasks/us-09-asignar-historias-sprint.md` y su espejo Engram con el resultado; HU-09 permanece pausada por US-12 aunque compile.

## Evidencia de cierre

Sin commits ni publicación. La tarea termina cuando los checks informados tengan salida observada; si queda un fallo, crear un paso de remediación y no marcar cierre completo.
