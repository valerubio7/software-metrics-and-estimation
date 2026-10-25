# Avance de implementación: navegación del Sprint 3 en la TUI (TUI-06)

**Cambio**: `tui-06-navegacion-sprint-3`
**Modo**: TDD estricto (`openspec/config.yaml`: `strict_tdd: true`; comando `go test ./...`)
**Rama**: `feat/tui-06-navigation-sections`, basada en `origin/main` (`64cbd93`, incluye #98)
**Estado**: implementación completa; suite unitaria y de integración en verde.

## Decisión de entrega

- Estrategia: `single-pr`.
- El usuario aprobó la opción A: incluir en TUI-06 las dos lecturas GET que faltaban
  para que los selectores usen datos reales.
- `size:exception`: aprobada antes de aplicar. El pronóstico de `tasks.md` fue de
  1.200 a 1.600 líneas; el resultado real se reporta abajo sin recortar pruebas.
- Sin migraciones nuevas: las tablas `project_members` (000006) y `tasks` (000009)
  ya existían.

## Commits

| # | Hash | Mensaje | Tipo |
|---|------|---------|------|
| 1 | `3dc4a8f` | `docs(openspec): specify Sprint 3 TUI navigation` | documentación |
| 2 | `5fdec00` | `test(api): cover project member listing` | RED |
| 3 | `78285cc` | `feat(api): list project members` | GREEN |
| 4 | `f6fc85d` | `test(api): cover story task listing` | RED |
| 5 | `8adc53b` | `feat(api): list tasks for a story` | GREEN |
| 6 | `98fadf6` | `test(tui): cover shared selector and section clients` | RED |
| 7 | `3a10d4d` | `feat(tui): add shared selector and section clients` | GREEN |
| 8 | `42da4ad` | `test(tui): cover Sprint 3 menu and section routing` | RED |
| 9 | `230919c` | `feat(tui): navigate Sprint 3 section shells` | GREEN |
| 10 | este commit | `docs(tui): document Sprint 3 navigation` | documentación |

Los flujos de la fase 5 se cubrieron en el mismo ciclo RED/GREEN de la fase 4
(commits 8 y 9); separarlos no agregaba un límite de reversión independiente.

## Evidencia TDD

### Integrantes de un proyecto (API)

**RED** (`5fdec00`) — `go test ./tests/unit/projectmember/application ./tests/unit/projectmember/http`:

```
--- FAIL: TestListMembersRejectsInvalidProjectIDBeforeReading
    err = <nil>, want project_id validation error
--- FAIL: TestListMembersReturnsTheProjectAndItsMembers
    project ids = "" / "", want canonical id
--- FAIL: TestListMembersPreservesAnEmptyCollectionAndPropagatesErrors
    empty listing = []domain.Member(nil), want non-nil empty slice
--- FAIL: TestListMembersHandlerRejectsNonGetMethods
    status = 200, want 405
--- FAIL: TestListMembersHandlerResponses
    invalid_uuid: status = 200, want 422
    project_not_found: status = 200, want 404
    empty_collection: body does not contain "members":[]
```

El esqueleto mínimo de `ListMembersUseCase` y `NewListMembersHandler` se agregó en
el mismo commit RED para que las pruebas fallaran por aserciones, no por compilación.

**GREEN** (`78285cc`) — `go test ./tests/unit/projectmember/application ./tests/unit/projectmember/http ./tests/unit/cmd/api`:

```
ok  .../tests/unit/projectmember/application
ok  .../tests/unit/projectmember/http
ok  .../tests/unit/cmd/api
```

Se agregó `PostgresMemberRepository.ListByProject` (proyecto existente, orden por
`full_name` e `id`, email nullable), el handler `{project_id, members}` con
405/422/404/500 y la composición condicionada por `MemberDependencies.Lister`.

### Tareas de una historia (API)

**RED** (`f6fc85d`) — `go test ./tests/unit/task/application ./tests/unit/task/transport/http`:

```
--- FAIL: TestListTasksRejectsInvalidRouteIDsBeforeReading
    invalid project: err = <nil>, want project_id validation error
    invalid story:   err = <nil>, want story_id validation error
--- FAIL: TestListTasksReturnsCanonicalContextAndTasks
    context = ""/"", want canonical ids
--- FAIL: TestListTasksPreservesAnEmptyCollectionAndPropagatesErrors
    empty listing = []domain.Task(nil), want non-nil empty slice
--- FAIL: TestListTasksHandlerRejectsNonGetMethods
    status/calls = 200/0, want 405/0
--- FAIL: TestListTasksHandlerResponses
    invalid_project/invalid_story: status = 200, want 422
```

**GREEN** (`8adc53b`) — `go test ./tests/unit/task/... ./tests/unit/cmd/api`:

```
ok  .../tests/unit/task/application
ok  .../tests/unit/task/domain
ok  .../tests/unit/task/transport/http
ok  .../tests/unit/cmd/api
```

Se agregó `PostgresTaskRepository.ListByProjectStory` (proyecto, historia del
proyecto, orden por `seq`), el handler `{project_id, story_id, tasks}` con
405/422/404/500 y la ruta `GET /projects/{project_id}/stories/{story_id}/tasks`
condicionada por `TaskDependencies.Lister`.

### Selector compartido y clientes TUI

**RED** (`98fadf6`) — `go test ./internal/tui ./tests/unit/tui`:

```
--- FAIL: TestSelectorFiltersLabelMetaAndIDWithoutLosingTheCollection
    filter "login" returned all three items
--- FAIL: TestSelectorSlashAndPrintableCharactersStartTheFilter
--- FAIL: TestSelectorEscapeClearsTheFilterBeforeGoingBack
--- FAIL: TestSelectorMovesTheCursorWithinVisibleItems
--- FAIL: TestSelectorChoosesOnlyWhenThereIsARow
--- FAIL: TestSelectorRefreshAndLifecycle
--- FAIL: TestSelectorSetItemsPreservesTheFilterAndClampsTheCursor
--- FAIL: TestTaskClientListByStoryUsesReadOnlyStoryTasksRoute
    requests = 0 ([]), want exactly 1
--- FAIL: TestSectionClientsEscapePathSegments
    requests = 0, want 2
```

El test básico de `MemberClient.List` ya pasaba porque la ruta simple era correcta;
el caso de escape de segmentos seguía en RED.

**GREEN** (`3a10d4d`) — `go test ./internal/tui ./tests/unit/tui`:

```
ok  .../internal/tui
ok  .../tests/unit/tui
```

`selector` implementa filtro por etiqueta/meta/ID, `/` y carácter inicial, `↑/↓`,
`Enter`, `Esc` (limpia filtro antes de volver), `r`, carga, error, vacío y
`setItems` con filtro preservado. `MemberClient.List` y `TaskClient.ListByStory`
usan GET y escapan los segmentos de ruta.

### Menú, ruteo, flujos y avisos

**RED** (`42da4ad`) — `go test ./tests/unit/tui`:

```
--- FAIL: TestMenuRoutesToEachSprint3Section/planning
    expected a command to execute, got nil
--- FAIL: TestMenuRoutesToEachSprint3Section/effort
    expected a command to execute, got nil
--- FAIL: TestMenuRoutesToEachSprint3Section/defects
    expected a command to execute, got nil
--- FAIL: TestMenuRoutesToEachSprint3Section/metrics
    expected a command to execute, got nil
--- FAIL: TestMenuRoutesToEachSprint3Section/dashboard
    expected a command to execute, got nil
--- FAIL: TestPlanningPokerFlowChoosesStoryAndMemberThenShowsTheTUI07Notice
    expected a command to execute, got nil
--- FAIL: TestEffortFlowChoosesStoryAndTaskThenShowsTheTUI08Notice
    expected a command to execute, got nil
--- FAIL: TestDefectsFlowChoosesStoryAndSprintThenShowsTheTUI09Notice
    expected a command to execute, got nil
--- FAIL: TestMetricsAndDashboardShowTheirFutureNoticeAfterChoosingASprint
    expected a command to execute, got nil
--- FAIL: TestSectionNavigationOnlyEmitsGetRequests
    expected a command to execute, got nil
```

`TestMenuExposesNineOptions` ya pasaba porque el esqueleto de `Model.Choices` se
incluyó en el commit RED. Los esqueletos de `sectionKind`, `sectionStep`,
`sectionState`, `openSection` y `SectionResultMsg` permitieron compilar y fallar
por aserciones.

**GREEN** (`230919c`) — `go test ./internal/tui ./tests/unit/tui`:

```
ok  .../internal/tui
ok  .../tests/unit/tui
```

La primera pasada GREEN dejó dos fallos reales, corregidos antes del commit:

1. `Esc` desde el aviso volvía siempre al selector principal. Se agregó
   `noticeReturn` para volver al paso que produjo el aviso (integrantes, tareas o
   Sprints) sin repetir el GET.
2. La aserción del pie esperaba un espacio simple; `keyHints` separa los pares con
   dos espacios. Se ajustó la prueba a `p  proyecto` y `?  ayuda`.

La implementación final cubre:

- Menú de nueve opciones y ruteo a `section`.
- Proyecto activo o `project-picker` con propósito `section:<kind>`.
- Carga principal de historias o Sprints abiertos.
- Segundo paso: integrantes, tareas de la historia o Sprints.
- Avisos TUI-07..TUI-11 con `No se guardó ningún dato`.
- `Esc` en aviso → segundo paso; en segundo paso → principal; en principal → menú.
- `r` refresca el paso actual; `p` cambia de proyecto.
- Filtro con `/`, carácter inicial, `↑/↓`, `Enter`, `Esc`, `r`.
- Pie y ayuda contextuales; sin `n`, `e`, `x`, `h` ni `Space` en secciones.
- Errores HTTP/de transporte y estados vacíos sin cerrar la TUI.

## Verificación final

Comandos ejecutados sobre el árbol final:

| Comando | Resultado |
|---------|-----------|
| `gofmt -l internal tests` | sin salida |
| `go vet ./...` | sin salida |
| `go build ./...` | sin salida |
| `go test ./tests/unit/...` | verde |
| `go test ./internal/...` | verde |
| `go test ./...` | 48 paquetes `ok`, 0 `FAIL` |

`go test ./...` incluyó los cinco paquetes de integración PostgreSQL con
Testcontainers: `project`, `projectmember`, `sprint`, `story` y `task`. Docker
estaba disponible (`docker info` respondió correctamente); no hubo paquetes
omitidos por falta de contenedor.

Conteo focalizado:

- `tests/unit/tui`: 123 pruebas de nivel superior en verde.
- `internal/tui`: 7 pruebas internas del selector en verde.

## Criterios de cierre

- [x] Menú de nueve opciones y ruteo directo a cada sección.
- [x] Selectores de integrante, historia, tarea y Sprint con datos reales.
- [x] Sin solicitudes de escritura durante la navegación (prueba con grabador de métodos).
- [x] Ayuda y pie contextuales, sin teclas ajenas.
- [x] Manejo de API caída y estados vacíos sin cerrar la TUI.
- [x] Dos rutas GET nuevas con pruebas de aplicación, handler y composición.
- [x] Suite unitaria y de integración en verde.

## Desviaciones y observaciones

- **Fase 5 combinada con la fase 4**: los flujos se probaron en el RED de menú/ruteo
  y se implementaron en el GREEN de shells. Se documenta en `tasks.md`.
- **Ruta de tareas sin `Lister`**: como la ruta GET tiene un patrón distinto al POST,
  sin `Lister` el mux responde 404, no 405. La prueba de composición se ajustó a 404.
  Integrantes comparte el patrón con el POST y por eso sin `Lister` responde 405.
- **Validación de UUID duplicada**: el handler valida los UUID antes de llamar al caso
  de uso y el caso de uso también los valida. La duplicación es deliberada para que
  cada capa sea segura por sí misma y las pruebas de handler puedan usar dobles.
- **`gofmt`**: además de los archivos nuevos, se normalizaron tres archivos de prueba
  preexistentes (`tests/unit/project/domain/project_test.go`,
  `tests/unit/story/transport/http/handler_test.go` y
  `tests/unit/task/transport/http/list_handler_test.go`) para que
  `gofmt -l internal tests` quedara limpio. Son cambios de alineación, sin comportamiento.
- **Sin migraciones**: no se tocó el esquema ni el gate de `schema_migrations`.
- **Prueba manual interactiva**: no se realizó una sesión manual de la TUI; la
  navegación se verificó con pruebas de caja negra sobre `Model.Update`, `httptest`
  y la suite de integración de la API.

## Tamaño de entrega

- Diff acumulado frente a `64cbd93`: **3.577 líneas** (3.552 añadidas, 25 eliminadas;
  34 archivos).
  - Producción (`internal/`, `cmd/`): 1.371 líneas (1.354 añadidas, 17 eliminadas).
  - Pruebas (`tests/`): 1.089 líneas (1.084 añadidas, 5 eliminadas).
  - OpenSpec: 1.083 líneas añadidas.
  - `README.md`: 34 líneas (31 añadidas, 3 eliminadas).
- Presupuesto de revisión: 400 líneas. Pronóstico: 1.200 a 1.600. El resultado real
  supera el pronóstico y el presupuesto; se mantiene la excepción `size:exception`
  aprobada. La mayor parte del excedente son pruebas de caja negra por escenario y
  los artefactos OpenSpec.

## Archivos principales

- API: `internal/projectmember/{application,infrastructure,transport}`,
  `internal/task/{application,infrastructure,transport}`, `internal/api/api.go`,
  `cmd/api/main.go`.
- TUI: `internal/tui/selector.go`, `sections.go`, `members.go`, `model.go`,
  `project_picker.go`, `sprints.go`.
- Pruebas: `tests/unit/projectmember/*`, `tests/unit/task/*`,
  `tests/unit/cmd/api/*_list_route_test.go`, `internal/tui/selector_test.go`,
  `tests/unit/tui/sections_*.go`.
- Documentación: `README.md`, `openspec/changes/tui-06-navegacion-sprint-3/*`.
