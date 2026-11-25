package incidents

import "fmt"

// ValidTransitions defines the explicit Incident Lifecycle state machine.
var ValidTransitions = map[string][]string{
	"OPEN":          {"INVESTIGATING", "ACKNOWLEDGED", "RESOLVED"},
	"INVESTIGATING": {"ACKNOWLEDGED", "RESOLVED"},
	"ACKNOWLEDGED":  {"INVESTIGATING", "RESOLVED"},
	"RESOLVED":      {"CLOSED", "OPEN"},
	"CLOSED":        {}, // Terminal state
}

// CanTransition returns true if transitioning from `from` to `to` is permitted.
func CanTransition(from, to string) bool {
	if from == to {
		return true
	}
	allowed, ok := ValidTransitions[from]
	if !ok {
		return false
	}
	for _, s := range allowed {
		if s == to {
			return true
		}
	}
	return false
}

// ValidateTransition returns an error if the transition is not permitted.
func ValidateTransition(from, to string) error {
	if !CanTransition(from, to) {
		return fmt.Errorf("cannot transition incident from %s to %s", from, to)
	}
	return nil
}
