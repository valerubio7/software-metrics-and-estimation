# Diseño: US-06 — Modificar una historia del Product Backlog

Artefactos de entrada: `proposal.md` (decisiones aceptadas) y `specs/historia/spec.md` (requisitos y escenarios)
de esta misma carpeta de cambio. Fuente del requisito: issue #34.

Este documento describe el **cómo** a nivel arquitectónico. No contiene implementación; las firmas de Go
que aparecen son contratos, no cuerpos de función. El desglose en pasos ejecutables corresponde a la fase
de tareas.

## Enfoque técnico

Enfoque A de la propuesta: **PUT con reemplazo completo, validar primero, un único `UPDATE ... RETURNING`,
sin lectura previa.** El cambio se apoya íntegramente en la estructura hexagonal que US-05 ya dejó en
`internal/story/{domain,application,infrastructure,transport}` y no introduce ninguna capa, paquete ni
dependencia nueva.

Las cuatro piezas del enfoque, en el orden en que un request las atraviesa:

1. **Transporte como traductor puro.** `UpdateStoryHandler` sólo decide entre "el cuerpo no es un objeto
   JSON válido con las claves permitidas" (`400 invalid_request`) y "el cuerpo es decodificable"; en el
   segundo caso traduce JSON → `UpdateStoryCommand` y mapea el error que le devuelve el caso de uso a un
   código HTTP. No contiene reglas de negocio ni compone mensajes de validación.
2. **Aplicación como único punto de fusión de errores.** `UpdateStoryUseCase` es el único lugar que ve a la
   vez los identificadores de la ruta, las claves ausentes del cuerpo y los valores presentes, así que es el
   único lugar donde puede construirse **una sola** respuesta `422` con todos los `fields` que la spec exige
   reportar juntos. Si hay cualquier error de validación, el puerto no se toca: cero escrituras, cero
   consultas de existencia.
3. **Dominio como autoridad de las reglas.** `NewStoryUpdate` valida los seis campos editables reutilizando
   —no copiando— los chequeos de contenido de `NewStory`, y agrega encima el conjunto cerrado de `status` y
   las reglas de la estimación. Devuelve una `domain.Story` que representa el contenido *pretendido*.
4. **Persistencia como autoridad del resultado.** `Update` emite una única sentencia acotada por
   `id` **y** `project_id` con `RETURNING` de la fila completa. La historia que viaja de vuelta al cliente es
   la que devolvió la base, no la que construyó el dominio: así `project_id` y `story_points` que se leen en
   la respuesta son los almacenados, y no hay forma de que la respuesta afirme algo que la base no guardó.

Consecuencia transversal del punto 4: la pertenencia de la historia al proyecto se verifica **dentro de la
escritura**, no antes. Cero filas afectadas es indistinguible entre "la historia no existe" y "la historia
es de otro proyecto", que es exactamente lo que pide la spec, y no queda ninguna ventana entre verificar y
escribir.

La operación sólo se expone cuando el esquema está en `version >= 3 && !dirty`. La creación de US-05 conserva
su condición `>= 2` y su comportamiento, con un único cambio observable y aditivo: la respuesta de creación
pasa a emitir `"estimated_hours": null`.

## Decisiones de arquitectura

### Decisión 1: puerto `StoryUpdater` separado en lugar de extender `StoryRepository`

**Elección.** Declarar en `internal/story/application/update_story.go` una interfaz nueva
`StoryUpdater { Update(...) }`. `PostgresStoryRepository` implementa las dos interfaces; nada declara que
sean el mismo tipo.

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| Agregar `Update` a `StoryRepository` | Un solo puerto "repositorio de historias", más cercano al vocabulario habitual. Pero rompe la compilación de **todos** los fakes actuales (`tests/unit/story/application/create_story_test.go`, `tests/unit/story/transport/http/handler_test.go`, `tests/unit/cmd/api/main_test.go`), obliga a que el fake de creación implemente un método que no usa, y acopla la creación a la migración `000003`. |
| Un puerto único con métodos opcionales (type assertion en el registro) | Evita tocar los fakes, pero hace que la exposición de la ruta dependa del tipo concreto en runtime en lugar de del estado del esquema; el gate deja de ser verificable desde un test de composición. |
| `StoryUpdater` separado (**elegida**) | Un método por caso de uso, fakes existentes intactos, creación independiente de `000003`. Costo: dos interfaces para un mismo adaptador Postgres. |

**Rationale.** Los puertos en este proyecto ya son estrechos y pertenecen al caso de uso que los necesita
(`StoryRepository` tiene exactamente el método que `CreateStoryUseCase` usa). Un puerto por operación
mantiene esa convención, preserva el radio de impacto pequeño y hace que la disponibilidad de la ruta
dependa de una dependencia inyectada explícita y no de una capacidad implícita del tipo.

### Decisión 2: extraer un helper de validación de contenido compartido

**Elección.** Extraer los chequeos de `title`, `description`, `priority` y `acceptance_criteria` de
`NewStory` a una función no exportada del paquete `domain` que **acumula** en un `map[string]string` recibido
por parámetro. `NewStory` y `NewStoryUpdate` la invocan; cada uno agrega después sus propias reglas.

```go
// dentro de internal/story/domain/story.go, no exportada
func validateStoryContent(fields map[string]string, title, description, priority string, criteria []string)
```

**Alternativas consideradas.** (a) Duplicar los chequeos en `NewStoryUpdate`: cero riesgo sobre US-05, pero
garantiza divergencia futura (arreglar una regla en un lugar y no en el otro) y duplica los mensajes de
error que la spec fija textualmente. (b) Un helper que **devuelva** el mapa y luego fusionarlo: fusionar dos
mapas reintroduce la pregunta de qué mensaje gana; acumular sobre un mapa recibido no tiene ese problema.
(c) Un tipo `StoryContent` con método `Validate()`: más ceremonia de la que el paquete usa hoy, sin ganancia.

**Rationale.** La spec exige que la modificación aplique *las mismas* reglas que la creación, con los mismos
mensajes. Una sola implementación es la única forma de que eso sea verdad por construcción y no por
disciplina. La extracción es un paso **REFACTOR** posterior a GREEN, protegido por la suite de creación
existente (`tests/unit/story/domain/story_test.go` cubre 14 casos de contenido, incluidos blancos Unicode),
y no cambia ningún comportamiento en el mismo paso.

### Decisión 3: forma de `NewStoryUpdate` y del conjunto de estados

**Elección.** Constructor posicional que refleja la forma de `NewStory`, con la estimación como único
parámetro nullable, y el conjunto de estados como constantes de dominio más un accesor de sólo lectura:

```go
const (
	StatusPending    = "pendiente"
	StatusInProgress = "en_progreso"
	StatusCompleted  = "completada"
)

// AllowedStatuses devuelve una copia del conjunto cerrado de estados, en orden.
func AllowedStatuses() []string

// NewStoryUpdate valida el contenido editable completo de una historia existente.
func NewStoryUpdate(id, projectID, title, description, priority, status string, criteria []string, estimatedHours *float64) (Story, error)
```

`NewStory` deja de usar el literal `"pendiente"` y pasa a usar `StatusPending`: una sola fuente de verdad en
el código.

**Alternativas consideradas.** (a) `NewStoryUpdate(id, projectID string, update StoryUpdate)` con una
struct de parámetros: más legible en el call site con ocho argumentos, pero introduce un tipo que sólo existe
para transportar argumentos y que se solapa con `UpdateStoryCommand`. (b) `var AllowedStatuses = []string{...}`
exportada: más simple, pero es un slice exportado mutable —cualquier paquete podría reordenarlo o vaciarlo—.
(c) Un tipo `Status string` con métodos: expresivo, pero obligaría a convertir en el borde HTTP y en el SQL,
y `priority` —que tiene exactamente la misma naturaleza— hoy es `string`.

**Rationale.** Consistencia con el código existente gana sobre elegancia: `NewStory` es posicional,
`priority` es `string` validado, y el único call site de `NewStoryUpdate` es el caso de uso. `*float64`
expresa "ausente o un valor", que es exactamente la semántica de la columna. `AllowedStatuses()` devolviendo
una copia permite que los tests iteren el conjunto sin poder corromperlo.

### Decisión 4: detección de claves obligatorias y fusión de `fields` en el caso de uso

Es la decisión más delicada del cambio, porque la spec pide tres cosas que tiran en direcciones distintas:
distinguir "clave ausente" de `null` en `estimated_hours`, reportar **todas** las claves ausentes a la vez, y
reportar en una **misma** respuesta `422` los identificadores de ruta inválidos junto con los errores de
contenido del cuerpo.

**Elección.**

1. El handler ya des-serializa el cuerpo dos veces (una estricta hacia la struct, otra hacia
   `map[string]json.RawMessage` para la guarda de `null` dentro de los criterios). Ese mapa de claves crudas
   es la **única** fuente que distingue "clave ausente" de `"estimated_hours": null`: `encoding/json` decodifica
   ambos casos como `nil` en un `*float64`. El handler calcula la lista de claves ausentes y la pasa como dato
   del comando.
2. El comando lleva los valores en tipos planos más la lista de claves ausentes:

```go
type UpdateStoryCommand struct {
	ProjectID          string
	StoryID            string
	Title              string
	Description        string
	Priority           string
	Status             string
	AcceptanceCriteria []string
	EstimatedHours     *float64
	MissingFields      []string // claves JSON obligatorias ausentes del cuerpo
}
```

3. `Execute` construye un único `map[string]string` en este orden y con esta regla de precedencia:
   - una entrada `"is required"` por cada nombre en `MissingFields`;
   - `project_id` / `story_id` con `"must be a valid UUID"` si el identificador de la ruta no parsea;
   - los `Fields` que devuelva `domain.NewStoryUpdate`, **sin sobrescribir** una clave ya presente (el mensaje
     de clave ausente gana sobre el mensaje de contenido que el dominio produciría para un valor cero).
4. Si el mapa no está vacío, `Execute` devuelve `&domain.ValidationError{Fields: ...}` y **no invoca el
   puerto**. Si está vacío, normaliza ambos identificadores a su forma canónica (`uuid.UUID.String()`, igual
   que hace hoy el handler de creación) y llama a `Update`.

Verificación contra los escenarios de la spec: cuerpo `{}` → exactamente las seis claves con `is required`
(el dominio no agrega nada nuevo porque sus cuatro claves ya están tomadas y `estimated_hours` nil le parece
válida). Una sola clave omitida y el resto válido → un solo campo. `story_id` inválido + `title` vacío →
`story_id` y `title`. Ambos identificadores inválidos → `project_id` y `story_id`.

**Alternativas consideradas.**

| Opción | Tradeoff |
|---|---|
| Campos puntero en el comando (`Title *string`, …) para señalar ausencia | Type-safe, pero `estimated_hours` necesita tres estados (ausente / `null` / valor), lo que fuerza un `**float64` o un `EstimatedHoursPresent bool`; ambos son más crípticos que una lista de nombres, y el resto de los punteros hay que desreferenciar en el caso de uso igual. |
| Validar los UUID en el handler (como hace la creación) y fusionar allí | Mantiene el parseo de la ruta en la capa de transporte, que es donde conceptualmente vive. Pero para fusionar identificadores con errores de contenido el handler tendría que invocar `domain.NewStoryUpdate` por su cuenta en la rama de error, duplicando el call site de la validación; y una clave ausente más un identificador inválido sólo se pueden reportar juntos si el handler también conoce la regla de claves obligatorias. |
| Validar los UUID en el dominio | Un único call site y fusión natural, pero mete `uuid` en el dominio y haría a `NewStoryUpdate` inconsistente con `NewStory`, que acepta cualquier string como identidad (los tests existentes usan `"story-id"`). |
| Cortocircuitar en el handler cuando falta una clave (sin fusionar con los identificadores) | El más simple. Ningún escenario de la spec lo contradice, pero pierde información que el sistema ya tiene: un cuerpo incompleto con un `story_id` inválido reportaría sólo el cuerpo. |

**Rationale.** La elección concentra **toda** la decisión de validez en un único punto (el caso de uso, que
delega las reglas de contenido al dominio) y deja al handler como traductor. El precio es un campo de sabor
"transporte" (`MissingFields`) en un comando de aplicación; se acepta porque describe el *input recibido*
("estos datos no fueron suministrados"), no un detalle de HTTP: un comando idéntico llegado por CLI o por
cola tendría el mismo problema. La inconsistencia deliberada con la creación —que parsea `project_id` en el
handler— queda registrada como pregunta abierta; unificar la creación está fuera de alcance porque cambiaría
código de US-05 sin necesidad funcional.

**Regla derivada, explícita.** Sólo `estimated_hours` admite `null`. Para las otras cinco claves, un `null`
JSON decae al valor cero de su tipo (`encoding/json` no falla) y se rechaza con `422` por regla de contenido,
no con `400`. Es coherente con la spec (que lista como `400` los *tipos incorrectos*, p. ej. `title`
numérico) y queda anotado como pregunta abierta.

### Decisión 5: una única sentencia `UPDATE ... WHERE id AND project_id ... RETURNING`

**Elección.** `Update` ejecuta exactamente una sentencia parametrizada, con `QueryRow(...).Scan(...)`:

```sql
UPDATE stories
SET title = $3, description = $4, priority = $5, status = $6,
    acceptance_criteria = $7, estimated_hours = $8
WHERE id = $1 AND project_id = $2
RETURNING id, project_id, title, description, priority, status,
          story_points, acceptance_criteria, estimated_hours
```

`project_id`, `id` y `story_points` **no aparecen en el `SET`**: la inmutabilidad no es una regla que se
chequea, es una propiedad de la sentencia. `pgx.ErrNoRows` (cero filas actualizadas) se traduce a
`application.ErrStoryNotFound`; cualquier otro error se propaga tal cual y termina en `500`.

**Alternativas consideradas.** (a) `SELECT` de pertenencia y luego `UPDATE`: dos viajes y una ventana de
carrera entre verificar y escribir, o una transacción para cerrarla. (b) `UPDATE` acotado sólo por `id` y
comparar el `project_id` devuelto: escribiría en una historia de otro proyecto antes de darse cuenta.
(c) `UPDATE` + `SELECT` posterior para leer `story_points`: un viaje extra y una ventana en la que otra
operación (US-14) podría cambiar los puntos entre ambas lecturas.

**Rationale.** `RETURNING` convierte "escribir y leer el resultado" en una operación atómica. Acotar por los
dos identificadores hace que la edición cruzada entre proyectos sea, por construcción, indistinguible de una
historia inexistente —el requisito de no divulgación se cumple sin código defensivo—. Y `story_points` viaja
de vuelta sin que esta operación necesite conocer su semántica.

**Traducción de errores, deliberadamente estrecha.** Sólo `pgx.ErrNoRows` se interpreta. Las violaciones de
`CHECK` (SQLSTATE `23514`, estado o estimación inválidos que el dominio debería haber bloqueado antes) y el
desborde numérico (`22003`, un valor por encima de `NUMERIC(7,2)`) **no** se traducen: son bugs, y salen como
`500 internal_error`. Es la misma política que `Create`, que sólo interpreta el FK nombrado
`stories_project_id_fkey`. Confundir un `CHECK` violado con "no encontrada" ocultaría precisamente el defecto
que ese `CHECK` existe para delatar.

### Decisión 6: `NUMERIC(7,2)` ↔ `*float64`, y la escalera de fallback

**Elección.** Almacenar `estimated_hours NUMERIC(7,2) NULL`, exponerlo como número JSON y representarlo en Go
como `*float64`, pasándolo como parámetro y escaneándolo directamente desde `RETURNING`. El round-trip de
pgx **no está verificado** y hay dos riesgos distintos en juego:

- **(a) codec `numeric` ↔ `float64`**: precisión y errores de escaneo en los bordes `0.01` y `99999.99`.
- **(b) `NULL` → `**float64`**: escanear una columna nullable en un destino `*float64` requiere pasar
  `&story.EstimatedHours`, es decir un puntero a puntero. El mismo riesgo existe **ya** para
  `story_points INTEGER NULL` → `*int`, que hoy nunca se escanea (US-05 sólo inserta `NULL`) y que esta
  operación escanea por primera vez.

Ambos se verifican en el test **RED** del slice de almacenamiento, antes de cualquier implementación. La
escalera de fallback, en orden y con el criterio de disparo explícito:

| Paso | Disparo | Respuesta | Costo |
|---|---|---|---|
| 1 | — | Parámetro `*float64`, `RETURNING estimated_hours` escaneado en `&story.EstimatedHours` | Ninguno |
| 2 | El escaneo falla, o `0.01` / `99999.99` no vuelven idénticos | Casteos explícitos en SQL: `$8::numeric` y `RETURNING estimated_hours::double precision`; intermediarios `pgtype.Numeric` / `sql.NullFloat64` (y `pgtype.Int4` para `story_points`) con conversión explícita en el repositorio | SQL algo más verboso; **la garantía de escala en la base se conserva** |
| 3 | El paso 2 tampoco hace round-trip | Cambiar la columna a `DOUBLE PRECISION NULL CHECK (estimated_hours > 0)` **dentro del mismo archivo `000003`** | Se pierde la garantía de 2 decimales en la base; el dominio queda como única autoridad de precisión. Se documenta en README y se anota en el riesgo correspondiente |

**Alternativas consideradas.** Ir directo a `DOUBLE PRECISION` (más simple, pero renuncia de entrada a una
garantía que la base da gratis y admite `8.333333` almacenado si el dominio falla); usar `NUMERIC` sin escala
(no acota el rango); representar las horas como enteros de centésimas en Go (exacto, pero deforma el contrato
JSON que la spec ya fijó como número decimal).

**Rationale.** El orden de la escalera garantiza que un problema de *plumbing* de pgx nunca fuerce un cambio
de esquema: primero se arregla donde está el problema (la conversión), y sólo se degrada el esquema si la
conversión es irrecuperable. Editar `000003` en el paso 3 es legítimo porque la migración no está aplicada en
ningún entorno todavía; una vez liberada, el mismo cambio exigiría una `000004`.

**Nota sobre el tope superior.** `NUMERIC(7,2)` ya acota a `99999.99` por precisión: un valor mayor produce
error de desborde, no truncamiento. El `CHECK` sólo necesita expresar la positividad; el tope y la cantidad
de decimales los exige el dominio con un mensaje de campo, y la base actúa como red de seguridad.

### Decisión 7: validar "como máximo 2 decimales" con `strconv.FormatFloat`, no con aritmética de floats

**Elección.** En el dominio, decidir la precisión sobre la representación decimal más corta que reproduce el
`float64`: `strconv.FormatFloat(v, 'f', -1, 64)` y contar los dígitos después del punto.

**Alternativas consideradas.** (a) `v*100 == math.Trunc(v*100)`: falla con `0.07` (`0.07*100` es
`7.000000000000001` en binario) y rechazaría un valor válido. (b) `math.Round(v*100)/100 == v`: correcto en
todos los casos del rango examinados (`0.01`, `0.07`, `99999.99` pasan; `1.001`, `0.005` se rechazan), pero
exige razonar sobre error de redondeo para convencerse. (c) Validar el literal JSON crudo en el handler:
exacto, pero mueve una regla de negocio al transporte y dejaría al dominio aceptando valores que la API
rechaza. (d) `math/big.Rat` o una librería decimal: exacto y sobredimensionado para una regla de dos
decimales.

**Rationale.** `FormatFloat(v, 'f', -1, 64)` produce la cadena decimal más corta que round-trippea al mismo
`float64`, así que "cuántos decimales tiene este valor" se responde sin aritmética de punto flotante y el
test lo lee igual que una persona: `1.001` → `"1.001"` → 3 decimales → rechazado; `3` → `"3"` → 0 decimales →
aceptado. Se queda además en el dominio, donde la regla pertenece. `NaN` e `Inf` no son representables en
JSON y, si aparecieran, la comparación `> 0` los rechaza.

### Decisión 8: conjunto de estados en el dominio **y** `CHECK` en la base

**Elección.** Duplicar deliberadamente el conjunto cerrado: constantes de dominio (Decisión 3) como
autoridad para la API, y `CHECK (status IN ('pendiente','en_progreso','completada'))` en `000003` como red de
seguridad de almacenamiento. La spec lo exige en ambos lados ("El conjunto cerrado MUST garantizarse también
en el almacenamiento").

**Alternativas consideradas.** (a) Sólo el dominio: cualquier escritura por fuera de la aplicación (script,
migración futura, psql) podría dejar un estado que el código no entiende. (b) Sólo la base: el error llegaría
como `500` en lugar de `422` con `fields.status`, y el dominio dejaría de ser la autoridad. (c) Un `ENUM` de
PostgreSQL: más expresivo, pero ampliar un enum y revertirlo es más rígido que un `CHECK`, y `priority` —el
precedente directo en `000002`— usa `CHECK`.

**Rationale.** Seguir el precedente de `priority` mantiene el esquema legible y uniforme. La duplicación es
el costo consciente de tener dos autoridades con propósitos distintos; queda anotada: ampliar el conjunto
exige tocar la constante del dominio **y** una migración.

### Decisión 9: migración `000003`, `up` y `down`

**Elección.** Un archivo por dirección, siguiendo la convención de nombre y de constraints nombradas de
`000002` (que nombra explícitamente `stories_project_id_fkey`).

`000003_add_story_estimated_hours.up.sql`:

```sql
ALTER TABLE stories
    ADD COLUMN estimated_hours NUMERIC(7,2) NULL
    CONSTRAINT stories_estimated_hours_positive CHECK (estimated_hours > 0);

ALTER TABLE stories
    ADD CONSTRAINT stories_status_check
    CHECK (status IN ('pendiente', 'en_progreso', 'completada'));
```

`000003_add_story_estimated_hours.down.sql`:

```sql
ALTER TABLE stories DROP CONSTRAINT stories_status_check;

ALTER TABLE stories DROP COLUMN estimated_hours;
```

**Alternativas consideradas.** (a) Dos migraciones separadas (`000003` columna, `000004` `CHECK` de estado):
más granular, pero obligaría a un gate `>= 4` y las dos cosas son requisitos de la misma operación.
(b) `CHECK` de estado sin nombre: PostgreSQL le asignaría uno generado y el `down` tendría que adivinarlo.
(c) Columna `NOT NULL DEFAULT 0`: contradice la regla de que `0` nunca es válido y que la ausencia es `null`.

**Rationale.** Un solo par `up`/`down` mantiene el gate de esquema en un único número. Nombrar las
constraints hace el `down` determinista y permite que un test de integración las verifique por nombre, tal
como ya hace `TestStoryRepositoryRejectsMissingProjectWithNamedForeignKey` con el FK. El `ADD CONSTRAINT` de
estado valida las filas existentes, todas en `pendiente`, así que la migración no puede fallar por datos
previos.

**Nota de reversibilidad.** El `down` es **destructivo**: borra la columna y con ella todas las estimaciones.
Además, quitar el `CHECK` no revierte los datos: filas ya guardadas con `en_progreso` o `completada`
sobrevivirían con un valor que el código revertido no entiende. El plan de rollback de la propuesta exige
exportar antes y devolver esas filas a `pendiente` de forma deliberada.

### Decisión 10: wiring en `api.go` / `main.go` y gate `version >= 3`

**Elección.** Un campo **aditivo y opcional** en `StoryDependencies`, y el registro de la ruta condicionado a
que ese campo esté presente:

```go
// StoryDependencies habilita las operaciones de historia disponibles según el esquema.
type StoryDependencies struct {
	Repository storyapplication.StoryRepository
	GenerateID storyapplication.IDGenerator
	Updater    storyapplication.StoryUpdater // nil: la ruta de modificación no se registra
}
```

`NewHTTPHandler` registra `PUT /projects/{project_id}/stories/{story_id}` **sólo** si `Updater != nil`.
`cmd/api/main.go` resuelve tres casos a partir de una única lectura de `schema_migrations`, con un log propio
cada uno:

| Estado del esquema | Rutas expuestas |
|---|---|
| `err == nil && !dirty && version >= 3` | proyectos + crear historia + **modificar historia** |
| `err == nil && !dirty && version >= 2` | proyectos + crear historia |
| cualquier otro (error, `dirty`, `version < 2`) | sólo proyectos |

El mismo `*PostgresStoryRepository` se inyecta como `Repository` y como `Updater`; satisface ambas interfaces.

**Alternativas consideradas.** (a) Un segundo parámetro variádico o una segunda función constructora
(`NewHTTPHandlerWithUpdates`): duplica la superficie de composición. (b) Derivar el updater con un
type-assertion sobre `Repository`: la exposición de la ruta pasaría a depender del tipo concreto en lugar del
estado del esquema, y el gate dejaría de ser verificable desde `main_test.go` con fakes. (c) Registrar siempre
la ruta y devolver `503` si el esquema no está migrado: la spec pide explícitamente `404` "porque la ruta no
está registrada".

**Rationale.** El campo aditivo mantiene compilando a todos los llamadores actuales —incluido
`tests/unit/cmd/api/main_test.go`, que construye `StoryDependencies{Repository, GenerateID}`— y convierte la
disponibilidad de la ruta en un dato inyectable, es decir testeable sin base de datos. Un despliegue sin
migrar simplemente carece de la ruta en lugar de fallar en runtime contra una columna inexistente.

**Nota de compatibilidad verificada.** `ServeMux` de Go 1.22+ hace que `PUT /projects/{id}/stories/{story_id}`
**no** capture `/projects/{id}/stories` (los comodines no matchean un segmento vacío), así que las
aserciones existentes de `405` para `GET`/`PUT`/`DELETE` sobre la ruta de colección siguen valiendo sin
cambios. Por simetría con la ruta de colección, `/projects/{id}/stories/` (con `story_id` vacío) responde
`404`, no `405`.

### Decisión 11: ubicación de los archivos del transporte y helpers compartidos

**Elección.** Un archivo hermano `internal/story/transport/http/update_handler.go` para el handler nuevo, y
**dos** helpers compartidos en `handler.go`, obtenidos generalizando lo que ya existe:

```go
// decodeStoryObject decodifica estrictamente un único objeto JSON en target
// y devuelve sus claves crudas para inspeccionar presencia.
func decodeStoryObject(body io.Reader, target any) (map[string]json.RawMessage, error)

// newStoryResponse proyecta una historia del dominio a la respuesta pública.
func newStoryResponse(story domain.Story) storyResponse
```

`decodeCreateStoryRequest` desaparece a favor del primero (mismo comportamiento: un solo objeto,
`DisallowUnknownFields`, guarda contra `null` dentro de los criterios); el handler de creación ignora el mapa
de claves y el de modificación lo usa para las claves obligatorias. `storyResponse` gana
`EstimatedHours *float64 \`json:"estimated_hours"\`` y los dos handlers construyen la respuesta con
`newStoryResponse`, de modo que la forma del JSON no pueda divergir entre operaciones.
`errorResponse` y `writeJSON` se quedan donde están: ya son compartidos dentro del paquete.

**Alternativas consideradas.** (a) Todo en `handler.go`: un archivo con dos handlers y sus helpers, más
difícil de revisar. (b) Mover `errorResponse`/`writeJSON`/`storyResponse` a un `json.go` nuevo: más limpio
conceptualmente, pero infla el diff del slice con líneas puramente movidas sin beneficio funcional.
(c) Un decodificador genérico `decodeStoryObject[T any](body) (T, map[string]json.RawMessage, error)`: válido
en Go 1.27, pero `target any` imita la API de `encoding/json` que el equipo ya conoce.

**Rationale.** El diff mínimo con la mayor cobertura de duplicación real: lo que se comparte es lo que
*tiene* que coincidir entre creación y modificación (la política de decodificación y la forma de la
respuesta). Todo esto ocurre como paso **REFACTOR** del slice del handler, con los tests de creación en verde
antes y después.

### Decisión 12: mapa de errores del handler

Orden de evaluación estricto (importa: `errors.As` de `ValidationError` antes que `errors.Is` de
`ErrStoryNotFound`, igual que en la creación):

| Condición | HTTP | `error` | Cuerpo |
|---|---|---|---|
| Método distinto de `PUT` (defensa; `ServeMux` ya filtra) | `405` | `method_not_allowed` | mensaje `only PUT is supported` |
| Fallo de decodificación: JSON malformado, cuerpo vacío, valor no-objeto, varios valores JSON, clave desconocida (incluidas `id`, `project_id`, `story_points`), tipo incorrecto, `null` dentro de los criterios | `400` | `invalid_request` | mensaje genérico, sin detalle interno |
| `*domain.ValidationError` (claves ausentes, UUID inválidos de la ruta, contenido, estado, estimación) | `422` | `validation_failed` | `fields` con una entrada por dato inválido |
| `application.ErrStoryNotFound` | `404` | `story_not_found` | mensaje genérico; no distingue inexistente de otro proyecto |
| Cualquier otro error (incluye `CHECK` violado, desborde numérico, base caída) | `500` | `internal_error` | mensaje genérico |
| Éxito | `200` | — | historia completa **tal como la devolvió `RETURNING`** |

## Flujo de datos

```
PUT /projects/{project_id}/stories/{story_id}
        │
        ▼
UpdateStoryHandler  ── método ≠ PUT ─────────────────────────────► 405
  decodeStoryObject ── malformado / desconocido / tipo malo ─────► 400 invalid_request
        │  (input tipado + mapa de claves crudas)
        │  claves ausentes = seis obligatorias − claves presentes
        ▼
UpdateStoryCommand{ProjectID, StoryID, ...valores, EstimatedHours *float64, MissingFields}
        │
        ▼
UpdateStoryUseCase.Execute
  ├─ fields ← "is required" por cada clave ausente
  ├─ fields ← project_id / story_id si uuid.Parse falla
  ├─ fields ← domain.NewStoryUpdate(...)  ◄── validateStoryContent (compartido con NewStory)
  │                                        ├── conjunto cerrado de status
  │                                        └── estimación: > 0, ≤ 99999.99, ≤ 2 decimales, nil válido
  ├─ len(fields) > 0 ──► *domain.ValidationError ──────────────► 422 validation_failed + fields
  │                      (CERO llamadas al puerto)
  ▼
StoryUpdater.Update(ctx, story)            [puerto]
        │
        ▼
PostgresStoryRepository.Update
  UPDATE stories SET <6 campos editables>
  WHERE id = $1 AND project_id = $2
  RETURNING <9 columnas>
        ├─ pgx.ErrNoRows ──► application.ErrStoryNotFound ──────► 404 story_not_found
        ├─ otro error ─────────────────────────────────────────► 500 internal_error
        ▼
   domain.Story almacenada (project_id y story_points de la base)
        │
        ▼
newStoryResponse(story) ─────────────────────────────────────────► 200 + historia completa
```

Propiedades que el flujo garantiza por construcción, no por chequeos adicionales:

- **Ninguna escritura ante input inválido**: el puerto se invoca después de la validación completa.
- **`id`, `project_id`, `story_points` inmutables**: no están en el `SET`; `id` y `project_id` vienen sólo de
  la ruta; el cuerpo que los mencione muere en la decodificación estricta.
- **La respuesta no puede mentir**: sale de `RETURNING`, no del objeto que construyó el dominio.
- **No encontrada ≡ de otro proyecto**: ambas son cero filas en la misma sentencia.

## Cambios por archivo

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/story/domain/story.go` | Modificar | `EstimatedHours *float64` en `Story`; constantes `StatusPending`/`StatusInProgress`/`StatusCompleted` y `AllowedStatuses()`; `validateStoryContent` extraída de `NewStory`; `NewStoryUpdate`; `NewStory` usa `StatusPending` en lugar del literal. |
| `internal/story/application/update_story.go` | Crear | `ErrStoryNotFound`, `UpdateStoryCommand` (con `MissingFields`), puerto `StoryUpdater`, `UpdateStoryUseCase`, `NewUpdateStoryUseCase`, `Execute` con la fusión de `fields` y la normalización de UUID. |
| `internal/story/application/create_story.go` | Sin cambios | `StoryRepository`, `CreateStoryCommand` y `CreateStoryUseCase` quedan intactos. |
| `internal/story/infrastructure/postgres/repository.go` | Modificar | `Update(ctx, story) (domain.Story, error)`: una sentencia `UPDATE ... WHERE id AND project_id ... RETURNING`; `pgx.ErrNoRows` → `application.ErrStoryNotFound`; ningún otro error se reinterpreta. |
| `internal/project/infrastructure/postgres/migrations/000003_add_story_estimated_hours.up.sql` | Crear | Columna `NUMERIC(7,2) NULL` con `stories_estimated_hours_positive`, más `stories_status_check`. |
| `internal/project/infrastructure/postgres/migrations/000003_add_story_estimated_hours.down.sql` | Crear | Elimina el `CHECK` de estado y la columna (destructivo para las estimaciones). |
| `internal/story/transport/http/update_handler.go` | Crear | `UpdateStoryHandler`, `NewUpdateStoryHandler`, `updateStoryRequest`, detección de claves obligatorias y el mapa de errores de la Decisión 12. |
| `internal/story/transport/http/handler.go` | Modificar | `decodeCreateStoryRequest` → `decodeStoryObject(body, target)`; `storyResponse` gana `estimated_hours`; nuevo `newStoryResponse` usado por ambos handlers. |
| `internal/api/api.go` | Modificar | `StoryDependencies.Updater storyapplication.StoryUpdater`; registro condicional de `PUT /projects/{project_id}/stories/{story_id}`. |
| `cmd/api/main.go` | Modificar | Tres ramas de esquema (`>= 3`, `>= 2`, resto) con su log; el repositorio Postgres se inyecta como `Repository` y como `Updater`. |
| `tests/unit/story/domain/story_test.go` | Modificar | Casos de `NewStoryUpdate`; los casos de `NewStory` se preservan como red del REFACTOR; `estimated_hours` inicial `nil` en la creación. |
| `tests/unit/story/application/update_story_test.go` | Crear | Fake de `StoryUpdater`; cero llamadas ante input inválido, exactamente una ante input válido, fusión de `fields`, propagación de errores. |
| `tests/unit/story/application/create_story_test.go` | Modificar | Aserción aditiva de `EstimatedHours == nil` en la historia creada. |
| `tests/unit/story/transport/http/update_handler_test.go` | Crear | Tabla de escenarios HTTP con `SetPathValue("project_id"/"story_id")`, caso de uso real sobre un updater falso. |
| `tests/unit/story/transport/http/handler_test.go` | Modificar | La respuesta de creación ahora incluye `"estimated_hours": null`. |
| `tests/unit/cmd/api/main_test.go` | Modificar | Fake de updater; ruta registrada sólo con `Updater` presente; se preservan las aserciones `405` de la ruta de colección y el caso sólo-proyectos. |
| `tests/integration/story/postgres/repository_integration_test.go` | Modificar | Agregar `000003_...up.sql` a la lista de migraciones (línea 371); tests de `Update` de punta a punta, round-trip de la estimación, `story_points` sembrado por SQL, constraints por nombre. |
| `tests/integration/story/postgres/http_integration_test.go` | Modificar | Escenario de arranque `version three`; crear → modificar `200`; historia inexistente `404`; con `version 2` la ruta no existe y la creación sigue funcionando. |
| `README.md` | Modificar | Operación, conjunto de `status`, reglas de la estimación, migración `000003` requerida, `estimated_hours` en la respuesta de creación y la limitación de última escritura gana. |
| `openspec/specs/historia/spec.md` | Modificar al archivar | Se aplica el delta ya escrito en `specs/historia/spec.md` de esta carpeta. |

## Interfaces y contratos

Sólo firmas: la implementación corresponde a la fase de aplicación.

### Dominio — `internal/story/domain/story.go`

```go
// Story gana la estimación de esfuerzo, independiente de los Story Points.
type Story struct {
	ID                 string
	ProjectID          string
	Title              string
	Description        string
	Priority           string
	Status             string
	StoryPoints        *int
	AcceptanceCriteria []string
	EstimatedHours     *float64
}

const (
	StatusPending    = "pendiente"
	StatusInProgress = "en_progreso"
	StatusCompleted  = "completada"
)

// AllowedStatuses devuelve una copia del conjunto cerrado de estados, en orden.
func AllowedStatuses() []string

// NewStory conserva su firma actual.
func NewStory(id, projectID, title, description, priority string, criteria []string) (Story, error)

// NewStoryUpdate valida el contenido editable completo de una historia existente.
// estimatedHours nil significa "sin estimación" y es válido.
func NewStoryUpdate(id, projectID, title, description, priority, status string, criteria []string, estimatedHours *float64) (Story, error)

// validateStoryContent acumula en fields las reglas de contenido compartidas por creación y modificación.
func validateStoryContent(fields map[string]string, title, description, priority string, criteria []string)

// validateStatus acumula en fields el conjunto cerrado de estados.
func validateStatus(fields map[string]string, status string)

// validateEstimatedHours acumula en fields las reglas de la estimación (> 0, <= 99999.99, <= 2 decimales).
func validateEstimatedHours(fields map[string]string, estimatedHours *float64)
```

Mensajes de `fields` (contrato observable por los tests y por la spec):

| Campo | Mensaje |
|---|---|
| clave obligatoria ausente | `is required` |
| `title`, `description` | `is required` |
| `priority` | `must be alta, media or baja` |
| `status` | `must be pendiente, en_progreso or completada` |
| `acceptance_criteria` | `must contain at least one non-blank criterion` / `must contain only non-blank criteria` |
| `estimated_hours` | `must be greater than 0, at most 99999.99 and have at most 2 decimals` |
| `project_id`, `story_id` | `must be a valid UUID` |

### Aplicación — `internal/story/application/update_story.go`

```go
// ErrStoryNotFound identifica una historia inexistente o perteneciente a otro proyecto.
var ErrStoryNotFound = errors.New("story not found")

// UpdateStoryCommand contiene el reemplazo completo del contenido editable.
// MissingFields nombra las claves obligatorias que el input no suministró.
type UpdateStoryCommand struct {
	ProjectID          string
	StoryID            string
	Title              string
	Description        string
	Priority           string
	Status             string
	AcceptanceCriteria []string
	EstimatedHours     *float64
	MissingFields      []string
}

// StoryUpdater reemplaza el contenido editable de una historia de un proyecto y devuelve la fila almacenada.
type StoryUpdater interface {
	Update(ctx context.Context, story domain.Story) (domain.Story, error)
}

// UpdateStoryUseCase valida por completo antes de una única escritura.
type UpdateStoryUseCase struct {
	updater StoryUpdater
}

func NewUpdateStoryUseCase(updater StoryUpdater) *UpdateStoryUseCase

// Execute devuelve *domain.ValidationError con todos los campos inválidos y sin tocar el puerto,
// ErrStoryNotFound si la historia no existe bajo ese proyecto, o la historia almacenada.
func (u *UpdateStoryUseCase) Execute(ctx context.Context, command UpdateStoryCommand) (domain.Story, error)
```

### Persistencia — `internal/story/infrastructure/postgres/repository.go`

```go
// Update reemplaza el contenido editable de una historia acotada por id y project_id.
// Cero filas afectadas es ErrStoryNotFound; ningún otro error se reinterpreta.
func (r *PostgresStoryRepository) Update(ctx context.Context, story domain.Story) (domain.Story, error)
```

`*PostgresStoryRepository` satisface `application.StoryRepository` y `application.StoryUpdater`.

### Transporte — `internal/story/transport/http/`

```go
// NewUpdateStoryHandler construye el adaptador HTTP de modificación; el routing se compone en internal/api.
func NewUpdateStoryHandler(useCase *application.UpdateStoryUseCase) http.Handler

type updateStoryRequest struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"`
}

// storyResponse gana estimated_hours; la respuesta de creación pasa a emitir null (cambio aditivo).
type storyResponse struct {
	ID                 string   `json:"id"`
	ProjectID          string   `json:"project_id"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Priority           string   `json:"priority"`
	Status             string   `json:"status"`
	StoryPoints        *int     `json:"story_points"`
	AcceptanceCriteria []string `json:"acceptance_criteria"`
	EstimatedHours     *float64 `json:"estimated_hours"`
}

func decodeStoryObject(body io.Reader, target any) (map[string]json.RawMessage, error)
func newStoryResponse(story domain.Story) storyResponse
```

### Composición — `internal/api/api.go`

```go
type StoryDependencies struct {
	Repository storyapplication.StoryRepository
	GenerateID storyapplication.IDGenerator
	Updater    storyapplication.StoryUpdater // nil: la ruta de modificación no se registra
}

// NewHTTPHandler conserva su firma; registra la modificación sólo si Updater no es nil.
func NewHTTPHandler(repository application.ProjectRepository, generateID application.IDGenerator, stories ...StoryDependencies) http.Handler
```

### Contrato HTTP

| Método y ruta | Cuerpo | Éxito | Errores |
|---|---|---|---|
| `PUT /projects/{project_id}/stories/{story_id}` | objeto con las seis claves obligatorias (`estimated_hours` puede ser `null`) | `200` + historia completa (9 campos) | `400 invalid_request`, `404 story_not_found`, `405 method_not_allowed`, `422 validation_failed` + `fields`, `500 internal_error` |
| `POST /projects/{project_id}/stories` | sin cambios | `201` + historia completa, ahora con `"estimated_hours": null` | sin cambios |

## Estrategia de pruebas

TDD estricto (`openspec/config.yaml`: `strict_tdd: true`, `test_command: go test ./...`). Cada slice es un
**commit de unidad de trabajo** con Conventional Commits que lleva sus tests y su documentación, y cuyo
mensaje registra la evidencia del ciclo: **RED** (el test que falla y la falla observada), **GREEN** (la
implementación mínima y el pase observado), **TRIANGULATE** (los casos adicionales que fijan el
comportamiento) y **REFACTOR** (el cambio estructural y la corrida en verde observada). Sin evidencia
fabricada: un test de integración saltado por falta de Docker se registra como **saltado**, nunca como
prueba de persistencia.

### Por capa

| Capa | Qué se prueba | Cómo |
|---|---|---|
| Unitario — dominio | Reglas de contenido reutilizadas, conjunto cerrado de estados, reglas y bordes de la estimación, texto y orden preservados, `nil` válido | Tests de tabla con `t.Run` sobre funciones puras; sin dobles |
| Unitario — aplicación | Cero llamadas al puerto ante cualquier input inválido, exactamente una ante input válido, fusión completa de `fields`, normalización de UUID, propagación de `ErrStoryNotFound` y de errores inesperados | Fake de `StoryUpdater` escrito a mano con contador de llamadas y story capturada |
| Unitario — transporte | Mapa de errores completo, detección de claves obligatorias, `null` vs ausente en `estimated_hours`, forma exacta del JSON de respuesta | `httptest` + `SetPathValue`; caso de uso **real** sobre un updater falso (convención del handler de creación existente) |
| Unitario — composición | La ruta existe sólo con `Updater` presente; creación, proyectos y los `405` de la ruta de colección intactos | `api.NewHTTPHandler` con fakes, sin base de datos |
| Integración — almacenamiento | Persistencia real, round-trip de `NUMERIC(7,2)` ↔ `*float64` y de `story_points` nullable, borrado a `NULL`, `project_id`/`story_points` intactos, cero filas, constraints por nombre | Testcontainers `postgres:16-alpine` con las migraciones `000001`–`000003`; `story_points` sembrado por SQL directo |
| Integración — arranque | Rutas expuestas según `version`/`dirty`; crear → modificar de punta a punta | Binario compilado por el propio test contra la base migrada (patrón existente de `http_integration_test.go`) |

Los tests de integración conservan su política actual: `testing.Short()` salta explícitamente, y un fallo al
levantar el contenedor es `t.Fatalf`, nunca un salto silencioso.

### Mapeo de los cinco slices a sus archivos y seams

Cada slice es autocontenido, verificable y revertible sin remover nada más. **No** se elige aquí el orden de
entrega en PRs: sólo el orden técnico de construcción.

**Slice 1 — Dominio.** Seam: funciones puras de `domain`; sin dobles.
- Archivos de test: `tests/unit/story/domain/story_test.go`.
- RED: `NewStoryUpdate` con título/descripción vacíos y en blanco (ASCII y Unicode), cada prioridad válida e
  inválida, cada estado del conjunto y fuera del conjunto (`"cancelada"`, `"Completada"`, `"en progreso"`,
  `""`), criterios `nil`/`[]`/con entrada en blanco, estimación `nil` (válida), `0`, negativa, `1.001`,
  `0.005`, `100000`, `99999.991`, y texto y orden de criterios preservados.
- TRIANGULATE: bordes `0.01` y `99999.99` aceptados; entero `3` y un decimal `2.5`; blancos Unicode
  (` `, ` `); acumulación de varios campos inválidos en un mismo `ValidationError`.
- REFACTOR: extraer `validateStoryContent` de `NewStory` y sustituir el literal `"pendiente"` por
  `StatusPending`, con los 14 casos de `NewStory` en verde antes y después.
- Evidencia de no regresión: `go test ./...` completo, no sólo el paquete `domain`.

**Slice 2 — Aplicación.** Seam: la interfaz `StoryUpdater`.
- Archivos de test: `tests/unit/story/application/update_story_test.go` (nuevo);
  `create_story_test.go` gana la aserción aditiva `EstimatedHours == nil`.
- RED: input inválido → `*domain.ValidationError` y `updater.calls == 0`; input válido → exactamente una
  llamada con `id`/`project_id` canónicos y los seis campos; `ErrStoryNotFound` y un error inesperado se
  propagan con `errors.Is` y devuelven `domain.Story{}`; no se genera ningún identificador.
- TRIANGULATE (la fusión de `fields`, caso por caso de la spec): `MissingFields` con las seis claves →
  seis `is required`; una sola clave ausente → un solo campo; `story_id` inválido + `title` vacío → los dos
  campos; ambos identificadores inválidos → `project_id` y `story_id`; clave ausente cuyo valor cero también
  violaría el contenido → gana `is required`.
- REFACTOR: unificar la construcción del mapa de `fields` si los tests muestran duplicación.

**Slice 3 — Almacenamiento (requiere Docker).** Seam: PostgreSQL real.
- Archivos de test: `tests/integration/story/postgres/repository_integration_test.go`, agregando
  `000003_add_story_estimated_hours.up.sql` a la lista de migraciones de `storyDatabase` (línea 371).
- RED, **antes** de escribir `Update`, y en este orden porque el primero resuelve el riesgo más alto:
  1. round-trip de la estimación en los bordes `0.01` y `99999.99` y con un entero, leyendo la columna por SQL
     y por el `RETURNING` (resuelve los riesgos (a) codec `numeric` y (b) `NULL` → `**float64`, incluido el
     de `story_points`);
  2. actualización completa persistida con el orden de los criterios preservado;
  3. `null` borra una estimación previa y deja SQL `NULL`, no `0`;
  4. `project_id` sin cambios y `story_points` (sembrado por SQL) sin cambios;
  5. proyecto equivocado → `ErrStoryNotFound` con la fila original íntegra;
  6. historia inexistente → `ErrStoryNotFound`;
  7. `stories_status_check` y `stories_estimated_hours_positive` existen con la definición esperada
     (consulta a `pg_constraint`, como el test existente del FK) y rechazan una escritura directa inválida;
  8. un error de base no relacionado no se etiqueta como `ErrStoryNotFound`.
- GREEN: `000003` up/down y `Update`. Si el test 1 falla, aplicar la escalera de la Decisión 6 **en orden** y
  registrar en el mensaje del commit qué paso fue necesario y por qué.
- Si Docker no está disponible: el slice queda **sin verificar**, se reporta como saltado y no puede
  declararse hecho.

**Slice 4 — Handler HTTP.** Seam: `http.Handler` + `httptest`, con el caso de uso real sobre un updater falso.
- Archivos de test: `tests/unit/story/transport/http/update_handler_test.go` (nuevo);
  `handler_test.go` actualizado para el `"estimated_hours": null` de la creación.
- RED: `200` con los nueve campos, incluidos `story_points` y `estimated_hours`; `400` por JSON malformado,
  cuerpo vacío, valor no-objeto (`[]`, `"texto"`), dos objetos consecutivos, clave desconocida (`owner`),
  `id`/`project_id`/`story_points` en el cuerpo, tipo incorrecto (`title` numérico,
  `acceptance_criteria: "criterio"`, `estimated_hours: "8"` / `true` / objeto) y `null` dentro de los
  criterios; `422` por UUID de ruta inválidos, claves obligatorias ausentes (una, y el cuerpo `{}`) y errores
  de dominio, siempre con `updater.calls == 0`; `404`; `500` genérico sin detalle interno; `405` para otros
  métodos.
- TRIANGULATE: `estimated_hours` presente con `null` → `200` y `null` en la respuesta, frente a
  `estimated_hours` ausente → `422 is required`; `Content-Type: application/json` en todas las respuestas.
- REFACTOR: generalizar `decodeCreateStoryRequest` a `decodeStoryObject` e introducir `newStoryResponse`,
  con toda la suite de creación en verde antes y después.

**Slice 5 — Composición y documentación.** Seam: `api.NewHTTPHandler` con fakes, y el binario real contra la
base migrada.
- Archivos de test: `tests/unit/cmd/api/main_test.go`; `tests/integration/story/postgres/http_integration_test.go`.
- RED unitario: con `Updater` presente, `PUT` sobre la ruta del ítem responde `200` y hay exactamente una
  actualización; con `Updater` nil, la misma ruta responde `404` y la creación sigue funcionando; los `405`
  existentes de `GET`/`PUT`/`DELETE` sobre `/projects/{id}/stories` siguen en verde; `/projects/{id}/stories/`
  responde `404`.
- RED de integración: nuevo escenario `version three` en la tabla de
  `TestAPIStartupRoutesFollowMigrationState` (modificación disponible); `version two` conserva la creación y
  la ruta de modificación no existe; `dirty` en versión 3 no expone la ruta; de punta a punta, crear →
  modificar `200` con `project_id` intacto, y una historia inexistente → `404`.
- GREEN: `api.go`, las tres ramas de `main.go`, README y confirmación del delta de spec.
- Regla de despliegue que el slice codifica: la ruta no se expone antes de que `000003` esté aplicada y
  limpia.

### Presupuesto de revisión

El pronóstico de la propuesta (~900–1300 líneas) excede el presupuesto de 400 líneas, y este diseño no lo
reduce: lo confirma en cinco unidades de trabajo autocontenidas. **`Decision needed before apply: Yes`** — la
estrategia de cadena (`stacked-to-main` o `feature-branch-chain`) la elige el usuario y **no** se decide en
este documento.

## Matriz de amenazas

Aplicable de forma parcial: el cambio agrega una ruta HTTP (routing de aplicación), pero no ejecuta comandos
de shell, subprocesos de negocio, automatización de VCS/PR ni clasificación de archivos ejecutables. Las
filas de la matriz de referencia, que cubren límites de shell/VCS/PR, se marcan `N/A` con su razón, y se
agregan las filas de routing HTTP que sí aplican a este cambio.

| Límite | Casos adversariales mínimos | Aplicabilidad | Respuesta de diseño | Tests RED planificados |
|---|---|---|---|---|
| Rutas tipo documentación (`requirements.txt`, Markdown ejecutable, `README.sh`) | clasificación y ejecución | **N/A**: el cambio no clasifica ni ejecuta archivos; sólo agrega SQL, Go y documentación | — | — |
| Selección de repositorio Git (`git -C`, rutas relativas/absolutas) | autoridad de cwd | **N/A**: no hay automatización de Git en el código de producto; los commits los hace una persona | — | — |
| Estado del commit (staged, `commit -a`, índice vacío) | semántica de índice/worktree | **N/A**: ninguna ruta de código toca el índice de Git | — | — |
| Estado del push (rama de tracking, primer push, refspec explícita) | resolución de destino | **N/A**: el cambio no hace push ni resuelve refs | — | — |
| Comandos de PR (`--head`, prefijo de entorno, comandos compuestos) | composición de argumentos | **N/A**: sin automatización de PR en el producto | — | — |
| **Routing HTTP: solapamiento de patrones** | `PUT /projects/{id}/stories` (colección), `/projects/{id}/stories/` (`story_id` vacío), `/projects/{id}/stories/a/b` | **Aplicable** | El comodín de `ServeMux` no matchea un segmento vacío ni dos segmentos; la colección conserva su `405` y la ruta del ítem con id vacío da `404` | Slices 4 y 5: `405` de colección preservado, `404` para `/stories/` |
| **Routing HTTP: identificadores de la ruta como input hostil** | `story_id` = `"abc"`, cadena vacía, UUID de otro proyecto, UUID en mayúsculas | **Aplicable** | Los identificadores se parsean como UUID y se normalizan antes de la escritura; sólo viajan como parámetros de consulta, nunca concatenados en SQL | Slices 2 y 4: `422` con `fields.story_id`/`fields.project_id`; slice 3: UUID de otro proyecto → `ErrStoryNotFound` |
| **Routing HTTP: identidad tomada del cuerpo** | `id`, `project_id`, `story_points` en el cuerpo, con valor propio o ajeno | **Aplicable** | Decodificación estricta (`DisallowUnknownFields`) los rechaza; además no están en el `SET` | Slice 4: `400 invalid_request` por cada una; slice 3: `project_id` y `story_points` intactos tras una modificación válida |
| **Superficie de SQL** | comillas, `;`, `--` dentro de `title`, `description` o un criterio | **Aplicable** | Una única sentencia con parámetros posicionales; ningún fragmento de SQL se construye con datos del cliente | Slice 3: un título con comillas y punto y coma hace round-trip sin alterar la tabla |
| **Fuga de información en los errores** | historia de otro proyecto, `CHECK` violado, base caída | **Aplicable** | `404 story_not_found` indistinguible; `500` con mensaje genérico; ningún detalle de base en el cuerpo | Slice 4: mismos código y cuerpo para inexistente y de otro proyecto; `500` sin detalle interno |

## Migración y despliegue

Orden obligatorio, porque la ruta depende de la columna:

1. Mergear los slices de dominio, aplicación, almacenamiento y handler. Ninguno expone la ruta: el sistema
   sigue comportándose como US-05.
2. Aplicar la migración `000003` en el entorno (proceso externo al binario, como `000001` y `000002`).
3. Mergear o desplegar el slice de composición. El binario lee `schema_migrations` al arrancar: con
   `version >= 3 && !dirty` registra la ruta; en caso contrario la omite y lo registra en el log.

Un despliegue en el orden equivocado no rompe nada: sin `000003`, la ruta simplemente no existe (`404`) y la
creación sigue funcionando. No hay feature flag adicional; la versión de esquema **es** el flag.

Rollback (detalle completo en `proposal.md`): primero dejar de exponer la ruta (revertir el slice de
composición o correr contra un esquema en versión 2); recién después, y sólo tras confirmar que ninguna
estimación vale la pena conservar, ejecutar el `down` de `000003`, que es destructivo. Las filas con
`en_progreso` o `completada` deben volver a `pendiente` de forma deliberada, nunca en silencio.

## Riesgos de diseño

| Riesgo | Probabilidad | Mitigación en este diseño |
|---|---|---|
| El round-trip de pgx `NUMERIC(7,2)` ↔ `*float64` falla o pierde precisión | Media | Es el **primer** test RED del slice 3; escalera de fallback en tres pasos (Decisión 6) que arregla la conversión antes de degradar el esquema |
| Escanear columnas nullable (`story_points`, `estimated_hours`) en destinos `*T` no había ocurrido nunca en este código | Media | Cubierto por el mismo test RED; el paso 2 de la escalera usa intermediarios `pgtype`/`sql.Null*` explícitos |
| La regla de "≤ 2 decimales" sobre `float64` produce falsos rechazos | Media | Se decide sobre la representación decimal (`FormatFloat`), no con aritmética; casos `0.07`, `0.01`, `99999.99`, `1.001`, `0.005` fijados en los tests del slice 1 |
| El REFACTOR que extrae `validateStoryContent` toca código de US-05 | Media | Paso REFACTOR posterior a GREEN, con los 14 casos de `NewStory` y la suite completa en verde antes y después; sin cambio de comportamiento en el mismo paso |
| `MissingFields` en un comando de aplicación es un préstamo del vocabulario del transporte | Baja | Documentado y acotado a "estos datos no fueron suministrados"; la alternativa (punteros + `**float64`) queda registrada si el equipo prefiere pagar ese precio |
| Validar UUID en el caso de uso (modificación) y en el handler (creación) es inconsistente | Media | Deliberado y documentado (Decisión 4); unificar la creación está fuera de alcance y queda como pregunta abierta |
| El conjunto de `status` fue inventado en la exploración | Media | Vive en una constante de dominio más un `CHECK` nombrado; corregirlo es una migración acotada, no un cambio disperso |
| Agregar `estimated_hours` a `storyResponse` cambia la respuesta de creación | Alta | Cambio aditivo (`null`); `newStoryResponse` garantiza que ambas operaciones no puedan divergir; tests, README y delta de spec se actualizan en los slices que lo introducen |
| Última escritura gana: dos modificaciones concurrentes se pisan | Media | Fuera de alcance explícito y documentado en README y spec; el diseño no insinúa versionado que después haya que sostener |
| La cobertura del slice 3 depende de Docker | Media | Los saltos se reportan como saltos; el slice no puede declararse verificado sin contenedor |
| El pronóstico excede el presupuesto de 400 líneas | Alta | Cinco unidades de trabajo autocontenidas ya delimitadas; la estrategia de cadena la decide el usuario antes de aplicar |

## Preguntas abiertas

- [ ] **Estrategia de cadena de PRs** (`stacked-to-main` o `feature-branch-chain`): decisión del usuario,
      necesaria **antes** de aplicar. Este documento no la elige (US-05 usó `stacked-to-main`).
- [ ] `{"title": null}` (y lo mismo para `description`, `priority`, `status`, `acceptance_criteria`) se
      resuelve como `422` por regla de contenido, porque `encoding/json` decodifica `null` al valor cero sin
      error. ¿Debería ser `400 invalid_request` como los demás tipos incorrectos? Default de este diseño:
      `422`. Ningún escenario de la spec lo contradice.
- [ ] Claves JSON duplicadas (`{"title":"a","title":"b"}`): `encoding/json` toma la última y la modificación
      quedaría con `"b"`. Comportamiento heredado de la creación, no especificado. Default: se acepta la
      última, sin tratamiento especial.
- [ ] La validación de formato de `project_id` queda en el handler para la creación y en el caso de uso para
      la modificación. ¿Unificar moviendo la de creación al caso de uso, como cambio de seguimiento? Fuera
      del alcance de US-06.
- [ ] Si la escalera de la Decisión 6 llega al paso 3 (`DOUBLE PRECISION`), se pierde la garantía de dos
      decimales en la base. ¿Se acepta con el dominio como única autoridad, o se prefiere almacenar centésimas
      enteras? Sólo se decide si el test RED lo fuerza.
