package tui_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

type recordedRequest struct {
	Method string
	Path   string // escaped path, as sent on the wire
	Body   []byte
}

// recorder captures every request received by a test server.
type recorder struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (r *recorder) all() []recordedRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]recordedRequest(nil), r.requests...)
}

// recordingServer answers every request with status and body, recording method, escaped path and raw body.
func recordingServer(t *testing.T, status int, body string) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		rec.mu.Lock()
		rec.requests = append(rec.requests, recordedRequest{Method: r.Method, Path: r.URL.EscapedPath(), Body: raw})
		rec.mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, rec
}

// onlyRequest fails unless exactly one request was recorded.
func onlyRequest(t *testing.T, rec *recorder) recordedRequest {
	t.Helper()
	requests := rec.all()
	if len(requests) != 1 {
		t.Fatalf("requests = %d (%+v), want exactly 1", len(requests), requests)
	}
	return requests[0]
}

// bodyKeys decodes a JSON object body into its raw values.
func bodyKeys(t *testing.T, body []byte) map[string]json.RawMessage {
	t.Helper()
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(body, &keys); err != nil {
		t.Fatalf("body %q is not a JSON object: %v", body, err)
	}
	return keys
}

func sortedKeys(keys map[string]json.RawMessage) []string {
	names := make([]string, 0, len(keys))
	for name := range keys {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func floatPtr(v float64) *float64 { return &v }

func updateInput(hours *float64) tui.StoryUpdateInput {
	return tui.StoryUpdateInput{
		Title: "Login", Description: "d", Priority: "alta", Status: "pendiente",
		AcceptanceCriteria: []string{"A", "B"}, EstimatedHours: hours,
	}
}

func TestBacklogClientListUsesGetOnProjectStories(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"project_id":"p1","stories":[]}`)
	response, err := tui.NewBacklogClient(server.URL).List(t.Context(), "p1")
	if err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if got.Method != http.MethodGet || got.Path != "/projects/p1/stories" || len(got.Body) != 0 {
		t.Errorf("request = %+v, want GET /projects/p1/stories without body", got)
	}
	if response.StatusCode != http.StatusOK || string(response.Body) != `{"project_id":"p1","stories":[]}` {
		t.Errorf("response = %+v, want 200 with the API body", response)
	}
}

func TestBacklogClientCreateSendsExactlyFourKeys(t *testing.T) {
	server, rec := recordingServer(t, http.StatusCreated, `{"id":"s1"}`)
	input := tui.StoryCreateInput{Title: "Login", Description: "d", Priority: "alta", AcceptanceCriteria: []string{"A", "B"}}
	response, err := tui.NewBacklogClient(server.URL).Create(t.Context(), "p1", input)
	if err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if got.Method != http.MethodPost || got.Path != "/projects/p1/stories" {
		t.Errorf("request = %s %s, want POST /projects/p1/stories", got.Method, got.Path)
	}
	keys := bodyKeys(t, got.Body)
	if want := []string{"acceptance_criteria", "description", "priority", "title"}; !equalStrings(sortedKeys(keys), want) {
		t.Errorf("body keys = %v, want %v", sortedKeys(keys), want)
	}
	var criteria []string
	if err := json.Unmarshal(keys["acceptance_criteria"], &criteria); err != nil || !equalStrings(criteria, []string{"A", "B"}) {
		t.Errorf("acceptance_criteria = %s (%v), want a JSON list [A B]", keys["acceptance_criteria"], err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want 201", response.StatusCode)
	}
}

func TestBacklogClientCreateSendsEmptyCriteriaAsList(t *testing.T) {
	server, rec := recordingServer(t, http.StatusCreated, `{}`)
	input := tui.StoryCreateInput{Title: "T", Description: "D", Priority: "alta", AcceptanceCriteria: []string{}}
	if _, err := tui.NewBacklogClient(server.URL).Create(t.Context(), "p1", input); err != nil {
		t.Fatal(err)
	}
	keys := bodyKeys(t, onlyRequest(t, rec).Body)
	if got := string(keys["acceptance_criteria"]); got != "[]" {
		t.Errorf("acceptance_criteria = %s, want []", got)
	}
}

func TestBacklogClientUpdateSendsSixKeysWithNullHours(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"id":"s1"}`)
	response, err := tui.NewBacklogClient(server.URL).Update(t.Context(), "p1", "s1", updateInput(nil))
	if err != nil {
		t.Fatal(err)
	}
	got := onlyRequest(t, rec)
	if got.Method != http.MethodPut || got.Path != "/projects/p1/stories/s1" {
		t.Errorf("request = %s %s, want PUT /projects/p1/stories/s1", got.Method, got.Path)
	}
	keys := bodyKeys(t, got.Body)
	want := []string{"acceptance_criteria", "description", "estimated_hours", "priority", "status", "title"}
	if !equalStrings(sortedKeys(keys), want) {
		t.Errorf("body keys = %v, want %v", sortedKeys(keys), want)
	}
	if hours := string(keys["estimated_hours"]); hours != "null" {
		t.Errorf("estimated_hours = %s, want literal null", hours)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("StatusCode = %d, want 200", response.StatusCode)
	}
}

func TestBacklogClientUpdateSendsHoursNumber(t *testing.T) {
	for _, tc := range []struct {
		hours float64
		want  string
	}{{8, "8"}, {2.5, "2.5"}, {-3, "-3"}} {
		t.Run(tc.want, func(t *testing.T) {
			server, rec := recordingServer(t, http.StatusOK, `{}`)
			if _, err := tui.NewBacklogClient(server.URL).Update(t.Context(), "p1", "s1", updateInput(floatPtr(tc.hours))); err != nil {
				t.Fatal(err)
			}
			keys := bodyKeys(t, onlyRequest(t, rec).Body)
			if got := string(keys["estimated_hours"]); got != tc.want {
				t.Errorf("estimated_hours = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestBacklogClientEscapesPathSegments(t *testing.T) {
	cases := []struct {
		id, escaped string
	}{
		{"a/b", "a%2Fb"},
		{"a b", "a%20b"},
		{"a?b", "a%3Fb"},
	}
	for _, tc := range cases {
		t.Run(tc.escaped, func(t *testing.T) {
			server, rec := recordingServer(t, http.StatusOK, `{}`)
			client := tui.NewBacklogClient(server.URL)
			if _, err := client.List(t.Context(), tc.id); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Create(t.Context(), tc.id, tui.StoryCreateInput{AcceptanceCriteria: []string{}}); err != nil {
				t.Fatal(err)
			}
			if _, err := client.Update(t.Context(), tc.id, tc.id, updateInput(nil)); err != nil {
				t.Fatal(err)
			}
			want := []string{
				"/projects/" + tc.escaped + "/stories",
				"/projects/" + tc.escaped + "/stories",
				"/projects/" + tc.escaped + "/stories/" + tc.escaped,
			}
			requests := rec.all()
			if len(requests) != len(want) {
				t.Fatalf("requests = %d, want %d", len(requests), len(want))
			}
			for i, request := range requests {
				if request.Path != want[i] {
					t.Errorf("request %d path = %q, want %q", i, request.Path, want[i])
				}
			}
		})
	}
}

func TestBacklogClientPreservesErrorBodies(t *testing.T) {
	cases := []struct {
		name   string
		status int
		body   string
	}{
		{"422-with-fields", http.StatusUnprocessableEntity, `{"error":"validation_failed","message":"invalid","fields":{"title":"required","priority":"invalid"}}`},
		{"404-json", http.StatusNotFound, `{"error":"project_not_found","message":"project not found"}`},
		{"405-plain-text", http.StatusMethodNotAllowed, "Method Not Allowed\n"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server, _ := recordingServer(t, tc.status, tc.body)
			client := tui.NewBacklogClient(server.URL)
			calls := []struct {
				op   string
				call func() (tui.APIResponse, error)
			}{
				{"List", func() (tui.APIResponse, error) { return client.List(t.Context(), "p1") }},
				{"Create", func() (tui.APIResponse, error) { return client.Create(t.Context(), "p1", tui.StoryCreateInput{}) }},
				{"Update", func() (tui.APIResponse, error) { return client.Update(t.Context(), "p1", "s1", updateInput(nil)) }},
			}
			for _, c := range calls {
				response, err := c.call()
				if err != nil {
					t.Errorf("%s error = %v, want nil for an HTTP error status", c.op, err)
				}
				if response.StatusCode != tc.status || string(response.Body) != tc.body {
					t.Errorf("%s response = %d %q, want %d %q", c.op, response.StatusCode, response.Body, tc.status, tc.body)
				}
			}
		})
	}
}

func TestBacklogClientTransportErrorWhenServerClosed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	url := server.URL
	server.Close()
	client := tui.NewBacklogClient(url)

	if _, err := client.List(t.Context(), "p1"); err == nil {
		t.Error("List error = nil, want transport error")
	}
	if _, err := client.Create(t.Context(), "p1", tui.StoryCreateInput{}); err == nil {
		t.Error("Create error = nil, want transport error")
	}
	if _, err := client.Update(t.Context(), "p1", "s1", updateInput(nil)); err == nil {
		t.Error("Update error = nil, want transport error")
	}
}
