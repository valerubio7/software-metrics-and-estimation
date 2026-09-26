# Exploración: US-07 — Consultar el Product Backlog (issue #35)

## Historia

Como usuario del sistema quiero consultar el Product Backlog de un proyecto para conocer las historias existentes y decidir sobre su planificación y desarrollo.

No incluye crear historias (US-05), modificarlas (US-06) ni asignarlas a un Sprint (US-09).

## Estado actual

- US-05 (crear) y US-06 (modificar) están archivadas; la especificación vigente está en `openspec/specs/historia/spec.md`. **No existe ninguna lectura de historias.**
- Dominio (`internal/story/domain/story.go`): `Story{ID, ProjectID, Title, Description, Priority (alta|media|baja), Status (pendiente|en_progreso|completada), StoryPoints *int, AcceptanceCriteria []string, EstimatedHours *float64}`. **No existe concepto de orden**: no hay posición, ranking, fecha de creación ni secuencia.
- Migraciones 000001–000003 (`internal/project/infrastructure/postgres/migrations/`): tabla `stories` con `id UUID PK`, `project_id` FK `NOT NULL ON DELETE RESTRICT`, `title`, `description`, `priority` con `CHECK`, `status` con `CHECK`, `story_points INT NULL`, `acceptance_criteria TEXT[]`, `estimated_hours NUMERIC(7,2) NULL`. Los IDs son `uuid.NewString()` (v4, aleatorios), por lo que ordenar por `id` **no** equivale a orden cronológico.
- `PostgresStoryRepository` solo tiene `Create` y `Update`; no hay `FindByID` ni listado.
- Puertos de aplicación separados: `StoryRepository{Create}` y `StoryUpdater{Update}`. Errores existentes: `ErrProjectNotFound` (creación, vía FK 23503) y `ErrStoryNotFound`.
- Transporte HTTP: `storyResponse`, `newStoryResponse`, `errorResponse`, `writeJSON` y `decodeStoryObject` son privados del paquete y ya los reutiliza `update_handler.go`; un handler nuevo en el mismo paquete los obtiene sin cambios. Convención de errores: `422 validation_failed` con `fields.project_id` por UUID inválido, `404 project_not_found` y `500 internal_error`.
- Composición: `internal/api/api.go` define `StoryDependencies{Repository, GenerateID, Updater (opcional)}` y registra `POST /projects/{project_id}/stories` y `PUT .../{story_id}` con patrones de `ServeMux` (go 1.27.0). `cmd/api/main.go` decide con un `switch` sobre `schema_migrations` (`>=3`, `>=2`, otro).
- `FindByID` (mencionado en la exploración de US-06): US-07 pide un listado por proyecto, no la lectura individual. **No se introduce** `FindByID` ni `GET .../{story_id}`; la spec vigente exige 405 sobre el ítem.
- Tests afectados: `tests/unit/cmd/api/main_test.go` líneas 117 (`GET` sobre la colección → 405) y 173 (`GET/PUT/DELETE` sobre la colección → 405). `GET` pasa a 200/404; `PUT` y `DELETE` siguen en 405. El escenario "La ruta de colección conserva su comportamiento" de la spec `historia` queda inexacto y requiere un delta `MODIFIED`. Los fakes existentes solo implementan `Create`/`Update`, por lo que un puerto de lectura separado evita romperlos.

## Orden hoy

No está definido. Solo `priority` tiene semántica de orden (alta > media > baja) y, con solo 3 valores, hay muchos empates sin un dato estable para desempatar. La observación de la historia sugiere que la priorización o el reordenamiento manual pertenece a una historia posterior.

## Áreas afectadas

- `internal/story/application/list_stories.go` (nuevo): puerto `StoryLister`/`ListByProject`, `ListStoriesUseCase`, reutiliza `ErrProjectNotFound`, valida el UUID antes de tocar el puerto.
- `internal/story/infrastructure/postgres/repository.go`: `ListByProject` ordenado.
- `internal/story/transport/http/list_handler.go` (nuevo): `GET /projects/{project_id}/stories`.
- `internal/api/api.go`: campo `Lister` opcional en `StoryDependencies`.
- `cmd/api/main.go`: gate de migración (`>=4` si se agrega la migración 000004).
- Migración `000004` (solo si el orden usa una secuencia de creación).
- Tests unitarios e integración, `tests/unit/cmd/api/main_test.go`, `README.md` y spec `historia` (delta al archivar).

## Decisiones abiertas

**D1 — Definición del orden (decisión de producto, sin resolver).**

| Opción | Descripción | Costo |
|---|---|---|
| (a) | Prioridad (alta > media > baja) con desempate por orden de creación. Requiere columna `seq GENERATED ALWAYS AS IDENTITY`, migración 000004 y gate `>=4`. | Medio |
| (b) | Solo prioridad, desempate por `id` (sin migración; desempate aleatorio). | Bajo |
| (c) | Orden manual con columna `position`. Toca la creación de US-05 y exige una operación de reordenar que ninguna historia pide. | Alto |

Default recomendado: (a).

**D2 — Formato de respuesta.** Contenedor `{"project_id": "...", "stories": [...]}` (extensible) frente a un arreglo desnudo; cada elemento usa el `storyResponse` completo. Una lista vacía debe salir como `[]` y no como `null` (un slice `nil` de Go se serializa como `null`).

**D3 — Proyecto inexistente frente a lista vacía.** `404 project_not_found` si el proyecto no existe; `200` con `stories: []` si existe sin historias; `422` con `fields.project_id` por UUID inválido. Detección atómica con una sola consulta `projects LEFT JOIN stories`, sin dependencia entre módulos. Pregunta menor P2: ¿basta con la lista vacía para "informar que no existen elementos" o se espera un mensaje textual? Default: lista vacía (el mensaje es responsabilidad de la capa de presentación).

**D4 — Paginación.** No en US-07 (no se pide; el contenedor deja la extensión aditiva). Riesgo aceptado: respuesta sin límite.

**D5 — Contrato.** `GET /projects/{project_id}/stories`, sin parámetros de consulta; códigos 200/404/405/422/500; `HEAD` queda cubierto por el mux.

**D6 — Solo lectura.** La consulta es un `SELECT`. Se prueba con un fake que solo expone lectura y, en integración, con una instantánea de la tabla antes y después.

**D7 — "Seleccionar un proyecto existente".** Se identifica con `project_id` en la ruta; no se agrega `GET /projects`. Pregunta menor P3, default: no.

## Enfoques

| # | Enfoque | Pros | Contras | Esfuerzo |
|---|---|---|---|---|
| A | SQL ordenado (`CASE priority`, luego `seq`), migración 000004, puerto `StoryLister` separado, `LEFT JOIN` | Orden determinista y con sentido; consulta atómica; no rompe fakes | Migración y gate `>=4`; `seq` de filas previas arbitrario; regla de prioridad en SQL solo verificable con Docker | Medio |
| B | Igual, sin migración: `ORDER BY CASE priority, id` | Menor tamaño; el gate sigue en `>=3` | Desempate sin significado (UUID aleatorio) | Bajo |
| C | Columna `position` manual | Fiel a un backlog priorizado a mano | Toca US-05, carrera al asignar posición, exige reordenar; excede el alcance | Alto |
| A' | Regla de orden como función de dominio (`SortBacklog`) | Testeable sin Docker; encaja con hexagonal y TDD estricto | Requiere cargar `seq` en `domain.Story`; orden en memoria | Medio |

## Recomendación

Enfoque A, condicionado a la respuesta de D1. B es la variante mínima sin migración. Si D1 resulta (c), hay que replantear el alcance. Patrón: puerto de lectura separado, `Lister` opcional en `StoryDependencies`, gate de migración en `main.go`, sin `FindByID` y sin paginación.

## Rebanado TDD sugerido (un commit por paso)

1. Aplicación (~130 líneas).
2. Almacenamiento (~200–250 líneas; requiere Docker; migración si se elige A).
3. Handler HTTP (~150 líneas).
4. Composición y documentación (~150 líneas).

Pronóstico: A ≈ 550–750 líneas cambiadas; B ≈ 450–600. Ambos superan el presupuesto de 400 líneas.

## Riesgos

- La estrategia `single-pr` choca con el presupuesto de 400 líneas: hace falta aceptar explícitamente una excepción `size:exception` antes de `sdd-apply`.
- D1 abierta: un orden equivocado implica una migración posterior; el desempate por `id` no es cronológico.
- Con la opción A, las historias existentes reciben `seq` en orden físico arbitrario.
- `GET` sobre la colección deja de ser 405: actualizar tests y delta de spec, o quedarán tests verdes por motivos equivocados.
- Riesgo de `null` en lugar de `[]` para lista vacía.
- El orden real y la detección de proyecto inexistente solo se demuestran con PostgreSQL (Docker); los tests de integración omitidos no prueban nada.
- Respuesta sin límite por falta de paginación.
- El `CASE` de SQL duplica el conjunto de prioridades del dominio y del `CHECK`.

## Decisión del usuario

D1 resuelta (2026-09-26): orden por **prioridad (alta > media > baja) con desempate por orden de creación** (enfoque A: migración 000004 con columna `seq`, gate `>=4`).

## Listo para propuesta

Sí. D1 ya está resuelta (ver arriba). D3/P2 y D7/P3 tienen defaults seguros (lista vacía y sin listado de proyectos).
