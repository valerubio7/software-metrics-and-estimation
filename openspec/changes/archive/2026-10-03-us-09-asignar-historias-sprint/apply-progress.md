# Apply Progress: Asignar historias a Sprint (US-09)

## Estado

- Cambio: `us-09-asignar-historias-sprint`
- Almacén de artefactos: OpenSpec
- Modo: TDD estricto
- Naturaleza de este registro: **retroactivo**. La implementación, las pruebas y la fusión ya ocurrieron antes de que este expediente SDD existiera; este documento reconstruye la evidencia de aplicación a partir del código, las pruebas y los commits reales.
- Commit de comportamiento: `f544916 feat(story): integrar US-09 sobre migraciones canónicas` — 23 archivos, 1158 inserciones, 45 eliminaciones (confirmado con `git show --stat f544916`).
- Commit de cierre ODD original (ahora superado por este expediente SDD): `3ad5433 docs(odd): cerrar verificación y entrega de US-09`.
- Commit de fusión a `main`: `63a6527 Merge pull request #78 from valerubio7/fix/us09-consolidation`.
- Issue cerrada: #37.
- Todas las tareas 1.1–4.3 de `tasks.md` están completas (ver evidencia abajo).

## Evidencia de ciclo TDD (reconstruida del código y pruebas existentes)

| Tarea | Archivo de prueba | Capa | RED | GREEN | REFACTOR |
|---|---|---|---|---|---|
| 1.1–1.3 | `tests/unit/story/application/assign_stories_test.go` | Unitaria, aplicación | Casos de selección vacía/duplicada y delegación con alcance de proyecto, escritos contra la interfaz `ProjectScopedStorySprintAssigner` antes de la implementación final. | `go test ./tests/unit/story/application` pasa con `AssignStoriesUseCase.Execute` implementado en `internal/story/application/assign_stories.go:44`. | Interfaces `StorySprintAssigner`/`ProjectScopedStorySprintAssigner` mantienen límites pequeños; comportamiento sin cambios. |
| 2.1–2.4 | `tests/integration/story/postgres/assignment_integration_test.go` (277 líneas) | Integración PostgreSQL | Casos de atomicidad, bloqueo de fila, condiciones de carrera y preservación del backlog, contra el repositorio final `AssignStoriesForProject`. | Migraciones `000007`/`000008` aplicadas; repositorio en `internal/story/infrastructure/postgres/repository.go:38` con transacción, `FOR UPDATE`, `EXISTS` y manejo de `23505`. | `tests/integration/sprint/postgres/repository_integration_test.go` y `tests/integration/migrations/valid_history_convergence.sh` ampliados para convergencia hasta v8. |
| 3.1–3.4 | `tests/unit/story/transport/http/assign_stories_test.go` (159 líneas), `tests/unit/cmd/api/main_test.go` (+59 líneas) | Unitaria HTTP y composición | Casos de método, UUID, JSON, selección vacía/duplicada, canonicalización y readiness `version >= 8`, contra el handler y composición finales. | `internal/story/transport/http/assign_stories_handler.go` y gating en `internal/api/api.go`/`cmd/api/main.go`. | Rutas preexistentes (proyectos, historias, Sprints, integrantes) verificadas sin regresión. |
| 4.1–4.3 | `tests/integration/story/postgres/http_integration_test.go` (+74 líneas), `README.md` (+30 líneas) | Integración end-to-end y documentación | Escenarios HTTP+PostgreSQL completos antes del cableado final de composición. | Endpoint completo operativo; documentación del contrato y requisito de esquema ≥8. | N/A — documentación sin comportamiento ejecutable propio. |

## Evidencia de verificación histórica (al momento de la fusión)

Según el registro de entrega previo (ya eliminado de `odd/tasks/us09-consolidation.md`, cuyo contenido se reconstruye aquí por referencia, no se cita textualmente): la suite completa `go test ./...` reportó 183 tests + 346 subtests = 529 PASS, 0 FAIL/SKIP en el momento de la fusión del PR #78.

## Evidencia de verificación re-ejecutada hoy (2026-10-03)

| Comando | Resultado |
|---|---|
| `go build ./...` | Exit 0. |
| `go vet ./...` | Exit 0, sin advertencias. |
| `go test ./tests/unit/...` | Todos los paquetes `ok`, incluidos `tests/unit/story/application`, `tests/unit/story/transport/http` y `tests/unit/cmd/api`. |
| Pruebas de integración (`tests/integration/story/postgres/assignment_integration_test.go` y demás, Testcontainers/Docker) | **No ejecutadas**: Docker no está disponible en este entorno de verificación. Limitación de entorno, no defecto de código; la evidencia histórica de su ejecución exitosa está en el commit `f544916`. |

## Desviaciones y hallazgos

- Ninguno. La revisión de hoy contra `openspec/specs/historia/spec.md` (líneas 843–889) y `openspec/specs/sprint/spec.md` (líneas 60–77) no encontró inconsistencias, errores ni validaciones faltantes respecto al comportamiento implementado.
- La única observación registrada es de diseño, no de defecto: `dependencies.Stories` (`cmd/api/main.go:53`, no nulo ya desde gate `>=2`) y `readiness.Assignment` (`internal/api/api.go:95`, gate `>=8`) son dos comprobaciones de versión separadas que deben revisarse juntas ante cambios futuros de migración. Documentado en `design.md`.

## Reconciliación final

- 10 de 10 tareas marcadas en `tasks.md` están completas (1.1–4.3, más las tres de verificación re-ejecutada hoy, salvo integración omitida por entorno).
- Próximo paso recomendado: archivo nativo SDD de este expediente retroactivo.
