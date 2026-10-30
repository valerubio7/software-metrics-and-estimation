# Tasks

## 1. Completar la consulta de métricas en la TUI

- [x] 1.1 Agregar cliente GET y DTO compatibles con el contrato de US-19, con pruebas que fijen ambos endpoints y la interpretación de `null`, cero y cobertura; verificar con `go test ./tests/unit/tui -run 'TestMetrics' -count=1`.
- [x] 1.2 Integrar carga, refresco, errores, retorno y presentación de Proyecto/Sprint en la sección existente según `specs/tui/spec.md`; verificar 80/44 columnas y confirmar que solo se emiten GET con `go test ./tests/unit/tui`.
- [x] 1.3 Actualizar README y especificación TUI para reflejar que Métricas ya consulta la API, sin fórmulas locales; verificar la documentación y `openspec validate tui-10-consultar-metricas`.

## 2. Verificación de integración

- [ ] 2.1 Ejecutar `go test ./...` y registrar el resultado; confirmar que TUI-10 solo depende de que US-19 sea integrado para la verificación end-to-end, sin fusionar ramas remotas. Última ejecución: falló en `tests/integration/defect/postgres` porque la migración 000014 vuelve a agregar `resolution_sprint_id`; no modificar ese alcance sin autorización.

## Workflow follow-up

- Archivar el cambio OpenSpec después de completar las tareas y satisfacer la revisión requerida; verificar que la especificación principal `openspec/specs/tui/spec.md` quede actualizada.
