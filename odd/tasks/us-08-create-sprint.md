# ODD: Crear un Sprint y definir su Sprint Goal (US-08)

**Estado:** SDD apply y archive completos: 24/24 tareas SDD y 3/3 unidades ODD cerradas, sin tareas de implementación pendientes. El cambio está archivado en `openspec/changes/archive/2026-09-27-us-08-create-sprint/`. Los intentos test-first estructurales de las unidades 2 y 3 no ejecutaron cuerpos de tests ni demostraron fallos de comportamiento; las pruebas PostgreSQL reales pasaron después de implementar.

## Objetivo

Permitir crear y persistir un Sprint asociado a un proyecto existente mediante `POST /projects/{project_id}/sprints`, con UUID generado por el servidor y Sprint Goal obligatorio, sin consultar ni asignar historias.

## Problema y motivo

US-08 permite que una persona cree un Sprint para un proyecto existente y defina el Sprint Goal que orientará su trabajo. La creación no requiere consultar ni asignar historias.

## Alcance

- Implementar dominio, aplicación, persistencia PostgreSQL, migraciones, endpoint HTTP, composición/arranque, pruebas de regresión y documentación según OpenSpec.
- Mantener las pruebas y documentación junto con el comportamiento que verifican o explican, en tres unidades de trabajo revisables.
- Aplicar las migraciones externamente; el down de Sprint elimina datos y no se ejecutará como rollback automático.

### Fuera de alcance

- Asignar historias, registrar historias completadas, cerrar Sprints o consultar Sprints históricos.
- Fechas, duración, estado, orden, identificador visible adicional, reglas de contenido del goal no acordadas o dependencia funcional de US-07.
- Cambiar Git config, crear ramas adicionales/hijas, publicar cambios o iniciar operaciones remotas.

## Criterios de aceptación US-08

- [x] Una solicitud válida crea exactamente un Sprint para el proyecto indicado, conserva el Sprint Goal y devuelve sus datos con UUID generado por el servidor.
- [x] La ausencia de Sprint Goal se rechaza antes de persistir y se informa al cliente; no se persiste ningún Sprint.
- [x] Un `project_id` inválido o inexistente se rechaza sin persistencia y con el error correspondiente.
- [x] Crear Sprint no requiere ni consulta historias y no asigna ninguna historia.

## Restricciones técnicas

Las siguientes condiciones son restricciones técnicas derivadas de `openspec/changes/archive/2026-09-27-us-08-create-sprint/design.md`; **no son criterios de aceptación adicionales de US-08**:

- Registrar la ruta Sprint únicamente con esquema limpio versión 3 o superior.
- Conservar la ruta de historias con esquema limpio versión 2 o superior.
- Con estado `dirty` o error al consultar el esquema, no habilitar rutas dependientes de migraciones y conservar disponible la ruta de proyectos.

## Restricciones y decisiones de entrega

| Tema | Decisión vigente |
|---|---|
| OpenSpec | Cambio archivado en `openspec/changes/archive/2026-09-27-us-08-create-sprint/`. La especificación canónica completa es `openspec/specs/sprint/spec.md` (3 requisitos; no existía una spec canónica previa). El informe de archivo está en `openspec/changes/archive/2026-09-27-us-08-create-sprint/archive-report.md`; no se solicitó un `verify-report.md` separado. |
| TDD | **STRICT TDD**. Fuente: `openspec/config.yaml` (`strict_tdd: true`) y esta instrucción. Secuencia obligatoria RED → GREEN → REFACTOR por unidad. |
| Runner de suite | `go test ./...` (configurado en `openspec/config.yaml`). Ejecutar las verificaciones enfocadas de cada unidad y la suite pertinente antes de cerrar la unidad. |
| Estimación | Pronóstico original: 550–750 líneas modificadas; riesgo High frente al umbral de 400. Actual: 1,161 líneas autoradas agregadas y eliminadas (283 + 302 + 576). No reducir alcance ni pruebas para ajustar el conteo. |
| Estrategia | `exception-ok`; `size:exception` aceptado para una PR de `feat/us-08-create-sprint` a `main`. Sin PRs encadenadas. |
| Ramas / publicación | No crear ramas adicionales/hijas ni PRs hijos. No hay autorización para push, crear PR ni realizar ninguna operación remota. |
| Ejecución | Cada unidad corresponde al worker dedicado de SDD apply. Todo el código futuro, las tareas y los commits permanecen en `feat/us-08-create-sprint`. |
| Commits | Al completar cada unidad, crear un commit de trabajo con mensaje Conventional Commit. Identidad local esperada: `SantyCalz <santyxd321@gmail.com>`. No añadir `Co-Authored-By` ni atribución de IA. No modificar Git config. |
| Estado/evidencia | Registrar resultados reales, comandos y evidencias al completar. No inventar resultados, hashes de commit ni evidencia; reflejar cada estado según evidencia verificada. |

## Plan de unidades de trabajo

Cada unidad requiere su propia evidencia RED → GREEN → REFACTOR, verificación enfocada, harness de runtime o N/A justificado, límite de rollback y referencia de commit Conventional Commit al completarse. Las pruebas y los documentos pertinentes permanecen con su comportamiento.

### Unidad 1 — Dominio y aplicación de Sprint — COMPLETA

**Tareas:** 1.1–1.6. **Resultado:** entidad/validación y caso de uso con generación de ID, validación antes de persistir y exactamente una escritura.

- **RED → GREEN → REFACTOR:** completado en secuencia estricta; evidencia detallada en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`.
- **Check enfocado:** el worker ejecutó `go test -v ./tests/unit/sprint/domain ./tests/unit/sprint/application` (exit 0): dominio, 2 pruebas principales + 4 subtests; aplicación, 3 pruebas principales + 2 subtests. Verificación independiente del padre: `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application` (exit 0; ambos paquetes `ok`).
- **Runtime harness:** N/A; unidad de dominio/aplicación pura sin frontera de runtime externo.
- **Rollback boundary:** retirar `internal/sprint/domain/`, `internal/sprint/application/`, `tests/unit/sprint/domain/` y `tests/unit/sprint/application/`, sin composición de ruta ni cambios ajenos.
- **Commit al completar:** `e9dd5ff feat(sprint): add domain and creation use case`; identidad `SantyCalz <santyxd321@gmail.com>`.

### Unidad 2 — Migración y repositorio PostgreSQL

**Tareas:** 2.1–2.4. **Resultado:** migración 000003 reversible en archivos y repositorio con FK nombrada y traducción selectiva del error.

- **Estado:** tareas 2.1–2.4 completas según resultado de SDD apply y verificación independiente del padre. El commit es `ed8aac9 feat(sprint): persist sprints in PostgreSQL`, local en la misma rama feature; identidad `SantyCalz <santyxd321@gmail.com>`.
- **RED → GREEN → REFACTOR (evidencia estricta):** el primer intento test-first solo mostró un RED estructural en la configuración del paquete porque todavía faltaba el paquete del repositorio; no se ejecutó el cuerpo de ningún test ni se observó un fallo de aserción de comportamiento. Tras agregar la migración y el repositorio, las pruebas reales de integración con PostgreSQL 16 corrieron y pasaron. No se presenta el fallo estructural inicial como RED de comportamiento.
- **Check enfocado del worker:** `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 -v ./tests/integration/sprint/postgres -run 'TestSprintRepository'` — exit 0; ambos tests pasaron (3.965s).
- **Verificación independiente del padre:** `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres -run 'TestSprintRepository'` — exit 0; paquete `ok` (3.961s).
- **Runtime harness:** Testcontainers usó PostgreSQL 16 desechable a través del socket Docker rootless local autorizado. Pasaron persistencia exacta, FK de proyecto inexistente y clasificación de clave primaria duplicada. El servicio de usuario está activo; `DOCKER_HOST=unix:///run/user/1000/docker.sock` funciona; Docker Engine es 29.8.1; se usó la imagen local `postgres:16-alpine` sin pull. No se tocó ninguna base de datos persistente del usuario ni se ejecutó migration down.
- **Rollback boundary:** retirar únicamente `internal/project/infrastructure/postgres/migrations/000003_create_sprints.up.sql`, `internal/project/infrastructure/postgres/migrations/000003_create_sprints.down.sql`, `internal/sprint/infrastructure/postgres/repository.go` y `tests/integration/sprint/postgres/repository_integration_test.go`, preservando todos los cambios de Unidad 1. Nunca ejecutar down automáticamente; puede eliminar Sprints persistidos.
- **Commit:** `ed8aac9 feat(sprint): persist sprints in PostgreSQL`; identidad `SantyCalz <santyxd321@gmail.com>`.

### Unidad 3 — HTTP, composición, arranque, regresión y README — COMPLETA

**Tareas:** 3.1–3.11 y 4.1–4.3. **Resultado:** endpoint y composición compatibles, gating independiente por versión de esquema, cobertura de regresión e instrucciones de operación.

- **RED → GREEN → REFACTOR:** completado. Las primeras fallas de handler/composición/readiness/integración fueron estructurales porque los paquetes/API todavía no existían; no se afirma que fueran RED de comportamiento. Tras implementar, las aserciones de comportamiento pasaron.
- **Check enfocado:** `go test ./tests/unit/sprint/transport/http ./tests/unit/cmd/api` — exit 0, ambos paquetes pasaron. Las integraciones focalizadas Sprint/story/PostgreSQL también pasaron con `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres ./tests/integration/story/postgres` (exit 0).
- **Suite completa:** worker y verificación independiente del padre ejecutaron `DOCKER_HOST=unix:///run/user/1000/docker.sock go test ./...` — exit 0; todas las unidades e integraciones pasaron y ninguna integración fue omitida. El padre también verificó `git diff --check` — pasó.
- **Runtime harness:** Testcontainers utilizó PostgreSQL 16 desechable vía el socket Docker rootless local; el servicio estaba activo, Engine 29.8.1 e imagen local `postgres:16-alpine` sin pull. No se usó base persistente.
- **Rollback boundary:** retirar ruta Sprint, composición/gating nuevo, pruebas asociadas y sección nueva del README; dejar esquema y datos intactos.
- **Tamaño real:** 576 líneas autoradas modificadas (540 adiciones, 36 eliminaciones).
- **Commit al completar:** `d616714 feat(sprint): expose sprint creation over HTTP`; local en `feat/us-08-create-sprint`, identidad `SantyCalz <santyxd321@gmail.com>`.

## Checklist de tareas y evidencia por ruta/trigger

**Ruta** identifica el artefacto/capa a cambiar; **trigger y evidencia** identifica el comportamiento que debe provocar y la observación que debe guardarse, incluyendo el ciclo RED/GREEN/REFACTOR de su unidad. Las 24 filas registran tareas completadas; el detalle está en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`.

| ID estable | Tarea | Ruta | Trigger y evidencia requerida | Estado |
|---|---|---|---|---|
| 1.1 | **RED — Dominio:** pruebas table-driven de goal válido preservado, ausente, solo espacios y contenido sin reglas adicionales. | `tests/unit/sprint/domain/sprint_test.go` | Ejecutada antes del tipo/constructor; evidencia RED por capacidad ausente en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`. | Completada |
| 1.2 | **GREEN — Dominio:** entidad, error por campo y constructor que rechaza solo goal vacío/blanco, preservando el texto. | `internal/sprint/domain/sprint.go` | Casos 1.1 ejecutados tras implementación; evidencia GREEN en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`. | Completada |
| 1.3 | **REFACTOR — Dominio:** refinar sin cambiar contrato. | Dominio y prueba de la unidad 1 | Secuencia estricta RED → GREEN → REFACTOR registrada en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`; verificación final conjunta: `go test -v ./tests/unit/sprint/domain ./tests/unit/sprint/application` (exit 0). | Completada |
| 1.4 | **RED — Aplicación:** pruebas de ID, asociación, goal, una escritura, cero escrituras con goal inválido y propagación de error. | `tests/unit/sprint/application/create_sprint_test.go` | Ejecutada antes del caso de uso; evidencia RED por capacidad ausente en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`. | Completada |
| 1.5 | **GREEN — Aplicación:** command, interfaces, errores y caso de uso que valida antes de generar/persistir y escribe una vez. | `internal/sprint/application/create_sprint.go` | Casos 1.4 ejecutados; evidencia GREEN y ausencia de writes en validación fallida en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`. | Completada |
| 1.6 | **REFACTOR — Aplicación:** refinar interfaces y caso de uso manteniendo cobertura. | Aplicación y pruebas unitarias de Sprint | `go test -v ./tests/unit/sprint/domain ./tests/unit/sprint/application` (worker, exit 0: dominio 2 pruebas principales + 4 subtests; aplicación 3 pruebas principales + 2 subtests). Verificación independiente: `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application` (exit 0; ambos paquetes `ok`). | Completada |
| 2.1 | **RED — PostgreSQL:** integración real para escritura exacta, FK nombrada y clasificación selectiva de error. | `tests/integration/sprint/postgres/repository_integration_test.go` | Primer intento test-first: error estructural de configuración porque faltaba el paquete del repositorio; no corrió ningún cuerpo de test ni hubo RED de aserción. Después de agregar migración/repositorio, el worker ejecutó `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 -v ./tests/integration/sprint/postgres -run 'TestSprintRepository'` (exit 0; ambos tests pasaron, 3.965s); verificación independiente: `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres -run 'TestSprintRepository'` (exit 0; paquete `ok`, 3.961s). | Completada |
| 2.2 | **GREEN — Migración:** crear up/down 000003 solo con columnas y FK acordadas. | `internal/project/infrastructure/postgres/migrations/000003_create_sprints.{up,down}.sql` | Migración verificada por las pruebas de integración reales PostgreSQL 16, incluyendo FK de proyecto inexistente; no alterar 000001/000002. | Completada |
| 2.3 | **GREEN — Repositorio:** INSERT parametrizado y traducir solo SQLSTATE 23503 de `sprints_project_id_fkey`. | `internal/sprint/infrastructure/postgres/repository.go` | Pruebas reales cubrieron persistencia exacta, FK de proyecto inexistente y clasificación de clave primaria duplicada. | Completada |
| 2.4 | **REFACTOR — Persistencia:** verificar paridad up/down y errores selectivos. | Migración, repositorio y pruebas de integración | Worker: `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 -v ./tests/integration/sprint/postgres -run 'TestSprintRepository'` (exit 0; ambos tests pasaron, 3.965s). Padre: `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres -run 'TestSprintRepository'` (exit 0; paquete `ok`, 3.961s). Runtime con Testcontainers/PostgreSQL 16 desechable. | Completada |
| 3.1 | **RED — Handler:** método no POST da 405 y UUID de ruta inválido da 422, sin writes. | `tests/unit/sprint/transport/http/handler_test.go` | Tests fueron escritos primero; primer run falló estructuralmente porque no existía el paquete. Después pasaron las aserciones 405/422 y cero escrituras. | Completada |
| 3.2 | **RED — Handler:** JSON malformado, múltiple o con campos desconocidos da 400; goal ausente/vacío/blanco da 422. | `tests/unit/sprint/transport/http/handler_test.go` | Evidencia de comportamiento tras implementar: JSON estricto, validación de goal, cero writes en rechazos y conservación del texto válido; suite enfocada pasó. | Completada |
| 3.3 | **RED — Handler:** 201, 404 por proyecto ausente y 500 genérico ante error inesperado. | `tests/unit/sprint/transport/http/handler_test.go` | Tests cubren respuesta JSON, privacidad de error y cero escrituras ante rechazo; el paquete HTTP unitario pasó. | Completada |
| 3.4 | **RED — Matriz de amenazas/composición:** ruta ausente sin deps, POST disponible con deps y verbos no admitidos sin escritura; preservar composición existente. | `tests/unit/cmd/api/main_test.go` o pruebas de API existentes | Primer fallo estructural por APIs de composición ausentes; luego pasaron tests de ruta opcional, métodos y rutas previas. | Completada |
| 3.5 | **RED — Matriz de amenazas/readiness:** v2 limpia habilita historias; v3 limpia ambas; dirty/error omite rutas migradas y conserva proyectos. | `tests/unit/cmd/api/main_test.go` | Resolver y matriz de versiones/estados pasaron en tests unitarios y harness de proceso/PostgreSQL. | Completada |
| 3.6 | **RED — Integración Sprint:** creación 201 coincide con fila, proyecto ausente 404 y rechazo no escribe. | `tests/integration/sprint/postgres/http_integration_test.go` | Primer compile/setup falló estructuralmente antes de ejecutar aserciones; después pasó integración HTTP/PostgreSQL real. | Completada |
| 3.7 | **RED — Regresión/versionado:** escenarios v2/v3/dirty/error y disponibilidad de proyectos. | `tests/integration/story/postgres/http_integration_test.go` | Harness de proceso/PostgreSQL pasó para v1/v2/v3, dirty y lookup-error; no se reclama RED conductual por el setup inicial. | Completada |
| 3.8 | **GREEN — Handler:** JSON estricto, UUID de ruta, caso de uso y estados/contratos HTTP definidos. | `internal/sprint/transport/http/handler.go` | Tests unitarios pasaron para métodos, UUID, JSON, errores de repositorio y privacidad de respuesta. | Completada |
| 3.9 | **GREEN — Composición API:** dependencias de Sprint opcionales, ruta condicional y constructor existente compatible. | `internal/api/api.go` | Suite de composición pasó manteniendo rutas de proyecto/historia y registro condicional de Sprint. | Completada |
| 3.10 | **GREEN — Readiness:** una lectura de schema; historias desde v2 limpia y Sprint desde v3 limpia; proyectos disponibles ante dirty/error. | `cmd/api/main.go` | Unit e integración de proceso verificaron v1/v2/v3, dirty y error de lookup; suite completa pasó. | Completada |
| 3.11 | **REFACTOR — HTTP/composición:** refinar preservando todos los RED verdes y rutas previas. | Handler, composición, selección de deps y pruebas | `go test ./tests/unit/sprint/transport/http ./tests/unit/cmd/api` — exit 0; ambos paquetes pasaron. | Completada |
| 4.1 | **GREEN — Integración/regresión:** completar cableado real y validar persistencia, no asignación y compatibilidad. | Pruebas de integración Sprint y story/PostgreSQL | Integraciones Sprint/story con PostgreSQL 16 desechable pasaron; verificaron persistencia, rechazo y readiness. | Completada |
| 4.2 | **REFACTOR — Suite global:** comprobar contratos, errores, migraciones y aislamiento de pruebas. | Suite del repositorio | Worker y padre ejecutaron `DOCKER_HOST=unix:///run/user/1000/docker.sock go test ./...` — exit 0; todas las unidades e integraciones pasaron, sin omisiones. | Completada |
| 4.3 | **Documentación:** explicar migración externa, endpoint, request/response, errores y riesgo del down. | `README.md` | README actualizado y revisado; `git diff --check` independiente pasó. Se advierte que migration down elimina Sprints persistidos. | Completada |

## Progreso y siguiente paso

- **Progreso:** SDD apply y archive completos; 24/24 tareas SDD completadas, 3/3 unidades ODD cerradas y ninguna tarea de implementación pendiente. Cambio archivado en `openspec/changes/archive/2026-09-27-us-08-create-sprint/`; spec canónica en `openspec/specs/sprint/spec.md` (3 requisitos). Informe: `openspec/changes/archive/2026-09-27-us-08-create-sprint/archive-report.md`; no se solicitó un `verify-report.md` separado.
- **Evidencia de cierre de Unidad 1:** ciclo estricto RED → GREEN → REFACTOR documentado en `openspec/changes/archive/2026-09-27-us-08-create-sprint/apply-progress.md`; comandos enfocados del worker y verificación independiente terminaron con exit 0; runtime harness N/A justificado; límite de rollback registrado arriba; commit `e9dd5ff feat(sprint): add domain and creation use case`.
- **Evidencia de cierre de Unidad 2:** el primer intento test-first falló solo estructuralmente durante setup del paquete (repositorio aún inexistente); ningún cuerpo de test se ejecutó y no fue una aserción RED de comportamiento. Tras agregar migración y repositorio, el worker ejecutó `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 -v ./tests/integration/sprint/postgres -run 'TestSprintRepository'` (exit 0; ambos tests, 3.965s) y el padre verificó `DOCKER_HOST=unix:///run/user/1000/docker.sock go test -count=1 ./tests/integration/sprint/postgres -run 'TestSprintRepository'` (exit 0; paquete `ok`, 3.961s). Testcontainers usó PostgreSQL 16 desechable vía el servicio Docker rootless activo y socket `/run/user/1000/docker.sock`; Docker 29.8.1 e imagen cacheada `postgres:16-alpine` sin pull. Se probaron persistencia exacta, FK de proyecto inexistente y clasificación de clave primaria duplicada. No se tocó una base persistente del usuario ni se ejecutó down. Commit `ed8aac9 feat(sprint): persist sprints in PostgreSQL`.
- **Unidad 3:** `d616714 feat(sprint): expose sprint creation over HTTP`; 540 adiciones y 36 eliminaciones (576 líneas autoradas). Los primeros fallos test-first fueron estructurales y no ejecutaron aserciones de comportamiento; los tests unitarios, las integraciones PostgreSQL reales y la suite global sí pasaron.
- **Verificación final independiente:** `DOCKER_HOST=unix:///run/user/1000/docker.sock go test ./...` (exit 0; todas las unidades e integraciones pasaron, no omitidas); `git diff --check` pasó.
- **Tamaño/entrega:** conteo autorado real 1,161 (Unidad 1: 283; Unidad 2: 302; Unidad 3: 576), frente al pronóstico original de 550–750. Se conserva `exception-ok` y `size:exception` aceptado para una única PR prevista de `feat/us-08-create-sprint` → `main`, sin encadenamiento. No hay PR creada: los commits siguen locales y la entrega queda bajo la política ordinaria de push/PR a cargo del usuario. El pronóstico permanece identificado como tal y no se redujeron pruebas ni alcance.
- **Rollback y datos:** no ejecutar migration down automáticamente porque elimina Sprints persistidos. Solo se usaron contenedores PostgreSQL 16 desechables locales; no se tocó base persistente ni se ejecutó down.
- **Remoto:** los commits son locales; no hubo push, creación de PR ni otra operación remota.
- **Siguiente paso:** no hay trabajo local de implementación pendiente. La publicación permanece sin autorizar ni ejecutar; cualquier push o creación de PR queda bajo la política ordinaria de publicación a cargo del usuario.

## Artefactos finales OpenSpec

- Cambio archivado: `openspec/changes/archive/2026-09-27-us-08-create-sprint/`
- Especificación canónica Sprint: `openspec/specs/sprint/spec.md` (3 requisitos; creada porque no había una spec canónica previa).
- Informe de archivo: `openspec/changes/archive/2026-09-27-us-08-create-sprint/archive-report.md`.
- No se solicitó un `verify-report.md` separado. La suite completa y `git diff --check` fueron verificados independientemente durante apply.

## Localización y espejo

- **Archivo del repositorio:** `odd/tasks/us-08-create-sprint.md`
- **Engram:** proyecto `software-metrics-and-estimation`, observación #56, topic key `odd/us-08-create-sprint/tasks`; el espejo contiene este documento completo y este localizador relativo al repositorio.
