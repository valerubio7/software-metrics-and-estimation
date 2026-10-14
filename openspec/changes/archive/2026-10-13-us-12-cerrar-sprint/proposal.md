# Propuesta

## Por qué

El esquema ya registra `sprints.is_closed`, y la asignación de historias y su finalización respetan ese estado, pero el sistema todavía no ofrece una operación para cerrar un Sprint. US-12 completa el ciclo de vida solicitado y mantiene disponibles los resultados históricos de la iteración.

## Qué cambia

- Agregar una operación para cerrar un Sprint abierto dentro del proyecto indicado.
- Rechazar solicitudes inválidas, recursos inexistentes y el cierre repetido sin modificar datos indebidamente.
- Persistir el cierre sin borrar ni alterar asociaciones, finalizaciones, tareas, esfuerzos o defectos existentes.
- Impedir que una asignación concurrente posterior al cierre incorpore historias.

## Capacidades

### Capacidades nuevas

Ninguna.

### Capacidades modificadas

- `sprint`: agregar el contrato de cierre de Sprint y conservación de su historial.

## Impacto

Afecta la API HTTP de Sprints, la capa de aplicación, el repositorio PostgreSQL, la composición de dependencias condicionada por la versión del esquema y las pruebas unitarias/de integración. Usa la columna existente `sprints.is_closed` (migración `000008`); no requiere una migración nueva.
