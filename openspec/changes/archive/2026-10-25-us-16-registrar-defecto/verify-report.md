# Informe de verificación: us-16-registrar-defecto

- Commit verificado: `155524d`, rama `feat/us-16-registrar-defecto`.
- Modo: TDD estricto, almacén `openspec`. Esta verificación cumple la tarea 7.2 de `tasks.md`; no se modificó código ni pruebas.
- Issue contrastado: **#44 (US-16 — Registrar un defecto detectado durante el desarrollo)**, revisto con `gh issue view 44` el 2026-10-25.
- **Veredicto: PASS** (0 CRITICAL, 0 WARNING, 3 SUGGESTION).

## 1. Ejecución real

| Comando | Resultado |
|---|---|
| `go build ./...` | exit 0, sin salida |
| `go vet ./...` | sin hallazgos |
| `gofmt -l internal tests` | sin salida |
| `go test ./internal/... ./tests/unit/... -count=1` | ok: 26 paquetes con pruebas, 0 FAIL |
| `go test -p 1 ./tests/integration/... -count=1` | ok: los 8 paquetes (defect 48,8 s; effort 86,1 s; migrations 0,0 s; project 10,8 s; projectmember 14,1 s; sprint 96,2 s; story 236,3 s; task 44,4 s), con Testcontainers y Docker disponible |

## 2. Contraste del issue: criterio por criterio

| Criterio de aceptación del issue #44 | Estado | Evidencia |
|---|---|---|
| Registrar un defecto dentro de un proyecto | ✔ | `POST /projects/{project_id}/defects`; e2e real `TestDefectHTTPEndToEnd/...201` |
| Registrar una descripción | ✔ | obligatoria, recortada, 1–500 caracteres, sin NUL; `TestNewDefectDescription*`, test HTTP `empty_fields`/`too_long_description`/`nul_description` |
| Registrar su severidad | ✔ | catálogo cerrado `alta`/`media`/`baja`; CHECK de integración y `TestNewDefectSeverityErrors` |
| Registrar su estado | ✔ | `detectado`/`en_correccion` al alta; `resuelto` rechazado con `422` y mensaje propio (`resuelto_status`) |
| Asociación con la historia relacionada | ✔ | FK compuesta `(story_id, project_id)`; verificación transaccional; historias de backlog y de Sprint cerrado aceptadas; `TestDefectRepositoryAcceptsBacklogStory`, `...ClosedDetectionSprint` |
| Registrar el Sprint de detección | ✔ | obligatorio del proyecto; `sprint_not_found` si es ajeno o inexistente (unitario + e2e) |
| Pendiente de registrar Sprint de resolución mientras no esté resuelto | ✔ | el alta nunca persiste `resuelto`; la columna de resolución no existe hasta US-17 (US-16 no la adelanta); `status_resuelto_is_valid_storage` verifica que `resuelto` sí es valor válido de almacenamiento para la US-17 |
| Falta información obligatoria → no se registra | ✔ | `422 validation_failed` con `fields` acumulados; cero filas tras cada fallo (unitario con fake, e2e en Postgres) |
| Datos válidos → el defecto queda registrado | ✔ | `201` con las 7 claves y la fila persistida verificada por lectura directa |

Criterios no cubiertos deliberadamente (acuerdo con el propio issue): resolución (US-17), consulta (US-18), métricas (US-19) y TUI (TUI-09). El spec delta los excluye con un Requirement dedicado ("Ofrecer únicamente el registro de defectos") y las rutas de consulta responden `404` (sin registro).

## 3. Trazabilidad spec ↔ pruebas

| Requirement (spec delta `defecto`) | Pruebas principales |
|---|---|
| Registrar un defecto de un proyecto (201, 405) | `TestRegisterDefectResponds201WithExactBody`, `...RejectsMethod`, e2e 201 |
| Asignar el identificador en el servidor | `id` generado (`countingID` unitario + `api.NewProjectID` e2e); `id` inyectada → 400 |
| Severidad contra catálogo cerrado | `TestNewDefectSeverityErrors`, tablas de severidad, CHECK de integración |
| Estado contra el subconjunto inicial | `TestNewDefectStatusErrors` (`resuelto` con mensaje propio), HTTP `resuelto_status` |
| Descripción del defecto | `TestNewDefectDescription*` (trim, 500/501 multibyte, NUL, vacía), HTTP equivalentes |
| Identificadores del cuerpo | unitario de aplicación (vacío/inválido/canónico), HTTP `invalid_uuid`/`missing_story` |
| Reportar todos los errores juntos | `TestNewDefectAccumulatesAllFieldErrors`, HTTP `accumulated` (5 campos) |
| Rechazar solicitudes malformadas | `TestRegisterDefectMalformedBodies` (10 casos de cuerpos) |
| Referencias dentro del proyecto con precedencia | `TestDefectRepositoryReferencePrecedence`, e2e real de precedencia, `TestRegisterDefectReferenceErrors` |
| Historias en cualquier situación sin alterar datos | `TestDefectRepositoryAcceptsBacklogStory`, `...ClosedDetectionSprint`, integración "no modifica datos ajenos" vía conteo de padres intactos |
| Permitir defectos duplicados | `TestDefectRepositoryAllowsExactDuplicates` (3 filas), e2e (2 posts → 2 filas) |
| Alta atómica | triggers de integración (FK violada → `ErrStoryNotFound`, error arbitrario → pasa sin traducir), transacción con rollback |
| Integridad en la base de datos | CHECK y FK de migración con violaciones directas (`23514`/`23503` nombradas) |
| Habilitar con esquema en versión 13 | `TestMigrationReadinessResolvesDefectByVersion` (11/12/13/14/dirty/lookup), e2e la composición real |
| Ofrecer únicamente el registro | expectativa de 7 claves exactas, sin rutas de consulta; `api.HTTPDependencies{}` → 404 |

## 4. Calidad del proceso

- RED observado por UT (paquetes inexistente, migraciones faltantes, campos faltantes de API) registrado comando por comando en `apply-progress.md`.
- Cada commit compila con la suite verde; los 6 commits de código más 3 de documentación separan el volumen para revisión.
- Los ciclos cortos detectados (JSON NUL con escape, precedencia del proyecto en `defectof`, ID de subtest duplicado, 405 por el propio mux) se documentaron honestamente en `apply-progress.md`.

## 5. Contraste con el backlog: errores del spec original a partir del issue

No hubo desviaciones: todas las decisiones se registraron antes de escribir código (severidad, estados, Sprint de detección, duplicados, entrega) y el issue no introduce reglas adicionales sin cobertura.

## 6. Sugerencias (no bloqueantes)

1. **SUGGESTION**: el `CHECK` de la descripción usa `char_length BETWEEN 1 AND 500`; también es alcanzable escribir el NUL directamente en SQL por vías fuera del caso de uso (PostgreSQL rechaza el byte NUL en `text` por SQLSTATE 22021, así que la vía directa de SQL no puede persistirlo). Cobertura ya suficiente; si la US-17 necesita un `CHECK` adicional, agregarlo en esa migración.
2. **SUGGESTION**: `defects_project_seq_idx (project_id, seq)` se superpone con los dos índices compuestos de historia y Sprint; es un índice de acceso directo para US-18 por proyecto, costo de escritura mínimo. Si el listado de US-18 cambia su orden, revisar su utilidad.
3. **SUGGESTION**: la protección del catálogo de "Sprint de detección cerrado" es explícita en la spec; si el equipo quiere renombrar la columna `sprint_id` a `detection_sprint_id` en algún momento, producirlo como migración dedicada para no romper TUI-09 (el issue usa "Sprint de detección").

## 7. Veredicto

**PASS.** El issue #44 queda satisfecho y los entregables de US-16 (migración, módulo completo, cableado con gate, README y nota de config) están verificados contra el schema real con Testcontainers. El paso a archivar queda listo: consolidar `openspec/specs/defecto/spec.md` desde el spec delta y archivar el cambio.
