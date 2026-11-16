# Informe de cierre y archivo — US-17 (#45)

## Resultado

Cambio SDD `archive-us-17-actualizar-defecto` archivado en la rama `docs/archive-us-17`, reconstruido a posteriori el 2026-11-16: la implementación ya estaba en `main` desde el 26/10/2026 (PRs #111, #112, #113 y #116) sin su propio SDD. Este PR solo agrega especificación y su archivo; no se fusiona nada y la aprobación/merge siguen siendo decisión del usuario.

## Consolidación del spec

Siguiendo el precedente de TUI-08 y US-18, el archivado se ejecuta con `openspec archive archive-us-17-actualizar-defecto --skip-specs --yes` (el CLI exige encabezados ingleses y este repo mantiene encabezados españoles) y el delta se aplica manualmente a `openspec/specs/defecto/spec.md`:
- **MODIFIED**: el requisito «Ofrecer el registro y el listado de defectos» retira la mención a US-17 de su lista de exclusiones y su escenario «Sin rutas de consulta, edición o resolución» se actualiza para reflejar que `PATCH` ya forma parte del contrato.
- **ADDED** al final de `## Requirements`: los requisitos del contrato de actualización, con sus escenarios.

## Evidencia observada

- `openspec validate archive-us-17-actualizar-defecto --strict`: inicialmente reportó un `ERROR` (el `MODIFIED` omitía el escenario existente `Sin rutas de consulta, edición o resolución` al renombrarlo) y `WARNING`s de longitud (>500 caracteres) en 4 requisitos `ADDED`. Se corrigió conservando el título del escenario existente y acortando la prosa de los 4 requisitos sin perder ningún MUST. Resultado final: `Change 'archive-us-17-actualizar-defecto' is valid` (exit 0).
- `go build ./...`: PASS (exit 0), sin salida.
- `go vet ./...`: PASS (exit 0), sin salida.
- `go test ./tests/unit/defect/... ./tests/unit/cmd/api/... ./tests/integration/defect/... -count=1`: PASS (exit 0) — `ok` en `tests/unit/defect/application` (0.003s), `tests/unit/defect/domain` (0.003s), `tests/unit/defect/infrastructure/postgres` (0.002s), `tests/unit/defect/transport/http` (0.004s), `tests/unit/cmd/api` (0.008s) y `tests/integration/defect/postgres` (28.266s, PostgreSQL real vía Testcontainers).

## Trazabilidad

| Delta | Aplicación | Evidencia en el repo |
|---|---|---|
| ADDED: contrato PATCH | spec consolidado | `tests/unit/cmd/api/defect_route_test.go` (`TestDefectUpdateRoutePersistsLifecycleState`) |
| ADDED: validación de estado/resolución | spec consolidado | `update_defect_test.go` (`TestUpdateDefectValidatesResolutionState`) |
| ADDED: errores y atomicidad | spec consolidado | `repository_integration_test.go` (`TestDefectRepositoryUpdatesResolutionAtomically`, `TestDefectRepositoryRejectsResolutionSprintBeforeDetection`) |
| ADDED: gate versión 14 | spec consolidado | `TestDefectUpdateRouteIsGatedWithoutResolutionMigration` |
| MODIFIED: exclusión retirada | spec consolidado | revisión estructural |

## Límites

- Reconstruido sin acceso a la discusión original de las PRs (solo commits y diffs de Git); los títulos exactos de PR (#111-113, #116) vienen de `git log`, no de la UI de GitHub.
- Sin prueba HTTP de extremo a extremo para `PATCH` y sin prueba de ruta dedicada a la reapertura: límites de cobertura documentados en `design.md`, no corregidos en este cambio porque sería tocar código de producción o pruebas existentes, fuera del alcance acordado.

## Entrega

Branch `docs/archive-us-17`, PR a `main` pendiente de creación (no crees el PR vos mismo ni hagas push; eso lo decide el usuario). Pendiente de revisión, aprobación humana y merge.
