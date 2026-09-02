package entities

import "time"

// AuthRateLimit stores a shared, database-backed authentication budget. The
// identity is stored as a hash so usernames, email addresses, and IPs are not
// retained in plaintext in the limiter table.
type AuthRateLimit struct {
	ID              string    `gorm:"type:varchar(36);primaryKey"`
	Scope           string    `gorm:"type:varchar(64);not null;uniqueIndex:idx_auth_rate_limits_scope_key"`
	KeyHash         string    `gorm:"type:char(64);not null;uniqueIndex:idx_auth_rate_limits_scope_key"`
	Attempts        int       `gorm:"type:int;not null;default:0"`
	WindowExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`
}
