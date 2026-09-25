# Propuesta: Modificar un proyecto existente

## Intención

Implementar US-02 (issue #30 de GitHub) para que un cliente que ya conoce el ID de un proyecto pueda actualizar su nombre, fecha de inicio y fecha de finalización planificada. Las actualizaciones deben aplicar la invariante de fechas existente, persistir los cambios válidos y mantener sin cambios todos los datos no relacionados del proyecto, sus miembros y sus historias.

## Alcance

### Incluido
- Agregar `PUT /projects/{project_id}` para reemplazar la representación completa de los datos básicos (`name`, `start_date`, `planned_finish_date`) del proyecto identificado.
- Rechazar datos inválidos o incompletos, incluida una fecha de finalización planificada anterior a la fecha de inicio; informar el motivo y no persistir los cambios inválidos.
- Persistir únicamente los tres campos básicos del proyecto seleccionado cuando los cambios sean válidos; devolver not found si el ID del proyecto es desconocido.
- Cubrir la validación, el comportamiento HTTP, la persistencia y la preservación de datos no relacionados mediante pruebas unitarias y de integración.

### Fuera de alcance
- Listado, búsqueda o selección interactiva de proyectos; quien realiza la llamada ya dispone del ID.
- Creación de proyectos (US-01), gestión de miembros (US-03) y estado del proyecto (US-04).
- Semántica de actualización parcial (`PATCH`) o cambios en los modelos de datos de proyectos, historias o miembros.

## Capacidades

### Capacidades nuevas
- Ninguna.

### Capacidades modificadas
- `project`: agregar el comportamiento de actualización de los datos básicos de un proyecto existente y preservar los requisitos actuales de creación de US-01. La fase de especificación debe crear un delta para `openspec/specs/project/spec.md`.

## Enfoque

Respetar los límites existentes de Go entre HTTP, aplicación, dominio y PostgreSQL. Implementar `PUT /projects/{project_id}` como reemplazo completo de los tres campos básicos obligatorios, validar el estado resultante completo antes de persistir y actualizar únicamente `name`, `start_date` y `planned_finish_date` para el ID de la ruta. Usar la respuesta convencional not found para un ID desconocido. Mantener informativas las respuestas de error y asegurar que las entradas inválidas nunca lleguen a persistencia. Seguir TDD estricto con pruebas unitarias/HTTP específicas y cobertura de integración con PostgreSQL; ejecutar `go test ./...` (las pruebas de integración requieren Docker/Testcontainers). Mantener la implementación revisable dentro del límite configurado de 400 líneas modificadas en un único PR.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|--------|-------------|
| `internal/api/api.go` | Modificado | Registrar la ruta de actualización del proyecto. |
| `internal/project/domain/` y `internal/project/application/` | Modificado | Reutilizar las invariantes del proyecto y agregar el caso de uso/contrato del repositorio para actualizarlo. |
| `internal/project/transport/http/handler.go` | Modificado | Decodificar y gestionar las solicitudes y respuestas de actualización. |
| `internal/project/infrastructure/postgres/repository.go` | Modificado | Actualizar únicamente las tres columnas de datos básicos y detectar si falta el proyecto. |
| `tests/unit/project/` | Modificado | Cubrir actualizaciones válidas, datos inválidos, IDs inexistentes y ausencia de persistencia cuando falla la validación. |
| `tests/integration/project/postgres/repository_integration_test.go` | Modificado | Verificar los campos actualizados y la preservación del ID del proyecto y de los datos relacionados y no relacionados. |
| `openspec/specs/project/spec.md` | Se modificará en la futura fase de especificación/archivo | Definir el comportamiento de US-02 como delta sin eliminar los requisitos de US-01. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|------|------------|------------|
| Una actualización amplia podría sobrescribir datos no relacionados. | Media | Limitar la sentencia de actualización a los tres campos básicos y comprobar su preservación en pruebas de integración. |
| El `PUT` de reemplazo completo exige que los clientes envíen los tres campos, incluso al cambiar uno solo. | Baja | Dejar explícito el comportamiento de campos obligatorios/reemplazo completo en el contrato de la API y en los errores de validación. |
| La verificación de integración con PostgreSQL podría no estar disponible sin Docker. | Media | Ejecutar las pruebas unitarias de todos modos; ejecutar e informar `go test ./...` con Docker/Testcontainers cuando esté disponible. |
| El cambio en un solo PR podría superar el límite de revisión de 400 líneas. | Baja | Limitar el alcance a US-02 y monitorear las adiciones y eliminaciones propias durante la implementación; no agregar flujos no relacionados. |

## Plan de reversión

Revertir el único PR de implementación. Este cambio no requiere una migración de esquema; revertir la ruta, el caso de uso, la actualización del repositorio y las pruebas restaura el comportamiento anterior, limitado a la creación. Los registros de proyectos ya persistidos siguen siendo válidos y la reversión no los elimina.

## Dependencias

- Tabla de proyectos de PostgreSQL e invariante de fechas del proyecto existentes; no se prevé una dependencia externa ni una migración de esquema.
- Se requiere Docker para ejecutar las pruebas de integración basadas en Testcontainers.

## Criterios de éxito

- [ ] Un cliente puede reemplazar por ID los tres campos básicos de un proyecto y la respuesta exitosa refleja los valores persistidos.
- [ ] Los IDs desconocidos devuelven not found; los campos faltantes/inválidos y `planned_finish_date < start_date` devuelven un error explicativo y dejan sin cambios los valores persistidos.
- [ ] La actualización de un proyecto preserva su ID y todos los datos no relacionados de proyectos, miembros e historias.
- [ ] `go test ./...` pasa, incluidas las pruebas de integración con PostgreSQL cuando Docker está disponible.
- [ ] La implementación se entrega en un PR con no más de 400 líneas modificadas de autoría.
