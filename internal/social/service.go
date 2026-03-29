package social

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social/domain"
	"gorm.io/gorm"
)

// ─── Service Implementation ──────────────────────────────────────────────────

type socialServiceImpl struct {
	profileRepo     ProfileRepository
	friendshipRepo  FriendshipRepository
	blockRepo       BlockRepository
	postRepo        PostRepository
	commentRepo     CommentRepository
	likeRepo        PostLikeRepository
	convRepo        ConversationRepository
	convPartRepo    ConversationParticipantRepository
	msgRepo         MessageRepository
	groupRepo       GroupRepository
	groupMemberRepo GroupMemberRepository
	pinnedPostRepo  PinnedPostRepository
	eventRepo       EventRepository
	rsvpRepo        EventRSVPRepository
	iam             IamServiceForSocial
	method          MethodServiceForSocial
	feedStore       shared.FeedStore
	pubsub          shared.PubSub
	jobs            shared.JobEnqueuer
	eventBus        *shared.EventBus
	db              *gorm.DB
}

// NewSocialService creates a new SocialService.
func NewSocialService(
	profileRepo ProfileRepository,
	friendshipRepo FriendshipRepository,
	blockRepo BlockRepository,
	postRepo PostRepository,
	commentRepo CommentRepository,
	likeRepo PostLikeRepository,
	convRepo ConversationRepository,
	convPartRepo ConversationParticipantRepository,
	msgRepo MessageRepository,
	groupRepo GroupRepository,
	groupMemberRepo GroupMemberRepository,
	pinnedPostRepo PinnedPostRepository,
	eventRepo EventRepository,
	rsvpRepo EventRSVPRepository,
	iam IamServiceForSocial,
	method MethodServiceForSocial,
	feedStore shared.FeedStore,
	pubsub shared.PubSub,
	jobs shared.JobEnqueuer,
	eventBus *shared.EventBus,
	db *gorm.DB,
) SocialService {
	return &socialServiceImpl{
		profileRepo:     profileRepo,
		friendshipRepo:  friendshipRepo,
		blockRepo:       blockRepo,
		postRepo:        postRepo,
		commentRepo:     commentRepo,
		likeRepo:        likeRepo,
		convRepo:        convRepo,
		convPartRepo:    convPartRepo,
		msgRepo:         msgRepo,
		groupRepo:       groupRepo,
		groupMemberRepo: groupMemberRepo,
		pinnedPostRepo:  pinnedPostRepo,
		eventRepo:       eventRepo,
		rsvpRepo:        rsvpRepo,
		iam:             iam,
		method:          method,
		feedStore:       feedStore,
		pubsub:          pubsub,
		jobs:            jobs,
		eventBus:        eventBus,
		db:              db,
	}
}

// methodologyDisplayName resolves a methodology slug to a display name.
// Falls back to slug on error (graceful degradation). [05-social §8.2]
func (s *socialServiceImpl) methodologyDisplayName(ctx context.Context, slug *string) *string {
	if slug == nil || *slug == "" {
		return nil
	}
	name, err := s.method.GetMethodologyDisplayName(ctx, *slug)
	if err != nil {
		return slug // graceful fallback to slug
	}
	return &name
}

// ─── Profile ────────────────────────────────────────────────────────────────

func (s *socialServiceImpl) CreateProfile(ctx context.Context, familyID uuid.UUID) error {
	// RLS bypass: called from FamilyCreated handler — no auth context available.
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		repo := &PgProfileRepository{db: tx}
		defaultSettings, _ := json.Marshal(domain.DefaultPrivacySettings())
		return repo.Create(ctx, &Profile{
			FamilyID:        familyID,
			PrivacySettings: defaultSettings,
		})
	})
}

func (s *socialServiceImpl) GetOwnProfile(ctx context.Context, scope *shared.FamilyScope) (*ProfileResponse, error) {
	var profile *Profile
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgProfileRepository{db: tx}
		var findErr error
		profile, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		return findErr
	})
	if err != nil {
		return nil, err
	}
	info, _ := s.iam.GetFamilyInfo(ctx, scope.FamilyID())
	resp := buildProfileResponse(profile, info, true, true)
	return resp, nil
}

func (s *socialServiceImpl) GetFamilyProfile(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*ProfileResponse, error) {
	// Block check first — blocked interactions return 404. [05-social §16]
	blocked, err := s.blockRepo.IsEitherBlocked(ctx, auth.FamilyID, targetFamilyID)
	if err != nil {
		return nil, err
	}
	if blocked {
		return nil, &SocialError{Err: domain.ErrContentNotVisible}
	}

	areFriends, err := s.friendshipRepo.AreFriends(ctx, auth.FamilyID, targetFamilyID)
	if err != nil {
		return nil, err
	}
	if !domain.CanViewProfile(auth.FamilyID, targetFamilyID, areFriends) {
		return nil, &SocialError{Err: domain.ErrContentNotVisible}
	}

	profile, err := s.profileRepo.FindByFamilyID(ctx, targetFamilyID)
	if err != nil {
		return nil, err
	}
	info, _ := s.iam.GetFamilyInfo(ctx, targetFamilyID)
	isOwn := auth.FamilyID == targetFamilyID
	resp := buildProfileResponse(profile, info, isOwn, areFriends)
	return resp, nil
}

func (s *socialServiceImpl) UpdateProfile(ctx context.Context, scope *shared.FamilyScope, cmd UpdateProfileCommand) (*ProfileResponse, error) {
	// Validate privacy settings before persisting. [05-social §4.1] (M1)
	if cmd.PrivacySettings != nil {
		var ps domain.PrivacySettings
		if err := json.Unmarshal(*cmd.PrivacySettings, &ps); err != nil {
			return nil, &SocialError{Err: domain.ErrInvalidPrivacySettings}
		}
		if err := domain.ValidatePrivacySettings(ps); err != nil {
			return nil, &SocialError{Err: err}
		}
	}

	var profile *Profile
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgProfileRepository{db: tx}
		var findErr error
		profile, findErr = repo.FindByFamilyID(ctx, scope.FamilyID())
		if findErr != nil {
			return findErr
		}
		if cmd.Bio != nil {
			profile.Bio = cmd.Bio
		}
		if cmd.ProfilePhotoURL != nil {
			profile.ProfilePhotoURL = cmd.ProfilePhotoURL
		}
		if cmd.PrivacySettings != nil {
			profile.PrivacySettings = *cmd.PrivacySettings
		}
		if cmd.LocationVisible != nil {
			profile.LocationVisible = *cmd.LocationVisible
		}
		return repo.Update(ctx, profile)
	})
	if err != nil {
		return nil, err
	}
	info, _ := s.iam.GetFamilyInfo(ctx, scope.FamilyID())
	resp := buildProfileResponse(profile, info, true, true)
	return resp, nil
}

// ─── Friends ────────────────────────────────────────────────────────────────

func (s *socialServiceImpl) SendFriendRequest(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) (*FriendshipResponse, error) {
	var friendship *Friendship
	// CROSS-FAMILY: friend requests involve two families.
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		blockRepo := &PgBlockRepository{db: tx}
		friendRepo := &PgFriendshipRepository{db: tx}

		blocked, blockErr := blockRepo.IsEitherBlocked(ctx, auth.FamilyID, targetFamilyID)
		if blockErr != nil {
			return blockErr
		}

		existing, findErr := friendRepo.FindBetween(ctx, auth.FamilyID, targetFamilyID)
		if findErr != nil {
			return findErr
		}
		var existingStatus *string
		if existing != nil {
			existingStatus = &existing.Status
		}

		if valErr := domain.ValidateSendRequest(auth.FamilyID, targetFamilyID, existingStatus, blocked); valErr != nil {
			return &SocialError{Err: valErr}
		}

		friendship = &Friendship{
			RequesterFamilyID: auth.FamilyID,
			AccepterFamilyID:  targetFamilyID,
			Status:            domain.FriendshipStatusPending,
		}
		if createErr := friendRepo.Create(ctx, friendship); createErr != nil {
			return createErr
		}

		_ = s.eventBus.Publish(ctx, FriendRequestSent{
			FriendshipID:      friendship.ID,
			RequesterFamilyID: auth.FamilyID,
			AccepterFamilyID:  targetFamilyID,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &FriendshipResponse{
		ID:                friendship.ID,
		RequesterFamilyID: friendship.RequesterFamilyID,
		AccepterFamilyID:  friendship.AccepterFamilyID,
		Status:            friendship.Status,
		CreatedAt:         friendship.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) AcceptFriendRequest(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) (*FriendshipResponse, error) {
	var friendship *Friendship
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		friendRepo := &PgFriendshipRepository{db: tx}

		var findErr error
		friendship, findErr = friendRepo.FindByID(ctx, friendshipID)
		if findErr != nil {
			return findErr
		}
		if valErr := domain.ValidateAcceptRequest(auth.FamilyID, friendship.AccepterFamilyID, friendship.Status); valErr != nil {
			return &SocialError{Err: valErr}
		}

		friendship.Status = domain.FriendshipStatusAccepted
		if updateErr := friendRepo.Update(ctx, friendship); updateErr != nil {
			return updateErr
		}

		_ = s.eventBus.Publish(ctx, FriendRequestAccepted{
			FriendshipID:      friendship.ID,
			RequesterFamilyID: friendship.RequesterFamilyID,
			AccepterFamilyID:  friendship.AccepterFamilyID,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// C4: Rebuild feeds for both families so existing posts appear. [05-social §10]
	_ = s.jobs.Enqueue(ctx, FeedRebuildPayload{
		FamilyID:       friendship.RequesterFamilyID,
		FriendFamilyID: friendship.AccepterFamilyID,
	})
	_ = s.jobs.Enqueue(ctx, FeedRebuildPayload{
		FamilyID:       friendship.AccepterFamilyID,
		FriendFamilyID: friendship.RequesterFamilyID,
	})

	return &FriendshipResponse{
		ID:                friendship.ID,
		RequesterFamilyID: friendship.RequesterFamilyID,
		AccepterFamilyID:  friendship.AccepterFamilyID,
		Status:            friendship.Status,
		CreatedAt:         friendship.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) RejectFriendRequest(ctx context.Context, auth *shared.AuthContext, friendshipID uuid.UUID) error {
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		friendRepo := &PgFriendshipRepository{db: tx}

		friendship, findErr := friendRepo.FindByID(ctx, friendshipID)
		if findErr != nil {
			return findErr
		}
		if valErr := domain.ValidateRejectRequest(auth.FamilyID, friendship.AccepterFamilyID, friendship.Status); valErr != nil {
			return &SocialError{Err: valErr}
		}
		return friendRepo.Delete(ctx, friendshipID)
	})
}

func (s *socialServiceImpl) Unfriend(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error {
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		friendRepo := &PgFriendshipRepository{db: tx}

		areFriends, checkErr := friendRepo.AreFriends(ctx, auth.FamilyID, targetFamilyID)
		if checkErr != nil {
			return checkErr
		}
		if !areFriends {
			return &SocialError{Err: domain.ErrNotFriends}
		}
		return friendRepo.DeleteBetween(ctx, auth.FamilyID, targetFamilyID)
	})
}

func (s *socialServiceImpl) BlockFamily(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error {
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		blockRepo := &PgBlockRepository{db: tx}
		friendRepo := &PgFriendshipRepository{db: tx}

		alreadyBlocked, checkErr := blockRepo.IsBlocked(ctx, auth.FamilyID, targetFamilyID)
		if checkErr != nil {
			return checkErr
		}
		if valErr := domain.ValidateBlock(auth.FamilyID, targetFamilyID, alreadyBlocked); valErr != nil {
			return &SocialError{Err: valErr}
		}

		// Create block.
		if createErr := blockRepo.Create(ctx, &Block{
			BlockerFamilyID: auth.FamilyID,
			BlockedFamilyID: targetFamilyID,
		}); createErr != nil {
			return createErr
		}

		// Remove friendship if exists. [05-social §16]
		_ = friendRepo.DeleteBetween(ctx, auth.FamilyID, targetFamilyID)

		return nil
	})
	if err != nil {
		return err
	}

	// Purge blocked family's posts from user's Redis feed. [05-social §16] (C2)
	// Best-effort: query recent posts by blocked family, then remove from feed.
	blockedPosts, _ := s.postRepo.ListByFamilyIDs(ctx, []uuid.UUID{targetFamilyID}, 0, 1000)
	if len(blockedPosts) > 0 {
		postIDs := make([]string, len(blockedPosts))
		for i, p := range blockedPosts {
			postIDs[i] = p.ID.String()
		}
		_ = s.feedStore.RemoveFromFeedByFamily(ctx, auth.FamilyID.String(), postIDs)
	}
	return nil
}

func (s *socialServiceImpl) UnblockFamily(ctx context.Context, auth *shared.AuthContext, targetFamilyID uuid.UUID) error {
	return shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		blockRepo := &PgBlockRepository{db: tx}

		isBlocked, checkErr := blockRepo.IsBlocked(ctx, auth.FamilyID, targetFamilyID)
		if checkErr != nil {
			return checkErr
		}
		if !isBlocked {
			return &SocialError{Err: domain.ErrNotBlocked}
		}
		return blockRepo.Delete(ctx, auth.FamilyID, targetFamilyID)
	})
}

func (s *socialServiceImpl) ListFriends(ctx context.Context, scope *shared.FamilyScope, cursor *uuid.UUID, limit int) ([]FriendResponse, error) {
	// CROSS-FAMILY: friend list involves reading other families' profiles. Uses cursor pagination. (H6)
	friendships, err := s.friendshipRepo.ListFriendsCursor(ctx, scope.FamilyID(), cursor, limit)
	if err != nil {
		return nil, err
	}
	results := make([]FriendResponse, 0, len(friendships))
	for _, f := range friendships {
		friendFamilyID := f.AccepterFamilyID
		if f.AccepterFamilyID == scope.FamilyID() {
			friendFamilyID = f.RequesterFamilyID
		}
		name, _ := s.iam.GetFamilyDisplayName(ctx, friendFamilyID)
		var photoURL *string
		if profile, pErr := s.profileRepo.FindByFamilyID(ctx, friendFamilyID); pErr == nil && profile != nil {
			photoURL = profile.ProfilePhotoURL
		}
		results = append(results, FriendResponse{
			FamilyID:        friendFamilyID,
			DisplayName:     name,
			ProfilePhotoURL: photoURL,
			FriendsSince:    f.UpdatedAt,
		})
	}
	return results, nil
}

func (s *socialServiceImpl) ListIncomingRequests(ctx context.Context, scope *shared.FamilyScope) ([]FriendRequestResponse, error) {
	friendships, err := s.friendshipRepo.ListIncoming(ctx, scope.FamilyID())
	if err != nil {
		return nil, err
	}
	results := make([]FriendRequestResponse, 0, len(friendships))
	for _, f := range friendships {
		name, _ := s.iam.GetFamilyDisplayName(ctx, f.RequesterFamilyID)
		var photoURL *string
		if profile, pErr := s.profileRepo.FindByFamilyID(ctx, f.RequesterFamilyID); pErr == nil && profile != nil {
			photoURL = profile.ProfilePhotoURL
		}
		results = append(results, FriendRequestResponse{
			FriendshipID:    f.ID,
			FamilyID:        f.RequesterFamilyID,
			DisplayName:     name,
			ProfilePhotoURL: photoURL,
			CreatedAt:       f.CreatedAt,
		})
	}
	return results, nil
}

func (s *socialServiceImpl) ListOutgoingRequests(ctx context.Context, scope *shared.FamilyScope) ([]FriendRequestResponse, error) {
	friendships, err := s.friendshipRepo.ListOutgoing(ctx, scope.FamilyID())
	if err != nil {
		return nil, err
	}
	results := make([]FriendRequestResponse, 0, len(friendships))
	for _, f := range friendships {
		name, _ := s.iam.GetFamilyDisplayName(ctx, f.AccepterFamilyID)
		var photoURL *string
		if profile, pErr := s.profileRepo.FindByFamilyID(ctx, f.AccepterFamilyID); pErr == nil && profile != nil {
			photoURL = profile.ProfilePhotoURL
		}
		results = append(results, FriendRequestResponse{
			FriendshipID:    f.ID,
			FamilyID:        f.AccepterFamilyID,
			DisplayName:     name,
			ProfilePhotoURL: photoURL,
			CreatedAt:       f.CreatedAt,
		})
	}
	return results, nil
}

func (s *socialServiceImpl) ListBlocks(ctx context.Context, scope *shared.FamilyScope) ([]BlockedFamilyResponse, error) {
	var blocks []Block
	err := shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		repo := &PgBlockRepository{db: tx}
		var findErr error
		blocks, findErr = repo.ListByBlocker(ctx, scope.FamilyID())
		return findErr
	})
	if err != nil {
		return nil, err
	}
	results := make([]BlockedFamilyResponse, 0, len(blocks))
	for _, b := range blocks {
		name, _ := s.iam.GetFamilyDisplayName(ctx, b.BlockedFamilyID)
		results = append(results, BlockedFamilyResponse{
			FamilyID:    b.BlockedFamilyID,
			DisplayName: name,
			BlockedAt:   b.CreatedAt,
		})
	}
	return results, nil
}

// ─── Posts ───────────────────────────────────────────────────────────────────

func (s *socialServiceImpl) CreatePost(ctx context.Context, auth *shared.AuthContext, cmd CreatePostCommand) (*PostResponse, error) {
	hasAttachments := len(cmd.Attachments) > 0 && string(cmd.Attachments) != "[]" && string(cmd.Attachments) != "null"
	if valErr := domain.ValidatePostCreate(cmd.PostType, cmd.Content, hasAttachments); valErr != nil {
		return nil, &SocialError{Err: valErr}
	}

	visibility := domain.ResolvePostVisibility(cmd.GroupID)

	// If posting to a group, verify membership. [05-social §9]
	if cmd.GroupID != nil {
		isMember, memberErr := s.groupMemberRepo.IsMember(ctx, *cmd.GroupID, auth.FamilyID)
		if memberErr != nil {
			return nil, memberErr
		}
		if !isMember {
			return nil, &SocialError{Err: domain.ErrNotGroupMember}
		}
	}

	attachments := cmd.Attachments
	if attachments == nil {
		attachments = json.RawMessage("[]")
	}

	post := &Post{
		FamilyID:       auth.FamilyID,
		AuthorParentID: auth.ParentID,
		PostType:       cmd.PostType,
		Content:        cmd.Content,
		Attachments:    attachments,
		GroupID:        cmd.GroupID,
		Visibility:     visibility,
	}

	scope := shared.NewFamilyScopeFromAuth(auth)
	err := shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		repo := &PgPostRepository{db: tx}
		return repo.Create(ctx, post)
	})
	if err != nil {
		return nil, err
	}

	// Fan-out to friends' feeds via async job. [05-social §11]
	if visibility == domain.VisibilityFriends {
		_ = s.jobs.Enqueue(ctx, FanOutPostPayload{
			PostID:   post.ID,
			FamilyID: post.FamilyID,
			ScoreMs:  float64(post.CreatedAt.UnixMilli()),
		})
	}

	_ = s.eventBus.Publish(ctx, PostCreated{
		PostID:      post.ID,
		FamilyID:    post.FamilyID,
		PostType:    post.PostType,
		Content:     post.Content,
		Attachments: post.Attachments,
		GroupID:     post.GroupID,
	})

	authorName, _ := s.iam.GetParentDisplayName(ctx, auth.ParentID)
	return &PostResponse{
		ID:            post.ID,
		FamilyID:      post.FamilyID,
		AuthorName:    authorName,
		PostType:      post.PostType,
		Content:       post.Content,
		Attachments:   post.Attachments,
		GroupID:       post.GroupID,
		Visibility:    post.Visibility,
		LikesCount:    0,
		CommentsCount: 0,
		IsLikedByMe:   false,
		CreatedAt:     post.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) UpdatePost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd UpdatePostCommand) (*PostResponse, error) {
	if cmd.Content == nil && cmd.Attachments == nil {
		return nil, &SocialError{Err: domain.ErrPostEditEmpty}
	}

	var updated Post
	scope := shared.NewFamilyScopeFromAuth(auth)
	err := shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		repo := &PgPostRepository{db: tx}
		post, findErr := repo.FindByID(ctx, postID)
		if findErr != nil {
			return findErr
		}
		if post.FamilyID != auth.FamilyID {
			return &SocialError{Err: domain.ErrCannotEditPost}
		}
		if cmd.Content != nil {
			post.Content = cmd.Content
		}
		if cmd.Attachments != nil {
			post.Attachments = *cmd.Attachments
		}
		post.IsEdited = true
		if updateErr := repo.Update(ctx, post); updateErr != nil {
			return updateErr
		}
		updated = *post
		return nil
	})
	if err != nil {
		return nil, err
	}

	authorName, _ := s.iam.GetParentDisplayName(ctx, auth.ParentID)
	return &PostResponse{
		ID:            updated.ID,
		FamilyID:      updated.FamilyID,
		AuthorName:    authorName,
		PostType:      updated.PostType,
		Content:       updated.Content,
		Attachments:   updated.Attachments,
		GroupID:       updated.GroupID,
		Visibility:    updated.Visibility,
		LikesCount:    updated.LikesCount,
		CommentsCount: updated.CommentsCount,
		IsEdited:      true,
		IsLikedByMe:   false,
		CreatedAt:     updated.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) DeletePost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) error {
	scope := shared.NewFamilyScopeFromAuth(auth)
	return shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		repo := &PgPostRepository{db: tx}
		post, findErr := repo.FindByID(ctx, postID)
		if findErr != nil {
			return findErr
		}
		if post.FamilyID != auth.FamilyID {
			return &SocialError{Err: domain.ErrCannotDeletePost}
		}
		return repo.Delete(ctx, postID)
	})
}

func (s *socialServiceImpl) LikePost(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error {
	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		likeRepo := &PgPostLikeRepository{db: tx}
		postRepo := &PgPostRepository{db: tx}

		exists, checkErr := likeRepo.Exists(ctx, postID, scope.FamilyID())
		if checkErr != nil {
			return checkErr
		}
		if exists {
			return &SocialError{Err: domain.ErrAlreadyLiked}
		}
		if createErr := likeRepo.Create(ctx, &PostLike{
			PostID:   postID,
			FamilyID: scope.FamilyID(),
		}); createErr != nil {
			return createErr
		}
		return postRepo.IncrementLikes(ctx, postID)
	})
}

func (s *socialServiceImpl) UnlikePost(ctx context.Context, scope *shared.FamilyScope, postID uuid.UUID) error {
	return shared.ScopedTransaction(ctx, s.db, *scope, func(tx *gorm.DB) error {
		likeRepo := &PgPostLikeRepository{db: tx}
		postRepo := &PgPostRepository{db: tx}

		if delErr := likeRepo.Delete(ctx, postID, scope.FamilyID()); delErr != nil {
			return delErr
		}
		return postRepo.DecrementLikes(ctx, postID)
	})
}

func (s *socialServiceImpl) GetPost(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) (*PostDetailResponse, error) {
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return nil, err
	}

	// Block check. [05-social §16]
	if post.FamilyID != auth.FamilyID {
		blocked, blockErr := s.blockRepo.IsEitherBlocked(ctx, auth.FamilyID, post.FamilyID)
		if blockErr != nil {
			return nil, blockErr
		}
		if blocked {
			return nil, &SocialError{Err: domain.ErrContentNotVisible}
		}
	}

	// Visibility check.
	areFriends, _ := s.friendshipRepo.AreFriends(ctx, auth.FamilyID, post.FamilyID)
	isGroupMember := false
	if post.GroupID != nil {
		isGroupMember, _ = s.groupMemberRepo.IsMember(ctx, *post.GroupID, auth.FamilyID)
	}
	if !domain.CanViewPost(auth.FamilyID, post.FamilyID, post.Visibility, areFriends, isGroupMember) {
		return nil, &SocialError{Err: domain.ErrContentNotVisible}
	}

	authorName, _ := s.iam.GetParentDisplayName(ctx, post.AuthorParentID)
	liked, _ := s.likeRepo.Exists(ctx, postID, auth.FamilyID)

	// Embed comments in the detail response. [05-social §8.2]
	comments, err := s.commentRepo.ListByPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	commentResponses := make([]CommentResponse, 0, len(comments))
	for _, c := range comments {
		cAuthor, _ := s.iam.GetParentDisplayName(ctx, c.AuthorParentID)
		commentResponses = append(commentResponses, CommentResponse{
			ID:              c.ID,
			PostID:          c.PostID,
			FamilyID:        c.FamilyID,
			AuthorName:      cAuthor,
			ParentCommentID: c.ParentCommentID,
			Content:         c.Content,
			CreatedAt:       c.CreatedAt,
		})
	}

	return &PostDetailResponse{
		Post: PostResponse{
			ID:            post.ID,
			FamilyID:      post.FamilyID,
			AuthorName:    authorName,
			PostType:      post.PostType,
			Content:       post.Content,
			Attachments:   post.Attachments,
			GroupID:       post.GroupID,
			Visibility:    post.Visibility,
			LikesCount:    post.LikesCount,
			CommentsCount: post.CommentsCount,
			IsEdited:      post.IsEdited,
			IsLikedByMe:   liked,
			CreatedAt:     post.CreatedAt,
		},
		Comments: commentResponses,
	}, nil
}

func (s *socialServiceImpl) GetFeed(ctx context.Context, auth *shared.AuthContext, offset, limit int) (*FeedResponse, error) {
	// Get all friend IDs for feed query (unbounded — need all friends for fan-out). (H13)
	friendIDs, err := s.friendshipRepo.ListFriendFamilyIDs(ctx, auth.FamilyID)
	if err != nil {
		return nil, err
	}

	// Include own posts in feed.
	allFamilyIDs := append([]uuid.UUID{auth.FamilyID}, friendIDs...)

	// Try Redis feed first, fall back to PostgreSQL. [05-social §11]
	var posts []Post
	redisIDs, redisErr := s.feedStore.GetFeed(ctx, auth.FamilyID.String(), int64(offset), int64(limit))
	if redisErr == nil && len(redisIDs) > 0 {
		postIDs := make([]uuid.UUID, 0, len(redisIDs))
		for _, idStr := range redisIDs {
			if id, parseErr := uuid.Parse(idStr); parseErr == nil {
				postIDs = append(postIDs, id)
			}
		}
		var findErr error
		posts, findErr = s.postRepo.FindByIDs(ctx, postIDs)
		if findErr != nil {
			slog.Warn("social: Redis-backed FindByIDs failed, falling back to PostgreSQL", "error", findErr)
		}
	}

	// PostgreSQL fallback if Redis is empty or errored.
	if len(posts) == 0 {
		posts, err = s.postRepo.ListByFamilyIDs(ctx, allFamilyIDs, offset, limit)
		if err != nil {
			return nil, err
		}
	}

	// Filter out blocked families. [05-social §16]
	filtered := make([]Post, 0, len(posts))
	for _, p := range posts {
		if p.FamilyID == auth.FamilyID {
			filtered = append(filtered, p)
			continue
		}
		blocked, _ := s.blockRepo.IsEitherBlocked(ctx, auth.FamilyID, p.FamilyID)
		if !blocked {
			filtered = append(filtered, p)
		}
	}

	// Batch check likes.
	postIDs := make([]uuid.UUID, len(filtered))
	for i, p := range filtered {
		postIDs[i] = p.ID
	}
	likedMap, _ := s.likeRepo.ListByPostIDs(ctx, postIDs, auth.FamilyID)

	// Build response.
	respPosts := make([]PostResponse, len(filtered))
	for i, p := range filtered {
		authorName, _ := s.iam.GetParentDisplayName(ctx, p.AuthorParentID)
		respPosts[i] = PostResponse{
			ID:            p.ID,
			FamilyID:      p.FamilyID,
			AuthorName:    authorName,
			PostType:      p.PostType,
			Content:       p.Content,
			Attachments:   p.Attachments,
			GroupID:       p.GroupID,
			Visibility:    p.Visibility,
			LikesCount:    p.LikesCount,
			CommentsCount: p.CommentsCount,
			IsEdited:      p.IsEdited,
			IsLikedByMe:   likedMap[p.ID],
			CreatedAt:     p.CreatedAt,
		}
	}

	return &FeedResponse{Posts: respPosts}, nil
}

// ─── Comments ───────────────────────────────────────────────────────────────

func (s *socialServiceImpl) CreateComment(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID, cmd CreateCommentCommand) (*CommentResponse, error) {
	// Validate threading — parent comment must be top-level AND belong to the same post. (C5)
	if cmd.ParentCommentID != nil {
		parent, findErr := s.commentRepo.FindByID(ctx, *cmd.ParentCommentID)
		if findErr != nil {
			return nil, findErr
		}
		if valErr := domain.ValidateCommentThread(parent.ParentCommentID != nil); valErr != nil {
			return nil, &SocialError{Err: valErr}
		}
		if valErr := domain.ValidateCommentSamePost(parent.PostID, postID); valErr != nil {
			return nil, &SocialError{Err: valErr}
		}
	}

	comment := &Comment{
		PostID:          postID,
		FamilyID:        auth.FamilyID,
		AuthorParentID:  auth.ParentID,
		ParentCommentID: cmd.ParentCommentID,
		Content:         cmd.Content,
	}

	scope := shared.NewFamilyScopeFromAuth(auth)
	err := shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		commentRepo := &PgCommentRepository{db: tx}
		postRepo := &PgPostRepository{db: tx}
		if createErr := commentRepo.Create(ctx, comment); createErr != nil {
			return createErr
		}
		return postRepo.IncrementComments(ctx, postID)
	})
	if err != nil {
		return nil, err
	}

	authorName, _ := s.iam.GetParentDisplayName(ctx, auth.ParentID)
	return &CommentResponse{
		ID:              comment.ID,
		PostID:          comment.PostID,
		FamilyID:        comment.FamilyID,
		AuthorName:      authorName,
		ParentCommentID: comment.ParentCommentID,
		Content:         comment.Content,
		CreatedAt:       comment.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) DeleteComment(ctx context.Context, auth *shared.AuthContext, commentID uuid.UUID) error {
	comment, err := s.commentRepo.FindByID(ctx, commentID)
	if err != nil {
		return err
	}

	// Author or post author can delete. [05-social §7.3]
	post, postErr := s.postRepo.FindByID(ctx, comment.PostID)
	if postErr != nil {
		return postErr
	}
	if comment.FamilyID != auth.FamilyID && post.FamilyID != auth.FamilyID {
		return &SocialError{Err: domain.ErrCannotDeleteComment}
	}

	scope := shared.NewFamilyScopeFromAuth(auth)
	return shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		commentRepo := &PgCommentRepository{db: tx}
		postRepo := &PgPostRepository{db: tx}
		if delErr := commentRepo.Delete(ctx, commentID); delErr != nil {
			return delErr
		}
		return postRepo.DecrementComments(ctx, comment.PostID)
	})
}

func (s *socialServiceImpl) ListComments(ctx context.Context, auth *shared.AuthContext, postID uuid.UUID) ([]CommentResponse, error) {
	comments, err := s.commentRepo.ListByPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	results := make([]CommentResponse, len(comments))
	for i, c := range comments {
		authorName, _ := s.iam.GetParentDisplayName(ctx, c.AuthorParentID)
		results[i] = CommentResponse{
			ID:              c.ID,
			PostID:          c.PostID,
			FamilyID:        c.FamilyID,
			AuthorName:      authorName,
			ParentCommentID: c.ParentCommentID,
			Content:         c.Content,
			CreatedAt:       c.CreatedAt,
		}
	}
	return results, nil
}

// ─── Messaging ──────────────────────────────────────────────────────────────

func (s *socialServiceImpl) CreateConversation(ctx context.Context, auth *shared.AuthContext, cmd CreateConversationCommand) (*ConversationResponse, error) {
	// Self-message guard. [05-social §12]
	if cmd.RecipientParentID == auth.ParentID {
		return nil, &SocialError{Err: domain.ErrCannotMessageSelf}
	}

	// Friends-only guard for DMs. [05-social §12]
	recipientInfo, infoErr := s.iam.GetParentInfo(ctx, cmd.RecipientParentID)
	if infoErr != nil {
		return nil, infoErr
	}
	areFriends, friendErr := s.friendshipRepo.AreFriends(ctx, auth.FamilyID, recipientInfo.FamilyID)
	if friendErr != nil {
		return nil, friendErr
	}
	if !areFriends {
		return nil, &SocialError{Err: domain.ErrNotFriendsForDM}
	}

	// Create-or-get: reuse existing conversation between these parents. [05-social §12]
	existingID, findErr := s.convPartRepo.FindBetweenParents(ctx, auth.ParentID, cmd.RecipientParentID)
	if findErr != nil {
		return nil, findErr
	}
	if existingID != nil {
		conv, convErr := s.convRepo.FindByID(ctx, *existingID)
		if convErr != nil {
			return nil, convErr
		}
		return &ConversationResponse{
			ID:        conv.ID,
			UpdatedAt: conv.UpdatedAt,
		}, nil
	}

	var conv *Conversation
	// CROSS-FAMILY: conversations span multiple families.
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		convRepo := &PgConversationRepository{db: tx}
		partRepo := &PgConversationParticipantRepository{db: tx}

		conv = &Conversation{}
		if createErr := convRepo.Create(ctx, conv); createErr != nil {
			return createErr
		}

		// Add the initiator as participant.
		if partErr := partRepo.Create(ctx, &ConversationParticipant{
			ConversationID: conv.ID,
			ParentID:       auth.ParentID,
			FamilyID:       auth.FamilyID,
		}); partErr != nil {
			return partErr
		}

		// Add the recipient as participant.
		return partRepo.Create(ctx, &ConversationParticipant{
			ConversationID: conv.ID,
			ParentID:       cmd.RecipientParentID,
			FamilyID:       recipientInfo.FamilyID,
		})
	})
	if err != nil {
		return nil, err
	}

	return &ConversationResponse{
		ID:        conv.ID,
		UpdatedAt: conv.UpdatedAt,
		IsNew:     true,
	}, nil
}

func (s *socialServiceImpl) SendMessage(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, cmd SendMessageCommand) (*MessageResponse, error) {
	// Verify participant. [05-social §12]
	isParticipant, err := s.convPartRepo.IsParticipant(ctx, conversationID, auth.ParentID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, &SocialError{Err: domain.ErrNotConversationParticipant}
	}

	attachments := cmd.Attachments
	if attachments == nil {
		attachments = json.RawMessage("[]")
	}

	msg := &Message{
		ConversationID: conversationID,
		SenderParentID: auth.ParentID,
		SenderFamilyID: auth.FamilyID,
		Content:        cmd.Content,
		Attachments:    attachments,
	}

	scope := shared.NewFamilyScopeFromAuth(auth)
	err = shared.ScopedTransaction(ctx, s.db, scope, func(tx *gorm.DB) error {
		msgRepo := &PgMessageRepository{db: tx}
		return msgRepo.Create(ctx, msg)
	})
	if err != nil {
		return nil, err
	}

	// Push via WebSocket to other participants + clear deleted_at. [05-social §4.1] (C3)
	participants, _ := s.convPartRepo.ListByConversation(ctx, conversationID)
	var recipientParentID uuid.UUID
	var recipientFamilyID uuid.UUID
	for _, p := range participants {
		if p.ParentID != auth.ParentID {
			// Clear deleted_at: new message restores conversation for recipient.
			_ = s.convPartRepo.ClearDeletedAt(ctx, conversationID, p.ParentID)
			publishToParent(s.pubsub, p.ParentID, "new_message", msg)
			recipientParentID = p.ParentID
			recipientFamilyID = p.FamilyID
		}
	}

	_ = s.eventBus.Publish(ctx, MessageSent{
		MessageID:         msg.ID,
		ConversationID:    conversationID,
		SenderParentID:    auth.ParentID,
		SenderFamilyID:    auth.FamilyID,
		RecipientParentID: recipientParentID,
		RecipientFamilyID: recipientFamilyID,
	})

	senderName, _ := s.iam.GetParentDisplayName(ctx, auth.ParentID)
	return &MessageResponse{
		ID:             msg.ID,
		ConversationID: msg.ConversationID,
		SenderParentID: msg.SenderParentID,
		SenderName:     senderName,
		Content:        msg.Content,
		Attachments:    msg.Attachments,
		CreatedAt:      msg.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) MarkConversationRead(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error {
	return s.convPartRepo.UpdateLastRead(ctx, conversationID, auth.ParentID)
}

func (s *socialServiceImpl) DeleteConversation(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID) error {
	isParticipant, err := s.convPartRepo.IsParticipant(ctx, conversationID, auth.ParentID)
	if err != nil {
		return err
	}
	if !isParticipant {
		return &SocialError{Err: domain.ErrNotConversationParticipant}
	}
	return s.convPartRepo.SoftDelete(ctx, conversationID, auth.ParentID)
}

func (s *socialServiceImpl) ReportMessage(ctx context.Context, auth *shared.AuthContext, messageID uuid.UUID, cmd ReportMessageCommand) error {
	msg, err := s.msgRepo.FindByID(ctx, messageID)
	if err != nil {
		return err
	}
	_ = s.eventBus.Publish(ctx, MessageReported{
		MessageID:        msg.ID,
		ConversationID:   msg.ConversationID,
		ReporterFamilyID: auth.FamilyID,
		SenderParentID:   msg.SenderParentID,
		Reason:           cmd.Reason,
	})
	return nil
}

func (s *socialServiceImpl) ListConversations(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]ConversationSummaryResponse, error) {
	convs, err := s.convRepo.ListByParent(ctx, auth.ParentID, offset, limit)
	if err != nil {
		return nil, err
	}
	results := make([]ConversationSummaryResponse, 0, len(convs))
	for _, conv := range convs {
		participants, _ := s.convPartRepo.ListByConversation(ctx, conv.ID)
		var myLastRead *time.Time
		var otherParentName string
		for _, p := range participants {
			if p.ParentID == auth.ParentID {
				myLastRead = p.LastReadAt
			} else {
				name, _ := s.iam.GetParentDisplayName(ctx, p.ParentID)
				otherParentName = name
			}
		}

		lastMsg, _ := s.msgRepo.LastByConversation(ctx, conv.ID)
		var lastPreview *string
		if lastMsg != nil {
			lastPreview = &lastMsg.Content
		}

		unread, _ := s.msgRepo.CountUnread(ctx, conv.ID, myLastRead)

		results = append(results, ConversationSummaryResponse{
			ID:                 conv.ID,
			OtherParentName:    otherParentName,
			LastMessagePreview: lastPreview,
			UnreadCount:        unread,
			UpdatedAt:          conv.UpdatedAt,
		})
	}
	return results, nil
}

func (s *socialServiceImpl) GetConversationMessages(ctx context.Context, auth *shared.AuthContext, conversationID uuid.UUID, offset, limit int) ([]MessageResponse, error) {
	isParticipant, err := s.convPartRepo.IsParticipant(ctx, conversationID, auth.ParentID)
	if err != nil {
		return nil, err
	}
	if !isParticipant {
		return nil, &SocialError{Err: domain.ErrNotConversationParticipant}
	}

	msgs, err := s.msgRepo.ListByConversation(ctx, conversationID, offset, limit)
	if err != nil {
		return nil, err
	}
	results := make([]MessageResponse, len(msgs))
	for i, m := range msgs {
		senderName, _ := s.iam.GetParentDisplayName(ctx, m.SenderParentID)
		results[i] = MessageResponse{
			ID:             m.ID,
			ConversationID: m.ConversationID,
			SenderParentID: m.SenderParentID,
			SenderName:     senderName,
			Content:        m.Content,
			Attachments:    m.Attachments,
			CreatedAt:      m.CreatedAt,
		}
	}
	return results, nil
}

// ─── Groups ─────────────────────────────────────────────────────────────────

func (s *socialServiceImpl) JoinGroup(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return err
	}

	existing, _ := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, scope.FamilyID())
	if existing != nil {
		if existing.Status == domain.GroupMemberStatusActive {
			return &SocialError{Err: domain.ErrAlreadyGroupMember}
		}
		if existing.Status == domain.GroupMemberStatusBanned {
			return &SocialError{Err: domain.ErrMemberBanned}
		}
		if existing.Status == domain.GroupMemberStatusPending {
			return &SocialError{Err: domain.ErrMemberPending}
		}
	}

	status, joinErr := domain.ResolveJoinAction(group.JoinPolicy)
	if joinErr != nil {
		return &SocialError{Err: joinErr}
	}

	now := time.Now()
	member := &GroupMember{
		GroupID:  groupID,
		FamilyID: scope.FamilyID(),
		Role:    domain.GroupRoleMember,
		Status:  status,
	}
	if status == domain.GroupMemberStatusActive {
		member.JoinedAt = &now
	}

	if createErr := s.groupMemberRepo.Create(ctx, member); createErr != nil {
		return createErr
	}
	if status == domain.GroupMemberStatusActive {
		_ = s.groupRepo.IncrementMemberCount(ctx, groupID)
	}
	return nil
}

func (s *socialServiceImpl) LeaveGroup(ctx context.Context, scope *shared.FamilyScope, groupID uuid.UUID) error {
	member, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, scope.FamilyID())
	if err != nil {
		return err
	}
	if member == nil {
		return &SocialError{Err: domain.ErrGroupMemberNotFound}
	}
	if valErr := domain.ValidateLeave(member.Role); valErr != nil {
		return &SocialError{Err: valErr}
	}

	if delErr := s.groupMemberRepo.Delete(ctx, groupID, scope.FamilyID()); delErr != nil {
		return delErr
	}
	if member.Status == domain.GroupMemberStatusActive {
		_ = s.groupRepo.DecrementMemberCount(ctx, groupID)
	}
	return nil
}

func (s *socialServiceImpl) CreateGroup(ctx context.Context, auth *shared.AuthContext, cmd CreateGroupCommand) (*GroupResponse, error) {
	group := &Group{
		GroupType:       domain.GroupTypeUserCreated,
		Name:            cmd.Name,
		Description:     cmd.Description,
		CoverPhotoURL:   cmd.CoverPhotoURL,
		CreatorFamilyID: &auth.FamilyID,
		MethodologySlug: cmd.MethodologySlug,
		JoinPolicy:      cmd.JoinPolicy,
		MemberCount:     1,
	}

	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		groupRepo := &PgGroupRepository{db: tx}
		memberRepo := &PgGroupMemberRepository{db: tx}

		if createErr := groupRepo.Create(ctx, group); createErr != nil {
			return createErr
		}

		now := time.Now()
		return memberRepo.Create(ctx, &GroupMember{
			GroupID:  group.ID,
			FamilyID: auth.FamilyID,
			Role:    domain.GroupRoleOwner,
			Status:  domain.GroupMemberStatusActive,
			JoinedAt: &now,
		})
	})
	if err != nil {
		return nil, err
	}

	ownerRole := domain.GroupRoleOwner
	activeStatus := domain.GroupMemberStatusActive
	return &GroupResponse{
		Summary: GroupSummaryResponse{
			ID:              group.ID,
			GroupType:       group.GroupType,
			Name:            group.Name,
			Description:     group.Description,
			CoverPhotoURL:   group.CoverPhotoURL,
			MethodologyName: s.methodologyDisplayName(ctx, group.MethodologySlug),
			JoinPolicy:      group.JoinPolicy,
			MemberCount:     group.MemberCount,
			IsMember:        true,
		},
		MyRole:    &ownerRole,
		MyStatus:  &activeStatus,
		CreatedAt: group.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) UpdateGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, cmd UpdateGroupCommand) (*GroupResponse, error) {
	member, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, auth.FamilyID)
	if err != nil {
		return nil, err
	}
	if member == nil || !domain.CanModerate(member.Role) {
		return nil, &SocialError{Err: domain.ErrInsufficientGroupRole}
	}

	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if cmd.Name != nil {
		group.Name = *cmd.Name
	}
	if cmd.Description != nil {
		group.Description = cmd.Description
	}
	if cmd.CoverPhotoURL != nil {
		group.CoverPhotoURL = cmd.CoverPhotoURL
	}
	if cmd.JoinPolicy != nil {
		group.JoinPolicy = *cmd.JoinPolicy
	}
	if updateErr := s.groupRepo.Update(ctx, group); updateErr != nil {
		return nil, updateErr
	}

	return &GroupResponse{
		Summary: GroupSummaryResponse{
			ID:              group.ID,
			GroupType:       group.GroupType,
			Name:            group.Name,
			Description:     group.Description,
			CoverPhotoURL:   group.CoverPhotoURL,
			MethodologyName: s.methodologyDisplayName(ctx, group.MethodologySlug),
			JoinPolicy:      group.JoinPolicy,
			MemberCount:     group.MemberCount,
			IsMember:        true,
		},
		MyRole:    &member.Role,
		MyStatus:  &member.Status,
		CreatedAt: group.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) DeleteGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) error {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return err
	}
	if group == nil {
		return &SocialError{Err: domain.ErrGroupNotFound}
	}
	if group.GroupType == "platform" {
		return &SocialError{Err: domain.ErrCannotDeletePlatformGroup}
	}
	member, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, auth.FamilyID)
	if err != nil {
		return err
	}
	if member == nil || member.Role != domain.GroupRoleOwner {
		return &SocialError{Err: domain.ErrInsufficientGroupRole}
	}
	return s.groupRepo.Delete(ctx, groupID)
}

func (s *socialServiceImpl) ApproveMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	return s.moderateGroupAction(ctx, auth, groupID, familyID, func(member *GroupMember) error {
		if member.Status != domain.GroupMemberStatusPending {
			return &SocialError{Err: domain.ErrMemberPending}
		}
		now := time.Now()
		member.Status = domain.GroupMemberStatusActive
		member.JoinedAt = &now
		if updateErr := s.groupMemberRepo.Update(ctx, member); updateErr != nil {
			return updateErr
		}
		return s.groupRepo.IncrementMemberCount(ctx, groupID)
	})
}

func (s *socialServiceImpl) RejectMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	return s.moderateGroupAction(ctx, auth, groupID, familyID, func(member *GroupMember) error {
		return s.groupMemberRepo.Delete(ctx, groupID, familyID)
	})
}

func (s *socialServiceImpl) BanMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	return s.moderateGroupAction(ctx, auth, groupID, familyID, func(member *GroupMember) error {
		if banErr := domain.ValidateBan(member.Role); banErr != nil {
			return &SocialError{Err: banErr}
		}
		wasActive := member.Status == domain.GroupMemberStatusActive
		member.Status = domain.GroupMemberStatusBanned
		if updateErr := s.groupMemberRepo.Update(ctx, member); updateErr != nil {
			return updateErr
		}
		if wasActive {
			return s.groupRepo.DecrementMemberCount(ctx, groupID)
		}
		return nil
	})
}

func (s *socialServiceImpl) InviteToGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	actorMember, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, auth.FamilyID)
	if err != nil {
		return err
	}
	if actorMember == nil || !domain.CanModerate(actorMember.Role) {
		return &SocialError{Err: domain.ErrInsufficientGroupRole}
	}

	existing, _ := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, familyID)
	if existing != nil && existing.Status == domain.GroupMemberStatusActive {
		return &SocialError{Err: domain.ErrAlreadyGroupMember}
	}

	return s.groupMemberRepo.Create(ctx, &GroupMember{
		GroupID:  groupID,
		FamilyID: familyID,
		Role:    domain.GroupRoleMember,
		Status:  domain.GroupMemberStatusInvited,
	})
}

func (s *socialServiceImpl) PromoteMember(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, familyID uuid.UUID) error {
	return s.moderateGroupAction(ctx, auth, groupID, familyID, func(member *GroupMember) error {
		if promoteErr := domain.ValidatePromotion(member.Role, member.Status); promoteErr != nil {
			return &SocialError{Err: promoteErr}
		}
		member.Role = domain.GroupRoleModerator
		return s.groupMemberRepo.Update(ctx, member)
	})
}

// ─── Pinned Posts ────────────────────────────────────────────────────────────

func (s *socialServiceImpl) PinPost(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, postID uuid.UUID) error {
	// Verify caller is moderator/owner of the group. [05-social §4.2]
	member, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, auth.FamilyID)
	if err != nil {
		return err
	}
	if !domain.CanModerate(member.Role) {
		return &SocialError{Err: domain.ErrInsufficientGroupRole}
	}
	// Verify post belongs to the group.
	post, err := s.postRepo.FindByID(ctx, postID)
	if err != nil {
		return err
	}
	if post.GroupID == nil || *post.GroupID != groupID {
		return &SocialError{Err: domain.ErrPostNotInGroup}
	}
	return s.pinnedPostRepo.Create(ctx, &PinnedPost{
		GroupID:  groupID,
		PostID:   postID,
		PinnedBy: auth.FamilyID,
	})
}

func (s *socialServiceImpl) UnpinPost(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, postID uuid.UUID) error {
	// Verify caller is moderator/owner of the group. [05-social §4.2]
	member, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, auth.FamilyID)
	if err != nil {
		return err
	}
	if !domain.CanModerate(member.Role) {
		return &SocialError{Err: domain.ErrInsufficientGroupRole}
	}
	return s.pinnedPostRepo.Delete(ctx, groupID, postID)
}

func (s *socialServiceImpl) GetGroup(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) (*GroupResponse, error) {
	group, err := s.groupRepo.FindByID(ctx, groupID)
	if err != nil {
		return nil, err
	}
	member, _ := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, auth.FamilyID)
	isMember := member != nil && member.Status == domain.GroupMemberStatusActive
	var myRole *string
	if member != nil {
		myRole = &member.Role
	}
	var myStatus *string
	if member != nil {
		myStatus = &member.Status
	}
	return &GroupResponse{
		Summary: GroupSummaryResponse{
			ID:              group.ID,
			GroupType:       group.GroupType,
			Name:            group.Name,
			Description:     group.Description,
			CoverPhotoURL:   group.CoverPhotoURL,
			MethodologyName: s.methodologyDisplayName(ctx, group.MethodologySlug),
			JoinPolicy:      group.JoinPolicy,
			MemberCount:     group.MemberCount,
			IsMember:        isMember,
		},
		MyRole:    myRole,
		MyStatus:  myStatus,
		CreatedAt: group.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) ListMyGroups(ctx context.Context, scope *shared.FamilyScope) ([]GroupResponse, error) {
	groupIDs, err := s.groupMemberRepo.ListGroupsByFamily(ctx, scope.FamilyID())
	if err != nil {
		return nil, err
	}
	results := make([]GroupResponse, 0, len(groupIDs))
	for _, gid := range groupIDs {
		group, findErr := s.groupRepo.FindByID(ctx, gid)
		if findErr != nil {
			continue
		}
		results = append(results, GroupResponse{
			Summary: GroupSummaryResponse{
				ID:              group.ID,
				GroupType:       group.GroupType,
				Name:            group.Name,
				Description:     group.Description,
				MethodologyName: s.methodologyDisplayName(ctx, group.MethodologySlug),
				JoinPolicy:      group.JoinPolicy,
				MemberCount:     group.MemberCount,
				IsMember:        true,
			},
			CreatedAt: group.CreatedAt,
		})
	}
	return results, nil
}

func (s *socialServiceImpl) ListPlatformGroups(ctx context.Context) ([]GroupResponse, error) {
	groups, err := s.groupRepo.ListPlatform(ctx)
	if err != nil {
		return nil, err
	}
	results := make([]GroupResponse, len(groups))
	for i, g := range groups {
		results[i] = GroupResponse{
			Summary: GroupSummaryResponse{
				ID:              g.ID,
				GroupType:       g.GroupType,
				Name:            g.Name,
				Description:     g.Description,
				MethodologyName: s.methodologyDisplayName(ctx, g.MethodologySlug),
				JoinPolicy:      g.JoinPolicy,
				MemberCount:     g.MemberCount,
			},
			CreatedAt: g.CreatedAt,
		}
	}
	return results, nil
}

func (s *socialServiceImpl) ListGroupMembers(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID) ([]GroupMemberResponse, error) {
	isMember, err := s.groupMemberRepo.IsMember(ctx, groupID, auth.FamilyID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, &SocialError{Err: domain.ErrNotGroupMember}
	}

	members, err := s.groupMemberRepo.ListByGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	results := make([]GroupMemberResponse, len(members))
	for i, m := range members {
		name, _ := s.iam.GetFamilyDisplayName(ctx, m.FamilyID)
		results[i] = GroupMemberResponse{
			FamilyID:    m.FamilyID,
			DisplayName: name,
			Role:        m.Role,
			Status:      m.Status,
			JoinedAt:    m.JoinedAt,
		}
	}
	return results, nil
}

func (s *socialServiceImpl) ListGroupPosts(ctx context.Context, auth *shared.AuthContext, groupID uuid.UUID, offset, limit int) ([]PostResponse, error) {
	isMember, err := s.groupMemberRepo.IsMember(ctx, groupID, auth.FamilyID)
	if err != nil {
		return nil, err
	}
	if !isMember {
		return nil, &SocialError{Err: domain.ErrNotGroupMember}
	}

	posts, err := s.postRepo.ListByGroup(ctx, groupID, offset, limit)
	if err != nil {
		return nil, err
	}
	results := make([]PostResponse, len(posts))
	for i, p := range posts {
		authorName, _ := s.iam.GetParentDisplayName(ctx, p.AuthorParentID)
		liked, _ := s.likeRepo.Exists(ctx, p.ID, auth.FamilyID)
		results[i] = PostResponse{
			ID:            p.ID,
			FamilyID:      p.FamilyID,
			AuthorName:    authorName,
			PostType:      p.PostType,
			Content:       p.Content,
			Attachments:   p.Attachments,
			GroupID:       p.GroupID,
			Visibility:    p.Visibility,
			LikesCount:    p.LikesCount,
			CommentsCount: p.CommentsCount,
			IsEdited:      p.IsEdited,
			IsLikedByMe:   liked,
			CreatedAt:     p.CreatedAt,
		}
	}
	return results, nil
}

// ─── Events ─────────────────────────────────────────────────────────────────

func (s *socialServiceImpl) CreateEvent(ctx context.Context, auth *shared.AuthContext, cmd CreateEventCommand) (*EventDetailResponse, error) {
	// Validate event date is in the future. [05-social §4.1] (C6)
	if valErr := domain.ValidateEventDate(cmd.EventDate); valErr != nil {
		return nil, &SocialError{Err: valErr}
	}
	// Validate group visibility requires group_id. [05-social §4.1] (C7)
	if valErr := domain.ValidateEventGroupVisibility(cmd.Visibility, cmd.GroupID); valErr != nil {
		return nil, &SocialError{Err: valErr}
	}

	// Convert capacity from *int to *int32. (L1)
	var capacity *int32
	if cmd.Capacity != nil {
		c := int32(*cmd.Capacity)
		capacity = &c
	}

	event := &Event{
		CreatorFamilyID: auth.FamilyID,
		CreatorParentID: auth.ParentID,
		GroupID:         cmd.GroupID,
		Title:           cmd.Title,
		Description:     cmd.Description,
		EventDate:       cmd.EventDate,
		EndDate:         cmd.EndDate,
		LocationName:    cmd.LocationName,
		LocationRegion:  cmd.LocationRegion,
		IsVirtual:       cmd.IsVirtual,
		VirtualURL:      cmd.VirtualURL,
		Capacity:        capacity,
		Visibility:      cmd.Visibility,
		MethodologySlug: cmd.MethodologySlug,
	}

	// CROSS-FAMILY: events may reference groups, need bypass.
	err := shared.BypassRLSTransaction(ctx, s.db, func(tx *gorm.DB) error {
		repo := &PgEventRepository{db: tx}
		return repo.Create(ctx, event)
	})
	if err != nil {
		return nil, err
	}

	creatorName, _ := s.iam.GetFamilyDisplayName(ctx, auth.FamilyID)
	return &EventDetailResponse{
		EventSummaryResponse: EventSummaryResponse{
			ID:                event.ID,
			Title:             event.Title,
			EventDate:         event.EventDate,
			EndDate:           event.EndDate,
			LocationName:      event.LocationName,
			LocationRegion:    event.LocationRegion,
			IsVirtual:         event.IsVirtual,
			CreatorFamilyName: creatorName,
			Capacity:          event.Capacity,
			Visibility:        event.Visibility,
			Status:            event.Status,
			AttendeeCount:     0,
		},
		CreatorFamilyID: event.CreatorFamilyID,
		GroupID:         event.GroupID,
		Description:     event.Description,
		VirtualURL:      event.VirtualURL,
		MethodologyName: s.methodologyDisplayName(ctx, event.MethodologySlug),
		CreatedAt:       event.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) UpdateEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID, cmd UpdateEventCommand) (*EventDetailResponse, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}
	if event.CreatorFamilyID != auth.FamilyID {
		return nil, &SocialError{Err: domain.ErrCannotModifyEvent}
	}

	if cmd.Title != nil {
		event.Title = *cmd.Title
	}
	if cmd.Description != nil {
		event.Description = cmd.Description
	}
	if cmd.EventDate != nil {
		event.EventDate = *cmd.EventDate
	}
	if cmd.EndDate != nil {
		event.EndDate = cmd.EndDate
	}
	if cmd.LocationName != nil {
		event.LocationName = cmd.LocationName
	}
	if cmd.LocationRegion != nil {
		event.LocationRegion = cmd.LocationRegion
	}
	if cmd.IsVirtual != nil {
		event.IsVirtual = *cmd.IsVirtual
	}
	if cmd.VirtualURL != nil {
		event.VirtualURL = cmd.VirtualURL
	}
	if cmd.Capacity != nil {
		c := int32(*cmd.Capacity)
		event.Capacity = &c
	}
	if cmd.Visibility != nil {
		event.Visibility = *cmd.Visibility
	}

	if updateErr := s.eventRepo.Update(ctx, event); updateErr != nil {
		return nil, updateErr
	}

	creatorName, _ := s.iam.GetFamilyDisplayName(ctx, auth.FamilyID)
	return &EventDetailResponse{
		EventSummaryResponse: EventSummaryResponse{
			ID:                event.ID,
			Title:             event.Title,
			EventDate:         event.EventDate,
			EndDate:           event.EndDate,
			LocationName:      event.LocationName,
			LocationRegion:    event.LocationRegion,
			IsVirtual:         event.IsVirtual,
			CreatorFamilyName: creatorName,
			Capacity:          event.Capacity,
			Visibility:        event.Visibility,
			Status:            event.Status,
			AttendeeCount:     event.AttendeeCount,
		},
		CreatorFamilyID: event.CreatorFamilyID,
		GroupID:         event.GroupID,
		Description:     event.Description,
		VirtualURL:      event.VirtualURL,
		MethodologyName: s.methodologyDisplayName(ctx, event.MethodologySlug),
		CreatedAt:       event.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) CancelEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) error {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.CreatorFamilyID != auth.FamilyID {
		return &SocialError{Err: domain.ErrCannotModifyEvent}
	}
	event.Status = "cancelled"
	if updateErr := s.eventRepo.Update(ctx, event); updateErr != nil {
		return updateErr
	}

	goingIDs, _ := s.rsvpRepo.ListGoingFamilyIDs(ctx, eventID)
	_ = s.eventBus.Publish(ctx, EventCancelled{
		EventID:         event.ID,
		CreatorFamilyID: event.CreatorFamilyID,
		Title:           event.Title,
		EventDate:       event.EventDate,
		GoingFamilyIDs:  goingIDs,
	})
	return nil
}

func (s *socialServiceImpl) RSVPEvent(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID, cmd RSVPCommand) error {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return err
	}
	if event.Status == "cancelled" {
		return &SocialError{Err: domain.ErrEventCancelled}
	}

	existing, _ := s.rsvpRepo.FindByEventAndFamily(ctx, eventID, scope.FamilyID())
	if existing != nil {
		// Update existing RSVP.
		wasGoing := existing.Status == "going"
		existing.Status = cmd.Status
		if updateErr := s.rsvpRepo.Update(ctx, existing); updateErr != nil {
			return updateErr
		}
		// Adjust attendee count.
		if wasGoing && cmd.Status != "going" {
			_ = s.eventRepo.DecrementAttendeeCount(ctx, eventID)
		} else if !wasGoing && cmd.Status == "going" {
			if event.Capacity != nil && event.AttendeeCount >= int(*event.Capacity) {
				return &SocialError{Err: domain.ErrEventAtCapacity}
			}
			_ = s.eventRepo.IncrementAttendeeCount(ctx, eventID)
		}
		return nil
	}

	// New RSVP.
	if cmd.Status == "going" && event.Capacity != nil && event.AttendeeCount >= int(*event.Capacity) {
		return &SocialError{Err: domain.ErrEventAtCapacity}
	}

	rsvp := &EventRSVP{
		EventID:  eventID,
		FamilyID: scope.FamilyID(),
		Status:   cmd.Status,
	}
	if createErr := s.rsvpRepo.Create(ctx, rsvp); createErr != nil {
		return createErr
	}
	if cmd.Status == "going" {
		_ = s.eventRepo.IncrementAttendeeCount(ctx, eventID)
	}
	return nil
}

func (s *socialServiceImpl) RemoveRSVP(ctx context.Context, scope *shared.FamilyScope, eventID uuid.UUID) error {
	existing, err := s.rsvpRepo.FindByEventAndFamily(ctx, eventID, scope.FamilyID())
	if err != nil {
		return err
	}
	if existing == nil {
		return &SocialError{Err: domain.ErrRSVPNotFound}
	}
	wasGoing := existing.Status == "going"
	if delErr := s.rsvpRepo.Delete(ctx, eventID, scope.FamilyID()); delErr != nil {
		return delErr
	}
	if wasGoing {
		_ = s.eventRepo.DecrementAttendeeCount(ctx, eventID)
	}
	return nil
}

func (s *socialServiceImpl) GetEvent(ctx context.Context, auth *shared.AuthContext, eventID uuid.UUID) (*EventDetailResponse, error) {
	event, err := s.eventRepo.FindByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	// Block check. [05-social §16]
	if event.CreatorFamilyID != auth.FamilyID {
		blocked, blockErr := s.blockRepo.IsEitherBlocked(ctx, auth.FamilyID, event.CreatorFamilyID)
		if blockErr != nil {
			return nil, blockErr
		}
		if blocked {
			return nil, &SocialError{Err: domain.ErrContentNotVisible}
		}
	}

	areFriends, _ := s.friendshipRepo.AreFriends(ctx, auth.FamilyID, event.CreatorFamilyID)
	isGroupMember := false
	if event.GroupID != nil {
		isGroupMember, _ = s.groupMemberRepo.IsMember(ctx, *event.GroupID, auth.FamilyID)
	}
	if !domain.CanViewEvent(auth.FamilyID, event.CreatorFamilyID, event.Visibility, areFriends, isGroupMember) {
		return nil, &SocialError{Err: domain.ErrContentNotVisible}
	}

	creatorName, _ := s.iam.GetFamilyDisplayName(ctx, event.CreatorFamilyID)
	rsvp, _ := s.rsvpRepo.FindByEventAndFamily(ctx, eventID, auth.FamilyID)
	var myRSVP *string
	if rsvp != nil {
		myRSVP = &rsvp.Status
	}

	// Build RSVPs list for detail response. (H11)
	rsvps, _ := s.rsvpRepo.ListByEvent(ctx, eventID)
	rsvpResponses := make([]EventRsvpResponse, 0, len(rsvps))
	for _, r := range rsvps {
		name, _ := s.iam.GetFamilyDisplayName(ctx, r.FamilyID)
		rsvpResponses = append(rsvpResponses, EventRsvpResponse{
			FamilyID:    r.FamilyID,
			DisplayName: name,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
		})
	}

	return &EventDetailResponse{
		EventSummaryResponse: EventSummaryResponse{
			ID:                event.ID,
			Title:             event.Title,
			EventDate:         event.EventDate,
			EndDate:           event.EndDate,
			LocationName:      event.LocationName,
			LocationRegion:    event.LocationRegion,
			IsVirtual:         event.IsVirtual,
			CreatorFamilyName: creatorName,
			Capacity:          event.Capacity,
			Visibility:        event.Visibility,
			Status:            event.Status,
			AttendeeCount:     event.AttendeeCount,
			MyRSVP:            myRSVP,
		},
		CreatorFamilyID: event.CreatorFamilyID,
		GroupID:         event.GroupID,
		Description:     event.Description,
		VirtualURL:      event.VirtualURL,
		MethodologyName: s.methodologyDisplayName(ctx, event.MethodologySlug),
		Rsvps:           rsvpResponses,
		CreatedAt:       event.CreatedAt,
	}, nil
}

func (s *socialServiceImpl) ListEvents(ctx context.Context, auth *shared.AuthContext, offset, limit int) ([]EventDetailResponse, error) {
	// Get all friend IDs for visibility filtering (unbounded). (H13)
	friendIDs, err := s.friendshipRepo.ListFriendFamilyIDs(ctx, auth.FamilyID)
	if err != nil {
		return nil, err
	}
	groupIDs, err := s.groupMemberRepo.ListGroupsByFamily(ctx, auth.FamilyID)
	if err != nil {
		return nil, err
	}

	events, err := s.eventRepo.ListVisible(ctx, auth.FamilyID, friendIDs, groupIDs, offset, limit)
	if err != nil {
		return nil, err
	}

	results := make([]EventDetailResponse, 0, len(events))
	for _, e := range events {
		// Block filter.
		if e.CreatorFamilyID != auth.FamilyID {
			blocked, _ := s.blockRepo.IsEitherBlocked(ctx, auth.FamilyID, e.CreatorFamilyID)
			if blocked {
				continue
			}
		}

		creatorName, _ := s.iam.GetFamilyDisplayName(ctx, e.CreatorFamilyID)
		rsvp, _ := s.rsvpRepo.FindByEventAndFamily(ctx, e.ID, auth.FamilyID)
		var myRSVP *string
		if rsvp != nil {
			myRSVP = &rsvp.Status
		}

		results = append(results, EventDetailResponse{
			EventSummaryResponse: EventSummaryResponse{
				ID:                e.ID,
				Title:             e.Title,
				EventDate:         e.EventDate,
				EndDate:           e.EndDate,
				LocationName:      e.LocationName,
				LocationRegion:    e.LocationRegion,
				IsVirtual:         e.IsVirtual,
				CreatorFamilyName: creatorName,
				Capacity:          e.Capacity,
				Visibility:        e.Visibility,
				Status:            e.Status,
				AttendeeCount:     e.AttendeeCount,
				MyRSVP:            myRSVP,
			},
			CreatorFamilyID: e.CreatorFamilyID,
			GroupID:         e.GroupID,
			Description:     e.Description,
			VirtualURL:      e.VirtualURL,
			MethodologyName: s.methodologyDisplayName(ctx, e.MethodologySlug),
			CreatedAt:       e.CreatedAt,
		})
	}
	return results, nil
}

// ─── Discovery (Phase 2) ────────────────────────────────────────────────────

// DiscoverFamilies returns families with location_visible=true. [05-social §15]
// Phase 2: PostGIS radius queries not yet wired; returns non-blocked location-visible families
// filtered by methodology if provided.
func (s *socialServiceImpl) DiscoverFamilies(_ context.Context, _ *shared.FamilyScope, _ DiscoverFamiliesQuery) ([]DiscoverableFamilyResponse, error) {
	// Phase 2 stub: PostGIS proximity queries require iam:: location_point support.
	return []DiscoverableFamilyResponse{}, nil
}

// DiscoverEvents returns discoverable events filtered by methodology/location. [05-social §15]
func (s *socialServiceImpl) DiscoverEvents(ctx context.Context, scope *shared.FamilyScope, query DiscoverEventsQuery) ([]EventSummaryResponse, error) {
	events, err := s.eventRepo.ListDiscoverable(ctx, query.MethodologySlug, query.LocationRegion)
	if err != nil {
		return nil, err
	}
	results := make([]EventSummaryResponse, 0, len(events))
	for _, e := range events {
		// Block filter: silently skip events from blocked families. [05-social §9.2]
		if e.CreatorFamilyID != scope.FamilyID() {
			if blocked, _ := s.blockRepo.IsEitherBlocked(ctx, scope.FamilyID(), e.CreatorFamilyID); blocked {
				continue
			}
		}
		creatorName, _ := s.iam.GetFamilyDisplayName(ctx, e.CreatorFamilyID)
		var myRsvp *string
		if rsvp, rErr := s.rsvpRepo.FindByEventAndFamily(ctx, e.ID, scope.FamilyID()); rErr == nil && rsvp != nil {
			myRsvp = &rsvp.Status
		}
		results = append(results, EventSummaryResponse{
			ID:                e.ID,
			Title:             e.Title,
			EventDate:         e.EventDate,
			EndDate:           e.EndDate,
			LocationName:      e.LocationName,
			LocationRegion:    e.LocationRegion,
			IsVirtual:         e.IsVirtual,
			CreatorFamilyName: creatorName,
			Capacity:          e.Capacity,
			Visibility:        e.Visibility,
			Status:            e.Status,
			AttendeeCount:     e.AttendeeCount,
			MyRSVP:            myRsvp,
		})
	}
	return results, nil
}

// DiscoverGroups returns groups tagged with a methodology slug. [05-social §15]
func (s *socialServiceImpl) DiscoverGroups(ctx context.Context, scope *shared.FamilyScope, query DiscoverGroupsQuery) ([]GroupSummaryResponse, error) {
	if query.MethodologySlug == nil || *query.MethodologySlug == "" {
		return []GroupSummaryResponse{}, nil
	}
	groups, err := s.groupRepo.ListByMethodology(ctx, *query.MethodologySlug)
	if err != nil {
		return nil, err
	}
	results := make([]GroupSummaryResponse, 0, len(groups))
	for _, g := range groups {
		isMember, _ := s.groupMemberRepo.IsMember(ctx, g.ID, scope.FamilyID())
		results = append(results, GroupSummaryResponse{
			ID:          g.ID,
			GroupType:   g.GroupType,
			Name:        g.Name,
			Description: g.Description,
			JoinPolicy:  g.JoinPolicy,
			MemberCount: g.MemberCount,
			IsMember:    isMember,
			MethodologyName: s.methodologyDisplayName(ctx, g.MethodologySlug),
		})
	}
	return results, nil
}

// ─── Event Handlers ─────────────────────────────────────────────────────────

func (s *socialServiceImpl) HandleFamilyCreated(ctx context.Context, familyID uuid.UUID) error {
	return s.CreateProfile(ctx, familyID)
}

// Event handler stubs — full implementation deferred to M3. [05-social §5]
func (s *socialServiceImpl) HandleCoParentAdded(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (s *socialServiceImpl) HandleCoParentRemoved(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
	return nil
}

func (s *socialServiceImpl) HandleMilestoneAchieved(_ context.Context, _ uuid.UUID, _ MilestoneData) error {
	return nil
}

func (s *socialServiceImpl) HandleFamilyDeletionScheduled(_ context.Context, _ uuid.UUID) error {
	return nil
}

// ─── Helpers ────────────────────────────────────────────────────────────────

func (s *socialServiceImpl) moderateGroupAction(ctx context.Context, auth *shared.AuthContext, groupID, familyID uuid.UUID, action func(*GroupMember) error) error {
	actorMember, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, auth.FamilyID)
	if err != nil {
		return err
	}
	if actorMember == nil || !domain.CanModerate(actorMember.Role) {
		return &SocialError{Err: domain.ErrInsufficientGroupRole}
	}

	targetMember, err := s.groupMemberRepo.FindByGroupAndFamily(ctx, groupID, familyID)
	if err != nil {
		return err
	}
	if targetMember == nil {
		return &SocialError{Err: domain.ErrGroupMemberNotFound}
	}
	return action(targetMember)
}

func buildProfileResponse(profile *Profile, info *SocialFamilyInfo, isOwn, isFriend bool) *ProfileResponse {
	// Unmarshal per-field privacy settings. Fall back to defaults if missing/corrupt.
	var ps domain.PrivacySettings
	if err := json.Unmarshal(profile.PrivacySettings, &ps); err != nil {
		ps = domain.DefaultPrivacySettings()
	}
	vis := domain.FilterProfileFields(isOwn, isFriend, ps)

	resp := &ProfileResponse{
		FamilyID:        profile.FamilyID,
		Bio:             profile.Bio,
		ProfilePhotoURL: profile.ProfilePhotoURL,
		IsFriend:        isFriend,
	}

	// Own profile sees privacy settings and LocationVisible. (M2)
	if isOwn {
		resp.PrivacySettings = &profile.PrivacySettings
		resp.LocationVisible = &profile.LocationVisible
	}

	// Gate per-field data through privacy visibility map. [05-social §9.3]
	if info != nil && vis["display_name"] {
		resp.DisplayName = &info.DisplayName
	}
	if info != nil && vis["parent_names"] {
		resp.ParentNames = info.ParentNames
	}
	// Phase 2: LocationRegion, MethodologyNames, Children gated by
	// vis["location"], vis["methodology"], vis["children_names"]/vis["children_ages"].

	return resp
}

