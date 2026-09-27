# Propuesta: Crear un Sprint y definir su Sprint Goal

## Intención

Permitir registrar un Sprint para un proyecto existente y guardar su Sprint Goal, de modo que quede disponible para futuras historias del Product Backlog. La creación debe rechazar datos obligatorios ausentes y no depende de consultar el backlog (US-07).

## Alcance

### Incluido
- Crear un Sprint asociado a un proyecto existente e identificarlo mediante un UUID generado por el servidor, siguiendo el patrón actual del repositorio.
- Registrar el Sprint Goal y validar la información obligatoria antes de persistir.
- Persistir y exponer la creación sin asignar historias.

### Excluido
- Asignar historias (US-09), registrar historias completadas (US-11), cerrar el Sprint (US-12) y consultar Sprints históricos (US-13).
- Añadir fechas, duración, estado, orden, identificador visible adicional u otras reglas no definidas en la aceptación de US-08.

## Capacidades

### Capacidades nuevas
- `sprint`: creación de un Sprint asociado a un proyecto existente, registro de su Sprint Goal e identidad del Sprint.

### Capacidades modificadas
- Ninguna. La creación de Sprint introduce requisitos propios; no modifica los requisitos canónicos de `project` ni de `historia`.

## Enfoque

Implementar un módulo vertical `sprint` con las capas de dominio, aplicación, transporte HTTP e infraestructura PostgreSQL, siguiendo el patrón de creación de historias. Usar una ruta anidada al proyecto (`POST /projects/{project_id}/sprints`), generar el UUID en el servidor y persistir la asociación con integridad referencial. Validar el Sprint Goal obligatorio antes de escribir; no depender de US-07 ni añadir reglas de contenido no acordadas. Añadir la migración y habilitar la ruta de forma compatible con el mecanismo de migraciones existente.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/sprint/` | Nueva | Dominio, caso de uso, endpoint HTTP y persistencia de Sprint. |
| `internal/api/api.go`, `cmd/api/main.go` | Modificada | Componer la ruta de creación y respetar la disponibilidad de su esquema. |
| `internal/project/infrastructure/postgres/migrations/` | Nueva | Migración para almacenar Sprints vinculados a proyectos existentes. |
| `tests/unit/`, `tests/integration/` | Modificada | Verificar validaciones, creación, asociación y persistencia. |
| `README.md` | Modificada | Documentar la operación y su requisito de migración. |
| `openspec/changes/us-08-create-sprint/specs/sprint/spec.md` | Nueva en fase de specs | Especificar los requisitos de creación de Sprint. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| La nueva migración y su habilitación podrían afectar el arranque o las rutas actuales. | Media | Probar composición con esquema actualizado y verificar que los endpoints existentes conserven su comportamiento. |
| La implementación vertical podría acercarse o superar el presupuesto de revisión de 400 líneas. | Media | En `sdd-tasks`, estimar el tamaño y, por `ask-on-risk`, resolver el plan de entrega antes de implementar si se proyecta excederlo. |

## Plan de reversión

Revertir en conjunto la ruta, composición, módulo, pruebas y documentación introducidos. Si la migración ya se aplicó, no eliminar datos de Sprints automáticamente: deshabilitar la operación y conservar el esquema/datos hasta acordar una reversión de base de datos segura.

## Dependencias

- Debe existir el proyecto cuyo identificador se proporciona; la asociación persistida debe conservar integridad referencial.
- Aplicar la nueva migración antes de habilitar la creación de Sprints.
- No hay dependencia funcional de US-07.

## Criterios de éxito

- [ ] Una solicitud válida crea y persiste exactamente un Sprint vinculado al proyecto indicado, con UUID generado y Sprint Goal registrado.
- [ ] Si falta información obligatoria, la solicitud se rechaza y no se persiste ningún Sprint.
- [ ] La creación no consulta ni asigna historias y la ruta solo queda disponible cuando su esquema está preparado.
