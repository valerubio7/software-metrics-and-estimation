# Uso de IA: evidencia y responsabilidad

Hay evidencia de asistencia de IA en la preparación de estos documentos y una declaración en el PR #66. Eso no demuestra que todos los integrantes usaran las mismas herramientas ni que todo código generado haya sido revisado por todos.

## Evidencia disponible

| Actividad | Herramienta y evidencia | Límite |
|---|---|---|
| Preparación de documentación | Pi / el Gentleman, en la sesión actual dirigida por el usuario: alcance, hechos confirmados y restricciones documentales. | La asistencia no sustituye la aprobación humana de hechos o propuestas. |
| Trabajo declarado en PR #66 | [PR #66](https://github.com/valerubio7/software-metrics-and-estimation/pull/66): su cuerpo incluye «Generated with Claude Code». | Esa frase no identifica por sí sola al operador, las partes generadas ni quién las validó. |

No se atribuye el uso de otras herramientas a integrantes concretos. Tampoco se registran firmas de revisión inexistentes ni una reconstrucción del historial de prompts.

## Responsabilidad humana

La [consigna](assignment.md#uso-de-inteligencia-artificial) exige comprender, revisar y validar todo resultado generado mediante IA. El equipo sigue siendo responsable del producto, cualquiera sea el origen del código o del texto.

Los cinco [Product Builders](scrum/team.md) comparten esa responsabilidad; no significa que cada integrante haya revisado cada cambio. Valentín es Agile Enabler, no un aprobador automático de todos los resultados. Los profesores son Product Architect según la consigna.

## Pendientes

- Identificar, con evidencia, los cambios asistidos por IA y la herramienta utilizada cuando corresponda.
- Registrar quién comprendió/revisó cada resultado y qué validación realizó.
- Confirmar las aprobaciones humanas de estos documentos y del Sprint Goal propuesto.
- Distinguir pruebas ejecutadas, lectura de código y decisiones de producto: ninguna reemplaza las otras.

La asistencia actual incluye inspección de fuentes locales y redacción documental; no se ejecutó una nueva suite Go ni se generó cobertura como parte de este trabajo. No se afirma una validación humana completa del código existente.

## Registro futuro — estructura propuesta

Usar una entrada por cambio significativo, vinculada al artefacto o PR correspondiente:

| Campo | Qué registrar |
|---|---|
| Fecha y cambio | Identificador de issue/PR y enlaces a archivos afectados. |
| Uso de IA | Herramienta, propósito y partes asistidas, sin publicar credenciales o datos privados. |
| Responsable | Persona que incorpora el resultado y puede explicar su funcionamiento. |
| Comprensión y revisión | Persona, fecha y observaciones concretas; no una firma automática. |
| Validación | Comando/check exacto, resultado observado y límites o fallos. |
| Decisión | Aprobación, corrección o pendiente, con evidencia enlazada. |

Esta tabla define un formato propuesto, no entradas históricas completadas. Las afirmaciones sobre revisión o validación deben añadirse solo después de observarlas.
