# Especificación: TUI

## Purpose
Permitir operar el sistema desde consola sin curl mediante una TUI navegable que consume la API HTTP.

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

La TUI DEBERÁ mostrar un menú base con opciones proyectos, backlog, sprints y salir.

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

### Requirement: Gestionar proyectos desde la TUI

La TUI DEBERÁ permitir crear proyectos (nombre, fecha de inicio, fecha planificada de finalización), modificar sus datos básicos y consultar su estado derivado (`planned`, `active`, `overdue`) usando los endpoints HTTP existentes. La TUI NO DEBERÁ duplicar validaciones de dominio ni calcular estado; deberá presentar errores 422 y 404 con el motivo informado por la API.

#### Scenario: Crear un proyecto
- **WHEN** el usuario completa el formulario y confirma la creación
- **THEN** la TUI envía los campos a `POST /projects` y presenta el resultado de la API

#### Scenario: Mostrar validación de creación
- **WHEN** `POST /projects` responde 422 con detalles de validación
- **THEN** la TUI muestra esos detalles y no comunica éxito

#### Scenario: Modificar un proyecto
- **WHEN** el usuario proporciona el identificador y los datos básicos y confirma
- **THEN** la TUI envía los campos a `PUT /projects/{project_id}` sin validación de dominio local

#### Scenario: Informar proyecto inexistente
- **WHEN** una operación de modificación o consulta recibe 404
- **THEN** la TUI informa que el proyecto no existe y presenta el detalle de la API

#### Scenario: Consultar estado
- **WHEN** el usuario consulta el estado de un proyecto
- **THEN** la TUI solicita `GET /projects/{project_id}` y presenta el estado devuelto por el servidor sin recalcularlo

#### Scenario: Error de transporte
- **WHEN** una operación no logra conectarse con la API
- **THEN** la TUI informa el problema de conexión sin cerrarse abruptamente
