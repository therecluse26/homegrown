package social

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ─── GORM Models ─────────────────────────────────────────────────────────────
// One model per soc_ table. TableName() and BeforeCreate() follow the project pattern.
// soc_profiles uses family_id as PK (no generated UUID). All others use uuidv7().

// Profile is the GORM model for soc_profiles. [05-social §3.2]
type Profile struct {
	FamilyID        uuid.UUID       `gorm:"type:uuid;primaryKey"`
	Bio             *string         `gorm:""`
	ProfilePhotoURL *string         `gorm:""`
	PrivacySettings json.RawMessage `gorm:"type:jsonb;not null;default:'{}'"`
	LocationVisible bool            `gorm:"not null;default:false"`
	CreatedAt       time.Time       `gorm:"not null;default:now()"`
	UpdatedAt       time.Time       `gorm:"not null;default:now()"`
}

func (Profile) TableName() string { return "soc_profiles" }

// Friendship is the GORM model for soc_friendships. [05-social §3.2]
type Friendship struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	RequesterFamilyID uuid.UUID `gorm:"type:uuid;not null"`
	AccepterFamilyID  uuid.UUID `gorm:"type:uuid;not null"`
	Status            string    `gorm:"not null;default:'pending'"`
	CreatedAt         time.Time `gorm:"not null;default:now()"`
	UpdatedAt         time.Time `gorm:"not null;default:now()"`
}

func (Friendship) TableName() string { return "soc_friendships" }

func (m *Friendship) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// Block is the GORM model for soc_blocks. [05-social §3.2]
type Block struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	BlockerFamilyID uuid.UUID `gorm:"type:uuid;not null"`
	BlockedFamilyID uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt       time.Time `gorm:"not null;default:now()"`
}

func (Block) TableName() string { return "soc_blocks" }

func (m *Block) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// Group is the GORM model for soc_groups. [05-social §3.2]
type Group struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	GroupType         string     `gorm:"not null"`
	Name              string     `gorm:"not null"`
	Description       *string    `gorm:""`
	CoverPhotoURL     *string    `gorm:""`
	CreatorFamilyID   *uuid.UUID `gorm:"type:uuid"`
	MethodologySlug   *string    `gorm:""`
	JoinPolicy        string     `gorm:"not null;default:'open'"`
	MemberCount       int        `gorm:"not null;default:0"`
	CreatedAt         time.Time  `gorm:"not null;default:now()"`
	UpdatedAt         time.Time  `gorm:"not null;default:now()"`
}

func (Group) TableName() string { return "soc_groups" }

func (m *Group) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// GroupMember is the GORM model for soc_group_members. [05-social §3.2]
type GroupMember struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	GroupID   uuid.UUID  `gorm:"type:uuid;not null"`
	FamilyID  uuid.UUID  `gorm:"type:uuid;not null"`
	Role      string     `gorm:"not null;default:'member'"`
	Status    string     `gorm:"not null;default:'active'"`
	JoinedAt  *time.Time `gorm:""`
	CreatedAt time.Time  `gorm:"not null;default:now()"`
	UpdatedAt time.Time  `gorm:"not null;default:now()"`
}

func (GroupMember) TableName() string { return "soc_group_members" }

func (m *GroupMember) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// Post is the GORM model for soc_posts. [05-social §3.2]
type Post struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID       uuid.UUID       `gorm:"type:uuid;not null"`
	AuthorParentID uuid.UUID       `gorm:"type:uuid;not null"`
	PostType       string          `gorm:"not null"`
	Content        *string         `gorm:""`
	Attachments    json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	GroupID        *uuid.UUID      `gorm:"type:uuid"`
	Visibility     string          `gorm:"not null;default:'friends'"`
	LikesCount     int             `gorm:"not null;default:0"`
	CommentsCount  int             `gorm:"not null;default:0"`
	IsEdited       bool            `gorm:"not null;default:false"`
	CreatedAt      time.Time       `gorm:"not null;default:now()"`
	UpdatedAt      time.Time       `gorm:"not null;default:now()"`
}

func (Post) TableName() string { return "soc_posts" }

func (m *Post) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// PinnedPost is the GORM model for soc_pinned_posts. [05-social §4.2]
type PinnedPost struct {
	ID       uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	GroupID  uuid.UUID `gorm:"type:uuid;not null"`
	PostID   uuid.UUID `gorm:"type:uuid;not null"`
	PinnedBy uuid.UUID `gorm:"type:uuid;not null"`
	PinnedAt time.Time `gorm:"not null;default:now()"`
}

func (PinnedPost) TableName() string { return "soc_pinned_posts" }

func (m *PinnedPost) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// Comment is the GORM model for soc_comments. [05-social §3.2]
type Comment struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PostID          uuid.UUID  `gorm:"type:uuid;not null"`
	FamilyID        uuid.UUID  `gorm:"type:uuid;not null"`
	AuthorParentID  uuid.UUID  `gorm:"type:uuid;not null"`
	ParentCommentID *uuid.UUID `gorm:"type:uuid"`
	Content         string     `gorm:"not null"`
	CreatedAt       time.Time  `gorm:"not null;default:now()"`
	UpdatedAt       time.Time  `gorm:"not null;default:now()"`
}

func (Comment) TableName() string { return "soc_comments" }

func (m *Comment) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// PostLike is the GORM model for soc_post_likes. [05-social §3.2]
type PostLike struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PostID    uuid.UUID `gorm:"type:uuid;not null"`
	FamilyID  uuid.UUID `gorm:"type:uuid;not null"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
}

func (PostLike) TableName() string { return "soc_post_likes" }

func (m *PostLike) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// Conversation is the GORM model for soc_conversations. [05-social §3.2]
type Conversation struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (Conversation) TableName() string { return "soc_conversations" }

func (m *Conversation) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ConversationParticipant is the GORM model for soc_conversation_participants. [05-social §3.2]
type ConversationParticipant struct {
	ConversationID uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ParentID       uuid.UUID  `gorm:"type:uuid;primaryKey"`
	FamilyID       uuid.UUID  `gorm:"type:uuid;not null"`
	LastReadAt     *time.Time `gorm:""`
	DeletedAt      *time.Time `gorm:""`
}

func (ConversationParticipant) TableName() string { return "soc_conversation_participants" }

// Message is the GORM model for soc_messages. [05-social §3.2]
type Message struct {
	ID             uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ConversationID uuid.UUID       `gorm:"type:uuid;not null"`
	SenderParentID uuid.UUID       `gorm:"type:uuid;not null"`
	SenderFamilyID uuid.UUID       `gorm:"type:uuid;not null"`
	Content        string          `gorm:"not null"`
	Attachments    json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt      time.Time       `gorm:"not null;default:now()"`
}

func (Message) TableName() string { return "soc_messages" }

func (m *Message) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// Event is the GORM model for soc_events. [05-social §3.2]
type Event struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	CreatorFamilyID  uuid.UUID  `gorm:"type:uuid;not null"`
	CreatorParentID  uuid.UUID  `gorm:"type:uuid;not null"`
	GroupID          *uuid.UUID `gorm:"type:uuid"`
	Title            string     `gorm:"not null"`
	Description      *string    `gorm:""`
	EventDate        time.Time  `gorm:"not null"`
	EndDate          *time.Time `gorm:""`
	LocationName     *string    `gorm:""`
	LocationRegion   *string    `gorm:""`
	IsVirtual        bool       `gorm:"not null;default:false"`
	VirtualURL       *string    `gorm:""`
	Capacity         *int32     `gorm:""`
	Visibility       string     `gorm:"not null;default:'friends'"`
	Status           string     `gorm:"not null;default:'active'"`
	MethodologySlug  *string    `gorm:""`
	AttendeeCount    int        `gorm:"not null;default:0"`
	CreatedAt        time.Time  `gorm:"not null;default:now()"`
	UpdatedAt        time.Time  `gorm:"not null;default:now()"`
}

func (Event) TableName() string { return "soc_events" }

func (m *Event) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// EventRSVP is the GORM model for soc_event_rsvps. [05-social §3.2]
type EventRSVP struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	EventID   uuid.UUID `gorm:"type:uuid;not null"`
	FamilyID  uuid.UUID `gorm:"type:uuid;not null"`
	Status    string    `gorm:"not null;default:'going'"`
	CreatedAt time.Time `gorm:"not null;default:now()"`
	UpdatedAt time.Time `gorm:"not null;default:now()"`
}

func (EventRSVP) TableName() string { return "soc_event_rsvps" }

func (m *EventRSVP) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ─── API Request Types ───────────────────────────────────────────────────────

// UpdateProfileCommand is the request body for PATCH /v1/social/profile. [05-social §8.1]
type UpdateProfileCommand struct {
	Bio             *string          `json:"bio"              validate:"omitempty,max=2000"`
	ProfilePhotoURL *string          `json:"profile_photo_url" validate:"omitempty,url"`
	PrivacySettings *json.RawMessage `json:"privacy_settings" swaggertype:"object"`
	LocationVisible *bool            `json:"location_visible"`
}

// CreatePostCommand is the request body for POST /v1/social/posts. [05-social §8.1]
type CreatePostCommand struct {
	PostType    string          `json:"post_type"    validate:"required,oneof=text photo milestone event_share marketplace_review resource_share"`
	Content     *string         `json:"content"`
	Attachments json.RawMessage `json:"attachments" swaggertype:"object"`
	GroupID     *uuid.UUID      `json:"group_id"`
}

// UpdatePostCommand is the request body for PATCH /v1/social/posts/:id. [05-social §4.2]
// Author only. Sets is_edited = true. Pointer fields allow partial updates.
type UpdatePostCommand struct {
	Content     *string          `json:"content"`
	Attachments *json.RawMessage `json:"attachments" swaggertype:"object"`
}

// CreateCommentCommand is the request body for POST /v1/social/posts/:id/comments. [05-social §8.1]
type CreateCommentCommand struct {
	Content         string     `json:"content"           validate:"required,min=1,max=2000"`
	ParentCommentID *uuid.UUID `json:"parent_comment_id"`
}

// UpdateCommentCommand is the request body for PATCH /v1/social/comments/:id. [05-social §8.1]
type UpdateCommentCommand struct {
	Content string `json:"content" validate:"required,min=1,max=2000"`
}

// CreateConversationCommand is the request body for POST /v1/social/conversations. [05-social §8.1]
// Phase 1: single recipient (1:1 DMs). Multi-participant is Phase 3+.
type CreateConversationCommand struct {
	RecipientParentID uuid.UUID `json:"recipient_parent_id" validate:"required"`
}

// SendMessageCommand is the request body for POST /v1/social/conversations/:id/messages. [05-social §8.1]
type SendMessageCommand struct {
	Content     string          `json:"content"     validate:"required,min=1"`
	Attachments json.RawMessage `json:"attachments" swaggertype:"object"`
}

// CreateGroupCommand is the request body for POST /v1/social/groups. [05-social §8.1]
type CreateGroupCommand struct {
	Name            string  `json:"name"             validate:"required,min=1,max=200"`
	Description     *string `json:"description"      validate:"omitempty,max=2000"`
	CoverPhotoURL   *string `json:"cover_photo_url"  validate:"omitempty,url"`
	JoinPolicy      string  `json:"join_policy"      validate:"required,oneof=open request_to_join invite_only"`
	MethodologySlug *string `json:"methodology_slug"`
}

// UpdateGroupCommand is the request body for PATCH /v1/social/groups/:id. [05-social §8.1]
type UpdateGroupCommand struct {
	Name          *string `json:"name"            validate:"omitempty,min=1,max=200"`
	Description   *string `json:"description"     validate:"omitempty,max=2000"`
	CoverPhotoURL *string `json:"cover_photo_url" validate:"omitempty,url"`
	JoinPolicy    *string `json:"join_policy"     validate:"omitempty,oneof=open request_to_join invite_only"`
}

// CreateEventCommand is the request body for POST /v1/social/events. [05-social §8.1]
type CreateEventCommand struct {
	Title           string     `json:"title"            validate:"required,min=1,max=200"`
	Description     *string    `json:"description"      validate:"omitempty,max=5000"`
	EventDate       time.Time  `json:"event_date"       validate:"required"`
	EndDate         *time.Time `json:"end_date"`
	LocationName    *string    `json:"location_name"    validate:"omitempty,max=200"`
	LocationRegion  *string    `json:"location_region"  validate:"omitempty,max=200"`
	IsVirtual       bool       `json:"is_virtual"`
	VirtualURL      *string    `json:"virtual_url"      validate:"omitempty,url"`
	Capacity        *int       `json:"capacity"         validate:"omitempty,min=1"`
	Visibility      string     `json:"visibility"       validate:"required,oneof=friends group discoverable"`
	GroupID         *uuid.UUID `json:"group_id"`
	MethodologySlug *string    `json:"methodology_slug"`
}

// UpdateEventCommand is the request body for PATCH /v1/social/events/:id. [05-social §8.1]
type UpdateEventCommand struct {
	Title          *string    `json:"title"           validate:"omitempty,min=1,max=200"`
	Description    *string    `json:"description"     validate:"omitempty,max=5000"`
	EventDate      *time.Time `json:"event_date"`
	EndDate        *time.Time `json:"end_date"`
	LocationName   *string    `json:"location_name"   validate:"omitempty,max=200"`
	LocationRegion *string    `json:"location_region" validate:"omitempty,max=200"`
	IsVirtual      *bool      `json:"is_virtual"`
	VirtualURL     *string    `json:"virtual_url"     validate:"omitempty,url"`
	Capacity       *int       `json:"capacity"        validate:"omitempty,min=1"`
	Visibility     *string    `json:"visibility"      validate:"omitempty,oneof=friends group discoverable"`
}

// RSVPCommand is the request body for POST /v1/social/events/:id/rsvp. [05-social §8.1]
type RSVPCommand struct {
	Status string `json:"status" validate:"required,oneof=going interested not_going"`
}

// ReportMessageCommand is the request body for POST /v1/social/messages/:id/report. [05-social §8.1]
type ReportMessageCommand struct {
	Reason string `json:"reason" validate:"required,min=1,max=500"`
}

// ─── API Response Types ──────────────────────────────────────────────────────

// ProfileResponse is the response for profile endpoints. [05-social §8.2]
type ProfileResponse struct {
	FamilyID         uuid.UUID              `json:"family_id"`
	DisplayName      *string                `json:"display_name,omitempty"`
	Bio              *string                `json:"bio,omitempty"`
	ProfilePhotoURL  *string                `json:"profile_photo_url,omitempty"`
	ParentNames      []string               `json:"parent_names,omitempty"`
	Children         []ProfileChildResponse `json:"children,omitempty"`
	MethodologyNames []string               `json:"methodology_names,omitempty"`
	LocationRegion   *string                `json:"location_region,omitempty"`
	LocationVisible  *bool                  `json:"location_visible,omitempty"`
	PrivacySettings  *json.RawMessage       `json:"privacy_settings,omitempty" swaggertype:"object"`
	FriendshipStatus *string                `json:"friendship_status,omitempty"`
	IsFriend         bool                   `json:"is_friend"`
}

// ProfileChildResponse is child info in a profile. [05-social §8.2]
type ProfileChildResponse struct {
	DisplayName string  `json:"display_name"`
	Age         *int16  `json:"age,omitempty"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

// FriendResponse is the response for friend list endpoints. [05-social §8.2]
type FriendResponse struct {
	FamilyID        uuid.UUID `json:"family_id"`
	DisplayName     string    `json:"display_name"`
	ProfilePhotoURL *string   `json:"profile_photo_url,omitempty"`
	MethodologyNames []string `json:"methodology_names,omitempty"`
	FriendsSince    time.Time `json:"friends_since"`
}

// PostResponse is the response for post endpoints. [05-social §8.2]
type PostResponse struct {
	ID             uuid.UUID       `json:"id"`
	FamilyID       uuid.UUID       `json:"family_id"`
	AuthorName     string          `json:"author_name"`
	AuthorPhotoURL *string         `json:"author_photo_url,omitempty"`
	PostType       string          `json:"post_type"`
	Content        *string         `json:"content,omitempty"`
	Attachments    json.RawMessage `json:"attachments" swaggertype:"object"`
	GroupID        *uuid.UUID      `json:"group_id,omitempty"`
	GroupName      *string         `json:"group_name,omitempty"`
	Visibility     string          `json:"visibility"`
	LikesCount     int             `json:"likes_count"`
	CommentsCount  int             `json:"comments_count"`
	IsEdited       bool            `json:"is_edited"`
	IsLikedByMe    bool            `json:"is_liked_by_me"`
	IsMine         bool            `json:"is_mine"`
	CreatedAt      time.Time       `json:"created_at"`
}

// PostDetailResponse is the response for GetPost, including embedded comments. [05-social §8.2]
// Spec nests post under "post" key — not embedded flat.
type PostDetailResponse struct {
	Post     PostResponse      `json:"post"`
	Comments []CommentResponse `json:"comments"`
}

// CommentResponse is the response for comment endpoints. [05-social §8.2]
type CommentResponse struct {
	ID              uuid.UUID         `json:"id"`
	PostID          uuid.UUID         `json:"post_id"`
	FamilyID        uuid.UUID         `json:"family_id"`
	AuthorName      string            `json:"author_name"`
	AuthorPhotoURL  *string           `json:"author_photo_url,omitempty"`
	ParentCommentID *uuid.UUID        `json:"parent_comment_id,omitempty"`
	Content         string            `json:"content"`
	CreatedAt       time.Time         `json:"created_at"`
	Replies         []CommentResponse `json:"replies,omitempty"`
}

// ConversationSummaryResponse is the response for conversation list endpoints. [05-social §8.2]
// Shows OtherParentName, LastMessagePreview, UnreadCount for the list view.
type ConversationSummaryResponse struct {
	ID                 uuid.UUID `json:"id"`
	OtherParentName    string    `json:"other_parent_name"`
	LastMessagePreview *string   `json:"last_message_preview,omitempty"`
	UnreadCount        int       `json:"unread_count"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// ConversationResponse is the response for conversation detail endpoints. [05-social §8.2]
type ConversationResponse struct {
	ID           uuid.UUID                        `json:"id"`
	Participants []ConversationParticipantResponse `json:"participants"`
	UpdatedAt    time.Time                         `json:"updated_at"`
	IsNew        bool                             `json:"-"` // handler-only: 201 if new, 200 if existing
}

// ConversationParticipantResponse is a participant in a conversation. [05-social §8.2]
type ConversationParticipantResponse struct {
	ParentID        uuid.UUID `json:"parent_id"`
	FamilyID        uuid.UUID `json:"family_id"`
	DisplayName     string    `json:"display_name"`
	ProfilePhotoURL *string   `json:"profile_photo_url,omitempty"`
}

// MessageSummary is a preview of the last message in a conversation.
type MessageSummary struct {
	Content   string    `json:"content"`
	SenderID  uuid.UUID `json:"sender_id"`
	CreatedAt time.Time `json:"created_at"`
}

// MessageResponse is the response for message endpoints. [05-social §8.2]
type MessageResponse struct {
	ID             uuid.UUID       `json:"id"`
	ConversationID uuid.UUID       `json:"conversation_id"`
	SenderParentID uuid.UUID       `json:"sender_parent_id"`
	SenderName     string          `json:"sender_name"`
	Content        string          `json:"content"`
	Attachments    json.RawMessage `json:"attachments" swaggertype:"object"`
	CreatedAt      time.Time       `json:"created_at"`
}

// GroupSummaryResponse is the summary for group list endpoints. [05-social §8.2]
type GroupSummaryResponse struct {
	ID             uuid.UUID `json:"id"`
	GroupType      string    `json:"group_type"`
	Name           string    `json:"name"`
	Description    *string   `json:"description,omitempty"`
	CoverPhotoURL  *string   `json:"cover_photo_url,omitempty"`
	MethodologyName *string  `json:"methodology_name,omitempty"`
	JoinPolicy     string    `json:"join_policy"`
	MemberCount    int       `json:"member_count"`
	IsMember       bool      `json:"is_member"`
}

// GroupDetailResponse is the detail for GetGroup. [05-social §8.2]
// Spec nests summary under "summary" key — not embedded flat.
type GroupDetailResponse struct {
	Summary         GroupSummaryResponse `json:"summary"`
	CreatorFamilyID *uuid.UUID           `json:"creator_family_id,omitempty"`
	MyRole          *string              `json:"my_role,omitempty"`
	MyStatus        *string              `json:"my_status,omitempty"`
	CreatedAt       time.Time            `json:"created_at"`
}

// GroupResponse is an alias used by endpoints that return group detail. [05-social §8.2]
type GroupResponse = GroupDetailResponse

// GroupMemberResponse is the response for group member endpoints. [05-social §8.2]
type GroupMemberResponse struct {
	FamilyID    uuid.UUID  `json:"family_id"`
	DisplayName string     `json:"display_name"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	JoinedAt    *time.Time `json:"joined_at,omitempty"`
}

// EventSummaryResponse is the summary for event list endpoints. [05-social §8.2]
type EventSummaryResponse struct {
	ID                uuid.UUID  `json:"id"`
	Title             string     `json:"title"`
	EventDate         time.Time  `json:"event_date"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	LocationName      *string    `json:"location_name,omitempty"`
	LocationRegion    *string    `json:"location_region,omitempty"`
	IsVirtual         bool       `json:"is_virtual"`
	CreatorFamilyName string     `json:"creator_family_name"`
	Capacity          *int32     `json:"capacity,omitempty"`
	Visibility        string     `json:"visibility"`
	Status            string     `json:"status"`
	AttendeeCount     int        `json:"attendee_count"`
	MyRSVP            *string    `json:"my_rsvp,omitempty"`
}

// EventDetailResponse is the detail for GetEvent. [05-social §8.2]
type EventDetailResponse struct {
	EventSummaryResponse
	CreatorFamilyID uuid.UUID          `json:"creator_family_id"`
	GroupID         *uuid.UUID         `json:"group_id,omitempty"`
	GroupName       *string            `json:"group_name,omitempty"`
	Description     *string            `json:"description,omitempty"`
	VirtualURL      *string            `json:"virtual_url,omitempty"`
	MethodologyName *string            `json:"methodology_name,omitempty"`
	Rsvps           []EventRsvpResponse `json:"rsvps,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

// EventRsvpResponse is an RSVP entry in event detail. [05-social §8.2]
type EventRsvpResponse struct {
	FamilyID    uuid.UUID `json:"family_id"`
	DisplayName string    `json:"display_name"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// FriendshipResponse is the response for friend request creation/acceptance. [05-social §8.2]
type FriendshipResponse struct {
	ID                uuid.UUID `json:"id"`
	RequesterFamilyID uuid.UUID `json:"requester_family_id"`
	AccepterFamilyID  uuid.UUID `json:"accepter_family_id"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
}

// FeedResponse wraps a paginated list of posts for the feed endpoint.
type FeedResponse struct {
	Posts      []PostResponse `json:"posts"`
	NextCursor *string        `json:"next_cursor,omitempty"`
}

// MilestoneData carries milestone info from learn:: via domain event. [05-social §5]
type MilestoneData struct {
	StudentName   string `json:"student_name"`
	MilestoneType string `json:"milestone_type"`
	Description   string `json:"description"`
}

// DiscoverableFamilyResponse is the response for GET /v1/social/discover/families. [05-social §8.2]
type DiscoverableFamilyResponse struct {
	FamilyID         uuid.UUID `json:"family_id"`
	DisplayName      string    `json:"display_name"`
	ProfilePhotoURL  *string   `json:"profile_photo_url,omitempty"`
	MethodologyNames []string  `json:"methodology_names,omitempty"`
	LocationRegion   *string   `json:"location_region,omitempty"`
}

// ─── Cross-Domain DTOs ──────────────────────────────────────────────────────

// SocialFamilyInfo is the minimal family data needed by social:: from iam::.
type SocialFamilyInfo struct {
	FamilyID    uuid.UUID
	DisplayName string
	ParentNames []string
}

// SocialParentInfo is the minimal parent data needed by social:: from iam::.
type SocialParentInfo struct {
	ParentID    uuid.UUID
	DisplayName string
	FamilyID    uuid.UUID
}

// ─── Discovery Query Types (Phase 2) ────────────────────────────────────────

// DiscoverFamiliesQuery is the query params for GET /v1/social/discover/families. [05-social §4.2]
type DiscoverFamiliesQuery struct {
	MethodologySlug *string  `query:"methodology_slug"`
	RadiusKm        *float64 `query:"radius_km"`
}

// DiscoverEventsQuery is the query params for GET /v1/social/discover/events. [05-social §4.2]
type DiscoverEventsQuery struct {
	MethodologySlug *string `query:"methodology_slug"`
	LocationRegion  *string `query:"location_region"`
}

// DiscoverGroupsQuery is the query params for GET /v1/social/discover/groups. [05-social §4.2]
type DiscoverGroupsQuery struct {
	MethodologySlug *string `query:"methodology_slug"`
}
