# Diseño: Asignar historias a Sprint (US-09)

> Diseño retroactivo: documenta las decisiones efectivamente implementadas y fusionadas (PR #78, commit `f544916`), no un diseño previo a la implementación.

## Antecedentes: de la cadena de PRs abandonada a la consolidación

La planificación original de US-09 (issue #37) preveía una cadena de PRs hijos — `feat/us-09-assignment-01-core` hasta `06-progress`, coordinados por un tracker PR #71 y materializados como PRs #72 a #77 — para entregar el contrato de aplicación, las migraciones y el repositorio transaccional, las pruebas de integración PostgreSQL, el endpoint HTTP, el diseño/specs/tareas SDD y el cierre ODD en cortes revisables sucesivos, cada uno limitado a 400 líneas modificadas y apuntando al PR inmediatamente anterior. Esa cadena se construyó contra ramas intermedias y **nunca llegó a fusionarse en `main`**; sus PRs quedaron cerrados/obsoletos.

La integración real se rehízo directamente sobre el esquema canónico verdadero de `main` en un esfuerzo de consolidación único: rama `fix/us09-consolidation`, PR #78, que portó el comportamiento ya validado en `feat/us-09-assignment-05-sdd` (commit `e4ddb66`) y lo adaptó a la numeración de migraciones real de `main` en el momento de la fusión. El resultado fue un único commit de comportamiento (`f544916`, 23 archivos, 1158 inserciones/45 eliminaciones) seguido de un commit de cierre ODD (`3ad5433`), fusionados mediante el merge commit `63a6527`. Este documento de diseño sustituye el contenido de seguimiento que vivía en el ahora eliminado `chain-context.md`, que describía únicamente la cadena abandonada.

## Decisiones

### Decisión: FK compuesta `(id, project_id)` como autoridad de "mismo proyecto"

**Elección**: `000007_create_sprint_stories.up.sql` agrega `UNIQUE(id, project_id)` a `stories` y `sprints`, y define `sprint_stories` con dos FKs compuestas: `(sprint_id, project_id) → sprints(id, project_id)` y `(story_id, project_id) → stories(id, project_id)`, ambas `ON DELETE RESTRICT`.

**Alternativas consideradas**: validar "mismo proyecto" únicamente en la capa de aplicación mediante consultas previas; usar FKs simples sin la columna `project_id` denormalizada en `sprint_stories`.

**Justificación**: una FK compuesta sobre `project_id` compartido impide, a nivel de base de datos, insertar un vínculo cuyo Sprint y cuya historia pertenezcan a proyectos distintos — la base de datos rechazaría la fila incluso si la comprobación de aplicación tuviera un error. Esto es defensa en profundidad junto con la comprobación explícita en `AssignStoriesForProject` (`internal/story/infrastructure/postgres/repository.go:54` y `:84`).

### Decisión: Bloqueo de fila `SELECT ... FOR UPDATE` sobre el Sprint

**Elección**: la transacción de asignación bloquea la fila del Sprint con `FOR UPDATE` antes de leer `is_closed` (`internal/story/infrastructure/postgres/repository.go:47`).

**Alternativas consideradas**: verificar `is_closed` sin bloqueo y aceptar una ventana de carrera con el cierre; usar un nivel de aislamiento serializable sin bloqueo explícito.

**Justificación**: serializa asignaciones y cierres concurrentes del mismo Sprint: una transacción de cierre que confirma primero deja a la asignación en espera observando `is_closed=true` tras obtener el bloqueo, por lo que rechaza íntegramente el lote. Verificado por `TestAssignStoriesWaitsForConcurrentSprintClose`.

### Decisión: Canonicalización de UUIDs antes de deduplicar

**Elección**: el handler aplica `uuid.Parse(id).String()` sobre cada `story_id` del cuerpo (`internal/story/transport/http/assign_stories_handler.go:43-50`) **antes** de que el caso de uso compruebe duplicados.

**Alternativas consideradas**: deduplicar sobre el texto recibido tal cual, sin canonicalizar; fusionar silenciosamente alias del mismo UUID con distinta capitalización.

**Justificación**: dos representaciones del mismo UUID con capitalización distinta deben tratarse como el mismo ID. Canonicalizar antes de la comprobación de duplicados hace que ese caso se rechace como `422 validation_failed` (duplicado), en vez de insertarse dos veces o de colapsar sin que el cliente lo sepa. Verificado por `TestCON01UUIDAliasesAreDuplicate` y `TestAssignStoriesHandlerCanonicalizesWithoutChangingSelectionOrder`.

### Decisión: Manejo de `23505` como red de seguridad, no como mecanismo primario

**Elección**: antes de insertar, el repositorio comprueba `EXISTS` sobre `sprint_stories` para detectar asociaciones ya persistidas; además, cada `INSERT` individual captura `pgconn.PgError` con `Code == "23505"` y `ConstraintName == "sprint_stories_pkey"` y lo traduce a `ErrStoryAlreadyAssigned` (`internal/story/infrastructure/postgres/repository.go:88-104`).

**Alternativas consideradas**: confiar únicamente en la comprobación `EXISTS` previa sin manejar la violación de constraint; confiar únicamente en la constraint sin la comprobación previa.

**Justificación**: la comprobación `EXISTS` cubre el caso común sin carrera; el manejo de `23505` cubre la ventana entre esa comprobación y el `INSERT` cuando dos solicitudes concurrentes compiten por el mismo vínculo — solo una gana, la otra recibe `ErrStoryAlreadyAssigned` y revierte toda su transacción. Verificado por `TestAssignStoriesConcurrentDuplicateHasSingleWinner` y `TestAssignStoriesRollsBackWhenAnInsertFails`.

### Decisión: Habilitación de la ruta por versión de esquema y dependencia explícita

**Elección**: `api.ResolveMigrationReadiness` fija `Assignment: version >= 8` (`internal/api/api.go:95`); `cmd/api/main.go` solo compone el asignador (`dependencies.Stories.Assigner`) cuando esa condición se cumple junto con la versión limpia.

**Alternativas consideradas**: habilitar la ruta de asignación en el mismo umbral que la disponibilidad general de historias (`version >= 4`); no comprobar versión y dejar que la ruta falle en tiempo de ejecución si el esquema no está listo.

**Justificación**: la ruta de asignación depende de tablas/columnas que solo existen desde la versión 8 (`sprint_stories`, `sprints.is_closed`); registrarla antes expondría un endpoint que fallaría en cada solicitud. Seguir el patrón existente de gating por versión (ya usado para historias `>=4` y Sprints `>=5`) mantiene la API arrancando igual con esquemas parciales, sin exponer rutas rotas.

**Fragilidad señalada**: `dependencies.Stories` se compone de forma no nula ya desde `version >= 2` (`cmd/api/main.go:53`, la rama más baja del switch; `>=3` y `>=4` en las líneas 49 y 45 solo amplían qué campos trae el struct), mientras que `readiness.Assignment` (gate `>=8`, en `internal/api/api.go:95`) es una comprobación de versión completamente independiente, en otro archivo. Hoy no es un defecto porque `>=8` implica `>=2` con margen de sobra, pero cualquier cambio futuro a uno de los dos umbrales sin revisar el otro podría habilitar asignación sin la dependencia de historias, o viceversa. Se deja documentado para que una futura migración revise ambos puntos juntos.

## Alternativas consideradas (generales)

- **Exponer un endpoint de desasignación en este mismo cambio**: descartado; fuera del alcance aprobado por la issue #37, que solo pide asignar sin retirar del backlog.
- **Modificar el estado de las historias al asignarlas**: descartado explícitamente; el requisito exige que la asociación no modifique historias, su estado, estimaciones ni pertenencia/orden en el Product Backlog.
- **Permitir éxito parcial del lote**: descartado; el contrato exige todo-o-nada, con rollback completo ante cualquier fallo de inserción.

## Datos y persistencia

```sql
-- 000007_create_sprint_stories.up.sql
ALTER TABLE stories ADD CONSTRAINT stories_id_project_id_key UNIQUE (id, project_id);
ALTER TABLE sprints ADD CONSTRAINT sprints_id_project_id_key UNIQUE (id, project_id);

CREATE TABLE sprint_stories (
    sprint_id UUID NOT NULL,
    story_id UUID NOT NULL,
    project_id UUID NOT NULL,
    CONSTRAINT sprint_stories_pkey PRIMARY KEY (sprint_id, story_id),
    CONSTRAINT sprint_stories_sprint_project_fkey FOREIGN KEY (sprint_id, project_id)
        REFERENCES sprints (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT sprint_stories_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT
);
```

```sql
-- 000008_add_sprint_closed.up.sql
ALTER TABLE sprints ADD COLUMN is_closed BOOLEAN NOT NULL DEFAULT false;
```

Ambas migraciones son append-only sobre el esquema canónico existente; no modifican `000001`–`000006`. El `down` de `000008` elimina la columna `is_closed`; el `down` de `000007` elimina `sprint_stories` y las dos constraints `UNIQUE` agregadas. Revertir ambas desde v8 con `down 2` elimina asociaciones y cierre, no las tablas base de historias/Sprints/integrantes de v6; no se afirma seguridad de rollback productivo.

## Archivos/áreas previstos

| Archivo | Propósito |
|---|---|
| `internal/story/application/assign_stories.go` | Comando, errores tipados, interfaces `StorySprintAssigner`/`ProjectScopedStorySprintAssigner` y caso de uso que valida selección no vacía/sin duplicados antes de delegar en el repositorio. |
| `internal/story/transport/http/assign_stories_handler.go` | Decodificación estricta, canonicalización de UUIDs de ruta y de cada `story_id`, invocación del caso de uso y mapeo de errores a `201/400/404/409/422/500`. |
| `internal/story/infrastructure/postgres/repository.go` | Métodos `AssignStories` (deriva el proyecto desde el Sprint) y `AssignStoriesForProject` (transacción con bloqueo, validación completa e inserción atómica del lote). |
| `internal/project/infrastructure/postgres/migrations/000007_create_sprint_stories.{up,down}.sql` | Tabla de asociación Sprint-historia con FKs compuestas por proyecto. |
| `internal/project/infrastructure/postgres/migrations/000008_add_sprint_closed.{up,down}.sql` | Bandera de cierre de Sprint. |
| `internal/api/api.go` | `ResolveMigrationReadiness` con `Assignment: version >= 8`. |
| `cmd/api/main.go` | Composición del asignador (`dependencies.Stories.Assigner`) condicionada a la disponibilidad resuelta. |
| `tests/unit/story/application/assign_stories_test.go` | Validación de selección vacía/duplicada, delegación al repositorio con alcance de proyecto. |
| `tests/unit/story/transport/http/assign_stories_test.go` | Contrato HTTP, canonicalización, mapeo de errores, preservación de orden de entrada. |
| `tests/integration/story/postgres/assignment_integration_test.go` | Comportamiento transaccional real contra PostgreSQL: atomicidad, bloqueo, condiciones de carrera, preservación del backlog. |
| `tests/integration/migrations/` | Convergencia de migraciones canónicas hasta v8. |
| `README.md` | Documentación del endpoint, contrato de error y requisito de migración ≥8. |

## Interfaces y contratos

### HTTP

```http
POST /projects/{project_id}/sprints/{sprint_id}/stories
Content-Type: application/json

{"story_ids":["<uuid>","<uuid>"]}
```

Respuesta exitosa `201 Created`:

```json
{"sprint_id":"<uuid canónico>","story_ids":["<uuid canónico>","<uuid canónico>"]}
```

| Caso | HTTP | Contrato de error |
|---|---:|---|
| JSON malformado, múltiples valores o campos desconocidos | 400 | `{"error":"invalid_request", ...}` |
| `project_id`/`sprint_id`/`story_ids` con UUID inválido, selección vacía o con duplicados | 422 | `{"error":"validation_failed", "fields": {...}}` |
| Sprint o historia inexistente | 404 | `{"error":"resource_not_found", ...}` |
| Sprint cerrado, incompatibilidad de proyecto, o historia ya asignada | 409 | `{"error":"assignment_conflict", ...}` |
| Fallo inesperado de persistencia | 500 | `{"error":"internal_error", ...}` sin detalle interno (verificado por prueba) |

El orden de `story_ids` en la respuesta preserva el orden de entrada, con cada ID normalizado a su forma canónica de UUID.

### Dominio y aplicación

```go
type AssignStoriesCommand struct {
    ProjectID string
    SprintID  string
    StoryIDs  []string
}

type ProjectScopedStorySprintAssigner interface {
    AssignStoriesForProject(ctx context.Context, projectID string, sprintID string, storyIDs []string) error
}
```

El caso de uso rechaza selección vacía (`ErrEmptyStorySelection`) y duplicados (`ErrDuplicateStoryID`) antes de llamar al repositorio; si el asignador inyectado no implementa el alcance de proyecto, devuelve `ErrProjectScopedAssignerRequired` en vez de persistir sin esa garantía.

### PostgreSQL

La transacción de `AssignStoriesForProject` ejecuta, en orden: `SELECT project_id, is_closed FROM sprints WHERE id = $1 FOR UPDATE` → comprobación de proyecto/cierre → `SELECT id, project_id FROM stories WHERE id = ANY($1) FOR KEY SHARE` → comprobación de existencia/pertenencia de cada historia → `SELECT EXISTS(...)` sobre `sprint_stories` → un `INSERT` por historia con manejo de `23505` → `COMMIT` solo si todo el lote tuvo éxito, `ROLLBACK` diferido en cualquier otro caso.

## Estrategia de pruebas

TDD estricto (`strict_tdd: true`, `go test ./...`). Pruebas reales ya existentes que cubren el contrato:

| Prueba | Capa | Qué verifica |
|---|---|---|
| `TestAssignStoriesRejectsCrossProjectBatchWithoutPartialWrites` | Integración PostgreSQL | Un lote con una historia de otro proyecto se rechaza íntegramente, sin escrituras parciales. |
| `TestAssignStoriesRejectsMismatchedRouteProjectWithoutWrites` | Integración PostgreSQL | Un Sprint existente con `project_id` de ruta distinto responde `409` sin agregar vínculos. |
| `TestAssignStoriesConcurrentDuplicateHasSingleWinner` | Integración PostgreSQL | Dos solicitudes concurrentes por el mismo vínculo: una gana, la otra rechaza sin duplicados. |
| `TestAssignStoriesWaitsForConcurrentSprintClose` | Integración PostgreSQL | Un cierre concurrente que confirma primero bloquea la asignación en espera. |
| `TestAssignStoriesRollsBackWhenAnInsertFails` | Integración PostgreSQL | Un fallo en cualquier inserción del lote revierte todas las inserciones previas de esa transacción. |
| `TestAssignStoriesAtomicallyPreservesBacklogAndStoryFields` | Integración PostgreSQL | Tras asignar, el Product Backlog devuelve las mismas historias con todos sus valores intactos. |
| `TestCON01UUIDAliasesAreDuplicate` | Unitaria (HTTP) | Alias de UUID con distinta capitalización se detectan como duplicado tras canonicalizar. |
| `TestAssignStoriesHandlerCanonicalizesWithoutChangingSelectionOrder` | Unitaria (HTTP) | La canonicalización no altera el orden de `story_ids` en la respuesta. |

## Migración y despliegue

Igual que Sprint/project-members: las migraciones son externas, aplicadas con `golang-migrate` antes de desplegar el binario; la API no las ejecuta automáticamente. Desplegar `000007` y `000008` deja la ruta de asignación cerrada hasta que `schema_migrations` reporte versión limpia ≥8; un estado `dirty` o un error de lectura no habilita la ruta pero preserva las rutas previas (proyectos, historias, Sprints, integrantes). No se requiere backfill: ambas tablas/columnas son nuevas. No ejecutar `down` como mecanismo de rollback de aplicación sin decisión explícita, porque elimina asociaciones y cierre persistidos.
