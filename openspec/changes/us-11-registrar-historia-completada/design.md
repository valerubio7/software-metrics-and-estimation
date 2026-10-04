# Diseño: Registrar una historia del Sprint como completada (US-11)

## Enfoque técnico

Se extiende el módulo existente `internal/story` (dueño de la asociación `sprint_stories` desde US-09) con un corte vertical mínimo, sin crear un módulo nuevo:

1. La migración `000010_add_sprint_story_completion` agrega `completed_at TIMESTAMPTZ NULL` a `sprint_stories`. `NULL` significa "no completada en este Sprint".
2. El handler HTTP `CompleteSprintStoryHandler` valida los tres UUID de ruta (acumulando todos los inválidos en `fields`), **no lee el cuerpo** y arma un comando.
3. El caso de uso `CompleteSprintStoryUseCase` delega en un único puerto `SprintStoryCompleter`.
4. `PostgresStoryRepository.CompleteSprintStory`, en **una sola transacción**, verifica proyecto → Sprint del proyecto (`FOR UPDATE`) → historia del proyecto → Sprint cerrado → fila de `sprint_stories` (`FOR UPDATE`) → ya completada, y ejecuta `UPDATE ... SET completed_at = now() WHERE ... AND completed_at IS NULL RETURNING completed_at`.
5. La ruta se registra sólo si `ResolveMigrationReadiness` devuelve `Completion = true` (esquema limpio y `version >= 10`), con una dependencia opcional propia (`HTTPDependencies.Completion`) y un bloque propio en `cmd/api/main.go`, independiente de `dependencies.Stories`, igual que `Tasks`.

El diseño implementa el enfoque A de la propuesta y la delta `specs/historia/spec.md` (ya escrita), incluido su orden de precedencia de errores: `422` → `404` (proyecto, Sprint, historia) → `409 sprint_closed` → `409 story_not_in_sprint` → `409 story_already_completed`.

La cabeza real de `internal/project/infrastructure/postgres/migrations/` fue verificada al diseñar: el número más alto es `000009_create_tasks`. El subdirectorio `migrations/fixtures/` contiene `000003_create_sprints.{up,down}.sql` históricos; no cuenta como migración canónica porque `golang-migrate` y `migration_files_test.go` (`os.ReadDir` + `entry.IsDir()` → `continue`) no recorren subdirectorios. La nueva migración es `000010`.

## Decisiones de arquitectura

### Decisión 1: Extender `internal/story` en lugar de crear un módulo nuevo

**Elección**: Puerto, caso de uso, método de repositorio y handler nuevos dentro de `internal/story/{application,infrastructure/postgres,transport/http}`. No se toca `internal/story/domain`.
**Alternativas consideradas**: (a) módulo nuevo `internal/completion` o `internal/sprintstory`; (b) ubicarlo en `internal/sprint`; (c) ubicarlo en `internal/task`.
**Justificación**: `sprint_stories` lo creó y lo escribe sólo `PostgresStoryRepository.AssignStoriesForProject` (US-09); la finalización es otra transición sobre esa misma fila. `story/application` ya define `ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound` y `ErrSprintClosed`, que se reutilizan sin duplicar. Un módulo nuevo replicaría centinelas, `writeJSON` y `errorResponse` (como hizo US-10 por necesidad) y aumentaría el tamaño del PR, que ya está por encima del presupuesto. `sprint` no conoce `sprint_stories` y `task` es un agregado distinto. La capacidad funcional también es `historia` (ver la delta de spec).

### Decisión 2: El resultado es un tipo de aplicación, no una entidad de dominio

**Elección**: `application.SprintStoryCompletion{ProjectID, SprintID, StoryID string; CompletedAt time.Time}` devuelto por el puerto.
**Alternativas consideradas**: una entidad `domain.SprintStory` con un método `Complete()` que aplique las reglas.
**Justificación**: Todas las reglas (cerrado, pertenencia, ya completada, no doble conteo) dependen del estado concurrente de la base y sólo son correctas dentro de la transacción con bloqueos; una entidad en memoria duplicaría la regla sin poder garantizarla. Es el mismo criterio de `AssignStoriesForProject`, cuyas reglas viven en el repositorio. El tipo es una proyección de lectura sin invariantes.

### Decisión 3: El instante lo fija PostgreSQL (`now()`) y se devuelve con `RETURNING`

**Elección**: `SET completed_at = now()` y `RETURNING completed_at`; el valor devuelto es exactamente el persistido.
**Alternativas consideradas**: reloj inyectado en el caso de uso (como `GetProjectStatusHandler` con `time.Now().UTC()`), pasando el instante al repositorio.
**Justificación**: Una sola fuente de tiempo, coherente con `tasks.created_at DEFAULT now()`; evita desfases de reloj entre instancias de la API y la base; la spec exige que `completed_at` de la respuesta sea "el mismo valor persistido", lo que `RETURNING` garantiza sin redondeos (un `time.Time` de Go con nanosegundos se truncaría a microsegundos al guardarse y la respuesta diferiría del valor almacenado). Las pruebas no necesitan un reloj fijo: verifican que el valor no sea nulo, que esté entre un `before`/`after` medido en la prueba y que coincida con el leído por SQL.

### Decisión 4: Orden de verificación y de bloqueos en una única transacción

**Elección**: Secuencia `projects` (sin lock) → `sprints FOR UPDATE` → `stories FOR KEY SHARE` → chequeo de `is_closed` → `sprint_stories FOR UPDATE` → chequeo de `completed_at` → `UPDATE ... WHERE completed_at IS NULL`.
**Alternativas consideradas**: (a) `FOR SHARE` sobre el Sprint (permite finalizaciones concurrentes en el mismo Sprint y sigue bloqueando el cierre); (b) `FOR NO KEY UPDATE` sobre `sprint_stories` (no entra en conflicto con el `FOR KEY SHARE` de US-10); (c) un único `UPDATE ... FROM sprints JOIN stories` sin lecturas previas, clasificando el error después con una segunda consulta.
**Justificación**:
- El Sprint se bloquea **antes** que `sprint_stories`, en el mismo orden que `AssignStoriesForProject`, y con el mismo modo (`FOR UPDATE`). El futuro cierre de Sprint (US-12) hará `UPDATE sprints SET is_closed` (lock `NO KEY UPDATE`), que entra en conflicto con `FOR UPDATE`: el cierre y la finalización quedan serializados y `is_closed` leído no puede cambiar antes del commit.
- Análisis de interbloqueo: US-09 toma `sprints FOR UPDATE` y luego inserta en `sprint_stories`; US-10 toma `sprints FOR KEY SHARE`, `stories FOR KEY SHARE` y `sprint_stories FOR KEY SHARE`; US-11 toma `sprints FOR UPDATE` primero. Todas las rutas adquieren el Sprint antes que `sprint_stories`, así que no hay ciclo de espera. Una creación de tareas en curso sólo retrasa la finalización hasta su commit (y viceversa).
- (a) y (b) reducen contención, pero el lock del Sprint ya domina (cualquier transacción que llegue a `sprint_stories` pasó antes por el Sprint), así que la ganancia es nula para este volumen y se pierde la simetría con US-09.
- (c) produce dos lecturas no atómicas para clasificar errores; la spec exige que la pertenencia se resuelva "dentro de la misma operación".
- El `UPDATE` conserva `AND completed_at IS NULL` aunque la fila esté bloqueada: es la defensa en profundidad contra el doble conteo si algún día otra ruta escribe `completed_at` sin pasar por el lock del Sprint. Cero filas → `ErrStoryAlreadyCompleted`.

### Decisión 5: Centinelas reutilizados y nuevos, y mapeo HTTP al estilo US-10

**Elección**: Reutilizar `ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound` y `ErrSprintClosed` de `internal/story/application`; agregar `ErrStoryNotInSprint` y `ErrStoryAlreadyCompleted`. Códigos HTTP específicos por recurso (`project_not_found`, `sprint_not_found`, `story_not_found`, `story_not_in_sprint`, `sprint_closed`, `story_already_completed`), como `internal/task/transport/http/handler.go`. Un Sprint o una historia de otro proyecto se reportan como 404 (filtrado por `project_id` en el `WHERE`), no como `ErrProjectMismatch`.
**Alternativas consideradas**: el agrupamiento de US-09 (`404 resource_not_found`, `409 assignment_conflict`).
**Justificación**: Lo fija la propuesta y la delta de spec. Misma respuesta que US-10 para la misma situación en la misma jerarquía de rutas. `ErrProjectMismatch` no se usa en esta ruta.

### Decisión 6: Validación de los tres UUID acumulada en `fields`

**Elección**: El handler parsea `project_id`, `sprint_id` y `story_id` y acumula **todos** los inválidos en un único `422 validation_failed`.
**Alternativas consideradas**: cortar en el primero inválido, como `CreateTasksHandler` y `AssignStoriesHandler`.
**Justificación**: La delta de spec exige el escenario "Varios identificadores inválidos a la vez" con los tres en `fields`. Es una diferencia deliberada y local respecto de US-10; el resto del formato (`message: "one or more fields are invalid"`, `"must be a valid UUID"`) es idéntico.

### Decisión 7: El cuerpo de la solicitud no se lee

**Elección**: `ServeHTTP` no decodifica ni valida `request.Body`; cualquier contenido (incluido `{"x":1}` o JSON inválido) se ignora.
**Alternativas consideradas**: rechazar con `400 invalid_request` un cuerpo no vacío.
**Justificación**: La spec exige `200` "igual que sin cuerpo" y que el sistema "MUST NOT exigir ni interpretar un cuerpo". `net/http` descarta el cuerpo no leído al cerrar la solicitud.

### Decisión 8: Respuesta `200` con `completed_at` en UTC

**Elección**: `200 OK` con `{"project_id","sprint_id","story_id","completed_at"}`, IDs canónicos (minúsculas, de `uuid.Parse(...).String()`) y `completed_at` como `time.Time` en UTC (`CompletedAt.UTC()`), serializado por `encoding/json` en RFC 3339 con fracción.
**Alternativas consideradas**: `201 Created` (registro de un subrecurso `completion`); `204` sin cuerpo.
**Justificación**: La operación actualiza una fila existente y la spec fija `200` con esos campos. `pgx` devuelve `timestamptz` en la zona local del proceso; normalizar a UTC hace la respuesta determinista entre entornos sin cambiar el instante.

### Decisión 9: Gate `Completion` propio e independiente de `Stories`

**Elección**: `MigrationReadiness.Completion = version >= 10`; `HTTPDependencies.Completion *CompletionDependencies`; ruta registrada sólo con `dependencies.Completion != nil`; en `main.go`, bloque propio que crea su `PostgresStoryRepository` sin leer ni escribir `dependencies.Stories`.
**Alternativas consideradas**: agregar un campo `Completer` a `StoryDependencies` (como `Assigner`).
**Justificación**: El bloque de asignación de `main.go` desreferencia `dependencies.Stories` asumiendo que el `switch` previo lo creó; US-10 ya evitó repetir esa dependencia implícita y aquí se sigue el mismo criterio. Permite probar el gate sin componer historias. Se usa un struct puntero (no una interfaz directa en `HTTPDependencies`) para evitar el problema de interfaz no nula con puntero nulo y mantener el patrón de `TaskDependencies`.

### Decisión 10: Migración mínima sin índice ni default

**Elección**: `ALTER TABLE sprint_stories ADD COLUMN completed_at TIMESTAMPTZ NULL;` y su `DROP COLUMN` inverso.
**Alternativas consideradas**: índice parcial `WHERE completed_at IS NOT NULL`; `CHECK` de coherencia con `stories.status`; tabla separada `sprint_story_completions`.
**Justificación**: Todas las lecturas actuales y futuras (US-19) filtran por `sprint_id`, prefijo de la PK `(sprint_id, story_id)`; un índice adicional no aporta. Una columna nulable sin default es un cambio sólo de catálogo en PostgreSQL 11+ (sin reescritura de la tabla). `stories.status` queda deliberadamente desacoplado (spec). La tabla separada fue descartada en la exploración.

## Flujo de datos

```
POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion
        │  (cuerpo ignorado)
        ▼
storyhttp.CompleteSprintStoryHandler.ServeHTTP
   ├─ método != POST (defensivo) ................ 405 method_not_allowed
   ├─ uuid.Parse x3, acumulando ................. 422 validation_failed {fields: 1..3 claves}
   ▼
storyapplication.CompleteSprintStoryUseCase.Execute(CompleteSprintStoryCommand)
   ▼
SprintStoryCompleter.CompleteSprintStory(ctx, projectID, sprintID, storyID)
   │   (PostgresStoryRepository, una transacción)
   ├─ projects existe? ........................ no → ErrProjectNotFound       → 404 project_not_found
   ├─ sprints (id, project_id) FOR UPDATE ...... no → ErrSprintNotFound        → 404 sprint_not_found
   ├─ stories (id, project_id) FOR KEY SHARE ... no → ErrStoryNotFound         → 404 story_not_found
   ├─ is_closed? ............................... sí → ErrSprintClosed          → 409 sprint_closed
   ├─ sprint_stories (sprint, story) FOR UPDATE  no → ErrStoryNotInSprint      → 409 story_not_in_sprint
   ├─ completed_at IS NOT NULL? ................ sí → ErrStoryAlreadyCompleted → 409 story_already_completed
   ├─ UPDATE ... SET completed_at = now()
   │     WHERE ... AND completed_at IS NULL RETURNING completed_at
   │                                           0 filas → ErrStoryAlreadyCompleted
   │                                           error   → rollback → 500 internal_error
   └─ COMMIT
        ▼
200 {"project_id","sprint_id","story_id","completed_at"}
```

Composición al arrancar:

```
cmd/api/main.go
  SELECT version, dirty FROM schema_migrations
  api.ResolveMigrationReadiness(version, dirty, err).Completion  (version >= 10, limpio)
     ├─ true  → dependencies.Completion = &api.CompletionDependencies{Completer: storypostgres.NewPostgresStoryRepository(pool)}
     └─ false → log "story completion unavailable until migration 000010 is clean ..."
  api.NewHTTPHandlerWithDependencies(...)
     └─ dependencies.Completion != nil → mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion", ...)
```

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/project/infrastructure/postgres/migrations/000010_add_sprint_story_completion.up.sql` | Crear | Agrega `sprint_stories.completed_at TIMESTAMPTZ NULL`. |
| `internal/project/infrastructure/postgres/migrations/000010_add_sprint_story_completion.down.sql` | Crear | Elimina la columna (pierde los registros de finalización). |
| `internal/story/application/complete_sprint_story.go` | Crear | `ErrStoryNotInSprint`, `ErrStoryAlreadyCompleted`, `SprintStoryCompletion`, `CompleteSprintStoryCommand`, `SprintStoryCompleter`, `CompleteSprintStoryUseCase`. |
| `internal/story/infrastructure/postgres/repository.go` | Modificar | Método `CompleteSprintStory` transaccional y aserción `_ application.SprintStoryCompleter`. |
| `internal/story/transport/http/complete_sprint_story_handler.go` | Crear | `CompleteSprintStoryHandler`, DTO de respuesta y mapeo de errores; reutiliza `writeJSON` y `errorResponse` del paquete. |
| `internal/api/api.go` | Modificar | `CompletionDependencies`, `HTTPDependencies.Completion`, `MigrationReadiness.Completion`, umbral `>= 10`, registro condicional de la ruta. |
| `cmd/api/main.go` | Modificar | Bloque de gate de finalización con log; comentario de versiones actualizado. |
| `tests/unit/story/application/complete_sprint_story_test.go` | Crear | Caso de uso con fake del puerto. |
| `tests/unit/story/transport/http/complete_sprint_story_handler_test.go` | Crear | Handler por tabla de casos. |
| `tests/unit/cmd/api/main_test.go` | Modificar | Gate `>= 10`, independencia respecto de `Stories`. |
| `tests/integration/migrations/migration_files_test.go` | Modificar | Agregar `"000010"` y pasar de 9 a 10 pares. |
| `tests/integration/story/postgres/completion_integration_test.go` | Crear | Testcontainers: esquema, persistencia, reglas, no doble conteo, concurrencia, reversibilidad y composición HTTP real. |
| `README.md` | Modificar | Sección US-11 y actualización de "Migraciones y disponibilidad" (`000001`–`000010`, `>=10`). |
| `openspec/config.yaml` | Modificar | Nota `"Implementada: US-11 (registrar historia completada en el Sprint)."`. |
| `docs/scrum/sprint-1.md` | Modificar | Nota fechada de que el registro de finalización está implementado en código, sin reescribir la instantánea del tablero. |

No se modifican `internal/story/domain`, `internal/task`, `internal/sprint`, `docs/traceability.md` ni `docs/architecture`.

## Interfaces y contratos

### Aplicación — `internal/story/application/complete_sprint_story.go`

```go
package application

var (
	ErrStoryNotInSprint      = errors.New("story is not assigned to sprint")
	ErrStoryAlreadyCompleted = errors.New("story already completed in sprint")
)

// SprintStoryCompletion is the recorded completion of one story within one sprint.
type SprintStoryCompletion struct {
	ProjectID   string
	SprintID    string
	StoryID     string
	CompletedAt time.Time
}

// CompleteSprintStoryCommand identifies the story to complete within the sprint.
type CompleteSprintStoryCommand struct {
	ProjectID string
	SprintID  string
	StoryID   string
}

// SprintStoryCompleter verifies, in one transaction, that the project exists, the sprint
// and the story belong to it, the sprint is open, the story is assigned to the sprint and
// not yet completed there, then records the completion instant. It returns
// ErrProjectNotFound, ErrSprintNotFound, ErrStoryNotFound, ErrSprintClosed,
// ErrStoryNotInSprint or ErrStoryAlreadyCompleted, in that precedence; other errors are
// returned unchanged.
type SprintStoryCompleter interface {
	CompleteSprintStory(ctx context.Context, projectID, sprintID, storyID string) (SprintStoryCompletion, error)
}

// CompleteSprintStoryUseCase records the completion of a story within a sprint.
type CompleteSprintStoryUseCase struct {
	completer SprintStoryCompleter
}

func NewCompleteSprintStoryUseCase(completer SprintStoryCompleter) *CompleteSprintStoryUseCase

// Execute delegates to the completer and returns its result unchanged (errors unwrapped,
// so errors.Is keeps working).
func (u *CompleteSprintStoryUseCase) Execute(ctx context.Context, command CompleteSprintStoryCommand) (SprintStoryCompletion, error)
```

El caso de uso no revalida formato de IDs (lo hace el handler, igual que US-09/US-10) y no consulta `stories.status`.

### Infraestructura — `PostgresStoryRepository.CompleteSprintStory`

```go
func (r *PostgresStoryRepository) CompleteSprintStory(ctx context.Context, projectID, sprintID, storyID string) (application.SprintStoryCompletion, error)
```

Secuencia SQL dentro de `tx, err := r.pool.Begin(ctx)` y `defer func() { _ = tx.Rollback(ctx) }()`:

```sql
-- 1. Proyecto (sin bloqueo, igual que US-10)
SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1);
--    false → ErrProjectNotFound

-- 2. Sprint del proyecto; bloquea frente a asignación (US-09), cierre (US-12) y otras finalizaciones
SELECT is_closed FROM sprints WHERE id = $1 AND project_id = $2 FOR UPDATE;
--    pgx.ErrNoRows → ErrSprintNotFound   (inexistente o de otro proyecto)

-- 3. Historia del proyecto
SELECT 1 FROM stories WHERE id = $1 AND project_id = $2 FOR KEY SHARE;
--    pgx.ErrNoRows → ErrStoryNotFound    (inexistente o de otro proyecto)

-- 4. (en Go) is_closed = true → ErrSprintClosed

-- 5. Pertenencia y estado de finalización
SELECT completed_at FROM sprint_stories WHERE sprint_id = $1 AND story_id = $2 FOR UPDATE;
--    pgx.ErrNoRows → ErrStoryNotInSprint
--    completed_at != nil (*time.Time) → ErrStoryAlreadyCompleted

-- 6. Registro
UPDATE sprint_stories SET completed_at = now()
WHERE sprint_id = $1 AND story_id = $2 AND completed_at IS NULL
RETURNING completed_at;
--    pgx.ErrNoRows → ErrStoryAlreadyCompleted (defensa)
--    otro error → se devuelve sin reinterpretar (rollback por defer)

COMMIT;  -- error de commit → se devuelve sin reinterpretar
```

El resultado lleva los IDs recibidos (ya canónicos desde el handler) y el `completed_at` devuelto por `RETURNING`. El `project_id` de `sprint_stories` no se relee: las FKs compuestas de US-09 garantizan que coincide con el del Sprint ya verificado.

### Transporte HTTP — `internal/story/transport/http/complete_sprint_story_handler.go`

```go
type CompleteSprintStoryHandler struct{ useCase *application.CompleteSprintStoryUseCase }

func NewCompleteSprintStoryHandler(useCase *application.CompleteSprintStoryUseCase) http.Handler

type sprintStoryCompletionResponse struct {
	ProjectID   string    `json:"project_id"`
	SprintID    string    `json:"sprint_id"`
	StoryID     string    `json:"story_id"`
	CompletedAt time.Time `json:"completed_at"`
}
```

Orden en `ServeHTTP`: método → los tres UUID (acumulados en un mapa `fields`; si no está vacío, `422`) → caso de uso → mapeo. No se lee el cuerpo.

| Condición | Status | `error` | `message` | `fields` |
|---|---|---|---|---|
| Método distinto de POST (defensivo; el mux ya responde 405) | 405 | `method_not_allowed` | `only POST is supported` | — |
| Uno o más UUID de ruta inválidos | 422 | `validation_failed` | `one or more fields are invalid` | `{"project_id"|"sprint_id"|"story_id": "must be a valid UUID"}` (todas las inválidas) |
| `ErrProjectNotFound` | 404 | `project_not_found` | `project not found` | — |
| `ErrSprintNotFound` | 404 | `sprint_not_found` | `sprint not found` | — |
| `ErrStoryNotFound` | 404 | `story_not_found` | `story not found` | — |
| `ErrSprintClosed` | 409 | `sprint_closed` | `sprint is closed` | — |
| `ErrStoryNotInSprint` | 409 | `story_not_in_sprint` | `story is not assigned to the selected sprint` | — |
| `ErrStoryAlreadyCompleted` | 409 | `story_already_completed` | `story is already completed in the selected sprint` | — |
| Cualquier otro error | 500 | `internal_error` | `an unexpected error occurred` | — |
| Éxito | 200 | — | — | cuerpo `sprintStoryCompletionResponse` |

Ejemplo de respuesta `200`:

```json
{
  "project_id": "82d38423-f02d-4259-9e35-4a291585bb1e",
  "sprint_id": "b2a6a455-06e2-41ee-b011-7462c71375a0",
  "story_id": "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
  "completed_at": "2026-10-04T14:03:21.123456Z"
}
```

### Migraciones

`000010_add_sprint_story_completion.up.sql`:

```sql
ALTER TABLE sprint_stories ADD COLUMN completed_at TIMESTAMPTZ NULL;
```

`000010_add_sprint_story_completion.down.sql`:

```sql
ALTER TABLE sprint_stories DROP COLUMN completed_at;
```

No afecta la PK `sprint_stories_pkey`, las FKs compuestas de `000007` ni la FK `tasks_sprint_story_fkey` de `000009`.

### Cambios en `internal/api/api.go`

```go
// CompletionDependencies enables story completion within a sprint after migration 000010 is clean.
type CompletionDependencies struct {
	Completer storyapplication.SprintStoryCompleter
}

type HTTPDependencies struct {
	Stories    *StoryDependencies
	Sprints    *SprintDependencies
	Members    *MemberDependencies
	Tasks      *TaskDependencies
	Completion *CompletionDependencies // nil: the completion route is not registered (the mux answers 404)
}

type MigrationReadiness struct {
	// ... campos existentes
	Completion bool
}

// en ResolveMigrationReadiness, después de Tasks:
readiness.Completion = version >= 10

// en NewHTTPHandlerWithDependencies, después del bloque de Tasks (independiente de Stories):
if dependencies.Completion != nil {
	completeUseCase := storyapplication.NewCompleteSprintStoryUseCase(dependencies.Completion.Completer)
	mux.Handle("POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion", storyhttp.NewCompleteSprintStoryHandler(completeUseCase))
}
```

El patrón no colisiona con `.../stories/{story_id}/tasks` (literal distinto) ni con `POST .../sprints/{sprint_id}/stories` (distinta cantidad de segmentos).

### Cambios en `cmd/api/main.go`

```go
if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Completion {
	dependencies.Completion = &api.CompletionDependencies{Completer: storypostgres.NewPostgresStoryRepository(pool)}
	log.Printf("story completion available (schema version=%d)", version)
} else {
	log.Printf("story completion unavailable until migration 000010 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
}
```

Se actualiza el comentario de las líneas 36-38 para mencionar la versión 10. Nota sobre el umbral duplicado: el número `10` aparece en `ResolveMigrationReadiness` (fuente de verdad del gate) y en el texto del log de `main.go`; además el `switch` de historias de `main.go` mantiene umbrales manuales (`>= 4/3/2`) fuera de `ResolveMigrationReadiness`. Este cambio no agrega un umbral manual nuevo (el bloque usa sólo `readiness.Completion`) y no corrige la duplicación histórica (fuera de alcance según la propuesta); las pruebas de `main_test.go` fijan el valor `10`.

## Estrategia de pruebas

TDD estricto (`openspec/config.yaml: strict_tdd: true`, runner `go test ./...`): cada prueba se escribe primero (RED observado) y luego la producción (GREEN). Pruebas por tabla con `t.Run`. Las de integración saltan con `testing.Short()` y fallan (no saltan) si falta Docker, como `storyDatabase`. Para mantener el PR lo más pequeño posible, cada regla se prueba una vez en la capa donde vive: el mapeo de errores en HTTP unitario, las reglas y la concurrencia en integración de repositorio, y la composición real sólo con dos casos.

| Capa | Qué se prueba | Enfoque |
|---|---|---|
| Unit aplicación | `CompleteSprintStoryUseCase` | Fake manual del puerto |
| Unit HTTP | `CompleteSprintStoryHandler` | `httptest` con un mux real (`POST` + patrón) o `SetPathValue`; fake del puerto en un caso de uso real |
| Unit composición | Readiness `>= 10` y registro de la ruta | `api.NewHTTPHandlerWithDependencies` con fakes |
| Unit archivos | Secuencia de migraciones | `migration_files_test.go` |
| Integración | Repositorio, migración, concurrencia, HTTP real | Testcontainers `postgres:16-alpine` o `STORY_TEST_DATABASE_URL` |

### `tests/unit/story/application/complete_sprint_story_test.go` (paquete `application_test`)

- Éxito: una llamada al puerto con los tres IDs del comando; el resultado devuelto es idéntico al del fake (incluido `CompletedAt`).
- Tabla de errores del puerto (`ErrProjectNotFound`, `ErrSprintNotFound`, `ErrStoryNotFound`, `ErrSprintClosed`, `ErrStoryNotInSprint`, `ErrStoryAlreadyCompleted`, error genérico): `errors.Is` se conserva y el resultado es el valor cero.

### `tests/unit/story/transport/http/complete_sprint_story_handler_test.go` (paquete `transporthttp_test`)

Tabla con `name, path IDs, body, completerErr, wantStatus, wantError, wantFields, wantCalls`:

- `success` → 200; cuerpo con los tres IDs y `completed_at` igual al instante del fake en UTC (el fake devuelve un `time.Time` con zona no UTC para comprobar la normalización); `Content-Type: application/json`; 1 llamada.
- `uppercase UUIDs are canonicalized` → el fake recibe IDs en minúsculas.
- `body is ignored` (`{"x":1}` y `not json`) → 200, 1 llamada.
- `invalid project_id`, `invalid sprint_id`, `invalid story_id` → 422 con la clave correspondiente, 0 llamadas.
- `all route IDs invalid` → 422 con las tres claves, 0 llamadas.
- Un caso por centinela → status y código exactos de la tabla de mapeo, 1 llamada.
- `unexpected` (`errors.New("secret db detail")`) → 500 `internal_error`, cuerpo sin `"secret"`.
- `wrong method` (GET directo al handler) → 405.

### `tests/unit/cmd/api/main_test.go` (modificación)

- `fakeSprintStoryCompleter` (cuenta llamadas, devuelve una finalización fija).
- `TestCompletionRouteRequiresCleanVersionTenAndExplicitDependency`: casos `v9` (deshabilitado, y `readiness.Tasks == true`), `v10` y `future 12` (habilitado), `nil dependency`, `dirty v10`, `lookup error`, `missing migration table`. Verifica `readiness.Completion` y que `POST .../completion` responde 200 con 1 llamada sólo cuando corresponde; 404 y 0 llamadas en el resto.
- `TestCompletionRouteIsIndependentFromStories`: `HTTPDependencies{Completion: ...}` con `Stories == nil` → 200; `POST /projects/{id}/stories` sigue en 404.
- No se amplía `TestMigrationReadinessSelectsRoutesIndependently` (el caso `v9` del test nuevo ya comprueba que `Completion` no altera `Tasks`).

### `tests/integration/migrations/migration_files_test.go` (modificación)

- Agregar `"000010"` al arreglo canónico y cambiar `len(pairs) != 9` por `!= 10` con el mensaje `"canonical sequence must contain exactly ten migration pairs"`. Hoy exige exactamente 9 pares: fallaría (RED) al agregar `000010`.

### `tests/integration/story/postgres/completion_integration_test.go` (paquete `postgres_test`)

Helper `completionDatabase(t)`: `storyDatabase(t)` (aplica `000001`–`000008`) más `applyStoryMigration` de `000009_create_tasks.up.sql` y `000010_add_sprint_story_completion.up.sql`. No se modifica `storyDatabase`, para no alterar las suites existentes ni `TestAPIStartupRoutesFollowMigrationState`. Fixture con SQL directo y los helpers existentes del paquete (`insertProject`, `validStory`, `repo.Create`): proyecto A con Sprints S1 y S2 abiertos, historias H1 (asignada a S1 y S2) y H2 (asignada sólo a S1), H3 sin asignar; proyecto B con Sprint e historia propios. Helper `completedAt(t, pool, sprintID, storyID) *time.Time`.

- `TestCompletionSchemaRequiresMigrationTen`: `sprint_stories.completed_at` existe, es nulable y de tipo `timestamp with time zone`; una asociación creada antes de aplicar `000010` queda en `NULL`.
- `TestCompleteSprintStoryPersistsCompletionForThatSprintOnly`: completar H1 en S1 → resultado con IDs y `CompletedAt` entre `before`/`after` y igual al leído por SQL; H1 en S2 sigue `NULL`; `stories.status` y demás campos de H1 sin cambios (`ListByProject` antes/después); cantidad de filas de `sprint_stories` sin cambios. Luego completar H1 en S2 → éxito con su propio instante.
- `TestCompleteSprintStoryRejectsMissingOrForeignResourcesWithoutWrites` (tabla): proyecto inexistente, Sprint inexistente, Sprint de B, historia inexistente, historia de B → centinela correspondiente; ninguna fila con `completed_at` no nulo.
- `TestCompleteSprintStoryRejectsUnassignedStory`: H3 en S1 y H2 en S2 → `ErrStoryNotInSprint`; no se crea asociación.
- `TestCompleteSprintStoryRejectsClosedSprint`: S1 cerrado con H2 sin completar → `ErrSprintClosed`, sigue `NULL`; con H1 ya completada antes del cierre → `ErrSprintClosed` y `completed_at` original intacto.
- `TestCompleteSprintStoryNeverCountsTwice`: segundo llamado → `ErrStoryAlreadyCompleted` y `completed_at` igual al primero.
- `TestCompleteSprintStoryConcurrentRequestsRecordOnce`: 8 goroutines liberadas por un canal cerrado → exactamente 1 éxito, 7 `ErrStoryAlreadyCompleted`, y el `completed_at` persistido igual al del éxito.
- `TestSprintStoryCompletionMigrationCanBeReversedAndReapplied`: con una finalización registrada, `000010 down` elimina la columna y conserva las filas de `sprint_stories` (y una tarea creada sobre ellas sigue válida); `000010 up` la vuelve a agregar en `NULL`.
- `TestCompleteSprintStoryHTTPEndToEnd`: composición real `api.NewHTTPHandlerWithDependencies(projectpostgres..., api.NewProjectID, api.HTTPDependencies{Completion: &api.CompletionDependencies{Completer: storypostgres.NewPostgresStoryRepository(pool)}})`: primer `POST` → 200 con `completed_at` igual al persistido; segundo `POST` → 409 `story_already_completed`.

Escenarios de la spec cubiertos sin prueba dedicada adicional: "Modificar el estado de la historia no altera la finalización" (el `UPDATE` de `PostgresStoryRepository.Update` no toca `sprint_stories`; se puede agregar una aserción de una línea en `TestCompleteSprintStoryPersistsCompletionForThatSprintOnly` llamando a `repo.Update`), "Se permite completar con tareas pendientes" (el repositorio no lee `tasks`; la prueba de reversibilidad ya crea una tarea), "El instante queda disponible para métricas" (consulta SQL `COUNT(completed_at)` y `SUM(story_points)` dentro de la prueba de persistencia si `sdd-tasks` decide cubrirlo explícitamente).

## Matriz de amenazas

| Límite | Aplicabilidad |
|---|---|
| Rutas tipo documentación | N/A: el cambio no clasifica ni ejecuta archivos. |
| Selección de repositorio Git | N/A: no hay automatización de Git. |
| Estado de commit | N/A: no hay automatización de commits. |
| Estado de push | N/A: no hay automatización de push. |
| Comandos de PR | N/A: no hay automatización de PR. |

N/A: no hay límites de shell, subprocesos, automatización de VCS/PR, clasificación de archivos ejecutables ni integración de procesos. La ruta nueva usa el enrutamiento estándar de `net/http`; su seguridad de entrada (UUID validados, cuerpo no interpretado, SQL parametrizado, 500 sin detalles) está cubierta por la estrategia de pruebas.

## Migración y despliegue

- `000010` se aplica externamente, como las anteriores. Con el esquema en `<= 9` o sucio, la API arranca sin la ruta y registra `story completion unavailable until migration 000010 is clean ...`; las demás rutas conservan sus gates.
- `ALTER TABLE ... ADD COLUMN` nulable sin default toma un lock `ACCESS EXCLUSIVE` breve sobre `sprint_stories` y no reescribe la tabla.
- `000010 down` elimina la columna y **todos los registros de finalización**; antes de revertir en un entorno persistente se debe decidir si exportarlos (plan de reversión de la propuesta).
- Antes de abrir el PR, reconfirmar que `000010` sigue siendo el siguiente número libre.
- Tamaño estimado: producción ~190 líneas (migración 2, aplicación ~45, repositorio ~60, handler ~55, `api.go` ~20, `main.go` ~8), pruebas ~370, documentación ~25; total ~585, en línea con el pronóstico de ~590 de la propuesta. Con `single-pr` se requiere la excepción de tamaño que registra el orquestador.
- La nota de `openspec/config.yaml` sobre "dos migraciones numeradas 000003" ya está desactualizada (el histórico vive en `migrations/fixtures/`); no se corrige en este cambio (fuera de alcance).

## Preguntas abiertas

- [ ] Ninguna bloquea el diseño. La delta de spec ya fija `200`, el cuerpo ignorado, la acumulación de UUID inválidos y la precedencia `sprint_closed` antes de `story_not_in_sprint`; el diseño las sigue.
- [ ] A confirmar por el equipo (heredado de la propuesta, no bloqueante): divergencia aceptada entre `stories.status` y `sprint_stories.completed_at`, y `409` para Sprint cerrado e historia ya completada.
