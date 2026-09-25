# Tareas: US-06 — Modificar una historia del Product Backlog

Fuente: `proposal.md`, `design.md` y `specs/historia/spec.md` de esta carpeta de cambio (issue #34).
TDD estricto activo; comando de prueba: `go test ./...`. Los identificadores, comandos, nombres de rama y
plantillas de mensaje de commit están en inglés (Conventional Commits); el resto del documento, en español.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~900-1300 de código/tests/SQL/docs (sin contar los documentos de planificación del slice 0) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Dominio) → PR 2 (Aplicación) → PR 3 (Almacenamiento) → PR 4 (Handler HTTP) → PR 5 (Composición y docs) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

La estrategia de cadena ya fue elegida por el usuario (`stacked-to-main`, la misma que usó US-05), por lo
que no queda ninguna decisión pendiente antes de aplicar. El pronóstico total excede el presupuesto de 400
líneas, pero cada slice queda dentro de él; no se recorta código, tests ni docs para ajustar cifras (regla
de `chained-pr`: el presupuesto condiciona el corte, no el código).

| Slice | Líneas estimadas (adiciones + eliminaciones) | Composición aproximada |
|-------|----------------------------------------------|------------------------|
| 0. Documentos de planificación | ~2000 de Markdown (fuera de la estimación de código) | `exploration.md`, `proposal.md`, `design.md`, `specs/historia/spec.md`, `tasks.md` |
| 1. Dominio | ~250 | ~70 código + ~180 tests |
| 2. Aplicación | ~200 | ~60 código + ~140 tests |
| 3. Almacenamiento | ~300-350 | ~60 código + ~20 SQL + ~250 tests de integración |
| 4. Handler HTTP | ~300 | ~90 código + ~200 tests + README (ejemplo de creación) |
| 5. Composición y docs | ~250 | ~40 código + ~130 tests + ~60 README |
| **Total** | **~1300** | |

### Suggested Work Units

Base de la cadena `stacked-to-main`: el PR 1 apunta a `main`; cada PR siguiente apunta a la rama del PR
anterior y se reorienta a `main` cuando el anterior se mergea. Si el diff de un PR hijo muestra cambios del
anterior, la base está mal y debe reorientarse o rebasarse antes de la revisión.

| Unit | Goal | Likely PR (rama) | Focused test command | Runtime harness | Rollback boundary |
|------|------|------------------|----------------------|-----------------|-------------------|
| 0+1 | Commit de planificación y dominio: `NewStoryUpdate`, conjunto de estados, `EstimatedHours`, helper compartido | PR 1: `feat/us06-domain-story-update` (base `main`) | `go test ./tests/unit/story/domain/...` | N/A: funciones puras de dominio sin frontera de proceso; la evidencia es la suite unitaria más `go test ./...` completo | Revertir el commit de dominio: `internal/story/domain/story.go` y `tests/unit/story/domain/story_test.go`. El commit de planificación se revierte por separado |
| 2 | Caso de uso `UpdateStoryUseCase`, puerto `StoryUpdater`, `ErrStoryNotFound` | PR 2: `feat/us06-application-update-story` (base PR 1) | `go test ./tests/unit/story/application/...` | N/A: caso de uso sobre fake; sin frontera de proceso ni de red | Revertir el commit: `internal/story/application/update_story.go`, `tests/unit/story/application/update_story_test.go` y la aserción aditiva de `create_story_test.go` |
| 3 | Migración `000003` y `PostgresStoryRepository.Update` con round-trip `NUMERIC` ↔ `*float64` | PR 3: `feat/us06-storage-estimated-hours` (base PR 2) | `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` (requiere Docker) | PostgreSQL real vía Testcontainers `postgres:16-alpine` con migraciones `000001`-`000003`; sin Docker el slice queda sin verificar | Revertir el commit: `000003_*.sql`, `Update` en `repository.go` y los tests de integración. Antes de aplicar `000003` en un entorno no hay datos que preservar; después, seguir el plan de rollback de esquema de `proposal.md` (el `down` es destructivo) |
| 4 | `UpdateStoryHandler`, helpers compartidos y `estimated_hours` en `storyResponse` | PR 4: `feat/us06-http-update-handler` (base PR 3) | `go test ./tests/unit/story/transport/http/...` | N/A: `httptest` con el caso de uso real sobre un updater falso; el proceso real se ejerce en el slice 5 | Revertir el commit: `update_handler.go`, los helpers de `handler.go`, sus tests y el ejemplo de creación del README. La ruta no está expuesta hasta el slice 5 |
| 5 | Registro de la ruta `PUT`, gate `version >= 3 && !dirty`, README y confirmación del delta de spec | PR 5: `feat/us06-wire-update-route-docs` (base PR 4) | `go test ./tests/unit/cmd/api/...` y `go test ./tests/integration/story/postgres/... -run "TestAPIStartup" -v` | Binario compilado por el propio test contra la base migrada (patrón de `http_integration_test.go`); requiere Docker para el arranque real | Revertir el commit (`api.go`, `main.go`, tests, README): quita la ruta y deja la creación intacta; el esquema puede quedar en versión 3 |

Diagrama de dependencias de la cadena (el PR actual se marca con `📍` en cada PR hijo):

```
main ← PR 1 dominio ← PR 2 aplicación ← PR 3 almacenamiento ← PR 4 handler HTTP ← PR 5 composición y docs
```

Orden de despliegue (de `design.md`, Migración y despliegue): los PR 1-4 no exponen la ruta, el sistema
sigue comportándose como US-05; luego se aplica la migración `000003` en el entorno; recién entonces se
mergea/despliega el PR 5.

## Convenciones de ejecución

- Cada slice es un work unit con **exactamente un commit** que lleva sus tests y su documentación. El
  commit de planificación (slice 0) es un commit `docs(sdd)` adicional en la primera rama.
- Ciclo TDD por slice: RED (escribir el test, ejecutarlo, **observar** la falla) → GREEN (implementación
  mínima, observar el pase) → TRIANGULATE (casos extra) → REFACTOR (cambio estructural, observar verde) →
  COMMIT. Un RED por error de compilación (símbolo indefinido) cuenta, pero debe registrarse como tal.
- No se fabrica evidencia. Un test de integración saltado por falta de Docker se registra como **saltado**
  y nunca como prueba de persistencia; el slice 3 (y la parte de arranque real del slice 5) no puede
  declararse verificado sin contenedor.
- Después de cada commit: `gofmt -l .` sin salida y `go vet ./...` limpios (parte de la verificación).
- Empujar ramas y abrir PR son decisiones del usuario bajo la política de repositorio; este documento no
  las automatiza. Cada PR hijo debe incluir la sección Chain Context con el diagrama de arriba, qué
  cubre, dependencias previas, trabajo posterior y lo que queda fuera de alcance.
- Plantilla del cuerpo del commit (el asunto es Conventional Commits; el cuerpo registra la evidencia
  observada, con nombres de tests y salida reales):

```text
<type>(<scope>): <outcome in imperative mood>

TDD evidence
- RED: <test names> -> observed failure: <exact failure or compile error>
- GREEN: <minimal implementation> -> observed: <go test command> PASS
- TRIANGULATE: <extra cases added> -> observed PASS
- REFACTOR: <structural change> -> observed: go test ./... PASS
- Full suite: go test ./... -> <observed result>; integration tests: <ran | SKIPPED (no Docker)>

Refs: #34
```

## Slice 0: Documentos de planificación (primera rama)

- [x] 0.1 Crear la rama `feat/us06-domain-story-update` desde `main` y registrar la línea base: `go test ./...` en verde antes de tocar nada (anotar si los tests de integración corrieron o se saltaron por falta de Docker). Verificación: `git branch --show-current` devuelve la rama nueva.
- [x] 0.2 Commit de los documentos de planificación existentes en la primera rama: `openspec/changes/us-06-modificar-historia-product-backlog/exploration.md`, `proposal.md`, `design.md`, `specs/historia/spec.md` y `tasks.md`. Mensaje: `docs(sdd): add US-06 planning artifacts for story modification`. Sin código de producto. Verificación: `git show --stat HEAD` lista solo archivos bajo `openspec/changes/us-06-modificar-historia-product-backlog/`. Rollback: revertir este commit no afecta al slice 1.

## Slice 1: Dominio (PR 1, rama `feat/us06-domain-story-update`, base `main`)

Requisitos de la spec: reutilizar las reglas de contenido de la creación; conjunto cerrado de estados sin
transiciones; estimación de horas opcional y positiva; texto y orden de criterios preservados.
Amenaza cubierta: identidad y datos hostiles no aplican aquí; la regla de decimales se decide con
`strconv.FormatFloat` (Decisión 7).

### RED

- [x] 1.1 RED contenido y estado: en `tests/unit/story/domain/story_test.go` agregar `TestNewStoryUpdate` (tabla con `t.Run`) con `title`/`description` vacíos y en blanco (ASCII y Unicode), cada prioridad válida e inválida (`urgente`, `ALTA`, `""`), cada estado del conjunto (`pendiente`, `en_progreso`, `completada`) y fuera del conjunto (`cancelada`, `Completada`, `en progreso`, `""`), criterios `nil`/`[]`/con entrada vacía o en blanco, con los mensajes de `fields` fijados en `design.md` (`must be alta, media or baja`, `must be pendiente, en_progreso or completada`, etc.). Ejecutar `go test ./tests/unit/story/domain/...` y observar la falla (símbolo `NewStoryUpdate` indefinido). Registrar el nombre de los tests y la falla.
- [x] 1.2 RED estimación: en `tests/unit/story/domain/story_test.go` agregar `TestNewStoryUpdateEstimatedHours` con `nil` (válido), `0`, negativos (`-1`, `-0.01`), `1.001`, `0.005`, `100000`, `99999.991`, con mensaje `must be greater than 0, at most 99999.99 and have at most 2 decimals` en `fields.estimated_hours`. Ejecutar y observar la falla.
- [x] 1.3 RED preservación y defaults: en `tests/unit/story/domain/story_test.go` agregar tests de que `NewStoryUpdate` conserva el texto (espacios internos y Unicode) y el orden de los criterios, devuelve `id`/`projectID` recibidos y `StoryPoints == nil`, y de que `NewStory` devuelve `EstimatedHours == nil` y `Status == StatusPending`; además `AllowedStatuses()` devuelve una copia (mutar el resultado no altera una segunda llamada). Ejecutar y observar la falla.

### GREEN

- [x] 1.4 GREEN: en `internal/story/domain/story.go` agregar `EstimatedHours *float64` a `Story`, las constantes `StatusPending`/`StatusInProgress`/`StatusCompleted`, `AllowedStatuses()`, `validateStatus`, `validateEstimatedHours` (decimales vía `strconv.FormatFloat(v, 'f', -1, 64)`, rango `> 0` y `<= 99999.99`) y `NewStoryUpdate`, con la implementación mínima que pase 1.1-1.3 (se tolera duplicar temporalmente los chequeos de contenido; se elimina en 1.6). Ejecutar `go test ./tests/unit/story/domain/...` y observar el pase.

### TRIANGULATE

- [x] 1.5 TRIANGULATE: en `tests/unit/story/domain/story_test.go` agregar los bordes `0.01` y `99999.99` (aceptados), el entero `3` y `2.5`, `0.07` (no debe dar falso rechazo por aritmética de floats), blancos Unicode (U+00A0 y U+2003), y la acumulación de varios campos inválidos (título vacío, descripción en blanco, prioridad `urgente`, estado `cancelada`, criterios `[]`, horas `0`) en un único `ValidationError`. Ejecutar y observar el pase; si algún caso falla, corregir `story.go` y repetir.

### REFACTOR

- [x] 1.6 REFACTOR: en `internal/story/domain/story.go` extraer `validateStoryContent(fields map[string]string, title, description, priority string, criteria []string)` desde `NewStory`, hacer que `NewStory` y `NewStoryUpdate` la invoquen y sustituir el literal `"pendiente"` por `StatusPending`. Sin cambio de comportamiento. Ejecutar `go test ./tests/unit/story/domain/...` **antes** (verde) y **después** (verde) del cambio; los 14 casos existentes de `NewStory` son la red. Ejecutar además `go test ./...` completo y registrar el resultado observado.

### Verificación y commit

- [x] 1.7 Verificar el slice: `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./tests/unit/story/domain/...` y `go test ./...` en verde (anotar los tests de integración como corridos o saltados).
- [x] 1.8 COMMIT del slice 1 (un único commit, tests incluidos): `feat(story): add domain validation for story updates`. El cuerpo sigue la plantilla de evidencia con los nombres reales de tests RED (`TestNewStoryUpdate`, `TestNewStoryUpdateEstimatedHours`), la falla observada, el pase GREEN, los casos triangulados, el REFACTOR (`validateStoryContent`, `StatusPending`) y el resultado observado de `go test ./...`. Rollback: revertir solo este commit.

## Slice 2: Aplicación (PR 2, rama `feat/us06-application-update-story`, base PR 1)

Requisitos de la spec: exigir los seis campos editables; identificadores inválidos y datos inválidos con
detalle de campos; una solicitud inválida no llega al almacenamiento; `404` por historia inexistente o de
otro proyecto (propagación de `ErrStoryNotFound`); `500` sin confundir con no encontrada.
Amenaza cubierta: identificadores de ruta como input hostil (`"abc"`, vacío, mayúsculas) — `422` con
`fields.story_id`/`fields.project_id` y normalización canónica.

### Preparación

- [ ] 2.1 Crear la rama `feat/us06-application-update-story` a partir de `feat/us06-domain-story-update` y confirmar `go test ./...` en verde como línea base.

### RED

- [ ] 2.2 RED comportamiento base: crear `tests/unit/story/application/update_story_test.go` con un fake de `StoryUpdater` escrito a mano (contador de llamadas y story capturada) y tests para: input inválido → `*domain.ValidationError` y `updater.calls == 0`; input válido → exactamente una llamada con `id`/`project_id` canónicos (`uuid.UUID.String()`, p. ej. UUID en mayúsculas normalizado) y los seis campos; `ErrStoryNotFound` y un error inesperado se propagan con `errors.Is`/igualdad y devuelven `domain.Story{}`; el caso de uso solo recibe el updater (no se genera ningún identificador). Ejecutar `go test ./tests/unit/story/application/...` y observar la falla (símbolos indefinidos: `NewUpdateStoryUseCase`, `UpdateStoryCommand`, `ErrStoryNotFound`).

### GREEN

- [ ] 2.3 GREEN: crear `internal/story/application/update_story.go` con `ErrStoryNotFound`, `UpdateStoryCommand` (incluye `MissingFields []string`), interfaz `StoryUpdater`, `UpdateStoryUseCase`, `NewUpdateStoryUseCase` y `Execute` (construye un único `map[string]string`, llama a `domain.NewStoryUpdate`, no invoca el puerto si hay errores, normaliza UUID y llama a `Update`). Implementación mínima que pase 2.2. Ejecutar `go test ./tests/unit/story/application/...` y observar el pase.

### TRIANGULATE

- [ ] 2.4 TRIANGULATE fusión de `fields` (un test por escenario de la spec) en `tests/unit/story/application/update_story_test.go`: `MissingFields` con las seis claves → seis entradas `is required`; una sola clave ausente → un solo campo; `story_id` inválido + `title` vacío → `story_id` y `title`; ambos identificadores inválidos → `project_id` y `story_id`; clave ausente cuyo valor cero también violaría el contenido → gana `is required` (no se sobrescribe); `estimated_hours` ausente frente a `nil` presente (solo `MissingFields` distingue). Siempre `updater.calls == 0` en los casos inválidos. Ejecutar y observar el pase; corregir `Execute` si algún caso falla.
- [ ] 2.5 TRIANGULATE regresión aditiva: en `tests/unit/story/application/create_story_test.go` agregar la aserción `EstimatedHours == nil` en la historia creada. Nota honesta: si el slice 1 ya dejó el campo en `nil`, este test pasa de inmediato y es una prueba de caracterización, no un RED; registrarlo así en el commit.

### REFACTOR

- [ ] 2.6 REFACTOR: unificar en `internal/story/application/update_story.go` la construcción del mapa de `fields` si los tests muestran duplicación (helper interno no exportado); si no hay duplicación real, registrar "REFACTOR: sin cambios necesarios" en lugar de inventar uno. Ejecutar `go test ./tests/unit/story/application/...` antes y después (verde) y `go test ./...` completo.

### Verificación y commit

- [ ] 2.7 Verificar el slice: `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde, y comprobar que `tests/unit/story/application/create_story_test.go`, `tests/unit/story/transport/http/handler_test.go` y `tests/unit/cmd/api/main_test.go` siguen compilando sin cambios (el puerto es separado de `StoryRepository`).
- [ ] 2.8 COMMIT del slice 2 (un único commit): `feat(story): add update story use case and port`. Cuerpo con la evidencia TDD real (RED por símbolos indefinidos, GREEN, casos triangulados de la fusión de `fields`, REFACTOR o "sin cambios", resultado de `go test ./...`). Rollback: revertir solo este commit.

## Slice 3: Almacenamiento (PR 3, rama `feat/us06-storage-estimated-hours`, base PR 2)

Requisitos de la spec: estimación opcional y positiva con round-trip exacto en `0.01` y `99999.99`;
`null` borra; estado cerrado también en almacenamiento (`CHECK`); `id`/`project_id`/`story_points`
intactos; `404` indistinguible entre inexistente y otro proyecto sin verificación previa; `500` sin
confundirse con no encontrada; la modificación es atómica.
Amenazas cubiertas: superficie de SQL (comillas, `;`, `--`), UUID de otro proyecto → no encontrada, fuga de
información (un error de base no se etiqueta como no encontrada).

### Preparación

- [ ] 3.1 Crear la rama `feat/us06-storage-estimated-hours` a partir de `feat/us06-application-update-story` y **confirmar que Docker está disponible** (`docker info`). Si no lo está, registrarlo y detenerse: este slice no puede declararse verificado ni hecho sin contenedor.

### RED (antes de escribir cualquier implementación; el primero resuelve el riesgo más alto)

- [ ] 3.2 RED migraciones: en `tests/integration/story/postgres/repository_integration_test.go` agregar `000003_add_story_estimated_hours.up.sql` a la lista de migraciones de `storyDatabase` (línea ~371). Ejecutar `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` y observar la falla (el archivo de migración no existe todavía). Sembrar `story_points` por SQL directo en un helper de test.
- [ ] 3.3 RED round-trip de la estimación (riesgo (a) codec `numeric` y (b) `NULL` → `**float64`, incluido `story_points`): `TestStoryRepositoryUpdateRoundTripsEstimatedHours` en `tests/integration/story/postgres/repository_integration_test.go` con los bordes `0.01` y `99999.99` y un entero, comprobando el valor devuelto por `RETURNING` y el leído por SQL. Observar la falla (símbolo `Update` indefinido).
- [ ] 3.4 RED actualización completa: en `tests/integration/story/postgres/repository_integration_test.go`, actualización de los seis campos persistida con el orden de criterios preservado y reemplazo completo (los criterios anteriores desaparecen).
- [ ] 3.5 RED `null` borra: `estimated_hours` previa `8` y actualización con `nil` deja SQL `NULL`, no `0`.
- [ ] 3.6 RED inmutabilidad: `project_id` sin cambios y `story_points` (sembrado por SQL) sin cambios tras una actualización válida; la respuesta de `Update` trae los valores almacenados.
- [ ] 3.7 RED no encontrada: proyecto equivocado → `application.ErrStoryNotFound` con la fila original íntegra (comparar antes y después); historia inexistente → `ErrStoryNotFound`; un error de base no relacionado (p. ej. contexto cancelado) no se etiqueta como `ErrStoryNotFound`.
- [ ] 3.8 RED constraints por nombre: consulta a `pg_constraint` (como el test existente del FK) verifica `stories_status_check` y `stories_estimated_hours_positive` con la definición esperada y que una escritura directa inválida (`status = 'cancelada'`, `estimated_hours = 0`) es rechazada.
- [ ] 3.9 RED SQL hostil: un título con comillas, `;` y `--` hace round-trip sin alterar la tabla ni otras filas.

Ejecutar `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` y registrar la falla observada de cada grupo.

### GREEN

- [ ] 3.10 GREEN migración: crear `internal/project/infrastructure/postgres/migrations/000003_add_story_estimated_hours.up.sql` (columna `estimated_hours NUMERIC(7,2) NULL` con `stories_estimated_hours_positive CHECK (estimated_hours > 0)` y `stories_status_check CHECK (status IN ('pendiente','en_progreso','completada'))`) y `internal/project/infrastructure/postgres/migrations/000003_add_story_estimated_hours.down.sql` (elimina el `CHECK` de estado y la columna; destructivo para las estimaciones).
- [ ] 3.11 GREEN `Update`: en `internal/story/infrastructure/postgres/repository.go` agregar `Update(ctx, story domain.Story) (domain.Story, error)` con una única sentencia parametrizada `UPDATE stories SET title,description,priority,status,acceptance_criteria,estimated_hours WHERE id = $1 AND project_id = $2 RETURNING <9 columnas>`; `pgx.ErrNoRows` → `application.ErrStoryNotFound`; ningún otro error se reinterpreta. Agregar la aserción de compilación de que `*PostgresStoryRepository` satisface `application.StoryRepository` y `application.StoryUpdater`. Ejecutar `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` y observar el pase.
- [ ] 3.12 Escalera de fallback (solo si 3.3 falla, en orden y registrando en el commit qué paso hizo falta y por qué): (1) parámetro `*float64` y `RETURNING` en `&story.EstimatedHours` (default de 3.11); (2) casteos `$8::numeric` y `RETURNING estimated_hours::double precision`, con intermediarios `pgtype.Numeric`/`sql.NullFloat64` (y `pgtype.Int4` para `story_points`); (3) cambiar la columna a `DOUBLE PRECISION NULL CHECK (estimated_hours > 0)` en el mismo `000003_add_story_estimated_hours.up.sql`, documentando la pérdida de la garantía de 2 decimales en la base. Si no se necesita, registrar "paso 1 suficiente".

### TRIANGULATE

- [ ] 3.13 TRIANGULATE: en `tests/integration/story/postgres/repository_integration_test.go` agregar la actualización idempotente (mismos valores dos veces → mismo resultado), dos actualizaciones consecutivas (gana la última), todos los estados del conjunto persistidos, `story_points` no nulo (`13`) preservado junto con horas `1.5`, y que las pruebas de creación existentes siguen pasando con `000003` aplicada. Ejecutar y observar el pase.

### REFACTOR

- [ ] 3.14 REFACTOR: extraer en `internal/story/infrastructure/postgres/repository.go` el escaneo de las 9 columnas a un helper interno solo si `Update` y el código existente lo justifican; de lo contrario registrar "sin cambios necesarios". Ejecutar `go test ./tests/integration/story/postgres/... -v` antes y después (verde) y `go test ./...` completo.

### Verificación y commit

- [ ] 3.15 Verificar el slice: `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde **con Docker** (los tests de integración deben mostrar `PASS`, no `SKIP`). Si algún test se saltó por falta de Docker, registrarlo como saltado y no declarar el slice verificado.
- [ ] 3.16 COMMIT del slice 3 (un único commit): `feat(story): persist story updates and estimated hours`. Cuerpo con la evidencia TDD real (RED por archivo de migración y `Update` inexistentes, GREEN, casos triangulados, paso de la escalera usado, REFACTOR o "sin cambios", `go test ./...` y estado de los tests de integración: corridos o **SKIPPED**). Rollback: revertir solo este commit; si `000003` ya fue aplicada en algún entorno, seguir el plan de rollback de esquema de `proposal.md` (exportar estimaciones, devolver deliberadamente a `pendiente` las filas `en_progreso`/`completada`, y solo entonces ejecutar el `down`).

## Slice 4: Handler HTTP (PR 4, rama `feat/us06-http-update-handler`, base PR 3)

Requisitos de la spec: `200` con la historia completa (nueve campos); `400 invalid_request` (JSON
malformado, cuerpo vacío, no objeto, varios valores, claves desconocidas incluidas `id`/`project_id`/
`story_points`, tipos incorrectos, `null` dentro de criterios); `422 validation_failed` con `fields`
(claves ausentes, UUID de ruta inválidos, contenido, estado, estimación); `404 story_not_found`;
`500 internal_error` genérico; `405` para métodos no admitidos; la respuesta de creación emite
`"estimated_hours": null`.
Amenazas cubiertas: identidad tomada del cuerpo, solapamiento de patrones de routing, fuga de información
(mismo código y cuerpo para inexistente y de otro proyecto; `500` sin detalle interno).

### Preparación

- [ ] 4.1 Crear la rama `feat/us06-http-update-handler` a partir de `feat/us06-storage-estimated-hours` y confirmar `go test ./...` en verde como línea base.

### RED

- [ ] 4.2 RED creación aditiva: en `tests/unit/story/transport/http/handler_test.go` actualizar las aserciones de respuesta de creación para incluir `"estimated_hours": null`. Ejecutar `go test ./tests/unit/story/transport/http/...` y observar la falla (la respuesta actual no emite la clave).
- [ ] 4.3 RED camino feliz y errores de mapa: crear `tests/unit/story/transport/http/update_handler_test.go` (tabla de escenarios con `httptest` y `SetPathValue("project_id"/"story_id")`, caso de uso **real** sobre un updater falso) con: `200` con los nueve campos incluidos `story_points` y `estimated_hours`; `404` (mismo código y cuerpo para inexistente y otro proyecto); `500` con mensaje genérico sin detalle interno; `405` para `GET`/`POST`/`PATCH`/`DELETE`. Ejecutar y observar la falla (`NewUpdateStoryHandler` indefinido).
- [ ] 4.4 RED `400 invalid_request`: en `tests/unit/story/transport/http/update_handler_test.go` casos para JSON malformado, cuerpo vacío, `[]`, `"texto"`, dos objetos consecutivos, clave desconocida `owner`, `id`/`project_id`/`story_points` en el cuerpo (con entero, `0` y `null`), `title` numérico, `acceptance_criteria: "criterio"`, `estimated_hours` como `"8"`/`true`/objeto, y `null` dentro de los criterios; siempre `updater.calls == 0`.
- [ ] 4.5 RED `422 validation_failed`: en `tests/unit/story/transport/http/update_handler_test.go` casos para UUID de ruta inválidos (`story_id` `"abc"`, `project_id` `"no-es-uuid"`, ambos), una clave obligatoria ausente (cada una de las seis, incluida `estimated_hours`), el cuerpo `{}` (seis `is required`), errores de dominio (contenido, estado, estimación) e identificador inválido más `title` vacío; siempre `updater.calls == 0`. Ejecutar y observar las fallas.

### GREEN

- [ ] 4.6 GREEN: crear `internal/story/transport/http/update_handler.go` con `NewUpdateStoryHandler`, `updateStoryRequest`, la detección de claves obligatorias ausentes (seis obligatorias menos las presentes en el mapa de claves crudas), la construcción de `application.UpdateStoryCommand` y el mapa de errores de la Decisión 12 (`errors.As` de `ValidationError` antes que `errors.Is` de `ErrStoryNotFound`); en `internal/story/transport/http/handler.go` agregar `EstimatedHours *float64` con `json:"estimated_hours"` a `storyResponse`. Se tolera una decodificación duplicada temporal (se unifica en 4.8). Ejecutar `go test ./tests/unit/story/transport/http/...` y observar el pase.

### TRIANGULATE

- [ ] 4.7 TRIANGULATE: en `tests/unit/story/transport/http/update_handler_test.go` agregar `estimated_hours` presente con `null` → `200` y `null` en la respuesta, frente a `estimated_hours` ausente → `422 is required`; `Content-Type: application/json` en todas las respuestas; UUID de ruta en mayúsculas normalizados; texto Unicode preservado en la respuesta; la respuesta sale de lo devuelto por el updater y no del comando. Ejecutar y observar el pase.

### REFACTOR

- [ ] 4.8 REFACTOR: en `internal/story/transport/http/handler.go` generalizar `decodeCreateStoryRequest` a `decodeStoryObject(body io.Reader, target any) (map[string]json.RawMessage, error)` (misma política: un solo objeto, `DisallowUnknownFields`, guarda contra `null` en criterios) e introducir `newStoryResponse`, usados por ambos handlers; `update_handler.go` deja de duplicar la decodificación. Ejecutar `go test ./tests/unit/story/transport/http/...` **antes y después** (toda la suite de creación en verde) y `go test ./...` completo.

### Documentación y commit

- [ ] 4.9 Documentación mínima que introduce este slice: en `README.md` actualizar el ejemplo de respuesta de creación (línea ~72) para que incluya `"estimated_hours":null`. La documentación de la operación de modificación va en el slice 5, cuando la ruta queda expuesta.
- [ ] 4.10 Verificar el slice: `gofmt -l .` sin salida, `go vet ./...` limpio y `go test ./...` en verde.
- [ ] 4.11 COMMIT del slice 4 (un único commit, tests y README incluidos): `feat(story): add update story http handler`. Cuerpo con la evidencia TDD real (RED por `NewUpdateStoryHandler` indefinido y respuesta de creación sin `estimated_hours`, GREEN, casos triangulados `null` vs ausente, REFACTOR `decodeStoryObject`/`newStoryResponse`, `go test ./...`). Rollback: revertir solo este commit; la ruta no está expuesta, no hay efecto en runtime más allá del campo aditivo de la creación.

## Slice 5: Composición y documentación (PR 5, rama `feat/us06-wire-update-route-docs`, base PR 4)

Requisitos de la spec: disponibilidad de la modificación condicionada a `version >= 3 && !dirty`;
la ruta no existe en versión 2 (`404`) y la creación sigue funcionando; `405` de la colección
preservado; documentación de la operación, del conjunto de `status`, de las reglas de la estimación y de
la limitación de última escritura gana.
Amenaza cubierta: solapamiento de patrones de routing (`/projects/{id}/stories` conserva `405`;
`/projects/{id}/stories/` responde `404`).

### Preparación

- [ ] 5.1 Crear la rama `feat/us06-wire-update-route-docs` a partir de `feat/us06-http-update-handler` y confirmar `go test ./...` en verde como línea base.

### RED

- [ ] 5.2 RED unitario de composición: en `tests/unit/cmd/api/main_test.go` agregar un fake de updater y tests: con `Updater` presente, `PUT /projects/{id}/stories/{story_id}` responde `200` y hay exactamente una actualización; con `Updater` nil, la misma ruta responde `404` y la creación sigue funcionando; se preservan los `405` de `GET`/`PUT`/`DELETE` sobre `/projects/{id}/stories`; `/projects/{id}/stories/` responde `404`; el caso solo-proyectos se conserva. Ejecutar `go test ./tests/unit/cmd/api/...` y observar la falla (`StoryDependencies.Updater` indefinido).
- [ ] 5.3 RED de integración (requiere Docker): en `tests/integration/story/postgres/http_integration_test.go` agregar el escenario `version three` a `TestAPIStartupRoutesFollowMigrationState` (modificación disponible), `version two` conserva la creación y la ruta de modificación no existe (`404`), `dirty` en versión 3 no expone la ruta, y de punta a punta crear → modificar `200` con `project_id` intacto, y una historia inexistente → `404`. Ejecutar `go test ./tests/integration/story/postgres/... -run "TestAPIStartup" -v` y observar la falla.

### GREEN

- [ ] 5.4 GREEN `api.go`: en `internal/api/api.go` agregar `Updater storyapplication.StoryUpdater` a `StoryDependencies` (campo aditivo y opcional) y registrar `PUT /projects/{project_id}/stories/{story_id}` con `NewUpdateStoryHandler` solo si `Updater != nil`. Ejecutar `go test ./tests/unit/cmd/api/...` y observar el pase.
- [ ] 5.5 GREEN `main.go`: en `cmd/api/main.go` resolver los tres casos a partir de una única lectura de `schema_migrations` (`err == nil && !dirty && version >= 3` → proyectos + crear + modificar; `>= 2` → proyectos + crear; resto → solo proyectos), cada uno con su log, inyectando el mismo `*PostgresStoryRepository` como `Repository` y como `Updater`. Ejecutar `go test ./tests/integration/story/postgres/... -run "TestAPIStartup" -v` y observar el pase.

### TRIANGULATE

- [ ] 5.6 TRIANGULATE: en `tests/unit/cmd/api/main_test.go` y `tests/integration/story/postgres/http_integration_test.go` agregar modificar dos veces seguidas (gana la última), `PATCH`/`DELETE`/`POST`/`GET` sobre la ruta del ítem responden `405` cuando la ruta existe, versión 3 con `dirty` sin ruta, y que el log de arranque distingue los tres casos. Ejecutar y observar el pase.

### Documentación

- [ ] 5.7 Documentar en `README.md` la operación `PUT /projects/{project_id}/stories/{story_id}` (los seis campos obligatorios, `estimated_hours` puede ser `null`), el conjunto de `status` (`pendiente`, `en_progreso`, `completada`, sin reglas de transición), las reglas de la estimación (número `> 0`, máximo 2 decimales, `<= 99999.99`, `null` la borra), la migración `000003` requerida y su condición de exposición (`version >= 3`), la lista de errores (`400`, `404`, `405`, `422`, `500`) y la limitación **última escritura gana** (sin versionado ni control de concurrencia).
- [ ] 5.8 Confirmar que el delta `openspec/changes/us-06-modificar-historia-product-backlog/specs/historia/spec.md` describe exactamente lo implementado en los slices 1-5 (mensajes de `fields`, códigos HTTP, gate de versión). Solo se corrige el delta si la implementación difiere de lo especificado; la aplicación del delta a `openspec/specs/historia/spec.md` ocurre al archivar, no aquí.

### REFACTOR

- [ ] 5.9 REFACTOR: revisar `cmd/api/main.go` (las tres ramas de esquema) y simplificar solo si hay duplicación real; de lo contrario registrar "sin cambios necesarios". Ejecutar `go test ./tests/unit/cmd/api/...` antes y después (verde) y `go test ./...` completo.

### Verificación y commit

- [ ] 5.10 Verificar el slice: `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde **con Docker** (los tests de arranque real deben mostrar `PASS`, no `SKIP`; si se saltaron, registrarlo y no declarar verificado el arranque real). Revisión manual del README renderizado.
- [ ] 5.11 COMMIT del slice 5 (un único commit, tests y README incluidos): `feat(api): expose story update route behind schema version 3`. Cuerpo con la evidencia TDD real (RED por `StoryDependencies.Updater` indefinido y escenarios de arranque, GREEN, triangulación, REFACTOR o "sin cambios", `go test ./...` y estado de la integración: corrida o **SKIPPED**). Rollback: revertir solo este commit quita la ruta y deja la creación intacta; el esquema puede permanecer en versión 3.

## Trazabilidad de requisitos de la spec a tareas

| Requisito de la spec (`historia`) | Tareas |
|-----------------------------------|--------|
| Modificar una historia existente de un proyecto existente | 2.2, 3.4, 3.6, 4.3, 5.2, 5.3, 5.6 |
| Exigir los seis campos editables | 2.4, 4.5, 4.7 |
| Reutilizar las reglas de validación de contenido | 1.1, 1.5, 1.6, 2.2, 4.5 |
| Restringir el estado a un conjunto cerrado sin transiciones | 1.1, 3.8, 3.13, 4.5 |
| Estimar las horas con un valor opcional y positivo | 1.2, 1.5, 3.3, 3.5, 3.8, 3.12, 4.7 |
| Preservar identidad y Story Points | 3.6, 3.13, 4.4 |
| Rechazar historia inexistente o de otro proyecto | 2.2, 3.7, 4.3, 5.3 |
| Identificadores inválidos y datos inválidos con detalle de campos | 2.4, 4.5 |
| Rechazar solicitudes malformadas | 4.4 |
| Rechazar métodos no admitidos | 4.3, 5.2, 5.6 |
| Fallar sin divulgar detalles ni modificar datos | 2.2, 3.7, 4.3 |
| Disponibilidad condicionada a la versión del esquema | 5.2, 5.3, 5.5, 5.6 |
| MODIFIED: inicializar la historia sin estimación | 1.3, 2.5, 4.2, 4.9 |

## Riesgos de ejecución

- El slice 3 y el arranque real del slice 5 dependen de Docker; sin él quedan sin verificar y no pueden declararse hechos.
- El slice 0 agrega ~2000 líneas de Markdown al PR 1. Mitigación: es un commit separado y el primero de la rama, así que se puede abrir como PR de solo documentación o revisar commit por commit.
- El round-trip de pgx `NUMERIC(7,2)` ↔ `*float64` no está verificado: es el primer RED del slice 3 y la tarea 3.12 define la escalera de fallback.
- La cadena `stacked-to-main` exige reorientar cada PR hijo a `main` tras mergear el anterior; un diff hijo con cambios del padre indica base incorrecta.
