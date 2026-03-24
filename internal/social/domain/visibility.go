package domain

import "github.com/google/uuid"

// Visibility constants. No "public" option — privacy by design. [05-social §9]
const (
	VisibilityFriends      = "friends"
	VisibilityGroup        = "group"
	VisibilityDiscoverable = "discoverable" // events only
)

// Privacy field visibility levels for soc_profiles.privacy_settings.
const (
	PrivacyFieldFriends = "friends"
	PrivacyFieldHidden  = "hidden"
)

// PrivacySettings represents the JSONB privacy_settings on soc_profiles.
type PrivacySettings struct {
	DisplayName   string `json:"display_name"`
	ParentNames   string `json:"parent_names"`
	ChildrenNames string `json:"children_names"`
	ChildrenAges  string `json:"children_ages"`
	Location      string `json:"location"`
	Methodology   string `json:"methodology"`
}

// DefaultPrivacySettings returns the default privacy settings (all friends-visible).
func DefaultPrivacySettings() PrivacySettings {
	return PrivacySettings{
		DisplayName:   PrivacyFieldFriends,
		ParentNames:   PrivacyFieldFriends,
		ChildrenNames: PrivacyFieldFriends,
		ChildrenAges:  PrivacyFieldFriends,
		Location:      PrivacyFieldFriends,
		Methodology:   PrivacyFieldFriends,
	}
}

// CanViewProfile determines whether viewerFamilyID can view targetFamilyID's profile.
// Block check must be done FIRST by the caller — this function assumes no block exists.
// Returns true if the viewer is the owner or is friends with the target. [05-social §9]
func CanViewProfile(viewerFamilyID, targetFamilyID uuid.UUID, areFriends bool) bool {
	if viewerFamilyID == targetFamilyID {
		return true
	}
	return areFriends
}

// CanViewPost determines whether the viewer can see a post.
// Block check must be done FIRST by the caller — blocked interactions return 404.
// - friends-visibility: viewer must be author or friend of author
// - group-visibility: viewer must be a group member
// [05-social §9]
func CanViewPost(viewerFamilyID, authorFamilyID uuid.UUID, visibility string, areFriends, isGroupMember bool) bool {
	if viewerFamilyID == authorFamilyID {
		return true
	}
	switch visibility {
	case VisibilityFriends:
		return areFriends
	case VisibilityGroup:
		return isGroupMember
	default:
		return false
	}
}

// CanViewEvent determines whether the viewer can see an event.
// Block check must be done FIRST by the caller.
// - friends: viewer must be creator or friend of creator
// - group: viewer must be group member
// - discoverable: visible to all (non-blocked) users
// [05-social §9]
func CanViewEvent(viewerFamilyID, creatorFamilyID uuid.UUID, visibility string, areFriends, isGroupMember bool) bool {
	if viewerFamilyID == creatorFamilyID {
		return true
	}
	switch visibility {
	case VisibilityFriends:
		return areFriends
	case VisibilityGroup:
		return isGroupMember
	case VisibilityDiscoverable:
		return true
	default:
		return false
	}
}

// ValidatePrivacySettings checks that all privacy setting values are either "friends" or "hidden".
func ValidatePrivacySettings(settings PrivacySettings) error {
	for _, v := range []string{
		settings.DisplayName,
		settings.ParentNames,
		settings.ChildrenNames,
		settings.ChildrenAges,
		settings.Location,
		settings.Methodology,
	} {
		if v != PrivacyFieldFriends && v != PrivacyFieldHidden {
			return ErrInvalidPrivacySettings
		}
	}
	return nil
}

// FilterProfileFields applies per-field privacy settings to determine which fields
// the viewer can see. Returns a map of field name → visible. The viewer always sees
// all fields on their own profile. Friends see "friends"-level fields. [05-social §9]
func FilterProfileFields(isOwner, isFriend bool, settings PrivacySettings) map[string]bool {
	if isOwner {
		return map[string]bool{
			"display_name":   true,
			"parent_names":   true,
			"children_names": true,
			"children_ages":  true,
			"location":       true,
			"methodology":    true,
		}
	}
	return map[string]bool{
		"display_name":   isFriend && settings.DisplayName == PrivacyFieldFriends,
		"parent_names":   isFriend && settings.ParentNames == PrivacyFieldFriends,
		"children_names": isFriend && settings.ChildrenNames == PrivacyFieldFriends,
		"children_ages":  isFriend && settings.ChildrenAges == PrivacyFieldFriends,
		"location":       isFriend && settings.Location == PrivacyFieldFriends,
		"methodology":    isFriend && settings.Methodology == PrivacyFieldFriends,
	}
}
