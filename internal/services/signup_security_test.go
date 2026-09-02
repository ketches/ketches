package services

import (
	"errors"
	"sync"
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

	testDB, err := gorm.Open(sqlite.Open(t.TempDir()+"/signup-security.db?_pragma=busy_timeout(5000)"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	require.NoError(t, testDB.AutoMigrate(
		&entities.User{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.SystemSetting{},
		&entities.SignupVerificationCode{},
		&entities.AuthRateLimit{},
	))

	db.DB = testDB
	app.Config.JWTSecret = "signup-security-test-secret"
	app.Config.JWTIssuer = "ketches.test"
	app.Config.JWTAudience = "ketches-ui"
	app.Config.SignUpEmailVerificationRequired = true
}

func TestRequestSignUpVerificationCodeEnforcesResendCooldown(t *testing.T) {
	setupSignupSecurityTestDB(t)

	var sentCodes []string
	signupVerificationMailer = func(email, code string) error {
		sentCodes = append(sentCodes, code)
		return nil
	}
	signupVerificationCodeGenerator = func() (string, error) { return "123456", nil }
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
	signupVerificationCodeGenerator = func() (string, error) { return "123456", nil }
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
	signupVerificationCodeGenerator = func() (string, error) { return "123456", nil }
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

func TestSignUpKeepsValidVerificationCodeWhenAccountCreationFails(t *testing.T) {
	setupSignupSecurityTestDB(t)

	signupVerificationMailer = func(email, code string) error { return nil }
	signupVerificationCodeGenerator = func() (string, error) { return "123456", nil }
	currentTime = func() time.Time {
		return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	}

	require.NoError(t, db.DB.Create(&entities.User{
		Base:     entities.Base{ID: "existing-user"},
		Username: "alice",
		Email:    "existing@example.com",
		Password: "unused-password-hash",
		Role:     app.UserRoleUser,
	}).Error)
	require.NoError(t, func() error {
		_, err := RequestSignUpVerificationCode("new@example.com")
		return err
	}())

	_, err := SignUp(&models.SignUpRequest{
		Username:         "alice",
		Email:            "new@example.com",
		Password:         "Password#123",
		Fullname:         "New User",
		VerificationCode: "123456",
	})
	require.ErrorIs(t, err, ErrUsernameAlreadyExists)

	var codeCount int64
	require.NoError(t, db.DB.Model(&entities.SignupVerificationCode{}).
		Where("email = ?", "new@example.com").Count(&codeCount).Error)
	assert.Equal(t, int64(1), codeCount)

	user, err := SignUp(&models.SignUpRequest{
		Username:         "new-user",
		Email:            "new@example.com",
		Password:         "Password#123",
		Fullname:         "New User",
		VerificationCode: "123456",
	})
	require.NoError(t, err)
	assert.Equal(t, "new-user", user.Username)
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

func TestSignUpSkipsVerificationWhenEmailVerificationDisabled(t *testing.T) {
	setupSignupSecurityTestDB(t)

	app.Config.SignUpEmailVerificationRequired = false

	user, err := SignUp(&models.SignUpRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Password: "Password#123",
		Fullname: "Alice Example",
	})
	require.NoError(t, err)
	assert.Equal(t, "alice", user.Username)

	var codeCount int64
	require.NoError(t, db.DB.Model(&entities.SignupVerificationCode{}).Count(&codeCount).Error)
	assert.Equal(t, int64(0), codeCount)
}

func TestRequestSignUpVerificationCodeRejectsWhenEmailVerificationDisabled(t *testing.T) {
	setupSignupSecurityTestDB(t)

	app.Config.SignUpEmailVerificationRequired = false

	_, err := RequestSignUpVerificationCode("alice@example.com")
	require.ErrorIs(t, err, ErrSignUpEmailVerificationDisabled)
}

func TestRequestSignUpVerificationCodeReturnsRandomSourceFailure(t *testing.T) {
	setupSignupSecurityTestDB(t)
	expected := errors.New("random source unavailable")
	signupVerificationCodeGenerator = func() (string, error) { return "", expected }
	signupVerificationMailer = func(email, code string) error {
		t.Fatal("mailer must not run when code generation fails")
		return nil
	}

	_, err := RequestSignUpVerificationCode("alice@example.com")
	require.ErrorIs(t, err, expected)

	signupVerificationCodeGenerator = func() (string, error) { return "123456", nil }
	signupVerificationMailer = func(email, code string) error { return nil }
	_, err = RequestSignUpVerificationCode("alice@example.com")
	require.NoError(t, err, "failed generation must not consume the resend budget")
}

func TestSignupVerificationCodeExpiresAfterFailedAttemptLimit(t *testing.T) {
	setupSignupSecurityTestDB(t)
	signupVerificationCodeGenerator = func() (string, error) { return "123456", nil }
	signupVerificationMailer = func(email, code string) error { return nil }

	_, err := RequestSignUpVerificationCode("alice@example.com")
	require.NoError(t, err)
	for range signupVerificationCodeMaxAttempts {
		require.ErrorIs(t, consumeSignupVerificationCodeAtomically("alice@example.com", "000000"), ErrInvalidVerificationCode)
	}
	require.ErrorIs(t, consumeSignupVerificationCodeAtomically("alice@example.com", "123456"), ErrInvalidVerificationCode)

	var count int64
	require.NoError(t, db.DB.Model(&entities.SignupVerificationCode{}).Where("email = ?", "alice@example.com").Count(&count).Error)
	assert.Zero(t, count)
}

func TestAuthRateLimitIsResettable(t *testing.T) {
	setupSignupSecurityTestDB(t)
	const identity = "alice"
	require.NoError(t, EnforceAuthRateLimit(AuthRateScopeSignInAccount, identity, 2, time.Minute))
	require.NoError(t, EnforceAuthRateLimit(AuthRateScopeSignInAccount, identity, 2, time.Minute))
	require.ErrorIs(t, EnforceAuthRateLimit(AuthRateScopeSignInAccount, identity, 2, time.Minute), ErrAuthRateLimited)
	require.NoError(t, ResetAuthRateLimit(AuthRateScopeSignInAccount, identity))
	require.NoError(t, EnforceAuthRateLimit(AuthRateScopeSignInAccount, identity, 2, time.Minute))
}

func TestAuthRateLimitConcurrentConsumersUseTheirOwnAtomicCount(t *testing.T) {
	setupSignupSecurityTestDB(t)

	sqlDB, err := db.DB.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(2)

	firstUpserted := make(chan struct{})
	releaseFirst := make(chan struct{})
	var closeOnce sync.Once
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

	firstResult := make(chan error, 1)
	go func() {
		firstResult <- enforceAuthRateLimitAt(
			AuthRateScopeSignInAccount,
			"concurrent-alice",
			1,
			time.Minute,
			now,
			func() {
				closeOnce.Do(func() { close(firstUpserted) })
				<-releaseFirst
			},
		)
	}()

	select {
	case <-firstUpserted:
	case <-time.After(5 * time.Second):
		t.Fatal("first rate-limit transaction did not reach its atomic read")
	}

	secondResult := make(chan error, 1)
	go func() {
		secondResult <- enforceAuthRateLimitAt(
			AuthRateScopeSignInAccount,
			"concurrent-alice",
			1,
			time.Minute,
			now,
			nil,
		)
	}()

	select {
	case err := <-secondResult:
		t.Fatalf("second consumer completed before the first transaction released its row lock: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	close(releaseFirst)
	require.NoError(t, <-firstResult)
	require.ErrorIs(t, <-secondResult, ErrAuthRateLimited)
}
