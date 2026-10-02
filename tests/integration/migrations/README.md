# Fixtures locales de historial de migraciones (US-03)

Este directorio registra la caracterización RED de las dos formas hipotéticas de esquema al terminar la versión 4. No contiene copias de SQL histórico: cada fixture se construyó en una base desechable aplicando los archivos existentes `000001`, `000002`, exactamente una variante `000003` y `000004`. No conectar estas pruebas a una base persistente.

## Entorno histórico probado y límites

- PostgreSQL 18.6 instalado localmente bajo `$HOME/.local/opt/postgresql-18`.
- golang-migrate v4.19.1 instalado bajo `$HOME/.local/bin/migrate`.
- El clúster experimental usó `initdb`/`pg_ctl`, directorio `mktemp`, autenticación trust solo local, `-h 127.0.0.1`, socket `-k "$tmp"`, puerto efímero y limpieza con trap. No hubo acceso de producción.
- La invocación README-shaped `migrate -path internal/project/infrastructure/postgres/migrations -database "$URL" up` sobre DB nueva falla al descubrir el source con `duplicate migration file: 000003_create_sprints.down.sql`, antes de aplicar SQL. `schema_migrations` no se crea. Por tanto, esta prueba caracteriza SQL histórico directamente; no acredita ejecución de estas historias por golang-migrate ni por el runner productivo.

## Estados históricos observados

Aplicados los SQL en transacciones/DBs desechables con una fila válida `stories.status='pendiente'`:

| Rama 000003 | Después de 000004 | Datos preservados | Diferencia observable |
|---|---|---|---|
| `000003_add_story_estimated_hours` | Una historia, `estimated_hours` presente, `sprints` ausente, `status=pendiente`; 000004 agrega `seq` identity y unicidad `(project_id, seq)`. | 1 proyecto y 1 historia. | CHECK `stories_status_check` permite `pendiente`, `en_progreso`, `completada`. |
| `000003_create_sprints` | Una historia, `estimated_hours` ausente, tabla `sprints` presente, `status=pendiente`; 000004 agrega `seq` identity y unicidad `(project_id, seq)`. | 1 proyecto y 1 historia. | No contiene el CHECK de status. |

## Matriz reproducible de convergencia

Ejecutar desde la raíz del repositorio:

```bash
tests/integration/migrations/valid_history_convergence.sh
```

El harness crea únicamente un clúster PostgreSQL 18.6 desechable con `mktemp`, loopback y socket temporal, verifica que el binario local incluya golang-migrate v4.19.1, y elimina el clúster/directorio en el trap de salida. Permite sobrescribir `PG_BIN`, `PGSHAREDIR` y `MIGRATE_BIN`; no acepta una URL de base de datos ni se conecta a bases persistentes. Primero usa `goto 6` para instalación y ambas historias v4 válidas: conserva las comprobaciones de convergencia `6 false`, columnas/checks/Sprints, integrantes e índices NULL-aware, y reaplica el SQL histórico v5 **solo en el corte v6**. Después siembra integrantes y entidades base, ejecuta el upgrade canónico v6→v8 y verifica valores preservados, `sprint_stories`, FKs/PK y `sprints.is_closed` abierto por defecto. Persiste una asociación y cierre; `down 2` desde v8 vuelve a `6 false`, pierde esos datos de US-09 y conserva todas las filas/valores históricos, incluidos integrantes. Reaplicar 7/8 restaura esquema vacío y Sprints abiertos, no los datos perdidos. También comprueba instalación nueva con `up` hasta v8. Solo tras regresar a v6 se ejecuta el histórico `down 1` a `5 false`, que elimina integrantes y conserva proyectos/historias/Sprints. Nunca se reaplica el checker v5 sobre v8. Luego crea bases temporales separadas dentro del mismo clúster para story, sprint y project-member y ejecuta sus suites completas. Finalmente crea otra base temporal, exporta `PROJECT_TEST_DATABASE_URL`, `STORY_TEST_DATABASE_URL`, `SPRINT_TEST_DATABASE_URL` y `PROJECTMEMBER_TEST_DATABASE_URL` apuntando a esa base loopback, y ejecuta exactamente `go test ./... -count=1`. Los cuatro helpers rechazan hosts no loopback, crean schema/search_path único por test y eliminan schemas al limpiar. Si no se define su variable en otros entornos, conservan Testcontainers como fallback.

Resultado histórico previo a US-09 (no es evidencia de la matriz v8 actual): `PASS: fresh install and both valid v4 histories converge at 6 false with seeded data preserved; down 1 removes only v6 members and retains project/story/sprint rows; actual story, sprint, and project-member PostgreSQL integration suites passed.` Los intentos esperados de checks debilitados emiten error del migrator, pero el harness confirma fallo cerrado y preservación. El script también debe informar `server stopped` tras la limpieza. Esto solo valida el runner/version local y PostgreSQL desechable, no el runner ni el historial de producción.

### Alcance actual de US-09 y verificación

La secuencia ejecutable es única `000001`–`000008`; migraciones main 1–6 y fixtures históricos permanecen intactos. Las pruebas negativas de checks débiles y Sprints parciales siguen exigiendo fallo cerrado en `5 true` sin modificar datos; el fixture negativo legacy se conserva en su runner separado. `TestMigrationVersionsAreUnique` exige los ocho pares canónicos. Los helpers story/sprint aplican 1–8, sin resucitar la variante sprint 000003 como migración activa; conservan aislamiento loopback/schema del helper compartido.

Esta actualización no acredita ejecución de los shell runners ni instalación host: su matriz SQL final requiere verificación independiente en PostgreSQL 18.6 / migrate v4.19.1 descartables. Los checks Go usan Testcontainers salvo URL loopback explícita del harness. La compatibilidad pretendida es fresh/canonical-main-v6; el despliegue productivo de v5/v6 alternativos de la fuente es desconocido. Los down probados son destructivos y **no** una garantía de rollback seguro en producción.

## Fixture negativo y atomicidad local

En una DB nueva se insertó `status='legacy'` bajo el esquema de `000002` y se ejecutó `000003_add_story_estimated_hours.up.sql` mediante `psql -1` (una transacción). PostgreSQL devolvió error al agregar `stories_status_check`. Tras el rollback, la fila conservó `status='legacy'`, `estimated_hours` no existía y `stories_status_check` no existía. Esto demuestra rollback transaccional de PostgreSQL para ese SQL ejecutado con `psql -1`; **no** verifica dirty/rollback de golang-migrate, porque el source completo no es descubrible.

## Evidencia automatizada adicional (continuación apply)

- Se conserva `TestMigrationVersionsAreUnique`; RED observado contra las dos migraciones 000003 duplicadas.
- Las migraciones históricas exactas de creación de sprints se movieron a `migrations/fixtures/` con extensión `.sql` para retenerlas como fixtures no ejecutables por golang-migrate. El nombre de las migraciones activas es ahora único.
- Añadida `000005_reconcile_story_hours_and_sprints`: intenta completar la tabla `sprints` ausente y los objetos estimados ausentes, y valida objetos existentes esperados. No elimina objetos en su down.
- GREEN local: `go test ./tests/integration/migrations -run TestMigrationVersionsAreUnique -count=1` pasó.
- `go test ./...` se ejecutó: paquetes unitarios pasaron, pero pruebas PostgreSQL/Testcontainers fallaron por proveedor Docker no disponible (`failed to create Docker provider`). No se afirma convergencia automatizada.
- Un intento README-shaped de resolver source de golang-migrate con URL deliberadamente no conectable pasó la etapa de lectura de archivos y falló al conectar; no es una prueba de aplicación SQL. No se creó ni consultó ninguna base.
- Versiones comprobadas/disponibles: PostgreSQL local 18.6 y binario golang-migrate local v4.19.1 según caracterización previa. La semántica dirty/rollback de la migración nueva y las historias hipotéticas todavía requieren prueba en clúster desechable.

## Resultado de seguridad

La caracterización expone las dos ramas y la condición que bloquea una reconciliación segura sin más evidencia: la rama de horas puede fallar con estados previos incompatibles. No se probaron fixtures de avance/convergencia, no hay plan forward-only validado, no se modificó SQL y ninguna ruta se habilitó. Identidad del runner, historial/variante desplegados y checksum de producción permanecen desconocidos; rollout bloqueado.
