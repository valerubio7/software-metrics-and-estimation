# Informe de verificación: us-15-registrar-esfuerzo

- Commit verificado: `11457e8`, rama `feat/us-15-registrar-esfuerzo`.
- Modo: TDD estricto, almacén `openspec` con espejo en Engram (`sdd/us-15-registrar-esfuerzo/verify-report`).
- Esta verificación cumple la tarea 7.2 de `tasks.md`. No se modificó código ni pruebas.
- **Veredicto: PASS WITH WARNINGS** (0 CRITICAL, 3 WARNING, 6 SUGGESTION).

## 1. Ejecución real

| Comando | Resultado |
|---|---|
| `go build ./...` | exit 0, sin salida |
| `go vet ./...` | sin hallazgos |
| `go test ./internal/effort/... ./tests/unit/... -count=1` | ok en los 21 paquetes con pruebas (los 4 paquetes de `internal/effort` no tienen pruebas propias: viven en `tests/`) |
| `go test -p 1 ./tests/integration/migrations/... -count=1` | ok (0,8 s), 12 pares |
| `go test -p 1 ./tests/integration/effort/... -count=1 -v` | **FAIL** (213,7 s): 18 de 19 pruebas de primer nivel pasan; `TestEffortHTTPEndToEnd` falló en 2 subpruebas por `wait for PostgreSQL ... unexpected EOF` |
| `go test -p 1 ... -run TestEffortHTTPEndToEnd -count=1 -v` (aislado) | PASS 14/14 (73,3 s) |
| Cobertura unitaria con `-coverpkg=./internal/effort/...` | 84,2 % total; `application`, `domain` y `transport/http` al 100 %; `MapConstraintError` al 100 %; `Create` y `requireRow` al 0 % en unitarias (solo los cubre la integración con Docker) |
| `gofmt` sobre los blobs de HEAD de los 15 `.go` del cambio | ningún archivo sin formato |

El fallo de integración es el mismo `unexpected EOF` registrado en la Fase 7 de `apply-progress.md`: ocurre en `waitForEffortDatabaseReady` (plazo fijo de 10 s), antes de ejecutar código de la aplicación. No hubo ninguna aserción de negocio fallida.

## 2. Matriz de trazabilidad

El spec tiene 17 requisitos y 61 escenarios. Resultado: 59 COMPLIANT, 2 PARTIAL, 0 UNTESTED.

| Requisito (escenarios) | Cobertura principal | Estado |
|---|---|---|
| R1 Registrar sobre historia (3) | Handler (`success_over_story`, `SuccessBodyShape`), aplicación, repositorio (`PersistsRecordOverStory`), HTTP real (`201 ... persists the same data`) | COMPLIANT (el 201 no tiene `created_at`, como se reconcilió con el diseño) |
| R2 Registrar sobre tarea (2) | Handler `success_over_task`, repositorio (`PersistsRecordOverTask`, `CountsTaskEffortOnceForItsStory`), HTTP real | COMPLIANT |
| R3 Integrante del proyecto (3) | Repositorio y HTTP real: integrante inexistente y de otro proyecto | COMPLIANT |
| R4 Tarea de la historia (3) | Repositorio y HTTP real: tarea inexistente, de otra historia y de otro proyecto, sin filas | COMPLIANT |
| R5 Proyecto o historia inexistente o ajena (4) | Repositorio (incluye la precedencia proyecto > historia > integrante > tarea), HTTP real, handler | COMPLIANT |
| R6 Fecha (5) | Aplicación (`UTCClockBoundaries`, 10 bordes, formatos inválidos), dominio (futura, hoy, pasada), handler | COMPLIANT |
| R7 Horas (5) | Dominio (0, negativos, 24,01, 3 decimales, NaN, Inf, válidas), handler, tabla (CHECK) | COMPLIANT (sin prueba HTTP con `hours` ausente o `null`; sí en dominio y aplicación) |
| R8 Descripción (4) | Dominio (recorte, 500 y 501 runas multibyte, vacía, solo espacios), handler, tabla | COMPLIANT |
| R9 Identificadores (4) | Aplicación (UUID, vacío, normalización), handler (rutas inválidas) | COMPLIANT |
| R10 Errores juntos (2) | Aplicación, handler, HTTP real | COMPLIANT |
| R11 Solicitud malformada (4) | Handler (JSON roto, `null`, arreglo, contenido adicional, claves desconocidas, tipos incorrectos), HTTP real | COMPLIANT |
| R12 Independencia del Sprint (3) | Repositorio (`AcceptsClosedSprintAndBacklogStory`), HTTP real | COMPLIANT |
| R13 Duplicados (2) | Aplicación, repositorio (`AllowsExactDuplicates`, `DoesNotCapDailyHours`), HTTP real | COMPLIANT |
| R14 Persistencia (3) | Repositorio, HTTP real (trigger → 500 y 0 filas) | COMPLIANT |
| R15 Integridad en base de datos (6) | Pruebas de tabla (FKs compuestas, CHECK, RESTRICT) y `TestMapConstraintError` | 5 COMPLIANT, 1 **PARTIAL**: el escenario nombra una FK de proyecto que no existe (ver WARNING 2) |
| R16 Migración reversible (3) | `TestEffortMigration*`, pruebas de archivos de migración | COMPLIANT |
| R17 Gate de versión 12 (5) | `TestMigrationReadinessResolvesEffortByVersion`, `TestEffortRoute*`, HTTP real | 4 COMPLIANT, 1 **PARTIAL**: el bloque de `cmd/api/main.go` no tiene prueba ejecutable (ver WARNING 3) |

### Criterios de éxito de `proposal.md` (9)

- 8 cumplidos: registro sobre historia y tarea, validaciones con `422` y `fields`, errores `404`, Sprint cerrado/backlog/duplicados, FKs compuestas, gate `>= 12`, migración reversible con 12 pares.
- 1 PARTIAL: "`go test ./...` pasa". No hay una corrida completa única en verde (ver WARNING 1).

### Criterios del issue #43 (10)

Los 10 están cubiertos. Las lecturas de integrantes y de tareas por historia que menciona el issue ya estaban en `main`.

## 3. Auditoría de TDD estricto

- Existe una tabla "TDD Cycle Evidence" por cada UT (1 a 6).
- GREEN verificado hoy: todos los archivos de prueba citados existen y pasan.
- RED y mutaciones se toman de `apply-progress.md`: no se pueden reproducir desde git porque cada UT es un commit con prueba y código, y no se re-ejecutaron para no tocar el código.
- No se encontró evidencia inventada; `apply-progress.md` declara los casos que pasaron al primer intento y las mutaciones que sobreviven.
- Aserciones sin tautologías ni pruebas humo.
- Commits: UT 1 a 3 en inglés (aceptado), UT 4 con asunto en inglés y cuerpo en español, UT 5, UT 6 y docs en español. Sin `Co-Authored-By` ni atribución de IA. Cuerpos de 4 a 6 líneas, sin salida de `go test` pegada.

## 4. Coherencia diseño/spec del código

Coincide con `design.md` y el spec: ruta `POST /projects/{project_id}/stories/{story_id}/effort-records`; `201` plano con `task_id` siempre presente y sin `created_at`; códigos `400`, `404` (cuatro `*_not_found`), `422` con `fields`, `405` y `500` genérico; migración `000012` idéntica al diseño; gate `version >= 12` después de la salida temprana por `dirty` o error, independiente de `Stories`, `Members` y `Tasks`.

## 5. Hallazgos

### CRITICAL

Ninguno.

### WARNING

1. **Inestabilidad del paquete de integración.** `waitForEffortDatabaseReady` espera 10 s fijos y el `unexpected EOF` de Docker Desktop los agota. El paquete falló en 2 de 3 corridas completas observadas y pasó en la otra y en aislado. El mismo patrón de espera de 10 s está en los paquetes `task`, `sprint`, `project` y `story`, y `story/postgres` también cayó una vez. No es un defecto de negocio. Sin una corrida completa en verde, la tarea 7.1 se cumplió por unión de corridas.
2. **Spec frente a diseño.** No existe FK de proyecto: `MapConstraintError` mapea solo 3 FKs. `ErrProjectNotFound` sale de la verificación transaccional y está probado en repositorio, handler y HTTP real. Hay que ajustar el spec al consolidarlo.
3. **`cmd/api/main.go` sin prueba ejecutable.** El gate `>= 12` de `main.go` solo se verifica por inspección y mediante `ResolveMigrationReadiness`. Es la misma situación que los demás gates del proyecto.

### SUGGESTION

- (a) El round-trip de `work_date` no tiene prueba aislante: `time.Parse` ya rechaza todo lo no canónico. Es código defensivo.
- (b) `FOR KEY SHARE` no tiene prueba aislante; la FK mantiene la integridad y el mapeo `23503` está probado de forma sintética.
- (c) Dos mutaciones de la UT 4 sobreviven por redundancia deliberada con la FK compuesta.
- (e) El `422` para UUID de ruta inválido sigue el handler de `task` y el spec; la decisión 5 decía "solo campos del cuerpo". Documentar la diferencia al archivar.
- (f) El README usa una subsección nueva en lugar de una tabla general.
- Extras: falta una prueba HTTP con `hours` ausente o `null`; una descripción con `U+0000` llegaría a PostgreSQL y probablemente daría `500` (sin confirmar); el tamaño real es de unas 3.328 líneas de autoría contra ~1.650 previstas, cubierto por `size:exception`.

## 6. Estado de tareas observado

Todas las tareas están marcadas salvo 7.4 (sin correcciones detectadas) y 7.5 (cierre posterior, el archivado). La 7.2 queda cumplida con este informe.
