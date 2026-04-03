package services

import (
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupSignupSecurityTestDB(t *testing.T) {
	t.Helper()

	originalDB := db.DB
	originalConfig := app.Config
	originalMailer := signupVerificationMailer
	originalCodeGenerator := signupVerificationCodeGenerator
	originalNow := currentTime
	t.Cleanup(func() {
		db.DB = originalDB
		app.Config = originalConfig
		signupVerificationMailer = originalMailer
		signupVerificationCodeGenerator = originalCodeGenerator
		currentTime = originalNow
	})

	testDB, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.User{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.SystemSetting{},
		&entities.SignupVerificationCode{},
	))

	db.DB = testDB
	app.Config.JWTSecret = "signup-security-test-secret"
	app.Config.JWTIssuer = "ketches.test"
	app.Config.JWTAudience = "ketches-ui"
}

func TestRequestSignUpVerificationCodeEnforcesResendCooldown(t *testing.T) {
	setupSignupSecurityTestDB(t)

	var sentCodes []string
	signupVerificationMailer = func(email, code string) error {
		sentCodes = append(sentCodes, code)
		return nil
	}
	signupVerificationCodeGenerator = func() string { return "123456" }
	currentTime = func() time.Time {
		return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	}

	result, err := RequestSignUpVerificationCode("alice@example.com")
	require.NoError(t, err)
	assert.Equal(t, 60, result.ResendAfterSeconds)
	require.Equal(t, []string{"123456"}, sentCodes)

	_, err = RequestSignUpVerificationCode("alice@example.com")
	require.ErrorIs(t, err, ErrVerificationCodeResendTooSoon)
}

func TestSignUpRequiresValidVerificationCodeWhenPublicRegistrationEnabled(t *testing.T) {
	setupSignupSecurityTestDB(t)

	signupVerificationMailer = func(email, code string) error { return nil }
	signupVerificationCodeGenerator = func() string { return "123456" }
	currentTime = func() time.Time {
		return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	}

	_, err := RequestSignUpVerificationCode("alice@example.com")
	require.NoError(t, err)

	_, err = SignUp(&models.SignUpRequest{
		Username:         "alice",
		Email:            "alice@example.com",
		Password:         "Password#123",
		Fullname:         "Alice Example",
		VerificationCode: "000000",
	})
	require.ErrorIs(t, err, ErrInvalidVerificationCode)
}

func TestSignUpCreatesUserAfterSuccessfulVerification(t *testing.T) {
	setupSignupSecurityTestDB(t)

	signupVerificationMailer = func(email, code string) error { return nil }
	signupVerificationCodeGenerator = func() string { return "123456" }
	currentTime = func() time.Time {
		return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	}

	_, err := RequestSignUpVerificationCode("alice@example.com")
	require.NoError(t, err)

	user, err := SignUp(&models.SignUpRequest{
		Username:         "alice",
		Email:            "alice@example.com",
		Password:         "Password#123",
		Fullname:         "Alice Example",
		VerificationCode: "123456",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, app.UserRoleUser, user.Role)

	var codeCount int64
	require.NoError(t, db.DB.Model(&entities.SignupVerificationCode{}).Where("email = ?", "alice@example.com").Count(&codeCount).Error)
	assert.Equal(t, int64(0), codeCount)
}

func TestSignUpRejectsWhenPublicRegistrationIsDisabled(t *testing.T) {
	setupSignupSecurityTestDB(t)

	require.NoError(t, UpdatePublicSignUpEnabled(false))

	_, err := SignUp(&models.SignUpRequest{
		Username:         "alice",
		Email:            "alice@example.com",
		Password:         "Password#123",
		Fullname:         "Alice Example",
		VerificationCode: "123456",
	})
	require.ErrorIs(t, err, ErrPublicSignUpDisabled)
}
