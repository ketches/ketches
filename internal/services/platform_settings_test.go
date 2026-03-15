package services

import (
	"testing"

	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformBrandingDefaults(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	branding, err := GetPlatformBranding()
	require.NoError(t, err)
	assert.Equal(t, "Ketches Admin", branding.Name)
}

func TestUpdatePlatformBrandingPersistsNameOnly(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	branding, err := UpdatePlatformBranding(&models.UpdatePlatformBrandingRequest{
		Name: "Second Brand",
	}, nil)
	require.NoError(t, err)
	assert.Equal(t, "Second Brand", branding.Name)

	var setting entities.SystemSetting
	require.NoError(t, db.DB.Where("key = ?", "platform_branding").First(&setting).Error)
	assert.JSONEq(t, `{"name":"Second Brand"}`, setting.Value)
}

func TestUpdatePlatformBrandingRejectsEmptyName(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	_, err := UpdatePlatformBranding(&models.UpdatePlatformBrandingRequest{
		Name: "   ",
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "platform name is required")
}

func TestGetPlatformBrandingIgnoresLegacyLogoFields(t *testing.T) {
	setupPlatformUpdateServiceTestDB(t)

	require.NoError(t, db.DB.Create(&entities.SystemSetting{
		Base: entities.Base{ID: uuid.New()},
		Key:  "platform_branding",
		Value: `{
			"name":"Legacy Brand",
			"logo_path":"branding/logo.png",
			"logo_content_type":"image/png"
		}`,
	}).Error)

	branding, err := GetPlatformBranding()
	require.NoError(t, err)
	assert.Equal(t, "Legacy Brand", branding.Name)
}
