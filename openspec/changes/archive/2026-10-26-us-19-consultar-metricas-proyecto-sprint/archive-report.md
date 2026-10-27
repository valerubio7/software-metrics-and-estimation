# Informe de cierre y archivo — US-19 (#47)

## Resultado

API de métricas implementada en `feat/us-19-metricas`, sin cambios TUI/US-20/US-21 ni migraciones. Conserva el JSON plano de PR #115 y agrega desviación porcentual y cobertura. La publicación/PR queda pendiente de `status:approved` en #47; no se fusionó nada.

## Evidencia observada

- RED previo a dominio, aplicación y composición: paquetes/campos aún inexistentes; GREEN posterior documentado en el seguimiento ODD.
- `go test ./tests/unit/...`: PASS.
- `go test ./tests/integration/...`: PASS, PostgreSQL real, sin omitir paquetes; duración máxima story 94.645s, metrics 2.041s.
- `go build ./...`, `go vet ./...`: PASS.
- `openspec validate us-19-consultar-metricas-proyecto-sprint --strict`: PASS antes del archivo.
- Formato y `git diff --check`: sin errores.
- Go observado: 1.27.1; no se probó 1.27.0 exacto.
- Diferencial contra base `5564d8932bfc61fc8244191e98ec224e7e020ffe`: 55 tests API existentes, salida JSON normalizada solo en tiempos, stderr y códigos de salida iguales (0).

## Verificación independiente

`gentle-ai-verify` no encontró defectos funcionales bloqueantes. S1-S7 cumplidos en implementación; S8 parcial en su copia por no disponer de baseline. El padre realizó el diferencial indicado arriba. El verificador ejecutó suites completas, build y vet, y una prueba temporal independiente de historia completada en dos Sprints, puntos faltantes y snapshot SHA256 de ocho tablas sin mutación tras consultas repetidas/rechazadas. Esa prueba temporal no se incorporó al repositorio; su directorio se eliminó con autorización expresa.

## Trazabilidad

| Specs | Evidencia |
|---|---|
| S1 | Rama desde main; artefactos españoles; API únicamente; archivo OpenSpec |
| S2-S3 | `tests/unit/metrics/domain`, `tests/integration/metrics/postgres`: ámbitos, deduplicación, completed_at, ceros/nulos |
| S4 | Pruebas dominio y PostgreSQL de velocidad cerrada/vacía; prueba independiente de faltantes |
| S5 | Pruebas dominio/PostgreSQL de tareas, reemplazo, atribución estricta y registros de valores iguales |
| S6 | Integración de detección A/resolución B y ausencia de resolución pese a status |
| S7 | `tests/unit/cmd/api/metrics_route_test.go`: claves exactas, cobertura, errores, GET/HEAD, 405 y gates |
| S8 | Suites completas y diferencial anterior; snapshot independiente sin cambios |

## Límites

No se lanzó un proceso servidor para cada versión de migración: la matriz se verifica por composición HTTP pública y se inspeccionó/compiló `cmd/api`. El diferencial usa las entradas de los tests API existentes, no un barrido exhaustivo de entradas nuevas sobre cada ruta hermana. No se ejecutó revisión nativa RDD (desactivada); assess pidió verificación independiente por cambios en composición compartida (riesgo item 3). Su clasificación nativa no pudo calcularse inicialmente por archivos no rastreados.

## Entrega

Issue #47 continúa abierta sin `status:approved`. No abrir PR hasta resolver ese gate. El PR deberá apuntar a `main`, incluir `Closes #47` y no fusionarse sin aprobación humana. El tamaño total supera 400 líneas; separar unidades en commits revisables no reduce el tamaño acumulado del PR.
