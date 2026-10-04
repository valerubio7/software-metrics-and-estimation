# Propuesta: Registrar una historia del Sprint como completada (US-11)

## Intención

Implementar US-11 (issue #39) para que un integrante del equipo pueda seleccionar una historia asignada a un Sprint y registrarla como completada dentro de ese Sprint. El registro debe conservar la asociación con el Sprint, rechazar historias que no pertenezcan a él, no contar dos veces la misma historia y dejar el dato disponible para las métricas de US-19 (Story Points completados y porcentaje de historias completadas).

Hoy no existe ningún mecanismo que registre la finalización dentro de un Sprint. La pertenencia historia-Sprint vive en `sprint_stories` (US-09, migración `000007`), que no tiene columnas de estado ni de fecha. El único estado disponible es `stories.status`, que es global a la historia, no tiene reglas de transición y se edita libremente con `PUT /projects/{project_id}/stories/{story_id}`; usarlo no cumpliría la asociación con el Sprint ni la protección contra el doble conteo (ver `exploration.md`, enfoque B).

## Decisiones tomadas por el orquestador

Estas decisiones se registran como supuestos explícitos de la propuesta. Las marcadas como "a confirmar por el equipo" tienen impacto de producto y no se atribuyen al issue.

- **Enfoque A**: columna nulable `completed_at TIMESTAMPTZ` en `sprint_stories`, agregada por la migración `000010`. `NULL` significa "no completada en este Sprint". No se sincroniza ni se depende de `stories.status`. *(A confirmar por el equipo: `stories.status` puede mostrar un valor distinto del registro de finalización por Sprint.)*
- **Contrato HTTP**: `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/completion`, sin cuerpo, siguiendo la jerarquía de rutas de US-09 y US-10.
- **Historia ya completada en ese Sprint**: 409 conflicto; el registro original se conserva sin modificarse y nunca se cuenta dos veces. Coherente con US-09, que rechaza con 409 una historia ya asignada. *(A confirmar por el equipo: se descartó la respuesta idempotente 200.)*
- **Sprint cerrado (`sprints.is_closed = true`)**: 409 conflicto, coherente con US-09. *(A confirmar por el equipo: a diferencia de US-10, que permite crear tareas en un Sprint cerrado.)*
- **Sin deshacer**, sin registrar quién la completó, sin exigir que las tareas de US-10 estén completas y sin consultas de métricas (US-19).
- **Ruta opcional** habilitada por un flag nuevo de `MigrationReadiness` con versión de esquema `>= 10`, como `Tasks`.

### Convención de errores elegida

Existen dos convenciones en el código: US-09 (`assign_stories_handler.go`) agrupa en `404 resource_not_found` y `409 assignment_conflict`; US-10 (`internal/task/transport/http/handler.go`) usa códigos específicos (`project_not_found`, `sprint_not_found`, `story_not_found` con 404 y `story_not_in_sprint` con 409). Se propone seguir **US-10**, porque es la más reciente, opera sobre exactamente el mismo recurso anidado (historia dentro de un Sprint) y ya fijó en `openspec/specs/tarea/spec.md` que "historia ajena al Sprint" es 409 `story_not_in_sprint` y no 404. Así el cliente recibe la misma respuesta para la misma situación en ambas rutas anidadas.

| Situación | Respuesta propuesta |
|---|---|
| UUID de ruta inválido | 422 `validation_failed` con `fields` |
| Proyecto inexistente | 404 `project_not_found` |
| Sprint inexistente o de otro proyecto | 404 `sprint_not_found` |
| Historia inexistente o de otro proyecto | 404 `story_not_found` |
| Historia del proyecto no asignada al Sprint | 409 `story_not_in_sprint` |
| Sprint cerrado | 409 `sprint_closed` |
| Historia ya completada en ese Sprint | 409 `story_already_completed` |
| Error de almacenamiento | 500 `internal_error`, sin detalles internos |
| Éxito | 200 con `project_id`, `sprint_id`, `story_id` y `completed_at` |

Los códigos exactos de error y el código de éxito (200 frente a 201) se fijan en la especificación; la tabla es la propuesta de partida.

## Alcance

### Incluido
- Migración `000010` que agrega `completed_at TIMESTAMPTZ NULL` a `sprint_stories` y su `down` correspondiente.
- Caso de uso y puerto en `internal/story` para registrar la finalización de una historia dentro de un Sprint.
- Repositorio PostgreSQL transaccional: bloqueo del Sprint (`FOR UPDATE`), verificación de proyecto, Sprint, historia y fila de `sprint_stories` (`FOR UPDATE`), rechazo de Sprint cerrado y de historia ya completada, y `UPDATE ... WHERE completed_at IS NULL`.
- Handler HTTP sin cuerpo con el mapeo de errores de la tabla anterior.
- Flag nuevo de readiness (`version >= 10`), dependencia opcional en `api.HTTPDependencies`, registro condicional de la ruta y composición en `cmd/api/main.go`.
- Pruebas unitarias (caso de uso con fakes, handler por tabla de casos, `ResolveMigrationReadiness`) e integración PostgreSQL/Testcontainers (persistencia, pertenencia, Sprint cerrado, no doble conteo, concurrencia), además de actualizar `tests/integration/migrations/migration_files_test.go`. TDD estricto con `go test ./...`.
- Documentación al cierre: sección en `README.md`, notas de `openspec/config.yaml` y `docs/scrum/sprint-1.md`. Se verificó que US-10 no modificó `docs/traceability.md` ni `docs/architecture`, por lo que no se tocan.

### Fuera de alcance
- Asignar historias a un Sprint (US-09) y cerrar Sprints (US-12).
- Cálculo o consulta de métricas de Story Points completados y porcentaje (US-19).
- Deshacer una finalización, registrar el autor de la finalización o exigir tareas completas.
- Sincronizar, leer o restringir `stories.status`; la edición libre de estados sigue igual.
- Listar o consultar historias completadas de un Sprint.
- Resolver la fragilidad del doble umbral de habilitación (`api.go` y `cmd/api/main.go`) o el conflicto histórico de numeración `000003`.

## Capacidades

### Capacidades nuevas
- Ninguna.

### Capacidades modificadas
- `historia`: se agregan requisitos para registrar como completada una historia asignada a un Sprint (pertenencia, Sprint cerrado, ya completada, no doble conteo, conservación de la asociación, contrato HTTP y habilitación con esquema `>= 10`). Se ubica en `historia` porque la asignación de US-09, dueña de `sprint_stories`, ya está especificada allí ("Asignar un lote de historias sin retirarlas del Product Backlog (US-09)"). Los requisitos existentes de `historia`, `sprint`, `tarea` y `project` no cambian.

## Enfoque propuesto

1. **Persistencia**: `000010_add_sprint_story_completion.{up,down}.sql`. `up`: `ALTER TABLE sprint_stories ADD COLUMN completed_at TIMESTAMPTZ NULL`. `down`: elimina la columna. La cabeza actual es `000009_create_tasks`; reconfirmarla al aplicar.
2. **Dominio y aplicación** (`internal/story`): errores nuevos (`ErrStoryNotInSprint`, `ErrStoryAlreadyCompleted`, reutilizando `ErrSprintClosed` y los de "no encontrado" cuando ya existan), un puerto `CompleteSprintStory` y un caso de uso que valida los IDs y delega en el puerto. El instante de finalización lo fija la base de datos (`now()`) o un reloj inyectado; se decide en diseño.
3. **Repositorio**: una transacción que bloquea el Sprint con `FOR UPDATE` (mismo orden de locks que `AssignStoriesForProject`, compatible con el futuro cierre de US-12), valida proyecto y estado cerrado, bloquea la fila de `sprint_stories`, distingue "historia inexistente o ajena" de "no asignada al Sprint", y actualiza solo si `completed_at IS NULL`. Cero filas afectadas tras el lock implica `ErrStoryAlreadyCompleted`.
4. **Transporte**: handler en `internal/story/transport/http/` con UUIDs de ruta, sin decodificar cuerpo, y la tabla de errores de la sección anterior.
5. **Habilitación**: flag `Completion bool` en `api.MigrationReadiness` con `version >= 10` en `ResolveMigrationReadiness`, dependencia opcional propia en `api.HTTPDependencies` y bloque en `cmd/api/main.go` independiente de `dependencies.Stories`, igual que `Tasks`.
6. **Métricas futuras**: US-19 podrá calcular Story Points completados con `SUM(stories.story_points)` y el porcentaje con `COUNT(completed_at)` sobre `sprint_stories` del Sprint, sin cambios de esquema.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/project/infrastructure/postgres/migrations/000010_*.{up,down}.sql` | Nuevo | Columna `completed_at` en `sprint_stories`. |
| `internal/story/application/` | Modificado | Errores, puerto y caso de uso de registro de finalización. |
| `internal/story/infrastructure/postgres/repository.go` | Modificado | Método transaccional de finalización. |
| `internal/story/transport/http/` | Nuevo | Handler de finalización y mapeo de errores. |
| `internal/api/api.go` | Modificado | Flag de readiness, dependencia opcional y ruta condicional. |
| `cmd/api/main.go` | Modificado | Composición con el gate `>= 10` y log de disponibilidad. |
| `tests/unit/story/`, `tests/unit/api/` | Nuevo/Modificado | Caso de uso, handler y readiness. |
| `tests/integration/story/postgres/` | Nuevo/Modificado | Persistencia, reglas y concurrencia con Testcontainers. |
| `tests/integration/migrations/migration_files_test.go` | Modificado | Reconoce la migración `000010`. |
| `README.md`, `openspec/config.yaml`, `docs/scrum/sprint-1.md` | Modificado | Documentación de US-11. |
| `openspec/specs/historia/` | Futuro | Delta spec en la fase de spec, consolidada al archivar. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| `stories.status` diverge de `completed_at` y alguien calcula métricas con el campo equivocado. | Media | Documentar que la fuente de verdad por Sprint es `sprint_stories.completed_at`; US-19 debe leer solo esa columna. Marcado "a confirmar por el equipo". |
| Doble conteo bajo solicitudes concurrentes. | Baja | Lock de fila más `WHERE completed_at IS NULL`; prueba de integración concurrente. |
| Orden de locks incompatible con el cierre de Sprint (US-12). | Baja | Bloquear siempre el Sprint antes que `sprint_stories`, igual que US-09. |
| Umbral `>= 10` desalineado con `cmd/api/main.go` (doble umbral). | Media | Pruebas de `ResolveMigrationReadiness` para versiones 9, 10 y estado sucio; bloque independiente en `main.go`. |
| Otro cambio toma el número `000010`. | Baja | Reconfirmar la cabeza del directorio al aplicar y antes del PR. |
| El PR único supera el presupuesto de 400 líneas (pronóstico abajo). | Alta | `sdd-tasks` debe pronosticar el tamaño y dejar la decisión explícita antes de `sdd-apply`. |
| Pruebas de integración dependientes de Docker. | Media | Ejecutar con Docker disponible antes de verificar; registrar el resultado real. |

## Pronóstico de tamaño (PR único, presupuesto 400 líneas)

| Bloque | Líneas estimadas |
|---|---|
| Migración `up`/`down` | ~5 |
| Aplicación (errores, puerto, caso de uso) | ~50 |
| Repositorio PostgreSQL | ~70 |
| Handler HTTP | ~60 |
| `api.go` y `main.go` | ~35 |
| Pruebas unitarias | ~200 |
| Pruebas de integración y de migraciones | ~150 |
| Documentación | ~20 |
| **Total** | **~590** |

El código de producción ronda las 220 líneas y entra en el presupuesto; con pruebas y documentación el total lo supera en torno a un 50 %. Con `single-pr` esto requiere una excepción explícita del equipo, o reducir pruebas duplicadas entre capas sin perder cobertura de los criterios.

## Plan de reversión

Revertir el PR único (migración, cambios en `internal/story`, `internal/api/api.go`, `cmd/api/main.go`, pruebas y documentación). Si la migración `000010` ya se aplicó en un entorno persistente, la migración `down` **elimina la columna `completed_at` y con ella todos los registros de finalización**; antes de ejecutarla hay que decidir si se conservan esos datos (por ejemplo, exportándolos). Si el esquema queda en `>= 10` sin el código, la API simplemente no registra la ruta; con el esquema en `<= 9`, el gate la deshabilita sin afectar proyectos, historias, Sprints, integrantes, asignación ni tareas.

## Dependencias

- US-09 integrada (`sprint_stories`, migraciones `000007` y `000008`) y US-10 integrada (migración `000009`).
- Docker disponible para las pruebas de integración con Testcontainers.

## Decisiones abiertas para el orquestador

- Excepción al presupuesto de 400 líneas con PR único (pronóstico ~590 líneas).
- Confirmación del equipo sobre los puntos marcados "a confirmar": divergencia aceptada con `stories.status`, 409 para historia ya completada y 409 para Sprint cerrado.

## Criterios de éxito

- [ ] Un cliente puede registrar como completada una historia asignada al Sprint indicado y queda `completed_at` persistido en la fila de `sprint_stories` de ese Sprint.
- [ ] Una historia del proyecto no asignada al Sprint produce 409 `story_not_in_sprint` sin cambios en la base de datos.
- [ ] Un proyecto, Sprint o historia inexistente o de otro proyecto produce el 404 correspondiente sin cambios.
- [ ] Registrar dos veces la misma historia en el mismo Sprint, también en paralelo, deja un único registro y la segunda solicitud recibe 409.
- [ ] Un Sprint cerrado rechaza el registro con 409.
- [ ] `stories.status` no se modifica.
- [ ] La ruta solo se registra con esquema limpio en versión `>= 10`; las rutas existentes conservan sus gates.
- [ ] `go test ./...` pasa, con las pruebas de integración PostgreSQL/Testcontainers ejecutadas con Docker disponible.
