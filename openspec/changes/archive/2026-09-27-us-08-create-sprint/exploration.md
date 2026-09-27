# Exploración: US-08 Crear un Sprint y definir su Sprint Goal

## Conclusión ejecutiva y fuentes

US-08 puede implementarse sin US-07: el corte mínimo es crear un Sprint asociado a un proyecto existente, registrar su Sprint Goal y dejarlo disponible para futuras asignaciones, sin consultar ni asignar historias. El repositorio ya tiene un patrón vertical para crear recursos asociados a proyectos (US-05), pero no tiene dominio, API ni persistencia de Sprint. La principal decisión de producto pendiente es qué significa que cada Sprint sea «únicamente identificable dentro de su proyecto»: la generación actual de UUID resuelve identidad técnica, pero la issue no define si también se requiere un número o nombre visible único por proyecto.

Alcance confirmado por el contexto autorizado de la issue #36: proyecto existente, identidad única dentro del proyecto, Sprint Goal, validaciones que impiden persistir datos obligatorios ausentes; no incluye asignar historias (US-09), registrar historias completadas (US-11), cerrar Sprint (US-12) ni consultar Sprints previos (US-13). US-07 solo consulta y muestra el Product Backlog y no es una dependencia. No se consultó GitHub ni la red durante esta exploración.

## Estado actual

- La API usa `net/http` y `http.ServeMux`. `internal/api/api.go` compone `POST /projects` y `POST /projects/{project_id}/stories`; US-05 establece el precedente de ruta anidada, `project_id` conocido, UUID validado en HTTP y sin endpoint de listado/selección.
- Los módulos siguen separación `domain` → `application` → `transport/http` e `infrastructure/postgres`. El caso de uso recibe un repositorio pequeño y generador de ID; el dominio valida campos y la capa HTTP traduce errores a `400`/`422`/`404`/`500` sin filtrar detalles internos.
- `projects` tiene `id UUID PRIMARY KEY`; `stories` referencia `projects(id)` con FK `ON DELETE RESTRICT`. El repositorio de historias convierte únicamente la violación de la FK nombrada en `ErrProjectNotFound`, evitando consultas previas y carreras.
- Las migraciones están en `internal/project/infrastructure/postgres/migrations/`; la API no las aplica y hoy habilita creación de historias a partir de la versión 2 limpia. Un Sprint probablemente requiera una migración posterior y una condición de habilitación acorde; debe revisarse el ensamblado sin afectar las rutas existentes.
- Hay pruebas unitarias por dominio/caso de uso/HTTP y pruebas PostgreSQL con Testcontainers para persistencia y FK, además de pruebas de composición de API. `openspec/config.yaml` declara `go test ./...` y TDD estricto. No se ejecutaron pruebas en esta exploración.
- OpenSpec mantiene especificaciones canónicas de proyectos e historias y cambios archivados para US-01 y US-05. No hay especificación, código ni migración de Sprint. La rama `feat/us-08-create-sprint` estaba limpia y basada en `origin/main` al iniciar.

## Áreas afectadas

- `internal/api/api.go` y `cmd/api/main.go` — registrar y componer creación de Sprint; decidir cómo habilitarla tras su migración sin alterar proyectos ni historias.
- `internal/sprint/domain/`, `internal/sprint/application/`, `internal/sprint/transport/http/`, `internal/sprint/infrastructure/postgres/` — candidato coherente con el módulo autónomo existente de historias; hoy no hay código Sprint.
- `internal/project/infrastructure/postgres/migrations/` — ubicación usada actualmente para el esquema compartido; añadir tabla y FK exigiría una nueva versión.
- `tests/unit/` y `tests/integration/` — replicar el corte por capas y verificar validaciones, asociación a proyecto, creación y almacenamiento en PostgreSQL; las pruebas actuales comprueban que Testcontainers requiere Docker y no omite silenciosamente si falla.
- `README.md` — documentar ruta, migración externa y prerequisitos de creación, como se hace con historias.
- `openspec/specs/` — hay requisitos canónicos de project e historia; no existe todavía un dominio/spec de Sprint.

## Enfoques considerados

1. **Crear Sprint mediante ruta anidada al proyecto (recomendado)** — `POST /projects/{project_id}/sprints`; generar un UUID de Sprint en servidor y persistir el proyecto asociado y el Sprint Goal.
   - Ventajas: sigue el precedente de US-05, expresa la asociación requerida y no necesita US-07, una consulta del backlog ni listado de proyectos.
   - Desventajas: no decide por sí solo si se necesita además un identificador de negocio legible y único dentro del proyecto.
   - Esfuerzo: Medio.

2. **Crear Sprint en una ruta superior incluyendo `project_id` en el body** — por ejemplo `POST /sprints` con asociación explícita en el JSON.
   - Ventajas: mantiene una colección de Sprints como recurso de primer nivel.
   - Desventajas: se aparta del único patrón existente de creación de un recurso perteneciente a un proyecto y hace menos directa la validación de ruta; no aporta valor confirmado por la issue.
   - Esfuerzo: Medio.

## Recomendación

Seguir el corte vertical de US-05 con una ruta anidada y FK a `projects`, una validación del Sprint Goal antes de persistir y un UUID generado por servidor. Un diseño inicial de persistencia podría usar `sprints(id UUID PRIMARY KEY, project_id UUID NOT NULL REFERENCES projects(id) ON DELETE RESTRICT, sprint_goal TEXT NOT NULL)`. La FK permite rechazar proyectos inexistentes al insertar y evita huérfanos sin una lectura previa. Mantener la creación independiente de Product Backlog y no introducir asignación de historias.

Antes de cerrar la propuesta, definir si UUID basta para «únicamente identificable» o si se requiere un atributo visible (por ejemplo, número/nombre) con unicidad compuesta `(project_id, <atributo>)`. La issue proporcionada no especifica ese atributo ni reglas de formato. No inferir número, fechas, duración, estado, orden o unicidad por Sprint Goal; tampoco asumir que «información obligatoria» agrega campos aparte de proyecto y goal.

## Riesgos y decisiones pendientes

- **Semántica de identidad:** UUID global ya identifica técnicamente cada fila, pero podría no satisfacer una necesidad de identificación funcional dentro de un proyecto. Hace falta decidir si se solicita un número/nombre adicional y su política de unicidad.
- **Validación del goal:** tratar Sprint Goal como obligatorio está respaldado por la issue, pero faltan reglas de aceptación para espacios, límites de longitud y normalización; tomar como referencia validación no vacía sin inventar restricciones de texto.
- **Proyecto inexistente e integridad concurrente:** usar una FK en una sola escritura; no depender de US-07 ni de consulta previa para comprobar el proyecto.
- **Habilitación de ruta:** `cmd/api/main.go` comprueba actualmente versión limpia `>= 2` para historias. Una migración de Sprint requiere que su disponibilidad se controle separadamente o mediante versión suficiente, preservando la API cuando la migración no esté aplicada.
- **Presupuesto de revisión:** módulo, ruta, migración, composición, pruebas y documentación pueden acercarse o superar 400 líneas; la estrategia `ask-on-risk` requiere escalar el riesgo antes de implementar una entrega sobredimensionada.
- La exploración se basa en el texto de issue #36 entregado por el orquestador y evidencia local; no se ejecutaron pruebas ni comandos de red.

## Listo para propuesta

**Sí, con una decisión explícita pendiente en la propuesta:** indicar que UUID generado es la identidad suficiente para esta historia o solicitar definición de identificador legible/unicidad por proyecto. La ruta anidada, el Sprint Goal obligatorio, la FK y la exclusión de US-07/US-09 tienen sustento en el contexto y los patrones locales; no se necesita bloquear por dependencia con US-07.
