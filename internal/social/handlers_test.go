package social

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social/domain"
	"github.com/labstack/echo/v4"
)

// ─── Test Helpers ────────────────────────────────────────────────────────────

type echoValidator struct{ v *validator.Validate }

func (cv *echoValidator) Validate(i any) error { return cv.v.Struct(i) }

func newTestEcho() *echo.Echo {
	e := echo.New()
	e.Validator = &echoValidator{v: validator.New()}
	e.HTTPErrorHandler = shared.HTTPErrorHandler
	return e
}

var (
	testParentID = uuid.Must(uuid.NewV7())
	testFamilyID = uuid.Must(uuid.NewV7())
)

func setupSocialRoutes(e *echo.Echo, svc SocialService) {
	auth := e.Group("/v1")
	auth.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			shared.SetAuthContext(c, &shared.AuthContext{
				ParentID:           testParentID,
				FamilyID:           testFamilyID,
				CoppaConsentStatus: "consented",
			})
			return next(c)
		}
	})
	NewHandler(svc, shared.NoopPubSub{}, []string{"http://localhost:5673"}).Register(auth)
}

// ─── Mock SocialService ──────────────────────────────────────────────────────

type mockSocialService struct {
	// Profile
	createProfileFn    func(ctx context.Context, familyID uuid.UUID) error
	updateProfileFn    func(ctx context.Context, scope *shared.FamilyScope, cmd UpdateProfileCommand) (*ProfileResponse, error)
	getOwnProfileFn    func(ctx context.Context, scope *shared.FamilyScope) (*ProfileResponse, error)
	getFamilyProfileFn func(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*ProfileResponse, error)

	// Friends
	sendFriendRequestFn   func(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*FriendshipResponse, error)
	acceptFriendRequestFn func(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) (*FriendshipResponse, error)
	rejectFriendRequestFn func(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) error
	unfriendFn            func(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error
	blockFamilyFn         func(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error
	unblockFamilyFn       func(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error
	listFriendsFn         func(ctx context.Context, scope *shared.FamilyScope, cursor *uuid.UUID, limit int) ([]FriendResponse, error)
	listIncomingFn        func(ctx context.Context, scope *shared.FamilyScope) ([]FriendRequestResponse, error)
	listOutgoingFn        func(ctx context.Context, scope *shared.FamilyScope) ([]FriendRequestResponse, error)
	listBlocksFn          func(ctx context.Context, scope *shared.FamilyScope) ([]BlockedFamilyResponse, error)

	// Posts
	createPostFn func(ctx context.Context, auth *shared.AuthContext, cmd CreatePostCommand) (*PostResponse, error)
	updatePostFn func(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd UpdatePostCommand) (*PostResponse, error)
	deletePostFn func(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) error
	likePostFn   func(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error
	unlikePostFn func(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error
	getPostFn    func(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) (*PostDetailResponse, error)
	getFeedFn    func(ctx context.Context, auth *shared.AuthContext, offset, limit int) (*FeedResponse, error)

	// Comments
	createCommentFn func(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd CreateCommentCommand) (*CommentResponse, error)
	deleteCommentFn func(ctx context.Context, auth *shared.AuthContext, commentID uuid.UUID) error
	listCommentsFn  func(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) ([]CommentResponse, error)

	// Messaging
	createConversationFn     func(ctx context.Context, auth *shared.AuthContext, cmd CreateConversationCommand) (*ConversationResponse, error)
	sendMessageFn            func(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, cmd SendMessageCommand) (*MessageResponse, error)
	markConversationReadFn   func(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error
	deleteConversationFn     func(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error
	reportMessageFn          func(ctx context.Context, auth *shared.AuthContext, messageID uuid.UUID, cmd ReportMessageCommand) error
	listConversationsFn      func(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]ConversationSummaryResponse, error)
	getConversationMessagesFn func(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, offset, limit int) ([]MessageResponse, error)

	// Groups
	joinGroupFn         func(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error
	leaveGroupFn        func(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error
	createGroupFn       func(ctx context.Context, auth *shared.AuthContext, cmd CreateGroupCommand) (*GroupResponse, error)
	updateGroupFn       func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, cmd UpdateGroupCommand) (*GroupResponse, error)
	deleteGroupFn       func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) error
	approveMemberFn     func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	rejectMemberFn      func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	banMemberFn         func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	inviteToGroupFn     func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	promoteMemberFn     func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error
	getGroupFn          func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) (*GroupResponse, error)
	listMyGroupsFn      func(ctx context.Context, scope *shared.FamilyScope) ([]GroupResponse, error)
	listPlatformGroupsFn func(ctx context.Context) ([]GroupResponse, error)
	listGroupMembersFn  func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) ([]GroupMemberResponse, error)
	listGroupPostsFn    func(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, offset, limit int) ([]PostResponse, error)

	// Events
	createEventFn func(ctx context.Context, auth *shared.AuthContext, cmd CreateEventCommand) (*EventDetailResponse, error)
	updateEventFn func(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID, cmd UpdateEventCommand) (*EventDetailResponse, error)
	cancelEventFn func(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) error
	rsvpEventFn   func(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID, cmd RSVPCommand) error
	removeRSVPFn  func(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID) error
	getEventFn    func(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) (*EventDetailResponse, error)
	listEventsFn  func(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]EventDetailResponse, error)

	// Discovery
	discoverFamiliesFn func(ctx context.Context, scope *shared.FamilyScope, query DiscoverFamiliesQuery) ([]DiscoverableFamilyResponse, error)
	discoverEventsFn   func(ctx context.Context, scope *shared.FamilyScope, query DiscoverEventsQuery) ([]EventSummaryResponse, error)
	discoverGroupsFn   func(ctx context.Context, scope *shared.FamilyScope, query DiscoverGroupsQuery) ([]GroupSummaryResponse, error)

	// Event Handlers
	handleFamilyCreatedFn            func(ctx context.Context, familyID uuid.UUID) error
	handleCoParentRemovedFn          func(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID) error
	handleMilestoneAchievedFn        func(ctx context.Context, familyID uuid.UUID, milestone MilestoneData) error
	handleFamilyDeletionScheduledFn  func(ctx context.Context, familyID uuid.UUID) error
}

// ─── Profile ─────────────────────────────────────────────────────────────────

func (m *mockSocialService) CreateProfile(ctx context.Context, familyID uuid.UUID) error {
	if m.createProfileFn != nil {
		return m.createProfileFn(ctx, familyID)
	}
	panic("unexpected call to CreateProfile")
}

func (m *mockSocialService) UpdateProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateProfileCommand) (*ProfileResponse, error) {
	if m.updateProfileFn != nil {
		return m.updateProfileFn(ctx, scope, cmd)
	}
	panic("unexpected call to UpdateProfile")
}

func (m *mockSocialService) GetOwnProfile(ctx context.Context, scope *shared.FamilyScope) (*ProfileResponse, error) {
	if m.getOwnProfileFn != nil {
		return m.getOwnProfileFn(ctx, scope)
	}
	panic("unexpected call to GetOwnProfile")
}

func (m *mockSocialService) GetFamilyProfile(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*ProfileResponse, error) {
	if m.getFamilyProfileFn != nil {
		return m.getFamilyProfileFn(ctx, auth, targetFamilyID)
	}
	panic("unexpected call to GetFamilyProfile")
}

// ─── Friends ─────────────────────────────────────────────────────────────────

func (m *mockSocialService) SendFriendRequest(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*FriendshipResponse, error) {
	if m.sendFriendRequestFn != nil {
		return m.sendFriendRequestFn(ctx, auth, targetFamilyID)
	}
	panic("unexpected call to SendFriendRequest")
}

func (m *mockSocialService) AcceptFriendRequest(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) (*FriendshipResponse, error) {
	if m.acceptFriendRequestFn != nil {
		return m.acceptFriendRequestFn(ctx, auth, friendshipID)
	}
	panic("unexpected call to AcceptFriendRequest")
}

func (m *mockSocialService) RejectFriendRequest(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) error {
	if m.rejectFriendRequestFn != nil {
		return m.rejectFriendRequestFn(ctx, auth, friendshipID)
	}
	panic("unexpected call to RejectFriendRequest")
}

func (m *mockSocialService) Unfriend(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error {
	if m.unfriendFn != nil {
		return m.unfriendFn(ctx, auth, targetFamilyID)
	}
	panic("unexpected call to Unfriend")
}

func (m *mockSocialService) BlockFamily(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error {
	if m.blockFamilyFn != nil {
		return m.blockFamilyFn(ctx, auth, targetFamilyID)
	}
	panic("unexpected call to BlockFamily")
}

func (m *mockSocialService) UnblockFamily(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error {
	if m.unblockFamilyFn != nil {
		return m.unblockFamilyFn(ctx, auth, targetFamilyID)
	}
	panic("unexpected call to UnblockFamily")
}

func (m *mockSocialService) ListFriends(ctx context.Context, scope *shared.FamilyScope, cursor *uuid.UUID, limit int) ([]FriendResponse, error) {
	if m.listFriendsFn != nil {
		return m.listFriendsFn(ctx, scope, cursor, limit)
	}
	panic("unexpected call to ListFriends")
}

func (m *mockSocialService) ListIncomingRequests(ctx context.Context, scope *shared.FamilyScope) ([]FriendRequestResponse, error) {
	if m.listIncomingFn != nil {
		return m.listIncomingFn(ctx, scope)
	}
	panic("unexpected call to ListIncomingRequests")
}

func (m *mockSocialService) ListOutgoingRequests(ctx context.Context, scope *shared.FamilyScope) ([]FriendRequestResponse, error) {
	if m.listOutgoingFn != nil {
		return m.listOutgoingFn(ctx, scope)
	}
	panic("unexpected call to ListOutgoingRequests")
}

func (m *mockSocialService) ListBlocks(ctx context.Context, scope *shared.FamilyScope) ([]BlockedFamilyResponse, error) {
	if m.listBlocksFn != nil {
		return m.listBlocksFn(ctx, scope)
	}
	panic("unexpected call to ListBlocks")
}

// ─── Posts ───────────────────────────────────────────────────────────────────

func (m *mockSocialService) CreatePost(ctx context.Context, auth *shared.AuthContext, cmd CreatePostCommand) (*PostResponse, error) {
	if m.createPostFn != nil {
		return m.createPostFn(ctx, auth, cmd)
	}
	panic("unexpected call to CreatePost")
}

func (m *mockSocialService) UpdatePost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd UpdatePostCommand) (*PostResponse, error) {
	if m.updatePostFn != nil {
		return m.updatePostFn(ctx, auth, postID, cmd)
	}
	panic("unexpected call to UpdatePost")
}

func (m *mockSocialService) DeletePost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) error {
	if m.deletePostFn != nil {
		return m.deletePostFn(ctx, auth, postID)
	}
	panic("unexpected call to DeletePost")
}

func (m *mockSocialService) LikePost(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error {
	if m.likePostFn != nil {
		return m.likePostFn(ctx, scope, postID)
	}
	panic("unexpected call to LikePost")
}

func (m *mockSocialService) UnlikePost(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error {
	if m.unlikePostFn != nil {
		return m.unlikePostFn(ctx, scope, postID)
	}
	panic("unexpected call to UnlikePost")
}

func (m *mockSocialService) GetPost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) (*PostDetailResponse, error) {
	if m.getPostFn != nil {
		return m.getPostFn(ctx, auth, postID)
	}
	panic("unexpected call to GetPost")
}

func (m *mockSocialService) GetFeed(ctx context.Context, auth *shared.AuthContext, offset, limit int) (*FeedResponse, error) {
	if m.getFeedFn != nil {
		return m.getFeedFn(ctx, auth, offset, limit)
	}
	panic("unexpected call to GetFeed")
}

// ─── Comments ───────────────────────────────────────────────────────────────

func (m *mockSocialService) CreateComment(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd CreateCommentCommand) (*CommentResponse, error) {
	if m.createCommentFn != nil {
		return m.createCommentFn(ctx, auth, postID, cmd)
	}
	panic("unexpected call to CreateComment")
}

func (m *mockSocialService) DeleteComment(ctx context.Context, auth *shared.AuthContext, commentID uuid.UUID) error {
	if m.deleteCommentFn != nil {
		return m.deleteCommentFn(ctx, auth, commentID)
	}
	panic("unexpected call to DeleteComment")
}

func (m *mockSocialService) ListComments(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) ([]CommentResponse, error) {
	if m.listCommentsFn != nil {
		return m.listCommentsFn(ctx, auth, postID)
	}
	panic("unexpected call to ListComments")
}

// ─── Messaging ──────────────────────────────────────────────────────────────

func (m *mockSocialService) CreateConversation(ctx context.Context, auth *shared.AuthContext, cmd CreateConversationCommand) (*ConversationResponse, error) {
	if m.createConversationFn != nil {
		return m.createConversationFn(ctx, auth, cmd)
	}
	panic("unexpected call to CreateConversation")
}

func (m *mockSocialService) SendMessage(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, cmd SendMessageCommand) (*MessageResponse, error) {
	if m.sendMessageFn != nil {
		return m.sendMessageFn(ctx, auth, conversationID, cmd)
	}
	panic("unexpected call to SendMessage")
}

func (m *mockSocialService) MarkConversationRead(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error {
	if m.markConversationReadFn != nil {
		return m.markConversationReadFn(ctx, auth, conversationID)
	}
	panic("unexpected call to MarkConversationRead")
}

func (m *mockSocialService) DeleteConversation(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error {
	if m.deleteConversationFn != nil {
		return m.deleteConversationFn(ctx, auth, conversationID)
	}
	panic("unexpected call to DeleteConversation")
}

func (m *mockSocialService) ReportMessage(ctx context.Context, auth *shared.AuthContext, messageID uuid.UUID, cmd ReportMessageCommand) error {
	if m.reportMessageFn != nil {
		return m.reportMessageFn(ctx, auth, messageID, cmd)
	}
	panic("unexpected call to ReportMessage")
}

func (m *mockSocialService) ListConversations(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]ConversationSummaryResponse, error) {
	if m.listConversationsFn != nil {
		return m.listConversationsFn(ctx, auth, offset, limit)
	}
	panic("unexpected call to ListConversations")
}

func (m *mockSocialService) GetConversationMessages(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, offset, limit int) ([]MessageResponse, error) {
	if m.getConversationMessagesFn != nil {
		return m.getConversationMessagesFn(ctx, auth, conversationID, offset, limit)
	}
	panic("unexpected call to GetConversationMessages")
}

// ─── Groups ─────────────────────────────────────────────────────────────────

func (m *mockSocialService) JoinGroup(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error {
	if m.joinGroupFn != nil {
		return m.joinGroupFn(ctx, scope, groupID)
	}
	panic("unexpected call to JoinGroup")
}

func (m *mockSocialService) LeaveGroup(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error {
	if m.leaveGroupFn != nil {
		return m.leaveGroupFn(ctx, scope, groupID)
	}
	panic("unexpected call to LeaveGroup")
}

func (m *mockSocialService) CreateGroup(ctx context.Context, auth *shared.AuthContext, cmd CreateGroupCommand) (*GroupResponse, error) {
	if m.createGroupFn != nil {
		return m.createGroupFn(ctx, auth, cmd)
	}
	panic("unexpected call to CreateGroup")
}

func (m *mockSocialService) UpdateGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, cmd UpdateGroupCommand) (*GroupResponse, error) {
	if m.updateGroupFn != nil {
		return m.updateGroupFn(ctx, auth, groupID, cmd)
	}
	panic("unexpected call to UpdateGroup")
}

func (m *mockSocialService) DeleteGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) error {
	if m.deleteGroupFn != nil {
		return m.deleteGroupFn(ctx, auth, groupID)
	}
	panic("unexpected call to DeleteGroup")
}

func (m *mockSocialService) ApproveMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	if m.approveMemberFn != nil {
		return m.approveMemberFn(ctx, auth, groupID, familyID)
	}
	panic("unexpected call to ApproveMember")
}

func (m *mockSocialService) RejectMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	if m.rejectMemberFn != nil {
		return m.rejectMemberFn(ctx, auth, groupID, familyID)
	}
	panic("unexpected call to RejectMember")
}

func (m *mockSocialService) BanMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	if m.banMemberFn != nil {
		return m.banMemberFn(ctx, auth, groupID, familyID)
	}
	panic("unexpected call to BanMember")
}

func (m *mockSocialService) InviteToGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	if m.inviteToGroupFn != nil {
		return m.inviteToGroupFn(ctx, auth, groupID, familyID)
	}
	panic("unexpected call to InviteToGroup")
}

func (m *mockSocialService) PromoteMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	if m.promoteMemberFn != nil {
		return m.promoteMemberFn(ctx, auth, groupID, familyID)
	}
	panic("unexpected call to PromoteMember")
}

func (m *mockSocialService) PinPost(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}
func (m *mockSocialService) UnpinPost(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockSocialService) GetGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) (*GroupResponse, error) {
	if m.getGroupFn != nil {
		return m.getGroupFn(ctx, auth, groupID)
	}
	panic("unexpected call to GetGroup")
}

func (m *mockSocialService) ListMyGroups(ctx context.Context, scope *shared.FamilyScope) ([]GroupResponse, error) {
	if m.listMyGroupsFn != nil {
		return m.listMyGroupsFn(ctx, scope)
	}
	panic("unexpected call to ListMyGroups")
}

func (m *mockSocialService) ListPlatformGroups(ctx context.Context) ([]GroupResponse, error) {
	if m.listPlatformGroupsFn != nil {
		return m.listPlatformGroupsFn(ctx)
	}
	panic("unexpected call to ListPlatformGroups")
}

func (m *mockSocialService) ListGroupMembers(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) ([]GroupMemberResponse, error) {
	if m.listGroupMembersFn != nil {
		return m.listGroupMembersFn(ctx, auth, groupID)
	}
	panic("unexpected call to ListGroupMembers")
}

func (m *mockSocialService) ListGroupPosts(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, offset, limit int) ([]PostResponse, error) {
	if m.listGroupPostsFn != nil {
		return m.listGroupPostsFn(ctx, auth, groupID, offset, limit)
	}
	panic("unexpected call to ListGroupPosts")
}

// ─── Events ─────────────────────────────────────────────────────────────────

func (m *mockSocialService) CreateEvent(ctx context.Context, auth *shared.AuthContext, cmd CreateEventCommand) (*EventDetailResponse, error) {
	if m.createEventFn != nil {
		return m.createEventFn(ctx, auth, cmd)
	}
	panic("unexpected call to CreateEvent")
}

func (m *mockSocialService) UpdateEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID, cmd UpdateEventCommand) (*EventDetailResponse, error) {
	if m.updateEventFn != nil {
		return m.updateEventFn(ctx, auth, eventID, cmd)
	}
	panic("unexpected call to UpdateEvent")
}

func (m *mockSocialService) CancelEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) error {
	if m.cancelEventFn != nil {
		return m.cancelEventFn(ctx, auth, eventID)
	}
	panic("unexpected call to CancelEvent")
}

func (m *mockSocialService) RSVPEvent(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID, cmd RSVPCommand) error {
	if m.rsvpEventFn != nil {
		return m.rsvpEventFn(ctx, scope, eventID, cmd)
	}
	panic("unexpected call to RSVPEvent")
}

func (m *mockSocialService) RemoveRSVP(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID) error {
	if m.removeRSVPFn != nil {
		return m.removeRSVPFn(ctx, scope, eventID)
	}
	panic("unexpected call to RemoveRSVP")
}

func (m *mockSocialService) GetEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) (*EventDetailResponse, error) {
	if m.getEventFn != nil {
		return m.getEventFn(ctx, auth, eventID)
	}
	panic("unexpected call to GetEvent")
}

func (m *mockSocialService) ListEvents(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]EventDetailResponse, error) {
	if m.listEventsFn != nil {
		return m.listEventsFn(ctx, auth, offset, limit)
	}
	panic("unexpected call to ListEvents")
}

// ─── Discovery ───────────────────────────────────────────────────────────────

func (m *mockSocialService) DiscoverFamilies(ctx context.Context, scope *shared.FamilyScope, query DiscoverFamiliesQuery) ([]DiscoverableFamilyResponse, error) {
	if m.discoverFamiliesFn != nil {
		return m.discoverFamiliesFn(ctx, scope, query)
	}
	panic("unexpected call to DiscoverFamilies")
}

func (m *mockSocialService) DiscoverEvents(ctx context.Context, scope *shared.FamilyScope, query DiscoverEventsQuery) ([]EventSummaryResponse, error) {
	if m.discoverEventsFn != nil {
		return m.discoverEventsFn(ctx, scope, query)
	}
	panic("unexpected call to DiscoverEvents")
}

func (m *mockSocialService) DiscoverGroups(ctx context.Context, scope *shared.FamilyScope, query DiscoverGroupsQuery) ([]GroupSummaryResponse, error) {
	if m.discoverGroupsFn != nil {
		return m.discoverGroupsFn(ctx, scope, query)
	}
	panic("unexpected call to DiscoverGroups")
}

// ─── Event Handlers ─────────────────────────────────────────────────────────

func (m *mockSocialService) HandleFamilyCreated(ctx context.Context, familyID uuid.UUID) error {
	if m.handleFamilyCreatedFn != nil {
		return m.handleFamilyCreatedFn(ctx, familyID)
	}
	panic("unexpected call to HandleFamilyCreated")
}

func (m *mockSocialService) HandleCoParentAdded(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (m *mockSocialService) HandleCoParentRemoved(ctx context.Context, familyID uuid.UUID, parentID uuid.UUID) error {
	if m.handleCoParentRemovedFn != nil {
		return m.handleCoParentRemovedFn(ctx, familyID, parentID)
	}
	return nil
}

func (m *mockSocialService) HandleMilestoneAchieved(ctx context.Context, familyID uuid.UUID, milestone MilestoneData) error {
	if m.handleMilestoneAchievedFn != nil {
		return m.handleMilestoneAchievedFn(ctx, familyID, milestone)
	}
	return nil
}

func (m *mockSocialService) HandleFamilyDeletionScheduled(ctx context.Context, familyID uuid.UUID) error {
	if m.handleFamilyDeletionScheduledFn != nil {
		return m.handleFamilyDeletionScheduledFn(ctx, familyID)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// PROFILE TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── GET /v1/social/profile ──────────────────────────────────────────────────

func TestGetOwnProfile_200(t *testing.T) {
	e := newTestEcho()
	locVisible := true
	svc := &mockSocialService{
		getOwnProfileFn: func(_ context.Context, _ *shared.FamilyScope) (*ProfileResponse, error) {
			return &ProfileResponse{
				FamilyID:        testFamilyID,
				LocationVisible: &locVisible,
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/profile", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp ProfileResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if resp.LocationVisible == nil || !*resp.LocationVisible {
		t.Error("want location_visible=true for own profile")
	}
}

func TestGetOwnProfile_404_ProfileNotFound(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		getOwnProfileFn: func(_ context.Context, _ *shared.FamilyScope) (*ProfileResponse, error) {
			return nil, &SocialError{Err: domain.ErrProfileNotFound}
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/profile", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/families/:id/profile ─────────────────────────────────────

func TestGetFamilyProfile_200(t *testing.T) {
	targetID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		getFamilyProfileFn: func(_ context.Context, _ *shared.AuthContext, got uuid.UUID) (*ProfileResponse, error) {
			if got != targetID {
				t.Errorf("want targetID=%s, got=%s", targetID, got)
			}
			return &ProfileResponse{
				FamilyID: targetID,
				IsFriend: true,
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/families/"+targetID.String()+"/profile", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetFamilyProfile_400_InvalidUUID(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/families/not-a-uuid/profile", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── PATCH /v1/social/profile ────────────────────────────────────────────────

func TestUpdateProfile_200(t *testing.T) {
	e := newTestEcho()
	bio := "We love homeschooling!"
	svc := &mockSocialService{
		updateProfileFn: func(_ context.Context, _ *shared.FamilyScope, cmd UpdateProfileCommand) (*ProfileResponse, error) {
			if cmd.Bio == nil || *cmd.Bio != bio {
				t.Errorf("want bio=%q, got=%v", bio, cmd.Bio)
			}
			return &ProfileResponse{
				FamilyID: testFamilyID,
				Bio:      &bio,
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"bio":"We love homeschooling!"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/social/profile", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// FRIEND TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── POST /v1/social/friends/request/:familyId ───────────────────────────────

func TestSendFriendRequest_201(t *testing.T) {
	targetID := uuid.Must(uuid.NewV7())
	friendshipID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		sendFriendRequestFn: func(_ context.Context, _ *shared.AuthContext, got uuid.UUID) (*FriendshipResponse, error) {
			if got != targetID {
				t.Errorf("want targetID=%s, got=%s", targetID, got)
			}
			return &FriendshipResponse{
				ID:                friendshipID,
				RequesterFamilyID: testFamilyID,
				AccepterFamilyID:  targetID,
				Status:            "pending",
				CreatedAt:         time.Now(),
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/social/friends/request/"+targetID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestSendFriendRequest_422_CannotFriendSelf(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		sendFriendRequestFn: func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID) (*FriendshipResponse, error) {
			return nil, &SocialError{Err: domain.ErrCannotFriendSelf}
		},
	}
	setupSocialRoutes(e, svc)

	targetID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodPost, "/v1/social/friends/request/"+targetID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/social/friends/accept/:friendshipId ────────────────────────────

func TestAcceptFriendRequest_200(t *testing.T) {
	friendshipID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		acceptFriendRequestFn: func(_ context.Context, _ *shared.AuthContext, got uuid.UUID) (*FriendshipResponse, error) {
			if got != friendshipID {
				t.Errorf("want friendshipID=%s, got=%s", friendshipID, got)
			}
			return &FriendshipResponse{
				ID:     friendshipID,
				Status: "accepted",
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/social/friends/accept/"+friendshipID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/friends ──────────────────────────────────────────────────

func TestListFriends_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		listFriendsFn: func(_ context.Context, _ *shared.FamilyScope, _ *uuid.UUID, _ int) ([]FriendResponse, error) {
			return []FriendResponse{
				{FamilyID: uuid.Must(uuid.NewV7()), DisplayName: "Test Family", FriendsSince: time.Now()},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/friends", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp []FriendResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("want 1 friend, got %d", len(resp))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// BLOCK TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── POST /v1/social/blocks/:familyId ────────────────────────────────────────

func TestBlockFamily_201(t *testing.T) {
	targetID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		blockFamilyFn: func(_ context.Context, _ *shared.AuthContext, got uuid.UUID) error {
			if got != targetID {
				t.Errorf("want targetID=%s, got=%s", targetID, got)
			}
			return nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/social/blocks/"+targetID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/blocks ───────────────────────────────────────────────────

func TestListBlocks_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		listBlocksFn: func(_ context.Context, _ *shared.FamilyScope) ([]BlockedFamilyResponse, error) {
			return []BlockedFamilyResponse{
				{FamilyID: uuid.Must(uuid.NewV7()), DisplayName: "Blocked Family", BlockedAt: time.Now()},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/blocks", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// POST TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── POST /v1/social/posts ───────────────────────────────────────────────────

func TestCreatePost_201(t *testing.T) {
	postID := uuid.Must(uuid.NewV7())
	content := "Hello world!"
	e := newTestEcho()
	svc := &mockSocialService{
		createPostFn: func(_ context.Context, _ *shared.AuthContext, cmd CreatePostCommand) (*PostResponse, error) {
			if cmd.PostType != "text" {
				t.Errorf("want post_type=text, got=%s", cmd.PostType)
			}
			return &PostResponse{
				ID:       postID,
				PostType: "text",
				Content:  &content,
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"post_type":"text","content":"Hello world!"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreatePost_422_MissingPostType(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{}
	setupSocialRoutes(e, svc)

	body := `{"content":"Hello world!"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/posts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/posts/:id ────────────────────────────────────────────────

func TestGetPost_200(t *testing.T) {
	postID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		getPostFn: func(_ context.Context, _ *shared.AuthContext, got uuid.UUID) (*PostDetailResponse, error) {
			if got != postID {
				t.Errorf("want postID=%s, got=%s", postID, got)
			}
			return &PostDetailResponse{
				Post: PostResponse{
					ID:       postID,
					PostType: "text",
				},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/posts/"+postID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetPost_400_InvalidUUID(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/posts/not-a-uuid", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestGetPost_404_PostNotFound(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		getPostFn: func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID) (*PostDetailResponse, error) {
			return nil, &SocialError{Err: domain.ErrPostNotFound}
		},
	}
	setupSocialRoutes(e, svc)

	postID := uuid.Must(uuid.NewV7())
	req := httptest.NewRequest(http.MethodGet, "/v1/social/posts/"+postID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/feed ─────────────────────────────────────────────────────

func TestGetFeed_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		getFeedFn: func(_ context.Context, _ *shared.AuthContext, offset, limit int) (*FeedResponse, error) {
			if offset != 0 {
				t.Errorf("want offset=0, got=%d", offset)
			}
			if limit != 20 {
				t.Errorf("want limit=20, got=%d", limit)
			}
			return &FeedResponse{
				Posts: []PostResponse{
					{ID: uuid.Must(uuid.NewV7()), PostType: "text"},
				},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/feed", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── DELETE /v1/social/posts/:id ─────────────────────────────────────────────

func TestDeletePost_204(t *testing.T) {
	postID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		deletePostFn: func(_ context.Context, _ *shared.AuthContext, got uuid.UUID) error {
			if got != postID {
				t.Errorf("want postID=%s, got=%s", postID, got)
			}
			return nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/social/posts/"+postID.String(), nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// COMMENT TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── POST /v1/social/posts/:id/comments ──────────────────────────────────────

func TestCreateComment_201(t *testing.T) {
	postID := uuid.Must(uuid.NewV7())
	commentID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		createCommentFn: func(_ context.Context, _ *shared.AuthContext, gotPostID uuid.UUID, cmd CreateCommentCommand) (*CommentResponse, error) {
			if gotPostID != postID {
				t.Errorf("want postID=%s, got=%s", postID, gotPostID)
			}
			if cmd.Content != "Great post!" {
				t.Errorf("want content=Great post!, got=%s", cmd.Content)
			}
			return &CommentResponse{
				ID:      commentID,
				PostID:  postID,
				Content: cmd.Content,
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"content":"Great post!"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/posts/"+postID.String()+"/comments", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/posts/:id/comments ───────────────────────────────────────

func TestListComments_200(t *testing.T) {
	postID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		listCommentsFn: func(_ context.Context, _ *shared.AuthContext, got uuid.UUID) ([]CommentResponse, error) {
			if got != postID {
				t.Errorf("want postID=%s, got=%s", postID, got)
			}
			return []CommentResponse{
				{ID: uuid.Must(uuid.NewV7()), PostID: postID, Content: "Nice!"},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/posts/"+postID.String()+"/comments", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// MESSAGING TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── POST /v1/social/conversations ───────────────────────────────────────────

func TestCreateConversation_201(t *testing.T) {
	convID := uuid.Must(uuid.NewV7())
	recipientID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		createConversationFn: func(_ context.Context, _ *shared.AuthContext, cmd CreateConversationCommand) (*ConversationResponse, error) {
			if cmd.RecipientParentID != recipientID {
				t.Errorf("want recipient=%s, got=%s", recipientID, cmd.RecipientParentID)
			}
			return &ConversationResponse{
				ID:        convID,
				UpdatedAt: time.Now(),
				IsNew:     true,
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"recipient_parent_id":"` + recipientID.String() + `","initial_message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/conversations", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/social/conversations/:id/messages ─────────────────────────────

func TestSendMessage_201(t *testing.T) {
	convID := uuid.Must(uuid.NewV7())
	msgID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		sendMessageFn: func(_ context.Context, _ *shared.AuthContext, gotConvID uuid.UUID, cmd SendMessageCommand) (*MessageResponse, error) {
			if gotConvID != convID {
				t.Errorf("want convID=%s, got=%s", convID, gotConvID)
			}
			if cmd.Content != "Hey there!" {
				t.Errorf("want content=Hey there!, got=%s", cmd.Content)
			}
			return &MessageResponse{
				ID:             msgID,
				ConversationID: convID,
				Content:        cmd.Content,
				CreatedAt:      time.Now(),
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"content":"Hey there!"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/conversations/"+convID.String()+"/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/conversations ────────────────────────────────────────────

func TestListConversations_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		listConversationsFn: func(_ context.Context, _ *shared.AuthContext, _, _ int) ([]ConversationSummaryResponse, error) {
			return []ConversationSummaryResponse{
				{ID: uuid.Must(uuid.NewV7()), OtherParentName: "Jane Doe", UnreadCount: 3, UpdatedAt: time.Now()},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/conversations", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// GROUP TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── POST /v1/social/groups ──────────────────────────────────────────────────

func TestCreateGroup_201(t *testing.T) {
	groupID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		createGroupFn: func(_ context.Context, _ *shared.AuthContext, cmd CreateGroupCommand) (*GroupResponse, error) {
			if cmd.Name != "Charlotte Mason Parents" {
				t.Errorf("want name=Charlotte Mason Parents, got=%s", cmd.Name)
			}
			return &GroupResponse{
				Summary: GroupSummaryResponse{
					ID:         groupID,
					Name:       cmd.Name,
					JoinPolicy: cmd.JoinPolicy,
					IsMember:   true,
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"name":"Charlotte Mason Parents","join_policy":"open"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGroup_422_MissingName(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{}
	setupSocialRoutes(e, svc)

	body := `{"join_policy":"open"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/groups", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/social/groups/:id/join ─────────────────────────────────────────

func TestJoinGroup_204(t *testing.T) {
	groupID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		joinGroupFn: func(_ context.Context, _ *shared.FamilyScope, got uuid.UUID) error {
			if got != groupID {
				t.Errorf("want groupID=%s, got=%s", groupID, got)
			}
			return nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/social/groups/"+groupID.String()+"/join", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/groups ───────────────────────────────────────────────────

func TestListMyGroups_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		listMyGroupsFn: func(_ context.Context, _ *shared.FamilyScope) ([]GroupResponse, error) {
			return []GroupResponse{
				{Summary: GroupSummaryResponse{ID: uuid.Must(uuid.NewV7()), Name: "Study Group", IsMember: true}, CreatedAt: time.Now()},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/groups", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/groups/platform ──────────────────────────────────────────

func TestListPlatformGroups_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		listPlatformGroupsFn: func(_ context.Context) ([]GroupResponse, error) {
			return []GroupResponse{
				{Summary: GroupSummaryResponse{ID: uuid.Must(uuid.NewV7()), Name: "Methodology Hub", GroupType: "methodology"}, CreatedAt: time.Now()},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/groups/platform", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// EVENT TESTS
// ═══════════════════════════════════════════════════════════════════════════════

// ─── POST /v1/social/events ──────────────────────────────────────────────────

func TestCreateEvent_201(t *testing.T) {
	eventID := uuid.Must(uuid.NewV7())
	eventDate := time.Date(2026, 6, 15, 10, 0, 0, 0, time.UTC)
	e := newTestEcho()
	svc := &mockSocialService{
		createEventFn: func(_ context.Context, _ *shared.AuthContext, cmd CreateEventCommand) (*EventDetailResponse, error) {
			if cmd.Title != "Park Day" {
				t.Errorf("want title=Park Day, got=%s", cmd.Title)
			}
			return &EventDetailResponse{
				EventSummaryResponse: EventSummaryResponse{
					ID:        eventID,
					Title:     cmd.Title,
					EventDate: eventDate,
					Status:    "active",
				},
				CreatedAt: time.Now(),
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"title":"Park Day","event_date":"2026-06-15T10:00:00Z","visibility":"friends"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateEvent_422_MissingTitle(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{}
	setupSocialRoutes(e, svc)

	body := `{"event_date":"2026-06-15T10:00:00Z","visibility":"friends"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/events", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── GET /v1/social/events ───────────────────────────────────────────────────

func TestListEvents_200(t *testing.T) {
	e := newTestEcho()
	svc := &mockSocialService{
		listEventsFn: func(_ context.Context, _ *shared.AuthContext, _, _ int) ([]EventDetailResponse, error) {
			return []EventDetailResponse{
				{
					EventSummaryResponse: EventSummaryResponse{
						ID:        uuid.Must(uuid.NewV7()),
						Title:     "Field Trip",
						EventDate: time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC),
						Status:    "active",
					},
					CreatedAt: time.Now(),
				},
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodGet, "/v1/social/events", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/social/events/:id/rsvp ─────────────────────────────────────────

func TestRSVPEvent_204(t *testing.T) {
	eventID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		rsvpEventFn: func(_ context.Context, _ *shared.FamilyScope, got uuid.UUID, cmd RSVPCommand) error {
			if got != eventID {
				t.Errorf("want eventID=%s, got=%s", eventID, got)
			}
			if cmd.Status != "going" {
				t.Errorf("want status=going, got=%s", cmd.Status)
			}
			return nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"status":"going"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/events/"+eventID.String()+"/rsvp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestRSVPEvent_409_AlreadyRSVPd(t *testing.T) {
	eventID := uuid.Must(uuid.NewV7())
	e := newTestEcho()
	svc := &mockSocialService{
		rsvpEventFn: func(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ RSVPCommand) error {
			return &SocialError{Err: domain.ErrAlreadyRSVPd}
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"status":"going"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/social/events/"+eventID.String()+"/rsvp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Errorf("want 409, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── PATCH /v1/social/posts/:id ─────────────────────────────────────────────

func TestUpdatePost_200OnSuccess(t *testing.T) {
	e := newTestEcho()
	postID := uuid.Must(uuid.NewV7())
	content := "updated content"
	svc := &mockSocialService{
		updatePostFn: func(_ context.Context, _ *shared.AuthContext, id uuid.UUID, cmd UpdatePostCommand) (*PostResponse, error) {
			if id != postID {
				t.Errorf("want postID %s, got %s", postID, id)
			}
			return &PostResponse{
				ID:       id,
				FamilyID: testFamilyID,
				PostType: "text",
				Content:  cmd.Content,
				IsEdited: true,
			}, nil
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"content":"` + content + `"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/social/posts/"+postID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var resp PostResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.IsEdited {
		t.Error("want is_edited=true after update")
	}
}

func TestUpdatePost_403WhenNotAuthor(t *testing.T) {
	e := newTestEcho()
	postID := uuid.Must(uuid.NewV7())
	svc := &mockSocialService{
		updatePostFn: func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ UpdatePostCommand) (*PostResponse, error) {
			return nil, &SocialError{Err: domain.ErrCannotEditPost}
		},
	}
	setupSocialRoutes(e, svc)

	body := `{"content":"hacked"}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/social/posts/"+postID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("want 403, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUpdatePost_422WhenEmptyUpdate(t *testing.T) {
	e := newTestEcho()
	postID := uuid.Must(uuid.NewV7())
	svc := &mockSocialService{
		updatePostFn: func(_ context.Context, _ *shared.AuthContext, _ uuid.UUID, _ UpdatePostCommand) (*PostResponse, error) {
			return nil, &SocialError{Err: domain.ErrPostEditEmpty}
		},
	}
	setupSocialRoutes(e, svc)

	body := `{}`
	req := httptest.NewRequest(http.MethodPatch, "/v1/social/posts/"+postID.String(), strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("want 422, got %d: %s", rec.Code, rec.Body.String())
	}
}

// ─── POST /v1/social/groups/:id/posts/:postId/pin ───────────────────────────

func TestPinPost_200OnSuccess(t *testing.T) {
	e := newTestEcho()
	groupID := uuid.Must(uuid.NewV7())
	postID := uuid.Must(uuid.NewV7())
	svc := &mockSocialService{}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodPost, "/v1/social/groups/"+groupID.String()+"/posts/"+postID.String()+"/pin", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestUnpinPost_204OnSuccess(t *testing.T) {
	e := newTestEcho()
	groupID := uuid.Must(uuid.NewV7())
	postID := uuid.Must(uuid.NewV7())
	svc := &mockSocialService{}
	setupSocialRoutes(e, svc)

	req := httptest.NewRequest(http.MethodDelete, "/v1/social/groups/"+groupID.String()+"/posts/"+postID.String()+"/pin", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("want 204, got %d: %s", rec.Code, rec.Body.String())
	}
}
