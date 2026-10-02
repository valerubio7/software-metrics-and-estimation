# Sincronización de US-04 con main

## Objetivo y autorización

Actualizar el PR [#70](https://github.com/valerubio7/software-metrics-and-estimation/pull/70) con main, resolver los conflictos preservando US-03 y US-04, verificar y publicar. El usuario autorizó resolución/publicación y aprobación condicionada; no merge del PR a main, despliegue, force-push ni instalación de herramientas.

- HEAD original: `dd027fbbe2f71522cdac7b58dfde2b77bf76cf2f`; main integrado: `6ca3a65dd4d331e5ac92749f522c969cb162de27`.
- Rama local aislada `fix/us04-main-sync`; entrega fast-forward a `origin/feat/us-04-project-status`. Rama documental del usuario preservada.
- ODD directo delegado, no nuevo SDD ni reescritura de archivos archivados. Git/staging/commits/publicación del padre; escritor solo resolvió los dos archivos de conflicto.
- Conflictos: `openspec/specs/project/spec.md` y `tests/unit/cmd/api/main_test.go`, resueltos como unión, no elección wholesale de una rama.
- TDD configurado activo (`openspec/config.yaml`, `sdd.strict_tdd: true`, runner `go test ./...`). Integración de funcionalidades existentes, sin nueva implementación de comportamiento; parse failure de marcadores es evidencia de integración, no un nuevo RED conductual.
- RDD desactivado; ASSESS en cwd documental sin candidato fue `unassessable`, por lo que se aplicó verificador independiente conservador. No revisión/autoridad nativa.
- PR existente con size:exception documentada; código entrante de main no es implementación nueva de esta tarea.

## Tarea técnica completada

- [x] SYNC-01 — Integrar main, resolver conflictos, verificar y publicar US-04.
  - [x] Conservar todas las adiciones de requisitos/tests de US-03 y US-04, imports/helpers y funciones completas.
  - [x] Mantener GET status, members, rutas previas y gate de sprints clean v5+; members requiere v6+.
  - [x] Verificar fuente integrada independientemente, no solo el HEAD viejo.
  - [x] Guardar commit con ambos padres y publicar sin force-push; readback remoto confirmado.
  - Commit de integración: `06d564a8d8e3a148ae7cd13378e853d1a617d23a`, `fix(api): reconcile US-04 with current main`.
  - Árbol fuente verificado y publicado: `e91969acbce9d4d2392fba939f761a2acdff54df`, sin este registro de cierre.

La aprobación condicional es posterior al registro técnico y únicamente sobre el HEAD final publicado y comprobado. Su resultado y commit exacto se conservan en GitHub y memoria de sesión, sin alterar después el candidato aprobado.

## Evidencia

- Revisión original US-04 sin defectos bloqueantes; sus tests pasaban. La compatibilidad con main estaba bloqueada por los dos conflictos.
- Primer despacho detenido por typo de rama en la instrucción, sin cambios; continuación falló sin resultado utilizable. Padre verificó estado antes de retomar y no repitió el merge abierto.
- Escritor nuevo `muqdgkt4-c-yja9`: resolución únicamente de spec/tests, conservando ambas ramas. Parse failure previo observado, luego unitarias focalizadas, full suite con Docker, vet y gofmt PASS. Sin cambios manuales de producción ni Git.
- Padre leyó diff spec/tests/API contra main, registró los dos archivos y comprobó índice sin U/unstaged changes, whitespace y árbol staged. Spot-check status/member/readiness PASS.
- Verificador `muqdppgo-d-ws00`: clean y sin bloqueantes. HEAD/MERGE_HEAD/árbol fuente idénticos antes/después. Contra main: 19 archivos +783/-0; requisitos de ambas US y todas las declaraciones de tests preservados, sin duplicados ni marcadores.
- Unitarias focalizadas, `go test ./... -count=1 -json`, `go vet ./...`, gofmt y diff checks PASS. 164 tests y 306 subtests pasan, sin fallos/skips; otros 19 paquetes sin tests. PostgreSQL descartable real y cleanup observados. JSON local: `/tmp/us04-main-sync-verifier-fullsuite.lKm7nV.json`.
- SQL, módulo members y archivos archivados US-03 idénticos a main. No nueva matriz SQL ni servidor HTTP listening ejecutados; validación de handler en proceso e integración PostgreSQL.
- Commit tiene como padres el HEAD original y main fijado; su árbol coincide exactamente con el probado. Publicación/readback confirmó `06d564a` en la rama del PR. El cierre documental no cambia fuente ni SQL.

## Límites y cierre

No quedan tareas técnicas de esta sincronización. La revisión de GitHub determina el estado de aprobación del HEAD final y no autoriza merge ni despliegue. Compatibilidad del runner/historial productivo permanece fuera de esta validación. GitHub no reporta checks de CI; no afirmar CI aprobada.
