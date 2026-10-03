# Diseño: Descomponer una historia del Sprint en tareas (US-10)

## Enfoque técnico

Se agrega un corte vertical nuevo `internal/task/` con las mismas 4 capas que `story`, `sprint` y `projectmember` (dominio, aplicación, infraestructura PostgreSQL, transporte HTTP). El flujo es:

1. El handler HTTP decodifica de forma estricta el lote, valida los IDs de ruta y arma un comando.
2. El caso de uso valida el lote completo con el dominio (acumulando errores por elemento), genera los IDs y delega en un único puerto de persistencia.
3. El repositorio PostgreSQL, dentro de **una sola transacción**, verifica en orden proyecto → Sprint del proyecto → historia del proyecto → fila en `sprint_stories`, bloquea las filas leídas con `FOR KEY SHARE` e inserta todas las tareas. Cualquier fallo revierte el lote.
4. La migración `000009_create_tasks` crea la tabla `tasks` con FK compuesta hacia `sprint_stories(sprint_id, story_id)` (`ON DELETE RESTRICT`) y FKs compuestas por proyecto hacia `sprints` y `stories`.
5. La ruta se registra sólo si `ResolveMigrationReadiness` devuelve `Tasks = true` (esquema limpio y `version >= 9`), con un bloque propio en `cmd/api/main.go` independiente de `dependencies.Stories`.

Esto implementa las 6 decisiones técnicas de la propuesta sin modificar el código de `story`, `sprint` ni `projectmember`. La cabeza real del directorio `internal/project/infrastructure/postgres/migrations/` fue verificada al diseñar: el número más alto es `000008_add_sprint_closed` (más el fixture histórico `fixtures/000003_create_sprints`, que no cuenta como migración canónica). Por lo tanto la nueva migración es `000009`.

## Decisiones de arquitectura

### Decisión 1: Módulo independiente `internal/task/` sin importar `internal/story`

**Elección**: El módulo `task` tiene su propio dominio, errores y helpers. No importa paquetes de `story` (ni dominio, ni aplicación, ni transporte).
**Alternativas consideradas**: (a) agregar el caso de uso a `internal/story/application`; (b) extraer un paquete compartido para la validación de horas.
**Justificación**: Es el patrón existente: `sprint`, `story` y `projectmember` son cortes independientes que sólo se componen en `internal/api`. Extraer un paquete compartido obligaría a refactorizar `story`, fuera de alcance. La duplicación es mínima (una función de validación de horas y un decodificador JSON) y cada copia queda cubierta por sus propias pruebas.

### Decisión 2: Duplicar la regla de estimación de horas de `Story`

**Elección**: `internal/task/domain/task.go` define sus propias constantes `maxEstimatedHours = 99999.99`, `maxEstimatedHoursDecimal = 2`, el mismo mensaje `"must be greater than 0, at most 99999.99 and have at most 2 decimals"` y las funciones privadas `validateEstimatedHours` y `decimalPlaces`, con la misma semántica que `internal/story/domain/story.go:27-31,123-142` (nil es válido; NaN, 0, negativos, `> 99999.99` o más de 2 decimales sobre `strconv.FormatFloat(v, 'f', -1, 64)` son inválidos).
**Alternativas consideradas**: exportar `ValidateEstimatedHours` desde el dominio de `story`.
**Justificación**: Exportar cambiaría la API pública de `story` y acoplaría dos dominios. La regla es estable y la base de datos la refuerza con `NUMERIC(7,2)` y `CHECK (> 0)`.

### Decisión 3: Título obligatorio sin longitud máxima

**Elección**: El título es obligatorio (no vacío tras `strings.TrimSpace`), se persiste **tal como llega** (sin recortar, igual que `Story`) y no tiene longitud máxima.
**Alternativas consideradas**: fijar un máximo (por ejemplo 200 caracteres).
**Justificación**: La propuesta pide usar "el criterio usado para títulos de historia", y `Story` no tiene máximo (`stories.title TEXT NOT NULL`, `validateStoryContent` sólo exige no vacío). Introducir un máximo sería una regla nueva no pedida. Si la especificación fija uno, se agrega en `NewTask` sin cambiar el esquema.

### Decisión 4: Un único puerto que verifica pertenencia e inserta en la misma transacción

**Elección**: Puerto `TaskRepository.CreateForSprintStory(ctx, projectID, sprintID, storyID, tasks)` que verifica y escribe en una transacción.
**Alternativas consideradas**: (a) dos puertos separados (`SprintStoryMembershipChecker` + `TaskRepository.CreateBatch`) llamados en secuencia por el caso de uso; (b) reutilizar `AssignStoriesForProject`.
**Justificación**: Con dos puertos, la verificación y la escritura quedarían en transacciones distintas y habría una ventana de carrera. Es el mismo criterio que `ProjectScopedStorySprintAssigner.AssignStoriesForProject` (verificar y escribir de forma atómica). Reutilizar el método de asignación está prohibido por la propuesta y además tiene otra semántica (inserta asociaciones y rechaza Sprints cerrados).

### Decisión 5: Bloqueo `FOR KEY SHARE` en lugar de `FOR UPDATE`

**Elección**: Las lecturas de `sprints`, `stories` y `sprint_stories` dentro de la transacción usan `FOR KEY SHARE`.
**Alternativas consideradas**: `FOR UPDATE` sobre el Sprint, como hace `AssignStoriesForProject`.
**Justificación**: Las tareas sólo necesitan que las filas referenciadas no se borren ni cambien de clave hasta el commit (el mismo bloqueo que toma PostgreSQL al validar una FK). `FOR KEY SHARE` no entra en conflicto con `UPDATE sprints SET is_closed = ...` (no modifica claves) ni con otras altas de tareas, por lo que no serializa innecesariamente la creación de tareas con la asignación de US-09 ni con el cierre del Sprint. `FOR UPDATE` sí bloquearía la asignación y el cierre en paralelo sin aportar corrección adicional. Como las tareas se permiten en Sprints cerrados (decisión del usuario), no se lee `is_closed`.

### Decisión 6: Errores 404 por recurso y 409 `story_not_in_sprint`

**Elección**: Errores centinela en `internal/task/application`: `ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound`, `ErrStoryNotInSprint`. Un Sprint o una historia que existen pero pertenecen a otro proyecto se reportan como "no encontrado" (404), no como conflicto.
**Alternativas consideradas**: el código genérico `resource_not_found` y el 409 `assignment_conflict` por desajuste de proyecto que usa US-09.
**Justificación**: La propuesta fija los códigos `project_not_found`, `sprint_not_found`, `story_not_found` y `story_not_in_sprint`. Tratar un recurso ajeno como inexistente evita filtrar existencia entre proyectos y simplifica la consulta (se filtra por `project_id` en el `WHERE`). El 409 queda reservado para la regla de negocio "la historia no está asignada a este Sprint".

### Decisión 7: Validación del lote antes de cualquier acceso a persistencia

**Elección**: El caso de uso valida todos los elementos, acumula los errores en un único `*domain.ValidationError` con claves indexadas (`tasks[0].title`, `tasks[2].estimated_hours`) y sólo entonces genera IDs y llama al repositorio. En consecuencia, un lote inválido para un proyecto inexistente responde 422 (la validación precede a la existencia).
**Alternativas consideradas**: cortar en el primer error (como `RegisterMembersUseCase`, que devuelve `members[i]: ...`) o verificar existencia primero.
**Justificación**: La propuesta pide "rechazar todo el lote con 422 y el detalle por campo". Acumular todos los errores es coherente con `Story` (`ValidationError.Fields` con varias claves) y da al cliente toda la información de una vez. Validar antes evita abrir transacciones para entradas inválidas.

### Decisión 8: Inserción fila a fila dentro de la transacción

**Elección**: Un `tx.Exec("INSERT INTO tasks ...")` por tarea dentro de la misma transacción, como `PostgresMemberRepository.Register` y `AssignStoriesForProject`.
**Alternativas consideradas**: un único `INSERT ... SELECT FROM unnest($1::uuid[], $2::text[], $3::numeric[])` o `pgx.Batch`.
**Justificación**: La atomicidad la garantiza la transacción, no la forma del `INSERT`. Los lotes son pequeños (descomposición manual de una historia). `unnest` con arreglos de `*float64` nulables agrega riesgo de codificación sin beneficio medible y rompe con el patrón del repositorio.

### Decisión 9: FKs compuestas por proyecto además de la FK a `sprint_stories`

**Elección**: La tabla `tasks` tiene tres FKs compuestas: `(sprint_id, story_id) → sprint_stories`, `(sprint_id, project_id) → sprints(id, project_id)` y `(story_id, project_id) → stories(id, project_id)`, todas `ON DELETE RESTRICT`.
**Alternativas consideradas**: (a) sólo la FK a `sprint_stories` más una FK simple `project_id → projects`; (b) agregar una clave única `(sprint_id, story_id, project_id)` a `sprint_stories` y usar una sola FK de tres columnas.
**Justificación**: Con (a) la base aceptaría un `project_id` inconsistente con el Sprint/la historia. La opción (b) modifica la tabla de US-09. Las claves únicas `sprints_id_project_id_key` y `stories_id_project_id_key` ya existen (migración `000007`), así que las dos FKs compuestas no requieren cambios en tablas existentes y garantizan que el `project_id` de la tarea coincide con el de su Sprint y su historia.

### Decisión 10: Gate propio `Tasks` independiente de `Stories`

**Elección**: `MigrationReadiness.Tasks`, `HTTPDependencies.Tasks *TaskDependencies` y registro de la ruta sólo con `dependencies.Tasks != nil`. En `main.go`, el bloque crea su propio `PostgresTaskRepository` y no toca `dependencies.Stories`.
**Alternativas consideradas**: colgar el repositorio de tareas de `StoryDependencies` (como `Assigner`).
**Justificación**: El bloque de asignación (`main.go:70-71`) desreferencia `dependencies.Stories` asumiendo que el `switch` previo lo creó; la propuesta pide no repetir esa dependencia implícita. Un campo propio permite probar el gate de tareas sin componer historias.

## Flujo de datos

```
POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks
        │  body: {"tasks":[{"title":"...","estimated_hours":4.5}, ...]}
        ▼
taskhttp.CreateTasksHandler.ServeHTTP
   ├─ método != POST ............................ 405 method_not_allowed
   ├─ decodeTaskObject (estricto) falla ......... 400 invalid_request
   ├─ uuid.Parse(project_id|sprint_id|story_id) . 422 validation_failed {fields}
   ▼
taskapplication.CreateTasksUseCase.Execute(CreateTasksCommand)
   ├─ lote vacío / elementos inválidos .......... *domain.ValidationError → 422
   ├─ generateID() por tarea (en orden)
   ▼
TaskRepository.CreateForSprintStory(ctx, projectID, sprintID, storyID, tasks)
   │   (PostgresTaskRepository, una transacción)
   ├─ projects existe? ................ no → ErrProjectNotFound   → 404 project_not_found
   ├─ sprints (id, project_id) FOR KEY SHARE   no → ErrSprintNotFound → 404 sprint_not_found
   ├─ stories (id, project_id) FOR KEY SHARE   no → ErrStoryNotFound  → 404 story_not_found
   ├─ sprint_stories (sprint, story) FOR KEY SHARE no → ErrStoryNotInSprint → 409 story_not_in_sprint
   ├─ INSERT INTO tasks ... (una por tarea) error → rollback → 500 internal_error
   └─ COMMIT
        ▼
201 {"tasks":[{id, project_id, sprint_id, story_id, title, estimated_hours}, ...]}
```

Composición al arrancar:

```
cmd/api/main.go
  SELECT version, dirty FROM schema_migrations
  api.ResolveMigrationReadiness(version, dirty, err).Tasks  (version >= 9, limpio)
     ├─ true  → dependencies.Tasks = &api.TaskDependencies{taskpostgres.NewPostgresTaskRepository(pool), api.NewProjectID}
     └─ false → log "task creation unavailable until migration 000009 is clean ..."
  api.NewHTTPHandlerWithDependencies(...)
     └─ dependencies.Tasks != nil → mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks", ...)
```

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/task/domain/task.go` | Crear | `Task`, `ValidationError`, `NewTask`, validación de título y horas. |
| `internal/task/application/create_tasks.go` | Crear | Errores centinela, `TaskInput`, `CreateTasksCommand`, `TaskRepository`, `IDGenerator`, `CreateTasksUseCase`. |
| `internal/task/infrastructure/postgres/repository.go` | Crear | `PostgresTaskRepository` con `CreateForSprintStory` transaccional. |
| `internal/task/transport/http/handler.go` | Crear | `CreateTasksHandler`, DTOs, `decodeTaskObject`, `writeJSON`, mapeo de errores. |
| `internal/project/infrastructure/postgres/migrations/000009_create_tasks.up.sql` | Crear | Tabla `tasks`, FKs, checks e índice. |
| `internal/project/infrastructure/postgres/migrations/000009_create_tasks.down.sql` | Crear | `DROP TABLE tasks`. |
| `internal/api/api.go` | Modificar | `TaskDependencies`, `HTTPDependencies.Tasks`, `MigrationReadiness.Tasks`, umbral `>= 9`, registro condicional de la ruta. |
| `cmd/api/main.go` | Modificar | Bloque de gate de tareas con su propio repositorio y log. |
| `tests/unit/task/domain/task_test.go` | Crear | Pruebas de dominio. |
| `tests/unit/task/application/create_tasks_test.go` | Crear | Caso de uso con fake del puerto. |
| `tests/unit/task/transport/http/handler_test.go` | Crear | Handler por tabla de casos. |
| `tests/unit/cmd/api/main_test.go` | Modificar | Gate `>= 9`, independencia respecto de `Stories`, ampliación de la tabla de readiness. |
| `tests/integration/migrations/migration_files_test.go` | Modificar | Agregar `"000009"` a la secuencia canónica y pasar de 8 a 9 pares (hoy exige exactamente 8: fallaría al agregar la migración). |
| `tests/integration/task/postgres/repository_integration_test.go` | Crear | Testcontainers: persistencia, pertenencia, atomicidad, restricciones, reversibilidad de la migración. |
| `tests/integration/task/postgres/http_integration_test.go` | Crear | Composición HTTP real con PostgreSQL (201 y 409 sin escrituras). |
| `README.md` | Modificar | Documentar la ruta nueva y el umbral de esquema `000009` donde hoy se listan rutas/migraciones (línea ~142). |

## Interfaces y contratos

### Dominio — `internal/task/domain/task.go`

```go
package domain

// Task is a unit of work into which a story assigned to a sprint is decomposed.
type Task struct {
	ID             string
	ProjectID      string
	SprintID       string
	StoryID        string
	Title          string
	EstimatedHours *float64 // nil means "not estimated"
}

// ValidationError describes invalid task input by field.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string { return "task validation failed" }

// NewTask validates title (required, non-blank after TrimSpace, stored untrimmed) and the
// optional estimate (> 0, <= 99999.99, at most 2 decimals). On failure it returns
// *ValidationError with keys "title" and/or "estimated_hours". The returned task holds a
// copy of the estimate, never the caller's pointer.
func NewTask(id, projectID, sprintID, storyID, title string, estimatedHours *float64) (Task, error)
```

Mensajes: `title` → `"is required"`; `estimated_hours` → `"must be greater than 0, at most 99999.99 and have at most 2 decimals"`.

### Aplicación — `internal/task/application/create_tasks.go`

```go
package application

var (
	ErrProjectNotFound  = errors.New("project not found")
	ErrSprintNotFound   = errors.New("sprint not found")
	ErrStoryNotFound    = errors.New("story not found")
	ErrStoryNotInSprint = errors.New("story is not assigned to sprint")
)

// TaskInput is the client-supplied content of one task.
type TaskInput struct {
	Title          string
	EstimatedHours *float64
}

// CreateTasksCommand identifies the story within the sprint and the complete batch.
type CreateTasksCommand struct {
	ProjectID string
	SprintID  string
	StoryID   string
	Tasks     []TaskInput
}

// TaskRepository verifies, in one transaction, that the project exists, the sprint and the
// story belong to it and the story is assigned to the sprint, then inserts every task or
// none. It returns ErrProjectNotFound, ErrSprintNotFound, ErrStoryNotFound or
// ErrStoryNotInSprint for the corresponding failure; other errors are returned unchanged.
type TaskRepository interface {
	CreateForSprintStory(ctx context.Context, projectID, sprintID, storyID string, tasks []domain.Task) error
}

// IDGenerator creates identifiers for new tasks.
type IDGenerator func() string

type CreateTasksUseCase struct {
	repository TaskRepository
	generateID IDGenerator
}

func NewCreateTasksUseCase(repository TaskRepository, generateID IDGenerator) *CreateTasksUseCase

// Execute validates the whole batch, generates IDs and persists it atomically. It returns
// the created tasks only after the repository succeeds.
func (u *CreateTasksUseCase) Execute(ctx context.Context, command CreateTasksCommand) ([]domain.Task, error)
```

Algoritmo de `Execute`:

1. Si `len(command.Tasks) == 0` → `&domain.ValidationError{Fields: {"tasks": "must contain at least one task"}}`, sin llamar al generador ni al repositorio.
2. Para cada elemento `i`, `domain.NewTask("", ProjectID, SprintID, StoryID, input.Title, input.EstimatedHours)`. Si falla, copiar cada `field → message` como `fmt.Sprintf("tasks[%d].%s", i, field) → message` en un único mapa acumulado. Se procesan **todos** los elementos.
3. Si el mapa no está vacío → `&domain.ValidationError{Fields: acumulado}`; cero llamadas a `generateID` y al repositorio.
4. Asignar `task.ID = u.generateID()` en orden de entrada.
5. `repository.CreateForSprintStory(ctx, ProjectID, SprintID, StoryID, tasks)`; si falla, devolver `nil, err` (sin envolver, para que `errors.Is` funcione).
6. Devolver el slice creado.

### Infraestructura — `internal/task/infrastructure/postgres/repository.go`

```go
type PostgresTaskRepository struct{ pool *pgxpool.Pool }

func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository

func (r *PostgresTaskRepository) CreateForSprintStory(ctx context.Context, projectID, sprintID, storyID string, tasks []domain.Task) error

var _ application.TaskRepository = (*PostgresTaskRepository)(nil)
```

Secuencia SQL (todas con parámetros, dentro de `tx, err := r.pool.Begin(ctx)` y `defer func() { _ = tx.Rollback(ctx) }()`):

```sql
-- 1. Proyecto
SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1);
--    false → application.ErrProjectNotFound

-- 2. Sprint del proyecto (inexistente o ajeno → mismo resultado)
SELECT 1 FROM sprints WHERE id = $1 AND project_id = $2 FOR KEY SHARE;
--    pgx.ErrNoRows → application.ErrSprintNotFound

-- 3. Historia del proyecto (inexistente o ajena → mismo resultado)
SELECT 1 FROM stories WHERE id = $1 AND project_id = $2 FOR KEY SHARE;
--    pgx.ErrNoRows → application.ErrStoryNotFound

-- 4. Pertenencia historia-Sprint
SELECT 1 FROM sprint_stories WHERE sprint_id = $1 AND story_id = $2 FOR KEY SHARE;
--    pgx.ErrNoRows → application.ErrStoryNotInSprint

-- 5. Alta del lote (una sentencia por tarea, en orden)
INSERT INTO tasks (id, project_id, sprint_id, story_id, title, estimated_hours)
VALUES ($1, $2, $3, $4, $5, $6);
--    $6 = task.EstimatedHours (*float64; nil → NULL), igual que PostgresStoryRepository.Update
--    23503 con ConstraintName == "tasks_sprint_story_fkey" → application.ErrStoryNotInSprint
--    (defensa: no debería ocurrir tras el paso 4 con el bloqueo tomado)
--    cualquier otro error → se devuelve sin reinterpretar (rollback por defer)

COMMIT;
```

Notas:
- El `project_id` que se inserta es el de la ruta, ya verificado contra el Sprint y la historia en los pasos 2 y 3; las FKs compuestas lo refuerzan en base de datos.
- No se lee `sprints.is_closed`: las tareas se permiten en Sprints cerrados.
- Los IDs se pasan como `string`; PostgreSQL los convierte a `uuid` (el handler ya los normalizó con `uuid.Parse(...).String()`).

### Transporte HTTP — `internal/task/transport/http/handler.go`

Paquete `transporthttp` (mismo nombre de paquete que los demás módulos).

```go
type CreateTasksHandler struct{ useCase *application.CreateTasksUseCase }

func NewCreateTasksHandler(useCase *application.CreateTasksUseCase) http.Handler

type createTasksRequest struct {
	Tasks []taskRequest `json:"tasks"`
}
type taskRequest struct {
	Title          string   `json:"title"`
	EstimatedHours *float64 `json:"estimated_hours"`
}

type taskResponse struct {
	ID             string   `json:"id"`
	ProjectID      string   `json:"project_id"`
	SprintID       string   `json:"sprint_id"`
	StoryID        string   `json:"story_id"`
	Title          string   `json:"title"`
	EstimatedHours *float64 `json:"estimated_hours"`
}
type createTasksResponse struct {
	Tasks []taskResponse `json:"tasks"`
}

type errorResponse struct {
	Error   string            `json:"error"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}
```

`decodeTaskObject(body io.Reader, target any) error` replica la técnica de `decodeStoryObject` (`internal/story/transport/http/handler.go:113-147`): decodifica un `json.RawMessage`, exige que empiece con `{`, exige `io.EOF` tras el primer valor, decodifica con `DisallowUnknownFields` (que también aplica a los objetos anidados de `tasks`) y rechaza elementos `null` dentro del arreglo `tasks` (que de otro modo se decodificarían como una tarea vacía).

Orden de evaluación en `ServeHTTP` (igual que `AssignStoriesHandler`): método → cuerpo → IDs de ruta (`project_id`, `sprint_id`, `story_id`, en ese orden; se informa el primero inválido) → caso de uso.

#### Request

```json
{
  "tasks": [
    { "title": "Diseñar el esquema", "estimated_hours": 4.5 },
    { "title": "Escribir pruebas de integración" },
    { "title": "Revisar con el equipo", "estimated_hours": null }
  ]
}
```

`estimated_hours` ausente o `null` significan "sin estimar".

#### Respuesta 201 Created

```json
{
  "tasks": [
    {
      "id": "9b0c7f0e-6a51-4c0f-9a43-2f0b1f6d2c10",
      "project_id": "82d38423-f02d-4259-9e35-4a291585bb1e",
      "sprint_id": "b2a6a455-06e2-41ee-b011-7462c71375a0",
      "story_id": "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
      "title": "Diseñar el esquema",
      "estimated_hours": 4.5
    },
    {
      "id": "1d6b5c58-1f87-4a5e-bb0f-0b2f8e4a7c21",
      "project_id": "82d38423-f02d-4259-9e35-4a291585bb1e",
      "sprint_id": "b2a6a455-06e2-41ee-b011-7462c71375a0",
      "story_id": "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
      "title": "Escribir pruebas de integración",
      "estimated_hours": null
    }
  ]
}
```

El orden de la respuesta es el orden del request. `estimated_hours` siempre está presente (`null` si no hay estimación), igual que `storyResponse`.

#### Mapeo de errores

| Condición | Status | `error` | `message` | `fields` |
|---|---|---|---|---|
| Método distinto de POST (defensivo; el mux ya responde 405) | 405 | `method_not_allowed` | `only POST is supported` | — |
| Cuerpo no JSON, no objeto, varios valores, campo desconocido (raíz o tarea), tipo incorrecto, elemento `null` en `tasks` | 400 | `invalid_request` | `request body must be a single valid JSON object with allowed fields` | — |
| `project_id` no es UUID | 422 | `validation_failed` | `one or more fields are invalid` | `{"project_id": "must be a valid UUID"}` |
| `sprint_id` no es UUID | 422 | `validation_failed` | ídem | `{"sprint_id": "must be a valid UUID"}` |
| `story_id` no es UUID | 422 | `validation_failed` | ídem | `{"story_id": "must be a valid UUID"}` |
| `tasks` ausente, `null` o `[]` | 422 | `validation_failed` | ídem | `{"tasks": "must contain at least one task"}` |
| Título vacío o en blanco en el elemento `i` | 422 | `validation_failed` | ídem | `{"tasks[i].title": "is required"}` |
| Estimación inválida en el elemento `i` | 422 | `validation_failed` | ídem | `{"tasks[i].estimated_hours": "must be greater than 0, at most 99999.99 and have at most 2 decimals"}` |
| `ErrProjectNotFound` | 404 | `project_not_found` | `project not found` | — |
| `ErrSprintNotFound` (inexistente o de otro proyecto) | 404 | `sprint_not_found` | `sprint not found` | — |
| `ErrStoryNotFound` (inexistente o de otro proyecto) | 404 | `story_not_found` | `story not found` | — |
| `ErrStoryNotInSprint` | 409 | `story_not_in_sprint` | `story is not assigned to the selected sprint` | — |
| Cualquier otro error | 500 | `internal_error` | `an unexpected error occurred` | — |

El `*domain.ValidationError` se detecta con `errors.As`; los centinelas con `errors.Is`. El 500 nunca expone el detalle del error.

### Migración — `000009_create_tasks.up.sql`

```sql
CREATE TABLE tasks (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    sprint_id UUID NOT NULL,
    story_id UUID NOT NULL,
    title TEXT NOT NULL,
    estimated_hours NUMERIC(7,2) NULL
        CONSTRAINT tasks_estimated_hours_positive CHECK (estimated_hours > 0),
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tasks_sprint_story_fkey FOREIGN KEY (sprint_id, story_id)
        REFERENCES sprint_stories (sprint_id, story_id) ON DELETE RESTRICT,
    CONSTRAINT tasks_sprint_project_fkey FOREIGN KEY (sprint_id, project_id)
        REFERENCES sprints (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT tasks_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT
);

CREATE INDEX tasks_sprint_story_idx ON tasks (sprint_id, story_id, seq);
```

Notas de esquema:
- `NUMERIC(7,2)` + `CHECK (> 0)` replica exactamente la columna `stories.estimated_hours` (migración `000005`): tope `99999.99` y 2 decimales en base de datos.
- `seq` replica el patrón de `stories.seq` (migración `000004`) y da un orden de creación estable dentro del lote (en una misma transacción `now()` es idéntico para todas las filas). No se expone en el dominio en este alcance; lo podrá usar un futuro listado (US-15).
- `created_at` es la "marca de creación" pedida por la propuesta; no se expone en la respuesta.
- No hay `CHECK` de título en blanco: se replica `stories.title TEXT NOT NULL` y la regla vive en el dominio (`strings.TrimSpace` en Go y `btrim` en PostgreSQL no recortan los mismos caracteres; un `CHECK` con semántica distinta sería engañoso).
- No hay columna de estado (decisión del usuario).
- El índice cubre la consulta natural futura "tareas de una historia en un Sprint, en orden de creación" y además respalda la FK a `sprint_stories` (evita un barrido completo de `tasks` al validar un `DELETE` sobre `sprint_stories`).

### Migración — `000009_create_tasks.down.sql`

```sql
DROP TABLE tasks;
```

(El índice y las restricciones se eliminan con la tabla. No toca `sprint_stories`, `sprints` ni `stories`.)

### Cambios en `internal/api/api.go`

```go
import (
	// ...
	taskapplication "github.com/valerubio7/software-metrics-and-estimation/internal/task/application"
	taskhttp "github.com/valerubio7/software-metrics-and-estimation/internal/task/transport/http"
)

// TaskDependencies enables task creation after migration 000009 is clean.
type TaskDependencies struct {
	Repository taskapplication.TaskRepository
	GenerateID taskapplication.IDGenerator
}

type HTTPDependencies struct {
	Stories *StoryDependencies
	Sprints *SprintDependencies
	Members *MemberDependencies
	Tasks   *TaskDependencies // nil: the task route is not registered (the mux answers 404)
}

type MigrationReadiness struct {
	Projects   bool
	Stories    bool
	Sprints    bool
	Members    bool
	Assignment bool
	Tasks      bool
}

// en ResolveMigrationReadiness, después de Assignment:
readiness.Tasks = version >= 9

// en NewHTTPHandlerWithDependencies, después del bloque de Members (independiente de Stories):
if dependencies.Tasks != nil {
	createTasksUseCase := taskapplication.NewCreateTasksUseCase(dependencies.Tasks.Repository, dependencies.Tasks.GenerateID)
	mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks", taskhttp.NewCreateTasksHandler(createTasksUseCase))
}
```

El patrón no colisiona con `POST /projects/{project_id}/sprints/{sprint_id}/stories` ni con `PUT /projects/{project_id}/stories/{story_id}` (distinta cantidad de segmentos o distinto literal). Sin la dependencia, un POST a esa ruta responde 404, igual que la asignación sin `Assigner`.

### Cambios en `cmd/api/main.go`

```go
import taskpostgres "github.com/valerubio7/software-metrics-and-estimation/internal/task/infrastructure/postgres"

// después del bloque de Assignment y antes de NewHTTPHandlerWithDependencies:
if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Tasks {
	dependencies.Tasks = &api.TaskDependencies{Repository: taskpostgres.NewPostgresTaskRepository(pool), GenerateID: api.NewProjectID}
	log.Printf("task creation available (schema version=%d)", version)
} else {
	log.Printf("task creation unavailable until migration 000009 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
}
```

`api.NewProjectID` ya es el generador UUID genérico que usan historias, Sprints e integrantes. El bloque no lee ni escribe `dependencies.Stories`. Se actualiza el comentario de las líneas 35-37 para mencionar la versión 9.

## Estrategia de pruebas

Modo TDD estricto (`openspec/config.yaml: strict_tdd: true`, runner `go test ./...`). Cada archivo de prueba se escribe primero (RED) y luego la producción (GREEN). Pruebas por tabla con `t.Run(tc.name, ...)` y nombres por escenario (skill `go-testing`). Las de integración saltan con `testing.Short()` y nunca saltan silenciosamente si falta Docker (mismo criterio que `storyDatabase`).

| Capa | Qué se prueba | Enfoque |
|---|---|---|
| Unit dominio | `NewTask` | Tabla de casos |
| Unit aplicación | `CreateTasksUseCase` | Fake manual del puerto + generador contador |
| Unit HTTP | `CreateTasksHandler` | `httptest` + `SetPathValue`, tabla por status |
| Unit composición | Readiness `>= 9` y registro de ruta | `api.NewHTTPHandlerWithDependencies` con fakes |
| Integración | Repositorio, migración, restricciones, HTTP real | Testcontainers `postgres:16-alpine` o DSN local aislado |

### `tests/unit/task/domain/task_test.go` (paquete `domain_test`)

- Tarea válida: conserva IDs y título sin recortar (`"  Diseñar  "`), estimación nil.
- Estimación válida: `0.01`, `1.5`, `4.25`, `99999.99`; el puntero devuelto es una copia (mutar el original no cambia la tarea).
- Título inválido: `""`, `"   "`, `"\t\n"` → `Fields["title"] == "is required"`.
- Estimación inválida: `0`, `-1`, `100000`, `99999.991`, `1.234`, `NaN`, `+Inf` → `Fields["estimated_hours"]` con el mensaje exacto.
- Ambos inválidos a la vez → dos claves en `Fields`; `errors.As` a `*domain.ValidationError` funciona; `Error()` = `"task validation failed"`.

### `tests/unit/task/application/create_tasks_test.go` (paquete `application_test`)

Fake `fakeTaskRepository{calls int; projectID, sprintID, storyID string; tasks []domain.Task; err error}` y generador que devuelve IDs predecibles y cuenta llamadas.

- Éxito con 3 tareas: una llamada al repositorio con los tres IDs de ruta; IDs asignados en orden; título y estimación preservados; el resultado devuelto es igual a lo enviado al repositorio.
- Lote `nil` y `[]` → `ValidationError{"tasks": "must contain at least one task"}`, 0 llamadas al repositorio y al generador.
- Un elemento inválido entre válidos → clave `tasks[1].title`; 0 llamadas.
- Varios elementos inválidos → todas las claves (`tasks[0].title`, `tasks[2].estimated_hours`), sin cortar en el primero.
- Errores del repositorio: `ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound`, `ErrStoryNotInSprint` y un error genérico → `errors.Is` se conserva y el resultado es `nil`.
- Aislamiento: mutar el `*float64` del comando después de `Execute` no altera las tareas devueltas.

### `tests/unit/task/transport/http/handler_test.go` (paquete `transporthttp_test`)

Fake del puerto (`TaskRepository`) inyectado en un `CreateTasksUseCase` real, como `assign_stories_test.go`. Tabla con `name, body, path overrides, repoErr, wantStatus, wantError, wantFields, wantCalls`:

- `success` → 201, cuerpo con `tasks` en orden, `estimated_hours` presente (valor y `null`), 1 llamada con IDs normalizados.
- `success without estimate key` → 201, `estimated_hours: null`.
- `invalid project route`, `invalid sprint route`, `invalid story route` → 422 con la clave de campo correspondiente, 0 llamadas.
- `empty tasks`, `missing tasks`, `null tasks` → 422 `{"tasks": ...}`, 0 llamadas.
- `blank title`, `invalid estimate`, `multiple invalid elements` → 422 con claves indexadas, 0 llamadas.
- `unknown root field`, `unknown task field`, `trailing JSON`, `null body`, `array body`, `malformed`, `null task element`, `estimate as string` → 400 `invalid_request`, 0 llamadas.
- `project not found`, `sprint not found`, `story not found` → 404 con el código exacto, 1 llamada.
- `story not in sprint` → 409 `story_not_in_sprint`, 1 llamada.
- `unexpected` (`errors.New("secret db detail")`) → 500 `internal_error`, cuerpo sin `"secret"`.
- `wrong method` (GET directo al handler) → 405.
- Todas las respuestas tienen `Content-Type: application/json`.

### `tests/unit/cmd/api/main_test.go` (modificación)

- Nuevo `fakeTaskRepository` (cuenta llamadas, devuelve nil).
- Nuevo `TestTaskRouteRequiresCleanVersionNineAndExplicitDependency` con casos `v7`, `v8` (deshabilitado), `v9`, `future 12` (habilitado), `nil dependency`, `dirty v9`, `lookup error`, `missing migration table`: verifica `readiness.Tasks` y que un POST a `/projects/{id}/sprints/{id}/stories/{id}/tasks` con `{"tasks":[{"title":"A"}]}` responde 201 y 1 escritura sólo cuando corresponde; 404 y 0 escrituras en el resto.
- Caso de independencia: `HTTPDependencies{Tasks: ...}` con `Stories == nil` registra la ruta de tareas y responde 201 (y `POST /projects/{id}/stories` sigue sin registrarse).
- `TestMigrationReadinessSelectsRoutesIndependently`: agregar la columna `tasks` a la tabla; `clean v8` → `tasks: false`; `clean v9`/`clean future` → `tasks: true`; estados sucios y errores de lectura → `false`.

### `tests/integration/migrations/migration_files_test.go` (modificación)

- Agregar `"000009"` al arreglo de versiones canónicas y cambiar `len(pairs) != 8` por `!= 9` con el mensaje `"...exactly nine migration pairs"`.

### `tests/integration/task/postgres/repository_integration_test.go` (paquete `postgres_test`)

Helpers propios (no se comparten paquetes de prueba entre módulos): `taskDatabase(t)` (Testcontainers o `TASK_TEST_DATABASE_URL` vía `testpostgres.OpenIsolated(t, dsn, "TASK_TEST_DATABASE_URL", "task_test")`), `applyTaskMigration`, `taskModuleRoot`, `insertProject`, `insertSprint`, `insertStory` (SQL directo, sin depender del repositorio de historias), `assignStory` (INSERT directo en `sprint_stories`), `taskCount`, `assertDatabaseError` (código pg + nombre de restricción). Fixture `taskFixture(t)`: proyecto A con Sprint S1 y S2, historia H1 asignada a S1, historia H2 sin asignar; proyecto B con Sprint y historia propios.

- `TestTaskSchemaRequiresMigrationNine`: existe `tasks` con las columnas esperadas.
- `TestCreateTasksPersistsBatchLinkedToStorySprintAndProject`: dos tareas (una con `4.5`, otra nil); lectura SQL independiente devuelve `project_id`, `sprint_id`, `story_id`, título sin recortar, `estimated_hours::text` = `"4.50"` y `NULL`; `seq` creciente en el orden del lote; `created_at` no nulo.
- `TestCreateTasksRejectsMissingOrForeignResourcesWithoutWrites` (tabla): proyecto inexistente → `ErrProjectNotFound`; Sprint inexistente y Sprint del proyecto B → `ErrSprintNotFound`; historia inexistente e historia del proyecto B → `ErrStoryNotFound`; `taskCount == 0` tras cada caso.
- `TestCreateTasksRejectsStoryNotAssignedToSprint`: H2 (sin asignar) y H1 contra S2 (asignada a otro Sprint) → `ErrStoryNotInSprint`, 0 filas.
- `TestCreateTasksAllowsClosedSprint`: `UPDATE sprints SET is_closed = true` sobre S1 → éxito.
- `TestCreateTasksRollsBackWhenAnInsertFails`: trigger `BEFORE INSERT ON tasks` que falla en el segundo ID → error no clasificado como centinela, 0 filas (patrón de `TestAssignStoriesRollsBackWhenAnInsertFails`).
- `TestTasksTableEnforcesConstraints` (INSERT directo con `pool.Exec`): par historia-Sprint no asignado → `23503 tasks_sprint_story_fkey`; par asignado válido pero con `project_id` del proyecto B → `23503` cuyo nombre de restricción pertenece a `{tasks_sprint_project_fkey, tasks_story_project_fkey}` (ambas FKs fallan y PostgreSQL no garantiza cuál se reporta primero, así que la prueba acepta cualquiera de las dos, pero nunca un INSERT exitoso); `estimated_hours = 0` → `23514 tasks_estimated_hours_positive`; `estimated_hours = 100000` → `22003` (overflow numérico).
- `TestTaskBlocksUnassigningItsStory`: con una tarea creada, `DELETE FROM sprint_stories WHERE ...` → `23503 tasks_sprint_story_fkey` (`ON DELETE RESTRICT`).
- `TestTasksMigrationCanBeReversedAndReapplied`: aplicar `000009 down` (la tabla desaparece, `sprint_stories` intacta) y `000009 up` de nuevo.

### `tests/integration/task/postgres/http_integration_test.go`

Composición real `api.NewHTTPHandlerWithDependencies(projectpostgres..., api.NewProjectID, api.HTTPDependencies{Tasks: &api.TaskDependencies{Repository: taskpostgres.NewPostgresTaskRepository(pool), GenerateID: api.NewProjectID}})` sobre el mismo fixture:

- POST válido → 201, dos tareas con UUIDs generados, filas presentes en `tasks`.
- POST para historia no asignada → 409 `story_not_in_sprint`, 0 filas.
- POST con un elemento inválido → 422 con clave indexada, 0 filas.

## Matriz de amenazas

| Límite | Aplicabilidad |
|---|---|
| Rutas tipo documentación | N/A: el cambio no clasifica ni ejecuta archivos. |
| Selección de repositorio Git | N/A: no hay automatización de Git. |
| Estado de commit | N/A: no hay automatización de commits. |
| Estado de push | N/A: no hay automatización de push. |
| Comandos de PR | N/A: no hay automatización de PR. |

N/A: no hay límites de shell, subprocesos, automatización de VCS/PR, clasificación de archivos ejecutables ni integración de procesos. La ruta HTTP nueva usa el enrutamiento estándar de `net/http` ya existente; su seguridad de entrada (decodificación estricta, UUIDs validados, SQL parametrizado, 500 sin detalles) está cubierta por la estrategia de pruebas.

## Migración y despliegue

- La migración `000009` se aplica externamente, como las anteriores (`schema_migrations`). Mientras el esquema esté en `<= 8` o sucio, la API arranca sin la ruta de tareas y registra el log `task creation unavailable until migration 000009 is clean ...`; el resto de las rutas conserva sus gates.
- `000009_create_tasks.down.sql` elimina la tabla y sus datos: antes de revertir en un entorno persistente se debe decidir qué hacer con las tareas registradas (ver plan de reversión de la propuesta).
- Efecto en US-09: mientras exista al menos una tarea para un par historia-Sprint, la FK `tasks_sprint_story_fkey` impide borrar esa fila de `sprint_stories`. Hoy no existe operación de desasignación, por lo que no cambia ningún comportamiento actual; queda documentado como intencional.
- Antes de abrir el PR, reconfirmar que `000009` sigue siendo el siguiente número libre del directorio de migraciones.
- Tamaño: el cambio previsto (4 archivos de producción, 2 migraciones, 2 archivos modificados y ~7 archivos de prueba) supera con alta probabilidad el presupuesto de 400 líneas; la estrategia `single-pr` ya fue elegida por el usuario y `sdd-tasks` debe dejar el pronóstico explícito.

## Preguntas abiertas

- [ ] Ninguna que bloquee el diseño. La especificación (`sdd-spec`) puede fijar una longitud máxima de título; si lo hace, se agrega en `NewTask` (y como clave `tasks[i].title`) sin cambios de esquema.
- [ ] Opcional, no bloqueante: si se desea que el script `tests/integration/migrations/valid_history_convergence.sh` ejecute la suite de tareas en modo DSN, agregar `run_suite TASK_TEST_DATABASE_URL ./tests/integration/task/postgres task_suite`. No es necesario para `go test ./...` con Docker.
