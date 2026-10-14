# Reporte de archivo — TUI-03 gestionar backlog desde la TUI

**Fecha de archivo**: 2026-10-14
**Estado**: implementación completa (78/78 tareas de la implementación original más la tarea 7 de estilo visual). Verificación manual realizada por el usuario; `sdd-verify` no se ejecutó.
**Implementación**: rama `feat/tui-03-gestionar-backlog`, 12 commits de implementación más el commit de archivo. PR pendiente al momento de escribir este reporte.

**Escrito para**: revisores del equipo y futuros archivos del proyecto.

## Resumen

TUI-03 reemplaza el placeholder de backlog por una pantalla funcional que consume la API de historias existente sin lógica de negocio: elegir el proyecto una vez por sesión, listar el Product Backlog, crear historias y modificarlas eligiéndolas de la lista con el formulario precargado. También corrige tres defectos heredados de TUI-02 que impedían usar formularios de texto: la `q` cerraba la TUI dentro de un campo, la barra espaciadora se descartaba y el retroceso borraba bytes en vez de caracteres (corrompía acentos).

## Entrega

Un único PR con `size:exception`, decidido por el usuario antes de aplicar. El pronóstico era de 800–950 líneas y el diff real contra `main` es considerablemente mayor:

| Parte | Líneas cambiadas |
|---|---|
| Código de producción y tests (`internal/`, `tests/`) | 2 639 (2 571 agregadas, 68 eliminadas) |
| Documentos SDD (`openspec/`, sin contar el archivo) | 1 121 |
| **Total del diff de los 12 commits de implementación** | **3 762** (3 693 agregadas, 69 eliminadas) |

El exceso se debe sobre todo a los tests (`backlog_test.go` es el archivo más grande) y a los documentos de especificación. No se recortaron tests ni documentación para ajustarse al presupuesto de 400 líneas. Se recomienda avisar al revisor.

## Commits de implementación

| # | Hash | Mensaje |
|---|---|---|
| 1 | `518aadb` | docs(openspec): specify TUI backlog management |
| 2 | `0168323` | refactor(tui): share JSON request helper between API clients |
| 3 | `352965c` | test(tui): cover scoped quit, space and rune-aware backspace in forms |
| 4 | `c1cd178` | fix(tui): scope q to non-text screens and accept spaces in fields |
| 5 | `0b5a329` | test(tui): cover backlog API client |
| 6 | `af0ccf9` | feat(tui): add backlog API client |
| 7 | `a0ae950` | test(tui): cover backlog screens and error rendering |
| 8 | `ceb4ba8` | feat(tui): manage the product backlog from the terminal UI |
| 9 | `92cf3f6` | docs(openspec): record TUI-03 apply progress |
| 10 | `fcf9e0e` | test(tui): cover backlog screen styling (RED) |
| 11 | `281018e` | feat(tui): style backlog screens like the main menu (GREEN) |
| 12 | `657c4ff` | docs(openspec): record backlog styling task |

Los commits 3, 5, 7 y 10 contienen solo tests que fallaron antes de su implementación (RED observado). Tres casos del commit 3 ya pasaban con el código previo y se registraron como guardas de regresión, no como RED.

## Tareas

78/78 de la implementación original marcadas en `tasks.md`, más la tarea 7 (estilo visual del backlog). La tarea 7 fue un refinamiento pedido por el usuario después de probar la TUI a mano; no cambia la especificación porque el comportamiento y los textos no variaron.

## Specs fusionadas

Sobre la spec principal de TUI (`openspec/specs/tui/spec.md`):

- **ADDED**: operar el backlog, identificador de proyecto único por sesión, listar el backlog, crear historias, modificar historias desde la lista, análisis numérico mínimo de `estimated_hours`, presentación de errores de la API del backlog, campos de texto que aceptan espacios y campos de texto que borran por carácter.
- **MODIFIED**: "Mostrar menú base de navegación" (solo sprints queda sin pantallas funcionales), "Salir limpiamente de la TUI" (Ctrl+C sale desde cualquier pantalla; `q` solo desde menú, listas y resultados) y "Navegación de solo lectura" (admite POST y PUT de historias).

La spec principal no se editó durante las fases de especificación y diseño; solo se tocó en este archivo, siguiendo la corrección de proceso registrada en TUI-02.

## Artefactos SDD

| Artefacto | Estado |
|---|---|
| `proposal.md` | Presente |
| `design.md` | Presente (11 decisiones) |
| `specs/tui/spec.md` (delta) | Presente |
| `tasks.md` | Presente |
| `apply-progress.md` | Presente, persistido durante la implementación con la evidencia RED/GREEN |
| `exploration.md` | **Faltante como archivo.** La exploración se guardó solo en Engram (`sdd/tui-03-gestionar-backlog/explore`). |
| `verify-report.md` | **Faltante.** No se ejecutó `sdd-verify`. |
| `archive-report.md` | Este documento. |

## Verificación

**Automática** (Docker disponible, ejecutada el 2026-10-14):

- `go build ./...` y `go vet` sobre `internal/tui` y `tests/unit/tui`: sin hallazgos.
- `tests/unit/tui`: 70 tests de nivel superior pasando (62 de la implementación original más 8 de estilo).
- Paquetes unitarios restantes (sin integración): ok.
- Integración: `go test -count=1 -p 1 ./tests/integration/...` pasa en los 6 paquetes (`migrations`, `project`, `projectmember`, `sprint`, `story`, `task`). Una corrida previa en paralelo falló en 5 de 6 paquetes por el arranque de contenedores de Testcontainers (`wait for reaper ... unexpected container status "created"` y `failed to connect`), mientras otra tarea usaba el mismo Docker. El cambio no modifica código de backend: fuera de `openspec/`, `internal/tui/` y `tests/unit/tui/`, solo cambia una línea de `go.mod` (`termenv` pasa de dependencia indirecta a directa).

**Manual**, realizada por el usuario contra la API real y PostgreSQL en Docker (12 puntos, todos confirmados):

1. Título con espacios y `q` dentro de un formulario: no cierra la TUI.
2. Retroceso con acentos: "Canción" queda "Canció".
3. `q` sale desde el menú y las listas, no desde formularios; Ctrl+C cierra desde cualquier pantalla.
4. Crear con criterios `criterio uno; criterio dos`: se guardan como dos criterios.
5. Modificar solo el título conserva descripción, prioridad, estado, criterios y horas; modificar con horas `5.5` las persiste y no cambia nada más.
6. Cambiar a un proyecto sin historias muestra "El backlog no tiene historias.".
7. Crear con campos vacíos muestra el 422 con una línea por campo.
8. Prioridad inválida: 422 con una sola línea.
9. Horas no numéricas: bloqueado localmente, sin request.
10. Proyecto inexistente: 404 `project_not_found`, y vuelve a pedir el ID con el valor ingresado.
11. ID con formato inválido: 422 `project_id: must be a valid UUID`.
12. Regresión en Proyectos: crear con nombre con espacios funciona como antes.

## Limitaciones conocidas

- Un criterio de aceptación no puede contener `;`, porque ese es el separador de la línea de criterios.
- Si el usuario cambia de proyecto mientras una carga anterior sigue en curso, esa respuesta podría aplicarse a la nueva carga. Se aceptó en el diseño para no agregar el ID de proyecto al mensaje.
- Los 404 y 405 sin cuerpo JSON (migraciones sin aplicar) se muestran como error de la API y la TUI sigue abierta.
- No hay un endpoint que liste proyectos, por lo que el ID del proyecto se ingresa a mano.
- No se ejecutó `sdd-verify` ni existe `exploration.md` como archivo.
- Durante la prueba manual una historia apareció con el título "quiero ver mi". No se atribuyó a la TUI porque no se pudo reproducir; podría ser cómo se tipeó.
