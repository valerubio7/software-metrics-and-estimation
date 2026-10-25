# Propuesta: Registrar un defecto detectado durante el desarrollo (US-16)

## Intención

Implementar US-16 (issue #44) para que un proyecto pueda registrar un defecto detectado durante el desarrollo: descripción, severidad, estado, historia relacionada y Sprint de detección, con el defecto persistido y asociado a su proyecto. Hoy el sistema ya estimó historias (US-05/US-14), descompuso historias en tareas (US-10) y registró esfuerzo real (US-15), pero no guarda ningún dato de defectos: no existe tabla, módulo ni ruta. Sin ese registro no es posible construir US-18 (consulta de defectos), US-19 (métricas que cuentan defectos detectados y resueltos), TUI-09 ni TUI-10.

El alta queda por defecto sin Sprint de resolución (US-17, #45, la agrega con su propia migración) y sin listado (`GET`, US-18, #46). La operación desde la TUI es TUI-09 (#102) y queda fuera de este cambio.

## Decisiones confirmadas por el usuario

Estas decisiones están cerradas y son restricciones de este cambio; no se reabren en las fases siguientes.

| # | Tema | Decisión |
|---|------|----------|
| 1 | Severidad | Catálogo cerrado `alta`, `media`, `baja` (idéntico a la prioridad de historias). |
| 2 | Estado | Catálogo cerrado `detectado`, `en_correccion`, `resuelto`; el alta solo acepta `detectado` o `en_correccion`, porque `resuelto` exige un Sprint de resolución que no existe todavía (US-17). |
| 3 | Sprint de detección | Obligatorio y del proyecto; no se exige que sea el Sprint asignado de la historia (`sprint_stories` lo permite para historias de backlog). Se acepta el Sprint cerrado como Sprint de detección. |
| 4 | Duplicados | Se permiten defectos duplicados (misma historia, descripción y severidad). |
| 5 | Entrega | Un solo PR con `size:exception`. |

## Alcance

### Incluido
- Módulo nuevo `internal/defect` con las cuatro capas del proyecto (`domain`, `application`, `infrastructure/postgres`, `transport/http`), siguiendo el patrón de `internal/effort` (integrado en `main` por el PR #109).
- Entidad `Defect` con `ID`, `ProjectID`, `StoryID`, `SprintID`, `Description`, `Severity`, `Status` y marca de creación; ID generado en el servidor con `api.NewProjectID`.
- Validación del cuerpo en dominio/aplicación: `story_id` y `sprint_id` obligatorios y UUID válidos, `description` obligatoria (recortada, hasta 500 caracteres y sin carácter NUL), `severity` del catálogo de la decisión 1 y `status` del subconjunto inicial de la decisión 2. Los errores de campo se devuelven juntos en `422 validation_failed` con `fields` (decisión 2 de US-15, misma convención).
- Verificación de referencias acotada por proyecto, con precedencia fija: proyecto existente (`404 project_not_found`), historia del proyecto (`404 story_not_found`) y Sprint del proyecto (`404 sprint_not_found`).
- Migración `000013_create_defects` con la tabla `defects`, sus claves foráneas compuestas y los `CHECK` de catálogos (ver Enfoque), con su `down` correspondiente. No agrega restricciones sobre tablas existentes (`stories` y `sprints` ya tienen `UNIQUE(id, project_id)` desde la migración `000007`).
- Habilitación condicional de la ruta con un flag nuevo `Defect` en `api.MigrationReadiness` (`version >= 13`), dependencias nuevas en `api.HTTPDependencies` y composición en `cmd/api/main.go`.
- Actualización de `tests/integration/migrations/migration_files_test.go` a 13 pares de migraciones.
- Pruebas unitarias (dominio, aplicación con fakes, handler HTTP por tabla de casos, readiness) e integración PostgreSQL/Testcontainers, con TDD estricto.
- Documentación: fila nueva en la tabla de rutas del `README.md` y nota de US-16 en `openspec/config.yaml`.

### Fuera de alcance
- Actualizar el estado del defecto y registrar su Sprint de resolución (US-17, #45): ni ruta ni columna propia; la migración `000014` agregará `resolution_sprint_id`.
- Consulta/listado de defectos (`GET`, US-18, #46).
- Cálculo de métricas de defectos (US-19, #47) y Dashboard (US-20, #48).
- Operación desde la TUI (TUI-09, #102), que además requiere US-18.
- Edición, eliminación y transiciones de defectos fuera de las dos rutas de alta/resolución.
- Cambiar el catálogo de prioridad de historias ni tocar ningún otro módulo existente.

## Capacidades

### Capacidades nuevas
- `defecto`: registrar un defecto de un proyecto, con descripción, severidad, estado inicial, historia relacionada y Sprint de detección; validaciones de campos con errores acumulados; verificación de referencias acotada por proyecto con precedencia fija; regla explícita de que el alta nunca crea un defecto en estado `resuelto`; contrato HTTP y habilitación condicionada a la migración `000013`. El nombre sigue la convención en español de las capacidades existentes (`historia`, `sprint`, `tarea`, `esfuerzo`).

### Capacidades modificadas
- Ninguna. La tabla nueva solo referencia `stories` y `sprints` con sus restricciones existentes; `sprint` menciona defectos de forma genérica en su requisito de cierre ("el cierre no altera ... esfuerzos o defectos") y ese contrato no cambia: en este alcance los defectos no son parte de la asignación. Nada queda condicionado a este cambio.

## Enfoque propuesto

Corte vertical nuevo en `internal/defect/`, replicando la estructura y las convenciones de errores de `internal/effort` (`400 invalid_request` para JSON inválido o campos desconocidos, `422 validation_failed` con `fields`, `404 *_not_found`, `500 internal_error`).

1. **Modelo de datos: una tabla plana con historia y Sprint de detección obligatorios.** `defects` tiene `story_id NOT NULL` y `sprint_id NOT NULL` (detección); la resolución llega con US-17 como migración nueva (`ALTER TABLE`) sobre `000014`. Alternativas descartadas en la exploración: tabla con `resolution_sprint_id` desde ya (implementa almacenamiento de US-17 sin su caso de uso y adelanta decisiones de transición que la propuesta no cerró) y tabla sin `CHECK` de catálogos (dejaría basura persistible que US-19 tendría que tolerar).

2. **Integridad en base de datos con claves foráneas compuestas** (defensa en profundidad de la verificación de aplicación):
   - `(story_id, project_id)` → `stories(id, project_id)` (`UNIQUE(id, project_id)` ya existe).
   - `(sprint_id, project_id)` → `sprints(id, project_id)` (`UNIQUE(id, project_id)` ya existe).
   - `CHECK` de severidad, estado y longitud de descripción sobre la tabla nueva. La política `ON DELETE` y el detalle de índices se fijan en diseño.

3. **Migración `000013_create_defects`** en `internal/project/infrastructure/postgres/migrations/`. El `down` solo elimina `defects` (no toca tablas existentes). Se prueba aplicar, revertir y reaplicar.

4. **Contrato HTTP**: `POST /projects/{project_id}/defects` con `DisallowUnknownFields` y verificación de un solo objeto JSON. Cuerpo `{story_id, sprint_id, description, severity, status}`; respuesta `201` con el registro creado. Los identificadores de historia y Sprint viajan en el cuerpo porque la ruta solo lleva el proyecto, igual que el cuerpo del integrante en US-3.

5. **Verificación de referencias**: un puerto de repositorio en PostgreSQL resuelve, dentro de la transacción del alta, la existencia del proyecto, de la historia en el proyecto y del Sprint en el proyecto, en ese orden, y devuelve errores tipados que el handler mapea a los `404` correspondientes. Las violaciones de FK que lleguen igualmente se traducen al mismo error de dominio y no a `500`.

6. **Sin reloj inyectado**: el defecto no tiene fecha de trabajo ni campo comparable en este alcance (no hay desvío que calcular todavía); `created_at` se conserva en la base con `now()` y no se expone. Esto reduce el caso de uso respecto de US-15.

7. **Habilitación por esquema**: `Defect bool` en `api.MigrationReadiness` con `readiness.Defect = version >= 13` en `ResolveMigrationReadiness`, dependencias propias de defecto en `api.HTTPDependencies` y registro condicional de la ruta; en `cmd/api/main.go` se replica el bloque de composición existente con su log de disponibilidad, sin depender de dependencias creadas por otros gates.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/project/infrastructure/postgres/migrations/000013_create_defects.{up,down}.sql` | Nuevo | Tabla `defects`, FKs compuestas, `CHECK` de catálogos y longitud de descripción, secuencia de orden e índices. |
| `internal/defect/domain/` | Nuevo | Entidad `Defect`, catálogos, validaciones de campos y `ValidationError` con `Fields`. |
| `internal/defect/application/` | Nuevo | Comando, puertos (repositorio, generador de IDs) y caso de uso de registro. |
| `internal/defect/infrastructure/postgres/` | Nuevo | Repositorio transaccional con verificación de referencias y alta. |
| `internal/defect/transport/http/` | Nuevo | Handler con decodificación estricta y mapeo de errores. |
| `internal/api/api.go` | Modificado | Flag `Defect`, dependencias de defecto y registro condicional de la ruta. |
| `cmd/api/main.go` | Modificado | Composición del módulo de defecto con el gate `>= 13`. |
| `tests/integration/migrations/migration_files_test.go` | Modificado | Se esperan 13 pares de migraciones. |
| `tests/unit/defect/`, `tests/unit/cmd/api/` | Nuevo/Modificado | Dominio, caso de uso, handler y readiness `>= 13`. |
| `tests/integration/defect/postgres/` | Nuevo | Persistencia, FKs compuestas, `CHECK`s, `down` y reaplicación con Testcontainers. |
| `README.md` | Modificado | Fila nueva en la tabla de rutas. |
| `openspec/config.yaml` | Modificado | Nota de US-16 implementada. |
| `openspec/specs/defecto/spec.md` | Futuro | Especificación nueva, creada en la fase de spec y consolidada al archivar. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| La asimetría catálogo completo/aplicado (decisión 2) confunde una prueba de integración que insista en crear `resuelto`. | Baja | La spec distingue la validez de los valores del catálogo (para el `CHECK`) de la del subconjunto inicial aplicado por el alta; el escenario `resuelto` en alta queda como `422` de negocio y la transición es territorio de US-17. |
| Otro cambio toma el número `000013` antes de integrar. | Baja | Reconfirmar la cabeza del directorio de migraciones antes del PR; `main` ya trae `000012` de US-15. |
| Los consumer futuros (US-18, US-19, TUI-09) suponen coherencia historia/Sprint de detección que este alcance no garantiza. | Media | La spec declara campo por campo la integridad exigida (por proyecto) y la que no lo está (por asignación), sin dejar reglas implícitas. |
| Una violación de FK no anticipada termina como `500`. | Baja | Verificación previa en la transacción y traducción de errores de FK a errores de dominio, cubierta por integración. |
| El PR único supera el presupuesto de 400 líneas (~2.400). | Alta | `size:exception` pedido por el usuario (decisión 5); commits por unidad de trabajo para facilitar la revisión. |
| Los tests de integración requieren Docker. | Media | Se ejecutan con Docker disponible; los resultados omitidos se informan explícitamente. |

## Plan de reversión

Revertir el PR único que agrega `internal/defect/`, la migración `000013`, los cambios en `internal/api/api.go`, `cmd/api/main.go`, el test de migraciones, `README.md` y `openspec/config.yaml`. Si la migración ya se aplicó en un entorno persistente, comprobar antes si existen registros de defectos: el `down` elimina `defects` con sus datos, y como no añade restricciones sobre tablas existentes, no hay riesgo colateral. Con el esquema en `>= 13` y sin el código, la ruta simplemente no existe; con el esquema en `<= 12`, el gate deja la ruta deshabilitada sin afectar proyectos, historias, Sprints, integrantes, tareas, Planning Poker ni esfuerzo.

## Dependencias

- US-10 integrada en `main` (tabla `tasks`; los defectos no la referencian, pero da el patrón de módulo), US-03 (`project_members`) y US-12 (`sprints.seq`, para el orden que US-17 usará).
- US-15 mergeada en `main` (`migración 000012`): el directorio de migraciones queda en 12 pares y este cambio agrega el 13.
- Docker disponible para las pruebas de integración con Testcontainers.

## Plan de entrega

Un solo PR desde `feat/us-16-registrar-defecto` hacia `main` con la etiqueta `size:exception` (estrategia `single-pr`, decisión 5). Justificación: el corte es vertical y no tiene valor parcial entregable; una migración sin módulo, o un módulo sin ruta, no satisface ningún criterio de aceptación, y la mayor parte del volumen son pruebas. Dentro del PR se mantienen commits por unidad de trabajo (migración, dominio, aplicación, repositorio, HTTP, cableado y documentación) con mensajes Conventional Commits.

## Decisiones abiertas para el orquestador

Ninguna. Las cinco decisiones de producto están cerradas. Quedan para diseño solo detalles sin impacto de producto: politique `ON DELETE` de las FKs, índices, forma exacta de la respuesta `201` y el esquema interno de los `CHECK` (los nombres de catálogo ya están cerrados).

## Criterios de éxito

Derivados del issue #44 según la exploración; la fase de spec lo contrasta con `gh issue view 44`.

- [ ] Un proyecto puede registrar un defecto indicando descripción, severidad, estado, historia y Sprint de detección, y el registro queda persistido con `201`.
- [ ] Se valida que `story_id` y `sprint_id` sean UUID obligatorios, que la descripción sea obligatoria (recortada, 1–500 caracteres, sin NUL), que `severity` esté en `alta`/`media`/`baja` y que `status` esté en `detectado`/`en_correccion`; los errores se acumulan en `422 validation_failed` con `fields`, sin persistir nada.
- [ ] Un estado `resuelto` en el alta produce `422` (el Sprint de resolución es territorio de US-17).
- [ ] Un proyecto inexistente o ajeno produce `404 project_not_found`; una historia ajena o inexistente produce `404 story_not_found`; un Sprint ajeno o inexistente produce `404 sprint_not_found`, con precedencia favorable al error del proyecto; en todos los casos no se persiste nada.
- [ ] Se acepta el registro con historias de Sprints cerrados o del backlog, y se permiten duplicados exactos.
- [ ] La base de datos impide historias o Sprints de otro proyecto y valores fuera de catálogo, mediante FKs compuestas y `CHECK`s.
- [ ] La ruta solo se registra con esquema limpio en versión `>= 13`; las rutas existentes conservan sus gates.
- [ ] La migración `000013` se aplica, revierte y reaplica sin errores, y el test de archivos de migración espera 13 pares.
- [ ] `go test ./...` pasa; las pruebas de integración PostgreSQL/Testcontainers se ejecutan con Docker disponible.
