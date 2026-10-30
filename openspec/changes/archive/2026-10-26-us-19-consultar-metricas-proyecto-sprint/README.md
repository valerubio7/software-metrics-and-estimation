# Consultar métricas por proyecto o Sprint

US-19, issue #47. Cambio `spec-driven` preparado en español, únicamente como especificación previa a implementación. Cubre S1-S8 de `odd/tasks/us-19-metricas.md`, incluidas las decisiones de borde de L9 y la atribución estricta de esfuerzo aprobada en L12.

## Ruta de lectura

1. [Exploración](exploration.md): evidencia, criterios exactos y trazabilidad.
2. [Propuesta](proposal.md): motivación y alcance.
3. [Especificación de métricas](specs/metricas/spec.md): contrato observable y escenarios.
4. [Diseño](design.md): agregaciones, cobertura y compatibilidad.
5. [Tareas](tasks.md): implementación y verificaciones pendientes.

Las métricas permanecen en el objeto JSON raíz; `deviation` sigue siendo diferencia de horas. Se agregan `deviation_percentage` y `coverage`. Proyecto cuenta todo el esfuerzo una vez; Sprint solo el vinculado a tareas de ese Sprint. Los registros directos a historia se excluyen del total Sprint y se informan en `coverage.actual_hours.unattributed_story_level_records`. Se excluyen TUI-10 (#103), US-20 y US-21.

## Comprobación documental

```sh
openspec validate us-19-consultar-metricas-proyecto-sprint
openspec validate us-19-consultar-metricas-proyecto-sprint --strict
```

Los marcadores estructurales canónicos de OpenSpec se conservan por compatibilidad con su validador; el contenido está en español. La validación estructural no acredita funcionamiento de la API. RED/GREEN no corresponde a autoría pasiva; pruebas unitarias, integración, build y vet quedan para implementación según las tareas.

## Siguiente paso

Revisar los artefactos y ejecutar las tareas sin ampliar el alcance. El archivo final y los actos Git pertenecen al orquestador, no a esta entrega documental.
