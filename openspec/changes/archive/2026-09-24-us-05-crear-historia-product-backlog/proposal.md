# Propuesta: crear una historia en el Product Backlog (US-05)

## Intención y resultado

Permitir crear una historia asociada al Product Backlog de un proyecto existente a partir de su identificador conocido. Esto cubre la necesidad de registrar trabajo y sus criterios de aceptación antes de estimarlo, sin obligar a inventar Story Points durante la creación. Referencia: [issue #33](https://github.com/valerubio7/software-metrics-and-estimation/issues/33).

## Alcance y reglas de producto confirmadas

- Recibir el identificador de un proyecto ya conocido y comprobar que existe; no incorporar listado de proyectos ni interfaz para elegirlos.
- Exigir título, descripción, prioridad y criterios de aceptación. La prioridad la elige quien crea la historia entre **alta**, **media** y **baja**.
- Exigir una lista de criterios de aceptación no vacía y rechazar cualquier entrada en blanco.
- Generar el identificador de la historia; asignar **pendiente** como estado inicial y **null** como Story Points iniciales, nunca cero. En este alcance no se recibe ni acuerda estimación inicial: queda para US-14.
- Persistir la historia únicamente si sus datos son válidos y el proyecto existe; rechazar proyectos inexistentes sin dejar historias huérfanas, incluso ante fallos o cambios concurrentes que impidan asegurar la asociación.

### Fuera de alcance

Editar historias (US-06), consultar el Product Backlog (US-07), asignar historias a un Sprint (US-09) y estimarlas mediante Planning Poker (US-14). Estas funciones no son condiciones previas para registrar una historia.

## Áreas afectadas

| Área | Cambio esperado |
| --- | --- |
| API HTTP | Incorporar una operación de creación de historias vinculadas a un proyecto identificado; exponer el resultado y los rechazos sin revelar fallos internos. |
| Dominio y aplicación | Representar historia y criterios, validar los datos obligatorios y aplicar identificador, estado y Story Points iniciales. |
| Persistencia PostgreSQL | Almacenar historias y su asociación con proyectos, verificar existencia y asegurar integridad referencial; incorporar la migración correspondiente. |
| Pruebas y documentación | Cubrir creación, validaciones, ausencia de proyecto y asociación persistida; documentar cómo crear historias y aplicar migraciones. |

La exploración registra que actualmente solo existe `POST /projects`, sin módulo de historias ni lectura de proyectos. Esta propuesta fija el comportamiento de producto, **no** la ruta HTTP, el formato exacto de solicitud/respuesta, los códigos de error para proyecto inexistente, los límites de longitud, la estructura SQL ni la estrategia concreta de concurrencia: se resolverán en specs/diseño sin alterar las reglas confirmadas.

## Riesgos y mitigación

- **Integridad:** verificar existencia solo antes de insertar puede dejar una historia huérfana si el proyecto cambia concurrentemente; el diseño deberá preservar la asociación también en la base de datos y probar el rechazo sin persistencia.
- **Estimación prematura:** aceptar Story Points o confundir `null` con `0` cambiaría el significado de «pendiente de estimación»; se verificará explícitamente el estado inicial y el límite con US-14.
- **Alcance y revisión:** endpoint, dominio, persistencia, migración y pruebas pueden superar el presupuesto de revisión de **400 líneas**. Si el riesgo se materializa, la estrategia `ask-on-risk` requiere consultar antes de elegir una excepción de tamaño o encadenar entregas; esta propuesta no elige ninguna.
- **Ejecución posterior:** la implementación futura seguirá TDD estricto con evidencia RED/GREEN/REFACTOR en commits cronológicos autorizados por el usuario y validación con `go test ./...`; una prueba de integración omitida por falta de Docker no acredita integridad PostgreSQL. No se implementa ni se crean commits en esta fase.

## Reversión

Si la funcionalidad resultara incorrecta, retirar la operación de creación y revertir la migración de historias conforme al procedimiento de despliegue, evaluando primero si ya hay historias que preservar. No eliminar datos existentes silenciosamente ni modificar proyectos de US-01.

## Criterios de éxito

1. Una solicitud válida para un proyecto existente crea una sola historia asociada a ese proyecto, con identificador generado, prioridad elegida, estado **pendiente**, Story Points **null** y criterios de aceptación conservados.
2. La ausencia de título, descripción, prioridad o criterios, una prioridad fuera de los tres valores permitidos, una lista vacía o un criterio en blanco impiden la creación.
3. Un identificador de proyecto inexistente se rechaza y no queda registrada ninguna historia huérfana.
4. La creación no exige ni establece estimación, y no incorpora edición, consulta del backlog, asignación a Sprint ni Planning Poker.
