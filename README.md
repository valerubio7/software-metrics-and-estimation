# Software Metrics & Estimation

API HTTP en Go y una TUI de terminal (Bubble Tea) para gestionar proyectos,
backlog y Sprints.

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

Las listas (Proyectos, Backlog, Sprints, historias del Sprint e historial)
comparten el mismo patrón navegable:

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
