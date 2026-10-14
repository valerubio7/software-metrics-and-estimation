# Especificación: gestión de proyectos en la TUI

## ADDED Requirements

### Requirement: Operar proyectos mediante la TUI
La TUI DEBERÁ permitir operar proyectos desde la opción correspondiente. La gestión consume la API existente y no incorpora lógica de negocio. La navegación de menú existente se conserva.

#### Scenario: Crear un proyecto
- **GIVEN** que el usuario completa nombre, fecha de inicio y fecha planificada de finalización
- **WHEN** confirma la creación
- **THEN** la TUI envía esos campos al endpoint HTTP de creación y presenta el resultado de la API

#### Scenario: La API rechaza datos de creación
- **GIVEN** que la API responde HTTP 422 con detalles de validación
- **WHEN** la TUI presenta el resultado
- **THEN** muestra el motivo informado por la API y no comunica éxito

#### Scenario: Modificar un proyecto existente
- **GIVEN** que el usuario identifica un proyecto y completa sus datos básicos
- **WHEN** confirma la modificación
- **THEN** la TUI envía los campos al endpoint HTTP de actualización sin duplicar validaciones de negocio

#### Scenario: Proyecto inexistente al modificar
- **GIVEN** que el identificador no corresponde a un proyecto existente
- **WHEN** la API responde HTTP 404
- **THEN** la TUI informa que el proyecto no existe y presenta el detalle de la API

#### Scenario: Consultar el estado derivado
- **GIVEN** un identificador de proyecto válido
- **WHEN** el usuario consulta su estado
- **THEN** la TUI muestra el estado informado por la API (`planned`, `active` o `overdue`) sin recalcularlo

#### Scenario: Proyecto inexistente al consultar
- **GIVEN** que el identificador no corresponde a un proyecto existente
- **WHEN** la API responde HTTP 404
- **THEN** la TUI informa que el proyecto no existe y presenta el detalle de la API

#### Scenario: Error de transporte
- **GIVEN** que no se puede establecer conexión con la API
- **WHEN** falla una operación de proyectos
- **THEN** la TUI informa el problema de conexión sin cerrarse abruptamente

## MODIFIED Requirements

### Requirement: Mostrar menú base de navegación
La TUI DEBERÁ mostrar un menú con opciones proyectos, backlog, sprints y salir. La opción proyectos da acceso a la gestión de proyectos; las opciones backlog y sprints siguen sin pantallas funcionales.

#### Scenario: Menú inicial
- **WHEN** la TUI arranca correctamente
- **THEN** muestra las 4 opciones y permite moverse entre ellas

### Requirement: Navegación de solo lectura
La navegación entre el menú y las pantallas DEBERÁ ser de solo lectura y NO modificará ningún dato del sistema. Solo las operaciones explícitas de creación y modificación de proyectos (Requirement: Operar proyectos mediante la TUI) pueden emitir requests que modifiquen datos.

#### Scenario: Navegar no muta
- **WHEN** el usuario navega entre el menú y las pantallas sin confirmar una operación de proyectos
- **THEN** no se emite ninguna request que modifique datos
