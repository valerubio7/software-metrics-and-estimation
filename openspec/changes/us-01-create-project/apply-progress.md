# Progreso de aplicación: US-01 — Corte 2 (PostgreSQL)

## Resumen

Se añadió el corte de infraestructura para persistir proyectos en PostgreSQL: migraciones SQL versionadas, repositorio basado en `pgx` y andamiaje de integración con `testcontainers-go`. La composición de runtime en `cmd/api` no se modificó: todavía no existe una decisión de configuración de conexión ni de generación de UUID que permita conectarla sin inventar complejidad fuera de este corte.

## Implementado

- Migración ascendente `000001_create_projects.up.sql` para `projects` con `id UUID PRIMARY KEY`, `name TEXT NOT NULL`, fechas `DATE NOT NULL` y la restricción `CHECK (planned_finish_date >= start_date)`.
- Migración descendente que elimina `projects`.
- `PostgresProjectRepository`, que satisface `application.ProjectRepository` y usa un `INSERT` parametrizado mediante `pgxpool.Pool`.
- Pruebas de integración que levantan PostgreSQL real con `testcontainers-go`, aplican la migración y verifican:
  - persistencia de `id`, `name` y ambas fechas de calendario;
  - rechazo de la restricción de base de datos para una fecha final anterior a la inicial (`23514`).
- Dependencias directas: `pgx/v5` y el módulo PostgreSQL de `testcontainers-go`; sus dependencias transitivas quedan registradas en `go.sum`.
- El dominio y la aplicación continúan sin importar paquetes de PostgreSQL o del driver.

## Archivos modificados

- `go.mod`
- `go.sum`
- `internal/project/infrastructure/postgres/repository.go`
- `internal/project/infrastructure/postgres/repository_integration_test.go`
- `internal/project/infrastructure/postgres/migrations/000001_create_projects.up.sql`
- `internal/project/infrastructure/postgres/migrations/000001_create_projects.down.sql`
- `openspec/changes/us-01-create-project/tasks.md`
- `openspec/changes/us-01-create-project/apply-progress.md`

## Evidencia TDD

- **RED:** al añadir las pruebas de integración, `go test ./...` falló porque `pgx` todavía no era una dependencia declarada (`no required module provides package github.com/jackc/pgx/v5/pgconn`).
- **GREEN:** tras declarar las dependencias, implementar la migración y el repositorio, `go test ./...` pasó.
- **TRIANGULATE:** `go test -count=1 -v ./internal/project/infrastructure/postgres` ejecutó ambas pruebas contra contenedores PostgreSQL reales y pasó; verificó la recuperación de fechas de calendario y la violación `23514`.
- **REFACTOR:** los detalles SQL, `pgx` y las migraciones se mantienen en `internal/project/infrastructure/postgres`; no se modificaron los paquetes de dominio ni aplicación.

## Pruebas ejecutadas

```text
go test -count=1 -v ./internal/project/infrastructure/postgres
```

Resultado: exitoso. Ambas pruebas se ejecutaron contra contenedores PostgreSQL reales; se verificaron la persistencia y recuperación de fechas de calendario y el rechazo de la restricción `CHECK` (`23514`).

```text
go test ./...
```

Resultado: exitoso.

```text
git diff --check
```

Resultado: exitoso, sin errores de espacios en el diff.

## Resolución de integración PostgreSQL

Docker ya es accesible para el usuario actual y Testcontainers pudo crear contenedores PostgreSQL reales. En la primera ejecución con Docker disponible, las pruebas fallaron con un reinicio de conexión inmediatamente después de iniciar el contenedor. Tras añadir una espera explícita mediante `ping` de PostgreSQL antes de aplicar las migraciones, las pruebas de integración pasaron.

## Pendiente

- Definir la configuración de conexión PostgreSQL y el generador de UUID antes de componer el repositorio en `cmd/api`.
- Completar la verificación final de alcance de US-01.
