# Avance de implementación: Modificar un proyecto existente

## Estado

- Estado de implementación: complete; las 15 tareas están verificadas como completas.
- Modo: TDD estricto.
- Entrega: un solo PR con `size:exception` aprobado explícitamente; sin PR encadenados.
- Diff de implementación de autoría: 480 adiciones + 14 eliminaciones = 494 líneas modificadas entre código fuente y pruebas. El total, incluido este informe de implementación y las modificaciones de artefactos de tareas, es 582 adiciones + 45 eliminaciones = 627; se excluyen los cambios de inicialización de SDD en `openspec/config.yaml`. Se aplica `size:exception`, aprobado por la persona responsable del mantenimiento.
- Verificación: Go está disponible (`go1.27.1`). Los comandos focalizados y completos requeridos pasaron usando la configuración/socket de Docker rootless autorizados; también se ejecutó una suite completa sin caché y pasaron ambos paquetes de integración con PostgreSQL.

## Avance de tareas

| Tarea | Estado | Notas |
|------|-------|-------|
| 1.1 | Complete | Las pruebas del caso de uso para reemplazo, fechas iguales, validación/sin escritura, error del repositorio y not found pasan en la suite unitaria focalizada. |
| 1.2 | Complete | Las pruebas de la API compuesta cubren el registro de PUT, UUID mal formado/sin llamada al repositorio, ID desconocido/404 y las rutas existentes de proyectos/historias; pasa la suite unitaria focalizada. |
| 1.3 | Complete | Los casos HTTP de éxito, JSON estricto, campos faltantes/inválidos, orden de fechas, formato de respuesta y ausencia de escrituras pasan en la suite unitaria focalizada. |
| 1.4 | Complete | La prueba de composición de la API confirma la ruta PUT y las rutas existentes; pasa la suite unitaria focalizada. |
| 1.5 | Complete | El escenario de integración de actualización/preservación de historias y not found de PostgreSQL pasó en la suite completa sin caché. |
| 2.1 | Complete | Se implementaron el contrato del repositorio y el sentinel not found; pasan las pruebas de aplicación. |
| 2.2 | Complete | El caso de uso de actualización valida mediante el analizador de fechas compartido y la invariante del dominio; pasan las pruebas de aplicación. |
| 2.3 | Complete | La actualización acotada a tres columnas y la asignación de cero filas a not found pasaron la prueba de integración de PostgreSQL. |
| 2.4 | Complete | La decodificación estricta de solicitudes, la validación de UUID, la respuesta y la asignación de errores pasan las pruebas HTTP focalizadas. |
| 2.5 | Complete | La ruta PUT quedó registrada en la composición de la API; pasan las pruebas de rutas compuestas. |
| 3.1 | Complete | Se extrajeron funciones auxiliares compartidas para pruebas del comando/solicitud de actualización, se conservaron los casos tabulares, se ejecutó gofmt y se repitieron las pruebas focalizadas con éxito. |
| 3.2 | Complete | La revisión estática confirma que SQL asigna solo los tres campos básicos; las aserciones de integración verifican el ID del proyecto y los valores de las historias asociadas, y pasaron en la ejecución de PostgreSQL sin caché. |
| 3.3 | Complete | El comando unitario focalizado requerido pasa en los cuatro paquetes solicitados. |
| 3.4 | Complete | La suite completa requerida pasó con Docker rootless habilitado; la ejecución sin caché también pasó en los paquetes de integración de PostgreSQL para proyectos e historias. |
| 3.5 | Complete | Se revisó el alcance y se contabilizaron 494 líneas modificadas en el diff de implementación de autoría; `size:exception` está aprobado y registrado explícitamente. |

Las 15 tareas están marcadas `[x]` en `tasks.md`. Durante la primera implementación histórica no había toolchain de Go; este registro conserva que las pruebas se escribieron antes del código de producción, pero que la ejecución RED quedó bloqueada. No se afirma retroactivamente que se haya ejecutado RED. A continuación se registran los resultados actuales de GREEN y posteriores al refactor, incluida la ejecución real de la integración con PostgreSQL.

## Evidencia del ciclo TDD

| Tarea | Archivo(s) de prueba | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|------|--------------|-------|------------|-----|-------|-------------|----------|
| 1.1 / 2.1 / 2.2 | `tests/unit/project/application/update_project_test.go` | Unitaria | Intento previo: bloqueado (`go: command not found`); suite focalizada de continuación: pasó | Pruebas escritas antes de producción; la ejecución RED siguió bloqueada en el intento previo | Pasó en la suite focalizada | Pasan los casos de éxito con fechas distintas/iguales, siete casos de entrada inválida y propagación de errores del repositorio y not found | Se extrajo `validUpdateCommand`; la suite focalizada volvió a pasar |
| 1.2 / 1.4 / 2.5 | `tests/unit/cmd/api/main_test.go` | Composición de API | Intento previo: bloqueado; suite focalizada de continuación: pasó | La prueba inicial de ruta se escribió antes de producción; no se observó ejecución RED. Se agregaron casos de ruta con ID inválido/desconocido durante la continuación, después de implementar | Pasó en la suite focalizada | Se cubren la ruta exitosa, UUID mal formado antes de llamar al repositorio, asignación de not found y rutas existentes de creación/historias | Pasaron gofmt y la suite focalizada; no hubo refactor de producción |
| 1.3 / 2.4 | `tests/unit/project/transport/http/update_handler_test.go` | Unitaria HTTP | Intento previo: bloqueado; suite focalizada de continuación: pasó | Pruebas escritas antes de producción; la ejecución RED siguió bloqueada en el intento previo | Pasó en la suite focalizada | Se cubren fechas iguales, cuerpos mal formados/sobrantes/con campos desconocidos, cada campo faltante, nombre vacío, fecha inválida, finalización anterior al inicio y errores de UUID/not found/internos | Se extrajo `executeUpdateRequest`; pasaron gofmt y la suite focalizada |
| 1.5 / 2.3 | `tests/integration/project/postgres/repository_integration_test.go` | Integración con PostgreSQL | Intento previo: bloqueado; se ejecutó la suite de continuación sin caché | Prueba escrita antes de producción; la ejecución RED siguió bloqueada en el intento previo | Pasó en la suite completa sin caché | Pasaron contra PostgreSQL las aserciones del ID del proyecto, los tres campos básicos, la preservación de historias asociadas y el ID desconocido | Se revisó SQL de forma estática; no fue necesario refactorizar producción |
| 3.1 / 3.3 | Suites unitarias focalizadas de proyectos/API | Unitaria | Línea base de continuación: pasaron los 4 paquetes solicitados | No se observaron ejecuciones RED históricas | Pasó | Los casos de aplicación/HTTP/API anteriores pasaron en los cuatro paquetes | Se extrajeron funciones auxiliares compartidas de pruebas, se ejecutó gofmt y luego pasó la suite focalizada |
| 3.4 | `go test ./...` | Suite completa | N/A | N/A | Pasó el comando exacto requerido (caché de Go); también pasó la suite completa sin caché | Ambos paquetes de integración con PostgreSQL se ejecutaron con éxito en la ejecución sin caché | No fue necesario otro refactor después de que la suite pasó |

## Evidencia de la unidad de trabajo

| Evidencia | Resultado |
|----------|--------|
| Prueba focalizada | `DOCKER_CONFIG="$HOME/.config/docker-rootless" DOCKER_HOST="unix:///run/user/$(id -u)/docker.sock" go test ./tests/unit/project/... ./tests/unit/cmd/api/...` — exit 0; pasaron los 4 paquetes solicitados. |
| Entorno de ejecución | `DOCKER_CONFIG="$HOME/.config/docker-rootless" DOCKER_HOST="unix:///run/user/$(id -u)/docker.sock" go test ./...` — exit 0. El comando exacto usó la caché de pruebas; una ejecución sin caché con `... go test -count=1 ./...` también terminó con exit 0: `tests/integration/project/postgres` pasó en 6.493s y `tests/integration/story/postgres` pasó en 25.803s. |
| Límite de reversión | Revertir los cambios de US-02 en `internal/api/api.go`, `internal/project/application/create_project.go`, `internal/project/application/update_project.go`, `internal/project/infrastructure/postgres/repository.go`, `internal/project/transport/http/handler.go` y las pruebas relacionadas de proyectos/API en `tests/unit/` y `tests/integration/project/postgres/repository_integration_test.go`; no se requiere revertir el esquema ni los datos. |
| Higiene del diff | `git diff --check` — exit 0, sin salida después de gofmt y del refactor de funciones auxiliares de pruebas. |

## Archivos modificados

- `internal/api/api.go`
- `internal/project/application/create_project.go`
- `internal/project/application/update_project.go`
- `internal/project/infrastructure/postgres/repository.go`
- `internal/project/transport/http/handler.go`
- `tests/unit/project/application/create_project_test.go`
- `tests/unit/project/application/update_project_test.go`
- `tests/unit/project/transport/http/handler_test.go`
- `tests/unit/project/transport/http/update_handler_test.go`
- `tests/unit/cmd/api/main_test.go`
- `tests/integration/project/postgres/repository_integration_test.go`

`openspec/config.yaml` se actualizó durante la inicialización de SDD, después de la comprobación inicial del árbol de trabajo limpio; no se editó durante esta implementación.

## Bloqueo histórico y seguimiento actual

La primera implementación no pudo ejecutar las pruebas porque Go no estaba disponible; las pruebas se escribieron antes del código de producción, pero no se observó RED. En una continuación posterior sin las variables de entorno de Docker rootless, las pruebas no pudieron crear un proveedor de Docker. Luego, el usuario autorizó la configuración de Docker rootless; con la configuración y el socket aislados, pasaron la suite completa requerida y una suite completa sin caché. Durante esta continuación no se necesitaron cambios en la implementación de producción; solo se actualizaron funciones auxiliares de pruebas y evidencia de avance.
