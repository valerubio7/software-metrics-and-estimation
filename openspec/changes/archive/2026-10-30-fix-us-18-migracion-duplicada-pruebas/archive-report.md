# Informe de cierre y archivo — corrección de pruebas US-18 (#46)

## Resultado

Se quitó la reaplicación redundante de `000014` en tres pruebas de integración del listado de defectos. Sin cambios en código de producción, migraciones, rutas ni specs (`skip_specs: true`).

## Evidencia observada

- RED (base `72dcfe4`): `go test -count=1 ./tests/integration/defect/...` falla en `TestDefectRepositoryListsProjectDefectsWithResolutionSprint`, `TestDefectRepositoryListsEmptyExistingProject` y `TestDefectRepositoryListsUnknownProjectAsNotFound` con SQLSTATE 42701 (`resolution_sprint_id` ya existe).
- GREEN: el mismo comando pasa tras la corrección.
- `go build ./...`, `go vet ./...`: PASS.
- `go test -count=1 ./tests/unit/... ./tests/integration/...`: PASS, 33 paquetes, PostgreSQL real con Testcontainers.
- `openspec validate fix-us-18-migracion-duplicada-pruebas --strict`: PASS.

## Límites

`openspec/specs/defecto` todavía declara el listado (US-18) fuera de alcance aunque `main` ya lo implementa; su actualización corresponde al cambio SDD de US-18 y no se toca aquí.
