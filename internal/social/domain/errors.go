package domain

import "errors"

// Sentinel errors for the social domain. [05-social §12]
// Handlers convert these to AppError via mapSocialError(). [§12.1]

// ─── Profile Errors ─────────────────────────────────────────────────────────

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrProfileExists   = errors.New("profile already exists")
)

// ─── Friendship Errors ──────────────────────────────────────────────────────

var (
	ErrCannotFriendSelf       = errors.New("cannot send friend request to yourself")
	ErrAlreadyFriends         = errors.New("already friends")
	ErrFriendRequestPending   = errors.New("friend request already pending")
	ErrFriendRequestNotFound  = errors.New("friend request not found")
	ErrNotFriends             = errors.New("not friends")
	ErrNotAccepter            = errors.New("only the accepter can respond to a friend request")
	ErrFriendshipNotPending   = errors.New("friendship is not in pending state")
)

// ─── Block Errors ───────────────────────────────────────────────────────────

var (
	ErrCannotBlockSelf = errors.New("cannot block yourself")
	ErrAlreadyBlocked  = errors.New("already blocked")
	ErrNotBlocked      = errors.New("not blocked")
	ErrBlockedByTarget = errors.New("blocked by target") // maps to 404 (silent blocking)
	ErrBlockedByUser   = errors.New("blocked by user")   // maps to 404 (silent blocking)
)

// ─── Visibility / Access Errors ─────────────────────────────────────────────

var (
	ErrContentNotVisible    = errors.New("content not visible")
	ErrNotGroupMember       = errors.New("not a group member")
	ErrNotConversationParticipant = errors.New("not a conversation participant")
)

// ─── Post Errors ────────────────────────────────────────────────────────────

var (
	ErrPostNotFound       = errors.New("post not found")
	ErrInvalidPostType    = errors.New("invalid post type")
	ErrPostContentRequired = errors.New("content is required for text posts")
	ErrAlreadyLiked       = errors.New("already liked this post")
	ErrNotLiked           = errors.New("not liked this post")
	ErrCannotDeletePost   = errors.New("can only delete your own posts")
	ErrCannotEditPost     = errors.New("can only edit your own posts")
	ErrPostEditEmpty      = errors.New("update must include content or attachments")
)

// ─── Comment Errors ─────────────────────────────────────────────────────────

var (
	ErrCommentNotFound       = errors.New("comment not found")
	ErrNestedReplyNotAllowed = errors.New("only one level of reply threading is allowed")
	ErrCannotEditComment     = errors.New("can only edit your own comments")
	ErrCannotDeleteComment   = errors.New("can only delete your own comments or comments on your posts")
	ErrCommentCrossPost      = errors.New("parent comment must belong to the same post")
)

// ─── Group Errors ───────────────────────────────────────────────────────────

var (
	ErrGroupNotFound         = errors.New("group not found")
	ErrAlreadyGroupMember    = errors.New("already a group member")
	ErrGroupMemberNotFound   = errors.New("group membership not found")
	ErrGroupInviteOnly       = errors.New("group is invite only")
	ErrOwnerCannotLeave      = errors.New("group owner cannot leave without transferring ownership")
	ErrInsufficientGroupRole = errors.New("insufficient group role for this action")
	ErrCannotBanOwner        = errors.New("cannot ban the group owner")
	ErrMemberBanned          = errors.New("member is banned from this group")
	ErrMemberPending         = errors.New("membership request is pending approval")
	ErrCannotPromoteOwner       = errors.New("cannot promote the group owner")
	ErrMemberNotActive          = errors.New("can only promote active members")
	ErrCannotDeletePlatformGroup = errors.New("cannot delete a platform-managed group")
	ErrPostAlreadyPinned         = errors.New("post is already pinned in this group")
	ErrPinnedPostNotFound        = errors.New("pinned post not found")
	ErrPostNotInGroup            = errors.New("post does not belong to this group")
)

// ─── Messaging Errors ───────────────────────────────────────────────────────

var (
	ErrConversationNotFound = errors.New("conversation not found")
	ErrNotFriendsForDM      = errors.New("must be friends to start a conversation")
	ErrMessageNotFound      = errors.New("message not found")
	ErrCannotMessageSelf    = errors.New("cannot create a conversation with yourself")
)

// ─── Event Errors ───────────────────────────────────────────────────────────

var (
	ErrEventNotFound      = errors.New("event not found")
	ErrEventCancelled     = errors.New("event has been cancelled")
	ErrEventAtCapacity    = errors.New("event is at capacity")
	ErrAlreadyRSVPd       = errors.New("already RSVP'd to this event")
	ErrRSVPNotFound       = errors.New("RSVP not found")
	ErrCannotModifyEvent  = errors.New("can only modify your own events")
	ErrEventDatePast      = errors.New("event date must be in the future")
	ErrEventGroupRequired = errors.New("group visibility requires group_id")
)

// ─── Privacy / Validation Errors ───────────────────────────────────────────

var (
	ErrInvalidPrivacySettings = errors.New("privacy settings values must be 'friends' or 'hidden'")
)
