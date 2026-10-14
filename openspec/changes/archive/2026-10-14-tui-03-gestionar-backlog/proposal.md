# Propuesta: gestionar el backlog desde la TUI

## Motivo

La opción «Backlog» del menú de la TUI todavía muestra un marcador de posición (TUI-03). El backend ya expone los contratos para consultar el backlog de un proyecto y para crear y modificar historias de usuario (US-05, US-06, US-07), por lo que el usuario sigue necesitando la API directa para operar el backlog. Esta propuesta incorpora esos flujos en la TUI, que continúa siendo un cliente HTTP sin reglas de negocio.

Además, la TUI-02 dejó dos defectos que impiden completar formularios de texto: la tecla `q` cierra la aplicación aunque el usuario esté escribiendo en un campo, y la barra espaciadora (`tea.KeySpace`) se descarta, por lo que no se pueden ingresar espacios. Ambos bloquean la carga de títulos, descripciones y criterios de aceptación y se corrigen dentro de este cambio.

## Cambios

- Reemplazar el marcador TUI-03 de la opción «Backlog» por el flujo funcional; la pantalla alcanzada desde el menú conserva el nombre `backlog`.
- Solicitar el identificador del proyecto una sola vez, conservarlo durante la sesión y ofrecer la acción «Cambiar proyecto».
- Consultar el backlog (`GET /projects/{pid}/stories`) y presentar las historias en el orden informado por la API; `story_points` se muestra solo como lectura y con «—» cuando es nulo.
- Crear historias (`POST /projects/{pid}/stories`) con título, descripción, prioridad y criterios de aceptación.
- Modificar historias (`PUT /projects/{pid}/stories/{sid}`) seleccionándolas de la lista cargada, con el formulario precargado. Como el `PUT` es un reemplazo total, la TUI envía siempre las seis claves (`title`, `description`, `priority`, `status`, `acceptance_criteria`, `estimated_hours`), incluida `estimated_hours` con valor `null` cuando queda vacía.
- Prioridad y estado se ingresan como texto libre; la validación de sus valores permitidos queda en el backend.
- `estimated_hours` admite solo un análisis numérico local mínimo (vacío equivale a `null`); no se replican reglas de negocio.
- Los criterios de aceptación se ingresan en una sola línea separada por `;` y se envían como lista.
- Presentar los errores de la API: HTTP 422 con el detalle de `fields` como lista formateada; HTTP 404 con un mensaje genérico de recurso inexistente más el detalle de la API; errores de transporte sin cerrar la TUI.
- Corregir el manejo de teclado heredado de la TUI-02: `q` solo sale desde el menú, las listas y las pantallas de resultado; `Ctrl+C` sale desde cualquier pantalla; los campos de texto aceptan `tea.KeySpace`; el retroceso borra una runa completa y no un byte (la TUI-02 corrompía caracteres como «á» o «ñ» al borrar). Cada corrección tiene su propia prueba en RED.

## Enfoque

- Nuevo archivo `internal/tui/backlog.go` con un `BacklogClient` (operaciones `List`, `Create`, `Update`) y un mensaje `BacklogResultMsg` etiquetado con la operación (`Op`), siguiendo el patrón de `internal/tui/projects.go`.
- Nuevas pantallas: `backlog-project` (identificador del proyecto), `backlog` (lista seleccionable), `backlog-form` (alta y modificación con precarga) y `backlog-result`.
- El ajuste de `q` y de la barra espaciadora se aplica en `internal/tui/model.go` y beneficia también a los formularios de proyectos.
- TDD estricto: pruebas en RED confirmadas en commits separados de la implementación en GREEN.

## Capacidades

### Capacidades nuevas

- Ninguna: los flujos extienden la capacidad existente `tui`.

### Capacidades modificadas

- `tui`:
  - «Mostrar menú base de navegación»: la opción backlog pasa a dar acceso a la gestión del backlog; solo sprints queda sin pantallas funcionales.
  - «Salir limpiamente de la TUI»: `q` sale solo desde el menú, las listas y las pantallas de resultado; dentro de un formulario de texto se trata como un carácter. `Ctrl+C` sale siempre.
  - «Navegación de solo lectura»: además de las operaciones de proyectos, la creación y modificación explícitas de historias pueden emitir requests que modifiquen datos.
  - Nuevo requisito «Operar el backlog mediante la TUI»: consultar, crear y modificar historias, presentación de errores 422/404/transporte.
  - Nuevo requisito de edición en formularios de texto: los campos aceptan espacios.
  - Nuevo requisito de edición en formularios de texto: el retroceso borra por runa y mantiene el valor en UTF-8 válido.

## Impacto

- Cambios en `internal/tui/model.go` y `internal/tui/styles.go` (si corresponde), nuevo `internal/tui/backlog.go` y pruebas en `tests/unit/tui/` (por ejemplo `tests/unit/tui/backlog_test.go` y ampliaciones de `tests/unit/tui/model_test.go`).
- Actualización de la especificación de la capacidad `tui` mediante una spec delta en el cambio.
- Sin cambios en endpoints, contratos del backend, persistencia ni reglas de dominio.
- Fuera de alcance: listado de proyectos para elegir el identificador (no existe endpoint), edición de `story_points`, descomposición en tareas, gestión de sprints (TUI-04/TUI-05), validación local de prioridad o estado, y cualquier lógica de negocio dentro de la TUI.
- Entrega: un único PR con etiqueta `size:exception`; el pronóstico es de 800 a 950 líneas modificadas, por encima del presupuesto de 400.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| El `PUT` de reemplazo total pierde datos si falta alguna clave o se omite `estimated_hours` | Media | Precarga desde la historia seleccionada y prueba que verifica el cuerpo con las seis claves, incluido `null` |
| El separador `;` impide criterios de aceptación que contengan ese carácter | Baja | Limitación documentada; se recortan espacios y se descartan elementos vacíos |
| El cambio del manejo de `q` altera el comportamiento de los formularios de proyectos | Media | Pruebas RED específicas por pantalla y conservación de las pruebas existentes de salida |
| Rutas de historias no registradas si faltan migraciones (405/404 del mux) | Baja | Se presentan como error de la API sin cerrar la TUI |
| PR de gran tamaño dificulta la revisión | Media | `size:exception` acordado, commits RED/GREEN separados y archivo `backlog.go` aislado |

## Plan de reversión

Revertir el commit de merge del PR. El cambio se limita a `internal/tui/` y `tests/unit/tui/`; no hay migraciones ni cambios de contrato. Revertirlo restaura el marcador TUI-03 y el manejo de teclado previo, sin afectar datos.

## Dependencias

- Endpoints de historias del backend (US-05, US-06, US-07) disponibles con sus migraciones aplicadas.
- TUI-01 y TUI-02 integradas en `main`.

## Criterios de éxito

- [ ] Desde «Backlog» el usuario ingresa un identificador de proyecto, lo conserva en la sesión y puede cambiarlo.
- [ ] La TUI lista las historias del proyecto en el orden de la API, con `story_points` de solo lectura («—» si es nulo).
- [ ] Crear una historia envía título, descripción, prioridad y criterios de aceptación y presenta el resultado de la API.
- [ ] Modificar una historia precarga el formulario y envía las seis claves del reemplazo total.
- [ ] Los errores 422 muestran `fields` como lista; los 404 muestran un mensaje de inexistencia más el detalle de la API; los errores de transporte no cierran la TUI.
- [ ] Escribir `q` y espacios en un campo de texto no cierra la TUI y se refleja en el campo; `Ctrl+C` sale desde cualquier pantalla.
- [ ] Borrar con retroceso tras un carácter acentuado («á», «ñ») elimina el carácter completo y deja el valor en UTF-8 válido.
- [ ] La prueba existente que espera la pantalla `backlog` tras «Abajo + Enter» sigue pasando, y `go test ./...` termina en verde.
