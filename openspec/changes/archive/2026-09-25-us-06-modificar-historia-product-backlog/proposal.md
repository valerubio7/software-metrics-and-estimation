# Propuesta: US-06 — Modificar una historia del Product Backlog

Fuente del requisito: [issue #34 — "US-06 — Modificar una historia del Product Backlog"](https://github.com/valerubio7/software-metrics-and-estimation/issues/34).
Artefacto de entrada: `openspec/changes/us-06-modificar-historia-product-backlog/exploration.md`.

## Intención

US-05 solo permite crear historias. Una vez que una historia existe en el Product Backlog, su contenido
queda congelado: un error de tipeo en el título, una repriorización, un conjunto refinado de criterios de
aceptación o un cambio de estado no pueden registrarse en ningún lado del sistema. Hoy `status` está fijo
en `pendiente` en el dominio y no existe ningún campo de estimación de esfuerzo, así que el backlog no
puede reflejar trabajo iniciado o terminado, y las horas estimadas no tienen dónde vivir.

Este cambio agrega la operación de modificación que falta, para que una historia conocida de un proyecto
conocido pueda actualizarse in situ, con la misma rigurosidad de validación que la creación, e introduce
los dos datos que el issue requiere para esa operación: un conjunto real de `status` y un valor nullable
de horas estimadas. El éxito significa que quien gestiona una historia puede corregir o evolucionar
cualquier campo editable de una historia existente y leer el resultado almacenado, mientras que la
identidad (`id`, `project_id`) y la estimación en Story Points, que pertenece a US-14, no se tocan.

## Alcance

### Dentro del alcance

Mapeado al alcance del issue #34:

- Una operación de modificación para una historia existente de un proyecto existente: `PUT
  /projects/{project_id}/stories/{story_id}`, que devuelve `200` con la historia completa almacenada.
- Reemplazo completo de los seis campos editables, con las seis claves obligatorias en el cuerpo:
  `title`, `description`, `priority`, `status`, `acceptance_criteria`, `estimated_hours` (este último
  puede ser `null`).
- Reutilización de las reglas de validación de contenido de US-05 para `title`, `description`,
  `priority` y `acceptance_criteria`, aplicadas antes de cualquier acceso a la base.
- Un conjunto cerrado de `status`: `pendiente`, `en_progreso`, `completada`. Cualquier valor del conjunto
  puede reemplazar a otro; sin reglas de transición. Se exige en el dominio y con un `CHECK` en la
  migración `000003`.
- Una estimación nullable: `estimated_hours`, un número `> 0` con como máximo 2 decimales y
  `<= 99999.99`; `null` la borra; `0` nunca es válido. Se almacena como
  `estimated_hours NUMERIC(7,2) NULL CHECK (> 0)`, se expone como JSON `estimated_hours` y se representa
  en Go como `*float64`.
- Inmutabilidad de `id`, `project_id` y `story_points`: esta operación nunca los escribe y los rechaza
  como campos desconocidos del cuerpo.
- Superficie de errores: `400 invalid_request` (JSON malformado, tipos incorrectos, campos desconocidos
  incluyendo `id`, `project_id`, `story_points`), `404 story_not_found` (historia inexistente o de otro
  proyecto), `422 validation_failed` con `fields` (identificadores inválidos y violaciones de dominio),
  `500 internal_error`, `405` para otros métodos sobre la ruta del ítem.
- Persistencia con una única sentencia `UPDATE ... WHERE id = $story AND project_id = $project ...
  RETURNING`; cero filas afectadas significa "no encontrada".
- Exposición de la nueva ruta condicionada a `version >= 3 && !dirty` del esquema; la creación mantiene
  su gate `>= 2`.
- Tests (unitarios e integración con Testcontainers), documentación en el README de la operación y su
  migración, y el delta de la spec `historia`.

### Fuera del alcance

- Control de concurrencia. Sin versión, ETag ni `FOR UPDATE`: gana la última escritura, y esto es un
  comportamiento documentado, no un defecto aceptado (Q7).
- Modificación parcial (`PATCH`) y la semántica de "cambiar un solo campo".
- Leer una historia individual o listar el Product Backlog (US-07).
- Estimación en Story Points y Planning Poker (US-14); `story_points` solo se devuelve en la respuesta,
  nunca se escribe aquí.
- Asignación a Sprint (US-09) y borrado de historias.
- Cualquier guarda que acople las horas estimadas a los Story Points: son independientes (Q6).
- Ampliar el conjunto de `status` más allá de los tres valores confirmados y cualquier regla de máquina
  de estados.
- Elegir la estrategia de entrega encadenada (ver Entrega más abajo): esa decisión le corresponde al
  usuario.

## Capacidades

### Capacidades nuevas

Ninguna. La operación de modificación pertenece a la capacidad existente `historia`.

### Capacidades modificadas

- `historia`: hoy especifica solo la creación y un estado inicial `pendiente` con Story Points `null`.
  Los requisitos cambian en cuatro aspectos: (1) debe existir una operación de modificación para una
  historia existente de un proyecto existente, que reemplaza el contenido editable; (2) `status` pasa a
  ser un conjunto cerrado de tres valores en lugar de un valor inicial fijo, sin reglas de transición;
  (3) una historia gana un valor nullable de horas estimadas, positivo cuando está presente, distinto de
  los Story Points y borrable; (4) la identidad de la historia y su asociación al proyecto son
  inmutables, y una historia que no existe bajo el proyecto indicado debe rechazarse sin modificar nada.
  Los requisitos de creación existentes siguen vigentes; la historia registrada ahora también lleva la
  estimación, inicialmente ausente.

## Enfoque

Enfoque A de la exploración: **PUT con reemplazo completo, validar primero, un único `UPDATE ...
RETURNING`, sin lectura previa.**

1. **Dominio.** Agregar `EstimatedHours *float64` a `Story`, introducir el conjunto permitido de `status`
   y agregar un constructor `NewStoryUpdate` que valida los seis campos editables y devuelve el
   `ValidationError{Fields}` existente. Los chequeos de contenido compartidos con `NewStory` se extraen a
   un helper no exportado en lugar de duplicarse, para que creación y modificación no puedan divergir.
2. **Aplicación.** Un nuevo `update_story.go` con `UpdateStoryCommand`, `UpdateStoryUseCase` y
   `ErrStoryNotFound`. El puerto es una interfaz **separada** `StoryUpdater` en lugar de una extensión de
   `StoryRepository`: el repositorio Postgres implementa ambas, la creación sigue independiente de la
   migración `000003` y los fakes de test existentes siguen compilando. La validación corre antes de
   tocar el puerto, así que un comando inválido produce cero llamadas al repositorio y ningún
   identificador generado.
3. **Persistencia.** La migración `000003_add_story_estimated_hours` agrega la columna `NUMERIC(7,2)`
   nullable con su `CHECK` de positividad y el `CHECK` de `status`, igual que ya está restringida
   `priority`. `Update` emite una sola sentencia parametrizada acotada por `id` y `project_id` y devuelve
   la fila almacenada, incluyendo el `project_id` y el `story_points` intactos; `pgx.ErrNoRows` se
   traduce a `ErrStoryNotFound`. Acotar por ambos identificadores hace que una edición cruzada entre
   proyectos sea indistinguible de una historia inexistente y elimina cualquier ventana de carrera entre
   verificación y escritura.
4. **HTTP.** Un `UpdateStoryHandler` atado a la ruta del ítem, con decodificación estricta
   (`DisallowUnknownFields` más la guarda de null en criterios), detección de claves obligatorias y el
   mapa de errores anterior. Los helpers de decodificación/JSON hoy privados del handler de creación se
   comparten, y `storyResponse` gana `estimated_hours` — aditivo, así que la respuesta de creación de
   US-05 pasa a emitir `"estimated_hours": null`.
5. **Composición y docs.** `internal/api/api.go` registra `PUT /projects/{project_id}/stories/{story_id}`;
   `cmd/api/main.go` lo registra solo con el esquema en `version >= 3 && !dirty`, de modo que en versión 2
   la creación sigue funcionando y la ruta de modificación simplemente no existe. El README y el delta de
   la spec `historia` documentan la operación, el conjunto de `status`, las reglas de la estimación y la
   limitación de última escritura gana.

El texto de los deltas de spec sigue la convención existente en español de `openspec/specs/historia/spec.md`.
Los documentos de planificación de esta carpeta de cambio se redactan en español por pedido del usuario
(esto reemplaza el `technical_artifacts_language: en` de `openspec/config.yaml` para estos `.md`).

### Entrega y TDD

TDD estricto activo (`go test ./...`). Cada slice de abajo se entrega como un **commit de unidad de
trabajo** con Conventional Commits, con sus tests y su documentación en el mismo commit, y un mensaje
que registra la evidencia del ciclo TDD de esa unidad (RED — el test que falla y la falla observada;
GREEN — la implementación mínima y el pase observado; TRIANGULATE — los casos adicionales que fijan el
comportamiento; REFACTOR — el cambio estructural y la corrida en verde observada). Sin evidencia
fabricada: un test de integración saltado por falta de Docker se registra como saltado, nunca como
prueba.

El pronóstico (~900–1300 líneas cambiadas: ~300 código, ~600 tests, ~20 SQL, ~60 docs) **excede el
presupuesto de revisión de 400 líneas**. La estrategia de entrega es `ask-on-risk` y **la estrategia de
cadena no fue elegida**. Esta propuesta no la elige.

- `Decision needed before apply: Yes` — el usuario debe elegir `stacked-to-main` o
  `feature-branch-chain` (US-05 usó `stacked-to-main`) antes de empezar la implementación.
- Límites candidatos de PR, en orden, cada uno una unidad de trabajo autocontenida con su propia
  verificación y un rollback que no remueve nada más:
  1. **Dominio** (~250) — `NewStoryUpdate`, conjunto de estados, `EstimatedHours`, helper de validación
     compartido.
  2. **Aplicación** (~200) — `UpdateStoryCommand`, `StoryUpdater`, `ErrStoryNotFound`, caso de uso.
  3. **Almacenamiento** (~300–350, requiere Docker) — migración `000003` y `Update`, incluyendo el
     round-trip `NUMERIC` → `*float64`, el borrado a `NULL` y las aserciones de `project_id` /
     `story_points` intactos.
  4. **Handler HTTP** (~300) — `UpdateStoryHandler`, helpers compartidos de decodificación/respuesta,
     `estimated_hours` en `storyResponse`.
  5. **Composición y docs** (~250) — registro de la ruta, gate `version >= 3`, README y delta de spec.

## Áreas afectadas

| Área | Impacto | Descripción |
|------|---------|-------------|
| `internal/story/domain/story.go` | Modificado | `EstimatedHours *float64`, conjunto permitido de `status`, helper de validación compartido extraído de `NewStory`, `NewStoryUpdate`. |
| `internal/story/application/update_story.go` | Nuevo | `UpdateStoryCommand`, puerto `StoryUpdater`, `ErrStoryNotFound`, `UpdateStoryUseCase`. |
| `internal/story/infrastructure/postgres/repository.go` | Modificado | `Update` con un `UPDATE ... WHERE id AND project_id ... RETURNING`; `pgx.ErrNoRows` → `ErrStoryNotFound`. |
| `.../migrations/000003_add_story_estimated_hours.{up,down}.sql` | Nuevo | Columna `NUMERIC(7,2)` nullable con `CHECK` de positividad más el `CHECK` de `status`; la migración down borra la columna y sus datos. |
| `internal/story/transport/http/handler.go` (o un `update_handler.go` hermano) | Modificado | `UpdateStoryHandler`, helpers compartidos de decodificación/JSON, `estimated_hours` agregado a `storyResponse`. |
| `internal/api/api.go` | Modificado | `StoryDependencies` gana el updater; registra `PUT /projects/{project_id}/stories/{story_id}`. |
| `cmd/api/main.go` | Modificado | Ruta de modificación condicionada a `version >= 3 && !dirty`; la creación se queda en `>= 2`. |
| `tests/unit/story/**`, `tests/unit/cmd/api/main_test.go` | Modificado/Nuevo | Tests de dominio, aplicación, handler y composición; se preservan las aserciones `405` existentes sobre la ruta de colección. |
| `tests/integration/story/postgres/{repository,http}_integration_test.go` | Modificado | Agregar `000003` a la lista de migraciones (`repository_integration_test.go:371`) y cubrir los caminos de actualización de punta a punta. |
| `README.md` | Modificado | Documentar la operación, el conjunto de `status`, las reglas de la estimación, la migración requerida y la limitación de última escritura gana. |
| `openspec/specs/historia/spec.md` | Modificado (al archivar) | Delta de spec con los requisitos de modificación producidos en la fase de spec. |
| `POST /projects`, dominio de proyecto, semántica de Story Points | Sin cambios | Sin cambio de comportamiento. |

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|--------|--------------|------------|
| El conjunto de `status` (`pendiente`, `en_progreso`, `completada`) se inventó en la exploración; hoy solo existe `pendiente`. Un conjunto equivocado implica una migración de datos posterior y un cambio de `CHECK`. | Media | Aceptado por el product owner para este cambio; los valores viven en una constante de dominio y un `CHECK` de migración, así que corregirlos es una migración de seguimiento acotada y no un cambio en todo el código. |
| Orden de despliegue: la ruta de modificación requiere la migración `000003`; revertirla borra las estimaciones almacenadas. | Media | El registro de la ruta está condicionado a `version >= 3 && !dirty`, así que un despliegue sin migrar simplemente carece de la ruta en lugar de fallar en runtime; el plan de rollback exige revisar si hay estimaciones almacenadas antes. |
| El round-trip de pgx `NUMERIC(7,2)` → `*float64` no está verificado y puede perder precisión o fallar al escanear. | Media | Se verifica primero en el test RED de almacenamiento (bordes `0.01` y `99999.99`); alternativa documentada a `DOUBLE PRECISION` dentro de la misma migración si el round-trip falla. |
| Agregar `estimated_hours` al `storyResponse` compartido cambia la respuesta de creación de US-05. | Alta | El cambio es aditivo (`"estimated_hours": null`); los tests de creación, el README y el delta de spec se actualizan en los mismos slices que lo introducen. |
| Extraer la validación compartida de `NewStory` toca código de US-05 protegido solo por los tests existentes. | Media | Se hace como paso REFACTOR explícito tras GREEN, con toda la suite de creación en verde antes y después; sin cambio de comportamiento en el mismo paso. |
| Agregar `Update` a la interfaz `StoryRepository` existente rompería todos los fakes actuales. | Baja | Se usa un puerto `StoryUpdater` separado; el repositorio Postgres implementa ambas interfaces. |
| Actualizaciones perdidas: dos modificaciones concurrentes se pisan en silencio. | Media | Fuera de alcance de forma explícita y documentado como última escritura gana; no se introduce versionado ni se lo insinúa. |
| El pronóstico excede el presupuesto de revisión de 400 líneas. | Alta | Ya hay cinco slices candidatos de PR definidos; `ask-on-risk` exige que el usuario elija la estrategia de cadena antes de aplicar. |
| La cobertura de integración depende de Docker/Testcontainers. | Media | Los tests de integración saltados se reportan como saltados y nunca se cuentan como verificación de la persistencia. |

## Plan de rollback

- **Antes de aplicar la migración `000003` en cualquier entorno**: revertir los commits de los slices (o
  los PR mergeados) en orden inverso. Nada más depende de ellos; la extracción del helper de dominio es el
  único punto compartido y preserva el comportamiento, así que la creación sigue funcionando.
- **Después de exponer la ruta**: primero dejar de exponerla. Revertir el slice de composición, o correr
  contra una base en versión de esquema 2, quita `PUT /projects/{project_id}/stories/{story_id}` y deja la
  creación intacta; mientras tanto el esquema puede quedar en versión 3.
- **Rollback de esquema**: ejecutar la migración down `000003` solo tras confirmar que ninguna historia
  tiene un `estimated_hours` que valga la pena conservar; la migración down borra la columna y esos datos
  no se recuperan. Exportar antes las filas afectadas si existen. Quitar el `CHECK` de `status` no es
  destructivo, pero las filas ya guardadas con `en_progreso` o `completada` sobrevivirían con un valor que
  el código revertido no entiende, así que esas filas deben volver a `pendiente` de forma deliberada,
  nunca en silencio.
- Nunca borrar historias ni proyectos como parte de un rollback, y nunca tocar datos de proyectos de
  US-01.

## Dependencias

- US-05 (crear una historia) está archivada y aporta la tabla `stories`, la estructura de módulos y la
  validación que se reutiliza aquí.
- La migración PostgreSQL `000003` debe aplicarse antes de que la ruta esté disponible.
- Docker es necesario para los slices de integración con Testcontainers.
- El usuario debe elegir una estrategia de cadena (`stacked-to-main` o `feature-branch-chain`) antes de
  aplicar, porque el pronóstico excede el presupuesto de revisión.

## Criterios de éxito

- [ ] Una solicitud de modificación con los seis campos editables para una historia existente del
      proyecto indicado guarda el nuevo contenido y devuelve `200` con la historia completa, con el orden
      de los criterios de aceptación preservado.
- [ ] `id`, `project_id` y `story_points` nunca se modifican, y un cuerpo que contenga alguno de ellos se
      rechaza con `400 invalid_request`.
- [ ] Una historia inexistente, o de un proyecto distinto, devuelve `404 story_not_found` sin modificar
      ninguna fila.
- [ ] `title` o `description` vacíos, una `priority` fuera de `alta|media|baja`, un `status` fuera de
      `pendiente|en_progreso|completada`, criterios de aceptación inválidos o vacíos, una clave
      obligatoria faltante y un `estimated_hours` fuera de rango (`0`, negativo, más de 2 decimales,
      `> 99999.99`) devuelven `422 validation_failed` con `fields` y no escriben nada.
- [ ] `estimated_hours` hace round-trip por PostgreSQL en los bordes `0.01` y `99999.99`, y `null` borra un
      valor almacenado previamente.
- [ ] La ruta no existe con el esquema en versión 2 y está disponible en versión 3, con la creación
      funcionando en ambos casos.
- [ ] `go test ./...` pasa, y cada slice aterriza como un commit de unidad de trabajo cuyo mensaje
      Conventional Commit registra su evidencia RED / GREEN / TRIANGULATE / REFACTOR, con tests y docs
      incluidos.
- [ ] El README y el delta de spec `historia` documentan la operación, el conjunto de `status`, las
      reglas de la estimación y la limitación de última escritura gana.
