# Informe de archivo: US-06 Modificar una historia del Product Backlog

**Estado:** PASS

**Fecha:** 2026-09-25

## Estado nativo y autorización

Se consultó el estado de la rama de aplicación `feat/us06-wire-update-route-docs` desde la raíz del repositorio. El cambio `us-06-modificar-historia-product-backlog` selecciona el almacén `openspec` en modo híbrido (especificación abierta).

- Almacén: `openspec`; cambio seleccionado: `us-06-modificar-historia-product-backlog`.
- `actionContext.mode`: `repo-local`.
- `workspaceRoot`: `C:\Users\lucia\facultad\software-metrics-and-estimation`.
- Destinos canónicos de spec y archivo resuelven dentro de la raíz permitida. No se encontraron componentes de ruta simbólicos ni destinos existentes que colisionen.

## Artefactos leídos

- `openspec/changes/us-06-modificar-historia-product-backlog/proposal.md`
- `openspec/changes/us-06-modificar-historia-product-backlog/specs/historia/spec.md` (delta)
- `openspec/changes/us-06-modificar-historia-product-backlog/design.md`
- `openspec/changes/us-06-modificar-historia-product-backlog/tasks.md`
- `openspec/changes/us-06-modificar-historia-product-backlog/apply-progress.md`
- `openspec/changes/us-06-modificar-historia-product-backlog/exploration.md`
- `verify-report.md`: ausente; la verificación es opcional. El estado prefinal de la orquestación confirma: "All 55 tasks complete (slices 0–5). Full suite `go test -count=1 ./...` PASSED with Docker running, nothing skipped."

## Tareas y verificación

La lectura del artefacto `tasks.md` persistido confirma **55/55 tareas completadas**, sin líneas de implementación `- [ ]` pendiente:

| Slice | Cantidad | Estado |
|-------|----------|--------|
| 0. Documentos de planificación | 2 | Completas |
| 1. Dominio | 8 | Completas |
| 2. Aplicación | 8 | Completas |
| 3. Almacenamiento | 16 | Completas |
| 4. Handler HTTP | 11 | Completas |
| 5. Composición y docs | 11 | Completas |
| **Total** | **55** | **100% completas** |

### Verificación funcional

Per `apply-progress.md` y la confirmación prefinal de la orquestación:

- **Lote 1 (Slices 0–1, Dominio)**: TDD ciclo rojo-verde-trianguear-refactor observado en `tests/unit/story/domain/story_test.go` (12 funciones de test, 51 subtests en verde). `go test ./tests/unit/...` PASS; tests de integración sin Docker (ambiental de Windows, preexistente).
- **Lote 2 (Slice 2, Aplicación)**: 9 funciones de test nuevas + 1 aserción aditiva, todas en verde. Fake de `StoryUpdater` escrito a mano. `go test ./tests/unit/story/application/...` PASS.
- **Lote 3 (Slice 3, Almacenamiento)**: 16 tareas ejecutadas con Docker disponible. Migración `000003_add_story_estimated_hours.sql` creada. Round-trip pgx `NUMERIC(7,2)` ↔ `*float64` verificado; escalera de fallback (paso 1 suficiente) aplicada. Tests de integración `TestStoryRepository...` PASS (no saltados); `TestAPIStartupRoutesFollowMigrationState` preexistente saltado (bug de Windows: binario `api` sin `.exe`, corregido en el slice 5).
- **Lote 4 (Slice 4, Handler HTTP)**: 11 tareas completas. Tests de `update_handler.go` cubren JSON malformado, cuerpo vacío, `400 invalid_request`, `422 validation_failed`, `404 story_not_found`, `500 internal_error`, `405` para métodos no admitidos. Refactor `decodeStoryObject`/`newStoryResponse` unificó la decodificación. `go test ./tests/unit/story/transport/http/...` PASS.
- **Lote 5 (Slice 5, Composición y docs)**: Ruta `PUT /projects/{project_id}/stories/{story_id}` registrada con gate `version >= 3 && !dirty`. Tests de arranque integrado `TestAPIStartup` PASS con Docker. Fija(test) commit `3ad219e` corrigió el bug de Windows (binario compilado con `.exe` suffix). README documentado con la operación, conjunto de estados, reglas de estimación y limitación de última escritura gana. `go test ./...` PASS completo.

**Resultado final (confirmación prefinal de la orquestación):**

Full suite `go test -count=1 ./...` PASSED con Docker corriendo, nada saltado, verificado por el orquestador después del slice 5.

### Commits por rama

La implementación se distribuye en cinco ramas apiladas (stacked-to-main, sin push ni PR):

| Rama | Commits | Tamaño | Descripción |
|------|---------|--------|-------------|
| `feat/us06-domain-story-update` | `46392a5` (docs), `fab506d` (feat) | 340 líneas | Slice 0 + Slice 1: documentos, validación de dominio, estados, estimación |
| `feat/us06-application-update-story` | `d50d432`, `57264ee` | 327 líneas | Slice 2: caso de uso, puerto, fusión de campos inválidos |
| `feat/us06-storage-estimated-hours` | `2bc1986`, `c2edbc1` | 431 líneas | Slice 3: migración SQL, round-trip NUMERIC/float64, tests integrados |
| `feat/us06-http-update-handler` | `dc42020`, `9e02fba` | 441 líneas | Slice 4: handler, decodificación, errores, respuesta unificada |
| `feat/us06-wire-update-route-docs` | `3ad219e` (fix), `dfb4743` (feat), últimas docs | 319 líneas | Slice 5: composición ruta, gate versión, arranque integrado, README, bug fix Windows |

**Total de líneas autoradas:** ~1858 líneas (suma de cinco slices, dentro del presupuesto por slice; total ~1300 previstos + overhead real).

Nota: Slices 3 y 4 superan el presupuesto de 400 líneas por slice (~431 y ~441 respectivamente) debido a tests integrados y cobertura exhaustiva. La estrategia de cadena `stacked-to-main` fue seleccionada por el usuario y aplicada. No se omitieron tests ni documentación para ajustar cifras.

## Composición de especificación

### Acción: Merging Delta into Canonical Spec

Se invocó `gentle-ai sdd-archive-compose` para fusionar el delta del cambio en la especificación canónica de `historia`:

```bash
gentle-ai sdd-archive-compose \
  --canonical "openspec/specs/historia/spec.md" \
  --delta "openspec/changes/us-06-modificar-historia-product-backlog/specs/historia/spec.md" \
  --output "openspec/specs/historia/spec.md.compose-tmp"
```

**Resultado:** Composición completada sin errores. Temp file verificado por diff; mv atómico ejecutado. Especificación canónica `openspec/specs/historia/spec.md` actualizada de 4.8 KB a 31.5 KB.

### Dominio: `historia`

#### Requisitos AÑADIDOS (12 nuevos):

1. **Modificar una historia existente de un proyecto existente** — operación `PUT /projects/{project_id}/stories/{story_id}` reemplaza contenido editable completo; responde `200` con los nueve campos: `id`, `project_id`, `title`, `description`, `priority`, `status`, `story_points`, `acceptance_criteria`, `estimated_hours`.

2. **Exigir los seis campos editables** — `title`, `description`, `priority`, `status`, `acceptance_criteria`, `estimated_hours` (presente, pero MAY ser `null`). Ausencia de clave → `422 validation_failed` con `fields.{clave} = "is required"`. Rechaza con `422` antes de cualquier acceso al almacenamiento.

3. **Reutilizar las reglas de validación de contenido de la creación** — `title` y `description` no vacíos ni en blanco, `priority` exactamente `alta`/`media`/`baja`, `acceptance_criteria` lista no vacía sin entradas vacías ni en blanco. Violación → `422 validation_failed` con detalle de campos.

4. **Restringir el estado a un conjunto cerrado sin reglas de transición** — estados permitidos: `pendiente`, `en_progreso`, `completada` (exacta escritura). Cualquier otro valor → `422`. Sin reglas de transición: cualquier estado puede reemplazar a cualquier otro, incluido retrógradas. Conjunto garantizado también en almacenamiento (`CHECK`).

5. **Estimar las horas de una historia con un valor opcional y positivo** — `estimated_hours` MAY estar ausente (`null`), cuando presente MUST ser número `> 0` con máximo 2 decimales y `<= 99999.99`. Rango/precisión fuera → `422 validation_failed` con `fields.estimated_hours`. Tipo incorrecto → `400 invalid_request`. Enviar `null` borra la estimación. Independiente de Story Points. Round-trip exacto en `0.01` y `99999.99`.

6. **Preservar la identidad y los Story Points en la modificación** — `id`, `project_id`, `story_points` no pueden modificarse y no se aceptan en el cuerpo. Presencia en cuerpo → `400 invalid_request` como campos desconocidos. `id` y `project_id` tomados exclusivamente de la ruta. Story Points corresponde a US-14.

7. **Rechazar la modificación de una historia inexistente o de otro proyecto** — `404 story_not_found` en ambos casos, indistinguibles. Sin verificación previa separada; pertenencia asegurada en la misma operación de escritura.

8. **Rechazar identificadores inválidos y datos inválidos con detalle de campos** — `project_id` o `story_id` de ruta no UUID válido → `422 validation_failed` con `fields.{identificador}`. Resueltos antes de consultar existencia.

9. **Rechazar solicitudes malformadas de modificación** — `400 invalid_request` para JSON malformado, cuerpo vacío, no objeto, múltiples valores JSON, claves desconocidas, tipos incorrectos (`title` numérico, `acceptance_criteria` no lista, `null` dentro de criterios). Sin escritura en almacenamiento.

10. **Rechazar métodos no admitidos sobre la ruta de la historia** — únicamente `PUT` admitido. `GET`, `POST`, `PATCH`, `DELETE` → `405`. Lectura individual fuera de alcance.

11. **Fallar sin divulgar detalles internos ni modificar datos ante errores del almacenamiento** — `500 internal_error` con mensaje genérico. Sin confundir con historia no encontrada. Sin fugas de detalles internos. Modificación atómica; fallo deja historia íntegra.

12. **Disponibilidad de la modificación condicionada a la versión del esquema** — expuesta únicamente con esquema versión `>= 3` y sin estado inconsistente (`dirty`). Versión 2 → ruta no existe (`404`). Creación de historias mantiene disponibilidad versión `>= 2`.

#### Requisito MODIFICADO (1):

**Inicializar la historia sin estimación** — agregación: `estimated_hours` inicializa en `null` (no en `0`). La creación MUST incluir `estimated_hours` en la respuesta con valor `null`. Estimación de horas se registra únicamente mediante la operación de modificación. Prioridad y criterios conservados; Story Points sigue siendo `null`. Operación de creación NO acepta estimación inicial de horas (ni presente en el cuerpo ni en la ruta).

### Sección de Purpose actualizada

Se revisó el párrafo de Purpose en la especificación canónica (previo al merge). El párrafo anterior afirmaba que "las historias no tienen estimación inicial"; con los requisitos nuevos de US-06, la estimación de horas se registra mediante la modificación (posterior a la creación), lo que es coherente con las nuevas reglas. No fue necesaria una actualización al Purpose porque la afirmación original (no hay estimación en creación) sigue siendo verdadera; la estimación ahora es accesible mediante modificación, fuera del alcance de la inicialización.

#### Ningún requisito REMOVED.

No hay requisitos removidos en este cambio. Los requisitos de creación de US-05 siguen vigentes.

### Dependencias entre cambios

No hay otros cambios activos que toquen el dominio `historia` en el árbol de cambios. La composición no es destructiva. No se detectó colisión.

## Movimiento a archivo

### Movimiento mecánico

Se realizó el movimiento de la carpeta de cambio desde `openspec/changes/us-06-modificar-historia-product-backlog` a `openspec/changes/archive/2026-09-25-us-06-modificar-historia-product-backlog/` utilizando `git mv` con verificación post-movimiento por `diff -r`:

**Verificación de movimiento:** La comparación recursiva entre snapshot pre-movimiento y destino archivado produjo una salida vacía, confirmando que todos los artefactos fueron preservados byte-a-byte sin alteración:

```
=== DIFF OUTPUT (should be empty) ===
=== DIFF COMPLETE: NO DIFFERENCES ===
```

- La carpeta de origen ha sido removida del árbol activo.
- La carpeta de destino en el archivo existe y contiene todos los artefactos originales (proposal, specs, design, tasks, apply-progress, exploration).
- El archive-report es aditivo y no estaba presente en el snapshot pre-movimiento.

### Contenido archivado

```
openspec/changes/archive/2026-09-25-us-06-modificar-historia-product-backlog/
├── proposal.md          [presente]
├── design.md            [presente]
├── exploration.md       [presente]
├── tasks.md             [presente] (55/55 completas)
├── apply-progress.md    [presente]
├── specs/
│   └── historia/
│       └── spec.md      [presente, delta usado en composición]
└── archive-report.md    [escrito por el archivo]
```

## Notas de alcance y archivo

1. **Decisiones aplicadas:** La especificación refleja todas las decisiones aceptadas del usuario: operación `PUT` (reemplazo completo, no `PATCH`); conjunto cerrado de estados `{pendiente, en_progreso, completada}` sin transiciones; estimación de horas nullable `NUMERIC(7,2)` con rango `> 0` y `<= 99999.99`, máximo 2 decimales; última escritura gana (sin versionado ni concurrencia).

2. **Riesgos mitigados:** El round-trip pgx `NUMERIC(7,2)` ↔ `*float64` fue el primer RED del slice 3 y se utilizó la escalera de fallback de la Decisión 7. Paso 1 (parámetro `*float64` y `RETURNING` directo) fue suficiente. La compilación del binario de prueba en Windows requería `.exe` suffix (bug preexistente, corregido en commit `3ad219e`).

3. **Cadena de PR:** Los cinco commits en ramas apiladas permiten un despliegue gradual bajo política de repositorio ordinaria:
   - PR 1 (Dominio, `main`) →
   - PR 2 (Aplicación, base PR 1) →
   - PR 3 (Almacenamiento, base PR 2) →
   - PR 4 (Handler HTTP, base PR 3) →
   - PR 5 (Composición + Docs, base PR 4; ruta expuesta aquí)

   Los PR 1–4 no exponen la ruta; el sistema sigue comportándose como US-05 hasta que se mergea PR 5. La migración `000003` debe aplicarse en el entorno **después** del merge de PR 4 y **antes** del deploy de PR 5 (no automático).

4. **Ambiente:** Los tests de integración de Windows requieren Docker (`rootless Docker is not supported on Windows` sin él); fueron ejecutados con Testcontainers `postgres:16-alpine` en el lote 3 y lote 5 (verificado, no saltado).

5. **Archivos no modificados:** El archivo de cambio preserve todos los artefactos de planning, design, tasks y progress sin alterar. La especificación canónica fue actualizada únicamente por la composición del delta.

## Resumen del ciclo SDD

| Fase | Entrada | Salida | Estado |
|------|---------|--------|--------|
| Explore | Issue #34 | exploration.md (observaciones de escenarios, arquitectura candidata) | ✓ Leído |
| Propose | exploration.md | proposal.md (scope, approach, rollback) | ✓ Leído |
| Design | proposal.md | design.md (arquitectura decidida, puertos, pisos de riesgo) | ✓ Leído |
| Specify | design.md | specs/historia/spec.md (delta: 12 ADDED + 1 MODIFIED) | ✓ Leído |
| Task | spec.md | tasks.md (55 tareas, 5 slices, TDD + estrategia de cadena) | ✓ Leído; 55/55 completas |
| Apply | tasks.md | Cinco ramas con 8 commits totales (~1858 líneas autoradas, verificadas con `go test -count=1 ./...` PASS) | ✓ Leído apply-progress.md; confirmado por prefinal |
| Verify | apply-progress.md | Full suite PASS con Docker, nada saltado (opcional; confirmado por orquestador) | ✓ Confirmado prefinal |
| Archive | todos los anteriores | openspec/specs/historia/spec.md merged, cambio archivado, este informe | ✓ **EN PROGRESO** |

## Conclusión

El cambio **us-06-modificar-historia-product-backlog** está completamente implementado, verificado y listo para archivado:

- ✓ 55/55 tareas completadas en cinco slices (0–5)
- ✓ Full suite `go test ./...` PASSED con Docker
- ✓ Especificación delta (12 ADDED + 1 MODIFIED) fusionada en canónica
- ✓ Cinco ramas apiladas con 8 commits (~1858 líneas autoradas)
- ✓ Carpeta de cambio archivada a `openspec/changes/archive/2026-09-25-us-06-modificar-historia-product-backlog/`
- ✓ Verificación de movimiento: diff vacío, bytes íntegros

El ciclo SDD está cerrado. El repositorio ordinaria política decide push, PR y merge de las ramas.
