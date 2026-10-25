package domain_test

import (
	"errors"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/valerubio7/software-metrics-and-estimation/internal/defect/domain"
)

const (
	validID        = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
	validProjectID = "82d38423-f02d-4259-9e35-4a291585bb1e"
	validStoryID   = "e99c05a4-03ea-4c19-8f59-286dd59e7aa1"
	validSprintID  = "b2a6a455-06e2-41ee-b011-7462c71375a0"
)

func validDefectParams() domain.NewDefectParams {
	return domain.NewDefectParams{
		ID:          validID,
		ProjectID:   validProjectID,
		StoryID:     validStoryID,
		SprintID:    validSprintID,
		Description: "El login acepta contraseñas vacías",
		Severity:    "alta",
		Status:      "detectado",
	}
}

func TestNewDefectAcceptsCatalogValues(t *testing.T) {
	// Cada combinación de catálogo válida persiste sin transformar los valores.
	for _, severity := range domain.Severities {
		for _, status := range domain.InitialStatuses {
			params := validDefectParams()
			params.Severity = severity
			params.Status = status

			defect, err := domain.NewDefect(params)
			if err != nil {
				t.Fatalf("NewDefect(severity=%s, status=%s): %v", severity, status, err)
			}
			if defect.Severity != severity || defect.Status != status {
				t.Errorf("NewDefect persists severity=%q status=%q, got %q/%q", severity, status, defect.Severity, defect.Status)
			}
		}
	}
}

func TestNewDefectKeepsFieldsExactly(t *testing.T) {
	defect, err := domain.NewDefect(validDefectParams())
	if err != nil {
		t.Fatalf("NewDefect: %v", err)
	}
	want := domain.Defect{
		ID:          validID,
		ProjectID:   validProjectID,
		StoryID:     validStoryID,
		SprintID:    validSprintID,
		Description: "El login acepta contraseñas vacías",
		Severity:    "alta",
		Status:      "detectado",
	}
	if defect != want {
		t.Errorf("NewDefect = %+v, want %+v", defect, want)
	}
}

func TestNewDefectTrimsDescription(t *testing.T) {
	params := validDefectParams()
	params.Description = "  El login acepta contraseñas vacías  "

	defect, err := domain.NewDefect(params)
	if err != nil {
		t.Fatalf("NewDefect: %v", err)
	}
	if defect.Description != "El login acepta contraseñas vacías" {
		t.Errorf("description after trim = %q, want %q", defect.Description, "El login acepta contraseñas vacías")
	}
}

func TestNewDefectDescriptionBoundary(t *testing.T) {
	// 500 caracteres multi-byte exactos son válidos (contados como caracteres).
	params := validDefectParams()
	params.Description = strings.Repeat("ñ", 500)
	if _, err := domain.NewDefect(params); err != nil {
		t.Errorf("NewDefect with 500 multi-byte characters: %v", err)
	}

	params.Description = strings.Repeat("ñ", 501)
	validation := rejectedDefect(t, params)
	if got := validation.Fields["description"]; got != domain.DescriptionLengthMessage {
		t.Errorf("501 characters description field = %q, want %q", got, domain.DescriptionLengthMessage)
	}
	if runeCount := utf8.RuneCountInString(strings.Repeat("\"", 501)); runeCount != 501 {
		t.Errorf("sanity: rune count mismatch")
	}
}

func TestNewDefectDescriptionErrors(t *testing.T) {
	cases := map[string]struct {
		description string
		want        string
	}{
		"missing":        {description: "", want: domain.RequiredMessage},
		"only_spaces":    {description: "   ", want: domain.RequiredMessage},
		"nul_character":  {description: "El \x00 login", want: domain.NulMessage},
		"too_long_ascii": {description: strings.Repeat("a", 501), want: domain.DescriptionLengthMessage},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			params := validDefectParams()
			params.Description = tc.description
			validation := rejectedDefect(t, params)
			if got := validation.Fields["description"]; got != tc.want {
				t.Errorf("description field = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestNewDefectSeverityErrors(t *testing.T) {
	cases := map[string]string{
		"empty":    "",
		"capital":  "Alta",
		"spaces":   " alta",
		"unknown":  "grave",
		"critical": "crítica",
		"numeric":  "1",
	}
	for name, severity := range cases {
		t.Run(name, func(t *testing.T) {
			params := validDefectParams()
			params.Severity = severity
			validation := rejectedDefect(t, params)
			want := domain.SeverityMessage
			if severity == "" {
				want = domain.RequiredMessage
			}
			if got := validation.Fields["severity"]; got != want {
				t.Errorf("severity = %q, want %q", got, want)
			}
		})
	}
}

func TestNewDefectStatusErrors(t *testing.T) {
	cases := map[string]string{
		"resuelto_rejected": "resuelto",
		"capital":           "Detectado",
		"spaces":            " detectado",
		"unknown":           "abierto",
	}
	for name, status := range cases {
		t.Run(name, func(t *testing.T) {
			params := validDefectParams()
			params.Status = status
			validation := rejectedDefect(t, params)

			want := domain.StatusMessage
			if status == "resuelto" {
				// El estado resuelto requiere un Sprint de resolución (US-17); el alta lo informa distinto.
				want = domain.InitialStatusMessage
			}
			if got := validation.Fields["status"]; got != want {
				t.Errorf("status = %q, want %q", got, want)
			}
		})
	}
}

func TestNewDefectAccumulatesAllFieldErrors(t *testing.T) {
	params := validDefectParams()
	params.Description = ""
	params.Severity = "grave"
	params.Status = "resuelto"

	validation := rejectedDefect(t, params)
	if len(validation.Fields) != 3 {
		t.Errorf("accumulated fields = %d (%v), want 3", len(validation.Fields), validation.Fields)
	}
	for _, field := range []string{"description", "severity", "status"} {
		if _, ok := validation.Fields[field]; !ok {
			t.Errorf("missing field error %q in %v", field, validation.Fields)
		}
	}
}

func TestNewDefectMessageConstantsAreStable(t *testing.T) {
	// Los mensajes viajan en la respuesta del API; se congelan para detectar cambios accidentales.
	if domain.RequiredMessage != "is required" ||
		domain.NulMessage != "must not contain NUL characters" ||
		domain.DescriptionLengthMessage != "must be at most 500 characters" ||
		domain.SeverityMessage != "must be one of alta, media or baja" ||
		domain.StatusMessage != "must be one of detectado, en_correccion or resuelto" ||
		domain.InitialStatusMessage != "must be detectado or en_correccion when registering a defect" {
		t.Errorf("domain messages drifted: %q %q %q %q %q %q",
			domain.RequiredMessage, domain.NulMessage, domain.DescriptionLengthMessage,
			domain.SeverityMessage, domain.StatusMessage, domain.InitialStatusMessage)
	}
	if domain.MaxDescriptionLength != 500 {
		t.Errorf("MaxDescriptionLength = %d, want 500", domain.MaxDescriptionLength)
	}
}

// rejectedDefect asserts the domain returned *ValidationError and hands it to the caller.
func rejectedDefect(t *testing.T, params domain.NewDefectParams) *domain.ValidationError {
	t.Helper()
	defect, err := domain.NewDefect(params)
	if err == nil {
		t.Fatalf("NewDefect(%+v) = %+v, want validation error", params, defect)
	}
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("NewDefect error is not *ValidationError: %T (%v)", err, err)
	}
	return validation
}
