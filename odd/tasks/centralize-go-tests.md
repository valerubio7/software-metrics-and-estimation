# Centralizar las pruebas de Go

## Objetivo
Mover todas las pruebas de Go a `tests/unit/` o `tests/integration/`, preservando el comportamiento, `go test ./...` y el historial previo de TDD.

## Tareas
- [x] Extraer los helpers de inicialización del comando a un paquete interno importable para que las pruebas unitarias externas puedan ejercitarlos. Evidencia: `1a6c6cb`.
- [x] Reubicar todas las suites unitarias en `tests/unit/` y adaptar las pruebas del comando. Evidencia: `f15e3e9`.
- [x] Reubicar las pruebas de PostgreSQL en `tests/integration/` y corregir la búsqueda de migraciones. Evidencia: `a6cec35`.
- [x] Verificar todas las pruebas, incluida la ejecución real de contenedores PostgreSQL sin omisiones; actualizar las instrucciones de prueba y hacer commit de cada unidad de trabajo. Evidencia: `422fdd8` documenta los comandos; la verificación independiente confirmó que `go test -count=1 ./...` y ambas pruebas de integración reales de PostgreSQL finalizaron correctamente sin omisiones.

## Alcance
Solo organización de pruebas y extracción mínima de inicialización; sin comportamiento nuevo del producto.
