# Diseño: Crear un Sprint y definir su Sprint Goal

Este diseño implementa la creación de un Sprint para un proyecto existente mediante el corte vertical ya usado por historias. El alcance se limita a generar su UUID, validar y guardar el Sprint Goal, mantener la asociación referencial y devolver el recurso creado; no consulta el backlog ni asigna historias.

## Enfoque técnico

Seguir `POST /projects/{project_id}/sprints` e incorporar un módulo `internal/sprint/` con dominio, caso de uso, adaptador HTTP y repositorio PostgreSQL. La especificación `specs/sprint/spec.md` define el contrato de producto: UUID generado por servidor, Sprint Goal obligatorio, proyecto existente y ausencia de asignación de historias.

Persistir con una única inserción a `sprints`, protegida por FK nombrada hacia `projects(id)`. El endpoint valida el UUID de ruta en HTTP; la base de datos resuelve la existencia del proyecto en la misma escritura, evitando una consulta previa y la carrera entre comprobación e inserción. Registrar la ruta de Sprint únicamente con migración 000003 limpia; conservar por separado la disponibilidad actual de historias desde la versión 000002 limpia.

La recomendación de datos es validar Sprint Goal vacío o compuesto solo por espacios como requerido, siguiendo la validación existente de campos de texto, y conservar el valor original recibido. No fijar longitud máxima, normalización, unicidad del goal ni otras reglas no presentes en la aceptación.

## Decisiones de arquitectura

### Decisión: Crear el Sprint bajo la ruta del proyecto

**Elección**: `POST /projects/{project_id}/sprints`, con `project_id` exclusivamente en la ruta.

**Alternativas consideradas**: `POST /sprints` con `project_id` en el body; crear una colección global y vincularla después.

**Justificación**: La ruta anidada sigue el precedente de `POST /projects/{project_id}/stories` y expresa la asociación requerida en la entrada. No agrega una etapa de asociación ni dependencia de US-07.

### Decisión: Mantener un módulo vertical Sprint

**Elección**: Añadir `domain`, `application`, `transport/http` e `infrastructure/postgres` en `internal/sprint/`, con interfaces pequeñas siguiendo `internal/story/`.

**Alternativas consideradas**: incorporar Sprint dentro de `project` o reutilizar tipos de `story`.

**Justificación**: Sprint es una capacidad propia con identidad, invariantes y persistencia; separar el módulo evita acoplar la creación a la consulta/asignación de historias y respeta la estructura actual.

### Decisión: Garantizar proyecto existente por FK y traducir solo su violación

**Elección**: Una inserción parametrizada y constraint explícito `sprints_project_id_fkey`; mapear únicamente SQLSTATE `23503` con ese nombre a `application.ErrProjectNotFound`.

**Alternativas consideradas**: consultar `projects` antes del insert; convertir cualquier error de FK en proyecto ausente.

**Justificación**: La FK garantiza integridad ante operaciones concurrentes y evita escrituras huérfanas. La traducción selectiva sigue el patrón de historias y no clasifica errores de otras constraints como un proyecto ausente.

### Decisión: Habilitar Sprint según su versión de esquema sin alterar la ruta de historias

**Elección**: Añadir migración 000003 y evaluar la versión de migraciones una vez al inicio. Componer las dependencias de historias si la migración está limpia y `version >= 2`; componer Sprint de forma independiente si está limpia y `version >= 3`.

**Alternativas consideradas**: habilitar ambas rutas solo desde la versión 3; exponer Sprint sin comprobar readiness; cambiar el umbral existente de historias.

**Justificación**: La API ya usa `schema_migrations` y conserva proyectos sin dependencias de historias cuando el esquema no está listo. El requisito técnico es no registrar Sprint antes de su esquema, pero un despliegue de versión 2 debe conservar historias. Un lookup fallido o estado `dirty` no habilita ninguna ruta que dependa de migraciones.

### Decisión: Mantener compatibilidad de composición HTTP existente

**Elección**: Preservar la firma y el comportamiento de `api.NewHTTPHandler` para consumidores actuales y sumar una composición explícita que admita dependencias opcionales de historias y Sprint; la composición nueva puede delegar en un constructor interno común. El arranque utiliza la composición nueva según cada umbral.

**Alternativas consideradas**: reemplazar directamente los parámetros actuales por una estructura única de dependencias; agregar otra lista variádica independiente en la firma actual.

**Justificación**: La firma actual ya permite llamadas project-only y con dependencias story; mantenerla evita romper las pruebas y consumidores existentes, mientras que una configuración explícita resuelve la disponibilidad independiente de ambas rutas.

## Flujo de datos

```text
POST /projects/{project_id}/sprints
        │
        ▼
HTTP handler ── valida método, JSON y UUID de ruta
        │
        ▼
CreateSprintUseCase ── dominio valida Sprint Goal y asigna UUID generado
        │
        ▼
SprintRepository.Create ── INSERT parametrizado
        │                         │
        │                         └── FK sprints_project_id_fkey
        ▼
Sprint devuelto ── HTTP 201 + JSON
```

En error, la capa HTTP traduce validación de dominio a 422, formato JSON inválido a 400, FK específica de proyecto ausente a 404 e incidencias inesperadas a 500 genérico. No se filtran errores internos. La inserción fallida no deja un Sprint persistido.

## Cambios de archivos

| Archivo | Acción | Descripción |
|---|---|---|
| `internal/sprint/domain/sprint.go` | Crear | Entidad Sprint, `ValidationError` y constructor que exige goal no vacío; conservar el texto original. |
| `internal/sprint/application/create_sprint.go` | Crear | Command, `SprintRepository`, `IDGenerator`, `ErrProjectNotFound` y caso de uso de validación/creación/persistencia. |
| `internal/sprint/transport/http/handler.go` | Crear | Handler POST, decodificación estricta de un único objeto JSON, validación UUID, contrato JSON y traducción de errores. |
| `internal/sprint/infrastructure/postgres/repository.go` | Crear | Inserción parametrizada y traducción selectiva de la FK nombrada a `ErrProjectNotFound`. |
| `internal/project/infrastructure/postgres/migrations/000003_create_sprints.up.sql` | Crear | Crear `sprints(id UUID PRIMARY KEY, project_id UUID NOT NULL, sprint_goal TEXT NOT NULL)` y FK nombrada a `projects(id) ON DELETE RESTRICT`. No incluir fechas, duración, estado, índice adicional o reglas de contenido. |
| `internal/project/infrastructure/postgres/migrations/000003_create_sprints.down.sql` | Crear | Eliminar tabla `sprints`; el rollback de datos debe ser deliberado porque descarta los Sprints existentes. |
| `internal/api/api.go` | Modificar | Añadir `SprintDependencies`, registrar ruta únicamente cuando se suministren dependencias y mantener intacta la API de composición existente mediante un constructor compatible. |
| `cmd/api/main.go` | Modificar | Derivar disponibilidad de historias (versión limpia >=2) y Sprint (versión limpia >=3) de una misma lectura de `schema_migrations`; registrar disponibilidad y fallas sin impedir la ruta de proyectos. |
| `tests/unit/sprint/domain/sprint_test.go` | Crear | Casos válidos, goal ausente/espacios, conservación del texto y ausencia de reglas especulativas. |
| `tests/unit/sprint/application/create_sprint_test.go` | Crear | ID generado, asociación, una escritura, validación previa al repositorio y propagación de errores. |
| `tests/unit/sprint/transport/http/handler_test.go` | Crear | 201/JSON, JSON malformado o no singular, campos desconocidos, UUID inválido, goal obligatorio, 404, 500 y método no soportado; comprobar cero escrituras en rechazos. |
| `tests/unit/cmd/api/main_test.go` | Modificar | Probar composición de Sprint y preservar endpoints/composición previos; extraer a función pura la selección de dependencias por versión/dirty/error si facilita cubrirla sin arrancar el proceso. |
| `tests/integration/sprint/postgres/repository_integration_test.go` | Crear | Aplicar migraciones 000001–000003 con Testcontainers; comprobar campos guardados, FK nombrada, proyecto inexistente y que errores de otras constraints no se traduzcan como 404. |
| `tests/integration/sprint/postgres/http_integration_test.go` | Crear | Verificar recorrido handler-repositorio-PostgreSQL: creación 201, persistencia exacta, proyecto inexistente 404 y ausencia de escritura ante rechazo. |
| `tests/integration/story/postgres/http_integration_test.go` | Modificar | Extender escenarios de inicio/versionado para Sprint: v2 habilita historias pero no Sprint; v3 limpia habilita ambas; dirty/lookup error no habilita dependencias migradas y proyectos siguen disponibles. |
| `README.md` | Modificar | Documentar migración 000003, prerequisito de ruta, request/response y errores de `POST /projects/{project_id}/sprints`; advertir que el down elimina Sprints. |

La ubicación de migraciones compartidas sigue siendo `internal/project/infrastructure/postgres/migrations/`, tal como está organizada la base de datos actual; no se migra la ubicación durante esta historia. La clasificación de errores y la FK del repositorio se prueban con PostgreSQL real en integración, no con mocks que reproduzcan artificialmente constraints.

## Interfaces y contratos

### HTTP

```http
POST /projects/{project_id}/sprints
Content-Type: application/json

{"sprint_goal":"Entregar el flujo inicial de métricas"}
```

Respuesta exitosa `201 Created`:

```json
{
  "id": "<uuid generado por servidor>",
  "project_id": "<uuid de la ruta normalizado>",
  "sprint_goal": "Entregar el flujo inicial de métricas"
}
```

| Caso | HTTP | Contrato de error |
|---|---:|---|
| Body inválido, múltiples valores JSON o campos desconocidos | 400 | `{"error":"invalid_request","message":"..."}` |
| UUID de proyecto inválido | 422 | `validation_failed`, `fields.project_id = "must be a valid UUID"` |
| Sprint Goal ausente/vacío según validación del dominio | 422 | `validation_failed`, `fields.sprint_goal = "is required"` |
| Proyecto no encontrado por FK | 404 | `{"error":"project_not_found","message":"project not found"}` |
| Método distinto de POST | 405 | `{"error":"method_not_allowed","message":"only POST is supported"}` |
| Error inesperado de persistencia | 500 | `{"error":"internal_error","message":"an unexpected error occurred"}` |

Los errores mantienen la estructura y estados ya usados por `internal/story/transport/http/handler.go`. El decoder debe rechazar campos no definidos, incluyendo un `id` o campos de estado/asignación enviados por cliente; el servidor es dueño del UUID. La respuesta no agrega fechas, duración, estado ni historias.

### Dominio y aplicación

```go
type Sprint struct {
    ID        string
    ProjectID string
    SprintGoal string
}

type CreateSprintCommand struct {
    ProjectID  string
    SprintGoal string
}

type SprintRepository interface {
    Create(ctx context.Context, sprint domain.Sprint) error
}
```

El constructor de dominio valida solo que `SprintGoal` no sea vacío/blanco; al igual que el dominio actual de historia, devuelve el valor original en vez de normalizarlo. La capa de aplicación genera UUID usando el patrón existente (`api.NewProjectID`/`github.com/google/uuid`) y persiste una vez. `ProjectID` ya llega validado como UUID desde la ruta; su existencia es responsabilidad final de la FK, no de una lectura anticipada.

### PostgreSQL

```sql
CREATE TABLE sprints (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL CONSTRAINT sprints_project_id_fkey
        REFERENCES projects(id) ON DELETE RESTRICT,
    sprint_goal TEXT NOT NULL
);
```

La migración 000003 debe mantener paridad `up`/`down` conforme a `golang-migrate`; no se modifica ni renumera 000001/000002. La API no ejecuta migraciones automáticamente. La validación de no vacío ocurre en el dominio para devolver errores por campo; el `NOT NULL` ofrece integridad mínima en la base, sin agregar una regla CHECK adicional.

### Composición y disponibilidad

Usar la lectura existente de `schema_migrations(version, dirty)` durante el arranque. Si falla o está dirty, iniciar todavía la API de proyectos, sin rutas dependientes de migración. Si está limpio, proporcionar dependencias de historias para `version >= 2` y las de Sprint para `version >= 3`, cada una de modo independiente. La nueva forma de composición debe permitir ambos módulos o solo uno y preservar el constructor actual para los tests/llamadores que solo habilitan proyectos o historias.

## Estrategia de pruebas

La configuración del proyecto activa TDD estricto (`strict_tdd: true`) y declara `go test ./...`. La fase de implementación debe seguir **RED → GREEN → REFACTOR**: agregar primero cada prueba de comportamiento y confirmar que falla por la ausencia de la capacidad; implementar el mínimo para pasarla; refactorizar manteniendo las pruebas verdes. No se ejecutan pruebas en esta fase de diseño.

| Capa | Qué verificar | Enfoque |
|---|---|---|
| Unidad — dominio | Goal válido preservado; ausencia o solo espacios produce error de campo; no hay reglas extra. | Casos table-driven con `t.Run`, siguiendo `tests/unit/story/domain/`. |
| Unidad — aplicación | ID servidor, project ID y goal persistidos exactamente una vez; validación evita write; errores del repo propagados. | Fake de repositorio pequeño siguiendo `tests/unit/story/application/`. |
| Unidad — HTTP | JSON contract, UUID, errores y ausencia de writes en fallos; rutas/verbos correctos. | `httptest`, tabla de escenarios y fakes como `tests/unit/story/transport/http/`. |
| Integración — PostgreSQL | Migración, columnas, FK y traducción selectiva de errores. | Testcontainers PostgreSQL 16, aplicando SQL versionado desde el módulo de migraciones; no simular FK. Requiere Docker. |
| Integración — HTTP/API | Alta end-to-end en el handler montado sobre PostgreSQL y respuesta coincidente con fila persistida; gating del arranque por versiones. | Ampliar patrón en `tests/integration/story/postgres/http_integration_test.go`; contenerizar DB y usar la prueba existente de proceso para readiness. |
| Suite | No regresión entre módulos y paquetes. | Ejecutar `go test ./...` durante implementación; integración requiere Docker según README y pruebas actuales. |

## Matriz de amenazas

El cambio añade una ruta HTTP. Las cinco filas de la matriz base se explicitan abajo; ninguna implica por sí sola selección de repositorio local, comandos, VCS o automatización PR. Se agrega la frontera de ruta HTTP como aplicable para cubrir los riesgos reales de esta capacidad.

| Frontera | Aplicabilidad | Comportamiento seguro / fallo esperado | Pruebas RED previstas |
|---|---|---|---|
| Ruta HTTP de Sprint (adicional, por ser cambio de routing) | Aplicable | Registrar solo `POST /projects/{project_id}/sprints` con dependencias y esquema v3 limpio. Proyecto no listo conserva las rutas permitidas existentes y no registra Sprint. Método incorrecto: 405; UUID inválido: 422; JSON malformado/desconocido: 400; goal ausente: 422; proyecto inexistente: 404; error inesperado: 500 genérico, sin escritura ni detalle interno. | Composición HTTP: ruta ausente sin dependencias; POST válido registrado; GET/PUT/DELETE no escriben; constructor de API conserva rutas existentes. Pruebas de arranque para v2, v3 limpia, dirty y lookup error. Handler: un caso RED por método, UUID, JSON, goal, FK y fallo inesperado, comprobando cero writes. |
| Rutas similares a documentación (`requirements.txt`, `CMakeLists.txt`, Markdown/MDX ejecutable, `README.sh`) | N/A — no hay clasificación ni ejecución de archivos; README solo se documenta. | No se ejecutan archivos ni se modifica resolución de paths. | Ninguna. |
| Selección de repositorio Git (`git -C`, paths relativos/absolutos) | N/A — la API no ejecuta Git ni elige repositorios. | Ninguna operación Git forma parte del flujo. | Ninguna. |
| Estado de commit (staged, `commit -a`, índice vacío) | N/A — el cambio no automatiza commits ni toca el índice. | No se realizan operaciones VCS desde el servicio. | Ninguna. |
| Push (tracking branch, primer push, refspec explícito) | N/A — no existe operación push ni selección de destino remoto. | No se invoca red/VCS para publicar cambios. | Ninguna. |
| Comandos PR (`--head`, prefijo de entorno, composición de comandos) | N/A — no hay automatización de PR o composición de comandos. | El servicio no ejecuta `gh` ni comandos de shell. | Ninguna. |

Las pruebas de las fronteras aplicables se conservan con sus escenarios en la fase `sdd-tasks` y deberán escribirse como RED antes de implementar rutas/composición.

## Secuencia propuesta para tareas posteriores

1. **Contrato primero**: pruebas RED de dominio, aplicación, JSON/HTTP y composición, incluyendo todos los escenarios aplicables de la matriz.
2. **Modelo y persistencia**: dominio/caso de uso y migración 000003 con `up`/`down`; pruebas PostgreSQL con Testcontainers para FK, escritura única y clasificación de errores.
3. **Exposición y gating**: handler, composición retrocompatible, disponibilidad independiente v2/v3 en arranque; cubrir no regresión de historias y proyectos.
4. **Documentación y verificación**: actualizar README con migración/ruta/contrato; ejecutar pruebas unitarias y `go test ./...`, incluyendo integración cuando Docker esté disponible.

Estas son unidades de diseño para desglosar en `tasks.md`, no tareas ejecutadas. El umbral de revisión configurado es 400 líneas; el módulo vertical, integración, documentación y cambios de arranque podrían acercarse al límite. Por la estrategia `ask-on-risk`, `sdd-tasks` debe estimar el cambio agregado y pedir resolución antes de `apply` si el riesgo de excederlo es alto.

## Migración y despliegue

La API mantiene el patrón existente: las migraciones son externas y no se aplican al arrancar. Desplegar primero/aplicar `000003_create_sprints.up.sql` con `golang-migrate`; después, una instancia que observe `schema_migrations` limpio en versión >=3 registra Sprint. Con versión 2 limpia se mantienen historias habilitadas sin Sprint. Un estado dirty o error de lectura deja fuera las rutas dependientes de migración, sin retirar creación de proyectos. Documentar el prerrequisito en README.

No se requiere backfill: la tabla es nueva. No se debe aplicar el `down` automáticamente como mecanismo de rollback de aplicación porque elimina la tabla y los Sprints persistidos; revertir código sin destruir esos datos, salvo que se apruebe explícitamente una reversión de datos.

## Preguntas abiertas

- Ninguna bloqueante para el diseño: el contrato aprobado define UUID como identificador y no solicita identificador visible adicional, fechas, duración, estado u orden.
- La fase de tareas debe confirmar la estimación frente al presupuesto de revisión de 400 líneas antes de implementar; no cambia el alcance funcional.
