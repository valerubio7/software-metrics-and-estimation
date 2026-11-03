# Informe de cierre y archivo — US-18 (#46)

## Resultado

Cambio SDD `us-18-consultar-defectos-proyecto` archivado como `2026-11-03-us-18-consultar-defectos-proyecto` dentro del PR #136, a pedido expreso del usuario y antes del merge. No se fusionó nada: la aprobación y el merge siguen siendo decisión del usuario. La implementación del listado ya estaba en `main` (PRs #117/#118, commit `3035264`; fix de pruebas #126); este PR solo agrega especificación y su archivo.

## Consolidación del spec

El CLI fission-ai/openspec 1.14.1 exige encabezados ingleses (`## Purpose`) para reconstruir specs, y este repo mantiene los specs consolidados con encabezados españoles («## Propósito»). Siguiendo el precedente de TUI-08 (commit `6f5a27e`), el archivo se ejecutó con `openspec archive us-18-consultar-defectos-proyecto --skip-specs --yes` y el delta se aplicó manualmente a `openspec/specs/defecto/spec.md`:

- **Propósito** reescrito según la nota de cambio manual del delta: alta (US-16) y listado (US-18) ofrecidos; resolución (US-17), métricas (US-19) y operación desde la TUI (TUI-09) remitidas a sus especificaciones propias.
- **RENAMED + MODIFIED** aplicados: el requisito «Ofrecer únicamente el registro de defectos» pasó a «Ofrecer el registro y el listado de defectos», con su escenario conservando el título «Sin rutas de consulta, edición o resolución» y los pasos actualizados (versión 14, `405` para métodos de modificación, cero mutaciones/lecturas).
- **ADDED** agregados al final de `## Requirements`: «Consultar el listado de defectos de un proyecto», «Validar referencias y fallar sin filtrar detalles del listado», «Consultar el listado sin modificar los datos» y «Ofrecer el listado solo con el esquema limpio en versión 14», junto con sus trece escenarios.

El spec consolidado queda con 18 requisitos (líneas 387 y 399-489 el bloque nuevo).

## Evidencia observada

- `openspec validate us-18-consultar-defectos-proyecto --strict`: PASS antes de archivar.
- `openspec archive ... --skip-specs --yes`: `Task status: ✓ Complete`; archivado como `2026-11-03-us-18-consultar-defectos-proyecto`; advertencia «Change must have a Why section» no bloqueante (las propuestas del repo están en español, igual en todos los cambios previos).
- `openspec list`: sin cambios activos tras archivar.
- `go build ./...` y `go vet ./...`: PASS.
- `go test ./tests/unit/... -count=1`: PASS (27 paquetes, incluidos `defect`, `cmd/api` y `tui`).
- `go test ./tests/integration/defect/... -count=1`: PASS con PostgreSQL real (64,4 s).
- `git diff --check` limpio en el branch del PR.

## Trazabilidad

| Delta | Aplicación | Evidencia en el repo |
|---|---|---|
| ADDED: contrato del listado | spec consolidado (líneas 399-489) | `tests/unit/cmd/api/defect_list_route_test.go` (sobre, claves, UUID canónico, arreglo vacío), `repository_integration_test.go` (resolución, vacío) |
| ADDED: referencias y errores | spec consolidado | route test 422/404/405 sin lecturas, `list_defects_test.go` (validación antes de leer) |
| ADDED: lectura sin mutación | spec consolidado | route test sin escrituras, `DefectClient.List` read-only (TUI) |
| ADDED: gate versión 14 | spec consolidado | `TestMigrationReadinessResolvesDefectByVersion`, `TestDefectListRouteUnavailableWithoutV14Lister`, README «API de defectos» |
| RENAMED/MODIFIED + propósito | spec consolidado | revisión estructural (grep de encabezados + lectura completa del bloque) |

## Límites

- El CLI no valida el spec consolidado por sus encabezados españoles; la verificación es estructural (encabezados y lectura del bloque agregado), igual que en TUI-08.
- `HEAD` aceptado por el guard del handler sin prueba dedicada y orden por registro sin prueba específica: límites de cobertura documentados en `design.md`, siguen vigentes.
- La integración se ejecutó solo para el paquete de defectos, el relevante para este cambio.

## Entrega

Branch `docs/us-18-consultar-defectos` con los commits del SDD (853d981) y de la consolidación/archivo. PR #136 apunta a `main` con `Closes #46`; pendiente de revisión, aprobación humana y merge. Después del merge quedan pendientes (fuera de este PR): consolidar el SDD de TUI-09 y del updater de estado (US-17) como cambios propios, y resolver la deuda de `gofmt` en `internal/tui/metrics.go`.
