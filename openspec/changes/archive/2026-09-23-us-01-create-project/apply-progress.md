# Progreso de aplicación: US-01 — Composición de runtime y documentación

## Resumen

US-01 queda compuesta para ejecutarse contra PostgreSQL y documentada para uso local. `cmd/api` construye el repositorio PostgreSQL, el caso de uso y el handler de `POST /projects`; requiere `DATABASE_URL`, acepta `HTTP_ADDR` opcional y genera UUID. La migración sigue siendo un prerrequisito explícito: la API no la ejecuta automáticamente y el repositorio no instala una herramienta de migración.

## Implementado

- Composición de runtime en `cmd/api`:
  - `DATABASE_URL` obligatorio para la conexión PostgreSQL;
  - `HTTP_ADDR` opcional, con valor predeterminado `:8080`;
  - creación y verificación de `pgxpool.Pool`;
  - construcción de `PostgresProjectRepository`, `CreateProjectUseCase` y el handler de `POST /projects`;
  - generación de identificadores con `github.com/google/uuid`.
- Documentación de ejecución local en `README.md`, incluida la preparación de PostgreSQL, las variables de entorno, la aplicación manual de migraciones y un ejemplo `curl` de `POST /projects`.
- Migraciones SQL versionadas en `internal/project/infrastructure/postgres/migrations/`. Deben aplicarse antes de iniciar la API con una CLI `golang-migrate` instalada externamente:

  ```sh
  migrate -path internal/project/infrastructure/postgres/migrations -database "$DATABASE_URL" up
  ```

  No existe integración automática de migraciones en el proceso de la API ni se afirma que la CLI esté instalada.

## Evidencia TDD de la composición de runtime

- **RED — commit `e26b574` (`test(api): define runtime composition expectations`):** se añadieron pruebas para `DATABASE_URL` obligatorio, dirección HTTP predeterminada y configurable, generación de UUID y registro de `POST /projects`. El commit registra que `go test ./cmd/api` fallaba porque `loadConfig`, `newProjectID` y `newHTTPHandler` aún no existían.
- **GREEN — commit `a505396` (`feat(api): wire project creation to PostgreSQL`):** se implementó la composición de runtime. El commit registra que `go test -count=1 ./cmd/api` y `go test -count=1 ./...` pasaron, incluidas las pruebas con contenedores PostgreSQL reales.
- **TRIANGULATE:** las pruebas de integración de PostgreSQL verifican persistencia y recuperación de fechas de calendario y el rechazo de la restricción `CHECK` para una fecha final anterior a la inicial.
- **REFACTOR:** la composición permanece en `cmd/api`; el SQL, el driver y las migraciones permanecen en infraestructura, sin introducir PostgreSQL en dominio o aplicación.

## Verificación final de alcance

- La ruta de US-01 es únicamente `POST /projects`; no se agregaron operaciones de actualización, miembros, consulta de estado ni un campo `status`.
- La respuesta exitosa serializa solo `id`, `name`, `start_date` y `planned_finish_date`.
- Los fallos inesperados de infraestructura se convierten en `500 Internal Server Error` con el mensaje genérico `an unexpected error occurred`; no se exponen detalles internos de PostgreSQL.

## Pruebas ejecutadas

```text
go test -count=1 ./...
```

Resultado: exitoso. Pasaron los paquetes `cmd/api`, aplicación, dominio, transporte HTTP e infraestructura PostgreSQL; las pruebas de integración usaron Testcontainers con PostgreSQL real.

```text
git diff --check
```

Resultado: exitoso, sin errores de espacios en el diff después de la actualización documental.

## Estado

Las tareas de US-01 están completadas. El uso de una CLI de migraciones es documentación de un prerrequisito operativo, no una integración de migración automática en la aplicación.
