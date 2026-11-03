# Diseño: Dashboard de la TUI (TUI-11)

## Decisión 1: pantalla dedicada, no una sección de navegación

El Dashboard se implementa como una pantalla propia (`Screen == "dashboard"`) y deja de recorrer el shell de `sectionDashboard`. Motivos:

- El issue pide abrir el Dashboard del **proyecto activo** directamente desde el menú, sin selector de Sprint; el shell de TUI-06 obliga a elegir un Sprint que el Dashboard no necesita.
- La pantalla combina dos consultas y una representación gráfica; mantenerla dentro de `sectionState` agregaría un estado y un `sectionStep` que no comparten las demás secciones.
- El patrón de pantalla dedicada ya existe en la TUI (Defectos, Planning Poker, formularios) y permite footer, ayuda y miga específicos.

Se retira `sectionDashboard` de `sectionKind` y el caso de `sectionNotice`; el resto de las secciones no cambia. El menú pasa a llamar `openDashboard()`.

## Decisión 2: dos consultas existentes, un solo resultado de carga

`openDashboard` valida el proyecto activo. Si no hay, abre el navegador de proyectos con propósito `dashboard`; al confirmar, vuelve a `openDashboard`. Con proyecto activo:

- `ProjectClient.Status(ctx, projectID)` → `GET /projects/{id}` para el estado.
- `MetricsClient.Project(ctx, projectID)` → `GET /projects/{id}/metrics` para las métricas.

Ambas se ejecutan en un único `tea.Cmd` secuencial y producen un `dashboardResultMsg` con una secuencia (`seq`). El modelo descarta el mensaje si la pantalla cambió, si la secuencia no coincide o si ya no está cargando. Esto evita respuestas tardías al cambiar de proyecto con `p` o al reintentar con `r`.

No se agregan clientes ni endpoints. Los errores se capturan por separado para poder mostrar el estado aunque las métricas fallen y viceversa:

```go
type dashboardState struct {
    seq       uint64
    loading   bool
    status    string
    statusErr string
    metrics   metricsPayload
    metricsOK bool
    metricsErr string
}
```

- `loading = true` mientras corre el comando.
- `statusErr`/`metricsErr` contienen el motivo legible (`Error de conexión: ...` o `Error HTTP N: ...`).
- `metricsOK` distingue un sobre válido de uno ilegible.

## Decisión 3: estado del proyecto

Se muestra `Estado:` con `projectStatusBadge(status)`; el badge traduce `planned`, `active` y `overdue` a `planificado`, `en curso` y `atrasado`. Si la consulta falla o el estado viene vacío, se muestra `sin datos` y el motivo del fallo en un bloque de error, sin cerrar la pantalla. El nombre del proyecto activo ya está en la sesión y en la miga; no se duplica como dato de la API.

## Decisión 4: valores y «sin datos»

Se reutilizan `parseMetricsPayload` y `formatMetric` de US-19/TUI-10. Reglas:

- Un puntero `nil` se muestra como `sin datos` y no produce barra.
- Un cero devuelto por la API se muestra como `0` (o `0 %`); no se confunde con ausencia.
- Las relaciones se muestran como `completado / planificado` o `real / estimado`; los valores son exactamente los de la API.
- El porcentaje de historias completadas usa directamente `completed_stories_percentage`; no se recalcula desde historias.
- La barra es una representación de la relación para lectura visual, no un valor nuevo: la etiqueta y los números siempre están presentes.

Filas:

| Fila | Valor | Barra |
| --- | --- | --- |
| Story Points | `completados / planificados` | `completed / planned` si ambos existen y `planned > 0` |
| Horas | `reales / estimadas h` | `actual / estimated` si ambos existen y `estimated > 0` |
| Historias completadas | `porcentaje %` | `percentage / 100` si existe |
| Defectos | `resueltos / detectados` | `resolved / detected` si ambos existen y `detected > 0` |
| Velocidad | valor | sin barra (no tiene denominador) |
| Desviación | valor | sin barra (puede ser negativa) |

## Decisión 5: barras y terminales angostas

`dashboardBar(fraction float64, width int)` dibuja `width` celdas con `█` llenas y `░` vacías, redondeando la fracción y recortándola al rango `[0, 1]`. `nil` o denominador cero producen cadena vacía (sin barra).

El ancho de barra se calcula desde el ancho interior: `min(28, max(12, innerWidth-10))`. A 44 columnas el ancho interior es 40, así que la barra mide como máximo 28 y la línea `barra + valor` entra en 40. Cada fila se dibuja en dos líneas (etiqueta + valor, y barra) para que las etiquetas largas no compitan con los glifos. El color es opcional: los valores numéricos, las etiquetas y los glifos `█/░` conservan el significado al quitar ANSI.

## Decisión 6: navegación, refresco y errores

- `Esc` vuelve al menú.
- `r` vuelve a lanzar ambas consultas conservando la pantalla y el proyecto activo.
- `p` usa el atajo global existente para abrir el navegador de proyectos; al confirmar, `openDashboard` vuelve a cargar.
- `?` abre la ayuda con las filas específicas de Dashboard.
- Si la API falla, la pantalla sigue abierta, muestra el motivo y ofrece `r` para reintentar.
- El footer y la ayuda no anuncian acciones que no existen (sin `n`, `e`, `Space`, etc.).
- La miga de pan muestra `Dashboard` y el proyecto activo; el navegador de proyectos muestra el propósito `Dashboard` cuando corresponde.

## Alternativas descartadas

- **Reutilizar la pantalla de Métricas con un flag de gráficos**: mezcla dos contratos (consulta de métricas y Dashboard) y obliga a que Métricas muestre barras o estados que no le corresponden.
- **Agregar un endpoint `/dashboard`**: duplica lecturas existentes; el issue limita el alcance a la TUI y US-19 ya expone todo lo necesario.
- **Calcular métricas en la TUI**: viola la consistencia con TUI-10/US-19 y el requisito de no recalcular.
- **Barras con color como único indicador**: incumple «no depender solo del color»; por eso se mantienen valores y glifos.
- **Cadena de PRs**: el flujo no tiene una porción entregable por sí sola (una pantalla sin datos o sin gráficos no satisface el issue) y el repo usa `single-pr`; se documenta excepción de tamaño.

## Pruebas previstas

1. Menú → Dashboard con proyecto activo: consulta proyecto y métricas y muestra estado, valores y barras.
2. Menú → Dashboard sin proyecto activo: abre el navegador y continúa en Dashboard al confirmar.
3. `nil` → `sin datos`, sin barra y sin `0`.
4. 80 y 44 columnas: ninguna línea visible excede el ancho; etiquetas, valores y glifos siguen presentes al quitar ANSI.
5. Error de proyecto o de métricas: motivo visible, pantalla abierta, `r` reintenta, `Esc` vuelve.
6. Solo `GET`: el recorrido del Dashboard no emite escrituras.
7. Refresco y cambio de proyecto: se descartan respuestas tardías y se recargan ambos datos.

## Riesgos y mitigaciones

- **Dos lecturas por pantalla**: es el mismo costo que Métricas más el estado; se ejecutan en un solo comando y se reintentan juntas.
- **Ancho de barra**: se calcula desde `innerWidth`; las pruebas de 44 columnas lo verifican.
- **Estado derivado por fecha**: la API lo calcula con su reloj; la TUI solo lo presenta. Las pruebas fijan el cuerpo de la respuesta.
- **Deuda de consolidación de TUI-09**: el requisito de avisos todavía menciona Defectos; este cambio solo retira Dashboard y deja constancia de que la consolidación de TUI-09 sigue pendiente.
