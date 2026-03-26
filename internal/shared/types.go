package shared

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// ─── Newtype UUID Wrappers ────────────────────────────────────────────────────
// Prevents accidentally passing a FamilyID where a ParentID is expected. [ARCH §4.2]

// FamilyID is a type-safe wrapper for family UUIDs.
type FamilyID struct {
	uuid.UUID
}

// NewFamilyID creates a FamilyID from a UUID.
func NewFamilyID(id uuid.UUID) FamilyID {
	return FamilyID{UUID: id}
}

// MarshalJSON implements json.Marshaler.
func (id FamilyID) MarshalJSON() ([]byte, error) {
	return json.Marshal(id.UUID)
}

// UnmarshalJSON implements json.Unmarshaler.
func (id *FamilyID) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &id.UUID)
}

// ParentID is a type-safe wrapper for parent UUIDs.
type ParentID struct {
	uuid.UUID
}

func NewParentID(id uuid.UUID) ParentID { return ParentID{UUID: id} }

func (id ParentID) MarshalJSON() ([]byte, error)      { return json.Marshal(id.UUID) }
func (id *ParentID) UnmarshalJSON(data []byte) error  { return json.Unmarshal(data, &id.UUID) }

// StudentID is a type-safe wrapper for student UUIDs.
type StudentID struct {
	uuid.UUID
}

func NewStudentID(id uuid.UUID) StudentID { return StudentID{UUID: id} }

func (id StudentID) MarshalJSON() ([]byte, error)      { return json.Marshal(id.UUID) }
func (id *StudentID) UnmarshalJSON(data []byte) error  { return json.Unmarshal(data, &id.UUID) }

// CreatorID is a type-safe wrapper for creator UUIDs.
type CreatorID struct {
	uuid.UUID
}

func NewCreatorID(id uuid.UUID) CreatorID { return CreatorID{UUID: id} }

func (id CreatorID) MarshalJSON() ([]byte, error)      { return json.Marshal(id.UUID) }
func (id *CreatorID) UnmarshalJSON(data []byte) error  { return json.Unmarshal(data, &id.UUID) }

// ─── Subscription Tier ────────────────────────────────────────────────────────

// SubscriptionTier represents the family's subscription level.
type SubscriptionTier string

const (
	SubscriptionTierFree    SubscriptionTier = "free"
	SubscriptionTierPremium SubscriptionTier = "premium"
)

// ParseSubscriptionTier parses a string into a SubscriptionTier.
func ParseSubscriptionTier(s string) SubscriptionTier {
	switch s {
	case "premium":
		return SubscriptionTierPremium
	default:
		return SubscriptionTierFree
	}
}

// ─── AuthContext ──────────────────────────────────────────────────────────────

// AuthContext represents the authenticated user context, stored in Echo's request
// context by auth middleware. Consumed by every authenticated handler.
type AuthContext struct {
	ParentID           uuid.UUID        `json:"parent_id"`
	FamilyID           uuid.UUID        `json:"family_id"`
	IdentityID         uuid.UUID        `json:"identity_id"`
	DisplayName        string           `json:"display_name"`         // parent's display name
	IsPrimaryParent    bool             `json:"is_primary_parent"`
	IsPlatformAdmin    bool             `json:"is_platform_admin"`  // [S§3.1.5, 11-safety §9]
	SubscriptionTier   SubscriptionTier `json:"subscription_tier"`
	CoppaConsentStatus string           `json:"coppa_consent_status"` // [§20.6]
	Email              string           `json:"-"`                    // NOT logged or serialized — PII [CODING §5.2]
	SessionID          string           `json:"-"`                    // Kratos session ID — set by auth middleware [15-lifecycle §12]
}

// contextKey is an unexported type to prevent key collisions in Echo's context store.
type contextKey string

const authContextKey contextKey = "auth_context"

// SetAuthContext stores the AuthContext in the Echo request context.
func SetAuthContext(c echo.Context, auth *AuthContext) {
	c.Set(string(authContextKey), auth)
}

// GetAuthContext retrieves the AuthContext from the Echo request context.
// Returns ErrUnauthorized if not present — callers behind auth middleware always have it.
func GetAuthContext(c echo.Context) (*AuthContext, error) {
	val := c.Get(string(authContextKey))
	if val == nil {
		return nil, ErrUnauthorized()
	}
	auth, ok := val.(*AuthContext)
	if !ok {
		return nil, ErrUnauthorized()
	}
	return auth, nil
}
