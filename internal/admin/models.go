package admin

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types [16-admin §7]
// ═══════════════════════════════════════════════════════════════════════════════

// UserSearchQuery represents search parameters for user lookup.
type UserSearchQuery struct {
	Q            *string    `json:"q" query:"q"`                         // search by email or name
	FamilyID     *uuid.UUID `json:"family_id" query:"family_id"`         // filter by family
	Status       *string    `json:"status" query:"status"`               // "active", "suspended", "banned"
	Subscription *string    `json:"subscription" query:"subscription"`   // "free", "premium"
}

// CreateFlagInput represents input for creating a feature flag.
type CreateFlagInput struct {
	Key               string      `json:"key" validate:"required"`
	Description       string      `json:"description" validate:"required"`
	Enabled           bool        `json:"enabled"`
	RolloutPercentage *int16      `json:"rollout_percentage"`
	AllowedFamilyIDs  []uuid.UUID `json:"allowed_family_ids"`
}

// UpdateFlagInput represents input for updating a feature flag.
type UpdateFlagInput struct {
	Enabled           *bool        `json:"enabled"`
	Description       *string      `json:"description"`
	RolloutPercentage **int16      `json:"rollout_percentage"`  // nil = don't change, *nil = remove
	AllowedFamilyIDs  *[]uuid.UUID `json:"allowed_family_ids"` // nil = don't change
}

// AuditLogQuery represents search parameters for the audit log.
type AuditLogQuery struct {
	AdminID    *uuid.UUID `json:"admin_id" query:"admin_id"`
	Action     *string    `json:"action" query:"action"`
	TargetType *string    `json:"target_type" query:"target_type"`
	TargetID   *uuid.UUID `json:"target_id" query:"target_id"`
	FromDate   *time.Time `json:"from_date" query:"from_date"`
	ToDate     *time.Time `json:"to_date" query:"to_date"`
}

// AdminAction represents an admin action to be logged.
type AdminAction struct {
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   *uuid.UUID      `json:"target_id"`
	Details    json.RawMessage `json:"details"`
}

// CreateAuditLogEntry represents the data needed to create an audit log entry.
type CreateAuditLogEntry struct {
	AdminID    uuid.UUID       `json:"admin_id"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   *uuid.UUID      `json:"target_id"`
	Details    json.RawMessage `json:"details"`
	IPAddress  string          `json:"ip_address"`
	UserAgent  string          `json:"user_agent"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types [16-admin §7]
// ═══════════════════════════════════════════════════════════════════════════════

// AdminUserSummary represents a summary view of a user for admin listing.
type AdminUserSummary struct {
	FamilyID           uuid.UUID  `json:"family_id"`
	FamilyName         string     `json:"family_name"`
	PrimaryParentEmail string     `json:"primary_parent_email"`
	ParentCount        int32      `json:"parent_count"`
	StudentCount       int32      `json:"student_count"`
	SubscriptionTier   string     `json:"subscription_tier"`
	AccountStatus      string     `json:"account_status"`
	CreatedAt          time.Time  `json:"created_at"`
	LastActiveAt       *time.Time `json:"last_active_at"`
}

// AdminUserDetail represents the detailed view of a user for admin inspection.
type AdminUserDetail struct {
	Family            AdminFamilyInfo           `json:"family"`
	Parents           []AdminParentInfo         `json:"parents"`
	Students          []AdminStudentInfo        `json:"students"`
	Subscription      *AdminSubscriptionInfo    `json:"subscription"`
	ModerationHistory []ModerationActionSummary `json:"moderation_history"`
}

// AdminFamilyInfo represents family information for admin views.
type AdminFamilyInfo struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	AccountStatus string     `json:"account_status"`
	CreatedAt     time.Time  `json:"created_at"`
	LastActiveAt  *time.Time `json:"last_active_at"`
}

// AdminParentInfo represents parent information for admin views.
type AdminParentInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	Email       string    `json:"email"`
	IsPrimary   bool      `json:"is_primary"`
}

// AdminStudentInfo represents student information for admin views.
type AdminStudentInfo struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"display_name"`
	GradeLevel  *string   `json:"grade_level"`
}

// AdminSubscriptionInfo represents subscription information for admin views.
type AdminSubscriptionInfo struct {
	Tier      string     `json:"tier"`
	Status    string     `json:"status"`
	ExpiresAt *time.Time `json:"expires_at"`
}

// ModerationActionSummary represents a summary of a moderation action for admin views.
type ModerationActionSummary struct {
	Action    string    `json:"action"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// FeatureFlag represents a feature flag entity.
type FeatureFlag struct {
	ID                uuid.UUID   `json:"id"`
	Key               string      `json:"key"`
	Description       string      `json:"description"`
	Enabled           bool        `json:"enabled"`
	RolloutPercentage *int16      `json:"rollout_percentage"`
	AllowedFamilyIDs  []uuid.UUID `json:"allowed_family_ids"`
	CreatedBy         uuid.UUID   `json:"created_by"`
	UpdatedBy         *uuid.UUID  `json:"updated_by"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// SystemHealthResponse represents the aggregated system health status.
type SystemHealthResponse struct {
	Status     string            `json:"status"` // "healthy", "degraded", "unhealthy"
	Components []ComponentHealth `json:"components"`
	CheckedAt  time.Time         `json:"checked_at"`
}

// ComponentHealth represents the health of a single system component.
type ComponentHealth struct {
	Name      string  `json:"name"`       // "database", "redis", "r2", "kratos"
	Status    string  `json:"status"`     // "healthy", "degraded", "unhealthy"
	LatencyMs *int64  `json:"latency_ms"`
	Details   *string `json:"details"`
}

// JobStatusResponse represents background job queue status.
type JobStatusResponse struct {
	Queues          []QueueStatus `json:"queues"`
	DeadLetterCount int64         `json:"dead_letter_count"`
}

// QueueStatus represents the status of a single job queue.
type QueueStatus struct {
	Name         string `json:"name"`
	Pending      int64  `json:"pending"`
	Processing   int64  `json:"processing"`
	Completed24h int64  `json:"completed_24h"`
	Failed24h    int64  `json:"failed_24h"`
}

// DeadLetterJob represents a job in the dead-letter queue.
type DeadLetterJob struct {
	ID           string          `json:"id"`
	Queue        string          `json:"queue"`
	JobType      string          `json:"job_type"`
	Payload      json.RawMessage `json:"payload"`
	ErrorMessage string          `json:"error_message"`
	FailedAt     time.Time       `json:"failed_at"`
	RetryCount   int32           `json:"retry_count"`
}

// AuditLogEntry represents a single audit log entry.
type AuditLogEntry struct {
	ID         uuid.UUID       `json:"id"`
	AdminID    uuid.UUID       `json:"admin_id"`
	AdminEmail *string         `json:"admin_email"`
	Action     string          `json:"action"`
	TargetType string          `json:"target_type"`
	TargetID   *uuid.UUID      `json:"target_id"`
	Details    json.RawMessage `json:"details"`
	CreatedAt  time.Time       `json:"created_at"`
}
