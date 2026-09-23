# Diseño: US-01 Crear proyecto

## Objetivo y alcance

US-01 implementa el corte vertical mínimo para crear y persistir un proyecto. El dato de dominio es `Project` con `id`, `name`, `start_date` y `planned_finish_date`; no incorpora estado, miembros, actualización ni consulta de estado.

## Límite HTTP

La API expone `POST /projects` y recibe JSON:

```json
{
  "name": "Portal de métricas",
  "start_date": "2026-03-01",
  "planned_finish_date": "2026-06-30"
}
```

Las fechas se intercambian como fechas de calendario ISO 8601 (`YYYY-MM-DD`), sin componente horario. Ante una creación exitosa responde `201 Created` con los datos persistidos:

```json
{
  "id": "<uuid>",
  "name": "Portal de métricas",
  "start_date": "2026-03-01",
  "planned_finish_date": "2026-06-30"
}
```

El handler decodifica la solicitud, transforma el DTO en `CreateProjectCommand`, invoca el caso de uso y serializa el resultado. La respuesta no incluye ningún campo `status`.

| Situación | Respuesta |
| --- | --- |
| JSON malformado, tipo de campo incompatible o cuerpo vacío | `400 Bad Request` con un error de solicitud accionable |
| Nombre o fecha obligatoria ausente/inválida; fecha final anterior a la inicial | `422 Unprocessable Content` con errores por campo o de regla de negocio |
| Proyecto creado | `201 Created` y la representación anterior |
| Error inesperado de infraestructura/persistencia | `500 Internal Server Error`, sin exponer detalles internos |

Un formato de error inicial puede ser:

```json
{
  "error": "validation_failed",
  "fields": {
    "planned_finish_date": "must be on or after start_date"
  }
}
```

## Dominio y caso de uso

`Project` representa los datos básicos persistibles. `CreateProjectCommand` contiene los valores recibidos por la aplicación, y `CreateProjectService` (o `CreateProjectUseCase`) coordina la operación:

1. valida el comando y las invariantes de `Project`;
2. crea la entidad con un identificador;
3. persiste la entidad mediante `ProjectRepository`;
4. devuelve el proyecto creado.

La validación de reglas de negocio pertenece al dominio/caso de uso, no al handler ni al repositorio: `name` debe existir y no ser blanco; ambas fechas deben ser válidas y obligatorias; `planned_finish_date` debe ser igual o posterior a `start_date`. El handler solo valida la estructura y la decodificación HTTP para poder distinguir errores de protocolo (`400`) de errores de dominio (`422`). El repositorio no decide reglas de creación y solo persiste una entidad ya válida.

La abstracción propuesta es deliberadamente pequeña:

```go
type ProjectRepository interface {
    Create(ctx context.Context, project Project) error
}
```

El servicio recibe esa interfaz por constructor para mantener el dominio independiente de PostgreSQL y facilitar dobles de prueba.

## Persistencia PostgreSQL

PostgreSQL es el destino de persistencia. La tabla `projects` debe almacenar únicamente los datos de US-01:

```sql
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    start_date DATE NOT NULL,
    planned_finish_date DATE NOT NULL,
    CHECK (planned_finish_date >= start_date)
);
```

La restricción `CHECK` replica defensivamente la invariante de dominio, pero no reemplaza la validación previa del caso de uso ni define la respuesta HTTP. Un `PostgresProjectRepository` implementa `ProjectRepository` mediante un `INSERT` parametrizado y reside en la capa de infraestructura, no en el paquete de dominio. La configuración de conexión y el ensamblado de dependencias quedan en el proceso de API.

## Organización de paquetes y archivos

La estructura propuesta respeta el esqueleto Go actual y mantiene el flujo de dependencias hacia adentro:

```text
cmd/api/main.go                                  # configuración y composición HTTP/DB
internal/project/domain/project.go               # Project e invariantes
internal/project/application/create_project.go   # comando, caso de uso e interfaz de repositorio
internal/project/transport/http/handler.go       # POST /projects y DTOs HTTP
internal/project/infrastructure/postgres/repository.go # implementación SQL
internal/project/infrastructure/postgres/migrations/   # migración de projects, según herramienta elegida
internal/shared/                                 # solo utilidades transversales justificadas
```

`cmd/api` no contiene reglas de negocio ni SQL. `internal/shared` no se crea por anticipado: se usa únicamente si una dependencia o utilidad es realmente común a más de un módulo.

## Estrategia de pruebas y TDD

El proyecto declara TDD estricto y el runner obligatorio es `go test ./...`. La implementación debe seguir y registrar las etapas RED, GREEN, TRIANGULATE y REFACTOR.

- **Handler:** pruebas con `httptest` para el `201`, JSON inválido (`400`), campos faltantes, fechas inválidas y fecha final anterior (`422`); verifican que no se invoque la creación cuando la entrada no es válida.
- **Caso de uso/dominio:** pruebas unitarias con un `ProjectRepository` fake para las invariantes, la asignación de identificador y la invocación de persistencia solo para comandos válidos.
- **Repositorio PostgreSQL:** pruebas de integración para confirmar el `INSERT` y las columnas de `projects`. Se usarán `testcontainers-go` o una base de datos de prueba existente si el repositorio/proceso ya provee uno. Si no existe ninguno, se conserva como restricción explícita de diseño: no se debe simular PostgreSQL como sustituto de esa prueba de integración; hay que acordar y preparar el entorno antes de declarar cubierta la persistencia real.

Cada cambio se valida con:

```text
go test ./...
```

## Fuera de alcance

- Estado de proyecto: entrada, persistencia, cálculo y respuesta.
- Miembros de proyecto.
- Actualización de proyectos.
- Consulta del estado del proyecto.
- Funcionalidades de métricas, estimación, seguimiento, paneles o reportes.

## Riesgos y decisiones abiertas

- **Migraciones:** el repositorio no declara aún una herramienta de migración. Antes de implementar la tabla debe elegirse una opción compatible con Go y el flujo de despliegue (por ejemplo, migraciones SQL versionadas ejecutadas por una herramienta de CI/despliegue).
- **UUID:** no está definida la biblioteca o el generador de UUID. Debe seleccionarse una dependencia mínima o una alternativa estándar aprobada antes de implementar la generación de `id`.
- **Router HTTP:** `cmd/api/main.go` es esquelético y no fija router. Se puede comenzar con `net/http` o adoptar un router; la elección debe preservar la ruta y el contrato especificados.
- **Prueba de PostgreSQL:** la disponibilidad de Docker/testcontainers o de una base de prueba no está confirmada. La integración con PostgreSQL queda condicionada a esa capacidad de entorno, aunque las pruebas unitarias no dependen de ella.
