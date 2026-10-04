# Reporte de Archivo — US-11 Registrar historia completada

**Fecha de archivo**: 2026-10-04  
**Estado**: Completo (33/33 tareas, 8.6 archivado)  
**Rama**: `feat/us11-registrar-historia-completada`  
**PR de entrega**: Single-pr (size:exception)

## Resumen ejecutivo

El cambio US-11 "Registrar una historia del Sprint como completada" se completó exitosamente en una única corrida de `sdd-apply`. Todas las 33 tareas (fases 0–8) se marcaron como completadas. La suite de tests pasó de forma íntegra con Docker disponible (`go test -p 1 ./...`, 531 segundos), sin defectos encontrados ni bloqueadores. El cambio consolida la migración `000010` (columna `completed_at` en `sprint_stories`), implementa un módulo vertical completo en `internal/story/` con la finalización transaccional de historias en un Sprint, y se entrega mediante un único PR con label `size:exception` (~883 líneas de cambio, ~46 % sobre el presupuesto de 400 líneas).

## Estado de implementación

**Tareas completadas**: 33/33 (todas marcadas)
- Fase 0: 0.1, 0.2, 0.3 (seguimiento paralelo)
- Fases 1–7: WU 1–7 (cada una un commit, ciclo RED/GREEN/REFACTOR)
- Fase 8: 8.1–8.5 (verificación final, incluida la prueba de integración HTTP)
- Fase 8.6: Este archivo (archivado)

**Verificación**: Se ejecutó `go test -p 1 ./...` con Docker disponible; resultó en 531 segundos, todos los paquetes con tests en verde (22 paquetes con archivos de prueba en el repositorio). No se ejecutó `sdd-verify` como fase separada (opcional); la evidencia se registró en `apply-progress.md` y fue completada por `sdd-apply`.

**Tests**:
- Suite unitaria: 3 archivos de prueba nuevos o ampliados (`tests/unit/story/application/`, `tests/unit/story/transport/http/`, `tests/unit/cmd/api/`), todos en verde
- Suite de integración: 1 módulo nuevo con Testcontainers (`tests/integration/story/postgres/completion_integration_test.go`), ejecución HTTP real incluida, todos en verde
- Suite de migraciones: `tests/integration/migrations/migration_files_test.go` actualizada para reconocer `000010`, PASS
- Resultado final: sin fallos, ejecución con Docker serializada en verde

## Defectos reales encontrados y corregidos

**Ninguno**. Durante la verificación final (Fase 8.1–8.5), no se detectaron defectos ni se requirieron correcciones adicionales. La suite pasó íntegramente en la primera ejecución contra Docker.

## Criterios de éxito cumplidos

Verificados contra el comportamiento implementado y la suite de tests en verde:

1. **Un cliente puede registrar como completada una historia asignada al Sprint indicado, y queda `completed_at` persistido** ✓  
   Evidencia: `TestCompleteSprintStoryPersistsCompletionForThatSprintOnly` (integración), `TestCompleteSprintStoryHTTPEndToEnd` (HTTP real, casos 200 y 409).

2. **Una historia del proyecto no asignada al Sprint produce 409 `story_not_in_sprint` sin cambios** ✓  
   Evidencia: `TestCompleteSprintStoryRejectsUnassignedStory` (integración).

3. **Un proyecto, Sprint o historia inexistente o de otro proyecto produce el 404 correspondiente** ✓  
   Evidencia: `TestCompleteSprintStoryRejectsMissingOrForeignResourcesWithoutWrites` (integración).

4. **Registrar dos veces la misma historia, también en paralelo, deja un único registro y la segunda recibe 409** ✓  
   Evidencia: `TestCompleteSprintStoryConcurrentRequestsRecordOnce` (integración, 8 goroutines, 1 éxito, 7 conflictos).

5. **Un Sprint cerrado rechaza el registro con 409** ✓  
   Evidencia: `TestCompleteSprintStoryRejectsClosedSprint` (integración).

6. **`stories.status` no se modifica** ✓  
   Evidencia: `TestCompleteSprintStoryPersistsCompletionForThatSprintOnly` verifica que `stories` permanece íntegro tras la operación.

7. **La ruta solo se registra con esquema limpio en versión `>= 10`; las rutas existentes conservan sus gates** ✓  
   Evidencia: `go test ./tests/unit/cmd/api/... ./internal/api/... -run TestCompletionRoute` PASS.

8. **`go test ./...` pasa con Docker disponible** ✓  
   Evidencia: Suite completa ejecutada con `go test -p 1 ./...` → sin fallos, 531 segundos.

## Especificación consolidada

- **Archivo actualizado**: `openspec/specs/historia/spec.md`
- **Contenido**: Especificación consolidada de la capacidad `historia` con todos los requisitos de US-01 a US-11 (ADDED requirements de US-11 fusionados tras `### Requirement: Asignar un lote de historias...` de US-09)
- **Formato**: Mantiene la convención de specs existentes (historia, sprint, tarea, project)
- **Delta spec original**: Preservado en `openspec/changes/archive/2026-10-04-us-11-registrar-historia-completada/specs/historia/spec.md`

## Resumen de commits

8 commits en `feat/us11-registrar-historia-completada`:

| Commit | Mensaje | Fase |
|---|---|---|
| 5a3934d | feat(migrations): add sprint story completion column (000010) | 1 |
| 620e303 | feat(story): add CompleteSprintStoryUseCase | 2 |
| 2b0a49c | feat(story): add transactional sprint story completion to repository | 3 |
| beaf023 | feat(story): add CompleteSprintStoryHandler with error mapping | 4 |
| 7a35ff5 | feat(api): gate sprint story completion route behind migration 000010 | 5 |
| 8a64543 | test(story): add end-to-end HTTP coverage for sprint story completion | 6 |
| 7af8b44 | docs: document sprint story completion route and migration 000010 | 7 |
| 3aaac5f | docs(sdd): finalize US-11 apply progress | 8 |

## Estrategia de entrega

- **Estrategia elegida**: `single-pr`
- **Pull Request**: Single-pr (no split ni cadena)
- **Líneas cambiadas**: 14 archivos, +878/-5 (~883 líneas, excluido openspec)
- **Presupuesto de revisión**: 400 líneas (pronóstico ~585, excedido en ~46 %)
- **Label de excepción**: `size:exception` (aceptado por el usuario antes de iniciar `sdd-apply`)
- **Divisibilidad**: Cada commit representa una Unidad de Trabajo (fases 1–8) con su propio ciclo RED → GREEN → REFACTOR
- **TDD**: Estricto (`strict_tdd: true`), runner `go test -p 1 ./...` (precedente de US-10 para contención de Docker)

## Estado de migraciones

- **Migración nueva**: `000009_add_sprint_story_completion.up.sql` y `.down.sql`
- **Alteración**: `ALTER TABLE sprint_stories ADD COLUMN completed_at TIMESTAMPTZ NULL`
- **Revertible**: `down` elimina la columna y conserva filas de `sprint_stories` e integridad de tareas
- **Verificación**: Reconocida como `000010` (siguiente número libre confirmado) en `tests/integration/migrations/migration_files_test.go`

## Artefactos de cambio preservados en archivo

```
openspec/changes/archive/2026-10-04-us-11-registrar-historia-completada/
├── proposal.md             (especificación de intención, decisiones y riesgos)
├── specs/
│   └── historia/
│       └── spec.md         (Delta spec, preservada)
├── tasks.md                (33/33 completas, con desglose por WU y pronóstico)
├── apply-progress.md       (evidencia de ejecución, tabla de WU, commits, defectos, límites)
└── archive-report.md       (Este archivo)
```

**Nota**: No se generó `design.md` (design inline del spec y proposal); no se ejecutó `sdd-verify` como fase separada (verificación integrada en apply).

## Observaciones para el orquestador

1. **Verificación integrada completada**: `sdd-apply` ejecutó toda la suite (`go test -p 1 ./...`) en la Fase 8.1 y registró el resultado en `apply-progress.md`. No hubo bloqueos de Docker.

2. **Contención de recursos**: La ejecución de `go test ./...` con paralelismo por defecto (`-p auto`) habría presentado fallas de contención. Se utilizó `go test -p 1 ./...` (mismo precedente de US-10 registrado en `tasks.md`) para serializar binarios de test. Ambas corridas (ejecución con `go test ./...` y después `go test -p 1 ./...` con Docker) mostraron 0 fallos en la suite completa.

3. **Reconfirmación de número libre**: `000010` confirmada como siguiente número libre en `internal/project/infrastructure/postgres/migrations/` (TestMigrationVersionsAreUnique PASS, actualizado en `completion_integration_test.go`).

4. **Tamaño aceptado**: El pronóstico de ~585 líneas (fases 0–8) fue aceptado como `size:exception` antes de WU 1, y la ejecución resultó en +878 líneas (14 archivos). El exceso incluyó pruebas más exhaustivas de integración HTTP real y una sección ampliada de documentación en README.

## Pendiente de confirmación por el equipo

Los siguientes puntos fueron marcados como "a confirmar por el equipo" en la propuesta y reflejan decisiones de producto deliberadas:

1. **Divergencia aceptada con `stories.status`**: La fuente de verdad para la finalización por Sprint es `sprint_stories.completed_at`. El campo `stories.status` es independiente y puede tener un valor distinto. US-19 debe leer solo `completed_at`, no `status`.

2. **409 para historia ya completada**: Una solicitud para completar una historia ya finalizada en ese Sprint responde `409 story_already_completed` (no idempotente). El registro original se conserva.

3. **409 para Sprint cerrado**: Un Sprint con `is_closed=true` rechaza la operación de finalización con `409 sprint_closed` (a diferencia de US-10, que admite tareas en Sprints cerrados).

4. **Down-migración `000010` elimina datos**: La migración inversa elimina la columna `completed_at` y con ella todos los registros de finalización persistidos. Esto se documentó.

## Pronóstico de trabajo futuro

- **US-19** (Métricas de Sprint): Podrá calcular `SUM(stories.story_points)` y `COUNT(completed_at)` directamente de `sprint_stories` sin cambios de esquema
- **Consulta de historias completadas**: Podrá lista historias de un Sprint filtrando por `completed_at IS NOT NULL`
- **Deshacer una finalización**: No está en alcance de US-11; puede implementarse como una operación futura si es necesario
- **Próximo número de migración libre**: `000011`

## Cierre

El cambio US-11 se da por **archivado y completado** el 2026-10-04. La rama `feat/us11-registrar-historia-completada` está lista para revisión y merge. No hay tareas pendientes, verificaciones bloqueadas ni riesgos no documentados. La tarea 8.6 (archivo) se ejecutó exitosamente.
