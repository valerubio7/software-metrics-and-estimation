package tui

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestDefectCreateSuccessReturnsToStoryDetailWithToast(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defect-create-confirm", "p-1"
	model, _ = model.submitDefectCreate()
	if model.Screen != "defect-create-confirm" || !strings.Contains(stripANSI(model.View()), "Registrando defecto…") {
		t.Fatalf("submission should stay on the confirmation with an indicator: screen=%q\n%s", model.Screen, stripANSI(model.View()))
	}
	updated, cmd := model.Update(defectCreateResult{ProjectID: "p-1", RequestID: model.defectCreateRequestID, Response: APIResponse{StatusCode: http.StatusCreated, Body: []byte(`{"id":"defect-new"}`)}})
	model = updated.(Model)
	if model.Screen != "story-detail" || cmd != nil {
		t.Fatalf("success = screen %q cmd nil=%v, want the story detail without a request", model.Screen, cmd == nil)
	}
	if !strings.Contains(stripANSI(model.View()), "✓ Defecto registrado") {
		t.Fatalf("detail lacks the toast:\n%s", stripANSI(model.View()))
	}
	model = updateStatusTestModel(model, tea.KeyMsg{Type: tea.KeyDown})
	if strings.Contains(stripANSI(model.View()), "Defecto registrado") {
		t.Fatal("the toast survived the next key")
	}
}

func TestDefectCreateRejectionStaysOnConfirmation(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defect-create-confirm", "p-1"
	model.defectCreateForm = defectCreateForm{description: "Cache crash", storyLabel: "Chosen story"}
	model, _ = model.submitDefectCreate()
	model = updateStatusTestModel(model, defectCreateResult{ProjectID: "p-1", RequestID: model.defectCreateRequestID, Response: APIResponse{StatusCode: http.StatusUnprocessableEntity, Body: []byte(`invalid story`)}})
	view := stripANSI(model.View())
	if model.Screen != "defect-create-confirm" || model.defectCreateLoading || !strings.Contains(view, "Cache crash") || !strings.Contains(view, "invalid story") {
		t.Fatalf("rejection = screen %q loading=%v, want the confirmation with the draft and the reason:\n%s", model.Screen, model.defectCreateLoading, view)
	}
}

func TestDefectStatusSuccessReturnsToDetailWithToast(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defect-status-select", "p-1"
	model.defectDetail = Defect{ID: "d-1", Description: "Crash", Status: "detectado"}
	model.defects = []Defect{model.defectDetail}
	model, _ = model.submitDefectStatus("en_correccion", "")
	if model.Screen != "defect-status-select" || !strings.Contains(stripANSI(model.View()), "Actualizando estado…") {
		t.Fatalf("submission should stay on the selector with an indicator: screen=%q", model.Screen)
	}
	model = updateStatusTestModel(model, defectStatusResult{ProjectID: "p-1", RequestID: model.defectStatusRequestID, Response: APIResponse{StatusCode: http.StatusOK, Body: []byte(`{"id":"d-1","description":"Crash","status":"en_correccion"}`)}})
	view := stripANSI(model.View())
	if model.Screen != "defect-detail" || model.defectDetail.Status != "en_correccion" || model.defects[0].Status != "en_correccion" {
		t.Fatalf("success = screen %q detail %+v list %+v, want the refreshed detail", model.Screen, model.defectDetail, model.defects)
	}
	if !strings.Contains(view, "✓ Estado del defecto actualizado: en_correccion") {
		t.Fatalf("detail lacks the toast:\n%s", view)
	}
}

func TestDefectStatusRejectionStaysOnSelector(t *testing.T) {
	model := NewModel("http://example.invalid")
	model.Screen, model.activeProjectID = "defect-status-select", "p-1"
	model.defectDetail = Defect{ID: "d-1"}
	model, _ = model.submitDefectStatus("en_correccion", "")
	model = updateStatusTestModel(model, defectStatusResult{ProjectID: "p-1", RequestID: model.defectStatusRequestID, Response: APIResponse{StatusCode: http.StatusConflict, Body: []byte(`transition not allowed`)}})
	if model.Screen != "defect-status-select" || model.defectStatusLoading || !strings.Contains(stripANSI(model.View()), "transition not allowed") {
		t.Fatalf("rejection = screen %q loading=%v:\n%s", model.Screen, model.defectStatusLoading, stripANSI(model.View()))
	}
}
