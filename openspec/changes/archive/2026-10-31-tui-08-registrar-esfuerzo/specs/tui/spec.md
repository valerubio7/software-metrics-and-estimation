# Delta para tui: registrar esfuerzo desde la TUI (TUI-08)

## ADDED Requirements

### Requirement: Elegir historia, tarea e integrante para registrar esfuerzo
La sección Esfuerzo DEBERÁ guiar el registro con los selectores navegables existentes, en este orden: historia del backlog del proyecto activo, tarea de esa historia (o la historia completa) e integrante del proyecto activo. Una vez cargadas las tareas de la historia elegida, el selector de tareas DEBERÁ anteponer, como primera fila, la fila sintética «Toda la historia (sin tarea)», seguida de las tareas en el orden informado por la API; la fila sintética DEBERÁ estar presente también cuando la historia no tiene tareas. Elegir una tarea registra el esfuerzo sobre esa tarea; elegir la fila sintética lo registra sobre la historia completa. Elegir cualquiera de las dos filas DEBERÁ abrir el selector de integrantes del proyecto activo, que en Esfuerzo es de selección simple: `Enter` elige el integrante resaltado y abre el formulario de esfuerzo (Requirement: Completar el formulario de esfuerzo), y `Space` no marca integrantes. Ningún paso DEBERÁ pedir un identificador escrito a mano. El avance entre selectores y la apertura del formulario NO DEBERÁN emitir requests que modifiquen datos. Si el proyecto no tiene integrantes, el selector de integrantes DEBERÁ mostrar un estado vacío propio que informe que el proyecto no tiene integrantes registrados, que NO remita a ninguna opción del menú principal y que permita volver con `Esc` y refrescar con `r`.

#### Scenario: Flujo completo hasta el formulario
- **GIVEN** un proyecto activo con la historia «Login», que tiene la tarea «Diseñar», y el integrante «Ana Pérez»
- **WHEN** el usuario abre Esfuerzo, elige «Login», elige «Diseñar» y elige «Ana Pérez» con `Enter`
- **THEN** la TUI abre el formulario de esfuerzo mostrando la historia «Login», la tarea «Diseñar» y el integrante «Ana Pérez»
- **AND** todas las requests emitidas hasta ese punto son `GET` (historias, tareas e integrantes)
- **AND** la TUI no pidió ningún identificador

#### Scenario: La fila «Toda la historia» encabeza las tareas
- **GIVEN** que la historia elegida tiene las tareas «Diseñar» y «Probar», en ese orden
- **WHEN** la TUI presenta el selector de tareas de Esfuerzo
- **THEN** las filas son, en orden, «Toda la historia (sin tarea)», «Diseñar» y «Probar»

#### Scenario: Una historia sin tareas admite esfuerzo
- **GIVEN** que la API responde HTTP 200 con una lista de tareas vacía para la historia elegida
- **WHEN** la TUI presenta el selector de tareas de Esfuerzo
- **THEN** muestra la fila «Toda la historia (sin tarea)» y no la sustituye por un estado vacío bloqueante
- **AND** `Enter` sobre esa fila abre el selector de integrantes

#### Scenario: Refrescar las tareas conserva la fila sintética
- **GIVEN** el selector de tareas de Esfuerzo cargado
- **WHEN** el usuario presiona `r` y la API responde con las tareas
- **THEN** la TUI vuelve a consultar las tareas de la historia
- **AND** la fila «Toda la historia (sin tarea)» sigue siendo la primera fila

#### Scenario: Si falla la consulta de tareas no se ofrece la fila sintética
- **GIVEN** que la API responde HTTP 500 al consultar las tareas de la historia elegida
- **WHEN** la TUI presenta el selector de tareas
- **THEN** muestra el error de la API dentro de la pantalla sin cerrarse
- **AND** no ofrece la fila «Toda la historia (sin tarea)» ni abre el selector de integrantes
- **AND** `r` vuelve a consultar y, si la API responde, la fila aparece

#### Scenario: Selección simple de integrante
- **GIVEN** el selector de integrantes de Esfuerzo con dos integrantes
- **WHEN** el usuario presiona `Space` y luego `Enter` sobre el segundo integrante
- **THEN** `Space` no marca ninguna fila
- **AND** `Enter` abre el formulario de esfuerzo para el segundo integrante
- **AND** el pie y la ayuda del paso no ofrecen multiselección ni confirmación de participantes

#### Scenario: Proyecto sin integrantes
- **GIVEN** que la API responde HTTP 200 con una lista de integrantes vacía
- **WHEN** la TUI presenta el selector de integrantes de Esfuerzo
- **THEN** muestra un mensaje que indica que el proyecto no tiene integrantes registrados
- **AND** el mensaje no remite a ninguna opción del menú principal
- **AND** `Esc` vuelve al selector de tareas y `r` vuelve a consultar los integrantes

#### Scenario: Falla la consulta de integrantes
- **GIVEN** que la API responde HTTP 500 al consultar los integrantes
- **WHEN** la TUI presenta el selector de integrantes
- **THEN** muestra el error de la API dentro de la pantalla sin cerrarse
- **AND** no abre el formulario de esfuerzo
- **AND** permite reintentar con `r` o volver con `Esc`

### Requirement: Completar el formulario de esfuerzo
El formulario de esfuerzo DEBERÁ ser una pantalla propia, distinta de la pantalla de sección, con tres campos en este orden: «Fecha» (segmentada en año, mes y día), «Actividad» (texto) y «Horas» (texto). DEBERÁ mostrar, solo como lectura, la historia, la tarea (o «Toda la historia») y el integrante elegidos. El foco inicial DEBERÁ estar en «Fecha». La fecha DEBERÁ abrirse con la fecha UTC actual según el reloj de la TUI y ser editable; el ingreso de la fecha usa las mismas reglas que el formulario de proyectos (Requirement: Ingresar las fechas del proyecto como año, mes y día): solo dígitos, avance automático al completar cada segmento, flechas izquierda y derecha entre segmentos, `-` y `/` como separadores, retroceso y relleno con ceros. `Tab` y `↓` DEBERÁN pasar al campo siguiente y `Shift+Tab` y `↑` al anterior, de forma circular; `Enter` en «Fecha» o «Actividad» DEBERÁ pasar al campo siguiente y `Enter` en «Horas» DEBERÁ confirmar el envío. Cada apertura del formulario DEBERÁ comenzar con la fecha de hoy y con «Actividad» y «Horas» vacías. Dentro del formulario `q`, `Q`, `?`, `p` y `r` DEBERÁN tratarse como caracteres de texto en «Actividad» y «Horas»: no cierran la TUI, no abren la ayuda, no cambian el proyecto activo ni refrescan nada; `Ctrl+C` sigue cerrando la TUI. Los campos de texto DEBERÁN aceptar espacios y borrar una runa Unicode completa por pulsación de retroceso. El pie y la ayuda del formulario DEBERÁN mostrar únicamente las teclas válidas en él, sin ofrecer `q`, `p`, `r` ni `?` como acciones.

#### Scenario: La fecha inicial es la fecha UTC actual
- **GIVEN** que el reloj de la TUI indica `2026-10-31T23:30:00-03:00`, equivalente a `2026-11-01T02:30:00Z`
- **WHEN** el usuario abre el formulario de esfuerzo
- **THEN** la fecha muestra año 2026, mes 11 y día 01
- **AND** «Actividad» y «Horas» están vacías
- **AND** el foco está en «Fecha»

#### Scenario: Editar la fecha por segmentos
- **GIVEN** el formulario abierto con el foco en «Fecha»
- **WHEN** el usuario borra la fecha con retroceso y escribe `20261005`
- **THEN** los segmentos muestran año 2026, mes 10 y día 05
- **AND** el foco avanzó solo al llenarse cada segmento

#### Scenario: `Enter` avanza de campo sin enviar
- **GIVEN** el formulario con el foco en «Fecha»
- **WHEN** el usuario presiona `Enter` y luego `Enter` otra vez
- **THEN** el foco pasa a «Actividad» y después a «Horas»
- **AND** no se emite ninguna request

#### Scenario: Navegar con `Tab` y flechas de forma circular
- **GIVEN** el formulario con el foco en «Horas»
- **WHEN** el usuario presiona `Tab`
- **THEN** el foco vuelve a «Fecha»
- **AND** `Shift+Tab` desde «Fecha» lleva el foco a «Horas»

#### Scenario: Las teclas globales son caracteres en los campos de texto
- **GIVEN** el formulario con el foco en «Actividad»
- **WHEN** el usuario escribe `q`, `Q`, `?`, `p` y `r`
- **THEN** «Actividad» contiene `qQ?pr`
- **AND** la TUI sigue abierta en el formulario, la ayuda no se abrió y no se emitió ninguna request
- **AND** lo mismo ocurre con esos caracteres en «Horas»

#### Scenario: Espacios y retroceso por carácter en «Actividad»
- **GIVEN** el formulario con el foco en «Actividad»
- **WHEN** el usuario escribe «Revisión de diseño» y presiona retroceso una vez
- **THEN** el campo contiene «Revisión de diseñ»
- **AND** el valor es UTF-8 válido

#### Scenario: El pie del formulario no ofrece teclas ajenas
- **GIVEN** el formulario de esfuerzo activo
- **WHEN** la TUI muestra el pie
- **THEN** el pie incluye `Tab`, `Enter` y `Esc`
- **AND** no ofrece `q` para salir, `p` para cambiar de proyecto, `r` para refrescar ni `?` para la ayuda

#### Scenario: El formulario de proyectos conserva su ingreso de fechas
- **GIVEN** el formulario de creación de proyectos
- **WHEN** el usuario completa las fechas por segmentos, deja una incompleta o ingresa una fecha fin anterior a la de inicio
- **THEN** se mantienen los comportamientos del Requirement: Ingresar las fechas del proyecto como año, mes y día, sin cambios

### Requirement: Validar localmente la fecha y las horas del esfuerzo
Antes de emitir cualquier request, la TUI DEBERÁ validar únicamente que (a) la fecha esté completa y sea una fecha real del calendario y (b) «Horas», una vez recortados los espacios de los extremos, sea un número finito escrito con punto decimal y mayor que 0. La coma decimal NO se acepta. Si alguna validación falla, la TUI DEBERÁ mostrar el motivo en el formulario, nombrando el campo, conservar todos los valores escritos y NO emitir ninguna request; si fallan ambos campos se informa primero «Fecha». El error local DEBERÁ limpiarse al editar un campo. La TUI NO DEBERÁ validar localmente el máximo de horas, la cantidad de decimales, la longitud ni la ausencia de «Actividad», ni que la fecha sea futura: esas reglas las aplica la API y la TUI muestra su motivo. El integrante y la historia siempre están elegidos al llegar al formulario porque el flujo no permite abrirlo sin ellos.

#### Scenario: Fecha incompleta
- **GIVEN** el formulario con el día de la fecha vacío y «Horas» igual a `2`
- **WHEN** el usuario confirma desde «Horas»
- **THEN** la TUI muestra en el formulario que la fecha está incompleta
- **AND** no emite ninguna request y conserva los valores escritos

#### Scenario: Fecha imposible
- **GIVEN** el formulario con la fecha 2026-02-30 y «Horas» igual a `2`
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que la fecha no es válida
- **AND** no emite ninguna request

#### Scenario: Horas vacías
- **GIVEN** el formulario con fecha válida y «Horas» vacía
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request

#### Scenario: Horas no numéricas
- **GIVEN** el formulario con fecha válida y «Horas» igual a `abc`
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request y «Horas» conserva `abc`

#### Scenario: Coma decimal rechazada
- **GIVEN** el formulario con fecha válida y «Horas» igual a `1,5`
- **WHEN** el usuario confirma
- **THEN** la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request

#### Scenario: Horas igual a cero o negativas
- **GIVEN** el formulario con fecha válida
- **WHEN** el usuario confirma con «Horas» igual a `0` y luego con «Horas» igual a `-2`
- **THEN** en ambos casos la TUI muestra que las horas deben ser mayores que 0
- **AND** no emite ninguna request

#### Scenario: Horas no finitas
- **GIVEN** el formulario con fecha válida
- **WHEN** el usuario confirma con «Horas» igual a `NaN` y luego con «Horas» igual a `Inf`
- **THEN** en ambos casos la TUI muestra que las horas deben ser un número mayor que 0
- **AND** no emite ninguna request

#### Scenario: Con fecha y horas inválidas se informa primero la fecha
- **GIVEN** el formulario con la fecha incompleta y «Horas» igual a `abc`
- **WHEN** el usuario confirma
- **THEN** el error mostrado corresponde a la fecha
- **AND** no se emite ninguna request

#### Scenario: El error local se limpia al corregir
- **GIVEN** el formulario mostrando el error de horas por «Horas» igual a `0`
- **WHEN** el usuario edita «Horas»
- **THEN** el error deja de mostrarse
- **AND** al confirmar con `2` la TUI emite la request de registro

#### Scenario: El rango y los decimales los decide la API
- **GIVEN** el formulario con fecha válida, «Actividad» con texto y «Horas» igual a `25`
- **WHEN** el usuario confirma
- **THEN** la TUI emite la request con `hours` igual al número 25 sin rechazarla localmente
- **AND** lo mismo ocurre con «Horas» igual a `1.234`

#### Scenario: La actividad vacía y la fecha futura los decide la API
- **GIVEN** el formulario con «Actividad» vacía, «Horas» igual a `2` y una fecha posterior a la fecha UTC actual
- **WHEN** el usuario confirma
- **THEN** la TUI emite la request con `description` vacía y esa fecha, sin rechazarla localmente

### Requirement: Enviar el registro de esfuerzo y presentar los errores de la API
Al confirmar con datos válidos, la TUI DEBERÁ emitir exactamente una request `POST /projects/{pid}/stories/{sid}/effort-records`, con el proyecto activo y la historia elegida, y un cuerpo JSON cuyas claves son `member_id`, `work_date` (normalizada como `YYYY-MM-DD`), `description` (tal como fue escrita, sin recortar ni validar), `hours` (número JSON) y `task_id` únicamente cuando se eligió una tarea; al elegir «Toda la historia (sin tarea)» la clave `task_id` DEBERÁ omitirse por completo, no enviarse como `null`. Mientras el `POST` está en curso la TUI DEBERÁ indicar que está guardando e ignorar `Enter` y `Esc`, de modo que no se emita una segunda request ni se abandone el formulario antes de conocer el resultado. Solo la respuesta HTTP 201 comunica éxito (Requirement: Mostrar el esfuerzo registrado y las métricas del proyecto). Ante cualquier otra respuesta de error o ante un error de transporte, la TUI DEBERÁ permanecer en el formulario con todos los valores escritos, dejar de indicar el guardado, mostrar el motivo y NO comunicar éxito. Los errores de la API (HTTP 400, 404, 422 con `fields`, 500 y cualquier otro código no exitoso, incluido 409 si alguna vez se devolviera) DEBERÁN presentarse con las reglas del Requirement: Presentar los errores de la API del backlog. Un error de transporte DEBERÁ informar el problema de conexión y que no se pudo confirmar el registro, y aconsejar verificar antes de reintentar, porque la API no es idempotente. Tras un error el usuario puede corregir y confirmar de nuevo, lo que emite una nueva request.

#### Scenario: Registrar esfuerzo sobre una tarea
- **GIVEN** el formulario del integrante «Ana Pérez» (`member_id` `m-1`) para la tarea `t-1` de la historia `s-1` del proyecto `p-1`, con fecha 2026-10-05, «Actividad» «Implementar login» y «Horas» `3.5`
- **WHEN** el usuario confirma desde «Horas»
- **THEN** la TUI emite una única request `POST /projects/p-1/stories/s-1/effort-records`
- **AND** el cuerpo contiene exactamente `member_id` `m-1`, `task_id` `t-1`, `work_date` `2026-10-05`, `description` `Implementar login` y `hours` igual al número 3.5

#### Scenario: Registrar esfuerzo sobre la historia completa omite `task_id`
- **GIVEN** que el usuario eligió la fila «Toda la historia (sin tarea)» y completó el formulario con datos válidos
- **WHEN** confirma
- **THEN** el cuerpo enviado contiene `member_id`, `work_date`, `description` y `hours`
- **AND** el cuerpo no contiene la clave `task_id`

#### Scenario: La actividad se envía sin recortar
- **GIVEN** el formulario con «Actividad» igual a `  Implementar login  ` y datos válidos
- **WHEN** el usuario confirma
- **THEN** el cuerpo enviado contiene `description` igual a `  Implementar login  `

#### Scenario: El doble `Enter` emite una sola request
- **GIVEN** el formulario con datos válidos y el foco en «Horas»
- **WHEN** el usuario presiona `Enter` dos veces antes de que llegue la respuesta del `POST`
- **THEN** la TUI emite exactamente una request de registro
- **AND** mientras espera muestra un indicador de guardado

#### Scenario: `Esc` se ignora mientras se guarda
- **GIVEN** un `POST` de esfuerzo en curso
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI permanece en el formulario con el indicador de guardado
- **AND** al llegar la respuesta se presenta su resultado

#### Scenario: HTTP 422 con campos inválidos
- **GIVEN** que la API responde HTTP 422 con `fields` que detalla `hours` y `description`
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra cada campo y su motivo como un elemento de lista
- **AND** conserva fecha, «Actividad» y «Horas» tal como se escribieron
- **AND** no comunica éxito ni deja el guardado activo

#### Scenario: HTTP 400 por solicitud inválida
- **GIVEN** que la API responde HTTP 400 con `invalid_request` y un mensaje
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra el código HTTP y el detalle de la API
- **AND** conserva los valores escritos y no comunica éxito

#### Scenario: HTTP 404 por recurso inexistente
- **GIVEN** que la API responde HTTP 404 con `member_not_found` (o `task_not_found`, `story_not_found`, `project_not_found`)
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra un mensaje genérico de recurso inexistente y el detalle informado por la API
- **AND** conserva los valores escritos y no comunica éxito

#### Scenario: HTTP 500
- **GIVEN** que la API responde HTTP 500 con `internal_error`
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra el código HTTP y el detalle de la API
- **AND** conserva los valores escritos y no comunica éxito

#### Scenario: Otro código no exitoso se presenta de forma genérica
- **GIVEN** que la API responde HTTP 409 con un cuerpo JSON de error
- **WHEN** la TUI presenta el resultado
- **THEN** el formulario muestra el código 409 y el detalle de la API, sin comunicar éxito

#### Scenario: Error de conexión
- **GIVEN** que no se puede establecer conexión con la API o que la request excede el tiempo de espera
- **WHEN** falla el `POST` de esfuerzo
- **THEN** el formulario informa el problema de conexión
- **AND** indica que no se pudo confirmar el registro y aconseja verificar antes de reintentar
- **AND** conserva los valores escritos y la TUI sigue abierta

#### Scenario: Corregir y reenviar tras un rechazo
- **GIVEN** el formulario mostrando el error de un HTTP 422 con los valores conservados
- **WHEN** el usuario corrige «Horas» y confirma de nuevo
- **THEN** la TUI emite una nueva request de registro con los valores corregidos
- **AND** si la API responde HTTP 201 se presenta la pantalla de resultado

### Requirement: Mostrar el esfuerzo registrado y las métricas del proyecto
Ante HTTP 201 la TUI DEBERÁ pasar a una pantalla de resultado propia que comunique que el esfuerzo quedó registrado y muestre el registro devuelto por la API (fecha de trabajo, horas y actividad tal como los devolvió la API) junto con la historia, la tarea (o «Toda la historia») y el integrante elegidos. De inmediato DEBERÁ consultar `GET /projects/{pid}/metrics` del proyecto activo y mostrar «Horas reales» (`actual_hours`) y «Desviación» (`deviation`) tal como los devuelve la API, con «sin datos» cuando el valor es nulo y con el mismo formato de la vista de métricas de la sección. La TUI NO DEBERÁ sumar, calcular ni derivar horas ni desvíos. Mientras la consulta está en curso la pantalla DEBERÁ mostrar un estado de carga de las métricas. La tecla `r` DEBERÁ volver a consultar las métricas y DEBERÁ ignorarse mientras una consulta está en curso. Si la consulta de métricas falla (HTTP no exitoso, incluido 404 cuando el esquema de la base no habilita la ruta, error de transporte o cuerpo no interpretable), la pantalla DEBERÁ indicar que el esfuerzo quedó registrado y que las métricas no están disponibles, con el motivo, y NO DEBERÁ mostrar un error de registro; `r` reintenta. La pantalla de resultado DEBERÁ tratar `q` como salida, `?` como ayuda, `p` como cambio de proyecto activo y `Esc` como vuelta (Requirement: Recorrer hacia atrás el flujo de esfuerzo y descartar respuestas tardías), y su pie y ayuda DEBERÁN mostrar solo esas teclas más `r`.

#### Scenario: Registro exitoso con métricas
- **GIVEN** que la API responde HTTP 201 con un registro de fecha `2026-10-05`, horas `3.5` y actividad `Implementar login`
- **AND** que `GET /projects/p-1/metrics` responde HTTP 200 con `actual_hours` 7.5 y `deviation` -2.5
- **WHEN** la TUI presenta el resultado
- **THEN** la pantalla comunica que el esfuerzo quedó registrado y muestra fecha `2026-10-05`, horas 3.5 y actividad `Implementar login`
- **AND** muestra la historia, la tarea o «Toda la historia» y el integrante elegidos
- **AND** muestra «Horas reales» 7.5 h y «Desviación» -2.5
- **AND** no muestra ningún error

#### Scenario: Estado de carga de las métricas
- **GIVEN** que la API respondió HTTP 201 y la consulta de métricas aún no respondió
- **WHEN** la TUI presenta el resultado
- **THEN** la pantalla muestra el registro y un estado de carga de las métricas

#### Scenario: Los valores del registro son los de la API
- **GIVEN** un formulario enviado con «Actividad» `  Implementar login  ` y que la API responde HTTP 201 con `description` `Implementar login`
- **WHEN** la TUI presenta el resultado
- **THEN** la actividad mostrada es `Implementar login`

#### Scenario: La TUI no calcula horas ni desvíos
- **GIVEN** que se registraron 3.5 horas y que la API de métricas responde `actual_hours` 10 y `deviation` 4
- **WHEN** la TUI presenta el resultado
- **THEN** muestra «Horas reales» 10 h y «Desviación» 4 exactamente como los devolvió la API
- **AND** no muestra ninguna suma ni diferencia calculada por la TUI

#### Scenario: Métricas nulas
- **GIVEN** que la API de métricas responde HTTP 200 con `actual_hours` y `deviation` nulos
- **WHEN** la TUI presenta el resultado
- **THEN** «Horas reales» y «Desviación» muestran «sin datos»

#### Scenario: Refrescar las métricas con `r`
- **GIVEN** la pantalla de resultado con `actual_hours` 7.5 mostrado y la consulta terminada
- **WHEN** el usuario presiona `r` y la API de métricas responde ahora `actual_hours` 9
- **THEN** la TUI emite otra `GET /projects/p-1/metrics`
- **AND** la pantalla muestra «Horas reales» 9 h
- **AND** no se emite ningún `POST`

#### Scenario: `r` se ignora mientras hay una consulta en curso
- **GIVEN** la pantalla de resultado con la consulta de métricas en curso
- **WHEN** el usuario presiona `r`
- **THEN** la TUI no emite otra consulta de métricas

#### Scenario: Las métricas fallan con HTTP 404 tras un registro exitoso
- **GIVEN** que `POST .../effort-records` respondió HTTP 201 y `GET /projects/p-1/metrics` responde HTTP 404
- **WHEN** la TUI presenta el resultado
- **THEN** la pantalla indica que el esfuerzo quedó registrado y que las métricas no están disponibles, con el motivo de la API
- **AND** no muestra un error de registro ni sugiere volver a registrar
- **AND** `r` permite reintentar la consulta

#### Scenario: Las métricas fallan por error de conexión o HTTP 500
- **GIVEN** que el registro respondió HTTP 201 y la consulta de métricas falla por error de conexión (o responde HTTP 500)
- **WHEN** la TUI presenta el resultado
- **THEN** indica que el esfuerzo quedó registrado y que las métricas no están disponibles
- **AND** no muestra un error de registro

#### Scenario: Reintentar las métricas con éxito
- **GIVEN** la pantalla de resultado indicando métricas no disponibles
- **WHEN** el usuario presiona `r` y la API de métricas responde HTTP 200
- **THEN** la pantalla muestra «Horas reales» y «Desviación» y deja de indicar que no están disponibles

#### Scenario: Respuesta 201 con cuerpo no interpretable
- **GIVEN** que la API responde HTTP 201 con un cuerpo que no es un registro de esfuerzo interpretable
- **WHEN** la TUI presenta el resultado
- **THEN** la pantalla comunica que el esfuerzo quedó registrado y muestra el cuerpo recibido tal cual
- **AND** no muestra un error de registro y consulta las métricas

#### Scenario: Teclas de la pantalla de resultado
- **GIVEN** la pantalla de resultado de esfuerzo
- **WHEN** el usuario presiona `q`
- **THEN** la TUI se cierra con código 0
- **AND** el pie y la ayuda muestran `r`, `Esc`, `p`, `?` y `q`, y no muestran teclas de edición del formulario

### Requirement: Recorrer hacia atrás el flujo de esfuerzo y descartar respuestas tardías
`Esc` DEBERÁ retroceder un nivel por vez en el flujo de Esfuerzo: del formulario al selector de integrantes, del selector de integrantes al de tareas, del selector de tareas al de historias y del selector de historias al menú. Cada retroceso DEBERÁ restaurar el selector anterior tal como estaba al elegir su fila (filas, cursor y filtro), sin emitir requests; en un selector con filtro aplicado, el primer `Esc` limpia el filtro y el siguiente retrocede. Desde la pantalla de resultado, `Esc` DEBERÁ volver al selector de tareas de la misma historia, restaurado del mismo modo, para registrar otro esfuerzo. Salir del formulario con `Esc` descarta lo escrito. Cambiar el proyecto activo desde un selector o desde la pantalla de resultado DEBERÁ descartar el estado del flujo (historia, tarea, integrante, formulario, registro y métricas) y reabrir Esfuerzo en el selector de historias del nuevo proyecto. La TUI DEBERÁ descartar sin efecto toda respuesta que ya no corresponda a la pantalla, el paso o la consulta activos: una respuesta de tareas o de integrantes tras haber retrocedido, un resultado de guardado sin un guardado en curso y una respuesta de métricas que no sea la de la consulta más reciente de la pantalla de resultado activa.

#### Scenario: `Esc` recorre formulario, integrantes, tareas e historias
- **GIVEN** el formulario de esfuerzo abierto desde los selectores de historia, tarea e integrante
- **WHEN** el usuario presiona `Esc` tres veces
- **THEN** la TUI pasa sucesivamente al selector de integrantes, al selector de tareas y al selector de historias
- **AND** no emite ninguna request en esos retrocesos
- **AND** un cuarto `Esc` vuelve al menú

#### Scenario: El filtro del selector de tareas se conserva
- **GIVEN** que el usuario filtró las tareas con `dis` y eligió «Diseñar»
- **WHEN** retrocede desde el formulario al selector de integrantes y de allí al selector de tareas
- **THEN** el selector de tareas muestra el filtro `dis` y el cursor sobre «Diseñar», sin consultar de nuevo
- **AND** el siguiente `Esc` limpia el filtro y recién el que sigue vuelve al selector de historias

#### Scenario: Tras el éxito, `Esc` vuelve a las tareas de la misma historia
- **GIVEN** la pantalla de resultado de un esfuerzo registrado sobre la historia «Login»
- **WHEN** el usuario presiona `Esc`
- **THEN** la TUI muestra el selector de tareas de «Login» con la fila «Toda la historia (sin tarea)» y sus tareas
- **AND** no emite ninguna request
- **AND** otro `Esc` lleva al selector de historias

#### Scenario: Un segundo registro empieza con el formulario inicial
- **GIVEN** que el usuario volvió al selector de tareas tras un registro exitoso
- **WHEN** elige otra fila y un integrante
- **THEN** el formulario se abre con la fecha de hoy y «Actividad» y «Horas» vacías

#### Scenario: Respuesta tardía de integrantes tras retroceder
- **GIVEN** que el usuario eligió una tarea y la consulta de integrantes sigue en curso
- **WHEN** presiona `Esc` y después llega la respuesta de integrantes
- **THEN** la TUI sigue mostrando el selector de tareas sin cambios
- **AND** la respuesta se descarta

#### Scenario: Respuesta tardía de tareas tras retroceder
- **GIVEN** que el usuario eligió una historia y la consulta de tareas sigue en curso
- **WHEN** presiona `Esc` y después llega la respuesta de tareas
- **THEN** la TUI sigue mostrando el selector de historias sin cambios
- **AND** la respuesta se descarta

#### Scenario: Respuesta de métricas tardía tras salir del resultado
- **GIVEN** la pantalla de resultado con la consulta de métricas en curso
- **WHEN** el usuario presiona `Esc` y después llega la respuesta de métricas
- **THEN** la TUI sigue mostrando el selector de tareas sin cambios
- **AND** la respuesta se descarta

#### Scenario: Respuesta de métricas de un registro anterior
- **GIVEN** que el usuario registró un esfuerzo, volvió con `Esc` antes de que respondieran las métricas y registró un segundo esfuerzo cuya consulta de métricas está en curso
- **WHEN** llega la respuesta de la primera consulta con `actual_hours` 3 y después la de la segunda con `actual_hours` 8
- **THEN** la pantalla de resultado nunca muestra «Horas reales» 3 h
- **AND** termina mostrando «Horas reales» 8 h

#### Scenario: Resultado de guardado sin guardado en curso
- **GIVEN** que no hay un `POST` de esfuerzo en curso
- **WHEN** llega un resultado de guardado
- **THEN** la TUI lo descarta sin cambiar de pantalla ni de contenido

#### Scenario: Cambiar el proyecto descarta el flujo
- **GIVEN** la pantalla de resultado de un esfuerzo registrado en el proyecto `p-1`
- **WHEN** el usuario presiona `p` y confirma el proyecto `p-2`
- **THEN** la TUI abre Esfuerzo en el selector de historias de `p-2`
- **AND** la pantalla no conserva el registro, las métricas ni las selecciones de `p-1`

## MODIFIED Requirements

### Requirement: Navegación de solo lectura
La navegación entre el menú y las pantallas DEBERÁ ser de solo lectura y NO modificará ningún dato del sistema. Solo las operaciones explícitas de escritura pueden emitir requests que modifiquen datos: la creación y modificación de proyectos (Requirement: Operar proyectos mediante la TUI), la creación y modificación de historias (Requirement: Crear historias mediante la TUI y Requirement: Modificar historias desde la lista), las operaciones de los Sprints (Requirement: Operar los Sprints mediante la TUI), el guardado del Story Points acordado en Planning Poker (Requirement: Estimar historias con Planning Poker), que emite un `PUT`, y el registro de esfuerzo (Requirement: Enviar el registro de esfuerzo y presentar los errores de la API), que emite un `POST`. Las secciones Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard, sus selectores y sus pantallas de aviso DEBERÁN emitir únicamente consultas `GET`, salvo el `PUT` de Planning Poker al confirmar el Story Points y el `POST` de Esfuerzo al confirmar el formulario de registro.
(Previously: el texto afirmaba que las secciones Planning Poker y Esfuerzo emitían únicamente consultas `GET` y solo reconocía como escrituras las operaciones de proyectos e historias; ya no contemplaba el `PUT` de Planning Poker (TUI-07), las operaciones de Sprints ni el `POST` de Esfuerzo (TUI-08).)

#### Scenario: Navegar no muta
- **WHEN** el usuario navega entre el menú y las pantallas sin confirmar una operación explícita de escritura
- **THEN** no se emite ninguna request que modifique datos

#### Scenario: Consultar el backlog no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario abre la lista del backlog o cambia de proyecto
- **THEN** las únicas requests emitidas son las consultas `GET` (del catálogo de proyectos y del backlog) y no se emite ningún `POST` ni `PUT`

#### Scenario: Navegar por las secciones del Sprint 3 no muta
- **GIVEN** un proyecto activo
- **WHEN** el usuario recorre Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard, elige filas y avanza entre selectores, sin confirmar el guardado de Story Points ni el envío del formulario de esfuerzo
- **THEN** todas las requests emitidas son `GET`
- **AND** no se crea, modifica ni elimina ningún dato

#### Scenario: Abrir y completar el formulario de esfuerzo no muta hasta confirmar
- **GIVEN** que el usuario llegó al formulario de esfuerzo desde los selectores de historia, tarea e integrante
- **WHEN** edita los campos, navega entre ellos y retrocede con `Esc` sin confirmar desde «Horas»
- **THEN** no se emite ningún `POST` ni `PUT`

#### Scenario: Solo la confirmación del formulario emite el `POST` de esfuerzo
- **GIVEN** el formulario de esfuerzo con datos válidos
- **WHEN** el usuario confirma desde «Horas»
- **THEN** la única request que modifica datos es un `POST` a `/projects/{pid}/stories/{sid}/effort-records`
- **AND** las demás requests de la sección, incluida la consulta de métricas posterior, son `GET`

#### Scenario: Planning Poker solo escribe al confirmar el Story Points
- **GIVEN** una ronda de Planning Poker con participantes y votos
- **WHEN** el facilitador vota, revela y elige un valor sin confirmar el guardado
- **THEN** no se emite ninguna request que modifique datos
- **AND** al confirmar se emite únicamente el `PUT /projects/{pid}/stories/{sid}/story-points`

### Requirement: Elegir integrantes, historias, tareas y Sprints con selectores navegables
La TUI DEBERÁ ofrecer selectores de integrante, historia, tarea y Sprint como listas navegables con el mismo patrón de las colecciones existentes: la búsqueda se activa con `/` o al empezar a escribir, `↑`/`↓` mueven la selección, `Enter` ejecuta la acción primaria, `Esc` limpia el filtro y luego vuelve, `r` refresca y `?` abre la ayuda. Ningún selector DEBERÁ pedir un identificador escrito a mano. Los selectores DEBERÁN mostrar un estado de carga, un estado vacío y el error de la API sin cerrar la TUI. El selector de tareas de Esfuerzo es la excepción del estado vacío: antepone la fila «Toda la historia (sin tarea)» (Requirement: Elegir historia, tarea e integrante para registrar esfuerzo), por lo que ante una historia sin tareas muestra esa fila e indica que la historia no tiene tareas, en lugar de un estado vacío bloqueante.
(Previously: todos los selectores mostraban un estado vacío ante una colección vacía, sin excepción para las tareas de Esfuerzo.)

#### Scenario: Buscar en un selector
- **GIVEN** que un selector muestra una colección
- **WHEN** el usuario presiona `/` o empieza a escribir
- **THEN** la lista se reduce a las coincidencias sin emitir requests
- **AND** `Esc` limpia el filtro antes de volver

#### Scenario: Elegir una fila
- **GIVEN** que un selector tiene una fila resaltada
- **WHEN** el usuario presiona `Enter`
- **THEN** la TUI ejecuta la acción primaria de esa fila
- **AND** no solicita ningún UUID

#### Scenario: Refrescar un selector
- **GIVEN** que un selector muestra datos de la API
- **WHEN** el usuario presiona `r`
- **THEN** la TUI vuelve a consultar la colección del paso actual
- **AND** conserva el filtro escrito

#### Scenario: Selector vacío
- **GIVEN** que la API responde una colección vacía en un selector distinto del de tareas de Esfuerzo (por ejemplo, el de integrantes)
- **WHEN** la TUI presenta el selector
- **THEN** muestra un estado vacío claro
- **AND** permite volver o refrescar sin cerrarse

#### Scenario: El selector de tareas de Esfuerzo no queda vacío
- **GIVEN** que la API responde una lista de tareas vacía para la historia elegida en Esfuerzo
- **WHEN** la TUI presenta el selector de tareas
- **THEN** muestra la fila «Toda la historia (sin tarea)» e indica que la historia no tiene tareas
- **AND** permite volver o refrescar sin cerrarse

#### Scenario: Los cuatro tipos de selector son navegables
- **GIVEN** que el usuario recorre Planning Poker, Esfuerzo y Defectos
- **WHEN** elige una historia, un integrante, una tarea y un Sprint
- **THEN** cada colección se presenta como una lista navegable con las teclas comunes
- **AND** la selección de una fila no emite requests que modifiquen datos

### Requirement: Avisar el límite de las pantallas de navegación
Las pantallas de navegación de TUI-06 NO implementan los flujos de negocio de las secciones que todavía no tienen su historia de TUI integrada: hoy, Defectos (TUI-09) y Dashboard (TUI-11). Cuando el usuario completa la selección de navegación de una de esas secciones, la TUI DEBERÁ mostrar un aviso que indique en qué historia estará disponible el flujo, sin comunicar que la operación se guardó. Planning Poker (TUI-07), Esfuerzo (TUI-08) y Métricas tienen su propio flujo y NO DEBERÁN mostrar este aviso.
(Previously: el aviso cubría Planning Poker, Esfuerzo, Defectos, Métricas y Dashboard y remitía a TUI-07 a TUI-11; Planning Poker ya no lo muestra desde TUI-07, Métricas presenta sus valores desde la consulta de solo lectura y Esfuerzo deja de mostrarlo con TUI-08.)

#### Scenario: Aviso al completar una selección
- **GIVEN** que el usuario eligió los elementos de navegación de Defectos o de Dashboard
- **WHEN** la TUI alcanza el final del flujo disponible
- **THEN** muestra un aviso con la historia que implementará el flujo (TUI-09 o TUI-11)
- **AND** no comunica éxito de una operación de negocio
- **AND** permite volver con `Enter` o `Esc`

#### Scenario: Esfuerzo ya no muestra el aviso
- **GIVEN** que el usuario eligió una historia, una tarea (o «Toda la historia») y un integrante en Esfuerzo
- **WHEN** la TUI alcanza el final del flujo de selección
- **THEN** abre el formulario de esfuerzo
- **AND** no muestra ningún aviso que mencione TUI-08 ni «No se guardó ningún dato»
