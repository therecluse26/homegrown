package social

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
	"github.com/homegrown-academy/homegrown-academy/internal/social/domain"
	"gorm.io/gorm"
)

// ─── Profile Repository ─────────────────────────────────────────────────────

type PgProfileRepository struct {
	db *gorm.DB
}

func NewPgProfileRepository(db *gorm.DB) ProfileRepository {
	return &PgProfileRepository{db: db}
}

func (r *PgProfileRepository) Create(ctx context.Context, profile *Profile) error {
	if err := r.db.WithContext(ctx).Create(profile).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgProfileRepository) FindByFamilyID(ctx context.Context, familyID uuid.UUID) (*Profile, error) {
	var profile Profile
	err := r.db.WithContext(ctx).Where("family_id = ?", familyID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrProfileNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &profile, nil
}

func (r *PgProfileRepository) Update(ctx context.Context, profile *Profile) error {
	if err := r.db.WithContext(ctx).Save(profile).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Friendship Repository ──────────────────────────────────────────────────

type PgFriendshipRepository struct {
	db *gorm.DB
}

func NewPgFriendshipRepository(db *gorm.DB) FriendshipRepository {
	return &PgFriendshipRepository{db: db}
}

func (r *PgFriendshipRepository) Create(ctx context.Context, friendship *Friendship) error {
	if err := r.db.WithContext(ctx).Create(friendship).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// CROSS-FAMILY: checks both directions for an existing friendship.
func (r *PgFriendshipRepository) FindBetween(ctx context.Context, familyA, familyB uuid.UUID) (*Friendship, error) {
	var friendship Friendship
	err := r.db.WithContext(ctx).
		Where("(requester_family_id = ? AND accepter_family_id = ?) OR (requester_family_id = ? AND accepter_family_id = ?)",
			familyA, familyB, familyB, familyA).
		First(&friendship).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No friendship exists — not an error
		}
		return nil, shared.ErrDatabase(err)
	}
	return &friendship, nil
}

func (r *PgFriendshipRepository) FindByID(ctx context.Context, id uuid.UUID) (*Friendship, error) {
	var friendship Friendship
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&friendship).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrFriendRequestNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &friendship, nil
}

func (r *PgFriendshipRepository) Update(ctx context.Context, friendship *Friendship) error {
	if err := r.db.WithContext(ctx).Save(friendship).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgFriendshipRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&Friendship{}, "id = ?", id).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// CROSS-FAMILY: lists accepted friend family IDs (paginated).
func (r *PgFriendshipRepository) ListFriends(ctx context.Context, familyID uuid.UUID, offset, limit int) ([]uuid.UUID, error) {
	var friendships []Friendship
	err := r.db.WithContext(ctx).
		Where("(requester_family_id = ? OR accepter_family_id = ?) AND status = 'accepted'",
			familyID, familyID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&friendships).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	ids := make([]uuid.UUID, 0, len(friendships))
	for _, f := range friendships {
		if f.RequesterFamilyID == familyID {
			ids = append(ids, f.AccepterFamilyID)
		} else {
			ids = append(ids, f.RequesterFamilyID)
		}
	}
	return ids, nil
}

// CROSS-FAMILY: lists incoming pending requests.
func (r *PgFriendshipRepository) ListIncoming(ctx context.Context, familyID uuid.UUID) ([]Friendship, error) {
	var friendships []Friendship
	err := r.db.WithContext(ctx).
		Where("accepter_family_id = ? AND status = 'pending'", familyID).
		Order("created_at DESC").
		Find(&friendships).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return friendships, nil
}

// CROSS-FAMILY: lists outgoing pending requests.
func (r *PgFriendshipRepository) ListOutgoing(ctx context.Context, familyID uuid.UUID) ([]Friendship, error) {
	var friendships []Friendship
	err := r.db.WithContext(ctx).
		Where("requester_family_id = ? AND status = 'pending'", familyID).
		Order("created_at DESC").
		Find(&friendships).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return friendships, nil
}

// CROSS-FAMILY: checks if two families are accepted friends.
func (r *PgFriendshipRepository) AreFriends(ctx context.Context, familyA, familyB uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Friendship{}).
		Where("((requester_family_id = ? AND accepter_family_id = ?) OR (requester_family_id = ? AND accepter_family_id = ?)) AND status = 'accepted'",
			familyA, familyB, familyB, familyA).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

// CROSS-FAMILY: lists ALL accepted friend family IDs (unbounded, for feed fan-out). [05-social §6]
func (r *PgFriendshipRepository) ListFriendFamilyIDs(ctx context.Context, familyID uuid.UUID) ([]uuid.UUID, error) {
	var friendships []Friendship
	err := r.db.WithContext(ctx).
		Where("(requester_family_id = ? OR accepter_family_id = ?) AND status = 'accepted'",
			familyID, familyID).
		Find(&friendships).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	ids := make([]uuid.UUID, 0, len(friendships))
	for _, f := range friendships {
		if f.RequesterFamilyID == familyID {
			ids = append(ids, f.AccepterFamilyID)
		} else {
			ids = append(ids, f.RequesterFamilyID)
		}
	}
	return ids, nil
}

// CROSS-FAMILY: lists friends with cursor-based pagination (cursor = last friendship UUID).
func (r *PgFriendshipRepository) ListFriendsCursor(ctx context.Context, familyID uuid.UUID, cursor *uuid.UUID, limit int) ([]Friendship, error) {
	query := r.db.WithContext(ctx).
		Where("(requester_family_id = ? OR accepter_family_id = ?) AND status = 'accepted'",
			familyID, familyID).
		Order("id ASC")
	if cursor != nil {
		query = query.Where("id > ?", *cursor)
	}
	var friendships []Friendship
	err := query.Limit(limit).Find(&friendships).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return friendships, nil
}

func (r *PgFriendshipRepository) DeleteBetween(ctx context.Context, familyA, familyB uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Where("(requester_family_id = ? AND accepter_family_id = ?) OR (requester_family_id = ? AND accepter_family_id = ?)",
			familyA, familyB, familyB, familyA).
		Delete(&Friendship{}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Block Repository ───────────────────────────────────────────────────────

type PgBlockRepository struct {
	db *gorm.DB
}

func NewPgBlockRepository(db *gorm.DB) BlockRepository {
	return &PgBlockRepository{db: db}
}

func (r *PgBlockRepository) Create(ctx context.Context, block *Block) error {
	if err := r.db.WithContext(ctx).Create(block).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// CROSS-FAMILY: bidirectional block check.
func (r *PgBlockRepository) IsEitherBlocked(ctx context.Context, familyA, familyB uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Block{}).
		Where("(blocker_family_id = ? AND blocked_family_id = ?) OR (blocker_family_id = ? AND blocked_family_id = ?)",
			familyA, familyB, familyB, familyA).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

func (r *PgBlockRepository) IsBlocked(ctx context.Context, blockerID, blockedID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&Block{}).
		Where("blocker_family_id = ? AND blocked_family_id = ?", blockerID, blockedID).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

func (r *PgBlockRepository) Delete(ctx context.Context, blockerID, blockedID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Where("blocker_family_id = ? AND blocked_family_id = ?", blockerID, blockedID).
		Delete(&Block{}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgBlockRepository) ListByBlocker(ctx context.Context, blockerID uuid.UUID) ([]Block, error) {
	var blocks []Block
	err := r.db.WithContext(ctx).
		Where("blocker_family_id = ?", blockerID).
		Order("created_at DESC").
		Find(&blocks).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return blocks, nil
}

// ─── Post Repository ────────────────────────────────────────────────────────

type PgPostRepository struct {
	db *gorm.DB
}

func NewPgPostRepository(db *gorm.DB) PostRepository {
	return &PgPostRepository{db: db}
}

func (r *PgPostRepository) Create(ctx context.Context, post *Post) error {
	if err := r.db.WithContext(ctx).Create(post).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPostRepository) FindByID(ctx context.Context, id uuid.UUID) (*Post, error) {
	var post Post
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&post).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrPostNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &post, nil
}

func (r *PgPostRepository) Update(ctx context.Context, post *Post) error {
	if err := r.db.WithContext(ctx).Save(post).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPostRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&Post{}, "id = ?", id).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// CROSS-FAMILY: PostgreSQL fallback feed query.
func (r *PgPostRepository) ListByFamilyIDs(ctx context.Context, familyIDs []uuid.UUID, offset, limit int) ([]Post, error) {
	if len(familyIDs) == 0 {
		return nil, nil
	}
	var posts []Post
	err := r.db.WithContext(ctx).
		Where("family_id IN ? AND visibility = 'friends'", familyIDs).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return posts, nil
}

func (r *PgPostRepository) FindByIDs(ctx context.Context, ids []uuid.UUID) ([]Post, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var posts []Post
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&posts).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return posts, nil
}

// ListFriendsPosts returns recent posts from friends for feed fallback. [05-social §6]
// Includes the user's own posts and friends' posts with 'friends' visibility.
func (r *PgPostRepository) ListFriendsPosts(ctx context.Context, familyID uuid.UUID, friendIDs []uuid.UUID, offset, limit int) ([]Post, error) {
	allIDs := append([]uuid.UUID{familyID}, friendIDs...)
	if len(allIDs) == 0 {
		return nil, nil
	}
	var posts []Post
	err := r.db.WithContext(ctx).
		Where("family_id IN ? AND visibility = 'friends'", allIDs).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return posts, nil
}

func (r *PgPostRepository) ListByGroup(ctx context.Context, groupID uuid.UUID, offset, limit int) ([]Post, error) {
	var posts []Post
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&posts).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return posts, nil
}

func (r *PgPostRepository) IncrementLikes(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Post{}).Where("id = ?", id).
		Update("likes_count", gorm.Expr("likes_count + 1")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPostRepository) DecrementLikes(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Post{}).Where("id = ?", id).
		Update("likes_count", gorm.Expr("GREATEST(likes_count - 1, 0)")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPostRepository) IncrementComments(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Post{}).Where("id = ?", id).
		Update("comments_count", gorm.Expr("comments_count + 1")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPostRepository) DecrementComments(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Post{}).Where("id = ?", id).
		Update("comments_count", gorm.Expr("GREATEST(comments_count - 1, 0)")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Pinned Post Repository ──────────────────────────────────────────────────

type PgPinnedPostRepository struct {
	db *gorm.DB
}

func NewPgPinnedPostRepository(db *gorm.DB) PinnedPostRepository {
	return &PgPinnedPostRepository{db: db}
}

func (r *PgPinnedPostRepository) Create(ctx context.Context, pin *PinnedPost) error {
	if err := r.db.WithContext(ctx).Create(pin).Error; err != nil {
		if strings.Contains(err.Error(), "uq_soc_pinned_posts_group_post") {
			return &SocialError{Err: domain.ErrPostAlreadyPinned}
		}
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPinnedPostRepository) Delete(ctx context.Context, groupID, postID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("group_id = ? AND post_id = ?", groupID, postID).
		Delete(&PinnedPost{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return &SocialError{Err: domain.ErrPinnedPostNotFound}
	}
	return nil
}

func (r *PgPinnedPostRepository) FindByGroupAndPost(ctx context.Context, groupID, postID uuid.UUID) (*PinnedPost, error) {
	var pin PinnedPost
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND post_id = ?", groupID, postID).
		First(&pin).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrPinnedPostNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &pin, nil
}

func (r *PgPinnedPostRepository) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]PinnedPost, error) {
	var pins []PinnedPost
	err := r.db.WithContext(ctx).
		Where("group_id = ?", groupID).
		Order("pinned_at DESC").
		Find(&pins).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return pins, nil
}

// ─── Comment Repository ─────────────────────────────────────────────────────

type PgCommentRepository struct {
	db *gorm.DB
}

func NewPgCommentRepository(db *gorm.DB) CommentRepository {
	return &PgCommentRepository{db: db}
}

func (r *PgCommentRepository) Create(ctx context.Context, comment *Comment) error {
	if err := r.db.WithContext(ctx).Create(comment).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgCommentRepository) FindByID(ctx context.Context, id uuid.UUID) (*Comment, error) {
	var comment Comment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&comment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrCommentNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &comment, nil
}

func (r *PgCommentRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&Comment{}, "id = ?", id).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgCommentRepository) ListByPost(ctx context.Context, postID uuid.UUID) ([]Comment, error) {
	var comments []Comment
	err := r.db.WithContext(ctx).
		Where("post_id = ?", postID).
		Order("created_at ASC").
		Find(&comments).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return comments, nil
}

// ─── PostLike Repository ────────────────────────────────────────────────────

type PgPostLikeRepository struct {
	db *gorm.DB
}

func NewPgPostLikeRepository(db *gorm.DB) PostLikeRepository {
	return &PgPostLikeRepository{db: db}
}

func (r *PgPostLikeRepository) Create(ctx context.Context, like *PostLike) error {
	if err := r.db.WithContext(ctx).Create(like).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgPostLikeRepository) Delete(ctx context.Context, postID, familyID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("post_id = ? AND family_id = ?", postID, familyID).
		Delete(&PostLike{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return &SocialError{Err: domain.ErrNotLiked}
	}
	return nil
}

func (r *PgPostLikeRepository) Exists(ctx context.Context, postID, familyID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&PostLike{}).
		Where("post_id = ? AND family_id = ?", postID, familyID).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

func (r *PgPostLikeRepository) ListByPostIDs(ctx context.Context, postIDs []uuid.UUID, familyID uuid.UUID) (map[uuid.UUID]bool, error) {
	if len(postIDs) == 0 {
		return make(map[uuid.UUID]bool), nil
	}
	var likes []PostLike
	err := r.db.WithContext(ctx).
		Where("post_id IN ? AND family_id = ?", postIDs, familyID).
		Find(&likes).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	result := make(map[uuid.UUID]bool, len(likes))
	for _, l := range likes {
		result[l.PostID] = true
	}
	return result, nil
}

// ─── Conversation Repository ────────────────────────────────────────────────

type PgConversationRepository struct {
	db *gorm.DB
}

func NewPgConversationRepository(db *gorm.DB) ConversationRepository {
	return &PgConversationRepository{db: db}
}

func (r *PgConversationRepository) Create(ctx context.Context, conv *Conversation) error {
	if err := r.db.WithContext(ctx).Create(conv).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgConversationRepository) FindByID(ctx context.Context, id uuid.UUID) (*Conversation, error) {
	var conv Conversation
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&conv).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrConversationNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &conv, nil
}

// CROSS-FAMILY: lists conversations for a parent (paginated).
func (r *PgConversationRepository) ListByParent(ctx context.Context, parentID uuid.UUID, offset, limit int) ([]Conversation, error) {
	var convs []Conversation
	err := r.db.WithContext(ctx).
		Joins("JOIN soc_conversation_participants cp ON cp.conversation_id = soc_conversations.id").
		Where("cp.parent_id = ? AND cp.deleted_at IS NULL", parentID).
		Order("soc_conversations.updated_at DESC").
		Offset(offset).Limit(limit).
		Find(&convs).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return convs, nil
}

// ─── ConversationParticipant Repository ─────────────────────────────────────

type PgConversationParticipantRepository struct {
	db *gorm.DB
}

func NewPgConversationParticipantRepository(db *gorm.DB) ConversationParticipantRepository {
	return &PgConversationParticipantRepository{db: db}
}

func (r *PgConversationParticipantRepository) Create(ctx context.Context, participant *ConversationParticipant) error {
	if err := r.db.WithContext(ctx).Create(participant).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// CROSS-FAMILY: lists participants for a conversation.
func (r *PgConversationParticipantRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID) ([]ConversationParticipant, error) {
	var participants []ConversationParticipant
	err := r.db.WithContext(ctx).
		Where("conversation_id = ? AND deleted_at IS NULL", conversationID).
		Find(&participants).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return participants, nil
}

func (r *PgConversationParticipantRepository) IsParticipant(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&ConversationParticipant{}).
		Where("conversation_id = ? AND parent_id = ? AND deleted_at IS NULL", conversationID, parentID).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

func (r *PgConversationParticipantRepository) UpdateLastRead(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&ConversationParticipant{}).
		Where("conversation_id = ? AND parent_id = ?", conversationID, parentID).
		Update("last_read_at", now).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// CROSS-FAMILY: finds an existing 1-on-1 conversation between two parents.
func (r *PgConversationParticipantRepository) FindBetweenParents(ctx context.Context, parentA, parentB uuid.UUID) (*uuid.UUID, error) {
	var convID uuid.UUID
	err := r.db.WithContext(ctx).
		Raw(`SELECT cp1.conversation_id
			FROM soc_conversation_participants cp1
			JOIN soc_conversation_participants cp2
			  ON cp1.conversation_id = cp2.conversation_id
			WHERE cp1.parent_id = ? AND cp2.parent_id = ?
			  AND cp1.deleted_at IS NULL AND cp2.deleted_at IS NULL
			LIMIT 1`, parentA, parentB).
		Scan(&convID).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	if convID == uuid.Nil {
		return nil, nil
	}
	return &convID, nil
}

// ClearDeletedAt restores a soft-deleted participant (new message restores conversation). [05-social §4.1]
func (r *PgConversationParticipantRepository) ClearDeletedAt(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&ConversationParticipant{}).
		Where("conversation_id = ? AND parent_id = ?", conversationID, parentID).
		Update("deleted_at", nil).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgConversationParticipantRepository) SoftDelete(ctx context.Context, conversationID uuid.UUID, parentID uuid.UUID) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&ConversationParticipant{}).
		Where("conversation_id = ? AND parent_id = ?", conversationID, parentID).
		Update("deleted_at", now).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── Message Repository ─────────────────────────────────────────────────────

type PgMessageRepository struct {
	db *gorm.DB
}

func NewPgMessageRepository(db *gorm.DB) MessageRepository {
	return &PgMessageRepository{db: db}
}

func (r *PgMessageRepository) Create(ctx context.Context, msg *Message) error {
	if err := r.db.WithContext(ctx).Create(msg).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgMessageRepository) FindByID(ctx context.Context, id uuid.UUID) (*Message, error) {
	var msg Message
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrMessageNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &msg, nil
}

// CROSS-FAMILY: lists messages in a conversation.
func (r *PgMessageRepository) ListByConversation(ctx context.Context, conversationID uuid.UUID, offset, limit int) ([]Message, error) {
	var msgs []Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&msgs).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return msgs, nil
}

func (r *PgMessageRepository) LastByConversation(ctx context.Context, conversationID uuid.UUID) (*Message, error) {
	var msg Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at DESC").
		First(&msg).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, shared.ErrDatabase(err)
	}
	return &msg, nil
}

func (r *PgMessageRepository) CountUnread(ctx context.Context, conversationID uuid.UUID, lastReadAt *time.Time) (int, error) {
	query := r.db.WithContext(ctx).Model(&Message{}).Where("conversation_id = ?", conversationID)
	if lastReadAt != nil {
		query = query.Where("created_at > ?", *lastReadAt)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return int(count), nil
}

// ─── Group Repository ───────────────────────────────────────────────────────

type PgGroupRepository struct {
	db *gorm.DB
}

func NewPgGroupRepository(db *gorm.DB) GroupRepository {
	return &PgGroupRepository{db: db}
}

func (r *PgGroupRepository) Create(ctx context.Context, group *Group) error {
	if err := r.db.WithContext(ctx).Create(group).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgGroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*Group, error) {
	var group Group
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&group).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrGroupNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &group, nil
}

func (r *PgGroupRepository) Update(ctx context.Context, group *Group) error {
	if err := r.db.WithContext(ctx).Save(group).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgGroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&Group{}, "id = ?", id).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgGroupRepository) ListPlatform(ctx context.Context) ([]Group, error) {
	var groups []Group
	err := r.db.WithContext(ctx).
		Where("group_type = 'platform'").
		Order("name ASC").
		Find(&groups).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return groups, nil
}

func (r *PgGroupRepository) ListByMethodology(ctx context.Context, methodologySlug string) ([]Group, error) {
	var groups []Group
	err := r.db.WithContext(ctx).
		Where("methodology_slug = ?", methodologySlug).
		Order("name ASC").
		Find(&groups).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return groups, nil
}

func (r *PgGroupRepository) IncrementMemberCount(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Group{}).Where("id = ?", id).
		Update("member_count", gorm.Expr("member_count + 1")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgGroupRepository) DecrementMemberCount(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Group{}).Where("id = ?", id).
		Update("member_count", gorm.Expr("GREATEST(member_count - 1, 0)")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── GroupMember Repository ─────────────────────────────────────────────────

type PgGroupMemberRepository struct {
	db *gorm.DB
}

func NewPgGroupMemberRepository(db *gorm.DB) GroupMemberRepository {
	return &PgGroupMemberRepository{db: db}
}

func (r *PgGroupMemberRepository) Create(ctx context.Context, member *GroupMember) error {
	if err := r.db.WithContext(ctx).Create(member).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgGroupMemberRepository) FindByGroupAndFamily(ctx context.Context, groupID, familyID uuid.UUID) (*GroupMember, error) {
	var member GroupMember
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND family_id = ?", groupID, familyID).
		First(&member).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, shared.ErrDatabase(err)
	}
	return &member, nil
}

func (r *PgGroupMemberRepository) Update(ctx context.Context, member *GroupMember) error {
	if err := r.db.WithContext(ctx).Save(member).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgGroupMemberRepository) Delete(ctx context.Context, groupID, familyID uuid.UUID) error {
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND family_id = ?", groupID, familyID).
		Delete(&GroupMember{}).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgGroupMemberRepository) ListByGroup(ctx context.Context, groupID uuid.UUID) ([]GroupMember, error) {
	var members []GroupMember
	err := r.db.WithContext(ctx).
		Where("group_id = ? AND status = 'active'", groupID).
		Order("joined_at ASC").
		Find(&members).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return members, nil
}

func (r *PgGroupMemberRepository) ListGroupsByFamily(ctx context.Context, familyID uuid.UUID) ([]uuid.UUID, error) {
	var members []GroupMember
	err := r.db.WithContext(ctx).
		Select("group_id").
		Where("family_id = ? AND status = 'active'", familyID).
		Find(&members).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	ids := make([]uuid.UUID, len(members))
	for i, m := range members {
		ids[i] = m.GroupID
	}
	return ids, nil
}

func (r *PgGroupMemberRepository) IsMember(ctx context.Context, groupID, familyID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&GroupMember{}).
		Where("group_id = ? AND family_id = ? AND status = 'active'", groupID, familyID).
		Count(&count).Error
	if err != nil {
		return false, shared.ErrDatabase(err)
	}
	return count > 0, nil
}

// ─── Event Repository ───────────────────────────────────────────────────────

type PgEventRepository struct {
	db *gorm.DB
}

func NewPgEventRepository(db *gorm.DB) EventRepository {
	return &PgEventRepository{db: db}
}

func (r *PgEventRepository) Create(ctx context.Context, event *Event) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgEventRepository) FindByID(ctx context.Context, id uuid.UUID) (*Event, error) {
	var event Event
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&event).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, &SocialError{Err: domain.ErrEventNotFound}
		}
		return nil, shared.ErrDatabase(err)
	}
	return &event, nil
}

func (r *PgEventRepository) Update(ctx context.Context, event *Event) error {
	if err := r.db.WithContext(ctx).Save(event).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// CROSS-FAMILY: lists events visible to a family.
// Includes: own events + friend events + group events + discoverable events.
func (r *PgEventRepository) ListVisible(ctx context.Context, familyID uuid.UUID, friendIDs []uuid.UUID, groupIDs []uuid.UUID, offset, limit int) ([]Event, error) {
	query := r.db.WithContext(ctx).Where("status = 'active'")

	// Build OR conditions for visibility:
	// 1. Own events
	// 2. Friend events with friends visibility
	// 3. Group events where user is a member
	// 4. Discoverable events
	conditions := r.db.Where("creator_family_id = ?", familyID)
	if len(friendIDs) > 0 {
		conditions = conditions.Or("creator_family_id IN ? AND visibility = 'friends'", friendIDs)
	}
	if len(groupIDs) > 0 {
		conditions = conditions.Or("group_id IN ? AND visibility = 'group'", groupIDs)
	}
	conditions = conditions.Or("visibility = 'discoverable'")

	var events []Event
	err := query.Where(conditions).
		Order("event_date ASC").
		Offset(offset).Limit(limit).
		Find(&events).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return events, nil
}

func (r *PgEventRepository) ListDiscoverable(ctx context.Context, methodologySlug *string, locationRegion *string) ([]Event, error) {
	q := r.db.WithContext(ctx).
		Where("visibility = 'discoverable' AND status = 'active'").
		Order("event_date ASC")
	if methodologySlug != nil && *methodologySlug != "" {
		q = q.Where("methodology_slug = ?", *methodologySlug)
	}
	if locationRegion != nil && *locationRegion != "" {
		q = q.Where("location_region ILIKE ?", "%"+*locationRegion+"%")
	}
	var events []Event
	if err := q.Find(&events).Error; err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return events, nil
}

func (r *PgEventRepository) IncrementAttendeeCount(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Event{}).Where("id = ?", id).
		Update("attendee_count", gorm.Expr("attendee_count + 1")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgEventRepository) DecrementAttendeeCount(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Model(&Event{}).Where("id = ?", id).
		Update("attendee_count", gorm.Expr("GREATEST(attendee_count - 1, 0)")).Error
	if err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

// ─── EventRSVP Repository ──────────────────────────────────────────────────

type PgEventRSVPRepository struct {
	db *gorm.DB
}

func NewPgEventRSVPRepository(db *gorm.DB) EventRSVPRepository {
	return &PgEventRSVPRepository{db: db}
}

func (r *PgEventRSVPRepository) Create(ctx context.Context, rsvp *EventRSVP) error {
	if err := r.db.WithContext(ctx).Create(rsvp).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgEventRSVPRepository) FindByEventAndFamily(ctx context.Context, eventID, familyID uuid.UUID) (*EventRSVP, error) {
	var rsvp EventRSVP
	err := r.db.WithContext(ctx).
		Where("event_id = ? AND family_id = ?", eventID, familyID).
		First(&rsvp).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, shared.ErrDatabase(err)
	}
	return &rsvp, nil
}

func (r *PgEventRSVPRepository) Update(ctx context.Context, rsvp *EventRSVP) error {
	if err := r.db.WithContext(ctx).Save(rsvp).Error; err != nil {
		return shared.ErrDatabase(err)
	}
	return nil
}

func (r *PgEventRSVPRepository) Delete(ctx context.Context, eventID, familyID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("event_id = ? AND family_id = ?", eventID, familyID).
		Delete(&EventRSVP{})
	if result.Error != nil {
		return shared.ErrDatabase(result.Error)
	}
	if result.RowsAffected == 0 {
		return &SocialError{Err: domain.ErrRSVPNotFound}
	}
	return nil
}

func (r *PgEventRSVPRepository) CountGoing(ctx context.Context, eventID uuid.UUID) (int, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&EventRSVP{}).
		Where("event_id = ? AND status = 'going'", eventID).
		Count(&count).Error
	if err != nil {
		return 0, shared.ErrDatabase(err)
	}
	return int(count), nil
}

func (r *PgEventRSVPRepository) ListByEvent(ctx context.Context, eventID uuid.UUID) ([]EventRSVP, error) {
	var rsvps []EventRSVP
	err := r.db.WithContext(ctx).
		Where("event_id = ?", eventID).
		Order("created_at ASC").
		Find(&rsvps).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	return rsvps, nil
}

func (r *PgEventRSVPRepository) ListGoingFamilyIDs(ctx context.Context, eventID uuid.UUID) ([]uuid.UUID, error) {
	var rsvps []EventRSVP
	err := r.db.WithContext(ctx).
		Select("family_id").
		Where("event_id = ? AND status = 'going'", eventID).
		Find(&rsvps).Error
	if err != nil {
		return nil, shared.ErrDatabase(err)
	}
	ids := make([]uuid.UUID, len(rsvps))
	for i, r := range rsvps {
		ids[i] = r.FamilyID
	}
	return ids, nil
}
