# Tareas: Registrar integrantes en un proyecto (US-03)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 700–1.100 líneas (incluye reconciliación, tests y módulo vertical) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Single PR; user explicitly accepted `size:exception` for this US-03 change |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: size-exception
400-line budget risk: High

El riesgo se estima por la reconciliación histórica obligatoria, el nuevo módulo vertical, cambios de composición/readiness y pruebas unitarias e integración en varias áreas. Aunque se recomienda dividir la carga de revisión, el usuario autorizó explícitamente `size:exception` para mantener un solo PR en esta US-03; no se inicia encadenado.

## Secuencia de trabajo (Strict TDD)

**Alcance autorizado:** el trabajo local puede probar instalación limpia y ambas historias hipotéticas de `000003` usando exclusivamente un clúster PostgreSQL 18.6 desechable y golang-migrate v4.19.1 local. Esto no identifica ni prueba el runner, versión, checksum, ruta o historial de producción: el despliegue sigue prohibido hasta confirmar esos metadatos por separado. No usar entornos persistentes.

### 1. RED — fijar evidencia y pruebas de convergencia de migraciones
- [x] 1.1. Documentar el comando local de golang-migrate con forma README (`migrate -path <migrations> -database <disposable-postgres-url> up`) y usar golang-migrate v4.19.1 contra PostgreSQL 18.6 desechable para caracterizar instalación limpia y ambas historias hipotéticas `000003`, estructuras/constraints y efecto de `000004`. El comando es solo runner local de prueba: runner/proceso, versión, checksum, ruta y metadatos de producción siguen sin identificar; no afirmar compatibilidad productiva.
- [x] 1.2. Añadir primero pruebas/fixtures en `tests/integration/` para instalación limpia y cada historia hipotética de `000003`, ejecutadas con PostgreSQL 18.6 desechable y golang-migrate v4.19.1 local. Esta es la fase RED: capturar el fallo basal esperado de archivos duplicados y caracterizar las dos formas históricas del esquema. Incluir un fixture negativo con un `stories.status` incompatible con `stories_status_check`: se espera fallo cerrado antes de cambios persistentes, conservando status/esquema y readiness cerrados; este fixture no debe converger. No exigir convergencia de ese estado incompatible antes o después de implementar SQL. Estas pruebas no sustituyen la identificación del toolchain e historial de producción.
- [x] 1.3. **Gate de factibilidad pre-GREEN:** inspeccionar ambos estados iniciales y avanzar a SQL forward-only solo si se demuestra una estrategia concreta e idempotente para estados compatibles, sin pérdida de filas/valores. La rama de horas estimadas agrega `stories_status_check`; las historias exitosas deben usar estados válidos, sin relajar la validación ni reescribir statuses. Un valor legacy incompatible es un fixture negativo esperado: debe causar fallo cerrado antes de cambios persistentes de esquema/datos, conservar status/esquema y dejar readiness/ruta cerrados; no se espera que converja. Verificar localmente la transacción/atomicidad del runner y el estado dirty/rollback antes de confiar en esta semántica. Si la estrategia compatible no se demuestra, detenerse antes de SQL o habilitar la ruta. Después de implementar SQL, exigir convergencia de instalación limpia y ambas historias con estados válidos antes de habilitar la ruta; fallos bloquean ruta y despliegue. Runner e historial productivos siguen sin verificar y bloquean despliegue hasta confirmarlos; nunca adivinar, reescribir historial ni reparar destructivamente.

### 2. GREEN — reconciliar migraciones y habilitar el esquema de integrantes
- [x] 2.1. Solo tras superar 1.3, implementar en `internal/project/infrastructure/postgres/migrations/` una serie única append-only: preservar 001, 002 y 004; retener una identidad `000003`, reasignar el otro cambio a una versión posterior única e idempotente, y asignar después una versión distinta a `project_members`. Conservar los SQL up/down originales en operación; la transición forward debe detectar/crear solo objetos ausentes y fallar ante estructura parcial/incompatible, sin alterar filas. No fijar números antes de verificar la secuencia del runner.
- [x] 2.2. Actualizar `tests/integration/story/postgres/repository_integration_test.go` y `tests/integration/sprint/postgres/repository_integration_test.go` para aplicar la cadena canónica completa en lugar de listas manuales divergentes; añadir verificaciones de instalación limpia y convergencia de ambas historias 000003 con estados válidos y conteos/valores intactos. Cubrir por separado el fixture de status incompatible como fallo cerrado sin cambios persistentes, sin exigir que converja. Ejecutar `go test ./...`.

### 3. TRIANGULATE — comprobar variaciones históricas, persistencia y seguridad del gating
- [x] 3.1. En `tests/integration/` y los fixtures de migración, probar estructuras completas de ambas variantes, versión registrada, objetos ya existentes, ausencia de objetos opcionales y estructura incompatible/parcial; comprobar idempotencia, preservación de datos y fallo cerrado. Para estados válidos, exigir convergencia. Para `stories.status` incompatible con `stories_status_check`, exigir fallo cerrado antes de cambios persistentes, status/esquema original intactos y readiness cerrada; no exigir convergencia ni reescribir el valor. Si cualquier estado real no es reconciliable de forma segura, activar el bloqueo de 1.3 y no habilitar members.
- [x] 3.2. Ejecutar y contrastar las pruebas locales con golang-migrate v4.19.1 y PostgreSQL 18.6 desechable; anotar explícitamente que no prueban el runner/checksum/historial de producción, que siguen sin confirmar y bloquean rollout. Ejecutar `go test ./...` con Docker disponible; anotar cualquier prueba no ejecutable.

### 4. RED — contrato y comportamiento de dominio/API de integrantes
- [x] 4.1. Crear pruebas fallidas en `tests/unit/projectmember/` para lote vacío, nombre vacío/blanco, email opcional e inválido, igualdad exacta de `(full_name,email)` incluyendo email nulo y duplicados intralote; comprobar que el puerto no se llama ante lote inválido.
- [x] 4.2. Crear pruebas fallidas del handler/ruta en `tests/unit/projectmember/` y `tests/unit/cmd/api/main_test.go` para `POST /projects/{project_id}/members`, UUID/JSON inválidos, JSON adicional/trailing data, 201, 400, 404, 409 y 500; cubrir readiness ausente, dirty, error y head limpio sin habilitar la ruta prematuramente. Ejecutar `go test ./...` y registrar fallos iniciales.

### 5. GREEN — implementar dominio, caso de uso y transporte
- [x] 5.1. Implementar en `internal/projectmember/domain/` y `internal/projectmember/application/` el modelo separado, errores tipados y validación completa previa a persistencia: uno o más miembros, nombre requerido, email opcional, duplicados exactos intralote y rechazo integral.
- [x] 5.2. Implementar handler/DTO/mapping en `internal/projectmember/transport/http/` y ruta en `internal/api/api.go`: decodificación estricta, UUID, códigos/errores estables, éxito 201 con miembros creados y sin exponer detalles SQL. Mantener fuera de alcance cualquier listado.
- [x] 5.3. Implementar pruebas unitarias correspondientes en `tests/unit/projectmember/` y actualizar `tests/unit/cmd/api/main_test.go`; verificar que rechazos no llaman repositorio y errores mantienen el contrato especificado. Ejecutar `go test ./...`.

### 6. RED — persistencia transaccional, integración y composición
- [x] 6.1. Añadir pruebas de integración fallidas bajo `tests/integration/projectmember/postgres/` para asociación/FK, proyecto ausente, email NULL, duplicados existentes, rollback ante fallo, conflicto concurrente y conservación de campos de `projects`; preparar fallo inyectable para inserción/commit conforme a patrones existentes. Ejecutar `go test ./...`.
- [x] 6.2. Añadir/actualizar pruebas en `tests/unit/cmd/api/main_test.go`, `tests/integration/story/postgres/http_integration_test.go` y `tests/integration/sprint/postgres/http_integration_test.go` para gates por versiones finales, dirty/error y compatibilidad de funciones story/sprint en instalaciones convergidas. Ejecutar `go test ./...`.

### 7. GREEN — repositorio transaccional y readiness
- [x] 7.1. Implementar puerto y adaptador en `internal/projectmember/application/` y `internal/projectmember/infrastructure/postgres/`: existencia del proyecto dentro de transacción, conflicto persistido, inserts atómicos, rollback diferido, commit como único punto de publicación y mapeo tipado de unique conflict; la restricción debe tratar NULL como igual conforme a la versión PostgreSQL soportada.
- [x] 7.2. Componer dependencias y readiness en `internal/api/api.go` y `cmd/api/main.go` según la versión final limpia de members; ruta cerrada ante lookup fallido/dirty y sin cambiar disponibilidad de project, story o sprint. Actualizar comentarios y gates antiguos que dependan incorrectamente de `version >= 3`.
- [x] 7.3. Ejecutar pruebas de `tests/integration/projectmember/postgres/` y regresión story/sprint; confirmar 404 sin escrituras, rollback total, protección de concurrencia y que el proyecto permanece intacto. Ejecutar `go test ./...`.

### 8. TRIANGULATE — verificación cruzada del corte vertical
- [x] 8.1. Ampliar pruebas unitarias en `tests/unit/projectmember/` para límites de validación/errores y verificar contrato contra `openspec/changes/us-03-register-project-members/specs/project-members/spec.md` y `specs/project/spec.md`.
- [x] 8.2. Ejecutar la cadena de migraciones desde cero y desde ambas variantes históricas en `tests/integration/`, más `go test ./...`; inspeccionar que no haya escrituras parciales, pérdida de datos, ruta habilitada sobre esquema dirty/ausente ni alteración de datos básicos de proyectos. Documentar resultados y límites de Docker.

### 9. REFACTOR — consolidar sin cambiar comportamiento
- [x] 9.1. Refactorizar solo después de triangulación: reducir duplicación en fixtures/ayudantes de `tests/integration/{story,sprint,projectmember}/postgres/`, aislar mapeo de errores y revisar legibilidad de `internal/projectmember/` y `cmd/api/main.go`; preservar contrato y secuencia de migración.
- [x] 9.2. Ejecutar `go test ./...` tras el refactor y verificar nuevamente que los down no se presenten como rollback seguro de producción ni borren datos históricos; cualquier imposibilidad de convergencia mantiene el bloqueo y requiere decisión operativa antes de despliegue.
