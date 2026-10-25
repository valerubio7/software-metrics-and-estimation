# Tareas: Registrar un defecto detectado durante el desarrollo (US-16)

## Convención de commits

- Cada Unidad de Trabajo (UT) cierra con **un único commit** en la rama `feat/us-16-registrar-defecto`, con Conventional Commits. Cada commit contiene pruebas, código y documentación de su unidad, compila y deja `go test ./...` en verde (las pruebas de integración se ejecutan con Docker disponible).
- **Regla de mensaje de commit (obligatoria para `sdd-apply`)**:
  - Asunto en Conventional Commits (los asuntos de cada UT están fijados abajo).
  - Cuerpo de **5 a 8 líneas como máximo**, con **una línea compacta** por cada fase: `RED: <1-2 nombres de prueba clave> falla/n (<motivo corto>)`, `GREEN: <1-2 nombres de prueba clave> pasa/n`, `REFACTOR: <cambio o "sin cambios de comportamiento"> (<resultado>)`. Se puede agregar una línea de `TRIANGULATE` y una de contexto.
  - **Sin salida pegada** de `go test`, sin logs ni trazas. La evidencia detallada (comandos, salidas, duración) vive en `openspec/changes/us-16-registrar-defecto/apply-progress.md`.
  - Sin `Co-Authored-By` ni atribución de IA.
- **Evidencia de apply (obligatoria)**: al cerrar cada UT, se crea o actualiza `openspec/changes/us-16-registrar-defecto/apply-progress.md` (una sección por UT con RED/GREEN/TRIANGULATE/REFACTOR observados, comandos ejecutados y resultados, hash del commit). El archivo se agrega al commit de la UT correspondiente o al commit de documentación final si la UT lo deja pendiente.
- **Estrategia de entrega**: un solo PR (`delivery_strategy: single-pr`) con `size:exception` ya pedido por el usuario (decisión 5 de la propuesta). No se diseñan PRs encadenados.
- **TDD estricto**: cada ciclo es RED (prueba que falla y se observa fallar) → GREEN (código mínimo) → TRIANGULATE (casos adicionales que fuerzan generalizar o cubren bordes) → REFACTOR (sin cambiar comportamiento). Comando de referencia: `go test ./...`.
- Idioma: documentación y comentarios de prueba en español neutro; identificadores, rutas, SQL y códigos HTTP en inglés.

## Review Workload Forecast

| Campo | Valor |
|-------|-------|
| Estimated changed lines | ~1.750–2.100 (suma por UT: ~1.900; ver desglose abajo) |
| 400-line budget risk | High |
| Chained PRs recommended | Yes (el presupuesto de 400 líneas se excede ~5 veces; el usuario pidió `size:exception` y un PR único, por lo que no se aplica la cadena) |
| Suggested split | PR único con `size:exception`, revisable commit a commit (6 commits de código) |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: size-exception
400-line budget risk: High

**Aclaración sobre la decisión**: `Decision needed before apply` es `No` porque el usuario ya pidió explícitamente `size:exception` para un PR único (decisión 5 de la propuesta); no queda ninguna decisión pendiente antes de `sdd-apply`. `Chained PRs recommended` se mantiene en `Yes` por honestidad frente al presupuesto (volumen ~5x), pero la recomendación está **resuelta por excepción pedida**, no por una cadena. La mayor parte del volumen son pruebas (unitarias e integración).

### Líneas estimadas por Unidad de Trabajo

| UT | Commit | Contenido principal | Líneas aprox. |
|----|--------|---------------------|---------------|
| 1 | `feat(db): add defect records migration` | `000013` up/down, `migration_files_test.go`, `migration_integration_test.go` (reutilización de helpers de `testpostgres`) | ~250 |
| 2 | `feat(defect): add defect domain` | `domain/defect.go` + `defect_test.go` | ~240 |
| 3 | `feat(defect): add register defect use case` | `application/register_defect.go` + `register_defect_test.go` | ~260 |
| 4 | `feat(defect): persist defect records in PostgreSQL` | `repository.go`, `MapConstraintError`, `constraint_error_test.go`, `repository_integration_test.go` | ~430 |
| 5 | `feat(defect): agregar handler HTTP de registro de defectos` | `transport/http/handler.go` + `handler_test.go` | ~340 |
| 6 | `feat(defect): exponer la ruta de registro de defectos` | `api.go`, `main.go`, `defect_route_test.go`, `http_integration_test.go`, `README.md`, `config.yaml` | ~230 |
| 7 | `docs(defect): registrar la verificación final de la fase 7` | `apply-progress.md` completo + verificación responsable | ~120 |
| 8 | `docs(defect): agregar informe de verificación de la US-16` | `verify-report.md` | ~110 |
| 9 | `docs(defect): archivar la US-16 y consolidar el spec de defectos` | archive + consolidación + `archive-report.md` | ~180 |
| | **Total** | | **~2.160** |

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Migración `000013_create_defects` (up/down/reapply) y prueba canónica de 13 pares | PR único (size:exception) | `go test ./tests/integration/migrations/...` y `go test ./tests/integration/defect/postgres/... -run TestDefectMigration` | Testcontainers `postgres:16-alpine`: `up`, `down`, `down` repetido y reaplicación contra PostgreSQL real; requiere Docker | Revertir los dos `.sql`, el `"000013"` de `migration_files_test.go` y `migration_integration_test.go`; sin la migración no hay otro código que dependa de ella todavía |
| 2 | Dominio `Defect` con catálogos y validación acumulada | PR único (size:exception) | `go test ./tests/unit/defect/domain/...` | N/A — dominio puro sin dependencias externas | Eliminar `internal/defect/domain/` y su prueba; nada más lo importa aún |
| 3 | Caso de uso `RegisterDefectUseCase` con UUIDs y puertos | PR único (size:exception) | `go test ./tests/unit/defect/application/...` | N/A — probado con fake de repositorio y generador contador | Eliminar `internal/defect/application/`; depende solo de la UT 2 |
| 4 | `PostgresDefectRepository`, `MapConstraintError` y pruebas de integración de persistencia | PR único (size:exception) | `go test ./tests/unit/defect/infrastructure/postgres/...` y `go test ./tests/integration/defect/postgres/... -run TestDefectRepository` | Testcontainers `postgres:16-alpine` con migraciones `000001`–`000013` aplicadas desde archivos; requiere Docker | Eliminar `internal/defect/infrastructure/` y sus pruebas; la tabla `defects` (UT 1) queda sin código que la use |
| 5 | `RegisterDefectHandler` con decodificación estricta y mapeo de errores | PR único (size:exception) | `go test ./tests/unit/defect/transport/http/...` | N/A — `httptest` sobre `http.ServeMux` local con caso de uso real y fake de repositorio | Eliminar `internal/defect/transport/`; depende solo de la UT 3 |
| 6 | Cableado con gate `Defect` (`version >= 13`), integración HTTP real, README y `config.yaml` | PR único (size:exception) | `go test ./tests/unit/cmd/api/...` y `go test ./tests/integration/defect/postgres/... -run TestDefectHTTP` | `api.NewHTTPHandlerWithDependencies` con `PostgresDefectRepository` real sobre Testcontainers; requiere Docker | Revertir los bloques agregados en `api.go`, `main.go`, `README.md`, `config.yaml` y eliminar `defect_route_test.go`/`http_integration_test.go`; sin ellos la ruta nunca se registra |

### Orden y dependencias

Secuencial estricto: UT 1 → UT 2 → UT 3 → UT 4 → UT 5 → UT 6, y después las UT de documentación 7–9. Dependencias: UT 3 usa UT 2; UT 4 usa UT 1 (esquema del pool de integración), UT 2 y UT 3; UT 5 usa UT 3; UT 6 usa UT 1, UT 4 y UT 5. Paralelizables en la práctica (por no depender entre sí): UT 2 con UT 1, y UT 5 con UT 4, pero con un único escritor se ejecutan en el orden numérico para mantener cada commit autónomo y verde.

### Discrepancias entre spec y diseño (a resolver o confirmar)

La spec define el comportamiento y el diseño la estructura. Resueltas antes de aplicar:

- `created_at`: la tabla lo conserva y la entidad y la respuesta `201` NO lo exponen (spec Requirement "Asignar el identificador..."). La spec se alineó con esto; no se agrega `CreatedAt` a `Defect`.
- Catálogo completo (`resuelto`) en el `CHECK` de la base versus subconjunto aplicado por el alta: resuelto con el mensaje de dominio específico (`must be detectado or en_correccion when registering a defect`) y el escenario correspondiente de la spec.

Discrepancias que permanecen (ninguna cambia el comportamiento de la spec; se listan para que no se resuelvan en silencio):

1. **Violación de FK con comentario de proyecto**: la spec (Requirement "Garantizar la integridad en la base de datos", escenario "Una violación de clave foránea que llegue desde la aplicación no produce `500`") menciona la traducción de FKs a errores de dominio. El diseño no define una FK directa a `projects` (la composición de proyectos llega por sus FKs compuestas) y `MapConstraintError` mapea solo las dos restricciones reales (`defects_story_project_fkey` y `defects_sprint_project_fkey`). `ErrProjectNotFound` queda cubierto por la verificación transaccional (paso 1) y por la prueba del handler del caso de uso; no se inventa una FK ni un mapeo de proyecto. Es el mismo tratamiento honesto que la US-15 registró como discrepancia permanente.
2. **Escenario "Un fallo inesperado del almacenamiento no persiste nada"** (`500 internal_error` y ningún registro tras un fallo del almacenamiento): la estrategia de pruebas del diseño lo lista de forma explícita (UT 4 integración con un `TRIGGER BEFORE INSERT` que lanza excepción, verificando `500` y cero filas), y la UT 5 lo replica con un fake de repositorio que devuelve un error arbitrario.

## Checklist de tareas

### Fase 1 — Migración
- [x] 1.1 Escribir `000013_create_defects.up.sql` y `000013_create_defects.down.sql` según el diseño (RED primero con `migration_integration_test.go`).
- [x] 1.2 Actualizar `migration_files_test.go` a 13 pares canónicos.
- [x] 1.3 Verificar aplicar, revertir, revertir repetido y reaplicar contra Postgres real (Testcontainers).
- [x] 1.4 Confirmar que el `down` no toca tablas existentes y que `up` sobre versión 12 conserva sus filas.

### Fase 2 — Dominio
- [x] 2.1 Escribir `defect_test.go` (RED) y luego `defect.go` (GREEN): descripción, severidad, estado, acumulación.
- [x] 2.2 Casos borde: 500/501 caracteres multibyte, NUL, `resuelto` en alta, catálogos exactos sin trim.
- [x] 2.3 TRIANGULATE/REFACTOR sin cambiar comportamiento (mensajes estables).

### Fase 3 — Aplicación
- [x] 3.1 `register_defect_test.go` (RED) con fake de repositorio, generador contador y semillas de errores acumulados.
- [x] 3.2 `register_defect.go` (GREEN): normalización de UUIDs, llamada al dominio, sentinels, generador solo al éxito.
- [x] 3.3 Casos: `story_id`/`sprint_id` vacíos, inválidos, canónicos; error de repositorio transparente.

### Fase 4 — Persistencia
- [x] 4.1 `constraint_error_test.go` (RED): `MapConstraintError` con 23503 por cada FK y otros códigos.
- [x] 4.2 `repository.go` (GREEN): verificación transaccional en precedencia + INSERT + commit.
- [x] 4.3 `repository_integration_test.go`: persistencia, precedencia proyecto→historia→sprint, backlog, sprint cerrado, duplicados, FK violada entre verificación e INSERT (eliminación concurrente), error de almacenamiento → `500` sin filas.

### Fase 5 — HTTP
- [x] 5.1 `handler_test.go` (RED): 201 con forma exacta, 405, 400×N, 422 acumulados, 404×3 con precedencia, 500 genérico.
- [x] 5.2 `handler.go` (GREEN): decodificación estricta, parsing de `project_id`, mapeo de errores.
- [x] 5.3 TRIANGULATE: claves desconocidas, tipos incompletos, `id` inyectada.

### Fase 6 — Cableado y documentación
- [x] 6.1 `defect_route_test.go` (RED): readiness `>= 13` y rutas presenciales/ausentes.
- [x] 6.2 `api.go` + `main.go` (GREEN): `DefectDependencies`, `readiness.Defect`, bloque de composición, log y comentario de versiones.
- [x] 6.3 `http_integration_test.go`: e2e contra Postgres real (happy path, 404 real, duplicados reales, `GET → 405`).
- [x] 6.4 `README.md`: sección "API de defectos" con la tabla, cuerpo, códigos y nota de gating.
- [x] 6.5 `openspec/config.yaml`: nota "Implementada: US-16 (registrar un defecto...)". 

### Fase 7 — Verificación y cierre
- [x] 7.1 Corrida completa: `go build ./...`, `go vet ./...`, `gofmt -l` sobre los `.go` del cambio, `go test ./internal/... ./tests/unit/...` y `go test -p 1 ./tests/integration/...`; registrar evidencia en `apply-progress.md` (UT 7).
- [ ] 7.2 Verificación de trazabilidad spec↔pruebas y contraste con el issue #44 (`gh issue view 44`), produciendo `verify-report.md` (UT 8).
- [ ] 7.3 Archivar el cambio en `openspec/changes/archive/2026-10-25-us-16-registrar-defecto/`, consolidar el spec en `openspec/specs/defecto/spec.md` y escribir `archive-report.md` (UT 9).
- [ ] 7.4 Abrir el PR único hacia `main` con `size:exception` y asuntos por UT en la descripción.
