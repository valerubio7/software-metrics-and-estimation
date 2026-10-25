# Progreso de apply: us-16-registrar-defecto

Rama: `feat/us-16-registrar-defecto`. Modo: TDD estricto (RED → GREEN → TRIANGULATE → REFACTOR). Una sección por Unidad de Trabajo, con comandos, resultados observados y hash del commit correspondiente.

## UT 1 — `feat(db): add defect records migration`

### Alcance

- `internal/project/infrastructure/postgres/migrations/000013_create_defects.{up,down}.sql` (nuevo).
- `tests/integration/defect/postgres/migration_integration_test.go` (nuevo; reutiliza helpers del mismo paquete).
- `tests/integration/migrations/migration_files_test.go` (12 → 13 pares).

### RED observado

| Comando | Resultado |
|---|---|
| `go test ./tests/integration/defect/... -count=1` | FAIL en todas las pruebas: `migration 000013 up: found 0 files, want exactly 1` (GitHub Files sin migración por escribir) |

Duración cuando el contenedor arranca: ~19 s (una base por prueba; el paquete usa una base por test, como `effort`).

### GREEN observado

| Comando | Resultado |
|---|---|
| `go test ./tests/integration/defect/... -count=1` | ok (18,9 s): 7 pruebas de primer nivel + subpruebas de CHECK y FK |
| `go test ./tests/integration/migrations/... -count=1` | ok (0,003 s): 13 pares canónicos |
| `go build ./...` y `go vet ./...` | exit 0, sin salida |

Detalles de RED→GREEN observados en el ciclo (iteración corta de helpers):

1. Primer intento con expectativas de `CHECK` en forma textual (`CHECK (((severity = ANY ...)))`) falló contra la salida real del catálogo tras la canonicalización (`CHECK((severity=ANY(ARRAY['alta','media','baja'])))`); se ajustaron las expectativas a la forma canónica observada, sin cambiar el SQL.
2. `nonexistent_story` usaba por error la historia de backlog sembrada (existe en el proyecto A, por lo que el insert era válido). Se corrigió a `unknownStoryID`.
3. Los subtests de FK compartían un ID fijo y el segundo insert chocaba con la PK (`23505`); cada subtest ahora usa su propio ID (`%012d` con el índice).

### Cobertura de la UT

- `TestDefectMigrationCreatesSchema`: columnas (`uuid/NO` para todos los identificadores, `text/NO` para severidad/estado/descripción, `bigint/NO` para `seq`), FK compuestas exactas (`/s/r`), `CHECK`s canónicos e índices secundarios exactos.
- `TestDefectMigrationUpOverVersion12KeepsExistingData`: `000013` up sobre 12 preserva proyectos/2, historias/3, sprints/2 con datos intactos.
- `TestDefectMigrationDownRemovesSchemaAndKeepsData`: `down` con filas de defectos existentes deja la base en versión 12 sin residuos.
- `TestDefectMigrationDownIsIdempotent`: dos `down` en serie terminan sin errores.
- `TestDefectMigrationCanBeReapplied`: down + up conservan capacidad de insertar.
- `TestDefectMigrationCheckConstraintsRejectInvalidRows`: severidad fuera de catálogo y vacía, estado fuera de catálogo y vacío, descripción de 501 → `23514` con la restricción esperada y cero filas; `resuelto` insertado directo a la base es **válido de almacenamiento** (catálogo completo, subconjunto del alta es cosa del caso de uso).
- `TestDefectMigrationForeignKeyConstraintsScopeReferences`: historia/Sprint de otro proyecto o inexistentes producen `23503` con la FK compuesta esperada (precedencia textual real: la FK de historia queda primero en `pg_constraint`) y cero filas.

### Hash del commit

`a9d6717 feat(db): add defect records migration`.

## UT 2 — `feat(defect): add defect domain`

### Alcance

- `internal/defect/domain/defect.go` (nuevo).
- `tests/unit/defect/domain/defect_test.go` (nuevo).

### RED observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/domain/... -count=1` | FAIL de setup: `no required module provides package .../internal/defect/domain` |

### GREEN observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/domain/... -count=1` | ok (24 casos de primer nivel con subtests) |
| `gofmt -l internal/defect tests/unit/defect` | un archivo sin formato (test); corregido con `gofmt -w` y verificado vacío |
| `go vet ./internal/defect/... ./tests/unit/defect/...` | sin hallazgos |

Cobertura: catálogo ejecutado por combinación (3 severidades × 2 estados iniciales); 501 caracteres multibyte rechazados; trim; NUL; mensajes congelados por prueba (`TestNewDefectMessageConstantsAreStable`); acumulación de 3 errores; falso positivo de `resuelto → InitialStatusMessage` distinto del error de catálogo.

### Hash del commit

`3fb51b3 feat(defect): add defect domain`.

## UT 3 — `feat(defect): add register defect use case`

### Alcance

- `internal/defect/application/register_defect.go` (nuevo).
- `tests/unit/defect/application/register_defect_test.go` (nuevo).

### RED observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/application/... -count=1` | FAIL de setup: paquete `internal/defect/application` inexistente |

### GREEN observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/... -count=1` | ok (dominio + aplicación) |
| `go vet ./internal/defect/...` | sin hallazgos |
| `gofmt -l` | un archivo sin formato (test); corregido |

Cobertura: persistencia válida con descripción recortada; normalización canónica de UUIDs en mayúsculas; acumulación de 5 errores de campo (`story_id` inválido, `sprint_id` vacío, descripción vacía, severidad fuera, `resuelto`); faltantes con mensaje `is required`; paso transparente de sentinelas del repositorio (comparables con `errors.Is`, sin `wrap`); generador de IDs: 0 veces con errores de campo, 1 por intento que alcanza la persistencia.

### Hash del commit

`5c2f012 feat(defect): add register defect use case`.

## UT 4 — `feat(defect): persist defect records in PostgreSQL`

### RED observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/infrastructure/... -count=1` | FAIL de setup: paquete `internal/defect/infrastructure/postgres` inexistente |
| `go build ./tests/integration/defect/...` | no compila por la falta del paquete |

### GREEN observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/infrastructure/... -count=1` | ok (2 pruebas, 7 subpruebas: 2 FKs traducidas + 4 passthrough + no-PgError) |
| `go test ./tests/integration/defect/... -count=1` | ok (36,0 s): paquete completo con migración + repositorio |
| `go build ./...` y `go vet ./...` | exit 0 |

Ciclo corto observado: `defectof` necesitaba el proyecto como argumento (el caso de proyecto desconocido usaba el sembrado) y `fmt` quedó sin uso al recortar un bloque muerto en el test de error de almacenamiento; ambos se corrigieron antes del GREEN final.

Cobertura: persistencia y lectura de vuelta; precedencia proyecto→historia→Sprint (proyecto desconocido, historia+Sprint de otro proyecto, historias/Sprints desconocidos); historia de backlog; Sprint de detección cerrado (`is_closed=true`); tres duplicados exactos; FK violada entre verificación e INSERT vía trigger que borra la historia (→ `ErrStoryNotFound`, traducción determinista sin `500`); error de almacenamiento arbitrario vía trigger que levanta excepción (pasa sin traducir a centinela).

### Hash del commit

`5cc5be3 feat(defect): persist defect records in PostgreSQL`.

## UT 5 — `feat(defect): agregar handler HTTP de registro de defectos`
## UT 5 — `feat(defect): add defect HTTP handler`

### Alcance

- `internal/defect/transport/http/handler.go` (nuevo).
- `tests/unit/defect/transport/http/handler_test.go` (nuevo).

### RED observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/transport/http/... -count=1` | FAIL de setup: paquete `internal/defect/transport/http` inexistente |

### GREEN observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/defect/... -count=1` | ok (4 paquetes) |
| `go vet`/`gofmt -l`/`go build ./...` | limpios |

Ciclos cortos RED→GREEN observados:

1. El caso `nul_description` enviaba por error el byte NUL crudo (JSON inválido → `400` correcto); el caso `422` del NUL viaja como secuencia `\u0000` (JSON válido) que el decodificador convierte en NUL real y el dominio rechaza.
2. `405 method_not_allowed`: el `ServeMux` con patrón `POST /...` responde `405` con body plano *antes* de invocar al handler (igual que effort en producción), así que el subtest invoca al handler directo con `request.SetPathValue(...)` y verifica el JSON del handler (patrón del repositorio).

Cobertura: `201` con exactamente 7 claves (`id, project_id, story_id, sprint_id, description, severity, status`), id del servidor (`00000000-0000-0000-0000-000000000001`), descripción recortada, sin `created_at`/`seq`, `Content-Type: application/json`; `405` JSON para GET/PUT; `400 invalid_request` para 10 cuerpos malformados (vacío, texto, JSON roto, arreglo, null, contenido extra, clave desconocida `priority`, `id` y `created_at` inyectadas, tipo incompatible en `sprint_id`); `422` para `project_id` de ruta inválido y para errores de campo: UUID inválido, requeridos, `resuelto` con mensaje propio, acumulación de 5 errores, NUL, 501 caracteres; `404` con precedencia textual (`project_not_found`, `story_not_found`, `sprint_not_found`); `500 internal_error` genérico sin filtrar detalles del error interno; forma del error body (sin `fields` en `404`).

### Hash del commit

`c8161f7 feat(defect): add defect HTTP handler`.

## UT 6 — `feat(defect): expose defect registration route`
### Alcance

- `internal/api/api.go` (modificado: `DefectDependencies`, `HTTPDependencies.Defect`, `MigrationReadiness.Defect` con `version >= 13`, registro de la ruta).
- `cmd/api/main.go` (modificado: bloque de composición con gate `>= 13`, comentario de versiones, log de disponibilidad).
- `tests/unit/cmd/api/defect_route_test.go` (nuevo).
- `tests/integration/defect/postgres/http_integration_test.go` (nuevo).
- `README.md` (sección "API de defectos") y `openspec/config.yaml` (nota US-16 + 13 pares).

### RED observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/cmd/api/... -count=1 -run Defect` | FAIL de compilación: `api.DefectDependencies`, `readiness.Defect` y la ruta no existen |

### GREEN observado

| Comando | Resultado |
|---|---|
| `go test ./tests/unit/cmd/api/... -count=1 -run Defect` | ok (readiness por versiones 12/13/14/dirty/lookup, 404 sin dependencia, 201 con 1 escritura, GET → 405) |
| `go test ./tests/integration/defect/postgres -run TestDefectHTTPEndToEnd -count=1` | ok (11,2 s) |
| `go test ./internal/... ./tests/unit/... -count=1` | ok: 26 paquetes, 0 FAIL |
| `go test -p 1 ./tests/integration/... -count=1` | ok: los 8 paquetes (defect, effort, migrations, project, projectmember, sprint, story, task); sin regresiones |
| `gofmt -l`, `go vet ./...`, `go build ./...` | limpios |

Cobertura e2e real (Postgres + handler compuesto como en `main.go`): 201 con persistencia verificada leyendo la fila; duplicados reales (2 posts → 2 filas, IDs distintos, sin `UNIQUE` que lo impida); precedencia de 404 real (proyecto desconocido → `project_not_found`; historia+sprint de otro proyecto → `story_not_found`; sprint de otro proyecto → `sprint_not_found`), todas sin filas y sin `fields`; GET → 405 sin escrituras.

### Hash del commit

Registro tras el commit de la UT.
