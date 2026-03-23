package domain

import (
	"errors"
	"testing"
)

func TestResolveJoinAction(t *testing.T) {
	tests := []struct {
		name       string
		joinPolicy string
		wantStatus string
		wantErr    error
	}{
		{"open group joins immediately", JoinPolicyOpen, GroupMemberStatusActive, nil},
		{"request_to_join creates pending", JoinPolicyRequestToJoin, GroupMemberStatusPending, nil},
		{"invite_only rejects", JoinPolicyInviteOnly, "", ErrGroupInviteOnly},
		{"unknown policy rejects", "unknown", "", ErrGroupInviteOnly},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := ResolveJoinAction(tt.joinPolicy)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if status != tt.wantStatus {
				t.Errorf("got status %q, want %q", status, tt.wantStatus)
			}
		})
	}
}

func TestCanModerate(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{GroupRoleOwner, true},
		{GroupRoleModerator, true},
		{GroupRoleMember, false},
	}
	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			if got := CanModerate(tt.role); got != tt.want {
				t.Errorf("CanModerate(%q) = %v, want %v", tt.role, got, tt.want)
			}
		})
	}
}

func TestValidateLeave(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr error
	}{
		{"member can leave", GroupRoleMember, nil},
		{"moderator can leave", GroupRoleModerator, nil},
		{"owner cannot leave", GroupRoleOwner, ErrOwnerCannotLeave},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateLeave(tt.role)
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

func TestValidateBan(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr error
	}{
		{"member can be banned", GroupRoleMember, nil},
		{"moderator can be banned", GroupRoleModerator, nil},
		{"owner cannot be banned", GroupRoleOwner, ErrCannotBanOwner},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateBan(tt.role)
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
