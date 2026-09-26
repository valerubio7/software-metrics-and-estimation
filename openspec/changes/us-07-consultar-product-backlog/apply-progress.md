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
