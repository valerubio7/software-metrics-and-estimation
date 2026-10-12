# Cierre de Sprint (US-12)

La API permite cerrar un Sprint abierto que pertenece al proyecto indicado:

```http
POST /projects/{project_id}/sprints/{sprint_id}/closure
```

No hace falta enviar un cuerpo. Si el cierre se completa, responde `200`:

```json
{"project_id":"<uuid>","sprint_id":"<uuid>","is_closed":true}
```

El cierre es definitivo desde esta API. Si el Sprint ya está cerrado, responde `409` con `sprint_closed`. Un proyecto inexistente responde `404 project_not_found`; un Sprint inexistente o ajeno al proyecto responde `404 sprint_not_found`. Los identificadores inválidos responden `422 validation_failed`; los fallos inesperados responden `500 internal_error` sin detalles internos.

El sistema conserva las historias, finalizaciones, asociaciones, tareas, esfuerzos y defectos relacionados. El cierre no cambia los datos de las historias. Las asignaciones concurrentes se serializan con el cierre; si el cierre confirma primero, la asignación se rechaza íntegramente.

La ruta está disponible únicamente con el esquema PostgreSQL limpio en versión 8 o superior, que incluye `sprints.is_closed` de la migración `000008`. La aplicación no agrega una migración para esta operación.
