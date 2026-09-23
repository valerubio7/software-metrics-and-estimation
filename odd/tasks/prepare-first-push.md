# Preparar primer push

## Objetivo
Corregir la ruta del módulo Go, cerrar el cambio SDD US-01 y exportar memoria Engram antes de publicar.

## Tareas
- [x] Corregir `go.mod` e imports internos a la ruta del repositorio remoto; commit `ee4148e` (`fix(module): align import path with GitHub repository`).
- [x] Archivar US-01 mediante SDD; commit `a955009` (`docs(sdd): archive completed US-01 project change`). Especificación canónica: `openspec/specs/project/spec.md`; cambio archivado: `openspec/changes/archive/2026-09-23-us-01-create-project/`.
- [x] Exportar la memoria de Engram al archivo `docs/memory/engram-export.json`, limitado a este proyecto (sin proyectos externos). Exportación informada: 30 observaciones, 25 sesiones y 49 prompts; SHA-256 `faa05ede2b7e10b9de59d1ce22f6d942f131da5cbf788bef268416d66c5f2e7a`. Se decidió explícitamente incluirla en el repositorio para el push.
- [x] Comprobar árbol limpio, pruebas y estado del remoto antes de decidir el push. Verificación independiente: `go test -count=1 ./...` y ambas pruebas PostgreSQL reales pasaron sin omisiones al ejecutar con `newgrp docker`; `git diff --check` y árbol limpios. `origin/main` todavía no existe: el primer push creará la rama remota.

## Nota
La exportación fue verificada y el usuario autorizó incluirla en el repositorio. El push sigue siendo una acción de publicación separada.
