package application_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/effort/domain"
)

const (
	projectID = "11111111-1111-4111-8111-111111111111"
	storyID   = "22222222-2222-4222-8222-222222222222"
	memberID  = "33333333-3333-4333-8333-333333333333"
	taskID    = "44444444-4444-4444-8444-444444444444"
)

// fakeEffortRepository captures the record it receives and returns a configurable
// error, standing in for application.EffortRepository.
type fakeEffortRepository struct {
	calls  int
	record domain.EffortRecord
	err    error
}

func (r *fakeEffortRepository) Create(_ context.Context, record domain.EffortRecord) error {
	r.calls++
	r.record = record
	return r.err
}

func counterIDGenerator() (application.IDGenerator, *int) {
	calls := 0
	return func() string {
		calls++
		return "generated-id-" + strconv.Itoa(calls)
	}, &calls
}

func fixedClock(instant string) application.Clock {
	parsed, err := time.Parse(time.RFC3339, instant)
	if err != nil {
		panic(err)
	}
	return func() time.Time { return parsed }
}

func stringPtr(value string) *string { return &value }

func floatPtr(value float64) *float64 { return &value }

// validCommand returns a command that registers effort over a story on 2026-03-10.
func validCommand() application.RegisterEffortCommand {
	return application.RegisterEffortCommand{
		ProjectID:   projectID,
		StoryID:     storyID,
		MemberID:    memberID,
		WorkDate:    "2026-03-10",
		Description: "Implementar login",
		Hours:       floatPtr(3.5),
	}
}

// executeFixed runs a use case whose clock is fixed at 2026-03-10T12:00:00Z.
func executeFixed(t *testing.T, command application.RegisterEffortCommand) outcome {
	t.Helper()
	return executeAt(t, "2026-03-10T12:00:00Z", command)
}

// outcome gathers the result of one Execute call together with the observed collaborators.
type outcome struct {
	record         domain.EffortRecord
	err            error
	repo           *fakeEffortRepository
	generatorCalls int
}

func executeAt(t *testing.T, instant string, command application.RegisterEffortCommand) outcome {
	t.Helper()
	repo := &fakeEffortRepository{}
	generateID, calls := counterIDGenerator()
	useCase := application.NewRegisterEffortUseCase(repo, generateID, fixedClock(instant))
	record, err := useCase.Execute(context.Background(), command)
	return outcome{record: record, err: err, repo: repo, generatorCalls: *calls}
}

func requireFields(t *testing.T, err error) map[string]string {
	t.Helper()
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Execute() error = %v, want *domain.ValidationError", err)
	}
	return validation.Fields
}

func TestRegisterEffortUseCaseExecute(t *testing.T) {
	t.Run("success over a story persists one record with a generated ID and a trimmed description", func(t *testing.T) {
		command := validCommand()
		command.Description = "  Implementar login  "

		got := executeFixed(t, command)
		record, err, repo, generatorCalls := got.record, got.err, got.repo, got.generatorCalls

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if repo.calls != 1 {
			t.Fatalf("repository calls = %d, want 1", repo.calls)
		}
		if generatorCalls != 1 {
			t.Fatalf("generator calls = %d, want 1", generatorCalls)
		}
		if record.ID != "generated-id-1" {
			t.Errorf("record.ID = %q, want %q", record.ID, "generated-id-1")
		}
		if record.ProjectID != projectID || record.StoryID != storyID || record.MemberID != memberID {
			t.Errorf("record IDs = (%s,%s,%s), want (%s,%s,%s)", record.ProjectID, record.StoryID, record.MemberID, projectID, storyID, memberID)
		}
		if record.TaskID != nil {
			t.Errorf("record.TaskID = %q, want nil for effort over the story", *record.TaskID)
		}
		if want := time.Date(2026, time.March, 10, 0, 0, 0, 0, time.UTC); !record.WorkDate.Equal(want) {
			t.Errorf("record.WorkDate = %v, want %v", record.WorkDate, want)
		}
		if record.Description != "Implementar login" {
			t.Errorf("record.Description = %q, want trimmed %q", record.Description, "Implementar login")
		}
		if record.Hours != 3.5 {
			t.Errorf("record.Hours = %v, want 3.5", record.Hours)
		}
		if repo.record.ID != record.ID || repo.record.Description != record.Description {
			t.Errorf("repository received %#v, want the same record that was returned %#v", repo.record, record)
		}
	})

	t.Run("success over a task keeps the task ID", func(t *testing.T) {
		command := validCommand()
		command.TaskID = stringPtr(taskID)

		got := executeFixed(t, command)
		record, err, repo := got.record, got.err, got.repo

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if repo.calls != 1 {
			t.Fatalf("repository calls = %d, want 1", repo.calls)
		}
		if record.TaskID == nil || *record.TaskID != taskID {
			t.Fatalf("record.TaskID = %v, want %q", record.TaskID, taskID)
		}
		if repo.record.TaskID == nil || *repo.record.TaskID != taskID {
			t.Errorf("repository TaskID = %v, want %q", repo.record.TaskID, taskID)
		}
	})

	t.Run("uppercase member and task IDs are normalized to canonical lowercase", func(t *testing.T) {
		command := validCommand()
		command.MemberID = "33333333-3333-4333-8333-33333333ABCD"
		command.TaskID = stringPtr("44444444-4444-4444-8444-44444444ABCD")

		got := executeFixed(t, command)
		record, err, repo := got.record, got.err, got.repo

		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		if record.MemberID != "33333333-3333-4333-8333-33333333abcd" {
			t.Errorf("record.MemberID = %q, want lowercase canonical form", record.MemberID)
		}
		if record.TaskID == nil || *record.TaskID != "44444444-4444-4444-8444-44444444abcd" {
			t.Errorf("record.TaskID = %v, want lowercase canonical form", record.TaskID)
		}
		if repo.record.MemberID != record.MemberID {
			t.Errorf("repository MemberID = %q, want %q", repo.record.MemberID, record.MemberID)
		}
	})

	validationCases := []struct {
		name   string
		mutate func(*application.RegisterEffortCommand)
		want   map[string]string
	}{
		{
			name:   "empty member_id is required",
			mutate: func(c *application.RegisterEffortCommand) { c.MemberID = "" },
			want:   map[string]string{"member_id": "is required"},
		},
		{
			name:   "non UUID member_id is rejected",
			mutate: func(c *application.RegisterEffortCommand) { c.MemberID = "abc" },
			want:   map[string]string{"member_id": "must be a valid UUID"},
		},
		{
			name:   "empty task_id is rejected as an invalid UUID",
			mutate: func(c *application.RegisterEffortCommand) { c.TaskID = stringPtr("") },
			want:   map[string]string{"task_id": "must be a valid UUID"},
		},
		{
			name:   "non UUID task_id is rejected",
			mutate: func(c *application.RegisterEffortCommand) { c.TaskID = stringPtr("not-a-uuid") },
			want:   map[string]string{"task_id": "must be a valid UUID"},
		},
		{
			name:   "empty work_date is required",
			mutate: func(c *application.RegisterEffortCommand) { c.WorkDate = "" },
			want:   map[string]string{"work_date": "is required"},
		},
		{
			name:   "impossible calendar date is rejected without a future error",
			mutate: func(c *application.RegisterEffortCommand) { c.WorkDate = "2026-02-30" },
			want:   map[string]string{"work_date": "must be a valid YYYY-MM-DD date"},
		},
		{
			name:   "date without zero padding is rejected",
			mutate: func(c *application.RegisterEffortCommand) { c.WorkDate = "2026-3-1" },
			want:   map[string]string{"work_date": "must be a valid YYYY-MM-DD date"},
		},
		{
			name:   "day first format is rejected",
			mutate: func(c *application.RegisterEffortCommand) { c.WorkDate = "10/03/2026" },
			want:   map[string]string{"work_date": "must be a valid YYYY-MM-DD date"},
		},
		{
			name:   "timestamp instead of a date is rejected",
			mutate: func(c *application.RegisterEffortCommand) { c.WorkDate = "2026-03-10T00:00:00Z" },
			want:   map[string]string{"work_date": "must be a valid YYYY-MM-DD date"},
		},
		{
			name:   "future work_date is rejected by the domain",
			mutate: func(c *application.RegisterEffortCommand) { c.WorkDate = "2026-03-11" },
			want:   map[string]string{"work_date": "must not be in the future"},
		},
		{
			name: "application and domain errors accumulate in a single validation error",
			mutate: func(c *application.RegisterEffortCommand) {
				c.Hours = floatPtr(0)
				c.Description = "   "
				c.WorkDate = "2026-02-30"
			},
			want: map[string]string{
				"hours":       "must be greater than 0, at most 24 and have at most 2 decimals",
				"description": "is required",
				"work_date":   "must be a valid YYYY-MM-DD date",
			},
		},
		{
			name: "invalid body IDs and domain fields are reported together",
			mutate: func(c *application.RegisterEffortCommand) {
				c.MemberID = ""
				c.TaskID = stringPtr("abc")
				c.Hours = nil
			},
			want: map[string]string{
				"member_id": "is required",
				"task_id":   "must be a valid UUID",
				"hours":     "is required",
			},
		},
	}
	for _, tt := range validationCases {
		t.Run(tt.name, func(t *testing.T) {
			command := validCommand()
			tt.mutate(&command)

			got := executeFixed(t, command)
			record, err, repo, generatorCalls := got.record, got.err, got.repo, got.generatorCalls

			fields := requireFields(t, err)
			if len(fields) != len(tt.want) {
				t.Errorf("Fields = %v, want exactly %v", fields, tt.want)
			}
			for field, message := range tt.want {
				if fields[field] != message {
					t.Errorf("Fields[%s] = %q, want %q", field, fields[field], message)
				}
			}
			if repo.calls != 0 || generatorCalls != 0 {
				t.Errorf("calls = (repo %d, generator %d), want (0, 0) on validation failure", repo.calls, generatorCalls)
			}
			if record.ID != "" || record.Description != "" {
				t.Errorf("record = %#v, want the zero record on validation failure", record)
			}
		})
	}

	repositoryErrors := []struct {
		name string
		err  error
	}{
		{"project not found", application.ErrProjectNotFound},
		{"story not found", application.ErrStoryNotFound},
		{"member not found", application.ErrMemberNotFound},
		{"task not found", application.ErrTaskNotFound},
		{"generic repository failure", errors.New("connection reset by peer")},
	}
	for _, tt := range repositoryErrors {
		t.Run("repository error propagates unwrapped: "+tt.name, func(t *testing.T) {
			repo := &fakeEffortRepository{err: tt.err}
			generateID, _ := counterIDGenerator()
			useCase := application.NewRegisterEffortUseCase(repo, generateID, fixedClock("2026-03-10T12:00:00Z"))

			record, err := useCase.Execute(context.Background(), validCommand())

			if err != tt.err {
				t.Fatalf("Execute() error = %v, want the repository error unchanged (%v)", err, tt.err)
			}
			if !errors.Is(err, tt.err) {
				t.Errorf("errors.Is(err, %v) = false, want true", tt.err)
			}
			if repo.calls != 1 {
				t.Errorf("repository calls = %d, want 1", repo.calls)
			}
			if record.ID != "" || record.MemberID != "" {
				t.Errorf("record = %#v, want the zero record when the repository fails", record)
			}
		})
	}
}

func TestRegisterEffortUseCaseUTCClockBoundaries(t *testing.T) {
	tests := []struct {
		name     string
		clock    string
		workDate string
		wantErr  bool
	}{
		{"local evening at UTC-3 is already the next UTC day: that day is accepted", "2026-10-24T22:30:00-03:00", "2026-10-25", false},
		{"local evening at UTC-3: the day after the UTC date is rejected", "2026-10-24T22:30:00-03:00", "2026-10-26", true},
		{"local evening at UTC-3: the local date is still accepted", "2026-10-24T22:30:00-03:00", "2026-10-24", false},
		{"last second of the UTC day accepts that day", "2026-10-24T23:59:59Z", "2026-10-24", false},
		{"last second of the UTC day rejects the next day", "2026-10-24T23:59:59Z", "2026-10-25", true},
		{"first second of the UTC day rejects the next day", "2026-10-25T00:00:00Z", "2026-10-26", true},
		{"first second of the UTC day accepts that day", "2026-10-25T00:00:00Z", "2026-10-25", false},
		{"past date is accepted", "2026-03-10T08:00:00Z", "2026-01-05", false},
		{"comparison uses the UTC date, not the local one", "2026-03-10T23:30:00-05:00", "2026-03-11", false},
		{"comparison uses the UTC date when local time is ahead of UTC", "2026-03-11T01:30:00+05:00", "2026-03-11", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := validCommand()
			command.WorkDate = tt.workDate

			got := executeAt(t, tt.clock, command)

			if tt.wantErr {
				fields := requireFields(t, got.err)
				if fields["work_date"] != "must not be in the future" {
					t.Errorf("Fields[work_date] = %q, want %q", fields["work_date"], "must not be in the future")
				}
				if got.repo.calls != 0 || got.generatorCalls != 0 {
					t.Errorf("calls = (repo %d, generator %d), want (0, 0)", got.repo.calls, got.generatorCalls)
				}
				return
			}
			if got.err != nil {
				t.Fatalf("Execute() error = %v, want the date accepted", got.err)
			}
			if want := tt.workDate; got.record.WorkDate.Format("2006-01-02") != want {
				t.Errorf("record.WorkDate = %s, want %s", got.record.WorkDate.Format("2006-01-02"), want)
			}
			if got.repo.calls != 1 {
				t.Errorf("repository calls = %d, want 1", got.repo.calls)
			}
		})
	}
}

func TestRegisterEffortUseCaseNormalizesAlternativeUUIDSpellings(t *testing.T) {
	const canonicalMember = "33333333-3333-4333-8333-33333333abcd"
	spellings := map[string]string{
		"hyphenless":    "3333333333334333833333333333ABCD",
		"urn prefix":    "urn:uuid:33333333-3333-4333-8333-33333333ABCD",
		"braced":        "{33333333-3333-4333-8333-33333333abcd}",
		"already clean": canonicalMember,
	}
	for name, spelling := range spellings {
		t.Run(name, func(t *testing.T) {
			command := validCommand()
			command.MemberID = spelling

			got := executeFixed(t, command)

			if got.err != nil {
				t.Fatalf("Execute() error = %v", got.err)
			}
			if got.record.MemberID != canonicalMember {
				t.Errorf("record.MemberID = %q, want canonical %q", got.record.MemberID, canonicalMember)
			}
			if got.repo.record.MemberID != canonicalMember {
				t.Errorf("repository MemberID = %q, want canonical %q", got.repo.record.MemberID, canonicalMember)
			}
		})
	}
}

func TestRegisterEffortUseCaseAllowsExactDuplicates(t *testing.T) {
	repo := &fakeEffortRepository{}
	generateID, calls := counterIDGenerator()
	useCase := application.NewRegisterEffortUseCase(repo, generateID, fixedClock("2026-03-10T12:00:00Z"))
	command := validCommand()
	command.TaskID = stringPtr(taskID)

	first, err := useCase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("first Execute() error = %v", err)
	}
	second, err := useCase.Execute(context.Background(), command)
	if err != nil {
		t.Fatalf("second Execute() error = %v", err)
	}

	if repo.calls != 2 {
		t.Errorf("repository calls = %d, want 2 (duplicates are not detected)", repo.calls)
	}
	if *calls != 2 {
		t.Errorf("generator calls = %d, want 2", *calls)
	}
	if first.ID == second.ID {
		t.Errorf("both records got ID %q, want distinct IDs", first.ID)
	}
	if first.ID != "generated-id-1" || second.ID != "generated-id-2" {
		t.Errorf("IDs = (%q,%q), want (generated-id-1, generated-id-2)", first.ID, second.ID)
	}
}
