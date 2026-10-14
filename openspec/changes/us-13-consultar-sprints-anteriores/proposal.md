# Propuesta

Fuente del requisito: [issue #41 — "US-13 — Consultar Sprints anteriores"](https://github.com/valerubio7/software-metrics-and-estimation/issues/41).
Artefacto de entrada: `openspec/changes/us-13-consultar-sprints-anteriores/exploration.md`.

## Por qué

US-12 permite cerrar Sprints y su spec exige que el Sprint cerrado "siga disponible para consultas posteriores", pero el sistema no ofrece ninguna lectura de Sprints: una vez creado o cerrado, un Sprint solo es visible en la respuesta de la operación que lo escribió. Quien gestiona el proyecto no puede revisar el objetivo ni el trabajo realizado en iteraciones anteriores ni analizar la evolución del proyecto. US-13 agrega esa consulta de solo lectura sobre los datos históricos ya congelados por el cierre.

## Qué cambia

- Agregar `GET /projects/{project_id}/sprints`, que devuelve `200` con el contenedor `{"project_id": "...", "sprints": [...]}`.
- "Sprint anterior" significa Sprint cerrado (`sprints.is_closed = true`); los Sprints abiertos no se listan.
- Cada Sprint expone `id`, `project_id`, `sprint_goal`, `is_closed` (siempre `true`) y `stories`, con las historias planificadas (asignadas en `sprint_stories`).
- Cada historia expone `id`, `title`, `priority`, `story_points` (nullable; hoy siempre `null` porque ninguna operación lo escribe), `completed_at` (nullable, ISO 8601 UTC, tomado de `sprint_stories.completed_at`) y `completed` (booleano derivado de `completed_at`). No se usa `stories.status`, que es global y puede divergir (decisión de US-11).
- Orden: Sprints del más reciente al más antiguo por orden de creación (`sprints.seq DESC`); historias dentro de cada Sprint por `stories.seq` ascendente.
- Migración `000011` que agrega únicamente `sprints.seq BIGINT GENERATED ALWAYS AS IDENTITY` con `UNIQUE (project_id, seq)`; la migración `down` elimina ambos. `seq` no se expone en la API.
- Proyecto existente sin Sprints cerrados (aunque tenga uno abierto): `200` con `"sprints": []`, nunca `null`; un Sprint cerrado sin historias aparece con `"stories": []`. El texto informativo para la lista vacía es responsabilidad de la capa de presentación.
- Superficie de errores: `422 validation_failed` con `fields.project_id` si el identificador no es UUID (sin tocar el almacenamiento), `404 project_not_found` si el proyecto no existe, `405` para métodos distintos de `GET`/`HEAD` y `500 internal_error` genérico, sin lista parcial, ante un fallo inesperado del almacenamiento.
- Consulta única, atómica y de solo lectura (`projects LEFT JOIN sprints ... AND sp.is_closed`, con el filtro en la condición `ON` para distinguir proyecto inexistente de lista vacía); no escribe ni bloquea filas.
- Disponibilidad condicionada a esquema limpio en versión `>= 11`, con un flag de readiness independiente de `SprintClose`. Por debajo del gate la ruta no se registra: `GET` sobre la colección responde `405` con esquema limpio en versión 5 a 10 (existe `POST` sobre la misma ruta) y `404` si no hay rutas de Sprints (versión inferior a 5, estado `dirty` o error de resolución).

## Capacidades

### Capacidades nuevas

Ninguna. La consulta de Sprints anteriores pertenece a la capacidad existente `sprint`.

### Capacidades modificadas

- `sprint`: agregar los requisitos de consulta de Sprints anteriores (definición de "anterior" como Sprint cerrado, representación de Sprint e historias con la marca de finalización por Sprint, orden más reciente primero, lista vacía frente a proyecto inexistente, validación del identificador, solo lectura, métodos admitidos, fallo de almacenamiento y habilitación con esquema `>= 11` y `405`/`404` por debajo del gate). Los requisitos existentes de creación, asignación y cierre no cambian.

## Impacto

Afecta la API HTTP de Sprints, la capa de aplicación del módulo `sprint`, el repositorio PostgreSQL, la composición de dependencias condicionada por la versión del esquema y las pruebas unitarias y de integración:

| Área | Impacto | Descripción |
|---|---|---|
| `internal/project/infrastructure/postgres/migrations/000011_add_sprint_creation_sequence.{up,down}.sql` | Nuevo | Columna `sprints.seq` y restricción `UNIQUE (project_id, seq)`. |
| `internal/sprint/domain/sprint.go` | Modificado | Tipos de lectura para Sprint cerrado e historia planificada; sin reglas de negocio nuevas. |
| `internal/sprint/application/list_sprints.go` | Nuevo | Puerto de lectura separado de `SprintRepository` y caso de uso que valida el UUID antes del puerto y reutiliza `ErrProjectNotFound`. |
| `internal/sprint/infrastructure/postgres/repository.go` | Modificado | Consulta de Sprints cerrados con historias y agrupación por Sprint. |
| `internal/sprint/transport/http/list_handler.go` | Nuevo | Handler `GET`/`HEAD` con DTOs propios y slices no nulos. |
| `internal/api/api.go`, `cmd/api/main.go` | Modificado | Dependencia opcional de lectura en `SprintDependencies`, flag de readiness `>= 11`, registro condicional y log de disponibilidad. |
| `tests/unit/sprint/**`, `tests/unit/cmd/api/main_test.go` | Nuevo/Modificado | Caso de uso, handler y readiness; el bucle que espera `405` para `GET` sobre `/sprints` se ajusta para los casos con y sin la dependencia de lectura. |
| `tests/integration/sprint/postgres/**`, `tests/integration/migrations/migration_files_test.go` | Modificado | Agregar `000011` a las listas fijas de migraciones y pasar de diez a once pares; cubrir orden, proyecto con solo un Sprint abierto, proyecto inexistente, agrupación y ausencia de escrituras (Docker). |
| Documentación del endpoint, `openspec/config.yaml` | Modificado | Documentar la consulta (precedente: `docs/sprint-closure.md`) y registrar US-13 como implementada. |
| `openspec/specs/sprint/spec.md` | Futuro | Delta en la fase de spec, consolidado al archivar. |

Fuera de alcance: crear Sprints (US-08), cerrarlos (US-12), métricas o totales por Sprint (US-19), listar Sprints abiertos o el Sprint en curso, endpoint de detalle por Sprint, fechas (`closed_at`, inicio o fin), paginación, filtros o parámetros de consulta, y tareas, esfuerzos o defectos del Sprint; tampoco se exponen `description`, `acceptance_criteria`, `estimated_hours` ni `stories.status`.

## Plan de reversión

- Revertir el commit de composición, o arrancar con el esquema por debajo de `11`, quita la ruta `GET` y deja intactas creación, asignación y cierre; la columna `seq` es transparente para el código anterior.
- Rollback de esquema: la migración `down` `000011` elimina `UNIQUE (project_id, seq)` y la columna `seq`. Solo se pierde la secuencia de orden, no datos de negocio; si se vuelve a aplicar, las filas existentes reciben una secuencia nueva en orden físico arbitrario.
- Nunca borrar Sprints, asociaciones ni finalizaciones como parte de un rollback.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Los Sprints existentes reciben `seq` en orden físico arbitrario al aplicar `000011`. | Media | Limitación aceptada y documentada (mismo criterio que US-07); el orden es exacto para Sprints creados después de la migración. |
| `seq` ordena por creación, no por cierre. | Baja | Decisión de producto confirmada; un orden por cierre requeriría `closed_at`, fuera de alcance. |
| Mover `is_closed` del `ON` al `WHERE` confundiría "sin Sprints cerrados" con "proyecto inexistente". | Media | Prueba de integración con un proyecto que solo tiene un Sprint abierto. |
| `GET` sobre `/sprints` deja de responder `405` cuando existe la dependencia de lectura. | Alta | Ajustar `main_test.go` para ambos casos y redactar el delta con `405` (versión 5 a 10) o `404` (sin rutas de Sprints) por debajo del gate. |
| `story_points` siempre `null` puede parecer incompleto. | Media | Documentarlo; el campo queda listo para cuando exista la estimación. |
| Respuesta sin límite (sin paginación). | Baja | Riesgo aceptado; el contenedor permite agregar paginación de forma aditiva. |
| La verificación de orden, agrupación y lista vacía depende de Docker. | Media | Las pruebas omitidas se reportan como omitidas, nunca como verificación. |
| El pronóstico (~900–1100 líneas) excede el presupuesto de 400 con `single-pr`. | Alta | Requiere aceptación explícita de `size:exception` antes de `sdd-apply`, o PR encadenados; los commits de unidad de trabajo mantienen el PR revisable por pasos. |

## Dependencias

- US-08, US-09, US-11 y US-12 integradas (`sprints`, `sprint_stories`, `completed_at`, `is_closed`; migraciones `000005` a `000010`). Reconfirmar que `000011` sigue libre al aplicar.
- Docker para las pruebas de integración con Testcontainers.
- Aceptación explícita de `size:exception` por parte del usuario antes de aplicar.

## Decisiones abiertas para el orquestador

- Excepción al presupuesto de 400 líneas con PR único (pronóstico ~900–1100 líneas) o cambio a PR encadenados.

## Criterios de éxito

- [ ] Para un proyecto con Sprints cerrados, `GET /projects/{project_id}/sprints` responde `200` con el contenedor `{"project_id", "sprints"}`, solo con Sprints cerrados de ese proyecto, del más reciente al más antiguo.
- [ ] Cada Sprint incluye `sprint_goal` y sus historias planificadas ordenadas por `stories.seq`, con `completed_at` y `completed` reflejando la finalización en ese Sprint.
- [ ] Un proyecto sin Sprints cerrados (incluido uno con un Sprint abierto) responde `200` con `"sprints": []`; un Sprint cerrado sin historias muestra `"stories": []`.
- [ ] Un proyecto inexistente responde `404 project_not_found`; un `project_id` no UUID responde `422 validation_failed` sin tocar el almacenamiento; un fallo inesperado responde `500 internal_error` genérico.
- [ ] La consulta no modifica `sprints`, `sprint_stories`, `stories` ni `tasks` (instantánea antes y después en integración).
- [ ] La ruta solo existe con esquema limpio `>= 11`; por debajo, `GET` sobre la colección responde `405` (versión 5 a 10) o `404` (sin rutas de Sprints) y creación, asignación y cierre siguen disponibles según sus gates.
- [ ] `go test ./...` pasa, con las pruebas de integración ejecutadas con Docker, y cada slice aterriza como un commit de unidad de trabajo con su evidencia TDD.
