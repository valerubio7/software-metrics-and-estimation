# Tareas: US-01 Crear proyecto

## 0. Decisiones y preparación mínima

- [x] Decidir y documentar el conjunto mínimo de dependencias y enfoques: `net/http` para `POST /projects`, `github.com/google/uuid` para UUID, `pgx/v5` para PostgreSQL y migraciones SQL versionadas. La migración la ejecuta explícitamente el entorno local o de despliegue mediante la CLI `golang-migrate`; no se ejecuta automáticamente al iniciar la API ni se instala una herramienta de migración en el repositorio.
- [x] Agregar únicamente las dependencias aprobadas al módulo Go y comprobar que el proyecto continúa compilando con `go test ./...`.
- [x] Definir el contrato de composición en `cmd/api`: `DATABASE_URL` obligatorio, `HTTP_ADDR` opcional con valor predeterminado `:8080`, construcción de `pgxpool.Pool` y `PostgresProjectRepository`, generador UUID, construcción del caso de uso y registro de `POST /projects`, sin reglas de negocio ni SQL en el proceso HTTP.

## 1. Dominio y caso de uso (TDD)

- [x] **RED:** añadir pruebas unitarias para `Project`/`CreateProjectUseCase` que describan la creación válida, UUID asignado y llamada única a `ProjectRepository.Create` usando un fake de repositorio.
- [x] **RED:** añadir pruebas unitarias que demuestren que nombre ausente o en blanco, `start_date` ausente o inválida, `planned_finish_date` ausente o inválida, y fecha final anterior a la inicial son rechazados sin invocar el repositorio.
- [x] **GREEN:** implementar `Project`, `CreateProjectCommand`, errores de validación y la invariante de fechas; mantener `id`, `name`, `start_date` y `planned_finish_date` como único estado del dominio de US-01.
- [x] **GREEN:** definir `ProjectRepository` y completar `CreateProjectUseCase` con inyección del repositorio y del generador de UUID para persistir exclusivamente proyectos válidos.
- [x] **TRIANGULATE:** ampliar las pruebas del caso de uso para una fecha final igual a la inicial y para un error de `ProjectRepository`, verificando que el error de infraestructura se propaga sin transformarse en error de validación.
- [x] **REFACTOR:** simplificar nombres y helpers de pruebas del dominio/caso de uso sin cambiar el contrato ni introducir abstracciones compartidas prematuras.
- [x] Ejecutar `go test ./...` y registrar la evidencia RED/GREEN/TRIANGULATE/REFACTOR de este bloque.

## 2. Límite HTTP (TDD)

- [x] **RED:** añadir pruebas `httptest` para `POST /projects` válido que esperen `201 Created`, el UUID y los datos persistidos en JSON, sin campo `status`.
- [x] **RED:** añadir pruebas `httptest` para JSON malformado, cuerpo vacío o tipos incompatibles que esperen `400 Bad Request` y un error de solicitud accionable.
- [x] **RED:** añadir pruebas `httptest` para nombre y fechas ausentes o inválidas, y para fecha final anterior a la inicial, que esperen `422 Unprocessable Content`, errores accionables por campo o regla, y ausencia de persistencia.
- [x] **GREEN:** implementar DTOs, decodificación y serialización del handler `POST /projects`; mapear solo errores de protocolo a `400`, validación de dominio a `422` e infraestructura inesperada a `500` sin detalles internos.
- [x] **GREEN:** registrar la ruta HTTP elegida y conectar el handler al caso de uso mediante inyección de dependencias.
- [x] **TRIANGULATE:** verificar con pruebas que una fecha final igual a la inicial devuelve `201` y que un fallo inesperado del repositorio devuelve `500` sin filtrar su detalle.
- [x] **REFACTOR:** eliminar duplicación en fixtures y aserciones HTTP conservando los casos observables y sin mover validación de negocio al handler.
- [x] Ejecutar `go test ./...` y registrar la evidencia RED/GREEN/TRIANGULATE/REFACTOR de este bloque.

## 3. PostgreSQL y migraciones (TDD)

- [x] **RED:** preparar la estrategia acordada de prueba de integración PostgreSQL (entorno existente o `testcontainers-go`) y añadir una prueba que espere que `PostgresProjectRepository.Create` inserte `id`, `name`, `start_date` y `planned_finish_date`.
- [x] **RED:** añadir una prueba de integración que demuestre que la restricción de base de datos rechaza `planned_finish_date` anterior a `start_date`; conservar la validación de aplicación como barrera primaria.
- [x] **GREEN:** agregar la migración versionada que cree `projects` con `id UUID PRIMARY KEY`, `name TEXT NOT NULL`, `start_date DATE NOT NULL`, `planned_finish_date DATE NOT NULL` y `CHECK (planned_finish_date >= start_date)`; no agregar columnas de estado ni miembros.
- [x] **GREEN:** implementar `PostgresProjectRepository` con `INSERT` parametrizado y la configuración/conexión mínima necesaria para que la API use PostgreSQL. `cmd/api` exige `DATABASE_URL`, crea y verifica un `pgxpool.Pool`, compone el repositorio y el caso de uso, genera UUID y registra `POST /projects`. La aplicación no ejecuta migraciones: aplicar las SQL versionadas con la CLI `golang-migrate` es un prerrequisito explícito.
- [x] **TRIANGULATE:** ejecutar las pruebas de integración contra PostgreSQL real y verificar que las fechas de calendario se persisten y recuperan sin componente horario ni desplazamiento de zona. Docker ya está accesible; `go test -count=1 -v ./internal/project/infrastructure/postgres` pasó con ambas pruebas ejecutadas contra contenedores PostgreSQL reales.
- [x] **REFACTOR:** mantener SQL, detalles del driver y migraciones en infraestructura; confirmar que el dominio y el caso de uso no importan paquetes PostgreSQL.
- [x] Ejecutar `go test ./...` con el entorno de integración acordado y registrar la evidencia RED/GREEN/TRIANGULATE/REFACTOR. La cobertura de persistencia real quedó ejecutada: `go test -count=1 -v ./internal/project/infrastructure/postgres` pasó con PostgreSQL en contenedores, y `go test ./...` pasó.

## 4. Verificación de alcance y cierre

- [x] Ejecutar `go test ./...` desde la raíz del módulo como verificación final y registrar el resultado.
- [x] Revisar el diff contra propuesta, especificación y diseño: confirmar que solo existe creación mediante `POST /projects` y que no se introdujeron `status`, miembros, actualización ni consulta de estado.
- [x] Verificar que las respuestas de éxito contienen únicamente `id`, `name`, `start_date` y `planned_finish_date`, y que los errores no exponen detalles internos de PostgreSQL.

## Review Workload Forecast

- **Líneas estimadas modificadas:** 500–650 (dominio/caso de uso, handler y pruebas HTTP, repositorio PostgreSQL, migración y pruebas de integración).
- **Riesgo frente al presupuesto de 400 líneas:** Alto; el rango supera el presupuesto y combina dos áreas revisables: corte HTTP/aplicación y persistencia/migraciones.
- **Chained PRs recomendados:** Sí. Propuesta: PR 1 para dominio, caso de uso y handler con pruebas unitarias/HTTP; PR 2 para migración, repositorio PostgreSQL y pruebas de integración.
- **Decisión necesaria antes de apply:** Sí. Con `delivery_strategy: ask-on-risk`, acordar la división en PRs encadenados o autorizar explícitamente `size:exception`; además confirmar el enfoque de migraciones y la disponibilidad del entorno PostgreSQL de integración.

## Fuera de alcance confirmado

- Estado de proyecto: entrada, persistencia, cálculo y respuesta.
- Miembros de proyecto.
- Actualización de proyectos.
- Consulta del estado de un proyecto.
