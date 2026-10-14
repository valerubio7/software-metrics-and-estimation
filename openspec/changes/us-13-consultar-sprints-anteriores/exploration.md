# Exploración: US-13 — Consultar Sprints anteriores (issue #41)

## Historia

Como usuario del sistema quiero consultar los Sprints anteriores de un proyecto para revisar el trabajo realizado y analizar la evolución del proyecto.

Criterios de aceptación del issue #41 (EPIC-03, sin comentarios ni etiquetas):

1. Se puede seleccionar un proyecto existente.
2. El sistema muestra los Sprints registrados previamente del proyecto.
3. Cada Sprint se identifica con claridad.
4. Se puede ver el Sprint Goal.
5. Se pueden ver las historias planificadas para el Sprint.
6. Se puede identificar cuáles de esas historias se completaron.
7. Si no existen Sprints anteriores, el sistema informa al usuario.
8. La consulta no modifica información histórica.

Incluye: navegar el historial, ver objetivo y trabajo realizado de cada Sprint y preservar la información histórica. No incluye: crear Sprints (US-08), cerrarlos (US-12) ni calcular métricas del Sprint (US-19). La consigna (`docs/assignment.md`, sección 3) solo dice "Consultar Sprints anteriores".

## Estado actual

- **Sprint** (`internal/sprint/domain/sprint.go`): `Sprint{ID, ProjectID, SprintGoal}`. Tabla `sprints` (migración `000005`): `id UUID PK`, `project_id` FK `ON DELETE RESTRICT`, `sprint_goal TEXT NOT NULL`; `000007` agrega `UNIQUE (id, project_id)` y `000008` agrega `is_closed BOOLEAN NOT NULL DEFAULT false`. **No existe fecha de inicio, fecha de fin, nombre, número, `created_at`, `closed_at` ni secuencia.** US-08 excluyó explícitamente fechas, duración, estado y orden. El `id` es `uuid.NewString()` (v4), por lo que ordenar por `id` no es cronológico.
- **"Anterior" no está definido en ningún artefacto.** La única señal de ciclo de vida es `is_closed` (US-12). `is_closed` no está en el struct de dominio; se lee con SQL crudo en el repositorio (`CloseSprint`). La spec vigente solo dice que el Sprint cerrado "MUST seguir disponible para consultas posteriores".
- **Historias del Sprint**: `sprint_stories(sprint_id, story_id, project_id, completed_at TIMESTAMPTZ NULL)` (migraciones `000007` y `000010`). Una historia puede estar en varios Sprints, así que "completada" es un hecho por Sprint (`completed_at IS NOT NULL`), no `stories.status`, que es global, editable con `PUT` y puede divergir (decisión de US-11). Las asignaciones y finalizaciones se rechazan en Sprints cerrados (`FOR UPDATE` sobre `sprints`), por lo que el contenido de un Sprint cerrado es **estable**; solo `tasks` aún admite inserciones sobre un Sprint cerrado (fuera del alcance de US-13).
- **Story Points**: `stories.story_points INT NULL`. **Ninguna operación lo escribe**: `Create` inserta `NULL` y `Update` no lo toca (la estimación/Planning Poker no está implementada). Hoy es siempre `NULL`.
- **Orden de historias**: `stories.seq` (migración `000004`, `GENERATED ALWAYS AS IDENTITY`, `UNIQUE (project_id, seq)`) da el orden de creación; `sprint_stories` no tiene columna de orden ni de fecha de asignación.
- **Lecturas existentes**: ninguna sobre Sprints. El patrón vigente es US-07 (`internal/story`): puerto de lectura separado (`StoryLister`), `Lister` opcional en `StoryDependencies`, `projects LEFT JOIN stories` en una sola consulta (cero filas = proyecto inexistente; fila con `id` NULL = lista vacía), contenedor `{"project_id","stories":[]}` con slice no nulo, validación del UUID en la capa de aplicación (`*domain.ValidationError`), 404 `project_not_found`, 422, 405 para métodos distintos de `GET`/`HEAD`, 500 genérico. No tiene paginación.
- **Transporte de Sprint** (`internal/sprint/transport/http/handler.go`): `sprintResponse`, `errorResponse`, `writeJSON` y los handlers de creación y cierre son del paquete `transporthttp`; un handler nuevo en el mismo paquete los reutiliza. El `storyResponse` del módulo `story` es privado de su paquete, así que Sprint necesita su propio DTO.
- **Composición** (`internal/api/api.go`): `SprintDependencies{Repository, GenerateID, Closer (opcional)}`; `MigrationReadiness{..., Sprints (>=5), SprintClose (>=8), ...}`. `cmd/api/main.go` arma las dependencias con `ResolveMigrationReadiness`. Cada puerto opcional se registra solo si su campo no es nil.
- **Rutas hoy**: `POST /projects/{project_id}/sprints` (crear) y `POST .../sprints/{sprint_id}/closure`. `GET` sobre la colección `/sprints` responde 405 (el mux, porque existe el patrón `POST`); `GET .../sprints/{sprint_id}` responde 404 (ninguna ruta coincide con esa forma).
- **Issue #41 / GitHub**: `gh` no estaba disponible en la sesión de exploración; el texto del issue se leyó por la web pública (solo lectura).

## Qué significa "anterior"

| Definición | Pros | Contras |
|---|---|---|
| (a) Sprint cerrado (`is_closed = true`) | Única señal real y atómica; la spec de US-12 ya promete la consulta posterior; el contenido está congelado | Un Sprint abierto, pero ya "pasado" por calendario, no aparece; no hay forma de listar el Sprint en curso |
| (b) Todo Sprint del proyecto | Más simple; sin filtro | "Anterior" pierde significado; el Sprint en curso aparece mezclado |
| (c) Por fecha de fin vencida | Cercano al lenguaje Scrum | **Imposible hoy**: no existen fechas en `sprints`; exigiría rediseñar US-08/US-12 |

(a) es la única definición implementable sin ampliar el modelo de ciclo de vida. Cuando no hay Sprints cerrados, la respuesta es una lista vacía (criterio 7).

## Qué devolver

Por Sprint cerrado: `id`, `project_id`, `sprint_goal`, `is_closed` (siempre `true`; permite extender luego a abiertos) y `stories`.

Por historia planificada en ese Sprint: `id`, `title`, `priority`, `story_points` (nullable, hoy siempre `null`) y `completed_at` (nullable, ISO 8601 UTC) más un booleano `completed` derivado. Se evita `stories.status` por la divergencia documentada en US-11. `description` y `acceptance_criteria` se omiten por defecto para no inflar la respuesta (pregunta abierta).

Fechas: el Sprint no tiene ninguna; el único instante disponible es `completed_at` por historia. No se agregan totales ni porcentajes (son métricas de US-19, excluidas por el issue); cada historia expone sus puntos para que el cliente o US-19 los sumen.

## Forma del endpoint

| # | Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|---|
| A | `GET /projects/{project_id}/sprints` devuelve los Sprints cerrados con sus historias anidadas | Una llamada cubre los criterios 2 a 7; igual al patrón de US-07; una sola consulta SQL | Respuesta sin límite; trae historias aunque no se pidan | Medio |
| B | `GET /projects/{project_id}/sprints` (resumen sin historias) más `GET .../sprints/{sprint_id}` (detalle con historias) | Respuesta de lista liviana; recurso direccionable | Dos endpoints, dos casos de uso, doble superficie de 404 y de tests; el detalle de un Sprint abierto exige otra decisión | Alto |
| C | Mover la lectura de historias al módulo `story` (`GET .../sprints/{sprint_id}/stories`) | Reutiliza `storyResponse` | Rompe la cohesión (la lista de Sprints vive en `sprint`); sigue necesitando una lista | Medio-alto |

**Recomendación: A**, con el contenedor `{"project_id": "...", "sprints": [...]}` (extensible a paginación o filtros sin romper el contrato). Una lista vacía sale como `[]` y no como `null`.

Contrato propuesto: `GET /projects/{project_id}/sprints`, sin parámetros de consulta; `200` / `404 project_not_found` / `405` / `422 validation_failed` (con `fields.project_id`) / `500 internal_error`. `HEAD` queda cubierto por el mux.

## Orden y paginación

- Falta un dato estable de orden: se necesita **una migración nueva `000011`** análoga a `000004`: `ALTER TABLE sprints ADD COLUMN seq BIGINT GENERATED ALWAYS AS IDENTITY` y `UNIQUE (project_id, seq)`; el `down` elimina ambos. Los Sprints existentes reciben `seq` en orden físico arbitrario (mismo riesgo aceptado en US-07).
- Dirección: para un historial lo natural es **más reciente primero (`seq DESC`)**; el patrón de US-07 usa creación ascendente. Es una decisión de producto (pregunta abierta). Las historias dentro de cada Sprint salen por `stories.seq` ascendente.
- `seq` ordena por **creación**, no por cierre. Si interesa el orden de cierre, haría falta `closed_at` (alternativa abajo).
- Paginación: no se pide ni US-07 la incluyó; el contenedor la deja como extensión aditiva. Riesgo aceptado: respuesta sin límite.

Alternativa evaluada: `closed_at TIMESTAMPTZ NULL` (fijado en `CloseSprint`). Aporta una fecha real y orden por cierre, pero modifica la mutación de US-12 (su spec limita la mutación al indicador), los Sprints ya cerrados quedarían con `NULL` y se mezclaría con un cambio de alcance. Se descarta para US-13 salvo decisión de producto contraria.

## Persistencia

Consulta única, atómica y de solo lectura, que distingue proyecto inexistente de lista vacía por la forma del resultado:

```sql
SELECT sp.id, sp.sprint_goal, st.id, st.title, st.priority, st.story_points, ss.completed_at
FROM projects p
LEFT JOIN sprints sp ON sp.project_id = p.id AND sp.is_closed
LEFT JOIN sprint_stories ss ON ss.sprint_id = sp.id
LEFT JOIN stories st ON st.id = ss.story_id
WHERE p.id = $1
ORDER BY sp.seq DESC, st.seq
```

Cero filas: `ErrProjectNotFound`. Una fila con `sp.id` NULL: sin Sprints cerrados. Se agrupan las filas por Sprint en Go. `is_closed` va en la condición `ON` del `LEFT JOIN` (si estuviera en `WHERE`, un proyecto sin Sprints cerrados se confundiría con uno inexistente). Los `JOIN` cruzan tablas de otro módulo; ya hay precedente (`CloseSprint` lee `projects`; el repositorio de `task` lee `sprint_stories`). Los escaneos requieren punteros para los `NULL` del `LEFT JOIN`, como `listedStory` en US-07.

## Arquitectura (hexagonal)

- `internal/sprint/domain`: tipos de lectura (`ClosedSprint`/`SprintStory`); sin reglas de negocio nuevas. Reutiliza el `ValidationError` existente.
- `internal/sprint/application/list_sprints.go` (nuevo): puerto `SprintHistoryReader.ListClosedByProject(ctx, projectID) ([]domain.ClosedSprint, error)` **separado** de `SprintRepository` (el fake de `tests/unit/cmd/api/main_test.go` solo implementa `Create`; así no se rompe), `ListSprintsUseCase` que valida el UUID antes de tocar el puerto y reutiliza `ErrProjectNotFound`.
- `internal/sprint/infrastructure/postgres/repository.go`: `ListClosedByProject` y aserción `var _ application.SprintHistoryReader = (*PostgresSprintRepository)(nil)`.
- `internal/sprint/transport/http/list_handler.go` (nuevo): `GET`/`HEAD`; DTOs propios; slices no nulos.
- `internal/api/api.go`: campo opcional `Lister` en `SprintDependencies`; registro de `GET /projects/{project_id}/sprints` solo si no es nil; nuevo flag `SprintHistory` (`version >= 11`) en `MigrationReadiness`.
- `cmd/api/main.go`: asignar `sprintDependencies.Lister = repository` bajo el nuevo flag y registrar el log de disponibilidad.

## Gate de versión del esquema

Disponible solo con esquema limpio en versión 11 o superior. Con versión inferior, estado `dirty` o error de resolución, la ruta no se registra. Matiz importante: como `POST /projects/{project_id}/sprints` sí existe, el `GET` sobre esa colección responde **405** (no 404), igual que el caso de US-07 con el esquema por debajo de 4. La spec delta debe decirlo así y no copiar el "responde 404" de US-12, donde la ruta de cierre estaba completamente ausente.

## Áreas afectadas

- `internal/project/infrastructure/postgres/migrations/000011_add_sprint_creation_sequence.{up,down}.sql` (nuevo).
- `internal/sprint/domain/sprint.go`, `application/list_sprints.go` (nuevo), `infrastructure/postgres/repository.go`, `transport/http/list_handler.go` (nuevo).
- `internal/api/api.go`, `cmd/api/main.go`.
- Tests unitarios nuevos en `tests/unit/sprint/{application,transport/http}` y `tests/unit/cmd/api`; integración Testcontainers (requiere Docker) en `tests/integration/sprint/postgres` (agregar `000011` a `sprintMigrationNames`).
- `tests/integration/migrations/migration_files_test.go`: hoy exige exactamente diez pares (`000001`–`000010`); pasa a once.
- `tests/unit/cmd/api/main_test.go` línea 87: el bucle `GET/PUT/DELETE` → 405 sobre `/sprints` sigue siendo cierto solo mientras no haya `Lister`; con `Lister` el `GET` pasa a 200 (ajustar el test para el caso con y sin `Lister`). `tests/unit/sprint/transport/http/handler_test.go` línea 49 prueba `CreateSprintHandler` aislado y no cambia.
- `openspec/specs/sprint/spec.md` (delta al archivar), `openspec/config.yaml` (lista de historias implementadas) y documentación del endpoint (hay un precedente en `docs/sprint-closure.md`; no existe `README.md` con rutas).

## Casos de error y comportamiento

| Caso | Respuesta |
|---|---|
| `project_id` no es UUID | `422 validation_failed`, `fields.project_id`; sin consultar el almacenamiento |
| Proyecto inexistente | `404 project_not_found` |
| Proyecto sin Sprints cerrados (aunque tenga uno abierto) | `200` con `sprints: []` |
| Sprint cerrado sin historias asignadas | `200`; el Sprint aparece con `stories: []` |
| Método distinto de `GET`/`HEAD` | `405 method_not_allowed` |
| Esquema < 11, `dirty` o error de resolución | Ruta no registrada; `GET` sobre la colección → `405` |
| Fallo inesperado de almacenamiento | `500 internal_error` genérico, sin lista parcial |

Solo lectura: se prueba con un fake que solo expone lectura y, en integración, con una instantánea de `sprints`, `sprint_stories`, `stories` y `tasks` antes y después.

## Rebanado TDD sugerido (un commit por paso)

1. Migración `000011` + actualización de `migration_files_test` (~40 líneas).
2. Aplicación: puerto, caso de uso, validación del UUID (~140 líneas con tests).
3. Almacenamiento: consulta y agrupación (~350 líneas con tests de integración; requiere Docker).
4. Handler HTTP (~250 líneas con tests).
5. Composición, gate y documentación (~200 líneas con tests).

Pronóstico: 900 a 1100 líneas cambiadas, superior al presupuesto de 400. Con `delivery_strategy=single-pr` hace falta aceptar explícitamente una excepción `size:exception` antes de `sdd-apply` (como en US-07 y TUI-02), o dividir en PR encadenados (migración + aplicación / persistencia / transporte y composición).

## Preguntas abiertas de producto

1. **Qué es "anterior"**: ¿solo Sprints cerrados (recomendado) o todos los del proyecto? ¿Debe poder consultarse también el Sprint abierto en curso?
2. **Fechas**: el modelo no tiene ninguna. ¿Basta con `completed_at` por historia, o se quiere registrar `closed_at`/fechas de inicio y fin (cambia US-08/US-12 y exige migración con datos nulos para Sprints ya cerrados)?
3. **Orden**: ¿más reciente primero (recomendado para historial) o cronológico ascendente? ¿Por creación (`seq`) o por cierre (`closed_at`)?
4. **Forma**: ¿una sola lista con historias anidadas (recomendado) o lista resumida más detalle por Sprint?
5. **Campos de historia**: ¿bastan `id`, `title`, `priority`, `story_points`, `completed_at`/`completed`, o también `description`, `acceptance_criteria`, `estimated_hours`? ¿Se expone `stories.status`?
6. **Story Points**: hoy siempre `null` porque nada los escribe. ¿Se devuelven igual (recomendado) o se espera la estimación?
7. **Totales por Sprint** (cantidad planificada/completada, puntos): el issue los excluye por US-19; confirmar que no se agregan.
8. **Paginación**: ¿se acepta una respuesta sin límite por ahora (recomendado)?
9. **Tareas, esfuerzos y defectos** del Sprint: el issue no los pide; confirmar que no se incluyen.
10. **Mensaje explícito para lista vacía** (criterio 7): ¿basta con `sprints: []` (recomendado; el texto es de la capa de presentación) o se espera un mensaje en la API?
11. **Entrega**: ¿`single-pr` con `size:exception` o PR encadenados?

## Recomendación

Enfoque A: `GET /projects/{project_id}/sprints` devuelve los Sprints **cerrados** del proyecto, con sus historias planificadas y la marca de finalización por historia, desde una única consulta de solo lectura. Migración `000011` con `sprints.seq` para un orden determinista, puerto de lectura separado, `Lister` opcional en `SprintDependencies` y gate de esquema `>=11`. Sin fechas nuevas, sin totales, sin paginación y sin endpoint de detalle.

## Riesgos

- "Anterior" y el orden no están definidos por producto; un orden equivocado implica otra migración (los `seq` previos son arbitrarios).
- Sin fechas en el Sprint, la consulta no puede mostrar cuándo ocurrió cada iteración; si producto lo exige, el alcance crece hacia US-08/US-12.
- `story_points` siempre `null`: la respuesta es válida pero puede parecer incompleta para quien analiza la evolución.
- Mover `is_closed` de `ON` a `WHERE` en el `LEFT JOIN` confunde "sin Sprints cerrados" con "proyecto inexistente"; las pruebas deben cubrir un proyecto con un solo Sprint abierto.
- `GET` sobre `/sprints` deja de ser 405 cuando hay `Lister`: actualizar `main_test.go` y el delta de spec para no dejar pruebas verdes por motivos equivocados.
- Esquema parcial: registrar la ruta antes de `000011` rompe en tiempo de solicitud; el gate debe ser independiente de `SprintClose`.
- Falta de paginación y de límite sobre filas de historias (una fila por historia asignada).
- La migración altera una tabla compartida con integraciones que tienen listas fijas de migraciones y un conteo exacto de diez pares.
- Los tests de integración (orden, proyecto inexistente frente a vacío, agrupación) requieren Docker; los omitidos no prueban nada.
- La estrategia `single-pr` choca con el presupuesto de 400 líneas.

## Listo para propuesta

Sí, con defaults seguros para las preguntas 1, 4, 5, 6, 7, 8, 9 y 10. Las preguntas 2, 3 y 11 conviene confirmarlas antes de `sdd-propose`, porque 2 y 3 determinan si la migración agrega solo `seq` o también `closed_at`.
