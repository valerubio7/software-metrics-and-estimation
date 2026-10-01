# Design: US-04 — Consultar el estado de un proyecto

## Estado y alcance

Diseño únicamente; no se implementa código. Conserva exactamente las reglas confirmadas: `planned` antes de `start_date`; `active` desde `start_date` hasta `planned_finish_date`, ambos inclusive; `overdue` después de esa fecha. Nunca se devuelve `completed`. Lectura pura, sin migración ni escrituras, y sin dependencia de US-03.

## Decisiones

- **HTTP:** `GET /projects/{project_id}`, consistente con las rutas actuales (`POST /projects`, `PUT /projects/{project_id}`) y el uso de `PathValue` de `net/http`.
- **Éxito:** `200 application/json`, objeto `{ "id": "<uuid>", "name": "...", "status": "planned|active|overdue" }`. Sin fechas adicionales requeridas por este contrato; la identidad y el estado solicitado son suficientes.
- **ID inválido:** mantener exactamente la convención de `UpdateProjectHandler`: UUID inválido produce `422`, `error: validation_failed`, mensaje común y `fields.project_id: "must be a valid UUID"`. Validar antes de acceder al caso de uso/repositorio.
- **No encontrado:** reutilizar `application.ErrProjectNotFound`, mapeado como en actualización a `404` y `{ "error":"project_not_found", "message":"project not found" }`.
- **Fecha actual:** cálculo recibe una fecha de referencia date-only explícita/injectable (idealmente `time.Time` normalizada por año/mes/día o un tipo de fecha pequeña). La composición productiva obtiene `time.Now().UTC()` y toma sus componentes de calendario UTC; UTC es el valor predeterminado pragmático del servicio, no un requisito de producto ni política de zona horaria. No se aceptan fechas del cliente. Inyectar la fuente/fecha hace deterministas las pruebas.
- **Cálculo:** función pura en dominio, compara componentes de calendario date-only, nunca duración/instantes ni persistencia. El dominio ya garantiza fin no anterior al inicio. No añadir estado al agregado ni esquema.

## Arquitectura y flujo de datos

1. `internal/api/api.go` registra `GET /projects/{project_id}` junto a creación/actualización y compone el caso de uso de consulta usando el repositorio existente.
2. Handler parsea UUID conforme al contrato existente y pasa ID/contexto al caso de uso; la fuente de fecha del servicio es inyectable en composición (reloj/función de fecha).
3. Caso de uso de consulta pide al puerto de lectura el proyecto por ID; calcula estado mediante función pura de dominio con la fecha recibida y presenta datos para respuesta.
4. `PostgresProjectRepository` ejecuta un `SELECT` parametrizado por UUID para leer ID, nombre, fechas. `pgx.ErrNoRows` se convierte a `application.ErrProjectNotFound`. La consulta es de solo lectura y no depende de tablas de historias.
5. Handler serializa solamente identidad y estado; errores siguen el formato JSON existente. Errores de infraestructura se ocultan como `500 internal_error`.

## Cambios previstos por archivo

- `internal/project/domain/project.go`: tipo/función pura de estado derivado y comparación date-only (preferible archivo dedicado `status.go` para separar la regla); tabla de estados restringida a los tres valores.
- `internal/project/application/`: puerto de lectura por ID (separarlo del puerto de escritura si conviene a contratos mínimos), caso de uso de consulta que recibe la fecha de referencia.
- `internal/project/infrastructure/postgres/repository.go`: método de lectura `GetByID`/`FindByID` en el mismo adaptador y mapeo de ausencia; seleccionar solo columnas existentes.
- `internal/project/transport/http/handler.go` o handler dedicado: ruta GET, UUID/error mapping y DTO de respuesta `{id,name,status}`.
- `internal/api/api.go`: registrar ruta e inyectar fuente de fecha actual productiva.
- `tests/unit/project/domain/`: tabla de pruebas de clasificación, límites inclusivos y fuente fija.
- `tests/unit/project/application/` y `tests/unit/project/transport/http/`: lectura exitosa, ausencia, UUID inválido, errores y que no hay escrituras.
- `tests/integration/project/postgres/repository_integration_test.go` y/o integración HTTP de proyecto: lectura real, ausencia, respuesta y cero mutaciones.
- `openspec/specs/project/spec.md`: incorporar el requisito aprobado en el delta al spec vigente después de implementar/verificar; documentación API existente solo si ya hay una ubicación estable.

## Contratos y errores

Puerto de lectura: `GetByID(ctx, canonicalUUID) (domain.Project, error)` o equivalente; missing usa el sentinel compartido existente. El caso de uso no expone errores SQL. La función de estado debe aceptar proyecto válido y fecha de calendario de referencia, y devolver solo `planned`, `active` o `overdue`. `completed` no forma parte del tipo/contrato. La ruta inválida no toca la base de datos; ausencia no se presenta como éxito vacío.

## Pruebas

- Unitarias de dominio con fechas de referencia fijadas: víspera del inicio `planned`, día de inicio `active`, día de fin `active`, día siguiente `overdue`; proyecto de un día activo exactamente ese día; año/mes y cambio de día para proteger comparación date-only.
- Unitarias de aplicación: propaga ID, usa la fecha inyectada, traduce ausencia/sin datos y no invoca métodos de escritura.
- Unitarias HTTP/API: GET exitoso contiene id/nombre/estado únicamente según contrato; UUID malformado da 422 y cero llamadas; ausencia da 404 con error estándar; error interno no filtra detalles; método/ruta correctos y dependencias US-03 ausentes.
- Integración PostgreSQL: `SELECT` por ID devuelve fechas/name/id, ID ausente corresponde al sentinel, y datos persisten iguales tras consulta. Integración HTTP si sigue el patrón de suites actuales.
- Ejecutar suite Go completa y pruebas de integración disponibles; las de PostgreSQL usan testcontainers y pueden omitir por Docker no disponible, informándolo.

## Riesgos, rollout y reversión

Riesgo principal: confundir date-only con instante o cambiar límites por conversión de zona; evitarlo con comparación de componentes y pruebas de frontera. La fecha UTC es una decisión operacional pragmática explícita; si el servicio requiere otro calendario operativo, sería una decisión futura aparte, no un requisito añadido aquí. Otro riesgo es duplicar el manejo 404/422; reutilizar sentinel y forma existentes.

Rollout aditivo: publicar ruta y cálculo juntos; no hay migración ni backfill ni modificación de datos. Verificar tests antes de habilitar release. Reversión consiste en retirar ruta, caso de uso, lectura y tests/documentación asociados; no hay estado que restaurar. La estimación debe mantenerse bajo referencia de 400 líneas y estrategia autorizada de single-PR. El usuario ya autorizó explícitamente `size:exception` si fuera necesario; si el forecast supera 400 líneas, documentar el riesgo y aplicar solo dentro de esa autorización, sin ampliar alcance.

## Resolución de skill

No se inyectaron rutas de skill de fase. Fallback explícito al skill general `gentle-ai`; no se identificó skill SDD específica disponible en las rutas provistas. `skill_resolution: fallback-path`.
