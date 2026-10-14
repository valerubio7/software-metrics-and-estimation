# Reporte de archivo — US-13 consultar Sprints anteriores

**Fecha de archivo**: 2026-10-14
**Estado**: implementación completa de las tareas registradas (31/31 casillas marcadas en `tasks.md`). Verificación parcial, ver sección Verificación.
**Implementación**: 5 commits de unidad de trabajo, PR único con `size:exception` aceptado por el usuario (~1140 líneas autoradas estimadas).
**Commits**: `fad7f86`, `30d68cb`, `b782ef0`, `fa7c57b`, `a9d10e6`.
**Migración de encabezados de la spec**: commit `4695068` (`docs(sdd): align sprint spec headings with native composition format`).

## Resumen

US-13 permite consultar los Sprints cerrados de un proyecto mediante `GET /projects/{project_id}/sprints`, con sus historias planificadas, su orden y su estado de finalización por Sprint. La ruta se registra solo con esquema limpio en versión 11 o superior. La migración `000011` agrega `sprints.seq` para fijar el orden de creación.

## Tareas

31/31 casillas marcadas en `tasks.md`: slice 1 (1.1–1.6), slice 2 (2.1–2.6), slice 3 (3.1–3.7), slice 4 (4.1–4.5) y slice 5 (5.1–5.7). El detalle por commit y la traza TDD están en `tasks.md` y en el cuerpo de cada commit.

Nota de conteo: el prompt de cierre del orquestador indicaba 35 tareas. El archivo `tasks.md` contiene 31 casillas, todas marcadas. Se registran 31 porque `tasks.md` es la fuente de mayor rango para la visibilidad de completitud.

## Specs fusionadas

- Comando nativo: `gentle-ai sdd-archive-compose --canonical openspec/specs/sprint/spec.md --delta openspec/changes/us-13-consultar-sprints-anteriores/specs/sprint/spec.md --output openspec/specs/sprint/spec.md.compose-tmp && mv ...`. Código de salida `0`.
- Resultado: 9 requisitos agregados (ADDED) y 30 escenarios, todos a `openspec/specs/sprint/spec.md`. Diff del canónico: 178 inserciones y 0 eliminaciones, es decir, ningún requisito existente se alteró.
- Requisitos agregados:
  1. Consultar los Sprints anteriores de un proyecto
  2. Representar cada Sprint cerrado y sus historias planificadas
  3. Ordenar el historial de Sprints
  4. Informar listas vacías sin ambigüedad
  5. Validar el proyecto antes de listar
  6. Consultar sin modificar información histórica
  7. Manejar métodos no admitidos y fallos de almacenamiento en la consulta
  8. Habilitar la consulta solo con el esquema requerido
  9. Agregar la secuencia de creación de Sprints

Corrección: el reporte bloqueado previo (Engram, `sdd/us-13-consultar-sprints-anteriores/archive-report`, observación 42) decía 7 requisitos ADDED. El delta contiene 9. Este reporte usa el conteo verificado.

### Migración previa de encabezados (commit 4695068)

Antes de componer, `openspec/specs/sprint/spec.md` usaba encabezados en español (`### Requisito:` y `#### Escenario:`, producto del merge manual de US-12 en `9a422f8`). El compositor nativo exige `### Requirement:` y `#### Scenario:`, por lo que rechazó el canónico. El usuario autorizó la opción A: migrar solo los encabezados de los 11 requisitos y 20 escenarios existentes al formato de las otras cuatro specs. Los cuerpos no cambiaron (31 inserciones y 31 eliminaciones, todas de encabezados). Esta migración es un cambio de formato sobre contenido existente, no un requisito nuevo.

Efecto en la spec: tras el commit `4695068` y la composición de US-13, `openspec/specs/sprint/spec.md` contiene 20 requisitos (`### Requirement:`) y 50 escenarios (`#### Scenario:`).

## Artefactos SDD

| Artefacto | Estado |
|---|---|
| `proposal.md` | Presente |
| `design.md` | Presente |
| `specs/sprint/spec.md` (delta) | Presente, 9 ADDED |
| `tasks.md` | Presente, 31/31 marcadas |
| `apply-progress.md` | Presente (snapshot intermedio) |
| `exploration.md` | Presente. Su afirmación sobre el gate 405 quedó superada, ver Caveats. |
| `verify-report.md` | **Faltante.** No se ejecutó `sdd-verify` (opcional). |
| `archive-report.md` | Este documento. |

## Verificación

Reportada por el orquestador al cierre. No se re-ejecutó en esta fase, y no se ejecutó Docker.

- `go vet ./...`: limpio.
- `go test ./tests/unit/... ./tests/integration/migrations/...`: ok.
- `go test ./tests/integration/sprint/postgres/...`: ok contra PostgreSQL 16 real vía `SPRINT_TEST_DATABASE_URL`.
- Testcontainers falla en esta máquina Windows con `unexpected EOF`. Es un problema previo al cambio, no una aserción.
- Los paquetes de integración `project`, `projectmember`, `story` y `task` no se ejecutaron. El cambio no los tocó.

## Caveats

- El gate de disponibilidad de la consulta se corrigió durante la especificación: 405 para esquema limpio 5–10 (la creación sigue registrada, la consulta no), y 404 para versión < 5, estado `dirty` o error de lectura. `exploration.md` conserva su nota de que su afirmación de 405 fue superada.
- Los Sprints previos a la migración `000011` reciben un `seq` en orden físico arbitrario.
- `story_points` siempre es `null` por ahora.
- No hay paginación en la respuesta.
- No hay `closed_at` ni fechas del Sprint.

## Verificación de archivo (diff -r)

Comando: `diff -r "<snapshot pre-move>" "openspec/changes/archive/2026-10-14-us-13-consultar-sprints-anteriores"`. Salida: vacía, código `0`. El snapshot se tomó antes del `git mv`. `archive-report.md` se agregó después, y no está en el snapshot.

## Hallazgos

- Sin hallazgos de código reportados. No hay verificación independiente de los paquetes de integración sin Docker ni de los paquetes `project`, `projectmember`, `story` y `task`, y las pruebas de Testcontainers quedaron fuera por el problema de entorno indicado.
- Los `tasks.md` no fueron editados durante el archivo.
