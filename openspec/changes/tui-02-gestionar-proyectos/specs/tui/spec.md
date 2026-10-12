# Especificación: gestión de proyectos en la TUI

## ADDED Requirements

### Requirement: Operar proyectos mediante la TUI
La TUI DEBERÁ permitir operar proyectos desde la opción correspondiente. La gestión consume la API existente y no incorpora lógica de negocio. La navegación de menú existente se conserva.

#### Scenario: Crear un proyecto
- **Dado** que el usuario completa nombre, fecha de inicio y fecha planificada de finalización
- **Cuando** confirma la creación
- **Entonces** la TUI envía esos campos al endpoint HTTP de creación y presenta el resultado de la API

#### Scenario: La API rechaza datos de creación
- **Dado** que la API responde HTTP 422 con detalles de validación
- **Cuando** la TUI presenta el resultado
- **Entonces** muestra el motivo informado por la API y no comunica éxito

#### Scenario: Modificar un proyecto existente
- **Dado** que el usuario identifica un proyecto y completa sus datos básicos
- **Cuando** confirma la modificación
- **Entonces** la TUI envía los campos al endpoint HTTP de actualización sin duplicar validaciones de negocio

#### Scenario: Proyecto inexistente al modificar
- **Dado** que el identificador no corresponde a un proyecto existente
- **Cuando** la API responde HTTP 404
- **Entonces** la TUI informa que el proyecto no existe y presenta el detalle de la API

#### Scenario: Consultar el estado derivado
- **Dado** un identificador de proyecto válido
- **Cuando** el usuario consulta su estado
- **Entonces** la TUI muestra el estado informado por la API (`planned`, `active` o `overdue`) sin recalcularlo

#### Scenario: Proyecto inexistente al consultar
- **Dado** que el identificador no corresponde a un proyecto existente
- **Cuando** la API responde HTTP 404
- **Entonces** la TUI informa que el proyecto no existe y presenta el detalle de la API

#### Scenario: Error de transporte
- **Dado** que no se puede establecer conexión con la API
- **Cuando** falla una operación de proyectos
- **Entonces** la TUI informa el problema de conexión sin cerrarse abruptamente
