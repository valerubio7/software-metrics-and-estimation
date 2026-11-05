# Manual de usuario

Guía breve para usar la TUI de **Software Metrics & Estimation**. Para instalar
la API, preparar PostgreSQL y compilar el proyecto, ver el
[README](../README.md).

La TUI es la forma más directa de operar el sistema: permite crear proyectos,
cargar el backlog, planificar Sprints, estimar, registrar esfuerzo, seguir
defectos y descargar reportes sin escribir llamadas HTTP a mano.

---

## 1. Qué podés hacer

| Pestaña | Para qué sirve |
|---|---|
| **Dashboard** | Ver el estado del proyecto y sus métricas, y descargar el PDF. |
| **Backlog** | Cargar y mantener las historias, estimarlas y registrar esfuerzo o defectos. |
| **Sprints** | Crear Sprints, asignar historias, descomponerlas en tareas, completarlas y cerrar. |
| **Defectos** | Consultar los defectos del proyecto y actualizar su estado. |

La TUI muestra en pantalla los atajos principales; `?` abre la ayuda completa
con todas las teclas válidas de la pantalla actual.

---

## 2. Antes de empezar

1. La API tiene que estar corriendo y apuntar a una base PostgreSQL migrada.
2. La TUI se conecta a la API indicada en `API_URL` (por defecto,
   `http://localhost:8080`).
3. Al abrirla, la TUI muestra el navegador de proyectos. `Enter` sobre un
   proyecto abre su espacio de trabajo en la pestaña Dashboard.

Si la API no responde, la TUI lo avisa y no permite operar.
Revisá que la API esté levantada y que `API_URL` sea la URL correcta.

---

## 3. Crear un proyecto

1. En **Proyectos**, presioná `n`.
2. Completá el nombre y las fechas de inicio y de fin prevista.
3. `Enter` en el último campo guarda el proyecto.
4. `e` sobre un proyecto resaltado permite editarlo.

La fecha de fin no puede ser anterior a la de inicio.

---

## 4. Cargar el backlog

1. Abrí el proyecto y andá a **Backlog** con `2`.
2. `n` crea una historia: título, descripción, prioridad y criterios de
   aceptación.
3. `e` edita la historia resaltada.
4. `Enter` abre el detalle, donde están las acciones de estimación, esfuerzo y
   defectos.
5. `/` busca; cualquier otro carácter también inicia la búsqueda. `Esc` limpia
   el filtro.

El backlog es la lista priorizada de todo lo que el proyecto necesita, todavía
sin fechas. No hace falta asignar todo a un Sprint desde el principio.

---

## 5. Planificar un Sprint

1. Andá a **Sprints** con `3`.
2. `n` crea un Sprint. Solo se pide el **Sprint Goal**: el objetivo que resume
   qué se quiere lograr en ese período.
3. Con el Sprint resaltado, `a` abre el selector de historias del backlog.
4. Movete con `↑` / `↓`, marcá con `Space` y confirmá con `Enter`.
5. `Enter` sobre el Sprint muestra sus historias.

Un Sprint recién creado queda abierto. Podés asignarle historias mientras esté
abierto.

### Descomponer en tareas

1. Dentro de las historias del Sprint, `Enter` abre el detalle de una historia.
2. `n` agrega una tarea: título y horas estimadas.
3. Las tareas quedan asociadas a esa historia dentro del Sprint.

Las tareas reemplazan la estimación de la historia cuando existen: si una
historia tiene tareas, las horas estimadas se calculan sumando las tareas.

### Completar una historia

* En la lista de historias del Sprint, `c` marca la historia resaltada como
  completada.
* También podés hacerlo desde el detalle de la historia, con `c`.

La finalización registra la fecha en que se completó, no solo el estado.

---

## 6. Estimar con Planning Poker

1. Abrí el detalle de una historia desde **Backlog** o desde **Sprints**.
2. `p` inicia la estimación.
3. Marcá con `Space` los integrantes que participan y confirmá con `Enter`.
4. Cada participante vota en privado con un número de `0` a `100` (o `?` si no
   puede estimar). `Enter` confirma el voto y pasa al siguiente.
5. Cuando todos votaron, `Enter` revela la ronda.
6. Elegí la estimación final y confirmá con `y` o `Enter`.

Los Story Points quedan guardados en la historia y se reflejan en el backlog y
en las métricas.

---

## 7. Registrar esfuerzo

1. Abrí el detalle de una historia y presioná `t`.
2. Elegí la tarea sobre la que trabajaste, o **«Toda la historia (sin tarea)»**.
3. Elegí el integrante.
4. Completá la fecha, las horas y una descripción breve.
5. `Enter` en el último campo registra el esfuerzo.

Reglas de validación:

* Las horas deben ser mayores que cero, hasta 24 y con dos decimales como
  máximo.
* La fecha no puede ser posterior a hoy.
* La descripción admite hasta 500 caracteres.

El esfuerzo es **append-only**: se registra, pero no se edita ni se borra desde
la TUI. Si te equivocaste, registrá el ajuste correspondiente.

---

## 8. Reportar y seguir defectos

### Reportar

1. Abrí el detalle de la historia donde apareció el problema.
2. `d` inicia el alta del defecto. La historia ya viene elegida.
3. Completá descripción, severidad (`alta`, `media`, `baja`) y estado inicial
   (`detectado` o `en_correccion`).
4. Elegí el Sprint donde se detectó.
5. Confirmá con `s`.

El alta se hace desde la historia para no perder el vínculo entre el defecto y
lo que lo originó.

### Consultar y actualizar

1. Andá a **Defectos** con `4`.
2. `Enter` abre el detalle; `u` cambia el estado.
3. Si el estado pasa a `resuelto`, la TUI pide el Sprint de resolución.
4. `/` filtra y `r` refresca.

Un defecto resuelto conserva el Sprint donde se detectó y suma el Sprint donde
se resolvió. Las métricas usan esa distinción: los detectados se cuentan por
Sprint de detección y los resueltos por Sprint de resolución.

---

## 9. Ver métricas y descargar el PDF

### Dashboard

Muestra el estado del proyecto y las métricas principales con barras:

* Story Points planificados y completados.
* Velocidad.
* Horas estimadas y reales, con su desvío.
* Porcentaje de historias completadas.
* Defectos detectados y resueltos.

`s` cambia el ámbito entre **Proyecto completo** y los Sprints abiertos.
`r` refresca. Una métrica sin datos suficientes se muestra como **«sin datos»**,
nunca como cero: el cero es un valor y la falta de información no lo es.

### PDF

* `d` en el Dashboard descarga el reporte del ámbito que estás viendo.
* `d` en Sprints descarga el reporte del Sprint resaltado.

El archivo se guarda en el directorio desde donde se ejecuta la TUI, con el
nombre `reporte-proyecto-{id}.pdf` o `reporte-sprint-{id}.pdf`. Si ya existe, se
reemplaza.

---

## 10. Cerrar un Sprint

1. En **Sprints**, resaltá el Sprint abierto.
2. `x` pide confirmación; `s` confirma y `n` cancela.
3. `h` muestra el historial de Sprints cerrados y permite desplegar sus
   historias.

El cierre es **definitivo** desde la API: no se reabre. Antes de cerrar,
revisá que las historias completadas estén marcadas y que el esfuerzo esté
registrado.

---

## 11. Teclas

| Tecla | Acción |
|---|---|
| `Tab` / `Shift+Tab` | Pestaña siguiente / anterior |
| `1`–`4` | Ir directo a Dashboard, Backlog, Sprints o Defectos |
| `↑` / `↓` | Mover la selección |
| `Enter` | Acción principal de la fila o confirmar |
| `n` | Crear (Proyectos, Backlog y Sprints) |
| `e` | Editar (Proyectos y Backlog) |
| `a` | Asignar historias al Sprint |
| `Space` | Marcar en una multiselección |
| `c` | Completar una historia del Sprint |
| `x` | Cerrar el Sprint resaltado |
| `h` | Ver Sprints cerrados |
| `p` | Estimar con Planning Poker |
| `t` | Registrar esfuerzo |
| `d` | Reportar defecto o descargar PDF, según la pantalla |
| `u` | Actualizar el estado de un defecto |
| `/` | Buscar o filtrar |
| `r` | Refrescar |
| `Esc` | Volver un paso, limpiar el filtro o volver a Proyectos |
| `?` | Abrir la ayuda de la pantalla |
| `q` | Salir (dentro de un campo de texto es una letra más) |
| `Ctrl+C` | Salir desde cualquier pantalla |

Las teclas de pestaña (`Tab`, `1`–`4`) funcionan en la pantalla raíz de cada
pestaña. Dentro de un flujo, `Esc` retrocede un paso y los dígitos se escriben
en el campo o filtro activo.

---

## 12. Glosario

| Término | Qué es |
|---|---|
| **Proyecto** | El trabajo que se estima y se sigue. Agrupa backlog, Sprints y defectos. |
| **Product Backlog** | Lista priorizada de todo lo que el proyecto necesita, sin fechas. |
| **Historia** | Una unidad de valor del backlog, con prioridad y criterios de aceptación. |
| **Sprint** | Un período de trabajo con un objetivo (Sprint Goal) y un conjunto de historias. |
| **Story Points** | Unidad relativa de esfuerzo que se asigna al estimar una historia. |
| **Tarea** | Un paso concreto dentro de una historia del Sprint, con horas estimadas. |
| **Esfuerzo** | Horas reales trabajadas por un integrante en una historia o tarea. |
| **Defecto** | Un problema detectado en una historia, con severidad y estado. |
| **Velocidad** | Story Points completados por Sprint; en el proyecto, el promedio de los Sprints cerrados. |
| **Desvío** | Diferencia entre horas reales y estimadas. |

---

## 13. Problemas comunes

**La TUI dice que la API no está disponible.**
Verificá que la API esté corriendo y que `API_URL` apunte a su dirección. La
barra superior muestra el estado de la conexión.

**No veo un cambio que acabo de hacer.**
Presioná `r` para refrescar la lista o el Dashboard.

**Una métrica dice «sin datos».**
Falta alguna observación: Story Points, horas estimadas o registros de esfuerzo
según la métrica. La TUI no inventa un cero cuando falta información.

**No encuentro cómo crear un defecto en la pestaña Defectos.**
El alta se hace desde el detalle de la historia (`d`). La pestaña Defectos es
para consultar, filtrar y actualizar el estado.

**El PDF no aparece donde lo busco.**
Se guarda en el directorio desde donde se ejecutó la TUI, no en el directorio
del proyecto ni en el home.

**Cerré un Sprint sin querer.**
El cierre es definitivo desde la API. No hay reapertura; si necesitás seguir
trabajando, creá un Sprint nuevo y asignale las historias pendientes.

**`q` no me deja salir.**
Dentro de un campo de texto, `q` es una letra. Usá `Ctrl+C`, que sale desde
cualquier pantalla.

---

## 14. Límites conocidos

* El esfuerzo se registra, pero no se edita ni se elimina desde la TUI.
* Los Sprints cerrados no se reabren.
* La TUI opera sobre un proyecto a la vez; para cambiar de proyecto se vuelve a
  **Proyectos** con `Esc`.
* No hay autenticación ni separación de permisos: todos los usuarios de la API
  ven y modifican los mismos datos.
* Las métricas dependen de la calidad de carga: Story Points, estimaciones y
  esfuerzo tienen que estar registrados para que los agregados no queden «sin
  datos».
