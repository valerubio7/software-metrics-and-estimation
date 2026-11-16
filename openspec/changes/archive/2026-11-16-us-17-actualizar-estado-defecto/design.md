# Diseño de la actualización de estado y resolución de un defecto (US-17)

## Contexto

Véanse la motivación en [proposal.md](proposal.md) y el contrato en [specs/defecto/spec.md](specs/defecto/spec.md). La implementación ya está en `main`: caso de uso en `internal/defect/application/update_defect.go` (`DefectUpdater`, `UpdateDefectUseCase`), transporte HTTP en `internal/defect/transport/http/update_handler.go`, repositorio PostgreSQL en `internal/defect/infrastructure/postgres/repository.go` (`Update`), migración `000014` (agrega `resolution_sprint_id` con FK `(resolution_sprint_id, project_id) → sprints(id, project_id)`), y composición/cableado en `internal/api/api.go` y `cmd/api/main.go`. Se consolidó en cuatro PRs: #111 (`feat(defect): add lifecycle validation (US-17 tracker)`, caso de uso y validación de dominio), #112 (`feat(defect): persist resolution Sprint atomically (US-17 1/2)`, repositorio, migración, handler y cableado), #113 (`feat(api): expose defect lifecycle updates (US-17 2/2)`, el mismo alcance HTTP sobre otra base de rama) y #116 (`fix(defect): integrate US-17 lifecycle into main`, consolidación final a `main`, 26/10/2026). Este documento registra las decisiones tomadas en esa implementación; no introduce una nueva. Reconstruido el 2026-11-16 a partir del historial de Git (`git log --grep`, diffs de los cuatro merges) porque el cambio se completó sin su propio SDD.

## Objetivos

**Objetivos:** especificar con fidelidad el contrato implementado, mantener la transición de estado como una operación atómica y transaccional, validar la coherencia cronológica entre Sprint de detección y de resolución, y dejar trazabilidad verificable entre escenarios y pruebas existentes.

**No objetivos:** no se modifican código, migraciones, rutas ni composición; no se especifican el listado (US-18, ya consolidado), el cálculo de métricas de defectos (US-19) ni la operación desde la TUI (TUI-09, pendiente de su propio SDD).

## Decisiones registradas (tal como quedaron implementadas)

1. **`PATCH` parcial con dos campos.** El cuerpo acepta únicamente `status` y `resolution_sprint_id`; cualquier otra clave es rechazada por el decodificador estricto compartido (`decodeDefectObject`) con `400 invalid_request`.
2. **`resolution_sprint_id` condicionado al estado.** El caso de uso (`UpdateDefectUseCase.Execute`) exige `resolution_sprint_id` presente y UUID válido solo si `status == "resuelto"`; lo rechaza como `"must be omitted unless status is resuelto"` si se envía con cualquier otro estado. Esto permite reabrir un defecto (`en_correccion` o `detectado`) limpiando el Sprint de resolución.
3. **Validación de estado contra el catálogo completo.** A diferencia del alta (que usa `InitialStatuses`, sin `resuelto`), la actualización valida contra `domain.Statuses` completo (`detectado`, `en_correccion`, `resuelto`), porque esta es la única operación que puede llevar un defecto a `resuelto`.
4. **Atomicidad y bloqueo en el repositorio.** `PostgresDefectRepository.Update` abre una transacción, bloquea con `FOR KEY SHARE` el defecto y los Sprints relevantes (detección y resolución) antes de escribir, y aplica `UPDATE defects SET status = $3, resolution_sprint_id = $4 WHERE id = $1 AND project_id = $2`. Si el Sprint de resolución no pertenece al proyecto, devuelve `ErrResolutionSprintNotFound`; si el defecto no existe en ese proyecto, `ErrDefectNotFound`.
5. **Coherencia cronológica entre Sprints.** El repositorio rechaza con `ErrResolutionSprintBeforeDetection` cuando el Sprint de resolución es anterior al Sprint de detección del defecto (orden de Sprints del proyecto), sin dejar ninguna fila modificada.
6. **Códigos de error mapeados en el handler compartido.** `writeUseCaseError` (compartido con el alta) traduce `ErrDefectNotFound` → `404 defect_not_found`, `ErrResolutionSprintNotFound` → `404 resolution_sprint_not_found`, `ErrResolutionSprintBeforeDetection` → `409 resolution_sprint_before_detection`, `*domain.ValidationError` → `422 validation_failed` con `fields`, cualquier otro error → `500 internal_error` genérico.
7. **Gate por versión 14, compartido con el listado.** La ruta de actualización depende de `resolution_sprint_id` (migración `000014`), igual que el listado (US-18): con esquema en versión menor a 14, sucio, o con error de resolución, la ruta no se registra y la solicitud recibe `404` genérico.
8. **Contrato aditivo.** El alta (US-16) y el listado (US-18) no cambian: catálogos, rutas y gates propios se conservan íntegros.

## Riesgos y notas de verificación

- **Sin prueba HTTP de extremo a extremo con PostgreSQL real para `PATCH`.** `TestDefectHTTPEndToEnd` (`tests/integration/defect/postgres/http_integration_test.go`) cubre el alta y el listado end-to-end, pero no ejercita la ruta `PATCH` contra una base real a través del handler HTTP completo. La cobertura de la actualización está dividida: a nivel de unidad con repositorio falso (`tests/unit/cmd/api/defect_route_test.go`) y a nivel de integración directamente contra el repositorio PostgreSQL sin pasar por HTTP (`tests/integration/defect/postgres/repository_integration_test.go`). Se documenta como límite de cobertura, no como requisito no cumplido: cada efecto observable (persistencia, mapeo de errores, bloqueo de transacción) tiene prueba, pero no en una única prueba combinada de extremo a extremo.
- **Reapertura sin prueba HTTP dedicada.** La reapertura (`"resuelto"` → `"en_correccion"`/`"detectado"`) está probada a nivel de unidad (`TestUpdateDefectPersistsResolvedAndReopenedStates`) y de repositorio (`TestDefectRepositoryUpdatesResolutionAtomically`, segunda mitad), pero no hay una prueba de ruta HTTP (`defect_route_test.go`) dedicada a la reapertura; las pruebas de ruta existentes cubren resolver, no reabrir.
- **Consolidaciones pendientes fuera de este cambio.** TUI-09 carece de delta propio; el propósito renombrado la referencia como pendiente para no dejar contradicciones en el spec consolidado.

## Migración

Ninguna nueva: la actualización reutiliza la migración `000014` (ya consolidada por el delta de US-18), que agrega `resolution_sprint_id` con su FK y su índice parcial. Este cambio no agrega columnas ni tablas.
