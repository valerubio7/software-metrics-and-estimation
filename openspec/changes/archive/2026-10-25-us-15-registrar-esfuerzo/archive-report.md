# Reporte de Archivo — US-15 Registrar esfuerzo realizado en una historia o tarea

**Fecha de archivo**: 2026-10-25  
**Estado**: Archivado. Implementación completa; 47 de 48 tareas marcadas en el `tasks.md` archivado. La tarea 7.5 (cierre) queda sin marcar en ese archivo y su cierre se registra en este reporte (ver "Observaciones para el orquestador").  
**Rama**: `feat/us-15-registrar-esfuerzo`  
**PR de entrega**: pendiente. No se abrió ni se envió nada al remoto; abrir el PR con la etiqueta `size:exception` es decisión del usuario.

## Resumen ejecutivo

El cambio US-15 "Registrar el esfuerzo realizado en una historia o tarea" queda archivado. Consolida la capacidad nueva `esfuerzo` en `openspec/specs/esfuerzo/spec.md` (17 requisitos, 62 escenarios) e implementa la ruta `POST /projects/{project_id}/stories/{story_id}/effort-records` con la migración `000012_create_effort_records`, el dominio, el caso de uso, el repositorio PostgreSQL, el handler HTTP y el cableado con gate `version >= 12`.

La verificación final dio `PASS WITH WARNINGS` (0 CRITICAL). Después de ese informe, el commit `73bf159` resolvió el WARNING 2 (spec frente a diseño) y el sospechoso de NUL. Siguen abiertos el WARNING 1 (inestabilidad de Docker en pruebas de integración) y el WARNING 3 (gate de `cmd/api/main.go` sin prueba ejecutable). No hay una corrida completa única en verde de `go test -p 1 ./...`.

## Estado de implementación

- **Tareas en el archivo**: 47 `[x]` y 1 `[ ]` (7.5), con los bytes originales de `tasks.md`.
- **Cierre de la tarea 7.5 (registrado aquí)**: la consolidación de `specs/esfuerzo/spec.md` en `openspec/specs/esfuerzo/spec.md` está hecha y verificada con `diff -r`; el cambio está archivado; el PR con la etiqueta `size:exception` sigue pendiente de la decisión del usuario.
- **Tareas 1.1 a 7.4**: completadas según `tasks.md`.

### Commits de la rama (orden cronológico)

| Commit | Mensaje | Alcance |
|---|---|---|
| 11019a1 | docs(esfuerzo): add SDD planning artifacts for US-15 | Artefactos de planificación |
| af25b91 | feat(db): add effort records migration | UT 1: migración `000012` |
| 3e06cef | feat(effort): add effort record domain | UT 2: dominio `EffortRecord` |
| d2353e6 | feat(effort): add register effort use case | UT 3: caso de uso |
| cf4d223 | feat(effort): persist effort records in PostgreSQL | UT 4: repositorio PostgreSQL |
| a8cf4d4 | feat(effort): agregar handler HTTP de registro de esfuerzo | UT 5: handler HTTP |
| 8ea4329 | feat(effort): exponer la ruta de registro de esfuerzo | UT 6: cableado, gate, README y `config.yaml` |
| 11457e8 | docs(esfuerzo): registrar la verificación final de la fase 7 | Evidencia de la fase 7 |
| 8d4d205 | docs(esfuerzo): agregar informe de verificación de la US-15 | Informe de verificación |
| 73bf159 | fix(effort): rechazar el carácter NUL en la descripción | Corrección de la tarea 7.4 |

## Verificación

Fuentes: `verify-report.md` (archivado; snapshot del commit `11457e8`, escrito antes de `73bf159`) y los hechos finales del orquestador. El informe de verificación es un registro intermedio: sus afirmaciones de "pendiente" solo valen para el momento en que se escribió.

### Veredicto y cambios posteriores

- **Veredicto del informe**: `PASS WITH WARNINGS`, 0 CRITICAL, 3 WARNING, 6 SUGGESTION (en `11457e8`).
- **WARNING 2 (spec frente a diseño: FK de proyecto)**: **resuelto** en `73bf159`. El requisito R15 del delta ya no exige una FK de proyecto; la inexistencia del proyecto se verifica dentro de la transacción de inserción (`ErrProjectNotFound`), y `MapConstraintError` traduce tres FKs (integrante, tarea, historia). Verificado en `specs/esfuerzo/spec.md`.
- **Sospechoso de NUL (`U+0000`) en la sección 5, "Extras"**: **resuelto** en `73bf159`. La descripción con NUL responde `422` (R8, escenario nuevo) en lugar de `500`. La corrección se hizo con RED observado en HTTP, según la tarea 7.4 de `tasks.md`.
- **Matriz de trazabilidad (R15 y R17 en PARTIAL)**: el escenario de R15 que nombraba una FK de proyecto dejó de existir en la forma verificada tras la alineación de la spec. No hubo una nueva corrida de `sdd-verify` para re-verificarlo.

### Warnings que siguen abiertos

- **WARNING 1: inestabilidad de Docker en pruebas de integración.** Las funciones de espera de PostgreSQL usan un plazo fijo de 10 s y fallan con `wait for PostgreSQL ... unexpected EOF` antes de ejecutar código de la aplicación. El patrón aparece en los helpers de `effort`, `task`, `sprint`, `project` y `story`. Causa no confirmada: la contención de Docker Desktop es una hipótesis compatible con el precedente de US-10, no verificada con logs del daemon.
- **WARNING 3: gate de `cmd/api/main.go` sin prueba ejecutable.** El gate `>= 12` se verifica por `ResolveMigrationReadiness` (con pruebas) y por inspección del bloque de `main.go`.

### Evidencia de pruebas (registro intermedio, no re-ejecutado en esta fase)

- Pruebas unitarias y de migraciones: verdes en `11457e8` (`verify-report.md`: 21 paquetes con pruebas en `internal/effort/...` y `tests/unit/...`; migraciones 12 pares).
- Paquete de integración `tests/integration/effort/postgres`: verde tras `73bf159` (172,6 s, según `apply-progress.md`, tarea 7.4).
- Dos corridas completas `go test -p 1 ./...` sobre `8ea4329` (según `apply-progress.md`, Fase 7 y hechos del orquestador): en cada corrida falló un paquete distinto por `unexpected EOF` en `wait for PostgreSQL` (corrida A: `tests/integration/effort/postgres`; corrida B: `tests/integration/story/postgres`). Según el orquestador, cada paquete pasó al menos una vez entre ambas corridas.
- Una corrida previa con paralelismo por defecto, registrada en `apply-progress.md`, también falló en `story` (límite de 600 s) y `task` (`unexpected EOF`).
- **No existe una corrida completa única en verde.** Esta fase de archivo no volvió a ejecutar pruebas.

## Tamaño y entrega

- **Pronóstico** (`tasks.md`): ~1.450–1.650 líneas; riesgo de presupuesto de 400 líneas: `High`; estrategia `single-pr` con `size:exception` aceptado por el usuario.
- **Medición real** (`git diff --numstat` contra `main`, base `8307895`):
  - Código y pruebas (excluido `openspec/`): 3.392 adiciones + 4 eliminaciones = **3.396 líneas**, unas 8,5 veces el presupuesto de 400.
  - Total de la rama, incluidos los artefactos SDD bajo `openspec/`: 5.408 + 5 = **5.413 líneas**.
- Los commits `73bf159`, `11457e8` y `8d4d205` forman parte de esta medición.

## Especificación consolidada

- **Capacidad nueva**: `openspec/specs/esfuerzo/spec.md` (no existía una main spec previa, así que no hubo composición).
- **Método**: copia mecánica del delta con `cp` y `diff -r` vacío antes y después de publicar. No se usó `sdd-archive-compose`, porque la ruta sin main spec es la copia directa.
- **Contenido**: 17 requisitos y 62 escenarios (contados en el archivo con `### Requirement:` y `#### Scenario:`).
- **Encabezado**: el archivo conserva `## ADDED Requirements` del delta, porque la copia debe ser byte a byte. El precedente de US-10 cambió ese encabezado a `## Requirements` en la main spec, y el de US-01 lo conservó. Normalizarlo sería un cambio aparte y requeriría su propia verificación.
- **Fin de línea**: LF (0 bytes CR), igual que el delta y que el índice de git. Las main specs existentes tienen CRLF en el directorio de trabajo porque `core.autocrlf=true`; en el repositorio se guardan como LF.

## Artefactos preservados

Carpeta `openspec/changes/archive/2026-10-25-us-15-registrar-esfuerzo/`:

- `proposal.md`, `design.md`, `exploration.md`, `tasks.md`, `apply-progress.md`, `verify-report.md`, `specs/esfuerzo/spec.md`: presentes y byte a byte iguales al snapshot previo al movimiento.
- `archive-report.md`: este reporte (aditivo; no existía en el origen).

## Evidencia de copia (`diff -r`)

Salida verbatim de las comprobaciones. Una salida vacía es el único resultado válido.

1. Main spec, antes de publicar (`diff -r openspec/changes/us-15-registrar-esfuerzo/specs/esfuerzo/spec.md <temporal>`): sin salida, `diff_status=0`.
2. Main spec, después de publicar (`diff -r openspec/changes/us-15-registrar-esfuerzo/specs/esfuerzo/spec.md openspec/specs/esfuerzo/spec.md`): sin salida, `post_diff_status=0`.
3. Carpeta (`diff -r <snapshot previo al movimiento>/source openspec/changes/archive/2026-10-25-us-15-registrar-esfuerzo`): sin salida, `diff_status=0`. El snapshot de 7 archivos se tomó en el directorio temporal de la sesión antes de `git mv`.

## Riesgos residuales

1. **Inestabilidad de Docker en helpers de integración compartidos** (WARNING 1): espera fija de 10 s y `unexpected EOF`; afecta a varios paquetes. Causa no confirmada.
2. **Sin corrida completa única en verde** de `go test -p 1 ./...` sobre un mismo commit.
3. **Round-trip de `work_date` y `FOR KEY SHARE` sin prueba aislada** (SUGGESTION a y b): `time.Parse` ya rechaza formatos no canónicos, y la FK mantiene la integridad; el mapeo `23503` se prueba de forma sintética.
4. **Gate de `cmd/api/main.go`** verificado solo por `ResolveMigrationReadiness` y por inspección (WARNING 3).
5. **Encabezado `## ADDED Requirements` en la main spec**: cosmético, sin efecto sobre la verificación de los requisitos.
6. **Tamaño**: 3.396 líneas de código y pruebas frente a un presupuesto de 400; aceptado con `size:exception`.
7. **Diferencia de diseño documentada**: la decisión 5 de la propuesta hablaba solo de campos del cuerpo; la spec y el handler también responden `422` con `fields` para `project_id` y `story_id` no UUID, igual que en `tarea` (SUGGESTION e del informe).
8. **Prueba HTTP ausente** para `hours` ausente o `null` (cubierta en dominio y aplicación; extra del informe).

## Observaciones para el orquestador

1. **Tarea 7.5 sin marcar en el archivo**: el contrato de archivado prohíbe reparar casillas y exige que `tasks.md` archivado conserve sus bytes originales. Por eso no se marcó `[x]` en el archivo. Si se desea marcarla, debe hacerse como edición explícita y separada, con su propia verificación.
2. **Staging**: `git mv` dejó siete renombres en el índice; `openspec/specs/esfuerzo/` aparece como no rastreado. Falta añadirlo al commit del archivado.
3. **Seguimiento indicado por el orquestador** (no verificado en el repositorio): abrir el PR con la etiqueta `size:exception`; la tarea TUI-08 (#101) depende de esta ruta; US-19 agregará esfuerzo por `story_id`.

## Fuentes y trazabilidad

- **Artefactos leídos** (almacén `openspec`, localizadores de ruta del repositorio): `proposal.md`, `specs/esfuerzo/spec.md`, `design.md`, `tasks.md`, `verify-report.md` (intermedio), `apply-progress.md` (intermedio).
- **Observaciones de Engram**: ninguna leída. El almacén es `openspec` y todos los localizadores eran rutas del repositorio.
- **Jerarquía de fuentes aplicada**: 1) `tasks.md` archivado; 2) hechos finales del orquestador (`73bf159`, cierre de 7.5, estado del PR); 3) `verify-report.md` y `apply-progress.md` como snapshots.

## Cierre

El cambio `us-15-registrar-esfuerzo` queda archivado el 2026-10-25 en `openspec/changes/archive/2026-10-25-us-15-registrar-esfuerzo/`, y la capacidad `esfuerzo` queda consolidada en `openspec/specs/esfuerzo/spec.md`. La implementación está completa en la rama con 6 commits de unidades de trabajo, una corrección y dos commits de documentación de verificación. La verificación fue `PASS WITH WARNINGS`, con 0 CRITICAL; los WARNING 1 y 3 siguen abiertos y no hay una corrida completa única en verde. El PR sigue pendiente de la decisión del usuario.
