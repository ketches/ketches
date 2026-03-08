package db

import (
	"fmt"
	"log"

	"github.com/ketches/ketches/internal/db/entities"
	"gorm.io/gorm"
)

func Migrate() error {
	log.Printf("Running AutoMigrate...")

	if DB.Dialector.Name() == "mysql" {
		DB.Exec("SET FOREIGN_KEY_CHECKS = 0;")
		defer DB.Exec("SET FOREIGN_KEY_CHECKS = 1;")
	}

	if err := DB.AutoMigrate(
		&entities.User{},
		&entities.Cluster{},
		&entities.ClusterIntegration{},
		&entities.Project{},
		&entities.ProjectMember{},
		&entities.Env{},
		&entities.App{},
		&entities.AppEnvVar{},
		&entities.AppVolume{},
		&entities.AppGateway{},
		&entities.AppProbe{},
		&entities.AppConfigFile{},
		&entities.AppSchedulingRule{},
		&entities.AppAutoScaling{},
		&entities.Plugin{},
		&entities.AppPlugin{},
		&entities.AppGroup{},
		&entities.AppGroupMember{},
		&entities.AppFavorite{},
		&entities.ContainerRegistry{},
		&entities.BuildSetting{},
		&entities.Build{},
		&entities.BuildDeployment{},
		&entities.CodeRepository{},
		&entities.DeploymentHistory{},
		&entities.Certificate{},
		&entities.Extension{},
		&entities.ClusterExtension{},
	); err != nil {
		return err
	}
	// No physical foreign keys will be created due to DisableForeignKeyConstraintWhenMigrating: true

	// Data migration: copy old app_build_configs and code_repository_build_configs into build_settings,
	// then backfill builds.build_setting_id from the old columns.
	if err := migrateBuildSettingsData(DB); err != nil {
		log.Printf("Warning: build settings data migration encountered errors: %v", err)
	}

	// Data migration: backfill build_deployments from builds.app_id for existing records.
	if err := migrateBuildDeploymentsData(DB); err != nil {
		log.Printf("Warning: build deployments data migration encountered errors: %v", err)
	}

	return nil
}

// migrateBuildSettingsData migrates existing rows from old tables into build_settings
// and backfills builds.build_setting_id. Safe to run on a fresh DB (no-op).
func migrateBuildSettingsData(db *gorm.DB) error {
	// 1. Migrate app_build_configs → build_settings
	type OldAppBuildConfig struct {
		ID             string
		AppID          string
		Name           string
		GitRef         string
		GitUsername    string
		GitPassword    string
		DockerfilePath string
		BuildContext   string
		BuildArgs      string
		ImageName      string
		RegistryID     string
		AutoBuild      bool
		AutoDeploy     bool
		WebhookEnabled bool
		WebhookSecret  string
	}
	var appConfigs []OldAppBuildConfig
	if err := db.Raw("SELECT * FROM app_build_configs").Scan(&appConfigs).Error; err == nil {
		for _, c := range appConfigs {
			appID := c.AppID
			db.Exec(`INSERT OR IGNORE INTO build_settings
				(id, app_id, code_repository_id, name, git_ref, git_username, git_password,
				 dockerfile_path, build_context, build_args, image_name, registry_id,
				 auto_build, auto_deploy, webhook_enabled, webhook_secret)
				VALUES (?,?,NULL,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				c.ID, appID, c.Name, c.GitRef, c.GitUsername, c.GitPassword,
				c.DockerfilePath, c.BuildContext, c.BuildArgs, c.ImageName, c.RegistryID,
				c.AutoBuild, c.AutoDeploy, c.WebhookEnabled, c.WebhookSecret)
		}
	}

	// 2. Migrate code_repository_build_configs → build_settings
	type OldRepoConfig struct {
		ID               string
		CodeRepositoryID string
		Name             string
		GitRef           string
		DockerfilePath   string
		BuildContext     string
		BuildArgs        string
		ImageName        string
		RegistryID       string
		AutoBuild        bool
		AutoDeploy       bool
		WebhookEnabled   bool
	}
	var repoConfigs []OldRepoConfig
	if err := db.Raw("SELECT * FROM code_repository_build_configs").Scan(&repoConfigs).Error; err == nil {
		for _, c := range repoConfigs {
			repoID := c.CodeRepositoryID
			db.Exec(`INSERT OR IGNORE INTO build_settings
				(id, app_id, code_repository_id, name, git_ref, git_username, git_password,
				 dockerfile_path, build_context, build_args, image_name, registry_id,
				 auto_build, auto_deploy, webhook_enabled, webhook_secret)
				VALUES (?,NULL,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
				c.ID, repoID, c.Name, c.GitRef, "", "",
				c.DockerfilePath, c.BuildContext, c.BuildArgs, c.ImageName, c.RegistryID,
				c.AutoBuild, c.AutoDeploy, c.WebhookEnabled, "")
		}
	}

	// 3. Backfill builds.build_setting_id from old columns
	// App builds: build_config_id → build_setting_id
	db.Exec(`UPDATE builds SET build_setting_id = build_config_id
		WHERE build_config_id IS NOT NULL AND build_config_id != '' AND (build_setting_id IS NULL OR build_setting_id = '')`,)
	// Repo builds: code_repository_build_config_id → build_setting_id
	db.Exec(`UPDATE builds SET build_setting_id = code_repository_build_config_id
		WHERE code_repository_build_config_id IS NOT NULL AND code_repository_build_config_id != ''
		AND (build_setting_id IS NULL OR build_setting_id = '')`,)

	return nil
}

// migrateBuildDeploymentsData creates build_deployment rows for existing builds
// that have app_id set (i.e., were already deployed). Safe to run multiple times.
func migrateBuildDeploymentsData(db *gorm.DB) error {
	driver := db.Dialector.Name()
	var stmt string
	switch driver {
	case "mysql":
		stmt = `INSERT IGNORE INTO build_deployments
			(id, created_at, updated_at, build_id, app_id, env_id, app_name, app_slug, status, deployed_by, deployed_at, error_message)
			SELECT
				CONCAT(b.id, '-migrated'),
				b.created_at,
				b.updated_at,
				b.id,
				b.app_id,
				COALESCE(a.env_id, ''),
				'',
				'',
				'deployed',
				'migration',
				b.completed_at,
				''
			FROM builds b
			LEFT JOIN apps a ON a.id = b.app_id
			WHERE b.app_id IS NOT NULL AND b.app_id != ''`
	case "postgres":
		stmt = `INSERT INTO build_deployments
			(id, created_at, updated_at, build_id, app_id, env_id, app_name, app_slug, status, deployed_by, deployed_at, error_message)
			SELECT
				b.id || '-migrated',
				b.created_at,
				b.updated_at,
				b.id,
				b.app_id,
				COALESCE(a.env_id, ''),
				'',
				'',
				'deployed',
				'migration',
				b.completed_at,
				''
			FROM builds b
			LEFT JOIN apps a ON a.id = b.app_id
			WHERE b.app_id IS NOT NULL AND b.app_id != ''
			ON CONFLICT (id) DO NOTHING`
	default:
		// SQLite
		stmt = `INSERT OR IGNORE INTO build_deployments
			(id, created_at, updated_at, build_id, app_id, env_id, app_name, app_slug, status, deployed_by, deployed_at, error_message)
			SELECT
				b.id || '-migrated',
				b.created_at,
				b.updated_at,
				b.id,
				b.app_id,
				COALESCE(a.env_id, ''),
				'',
				'',
				'deployed',
				'migration',
				b.completed_at,
				''
			FROM builds b
			LEFT JOIN apps a ON a.id = b.app_id
			WHERE b.app_id IS NOT NULL AND b.app_id != ''`
	}
	if err := db.Exec(stmt).Error; err != nil {
		return fmt.Errorf("build_deployments migration: %w", err)
	}
	return nil
}
