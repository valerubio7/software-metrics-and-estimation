# Delta para Proyecto

## ADDED Requirements

### Requirement: Preservar los datos básicos del proyecto al registrar integrantes

Al registrar integrantes, el sistema MUST conservar sin cambios el nombre, las fechas y los demás datos básicos del proyecto al que se asocian.

#### Scenario: Registrar integrantes sin alterar el proyecto

- GIVEN un proyecto existente con datos básicos almacenados
- WHEN el cliente registra correctamente uno o más integrantes para ese proyecto
- THEN los integrantes quedan asociados al proyecto
- AND los datos básicos previamente almacenados del proyecto permanecen sin cambios
