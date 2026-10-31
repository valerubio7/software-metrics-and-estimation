package tui_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

const flatEffortRecord = `{"id":"e1","project_id":"p1","story_id":"s1","task_id":"t1","member_id":"m1","work_date":"2026-10-05","description":"Implementar login","hours":3.5}`

func TestEffortClientRegistersTaskEffortWithExactContract(t *testing.T) {
	server, rec := recordingServer(t, http.StatusCreated, flatEffortRecord)
	input := tui.EffortInput{
		MemberID: "m1", TaskID: "t1", WorkDate: "2026-10-05", Description: "Implementar login", Hours: 3.5,
	}
	response, err := tui.NewEffortClient(server.URL).Register(t.Context(), "p1", "s1", input)
	if err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if got.Method != http.MethodPost || got.Path != "/projects/p1/stories/s1/effort-records" {
		t.Errorf("request = %s %s, want POST /projects/p1/stories/s1/effort-records", got.Method, got.Path)
	}
	keys := bodyKeys(t, got.Body)
	want := []string{"description", "hours", "member_id", "task_id", "work_date"}
	if names := sortedKeys(keys); !equalStrings(names, want) {
		t.Errorf("body keys = %v, want exactly %v", names, want)
	}
	for key, wantRaw := range map[string]string{
		"member_id": `"m1"`, "task_id": `"t1"`, "work_date": `"2026-10-05"`,
		"description": `"Implementar login"`, "hours": `3.5`,
	} {
		if string(keys[key]) != wantRaw {
			t.Errorf("body %s = %s, want %s", key, keys[key], wantRaw)
		}
	}
	if response.StatusCode != http.StatusCreated || string(response.Body) != flatEffortRecord {
		t.Errorf("response = %+v, want 201 with the API body", response)
	}
}

func TestEffortClientOmitsTaskIDForWholeStory(t *testing.T) {
	server, rec := recordingServer(t, http.StatusCreated, flatEffortRecord)
	input := tui.EffortInput{MemberID: "m1", WorkDate: "2026-10-05", Description: "Revisión", Hours: 2}
	if _, err := tui.NewEffortClient(server.URL).Register(t.Context(), "p1", "s1", input); err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	keys := bodyKeys(t, got.Body)
	want := []string{"description", "hours", "member_id", "work_date"}
	if names := sortedKeys(keys); !equalStrings(names, want) {
		t.Errorf("body keys = %v, want exactly %v", names, want)
	}
	if strings.Contains(string(got.Body), "task_id") || strings.Contains(string(got.Body), "null") {
		t.Errorf("body %s must not mention task_id nor null for the whole story", got.Body)
	}
}

func TestEffortClientEscapesPathSegments(t *testing.T) {
	server, rec := recordingServer(t, http.StatusCreated, flatEffortRecord)
	input := tui.EffortInput{MemberID: "m1", WorkDate: "2026-10-05", Description: "x", Hours: 1}
	if _, err := tui.NewEffortClient(server.URL).Register(t.Context(), "p 1", "s/1", input); err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if want := "/projects/p%201/stories/s%2F1/effort-records"; got.Path != want {
		t.Errorf("effort path = %q, want %q", got.Path, want)
	}
}

func TestEffortClientReturnsHTTPErrorsAsResponses(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"validation failure", http.StatusUnprocessableEntity, `{"error":"validation_failed","fields":[{"field":"hours","message":"must be greater than 0"}]}`},
		{"unknown member", http.StatusNotFound, `{"error":"member_not_found"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, rec := recordingServer(t, tt.status, tt.body)
			input := tui.EffortInput{MemberID: "m1", WorkDate: "2026-10-05", Description: "x", Hours: 1}
			response, err := tui.NewEffortClient(server.URL).Register(t.Context(), "p1", "s1", input)
			if err != nil {
				t.Fatalf("HTTP %d must come back as a response, got error %v", tt.status, err)
			}
			onlyRequest(t, rec)
			if response.StatusCode != tt.status || string(response.Body) != tt.body {
				t.Errorf("response = %+v, want %d with the API body %s", response, tt.status, tt.body)
			}
		})
	}
}

func TestEffortClientReturnsTransportErrors(t *testing.T) {
	server, _ := recordingServer(t, http.StatusCreated, flatEffortRecord)
	client := tui.NewEffortClient(server.URL)
	server.Close()
	input := tui.EffortInput{MemberID: "m1", WorkDate: "2026-10-05", Description: "x", Hours: 1}
	response, err := client.Register(t.Context(), "p1", "s1", input)
	if err == nil {
		t.Fatalf("closed server must return a transport error, got response %+v", response)
	}
}

func TestEffortRecordDecodesFlatCreatedResponse(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantTaskID *string
	}{
		{"task effort carries the task id", flatEffortRecord, ptr("t1")},
		{"whole story effort has a null task id", `{"id":"e2","project_id":"p1","story_id":"s1","task_id":null,"member_id":"m1","work_date":"2026-10-06","description":"Revisión","hours":2}`, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server, _ := recordingServer(t, http.StatusCreated, tt.body)
			input := tui.EffortInput{MemberID: "m1", WorkDate: "2026-10-05", Description: "x", Hours: 1}
			response, err := tui.NewEffortClient(server.URL).Register(t.Context(), "p1", "s1", input)
			if err != nil {
				t.Fatal(err)
			}
			var record tui.EffortRecord
			if err := json.Unmarshal(response.Body, &record); err != nil {
				t.Fatalf("decode %s: %v", response.Body, err)
			}
			switch {
			case tt.wantTaskID == nil && record.TaskID != nil:
				t.Errorf("TaskID = %q, want nil pointer for null", *record.TaskID)
			case tt.wantTaskID != nil && (record.TaskID == nil || *record.TaskID != *tt.wantTaskID):
				t.Errorf("TaskID = %v, want pointer to %q", record.TaskID, *tt.wantTaskID)
			}
			if record.ProjectID != "p1" || record.StoryID != "s1" || record.MemberID != "m1" || record.ID == "" {
				t.Errorf("record = %+v, want flat ids decoded", record)
			}
			if record.WorkDate == "" || record.Description == "" || record.Hours <= 0 {
				t.Errorf("record = %+v, want date, description and hours decoded", record)
			}
		})
	}
}

func ptr[T any](v T) *T { return &v }
