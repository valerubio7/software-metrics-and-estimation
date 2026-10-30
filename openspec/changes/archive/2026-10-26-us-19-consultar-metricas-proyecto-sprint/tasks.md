# Tareas de implementación de US-19

Implementación y verificaciones completadas; la evidencia y los límites se registran en `archive-report.md`. Cada regla y escenario de [la especificación](specs/metricas/spec.md) debe verificarse por interfaz pública, con pruebas deterministas antes del comportamiento nuevo.

## 1. Dominio y aplicación de lectura (S2-S7)

- [x] 1.1 Incorporar contrato de lectura y valores anulables con cobertura; pruebas unitarias de interfaz pública cubren puntos, finalización independiente del estado, historias únicas, porcentaje sin denominador y velocidad cerrada, incluyendo ceros y faltantes; RED/GREEN observado y `go test ./tests/unit/...` pasa.
- [x] 1.2 Implementar cálculos de esfuerzo y desviaciones; pruebas públicas cubren tareas por Sprint, reemplazo por historia, faltantes, esfuerzo atribuible parcial/ausente, registros directos no atribuibles, filas iguales independientes, signos y denominador cero; `go test ./tests/unit/...` pasa. Fórmulas y unidades de cobertura documentadas.
- [x] 1.3 Validar identificadores y traducir errores de referencias/almacenamiento reutilizando convenciones existentes; pruebas cubren UUID inválidos sin acceso al repositorio, errores de referencias y fallos sin resultados parciales; `go test ./tests/unit/...` pasa.

## 2. Agregación PostgreSQL (S2-S7)

- [x] 2.1 Implementar lectura consistente por ámbito con conjuntos base únicos y preagregaciones de tareas/esfuerzo; verificar por interfaz pública datos cruzados entre proyectos y múltiples Sprints/tareas/registros, sin multiplicación ni deduplicación de filas iguales; incluir una historia compartida entre dos Sprints con registros directos y registros ligados a tareas de cada Sprint, comprobando total de proyecto, exclusión cruzada y contadores de cobertura, con `go test ./tests/integration/...`; documentar la estrategia junto a las consultas.
- [x] 2.2 Agregar velocidad por Sprint cerrado y conteos independientes de detección/resolución; verificar Sprints abiertos excluidos del promedio, cerrados vacíos e incompletos, detección en A/resolución en B y cero defectos con `go test ./tests/integration/...`.
- [x] 2.3 Verificar no mutación y consistencia mediante consultas públicas repetidas, vacías y rechazadas y lecturas públicas antes/después de los datos fuente; ejecutar `go test ./tests/integration/...` y conservar evidencia de conteos y datos sin cambios.

## 3. API y composición (S1,S7,S8)

- [x] 3.1 Exponer ambas rutas con contrato plano, `deviation_percentage` y `coverage`; pruebas HTTP verifican códigos/cuerpos y claves exactas, nulabilidad, GET/HEAD, métodos no admitidos y errores; `go test ./tests/unit/cmd/api/... -count=1` pasa. README documenta ejemplos completos por ámbito, incluyendo `coverage.actual_hours.unattributed_story_level_records` y `stories_without_records`.
- [x] 3.2 Cablear dependencia opcional y gate limpio >=14 reutilizando resolución/composición existentes; pruebas unitarias verifican v13, v14, `dirty`, error de resolución y dependencia ausente; matriz de rutas y arranque documentadas en README.
- [x] 3.3 Preservar operaciones existentes del código compartido: diferencial manual de los 55 tests API existentes contra base `5564d8932bfc61fc8244191e98ec224e7e020ffe`, con stdout normalizado, stderr y código de salida idénticos; no se incorpora lectura de Git en tests permanentes.

## 4. Verificación integral (S1-S8)

- [x] 4.1 Ejecutar `go test ./tests/unit/...`, `go test ./tests/integration/...`, `go build ./...` y `go vet ./...` con Go 1.27.1 (versión observada; no se ejecutó con 1.27.0 exacto) y prerrequisitos de integración disponibles; registrar resultados reales y cualquier fallo o comando no ejecutado.
- [x] 4.2 Verificar trazabilidad de todos los escenarios, documentación de API y matriz de rutas, ausencia de cambios TUI/US-20/US-21 y compatibilidad de claves de PR #115; ejecutar `openspec validate us-19-consultar-metricas-proyecto-sprint --strict` y dejar informe de cierre de implementación para revisión del orquestador.

## Seguimiento de implementación

- El orquestador revisa la entrega y archiva el cambio al finalizar, verificando el resultado archivado; no archivar estos artefactos antes de implementar y verificar.
- El orquestador conserva las decisiones de commits y publicación de un PR a `main` con `Closes #47`; no fusionar sin decisión del usuario.
