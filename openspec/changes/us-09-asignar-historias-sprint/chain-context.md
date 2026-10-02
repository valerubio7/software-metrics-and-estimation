# Contexto de cadena de PRs — HU-09

## Alcance

Esta rama funciona como tracker de integración de HU-09. Los PRs hijos incorporan la implementación en cortes revisables; no fusionar este tracker a `main` hasta integrar y revisar toda la cadena.

## Orden previsto

1. Contrato de aplicación, pruebas unitarias y propuesta/especificación funcional.
2. Migraciones y repositorio transaccional.
3. Pruebas de integración PostgreSQL y fixtures.
4. Endpoint HTTP, composición, gating, pruebas y README.
5. Diseño, delta de Sprint y tareas SDD.
6. Progreso de aplicación y registro ODD.

Cada PR hijo apunta al PR inmediatamente anterior y debe mantenerse en un máximo de 400 líneas modificadas. Todos se vinculan con la issue aprobada #37. Esta cadena no autoriza la fusión a `main`.
