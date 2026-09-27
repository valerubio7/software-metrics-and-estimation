## Exploración: US-02 — Modificar un proyecto existente (`us-02-modify-project`)

### Estado actual
Actualmente, los proyectos solo admiten la creación: `POST /projects` se configura en `internal/api/api.go`, se valida en el dominio/caso de uso de proyectos y se inserta en PostgreSQL. La tabla `projects` almacena exactamente `id`, `name`, `start_date` y `planned_finish_date`; tanto la validación del dominio como una restricción de la base de datos exigen que la fecha de finalización planificada sea igual o posterior a la fecha de inicio. No existe una operación de consulta/listado ni de actualización de proyectos. El usuario confirmó que el cliente ya tendrá el ID del proyecto existente, por lo que US-02 no requiere un flujo de listado ni de selección interactiva de proyectos. Las historias hacen referencia a los proyectos mediante `ON DELETE RESTRICT`; las actualizaciones de los campos básicos del proyecto no deben involucrar registros de historias.

Las pruebas existentes de proyectos cubren la validación de creación y la inserción en PostgreSQL, no el comportamiento de actualización. El repositorio usa pruebas unitarias con `httptest` y pruebas de integración con Testcontainers; estas últimas requieren Docker y se omiten si no está disponible. La especificación de proyectos de OpenSpec describe actualmente la creación de US-01, por lo que este cambio debe agregar un requisito específico para la actualización de proyectos, sin cambiar el comportamiento de creación ni agregar la gestión de miembros.

### Áreas afectadas
- `internal/api/api.go` — registrar una ruta de actualización junto a `POST /projects` en la composición actual de `net/http`.
- `internal/project/domain/project.go` — se pueden reutilizar o factorizar las invariantes existentes de los campos básicos para validar las actualizaciones.
- `internal/project/application/create_project.go` — contiene los contratos del repositorio específicos de creación; la actualización requiere una operación de aplicación y un límite de persistencia.
- `internal/project/transport/http/handler.go` — actualmente solo decodifica y gestiona solicitudes de creación de proyectos.
- `internal/project/infrastructure/postgres/repository.go` — actualmente solo implementa `INSERT`; las actualizaciones deben afectar únicamente las tres columnas de datos básicos.
- `tests/unit/project/{domain,application,transport/http}/` — agregar cobertura de validación, de ausencia de llamadas a persistencia ante entradas inválidas y del comportamiento HTTP, siguiendo los patrones actuales de pruebas.
- `tests/integration/project/postgres/repository_integration_test.go` — verificar que PostgreSQL cambie los campos básicos del proyecto seleccionado y preserve su ID y la información no relacionada de las historias dependientes.
- `openspec/specs/project/spec.md` y `specs/project/spec.md` del nuevo cambio — especificación de origen y futuro delta del comportamiento de actualización de proyectos.

### Alternativas
1. **Reemplazo completo de los datos básicos con `PUT /projects/{project_id}`** — identificar el proyecto mediante el ID de la ruta y exigir en el cuerpo la representación completa de los datos básicos (`name`, `start_date`, `planned_finish_date`).
   - Ventajas: validación inequívoca del estado resultante completo; se ajusta naturalmente a las reglas existentes de campos obligatorios y consistencia de fechas; una actualización SQL explícita puede preservar todas las demás columnas o registros relacionados.
   - Desventajas: los clientes deben enviar los tres campos incluso si cambian uno solo; es necesario definir la semántica de existencia del proyecto y de las respuestas de error.
   - Esfuerzo: Medio

2. **Actualización parcial con `PATCH /projects/{project_id}`** — aceptar solo los campos modificados y validar el proyecto resultante combinado.
   - Ventajas: los clientes pueden cambiar un campo básico sin reenviar los demás.
   - Desventajas: requiere definir la semántica de omitido frente a vacío y cargar los valores actuales antes de validar la consistencia de fechas; el repositorio actual no tiene una operación de lectura, por lo que se amplían el flujo y la superficie de pruebas.
   - Esfuerzo: Alto

### Recomendación
Se recomienda `PUT /projects/{project_id}` como reemplazo completo de la representación de los datos básicos del proyecto. Se ajusta al modelo de proyecto, actualmente pequeño, y garantiza que la invariante de fechas se compruebe contra el estado resultante completo. Implementar la persistencia como una actualización explícita únicamente de `name`, `start_date` y `planned_finish_date`; no modificar el ID, las filas de historias, los miembros ni ningún otro dato no relacionado. Considerar el ID de la ruta como el mecanismo para que un cliente de la API seleccione un proyecto; no existe un flujo de listado de proyectos o selección de interfaz de usuario, y no debe agregarse implícitamente. La propuesta debe explicitar la respuesta para un proyecto inexistente y el contrato de actualización con campos obligatorios, sin exceder US-02.

### Riesgos
- El issue no define la respuesta cuando el ID de proyecto proporcionado no existe; la propuesta debe explicitar este comportamiento de la API.
- El issue no define la semántica de actualización para los campos omitidos; esto afecta el contrato de la API.
- Una actualización amplia del repositorio podría modificar accidentalmente datos no relacionados; limitar el SQL a los tres campos básicos indicados y cubrir su preservación en las pruebas.
- La verificación de integración con PostgreSQL depende de que Docker/Testcontainers estén disponibles; el comando de pruebas completo configurado es `go test ./...`.

### Listo para la propuesta
Sí. El usuario confirmó que el cliente ya tendrá el ID del proyecto, por lo que se propone un reemplazo completo de los datos básicos por ID y se define la respuesta para un proyecto inexistente. Mantener fuera de alcance el listado/selección interactiva de proyectos, la gestión de miembros de US-03 y el comportamiento del estado de US-04.
