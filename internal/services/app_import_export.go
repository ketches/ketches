package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ketches/ketches/internal/app"
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
		return nil, app.NewErrorf("unsupported import type: %s", importType)
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
				return nil, app.NewErrorf("application with slug %s already exists", appMeta.AppSlug)
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

		createdApp, err := CreateApp(context.Background(), envID, createReq)
		if err != nil {
			return nil, app.WrapErrorf(err, "failed to create app %s: %w", appMeta.AppName, err)
		}

		// Add EnvVars
		if len(appMeta.EnvVars) > 0 {
			var envVars []entities.AppEnvVar
			for _, ev := range appMeta.EnvVars {
				envVars = append(envVars, entities.AppEnvVar{
					ID:    uuid.New(),
					AppID: createdApp.App.ID,
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
					AppID:       createdApp.App.ID,
					Port:        gw.Port,
					Protocol:    gw.Protocol,
					GatewayPort: gw.GatewayPort,
					ServiceType: firstNonEmpty(gw.ServiceType, "ClusterIP"),
					NodePort:    gw.NodePort,
				}
				if err := db.DB.Create(gateway).Error; err != nil {
					return nil, err
				}
				for routeIndex, routeMeta := range gw.Routes {
					route := &entities.AppGatewayHTTPRoute{
						ID:               uuid.New(),
						AppGatewayID:     gateway.ID,
						Host:             routeMeta.Host,
						ListenerProtocol: routeMeta.ListenerProtocol,
						Path:             firstNonEmpty(routeMeta.Path, "/"),
						PathMatchType:    firstNonEmpty(routeMeta.PathMatchType, "PathPrefix"),
						Enabled:          routeMeta.Enabled,
						MatchesJSON:      marshalGatewayJSON(routeMeta.Matches),
						FiltersJSON:      marshalGatewayJSON(routeMeta.Filters),
						TimeoutsJSON:     marshalGatewayJSON(routeMeta.Timeouts),
						RetryJSON:        marshalGatewayJSON(routeMeta.Retry),
						ExtensionJSON:    marshalGatewayJSON(routeMeta.Extension),
						SortOrder:        routeIndex,
					}
					if routeMeta.CertID != "" {
						certID := routeMeta.CertID
						route.CertID = &certID
					}
					if routeMeta.SessionPersistence != nil {
						route.SessionPersistenceJSON = marshalGatewayJSON(routeMeta.SessionPersistence)
					}
					if err := db.DB.Create(route).Error; err != nil {
						return nil, err
					}
					backends := routeMeta.Backends
					if len(backends) == 0 {
						backends = []models.GatewayRouteBackendMetadata{{BackendAppSlug: createdApp.App.Slug, BackendPort: gw.Port, Weight: 1}}
					}
					for _, backendMeta := range backends {
						backendAppID := createdApp.App.ID
						if backendMeta.BackendAppSlug != "" && backendMeta.BackendAppSlug != createdApp.App.Slug {
							var backendApp entities.App
							if err := db.DB.Where("env_id = ? AND slug = ?", createdApp.App.EnvID, backendMeta.BackendAppSlug).First(&backendApp).Error; err != nil {
								return nil, err
							}
							backendAppID = backendApp.ID
						}
						backendPort := backendMeta.BackendPort
						if backendPort == 0 {
							backendPort = gw.Port
						}
						weight := backendMeta.Weight
						if weight == 0 {
							weight = 1
						}
						if err := db.DB.Create(&entities.AppGatewayHTTPRouteBackend{
							ID:           uuid.New(),
							RouteID:      route.ID,
							BackendAppID: backendAppID,
							BackendPort:  backendPort,
							Weight:       weight,
						}).Error; err != nil {
							return nil, err
						}
					}
				}
			}
		}

		// Add ConfigFiles
		if len(appMeta.ConfigFiles) > 0 {
			for _, cf := range appMeta.ConfigFiles {
				configFile := &entities.AppConfigFile{
					ID:        uuid.New(),
					AppID:     createdApp.App.ID,
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
					AppID:        createdApp.App.ID,
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
					AppID:               createdApp.App.ID,
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
			Name:   createdApp.App.Name,
			Slug:   createdApp.App.Slug,
			Status: "created",
		})
	}

	return result, nil
}

// ExportApps exports applications by IDs
func ExportApps(appIDs []string, format exporter.ExportFormat) (string, error) {
	var appCtxs []*models.AppContext
	for _, id := range appIDs {
		appCtx, err := GetAppContext(context.Background(), id)
		if err != nil {
			return "", err
		}
		appCtxs = append(appCtxs, appCtx)
	}

	appMetadatas := convertAppContextsToMetadata(appCtxs)

	return generateExport(appMetadatas, format)
}

// ExportEnvApps exports all applications in an environment
func ExportEnvApps(envID string, appIDs []string, format exporter.ExportFormat) (string, error) {
	var apps []entities.App
	query := db.DB.Where("env_id = ?", envID)

	if len(appIDs) > 0 {
		query = query.Where("id IN ?", appIDs)
	}

	if err := query.Find(&apps).Error; err != nil {
		return "", err
	}

	var appCtxs []*models.AppContext
	for _, a := range apps {
		appCtx, err := GetAppContext(context.Background(), a.ID)
		if err != nil {
			return "", err
		}
		appCtxs = append(appCtxs, appCtx)
	}

	appMetadatas := convertAppContextsToMetadata(appCtxs)

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
		return "", app.NewErrorf("unsupported export format: %s", format)
	}

	return generator.Generate(appMetadatas)
}

func convertAppContextsToMetadata(appCtxs []*models.AppContext) []models.AppMetadata {
	var metadatas []models.AppMetadata
	for _, appCtx := range appCtxs {
		app := &appCtx.App
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
		}

		if appCtx.AutoScaling != nil {
			meta.AutoScaling = &models.AutoScalingMetadata{
				MinReplicas:             appCtx.AutoScaling.MinReplicas,
				MaxReplicas:             appCtx.AutoScaling.MaxReplicas,
				TargetCPUUtilization:    appCtx.AutoScaling.TargetCPUUtilization,
				TargetMemoryUtilization: appCtx.AutoScaling.TargetMemoryUtilization,
			}
		}

		if appCtx.SchedulingRule != nil {
			meta.SchedulingRule = &models.SchedulingMetadata{
				RuleType:     appCtx.SchedulingRule.RuleType,
				NodeName:     appCtx.SchedulingRule.NodeName,
				NodeSelector: appCtx.SchedulingRule.NodeSelector,
				NodeAffinity: appCtx.SchedulingRule.NodeAffinity,
				Tolerations:  appCtx.SchedulingRule.Tolerations,
			}
		}

		for _, env := range appCtx.EnvVars {
			meta.EnvVars = append(meta.EnvVars, models.EnvVarMetadata{
				Key:   env.Key,
				Value: env.Value,
			})
		}

		routesByGateway := groupGatewayRoutes(appCtx.GatewayRoutes)
		backendsByRoute := groupGatewayBackends(appCtx.GatewayBackends)
		for _, gw := range appCtx.Gateways {
			gatewayMeta := models.GatewayMetadata{
				Port:        gw.Port,
				Protocol:    gw.Protocol,
				GatewayPort: gw.GatewayPort,
				ServiceType: gw.ServiceType,
				NodePort:    gw.NodePort,
			}
			for _, route := range routesByGateway[gw.ID] {
				routeMeta := models.GatewayRouteMetadata{
					Host:               route.Host,
					ListenerProtocol:   route.ListenerProtocol,
					Path:               route.Path,
					PathMatchType:      route.PathMatchType,
					Enabled:            route.Enabled,
					Matches:            gatewayJSONToMetadata[models.GatewayRouteMatches](route.MatchesJSON),
					Filters:            gatewayJSONToMetadata[models.GatewayRouteFilters](route.FiltersJSON),
					Timeouts:           gatewayJSONToMetadata[models.GatewayRouteTimeouts](route.TimeoutsJSON),
					Retry:              gatewayJSONToMetadata[models.GatewayRouteRetry](route.RetryJSON),
					SessionPersistence: gatewayJSONToMetadata[models.GatewaySessionPersistence](route.SessionPersistenceJSON),
					Extension:          gatewayJSONToMetadata[models.GatewayRouteExtension](route.ExtensionJSON),
				}
				if route.CertID != nil {
					routeMeta.CertID = *route.CertID
				}
				for _, backend := range backendsByRoute[route.ID] {
					routeMeta.Backends = append(routeMeta.Backends, models.GatewayRouteBackendMetadata{
						BackendAppSlug: backendAppSlugForMetadata(appCtx, backend.BackendAppID),
						BackendPort:    backend.BackendPort,
						Weight:         backend.Weight,
					})
				}
				gatewayMeta.Routes = append(gatewayMeta.Routes, routeMeta)
			}
			meta.Gateways = append(meta.Gateways, gatewayMeta)
		}

		for _, cf := range appCtx.ConfigFiles {
			meta.ConfigFiles = append(meta.ConfigFiles, models.ConfigFileMetadata{
				Slug:      cf.Slug,
				MountPath: cf.MountPath,
				Content:   cf.Content,
				FileMode:  cf.FileMode,
			})
		}

		for _, vol := range appCtx.Volumes {
			meta.Volumes = append(meta.Volumes, models.VolumeMetadata{
				Slug:         vol.Slug,
				MountPath:    vol.MountPath,
				SubPath:      vol.SubPath,
				VolumeType:   vol.VolumeType,
				Capacity:     vol.Capacity,
				StorageClass: vol.StorageClass,
			})
		}

		for _, p := range appCtx.Probes {
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

func gatewayJSONToMetadata[T any](blob entities.JSONBlob) *T {
	if len(blob) == 0 {
		return nil
	}
	var decoded T
	if err := json.Unmarshal(blob, &decoded); err != nil {
		return nil
	}
	return &decoded
}

func backendAppSlugForMetadata(appCtx *models.AppContext, backendAppID string) string {
	if appCtx == nil || backendAppID == "" {
		return ""
	}
	if backendAppID == appCtx.App.ID {
		return appCtx.App.Slug
	}
	var backend entities.App
	if err := db.DB.Select("slug").Where("id = ?", backendAppID).First(&backend).Error; err != nil {
		return ""
	}
	return backend.Slug
}
