# Sprint 1 — Retrospectiva

* **Fecha:** 05/10/2026, 45 min aprox.
* **Participantes:** Valentín Rubio, Pablo Geyer, Luciano, Santiago Calzolari y Santiago Oses.
* **Formato:** qué mantener / qué mejorar, cierre con acciones.

## Qué mantener

* Trabajo por historias con PR separada (ej: PR #64 US-07, PR #65 US-08). Cada cambio se puede revisar sin mezclar objetivos.
* Pruebas en `tests/unit` y `tests/integration` para proyecto, historia y Sprint. Están en el repo y corren por comportamiento.
* Tablero en GitHub Projects con US-01 a US-11 cerradas al 04/10/2026 con la integración de la PR #81.

## Qué mejorar

* El merge `08e324b` dejó roto el wiring US-07/US-08 y hubo que corregirlo en `eb3a843` (PR #66, 29/09/2026). Terminar una historia no alcanzó para garantizar el recorrido completo.
* US-09 necesitó consolidación posterior (PR #78, 02/10/2026). Faltó acordar antes el esquema canónico y el orden de integración.
* No quedaron notas originales de las ceremonias. Sin acta con fecha y decisiones, no se puede distinguir lo acordado de lo reconstruido después.

## Acciones para Sprint 2

| Acción | Evidencia de cumplimiento | Responsable | Plazo |
|---|---|---|---|
| Anotar orden de integración y dependencias en la issue antes de mergear PRs relacionadas | Issue con lista de dependencias y PR de consolidación linkeada | Por definir en planning | Planning Sprint 2 |
| Probar el recorrido proyecto → backlog → Sprint además de los tests por historia | Registro de alcance probado y resultado | Por definir en planning | Fin Sprint 2 |
| Dejar acta breve de cada ceremonia con fecha, participantes y decisiones | Archivo en `docs/scrum/sprints/` por ceremonia | Por definir en planning | Al cerrar cada ceremonia |
