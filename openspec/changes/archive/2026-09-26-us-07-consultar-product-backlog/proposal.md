# Propuesta: US-07 — Consultar el Product Backlog

Fuente del requisito: [issue #35 — "US-07 — Consultar el Product Backlog"](https://github.com/valerubio7/software-metrics-and-estimation/issues/35).
Artefacto de entrada: `openspec/changes/us-07-consultar-product-backlog/exploration.md` (incluye la sección
"Decisión del usuario").

## Intención

US-05 permite crear historias y US-06 permite modificarlas, pero el sistema no ofrece ninguna forma de
leerlas: una vez registradas, las historias de un proyecto solo son visibles en la respuesta de la
operación que las escribió. Quien gestiona el proyecto no puede conocer el contenido actual del Product
Backlog y, por lo tanto, no puede decidir qué planificar o desarrollar a continuación.

Este cambio agrega la consulta del Product Backlog de un proyecto existente: una operación de solo lectura
que devuelve todas sus historias en un orden determinista y con sentido para la planificación —primero por
prioridad (`alta` > `media` > `baja`) y, dentro de la misma prioridad, por orden de creación—. El éxito
significa que, dado un proyecto existente, se obtienen sus historias completas en ese orden; que un
proyecto sin historias se distingue claramente de un proyecto inexistente; y que la consulta nunca
modifica datos.

## Alcance

### Dentro del alcance

Mapeado al alcance del issue #35:

- Una operación de consulta `GET /projects/{project_id}/stories` que devuelve `200` con un contenedor
  `{"project_id": "...", "stories": [...]}`, donde cada elemento usa la representación completa de
  historia ya existente (`storyResponse`, incluyendo `status`, `story_points` y `estimated_hours`).
- Orden de las historias: prioridad (`alta` > `media` > `baja`) y, como desempate, orden de creación.
  El desempate se apoya en una nueva columna `seq` (`GENERATED ALWAYS AS IDENTITY`) agregada por la
  migración `000004`; `seq` no se expone en la API.
- Proyecto existente sin historias: `200` con `"stories": []` (nunca `null`). La lista vacía es la forma de
  "informar que no existen elementos"; cualquier mensaje textual es responsabilidad de la capa de
  presentación.
- Superficie de errores: `422 validation_failed` con `fields.project_id` cuando el identificador no es un
  UUID válido (sin tocar el almacenamiento), `404 project_not_found` cuando el proyecto no existe y
  `500 internal_error` con mensaje genérico ante un fallo inesperado del almacenamiento.
- Detección de proyecto inexistente frente a proyecto vacío en una única consulta atómica
  (`projects LEFT JOIN stories`), sin dependencia de código entre los módulos `story` y `project`.
- Solo lectura: la consulta es un `SELECT`; no escribe, no bloquea filas y no altera ninguna historia.
- Un puerto de aplicación de lectura **separado** (`StoryLister`) y su caso de uso, para no romper los
  puertos y fakes existentes de creación y modificación.
- Exposición de la ruta de consulta condicionada a `version >= 4 && !dirty` del esquema; la modificación
  mantiene su gate `>= 3` y la creación su gate `>= 2`.
- `PUT` y `DELETE` sobre la ruta de colección siguen respondiendo `405`; `GET` deja de ser `405` y pasa a
  ser la consulta (con `HEAD` cubierto por el `ServeMux`).
- Tests (unitarios e integración con Testcontainers), documentación de la operación y de la migración
  `000004` en el README, y el delta de la spec `historia`.

### Fuera del alcance

- Crear historias (US-05), modificarlas (US-06) y asignarlas a un Sprint (US-09).
- Reordenamiento manual o priorización explícita del backlog (columna `position` u operación de
  reordenar): corresponde a una historia posterior.
- Paginación, filtros, parámetros de consulta y ordenamientos alternativos. El contenedor deja la
  extensión aditiva para el futuro.
- Lectura individual de una historia (`GET /projects/{project_id}/stories/{story_id}`) y un método
  `FindByID`: la ruta del ítem sigue respondiendo `405` a `GET`, como exige la spec vigente.
- Un listado de proyectos (`GET /projects`): el proyecto se selecciona por `project_id` en la ruta.
- Exponer `seq` o una fecha de creación en la respuesta.
- Elegir cómo se resuelve el exceso sobre el presupuesto de 400 líneas (ver Entrega más abajo): esa
  decisión le corresponde al usuario.

## Capacidades

### Capacidades nuevas

Ninguna. La consulta del Product Backlog pertenece a la capacidad existente `historia`.

### Capacidades modificadas

- `historia`: hoy especifica la creación y la modificación, y no existe ningún requisito de lectura.
  Los requisitos cambian en cuatro aspectos: (1) debe existir una consulta de solo lectura que devuelve
  todas las historias de un proyecto existente, con su representación completa; (2) esas historias se
  devuelven ordenadas por prioridad (`alta` > `media` > `baja`) y, a igual prioridad, por orden de
  creación; (3) un proyecto existente sin historias devuelve una lista vacía, un proyecto inexistente se
  rechaza con `404 project_not_found` y un identificador inválido con `422`; (4) la consulta solo está
  disponible con el esquema en versión 4 o superior sin estado `dirty`. Además, el escenario "La ruta de
  colección conserva su comportamiento" pasa a ser `MODIFIED`: `GET` sobre la colección deja de responder
  `405`, mientras `PUT` y `DELETE` lo siguen haciendo. Los requisitos de creación y modificación siguen
  vigentes sin cambios de comportamiento.

## Enfoque

Enfoque A de la exploración, confirmado por la decisión D1 del usuario: **SQL ordenado por prioridad y
secuencia de creación, migración `000004`, puerto de lectura separado y detección atómica del proyecto.**

1. **Aplicación.** Un nuevo `list_stories.go` con el puerto `StoryLister` (`ListByProject`), un
   `ListStoriesUseCase` y la reutilización de `ErrProjectNotFound`. El caso de uso valida el UUID de
   `project_id` antes de tocar el puerto, de modo que un identificador inválido produce cero llamadas al
   almacenamiento. El resultado vacío se normaliza a un slice no `nil`.
2. **Persistencia.** La migración `000004` agrega `seq BIGINT GENERATED ALWAYS AS IDENTITY` a `stories`
   (con un índice que respalde el orden por proyecto, a definir en diseño). `PostgresStoryRepository`
   implementa `ListByProject` con una sola consulta `projects LEFT JOIN stories` filtrada por
   `project_id`, ordenada por `CASE priority` y luego `seq`: cero filas significa proyecto inexistente y
   una única fila con columnas de historia nulas significa proyecto sin historias. La creación de US-05 no
   cambia: la base asigna `seq` automáticamente.
3. **HTTP.** Un `ListStoriesHandler` en un `list_handler.go` del mismo paquete de transporte, que reutiliza
   `storyResponse`, `newStoryResponse`, `errorResponse` y `writeJSON` sin cambios, y serializa
   `stories: []` para la lista vacía.
4. **Composición y docs.** `StoryDependencies` gana un campo `Lister` opcional; `internal/api/api.go`
   registra `GET /projects/{project_id}/stories` solo cuando se provee. `cmd/api/main.go` agrega un caso
   `version >= 4 && !dirty` que provee creación, modificación y consulta; los casos `>= 3` y `>= 2` quedan
   como hoy. El README y el delta de la spec `historia` documentan la operación, el orden, la lista vacía,
   la migración requerida y la ausencia de paginación.

Si la regla de orden por prioridad debe ser verificable sin Docker, el diseño puede evaluar la variante A'
(función de dominio de ordenamiento); esta propuesta no la exige y mantiene el orden en SQL.

El texto de los deltas de spec sigue la convención existente en español de `openspec/specs/historia/spec.md`,
y los documentos de planificación de esta carpeta de cambio se redactan en español
(`technical_artifacts_language: es` en `openspec/config.yaml`).

### Entrega y TDD

TDD estricto activo (`go test ./...`). Cada slice se entrega como un **commit de unidad de trabajo** con
Conventional Commits, con sus tests y su documentación en el mismo commit y una traza breve de la
evidencia TDD (RED / GREEN / TRIANGULATE / REFACTOR). Sin evidencia fabricada: un test de integración
saltado por falta de Docker se registra como saltado, nunca como prueba.

Slices sugeridos, en orden: (1) aplicación (~130 líneas), (2) almacenamiento y migración `000004`
(~200–250, requiere Docker), (3) handler HTTP (~150), (4) composición, gate `>= 4`, README y delta de spec
(~150).

La estrategia de entrega es `single-pr` (un único pull request al final, sin dividir). El pronóstico
(**~550–750 líneas cambiadas**) **excede el presupuesto de revisión de 400 líneas**, por lo que:

- `Decision needed before apply: Yes` — la implementación requiere que el usuario acepte explícitamente
  una excepción `size:exception` para el único PR. El orquestador la consultará en la fase de tareas;
  esta propuesta no la da por aceptada.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/story/application/list_stories.go` | Nuevo | Puerto `StoryLister` (`ListByProject`), `ListStoriesUseCase`, validación de UUID previa al puerto, reutilización de `ErrProjectNotFound`. |
| `internal/story/infrastructure/postgres/repository.go` | Modificado | `ListByProject` con una consulta `projects LEFT JOIN stories` ordenada por `CASE priority` y `seq`. |
| `internal/project/infrastructure/postgres/migrations/000004_*.{up,down}.sql` | Nuevo | Columna `seq GENERATED ALWAYS AS IDENTITY` (e índice de soporte); la migración down borra la columna. |
| `internal/story/transport/http/list_handler.go` | Nuevo | `ListStoriesHandler` para `GET /projects/{project_id}/stories`, con contenedor y `stories: []`. |
| `internal/api/api.go` | Modificado | Campo opcional `Lister` en `StoryDependencies`; registro condicional de la ruta `GET`. |
| `cmd/api/main.go` | Modificado | Nuevo caso `version >= 4 && !dirty`; los gates `>= 3` y `>= 2` se conservan. |
| `tests/unit/story/**` | Nuevo | Tests de caso de uso y handler con un fake que solo expone lectura. |
| `tests/unit/cmd/api/main_test.go` | Modificado | Las aserciones de `GET` sobre la colección (líneas ~117 y ~173) pasan de `405` a `200`/`404`; `PUT` y `DELETE` siguen en `405`. |
| `tests/integration/story/postgres/*_integration_test.go` | Modificado | Agregar `000004` a la lista de migraciones; cubrir orden, proyecto vacío, proyecto inexistente y ausencia de escrituras. |
| `README.md` | Modificado | Documentar la consulta, el orden, la lista vacía, la migración `000004` y la ausencia de paginación. |
| `openspec/specs/historia/spec.md` | Modificado (al archivar) | Delta con los requisitos de consulta y el escenario `MODIFIED` de la ruta de colección. |
| Creación (US-05), modificación (US-06), dominio de proyecto | Sin cambios | Sin cambio de comportamiento; `seq` lo asigna la base. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| El pronóstico (~550–750 líneas) excede el presupuesto de 400 líneas con estrategia `single-pr`. | Alta | Se requiere una excepción `size:exception` explícita que el orquestador consultará al usuario en la fase de tareas, antes de `sdd-apply`; los commits de unidad de trabajo mantienen el PR revisable por pasos. |
| Las historias existentes reciben `seq` en orden físico arbitrario al aplicar `000004`, no en su orden real de creación. | Media | Se documenta como limitación aceptada: el desempate cronológico solo es exacto para historias creadas después de la migración; el orden principal por prioridad no se ve afectado. |
| `GET` sobre la colección deja de responder `405`: los tests actuales podrían quedar en verde por motivos equivocados o romperse. | Alta | Actualizar explícitamente las aserciones de `main_test.go` y el escenario de spec como `MODIFIED`, manteniendo `PUT`/`DELETE` en `405`. |
| Una lista vacía serializada como `null` en lugar de `[]`. | Media | Normalizar a slice no `nil` y fijarlo con un test de handler y uno de integración. |
| El `CASE priority` en SQL duplica el conjunto de prioridades del dominio y del `CHECK` de la base. | Media | Cubrir el orden completo `alta`/`media`/`baja` en integración; evaluar en diseño si conviene una constante compartida o la variante A'. |
| El orden real y la detección de proyecto inexistente solo se demuestran contra PostgreSQL (Docker). | Media | Los tests de integración saltados se reportan como saltados y nunca cuentan como verificación. |
| Orden de despliegue: la ruta requiere la migración `000004`. | Baja | El gate `version >= 4 && !dirty` hace que un despliegue sin migrar simplemente carezca de la ruta, con creación y modificación intactas. |
| Respuesta sin límite por la ausencia de paginación. | Baja | Riesgo aceptado para el volumen actual; el contenedor permite agregar paginación de forma aditiva. |

## Plan de rollback

- **Antes de aplicar `000004` en cualquier entorno**: revertir los commits del cambio (o el PR mergeado).
  Ningún otro código depende de ellos; creación y modificación no se tocan.
- **Después de exponer la ruta**: revertir el commit de composición, o correr contra una base en versión 3,
  quita `GET /projects/{project_id}/stories` y deja creación y modificación intactas; el esquema puede
  quedar en versión 4 mientras tanto, porque la columna `seq` es transparente para el código anterior.
- **Rollback de esquema**: ejecutar la migración down `000004`, que borra la columna `seq` (y su índice).
  Solo se pierde la secuencia de desempate, no datos de negocio: ninguna historia ni proyecto se borra o
  modifica. Si luego se vuelve a aplicar, las filas existentes reciben una nueva secuencia arbitraria.
- Nunca borrar historias ni proyectos como parte de un rollback.

## Dependencias

- US-05 y US-06 archivadas: aportan la tabla `stories`, la estructura de módulos, `storyResponse` y los
  helpers de transporte que se reutilizan.
- La migración PostgreSQL `000004` debe aplicarse antes de que la ruta esté disponible.
- Docker es necesario para los tests de integración con Testcontainers.
- Aceptación explícita de `size:exception` por parte del usuario antes de aplicar, porque el pronóstico
  excede el presupuesto de revisión con estrategia `single-pr`.

## Criterios de éxito

- [ ] Para un proyecto existente con historias, `GET /projects/{project_id}/stories` responde `200` con el
      contenedor `{"project_id", "stories"}` y todas sus historias con la representación completa.
- [ ] Las historias salen ordenadas por prioridad (`alta` > `media` > `baja`) y, a igual prioridad, por
      orden de creación, verificado contra PostgreSQL.
- [ ] Un proyecto existente sin historias responde `200` con `"stories": []` (nunca `null`).
- [ ] Un proyecto inexistente responde `404 project_not_found`; un `project_id` que no es UUID responde
      `422 validation_failed` con `fields.project_id` sin tocar el almacenamiento; un fallo inesperado
      responde `500 internal_error` sin detalles internos.
- [ ] La consulta no modifica ninguna fila (instantánea de la tabla antes y después en integración).
- [ ] Solo se devuelven historias del proyecto indicado, nunca de otros proyectos.
- [ ] La ruta no existe con el esquema en versión 3 y está disponible en versión 4; creación y modificación
      siguen funcionando en ambos casos; `PUT` y `DELETE` sobre la colección y `GET` sobre el ítem siguen
      en `405`.
- [ ] `go test ./...` pasa y cada slice aterriza como un commit de unidad de trabajo con su traza TDD, tests
      y docs incluidos.
- [ ] El README y el delta de spec `historia` documentan la consulta, el orden, la lista vacía, la
      migración `000004` y la ausencia de paginación.
