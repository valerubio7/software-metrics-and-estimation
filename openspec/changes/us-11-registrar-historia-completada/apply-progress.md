# Apply progress — US-11 registrar una historia del Sprint como completada

## Resumen

Rama `feat/us11-registrar-historia-completada`. Modo: `single-pr`, `strict_tdd: true`, runner `go test ./...`. Corrida 1 de 2 de `sdd-apply`: Fase 0 y WU 1-4.

## Excepción de tamaño (`size:exception`)

Pronóstico: ~585 líneas cambiadas (producción ~190, pruebas ~370, documentación ~25) contra un presupuesto de 400 (~46 % sobre el límite). El usuario exige un único Pull Request (`delivery_strategy: single-pr`), por lo que el orquestador acepta `size:exception`. El PR es revisable commit por commit (una WU = un commit). Registrada antes de la WU 1 (tarea 0.2).

## Ciclo TDD por unidad de trabajo

| WU | RED | GREEN | REFACTOR | Commit |
|---|---|---|---|---|
| 1 | `TestMigrationVersionsAreUnique` falla ("exactly ten migration pairs", "missing canonical migration version 000010"); `TestCompletionSchemaRequiresMigrationTen` y `TestSprintStoryCompletionMigrationCanBeReversedAndReapplied` fallan (no existe `000010_*.up.sql`). | Se crean `000010_add_sprint_story_completion.{up,down}.sql`; las tres pruebas pasan (migraciones 1.2 s; story/postgres 14.2 s). Triangulación 1.3: suites completas de `story/postgres` (376 s), `task/postgres` (80 s) y `migrations` en verde con `-p 1`: PK, FKs de `000007` y `tasks_sprint_story_fkey` intactas. | Formato y nombres revisados contra `000001`-`000009`; `000010` es el siguiente número libre. Sin cambios de comportamiento. | ver `git log` (`feat(migrations): add sprint story completion column (000010)`) |
| 2 | pendiente | pendiente | pendiente | pendiente |
| 3 | pendiente | pendiente | pendiente | pendiente |
| 4 | pendiente | pendiente | pendiente | pendiente |
| 5-8 | corrida 2 | corrida 2 | corrida 2 | corrida 2 |

### Evidencia de unidad de trabajo

| WU | Prueba enfocada | Harness real | Límite de reversión |
|---|---|---|---|
| 1 | `go test ./tests/integration/migrations/... ./tests/integration/story/postgres/... -run "TestMigration\|TestCompletionSchema\|TestSprintStoryCompletionMigration"` → ok | Testcontainers `postgres:16-alpine` (Docker 29.8.1) | Revertir `000010_*.sql`, el `"000010"` del test y las pruebas de esquema |

## Commits y traza TDD

Los hashes se registran en el commit de la WU siguiente (un commit no puede contener su propio hash).

- WU 1: `feat(migrations): add sprint story completion column (000010)` — RED: `TestMigrationVersionsAreUnique` falla sin `000010`; GREEN: pasa junto con `TestCompletionSchemaRequiresMigrationTen`; REFACTOR: sin cambios de comportamiento.

## Defectos encontrados

Ninguno hasta ahora.

## Límites de entorno

Docker disponible (servidor 29.8.1). Las pruebas de integración usan Testcontainers `postgres:16-alpine` reales. Si hay contención de recursos se usa `go test -p 1`.
