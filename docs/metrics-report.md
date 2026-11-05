# Informe de métricas y cobertura de pruebas

* **Fecha de corte:** 05/11/2026
* **Commit:** `0bf82dd`
* **Alcance:** todo el módulo Go (`internal/...` y `cmd/...`)
* **Entorno:** Go 1.27, PostgreSQL 16 en `127.0.0.1:5433`, esquema `14`

Este informe cubre dos dimensiones distintas:

1. **Calidad del software:** cobertura de pruebas y composición de la suite.
2. **Métricas del producto:** los indicadores que calcula la propia aplicación
   (velocidad, esfuerzo, desvío y defectos) sobre una base de demostración.

Los valores son reproducibles con los comandos de la sección 2.

---

## 1. Resumen ejecutivo

| Indicador | Valor |
| --- | --- |
| Cobertura total de sentencias | **80.9%** (5092 / 6293) |
| Cobertura de dominio y aplicación | 97.4% y 97.2% |
| Funciones de prueba ejecutadas | **709** (576 unitarias + 133 de integración) |
| Pruebas fallidas / salteadas | 0 / 0 |
| Archivos de prueba | 121 |
| Líneas de código de producción | 15.672 |
| Líneas de prueba | 27.769 (1.77× el código de producción) |
| Commits | 348 en 45 días (22/09/2026 – 05/11/2026) |
| PRs mergeadas | 75 |
| Contribuyentes | 5 |

La relación prueba/producción de 1.77× y el 97% de cobertura en las capas de
dominio y aplicación son los dos indicadores más sólidos del proyecto. La capa
más débil es la TUI (74.6%), esperable por su naturaleza de interfaz.

---

## 2. Método y reproducción

### 2.1 Cobertura

```sh
# Suite completa (requiere PostgreSQL local, ver 2.3)
PROJECT_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
STORY_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
SPRINT_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
PROJECTMEMBER_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
TASK_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
DEFECT_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
EFFORT_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
METRICS_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
REPORT_TEST_DATABASE_URL='postgres://sme:sme@127.0.0.1:5433/sme?sslmode=disable' \
go test ./tests/... -count=1 \
  -coverpkg=./internal/...,./cmd/... \
  -coverprofile=coverage.out

go tool cover -func=coverage.out | tail -1
go tool cover -html=coverage.out -o coverage.html
```

### 2.2 Por qué `-coverpkg` es obligatorio

Las pruebas viven en paquetes propios (`tests/unit/...`), no en el paquete que
ejercitan. Sin `-coverpkg`, `go test` mide el paquete de prueba y reporta
`coverage: [no statements]` en cada paquete de la suite. El flag instrumenta
`internal/...` y `cmd/...` y hace que la cobertura refleje el código real.

Un segundo detalle: al ejecutar varios paquetes de prueba, el perfil resultante
contiene un bloque por binario de prueba. La deduplicación por ubicación
(tomando el máximo) es lo que produce el 80.9%; sumar las líneas sin deduplicar
da un 41.9% incorrecto. `go tool cover` ya hace esa deduplicación.

### 2.3 Integración

Cada prueba de integración crea un esquema PostgreSQL aislado
(`tests/integration/testpostgres`), ejecuta las migraciones y lo elimina con
`DROP SCHEMA ... CASCADE` al terminar. Si no se define la variable
`*_TEST_DATABASE_URL`, el helper levanta un contenedor con testcontainers y, si
Docker no está disponible, **saltea** la prueba en lugar de fallar. Para que el
informe refleje ejecución real y no skips, los valores de este documento se
obtuvieron con las nueve variables apuntando a PostgreSQL local.

---

## 3. Cobertura por capa

| Capa | Cobertura | Sentencias |
| --- | ---: | ---: |
| Dominio | 97.4% | 340 / 349 |
| Aplicación (casos de uso) | 97.2% | 379 / 390 |
| Transporte HTTP | 94.6% | 666 / 704 |
| Infraestructura PostgreSQL | 80.1% | 883 / 1102 |
| TUI (Bubble Tea) | 74.6% | 2707 / 3631 |
| Wiring de la API | 100.0% | 117 / 117 |
| **Total** | **80.9%** | **5092 / 6293** |

La arquitectura hexagonal se refleja en los números: el núcleo (dominio y
aplicación) está casi completamente cubierto, mientras que la infraestructura
baja al 80.1% porque parte de su código son adaptadores y manejo de errores de
driver que solo se ejercitan en integración.

---

## 4. Cobertura por módulo

| Módulo | Cobertura | Sentencias |
| --- | ---: | ---: |
| `internal/api` | 100.0% | 117 / 117 |
| `internal/effort` | 98.8% | 170 / 172 |
| `internal/report` | 93.9% | 520 / 554 |
| `internal/defect` | 91.5% | 258 / 282 |
| `internal/metrics` | 91.3% | 264 / 289 |
| `internal/project` | 86.7% | 170 / 196 |
| `internal/story` | 86.5% | 418 / 483 |
| `internal/sprint` | 84.4% | 178 / 211 |
| `internal/task` | 82.9% | 180 / 217 |
| `internal/projectmember` | 78.0% | 110 / 141 |
| `internal/tui` | 74.6% | 2707 / 3631 |

`projectmember` es el módulo con menor cobertura relativa (78.0%). Es el
candidato natural para la próxima iteración de pruebas.

---

## 5. Suite de pruebas

| Tipo | Funciones | Archivos | Requiere PostgreSQL |
| --- | ---: | ---: | --- |
| Unitarias | 576 | 99 | No (usan dobles de prueba) |
| Integración | 133 | 22 | Sí |
| **Total** | **709** | **121** | — |

Las pruebas de integración cubren los repositorios PostgreSQL, las migraciones
y los flujos HTTP completos. Las unitarias cubren dominio, casos de uso y
handlers con dobles.

Comparación de cobertura:

| Alcance | Cobertura |
| --- | ---: |
| Solo unitarias | 80.5% |
| Unitarias + integración | 80.9% |

La diferencia de 0.4 puntos porcentuales confirma que las unitarias ya cubren
casi todo el núcleo; la integración aporta principalmente confianza sobre los
adaptadores, no cobertura nueva.

---

## 6. Métricas del producto (datos de demostración)

Consultadas contra la API en ejecución (`GET /projects/{id}/metrics`) sobre la
base sembrada de demostración. **No son métricas del proceso del equipo**: son
la salida del motor de métricas del producto.

| Proyecto | Velocidad | SP completados | Historias completadas | Horas reales | Defectos detectados | Defectos resueltos |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| App de Turnos | 8.67 | — | 52.9% | 160.5 | 13 | 5 |
| Migración Core | 13.75 | 55 | 81.3% | 197.0 | 13 | 2 |
| Panel Logístico | — | 0 | 0.0% | — | 0 | 0 |
| Portal Alfa | 10.50 | 50 | 55.6% | 145.5 | 10 | 4 |
| Portal de Proveedores | 16.00 | 37 | 53.3% | 125.0 | 8 | 3 |
| **Total** | — | **142** | — | **627.5** | **44** | **14** |

Lectura:

* **14 de 44 defectos resueltos (31.8%)** en el conjunto de datos. Es el
  indicador que más conviene vigilar en un proyecto real.
* **Migración Core** muestra el mejor cierre de historias (81.3%) y la mayor
  velocidad (13.75).
* **Panel Logístico** no tiene Sprints cerrados ni esfuerzo registrado: la API
  devuelve `null` en velocidad y horas, en lugar de inventar un cero.

### 6.1 Calidad de los datos

Varios agregados de puntos y horas son `null` porque faltan observaciones en la
base de demostración. La API no las rellena: las expone en el bloque `coverage`
de cada respuesta.

| Proyecto | SP planificados faltantes | Estimaciones faltantes | Historias sin esfuerzo |
| --- | ---: | ---: | ---: |
| App de Turnos | 6 | 2 | 4 |
| Migración Core | 1 | 1 | 3 |
| Panel Logístico | 5 | 3 | 10 |
| Portal Alfa | 5 | 1 | 4 |
| Portal de Proveedores | 4 | 1 | 3 |

Esta es una decisión de diseño deliberada: un promedio calculado sobre datos
incompletos es peor que un `null` explícito con su contador de faltantes.

---

## 7. Métricas del proceso

| Indicador | Valor |
| --- | ---: |
| Commits totales | 348 |
| Commits de merge | 65 |
| PRs mergeadas | 75 |
| Período | 45 días (22/09 – 05/11/2026) |
| Promedio de commits por día | 7.7 |
| PRs mergeadas en septiembre | 18 |
| PRs mergeadas en octubre | 44 |
| PRs mergeadas en noviembre (hasta el 05) | 13 |

Contribuciones por autor:

| Autor | Commits |
| --- | ---: |
| Valentín Rubio | 131 |
| Lucho C (+ `Lucho-cas`) | 104 |
| Pablo Geyer | 59 |
| SantyCalz (+ `Santiago Calzolari`) | 37 |
| SantiagoMO3 (+ `Santiago Oses`) | 15 |
| `copilot-swe-agent[bot]` | 2 |

Los conteos consolidan identidades duplicadas de Git. Quedan pendientes de
unificar con un archivo `.mailmap` para que `git shortlog` reporte cinco
contribuyentes sin fragmentación.

---

## 8. Limitaciones

1. **La cobertura de sentencias no mide comportamiento.** Un 80.9% no garantiza
   que los escenarios BDD estén todos ejercitados; la trazabilidad
   escenario → prueba todavía no está automatizada.
2. **La TUI es la capa menos cubierta (74.6%).** Los flujos de navegación
   tienen pruebas, pero los estados visuales y el renderizado no se verifican
   por completo.
3. **Las métricas de la sección 6 usan datos sembrados.** Sirven para validar el
   motor de métricas, no para evaluar el desempeño del equipo.
4. **La integración depende del entorno.** Sin PostgreSQL local ni Docker, las
   133 pruebas se saltean silenciosamente y la cobertura reportada sería solo la
   unitaria (80.5%).

---

## 9. Conclusiones

* El proyecto cumple el umbral habitual de 80% de cobertura, con el núcleo de
  negocio por encima del 97%.
* La inversión en pruebas es alta y sostenida: 1.77 líneas de prueba por línea
  de producción.
* El motor de métricas del producto está operativo y expone explícitamente la
  calidad de sus datos en lugar de ocultarla.
* Las brechas a cerrar son `projectmember` (78.0%), la TUI (74.6%) y la
  trazabilidad automática entre escenarios BDD y pruebas.
