# tui Specification

## Purpose
Permitir operar el sistema desde consola sin curl mediante una TUI base navegable que solo consume la API HTTP.

## Requirements

### Requirement: Abrir la TUI con configuración de conexión a la API

El sistema DEBERÁ permitir abrir la TUI configurando la conexión vía `API_URL` con default `http://localhost:8080`.

#### Scenario: Apertura con API_URL explícita

- **WHEN** el usuario abre la TUI con `API_URL` válida
- **THEN** la TUI arranca y usa esa URL como base de la API

#### Scenario: Apertura sin API_URL

- **WHEN** el usuario abre la TUI sin definir `API_URL`
- **THEN** la TUI usa `http://localhost:8080` por defecto

### Requirement: Mostrar menú base de navegación

La TUI DEBERÁ mostrar un menú con opciones proyectos, backlog, sprints y salir. La opción proyectos da acceso a la gestión de proyectos; las opciones backlog y sprints siguen sin pantallas funcionales.

#### Scenario: Menú inicial

- **WHEN** la TUI arranca correctamente
- **THEN** muestra las 4 opciones y permite moverse entre ellas

### Requirement: Informar API no disponible sin cierre abrupto

Si la API no está disponible la TUI DEBERÁ informar el motivo y seguir abierta.

#### Scenario: API caída

- **WHEN** la API no responde al abrir o navegar
- **THEN** la TUI muestra el motivo del fallo y no se cierra abruptamente

### Requirement: Salir limpiamente de la TUI

El usuario DEBERÁ poder salir limpiamente con `q` / `Ctrl+C` / opción salir.

#### Scenario: Salida limpia

- **WHEN** el usuario pide salir
- **THEN** la TUI termina con código 0 sin traza de error

### Requirement: Navegación de solo lectura

La navegación entre el menú y las pantallas DEBERÁ ser de solo lectura y NO modificará ningún dato del sistema. Solo las operaciones explícitas de creación y modificación de proyectos (Requirement: Operar proyectos mediante la TUI) pueden emitir requests que modifiquen datos.

#### Scenario: Navegar no muta

- **WHEN** el usuario navega entre el menú y las pantallas sin confirmar una operación de proyectos
- **THEN** no se emite ninguna request que modifique datos

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
