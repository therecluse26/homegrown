package recs

import "github.com/google/uuid"

// ═══════════════════════════════════════════════════════════════════════════════
// Domain Events [13-recs §12]
// ═══════════════════════════════════════════════════════════════════════════════

// RecommendationsGenerated is published after the compute_recommendations task
// creates new recommendations for a family. Used by notify:: to inform the user.
type RecommendationsGenerated struct {
	FamilyID uuid.UUID
	Count    int64 // number of new recommendations created
}

func (RecommendationsGenerated) EventName() string { return "recs.recommendations_generated" }
