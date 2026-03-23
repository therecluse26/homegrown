package social

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Domain events published by the social domain. [CODING §8.4, 05-social §17.3]
// All events implement shared.DomainEvent.
// Subscribers are registered in cmd/server/main.go via eventBus.Subscribe().

// PostCreated is published after a new post is created.
// Subscribers:
//   - safety:: scans content (needs Content, Attachments)
//   - search:: indexes post
//   - social:: feed fan-out (via job enqueue, not event bus)
type PostCreated struct {
	PostID      uuid.UUID       `json:"post_id"`
	FamilyID    uuid.UUID       `json:"family_id"`
	PostType    string          `json:"post_type"`
	Content     *string         `json:"content,omitempty"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
	GroupID     *uuid.UUID      `json:"group_id,omitempty"`
}

func (PostCreated) EventName() string { return "social.PostCreated" }

// FriendRequestSent is published when a friend request is sent.
// Subscribers:
//   - notify:: sends notification to accepter
type FriendRequestSent struct {
	FriendshipID      uuid.UUID `json:"friendship_id"`
	RequesterFamilyID uuid.UUID `json:"requester_family_id"`
	AccepterFamilyID  uuid.UUID `json:"accepter_family_id"`
}

func (FriendRequestSent) EventName() string { return "social.FriendRequestSent" }

// FriendRequestAccepted is published when a friend request is accepted.
// Subscribers:
//   - notify:: sends notification to requester
//   - social:: rebuild feed for both families
type FriendRequestAccepted struct {
	FriendshipID      uuid.UUID `json:"friendship_id"`
	RequesterFamilyID uuid.UUID `json:"requester_family_id"`
	AccepterFamilyID  uuid.UUID `json:"accepter_family_id"`
}

func (FriendRequestAccepted) EventName() string { return "social.FriendRequestAccepted" }

// MessageSent is published after a DM is sent.
// Subscribers:
//   - safety:: scans content
//   - notify:: sends push notification (needs RecipientParentID, RecipientFamilyID)
type MessageSent struct {
	MessageID         uuid.UUID `json:"message_id"`
	ConversationID    uuid.UUID `json:"conversation_id"`
	SenderParentID    uuid.UUID `json:"sender_parent_id"`
	SenderFamilyID    uuid.UUID `json:"sender_family_id"`
	RecipientParentID uuid.UUID `json:"recipient_parent_id"`
	RecipientFamilyID uuid.UUID `json:"recipient_family_id"`
}

func (MessageSent) EventName() string { return "social.MessageSent" }

// EventCancelled is published when an event is cancelled.
// Subscribers:
//   - notify:: sends notification to attendees (needs EventDate, GoingFamilyIDs)
type EventCancelled struct {
	EventID         uuid.UUID   `json:"event_id"`
	CreatorFamilyID uuid.UUID   `json:"creator_family_id"`
	Title           string      `json:"title"`
	EventDate       time.Time   `json:"event_date"`
	GoingFamilyIDs  []uuid.UUID `json:"going_family_ids"`
}

func (EventCancelled) EventName() string { return "social.EventCancelled" }

// MessageReported is published when a message is reported.
// Subscribers:
//   - safety:: creates moderation ticket
type MessageReported struct {
	MessageID        uuid.UUID `json:"message_id"`
	ConversationID   uuid.UUID `json:"conversation_id"`
	ReporterFamilyID uuid.UUID `json:"reporter_family_id"`
	SenderParentID   uuid.UUID `json:"sender_parent_id"`
	Reason           string    `json:"reason"`
}

func (MessageReported) EventName() string { return "social.MessageReported" }
