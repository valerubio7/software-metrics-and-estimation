# Trazabilidad de las seis US implementadas

Este recorrido selecciona criterios y escenarios representativos de las seis US implementadas; no afirma cobertura exhaustiva. Las especificaciones canónicas son la referencia de comportamiento, y las pruebas enlazadas son funciones reales de Go. No hay un runner independiente de Gherkin ni ejecución automática de los encabezados Markdown como escenarios.

## US-01 — Crear proyecto

- Historia: [issue #29](https://github.com/valerubio7/software-metrics-and-estimation/issues/29).
- Especificación y escenario: [Especificación de Proyecto — «Escenario: Crear un proyecto válido»](../openspec/specs/project/spec.md#escenario-crear-un-proyecto-válido).
- Criterio: persistir nombre y fechas; la finalización planificada no puede preceder al inicio. No se introduce estado del proyecto.
- Prueba: [tests/unit/project/transport/http/handler_test.go](../tests/unit/project/transport/http/handler_test.go), `TestCreateProjectHandlerCreatesProjectWithoutStatus`.
- Regla alternativa: el mismo archivo contiene `TestCreateProjectHandlerAcceptsEqualDates`, que protege la igualdad entre fechas.
- Implementación: [internal/project/application/create_project.go](../internal/project/application/create_project.go) y [transporte HTTP](../internal/project/transport/http/handler.go).

## US-02 — Modificar proyecto

- Historia: [issue #30](https://github.com/valerubio7/software-metrics-and-estimation/issues/30).
- Especificación y escenario: [Especificación de Proyecto — «Scenario: Actualizar todos los campos básicos de un proyecto existente»](../openspec/specs/project/spec.md#scenario-actualizar-todos-los-campos-básicos-de-un-proyecto-existente).
- Criterio: reemplazar `name`, `start_date` y `planned_finish_date` conservando el ID; validar antes de persistir.
- Prueba: [tests/unit/project/application/update_project_test.go](../tests/unit/project/application/update_project_test.go), `TestUpdateProjectUseCaseReplacesValidProject`.
- Caso negativo: en ese archivo, `TestUpdateProjectUseCaseRejectsInvalidReplacementWithoutPersisting` protege el rechazo sin escritura.
- Implementación: [internal/project/application/update_project.go](../internal/project/application/update_project.go) y [repositorio PostgreSQL](../internal/project/infrastructure/postgres/repository.go).

## US-05 — Crear historia en el Product Backlog

- Historia: [issue #33](https://github.com/valerubio7/software-metrics-and-estimation/issues/33).
- Especificación y escenario: [Especificación de Historia — «Scenario: Crear una historia pendiente de estimación»](../openspec/specs/historia/spec.md#scenario-crear-una-historia-pendiente-de-estimación).
- Criterio: asociar al proyecto existente, conservar prioridad/criterios e inicializar `pendiente`, `story_points: null` y `estimated_hours: null`.
- Prueba: [tests/unit/story/transport/http/handler_test.go](../tests/unit/story/transport/http/handler_test.go), `TestCreateStoryReturnsPendingUnestimatedStory`.
- Caso negativo: `TestCreateStoryRejectsInvalidFieldsBeforeWriting`, en el mismo archivo, verifica validaciones sin escritura.
- Implementación: [internal/story/application/create_story.go](../internal/story/application/create_story.go) y [transporte HTTP](../internal/story/transport/http/handler.go).

## US-06 — Modificar historia

- Historia: [issue #34](https://github.com/valerubio7/software-metrics-and-estimation/issues/34).
- Especificación y escenario: [Especificación de Historia — «Scenario: Modificar los seis campos de una historia existente»](../openspec/specs/historia/spec.md#scenario-modificar-los-seis-campos-de-una-historia-existente).
- Criterio: reemplazar título, descripción, prioridad, estado, criterios y horas estimadas; conservar identidad y Story Points.
- Prueba: [tests/unit/story/application/update_story_test.go](../tests/unit/story/application/update_story_test.go), `TestUpdateStoryWritesOnceWithCanonicalIdentifiersAndAllSixFields`.
- Escenario seleccionado adicional: [«Scenario: Aceptar `estimated_hours` presente con valor nulo»](../openspec/specs/historia/spec.md#scenario-aceptar-estimated_hours-presente-con-valor-nulo).
- Prueba de esa distinción: en el mismo archivo, `TestUpdateStoryDistinguishesAbsentEstimatedHoursFromExplicitNull`. Omitir la clave se rechaza; enviarla con `null` borra la estimación.
- Implementación: [internal/story/application/update_story.go](../internal/story/application/update_story.go) y [handler de modificación](../internal/story/transport/http/update_handler.go).

## US-07 — Consultar Product Backlog

- Historia: [issue #35](https://github.com/valerubio7/software-metrics-and-estimation/issues/35).
- Especificación y escenario: [Especificación de Historia — «Scenario: Desempatar por orden de creación con prioridades repetidas»](../openspec/specs/historia/spec.md#scenario-desempatar-por-orden-de-creación-con-prioridades-repetidas).
- Criterio: ordenar `alta`, `media`, `baja`; a igual prioridad, conservar orden de creación. El ejemplo S1–S5 devuelve S2, S5, S1, S3, S4.
- Prueba: [tests/unit/story/domain/backlog_test.go](../tests/unit/story/domain/backlog_test.go), `TestNewBacklogOrdersByPriority`, subcaso `spec scenario S1 to S5 breaks ties by creation order`.
- Caso alternativo: [«Scenario: La lista vacía se serializa como lista y no como nulo»](../openspec/specs/historia/spec.md#scenario-la-lista-vacía-se-serializa-como-lista-y-no-como-nulo), protegido por `TestListStoriesRespondsWithAnEmptyArrayNeverNull` en [list_handler_test.go](../tests/unit/story/transport/http/list_handler_test.go).
- Implementación: [dominio del backlog](../internal/story/domain/backlog.go), [caso de uso](../internal/story/application/list_stories.go) y [repositorio PostgreSQL](../internal/story/infrastructure/postgres/repository.go).
- Límite: el test de dominio parte de una lista ya ordenada por creación; no prueba por sí solo la obtención de ese orden en PostgreSQL.

## US-08 — Crear Sprint

- Historia: [issue #36](https://github.com/valerubio7/software-metrics-and-estimation/issues/36).
- Especificación y escenario: [Especificación de Sprint — «Escenario: Crear un Sprint válido»](../openspec/specs/sprint/spec.md#escenario-crear-un-sprint-válido).
- Criterio: generar UUID, persistir exactamente un Sprint con proyecto existente y Sprint Goal; no asignar historias en esta operación.
- Prueba: [tests/unit/sprint/application/create_sprint_test.go](../tests/unit/sprint/application/create_sprint_test.go), `TestCreateSprintPersistsOnceAndReturnsServerGeneratedSprint`.
- Caso negativo: `TestCreateSprintRejectsInvalidGoalBeforeGeneratingIDOrWriting`, en el mismo archivo, protege la validación previa.
- Implementación: [internal/sprint/application/create_sprint.go](../internal/sprint/application/create_sprint.go) y [repositorio PostgreSQL](../internal/sprint/infrastructure/postgres/repository.go).

## Cómo interpretar la evidencia

Los enlaces permiten inspeccionar especificaciones, pruebas e implementación; no registran una ejecución nueva ni un porcentaje de cobertura. Los casos unitarios usan dobles donde corresponde: una escritura observada por un doble no equivale a un despliegue validado. Hay pruebas de integración en [tests/integration](../tests/integration/), pero el [riesgo de migraciones](architecture/overview.md#disponibilidad-al-arrancar) sigue pendiente.

Los cambios SDD históricos están en [openspec/changes/archive](../openspec/changes/archive/). Este documento usa las especificaciones canónicas actuales, que pueden incorporar reglas añadidas después de la creación original de cada US.
