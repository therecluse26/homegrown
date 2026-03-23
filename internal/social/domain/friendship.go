package domain

import "github.com/google/uuid"

// Friendship status constants. [05-social §10]
const (
	FriendshipStatusPending  = "pending"
	FriendshipStatusAccepted = "accepted"
)

// ValidateSendRequest validates whether a friend request can be sent.
// Pure function — no database access. Caller provides all state. [05-social §10]
func ValidateSendRequest(requesterID, accepterID uuid.UUID, existingStatus *string, isBlocked bool) error {
	if requesterID == accepterID {
		return ErrCannotFriendSelf
	}
	if isBlocked {
		// Silent blocking: blocked interactions return "not found" to the blocked party.
		return ErrBlockedByTarget
	}
	if existingStatus != nil {
		switch *existingStatus {
		case FriendshipStatusPending:
			return ErrFriendRequestPending
		case FriendshipStatusAccepted:
			return ErrAlreadyFriends
		}
	}
	return nil
}

// ValidateAcceptRequest validates whether a friend request can be accepted.
// Only the accepter can accept; the friendship must be pending. [05-social §10]
func ValidateAcceptRequest(actorFamilyID, accepterFamilyID uuid.UUID, status string) error {
	if actorFamilyID != accepterFamilyID {
		return ErrNotAccepter
	}
	if status != FriendshipStatusPending {
		return ErrFriendshipNotPending
	}
	return nil
}

// ValidateRejectRequest validates whether a friend request can be rejected.
// Only the accepter can reject; the friendship must be pending. [05-social §10]
func ValidateRejectRequest(actorFamilyID, accepterFamilyID uuid.UUID, status string) error {
	if actorFamilyID != accepterFamilyID {
		return ErrNotAccepter
	}
	if status != FriendshipStatusPending {
		return ErrFriendshipNotPending
	}
	return nil
}

// ValidateBlock validates whether a block can be created.
// Cannot block self; cannot re-block if already blocked. [05-social §10]
func ValidateBlock(blockerID, blockedID uuid.UUID, alreadyBlocked bool) error {
	if blockerID == blockedID {
		return ErrCannotBlockSelf
	}
	if alreadyBlocked {
		return ErrAlreadyBlocked
	}
	return nil
}
