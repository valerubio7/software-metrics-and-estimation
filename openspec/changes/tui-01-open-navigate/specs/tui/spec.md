# Spec Delta

## Purpose

Permitir operar el sistema desde consola sin curl mediante una TUI base navegable que solo consume la API HTTP.

## ADDED Requirements

### Requirement: Abrir la TUI con configuración de conexión a la API

El sistema DEBERÁ permitir abrir la TUI configurando la conexión vía `API_URL` con default `http://localhost:8080`.

#### Scenario: Apertura con API_URL explícita

- **WHEN** el usuario abre la TUI con `API_URL` válida
- **THEN** la TUI arranca y usa esa URL como base de la API

#### Scenario: Apertura sin API_URL

- **WHEN** el usuario abre la TUI sin definir `API_URL`
- **THEN** la TUI usa `http://localhost:8080` por defecto

### Requirement: Mostrar menú base de navegación

La TUI DEBERÁ mostrar un menú base con opciones proyectos, backlog, sprints y salir, sin pantallas funcionales.

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

La navegación DEBERÁ ser de solo lectura y NO modificará ningún dato del sistema.

#### Scenario: Navegar no muta

- **WHEN** el usuario navega entre menú y placeholders
- **THEN** no se emite ninguna request que modifique datos
