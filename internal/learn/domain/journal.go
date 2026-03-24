package domain

// ValidEntryTypes defines the valid journal entry types. [06-learn §8.1.4]
var ValidEntryTypes = map[string]bool{
	"freeform":   true,
	"narration":  true,
	"reflection": true,
}

// ValidateEntryType returns an error if the entry type is invalid.
func ValidateEntryType(entryType string) error {
	if !ValidEntryTypes[entryType] {
		return &ErrInvalidEntryType{EntryType: entryType}
	}
	return nil
}
