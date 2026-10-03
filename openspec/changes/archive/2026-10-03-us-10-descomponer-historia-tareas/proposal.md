# Propuesta: Descomponer una historia del Sprint en tareas (US-10)

## Intención

Implementar US-10 (issue #38) para que un integrante del equipo pueda descomponer una historia asignada a un Sprint en una o más tareas, con información suficiente para identificar el trabajo y una estimación de esfuerzo en horas opcional. La restricción central es que no se deben poder crear tareas para una historia que no pertenezca al Sprint seleccionado. Las tareas creadas quedan persistidas y asociadas a su historia para que US-15 (registro de esfuerzo) pueda usarlas después.

Hoy la pertenencia historia-Sprint solo existe en la tabla `sprint_stories` (US-09, migración `000007`) y únicamente se consulta dentro de la asignación (`PostgresStoryRepository.AssignStoriesForProject`). No hay un caso de uso de lectura reutilizable que la verifique, por lo que este cambio debe construir esa verificación.

## Decisiones confirmadas por el usuario

Estas decisiones fueron confirmadas por el usuario; no se atribuyen al issue.

- La entidad nueva se llama `Task` en el código (inglés, consistente con `Story`, `Sprint` y `Member`); el vocabulario de negocio visible se mantiene en español donde corresponda.
- La tarea **no tiene campo de estado propio** en este alcance. No se diseña campo `status` ni máquina de estados para `Task`; se difiere a US-15.
- La estimación de horas es opcional (nil = sin estimar) y reutiliza el patrón de validación de `Story.EstimatedHours *float64` (`internal/story/domain/story.go:28-30,123-142`): máximo `99999.99`, como máximo 2 decimales.
- Se reutiliza la arquitectura de 4 capas de `projectmember`/`story` (dominio, aplicación, infraestructura PostgreSQL, transporte HTTP) y las convenciones de errores HTTP existentes: 404 `*_not_found`, 409 conflicto, 422 `validation_failed` con `fields`, 400 `invalid_request`.
- La verificación de pertenencia historia-Sprint se construye nueva, reutilizando el patrón SQL de `AssignStoriesForProject` (`internal/story/infrastructure/postgres/repository.go:38-107`) sin copiar su código.
- No hay restricción de duplicados ni cantidad máxima de tareas por historia.
- Se pueden crear tareas para una historia cuyo Sprint ya esté cerrado (`is_closed=true`): no se agrega ninguna restricción nueva por ese estado. El AC no lo prohíbe y no se inventa una regla no pedida, aunque US-09 sí rechaza asignar historias a un Sprint cerrado.

## Alcance

### Incluido
- Contrato HTTP para crear una o más tareas para una historia seleccionada dentro de un Sprint de un proyecto, identificados por IDs que aporta el cliente.
- Validación del lote completo antes de persistir: título obligatorio (no vacío tras recortar espacios) y estimación de horas opcional con el patrón de `Story`. Ante datos inválidos se rechaza todo el lote con 422 y el detalle por campo.
- Verificación de pertenencia: proyecto existente, Sprint del proyecto, historia del proyecto y asociación historia-Sprint existente en `sprint_stories`. Si la historia no pertenece al Sprint seleccionado, se rechaza sin crear tareas.
- Lote atómico: se crean todas las tareas de la solicitud o ninguna, también ante fallo de persistencia.
- Tabla nueva de tareas con asociación a su historia (y al Sprint/proyecto en que fue creada) mediante la migración `000009`.
- Habilitación condicional de la ruta según la versión de esquema, con un flag nuevo `Tasks` en `api.MigrationReadiness`.
- Pruebas unitarias (dominio, aplicación, transporte HTTP) e integración PostgreSQL/Testcontainers, siguiendo TDD estricto.

### Fuera de alcance
- Crear historias (US-05), asignar historias a un Sprint (US-09) y registrar horas trabajadas (US-15).
- Estado de la tarea, transiciones o cierre de tareas (diferido a US-15 por decisión del usuario).
- Listar, consultar, modificar o eliminar tareas; ninguna operación de lectura de tareas es requerida por el AC.
- Asignar tareas a integrantes, descripción extendida u otros campos no pedidos por el AC.
- Restricciones de duplicados o de cantidad máxima de tareas.
- Resolver la fragilidad del doble umbral de habilitación ni el conflicto histórico de numeración `000003`.

## Capacidades

### Capacidades nuevas
- `tarea`: crear tareas asociadas a una historia asignada a un Sprint, con título obligatorio y estimación opcional de horas, validación de pertenencia historia-Sprint, atomicidad del lote, contrato HTTP y su habilitación condicionada a la migración `000009`. El nombre sigue la convención de las capacidades existentes en español (`historia`, `sprint`).

### Capacidades modificadas
- Ninguna. Los requisitos existentes de `historia`, `sprint` y `project` no cambian; la asignación de US-09 y sus gates se preservan sin modificaciones. El gate de arranque propio de tareas se especifica dentro de `tarea`.

## Enfoque propuesto

Corte vertical nuevo en `internal/task/` con las 4 capas existentes. Decisiones técnicas propuestas (criterio de arquitectura, sujetas a revisión del usuario en el resumen de esta fase):

1. **Contrato HTTP: ruta anidada** `POST /projects/{project_id}/sprints/{sprint_id}/stories/{story_id}/tasks`.
   Justificación: la asignación de US-09 ya expone `POST /projects/{project_id}/sprints/{sprint_id}/stories` (`internal/api/api.go:137`), por lo que la historia "dentro del Sprint" ya es un recurso direccionable por esa jerarquía; anidar `tasks` debajo expresa literalmente el AC ("seleccionar una historia asignada a un Sprint") y hace que el Sprint seleccionado sea parte obligatoria del contrato, no un campo opcional del cuerpo que podría omitirse. Todas las rutas actuales usan IDs de ruta para la jerarquía y el cuerpo solo para datos (`/projects/{project_id}/stories`, `/members`, `/sprints`). Alternativa descartada: `POST /projects/{project_id}/stories/{story_id}/tasks` con `sprint_id` en el cuerpo, que mezcla jerarquía y datos y rompe la consistencia con la asignación.

2. **Cuerpo como lote**: `{"tasks": [{"title": "...", "estimated_hours": 4.5}, ...]}` con al menos un elemento, `DisallowUnknownFields` y verificación de un solo objeto JSON. Justificación: el AC pide "una o más tareas" y el repositorio ya resuelve "uno o más" como lote atómico (integrantes en US-03, asignación en US-09). Respuesta de éxito 201 con las tareas creadas (ID generado, IDs de historia/Sprint/proyecto, título, estimación). La forma exacta queda para la especificación.

3. **Campos de `Task`**: `ID`, `ProjectID`, `SprintID`, `StoryID`, `Title` (obligatorio, con longitud máxima a fijar en la especificación según el criterio usado para títulos de historia) y `EstimatedHours *float64`. Sin `description`. Justificación: el AC exige "información suficiente para identificar el trabajo" y un título lo cumple; agregar descripción sería ampliar el alcance sin pedido y puede añadirse luego sin romper el contrato (campo opcional nuevo). No hay campo de estado (decisión del usuario).

4. **Persistencia**: migración `000009_create_tasks` (verificado: la cabeza actual del directorio `internal/project/infrastructure/postgres/migrations/` es `000008_add_sprint_closed`). Tabla `tasks` con `project_id`, `sprint_id`, `story_id`, `title`, `estimated_hours NUMERIC` nulable y marca de creación. La asociación se protege también en base de datos con una clave foránea hacia `sprint_stories(sprint_id, story_id)` con `ON DELETE RESTRICT`, como defensa en profundidad de la regla "la historia pertenece al Sprint"; el detalle de las FKs compuestas por proyecto se decide en diseño.

5. **Verificación de pertenencia**: un puerto nuevo del módulo de tareas implementado en PostgreSQL que, dentro de la misma transacción del alta, comprueba proyecto, Sprint del proyecto, historia del proyecto y existencia de la fila en `sprint_stories`, siguiendo el patrón de `AssignStoriesForProject` (consulta acotada por proyecto y bloqueo de la fila relevante), sin reutilizar ni modificar ese método. Errores propuestos: 404 `project_not_found`, `sprint_not_found`, `story_not_found` cuando el recurso no existe o pertenece a otro proyecto; 409 `story_not_in_sprint` cuando la historia existe en el proyecto pero no está asignada a ese Sprint. El mapeo final se fija en la especificación.

6. **Habilitación por esquema**: agregar `Tasks bool` a `api.MigrationReadiness` con `readiness.Tasks = version >= 9` en `ResolveMigrationReadiness` (`internal/api/api.go:87-97`), un `TaskDependencies` nuevo en `api.HTTPDependencies` y el registro condicional de la ruta en `NewHTTPHandlerWithDependencies`. En `cmd/api/main.go` se replica el bloque existente (`if readiness := api.ResolveMigrationReadiness(...); readiness.Tasks { ... } else { log ... }`). `TaskDependencies` será independiente de `dependencies.Stories`, para no repetir la dependencia implícita del bloque de asignación (`main.go:70-71`, que desreferencia `dependencies.Stories` asumiendo que el gate de historias ya la creó).
   **Fragilidad conocida (no se resuelve acá)**: los umbrales de esquema viven en dos lugares coordinados a mano, el `switch` de `cmd/api/main.go:44-58` (historias) y `ResolveMigrationReadiness`; US-09 ya lo documentó. Este cambio solo agrega su umbral en `ResolveMigrationReadiness` y lo cubre con pruebas, sin refactorizar el mecanismo.

## Áreas afectadas

| Área | Impacto | Descripción |
|---|---|---|
| `internal/task/domain/` | Nuevo | Entidad `Task`, validación de título y estimación, `ValidationError` con `Fields`. |
| `internal/task/application/` | Nuevo | Comando, puertos (alta atómica con verificación de pertenencia, generador de IDs) y caso de uso. |
| `internal/task/infrastructure/postgres/` | Nuevo | Repositorio transaccional con verificación de pertenencia y alta del lote. |
| `internal/task/transport/http/` | Nuevo | Handler de creación con decodificación estricta y mapeo de errores. |
| `internal/project/infrastructure/postgres/migrations/000009_create_tasks.{up,down}.sql` | Nuevo | Tabla `tasks` y sus claves foráneas. |
| `internal/api/api.go` | Modificado | Flag `Tasks`, `TaskDependencies`, registro condicional de la ruta. |
| `cmd/api/main.go` | Modificado | Composición del repositorio de tareas con el gate `>= 9` y log de disponibilidad. |
| `tests/unit/task/`, `tests/unit/api/` | Nuevo/Modificado | Dominio, caso de uso con fakes, handler HTTP por tabla de casos, readiness `>= 9`. |
| `tests/integration/task/postgres/` | Nuevo | Persistencia, pertenencia, atomicidad, restricciones de base de datos con Testcontainers. |
| `openspec/specs/tarea/` | Futuro | Especificación nueva, creada en la fase de spec y consolidada al archivar. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Carrera entre la verificación de pertenencia y el alta (la historia se desasigna en paralelo). | Baja | Verificar y escribir en la misma transacción con bloqueo, y FK a `sprint_stories` con `ON DELETE RESTRICT`. |
| Umbral `>= 9` desalineado con la composición de `main.go` (doble umbral). | Media | Pruebas de `ResolveMigrationReadiness` para versiones 8, 9 y estado sucio; bloque de `main.go` idéntico al patrón existente e independiente de `dependencies.Stories`. |
| Otro cambio en curso toma el número `000009` antes de integrar. | Baja | Reconfirmar la cabeza real del directorio de migraciones al aplicar y antes del PR. |
| La FK a `sprint_stories` bloquea una futura operación de desasignar historias con tareas. | Media | Documentarlo como comportamiento intencional; hoy no existe desasignación y la política se revisará cuando se especifique. |
| El PR único supera el presupuesto de revisión de 400 líneas. | Alta | La estrategia `single-pr` fue elegida por el usuario; `sdd-tasks` debe pronosticar el tamaño y dejar la decisión explícita antes de `sdd-apply`. |
| Interpretación distinta de "información suficiente" (se esperaba descripción). | Baja | Decisión explícita y reversible; agregar un campo opcional después no rompe el contrato. |

## Plan de reversión

Revertir el PR único que agrega el módulo `internal/task/`, la migración, los cambios en `internal/api/api.go` y `cmd/api/main.go`, y las pruebas. Si la migración `000009` ya se aplicó en un entorno persistente, comprobar primero si existen tareas registradas: la migración `down` elimina la tabla `tasks` y sus datos, por lo que no debe ejecutarse sin decidir la conservación de esos registros. Mientras el esquema quede en versión `>= 9` sin el código, la API simplemente no registra la ruta; con el esquema en `<= 8`, el gate deja la ruta deshabilitada sin afectar proyectos, historias, Sprints, integrantes ni asignación.

## Dependencias

- US-09 integrada en `main` (tabla `sprint_stories`, migraciones `000007` y `000008`).
- Docker disponible para las pruebas de integración con Testcontainers.

## Decisiones abiertas para el orquestador

Ninguna. La decisión sobre Sprints cerrados fue confirmada por el usuario (ver "Decisiones confirmadas por el usuario"): se permite crear tareas sin restricción nueva por ese estado.

## Criterios de éxito

- [ ] Un cliente puede crear una o más tareas para una historia asignada al Sprint indicado, y quedan persistidas asociadas a esa historia (y a su Sprint y proyecto).
- [ ] Cada tarea requiere título; la estimación de horas es opcional y respeta el máximo `99999.99` y 2 decimales.
- [ ] Una historia que no pertenece al Sprint seleccionado, o un proyecto/Sprint/historia inexistente o ajeno, produce el error correspondiente y cero tareas almacenadas.
- [ ] El lote es todo o nada, también ante fallo de persistencia.
- [ ] La ruta solo se registra con esquema limpio en versión `>= 9`; las rutas existentes conservan sus gates.
- [ ] Las tareas no tienen campo de estado y quedan disponibles para US-15.
- [ ] `go test ./...` pasa; las pruebas de integración PostgreSQL/Testcontainers se ejecutan con Docker disponible.
