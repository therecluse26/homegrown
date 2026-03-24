package domain

import (
	"time"

	"github.com/google/uuid"
)

// Post type constants. [05-social §7.2.1]
const (
	PostTypeText              = "text"
	PostTypePhoto             = "photo"
	PostTypeMilestone         = "milestone"
	PostTypeEventShare        = "event_share"
	PostTypeMarketplaceReview = "marketplace_review"
	PostTypeResourceShare     = "resource_share"
)

// ValidatePostCreate validates a post creation request.
// Text posts require content; photo posts require attachments. [05-social §7.2]
func ValidatePostCreate(postType string, content *string, hasAttachments bool) error {
	switch postType {
	case PostTypeText, PostTypeMilestone, PostTypeEventShare, PostTypeMarketplaceReview, PostTypeResourceShare:
		if content == nil || *content == "" {
			return ErrPostContentRequired
		}
	case PostTypePhoto:
		if !hasAttachments {
			return ErrPostContentRequired
		}
	default:
		return ErrInvalidPostType
	}
	return nil
}

// ResolvePostVisibility determines the visibility of a post based on group_id.
// Posts with a group_id get "group" visibility; all others get "friends". [05-social §9]
func ResolvePostVisibility(groupID *uuid.UUID) string {
	if groupID != nil {
		return VisibilityGroup
	}
	return VisibilityFriends
}

// ValidateCommentThread validates that a parent comment (if provided) is a top-level comment.
// Only one level of threading is allowed. [05-social §7.3]
func ValidateCommentThread(parentCommentHasParent bool) error {
	if parentCommentHasParent {
		return ErrNestedReplyNotAllowed
	}
	return nil
}

// ValidateCommentSamePost checks that the parent comment belongs to the same post.
// [05-social §4.1]: "parent_comment_id must reference a top-level comment on the same post"
func ValidateCommentSamePost(parentPostID, targetPostID uuid.UUID) error {
	if parentPostID != targetPostID {
		return ErrCommentCrossPost
	}
	return nil
}

// ValidateEventDate checks that the event date is in the future. [05-social §4.1]
func ValidateEventDate(eventDate time.Time) error {
	if !eventDate.After(time.Now()) {
		return ErrEventDatePast
	}
	return nil
}

// ValidateEventGroupVisibility checks that group visibility events have a group_id. [05-social §4.1]
func ValidateEventGroupVisibility(visibility string, groupID *uuid.UUID) error {
	if visibility == VisibilityGroup && groupID == nil {
		return ErrEventGroupRequired
	}
	return nil
}
