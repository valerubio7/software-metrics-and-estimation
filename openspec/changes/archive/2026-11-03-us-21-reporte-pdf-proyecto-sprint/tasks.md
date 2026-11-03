# Tareas del reporte PDF de proyecto o Sprint (US-21)

Implementación guiada por la [especificación](specs/reporte/spec.md) y el [diseño](design.md). Cada tarea incluye su verificación; las pruebas se escriben antes del comportamiento nuevo cuando la interfaz lo permite.

## 1. Preparación del módulo y dependencia PDF

- [x] 1.1 Crear la estructura de `internal/report` (dominio, aplicación, infraestructura PostgreSQL, infraestructura PDF y transporte HTTP) y verificar que `go list ./internal/report/...` reconoce los paquetes.
- [x] 1.2 Agregar `github.com/go-pdf/fpdf` en una versión fija y verificar `go mod tidy`, `go build ./...` y que la dependencia sea pure Go.

## 2. Dominio y aplicación

- [x] 2.1 Definir el snapshot y el reporte de dominio, con proyecto, Sprint, historias, tareas, esfuerzo, defectos, Sprints cerrados y métricas; pruebas unitarias verifican la derivación de métricas de proyecto y Sprint con `metrics/domain.Calculate` y `go test ./tests/unit/report/domain/...` pasa.
- [x] 2.2 Implementar la construcción del reporte y la atribución de esfuerzo: en Sprint se excluyen de la tabla los registros directos y los de tareas de otros Sprints, pero se conservan para la cobertura; pruebas unitarias cubren proyecto, Sprint, listas vacías y finalización por `completed_at`.
- [x] 2.3 Implementar el caso de uso de generación con validación de UUID, canonicalización, reloj inyectable y puerto de renderizado; pruebas unitarias cubren identificadores inválidos sin lecturas, errores de lectura, errores de renderizado y llamada única al renderizador; `go test ./tests/unit/report/application/...` pasa.

## 3. Lectura PostgreSQL

- [x] 3.1 Implementar el lector de snapshot en una transacción `RepeatableRead`/`ReadOnly` con verificación de proyecto y Sprint, consultas de historias, tareas, esfuerzo, defectos y Sprints cerrados; pruebas de integración cubren proyecto con historias dentro y fuera de Sprint, tareas de varios Sprints, esfuerzo directo y ligado, defectos detectados/resueltos en Sprints distintos y datos de otro proyecto.
- [x] 3.2 Verificar consistencia y solo lectura: el reporte calculado coincide con el lector de métricas de US-19 para proyecto y Sprint, las lecturas repetidas devuelven los mismos datos, las referencias inexistentes o ajenas devuelven los errores esperados y los conteos de las tablas no cambian; `go test ./tests/integration/report/...` pasa con PostgreSQL real.

## 4. Renderizado PDF

- [x] 4.1 Implementar el renderizador PDF con identificación, métricas y cobertura, historias, estimaciones, esfuerzo y defectos, incluyendo tablas, paginación, numeración y valores `null` como «No disponible».
- [x] 4.2 Agregar pruebas unitarias del renderizador con acentos, textos largos, listas vacías y más de una página; verifican firma `%PDF-`, cierre del archivo, presencia de secciones y contenido, y `go test ./tests/unit/report/infrastructure/pdf/...` pasa.

## 5. Transporte y composición

- [x] 5.1 Implementar el handler HTTP para ambas rutas con `GET`/`HEAD`, `Content-Type`, `Content-Disposition`, errores `422`/`404`/`405`/`500` y sin cuerpo en `HEAD`; pruebas de contrato cubren rutas ausentes, ambos ámbitos, métodos no admitidos, validación y errores sin fugas; `go test ./tests/unit/cmd/api/...` pasa.
- [x] 5.2 Componer `ReportDependencies` y el gate `Report` en `internal/api`, cablear el lector y el renderizador en `cmd/api` y probar v13, v14, `dirty`, error de resolución y dependencia ausente; `go test ./tests/unit/cmd/api/...` y `go build ./...` pasan.

## 6. Documentación y verificación integral

- [x] 6.1 Documentar en `README.md` las rutas, el contenido del PDF, la semántica de ámbito, los errores, el gate v14 y la ausencia de mutaciones; verificar que los ejemplos y rutas coinciden con las pruebas.
- [x] 6.2 Ejecutar `gofmt -l internal tests`, `go build ./...`, `go vet ./...`, `go test ./tests/unit/... -count=1`, `go test ./tests/integration/... -count=1` y `openspec validate us-21-reporte-pdf-proyecto-sprint --strict`; registrar resultados reales y cualquier límite de cobertura.

## Workflow follow-up

- Archivar el cambio y consolidar el spec de `reporte` después de la verificación y la revisión del usuario.
- Crear un PR a `main` con `Closes #49`; no fusionar sin decisión del usuario.
