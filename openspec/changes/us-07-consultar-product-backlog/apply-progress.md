# Progreso de aplicación: US-07 — Consultar el Product Backlog

Cambio: `us-07-consultar-product-backlog` · Rama: `feat/us07-consultar-product-backlog` · Modo: TDD estricto
(`go test ./...`) · Estrategia de entrega: `single-pr` con excepción `size:exception`.

## Lote 1: Unidades 0, 1 y 2

### Unidad 0: compuerta y línea base

#### 0.1 Excepción `size:exception` (compuerta bloqueante)

El usuario **aceptó explícitamente** la excepción `size:exception` para el PR único (~1080 líneas frente al
presupuesto de 400) el 2026-09-26, con `chain_strategy=size-exception` y `delivery_strategy=single-pr`. La
confirmación proviene de la instrucción de lanzamiento del lote 1 y satisface la tarea 0.1. El PR final debe
declarar `size:exception` en su descripción.

#### 0.2 Rama y línea base

- `git branch --show-current` → `feat/us07-consultar-product-backlog` (correcta).
- Docker: `docker info` responde con el cliente y el servidor (Docker Desktop 28.0.4, contexto `desktop-linux`);
  la imagen `postgres:16-alpine` ya estaba disponible y Testcontainers pudo crear contenedores.
- `go test ./...` **antes de tocar código** (con Docker disponible, `-count=1`):
  - Unitarios: `ok` en `tests/unit/cmd/api`, `tests/unit/project/{application,domain,transport/http}` y
    `tests/unit/story/{application,domain,transport/http}`.
  - Integración: **los tests SÍ corrieron (no se saltaron) pero hay fallas preexistentes de entorno**:
    - `tests/integration/project/postgres`: `TestPostgresProjectRepositoryCreatePersistsProject`,
      `TestPostgresProjectRepositoryUpdateChangesBasicFieldsOnly` y
      `TestProjectsMigrationRejectsFinishDateBeforeStartDate` fallan con
      `wait for PostgreSQL test container: failed to connect ... [::1]:<puerto>: unexpected EOF`
      (conexión al contenedor por `localhost`/IPv6 con Docker Desktop recién iniciado).
    - `tests/integration/story/postgres`: `TestAPIStartupRoutesFollowMigrationState` falla en los subtests
      `version_one`, `version_two`, `version_three`, `dirty` y `dirty_version_three` (pasa `lookup_error`);
      es la falla preexistente de Windows ya conocida desde US-06 (binario `api` sin `.exe`) y/o la misma
      conexión inestable.
  - Estas fallas ocurren en el árbol limpio, antes de cualquier cambio de US-07: son **límites de entorno
    preexistentes**, no resultado de este trabajo. Se reportan tal cual y no se corrigen (fuera de alcance).
- Notas de entorno relevantes:
  - `gofmt -l .` lista casi todos los archivos `.go` porque el árbol de trabajo está en CRLF
    (`core.autocrlf=true`; el repositorio guarda LF). Es ruido preexistente. La verificación de formato de
    este lote se hace normalizando el fin de línea (`tr -d '\r' | gofmt -l`) sobre los archivos tocados.
  - `go vet ./...` limpio.

#### 0.3 Commit de planificación

Commit `8159ccf` `docs(sdd): add US-07 planning artifacts for backlog query` (solo archivos bajo `openspec/changes/us-07-consultar-product-backlog/`).

### Unidad 1: Dominio (precedencia de prioridad y `Backlog`)

Ciclo TDD observado (todo ejecutado de verdad):

- **Red de seguridad** (archivos existentes tocados: `story.go`): `go test ./tests/unit/story/domain/...` antes
  de tocar nada → `ok`, 12 tests de nivel superior pasando.
- **RED**: `tests/unit/story/domain/backlog_test.go` con `TestNewBacklogOrdersByPriority`,
  `TestNewBacklogIsStableForEqualPriorities`, `TestNewBacklogDoesNotMutateInput`,
  `TestNewBacklogPreservesProjectID`, `TestNewBacklogHandlesNilAndEmpty` y
  `TestAllowedPrioritiesReturnsOrderedCopy`. Falla observada (error de compilación): `undefined:
  domain.NewBacklog` y `undefined: domain.AllowedPriorities` → `FAIL ... [build failed]`.
- **GREEN**: constantes `PriorityHigh/Medium/Low` y `AllowedPriorities()` en `story.go`; `backlog.go` con
  `Backlog`, `NewBacklog` (`slices.Clone` + `slices.SortStableFunc`) y `priorityRank`. Resultado: `ok`, los seis
  tests pasan (incluido el escenario S1…S5 → S2, S5, S1, S3, S4).
- **TRIANGULATE**: `TestNewBacklogTriangulatesSpecScenarios` con misma prioridad (tres `media`), reubicación por
  prioridad (S1 `media`, S2 `alta`, S3 `alta` → S2, S3, S1), prioridad desconocida al final conservando el
  orden de entrada (`urgente`, `critica`) y una sola historia. Resultado: `ok` sin cambios de producción
  (la implementación ya generalizaba).
- **REFACTOR**: `validateStoryContent` valida ahora con `slices.Contains(AllowedPriorities(), priority)` y arma
  el mismo mensaje `must be alta, media or baja` con las constantes. Suite de dominio verde antes (12/12) y
  después; el resto de `go test ./tests/unit/...` verde.
- Verificación: `go vet ./...` limpio; formato normalizado sin diferencias; `go test ./tests/unit/...` verde.
  La integración quedó como en la línea base (fallas de entorno preexistentes descritas arriba); la unidad no
  toca almacenamiento ni composición.

Work Unit Evidence (Unidad 1):

| Evidencia | Valor |
|---|---|
| Comando focalizado y resultado | `go test ./tests/unit/story/domain/... -count=1` → `ok` (12 tests previos + 7 nuevos de nivel superior) |
| Arnés de ejecución | N/A: funciones puras sin frontera de proceso; la evidencia es la suite unitaria |
| Frontera de rollback | `internal/story/domain/story.go`, `internal/story/domain/backlog.go`, `tests/unit/story/domain/backlog_test.go` |
