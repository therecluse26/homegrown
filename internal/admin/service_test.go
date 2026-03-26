package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/homegrown-academy/homegrown-academy/internal/shared"
)

// ds returns default stubs for the two new cross-domain interfaces.
// Keeps existing test call sites clean — they just append ds()... to existing args.
func ds() (MethodologyServiceForAdmin, LifecycleServiceForAdmin) {
	return &stubMethodService{}, &stubLifecycleService{}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area A: Feature Flag Evaluation — evaluateFlag (pure function, §10.2)
// ═══════════════════════════════════════════════════════════════════════════════

func TestEvaluateFlag_A1_DisabledReturnsFalse(t *testing.T) {
	flag := &FeatureFlag{Enabled: false}
	if evaluateFlag(flag, nil) {
		t.Error("disabled flag should return false")
	}
}

func TestEvaluateFlag_A2_EnabledNoConstraintsReturnsTrue(t *testing.T) {
	flag := &FeatureFlag{Enabled: true}
	familyID := uuid.Must(uuid.NewV7())
	if !evaluateFlag(flag, &familyID) {
		t.Error("enabled flag with no constraints should return true")
	}
}

func TestEvaluateFlag_A3_EnabledNilFamilyNoConstraintsReturnsTrue(t *testing.T) {
	flag := &FeatureFlag{Enabled: true}
	if !evaluateFlag(flag, nil) {
		t.Error("enabled flag with nil familyID and no constraints should return true")
	}
}

func TestEvaluateFlag_A4_AllowlistFamilyOnListReturnsTrue(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	flag := &FeatureFlag{
		Enabled:          true,
		AllowedFamilyIDs: []uuid.UUID{uuid.Must(uuid.NewV7()), familyID, uuid.Must(uuid.NewV7())},
	}
	if !evaluateFlag(flag, &familyID) {
		t.Error("family on allowlist should return true")
	}
}

func TestEvaluateFlag_A5_AllowlistFamilyNotOnListReturnsFalse(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	flag := &FeatureFlag{
		Enabled:          true,
		AllowedFamilyIDs: []uuid.UUID{uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())},
	}
	if evaluateFlag(flag, &familyID) {
		t.Error("family NOT on allowlist should return false")
	}
}

func TestEvaluateFlag_A6_AllowlistNilFamilySkipsCheckReturnsTrue(t *testing.T) {
	flag := &FeatureFlag{
		Enabled:          true,
		AllowedFamilyIDs: []uuid.UUID{uuid.Must(uuid.NewV7())},
	}
	// Per spec: allowlist check only runs if familyID != nil.
	// With nil familyID, the allowlist block is skipped, no rollout → returns true.
	if !evaluateFlag(flag, nil) {
		t.Error("allowlist with nil familyID should skip allowlist check and return true")
	}
}

func TestEvaluateFlag_A7_Rollout100ReturnsTrue(t *testing.T) {
	pct := int16(100)
	familyID := uuid.Must(uuid.NewV7())
	flag := &FeatureFlag{
		Enabled:           true,
		RolloutPercentage: &pct,
	}
	if !evaluateFlag(flag, &familyID) {
		t.Error("rollout 100% should return true for any family")
	}
}

func TestEvaluateFlag_A8_Rollout0ReturnsFalse(t *testing.T) {
	pct := int16(0)
	familyID := uuid.Must(uuid.NewV7())
	flag := &FeatureFlag{
		Enabled:           true,
		RolloutPercentage: &pct,
	}
	if evaluateFlag(flag, &familyID) {
		t.Error("rollout 0% should return false for any family")
	}
}

func TestEvaluateFlag_A9_RolloutIsDeterministic(t *testing.T) {
	pct := int16(50)
	familyID := uuid.Must(uuid.NewV7())
	flag := &FeatureFlag{
		Enabled:           true,
		RolloutPercentage: &pct,
	}
	result1 := evaluateFlag(flag, &familyID)
	result2 := evaluateFlag(flag, &familyID)
	if result1 != result2 {
		t.Error("rollout should be deterministic for the same familyID")
	}
}

func TestEvaluateFlag_A10_RolloutNilFamilyReturnsTrue(t *testing.T) {
	pct := int16(50)
	flag := &FeatureFlag{
		Enabled:           true,
		RolloutPercentage: &pct,
	}
	if !evaluateFlag(flag, nil) {
		t.Error("rollout with nil familyID should fall through to true")
	}
}

func TestEvaluateFlag_A11_AllowlistPrecedesRollout(t *testing.T) {
	pct := int16(0) // rollout 0% would return false
	familyID := uuid.Must(uuid.NewV7())
	flag := &FeatureFlag{
		Enabled:           true,
		AllowedFamilyIDs:  []uuid.UUID{familyID},
		RolloutPercentage: &pct,
	}
	if !evaluateFlag(flag, &familyID) {
		t.Error("allowlist should take precedence over rollout percentage")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area H: Flag Key Validation (pure function)
// ═══════════════════════════════════════════════════════════════════════════════

func TestValidateFlagKey_H1_ValidKeys(t *testing.T) {
	validKeys := []string{"new_quiz_builder", "beta-feature", "dark_mode_v2"}
	for _, key := range validKeys {
		if !validateFlagKey(key) {
			t.Errorf("expected key %q to be valid", key)
		}
	}
}

func TestValidateFlagKey_H2_EmptyKeyInvalid(t *testing.T) {
	if validateFlagKey("") {
		t.Error("empty key should be invalid")
	}
}

func TestValidateFlagKey_H3_KeyOver100CharsInvalid(t *testing.T) {
	longKey := strings.Repeat("a", 101)
	if validateFlagKey(longKey) {
		t.Error("key > 100 chars should be invalid")
	}
}

func TestValidateFlagKey_H4_InvalidChars(t *testing.T) {
	invalidKeys := []string{
		"Has Spaces",
		"HasUpperCase",
		"has!special",
		"has.dots",
		"has@symbol",
	}
	for _, key := range invalidKeys {
		if validateFlagKey(key) {
			t.Errorf("expected key %q to be invalid", key)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area B: Feature Flag CRUD (service → repo + audit, §5, §8)
// ═══════════════════════════════════════════════════════════════════════════════

func TestListFlags_B1_DelegatesToRepo(t *testing.T) {
	expected := []FeatureFlag{
		{Key: "flag1", Enabled: true},
		{Key: "flag2", Enabled: false},
	}
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{listAllFn: func(_ context.Context) ([]FeatureFlag, error) {
			return expected, nil
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	flags, err := svc.ListFlags(context.Background(), testAuth())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(flags))
	}
	if flags[0].Key != "flag1" || flags[1].Key != "flag2" {
		t.Errorf("unexpected flags: %+v", flags)
	}
}

func TestListFlags_B2_RepoErrorPropagates(t *testing.T) {
	repoErr := errors.New("database down")
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{listAllFn: func(_ context.Context) ([]FeatureFlag, error) {
			return nil, repoErr
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.ListFlags(context.Background(), testAuth())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetFlag_B2a_ReturnsFlag(t *testing.T) {
	expected := &FeatureFlag{Key: "my_flag", Enabled: true}
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, key string) (*FeatureFlag, error) {
			if key != "my_flag" {
				t.Errorf("expected key 'my_flag', got %q", key)
			}
			return expected, nil
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	flag, err := svc.GetFlag(context.Background(), testAuth(), "my_flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flag.Key != "my_flag" {
		t.Errorf("expected key 'my_flag', got %q", flag.Key)
	}
}

func TestGetFlag_B2b_NotFoundReturnsError(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, _ string) (*FeatureFlag, error) {
			return nil, nil // not found
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.GetFlag(context.Background(), testAuth(), "missing")
	if !errors.Is(err, ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestCreateFlag_B3_ValidInputCreatesFlag(t *testing.T) {
	created := &FeatureFlag{
		ID:      uuid.Must(uuid.NewV7()),
		Key:     "new_feature",
		Enabled: true,
	}

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{createFn: func(_ context.Context, input *CreateFlagInput, _ uuid.UUID) (*FeatureFlag, error) {
			if input.Key != "new_feature" {
				t.Errorf("expected key 'new_feature', got %q", input.Key)
			}
			return created, nil
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	flag, err := svc.CreateFlag(context.Background(), testAuth(), &CreateFlagInput{
		Key:         "new_feature",
		Description: "A new feature",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flag.Key != "new_feature" {
		t.Errorf("expected key 'new_feature', got %q", flag.Key)
	}
}

func TestCreateFlag_B4_DuplicateKeyReturnsErrFlagAlreadyExists(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{createFn: func(_ context.Context, _ *CreateFlagInput, _ uuid.UUID) (*FeatureFlag, error) {
			return nil, ErrFlagAlreadyExists
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.CreateFlag(context.Background(), testAuth(), &CreateFlagInput{
		Key:         "existing_flag",
		Description: "Already exists",
	})
	if !errors.Is(err, ErrFlagAlreadyExists) {
		t.Errorf("expected ErrFlagAlreadyExists, got %v", err)
	}
}

func TestCreateFlag_B5_InvalidKeyReturnsErrInvalidFlagKey(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.CreateFlag(context.Background(), testAuth(), &CreateFlagInput{
		Key:         "INVALID KEY",
		Description: "Bad key",
	})
	if !errors.Is(err, ErrInvalidFlagKey) {
		t.Errorf("expected ErrInvalidFlagKey, got %v", err)
	}
}

func TestCreateFlag_B6_LogsAuditEntry(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{createFn: func(_ context.Context, _ *CreateFlagInput, _ uuid.UUID) (*FeatureFlag, error) {
			return &FeatureFlag{ID: uuid.Must(uuid.NewV7()), Key: "new_flag"}, nil
		}},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.CreateFlag(context.Background(), testAuth(), &CreateFlagInput{
		Key:         "new_flag",
		Description: "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil {
		t.Fatal("expected audit entry to be logged")
	}
	if capturedEntry.Action != "flag_create" {
		t.Errorf("expected action 'flag_create', got %q", capturedEntry.Action)
	}
}

func TestUpdateFlag_B7_ValidUpdateReturnsUpdatedFlag(t *testing.T) {
	enabled := true
	updated := &FeatureFlag{
		ID:      uuid.Must(uuid.NewV7()),
		Key:     "my_flag",
		Enabled: true,
	}

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{updateFn: func(_ context.Context, key string, _ *UpdateFlagInput, _ uuid.UUID) (*FeatureFlag, error) {
			if key != "my_flag" {
				t.Errorf("expected key 'my_flag', got %q", key)
			}
			return updated, nil
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	flag, err := svc.UpdateFlag(context.Background(), testAuth(), "my_flag", &UpdateFlagInput{
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if flag.Key != "my_flag" {
		t.Errorf("expected key 'my_flag', got %q", flag.Key)
	}
}

func TestUpdateFlag_B8_NotFoundReturnsErrFlagNotFound(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{updateFn: func(_ context.Context, _ string, _ *UpdateFlagInput, _ uuid.UUID) (*FeatureFlag, error) {
			return nil, ErrFlagNotFound
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.UpdateFlag(context.Background(), testAuth(), "missing", &UpdateFlagInput{})
	if !errors.Is(err, ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestUpdateFlag_B9_LogsAuditEntry(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{updateFn: func(_ context.Context, _ string, _ *UpdateFlagInput, _ uuid.UUID) (*FeatureFlag, error) {
			return &FeatureFlag{ID: uuid.Must(uuid.NewV7()), Key: "my_flag"}, nil
		}},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.UpdateFlag(context.Background(), testAuth(), "my_flag", &UpdateFlagInput{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil {
		t.Fatal("expected audit entry to be logged")
	}
	if capturedEntry.Action != "flag_update" {
		t.Errorf("expected action 'flag_update', got %q", capturedEntry.Action)
	}
}

func TestDeleteFlag_B10_SuccessReturnsNil(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{deleteFn: func(_ context.Context, key string) error {
			if key != "old_flag" {
				t.Errorf("expected key 'old_flag', got %q", key)
			}
			return nil
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.DeleteFlag(context.Background(), testAuth(), "old_flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteFlag_B11_NotFoundReturnsErrFlagNotFound(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{deleteFn: func(_ context.Context, _ string) error {
			return ErrFlagNotFound
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.DeleteFlag(context.Background(), testAuth(), "missing")
	if !errors.Is(err, ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestDeleteFlag_B12_LogsAuditEntry(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{deleteFn: func(_ context.Context, _ string) error {
			return nil
		}},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.DeleteFlag(context.Background(), testAuth(), "old_flag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil {
		t.Fatal("expected audit entry to be logged")
	}
	if capturedEntry.Action != "flag_delete" {
		t.Errorf("expected action 'flag_delete', got %q", capturedEntry.Action)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area C: IsFlagEnabled (cache + repo + evaluateFlag, §10.2)
// ═══════════════════════════════════════════════════════════════════════════════

func TestIsFlagEnabled_C1_CacheHitReturnsFlagResult(t *testing.T) {
	flag := FeatureFlag{Key: "cached_flag", Enabled: true}
	flagJSON, _ := json.Marshal(flag)

	dbCalled := false
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, _ string) (*FeatureFlag, error) {
			dbCalled = true
			return nil, errors.New("should not be called")
		}},
		&stubAuditRepo{},
		&stubCache{getFn: func(_ context.Context, key string) (string, error) {
			if key == "flag:cached_flag" {
				return string(flagJSON), nil
			}
			return "", nil
		}},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.IsFlagEnabled(context.Background(), "cached_flag", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true from cached enabled flag")
	}
	if dbCalled {
		t.Error("DB should not have been called on cache hit")
	}
}

func TestIsFlagEnabled_C2_CacheMissFallsBackToDB(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, key string) (*FeatureFlag, error) {
			return &FeatureFlag{Key: key, Enabled: true}, nil
		}},
		&stubAuditRepo{},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.IsFlagEnabled(context.Background(), "db_flag", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result {
		t.Error("expected true from DB flag")
	}
}

func TestIsFlagEnabled_C3_CacheMissStoresInCache(t *testing.T) {
	var cachedKey string
	var cachedTTL time.Duration

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, _ string) (*FeatureFlag, error) {
			return &FeatureFlag{Key: "my_flag", Enabled: true}, nil
		}},
		&stubAuditRepo{},
		&stubCache{setFn: func(_ context.Context, key string, _ string, ttl time.Duration) error {
			cachedKey = key
			cachedTTL = ttl
			return nil
		}},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.IsFlagEnabled(context.Background(), "my_flag", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cachedKey != "flag:my_flag" {
		t.Errorf("expected cache key 'flag:my_flag', got %q", cachedKey)
	}
	if cachedTTL != 60*time.Second {
		t.Errorf("expected 60s TTL, got %v", cachedTTL)
	}
}

func TestIsFlagEnabled_C4_FlagNotFoundReturnsError(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, _ string) (*FeatureFlag, error) {
			return nil, nil // not found
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.IsFlagEnabled(context.Background(), "missing_flag", nil)
	if !errors.Is(err, ErrFlagNotFound) {
		t.Errorf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestIsFlagEnabled_C5_DBErrorPropagates(t *testing.T) {
	dbErr := errors.New("connection refused")
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, _ string) (*FeatureFlag, error) {
			return nil, dbErr
		}},
		&stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.IsFlagEnabled(context.Background(), "broken_flag", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIsFlagEnabled_C6_CacheWriteFailureIsNonFatal(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{findByKeyFn: func(_ context.Context, _ string) (*FeatureFlag, error) {
			return &FeatureFlag{Key: "flag", Enabled: true}, nil
		}},
		&stubAuditRepo{},
		&stubCache{setFn: func(_ context.Context, _ string, _ string, _ time.Duration) error {
			return errors.New("redis write failed")
		}},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.IsFlagEnabled(context.Background(), "flag", nil)
	if err != nil {
		t.Fatalf("cache write failure should be non-fatal, got: %v", err)
	}
	if !result {
		t.Error("expected true despite cache write failure")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area D: Audit Log (§8)
// ═══════════════════════════════════════════════════════════════════════════════

func TestLogAction_D1_DelegatesToAuditRepo(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	auth := testAuth()
	targetID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.LogAction(context.Background(), auth, &AdminAction{
		Action:     "user_suspend",
		TargetType: "family",
		TargetID:   &targetID,
		Details:    json.RawMessage(`{"reason":"test"}`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil {
		t.Fatal("expected audit entry to be created")
	}
	if capturedEntry.AdminID != auth.ParentID {
		t.Errorf("expected admin ID %s, got %s", auth.ParentID, capturedEntry.AdminID)
	}
	if capturedEntry.Action != "user_suspend" {
		t.Errorf("expected action 'user_suspend', got %q", capturedEntry.Action)
	}
	if capturedEntry.TargetType != "family" {
		t.Errorf("expected target type 'family', got %q", capturedEntry.TargetType)
	}
	if *capturedEntry.TargetID != targetID {
		t.Errorf("expected target ID %s, got %s", targetID, *capturedEntry.TargetID)
	}
}

func TestLogAction_D2_RepoErrorPropagates(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, _ *CreateAuditLogEntry) (*AuditLogEntry, error) {
			return nil, errors.New("audit write failed")
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.LogAction(context.Background(), testAuth(), &AdminAction{
		Action:     "test",
		TargetType: "system",
		Details:    json.RawMessage(`{}`),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSearchAuditLog_D3_DelegatesToRepo(t *testing.T) {
	expected := []AuditLogEntry{
		{ID: uuid.Must(uuid.NewV7()), Action: "flag_create"},
	}
	query := &AuditLogQuery{Action: strPtr("flag_create")}
	pagination := defaultPagination()

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{searchFn: func(_ context.Context, q *AuditLogQuery, p *shared.PaginationParams) ([]AuditLogEntry, error) {
			if *q.Action != "flag_create" {
				t.Errorf("expected action filter 'flag_create', got %q", *q.Action)
			}
			_ = p
			return expected, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.SearchAuditLog(context.Background(), testAuth(), query, pagination)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result.Data))
	}
}

func TestSearchAuditLog_D4_RepoErrorPropagates(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{searchFn: func(_ context.Context, _ *AuditLogQuery, _ *shared.PaginationParams) ([]AuditLogEntry, error) {
			return nil, errors.New("search failed")
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.SearchAuditLog(context.Background(), testAuth(), &AuditLogQuery{}, defaultPagination())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUserAuditTrail_D5_CallsFindByTarget(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())
	var capturedTargetType string
	var capturedTargetID uuid.UUID

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{findByTargetFn: func(_ context.Context, targetType string, targetID uuid.UUID, _ *shared.PaginationParams) ([]AuditLogEntry, error) {
			capturedTargetType = targetType
			capturedTargetID = targetID
			return []AuditLogEntry{}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.GetUserAuditTrail(context.Background(), testAuth(), familyID, defaultPagination())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedTargetType != "family" {
		t.Errorf("expected target type 'family', got %q", capturedTargetType)
	}
	if capturedTargetID != familyID {
		t.Errorf("expected target ID %s, got %s", familyID, capturedTargetID)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area E: User Management (delegates to IAM/safety/billing, §4)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSearchUsers_E1_DelegatesToIam(t *testing.T) {
	expected := &shared.PaginatedResponse[AdminUserSummary]{
		Data: []AdminUserSummary{{FamilyName: "Smith"}},
	}

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{searchUsersFn: func(_ context.Context, _ *UserSearchQuery, _ *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error) {
			return expected, nil
		}},
		&stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.SearchUsers(context.Background(), testAuth(), &UserSearchQuery{}, defaultPagination())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].FamilyName != "Smith" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestSearchUsers_E2_IamErrorPropagates(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{searchUsersFn: func(_ context.Context, _ *UserSearchQuery, _ *shared.PaginationParams) (*shared.PaginatedResponse[AdminUserSummary], error) {
			return nil, errors.New("IAM unavailable")
		}},
		&stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.SearchUsers(context.Background(), testAuth(), &UserSearchQuery{}, defaultPagination())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestGetUserDetail_E3_AggregatesFromMultipleDomains(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{
			getFamilyDetailFn: func(_ context.Context, _ uuid.UUID) (*AdminFamilyInfo, error) {
				return &AdminFamilyInfo{ID: familyID, Name: "Smith"}, nil
			},
			getParentsFn: func(_ context.Context, _ uuid.UUID) ([]AdminParentInfo, error) {
				return []AdminParentInfo{{DisplayName: "Alice"}}, nil
			},
			getStudentsFn: func(_ context.Context, _ uuid.UUID) ([]AdminStudentInfo, error) {
				return []AdminStudentInfo{{DisplayName: "Bob"}}, nil
			},
		},
		&stubSafetyService{getModerationHistoryFn: func(_ context.Context, _ uuid.UUID) ([]ModerationActionSummary, error) {
			return []ModerationActionSummary{{Action: "warn"}}, nil
		}},
		&stubBillingService{getSubscriptionInfoFn: func(_ context.Context, _ uuid.UUID) (*AdminSubscriptionInfo, error) {
			return &AdminSubscriptionInfo{Tier: "premium"}, nil
		}},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	detail, err := svc.GetUserDetail(context.Background(), testAuth(), familyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if detail.Family.Name != "Smith" {
		t.Errorf("expected family name 'Smith', got %q", detail.Family.Name)
	}
	if len(detail.Parents) != 1 || detail.Parents[0].DisplayName != "Alice" {
		t.Errorf("unexpected parents: %+v", detail.Parents)
	}
	if len(detail.Students) != 1 || detail.Students[0].DisplayName != "Bob" {
		t.Errorf("unexpected students: %+v", detail.Students)
	}
	if detail.Subscription == nil || detail.Subscription.Tier != "premium" {
		t.Errorf("unexpected subscription: %+v", detail.Subscription)
	}
	if len(detail.ModerationHistory) != 1 {
		t.Errorf("expected 1 moderation entry, got %d", len(detail.ModerationHistory))
	}
}

func TestGetUserDetail_E4_FamilyNotFoundReturnsErrUserNotFound(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{
			getFamilyDetailFn: func(_ context.Context, _ uuid.UUID) (*AdminFamilyInfo, error) {
				return nil, errors.New("not found")
			},
		},
		&stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.GetUserDetail(context.Background(), testAuth(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("expected ErrUserNotFound, got %v", err)
	}
}

func TestGetUserDetail_E5_BillingErrorIsNonFatal(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{
			getFamilyDetailFn: func(_ context.Context, _ uuid.UUID) (*AdminFamilyInfo, error) {
				return &AdminFamilyInfo{ID: familyID, Name: "Smith"}, nil
			},
			getParentsFn: func(_ context.Context, _ uuid.UUID) ([]AdminParentInfo, error) {
				return []AdminParentInfo{}, nil
			},
			getStudentsFn: func(_ context.Context, _ uuid.UUID) ([]AdminStudentInfo, error) {
				return []AdminStudentInfo{}, nil
			},
		},
		&stubSafetyService{},
		&stubBillingService{getSubscriptionInfoFn: func(_ context.Context, _ uuid.UUID) (*AdminSubscriptionInfo, error) {
			return nil, errors.New("billing unavailable")
		}},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	detail, err := svc.GetUserDetail(context.Background(), testAuth(), familyID)
	if err != nil {
		t.Fatalf("billing error should be non-fatal, got: %v", err)
	}
	if detail.Subscription != nil {
		t.Errorf("expected nil subscription on billing error, got %+v", detail.Subscription)
	}
}

func TestGetUserDetail_E6_SafetyErrorIsNonFatal(t *testing.T) {
	familyID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{
			getFamilyDetailFn: func(_ context.Context, _ uuid.UUID) (*AdminFamilyInfo, error) {
				return &AdminFamilyInfo{ID: familyID, Name: "Smith"}, nil
			},
			getParentsFn: func(_ context.Context, _ uuid.UUID) ([]AdminParentInfo, error) {
				return []AdminParentInfo{}, nil
			},
			getStudentsFn: func(_ context.Context, _ uuid.UUID) ([]AdminStudentInfo, error) {
				return []AdminStudentInfo{}, nil
			},
		},
		&stubSafetyService{getModerationHistoryFn: func(_ context.Context, _ uuid.UUID) ([]ModerationActionSummary, error) {
			return nil, errors.New("safety service down")
		}},
		&stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	detail, err := svc.GetUserDetail(context.Background(), testAuth(), familyID)
	if err != nil {
		t.Fatalf("safety error should be non-fatal, got: %v", err)
	}
	if len(detail.ModerationHistory) != 0 {
		t.Errorf("expected empty moderation history on safety error, got %d entries", len(detail.ModerationHistory))
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area F: System Health (§11)
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetSystemHealth_F1_AllHealthyReturnsHealthy(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l,
		&stubHealthChecker{checkAllFn: func(_ context.Context) []ComponentHealth {
			return []ComponentHealth{
				{Name: "database", Status: "healthy"},
				{Name: "redis", Status: "healthy"},
				{Name: "r2", Status: "healthy"},
				{Name: "kratos", Status: "healthy"},
			}
		}},
		&stubJobInspector{},
	)

	health, err := svc.GetSystemHealth(context.Background(), testAuth())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health.Status != "healthy" {
		t.Errorf("expected overall status 'healthy', got %q", health.Status)
	}
}

func TestGetSystemHealth_F2_OneDegradedReturnsDegraded(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l,
		&stubHealthChecker{checkAllFn: func(_ context.Context) []ComponentHealth {
			return []ComponentHealth{
				{Name: "database", Status: "healthy"},
				{Name: "redis", Status: "degraded"},
				{Name: "r2", Status: "healthy"},
				{Name: "kratos", Status: "healthy"},
			}
		}},
		&stubJobInspector{},
	)

	health, err := svc.GetSystemHealth(context.Background(), testAuth())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health.Status != "degraded" {
		t.Errorf("expected overall status 'degraded', got %q", health.Status)
	}
}

func TestGetSystemHealth_F3_AnyUnhealthyReturnsUnhealthy(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l,
		&stubHealthChecker{checkAllFn: func(_ context.Context) []ComponentHealth {
			return []ComponentHealth{
				{Name: "database", Status: "unhealthy"},
				{Name: "redis", Status: "degraded"},
				{Name: "r2", Status: "healthy"},
				{Name: "kratos", Status: "healthy"},
			}
		}},
		&stubJobInspector{},
	)

	health, err := svc.GetSystemHealth(context.Background(), testAuth())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if health.Status != "unhealthy" {
		t.Errorf("expected overall status 'unhealthy', got %q", health.Status)
	}
}

func TestGetSystemHealth_F4_IncludesAllComponents(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l,
		&stubHealthChecker{checkAllFn: func(_ context.Context) []ComponentHealth {
			return []ComponentHealth{
				{Name: "database", Status: "healthy"},
				{Name: "redis", Status: "healthy"},
				{Name: "r2", Status: "healthy"},
				{Name: "kratos", Status: "healthy"},
			}
		}},
		&stubJobInspector{},
	)

	health, err := svc.GetSystemHealth(context.Background(), testAuth())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(health.Components) != 4 {
		t.Errorf("expected 4 components, got %d", len(health.Components))
	}
	names := make(map[string]bool)
	for _, c := range health.Components {
		names[c.Name] = true
	}
	for _, expected := range []string{"database", "redis", "r2", "kratos"} {
		if !names[expected] {
			t.Errorf("expected component %q in response", expected)
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area G: Dead Letter Queue (§11.2)
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetJobStatus_G1_DelegatesToJobInspector(t *testing.T) {
	expected := &JobStatusResponse{
		Queues:          []QueueStatus{{Name: "default", Pending: 5}},
		DeadLetterCount: 2,
	}

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{},
		&stubJobInspector{getQueueStatusFn: func(_ context.Context) (*JobStatusResponse, error) {
			return expected, nil
		}},
	)

	result, err := svc.GetJobStatus(context.Background(), testAuth())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.DeadLetterCount != 2 {
		t.Errorf("expected dead letter count 2, got %d", result.DeadLetterCount)
	}
}

func TestGetDeadLetterJobs_G2_ReturnsJobsWithPagination(t *testing.T) {
	expected := []DeadLetterJob{
		{ID: "job1", JobType: "notify:send_email"},
		{ID: "job2", JobType: "media:scan_upload"},
	}

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{},
		&stubJobInspector{getDeadLetterJobsFn: func(_ context.Context, _ *shared.PaginationParams) ([]DeadLetterJob, error) {
			return expected, nil
		}},
	)

	result, err := svc.GetDeadLetterJobs(context.Background(), testAuth(), defaultPagination())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(result.Data))
	}
}

func TestRetryDeadLetterJob_G3_SuccessReturnsNil(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{},
		&stubJobInspector{retryDeadLetterJobFn: func(_ context.Context, jobID string) error {
			if jobID != "job1" {
				t.Errorf("expected job ID 'job1', got %q", jobID)
			}
			return nil
		}},
	)

	err := svc.RetryDeadLetterJob(context.Background(), testAuth(), "job1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRetryDeadLetterJob_G4_NotFoundReturnsError(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{},
		&stubJobInspector{retryDeadLetterJobFn: func(_ context.Context, _ string) error {
			return ErrDeadLetterNotFound
		}},
	)

	err := svc.RetryDeadLetterJob(context.Background(), testAuth(), "missing")
	if !errors.Is(err, ErrDeadLetterNotFound) {
		t.Errorf("expected ErrDeadLetterNotFound, got %v", err)
	}
}

func TestRetryDeadLetterJob_G5_LogsAuditEntry(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{},
		&stubJobInspector{retryDeadLetterJobFn: func(_ context.Context, _ string) error {
			return nil
		}},
	)

	err := svc.RetryDeadLetterJob(context.Background(), testAuth(), "job1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil {
		t.Fatal("expected audit entry to be logged")
	}
	if capturedEntry.TargetType != "system" {
		t.Errorf("expected target type 'system', got %q", capturedEntry.TargetType)
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area I: User Actions — suspend/unsuspend/ban (§4)
// ═══════════════════════════════════════════════════════════════════════════════

func TestSuspendUser_I1_DelegatesToSafety(t *testing.T) {
	var capturedFamilyID uuid.UUID
	var capturedReason string
	familyID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{},
		&stubSafetyService{suspendAccountFn: func(_ context.Context, fID uuid.UUID, reason string) error {
			capturedFamilyID = fID
			capturedReason = reason
			return nil
		}},
		&stubBillingService{}, m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.SuspendUser(context.Background(), testAuth(), familyID, "policy violation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedFamilyID != familyID {
		t.Errorf("expected family ID %s, got %s", familyID, capturedFamilyID)
	}
	if capturedReason != "policy violation" {
		t.Errorf("expected reason 'policy violation', got %q", capturedReason)
	}
}

func TestSuspendUser_I2_LogsAuditEntry(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	familyID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.SuspendUser(context.Background(), testAuth(), familyID, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil {
		t.Fatal("expected audit entry")
	}
	if capturedEntry.Action != "user_suspend" {
		t.Errorf("expected action 'user_suspend', got %q", capturedEntry.Action)
	}
}

func TestUnsuspendUser_I3_DelegatesToSafety(t *testing.T) {
	called := false
	familyID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{},
		&stubSafetyService{unsuspendAccountFn: func(_ context.Context, _ uuid.UUID) error {
			called = true
			return nil
		}},
		&stubBillingService{}, m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.UnsuspendUser(context.Background(), testAuth(), familyID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected UnsuspendAccount to be called")
	}
}

func TestUnsuspendUser_I4_LogsAuditEntry(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.UnsuspendUser(context.Background(), testAuth(), uuid.Must(uuid.NewV7()))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil || capturedEntry.Action != "user_unsuspend" {
		t.Errorf("expected audit action 'user_unsuspend', got %+v", capturedEntry)
	}
}

func TestBanUser_I5_DelegatesToSafetyAndLogsAudit(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	familyID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.BanUser(context.Background(), testAuth(), familyID, "TOS violation")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil || capturedEntry.Action != "user_ban" {
		t.Errorf("expected audit action 'user_ban', got %+v", capturedEntry)
	}
}

func TestSuspendUser_I6_SafetyErrorPropagates(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{},
		&stubSafetyService{suspendAccountFn: func(_ context.Context, _ uuid.UUID, _ string) error {
			return errors.New("safety error")
		}},
		&stubBillingService{}, m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.SuspendUser(context.Background(), testAuth(), uuid.Must(uuid.NewV7()), "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area J: Moderation Queue (§4)
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetModerationQueue_J1_DelegatesToSafety(t *testing.T) {
	expected := []ModerationQueueItem{{ID: uuid.Must(uuid.NewV7()), ContentType: "post"}}

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{},
		&stubSafetyService{getReviewQueueFn: func(_ context.Context, _ *shared.PaginationParams) ([]ModerationQueueItem, error) {
			return expected, nil
		}},
		&stubBillingService{}, m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.GetModerationQueue(context.Background(), testAuth(), defaultPagination())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(result.Data))
	}
}

func TestGetModerationQueueItem_J2_ReturnsItem(t *testing.T) {
	itemID := uuid.Must(uuid.NewV7())
	expected := &ModerationQueueItem{ID: itemID, ContentType: "post"}

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{},
		&stubSafetyService{getReviewQueueItemFn: func(_ context.Context, id uuid.UUID) (*ModerationQueueItem, error) {
			if id != itemID {
				t.Errorf("expected item ID %s, got %s", itemID, id)
			}
			return expected, nil
		}},
		&stubBillingService{}, m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	item, err := svc.GetModerationQueueItem(context.Background(), testAuth(), itemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if item.ID != itemID {
		t.Errorf("expected ID %s, got %s", itemID, item.ID)
	}
}

func TestGetModerationQueueItem_J3_NotFoundReturnsError(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{},
		&stubSafetyService{getReviewQueueItemFn: func(_ context.Context, _ uuid.UUID) (*ModerationQueueItem, error) {
			return nil, nil
		}},
		&stubBillingService{}, m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.GetModerationQueueItem(context.Background(), testAuth(), uuid.Must(uuid.NewV7()))
	if !errors.Is(err, ErrModerationItemNotFound) {
		t.Errorf("expected ErrModerationItemNotFound, got %v", err)
	}
}

func TestTakeModerationAction_J4_DelegatesAndLogsAudit(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	itemID := uuid.Must(uuid.NewV7())

	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.TakeModerationAction(context.Background(), testAuth(), itemID, &ModerationActionInput{
		Action: "reject",
		Reason: "inappropriate",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil || capturedEntry.Action != "moderation_action" {
		t.Errorf("expected audit action 'moderation_action', got %+v", capturedEntry)
	}
}

func TestTakeModerationAction_J5_SafetyErrorPropagates(t *testing.T) {
	m, l := ds()
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{},
		&stubSafetyService{takeModerationActionFn: func(_ context.Context, _ uuid.UUID, _ string, _ string) error {
			return errors.New("safety error")
		}},
		&stubBillingService{}, m, l, &stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.TakeModerationAction(context.Background(), testAuth(), uuid.Must(uuid.NewV7()), &ModerationActionInput{
		Action: "approve",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area K: Methodology Config (§4)
// ═══════════════════════════════════════════════════════════════════════════════

func TestListMethodologies_K1_DelegatesToMethodService(t *testing.T) {
	expected := []MethodologyConfig{{Slug: "charlotte-mason", DisplayName: "Charlotte Mason"}}

	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{listMethodologiesFn: func(_ context.Context) ([]MethodologyConfig, error) {
			return expected, nil
		}},
		&stubLifecycleService{}, &stubHealthChecker{}, &stubJobInspector{},
	)

	configs, err := svc.ListMethodologies(context.Background(), testAuth())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 1 || configs[0].Slug != "charlotte-mason" {
		t.Errorf("unexpected configs: %+v", configs)
	}
}

func TestUpdateMethodologyConfig_K2_DelegatesAndLogsAudit(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	expected := &MethodologyConfig{Slug: "classical", Enabled: true}

	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{updateMethodologyConfigFn: func(_ context.Context, slug string, _ *UpdateMethodologyInput) (*MethodologyConfig, error) {
			if slug != "classical" {
				t.Errorf("expected slug 'classical', got %q", slug)
			}
			return expected, nil
		}},
		&stubLifecycleService{}, &stubHealthChecker{}, &stubJobInspector{},
	)

	enabled := true
	config, err := svc.UpdateMethodologyConfig(context.Background(), testAuth(), "classical", &UpdateMethodologyInput{Enabled: &enabled})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if config.Slug != "classical" {
		t.Errorf("expected slug 'classical', got %q", config.Slug)
	}
	if capturedEntry == nil || capturedEntry.Action != "methodology_config_update" {
		t.Errorf("expected audit action 'methodology_config_update', got %+v", capturedEntry)
	}
}

func TestListMethodologies_K3_ErrorPropagates(t *testing.T) {
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{listMethodologiesFn: func(_ context.Context) ([]MethodologyConfig, error) {
			return nil, errors.New("method service error")
		}},
		&stubLifecycleService{}, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.ListMethodologies(context.Background(), testAuth())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUpdateMethodologyConfig_K4_ErrorPropagates(t *testing.T) {
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{updateMethodologyConfigFn: func(_ context.Context, _ string, _ *UpdateMethodologyInput) (*MethodologyConfig, error) {
			return nil, ErrMethodologyNotFound
		}},
		&stubLifecycleService{}, &stubHealthChecker{}, &stubJobInspector{},
	)

	_, err := svc.UpdateMethodologyConfig(context.Background(), testAuth(), "missing", &UpdateMethodologyInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Area L: Lifecycle Management (§4)
// ═══════════════════════════════════════════════════════════════════════════════

func TestGetPendingDeletions_L1_DelegatesToLifecycleService(t *testing.T) {
	expected := []DeletionSummary{{FamilyID: uuid.Must(uuid.NewV7()), FamilyName: "Jones"}}

	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{},
		&stubLifecycleService{getPendingDeletionsFn: func(_ context.Context, _ *shared.PaginationParams) ([]DeletionSummary, error) {
			return expected, nil
		}},
		&stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.GetPendingDeletions(context.Background(), testAuth(), defaultPagination())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].FamilyName != "Jones" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestGetRecoveryRequests_L2_DelegatesToLifecycleService(t *testing.T) {
	expected := []RecoverySummary{{ID: uuid.Must(uuid.NewV7()), Reason: "accidental"}}

	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{},
		&stubLifecycleService{getRecoveryRequestsFn: func(_ context.Context, _ *shared.PaginationParams) ([]RecoverySummary, error) {
			return expected, nil
		}},
		&stubHealthChecker{}, &stubJobInspector{},
	)

	result, err := svc.GetRecoveryRequests(context.Background(), testAuth(), defaultPagination())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Data) != 1 || result.Data[0].Reason != "accidental" {
		t.Errorf("unexpected result: %+v", result)
	}
}

func TestResolveRecoveryRequest_L3_ApprovesAndLogsAudit(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry
	reqID := uuid.Must(uuid.NewV7())

	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{}, &stubLifecycleService{},
		&stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.ResolveRecoveryRequest(context.Background(), testAuth(), reqID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil || capturedEntry.Action != "recovery_approved" {
		t.Errorf("expected audit action 'recovery_approved', got %+v", capturedEntry)
	}
}

func TestResolveRecoveryRequest_L4_DeniesAndLogsAudit(t *testing.T) {
	var capturedEntry *CreateAuditLogEntry

	svc := newTestService(
		&stubFlagRepo{},
		&stubAuditRepo{createFn: func(_ context.Context, entry *CreateAuditLogEntry) (*AuditLogEntry, error) {
			capturedEntry = entry
			return &AuditLogEntry{ID: uuid.Must(uuid.NewV7())}, nil
		}},
		&stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{}, &stubLifecycleService{},
		&stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.ResolveRecoveryRequest(context.Background(), testAuth(), uuid.Must(uuid.NewV7()), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedEntry == nil || capturedEntry.Action != "recovery_denied" {
		t.Errorf("expected audit action 'recovery_denied', got %+v", capturedEntry)
	}
}

func TestResolveRecoveryRequest_L5_LifecycleErrorPropagates(t *testing.T) {
	svc := newTestService(
		&stubFlagRepo{}, &stubAuditRepo{}, &stubCache{},
		&stubIamService{}, &stubSafetyService{}, &stubBillingService{},
		&stubMethodService{},
		&stubLifecycleService{resolveRecoveryRequestFn: func(_ context.Context, _ uuid.UUID, _ bool) error {
			return ErrRecoveryRequestNotFound
		}},
		&stubHealthChecker{}, &stubJobInspector{},
	)

	err := svc.ResolveRecoveryRequest(context.Background(), testAuth(), uuid.Must(uuid.NewV7()), true)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// Test Helpers
// ═══════════════════════════════════════════════════════════════════════════════

func strPtr(s string) *string {
	return &s
}
