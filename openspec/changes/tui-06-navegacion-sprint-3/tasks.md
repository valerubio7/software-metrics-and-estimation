# Tareas: navegación del Sprint 3 en la TUI (TUI-06)

## Pronóstico de carga de revisión

| Campo | Valor |
|-------|-------|
| Líneas modificadas estimadas | 1.200 a 1.600 (pruebas ~40 %, producción ~40 %, documentación ~20 %) |
| Riesgo frente al presupuesto de 400 líneas | Alto |
| PRs encadenados recomendados | No |
| División sugerida | Un único PR con `size:exception` y commits RED/GREEN por unidad |
| Estrategia de entrega | single-pr |
| Estrategia de encadenado | size-exception |

Decisión necesaria antes de apply: sí. El cambio incluye dos rutas GET nuevas, el menú/ruteo y un selector compartido. Se propone `size:exception` para un único PR; los commits por unidad mantienen límites de reversión independientes.

## Contexto de ejecución

- TDD estricto (`openspec/config.yaml`: `strict_tdd: true`, `test_command: go test ./...`). Cada RED es un commit propio y su salida fallida se registra en `apply-progress.md`.
- Un RED válido falla por aserciones, no por compilación. Cuando la prueba necesita tipos nuevos, se agrega el esqueleto mínimo en el mismo commit RED.
- Pruebas de caja negra de TUI en `tests/unit/tui/` con paquete `tui_test`; pruebas internas del selector en `internal/tui/selector_test.go` con paquete `tui`.
- API: pruebas de aplicación, handler y composición. Las pruebas de integración con PostgreSQL se ejecutan solo si Docker/Testcontainers está disponible.
- Commits convencionales, sin atribución de IA.
- Ramas: `feat/tui-06-navigation-sections`, basada en `origin/main` con #98 integrado.

## Unidades de trabajo sugeridas

| Unidad | Objetivo | Límite de reversión |
|--------|----------|---------------------|
| 1 | Lectura de integrantes en la API | `projectmember` + `api` + `cmd/api` |
| 2 | Lectura de tareas de una historia en la API | `task` + `api` + `cmd/api` |
| 3 | Selector compartido y clientes TUI | `internal/tui/selector.go`, `members.go`, `sprints.go` |
| 4 | Menú, ruteo y shells de sección | `internal/tui/model.go`, `sections.go`, `project_picker.go` |
| 5 | Flujos de selectores y avisos | `sections.go`, pruebas de flujo |
| 6 | Verificación y documentación | `README.md`, `apply-progress.md` |

## Fase 1: Lectura de integrantes (API)

### Commit 1 — RED

Commit: `test(api): cover project member listing`
Evidencia: pendiente.

- [ ] 1.1 Agregar `tests/unit/projectmember/application/list_members_test.go`: UUID inválido no toca el puerto; proyecto sin integrantes devuelve lista vacía; error del puerto se propaga; el caso de uso delega el UUID canónico.
- [ ] 1.2 Agregar `tests/unit/projectmember/http/list_handler_test.go`: 405 para métodos no GET/HEAD; 422 para UUID inválido; 404 `project_not_found`; 200 con `members: []`; 200 con integrantes y `email: null`; 500 sin detalle interno.
- [ ] 1.3 Agregar un esqueleto mínimo de `ListMembersUseCase`, `MemberLister` y `NewListMembersHandler` para que las pruebas fallen por aserciones, no por compilación.
- [ ] 1.4 Ejecutar `go test ./tests/unit/projectmember/...` y registrar el RED.

### Commit 2 — GREEN

Commit: `feat(api): list project members`
Evidencia: pendiente.

- [ ] 2.1 Implementar `internal/projectmember/application/list_members.go` con `MemberLister`, `ListMembersQuery`, `ProjectMembers` y `ListMembersUseCase`.
- [ ] 2.2 Implementar `PostgresMemberRepository.ListByProject`: verificar proyecto, consultar por `project_id`, ordenar por `full_name` e `id`, devolver `[]` no nulo.
- [ ] 2.3 Implementar `internal/projectmember/transport/http/list_handler.go` con la respuesta `{project_id, members}` y los códigos 405/422/404/500.
- [ ] 2.4 Extender `MemberDependencies` con `Lister`, componer `GET /projects/{project_id}/members` solo si está presente y cablear el repositorio en `cmd/api/main.go`.
- [ ] 2.5 Agregar `tests/unit/cmd/api/member_list_route_test.go`: la ruta existe solo con `Lister`; sin `Lister`, GET responde 405; la respuesta vacía es `[]`; el error interno no filtra detalles.
- [ ] 2.6 Verificar `go test ./tests/unit/projectmember/... ./tests/unit/cmd/api` y registrar el GREEN.

## Fase 2: Lectura de tareas de una historia (API)

### Commit 3 — RED

Commit: `test(api): cover story task listing`
Evidencia: pendiente.

- [ ] 3.1 Agregar `tests/unit/task/application/list_tasks_test.go`: UUID inválido de proyecto o historia; lista vacía; error del puerto; delegación de identificadores canónicos.
- [ ] 3.2 Agregar `tests/unit/task/transport/http/list_handler_test.go`: 405; 422 para cada UUID inválido; 404 `project_not_found`; 404 `story_not_found`; 200 con `tasks: []`; 200 con tareas y `estimated_hours: null`; 500 sin detalle interno.
- [ ] 3.3 Agregar el esqueleto mínimo de `ListTasksUseCase`, `TaskLister` y `NewListTasksHandler`.
- [ ] 3.4 Ejecutar `go test ./tests/unit/task/...` y registrar el RED.

### Commit 4 — GREEN

Commit: `feat(api): list tasks for a story`
Evidencia: pendiente.

- [ ] 4.1 Implementar `internal/task/application/list_tasks.go` con `TaskLister`, `ListTasksQuery`, `ProjectStoryTasks` y `ListTasksUseCase`.
- [ ] 4.2 Implementar `PostgresTaskRepository.ListByProjectStory`: verificar proyecto, verificar historia del proyecto, consultar por `project_id` y `story_id` ordenando por `seq`, devolver `[]` no nulo.
- [ ] 4.3 Implementar `internal/task/transport/http/list_handler.go` reutilizando `taskResponse` y `writeJSON`; respuesta `{project_id, story_id, tasks}`.
- [ ] 4.4 Extender `TaskDependencies` con `Lister`, componer `GET /projects/{project_id}/stories/{story_id}/tasks` solo si está presente y cablear el repositorio en `cmd/api/main.go`.
- [ ] 4.5 Agregar `tests/unit/cmd/api/task_list_route_test.go`: ruta condicionada, 405 sin `Lister`, 200 con `[]`, error interno sin detalle.
- [ ] 4.6 Verificar `go test ./tests/unit/task/... ./tests/unit/cmd/api` y registrar el GREEN.

## Fase 3: Selector compartido y clientes TUI

### Commit 5 — RED

Commit: `test(tui): cover shared selector and section clients`
Evidencia: pendiente.

- [ ] 5.1 Agregar `internal/tui/selector_test.go` (paquete `tui`): filtro por etiqueta/meta/ID, activación con `/` y con carácter inicial, `↑/↓`, `Enter` sobre vacío, `Esc` limpia filtro y luego vuelve, `r` devuelve `selectorRefresh`, `startLoading`/`fail`/`setItems`.
- [ ] 5.2 Agregar `tests/unit/tui/sections_api_test.go`: `MemberClient.List` usa `GET /projects/{pid}/members`; `TaskClient.ListByStory` usa `GET /projects/{pid}/stories/{sid}/tasks`; no hay cuerpo ni métodos mutantes.
- [ ] 5.3 Agregar el esqueleto mínimo de `selector`, `MemberClient`, `Member` y `TaskClient.ListByStory`.
- [ ] 5.4 Ejecutar `go test ./internal/tui ./tests/unit/tui` y registrar el RED.

### Commit 6 — GREEN

Commit: `feat(tui): add shared selector and section clients`
Evidencia: pendiente.

- [ ] 6.1 Implementar `internal/tui/selector.go` con `selectorItem`, `selector`, `selectorAction`, `visible`, `selected`, `update`, `startLoading`, `fail` y `setItems`.
- [ ] 6.2 Implementar `internal/tui/members.go` con `Member` y `MemberClient.List`.
- [ ] 6.3 Agregar `Task` y `TaskClient.ListByStory` en `internal/tui/sprints.go`.
- [ ] 6.4 Implementar `viewSelector` reutilizando `searchLine`, `emptyState`, `errorBlock` y `row`.
- [ ] 6.5 Verificar `go test ./internal/tui ./tests/unit/tui` y registrar el GREEN.

## Fase 4: Menú, ruteo y shells

### Commit 7 — RED

Commit: `test(tui): cover Sprint 3 menu and section routing`
Evidencia: pendiente.

- [ ] 7.1 Agregar `tests/unit/tui/sections_test.go`: el menú tiene nueve opciones; cada índice nuevo abre `Screen == "section"` con el `kind` correcto; sin proyecto activo abre `project-picker` y al confirmar vuelve a la sección.
- [ ] 7.2 Cubrir carga de historias para Planning Poker, Esfuerzo y Defectos, y de Sprints para Métricas y Dashboard, con `httptest`.
- [ ] 7.3 Cubrir error de transporte y HTTP no-2xx dentro de la sección sin cerrar la TUI; `r` refresca; `Esc` vuelve al menú.
- [ ] 7.4 Cubrir que la navegación solo emite GET (registrar métodos del servidor falso).
- [ ] 7.5 Agregar esqueletos mínimos de `sectionKind`, `sectionStep`, `sectionState`, `openSection` y `SectionResultMsg`.
- [ ] 7.6 Ejecutar `go test ./tests/unit/tui` y registrar el RED.

### Commit 8 — GREEN

Commit: `feat(tui): add Sprint 3 menu and section shells`
Evidencia: pendiente.

- [ ] 8.1 Ampliar `Model.Choices` a nueve opciones y rutear índices 3 a 7 a `openSection`.
- [ ] 8.2 Implementar `internal/tui/sections.go`: estado, carga principal, `applySectionResult`, `updateSection`, `viewSection`, títulos, hints, estados vacíos y errores.
- [ ] 8.3 Extender `selectProject` para el propósito `section:<kind>` y `openProjectPicker` para volver a la sección.
- [ ] 8.4 Incorporar `section` a `quitsOnQ`, `filtering()`, `crumbParts`, `viewBody` y `footer`.
- [ ] 8.5 Verificar `go test ./tests/unit/tui` y registrar el GREEN.

## Fase 5: Flujos de selectores y avisos

### Commit 9 — RED

Commit: `test(tui): cover Sprint 3 selector flows`
Evidencia: pendiente.

- [ ] 9.1 Planning Poker: historia → integrantes → aviso TUI-07; el GET de integrantes usa el proyecto activo.
- [ ] 9.2 Esfuerzo: historia → tareas → aviso TUI-08; el GET de tareas usa proyecto e historia elegidos.
- [ ] 9.3 Defectos: historia → Sprints abiertos → aviso TUI-09.
- [ ] 9.4 Métricas: Sprint → aviso TUI-10. Dashboard: Sprint → aviso TUI-11.
- [ ] 9.5 Cubrir `Esc` desde aviso y segundo paso, `r` en cada paso, filtro en cada selector y ausencia de POST/PUT.
- [ ] 9.6 Ejecutar `go test ./tests/unit/tui` y registrar el RED.

### Commit 10 — GREEN

Commit: `feat(tui): add Sprint 3 selector flows`
Evidencia: pendiente.

- [ ] 10.1 Implementar el avance de paso y las cargas secundarias en `sections.go`.
- [ ] 10.2 Implementar los avisos de límite con la historia futura correspondiente.
- [ ] 10.3 Implementar el pie y la ayuda específicos por sección y paso.
- [ ] 10.4 Verificar `go test ./tests/unit/tui` y registrar el GREEN.

## Fase 6: Verificación y documentación

### Commit 11 — Documentación

Commit: `docs(tui): document Sprint 3 navigation`
Evidencia: pendiente.

- [ ] 11.1 Actualizar `README.md` con las secciones nuevas y las teclas comunes.
- [ ] 11.2 Registrar en `apply-progress.md` el alcance, los RED/GREEN, los comandos ejecutados y el estado de Docker/Testcontainers.
- [ ] 11.3 Ejecutar `gofmt -l internal tests`, `go vet ./...`, `go build ./...`.
- [ ] 11.4 Ejecutar `go test ./tests/unit/...` y `go test ./...`; registrar el resultado real. Si Docker no está disponible, dejar constancia de las pruebas de integración no ejecutadas.
- [ ] 11.5 Revisar el diff medido y confirmar si se mantiene el `size:exception`.

## Criterios de cierre

- [ ] Menú de nueve opciones y ruteo directo a cada sección.
- [ ] Selectores de integrante, historia, tarea y Sprint funcionando con datos reales.
- [ ] Sin solicitudes de escritura durante la navegación.
- [ ] Ayuda y pie contextuales.
- [ ] Manejo de API caída y estados vacíos.
- [ ] Dos rutas GET nuevas con pruebas de aplicación, handler y composición.
- [ ] Suite unitaria en verde y resultado completo registrado.
