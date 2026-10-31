# Reporte de archivado — TUI-08: registrar esfuerzo desde la TUI (issue #101)

**Fecha de archivado**: 2026-10-31  
**Estado**: archivado. Implementación completa en 7 commits de unidades de trabajo. Verificación con 2 advertencias y 6 sugerencias abiertas. El recorrido manual (tarea 7.3) no se ejecutó. La tarea 7.1 queda parcial.  
**Rama**: `feat/tui-08-registrar-esfuerzo` (7 commits sobre `main`). El commit de archivado es local.  
**PR**: no abierto. La entrega es un PR único con `size:exception`, aceptado por el usuario. No se empujó nada al remoto (instrucción del usuario).

## Resumen ejecutivo

TUI-08 completa el flujo de Esfuerzo en la TUI: historia → tarea o «Toda la historia (sin tarea)» → integrante → formulario → `POST /projects/{pid}/stories/{sid}/effort-records` → resultado. El resultado muestra el registro devuelto por la API y las métricas del proyecto (`actual_hours`, `deviation`) leídas de `GET /projects/{pid}/metrics`. La TUI no calcula horas ni desvíos, y desaparece el aviso «estará disponible en TUI-08».

Se fusionan en las specs principales el delta de `tui` (6 requisitos añadidos y 3 modificados) y el de `esfuerzo` (1 requisito añadido, más el Propósito editado a mano). La verificación de `sdd-verify` dio 0 CRITICAL, 2 WARNING y 6 SUGGESTION. No se corrigió ninguna advertencia ni sugerencia en código ni en specs; quedan como decisiones del usuario (ver «Hallazgos abiertos»).

## Specs consolidadas

| Dominio | Antes | Después | Cambios |
|---|---|---|---|
| `tui` (`openspec/specs/tui/spec.md`) | 27 requisitos | 33 requisitos | 6 ADDED al final. 3 MODIFIED reemplazados en su posición: «Navegación de solo lectura», «Elegir integrantes, historias, tareas y Sprints con selectores navegables» y «Avisar el límite de las pantallas de navegación». |
| `esfuerzo` (`openspec/specs/esfuerzo/spec.md`) | 17 requisitos | 18 requisitos | 1 ADDED: «Permitir operar el registro desde la TUI sin ampliar el contrato HTTP». Propósito: se reemplazó la última oración (tarea 7.5). El resto del Propósito no cambió. |

**Método**: `gentle-ai sdd-archive-compose` terminó con exit 0 para `tui` y para `esfuerzo`. Escribió en `spec.md.compose-tmp` y después se movió con `mv`. La composición dejó fines de línea mixtos (LF en los bloques nuevos y CRLF en el resto). Se normalizó cada archivo a CRLF, que es la convención de la copia de trabajo (`core.autocrlf=true`). El índice de git sigue en LF y no cambió por esta normalización.

**Verificación de la fusión** (sin copia de origen, así que `diff -r` no aplica; se usaron estas comprobaciones):
- Los 9 bloques de requisito del delta de `tui` (6 ADDED y 3 MODIFIED) y el bloque de `esfuerzo` aparecen byte a byte en la spec principal. Resultado del script de lectura: `tui ok=9 missing=0`, `esfuerzo ok=1 missing=0`.
- No hay encabezados de requisito duplicados (`uniq -d` sin salida en ambas specs).
- `git diff` contra `HEAD`: `tui` +428 / −8 y `esfuerzo` +26 / −1. Las 8 líneas eliminadas de `tui` son el texto anterior de los tres requisitos MODIFIED (3 líneas de cuerpo y 5 líneas de escenario). No se eliminó ninguna línea en blanco ni texto de otro requisito. La línea eliminada de `esfuerzo` es la oración del Propósito.
- Comandos de la tarea 7.5 (salida verbatim, con `--crlf` donde importa):
  - `rg -c "no se opera desde la TUI" openspec/specs/esfuerzo/spec.md` → sin salida, exit 1 (0 coincidencias).
  - `rg -n "El registro puede operarse desde la TUI" openspec/specs/esfuerzo/spec.md` → una coincidencia, línea 5 (el Propósito).
  - `rg -n "Esfuerzo ya no muestra el aviso|Registrar esfuerzo|Recorrer hacia atrás el flujo de esfuerzo" openspec/specs/tui/spec.md` → coincidencias en las líneas 715, 898, 904, 971 y 1045, incluido el encabezado `### Requirement: Recorrer hacia atrás el flujo de esfuerzo y descartar respuestas tardías`.

## Commits de la rama

| Commit | Mensaje | Unidad de trabajo |
|---|---|---|
| `342b618` | refactor(tui): extraer helpers genéricos del campo de fecha | UT 1 |
| `77ac04d` | feat(tui): agregar cliente de registro de esfuerzo | UT 2 |
| `6f1a24f` | feat(tui): elegir tarea o historia completa e integrante en Esfuerzo | UT 3, parte 1 |
| `d83c4a0` | feat(tui): abrir y editar el formulario de esfuerzo sin enviar | UT 3, parte 2 |
| `46a6beb` | feat(tui): registrar esfuerzo con validación local y anti doble envío | UT 4 |
| `acb64f2` | feat(tui): mostrar el registro y las métricas del proyecto tras registrar | UT 5 |
| `6f5a27e` | docs(tui): actualizar specs y README para TUI-08 | UT 6 (specs, README, artefactos) |
| (commit de archivado, local) | docs(sdd): archivar tui-08-registrar-esfuerzo | merge de specs, Propósito, movimiento y este reporte |

No hay commit `fix(tui)` (tarea 7.4). No hay commits de código posteriores a `6f5a27e`. El hash del commit de archivado no figura en este archivo, para evitar una autorreferencia; consta en el historial de la rama y en el resultado de la fase.

## Verificación

**Fuente**: `sdd-verify`, Engram `sdd/tui-08-registrar-esfuerzo/verify-report`, observación #63. Es un registro intermedio, válido en el momento de la verificación. Como no hubo commits de código posteriores, su evidencia de código sigue siendo la del estado final.

Resultados observados por `sdd-verify`:
- `go build ./...` y `go vet ./...`: exit 0. Re-ejecutado en el archivado: `go build ./...` exit 0.
- `go test ./... -p 1 -count=1`: exit 0. 37 paquetes `ok`, ninguno en fallo ni en caché. Docker disponible, así que los paquetes de integración corrieron contra contenedores reales (`defect` 147 s, `effort` 153 s, `metrics` 5,5 s, `migrations` 0,9 s, `project` 18 s, `projectmember` 31 s, `sprint` 106 s, `story` 371 s, `task` 76 s; `internal/tui` 2,5 s; `tests/unit/tui` 2,8 s).
- `go test ./...` en paralelo: exit 0 y 37 `ok`, pero 36 resultados vinieron de caché. La inestabilidad de Testcontainers (`unexpected EOF`) vista una vez durante apply no se reprodujo ni se descartó. La referencia fiable es `-p 1`.
- Pruebas focalizadas (`-count=1`, `Effort|DateField|DateRow|AlignedTextRow|Today|OpenSectionResets`): 77 pruebas de nivel superior y 77 subpruebas en verde.
- Cobertura unitaria: `effort.go` y `effort_form.go` al 100 %; `selectProject` 95,7 %; `chooseSectionItem` 97,7 %; `viewSection` 96 %.
- Mutantes de sondeo: 5 de 6 eliminados. Sobrevive la guarda `metricsStatus == loading` (ver hallazgos).
- Criterios: 9 de 9 de cierre y 11 de 11 de `proposal.md`, cumplidos según las pruebas nombradas en #63.
- Escenarios: 78 (75 de `tui` y 3 de `esfuerzo`) en 10 requisitos, cada grupo con al menos una prueba que lo cubre.
- Discrepancias D1–D16: confirmadas. En cada caso el código sigue a la spec.

**Re-ejecutado en el archivado**:
- `go build ./...`: exit 0.
- `gofmt` sobre el contenido del índice (`git show :<ruta> | gofmt -l`) para todos los `.go` de `internal`, `tests` y `cmd`: dos archivos, ambos ya sin formato en `main`. `internal/tui/metrics.go` no cambió en este cambio. `tests/unit/tui/sections_flows_test.go` sí se tocó, pero ya estaba sin formato en `main`.
- No se volvieron a ejecutar las pruebas. La evidencia de pruebas es la de #63.

**Recorrido manual (tarea 7.3): no ejecutado.** Requiere una TUI interactiva contra API y PostgreSQL en vivo, que no se levantaron en esta fase. Evidencia parcial solo a nivel API: `internal/api/api.go:177` (`readiness.Effort`, versión >= 12) y `:181` (`readiness.Metrics`, versión >= 14), con `TestMigrationReadinessResolvesEffortByVersion` y `TestMetricsReadinessGatesRoutesWithoutChangingExistingReadiness` en verde. No se comprobó: render real en terminal; el aviso de métricas no disponibles contra una API en esquema 12–13; `r` actualizando «Horas reales» de extremo a extremo.

## Estado de tareas en `tasks.md`

- Casillas: 66 en total. 55 marcadas y 11 sin marcar.
- Sin marcar: 7.1 (parcial), 7.3 (no ejecutada) y las 9 casillas de «Criterios de cierre». Estas últimas no se marcaron porque la fase no lo pedía. `sdd-verify` observó los 9 criterios cumplidos (#63).
- Marcadas en este archivado, con nota en la línea: 7.2 (evidencia en `apply-progress.md`, Fase 7), 7.4 (sin commit) y 7.5 (commit local, sin empujar).
- La marca de 7.1 queda sin marcar. Su criterio literal exige `gofmt -l` sin salida, y no se cumple por los dos archivos preexistentes en `main` (ver «Verificación»).

## Hallazgos abiertos (decisiones del usuario, no corregidos)

Advertencias de `sdd-verify`:
1. **Horas con formatos aceptados por `ParseFloat`** (WARNING 1). `parseEffortHours` acepta cualquier valor finito que acepte `strconv.ParseFloat`: `1e2`, `0x1p-2`, `1_0`, `+3`, `.5` y `5.`. Rechaza `0x10`, `Infinity`, `-Inf`, `1e400` y `1,5`. La spec dice «escrito con punto decimal». Es una decisión deliberada (propuesta, supuesto S3): la API valida el rango y el valor enviado siempre es un número JSON válido. No hay escenario que lo fije. Opciones: endurecer la validación con una expresión regular, o aclarar la redacción de la spec.
2. **Pruebas paralelas en caché** (WARNING 2). `go test ./...` en paralelo quedó en gran parte en caché. La inestabilidad de Testcontainers no se reprodujo ni se descartó. La evidencia fiable es `go test ./... -p 1 -count=1`.

Sugerencias de `sdd-verify`:
3. **Guarda `metricsStatus == loading`**. El mutante que la elimina sobrevive. Es equivalente en caja negra y se conserva como defensa en profundidad. El racional escrito en `apply-progress.md` es algo impreciso. No se reescribió.
4. **`design.md` desactualizado** frente al comportamiento final en D1, D5, D6, D7 y D8, y en el texto de la guarda. La spec prevalece. No se modificó.
5. **README**. El párrafo previo a la frase nueva aún dice que las pantallas «terminan con un aviso» y «no crean, modifican ni eliminan datos»; la frase nueva lo matiza para Esfuerzo. La fila de Esfuerzo queda bajo la columna «Flujo futuro» y excede el ancho de su columna. La introducción (línea 4) aún menciona Esfuerzo como navegación. La fila de Planning Poker sigue como `TUI-07`, obsoleta antes de este cambio.
6. **Miga del selector de proyectos abierto desde el resultado**. Muestra «Proyectos › Elegir proyecto» en lugar de «Esfuerzo › …» (rama `default` de `model.go`). No hay requisito ni prueba.
7. **`apply-progress.md` sin Fase 7 y sin hash de la UT 6**. Resuelto en este archivado: la sección «Fase 7» registra el estado final y el hash `6f5a27e`.

Desviaciones de la implementación respecto de `tasks.md`, documentadas en `apply-progress.md` (sin efecto observable en el comportamiento):
- Las guardas de `applyEffortSave` y `applyEffortMetrics` no tienen la condición `Screen == …` pedida en 4.2 y 5.2. En la UT 5 se quitó porque dejaba «Cargando métricas…» si el usuario abría el selector de proyectos durante la consulta.
- `effortLabelWidth = len("Actividad:")`, con dos puntos, en lugar de `len("Actividad")`. Se añadió `TestEffortFormAlignsTheValueColumn`.
- Pruebas añadidas no previstas en `tasks.md`: `TestEffortFormAlignsTheValueColumn`, `TestEffortFormEnterOnEarlierFieldsNeverSendsAValidForm`, las filas `200` y `204` de D5 y subcasos de `EffortSaveResult`.
- Pruebas `[caracterización]` que nacieron en verde, con mutantes manuales como respaldo (listadas en `apply-progress.md`).
- La UT 3 se partió en dos commits, lo que permite la cabecera de `tasks.md`.

## Trabajo no completado

- **7.3 no ejecutada** (recorrido manual con API y PostgreSQL). Opcional según `tasks.md`.
- **7.1 parcial**. `gofmt -l` no está vacío, por las causas descritas en «Verificación».
- **Casillas de «Criterios de cierre»** sin marcar en `tasks.md`, aunque `sdd-verify` observó los 9 criterios cumplidos.
- No hay una corrida única en verde de `go test ./...` en paralelo. La referencia es `-p 1`.

## Tamaño y entrega

- Pronóstico de `tasks.md`: unas 1.270 líneas (adiciones más eliminaciones). Riesgo de presupuesto de 400 líneas: `High`. Estrategia `single-pr` con `size:exception`, aceptada por el usuario.
- Medición real contra `main` (`git diff --numstat main HEAD`, antes del archivado):
  - Código, pruebas y README: +2.941 / −108 (3.049 líneas).
  - Artefactos del cambio (`openspec/changes/tui-08-registrar-esfuerzo/`): +1.817 / −0.
  - Total de la rama: +4.758 / −108.
- La solicitud citaba 2.937 / 107 para código y 1.821 para artefactos. La diferencia es la fila del README (+4 / −1), que la solicitud contó en artefactos. El total de adiciones coincide. La medición da 108 eliminaciones, no 107.
- Commit de archivado: merge de specs (+454 / −9 contando el Propósito), movimiento de la carpeta, ediciones de `tasks.md` y `apply-progress.md`, y este reporte.

## Artefactos preservados

Carpeta `openspec/changes/archive/2026-10-31-tui-08-registrar-esfuerzo/`:
- `proposal.md`, `design.md`, `exploration.md`, `tasks.md`, `apply-progress.md`, `specs/tui/spec.md` y `specs/esfuerzo/spec.md`: presentes. `diff -r` contra el snapshot tomado antes de `git mv` no produjo salida.
- `verify-report.md`: no existe como archivo, porque el almacén es `openspec`. El reporte de verificación vive en Engram (#63).
- `archive-report.md`: este reporte. Es aditivo y no existía en el origen.
- `tasks.md` y `apply-progress.md` se editaron durante el archivado (marcas de 7.x y Fase 7). El snapshot de `diff -r` se tomó después de esas ediciones, y el estado previo queda en el commit `6f5a27e`.

## Evidencia de copia (`diff -r`)

Operación: `git mv openspec/changes/tui-08-registrar-esfuerzo openspec/changes/archive/2026-10-31-tui-08-registrar-esfuerzo`, con snapshot previo `cp -R` y el movimiento confirmado por `git mv` (no se usó la ruta de respaldo con `mv`).

Salida verbatim de `diff -r "$snapshot_root/source" "$destination"`:

```
(sin salida)
DIFF_R_EXIT=0
```

El origen quedó ausente tras el movimiento (`source_absent_after_move: yes`).

## Fuentes y trazabilidad

- **Artefactos leídos** (almacén `openspec`, localizadores de ruta): `proposal.md` completo; `tasks.md` completo; `apply-progress.md` completo; `specs/esfuerzo/spec.md` completo; `design.md` solo sus primeras 60 líneas; `specs/tui/spec.md` por encabezados y verificación de bloques por script, sin relectura completa.
- **Observaciones de Engram leídas completas**: #63 (`verify-report`).
- **Observaciones de Engram localizadas por `mem_search` (solo vista previa, no leídas completas)**: #57 (`explore`), #58 (`proposal`), #59 (`spec`), #60 (`design`), #61 (`tasks`) y #62 (`apply-progress`). Como el almacén es `openspec`, los localizadores de ruta son los autoritativos.
- **Formato de referencia**: los reportes de archivado de US-15 (2026-10-25) y de US-18 (2026-10-30).
- **Jerarquía de fuentes aplicada**: 1) `tasks.md` archivado; 2) hechos finales de la solicitud de archivado (7.4 sin commit, 7.3 no ejecutada, sin empujar, specs fusionadas); 3) `verify-report` (#63) y `apply-progress` como snapshots intermedios.
- **Contradicciones y resolución**:
  - #57 dice que `exploration.md` no se escribió. El archivo existe desde `6f5a27e`, y la evidencia de repositorio es de rango superior. Se conserva en la carpeta.
  - La solicitud pedía empujar el commit de 7.5 y también pedía no empujar. Se siguió la segunda instrucción. La casilla 7.5 tiene nota.
  - Las cifras de tamaño de la solicitud difieren de la medición. Se reporta la medición (ver «Tamaño»).
  - La tabla «Estado de tareas» de `apply-progress.md` marca la Fase 7 como pendiente. Es un estado histórico del lote 5, y la sección «Fase 7» registra el estado final.
  - `gofmt` de `sdd-verify` mencionaba un solo archivo. El escaneo del índice encuentra también `internal/tui/metrics.go`, que no cambió en este cambio. Ambos ya estaban sin formato en `main`.
  - La tarea 6.1 usa `rg` sin `--crlf`. En la copia de trabajo CRLF ese comando no devuelve nada (ver `apply-progress.md`, UT 6).

## Notas de proceso

- Un `404` de métricas con cuerpo JSON se interpretó como carga válida con todos los valores nulos, hasta que se comprobó primero el estado HTTP (UT 5).
- La prueba de solo `GET` de la Fase 3 (3.12) se reescribió en la UT 4, porque su helper `press` descartaba el comando devuelto.
- Un intento de mutante en la UT 5 tocó por error la guarda de `applyEffortSave`. Se detectó porque la prueba seguía en verde; se restauró y se repitió sobre la guarda correcta.

## Observaciones para el orquestador

1. Decidir si la excepción de `gofmt` de la tarea 7.1 es aceptable. Si lo es, la casilla puede marcarse. Los archivos afectados son `internal/tui/metrics.go` y `tests/unit/tui/sections_flows_test.go`, ambos sin formato en `main`.
2. Decidir sobre el WARNING 1 (`parseEffortHours`): endurecer la validación o aclarar la spec.
3. Ejecutar 7.3 cuando haya API y PostgreSQL disponibles. Es opcional.
4. Push y PR quedan pendientes de la decisión del usuario. El commit de archivado es local.

## Cierre

El cambio `tui-08-registrar-esfuerzo` queda archivado el 2026-10-31 en `openspec/changes/archive/2026-10-31-tui-08-registrar-esfuerzo/`. Las specs `tui` y `esfuerzo` consolidan el comportamiento. La implementación está completa en la rama en 7 commits, con verificación de 0 CRITICAL. El recorrido manual (7.3) no se ejecutó. Las dos advertencias quedan como decisiones del usuario.
