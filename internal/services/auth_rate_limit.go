package services

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrAuthRateLimited = errors.New("too many authentication attempts; try again later")

const (
	AuthRateScopeSignInIP      = "sign-in-ip"
	AuthRateScopeSignInAccount = "sign-in-account"
	AuthRateScopeSignUpIP      = "sign-up-ip"
	AuthRateScopeSignUpEmail   = "sign-up-email"
	AuthRateScopeVerifyIP      = "verify-code-ip"
	AuthRateScopeVerifyEmail   = "verify-code-email"
)

// EnforceAuthRateLimit consumes one budget atomically. The unique scope/key
// pair and database upsert make the limiter shared across API replicas.
func EnforceAuthRateLimit(scope, identity string, limit int, window time.Duration) error {
	if db.DB == nil || limit < 1 || window <= 0 {
		return nil
	}
	identity = strings.TrimSpace(strings.ToLower(identity))
	if identity == "" {
		identity = "unknown"
	}
	return enforceAuthRateLimitAt(scope, identity, limit, window, time.Now().UTC(), nil)
}

func enforceAuthRateLimitAt(scope, identity string, limit int, window time.Duration, now time.Time, afterUpsert func()) error {
	expires := now.Add(window)
	keyHash := hashRateLimitIdentity(identity)
	row := &entities.AuthRateLimit{
		ID:              uuid.New(),
		Scope:           scope,
		KeyHash:         keyHash,
		Attempts:        1,
		WindowExpiresAt: expires,
	}

	limited := false
	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "scope"}, {Name: "key_hash"}},
			DoUpdates: clause.Assignments(map[string]any{
				"attempts":          clause.Expr{SQL: "CASE WHEN auth_rate_limits.window_expires_at <= ? THEN 1 WHEN auth_rate_limits.attempts >= ? THEN auth_rate_limits.attempts ELSE auth_rate_limits.attempts + 1 END", Vars: []any{now, limit + 1}},
				"window_expires_at": clause.Expr{SQL: "CASE WHEN auth_rate_limits.window_expires_at <= ? THEN ? ELSE auth_rate_limits.window_expires_at END", Vars: []any{now, expires}},
				"updated_at":        now,
			}),
		}).Create(row)
		if result.Error != nil {
			return result.Error
		}
		if afterUpsert != nil {
			afterUpsert()
		}

		var current entities.AuthRateLimit
		if err := tx.Where("scope = ? AND key_hash = ?", scope, keyHash).First(&current).Error; err != nil {
			return err
		}
		limited = current.Attempts > limit && current.WindowExpiresAt.After(now)
		return nil
	}); err != nil {
		return err
	}
	if limited {
		return ErrAuthRateLimited
	}
	return nil
}

func ResetAuthRateLimit(scope, identity string) error {
	if db.DB == nil {
		return nil
	}
	return db.DB.Where("scope = ? AND key_hash = ?", scope, hashRateLimitIdentity(identity)).Delete(&entities.AuthRateLimit{}).Error
}

func hashRateLimitIdentity(identity string) string {
	digest := sha256.Sum256([]byte(strings.TrimSpace(strings.ToLower(identity))))
	return hex.EncodeToString(digest[:])
}
