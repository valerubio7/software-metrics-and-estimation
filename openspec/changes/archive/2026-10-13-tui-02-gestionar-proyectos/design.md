# Diseño: gestión de proyectos en la TUI

## Contexto

La TUI usa Bubble Tea. La pantalla actual de proyectos es un placeholder y `internal/tui/api.go` solo sondea disponibilidad. La API ya ofrece `POST /projects`, `PUT /projects/{project_id}` y `GET /projects/{project_id}/status`; crear y modificar reciben `name`, `start_date` y `planned_finish_date`. El backend es responsable de validar y calcular el estado.

## Enfoque

- Extender el modelo Bubble Tea con pantallas y estados explícitos para crear, identificar un proyecto para modificar, modificar, consultar estado y presentar resultado/error. La API no expone un endpoint de listado de proyectos, por lo que la TUI solicita el identificador al usuario.
- Encapsular llamadas HTTP de proyectos en el cliente TUI, reutilizando `APIURL` y configuración existente.
- Para modificar o consultar un proyecto, permitir identificarlo mediante su ID y usar los endpoints existentes, sin inventar endpoints.
- Crear/modificar enviando los campos del formulario sin validación de dominio en la TUI. El servidor conserva autoridad sobre errores 422.
- Interpretar respuestas HTTP como resultados funcionales: los errores 422 y 404 deben mostrarse en la pantalla y no tratarse como fallo de conectividad. Preservar el código y mensaje/campos JSON de la API.
- Obtener el estado solo del endpoint de estado; mostrar sus valores sin recalcularlos localmente.
- Mantener la navegación con escape/retorno al menú y no afectar las secciones backlog/sprints.

## Manejo de errores

El cliente debe leer cuerpos de respuesta tanto en éxito como en error. Para 422 y 404 presenta la información devuelta por la API; para otros fallos HTTP informa estado y detalle disponible. Solo errores de transporte se muestran como problemas de conexión. No se modifica el estado remoto cuando el servidor rechaza la petición.

## Pruebas

Usar un servidor HTTP de prueba para verificar método, ruta, payload y respuestas exitosas; respuestas 422/404 deben llegar intactas al modelo/vista. Probar que la navegación existente y el sondeo de API mantienen su comportamiento. Ejecutar `go test ./tests/unit/tui/` y `go test ./...`.

## Riesgos y decisiones

- No asumir una ruta de listado de proyectos sin verificar el registro HTTP. Si no hay listado, usar flujo por ID compatible con endpoints existentes.
- Los formularios no deben rechazar entradas que el backend pueda interpretar; solo controles de edición y navegación son responsabilidad local.
