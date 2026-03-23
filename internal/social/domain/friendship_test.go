package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestValidateSendRequest(t *testing.T) {
	familyA := uuid.New()
	familyB := uuid.New()
	pending := FriendshipStatusPending
	accepted := FriendshipStatusAccepted

	tests := []struct {
		name           string
		requester      uuid.UUID
		accepter       uuid.UUID
		existingStatus *string
		isBlocked      bool
		wantErr        error
	}{
		{"valid request", familyA, familyB, nil, false, nil},
		{"cannot friend self", familyA, familyA, nil, false, ErrCannotFriendSelf},
		{"blocked by target", familyA, familyB, nil, true, ErrBlockedByTarget},
		{"pending request exists", familyA, familyB, &pending, false, ErrFriendRequestPending},
		{"already friends", familyA, familyB, &accepted, false, ErrAlreadyFriends},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSendRequest(tt.requester, tt.accepter, tt.existingStatus, tt.isBlocked)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateAcceptRequest(t *testing.T) {
	accepter := uuid.New()
	other := uuid.New()

	tests := []struct {
		name    string
		actor   uuid.UUID
		status  string
		wantErr error
	}{
		{"valid accept", accepter, FriendshipStatusPending, nil},
		{"not the accepter", other, FriendshipStatusPending, ErrNotAccepter},
		{"already accepted", accepter, FriendshipStatusAccepted, ErrFriendshipNotPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAcceptRequest(tt.actor, accepter, tt.status)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateRejectRequest(t *testing.T) {
	accepter := uuid.New()
	other := uuid.New()

	tests := []struct {
		name    string
		actor   uuid.UUID
		status  string
		wantErr error
	}{
		{"valid reject", accepter, FriendshipStatusPending, nil},
		{"not the accepter", other, FriendshipStatusPending, ErrNotAccepter},
		{"not pending", accepter, FriendshipStatusAccepted, ErrFriendshipNotPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRejectRequest(tt.actor, accepter, tt.status)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateBlock(t *testing.T) {
	familyA := uuid.New()
	familyB := uuid.New()

	tests := []struct {
		name           string
		blocker        uuid.UUID
		blocked        uuid.UUID
		alreadyBlocked bool
		wantErr        error
	}{
		{"valid block", familyA, familyB, false, nil},
		{"cannot block self", familyA, familyA, false, ErrCannotBlockSelf},
		{"already blocked", familyA, familyB, true, ErrAlreadyBlocked},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBlock(tt.blocker, tt.blocked, tt.alreadyBlocked)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}
