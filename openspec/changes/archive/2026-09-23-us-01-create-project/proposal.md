# Propuesta: US-01 Crear proyecto

## Origen

- Incidencia de GitHub: #29 — US-01 Crear proyecto
- Épica/hito: EPIC-01 — Gestión de Proyectos

## Problema

Los usuarios necesitan crear un proyecto con la información básica mínima requerida para comenzar a planificar, realizar seguimiento y medir el trabajo.

## Historia de Usuario

Como usuario del sistema, quiero crear un proyecto registrando su información básica para poder comenzar a gestionar su planificación, seguimiento y medición.

## Alcance Propuesto

Implementar el primer corte vertical para la creación de proyectos:

- Aceptar un nombre de proyecto.
- Aceptar una fecha de inicio.
- Aceptar una fecha de finalización planificada.
- Rechazar datos obligatorios faltantes.
- Rechazar una fecha de finalización planificada anterior a la fecha de inicio.
- Devolver errores de validación accionables.
- Persistir un proyecto válido mediante una abstracción de repositorio de proyectos.

## Primer Corte Recomendado

Usar una API HTTP como límite orientado al usuario:

- `POST /projects`
- Campos de la solicitud:
  - `name`
  - `start_date`
  - `planned_finish_date`
- Respuesta en caso de éxito:
  - identificador del proyecto creado
  - datos básicos almacenados del proyecto
- Respuesta ante error de validación:
  - información de error por campo o específica del motivo

La persistencia debe implementarse mediante una abstracción `ProjectRepository` respaldada por PostgreSQL, porque el equipo ya seleccionó psql/PostgreSQL como dirección de base de datos. Las pruebas todavía pueden usar fakes o dobles de prueba cuando corresponda, pero el objetivo de implementación es PostgreSQL en lugar de un almacén de productos en memoria.

## Decisiones de Producto Ya Tomadas

- El estado de un proyecto no forma parte de US-01 y permanece diferido hasta US-04.
- La fecha de finalización planificada es obligatoria para US-01.
- Modificar proyectos, registrar miembros y consultar el estado de un proyecto están explícitamente fuera de alcance.

## Decisiones Confirmadas

1. El primer límite demostrable para esta historia es un endpoint de API HTTP en lugar de solo un caso de uso interno de Go.
2. PostgreSQL es la tecnología de base de datos esperada para la persistencia.

## No Objetivos

- Actualizar un proyecto existente.
- Registrar miembros del proyecto.
- Calcular o mostrar el estado del proyecto.
- Reevaluar la tecnología final de base de datos.
- Construir comportamiento de paneles, métricas, informes, backlog, sprint, defectos, esfuerzo o estimación.

## Criterios de Éxito

- Se puede crear un proyecto válido y observarlo posteriormente a través del límite o repositorio implementado.
- Una entrada no válida no crea un proyecto.
- Los errores de validación explican el motivo.
- Los criterios de aceptación existentes de la incidencia #29 están cubiertos por pruebas.
