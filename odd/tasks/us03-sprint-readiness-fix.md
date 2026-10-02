# Corrección de readiness de sprints en US-03

## Objetivo y autorización

Corregir el bloqueante P2 del PR [#69](https://github.com/valerubio7/software-metrics-and-estimation/pull/69): la cadena canónica crea `sprints` en v5, pero la API habilitaba su ruta desde v3. El usuario autorizó corregir, guardar y publicar en la misma rama del PR, y aprobar únicamente si la revisión y las pruebas son satisfactorias. No autoriza merge, despliegue ni instalación de herramientas en el sistema.

- Base revisada: `d542c561dcf1cbe89cdf3926522599c7f885ccf7`.
- Rama local aislada: `fix/us03-sprint-readiness`; entrega fast-forward a `origin/feat/us-03-register-project-members`, sin force-push. Rama documental del usuario preservada.
- ODD directo delegado, sin nuevo flujo SDD ni cambios en artefactos archivados. Escritor por tres archivos no triviales; verificadores independientes para código y migraciones; padre coordina Git/GitHub.
- TDD activo según `openspec/config.yaml`, `sdd.strict_tdd: true`; runner `go test ./...`. RDD desactivado, sin revisión/autoridad nativa.
- Alcance: gate/comentarios en `internal/api/api.go`, arranque/log en `cmd/api/main.go`, regresiones en `tests/unit/cmd/api/main_test.go` y este registro. No cambiar SQL, integrantes ni historias.
- Mantener el PR existente y su excepción de tamaño documentada. Corrección de código: 51 adiciones/7 eliminaciones; registro de cierre separado. No crear otro PR.

## Tarea técnica completada

- [x] FIX-01 — Corregir, verificar y publicar la habilitación segura de sprints.
  - [x] RED en v3/v4; corregir a clean v5+, preservar stories/members/proyectos y fail-closed dirty/error.
  - [x] GREEN y refactor; regresiones de HTTP y write counts independientes del resolver.
  - [x] Verificación independiente del código, suite y matriz descartable de migraciones.
  - [x] Commit de unidad de trabajo y publicación fast-forward confirmada por readback remoto.
  - Commit: `c3f765a3fdd9055f76a6e2030c2f5284d268c8f3`, `fix(api): gate sprint creation on reconciled schema`.
  - Aceptación observada: v3/v4 no habilitan sprints; v5/v6+ clean sí; dirty/error no; members requiere v6; stories/proyectos preservados. El arranque usa el resolver común.

La aprobación condicional es una acción externa posterior al registro técnico: solo sobre el HEAD final publicado y comprobado. Su resultado y commit exacto se registran en la revisión de GitHub y la memoria de sesión, sin modificar después el candidato aprobado.

## Evidencia

- Escritor `muqaxhf5-5-bkk0`: RED focalizado observado en v3/v4 (gate true, HTTP 201/un write frente a 404/cero); GREEN y rerun tras gofmt. Suite completa con Docker y sin DSNs externos, `go vet ./...` y `git diff --check` aprobados. Modelo `gpt-6.1-sol`, esfuerzo desconocido.
- Padre leyó el diff completo y repitió `go test ./tests/unit/cmd/api/... -run TestMigrationReadinessSelectsRoutesIndependently -count=1`: PASS. El commit publicado conserva exactamente los tres archivos y 51/7 líneas verificados.
- ASSESS en cwd documental sin candidato devolvió `unassessable`, no evaluó el worktree secundario. Aplicada ruta conservadora de riesgo alto con verificador independiente, sin habilitar RDD.
- Verificador código `muqb6le9-7-egqj`: PASS sin bloqueantes. Unitarias focalizadas, `go test ./... -count=1 -json`, vet, whitespace y scope aprobados; 454 tests/subtests en 18 paquetes, sin fallos/skips de tests; otros 19 paquetes no contienen tests. PostgreSQL real con Testcontainers. Arranque inspeccionado por source, no servidor lanzado.
- Verificador matriz `muqay4hm-6-utbu`: PASS en PostgreSQL `18.6 (Debian 18.6-1.pgdg13+2)` / `180006` y migrate `4.19.1` / tag `v4.19.1`. Ejecución equivalente Docker, no los scripts locales originales (sus herramientas de servidor están ausentes).
- Fresh y ambas historias v4 convergen a `6 false` conservando datos; v5 idempotente; weak-hours/status, weak-status-only, partial-sprints y legacy-status fallan en `5 true` sin alterar snapshots de esquema/datos. Down1 desde v6 conserva relaciones/filas históricas y elimina solo members. Canonical v3/v4 no contienen sprints; v5 sí.
- SQL idéntico a la base antes/después. Padre leyó el log de matriz: salida 0 y cleanup rc0, sin contenedores/network remanentes. Evidencia local: `/tmp/pr69-docker-matrix-zwvb77k3/` y `/tmp/us03-readiness-verify-go-test.e2Ss67.json`.
- Readback GitHub confirmó `c3f765a` en la rama del PR. No sobrescritura concurrente ni force-push. GitHub no muestra checks automatizados; no afirmar CI aprobada.

## Límites y cierre

Compatibilidad, runner/historial y rollback de producción siguen sin demostrarse. El rollback histórico v3 no es un defecto introducido: contrato forward-only y SQL histórico intacto. La revisión de GitHub determina el estado de aprobación del HEAD final; no autoriza merge ni despliegue. No quedan tareas técnicas de esta corrección.
