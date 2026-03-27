package recs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"gorm.io/gorm"
)

// Task type constants for background job routing. [13-recs §11]
const (
	TaskTypeComputeRecommendations = "recs:compute_recommendations"
	TaskTypeAggregatePopularity    = "recs:aggregate_popularity"
	TaskTypePurgeStaleSignals      = "recs:purge_stale_signals"
	TaskTypeAnonymizeInteractions  = "recs:anonymize_interactions"
)

// ─── Task Payload Types ───────────────────────────────────────────────────────

// ComputeRecommendationsPayload is the payload for the daily recommendation computation task.
// [13-recs §11.1] Schedule: daily at 4:00 AM UTC.
type ComputeRecommendationsPayload struct{}

func (ComputeRecommendationsPayload) TaskType() string { return TaskTypeComputeRecommendations }

// AggregatePopularityPayload is the payload for the daily popularity aggregation task.
// [13-recs §11.2] Schedule: daily at 3:00 AM UTC (before ComputeRecommendationsTask).
type AggregatePopularityPayload struct{}

func (AggregatePopularityPayload) TaskType() string { return TaskTypeAggregatePopularity }

// PurgeStaleSignalsPayload is the payload for the weekly signal purge task.
// [13-recs §11.3] Schedule: weekly, Sunday at 2:00 AM UTC.
type PurgeStaleSignalsPayload struct{}

func (PurgeStaleSignalsPayload) TaskType() string { return TaskTypePurgeStaleSignals }

// AnonymizeInteractionsPayload is the payload for the weekly anonymization task.
// [13-recs §11.4] Schedule: weekly, Sunday at 3:00 AM UTC (after PurgeStaleSignalsTask).
type AnonymizeInteractionsPayload struct{}

func (AnonymizeInteractionsPayload) TaskType() string { return TaskTypeAnonymizeInteractions }

// Compile-time interface checks.
var (
	_ shared.JobPayload = ComputeRecommendationsPayload{}
	_ shared.JobPayload = AggregatePopularityPayload{}
	_ shared.JobPayload = PurgeStaleSignalsPayload{}
	_ shared.JobPayload = AnonymizeInteractionsPayload{}
)

// ─── Task Registration ────────────────────────────────────────────────────────

// RegisterTaskHandlers registers recs:: background task handlers with the job worker.
// Accepts *gorm.DB for cross-family batch reads that bypass FamilyScope. [13-recs §5]
func RegisterTaskHandlers(
	worker shared.JobWorker,
	db *gorm.DB,
	signalRepo SignalRepository,
	recRepo RecommendationRepository,
	feedbackRepo FeedbackRepository,
	popularityRepo PopularityRepository,
	prefRepo PreferenceRepository,
	anonRepo AnonymizedInteractionRepository,
	anonymizationSecret string,
) {
	worker.Handle(TaskTypePurgeStaleSignals, handlePurgeStaleSignalsTask(signalRepo))
	worker.Handle(TaskTypeAnonymizeInteractions, handleAnonymizeInteractionsTask(db, signalRepo, anonRepo, anonymizationSecret))
	worker.Handle(TaskTypeComputeRecommendations, handleComputeRecommendationsTask(db, signalRepo, recRepo, feedbackRepo, popularityRepo, prefRepo))
	worker.Handle(TaskTypeAggregatePopularity, handleAggregatePopularityTask(db, popularityRepo))
}

// ─── Task Handlers ────────────────────────────────────────────────────────────

// handlePurgeStaleSignalsTask returns a JobHandler that deletes signals older than 90 days. [13-recs §11.3]
func handlePurgeStaleSignalsTask(signalRepo SignalRepository) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task PurgeStaleSignalsPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("recs: unmarshal purge_stale_signals task: %w", err)
		}

		cutoff := time.Now().UTC().AddDate(0, 0, -90)
		deleted, err := signalRepo.DeleteStale(ctx, cutoff)
		if err != nil {
			return fmt.Errorf("recs: purge stale signals: %w", err)
		}
		slog.Info("recs: purge_stale_signals complete", "deleted", deleted)
		return nil
	}
}

// handleAnonymizeInteractionsTask returns a JobHandler that anonymizes recent signals
// into the recs_anonymized_interactions table. [13-recs §11.4]
func handleAnonymizeInteractionsTask(db *gorm.DB, _ SignalRepository, anonRepo AnonymizedInteractionRepository, secret string) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task AnonymizeInteractionsPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("recs: unmarshal anonymize_interactions task: %w", err)
		}

		if secret == "" {
			slog.Info("recs: anonymize_interactions skipped (no RECS_ANONYMIZATION_SECRET)")
			return nil
		}

		secretKey := []byte(secret)
		now := time.Now().UTC()
		since := now.AddDate(0, 0, -7)

		// Query signals from last 7 days. Cross-family batch read — no FamilyScope. [13-recs §5]
		type signalRow struct {
			ID              uuid.UUID `gorm:"column:id"`
			FamilyID        uuid.UUID `gorm:"column:family_id"`
			StudentID       *uuid.UUID `gorm:"column:student_id"`
			SignalType      string    `gorm:"column:signal_type"`
			MethodologySlug string    `gorm:"column:methodology_slug"`
			Payload         []byte    `gorm:"column:payload"`
			SignalDate      time.Time `gorm:"column:signal_date"`
		}
		var signals []signalRow
		if err := db.WithContext(ctx).Raw(`
			SELECT id, family_id, student_id, signal_type, methodology_slug, payload, signal_date
			FROM recs_signals
			WHERE signal_date >= ?
			ORDER BY signal_date DESC`,
			since,
		).Scan(&signals).Error; err != nil {
			return fmt.Errorf("recs: anonymize query signals: %w", err)
		}

		if len(signals) == 0 {
			slog.Info("recs: anonymize_interactions complete", "processed", 0)
			return nil
		}

		// Collect unique student IDs for birth_year lookup.
		studentIDs := make(map[uuid.UUID]struct{})
		for _, s := range signals {
			if s.StudentID != nil {
				studentIDs[*s.StudentID] = struct{}{}
			}
		}

		// Batch-load student birth years for age-band computation.
		birthYears := make(map[uuid.UUID]int16)
		if len(studentIDs) > 0 {
			ids := make([]uuid.UUID, 0, len(studentIDs))
			for id := range studentIDs {
				ids = append(ids, id)
			}
			type birthRow struct {
				ID        uuid.UUID `gorm:"column:id"`
				BirthYear int16     `gorm:"column:birth_year"`
			}
			var rows []birthRow
			if err := db.WithContext(ctx).Raw(
				`SELECT id, birth_year FROM iam_students WHERE id IN ?`, ids,
			).Scan(&rows).Error; err != nil {
				slog.Error("recs: anonymize birth_year lookup", "error", err)
				// Non-fatal — age bands will be empty.
			} else {
				for _, r := range rows {
					birthYears[r.ID] = r.BirthYear
				}
			}
		}

		// Build anonymized interactions.
		currentYear := int16(now.Year())
		var interactions []NewAnonymizedInteraction
		for _, s := range signals {
			anonID := computeHMAC(s.FamilyID, secretKey)

			// Determine age band from birth year.
			ageBand := ""
			if s.StudentID != nil {
				if by, ok := birthYears[*s.StudentID]; ok && by > 0 {
					age := int(currentYear - by)
					ageBand = CoarsenAgeBand(age)
				}
			}
			if ageBand == "" {
				continue // skip signals without a valid age band [13-recs §14.1]
			}

			// Extract subject category and duration from payload.
			var payloadMap map[string]any
			_ = json.Unmarshal(s.Payload, &payloadMap)

			var subjectCategory *string
			if tags, ok := payloadMap["subject_tags"]; ok {
				if tagSlice, ok := tags.([]any); ok && len(tagSlice) > 0 {
					if first, ok := tagSlice[0].(string); ok {
						subjectCategory = &first
					}
				}
			}

			var durationMinutes *int
			if dur, ok := payloadMap["duration_minutes"]; ok {
				if durFloat, ok := dur.(float64); ok {
					rounded := RoundDurationToNearest5(int(durFloat))
					durationMinutes = &rounded
				}
			}

			interactions = append(interactions, NewAnonymizedInteraction{
				AnonymousID:     anonID,
				InteractionType: s.SignalType,
				MethodologySlug: s.MethodologySlug,
				AgeBand:         ageBand,
				SubjectCategory: subjectCategory,
				DurationMinutes: durationMinutes,
				InteractionDate: s.SignalDate,
			})
		}

		if len(interactions) > 0 {
			count, err := anonRepo.CreateBatch(ctx, interactions)
			if err != nil {
				return fmt.Errorf("recs: anonymize create batch: %w", err)
			}
			slog.Info("recs: anonymize_interactions complete", "processed", count)
		} else {
			slog.Info("recs: anonymize_interactions complete", "processed", 0, "skipped", "no valid age bands")
		}
		return nil
	}
}

// handleAggregatePopularityTask returns a JobHandler that aggregates purchase signals
// into per-methodology popularity scores. [13-recs §11.2]
func handleAggregatePopularityTask(db *gorm.DB, popularityRepo PopularityRepository) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task AggregatePopularityPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("recs: unmarshal aggregate_popularity task: %w", err)
		}

		now := time.Now().UTC()
		since := now.AddDate(0, 0, -90)

		// Query active methodology slugs.
		var methodologySlugs []string
		if err := db.WithContext(ctx).Raw(
			`SELECT slug FROM method_definitions WHERE active = true`,
		).Scan(&methodologySlugs).Error; err != nil {
			return fmt.Errorf("recs: aggregate popularity list methodologies: %w", err)
		}

		periodStart := since
		periodEnd := now
		var totalUpserted int

		for _, slug := range methodologySlugs {
			// Aggregate purchases with recency decay: e^(-0.03 * days_ago)
			type purchaseRow struct {
				ListingID uuid.UUID `gorm:"column:listing_id"`
				Score     float64   `gorm:"column:score"`
				Count     int       `gorm:"column:count"`
			}
			var rows []purchaseRow
			if err := db.WithContext(ctx).Raw(`
				SELECT
					(payload->>'listing_id')::uuid AS listing_id,
					SUM(EXP(-0.03 * EXTRACT(EPOCH FROM (? - signal_date)) / 86400)) AS score,
					COUNT(*) AS count
				FROM recs_signals
				WHERE signal_type = 'purchase_completed'
				  AND methodology_slug = ?
				  AND signal_date >= ?
				GROUP BY (payload->>'listing_id')::uuid`,
				now, slug, since,
			).Scan(&rows).Error; err != nil {
				slog.Error("recs: aggregate popularity query", "methodology", slug, "error", err)
				continue
			}

			for _, r := range rows {
				if err := popularityRepo.Upsert(ctx, NewPopularityScore{
					ListingID:       r.ListingID,
					MethodologySlug: slug,
					PeriodStart:     periodStart,
					PeriodEnd:       periodEnd,
					PopularityScore: float32(r.Score),
					PurchaseCount:   r.Count,
				}); err != nil {
					slog.Error("recs: aggregate popularity upsert", "listing", r.ListingID, "error", err)
					continue
				}
				totalUpserted++
			}
		}

		// Purge stale scores older than 90 days.
		deleted, err := popularityRepo.DeleteStale(ctx, since)
		if err != nil {
			slog.Error("recs: aggregate popularity purge stale", "error", err)
		}

		slog.Info("recs: aggregate_popularity complete",
			"methodologies", len(methodologySlugs),
			"upserted", totalUpserted,
			"purged", deleted,
		)
		return nil
	}
}

// handleComputeRecommendationsTask returns a JobHandler that generates recommendations
// for premium families. [13-recs §11.1]
func handleComputeRecommendationsTask(
	db *gorm.DB,
	signalRepo SignalRepository,
	recRepo RecommendationRepository,
	feedbackRepo FeedbackRepository,
	popularityRepo PopularityRepository,
	prefRepo PreferenceRepository,
) shared.JobHandler {
	return func(ctx context.Context, payload []byte) error {
		var task ComputeRecommendationsPayload
		if err := json.Unmarshal(payload, &task); err != nil {
			return fmt.Errorf("recs: unmarshal compute_recommendations task: %w", err)
		}

		now := time.Now().UTC()

		// Step 1: Expire stale recommendations.
		expired, err := recRepo.ExpireStale(ctx)
		if err != nil {
			slog.Error("recs: compute expire stale", "error", err)
		}

		// Step 2: Query premium families.
		type familyRow struct {
			ID                      uuid.UUID `gorm:"column:id"`
			PrimaryMethodologySlug  string    `gorm:"column:primary_methodology_slug"`
		}
		var families []familyRow
		if err := db.WithContext(ctx).Raw(`
			SELECT id, primary_methodology_slug
			FROM iam_families
			WHERE subscription_tier = 'premium'`,
		).Scan(&families).Error; err != nil {
			return fmt.Errorf("recs: compute query families: %w", err)
		}

		slog.Info("recs: compute_recommendations starting",
			"families", len(families),
			"expired", expired,
		)

		var totalCreated int64
		for _, family := range families {
			familyID := shared.NewFamilyID(family.ID)
			scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: family.ID})

			// Load 90-day signals.
			since := now.AddDate(0, 0, -90)
			signals, err := signalRepo.FindByFamily(ctx, &scope, since)
			if err != nil {
				slog.Error("recs: compute load signals", "family_id", family.ID, "error", err)
				continue
			}

			// Load blocked entities.
			blockedEntityIDs, err := feedbackRepo.FindBlockedByFamily(ctx, &scope)
			if err != nil {
				slog.Error("recs: compute load blocked", "family_id", family.ID, "error", err)
				blockedEntityIDs = nil
			}
			blockedSet := make(map[uuid.UUID]struct{}, len(blockedEntityIDs))
			for _, id := range blockedEntityIDs {
				blockedSet[id] = struct{}{}
			}

			// Load preferences.
			prefs, err := prefRepo.FindOrDefault(ctx, &scope)
			if err != nil {
				slog.Error("recs: compute load preferences", "family_id", family.ID, "error", err)
				continue
			}

			// Determine exploration ratio.
			explorationRatio := ExplorationFrequency(prefs.ExplorationFrequency).ExplorationRatio()

			// Collect subject tags from recent signals.
			var recentSubjectTags []string
			for _, s := range signals {
				if tags, ok := s.Payload["subject_tags"]; ok {
					if tagSlice, ok := tags.([]any); ok {
						for _, t := range tagSlice {
							if ts, ok := t.(string); ok {
								recentSubjectTags = append(recentSubjectTags, ts)
							}
						}
					}
				}
			}

			// Determine active methodology slugs.
			primarySlug := family.PrimaryMethodologySlug
			var secondarySlugs []string
			if err := db.WithContext(ctx).Raw(`
				SELECT unnest(secondary_methodology_slugs)
				FROM iam_families WHERE id = ?`, family.ID,
			).Scan(&secondarySlugs).Error; err != nil {
				// Non-fatal.
				secondarySlugs = nil
			}

			// Generate marketplace_content + reading_suggestion candidates.
			candidates := generateMarketplaceCandidates(ctx, db, primarySlug, secondarySlugs, signals, blockedSet, recentSubjectTags, popularityRepo, now)

			// Generate community_group candidates.
			communityCandidates := generateCommunityCandidates(ctx, db, primarySlug, secondarySlugs, blockedSet, now)
			candidates = append(candidates, communityCandidates...)

			// Generate activity_idea candidates from progress gap detection. [13-recs §10.2]
			activityCandidates := generateActivityIdeaCandidates(ctx, db, primarySlug, signals, now)
			candidates = append(candidates, activityCandidates...)

			// Generate age transition candidates for students approaching stage boundaries. [13-recs §10.6]
			ageTransitionCandidates := generateAgeTransitionCandidates(ctx, db, primarySlug, family.ID, blockedSet, now)
			candidates = append(candidates, ageTransitionCandidates...)

			// Generate exploration candidates from other methodologies. [13-recs §10.7]
			explorationCandidates := generateExplorationCandidates(ctx, db, primarySlug, secondarySlugs, blockedSet, popularityRepo, now)
			candidates = append(candidates, explorationCandidates...)

			// Apply enabled_types preference — filter out types the family has disabled.
			// Exploration candidates bypass this filter (controlled separately via exploration_frequency). [13-recs §11.1]
			candidates = filterByEnabledTypes(candidates, prefs.EnabledTypes)

			if len(candidates) == 0 {
				continue
			}

			// Allocate exploration slots.
			totalSlots := 50
			explorationSlots := int(float32(totalSlots) * explorationRatio)
			regularSlots := totalSlots - explorationSlots

			// Sort candidates by score (descending) — take top regularSlots.
			sortCandidatesByScore(candidates)

			var selected []NewRecommendation
			explorationCount := 0
			regularCount := 0

			for _, c := range candidates {
				if regularCount >= regularSlots && explorationCount >= explorationSlots {
					break
				}
				if c.SourceSignal == SourceExploration {
					if explorationCount < explorationSlots {
						selected = append(selected, newRecommendationFromCandidate(c, familyID, now))
						explorationCount++
					}
				} else {
					if regularCount < regularSlots {
						selected = append(selected, newRecommendationFromCandidate(c, familyID, now))
						regularCount++
					}
				}
			}

			if len(selected) > 0 {
				created, err := recRepo.CreateBatch(ctx, selected)
				if err != nil {
					slog.Error("recs: compute create batch", "family_id", family.ID, "error", err)
					continue
				}
				totalCreated += created
			}
		}

		slog.Info("recs: compute_recommendations complete",
			"families", len(families),
			"recommendations_created", totalCreated,
		)
		return nil
	}
}

// ─── Candidate Helpers ──────────────────────────────────────────────────────

type candidate struct {
	RecommendationType RecommendationType
	TargetEntityID     uuid.UUID
	TargetEntityLabel  string
	SourceSignal       SourceSignalType
	SourceLabel        string
	Score              float32
	StudentID          *uuid.UUID
}

func generateMarketplaceCandidates(
	ctx context.Context,
	db *gorm.DB,
	primarySlug string,
	secondarySlugs []string,
	signals []Signal,
	blockedSet map[uuid.UUID]struct{},
	recentSubjectTags []string,
	popularityRepo PopularityRepository,
	now time.Time,
) []candidate {
	// Query ALL published listings. No methodology filter — mkt_listings does not have a
	// methodology_slug column. Methodology alignment is derived from popularity_scores. [Fix 1a]
	type listingRow struct {
		ID          uuid.UUID `gorm:"column:id"`
		Title       string    `gorm:"column:title"`
		ContentType string    `gorm:"column:content_type"`
		SubjectTags string    `gorm:"column:subject_tags"` // PostgreSQL text[] as string
		CreatedAt   time.Time `gorm:"column:created_at"`
	}

	var listings []listingRow
	if err := db.WithContext(ctx).Raw(`
		SELECT id, title, content_type, subject_tags::text, created_at
		FROM mkt_listings
		WHERE status = 'published'
		LIMIT 500`,
	).Scan(&listings).Error; err != nil {
		slog.Error("recs: compute marketplace candidates", "error", err)
		return nil
	}

	// Load popularity scores for primary methodology.
	popScores, _ := popularityRepo.FindByMethodology(ctx, primarySlug, 500)
	popMap := make(map[uuid.UUID]float32, len(popScores))
	var maxPop float32
	for _, p := range popScores {
		popMap[p.ListingID] = p.PopularityScore
		if p.PopularityScore > maxPop {
			maxPop = p.PopularityScore
		}
	}

	// Load secondary methodology popularity for secondary match detection.
	secondaryPopMap := make(map[uuid.UUID]float32)
	for _, slug := range secondarySlugs {
		scores, _ := popularityRepo.FindByMethodology(ctx, slug, 500)
		for _, p := range scores {
			if existing, ok := secondaryPopMap[p.ListingID]; !ok || p.PopularityScore > existing {
				secondaryPopMap[p.ListingID] = p.PopularityScore
			}
		}
	}

	// Collect purchased listing IDs from signals.
	purchasedSet := make(map[uuid.UUID]struct{})
	for _, s := range signals {
		if s.SignalType == SignalPurchaseCompleted {
			if lid, ok := s.Payload["listing_id"]; ok {
				if lidStr, ok := lid.(string); ok {
					if id, err := uuid.Parse(lidStr); err == nil {
						purchasedSet[id] = struct{}{}
					}
				}
			}
		}
	}

	// Determine current season for seasonal scoring. [13-recs §10.5]
	season := SeasonForMonth(now.Month())

	var candidates []candidate
	for _, l := range listings {
		if _, blocked := blockedSet[l.ID]; blocked {
			continue
		}
		if _, purchased := purchasedSet[l.ID]; purchased {
			continue
		}

		// Methodology match via popularity proxy: if families with this methodology
		// purchase the listing, it's methodology-aligned.
		var methodMatch float32
		if _, ok := popMap[l.ID]; ok {
			methodMatch = 1.0 // primary methodology families buy this
		} else if _, ok := secondaryPopMap[l.ID]; ok {
			methodMatch = 0.7 // secondary methodology alignment
		}

		// Popularity percentile.
		var popScore float32
		if maxPop > 0 {
			popScore = popMap[l.ID] / maxPop
		}

		// Subject tag relevance (Jaccard).
		listingTags := parsePostgresArray(l.SubjectTags)
		relevance := computeJaccardSimilarity(recentSubjectTags, listingTags)

		// Freshness decay.
		freshness := computeFreshness(l.CreatedAt, now)

		factors := ScoringFactors{
			MethodologyMatch: methodMatch,
			Popularity:       popScore,
			Relevance:        relevance,
			Freshness:        freshness,
			Exploration:      0.0,
		}
		score := ComputeScore(factors)

		// Seasonal boost. [13-recs §10.5]
		seasonalMatch, _ := HasSeasonalOverlap(listingTags, season)
		if seasonalMatch {
			score += seasonalBoost
		}

		// Determine dominant signal for transparency. [13-recs §13.1]
		dominant := DetermineDominantSignal(factors, primarySlug, seasonalMatch, season)

		// Determine recommendation type: book-related content → reading_suggestion. [13-recs §10.2]
		recType := RecommendationMarketplaceContent
		if l.ContentType == "book_list" || l.ContentType == "reading_guide" {
			recType = RecommendationReadingSuggestion
			if dominant.Signal == SourceMethodologyMatch {
				dominant.Signal = SourceReadingHistory
				dominant.Label = "Based on your reading interests"
			}
		}

		candidates = append(candidates, candidate{
			RecommendationType: recType,
			TargetEntityID:     l.ID,
			TargetEntityLabel:  l.Title,
			SourceSignal:       dominant.Signal,
			SourceLabel:        dominant.Label,
			Score:              score,
		})
	}
	return candidates
}

func generateCommunityCandidates(
	ctx context.Context,
	db *gorm.DB,
	primarySlug string,
	secondarySlugs []string,
	blockedSet map[uuid.UUID]struct{},
	now time.Time,
) []candidate {
	type groupRow struct {
		ID              uuid.UUID `gorm:"column:id"`
		Name            string    `gorm:"column:name"`
		MethodologySlug string    `gorm:"column:methodology_slug"`
		CreatedAt       time.Time `gorm:"column:created_at"`
	}

	allSlugs := append([]string{primarySlug}, secondarySlugs...)

	var groups []groupRow
	if err := db.WithContext(ctx).Raw(`
		SELECT id, name, methodology_slug, created_at
		FROM soc_groups
		WHERE join_policy IN ('open', 'request_to_join')
		  AND methodology_slug IN ?
		LIMIT 200`,
		allSlugs,
	).Scan(&groups).Error; err != nil {
		slog.Error("recs: compute community candidates", "error", err)
		return nil
	}

	var candidates []candidate
	for _, g := range groups {
		if _, blocked := blockedSet[g.ID]; blocked {
			continue
		}

		var methodMatch float32
		if g.MethodologySlug == primarySlug {
			methodMatch = 1.0
		} else {
			methodMatch = 0.7
		}

		freshness := computeFreshness(g.CreatedAt, now)

		score := ComputeScore(ScoringFactors{
			MethodologyMatch: methodMatch,
			Popularity:       0.5, // neutral — no group popularity metric yet
			Relevance:        0.0, // no subject tags on groups
			Freshness:        freshness,
			Exploration:      0.0,
		})

		label := "Methodology-aligned community"
		if g.MethodologySlug == primarySlug {
			label = primarySlug + " community"
		}

		candidates = append(candidates, candidate{
			RecommendationType: RecommendationCommunityGroup,
			TargetEntityID:     g.ID,
			TargetEntityLabel:  g.Name,
			SourceSignal:       SourceMethodologyMatch,
			SourceLabel:        label,
			Score:              score,
		})
	}
	return candidates
}

// generateActivityIdeaCandidates produces activity_idea candidates by detecting subject
// engagement gaps — subjects the family's methodology emphasizes but that have low or no
// recent signal activity. [13-recs §10.2, §10.3]
func generateActivityIdeaCandidates(
	ctx context.Context,
	db *gorm.DB,
	primarySlug string,
	signals []Signal,
	now time.Time,
) []candidate {
	// Baseline subjects from method_definitions.baseline_subjects column. [13-recs §10.2]
	var baselineSubjects []string
	if err := db.WithContext(ctx).Raw(
		`SELECT baseline_subjects FROM method_definitions WHERE slug = ?`, primarySlug,
	).Scan(&baselineSubjects).Error; err != nil {
		slog.Error("recs: load baseline subjects", "slug", primarySlug, "error", err)
	}
	if len(baselineSubjects) == 0 {
		return nil
	}

	// Compute engagement from recent signals (last 14 days).
	cutoff := now.AddDate(0, 0, -14)
	engaged := make(map[string]int)
	for _, s := range signals {
		if s.SignalDate.Before(cutoff) {
			continue
		}
		if tags, ok := s.Payload["subject_tags"]; ok {
			if tagSlice, ok := tags.([]any); ok {
				for _, t := range tagSlice {
					if ts, ok := t.(string); ok {
						engaged[ts]++
					}
				}
			}
		}
	}

	var candidates []candidate
	for _, subject := range baselineSubjects {
		if engaged[subject] > 0 {
			continue // already engaged recently
		}

		// Generate a synthetic candidate for the gap subject.
		// TargetEntityID is deterministic from subject+methodology to enable dedup.
		entityID := uuid.NewSHA1(uuid.NameSpaceDNS, []byte("activity:"+primarySlug+":"+subject))

		candidates = append(candidates, candidate{
			RecommendationType: RecommendationActivityIdea,
			TargetEntityID:     entityID,
			TargetEntityLabel:  "Try a " + subject + " activity",
			SourceSignal:       SourceProgressGap,
			SourceLabel:        "It's been a while since you explored " + subject,
			Score:              0.5, // neutral baseline — not competing with marketplace scores
		})
	}
	return candidates
}


// generateExplorationCandidates produces candidates from methodologies the family does NOT
// use, filtered to high-popularity items. These fill the exploration slots to prevent
// filter bubbles. [13-recs §10.7]
func generateExplorationCandidates(
	ctx context.Context,
	db *gorm.DB,
	primarySlug string,
	secondarySlugs []string,
	blockedSet map[uuid.UUID]struct{},
	popularityRepo PopularityRepository,
	_ time.Time,
) []candidate {
	// Query methodologies the family does NOT use.
	familySlugs := make(map[string]struct{}, 1+len(secondarySlugs))
	familySlugs[primarySlug] = struct{}{}
	for _, s := range secondarySlugs {
		familySlugs[s] = struct{}{}
	}

	var allSlugs []string
	if err := db.WithContext(ctx).Raw(
		`SELECT slug FROM method_definitions WHERE active = true`,
	).Scan(&allSlugs).Error; err != nil {
		slog.Error("recs: exploration list methodologies", "error", err)
		return nil
	}

	var otherSlugs []string
	for _, s := range allSlugs {
		if _, used := familySlugs[s]; !used {
			otherSlugs = append(otherSlugs, s)
		}
	}
	if len(otherSlugs) == 0 {
		return nil
	}

	// For each other methodology, get top-quartile popularity items.
	listingCache := make(map[uuid.UUID]string)

	var candidates []candidate
	for _, slug := range otherSlugs {
		scores, _ := popularityRepo.FindByMethodology(ctx, slug, 100)
		if len(scores) == 0 {
			continue
		}
		// Top quartile threshold.
		topCount := max(len(scores)/4, 1)

		for i := 0; i < topCount && i < len(scores); i++ {
			lid := scores[i].ListingID
			if _, blocked := blockedSet[lid]; blocked {
				continue
			}

			// Fetch listing title if not cached.
			if _, ok := listingCache[lid]; !ok {
				var title string
				if err := db.WithContext(ctx).Raw(
					`SELECT title FROM mkt_listings WHERE id = ? AND status = 'published'`, lid,
				).Scan(&title).Error; err != nil || title == "" {
					continue
				}
				listingCache[lid] = title
			}

			candidates = append(candidates, candidate{
				RecommendationType: RecommendationMarketplaceContent,
				TargetEntityID:     lid,
				TargetEntityLabel:  listingCache[lid],
				SourceSignal:       SourceExploration,
				SourceLabel:        "Something different — popular with " + slug + " families",
				Score:              0.6, // modest base score; exploration slots are allocated separately
			})
		}
	}
	return candidates
}

// transitionAge represents a methodology-specific stage boundary. [13-recs §10.6]
type transitionAge struct {
	StageName string // upcoming stage name (e.g., "Logic")
	Age       int    // approximate age at which transition occurs
}


// generateAgeTransitionCandidates produces candidates for students approaching a
// methodology-specific stage transition. [13-recs §10.6]
func generateAgeTransitionCandidates(
	ctx context.Context,
	db *gorm.DB,
	primarySlug string,
	familyID uuid.UUID,
	blockedSet map[uuid.UUID]struct{},
	now time.Time,
) []candidate {
	// Stage transition ages from method_definitions.transition_ages column. [13-recs §10.6]
	var transitionsJSON []byte
	if err := db.WithContext(ctx).Raw(
		`SELECT transition_ages FROM method_definitions WHERE slug = ?`, primarySlug,
	).Scan(&transitionsJSON).Error; err != nil {
		slog.Error("recs: load transition ages", "slug", primarySlug, "error", err)
		return nil
	}
	var transitions []transitionAge
	if err := json.Unmarshal(transitionsJSON, &transitions); err != nil {
		slog.Error("recs: unmarshal transition ages", "slug", primarySlug, "error", err)
		return nil
	}
	if len(transitions) == 0 {
		return nil
	}

	// Query family's students with birth_year.
	type studentRow struct {
		ID        uuid.UUID `gorm:"column:id"`
		BirthYear int16     `gorm:"column:birth_year"`
	}
	var students []studentRow
	if err := db.WithContext(ctx).Raw(`
		SELECT id, birth_year
		FROM iam_students
		WHERE family_id = ? AND birth_year > 0`,
		familyID,
	).Scan(&students).Error; err != nil {
		slog.Error("recs: compute age transition candidates", "error", err)
		return nil
	}
	if len(students) == 0 {
		return nil
	}

	currentYear := int(now.Year())

	var candidates []candidate
	for _, student := range students {
		approximateAge := currentYear - int(student.BirthYear)

		for _, tr := range transitions {
			// "Within 1 year" of transition — since we only have birth_year, not exact date.
			diff := tr.Age - approximateAge
			if diff < 0 || diff > 1 {
				continue
			}

			// Deterministic entity ID for dedup.
			entityID := uuid.NewSHA1(uuid.NameSpaceDNS,
				fmt.Appendf(nil, "age_transition:%s:%s:%s", primarySlug, student.ID, tr.StageName),
			)

			if _, blocked := blockedSet[entityID]; blocked {
				continue
			}

			label := fmt.Sprintf("Preparing for the %s stage (~age %d)", tr.StageName, tr.Age)
			sid := student.ID // capture for pointer

			candidates = append(candidates, candidate{
				RecommendationType: RecommendationMarketplaceContent,
				TargetEntityID:     entityID,
				TargetEntityLabel:  fmt.Sprintf("Resources for %s transition", tr.StageName),
				SourceSignal:       SourceAgeTransition,
				SourceLabel:        label,
				Score:              0.65, // slightly above activity ideas — transitions are time-sensitive
				StudentID:          &sid,
			})
		}
	}
	return candidates
}

func sortCandidatesByScore(candidates []candidate) {
	// Simple insertion sort — good enough for ≤700 candidates.
	for i := 1; i < len(candidates); i++ {
		key := candidates[i]
		j := i - 1
		for j >= 0 && candidates[j].Score < key.Score {
			candidates[j+1] = candidates[j]
			j--
		}
		candidates[j+1] = key
	}
}

// filterByEnabledTypes removes candidates whose RecommendationType is not in enabledTypes.
// Exploration candidates (SourceExploration) are always preserved — they are controlled
// separately by the exploration_frequency preference. [13-recs §11.1, §10.7]
func filterByEnabledTypes(candidates []candidate, enabledTypes []string) []candidate {
	if len(enabledTypes) == 0 {
		return candidates
	}
	allowed := make(map[RecommendationType]struct{}, len(enabledTypes))
	for _, t := range enabledTypes {
		allowed[RecommendationType(t)] = struct{}{}
	}
	filtered := make([]candidate, 0, len(candidates))
	for _, c := range candidates {
		if c.SourceSignal == SourceExploration {
			filtered = append(filtered, c)
			continue
		}
		if _, ok := allowed[c.RecommendationType]; ok {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func newRecommendationFromCandidate(c candidate, familyID shared.FamilyID, now time.Time) NewRecommendation {
	return NewRecommendation{
		FamilyID:           familyID,
		StudentID:          c.StudentID,
		RecommendationType: c.RecommendationType,
		TargetEntityID:     c.TargetEntityID,
		TargetEntityLabel:  c.TargetEntityLabel,
		SourceSignal:       c.SourceSignal,
		SourceLabel:        c.SourceLabel,
		Score:              c.Score,
		ExpiresAt:          now.Add(14 * 24 * time.Hour), // 14-day TTL
	}
}

