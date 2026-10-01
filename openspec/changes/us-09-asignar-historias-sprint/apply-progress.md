# Progreso de aplicación — HU-09

## Estado acumulado

- Estado nativo v2 consumido para `us-09-asignar-historias-sprint`: apply ready, `nextRecommended: apply`, sin bloqueadores/dependencias pendientes; `actionContext.mode: repo-local`, workspace autorizado. La continuación de esta sesión fue autorizada por el usuario tras diagnóstico de solo lectura y estado nativo actualizado. No se modificó `internal/api`.
- Corte autorizado #1: tareas 1–2, contrato de aplicación y caso de uso; estrategia `auto-chain` / `feature-branch-chain`; meta ≤200 líneas. Se conserva la rama tracker actual. Sin commit, push, ramas nuevas ni PR.
- Tareas completadas y marcadas: 1 y 2 en `tasks.md` (`- [x]`). Tareas 3–9 continúan pendientes y fuera de este corte.

## TDD Cycle Evidence

| Tarea | Archivo de prueba | Capa | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1 | `tests/unit/story/application/assign_stories_test.go` | Unit | `go test ./tests/unit/story/application` pasó antes de cambios | Prueba compiló con fallo esperado por API de asignación inexistente | N/A (tarea de pruebas) | Cubiertos selección individual/lote, selección vacía/repetida y rechazos de repositorio para sprint/historia inexistente, proyecto distinto y asociación previa; cero escrituras en validación local | N/A |
| 2 | `tests/unit/story/application/assign_stories_test.go` | Unit | Safety net anterior | RED de tarea 1 | Implementado `AssignStoriesUseCase` y prueba enfocada pasó | Entrada individual y múltiple, validación local sin escritura y errores de elegibilidad propagados por una única llamada atómica al puerto | Revisión sin cambio adicional necesario; pruebas permanecen verdes |

Detalle de RED: tras añadir pruebas primero, `go test ./tests/unit/story/application` falló como se esperaba porque faltaban `NewAssignStoriesUseCase`, `AssignStoriesCommand` y errores tipados. Después de implementar el caso de uso, el mismo comando pasó. Durante GREEN se detectó que `ErrStoryNotFound` ya existía; se reutilizó en lugar de duplicarlo.

## Cambios y verificación

Archivos de código añadidos:

- `internal/story/application/assign_stories.go`: comando, errores de aplicación, puerto atómico e implementación de coordinación/validación vacía y duplicada.
- `tests/unit/story/application/assign_stories_test.go`: pruebas unitarias con repositorio falso.

Verificaciones:

- `go test ./tests/unit/story/application` — pasó tras GREEN.
- `go test -count=1 ./tests/unit/story/...` — pasó en application, domain y HTTP de story.
- `go test ./...` — sigue fallando por el bloqueo preexistente y ajeno a este corte: `internal/api/api.go:103-111`, `undefined: stories`; también impide compilar `cmd/api`, pruebas de integración sprint/story y `tests/unit/cmd/api`. No se corrigió aquí.
- `git diff --check` en los dos archivos nuevos — sin hallazgos.
- Conteo de código + pruebas añadido en este corte: 140 líneas (59 producción + 81 pruebas), calculado contra `/dev/null` por ser archivos nuevos sin seguimiento. Dentro de la meta de 200 y del máximo de 400 líneas del PR.

## Diseño, desviaciones y pendientes

La aplicación valida lista no vacía y ausencia de IDs repetidos antes de llamar al puerto una sola vez con el lote completo. El puerto `StorySprintAssigner` establece como contrato que la persistencia implemente atomicidad y verifique existencia, proyecto y asociación previa. No se añadió acoplamiento a PostgreSQL ni se cambian historias, pertenencia al backlog o estado. La regla de Sprint cerrado permanece fuera de este corte, conforme a la dependencia US-12.

No se hicieron cambios a tareas fuera del corte. Las líneas pendientes exactas de tasks.md:

- [ ] 3. **RED — Persistencia y migración:** en `tests/integration/story/postgres/` añadir pruebas PostgreSQL para asociación válida, proyecto incompatible, inexistencias, duplicados, atomicidad ante fallo de un elemento y unicidad bajo concurrencia cuando el harness permita coordinación determinista; añadir prueba de conservación de proyecto/estado/backlog. Ejecutar `go test ./...` y verificar fallos esperados.
- [ ] 4. **GREEN — Repositorio transaccional:** crear migración reversible en `internal/project/infrastructure/postgres/migrations/` y persistencia en `internal/story/infrastructure/postgres/`; comprobar elegibilidad y escribir el lote en una transacción única, con rollback completo y traducción específica de constraints. Ejecutar `go test ./...` con PostgreSQL disponible; confirmar que toda prueba de integración aplicable pasa.
- [ ] 5. **RED — Contrato HTTP:** en `tests/unit/story/transport/http/` probar JSON estricto, ruta/UUIDs, lista vacía y repetida, códigos y respuestas para éxito, not-found, conflicto y error interno; confirmar que entrada inválida no invoca el caso de uso. Ejecutar `go test ./...` y observar los fallos esperados.
- [ ] 6. **GREEN — Endpoint e integración de rutas:** implementar handler en `internal/story/transport/http/`, composición/ruta en `internal/api/` y gating de disponibilidad en `cmd/api/`; fijar un único código y esquema de éxito, y documentar errores deterministas sin exponer errores SQL. Ejecutar `go test ./...`.
- [ ] 7. **GREEN — Regla de cierre dependiente:** inspeccionar el contrato aprobado de US-12 (#40) y conectar la asignación a su fuente de verdad, sin agregar campo, enum, endpoint ni semántica propia. Si US-12 no está disponible o no define cómo determinar cierre, dejar la ruta deshabilitada y escalar el bloqueo; no declarar satisfecha esta regla. Añadir pruebas unitarias y de integración en `tests/unit/story/application/`, `tests/unit/story/transport/http/` y `tests/integration/` según el contrato real; ejecutar `go test ./...`.
- [ ] 8. **TRIANGULATE — Garantías y regresión:** ampliar/ajustar `tests/unit/` y `tests/integration/` para demostrar atomicidad de todo rechazo, persistencia completa, ausencia de asignaciones parciales y no regresión de creación/consulta/actualización de historias y creación de Sprint. Ejecutar `go test ./...` con los prerrequisitos de PostgreSQL disponibles y revisar explícitamente los resultados omitidos por falta de Docker.
- [ ] 9. **REFACTOR — Claridad y documentación:** refactorizar únicamente tras la triangulación, conservando contratos; actualizar `README.md` con ruta, prerequisito de migración, solicitud/respuesta y errores finales. Ejecutar `go test ./...` y verificar que la documentación no prometa asignación a Sprint cerrado si la dependencia US-12 sigue bloqueada.

---

## Continuación de aplicación — Corte #2 (tareas 3–4)

- Estado nativo v2 consumido antes de editar: cambio `us-09-asignar-historias-sprint`, store `openspec`, `applyState: ready`, `nextRecommended: apply`, sin bloqueadores; `actionContext.mode: repo-local` y raíz autorizada = workspace del repositorio. Proyección recibida reportaba 2/9 tareas hechas. No se inició otra fase ni se hicieron commits/PRs/ramas.
- Corte #2 autorizado: `auto-chain`, `feature-branch-chain`, tareas 3–4 solamente; presupuesto meta 300 líneas, sin excepción. Resultado del corte: 287 líneas agregadas (repo + fixtures/migración + pruebas), debajo de 300/400. Conteo compuesto de 66 líneas en archivos versionados modificados y 221 líneas en los archivos nuevos de este corte. Artefactos previos de corte #1 no incluidos en este conteo.
- Tareas 3 y 4 se marcaron `- [x]` en `tasks.md` tras pruebas enfocadas verdes. Quedan 5–9 sin tocar; la tarea 7 sigue dependiendo del contrato US-12.

### TDD Cycle Evidence

| Tarea | Prueba | Capa | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 3. Persistencia/migración | `tests/integration/story/postgres/assignment_integration_test.go` | Integración PostgreSQL 16/Testcontainers | No se pudo ejecutar baseline completo del paquete: éste importa `internal/api` que no compila; pruebas dirigidas en modo de lista de archivos evitan ese paquete | Pruebas escritas primero y `go test tests/integration/story/postgres/repository_integration_test.go tests/integration/story/postgres/assignment_integration_test.go -run '^TestAssignStories'` falló compilando porque `AssignStories` aún no existía | Tras el repositorio y migración, las pruebas dirigidas pasaron | Validez con preservación backlog/proyecto/estado, sprint/historia inexistentes, lote mixto incompatible/inexistente sin escritura parcial, asociación previa, fallo forzado de INSERT con rollback, dos asignaciones concurrentes con un único ganador y migración down/up | `gofmt`; repetición de todos los casos asignados en el mismo comando de integración, verde |
| 4. Repositorio transaccional | La misma suite de integración | Integración PostgreSQL real | `go test ./tests/unit/story/application ./internal/story/infrastructure/postgres` pasó; paquete de infraestructura no tiene tests propios | RED del task 3 observado antes de producción | Operación implementada en una sola transacción, bloqueos `FOR KEY SHARE`, comprobaciones de recursos/proyecto/asociación y traducción del constraint PK a `ErrStoryAlreadyAssigned`; lote insertado completo y rollback ante fallo | Casos concurrentes y de error SQL forzado ejercitaron carrera y rollback | Código gofmt; migración compuesta refuerza el mismo proyecto con FKs de `(id, project_id)` a ambas tablas y restricción única del par Sprint-historia |

### Archivos del corte #2

- `internal/project/infrastructure/postgres/migrations/000005_create_sprint_stories.up.sql` y `.down.sql`: tabla de unión con PK única, FKs compuestas y restricciones de soporte reversibles; no se añadió dato de cierre.
- `internal/story/infrastructure/postgres/repository.go`: `AssignStories` valida sprint e historias bajo transacción, compara proyectos, detecta asociaciones previas, inserta la selección y traduce la colisión de unicidad; otros errores SQL no se disfrazan.
- `tests/integration/story/postgres/assignment_integration_test.go`: pruebas PostgreSQL de migración reversible, éxito/campos conservados, rechazos, rollback por fallo de escritura y carrera concurrente.
- `tests/integration/story/postgres/repository_integration_test.go`: fixture ahora aplica migración de Sprint y la nueva migración HU-09.

### Verificación y desviaciones

- RED: comando de integración de lista explícita falló como se esperaba con `repo.AssignStories undefined` antes de producción.
- GREEN/TRIANGULATE: `go test tests/integration/story/postgres/repository_integration_test.go tests/integration/story/postgres/assignment_integration_test.go -run '^(TestSprintStoriesMigration|TestAssignStories)' -count=1` — pasó con Docker/PostgreSQL 16 disponible.
- `go test ./tests/unit/story/application ./internal/story/infrastructure/postgres` — pasó.
- `go test ./...` — bloqueado por fallo de compilación preexistente fuera del corte: `internal/api/api.go:103-111`, `undefined: stories`; también impide construir el paquete integrado completo y `cmd/api`. No se tocó `internal/api`.
- `git diff --check` — limpio.
- Desviación de diseño: se eligió la opción más fuerte recomendada, con FKs compuestas que hacen imposible insertar una asociación de proyectos distintos incluso fuera del repositorio. No se hizo prueba concurrente de dos lotes solapados con inserción parcial, sí carrera determinista de una asociación duplicada; la transacción comparte esas garantías.
- La regla de Sprint cerrado no se implementa, ya que depende de US-12 y queda fuera de este corte.

### Trabajo restante

Líneas exactas pendientes en `tasks.md`:

- [ ] 5. **RED — Contrato HTTP:** en `tests/unit/story/transport/http/` probar JSON estricto, ruta/UUIDs, lista vacía y repetida, códigos y respuestas para éxito, not-found, conflicto y error interno; confirmar que entrada inválida no invoca el caso de uso. Ejecutar `go test ./...` y observar los fallos esperados.
- [ ] 6. **GREEN — Endpoint e integración de rutas:** implementar handler en `internal/story/transport/http/`, composición/ruta en `internal/api/` y gating de disponibilidad en `cmd/api/`; fijar un único código y esquema de éxito, y documentar errores deterministas sin exponer errores SQL. Ejecutar `go test ./...`.
- [ ] 7. **GREEN — Regla de cierre dependiente:** inspeccionar el contrato aprobado de US-12 (#40) y conectar la asignación a su fuente de verdad, sin agregar campo, enum, endpoint ni semántica propia. Si US-12 no está disponible o no define cómo determinar cierre, dejar la ruta deshabilitada y escalar el bloqueo; no declarar satisfecha esta regla. Añadir pruebas unitarias y de integración en `tests/unit/story/application/`, `tests/unit/story/transport/http/` y `tests/integration/` según el contrato real; ejecutar `go test ./...`.
- [ ] 8. **TRIANGULATE — Garantías y regresión:** ampliar/ajustar `tests/unit/` y `tests/integration/` para demostrar atomicidad de todo rechazo, persistencia completa, ausencia de asignaciones parciales y no regresión de creación/consulta/actualización de historias y creación de Sprint. Ejecutar `go test ./...` con los prerrequisitos de PostgreSQL disponibles y revisar explícitamente los resultados omitidos por falta de Docker.
- [ ] 9. **REFACTOR — Claridad y documentación:** refactorizar únicamente tras la triangulación, conservando contratos; actualizar `README.md` con ruta, prerequisito de migración, solicitud/respuesta y errores finales. Ejecutar `go test ./...` y verificar que la documentación no prometa asignación a Sprint cerrado si la dependencia US-12 sigue bloqueada.

## Pausa de alcance acordada

El usuario eligió esperar a US-12 (#40), no ampliar HU-09. La issue US-12 sigue abierta y no proporciona una representación/fuente de verdad implementada para determinar si un Sprint está cerrado. No comenzar tareas 5–9 ni habilitar la ruta hasta que ese contrato exista; luego revisar tareas y diseño frente al contrato efectivo. No inventar un campo, estado o endpoint de cierre.

## Corrección de composición: assigner project-scoped obligatorio

La prueba `TestAssignmentRouteCompositionRequiresAssigner` ahora entrega un assigner que implementa `ProjectScopedStorySprintAssigner`, en vez del repositorio legacy sin alcance de proyecto. Se preserva la aserción de que, sin assigner, la ruta no se registra y responde 404. El RED uncached observado fue HTTP 500 con cero llamadas cuando se suministró el assigner legacy; tras actualizar el fake a `AssignStoriesForProject`, `go test -count=1 ./tests/unit/cmd/api` pasó. También pasaron `go test ./...` y `git diff --check`.

## Corrección del bloqueo del verificador: proyecto de ruta

El handler ahora valida `project_id` como UUID y lo propaga al caso de uso. La aplicación elige el puerto project-scoped; PostgreSQL compara dentro de la transacción el proyecto recibido en la ruta con el proyecto del Sprint antes de evaluar las historias o ejecutar escrituras. El mismo paso valida las historias respecto del proyecto del Sprint, por lo que Sprint e historias deben coincidir con el proyecto seleccionado. Los rechazos ocurren antes de cualquier asociación; se conserva el puerto legado para compatibilidad de composición no-HTTP.

### Strict TDD

- RED: `go test -count=1 ./tests/unit/story/transport/http ./tests/unit/story/application` falló como se esperaba: la ruta inválida fue aceptada con HTTP 201 y la aplicación no tenía el contrato project-scoped.
- GREEN: el mismo comando pasó tras validar/propagar el proyecto y aplicar la comparación transaccional.
- TRIANGULATE: `go test -count=1 tests/integration/story/postgres/repository_integration_test.go tests/integration/story/postgres/assignment_integration_test.go -run '^(TestSprintStoriesMigration|TestAssignStories)'` pasó, incluyendo mismatch de ruta sin escrituras.
- REFACTOR: se preservó el puerto legado mientras la petición HTTP exige el puerto project-scoped; la suite completa pasó.



Después de los cortes 1–2, el usuario autorizó reparar por separado la composición rota de API que impedía la suite global. Se corrigió `internal/api/api.go` para componer desde `HTTPDependencies.Stories`, se consolidó `cmd/api/main.go` sobre `ResolveMigrationReadiness` y se preservaron umbrales: Projects siempre; Stories v2; updater v3; backlog lister v4; Sprints v3; dirty/error cierran las rutas dependientes de esquema. Se eliminó un import de prueba no usado y se ajustó el escenario v1 para soltar primero la tabla `sprint_stories` dependiente.

Verificación independiente anterior: `go test ./tests/unit/cmd/api`, `go test ./internal/api ./cmd/api ./tests/unit/cmd/api`, `go test ./tests/integration/story/postgres -run TestAPIStartupRoutesFollowMigrationState -count=1`, `go test ./...` y `git diff --check` pasaron.

## Continuación autorizada — Corte 3 (tareas 5–9)

El usuario reemplazó explícitamente la dependencia US-12 (#40) con `Sprint.is_closed BOOLEAN NOT NULL DEFAULT false`. Se actualizaron propuesta, diseño, deltas de especificación, tareas SDD y especificación vigente de Sprint. Migración 000006 agrega la columna con default false; la asociación consulta el flag dentro de la transacción y rechaza lote cerrado sin escrituras. Se implementaron el handler POST `/projects/{project_id}/sprints/{sprint_id}/stories`, composición, gating desde schema v6 y documentación.

Strict TDD: RED observado con `go test ./tests/unit/story/transport/http` fallando porque `ErrSprintClosed` y `NewAssignStoriesHandler` no existían. GREEN: `go test ./tests/unit/story/transport/http ./tests/unit/story/application ./tests/unit/cmd/api` pasó. Integración/triangulación: `go test ./tests/integration/story/postgres -run '^TestAssignStoriesRejectsClosedSprintWithoutWrites$' -count=1` pasó. Suite completa: `go test ./...` pasó, incluyendo Testcontainers PostgreSQL para project, sprint y story. Sin fallos ni omisiones reportados.
