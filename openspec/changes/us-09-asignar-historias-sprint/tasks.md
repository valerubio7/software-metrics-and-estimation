# Tareas: Asignar historias a un Sprint (HU-09)

## Pronóstico de carga de revisión

| Campo | Valor |
|-------|-------|
| Líneas modificadas estimadas totales | 450–700 |
| Presupuesto máximo por PR | 400 líneas modificadas |
| Riesgo del cambio acumulado | Alto si se entrega en un solo PR |
| Se recomiendan PR encadenados | Sí |
| Estrategia de entrega aprobada | `auto-chain` |
| Estrategia de cadena aprobada | `feature-branch-chain` |
| Excepción `size:exception` | No autorizada ni necesaria si cada corte queda dentro del presupuesto |
| Decisión necesaria antes de apply | No; la estrategia fue elegida por el usuario |

El usuario rechazó una excepción de tamaño, eligió PRs encadenados y seleccionó `feature-branch-chain`. Cada PR debe ser un corte cohesivo, incluir sus pruebas y documentación pertinentes, y mantenerse en ≤400 líneas agregadas + eliminadas. No comprimir ni recortar pruebas para satisfacer el límite. La rama actual `feat/us-09-asignar-historias-sprint` es la rama tracker/final desde `main`; los PR hijos se crearán únicamente con autorización de publicación y en el orden indicado abajo. El tracker será draft/no-merge y ningún PR se crea automáticamente durante apply.

### Cortes de revisión previstos

| Orden | Rama/corte | Tareas | Objetivo y límite | Dependencias |
|------:|-------------|--------|------------------|--------------|
| 1 | `feat/us-09-asignar-historias-sprint-01-core` | 1–2 | Contrato de aplicación y coordinación del caso de uso con pruebas unitarias; sin transporte ni persistencia PostgreSQL. Meta: ≤200 líneas. | Ninguna; PR #1 apunta a la rama tracker HU-09. |
| 2 | `feat/us-09-asignar-historias-sprint-02-persistence` | 3–4 | Migración, asociación persistida, transacción/rollback e integración PostgreSQL. Meta: ≤300 líneas. | PR #2 apunta al PR #1. |
| 3 | `feat/us-09-asignar-historias-sprint-03-http` | 5–9 | Handler, composición, gating, reglas de cierre conforme a US-12, regresión y README. Meta: ≤350 líneas. | PR #3 apunta al PR #2; la fuente de verdad de cierre queda establecida por decisión del usuario como `is_closed`. |

Las cifras por corte son metas preliminares, no conteos medidos. Si una unidad excede 400 líneas tras una única división honesta, detener esa unidad y solicitar decisión; no crear un PR sobredimensionado ni alterar el alcance aprobado. La rama tracker solo integra hijos después de revisión y no se fusiona a `main` hasta completar la cadena.

### Regresión del verificador: proyecto de la ruta

- [x] RED — Agregar cobertura HTTP para `project_id` inválido, aplicación para propagar el proyecto de la ruta e integración PostgreSQL que rechaza un proyecto de ruta distinto sin escrituras. `go test -count=1 ./tests/unit/story/transport/http ./tests/unit/story/application` falló por ignorar el parámetro del proyecto en HTTP y por la ausencia del contrato de aplicación.
- [x] GREEN — Validar el UUID `project_id`, pasarlo al caso de uso y comparar dentro de la transacción el proyecto de ruta con el proyecto del Sprint antes de validar/escribir historias; usar el contrato project-scoped para la persistencia real. Pruebas unitarias e integración enfocadas pasaron.
- [x] TRIANGULATE — Ejecutar casos de asignación PostgreSQL para verificar que incompatibilidad del proyecto de ruta deja cero asociaciones y que los rechazos/caminos existentes siguen íntegros.
- [x] REFACTOR — Mantener el puerto legado para composición existente y agregar el puerto project-scoped para garantizar el contrato HTTP en el repositorio PostgreSQL. `go test ./...` pasó.

## Secuencia TDD (estricta)

- [x] 1. **RED — Contrato de aplicación:** en `tests/unit/story/application/` agregar pruebas para asignación individual y por lote, lote vacío/IDs repetidos, Sprint e historias inexistentes, incompatibilidad de proyecto y asociación previa; verificar cero escrituras ante cada rechazo. Ejecutar `go test ./...` y registrar el fallo esperado antes de producción.
- [x] 2. **GREEN — Caso de uso:** implementar comando, errores tipados, interfaz de persistencia y coordinación en `internal/story/application/`; satisfacer las pruebas del paso 1 sin acoplar la aplicación a PostgreSQL ni alterar backlog/estado. Ejecutar `go test ./...`.
- [x] 3. **RED — Persistencia y migración:** en `tests/integration/story/postgres/` añadir pruebas PostgreSQL para asociación válida, proyecto incompatible, inexistencias, duplicados, atomicidad ante fallo de un elemento y unicidad bajo concurrencia cuando el harness permita coordinación determinista; añadir prueba de conservación de proyecto/estado/backlog. Ejecutar `go test ./...` y verificar fallos esperados.
- [x] 4. **GREEN — Repositorio transaccional:** crear migración reversible en `internal/project/infrastructure/postgres/migrations/` y persistencia en `internal/story/infrastructure/postgres/`; comprobar elegibilidad y escribir el lote en una transacción única, con rollback completo y traducción específica de constraints. Ejecutar `go test ./...` con PostgreSQL disponible; confirmar que toda prueba de integración aplicable pasa.
- [x] 5. **RED — Contrato HTTP:** en `tests/unit/story/transport/http/` probar JSON estricto, ruta/UUIDs, lista vacía y repetida, códigos y respuestas para éxito, not-found, conflicto y error interno; confirmar que entrada inválida no invoca el caso de uso. Ejecutar `go test ./...` y observar los fallos esperados.
- [x] 6. **GREEN — Endpoint e integración de rutas:** implementar handler en `internal/story/transport/http/`, composición/ruta en `internal/api/` y gating de disponibilidad en `cmd/api/`; fijar un único código y esquema de éxito, y documentar errores deterministas sin exponer errores SQL. Ejecutar `go test ./...`.
- [x] 7. **GREEN — Regla de cierre:** por decisión del usuario, sustituir la dependencia US-12 (#40) con `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`; rechazar asignación a Sprint cerrado y cubrirlo en integración.
- [x] 8. **TRIANGULATE — Garantías y regresión:** ampliar/ajustar `tests/unit/` y `tests/integration/` para demostrar atomicidad de todo rechazo, persistencia completa, ausencia de asignaciones parciales y no regresión de creación/consulta/actualización de historias y creación de Sprint. Ejecutar `go test ./...` con los prerrequisitos de PostgreSQL disponibles y revisar explícitamente los resultados omitidos por falta de Docker.
- [x] 9. **REFACTOR — Claridad y documentación:** refactorizar únicamente tras la triangulación, conservando contratos; actualizar `README.md` con ruta, prerequisito de migración, solicitud/respuesta y errores finales. Ejecutar `go test ./...` y verificar que la documentación describa `is_closed`.

## Límites de aplicación

La estrategia de entrega está resuelta por decisión del usuario: `auto-chain` con `feature-branch-chain`; no se requiere `size:exception` mientras cada PR se mantenga dentro de 400 líneas modificadas. Antes de cada nuevo corte, medir el diff contra su rama padre y detenerse si excede el límite después de una división honesta. Mantener la pertenencia al Product Backlog y el estado de historias; no crear representación de cierre. La migración debe aplicarse externamente, no desde el servicio; su rollback puede borrar datos de planificación y no es un mecanismo rutinario de reversión. Los PRs tracker/hijos no se publican ni crean sin autorización explícita del usuario.

### Decisión de cierre

El usuario autorizó reemplazar la dependencia US-12 (#40) por `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`. El corte 3 implementa esta fuente de verdad y las tareas 5–9 están completadas; los resultados finales de verificación se registran en `apply-progress.md`.

### Regresión: composición requiere assigner project-scoped

- [x] RED — Actualizar la prueba de composición para proporcionar un assigner que implemente `ProjectScopedStorySprintAssigner`; conservar la comprobación de que, sin assigner, la ruta devuelve 404. Antes del fake project-scoped, el test dirigido falló como se esperaba con HTTP 500 y cero asignaciones.
- [x] GREEN — Agregar al fake de la prueba `AssignStoriesForProject` y validar éxito de composición; ejecutar el test enfocado uncached, la suite `go test ./...` y `git diff --check`.
