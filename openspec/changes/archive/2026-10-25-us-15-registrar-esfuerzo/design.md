# Diseño: Registrar el esfuerzo realizado en una historia o tarea (US-15)

## Enfoque técnico

Corte vertical nuevo en `internal/effort/` con las cuatro capas del proyecto (`domain`, `application`, `infrastructure/postgres`, `transport/http`), replicando las convenciones de `internal/task` (decodificación estricta, `400 invalid_request`, `422 validation_failed` con `fields`, `404 *_not_found`, `500 internal_error` sin detalle interno) y el mecanismo de habilitación por esquema de `internal/api` (`MigrationReadiness`, punteros en `HTTPDependencies`).

La persistencia usa una sola tabla `effort_records` con `story_id NOT NULL` y `task_id` anulable, creada por la migración `000012_create_effort_records`. La integridad entre proyecto, historia, integrante y tarea se garantiza dos veces: (1) verificación explícita dentro de la transacción del alta, con bloqueo `FOR KEY SHARE`, que produce errores de dominio tipados; y (2) claves foráneas compuestas en la base de datos, cuyas violaciones (`23503`) se traducen a los mismos errores de dominio.

La validación de campos se reparte así: la aplicación valida formato (UUID del cuerpo y `work_date` con parseo de ida y vuelta) y obtiene la fecha UTC actual de un reloj inyectado; el dominio valida reglas de negocio puras (descripción, horas, fecha no futura respecto de un `today` recibido por parámetro). Todos los errores de campo se acumulan en un único `*domain.ValidationError`.

Las ocho decisiones de producto de la propuesta son restricciones y no se reabren aquí.

## Estructura de módulos y archivos

```
internal/effort/
├── domain/
│   └── effort_record.go          EffortRecord, NewEffortRecordParams, NewEffortRecord, ValidationError
├── application/
│   └── register_effort.go        RegisterEffortCommand, EffortRepository, IDGenerator, Clock,
│                                 RegisterEffortUseCase, NewRegisterEffortUseCase, sentinelas Err*
├── infrastructure/postgres/
│   └── repository.go             PostgresEffortRepository, NewPostgresEffortRepository,
│                                 MapConstraintError
└── transport/http/
    └── handler.go                RegisterEffortHandler, NewRegisterEffortHandler
```

### Capa `domain` (`internal/effort/domain/effort_record.go`, paquete `domain`)

```go
const (
    MaxHours             = 24.0
    maxHoursDecimals     = 2
    MaxDescriptionLength = 500 // en caracteres (runas), después de TrimSpace

    hoursMessage       = "must be greater than 0, at most 24 and have at most 2 decimals"
    descriptionMessage = "must be at most 500 characters"
)

// EffortRecord es esfuerzo real registrado por un integrante. Siempre pertenece a
// StoryID; TaskID solo precisa el destino dentro de esa historia (nil = la historia).
type EffortRecord struct {
    ID          string
    ProjectID   string
    StoryID     string
    TaskID      *string
    MemberID    string
    WorkDate    time.Time // fecha civil en UTC a medianoche
    Description string    // ya recortada
    Hours       float64
}

// NewEffortRecordParams agrupa la entrada ya normalizada por la aplicación.
type NewEffortRecordParams struct {
    ID, ProjectID, StoryID, MemberID string
    TaskID      *string
    WorkDate    time.Time // cero = ya informado por la aplicación; no se revalida
    Description string
    Hours       *float64  // nil = ausente
}

// ValidationError describe entrada inválida por campo.
type ValidationError struct{ Fields map[string]string }
func (e *ValidationError) Error() string { return "effort record validation failed" }

// NewEffortRecord valida description, hours y work_date frente a today
// (fecha UTC a medianoche) y devuelve una copia sin aliasing de TaskID.
func NewEffortRecord(params NewEffortRecordParams, today time.Time) (EffortRecord, error)
```

Reglas del dominio:

| Campo | Regla | Mensaje |
|---|---|---|
| `description` | `strings.TrimSpace`; vacía → error | `is required` |
| `description` | `utf8.RuneCountInString(trimmed) > 500` | `must be at most 500 characters` |
| `hours` | `nil` | `is required` |
| `hours` | `!(v > 0) \|\| v > 24 \|\| decimalPlaces(v) > 2` | `must be greater than 0, at most 24 and have at most 2 decimals` |
| `work_date` | `!WorkDate.IsZero() && WorkDate.After(today)` | `must not be in the future` |

`decimalPlaces` replica la técnica de `internal/task/domain/task.go` (`strconv.FormatFloat(v, 'f', -1, 64)`), para no depender de aritmética de punto flotante. Se duplica deliberadamente en el módulo en lugar de exportarla desde `task` (ver decisiones). La descripción se almacena recortada (decisión 7 de la propuesta), a diferencia de `task.title`, que se guarda sin recortar.

### Capa `application` (`internal/effort/application/register_effort.go`, paquete `application`)

```go
var (
    ErrProjectNotFound = errors.New("project not found")
    ErrStoryNotFound   = errors.New("story not found")
    ErrMemberNotFound  = errors.New("member not found")
    ErrTaskNotFound    = errors.New("task not found")
)

// RegisterEffortCommand: IDs de ruta ya canónicos; campos del cuerpo tal como llegaron.
type RegisterEffortCommand struct {
    ProjectID   string
    StoryID     string
    MemberID    string
    TaskID      *string // nil = esfuerzo sobre la historia
    WorkDate    string  // YYYY-MM-DD
    Description string
    Hours       *float64
}

// EffortRepository verifica en una transacción que el proyecto existe, la historia y el
// integrante pertenecen a él y, si TaskID no es nil, la tarea pertenece a esa historia y
// proyecto; luego inserta el registro. Devuelve ErrProjectNotFound, ErrStoryNotFound,
// ErrMemberNotFound o ErrTaskNotFound; cualquier otro error se devuelve sin cambios.
type EffortRepository interface {
    Create(ctx context.Context, record domain.EffortRecord) error
}

type IDGenerator func() string
type Clock func() time.Time

type RegisterEffortUseCase struct { repository EffortRepository; generateID IDGenerator; now Clock }

func NewRegisterEffortUseCase(repository EffortRepository, generateID IDGenerator, now Clock) *RegisterEffortUseCase

// Execute valida todo, genera el ID solo si la entrada es válida, persiste y devuelve el
// registro creado únicamente si el repositorio confirma.
func (u *RegisterEffortUseCase) Execute(ctx context.Context, command RegisterEffortCommand) (domain.EffortRecord, error)
```

Flujo de `Execute`:

1. `fields := map[string]string{}`.
2. `member_id`: vacío → `is required`; `uuid.Parse` falla → `must be a valid UUID`; si es válido se normaliza a `parsed.String()` (forma canónica en minúsculas).
3. `task_id` (solo si no es `nil`): `uuid.Parse` falla (incluida la cadena vacía) → `must be a valid UUID`; si es válido se normaliza.
4. `work_date`: vacío → `is required`; `time.Parse("2006-01-02", v)` falla o `parsed.Format(layout) != v` → `must be a valid YYYY-MM-DD date`. Con error, se pasa `time.Time{}` al dominio.
5. `today := truncateToUTCDate(u.now())` = `time.Date(y, m, d, 0, 0, 0, 0, time.UTC)` sobre `now().UTC()`.
6. `domain.NewEffortRecord(params, today)`; si devuelve `*domain.ValidationError`, sus campos se fusionan en `fields` (no hay colisiones: el dominio no revalida una fecha cero ni los IDs).
7. Si `len(fields) > 0` → `&domain.ValidationError{Fields: fields}` sin llamar al generador de IDs ni al repositorio.
8. `record.ID = u.generateID()`; `u.repository.Create(ctx, record)`; error → se devuelve sin envolver (los sentinelas se comparan con `errors.Is`).

La comparación de fechas en UTC es la decisión 3: un cliente en UTC-3 a las 22:00 del día 24 ya está en el día 25 UTC y puede registrar el 25.

### Capa `infrastructure/postgres` (`internal/effort/infrastructure/postgres/repository.go`, paquete `postgres`)

```go
type PostgresEffortRepository struct{ pool *pgxpool.Pool }

func NewPostgresEffortRepository(pool *pgxpool.Pool) *PostgresEffortRepository
func (r *PostgresEffortRepository) Create(ctx context.Context, record domain.EffortRecord) error

// MapConstraintError traduce violaciones de FK de effort_records (23503) al error de
// dominio de la referencia violada; cualquier otro error se devuelve sin cambios.
func MapConstraintError(err error) error

var _ application.EffortRepository = (*PostgresEffortRepository)(nil)
```

Flujo transaccional de `Create` (una sola transacción, `defer tx.Rollback`):

| Paso | SQL | Sin fila → |
|---|---|---|
| 1 | `SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)` | `ErrProjectNotFound` |
| 2 | `SELECT 1 FROM stories WHERE id = $1 AND project_id = $2 FOR KEY SHARE` | `ErrStoryNotFound` |
| 3 | `SELECT 1 FROM project_members WHERE id = $1 AND project_id = $2 FOR KEY SHARE` | `ErrMemberNotFound` |
| 4 | solo si `TaskID != nil`: `SELECT 1 FROM tasks WHERE id = $1 AND story_id = $2 AND project_id = $3 FOR KEY SHARE` | `ErrTaskNotFound` |
| 5 | `INSERT INTO effort_records (id, project_id, story_id, task_id, member_id, work_date, description, hours) VALUES ($1..$8)` | error → `MapConstraintError(err)` |
| 6 | `tx.Commit(ctx)` | — |

- El orden fija la precedencia observable: `project_not_found` > `story_not_found` > `member_not_found` > `task_not_found`. Una tarea inexistente, de otra historia o de otro proyecto produce el mismo `ErrTaskNotFound` (decisión 5).
- `FOR KEY SHARE` es el mismo bloqueo que tomaría la FK y el que ya usa `internal/task`: impide que una eliminación concurrente invalide la verificación previa sin bloquear actualizaciones de columnas no clave (por ejemplo, el cierre de un Sprint).
- No se consulta el estado del Sprint ni la pertenencia de la historia a un Sprint (decisiones 1 y 2). Como una tarea solo existe dentro de `sprint_stories`, una tarea de un Sprint cerrado sigue siendo válida.
- `work_date` se pasa como `time.Time` a una columna `DATE` (pgx lo codifica como fecha civil); `hours` como `float64` a `NUMERIC(5,2)`; `task_id` como `*string` (`NULL` si es `nil`).

Mapeo de `MapConstraintError` (solo `pgErr.Code == "23503"`):

| `ConstraintName` | Error de dominio |
|---|---|
| `effort_records_story_project_fkey` | `ErrStoryNotFound` |
| `effort_records_member_project_fkey` | `ErrMemberNotFound` |
| `effort_records_task_story_project_fkey` | `ErrTaskNotFound` |
| otro / otro código | error original (termina en `500`) |

Las violaciones de `CHECK` (`23514`) no se mapean: la validación de dominio las vuelve inalcanzables y, si ocurrieran, indican un defecto que debe verse como `500`.

### Capa `transport/http` (`internal/effort/transport/http/handler.go`, paquete `transporthttp`)

```go
type RegisterEffortHandler struct{ useCase *application.RegisterEffortUseCase }
func NewRegisterEffortHandler(useCase *application.RegisterEffortUseCase) http.Handler

type registerEffortRequest struct {
    MemberID    string   `json:"member_id"`
    TaskID      *string  `json:"task_id"`
    WorkDate    string   `json:"work_date"`
    Description string   `json:"description"`
    Hours       *float64 `json:"hours"`
}

type effortRecordResponse struct {
    ID          string  `json:"id"`
    ProjectID   string  `json:"project_id"`
    StoryID     string  `json:"story_id"`
    TaskID      *string `json:"task_id"`    // siempre presente; null si es sobre la historia
    MemberID    string  `json:"member_id"`
    WorkDate    string  `json:"work_date"`  // YYYY-MM-DD
    Description string  `json:"description"` // recortada, tal como se persistió
    Hours       float64 `json:"hours"`
}

type errorResponse struct { Error, Message string; Fields map[string]string `json:"fields,omitempty"` }
```

Orden de evaluación en `ServeHTTP` (idéntico en estructura a `CreateTasksHandler`):

1. Método distinto de `POST` → `405 method_not_allowed` (defensivo; el patrón del mux ya filtra).
2. `decodeEffortObject`: un único objeto JSON (primer byte `{`, sin datos posteriores), `DisallowUnknownFields`. JSON inválido, no objeto, varios objetos, campo desconocido o tipo incorrecto (por ejemplo `"hours":"2"`) → `400 invalid_request` con el mismo mensaje que `task`.
3. `project_id` de la ruta no es UUID → `422 validation_failed` con `fields.project_id = "must be a valid UUID"`; luego lo mismo para `story_id`. Se devuelven de a uno, como en `task`.
4. Comando con IDs de ruta canónicos y campos del cuerpo tal como llegaron.
5. Mapeo de errores del caso de uso:

| Error | HTTP | `error` | `message` |
|---|---|---|---|
| `*domain.ValidationError` | 422 | `validation_failed` | `one or more fields are invalid` + `fields` |
| `ErrProjectNotFound` | 404 | `project_not_found` | `project not found` |
| `ErrStoryNotFound` | 404 | `story_not_found` | `story not found` |
| `ErrMemberNotFound` | 404 | `member_not_found` | `member not found` |
| `ErrTaskNotFound` | 404 | `task_not_found` | `task not found` |
| cualquier otro | 500 | `internal_error` | `an unexpected error occurred` |

6. Éxito → `201` con `effortRecordResponse` como objeto plano (sin envoltorio), `Content-Type: application/json`.

Ejemplo de respuesta `201`:

```json
{
  "id": "0f8b7c1e-...",
  "project_id": "82d38423-...",
  "story_id": "e99c05a4-...",
  "task_id": null,
  "member_id": "5c21cbd4-...",
  "work_date": "2026-10-24",
  "description": "Revisión del diseño",
  "hours": 2.5
}
```

## Migración `000012_create_effort_records`

Ubicación: `internal/project/infrastructure/postgres/migrations/`.

### `000012_create_effort_records.up.sql`

```sql
ALTER TABLE project_members
    ADD CONSTRAINT project_members_id_project_id_key UNIQUE (id, project_id);

ALTER TABLE tasks
    ADD CONSTRAINT tasks_id_story_id_project_id_key UNIQUE (id, story_id, project_id);

CREATE TABLE effort_records (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    story_id UUID NOT NULL,
    task_id UUID NULL,
    member_id UUID NOT NULL,
    work_date DATE NOT NULL,
    description TEXT NOT NULL
        CONSTRAINT effort_records_description_length
            CHECK (char_length(description) BETWEEN 1 AND 500),
    hours NUMERIC(5,2) NOT NULL
        CONSTRAINT effort_records_hours_range CHECK (hours > 0 AND hours <= 24),
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT effort_records_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT effort_records_member_project_fkey FOREIGN KEY (member_id, project_id)
        REFERENCES project_members (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT effort_records_task_story_project_fkey FOREIGN KEY (task_id, story_id, project_id)
        REFERENCES tasks (id, story_id, project_id) MATCH SIMPLE ON DELETE RESTRICT
);

CREATE INDEX effort_records_project_story_idx ON effort_records (project_id, story_id, seq);
CREATE INDEX effort_records_member_idx ON effort_records (member_id);
CREATE INDEX effort_records_task_idx ON effort_records (task_id) WHERE task_id IS NOT NULL;
```

### `000012_create_effort_records.down.sql`

```sql
DROP TABLE IF EXISTS effort_records;
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_id_story_id_project_id_key;
ALTER TABLE project_members DROP CONSTRAINT IF EXISTS project_members_id_project_id_key;
```

### Notas de la migración

- **Orden del `down`**: primero la tabla, porque sus FKs dependen de las dos `UNIQUE`; PostgreSQL rechaza eliminar una restricción única referenciada por una FK. El `down` es idempotente (`IF EXISTS`): ejecutarlo dos veces no falla. Esto se aparta del estilo de los `down` existentes (sin `IF EXISTS`) por requisito explícito de este cambio; el `up` mantiene el estilo del proyecto (sin `IF NOT EXISTS`), de modo que un `up` repetido sin `down` falla de forma visible en lugar de enmascarar un estado inconsistente.
- **Reaplicación**: `down` seguido de `up` recrea todo, también con filas existentes en `tasks` y `project_members`, porque las dos `UNIQUE` son superconjuntos de claves primarias y no pueden violarse.
- **`MATCH SIMPLE` con `task_id NULL`**: con `MATCH SIMPLE` (el valor por defecto, escrito de forma explícita para documentarlo) una FK compuesta no se evalúa si alguna columna referenciante es `NULL`. Como `story_id` y `project_id` son `NOT NULL`, la única columna anulable es `task_id`: un registro sobre la historia omite la FK de tarea y sigue sujeto a la FK de historia. `MATCH FULL` se descarta porque rechazaría justamente esas filas parcialmente nulas.
- **`ON DELETE RESTRICT`** en las tres FKs: consistente con todas las FKs del esquema; el esfuerzo es dato histórico para métricas y no existe ninguna ruta de eliminación de historias, integrantes ni tareas. Eliminar un padre referenciado falla con `23503` en lugar de borrar datos silenciosamente.
- **Sin FK directa a `projects`**: `effort_records_story_project_fkey` ya implica que el proyecto existe (`stories.project_id` referencia `projects`).
- **Sin `CHECK` de fecha futura**: depende del tiempo y no es una invariante estable de la fila; se valida solo en la aplicación con el reloj inyectado.
- **`NUMERIC(5,2)`** redondearía en silencio un tercer decimal; por eso el dominio rechaza más de dos decimales antes de llegar a la base.
- **Índices**: PostgreSQL no indexa automáticamente las columnas referenciantes. `effort_records_project_story_idx` sirve a la verificación de `RESTRICT` al eliminar historias y a la futura agregación de US-19 por historia (`seq` da orden de creación estable para un futuro `GET`); `effort_records_member_idx` y el índice parcial `effort_records_task_idx` sirven a las verificaciones de `RESTRICT` de integrantes y tareas.
- **`seq` y `created_at`** siguen la convención de `tasks`; no se exponen en la respuesta ni en la entidad de dominio en esta US.

## Cableado (`internal/api/api.go` y `cmd/api/main.go`)

### `internal/api/api.go`

```go
effortapplication "github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
efforthttp "github.com/valerubio7/software-metrics-and-estimation/internal/effort/transport/http"

// EffortDependencies enables effort registration after migration 000012 is clean.
// Now is optional: nil uses time.Now().UTC().
type EffortDependencies struct {
    Repository effortapplication.EffortRepository
    GenerateID effortapplication.IDGenerator
    Now        effortapplication.Clock
}

type HTTPDependencies struct {
    // ... campos existentes sin cambios ...
    // Effort nil: the effort route is not registered (the mux answers 404).
    Effort *EffortDependencies
}

type MigrationReadiness struct {
    // ... campos existentes sin cambios ...
    // Effort is true for a clean schema at version 12 or later (effort_records from migration 000012).
    Effort bool
}

// en ResolveMigrationReadiness:
readiness.Effort = version >= 12

// en NewHTTPHandlerWithDependencies, después del bloque de Completion:
if dependencies.Effort != nil {
    now := dependencies.Effort.Now
    if now == nil {
        now = func() time.Time { return time.Now().UTC() }
    }
    registerEffortUseCase := effortapplication.NewRegisterEffortUseCase(dependencies.Effort.Repository, dependencies.Effort.GenerateID, now)
    mux.Handle("POST /projects/{project_id}/stories/{story_id}/effort-records", efforthttp.NewRegisterEffortHandler(registerEffortUseCase))
}
```

El patrón nuevo no colisiona en `http.ServeMux` con `PUT /projects/{project_id}/stories/{story_id}`, `.../story-points` ni `GET .../tasks`: el segmento literal `effort-records` lo distingue. Con `Effort` nil, la ruta no existe y el mux responde `404`; un `GET` a la misma ruta con `Effort` presente responde `405` (no hay `GET`, decisión 6).

### `cmd/api/main.go`

Bloque nuevo después del de Completion, independiente de los demás gates (no reutiliza `dependencies.Stories`, `Members` ni `Tasks`, que podrían ser nil):

```go
if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Effort {
    dependencies.Effort = &api.EffortDependencies{
        Repository: effortpostgres.NewPostgresEffortRepository(pool),
        GenerateID: api.NewProjectID,
        Now:        func() time.Time { return time.Now().UTC() },
    }
    log.Printf("effort registration available (schema version=%d)", version)
} else {
    log.Printf("effort registration unavailable until migration 000012 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
}
```

Se actualiza también el comentario de cabecera del bloque de migraciones para mencionar la versión 12. Los IDs se generan con `api.NewProjectID` (`uuid.NewString()`), como en todos los módulos.

## Flujo de datos

```
POST /projects/{project_id}/stories/{story_id}/effort-records
        │
        ▼
RegisterEffortHandler ── 400 (JSON) / 422 (UUID de ruta)
        │ RegisterEffortCommand
        ▼
RegisterEffortUseCase ── Clock() → today UTC
        │   valida UUID del cuerpo + work_date (formato)
        │   domain.NewEffortRecord(params, today) → 422 acumulado
        │   generateID()
        ▼
PostgresEffortRepository.Create (una transacción)
        │   projects → stories FOR KEY SHARE → project_members FOR KEY SHARE
        │   → [tasks FOR KEY SHARE] → INSERT effort_records
        │   23503 → MapConstraintError → Err*NotFound
        ▼
PostgreSQL (FKs compuestas + CHECKs)
        │
        ▼
201 effortRecordResponse / 404 *_not_found / 500 internal_error
```

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/project/infrastructure/postgres/migrations/000012_create_effort_records.up.sql` | Crear | `UNIQUE` nuevas, tabla `effort_records`, FKs, `CHECK`s e índices. |
| `internal/project/infrastructure/postgres/migrations/000012_create_effort_records.down.sql` | Crear | `down` idempotente en orden tabla → `UNIQUE` de `tasks` → `UNIQUE` de `project_members`. |
| `internal/effort/domain/effort_record.go` | Crear | Entidad, parámetros, validación y `ValidationError`. |
| `internal/effort/application/register_effort.go` | Crear | Comando, puertos, reloj, sentinelas y caso de uso. |
| `internal/effort/infrastructure/postgres/repository.go` | Crear | Repositorio transaccional y `MapConstraintError`. |
| `internal/effort/transport/http/handler.go` | Crear | Handler, decodificación estricta, respuesta y mapeo de errores. |
| `internal/api/api.go` | Modificar | `EffortDependencies`, `HTTPDependencies.Effort`, `MigrationReadiness.Effort`, gate `>= 12` y ruta. |
| `cmd/api/main.go` | Modificar | Composición del módulo de esfuerzo con log de disponibilidad. |
| `tests/integration/migrations/migration_files_test.go` | Modificar | Se agrega `"000012"` a la lista canónica y se exigen 12 pares (`twelve`). |
| `tests/unit/effort/domain/effort_record_test.go` | Crear | Reglas de dominio. |
| `tests/unit/effort/application/register_effort_test.go` | Crear | Caso de uso con reloj fijo y fakes. |
| `tests/unit/effort/infrastructure/postgres/constraint_error_test.go` | Crear | `MapConstraintError` con `*pgconn.PgError` sintéticos. |
| `tests/unit/effort/transport/http/handler_test.go` | Crear | Tabla de casos HTTP. |
| `tests/unit/cmd/api/effort_route_test.go` | Crear | Readiness `>= 12` y composición de la ruta, sin dependencia de la TUI. |
| `tests/integration/effort/postgres/repository_integration_test.go` | Crear | Persistencia, referencias, FKs, `CHECK`s, `RESTRICT` y helpers de base. |
| `tests/integration/effort/postgres/migration_integration_test.go` | Crear | Esquema, `down`, `down` repetido y reaplicación. |
| `tests/integration/effort/postgres/http_integration_test.go` | Crear | Pila completa HTTP + PostgreSQL. |
| `README.md` | Modificar | Subsección breve con la ruta nueva de la API (ver riesgos). |
| `openspec/config.yaml` | Modificar | Reemplazo de la nota obsoleta sobre la migración `000003` duplicada. |

Total: 15 archivos nuevos y 5 modificados; ninguno eliminado.

## Contratos

### HTTP

`POST /projects/{project_id}/stories/{story_id}/effort-records`

Cuerpo:

```json
{ "member_id": "uuid", "task_id": "uuid | null | ausente", "work_date": "YYYY-MM-DD", "description": "texto", "hours": 2.5 }
```

| Situación | Respuesta |
|---|---|
| Registro válido | `201` + `effortRecordResponse` |
| Cuerpo no es un único objeto JSON válido, campo desconocido o tipo JSON incorrecto | `400 invalid_request` |
| `project_id` o `story_id` de la ruta no es UUID | `422 validation_failed` (un campo) |
| Campos del cuerpo inválidos (`member_id`, `task_id`, `work_date`, `description`, `hours`) | `422 validation_failed` con todos los `fields` |
| Proyecto inexistente | `404 project_not_found` |
| Historia inexistente o de otro proyecto | `404 story_not_found` |
| Integrante inexistente o de otro proyecto | `404 member_not_found` |
| Tarea inexistente, de otra historia o de otro proyecto | `404 task_not_found` |
| Error inesperado | `500 internal_error` sin detalle interno |

Precedencia: `400` > `422` de ruta > `422` de cuerpo > `404` (en el orden de la tabla) > `500`. Ninguna respuesta distinta de `201` persiste datos.

## Estrategia de pruebas

TDD estricto: cada unidad de trabajo empieza con la prueba en ROJO observada. `go test ./...` es el comando de referencia; las pruebas de integración se saltan con `-short` y, sin `-short`, fallan (no se omiten en silencio) si Docker no está disponible, igual que `internal/task`.

| Capa | Archivo | Qué se prueba | Enfoque |
|---|---|---|---|
| Unit dominio | `tests/unit/effort/domain/effort_record_test.go` | Descripción vacía, solo espacios, recortada al guardar, 500 vs 501 runas (con caracteres multibyte); horas `nil`, `0`, negativas, `0.01`, `24`, `24.01`, `2.555`, `2.5`; fecha igual a `today`, anterior y posterior; varios campos inválidos a la vez; `TaskID` copiado sin aliasing; fecha cero no revalidada. | Tabla de casos sobre `NewEffortRecord`. |
| Unit aplicación | `tests/unit/effort/application/register_effort_test.go` | `member_id` vacío/no UUID; `task_id` `nil`, vacío, no UUID, en mayúsculas (normalizado); `work_date` vacía, `2026-2-1`, `2026-02-30`, no futura; borde UTC con reloj fijo `2026-10-24T22:30:00-03:00` (hoy UTC = 25: se acepta el 25 y se rechaza el 26) y `2026-10-24T23:59:59Z`; acumulación de errores de aplicación y dominio; generador y repositorio no invocados ante `422`; ID generado asignado; errores del repositorio propagados sin cambios (`errors.Is`). | Reloj fijo, fake de repositorio que registra llamadas, generador contador. |
| Unit infraestructura | `tests/unit/effort/infrastructure/postgres/constraint_error_test.go` | Cada nombre de FK → sentinela; `23503` con otro nombre, `23514` y error no Postgres → sin cambios; `nil` → `nil`. | `*pgconn.PgError` sintéticos; prueba determinista de la defensa en profundidad, inalcanzable por la vía normal gracias a `FOR KEY SHARE`. |
| Unit HTTP | `tests/unit/effort/transport/http/handler_test.go` | `201` con `task_id` null y con valor, forma exacta del cuerpo y `Content-Type`; `400` (JSON roto, array, `null`, dos objetos, campo desconocido, `hours` como string); `422` de `project_id` y `story_id`; `422` con varios `fields`; los cuatro `404`; `500` sin filtrar el mensaje interno; método no `POST`. | Tabla de casos con `httptest` y caso de uso real sobre fake de repositorio y reloj fijo; la ruta se monta en un `http.ServeMux` local para que `PathValue` funcione. |
| Unit readiness/ruta | `tests/unit/cmd/api/effort_route_test.go` | `ResolveMigrationReadiness`: `Effort` falso en v11, verdadero en v12 y v13, falso con `dirty` o error de lectura; `Tasks` y `SprintHistory` conservan sus valores. Composición: con `Effort` nil → `404`; con dependencias → `201`; `GET` → `405`; reloj de `EffortDependencies.Now` respetado (fecha futura → `422`); `Now` nil usa el reloj por defecto. | `api.NewHTTPHandlerWithDependencies` con `fakeProjectRepository` y `serve` existentes del paquete `api_test`, más un `fakeEffortRepository` nuevo; sin importar `internal/tui`. |
| Integración repositorio | `tests/integration/effort/postgres/repository_integration_test.go` | Alta sobre historia (`task_id NULL`) y sobre tarea; historia en backlog; tarea de Sprint cerrado (`is_closed = true`); duplicados exactos permitidos; `hours` almacenado como `2.50`; `work_date` como `DATE`; descripción recortada; cada `404` (proyecto inexistente, historia de otro proyecto, integrante de otro proyecto, tarea inexistente, tarea de otra historia, tarea de otro proyecto) sin escrituras; inserciones SQL directas que violan cada FK con su `ConstraintName`; `CHECK` de horas (`0`, `24.01`) y descripción (`''`, 501 caracteres) con `23514`; `task_id NULL` acepta la fila (`MATCH SIMPLE`); `DELETE` de tarea, integrante e historia referenciados → `23503`. | Testcontainers `postgres:16-alpine` o `EFFORT_TEST_DATABASE_URL` con `testpostgres.OpenIsolated`; migraciones `000001`–`000012` aplicadas desde archivos, como en `tests/integration/task`. |
| Integración migración | `tests/integration/effort/postgres/migration_integration_test.go` | Tras `up`: tabla, columnas, las tres FKs, las dos `UNIQUE` y los tres índices (`pg_constraint`, `pg_indexes`). `down` con datos previos en `tasks` y `project_members`: tabla y `UNIQUE` eliminadas, datos de `tasks`/`project_members` intactos; segundo `down` sin error; `up` reaplicado restaura todo y admite un alta. | Mismo helper de base de datos. |
| Integración HTTP | `tests/integration/effort/postgres/http_integration_test.go` | `201` con fila persistida; `404 member_not_found` y `422` sin filas; `400` sin filas. | `api.NewHTTPHandlerWithDependencies` con el repositorio real y reloj fijo. |
| Archivos de migración | `tests/integration/migrations/migration_files_test.go` | 12 pares con `000012`. | Actualización de la lista y del conteo. |

## Matriz de amenazas

N/A: el cambio agrega una ruta HTTP dentro del proceso, sin comandos de shell, subprocesos, automatización de VCS/PR, clasificación de archivos ejecutables ni integración de procesos.

## Decisiones de arquitectura

| # | Decisión | Elección | Alternativas descartadas | Razón |
|---|---|---|---|---|
| D1 | Módulo | Módulo nuevo `internal/effort` con cuatro capas | Extender `internal/task` o `internal/story` | El esfuerzo es una capacidad propia (`esfuerzo`) consumida por US-19 y TUI-08; mezclarlo acoplaría su ciclo de vida al de tareas o historias. |
| D2 | Modelo de datos | Una tabla, `story_id NOT NULL`, `task_id` anulable | `CHECK` XOR con ambos anulables; dos tablas | Cerrado en la propuesta: agregación por historia sin `JOIN` y sin duplicar código. |
| D3 | Integridad | Verificación transaccional con `FOR KEY SHARE` + FKs compuestas | Solo verificación en aplicación; solo FKs y mapeo de `23503` | La verificación previa da precedencia de errores determinista y mensajes claros; las FKs protegen frente a escrituras fuera de la API. Solo con FKs, una fila con varias referencias inválidas reportaría la FK que PostgreSQL evalúe primero. |
| D4 | `MATCH` | `MATCH SIMPLE` explícito | `MATCH FULL` | `MATCH FULL` rechaza filas con `task_id NULL`, que son el caso de esfuerzo sobre la historia. |
| D5 | `ON DELETE` | `RESTRICT` en las tres FKs | `CASCADE`; `SET NULL` en `task_id` | Coherente con todo el esquema; el esfuerzo es histórico y no debe desaparecer en silencio. `SET NULL` sin lista de columnas anularía también `story_id` y `project_id` (`NOT NULL`) y fallaría; con lista de columnas cambiaría en silencio el destino del registro. |
| D6 | Índices | `(project_id, story_id, seq)`, `(member_id)`, parcial `(task_id) WHERE task_id IS NOT NULL` | Sin índices; un índice por FK completo | Cubren las verificaciones de `RESTRICT` y la lectura de US-19 sin indexar filas con `task_id NULL`. |
| D7 | `down` | Idempotente con `IF EXISTS`, tabla primero | Estilo actual sin `IF EXISTS` | Requisito explícito de este cambio; el orden es obligatorio por la dependencia FK → `UNIQUE`. |
| D8 | Reparto de validación | Formato (UUID, fecha) en aplicación; reglas en dominio con `today` como parámetro | Todo en el handler; reloj dentro del dominio | Sigue el patrón de `create_project.go` (parseo en aplicación) y mantiene el dominio puro y determinista. |
| D9 | Validación de UUID del cuerpo | En aplicación, como `422` por campo, con normalización canónica | En el handler con retorno temprano | Los errores de cuerpo deben devolverse juntos (propuesta); los de ruta siguen el patrón temprano de `task`. |
| D10 | Tipo JSON incorrecto | `400 invalid_request` | `422` por campo | Coherente con `task`, donde un error de decodificación es un problema estructural del cuerpo. |
| D11 | Reloj | `application.Clock func() time.Time` inyectado en el caso de uso; `EffortDependencies.Now` opcional con valor por defecto UTC | Reloj global; `time.Now` dentro del caso de uso | Pruebas deterministas en los bordes de día UTC y en las pruebas de ruta. |
| D12 | Respuesta `201` | Objeto plano con `task_id` siempre presente (null) y `work_date` como `YYYY-MM-DD`; sin `created_at` | Envoltorio `{"effort_record": ...}`; incluir `created_at`/`seq` | Las creaciones de un solo recurso (proyecto, historia, Sprint) devuelven objeto plano; el envoltorio de `task` existe por ser un lote. `task_id: null` sigue la convención de `estimated_hours: null`. `created_at` exigiría `RETURNING` y no lo pide ningún criterio. |
| D13 | Entidad sin `CreatedAt` | La marca de creación vive solo en la tabla | Campo `CreatedAt` en `EffortRecord` | La propuesta lo mencionaba, pero ningún consumidor de esta US lo lee; se agrega cuando exista un `GET`. Desviación menor y explícita. |
| D14 | `decimalPlaces` | Copia local en `internal/effort/domain` | Exportarla desde `internal/task/domain` | Evita acoplar dominios entre módulos por una función de cinco líneas. |
| D15 | Prueba del mapeo `23503` | `MapConstraintError` exportada y probada con errores sintéticos | Prueba de concurrencia; función no exportada sin prueba | Con `FOR KEY SHARE` la carrera no puede reproducirse de forma determinista; TDD estricto exige cubrir el código defensivo. |
| D16 | Descripción | Se guarda recortada; longitud en runas; `CHECK` con `char_length` | Guardar sin recortar como `task.title`; contar bytes | Decisión 7 de la propuesta; `char_length` cuenta caracteres igual que `utf8.RuneCountInString`. |

## Migración y despliegue

La migración `000012` se aplica externamente, como las anteriores. Hasta que `schema_migrations` esté limpio en versión `>= 12`, la ruta no se registra y el resto de la API conserva sus gates. Antes de abrir el PR se reconfirma que `000012` sigue siendo el siguiente número libre. El `down` elimina los registros de esfuerzo; no debe ejecutarse en un entorno persistente sin decidir antes su conservación.

## Unidades de trabajo y commits recomendados

Un solo PR con `size:exception` (decisión 8). Cada commit incluye pruebas, código y documentación de su unidad, compila y deja `go test ./...` en verde.

| # | Commit (Conventional Commits) | Contenido | Líneas aprox. |
|---|---|---|---|
| 1 | `feat(db): add effort records migration` | `000012` up/down, `migration_files_test.go` a 12 pares, `tests/integration/effort/postgres/migration_integration_test.go` con los helpers de base de datos | ~230 |
| 2 | `feat(effort): add effort record domain` | `internal/effort/domain/effort_record.go` + pruebas de dominio | ~230 |
| 3 | `feat(effort): add register effort use case` | `internal/effort/application/register_effort.go` + pruebas con reloj fijo | ~280 |
| 4 | `feat(effort): persist effort records in PostgreSQL` | `repository.go`, `MapConstraintError`, prueba unitaria del mapeo y `repository_integration_test.go` | ~380 |
| 5 | `feat(effort): expose effort registration over HTTP` | `handler.go` + pruebas unitarias HTTP | ~330 |
| 6 | `feat(api): gate effort registration on migration 12` | `api.go`, `main.go`, `effort_route_test.go`, `http_integration_test.go`, `README.md`, `openspec/config.yaml` | ~200 |

Total estimado: ~1.450–1.650 líneas, ligeramente por encima del rango previsto en la exploración debido a la prueba de migración dedicada y a la prueba del mapeo de restricciones.

## Riesgos

| Riesgo | Mitigación |
|---|---|
| `README.md` no tiene una tabla general de rutas de la API; la única tabla de rutas es la de lecturas de la TUI. Agregar el `POST` allí sería engañoso. | Se agrega una subsección breve "API de esfuerzo" con una tabla de una fila. La fase de tareas debe confirmarlo; es una interpretación de "fila nueva en la tabla de rutas" de la propuesta. |
| La spec se escribe en paralelo y podría fijar mensajes, precedencia de errores o forma de respuesta distintos. | Reconciliar en `sdd-tasks`: mensajes de `fields`, precedencia `member` > `task` y forma plana del `201` son los puntos más probables de divergencia. |
| La UTC del servidor difiere de la fecha local del usuario (UTC-3): entre las 21:00 y las 24:00 locales se admite la fecha del día siguiente. | Comportamiento aceptado por la decisión 3; queda documentado y probado en el borde. |
| `pgx` y `DATE`: codificar un `time.Time` no UTC podría desplazar el día. | La aplicación construye `WorkDate` con `time.Parse` (UTC a medianoche); prueba de integración que lee `work_date::text`. |
| Otro cambio toma `000012` antes de integrar. | Reconfirmar la cabeza del directorio de migraciones al aplicar y antes del PR. |
| Las pruebas de integración requieren Docker. | Ejecutarlas con Docker disponible e informar explícitamente cualquier omisión. |
| `MapConstraintError` exportada amplía la superficie pública del adaptador. | Se documenta como parte del contrato del adaptador; no la usa ningún otro paquete. |

## Preguntas abiertas

Ninguna bloqueante. A confirmar en `sdd-tasks` contra la spec: mensajes exactos de `fields` y la ubicación de la documentación en `README.md`.
