# Tareas del SDD de la actualización de estado y resolución de un defecto (US-17)

Cambio de especificación posterior a la implementación: la actualización ya está en `main` (PRs #111, #112, #113 y #116, consolidados el 26/10/2026). Estas tareas documentan el contrato, lo verifican contra `main` y dejan la trazabilidad de escenarios a pruebas existentes; ninguna tarea modifica código, migraciones, rutas ni composición de pruebas. La operación en la TUI (TUI-09) queda fuera de este cambio. Reconstruido el 2026-11-16 a partir del historial de Git (commits, diffs de las cuatro PRs mencionadas y `git log --grep="US-17"`); no es un registro contemporáneo a la implementación.

## 1. Especificación

- [x] 1.1 Redactar [specs/defecto/spec.md](specs/defecto/spec.md) con `## ADDED Requirements` (contrato de la actualización, validación de estado y Sprint de resolución, mapeo de errores, atomicidad, gate por versión 14) y `## MODIFIED Requirements` sobre «Ofrecer el registro y el listado de defectos» (retira la exclusión de la actualización de ese requisito y actualiza su escenario «Sin rutas de consulta, edición o resolución»), más la nota de cambio manual del «Propósito» para el archivado.

## 2. Verificación de la especificación contra la implementación en `main`

- [x] 2.1 Contrato HTTP: `PATCH /projects/{project_id}/defects/{defect_id}` con cuerpo `{status, resolution_sprint_id}`, `200` con el defecto actualizado, `405` para otros métodos. Fuente: `internal/defect/transport/http/update_handler.go`, `tests/unit/cmd/api/defect_route_test.go` (`TestDefectUpdateRoutePersistsLifecycleState`).
- [x] 2.2 Validación de estado y Sprint de resolución condicionado: `resolution_sprint_id` requerido solo con `status="resuelto"`, rechazado con cualquier otro estado, UUID válido, estado dentro del catálogo completo. Fuente: `internal/defect/application/update_defect.go`, `tests/unit/defect/application/update_defect_test.go` (`TestUpdateDefectValidatesResolutionState`, `TestUpdateDefectPersistsResolvedAndReopenedStates`).
- [x] 2.3 Errores: `422 validation_failed`, `404 defect_not_found`, `404 resolution_sprint_not_found`, `409 resolution_sprint_before_detection`, `400 invalid_request` (cuerpo malformado o claves desconocidas), `500 internal_error` genérico (no probado end-to-end, ver design.md). Fuente: `internal/defect/transport/http/handler.go` (`writeUseCaseError`), `tests/unit/cmd/api/defect_route_test.go` (`TestDefectUpdateRouteMapsValidationAndDomainErrors`, `TestDefectUpdateRouteRejectsMalformedJSONWithoutWrites`), `tests/unit/defect/application/update_defect_test.go` (`TestUpdateDefectPropagatesRepositoryErrors`).
- [x] 2.4 Atomicidad, bloqueo transaccional y coherencia cronológica entre Sprint de detección y de resolución. Fuente: `internal/defect/infrastructure/postgres/repository.go` (`Update`), `tests/integration/defect/postgres/repository_integration_test.go` (`TestDefectRepositoryUpdatesResolutionAtomically`, `TestDefectRepositoryRejectsResolutionSprintBeforeDetection`).
- [x] 2.5 Gate por versión 14, compartido con el listado (US-18). Fuente: `internal/api/api.go` (`ResolveMigrationReadiness`), `tests/unit/cmd/api/defect_route_test.go` (`TestMigrationReadinessResolvesDefectByVersion`, `TestDefectUpdateRouteIsGatedWithoutResolutionMigration`).
- [x] 2.6 Escenarios sin prueba dedicada, documentados como límite (no como incumplimiento): no existe una prueba HTTP de extremo a extremo con PostgreSQL real para `PATCH` (equivalente a `TestDefectHTTPEndToEnd`, que solo cubre `POST`/`GET`), y no existe una prueba de ruta HTTP dedicada a la reapertura (`"resuelto"` → otro estado); ambas quedan cubiertas de forma fragmentada por pruebas de unidad y de repositorio, como se detalla en 2.2-2.4.
- [x] 2.7 Coherencia con documentación vigente: revisar si `README.md` describe esta ruta en alguna sección de API; si no la menciona, se reporta como hallazgo abierto, sin corregirlo en este PR (sería tocar un artefacto fuera del alcance acordado con el usuario para este cambio).

## 3. Validación y entrega

- [x] 3.1 Ejecutar `openspec validate archive-us-17-actualizar-defecto --strict` y `go build ./...`, `go vet ./...` y `go test ./tests/unit/defect/... ./tests/unit/cmd/api/... ./tests/integration/defect/... -count=1`. Registrar en archive-report.md los resultados REALES obtenidos (no inventar números).
- [x] 3.2 Commit(s) con Conventional Commits en la rama `docs/archive-us-17`; PR a `main` pendiente (sin fusionar: la aprobación y el merge son decisión del usuario).

## Seguimiento

- Límites de cobertura declarados en [design.md](design.md): sin prueba HTTP de extremo a extremo para `PATCH`, reapertura sin prueba de ruta dedicada.
- Pendiente excluido del alcance: consolidar el SDD de TUI-09 en su propio cambio.
