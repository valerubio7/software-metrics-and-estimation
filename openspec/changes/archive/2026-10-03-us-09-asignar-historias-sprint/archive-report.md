# Archive Report: us-09-asignar-historias-sprint

## Naturaleza de este cierre

Este cambio se entregó originalmente fuera del pipeline SDD estándar: un esfuerzo de consolidación ad-hoc (PR #78, rama `fix/us09-consolidation`) que portó el comportamiento ya construido en la cadena abandonada `feat/us-09-assignment-01..06` (tracker PR #71, PRs #72–77, nunca fusionados) directamente sobre el esquema canónico real de `main`. Este archivo completa retroactivamente el expediente SDD de ese cambio, conforme a la política actualizada del equipo de documentar todo cambio mediante SDD a partir de ahora — ya no mediante ODD.

## Estado de cierre

- **Implementado y fusionado en `main`.** Commit de comportamiento `f544916 feat(story): integrar US-09 sobre migraciones canónicas` (23 archivos, 1158 inserciones, 45 eliminaciones, confirmado con `git show --stat`); commit de cierre ODD original `3ad5433 docs(odd): cerrar verificación y entrega de US-09`; merge commit `63a6527 Merge pull request #78 from valerubio7/fix/us09-consolidation`.
- **Issue #37 cerrada** en GitHub; no se modifica su estado desde este archivo, solo se referencia.
- Todas las tareas de `tasks.md` (1.1–4.3) están marcadas completas con evidencia reconstruida del código y las pruebas existentes.

## Especificaciones

- Las especificaciones canónicas **ya contienen** el texto de requisito fusionado directamente por el commit `f544916`, sin pasar por la fase `sdd-spec` de este cambio:
  - `openspec/specs/historia/spec.md`, líneas 843–889: `### Requirement: Asignar un lote de historias sin retirarlas del Product Backlog (US-09)` y `### Requirement: Habilitar asignación solo con esquema completo y dependencia explícita`.
  - `openspec/specs/sprint/spec.md`, líneas 60–77: `### Requisito: Rechazar asignaciones a un Sprint cerrado (US-09)`.
- Este archivo conserva en `specs/historia/spec.md` y `specs/sprint/spec.md` el delta `## ADDED Requirements` correspondiente, copiado verbatim del contenido canónico vigente, como registro de auditoría de este cambio. **No se modifica `openspec/specs/` desde esta fase**: el contenido ya está presente allí; este delta documenta qué se habría agregado si el flujo SDD se hubiera seguido en su momento.

## Verificación

- Verificación re-ejecutada hoy (2026-10-03): `go build ./...` exit 0; `go vet ./...` exit 0 sin advertencias; `go test ./tests/unit/...` con todos los paquetes `ok`.
- Pruebas de integración (Testcontainers/Docker): **omitidas por entorno** — Docker no disponible en esta sesión de verificación. Esto es una limitación del entorno, explícitamente no un defecto de código. La evidencia histórica de su ejecución exitosa consta en el commit `f544916` y en el registro de entrega original (529 PASS, 0 FAIL/SKIP al momento de la fusión, según `odd/tasks/us09-consolidation.md`, ya eliminado de este repositorio).
- No se encontraron inconsistencias, errores ni validaciones faltantes contra ningún contrato declarado (coincidencia de proyecto, no duplicados, rechazo de Sprint cerrado, atomicidad, visibilidad en backlog, canonicalización de UUID).
- Fragilidad de diseño documentada, no un defecto: `dependencies.Stories` (`cmd/api/main.go:53`, no nulo ya desde gate `>=2`) y `readiness.Assignment` (`internal/api/api.go:95`, gate `>=8`) son dos comprobaciones de versión separadas en dos archivos que deben revisarse juntas en futuras migraciones.

## Antecedente de entrega

La cadena de PRs hijos originalmente planificada (`feat/us-09-assignment-01-core` a `06-progress`, tracker PR #71, PRs #72–77) nunca llegó a `main`; quedó cerrada/obsoleta. La integración real se rehízo en un único esfuerzo de consolidación directamente contra el esquema canónico verdadero de `main`, portando el comportamiento validado en `feat/us-09-assignment-05-sdd` (commit `e4ddb66`). El detalle completo de este antecedente está en `design.md`, que sustituye el contenido del ahora eliminado `chain-context.md`.

## Resultado final

- Implementado: sí. Fusionado a `main`: sí. Issue #37: cerrada.
- Especificaciones: reconciliadas (ya presentes en los archivos canónicos; este cambio formaliza su delta retroactivamente).
- Sin bloqueos pendientes. Próximo paso: ninguno — este archivo completa el expediente SDD retroactivo del cambio.
