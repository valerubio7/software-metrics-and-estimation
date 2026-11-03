# Informe de verificación: tui-12-rediseno-ux

- Commit verificado: `f7b4ee1`, rama `feat/tui-ux-redesign`, con los cambios de documentación de la tarea 6 sin commitear (README, delta, este informe).
- Modo: TDD estricto, almacén `openspec`. Esta verificación cumple la tarea 6 de `tasks.md`; no se modificó código de producción ni pruebas durante la verificación.
- **Veredicto: PASS** con una limitación: no se ejecutaron las pruebas de integración (ver §4).

## 1. Ejecución real

| Comando | Resultado |
|---|---|
| `go build ./...` | exit 0, sin salida |
| `go vet ./...` | exit 0, sin hallazgos |
| `gofmt -l internal tests` | `internal/tui/metrics.go` (preexistente, no modificado por este cambio) |
| `go test -count=1 ./internal/tui ./tests/unit/tui` | `ok internal/tui 0.021s`, `ok tests/unit/tui 0.178s` |
| `openspec validate tui-12-rediseno-ux --strict` | sin `ERROR`; quedan advertencias (ver §2) |

## 2. Validación del delta

La primera corrida de `openspec validate --strict` reportó 6 `ERROR`: seis requisitos MODIFIED omitían escenarios que la spec vigente todavía tiene. Correcciones:

| Requirement | Corrección |
|---|---|
| Listar y elegir proyectos con un buscador | Se reescribieron a la nueva navegación «Abrir el navegador desde Proyectos» (arranque), «Abrir el selector desde el backlog» (`Esc` en la raíz del Backlog) y «Volver sin cambiar de proyecto» (`Esc` sin filtro en el navegador no hace nada y conserva el `✓`) |
| Precargar la modificación de proyecto y resolver el resultado | Se restituyó el nombre «Vuelta al formulario tras el rechazo», con el comportamiento nuevo (permanece en el formulario) |
| Presentar los errores de la API del backlog | Se restituyó el nombre «HTTP 422 al consultar la lista del proyecto activo» |
| Operar los Sprints mediante la TUI | Se restituyó «La opción Sprints abre el navegador» (ahora la pestaña) y se reescribió «Sin proyecto activo abre el selector» (al arrancar, el navegador precede a toda consulta de Sprints) |
| Salir limpiamente de la TUI | Pasa a REMOVED: el menú y las pantallas de resultado ya no existen. Lo vigente queda en el ADDED «Salir limpiamente desde el navegador y las pestañas» |
| Recorrer hacia atrás el flujo de esfuerzo y descartar respuestas tardías | Pasa a REMOVED: ya no hay pantalla de resultado ni consulta de métricas posterior. Lo vigente queda en el ADDED «Retroceder en el flujo de esfuerzo y descartar respuestas tardías» |

El texto de «Atajos contextuales de estimación y esfuerzo» se acortó: el retroceso de a uno, el solo-lectura y las minúsculas pasaron a escenarios. Las advertencias SHALL/MUST son esperables en una spec en español (`DEBERÁ`). Quedan cuatro advertencias de texto largo (>500 caracteres), que no bloquean, en «Espacio de trabajo del proyecto con pestañas», «Dashboard con las métricas del proyecto y de un Sprint», «Avisos efímeros en lugar de pantallas de resultado» y «Retroceder en el flujo de esfuerzo…».

Contraste con el código (`workspace.go`, `project_picker.go`, `model.go`, `keys.go`): el orden de pestañas, `tabForKey`, `autoOpen` en la primera carga, `Esc` sin filtro inerte en el navegador, la marca `✓` del último proyecto abierto y `quitsOnQ` (navegador, listas, selectores, confirmaciones y raíces de pestaña) coinciden con el delta. Ajuste aplicado a la spec: el requisito de salida nombra también los selectores y las confirmaciones, donde `q` ya salía.

## 3. Trazabilidad spec ↔ pruebas

| Requirement (delta `tui`) | Pruebas principales |
|---|---|
| Iniciar en el navegador de proyectos (ADDED) | `TestStartupOpensTheProjectBrowser`, `TestStartupWithASingleProjectOpensItsDashboard`, `TestLeavingTheSingleProjectDoesNotReopenIt` |
| Espacio de trabajo del proyecto con pestañas (ADDED) | `TestEnterOnAProjectOpensItsWorkspaceTabs`, `TestTabsSwitchWithTabDigitsAndShiftTab`, `TestShiftTabFromDashboardWrapsToDefects`, `TestDigitsAreTypedIntoAFilter`, `TestEscOnATabRootReturnsToProjects`, `TestEscInsideAFlowStaysInTheWorkspace`, `TestChangingProjectRetargetsEveryTab`, `TestTabsRouteToEachSprint3Section` |
| Dashboard con las métricas del proyecto y de un Sprint (ADDED) | `TestDashboardScopeSelectorListsProjectAndOpenSprints`, `TestDashboardSprintScopeShowsSprintMetricsExactly`, `TestDashboardWholeProjectScopeReturnsToProjectMetrics`, `TestDashboardScopeSelectorEscKeepsScope`, `TestDashboardDownloadsSprintPDFInSprintScope`, `TestDashboardRefreshReloadsTheSprintScope`, `TestDashboardDiscardsStaleScopeResults`, `TestOldMetricsSectionIsUnreachable` |
| Avisos efímeros en lugar de pantallas de resultado (ADDED) | `TestProjectCreateReturnsToBrowserWithToast`, `TestStoryCreateReturnsToBacklogWithToast`, `TestSprintCreateReturnsToSprintsWithToast`, `TestEffortRegistrationReturnsToEffortRootWithToast`, `TestProjectFormIgnoresRepeatsWhileSaving`, `TestDefectCreateSuccessReturnsToListWithToast`, `TestDefectStatusSuccessReturnsToDetailWithToast` |
| Pie de teclas acotado y ayuda agrupada (ADDED) | `TestSprintListFooterShowsAtMostFiveShortcutsAndMore`, `TestSprintListHelpGroupsEveryShortcut`, `TestScreenWithFewShortcutsHasNoMore`, `TestFilterAndTextFieldFootersDoNotAdvertiseHelp`, `TestTabRootHelpListsTabKeys` |
| Atajos contextuales de estimación y esfuerzo (ADDED) | `TestBacklogPOpensEstimationParticipantsForHighlightedStory`, `TestBacklogEOpensEffortTasksForHighlightedStory`, `TestEscFromPreselectedSelectorReturnsToBacklog`, `TestDeeperEscStillGoesBackOneStep`, `TestContextualShortcutFlowOnlyReads`, `TestShortcutsDoNothingOnEmptyBacklog`, `TestLowercasePAndEStillStartASearch`, `TestBacklogHelpListsPAndE` |
| Salir limpiamente desde el navegador y las pestañas (ADDED) | `TestQuitOnQFromProjectBrowserAndLists`, `TestBrowserEscWithoutFilterStaysAndQQuits`, `TestQuitOnQDoesNotExitInProjectTextFields`, `TestQuitOnQIsScopedInBacklogScreens`, `TestCtrlCQuitsFromEveryBacklogScreen`, `TestCtrlCQuitsFromProjectForms` |
| Retroceder en el flujo de esfuerzo y descartar respuestas tardías (ADDED) | `TestEffortEscWalksBackThroughThreeLevels`, `TestEffortEscFromMembersRestoresTasksSnapshotWithoutRequests`, `TestEffortTaskFilterIsKeptWhenWalkingBackFromForm`, `TestEffortSecondRegistrationStartsWithInitialForm`, `TestEffortLateMembersResponseAfterEscIsDropped`, `TestEffortLateTasksResponseAfterEscIsDropped`, `TestEffortProjectSwitchAfterRegistrationReopensStoriesAndDiscardsFlow` |
| Listar y elegir proyectos con un buscador (MODIFIED) | `TestStartupOpensTheProjectBrowser`, `TestBacklogPickerFiltersAndConfirmsProject`, `TestEscOnATabRootReturnsToProjects`, `TestBrowserEscWithoutFilterStaysAndQQuits` |
| Precargar la modificación de proyecto y resolver el resultado (MODIFIED) | `TestEditProjectPrefillsFromSelectedSummary`, `TestProjectSuccessReturnsToBrowser`, `TestProjectFailureStaysOnFormWithError`, `TestProjectRejectionKeepsTheForm` |
| Listas navegables con teclas uniformes (MODIFIED) | `TestBacklogSearchFiltersStories`, `TestSprintBrowserFiltersByGoalWithoutRequests`, `TestSprintBrowserEscapeClearsTheFilterFirst`, `TestDigitsAreTypedIntoAFilter` |
| Crear historias mediante la TUI (MODIFIED) | `TestStoryCreateReturnsToBacklogWithToast`, `TestBacklogCreateSplitsCriteriaBySemicolon`, `TestBacklogClientCreateSendsEmptyCriteriaAsList`, `TestBacklogCreateDoesNotValidatePriorityLocally` |
| Presentar los errores de la API del backlog (MODIFIED) | `TestStoryRejectionKeepsTheForm`, `TestProjectAPIErrorPreserves422And404Body`, `TestBacklogTransportErrorDoesNotQuit`, `TestBacklogClientTransportErrorWhenServerClosed` |
| Operar los Sprints mediante la TUI (MODIFIED) | `TestSprintBrowserListsOpenSprintsWithNavigatorKeys`, `TestSprintCreateReturnsToSprintsWithToast`, `TestSprintAssignmentSelectsOpenSprintAndPreservesSelectedStoryIDs`, `TestSprintCloseAndHistoryFlows`, `TestSprintStoryActionsCreateTasksAndComplete`, `TestSprintClosureConflictAndEmptyHistory` |
| Enviar el registro de esfuerzo y presentar los errores de la API (MODIFIED) | `TestEffortClientRegistersTaskEffortWithExactContract`, `TestEffortClientOmitsTaskIDForWholeStory`, `TestEffortSubmitSendsDescriptionUntrimmed`, `TestEffortDoubleEnterEmitsSingleRequest`, `TestEffortEscIgnoredWhileSaving`, `TestEffortFormTransportErrorWarnsUnconfirmed`, `TestEffortFormCorrectAndResubmitAfterRejection`, `TestEffortToastShowsRecordFromAPIAndSelection` |
| REMOVED (menú, proyecto activo único, secciones y Dashboard desde el menú, resultado del esfuerzo, salida y retroceso previos) | `TestOldProjectShortcutsAreGone`, `TestOldMetricsSectionIsUnreachable`; las pruebas del menú y de las pantallas de resultado se eliminaron en las UT 1 y 3 |

## 4. No verificado

- **Pruebas de integración**: no se ejecutó `go test ./...` porque los paquetes de `tests/integration` requieren Docker y Testcontainers, fuera del alcance de esta tarea. El cambio no toca la API ni la persistencia (solo `internal/tui` y documentación), por lo que el riesgo es bajo, pero no hay evidencia observada en esta verificación.
- **Uso interactivo real**: las pruebas manejan el modelo con `Model.Update` y `httptest`; no se probó la TUI contra una API en vivo en una terminal.

## 5. Veredicto

**PASS.** El delta valida en modo estricto sin errores, describe el comportamiento implementado y cada requisito tiene pruebas que pasan. El cambio queda listo para archivar y consolidar en `openspec/specs/tui/spec.md`.
