# Tareas: Modificar un proyecto existente

## Previsión de carga de revisión

| Campo | Valor |
|-------|-------|
| Líneas modificadas estimadas | 450–550 líneas modificadas de autoría |
| Riesgo del límite de 400 líneas | Alto |
| PR encadenados recomendados | Sí (solo como señal de riesgo; la entrega sigue siendo un PR) |
| División sugerida | Un solo PR; no se planifican partes encadenadas; obtener aprobación de `size:exception` antes de la implementación |
| Estrategia de entrega | single-pr |
| Estrategia de encadenamiento | pending |

Decisión necesaria antes de la implementación: Sí
PR encadenados recomendados: Sí
Estrategia de encadenamiento: pending
Riesgo del límite de 400 líneas: Alto

El diseño considera alto el riesgo para el límite de revisión. Este plan respeta la estrategia de un solo PR confirmada por el cambio principal y no crea un plan de PR encadenados. La persona responsable del mantenimiento aprobó explícitamente `size:exception` antes de la implementación; no se planifican PR encadenados.

### Unidades de trabajo sugeridas

| Unidad | Objetivo | PR probable | Comando de pruebas focalizadas | Entorno de ejecución | Límite de reversión |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Entregar el endpoint de reemplazo completo de US-02, su validación, la actualización acotada en PostgreSQL y la cobertura unitaria/de integración asociada como una sola funcionalidad revisable | Un solo PR (`size:exception` aprobado) | `go test ./tests/unit/project/... ./tests/unit/cmd/api/...` | Ejecutar `go test ./...` con Docker disponible para probar los escenarios de actualización PostgreSQL y preservación de historias basados en Testcontainers; sin Docker, registrar las pruebas de integración omitidas | Revertir el caso de uso de actualización de US-02, la conexión de la ruta/handler, la actualización del repositorio y sus pruebas; no se necesita revertir el esquema ni los datos persistidos |

## Fase 1: RED — Especificar pruebas de comportamiento que fallen

- [x] 1.1 Agregar pruebas RED tabulares del caso de uso en `tests/unit/project/application/update_project_test.go` para el reemplazo válido completo, fechas de inicio/finalización iguales, nombre faltante/vacío, fechas faltantes/inválidas, finalización anterior al inicio, error del repositorio y `application.ErrProjectNotFound`; comprobar que los comandos inválidos no llamen al repositorio.
- [x] 1.2 Agregar pruebas RED de enrutamiento del handler compuesto en `tests/unit/project/transport/http/handler_test.go` para demostrar que `PUT /projects/{project_id}` está registrado, que un UUID mal formado se rechaza antes de escribir en el repositorio, que un UUID bien formado pero desconocido devuelve 404 sin crear ni modificar datos y que las rutas existentes de proyectos/historias siguen disponibles.
- [x] 1.3 Agregar pruebas RED de solicitud/respuesta HTTP en `tests/unit/project/transport/http/handler_test.go` para un reemplazo completo exitoso y el ID/campos básicos devueltos; rechazar JSON mal formado, valores JSON sobrantes, campos JSON desconocidos, cada campo faltante, nombre vacío, fechas mal formadas y finalización anterior al inicio con el formato de error JSON existente y sin escrituras.
- [x] 1.4 Agregar cobertura RED de composición de la API en `tests/unit/cmd/api/main_test.go` para confirmar que la API compuesta expone la ruta de actualización y que las rutas existentes siguen funcionando; ampliar los dobles de prueba solo si lo requiere el nuevo método del repositorio.
- [x] 1.5 Agregar casos RED de PostgreSQL en `tests/integration/project/postgres/repository_integration_test.go` para actualizar exactamente las tres columnas básicas, conservar el ID del proyecto, devolver not found para un ID desconocido y preservar los datos de historias asociadas después de aplicar las migraciones `000001` y `000002`.

## Fase 2: GREEN — Implementar el comportamiento de aplicación y persistencia

- [x] 2.1 Ampliar `ProjectRepository` en `internal/project/application/create_project.go` con `Update(context.Context, domain.Project) error` y agregar `ErrProjectNotFound` al paquete de aplicación; lograr que compilen y pasen las pruebas de 1.1 para la propagación del sentinel.
- [x] 2.2 Crear `internal/project/application/update_project.go` con `UpdateProjectCommand` y `UpdateProjectUseCase`; analizar ambas fechas ISO con el `parseDate` existente, construir el reemplazo completo con `domain.NewProject` y llamar a `Update` solo después de que pase toda la validación.
- [x] 2.3 Implementar la actualización del repositorio en `internal/project/infrastructure/postgres/repository.go` mediante un `UPDATE` parametrizado que asigne únicamente `name`, `start_date` y `planned_finish_date` para el ID proporcionado; asignar cero filas afectadas a `application.ErrProjectNotFound` y preservar los demás errores de base de datos.
- [x] 2.4 Implementar la gestión de solicitudes de actualización en `internal/project/transport/http/handler.go`: decodificar estrictamente un único objeto completo de solicitud, analizar el UUID `project_id` de la ruta, ejecutar el caso de uso, devolver HTTP 200 con el formato de respuesta existente del proyecto, asignar 400 al cuerpo mal formado, 422 al UUID mal formado/errores de validación, 404 a not found y 500 a errores de persistencia inesperados, sin filtrar detalles del repositorio.
- [x] 2.5 Conectar el caso de uso de actualización y registrar `PUT /projects/{project_id}` en `internal/api/api.go`; lograr que pasen los casos RED de enrutamiento de 1.2 y de composición de 1.4 sin modificar las rutas POST ni las de historias existentes.

## Fase 3: REFACTOR — Verificar contratos e integración

- [x] 3.1 Refactorizar las nuevas pruebas en `tests/unit/project/application/update_project_test.go` y `tests/unit/project/transport/http/handler_test.go` para compartir las funciones auxiliares existentes y usar casos tabulares cuando el comportamiento sea común, conservando aserciones explícitas para errores de validación y cero llamadas a persistencia.
- [x] 3.2 Revisar `internal/project/infrastructure/postgres/repository.go` y sus aserciones de integración en `tests/integration/project/postgres/repository_integration_test.go` para asegurar que el SQL de actualización afecte solo las tres columnas especificadas y que las pruebas verifiquen la identidad del proyecto y la preservación de historias relacionadas.
- [x] 3.3 Ejecutar `go test ./tests/unit/project/... ./tests/unit/cmd/api/...` y resolver todos los fallos de las pruebas unitarias focalizadas.
- [x] 3.4 Ejecutar `go test ./...`; verificar los escenarios de integración basados en Testcontainers con Docker disponible o documentar las omisiones dependientes de Docker si no está disponible.
- [x] 3.5 Revisar las adiciones más eliminaciones finales de autoría frente al límite de 400 líneas antes de completar la implementación; conservar la cobertura y el alcance completos en lugar de comprimir o eliminar pruebas, y no continuar con la entrega en un solo PR sin la aprobación explícita de `size:exception` por parte de quien mantiene el proyecto.
