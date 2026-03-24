package domain

// Assignment statuses.
const (
	AssignmentStatusAssigned   = "assigned"
	AssignmentStatusInProgress = "in_progress"
	AssignmentStatusCompleted  = "completed"
	AssignmentStatusSkipped    = "skipped"
)

// assignmentTransitions defines valid status transitions.
// Key: from status → allowed next statuses.
var assignmentTransitions = map[string]map[string]bool{
	AssignmentStatusAssigned: {
		AssignmentStatusInProgress: true,
		AssignmentStatusSkipped:    true,
	},
	AssignmentStatusInProgress: {
		AssignmentStatusCompleted: true,
		AssignmentStatusSkipped:   true,
	},
	// completed and skipped are terminal — no outgoing transitions
}

// ValidateAssignmentStatusTransition checks if the transition is legal.
func ValidateAssignmentStatusTransition(from, to string) error {
	allowed, ok := assignmentTransitions[from]
	if !ok || !allowed[to] {
		return ErrInvalidAssignmentStatusTransition
	}
	return nil
}
