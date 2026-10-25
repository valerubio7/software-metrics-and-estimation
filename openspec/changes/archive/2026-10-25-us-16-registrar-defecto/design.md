# Diseño: Registrar un defecto detectado durante el desarrollo (US-16)

Commit base: `df1ff7b Merge pull request #109` (US-15 integrada; `main` con migraciones hasta `000012`). Rama: `feat/us-16-registrar-defecto`. Issue objetivo: **#44 (US-16)**; fuera de alcance US-17 (#45), US-18 (#46), US-19 (#47), TUI-09 (#102).

## 1. Modelo de datos

### 1.1 Tabla `defects` (migración `000013_create_defects`)

```sql
CREATE TABLE defects (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL,
    story_id UUID NOT NULL,
    sprint_id UUID NOT NULL,
    description TEXT NOT NULL
        CONSTRAINT defects_description_length
            CHECK (char_length(description) BETWEEN 1 AND 500),
    severity TEXT NOT NULL
        CONSTRAINT defects_severity_check CHECK (severity IN ('alta', 'media', 'baja')),
    status TEXT NOT NULL
        CONSTRAINT defects_status_check CHECK (status IN ('detectado', 'en_correccion', 'resuelto')),
    seq BIGINT GENERATED ALWAYS AS IDENTITY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT defects_story_project_fkey FOREIGN KEY (story_id, project_id)
        REFERENCES stories (id, project_id) ON DELETE RESTRICT,
    CONSTRAINT defects_sprint_project_fkey FOREIGN KEY (sprint_id, project_id)
        REFERENCES sprints (id, project_id) ON DELETE RESTRICT
);

CREATE INDEX defects_project_story_idx ON defects (project_id, story_id, seq);
CREATE INDEX defects_project_sprint_idx ON defects (project_id, sprint_id, seq);
```

Decisiones de DDL y su justificación:

| Elemento | Decisión | Justificación |
|---|---|---|
| `sprint_id` | Columna única que almacena el **Sprint de detección** | Terminología del issue (`gh issue view 44`: "Sprint de detección"). US-17 agregará `resolution_sprint_id` con su propia migración; el par queda autoexplicativo (`defects.sprint_id` + `defects.resolution_sprint_id`). |
| `CHECK` de estado | Catálogo completo (`detectado`/`en_correccion`/`resuelto`) desde `000013` | El `CHECK` valida **valores**, no **transiciones**: incluye `resuelto` para que US-17 no reemplace la restricción en `000014`; el caso de uso del alta aplica solo el subconjunto inicial. La asimetría queda exportada en la spec (Requirement "Validar el estado..."). |
| `char_length` vs `octet_length` | `char_length(... BETWEEN 1 AND 500)` | Cuenta caracteres, igual que `utf8.RuneCountInString` en Go. El NUL se rechaza en la aplicación (`22021` en PostgreSQL `text`, patrón ya verificado en US-15); el `CHECK` no lo rechaza porque `char_length` lo cuenta como un carácter más; el rechazo vive en la capa de aplicación, patrón ya verificado en US-15. |
| `CHECK` de severidad/estado con literales minúsculas | Coincidencia exacta | El alta no recorta ni normaliza catálogos: solo valores exactos llegan a la base (decisión 1 y 2 del usuario). |
| FKs compuestas con `ON DELETE RESTRICT` | Igual que `sprint_stories` y `tasks` | Un proyecto con defectos no se borra por accidente (los `projects` ya usan `RESTRICT` en cascada de referencias existentes). `MATCH SIMPLE` implícito: no hay columnas anulables en las FKs, así que tampoco aplica. |
| `UNIQUE` nuevas | Ninguna | `stories(id, project_id)` y `sprints(id, project_id)` ya existen desde la migración `000007`; el `down` de `000013` queda trivial. |
| Índices | `(project_id, story_id, seq)` y `(project_id, sprint_id, seq)` | Preparan US-18 (lista de defectos por proyecto/historia y por Sprint de detección) y el conteo de defectos de US-19. El `PK(id)` cubre los accesos por identificador. |
| `seq` | `GENERATED ALWAYS AS IDENTITY` | Patrón de `tasks` (000009) y `sprints` (000011): orden estable de inserción para la lista de US-18. |
| `created_at` | `TIMESTAMPTZ NOT NULL DEFAULT now()`, no expuesto en el contrato 201 | Igual que `effort_records`: la respuesta 201 no expone marcas internas; concreto para el par, si alguien quiere auditoría la base la tiene. |

### 1.2 `down`

```sql
DROP TABLE IF EXISTS defects;
```

Idempotente y sin efectos colaterales: no se agregan restricciones sobre tablas existentes.

## 2. Capas del módulo `internal/defect`

### 2.1 `domain/defect.go`

- `type Defect struct { ID, ProjectID, StoryID, SprintID, Description, Severity, Status string }` — sin punteros: todos los campos del alta son obligatorios (a diferencia de `EffortRecord.TaskID *string`).
- `type ValidationError struct { Fields map[string]string }` + método `Error()` (`"defect validation failed"`), espejo de `effort/domain`.
- `NewDefectParams { ID, ProjectID, StoryID, SprintID, Description, Severity, Status string }`.
- `NewDefect(params NewDefectParams) (Defect, error)`:
  - `description := strings.TrimSpace(params.Description)`; errores: vacía/solo espacios → `"is required"`; `strings.ContainsRune(description, 0)` → `"must not contain NUL characters"`; `utf8.RuneCountInString(description) > 500` → `"must be at most 500 characters"`.
  - `validateSeverity`: `severity == ""` → `"is required"`; fuera de `{"alta","media","baja"}` → `"must be one of alta, media or baja"`. Sin trim/case-fold: coincidencia exacta (espejo del issue y del `CHECK`).
  - `validateStatus`: `status == ""` → `"is required"`; `status == "resuelto"` → `"must be detectado or en_correccion when registering a defect"`; fuera de `{"detectado","en_correccion","resuelto"}` → `"must be one of detectado, en_correccion or resuelto"`. El mensaje específico de `resuelto` orienta hacia US-17 sin acoplar módulos.
  - Acumula todos los campos con error en el `map` y devuelve un único `*ValidationError` (decisión "reportar todos juntos").
  - No valida IDs: la capa de aplicación los normaliza antes de llamar al dominio (patrón de `effort/application`).
- Constantes exportadas `MaxDescriptionLength = 500` y los catálogos (`Severities`, `Statuses`, `InitialStatuses`) para su uso en mensajes y tests.

### 2.2 `application/register_defect.go`

- `RegisterDefectCommand { ProjectID, StoryID, SprintID, Description, Severity, Status string }` — sin punteros ni reloj (US-16 no tiene fecha de trabajo).
- Sentinels: `ErrProjectNotFound`, `ErrStoryNotFound`, `ErrSprintNotFound` (`errors.New`), para `errors.Is` en el transporte.
- `DefectRepository interface { Create(ctx context.Context, defect domain.Defect) error }` — el método encapsula la verificación transaccional (patrón `effort`).
- `IDGenerator func() string` (impl.: `api.NewProjectID`).
- `RegisterDefectUseCase.Execute`:
  1. `storyID := normalizeUUID(fields, "story_id", command.StoryID, true)`; ídem `sprint_id`. `normalizeUUID` (inlined, misma semántica que `effort`): vacío → `"is required"`, inválido → `"must be a valid UUID"`, válido → `uuid.Parse(value).String()`.
  2. Ejecutar validación de dominio activamente: `domain.NewDefect(...)` y fusionar su `*ValidationError` en `fields` (patrón `mergeDomainFields`).
  3. Con errores: `return domain.Defect{}, &domain.ValidationError{Fields: fields}` (todas los errores acumulados en una respuesta).
  4. `record.ID = generateID()` recién tras la validación de campos (no desperdicia UUID ante errores).
  5. `repository.Create(ctx, record)` — el error de referencia viaja tal cual (`errors.Is` del transporte).
- Sin `Clock` ni `WorkDate`: no hay fecha comparable en el alta del defecto (diferencia con `effort`).

### 2.3 `infrastructure/postgres/repository.go`

- `PostgresDefectRepository{pool *pgxpool.Pool}` con `Create` transaccional y `defer rollback`:
  1. `SELECT EXISTS (SELECT 1 FROM projects WHERE id = $1)` → si falta, `ErrProjectNotFound`.
  2. `SELECT 1 FROM stories WHERE id = $1 AND project_id = $2 FOR KEY SHARE` → si falta, `ErrStoryNotFound`.
  3. `SELECT 1 FROM sprints WHERE id = $1 AND project_id = $2 FOR KEY SHARE` → si falta, `ErrSprintNotFound`.
  4. `INSERT INTO defects (id, project_id, story_id, sprint_id, description, severity, status) VALUES (...)`.
  5. `tx.Commit` — el `defer rollback` cubre todo camino que no llegue al `Commit`; el `FOR KEY SHARE` de las consultas 2 y 3 es el mismo lock que la FK toma en el INSERT, así que una eliminación concurrente no invalida la verificación (patrón y justificación copiados de `effort`).
- `requireRow` helper extraído igual que en `effort` (consulta de una fila y mapeo a error de dominio).
- `MapConstraintError`:
  - `defects_story_project_fkey` → `ErrStoryNotFound`.
  - `defects_sprint_project_fkey` → `ErrSprintNotFound`.
  - `(23503)` con otra restricción u otro código → error sin transformar (el `CHECK` violado por otra vía termina en `500`, caso que solo una ruta ajena al caso de uso puede provocar).
- `var _ application.DefectRepository = (*PostgresDefectRepository)(nil)` (compilación de contrato).

### 2.4 `transport/http/handler.go`

- `NewRegisterDefectHandler(useCase) http.Handler`, `ServeHTTP` con:
  - `method != POST` → `405 method_not_allowed` ("only POST is supported").
  - `decodeDefectObject` (clon de `decodeEffortObject`): primer `json.RawMessage` debe iniciar con `{`; segundo decode debe terminar en `io.EOF`; re-decode estricto con `DisallowUnknownFields` sobre el raw → cualquier fallo → `400 invalid_request` con mensaje genérico.
  - `project_id` de la ruta: `uuid.Parse` → si falla, `422 validation_failed` con `fields.project_id = "must be a valid UUID"` (convención de las rutas anidadas existentes).
  - `useCase.Execute(cmd)` y mapeo de errores:
    - `*domain.ValidationError` → `422 validation_failed` + `fields`.
    - `Err{Story,Sprint,Project}NotFound` → `404 story_not_found`, `404 sprint_not_found`, `404 project_not_found` con su mensaje.
    - default → `500 internal_error` ("an unexpected error occurred").
- Respuesta `defectResponse`: `{id, project_id, story_id, sprint_id, description, severity, status}` (JSON en inglés, SIN `created_at`; alineación con el 201 de esfuerzo).
- `writeJSON` no setea cabeceras raras: `application/json` + status + encode una línea (convención compartida).

## 3. Cableado y habilitación

### 3.1 `internal/api/api.go`

- `EffortDependencies` ya existe; nuevo:

```go
// DefectDependencies enables defect registration after migration 000013 is clean.
type DefectDependencies struct {
    Repository defectapplication.DefectRepository
    GenerateID defectapplication.IDGenerator
}
```

- `HTTPDependencies.Defect *DefectDependencies // nil: the defect route is not registered (the mux answers 404)`.
- `MigrationReadiness.Defect bool` con el comentario `// Defect is true for a clean schema at version 13 or later (defects from migration 000013).`
- `ResolveMigrationReadiness`: `readiness.Defect = version >= 13` (independiente de `Effort`).
- `NewHTTPHandlerWithDependencies`: bloque nuevo tras el de `Effort`:

```go
if dependencies.Defect != nil {
    registerDefectUseCase := defectapplication.NewRegisterDefectUseCase(dependencies.Defect.Repository, dependencies.Defect.GenerateID)
    mux.Handle("POST /projects/{project_id}/defects", defecthttp.NewRegisterDefectHandler(registerDefectUseCase))
}
```

### 3.2 `cmd/api/main.go`

- Recalcular `readiness := api.ResolveMigrationReadiness(...)` (el cálculo se hace una vez por arranque; el bloque actual ya sigue ese patrón).
- Bloque nuevo transitivamente tras el de esfuerzo:

```go
if readiness := api.ResolveMigrationReadiness(version, dirty, migrationErr); readiness.Defect {
    dependencies.Defect = &api.DefectDependencies{
        Repository: defectpostgres.NewPostgresDefectRepository(pool),
        GenerateID: api.NewProjectID,
    }
    log.Printf("defect registration available (schema version=%d)", version)
} else {
    log.Printf("defect registration unavailable until migration 000013 is clean (version=%d, dirty=%t, lookup error=%v)", version, dirty, migrationErr)
}
```

- Actualización del comentario de bloque de versiones: agregar "defect registration needs schema version 13 (migration 000013, defects)".

## 4. Estrategia de pruebas (TDD estricto, serie RED/GREEN/TRIANGULATE/REFACTOR por UT)

### 4.1 Matriz de coberturas por capa

| UT | Paquete de pruebas | Alcance |
|---|---|---|
| 1 | `tests/integration/migrations` + `tests/integration/defect/postgres` (migration) | 13 pares canónicos; `000013` up/down/reapply/repeat-down/up; CHECKs de severidad/estado/descripción; FKs compuestas; preservación de tablas existentes |
| 2 | `tests/unit/defect/domain` | Descripción (vacía, espacios, 500/501 multibyte, NUL), severidad (catálogo/rechazos), estado (subconjunto inicial/resuelto/fuera), combinaciones acumuladas |
| 3 | `tests/unit/defect/application` | UUID obligatory/opcionales, normalización canónica, acumulación de errores, IDs generados solo al éxito, sentinelas que pasan sin wrap, fake de repositorio |
| 4 | `tests/unit/defect/infrastructure/postgres` + `tests/integration/defect/postgres` (repository) | `MapConstraintError` (3 FKs + resto); integración real: persistencia, precedencia proyecto→historia→sprint, backlog story, sprint cerrado, duplicados, FK engañadas concurrentes indirectamente (consulta+INSERT en transacción), 500 no filtrado |
| 5 | `tests/unit/defect/transport/http` | `httptest`: 201 forma del body, 405, 400 (cuerpos malformados: vacío, roto, no objeto, contenido extra, claves desconocidas, tipos mal), 422 acumulados por campo, 404×3 con precedencia, 500 genérico sin detalle |
| 6 | `tests/unit/cmd/api` + `tests/integration/defect/postgres` (http e2e) | Readiness `>= 13` (registro/omisión de la ruta), composición `main.go` por consiguiente, e2e real contra Postgres: happy path, 404 precedencia real, duplicados reales, `GET → 405` |

### 4.2 Convenciones de pruebas

- Tablas de casos (`map[string]struct{...}`) con nombres de caso en inglés snake_case (identificadores en inglés; comentarios理由 en español).
- Fakes en memoria: `stubRepository{err error}` para el caso de uso; no se simula el pool.
- Integración: helpers de `tests/integration/testpostgres` (`postgres:16-alpine`, migraciones por archivos, `waitForDatabaseReady`); seeds con UUID fijos (patrón de `effort`).
- El conjunto queda verde sin Docker para unitarios (`go test ./tests/unit/...`), como en US-15.

## 5. Áreas de riesgo técnico y mitigación

| Riesgo | Mitigación |
|---|---|
| Divergencia `CHECK` full-catálogo vs. subconjunto aplicado | El mensaje del dominio para `resuelto` es explícito (`must be detectado or en_correccion when registering a defect`) y la spec lo distingue; la US-17 no toca la restricción |
| Precedencia de `404` inconsistente entre repos y HTTP | El orden de verificación en el repositorio fija el orden; el test del caso 4 usa UUIDs cuyo resultado cruza varios tipos de referencias para triangulación; el test HTTP 5 replica la tabla |
| FK violada concurrente | `FOR KEY SHARE` + verificación transaccional; `MapConstraintError` como red de seguridad; test de integración con eliminación en otra conexión entre verificación y commit (sleep corto + goroutine, patrón de US-15) |
| Doble solicitud idéntica (duplicados) | No hay constraint que lo impida; la prueba afirma 2 filas con IDs distintos |
| Regresión de gates | `ResolveMigrationReadiness` versionado a `13` mantiene la tabla expandida; test `>= 13` y otros casos de versiones existentes |

## 6. Archivos de código y pruebas (plan completo)

```
internal/project/infrastructure/postgres/migrations/000013_create_defects.up.sql   (nuevo)
internal/project/infrastructure/postgres/migrations/000013_create_defects.down.sql (nuevo)
internal/defect/domain/defect.go                                                    (nuevo)
internal/defect/application/register_defect.go                                      (nuevo)
internal/defect/infrastructure/postgres/repository.go                               (nuevo)
internal/defect/transport/http/handler.go                                           (nuevo)
internal/api/api.go                                                                 (modificado: DefectDependencies + readiness == 13 + handler)
cmd/api/main.go                                                                     (modificado: bloque de composición + comentario de versiones)
tests/integration/migrations/migration_files_test.go                                (13 pares)
tests/integration/defect/postgres/migration_integration_test.go                     (nuevo)
tests/integration/defect/postgres/repository_integration_test.go                    (nuevo)
tests/integration/defect/postgres/http_integration_test.go                          (nuevo)
tests/unit/defect/domain/defect_test.go                                             (nuevo)
tests/unit/defect/application/register_defect_test.go                               (nuevo)
tests/unit/defect/infrastructure/postgres/constraint_error_test.go                  (nuevo)
tests/unit/defect/transport/http/handler_test.go                                    (nuevo)
tests/unit/cmd/api/defect_route_test.go                                             (nuevo)
README.md                                                                           (modificado: sección "API de defectos")
openspec/config.yaml                                                                (modificado: nota "Implementada: US-16")
```

## 7. Decisiones de diseño ya cerradas (sin recursividad)

1. **Sin Clock/WorkDate en US-16**: el defecto no tiene fecha de trabajo; si US-19 la necesitara, sería una migración `ALTER TABLE` posterior (patrón `000003`/`000011`). No se adelanta.
2. **Sin `resolution_sprint_id` en `000013`**: la columna y sus validaciones (más la transición de estado) llegan con US-17 y su `000014`. Adelantarla sería implementar almacenamiento de US-17 sin el caso de uso que la gobierna.
3. **JSON body con `story_id` y `sprint_id` en el cuerpo**: la ruta solo lleva el proyecto, como `US-3` registra integrantes. Alternativa `POST /projects/{id}/stories/{id}/defects` descartada: colisiona semánticamente con los defectos "de una historia" y la TUI-09 elige historia y Sprint desde listas del proyecto.
4. **`405` solo en el metodo distinto de `POST`**: el mux con patrón `POST /...` ya maneja `405` para otros métodos automáticamente en rutas de un solo método (compensación: el handler igual lo re-verifica, como `effort`).
5. **Nombres de restricciones**: `defects_*` (prefijo de tabla, igual que `effort_records_*`).

## 8. Lo que NO cambia

- `internal/story`, `internal/task`, `internal/effort`, `internal/sprint`, `internal/projectmember`, `internal/project` — sin modificaciones de código (solo lectura de sus tablas por FKs nuevas).
- Toda ruta existente y su gating (`readiness.*` versiones 2–12) — sin cambios; solo se agrega `Defect bool` con `>= 13`.
- `tests/integration/effort/...` — sin cambios (indirectamente re-ejecutables para detectar regressiones).
- `openspec/specs/{historia,sprint,tarea,tui,project,esfuerzo}` — sin deltas; la capacidad `defecto` es nueva y las menciones genéricas sobre defectos en otros specs no cambian contratos.
