# Propuesta: navegación del Sprint 3 en la TUI (TUI-06)

## Motivo

El rediseño de la TUI (#98) dejó un patrón navegable común para Proyectos, Backlog y Sprints, pero el menú todavía expone solo esas tres secciones. Las historias **TUI-07** a **TUI-11** necesitan un punto de entrada único, selectores reutilizables y lecturas de integrantes y tareas que hoy no existen. Sin esta base, cada pantalla nueva repetiría ruteo, búsqueda, estados vacíos y manejo de errores, y algunas no podrían cumplir el requisito de operar sin escribir UUIDs.

TUI-06 es la historia habilitante del Sprint 3: agrega el menú ampliado, el ruteo a las secciones nuevas y el selector compartido. No implementa los flujos de negocio de las secciones; esos quedan para TUI-07 a TUI-11.

## Cambios

- Ampliar el menú a nueve opciones: Proyectos, Backlog, Sprints, Planning Poker, Esfuerzo, Defectos, Métricas, Dashboard y Salir.
- Rutear cada sección nueva a su pantalla principal, sin submenús de operaciones.
- Incorporar un selector compartido para integrantes, historias, tareas y Sprints, con el patrón de las listas navegables: `/` para buscar, `↑`/`↓`, `Enter`, `Esc`, `r` y `?`.
- Usar el chrome del rediseño #98 en las pantallas nuevas: panel, barra superior, miga de pan y pie con wrapping.
- Mostrar en la ayuda `?` y en el pie solo las teclas válidas de la pantalla activa.
- Agregar dos lecturas de solo lectura que faltan para que los selectores funcionen con datos reales:
  - `GET /projects/{project_id}/members`.
  - `GET /projects/{project_id}/stories/{story_id}/tasks`.
- Actualizar la especificación de la capacidad `tui` y las especificaciones de `project` y `tarea` con las nuevas lecturas.
- No emitir requests que modifiquen datos durante la navegación.

## Enfoque

- **TUI**: un componente `selector` no exportado concentra filtro, cursor, estados de carga/error/vacío y las teclas comunes. Las pantallas de sección lo reutilizan con historias, integrantes, tareas y Sprints.
- **Shells de sección**: cada opción nueva abre una pantalla de navegación con la colección principal de esa sección. Al elegir un elemento se muestra el siguiente selector que corresponda y, al finalizar, un aviso de que el flujo de negocio llega en TUI-07..TUI-11. No se ejecuta ninguna operación de negocio.
- **Proyecto activo**: si no hay proyecto elegido, la sección abre el navegador de proyectos existente; al confirmarlo, vuelve a la sección. El atajo global `p` sigue disponible.
- **API**: se agregan dos rutas GET con sus casos de uso, puertos, adaptadores PostgreSQL, handlers y composición condicionada por migraciones:
  - Integrantes: lista ordenada por nombre, `404 project_not_found`, `422` para UUID inválido.
  - Tareas: lista ordenada por secuencia de creación, `404 project_not_found` / `story_not_found`, `422` para UUID inválido.
- **TDD estricto**: cada comportamiento se cubre primero con una prueba en RED y luego con la implementación mínima en GREEN, según `openspec/config.yaml`.

## Capacidades

### Capacidades nuevas

- Ninguna.

### Capacidades modificadas

- `tui`:
  - «Mostrar menú base de navegación»: pasa de cuatro a nueve opciones y las secciones nuevas abren su pantalla principal.
  - Nuevo requisito «Abrir las secciones del Sprint 3 desde el menú».
  - Nuevo requisito «Elegir integrantes, historias, tareas y Sprints con selectores navegables».
  - «Navegación de solo lectura»: incluye las secciones nuevas y sus selectores; solo se permiten GET.
- `project`:
  - Nuevo requisito «Consultar los integrantes de un proyecto».
- `tarea`:
  - Nuevo requisito «Consultar las tareas de una historia».

## Impacto

- **API**: `internal/projectmember/*`, `internal/task/*`, `internal/api/api.go`, `cmd/api/main.go`.
- **TUI**: `internal/tui/model.go`, `internal/tui/selector.go` (nuevo), `internal/tui/sections.go` (nuevo), `internal/tui/members.go` (nuevo), `internal/tui/sprints.go` (listado de tareas), `internal/tui/project_picker.go` (ruteo de secciones).
- **Pruebas**: `tests/unit/projectmember/*`, `tests/unit/task/*`, `tests/unit/cmd/api/*`, `tests/unit/tui/*`.
- **Documentación**: `openspec/changes/tui-06-navegacion-sprint-3/*`, `README.md` (teclas y secciones).
- **Sin migraciones nuevas**: las tablas `project_members` y `tasks` ya existen en las migraciones 000006 y 000009.

## Presupuesto de revisión

El cambio supera el presupuesto orientativo de 400 líneas: se estiman **1.200 a 1.600 líneas** entre API, TUI, pruebas y documentación. Se propone un único PR con `size:exception`, manteniendo commits RED/GREEN separados y límites de reversión por unidad. La decisión queda registrada en `tasks.md` antes de aplicar.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| Duplicar el patrón de listas en vez de compartirlo | Media | El selector concentra filtro, cursor y teclas; las pantallas nuevas solo aportan datos y textos |
| Que los shells de sección se confundan con las pantallas funcionales de TUI-07..11 | Media | Aviso explícito de «disponible en TUI-0X», sin acciones de negocio ni mutaciones |
| Respuestas tardías de la API pisen otra pantalla | Media | `SectionResultMsg` incluye sección y paso; se descarta si no coinciden o si no hay carga activa |
| Rutas nuevas no registradas por migraciones faltantes | Baja | Composición condicionada a `Members` y `Tasks`; error de API visible sin cerrar la TUI |
| Superar el presupuesto de revisión | Alta | `size:exception` acordado, commits por unidad y pruebas focalizadas |

## Plan de reversión

Revertir el merge del PR. Las rutas nuevas son de solo lectura y no alteran datos; la TUI vuelve al menú de cuatro opciones. No hay migraciones que deshacer.

## Dependencias

- #98 (rediseño de la TUI) integrado en `main`.
- Migraciones 000006 (`project_members`) y 000009 (`tasks`) aplicadas para las lecturas nuevas.
- US-14/US-15 no se implementan acá: solo se preparan los selectores y las lecturas de navegación.

## Criterios de éxito

- [ ] El menú muestra las nueve opciones y cada una abre su pantalla principal.
- [ ] Las secciones nuevas no tienen submenús de operaciones.
- [ ] Los selectores de integrante, historia, tarea y Sprint permiten buscar, moverse, elegir, volver y refrescar con `/`, `↑/↓`, `Enter`, `Esc`, `r` y `?`.
- [ ] Ninguna pantalla pide UUIDs escritos a mano.
- [ ] La navegación emite solo GET y no modifica datos.
- [ ] La ayuda `?` y el pie muestran solo las teclas válidas de la pantalla activa.
- [ ] Las pantallas respetan panel, barra superior, miga y pie con wrapping en 80 y 44 columnas.
- [ ] Si la API no responde, la TUI muestra el motivo y sigue abierta.
- [ ] `go test ./tests/unit/...` pasa; `go test ./...` se ejecuta y se registra el resultado de las pruebas que requieren Docker.
