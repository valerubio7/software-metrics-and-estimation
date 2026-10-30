# Tareas: corregir migración duplicada en pruebas de listado de defectos

## 1. Corrección (strict TDD)

- [x] 1.1 RED: `go test -count=1 ./tests/integration/defect/...` falla en las tres pruebas de listado con SQLSTATE 42701 sobre `000014`.
- [x] 1.2 Quitar `applyDefectMigration(t, pool, defectResolutionMigrationVersion, "up")` de las tres pruebas, que ya parten de `seededDefectDatabase` (versión 14).
- [x] 1.3 GREEN: `go test -count=1 ./tests/integration/defect/...` pasa.

## 2. Verificación

- [x] 2.1 `go build ./...`, `go vet ./...`, `go test ./tests/unit/...` y `go test ./tests/integration/...` pasan.
- [x] 2.2 `git diff` solo toca el archivo de pruebas y este cambio OpenSpec.
