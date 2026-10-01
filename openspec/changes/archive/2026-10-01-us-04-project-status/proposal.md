# Proposal: US-04 — Consultar el estado de un proyecto

## Intent

Permitir que un cliente consulte un proyecto existente y obtenga su estado actual derivado de sus fechas planificadas, sin persistir un ciclo de vida ni modificar el proyecto. Esto responde a la necesidad de distinguir proyectos aún no iniciados, dentro de su ventana planificada y cuyo plazo ya pasó.

## Alcance

- Exponer una operación de lectura de un proyecto identificado, con la ruta/método HTTP definitivo pendiente de especificación.
- La respuesta identifica claramente el proyecto; incluirá ID y nombre, además del estado derivado.
- Calcular el estado usando fechas date-only y la fecha actual:
  - `planned` si `current_date < start_date`.
  - `active` si `start_date <= current_date <= planned_finish_date`.
  - `overdue` si `current_date > planned_finish_date`.
- Reportar claramente que el proyecto no existe.
- Mantener el cálculo sin efectos secundarios y sin almacenamiento de estado.

No hay un estado `completed`: el modelo no contiene fecha real de finalización, y una fecha planificada vencida no prueba que el trabajo haya terminado.

## Fuera de alcance

- US-03 y cualquier dato o regla de miembros.
- Estados derivados de historias, sprints, progreso, aprobaciones o intervención manual.
- Añadir un campo de estado persistido, migración de ciclo de vida o endpoint de escritura.
- Incorporar un estado `completed` sin una fuente de finalización real.
- Diseñar políticas de zona horaria sin requisito respaldado.

## Áreas afectadas

- `internal/project`: cálculo de dominio/aplicación y contrato de lectura.
- Transporte HTTP y composición de rutas para exponer la consulta.
- Adaptador/repositorio PostgreSQL para recuperar un proyecto por ID (sin cambio de esquema esperado).
- Pruebas unitarias, HTTP e integración para límites de fechas, identidad y ausencia.
- `openspec/specs/project/spec.md` y documentación de API, según los patrones ya existentes.

Las rutas y archivos concretos se decidirán en diseño; no se incluye implementación en esta propuesta.

## Riesgos y mitigaciones

- **Límites inclusivos y errores de un día:** modelar y probar explícitamente el día de inicio y el día de fin como `active`.
- **Fecha actual no determinista:** aislar la lectura de fecha actual para permitir pruebas reproducibles. El significado de su fuente concreta queda abierto; no imponer zona horaria sin fundamento.
- **Confundir vencido con completado:** mantener `overdue` como único resultado posterior a planned finish y no inferir finalización.
- **Proyecto inexistente:** especificar una respuesta de error inequívoca y probarla.
- **Ampliación accidental:** excluir expresamente US-03, miembros, historias y sprints.
- **Cambios excesivos:** 400 líneas es el presupuesto de referencia; la estrategia seleccionada es un solo PR y el usuario autorizó explícitamente `size:exception` si fuera necesario. Mantener la solución acotada y reportar cualquier riesgo real de tamaño.

## Rollback

La operación es aditiva, de solo lectura y sin migración ni cambio de datos. Se puede retirar la ruta y su cálculo/adaptador junto con las pruebas/documentación de US-04. No requiere restauración de datos ni reversión de escrituras. Si se actualiza la spec compartida, revertir únicamente el bloque de requisito de consulta para conservar los requisitos previos de proyecto.

## Criterios de éxito medibles

1. Una consulta de proyecto existente devuelve ID, nombre y exactamente uno de `planned`, `active` o `overdue`.
2. Pruebas verifican al menos: víspera de inicio = `planned`; día de inicio = `active`; día de fin planificado = `active`; día posterior = `overdue`.
3. Una consulta de ID inexistente produce una respuesta de ausencia explícita y verificable, no un estado inventado ni un éxito vacío.
4. La consulta no realiza escrituras: no se agrega estado almacenado, columna/migración ni mutación del proyecto.
5. No existe estado `completed` en esta capacidad y no se requiere ni invoca US-03.
6. La consulta queda documentada como lectura identificada de proyecto y la suite pertinente valida contrato y reglas de límites.

## Supuesto abierto

La regla de negocio establece que se usa la fecha actual en formato de calendario, pero no especifica la fuente/configuración que define esa fecha para el servicio. Diseño debe elegir un mecanismo coherente con el entorno y testeable, sin elevar una zona horaria concreta a requisito hasta que exista evidencia.

## Preguntas para la siguiente fase

La decisión de producto y las reglas de derivación están confirmadas por el usuario. La siguiente fase debe concretar la ruta HTTP, el contrato/código de error para ID inexistente y el mecanismo de obtención de fecha actual, conservando los límites de alcance anteriores.
