# Reporte de Archivo — US-10 Descomponer historia en tareas

**Fecha de archivo**: 2026-10-03  
**Estado**: Completo (33/33 tareas)  
**Rama**: `feat/us10-descomponer-historia-tareas`  
**PR de entrega**: #80 (single-pr, size:exception)

## Resumen ejecutivo

El cambio US-10 "Descomponer una historia del Sprint en tareas" se completó exitosamente en dos corridas de `sdd-apply`:
- **Primera corrida**: 22/33 tareas (bloqueadas en Fases 4, 7 y 9 por falta de Docker)
- **Segunda corrida**: 33/33 tareas (con Docker disponible; se encontraron y corrigieron 2 defectos reales)

El cambio consolida la especificación de la nueva capacidad `tarea` en `openspec/specs/tarea/spec.md`, implementa un módulo vertical completo en `internal/task/` con 4 capas (dominio, aplicación, infraestructura PostgreSQL, transporte HTTP), añade la migración `000009_create_tasks`, y se entrega mediante un único PR con label `size:exception` (aceptado por el usuario).

## Estado de implementación

**Tareas completadas**: 33/33  
**Fase de verificación**: No se ejecutó `sdd-verify` como fase separada; la evidencia se registró en `apply-progress.md` y fue confirmada de forma independiente por el orquestador.  
**Tests**:
- Suite unitaria: 15 paquetes (`tests/unit/...`), todos en verde
- Suite de integración: 5 módulos con Testcontainers (project, projectmember, sprint, story, task), todos en verde cuando se ejecutan serializados (`go test -p 1 ./...`)
- Resultado final: 0 fallos, 0 saltos, 29 paquetes con tests, ejecución con Docker en verde

## Defectos reales encontrados y corregidos durante `apply`

Dos defectos reales fueron descubiertos durante la ejecución contra PostgreSQL real (no fueron detectados por tests con fakes):

1. **Escaneo de tipo `int4→bool` en `PostgresTaskRepository`**
   - **Síntoma**: Las tres verificaciones de pertenencia (`SELECT 1 FROM sprints/stories/sprint_stories`) escaneaban el literal `SELECT 1` (tipo `int4` en PostgreSQL) directamente a una variable `bool`. El protocolo binario de pgx rechaza esa conversión cuando la fila existe: `cannot scan int4 (OID 23) in binary format into *bool`.
   - **Causa**: Desajuste de tipo entre el valor retornado y el destino de escaneo.
   - **Corrección**: Cambiar los destinos de `Scan` de `bool` a `int` en `internal/task/infrastructure/postgres/repository.go`.
   - **Commit**: `fix(task): fix membership scan type mismatch and container readiness race` (2b3a0c4 en la rama).

2. **Carrera de disponibilidad del contenedor de test**
   - **Síntoma**: `taskDatabase(t)` en `tests/integration/task/postgres/repository_integration_test.go` no esperaba a que PostgreSQL aceptara conexiones después de que Testcontainers reportara "ready". Causaba un `EOF` intermitente en el primer `apply` de migración.
   - **Causa**: Falta de sincronización; helpers hermanos de `sprint`/`story`/`project` ya implementaban un patrón de espera con `Ping`.
   - **Corrección**: Agregar `waitForTaskDatabaseReady` (bucle de `Ping` con reintentos, mismo patrón que los módulos existentes).
   - **Commit**: Mismo commit que arriba (fix del punto 1).

Ambos defectos fueron ocultos durante la primera corrida (sin Docker disponible) porque solo se pudo verificar compilación (`go vet`/`go build`), nunca ejecución real contra PostgreSQL.

## Criterios de éxito cumplidos

Verificados contra el comportamiento implementado y la suite de tests en verde:

1. **Un cliente puede crear una o más tareas para una historia asignada al Sprint indicado, y quedan persistidas** ✓  
   Evidencia: `TestCreateTasksPersistsBatchLinkedToStorySprintAndProject` (integración), `TestCreateTasksHTTPEndToEnd/valid_batch_persists_and_returns_generated_tasks` (HTTP real).

2. **Cada tarea requiere título; la estimación de horas es opcional y respeta el máximo `99999.99` y 2 decimales** ✓  
   Evidencia: `go test ./tests/unit/task/domain/... ./tests/unit/task/application/...` PASS; `TestTasksTableEnforcesConstraints` (integración) verifica restricciones en base de datos.

3. **Una historia que no pertenece al Sprint seleccionado, o un proyecto/Sprint/historia inexistente o ajeno, produce el error correspondiente y cero tareas almacenadas** ✓  
   Evidencia: `TestCreateTasksRejectsMissingOrForeignResourcesWithoutWrites`, `TestCreateTasksRejectsStoryNotAssignedToSprint` (integración), `TestCreateTasksHTTPEndToEnd/story_not_assigned_to_sprint_responds_409_without_writes` (HTTP real).

4. **El lote es todo o nada, también ante fallo de persistencia** ✓  
   Evidencia: `TestCreateTasksRollsBackWhenAnInsertFails` (integración) simula un error forzado en el segundo insert y confirma 0 filas tras rollback.

5. **La ruta solo se registra con esquema limpio en versión `>= 9`; las rutas existentes conservan sus gates** ✓  
   Evidencia: `go test ./tests/unit/cmd/api/... ./internal/api/...` PASS.

6. **Las tareas no tienen campo de estado y quedan disponibles para US-15** ✓  
   Evidencia: `domain.Task` no incluye campo de estado; el handler rechaza una clave `status` en el cuerpo con `400 invalid_request` (`TestCreateTasksHandler/unknown_task_field`).

7. **`go test ./...` pasa; las pruebas de integración PostgreSQL/Testcontainers se ejecutan con Docker disponible** ✓  
   Evidencia: Suite completa ejecutada con `go test -p 1 ./...` → 0 fallos, 0 saltos, 29 paquetes con tests.

## Especificación consolidada

- **Archivo nuevo**: `openspec/specs/tarea/spec.md`
- **Contenido**: Especificación completa de la capacidad `tarea` con todos los requirements y scenarios
- **Formato**: Sigue la convención de los specs existentes (historia, sprint, project)
- **Delta spec original**: `openspec/changes/archive/2026-10-03-us-10-descomponer-historia-tareas/specs/tarea/spec.md` (preservado)

## Resumen de commits

11 commits en `feat/us10-descomponer-historia-tareas`:

| Commit | Mensaje | Fase |
|---|---|---|
| c394d08 | feat(migrations): add tasks table migration 000009 | 1 |
| 99dea80 | feat(task): add Task domain validation | 2 |
| 6b46ec5 | feat(task): add CreateTasksUseCase with batch validation | 3 |
| 93fcb05 | feat(task): add PostgresTaskRepository with sprint-story membership check | 4 |
| 761a8cf | feat(task): add CreateTasksHandler with strict decoding and error mapping | 5 |
| 81b1576 | feat(api): gate task creation route behind migration 000009 | 6 |
| 1b5abc1 | test(task): add end-to-end HTTP integration coverage | 7 |
| 191bb3a | docs: document task creation route and migration 000009 | 8 |
| 2b3a0c4 | docs(sdd): record final verification evidence for US-10 (Docker-blocked) | Cierre 1ª corrida |
| 47ce4c1 | fix(task): fix membership scan type mismatch and container readiness race | Defectos reales |
| 3e7361b | docs(sdd): confirm final verification for US-10 with Docker available | Cierre 2ª corrida |

## Estrategia de entrega

- **Estrategia elegida**: `single-pr`
- **Pull Request**: #80
- **Líneas estimadas**: ~1640 (Pronóstico de revisión detallado en `tasks.md`, sección "Pronóstico de revisión")
- **Presupuesto de revisión**: 400 líneas (excedido)
- **Label de excepción**: `size:exception` (aceptado por el usuario antes de iniciar `sdd-apply`)
- **Divisibilidad**: Cada commit representa una Unidad de Trabajo (fases 1–8) con su propio ciclo RED → GREEN → REFACTOR

## Estado de migraciones

- **Migración nueva**: `000009_create_tasks.up.sql` y `000009_create_tasks.down.sql`
- **Tabla**: `tasks` (id UUID, project_id, sprint_id, story_id, title TEXT, estimated_hours NUMERIC(7,2), seq BIGINT GENERATED ALWAYS AS IDENTITY, created_at TIMESTAMPTZ DEFAULT now())
- **Claves foráneas compuestas**:
  - `tasks_sprint_story_fkey (sprint_id, story_id) → sprint_stories` (ON DELETE RESTRICT)
  - `tasks_sprint_project_fkey (sprint_id, project_id) → sprints` (ON DELETE RESTRICT)
  - `tasks_story_project_fkey (story_id, project_id) → stories` (ON DELETE RESTRICT)
- **Índice**: `tasks_sprint_story_idx` sobre `(sprint_id, story_id, seq)`
- **Check**: `tasks_estimated_hours_positive (estimated_hours > 0)`
- **Verificación**: La migración se revierte con `DOWN` sin impacto en historias, Sprints ni asignaciones

## Artefactos de cambio preservados en archivo

```
openspec/changes/archive/2026-10-03-us-10-descomponer-historia-tareas/
├── proposal.md             (13.9 KB)
├── design.md               (40.2 KB)
├── exploration.md          (7.5 KB)
├── specs/
│   └── tarea/
│       └── spec.md         (Delta spec, preservada)
├── tasks.md                (33.0 KB, 33/33 completas)
├── apply-progress.md       (9.3 KB, evidencia de verificación)
└── archive-report.md       (Este archivo)
```

## Observaciones para el orquestrador

1. **Verificación independiente completada**: El orquestador verificó de forma independiente la suite de integración del módulo `task` (`go test -p 1 ./tests/integration/task/postgres/... -v` → PASS, 75.573s, 0 fallos).

2. **Contención de recursos**: La ejecución de `go test ./...` con paralelismo por defecto mostró fallas de contención de Docker Desktop en módulos preexistentes (project), confirmando que es un límite ambiental, no un defecto de US-10. Se ejecutó `go test -p 1 ./...` para serializar binarios de test.

3. **Reconfirmación de número libre**: `000009` confirmada como siguiente número libre en `internal/project/infrastructure/postgres/migrations/` (TestMigrationVersionsAreUnique PASS).

4. **PR abierto**: El usuario abrió PR #80 sobre la rama `feat/us10-descomponer-historia-tareas` antes de solicitar el archivo. El PR contiene todos los 11 commits listados arriba, con size:exception aceptado.

## Pronóstico de trabajo futuro

- **US-15** podrá usar las tareas creadas por US-10 para registrar horas trabajadas
- **Operación de desasignación**: La FK `tasks_sprint_story_fkey` impide desasignar una historia que tiene tareas (ON DELETE RESTRICT). Esto es intencional; la política se revisará cuando se especifique la desasignación.
- **Listado de tareas**: No está en alcance de US-10; puede implementarse como una operación futura usando el índice `tasks_sprint_story_idx` y el campo `seq`.

## Cierre

El cambio US-10 se da por **archivado y completado** el 2026-10-03. La rama `feat/us10-descomponer-historia-tareas` está lista para revisión y merge bajo PR #80. No hay tareas pendientes, verificaciones bloqueadas ni riesgos no documentados.
