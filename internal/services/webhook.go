package services

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
)

func HandleGitWebhook(c *gin.Context, appID, secret string) error {
	// Get build config for this app
	var config entities.AppBuildConfig
	if err := db.DB.Where("app_id = ?", appID).First(&config).Error; err != nil {
		return fmt.Errorf("build config not found for app %s", appID)
	}

	if !config.WebhookEnabled {
		return errors.New("webhook is not enabled for this app")
	}

	// Verify webhook secret
	if secret != "" {
		if secret != config.WebhookSecret {
			return errors.New("invalid webhook secret")
		}
	} else {
		// Try to verify via signature headers (GitHub, GitLab, Bitbucket)
		if err := verifyWebhookSignature(c, config.WebhookSecret); err != nil {
			return fmt.Errorf("webhook signature verification failed: %w", err)
		}
	}

	// Trigger a build
	triggerReq := &models.TriggerBuildRequest{
		GitRef: config.GitRef,
	}

	build, err := TriggerBuild(c.Request.Context(), appID, "", triggerReq)
	if err != nil {
		return fmt.Errorf("failed to trigger build: %w", err)
	}

	// Update trigger type to webhook
	build.TriggerType = entities.BuildTriggerWebhook
	db.DB.Model(build).Update("trigger_type", entities.BuildTriggerWebhook)

	log.Printf("Webhook triggered build #%d for app %s", build.BuildNumber, appID)
	return nil
}

// HandleGitWebhookForCodeRepo handles webhook for a code repository. Triggers a build for each build config that has WebhookEnabled.
func HandleGitWebhookForCodeRepo(c *gin.Context, repoID, secret string) error {
	repo, err := GetCodeRepository(repoID)
	if err != nil {
		return fmt.Errorf("code repository not found: %w", err)
	}
	if !repo.WebhookEnabled {
		return errors.New("webhook is not enabled for this code repository")
	}
	if secret != "" {
		if secret != repo.WebhookSecret {
			return errors.New("invalid webhook secret")
		}
	} else {
		if err := verifyWebhookSignature(c, repo.WebhookSecret); err != nil {
			return fmt.Errorf("webhook signature verification failed: %w", err)
		}
	}

	configs, err := ListCodeRepositoryBuildConfigs(repoID)
	if err != nil {
		return fmt.Errorf("failed to list build configs: %w", err)
	}
	var toTrigger []CodeRepositoryBuildConfigWithRegistry
	for i := range configs {
		if configs[i].WebhookEnabled {
			toTrigger = append(toTrigger, configs[i])
		}
	}
	if len(toTrigger) == 0 {
		log.Printf("Webhook for repo %s: no build configs with webhook enabled", repoID)
		return nil
	}

	buildEnv, err := GetProjectBuildEnv(repo.ProjectID)
	if err != nil {
		return fmt.Errorf("no build environment configured for this project: %w", err)
	}

	for _, cfg := range toTrigger {
		build, err := TriggerCodeRepositoryBuild(repoID, "", &models.TriggerCodeRepositoryBuildRequest{
			BuildConfigID: cfg.ID,
			BuildEnvID:    buildEnv.ID,
		})
		if err != nil {
			log.Printf("Webhook: failed to trigger build for config %s: %v", cfg.Name, err)
			continue
		}
		build.TriggerType = entities.BuildTriggerWebhook
		db.DB.Model(build).Update("trigger_type", entities.BuildTriggerWebhook)
		log.Printf("Webhook triggered build #%d for code repository %s (config %s)", build.BuildNumber, repoID, cfg.Name)
		_ = build
	}
	return nil
}

func verifyWebhookSignature(c *gin.Context, secret string) error {
	// GitHub: X-Hub-Signature-256
	githubSig := c.GetHeader("X-Hub-Signature-256")
	if githubSig != "" {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			return err
		}
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))
		if !hmac.Equal([]byte(githubSig), []byte(expectedSig)) {
			return errors.New("github signature mismatch")
		}
		return nil
	}

	// GitLab: X-Gitlab-Token
	gitlabToken := c.GetHeader("X-Gitlab-Token")
	if gitlabToken != "" {
		if gitlabToken != secret {
			return errors.New("gitlab token mismatch")
		}
		return nil
	}

	// If no signature header found, require the query secret
	return errors.New("no webhook signature or secret provided")
}
