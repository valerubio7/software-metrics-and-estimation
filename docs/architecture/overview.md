# Arquitectura actual del backend

El sistema es un monolito modular Go con persistencia PostgreSQL. Su alcance implementado es crear/modificar proyectos, crear/modificar historias, consultar el Product Backlog y crear Sprints; no es aún la aplicación completa de la [consigna](../assignment.md).

## Flujo y límites de módulos

```text
Cliente HTTP → net/http → transporte → aplicación → puerto de repositorio
                                      ↓                    ↓
                                    dominio           PostgreSQL (pgxpool)
```

| Área | Responsabilidad y ubicación |
|---|---|
| Arranque | [cmd/api/main.go](../../cmd/api/main.go): configuración, pool, ping, lectura de migraciones y servidor. |
| Composición HTTP | [internal/api/api.go](../../internal/api/api.go): `NewHTTPHandlerWithDependencies` registra rutas e inyecta dependencias opcionales. |
| Proyectos | [internal/project](../../internal/project/): datos básicos, fechas, creación y reemplazo. |
| Historias | [internal/story](../../internal/story/): validaciones, estado, horas estimadas y orden del backlog. |
| Sprints | [internal/sprint](../../internal/sprint/): creación asociada a un proyecto y Sprint Goal. |

Cada módulo separa `domain`, `application`, `transport/http` e `infrastructure/postgres`. Los casos de uso dependen de interfaces de repositorio; los adaptadores PostgreSQL las implementan. El transporte convierte JSON, identificadores y errores en respuestas HTTP. La composición conecta estas capas, sin un framework web adicional.

Las [dependencias declaradas](../../go.mod) incluyen Go 1.27.0, pgx v5 para PostgreSQL y Testcontainers para integración. Las pruebas no son dependencias de ejecución del servidor.

## Las seis operaciones de negocio

| US | Método y ruta | Resultado principal |
|---|---|---|
| 01 | `POST /projects` | Proyecto con nombre y fechas. |
| 02 | `PUT /projects/{project_id}` | Reemplazo de los tres datos básicos. |
| 05 | `POST /projects/{project_id}/stories` | Historia pendiente, sin Story Points ni horas estimadas iniciales. |
| 06 | `PUT /projects/{project_id}/stories/{story_id}` | Reemplazo de seis campos editables; identidad y Story Points intactos. |
| 07 | `GET /projects/{project_id}/stories` | Backlog completo ordenado por prioridad y creación. |
| 08 | `POST /projects/{project_id}/sprints` | Sprint con UUID y Sprint Goal, sin asignar historias. |

El listado usa un contenedor con `project_id` y `stories`; un proyecto sin historias devuelve `[]`, no `null`. Las horas estimadas son distintas del esfuerzo real. Consultar una historia individual, gestionar integrantes o asignar historias al Sprint no forman parte de estas seis operaciones.

## Disponibilidad al arrancar

El servidor consulta una única tabla global `schema_migrations`. Las dependencias opcionales se habilitan solo si la lectura no falla y `dirty` es falso:

| Funcionalidad opcional | Versión mínima |
|---|---|
| Crear historias | 2 |
| Modificar historias | 3 |
| Consultar backlog | 4 |
| Crear Sprints | 3 |

Las rutas de proyectos se registran independientemente de este gate; eso no garantiza que sus tablas estén disponibles. Un estado `dirty` o un error de lectura inhibe todas las rutas opcionales. La decisión ocurre en el arranque, no mediante un chequeo por solicitud.

Las migraciones son externas: el bootstrap no las aplica. Todos los SQL están en el [directorio compartido de migraciones](../../internal/project/infrastructure/postgres/migrations/).

**Riesgo pendiente:** allí coexisten `000003_add_story_estimated_hours` y `000003_create_sprints`, ambos con archivos `up`/`down`. La numeración duplicada debe reconciliarse antes de confiar en una secuencia de despliegue. Un número global de versión no acredita por sí solo que ambas estructuras existan. No se afirma que `migrate up` funcione ni que las pruebas certifiquen un despliegue completo.

## Alcance pendiente

La superficie actual no incorpora frontend, autenticación, Planning Poker, registro de esfuerzo real, gestión de defectos, cálculo de métricas, dashboard ni reportes. Son capacidades pendientes frente a la consigna, no servicios ocultos dentro de este backend.

Para verificar comportamientos concretos, seguir la [trazabilidad](../traceability.md). Esta descripción no constituye una guía de instalación ni una nueva ejecución de pruebas.
