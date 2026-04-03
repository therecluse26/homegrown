package learn

import (
	"database/sql/driver"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ═══════════════════════════════════════════════════════════════════════════════
// GORM Models — one per learn_ table. [06-learn §3.2]
// ═══════════════════════════════════════════════════════════════════════════════

// ─── Layer 1: Published Content ─────────────────────────────────────────────

// ActivityDefModel is the GORM model for learn_activity_defs. [S§8.1.1]
type ActivityDefModel struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID        uuid.UUID       `gorm:"type:uuid;not null"`
	Title              string          `gorm:"not null"`
	Description        *string         `gorm:""`
	SubjectTags        StringArray     `gorm:"type:text[];not null;default:'{}'"`
	MethodologyID      *uuid.UUID      `gorm:"type:uuid"`
	ToolID             *uuid.UUID      `gorm:"type:uuid"`
	EstDurationMinutes *int16          `gorm:""`
	Attachments        json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	IsActive           bool            `gorm:"not null;default:true"`
	CreatedAt          time.Time       `gorm:"not null;default:now()"`
	UpdatedAt          time.Time       `gorm:"not null;default:now()"`
}

func (ActivityDefModel) TableName() string { return "learn_activity_defs" }

func (m *ActivityDefModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ReadingItemModel is the GORM model for learn_reading_items. [S§8.1.3]
type ReadingItemModel struct {
	ID            uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID   uuid.UUID   `gorm:"type:uuid;not null"`
	Title         string      `gorm:"not null"`
	Author        *string     `gorm:""`
	ISBN          *string     `gorm:""`
	SubjectTags   StringArray `gorm:"type:text[];not null;default:'{}'"`
	Description   *string     `gorm:""`
	CoverImageURL *string     `gorm:""`
	PageCount     *int16      `gorm:""`
	IsActive      bool        `gorm:"not null;default:true"`
	CreatedAt     time.Time   `gorm:"not null;default:now()"`
	UpdatedAt     time.Time   `gorm:"not null;default:now()"`
}

func (ReadingItemModel) TableName() string { return "learn_reading_items" }

func (m *ReadingItemModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// VideoDefModel is the GORM model for learn_video_defs. [S§8.1.11]
type VideoDefModel struct {
	ID              uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID     uuid.UUID   `gorm:"type:uuid;not null"`
	Title           string      `gorm:"not null"`
	Description     *string     `gorm:""`
	SubjectTags     StringArray `gorm:"type:text[];not null;default:'{}'"`
	MethodologyID   *uuid.UUID  `gorm:"type:uuid"`
	DurationSeconds *int        `gorm:""`
	ThumbnailURL    *string     `gorm:""`
	VideoURL        string      `gorm:"not null"`
	VideoSource     string      `gorm:"not null;default:'self_hosted'"`
	ExternalVideoID *string     `gorm:""`
	TranscodeJobID  *uuid.UUID  `gorm:"type:uuid"`
	IsActive        bool        `gorm:"not null;default:true"`
	CreatedAt       time.Time   `gorm:"not null;default:now()"`
	UpdatedAt       time.Time   `gorm:"not null;default:now()"`
}

func (VideoDefModel) TableName() string { return "learn_video_defs" }

func (m *VideoDefModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// AssessmentDefModel is the GORM model for learn_assessment_defs. [S§8.1.2]
type AssessmentDefModel struct {
	ID          uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID uuid.UUID   `gorm:"type:uuid;not null"`
	Title       string      `gorm:"not null"`
	Description *string     `gorm:""`
	SubjectTags StringArray `gorm:"type:text[];not null;default:'{}'"`
	ScoringType string      `gorm:"not null;default:'percentage'"`
	MaxScore    *float64    `gorm:"type:numeric"`
	IsActive    bool        `gorm:"not null;default:true"`
	CreatedAt   time.Time   `gorm:"not null;default:now()"`
	UpdatedAt   time.Time   `gorm:"not null;default:now()"`
}

func (AssessmentDefModel) TableName() string { return "learn_assessment_defs" }

func (m *AssessmentDefModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ProjectDefModel is the GORM model for learn_project_defs. [S§8.1.5]
type ProjectDefModel struct {
	ID                 uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID        uuid.UUID       `gorm:"type:uuid;not null"`
	Title              string          `gorm:"not null"`
	Description        *string         `gorm:""`
	SubjectTags        StringArray     `gorm:"type:text[];not null;default:'{}'"`
	MilestoneTemplates json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	IsActive           bool            `gorm:"not null;default:true"`
	CreatedAt          time.Time       `gorm:"not null;default:now()"`
	UpdatedAt          time.Time       `gorm:"not null;default:now()"`
}

func (ProjectDefModel) TableName() string { return "learn_project_defs" }

func (m *ProjectDefModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// QuestionModel is the GORM model for learn_questions. [S§8.1.9]
type QuestionModel struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID      uuid.UUID       `gorm:"type:uuid;not null"`
	QuestionType     string          `gorm:"not null"`
	Content          string          `gorm:"not null"`
	MediaAttachments json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	AnswerData       json.RawMessage `gorm:"type:jsonb;not null"`
	SubjectTags      StringArray     `gorm:"type:text[];not null;default:'{}'"`
	MethodologyID    *uuid.UUID      `gorm:"type:uuid"`
	DifficultyLevel  *int16          `gorm:""`
	AutoScorable     bool            `gorm:"not null;default:true"`
	Points           float64         `gorm:"not null;default:1"`
	IsActive         bool            `gorm:"not null;default:true"`
	CreatedAt        time.Time       `gorm:"not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"not null;default:now()"`
}

func (QuestionModel) TableName() string { return "learn_questions" }

func (m *QuestionModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// QuizDefModel is the GORM model for learn_quiz_defs. [S§8.1.9]
type QuizDefModel struct {
	ID                  uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID         uuid.UUID   `gorm:"type:uuid;not null"`
	Title               string      `gorm:"not null"`
	Description         *string     `gorm:""`
	SubjectTags         StringArray `gorm:"type:text[];not null;default:'{}'"`
	MethodologyID       *uuid.UUID  `gorm:"type:uuid"`
	TimeLimitMinutes    *int16      `gorm:""`
	PassingScorePercent int16       `gorm:"not null;default:70"`
	ShuffleQuestions    bool        `gorm:"not null;default:false"`
	ShowCorrectAfter    bool        `gorm:"not null;default:true"`
	QuestionCount       int16       `gorm:"not null;default:0"`
	IsActive            bool        `gorm:"not null;default:true"`
	CreatedAt           time.Time   `gorm:"not null;default:now()"`
	UpdatedAt           time.Time   `gorm:"not null;default:now()"`
}

func (QuizDefModel) TableName() string { return "learn_quiz_defs" }

func (m *QuizDefModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// QuizQuestionModel is the GORM model for learn_quiz_questions. [S§8.1.9]
type QuizQuestionModel struct {
	QuizDefID      uuid.UUID `gorm:"type:uuid;primaryKey"`
	QuestionID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	SortOrder      int16     `gorm:"not null;default:0"`
	PointsOverride *float64  `gorm:""`
}

func (QuizQuestionModel) TableName() string { return "learn_quiz_questions" }

// SequenceDefModel is the GORM model for learn_sequence_defs. [S§8.1.12]
type SequenceDefModel struct {
	ID            uuid.UUID   `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	PublisherID   uuid.UUID   `gorm:"type:uuid;not null"`
	Title         string      `gorm:"not null"`
	Description   *string     `gorm:""`
	SubjectTags   StringArray `gorm:"type:text[];not null;default:'{}'"`
	MethodologyID *uuid.UUID  `gorm:"type:uuid"`
	IsLinear      bool        `gorm:"not null;default:true"`
	IsActive      bool        `gorm:"not null;default:true"`
	CreatedAt     time.Time   `gorm:"not null;default:now()"`
	UpdatedAt     time.Time   `gorm:"not null;default:now()"`
}

func (SequenceDefModel) TableName() string { return "learn_sequence_defs" }

func (m *SequenceDefModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// SequenceItemModel is the GORM model for learn_sequence_items. [S§8.1.12]
type SequenceItemModel struct {
	ID                  uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	SequenceDefID       uuid.UUID `gorm:"type:uuid;not null"`
	SortOrder           int16     `gorm:"not null"`
	ContentType         string    `gorm:"not null"`
	ContentID           uuid.UUID `gorm:"type:uuid;not null"`
	IsRequired          bool      `gorm:"not null;default:true"`
	UnlockAfterPrevious bool      `gorm:"not null;default:false"`
	CreatedAt           time.Time `gorm:"not null;default:now()"`
}

func (SequenceItemModel) TableName() string { return "learn_sequence_items" }

func (m *SequenceItemModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ArtifactLinkModel is the GORM model for learn_artifact_links. [06-learn §9]
type ArtifactLinkModel struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	SourceType   string    `gorm:"not null"`
	SourceID     uuid.UUID `gorm:"type:uuid;not null"`
	TargetType   string    `gorm:"not null"`
	TargetID     uuid.UUID `gorm:"type:uuid;not null"`
	Relationship string    `gorm:"not null;default:'about'"`
	CreatedAt    time.Time `gorm:"not null;default:now()"`
}

func (ArtifactLinkModel) TableName() string { return "learn_artifact_links" }

func (m *ArtifactLinkModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ─── Layer 3: Family Tracking ───────────────────────────────────────────────

// ActivityLogModel is the GORM model for learn_activity_logs. [S§8.1.1]
type ActivityLogModel struct {
	ID              uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID        uuid.UUID       `gorm:"type:uuid;not null"`
	StudentID       uuid.UUID       `gorm:"type:uuid;not null"`
	Title           string          `gorm:"not null"`
	Description     *string         `gorm:""`
	SubjectTags     StringArray     `gorm:"type:text[];not null;default:'{}'"`
	ContentID       *uuid.UUID      `gorm:"type:uuid"`
	MethodologyID   *uuid.UUID      `gorm:"type:uuid"`
	ToolID          *uuid.UUID      `gorm:"type:uuid"`
	DurationMinutes *int16          `gorm:""`
	Attachments     json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	ActivityDate    time.Time       `gorm:"type:date;not null;default:CURRENT_DATE"`
	CreatedAt       time.Time       `gorm:"not null;default:now()"`
	UpdatedAt       time.Time       `gorm:"not null;default:now()"`
}

func (ActivityLogModel) TableName() string { return "learn_activity_logs" }

func (m *ActivityLogModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// JournalEntryModel is the GORM model for learn_journal_entries. [S§8.1.4]
type JournalEntryModel struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID    uuid.UUID       `gorm:"type:uuid;not null"`
	StudentID   uuid.UUID       `gorm:"type:uuid;not null"`
	EntryType   string          `gorm:"not null"`
	Title       *string         `gorm:""`
	Content     string          `gorm:"not null"`
	SubjectTags StringArray     `gorm:"type:text[];not null;default:'{}'"`
	ContentID   *uuid.UUID      `gorm:"type:uuid"`
	Attachments json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	EntryDate   time.Time       `gorm:"type:date;not null;default:CURRENT_DATE"`
	CreatedAt   time.Time       `gorm:"not null;default:now()"`
	UpdatedAt   time.Time       `gorm:"not null;default:now()"`
}

func (JournalEntryModel) TableName() string { return "learn_journal_entries" }

func (m *JournalEntryModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ReadingProgressModel is the GORM model for learn_reading_progress. [S§8.1.3]
type ReadingProgressModel struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID      uuid.UUID  `gorm:"type:uuid;not null"`
	StudentID     uuid.UUID  `gorm:"type:uuid;not null"`
	ReadingItemID uuid.UUID  `gorm:"type:uuid;not null"`
	ReadingListID *uuid.UUID `gorm:"type:uuid"`
	Status        string     `gorm:"not null;default:'to_read'"`
	StartedAt     *time.Time `gorm:""`
	CompletedAt   *time.Time `gorm:""`
	Notes         *string    `gorm:""`
	CreatedAt     time.Time  `gorm:"not null;default:now()"`
	UpdatedAt     time.Time  `gorm:"not null;default:now()"`
}

func (ReadingProgressModel) TableName() string { return "learn_reading_progress" }

func (m *ReadingProgressModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ReadingListModel is the GORM model for learn_reading_lists. [S§8.1.3]
type ReadingListModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID    uuid.UUID  `gorm:"type:uuid;not null"`
	Name        string     `gorm:"not null"`
	Description *string    `gorm:""`
	StudentID   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time  `gorm:"not null;default:now()"`
	UpdatedAt   time.Time  `gorm:"not null;default:now()"`
}

func (ReadingListModel) TableName() string { return "learn_reading_lists" }

func (m *ReadingListModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ReadingListItemModel is the GORM model for learn_reading_list_items.
type ReadingListItemModel struct {
	ReadingListID uuid.UUID `gorm:"type:uuid;primaryKey"`
	ReadingItemID uuid.UUID `gorm:"type:uuid;primaryKey"`
	SortOrder     int16     `gorm:"not null;default:0"`
	AddedAt       time.Time `gorm:"not null;default:now()"`
}

func (ReadingListItemModel) TableName() string { return "learn_reading_list_items" }

// VideoProgressModel is the GORM model for learn_video_progress. [S§8.1.11]
type VideoProgressModel struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID            uuid.UUID  `gorm:"type:uuid;not null"`
	StudentID           uuid.UUID  `gorm:"type:uuid;not null"`
	VideoDefID          uuid.UUID  `gorm:"type:uuid;not null"`
	WatchedSeconds      int        `gorm:"not null;default:0"`
	Completed           bool       `gorm:"not null;default:false"`
	LastPositionSeconds int        `gorm:"not null;default:0"`
	CompletedAt         *time.Time `gorm:""`
	CreatedAt           time.Time  `gorm:"not null;default:now()"`
	UpdatedAt           time.Time  `gorm:"not null;default:now()"`
}

func (VideoProgressModel) TableName() string { return "learn_video_progress" }

func (m *VideoProgressModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// QuizSessionModel is the GORM model for learn_quiz_sessions. [S§8.1.9]
type QuizSessionModel struct {
	ID          uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID    uuid.UUID       `gorm:"type:uuid;not null"`
	StudentID   uuid.UUID       `gorm:"type:uuid;not null"`
	QuizDefID   uuid.UUID       `gorm:"type:uuid;not null"`
	Status      string          `gorm:"not null;default:'not_started'"`
	StartedAt   *time.Time      `gorm:""`
	SubmittedAt *time.Time      `gorm:""`
	ScoredAt    *time.Time      `gorm:""`
	Score       *float64        `gorm:""`
	MaxScore    *float64        `gorm:""`
	Passed      *bool           `gorm:""`
	Answers     json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	ScoredBy    *uuid.UUID      `gorm:"type:uuid"`
	CreatedAt   time.Time       `gorm:"not null;default:now()"`
	UpdatedAt   time.Time       `gorm:"not null;default:now()"`
}

func (QuizSessionModel) TableName() string { return "learn_quiz_sessions" }

func (m *QuizSessionModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// SequenceProgressModel is the GORM model for learn_sequence_progress. [S§8.1.12]
type SequenceProgressModel struct {
	ID               uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID         uuid.UUID       `gorm:"type:uuid;not null"`
	StudentID        uuid.UUID       `gorm:"type:uuid;not null"`
	SequenceDefID    uuid.UUID       `gorm:"type:uuid;not null"`
	CurrentItemIndex int16           `gorm:"not null;default:0"`
	Status           string          `gorm:"not null;default:'not_started'"`
	ItemCompletions  json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	StartedAt        *time.Time      `gorm:""`
	CompletedAt      *time.Time      `gorm:""`
	CreatedAt        time.Time       `gorm:"not null;default:now()"`
	UpdatedAt        time.Time       `gorm:"not null;default:now()"`
}

func (SequenceProgressModel) TableName() string { return "learn_sequence_progress" }

func (m *SequenceProgressModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// StudentAssignmentModel is the GORM model for learn_student_assignments. [S§8.6.3]
type StudentAssignmentModel struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID    uuid.UUID  `gorm:"type:uuid;not null"`
	StudentID   uuid.UUID  `gorm:"type:uuid;not null"`
	AssignedBy  uuid.UUID  `gorm:"type:uuid;not null"`
	ContentType string     `gorm:"not null"`
	ContentID   uuid.UUID  `gorm:"type:uuid;not null"`
	DueDate     *time.Time `gorm:"type:date"`
	Status      string     `gorm:"not null;default:'assigned'"`
	AssignedAt  time.Time  `gorm:"not null;default:now()"`
	CompletedAt *time.Time `gorm:""`
	CreatedAt   time.Time  `gorm:"not null;default:now()"`
	UpdatedAt   time.Time  `gorm:"not null;default:now()"`
}

func (StudentAssignmentModel) TableName() string { return "learn_student_assignments" }

func (m *StudentAssignmentModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// AssessmentResultModel is the GORM model for learn_assessment_results. [S§8.1.2]
type AssessmentResultModel struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID        uuid.UUID `gorm:"type:uuid;not null"`
	StudentID       uuid.UUID `gorm:"type:uuid;not null"`
	AssessmentDefID uuid.UUID `gorm:"type:uuid;not null"`
	Score           float64   `gorm:"type:numeric;not null"`
	MaxScore        *float64  `gorm:"type:numeric"`
	Weight          float64   `gorm:"type:numeric;not null;default:1.0"`
	Notes           *string   `gorm:""`
	AssessmentDate  time.Time `gorm:"type:date;not null;default:CURRENT_DATE"`
	CreatedAt       time.Time `gorm:"not null;default:now()"`
	UpdatedAt       time.Time `gorm:"not null;default:now()"`
}

func (AssessmentResultModel) TableName() string { return "learn_assessment_results" }

func (m *AssessmentResultModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ProjectProgressModel is the GORM model for learn_project_progress. [S§8.1.5]
type ProjectProgressModel struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID     uuid.UUID       `gorm:"type:uuid;not null"`
	StudentID    uuid.UUID       `gorm:"type:uuid;not null"`
	ProjectDefID uuid.UUID       `gorm:"type:uuid;not null"`
	Status       string          `gorm:"not null;default:'planning'"`
	Milestones   json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	StartedAt    *time.Time      `gorm:""`
	CompletedAt  *time.Time      `gorm:""`
	Notes        *string         `gorm:""`
	Attachments  json.RawMessage `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt    time.Time       `gorm:"not null;default:now()"`
	UpdatedAt    time.Time       `gorm:"not null;default:now()"`
}

func (ProjectProgressModel) TableName() string { return "learn_project_progress" }

func (m *ProjectProgressModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// GradingScaleModel is the GORM model for learn_grading_scales. [S§8.1.2]
type GradingScaleModel struct {
	ID        uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID  uuid.UUID       `gorm:"type:uuid;not null"`
	Name      string          `gorm:"not null"`
	ScaleType string          `gorm:"not null"`
	Grades    json.RawMessage `gorm:"type:jsonb;not null"`
	IsDefault bool            `gorm:"not null;default:false"`
	CreatedAt time.Time       `gorm:"not null;default:now()"`
	UpdatedAt time.Time       `gorm:"not null;default:now()"`
}

func (GradingScaleModel) TableName() string { return "learn_grading_scales" }

func (m *GradingScaleModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ─── Supporting Tables ──────────────────────────────────────────────────────

// SubjectTaxonomyModel is the GORM model for learn_subject_taxonomy. [S§8.3]
type SubjectTaxonomyModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	ParentID     *uuid.UUID `gorm:"type:uuid"`
	Name         string     `gorm:"not null"`
	Slug         string     `gorm:"not null;uniqueIndex"`
	Level        int16      `gorm:"not null"`
	DisplayOrder int16      `gorm:"not null;default:0"`
	IsActive     bool       `gorm:"not null;default:true"`
	CreatedAt    time.Time  `gorm:"not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"not null;default:now()"`
}

func (SubjectTaxonomyModel) TableName() string { return "learn_subject_taxonomy" }

// CustomSubjectModel is the GORM model for learn_custom_subjects. [S§8.3]
type CustomSubjectModel struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID         uuid.UUID  `gorm:"type:uuid;not null"`
	ParentTaxonomyID *uuid.UUID `gorm:"type:uuid"`
	Name             string     `gorm:"not null"`
	Slug             string     `gorm:"not null"`
	CreatedAt        time.Time  `gorm:"not null;default:now()"`
}

func (CustomSubjectModel) TableName() string { return "learn_custom_subjects" }

func (m *CustomSubjectModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ProgressSnapshotModel is the GORM model for learn_progress_snapshots. [S§8.1.7]
type ProgressSnapshotModel struct {
	ID           uuid.UUID       `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID     uuid.UUID       `gorm:"type:uuid;not null"`
	StudentID    uuid.UUID       `gorm:"type:uuid;not null"`
	SnapshotDate time.Time       `gorm:"type:date;not null"`
	Data         json.RawMessage `gorm:"type:jsonb;not null"`
	CreatedAt    time.Time       `gorm:"not null;default:now()"`
}

func (ProgressSnapshotModel) TableName() string { return "learn_progress_snapshots" }

func (m *ProgressSnapshotModel) BeforeCreate(_ *gorm.DB) error {
	if m.ID == uuid.Nil {
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}
		m.ID = id
	}
	return nil
}

// ExportRequestModel is the GORM model for learn_export_requests. [S§8.5]
type ExportRequestModel struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:uuidv7()"`
	FamilyID     uuid.UUID  `gorm:"type:uuid;not null"`
	RequestedBy  uuid.UUID  `gorm:"type:uuid;not null"`
	Status       string     `gorm:"not null;default:'pending'"`
	FileURL      *string    `gorm:""`
	ExpiresAt    *time.Time `gorm:""`
	ErrorMessage *string    `gorm:""`
	CreatedAt    time.Time  `gorm:"not null;default:now()"`
	UpdatedAt    time.Time  `gorm:"not null;default:now()"`
}

func (ExportRequestModel) TableName() string { return "learn_export_requests" }

func (m *ExportRequestModel) BeforeCreate(_ *gorm.DB) error {
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
// StringArray — custom type for TEXT[] PostgreSQL arrays. [CODING §2.3]
// ═══════════════════════════════════════════════════════════════════════════════

// StringArray implements driver.Valuer and sql.Scanner for PostgreSQL TEXT[] columns.
type StringArray []string

// Scan implements sql.Scanner.
func (a *StringArray) Scan(src any) error {
	if src == nil {
		*a = StringArray{}
		return nil
	}
	bytes, ok := src.([]byte)
	if !ok {
		str, ok2 := src.(string)
		if !ok2 {
			*a = StringArray{}
			return nil
		}
		bytes = []byte(str)
	}
	*a = parsePostgresArray(string(bytes))
	return nil
}

// Value implements driver.Valuer. Serializes to PostgreSQL array literal.
func (a StringArray) Value() (driver.Value, error) {
	if len(a) == 0 {
		return "{}", nil
	}
	// Quote elements that contain special chars (commas, braces, quotes, backslashes, spaces).
	var b strings.Builder
	b.WriteByte('{')
	for i, v := range a {
		if i > 0 {
			b.WriteByte(',')
		}
		if strings.ContainsAny(v, `,"{}\ `) {
			b.WriteByte('"')
			b.WriteString(strings.ReplaceAll(strings.ReplaceAll(v, `\`, `\\`), `"`, `\"`))
			b.WriteByte('"')
		} else {
			b.WriteString(v)
		}
	}
	b.WriteByte('}')
	return b.String(), nil
}

// parsePostgresArray parses a PostgreSQL text array literal like {foo,bar,"baz qux"}.
func parsePostgresArray(s string) []string {
	if s == "" || s == "{}" {
		return []string{}
	}
	// Strip outer braces
	s = s[1 : len(s)-1]
	var result []string
	var current []byte
	inQuotes := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' && !inQuotes:
			inQuotes = true
		case c == '"' && inQuotes:
			inQuotes = false
		case c == ',' && !inQuotes:
			result = append(result, string(current))
			current = current[:0]
		default:
			current = append(current, c)
		}
	}
	if len(current) > 0 {
		result = append(result, string(current))
	}
	return result
}

// ═══════════════════════════════════════════════════════════════════════════════
// Request Types [06-learn §8.1]
// ═══════════════════════════════════════════════════════════════════════════════

// --- Layer 1: Definition Commands -------------------------------------------

// CreateActivityDefCommand creates an activity definition. [S§8.1.1]
type CreateActivityDefCommand struct {
	PublisherID        uuid.UUID         `json:"publisher_id" validate:"required"`
	Title              string            `json:"title" validate:"required,min=1,max=500"`
	Description        *string           `json:"description,omitempty" validate:"omitempty,max=5000"`
	SubjectTags        []string          `json:"subject_tags"`
	MethodologyID      *uuid.UUID        `json:"methodology_id,omitempty"`
	ToolID             *uuid.UUID        `json:"tool_id,omitempty"`
	EstDurationMinutes *int16            `json:"est_duration_minutes,omitempty"`
	Attachments        []AttachmentInput `json:"attachments"`
}

// UpdateActivityDefCommand updates an activity definition.
type UpdateActivityDefCommand struct {
	Title              *string            `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description        *string            `json:"description,omitempty" validate:"omitempty,max=5000"`
	SubjectTags        *[]string          `json:"subject_tags,omitempty"`
	MethodologyID      *uuid.UUID         `json:"methodology_id,omitempty"`
	ToolID             *uuid.UUID         `json:"tool_id,omitempty"`
	EstDurationMinutes *int16             `json:"est_duration_minutes,omitempty"`
	Attachments        *[]AttachmentInput `json:"attachments,omitempty"`
}

// CreateReadingItemCommand creates a reading item. [S§8.1.3]
type CreateReadingItemCommand struct {
	PublisherID   uuid.UUID `json:"publisher_id" validate:"required"`
	Title         string    `json:"title" validate:"required,min=1,max=500"`
	Author        *string   `json:"author,omitempty" validate:"omitempty,max=300"`
	ISBN          *string   `json:"isbn,omitempty" validate:"omitempty,max=20"`
	SubjectTags   []string  `json:"subject_tags"`
	Description   *string   `json:"description,omitempty" validate:"omitempty,max=2000"`
	CoverImageURL *string   `json:"cover_image_url,omitempty"`
	PageCount     *int16    `json:"page_count,omitempty"`
}

// UpdateReadingItemCommand updates a reading item.
type UpdateReadingItemCommand struct {
	Title         *string   `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Author        *string   `json:"author,omitempty" validate:"omitempty,max=300"`
	ISBN          *string   `json:"isbn,omitempty"`
	SubjectTags   *[]string `json:"subject_tags,omitempty"`
	Description   *string   `json:"description,omitempty"`
	CoverImageURL *string   `json:"cover_image_url,omitempty"`
	PageCount     *int16    `json:"page_count,omitempty"`
}

// CreateArtifactLinkCommand creates an artifact link. [06-learn §9]
type CreateArtifactLinkCommand struct {
	SourceType   string    `json:"source_type" validate:"required"`
	SourceID     uuid.UUID `json:"source_id" validate:"required"`
	TargetType   string    `json:"target_type" validate:"required"`
	TargetID     uuid.UUID `json:"target_id" validate:"required"`
	Relationship *string   `json:"relationship,omitempty"` // defaults to "about"
}

// --- Layer 3: Instance Commands ---------------------------------------------

// LogActivityCommand logs an activity. [S§8.1.1]
type LogActivityCommand struct {
	Title           string            `json:"title" validate:"required,min=1,max=500"`
	Description     *string           `json:"description,omitempty" validate:"omitempty,max=5000"`
	SubjectTags     []string          `json:"subject_tags"`
	ContentID       *uuid.UUID        `json:"content_id,omitempty"`
	MethodologyID   *uuid.UUID        `json:"methodology_id,omitempty"`
	ToolID          *uuid.UUID        `json:"tool_id,omitempty"`
	DurationMinutes *int16            `json:"duration_minutes,omitempty"`
	Attachments     []AttachmentInput `json:"attachments"`
	ActivityDate    *time.Time        `json:"activity_date,omitempty"`
}

// UpdateActivityLogCommand updates an activity log.
type UpdateActivityLogCommand struct {
	Title           *string            `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description     *string            `json:"description,omitempty" validate:"omitempty,max=5000"`
	SubjectTags     *[]string          `json:"subject_tags,omitempty"`
	DurationMinutes *int16             `json:"duration_minutes,omitempty"`
	Attachments     *[]AttachmentInput `json:"attachments,omitempty"`
	ActivityDate    *time.Time         `json:"activity_date,omitempty"`
}

// CreateJournalEntryCommand creates a journal entry. [S§8.1.4]
type CreateJournalEntryCommand struct {
	EntryType   string            `json:"entry_type" validate:"required"`
	Title       *string           `json:"title,omitempty" validate:"omitempty,max=500"`
	Content     string            `json:"content" validate:"required,min=1,max=50000"`
	SubjectTags []string          `json:"subject_tags"`
	ContentID   *uuid.UUID        `json:"content_id,omitempty"`
	Attachments []AttachmentInput `json:"attachments"`
	EntryDate   *time.Time        `json:"entry_date,omitempty"`
}

// UpdateJournalEntryCommand updates a journal entry.
type UpdateJournalEntryCommand struct {
	EntryType   *string            `json:"entry_type,omitempty"`
	Title       *string            `json:"title,omitempty" validate:"omitempty,max=500"`
	Content     *string            `json:"content,omitempty" validate:"omitempty,min=1,max=50000"`
	SubjectTags *[]string          `json:"subject_tags,omitempty"`
	Attachments *[]AttachmentInput `json:"attachments,omitempty"`
	EntryDate   *time.Time         `json:"entry_date,omitempty"`
}

// StartReadingCommand starts tracking a reading item. [S§8.1.3]
type StartReadingCommand struct {
	ReadingItemID uuid.UUID  `json:"reading_item_id" validate:"required"`
	ReadingListID *uuid.UUID `json:"reading_list_id,omitempty"`
}

// UpdateReadingProgressCommand updates reading progress.
type UpdateReadingProgressCommand struct {
	Status *string `json:"status,omitempty"`
	Notes  *string `json:"notes,omitempty" validate:"omitempty,max=2000"`
}

// CreateReadingListCommand creates a reading list. [S§8.1.3]
type CreateReadingListCommand struct {
	Name           string      `json:"name" validate:"required,min=1,max=200"`
	Description    *string     `json:"description,omitempty" validate:"omitempty,max=2000"`
	StudentID      *uuid.UUID  `json:"student_id,omitempty"`
	ReadingItemIDs []uuid.UUID `json:"reading_item_ids"`
}

// UpdateReadingListCommand updates a reading list.
type UpdateReadingListCommand struct {
	Name          *string      `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Description   *string      `json:"description,omitempty" validate:"omitempty,max=2000"`
	AddItemIDs    *[]uuid.UUID `json:"add_item_ids,omitempty"`
	RemoveItemIDs *[]uuid.UUID `json:"remove_item_ids,omitempty"`
}

// CreateCustomSubjectCommand creates a custom subject. [S§8.3]
type CreateCustomSubjectCommand struct {
	Name             string     `json:"name" validate:"required,min=1,max=200"`
	ParentTaxonomyID *uuid.UUID `json:"parent_taxonomy_id,omitempty"`
}

// RequestExportCommand requests a data export. [S§8.5]
type RequestExportCommand struct {
	Format *string `json:"format,omitempty"` // "json" | "csv", defaults to "json"
}

// --- Assessment Engine Commands [S§8.1.9] -----------------------------------

// CreateQuestionCommand creates a question.
type CreateQuestionCommand struct {
	PublisherID      uuid.UUID       `json:"publisher_id" validate:"required"`
	QuestionType     string          `json:"question_type" validate:"required"`
	Content          string          `json:"content" validate:"required,min=1"`
	MediaAttachments json.RawMessage `json:"media_attachments"`
	AnswerData       json.RawMessage `json:"answer_data" validate:"required"`
	SubjectTags      []string        `json:"subject_tags"`
	MethodologyID    *uuid.UUID      `json:"methodology_id,omitempty"`
	DifficultyLevel  *int16          `json:"difficulty_level,omitempty"`
	Points           *float64        `json:"points,omitempty"`
}

// UpdateQuestionCommand updates a question.
type UpdateQuestionCommand struct {
	Content          *string          `json:"content,omitempty"`
	MediaAttachments *json.RawMessage `json:"media_attachments,omitempty"`
	AnswerData       *json.RawMessage `json:"answer_data,omitempty"`
	SubjectTags      *[]string        `json:"subject_tags,omitempty"`
	DifficultyLevel  *int16           `json:"difficulty_level,omitempty"`
	Points           *float64         `json:"points,omitempty"`
}

// QuizQuestionInput is a question reference for quiz creation.
type QuizQuestionInput struct {
	QuestionID     uuid.UUID `json:"question_id" validate:"required"`
	SortOrder      int16     `json:"sort_order"`
	PointsOverride *float64  `json:"points_override,omitempty"`
}

// CreateQuizDefCommand creates a quiz definition.
type CreateQuizDefCommand struct {
	PublisherID         uuid.UUID           `json:"publisher_id" validate:"required"`
	Title               string              `json:"title" validate:"required,min=1,max=500"`
	Description         *string             `json:"description,omitempty"`
	SubjectTags         []string            `json:"subject_tags"`
	MethodologyID       *uuid.UUID          `json:"methodology_id,omitempty"`
	TimeLimitMinutes    *int16              `json:"time_limit_minutes,omitempty"`
	PassingScorePercent *int16              `json:"passing_score_percent,omitempty"`
	ShuffleQuestions    *bool               `json:"shuffle_questions,omitempty"`
	ShowCorrectAfter    *bool               `json:"show_correct_after,omitempty"`
	QuestionIDs         []QuizQuestionInput `json:"question_ids" validate:"required,min=1"`
}

// UpdateQuizDefCommand updates a quiz definition.
type UpdateQuizDefCommand struct {
	Title               *string              `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description         *string              `json:"description,omitempty"`
	SubjectTags         *[]string            `json:"subject_tags,omitempty"`
	TimeLimitMinutes    *int16               `json:"time_limit_minutes,omitempty"`
	PassingScorePercent *int16               `json:"passing_score_percent,omitempty"`
	ShuffleQuestions    *bool                `json:"shuffle_questions,omitempty"`
	ShowCorrectAfter    *bool                `json:"show_correct_after,omitempty"`
	QuestionIDs         *[]QuizQuestionInput `json:"question_ids,omitempty"`
}

// StartQuizSessionCommand starts a quiz session.
type StartQuizSessionCommand struct {
	QuizDefID uuid.UUID `json:"quiz_def_id" validate:"required"`
}

// UpdateQuizSessionCommand saves progress or submits.
type UpdateQuizSessionCommand struct {
	Answers json.RawMessage `json:"answers,omitempty"`
	Submit  *bool           `json:"submit,omitempty"`
}

// QuestionScore is a parent-provided score for a short-answer question.
type QuestionScore struct {
	QuestionID    uuid.UUID `json:"question_id" validate:"required"`
	PointsAwarded float64   `json:"points_awarded"`
	Feedback      *string   `json:"feedback,omitempty"`
}

// ScoreQuizCommand allows parent scoring of short-answer questions.
type ScoreQuizCommand struct {
	Scores []QuestionScore `json:"scores" validate:"required,min=1"`
}

// --- Sequence Engine Commands [S§8.1.12] ------------------------------------

// SequenceItemInput is a content reference for sequence creation.
type SequenceItemInput struct {
	ContentType         string    `json:"content_type" validate:"required"`
	ContentID           uuid.UUID `json:"content_id" validate:"required"`
	SortOrder           int16     `json:"sort_order"`
	IsRequired          *bool     `json:"is_required,omitempty"`
	UnlockAfterPrevious *bool     `json:"unlock_after_previous,omitempty"`
}

// CreateSequenceDefCommand creates a sequence definition.
type CreateSequenceDefCommand struct {
	PublisherID   uuid.UUID           `json:"publisher_id" validate:"required"`
	Title         string              `json:"title" validate:"required,min=1,max=500"`
	Description   *string             `json:"description,omitempty"`
	SubjectTags   []string            `json:"subject_tags"`
	MethodologyID *uuid.UUID          `json:"methodology_id,omitempty"`
	IsLinear      *bool               `json:"is_linear,omitempty"`
	Items         []SequenceItemInput `json:"items" validate:"required,min=1"`
}

// UpdateSequenceDefCommand updates a sequence definition.
type UpdateSequenceDefCommand struct {
	Title       *string              `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description *string              `json:"description,omitempty"`
	SubjectTags *[]string            `json:"subject_tags,omitempty"`
	IsLinear    *bool                `json:"is_linear,omitempty"`
	Items       *[]SequenceItemInput `json:"items,omitempty"`
}

// StartSequenceCommand starts a sequence for a student.
type StartSequenceCommand struct {
	SequenceDefID uuid.UUID `json:"sequence_def_id" validate:"required"`
}

// UpdateSequenceProgressCommand advances/skips/unlocks items.
type UpdateSequenceProgressCommand struct {
	CompleteItemID *uuid.UUID `json:"complete_item_id,omitempty"`
	SkipItemID     *uuid.UUID `json:"skip_item_id,omitempty"`
	UnlockItemID   *uuid.UUID `json:"unlock_item_id,omitempty"`
}

// --- Assignment Commands [S§8.6.3] ------------------------------------------

// CreateAssignmentCommand assigns content to a student.
type CreateAssignmentCommand struct {
	ContentType string     `json:"content_type" validate:"required"`
	ContentID   uuid.UUID  `json:"content_id" validate:"required"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	AssignedBy  uuid.UUID  `json:"-"` // set by handler from auth context
}

// UpdateAssignmentCommand updates assignment status.
type UpdateAssignmentCommand struct {
	Status  *string    `json:"status,omitempty"`
	DueDate *time.Time `json:"due_date,omitempty"`
}

// --- Video Commands ---------------------------------------------------------

// UpdateVideoProgressCommand updates video progress.
type UpdateVideoProgressCommand struct {
	VideoDefID          uuid.UUID `json:"video_def_id" validate:"required"`
	WatchedSeconds      *int      `json:"watched_seconds,omitempty"`
	LastPositionSeconds *int      `json:"last_position_seconds,omitempty"`
	Completed           *bool     `json:"completed,omitempty"`
}

// --- Assessment Def Commands (Phase 2) ----------------------------------------

// CreateAssessmentDefCommand creates an assessment definition.
type CreateAssessmentDefCommand struct {
	PublisherID uuid.UUID `json:"publisher_id" validate:"required"`
	Title       string    `json:"title" validate:"required,min=1,max=500"`
	Description *string   `json:"description,omitempty"`
	SubjectTags []string  `json:"subject_tags"`
	ScoringType string    `json:"scoring_type" validate:"required,oneof=percentage letter points pass_fail"`
	MaxScore    *float64  `json:"max_score,omitempty"`
	CallerID    uuid.UUID `json:"-"`
}

// UpdateAssessmentDefCommand updates an assessment definition.
type UpdateAssessmentDefCommand struct {
	Title       *string  `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description *string  `json:"description,omitempty"`
	SubjectTags []string `json:"subject_tags,omitempty"`
	ScoringType *string  `json:"scoring_type,omitempty" validate:"omitempty,oneof=percentage letter points pass_fail"`
	MaxScore    *float64 `json:"max_score,omitempty"`
	CallerID    uuid.UUID `json:"-"`
}

// --- Project Def Commands (Phase 2) -------------------------------------------

// CreateProjectDefCommand creates a project definition.
type CreateProjectDefCommand struct {
	PublisherID        uuid.UUID       `json:"publisher_id" validate:"required"`
	Title              string          `json:"title" validate:"required,min=1,max=500"`
	Description        *string         `json:"description,omitempty"`
	SubjectTags        []string        `json:"subject_tags"`
	MilestoneTemplates json.RawMessage `json:"milestone_templates"`
	CallerID           uuid.UUID       `json:"-"`
}

// UpdateProjectDefCommand updates a project definition.
type UpdateProjectDefCommand struct {
	Title              *string         `json:"title,omitempty" validate:"omitempty,min=1,max=500"`
	Description        *string         `json:"description,omitempty"`
	SubjectTags        []string        `json:"subject_tags,omitempty"`
	MilestoneTemplates json.RawMessage `json:"milestone_templates,omitempty"`
	CallerID           uuid.UUID       `json:"-"`
}

// --- Assessment Result Commands (Phase 2) -------------------------------------

// RecordAssessmentResultCommand records an assessment result for a student.
type RecordAssessmentResultCommand struct {
	AssessmentDefID uuid.UUID `json:"assessment_def_id" validate:"required"`
	Score           float64   `json:"score" validate:"required"`
	MaxScore        *float64  `json:"max_score,omitempty"`
	Weight          *float64  `json:"weight,omitempty"`
	Notes           *string   `json:"notes,omitempty"`
	AssessmentDate  *string   `json:"assessment_date,omitempty"` // YYYY-MM-DD
}

// UpdateAssessmentResultCommand updates an assessment result.
type UpdateAssessmentResultCommand struct {
	Score          *float64 `json:"score,omitempty"`
	MaxScore       *float64 `json:"max_score,omitempty"`
	Weight         *float64 `json:"weight,omitempty"`
	Notes          *string  `json:"notes,omitempty"`
	AssessmentDate *string  `json:"assessment_date,omitempty"` // YYYY-MM-DD
}

// --- Project Progress Commands (Phase 2) --------------------------------------

// StartProjectCommand starts a project for a student.
type StartProjectCommand struct {
	ProjectDefID uuid.UUID `json:"project_def_id" validate:"required"`
	Notes        *string   `json:"notes,omitempty"`
}

// UpdateProjectProgressCommand updates project progress.
type UpdateProjectProgressCommand struct {
	Status      *string         `json:"status,omitempty" validate:"omitempty,oneof=planning in_progress completed"`
	Milestones  json.RawMessage `json:"milestones,omitempty"`
	Notes       *string         `json:"notes,omitempty"`
	Attachments json.RawMessage `json:"attachments,omitempty"`
}

// --- Grading Scale Commands (Phase 2) -----------------------------------------

// CreateGradingScaleCommand creates a grading scale.
type CreateGradingScaleCommand struct {
	Name      string          `json:"name" validate:"required,min=1,max=200"`
	ScaleType string          `json:"scale_type" validate:"required,oneof=letter pass_fail custom"`
	Grades    json.RawMessage `json:"grades" validate:"required"`
	IsDefault bool            `json:"is_default"`
}

// UpdateGradingScaleCommand updates a grading scale.
type UpdateGradingScaleCommand struct {
	Name      *string         `json:"name,omitempty" validate:"omitempty,min=1,max=200"`
	Grades    json.RawMessage `json:"grades,omitempty"`
	IsDefault *bool           `json:"is_default,omitempty"`
}

// --- Phase 2 Query Types ------------------------------------------------------

// AssessmentDefQuery contains query parameters for assessment definitions.
type AssessmentDefQuery struct {
	Subject     *string
	ScoringType *string
	PublisherID *uuid.UUID
	Search      *string
	Cursor      *uuid.UUID
	Limit       int64
}

// ProjectDefQuery contains query parameters for project definitions.
type ProjectDefQuery struct {
	Subject     *string
	PublisherID *uuid.UUID
	Search      *string
	Cursor      *uuid.UUID
	Limit       int64
}

// AssessmentResultQuery contains query parameters for assessment results.
type AssessmentResultQuery struct {
	AssessmentDefID *uuid.UUID
	DateFrom        *time.Time
	DateTo          *time.Time
	Cursor          *uuid.UUID
	Limit           int64
}

// ProjectProgressQuery contains query parameters for project progress.
type ProjectProgressQuery struct {
	Status       *string
	ProjectDefID *uuid.UUID
	Cursor       *uuid.UUID
	Limit        int64
}

// --- Shared Input Types -----------------------------------------------------

// AttachmentInput is a shared attachment input type.
type AttachmentInput struct {
	URL         string  `json:"url" validate:"required"`
	ContentType string  `json:"content_type" validate:"required"`
	Filename    *string `json:"filename,omitempty"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Response Types [06-learn §8.2]
// ═══════════════════════════════════════════════════════════════════════════════

// --- Layer 1: Definition Responses ------------------------------------------

// ActivityDefResponse is the activity definition response. [S§8.1.1]
type ActivityDefResponse struct {
	ID                 uuid.UUID         `json:"id"`
	PublisherID        uuid.UUID         `json:"publisher_id"`
	Title              string            `json:"title"`
	Description        *string           `json:"description,omitempty"`
	SubjectTags        []string          `json:"subject_tags"`
	MethodologyID      *uuid.UUID        `json:"methodology_id,omitempty"`
	ToolID             *uuid.UUID        `json:"tool_id,omitempty"`
	EstDurationMinutes *int16            `json:"est_duration_minutes,omitempty"`
	Attachments        []AttachmentInput `json:"attachments"`
	CreatedAt          time.Time         `json:"created_at"`
	UpdatedAt          time.Time         `json:"updated_at"`
}

// ActivityDefSummaryResponse is the activity definition summary.
type ActivityDefSummaryResponse struct {
	ID                 uuid.UUID  `json:"id"`
	Title              string     `json:"title"`
	SubjectTags        []string   `json:"subject_tags"`
	MethodologyID      *uuid.UUID `json:"methodology_id,omitempty"`
	EstDurationMinutes *int16     `json:"est_duration_minutes,omitempty"`
}

// ReadingItemResponse is the reading item response. [S§8.1.3]
type ReadingItemResponse struct {
	ID            uuid.UUID `json:"id"`
	PublisherID   uuid.UUID `json:"publisher_id"`
	Title         string    `json:"title"`
	Author        *string   `json:"author,omitempty"`
	ISBN          *string   `json:"isbn,omitempty"`
	SubjectTags   []string  `json:"subject_tags"`
	Description   *string   `json:"description,omitempty"`
	CoverImageURL *string   `json:"cover_image_url,omitempty"`
	PageCount     *int16    `json:"page_count,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// ReadingItemSummaryResponse is the reading item summary.
type ReadingItemSummaryResponse struct {
	ID            uuid.UUID `json:"id"`
	Title         string    `json:"title"`
	Author        *string   `json:"author,omitempty"`
	SubjectTags   []string  `json:"subject_tags"`
	CoverImageURL *string   `json:"cover_image_url,omitempty"`
}

// ReadingItemDetailResponse is the reading item detail with linked artifacts.
type ReadingItemDetailResponse struct {
	ReadingItemResponse
	LinkedArtifacts []ArtifactLinkResponse `json:"linked_artifacts"`
}

// ArtifactLinkResponse is the artifact link response. [06-learn §9]
type ArtifactLinkResponse struct {
	ID           uuid.UUID `json:"id"`
	SourceType   string    `json:"source_type"`
	SourceID     uuid.UUID `json:"source_id"`
	TargetType   string    `json:"target_type"`
	TargetID     uuid.UUID `json:"target_id"`
	Relationship string    `json:"relationship"`
	CreatedAt    time.Time `json:"created_at"`
}

// QuestionResponse is the question response. [S§8.1.9]
type QuestionResponse struct {
	ID               uuid.UUID       `json:"id"`
	PublisherID      uuid.UUID       `json:"publisher_id"`
	QuestionType     string          `json:"question_type"`
	Content          string          `json:"content"`
	MediaAttachments json.RawMessage `json:"media_attachments"`
	AnswerData       json.RawMessage `json:"answer_data"`
	SubjectTags      []string        `json:"subject_tags"`
	MethodologyID    *uuid.UUID      `json:"methodology_id,omitempty"`
	DifficultyLevel  *int16          `json:"difficulty_level,omitempty"`
	AutoScorable     bool            `json:"auto_scorable"`
	Points           float64         `json:"points"`
	CreatedAt        time.Time       `json:"created_at"`
}

// QuestionSummaryResponse is the question summary for quiz building.
type QuestionSummaryResponse struct {
	ID              uuid.UUID  `json:"id"`
	QuestionType    string     `json:"question_type"`
	Content         string     `json:"content"`
	SubjectTags     []string   `json:"subject_tags"`
	DifficultyLevel *int16     `json:"difficulty_level,omitempty"`
	Points          float64    `json:"points"`
	AutoScorable    bool       `json:"auto_scorable"`
	MethodologyID   *uuid.UUID `json:"methodology_id,omitempty"`
}

// QuizDefResponse is the quiz definition response. [S§8.1.9]
type QuizDefResponse struct {
	ID                  uuid.UUID  `json:"id"`
	PublisherID         uuid.UUID  `json:"publisher_id"`
	Title               string     `json:"title"`
	Description         *string    `json:"description,omitempty"`
	SubjectTags         []string   `json:"subject_tags"`
	MethodologyID       *uuid.UUID `json:"methodology_id,omitempty"`
	TimeLimitMinutes    *int16     `json:"time_limit_minutes,omitempty"`
	PassingScorePercent int16      `json:"passing_score_percent"`
	ShuffleQuestions    bool       `json:"shuffle_questions"`
	ShowCorrectAfter    bool       `json:"show_correct_after"`
	QuestionCount       int16      `json:"question_count"`
	CreatedAt           time.Time  `json:"created_at"`
}

// QuizDefDetailResponse is the quiz definition with questions.
type QuizDefDetailResponse struct {
	QuizDefResponse
	Questions []QuizQuestionResponse `json:"questions"`
}

// QuizQuestionResponse is a question within a quiz (may omit answers for student view).
type QuizQuestionResponse struct {
	QuestionID     uuid.UUID       `json:"question_id"`
	SortOrder      int16           `json:"sort_order"`
	Points         float64         `json:"points"`
	QuestionType   string          `json:"question_type"`
	Content        string          `json:"content"`
	AnswerData     json.RawMessage `json:"answer_data,omitempty"` // omitted for student view
	AutoScorable   bool            `json:"auto_scorable"`
}

// QuizSessionResponse is the quiz session response. [S§8.1.9]
type QuizSessionResponse struct {
	ID          uuid.UUID       `json:"id"`
	StudentID   uuid.UUID       `json:"student_id"`
	QuizDefID   uuid.UUID       `json:"quiz_def_id"`
	Status      string          `json:"status"`
	StartedAt   *time.Time      `json:"started_at,omitempty"`
	SubmittedAt *time.Time      `json:"submitted_at,omitempty"`
	ScoredAt    *time.Time      `json:"scored_at,omitempty"`
	Score       *float64        `json:"score,omitempty"`
	MaxScore    *float64        `json:"max_score,omitempty"`
	Passed      *bool           `json:"passed,omitempty"`
	Answers     json.RawMessage `json:"answers"`
	CreatedAt   time.Time       `json:"created_at"`
}

// SequenceDefResponse is the sequence definition response. [S§8.1.12]
type SequenceDefResponse struct {
	ID            uuid.UUID  `json:"id"`
	PublisherID   uuid.UUID  `json:"publisher_id"`
	Title         string     `json:"title"`
	Description   *string    `json:"description,omitempty"`
	SubjectTags   []string   `json:"subject_tags"`
	MethodologyID *uuid.UUID `json:"methodology_id,omitempty"`
	IsLinear      bool       `json:"is_linear"`
	CreatedAt     time.Time  `json:"created_at"`
}

// SequenceDefDetailResponse is the sequence definition with items.
type SequenceDefDetailResponse struct {
	SequenceDefResponse
	Items []SequenceItemResponse `json:"items"`
}

// SequenceItemResponse is a content step within a sequence.
type SequenceItemResponse struct {
	ID                  uuid.UUID `json:"id"`
	SortOrder           int16     `json:"sort_order"`
	ContentType         string    `json:"content_type"`
	ContentID           uuid.UUID `json:"content_id"`
	IsRequired          bool      `json:"is_required"`
	UnlockAfterPrevious bool      `json:"unlock_after_previous"`
}

// SequenceProgressResponse is the sequence progress response. [S§8.1.12]
type SequenceProgressResponse struct {
	ID               uuid.UUID       `json:"id"`
	StudentID        uuid.UUID       `json:"student_id"`
	SequenceDefID    uuid.UUID       `json:"sequence_def_id"`
	CurrentItemIndex int16           `json:"current_item_index"`
	Status           string          `json:"status"`
	ItemCompletions  json.RawMessage `json:"item_completions"`
	StartedAt        *time.Time      `json:"started_at,omitempty"`
	CompletedAt      *time.Time      `json:"completed_at,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// VideoDefResponse is the video definition response. [S§8.1.11]
type VideoDefResponse struct {
	ID              uuid.UUID  `json:"id"`
	PublisherID     uuid.UUID  `json:"publisher_id"`
	Title           string     `json:"title"`
	Description     *string    `json:"description,omitempty"`
	SubjectTags     []string   `json:"subject_tags"`
	MethodologyID   *uuid.UUID `json:"methodology_id,omitempty"`
	DurationSeconds *int       `json:"duration_seconds,omitempty"`
	ThumbnailURL    *string    `json:"thumbnail_url,omitempty"`
	VideoURL        string     `json:"video_url"`
	VideoSource     string     `json:"video_source"`
	CreatedAt       time.Time  `json:"created_at"`
}

// VideoProgressResponse is the video progress response. [S§8.1.11]
type VideoProgressResponse struct {
	ID                  uuid.UUID  `json:"id"`
	StudentID           uuid.UUID  `json:"student_id"`
	VideoDefID          uuid.UUID  `json:"video_def_id"`
	WatchedSeconds      int        `json:"watched_seconds"`
	Completed           bool       `json:"completed"`
	LastPositionSeconds int        `json:"last_position_seconds"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// AssignmentResponse is the assignment response. [S§8.6.3]
type AssignmentResponse struct {
	ID          uuid.UUID  `json:"id"`
	StudentID   uuid.UUID  `json:"student_id"`
	AssignedBy  uuid.UUID  `json:"assigned_by"`
	ContentType string     `json:"content_type"`
	ContentID   uuid.UUID  `json:"content_id"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	Status      string     `json:"status"`
	AssignedAt  time.Time  `json:"assigned_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// AssessmentDefResponse is the assessment definition response. [S§8.1.2]
type AssessmentDefResponse struct {
	ID          uuid.UUID `json:"id"`
	PublisherID uuid.UUID `json:"publisher_id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	SubjectTags []string  `json:"subject_tags"`
	ScoringType string    `json:"scoring_type"`
	MaxScore    *float64  `json:"max_score,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AssessmentDefSummaryResponse is the assessment definition summary.
type AssessmentDefSummaryResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	SubjectTags []string  `json:"subject_tags"`
	ScoringType string    `json:"scoring_type"`
	MaxScore    *float64  `json:"max_score,omitempty"`
}

// ProjectDefResponse is the project definition response. [S§8.1.5]
type ProjectDefResponse struct {
	ID                 uuid.UUID       `json:"id"`
	PublisherID        uuid.UUID       `json:"publisher_id"`
	Title              string          `json:"title"`
	Description        *string         `json:"description,omitempty"`
	SubjectTags        []string        `json:"subject_tags"`
	MilestoneTemplates json.RawMessage `json:"milestone_templates"`
	CreatedAt          time.Time       `json:"created_at"`
	UpdatedAt          time.Time       `json:"updated_at"`
}

// ProjectDefSummaryResponse is the project definition summary.
type ProjectDefSummaryResponse struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	SubjectTags []string  `json:"subject_tags"`
}

// --- Cross-Domain Summary Types -----------------------------------------------

// PortfolioItemSummary is a minimal summary of a learn domain entity,
// consumed by comply:: for portfolio item display. [06-learn §15, 14-comply §9.2]
type PortfolioItemSummary struct {
	Title       string
	Description *string
	Subject     *string
	Date        time.Time
}

// --- Layer 3: Instance Responses --------------------------------------------

// ActivityLogResponse is the activity log response. [S§8.1.1]
type ActivityLogResponse struct {
	ID              uuid.UUID         `json:"id"`
	StudentID       uuid.UUID         `json:"student_id"`
	Title           string            `json:"title"`
	Description     *string           `json:"description,omitempty"`
	SubjectTags     []string          `json:"subject_tags"`
	ContentID       *uuid.UUID        `json:"content_id,omitempty"`
	ContentTitle    *string           `json:"content_title,omitempty"`
	MethodologyID   *uuid.UUID        `json:"methodology_id,omitempty"`
	ToolID          *uuid.UUID        `json:"tool_id,omitempty"`
	DurationMinutes *int16            `json:"duration_minutes,omitempty"`
	Attachments     []AttachmentInput `json:"attachments"`
	ActivityDate    time.Time         `json:"activity_date"`
	CreatedAt       time.Time         `json:"created_at"`
}

// JournalEntryResponse is the journal entry response. [S§8.1.4]
type JournalEntryResponse struct {
	ID          uuid.UUID         `json:"id"`
	StudentID   uuid.UUID         `json:"student_id"`
	EntryType   string            `json:"entry_type"`
	Title       *string           `json:"title,omitempty"`
	Content     string            `json:"content"`
	SubjectTags []string          `json:"subject_tags"`
	Attachments []AttachmentInput `json:"attachments"`
	EntryDate   time.Time         `json:"entry_date"`
	CreatedAt   time.Time         `json:"created_at"`
}

// ReadingProgressResponse is the reading progress response. [S§8.1.3]
type ReadingProgressResponse struct {
	ID            uuid.UUID                  `json:"id"`
	StudentID     uuid.UUID                  `json:"student_id"`
	ReadingItem   ReadingItemSummaryResponse `json:"reading_item"`
	ReadingListID *uuid.UUID                 `json:"reading_list_id,omitempty"`
	Status        string                     `json:"status"`
	StartedAt     *time.Time                 `json:"started_at,omitempty"`
	CompletedAt   *time.Time                 `json:"completed_at,omitempty"`
	Notes         *string                    `json:"notes,omitempty"`
}

// ReadingListSummaryResponse is the reading list summary.
type ReadingListSummaryResponse struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Description    *string    `json:"description,omitempty"`
	StudentID      *uuid.UUID `json:"student_id,omitempty"`
	ItemCount      int64      `json:"item_count"`
	CompletedCount int64      `json:"completed_count"`
}

// ReadingListResponse is the reading list response.
type ReadingListResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Description *string    `json:"description,omitempty"`
	StudentID   *uuid.UUID `json:"student_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// ReadingListDetailResponse is the reading list detail with items and progress.
type ReadingListDetailResponse struct {
	ID          uuid.UUID                    `json:"id"`
	Name        string                       `json:"name"`
	Description *string                      `json:"description,omitempty"`
	StudentID   *uuid.UUID                   `json:"student_id,omitempty"`
	Items       []ReadingListItemWithProgress `json:"items"`
	CreatedAt   time.Time                    `json:"created_at"`
}

// ReadingListItemWithProgress is a reading list item with optional student progress.
type ReadingListItemWithProgress struct {
	ReadingItem ReadingItemSummaryResponse `json:"reading_item"`
	SortOrder   int16                      `json:"sort_order"`
	Progress    *ReadingProgressResponse   `json:"progress,omitempty"`
}

// --- Cross-Cutting Responses ------------------------------------------------

// ProgressSummaryResponse is the progress summary. [S§8.1.7]
type ProgressSummaryResponse struct {
	StudentID       uuid.UUID              `json:"student_id"`
	DateFrom        time.Time              `json:"date_from"`
	DateTo          time.Time              `json:"date_to"`
	TotalActivities int64                  `json:"total_activities"`
	TotalHours      float64                `json:"total_hours"`
	HoursBySubject  []SubjectHoursResponse `json:"hours_by_subject"`
	BooksCompleted  int64                  `json:"books_completed"`
	JournalEntries  int64                  `json:"journal_entries"`
}

// SubjectHoursResponse is hours per subject.
type SubjectHoursResponse struct {
	SubjectSlug string  `json:"subject_slug"`
	SubjectName string  `json:"subject_name"`
	Hours       float64 `json:"hours"`
}

// SubjectProgressResponse is per-subject progress breakdown.
type SubjectProgressResponse struct {
	SubjectSlug    string  `json:"subject_slug"`
	SubjectName    string  `json:"subject_name"`
	ActivityCount  int64   `json:"activity_count"`
	TotalHours     float64 `json:"total_hours"`
	JournalCount   int64   `json:"journal_count"`
	BooksCompleted int64   `json:"books_completed"`
}

// TimelineEntryResponse is an activity timeline entry.
type TimelineEntryResponse struct {
	ID          uuid.UUID `json:"id"`
	EntryType   string    `json:"entry_type"` // "activity" | "journal" | "reading_completed"
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	SubjectTags []string  `json:"subject_tags"`
	Date        time.Time `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

// AssessmentResultResponse is the assessment result response. [S§8.1.2]
type AssessmentResultResponse struct {
	ID              uuid.UUID `json:"id"`
	StudentID       uuid.UUID `json:"student_id"`
	AssessmentDefID uuid.UUID `json:"assessment_def_id"`
	Score           float64   `json:"score"`
	MaxScore        *float64  `json:"max_score,omitempty"`
	Weight          float64   `json:"weight"`
	Notes           *string   `json:"notes,omitempty"`
	AssessmentDate  time.Time `json:"assessment_date"`
	CreatedAt       time.Time `json:"created_at"`
}

// ProjectProgressResponse is the project progress response. [S§8.1.5]
type ProjectProgressResponse struct {
	ID           uuid.UUID       `json:"id"`
	StudentID    uuid.UUID       `json:"student_id"`
	ProjectDefID uuid.UUID       `json:"project_def_id"`
	Status       string          `json:"status"`
	Milestones   json.RawMessage `json:"milestones"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	Notes        *string         `json:"notes,omitempty"`
	Attachments  json.RawMessage `json:"attachments"`
	CreatedAt    time.Time       `json:"created_at"`
}

// GradingScaleResponse is the grading scale response. [S§8.1.2]
type GradingScaleResponse struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	ScaleType string          `json:"scale_type"`
	Grades    json.RawMessage `json:"grades"`
	IsDefault bool            `json:"is_default"`
	CreatedAt time.Time       `json:"created_at"`
}

// SubjectTaxonomyResponse is a subject taxonomy node. [S§8.3]
type SubjectTaxonomyResponse struct {
	ID       uuid.UUID                 `json:"id"`
	ParentID *uuid.UUID                `json:"parent_id,omitempty"`
	Name     string                    `json:"name"`
	Slug     string                    `json:"slug"`
	Level    int16                     `json:"level"`
	Children []SubjectTaxonomyResponse `json:"children"`
	IsCustom bool                      `json:"is_custom"`
}

// CustomSubjectResponse is the custom subject response.
type CustomSubjectResponse struct {
	ID               uuid.UUID  `json:"id"`
	Name             string     `json:"name"`
	Slug             string     `json:"slug"`
	ParentTaxonomyID *uuid.UUID `json:"parent_taxonomy_id,omitempty"`
}

// ExportRequestResponse is the export request response. [S§8.5]
type ExportRequestResponse struct {
	ID        uuid.UUID  `json:"id"`
	Status    string     `json:"status"`
	FileURL   *string    `json:"file_url,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ActiveToolResponse is the active tool response. [S§4.2]
type ActiveToolResponse struct {
	ToolID          uuid.UUID              `json:"tool_id"`
	Slug            string                 `json:"slug"`
	DisplayName     string                 `json:"display_name"`
	Label           *string                `json:"label,omitempty"`
	Description     *string                `json:"description,omitempty"`
	Tier            string                 `json:"tier"`
	Guidance        *string                `json:"guidance,omitempty"`
	ConfigOverrides map[string]any `json:"config_overrides"`
	SortOrder       int16                  `json:"sort_order"`
}

// UploadURLResponse is the upload URL response (from media:: adapter). [06-learn §7]
type UploadURLResponse struct {
	UploadURL string `json:"upload_url"`
	FileKey   string `json:"file_key"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Internal Types [06-learn §8.3]
// ═══════════════════════════════════════════════════════════════════════════════

// SubjectHours is a subject hours aggregation (from repository query).
type SubjectHours struct {
	SubjectSlug  string
	TotalMinutes int64
}

// LinkDirection is a link direction filter for artifact queries.
type LinkDirection int

const (
	LinkDirectionSource LinkDirection = iota
	LinkDirectionTarget
	LinkDirectionBoth
)

// PurchaseMetadata is metadata from a marketplace purchase (received via event).
type PurchaseMetadata struct {
	ContentType string
	ContentIDs  []uuid.UUID
	PublisherID uuid.UUID
}

// ═══════════════════════════════════════════════════════════════════════════════
// Query Types [06-learn §8.3]
// ═══════════════════════════════════════════════════════════════════════════════

// ActivityDefQuery contains query parameters for activity definitions.
type ActivityDefQuery struct {
	Subject       *string
	MethodologyID *uuid.UUID
	PublisherID   *uuid.UUID
	Search        *string
	Cursor        *uuid.UUID
	Limit         int64
}

// ReadingItemQuery contains query parameters for reading items.
type ReadingItemQuery struct {
	Search  *string
	Subject *string
	ISBN    *string
	Cursor  *uuid.UUID
	Limit   int64
}

// ActivityLogQuery contains query parameters for activity logs.
type ActivityLogQuery struct {
	Subject  *string
	DateFrom *time.Time
	DateTo   *time.Time
	Cursor   *uuid.UUID
	Limit    int64
}

// JournalEntryQuery contains query parameters for journal entries.
type JournalEntryQuery struct {
	EntryType *string
	DateFrom  *time.Time
	DateTo    *time.Time
	Search    *string
	Cursor    *uuid.UUID
	Limit     int64
}

// ReadingProgressQuery contains query parameters for reading progress.
type ReadingProgressQuery struct {
	Status *string
	Cursor *uuid.UUID
	Limit  int64
}

// ProgressQuery contains query parameters for progress.
type ProgressQuery struct {
	DateFrom *time.Time
	DateTo   *time.Time
}

// TimelineQuery contains query parameters for timeline.
type TimelineQuery struct {
	DateFrom *time.Time
	DateTo   *time.Time
	Cursor   *uuid.UUID
	Limit    int64
}

// TaxonomyQuery contains query parameters for taxonomy.
type TaxonomyQuery struct {
	Level    *int16
	ParentID *uuid.UUID
}

// QuestionQuery contains query parameters for questions.
type QuestionQuery struct {
	PublisherID   *uuid.UUID
	QuestionType  *string
	Subject       *string
	MethodologyID *uuid.UUID
	Search        *string
	Cursor        *uuid.UUID
	Limit         int64
}

// AssignmentQuery contains query parameters for assignments.
type AssignmentQuery struct {
	Status    *string
	DueBefore *time.Time
	Cursor    *uuid.UUID
	Limit     int64
}

// VideoDefQuery contains query parameters for video definitions.
type VideoDefQuery struct {
	Subject       *string
	MethodologyID *uuid.UUID
	PublisherID   *uuid.UUID
	Search        *string
	Cursor        *uuid.UUID
	Limit         int64
}

// ═══════════════════════════════════════════════════════════════════════════════
// Paginated Response (re-export from shared for convenience)
// ═══════════════════════════════════════════════════════════════════════════════

// PaginatedResponse wraps a page of results with cursor metadata.
type PaginatedResponse[T any] struct {
	Data       []T        `json:"data"`
	NextCursor *uuid.UUID `json:"next_cursor,omitempty"`
	HasMore    bool       `json:"has_more"`
}
