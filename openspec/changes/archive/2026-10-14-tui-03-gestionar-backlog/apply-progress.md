# Avance de implementación: gestionar el backlog desde la TUI (TUI-03)

**Cambio**: `tui-03-gestionar-backlog`
**Modo**: TDD estricto (`openspec/config.yaml`: `strict_tdd: true`; ejecutor de pruebas `go test ./...`, foco `go test ./tests/unit/tui/`)
**Rama**: `feat/tui-03-gestionar-backlog`
**Estado**: 78 de 78 tareas completas (fases 1 a 6, nueve commits).

## Decisión de entrega

- Estrategia: `single-pr`.
- `size:exception`: el usuario confirmó un único PR con la etiqueta `size:exception` antes de aplicar (pronóstico original de 800 a 950 líneas modificadas, presupuesto de 400). La confirmación quedó asentada en este documento antes del commit 2, como exigía `tasks.md`.
- Resultado real: 3262 líneas modificadas hasta el commit 8 (ver «Línea final de entrega»). El pronóstico se superó de forma material y se reporta sin recortar pruebas, comentarios ni documentación.

## Commits

| # | Hash | Mensaje | Tipo |
|---|------|---------|------|
| 1 | `518aadb` | `docs(openspec): specify TUI backlog management` | documentación |
| 2 | `0168323` | `refactor(tui): share JSON request helper between API clients` | refactor |
| 3 | `352965c` | `test(tui): cover scoped quit, space and rune-aware backspace in forms` | RED |
| 4 | `c1cd178` | `fix(tui): scope q to non-text screens and accept spaces in fields` | GREEN |
| 5 | `0b5a329` | `test(tui): cover backlog API client` | RED |
| 6 | `af0ccf9` | `feat(tui): add backlog API client` | GREEN |
| 7 | `a0ae950` | `test(tui): cover backlog screens and error rendering` | RED |
| 8 | `ceb4ba8` | `feat(tui): manage the product backlog from the terminal UI` | GREEN |
| 9 | (este commit) | `docs(openspec): record TUI-03 apply progress` | documentación |

## Evidencia TDD

### Commit 2: refactor (línea base y resultado)

- Antes (HEAD `518aadb`): `go test ./tests/unit/tui/` -> `ok` (17 pruebas de nivel superior en verde, sin pruebas nuevas; la protección es la suite existente, `api_test.go` incluida).
- Después: `gofmt` sobre `api.go` y `projects.go` sin diferencias, `go vet ./internal/tui/...` limpio, `go test -count=1 ./tests/unit/tui/` -> `ok` con las mismas 17 pruebas.

### Commit 3: RED (pruebas de teclado de proyectos)

Comando: `go test -count=1 ./tests/unit/tui/` sobre `352965c` (solo `model_test.go` cambia; no hay cambios de producción, así que compila y falla por aserciones).

Resultado: `FAIL` (`FAIL github.com/.../tests/unit/tui 2.093s`). Fallos por aserciones:

```
--- FAIL: TestQuitOnQDoesNotExitInProjectTextFields (project-id y create-project, q y Q)
    model_test.go:270: "q" quit the TUI from project-id, want it typed into the field
--- FAIL: TestSpaceIsAcceptedInProjectTextFields
    model_test.go:340: field "ID" = "ab", want "a b"
    model_test.go:340: field "Nombre" = "ab", want "a b"
--- FAIL: TestSpaceInProjectNameIsSentToAPI
    model_test.go:358: name = "ProyectoAlfa", want "Proyecto Alfa"
--- FAIL: TestBackspaceRemovesWholeRuneInProjectFields (accented-final-rune y multibyte-until-empty, ambos campos)
    model_test.go:381: backspace 1 left invalid UTF-8: "Canci\xc3"
    model_test.go:381: backspace 1 left invalid UTF-8: "añ\xc3"
--- FAIL: TestBackspaceThenTypeSendsValidUTF8Name
    model_test.go:438: name = "Gestióna�o", want "Gestión ao"
```

Pruebas que ya pasaban (protección de regresión, como anticipaba `tasks.md`): `TestCtrlCQuitsFromProjectForms` (3.2), `TestQuitOnQFromMenuAndProjectLists` (3.3), `TestBackspaceOnEmptyFieldIsNoop` (3.7) y el subcaso `accented-last-rune` de 3.6.

Hallazgo de triangulación (3.6): el escenario literal «Canción -> Canció» pasa también con borrado por bytes porque la última runa («n») es ASCII. Se conservó el subcaso literal de la spec y se agregaron `accented-final-rune` («Canció» -> «Canci») y `multibyte-until-empty` («añá»), que sí fallan hoy.

### Commit 4: GREEN

Comando: `go test -count=1 ./tests/unit/tui/` -> `ok` (25 pruebas de nivel superior en verde: las 17 previas, incluidas `TestProjectNavigationPreservesOtherSectionsAndQuit` y `TestQuitOnQAndCtrlC` sin modificar, más las 8 nuevas). `go vet ./internal/tui/...` limpio. Suite sin Docker (17 paquetes) en verde.

### Commit 5: RED (cliente HTTP del backlog)

Comando: `go test -count=1 ./tests/unit/tui/ -run TestBacklogClient` sobre `0b5a329` (esqueleto de `backlog.go` con firmas que devuelven valores cero; compila).

Resultado: `FAIL`, las 8 pruebas de nivel superior fallan por aserciones:

```
--- FAIL: TestBacklogClientListUsesGetOnProjectStories      backlog_api_test.go:105: requests = 0 ([]), want exactly 1
--- FAIL: TestBacklogClientCreateSendsExactlyFourKeys       backlog_api_test.go:121: requests = 0 ([]), want exactly 1
--- FAIL: TestBacklogClientCreateSendsEmptyCriteriaAsList   backlog_api_test.go:144: requests = 0 ([]), want exactly 1
--- FAIL: TestBacklogClientUpdateSendsSixKeysWithNullHours  backlog_api_test.go:156: requests = 0 ([]), want exactly 1
--- FAIL: TestBacklogClientUpdateSendsHoursNumber (8, 2.5, -3)
--- FAIL: TestBacklogClientEscapesPathSegments (a%2Fb, a%20b, a%3Fb)    requests = 0, want 3
--- FAIL: TestBacklogClientPreservesErrorBodies (422, 404 JSON, 405 texto)   response = 0 "", want 422 "{...}"
--- FAIL: TestBacklogClientTransportErrorWhenServerClosed   List/Create/Update error = nil, want transport error
```

### Commit 6: GREEN

Comando: `go test -count=1 ./tests/unit/tui/ -run TestBacklogClient -v` -> las 8 pruebas `PASS` (incluidos subcasos); `go test -count=1 ./tests/unit/tui/` completo -> `ok`. `go vet ./internal/tui/...` limpio. Refactor 6.4: no se halló duplicación con `ProjectClient` que justifique cambiar la API pública (ambos delegan en `apiClient.do`).

### Commit 7: RED (pantallas del backlog)

Comando: `go test -count=1 ./tests/unit/tui/ -run 'TestBacklog|TestQuitOnQIsScoped|TestCtrlCQuitsFromEveryBacklogScreen|TestSpaceAndBackspace' -v` sobre `a0ae950` (sin esqueleto adicional: compila con la API ya existente).

Resultado: `FAIL`. Las 29 pruebas nuevas de nivel superior fallan; las 8 `TestBacklogClient*` del commit 6 siguen en `PASS` (coinciden con el filtro `TestBacklog`). Ejemplos de salida:

```
--- FAIL: TestBacklogMenuOpensBacklogScreen
    View() = "...Backlog   Disponible en TUI-03 ..." want the TUI-03 placeholder gone
--- FAIL: TestBacklogWithoutProjectOffersChooseProjectAndEmitsNoRequest
    View() = "...Disponible en TUI-03..." want "Elegir proyecto" / want "Volver"
--- FAIL: TestBacklogProjectIDEmptyStaysAndEmitsNoRequest
    Screen = "backlog", want backlog-project
--- FAIL: TestBacklogProjectSelectionLoadsBacklogInAPIOrder
    View() = "...TUI-03..." want the loading message
```

Salvedad honesta: las 29 fallan por aserciones (`t.Fatal`/`t.Errorf` sobre vista, pantalla o comando), no por compilación ni pánico. Pero la mayoría de las pruebas de flujo fallan en el primer paso compartido (`openBacklogWithProject` exige que confirmar el identificador devuelva un comando `List`, y el marcador de posición no lo hace), por lo que no discriminan todavía cada requisito por separado. Cada una queda discriminada al pasar a verde.

### Commit 8: GREEN

Comandos y resultados:

- `go test -count=1 ./tests/unit/tui/ -v` -> `ok`: 62 pruebas de nivel superior y 93 subpruebas en `PASS`, 0 `FAIL`, 0 `SKIP`.
- `go vet ./...` y `go build ./...` limpios (también compila `cmd/tui` y `cmd/api`).
- Suite sin Docker (17 paquetes unitarios): todos `ok`. `go test -count=1 ./...` completo: 18 paquetes `ok` (incluye `tests/integration/migrations`) y 5 paquetes `FAIL` solo de integración con PostgreSQL (ver «Pruebas que dependen de Docker»).
- Las pruebas preexistentes (`TestProjectNavigationPreservesOtherSectionsAndQuit`, `TestQuitOnQAndCtrlC` y la de «Abajo + Enter -> `backlog`») pasan sin modificar.

### Tabla de ciclo TDD

| Tareas | Archivo de prueba | Capa | Red de seguridad | RED | GREEN | Triangulación | Refactor |
|--------|-------------------|------|------------------|-----|-------|---------------|----------|
| 2.1 a 2.4 | `tests/unit/tui/api_test.go` (existente) | Unitaria (aprobación) | 17/17 antes | N/A (refactor sin cambio de comportamiento) | 17/17 después | N/A | `apiClient` extraído |
| 3.1 a 3.9 / 4.1 a 4.6 | `tests/unit/tui/model_test.go` | Unitaria (caja negra) | 17/17 | 8 pruebas escritas; fallan 5 (3.1, 3.4, 3.5, 3.6, 3.8) y 3 ya pasaban (3.2, 3.3, 3.7) | 25/25 | 2 teclas (`q`, `Q`), 2 pantallas, 3 casos de borrado | `editText`/`quitsOnQ` sin duplicación restante |
| 5.1 a 5.10 / 6.1 a 6.4 | `tests/unit/tui/backlog_api_test.go` | Unitaria (`httptest`) | N/A (archivo nuevo) | 8 pruebas, todas fallan | 8/8 y suite 33/33 | 3 valores de horas, 3 identificadores, 3 cuerpos de error | Sin duplicación que justifique cambio |
| 7.1 a 7.31 / 8.1 a 8.9 | `tests/unit/tui/backlog_test.go` | Unitaria (caja negra + `httptest`) | 33/33 | 29 pruebas, todas fallan | 62/62 | tablas por escenario (listas vacías, valores de horas, criterios, errores 422/404/texto) | Nombres y bloques revisados; `gofmt` y `go vet` limpios |

Resumen: 45 pruebas nuevas de nivel superior (8 + 8 + 29), 93 subpruebas en total en el paquete. Funciones puras nuevas: `splitCriteria`, `parseEstimatedHours`, `describeBacklogFailure`, `describeStory`, `storyRow`, `quitsOnQ`, `editText`.

## Tabla de evidencia por unidad de trabajo

| Unidad | Comando focalizado y resultado | Arnés de ejecución | Límite de reversión |
|--------|--------------------------------|--------------------|---------------------|
| 1 (refactor) | `go test ./tests/unit/tui/` -> ok antes y después | N/A: refactor interno cubierto por `httptest` | commit 2 (`api.go`, `projects.go`) |
| 2 (teclado) | `go test ./tests/unit/tui/ -run 'TestQuitOnQ|TestCtrlC|TestSpace|TestBackspace'` -> ok tras el commit 4 | N/A: caja negra con `Model.Update` y `httptest` | commits 3 y 4 |
| 3 (cliente) | `go test ./tests/unit/tui/ -run TestBacklogClient` -> 8/8 ok tras el commit 6 | N/A: `httptest` registra método, ruta y cuerpo | commits 5 y 6 |
| 4 (pantallas) | `go test ./tests/unit/tui/` -> 62 pruebas ok tras el commit 8 | Los `tea.Cmd` se ejecutan contra `httptest`; la ejecución manual con API real no se hizo (ver «Entrada para ver la TUI») | commits 7 y 8 |
| 5 (registro) | `go test` sin Docker + `go vet ./...` + `go build ./...` | N/A: solo documentación | commit 9 |

## Pruebas que dependen de Docker (informe honesto)

`go test ./...` completo no termina en verde en esta máquina por una causa ajena al cambio: los 5 paquetes `tests/integration/{project,projectmember,sprint,story,task}/postgres` arrancan PostgreSQL con Testcontainers y fallan con `rootless Docker is not supported on Windows, failed to create Docker provider`. El mismo conjunto de 5 paquetes falló idénticamente sobre la rama base antes de tocar código (verificado al inicio y en los commits 4 y 8). No hay cambios de backend en este cambio, por lo que no se esperan regresiones allí, pero no se ejecutaron. Los 17 paquetes unitarios más `tests/integration/migrations` pasan.

## Desviaciones y observaciones

- **Esqueleto del commit 5**: `backlog.go` se creó con los tipos completos (`Story`, `StoryCreateInput`, `StoryUpdateInput`, `BacklogOp`, `BacklogResultMsg`) y tres firmas de cliente que devuelven valores cero. La tarea 6.2 («completar tipos») no tuvo trabajo adicional en el commit 6 porque los tipos ya estaban completos desde el esqueleto.
- **Retroceso por runa en 3.6**: se agregó un subcaso extra de triangulación (ver commit 3).
- **Cursor de `backlog`**: al salir de `backlog-project` con `Esc`, el cursor vuelve a 0 (el diseño no lo precisaba). Con `Esc` desde el formulario se conserva la posición.
- **Mensajes de éxito**: «Historia creada correctamente (201)» / «Historia modificada correctamente (200)» seguido de la fila de la historia informada por la API (o el cuerpo recortado si no es una historia).
- **Verificación de formato**: el árbol de trabajo en Windows tiene finales de línea CRLF (`core.autocrlf=true`), por lo que `gofmt -l .` lista casi todos los archivos del repositorio. Se verificó el contenido almacenado en el índice (LF) con `git show HEAD:<archivo> | gofmt -l`: los archivos de este cambio no tienen diferencias. Solo dos archivos no tocados (`tests/unit/project/domain/project_test.go`, `tests/unit/story/transport/http/handler_test.go`) están sin formatear en el repositorio previo a este cambio.
- **Ejecución con `-race`**: no disponible en este entorno de Windows sin cgo; no se ejecutó.
- Resto del diseño: sin desviaciones.

## Riesgos conocidos

- Respuesta obsoleta de `List` durante un cambio de proyecto (aceptado en el diseño).
- Los criterios de aceptación no pueden contener `;` (limitación documentada en el formulario).
- Los 5 paquetes de integración PostgreSQL no se ejecutaron en este equipo (Docker); se recomienda confirmar el CI.
- Tamaño del PR: ver siguiente sección.

## Línea final de entrega

- Estrategia `single-pr` con etiqueta `size:exception` (acordada con el usuario).
- Diff acumulado de los commits 1 a 8 frente a `main` (`git diff --numstat main..HEAD`): 3194 líneas añadidas y 68 eliminadas, 3262 en total.
  - Producción (`internal/tui/`): 648 añadidas y 68 eliminadas (716).
  - Pruebas (`tests/unit/tui/`): 1643 añadidas (`backlog_test.go` 1067, `backlog_api_test.go` 280, `model_test.go` 296).
  - Especificación (`openspec/changes/tui-03-gestionar-backlog/`): 903 añadidas.
- Presupuesto de revisión: 400 líneas; pronóstico original 800 a 950. El resultado real excede ambos. Reportado sin recortes artificiales; la causa principal es la cobertura de pruebas de caja negra por escenario de la spec (50 % del total son pruebas).
- Mitigación de revisión: nueve commits temáticos con RED y GREEN separados; transporte (`backlog.go`) aislado de presentación (`backlog_screens.go`).

## Entrada para ver la TUI

- Binario: `go run ./cmd/tui` (o `go build ./cmd/tui`).
- Variable de entorno de la TUI: `API_URL` (por defecto `http://localhost:8080`; se recortan espacios y barra final).
- Levantar la API: `go run ./cmd/api` con `DATABASE_URL` obligatoria (cadena de conexión PostgreSQL) y `HTTP_ADDR` opcional (por defecto `:8080`). La base necesita las migraciones aplicadas externamente en `internal/project/infrastructure/postgres/migrations/`; las rutas de historias (lista y modificación) se registran según `schema_migrations` (la lista de backlog requiere versión 4 limpia o superior).

## Tarea 7 — estilo visual del backlog

Refinamiento visual pedido por el usuario después de probar la TUI manualmente. No cambia la especificación (`openspec/specs/` intacto): el comportamiento, las teclas y los textos son los mismos, y las 62 pruebas existentes de `tests/unit/tui` pasan sin modificarse.

| Commit | Hash | Contenido |
|--------|------|-----------|
| 10 RED | `fcf9e0e` | `test(tui): cover backlog screen styling (RED)`: `backlog_style_test.go` (8 pruebas) y `termenv` como dependencia directa en `go.mod` |
| 11 GREEN | `281018e` | `feat(tui): style backlog screens like the main menu (GREEN)`: `internal/tui/backlog_screens.go` |
| 12 docs | este commit | `docs(openspec): record backlog styling task` |

**RED** (`go test ./tests/unit/tui/`): las 8 pruebas nuevas fallan, todas con mensajes del tipo `line "▸ Crear historia" has no ANSI styling, want it styled`, `selected row ... has no background`, `footer ... is one unstyled run, want the keys styled apart from the text` y `error line ... has no banner border`. Las aserciones del texto plano sin ANSI ya pasaban (el texto no cambia) y las 62 pruebas previas seguían verdes.

**GREEN**: `go test ./tests/unit/tui/` en verde con 70 pruebas de nivel superior (62 + 8), sin ningún FAIL; `go test $(go list ./... | tr -d '\r' | rg -v tests/integration)` en verde; `go build ./...` y `go vet ./internal/tui/ ./tests/unit/tui/` limpios.

Decisiones y observaciones:

- La prueba fuerza `termenv.TrueColor` porque `lipgloss` no emite ANSI sin TTY; así las pruebas existentes no cambian. Una prueba adicional confirma que con perfil `Ascii` no hay secuencias ANSI.
- Las filas del backlog usan `selectedStyle` e `itemStyle` con `UnsetPadding()`: el relleno horizontal del menú agregaría espacios al texto plano y rompería el contrato de que el texto sin ANSI no cambia (por ejemplo `fieldValue` en las pruebas de formulario). Visualmente la barra de selección no tiene el margen lateral que sí tiene el menú.
- `errorBannerStyle` dibuja un borde redondeado, de modo que el texto del error queda dentro de un recuadro (el contenido de cada línea no cambia).
- La pantalla de resultado distingue éxito de error con `backlogReturn` (`"backlog"` solo en éxito), sin agregar campos al modelo. En éxito se resalta solo la primera línea (`Historia creada correctamente (201)`).
- No se tocaron las pantallas de proyectos ni el menú.
