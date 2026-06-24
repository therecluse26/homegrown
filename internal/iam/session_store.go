package iam

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ErrSessionNotFound is returned by SessionStore.Get when no session exists for the given sid.
var ErrSessionNotFound = errors.New("session not found")

// ServerSession holds the server-side token data for a single BFF auth session.
// Access and refresh tokens are stored server-side only and MUST NOT be sent to
// the browser. The browser receives only the opaque sid cookie. [ARCH ADR-020, CODING §5.2]
type ServerSession struct {
	SID          string
	HearthUserID uuid.UUID
	FamilyID     uuid.UUID
	AccessToken  string    // decrypted; NEVER log
	RefreshToken string    // decrypted; NEVER log
	ExpiresAt    time.Time // access token expiry
	CreatedAt    time.Time
	LastUsedAt   time.Time
}

// SessionStore is the server-side session repository for the Hearth BFF flow.
// Implementations encrypt tokens at rest (AES-256-GCM) and are safe for concurrent use.
// [ARCH ADR-020]
type SessionStore interface {
	// Create persists a new session and returns the generated sid.
	// The sid is 32 bytes of cryptographic randomness (base64url encoded).
	Create(ctx context.Context, sess CreateServerSession) (sid string, err error)
	// Get retrieves a session by sid. Returns ErrSessionNotFound if absent.
	Get(ctx context.Context, sid string) (*ServerSession, error)
	// UpdateTokens replaces the token pair after a silent refresh (atomic update).
	UpdateTokens(ctx context.Context, sid, accessToken, refreshToken string, expiresAt time.Time) error
	// Delete removes a session on logout.
	Delete(ctx context.Context, sid string) error
	// DeleteByFamily revokes all sessions for a family (account lock, family deletion).
	DeleteByFamily(ctx context.Context, familyID uuid.UUID) error
}

// CreateServerSession is the input to SessionStore.Create.
type CreateServerSession struct {
	HearthUserID uuid.UUID
	FamilyID     uuid.UUID
	AccessToken  string    // plain text; encrypted before storage
	RefreshToken string    // plain text; encrypted before storage; NEVER log
	ExpiresAt    time.Time
}

// ── Postgres implementation ───────────────────────────────────────────────────

// iamSessionRow is the GORM model for the iam_sessions table. [§18.2]
type iamSessionRow struct {
	SID           string    `gorm:"column:sid;primaryKey"`
	HearthUserID  uuid.UUID `gorm:"column:hearth_user_id;type:uuid;not null"`
	FamilyID      uuid.UUID `gorm:"column:family_id;type:uuid;not null"`
	AccessToken   string    `gorm:"column:access_token;not null"`  // AES-GCM encrypted
	RefreshToken  string    `gorm:"column:refresh_token;not null"` // AES-GCM encrypted
	TokenExpiresAt time.Time `gorm:"column:token_expires_at;not null"`
	CreatedAt     time.Time `gorm:"column:created_at;autoCreateTime"`
	LastUsedAt    time.Time `gorm:"column:last_used_at"`
}

func (iamSessionRow) TableName() string { return "iam_sessions" }

// PgSessionStore is a PostgreSQL-backed SessionStore with AES-256-GCM token encryption.
type PgSessionStore struct {
	db  *gorm.DB
	key []byte // 32-byte AES-256 key
}

// NewPgSessionStore creates a PgSessionStore.
// key must be exactly 32 bytes (AES-256). Derived from config.HearthSessionEncryptionKey.
func NewPgSessionStore(db *gorm.DB, key []byte) (*PgSessionStore, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("session store: encryption key must be 32 bytes, got %d", len(key))
	}
	return &PgSessionStore{db: db, key: key}, nil
}

// generateSID returns 32 bytes of cryptographic randomness as a base64url string.
func generateSID() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("session store: failed to generate sid: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// encrypt encrypts plaintext with AES-256-GCM. Returns base64url ciphertext.
func (s *PgSessionStore) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a base64url ciphertext encrypted with AES-256-GCM.
func (s *PgSessionStore) decrypt(ciphertext string) (string, error) {
	data, err := base64.RawURLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("session decrypt: base64: %w", err)
	}
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("session decrypt: ciphertext too short")
	}
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("session decrypt: %w", err)
	}
	return string(plaintext), nil
}

func (s *PgSessionStore) Create(ctx context.Context, sess CreateServerSession) (string, error) {
	sid, err := generateSID()
	if err != nil {
		return "", err
	}
	encAccess, err := s.encrypt(sess.AccessToken)
	if err != nil {
		return "", fmt.Errorf("session store create: %w", err)
	}
	encRefresh, err := s.encrypt(sess.RefreshToken)
	if err != nil {
		return "", fmt.Errorf("session store create: %w", err)
	}
	row := iamSessionRow{
		SID:           sid,
		HearthUserID:  sess.HearthUserID,
		FamilyID:      sess.FamilyID,
		AccessToken:   encAccess,
		RefreshToken:  encRefresh,
		TokenExpiresAt: sess.ExpiresAt,
		LastUsedAt:    time.Now().UTC(),
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return "", fmt.Errorf("session store create: %w", err)
	}
	return sid, nil
}

func (s *PgSessionStore) Get(ctx context.Context, sid string) (*ServerSession, error) {
	var row iamSessionRow
	err := s.db.WithContext(ctx).First(&row, "sid = ?", sid).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSessionNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("session store get: %w", err)
	}
	accessToken, err := s.decrypt(row.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("session store get: %w", err)
	}
	refreshToken, err := s.decrypt(row.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("session store get: %w", err)
	}
	// Update last_used_at async — fire and forget, best-effort.
	go func() {
		_ = s.db.Model(&iamSessionRow{}).Where("sid = ?", sid).
			Update("last_used_at", time.Now().UTC()).Error
	}()
	return &ServerSession{
		SID:          row.SID,
		HearthUserID: row.HearthUserID,
		FamilyID:     row.FamilyID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    row.TokenExpiresAt,
		CreatedAt:    row.CreatedAt,
		LastUsedAt:   row.LastUsedAt,
	}, nil
}

func (s *PgSessionStore) UpdateTokens(ctx context.Context, sid, accessToken, refreshToken string, expiresAt time.Time) error {
	encAccess, err := s.encrypt(accessToken)
	if err != nil {
		return fmt.Errorf("session store update: %w", err)
	}
	encRefresh, err := s.encrypt(refreshToken)
	if err != nil {
		return fmt.Errorf("session store update: %w", err)
	}
	result := s.db.WithContext(ctx).Model(&iamSessionRow{}).
		Where("sid = ?", sid).
		Updates(map[string]any{
			"access_token":    encAccess,
			"refresh_token":   encRefresh,
			"token_expires_at": expiresAt,
			"last_used_at":    time.Now().UTC(),
		})
	if result.Error != nil {
		return fmt.Errorf("session store update: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *PgSessionStore) Delete(ctx context.Context, sid string) error {
	return s.db.WithContext(ctx).Delete(&iamSessionRow{}, "sid = ?", sid).Error
}

func (s *PgSessionStore) DeleteByFamily(ctx context.Context, familyID uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&iamSessionRow{}, "family_id = ?", familyID).Error
}
