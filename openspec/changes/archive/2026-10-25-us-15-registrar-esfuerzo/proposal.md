# Propuesta: Registrar el esfuerzo realizado en una historia o tarea (US-15)

## Intención

Implementar US-15 (issue #43) para que un integrante del equipo pueda registrar las horas que dedicó a una historia o a una tarea de esa historia, indicando quién trabajó, en qué fecha, qué hizo y cuántas horas. Hoy el sistema permite estimar historias (US-05, Planning Poker) y descomponerlas en tareas (US-10), pero no guarda ningún dato de esfuerzo real: no existe tabla, módulo ni ruta de esfuerzo. Sin ese registro no es posible comparar lo estimado con lo realizado, que es la base de las métricas de US-19 (totales y desvíos).

El cambio deja el esfuerzo persistido y siempre asociado a una historia (también cuando se registra sobre una tarea), de modo que US-19 pueda agregarlo por historia sin contar dos veces y TUI-08 (#101) pueda operar la funcionalidad desde la terminal.

## Decisiones confirmadas por el usuario

Estas decisiones están cerradas y son restricciones de este cambio; no se reabren en las fases siguientes.

| # | Tema | Decisión |
|---|------|----------|
| 1 | Sprint cerrado | Se permite registrar esfuerzo sobre historias o tareas de un Sprint cerrado (`is_closed=true`). |
| 2 | Historia sin Sprint (backlog) | Se permite registrar esfuerzo sobre una historia que no está asignada a ningún Sprint. |
| 3 | Fecha futura | Se rechaza una `work_date` posterior a la fecha UTC actual del servidor, obtenida de un reloj inyectado (`func() time.Time`). |
| 4 | Ruta | `POST /projects/{project_id}/stories/{story_id}/effort-records`; cuerpo `{member_id, task_id?, work_date, description, hours}`; responde `201` con el registro creado. |
| 5 | Errores de referencia | `404 member_not_found` y `404 task_not_found` (integrante fuera del proyecto, tarea inexistente o de otra historia); `422 validation_failed` solo para campos del cuerpo. |
| 6 | Listado | No se agrega `GET` en esta US. |
| 7 | Límites | `hours` > 0 y ≤ 24 con hasta 2 decimales (`NUMERIC(5,2)`); `description` obligatoria, recortada, hasta 500 caracteres; se permiten registros duplicados. |
| 8 | Entrega | Un solo PR con `size:exception`. |

## Alcance

### Incluido
- Módulo nuevo `internal/effort` con las cuatro capas del proyecto (`domain`, `application`, `infrastructure/postgres`, `transport/http`), siguiendo el patrón de `internal/task`.
- Entidad `EffortRecord` con `ID`, `ProjectID`, `StoryID`, `TaskID` opcional, `MemberID`, `WorkDate`, `Description`, `Hours` y marca de creación; ID generado en el servidor con `api.NewProjectID`.
- Validación del cuerpo en dominio/aplicación: `member_id` obligatorio, `task_id` opcional, `work_date` en formato `YYYY-MM-DD` con parseo de ida y vuelta y rechazo de fecha futura, `description` y `hours` según la decisión 7. Los errores de campo se devuelven juntos en `422 validation_failed` con `fields`.
- Verificación de referencias acotada por proyecto: proyecto existente, historia del proyecto, integrante del proyecto y, si se indica, tarea de esa historia. Errores `404 project_not_found` y `404 story_not_found` (convención existente) más `404 member_not_found` y `404 task_not_found` (decisión 5).
- Migración `000012_create_effort_records` con la tabla `effort_records`, sus claves foráneas compuestas y las dos restricciones `UNIQUE` nuevas sobre tablas existentes (ver Enfoque), con su `down` correspondiente.
- Habilitación condicional de la ruta con un flag nuevo `Effort` en `api.MigrationReadiness` (`version >= 12`), dependencias nuevas en `api.HTTPDependencies` y composición en `cmd/api/main.go`.
- Actualización de `tests/integration/migrations/migration_files_test.go` a 12 pares de migraciones.
- Pruebas unitarias (dominio, aplicación con reloj fijo y fakes, handler HTTP por tabla de casos, readiness) e integración PostgreSQL/Testcontainers, con TDD estricto.
- Documentación: fila nueva en la tabla de rutas del `README.md` y corrección de la nota desactualizada sobre la migración `000003` duplicada en `openspec/config.yaml`.

### Fuera de alcance
- Operación desde la TUI (TUI-08, #101).
- Listado o consulta de registros de esfuerzo (`GET`), por la decisión 6.
- Totales, comparación estimado/real y desvíos (US-19).
- Edición o eliminación de registros de esfuerzo.
- Estado de la tarea: `openspec/specs/tarea/spec.md` lo difiere a "US-15", pero el issue #43 no lo pide; sigue diferido y la tarea mantiene su contrato actual sin campo de estado.
- Restricciones por estado del Sprint o por pertenencia a un Sprint (decisiones 1 y 2), detección de duplicados (decisión 7) y límite de horas acumuladas por integrante y día.
- Refactorizar el mecanismo de doble umbral de habilitación por esquema.

## Capacidades

### Capacidades nuevas
- `esfuerzo`: registrar esfuerzo realizado sobre una historia o una tarea de esa historia, con integrante, fecha de trabajo, descripción y horas; validaciones de campos y de fecha futura con reloj inyectado; verificación de referencias acotada por proyecto/historia; regla explícita de que todo registro pertenece a su historia aunque indique una tarea (para que US-19 no cuente doble); contrato HTTP y habilitación condicionada a la migración `000012`. El nombre sigue la convención en español de las capacidades existentes (`historia`, `sprint`, `tarea`).

### Capacidades modificadas
- Ninguna. Las restricciones `UNIQUE` nuevas sobre `tasks` y `project_members` son soporte de integridad para `esfuerzo` y no cambian ningún requisito observable de `tarea` ni de `project`. El estado de la tarea no se incorpora, por lo que `tarea` no requiere delta.

## Enfoque propuesto

Corte vertical nuevo en `internal/effort/`, replicando la estructura y las convenciones de errores de `internal/task` (`400 invalid_request` para JSON inválido o campos desconocidos, `422 validation_failed` con `fields`, `404 *_not_found`, `500 internal_error`).

1. **Modelo de datos: una sola tabla con historia obligatoria.** `effort_records` tiene `story_id NOT NULL` y `task_id` anulable. Todo registro cuelga de una historia, de modo que US-19 agrega con un `GROUP BY story_id` sin unir con `tasks`. Alternativas descartadas en la exploración: `story_id`/`task_id` anulables con `CHECK` XOR (obliga a unir con `tasks` para agregar por historia) y dos tablas separadas (duplica código, migraciones y lecturas).

2. **Integridad en base de datos con claves foráneas compuestas** (defensa en profundidad de la verificación de aplicación):
   - `(story_id, project_id)` → `stories(id, project_id)` (ya existe `UNIQUE(id, project_id)`).
   - `(member_id, project_id)` → `project_members(id, project_id)`, para lo cual la migración agrega `UNIQUE(id, project_id)` en `project_members`.
   - `(task_id, story_id, project_id)` → `tasks(id, story_id, project_id)`, para lo cual la migración agrega `UNIQUE(id, story_id, project_id)` en `tasks`. Con `MATCH SIMPLE` (comportamiento por defecto), la FK no se evalúa cuando `task_id` es `NULL`.
   - `CHECK` de `hours > 0 AND hours <= 24` sobre `NUMERIC(5,2)` y de longitud de `description`. La política `ON DELETE` y el detalle de índices se fijan en diseño.

3. **Migración `000012_create_effort_records`** en `internal/project/infrastructure/postgres/migrations/`. El `down` elimina `effort_records` y luego las dos restricciones `UNIQUE` agregadas, en ese orden. Se prueba aplicar, revertir y reaplicar.

4. **Contrato HTTP**: `POST /projects/{project_id}/stories/{story_id}/effort-records` con `DisallowUnknownFields` y verificación de un solo objeto JSON. La jerarquía viaja en la ruta y los datos en el cuerpo, igual que en las rutas existentes. Respuesta `201` con el registro creado; la forma exacta queda para la especificación.

5. **Fecha y reloj**: la capa de aplicación recibe `func() time.Time`, compara `work_date` contra la fecha UTC del reloj y rechaza fechas futuras con `422`. Los tests usan un reloj fijo para ser deterministas.

6. **Verificación de referencias**: un puerto de repositorio en PostgreSQL resuelve, dentro de la transacción del alta, la existencia del proyecto, de la historia en el proyecto, del integrante en el proyecto y de la tarea en esa historia, y devuelve errores tipados que el handler mapea a los `404` correspondientes. Las violaciones de FK que lleguen igualmente se traducen al mismo error de dominio y no a `500`.

7. **Habilitación por esquema**: `Effort bool` en `api.MigrationReadiness` con `readiness.Effort = version >= 12` en `ResolveMigrationReadiness`, dependencias propias de esfuerzo en `api.HTTPDependencies` y registro condicional de la ruta; en `cmd/api/main.go` se replica el bloque de composición existente con su log de disponibilidad, sin depender de dependencias creadas por otros gates.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/project/infrastructure/postgres/migrations/000012_create_effort_records.{up,down}.sql` | Nuevo | Tabla `effort_records`, FKs compuestas, `CHECK`s y `UNIQUE` nuevas en `tasks` y `project_members`. |
| `internal/effort/domain/` | Nuevo | Entidad `EffortRecord`, validaciones de campos y `ValidationError` con `Fields`. |
| `internal/effort/application/` | Nuevo | Comando, puertos (repositorio, generador de IDs, reloj) y caso de uso de registro. |
| `internal/effort/infrastructure/postgres/` | Nuevo | Repositorio transaccional con verificación de referencias y alta. |
| `internal/effort/transport/http/` | Nuevo | Handler con decodificación estricta y mapeo de errores. |
| `internal/api/api.go` | Modificado | Flag `Effort`, dependencias de esfuerzo y registro condicional de la ruta. |
| `cmd/api/main.go` | Modificado | Composición del módulo de esfuerzo con el gate `>= 12`. |
| `tests/integration/migrations/migration_files_test.go` | Modificado | Se esperan 12 pares de migraciones. |
| `tests/unit/effort/`, `tests/unit/api/`, `tests/unit/cmd/api/` | Nuevo/Modificado | Dominio, caso de uso, handler y readiness `>= 12`. |
| `tests/integration/effort/postgres/` | Nuevo | Persistencia, FKs compuestas, `CHECK`s, `down` y reaplicación con Testcontainers. |
| `README.md` | Modificado | Fila nueva en la tabla de rutas. |
| `openspec/config.yaml` | Modificado | Corrección de la nota sobre la migración `000003` duplicada (ya no existe). |
| `openspec/specs/esfuerzo/spec.md` | Futuro | Especificación nueva, creada en la fase de spec y consolidada al archivar. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Las `UNIQUE` nuevas en `tasks` y `project_members` modifican tablas existentes y podrían fallar con datos previos o romper el `down`. | Baja | Son superconjuntos de claves primarias, por lo que no pueden violarse con datos existentes; pruebas de integración de `up`, `down` y reaplicación. |
| US-19 suma dos veces el esfuerzo de una tarea (por tarea y por historia). | Media | La spec de `esfuerzo` deja explícito que todo registro pertenece a su historia y que `task_id` solo precisa el destino. |
| La validación de fecha futura depende del reloj y de la zona horaria. | Media | Reloj inyectado y comparación en UTC; tests con reloj fijo en bordes (hoy, mañana). |
| Una violación de FK no anticipada termina como `500`. | Baja | Verificación previa en la transacción y traducción de errores de FK a errores de dominio, cubierta por integración. |
| Otro cambio toma el número `000012` antes de integrar. | Baja | Reconfirmar la cabeza del directorio de migraciones al aplicar y antes del PR. |
| El PR único supera el presupuesto de 400 líneas (~1.300–1.500). | Alta | `size:exception` aceptado por el usuario; commits por unidad de trabajo para facilitar la revisión. |
| Los tests de integración requieren Docker. | Media | Se ejecutan con Docker disponible; los resultados omitidos se informan explícitamente. |

## Plan de reversión

Revertir el PR único que agrega `internal/effort/`, la migración `000012`, los cambios en `internal/api/api.go`, `cmd/api/main.go`, el test de migraciones, `README.md` y `openspec/config.yaml`. Si la migración ya se aplicó en un entorno persistente, comprobar antes si existen registros de esfuerzo: el `down` elimina `effort_records` con sus datos y luego las dos `UNIQUE` agregadas, por lo que no debe ejecutarse sin decidir la conservación de esos registros. Con el esquema en `>= 12` y sin el código, la ruta simplemente no existe; con el esquema en `<= 11`, el gate deja la ruta deshabilitada sin afectar proyectos, historias, Sprints, integrantes, tareas ni Planning Poker.

## Dependencias

- US-10 integrada en `main` (tabla `tasks`) y US-03 (tabla `project_members`).
- Migraciones hasta `000011` presentes en `main`.
- Docker disponible para las pruebas de integración con Testcontainers.

## Plan de entrega

Un solo PR desde `feat/us-15-registrar-esfuerzo` hacia `main` con la etiqueta `size:exception` (estrategia `single-pr`, decisión 8). Justificación: el corte es vertical y no tiene valor parcial entregable; una migración sin módulo, o un módulo sin ruta, no satisface ningún criterio de aceptación, y la mayor parte del volumen son pruebas. Dentro del PR se mantienen commits por unidad de trabajo (migración, dominio, aplicación, repositorio, HTTP y cableado, documentación) con mensajes Conventional Commits.

## Decisiones abiertas para el orquestador

Ninguna. Las ocho decisiones de producto están cerradas. Quedan para diseño solo detalles técnicos sin impacto de producto: política `ON DELETE` de las FKs, índices y forma exacta de la respuesta `201`.

## Criterios de éxito

Derivados del issue #43 según la exploración (el texto literal del issue no se releyó en esta fase; la fase de spec debe contrastarlo con `gh issue view 43`).

- [ ] Un integrante del proyecto puede registrar esfuerzo sobre una historia del proyecto indicando integrante, fecha, descripción y horas, y el registro queda persistido con `201`.
- [ ] Puede registrar esfuerzo sobre una tarea indicando `task_id`; el registro queda asociado a la tarea y a su historia.
- [ ] Se valida que las horas sean > 0 y ≤ 24 con hasta 2 decimales, que la descripción sea obligatoria (recortada, ≤ 500 caracteres) y que la fecha tenga formato `YYYY-MM-DD` y no sea futura; los errores se informan con `422 validation_failed` y `fields`, sin persistir nada.
- [ ] Un integrante ajeno al proyecto produce `404 member_not_found`; una tarea inexistente o de otra historia produce `404 task_not_found`; un proyecto o historia inexistente o ajeno produce `404 project_not_found` o `404 story_not_found`; en todos los casos no se persiste nada.
- [ ] Se acepta esfuerzo sobre historias de Sprints cerrados y sobre historias en el backlog, y se aceptan registros duplicados.
- [ ] La base de datos impide registros con integrante, historia o tarea de otro proyecto, o con una tarea de otra historia, mediante FKs compuestas.
- [ ] La ruta solo se registra con esquema limpio en versión `>= 12`; las rutas existentes conservan sus gates.
- [ ] La migración `000012` se aplica, revierte y reaplica sin errores, y el test de archivos de migración espera 12 pares.
- [ ] `go test ./...` pasa; las pruebas de integración PostgreSQL/Testcontainers se ejecutan con Docker disponible.
