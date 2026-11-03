# Propuesta: rediseño de la experiencia de usuario de la TUI (TUI-12)

> Estado: **aprobada** (2026-11-03). Decisiones resueltas al final.

## Intención

La TUI hoy es confusa y repetitiva: el proyecto se elige a mitad de cada flujo, el Dashboard está escondido en la octava opción, cada operación termina en una pantalla de resultado que exige otro `Enter` y los footers listan una docena de atajos. Este cambio reorganiza la navegación alrededor de una idea: **primero elegís el proyecto, después trabajás dentro de él**, y el Dashboard es lo primero que ves.

## Cambios propuestos

### 1. Jerarquía: Proyectos → Espacio de trabajo del proyecto

- **Inicio**: la TUI abre en «Proyectos». Si existe un solo proyecto, entra directo a su espacio de trabajo. Desde la lista: `Enter` abre, `n` crea, `e` edita, `/` busca.
- **Espacio de trabajo**: encabezado fijo con el nombre y estado del proyecto, y una barra de pestañas:
  `Dashboard · Backlog · Sprints · Estimación · Esfuerzo · Defectos`.
  - El **Dashboard es la pestaña inicial** al entrar a un proyecto.
  - `Tab`/`Shift+Tab` o `1`–`6` cambian de pestaña desde cualquier lista raíz.
  - `Esc` en la raíz de una pestaña vuelve a Proyectos (cambiar de proyecto deja de ser un atajo escondido).
- Se eliminan del flujo el `pickerReturn`/`pickerPurpose`, los atajos `p` y `c` de cambio de proyecto y el menú plano de nueve opciones.

### 2. Dashboard como centro del proyecto

- Une el Dashboard actual y la sección Métricas: métricas del proyecto arriba y, debajo, el Sprint abierto con su progreso. Elegir otro Sprint (`s`) muestra sus métricas en la misma pantalla.
- `d` descarga el PDF del ámbito mostrado; las demás secciones quedan a una tecla con las pestañas (`1`–`6`), por lo que no se agregan accesos rápidos duplicados.

### 3. Menos pasos, menos pantallas

- Las operaciones exitosas vuelven a la lista de origen con un **aviso efímero** (toast) en lugar de una pantalla de resultado. Los errores se muestran en el mismo formulario, conservando lo escrito.
- Esfuerzo y Estimación parten de la historia resaltada cuando se invocan desde el backlog (`E` registrar esfuerzo, `P` estimar), sin recorrer de nuevo el selector de historias.

### 4. Atajos consistentes y ayuda contextual

- Un contrato único: `↑/↓` mover, `Enter` abrir/acción primaria, `n` nuevo, `e` editar, `/` buscar, `r` refrescar, `Esc` volver, `?` ayuda, `q` salir.
- El footer muestra **como máximo 5 atajos** prioritarios + `? más`; la ayuda (`?`) lista todos los de la pantalla actual agrupados.

### 5. Base técnica para sostenerlo

- Declarar los atajos de cada pantalla en un registro único del que se derivan el pie y la ayuda, y reemplazar el retorno dinámico del selector de proyectos por la jerarquía fija Proyectos → pestaña → flujo (ver `design.md`, D1–D3).
- La refactorización se hace por pasos con las pruebas existentes como red, antes de cambiar comportamiento.

## Capacidades

### Capacidades modificadas

- `tui`: se reescriben los requisitos de navegación (menú base, proyecto activo único, apertura de secciones, Dashboard, pantallas de resultado y atajos). Los requisitos de formularios, validaciones y errores de la API se conservan salvo donde cambie la presentación (toast/modal).

## Impacto

Solo `internal/tui/`, sus pruebas, `README.md` y la especificación `tui`. No cambia API, dominio, migraciones ni datos. Supera el presupuesto de 400 líneas: se entrega como **PRs encadenados** (ver `tasks.md` cuando se apruebe).

## Fuera de alcance

- Cambios en endpoints o reglas de negocio.
- Soporte de mouse y temas configurables.

## Decisiones resueltas

1. Navegación dentro del proyecto: **pestañas superiores**.
2. Métricas: **se integra en el Dashboard**.
3. Pantallas de resultado: **se reemplazan por avisos efímeros**.
4. Refactorización interna: **incluida como primera tarea**. En el diseño se acota al contrato de presentación (registro de atajos) y la pila de navegación se reemplaza por la jerarquía fija de pestañas, porque no hay retornos dinámicos que la justifiquen (D1 y D3).
