# Consolidación de US-09 hacia main

## Objetivo y autorización

Integrar la implementación acumulada de asignación de historias a Sprint en `main` mediante un PR nuevo, conservando US-03/US-04. Los PR #72–77 se cerraron contra ramas intermedias después del tracker #71 y su contenido no llegó a main. El usuario autorizó integración, pruebas, commits/publicación y PR; eligió explícitamente `exception-ok`, un PR único con `size:exception`. No autoriza merge, despliegue, force-push, deshacer merges públicos ni instalación de herramientas en el sistema.

- Base: main `21ef90fdc5f3710ec6df8c399d51a4a76f0ddcd8`.
- Fuente de comportamiento: `e4ddb669ebe6beec1d80e492c7551f423144f31d`, branch `feat/us-09-assignment-05-sdd`; árbol `c55cabaa0bdfa32af72705d902aef6190c5d838b` igual al branch de progreso.
- Rama aislada de entrega: `fix/us09-consolidation`; fuente original solo lectura. Rama documental del usuario preservada.
- Issue #37 comprobada: `status:approved`, actualmente CLOSED; reutilizar su trazabilidad sin modificar el estado del issue.
- ODD directo delegado, no nueva ejecución SDD ni reescritura de artefactos archivados. Excluir registros de reparación/entrega históricos y no copiar el árbol completo de la cadena.
- TDD activo en `openspec/config.yaml`, `sdd.strict_tdd: true`; runner `go test ./...`. RED real antes de producción: prueba SQL sobre el helper existente de main que detecte ausencia de la asociación/columna de cierre, sin errores de sintaxis o símbolos faltantes; después GREEN/refactor y suite completa.
- RDD desactivado; evaluar diff del escritor, aplicar verificación independiente conservadora y conservar evidencia. Sin revisión/autoridad nativa.
- Forecast inicial del mapper: 600–800 líneas; medir el diff real y mantener todas las pruebas, sin code-golf. Excepción de tamaño autorizada para la unidad coherente; código/copias de tests también cuentan.

## Reglas de integración

- Migraciones canónicas existentes 000001–000006 y archivos archivados de main permanecen byte a byte intactos.
- Nuevas migraciones: 000007 asociación, 000008 `sprints.is_closed`. Assignment solo con estado limpio y versión >=8; sprints >=5 y miembros >=6 conservados.
- Soporte comprobable: instalación nueva y upgrade del esquema canónico main v6. No prometer compatibilidad de producción ni de bases que hayan aplicado los v5/v6 alternativos de la cadena; no conectarse a ninguna base persistente.
- Mantener contratos originales de selección, pertenencia al proyecto, duplicados, Sprint cerrado y atomicidad. Las historias asignadas siguen perteneciendo al Product Backlog: probar `ListByProject` después de asignar, no solo contar filas.
- Preservar endpoints y tests US-03/US-04. Documentar contratos canónicos de US-09 sin ampliar funcionalidad (no tareas, cierre HTTP ni estados completados).
- Actualizar los helpers al orden canónico completo y mantener útil la matriz histórica: caracterización/reapply v5 en el corte v6, luego upgrade a v8. No exigir reapply histórico v5 sobre el nuevo esquema v8 ni interpretar down1 de v8 como eliminación de miembros. Los down de US-09 eliminan datos/cierre, no son rollback productivo seguro.

## Tarea técnica

- [x] CON-01 — Integrar, verificar y publicar US-09 en un PR único hacia main.
  - [x] RED real de esquema ausente en main y port/adaptación del comportamiento y pruebas de la cadena.
  - [x] Migraciones nuevas/gates/helpers coherentes; README/specs y matriz descartable actualizados y verificados.
  - [x] Verificación independiente de fuente, suite y upgrade canónico v6→v8 con datos preservados.
  - [x] Commit de unidad de trabajo, publicación y nuevo PR hacia main ligado a #37 con excepción de tamaño.
  - Ruta: `gentle-ai-worker` único por múltiples archivos; verificadores independientes para código/pruebas y matriz SQL; padre mantiene este registro y coordina Git/GitHub.
  - Aceptación: asociación persistida del mismo proyecto, sin duplicados ni incorporación a cerrado, sin escrituras parciales ante rechazo; backlog conserva historias. US-03/US-04 y gates previos intactos. Versiones únicas, ruta ausente v6/v7/dirty/error, presente v8+ con dependencia explícita.
  - Checks: RED/GREEN focalizado; unitarias de assignment/API; integración story/sprint; `go test ./... -count=1`; `go vet ./...`; gofmt/diff; bash syntax y ShellCheck de scripts modificados (Docker sin instalación host si necesario); matriz PostgreSQL 18.6/migrate v4.19.1 descartable, v6 seed/upgrade v8/down/reapply y fixtures históricos válidos/negativos.
  - Commit fuente verificada: `f544916721033090349adb2ab3d266d841ebd3c1`, árbol `a26f4b2607010849ae72e7d02ccde562f93e4d6a`.
  - PR: [#78](https://github.com/valerubio7/software-metrics-and-estimation/pull/78), `fix/us09-consolidation` → `main`; labels `type:feature`, `size:exception`. No merge ni aprobación automática.

## Superficies previstas

- Nuevos: `internal/story/application/assign_stories.go`, `internal/story/transport/http/assign_stories_handler.go`, tests unitarios e integración de assignment, cuatro archivos SQL v7/v8.
- Modificar de forma quirúrgica: `cmd/api/main.go`, `internal/api/api.go`, repo PostgreSQL de story, tests API/helpers/integración story/sprint y checks de migración; `internal/sprint/domain/sprint.go` solo si el campo de la fuente es necesario.
- Documentación: README y specs canónicas historia/sprint. Harness/README de migraciones para preservar el corte v6 y probar la nueva cabeza. Ningún archivo histórico SQL, configuración metodológica ni archivo archivado.
- Padre es único escritor de este registro; trabajador sin staging/commits/push/acciones GitHub.

## Evidencia y estado

- Exploración `muqfygqd-f-5co1` read-only: fuente/main caracterizados; gate v6 de la cadena es incorrecto en main, cuyo v6 crea miembros. Identificada falta de regresión directa del backlog.
- Escritor `muqgooef-g-sat5` observó RED real antes de producción: test compila y falla por `sprint_stories=false, sprints.is_closed=false` en PostgreSQL descartable. Continuación `muqh6di8-h-l5q4` completó port hasta GREEN; el mismo test pasa con helpers canónicos 1–8.
- Revisión independiente identificó un defecto importado de UUID alias: mismo UUID lower/uppercase esquivaba la validación raw. Repro `/tmp/con01-uuid-aliases-repro-result.log` dio HTTP201/1 llamada; no prueba duplicados persistidos. Fix `muqizr68-l-bvoy` solo handler/tests: canonicalizar UUID sin deduplicar ni cambiar el contrato opaco de aplicación; RED real en repo y GREEN422/0 llamadas, orden/éxito preservados.
- Escritor final informa 22 archivos, 1.104 adiciones/45 eliminaciones excluyendo registro. Candidato verificado incluyendo registro: 23 archivos, 1.158 adiciones/45 eliminaciones. Excepción autorizada; aumento por pruebas/integración necesarias, sin recortes.
- Verificación independiente final `muqxllnh-m-lalk`: árbol `a26f4b2`, fuente/index iguales antes/después, **183 tests + 346 subtests = 529 PASS**, 0 FAIL/test SKIP; 18 paquetes con tests y 19 sin tests. Vet/gofmt/diff pasan. Baseline bash/ShellCheck Docker v0.11.0 aprobado y scripts idénticos. DSNs persistentes desactivados. JSON `/tmp/con01-final-go-suite-a26f4b26.json.log`, stderr vacío. Assessment nativo no expuesto en el verificador: riesgo unassessable tratado como alto; sin aprobación nativa ni presunción de perfil grande.
- Matriz independiente `muqi9jp5-j-zi7a`: **15 PASS/0 FAIL/0 SKIP**, PostgreSQL18.6 real, migrate4.19.1 y módulo binario confirmados. Fresh, canonicalv6 y ambas historiasv4 llegan a `8 false`, preservan todos los campos base/miembros, idempotencia, asociaciones válidas y rechazos FK/duplicados atómicos. Once fixtures negativas mantienen schema/data exactos y fallan `5 true`. Down2→6 conserva base/miembros; down1 solo desde6 elimina miembros; reapply recrea vacío/abierto. Datos de asociaciones/cierre no se preservan en down.
- Evidencia SQL `/tmp/us09-sql-verifier-JM8qNHRN/` (`output-full.log`, `summary.json`, snapshots y `migrate-module-version.txt`). Containers/red propios eliminados por ID. Todos los SQL1–8/scripts idénticos entre f2 y a26: matriz reutilizada por identidad. 71 archivos protegidos BASE (14 SQL/fixtures, 57 archives) byte-idénticos; fuente original e4 limpia. Runners host originales no ejecutados: equivalencia Docker, sin herramientas instaladas en host.
- Readback inicial PR#78: OPEN, MERGEABLE, mergeStateStatus BLOCKED, mergedAt null, checks reportados `[]`. No inferir CI aprobado ni habilitación de merge; este registro de cierre es la única modificación posterior a la fuente verificada.
- Producción e historial de la cadena desconocidos; limitación explícita de rollout, no pretexto para renumerar historial de main.

## Siguiente paso

Revisión humana del PR#78 y resolución de requisitos GitHub antes de merge. No merge autorizado en esta tarea. Antes de desplegar, comprobar el historial real del ambiente; US-09 todavía no está en main mientras este PR siga abierto.
