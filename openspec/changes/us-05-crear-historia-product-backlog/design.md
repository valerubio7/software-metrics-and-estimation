# Diseño: crear una historia en el Product Backlog (US-05)

## Decisión y límite

Agregar **solo creación** mediante `POST /projects/{project_id}/stories`. El identificador de proyecto se conoce de antemano (por ejemplo, el devuelto por `POST /projects`); no se añade listado, selector ni consulta del backlog. Una inserción con clave foránea a `projects` verifica la existencia **en el momento de persistir**, sin consulta previa ni ventana de carrera entre `SELECT` e `INSERT`. No se habilita estimación inicial: `story_points` sale y se almacena como `null`.

Fuente de reglas: `proposal.md`, `specs/historia/spec.md` y `exploration.md` de este cambio. Precedentes: `internal/api/api.go`, `internal/project/{domain/project.go,application/create_project.go,transport/http/handler.go,infrastructure/postgres/repository.go}` y la migración de proyectos. Este diseño no modifica el contrato de US-01.

## Recorrido y contrato HTTP

1. `http.ServeMux` registra `POST /projects/{project_id}/stories`; el handler obtiene `request.PathValue("project_id")` y decodifica un solo objeto JSON con `DisallowUnknownFields`, como el handler de proyectos. Otros métodos no crean recursos (comportamiento normal de `ServeMux`).
2. Validar que `project_id` sea un UUID interpretable y normalizarlo para el parámetro UUID de PostgreSQL; validar campos de negocio antes de invocar el repositorio. No consultar proyectos para verificar existencia.
3. El caso de uso genera un UUID para la historia, construye el dominio con estado `pendiente` y puntos ausentes, e invoca **una** operación de inserción. La FK decide la existencia del proyecto. Solo si el `INSERT` termina bien se devuelve `201`.

Solicitud (JSON): `title` (string), `description` (string), `priority` (string: `alta`, `media` o `baja`) y `acceptance_criteria` (array de strings, al menos uno). `project_id` va exclusivamente en la ruta. Ejemplo: `POST /projects/5c21cbd4-d9a7-42df-9c3a-c0866f058746/stories` con `{"title":"Registro","description":"Crear historia","priority":"media","acceptance_criteria":["Se registra el trabajo","Se conserva el criterio"]}`.

Respuesta `201 Created`: `{"id":"<uuid generado>","project_id":"<uuid del proyecto>","title":"Registro","description":"Crear historia","priority":"media","status":"pendiente","story_points":null,"acceptance_criteria":["Se registra el trabajo","Se conserva el criterio"]}`. Serializar `story_points` como `null` **sin `omitempty`**, nunca como `0` ni omitido. Mantener el orden y el contenido de los criterios proporcionados; usar espacios solo para comprobar si un valor está en blanco, no para reescribir texto válido. No aceptar `id`, `status`, `story_points` ni `project_id` en el cuerpo: son campos controlados por servidor/ruta. Esto incluye `story_points:null` explícito: no es lo mismo que omitir el campo y la operación no admite ningún campo de estimación; cualquier clave desconocida se rechaza también.

| Caso | HTTP y cuerpo (forma compatible con errores de proyectos) | Persistencia |
| --- | --- | --- |
| JSON mal formado, varios valores, tipo incompatible o campo extra (incluidos `id`, `status`, `story_points`) | `400`, `{"error":"invalid_request","message":"request body must be a single valid JSON object with allowed fields"}`; texto fijo sin eco del decodificador | Ninguna |
| UUID de ruta mal formado, título/descripción ausentes o en blanco, prioridad ausente/inválida, criterios ausentes/nulos/vacíos o alguna entrada en blanco | `422`, `{"error":"validation_failed","message":"one or more fields are invalid","fields":{"<campo>":"<motivo seguro>"}}`; campos `project_id`, `title`, `description`, `priority`, `acceptance_criteria` según corresponda | Ninguna |
| Proyecto inexistente al intentar insertar | `404`, `{"error":"project_not_found","message":"project not found"}` | Ninguna |
| Error de DB inesperado, fallo de integridad no identificable o asociación no asegurada | `500`, `{"error":"internal_error","message":"an unexpected error occurred"}` | No responder éxito; no queda historia huérfana |

La distinción 404 se reserva al error PostgreSQL `23503` de **la FK nombrada** `stories_project_id_fkey`; otros errores, incluidos otros `23503`, no se convierten en «no existe». No exponer SQL, identificadores internos de restricciones ni textos de `pgx` al cliente. Para fallos de red con resultado de commit incierto, no afirmar que no se persistió la historia: garantizar **no orfandad**, responder 500 y no reintentar automáticamente el `INSERT` (el resultado de creación podría ser desconocido). No se prometen claves de idempotencia ni semántica de reintento en US-05.

## Modelo, puertos y archivos previstos para implementación posterior

| Archivo/área | Cambio acotado |
| --- | --- |
| `internal/story/domain/story.go` | `Story` con ID, ProjectID, Title, Description, Priority, Status, StoryPoints opcionales y `[]string` de criterios; constructor valida textos no blancos, prioridad exacta `alta/media/baja` y lista sin entradas en blanco, asigna `pendiente` y puntos nulos. Error de validación por campo similar a `project/domain.ValidationError`. No inventar longitud máxima, unicidad del título o reglas de autorización. |
| `internal/story/application/create_story.go` | `CreateStoryCommand` con ProjectID y solo cuatro campos de entrada; `StoryRepository` con `Create(ctx, story) error` y generador de ID inyectado (mismo patrón de US-01). Dejar el error tipado `ErrProjectNotFound` en aplicación/infraestructura, no acoplar el caso de uso a `pgconn`. |
| `internal/story/transport/http/handler.go` | Decodificador estricto, UUID de ruta, mapeo de errores, respuesta de creación. El handler no acepta estado/puntos/ID del cliente y no ejecuta SQL. |
| `internal/story/infrastructure/postgres/repository.go` | `INSERT` parametrizado; traducir **solo** la violación de FK nombrada a `ErrProjectNotFound`. Sin lectura previa ni transacción adicional para una sola escritura atómica. |
| `internal/api/api.go`, `cmd/api/main.go` | Construir el caso de uso de historias con repositorio e ID UUID inyectados; registrar la ruta nueva manteniendo `POST /projects`. Adaptar la firma de composición y sus pruebas donde corresponda; reutilizar `uuid.NewString()` sin introducir un servicio global adicional. |
| `internal/project/infrastructure/postgres/migrations/000002_create_stories.{up,down}.sql`, `README.md` | Añadir migración **en el mismo directorio** que ya usa `migrate -path ... up`; documentar endpoint y que la migración sigue siendo externa al inicio del servidor. Down elimina solo `stories` y requiere evaluar datos antes de ejecutarse; no tocar `projects`. |

La verificación de existencia está concentrada en el puerto de escritura: un puerto `ProjectExists` o un `GET /projects` solo duplicaría estado y abriría carrera. El generador de UUID es del servidor; la capa HTTP normaliza UUID de ruta para que un ID inválido no llegue como error SQL 500. La validación de dominio protege también llamadas no HTTP; la capa HTTP reserva 400 para forma JSON y 422 para datos semánticos.

## PostgreSQL e integridad concurrente

Migración `000002` tras `000001`: tabla `stories` con `id UUID PRIMARY KEY`, `project_id UUID NOT NULL CONSTRAINT stories_project_id_fkey REFERENCES projects(id) ON DELETE RESTRICT`, `title TEXT NOT NULL`, `description TEXT NOT NULL`, `priority TEXT NOT NULL`, `status TEXT NOT NULL`, `story_points INTEGER NULL` y `acceptance_criteria TEXT[] NOT NULL`. Aplicar `CHECK (priority IN ('alta','media','baja'))`, `CHECK (cardinality(acceptance_criteria) > 0)` y `CHECK (array_position(acceptance_criteria, NULL) IS NULL)` como protección básica. En US-05 el repositorio inserta explícitamente `status='pendiente'` y `story_points=NULL` (o deja el valor NULL predeterminado), nunca recibe puntos desde el comando. La validación de espacios en criterios y textos reside en dominio; la BD no debe pretender reemplazarla con checks complejos. No añadir un CHECK que prohíba para siempre puntos no nulos: US-14 deberá poder estimar.

Una sola inserción transaccional implícita con FK `ON DELETE RESTRICT` evita huérfanos: si el proyecto se borró antes, la inserción viola la FK y se informa 404; si la inserción gana, PostgreSQL no permite borrar después el proyecto mientras exista la historia; si hay conflicto concurrente, PostgreSQL sincroniza la comprobación referencial y el resultado no permite dejar una fila huérfana. Si un fallo impide confirmar el vínculo, no devolver 201. No introducir endpoint de borrado ni semántica nueva de eliminación de proyectos. El índice de la PK de proyectos ya existe; no crear índices de consulta de backlog anticipados a US-07.

## Verificación futura y despliegue

- `tests/unit/story/domain/`: obligatorios y espacios (incluido Unicode reconocido por `strings.TrimSpace`), prioridad exacta, lista vacía/entrada vacía, conservación y estado/puntos iniciales.
- `tests/unit/story/application/`: ID generado, llamada única en caso válido, cero llamadas ante error, y propagación de proyecto inexistente/error inesperado; dobles de repositorio, sin lectura de proyectos.
- `tests/unit/story/transport/http/`: 201 completo con clave `story_points:null`, criterios ordenados; 400 ante JSON inválido, varios valores, tipos erróneos y campos controlados aun con `null` o `0`; 422 para UUID y campos, sin escritura; 404 para error tipado; 500 genérico sin filtrar secretos. `tests/unit/cmd/api/main_test.go`: nueva ruta y preservación de ruta existente.
- `tests/integration/story/postgres/`: Testcontainers PostgreSQL con migraciones `000001` y `000002`, creación y lectura de datos/fila vinculada, ausencia de proyecto (`23503` traducido, ninguna fila creada), rechazo de eliminación de proyecto referenciado y carrera inserción/borrado controlada o verificación de ambos órdenes sin huérfanos; confirmar puntos SQL `NULL` (no cero) y orden de `TEXT[]`. Las pruebas de integración omitidas por falta de Docker **no** demuestran integridad.
- Implementación futura: TDD estricto en commits cronológicos RED/GREEN/REFACTOR autorizados por el usuario; runner configurado `go test ./...` y pruebas ubicadas en `tests/unit` y `tests/integration`. **En esta fase no se ejecutan pruebas ni se crean pruebas, cambios de fuente o commits.** Aplicar `migrate ... up` antes de publicar el endpoint; rollback de la migración solo tras confirmar que no se pierden historias ya creadas.

## Riesgo de revisión y alcance

El corte completo (nuevo módulo en cuatro capas, migración, composición, README y pruebas unitarias/integración) tiene riesgo **alto** de superar 400 líneas cambiadas. No hay excepción `size:exception` ni encadenamiento autorizados. Antes de implementar, si la estimación o el diff supera el presupuesto, detenerse y consultar al usuario mediante `ask-on-risk`, sin partir arbitrariamente una entrega que deje el endpoint sin integridad o sin pruebas. US-06, US-07, US-09 y US-14 quedan fuera.
