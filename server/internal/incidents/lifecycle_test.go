package incidents

import "testing"

func TestIncidentLifecycleTransitions(t *testing.T) {
	validCases := []struct {
		from string
		to   string
	}{
		{"OPEN", "INVESTIGATING"},
		{"OPEN", "ACKNOWLEDGED"},
		{"OPEN", "RESOLVED"},
		{"INVESTIGATING", "ACKNOWLEDGED"},
		{"INVESTIGATING", "RESOLVED"},
		{"ACKNOWLEDGED", "INVESTIGATING"},
		{"ACKNOWLEDGED", "RESOLVED"},
		{"RESOLVED", "CLOSED"},
		{"RESOLVED", "OPEN"},
		{"OPEN", "OPEN"}, // idempotent
	}

	for _, tc := range validCases {
		if !CanTransition(tc.from, tc.to) {
			t.Errorf("expected transition %s -> %s to be valid", tc.from, tc.to)
		}
	}

	invalidCases := []struct {
		from string
		to   string
	}{
		{"OPEN", "CLOSED"},
		{"INVESTIGATING", "CLOSED"},
		{"ACKNOWLEDGED", "CLOSED"},
		{"CLOSED", "OPEN"},
		{"CLOSED", "RESOLVED"},
		{"CLOSED", "INVESTIGATING"},
		{"UNKNOWN", "OPEN"},
	}

	for _, tc := range invalidCases {
		if CanTransition(tc.from, tc.to) {
			t.Errorf("expected transition %s -> %s to be invalid", tc.from, tc.to)
		}
	}
}
