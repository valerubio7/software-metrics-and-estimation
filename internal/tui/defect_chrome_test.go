package tui

import (
	"strings"
	"testing"
)

func TestDefectFlowFootersAdvertiseOnlyAvailableActions(t *testing.T) {
	tests := []struct {
		name      string
		screen    string
		filtering bool
		loading   bool
		want      []string
		mustNot   []string
	}{
		{name: "list", screen: "defects", want: []string{"u", "actualizar estado"}},
		{name: "filtered list", screen: "defects", filtering: true, want: []string{"escribí", "para filtrar"}, mustNot: []string{"nuevo defecto", "actualizar estado"}},
		{name: "detail", screen: "defect-detail", want: []string{"u", "cambiar estado"}},
		{name: "create form", screen: "defect-create-form", want: []string{"Enter", "continuar"}},
		{name: "story selector", screen: "defect-story-select", want: []string{"r", "refrescar", "Enter", "elegir"}},
		{name: "filtered story selector", screen: "defect-story-select", filtering: true, want: []string{"escribí", "para filtrar"}, mustNot: []string{"refrescar", "proyecto"}},
		{name: "detection Sprint selector", screen: "defect-sprint-select", want: []string{"r", "refrescar", "Enter", "elegir"}},
		{name: "create confirmation", screen: "defect-create-confirm", want: []string{"s", "confirmar registro", "n", "cancelar"}},
		{name: "create loading", screen: "defect-create-result", loading: true, want: []string{"Ctrl+C", "salir"}, mustNot: []string{"Enter", "volver", "q", "proyecto"}},
		{name: "create result", screen: "defect-create-result", want: []string{"Enter/Esc", "volver"}},
		{name: "status selection", screen: "defect-status-select", want: []string{"Enter", "elegir estado"}},
		{name: "resolution Sprint selector", screen: "defect-resolution-select", want: []string{"r", "refrescar", "Enter", "elegir"}},
		{name: "filtered resolution selector", screen: "defect-resolution-select", filtering: true, want: []string{"escribí", "para filtrar"}, mustNot: []string{"refrescar", "proyecto"}},
		{name: "status loading", screen: "defect-status-result", loading: true, want: []string{"Ctrl+C", "salir"}, mustNot: []string{"Enter", "volver", "q", "proyecto"}},
		{name: "status result", screen: "defect-status-result", want: []string{"Enter/Esc", "volver"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel("http://localhost:8080")
			m.Screen, m.defectFiltering, m.defectCreateLoading, m.defectStatusLoading = tc.screen, tc.filtering, tc.loading, tc.loading
			if tc.screen == "defects" && !tc.filtering {
				m.defects = []Defect{{ID: "d-1", Description: "Visible defect"}}
			}
			m.defectStorySelector.filtering = tc.filtering && tc.screen == "defect-story-select"
			m.defectSprintSelector.filtering = tc.filtering && tc.screen == "defect-sprint-select"
			m.defectResolutionSelector.filtering = tc.filtering && tc.screen == "defect-resolution-select"
			if !tc.filtering && !tc.loading {
				item := []selectorItem{{ID: "item-1", Label: "Option"}}
				m.defectStorySelector.items = item
				m.defectSprintSelector.items = item
				m.defectResolutionSelector.items = item
			}
			footer := stripANSI(m.footer())
			for _, want := range tc.want {
				if !strings.Contains(footer, want) {
					t.Errorf("footer %q missing %q", footer, want)
				}
			}
			for _, unwanted := range tc.mustNot {
				if strings.Contains(footer, unwanted) {
					t.Errorf("footer %q must not advertise %q", footer, unwanted)
				}
			}
		})
	}
}

func TestDefectSelectorFooterDoesNotOfferChooseWhenEmpty(t *testing.T) {
	for _, screen := range []string{"defect-story-select", "defect-sprint-select", "defect-resolution-select"} {
		t.Run(screen, func(t *testing.T) {
			m := NewModel("http://localhost:8080")
			m.Screen = screen
			footer := stripANSI(m.footer())
			if strings.Contains(footer, "elegir") || strings.Contains(footer, "Enter") {
				t.Errorf("empty selector footer advertises choosing an unavailable option: %q", footer)
			}
			m.showHelp = true
			for _, row := range m.helpRows() {
				if row[0] == "Enter" {
					t.Errorf("empty selector help advertises choosing an unavailable option: %#v", row)
				}
			}
		})
	}
}

func TestDefectSelectorFooterReflectsLoadingAndError(t *testing.T) {
	for _, screen := range []string{"defect-story-select", "defect-sprint-select", "defect-resolution-select"} {
		t.Run(screen, func(t *testing.T) {
			for _, state := range []string{"loading", "error"} {
				t.Run(state, func(t *testing.T) {
					m := NewModel("http://localhost:8080")
					m.Screen = screen
					selector := defectSelectorForScreen(&m, screen)
					if state == "loading" {
						selector.startLoading()
					} else {
						selector.fail("API unavailable")
					}
					footer := stripANSI(m.footer())
					if strings.Contains(footer, "Enter") || strings.Contains(footer, "elegir") {
						t.Errorf("%s footer advertises an unavailable selection: %q", state, footer)
					}
					want := "Esc"
					if state == "error" {
						want = "reintentar"
					}
					if !strings.Contains(footer, want) {
						t.Errorf("%s footer %q missing %q", state, footer, want)
					}
				})
			}
		})
	}
}

func defectSelectorForScreen(m *Model, screen string) *selector {
	switch screen {
	case "defect-story-select":
		return &m.defectStorySelector
	case "defect-sprint-select":
		return &m.defectSprintSelector
	default:
		return &m.defectResolutionSelector
	}
}

func TestDefectFlowBreadcrumbsUseSpanishLabels(t *testing.T) {
	labels := map[string]string{
		"defects": "Defectos", "defect-detail": "Detalle", "defect-create-form": "Registrar defecto",
		"defect-story-select": "Historia", "defect-sprint-select": "Sprint de detección",
		"defect-create-confirm": "Confirmar registro", "defect-create-result": "Resultado del registro",
		"defect-status-select": "Cambiar estado", "defect-resolution-select": "Sprint de resolución",
		"defect-status-result": "Resultado del estado",
	}
	for screen, label := range labels {
		t.Run(screen, func(t *testing.T) {
			m := NewModel("http://localhost:8080")
			m.Screen, m.activeProjectID, m.activeProjectName = screen, "p-1", "Proyecto Alfa"
			view := stripANSI(m.View())
			if !strings.Contains(view, label) || !strings.Contains(view, "Proyectos › Proyecto Alfa › Defectos") {
				t.Errorf("breadcrumb %q does not contain %q and the project context: %q", view, label, view)
			}
			for _, raw := range []string{"status-select", "status-result", "create-result", "resolution-select", "sprint-select", "story-select", "create-confirm", "create-form"} {
				if strings.Contains(view, raw) {
					t.Errorf("breadcrumb exposes internal screen name %q: %q", raw, view)
				}
			}
		})
	}
}

func TestDefectHelpDescribesSupportedActionsOnly(t *testing.T) {
	for _, screen := range []string{"defects", "defect-detail", "defect-create-form", "defect-story-select", "defect-sprint-select", "defect-create-confirm", "defect-create-result", "defect-status-select", "defect-resolution-select", "defect-status-result"} {
		t.Run(screen, func(t *testing.T) {
			m := NewModel("http://localhost:8080")
			m.Screen, m.showHelp = screen, true
			if screen == "defects" {
				m.defects = []Defect{{ID: "d-1", Description: "Visible defect"}}
			}
			view := stripANSI(m.View())
			if !strings.Contains(view, "Atajos de teclado") {
				t.Fatal("help overlay is not visible")
			}
			for _, unsupported := range []string{"Editar el elemento", "Marcar historias", "Cerrar el Sprint"} {
				if strings.Contains(view, unsupported) {
					t.Errorf("help advertises unsupported action %q on %s", unsupported, screen)
				}
			}
			if screen == "defects" && (!strings.Contains(view, "Actualizar el estado") || !strings.Contains(view, "Crear un defecto")) {
				t.Error("defect list help must describe creation and status update")
			}
			if screen == "defect-detail" && !strings.Contains(view, "Cambiar el estado") {
				t.Error("defect detail help must describe status update")
			}
		})
	}
}

func TestDefectSelectorHelpDistinguishesFilteringFromChoosing(t *testing.T) {
	for _, screen := range []string{"defect-story-select", "defect-sprint-select", "defect-resolution-select"} {
		t.Run(screen, func(t *testing.T) {
			for _, filtering := range []bool{false, true} {
				state := "idle"
				if filtering {
					state = "filtering"
				}
				t.Run(state, func(t *testing.T) {
					m := NewModel("http://localhost:8080")
					m.Screen = screen
					selector := defectSelectorForScreen(&m, screen)
					selector.items = []selectorItem{{ID: "option-1", Label: "Option"}}
					selector.filtering = filtering
					rows := m.helpRows()
					enterAction := helpDescription(rows, "Enter")
					if filtering {
						if strings.Contains(enterAction, "Elegir") || enterAction != "Dejar de filtrar" {
							t.Errorf("filtering Enter help = %q, want only Dejar de filtrar", enterAction)
						}
						if helpDescription(rows, "?") != "" {
							t.Error("filtering help must not advertise the suppressed help shortcut")
						}
					} else if enterAction != "Elegir la opción resaltada" {
						t.Errorf("idle Enter help = %q, want choosing the highlighted option", enterAction)
					}
				})
			}
		})
	}
}

func TestDefectListItemActionsRequireVisibleDefects(t *testing.T) {
	tests := []struct {
		name        string
		defects     []Defect
		filter      string
		wantActions bool
	}{
		{name: "empty list"},
		{name: "filtered list with no matches", defects: []Defect{{ID: "d-1", Description: "Visible defect"}}, filter: "missing"},
		{name: "visible match", defects: []Defect{{ID: "d-1", Description: "Visible defect"}}, filter: "visible", wantActions: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel("http://localhost:8080")
			m.Screen, m.defects, m.defectFilter = "defects", tc.defects, tc.filter
			footer := stripANSI(m.footer())
			m.showHelp = true
			help := stripANSI(m.View())
			for name, assertions := range map[string]struct {
				view    string
				actions []string
			}{
				"footer": {view: footer, actions: []string{"detalle", "actualizar estado"}},
				"help":   {view: help, actions: []string{"detalle del defecto", "actualizar el estado"}},
			} {
				for _, action := range assertions.actions {
					if got := strings.Contains(strings.ToLower(assertions.view), action); got != tc.wantActions {
						t.Errorf("%s action %q visibility = %v, want %v: %q", name, action, got, tc.wantActions, assertions.view)
					}
				}
			}
		})
	}
}

func helpDescription(rows [][2]string, key string) string {
	for _, row := range rows {
		if row[0] == key {
			return row[1]
		}
	}
	return ""
}
