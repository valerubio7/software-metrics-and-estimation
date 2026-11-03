# Tareas: rediseño de la experiencia de usuario de la TUI (TUI-12)

Cada tarea es una unidad de trabajo con sus pruebas (TDD estricto) y su documentación, y un commit propio. La evidencia observada de cada una está en `apply-progress.md`.

El orden de ejecución fue 2 → 3 → 4 → 1 → 5 → 6: el registro de atajos se aplicó después de los cambios de navegación para no declarar atajos de pantallas que esos cambios eliminan.

- [x] 1. Registro de atajos por pantalla: pie con máximo cinco atajos y `? más`, ayuda agrupada (Requirement: Pie de teclas acotado y ayuda agrupada). `9fea460`
- [x] 2. Navegador de proyectos inicial y espacio de trabajo con pestañas; eliminar menú, `p`, `c` y el retorno del selector (Requirements: Iniciar en el navegador de proyectos, Espacio de trabajo del proyecto con pestañas, Listar y elegir proyectos con un buscador, Salir limpiamente, Listas navegables con teclas uniformes). `8695026`
- [x] 3. Dashboard con métricas del proyecto y de un Sprint; eliminar la sección Métricas (Requirement: Dashboard con las métricas del proyecto y de un Sprint). `358299c`
- [x] 4. Avisos efímeros en lugar de pantallas de resultado (Requirements: Avisos efímeros en lugar de pantallas de resultado, Precargar la modificación de proyecto y resolver el resultado). `c7dddb0`
- [x] 5. Atajos contextuales `P` y `E` desde el backlog (Requirement: Atajos contextuales de estimación y esfuerzo). `f7b4ee1`
- [x] 6. README, consolidación de la especificación `tui` y archivado del cambio.
