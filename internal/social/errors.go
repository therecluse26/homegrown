package social

import (
	"errors"
	"net/http"

	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social/domain"
)

// SocialError wraps a sentinel error with optional context. [CODING §2.2]
type SocialError struct {
	Err error
}

func (e *SocialError) Error() string { return e.Err.Error() }
func (e *SocialError) Unwrap() error { return e.Err }

// toAppError maps a SocialError sentinel to an *shared.AppError with the correct
// HTTP status code. Called by mapSocialError in handler.go. [05-social §12.1]
func (e *SocialError) toAppError() *shared.AppError {
	switch {
	// ─── Profile ─────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrProfileNotFound):
		return &shared.AppError{Code: "profile_not_found", Message: "Profile not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrProfileExists):
		return &shared.AppError{Code: "profile_exists", Message: "Profile already exists", StatusCode: http.StatusConflict}

	// ─── Friendship ─────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrCannotFriendSelf):
		return &shared.AppError{Code: "cannot_friend_self", Message: "Cannot send a friend request to yourself", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrAlreadyFriends):
		return &shared.AppError{Code: "already_friends", Message: "Already friends", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrFriendRequestPending):
		return &shared.AppError{Code: "friend_request_pending", Message: "Friend request already pending", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrFriendRequestNotFound):
		return &shared.AppError{Code: "friend_request_not_found", Message: "Friend request not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrNotFriends):
		return &shared.AppError{Code: "not_friends", Message: "Not friends", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrNotAccepter):
		return &shared.AppError{Code: "not_accepter", Message: "Only the recipient can respond to a friend request", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrFriendshipNotPending):
		return &shared.AppError{Code: "friendship_not_pending", Message: "Friendship is not in pending state", StatusCode: http.StatusConflict}

	// ─── Block (silent — always 404) ────────────────────────────
	case errors.Is(e.Err, domain.ErrCannotBlockSelf):
		return &shared.AppError{Code: "cannot_block_self", Message: "Cannot block yourself", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrAlreadyBlocked):
		return &shared.AppError{Code: "already_blocked", Message: "Already blocked", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrNotBlocked):
		return &shared.AppError{Code: "not_blocked", Message: "Not blocked", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrBlockedByTarget):
		// Silent blocking: return 404 so the blocked party doesn't know they're blocked. [05-social §16]
		return &shared.AppError{Code: "not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrBlockedByUser):
		return &shared.AppError{Code: "not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}

	// ─── Visibility ─────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrContentNotVisible):
		return &shared.AppError{Code: "not_found", Message: "Resource not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrNotGroupMember):
		return &shared.AppError{Code: "not_group_member", Message: "Not a group member", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrNotConversationParticipant):
		return &shared.AppError{Code: "not_participant", Message: "Not a conversation participant", StatusCode: http.StatusForbidden}

	// ─── Post ───────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrPostNotFound):
		return &shared.AppError{Code: "post_not_found", Message: "Post not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrInvalidPostType):
		return &shared.AppError{Code: "invalid_post_type", Message: "Invalid post type", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrPostContentRequired):
		return &shared.AppError{Code: "content_required", Message: "Content is required", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrAlreadyLiked):
		return &shared.AppError{Code: "already_liked", Message: "Already liked this post", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrNotLiked):
		return &shared.AppError{Code: "not_liked", Message: "Not liked this post", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrCannotDeletePost):
		return &shared.AppError{Code: "cannot_delete_post", Message: "Can only delete your own posts", StatusCode: http.StatusForbidden}

	// ─── Comment ────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrCommentNotFound):
		return &shared.AppError{Code: "comment_not_found", Message: "Comment not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrNestedReplyNotAllowed):
		return &shared.AppError{Code: "nested_reply", Message: "Only one level of reply threading is allowed", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrCannotDeleteComment):
		return &shared.AppError{Code: "cannot_delete_comment", Message: "Cannot delete this comment", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrCommentCrossPost):
		return &shared.AppError{Code: "comment_cross_post", Message: "Parent comment must belong to the same post", StatusCode: http.StatusUnprocessableEntity}

	// ─── Group ──────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrGroupNotFound):
		return &shared.AppError{Code: "group_not_found", Message: "Group not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrAlreadyGroupMember):
		return &shared.AppError{Code: "already_group_member", Message: "Already a group member", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrGroupMemberNotFound):
		return &shared.AppError{Code: "group_member_not_found", Message: "Group membership not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrGroupInviteOnly):
		return &shared.AppError{Code: "group_invite_only", Message: "Group is invite only", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrOwnerCannotLeave):
		return &shared.AppError{Code: "owner_cannot_leave", Message: "Owner cannot leave without transferring ownership", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrInsufficientGroupRole):
		return &shared.AppError{Code: "insufficient_role", Message: "Insufficient role for this action", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrCannotBanOwner):
		return &shared.AppError{Code: "cannot_ban_owner", Message: "Cannot ban the group owner", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrMemberBanned):
		return &shared.AppError{Code: "member_banned", Message: "Banned from this group", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrMemberPending):
		return &shared.AppError{Code: "member_pending", Message: "Membership request is pending", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrCannotDeletePlatformGroup):
		return &shared.AppError{Code: "cannot_delete_platform_group", Message: "Platform groups cannot be deleted", StatusCode: http.StatusForbidden}

	// ─── Messaging ──────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrConversationNotFound):
		return &shared.AppError{Code: "conversation_not_found", Message: "Conversation not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrNotFriendsForDM):
		return &shared.AppError{Code: "not_friends_for_dm", Message: "Must be friends to start a conversation", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrMessageNotFound):
		return &shared.AppError{Code: "message_not_found", Message: "Message not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrCannotMessageSelf):
		return &shared.AppError{Code: "cannot_message_self", Message: "Cannot create a conversation with yourself", StatusCode: http.StatusUnprocessableEntity}

	// ─── Events ─────────────────────────────────────────────────
	case errors.Is(e.Err, domain.ErrEventNotFound):
		return &shared.AppError{Code: "event_not_found", Message: "Event not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrEventCancelled):
		return &shared.AppError{Code: "event_cancelled", Message: "Event has been cancelled", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrEventAtCapacity):
		return &shared.AppError{Code: "event_at_capacity", Message: "Event is at capacity", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrAlreadyRSVPd):
		return &shared.AppError{Code: "already_rsvpd", Message: "Already RSVP'd to this event", StatusCode: http.StatusConflict}
	case errors.Is(e.Err, domain.ErrRSVPNotFound):
		return &shared.AppError{Code: "rsvp_not_found", Message: "RSVP not found", StatusCode: http.StatusNotFound}
	case errors.Is(e.Err, domain.ErrCannotModifyEvent):
		return &shared.AppError{Code: "cannot_modify_event", Message: "Can only modify your own events", StatusCode: http.StatusForbidden}
	case errors.Is(e.Err, domain.ErrEventDatePast):
		return &shared.AppError{Code: "event_date_past", Message: "Event date must be in the future", StatusCode: http.StatusUnprocessableEntity}
	case errors.Is(e.Err, domain.ErrEventGroupRequired):
		return &shared.AppError{Code: "event_group_required", Message: "Group visibility requires group_id", StatusCode: http.StatusUnprocessableEntity}

	// ─── Privacy / Validation ───────────────────────────────────
	case errors.Is(e.Err, domain.ErrInvalidPrivacySettings):
		return &shared.AppError{Code: "invalid_privacy_settings", Message: "Privacy settings values must be 'friends' or 'hidden'", StatusCode: http.StatusUnprocessableEntity}

	default:
		return shared.ErrInternal(e)
	}
}
