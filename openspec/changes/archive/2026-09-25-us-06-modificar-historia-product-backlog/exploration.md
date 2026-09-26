# Exploración: US-06 — Modificar una historia del Product Backlog (issue #34)

## Estado actual

US-05 está archivada. `openspec/specs/historia/spec.md` solo contiene requisitos de creación.

**Dominio — `internal/story/domain/story.go`**
- `Story{ID, ProjectID, Title, Description, Priority, Status, StoryPoints *int, AcceptanceCriteria []string}` y `ValidationError{Fields map[string]string}`.
- `NewStory` valida en línea: título y descripción no vacíos (`TrimSpace`), prioridad exactamente `alta|media|baja`, criterios no vacíos y sin entradas en blanco.
- El estado está fijo en `"pendiente"`. No existe otro valor de estado en el código, las specs ni el SQL. No hay campo de estimación.

**Aplicación — `internal/story/application/create_story.go`**
- `CreateStoryCommand`, `StoryRepository{Create}`, `IDGenerator`, `ErrProjectNotFound`. `Execute` valida, genera el ID y hace una sola escritura.

**Persistencia — `internal/story/infrastructure/postgres/repository.go`**
- Un único INSERT parametrizado. Solo el SQLSTATE 23503 sobre `stories_project_id_fkey` se traduce a `ErrProjectNotFound`.

**HTTP — `internal/story/transport/http/handler.go`**
- `CreateStoryHandler` acepta solo POST, decodificación estricta de un único objeto (`DisallowUnknownFields`) más una guarda contra `null` dentro de los criterios.
- Errores: 400 `invalid_request`, 422 `validation_failed` (`fields`), 404 `project_not_found`, 500 `internal_error`.
- `decodeCreateStoryRequest`, `storyResponse`, `errorResponse` y `writeJSON` son privados del archivo. `storyResponse` siempre emite `"story_points": null`.

**Composición**
- `internal/api/api.go`: `StoryDependencies{Repository, GenerateID}` registra `POST /projects/{project_id}/stories` (patrones de `ServeMux` de Go 1.22+; el módulo es `go 1.27.0`).
- `cmd/api/main.go:38`: las rutas de historias se registran solo cuando `schema_migrations` muestra `version >= 2 && !dirty`.

**Esquema — migración `000002`**
- `project_id` es FK NOT NULL con `ON DELETE RESTRICT`; hay CHECK de prioridad; `status TEXT NOT NULL` sin CHECK; `story_points INTEGER NULL`; `acceptance_criteria TEXT[]` con checks de cardinalidad y de elementos no nulos.

**Tests**
- `tests/unit/story/{domain,application,transport/http}` y `tests/unit/cmd/api/main_test.go` (`fakeStoryRepository` implementa solo `Create`; además verifica que GET/PUT/DELETE sobre la ruta de colección devuelvan 405).
- `tests/integration/story/postgres/{repository,http}_integration_test.go` usan Testcontainers; la lista de migraciones está en `repository_integration_test.go:371` y aplica solo `000001` y `000002`.

## Qué falta para poder modificar

- **Dominio:** no hay comportamiento de actualización, ni conjunto de estados, ni campo de estimación; la validación está embebida en `NewStory` y hay que reutilizarla, no copiarla.
- **Aplicación:** no hay caso de uso, ni puerto, ni `ErrStoryNotFound`.
- **Persistencia:** no hay `Update`, ni columna `estimated_hours`, ni CHECK de estado.
- **HTTP:** no hay ruta de ítem ni parámetro `story_id`; los helpers son privados del handler de creación.
- **Composición:** `StoryDependencies.Repository` está tipado con la interfaz que solo crea; el gate de migración es `>= 2`.

## Observaciones de diseño

1. **Inmutabilidad de `project_id`:** nunca incluirlo en el `SET`; rechazar `id`, `project_id` y `story_points` en el cuerpo (`DisallowUnknownFields`). Usar `UPDATE ... WHERE id=$story AND project_id=$project`; cero filas significa "no encontrada", lo que además bloquea ediciones cruzadas entre proyectos con una URL inconsistente.
2. **`story_points` intacto:** omitirlo del `SET` (lo gobierna US-14). `RETURNING project_id, story_points` devuelve la historia completa en una sola sentencia, sin SELECT extra ni ventana de carrera.
3. **No encontrada:** agregar `application.ErrStoryNotFound`; el repositorio traduce cero filas a ese error; el handler devuelve `404 story_not_found`. Los UUID inválidos dan `422` con `fields`. La validación corre antes de tocar la base.
4. **Puerto separado:** agregar `StoryUpdater{Update(...)}` en lugar de extender `StoryRepository`; el repositorio Postgres implementa ambos. Así la creación no depende de la migración `000003` y los fakes existentes siguen compilando.
5. **Gate:** registrar la actualización solo con `version >= 3 && !dirty`; la creación se queda en `>= 2`. En versión 2 la creación funciona y la actualización devuelve 404.
6. **Reutilización de validación:** extraer los chequeos de contenido de `NewStory` a un helper no exportado (paso REFACTOR protegido por los tests de creación). `NewStoryUpdate` agrega estado y estimación encima.
7. **Forma de la respuesta:** agregar `estimated_hours` al `storyResponse` compartido hace que la respuesta de creación de US-05 emita `null` (cambio aditivo). Hay que actualizar README y la spec de `historia`.

## Preguntas de producto abiertas

| # | Pregunta | Default recomendado |
|---|---|---|
| Q1 | ¿PUT (completo) o PATCH (parcial)? | PUT de los seis campos editables |
| Q2 | ¿Valores permitidos de `status`? | Conjunto cerrado `pendiente`, `en_progreso`, `completada`, si no `422 fields.status` (inventado; requiere confirmación) |
| Q3 | ¿Reglas de transición? | Ninguna; cualquier valor del conjunto puede reemplazar a otro |
| Q4 | ¿Reglas de la estimación? | Nullable; si está presente, número `> 0`, hasta 2 decimales, `<= 99999.99`; `null` la borra; `0` nunca es válido |
| Q5 | ¿Nombre y almacenamiento de la estimación? | Columna `estimated_hours NUMERIC(7,2) NULL CHECK (> 0)`, JSON `estimated_hours`, Go `*float64` (alternativa `DOUBLE PRECISION` si el round-trip de pgx falla) |
| Q6 | ¿Guarda al editar cuando hay `story_points`? | No; horas y puntos son independientes |
| Q7 | ¿Concurrencia? | Fuera de alcance; gana la última escritura (documentado) |
| Q8 | ¿Claves faltantes en un PUT completo? | Las seis claves son obligatorias; `estimated_hours` puede ser `null`; una clave faltante da `422 fields.<nombre>: is required` |
| Q9 | ¿CHECK de estado a nivel de base? | Sí, en `000003`, como en prioridad; ampliar el conjunto después requiere una migración |

## Áreas afectadas

- `internal/story/domain/story.go` — `EstimatedHours *float64`, conjunto de estados, helper de validación compartido, `NewStoryUpdate`.
- `internal/story/application/update_story.go` (nuevo) — `UpdateStoryCommand`, puerto `StoryUpdater`, `ErrStoryNotFound`, `UpdateStoryUseCase`.
- `internal/story/infrastructure/postgres/repository.go` — `Update`, `pgx.ErrNoRows` → `ErrStoryNotFound`.
- `.../migrations/000003_add_story_estimated_hours.{up,down}.sql` (nuevo) — columna más checks; la migración down borra la columna y pierde los datos.
- `internal/story/transport/http/handler.go` (o un `update_handler.go` hermano) — `UpdateStoryHandler`, helpers compartidos, `estimated_hours` en `storyResponse`.
- `internal/api/api.go`, `cmd/api/main.go` — registrar `PUT /projects/{project_id}/stories/{story_id}`, gate en `version >= 3`.
- Tests: nuevos tests unitarios; `main_test.go` y ambos archivos de integración ampliados (agregar `000003` a la lista de `repository_integration_test.go:371`); `README.md`; `openspec/specs/historia/spec.md` al archivar.
- Sin cambios: `POST /projects`, dominio de proyecto, semántica de `story_points`, aserciones 405 existentes sobre la ruta de colección.

## Enfoques

| # | Enfoque | Ventajas | Desventajas | Esfuerzo |
|---|---|---|---|---|
| A | **PUT con reemplazo completo**, un solo `UPDATE ... RETURNING`, sin lectura previa | Simple, sin estado, reutiliza la validación de creación, sin JSON de tres estados, un solo viaje a la base, 404 sin carreras, `project_id`/`story_points` intactos por construcción | El cliente reenvía los seis campos; "cambiar solo el título" no se puede expresar; las reglas de transición futuras requerirían una lectura | Medio |
| B | **PATCH parcial** (ausente vs `null` vs valor para `estimated_hours`) | Coincide literalmente con "modificar solo el título" | Decodificación de tres estados y SQL de merge o lectura-modificación-escritura; validación sobre el estado fusionado; ~1,5x de tests; semántica del patch vacío | Alto |
| C | **Lectura-modificación-escritura** vía `Apply` de dominio, `FindByID` + `Save` en transacción (`FOR UPDATE`) | El dominio dueño de los invariantes y de futuras transiciones; permite concurrencia optimista después | Dos viajes y una transacción para una necesidad no especificada; `FindByID` se solapa con US-07; más tests de carrera | Alto |

## Recomendación

Enfoque A: `PUT /projects/{project_id}/stories/{story_id}` con `title`, `description`, `priority`, `status`, `acceptance_criteria` y `estimated_hours` nullable. Validar en el dominio antes de cualquier escritura, persistir con un único `UPDATE ... WHERE id AND project_id ... RETURNING` y devolver 200 con la historia completa.

Mapa HTTP: 200 éxito; 400 `invalid_request` (JSON malformado, tipos incorrectos, campos desconocidos incluyendo `id`/`project_id`/`story_points`); 404 `story_not_found` (inexistente o de otro proyecto); 422 `validation_failed` con `fields` (incluye UUID inválidos); 500 `internal_error` (genérico); 405 para otros métodos vía `ServeMux`.

## Rebanado TDD estricto (un commit de unidad de trabajo por slice, `go test ./...` tras cada paso)

Pronóstico ≈ 900–1300 líneas cambiadas (código ~300, tests ~600, SQL ~20, docs ~60), por encima del presupuesto de 400 líneas → se recomienda entrega encadenada (US-05 usó `stacked-to-main`).

1. **Dominio (~250).** RED: casos de `NewStoryUpdate` (vacíos, conjuntos de prioridad/estado, criterios, límites de la estimación, `nil` permitido, texto y orden preservados). GREEN, TRIANGULATE (espacios Unicode, bordes 0,01 / 99999,99). REFACTOR: extraer la validación compartida de `NewStory`; agregar `EstimatedHours`.
2. **Aplicación (~200).** RED con `StoryUpdater` falso: entrada inválida → cero llamadas al repositorio; válida → exactamente una llamada con los IDs correctos; `ErrStoryNotFound` y errores inesperados se propagan; no se genera ID. GREEN, TRIANGULATE, REFACTOR.
3. **Almacenamiento (~300–350, requiere Docker; los tests saltados no cuentan).** Migración `000003` + `Update`. Primero los tests de integración: actualización completa persistida (orden de criterios preservado); round-trip de horas y borrado a `NULL`; `project_id` sin cambios; `story_points` sin cambios (sembrado por SQL); proyecto equivocado → no encontrada con la fila intacta; historia inexistente → no encontrada; violaciones de CHECK salen como error; errores de base no relacionados no se etiquetan mal. Agregar `000003` a la lista de migraciones del helper.
4. **Handler HTTP (~300).** RED con caso de uso falso: 200 con cuerpo completo incl. `story_points` y `estimated_hours:null`; 400 por malformado / múltiples valores / tipos incorrectos / campos desconocidos / null en criterios; 422 por UUID inválidos y errores de dominio sin escritura; 404; 500 genérico. GREEN. REFACTOR: compartir helpers de decodificación/JSON con el handler de creación; agregar `estimated_hours` a `storyResponse`.
5. **Composición y docs (~250).** RED: `main_test.go` (ruta registrada solo con un updater; creación y proyectos preservados); integración HTTP sobre base migrada (crear → actualizar 200, `project_id` intacto, historia inexistente 404); escenarios de arranque para versión 3 y versión 2. GREEN: `api.go`, gate `>= 3` en `main.go`, README, delta de spec. No exponer la ruta antes de aplicar `000003`.

Convenciones de tests (go-testing): `t.Run` orientados a tablas por escenario, fakes escritos a mano en el borde del puerto, Testcontainers para integración.

## Riesgos

- **La semántica de estados es inventada** (hoy solo existe `pendiente`); un conjunto equivocado implica una migración de datos posterior.
- **Orden de despliegue:** la ruta de actualización necesita `000003`; revertirla borra las estimaciones.
- **Round-trip de pgx** (`NUMERIC` → `*float64`) sin verificar; comprobarlo en el test RED de almacenamiento.
- **Cambio en la respuesta compartida** altera la respuesta de creación de US-05 (`null` aditivo).
- **Radio de impacto del refactor:** extraer la validación de `NewStory` toca código de US-05 protegido solo por los tests existentes.
- **Rotura de fakes** si se agrega `Update` a `StoryRepository`; mitigado con el puerto separado.
- **Actualizaciones perdidas:** sin versionado, gana la última escritura.
- **Tamaño:** el pronóstico excede las 400 líneas; hay que elegir una estrategia de cadena antes de aplicar.

## Listo para propuesta

Sí, pendiente de las decisiones abiertas: Q1 (PUT vs PATCH), Q2/Q3 (conjunto de estados y transiciones), Q4/Q5 (reglas y almacenamiento de la estimación) y la estrategia de cadena.
