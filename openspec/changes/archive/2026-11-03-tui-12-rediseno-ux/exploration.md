# Exploración: rediseño de la experiencia de usuario de la TUI (TUI-12)

## Pedido

> bueno vamos a trabajar en rehacer o mejorar en un mil porciento la USER EXPERIENCE de la TUI, actualmente es muy confusa, poco intuitiva, repetitiva, mala para el usario. debe ser mejor, no no te limiter por las specs actuales o reglas actuales, todo esta sujeto a cambio o mejora. por ejemplo estaria bueno q el dashboard te reciba al entrar, y la seccion de proyectos esta mal porque todas las demas secciones van vinculadas a un solo proyecto en realidad deberia de seleciconar un proyecto y de ahi ver todas las demas secciones de ese proyecto.

## Estado actual (evidencia)

- **Menú plano de nueve opciones** (`internal/tui/model.go:150-160`, `updateMenu` en `model.go:391`): Proyectos aparece como una sección más, al mismo nivel que Backlog, Sprints, Planning Poker, Esfuerzo, Defectos, Métricas, Dashboard y Salir. El Dashboard es la octava opción.
- **El proyecto se elige "de rebote"**: cada sección sin proyecto activo abre el navegador de proyectos y vuelve con `pickerReturn`/`pickerPurpose` (`model.go:31-32`, `crumbParts` en `model.go:814-850`). El usuario no decide primero en qué proyecto trabaja; se lo piden a mitad de camino.
- **Atajos de cambio de proyecto inconsistentes**: `p` global (`model.go:233`) y `c` «proyecto» en Backlog (`model.go:966`).
- **Pantallas de resultado intermedias**: `project-result`, `backlog-result`, `sprint-result`, `effort-result`, `defect-create-result`, `defect-status-result` obligan a un `Enter` extra después de cada operación exitosa en vez de volver a la lista con un aviso.
- **Footers sobrecargados**: Sprints muestra 12 atajos en una línea (`model.go:973`); el usuario no distingue lo importante de lo secundario.
- **Métricas y Dashboard se solapan**: Métricas pide elegir un Sprint (`sections.go`), Dashboard muestra las métricas del proyecto; son dos lugares para el mismo concepto.
- **Pantalla de inicio sin información**: el «Panel de control» es solo la lista del menú más «Sin proyecto activo · elegí uno en Proyectos».
- **Modelo monolítico**: `Model` (`model.go:17-131`) tiene ~110 campos y el `Screen` es un string con ~35 valores; navegación, breadcrumb, footer y ayuda se resuelven con `switch` paralelos que deben mantenerse sincronizados a mano. Cada cambio de flujo toca cuatro lugares.

## Restricciones

- El backend no cambia: la TUI sigue consumiendo los endpoints HTTP existentes.
- `strict_tdd: true` y `go test ./...` (`openspec/config.yaml`).
- Artefactos técnicos en español (`conventions.technical_artifacts_language: es`).
- Presupuesto de revisión: 400 líneas por PR; este cambio lo supera y necesita entrega en cadena.

## Opciones de arquitectura de navegación

1. **Proyecto primero + espacio de trabajo con pestañas (recomendada)**: al abrir, la TUI muestra la lista de proyectos (o entra directo al último/único). Elegido el proyecto, se entra a su *espacio de trabajo*: una barra de pestañas `Dashboard · Backlog · Sprints · Estimación · Esfuerzo · Defectos` con el Dashboard como pestaña inicial. `Tab`/`1..6` cambian de pestaña; `Esc` en la raíz vuelve a Proyectos.
2. **Proyecto primero + menú lateral**: igual jerarquía, pero con una barra lateral persistente. Mejor descubribilidad, peor en terminales de 80 columnas.
3. **Mantener el menú plano y mejorar textos**: costo bajo, no resuelve la queja central (las secciones dependen de un proyecto que no se eligió explícitamente).
