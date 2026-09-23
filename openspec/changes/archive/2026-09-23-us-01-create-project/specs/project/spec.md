# Especificación de Proyecto

## Requisitos AÑADIDOS

### Requisito: Crear un proyecto mediante la API HTTP
El sistema DEBERÁ exponer `POST /projects` como una API HTTP para crear un proyecto. La solicitud DEBERÁ aceptar `name`, `start_date` y `planned_finish_date`. Cuando la creación sea exitosa, la respuesta DEBERÁ identificar el proyecto creado e incluir sus datos básicos almacenados.

#### Escenario: Crear un proyecto válido
- **GIVEN** una solicitud con `name`, `start_date` y `planned_finish_date`
- **AND** `planned_finish_date` es igual o posterior a `start_date`
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema crea y persiste el proyecto
- **AND** devuelve una respuesta de creación exitosa que contiene el identificador del proyecto y los datos básicos almacenados

### Requisito: El nombre del proyecto es obligatorio
El sistema DEBERÁ rechazar una solicitud de creación de proyecto que no proporcione un nombre.

#### Escenario: Rechazar una solicitud sin nombre
- **GIVEN** una solicitud sin `name` o con `name` en blanco
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que identifica el nombre faltante o inválido
- **AND** no se crea ningún proyecto

### Requisito: La fecha de inicio del proyecto es obligatoria
El sistema DEBERÁ rechazar una solicitud de creación de proyecto que no proporcione una fecha de inicio válida.

#### Escenario: Rechazar una solicitud sin fecha de inicio válida
- **GIVEN** una solicitud sin `start_date` o con `start_date` inválida
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que identifica la fecha de inicio faltante o inválida
- **AND** no se crea ningún proyecto

### Requisito: La fecha de finalización planificada del proyecto es obligatoria
El sistema DEBERÁ rechazar una solicitud de creación de proyecto que no proporcione una fecha de finalización planificada válida.

#### Escenario: Rechazar una solicitud sin fecha de finalización planificada válida
- **GIVEN** una solicitud sin `planned_finish_date` o con `planned_finish_date` inválida
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que identifica la fecha de finalización planificada faltante o inválida
- **AND** no se crea ningún proyecto

### Requisito: Las fechas del proyecto son consistentes
El sistema DEBERÁ rechazar una solicitud de creación de proyecto cuando `planned_finish_date` sea anterior a `start_date` y DEBERÁ informar ese motivo de consistencia de fechas.

#### Escenario: Rechazar una fecha de finalización planificada anterior a la fecha de inicio
- **GIVEN** una solicitud con `start_date` y `planned_finish_date`
- **AND** `planned_finish_date` es anterior a `start_date`
- **WHEN** el cliente envía `POST /projects`
- **THEN** el sistema devuelve un error de validación que explica que la fecha de finalización planificada no puede ser anterior a la fecha de inicio
- **AND** no se crea ningún proyecto

### Requisito: Persistir proyectos en PostgreSQL
El sistema DEBERÁ persistir cada proyecto válido creado en PostgreSQL con su nombre, fecha de inicio y fecha de finalización planificada.

#### Escenario: Almacenar un proyecto válido en PostgreSQL
- **GIVEN** una solicitud válida de creación de proyecto
- **WHEN** el sistema crea el proyecto mediante la API HTTP
- **THEN** PostgreSQL contiene el nombre, la fecha de inicio y la fecha de finalización planificada del proyecto

### Requisito: El estado del proyecto está fuera de alcance para US-01
El sistema NO DEBERÁ introducir comportamiento de entrada, persistencia, cálculo o respuesta de estado del proyecto como parte de la creación de proyectos de US-01.

#### Escenario: Crear un proyecto sin estado
- **GIVEN** una solicitud válida de creación de proyecto sin información de estado
- **WHEN** el cliente envía `POST /projects`
- **THEN** la creación del proyecto se realiza correctamente sin requerir ni producir comportamiento de estado del proyecto
