# Tareas: US-07 — Consultar el Product Backlog

Fuente: `proposal.md`, `design.md` y `specs/historia/spec.md` de esta carpeta de cambio (issue #35).
TDD estricto activo; comando de prueba: `go test ./...`. Los identificadores, comandos, nombres de rama y
plantillas de mensaje de commit están en inglés (Conventional Commits); el resto del documento, en español.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1080 (rango 900–1200): ~265 producción, ~775 tests, ~40 docs |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | Un único PR con 5 commits de unidad de trabajo (1 dominio → 2 aplicación → 3 almacenamiento → 4 handler → 5 composición y docs); si el usuario no acepta la excepción, cortar en PR 1 (unidades 1–2) → PR 2 (unidad 3) → PR 3 (unidades 4–5) |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: size-exception
400-line budget risk: High

**Nota sobre la estrategia elegida.** La estrategia de entrega es `single-pr`: el usuario pidió
explícitamente un único pull request al final, sin dividirlo. El pronóstico (~1080 líneas) supera de forma
clara el presupuesto de revisión de 400 líneas, por lo que la aplicación **requiere una excepción
`size:exception` explícita del usuario antes de `sdd-apply`**. Este documento **no** la da por aceptada: la
tarea 0.1 es una compuerta bloqueante y ninguna tarea posterior puede iniciarse hasta que el usuario la
confirme. Los cinco commits autocontenidos mantienen el PR revisable commit por commit. No se recorta
código, tests ni docs para ajustar cifras (regla de `chained-pr`: el presupuesto condiciona el corte, no el
código).

| Unidad | Líneas estimadas (adiciones + eliminaciones) | Composición aproximada | Docker |
|--------|----------------------------------------------|------------------------|--------|
| 0. Documentos de planificación | ~2000 de Markdown (fuera de la estimación de código) | `exploration.md`, `proposal.md`, `design.md`, `specs/historia/spec.md`, `tasks.md` | No |
| 1. Dominio | ~150 | ~60 código + ~90 tests | No |
| 2. Aplicación | ~170 | ~50 código + ~120 tests | No |
| 3. Almacenamiento y migración `000004` | ~275 | ~64 código/SQL + ~210 tests de integración | **Sí** |
| 4. Handler HTTP | ~225 | ~65 código + ~160 tests | No |
| 5. Composición, arranque y README | ~260 | ~24 código + ~195 tests + ~40 README | **Sí** (arranque real) |
| **Total** | **~1080** | | |

### Suggested Work Units

Estrategia `single-pr`: todas las unidades viven en la rama `feat/us07-consultar-product-backlog` y se
entregan en un único PR hacia `main`. La columna "Likely PR" indica el commit dentro de ese PR; si el
usuario rechaza la excepción, el corte sugerido está en la tabla del Forecast.

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 0 | Compuerta `size:exception`, línea base y commit de los documentos de planificación | PR único, commit `docs(sdd)` | `go test ./...` (línea base) | N/A: solo documentación y verificación de línea base; sin código de producto | Revertir el commit de planificación; no afecta a ninguna unidad de código |
| 1 | Dominio: `Backlog`, `NewBacklog`, constantes de prioridad y `AllowedPriorities()` | PR único, commit 1 | `go test ./tests/unit/story/domain/...` | N/A: funciones puras sin frontera de proceso; la evidencia es la suite unitaria y `go test ./...` | Revertir el commit: `internal/story/domain/story.go`, `internal/story/domain/backlog.go` y `tests/unit/story/domain/backlog_test.go` |
| 2 | Aplicación: puerto `StoryLister`, `ListStoriesUseCase`, UUID validado antes del puerto | PR único, commit 2 | `go test ./tests/unit/story/application/...` | N/A: caso de uso sobre fake, sin frontera de proceso ni de red | Revertir el commit: `internal/story/application/list_stories.go` y `tests/unit/story/application/list_stories_test.go` |
| 3 | Migración `000004` (`seq`) y `PostgresStoryRepository.ListByProject` con `LEFT JOIN` atómico (**requiere Docker**) | PR único, commit 3 | `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` | PostgreSQL real vía Testcontainers `postgres:16-alpine` con migraciones `000001`–`000004`; sin Docker la unidad queda sin verificar | Revertir el commit: migraciones `000004`, `ListByProject` y sus tests. Si `000004` ya se aplicó en un entorno, ejecutar el `down` (solo pierde la secuencia de desempate; ningún dato de negocio) |
| 4 | Handler HTTP `ListStoriesHandler` con contenedor y `stories: []` | PR único, commit 4 | `go test ./tests/unit/story/transport/http/...` | N/A: `httptest` con el caso de uso real sobre un lister falso; el proceso real se ejerce en la unidad 5 | Revertir el commit: `list_handler.go` y su test; la ruta no está expuesta hasta la unidad 5 |
| 5 | Composición (`Lister`), gate `version >= 4 && !dirty`, integración de arranque y README (**requiere Docker**) | PR único, commit 5 | `go test ./tests/unit/cmd/api/...` y `go test ./tests/integration/story/postgres/... -run "TestAPIStartup" -v` | Binario compilado por el propio test contra la base migrada (patrón de `http_integration_test.go`); requiere Docker | Revertir el commit (`api.go`, `main.go`, tests, README): quita la ruta y deja creación y modificación intactas; el esquema puede quedar en versión 4 |

Orden de despliegue (de `design.md`, Migración y despliegue): mergear el PR; aplicar `000004` en el entorno;
reiniciar el binario. Sin `000004` la ruta simplemente no se registra.

## Convenciones de ejecución

- Cada unidad es un work unit con **exactamente un commit** que lleva sus tests y su documentación. El
  commit de planificación (unidad 0) es un commit `docs(sdd)` adicional al inicio de la rama.
- Ciclo TDD por unidad: RED (escribir el test, ejecutarlo y **observar** la falla) → GREEN (implementación
  mínima, observar el pase) → TRIANGULATE (casos extra) → REFACTOR (cambio estructural, observar verde) →
  COMMIT. Un RED por error de compilación (símbolo indefinido) cuenta, pero debe registrarse como tal.
- No se fabrica evidencia. Un test de integración saltado por falta de Docker se registra como **saltado**
  y nunca como prueba; las unidades 3 y 5 no pueden declararse verificadas sin contenedor.
- Después de cada commit: `gofmt -l .` sin salida y `go vet ./...` limpios (parte de la verificación).
- Empujar la rama y abrir el PR son decisiones del usuario bajo la política de repositorio; este documento no
  las automatiza. El PR debe declarar la excepción `size:exception` aceptada por el usuario.
- El detalle largo de la evidencia TDD (nombres completos de tests, fallas observadas, pasos de fallback
  usados) se registra en `openspec/changes/us-07-consultar-product-backlog/apply-progress.md`, no en los
  commits.

### Regla de commits (obligatoria)

Cada commit registra el ciclo TDD, pero con cuerpo **corto**: de **5 a 8 líneas como máximo**.

- Una línea por paso (RED, GREEN, REFACTOR) con, a lo sumo, uno o dos nombres de test clave y el resultado
  observado. Sin enumerar todos los tests y sin pegar salida de comandos.
- Si hay un límite de entorno (por ejemplo, integración no verificada por Docker apagado), se registra en
  **una sola línea**.
- El detalle largo va a `apply-progress.md`.
- Sin líneas de co-autoría de IA (`Co-Authored-By` ni atribución similar). Solo Conventional Commits en inglés.

Plantilla del cuerpo (el asunto es Conventional Commits; el cuerpo tiene 5–8 líneas):

```text
<type>(<scope>): <outcome in imperative mood>

TDD trace
- RED: <one or two key test names> -> <observed failure>
- GREEN: <minimal implementation> -> <observed result>
- REFACTOR: <structural change or "none needed"> -> <observed result>
- Env: <one line only if a limit applies, e.g. integration NOT verified (Docker off)>

Refs: #35
```

## Unidad 0: Compuerta de excepción y planificación (rama `feat/us07-consultar-product-backlog`)

- [x] 0.1 **COMPUERTA BLOQUEANTE.** Obtener la confirmación explícita del usuario de la excepción `size:exception` para el único PR (~1080 líneas frente al presupuesto de 400). No asumirla aceptada. Si el usuario no la acepta, detenerse y volver a la fase de tareas para dividir el trabajo según el corte sugerido del Forecast. Verificación: la confirmación queda registrada en `openspec/changes/us-07-consultar-product-backlog/apply-progress.md`.
- [x] 0.2 Confirmar que la rama actual es `feat/us07-consultar-product-backlog` (`git branch --show-current`) y registrar la línea base: `go test ./...` en verde antes de tocar nada, anotando si los tests de integración corrieron o se saltaron por falta de Docker (`docker info`). Resultado en `apply-progress.md`.
- [x] 0.3 Commit de los documentos de planificación existentes: los archivos bajo `openspec/changes/us-07-consultar-product-backlog/` (`exploration.md`, `proposal.md`, `design.md`, `specs/historia/spec.md` y `tasks.md`). Mensaje: `docs(sdd): add US-07 planning artifacts for backlog query`, con un cuerpo de 1–2 líneas (sin ciclo TDD: no hay código). Verificación: `git show --stat HEAD` lista solo archivos bajo esa carpeta.

## Unidad 1: Dominio — precedencia de prioridad y `Backlog` (commit 1, sin Docker)

Requisitos de la spec: "Ordenar el backlog por prioridad y, a igual prioridad, por orden de creación"
(escenarios: prioridad alta/media/baja, desempate S1…S5, misma prioridad, estabilidad, reubicación por
prioridad). La precedencia vive en un solo lugar: `AllowedPriorities()`.

### RED

- [x] 1.1 RED precedencia: crear `tests/unit/story/domain/backlog_test.go` con `TestNewBacklogOrdersByPriority` (tabla con `t.Run`): entrada `baja, alta, media` → `alta, media, baja`; escenario de la spec S1 `media`, S2 `alta`, S3 `media`, S4 `baja`, S5 `alta` → S2, S5, S1, S3, S4. Ejecutar `go test ./tests/unit/story/domain/...` y observar la falla (símbolos `NewBacklog` y `domain.Backlog` indefinidos). Registrar el nombre del test y la falla.
- [x] 1.2 RED estabilidad y contrato: en `tests/unit/story/domain/backlog_test.go` agregar `TestNewBacklogIsStableForEqualPriorities` (tres `media` en orden de entrada no alfabético conservan ese orden; `id` que ordenan al revés respecto de la entrada no alteran el resultado), `TestNewBacklogDoesNotMutateInput` (la entrada conserva su orden original), `TestNewBacklogPreservesProjectID`, `TestNewBacklogHandlesNilAndEmpty` y `TestAllowedPrioritiesReturnsOrderedCopy` (mutar el resultado no altera una segunda llamada; orden `alta`, `media`, `baja`). Ejecutar y observar la falla.

### GREEN

- [x] 1.3 GREEN: en `internal/story/domain/story.go` agregar las constantes `PriorityHigh`/`PriorityMedium`/`PriorityLow` y `AllowedPriorities()` (copia, de mayor a menor precedencia); crear `internal/story/domain/backlog.go` con `Backlog{ProjectID, Stories}`, `NewBacklog` (copia + `slices.SortStableFunc` por rango) y `priorityRank` no exportada (prioridad desconocida → `len(AllowedPriorities())`). Implementación mínima que pase 1.1 y 1.2. Ejecutar `go test ./tests/unit/story/domain/...` y observar el pase.

### TRIANGULATE

- [x] 1.4 TRIANGULATE: en `tests/unit/story/domain/backlog_test.go` agregar los escenarios de la spec "todas las historias con la misma prioridad conservan el orden de creación" (tres `media`), "cambiar la prioridad reubica la historia" (S1 `media`, S2 `alta`, S3 `alta` → S2, S3, S1), una prioridad desconocida que queda al final conservando su orden de entrada, y una entrada con una sola historia. Ejecutar y observar el pase; si algún caso falla, corregir `backlog.go` y repetir.

### REFACTOR

- [x] 1.5 REFACTOR: en `internal/story/domain/story.go` hacer que `validateStoryContent` valide la prioridad contra `AllowedPriorities()` en lugar de comparar literales, con el mismo mensaje `must be alta, media or baja`. Sin cambio de comportamiento. Ejecutar `go test ./tests/unit/story/domain/...` **antes** (verde) y **después** (verde): la suite existente de `NewStory`/`NewStoryUpdate` es la red de seguridad. Ejecutar además `go test ./...` completo y registrar el resultado observado.

### Verificación y commit

- [x] 1.6 Verificar la unidad: `gofmt -l .` sin salida, `go vet ./...` limpio y `go test ./...` en verde (anotar los tests de integración como corridos o saltados). Volcar el detalle de la evidencia en `apply-progress.md`.
- [x] 1.7 COMMIT de la unidad 1 (un único commit, tests incluidos): `feat(story): order the product backlog by priority`. Cuerpo de 5–8 líneas según la Regla de commits. Rollback: revertir solo este commit.

## Unidad 2: Aplicación — `ListStoriesUseCase` (commit 2, sin Docker)

Requisitos de la spec: "Rechazar un identificador de proyecto inválido en la consulta" (`422` sin tocar el
almacenamiento); "Rechazar la consulta de un proyecto inexistente" (propagación de `ErrProjectNotFound`);
"Fallar sin divulgar detalles internos" (sin lista parcial); "La consulta MUST NOT modificar" (fake de solo
lectura).
Amenaza cubierta: identificador de ruta como input hostil (`"abc"`, `"123"`, UUID truncado, vacío,
mayúsculas) y solo lectura.

### RED

- [x] 2.1 RED comportamiento base: crear `tests/unit/story/application/list_stories_test.go` con un fake de `StoryLister` escrito a mano (contador de llamadas y `projectID` capturado; expone únicamente lectura). Tests: `project_id` inválido → `*domain.ValidationError` con `fields.project_id` y `lister.calls == 0`; UUID válido → exactamente una llamada y `domain.Backlog` ordenado a partir de un fake en orden de creación. Ejecutar `go test ./tests/unit/story/application/...` y observar la falla (símbolos `NewListStoriesUseCase`, `ListStoriesQuery` y `StoryLister` indefinidos).

### GREEN

- [x] 2.2 GREEN: crear `internal/story/application/list_stories.go` con `ListStoriesQuery`, la interfaz `StoryLister`, `ListStoriesUseCase`, `NewListStoriesUseCase` y `Execute` (`uuid.Parse` antes del puerto con el error `must be a valid UUID`; llamada con el UUID canónico; error del puerto propagado con `domain.Backlog{}`; `domain.NewBacklog` sobre el resultado). Reutilizar `ErrProjectNotFound` de `internal/story/application/create_story.go` (sin modificarlo). Implementación mínima que pase 2.1. Ejecutar `go test ./tests/unit/story/application/...` y observar el pase.

### TRIANGULATE

- [x] 2.3 TRIANGULATE identificadores hostiles: en `tests/unit/story/application/list_stories_test.go` tabla de `project_id` inválidos (`"no-es-uuid"`, `"abc"`, `"123"`, UUID truncado, `""`), todos con `lister.calls == 0`; UUID en mayúsculas → una llamada con el UUID canónico en minúsculas y `Backlog.ProjectID` canónico.
- [x] 2.4 TRIANGULATE errores: `ErrProjectNotFound` se propaga con `errors.Is` y devuelve `domain.Backlog{}`; un error inesperado se propaga tal cual y no se confunde con `ErrProjectNotFound`; un fake que devuelve historias **junto con** un error produce igualmente `domain.Backlog{}` (sin lista parcial); un proyecto sin historias (fake devuelve `nil`) no es un error y produce un `Backlog` sin historias. Ejecutar y observar el pase; corregir `Execute` si algún caso falla.

### REFACTOR

- [x] 2.5 REFACTOR: solo si los tests muestran duplicación real en `internal/story/application/list_stories.go`; si no la hay, registrar "REFACTOR: sin cambios necesarios" en lugar de inventar uno. Ejecutar `go test ./tests/unit/story/application/...` antes y después (verde) y `go test ./...` completo.

### Verificación y commit

- [x] 2.6 Verificar la unidad: `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde, y comprobar que `tests/unit/story/application/create_story_test.go`, `tests/unit/story/transport/http/handler_test.go` y `tests/unit/cmd/api/main_test.go` siguen compilando sin cambios (el puerto es separado de `StoryRepository`).
- [x] 2.7 COMMIT de la unidad 2 (un único commit): `feat(story): add the list stories use case`. Cuerpo de 5–8 líneas según la Regla de commits. Rollback: revertir solo este commit.

## Unidad 3: Almacenamiento y migración `000004` (commit 3, **requiere Docker**)

Requisitos de la spec: orden de creación determinista (`seq`) que la modificación no altera; "Informar la
ausencia de historias con una lista vacía" y "Rechazar la consulta de un proyecto inexistente" resueltos en
una única consulta atómica; solo historias del proyecto consultado; consulta de solo lectura; errores de
almacenamiento sin reinterpretar.
Amenazas cubiertas: exposición de datos de otros proyectos, solo lectura (instantánea con `xmin`), fuga de
información (un error de base no se etiqueta como no encontrado) e identificador tratado solo como parámetro
posicional `$1`.

### Preparación

- [x] 3.1 **Confirmar que Docker está disponible** (`docker info`). Si no lo está, registrarlo en `apply-progress.md` y detenerse: esta unidad no puede declararse verificada ni hecha sin contenedor.

### RED (antes de escribir cualquier implementación; el primero resuelve el único riesgo técnico nuevo)

- [x] 3.2 RED migraciones: en `tests/integration/story/postgres/repository_integration_test.go` agregar `000004_add_story_creation_sequence.up.sql` a la lista de migraciones de `storyDatabase` (línea ~758). Ejecutar `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` y observar la falla (el archivo de migración no existe todavía).
- [x] 3.3 RED proyecto existente sin historias (riesgo técnico: escaneo de `NULL` del `LEFT JOIN` en destinos anulables): `TestStoryRepositoryListByProjectReturnsEmptyForProjectWithoutStories` en `tests/integration/story/postgres/repository_integration_test.go` → slice vacío y `nil` (no `ErrProjectNotFound`). Observar la falla (símbolo `ListByProject` indefinido).
- [x] 3.4 RED proyecto inexistente: UUID válido sin proyecto → `application.ErrProjectNotFound` y ninguna historia devuelta.
- [x] 3.5 RED orden de creación: historias sembradas con prioridades `baja`, `alta`, `media` e `id` que ordenan alfabéticamente al revés de la siembra → devueltas en orden de siembra (orden de creación, no de prioridad ni de `id`).
- [x] 3.6 RED aislamiento entre proyectos (amenaza: datos de otros proyectos): con dos proyectos A y B poblados, la consulta de A devuelve solo las historias de A; con A vacío y B con dos historias, la consulta de A devuelve una lista vacía.
- [x] 3.7 RED mapeo completo: `story_points` `8`, `estimated_hours` `8.50`, estado `en_progreso` y criterios en orden almacenado; y `NULL` → `nil` en ambos anulables (`story_points` y `estimated_hours`).
- [x] 3.8 RED la modificación no altera `seq`: tras `Update` de una historia (cambio de título, estado y prioridad), su posición de creación y el orden devuelto por `ListByProject` se conservan.
- [x] 3.9 RED solo lectura (amenaza: solo lectura): instantánea textual de `stories` y `projects` con `xmin` y `to_jsonb` (consultas del diseño, "Instantánea de solo lectura") idéntica byte a byte antes y después de consultar un proyecto con historias, uno vacío y uno inexistente, varias veces.
- [x] 3.10 RED esquema por catálogo: `stories_project_id_seq_key` existe con la definición esperada (`pg_constraint`); `seq` es identidad `ALWAYS` (`information_schema.columns`); un `INSERT` con `seq` explícito falla con SQLSTATE `428C9`.
- [x] 3.11 RED error no reinterpretado (amenaza: fuga de información): con la columna `seq` renombrada, `ListByProject` falla, el error no es `ErrProjectNotFound` y no devuelve historias.

Ejecutar `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` y registrar la falla observada de cada grupo.

### GREEN

- [x] 3.12 GREEN migración: crear `internal/project/infrastructure/postgres/migrations/000004_add_story_creation_sequence.up.sql` (`ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY` y `stories_project_id_seq_key UNIQUE (project_id, seq)`) y `internal/project/infrastructure/postgres/migrations/000004_add_story_creation_sequence.down.sql` (elimina la constraint y luego la columna).
- [x] 3.13 GREEN `ListByProject`: en `internal/story/infrastructure/postgres/repository.go` agregar `ListByProject(ctx, projectID string) ([]domain.Story, error)` con la única sentencia `SELECT ... FROM projects p LEFT JOIN stories s ON s.project_id = p.id WHERE p.id = $1 ORDER BY s.seq`, fila privada con destinos anulables, cero filas → `application.ErrProjectNotFound`, fila con `s.id` nulo → slice vacío, `rows.Err()` consultado antes de devolver y `nil` ante cualquier error. Agregar la aserción de compilación `_ application.StoryLister = (*PostgresStoryRepository)(nil)`. Ejecutar `go test ./tests/integration/story/postgres/... -run "TestStoryRepository" -v` y observar el pase.
- [x] 3.14 Escalera de fallback (solo si 3.3 falla, en orden, registrando en `apply-progress.md` y en una línea del commit qué paso hizo falta): (1) destinos `*string` directos (default de 3.13); (2) `s.id::text` en el `SELECT`; (3) `pgtype.UUID`/`pgtype.Text` con conversión explícita. Ninguno cambia el esquema. Si no se necesita, registrar "paso 1 suficiente".

### TRIANGULATE

- [x] 3.15 TRIANGULATE `down`/`up`: `down` sobre una base con historias elimina la columna y la constraint sin perder filas ni valores de negocio; aplicar `up` de nuevo asigna `seq` a todas las filas.
- [x] 3.16 TRIANGULATE en `tests/integration/story/postgres/repository_integration_test.go`: dos consultas consecutivas devuelven el mismo orden; una historia creada después de otra siempre aparece después dentro de una misma prioridad (creación por el repositorio real); las pruebas de creación y modificación existentes siguen pasando con `000004` aplicada. Ejecutar y observar el pase.

### REFACTOR

- [x] 3.17 REFACTOR: extraer el escaneo de las 9 columnas a un helper interno de `internal/story/infrastructure/postgres/repository.go` solo si `Update` y `ListByProject` lo justifican; de lo contrario registrar "sin cambios necesarios". Ejecutar `go test ./tests/integration/story/postgres/... -v` antes y después (verde) y `go test ./...` completo.

### Verificación y commit

- [x] 3.18 Verificar la unidad: `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde **con Docker** (los tests de integración deben mostrar `PASS`, no `SKIP`). Si algún test se saltó por falta de Docker, registrarlo como saltado y no declarar la unidad verificada.
- [x] 3.19 COMMIT de la unidad 3 (un único commit): `feat(story): list project stories in creation order`. Cuerpo de 5–8 líneas según la Regla de commits, incluyendo (si aplica) el paso de la escalera usado y, en una sola línea, si la integración se corrió o no. Rollback: revertir solo este commit; si `000004` ya fue aplicada en algún entorno, ejecutar el `down` (solo se pierde la secuencia de desempate).

## Unidad 4: Handler HTTP — `ListStoriesHandler` (commit 4, sin Docker)

Requisitos de la spec: `200` con el contenedor `{"project_id", "stories"}` y la representación completa (nueve
claves, sin `seq`); `stories: []` y nunca `null`; `404 project_not_found` sin la clave `stories`;
`422 validation_failed` con `fields.project_id`; `500 internal_error` genérico; los métodos no admitidos
siguen en `405`.
Amenazas cubiertas: solapamiento de routing y métodos (`HEAD` aceptado, `405` directo), identificador de ruta
hostil (`422` con cero llamadas) y fuga de información (`404` genérico, `500` sin el texto del error interno).

### RED

- [ ] 4.1 RED camino feliz y contenedor: crear `tests/unit/story/transport/http/list_handler_test.go` (`httptest` con `SetPathValue("project_id")`, caso de uso **real** sobre un lister falso) con: `200` con `project_id` canónico y `stories` en el orden del `Backlog`; los valores ausentes `story_points` y `estimated_hours` como `null`. Ejecutar `go test ./tests/unit/story/transport/http/...` y observar la falla (`NewListStoriesHandler` indefinido).
- [ ] 4.2 RED lista vacía (amenaza de `null`): con un fake que devuelve **`nil`**, el cuerpo crudo contiene `"stories":[]` y no `"stories":null`.
- [ ] 4.3 RED errores: `422 validation_failed` con `fields.project_id` y cero llamadas al lister (`"no-es-uuid"`, `"abc"`); `404 project_not_found` sin la clave `stories`; `500 internal_error` con mensaje genérico `an unexpected error occurred` que no contiene el texto del error interno. Ejecutar y observar las fallas.

### GREEN

- [ ] 4.4 GREEN: crear `internal/story/transport/http/list_handler.go` con `NewListStoriesHandler`, `ListStoriesHandler`, `backlogResponse` y el mapa de errores del diseño (`errors.As` de `ValidationError` antes que `errors.Is` de `ErrProjectNotFound`; respuesta construida con `make([]storyResponse, 0, len(backlog.Stories))`). Reutilizar sin cambios `storyResponse`, `newStoryResponse`, `errorResponse` y `writeJSON` de `internal/story/transport/http/handler.go`. Implementación mínima que pase 4.1–4.3. Ejecutar `go test ./tests/unit/story/transport/http/...` y observar el pase.

### TRIANGULATE

- [ ] 4.5 TRIANGULATE en `tests/unit/story/transport/http/list_handler_test.go`: cada elemento tiene exactamente las nueve claves (decodificando a `map[string]json.RawMessage`) y ni el contenedor ni los elementos incluyen `seq`; `HEAD` aceptado; `POST` directo al handler responde `405 method_not_allowed`; `Content-Type: application/json` en todas las respuestas; UUID en mayúsculas normalizado en `project_id`; un error del lister junto con historias no devuelve lista parcial. Ejecutar y observar el pase.

### REFACTOR

- [ ] 4.6 REFACTOR: ninguno previsto (los helpers se reutilizan sin cambios); registrar "REFACTOR: sin cambios necesarios" salvo que los tests muestren duplicación. Ejecutar `go test ./tests/unit/story/transport/http/...` y `go test ./...` completo.

### Verificación y commit

- [ ] 4.7 Verificar la unidad: `gofmt -l .` sin salida, `go vet ./...` limpio y `go test ./...` en verde; confirmar que `internal/story/transport/http/handler.go` no fue modificado.
- [ ] 4.8 COMMIT de la unidad 4 (un único commit): `feat(story): serve the product backlog over HTTP`. Cuerpo de 5–8 líneas según la Regla de commits. Rollback: revertir solo este commit; la ruta no está expuesta hasta la unidad 5.

## Unidad 5: Composición, arranque y documentación (commit 5, **requiere Docker** para el arranque real)

Requisitos de la spec: "Disponibilidad de la consulta condicionada a la versión del esquema"
(`version >= 4 && !dirty`); MODIFIED: la ruta de colección conserva `405` para `PUT`/`DELETE`, `GET` deja de
responder `405` con la consulta disponible y la ruta del ítem sigue sin admitir `GET`; orden de punta a punta
tras modificaciones; documentación de la operación.
Amenaza cubierta: solapamiento de routing (`GET` de la colección con y sin `Lister`; `PUT`/`DELETE`
colección; `GET` ítem; `HEAD`; `/stories/`) y exposición de datos de otros proyectos.
Nota (Decisión 8 del diseño): las aserciones existentes de `tests/unit/cmd/api/main_test.go` (líneas ~117 y
~173) **no se modifican**: al no componer un `Lister` siguen respondiendo `405` y pasan a ser la regresión del
requisito "con esquema inferior a 4, `GET` sobre la colección responde `405`".

### RED

- [ ] 5.1 RED unitario de composición: en `tests/unit/cmd/api/main_test.go` agregar `fakeStoryLister` y el helper `newBacklogHandler` (sin tocar `newStoryHandler`) y tests con `Lister` presente: `GET` colección `200`/`404`/`422`, `HEAD` `200`, `POST` colección `201`, `PUT`/`DELETE` colección `405`, `GET` ítem `405`, cero lecturas en los métodos no admitidos. Ejecutar `go test ./tests/unit/cmd/api/...` y observar la falla (`StoryDependencies.Lister` indefinido).
- [ ] 5.2 RED sin `Lister`: con `Lister` nil, `GET` colección responde `405`; el caso solo-proyectos se conserva (`GET` sobre la colección `404`). Agregar en los tests existentes (líneas ~117 y ~173) un comentario de una línea que explique que representan composiciones sin consulta. Ejecutar y observar el estado.
- [ ] 5.3 RED de integración de arranque (requiere Docker): en `tests/integration/story/postgres/http_integration_test.go` agregar el escenario `version four` (consulta disponible; log `story creation, update and backlog available`), `dirty version four` (sin rutas de historia), actualizar el log esperado de `version three` a `story backlog unavailable`; extender `testAPIStartupRoutes` con un parámetro `lists` y la función `assertBacklogRoute` (`200` con la historia modificada cuando hay consulta, `405` cuando existe la creación pero no la consulta y `404` cuando no hay rutas de historia). Ejecutar `go test ./tests/integration/story/postgres/... -run "TestAPIStartup" -v` y observar la falla.
- [ ] 5.4 RED de punta a punta en proceso (requiere Docker; `api.NewHTTPHandler` con el repositorio real): crear S1…S5 por `POST` y `GET` → S2, S5, S1, S3, S4; modificar S3 a `alta` → S2, S3, S5, S1, S4; modificar título y estado de S1 → no cambia de posición; proyecto vacío → `stories: []`; inexistente → `404` sin `stories`; inválido → `422`; dos `GET` consecutivos con cuerpos idénticos. Observar la falla.

### GREEN

- [ ] 5.5 GREEN `api.go`: en `internal/api/api.go` agregar `Lister storyapplication.StoryLister` a `StoryDependencies` (campo aditivo y opcional) y registrar `GET /projects/{project_id}/stories` con `NewListStoriesHandler` solo si `Lister != nil`; actualizar los comentarios de `StoryDependencies` y `NewHTTPHandler` para nombrar `000004`. Ejecutar `go test ./tests/unit/cmd/api/...` y observar el pase.
- [ ] 5.6 GREEN `main.go`: en `cmd/api/main.go` agregar la rama `err == nil && !dirty && version >= 4` (mismo `*PostgresStoryRepository` como `Repository`, `Updater` y `Lister`, log `story creation, update and backlog available (schema version=%d)`), actualizar el log de la rama `>= 3` a `story backlog unavailable until migration 000004 is clean (version=%d)` y el comentario del gate; las ramas `>= 3` y `>= 2` conservan su comportamiento. Ejecutar `go test ./tests/integration/story/postgres/... -run "TestAPIStartup" -v` y observar el pase.

### TRIANGULATE

- [ ] 5.7 TRIANGULATE en `tests/unit/cmd/api/main_test.go` y `tests/integration/story/postgres/http_integration_test.go`: `/projects/{id}/stories/` responde `404`; `GET` sobre la ruta del ítem `405` con la consulta disponible; creación y modificación siguen funcionando con esquema 3 y 4; versión 4 con `dirty` no expone la consulta; con `Lister` presente y un proyecto con historias de otro proyecto, la respuesta solo incluye las del consultado. Ejecutar y observar el pase.

### Documentación

- [ ] 5.8 Documentar en `README.md` una sección "Consultar el Product Backlog": la operación `GET /projects/{project_id}/stories` con un ejemplo de respuesta, el orden (prioridad `alta` > `media` > `baja` y, a igual prioridad, orden de creación), la lista vacía `[]` para un proyecto sin historias, la lista de errores (`404 project_not_found`, `422 validation_failed`, `500 internal_error`), la ausencia de paginación y filtros, la migración `000004` requerida y su condición de exposición (`version >= 4`), la limitación del orden de las filas previas a la migración y la nota operativa del bloqueo exclusivo al agregar la columna de identidad.
- [ ] 5.9 Confirmar que el delta `openspec/changes/us-07-consultar-product-backlog/specs/historia/spec.md` describe exactamente lo implementado en las unidades 1–5 (códigos HTTP, gate de versión, comportamiento de la ruta de colección). Solo se corrige el delta si la implementación difiere de lo especificado; en particular, precisar la redacción sobre `404` frente a `405` con esquemas sin creación (pregunta abierta del diseño) si corresponde. La aplicación del delta a `openspec/specs/historia/spec.md` ocurre al archivar, no aquí.

### REFACTOR

- [ ] 5.10 REFACTOR: revisar `cmd/api/main.go` (las cuatro ramas de esquema) y simplificar solo si hay duplicación real; de lo contrario registrar "sin cambios necesarios". Ejecutar `go test ./tests/unit/cmd/api/...` antes y después (verde) y `go test ./...` completo.

### Verificación y commit

- [ ] 5.11 Verificar la unidad: `gofmt -l .` sin salida, `go vet ./...` limpio, `go test ./...` en verde **con Docker** (los tests de arranque real deben mostrar `PASS`, no `SKIP`; si se saltaron, registrarlo y no declarar verificado el arranque real). Revisión manual del README renderizado. Nota: en US-06 `TestAPIStartupRoutesFollowMigrationState` falló de forma preexistente en Windows (binario `api` sin `.exe`); si reaparece, registrarlo como límite de entorno y no como resultado de esta unidad.
- [ ] 5.12 COMMIT de la unidad 5 (un único commit, tests y README incluidos): `feat(api): expose the backlog behind schema version 4`. Cuerpo de 5–8 líneas según la Regla de commits, con una sola línea de entorno si la integración no se corrió. Rollback: revertir solo este commit quita la ruta y deja creación y modificación intactas; el esquema puede permanecer en versión 4.

## Unidad 6: Cierre y verificación completa

- [ ] 6.1 Revisar que el `README.md` final documenta la consulta, el orden, la lista vacía, los errores, la ausencia de paginación, la migración `000004` y su gate `>= 4` (criterio de éxito de la propuesta). Si falta algo, corregirlo en un commit `docs(readme)` de una sola línea de cuerpo; de lo contrario registrar "README completo" en `apply-progress.md`.
- [ ] 6.2 Verificación completa final: `gofmt -l .` sin salida, `go vet ./...` limpio y `go test ./...` en verde **con Docker**. Registrar en `apply-progress.md` el resultado observado, los tests de integración corridos o saltados y cualquier límite de entorno; nunca declarar verificado lo que se saltó.
- [ ] 6.3 Confirmar los criterios de éxito de `proposal.md` contra la evidencia observada (orden verificado contra PostgreSQL, lista vacía sin `null`, `404`/`422`/`500`, solo lectura, aislamiento entre proyectos, gate `>= 4`) y verificar que `git log` muestra los cinco commits de unidad de trabajo (más el de planificación) con cuerpos de 5 a 8 líneas y sin líneas de co-autoría.

## Trazabilidad de requisitos de la spec a tareas

| Requisito de la spec (`historia`) | Tareas |
|-----------------------------------|--------|
| Consultar el Product Backlog de un proyecto existente | 2.1, 3.5, 3.6, 3.7, 4.1, 4.5, 5.1, 5.4 |
| Ordenar por prioridad y, a igual prioridad, por orden de creación | 1.1, 1.2, 1.4, 3.5, 3.8, 3.16, 5.4 |
| Informar la ausencia de historias con una lista vacía | 3.3, 3.6, 4.2, 5.4 |
| Rechazar la consulta de un proyecto inexistente | 2.4, 3.4, 4.3, 5.1, 5.4 |
| Rechazar un identificador de proyecto inválido | 2.1, 2.3, 4.3, 5.1, 5.4 |
| La consulta MUST NOT modificar el Product Backlog | 2.1, 3.9, 5.1 |
| Fallar sin divulgar detalles internos ante errores del almacenamiento | 2.4, 3.11, 4.3 |
| Disponibilidad condicionada a la versión del esquema | 5.1, 5.2, 5.3, 5.6, 5.7 |
| MODIFIED: rechazar métodos no admitidos (ruta de colección y de ítem) | 4.5, 5.1, 5.2, 5.7 |

## Riesgos de ejecución

- **Tamaño**: ~1080 líneas en un único PR exceden el presupuesto de 400; sin la excepción `size:exception` explícita (tarea 0.1) no puede iniciarse la aplicación.
- Las unidades 3 y 5 (arranque real) dependen de Docker; sin él quedan sin verificar y no pueden declararse hechas.
- El escaneo de `NULL` de un `LEFT JOIN` en destinos anulables de pgx es el único riesgo técnico nuevo: es el primer RED de la unidad 3 y la tarea 3.14 define la escalera de fallback.
- La corrección del orden depende de dos contratos (el repositorio entrega el orden de creación y el dominio ordena de forma estable): cada uno tiene su test dedicado (1.2 y 3.5) y un test de punta a punta (5.4).
- El REFACTOR de `validateStoryContent` (1.5) toca código de US-05/US-06: se exige la suite de dominio en verde antes y después.
- Todas las unidades comparten una sola rama y un solo PR: un fallo en una unidad tardía no permite mergear las anteriores por separado sin dividir el PR.
