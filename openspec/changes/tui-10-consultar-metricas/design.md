# Design

## Context

La rama actual ya contiene el selector de ámbito Proyecto/Sprint, pero no el cliente ni el renderizado de métricas. El contrato disponible en la especificación de US-19 define las rutas GET, el JSON plano y los campos opcionales; la implementación de US-19 todavía no está integrada en `origin/main`. Véase `proposal.md` para el motivo y `specs/tui/spec.md` para el comportamiento observable.

## Goals / Non-Goals

**Goals:**
- Añadir consultas GET aisladas por ámbito mediante el cliente HTTP ya usado por la TUI.
- Mantener la selección del proyecto/Sprint actual, representar los valores desconocidos y cobertura, y manejar carga, error, refresco y retorno.
- Presentar el resultado con wrapping/recorte compatible con los anchos exigidos.

**Non-Goals:**
- Implementar o copiar el backend de US-19, modificar contratos de API, calcular métricas o cambiar persistencia.
- Fusionar ramas remotas de US-19 o de otra implementación de TUI-10.
- Implementar visualizaciones de TUI-11 o reportes de US-21.

## Decisions

- **Consumir las rutas GET publicadas por US-19.** Proyecto usa `/projects/{project_id}/metrics`; Sprint usa `/projects/{project_id}/sprints/{sprint_id}/metrics`. El proyecto y Sprint vienen de los selectores existentes, no de UUIDs ingresados. Alternativa descartada: definir rutas paralelas o hacer llamadas directas al almacenamiento, porque duplicaría el contrato y acoplaría la TUI al backend.
- **Decodificar un DTO de presentación tolerante a campos nulos y extensiones del JSON.** Los punteros conservan la diferencia entre `null` y cero; los campos `coverage` y `deviation_percentage` se leen si están presentes. Alternativa descartada: decodificar en tipos enteros no opcionales o ocultar campos no calculables, que perdería semántica observable.
- **Mantener consulta y renderizado como pasos de sección independientes.** Los resultados asíncronos se aplican solo al ámbito/paso que los originó; refrescar repite el GET del mismo ámbito, `Esc` vuelve al selector anterior. Alternativa descartada: bloquear el loop de Bubble Tea durante HTTP, porque degradaría la capacidad de responder a errores y navegación.
- **Usar formato textual con labels además del color.** Las filas se truncan/envuelven dentro del ancho de panel y el valor tiene texto «sin datos» para `null`. Alternativa descartada: depender de color o de tablas de ancho fijo, que se vuelven ilegibles a 44 columnas.
- **Probar el contrato desde el cliente HTTP con servidores de prueba.** Como US-19 no está en `main`, las pruebas de TUI fijan método, rutas, payloads y efectos de navegación sin exigir un backend integrado. La ejecución end-to-end queda supeditada a la integración de US-19; no se integrará esa dependencia en esta rama.

## Risks / Trade-offs

- [US-19 aún no está integrada en `main`] → fijar en pruebas el contrato documentado por su spec y dejar explícita la dependencia; no fusionar una rama remota por cuenta propia.
- [El payload puede contener más campos que los mostrados] → decodificar solo el contrato publicado y no rechazar campos JSON adicionales.
- [Un resultado de una consulta anterior puede llegar después de cambiar el ámbito] → asociar cada respuesta con su paso/ámbito y descartar resultados obsoletos.
- [Terminal de 44 columnas reduce espacio para etiquetas] → abreviar/truncar etiquetas de forma estable, sin quitar el nombre ni usar color como único indicador.

## Migration Plan

No hay migraciones ni cambios de persistencia. Al integrar US-19 en la rama base, ejecutar la suite completa con la API disponible y verificar el flujo TUI en ambos ámbitos. Si el contrato no coincide, detener la integración de la consulta hasta reconciliar la especificación de US-19; no añadir compatibilidad especulativa.
