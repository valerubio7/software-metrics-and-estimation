# Propuesta: limpiar las referencias al menú eliminado en la especificación `tui` (TUI-13)

## Intención

TUI-12 reemplazó el menú de nueve opciones por el navegador de proyectos y el espacio de trabajo con pestañas, integró Métricas en el Dashboard, cambió las pantallas de resultado por avisos `✓` y eliminó los atajos `p` y `c`. Algunos requisitos de la especificación consolidada `tui` no estaban en ese delta y siguen describiendo el comportamiento anterior. Además, el paso de aviso de TUI-06 («Próximamente») quedó inalcanzable en el código. Este cambio alinea la especificación con lo implementado y borra ese código muerto.

## Cambios propuestos

- Reescribir los requisitos «Navegación de solo lectura», «Operar proyectos mediante la TUI», «Operar el backlog mediante la TUI», «Elegir historia, tarea e integrante para registrar esfuerzo» e «Iniciar en el navegador de proyectos» para que hablen de pestañas, avisos `✓` y `Esc` a Proyectos en lugar del menú, las pantallas de resultado y la tecla `c`. Se conservan todos sus escenarios.
- Eliminar el requisito «Avisar el límite de las pantallas de navegación»: cada pestaña tiene su flujo (Estimación TUI-07, Esfuerzo TUI-08, Defectos TUI-09, Dashboard TUI-11 con las métricas).
- Borrar de `internal/tui` el paso `sectionStepNotice`, `sectionNotice`, `viewSectionNotice`, los campos `notice`/`noticeReturn` y sus ramas en el pie, la ayuda y la miga de pan. Elegir una fila en un paso sin siguiente paso deja la lista como está.

## Capacidades

### Capacidades modificadas

- `tui`: cinco requisitos modificados y uno eliminado; ver `specs/tui/spec.md`.

## Impacto

Solo `internal/tui/sections.go`, `keys.go`, `model.go`, una prueba nueva en `internal/tui/sections_test.go` y la especificación `tui`. No cambia API, dominio, migraciones ni datos. El `README.md` ya no mencionaba el menú ni los avisos.

## Fuera de alcance

- Eliminar el tipo `sectionDefects` de las secciones: la pestaña Defectos usa su propio flujo, pero el tipo sigue nombrando la sección en la miga de pan y el selector.
