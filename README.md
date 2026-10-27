# Software Metrics & Estimation

API HTTP en Go y una TUI de terminal (Bubble Tea) para gestionar proyectos,
backlog y Sprints, y para navegar las secciones de Planning Poker, Esfuerzo,
Defectos, Métricas y Dashboard.

## Requisitos

- Go 1.27+
- PostgreSQL con el esquema migrado (`schema_migrations` en la versión esperada
  por la API)

## Configuración

| Variable       | Default                 | Uso                              |
| -------------- | ----------------------- | -------------------------------- |
| `DATABASE_URL` | — (obligatoria)         | Conexión a PostgreSQL de la API  |
| `HTTP_ADDR`    | `:8080`                 | Dirección de escucha de la API   |
| `API_URL`      | `http://localhost:8080` | URL que consume la TUI           |

## Ejecutar

API:

```sh
DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
HTTP_ADDR=':8080' \
go run ./cmd/api
```

TUI, en otra terminal:

```sh
API_URL='http://127.0.0.1:8080' go run ./cmd/tui
```

## Teclas

Las listas (Proyectos, Backlog, Sprints, historias del Sprint, historial y los
selectores de las secciones del Sprint 3) comparten el mismo patrón navegable:

| Tecla         | Acción                                                        |
| ------------- | ------------------------------------------------------------- |
| `↑` / `↓`     | Mover la selección                                            |
| `Enter`       | Acción primaria de la fila (usar, editar, abrir, desplegar)    |
| `n`           | Crear                                                         |
| `e`           | Editar (Proyectos)                                            |
| `Space`       | Marcar historias en la multiselección                         |
| `/`           | Buscar; cualquier otro carácter también inicia la búsqueda    |
| `r`           | Refrescar la lista                                            |
| `x`           | Cerrar el Sprint resaltado (pide confirmación `s/n`)          |
| `h`           | Ver los Sprints cerrados                                      |
| `p`           | Cambiar el proyecto activo                                    |
| `Esc`         | Limpiar el filtro y luego volver                              |
| `?`           | Ayuda                                                         |
| `q` / `Ctrl+C`| Salir                                                         |

## Secciones

El menú incluye nueve opciones: Proyectos, Backlog, Sprints, Planning Poker,
Esfuerzo, Defectos, Métricas, Dashboard y Salir. Las cinco secciones nuevas
abren directamente su pantalla principal y encadenan selectores según el flujo:

| Sección        | Navegación                                      | Flujo futuro |
| -------------- | ----------------------------------------------- | ------------ |
| Planning Poker | historia → integrante                           | TUI-07       |
| Esfuerzo       | historia → tarea                                | TUI-08       |
| Defectos       | historia → Sprint                               | TUI-09       |
| Métricas       | Sprint                                          | TUI-10       |
| Dashboard      | Sprint                                          | TUI-11       |

Estas pantallas son *shells* de navegación de solo lectura: consultan la API con
`GET`, muestran estados de carga, vacío y error, y terminan con un aviso de la
historia que implementará el flujo de negocio. No crean, modifican ni eliminan
datos. Si no hay proyecto activo, la sección abre primero el navegador de
proyectos; el atajo `p` permite cambiarlo sin salir de la sección.

Lecturas nuevas usadas por los selectores:

| Método | Ruta                                              | Uso                    |
| ------ | ------------------------------------------------- | ---------------------- |
| `GET`  | `/projects/{project_id}/members`                  | Integrantes            |
| `GET`  | `/projects/{project_id}/stories/{story_id}/tasks` | Tareas de una historia |

## API de esfuerzo

Registra el esfuerzo realizado por un integrante en una historia o en una de sus
tareas:

| Método | Ruta                                                    | Uso                                                |
| ------ | ------------------------------------------------------- | -------------------------------------------------- |
| `POST` | `/projects/{project_id}/stories/{story_id}/effort-records` | Registrar esfuerzo realizado en una historia o tarea |

Cuerpo (JSON, solo estas claves):

```json
{
  "member_id": "<uuid del integrante>",
  "task_id": "<uuid de la tarea, opcional>",
  "work_date": "2026-03-10",
  "description": "Revisión de código",
  "hours": 2.5
}
```

- `201`: devuelve `id`, `project_id`, `story_id`, `task_id` (`null` si el esfuerzo es de la
  historia), `member_id`, `work_date`, `description` (sin espacios sobrantes) y `hours`.
- `400 invalid_request`: el cuerpo no es un objeto JSON válido o trae claves desconocidas.
- `422 validation_failed`: `fields` indica cada campo inválido (`project_id`, `story_id`,
  `member_id`, `task_id`, `work_date`, `description`, `hours`). Las horas deben ser mayores
  que 0, hasta 24 y con a lo sumo dos decimales; la fecha (`YYYY-MM-DD`) no puede ser
  posterior al día actual en UTC; la descripción admite hasta 500 caracteres.
- `404`: `project_not_found`, `story_not_found`, `member_not_found` o `task_not_found`.
  La tarea debe pertenecer a la historia y el integrante al proyecto.
- `500 internal_error`: fallo de almacenamiento, sin detalles internos; no se persiste nada.

La ruta solo existe cuando el esquema está limpio en la versión 12 o superior (migración
`000012`); con una versión menor, o en estado `dirty`, responde `404`. Esta capacidad solo
registra: no lista, edita ni elimina registros de esfuerzo.

## API de defectos

Registra un defecto detectado durante el desarrollo dentro de un proyecto, indicando la
historia relacionada y el Sprint de detección:

| Método | Ruta                                    | Uso                                         |
| ------ | --------------------------------------- | ------------------------------------------- |
| `POST` | `/projects/{project_id}/defects`        | Registrar un defecto de un proyecto          |
| `GET`  | `/projects/{project_id}/defects`        | Consultar los defectos del proyecto          |

La consulta responde `200` con el proyecto y una lista `defects`. Cada elemento incluye
`id`, `description`, `severity`, `status`, `story_id`, `sprint_id` (Sprint de detección) y
`resolution_sprint_id` (Sprint de resolución o `null` si sigue sin resolverse). Si el proyecto
no tiene defectos, `defects` es un arreglo vacío:

```json
{
  "project_id": "<uuid>",
  "defects": [
    {
      "id": "<uuid>",
      "description": "El login acepta contraseñas vacías",
      "severity": "alta",
      "status": "detectado",
      "story_id": "<uuid>",
      "sprint_id": "<uuid>",
      "resolution_sprint_id": null
    }
  ]
}
```

Un `project_id` con formato inválido responde `422 validation_failed`; un proyecto inexistente
responde `404 project_not_found`. La lectura no modifica defectos.

Cuerpo (JSON, solo estas claves):

```json
{
  "story_id": "<uuid de la historia relacionada>",
  "sprint_id": "<uuid del Sprint de detección>",
  "description": "El login acepta contraseñas vacías",
  "severity": "alta",
  "status": "detectado"
}
```

- `201`: devuelve `id`, `project_id`, `story_id`, `sprint_id`, `description` (sin espacios
  sobrantes), `severity` y `status`.
- `400 invalid_request`: el cuerpo no es un objeto JSON válido o trae claves desconocidas.
- `422 validation_failed`: `fields` indica cada campo inválido (`project_id`, `story_id`,
  `sprint_id`, `description`, `severity`, `status`). La descripción admite hasta 500
  caracteres; `severity` es `alta`, `media` o `baja`; el alta acepta `detectado` o
  `en_correccion` (actualizar a `resuelto` es US-17, no este alta).
- `404`: `project_not_found`, `story_not_found` o `sprint_not_found`; la historia y el
  Sprint de detección deben pertenecer al proyecto. Una historia del backlog sin Sprint
  asignado es válida, igual que un defecto detectado en un Sprint cerrado.
- `500 internal_error`: fallo de almacenamiento, sin detalles internos; no se persiste nada.

Se permiten defectos duplicados (mismos valores en los cinco campos). El registro solo está
disponible cuando el esquema está limpio en la versión 13 o superior (migración `000013`); la
consulta requiere la versión 14 o superior (migración `000014`) para incluir el Sprint de
resolución. En una versión menor, o en estado `dirty`, la ruta correspondiente responde `404`.
El registro y la consulta no eliminan defectos; la actualización de estado y resolución es US-17.

## Pruebas

```sh
gofmt -l internal tests
go build ./...
go vet ./...
go test ./tests/unit/...
```

Las pruebas de integración requieren PostgreSQL y se ejecutan aparte:

```sh
go test ./tests/integration/...
```
