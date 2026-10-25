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

Registro tras el commit de la UT.
