# Métricas de software y estimación

Este servicio implementa actualmente US-01: crear un proyecto mediante una API HTTP respaldada por PostgreSQL.

## Requisitos previos

- Go (la versión declarada en `go.mod`)
- PostgreSQL, con una base de datos disponible para la aplicación
- La CLI de [`golang-migrate`](https://github.com/golang-migrate/migrate) instalada y disponible como `migrate`

La aplicación **no** ejecuta las migraciones automáticamente y este repositorio no instala una herramienta de migración. Aplique la migración SQL versionada antes de iniciar la API.

## Ejecutar localmente

1. Cree una base de datos de PostgreSQL y establezca su URL de conexión. Adapte las credenciales, el host, el puerto y el nombre de la base de datos a su configuración local de PostgreSQL:

   ```sh
   export DATABASE_URL='postgres://postgres:postgres@localhost:5432/projects?sslmode=disable'
   ```

2. Aplique las migraciones SQL versionadas con la CLI de `golang-migrate`:

   ```sh
   migrate -path internal/project/infrastructure/postgres/migrations -database "$DATABASE_URL" up
   ```

   Esto aplica `000001_create_projects.up.sql`, que crea la tabla `projects`. La ejecución de migraciones es un requisito explícito del despliegue o del entorno local, no comportamiento de la API en tiempo de ejecución.

3. De forma opcional, elija la dirección de escucha HTTP. Su valor predeterminado es `:8080` cuando `HTTP_ADDR` no está configurada:

   ```sh
   export HTTP_ADDR=':8080'
   ```

4. Inicie la API:

   ```sh
   go run ./cmd/api
   ```

   `DATABASE_URL` es obligatoria. Durante el inicio, la API se conecta a PostgreSQL y comprueba su disponibilidad antes de escuchar solicitudes.

## Crear un proyecto

Con la API en ejecución, cree un proyecto con:

```sh
curl -i -X POST http://localhost:8080/projects \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Metrics portal",
    "start_date": "2026-03-01",
    "planned_finish_date": "2026-06-30"
  }'
```

Una solicitud válida devuelve `201 Created` con `id`, `name`, `start_date` y `planned_finish_date`. Las fechas deben usar `YYYY-MM-DD` y `planned_finish_date` debe ser igual o posterior a `start_date`.

## Pruebas

Todas las pruebas se encuentran en `tests/unit/` y `tests/integration/`. Ejecútelas desde la raíz del repositorio:

```sh
go test -count=1 ./...
```

Para ejecutar solo las pruebas unitarias:

```sh
go test -count=1 ./tests/unit/...
```

Para ejecutar las pruebas de integración de PostgreSQL con salida detallada:

```sh
go test -count=1 -v ./tests/integration/...
```

Las pruebas de integración usan Testcontainers y requieren que Docker esté disponible.
