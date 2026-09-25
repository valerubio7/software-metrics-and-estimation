# Especificación de Proyecto

## Requisitos AÑADIDOS

### Requirement: Crear un proyecto mediante la API HTTP
El sistema DEBERÁ exponer `POST /projects` como una API HTTP para crear un proyecto. La solicitud DEBERÁ aceptar `name`, `start_date` y `planned_finish_date`. Cuando la creación sea exitosa, la respuesta DEBERÁ identificar el proyecto creado e incluir sus datos básicos almacenados.

#### Escenario: Crear un proyecto válido
- **GIVEN** una solicitud con `name`, `start_date` y `planned_finish_date`
- **AND** `planned_finish_date` es igual o posterior a `start_date`
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema crea y persiste el proyecto
- **AND** devuelve una respuesta de creación exitosa que contiene el identificador del proyecto y los datos básicos almacenados

### Requirement: El nombre del proyecto es obligatorio
El sistema DEBERÁ rechazar una solicitud de creación de proyecto que no proporcione un nombre.

#### Escenario: Rechazar una solicitud sin nombre
- **GIVEN** una solicitud sin `name` o con `name` en blanco
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que identifica el nombre faltante o inválido
- **AND** no se crea ningún proyecto

### Requirement: La fecha de inicio del proyecto es obligatoria
El sistema DEBERÁ rechazar una solicitud de creación de proyecto que no proporcione una fecha de inicio válida.

#### Escenario: Rechazar una solicitud sin fecha de inicio válida
- **GIVEN** una solicitud sin `start_date` o con `start_date` inválida
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que identifica la fecha de inicio faltante o inválida
- **AND** no se crea ningún proyecto

### Requirement: La fecha de finalización planificada del proyecto es obligatoria
El sistema DEBERÁ rechazar una solicitud de creación de proyecto que no proporcione una fecha de finalización planificada válida.

#### Escenario: Rechazar una solicitud sin fecha de finalización planificada válida
- **GIVEN** una solicitud sin `planned_finish_date` o con `planned_finish_date` inválida
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que identifica la fecha de finalización planificada faltante o inválida
- **AND** no se crea ningún proyecto

### Requirement: Las fechas del proyecto son consistentes
El sistema DEBERÁ rechazar una solicitud de creación de proyecto cuando `planned_finish_date` sea anterior a `start_date` y DEBERÁ informar ese motivo de consistencia de fechas.

#### Escenario: Rechazar una fecha de finalización planificada anterior a la fecha de inicio
- **GIVEN** una solicitud con `start_date` y `planned_finish_date`
- **AND** `planned_finish_date` es anterior a `start_date`
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que explica que la fecha de finalización planificada no puede ser anterior a la fecha de inicio
- **AND** no se crea ningún proyecto

### Requirement: Persistir proyectos en PostgreSQL
El sistema DEBERÁ persistir cada proyecto válido creado en PostgreSQL con su nombre, fecha de inicio y fecha de finalización planificada.

#### Escenario: Almacenar un proyecto válido en PostgreSQL
- **GIVEN** una solicitud válida de creación de proyecto
- **WHEN** el sistema crea el proyecto mediante la API HTTP
- **THEN** PostgreSQL contiene el nombre, la fecha de inicio y la fecha de finalización planificada del proyecto

### Requirement: El estado del proyecto está fuera de alcance para US-01
El sistema NO DEBERÁ introducir comportamiento de entrada, persistencia, cálculo o respuesta de estado del proyecto como parte de la creación de proyectos de US-01.

#### Escenario: Crear un proyecto sin estado
- **GIVEN** una solicitud válida de creación de proyecto sin información de estado
- **WHEN** el cliente envía `POST /projects`
- **THEN** la creación del proyecto se realiza correctamente sin requerir ni producir comportamiento de estado del proyecto
### Requirement: Reemplazar los datos básicos de un proyecto existente

El sistema MUST ofrecer `PUT /projects/{project_id}` para reemplazar la representación completa de los datos básicos del proyecto identificado. Una solicitud válida MUST incluir `name`, `start_date` y `planned_finish_date`. Si la operación tiene éxito, el sistema MUST persistir los valores proporcionados y devolver el identificador y los datos básicos del proyecto actualizado.

#### Scenario: Actualizar todos los campos básicos de un proyecto existente

- GIVEN un proyecto existente y una solicitud con `name` no vacío, `start_date` válida y `planned_finish_date` válida que no sea anterior a la fecha de inicio
- WHEN el cliente envía `PUT /projects/{project_id}` con el ID del proyecto
- THEN el sistema persiste los tres campos básicos proporcionados para ese proyecto
- AND devuelve una respuesta exitosa con el mismo ID del proyecto y los campos básicos actualizados

#### Scenario: Aceptar una fecha de finalización planificada igual a la fecha de inicio

- GIVEN un proyecto existente y una solicitud completa cuyo `planned_finish_date` es igual a su `start_date`
- WHEN el cliente envía `PUT /projects/{project_id}`
- THEN el sistema acepta y persiste los datos básicos actualizados

### Requirement: Rechazar datos incompletos o inválidos al actualizar un proyecto

El sistema MUST rechazar una solicitud de actualización si falta algún campo básico obligatorio o es inválido, MUST explicar el motivo de validación y MUST NOT persistir ninguna parte de la actualización rechazada.

#### Scenario: Rechazar una solicitud a la que le falte un campo básico obligatorio

- GIVEN un proyecto existente
- AND una solicitud de actualización omite `name`, `start_date` o `planned_finish_date`
- WHEN el cliente envía `PUT /projects/{project_id}`
- THEN el sistema devuelve un error de validación que identifica el campo faltante
- AND los datos básicos previamente almacenados del proyecto permanecen sin cambios

#### Scenario: Rechazar un nombre de proyecto vacío

- GIVEN un proyecto existente y una solicitud de actualización con `name` faltante o vacío
- WHEN el cliente envía `PUT /projects/{project_id}`
- THEN el sistema devuelve un error de validación que identifica el nombre inválido
- AND los datos básicos previamente almacenados del proyecto permanecen sin cambios

#### Scenario: Rechazar un valor de fecha inválido

- GIVEN un proyecto existente y una solicitud de actualización completa con `start_date` o `planned_finish_date` inválida
- WHEN el cliente envía `PUT /projects/{project_id}`
- THEN el sistema devuelve un error de validación que identifica la fecha inválida
- AND los datos básicos previamente almacenados del proyecto permanecen sin cambios

### Requirement: Aplicar la consistencia de fechas del proyecto durante las actualizaciones

El sistema MUST rechazar una actualización cuando `planned_finish_date` sea anterior a `start_date`, MUST explicar la inconsistencia de fechas y MUST NOT persistir ninguna parte de la actualización rechazada.

#### Scenario: Rechazar una fecha de finalización planificada anterior a la fecha de inicio

- GIVEN un proyecto existente y una solicitud de actualización completa cuyo `planned_finish_date` es anterior a `start_date`
- WHEN el cliente envía `PUT /projects/{project_id}`
- THEN el sistema devuelve un error de validación que explica que la fecha de finalización planificada no puede ser anterior a la fecha de inicio
- AND los datos básicos previamente almacenados del proyecto permanecen sin cambios

### Requirement: Informar si el proyecto no existe durante la actualización

El sistema MUST devolver una respuesta not found cuando el ID de `PUT /projects/{project_id}` no identifique un proyecto existente y MUST NOT crear un proyecto ni modificar los datos de ningún proyecto existente.

#### Scenario: Actualizar un proyecto con un ID desconocido

- GIVEN el ID del proyecto solicitado no existe
- WHEN el cliente envía `PUT /projects/{project_id}` con datos básicos válidos en los demás aspectos
- THEN el sistema devuelve una respuesta not found
- AND no se crea ningún proyecto ni se modifican los datos de proyectos existentes

### Requirement: Preservar los datos ajenos a los campos básicos del proyecto

Al actualizar un proyecto, el sistema MUST cambiar únicamente `name`, `start_date` y `planned_finish_date` de ese proyecto. MUST preservar el identificador del proyecto, los demás datos del proyecto y todos los datos de miembros e historias asociados al proyecto o no relacionados con él.

#### Scenario: Actualizar los datos básicos sin modificar los datos no relacionados

- GIVEN un proyecto existente con un identificador, otros datos de proyecto, miembros asociados e historias asociadas
- WHEN el cliente actualiza correctamente `name`, `start_date` y `planned_finish_date` del proyecto
- THEN el proyecto conserva el mismo identificador y todos los demás datos del proyecto
- AND todos los datos de miembros e historias asociados permanecen sin cambios
