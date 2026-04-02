package social

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Service Interface ───────────────────────────────────────────────────────

// SocialService defines all use cases for the social domain. [05-social §5]
// CQRS: commands modify state; queries are read-only. [ARCH §4.7]
type SocialService interface {
	// ─── Profile Commands ────────────────────────────────────────────────
	CreateProfile(ctx context.Context, familyID uuid.UUID) error
	UpdateProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateProfileCommand) (*ProfileResponse, error)

	// ─── Profile Queries ────────────────────────────────────────────────
	GetOwnProfile(ctx context.Context, scope *shared.FamilyScope) (*ProfileResponse, error)
	GetFamilyProfile(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*ProfileResponse, error)

	// ─── Friend Commands ────────────────────────────────────────────────
	SendFriendRequest(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*FriendshipResponse, error)
	AcceptFriendRequest(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) (*FriendshipResponse, error)
	RejectFriendRequest(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) error
	Unfriend(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error
	BlockFamily(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error
	UnblockFamily(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error

	// ─── Friend Queries ─────────────────────────────────────────────────
	ListFriends(ctx context.Context, scope *shared.FamilyScope, cursor *uuid.UUID, limit int) ([]FriendResponse, error)
	ListIncomingRequests(ctx context.Context, scope *shared.FamilyScope, offset, limit int) ([]FriendRequestResponse, error)
	ListOutgoingRequests(ctx context.Context, scope *shared.FamilyScope, offset, limit int) ([]FriendRequestResponse, error)
	ListBlocks(ctx context.Context, scope *shared.FamilyScope) ([]BlockedFamilyResponse, error)

	// ─── Post Commands ──────────────────────────────────────────────────
	CreatePost(ctx context.Context, auth *shared.AuthContext, cmd CreatePostCommand) (*PostResponse, error)
	UpdatePost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd UpdatePostCommand) (*PostResponse, error)
	DeletePost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) error
	LikePost(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error
	UnlikePost(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error

	// ─── Post / Feed Queries ────────────────────────────────────────────
	GetPost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) (*PostDetailResponse, error)
	GetFeed(ctx context.Context, auth *shared.AuthContext, offset, limit int) (*FeedResponse, error)

	// ─── Comment Commands ───────────────────────────────────────────────
	CreateComment(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd CreateCommentCommand) (*CommentResponse, error)
	DeleteComment(ctx context.Context, auth *shared.AuthContext, commentID uuid.UUID) error

	// ─── Comment Queries ────────────────────────────────────────────────
	ListComments(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) ([]CommentResponse, error)

	// ─── Messaging Commands ─────────────────────────────────────────────
	CreateConversation(ctx context.Context, auth *shared.AuthContext, cmd CreateConversationCommand) (*ConversationResponse, error)
	SendMessage(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, cmd SendMessageCommand) (*MessageResponse, error)
	MarkConversationRead(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error
	DeleteConversation(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error
	ReportMessage(ctx context.Context, auth *shared.AuthContext, messageID uuid.UUID, cmd ReportMessageCommand) error

	// ─── Messaging Queries ──────────────────────────────────────────────
	ListConversations(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]ConversationSummaryResponse, error)
	GetConversationMessages(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, offset, limit int) ([]MessageResponse, error)

	// ─── Group Commands ─────────────────────────────────────────────────
	JoinGroup(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error
	LeaveGroup(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error
	CreateGroup(ctx context.Context, auth *shared.AuthContext, cmd CreateGroupCommand) (*GroupResponse, error)
	UpdateGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, cmd UpdateGroupCommand) (*GroupResponse, error)
	DeleteGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) error
	ApproveMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	RejectMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	BanMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	InviteToGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	PromoteMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error

	// ─── Group Queries ──────────────────────────────────────────────────
	GetGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) (*GroupResponse, error)
	ListMyGroups(ctx context.Context, scope *shared.FamilyScope) ([]GroupResponse, error)
	ListPlatformGroups(ctx context.Context) ([]GroupResponse, error)
	ListGroupMembers(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) ([]GroupMemberResponse, error)
	ListGroupPosts(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, offset, limit int) ([]PostResponse, error)

	// ─── Pinned Post Commands ───────────────────────────────────────────
	PinPost(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, postID uuid.UUID) error
	UnpinPost(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, postID uuid.UUID) error

	// ─── Event Commands ─────────────────────────────────────────────────
	CreateEvent(ctx context.Context, auth *shared.AuthContext, cmd CreateEventCommand) (*EventDetailResponse, error)
	UpdateEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID, cmd UpdateEventCommand) (*EventDetailResponse, error)
	CancelEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) error
	RSVPEvent(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID, cmd RSVPCommand) error
	RemoveRSVP(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID) error

	// ─── Event Queries ──────────────────────────────────────────────────
	GetEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) (*EventDetailResponse, error)
	ListEvents(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]EventDetailResponse, error)
	// ListEventsForDateRange returns visible events within a date range. [17-planning §9.1]
	ListEventsForDateRange(ctx context.Context, auth *shared.AuthContext, start, end time.Time) ([]EventDetailResponse, error)

	// ─── Event Handlers (no auth context) ───────────────────────────────
	HandleFamilyCreated(ctx context.Context, familyID uuid.UUID) error

	// ─── Event Handlers (implementations return nil; full impl deferred to M3) ──
	// Interface signatures present per spec §5. [05-social §5]
	HandleCoParentAdded(ctx context.Context, familyID uuid.UUID, coParentID uuid.UUID) error
	HandleCoParentRemoved(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID) error
	HandleMilestoneAchieved(ctx context.Context, familyID uuid.UUID, milestone MilestoneData) error
	HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error

	// ─── Discovery Queries (Phase 2) ─────────────────────────────────────
	// [05-social §4.2, §15]

	// DiscoverFamilies discovers nearby families with location sharing enabled. [S§7.8]
	DiscoverFamilies(ctx context.Context, scope *shared.FamilyScope, query DiscoverFamiliesQuery) ([]DiscoverableFamilyResponse, error)

	// DiscoverEvents discovers discoverable events by location/methodology. [S§7.7]
	DiscoverEvents(ctx context.Context, scope *shared.FamilyScope, query DiscoverEventsQuery) ([]EventSummaryResponse, error)

	// DiscoverGroups discovers groups by methodology. [S§7.6]
	DiscoverGroups(ctx context.Context, scope *shared.FamilyScope, query DiscoverGroupsQuery) ([]GroupSummaryResponse, error)
}

// FriendRequestResponse is the response for friend request list endpoints. [05-social §8.2]
type FriendRequestResponse struct {
	FriendshipID    uuid.UUID `json:"friendship_id"`
	FamilyID        uuid.UUID `json:"family_id"`
	DisplayName     string    `json:"display_name"`
	ProfilePhotoURL *string   `json:"profile_photo_url,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// BlockedFamilyResponse is the response for block list endpoints. [05-social §8.2]
type BlockedFamilyResponse struct {
	FamilyID    uuid.UUID `json:"family_id"`
	DisplayName string    `json:"display_name"`
	BlockedAt   time.Time `json:"blocked_at"`
}

// ─── Repository Interfaces ───────────────────────────────────────────────────
// Each repo maps to one soc_ table. Methods marked CROSS-FAMILY require
// BypassRLSTransaction or unscoped access. [CODING §2.4]

// ProfileRepository provides persistence for soc_profiles.
type ProfileRepository interface {
	Create(ctx context.Context, profile *Profile) error
	FindByFamilyID(ctx context.Context, familyID uuid.UUID) (*Profile, error)
	Update(ctx context.Context, profile *Profile) error
}

// FriendshipRepository provides persistence for soc_friendships.
type FriendshipRepository interface {
	Create(ctx context.Context, friendship *Friendship) error
	// CROSS-FAMILY: finds friendship between two families (checks both directions).
	FindBetween(ctx context.Context, familyA, familyB uuid.UUID) (*Friendship, error)
	FindByID(ctx context.Context, id uuid.UUID) (*Friendship, error)
	Update(ctx context.Context, friendship *Friendship) error
	Delete(ctx context.Context, id uuid.UUID) error
	// CROSS-FAMILY: lists accepted friends for a family (paginated).
	ListFriends(ctx context.Context, familyID uuid.UUID, offset, limit int) ([]uuid.UUID, error)
	// CROSS-FAMILY: lists ALL accepted friend family IDs (for feed fan-out). [05-social §6]
	ListFriendFamilyIDs(ctx context.Context, familyID uuid.UUID) ([]uuid.UUID, error)
	// CROSS-FAMILY: lists friends using cursor pagination (cursor = last friendship UUID).
	ListFriendsCursor(ctx context.Context, familyID uuid.UUID, cursor *uuid.UUID, limit int) ([]Friendship, error)
	// CROSS-FAMILY: lists incoming pending requests for a family (paginated).
	ListIncoming(ctx context.Context, familyID uuid.UUID, offset, limit int) ([]Friendship, error)
	// CROSS-FAMILY: lists outgoing pending requests for a family (paginated).
	ListOutgoing(ctx context.Context, familyID uuid.UUID, offset, limit int) ([]Friendship, error)
	// CROSS-FAMILY: checks if two families are friends (accepted).
	AreFriends(ctx context.Context, familyA, familyB uuid.UUID) (bool, error)
	// DeleteBetween removes friendship between two families (for unfriend/block).
	DeleteBetween(ctx context.Context, familyA, familyB uuid.UUID) error
}

// BlockRepository provides persistence for soc_blocks.
type BlockRepository interface {
	Create(ctx context.Context, block *Block) error
	// CROSS-FAMILY: checks if either family has blocked the other.
	IsEitherBlocked(ctx context.Context, familyA, familyB uuid.UUID) (bool, error)
	// IsBlocked checks if blockerID has blocked blockedID (one direction).
	IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error)
	Delete(ctx context.Context, blockerID, blockedID uuid.UUID) error
	ListByBlocker(ctx context.Context, blockerID uuid.UUID) ([]Block, error)
}

// PostRepository provides persistence for soc_posts.
type PostRepository interface {
	Create(ctx context.Context, post *Post) error
	FindByID(ctx context.Context, id uuid.UUID) (*Post, error)
	Update(ctx context.Context, post *Post) error
	Delete(ctx context.Context, id uuid.UUID) error
	// CROSS-FAMILY: lists posts by multiple family IDs for feed.
	ListByFamilyIDs(ctx context.Context, familyIDs []uuid.UUID, offset, limit int) ([]Post, error)
	// ListFriendsPosts returns recent posts from friends for feed fallback. [05-social §6]
	ListFriendsPosts(ctx context.Context, familyID uuid.UUID, friendIDs []uuid.UUID, offset, limit int) ([]Post, error)
	// FindByIDs retrieves posts by a list of IDs (for Redis feed resolution).
	FindByIDs(ctx context.Context, ids []uuid.UUID) ([]Post, error)
	// ListByGroup lists posts in a specific group.
	ListByGroup(ctx context.Context, groupID uuid.UUID, offset, limit int) ([]Post, error)
	IncrementLikes(ctx context.Context, id uuid.UUID) error
	DecrementLikes(ctx context.Context, id uuid.UUID) error
	IncrementComments(ctx context.Context, id uuid.UUID) error
	DecrementComments(ctx context.Context, id uuid.UUID) error
}

// CommentRepository provides persistence for soc_comments.
type CommentRepository interface {
	Create(ctx context.Context, comment *Comment) error
	FindByID(ctx context.Context, id uuid.UUID) (*Comment, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByPost(ctx context.Context, postID uuid.UUID) ([]Comment, error)
}

// PostLikeRepository provides persistence for soc_post_likes.
type PostLikeRepository interface {
	Create(ctx context.Context, like *PostLike) error
	Delete(ctx context.Context, postID, familyID uuid.UUID) error
	Exists(ctx context.Context, postID, familyID uuid.UUID) (bool, error)
	// ListByPostAndFamily returns whether the family has liked each post in the list.
	ListByPostIDs(ctx context.Context, postIDs []uuid.UUID, familyID uuid.UUID) (map[uuid.UUID]bool, error)
}

// ConversationRepository provides persistence for soc_conversations.
type ConversationRepository interface {
	Create(ctx context.Context, conv *Conversation) error
	FindByID(ctx context.Context, id uuid.UUID) (*Conversation, error)
	// CROSS-FAMILY: lists conversations for a parent (paginated).
	ListByParent(ctx context.Context, parentID uuid.UUID, offset, limit int) ([]Conversation, error)
}

// ConversationParticipantRepository provides persistence for soc_conversation_participants.
type ConversationParticipantRepository interface {
	Create(ctx context.Context, participant *ConversationParticipant) error
	// CROSS-FAMILY: lists participants for a conversation.
	ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]ConversationParticipant, error)
	// IsParticipant checks if a parent is part of a conversation.
	IsParticipant(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) (bool, error)
	// FindBetweenParents finds an existing conversation between two parents.
	// CROSS-FAMILY: used for create-or-get conversation semantics.
	FindBetweenParents(ctx context.Context, parentA, parentB uuid.UUID) (*uuid.UUID, error)
	UpdateLastRead(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error
	SoftDelete(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error
	// ClearDeletedAt restores a soft-deleted participant (new message restores conversation). [05-social §4.1]
	ClearDeletedAt(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error
}

// MessageRepository provides persistence for soc_messages.
type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	FindByID(ctx context.Context, id uuid.UUID) (*Message, error)
	// CROSS-FAMILY: lists messages in a conversation.
	ListByConversation(ctx context.Context, conversationID uuid.UUID, offset, limit int) ([]Message, error)
	// LastByConversation returns the most recent message in a conversation.
	LastByConversation(ctx context.Context, conversationID uuid.UUID) (*Message, error)
	// CountUnread counts messages after lastReadAt.
	CountUnread(ctx context.Context, conversationID uuid.UUID, lastReadAt *time.Time) (int, error)
}

// GroupRepository provides persistence for soc_groups.
type GroupRepository interface {
	Create(ctx context.Context, group *Group) error
	FindByID(ctx context.Context, id uuid.UUID) (*Group, error)
	Update(ctx context.Context, group *Group) error
	Delete(ctx context.Context, id uuid.UUID) error
	ListPlatform(ctx context.Context) ([]Group, error)
	// ListByMethodology lists groups tagged with a methodology slug. Used for discovery. [05-social §15]
	ListByMethodology(ctx context.Context, methodologySlug string) ([]Group, error)
	IncrementMemberCount(ctx context.Context, id uuid.UUID) error
	DecrementMemberCount(ctx context.Context, id uuid.UUID) error
}

// GroupMemberRepository provides persistence for soc_group_members.
type GroupMemberRepository interface {
	Create(ctx context.Context, member *GroupMember) error
	FindByGroupAndFamily(ctx context.Context, groupID, familyID uuid.UUID) (*GroupMember, error)
	Update(ctx context.Context, member *GroupMember) error
	Delete(ctx context.Context, groupID, familyID uuid.UUID) error
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]GroupMember, error)
	// ListGroupsByFamily returns group IDs where the family is an active member.
	ListGroupsByFamily(ctx context.Context, familyID uuid.UUID) ([]uuid.UUID, error)
	// IsMember checks if a family is an active member of a group.
	IsMember(ctx context.Context, groupID, familyID uuid.UUID) (bool, error)
}

// EventRepository provides persistence for soc_events.
type EventRepository interface {
	Create(ctx context.Context, event *Event) error
	FindByID(ctx context.Context, id uuid.UUID) (*Event, error)
	Update(ctx context.Context, event *Event) error
	// CROSS-FAMILY: lists events visible to a family (friends, groups, discoverable), paginated.
	ListVisible(ctx context.Context, familyID uuid.UUID, friendIDs []uuid.UUID, groupIDs []uuid.UUID, offset, limit int) ([]Event, error)
	// CROSS-FAMILY: lists visible events within a date range (for calendar integration). [17-planning §9.1]
	ListVisibleInDateRange(ctx context.Context, familyID uuid.UUID, friendIDs []uuid.UUID, groupIDs []uuid.UUID, start, end time.Time) ([]Event, error)
	// ListDiscoverable lists events with 'discoverable' visibility, filtered by methodology/location. [05-social §15]
	ListDiscoverable(ctx context.Context, methodologySlug *string, locationRegion *string) ([]Event, error)
	IncrementAttendeeCount(ctx context.Context, id uuid.UUID) error
	DecrementAttendeeCount(ctx context.Context, id uuid.UUID) error
}

// EventRSVPRepository provides persistence for soc_event_rsvps.
type EventRSVPRepository interface {
	Create(ctx context.Context, rsvp *EventRSVP) error
	FindByEventAndFamily(ctx context.Context, eventID, familyID uuid.UUID) (*EventRSVP, error)
	Update(ctx context.Context, rsvp *EventRSVP) error
	Delete(ctx context.Context, eventID, familyID uuid.UUID) error
	// CountGoing returns the number of "going" RSVPs for an event. Used for capacity enforcement.
	CountGoing(ctx context.Context, eventID uuid.UUID) (int, error)
	// ListByEvent returns all RSVPs for an event.
	ListByEvent(ctx context.Context, eventID uuid.UUID) ([]EventRSVP, error)
	// ListGoingFamilyIDs returns family IDs with "going" status. Used by EventCancelled event.
	ListGoingFamilyIDs(ctx context.Context, eventID uuid.UUID) ([]uuid.UUID, error)
}

// PinnedPostRepository provides persistence for soc_pinned_posts. [05-social §4.2]
type PinnedPostRepository interface {
	Create(ctx context.Context, pin *PinnedPost) error
	Delete(ctx context.Context, groupID, postID uuid.UUID) error
	FindByGroupAndPost(ctx context.Context, groupID, postID uuid.UUID) (*PinnedPost, error)
	ListByGroup(ctx context.Context, groupID uuid.UUID) ([]PinnedPost, error)
}

// ─── Consumer-Defined Cross-Domain Interfaces ────────────────────────────────
// Narrow interfaces for cross-domain service calls. Adapters wired in main.go. [ARCH §4.2]

// IamServiceForSocial is the subset of iam::IamService that social:: needs.
type IamServiceForSocial interface {
	GetFamilyDisplayName(ctx context.Context, familyID uuid.UUID) (string, error)
	GetParentDisplayName(ctx context.Context, parentID uuid.UUID) (string, error)
	GetFamilyInfo(ctx context.Context, familyID uuid.UUID) (*SocialFamilyInfo, error)
	GetParentInfo(ctx context.Context, parentID uuid.UUID) (*SocialParentInfo, error)
}

// MethodServiceForSocial is the subset of method::MethodologyService that social:: needs.
type MethodServiceForSocial interface {
	GetMethodologyDisplayName(ctx context.Context, slug string) (string, error)
}
