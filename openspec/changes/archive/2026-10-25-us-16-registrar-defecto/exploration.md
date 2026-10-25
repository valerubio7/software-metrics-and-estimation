# Exploración: US-16 — Registrar un defecto detectado durante el desarrollo

Issue: #44. La actualización de estado y resolución es US-17 (#45), la consulta es US-18 (#46) y la operación desde la TUI es TUI-09 (#102); todas quedan fuera de este cambio.

## Estado actual (verificado)

- No existe ninguna tabla, módulo ni ruta de defectos: no hay `internal/defect` ni migración que cree una tabla de defectos. La única presencia de "defectos" es la de los placeholders de la TUI (`internal/tui/sections.go`) y menciones genéricas en otros specs.
- Las migraciones llegan hasta `000012` (`000012_create_effort_records`, US-15 mergeada en `main` como `df1ff7b`). La siguiente es `000013`.
- `tests/integration/migrations/migration_files_test.go` exige exactamente 12 pares de migraciones; hay que actualizarlo a 13.
- `stories` y `sprints` ya tienen `UNIQUE(id, project_id)` (migración `000007`): las FKs compuestas de defectos contra esas dos tablas no exigen restricciones nuevas sobre tablas existentes, y el `down` de la migración queda trivial.
- `sprint_stories` no exige que una historia esté asignada: una historia puede estar solo en el backlog. Un defecto detectado en una historia de backlog es un caso válido de primer orden.
- No hay catálogo publicado de severidad ni de estado para defectos: se define en este cambio (decisiones 1 y 2).
- Los módulos siguen cuatro capas (`domain`, `application`, `infrastructure/postgres`, `transport/http`). `internal/effort` (US-15, recién integrada) es el patrón más fresco: `400 invalid_request`, `422 validation_failed` con `fields`, `404 *_not_found` con precedencia fija y `500 internal_error`; decodificación JSON estricta de un solo objeto; descripción recortada, obligatoria, hasta 500 caracteres y sin carácter NUL porque PostgreSQL rechaza `0x00` (`SQLSTATE 22021`).
- `internal/api/api.go` registra rutas opcionales mediante punteros de dependencias y `MigrationReadiness`; defectos exigiría `version >= 13`. `cmd/api/main.go` replica el bloque de composición por módulo con su log de disponibilidad.
- Los IDs se generan en el servidor con `api.NewProjectID` (UUID v4) y viajan como identificadores independientes; la jerarquía va en la ruta y los datos de la entidad en el cuerpo.
- La TUI ya consulta los listados que necesita (proyectos, backlog, tareas, integrantes) y los selectores navegables de TUI-09 reutilizarán esa base; ninguna lectura adicional es requisito de esta US.
- `tests/integration/testpostgres` provee los helpers de Testcontainers que reutilizan todos los paquetes de integración nuevos.

## Enfoque recomendado

Módulo nuevo `internal/defect` con la tabla `defects`:

- `story_id` obligatorio y `sprint_id` obligatorio (Sprint de detección del proyecto). `resolution_sprint_id` y la transición a "resuelto" se difieren por completo a US-17, que agregará su propia migración `ALTER TABLE` (patrón de `000003`/`000011`).
- La regla de estados del alta: el catálogo completo (`detectado`, `en_correccion`, `resuelto`) vive en la base para no reescribir el `CHECK` en `000014`, pero el alta solo acepta estados iniciales (`detectado`, `en_correccion`); `resuelto` solo es alcanzable por US-17.
- FKs compuestas `(story_id, project_id)` y `(sprint_id, project_id)` como defensa en profundidad de la verificación transaccional; sin `UNIQUE` nuevas.
- Secuencia ordinal (`seq BIGINT GENERATED ALWAYS AS IDENTITY`) e índice por proyecto para la lista de US-18, que hoy no tiene ruta pero sí esquema disponible.

## Decisiones del usuario

| # | Tema | Decisión |
|---|------|----------|
| 1 | Severidad | Catálogo cerrado `alta`, `media`, `baja`, igual que la prioridad de historias. |
| 2 | Estado | Catálogo cerrado `detectado`, `en_correccion`, `resuelto`; el alta solo acepta `detectado` o `en_correccion`. |
| 3 | Sprint de detección | Obligatorio y del proyecto; no se exige que sea el Sprint asignado de la historia (una historia de backlog defiende su detección en cualquier Sprint del proyecto). |
| 4 | Duplicados | Se permiten defectos duplicados (misma historia, descripción y severidad). |
| 5 | Entrega | Un solo PR con `size:exception` (estrategia `single-pr`). |

## Áreas afectadas

- `internal/project/infrastructure/postgres/migrations/000013_create_defects.{up,down}.sql` (nuevo).
- `tests/integration/migrations/migration_files_test.go` (13 pares).
- `internal/defect/{domain,application,infrastructure/postgres,transport/http}` (nuevo).
- `internal/api/api.go` y `cmd/api/main.go` (dependencias, bandera `Defect`, ruta con gate `>= 13`).
- `tests/unit/defect/...`, `tests/unit/cmd/api/...`, `tests/integration/defect/postgres/...` (nuevo).
- `openspec/specs/defecto/spec.md` (nuevo, al archivar), fila nueva en la tabla de rutas del `README.md` y nota de US-16 en `openspec/config.yaml`.

## Tamaño estimado

Aproximadamente 2.200–2.600 líneas con tests (migración 40, dominio 110, aplicación 120, repositorio 130, HTTP 200, cableado 60, tests unitarios ~1.000, integración ~550, spec y ajustes ~120). Supera el presupuesto de 400 líneas; se pide `size:exception` con un PR único.

## Riesgos

- Que otro cambio tome el número `000013` antes de integrar: reconfirmar la cabeza del directorio de migraciones antes del PR (el riesgo real ya se materializó y se resolvió con US-15 → `000012` mergeada; `main` está actualizada).
- El catálogo de estados debe anticipar US-17: el `CHECK` de la base incluye `resuelto` desde el inicio para que la US-17 no requiera reemplazar la restricción, y solo el alta aplica el subconjunto inicial.
- La spec debe dejar explícito que el defecto queda asociado a la historia y al Sprint de detección del proyecto, sin exigir coherencia entre ambos, para que US-18 y US-19 (conteo de detectados/resueltos por Sprint) no deban suponer reglas no documentadas.
- Los tests de integración requieren Docker (Testcontainers); en esta máquina ya se verificó que el patrón de US-15 corre completo.
- Registro de un defecto cuya historia ya no pertenecerá en el futuro al Sprint de detección indicado: las FKs no lo expresan y ninguna US lo exige; queda explícito en la spec como comportamiento NO exigido (integridad por proyecto, no por asignación).

## Listo para propuesta

Sí.
