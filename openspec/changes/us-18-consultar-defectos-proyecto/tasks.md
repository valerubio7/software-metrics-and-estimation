# Tareas del SDD del listado de defectos (US-18)

Cambio de especificación posterior a la implementación: el listado ya está en `main` (PRs #117/#118, commit `3035264`; el fix de pruebas es el PR #126). Estas tareas documentan el contrato, lo verifican contra `main` y dejan la trazabilidad de escenarios a pruebas existentes; ninguna tarea modifica código, migraciones, rutas ni composición de pruebas. La operación en la TUI (TUI-09) y la resolución de estado (US-17) quedan fuera de este cambio.

## 1. Especificación

- [x] 1.1 Redactar [specs/defecto/spec.md](specs/defecto/spec.md) con `## ADDED Requirements` (contrato del listado, validación de referencias y errores, lectura sin mutación, gate por versión 14), `## RENAMED Requirements` («Ofrecer únicamente el registro de defectos» → «Ofrecer el registro y el listado de defectos») y `## MODIFIED Requirements` con todos los escenarios vigentes, más la nota de cambio manual del «Propósito» para el archivado.

## 2. Verificación de la especificación contra la implementación en `main`

- [x] 2.1 Contrato HTTP: conferir ruta `GET /projects/{project_id}/defects` con `GET`/`HEAD`, sobre `{project_id, defects}`, siete claves por elemento, `resolution_sprint_id` anulable, arreglo vacío explícito, orden `ORDER BY seq`, sin filtros/paginación, canonicalización en minúsculas del UUID. Fuente: `internal/defect/transport/http/list_handler.go`, `internal/defect/application/list_defects.go`, `internal/defect/infrastructure/postgres/repository.go` (`ListByProject`).
- [x] 2.2 Errores: `422 validation_failed` (UUID inválido, sin lecturas), `404 project_not_found`, `405 method_not_allowed` (con `PATCH` y demás métodos y cero lecturas), `500 internal_error` genérico. Fuente: `tests/unit/cmd/api/defect_list_route_test.go` (`TestDefectListRouteMapsInvalidAndUnknownProjects`, `TestDefectListRouteRejectsMutationMethodsWithoutWrites`).
- [x] 2.3 Lectura de solo lectura y alcance por proyecto: aislamiento entre proyectos, arreglo vacío no nulo, consistencia repetida sin multiplicar filas. Fuente: `tests/unit/defect/application/list_defects_test.go`, `tests/unit/cmd/api/defect_list_route_test.go` (`TestDefectListRouteReturnsProjectScopedItems`, `TestDefectListRouteReturnsExplicitEmptyArray`), `tests/integration/defect/postgres/repository_integration_test.go` (`TestDefectRepositoryListsProjectDefectsWithResolutionSprint`, `TestDefectRepositoryListsEmptyExistingProject`, `TestDefectRepositoryListsUnknownProjectAsNotFound`).
- [x] 2.4 Gates: `DefectList = esquema limpio && versión >= 14`, independiente del alta (versión 13) y de los demás; módulo ausente → `404` genérico; listador ausente → `405` en `GET`. Fuente: `internal/api/api.go` (`ResolveMigrationReadiness`, composición), `tests/unit/cmd/api/defect_route_test.go` (`TestMigrationReadinessResolvesDefectByVersion`), `tests/unit/cmd/api/defect_list_route_test.go` (`TestDefectListRouteUnavailableWithoutV14Lister`).
- [x] 2.5 Coherencia con documentación vigente: la sección «API de defectos» del `README.md` (rutas, sobre, las siete claves, errores y gates del alta y del listado) coincide con el delta; no requiere cambios en este PR.

## 3. Validación y entrega

- [x] 3.1 Ejecutar `openspec validate us-18-consultar-defectos-proyecto --strict` y verificar `go build ./...`, `go vet ./...` y las suites de pruebas del listado antes de escribir 3.2. Resultados: validación estricta PASS; `go build ./...` y `go vet ./...` OK; `go test ./tests/unit/... -count=1` PASS (27 paquetes, incluidos los de `defect`, `cmd/api` y `tui`); `go test ./tests/integration/defect/... -count=1` PASS con PostgreSQL real (64,4 s).
- [x] 3.2 Commit único `docs(defecto)` con los cuatro artefactos y crear un PR a `main` con `Closes #46`; no fusionar: la aprobación y el merge son decisión del usuario, y el archivado del cambio se ejecuta después de fusionar.

## Seguimiento

- Límites de cobertura declarados en [design.md](design.md): orden sin prueba dedicada y `HEAD` sin prueba explícita.
- Pendientes excluidos del alcance: consolidar el SDD de TUI-09 y del updater de US-17 en sus propios cambios.
