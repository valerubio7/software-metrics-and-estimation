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
- **TRIANGULATE:** las pruebas contienen las verificaciones de recuperación de fechas de calendario y de la violación `23514`, pero su ejecución real contra PostgreSQL quedó bloqueada por Docker.
- **REFACTOR:** los detalles SQL, `pgx` y las migraciones se mantienen en `internal/project/infrastructure/postgres`; no se modificaron los paquetes de dominio ni aplicación.

## Pruebas ejecutadas

```text
go test ./...
```

Resultado: exitoso. El paquete `internal/project/infrastructure/postgres` compila y pasa, con las pruebas de integración omitidas cuando Docker no está disponible.

```text
go test -v ./internal/project/infrastructure/postgres
```

Resultado: exitoso con ambas pruebas omitidas explícitamente. La salida observada fue `permission denied while trying to connect to the docker API at unix:///var/run/docker.sock`.

## Bloqueo de integración PostgreSQL

Docker no es accesible para el usuario actual: Testcontainers no puede ejecutar `docker info` contra `/var/run/docker.sock` por permiso denegado. Por ello no se declara completa la ejecución de integración contra PostgreSQL real. Las pruebas no simulan PostgreSQL: se omiten con un mensaje explícito hasta que Docker sea accesible (por ejemplo, mediante pertenencia al grupo `docker` o un daemon autorizado).

## Pendiente

- Ejecutar las pruebas de integración contra PostgreSQL real cuando Docker esté disponible y registrar los resultados de inserción, fechas y restricción `CHECK`.
- Definir la configuración de conexión PostgreSQL y el generador de UUID antes de componer el repositorio en `cmd/api`.
- Completar la verificación final de alcance de US-01 una vez resuelto el bloqueo de integración.
