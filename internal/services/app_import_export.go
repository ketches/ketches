package services

import (
	"fmt"
	"time"

	"github.com/ketches/ketches/internal/core/exporter"
	"github.com/ketches/ketches/internal/core/importer"
	"github.com/ketches/ketches/internal/db"
	"github.com/ketches/ketches/internal/db/entities"
	"github.com/ketches/ketches/internal/models"
	"github.com/ketches/ketches/pkg/uuid"
)

type ImportResult struct {
	Imported  []ImportAppResult `json:"imported"`
	Conflicts []ConflictInfo    `json:"conflicts"`
}

type ImportAppResult struct {
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	Status string `json:"status"` // "created", "updated"
}

type ConflictInfo struct {
	ExistingApp *entities.App       `json:"existing_app"`
	NewApp      *models.AppMetadata `json:"new_app"`
}

// ImportApps imports applications from content string
func ImportApps(envID string, importType string, content string, conflictStrategy string) (*ImportResult, error) {
	var converter importer.ImportConverter

	switch importType {
	case "dockercompose":
		converter = &importer.DockerComposeConverter{}
	case "kubernetes":
		converter = &importer.K8sManifestConverter{}
	case "ketches":
		converter = &importer.KetchesMetadataConverter{}
	default:
		return nil, fmt.Errorf("unsupported import type: %s", importType)
	}

	appMetadatas, err := converter.Parse(content)
	if err != nil {
		return nil, err
	}

	if err := converter.Validate(appMetadatas); err != nil {
		return nil, err
	}

	result := &ImportResult{
		Imported:  []ImportAppResult{},
		Conflicts: []ConflictInfo{},
	}

	for _, appMeta := range appMetadatas {
		// Check for existing app
		var existingApp entities.App
		err := db.DB.Where("env_id = ? AND slug = ?", envID, appMeta.AppSlug).First(&existingApp).Error
		exists := err == nil

		if exists {
			if conflictStrategy == "error" {
				return nil, fmt.Errorf("application with slug %s already exists", appMeta.AppSlug)
			} else if conflictStrategy == "ask" {
				result.Conflicts = append(result.Conflicts, ConflictInfo{
					ExistingApp: &existingApp,
					NewApp:      &appMeta,
				})
				continue
			} else if conflictStrategy == "rename" {
				// Generate new slug
				uniqueSuffix := uuid.New()[:8]
				appMeta.AppSlug = fmt.Sprintf("%s-%s", appMeta.AppSlug, uniqueSuffix)
				appMeta.AppName = fmt.Sprintf("%s-%s", appMeta.AppName, uniqueSuffix)
			} else {
				// Default to ask if unknown strategy
				result.Conflicts = append(result.Conflicts, ConflictInfo{
					ExistingApp: &existingApp,
					NewApp:      &appMeta,
				})
				continue
			}
		}

		// Create app
		createReq := appMeta.ToCreateAppRequest()
		createReq.Description = "Imported application"
		createReq.Deploy = false            // Do not deploy immediately
		createReq.SeedImageMetadata = false // pass through seed image metadata flag

		createdApp, err := CreateApp(envID, createReq)
		if err != nil {
			return nil, fmt.Errorf("failed to create app %s: %w", appMeta.AppName, err)
		}

		// Add EnvVars
		if len(appMeta.EnvVars) > 0 {
			var envVars []entities.AppEnvVar
			for _, ev := range appMeta.EnvVars {
				envVars = append(envVars, entities.AppEnvVar{
					ID:    uuid.New(),
					AppID: createdApp.ID,
					Key:   ev.Key,
					Value: ev.Value,
				})
			}
			if err := db.DB.Create(&envVars).Error; err != nil {
				return nil, err
			}
		}

		// Add Gateways
		if len(appMeta.Gateways) > 0 {
			for _, gw := range appMeta.Gateways {
				gateway := &entities.AppGateway{
					ID:          uuid.New(),
					AppID:       createdApp.ID,
					Port:        gw.Port,
					Protocol:    gw.Protocol,
					GatewayPort: gw.GatewayPort,
					Exposed:     gw.Exposed,
					Domain:      gw.Domain,
					Path:        gw.Path,
					CertID:      gw.CertID,
				}
				if err := db.DB.Create(gateway).Error; err != nil {
					return nil, err
				}
			}
		}

		// Add ConfigFiles
		if len(appMeta.ConfigFiles) > 0 {
			for _, cf := range appMeta.ConfigFiles {
				configFile := &entities.AppConfigFile{
					ID:        uuid.New(),
					AppID:     createdApp.ID,
					Slug:      cf.Slug,
					MountPath: cf.MountPath,
					Content:   cf.Content,
					FileMode:  cf.FileMode,
				}
				if configFile.FileMode == "" {
					configFile.FileMode = "0644"
				}
				if err := db.DB.Create(configFile).Error; err != nil {
					return nil, err
				}
			}
		}

		// Add Volumes
		if len(appMeta.Volumes) > 0 {
			for _, vol := range appMeta.Volumes {
				volume := &entities.AppVolume{
					ID:           uuid.New(),
					AppID:        createdApp.ID,
					Slug:         vol.Slug,
					MountPath:    vol.MountPath,
					SubPath:      vol.SubPath,
					VolumeType:   vol.VolumeType,
					Capacity:     vol.Capacity,
					StorageClass: vol.StorageClass,
				}
				if err := db.DB.Create(volume).Error; err != nil {
					return nil, err
				}
			}
		}

		// Add Probes
		if len(appMeta.Probes) > 0 {
			for _, p := range appMeta.Probes {
				probe := &entities.AppProbe{
					ID:                  uuid.New(),
					AppID:               createdApp.ID,
					Type:                p.Type,
					ProbeMode:           p.ProbeMode,
					Enabled:             p.Enabled,
					HttpGetPath:         p.HttpGetPath,
					HttpGetPort:         p.HttpGetPort,
					TcpSocketPort:       p.TcpSocketPort,
					ExecCommand:         p.ExecCommand,
					InitialDelaySeconds: p.InitialDelaySeconds,
					PeriodSeconds:       p.PeriodSeconds,
					TimeoutSeconds:      p.TimeoutSeconds,
					SuccessThreshold:    p.SuccessThreshold,
					FailureThreshold:    p.FailureThreshold,
				}
				if err := db.DB.Create(probe).Error; err != nil {
					return nil, err
				}
			}
		}

		result.Imported = append(result.Imported, ImportAppResult{
			Name:   createdApp.Name,
			Slug:   createdApp.Slug,
			Status: "created",
		})
	}

	return result, nil
}

// ExportApps exports applications by IDs
func ExportApps(appIDs []string, format exporter.ExportFormat) (string, error) {
	var apps []entities.App
	if err := db.DB.Preload("EnvVars").Preload("Gateways").Preload("ConfigFiles").Preload("Volumes").Preload("AutoScaling").Preload("SchedulingRule").Preload("Probes").Where("id IN ?", appIDs).Find(&apps).Error; err != nil {
		return "", err
	}

	appMetadatas := convertAppsToMetadata(apps)

	return generateExport(appMetadatas, format)
}

// ExportEnvApps exports all applications in an environment
func ExportEnvApps(envID string, appIDs []string, format exporter.ExportFormat) (string, error) {
	var apps []entities.App
	query := db.DB.Preload("EnvVars").Preload("Gateways").Preload("ConfigFiles").Preload("Volumes").Preload("AutoScaling").Preload("SchedulingRule").Preload("Probes").Where("env_id = ?", envID)

	if len(appIDs) > 0 {
		query = query.Where("id IN ?", appIDs)
	}

	if err := query.Find(&apps).Error; err != nil {
		return "", err
	}

	appMetadatas := convertAppsToMetadata(apps)

	return generateExport(appMetadatas, format)
}

func generateExport(appMetadatas []models.AppMetadata, format exporter.ExportFormat) (string, error) {
	var generator exporter.ExportGenerator
	switch format {
	case exporter.FormatKubernetes:
		generator = &exporter.K8sManifestGenerator{}
	case exporter.FormatKetches:
		generator = &exporter.KetchesMetadataGenerator{}
	case exporter.FormatHelm:
		generator = &exporter.HelmChartGenerator{}
	case exporter.FormatDockerCompose:
		generator = &exporter.DockerComposeGenerator{}
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}

	return generator.Generate(appMetadatas)
}

func convertAppsToMetadata(apps []entities.App) []models.AppMetadata {
	var metadatas []models.AppMetadata
	for _, app := range apps {
		meta := models.AppMetadata{
			AppName:          app.Name,
			AppSlug:          app.Slug,
			AppType:          app.AppType,
			Description:      app.Description,
			ContainerImage:   app.ContainerImage,
			ContainerCommand: app.ContainerCommand,
			Replicas:         app.Replicas,
			RequestCPU:       app.RequestCPU,
			RequestMemory:    app.RequestMemory,
			LimitCPU:         app.LimitCPU,
			LimitMemory:      app.LimitMemory,
			RegistryUsername: app.RegistryUsername,
			RegistryPassword: app.RegistryPassword,
			ImportedAt:       time.Now(),
		}

		if app.AutoScaling != nil {
			meta.AutoScaling = &models.AutoScalingMetadata{
				MinReplicas:             app.AutoScaling.MinReplicas,
				MaxReplicas:             app.AutoScaling.MaxReplicas,
				TargetCPUUtilization:    app.AutoScaling.TargetCPUUtilization,
				TargetMemoryUtilization: app.AutoScaling.TargetMemoryUtilization,
			}
		}

		if app.SchedulingRule != nil {
			meta.SchedulingRule = &models.SchedulingMetadata{
				RuleType:     app.SchedulingRule.RuleType,
				NodeName:     app.SchedulingRule.NodeName,
				NodeSelector: app.SchedulingRule.NodeSelector,
				NodeAffinity: app.SchedulingRule.NodeAffinity,
				Tolerations:  app.SchedulingRule.Tolerations,
			}
		}

		for _, env := range app.EnvVars {
			meta.EnvVars = append(meta.EnvVars, models.EnvVarMetadata{
				Key:   env.Key,
				Value: env.Value,
			})
		}

		for _, gw := range app.Gateways {
			meta.Gateways = append(meta.Gateways, models.GatewayMetadata{
				Port:        gw.Port,
				Protocol:    gw.Protocol,
				GatewayPort: gw.GatewayPort,
				Exposed:     gw.Exposed,
				Domain:      gw.Domain,
				Path:        gw.Path,
				CertID:      gw.CertID,
			})
		}

		for _, cf := range app.ConfigFiles {
			meta.ConfigFiles = append(meta.ConfigFiles, models.ConfigFileMetadata{
				Slug:      cf.Slug,
				MountPath: cf.MountPath,
				Content:   cf.Content,
				FileMode:  cf.FileMode,
			})
		}

		for _, vol := range app.Volumes {
			meta.Volumes = append(meta.Volumes, models.VolumeMetadata{
				Slug:         vol.Slug,
				MountPath:    vol.MountPath,
				SubPath:      vol.SubPath,
				VolumeType:   vol.VolumeType,
				Capacity:     vol.Capacity,
				StorageClass: vol.StorageClass,
			})
		}

		for _, p := range app.Probes {
			meta.Probes = append(meta.Probes, models.ProbeMetadata{
				Type:                p.Type,
				ProbeMode:           p.ProbeMode,
				Enabled:             p.Enabled,
				HttpGetPath:         p.HttpGetPath,
				HttpGetPort:         p.HttpGetPort,
				TcpSocketPort:       p.TcpSocketPort,
				ExecCommand:         p.ExecCommand,
				InitialDelaySeconds: p.InitialDelaySeconds,
				PeriodSeconds:       p.PeriodSeconds,
				TimeoutSeconds:      p.TimeoutSeconds,
				SuccessThreshold:    p.SuccessThreshold,
				FailureThreshold:    p.FailureThreshold,
			})
		}

		metadatas = append(metadatas, meta)
	}
	return metadatas
}
