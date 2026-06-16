package learner_profile

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Custom DB Types [18-learner-profile §2.1]
// ═══════════════════════════════════════════════════════════════════════════════

// StringSlice is a custom type for PostgreSQL TEXT[] columns (e.g., interests).
// Follows the WizardStepArray pattern from onboard::. Safe for slug values that
// never contain commas, quotes, or braces.
type StringSlice []string

// Value implements driver.Valuer — serialises to PostgreSQL array literal.
func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "{}", nil
	}
	parts := make([]string, len(s))
	copy(parts, s)
	return "{" + strings.Join(parts, ",") + "}", nil
}

// Scan implements sql.Scanner — parses PostgreSQL array literal.
func (s *StringSlice) Scan(src any) error {
	if src == nil {
		*s = nil
		return nil
	}
	var str string
	switch v := src.(type) {
	case []byte:
		str = string(v)
	case string:
		str = v
	default:
		return fmt.Errorf("StringSlice.Scan: unsupported type %T", src)
	}
	str = strings.TrimPrefix(str, "{")
	str = strings.TrimSuffix(str, "}")
	if str == "" {
		*s = StringSlice{}
		return nil
	}
	parts := strings.Split(str, ",")
	result := make(StringSlice, len(parts))
	for i, p := range parts {
		result[i] = strings.TrimSpace(strings.Trim(p, `"`))
	}
	*s = result
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// GORM Model [18-learner-profile §2.1]
// ═══════════════════════════════════════════════════════════════════════════════

// LearnerProfile is the GORM model for learner_profiles.
type LearnerProfile struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID          uuid.UUID  `gorm:"type:uuid;not null"`
	StudentID         uuid.UUID  `gorm:"type:uuid;not null;uniqueIndex"`
	ActivityFormat    *float64   `gorm:"type:numeric(5,4)"`
	SessionLength     *float64   `gorm:"type:numeric(5,4)"`
	Motivation        *float64   `gorm:"type:numeric(5,4)"`
	SoloCollaborative *float64   `gorm:"type:numeric(5,4)"`
	Structure         *float64   `gorm:"type:numeric(5,4)"`
	OutdoorKinesthetic *float64  `gorm:"type:numeric(5,4)"`
	Interests         StringSlice `gorm:"type:text[]"`
	AnsweredCount     int16      `gorm:"type:smallint;not null;default:0"`
	Confidence        float64    `gorm:"type:numeric(4,3);not null;default:0"`
	Source            string     `gorm:"type:text;not null;default:'declared'"`
	Respondent        string     `gorm:"type:text;not null;default:'parent'"`
	CreatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()"`
	UpdatedAt         time.Time  `gorm:"type:timestamptz;not null;default:now()"`
}

func (LearnerProfile) TableName() string { return "learner_profiles" }

func (m *LearnerProfile) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types [18-learner-profile §4]
// ═══════════════════════════════════════════════════════════════════════════════

// QuizAnswer is a single scored-question answer from the quiz submission.
// Value is nil when the question was skipped.
type QuizAnswer struct {
	QuestionID int      `json:"question_id" validate:"required,min=1,max=12"`
	Value      *float64 `json:"value"       validate:"omitempty,min=0,max=1"`
}

// SubmitProfileCommand is the request body for POST /v1/students/:student_id/learner-profile/submissions.
type SubmitProfileCommand struct {
	Answers    []QuizAnswer `json:"answers"    validate:"required,min=1,max=12"`
	Respondent string       `json:"respondent" validate:"required,oneof=parent child"`
	Interests  []string     `json:"interests"  validate:"max=20"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types [18-learner-profile §4]
// ═══════════════════════════════════════════════════════════════════════════════

// LearnerProfileResponse is returned by both the submit and get endpoints.
type LearnerProfileResponse struct {
	ID                uuid.UUID `json:"id"`
	StudentID         uuid.UUID `json:"student_id"`
	ActivityFormat    *float64  `json:"activity_format,omitempty"`
	SessionLength     *float64  `json:"session_length,omitempty"`
	Motivation        *float64  `json:"motivation,omitempty"`
	SoloCollaborative *float64  `json:"solo_collaborative,omitempty"`
	Structure         *float64  `json:"structure,omitempty"`
	OutdoorKinesthetic *float64 `json:"outdoor_kinesthetic,omitempty"`
	Interests         []string  `json:"interests"`
	AnsweredCount     int16     `json:"answered_count"`
	Confidence        float64   `json:"confidence"`
	Source            string    `json:"source"`
	Respondent        string    `json:"respondent"`
	SummaryText       string    `json:"summary_text"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// toResponse converts a LearnerProfile model to a LearnerProfileResponse.
func toResponse(p *LearnerProfile, studentName string) *LearnerProfileResponse {
	interests := []string(p.Interests)
	if interests == nil {
		interests = []string{}
	}
	return &LearnerProfileResponse{
		ID:                 p.ID,
		StudentID:          p.StudentID,
		ActivityFormat:     p.ActivityFormat,
		SessionLength:      p.SessionLength,
		Motivation:         p.Motivation,
		SoloCollaborative:  p.SoloCollaborative,
		Structure:          p.Structure,
		OutdoorKinesthetic: p.OutdoorKinesthetic,
		Interests:          interests,
		AnsweredCount:      p.AnsweredCount,
		Confidence:         p.Confidence,
		Source:             p.Source,
		Respondent:         p.Respondent,
		SummaryText:        buildSummaryText(p, studentName),
		CreatedAt:          p.CreatedAt,
		UpdatedAt:          p.UpdatedAt,
	}
}

// buildSummaryText generates a 1–2 sentence human-readable profile summary.
func buildSummaryText(p *LearnerProfile, name string) string {
	if p.AnsweredCount == 0 {
		return ""
	}

	var parts []string

	// Activity format
	if p.ActivityFormat != nil {
		if *p.ActivityFormat >= 0.6 {
			parts = append(parts, "hands-on and creative activities")
		} else if *p.ActivityFormat <= 0.4 {
			parts = append(parts, "reading and listening")
		}
	}

	// Session length
	if p.SessionLength != nil {
		if *p.SessionLength >= 0.6 {
			parts = append(parts, "long focused sessions")
		} else if *p.SessionLength <= 0.4 {
			parts = append(parts, "short focused bursts")
		}
	}

	// Solo vs collaborative
	if p.SoloCollaborative != nil {
		if *p.SoloCollaborative >= 0.6 {
			parts = append(parts, "working with others")
		} else if *p.SoloCollaborative <= 0.4 {
			parts = append(parts, "working independently")
		}
	}

	if len(parts) == 0 {
		return name + " has a balanced learning style across multiple dimensions."
	}

	var sb strings.Builder
	sb.WriteString(name)
	sb.WriteString(" learns best with ")
	for i, p := range parts {
		if i > 0 && i == len(parts)-1 {
			sb.WriteString(" and ")
		} else if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(p)
	}
	sb.WriteByte('.')
	return sb.String()
}
