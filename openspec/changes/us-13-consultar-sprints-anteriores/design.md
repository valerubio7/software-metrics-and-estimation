# Diseño: US-13 — Consultar Sprints anteriores

Artefactos de entrada: `proposal.md` (decisiones aceptadas), `specs/sprint/spec.md` (9 requisitos `ADDED`) y
`exploration.md` de esta misma carpeta de cambio. Fuente del requisito: issue #41.

Este documento describe el **cómo** a nivel arquitectónico. No contiene implementación; las firmas de Go y
las sentencias SQL que aparecen son contratos, no cuerpos de función. El desglose en pasos ejecutables
corresponde a la fase de tareas.

## Enfoque técnico

Enfoque A de la propuesta: **migración `000011` con `sprints.seq`, puerto de lectura separado y una única
consulta `projects LEFT JOIN sprints LEFT JOIN sprint_stories LEFT JOIN stories`**, con la agrupación por
Sprint hecha en Go. Sigue el patrón ya probado de US-07 (`internal/story`), adaptado a una lista anidada.

Las cuatro piezas, en el orden en que un request las atraviesa:

1. **Transporte como traductor puro.** `ListSprintsHandler` toma `project_id` de la ruta, invoca el caso de
   uso y proyecta el resultado al contenedor `{"project_id", "sprints"}`. Es el único punto donde existe la
   diferencia entre `[]` y `null` (en `sprints` y en cada `stories`), donde se normaliza `completed_at` a UTC y
   donde se fija `is_closed: true`.
2. **Aplicación como guardia del almacenamiento.** `ListSprintsUseCase` valida el UUID **antes** de tocar el
   puerto, normaliza el identificador a su forma canónica y devuelve el historial sin reordenarlo. Un error
   del puerto se propaga sin lista parcial.
3. **Dominio como vocabulario de lectura.** `ClosedSprint`, `PlannedStory` y `SprintHistory` nombran los
   conceptos del historial. La única regla es `PlannedStory.Completed()` (verdadero si y solo si existe
   `CompletedAt`), que vive en un solo lugar del código.
4. **Persistencia como autoridad de existencia, de filtro y de cronología.** Una sola sentencia `SELECT`
   decide a la vez si el proyecto existe, cuáles de sus Sprints están cerrados, qué historias se planificaron
   en cada uno, si se completaron en ese Sprint y en qué orden se crearon.

La operación solo se expone con el esquema limpio en `version >= 11`, con un flag de readiness propio
(`SprintHistory`) independiente de `Sprints` (`>= 5`) y de `SprintClose` (`>= 8`). Creación, asignación y
cierre no cambian de código ni de condición.

## Decisiones de arquitectura

### Decisión 1: puerto de lectura separado de `SprintRepository`

**Elección.** Un puerto propio en `internal/sprint/application/list_sprints.go`, con un único método:

```go
// SprintHistoryReader lee los Sprints cerrados de un proyecto existente, del más reciente al más
// antiguo por orden de creación, cada uno con sus historias planificadas en orden de creación
// ascendente. Devuelve ErrProjectNotFound si el proyecto no existe; un proyecto sin Sprints
// cerrados no es un error.
type SprintHistoryReader interface {
	ListClosedByProject(ctx context.Context, projectID string) ([]domain.ClosedSprint, error)
}
```

- Se **reutiliza** `application.ErrProjectNotFound` (declarado en `create_sprint.go`): mismo significado y
  mismo mapeo HTTP (`404 project_not_found`) que la creación y el cierre. No se declara un error nuevo.
- `*PostgresSprintRepository` implementa `SprintRepository`, `SprintCloser` y `SprintHistoryReader`; se
  agrega la aserción de compilación `var _ application.SprintHistoryReader = (*PostgresSprintRepository)(nil)`.
- El contrato del puerto fija el orden completo (Sprints por `seq DESC`, historias por `stories.seq`). No
  promete nada sobre `nil` frente a slice vacío: la distinción solo importa en JSON (Decisión 6).

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| Agregar `ListClosedByProject` a `SprintRepository` | Rompe la compilación de `fakeSprintRepository` en `tests/unit/cmd/api/main_test.go` (solo implementa `Create`) y de los fakes de creación en `tests/unit/sprint/**`; además acopla la creación (`>= 5`) a la columna de `000011`. Ya descartado en US-06/US-07 por la misma razón. |
| Agregar el método a `SprintCloser` | Mezcla una lectura con una mutación de ciclo de vida y ata la consulta al gate `>= 8`. |
| Derivar el lector por *type assertion* sobre `Repository` (como `ProjectReader` en `api.go`) | La exposición dejaría de depender del estado del esquema: con esquema `5..10` el repositorio real satisface la interfaz y la ruta se registraría sin la columna `seq`. |
| Puerto que devuelva `(exists bool, sprints)` | Expone un detalle de la consulta; el error centinela ya es la convención del módulo. |

**Rationale.** Un puerto por caso de uso es la convención establecida (US-05, US-06, US-07, US-12) y deja la
disponibilidad de la ruta como un dato inyectable, gobernado solo por el estado del esquema.

### Decisión 2: tipos de lectura en el dominio

**Elección.** Tres tipos de solo lectura agregados a `internal/sprint/domain/sprint.go`, sin reglas de
negocio nuevas salvo la derivación de `Completed`:

```go
// ClosedSprint es una instantánea de solo lectura de un Sprint cerrado y sus historias planificadas.
type ClosedSprint struct {
	ID         string
	ProjectID  string
	SprintGoal string
	Stories    []PlannedStory
}

// PlannedStory es una historia asignada a un Sprint, con su finalización en ese Sprint.
type PlannedStory struct {
	ID          string
	Title       string
	Priority    string
	StoryPoints *int
	CompletedAt *time.Time
}

// Completed informa si la historia se completó en el Sprint que la contiene.
func (s PlannedStory) Completed() bool

// SprintHistory es el historial de Sprints cerrados de un proyecto, del más reciente al más antiguo.
type SprintHistory struct {
	ProjectID string
	Sprints   []ClosedSprint
}
```

Detalles y sus razones:

- **Sin campo `IsClosed`.** El nombre del tipo ya expresa la invariante (solo se construye con Sprints
  cerrados); el transporte emite `is_closed: true` por construcción. Un campo siempre verdadero invitaría a
  ramas muertas y a pruebas sin valor.
- **`CompletedAt *time.Time` tomado de `sprint_stories.completed_at`**, nunca de `stories.status`, que es
  global, editable con `PUT` y puede divergir (decisión de US-11). Una historia en dos Sprints tiene dos
  `PlannedStory` independientes.
- **`Completed()` como método**, no como campo: no puede quedar desincronizado de `CompletedAt` y la regla
  "completada si y solo si hay marca" existe en un solo lugar, probada sin dobles.
- **Sin `seq`, `description`, `acceptance_criteria`, `estimated_hours` ni `status`.** Lo que no está en el
  dominio no se puede serializar por accidente.
- **Nombre `PlannedStory`** (no `SprintStory`): coincide con el criterio de aceptación ("historias
  planificadas") y evita confusión con `storyapplication.SprintStoryCompletion` del módulo `story`.

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| Reutilizar `storydomain.Story` | Acopla `sprint` al dominio de `story`, arrastra campos que la spec prohíbe exponer y no tiene lugar para la finalización por Sprint. |
| Archivo nuevo `domain/history.go` | Igual de válido; se respeta la tabla de impacto de la propuesta (`sprint.go` modificado). La diferencia es solo de organización. |
| Ordenar en el dominio (como `NewBacklog` en US-07) | No hay regla de negocio que ordenar: ambos criterios (`sprints.seq`, `stories.seq`) son secuencias de almacenamiento que el dominio no conoce (Decisión 3). |

### Decisión 3: consulta única, filtro en `ON` y agrupación en Go

**Elección.** `ListClosedByProject` ejecuta exactamente esta sentencia con `pool.Query`:

```sql
SELECT sp.id, sp.project_id, sp.sprint_goal,
       st.id, st.title, st.priority, st.story_points, ss.completed_at
FROM projects p
LEFT JOIN sprints sp ON sp.project_id = p.id AND sp.is_closed
LEFT JOIN sprint_stories ss ON ss.sprint_id = sp.id
LEFT JOIN stories st ON st.id = ss.story_id
WHERE p.id = $1
ORDER BY sp.seq DESC, st.seq ASC
```

Interpretación de las filas:

| Resultado | Significado | Retorno |
|---|---|---|
| Cero filas | El proyecto no existe | `nil, application.ErrProjectNotFound` |
| Una fila con `sp.id` `NULL` | El proyecto existe y no tiene Sprints cerrados (incluido el caso de un único Sprint abierto) | slice vacío, `nil` |
| Una fila con `sp.id` no nulo y `st.id` `NULL` | Sprint cerrado sin historias asignadas | Sprint con `Stories` vacío |
| N filas con `sp.id` y `st.id` no nulos | Una fila por historia planificada | historias agrupadas por Sprint |
| Error al consultar, al escanear o en `rows.Err()` | Fallo inesperado | `nil, err` sin reinterpretar |

Reglas de diseño de la consulta:

- **`sp.is_closed` va en el `ON`, nunca en el `WHERE`.** En el `WHERE` eliminaría la fila de relleno de un
  proyecto sin Sprints cerrados y lo confundiría con un proyecto inexistente. Lo fija el primer test RED de
  integración (proyecto con solo un Sprint abierto → `[]`, no `ErrProjectNotFound`).
- **Todos los `JOIN` son `LEFT`.** Aunque las FK compuestas de `000007` garantizan que cada `sprint_stories`
  tiene su Sprint y su historia del mismo proyecto, un `INNER JOIN stories` descartaría las filas con
  `ss` nulo (Sprint sin historias) y la fila de relleno del proyecto.
- **Orden de agrupación garantizado por el `ORDER BY`.** `seq` es único (Decisión 5), así que `sp.seq DESC`
  como primera clave deja contiguas todas las filas de un mismo Sprint. La agrupación en Go abre un
  `ClosedSprint` nuevo cada vez que `sp.id` cambia respecto de la fila anterior; no necesita mapas ni un
  segundo ordenamiento. `st.seq` es único por proyecto, y las historias de un Sprint son del mismo proyecto,
  así que el orden interno también es total.
- **Sin `seq` en el `SELECT`.** Solo se usa en el `ORDER BY`; nunca sale de la infraestructura.
- **Solo lectura y consistencia.** Un `SELECT` sin transacción explícita, sin `FOR UPDATE`/`FOR SHARE`: no
  escribe ni bloquea filas y ve una única instantánea MVCC, sin ventana entre "existe", "cuáles están
  cerrados" y "qué historias tienen".
- **Acotado al proyecto por construcción.** `WHERE p.id = $1` y `ON sp.project_id = p.id` están en la misma
  sentencia; el identificador viaja solo como parámetro posicional.
- **Soporte de índice.** `sprints_project_id_seq_key UNIQUE (project_id, seq)` resuelve el filtro por
  proyecto y el orden descendente con un recorrido inverso; `sprint_stories_pkey (sprint_id, story_id)`
  resuelve el join por Sprint.

Reglas de lectura:

- Cada fila se escanea sobre una estructura privada `historyRow` con **destinos anulables** (`*string` para
  `sp.id`, `sp.project_id`, `sp.sprint_goal`, `st.id`, `st.title`, `st.priority`; `*int` para
  `story_points`; `*time.Time` para `completed_at`), porque en las filas de relleno esas columnas son `NULL`.
  Escanear un UUID `NULL` en `*string` ya está verificado por `listedStory` (US-07), así que no hay riesgo
  técnico nuevo.
- Se recorren **todas** las filas y se consulta `rows.Err()` antes de devolver; ante cualquier error se
  devuelve `nil`. Así se garantiza "sin lista parcial".
- Ningún error se traduce salvo el caso "cero filas", que no es un error de la base sino un resultado vacío.

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| `is_closed` en el `WHERE` | Más legible, pero confunde "sin Sprints cerrados" con "proyecto inexistente" (`404` incorrecto). |
| Dos consultas (existencia + Sprints con historias) | Abre una ventana entre lecturas o exige `REPEATABLE READ`; ya descartado en US-07. |
| N+1: una consulta por Sprint para sus historias | Latencia lineal en la cantidad de Sprints y la misma ventana de consistencia. |
| Agregación en SQL (`json_agg(... ORDER BY st.seq) FILTER (WHERE st.id IS NOT NULL)` por Sprint) | Evita destinos anulables y la agrupación en Go, pero traslada la serialización al SQL, pierde el codec de `timestamptz` de pgx y deja la forma JSON fuera del transporte. |
| `COALESCE` a valores vacíos | Evita punteros, pero mezcla "dato vacío" con "fila de relleno" y rompe `story_points`/`completed_at` nulos. |

### Decisión 4: caso de uso `ListSprintsUseCase`

**Elección.** En `internal/sprint/application/list_sprints.go`:

```go
type ListSprintsQuery struct {
	ProjectID string
}

func (u *ListSprintsUseCase) Execute(ctx context.Context, query ListSprintsQuery) (domain.SprintHistory, error)
```

`Execute`, en orden:

1. `uuid.Parse(query.ProjectID)`; si falla, devuelve
   `&domain.ValidationError{Fields: {"project_id": "must be a valid UUID"}}` **sin invocar el puerto**.
2. Invoca `ListClosedByProject(ctx, id.String())` con el identificador canónico (minúsculas).
3. Si el puerto devuelve error, devuelve `domain.SprintHistory{}` y el error **tal cual** (incluido
   `ErrProjectNotFound`), aunque el puerto también haya devuelto Sprints: nunca hay lista parcial.
4. Si no, devuelve `domain.SprintHistory{ProjectID: id.String(), Sprints: sprints}` sin reordenar.

**Alternativas consideradas.** Validar el UUID en el handler (como `CreateSprintHandler` y
`CloseSprintHandler`): funciona, pero el precedente de lectura más reciente (US-07) valida en el caso de uso,
lo que hace verificable "cero lecturas ante input inválido" con un test de aplicación sin HTTP. Firma
`Execute(ctx, projectID string) ([]domain.ClosedSprint, error)`: el handler tendría que repetir el parseo
para emitir el `project_id` canónico del contenedor.

### Decisión 5: migración `000011`

**Elección.** Un par `up`/`down` análogo a `000004`, con constraint nombrada.

`000011_add_sprint_creation_sequence.up.sql`:

```sql
ALTER TABLE sprints
    ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY;

ALTER TABLE sprints
    ADD CONSTRAINT sprints_project_id_seq_key UNIQUE (project_id, seq);
```

`000011_add_sprint_creation_sequence.down.sql`:

```sql
ALTER TABLE sprints DROP CONSTRAINT sprints_project_id_seq_key;

ALTER TABLE sprints DROP COLUMN seq;
```

Detalles y sus razones:

- **`GENERATED ALWAYS`**: la base rechaza un `seq` explícito (SQLSTATE `428C9`), así que el orden de
  creación es inmutable por esquema. `Create` (`INSERT INTO sprints (id, project_id, sprint_goal)`) y
  `CloseSprint` no nombran `seq` y no cambian.
- **`UNIQUE (project_id, seq)` nombrada**: expresa en el esquema la ausencia de empates de la que dependen
  el determinismo y la agrupación (Decisión 3), respalda la consulta y es verificable por `pg_constraint`.
  Para probar el escenario de la spec "dos Sprints del mismo proyecto con el mismo `seq`" se usa
  `INSERT ... OVERRIDING SYSTEM VALUE` con un `seq` repetido, que debe fallar con `23505` sobre
  `sprints_project_id_seq_key`.
- **Relleno automático en orden físico** para los Sprints existentes (limitación aceptada por la spec, igual
  que en US-07). Exacto para los Sprints creados después de migrar.
- **`down` en dos pasos**, simétrico con `000004`; solo pierde la secuencia de orden, no datos de negocio.
- **Nota operativa.** `ADD COLUMN ... IDENTITY` reescribe la tabla bajo `ACCESS EXCLUSIVE`; instantáneo con
  el volumen actual.

**Alternativas consideradas.** `GENERATED BY DEFAULT` (permite alterar la posición de creación);
`created_at TIMESTAMPTZ` (empates dentro de una transacción y tendencia a exponerse); `closed_at` para ordenar
por cierre (modifica la mutación de US-12 y deja `NULL` en Sprints ya cerrados; descartado por producto);
índice sin `UNIQUE` (no expresa la garantía de "sin empates").

### Decisión 6: handler HTTP, DTOs y mapa de errores

**Elección.** `internal/sprint/transport/http/list_handler.go`, en el paquete `transporthttp` existente, con
`ListSprintsHandler`, `NewListSprintsHandler` y DTOs propios. Reutiliza sin cambios `errorResponse` y
`writeJSON` de `handler.go`; no lee el cuerpo del request.

```go
type sprintHistoryResponse struct {
	ProjectID string                 `json:"project_id"`
	Sprints   []closedSprintResponse `json:"sprints"`
}

type closedSprintResponse struct {
	ID         string                 `json:"id"`
	ProjectID  string                 `json:"project_id"`
	SprintGoal string                 `json:"sprint_goal"`
	IsClosed   bool                   `json:"is_closed"`
	Stories    []plannedStoryResponse `json:"stories"`
}

type plannedStoryResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Priority    string     `json:"priority"`
	StoryPoints *int       `json:"story_points"`
	CompletedAt *time.Time `json:"completed_at"`
	Completed   bool       `json:"completed"`
}
```

Reglas del transporte:

- **`[]`, nunca `null`.** `sprints` y cada `stories` se construyen con `make(..., 0, len(...))`. Se fija con
  tests cuyo fake devuelve `nil` (Sprints) y un Sprint con `Stories: nil`, comprobados sobre el cuerpo crudo.
- **Sin `omitempty`** en `story_points` ni `completed_at`: la spec exige `null` explícito.
- **`completed_at` en UTC.** pgx devuelve `timestamptz` en la zona local del proceso; el handler emite
  `CompletedAt.UTC()` (copia), igual que `CompleteSprintStoryHandler` (US-11). `encoding/json` serializa en
  RFC 3339 con fracción, coherente con la respuesta de finalización.
- **`completed` desde `PlannedStory.Completed()`**, y **`is_closed` constante `true`**.
- **DTOs propios**, sin embeber `sprintResponse` de la creación: un cambio futuro en la respuesta de
  creación no debe alterar el contrato del historial. `storyResponse` del módulo `story` es privado y además
  expone campos prohibidos.

Orden de evaluación, igual que los handlers existentes (`errors.As` antes que `errors.Is`):

| Condición | HTTP | `error` | Cuerpo |
|---|---|---|---|
| Método distinto de `GET`/`HEAD` (defensa; `ServeMux` ya filtra) | `405` | `method_not_allowed` | mensaje `only GET is supported` |
| `*domain.ValidationError` (UUID inválido) | `422` | `validation_failed` | `fields.project_id` |
| `application.ErrProjectNotFound` | `404` | `project_not_found` | mensaje `project not found`; **sin** clave `sprints` |
| Cualquier otro error | `500` | `internal_error` | mensaje genérico `an unexpected error occurred`, sin lista ni detalle interno |
| Éxito | `200` | — | `{"project_id", "sprints"}` en el orden del historial |

`HEAD`: un patrón `GET` de `ServeMux` también atiende `HEAD`, por eso la defensa del handler admite ambos.

### Decisión 7: composición opcional y flag de readiness `>= 11`

**Elección.** Campo aditivo y opcional en `internal/api/api.go`, mismo patrón que `Closer`:

```go
type SprintDependencies struct {
	Repository sprintapplication.SprintRepository
	GenerateID sprintapplication.IDGenerator
	Closer     sprintapplication.SprintCloser        // supplied only after migration 000008 is clean
	Lister     sprintapplication.SprintHistoryReader // supplied only after migration 000011 is clean; nil: GET is not registered
}

type MigrationReadiness struct {
	// ... campos existentes sin cambios
	SprintHistory bool // version >= 11
}
```

- `ResolveMigrationReadiness` agrega `readiness.SprintHistory = version >= 11` después del corte por
  `lookupErr`/`dirty`, sin tocar los demás flags.
- `NewHTTPHandlerWithDependencies` registra `GET /projects/{project_id}/sprints` dentro del bloque
  `if dependencies.Sprints != nil`, solo si `Lister != nil`, independiente de `Closer`. El patrón no entra en
  conflicto con `GET /projects/{project_id}` (distinta cantidad de segmentos) ni con
  `POST /projects/{project_id}/sprints`.
- `cmd/api/main.go`, dentro de la rama `readiness.Sprints` existente:
  `if readiness.SprintHistory { sprintDependencies.Lister = repository; log.Printf("sprint history available (schema version=%d)", version) }`.
  Se reutiliza la misma instancia de `*PostgresSprintRepository`. Se actualiza el comentario del gate.
  Se sigue el precedente de `SprintClose` (solo log de disponibilidad, sin rama `else`).

**Consecuencia sobre `GET /projects/{project_id}/sprints`, por estado del esquema (verificado contra el
código):**

| Estado del esquema | Rutas de Sprint registradas | `GET` sobre la colección |
|---|---|---|
| limpio, `version >= 11` | crear + cerrar + **consultar** | `200`/`404`/`422`/`500` del handler |
| limpio, `8 <= version <= 10` | crear + cerrar | `405` del `ServeMux` (existe `POST` en el patrón) |
| limpio, `5 <= version <= 7` | crear | `405` del `ServeMux` |
| limpio `version < 5`, `dirty` o error de lectura | ninguna | **`404`** (ninguna ruta coincide; comportamiento vigente) |

La última fila contradice la redacción del cuerpo del requisito de disponibilidad en
`specs/sprint/spec.md`, que dice `405` también para `dirty` o error de resolución. En esos estados
`ResolveMigrationReadiness` devuelve `Sprints=false`, la creación no se registra y el mux responde `404`
(comportamiento ya fijado por `TestSprintCompositionIsOptionalAndPreservesExistingRoutes`, línea 74, y por
`TestMigrationReadinessSelectsRoutesIndependently`). El escenario "Esquema incompatible" de la spec sí es
correcto porque condiciona "con la creación de Sprints registrada". **Default de este diseño: conservar
`404` en esos estados** y corregir la redacción del requisito antes de aplicar (ver Preguntas abiertas).

**Alternativas consideradas.** (a) Registrar siempre `GET` y responder `503` sin esquema: la spec exige
`405` con la creación registrada. (b) Reutilizar `SprintClose` como gate: con esquema `8..10` la consulta
fallaría en tiempo de solicitud por falta de `seq`. (c) Agregar un log `unavailable until migration 000011`
en una rama `else`: útil para operación, pero rompe la simetría con el cierre dentro del mismo bloque; queda
como mejora opcional.

### Decisión 8: tests existentes de `main_test.go` (línea 87)

**Hallazgo.** La propuesta anticipaba ajustar el bucle `GET/PUT/DELETE → 405` de
`TestSprintCompositionIsOptionalAndPreservesExistingRoutes`. Verificado contra el código: **no hace falta
cambiar la aserción**. El test compone `SprintDependencies{Repository, GenerateID}` sin `Lister`; con
`Lister` nil la ruta `GET` no se registra y el mux sigue respondiendo `405`. Esa aserción pasa a ser, sin
tocarla, la regresión de "con esquema `5..10`, `GET` sobre la colección responde `405`".

**Elección.** No modificar la aserción; agregar un comentario de una línea que explique que representa una
composición sin consulta, y cubrir la composición con `Lister` en un archivo nuevo
`tests/unit/cmd/api/sprint_history_test.go` (mismo estilo que `sprint_close_test.go`). El
`fakeSprintRepository` existente sigue compilando porque el puerto de lectura es separado.

## Flujo de datos

```
GET /projects/{project_id}/sprints        (registrado sólo si Lister != nil ⇔ schema >= 11 && !dirty)
        │
        ▼
ListSprintsHandler ── método ∉ {GET, HEAD} ──────────────────────────► 405 method_not_allowed
        │  PathValue("project_id")
        ▼
ListSprintsUseCase.Execute(ListSprintsQuery{ProjectID})
  ├─ uuid.Parse falla ──► *domain.ValidationError{project_id} ───────► 422 validation_failed
  │                       (CERO llamadas al puerto)
  ▼
SprintHistoryReader.ListClosedByProject(ctx, id canónico)     [puerto]
        │
        ▼
PostgresSprintRepository.ListClosedByProject
  SELECT sp.*, st.*, ss.completed_at
  FROM projects p
  LEFT JOIN sprints sp ON sp.project_id = p.id AND sp.is_closed
  LEFT JOIN sprint_stories ss ON ss.sprint_id = sp.id
  LEFT JOIN stories st ON st.id = ss.story_id
  WHERE p.id = $1 ORDER BY sp.seq DESC, st.seq
        ├─ 0 filas ─────────────────► ErrProjectNotFound ─────────────► 404 project_not_found
        ├─ error / rows.Err() ──────► err sin reinterpretar, sin lista ► 500 internal_error
        ├─ 1 fila con sp.id NULL ───► []                (sin Sprints cerrados)
        └─ filas ───────────────────► agrupar por cambio de sp.id
                                       (st.id NULL ⇒ Sprint con Stories vacío)
        │
        ▼
domain.SprintHistory{ProjectID canónico, Sprints}   (sin reordenar)
        │
        ▼
sprintHistoryResponse{project_id, sprints: make(..., 0, n)}
  └─ closedSprintResponse{..., is_closed: true, stories: make(..., 0, m)}
       └─ plannedStoryResponse{..., completed_at: UTC o null, completed: Completed()} ──► 200
```

Propiedades garantizadas por construcción:

- **Solo lectura**: un único `SELECT` sin bloqueo; el puerto solo expone `ListClosedByProject`.
- **Solo Sprints cerrados y solo del proyecto**: filtro y `JOIN` en la misma sentencia.
- **Proyecto sin Sprints cerrados ≠ inexistente**, resuelto en la misma instantánea por el filtro en `ON`.
- **Finalización por Sprint**: `completed_at` proviene de la fila de `sprint_stories` de ese Sprint.
- **`[]`, nunca `null`**, en ambos niveles.
- **Sin `seq` ni campos prohibidos en la API**: no existen en el dominio ni en los DTOs.

## Cambios por archivo

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/project/infrastructure/postgres/migrations/000011_add_sprint_creation_sequence.up.sql` | Crear | Columna `seq BIGINT GENERATED ALWAYS AS IDENTITY` y `sprints_project_id_seq_key UNIQUE (project_id, seq)`. |
| `internal/project/infrastructure/postgres/migrations/000011_add_sprint_creation_sequence.down.sql` | Crear | Elimina la constraint y la columna (no destructivo para datos de negocio). |
| `internal/sprint/domain/sprint.go` | Modificar | `ClosedSprint`, `PlannedStory` con `Completed()` y `SprintHistory`; `Sprint`, `ValidationError` y `NewSprint` sin cambios. |
| `internal/sprint/application/list_sprints.go` | Crear | `ListSprintsQuery`, puerto `SprintHistoryReader`, `ListSprintsUseCase`, `NewListSprintsUseCase`, `Execute`. |
| `internal/sprint/application/create_sprint.go` | Sin cambios | `ErrProjectNotFound` se reutiliza tal cual. |
| `internal/sprint/infrastructure/postgres/repository.go` | Modificar | `historyRow` privada con destinos anulables, `ListClosedByProject` (Decisión 3) y aserción `SprintHistoryReader`. |
| `internal/sprint/transport/http/list_handler.go` | Crear | `ListSprintsHandler`, `NewListSprintsHandler`, DTOs y mapa de errores (Decisión 6). |
| `internal/sprint/transport/http/handler.go` | Sin cambios | Se reutilizan `errorResponse` y `writeJSON`. |
| `internal/api/api.go` | Modificar | `SprintDependencies.Lister`, `MigrationReadiness.SprintHistory` (`>= 11`), registro condicional de `GET /projects/{project_id}/sprints`, comentarios con `000011`. |
| `cmd/api/main.go` | Modificar | Asignar `Lister` bajo `readiness.SprintHistory`, log `sprint history available`, comentario del gate. |
| `tests/integration/migrations/migration_files_test.go` | Modificar | Agregar `"000011"` a la lista canónica; `len(pairs) != 11` y mensaje "exactly eleven migration pairs". |
| `tests/unit/sprint/domain/sprint_test.go` | Modificar | `PlannedStory.Completed()` con y sin marca. |
| `tests/unit/sprint/application/list_sprints_test.go` | Crear | Fake de `SprintHistoryReader` con contador; cero llamadas ante UUID inválido, canonicalización, orden preservado, propagación sin lista parcial. |
| `tests/unit/sprint/transport/http/list_handler_test.go` | Crear | Tabla HTTP con `SetPathValue` y caso de uso real sobre lector falso. |
| `tests/unit/cmd/api/sprint_history_test.go` | Crear | Readiness `SprintHistory` por versión/`dirty`/error; composición con y sin `Lister`; independencia de `SprintClose`. |
| `tests/unit/cmd/api/main_test.go` | Modificar | Solo un comentario en la línea 87 (Decisión 8); sin cambios de aserción. |
| `tests/integration/sprint/postgres/repository_integration_test.go` | Modificar | `000011_...up.sql` en `sprintMigrationNames`; tests de migración (identidad, constraint, `down`) y de `ListClosedByProject`. |
| `tests/integration/sprint/postgres/http_integration_test.go` | Modificar | Test de punta a punta en proceso con el repositorio real y `Lister`. |
| `docs/sprint-history.md` | Crear | Documentación del endpoint (precedente `docs/sprint-closure.md`): contrato, orden, listas vacías, errores, gate `>= 11`, `story_points` siempre `null` por ahora, sin paginación y limitación del `seq` de Sprints previos. |
| `openspec/config.yaml` | Modificar | Nota "Implementada: US-13 (consultar Sprints anteriores)." |
| `openspec/specs/sprint/spec.md` | Modificar al archivar | Se aplica el delta de `specs/sprint/spec.md` de esta carpeta. |

Resumen: 8 archivos nuevos (2 SQL, 2 Go de producción, 3 de test y 1 de documentación), 10 modificados y
ninguno eliminado (sin contar el delta de spec que se consolida al archivar).

## Interfaces y contratos

Solo firmas; la implementación corresponde a la fase de aplicación.

### Dominio — `internal/sprint/domain/sprint.go`

```go
type ClosedSprint struct {
	ID         string
	ProjectID  string
	SprintGoal string
	Stories    []PlannedStory
}

type PlannedStory struct {
	ID          string
	Title       string
	Priority    string
	StoryPoints *int
	CompletedAt *time.Time
}

// Completed devuelve true si y solo si CompletedAt no es nil.
func (s PlannedStory) Completed() bool

type SprintHistory struct {
	ProjectID string
	Sprints   []ClosedSprint
}
```

### Aplicación — `internal/sprint/application/list_sprints.go`

```go
type ListSprintsQuery struct {
	ProjectID string
}

type SprintHistoryReader interface {
	ListClosedByProject(ctx context.Context, projectID string) ([]domain.ClosedSprint, error)
}

type ListSprintsUseCase struct {
	reader SprintHistoryReader
}

func NewListSprintsUseCase(reader SprintHistoryReader) *ListSprintsUseCase

// Execute devuelve *domain.ValidationError sin tocar el puerto si project_id no es un UUID,
// el error del puerto sin lista parcial, o el historial con el identificador canónico.
func (u *ListSprintsUseCase) Execute(ctx context.Context, query ListSprintsQuery) (domain.SprintHistory, error)
```

### Persistencia — `internal/sprint/infrastructure/postgres/repository.go`

```go
// ListClosedByProject lee con una sola consulta la existencia del proyecto, sus Sprints cerrados
// (seq DESC) y sus historias planificadas (stories.seq ASC). Cero filas es ErrProjectNotFound;
// ningún otro error se reinterpreta.
func (r *PostgresSprintRepository) ListClosedByProject(ctx context.Context, projectID string) ([]domain.ClosedSprint, error)
```

### Transporte — `internal/sprint/transport/http/list_handler.go`

```go
// NewListSprintsHandler construye el adaptador HTTP de consulta; el routing se compone en internal/api.
func NewListSprintsHandler(useCase *application.ListSprintsUseCase) http.Handler
```

### Composición — `internal/api/api.go`

```go
type SprintDependencies struct {
	Repository sprintapplication.SprintRepository
	GenerateID sprintapplication.IDGenerator
	Closer     sprintapplication.SprintCloser
	Lister     sprintapplication.SprintHistoryReader // nil: GET on the collection is not registered
}

type MigrationReadiness struct {
	Projects, Stories, Sprints, Members, Assignment, SprintClose, Tasks, Completion bool // sin cambios
	SprintHistory bool // version >= 11 && !dirty && lookupErr == nil
}
```

### Contrato HTTP

| Método y ruta | Éxito | Errores |
|---|---|---|
| `GET /projects/{project_id}/sprints` (y `HEAD`) | `200` + `{"project_id": "<uuid canónico>", "sprints": [{"id","project_id","sprint_goal","is_closed":true,"stories":[{"id","title","priority","story_points","completed_at","completed"}]}]}` | `404 project_not_found`, `422 validation_failed` + `fields.project_id`, `500 internal_error` |
| `POST /projects/{project_id}/sprints` | sin cambios | sin cambios |
| `PUT`/`DELETE /projects/{project_id}/sprints` | — | `405` (mux) |
| `POST /projects/{project_id}/sprints/{sprint_id}/closure` | sin cambios | sin cambios |

Ejemplo de éxito:

```json
{
  "project_id": "82d38423-f02d-4259-9e35-4a291585bb1e",
  "sprints": [
    {
      "id": "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
      "project_id": "82d38423-f02d-4259-9e35-4a291585bb1e",
      "sprint_goal": "Entregar el login",
      "is_closed": true,
      "stories": [
        {"id": "a25b26b4-26bf-482e-a318-74e6cb845462", "title": "Login", "priority": "alta",
         "story_points": null, "completed_at": "2026-10-11T10:00:00Z", "completed": true}
      ]
    }
  ]
}
```

## Estrategia de pruebas

TDD estricto (`openspec/config.yaml`: `strict_tdd: true`, `test_command: go test ./...`). Cada unidad es un
**commit de unidad de trabajo** con Conventional Commits, tests y documentación en el mismo commit, y una
traza breve **RED → GREEN → TRIANGULATE → REFACTOR** en el cuerpo del mensaje. Sin evidencia fabricada: un
test de integración saltado por falta de Docker (`go test -short` o `sprintDatabase` sin contenedor) se
registra como **saltado** y la unidad no se declara verificada.

### Por capa

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitario — dominio | `Completed()` verdadero con marca y falso sin ella | Tests de tabla puros, sin dobles |
| Unitario — aplicación | Cero llamadas ante UUID inválido (`"no-es-uuid"`, `"abc"`, `"123"`, UUID truncado, `""`); una llamada con el UUID canónico ante uno en mayúsculas; historial devuelto en el mismo orden que el fake (sin reordenar); `ErrProjectNotFound` y error inesperado propagados con `errors.Is`, con `SprintHistory{}` aunque el fake devuelva Sprints junto al error | Fake de `SprintHistoryReader` escrito a mano con contador y `projectID` capturado; el fake solo expone lectura (evidencia estructural de solo lectura) |
| Unitario — transporte | `200` con contenedor y orden; claves exactas por Sprint (5) y por historia (6), sin `seq`, `description`, `acceptance_criteria`, `estimated_hours` ni `status` (decodificando a `map[string]json.RawMessage`); `is_closed: true`; `story_points` y `completed_at` `null` y `completed: false`; `completed_at` en UTC (`Z`) a partir de un `time.Time` en otra zona y `completed: true`; `"sprints":[]` con fake que devuelve `nil`; `"stories":[]` con un Sprint de `Stories: nil`; `422` con `fields.project_id` y cero llamadas; `404 project_not_found` sin clave `sprints`; `500` genérico sin el texto del error interno; `405` para `POST` directo; `HEAD` aceptado; `Content-Type: application/json` | `httptest` + `SetPathValue`, caso de uso **real** sobre lector falso (convención de los handlers existentes) |
| Unitario — composición | `ResolveMigrationReadiness`: `SprintHistory` verdadero solo con limpio `>= 11` (v10 falso, v11 y v12 verdadero, `dirty` v11 falso, error falso), `SprintClose` intacto en v8–v10; con `Lister`: `GET` colección `200`/`404`/`422`, `HEAD` `200`, `POST` `201`, `PUT`/`DELETE` `405`, cero lecturas en los métodos no admitidos; sin `Lister` y con creación: `GET` `405` (regresión existente de la línea 87); sin dependencias de Sprint: `404` | `api.NewHTTPHandlerWithDependencies` con fakes, sin base de datos (`tests/unit/cmd/api/sprint_history_test.go`) |
| Integración — migración | Identidad `ALWAYS` (`information_schema.columns`); constraint con la definición esperada (`pg_constraint`); dos `Create` consecutivos → `seq` creciente; `INSERT ... OVERRIDING SYSTEM VALUE` con `seq` repetido en el mismo proyecto → `23505` sobre `sprints_project_id_seq_key`; `down` sobre una base con Sprints, asociaciones y finalizaciones → columna y constraint eliminadas, datos intactos; `up` de nuevo asigna `seq` a todos | Testcontainers `postgres:16-alpine`, migraciones `000001`–`000011` (Docker) |
| Integración — almacenamiento | Ver orden RED de la unidad 3 | Testcontainers (Docker) |
| Integración — HTTP en proceso | Crear Sprints por `POST`, cerrarlos por `POST .../closure`, sembrar historias, asignaciones y finalizaciones por SQL; `GET` → orden, agrupación, `completed`, `[]`; inexistente `404` sin `sprints`; inválido `422`; dos `GET` consecutivos con cuerpos idénticos | `api.NewHTTPHandlerWithDependencies` con el repositorio real y `Lister` (Docker) |

El cableado de `cmd/api/main.go` se cubre con los tests de readiness y composición, igual que US-12; no se
agrega un escenario de arranque del binario (ver Decisión 7 y Riesgos).

### Instantánea de solo lectura (integración)

Antes y después de consultar (proyecto con Sprints cerrados y abiertos, proyecto sin Sprints, proyecto
inexistente, varias veces), se toma una instantánea textual de las cuatro tablas con la **versión de fila**:

```sql
SELECT coalesce(string_agg(t.xmin::text || ' ' || to_jsonb(t)::text, E'\n' ORDER BY to_jsonb(t)::text), '')
FROM <tabla> t;   -- para sprints, sprint_stories, stories y tasks
```

`to_jsonb` incluye todas las columnas (también `seq` y `completed_at`); `xmin` cambia con **cualquier**
`UPDATE`, incluso uno que no altere valores. Las instantáneas deben ser idénticas byte a byte. El orden por
`to_jsonb(t)::text` evita depender de una columna `id` (`sprint_stories` no la tiene).

### Unidades de trabajo (orden técnico de construcción)

**Unidad 1 — Migración `000011`.** Seams: sistema de archivos y PostgreSQL real.
- RED (sin Docker): `migration_files_test` exige `000011` y once pares → falla.
- GREEN: crear el par `up`/`down`; agregar `000011_...up.sql` a `sprintMigrationNames`.
- TRIANGULATE (Docker): identidad `ALWAYS`, constraint por catálogo, `seq` creciente, `23505` con
  `OVERRIDING SYSTEM VALUE`, `down` sin pérdida de datos y `up` de nuevo.
- Verificar que los tests de integración existentes de Sprint siguen en verde con `seq` presente.

**Unidad 2 — Dominio y aplicación.** Seams: tipos puros e interfaz `SprintHistoryReader`.
- RED: `Completed()`; UUID inválido → `ValidationError{project_id}` y `calls == 0`; UUID válido → una
  llamada y historial con `ProjectID` canónico.
- TRIANGULATE: tabla de UUID inválidos; mayúsculas → canónico en puerto y resultado; orden del fake
  preservado; `ErrProjectNotFound`; error inesperado con Sprints devueltos junto al error → `SprintHistory{}`.
- REFACTOR: solo si los tests muestran duplicación.

**Unidad 3 — Almacenamiento (requiere Docker).** Seam: PostgreSQL real.
- RED, antes de escribir `ListClosedByProject`, en este orden (el primero fija el riesgo del `ON`):
  1. proyecto cuyo único Sprint está abierto → slice vacío, `nil` (no `ErrProjectNotFound`);
  2. proyecto sin Sprints → slice vacío, `nil`;
  3. proyecto inexistente → `ErrProjectNotFound`;
  4. tres Sprints creados A, B, C y cerrados en orden B, C, A → devueltos C, B, A;
  5. historias creadas S1, S2, S3 y asignadas en orden S3, S1, S2 → devueltas S1, S2, S3;
  6. una historia en dos Sprints cerrados, completada solo en uno, con `stories.status='completada'` →
     `CompletedAt` presente en uno y `nil` en el otro;
  7. Sprint cerrado sin historias → aparece con `Stories` vacío, junto a otro con historias;
  8. dos proyectos con Sprints cerrados → solo los del consultado;
  9. mapeo: `title`, `priority`, `story_points` `NULL` → `nil` y sembrado por SQL `5` → `5`, `completed_at`
     igual al instante sembrado;
  10. instantánea de solo lectura idéntica antes y después;
  11. error no reinterpretado: con `sprints.seq` renombrada, la consulta falla, el error no es
      `ErrProjectNotFound` y no se devuelven Sprints.
- GREEN: `historyRow`, `ListClosedByProject` con agrupación por cambio de `sp.id`, aserción del puerto.
- TRIANGULATE: consultas repetidas devuelven resultados idénticos.

**Unidad 4 — Handler HTTP.** Seam: `http.Handler` + `httptest`.
- RED: `200` con contenedor; `"sprints":[]` con fake `nil`; `"stories":[]`; `422`; `404` sin `sprints`;
  `500` genérico.
- TRIANGULATE: claves exactas en ambos niveles; `null` en anulables; `completed_at` en UTC; `is_closed`;
  `HEAD` aceptado; `405` directo; `Content-Type`.
- REFACTOR: ninguno previsto; los helpers se reutilizan sin cambios.

**Unidad 5 — Composición, readiness y documentación.** Seams: `api.NewHTTPHandlerWithDependencies` con fakes
y con el repositorio real.
- RED unitario (`sprint_history_test.go`): readiness `SprintHistory` y tabla de composición con y sin
  `Lister`.
- RED de integración en proceso (`http_integration_test.go`): flujo de punta a punta descrito arriba.
- GREEN: `api.go`, `main.go`, comentario en `main_test.go`, `docs/sprint-history.md`, `openspec/config.yaml`.

## Matriz de amenazas

Aplicable de forma parcial: el cambio agrega una ruta HTTP (routing de aplicación) y una lectura de base de
datos, pero no ejecuta comandos de shell, subprocesos, automatización de VCS/PR ni clasificación de archivos
ejecutables.

| Límite | Casos adversariales mínimos | Aplicabilidad | Respuesta de diseño | Tests RED planificados |
|---|---|---|---|---|
| Rutas tipo documentación (`requirements.txt`, Markdown ejecutable, `README.sh`) | clasificación y ejecución | **N/A**: el cambio no clasifica ni ejecuta archivos | — | — |
| Selección de repositorio Git | autoridad de cwd | **N/A**: sin automatización de Git en el producto | — | — |
| Estado del commit | semántica de índice/worktree | **N/A**: ninguna ruta de código toca Git | — | — |
| Estado del push | resolución de destino | **N/A**: el cambio no hace push | — | — |
| Comandos de PR | composición de argumentos | **N/A**: sin automatización de PR | — | — |
| **Routing HTTP: solapamiento y métodos** | `GET` colección con y sin `Lister`; `PUT`/`DELETE` colección; `HEAD`; `/sprints/`; convivencia con `POST` y `.../closure` | **Aplicable** | `GET` solo se registra con `Lister`; los demás métodos siguen en `405` del mux; `HEAD` lo enruta el mux y el handler lo admite | Unidades 4 y 5: tabla de métodos con y sin `Lister` |
| **Identificador de ruta como input hostil** | `"abc"`, `"123"`, UUID truncado, vacío, mayúsculas, UUID de otro proyecto | **Aplicable** | UUID validado antes del puerto; viaja solo como parámetro posicional `$1` | Unidades 2 y 4: `422` y cero llamadas; unidad 3: aislamiento entre proyectos |
| **Exposición de datos de otros proyectos o de Sprints abiertos** | A sin Sprints con B poblado; A con abierto y cerrado | **Aplicable** | `WHERE p.id = $1`, `ON sp.project_id = p.id AND sp.is_closed` en la misma sentencia | Unidad 3 (casos 1, 7 y 8) y unidad 5 |
| **Solo lectura** | consulta repetida; proyecto vacío; inexistente | **Aplicable** | Un `SELECT` sin bloqueo; el puerto no expone escrituras | Unidad 3: instantánea con `xmin` sobre cuatro tablas; unidad 2: fake solo de lectura |
| **Fuga de información en errores** | proyecto inexistente, columna ausente, base caída | **Aplicable** | `404` genérico sin `sprints`; `500` genérico; ningún detalle de base en el cuerpo | Unidad 4: cuerpos exactos; unidad 3: error no reinterpretado |
| **Respuesta sin límite** | proyecto con muchos Sprints e historias | **Aplicable, riesgo aceptado** | Sin paginación por alcance; el contenedor permite agregarla de forma aditiva | Ninguno (documentado en `docs/sprint-history.md`) |

## Migración y despliegue

Orden obligatorio, porque la ruta depende de la columna:

1. Mergear el PR. Hasta que el esquema llegue a la versión 11, la ruta no existe y el sistema se comporta como
   con US-12 (`GET` sobre la colección sigue en `405` con creación registrada, `404` sin ella).
2. Aplicar `000011` en el entorno (proceso externo al binario, como `000001`–`000010`). Reconfirmar antes que
   `000011` sigue libre.
3. Reiniciar el binario: lee `schema_migrations` al arrancar y, con `version >= 11 && !dirty`, registra la
   consulta.

Un orden equivocado no rompe nada: sin `000011`, la ruta no se registra; con `000011` y un binario anterior,
`seq` es transparente (ni `Create` ni `CloseSprint` la nombran). No hay feature flag; la versión del esquema
**es** el flag.

Rollback: revertir el PR o arrancar con esquema `< 11` quita la ruta; el `down` de `000011` solo borra la
secuencia de orden. Reaplicarla asigna una nueva secuencia arbitraria a las filas existentes. Nunca borrar
Sprints, asociaciones ni finalizaciones como parte de un rollback.

## Pronóstico de líneas cambiadas (Review Workload Guard)

Líneas de autor (altas + bajas), sin archivos generados:

| Archivo | Tipo | Líneas estimadas |
|---|---|---|
| `000011_*.up.sql` + `000011_*.down.sql` | producción | ~8 |
| `internal/sprint/domain/sprint.go` | producción | ~30 |
| `internal/sprint/application/list_sprints.go` | producción | ~45 |
| `internal/sprint/infrastructure/postgres/repository.go` | producción | ~75 |
| `internal/sprint/transport/http/list_handler.go` | producción | ~85 |
| `internal/api/api.go` | producción | ~10 |
| `cmd/api/main.go` | producción | ~6 |
| `tests/integration/migrations/migration_files_test.go` | test | ~4 |
| `tests/unit/sprint/domain/sprint_test.go` | test | ~30 |
| `tests/unit/sprint/application/list_sprints_test.go` | test | ~110 |
| `tests/unit/sprint/transport/http/list_handler_test.go` | test | ~180 |
| `tests/unit/cmd/api/sprint_history_test.go` + `main_test.go` | test | ~90 |
| `tests/integration/sprint/postgres/repository_integration_test.go` | test | ~350 (migración ~90 + consulta ~260) |
| `tests/integration/sprint/postgres/http_integration_test.go` | test | ~80 |
| `docs/sprint-history.md` + `openspec/config.yaml` | docs | ~37 |
| **Total** | | **~1140 (rango 950–1250)** |

Reparto: producción ~260, tests ~845, docs ~37. Ligeramente por encima del rango de la propuesta
(900–1100) por las pruebas de catálogo de la migración y la instantánea de cuatro tablas.

- `Decision needed before apply: Yes`
- `Chained PRs recommended: Yes` (por tamaño; la estrategia vigente es `single-pr`)
- `400-line budget risk: High`

Con `delivery_strategy: single-pr`, la aplicación requiere que el usuario acepte explícitamente una excepción
`size:exception`. Este documento **no** la da por aceptada; la decisión corresponde a la fase de tareas.

### Orden sugerido de commits (un ciclo TDD por commit)

| # | Commit (Conventional Commits) | Unidad | Líneas aprox. | Docker |
|---|---|---|---|---|
| 1 | `feat(sprint): add the sprint creation sequence migration` | 1 — migración | ~105 | Sí (triangulación) |
| 2 | `feat(sprint): add the list sprints use case` | 2 — dominio + aplicación | ~215 | No |
| 3 | `feat(sprint): list closed sprints with planned stories` | 3 — repositorio | ~335 | Sí |
| 4 | `feat(sprint): serve the sprint history over HTTP` | 4 — handler | ~265 | No |
| 5 | `feat(api): expose the sprint history behind schema version 11` | 5 — composición + docs | ~225 | Sí (HTTP en proceso) |

Cada commit compila y deja `go test ./...` en verde por sí solo; ninguno expone la ruta hasta el 5. Si se
eligen PR encadenados, un corte natural es: PR 1 = commits 1–2 (~320), PR 2 = commit 3 (~335), PR 3 =
commit 4 (~265), PR 4 = commit 5 (~225); cada uno tiene inicio, fin, verificación y reversión propios.

## Riesgos de diseño

| Riesgo | Probabilidad | Mitigación en este diseño |
|---|---|---|
| Mover `is_closed` del `ON` al `WHERE` confunde "sin Sprints cerrados" con "proyecto inexistente" | Media | Primer test RED de la unidad 3 (proyecto con solo un Sprint abierto) |
| La agrupación en Go depende de que el `ORDER BY` deje contiguas las filas de cada Sprint | Baja | `seq` único por constraint; tests de orden y de Sprint sin historias junto a otro con historias |
| `completed_at` serializado en la zona local del proceso | Media | Normalización `.UTC()` en el transporte (precedente US-11) y test con un `time.Time` en otra zona |
| `seq` de Sprints previos en orden físico arbitrario | Media | Limitación aceptada en la spec; documentada |
| `seq` ordena por creación, no por cierre | Baja | Decisión de producto confirmada; escenario de spec "sin importar el orden en que se cerraron" |
| Redacción del requisito de disponibilidad (`405` en `dirty`/error) no coincide con el comportamiento real (`404`) | Alta | Default: conservar `404`; corregir la spec antes de aplicar (Preguntas abiertas) |
| El cableado de `main.go` no tiene test de arranque del binario | Baja | Cambio de 6 líneas cubierto por readiness y composición, mismo criterio que US-12 |
| Un escenario futuro de arranque con versión `>= 11` sobre una base sin `000011` registraría la ruta sin la columna | Baja | `storyDatabase`/`sprintDatabase` aplican listas fijas; quien suba la versión simulada debe aplicar `000011` |
| El pronóstico (~1140) excede el presupuesto de 400 con `single-pr` | Alta | Cinco commits autocontenidos; `size:exception` explícita o PR encadenados antes de aplicar |
| La cobertura de las unidades 1, 3 y 5 depende de Docker | Media | Los saltos se reportan como saltos; esas unidades no se declaran verificadas sin contenedor |

## Preguntas abiertas

- [ ] **Redacción del requisito "Habilitar la consulta solo con el esquema requerido".** El cuerpo dice que
      con `dirty` o error de resolución `GET` "MUST responder `405`". En esos estados (y con versión `< 5`)
      no se registra ninguna ruta de Sprint y el mux responde `404`, comportamiento vigente y ya probado. El
      escenario "Esquema incompatible" es correcto porque condiciona "con la creación de Sprints registrada".
      Default de este diseño: conservar `404` en esos estados y precisar el cuerpo del requisito ("con
      versión `5..10`, `GET` responde `405`; sin creación registrada, responde `404`") antes de `sdd-apply`.
- [ ] **Excepción de tamaño** (`size:exception`) para el único PR, o cambio a PR encadenados: decisión del
      usuario, necesaria **antes** de aplicar. Este documento no la toma.
- [ ] La respuesta `405` del mux para métodos no admitidos es `text/plain`, no el JSON `method_not_allowed` de
      los handlers. Comportamiento heredado; queda sin cambios.
