# Especificación: consulta de integrantes del proyecto

## ADDED Requirements

### Requirement: Consultar los integrantes de un proyecto
El sistema MUST ofrecer una operación de solo lectura, `GET /projects/{project_id}/members`, que reciba el identificador de un proyecto existente en la ruta. Ante una consulta válida, MUST responder `200` con un contenedor `{"project_id": "...", "members": [...]}`, donde `project_id` es el identificador consultado y `members` contiene todos los integrantes de ese proyecto y únicamente los de ese proyecto. Cada integrante MUST incluir `id`, `project_id`, `full_name` y `email`; `email` MUST ser `null` cuando no se registró. La lista MUST ordenarse por `full_name` ascendente y, ante nombres iguales, por `id` ascendente, para que la presentación sea determinista. Un proyecto sin integrantes MUST responder `200` con `members` igual a `[]`, nunca `null`. La operación MUST NOT modificar datos.

#### Scenario: Consultar un proyecto con integrantes
- GIVEN un proyecto existente con dos integrantes registrados
- WHEN se solicita `GET /projects/{project_id}/members` con el identificador de ese proyecto
- THEN el sistema responde `200`
- AND el cuerpo contiene `project_id` igual al identificador consultado
- AND `members` contiene exactamente dos elementos con `id`, `project_id`, `full_name` y `email`

#### Scenario: Proyecto sin integrantes
- GIVEN un proyecto existente sin integrantes
- WHEN se solicita `GET /projects/{project_id}/members`
- THEN el sistema responde `200`
- AND `members` es una lista vacía `[]`, no `null`

#### Scenario: Orden determinista por nombre e identificador
- GIVEN un proyecto con integrantes cuyos nombres producen empates
- WHEN se consulta la colección
- THEN los integrantes aparecen ordenados por `full_name` ascendente
- AND los empates se resuelven por `id` ascendente
- AND dos consultas consecutivas sin cambios devuelven el mismo orden

#### Scenario: Integrante sin email
- GIVEN un proyecto con un integrante registrado sin email
- WHEN se consulta la colección
- THEN ese integrante incluye la clave `email` con valor `null`

#### Scenario: Proyecto inexistente
- GIVEN un `project_id` con formato válido que no corresponde a ningún proyecto existente
- WHEN se solicita consultar sus integrantes
- THEN el sistema responde `404` con el código `project_not_found`
- AND no modifica datos

#### Scenario: Identificador inválido
- GIVEN un `project_id` que no es un UUID válido
- WHEN se solicita consultar sus integrantes
- THEN el sistema responde `422 validation_failed`
- AND `fields` identifica `project_id` como inválido
- AND no modifica datos

#### Scenario: Método no admitido
- GIVEN la ruta `/projects/{project_id}/members`
- WHEN se solicita con un método distinto de `GET` o `HEAD`
- THEN el sistema responde `405 method_not_allowed`
- AND no modifica datos

#### Scenario: Error inesperado del almacenamiento
- GIVEN un fallo inesperado al leer los integrantes
- WHEN se procesa la consulta
- THEN el sistema responde `500 internal_error` con un mensaje genérico
- AND no divulga detalles internos
- AND no modifica datos
