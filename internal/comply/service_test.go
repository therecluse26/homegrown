package comply

import (
	"testing"
)

// TestNewComplianceService_Scaffolding verifies the package compiles
// and the constructor wires all dependencies correctly.
func TestNewComplianceService_Scaffolding(t *testing.T) {
	svc := newTestService(
		&stubStateConfigRepo{},
		&stubFamilyConfigRepo{},
		&stubScheduleRepo{},
		&stubAttendanceRepo{},
		&stubAssessmentRepo{},
		&stubTestScoreRepo{},
		&stubPortfolioRepo{},
		&stubPortfolioItemRepo{},
		&stubTranscriptRepo{},
		&stubCourseRepo{},
		&stubIamService{},
		&stubLearningService{},
		&stubDiscoveryService{},
		&stubMediaService{},
	)
	if svc == nil {
		t.Fatal("expected non-nil ComplianceService")
	}
}
