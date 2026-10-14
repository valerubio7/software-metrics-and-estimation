# Historial de Sprints cerrados (US-13)

La API permite consultar los Sprints cerrados de un proyecto, con las historias planificadas en cada uno:

```http
GET /projects/{project_id}/sprints
```

`HEAD` se acepta con el mismo estado y sin cuerpo. La consulta es de solo lectura: no modifica Sprints, historias, asociaciones ni tareas.

## Respuesta

Si el proyecto existe, responde `200` con un contenedor que identifica al proyecto y la lista de Sprints cerrados:

```json
{
  "project_id": "5c21cbd4-d9a7-42df-9c3a-c0866f058746",
  "sprints": [
    {
      "id": "e99c05a4-03ea-4c19-8f59-286dd59e7aa1",
      "project_id": "5c21cbd4-d9a7-42df-9c3a-c0866f058746",
      "sprint_goal": "Entregar el flujo de métricas",
      "is_closed": true,
      "stories": [
        {
          "id": "5a3e0000-0000-4000-8000-000000000001",
          "title": "Registrar esfuerzo",
          "priority": "alta",
          "story_points": null,
          "completed_at": "2026-10-11T10:30:00Z",
          "completed": true
        },
        {
          "id": "5a3e0000-0000-4000-8000-000000000002",
          "title": "Listar defectos",
          "priority": "baja",
          "story_points": null,
          "completed_at": null,
          "completed": false
        }
      ]
    }
  ]
}
```

- Solo se incluyen Sprints cerrados del proyecto indicado; un Sprint abierto no aparece.
- `completed` es `true` cuando la historia se completó **en ese Sprint** (`completed_at` presente). Una historia que pasa por varios Sprints puede figurar completada en uno y no completada en otro.
- `completed_at` se devuelve en UTC (`Z`); es `null` si la historia no se completó en el Sprint.
- `story_points` se devuelve siempre y vale `null` por ahora: la API todavía no permite estimar historias.
- No se exponen `seq`, `description`, `acceptance_criteria`, `estimated_hours` ni `status`.

## Orden

- Los Sprints se ordenan por creación descendente: primero el creado más recientemente, sin importar cuándo se cerró.
- Las historias de cada Sprint se ordenan por creación ascendente, sin importar el orden en que se asignaron.
- Consultas repetidas sobre los mismos datos devuelven el mismo cuerpo.

## Listas vacías

Un proyecto sin Sprints cerrados responde `200` con `{"project_id":"<uuid>","sprints":[]}`. Un Sprint cerrado sin historias se devuelve con `"stories":[]`. Ninguna lista se devuelve como `null`.

## Errores

| Estado | Código | Cuándo |
|--------|--------|--------|
| `404` | `project_not_found` | El proyecto no existe. El cuerpo no incluye la clave `sprints`. |
| `422` | `validation_failed` | `project_id` no es un UUID; `fields.project_id` indica el problema y no se consulta la base. |
| `405` | `method_not_allowed` | Método distinto de `GET` y `HEAD` invocado sobre el handler; el router responde `405` en texto plano para `PUT` y `DELETE` sobre la colección. |
| `500` | `internal_error` | Fallo inesperado, con el mensaje genérico `an unexpected error occurred` y sin detalles internos. |

## Disponibilidad según el esquema

La ruta está disponible únicamente con el esquema PostgreSQL limpio en versión 11 o superior, que agrega `sprints.seq` con la migración `000011` y la restricción `UNIQUE (project_id, seq)`. La compuerta es independiente de la del cierre de Sprints (versión 8).

| Estado del esquema | `GET /projects/{project_id}/sprints` |
|--------------------|--------------------------------------|
| Limpio, versión 11 o superior | `200`, `404`, `422` o `500` según lo anterior |
| Limpio, versiones 5 a 10 | `405` (existe la creación con `POST` sobre la misma ruta) |
| Versión anterior a 5, esquema sucio o error al leer `schema_migrations` | `404` (no hay ninguna ruta de Sprint registrada) |

## Limitaciones

- **Sin paginación:** la respuesta incluye todos los Sprints cerrados del proyecto y todas sus historias. Es un riesgo aceptado mientras los volúmenes sean pequeños; un contenedor con `project_id` permite agregar metadatos de paginación sin romper el contrato.
- **Sprints anteriores a la migración:** `seq` se asigna a los Sprints que ya existían al aplicar `000011` en un orden que PostgreSQL no garantiza. Los Sprints creados después siguen el orden real de creación; el orden relativo de los previos puede no coincidir con el orden histórico.
- **Migraciones externas:** las migraciones se aplican fuera de la aplicación; esta consulta no agrega ninguna acción de migración al arranque.
