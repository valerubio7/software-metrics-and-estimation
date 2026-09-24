# Exploración: US-05 Crear una historia en el Product Backlog

## Resultado y fuentes

La US-05 (#33, abierta) pide crear una historia asociada a un proyecto existente y conservar identificador, título, descripción, prioridad, estado, Story Points y criterios de aceptación. Una historia puede quedar pendiente de estimación hasta US-14. Esta exploración **no fija todavía** el contrato HTTP ni las reglas de obligatoriedad ambiguas.

Fuentes: texto y criterios completos de `gh issue view 33 --json number,title,body,state,url` obtenidos por el orquestador y comunicados a este ejecutor (no pude ejecutar `gh` aquí, porque no existe herramienta de comandos); `README.md`, `openspec/config.yaml`, `openspec/specs/project/spec.md`, código y pruebas de US-01. URL: https://github.com/valerubio7/software-metrics-and-estimation/issues/33. No había artefacto previo de exploración para esta historia. El preflight recibido es interactivo, OpenSpec, ask-on-risk y 400 líneas; los artefactos de esta sesión se redactan en español.

## Requisitos constatados de #33

- Seleccionar un proyecto existente y crear la historia en el Product Backlog de ese proyecto; datos válidos implican registro asociado, datos obligatorios ausentes implican que **no se crea**.
- Cada historia tiene identificador, título, descripción, prioridad, estado, campo de Story Points y criterios de aceptación.
- Story Points puede permanecer inicialmente pendiente de estimación; la estimación acordada corresponde a US-14.
- No incluye modificar historias (US-06), consultar Product Backlog (US-07), obtener estimación por Planning Poker (US-14) ni asignar a Sprint (US-09).

## Arquitectura actual e integración

| Área | Evidencia y punto de integración probable |
| --- | --- |
| Entrada | `internal/api/api.go` compone `http.NewServeMux` y registra únicamente `POST /projects`; `cmd/api/main.go` crea `pgxpool.Pool`, comprueba conectividad y compone la API. Habría que integrar una ruta de creación de historia, todavía sin definir su forma ni fingir una interfaz de selección de proyectos. |
| Dominio/aplicación | `internal/project/domain/project.go` define `Project` y `ValidationError` por campo; `internal/project/application/create_project.go` valida y persiste mediante interfaz pequeña e ID inyectado. Un módulo/caso de uso de historias podría seguir estos límites sin añadir campos al modelo de proyecto por anticipado. |
| Persistencia | `internal/project/infrastructure/postgres/repository.go` solo inserta proyectos; `000001_create_projects.up.sql` define `projects(id UUID PRIMARY KEY, name, start_date, planned_finish_date)`. La asociación duradera de una historia con proyecto requiere comprobar existencia y definir esquema de historias con referencia a `projects`, idealmente reforzada por clave foránea; el mecanismo de lectura/verificación de proyectos aún no existe. |
| Contrato y errores | `internal/project/transport/http/handler.go` separa JSON inválido (`400`), validación (`422`), éxito (`201`) y fallo inesperado (`500` sin detalles internos). Son precedentes, no decisiones automáticas para proyecto inexistente ni para historia. |
| Pruebas | `tests/unit/project/{domain,application,transport/http}/`, `tests/unit/cmd/api/main_test.go` y `tests/integration/project/postgres/repository_integration_test.go` cubren el corte US-01. Integración usa Testcontainers/PostgreSQL y omite pruebas si Docker no está disponible; no se ejecutaron pruebas en esta exploración. `README.md` y `openspec/config.yaml` indican `go test ./...`; TDD estricto rige trabajo futuro. |

No hay módulo de historia/backlog ni operación de lectura de proyectos en el código revisado. El README solo documenta crear proyectos; no hay endpoint para listarlos, de modo que «seleccionar un proyecto» no implica automáticamente implementar una UI o una lista en US-05.

## Preguntas de negocio para antes de la propuesta

1. ¿La selección del proyecto es por un `project_id` ya conocido en una API de creación, o US-05 necesita descubrir/listar proyectos, pese a que US-01 solo implementó `POST /projects`?
2. ¿Qué campos son obligatorios **al crear**? En particular, ¿Story Points se omite/queda `null` hasta US-14 y se prohíbe que US-05 acepte una estimación acordada? ¿Hay diferencia entre pendiente y `0`?
3. ¿Cuáles son los valores permitidos y el valor inicial de prioridad y estado? ¿Se reciben del usuario o los define el sistema al crear la historia?
4. ¿Cómo se registran los criterios de aceptación: lista de criterios no vacía, texto único, formato libre? ¿Se rechaza una lista vacía o entradas en blanco?
5. ¿Qué debe ocurrir si el proyecto no existe o deja de existir durante la creación? ¿Se rechaza de manera distinguible y se impide toda historia huérfana?

## Riesgos, límites y siguiente paso

- **Ambigüedad de producto:** el criterio «si falta algún dato obligatorio» no identifica cuáles de los campos enumerados son obligatorios; imponer Story Points obligatorio contradiría la observación de estimación pendiente.
- **Integridad y alcance:** sin lectura/verificación de proyectos ni clave foránea para historias, no se garantiza la asociación exigida; listar proyectos o construir UI sería expansión de alcance sin confirmación.
- **Contrato:** ruta, representación, errores de proyecto inexistente, validación de valores y política de estimación quedan abiertos; no derivarlos silenciosamente de US-01.
- **Revisión:** nuevo dominio, aplicación, endpoint, migración, repositorio y pruebas probablemente ponen en riesgo el presupuesto de 400 líneas; `ask-on-risk` exige consultar al usuario si aparece ese riesgo antes de escoger cadena o excepción.
- **Validación futura:** TDD estricto con secuencia cronológica RED/GREEN/REFACTOR y `go test ./...`, incluyendo prueba PostgreSQL realmente ejecutada para demostrar asociación e integridad; no equivale a cobertura real una prueba omitida por Docker.

Siguiente paso: resolver las preguntas de negocio con el usuario antes de redactar propuesta; en modo interactivo esta exploración **no autoriza** avanzar de fase. No se implementó código ni se ejecutaron pruebas.
