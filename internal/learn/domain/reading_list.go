package domain

// Reading status constants. [06-learn §8.1.3]
const (
	ReadingStatusToRead     = "to_read"
	ReadingStatusInProgress = "in_progress"
	ReadingStatusCompleted  = "completed"
)

// validReadingTransitions defines allowed status transitions.
// to_read → in_progress → completed
var validReadingTransitions = map[string]map[string]bool{
	ReadingStatusToRead:     {ReadingStatusInProgress: true},
	ReadingStatusInProgress: {ReadingStatusCompleted: true},
}

// ValidateReadingStatusTransition checks if a status transition is valid.
func ValidateReadingStatusTransition(from, to string) error {
	allowed, ok := validReadingTransitions[from]
	if !ok || !allowed[to] {
		return ErrInvalidReadingStatusTransition
	}
	return nil
}
