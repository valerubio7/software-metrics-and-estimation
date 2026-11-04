# Reporte de archivado — TUI: acciones desde la historia (tui-refactor-acciones-desde-historia)

**Fecha de archivado**: 2026-11-04  
**Estado**: archivado. Implementación completa en 6 commits de unidades de trabajo. Verificación intermedia con 0 CRITICAL, 5 WARNING y 7 SUGGESTION, sin correcciones. Las pruebas de integración (Docker no disponible) y el recorrido manual con la API (tarea 7.4) no se ejecutaron.  
**Rama**: `feat/tui-refactor-acciones-desde-historia` (6 commits sobre `main`, en `9bda696`). El commit de archivado es local; su hash no figura en este archivo, para evitar una autorreferencia.  
**PR**: no abierto por esta fase. El orquestador lo abre después del archivado, con `size:exception` y estrategia `single-pr` (decisión 10 de la propuesta).  
**Momento del archivado**: el usuario pidió archivar antes del merge del PR. La tarea 7.7 de `tasks.md` prevé el archivado después del merge; esa condición no se cumple y se documenta en este reporte.

## Resumen ejecutivo

El cambio retira las pestañas Estimación y Esfuerzo. Desde la lista del Backlog, `Enter` abre el detalle de la historia, y desde ese detalle se ofrecen Estimar (`p`), Registrar esfuerzo (`t`) y Reportar defecto (`d`). La barra queda con cuatro pestañas (Dashboard, Backlog, Sprints y Defectos, dígitos `1` a `4`) y se retiran `P` y `E`. No se agregan endpoints ni requests HTTP.

El delta de `tui` se fusiona en la spec principal: 2 requisitos añadidos, 13 modificados y 1 retirado. La verificación intermedia dio 0 CRITICAL y 5 WARNING. Ninguna advertencia se corrigió: el usuario decidió archivar y abrir el PR tal como está.

## Specs consolidadas

| Dominio | Antes | Después | Cambios |
|---|---|---|---|
| `tui` (`openspec/specs/tui/spec.md`) | 37 requisitos | 38 requisitos | 2 ADDED al final; 13 MODIFIED reemplazados en su posición; 1 REMOVED |

- **ADDED**: «Detalle de la historia con sus acciones»; «Reportar un defecto desde la historia».
- **MODIFIED**: «Navegación de solo lectura»; «Operar el backlog mediante la TUI»; «Modificar historias desde la lista»; «Listas navegables con teclas uniformes»; «Estimar historias con Planning Poker»; «Elegir integrantes, historias, tareas y Sprints con selectores navegables»; «Elegir historia, tarea e integrante para registrar esfuerzo»; «Enviar el registro de esfuerzo y presentar los errores de la API»; «Retroceder en el flujo de esfuerzo y descartar respuestas tardías»; «Espacio de trabajo del proyecto con pestañas»; «Avisos efímeros en lugar de pantallas de resultado»; «Pie de teclas acotado y ayuda agrupada»; «Salir limpiamente desde el navegador y las pestañas».
- **REMOVED**: «Atajos contextuales de estimación y esfuerzo».

**Método**: `gentle-ai sdd-archive-compose --canonical openspec/specs/tui/spec.md --delta openspec/changes/tui-refactor-acciones-desde-historia/specs/tui/spec.md --output openspec/specs/tui/spec.md.compose-tmp` terminó con exit 0. Antes de moverlo, el archivo temporal se normalizó a CRLF (la salida mezclaba LF y CRLF) y se dejó una única línea final (la última requisición del delta arrastraba una línea en blanco). Después se movió con `mv`. Ninguno de los dos ajustes cambia el contenido de los requisitos, y lo comprueba el readback de abajo.

**Anotaciones `(Previously: …)`**: la spec principal conserva las líneas `(Previously: …)` de los requisitos modificados, igual que el archivado tui-13 (su delta trae las mismas líneas que hoy figuran en la spec principal). Pasaron de 13 a 20 líneas. Las anotaciones `(Reason: …)` y `(Migration: …)` no se trasladan, porque el requisito retirado sale completo de la spec.

**Readback estructural** (comparación por nombre de requisito, con los dos lados normalizados a LF):

```
head requirements: 37
new requirements:  38
delta ADDED=2 MODIFIED=13 REMOVED=1
duplicate names in new: none
ADDED    OK (count=1, byte-equal-to-delta=yes) x2
MODIFIED OK (count=1, byte-equal-to-delta=yes) x13
REMOVED  OK (absent-in-new=yes)
untouched requirements byte-identical to HEAD: 23/23
names in new not explained by HEAD/delta: none
expected names missing in new: none
untouched relative order preserved: yes
preamble (before first requirement) identical to HEAD: yes
PROBLEMS: 0
```

`git diff --numstat` contra `HEAD` para `openspec/specs/tui/spec.md`: 405 inserciones y 92 eliminaciones.

**Menciones residuales** tras la fusión. Búsqueda de «Estimaci», pestaña(s) con Esfuerzo, `1–6`, `1` a `6`, «seis», `P`/`E` y «Atajos contextuales» en la spec principal:

| Línea | Contexto | Clasificación |
|---|---|---|
| 64 | `(Previously:` … Estimación, Esfuerzo y Defectos) en Navegación de solo lectura | Histórica |
| 169 | Cuerpo de Operar el backlog: las teclas `P` y `E` ya no inician flujos | Negativa vigente |
| 170 | `(Previously:` … `P` y `E` iniciaban Estimación y Esfuerzo) | Histórica |
| 196, 198 | Escenario «`P` y `E` ya no inician flujos» | Negativa vigente |
| 199 | «la TUI no abre Planning Poker, Esfuerzo ni ningún flujo» | Negativa vigente (Esfuerzo como flujo) |
| 511 | `(Previously:` dígitos `1` a `6`) en Listas navegables | Histórica |
| 550 | `(Previously:` selector de la pestaña Estimación o atajo `P`) en Estimar | Histórica |
| 646 | `(Previously:` pestañas Estimación y Esfuerzo que se retiran) en selectores | Histórica |
| 692 | `(Previously:` pestaña Esfuerzo y atajo `E`) en Retroceder | Histórica |
| 869 | `(Previously:` pestaña Esfuerzo) en Enviar el registro | Histórica |
| 1028 | «No existen las pestañas Estimación ni Esfuerzo» en Espacio de trabajo | Negativa vigente |
| 1031 | `(Previously:` seis pestañas, dígitos `1` a `6`) | Histórica |
| 1038 | «la barra no muestra Estimación ni Esfuerzo» | Negativa vigente |
| 1114 | `(Previously:` pestaña Esfuerzo) en Avisos efímeros | Histórica |
| 1164 | `(Previously:` dígitos `1–6`) en Pie de teclas | Histórica |
| 1196 | «no lista `P` ni `E`» en la ayuda del Backlog | Negativa vigente |
| 1234 | `(Previously:` pestaña Esfuerzo, atajo `E`) en Retroceder | Histórica |

El resto de las apariciones de «Esfuerzo» (23 líneas en total) nombra el flujo vigente («Registrar esfuerzo», «selector de tareas de Esfuerzo», formulario de esfuerzo) y no una pestaña. El nombre del requisito retirado no aparece en la spec. No queda ninguna mención de pestañas Estimación o Esfuerzo, ni de `1–6`, presentada como comportamiento vigente.

## Artefactos archivados

Carpeta: `openspec/changes/archive/2026-11-04-tui-refactor-acciones-desde-historia/`

| Artefacto | Presente | Método |
|---|---|---|
| `proposal.md` | sí | `git mv` (rastreado) |
| `specs/tui/spec.md` (delta) | sí | `git mv` |
| `design.md` | sí | `git mv` |
| `exploration.md` | sí | `git mv` |
| `tasks.md` | sí, sin cambios | `git mv`; 54 de 61 casillas marcadas |
| `apply-progress.md` | sí, archivo físico | `git mv` |
| `verify-report.md` | sí, archivo físico | `mv` (sin rastrear en el origen; se añade en el commit de archivado) |
| `archive-report.md` | sí | este reporte (aditivo; no existía en el origen) |

## Verificación

Fuente: `verify-report.md` (archivo físico en la carpeta archivada), sección «Resultado», fechado 2026-11-04. Es un snapshot intermedio. La rama sigue en `e0cf312`, el último de los seis commits que ese informe verifica, así que su evidencia de código describe el estado final. Engram: `sdd/tui-refactor-acciones-desde-historia/verify-report` (#74).

Resultado de `sdd-verify`: **CRITICAL 0 · WARNING 5 · SUGGESTION 7**.

Resultados observados según el informe:
- `go build ./...` y `go vet ./...`: sin salida, exit 0.
- `gofmt -l` sobre los 25 archivos `.go` cambiados: formateados.
- `go test -count=1 -v ./internal/tui/... ./tests/unit/tui/...`: 361 pruebas de primer nivel en PASS, 0 FAIL, 0 SKIP.
- `go test -count=1 ./internal/... ./tests/unit/... ./cmd/...`: 31 paquetes `ok`.
- `go test -count=1 ./tests/integration/...`: no ejecutado en la práctica. 9 paquetes fallan con `rootless Docker is not supported on Windows`. El informe no los cuenta como regresión y no se confirmó contra `main`.
- Matriz requisito → escenario → prueba: 136 ítems (135 escenarios y el requisito REMOVED): 131 COMPLIANT, 4 PARTIAL (W1 y W2) y 1 UNTESTED (S-a).
- Requests: el diff de `internal/tui/api.go` y de los clientes está vacío; el cambio no agrega requests HTTP.

### Advertencias (WARNING), no corregidas

- **W1**: ninguna prueba afirma que las filas «Estimar», «Registrar esfuerzo» y «Reportar defecto» no se dibujan en los estados de carga, historia ausente y error de recarga (escenarios 12, 13 y 14 del requisito «Detalle de la historia con sus acciones»: PARTIAL).
- **W2**: el escenario de Estimar «Respuesta tardía de integrantes tras volver al detalle» no tiene prueba propia, y la guarda de pantalla de `applySectionResult` no tiene prueba que la proteja (PARTIAL).
- **W3**: tamaño real de ≈ 3.575 líneas sin `openspec/` (producción 616, pruebas 2.914, README 45) frente a ≈ 1.770 estimadas. Queda cubierto por el `size:exception` aceptado por el usuario.
- **W4**: evidencia TDD parcialmente no corroborable. Los RED históricos y las tablas de mutación no son reproducibles en esta verificación, y algunas pruebas de caracterización nacieron en verde.
- **W5**: verificación manual no ejecutada (tarea 7.4).

Sugerencias S-a a S-g: no corregidas; detalle en `verify-report.md`.

### Pruebas de integración y recorrido manual

- Tarea 7.1: parcial. Ejecutadas: `gofmt`, `go build`, `go vet` y las pruebas sin Docker. No ejecutadas: las pruebas de integración con Testcontainers, porque Docker no está disponible en este equipo.
- Tarea 7.4: no ejecutada. Requiere la API con PostgreSQL.

## Estado de tareas

`tasks.md` no se modificó. Hay 61 tareas: 54 marcadas (fases 1 a 6) y 7 sin marcar (fase 7). Además, quedan sin marcar 12 criterios de cierre de `tasks.md` y 10 criterios de éxito de `proposal.md`. Ninguno se marcó en el archivado.

Estado observado de la fase 7 (según `verify-report.md` y los hechos finales de la solicitud):

| Tarea | Estado observado |
|---|---|
| 7.1 | Parcial. Integración no ejecutada (Docker no disponible). |
| 7.2 | Revisión de criterios ejecutada en la verificación; casilla sin marcar. |
| 7.3 | Búsqueda de referencias ejecutada; sin referencias sueltas fuera de notas históricas. |
| 7.4 | No ejecutada. |
| 7.5 | No aplicable: no hubo correcciones y no se creó commit `fix(tui)`. |
| 7.6 | Cumple según la verificación: `apply-progress.md` físico y `git status` sin cambios ajenos. |
| 7.7 | Pendiente en su condición original (posterior al merge). Este archivado se hizo antes del merge, por solicitud explícita del usuario. |

## Commits de la rama

| Commit | Mensaje | Unidad de trabajo |
|---|---|---|
| `b3c10ca` | feat(tui): abrir el detalle de la historia desde el Backlog | UT 1 |
| `84850d9` | feat(tui): estimar y registrar esfuerzo desde el detalle | UT 2 |
| `a3d9f52` | feat(tui): reportar un defecto desde el detalle con la historia precargada | UT 3 |
| `7e5b074` | test(tui): recorrer los flujos de estimación y esfuerzo desde el detalle | UT 4 |
| `cd7979c` | refactor(tui): retirar las pestañas Estimación y Esfuerzo y el código muerto de secciones | UT 5 |
| `e0cf312` | docs(tui): actualizar el README para las acciones desde la historia | UT 6 |

Verificados con `git log main..HEAD` (6 commits).

## Estado final

Hechos finales de la solicitud de archivado. Prevalecen sobre `verify-report.md` y `apply-progress.md`, que son snapshots intermedios:

- Seis commits de unidades de trabajo en la rama, listados arriba.
- Verificación: 0 CRITICAL y 5 WARNING, en coincidencia con `verify-report.md`. Las advertencias W1 a W5 no se corrigieron: el usuario decidió archivar y abrir el PR tal como está.
- Tamaño: ≈ 3.575 líneas sin `openspec/` frente a ≈ 1.770 estimadas (W3), cubierto por `size:exception`.
- Supuesto S1 (la barra resalta Backlog en el detalle y en los flujos de acción): sigue sin confirmar por el usuario.
- Defecto preexistente de Planning Poker (doble envío en la confirmación del Story Points): fuera de alcance. `applyPokerSave` no tiene guarda; `tasks.md` (tarea 2.10) lo deja para un cambio aparte.
- Apertura del PR: la hace el orquestador después de este archivado, fuera de esta fase.
- Contradicciones: ninguna sin resolver. El informe de verificación recomendaba cubrir W1 y W2 antes de archivar; el usuario eligió archivar sin hacerlo.

## Evidencia de copia (`diff -r`)

Operación: `cp -R` del origen a un snapshot previo al movimiento; `git mv` para los 6 archivos rastreados; `mv` para `verify-report.md` (sin rastrear); `rmdir` solo de directorios vacíos. Comprobación de que el origen quedó ausente: sí.

Salida verbatim del readback (`diff -r` del snapshot contra el destino archivado):

```
=== diff -r (snapshot source vs archive destination) ===
DIFF_R_EXIT=0
```

No hubo líneas de diferencia. `archive-report.md` es aditivo y no existía en el origen, por lo que queda fuera de la comparación.

## Fuentes y trazabilidad

- Almacén: `openspec`. Fuentes autoritativas, leídas como archivos de la carpeta del cambio: `proposal.md` y `tasks.md` completos; `verify-report.md` completo; el delta `specs/tui/spec.md` completo; la sección final de `apply-progress.md` (UT 6 y notas para la fase 7); los encabezados de `design.md` y `exploration.md`. La spec principal se leyó completa antes de fusionar.
- Observaciones de Engram localizadas con `mem_search` (solo vista previa; no se usaron como fuente): #67 `explore`, #68 `decisions`, #69 `proposal`, #70 `spec`, #71 `design`, #72 `tasks`, #73 `apply-progress` y #74 `verify-report`.
- Formato de referencia: `openspec/changes/archive/2026-10-31-tui-08-registrar-esfuerzo/archive-report.md` (commit de archivado `939dc30`).

## Contradicciones y decisiones de proceso

- `(Previously: …)`: se conservan en la spec principal, como en tui-13. Decisión registrada aquí; no se eliminaron.
- La composición mezcló LF y CRLF y dejó una línea en blanco final sobrante. Ambos se normalizaron antes del `mv`, como en tui-08.
- `tasks.md`, `proposal.md` y los demás artefactos no se modificaron: no se marcó ninguna casilla ni se reescribió ningún informe.
- El commit de archivado añade solo rutas explícitas. `openspec/config.yaml` tiene un cambio ajeno al cambio y no se incluye.

## Observaciones para el orquestador

1. Abrir el PR hacia `main` con `size:exception` (estrategia `single-pr`), desde la rama, que ya incluye el commit de archivado.
2. Decidir sobre W1 y W2: cubrirlos con pruebas pequeñas o aceptarlos tal como están.
3. Ejecutar la tarea 7.4 cuando haya Docker y PostgreSQL disponibles.
4. Confirmar S1 con el usuario.
5. Tratar el doble envío de Planning Poker en un cambio aparte.
6. La spec principal ya contiene el delta. No volver a fusionarlo después del merge del PR.

## Cierre

El cambio `tui-refactor-acciones-desde-historia` queda archivado el 2026-11-04 en `openspec/changes/archive/2026-11-04-tui-refactor-acciones-desde-historia/`. La spec `tui` consolida el comportamiento vigente (38 requisitos). La implementación está en la rama en 6 commits. La verificación intermedia dio 0 CRITICAL y 5 WARNING, sin corregir. Quedan sin ejecutar las pruebas de integración y el recorrido manual (7.4).
