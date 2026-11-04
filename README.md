# Software Metrics & Estimation

API HTTP en Go y una TUI de terminal (Bubble Tea) para gestionar proyectos,
backlog y Sprints, y para navegar el Dashboard, la estimación con Planning
Poker, el esfuerzo y los defectos de cada proyecto.

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

La TUI arranca en el navegador de proyectos («Proyectos»). Si la API devuelve un
único proyecto, la primera carga abre directamente su espacio de trabajo; al
volver con `Esc` la TUI se queda en el navegador. `Enter` sobre un proyecto abre
su espacio de trabajo en la pestaña Dashboard, con una línea con el nombre y el
estado del proyecto y la barra de pestañas:

`1 Dashboard` · `2 Backlog` · `3 Sprints` · `4 Defectos`

| Tecla               | Acción                                                     |
| ------------------- | ---------------------------------------------------------- |
| `Tab` / `Shift+Tab` | Pestaña siguiente / anterior (circular)                    |
| `1`–`4`             | Abrir la pestaña de esa posición                           |
| `Esc`               | En la raíz de una pestaña, volver a Proyectos              |

Las teclas de pestaña rigen en la pantalla raíz de cada pestaña; dentro de un
flujo `Esc` retrocede un paso y, con un filtro en edición, los dígitos se
escriben en el filtro. Todas las pestañas operan sobre el proyecto abierto; para
cambiarlo se vuelve a Proyectos con `Esc` y se elige otro.

Las listas (Proyectos, Backlog, Sprints, historias del Sprint, historial y los
selectores de las pestañas) comparten el mismo patrón navegable:

| Tecla         | Acción                                                        |
| ------------- | ------------------------------------------------------------- |
| `↑` / `↓`     | Mover la selección                                            |
| `Enter`       | Acción primaria de la fila (abrir, editar, desplegar)         |
| `n`           | Crear                                                         |
| `e`           | Editar (Proyectos y Backlog)                                  |
| `Space`       | Marcar historias en la multiselección                         |
| `/`           | Buscar; cualquier otro carácter también inicia la búsqueda    |
| `r`           | Refrescar la lista                                            |
| `d`           | Descargar el reporte PDF (Dashboard y Sprints)                |
| `x`           | Cerrar el Sprint resaltado (pide confirmación `s/n`)          |
| `h`           | Ver los Sprints cerrados                                      |
| `Esc`         | Limpiar el filtro y luego volver                              |
| `?`           | Ayuda                                                         |
| `q` / `Ctrl+C`| Salir (`q` es texto dentro de un campo; `Ctrl+C` sale siempre) |

El pie muestra como máximo cinco atajos, los principales de la pantalla, y
agrega `? más` cuando hay otros. `?` abre la ayuda con todos los atajos válidos
de la pantalla agrupados en «Navegación», «Acciones» y «General». En un filtro o
un campo de texto, `?` es un carácter y el pie no lo ofrece.

Al terminar con éxito una escritura (proyecto, historia, Sprint, asignación,
tarea, cierre, esfuerzo o defecto), la TUI vuelve a la pantalla de origen,
recarga su lista y muestra un aviso que empieza con `✓`; el aviso desaparece con
la siguiente tecla (las acciones del detalle de una historia vuelven al detalle,
ver «Secciones»). Si la API rechaza la operación, el formulario conserva lo
escrito y muestra el motivo. Mientras se guarda, la pantalla indica
«Guardando…» e ignora los reenvíos.

## Secciones

| Pestaña    | Navegación                                                    | Estado |
| ---------- | ------------------------------------------------------------- | ------ |
| Dashboard  | estado y métricas del proyecto o de un Sprint (`s`)           | disponible (TUI-11, TUI-12) |
| Backlog    | lista → detalle de la historia → Estimar, Registrar esfuerzo o Reportar defecto (crear con `n`, editar con `e`) | disponible |
| Sprints    | Sprints abiertos → historias del Sprint → tareas; asignar, cerrar, historial | disponible |
| Defectos   | listado/búsqueda → detalle; alta con historia y Sprint de detección; actualización de estado con Sprint de resolución si queda resuelto | disponible (TUI-09) |

Defectos permite consultar, filtrar y abrir el detalle del listado del proyecto,
registrar un defecto mediante confirmación y cambiar su estado entre las opciones
admitidas por la API. Los flujos que encadenan selectores consultan la API con
`GET` y muestran estados de carga, vacío y error.

En el Backlog, `Enter` abre el detalle de la historia resaltada, `e` la edita,
`n` crea una historia y `/` inicia la búsqueda; las demás letras no reservadas,
incluidas `p`, `P` y `E`, también la inician. El detalle muestra los datos de la
historia y ofrece tres acciones, que se eligen con `↑` / `↓` y `Enter` sobre su
fila o con su atajo (no hacen nada mientras el backlog se recarga):

| Tecla | Acción                                                                 |
| ----- | ---------------------------------------------------------------------- |
| `p`   | Estimar (Planning Poker: integrantes → votación → Story Points)        |
| `t`   | Registrar esfuerzo (tarea o historia completa → integrante → registro) |
| `d`   | Reportar defecto (la historia ya viene elegida; luego el Sprint de detección) |

`Esc` retrocede un nivel por vez: dentro de una acción vuelve al paso anterior y,
desde el primero, al detalle; del detalle vuelve al Backlog conservando el filtro
y el cursor. Al terminar una acción la TUI vuelve al detalle de la misma historia:
tras estimar relee el backlog y el detalle muestra los Story Points nuevos; tras
registrar el esfuerzo o el defecto muestra el aviso `✓`. Los defectos también se
listan, abren y gestionan desde la pestaña Defectos, donde `n` conserva el alta con
selector de historias.

Registrar esfuerzo envía el registro mediante `POST` y, ante el éxito, vuelve al
detalle de la historia con un aviso que resume horas, fecha, historia, tarea e
integrante, sin calcular horas ni desvíos.

Lecturas nuevas usadas por los selectores:

| Método | Ruta                                              | Uso                    |
| ------ | ------------------------------------------------- | ---------------------- |
| `GET`  | `/projects/{project_id}/members`                  | Integrantes            |
| `GET`  | `/projects/{project_id}/stories/{story_id}/tasks` | Tareas de una historia |

### Dashboard

El Dashboard es la pestaña inicial del espacio de trabajo e integra la consulta
de métricas: combina `GET /projects/{project_id}` con
`GET /projects/{project_id}/metrics`. Muestra el estado del proyecto
(`planificado`, `en curso` o `atrasado`) y las métricas de US-19 con barras
Unicode (`█` lleno, `░` vacío) para Story Points, horas, historias completadas y
defectos. Una métrica `null` se presenta como «sin datos», nunca como cero. La
pantalla es de solo lectura, se mantiene legible en terminales de 80 y 44
columnas y no depende del color: cada fila conserva su etiqueta y sus valores.

`s` abre el selector de ámbito con «Proyecto completo» y los Sprints abiertos;
elegir un Sprint consulta `GET /projects/{project_id}/sprints/{sprint_id}/metrics`
y muestra el bloque «Métricas del Sprint» con las mismas barras. `Esc` en el
selector vuelve sin cambiar el ámbito. `r` recarga el ámbito mostrado, `d`
descarga su reporte PDF y `Esc` vuelve a Proyectos.

### Reportes en PDF desde la TUI

La TUI usa las mismas rutas de reporte de solo lectura que la API y guarda el
archivo en el directorio desde donde se ejecuta:

- `d` en Dashboard guarda el reporte del ámbito mostrado:
  `reporte-proyecto-{project_id}.pdf` o `reporte-sprint-{sprint_id}.pdf`.
- `d` en Sprints guarda `reporte-sprint-{sprint_id}.pdf` del Sprint resaltado.

La descarga solo emite `GET`, valida que la respuesta empiece con la firma
`%PDF-` y muestra en pantalla la ruta guardada o el error de la API. Si el archivo
ya existe en el directorio actual, se reemplaza.

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

## API de reportes

Genera un PDF de solo lectura para un proyecto o un Sprint:

```http
GET  /projects/{project_id}/report.pdf
GET  /projects/{project_id}/sprints/{sprint_id}/report.pdf
HEAD /projects/{project_id}/report.pdf
HEAD /projects/{project_id}/sprints/{sprint_id}/report.pdf
```

`GET` responde `200` con `Content-Type: application/pdf`, `Content-Disposition: attachment`
y `Content-Length`. El nombre descargado es `reporte-proyecto-{project_id}.pdf` o
`reporte-sprint-{sprint_id}.pdf`. `HEAD` ejecuta la misma consulta y devuelve los mismos
encabezados sin cuerpo.

El PDF identifica el proyecto y, en ámbito Sprint, el Sprint con su objetivo y estado. Incluye
las métricas del ámbito, las historias planificadas y completadas, las tareas con sus
estimaciones, el esfuerzo registrado y los defectos relacionados. Los valores faltantes se
muestran como `No disponible` y la cobertura de las métricas acompaña cada agregado.

- Proyecto incluye todas las historias, tareas, registros de esfuerzo y defectos del proyecto;
  la velocidad promedia los Sprints cerrados. Sprint incluye las historias asignadas, las
  tareas del Sprint, el esfuerzo atribuible a sus tareas y los defectos detectados o resueltos
  en él.
- Las métricas se derivan con las mismas fórmulas de US-19 desde una única transacción
  `RepeatableRead`/`ReadOnly`, por lo que coinciden con `GET .../metrics` para el mismo ámbito.
  La tabla de esfuerzo de un Sprint excluye registros directos a historia y esfuerzo de tareas
  de otros Sprints; esos registros siguen contabilizados en la cobertura como no atribuibles.
- La generación no crea ni modifica datos. `422 validation_failed` indica UUID inválidos sin
  consultar el almacenamiento; `404 project_not_found` o `404 sprint_not_found` indican que el
  ámbito no existe o no pertenece al proyecto; `405` corresponde a métodos distintos de `GET`
  o `HEAD`; `500 internal_error` no expone detalles internos.
- Las rutas solo se registran con el esquema limpio en versión 14 o superior. Con versión
  menor, estado `dirty` o error al leer `schema_migrations`, responden `404`.

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
