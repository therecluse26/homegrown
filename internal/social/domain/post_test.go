package domain

import (
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestValidatePostCreate(t *testing.T) {
	content := "hello world"
	empty := ""

	tests := []struct {
		name           string
		postType       string
		content        *string
		hasAttachments bool
		wantErr        error
	}{
		{"text post with content", PostTypeText, &content, false, nil},
		{"text post without content", PostTypeText, nil, false, ErrPostContentRequired},
		{"text post with empty content", PostTypeText, &empty, false, ErrPostContentRequired},
		{"photo post with attachments", PostTypePhoto, nil, true, nil},
		{"photo post without attachments", PostTypePhoto, nil, false, ErrPostContentRequired},
		{"milestone post with content", PostTypeMilestone, &content, false, nil},
		{"milestone post without content", PostTypeMilestone, nil, false, ErrPostContentRequired},
		{"event_share with content", PostTypeEventShare, &content, false, nil},
		{"resource_share with content", PostTypeResourceShare, &content, false, nil},
		{"marketplace_review with content", PostTypeMarketplaceReview, &content, false, nil},
		{"unknown post type", "unknown", &content, false, ErrInvalidPostType},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePostCreate(tt.postType, tt.content, tt.hasAttachments)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestResolvePostVisibility(t *testing.T) {
	groupID := uuid.New()

	tests := []struct {
		name    string
		groupID *uuid.UUID
		want    string
	}{
		{"no group returns friends", nil, VisibilityFriends},
		{"with group returns group", &groupID, VisibilityGroup},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolvePostVisibility(tt.groupID)
			if got != tt.want {
				t.Errorf("ResolvePostVisibility() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValidateCommentThread(t *testing.T) {
	tests := []struct {
		name             string
		parentHasParent  bool
		wantErr          error
	}{
		{"top-level comment is valid", false, nil},
		{"nested reply is not allowed", true, ErrNestedReplyNotAllowed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCommentThread(tt.parentHasParent)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("expected nil, got %v", err)
				}
			} else {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}
