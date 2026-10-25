package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/application"
	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const (
	projectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	storyID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	sprintID  = "b2a6a455-06e2-41ee-b011-7462c71375a0"
)

// stubRepository records the received defect and returns a preconfigured error so the
// tests can emulate the repository outcomes (reference checks and storage failures).
type stubRepository struct {
	received *domain.Defect
	err      error
}

func (s *stubRepository) Create(_ context.Context, defect domain.Defect) error {
	s.received = &defect
	return s.err
}

// countingGenerator hands out a fixed progression of IDs so the tests can prove the server
// generates them only on success.
type countingGenerator struct {
	count int
}

func (g *countingGenerator) next() string {
	g.count++
	return "00000000-0000-0000-0000-" + padID(g.count)
}

func padID(count int) string {
	return map[int]string{
		1: "f47ac10b58cc4372a5670e02b2c3d479",
		2: "f47ac10b58cc4372a5670e02b2c3d480",
		3: "f47ac10b58cc4372a5670e02b2c3d481",
	}[count]
}

func validCommand() application.RegisterDefectCommand {
	return application.RegisterDefectCommand{
		ProjectID:   projectID,
		StoryID:     storyID,
		SprintID:    sprintID,
		Description: "  El login acepta contraseñas vacías  ",
		Severity:    "alta",
		Status:      "detectado",
	}
}

func newUseCaseCase(t *testing.T) (*application.RegisterDefectUseCase, *stubRepository, *countingGenerator) {
	t.Helper()
	repository := &stubRepository{}
	generator := &countingGenerator{}
	useCase := application.NewRegisterDefectUseCase(repository, generator.next)
	return useCase, repository, generator
}

func TestRegisterDefectPersistsValidDefect(t *testing.T) {
	useCase, repository, _ := newUseCaseCase(t)

	defect, err := useCase.Execute(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repository.received == nil {
		t.Fatal("repository did not receive the defect")
	}
	want := domain.Defect{
		ID:          "00000000-0000-0000-0000-f47ac10b58cc4372a5670e02b2c3d479",
		ProjectID:   projectID,
		StoryID:     storyID,
		SprintID:    sprintID,
		Description: "El login acepta contraseñas vacías",
		Severity:    "alta",
		Status:      "detectado",
	}
	if *repository.received != want {
		t.Errorf("repository received %+v, want %+v", *repository.received, want)
	}
	if *repository.received != defect {
		t.Errorf("returned defect %+v differs from persisted %+v", defect, *repository.received)
	}
}

func TestRegisterDefectNormalizesUUIDs(t *testing.T) {
	useCase, repository, _ := newUseCaseCase(t)

	command := validCommand()
	command.StoryID = "E99C05A4-03EA-4C19-8F59-286DD59E7AA1"
	command.SprintID = "B2A6A455-06E2-41EE-B011-7462C71375A0"

	if _, err := useCase.Execute(context.Background(), command); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if repository.received.StoryID != storyID || repository.received.SprintID != sprintID {
		t.Errorf("normalized ids = (%s, %s), want (%s, %s)",
			repository.received.StoryID, repository.received.SprintID, storyID, sprintID)
	}
}

func TestRegisterDefectFieldErrorsAccumulate(t *testing.T) {
	useCase, repository, generator := newUseCaseCase(t)

	command := validCommand()
	command.StoryID = "no-es-un-uuid"
	command.SprintID = ""
	command.Description = ""
	command.Severity = "grave"
	command.Status = "resuelto"

	_, err := useCase.Execute(context.Background(), command)
	if err == nil {
		t.Fatal("Execute: want validation error")
	}
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Execute error is not *ValidationError: %T (%v)", err, err)
	}
	want := map[string]string{
		"story_id":    "must be a valid UUID",
		"sprint_id":   "is required",
		"description": "is required",
		"severity":    domain.SeverityMessage,
		"status":      domain.InitialStatusMessage,
	}
	if len(validation.Fields) != len(want) {
		t.Errorf("fields = %v, want %v", validation.Fields, want)
	}
	for field, message := range want {
		if got := validation.Fields[field]; got != message {
			t.Errorf("field %q = %q, want %q", field, got, message)
		}
	}
	if repository.received != nil {
		t.Error("repository received a defect despite validation errors")
	}
	if generator.count != 0 {
		t.Errorf("identifier generated %d times before persistence, want 0", generator.count)
	}
}

func TestRegisterDefectEmptyIDsReportRequired(t *testing.T) {
	useCase, _, _ := newUseCaseCase(t)

	command := validCommand()
	command.StoryID = ""
	command.SprintID = ""

	_, err := useCase.Execute(context.Background(), command)
	if err == nil {
		t.Fatal("Execute: want validation error")
	}
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("Execute error is not *ValidationError: %T (%v)", err, err)
	}
	for _, field := range []string{"story_id", "sprint_id"} {
		if got := validation.Fields[field]; got != "is required" {
			t.Errorf("field %q = %q, want %q", field, got, "is required")
		}
	}
}

func TestRegisterDefectReferenceErrorsPassThrough(t *testing.T) {
	cases := map[string]error{
		"project_not_found": application.ErrProjectNotFound,
		"story_not_found":   application.ErrStoryNotFound,
		"sprint_not_found":  application.ErrSprintNotFound,
		"unexpected":        errors.New("connection reset"),
	}
	for name, wantErr := range cases {
		t.Run(name, func(t *testing.T) {
			repository := &stubRepository{err: wantErr}
			generator := &countingGenerator{}
			useCase := application.NewRegisterDefectUseCase(repository, generator.next)

			_, err := useCase.Execute(context.Background(), validCommand())
			if !errors.Is(err, wantErr) {
				t.Errorf("Execute error = %v, want %v (sentinel passthrough, no wrapping)", err, wantErr)
			}
		})
	}
}

func TestRegisterDefectGeneratesIDOnSuccessOnly(t *testing.T) {
	useCase, _, generator := newUseCaseCase(t)

	if _, err := useCase.Execute(context.Background(), validCommand()); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if generator.count != 1 {
		t.Fatalf("identifier generated %d times on success, want 1", generator.count)
	}

	// Con errores de campo el identificador no se genera.
	invalid := validCommand()
	invalid.Description = ""
	_, err := useCase.Execute(context.Background(), invalid)
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("second Execute error is not a ValidationError: %v", err)
	}
	if generator.count != 1 {
		t.Errorf("identifier generated %d times after field errors, want 1", generator.count)
	}

	// Un intento que alcanza la persistencia consume un identificador aunque falle el almacenamiento.
	failingRepository := &stubRepository{err: application.ErrStoryNotFound}
	failingUseCase := application.NewRegisterDefectUseCase(failingRepository, generator.next)
	if _, err := failingUseCase.Execute(context.Background(), validCommand()); !errors.Is(err, application.ErrStoryNotFound) {
		t.Fatalf("third Execute error = %v, want ErrStoryNotFound", err)
	}
	if generator.count != 2 {
		t.Errorf("identifier generated %d times after failed persistence, want 2", generator.count)
	}
}
