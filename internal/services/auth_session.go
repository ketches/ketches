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
	signupVerificationCodeLength         = 6
	signupVerificationCodeExpiresIn      = 5 * time.Minute
	signupVerificationCodeResendCooldown = 60 * time.Second
)

var (
	ErrAccountLocked                 = errors.New("account is locked")
	ErrPublicSignUpDisabled          = errors.New("public registration is disabled")
	ErrInvalidVerificationCode       = errors.New("verification code is invalid or expired")
	ErrVerificationCodeResendTooSoon = errors.New("verification code was sent too recently")
	ErrInvalidRefreshToken           = errors.New("refresh token is invalid")
	ErrEmailDeliveryNotConfigured    = errors.New("email delivery is not configured")
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

func RequestSignUpVerificationCode(email string) (*models.SignUpVerificationCodeResponse, error) {
	enabled, err := GetPublicSignUpEnabled()
	if err != nil {
		return nil, err
	}
	if !enabled {
		return nil, ErrPublicSignUpDisabled
	}

	normalizedEmail, err := normalizeEmailAddress(email)
	if err != nil {
		return nil, err
	}

	now := currentTime()
	var latest entities.SignupVerificationCode
	if err := db.DB.Where("email = ?", normalizedEmail).Order("created_at DESC").First(&latest).Error; err == nil {
		if latest.ResendAvailableAt.After(now) {
			return nil, ErrVerificationCodeResendTooSoon
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	code := signupVerificationCodeGenerator()
	record := entities.SignupVerificationCode{
		Base:              entities.Base{ID: uuid.New()},
		Email:             normalizedEmail,
		CodeHash:          hashOpaqueValue(code),
		ExpiresAt:         now.Add(signupVerificationCodeExpiresIn),
		ResendAvailableAt: now.Add(signupVerificationCodeResendCooldown),
	}

	if err := db.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("email = ?", normalizedEmail).Delete(&entities.SignupVerificationCode{}).Error; err != nil {
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

	var latest entities.SignupVerificationCode
	if err := tx.Where("email = ?", normalizedEmail).Order("created_at DESC").First(&latest).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidVerificationCode
		}
		return err
	}

	now := currentTime()
	if latest.ExpiresAt.Before(now) || !secureCompareHash(latest.CodeHash, strings.TrimSpace(code)) {
		return ErrInvalidVerificationCode
	}

	return tx.Where("email = ?", normalizedEmail).Delete(&entities.SignupVerificationCode{}).Error
}

func generateSignupVerificationCode() string {
	buf := make([]byte, signupVerificationCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "000000"
	}

	for i := range buf {
		buf[i] = '0' + (buf[i] % 10)
	}
	return string(buf)
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
