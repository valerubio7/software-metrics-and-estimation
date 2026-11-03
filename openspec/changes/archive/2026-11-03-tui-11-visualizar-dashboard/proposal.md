# Propuesta: visualizar el Dashboard desde la TUI (TUI-11)

## Intención

TUI-11 (#104) permite obtener una visión general del proyecto activo desde la TUI: su estado actual y sus principales métricas, con una representación gráfica de terminal. La navegación de TUI-06 ya expone Dashboard como destino del menú, pero termina en un aviso que remite a esta historia. Las métricas de US-19 ya están disponibles por HTTP y la pantalla de Métricas de TUI-10 las presenta como texto; este cambio reemplaza el aviso por una pantalla de solo lectura que combina estado y métricas y las representa con barras Unicode.

## Cambios propuestos

- Abrir Dashboard directamente desde el menú para el proyecto activo, sin selector de Sprint ni identificadores escritos a mano. Si no hay proyecto activo, reutilizar el navegador de proyectos y continuar en Dashboard al confirmarlo.
- Consultar `GET /projects/{project_id}` y `GET /projects/{project_id}/metrics` con los clientes existentes. No se agregan endpoints, cálculos de negocio ni reglas nuevas al backend.
- Mostrar el estado derivado del proyecto (`planned`, `active`, `overdue`) con una etiqueta textual y el nombre del proyecto activo.
- Presentar las métricas de US-19 con los valores exactos que devuelve la API. Una métrica ausente (`null`) se muestra como «sin datos», nunca como cero.
- Representar con barras Unicode las relaciones disponibles: Story Points completados/planificados, horas reales/estimadas, porcentaje de historias completadas y defectos resueltos/detectados. Velocidad y desviación se muestran como valores.
- Mantener la pantalla legible en terminales de 80 y 44 columnas, sin depender del color: cada barra conserva su valor numérico y sus glifos.
- Mostrar la carga, el error de la API dentro de la pantalla y permitir reintentar con `r` o volver con `Esc` sin cerrar la TUI.
- Retirar el aviso «TUI-11» del shell de navegación y actualizar README y la especificación consolidada de `tui`.

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `tui`: agrega el requisito del Dashboard del proyecto activo (TUI-11) y retira Dashboard del aviso de flujos pendientes de TUI-06. La navegación de solo lectura se mantiene: el Dashboard solo emite consultas `GET`.

## Impacto

Código de la TUI (`internal/tui/dashboard.go` nuevo; ajustes en `model.go`, `project_picker.go` y `sections.go`), pruebas unitarias de la TUI, `README.md`, el delta de `tui` y el archivado del cambio. No cambia código de API, dominio, migraciones ni datos. La entrega es un único PR sobre `main` con `Closes #104`; el volumen estimado supera el presupuesto de 400 líneas y se documenta como excepción de tamaño, igual que TUI-08, porque el flujo no tiene una porción entregable que justifique una cadena de PRs.
