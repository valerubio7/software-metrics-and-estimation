package domain_test

import (
	"reflect"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/story/domain"
)

const backlogProjectID = "0f8fad5b-d9cb-469f-a165-70867728950e"

// storyWith builds the minimal story the backlog needs to be ordered.
func storyWith(id, priority string) domain.Story {
	return domain.Story{ID: id, ProjectID: backlogProjectID, Priority: priority}
}

func storyIDs(stories []domain.Story) []string {
	ids := make([]string, 0, len(stories))
	for _, story := range stories {
		ids = append(ids, story.ID)
	}
	return ids
}

func TestNewBacklogOrdersByPriority(t *testing.T) {
	tests := []struct {
		name  string
		input []domain.Story
		want  []string
	}{
		{
			name:  "high before medium before low",
			input: []domain.Story{storyWith("low", "baja"), storyWith("high", "alta"), storyWith("medium", "media")},
			want:  []string{"high", "medium", "low"},
		},
		{
			name: "spec scenario S1 to S5 breaks ties by creation order",
			input: []domain.Story{
				storyWith("S1", "media"), storyWith("S2", "alta"), storyWith("S3", "media"),
				storyWith("S4", "baja"), storyWith("S5", "alta"),
			},
			want: []string{"S2", "S5", "S1", "S3", "S4"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backlog := domain.NewBacklog(backlogProjectID, test.input)

			if got := storyIDs(backlog.Stories); !reflect.DeepEqual(got, test.want) {
				t.Errorf("order = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewBacklogTriangulatesSpecScenarios(t *testing.T) {
	tests := []struct {
		name  string
		input []domain.Story
		want  []string
	}{
		{
			name:  "same priority keeps creation order",
			input: []domain.Story{storyWith("S1", "media"), storyWith("S2", "media"), storyWith("S3", "media")},
			want:  []string{"S1", "S2", "S3"},
		},
		{
			name:  "raising a priority relocates the story",
			input: []domain.Story{storyWith("S1", "media"), storyWith("S2", "alta"), storyWith("S3", "alta")},
			want:  []string{"S2", "S3", "S1"},
		},
		{
			name: "unknown priority goes last keeping its input order",
			input: []domain.Story{
				storyWith("X1", "urgente"), storyWith("L", "baja"), storyWith("X2", "critica"), storyWith("H", "alta"),
			},
			want: []string{"H", "L", "X1", "X2"},
		},
		{
			name:  "single story",
			input: []domain.Story{storyWith("only", "baja")},
			want:  []string{"only"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backlog := domain.NewBacklog(backlogProjectID, test.input)

			if got := storyIDs(backlog.Stories); !reflect.DeepEqual(got, test.want) {
				t.Errorf("order = %v, want %v", got, test.want)
			}
		})
	}
}

func TestNewBacklogIsStableForEqualPriorities(t *testing.T) {
	// Identifiers deliberately sort against the input order, so an unstable sort
	// or an ordering by id would change the result.
	input := []domain.Story{storyWith("c", "media"), storyWith("a", "media"), storyWith("b", "media")}

	backlog := domain.NewBacklog(backlogProjectID, input)

	if got, want := storyIDs(backlog.Stories), []string{"c", "a", "b"}; !reflect.DeepEqual(got, want) {
		t.Errorf("order = %v, want the input order %v", got, want)
	}
}

func TestNewBacklogDoesNotMutateInput(t *testing.T) {
	input := []domain.Story{storyWith("low", "baja"), storyWith("high", "alta"), storyWith("medium", "media")}

	backlog := domain.NewBacklog(backlogProjectID, input)

	if got, want := storyIDs(input), []string{"low", "high", "medium"}; !reflect.DeepEqual(got, want) {
		t.Errorf("input order = %v, want it untouched %v", got, want)
	}
	backlog.Stories[0].ID = "changed"
	if input[0].ID == "changed" || input[1].ID == "changed" {
		t.Errorf("backlog shares its backing array with the input: %v", storyIDs(input))
	}
}

func TestNewBacklogPreservesProjectID(t *testing.T) {
	backlog := domain.NewBacklog(backlogProjectID, []domain.Story{storyWith("only", "alta")})

	if backlog.ProjectID != backlogProjectID {
		t.Errorf("ProjectID = %q, want %q", backlog.ProjectID, backlogProjectID)
	}
}

func TestNewBacklogHandlesNilAndEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input []domain.Story
	}{
		{name: "nil", input: nil},
		{name: "empty", input: []domain.Story{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			backlog := domain.NewBacklog(backlogProjectID, test.input)

			if len(backlog.Stories) != 0 {
				t.Errorf("Stories = %v, want none", storyIDs(backlog.Stories))
			}
			if backlog.ProjectID != backlogProjectID {
				t.Errorf("ProjectID = %q, want %q", backlog.ProjectID, backlogProjectID)
			}
		})
	}
}

func TestAllowedPrioritiesReturnsOrderedCopy(t *testing.T) {
	first := domain.AllowedPriorities()
	if want := []string{"alta", "media", "baja"}; !reflect.DeepEqual(first, want) {
		t.Fatalf("AllowedPriorities() = %v, want %v", first, want)
	}

	first[0] = "changed"

	if second, want := domain.AllowedPriorities(), []string{"alta", "media", "baja"}; !reflect.DeepEqual(second, want) {
		t.Errorf("second call = %v, want %v after mutating the first result", second, want)
	}
}
