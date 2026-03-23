package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestCanViewProfile(t *testing.T) {
	familyA := uuid.New()
	familyB := uuid.New()

	tests := []struct {
		name       string
		viewer     uuid.UUID
		target     uuid.UUID
		areFriends bool
		want       bool
	}{
		{"owner can always view own profile", familyA, familyA, false, true},
		{"friend can view profile", familyA, familyB, true, true},
		{"non-friend cannot view profile", familyA, familyB, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanViewProfile(tt.viewer, tt.target, tt.areFriends)
			if got != tt.want {
				t.Errorf("CanViewProfile() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanViewPost(t *testing.T) {
	author := uuid.New()
	viewer := uuid.New()

	tests := []struct {
		name          string
		viewerID      uuid.UUID
		authorID      uuid.UUID
		visibility    string
		areFriends    bool
		isGroupMember bool
		want          bool
	}{
		{"author can view own post", author, author, VisibilityFriends, false, false, true},
		{"friend can view friends-only post", viewer, author, VisibilityFriends, true, false, true},
		{"non-friend cannot view friends-only post", viewer, author, VisibilityFriends, false, false, false},
		{"group member can view group post", viewer, author, VisibilityGroup, false, true, true},
		{"non-member cannot view group post", viewer, author, VisibilityGroup, false, false, false},
		{"unknown visibility denies access", viewer, author, "unknown", true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanViewPost(tt.viewerID, tt.authorID, tt.visibility, tt.areFriends, tt.isGroupMember)
			if got != tt.want {
				t.Errorf("CanViewPost() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanViewEvent(t *testing.T) {
	creator := uuid.New()
	viewer := uuid.New()

	tests := []struct {
		name          string
		viewerID      uuid.UUID
		creatorID     uuid.UUID
		visibility    string
		areFriends    bool
		isGroupMember bool
		want          bool
	}{
		{"creator can view own event", creator, creator, VisibilityFriends, false, false, true},
		{"friend can view friends-only event", viewer, creator, VisibilityFriends, true, false, true},
		{"non-friend cannot view friends-only event", viewer, creator, VisibilityFriends, false, false, false},
		{"group member can view group event", viewer, creator, VisibilityGroup, false, true, true},
		{"non-member cannot view group event", viewer, creator, VisibilityGroup, false, false, false},
		{"discoverable event is visible to all", viewer, creator, VisibilityDiscoverable, false, false, true},
		{"unknown visibility denies access", viewer, creator, "unknown", true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CanViewEvent(tt.viewerID, tt.creatorID, tt.visibility, tt.areFriends, tt.isGroupMember)
			if got != tt.want {
				t.Errorf("CanViewEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterProfileFields_Owner(t *testing.T) {
	settings := DefaultPrivacySettings()
	result := FilterProfileFields(true, false, settings)

	for _, field := range []string{"display_name", "parent_names", "children_names", "children_ages", "location", "methodology"} {
		if !result[field] {
			t.Errorf("owner should see %s, got false", field)
		}
	}
}

func TestFilterProfileFields_Friend_DefaultSettings(t *testing.T) {
	settings := DefaultPrivacySettings()
	result := FilterProfileFields(false, true, settings)

	for _, field := range []string{"display_name", "parent_names", "children_names", "children_ages", "location", "methodology"} {
		if !result[field] {
			t.Errorf("friend with default settings should see %s, got false", field)
		}
	}
}

func TestFilterProfileFields_Friend_HiddenFields(t *testing.T) {
	settings := PrivacySettings{
		DisplayName:   PrivacyFieldFriends,
		ParentNames:   PrivacyFieldHidden,
		ChildrenNames: PrivacyFieldHidden,
		ChildrenAges:  PrivacyFieldFriends,
		Location:      PrivacyFieldHidden,
		Methodology:   PrivacyFieldFriends,
	}
	result := FilterProfileFields(false, true, settings)

	if !result["display_name"] {
		t.Error("friend should see display_name (friends)")
	}
	if result["parent_names"] {
		t.Error("friend should not see parent_names (hidden)")
	}
	if result["children_names"] {
		t.Error("friend should not see children_names (hidden)")
	}
	if !result["children_ages"] {
		t.Error("friend should see children_ages (friends)")
	}
	if result["location"] {
		t.Error("friend should not see location (hidden)")
	}
	if !result["methodology"] {
		t.Error("friend should see methodology (friends)")
	}
}

func TestFilterProfileFields_NonFriend(t *testing.T) {
	settings := DefaultPrivacySettings()
	result := FilterProfileFields(false, false, settings)

	for _, field := range []string{"display_name", "parent_names", "children_names", "children_ages", "location", "methodology"} {
		if result[field] {
			t.Errorf("non-friend should not see %s even with friends-level settings", field)
		}
	}
}

func TestDefaultPrivacySettings(t *testing.T) {
	settings := DefaultPrivacySettings()
	if settings.DisplayName != PrivacyFieldFriends {
		t.Error("default display_name should be friends")
	}
	if settings.ParentNames != PrivacyFieldFriends {
		t.Error("default parent_names should be friends")
	}
}
