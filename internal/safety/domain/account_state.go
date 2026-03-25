package domain

import (
	"time"

	"github.com/google/uuid"
)

// AccountStatus represents the account moderation status. [11-safety §12.2]
type AccountStatus string

const (
	AccountStatusActive    AccountStatus = "active"
	AccountStatusSuspended AccountStatus = "suspended"
	AccountStatusBanned    AccountStatus = "banned"
)

// AccountSuspendedEvent is emitted when an account is suspended. [11-safety §14]
type AccountSuspendedEvent struct {
	FamilyID  uuid.UUID
	AdminID   uuid.UUID
	Reason    string
	ExpiresAt time.Time
}

// AccountModerationState is the aggregate root for account moderation state.
// All fields are unexported; state transitions happen via methods only. [ARCH §4.5]
type AccountModerationState struct {
	familyID            uuid.UUID
	status              AccountStatus
	suspendedAt         *time.Time
	suspensionExpiresAt *time.Time
	suspensionReason    *string
	bannedAt            *time.Time
	banReason           *string
	lastActionID        *uuid.UUID
	createdAt           time.Time
	updatedAt           time.Time
}

// NewAccountModerationState creates a new account state defaulting to active.
func NewAccountModerationState(familyID uuid.UUID) *AccountModerationState {
	now := time.Now().UTC()
	return &AccountModerationState{
		familyID:  familyID,
		status:    AccountStatusActive,
		createdAt: now,
		updatedAt: now,
	}
}

// AccountStateFromPersistence reconstructs an AccountModerationState from persisted data.
func AccountStateFromPersistence(
	familyID uuid.UUID,
	status AccountStatus,
	suspendedAt, suspensionExpiresAt *time.Time,
	suspensionReason *string,
	bannedAt *time.Time,
	banReason *string,
	lastActionID *uuid.UUID,
	createdAt, updatedAt time.Time,
) *AccountModerationState {
	return &AccountModerationState{
		familyID:            familyID,
		status:              status,
		suspendedAt:         suspendedAt,
		suspensionExpiresAt: suspensionExpiresAt,
		suspensionReason:    suspensionReason,
		bannedAt:            bannedAt,
		banReason:           banReason,
		lastActionID:        lastActionID,
		createdAt:           createdAt,
		updatedAt:           updatedAt,
	}
}

// ─── Queries ─────────────────────────────────────────────────────────

func (a *AccountModerationState) FamilyID() uuid.UUID           { return a.familyID }
func (a *AccountModerationState) Status() AccountStatus         { return a.status }
func (a *AccountModerationState) SuspendedAt() *time.Time       { return a.suspendedAt }
func (a *AccountModerationState) SuspensionExpiresAt() *time.Time { return a.suspensionExpiresAt }
func (a *AccountModerationState) SuspensionReason() *string     { return a.suspensionReason }
func (a *AccountModerationState) BannedAt() *time.Time          { return a.bannedAt }
func (a *AccountModerationState) BanReason() *string            { return a.banReason }
func (a *AccountModerationState) LastActionID() *uuid.UUID      { return a.lastActionID }

// ─── State Transitions ──────────────────────────────────────────────

// Suspend transitions active → suspended. Returns AccountSuspendedEvent on success. [11-safety §12.2]
func (a *AccountModerationState) Suspend(adminID uuid.UUID, reason string, days int32) (*AccountSuspendedEvent, error) {
	if a.status == AccountStatusBanned {
		return nil, ErrAccountBanned
	}

	now := time.Now().UTC()
	expiresAt := now.AddDate(0, 0, int(days))

	a.status = AccountStatusSuspended
	a.suspendedAt = &now
	a.suspensionExpiresAt = &expiresAt
	a.suspensionReason = &reason
	a.updatedAt = now

	return &AccountSuspendedEvent{
		FamilyID:  a.familyID,
		AdminID:   adminID,
		Reason:    reason,
		ExpiresAt: expiresAt,
	}, nil
}

// Ban transitions active|suspended → banned. Clears suspension fields. [11-safety §12.2]
func (a *AccountModerationState) Ban(reason string) error {
	if a.status == AccountStatusBanned {
		return ErrAccountBanned
	}

	now := time.Now().UTC()
	a.status = AccountStatusBanned
	a.bannedAt = &now
	a.banReason = &reason
	// Clear suspension fields.
	a.suspendedAt = nil
	a.suspensionExpiresAt = nil
	a.suspensionReason = nil
	a.updatedAt = now
	return nil
}

// LiftSuspension transitions suspended → active. [11-safety §12.2]
func (a *AccountModerationState) LiftSuspension() error {
	if a.status != AccountStatusSuspended {
		return ErrInvalidActionType
	}

	now := time.Now().UTC()
	a.status = AccountStatusActive
	a.suspendedAt = nil
	a.suspensionExpiresAt = nil
	a.suspensionReason = nil
	a.updatedAt = now
	return nil
}

// CheckExpiry checks if a suspension has expired and transitions to active if so. [11-safety §12.5]
// Returns true if the suspension was expired and status was transitioned.
func (a *AccountModerationState) CheckExpiry() bool {
	if a.status != AccountStatusSuspended || a.suspensionExpiresAt == nil {
		return false
	}

	if time.Now().After(*a.suspensionExpiresAt) {
		now := time.Now().UTC()
		a.status = AccountStatusActive
		a.suspendedAt = nil
		a.suspensionExpiresAt = nil
		a.suspensionReason = nil
		a.updatedAt = now
		return true
	}

	return false
}

// Unban transitions banned → active. CSAM bans are NOT appealable. [11-safety §10.6]
func (a *AccountModerationState) Unban() error {
	if a.status != AccountStatusBanned {
		return ErrInvalidActionType
	}

	if a.banReason != nil && *a.banReason == "csam_violation" {
		return ErrCsamBanNotAppealable
	}

	now := time.Now().UTC()
	a.status = AccountStatusActive
	a.bannedAt = nil
	a.banReason = nil
	a.updatedAt = now
	return nil
}

// SetLastActionID sets the last moderation action ID.
func (a *AccountModerationState) SetLastActionID(actionID uuid.UUID) {
	a.lastActionID = &actionID
}
