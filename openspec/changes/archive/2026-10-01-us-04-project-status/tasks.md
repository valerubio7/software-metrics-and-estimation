# Tasks: US-04 — Consultar el estado de un proyecto

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 450–650 changed lines (implementation, tests, spec/API documentation) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1: dominio/aplicación/repositorio; PR 2: HTTP, integración y documentación |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: size-exception
400-line budget risk: High

La estimación supera probablemente el umbral de 400 líneas. Se conserva la selección explícita de `single-pr` y la autorización explícita `size:exception`; implementar en un solo PR bajo esa excepción, sin cambiar a PR encadenados. Señalar el riesgo al iniciar apply y mantener el alcance acotado.

## Tasks

- [x] 1. **RED — Dominio:** Añadir pruebas de tabla en `tests/unit/project/domain/` para víspera de inicio, inicio, fin inclusivo, día posterior, proyecto de un día y comparación date-only/año-mes; usar la fecha de referencia fija y confirmar fallo inicial.
- [x] 2. **GREEN — Dominio:** Implementar estado derivado puro en `internal/project/domain/` (`planned`, `active`, `overdue` únicamente), con comparación de fechas de calendario y sin estado persistido; pasar las pruebas de dominio.
- [x] 3. **RED — Aplicación/repositorio:** Añadir pruebas en `tests/unit/project/application/` y `tests/integration/project/postgres/repository_integration_test.go` para recuperar por ID, ausencia y lectura sin escrituras; establecer fallo inicial.
- [x] 4. **GREEN — Aplicación/repositorio:** Incorporar puerto/caso de uso de lectura en `internal/project/application/` y consulta por ID con mapeo de `pgx.ErrNoRows` a `ErrProjectNotFound` en `internal/project/infrastructure/postgres/repository.go`; pasar pruebas unitarias e integración disponibles.
- [x] 5. **RED — HTTP/composición:** Añadir pruebas en `tests/unit/project/transport/http/` y pruebas de integración de proyecto existentes para GET exitoso, UUID inválido (422 sin acceso), no encontrado (404), error interno y respuesta sin campos extra; confirmar fallo inicial.
- [x] 6. **GREEN — HTTP/composición:** Implementar handler y DTO en `internal/project/transport/http/`, registrar `GET /projects/{project_id}` e inyectar fecha productiva UTC desde `internal/api/api.go`; respetar respuestas y errores existentes y pasar pruebas HTTP.
- [x] 7. **TRIANGULATE:** Verificar aislamiento de fecha de referencia, ausencia de llamadas de escritura/dependencia US-03 y persistencia intacta en pruebas; ejecutar `go test ./...` (integración PostgreSQL requiere Docker; reportar si no está disponible).
- [x] 8. **REFACTOR:** Revisar y simplificar implementación en `internal/project/` sin cambiar contrato; actualizar `openspec/specs/project/spec.md` con el requisito aprobado y documentar la ruta solo donde exista documentación API estable. Ejecutar nuevamente `go test ./...` y confirmar que no se añadió migración ni estado `completed`.
