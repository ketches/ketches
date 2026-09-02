package services

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"time"

	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"gorm.io/gorm"
)

const (
	publicSignUpSettingKey               = "public_sign_up_enabled"
	signUpEmailVerificationSettingKey    = "sign_up_email_verification_required"
	signupVerificationCodeLength         = 6
	signupVerificationCodeExpiresIn      = 5 * time.Minute
	signupVerificationCodeResendCooldown = 60 * time.Second
	signupVerificationCodeMaxAttempts    = 5
)

var (
	ErrAccountLocked                   = errors.New("account is locked")
	ErrPublicSignUpDisabled            = errors.New("public registration is disabled")
	ErrSignUpEmailVerificationDisabled = errors.New("email verification is disabled")
	ErrInvalidVerificationCode         = errors.New("verification code is invalid or expired")
	ErrVerificationCodeResendTooSoon   = errors.New("verification code was sent too recently")
	ErrInvalidRefreshToken             = errors.New("refresh token is invalid")
	ErrEmailDeliveryNotConfigured      = errors.New("email delivery is not configured")
)

var (
	currentTime                     = time.Now
	signupVerificationCodeGenerator = generateSignupVerificationCode
	signupVerificationMailer        = sendSignupVerificationEmail
)

func GetPublicSignUpEnabled() (bool, error) {
	setting, err := getSystemSetting(publicSignUpSettingKey)
	if err != nil {
		return false, err
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return true, nil
	}
	return strings.EqualFold(strings.TrimSpace(setting.Value), "true"), nil
}

func UpdatePublicSignUpEnabled(enabled bool) error {
	value := "false"
	if enabled {
		value = "true"
	}

	setting, err := getSystemSetting(publicSignUpSettingKey)
	if err != nil {
		return err
	}
	if setting == nil {
		return db.DB.Create(&entities.SystemSetting{
			Base:  entities.Base{ID: uuid.New()},
			Key:   publicSignUpSettingKey,
			Value: value,
		}).Error
	}

	return db.DB.Model(&entities.SystemSetting{}).Where("id = ?", setting.ID).Update("value", value).Error
}

func GetSignUpEmailVerificationRequired() (bool, error) {
	setting, err := getSystemSetting(signUpEmailVerificationSettingKey)
	if err != nil {
		return false, err
	}
	if setting == nil || strings.TrimSpace(setting.Value) == "" {
		return app.Config.SignUpEmailVerificationRequired, nil
	}
	return strings.EqualFold(strings.TrimSpace(setting.Value), "true"), nil
}

func UpdateSignUpEmailVerificationRequired(required bool) error {
	value := "false"
	if required {
		value = "true"
	}

	setting, err := getSystemSetting(signUpEmailVerificationSettingKey)
	if err != nil {
		return err
	}
	if setting == nil {
		return db.DB.Create(&entities.SystemSetting{
			Base:  entities.Base{ID: uuid.New()},
			Key:   signUpEmailVerificationSettingKey,
			Value: value,
		}).Error
	}

	return db.DB.Model(&entities.SystemSetting{}).Where("id = ?", setting.ID).Update("value", value).Error
}

func RequestSignUpVerificationCode(email string) (*models.SignUpVerificationCodeResponse, error) {
	enabled, err := GetPublicSignUpEnabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrPublicSignUpDisabled
	}
	verificationRequired, err := GetSignUpEmailVerificationRequired()
	if err != nil {
		return nil, err
	}
	if !verificationRequired {
		return nil, ErrSignUpEmailVerificationDisabled
	}

	normalizedEmail, err := normalizeEmailAddress(email)
	if err != nil {
		return nil, err
	}

	if err := EnforceAuthRateLimit(AuthRateScopeVerifyEmail, normalizedEmail, 1, signupVerificationCodeResendCooldown); err != nil {
		if errors.Is(err, ErrAuthRateLimited) {
			return nil, ErrVerificationCodeResendTooSoon
		}
		return nil, err
	}

	now := currentTime()
	code, err := signupVerificationCodeGenerator()
	if err != nil {
		_ = ResetAuthRateLimit(AuthRateScopeVerifyEmail, normalizedEmail)
		return nil, err
	}
	record := entities.SignupVerificationCode{
		Base:              entities.Base{ID: uuid.New()},
		Email:             normalizedEmail,
		CodeHash:          hashOpaqueValue(code),
		ExpiresAt:         now.Add(signupVerificationCodeExpiresIn),
		ResendAvailableAt: now.Add(signupVerificationCodeResendCooldown),
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		var latest entities.SignupVerificationCode
		if err := tx.Unscoped().Where("email = ?", normalizedEmail).First(&latest).Error; err == nil {
			if latest.ResendAvailableAt.After(now) {
				return ErrVerificationCodeResendTooSoon
			}
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Unscoped().Where("email = ?", normalizedEmail).Delete(&entities.SignupVerificationCode{}).Error; err != nil {
			return err
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		if err := signupVerificationMailer(normalizedEmail, code); err != nil {
			return err
		}
		return nil
	}); err != nil {
		_ = ResetAuthRateLimit(AuthRateScopeVerifyEmail, normalizedEmail)
		return nil, err
	}

	return &models.SignUpVerificationCodeResponse{
		ExpiresInSeconds:   int(signupVerificationCodeExpiresIn / time.Second),
		ResendAfterSeconds: int(signupVerificationCodeResendCooldown / time.Second),
	}, nil
}

func StoreRefreshToken(userID, refreshToken string) error {
	refreshToken = strings.TrimSpace(refreshToken)
	if userID == "" || refreshToken == "" {
		return ErrInvalidRefreshToken
	}
	return db.DB.Model(&entities.User{}).Where("id = ?", userID).Update("refresh_token", hashOpaqueValue(refreshToken)).Error
}

func ClearRefreshToken(userID string) error {
	return db.DB.Model(&entities.User{}).Where("id = ?", userID).Update("refresh_token", "").Error
}

func VerifyRefreshToken(user *entities.User, refreshToken string) error {
	if user == nil || strings.TrimSpace(refreshToken) == "" {
		return ErrInvalidRefreshToken
	}
	if user.IsLocked {
		return ErrAccountLocked
	}
	if strings.TrimSpace(user.RefreshToken) == "" {
		return ErrInvalidRefreshToken
	}
	if !secureCompareHash(user.RefreshToken, refreshToken) {
		return ErrInvalidRefreshToken
	}
	return nil
}

func GenerateCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func normalizeEmailAddress(email string) (string, error) {
	address, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return "", errors.New("invalid email address")
	}
	return strings.ToLower(address.Address), nil
}

func consumeSignupVerificationCode(tx *gorm.DB, email, code string) error {
	normalizedEmail, err := normalizeEmailAddress(email)
	if err != nil {
		return err
	}

	now := currentTime()
	codeHash := hashOpaqueValue(strings.TrimSpace(code))
	result := tx.Unscoped().Where(
		"email = ? AND code_hash = ? AND expires_at > ? AND attempt_count < ?",
		normalizedEmail, codeHash, now, signupVerificationCodeMaxAttempts,
	).Delete(&entities.SignupVerificationCode{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}

	if err := tx.Model(&entities.SignupVerificationCode{}).
		Where("email = ? AND expires_at > ? AND attempt_count < ?", normalizedEmail, now, signupVerificationCodeMaxAttempts).
		UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1")).Error; err != nil {
		return err
	}
	if err := tx.Unscoped().Where("email = ? AND (expires_at <= ? OR attempt_count >= ?)", normalizedEmail, now, signupVerificationCodeMaxAttempts).
		Delete(&entities.SignupVerificationCode{}).Error; err != nil {
		return err
	}
	return ErrInvalidVerificationCode
}

func consumeSignupVerificationCodeAtomically(email, code string) error {
	invalid := false
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		err := consumeSignupVerificationCode(tx, email, code)
		if errors.Is(err, ErrInvalidVerificationCode) {
			invalid = true
			return nil
		}
		return err
	})
	if err != nil {
		return err
	}
	if invalid {
		return ErrInvalidVerificationCode
	}
	return nil
}

func generateSignupVerificationCode() (string, error) {
	buf := make([]byte, signupVerificationCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	for i := range buf {
		buf[i] = '0' + (buf[i] % 10)
	}
	return string(buf), nil
}

func sendSignupVerificationEmail(email, code string) error {
	if strings.TrimSpace(app.Config.SMTPHost) == "" || strings.TrimSpace(app.Config.SMTPFrom) == "" {
		return ErrEmailDeliveryNotConfigured
	}

	addr := fmt.Sprintf("%s:%d", app.Config.SMTPHost, app.Config.SMTPPort)
	auth := smtp.PlainAuth("", app.Config.SMTPUsername, app.Config.SMTPPassword, app.Config.SMTPHost)
	message := []byte(fmt.Sprintf("To: %s\r\nSubject: Ketches sign-up verification code\r\n\r\nYour verification code is %s. It expires in 300 seconds.\r\n", email, code))

	if err := smtp.SendMail(addr, auth, app.Config.SMTPFrom, []string{email}, message); err != nil {
		return app.WrapErrorf(err, "send signup verification email: %w", err)
	}
	return nil
}

func hashOpaqueValue(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func secureCompareHash(storedHash, value string) bool {
	computedHash := hashOpaqueValue(value)
	return subtle.ConstantTimeCompare([]byte(storedHash), []byte(computedHash)) == 1
}
