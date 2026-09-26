# Diseño: US-07 — Consultar el Product Backlog

Artefactos de entrada: `proposal.md` (decisiones aceptadas), `specs/historia/spec.md` (8 requisitos `ADDED` y
1 `MODIFIED`) y `exploration.md` de esta misma carpeta de cambio. Fuente del requisito: issue #35.

Este documento describe el **cómo** a nivel arquitectónico. No contiene implementación; las firmas de Go y
las sentencias SQL que aparecen son contratos, no cuerpos de función. El desglose en pasos ejecutables
corresponde a la fase de tareas.

## Enfoque técnico

Enfoque A de la propuesta —**migración `000004` con `seq`, puerto de lectura separado, una única consulta
`projects LEFT JOIN stories`**— con un ajuste deliberado en el reparto de la regla de orden (Decisión 1):

- **El almacenamiento sabe *cuándo*.** `ListByProject` devuelve las historias del proyecto en **orden de
  creación** (`ORDER BY s.seq`). `seq` es un dato de almacenamiento, asignado por la base, que nunca sale de
  la capa de infraestructura.
- **El dominio sabe *qué importa más*.** `domain.NewBacklog` aplica un ordenamiento **estable** por la
  precedencia de prioridad (`alta` > `media` > `baja`), tomada de la misma lista ordenada que usa la
  validación de contenido. Por ser estable, el orden de creación que llegó del almacenamiento se conserva
  como desempate.

Las cuatro piezas, en el orden en que un request las atraviesa:

1. **Transporte como traductor puro.** `ListStoriesHandler` toma `project_id` de la ruta, invoca el caso de
   uso y proyecta el resultado al contenedor `{"project_id", "stories"}`. Es el único punto donde existe la
   diferencia entre `[]` y `null`, y es donde se garantiza `[]` por construcción.
2. **Aplicación como guardia del almacenamiento.** `ListStoriesUseCase` valida el UUID **antes** de tocar el
   puerto (cero lecturas ante un identificador inválido), normaliza el identificador a su forma canónica y
   delega el orden al dominio. No inventa resultados: un error del puerto se propaga sin lista parcial.
3. **Dominio como autoridad de la precedencia.** `Backlog` es el concepto del Product Backlog: un proyecto y
   sus historias en orden de planificación. La precedencia de prioridades existe en un solo lugar del código.
4. **Persistencia como autoridad de existencia y de cronología.** Una sola sentencia `SELECT` decide a la vez
   si el proyecto existe, cuáles son sus historias y en qué orden fueron creadas. Cero filas es "proyecto
   inexistente"; una fila con columnas de historia nulas es "proyecto sin historias".

La operación sólo se expone con el esquema en `version >= 4 && !dirty`. Creación (`>= 2`) y modificación
(`>= 3`) conservan sus condiciones y su comportamiento; ninguna de las dos cambia de código.

## Decisiones de arquitectura

### Decisión 1: dónde vive la regla de orden

Es la decisión central del diseño. La propuesta mantenía todo el orden en SQL (`ORDER BY CASE priority ...,
seq`) y dejaba a este documento evaluar la variante A' (función de dominio).

**Elección.** Repartir la regla según quién posee cada dato:

- `ListByProject` ordena **sólo** por `s.seq` (orden de creación).
- `domain.NewBacklog(projectID, stories)` aplica `slices.SortStableFunc` por el rango de prioridad, donde el
  rango es la posición de la prioridad en `AllowedPriorities()` (`alta`=0, `media`=1, `baja`=2).
- `validateStoryContent` deja de comparar literales y valida la prioridad contra `AllowedPriorities()`
  (paso REFACTOR, mismo mensaje `must be alta, media or baja`).

Así, el conjunto de prioridades y su precedencia tienen **una sola** fuente en el código (el dominio) y una
red de seguridad en la base (el `CHECK` de `000002`), igual que ya ocurre con `status` desde US-06.

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| (a) SQL completo: `ORDER BY CASE s.priority WHEN 'alta' THEN 0 WHEN 'media' THEN 1 WHEN 'baja' THEN 2 END, s.seq` | Un solo lugar para todo el orden y la base hace el trabajo. Pero agrega una **tercera** copia del conjunto de prioridades (dominio, `CHECK`, `CASE`); un `CASE` sin `ELSE` ordena en silencio una prioridad nueva al final (`NULL` en `ASC` va último); y la regla sólo se puede demostrar con Docker, en un proyecto con TDD estricto donde Docker ya es un riesgo recurrente. |
| (b) SQL parametrizado con la lista del dominio: `ORDER BY array_position($2::text[], s.priority), s.seq` pasando `domain.AllowedPriorities()` | Elimina la copia y conserva el orden en la base (útil si llega paginación). Pero la regla sigue siendo verificable únicamente contra PostgreSQL, y el SQL queda menos legible que un `ORDER BY` simple. |
| (c) Dominio completo (A' literal): cargar `seq` en `domain.Story` y ordenar por `(rango, seq)` en memoria | Testeable sin Docker, pero introduce en el modelo de dominio un campo técnico que no es parte del negocio ni de la API, y que habría que ocultar explícitamente en la respuesta. |
| (d) **Reparto: `seq` en SQL + ordenamiento estable por prioridad en el dominio** (**elegida**) | Precedencia testeable con tests puros; ningún campo técnico en el dominio; una sola copia del conjunto. Costo: la corrección depende de **dos** contratos (el repositorio entrega orden de creación y el dominio ordena de forma estable), cada uno cubierto por su propio test. |

**Rationale.** El orden de creación es un hecho de almacenamiento (`seq` lo asigna la base y el dominio no lo
conoce); la precedencia de prioridad es una regla de negocio. Ponerlos cada uno en su capa hace que ambos se
prueben donde son baratos de probar: la precedencia y los desempates de la spec (S1…S5) en tests unitarios
sin dobles, y el orden de creación en integración. El ordenamiento en memoria no tiene costo relevante porque,
sin paginación, la lista completa se carga igual.

**Condiciones que el diseño fija para que el reparto sea correcto.**

- El ordenamiento **debe** ser estable (`slices.SortStableFunc`, nunca `sort.Slice`); un test con empates
  de prioridad en orden de entrada no alfabético lo delata.
- El contrato del puerto dice explícitamente "en orden de creación ascendente"; un test de integración con
  prioridades e identificadores deliberadamente desalineados respecto de la creación lo fija.
- `NewBacklog` ordena una **copia**: no muta el slice recibido.
- Una prioridad desconocida (imposible con el `CHECK`, pero defensivo) recibe el rango
  `len(AllowedPriorities())` y queda al final, conservando su orden de creación.

**Camino de evolución.** Si una historia futura agrega paginación, el orden tiene que volver a la base,
porque no se puede paginar un orden en memoria. En ese momento el cambio natural es la opción (b), que
conserva la única fuente del dominio. Queda registrado como consecuencia, no como trabajo de US-07.

### Decisión 2: migración `000004`

**Elección.** Un par `up`/`down` que sigue la convención de nombres y de constraints nombradas de `000002` y
`000003`.

`000004_add_story_creation_sequence.up.sql`:

```sql
ALTER TABLE stories
    ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY;

ALTER TABLE stories
    ADD CONSTRAINT stories_project_id_seq_key UNIQUE (project_id, seq);
```

`000004_add_story_creation_sequence.down.sql`:

```sql
ALTER TABLE stories DROP CONSTRAINT stories_project_id_seq_key;

ALTER TABLE stories DROP COLUMN seq;
```

Detalles y sus razones:

- **`GENERATED ALWAYS`** (no `BY DEFAULT`): la base rechaza un valor explícito de `seq` en `INSERT`
  (SQLSTATE `428C9`) y en `UPDATE`. La inmutabilidad de la posición de creación es una propiedad del esquema,
  no una disciplina del código. `Create` y `Update` no nombran `seq`, así que no cambian.
- **`BIGINT`** y `NOT NULL` implícito por la identidad.
- **`UNIQUE (project_id, seq)` nombrada.** Garantiza el determinismo que exige la spec (ningún empate posible
  dentro de un proyecto), respalda la consulta (`WHERE project_id` + `ORDER BY seq` se resuelven con un solo
  índice) y, como beneficio colateral, indexa `project_id`, que hoy no tiene índice aunque es FK. Nombrarla
  permite verificarla por `pg_constraint`, igual que `stories_project_id_fkey` y `stories_status_check`.
- **Relleno (backfill) automático.** Al agregar la columna de identidad, PostgreSQL reescribe la tabla y
  asigna valores a las filas existentes en el **orden físico** del recorrido. No hay ningún dato previo
  (fecha de creación, contador) del cual reconstruir el orden real.
- **`down` explícito en dos pasos.** `DROP COLUMN` eliminaría la constraint y la secuencia propia por sí
  solo, pero borrarla primero deja el `down` legible y simétrico con `000003`.

**Limitación aceptada (ya reflejada en la spec).** Para las historias existentes antes de aplicar `000004`,
el desempate cronológico dentro de una misma prioridad **no** está garantizado: refleja el orden físico al
momento de migrar. Es determinista a partir de ese momento (el valor queda fijo), y el orden por prioridad
rige igual para todas. Se documenta en el README.

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| `GENERATED BY DEFAULT AS IDENTITY` | Permite sembrar `seq` a mano en tests, pero también que cualquier escritura altere la posición de creación. La spec exige que ninguna operación la cambie. |
| Relleno determinista ordenado por `(project_id, id)`: columna simple, `UPDATE ... row_number()`, luego `ADD GENERATED ... (START WITH máx+1)` | Reproducible entre entornos, pero el orden por UUID v4 es tan arbitrario respecto de la creación como el físico, y `START WITH` dinámico exige un bloque `DO` o `setval`. Más complejidad sin ganancia de significado. |
| `created_at TIMESTAMPTZ DEFAULT now()` en lugar de `seq` | Legible, pero dos inserciones en la misma transacción comparten `now()` y el empate exige igualmente un segundo criterio; además tiende a exponerse en la API, algo que la propuesta excluye. |
| Índice simple `(project_id, seq)` sin `UNIQUE` | Mismo soporte de consulta, pero no expresa en el esquema la garantía de "sin empates" de la que depende el determinismo. |
| Sin índice | Suficiente con el volumen actual, pero la `UNIQUE` cuesta una línea y cierra el determinismo y el acceso por FK a la vez. |

**Nota operativa.** `ADD COLUMN ... GENERATED ... AS IDENTITY` reescribe la tabla bajo `ACCESS EXCLUSIVE`.
Con el volumen actual es instantáneo; se deja anotado para entornos con muchos datos.

### Decisión 3: firma del puerto, tipo de retorno y proyecto inexistente

**Elección.** Un puerto de lectura propio en `internal/story/application/list_stories.go`, con un único
método:

```go
// StoryLister lee las historias de un proyecto existente en orden de creación ascendente.
// Devuelve ErrProjectNotFound si el proyecto no existe; un proyecto sin historias no es un error.
type StoryLister interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Story, error)
}
```

- Se **reutiliza** `application.ErrProjectNotFound` (declarado en `create_story.go`): mismo significado,
  mismo mapeo HTTP (`404 project_not_found`) que ya hace la creación. No se declara un error nuevo.
- El puerto no promete nada sobre `nil` frente a slice vacío: en Go ambos tienen largo cero, y la distinción
  sólo importa en JSON (Decisión 4).
- `*PostgresStoryRepository` implementa `StoryRepository`, `StoryUpdater` y `StoryLister`; se agrega la
  aserción de compilación `_ application.StoryLister = (*PostgresStoryRepository)(nil)`.

El caso de uso recibe un query y devuelve el concepto de dominio:

```go
type ListStoriesQuery struct {
	ProjectID string
}

func (u *ListStoriesUseCase) Execute(ctx context.Context, query ListStoriesQuery) (domain.Backlog, error)
```

`Execute`, en orden:

1. `uuid.Parse(query.ProjectID)`; si falla, devuelve
   `&domain.ValidationError{Fields: {"project_id": "must be a valid UUID"}}` **sin invocar el puerto**.
2. Invoca `ListByProject(ctx, id.String())` con el identificador canónico (minúsculas), igual que la creación.
3. Si el puerto devuelve error, devuelve `domain.Backlog{}` y el error **tal cual** (incluido
   `ErrProjectNotFound`), sin importar si el puerto también devolvió historias: nunca hay lista parcial.
4. Si no, devuelve `domain.NewBacklog(id.String(), stories)`.

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| Agregar `ListByProject` a `StoryRepository` | Rompe la compilación de todos los fakes de creación (unitarios de aplicación, transporte y `main_test.go`) y acopla la creación a `000004`. Ya descartado en US-06 por la misma razón. |
| `Execute(ctx, projectID string) ([]domain.Story, error)` | Firma mínima, pero el handler tendría que repetir el parseo del UUID para emitir `project_id` canónico en el contenedor, y agregar paginación o filtros más adelante cambiaría la firma. |
| Error propio `ErrBacklogProjectNotFound` | Sin ganancia: la condición y la respuesta HTTP son idénticas a las de la creación. |
| Que el puerto devuelva `(exists bool, stories)` | Expone en la interfaz un detalle de la consulta; el error centinela ya es la convención del módulo. |
| Validar el UUID en el handler (como la creación) | Funciona, pero la modificación (US-06) ya valida en el caso de uso; se sigue el precedente más reciente, que además es el que deja la regla "cero lecturas ante input inválido" verificable en el test de aplicación. |

**Rationale.** Un puerto por caso de uso sigue la convención establecida en US-05/US-06 y deja la
disponibilidad de la ruta como un dato inyectable. Devolver `domain.Backlog` entrega el identificador
canónico junto con las historias ya ordenadas, así que el handler no toma ninguna decisión.

### Decisión 4: consulta única `projects LEFT JOIN stories` y su lectura

**Elección.** `ListByProject` ejecuta exactamente esta sentencia con `pool.Query`:

```sql
SELECT s.id, s.project_id, s.title, s.description, s.priority, s.status,
       s.story_points, s.acceptance_criteria, s.estimated_hours
FROM projects p
LEFT JOIN stories s ON s.project_id = p.id
WHERE p.id = $1
ORDER BY s.seq
```

Interpretación de las filas:

| Resultado | Significado | Retorno |
|---|---|---|
| Cero filas | El proyecto no existe | `nil, application.ErrProjectNotFound` |
| Una fila con `s.id` `NULL` | El proyecto existe y no tiene historias | slice vacío, `nil` |
| N filas con `s.id` no nulo | Las N historias del proyecto, en orden de `seq` | historias, `nil` |
| Error al consultar, al escanear o en `rows.Err()` | Fallo inesperado | `nil, err` sin reinterpretar |

Reglas de lectura:

- Cada fila se escanea sobre una estructura privada con **destinos anulables** (`*string` para las columnas
  de texto y UUID, `*int`, `[]string`, `*float64`), porque en la fila de "proyecto vacío" todas las columnas
  de `stories` son `NULL` y pgx no puede escanear `NULL` en un `string`. Sólo las filas con `id` no nulo se
  convierten a `domain.Story`.
- Se recorren **todas** las filas y se consulta `rows.Err()` **antes** de devolver; ante cualquier error se
  devuelve `nil`. Es lo que garantiza "sin lista parcial".
- La consulta es un `SELECT` sin transacción explícita, sin `FOR UPDATE`/`FOR SHARE`: no escribe ni bloquea
  filas. Al ser una sola sentencia, ve una única instantánea MVCC; no hay ventana entre "existe" y "sus
  historias".
- `WHERE p.id = $1` y `ON s.project_id = p.id` acotan el resultado al proyecto consultado por construcción.
- Ningún error se traduce salvo el caso "cero filas", que no es un error de la base sino un resultado vacío.

**Alternativas consideradas.** (a) Dos consultas (existencia y luego historias): ya descartado; abre una
ventana entre lecturas o exige una transacción `REPEATABLE READ`. (b) Agregación en una fila
(`json_agg(... ORDER BY s.seq) FILTER (WHERE s.id IS NOT NULL)`): evita los destinos anulables, pero
traslada la serialización al SQL y pierde los codecs de pgx para `NUMERIC(7,2)` → `*float64` que US-06 ya
verificó. (c) `COALESCE` de cada columna de texto a `''`: evita punteros, pero mezcla "historia con título
vacío" con "fila de relleno" y depende de que el dominio nunca permita vacíos.

**Riesgo técnico resuelto primero.** Escanear un UUID `NULL` en `*string` es la única conversión nueva
respecto de `Update`. Se cubre en el **primer** test RED de integración (proyecto vacío). Fallback, en orden:
(1) `*string` directo; (2) si falla, `s.id::text` en el `SELECT`; (3) si falla, `pgtype.UUID`/`pgtype.Text`
con conversión explícita. Ninguno cambia el esquema.

### Decisión 5: `[]` frente a `null`

**Elección.** La garantía vive en el transporte: la respuesta se construye con
`make([]storyResponse, 0, len(backlog.Stories))` y se completa en orden. Un slice vacío no nulo se serializa
como `[]` sin importar si el caso de uso o el puerto devolvieron `nil`.

**Alternativas consideradas.** (a) Normalizar en el caso de uso (como sugería la propuesta): funciona, pero
deja la garantía lejos del lugar donde nace el problema; cualquier camino futuro que construya la respuesta
sin pasar por ese caso de uso podría volver a emitir `null`. (b) Normalizar en `NewBacklog`: mezcla un
detalle de serialización con la regla de orden. (c) Normalizar en el repositorio: el puerto tendría que
prometer algo que en Go no es observable semánticamente.

**Rationale.** `nil` y `[]` son equivalentes en Go y sólo difieren en JSON; la capa que produce JSON es la
única que puede garantizarlo por construcción. Se fija con un test de handler cuyo fake devuelve **`nil`** y
se comprueba sobre el cuerpo crudo (`"stories":[]`), y con un test de integración de punta a punta.

### Decisión 6: forma del handler y mapa de errores

**Elección.** `internal/story/transport/http/list_handler.go` con `ListStoriesHandler`,
`NewListStoriesHandler` y el tipo privado del contenedor:

```go
type backlogResponse struct {
	ProjectID string          `json:"project_id"`
	Stories   []storyResponse `json:"stories"`
}
```

Reutiliza sin cambios `storyResponse`, `newStoryResponse`, `errorResponse` y `writeJSON` de `handler.go`. No
lee el cuerpo del request (un `GET` con cuerpo se ignora).

Orden de evaluación, igual que los handlers existentes (`errors.As` antes que `errors.Is`):

| Condición | HTTP | `error` | Cuerpo |
|---|---|---|---|
| Método distinto de `GET`/`HEAD` (defensa; `ServeMux` ya filtra) | `405` | `method_not_allowed` | mensaje `only GET is supported` |
| `*domain.ValidationError` (UUID inválido) | `422` | `validation_failed` | `fields.project_id` |
| `application.ErrProjectNotFound` | `404` | `project_not_found` | mensaje `project not found`; **sin** clave `stories` |
| Cualquier otro error | `500` | `internal_error` | mensaje genérico `an unexpected error occurred` |
| Éxito | `200` | — | `{"project_id", "stories"}` en el orden de `Backlog` |

**Detalle verificado sobre `HEAD`.** Un patrón `GET /...` de `ServeMux` también atiende `HEAD`. Por eso la
defensa del handler admite `GET` **y** `HEAD`; si sólo admitiera `GET`, un `HEAD` recibiría `405` del propio
handler pese a estar enrutado.

**Alternativas consideradas.** Arreglo desnudo en lugar de contenedor (descartado en la propuesta: no admite
extensión aditiva); poner el handler en `handler.go` (un archivo con tres handlers, más difícil de revisar);
un `writeError` compartido nuevo (refactor sin necesidad funcional que infla el diff).

### Decisión 7: composición en `api.go` y gate en `main.go`

**Elección.** Campo aditivo y opcional, mismo patrón que `Updater`:

```go
type StoryDependencies struct {
	Repository storyapplication.StoryRepository
	GenerateID storyapplication.IDGenerator
	Updater    storyapplication.StoryUpdater // nil: la ruta de modificación no se registra
	Lister     storyapplication.StoryLister  // nil: GET sobre la colección no se registra (el mux responde 405)
}
```

`NewHTTPHandler` registra `GET /projects/{project_id}/stories` sólo si `Lister != nil`, dentro del bloque de
historias existente. `Lister` es independiente de `Updater` en `api.go`; es `main.go` quien los provee
juntos. Se actualizan los comentarios de `StoryDependencies` y de `NewHTTPHandler` para nombrar `000004`.

`cmd/api/main.go` pasa de tres a cuatro ramas, a partir de la misma única lectura de `schema_migrations`:

| Estado del esquema | Rutas de historia | Log |
|---|---|---|
| `err == nil && !dirty && version >= 4` | crear + modificar + **consultar** | `story creation, update and backlog available (schema version=%d)` |
| `err == nil && !dirty && version >= 3` | crear + modificar | `story backlog unavailable until migration 000004 is clean (version=%d)` |
| `err == nil && !dirty && version >= 2` | crear | sin cambios |
| cualquier otro | ninguna | sin cambios |

En la rama `>= 4` el mismo `*PostgresStoryRepository` se inyecta como `Repository`, `Updater` y `Lister`. El
log de la rama `>= 3` cambia para seguir el patrón existente ("qué falta y qué migración lo habilita"), igual
que la rama `>= 2` nombra `000003`.

**Consecuencia sobre `GET` en la colección, por estado.** Con `>= 4`: consulta. Con `2 <= version < 4`: `405`
del `ServeMux`, porque el patrón de la colección existe (sólo `POST`). Con `version < 2`, `dirty` o error de
lectura: `404`, porque no hay ninguna ruta de historias (comportamiento actual, sin cambios; ver Preguntas
abiertas).

**Alternativas consideradas.** (a) Registrar siempre `GET` y responder `503` sin esquema: la spec exige `405`
con esquema 3. (b) Derivar el lister por type-assertion sobre `Repository`: la exposición dejaría de depender
del estado del esquema y del dato inyectado. (c) Conservar el log de la rama `>= 3`: menos cambios en el test
de arranque, pero rompe la simetría de los mensajes que ya existe.

### Decisión 8: tests existentes de `main_test.go` (líneas 117 y 173)

**Hallazgo.** La propuesta anticipaba que esas aserciones pasaran de `405` a `200`/`404`. Verificado contra el
código: **no hace falta cambiarlas**. Ninguno de los dos tests compone un `Lister`:

- línea 117 (`TestStoryRoutingPreservesCollectionAndItemBoundaries`): usa `newStoryHandler(..., updater)`,
  que construye `StoryDependencies{Repository, GenerateID, Updater}`;
- línea 173 (`TestNewHTTPHandlerRegistersStoryAndPreservesProjects`): construye
  `StoryDependencies{Repository, GenerateID}`.

Con `Lister` nil la ruta `GET` no se registra y el `ServeMux` sigue respondiendo `405`. Esas aserciones pasan a
ser, sin tocarlas, la regresión del requisito "con esquema inferior a 4, `GET` sobre la colección responde
`405`" (composiciones equivalentes a las versiones 3 y 2).

**Elección.** No modificar esas líneas. Sólo agregar un comentario de una línea en cada test que explique que
representan composiciones sin consulta, y agregar tests nuevos para la composición con `Lister` (Estrategia de
pruebas). Se deja constancia en el mensaje del commit de composición.

**Alternativa descartada.** Cambiar `newStoryHandler` para aceptar un lister: modifica cuatro llamadores y
destruye justamente la regresión de "esquema 3 sin consulta". Se agrega en su lugar un helper nuevo
`newBacklogHandler`.

## Flujo de datos

```
GET /projects/{project_id}/stories            (sólo registrado si Lister != nil ⇔ schema >= 4 && !dirty)
        │
        ▼
ListStoriesHandler ── método ∉ {GET, HEAD} ─────────────────────────► 405 method_not_allowed
        │  PathValue("project_id")
        ▼
ListStoriesUseCase.Execute(ListStoriesQuery{ProjectID})
  ├─ uuid.Parse falla ──► *domain.ValidationError{project_id} ───────► 422 validation_failed
  │                       (CERO llamadas al puerto)
  ▼
StoryLister.ListByProject(ctx, id canónico)          [puerto]
        │
        ▼
PostgresStoryRepository.ListByProject
  SELECT <9 columnas de s>
  FROM projects p LEFT JOIN stories s ON s.project_id = p.id
  WHERE p.id = $1 ORDER BY s.seq
        ├─ 0 filas ──────────────► application.ErrProjectNotFound ───► 404 project_not_found
        ├─ error / rows.Err() ───► err sin reinterpretar, sin lista ──► 500 internal_error
        ├─ 1 fila con s.id NULL ─► []            (proyecto vacío)
        └─ N filas ──────────────► historias en orden de creación
        │
        ▼
domain.NewBacklog(id canónico, historias)
  copia + slices.SortStableFunc por rango en AllowedPriorities()
  (alta > media > baja; empates conservan el orden de creación)
        │
        ▼
backlogResponse{project_id, stories: make([]storyResponse, 0, n)} ──► 200
```

Propiedades garantizadas por construcción:

- **Solo lectura**: un único `SELECT` sin bloqueo; ninguna capa tiene acceso a un método de escritura en este
  flujo (el puerto sólo expone `ListByProject`).
- **Sin datos de otros proyectos**: el filtro y el `JOIN` están en la misma sentencia.
- **Proyecto vacío ≠ inexistente**, resuelto en la misma instantánea.
- **`[]`, nunca `null`**: la respuesta se construye con un slice no nulo.
- **Sin `seq` en la API**: no está en `domain.Story` ni en `storyResponse`; no hay forma de serializarlo.
- **La modificación no mueve la posición de creación**: `Update` no nombra `seq` y la base rechazaría hacerlo.

## Cambios por archivo

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/story/domain/story.go` | Modificar | Constantes `PriorityHigh`/`PriorityMedium`/`PriorityLow` y `AllowedPriorities()` (copia, en orden de precedencia); `validateStoryContent` valida contra `AllowedPriorities()` con el mismo mensaje. |
| `internal/story/domain/backlog.go` | Crear | `Backlog{ProjectID, Stories}`, `NewBacklog` (copia + orden estable por rango) y `priorityRank` no exportada. |
| `internal/story/application/list_stories.go` | Crear | `ListStoriesQuery`, puerto `StoryLister`, `ListStoriesUseCase`, `NewListStoriesUseCase`, `Execute` (UUID antes del puerto, sin lista parcial). |
| `internal/story/application/create_story.go` | Sin cambios | `ErrProjectNotFound` se reutiliza tal cual. |
| `internal/story/infrastructure/postgres/repository.go` | Modificar | `ListByProject` con la consulta de la Decisión 4, fila privada con destinos anulables y aserción `StoryLister`. |
| `internal/project/infrastructure/postgres/migrations/000004_add_story_creation_sequence.up.sql` | Crear | Columna `seq BIGINT GENERATED ALWAYS AS IDENTITY` y `stories_project_id_seq_key UNIQUE (project_id, seq)`. |
| `internal/project/infrastructure/postgres/migrations/000004_add_story_creation_sequence.down.sql` | Crear | Elimina la constraint y la columna (no destructivo para datos de negocio). |
| `internal/story/transport/http/list_handler.go` | Crear | `ListStoriesHandler`, `NewListStoriesHandler`, `backlogResponse` y el mapa de errores de la Decisión 6. |
| `internal/story/transport/http/handler.go` | Sin cambios | Se reutilizan los helpers existentes. |
| `internal/api/api.go` | Modificar | `StoryDependencies.Lister`; registro condicional de `GET /projects/{project_id}/stories`; comentarios con `000004`. |
| `cmd/api/main.go` | Modificar | Rama `>= 4` con `Lister`; log de la rama `>= 3`; comentario del gate. |
| `tests/unit/story/domain/backlog_test.go` | Crear | Precedencia, estabilidad, escenarios S1…S5, no mutación, entrada vacía, `AllowedPriorities()` como copia. |
| `tests/unit/story/application/list_stories_test.go` | Crear | Fake de `StoryLister` con contador; cero llamadas ante UUID inválido, canonicalización, orden, propagación sin lista parcial. |
| `tests/unit/story/transport/http/list_handler_test.go` | Crear | Tabla HTTP con `SetPathValue`, caso de uso real sobre lister falso; `[]` con fake que devuelve `nil`; nueve claves exactas. |
| `tests/unit/cmd/api/main_test.go` | Modificar | `fakeStoryLister`, helper `newBacklogHandler`, tests de la composición con `Lister`; líneas 117 y 173 sin cambios de aserción (Decisión 8). |
| `tests/integration/story/postgres/repository_integration_test.go` | Modificar | `000004_...up.sql` en la lista de `storyDatabase` (línea ~758); tests de `ListByProject`, instantánea de solo lectura, constraint e identidad, `down`. |
| `tests/integration/story/postgres/http_integration_test.go` | Modificar | Escenarios `version four` y `dirty version four`; log de `version three`; `assertBacklogRoute`; test de punta a punta del orden. |
| `README.md` | Modificar | Nueva sección "Consultar el Product Backlog": operación, orden, lista vacía, errores, sin paginación, migración `000004` y la limitación del orden de filas previas. |
| `openspec/specs/historia/spec.md` | Modificar al archivar | Se aplica el delta ya escrito en `specs/historia/spec.md` de esta carpeta. |

## Interfaces y contratos

Sólo firmas; la implementación corresponde a la fase de aplicación.

### Dominio — `internal/story/domain/`

```go
// story.go
const (
	PriorityHigh   = "alta"
	PriorityMedium = "media"
	PriorityLow    = "baja"
)

// AllowedPriorities devuelve una copia del conjunto cerrado de prioridades, de mayor a menor precedencia.
func AllowedPriorities() []string

// backlog.go

// Backlog es el Product Backlog de un proyecto: sus historias en orden de planificación.
type Backlog struct {
	ProjectID string
	Stories   []Story
}

// NewBacklog ordena una copia de stories por precedencia de prioridad de forma estable:
// las historias con igual prioridad conservan el orden en que se recibieron.
func NewBacklog(projectID string, stories []Story) Backlog

// priorityRank devuelve la posición de priority en AllowedPriorities(); una prioridad desconocida va al final.
func priorityRank(priority string) int
```

### Aplicación — `internal/story/application/list_stories.go`

```go
// ListStoriesQuery identifica el proyecto cuyo Product Backlog se consulta.
type ListStoriesQuery struct {
	ProjectID string
}

// StoryLister lee las historias de un proyecto existente en orden de creación ascendente.
// Devuelve ErrProjectNotFound si el proyecto no existe; un proyecto sin historias no es un error.
type StoryLister interface {
	ListByProject(ctx context.Context, projectID string) ([]domain.Story, error)
}

type ListStoriesUseCase struct {
	lister StoryLister
}

func NewListStoriesUseCase(lister StoryLister) *ListStoriesUseCase

// Execute devuelve *domain.ValidationError sin tocar el puerto si project_id no es un UUID,
// el error del puerto sin lista parcial, o el Backlog ordenado con el identificador canónico.
func (u *ListStoriesUseCase) Execute(ctx context.Context, query ListStoriesQuery) (domain.Backlog, error)
```

### Persistencia — `internal/story/infrastructure/postgres/repository.go`

```go
// ListByProject lee con una sola consulta la existencia del proyecto y sus historias en orden de seq.
// Cero filas es ErrProjectNotFound; ningún otro error se reinterpreta.
func (r *PostgresStoryRepository) ListByProject(ctx context.Context, projectID string) ([]domain.Story, error)
```

### Transporte — `internal/story/transport/http/list_handler.go`

```go
// NewListStoriesHandler construye el adaptador HTTP de consulta; el routing se compone en internal/api.
func NewListStoriesHandler(useCase *application.ListStoriesUseCase) http.Handler

type backlogResponse struct {
	ProjectID string          `json:"project_id"`
	Stories   []storyResponse `json:"stories"`
}
```

### Composición — `internal/api/api.go`

```go
type StoryDependencies struct {
	Repository storyapplication.StoryRepository
	GenerateID storyapplication.IDGenerator
	Updater    storyapplication.StoryUpdater
	Lister     storyapplication.StoryLister // nil: GET sobre la colección no se registra
}

// NewHTTPHandler conserva su firma.
func NewHTTPHandler(repository application.ProjectRepository, generateID application.IDGenerator, stories ...StoryDependencies) http.Handler
```

### Contrato HTTP

| Método y ruta | Éxito | Errores |
|---|---|---|
| `GET /projects/{project_id}/stories` (y `HEAD`) | `200` + `{"project_id": "<uuid canónico>", "stories": [<storyResponse>...]}` | `404 project_not_found`, `422 validation_failed` + `fields.project_id`, `500 internal_error` |
| `POST /projects/{project_id}/stories` | sin cambios | sin cambios |
| `PUT`/`DELETE /projects/{project_id}/stories` | — | `405` (mux) |
| `GET /projects/{project_id}/stories/{story_id}` | — | `405` (mux; sólo `PUT` registrado) |

## Estrategia de pruebas

TDD estricto (`openspec/config.yaml`: `strict_tdd: true`, `test_command: go test ./...`). Cada unidad es un
**commit de unidad de trabajo** con Conventional Commits, tests y documentación en el mismo commit, y una
traza breve **RED → GREEN → TRIANGULATE → REFACTOR** en el cuerpo del mensaje. Sin evidencia fabricada: un
test de integración saltado por falta de Docker se registra como **saltado** y la unidad no se declara
verificada.

### Por capa

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitario — dominio | Precedencia `alta`>`media`>`baja`, estabilidad ante empates, escenarios S1…S5 de la spec, no mutación de la entrada, entrada `nil`/vacía, `AllowedPriorities()` como copia, prioridad desconocida al final, validación de prioridad intacta | Tests de tabla puros, sin dobles |
| Unitario — aplicación | Cero llamadas ante UUID inválido (`"no-es-uuid"`, `"abc"`, `"123"`, UUID truncado, `""`); exactamente una llamada con el UUID canónico ante uno en mayúsculas; `Backlog` ordenado a partir de un fake en orden de creación; `ErrProjectNotFound` y error inesperado propagados con `errors.Is`, con `Backlog{}` aun si el fake devuelve historias junto al error | Fake de `StoryLister` escrito a mano con contador y `projectID` capturado; el fake sólo expone lectura (evidencia estructural de solo lectura) |
| Unitario — transporte | `200` con contenedor y orden; nueve claves exactas por elemento y ausencia de `seq` (decodificando a `map[string]json.RawMessage`); `story_points`/`estimated_hours` `null`; `"stories":[]` con fake que devuelve `nil`; `422` con `fields.project_id` y cero llamadas; `404 project_not_found` sin clave `stories`; `500` genérico sin el texto del error interno; `405` para `POST` directo al handler; `HEAD` aceptado; `Content-Type: application/json` | `httptest` + `SetPathValue`, caso de uso **real** sobre lister falso (convención de los handlers existentes) |
| Unitario — composición | Con `Lister`: `GET` colección `200`/`404`/`422`, `HEAD` `200`, `POST` colección `201`, `PUT`/`DELETE` colección `405`, `GET` ítem `405`, cero lecturas en los métodos no admitidos. Sin `Lister`: las aserciones existentes de las líneas 117 y 173 (`405`) | `api.NewHTTPHandler` con fakes, sin base de datos |
| Integración — almacenamiento | Proyecto vacío, inexistente, orden de creación desalineado de prioridad e `id`, aislamiento entre proyectos, mapeo completo de campos, `Update` no altera `seq`, instantánea de solo lectura, constraint e identidad por catálogo, `down` sin pérdida de datos, error no reinterpretado | Testcontainers `postgres:16-alpine`, migraciones `000001`–`000004` |
| Integración — arranque y HTTP | Rutas según `version`/`dirty`; orden de punta a punta vía `POST` + `PUT` + `GET`; consultas repetidas idénticas | Binario compilado por el test (patrón de `TestAPIStartupRoutesFollowMigrationState`) y `api.NewHTTPHandler` en proceso con el repositorio real |

### Instantánea de solo lectura (integración)

Antes y después de consultar (proyecto con historias, proyecto vacío, proyecto inexistente, varias veces), se
toma una instantánea textual de ambas tablas que incluye la **versión de fila** de PostgreSQL:

```sql
SELECT coalesce(string_agg(s.xmin::text || ' ' || to_jsonb(s)::text, E'\n' ORDER BY s.id), '') FROM stories s;
SELECT coalesce(string_agg(p.xmin::text || ' ' || to_jsonb(p)::text, E'\n' ORDER BY p.id), '') FROM projects p;
```

`to_jsonb(s)` incluye todas las columnas (también `seq`); `xmin` cambia con **cualquier** `UPDATE`, incluso
uno que no altere valores, y con un borrado y reinserción. Las instantáneas deben ser idénticas byte a byte.
Es una prueba más fuerte que comparar sólo valores.

### Unidades de trabajo (orden técnico de construcción)

**Unidad 1 — Dominio: precedencia y `Backlog`.** Seam: funciones puras.
- RED: `NewBacklog` con `baja, alta, media` → `alta, media, baja`; S1 `media`, S2 `alta`, S3 `media`,
  S4 `baja`, S5 `alta` → S2, S5, S1, S3, S4.
- TRIANGULATE: tres `media` conservan el orden de entrada; entrada `nil` y vacía; la entrada no se muta;
  `ProjectID` se conserva; prioridad desconocida al final; `AllowedPriorities()` devuelve una copia en orden.
- REFACTOR: constantes de prioridad y `validateStoryContent` sobre `AllowedPriorities()`, con la suite de
  `NewStory`/`NewStoryUpdate` en verde antes y después.

**Unidad 2 — Aplicación: `ListStoriesUseCase`.** Seam: interfaz `StoryLister`.
- RED: UUID inválido → `ValidationError{project_id}` y `calls == 0`; UUID válido → una llamada y `Backlog`
  ordenado.
- TRIANGULATE: tabla de UUID inválidos; UUID en mayúsculas → canónico en el puerto y en `Backlog.ProjectID`;
  `ErrProjectNotFound`; error inesperado con historias devueltas junto al error → `Backlog{}`.
- REFACTOR: sólo si los tests muestran duplicación.

**Unidad 3 — Almacenamiento y migración `000004` (requiere Docker).** Seam: PostgreSQL real.
- RED, antes de escribir `ListByProject`, en este orden (el primero resuelve el único riesgo técnico nuevo):
  1. proyecto existente sin historias → slice vacío, `nil` (escaneo de `NULL` en destinos anulables);
  2. proyecto inexistente → `ErrProjectNotFound`;
  3. historias sembradas en este orden con prioridades `baja`, `alta`, `media` e `id` que ordenan al revés
     alfabéticamente → devueltas en orden de siembra (orden de creación, no de prioridad ni de `id`);
  4. dos proyectos → sólo las del consultado;
  5. mapeo completo: `story_points` `8`, `estimated_hours` `8.50`, estado `en_progreso`, criterios en orden;
     y `NULL` → `nil` en ambos anulables;
  6. `Update` no altera `seq` ni el orden devuelto;
  7. instantánea de solo lectura idéntica antes y después (tres casos);
  8. `stories_project_id_seq_key` existe con la definición esperada (`pg_constraint`); `seq` es identidad
     `ALWAYS` (`information_schema.columns`); un `INSERT` con `seq` explícito falla con `428C9`;
  9. error no reinterpretado: con la columna `seq` renombrada, la consulta falla, el error no es
     `ErrProjectNotFound` y no se devuelven historias.
- GREEN: `000004` up/down y `ListByProject`; aplicar el fallback de la Decisión 4 **en orden** si el test 1
  falla, y registrar en el commit qué paso hizo falta.
- TRIANGULATE: `down` sobre una base con historias → columna eliminada, filas y valores de negocio intactos;
  `up` de nuevo asigna `seq` a todas.

**Unidad 4 — Handler HTTP.** Seam: `http.Handler` + `httptest`.
- RED: `200` con el contenedor; `"stories":[]` con fake que devuelve `nil`; `422`; `404` sin `stories`;
  `500` genérico.
- TRIANGULATE: nueve claves exactas y sin `seq`; `null` en anulables; `HEAD` aceptado; `405` directo;
  `Content-Type`.
- REFACTOR: ninguno previsto; los helpers se reutilizan sin cambios.

**Unidad 5 — Composición, arranque y documentación.** Seams: `api.NewHTTPHandler` con fakes y el binario real.
- RED unitario (`main_test.go`): composición con `Lister` según la tabla de la capa de composición.
- RED de integración (`http_integration_test.go`): escenario `version four` (consulta disponible, log
  `story creation, update and backlog available`), `dirty version four` (sin rutas de historia), log de
  `version three` actualizado a `story backlog unavailable`; `testAPIStartupRoutes` recibe un parámetro
  `lists` y llama a `assertBacklogRoute`, que espera `200` con la historia modificada cuando hay consulta,
  `405` cuando existe la creación pero no la consulta, y `404` cuando no hay rutas de historia.
- De punta a punta en proceso: crear S1…S5 por `POST`, `GET` → S2, S5, S1, S3, S4; modificar S3 a `alta` →
  S2, S3, S5, S1, S4; modificar título y estado de S1 → no cambia de posición; proyecto vacío → `[]`;
  inexistente → `404` sin `stories`; inválido → `422`; dos `GET` consecutivos con cuerpos idénticos.
- GREEN: `api.go`, `main.go`, README.

## Matriz de amenazas

Aplicable de forma parcial: el cambio agrega una ruta HTTP (routing de aplicación) y una lectura de base de
datos, pero no ejecuta comandos de shell, subprocesos de negocio, automatización de VCS/PR ni clasificación
de archivos ejecutables.

| Límite | Casos adversariales mínimos | Aplicabilidad | Respuesta de diseño | Tests RED planificados |
|---|---|---|---|---|
| Rutas tipo documentación (`requirements.txt`, Markdown ejecutable, `README.sh`) | clasificación y ejecución | **N/A**: el cambio no clasifica ni ejecuta archivos | — | — |
| Selección de repositorio Git | autoridad de cwd | **N/A**: sin automatización de Git en el producto | — | — |
| Estado del commit | semántica de índice/worktree | **N/A**: ninguna ruta de código toca Git | — | — |
| Estado del push | resolución de destino | **N/A**: el cambio no hace push | — | — |
| Comandos de PR | composición de argumentos | **N/A**: sin automatización de PR | — | — |
| **Routing HTTP: solapamiento y métodos** | `GET` colección con y sin consulta; `PUT`/`DELETE` colección; `GET` ítem; `HEAD`; `/stories/` | **Aplicable** | `GET` sólo se registra con `Lister`; los demás métodos siguen en `405` del mux; `HEAD` lo enruta el mux y el handler lo admite | Unidades 4 y 5: tabla de métodos con y sin `Lister` |
| **Identificador de ruta como input hostil** | `"abc"`, `"123"`, UUID truncado, vacío, mayúsculas, UUID de otro proyecto | **Aplicable** | UUID validado antes del puerto; viaja sólo como parámetro posicional `$1` | Unidades 2 y 4: `422` y cero llamadas; unidad 3: aislamiento entre proyectos |
| **Exposición de datos de otros proyectos** | proyecto A vacío con B poblado; A y B poblados | **Aplicable** | Filtro `WHERE p.id = $1` y `JOIN` por `project_id` en la misma sentencia | Unidades 3 y 5 |
| **Solo lectura** | consulta repetida; proyecto vacío; inexistente | **Aplicable** | Un `SELECT` sin bloqueo; el puerto no expone escrituras | Unidad 3: instantánea con `xmin`; unidad 2: fake sólo de lectura |
| **Fuga de información en errores** | proyecto inexistente, base caída, columna ausente | **Aplicable** | `404` genérico sin `stories`; `500` genérico; ningún detalle de base en el cuerpo | Unidad 4: cuerpos exactos; unidad 3: error no reinterpretado |
| **Respuesta sin límite** | proyecto con muchas historias | **Aplicable, riesgo aceptado** | Sin paginación por alcance; el contenedor permite agregarla de forma aditiva | Ninguno (documentado en README) |

## Migración y despliegue

Orden obligatorio, porque la ruta depende de la columna:

1. Mergear el PR. Hasta que el esquema llegue a la versión 4, la ruta no existe y el sistema se comporta como
   con US-06 (`GET` sobre la colección sigue en `405`).
2. Aplicar `000004` en el entorno (proceso externo al binario, como `000001`–`000003`).
3. Reiniciar el binario: lee `schema_migrations` al arrancar y, con `version >= 4 && !dirty`, registra la
   consulta.

Un orden equivocado no rompe nada: sin `000004`, la ruta no se registra. Con `000004` aplicada y un binario
anterior, `seq` es transparente (ni `Create` ni `Update` la nombran). No hay feature flag; la versión del
esquema **es** el flag.

Rollback: revertir el PR o correr contra un esquema en versión 3 quita la ruta; el `down` de `000004` sólo
borra la secuencia de desempate (ningún dato de negocio). Reaplicarla asigna una nueva secuencia arbitraria a
las filas existentes.

## Pronóstico de líneas cambiadas (Review Workload Guard)

Líneas de autor (altas + bajas), sin archivos generados:

| Archivo | Tipo | Líneas estimadas |
|---|---|---|
| `internal/story/domain/story.go` | producción | ~20 |
| `internal/story/domain/backlog.go` | producción | ~40 |
| `internal/story/application/list_stories.go` | producción | ~50 |
| `internal/story/infrastructure/postgres/repository.go` | producción | ~55 |
| `000004_*.up.sql` + `000004_*.down.sql` | producción | ~9 |
| `internal/story/transport/http/list_handler.go` | producción | ~65 |
| `internal/api/api.go` | producción | ~12 |
| `cmd/api/main.go` | producción | ~12 |
| `tests/unit/story/domain/backlog_test.go` | test | ~90 |
| `tests/unit/story/application/list_stories_test.go` | test | ~120 |
| `tests/unit/story/transport/http/list_handler_test.go` | test | ~160 |
| `tests/unit/cmd/api/main_test.go` | test | ~75 |
| `tests/integration/story/postgres/repository_integration_test.go` | test | ~210 |
| `tests/integration/story/postgres/http_integration_test.go` | test | ~120 |
| `README.md` | docs | ~40 |
| **Total** | | **~1080 (rango 900–1200)** |

Reparto: producción ~265, tests ~775, docs ~40. El pronóstico de la propuesta (550–750) **subestimaba** el
volumen de tests que exigen la spec (8 requisitos nuevos con 35 escenarios) y el TDD estricto.

- `Decision needed before apply: Yes`
- `Chained PRs recommended: Yes` (por tamaño; la estrategia vigente es `single-pr`)
- `400-line budget risk: High`

Con `delivery_strategy: single-pr`, la aplicación requiere que el usuario acepte explícitamente una excepción
`size:exception`. Este documento **no** la da por aceptada; la decisión corresponde a la fase de tareas.

### Orden sugerido de commits (un ciclo TDD por commit)

| # | Commit (Conventional Commits) | Unidad | Líneas aprox. | Docker |
|---|---|---|---|---|
| 1 | `feat(story): order the product backlog by priority` | 1 — dominio | ~150 | No |
| 2 | `feat(story): add the list stories use case` | 2 — aplicación | ~170 | No |
| 3 | `feat(story): list project stories in creation order` | 3 — migración + repositorio | ~275 | Sí |
| 4 | `feat(story): serve the product backlog over HTTP` | 4 — handler | ~225 | No |
| 5 | `feat(api): expose the backlog behind schema version 4` | 5 — composición + README | ~260 | Sí (arranque) |

Cada commit compila y deja `go test ./...` en verde por sí solo; ninguno expone la ruta hasta el 5.

## Riesgos de diseño

| Riesgo | Probabilidad | Mitigación en este diseño |
|---|---|---|
| El orden depende de dos contratos (repositorio en orden de creación + orden estable en el dominio) | Media | Cada contrato tiene su test dedicado (unidad 1: estabilidad con empates; unidad 3: orden de siembra desalineado de prioridad e `id`) y un test de punta a punta en la unidad 5 |
| Escanear `NULL` de un `LEFT JOIN` en destinos `*string` falla en pgx | Baja | Primer test RED de la unidad 3; fallback en tres pasos sin cambio de esquema (Decisión 4) |
| `seq` de filas previas en orden físico arbitrario | Media | Limitación ya aceptada en la spec; documentada en el README; el orden por prioridad no se ve afectado |
| El REFACTOR de `validateStoryContent` toca código de US-05/US-06 | Baja | Paso REFACTOR con toda la suite de dominio en verde antes y después; mismo mensaje literal |
| Paginación futura obliga a mover el orden a SQL | Baja | Camino registrado (Decisión 1, opción b) que conserva la única fuente del dominio |
| El pronóstico (~1080) excede el presupuesto de 400 con `single-pr` | Alta | Cinco commits autocontenidos y revisables por separado; `size:exception` explícita antes de aplicar |
| La cobertura de las unidades 3 y 5 depende de Docker | Media | Los saltos se reportan como saltos; esas unidades no se declaran verificadas sin contenedor |
| `ADD COLUMN ... IDENTITY` reescribe la tabla con bloqueo exclusivo | Baja | Irrelevante con el volumen actual; anotado en el README para entornos grandes |

## Preguntas abiertas

- [ ] **Excepción de tamaño** (`size:exception`) para el único PR: decisión del usuario, necesaria **antes** de
      aplicar. Este documento no la toma.
- [ ] **Redacción de la spec para esquemas sin creación.** El requisito de disponibilidad dice que con versión
      inferior a 4 `GET` sobre la colección "MUST responder `405`, porque esa ruta admite únicamente la
      creación". Con versión 1, `dirty` o error de lectura no existe ninguna ruta de historias y el mux
      responde `404` (comportamiento vigente, cubierto por `TestProjectOnlyCompositionDoesNotExposeStoryRoute`
      y por el escenario `version one`). Default de este diseño: conservar `404` en esos estados. Ningún
      escenario de la spec lo contradice (sólo se prueba la versión 3); se sugiere precisar la redacción al
      archivar.
- [ ] La respuesta `405` del mux para métodos no admitidos es `text/plain`, no el JSON `method_not_allowed` de
      los handlers. Es comportamiento heredado de US-05/US-06 y queda sin cambios.
