package domain

import "encoding/json"

// Sequence ordering modes.
const (
	SequenceStatusNotStarted = "not_started"
	SequenceStatusInProgress = "in_progress"
	SequenceStatusCompleted  = "completed"
)

// ItemCompletion tracks completion of a single sequence item.
type ItemCompletion struct {
	ItemID      string  `json:"item_id"`
	CompletedAt string  `json:"completed_at"`
	Score       *string `json:"score,omitempty"`
	Skipped     bool    `json:"skipped,omitempty"`
}

// ParseItemCompletions deserialises the JSONB item_completions column.
func ParseItemCompletions(raw json.RawMessage) ([]ItemCompletion, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var completions []ItemCompletion
	if err := json.Unmarshal(raw, &completions); err != nil {
		return nil, err
	}
	return completions, nil
}
