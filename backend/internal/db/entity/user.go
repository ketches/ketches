package entity

type User struct {
	UUIDBase
	Username string `json:"username" gorm:"not null;uniqueIndex;size:32"` // Unique username for the user
	Password string `json:"password" gorm:"size:255;not null"`            // Hashed password for the user
	Email    string `json:"email" gorm:"not null;uniqueIndex;size:255"`   // User's email address, must be unique
	Role     string `json:"role" gorm:"size:32;not null;default:'user'"`  // user, admin, etc.
	Fullname string `json:"fullname" gorm:"size:255;not null"`            // User's full name
	Gender   int8   `json:"gender"`                                       // 0: female, 1: male, 2: other
	Phone    string `json:"phone"`                                        // User's phone number, optional
	AuditBase
}

type UserToken struct {
	UUIDBase
	UserID    string `json:"user_id" gorm:"not null;index;size:64"`      // User UUID this token belongs to
	Token     string `json:"token" gorm:"not null;uniqueIndex;size:512"` // Unique token for the user session
	TokenType string `json:"token_type" gorm:"not null;size:32"`         // access_token, refresh_token
	ExpiresAt int64  `json:"expires_at"`                                 // Expiration timestamp for the token
	AuditBase
}
