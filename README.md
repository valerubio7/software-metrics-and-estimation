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

Se permiten defectos duplicados (mismos valores en los cinco campos). La ruta solo existe
cuando el esquema está limpio en la versión 13 o superior (migración `000013`); con una
versión menor, o en estado `dirty`, responde `404`. Esta capacidad solo registra: no lista,
edita ni elimina defectos, y la resolución con Sprint de resolución es US-17.

## API de métricas

Consulta métricas de solo lectura por proyecto o por Sprint:

```http
GET  /projects/{project_id}/metrics
GET  /projects/{project_id}/sprints/{sprint_id}/metrics
HEAD /projects/{project_id}/metrics
HEAD /projects/{project_id}/sprints/{sprint_id}/metrics
```

`GET` responde `200` con las métricas en el nivel raíz (sin contenedor). `HEAD` ejecuta la
misma consulta y devuelve el mismo estado y encabezados, sin cuerpo. La respuesta conserva
`project_id`, `sprint_id`, `story_points_planned`, `story_points_completed`, `velocity`,
`estimated_hours`, `actual_hours`, `deviation`, `completed_stories_percentage`,
`defects_detected` y `defects_resolved`; agrega `deviation_percentage` y `coverage`.

Ejemplo de proyecto:

```json
{
  "project_id": "11111111-1111-4111-8111-111111111111",
  "sprint_id": null,
  "story_points_planned": 8,
  "story_points_completed": 5,
  "velocity": 4,
  "estimated_hours": 9,
  "actual_hours": 5,
  "deviation": -4,
  "deviation_percentage": -44.44444444444444,
  "completed_stories_percentage": 50,
  "defects_detected": 2,
  "defects_resolved": 1,
  "coverage": {
    "story_points_planned": {"missing_observations": 0},
    "story_points_completed": {"missing_observations": 0},
    "velocity": {"missing_observations": 0},
    "estimated_hours": {"missing_observations": 0},
    "actual_hours": {
      "stories_without_records": 1,
      "unattributed_story_level_records": 0
    }
  }
}
```

Ejemplo de Sprint:

```json
{
  "project_id": "11111111-1111-4111-8111-111111111111",
  "sprint_id": "22222222-2222-4222-8222-222222222222",
  "story_points_planned": 5,
  "story_points_completed": 5,
  "velocity": 5,
  "estimated_hours": 2,
  "actual_hours": 3,
  "deviation": 1,
  "deviation_percentage": 50,
  "completed_stories_percentage": 100,
  "defects_detected": 1,
  "defects_resolved": 0,
  "coverage": {
    "story_points_planned": {"missing_observations": 0},
    "story_points_completed": {"missing_observations": 0},
    "velocity": {"missing_observations": 0},
    "estimated_hours": {"missing_observations": 0},
    "actual_hours": {
      "stories_without_records": 0,
      "unattributed_story_level_records": 1
    }
  }
}
```

### Ámbito y fórmulas

- Proyecto cuenta cada historia del proyecto una sola vez, incluidas las que no pertenecen a
  un Sprint. Sprint cuenta únicamente las historias asignadas a ese Sprint.
- La finalización usa `sprint_stories.completed_at`, nunca `stories.status`. Una historia del
  proyecto cuenta completada una sola vez si completó en al menos un Sprint.
- Los Story Points planificados suman todas las historias del ámbito; los completados solo
  las completadas. La velocidad de Sprint es su suma de puntos completados; la velocidad de
  proyecto promedia los puntos completados de todos los Sprints cerrados, incluidos los de
  cero puntos.
- Para horas estimadas, las tareas reemplazan la estimación de su historia: proyecto usa
  todas sus tareas y Sprint solo las del Sprint consultado. Si no hay tareas seleccionadas,
  se usa la estimación de la historia. Una estimación relevante faltante hace que el agregado
  sea `null` y aumenta `coverage.estimated_hours.missing_observations`.
- Proyecto suma una vez cada registro de esfuerzo. Sprint solo atribuye registros ligados a
  tareas de ese Sprint; los registros directos a historia se excluyen e incrementan
  `coverage.actual_hours.unattributed_story_level_records`. Los registros parciales se suman
  y las historias sin esfuerzo atribuible se cuentan en `stories_without_records`.
- `deviation` es horas reales menos estimadas. `deviation_percentage` divide esa diferencia
  por las horas estimadas y multiplica por 100; es `null` si falta un operando o la estimación
  es cero. Los defectos detectados se filtran por `sprint_id`; los resueltos por
  `resolution_sprint_id`, sin inferir resolución desde `status`.
- Un agregado de puntos o velocidad de Sprint sin observaciones vale `0`. Porcentajes sin
  denominador, horas sin observaciones y velocidad de proyecto sin Sprints cerrados valen
  `null`. Los puntos/estimaciones relevantes faltantes producen `null` con su contador de
  cobertura; la cobertura no reemplaza el valor métrico.

### Errores y disponibilidad

- `422 validation_failed`: `fields` indica `project_id`, `sprint_id` o ambos inválidos; no se
  consulta el almacenamiento.
- `404 project_not_found` o `404 sprint_not_found`: el proyecto no existe o el Sprint no
  existe/no pertenece al proyecto.
- `405`: método distinto de `GET` o `HEAD`. `500 internal_error`: fallo inesperado, sin
  detalles internos ni métricas parciales.
- Las rutas solo se registran con el esquema PostgreSQL limpio en versión 14 o superior
  (migración `000014`, que incorpora el Sprint de resolución del defecto). Con versión menor,
  estado `dirty` o error al leer `schema_migrations`, las rutas no están disponibles y
  responden `404`. Las migraciones se aplican externamente; la API no las ejecuta.

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
