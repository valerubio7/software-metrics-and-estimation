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
