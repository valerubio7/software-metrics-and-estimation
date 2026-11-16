# Medición de rendimiento mínima (smoke test)

## Qué es esto y qué no es

Esta es una **medición puntual de referencia**, ejecutada una sola vez en una
sola máquina de desarrollo, con un volumen de datos chico y sin coordinación
previa con el equipo sobre umbrales de aceptación. **No es una prueba de
carga ni de estrés**, no valida comportamiento bajo concurrencia sostenida ni
bajo datos de producción, y no se declara ningún resultado como "cumple" o
"no cumple" porque no hay un objetivo acordado con el equipo para comparar.
Sirve únicamente como punto de referencia inicial, reproducible con los
comandos documentados abajo.

## Fecha, commit y entorno

- **Fecha de ejecución:** 2026-11-16
- **Commit medido:** `10199f67bb01a1426905e4a8b0a60262fa031c59` (`main`, HEAD al momento de medir)
- **Sistema operativo:** Ubuntu 26.04.1 LTS (kernel `7.0.0-43-generic`, x86_64)
- **CPU:** AMD Ryzen AI 7 350 w/ Radeon 860M (16 hilos lógicos reportados por `nproc`)
- **RAM:** 22 GiB totales (según `free -h` al momento de medir)
- **Entorno de ejecución de la API y PostgreSQL:** en contenedores, levantados con `docker compose up -d` (servicios `api` y `postgres`, imagen `postgres:16`)
- **Cliente de carga (`hey`):** corrido en el host, fuera de los contenedores, contra `http://localhost:8080`

## Carga de datos de prueba

Se levantó la aplicación y se cargaron datos sintéticos vía la propia API HTTP (no inserción directa en la base):

- 1 proyecto (`POST /projects`)
- 1 Sprint (`POST /projects/{project_id}/sprints`)
- 100 historias de usuario (`POST /projects/{project_id}/stories`), todas asignadas a ese Sprint (`POST /projects/{project_id}/sprints/{sprint_id}/stories`)
- 10 defectos (`POST /projects/{project_id}/defects`), todos asociados a la primera historia y a ese Sprint

Este es un volumen chico, pensado para que las rutas de listado y de
agregación tengan datos no triviales sobre los que operar, no para simular
una base de producción.

## Endpoints elegidos y por qué

Se eligieron tres endpoints `GET` representativos, sin efectos secundarios, que cubren perfiles de costo distintos:

1. **`GET /projects/{project_id}/stories`** — listado del backlog del proyecto. Representa una lectura de colección de tamaño moderado (100 filas en esta medición), el tipo de consulta más frecuente desde la TUI al navegar un proyecto.
2. **`GET /projects/{project_id}/metrics`** — cálculo de métricas agregadas del proyecto (story points, horas estimadas/reales, desviación, defectos detectados/resueltos, cobertura). Es, de los tres, el que combina más tablas y más cómputo por solicitud, por lo que es el más representativo para detectar una ruta costosa si la hubiera.
3. **`GET /projects/{project_id}/defects`** — listado simple de defectos (10 filas en esta medición). Se usa como línea de base liviana para comparar contra los otros dos.

## Comandos ejecutados

Arranque de la aplicación:

```sh
docker compose up -d
```

Herramienta de carga: `hey` (`github.com/rakyll/hey`), instalada con `go install github.com/rakyll/hey@latest` porque no estaba disponible en el sistema (tampoco `ab`).

Carga corta por endpoint: 200 requests, concurrencia 10.

```sh
PID=557035e1-c1db-4567-a3d3-f7ec2dbcee7f   # project_id del proyecto de prueba creado para esta medición

hey -n 200 -c 10 "http://localhost:8080/projects/$PID/stories"
hey -n 200 -c 10 "http://localhost:8080/projects/$PID/metrics"
hey -n 200 -c 10 "http://localhost:8080/projects/$PID/defects"
```

## Resultados reales obtenidos

| Endpoint | Requests/seg | Latencia p50 | Latencia p95 | Latencia p99 | Errores |
|---|---|---|---|---|---|
| `GET /stories` (100 historias) | 6184.68 | 1.2 ms | 3.1 ms | 9.1 ms | 0 (200/200 con status 200) |
| `GET /metrics` | 7596.89 | 0.9 ms | 4.3 ms | 6.1 ms | 0 (200/200 con status 200) |
| `GET /defects` (10 defectos) | 19478.78 | 0.4 ms | 1.5 ms | 1.9 ms | 0 (200/200 con status 200) |

Salida completa de `hey` por endpoint (tal como se obtuvo, sin editar):

### `GET /projects/{project_id}/stories`

```
Summary:
  Total:	0.0323 secs
  Slowest:	0.0095 secs
  Fastest:	0.0003 secs
  Average:	0.0015 secs
  Requests/sec:	6184.6830

Latency distribution:
  10%% in 0.0007 secs
  25%% in 0.0009 secs
  50%% in 0.0012 secs
  75%% in 0.0015 secs
  90%% in 0.0019 secs
  95%% in 0.0031 secs
  99%% in 0.0091 secs

Status code distribution:
  [200]	200 responses
```

### `GET /projects/{project_id}/metrics`

```
Summary:
  Total:	0.0263 secs
  Slowest:	0.0063 secs
  Fastest:	0.0004 secs
  Average:	0.0012 secs
  Requests/sec:	7596.8919
  Total data:	118400 bytes
  Size/request:	592 bytes

Latency distribution:
  10%% in 0.0006 secs
  25%% in 0.0007 secs
  50%% in 0.0009 secs
  75%% in 0.0013 secs
  90%% in 0.0019 secs
  95%% in 0.0043 secs
  99%% in 0.0061 secs

Status code distribution:
  [200]	200 responses
```

### `GET /projects/{project_id}/defects`

```
Summary:
  Total:	0.0103 secs
  Slowest:	0.0020 secs
  Fastest:	0.0001 secs
  Average:	0.0005 secs
  Requests/sec:	19478.7828

Latency distribution:
  10%% in 0.0002 secs
  25%% in 0.0003 secs
  50%% in 0.0004 secs
  75%% in 0.0005 secs
  90%% in 0.0008 secs
  95%% in 0.0015 secs
  99%% in 0.0019 secs

Status code distribution:
  [200]	200 responses
```

## Limitaciones explícitas

- **Medición puntual, no continua.** Se ejecutó una sola vez; no hay repeticiones ni promedio entre corridas, así que no se puede descartar ruido puntual del sistema (otros procesos del host, variabilidad del contenedor).
- **Una sola máquina de desarrollo, no un entorno de staging ni de producción.** El resultado no es representativo de hardware, red o configuración de un entorno real.
- **Poco volumen de datos.** 100 historias y 10 defectos en un solo proyecto; no se probó con múltiples proyectos concurrentes, con miles de filas, ni con datos que generen planes de consulta distintos (por ejemplo, forzando un `seq scan` en vez de usar un índice).
- **Carga corta y de baja concurrencia.** 200 requests con concurrencia 10 no es una prueba de carga sostenida ni de estrés; no mide comportamiento bajo picos, bajo saturación de conexiones a PostgreSQL, ni degradación a lo largo del tiempo.
- **Sin umbrales de aceptación definidos.** No hay un objetivo de latencia o throughput acordado con el equipo; los números de esta tabla son una referencia, no un criterio de aprobación o rechazo. No se declara que ningún endpoint "cumple" o "no cumple" nada.
- **API y cliente de carga en la misma máquina.** No se midió latencia de red real; todo el tráfico fue loopback (`localhost`).
- **No se perfiló el código ni la base de datos.** No se usó `EXPLAIN ANALYZE`, ni profiling de Go, ni métricas de PostgreSQL (locks, uso de índices, cache hit ratio); esta medición es de caja negra, solo a nivel de API HTTP.
