package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"regexp"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// flagKeyRegex validates feature flag keys: lowercase, digits, hyphens, underscores.
// Max 100 chars, enforced separately. [16-admin §10, Plan §H]
var flagKeyRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// flagCacheTTL is the Redis cache TTL for feature flags. [16-admin §10.2]
const flagCacheTTL = 60 * time.Second

// ═══════════════════════════════════════════════════════════════════════════════
// Pure Functions (Wave 1)
// ═══════════════════════════════════════════════════════════════════════════════

// validateFlagKey checks that a flag key matches the required format.
// Valid: lowercase alphanumeric, hyphens, underscores. 1–100 chars.
func validateFlagKey(key string) bool {
	if key == "" || len(key) > 100 {
		return false
	}
	return flagKeyRegex.MatchString(key)
}

// evaluateFlag determines whether a feature flag is enabled for a given family.
// Pure function — no I/O, fully deterministic. [16-admin §10.2]
func evaluateFlag(flag *FeatureFlag, familyID *uuid.UUID) bool {
	if !flag.Enabled {
		return false
	}

	// If allowlist exists and family is specified, check membership.
	if len(flag.AllowedFamilyIDs) > 0 && familyID != nil {
		return slices.Contains(flag.AllowedFamilyIDs, *familyID)
	}

	// If percentage rollout, hash family_id for deterministic bucket.
	if flag.RolloutPercentage != nil && familyID != nil {
		hash := crc32.ChecksumIEEE(familyID[:]) % 100
		return int16(hash) < *flag.RolloutPercentage
	}

	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// Service Implementation
// ═══════════════════════════════════════════════════════════════════════════════

// AdminServiceImpl implements AdminService. [16-admin §5]
type AdminServiceImpl struct {
	flagRepo      FeatureFlagRepository
	auditRepo     AuditLogRepository
	cache         shared.Cache
	iamSvc        IamServiceForAdmin
	safetySvc     SafetyServiceForAdmin
	billingSvc    BillingServiceForAdmin
	healthChecker HealthChecker
	jobInspector  JobInspector
}

// NewAdminService creates an AdminService with all required dependencies.
func NewAdminService(
	flagRepo FeatureFlagRepository,
	auditRepo AuditLogRepository,
	cache shared.Cache,
	iamSvc IamServiceForAdmin,
	safetySvc SafetyServiceForAdmin,
	billingSvc BillingServiceForAdmin,
	healthChecker HealthChecker,
	jobInspector JobInspector,
) AdminService {
	return &AdminServiceImpl{
		flagRepo:      flagRepo,
		auditRepo:     auditRepo,
		cache:         cache,
		iamSvc:        iamSvc,
		safetySvc:     safetySvc,
		billingSvc:    billingSvc,
		healthChecker: healthChecker,
		jobInspector:  jobInspector,
	}
}

// ─── Feature Flag CRUD ──────────────────────────────────────────────────────

// ListFlags lists all feature flags. [16-admin §5]
func (s *AdminServiceImpl) ListFlags(ctx context.Context, _ *shared.AuthContext) ([]FeatureFlag, error) {
	flags, err := s.flagRepo.ListAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing flags: %w", err)
	}
	return flags, nil
}

// CreateFlag creates a new feature flag with audit logging. [16-admin §5, §8]
func (s *AdminServiceImpl) CreateFlag(ctx context.Context, auth *shared.AuthContext, input *CreateFlagInput) (*FeatureFlag, error) {
	if !validateFlagKey(input.Key) {
		return nil, ErrInvalidFlagKey
	}

	flag, err := s.flagRepo.Create(ctx, input, auth.ParentID)
	if err != nil {
		return nil, err
	}

	// Audit log — fire-and-forget style but propagate error per spec §8.
	flagID := flag.ID
	if _, auditErr := s.auditRepo.Create(ctx, &CreateAuditLogEntry{
		AdminID:    auth.ParentID,
		Action:     "flag_create",
		TargetType: "feature_flag",
		TargetID:   &flagID,
		Details:    json.RawMessage(fmt.Sprintf(`{"key":%q}`, input.Key)),
	}); auditErr != nil {
		return nil, fmt.Errorf("logging audit: %w", auditErr)
	}

	return flag, nil
}

// UpdateFlag updates a feature flag with audit logging. [16-admin §5, §8]
func (s *AdminServiceImpl) UpdateFlag(ctx context.Context, auth *shared.AuthContext, key string, input *UpdateFlagInput) (*FeatureFlag, error) {
	flag, err := s.flagRepo.Update(ctx, key, input, auth.ParentID)
	if err != nil {
		return nil, err
	}

	flagID := flag.ID
	if _, auditErr := s.auditRepo.Create(ctx, &CreateAuditLogEntry{
		AdminID:    auth.ParentID,
		Action:     "flag_update",
		TargetType: "feature_flag",
		TargetID:   &flagID,
		Details:    json.RawMessage(fmt.Sprintf(`{"key":%q}`, key)),
	}); auditErr != nil {
		return nil, fmt.Errorf("logging audit: %w", auditErr)
	}

	return flag, nil
}

// DeleteFlag deletes a feature flag with audit logging. [16-admin §5, §8]
func (s *AdminServiceImpl) DeleteFlag(ctx context.Context, auth *shared.AuthContext, key string) error {
	if err := s.flagRepo.Delete(ctx, key); err != nil {
		return err
	}

	if _, auditErr := s.auditRepo.Create(ctx, &CreateAuditLogEntry{
		AdminID:    auth.ParentID,
		Action:     "flag_delete",
		TargetType: "feature_flag",
		Details:    json.RawMessage(fmt.Sprintf(`{"key":%q}`, key)),
	}); auditErr != nil {
		return fmt.Errorf("logging audit: %w", auditErr)
	}

	return nil
}

// ─── Feature Flag Evaluation ────────────────────────────────────────────────

// IsFlagEnabled evaluates whether a flag is enabled for a specific family.
// Checks Redis cache first, falls back to DB. [16-admin §10.2]
func (s *AdminServiceImpl) IsFlagEnabled(ctx context.Context, key string, familyID *uuid.UUID) (bool, error) {
	// 1. Check Redis cache first (1-minute TTL).
	cacheKey := fmt.Sprintf("flag:%s", key)
	cached, err := shared.CacheGet[FeatureFlag](ctx, s.cache, cacheKey)
	if err == nil && cached != nil {
		return evaluateFlag(cached, familyID), nil
	}

	// 2. Fall back to database.
	flag, err := s.flagRepo.FindByKey(ctx, key)
	if err != nil {
		return false, fmt.Errorf("looking up flag: %w", err)
	}
	if flag == nil {
		return false, ErrFlagNotFound
	}

	// 3. Cache for 1 minute. Cache write failure is non-fatal. [16-admin §10.2]
	_ = shared.CacheSet(ctx, s.cache, cacheKey, *flag, flagCacheTTL)

	return evaluateFlag(flag, familyID), nil
}

// ─── Audit Log ──────────────────────────────────────────────────────────────

// LogAction records an admin action in the audit log. [16-admin §8]
func (s *AdminServiceImpl) LogAction(ctx context.Context, auth *shared.AuthContext, action *AdminAction) error {
	if _, err := s.auditRepo.Create(ctx, &CreateAuditLogEntry{
		AdminID:    auth.ParentID,
		Action:     action.Action,
		TargetType: action.TargetType,
		TargetID:   action.TargetID,
		Details:    action.Details,
	}); err != nil {
		return fmt.Errorf("logging audit: %w", err)
	}
	return nil
}

// SearchAuditLog searches/filters the admin audit log. [16-admin §5]
func (s *AdminServiceImpl) SearchAuditLog(ctx context.Context, _ *shared.AuthContext, query *AuditLogQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AuditLogEntry], error) {
	entries, err := s.auditRepo.Search(ctx, query, pagination)
	if err != nil {
		return nil, fmt.Errorf("searching audit log: %w", err)
	}

	return &shared.PaginatedResponse[AuditLogEntry]{
		Data:    entries,
		HasMore: len(entries) >= pagination.EffectiveLimit(),
	}, nil
}

// GetUserAuditTrail returns audit trail for a specific family. [16-admin §5]
func (s *AdminServiceImpl) GetUserAuditTrail(ctx context.Context, _ *shared.AuthContext, familyID uuid.UUID, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AuditLogEntry], error) {
	entries, err := s.auditRepo.FindByTarget(ctx, "family", familyID, pagination)
	if err != nil {
		return nil, fmt.Errorf("getting user audit trail: %w", err)
	}

	return &shared.PaginatedResponse[AuditLogEntry]{
		Data:    entries,
		HasMore: len(entries) >= pagination.EffectiveLimit(),
	}, nil
}

// ─── User Management ────────────────────────────────────────────────────────

// SearchUsers searches users by email, name, or family ID. Delegates to IAM. [16-admin §4, §14]
func (s *AdminServiceImpl) SearchUsers(ctx context.Context, _ *shared.AuthContext, query *UserSearchQuery, pagination *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error) {
	result, err := s.iamSvc.SearchUsers(ctx, query, pagination)
	if err != nil {
		return nil, fmt.Errorf("searching users: %w", err)
	}
	return result, nil
}

// GetUserDetail returns aggregated user info from IAM + billing + safety.
// Billing and safety errors are non-fatal — those sections are omitted on failure.
// [16-admin §4, §14]
func (s *AdminServiceImpl) GetUserDetail(ctx context.Context, _ *shared.AuthContext, familyID uuid.UUID) (*AdminUserDetail, error) {
	// Family detail is required — if not found, return ErrUserNotFound.
	family, err := s.iamSvc.GetFamilyDetail(ctx, familyID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	parents, err := s.iamSvc.GetParents(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("getting parents: %w", err)
	}

	students, err := s.iamSvc.GetStudents(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("getting students: %w", err)
	}

	detail := &AdminUserDetail{
		Family:   *family,
		Parents:  parents,
		Students: students,
	}

	// Billing — non-fatal. [16-admin §4]
	sub, subErr := s.billingSvc.GetSubscriptionInfo(ctx, familyID)
	if subErr == nil {
		detail.Subscription = sub
	}

	// Safety — non-fatal. [16-admin §4]
	history, histErr := s.safetySvc.GetModerationHistory(ctx, familyID)
	if histErr == nil {
		detail.ModerationHistory = history
	} else {
		detail.ModerationHistory = []ModerationActionSummary{}
	}

	return detail, nil
}

// ─── System Health ──────────────────────────────────────────────────────────

// GetSystemHealth returns aggregated system health status. [16-admin §11.1]
func (s *AdminServiceImpl) GetSystemHealth(ctx context.Context, _ *shared.AuthContext) (*SystemHealthResponse, error) {
	components := s.healthChecker.CheckAll(ctx)

	overall := "healthy"
	for _, c := range components {
		if c.Status == "unhealthy" {
			overall = "unhealthy"
			break
		}
		if c.Status == "degraded" {
			overall = "degraded"
		}
	}

	return &SystemHealthResponse{
		Status:     overall,
		Components: components,
		CheckedAt:  time.Now(),
	}, nil
}

// GetJobStatus returns background job queue status. [16-admin §11.2]
func (s *AdminServiceImpl) GetJobStatus(ctx context.Context, _ *shared.AuthContext) (*JobStatusResponse, error) {
	return s.jobInspector.GetQueueStatus(ctx)
}

// GetDeadLetterJobs returns dead-letter queue contents. [16-admin §11.2]
func (s *AdminServiceImpl) GetDeadLetterJobs(ctx context.Context, _ *shared.AuthContext, pagination *shared.PaginationParams) (*shared.PaginatedResponse[DeadLetterJob], error) {
	jobs, err := s.jobInspector.GetDeadLetterJobs(ctx, pagination)
	if err != nil {
		return nil, fmt.Errorf("getting dead-letter jobs: %w", err)
	}

	return &shared.PaginatedResponse[DeadLetterJob]{
		Data:    jobs,
		HasMore: len(jobs) >= pagination.EffectiveLimit(),
	}, nil
}

// RetryDeadLetterJob retries a dead-letter job with audit logging. [16-admin §11.2]
func (s *AdminServiceImpl) RetryDeadLetterJob(ctx context.Context, auth *shared.AuthContext, jobID string) error {
	if err := s.jobInspector.RetryDeadLetterJob(ctx, jobID); err != nil {
		return err
	}

	if _, auditErr := s.auditRepo.Create(ctx, &CreateAuditLogEntry{
		AdminID:    auth.ParentID,
		Action:     "system_config_update",
		TargetType: "system",
		Details:    json.RawMessage(fmt.Sprintf(`{"action":"retry_dead_letter","job_id":%q}`, jobID)),
	}); auditErr != nil {
		return fmt.Errorf("logging audit: %w", auditErr)
	}

	return nil
}
