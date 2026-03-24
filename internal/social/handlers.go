package social

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social/domain"
	"github.com/labstack/echo/v4"
)

// Handler holds the social HTTP handler dependencies.
type Handler struct {
	svc    SocialService
	pubsub shared.PubSub
}

// NewHandler creates a new social Handler.
func NewHandler(svc SocialService, pubsub shared.PubSub) *Handler {
	return &Handler{svc: svc, pubsub: pubsub}
}

// Register registers all social routes on the authenticated route group.
// All social endpoints require authentication. [05-social §4]
func (h *Handler) Register(authGroup *echo.Group) {
	soc := authGroup.Group("/social")

	// Profile
	soc.GET("/profile", h.getOwnProfile)
	soc.PATCH("/profile", h.updateProfile)
	soc.GET("/families/:id/profile", h.getFamilyProfile)

	// Friends
	soc.POST("/friends/request/:familyId", h.sendFriendRequest)
	soc.POST("/friends/accept/:friendshipId", h.acceptFriendRequest)
	soc.POST("/friends/reject/:friendshipId", h.rejectFriendRequest)
	soc.DELETE("/friends/:familyId", h.unfriend)
	soc.GET("/friends", h.listFriends)
	soc.GET("/friends/requests/incoming", h.listIncomingRequests)
	soc.GET("/friends/requests/outgoing", h.listOutgoingRequests)

	// Blocks
	soc.POST("/blocks/:familyId", h.blockFamily)
	soc.DELETE("/blocks/:familyId", h.unblockFamily)
	soc.GET("/blocks", h.listBlocks)

	// Posts / Feed
	soc.POST("/posts", h.createPost)
	soc.GET("/posts/:id", h.getPost)
	soc.DELETE("/posts/:id", h.deletePost)
	soc.POST("/posts/:id/like", h.likePost)
	soc.DELETE("/posts/:id/like", h.unlikePost)
	soc.GET("/feed", h.getFeed)

	// Comments
	soc.POST("/posts/:id/comments", h.createComment)
	soc.GET("/posts/:id/comments", h.listComments)
	soc.DELETE("/comments/:id", h.deleteComment)

	// Messaging
	soc.POST("/conversations", h.createConversation)
	soc.GET("/conversations", h.listConversations)
	soc.GET("/conversations/:id/messages", h.getConversationMessages)
	soc.POST("/conversations/:id/messages", h.sendMessage)
	soc.POST("/conversations/:id/read", h.markConversationRead)
	soc.DELETE("/conversations/:id", h.deleteConversation)
	soc.POST("/messages/:id/report", h.reportMessage)

	// Groups
	soc.POST("/groups", h.createGroup)
	soc.GET("/groups", h.listMyGroups)
	soc.GET("/groups/platform", h.listPlatformGroups)
	soc.GET("/groups/:id", h.getGroup)
	soc.PATCH("/groups/:id", h.updateGroup)
	soc.DELETE("/groups/:id", h.deleteGroup)
	soc.POST("/groups/:id/join", h.joinGroup)
	soc.POST("/groups/:id/leave", h.leaveGroup)
	soc.GET("/groups/:id/members", h.listGroupMembers)
	soc.GET("/groups/:id/posts", h.listGroupPosts)
	soc.POST("/groups/:id/members/:familyId/approve", h.approveMember)
	soc.POST("/groups/:id/members/:familyId/reject", h.rejectMember)
	soc.POST("/groups/:id/members/:familyId/ban", h.banMember)
	soc.POST("/groups/:id/members/:familyId/invite", h.inviteToGroup)
	soc.POST("/groups/:id/members/:familyId/promote", h.promoteMember)

	// Events
	soc.POST("/events", h.createEvent)
	soc.GET("/events", h.listEvents)
	soc.GET("/events/:id", h.getEvent)
	soc.PATCH("/events/:id", h.updateEvent)
	soc.POST("/events/:id/cancel", h.cancelEvent)
	soc.POST("/events/:id/rsvp", h.rsvpEvent)
	soc.DELETE("/events/:id/rsvp", h.removeRSVP)

	// Discovery
	soc.GET("/discover/families", h.discoverFamilies)
	soc.GET("/discover/events",   h.discoverEvents)
	soc.GET("/discover/groups",   h.discoverGroups)

	// WebSocket
	soc.GET("/ws", handleWebSocket(h.pubsub))
}

// ─── Profile Handlers ───────────────────────────────────────────────────────

func (h *Handler) getOwnProfile(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.GetOwnProfile(c.Request().Context(), &scope)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) updateProfile(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var cmd UpdateProfileCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateProfile(c.Request().Context(), &scope, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getFamilyProfile(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	targetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	resp, err := h.svc.GetFamilyProfile(c.Request().Context(), auth, targetID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Friend Handlers ────────────────────────────────────────────────────────

func (h *Handler) sendFriendRequest(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	targetID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	resp, err := h.svc.SendFriendRequest(c.Request().Context(), auth, targetID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) acceptFriendRequest(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	friendshipID, err := uuid.Parse(c.Param("friendshipId"))
	if err != nil {
		return shared.ErrBadRequest("invalid friendship ID")
	}
	resp, err := h.svc.AcceptFriendRequest(c.Request().Context(), auth, friendshipID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) rejectFriendRequest(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	friendshipID, err := uuid.Parse(c.Param("friendshipId"))
	if err != nil {
		return shared.ErrBadRequest("invalid friendship ID")
	}
	if err := h.svc.RejectFriendRequest(c.Request().Context(), auth, friendshipID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) unfriend(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	targetID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.Unfriend(c.Request().Context(), auth, targetID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) listFriends(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	// Cursor-based pagination. [05-social §4.1] (H6)
	var cursor *uuid.UUID
	if cursorStr := c.QueryParam("cursor"); cursorStr != "" {
		parsed, parseErr := uuid.Parse(cursorStr)
		if parseErr != nil {
			return shared.ErrBadRequest("invalid cursor")
		}
		cursor = &parsed
	}
	limit := 20
	if v := c.QueryParam("limit"); v != "" {
		if n, parseErr := strconv.Atoi(v); parseErr == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	resp, err := h.svc.ListFriends(c.Request().Context(), &scope, cursor, limit)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) listIncomingRequests(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ListIncomingRequests(c.Request().Context(), &scope)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) listOutgoingRequests(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ListOutgoingRequests(c.Request().Context(), &scope)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Block Handlers ─────────────────────────────────────────────────────────

func (h *Handler) blockFamily(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	targetID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.BlockFamily(c.Request().Context(), auth, targetID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusCreated)
}

func (h *Handler) unblockFamily(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	targetID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.UnblockFamily(c.Request().Context(), auth, targetID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) listBlocks(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ListBlocks(c.Request().Context(), &scope)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Post / Feed Handlers ───────────────────────────────────────────────────

func (h *Handler) createPost(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	var cmd CreatePostCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreatePost(c.Request().Context(), auth, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) getPost(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid post ID")
	}
	resp, err := h.svc.GetPost(c.Request().Context(), auth, postID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) deletePost(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid post ID")
	}
	if err := h.svc.DeletePost(c.Request().Context(), auth, postID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) likePost(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid post ID")
	}
	if err := h.svc.LikePost(c.Request().Context(), &scope, postID); err != nil {
		// Idempotent: already liked is not an error — return 200 OK. [05-social §4.1]
		var socErr *SocialError
		if errors.As(err, &socErr) && errors.Is(socErr.Err, domain.ErrAlreadyLiked) {
			return c.NoContent(http.StatusOK)
		}
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusCreated)
}

func (h *Handler) unlikePost(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid post ID")
	}
	if err := h.svc.UnlikePost(c.Request().Context(), &scope, postID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) getFeed(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	offset, limit := parsePagination(c)
	resp, err := h.svc.GetFeed(c.Request().Context(), auth, offset, limit)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Comment Handlers ───────────────────────────────────────────────────────

func (h *Handler) createComment(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid post ID")
	}
	var cmd CreateCommentCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateComment(c.Request().Context(), auth, postID, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) listComments(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid post ID")
	}
	resp, err := h.svc.ListComments(c.Request().Context(), auth, postID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) deleteComment(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	commentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid comment ID")
	}
	if err := h.svc.DeleteComment(c.Request().Context(), auth, commentID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Messaging Handlers ────────────────────────────────────────────────────

func (h *Handler) createConversation(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	var cmd CreateConversationCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateConversation(c.Request().Context(), auth, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	status := http.StatusOK
	if resp.IsNew {
		status = http.StatusCreated
	}
	return c.JSON(status, resp)
}

func (h *Handler) listConversations(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	offset, limit := parsePagination(c)
	resp, err := h.svc.ListConversations(c.Request().Context(), auth, offset, limit)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getConversationMessages(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid conversation ID")
	}
	offset, limit := parsePagination(c)
	resp, err := h.svc.GetConversationMessages(c.Request().Context(), auth, convID, offset, limit)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) sendMessage(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid conversation ID")
	}
	var cmd SendMessageCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.SendMessage(c.Request().Context(), auth, convID, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) markConversationRead(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid conversation ID")
	}
	if err := h.svc.MarkConversationRead(c.Request().Context(), auth, convID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) deleteConversation(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	convID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid conversation ID")
	}
	if err := h.svc.DeleteConversation(c.Request().Context(), auth, convID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) reportMessage(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	msgID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid message ID")
	}
	var cmd ReportMessageCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	if err := h.svc.ReportMessage(c.Request().Context(), auth, msgID, cmd); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Group Handlers ─────────────────────────────────────────────────────────

func (h *Handler) createGroup(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	var cmd CreateGroupCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateGroup(c.Request().Context(), auth, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) listMyGroups(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	resp, err := h.svc.ListMyGroups(c.Request().Context(), &scope)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) listPlatformGroups(c echo.Context) error {
	resp, err := h.svc.ListPlatformGroups(c.Request().Context())
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getGroup(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	resp, err := h.svc.GetGroup(c.Request().Context(), auth, groupID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) updateGroup(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	var cmd UpdateGroupCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateGroup(c.Request().Context(), auth, groupID, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) deleteGroup(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	if err := h.svc.DeleteGroup(c.Request().Context(), auth, groupID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) joinGroup(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	if err := h.svc.JoinGroup(c.Request().Context(), &scope, groupID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) leaveGroup(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	if err := h.svc.LeaveGroup(c.Request().Context(), &scope, groupID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) listGroupMembers(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	resp, err := h.svc.ListGroupMembers(c.Request().Context(), auth, groupID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) listGroupPosts(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	offset, limit := parsePagination(c)
	resp, err := h.svc.ListGroupPosts(c.Request().Context(), auth, groupID, offset, limit)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) approveMember(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	familyID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.ApproveMember(c.Request().Context(), auth, groupID, familyID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) rejectMember(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	familyID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.RejectMember(c.Request().Context(), auth, groupID, familyID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) banMember(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	familyID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.BanMember(c.Request().Context(), auth, groupID, familyID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) inviteToGroup(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	familyID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.InviteToGroup(c.Request().Context(), auth, groupID, familyID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) promoteMember(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	groupID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid group ID")
	}
	familyID, err := uuid.Parse(c.Param("familyId"))
	if err != nil {
		return shared.ErrBadRequest("invalid family ID")
	}
	if err := h.svc.PromoteMember(c.Request().Context(), auth, groupID, familyID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Event Handlers ─────────────────────────────────────────────────────────

func (h *Handler) createEvent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	var cmd CreateEventCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.CreateEvent(c.Request().Context(), auth, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusCreated, resp)
}

func (h *Handler) listEvents(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	offset, limit := parsePagination(c)
	resp, err := h.svc.ListEvents(c.Request().Context(), auth, offset, limit)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

// ─── Discovery Handlers ──────────────────────────────────────────────────────

func (h *Handler) discoverFamilies(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var q DiscoverFamiliesQuery
	if err := c.Bind(&q); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}
	resp, err := h.svc.DiscoverFamilies(c.Request().Context(), &scope, q)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) discoverEvents(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var q DiscoverEventsQuery
	if err := c.Bind(&q); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}
	resp, err := h.svc.DiscoverEvents(c.Request().Context(), &scope, q)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) discoverGroups(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	var q DiscoverGroupsQuery
	if err := c.Bind(&q); err != nil {
		return shared.ErrBadRequest("invalid query params")
	}
	resp, err := h.svc.DiscoverGroups(c.Request().Context(), &scope, q)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) getEvent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid event ID")
	}
	resp, err := h.svc.GetEvent(c.Request().Context(), auth, eventID)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) updateEvent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid event ID")
	}
	var cmd UpdateEventCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	resp, err := h.svc.UpdateEvent(c.Request().Context(), auth, eventID, cmd)
	if err != nil {
		return mapSocialError(err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (h *Handler) cancelEvent(c echo.Context) error {
	auth, err := shared.GetAuthContext(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid event ID")
	}
	if err := h.svc.CancelEvent(c.Request().Context(), auth, eventID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) rsvpEvent(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid event ID")
	}
	var cmd RSVPCommand
	if err := c.Bind(&cmd); err != nil {
		return shared.ErrBadRequest("invalid request body")
	}
	if err := c.Validate(&cmd); err != nil {
		return shared.ValidationError(err)
	}
	if err := h.svc.RSVPEvent(c.Request().Context(), &scope, eventID, cmd); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

func (h *Handler) removeRSVP(c echo.Context) error {
	scope, err := shared.GetFamilyScope(c)
	if err != nil {
		return err
	}
	eventID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return shared.ErrBadRequest("invalid event ID")
	}
	if err := h.svc.RemoveRSVP(c.Request().Context(), &scope, eventID); err != nil {
		return mapSocialError(err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ─── Error Mapping ───────────────────────────────────────────────────────────

// mapSocialError converts social domain errors to shared.AppError HTTP responses.
// Internal error details are never exposed to the client. [CODING §2.2]
func mapSocialError(err error) error {
	if err == nil {
		return nil
	}

	var socErr *SocialError
	if errors.As(err, &socErr) {
		return socErr.toAppError()
	}

	// Pass through AppError (already mapped, e.g. from shared package or cross-domain calls).
	var appErr *shared.AppError
	if errors.As(err, &appErr) {
		return appErr
	}

	// Default: internal error — log internally, never expose details. [CODING §2.2]
	return shared.ErrInternal(err)
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func parsePagination(c echo.Context) (offset, limit int) {
	offset = 0
	limit = 20
	if v := c.QueryParam("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if v := c.QueryParam("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 100 {
			limit = n
		}
	}
	return
}
