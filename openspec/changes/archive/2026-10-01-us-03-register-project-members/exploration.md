# Exploración: US-03 — Registrar integrantes en un proyecto (`us-03-register-project-members`)

## Alcance y fuente

Issue #31 solicita elegir un proyecto existente y registrar uno o más integrantes asociados, con información identificativa mínima; entradas incompletas/ inválidas no deben registrar nada y deben explicar el motivo. No se deben modificar los datos básicos del proyecto. US-01 (crear proyecto), US-02 (modificar datos básicos) y US-04 (estado) quedan fuera. El nombre completo requerido y el email opcional son una **decisión de dominio aportada por el usuario para esta sesión**, no un requisito atribuible al issue.

La configuración activa declara `openspec` como almacén, Go, TDD estricto y `go test ./...` (`openspec/config.yaml`). El árbol del repo no incluye un dominio, especificación ni migración actuales de miembros.

## Estructura y estado actual

- `internal/api/api.go` compone `net/http` con `POST /projects`, `PUT /projects/{project_id}` y rutas opcionales de historias/Sprints. Los módulos opcionales dependen de migraciones comprobadas en el arranque.
- `internal/project/` contiene dominio, casos de uso, transporte HTTP y repositorio PostgreSQL de proyectos; `projects` solo tiene ID, nombre y fechas básicas. La actualización SQL restringe sus cambios a esos campos.
- Historias ya usan rutas anidadas (`/projects/{project_id}/stories...`) y capas similares de application, transport e infraestructura. Sprint también tiene módulo separado.
- Pruebas: unitarias bajo `tests/unit/{project,story,sprint}` y de integración con PostgreSQL/Testcontainers bajo `tests/integration`. La configuración indica Docker como prerrequisito de integración.
- OpenSpec mantiene `openspec/specs/project/spec.md` y cambios archivados. El cambio solicitado no existe todavía; el artefacto pedido es solo esta exploración.

## Áreas probablemente afectadas

1. Nuevo módulo de integrantes — dominio/validación de miembro, contratos/caso de uso de alta y transporte HTTP; definir si lote y elemento individual se modelan explícitamente.
2. Composición de API — ruta anidada para integrantes del proyecto y dependencias opcionales condicionadas por disponibilidad de esquema, en consonancia con la estrategia de arranque existente.
3. PostgreSQL — nueva tabla/migración con referencia al proyecto, datos identificativos y política de eliminación/consistencia por decidir. La configuración advierte que existen dos migraciones `000003` en el directorio compartido y que su reconciliación sigue pendiente; cualquier numeración nueva debe investigarse antes de proponerse.
4. Pruebas unitarias del dominio, aplicación y HTTP; pruebas de integración del repositorio/API para persistencia, asociación y atomicidad del lote.
5. Delta OpenSpec futuro de la capacidad de miembros (no redactado en esta fase).

Estas son áreas a explorar en propuesta, no un diseño ya decidido.

## Estrategia de pruebas para una fase posterior

- Unitarias de validación de nombre completo y email opcional; cuerpos ausentes/malformados y datos no válidos; verificar que el repositorio no recibe escrituras si el lote contiene un elemento inválido.
- Pruebas HTTP para selección por ID del proyecto, respuestas de éxito/error y comportamiento ante proyecto inexistente, según contrato acordado.
- Integración PostgreSQL para persistencia y asociación al proyecto; confirmar si lote inválido implica cero filas y si lote válido es atómico (transacción) o admite éxito parcial.
- Correr `go test ./...` como suite configurada; las pruebas Testcontainers requieren Docker. Las pruebas deben seguir TDD estricto.

## Riesgos y preguntas abiertas

- “Seleccionar un proyecto existente” no precisa cómo lo selecciona el usuario/cliente. No se encontró una ruta de listado/consulta de proyectos en el API leído; la ruta anidada con ID puede ser contrato de API, pero debe confirmarse en propuesta, sin inferir una interfaz de selección.
- “Uno o más” no define si la solicitud es un lote atómico, permite resultados parciales ni qué sucede con entradas duplicadas. Evitar registros parciales ante lote inválido parece alinearse con el criterio de no registrar ante entrada inválida, pero atomicidad requiere precisión.
- “Información identificativa mínima” no define campos. Nombre completo requerido/email opcional provienen de la decisión de usuario, no del issue. No se especifican formato/normalización del nombre o email, ni unicidad/duplicados.
- No se define respuesta al proyecto inexistente, método/ruta, códigos HTTP ni esquema de errores; tampoco política de eliminación/retención de integrantes vinculados.
- Migraciones numeradas conflictivas ya documentadas pueden complicar habilitar la ruta sin romper el arranque condicional existente.
- Se debe mantener separado el modelo de integrante de las fechas/nombre del proyecto; evitar cambios de datos básicos y exclusiones US-01/02/04.

## Preparación para propuesta

**Parcialmente listo; requiere decisiones antes de una propuesta normativa.** Hay suficiente evidencia para delimitar un corte vertical probable en un módulo de miembros anidado al proyecto, con persistencia PostgreSQL y cobertura por capas. Antes de proponer, confirmar contrato de selección/ID, atomicidad y duplicados, tratamiento de proyecto inexistente y alcance exacto del modelo de identificación; mantener explícito que nombre requerido/email opcional es decisión de sesión. Investigar numeración/estado de migraciones. No se ejecutaron pruebas porque esta fase es read-only y no implementa.
