package method

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/method/domain"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ─── Test fixtures ──────────────────────────────────────────────────────────

var (
	cmSlug   = MethodologyID("charlotte-mason")
	tradSlug = MethodologyID("traditional")
)

func newTestService(defRepo MethodologyDefinitionRepository, activationRepo ToolActivationRepository, iamSvc IamServiceForMethod) MethodologyService {
	return NewMethodologyService(defRepo, nil, activationRepo, iamSvc, shared.NewEventBus())
}

// ─── Test: UpdateFamilyMethodology validation ────────────────────────────────

func TestUpdateFamilyMethodology_PrimaryInSecondary(t *testing.T) {
	svc := newTestService(&stubDefRepo{allActive: true}, nil, &stubIamSvc{})
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.Must(uuid.NewV7())})

	_, err := svc.UpdateFamilyMethodology(context.Background(), &scope, UpdateMethodologyCommand{
		PrimaryMethodologySlug:    cmSlug,
		SecondaryMethodologySlugs: []MethodologyID{cmSlug}, // primary also in secondary
	})

	if err == nil {
		t.Fatal("expected error")
	}
	var methodErr *domain.MethodError
	if !errors.As(err, &methodErr) {
		t.Fatalf("want *MethodError, got %T", err)
	}
	if !errors.Is(methodErr.Err, domain.ErrPrimaryInSecondary) {
		t.Errorf("want ErrPrimaryInSecondary, got %v", methodErr.Err)
	}
}

func TestUpdateFamilyMethodology_DuplicateSecondary(t *testing.T) {
	svc := newTestService(&stubDefRepo{allActive: true}, nil, &stubIamSvc{})
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.Must(uuid.NewV7())})

	_, err := svc.UpdateFamilyMethodology(context.Background(), &scope, UpdateMethodologyCommand{
		PrimaryMethodologySlug:    cmSlug,
		SecondaryMethodologySlugs: []MethodologyID{tradSlug, tradSlug}, // duplicate
	})

	if err == nil {
		t.Fatal("expected error")
	}
	var methodErr *domain.MethodError
	if !errors.As(err, &methodErr) {
		t.Fatalf("want *MethodError, got %T", err)
	}
	if !errors.Is(methodErr.Err, domain.ErrDuplicateSecondary) {
		t.Errorf("want ErrDuplicateSecondary, got %v", methodErr.Err)
	}
}

func TestUpdateFamilyMethodology_InvalidSlugs(t *testing.T) {
	svc := newTestService(&stubDefRepo{allActive: false}, nil, &stubIamSvc{})
	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.Must(uuid.NewV7())})

	_, err := svc.UpdateFamilyMethodology(context.Background(), &scope, UpdateMethodologyCommand{
		PrimaryMethodologySlug:    "nonexistent-method", // doesn't exist
		SecondaryMethodologySlugs: nil,
	})

	if err == nil {
		t.Fatal("expected error")
	}
	var methodErr *domain.MethodError
	if !errors.As(err, &methodErr) {
		t.Fatalf("want *MethodError, got %T", err)
	}
	if !errors.Is(methodErr.Err, domain.ErrInvalidMethodologyIDs) {
		t.Errorf("want ErrInvalidMethodologyIDs, got %v", methodErr.Err)
	}
}

// ─── Test: GetDefaultMethodologySlug ────────────────────────────────────────

func TestGetDefaultMethodologySlug_ReturnsFirst(t *testing.T) {
	svc := newTestService(&stubDefRepo{
		methodologies: []MethodologyDefinition{
			{Slug: cmSlug, DisplayOrder: 1, IsActive: true},
			{Slug: tradSlug, DisplayOrder: 2, IsActive: true},
		},
	}, nil, &stubIamSvc{})

	slug, err := svc.GetDefaultMethodologySlug(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if slug != cmSlug {
		t.Errorf("want first methodology slug %q, got %q", cmSlug, slug)
	}
}

func TestGetDefaultMethodologySlug_NoMethodologies(t *testing.T) {
	svc := newTestService(&stubDefRepo{methodologies: nil}, nil, &stubIamSvc{})

	_, err := svc.GetDefaultMethodologySlug(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

// ─── Test: ValidateMethodologySlug ──────────────────────────────────────────

func TestValidateMethodologySlug_Active(t *testing.T) {
	svc := newTestService(&stubDefRepo{
		findBySlugResult: &MethodologyDefinition{Slug: cmSlug, IsActive: true},
	}, nil, &stubIamSvc{})

	valid, err := svc.ValidateMethodologySlug(context.Background(), cmSlug)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Error("want valid=true for active methodology")
	}
}

func TestValidateMethodologySlug_Inactive(t *testing.T) {
	svc := newTestService(&stubDefRepo{
		findBySlugResult: &MethodologyDefinition{Slug: cmSlug, IsActive: false},
	}, nil, &stubIamSvc{})

	valid, err := svc.ValidateMethodologySlug(context.Background(), cmSlug)
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Error("want valid=false for inactive methodology")
	}
}

func TestValidateMethodologySlug_NotFound(t *testing.T) {
	svc := newTestService(&stubDefRepo{}, nil, &stubIamSvc{})

	valid, err := svc.ValidateMethodologySlug(context.Background(), "nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	if valid {
		t.Error("want valid=false for not-found methodology")
	}
}

// ─── Test: ResolveFamilyTools ───────────────────────────────────────────────

func TestResolveFamilyTools_SingleMethodology(t *testing.T) {
	svc := newTestService(nil, &stubActivationRepo{
		activations: []domain.ToolActivationWithTool{
			{
				MethodologySlug: string(cmSlug),
				ToolSlug:        "activities",
				ToolDisplayName: "Activities",
				ToolTier:        "free",
				ToolIsActive:    true,
				ConfigOverrides: json.RawMessage(`{"label": "Lessons"}`),
				SortOrder:       1,
			},
		},
	}, &stubIamSvc{primarySlug: cmSlug})

	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.Must(uuid.NewV7())})
	tools, err := svc.ResolveFamilyTools(context.Background(), &scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("want 1 tool, got %d", len(tools))
	}
	if tools[0].Slug != "activities" {
		t.Errorf("want slug 'activities', got %q", tools[0].Slug)
	}
	if tools[0].Label == nil || *tools[0].Label != "Lessons" {
		t.Errorf("want label 'Lessons', got %v", tools[0].Label)
	}
}

// ─── Test: ResolveStudentTools with override ─────────────────────────────────

func TestResolveStudentTools_WithOverride(t *testing.T) {
	studentID := uuid.Must(uuid.NewV7())

	svc := newTestService(nil, &stubActivationRepo{
		activations: []domain.ToolActivationWithTool{
			{
				MethodologySlug: string(tradSlug),
				ToolSlug:        "tests-grades",
				ToolDisplayName: "Tests & Grades",
				ToolTier:        "free",
				ToolIsActive:    true,
				ConfigOverrides: json.RawMessage(`{}`),
				SortOrder:       1,
			},
		},
	}, &stubIamSvc{
		primarySlug: cmSlug,
		student:     &StudentInfo{ID: studentID, MethodologyOverrideSlug: &tradSlug},
	})

	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.Must(uuid.NewV7())})
	tools, err := svc.ResolveStudentTools(context.Background(), &scope, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("want 1 tool from override methodology, got %d", len(tools))
	}
	if tools[0].Slug != "tests-grades" {
		t.Errorf("want slug 'tests-grades', got %q", tools[0].Slug)
	}
}

// ─── Test: ResolveStudentTools without override (falls through to family) ────

func TestResolveStudentTools_WithoutOverride(t *testing.T) {
	studentID := uuid.Must(uuid.NewV7())

	svc := newTestService(nil, &stubActivationRepo{
		activations: []domain.ToolActivationWithTool{
			{
				MethodologySlug: string(cmSlug),
				ToolSlug:        "activities",
				ToolDisplayName: "Activities",
				ToolTier:        "free",
				ToolIsActive:    true,
				ConfigOverrides: json.RawMessage(`{}`),
				SortOrder:       1,
			},
		},
	}, &stubIamSvc{
		primarySlug: cmSlug,
		student:     &StudentInfo{ID: studentID, MethodologyOverrideSlug: nil}, // no override
	})

	scope := shared.NewFamilyScopeFromAuth(&shared.AuthContext{FamilyID: uuid.Must(uuid.NewV7())})
	tools, err := svc.ResolveStudentTools(context.Background(), &scope, studentID)
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("want 1 tool from family methodology, got %d", len(tools))
	}
}

// ─── Test: ListMethodologies ─────────────────────────────────────────────────

func TestListMethodologies_ReturnsSummaries(t *testing.T) {
	svc := newTestService(&stubDefRepo{
		methodologies: []MethodologyDefinition{
			{Slug: cmSlug, DisplayName: "Charlotte Mason", ShortDesc: "Living books", IsActive: true},
			{Slug: tradSlug, DisplayName: "Traditional", ShortDesc: "Textbook-based", IsActive: true},
		},
	}, nil, &stubIamSvc{})

	resp, err := svc.ListMethodologies(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(resp) != 2 {
		t.Fatalf("want 2 methodologies, got %d", len(resp))
	}
	if resp[0].Slug != cmSlug {
		t.Errorf("want first slug %q, got %q", cmSlug, resp[0].Slug)
	}
	if resp[1].Slug != tradSlug {
		t.Errorf("want second slug %q, got %q", tradSlug, resp[1].Slug)
	}
}

// ─── Test: GetMethodology ─────────────────────────────────────────────────────

func TestGetMethodology_ReturnsDetail(t *testing.T) {
	svc := newTestService(&stubDefRepo{
		findBySlugResult: &MethodologyDefinition{
			Slug:        cmSlug,
			DisplayName: "Charlotte Mason",
			ShortDesc:   "Living books",
			Philosophy:  json.RawMessage(`{"history": "Developed by Charlotte Mason"}`),
		},
	}, nil, &stubIamSvc{})

	resp, err := svc.GetMethodology(context.Background(), "charlotte-mason")
	if err != nil {
		t.Fatal(err)
	}
	if resp.Slug != cmSlug {
		t.Errorf("want slug %q, got %q", cmSlug, resp.Slug)
	}
	if len(resp.Philosophy) == 0 {
		t.Error("philosophy field should not be empty")
	}
}

// ─── Test: GetMethodologyTools ────────────────────────────────────────────────

func TestGetMethodologyTools_ReturnsList(t *testing.T) {
	svc := newTestService(
		&stubDefRepo{
			findBySlugResult: &MethodologyDefinition{Slug: cmSlug, IsActive: true},
		},
		&stubActivationRepo{
			activations: []domain.ToolActivationWithTool{
				{
					MethodologySlug: string(cmSlug),
					ToolSlug:        "activities",
					ToolDisplayName: "Activities",
					ToolTier:        "free",
					ToolIsActive:    true,
					ConfigOverrides: json.RawMessage(`{"label": "Lessons"}`),
					SortOrder:       1,
				},
			},
		},
		&stubIamSvc{},
	)

	tools, err := svc.GetMethodologyTools(context.Background(), "charlotte-mason")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) != 1 {
		t.Fatalf("want 1 tool, got %d", len(tools))
	}
	if tools[0].Slug != "activities" {
		t.Errorf("want slug 'activities', got %q", tools[0].Slug)
	}
	if tools[0].Label == nil || *tools[0].Label != "Lessons" {
		t.Errorf("want label 'Lessons', got %v", tools[0].Label)
	}
}

// ─── Stubs ────────────────────────────────────────────────────────────────────

type stubDefRepo struct {
	methodologies    []MethodologyDefinition
	allActive        bool
	findBySlugResult *MethodologyDefinition
}

func (r *stubDefRepo) ListActive(_ context.Context) ([]MethodologyDefinition, error) {
	return r.methodologies, nil
}
func (r *stubDefRepo) FindBySlug(_ context.Context, slug string) (*MethodologyDefinition, error) {
	if r.findBySlugResult != nil {
		return r.findBySlugResult, nil
	}
	return nil, &domain.MethodError{Err: domain.ErrMethodologyNotFound, Slug: slug}
}
func (r *stubDefRepo) FindBySlugs(_ context.Context, slugs []MethodologyID) ([]MethodologyDefinition, error) {
	result := make([]MethodologyDefinition, 0)
	for _, slug := range slugs {
		for _, m := range r.methodologies {
			if m.Slug == slug {
				result = append(result, m)
			}
		}
	}
	return result, nil
}
func (r *stubDefRepo) AllActive(_ context.Context, _ []MethodologyID) (bool, error) {
	return r.allActive, nil
}

type stubActivationRepo struct {
	activations []domain.ToolActivationWithTool
}

func (r *stubActivationRepo) ListByMethodology(_ context.Context, _ MethodologyID) ([]domain.ToolActivationWithTool, error) {
	return r.activations, nil
}
func (r *stubActivationRepo) ListByMethodologies(_ context.Context, _ []MethodologyID) ([]domain.ToolActivationWithTool, error) {
	return r.activations, nil
}

type stubIamSvc struct {
	primarySlug    MethodologyID
	secondarySlugs []MethodologyID
	student        *StudentInfo
}

func (s *stubIamSvc) GetFamilyMethodologyIDs(_ context.Context, _ *shared.FamilyScope) (MethodologyID, []MethodologyID, error) {
	sec := s.secondarySlugs
	if sec == nil {
		sec = []MethodologyID{}
	}
	return s.primarySlug, sec, nil
}
func (s *stubIamSvc) GetStudent(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID) (*StudentInfo, error) {
	if s.student != nil {
		return s.student, nil
	}
	return nil, &domain.MethodError{Err: domain.ErrStudentNotFound}
}
func (s *stubIamSvc) SetFamilyMethodology(_ context.Context, _ *shared.FamilyScope, _ MethodologyID, _ []MethodologyID) error {
	return nil
}
func (s *stubIamSvc) SetStudentMethodologyOverride(_ context.Context, _ *shared.FamilyScope, _ uuid.UUID, _ *MethodologyID) error {
	return nil
}
