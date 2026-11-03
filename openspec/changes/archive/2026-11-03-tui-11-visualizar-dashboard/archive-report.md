# Reporte de archivado — TUI-11: visualizar el Dashboard desde la TUI (issue #104)

**Fecha de archivado**: 2026-11-03  
**Estado**: archivado. Implementación completa en 3 commits de código/pruebas más el commit de documentación. Verificación PASS (0 CRITICAL, 0 WARNING, 2 SUGGESTION).  
**Rama**: `feat/tui-11-dashboard`, creada desde `origin/main` (`7f3604d`).  
**PR**: se abre después de este commit, único y con `size:exception` documentada; no se fusiona en esta fase.

## Resumen ejecutivo

TUI-11 reemplaza el aviso de TUI-06 por una pantalla Dashboard real para el proyecto activo. Se abre directamente desde el menú (índice 7), pide el proyecto con el navegador existente cuando la sesión no tiene uno y combina dos lecturas ya existentes: `GET /projects/{project_id}` para el estado y `GET /projects/{project_id}/metrics` para el sobre de US-19. No se agregan endpoints ni cálculos: los valores se muestran tal como llegan y `null` se presenta como «sin datos», nunca como cero.

La pantalla incluye barras Unicode (`█` lleno, `░` vacío) para Story Points, horas, porcentaje de historias completadas y defectos; velocidad y desviación quedan como valores. Cada fila conserva etiqueta y números, de modo que la pantalla sigue siendo legible sin color. El ancho de barra se calcula desde `innerWidth` y las pruebas cubren 80 y 44 columnas. `r` refresca, `p` cambia el proyecto, `Esc` vuelve al menú y los errores de la API se muestran dentro de la pantalla sin cerrarla. Una secuencia (`seq`) descarta respuestas tardías de un proyecto o reintento anterior.

## Specs consolidadas

| Dominio | Antes | Después | Cambios |
|---|---|---|---|
| `tui` (`openspec/specs/tui/spec.md`) | 33 requisitos | 37 requisitos | 4 ADDED al final: «Abrir el Dashboard desde el menú», «Presentar los valores de las métricas sin recalcularlos», «Graficar las métricas con barras Unicode» y «Mantener el Dashboard de solo lectura y tolerante a errores». 1 MODIFIED en su posición: «Avisar el límite de las pantallas de navegación», que retira Dashboard y conserva los escenarios de avisos y de Esfuerzo. |

**Método**: `openspec archive tui-11-visualizar-dashboard --yes --skip-specs` movió la carpeta a `openspec/changes/archive/2026-11-03-tui-11-visualizar-dashboard`. La consolidación se hizo a mano por los encabezados en español, igual que en TUI-08 y US-18. Comprobaciones:
- `grep -c '^### Requirement' openspec/specs/tui/spec.md` → `37`.
- Los 4 requisitos ADDED del delta aparecen al final de la spec; el requisito MODIFIED reemplazó el bloque anterior sin perder los escenarios «Aviso al completar una selección» ni «Esfuerzo ya no muestra el aviso».
- Los 4 escenarios nuevos del Dashboard están presentes: abrir con y sin proyecto activo, filas de métricas, métrica no calculable, consistencia con Métricas, valores desde la API, barras sin color, terminal angosta, API no disponible, actualización y recorrido de solo lectura.

## Commits de la rama

| Commit | Mensaje | Unidad de trabajo |
|---|---|---|
| `dcf7c3e` | `feat(tui): abrir el Dashboard del proyecto activo` | UT 1: pantalla, menú, picker, miga, footer y ayuda |
| `09dcfaf` | `feat(tui): graficar las metricas del Dashboard` | UT 2: barras Unicode, «sin datos» y ancho adaptable |
| `1ec3ff9` | `test(tui): cubrir errores y solo lectura del Dashboard` | UT 3: errores, reintento, `Esc`, cambio de proyecto y GET-only |
| (commit de este reporte) | `docs(tui): especificar y archivar el Dashboard (TUI-11)` | UT 4: README, delta, verificación y archivado |

## Verificación

Fuente: `verify-report.md` de esta misma carpeta, commit `1ec3ff9`.

- `go build ./...`: exit 0.
- `go vet ./...`: exit 0.
- `go test ./...`: exit 0, 37 paquetes `ok`, 0 `FAIL`, con Docker y Testcontainers.
- `openspec validate tui-11-visualizar-dashboard --strict`: válido antes de archivar.
- `gofmt -l internal tests cmd`: solo `internal/tui/metrics.go`, preexistente en `main` y no modificado por este cambio.
- La primera corrida completa falló en `tests/integration/story/postgres` por `disk quota exceeded` en `/tmp`; el paquete pasó al reejecutarse solo y la corrida completa posterior terminó en verde. No fue un fallo de código.

**Criterios del issue #104**: 8 de 8 verificados con pruebas nombradas en el informe. El recorrido manual con API y PostgreSQL en vivo no se ejecutó; la verificación es de modelo con `httptest` y de suites automatizadas.

## Tamaño y entrega

- Pronóstico de `tasks.md`: ~850–950 líneas. Riesgo de presupuesto de 400 líneas: `High`; estrategia `single-pr` con `size:exception`, igual que TUI-08.
- Medición real contra `origin/main`, antes de este reporte: código, pruebas, README y spec consolidada +827 / −33 (860 líneas). Artefactos del cambio: ~464 líneas de Markdown, más este reporte.
- El PR único documenta la excepción de tamaño y referencia `Closes #104`.

## Artefactos preservados

Carpeta `openspec/changes/archive/2026-11-03-tui-11-visualizar-dashboard/`:
- `proposal.md`, `exploration.md`, `design.md`, `tasks.md`, `apply-progress.md`, `verify-report.md`, `specs/tui/spec.md` y `archive-report.md`.
- `tasks.md` se marca completo al archivar; `apply-progress.md` conserva la evidencia RED/GREEN/TRIANGULATE/REFACTOR de las cuatro unidades.
EOF