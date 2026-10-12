# Tareas

## 1. Caso de uso y persistencia

- [x] 1.1 Agregar la prueba unitaria del comando de cierre para verificar delegación de IDs y propagación de errores; comprobar el comportamiento ejecutando `go test ./tests/unit/sprint/application`.
- [x] 1.2 Implementar comando, errores tipados, puerto de cierre y caso de uso; verificar las pruebas unitarias de aplicación.
- [x] 1.3 Agregar y ejecutar pruebas PostgreSQL de cierre exitoso, proyecto/Sprint inexistentes, Sprint de otro proyecto, cierre repetido, preservación de asociaciones/finalizaciones/tareas y serialización frente a asignación concurrente; verificar estado y datos por la interfaz pública del repositorio.
- [x] 1.4 Implementar el cierre transaccional del Sprint con bloqueo `FOR UPDATE` compatible con asignación y finalización; ejecutar las pruebas de integración relevantes.

## 2. Endpoint HTTP y disponibilidad

- [x] 2.1 Agregar pruebas del contrato HTTP para éxito, UUID inválido, proyecto/Sprint no encontrados, Sprint cerrado, método no admitido y fallo interno; afirmar código, cuerpo y llamadas al puerto.
- [x] 2.2 Implementar handler y registro de `POST /projects/{project_id}/sprints/{sprint_id}/closure`; verificar las pruebas HTTP.
- [x] 2.3 Agregar gate independiente para esquema limpio >=8 sin alterar rutas existentes; verificar con pruebas de disponibilidad de rutas en versiones límite, estado `dirty` y error de lectura.

## 3. Verificación y documentación

- [x] 3.1 Documentar endpoint, respuestas, errores y requisito de esquema; revisar que los ejemplos coincidan con el handler.
- [x] 3.2 Ejecutar `go test ./...` y revisar el diff para confirmar preservación de historial y ausencia de cambios fuera de alcance; la suite completa pasó.
