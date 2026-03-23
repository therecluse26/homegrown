package domain

// Group role and status constants. [05-social §13]
const (
	GroupRoleMember    = "member"
	GroupRoleModerator = "moderator"
	GroupRoleOwner     = "owner"
)

const (
	GroupMemberStatusActive  = "active"
	GroupMemberStatusPending = "pending"
	GroupMemberStatusInvited = "invited"
	GroupMemberStatusBanned  = "banned"
)

// Join policy constants. [05-social §13]
const (
	JoinPolicyOpen          = "open"
	JoinPolicyRequestToJoin = "request_to_join"
	JoinPolicyInviteOnly    = "invite_only"
)

// Group type constants.
const (
	GroupTypePlatform    = "platform"
	GroupTypeUserCreated = "user_created"
)

// ResolveJoinAction determines the resulting member status when a family tries to join a group.
// open → active, request_to_join → pending, invite_only → error. [05-social §13]
func ResolveJoinAction(joinPolicy string) (string, error) {
	switch joinPolicy {
	case JoinPolicyOpen:
		return GroupMemberStatusActive, nil
	case JoinPolicyRequestToJoin:
		return GroupMemberStatusPending, nil
	case JoinPolicyInviteOnly:
		return "", ErrGroupInviteOnly
	default:
		return "", ErrGroupInviteOnly
	}
}

// CanModerate returns true if the given role has moderation privileges.
// Moderators and owners can moderate. [05-social §13]
func CanModerate(role string) bool {
	return role == GroupRoleModerator || role == GroupRoleOwner
}

// ValidateLeave validates whether a member can leave a group.
// The owner cannot leave without first transferring ownership. [05-social §13]
func ValidateLeave(role string) error {
	if role == GroupRoleOwner {
		return ErrOwnerCannotLeave
	}
	return nil
}

// ValidateBan validates whether a member can be banned.
// Cannot ban the group owner. [05-social §13]
func ValidateBan(targetRole string) error {
	if targetRole == GroupRoleOwner {
		return ErrCannotBanOwner
	}
	return nil
}

// ValidatePromotion validates whether a member can be promoted to moderator.
// Only active members can be promoted; owners are already at the highest role. [05-social §13]
func ValidatePromotion(targetRole, targetStatus string) error {
	if targetRole == GroupRoleOwner {
		return ErrCannotPromoteOwner
	}
	if targetStatus != GroupMemberStatusActive {
		return ErrMemberNotActive
	}
	return nil
}
