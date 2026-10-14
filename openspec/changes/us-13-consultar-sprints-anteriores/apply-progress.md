# Progreso de aplicación: US-13 — Consultar Sprints anteriores

**Modo**: Strict TDD (`go test ./...`). **Entrega**: `single-pr` con `size:exception` aceptada (un PR, 5 commits, uno por slice). **Cadena**: n/a.

**Lote 1**: slices 1 y 2 (tareas 1.1-1.6 y 2.1-2.6). Los slices 3, 4 y 5 no se tocaron.

## Estado acumulado de tareas

| Tarea | Estado | Nota |
|-------|--------|------|
| 1.1 | Hecha | RED sin Docker observado |
| 1.2 | Hecha | GREEN observado |
| 1.3 [DOCKER] | Escrita, pendiente de ejecución con Docker | Sin casilla marcada: no se observó el resultado |
| 1.4 [DOCKER] | Escrita, pendiente de ejecución con Docker | Sin casilla marcada |
| 1.5 [DOCKER] | Pendiente de ejecución con Docker | No requiere código nuevo |
| 1.6 | Hecha | Sin duplicación que extraer; `go test ./...` sin cambios respecto de la línea base |
| 2.1 - 2.6 | Hechas | Evidencia completa abajo |
| 3.x, 4.x, 5.x | Pendientes | Fuera de este lote |

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

### Resumen de pruebas

- **Pruebas escritas**: 5 funciones de integración Docker (no ejecutadas), 1 de dominio (3 casos), 5 de aplicación (con subcasos: 5 identificadores hostiles y 2 de error) y la modificación de la prueba de archivos de migración (sin Docker).
- **Capas**: unitaria (dominio y aplicación), archivos de migración, integración Docker (pendiente).
- **Pruebas de aprobación (refactor)**: ninguna.
- **Funciones puras creadas**: 1 (`PlannedStory.Completed`).

## Work Unit Evidence

| Unidad | Comando focal y resultado | Harness de runtime | Frontera de reversión |
|--------|---------------------------|--------------------|-----------------------|
| 1 | `go test ./tests/integration/migrations/...` -> `ok` (de `FAIL` a `ok`). `go test ./tests/integration/sprint/postgres/... -run 'Seq|CreationSequence'` -> **no ejecutado** (requiere Docker); `go vet ./...` limpio y con `-short` las 5 pruebas se saltan | Testcontainers `postgres:16-alpine`: **pendiente de ejecución con Docker** | Revertir el commit 1: quita los dos `.sql` y los cambios de pruebas; `Create`/`CloseSprint` no nombran `seq` |
| 2 | `go test ./tests/unit/sprint/domain ./tests/unit/sprint/application` -> `ok` ambos | N/A: tipos puros y fake del puerto; ninguna ruta expuesta | Revertir el commit 2: elimina tipos de lectura, `list_sprints.go` y sus pruebas; nada depende de ellos |

## Commits

| Slice | Mensaje | Hash |
|-------|---------|------|
| 1 | `feat(sprint): add the sprint creation sequence migration` | `fad7f86` |
| 2 | `feat(sprint): add the list sprints use case` | ver `git log` (el hash propio de un commit no puede escribirse dentro de ese mismo commit) |

El commit 1 incluye además los artefactos de la carpeta `openspec/changes/us-13-consultar-sprints-anteriores/` que estaban sin seguimiento (propuesta, exploración, spec, diseño y tareas).

## Desviaciones del diseño

Ninguna: la implementación coincide con el diseño (Decisiones 2, 4 y 5). Detalles menores:

- Las pruebas de migración de 1.3/1.4 usan una instantánea JSON de las cuatro tablas de negocio (`to_jsonb(row) - 'seq'` para `sprints`) para probar "datos intactos"; es una elección de prueba, no de producción.
- `uuid.Parse` de `google/uuid` también acepta formas no canónicas (sin guiones, con llaves, `urn:uuid:`); el caso de uso las canonicaliza y el puerto recibe siempre la forma con guiones en minúsculas. Es coherente con `ListStoriesUseCase` (US-07).

## Problemas encontrados

- En este entorno Windows no hay Docker utilizable: las pruebas de integración existentes de los 5 paquetes `tests/integration/*/postgres` **fallan** al arrancar el contenedor en lugar de saltarse (fallo preexistente de entorno, no del cambio). Con `-short` se saltan las de `sprint/postgres`.
- `gofmt -l` marca todos los archivos del repositorio por los finales de línea CRLF del árbol de trabajo (`autocrlf`); el índice guarda LF. No es un problema introducido por este cambio.

## Frontera de PR / carga de revisión

- Modo: `size:exception` (un único PR, 5 commits, uno por slice).
- Unidad actual: slices 1 y 2. Comienzan en `main` (`5b356fb`) y terminan en el commit 2.
- Líneas aproximadas: commit 1 ~220 de código y pruebas (más los artefactos de openspec); commit 2 ~215.

## Pendiente de ejecución con Docker (no verificado)

```
go test ./tests/integration/sprint/postgres/... -run "Seq|CreationSequence" -v
go test ./tests/integration/sprint/postgres/... -v
```

La unidad 1 no puede declararse verificada hasta que esas dos ejecuciones den verde.
