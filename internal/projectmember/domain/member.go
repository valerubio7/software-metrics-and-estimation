package domain

import (
	"fmt"
	"net/mail"
	"strings"
)

// Member is an individual associated with a project.
type Member struct {
	ID        string  `json:"id"`
	ProjectID string  `json:"project_id"`
	FullName  string  `json:"full_name"`
	Email     *string `json:"email"`
}

// ValidationError describes invalid member input.
type ValidationError struct{ Field, Message string }

func (e *ValidationError) Error() string { return fmt.Sprintf("%s: %s", e.Field, e.Message) }

// NewMember validates input while preserving the exact supplied values.
func NewMember(id, projectID, fullName, email string) (Member, error) {
	if strings.TrimSpace(fullName) == "" {
		return Member{}, &ValidationError{Field: "full_name", Message: "is required"}
	}
	var optionalEmail *string
	if email != "" {
		address, err := mail.ParseAddress(email)
		if err != nil || address.Address != email || !strings.Contains(email, "@") {
			return Member{}, &ValidationError{Field: "email", Message: "must be a valid email address"}
		}
		optionalEmail = &email
	}
	return Member{ID: id, ProjectID: projectID, FullName: fullName, Email: optionalEmail}, nil
}

// SameIdentity compares the exact identity pair; nil email equals nil email.
func (m Member) SameIdentity(other Member) bool {
	if m.FullName != other.FullName {
		return false
	}
	if m.Email == nil || other.Email == nil {
		return m.Email == nil && other.Email == nil
	}
	return *m.Email == *other.Email
}
