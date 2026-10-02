package domain_test

import (
	"errors"
	"testing"

	"github.com/valerubio7/software-metrics-and-estimation/internal/projectmember/domain"
)

func TestNewMemberRequiresNonBlankNameAndValidOptionalEmail(t *testing.T) {
	for _, tc := range []struct {
		name, email string
		wantErr     bool
	}{
		{"", "", true}, {"  \t", "", true}, {"Ada Lovelace", "", false}, {"Ada Lovelace", "ada@example.com", false}, {"Ada", "not-an-email", true},
	} {
		_, err := domain.NewMember("id", "project", tc.name, tc.email)
		if (err != nil) != tc.wantErr {
			t.Errorf("NewMember(%q, %q) error = %v", tc.name, tc.email, err)
		}
	}
}

func TestDuplicateIdentityUsesExactNameAndNullableEmail(t *testing.T) {
	a, _ := domain.NewMember("1", "p", "Ada", "")
	b, _ := domain.NewMember("2", "p", "Ada", "")
	if !a.SameIdentity(b) {
		t.Fatal("identical name and null email should compare equal")
	}
	c, _ := domain.NewMember("3", "p", "ada", "")
	d, _ := domain.NewMember("4", "p", "Ada", "ada@example.com")
	if a.SameIdentity(c) || a.SameIdentity(d) {
		t.Fatal("identity comparison must be exact")
	}
}

func TestValidationErrorIsTyped(t *testing.T) {
	_, err := domain.NewMember("id", "project", " ", "")
	var validation *domain.ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}
