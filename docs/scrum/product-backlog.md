# Product Backlog

* **Proyecto:** Software Metrics & Estimation
* **Fecha de corte:** 05/11/2026
* **Fuente:** GitHub Projects #4, issues del repositorio y changes archivados en `openspec/changes/archive/`.
* **Ítems:** 33 historias y tareas de producto; 32 completadas y 1 en progreso en el tablero.

> **Naturaleza del documento.** Es la vista versionada del Product Backlog para
> la entrega. La fuente operativa sigue siendo el [tablero](https://github.com/users/valerubio7/projects/4);
> si se agrega o cambia una historia, hay que actualizar ambos.

## Resumen

| Épica | Historias | Completadas |
|---|---:|---:|
| EPIC-01 — Gestión de Proyectos | 4 | 4 |
| EPIC-02 — Product Backlog | 3 | 3 |
| EPIC-03 — Gestión de Sprints | 6 | 6 |
| EPIC-04 — Estimación | 2 | 2 |
| EPIC-05 — Registro de Esfuerzo | 2 | 2 |
| EPIC-06 — Gestión de Defectos | 4 | 4 |
| EPIC-07 — Métricas | 2 | 1 |
| EPIC-08 — Dashboard | 9 | 9 |
| EPIC-09 — Reportes | 1 | 1 |
| **Total** | **33** | **32** |

## Cómo está ordenado

El tablero no tiene cargadas las prioridades ni las estimaciones, así que el
orden refleja la secuencia de épicas y, dentro de cada una, el orden en que las
historias entraron al backlog. Cada historia enlaza su issue y, cuando existe,
su change de OpenSpec con la especificación y los escenarios BDD.

## Historias por épica

### EPIC-01 — Gestión de Proyectos

- **US-01 — Crear proyecto** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** crear un proyecto registrando su información básica, **para** poder comenzar a gestionar su planificación, seguimiento y medición.
  [issue #29](https://github.com/valerubio7/software-metrics-and-estimation/issues/29) · [change](../../openspec/changes/archive/2026-09-23-us-01-create-project)

- **US-02 — Modificar un proyecto** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** modificar la información de un proyecto existente, **para** mantener actualizados sus datos durante su desarrollo.
  [issue #30](https://github.com/valerubio7/software-metrics-and-estimation/issues/30) · [change](../../openspec/changes/archive/2026-09-25-us-02-modify-project) · [PR #57](https://github.com/valerubio7/software-metrics-and-estimation/pull/57)

- **US-03 — Registrar integrantes en un proyecto** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** registrar integrantes en un proyecto, **para** identificar a las personas que forman parte del equipo responsable de su desarrollo.
  [issue #31](https://github.com/valerubio7/software-metrics-and-estimation/issues/31) · [change](../../openspec/changes/archive/2026-10-01-us-03-register-project-members) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/69)

- **US-04 — Consultar el estado de un proyecto** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** consultar el estado de un proyecto, **para** conocer su situación actual y realizar su seguimiento.
  [issue #32](https://github.com/valerubio7/software-metrics-and-estimation/issues/32) · [change](../../openspec/changes/archive/2026-10-01-us-04-project-status) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/70)

### EPIC-02 — Product Backlog

- **US-05 — Crear una historia en el Product Backlog** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** crear una historia dentro del Product Backlog de un proyecto, **para** registrar una funcionalidad que deberá ser considerada para el desarrollo del producto.
  [issue #33](https://github.com/valerubio7/software-metrics-and-estimation/issues/33) · [change](../../openspec/changes/archive/2026-09-24-us-05-crear-historia-product-backlog) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/56)

- **US-06 — Modificar una historia del Product Backlog** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** modificar una historia existente del Product Backlog, **para** mantener actualizada su información a medida que evoluciona el proyecto.
  [issue #34](https://github.com/valerubio7/software-metrics-and-estimation/issues/34) · [change](../../openspec/changes/archive/2026-09-25-us-06-modificar-historia-product-backlog) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/63)

- **US-07 — Consultar el Product Backlog** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** consultar el Product Backlog de un proyecto, **para** conocer las historias existentes y decidir sobre su planificación y desarrollo.
  [issue #35](https://github.com/valerubio7/software-metrics-and-estimation/issues/35) · [change](../../openspec/changes/archive/2026-09-26-us-07-consultar-product-backlog) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/64)

### EPIC-03 — Gestión de Sprints

- **US-08 — Crear un Sprint y definir su Sprint Goal** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** crear un Sprint definiendo su objetivo, **para** organizar una iteración de trabajo orientada a un resultado concreto.
  [issue #36](https://github.com/valerubio7/software-metrics-and-estimation/issues/36) · [change](../../openspec/changes/archive/2026-09-27-us-08-create-sprint) · [PR #65](https://github.com/valerubio7/software-metrics-and-estimation/pull/65)

- **US-09 — Asignar historias a un Sprint** · Sprint 1 — MVP · Completada
  **Como** usuario del sistema, **quiero** asignar historias del Product Backlog a un Sprint, **para** definir el trabajo planificado para esa iteración.
  [issue #37](https://github.com/valerubio7/software-metrics-and-estimation/issues/37) · [change](../../openspec/changes/archive/2026-10-03-us-09-asignar-historias-sprint) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/71)

- **US-10 — Descomponer una historia del Sprint en tareas** · Sprint 1 — MVP · Completada
  **Como** integrante del equipo, **quiero** descomponer una historia asignada a un Sprint en tareas, **para** organizar el trabajo necesario para completar su implementación.
  [issue #38](https://github.com/valerubio7/software-metrics-and-estimation/issues/38) · [change](../../openspec/changes/archive/2026-10-03-us-10-descomponer-historia-tareas) · [PR #80](https://github.com/valerubio7/software-metrics-and-estimation/pull/80)

- **US-11 — Registrar una historia del Sprint como completada** · Sprint 1 — MVP · Completada
  **Como** integrante del equipo, **quiero** registrar una historia del Sprint como completada, **para** reflejar el trabajo efectivamente terminado durante la iteración.
  [issue #39](https://github.com/valerubio7/software-metrics-and-estimation/issues/39) · [change](../../openspec/changes/archive/2026-10-04-us-11-registrar-historia-completada) · [PR #81](https://github.com/valerubio7/software-metrics-and-estimation/pull/81)

- **US-12 — Cerrar un Sprint** · Sprint 2 — Interfaz Objetivo · Completada
  **Como** usuario del sistema, **quiero** cerrar un Sprint, **para** finalizar formalmente la iteración y conservar sus resultados para el seguimiento del proyecto.
  [issue #40](https://github.com/valerubio7/software-metrics-and-estimation/issues/40) · [change](../../openspec/changes/archive/2026-10-13-us-12-cerrar-sprint) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/91)

- **US-13 — Consultar Sprints anteriores** · Sprint 2 — Interfaz Objetivo · Completada
  **Como** usuario del sistema, **quiero** consultar los Sprints anteriores de un proyecto, **para** revisar el trabajo realizado y analizar la evolución del proyecto.
  [issue #41](https://github.com/valerubio7/software-metrics-and-estimation/issues/41) · [change](../../openspec/changes/archive/2026-10-14-us-13-consultar-sprints-anteriores) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/94)

### EPIC-04 — Estimación

- **US-14 — Estimar una historia mediante Planning Poker** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** integrante del equipo, **quiero** estimar una historia mediante Planning Poker, **para** obtener de forma colaborativa una estimación acordada de su esfuerzo relativo en Story Points.
  [issue #42](https://github.com/valerubio7/software-metrics-and-estimation/issues/42) · [PR #107](https://github.com/valerubio7/software-metrics-and-estimation/pull/107)

- **TUI-07 — Estimar historias con Planning Poker desde la TUI** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** integrante del equipo, **quiero** estimar una historia con Planning Poker desde la TUI, **para** acordar sus Story Points sin usar la API directamente.
  [issue #100](https://github.com/valerubio7/software-metrics-and-estimation/issues/100) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/107)

### EPIC-05 — Registro de Esfuerzo

- **US-15 — Registrar el esfuerzo realizado en una historia o tarea** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** integrante del equipo, **quiero** registrar el esfuerzo realizado sobre una historia o tarea, **para** dejar constancia del trabajo efectivamente invertido durante el desarrollo.
  [issue #43](https://github.com/valerubio7/software-metrics-and-estimation/issues/43) · [change](../../openspec/changes/archive/2026-10-25-us-15-registrar-esfuerzo) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/109)

- **TUI-08 — Registrar esfuerzo desde la TUI** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** integrante del equipo, **quiero** registrar el esfuerzo realizado sobre una historia o tarea desde la TUI, **para** dejar constancia del trabajo invertido sin usar la API directamente.
  [issue #101](https://github.com/valerubio7/software-metrics-and-estimation/issues/101) · [change](../../openspec/changes/archive/2026-10-31-tui-08-registrar-esfuerzo) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/128)

### EPIC-06 — Gestión de Defectos

- **US-16 — Registrar un defecto** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** integrante del equipo, **quiero** registrar un defecto detectado durante el desarrollo, **para** mantener evidencia de los problemas encontrados y poder realizar su seguimiento.
  [issue #44](https://github.com/valerubio7/software-metrics-and-estimation/issues/44) · [change](../../openspec/changes/archive/2026-10-25-us-16-registrar-defecto) · [PR #110](https://github.com/valerubio7/software-metrics-and-estimation/pull/110)

- **US-17 — Actualizar el estado y resolución de un defecto** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** integrante del equipo, **quiero** actualizar el estado de un defecto y registrar su resolución, **para** reflejar correctamente su evolución hasta que haya sido solucionado.
  [issue #45](https://github.com/valerubio7/software-metrics-and-estimation/issues/45) · [PR #11](https://github.com/valerubio7/software-metrics-and-estimation/pull/11)1–#113, #116

- **US-18 — Consultar los defectos de un proyecto** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** usuario del sistema, **quiero** consultar los defectos registrados en un proyecto, **para** conocer los problemas detectados y su situación actual.
  [issue #46](https://github.com/valerubio7/software-metrics-and-estimation/issues/46) · [change](../../openspec/changes/archive/2026-11-03-us-18-consultar-defectos-proyecto) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/136)

- **TUI-09 — Gestionar defectos desde la TUI** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** integrante del equipo, **quiero** registrar, consultar y actualizar defectos desde la TUI, **para** hacer el seguimiento de los problemas sin usar la API directamente.
  [issue #102](https://github.com/valerubio7/software-metrics-and-estimation/issues/102) · [PR #12](https://github.com/valerubio7/software-metrics-and-estimation/pull/12)9–#135

### EPIC-07 — Métricas

- **US-19 — Consultar las métricas de un proyecto o Sprint** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** usuario del sistema, **quiero** consultar las métricas de un proyecto o Sprint, **para** evaluar objetivamente su avance, esfuerzo y calidad.
  [issue #47](https://github.com/valerubio7/software-metrics-and-estimation/issues/47) · [change](../../openspec/changes/archive/2026-10-26-us-19-consultar-metricas-proyecto-sprint) · [PR #12](https://github.com/valerubio7/software-metrics-and-estimation/pull/12)0–#123, #125

- **TUI-10 — Consultar métricas desde la TUI** · Sprint 3 — Funcionalidad y Calidad · En progreso
  **Como** usuario del sistema, **quiero** consultar las métricas de un proyecto o Sprint desde la TUI, **para** evaluar el avance y la calidad sin usar la API directamente.
  [issue #103](https://github.com/valerubio7/software-metrics-and-estimation/issues/103) · [PR #115](https://github.com/valerubio7/software-metrics-and-estimation/pull/115) (fix #124)

### EPIC-08 — Dashboard

- **US-20 — Visualizar el Dashboard de un proyecto** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** usuario del sistema, **quiero** visualizar un Dashboard del proyecto en la TUI, **para** obtener rápidamente una visión general de su estado y de sus principales métricas.
  [issue #48](https://github.com/valerubio7/software-metrics-and-estimation/issues/48) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/137)

- **TUI-01 — Abrir TUI y navegar** · Sprint 2 — Interfaz Objetivo · Completada
  **Como** usuario del sistema, **quiero** abrir la TUI y navegar entre proyectos/sprints/salir, **para** operar el sistema sin usar curl.
  [issue #83](https://github.com/valerubio7/software-metrics-and-estimation/issues/83) · [change](../../openspec/changes/archive/2026-10-10-tui-01-open-navigate) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/88)

- **TUI-02 — Gestionar proyectos desde TUI** · Sprint 2 — Interfaz Objetivo · Completada
  **Como** usuario del sistema, **quiero** crear, modificar y consultar el estado de proyectos desde la TUI, **para** administrar el proyecto sin llamar a la API directamente.
  [issue #84](https://github.com/valerubio7/software-metrics-and-estimation/issues/84) · [change](../../openspec/changes/archive/2026-10-13-tui-02-gestionar-proyectos) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/92)

- **TUI-03 — Gestionar backlog desde TUI** · Sprint 2 — Interfaz Objetivo · Completada
  **Como** integrante del equipo, **quiero** crear, modificar y listar historias del Product Backlog desde la TUI, **para** mantener el backlog sin usar la API directamente.
  [issue #85](https://github.com/valerubio7/software-metrics-and-estimation/issues/85) · [change](../../openspec/changes/archive/2026-10-14-tui-03-gestionar-backlog) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/95)

- **TUI-04 — Gestionar Sprint I: operar desde TUI** · Sprint 2 — Interfaz Objetivo · Completada
  **Como** integrante del equipo, **quiero** crear sprints, asignar historias, descomponer en tareas y completar historias desde la TUI, **para** ejecutar la iteración sin usar la API directamente.
  [issue #86](https://github.com/valerubio7/software-metrics-and-estimation/issues/86) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/96)

- **TUI-05 — Gestionar Sprint II: cerrar e historial desde TUI** · Sprint 2 — Interfaz Objetivo · Completada
  **Como** usuario del sistema, **quiero** cerrar Sprints y consultar Sprints anteriores desde la TUI, **para** finalizar formalmente las iteraciones y revisar su historia.
  [issue #87](https://github.com/valerubio7/software-metrics-and-estimation/issues/87) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/97)

- **TUI-11 — Visualizar el Dashboard desde la TUI** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** usuario del sistema, **quiero** visualizar el Dashboard del proyecto desde la TUI, **para** obtener rápidamente una visión general de su estado y de sus principales métricas.
  [issue #104](https://github.com/valerubio7/software-metrics-and-estimation/issues/104) · [change](../../openspec/changes/archive/2026-11-03-tui-11-visualizar-dashboard) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/137)

- **TUI-06 — Extender la navegación a las secciones del Sprint 3** · Sprint 3 — Funcionalidad y Calidad · Completada
  **Como** usuario del sistema, **quiero** acceder a las nuevas secciones del Sprint 3 desde el menú y elegir integrantes, historias, tareas y Sprints con el mismo patrón de navegación, **para** operar el sistema sin escribir identificadores ni aprender interacciones distintas.
  [issue #105](https://github.com/valerubio7/software-metrics-and-estimation/issues/105) · [change](../../openspec/changes/archive/2026-10-25-tui-06-navegacion-sprint-3) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/106)

- **TUI-12 — Rediseñar la experiencia de usuario de la TUI** · (sin sprint) · Completada
  **Como** usuario del sistema, **quiero** que la TUI me reciba en el Dashboard del proyecto que elijo y que todas las secciones cuelguen de ese proyecto, **para** no perderme en un menú plano ni volver a elegir el proyecto en cada flujo.
  [issue #140](https://github.com/valerubio7/software-metrics-and-estimation/issues/140) · [change](../../openspec/changes/archive/2026-11-03-tui-12-rediseno-ux) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/141)

### EPIC-09 — Reportes

- **US-21 — Generar un reporte PDF de un proyecto o Sprint** · Sprint 4 — Cierre y la Entrega Final · Completada
  **Como** usuario del sistema, **quiero** generar un reporte en formato PDF de un proyecto o Sprint, **para** disponer de un documento consolidado con la información relevante de su planificación, seguimiento y calidad.
  [issue #49](https://github.com/valerubio7/software-metrics-and-estimation/issues/49) · [change](../../openspec/changes/archive/2026-11-03-us-21-reporte-pdf-proyecto-sprint) · [PR](https://github.com/valerubio7/software-metrics-and-estimation/pull/138)

## Estado del tablero

| Situación | Detalle | Acción sugerida |
|---|---|---|
| Fechas de Sprint 3 y 4 | Sprint 3 figura con 29 días (19/10–17/11) y Sprint 4 arranca el 17/11, pero el proyecto cerró el 05/11. | Ajustar Sprint 3 a 19/10–01/11 y Sprint 4 a 02/11–05/11. |
| TUI-10 (#103) | Está «In progress», pero se entregó el 30/10 ([PR #115](https://github.com/valerubio7/software-metrics-and-estimation/pull/115), corregida en #124). | Mover a Done. |
| TUI-12 (#140) | Completada, sin Sprint asignado. | Asignar al Sprint 4. |
| Prioridad y estimación | Las 33 historias tienen Priority, Size y Estimate vacíos. | Cargarlos para poder ordenar por valor y planificar capacidad. |

## Trazabilidad SDD

Todas las historias enlazan su change, salvo estas excepciones:

* **US-14** (Planning Poker) y **US-17** (ciclo de vida del defecto): se
  implementaron y tienen pruebas, pero no tienen change archivado en
  `openspec/changes/archive/`.
* **TUI-04**, **TUI-05**, **TUI-07**, **TUI-09** y **TUI-10**: no tienen change
  propio; su especificación vive en `openspec/specs/tui/spec.md`.

## Mantenimiento

1. Toda historia nueva entra como issue y se agrega al tablero con épica, Sprint
   y prioridad.
2. Al cerrar un Sprint, se actualiza el estado y se archiva el change de
   OpenSpec.
3. Este documento se regenera desde el tablero al cerrar cada Sprint; no se
   edita a mano para cambiar estados.

