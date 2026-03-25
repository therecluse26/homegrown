package recs

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Signal Repository [13-recs §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgSignalRepository implements SignalRepository using GORM/PostgreSQL.
type PgSignalRepository struct {
	db *gorm.DB
}

// NewPgSignalRepository creates a new PgSignalRepository.
func NewPgSignalRepository(db *gorm.DB) *PgSignalRepository {
	return &PgSignalRepository{db: db}
}

func (r *PgSignalRepository) Create(ctx context.Context, s NewSignal) error {
	payload, err := json.Marshal(s.Payload)
	if err != nil {
		return fmt.Errorf("recs signal create: marshal payload: %w", err)
	}

	sql := `
		INSERT INTO recs_signals (family_id, student_id, signal_type, methodology_slug, payload, signal_date)
		VALUES (?, ?, ?, ?, ?, ?)`

	return r.db.WithContext(ctx).Exec(sql,
		s.FamilyID.UUID,
		s.StudentID,
		string(s.SignalType),
		s.MethodologySlug,
		payload,
		s.SignalDate.Format("2006-01-02"),
	).Error
}

func (r *PgSignalRepository) FindByFamily(ctx context.Context, scope *shared.FamilyScope, since time.Time) ([]Signal, error) {
	type row struct {
		ID              uuid.UUID       `gorm:"column:id"`
		FamilyID        uuid.UUID       `gorm:"column:family_id"`
		StudentID       *uuid.UUID      `gorm:"column:student_id"`
		SignalType      string          `gorm:"column:signal_type"`
		MethodologySlug string          `gorm:"column:methodology_slug"`
		Payload         json.RawMessage `gorm:"column:payload"`
		SignalDate      time.Time       `gorm:"column:signal_date"`
		CreatedAt       time.Time       `gorm:"column:created_at"`
	}

	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, signal_type, methodology_slug, payload, signal_date, created_at
		FROM recs_signals
		WHERE family_id = ? AND signal_date >= ?
		ORDER BY signal_date DESC`,
		scope.FamilyID(), since,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("recs signal find by family: %w", err)
	}

	out := make([]Signal, len(rows))
	for i, row := range rows {
		var p map[string]any
		_ = json.Unmarshal(row.Payload, &p)
		out[i] = Signal{
			ID:              row.ID,
			FamilyID:        shared.NewFamilyID(row.FamilyID),
			StudentID:       row.StudentID,
			SignalType:      SignalType(row.SignalType),
			MethodologySlug: row.MethodologySlug,
			Payload:         p,
			SignalDate:      row.SignalDate,
			CreatedAt:       row.CreatedAt,
		}
	}
	return out, nil
}

func (r *PgSignalRepository) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM recs_signals WHERE family_id = ?`,
		familyID.UUID,
	)
	return tx.RowsAffected, tx.Error
}

func (r *PgSignalRepository) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM recs_signals WHERE created_at < ?`,
		before,
	)
	return tx.RowsAffected, tx.Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Recommendation Repository [13-recs §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgRecommendationRepository implements RecommendationRepository using GORM/PostgreSQL.
type PgRecommendationRepository struct {
	db *gorm.DB
}

// NewPgRecommendationRepository creates a new PgRecommendationRepository.
func NewPgRecommendationRepository(db *gorm.DB) *PgRecommendationRepository {
	return &PgRecommendationRepository{db: db}
}

func (r *PgRecommendationRepository) CreateBatch(ctx context.Context, recs []NewRecommendation) (int64, error) {
	if len(recs) == 0 {
		return 0, nil
	}
	var count int64
	for _, rec := range recs {
		tx := r.db.WithContext(ctx).Exec(`
			INSERT INTO recs_recommendations
				(family_id, student_id, recommendation_type, target_entity_id, target_entity_label,
				 source_signal, source_label, score, status, expires_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'active', ?)
			ON CONFLICT DO NOTHING`,
			rec.FamilyID.UUID, rec.StudentID, string(rec.RecommendationType),
			rec.TargetEntityID, rec.TargetEntityLabel, string(rec.SourceSignal),
			rec.SourceLabel, rec.Score, rec.ExpiresAt,
		)
		if tx.Error != nil {
			return count, fmt.Errorf("recs recommendation create batch: %w", tx.Error)
		}
		count += tx.RowsAffected
	}
	return count, nil
}

func (r *PgRecommendationRepository) FindByID(ctx context.Context, scope *shared.FamilyScope, id uuid.UUID) (*Recommendation, error) {
	var row recommendationRow
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, student_id, recommendation_type, target_entity_id, target_entity_label,
		       source_signal, source_label, score, status, expires_at, created_at, updated_at
		FROM recs_recommendations
		WHERE id = ? AND family_id = ?`,
		id, scope.FamilyID(),
	).First(&row).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRecommendationNotFound
		}
		return nil, fmt.Errorf("recs find by id: %w", err)
	}

	recs := rowsToRecommendations([]recommendationRow{row})
	return &recs[0], nil
}

func (r *PgRecommendationRepository) FindActiveByFamily(ctx context.Context, scope *shared.FamilyScope, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error) {
	offset := 0
	if cursor != nil {
		var err error
		offset, err = decodeRecsCursor(*cursor)
		if err != nil {
			return nil, nil, err
		}
	}

	args := []any{scope.FamilyID()}
	typeFilter := ""
	if recommendationType != nil {
		typeFilter = " AND recommendation_type = ?"
		args = append(args, *recommendationType)
	}
	args = append(args, limit+1, offset)

	var rows []recommendationRow
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT id, family_id, student_id, recommendation_type, target_entity_id, target_entity_label,
		       source_signal, source_label, score, status, expires_at, created_at, updated_at
		FROM recs_recommendations
		WHERE family_id = ? AND status = 'active'%s
		ORDER BY score DESC
		LIMIT ? OFFSET ?`, typeFilter), args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, nil, fmt.Errorf("recs find active by family: %w", err)
	}

	var nextCursor *string
	if int64(len(rows)) > limit {
		rows = rows[:limit]
		c := encodeRecsCursor(offset + int(limit))
		nextCursor = &c
	}

	return rowsToRecommendations(rows), nextCursor, nil
}

func (r *PgRecommendationRepository) FindActiveByStudent(ctx context.Context, scope *shared.FamilyScope, studentID uuid.UUID, recommendationType *string, cursor *string, limit int64) ([]Recommendation, *string, error) {
	offset := 0
	if cursor != nil {
		var err error
		offset, err = decodeRecsCursor(*cursor)
		if err != nil {
			return nil, nil, err
		}
	}

	args := []any{scope.FamilyID(), studentID}
	typeFilter := ""
	if recommendationType != nil {
		typeFilter = " AND recommendation_type = ?"
		args = append(args, *recommendationType)
	}
	args = append(args, limit+1, offset)

	var rows []recommendationRow
	err := r.db.WithContext(ctx).Raw(fmt.Sprintf(`
		SELECT id, family_id, student_id, recommendation_type, target_entity_id, target_entity_label,
		       source_signal, source_label, score, status, expires_at, created_at, updated_at
		FROM recs_recommendations
		WHERE family_id = ? AND student_id = ? AND status = 'active'%s
		ORDER BY score DESC
		LIMIT ? OFFSET ?`, typeFilter), args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, nil, fmt.Errorf("recs find active by student: %w", err)
	}

	var nextCursor *string
	if int64(len(rows)) > limit {
		rows = rows[:limit]
		c := encodeRecsCursor(offset + int(limit))
		nextCursor = &c
	}

	return rowsToRecommendations(rows), nextCursor, nil
}

func (r *PgRecommendationRepository) UpdateStatus(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID, status string) error {
	tx := r.db.WithContext(ctx).Exec(`
		UPDATE recs_recommendations SET status = ?, updated_at = now()
		WHERE id = ? AND family_id = ?`,
		status, recommendationID, scope.FamilyID(),
	)
	if tx.Error != nil {
		return fmt.Errorf("recs update status: %w", tx.Error)
	}
	if tx.RowsAffected == 0 {
		return ErrRecommendationNotFound
	}
	return nil
}

func (r *PgRecommendationRepository) ExpireStale(ctx context.Context) (int64, error) {
	tx := r.db.WithContext(ctx).Exec(`
		UPDATE recs_recommendations SET status = 'expired', updated_at = now()
		WHERE status = 'active' AND expires_at < now()`)
	return tx.RowsAffected, tx.Error
}

func (r *PgRecommendationRepository) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM recs_recommendations WHERE family_id = ?`,
		familyID.UUID,
	)
	return tx.RowsAffected, tx.Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Feedback Repository [13-recs §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgFeedbackRepository implements FeedbackRepository using GORM/PostgreSQL.
type PgFeedbackRepository struct {
	db *gorm.DB
}

// NewPgFeedbackRepository creates a new PgFeedbackRepository.
func NewPgFeedbackRepository(db *gorm.DB) *PgFeedbackRepository {
	return &PgFeedbackRepository{db: db}
}

func (r *PgFeedbackRepository) Create(ctx context.Context, f NewFeedback) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO recs_recommendation_feedback (family_id, recommendation_id, action, blocked_entity_id)
		VALUES (?, ?, ?, ?)`,
		f.FamilyID.UUID, f.RecommendationID, f.Action, f.BlockedEntityID,
	).Error
}

func (r *PgFeedbackRepository) FindByRecommendation(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) (*Feedback, error) {
	type row struct {
		ID               uuid.UUID  `gorm:"column:id"`
		FamilyID         uuid.UUID  `gorm:"column:family_id"`
		RecommendationID uuid.UUID  `gorm:"column:recommendation_id"`
		Action           string     `gorm:"column:action"`
		BlockedEntityID  *uuid.UUID `gorm:"column:blocked_entity_id"`
		CreatedAt        time.Time  `gorm:"column:created_at"`
	}
	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, recommendation_id, action, blocked_entity_id, created_at
		FROM recs_recommendation_feedback
		WHERE recommendation_id = ? AND family_id = ?`,
		recommendationID, scope.FamilyID(),
	).First(&r_).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFeedbackNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("recs feedback find: %w", err)
	}
	return &Feedback{
		ID:               r_.ID,
		FamilyID:         shared.NewFamilyID(r_.FamilyID),
		RecommendationID: r_.RecommendationID,
		Action:           r_.Action,
		BlockedEntityID:  r_.BlockedEntityID,
		CreatedAt:        r_.CreatedAt,
	}, nil
}

func (r *PgFeedbackRepository) FindBlockedByFamily(ctx context.Context, scope *shared.FamilyScope) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	err := r.db.WithContext(ctx).Raw(`
		SELECT blocked_entity_id
		FROM recs_recommendation_feedback
		WHERE family_id = ? AND action = 'block' AND blocked_entity_id IS NOT NULL`,
		scope.FamilyID(),
	).Scan(&ids).Error
	return ids, err
}

func (r *PgFeedbackRepository) Delete(ctx context.Context, scope *shared.FamilyScope, recommendationID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		DELETE FROM recs_recommendation_feedback
		WHERE recommendation_id = ? AND family_id = ?`,
		recommendationID, scope.FamilyID(),
	).Error
}

func (r *PgFeedbackRepository) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM recs_recommendation_feedback WHERE family_id = ?`,
		familyID.UUID,
	)
	return tx.RowsAffected, tx.Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Popularity Repository [13-recs §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgPopularityRepository implements PopularityRepository using GORM/PostgreSQL.
type PgPopularityRepository struct {
	db *gorm.DB
}

// NewPgPopularityRepository creates a new PgPopularityRepository.
func NewPgPopularityRepository(db *gorm.DB) *PgPopularityRepository {
	return &PgPopularityRepository{db: db}
}

func (r *PgPopularityRepository) Upsert(ctx context.Context, s NewPopularityScore) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO recs_popularity_scores
			(listing_id, methodology_slug, period_start, period_end, popularity_score, purchase_count)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (listing_id, methodology_slug, period_start)
		DO UPDATE SET
			popularity_score = EXCLUDED.popularity_score,
			purchase_count = EXCLUDED.purchase_count,
			computed_at = now()`,
		s.ListingID, s.MethodologySlug,
		s.PeriodStart.Format("2006-01-02"), s.PeriodEnd.Format("2006-01-02"),
		s.PopularityScore, s.PurchaseCount,
	).Error
}

func (r *PgPopularityRepository) FindByMethodology(ctx context.Context, methodologySlug string, limit int64) ([]PopularityScore, error) {
	type row struct {
		ID              uuid.UUID `gorm:"column:id"`
		ListingID       uuid.UUID `gorm:"column:listing_id"`
		MethodologySlug string    `gorm:"column:methodology_slug"`
		PeriodStart     time.Time `gorm:"column:period_start"`
		PeriodEnd       time.Time `gorm:"column:period_end"`
		PopularityScore float32   `gorm:"column:popularity_score"`
		PurchaseCount   int       `gorm:"column:purchase_count"`
		ComputedAt      time.Time `gorm:"column:computed_at"`
	}
	var rows []row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, listing_id, methodology_slug, period_start, period_end, popularity_score, purchase_count, computed_at
		FROM recs_popularity_scores
		WHERE methodology_slug = ?
		ORDER BY popularity_score DESC
		LIMIT ?`,
		methodologySlug, limit,
	).Scan(&rows).Error
	if err != nil {
		return nil, fmt.Errorf("recs popularity find by methodology: %w", err)
	}
	out := make([]PopularityScore, len(rows))
	for i, r := range rows {
		out[i] = PopularityScore(r)
	}
	return out, nil
}

func (r *PgPopularityRepository) DeleteStale(ctx context.Context, before time.Time) (int64, error) {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM recs_popularity_scores WHERE period_end < ?`,
		before.Format("2006-01-02"),
	)
	return tx.RowsAffected, tx.Error
}

// ═══════════════════════════════════════════════════════════════════════════════
// Preference Repository [13-recs §6]
// ═══════════════════════════════════════════════════════════════════════════════

// PgPreferenceRepository implements PreferenceRepository using GORM/PostgreSQL.
type PgPreferenceRepository struct {
	db *gorm.DB
}

// NewPgPreferenceRepository creates a new PgPreferenceRepository.
func NewPgPreferenceRepository(db *gorm.DB) *PgPreferenceRepository {
	return &PgPreferenceRepository{db: db}
}

// defaultEnabledTypes returns the default recommendation types enabled for new families.
func defaultEnabledTypes() []string {
	return []string{
		string(RecommendationMarketplaceContent),
		string(RecommendationActivityIdea),
		string(RecommendationReadingSuggestion),
		string(RecommendationCommunityGroup),
	}
}

func (r *PgPreferenceRepository) FindOrDefault(ctx context.Context, scope *shared.FamilyScope) (*Preferences, error) {
	type row struct {
		ID                   uuid.UUID `gorm:"column:id"`
		FamilyID             uuid.UUID `gorm:"column:family_id"`
		EnabledTypes         string    `gorm:"column:enabled_types"` // PostgreSQL text[] returned as string
		ExplorationFrequency string    `gorm:"column:exploration_frequency"`
		CreatedAt            time.Time `gorm:"column:created_at"`
		UpdatedAt            time.Time `gorm:"column:updated_at"`
	}

	var r_ row
	err := r.db.WithContext(ctx).Raw(`
		SELECT id, family_id, enabled_types::text, exploration_frequency, created_at, updated_at
		FROM recs_preferences
		WHERE family_id = ?`,
		scope.FamilyID(),
	).First(&r_).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &Preferences{
			FamilyID:             shared.NewFamilyID(scope.FamilyID()),
			EnabledTypes:         defaultEnabledTypes(),
			ExplorationFrequency: string(ExplorationOccasional),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("recs preference find: %w", err)
	}

	return &Preferences{
		ID:                   r_.ID,
		FamilyID:             shared.NewFamilyID(r_.FamilyID),
		EnabledTypes:         parsePostgresArray(r_.EnabledTypes),
		ExplorationFrequency: r_.ExplorationFrequency,
		CreatedAt:            r_.CreatedAt,
		UpdatedAt:            r_.UpdatedAt,
	}, nil
}

func (r *PgPreferenceRepository) DeleteByFamily(ctx context.Context, familyID shared.FamilyID) (int64, error) {
	tx := r.db.WithContext(ctx).Exec(
		`DELETE FROM recs_preferences WHERE family_id = ?`,
		familyID.UUID,
	)
	return tx.RowsAffected, tx.Error
}

func (r *PgPreferenceRepository) Upsert(ctx context.Context, scope *shared.FamilyScope, p UpdatePreferences) (*Preferences, error) {
	typesJSON, _ := json.Marshal(p.EnabledTypes)
	err := r.db.WithContext(ctx).Exec(`
		INSERT INTO recs_preferences (family_id, enabled_types, exploration_frequency)
		VALUES (?, ?::text[]::text[], ?)
		ON CONFLICT (family_id) DO UPDATE SET
			enabled_types = EXCLUDED.enabled_types,
			exploration_frequency = EXCLUDED.exploration_frequency,
			updated_at = now()`,
		scope.FamilyID(), string(typesJSON), p.ExplorationFrequency,
	).Error
	if err != nil {
		return nil, fmt.Errorf("recs preference upsert: %w", err)
	}
	return r.FindOrDefault(ctx, scope)
}

// ═══════════════════════════════════════════════════════════════════════════════
// Anonymized Interaction Repository [13-recs §6, §14]
// ═══════════════════════════════════════════════════════════════════════════════

// PgAnonymizedInteractionRepository implements AnonymizedInteractionRepository.
// This table MUST NOT contain family_id or student_id. [13-recs §14.5]
type PgAnonymizedInteractionRepository struct {
	db *gorm.DB
}

// NewPgAnonymizedInteractionRepository creates a new PgAnonymizedInteractionRepository.
func NewPgAnonymizedInteractionRepository(db *gorm.DB) *PgAnonymizedInteractionRepository {
	return &PgAnonymizedInteractionRepository{db: db}
}

func (r *PgAnonymizedInteractionRepository) CreateBatch(ctx context.Context, interactions []NewAnonymizedInteraction) (int64, error) {
	if len(interactions) == 0 {
		return 0, nil
	}
	var count int64
	for _, i := range interactions {
		tx := r.db.WithContext(ctx).Exec(`
			INSERT INTO recs_anonymized_interactions
				(anonymous_id, interaction_type, methodology_slug, age_band, subject_category, duration_minutes, interaction_date)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			i.AnonymousID, i.InteractionType, i.MethodologySlug, i.AgeBand,
			i.SubjectCategory, i.DurationMinutes, i.InteractionDate.Format("2006-01-02"),
		)
		if tx.Error != nil {
			return count, fmt.Errorf("recs anonymized interaction create batch: %w", tx.Error)
		}
		count += tx.RowsAffected
	}
	return count, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

type recommendationRow struct {
	ID                 uuid.UUID  `gorm:"column:id"`
	FamilyID           uuid.UUID  `gorm:"column:family_id"`
	StudentID          *uuid.UUID `gorm:"column:student_id"`
	RecommendationType string     `gorm:"column:recommendation_type"`
	TargetEntityID     uuid.UUID  `gorm:"column:target_entity_id"`
	TargetEntityLabel  string     `gorm:"column:target_entity_label"`
	SourceSignal       string     `gorm:"column:source_signal"`
	SourceLabel        string     `gorm:"column:source_label"`
	Score              float32    `gorm:"column:score"`
	Status             string     `gorm:"column:status"`
	ExpiresAt          time.Time  `gorm:"column:expires_at"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func rowsToRecommendations(rows []recommendationRow) []Recommendation {
	out := make([]Recommendation, len(rows))
	for i, row := range rows {
		out[i] = Recommendation{
			ID:                 row.ID,
			FamilyID:           shared.NewFamilyID(row.FamilyID),
			StudentID:          row.StudentID,
			RecommendationType: RecommendationType(row.RecommendationType),
			TargetEntityID:     row.TargetEntityID,
			TargetEntityLabel:  row.TargetEntityLabel,
			SourceSignal:       SourceSignalType(row.SourceSignal),
			SourceLabel:        row.SourceLabel,
			Score:              row.Score,
			Status:             row.Status,
			ExpiresAt:          row.ExpiresAt,
			CreatedAt:          row.CreatedAt,
			UpdatedAt:          row.UpdatedAt,
		}
	}
	return out
}

func encodeRecsCursor(offset int) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.Itoa(offset)))
}

func decodeRecsCursor(cursor string) (int, error) {
	data, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		return 0, fmt.Errorf("recs: invalid cursor: %w", err)
	}
	n, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("recs: invalid cursor offset: %w", err)
	}
	return n, nil
}

// parsePostgresArray parses a PostgreSQL text[] literal like {"a","b"} into a []string.
func parsePostgresArray(s string) []string {
	if s == "" || s == "{}" {
		return nil
	}
	// Strip braces.
	if len(s) >= 2 && s[0] == '{' && s[len(s)-1] == '}' {
		s = s[1 : len(s)-1]
	}
	if s == "" {
		return nil
	}
	var out []string
	for _, part := range splitPostgresArray(s) {
		// Unquote if quoted.
		if len(part) >= 2 && part[0] == '"' && part[len(part)-1] == '"' {
			part = part[1 : len(part)-1]
		}
		out = append(out, part)
	}
	return out
}

func splitPostgresArray(s string) []string {
	var parts []string
	inQuotes := false
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuotes = !inQuotes
		case ',':
			if !inQuotes {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}
