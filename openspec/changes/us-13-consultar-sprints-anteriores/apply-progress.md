# Progreso de aplicación: US-13 — Consultar Sprints anteriores

**Modo**: Strict TDD (`go test ./...`). **Entrega**: `single-pr` con `size:exception` aceptada (un PR, 5 commits, uno por slice). **Cadena**: n/a.

**Lote 1**: slices 1 y 2 (tareas 1.1-1.6 y 2.1-2.6). **Lote 2**: slice 4 (tareas 4.1-4.5, handler HTTP; no depende de Docker). **Lote 3**: slice 3 (tareas 3.1-3.7, `ListClosedByProject` sobre PostgreSQL, con Docker). El slice 5 sigue pendiente.

## Estado acumulado de tareas

| Tarea | Estado | Nota |
|-------|--------|------|
| 1.1 | Hecha | RED sin Docker observado |
| 1.2 | Hecha | GREEN observado |
| 1.3 [DOCKER] | Hecha | Verificada con Docker (lote 3): 5/5 en verde |
| 1.4 [DOCKER] | Hecha | Verificada con Docker (lote 3) |
| 1.5 [DOCKER] | Hecha | Suite completa del paquete 9/9 en verde con `seq` presente |
| 1.6 | Hecha | Sin duplicación que extraer; `go test ./...` sin cambios respecto de la línea base |
| 2.1 - 2.6 | Hechas | Evidencia completa abajo |
| 3.1 - 3.7 | Hechas | Evidencia completa abajo (lote 3, Docker) |
| 4.1 - 4.5 | Hechas | Evidencia completa abajo (lote 2) |
| 5.x | Pendientes | Fuera de este lote |

## Línea base (antes de tocar código)

`go test ./...` en la rama `feat/us-13-consultar-sprints-anteriores` sin cambios: todos los paquetes unitarios en verde. Fallan **solo** los paquetes de integración que requieren Docker, por el entorno (`rootless Docker is not supported on Windows, failed to create Docker provider`): `tests/integration/project/postgres`, `projectmember/postgres`, `sprint/postgres`, `story/postgres` y `task/postgres`. Esa lista se repitió idéntica después de cada slice (diferencia nula salvo la marca `(cached)`). Ninguna falla es atribuible a este cambio. Con `-short`, las pruebas de `sprint/postgres` se saltan (9 saltadas: 4 previas + 5 nuevas); `go test` sin `-short` en este entorno las reporta como **falla de arranque del contenedor**, no como salto.

## Evidencia del ciclo TDD

| Tarea | Archivo de prueba | Capa | Red de seguridad | RED | GREEN | TRIANGULATE | REFACTOR |
|-------|-------------------|------|------------------|-----|-------|-------------|----------|
| 1.1 / 1.2 | `tests/integration/migrations/migration_files_test.go` | Archivos (sin Docker) | ✅ pasaba con 10 pares | ✅ Falló: `canonical sequence must contain exactly eleven migration pairs` y `missing canonical migration version 000011` | ✅ `ok` tras crear el par `000011_*.sql` | ➖ Omitida: tarea estructural con un único resultado posible | ✅ Nada que extraer |
| 1.3 | `tests/integration/sprint/postgres/repository_integration_test.go` | Integración (Docker) | N/A (pruebas nuevas; las existentes no se pudieron ejecutar sin Docker) | ✅ Escritas (identidad `ALWAYS`/`NOT NULL`; constraint `UNIQUE (project_id, seq)` por `pg_constraint`; `seq` creciente; `23505` con `OVERRIDING SYSTEM VALUE`; `428C9` sin `OVERRIDING`) | ⏳ **No ejecutado**: compilan (`go vet ./...`) y con `-short` se saltan; sin contenedor no hay GREEN observado | — | — |
| 1.4 | ídem | Integración (Docker) | — | ✅ Escrita (`TestSprintCreationSequenceMigrationCanBeReversedAndReapplied`: `down` elimina columna y constraint, datos de negocio idénticos por instantánea JSON sin `seq`; `up` asigna `seq` único a las 3 filas) | ⏳ **No ejecutado** | — | — |
| 1.5 | ídem (suite existente) | Integración (Docker) | — | N/A | ⏳ **No ejecutado** | — | — |
| 2.1 / 2.2 | `tests/unit/sprint/domain/sprint_test.go` | Unitaria | ✅ 2 pruebas del archivo en verde | ✅ Falló la compilación: `undefined: domain.PlannedStory` | ✅ `ok` con `ClosedSprint`, `PlannedStory.Completed()` y `SprintHistory` | ✅ 3 casos (nil, marca, `time.Time` cero = marca registrada) | ➖ Nada que extraer |
| 2.3 / 2.4 | `tests/unit/sprint/application/list_sprints_test.go` | Unitaria | N/A (archivo nuevo) | ✅ Falló la compilación: `undefined: application.NewListSprintsUseCase` y `ListSprintsQuery` | ✅ 2 pruebas en verde con la implementación mínima (que ignoraba el error del puerto a propósito) | ver 2.5 | ver 2.6 |
| 2.5 | ídem | Unitaria | ✅ 2 pruebas en verde | ✅ **Falló de verdad**: `TestListSprintsPropagatesPortErrorsWithoutAPartialList` (`Execute() error = <nil>, want project not found` / `want connection reset`) | ✅ `ok` tras propagar el error y devolver `SprintHistory{}` | ✅ 5 identificadores hostiles; mayúsculas -> canónico en puerto y resultado; orden preservado (3 Sprints y 3 historias); `ErrProjectNotFound` e inesperado con Sprints devueltos junto al error | ✅ Ver 2.6 |
| 2.6 | — | — | — | — | ✅ `go test ./...` igual a la línea base | — | ➖ Sin duplicación que justifique extraer |
| 4.1 / 4.2 | `tests/unit/sprint/transport/http/list_handler_test.go` | Unitaria (`httptest`, caso de uso real sobre lector falso) | ✅ 7 pruebas del paquete en verde (create/close) | ✅ Falló la compilación: `undefined: transporthttp.NewListSprintsHandler` | ✅ 12/12 del paquete en verde con el handler mínimo (sin guarda de método ni normalización UTC a propósito) | ✅ 5 pruebas: contenedor y orden del lector; `[]` con `nil` en `sprints` y en `stories` (cuerpo crudo); `422` con cero lecturas; `404` sin clave `sprints`; `500` genérico sin filtrar el error ni la lista | ➖ Ver 4.5 |
| 4.3 | ídem | Unitaria | ✅ 12/12 | ✅ **Falló de verdad**: `completed_at:"2026-10-11T07:30:00-03:00", want ... UTC instant`; las claves exactas (5 por Sprint, 6 por historia), la ausencia de campos prohibidos y `Content-Type` ya pasaban (caracterización del contrato, no fabricado como fallo) | ✅ `ok` tras `utcCopy` (copia UTC, `nil` se mantiene `nil`) | ✅ historia completada (`story_points` 8, UTC) y pendiente (`story_points`, `completed_at` `null`, `completed:false`) | ➖ Ver 4.5 |
| 4.4 | ídem | Unitaria | ✅ 14/14 | ✅ **Falló de verdad**: `POST`/`PUT`/`DELETE`/`PATCH` devolvían `status=200 calls=1` | ✅ `ok` tras la guarda `GET`/`HEAD` | ✅ `HEAD` -> `200` con una lectura; 4 métodos -> `405 method_not_allowed` / `only GET is supported` con cero lecturas | ➖ Ver 4.5 |
| 4.5 | — | — | — | — | ✅ `go test ./tests/unit/...` en verde; `go vet ./...` limpio | — | ➖ Ninguno previsto: `handler.go` no cambió; solo se añadió el helper privado `utcCopy` en `list_handler.go` |

| 3.1 - 3.4 | `tests/integration/sprint/postgres/list_closed_integration_test.go` (archivo nuevo) | Integración (PostgreSQL 16 vía `SPRINT_TEST_DATABASE_URL`) | ✅ 9/9 del paquete en verde antes de tocar nada | ✅ **Falló de verdad** en dos pasos: (a) compilación (`ListClosedByProject undefined`); (b) con un stub `nil, nil`: fallan 8 de 9 funciones (`missing_project`, orden de Sprints, orden de historias, finalización por Sprint, Sprint sin historias, aislamiento, mapeo, error de BD, consultas repetidas); pasan trivialmente `only_open_sprint`, `project_without_sprints` y `DoesNotWrite` | — | — | — | — |
| 3.5 | ídem + `internal/sprint/infrastructure/postgres/repository.go` | Integración | — | — | ✅ 9 funciones (11 pruebas contando subpruebas) en verde con la consulta única (filtro `is_closed` en el `ON`, agrupación por cambio de `sp.id`, cero filas -> `ErrProjectNotFound`, `rows.Err()`, error sin reinterpretar -> `nil`) y la aserción `SprintHistoryReader` | — | — |
| 3.6 | ídem | Integración | ✅ | ✅ Pasó de inmediato (caracterización de la propiedad; no se fabrica un fallo previo) | ✅ 3 Sprints con historia compartida, 3 consultas repetidas idénticas | ✅ Mutación: mover `is_closed` de `ON` a `WHERE` hace fallar `only_open_sprint` y `project_without_sprints` (el riesgo central del diseño queda cubierto); revertida | ➖ |
| 3.7 | — | — | — | — | ✅ `go vet ./...` limpio; paquete `sprint/postgres` completo y `go test ./tests/unit/... ./tests/integration/migrations/...` en verde | — | ➖ Sin duplicación real que extraer; firmas de `Create` y `CloseSprint` intactas |

### Resumen de pruebas

- **Pruebas escritas**: 5 funciones de integración Docker (no ejecutadas), 1 de dominio (3 casos), 5 de aplicación (con subcasos: 5 identificadores hostiles y 2 de error), 8 funciones de handler HTTP (lote 2; 8 funciones con subcasos, 15 funciones en total en el paquete) y la modificación de la prueba de archivos de migración (sin Docker).
- **Capas**: unitaria (dominio, aplicación y transporte), archivos de migración, integración Docker (pendiente).
- **Pruebas de aprobación (refactor)**: ninguna.
- **Funciones puras creadas**: 2 (`PlannedStory.Completed`, `utcCopy`).

## Work Unit Evidence

| Unidad | Comando focal y resultado | Harness de runtime | Frontera de reversión |
|--------|---------------------------|--------------------|-----------------------|
| 1 | `go test ./tests/integration/migrations/...` -> `ok` (de `FAIL` a `ok`). `go test ./tests/integration/sprint/postgres/... -run 'Seq|CreationSequence'` -> **no ejecutado** (requiere Docker); `go vet ./...` limpio y con `-short` las 5 pruebas se saltan | Testcontainers `postgres:16-alpine`: **pendiente de ejecución con Docker** | Revertir el commit 1: quita los dos `.sql` y los cambios de pruebas; `Create`/`CloseSprint` no nombran `seq` |
| 2 | `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application` -> `ok` ambos | N/A: tipos puros y fake del puerto; ninguna ruta expuesta | Revertir el commit 2: elimina tipos de lectura, `list_sprints.go` y sus pruebas; nada depende de ellos |
| 3 | `SPRINT_TEST_DATABASE_URL=... go test ./tests/integration/sprint/postgres/... -run ListClosedByProject -v -count=1` -> de `build failed` a `ok` (11 pruebas); paquete completo `ok`; `go vet ./...` limpio | PostgreSQL 16 en contenedor `pg-us13` (127.0.0.1:55432), no Testcontainers (falla con `unexpected EOF` en esta máquina); el helper crea una base aislada por prueba | Revertir el commit 3: elimina el método, la aserción del puerto y las pruebas; sin efecto en producción (aún no está compuesto) |
| 4 | `go test ./tests/unit/sprint/transport/http` -> de `build failed` a `ok` (15 funciones de prueba del paquete, 8 nuevas); `go test ./tests/unit/...` -> todos `ok`; `go vet ./...` limpio | N/A: `httptest` con caso de uso real sobre lector falso; sin registro en el router, sin Docker | Revertir el commit 4: elimina `list_handler.go` y su prueba; sin ruta registrada, sin efecto |

## Commits

| Slice | Mensaje | Hash |
|-------|---------|------|
| 1 | `feat(sprint): add the sprint creation sequence migration` | `fad7f86` |
| 2 | `feat(sprint): add the list sprints use case` | `30d68cb` (registrado después del commit; ese cambio de `apply-progress.md` se incluyó en el commit 4) |
| 3 | `feat(sprint): list closed sprints with planned stories` | ver `git log` (el hash no puede constar dentro del propio commit) |
| 4 | `feat(sprint): serve the sprint history over HTTP` | `b782ef0` |

El commit 1 incluye además los artefactos de la carpeta `openspec/changes/us-13-consultar-sprints-anteriores/` que estaban sin seguimiento (propuesta, exploración, spec, diseño y tareas).

## Desviaciones del diseño

Ninguna: la implementación coincide con el diseño (Decisiones 2, 4 y 5). Detalles menores:

- Las pruebas de migración de 1.3/1.4 usan una instantánea JSON de las cuatro tablas de negocio (`to_jsonb(row) - 'seq'` para `sprints`) para probar "datos intactos"; es una elección de prueba, no de producción.
- `uuid.Parse` de `google/uuid` también acepta formas no canónicas (sin guiones, con llaves, `urn:uuid:`); el caso de uso las canonicaliza y el puerto recibe siempre la forma con guiones en minúsculas. Es coherente con `ListStoriesUseCase` (US-07).

## Problemas encontrados

- Las pruebas del slice 3 se escribieron en `list_closed_integration_test.go` (archivo nuevo del mismo paquete) en lugar de añadirse a `repository_integration_test.go` como decía 3.1: evita agrandar un archivo de 480 líneas y reutiliza sus helpers. Es una desviación de ubicación, no de contenido.

- En este entorno Windows no hay Docker utilizable: las pruebas de integración existentes de los 5 paquetes `tests/integration/*/postgres` **fallan** al arrancar el contenedor en lugar de saltarse (fallo preexistente de entorno, no del cambio). Con `-short` se saltan las de `sprint/postgres`.
- `gofmt -l` marca todos los archivos del repositorio por los finales de línea CRLF del árbol de trabajo (`autocrlf`); el índice guarda LF. No es un problema introducido por este cambio.

## Frontera de PR / carga de revisión

- Modo: `size:exception` (un único PR, 5 commits, uno por slice).
- Unidad actual: slices 1 y 2 (lote 1), slice 4 (lote 2) y slice 3 (lote 3). Comienzan en `main` (`5b356fb`).
- Líneas aproximadas: commit 1 ~220 de código y pruebas (más los artefactos de openspec); commit 2 ~215; commit 4 ~370 (producción ~95, pruebas ~275).

## Verificación con Docker

Ejecutado en el lote 3 contra PostgreSQL 16 (`pg-us13`, vía `SPRINT_TEST_DATABASE_URL`):

```
go test ./tests/integration/sprint/postgres/... -count=1   # ok (incluye Seq/CreationSequence, 1.3-1.5, y ListClosedByProject)
```

La unidad 1 queda verificada. Testcontainers sigue sin funcionar en esta máquina (`unexpected EOF`), por lo que `go test ./...` sin `SPRINT_TEST_DATABASE_URL` sigue fallando en los paquetes de integración por el entorno (preexistente).
