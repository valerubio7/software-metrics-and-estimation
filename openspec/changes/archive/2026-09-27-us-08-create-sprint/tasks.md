# Tareas: Crear un Sprint y definir su Sprint Goal

## Pronóstico de carga de revisión

| Campo | Valor |
|-------|-------|
| Líneas modificadas estimadas | 550–750 líneas agregadas y eliminadas (pronóstico original; actual: 1,161) |
| Riesgo del límite de 400 líneas | High |
| Se recomiendan PR encadenados | No |
| División sugerida | Un único PR con todas las unidades de trabajo; `feat/us-08-create-sprint` → `main` |
| Estrategia de entrega | exception-ok |
| Estrategia de cadena | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

La estimación original de 550–750 líneas cruzaba dominio, aplicación, persistencia, migración, transporte HTTP, composición y arranque, además de pruebas unitarias, pruebas con PostgreSQL real y documentación. El resultado final suma 1,161 líneas autoradas agregadas y eliminadas: Unidad 1, 283; Unidad 2, 302; Unidad 3, 576. Se acepta `size:exception` para entregar todo en un único PR, sin reducir pruebas ni alcance. El pronóstico original se conserva como pronóstico, no como conteo real.

### Unidades de trabajo sugeridas

| Unidad | Objetivo | PR probable | Comando de prueba enfocado | Harness de ejecución | Límite de reversión |
|--------|----------|-------------|----------------------------|----------------------|---------------------|
| 1 | Crear el contrato de dominio y caso de uso de Sprint con validación del goal, generación de ID y escritura única. | PR único; head: `feat/us-08-create-sprint`; base: `main` | `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application` | N/A: pruebas unitarias sin servicio externo ni ejecución del proceso API. | Revertir `internal/sprint/domain/`, `internal/sprint/application/` y sus pruebas unitarias; no afecta otros módulos mientras no se componga la ruta. |
| 2 | Añadir el esquema 000003 y persistencia PostgreSQL con FK y traducción selectiva de la constraint. | PR único; head: `feat/us-08-create-sprint`; base: `main` | `go test ./tests/integration/sprint/postgres -run 'TestSprintRepository'` | `go test ./tests/integration/sprint/postgres -run 'TestSprintRepository'` con Docker/Testcontainers y PostgreSQL 16. | Revertir el repositorio y sus pruebas; conservar la migración aplicada y sus datos. Ejecutar el down solo con aprobación explícita porque elimina Sprints. |
| 3 | Exponer y habilitar Sprint de manera compatible, probar escenarios HTTP/versionado y documentar la operación. | PR único; head: `feat/us-08-create-sprint`; base: `main` | `go test ./tests/unit/sprint/transport/http ./tests/unit/cmd/api ./tests/integration/sprint/postgres ./tests/integration/story/postgres` | Pruebas de integración con Docker/Testcontainers para PostgreSQL 16 y harness de proceso API de readiness ya usado por las pruebas existentes. | Retirar la ruta Sprint, la nueva composición/gating, las pruebas asociadas y la sección nueva del README; dejar el esquema y los datos intactos. |

No habrá ramas hijas ni PR hijos: todo el trabajo se entregará en un único PR desde `feat/us-08-create-sprint` directamente a `main`. No se requiere tracker ni diagrama de dependencias de PR hijos. El usuario autorizó explícitamente la excepción de tamaño. El apply está completo; el archive nativo es el siguiente paso recomendado y aún no se realizó.

## Fase 1: Contratos y lógica de dominio/aplicación

- [x] 1.1 **RED — Dominio:** crear `tests/unit/sprint/domain/sprint_test.go` con casos table-driven para goal válido preservado literalmente, goal ausente, goal compuesto solo por espacios y aceptación de contenido sin reglas adicionales; confirmar que falla antes de existir el tipo/constructor esperado.
- [x] 1.2 **GREEN — Dominio:** crear `internal/sprint/domain/sprint.go` con entidad `Sprint`, error de validación por campo y constructor que rechaza únicamente goal vacío/blanco, conservando el texto recibido.
- [x] 1.3 **REFACTOR — Dominio:** ajustar `internal/sprint/domain/sprint.go` y `tests/unit/sprint/domain/sprint_test.go` sin cambiar el contrato, y verificar con `go test ./tests/unit/sprint/domain`.
- [x] 1.4 **RED — Aplicación:** crear `tests/unit/sprint/application/create_sprint_test.go` para verificar ID generado por servidor, `ProjectID` y goal preservados, exactamente una llamada al repositorio, cero llamadas si el goal es inválido y propagación del error de repositorio; confirmar que falla por ausencia del caso de uso.
- [x] 1.5 **GREEN — Aplicación:** crear `internal/sprint/application/create_sprint.go` con `CreateSprintCommand`, `SprintRepository`, `IDGenerator`, `ErrProjectNotFound` y el caso de uso que valida antes de generar/persistir y escribe exactamente una vez.
- [x] 1.6 **REFACTOR — Aplicación:** refinar `internal/sprint/application/create_sprint.go` y `tests/unit/sprint/application/create_sprint_test.go` manteniendo casos y límites de interfaces pequeños; verificar con `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application`.

## Fase 2: Esquema y persistencia PostgreSQL

- [x] 2.1 **RED — Repositorio PostgreSQL real:** crear `tests/integration/sprint/postgres/repository_integration_test.go` con Testcontainers/PostgreSQL 16 para comprobar persistencia exacta, FK nombrada ante proyecto ausente y que una violación de otra constraint no se traduzca a `ErrProjectNotFound`; confirmar fallo antes de añadir migración/repositorio. Las pruebas deben poder omitirse con `testing.Short()` y requerir Docker fuera de modo short.
- [x] 2.2 **GREEN — Migración:** crear `internal/project/infrastructure/postgres/migrations/000003_create_sprints.up.sql` y `internal/project/infrastructure/postgres/migrations/000003_create_sprints.down.sql`; crear solo `sprints(id UUID PRIMARY KEY, project_id UUID NOT NULL, sprint_goal TEXT NOT NULL)` con `sprints_project_id_fkey` a `projects(id) ON DELETE RESTRICT`, sin modificar 000001/000002 ni agregar campos/reglas no acordados.
- [x] 2.3 **GREEN — Repositorio:** crear `internal/sprint/infrastructure/postgres/repository.go` con INSERT parametrizado y traducción exclusiva de SQLSTATE `23503` de `sprints_project_id_fkey` a `application.ErrProjectNotFound`; propagar el resto de errores sin clasificarlos como proyecto ausente.
- [x] 2.4 **REFACTOR — Persistencia:** revisar migración y repositorio para mantener paridad up/down y traducción selectiva; ejecutar `go test ./tests/integration/sprint/postgres -run 'TestSprintRepository'` con Docker y confirmar que otros errores de FK no se mapean a 404.

## Fase 3: Contrato HTTP y composición compatible

- [x] 3.1 **RED — Handler: método y UUID:** ampliar `tests/unit/sprint/transport/http/handler_test.go` con un caso por método distinto de POST (405, cero escrituras) y UUID de `project_id` inválido (422 con error de campo, cero escrituras); confirmar los fallos antes de crear el handler.
- [x] 3.2 **RED — Handler: JSON y goal:** ampliar `tests/unit/sprint/transport/http/handler_test.go` con JSON malformado, múltiples objetos JSON y campos desconocidos (incluidos `id` y campos de estado/asignación) que devuelven 400; probar goal ausente/vacío/blanco con 422 y cero escrituras; verificar que el texto válido se conserva.
- [x] 3.3 **RED — Handler: persistencia y fallos:** ampliar `tests/unit/sprint/transport/http/handler_test.go` para 201 con JSON de `id`, `project_id` normalizado y `sprint_goal`; `ErrProjectNotFound` a 404; error inesperado a 500 genérico sin detalle interno; los rechazos no deben escribir.
- [x] 3.4 **RED — Matriz de amenazas, composición HTTP:** añadir en `tests/unit/cmd/api/main_test.go` o pruebas de API existentes casos que demuestren ruta Sprint ausente sin dependencias, POST válido registrado al suministrar dependencias, métodos GET/PUT/DELETE sin escritura y conservación de la composición/rutas previas con dependencias solo de proyecto o historia; confirmar fallo antes de registrar la ruta.
- [x] 3.5 **RED — Matriz de amenazas, readiness de arranque:** ampliar `tests/unit/cmd/api/main_test.go` con versiones limpias 2 y 3, estado dirty y error de lookup: v2 habilita historias y no Sprint; v3 limpia habilita ambas; dirty/error no habilitan rutas dependientes de migración y proyectos siguen disponibles. Confirmar los fallos antes del nuevo gating.
- [x] 3.6 **RED — Integración Sprint completa:** crear `tests/integration/sprint/postgres/http_integration_test.go` para verificar creación 201, igualdad de respuesta y fila persistida, proyecto inexistente 404 y ausencia de escritura ante rechazo; aplicar migraciones 000001–000003 con PostgreSQL 16 en Testcontainers y confirmar fallo antes de completar el cableado.
- [x] 3.7 **RED — Integración de regresión/versionado:** ampliar `tests/integration/story/postgres/http_integration_test.go` para cubrir v2 limpia (historias sí, Sprint no), v3 limpia (ambas), dirty y error de lookup (ninguna ruta dependiente de migración; proyectos disponibles); comprobar readiness mediante el harness de proceso existente y confirmar fallo antes de la nueva composición.
- [x] 3.8 **GREEN — Handler:** crear `internal/sprint/transport/http/handler.go` con decodificación estricta de un único objeto, rechazo de campos desconocidos, validación de UUID de ruta, uso del caso de aplicación y contratos HTTP definidos (201, 400, 404, 405, 422 y 500); no exponer errores internos.
- [x] 3.9 **GREEN — Composición API:** modificar `internal/api/api.go` para incorporar `SprintDependencies` y registrar `POST /projects/{project_id}/sprints` solo cuando se provean; preservar firma y comportamiento de `api.NewHTTPHandler` y usar una composición explícita compatible con dependencias opcionales de historias y Sprint.
- [x] 3.10 **GREEN — Disponibilidad en arranque:** modificar `cmd/api/main.go` para leer una sola vez `schema_migrations(version, dirty)`, habilitar historias solo si está limpio y `version >= 2`, Sprint solo si está limpio y `version >= 3`, y mantener la ruta de proyectos ante dirty/error; registrar disponibilidad y fallos conforme al patrón existente.
- [x] 3.11 **REFACTOR — HTTP/composición:** ajustar handler, constructor compatible y selección de dependencias manteniendo todos los RED verdes; verificar `go test ./tests/unit/sprint/transport/http ./tests/unit/cmd/api` y preservar explícitamente las rutas preexistentes.

## Fase 4: Integración, regresión y documentación

- [x] 4.1 **GREEN — Integración y regresión:** completar el cableado/configuración real de `tests/integration/sprint/postgres/http_integration_test.go` y `tests/integration/story/postgres/http_integration_test.go`; asegurar asociación persistida, ausencia de asignaciones de historias y compatibilidad de rutas de proyectos/historias.
- [x] 4.2 **REFACTOR — Verificación global:** ejecutar `go test ./...` conforme a la configuración del proyecto, con Docker disponible para integración; revisar que migraciones, routing y errores observados correspondan a los contratos y que ninguna prueba dependa de datos externos no declarados.
- [x] 4.3 **Documentación:** actualizar `README.md` junto con el cambio visible para describir la aplicación externa de 000003, `POST /projects/{project_id}/sprints`, request/response, errores y requisito de esquema; advertir que ejecutar la migración down elimina los Sprints persistidos.
