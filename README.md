# Métricas de software y estimación

Este servicio permite crear proyectos (US-01), crear historias en el Product Backlog de un proyecto conocido (US-05) y modificarlas (US-06) mediante una API HTTP respaldada por PostgreSQL.

## Requisitos previos

- Go (la versión declarada en `go.mod`)
- PostgreSQL, con una base de datos disponible para la aplicación
- La CLI de [`golang-migrate`](https://github.com/golang-migrate/migrate) instalada y disponible como `migrate`

La aplicación **no** ejecuta las migraciones automáticamente y este repositorio no instala una herramienta de migración. Aplique `000001` antes de crear proyectos, `000002_create_stories.up.sql` antes de habilitar o publicar la ruta de creación de historias y `000003_add_story_estimated_hours.up.sql` antes de habilitar o publicar la ruta de modificación.

## Ejecutar localmente

1. Cree una base de datos de PostgreSQL y establezca su URL de conexión. Adapte las credenciales, el host, el puerto y el nombre de la base de datos a su configuración local de PostgreSQL:

   ```sh
   export DATABASE_URL='postgres://postgres:postgres@localhost:5432/projects?sslmode=disable'
   ```

2. Aplique las migraciones SQL versionadas con la CLI de `golang-migrate`:

   ```sh
   migrate -path internal/project/infrastructure/postgres/migrations -database "$DATABASE_URL" up
   ```

   Esto aplica `000001_create_projects.up.sql`, `000002_create_stories.up.sql` y `000003_add_story_estimated_hours.up.sql`, que crean `projects` y `stories` con su clave foránea y agregan la estimación de horas y la restricción del estado. La ejecución de migraciones es externa a la API: con solo `000001` la creación de proyectos sigue disponible, pero la ruta de creación de historias no se registra hasta que la versión 2 esté aplicada sin estado `dirty`, y la ruta de modificación no se registra hasta que la versión 3 esté aplicada sin estado `dirty`. Con el esquema en versión 2 la creación de historias sigue funcionando y `PUT /projects/{project_id}/stories/{story_id}` responde `404` porque la ruta no existe. Un error al consultar la versión tampoco habilita historias. No publique las rutas nuevas antes de aplicar `000002` (creación) y `000003` (modificación); el log de arranque indica cuál de los tres casos quedó activo.

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
{"id":"<uuid generado>","project_id":"5c21cbd4-d9a7-42df-9c3a-c0866f058746","title":"Registro","description":"Crear historia","priority":"media","status":"pendiente","story_points":null,"acceptance_criteria":["Se registra el trabajo","Se conserva el criterio"],"estimated_hours":null}
```

Una forma JSON inválida devuelve `400`, datos inválidos `422` y un proyecto inexistente `404`; los fallos inesperados devuelven `500` sin detalles internos. La FK impide historias huérfanas. Antes de revertir la migración `000002` evalúe y preserve los datos existentes: su `down` elimina la tabla `stories` y todas las historias almacenadas, no los proyectos.

## Modificar una historia

`PUT /projects/{project_id}/stories/{story_id}` reemplaza por completo el contenido editable de una historia existente del proyecto indicado. Solo está disponible con el esquema en versión 3 o superior sin estado `dirty` (migración `000003`). Ambos identificadores van únicamente en la URL y deben ser UUID.

Envíe exactamente seis claves, todas obligatorias: `title` y `description` no vacíos, `priority` (`alta`, `media` o `baja`), `status`, `acceptance_criteria` (array no vacío de textos no blancos) y `estimated_hours`. Una clave ausente se rechaza con `422`: no significa "conservar el valor actual". `estimated_hours` debe estar presente, pero puede ser `null`. El cuerpo no admite `id`, `project_id` ni `story_points`: la identidad y los Story Points no cambian con esta operación.

```sh
curl -i -X PUT http://localhost:8080/projects/5c21cbd4-d9a7-42df-9c3a-c0866f058746/stories/e99c05a4-03ea-4c19-8f59-286dd59e7aa1   -H 'Content-Type: application/json'   -d '{"title":"Login con email","description":"Iniciar sesión con email y contraseña","priority":"alta","status":"en_progreso","acceptance_criteria":["Valida el email","Rechaza contraseña incorrecta"],"estimated_hours":8.5}'
```

La respuesta `200 OK` devuelve la historia completa tal como quedó almacenada, con los nueve campos (`id`, `project_id`, `title`, `description`, `priority`, `status`, `story_points`, `acceptance_criteria` y `estimated_hours`) y el orden de los criterios preservado.

- **Estado (`status`)**: uno de `pendiente`, `en_progreso` o `completada`. Cualquier otro valor se rechaza con `422`. No hay reglas de transición: se puede pasar de cualquier estado a cualquier otro. Una historia nueva nace `pendiente`.
- **Estimación (`estimated_hours`)**: número JSON mayor que `0`, con como máximo dos decimales y menor o igual que `99999.99`. `null` borra la estimación almacenada. `0`, negativos, más de dos decimales o valores mayores que `99999.99` devuelven `422`; un tipo incorrecto (por ejemplo, un texto) devuelve `400`. Es independiente de los Story Points.
- **Errores**: `400 invalid_request` (JSON malformado, cuerpo vacío o que no es un objeto, más de un valor, claves desconocidas o de identidad, tipos incorrectos), `404 story_not_found` (la historia no existe o pertenece a otro proyecto; ambos casos son indistinguibles), `405` (método no admitido sobre la ruta), `422 validation_failed` con el detalle en `fields` (identificadores que no son UUID, claves ausentes, contenido, estado o estimación inválidos) y `500 internal_error` sin detalles internos.
- **Concurrencia**: la última escritura gana. No hay versionado ni control de concurrencia optimista: dos modificaciones simultáneas se aplican en orden de llegada y la segunda reemplaza a la primera sin aviso.
- **Reversión de la migración `000003`**: su `down` elimina la columna `estimated_hours` (se pierden todas las estimaciones) y la restricción del estado. Exporte las estimaciones y devuelva deliberadamente a `pendiente` las historias `en_progreso` o `completada` antes de ejecutarlo.

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
