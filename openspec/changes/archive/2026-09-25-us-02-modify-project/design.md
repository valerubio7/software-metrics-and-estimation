# Diseño: Modificar un proyecto existente

Agregar un endpoint de actualización de proyectos con reemplazo completo que use los límites existentes de HTTP, aplicación, dominio y PostgreSQL. El diseño implementa los requisitos delta de `project` y mantiene sin cambios el comportamiento de creación de proyectos, creación de historias y estado de proyectos.

## Enfoque técnico

Registrar `PUT /projects/{project_id}` en el `http.ServeMux` existente. El handler decodifica estrictamente el objeto JSON completo de datos básicos y obtiene el ID de destino de la ruta. El caso de uso de actualización analiza las fechas ISO mediante el analizador de fechas existente de la aplicación, valida el proyecto resultante completo con `domain.NewProject` y luego realiza una actualización en el repositorio. La validación ocurre antes de la llamada al repositorio. PostgreSQL actualiza únicamente `name`, `start_date` y `planned_finish_date`; cero filas afectadas se traduce en un error de proyecto inexistente. Las actualizaciones exitosas devuelven HTTP 200 con el formato de respuesta existente del proyecto y el ID de la ruta.

Esto sigue el contrato de reemplazo completo de la propuesta y los escenarios delta para fechas iguales, errores de validación accionables, IDs inexistentes y preservación de datos ajenos a los tres campos básicos. No se necesita modificar el esquema: la tabla `projects` existente ya almacena estos campos y aplica la restricción de orden de fechas.

## Decisiones de arquitectura

### Decisión: Reutilizar las invariantes del proyecto y el análisis de fechas

**Elección**: Agregar `UpdateProjectUseCase` en `internal/project/application/update_project.go`. Reutilizar la función auxiliar `parseDate` del paquete y `domain.NewProject` para validar el reemplazo completo antes de persistirlo.

**Alternativas consideradas**: Ubicar el análisis y la validación en el handler HTTP, o introducir un validador de dominio exclusivo para actualizaciones.

**Justificación**: La validación corresponde al límite de aplicación/dominio, no al de transporte. `NewProject` ya exige nombres no vacíos, fechas obligatorias y el orden correcto de las fechas para la misma representación de tres campos. Reutilizarlo evita que diverjan las reglas de creación y actualización, y evita cambios innecesarios en la API del dominio.

### Decisión: Ampliar el contrato del repositorio de proyectos con la actualización y la semántica not found

**Elección**: Agregar `Update(context.Context, domain.Project) error` a `application.ProjectRepository` y definir `application.ErrProjectNotFound`. La implementación de PostgreSQL comprueba `RowsAffected()` y devuelve ese sentinel cuando el ID no coincide con ninguna fila.

**Alternativas consideradas**: Agregar una segunda interfaz de repositorio e inyectarla por separado, o considerar exitosa una actualización que afecta cero filas.

**Justificación**: La composición actual de la API ya recibe un repositorio de proyectos para todas las operaciones relacionadas. Un único contrato ampliado mantiene sencilla la construcción y permite que las pruebas del caso de uso y del handler inyecten un solo doble de prueba. Devolver not found desde el límite de persistencia distingue una fila inexistente de un fallo de base de datos sin una condición de carrera entre lectura y escritura.

### Decisión: Usar el ID de la ruta como identidad de actualización y mantener acotada la sentencia de actualización

**Elección**: Analizar `project_id` como UUID en el límite HTTP, pasar su cadena canónica al caso de uso y usarla como parámetro de `WHERE id = $1`. La sentencia SQL asigna únicamente las tres columnas de datos básicos.

**Alternativas consideradas**: Aceptar un ID en el cuerpo, cargar y reemplazar un registro de proyecto completo, o actualizar columnas adicionales anticipándose a futuras historias de usuario.

**Justificación**: El endpoint se direcciona explícitamente mediante un ID y quien realiza la llamada ya conoce ese ID. Un `UPDATE` parametrizado y acotado no puede reemplazar la clave primaria ni afectar los datos de historias o miembros. Esto también evita agregar comportamientos de listado o selección, que están fuera de alcance.

### Decisión: Mantener las convenciones existentes de errores y respuestas HTTP

**Elección**: Devolver `200 OK` con los campos de respuesta existentes del proyecto si la operación tiene éxito; devolver `400` para JSON mal formado, `422` para UUID mal formados o errores de validación del dominio, `404` para un UUID bien formado pero desconocido, y `500` para errores inesperados de persistencia. Reutilizar el formato de error JSON existente y evitar exponer detalles del repositorio.

**Alternativas consideradas**: Devolver `204 No Content` o introducir un formato de error/convención de estados nuevo.

**Justificación**: Devolver la representación actualizada explicita el resultado del reemplazo y coincide con la serialización existente de las respuestas de proyectos. Los handlers existentes ya usan objetos de error JSON y errores de campos con `422`; el handler de historias ofrece un precedente para analizar UUID en la ruta y asignar not found.

## Flujo de datos

```text
PUT /projects/{project_id}
  → Ruta ServeMux
  → UpdateProjectHandler: decodificación JSON estricta + análisis de UUID
  → UpdateProjectUseCase: análisis de fechas + validación del Project completo
  → ProjectRepository.Update
  → UPDATE de PostgreSQL únicamente para name/start_date/planned_finish_date
  ← Representación actualizada del proyecto (200) o respuesta asignada de validación/not found/error
```

Un cuerpo o UUID mal formado, o datos de proyecto inválidos, detienen el flujo antes de la persistencia. La operación de actualización consta de una sentencia SQL; un UUID desconocido produce cero filas afectadas y se informa como not found. No se crea un proyecto como alternativa.

## Cambios en archivos

| Archivo | Acción | Descripción |
|------|--------|-------------|
| `internal/api/api.go` | Modificar | Construir el caso de uso de actualización y registrar `PUT /projects/{project_id}` junto a la ruta existente de proyectos. |
| `internal/project/application/create_project.go` | Modificar | Ampliar el contrato compartido de `ProjectRepository` con la operación de actualización. |
| `internal/project/application/update_project.go` | Crear | Definir el comando de reemplazo, el caso de uso y el sentinel de proyecto inexistente; analizar las fechas, validar mediante el constructor existente del dominio y persistir solo proyectos válidos. |
| `internal/project/transport/http/handler.go` | Modificar | Agregar el flujo de solicitud/handler de actualización, el análisis de UUID, la asignación de resultados/errores y reutilizar las funciones auxiliares existentes para respuestas y JSON. |
| `internal/project/infrastructure/postgres/repository.go` | Modificar | Implementar una actualización parametrizada limitada a tres columnas básicas y asignar cero filas afectadas a not found. |
| `tests/unit/project/application/update_project_test.go` | Crear | Probar con una tabla el reemplazo exitoso, las fechas iguales, los casos de validación sin escritura y la propagación de errores del repositorio/not found. |
| `tests/unit/project/transport/http/handler_test.go` | Modificar | Probar la validación de solicitudes en la ruta, los estados y cuerpos de éxito/error, y que las solicitudes inválidas no lleguen a persistencia. |
| `tests/unit/cmd/api/main_test.go` | Modificar | Ampliar los dobles de prueba de proyectos y verificar que la composición de la API registre la ruta de actualización sin afectar las rutas existentes. |
| `tests/integration/project/postgres/repository_integration_test.go` | Modificar | Verificar los valores actualizados, la conservación del ID del proyecto y los datos de historias asociadas, y el comportamiento not found contra PostgreSQL. Aplicar la migración existente de historias en el escenario de preservación. |

No se prevén cambios de producción en `internal/project/domain/project.go`: su constructor actual ya representa la invariante requerida para las actualizaciones. No se prevén cambios en archivos de migración.

## Interfaces / contratos

El contrato del repositorio permanece en el paquete de aplicación de proyectos y suma una operación:

```go
type ProjectRepository interface {
    Create(ctx context.Context, project domain.Project) error
    Update(ctx context.Context, project domain.Project) error
}

var ErrProjectNotFound = errors.New("project not found")
```

El comando de actualización contiene la identidad de la ruta y todos los campos obligatorios del reemplazo:

```go
type UpdateProjectCommand struct {
    ProjectID         string
    Name              string
    StartDate         string
    PlannedFinishDate string
}
```

Solicitud HTTP y representación de éxito:

```http
PUT /projects/{project_id}
Content-Type: application/json

{"name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}
```

```json
{"id":"<project_id>","name":"Metrics portal","start_date":"2026-03-01","planned_finish_date":"2026-06-30"}
```

La solicitud es un reemplazo completo, no un patch. La validación exige los tres campos JSON; el patrón de decodificación estricta existente rechaza los campos desconocidos, el JSON mal formado y los valores JSON sobrantes. Los nombres faltantes/vacíos y las fechas faltantes/inválidas producen errores de validación específicos por campo. Las fechas iguales son válidas. Los errores de actualización del repositorio no deben exponer detalles de la base de datos por HTTP.

## Estrategia de pruebas

Se aplica TDD estricto: primero escribir las pruebas específicas que fallan, implementar hasta que pasen y luego refactorizar. Las pruebas deben organizarse por escenario y usar tablas cuando los casos compartan el mismo comportamiento.

| Capa | Qué probar | Enfoque |
|-------|-------------|----------|
| Pruebas unitarias de dominio/aplicación | Validación del reemplazo completo, fechas iguales, cada campo faltante/vacío/inválido, finalización anterior al inicio y ausencia de escritura en el repositorio para valores inválidos | Usar el constructor de dominio existente y un doble de prueba del repositorio que registre llamadas; comprobar los errores por campo y el número exacto de llamadas. |
| Pruebas unitarias HTTP/ruta | Registro de `PUT`, respuesta exitosa, cuerpo mal formado/campos adicionales, campos faltantes, UUID inválido, ID desconocido y errores de persistencia/internos | Usar `httptest` contra el handler compuesto; comprobar el estado, los campos JSON y que las entradas rechazadas no generen escrituras. Mantener la cobertura existente de las rutas POST e historias. |
| Integración con PostgreSQL | Cambian tres columnas, el ID permanece fijo, un ID desconocido devuelve not found y la asociación/datos de historias existentes no cambian | Ampliar la prueba del repositorio con Testcontainers, aplicar las migraciones 000001 y 000002 para el caso de preservación de historias y luego consultar las filas persistidas del proyecto y las historias. Se requiere Docker; los helpers actuales omiten estas pruebas si Docker no está disponible. |
| Suite completa | Compatibilidad entre paquetes tras ampliar la interfaz del repositorio | Ejecutar `go test ./...`; informar como omitidas las pruebas de integración dependientes de Docker si Docker no está disponible. |

El alcance de autoría previsto incluye un caso de uso, cambios de transporte y repositorio, y pruebas en tres límites. Existe un **riesgo alto** realista de superar el límite fijo de 400 líneas modificadas, porque las pruebas existentes son explícitas y el escenario de integración para preservar historias agrega configuración y aserciones. Mantener las pruebas tabulares y enfocadas en el contrato especificado; no agregar flujos no relacionados. La entrega sigue siendo un solo PR, según lo indicado.

## Matriz de amenazas

Este cambio agrega enrutamiento HTTP, por lo que a continuación se aborda su aplicabilidad. No agrega comandos de shell, subprocesos, automatización de VCS/PR, clasificación de archivos ejecutables ni integración de procesos. Las filas N/A de la matriz proporcionada se indican explícitamente porque esos límites no están presentes; no crear pruebas ni tareas para dichas filas N/A.

| Límite | Aplicabilidad | Respuesta de diseño / comportamiento seguro y ante fallos | Pruebas RED previstas |
|---|---|---|---|
| Registro de rutas HTTP e identidad de ruta (límite de enrutamiento específico del cambio) | Aplicable | Coincidir únicamente con `PUT /projects/{project_id}`; analizar la ruta como UUID y rechazar IDs mal formados antes de acceder al repositorio; un ID válido pero inexistente devuelve 404 sin crear ni modificar datos. | Las pruebas del handler compuesto demuestran que la ruta está registrada, que un UUID inválido no genera escrituras, que un UUID desconocido devuelve 404/sin escrituras y que las demás rutas existentes siguen intactas. |
| Rutas similares a documentación | N/A — no hay cambios en la clasificación ni en la ejecución de archivos. | No se requiere respuesta de diseño. | Ninguna. |
| Selección del repositorio Git | N/A — no hay cambios en comandos de Git ni en el comportamiento de selección del repositorio. | No se requiere respuesta de diseño. | Ninguna. |
| Estado de commit | N/A — no hay cambios en automatización de commit/índice. | No se requiere respuesta de diseño. | Ninguna. |
| Estado de push | N/A — no hay cambios en automatización de push ni de refspec. | No se requiere respuesta de diseño. | Ninguna. |
| Comandos de PR | N/A — no hay cambios en la construcción ni ejecución de comandos de PR. | No se requiere respuesta de diseño. | Ninguna. |

## Migración / despliegue

No se requiere migración. La tabla `projects` ya contiene todos los campos y su restricción de fechas permite fechas iguales. La ruta está disponible cuando se compone el handler existente de la API; no se necesita un feature flag ni un despliegue gradual. La reversión consiste en eliminar/revertir la aplicación, la ruta, la operación del repositorio y las pruebas; las filas existentes siguen siendo válidas.

## Preguntas abiertas

Ninguna. El límite de 400 líneas es un riesgo de entrega que debe monitorearse durante la planificación de tareas y la implementación; no bloquea el diseño técnico.
