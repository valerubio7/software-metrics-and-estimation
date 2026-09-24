# Métricas de software y estimación

Este servicio permite crear proyectos (US-01) e historias en el Product Backlog de un proyecto conocido (US-05) mediante una API HTTP respaldada por PostgreSQL.

## Requisitos previos

- Go (la versión declarada en `go.mod`)
- PostgreSQL, con una base de datos disponible para la aplicación
- La CLI de [`golang-migrate`](https://github.com/golang-migrate/migrate) instalada y disponible como `migrate`

La aplicación **no** ejecuta las migraciones automáticamente y este repositorio no instala una herramienta de migración. Aplique `000001` antes de crear proyectos y `000002_create_stories.up.sql` antes de habilitar o publicar la ruta de historias.

## Ejecutar localmente

1. Cree una base de datos de PostgreSQL y establezca su URL de conexión. Adapte las credenciales, el host, el puerto y el nombre de la base de datos a su configuración local de PostgreSQL:

   ```sh
   export DATABASE_URL='postgres://postgres:postgres@localhost:5432/projects?sslmode=disable'
   ```

2. Aplique las migraciones SQL versionadas con la CLI de `golang-migrate`:

   ```sh
   migrate -path internal/project/infrastructure/postgres/migrations -database "$DATABASE_URL" up
   ```

   Esto aplica `000001_create_projects.up.sql` y `000002_create_stories.up.sql`, que crean `projects` y `stories` con su clave foránea. La ejecución de migraciones es externa a la API: con solo `000001` la creación de proyectos sigue disponible, pero la ruta de historias no se registra hasta que la versión 2 esté aplicada sin estado `dirty`. Un error al consultar la versión tampoco habilita historias. No publique la ruta nueva antes de aplicar `000002`.

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

## Crear una historia

Use un `project_id` conocido, por ejemplo el `id` devuelto al crear un proyecto. Envíe exactamente cuatro campos: `title` y `description` no vacíos, `priority` (`alta`, `media` o `baja`) y `acceptance_criteria` (array no vacío de textos no blancos). El ID de proyecto va solo en la URL; la solicitud no admite `id`, `status`, `project_id` ni `story_points` en el cuerpo.

```sh
curl -i -X POST http://localhost:8080/projects/5c21cbd4-d9a7-42df-9c3a-c0866f058746/stories \
  -H 'Content-Type: application/json' \
  -d '{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Se registra el trabajo","Se conserva el criterio"]}'
```

La respuesta `201 Created` tiene esta forma (el `id` se genera en el servidor):

```json
{"id":"<uuid generado>","project_id":"5c21cbd4-d9a7-42df-9c3a-c0866f058746","title":"Registro","description":"Crear historia","priority":"media","status":"pendiente","story_points":null,"acceptance_criteria":["Se registra el trabajo","Se conserva el criterio"]}
```

Una forma JSON inválida devuelve `400`, datos inválidos `422` y un proyecto inexistente `404`; los fallos inesperados devuelven `500` sin detalles internos. La FK impide historias huérfanas. Antes de revertir la migración `000002` evalúe y preserve los datos existentes: su `down` elimina la tabla `stories` y todas las historias almacenadas, no los proyectos.

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
