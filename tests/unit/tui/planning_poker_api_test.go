package tui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/tui"
)

func TestStoryPointsClientSetsPointsWithExactContract(t *testing.T) {
	server, rec := recordingServer(t, http.StatusOK, `{"id":"s1","title":"Login","story_points":8}`)
	client := tui.NewStoryPointsClient(server.URL)
	response, err := client.Set(t.Context(), "p1", "s1", 8)
	if err != nil {
		t.Fatal(err)
	}
	request := onlyRequest(t, rec)
	if request.Method != http.MethodPut || request.Path != "/projects/p1/stories/s1/story-points" {
		t.Fatalf("request = %s %s, want PUT /projects/p1/stories/s1/story-points", request.Method, request.Path)
	}
	var body map[string]any
	if err := json.Unmarshal(request.Body, &body); err != nil {
		t.Fatal(err)
	}
	if len(body) != 1 || body["story_points"] != float64(8) {
		t.Errorf("body = %#v, want only story_points: 8", body)
	}
	if response.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", response.StatusCode)
	}
}

func TestStoryPointsUpdateCanBeFollowedByBacklogRefresh(t *testing.T) {
	requests := []string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPut {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"s1","title":"Login","story_points":13}`))
			return
		}
		_, _ = w.Write([]byte(`{"stories":[{"id":"s1","title":"Login","story_points":13}]}`))
	}))
	defer server.Close()
	if _, err := tui.NewStoryPointsClient(server.URL).Set(t.Context(), "p1", "s1", 13); err != nil {
		t.Fatal(err)
	}
	if _, err := tui.NewBacklogClient(server.URL).List(t.Context(), "p1"); err != nil {
		t.Fatal(err)
	}
	want := []string{"PUT /projects/p1/stories/s1/story-points", "GET /projects/p1/stories"}
	if !reflect.DeepEqual(requests, want) {
		t.Fatalf("requests = %v, want %v", requests, want)
	}
}
