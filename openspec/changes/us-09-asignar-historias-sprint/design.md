# Diseño: Asignar historias a un Sprint (HU-09)

## Objetivo y límites

Añadir una operación para asociar una selección completa de historias existentes con un Sprint existente. La asociación representa planificación: no mueve ni elimina historias del Product Backlog y no modifica campos de historia, incluido su estado. Se rechaza el lote entero ante cualquier selección inválida. La prohibición de asignar a Sprint cerrado usa la decisión autorizada para HU-09: `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`.

El repositorio está organizado como módulos Go verticales (`internal/story/`, `internal/sprint/`), con casos de uso, adaptadores HTTP y PostgreSQL. La tabla `stories` guarda `project_id` y el estado; `sprints` guarda `project_id`, pero hoy no hay tabla de asociaciones. El handler de creación de historias sigue el patrón de ruta anidada, decoder JSON estricto y traducción de errores por capa. El wiring y el gating de rutas se componen en `internal/api/` y `cmd/api/` según disponibilidad de esquema. La migración compartida reside en `internal/project/infrastructure/postgres/migrations/`.

## Decisiones de arquitectura

### Asociación independiente, no cambio de estado

Crear una asociación persistida entre Sprint e historia, en vez de añadir un `sprint_id` como sustituto de la pertenencia al backlog o reutilizar el estado de la historia. Esto permite conservar la relación existente de cada historia con su proyecto/Product Backlog y su estado actual, y hace explícita la semántica de trabajo planificado.

La opción preferida es una tabla de unión nueva, por ejemplo `sprint_stories`, con identificadores de Sprint e historia y una restricción única por par. Se debe preservar la integridad referencial hacia ambas entidades y restringir la eliminación de filas padre en coherencia con las FKs existentes `ON DELETE RESTRICT`. La migración concreta debe ajustarse a la estrategia de migraciones ya usada. No se requiere backfill.

### Lote como unidad transaccional

La aplicación valida la forma básica de la selección (incluida la repetición de IDs) antes de solicitar persistencia. El repositorio realiza la comprobación de elegibilidad y todas las inserciones dentro de una sola transacción: valida existencia del Sprint, historias, pertenencia al proyecto, asociación previa y valor de `is_closed`; solo después inserta el conjunto. Cualquier error provoca rollback completo. No se debe implementar como una sucesión de escrituras independientes ni aceptar inserciones parciales.

La base de datos debe respaldar unicidad de la asociación y evitar referencias huérfanas. La comprobación de proyecto y las escrituras deben compartir la misma transacción. Para carreras entre dos asignaciones concurrentes del mismo par, la restricción única es autoridad final; el conflicto se traduce al mismo rechazo de asociación existente, con rollback de todo el lote. El mecanismo exacto de aislamiento/bloqueo se elige durante implementación según el esquema y las capacidades del controlador; no se debilita la atomicidad.

Para hacer cumplir también que Sprint e historia pertenecen al mismo proyecto sin confiar solamente en la capa HTTP, conviene incluir `project_id` en la fila de asociación y usar claves foráneas compuestas hacia pares `(id, project_id)` de ambas tablas, agregando las restricciones únicas de soporte necesarias si la base actual no las ofrece. Alternativamente, si esa expansión resulta incompatible con las migraciones vigentes, la transacción debe comprobar ambas pertenencias antes de insertar, y la decisión y sus garantías frente a concurrencia deben quedar explícitas. No se permite relajar la regla de mismo proyecto.

### Contrato de aplicación y límites de módulo

Seguir el corte vertical existente: un caso de uso de asignación en `internal/story/application/` o un módulo de aplicación compartido claramente delimitado, con dependencias explícitas de lectura/escritura para historias y Sprint. Evitar que `story` dependa de detalles PostgreSQL de `sprint` o que el transporte consulte directamente repositorios. El comando lleva `SprintID` y una lista de `StoryIDs`; el caso de uso coordina validación y persiste el lote mediante una operación de repositorio de alto nivel que garantice atomicidad.

La operación debe distinguir, como errores de aplicación, al menos Sprint inexistente, historia(s) inexistente(s), proyectos incompatibles, duplicado en la entrada o ya asociado, y Sprint cerrado (`is_closed = true`). No se exponen errores SQL ni se confunden errores de infraestructura con ausencia de recursos.

## Ruta y contrato HTTP

Mantener el patrón de rutas bajo el proyecto: `POST /projects/{project_id}/sprints/{sprint_id}/stories` es la opción recomendada para expresar que el recurso padre es el Sprint del proyecto. El cuerpo recibe una lista de identificadores de historias, por ejemplo `{"story_ids":["…","…"]}`; UUIDs de ruta y elementos se validan en el adaptador. El servidor no permite campos desconocidos y no admite identificadores duplicados. La ruta se registra solo cuando estén disponibles las dependencias de Sprint e historias y el esquema requerido.

Éxito: `200 OK` o `201 Created` con una respuesta estable que confirme Sprint e historias asociadas; escoger un único código y esquema al implementar, sin introducir cambios de estado ni representar información no almacenada. Entrada malformada o campos desconocidos: 400 según el patrón actual; UUID inválido o lista vacía/duplicada: 422 con errores por campo. No encontrados: 404 para Sprint o historia ausente. Conflicto de elegibilidad (proyecto distinto, ya asociado o Sprint cerrado): 409. Error inesperado: 500 genérico, sin detalle interno. La clasificación exacta para lote con más de una clase de error debe ser determinista y no afectar el rollback.

No se debe confundir «duplicado en la selección» con «ya asociado»: ambos rechazan el lote, pero deben poder diagnosticarse claramente en la respuesta interna/aplicación. El contrato público concreto puede usar códigos de error estables compatibles con `errorResponse` de `internal/story/transport/http/handler.go`.

## Flujo de datos

```text
POST con IDs ──> handler: método, JSON estricto y UUIDs
                    │
                    ▼
             caso de uso: validar selección y coordinar
                    │
                    ▼
     repositorio: transacción, comprobar Sprint/historias/proyecto,
                 asociación previa y cierre US-12 si ya está definido
                    │
           todas válidas ──> insertar asociaciones y commit
           cualquier error ──> rollback completo
                    │
                    ▼
          respuesta de éxito/error; historias y estados intactos
```

La existencia y elegibilidad se comprueban como parte de la operación transaccional, no mediante una lectura previa separada que pueda quedar obsoleta. Ninguna ruta de error puede dejar una parte del lote persistida.

## Archivos afectados previstos

| Ruta | Cambio previsto |
|---|---|
| `internal/story/application/` | Añadir comando, errores de aplicación, interfaz de repositorio de asignación y caso de uso para un lote. |
| `internal/story/transport/http/` | Añadir handler de asignación con decodificación estricta y traducción de errores siguiendo los handlers existentes. |
| `internal/story/infrastructure/postgres/` | Implementar operación transaccional, comprobaciones de elegibilidad y rollback; traducir solo errores de constraints relevantes. |
| `internal/project/infrastructure/postgres/migrations/` | Añadir migración reversible para tabla de asociación, claves/índices y restricciones de integridad necesarias. Sin almacenar estado nuevo en historias. |
| `internal/api/` | Componer y registrar la ruta con las dependencias requeridas, conservando rutas existentes cuando la capacidad no esté disponible. |
| `cmd/api/` | Habilitar HU-09 solo si el esquema requerido está limpio; no degradar disponibilidad de proyectos, historias o creación de Sprint ya existente. |
| `tests/unit/story/application/`, `tests/unit/story/transport/http/` | Pruebas de lote, validaciones, contrato HTTP y cero escrituras en rechazos. |
| `tests/integration/` | Pruebas PostgreSQL reales de persistencia, constraints, transacción/rollback y concurrencia relevante. |
| `README.md` | Documentar ruta, prerequisito de migración, solicitud/respuesta y errores una vez fijados durante implementación. |

No modificar historias para simular pertenencia al Sprint ni eliminar su registro del backlog. Las pruebas de integración deben usar PostgreSQL real para validar transacciones y constraints, siguiendo las convenciones del repositorio.

## Fuente de verdad de cierre

Por decisión explícita del usuario para HU-09 (#37), se reemplaza la dependencia previa de US-12 (#40) por `sprints.is_closed BOOLEAN NOT NULL DEFAULT false`, agregada mediante migración reversible `000006`. Sprints existentes quedan abiertos por el default. La operación transaccional lee este valor antes de insertar y rechaza el lote completo si es `true`.

## Pruebas y garantías

- Una y varias historias válidas quedan vinculadas en una operación y conservan proyecto, Product Backlog y estado.
- Sprint ausente, historia ausente, proyecto distinto, ID repetido, asociación previa o cierre conforme a US-12 rechazan toda la selección.
- Un error en cualquier inserción, constraint o commit revierte todos los vínculos del lote.
- Dos solicitudes concurrentes no pueden crear duplicados ni dejar medias asignaciones; la restricción única y la transacción sostienen esta garantía.
- Errores de base de datos no reconocidos se propagan como fallo interno y no se disfrazan de not-found/conflicto.
- La ruta no altera creación/consulta/actualización de historias ni la creación de Sprint.

Las pruebas unitarias cubren validación del comando, traducción HTTP y que el handler no invoque persistencia ante formato inválido. Las pruebas de integración verifican atomicidad en PostgreSQL: preparar un lote donde un elemento falle tras detectar los demás elegibles y comprobar que no queda asociación alguna. Añadir prueba concurrente para la unicidad si el harness de integración permite coordinar ambas transacciones de forma determinista. No se ejecutan pruebas ni se escribe código en esta fase de diseño.

## Despliegue y reversión

Aplicar la migración mediante el mecanismo externo existente antes de habilitar la ruta; el arranque debe comprobar versión limpia que incluya la tabla nueva. No ejecutar migraciones automáticamente desde el servicio. La bajada elimina asociaciones y, por tanto, datos de planificación: no usarla como rollback rutinario de aplicación. Ante reversión de código, conservar tabla y asociaciones salvo aprobación explícita de una reversión destructiva. El rollout debe mantener la ruta ausente mientras el esquema no esté listo.

## Riesgos y decisiones pendientes para implementación

1. **Cierre de Sprint**: resuelto por la decisión autorizada; `is_closed` es la fuente de verdad.
2. **Integridad de mismo proyecto**: confirmar si las FKs compuestas requieren ampliar claves únicas existentes o si la transacción más las FKs actuales ofrece la garantía necesaria sin carrera. Preferir integridad impuesta por base cuando sea compatible.
3. **Respuesta de éxito y clasificación pública de conflictos**: fijar un contrato único al implementar, manteniendo patrones HTTP existentes y sin alterar la semántica de atomicidad.
4. **Revisión de alcance**: conservar la selección single-pr aprobada y el presupuesto de 400 líneas; si el cambio previsto presenta riesgo significativo de excederlo, pausar para decisión bajo `ask-on-risk`, sin inferir estrategia de cadena ni excepción.
