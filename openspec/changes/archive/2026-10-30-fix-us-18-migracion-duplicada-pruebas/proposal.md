# Propuesta: corregir migración duplicada en pruebas de listado de defectos

## Intención

En `main` (`72dcfe4`) fallan tres pruebas de integración de US-18 (#46) en `tests/integration/defect/postgres/repository_integration_test.go`:

- `TestDefectRepositoryListsProjectDefectsWithResolutionSprint`
- `TestDefectRepositoryListsEmptyExistingProject`
- `TestDefectRepositoryListsUnknownProjectAsNotFound`

```
apply migration 000014_add_defect_resolution_sprint.up.sql:
ERROR: column "resolution_sprint_id" of relation "defects" already exists (SQLSTATE 42701)
```

`seededDefectDatabase` ya migra hasta la versión 14 (`defectDatabaseAtVersion(t, 14)`), y cada prueba vuelve a aplicar `000014` con `applyDefectMigration(..., "up")`. La segunda aplicación falla antes de ejercitar el repositorio.

## Cambios propuestos

- Quitar la reaplicación redundante de `000014` en esas tres pruebas; la base sembrada ya está en la versión que necesitan.

## Capacidades

Ninguna nueva ni modificada. No cambia código de producción, migraciones, rutas ni contratos; solo pruebas.

## Impacto

`go test ./tests/integration/defect/...` vuelve a pasar, y con él la suite de integración completa. Fuera de alcance: actualizar `openspec/specs/defecto` para reflejar el listado de US-18 (pendiente de su propio cambio SDD).
