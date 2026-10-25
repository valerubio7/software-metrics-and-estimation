# Reporte de Archivo — US-16 Registrar un defecto detectado durante el desarrollo

**Fecha de archivo**: 2026-10-25  
**Estado**: Archivado. Implementación completa; las fases 1–6 y las tareas 7.1–7.3 quedan marcadas en el `tasks.md` archivado; la tarea 7.4 (abrir el PR) se resuelve inmediatamente después del archivo y se registra en este reporte.  
**Rama**: `feat/us-16-registrar-defecto`  
**PR de entrega**: único, con etiqueta `size:exception` (decisión 5 del usuario).

## Resumen ejecutivo

El cambio US-16 "Registrar un defecto detectado durante el desarrollo" queda archivado. Consolida la capacidad nueva `defecto` en `openspec/specs/defecto/spec.md` (15 requisitos, 40 escenarios) e implementa `POST /projects/{project_id}/defects` con la migración `000013_create_defects`, el módulo `internal/defect` completo (dominio, caso de uso, repositorio PostgreSQL, handler HTTP) y el cableado con gate `version >= 13` en `internal/api/api.go` y `cmd/api/main.go`.

La verificación final dio **PASS** (0 CRITICAL, 0 WARNING, 3 SUGGESTION). La suite completa (26 paquetes unitarios + 8 paquetes de integración con Testcontainers) quedó en verde en una única corrida previa al archivo; el archivado no volvió a ejecutar pruebas.

## Estado de implementación

- **Tareas en el archivo**: fases 1–6 completas `[x]` y fase 7 con 7.1–7.3 completas; solo 7.4 queda abierta en ese archivo.
- **Cierre de la tarea 7.4 (registrado aquí)**: PR único hacia `main` con etiqueta `size:exception`, de la rama `feat/us-16-registrar-defecto`.

### Commits de la rama (orden cronológico)

| Commit | Mensaje | Alcance |
|---|---|---|
| b7e7b26 | docs(defecto): add SDD planning artifacts for US-16 | exploration, proposal, spec delta y tasks |
| a9d6717 | feat(db): add defect records migration | UT 1: migración `000013` + 13 pares |
| 3fb51b3 | feat(defect): add defect domain | UT 2: dominio `Defect` |
| 5c2f012 | feat(defect): add register defect use case | UT 3: caso de uso |
| 5cc5be3 | feat(defect): persist defect records in PostgreSQL | UT 4: repositorio y traducción de FKs |
| c8161f7 | feat(defect): add defect HTTP handler | UT 5: handler HTTP |
| 4ac8107 | feat(defect): expose defect registration route | UT 6: cableado, gate, e2e, README y `config.yaml` |
| 155524d | docs(defecto): cerrar el progreso de apply de la US-16 | UT 7: evidencia y cierre |
| 2c66a6f | docs(defecto): agregar informe de verificación de la US-16 | UT 8: verify-report |

## Verificación

Fuentes: `verify-report.md` (archivado, sobre el commit `155524d`) y los hechos finales.

- **Veredicto**: PASS. Contraste criterio por criterio contra el issue #44: los 9 criterios de aceptación del alta quedan satisfechos; el Sprint de resolución queda pendiente por diseño (US-17), tal como el propio issue indica en su "Observación".
- **Trazabilidad**: cada Requirement del delta tiene pruebas de cobertura (matriz completa en la sección 3 del `verify-report.md`).
- **Sugerencias abiertas (no bloqueantes)**: (1) no añadir un `CHECK` de NUL de más en la base — PostgreSQL rechaza el byte NUL en `text` por su cuenta; (2) revisar la utilidad de `defects_project_seq_idx` cuando exista el listado de US-18; (3) si un día se quiere `detection_sprint_id` como nombre de columna, hacerlo con migración dedicada.

### Evidencia de pruebas (registro, no re-ejecutado en el archivo)

- `go test ./internal/... ./tests/unit/... -count=1`: 26 paquetes verdes.
- `go test -p 1 ./tests/integration/... -count=1`: 8 paquetes verdes en una sola corrida (defect 48,8 s; effort 86,1 s; migrations 0,0 s; project 10,8 s; projectmember 14,1 s; sprint 96,2 s; story 236,3 s; task 44,4 s). Sin regresiones en los módulos existentes.
- `go build ./...`, `go vet ./...`, `gofmt -l`: limpios.

## Tamaño y entrega

- **Pronóstico** (`tasks.md`): ~1.750–2.100 líneas; riesgo del presupuesto de 400 líneas: `High`; estrategia `single-pr` con `size:exception`.
- **Medición real** (`git diff --numstat` contra `main`, base `df1ff7b`):
  - Código y pruebas (excluido `openspec/`): 2.514 adiciones + 4 eliminaciones ≈ **2.518 líneas**, ~6,3 veces el presupuesto de 400 (mayormente pruebas).
  - Total de la rama, incluidos los artefactos SDD bajo `openspec/`: 3.719 + 5 ≈ **3.724 líneas**.

## Especificación consolidada

- **Capacidad nueva**: `openspec/specs/defecto/spec.md` (sin main spec previa, copia directa del delta).
- **Método**: copia del delta con el encabezado del delta normalizado a `## Requirements` (precedente de US-10); el delta archivado conserva `## ADDED Requirements` byte a byte.
- **Contenido**: 15 requisitos (`### Requirement:`) y 40 escenarios (`#### Scenario:`), contados en el archivo consolidado.

## Artefactos preservados

Carpeta `openspec/changes/archive/2026-10-25-us-16-registrar-defecto/`: `proposal.md`, `design.md`, `exploration.md`, `tasks.md`, `apply-progress.md`, `verify-report.md`, `specs/defecto/spec.md`, `archive-report.md` (este reporte, aditivo).

## Cierre del change local

`openspec/changes/us-16-registrar-defecto/` se elimina del directorio activo en el mismo commit de archivo (queda solo el snapshot bajo `archive/`), para que `main` no muestre cambios pendientes de otra fase.

## Riesgos residuales

- US-17 comparte la tabla `defects`: su migración `000014` agregará `resolution_sprint_id`; el `CHECK` de estado ya acepta `resuelto`, el alta nunca lo produce y la transición es exclusión de este alcance.
- El gate `>= 13` detiene la ruta en esquemas atrasados; el despliegue de la migración sigue siendo externo (misma política que el resto del repo).
- Duplicados permitidos: nada impide registrar descripciones idénticas repetidas; si US-19 necesita deduplicación estadística, es una decisión de consulta, no de persistencia.
