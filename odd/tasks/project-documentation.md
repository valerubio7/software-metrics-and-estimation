# Documentación inicial del proyecto

## Objetivo y alcance

Documentar la arquitectura y las seis US implementadas, el equipo confirmado, la planificación observable del Sprint 1 y el uso de IA respaldado por evidencia. Conservar la consigna original con un nombre claro y actualizar únicamente la descripción del estado del proyecto en OpenSpec.

- Rama: `docs/project-documentation`; base: `56f5b49f78e0231e69c34bbe5efc97079d48baf8`.
- Idioma: español, según la convención del repositorio.
- Fuera de alcance: README raíz, manual, guía/informe de pruebas, reviews, retrospectivas, código, migraciones, cambios en GitHub y configuración de metodología.
- Autorización posterior: el usuario eligió commit, push y PR hacia main, con issue aprobada y excepción explícita de tamaño (`size:exception`). No autoriza merge ni eliminación de rama.
- TDD configurado: activo en `openspec/config.yaml` (`sdd.strict_tdd: true`); runner: `go test ./...`. Este trabajo solo modifica documentación y notas descriptivas; no hay cambio de comportamiento que requiera un ciclo RED/GREEN. No afirmar una nueva ejecución de pruebas funcionales.
- RDD: desactivado en la sesión; sin revisión nativa.
- Estrategia de entrega: `ask-on-risk`; se estiman 250–350 líneas nuevas de documentación/configuración, además del registro de tareas. La consigna contiene 400 líneas preexistentes que se conservarán byte a byte; no constituyen redacción nueva, pero cuentan en una futura primera incorporación a Git. Evaluar el tamaño antes de autorizar una entrega.

## Hechos y restricciones

- Equipo: Valentín Rubio (`valerubio7`), Pablo Geyer (`PabloGeyer`), Luciano (`Lucho-cas`), Santiago Calzolari (`SantyCalz`) y Santiago Oses (`SantiagoMO3`). Valentín confirmó ser Agile Enabler; no inferir responsabilidades exclusivas ni ceremonias.
- La consigna asigna Product Architect a profesores; nombres y asignatura/actores exactos pendientes de confirmación si no figuran en la fuente.
- Sprint 1 observado: US-01 a US-11; seis Done (01,02,05,06,07,08), 09 In progress, 03 Ready, 04/10/11 Backlog. Objetivo concreto, capacidad y estimaciones no confirmados: usar secciones pendientes/propuestas, no acuerdos ficticios.
- La API es backend Go/PostgreSQL sin interfaz, métricas ni reportes implementados.
- Hay dos migraciones con número `000003` en el directorio compartido. Documentar el riesgo sin corregir código, SQL ni prometer un despliegue exitoso.
- `docs/PROJECT.md` era un archivo del usuario no versionado. Renombrar a `docs/assignment.md` sin modificar contenido. SHA-256 original: `f8d6cab13e6f4a597e372dce43de507b71a62919213c396711890cb3d5bc2159`.
- Uso de IA: preparación/documentación con Pi/el Gentleman; PR #66 menciona Claude Code. Distinguir evidencia de uso de validación humana aún no confirmada.

## Tareas

- [x] DOC-01 — Reconciliar alcance y evidencia, confirmar integrantes y crear registro/rama.
  - Ruta: exploración delegada (`muo6qlwr-6-8z2y`) y coordinación inline.
  - Evidencia: fuente leída, perfil GitHub de SantiagoMO3 confirmado, estado Git y hash original observados.
  - Commit: pendiente de autorización; no realizado.
- [x] DOC-02 — Crear índice, arquitectura, trazabilidad, equipo, plan de Sprint 1 y uso de IA; renombrar consigna y actualizar estado descriptivo de OpenSpec.
  - Ruta: un escritor delegado; dispara regla de múltiples archivos no triviales.
  - Superficies: `docs/README.md`, `docs/architecture/overview.md`, `docs/traceability.md`, `docs/scrum/team.md`, `docs/scrum/sprint-1.md`, `docs/ai-usage.md`, `docs/PROJECT.md`, `docs/assignment.md`, `openspec/config.yaml`.
  - Aceptación: seis US con enlaces a historia/spec/escenario/test/código; hechos separados de propuestas y pendientes; configuración metodológica intacta; consigna con hash idéntico; ningún documento postergado creado.
  - Estado: implementado por `muo727p0-7-p3vv`; 275 líneas de documentación nueva y 7 líneas descriptivas de configuración. `git diff --check` y SHA-256 aprobados por escritor y spot-check del padre. Modelo observado: `gpt-6.1-sol`; esfuerzo no informado.
  - Commit: pendiente de autorización; no realizado.
- [x] DOC-03 — Verificar contenido, referencias, preservación y alcance; corregir defectos documentales si aparecen.
  - Ruta: verificador delegado independiente y lectura estructural del padre.
  - Checks: enlaces locales, símbolos de tests/código, consistencia de campos del YAML, hash de consigna, ausencia de cambios fuera del alcance, whitespace y estado Git.
  - No ejecutar suite Go, instalar herramientas o generar cobertura por cambios exclusivamente documentales.
  - Evidencia: verificador `muo7fib8-8-02do` sin defectos concretos; 55 enlaces locales, 12 anchors, 12 funciones de tests y subcaso de US-07 comprobados. Hash de consigna idéntico; configuración fuera de stage/notes preservada. Estado Git sin cambios durante verificación.
  - Límites: no consulta nueva a GitHub ni suite Go/cobertura/despliegue; `git diff --check` no incluye documentos sin seguimiento. Código, SQL y README raíz intactos.
  - Commit: pendiente de autorización; no realizado.

- [x] DOC-04 — Guardar, publicar y abrir un PR documental único.
  - Ruta: operaciones Git/GitHub inline; verificación independiente del candidato publicado si corresponde.
  - Issue aprobada: [#67](https://github.com/valerubio7/software-metrics-and-estimation/issues/67); creación y cuerpo confirmados por lectura posterior.
  - Estrategia elegida por el usuario: `exception-ok`, PR único a main con `type:docs` y `size:exception`; la consigna original de 400 líneas explica gran parte del tamaño.
  - Aceptación: cambios exactos guardados, rama publicada, PR enlazado a #67; no merge ni borrado de rama.
  - Estado: publicado. Commits `1d92cb6` y `d5cc1c2`; este último preserva el hash original de la consigna tras quitar un LF final añadido. Verificador `muo8p2u8-a-qg8t`: PASS, bytes de archivo y commit idénticos, worktree limpio, 9 archivos +740/−3 y diff sin findings.
  - PR: [#68](https://github.com/valerubio7/software-metrics-and-estimation/pull/68), abierto hacia main, cuerpo/issue/labels confirmados. No aparecen checks automatizados en `statusCheckRollup`; no se afirma CI aprobada.
  - Rama publicada; este registro se guarda en un commit de cierre documental. PR pendiente de revisión; no merge ni eliminación de rama.

## Verificación y siguiente paso

- Preflight observado: `main` en la base indicada; únicamente `docs/` sin seguimiento. Rama documental creada sin alterar cambios del usuario.
- Verificación funcional previa, no repetida para este trabajo: `go test ./...` y `go vet ./...` pasaron en `56f5b49`; integración historias/sprints sin caché, proyectos con caché.
- Evaluación nativa read-only: riesgo no evaluable por archivos sin seguimiento; RDD off y plan devuelto exige verificador independiente. No se inició revisión nativa ni se modificó autoridad.
- DOC-03 completado sin defectos concretos; 275 líneas en los seis documentos nuevos y consigna de 400 líneas preservada. Los documentos siguen sin seguimiento de Git.
- Siguiente paso: revisar PR #68 y autorizar su integración por separado. Tras integrar puede eliminarse la rama. Confirmación de Sprint Goal/capacidad y documentación postergada siguen pendientes.
