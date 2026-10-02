# Propuesta: Registrar integrantes en un proyecto (US-03)

## Intención

Implementar US-03 (issue #31) para que un cliente que conoce el ID de un proyecto existente pueda registrar uno o más integrantes asociados a él. La solicitud debe validarse como un todo: datos inválidos o incompletos no deben producir registros y deben explicar el motivo. El registro de integrantes no debe modificar los datos básicos del proyecto.

El modelo mínimo de identificación confirmado para esta propuesta requiere nombre completo y permite email opcional. Esta decisión fue confirmada por el usuario; no se atribuye al issue.

## Alcance

### Incluido
- Ofrecer un contrato de registro de uno o más integrantes asociados al proyecto indicado por un ID existente suministrado por el cliente. No se incluye un endpoint de listado de proyectos ni una interfaz de selección.
- Validar todos los integrantes antes de almacenar: el nombre completo es obligatorio y el email es opcional. Si hay datos inválidos o incompletos, rechazar la solicitud, explicar el motivo y no almacenar ningún integrante.
- Hacer el lote atómico: o se registran todos sus integrantes válidos, o ninguno.
- Rechazar duplicados dentro de una misma solicitud y duplicados que ya estén registrados para ese proyecto; no se persiste ningún integrante del lote rechazado.
- Responder 404 para un proyecto inexistente y no almacenar miembros.
- Mantener intactos los datos básicos del proyecto y separar el modelo/ciclo de vida de integrantes del proyecto.
- Cubrir validación, errores HTTP, persistencia, asociación con proyecto, duplicados y atomicidad mediante pruebas unitarias y de integración.

### Fuera de alcance
- Crear proyectos (US-01), modificar sus datos básicos (US-02) o consultar su estado (US-04).
- Listado/búsqueda de proyectos, registro de integrantes en lote con éxito parcial, u otras operaciones de gestión de integrantes no solicitadas.
- Definir reglas de unicidad globales, eliminación/retención de integrantes, normalización de nombres o emails, o más campos identificativos sin decisión adicional.

## Capacidades

### Capacidades nuevas
- `project-members` (nombre provisional): registrar integrantes vinculados a un proyecto existente. La fase de especificación determinará si se integra como capacidad separada o como delta de `project` según las convenciones del repositorio.

### Capacidades modificadas
- `project`: únicamente en lo necesario para aceptar el registro asociado al proyecto; las operaciones existentes de creación y modificación de datos básicos deben conservarse sin cambios. La fase de especificación definirá el delta OpenSpec correspondiente.

## Enfoque propuesto

Añadir un corte vertical siguiendo las capas existentes de Go (transporte HTTP, aplicación, dominio y PostgreSQL), usando una ruta anidada bajo el proyecto como forma natural de asociación. El cliente aporta el ID; la selección/listado queda fuera. La solicitud representa un lote y el caso de uso debe validar duplicados y todos los datos antes de confirmar persistencia, con transacción para cumplir la atomicidad incluso ante errores de almacenamiento. La existencia del proyecto debe comprobarse antes de completar el registro.

La ruta/método exactos, formato del cuerpo y esquema de errores quedan para la especificación. La persistencia requerirá una tabla relacionada y migración; investigar y resolver antes la numeración de migraciones, pues la exploración detectó dos migraciones `000003` en el directorio compartido. La implementación debe preservar el arranque condicional de módulos existente, sin introducir cambios en los datos básicos de proyectos.

## Áreas afectadas

| Área | Impacto esperado | Descripción |
|---|---|---|
| `internal/` módulo de integrantes (nuevo) | Nuevo | Dominio y validación, caso de uso/puertos, transporte HTTP y repositorio PostgreSQL siguiendo el patrón existente. |
| `internal/api/api.go` | Modificado | Registrar la ruta anidada de integrantes y componer dependencias conforme a la disponibilidad de esquema existente. |
| Migraciones PostgreSQL | Nuevo | Tabla de integrantes y referencia al proyecto; resolver el conflicto de numeración existente antes de elegir la nueva migración. |
| `tests/unit/` | Nuevo/modificado | Validación de campos, duplicados, atomicidad lógica y errores de transporte/aplicación sin escrituras ante solicitud rechazada. |
| `tests/integration/` | Nuevo/modificado | Persistencia/asociación, proyecto ausente, conflictos de duplicados y rollback del lote ante fallo. |
| `openspec/specs/` | Futuro | Delta de especificación para el comportamiento de registro; no se crea en esta fase. |

## Supuestos y preguntas abiertas

- La API utilizará el ID de proyecto entregado por el cliente; no se infiere cómo obtiene dicho ID fuera de este cambio.
- El contrato HTTP exacto (ruta/método, forma de lista, respuesta de éxito y estructura/códigos para datos inválidos y duplicados) se definirá en la especificación.
- Debe acordarse el criterio de duplicidad y comparación (por ejemplo, igualdad exacta o normalizada de los campos disponibles), sin convertir email opcional en requisito ni asumir unicidad global.
- La exploración no determinó política de borrado/retención ni comportamiento futuro ante eliminación de un proyecto con integrantes. Esta propuesta no añade operaciones de eliminación; la integridad referencial y su política concreta requieren decisión en diseño.
- La exploración registró numeración conflictiva de migraciones `000003`; no se debe asumir un número hasta verificar el estado real.

## Riesgos

| Riesgo | Probabilidad | Mitigación |
|---|---|---|
| Lote parcialmente persistido por validación tardía o fallo de base de datos. | Media | Validar el lote antes de escribir y confirmar todas las escrituras en una sola transacción; probar rollback. |
| Regla de duplicados ambigua provoca rechazos inesperados o duplicación. | Media | Definir en la especificación la clave/semántica de comparación y limitar el alcance a duplicados del lote y del mismo proyecto. |
| Conflicto de numeración de migraciones rompe instalación o arranque. | Media | Auditar las migraciones existentes y resolver la numeración antes de añadir la tabla; probar inicialización del esquema. |
| Validación/almacenamiento de email opcional se interpreta como obligatorio o como identificador global. | Media | Mantener nombre completo requerido y email opcional explícitos; no introducir unicidad global sin decisión. |
| Errores en composición o integridad referencial mezclan el ciclo de vida del miembro con los datos básicos del proyecto. | Baja | Mantener límites de módulo y probar que el registro no altera los campos básicos. |

## Plan de reversión

Revertir el PR de implementación que agregue la ruta, módulo, migración y pruebas. Antes de revertir o retirar la migración, comprobar si ya existen datos de integrantes en entornos persistentes; la reversión no debe eliminar datos de usuario silenciosamente. Si la migración ya se aplicó, la estrategia de rollback de esquema/datos debe decidirse según el entorno y conservar los registros cuando corresponda. El cambio no debe requerir modificar ni revertir datos básicos de proyectos.

## Criterios de éxito

- [ ] Un cliente puede enviar uno o varios integrantes con el ID de un proyecto existente y quedan asociados a ese proyecto.
- [ ] Nombre completo es obligatorio y email opcional; solicitudes inválidas/incompletas explican el motivo y no almacenan ningún integrante.
- [ ] El lote es todo-o-nada, también ante error durante persistencia; no hay resultados parciales.
- [ ] Duplicados dentro del lote y ya registrados para el proyecto se rechazan sin escrituras del lote.
- [ ] Un proyecto inexistente produce 404 y cero integrantes almacenados.
- [ ] El registro no cambia nombre, fechas ni otros datos básicos del proyecto.
- [ ] Las pruebas `go test ./...` pasan; las pruebas de integración PostgreSQL/Testcontainers se ejecutan con Docker disponible.
