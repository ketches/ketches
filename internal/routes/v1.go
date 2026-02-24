package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/handlers"
	"github.com/ketches/ketches/internal/middlewares"
)

func SetupV1Routes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	{
		// Webhooks (public, no auth)
		v1.POST("/webhooks/git/:appID", handlers.HandleGitWebhook)
		v1.POST("/webhooks/git/repo/:repoID", handlers.HandleGitWebhookForCodeRepo)

		users := v1.Group("/users")
		{
			users.POST("/sign-up", handlers.SignUp)
			users.POST("/sign-in", handlers.SignIn)
		}

		authorized := v1.Group("")
		authorized.Use(middlewares.Auth())
		{
			users := authorized.Group("/users")
			{
				users.GET("", handlers.ListUsers)
				users.POST("", middlewares.AdminOnly(), handlers.CreateUser)
				users.PUT("/:userID", handlers.UpdateUser)
				users.DELETE("/:userID", handlers.DeleteUser)
				users.PUT("/:userID/change-role", middlewares.AdminOnly(), handlers.ChangeUserRole)
			}

			projects := authorized.Group("/projects")
			{
				projects.GET("", handlers.ListProjects)
				projects.POST("", handlers.CreateProject)
				projects.GET("/:projectID", handlers.GetProject)
				projects.PUT("/:projectID", handlers.UpdateProject)
				projects.DELETE("/:projectID", handlers.DeleteProject)
				projects.GET("/:projectID/members", handlers.ListProjectMembers)
				projects.POST("/:projectID/members", handlers.AddProjectMember)
				projects.DELETE("/:projectID/members", handlers.RemoveProjectMember)
				projects.GET("/:projectID/envs", handlers.ListEnvs)
				projects.POST("/:projectID/envs", handlers.CreateEnv)

				// Image Registries (project scope)
				projects.GET("/:projectID/container-registries", handlers.ListProjectContainerRegistries)
				projects.POST("/:projectID/container-registries", handlers.CreateProjectContainerRegistry)

				// Code Repositories (project scope)
				projects.GET("/:projectID/code-repositories", handlers.ListCodeRepositories)
				projects.POST("/:projectID/code-repositories", handlers.CreateCodeRepository)

				projects.GET("/:projectID/plugins", handlers.ListPlugins)
				projects.POST("/:projectID/plugins", handlers.CreatePlugin)

				projects.GET("/:projectID/templates", handlers.ListTemplates)
				projects.POST("/:projectID/templates", handlers.CreateTemplate)
			}

			templates := authorized.Group("/templates")
			{
				templates.GET("/:templateID", handlers.GetTemplate)
				templates.PUT("/:templateID", handlers.UpdateTemplate)
				templates.DELETE("/:templateID", handlers.DeleteTemplate)
			}

			codeRepos := authorized.Group("/code-repositories")
			{
				codeRepos.GET("/:repoID", handlers.GetCodeRepository)
				codeRepos.GET("/:repoID/topology", handlers.GetCodeRepositoryTopology)
				codeRepos.GET("/:repoID/refs", handlers.ListCodeRepositoryRefs)
				codeRepos.PUT("/:repoID", handlers.UpdateCodeRepository)
				codeRepos.DELETE("/:repoID", handlers.DeleteCodeRepository)
				codeRepos.GET("/:repoID/container-registries", handlers.ListCodeRepositoryContainerRegistries)
				codeRepos.POST("/:repoID/test-git", handlers.TestCodeRepositoryGit)
				codeRepos.GET("/:repoID/build-configs", handlers.ListCodeRepositoryBuildConfigs)
				codeRepos.POST("/:repoID/build-configs", handlers.CreateCodeRepositoryBuildConfig)
				codeRepos.GET("/:repoID/build-configs/:configID", handlers.GetCodeRepositoryBuildConfig)
				codeRepos.PUT("/:repoID/build-configs/:configID", handlers.UpdateCodeRepositoryBuildConfig)
				codeRepos.DELETE("/:repoID/build-configs/:configID", handlers.DeleteCodeRepositoryBuildConfig)
				codeRepos.GET("/:repoID/builds", handlers.ListCodeRepositoryBuilds)
				codeRepos.GET("/:repoID/deployments", handlers.ListCodeRepositoryDeployments)
				codeRepos.POST("/:repoID/builds", handlers.TriggerCodeRepositoryBuild)
				codeRepos.GET("/:repoID/builds/:buildID", handlers.GetCodeRepositoryBuild)
				codeRepos.GET("/:repoID/builds/:buildID/logs", handlers.StreamCodeRepositoryBuildLogs)
				codeRepos.POST("/:repoID/builds/:buildID/cancel", handlers.CancelCodeRepositoryBuild)
				codeRepos.POST("/:repoID/builds/:buildID/deploy", handlers.DeployCodeRepositoryBuild)
			}

			envs := authorized.Group("/envs")
			{
				envs.GET("/:envID", handlers.GetEnv)
				envs.PUT("/:envID", handlers.UpdateEnv)
				envs.PATCH("/:envID/basic", handlers.UpdateEnvBasic)
				envs.DELETE("/:envID", handlers.DeleteEnv)
				envs.PATCH("/:envID/set-build-env", handlers.SetBuildEnv)
				envs.PATCH("/:envID/unset-build-env", handlers.UnsetBuildEnv)
				envs.GET("/:envID/apps", handlers.ListApps)
				envs.POST("/:envID/apps", handlers.CreateApp)
			}

			apps := authorized.Group("/apps")
			{
				apps.GET("/:appID", handlers.GetApp)
				apps.PUT("/:appID", handlers.UpdateApp)
				apps.PATCH("/:appID/basic", handlers.UpdateAppBasic)
				apps.DELETE("/:appID", handlers.DeleteApp)
				apps.GET("/:appID/available-actions", handlers.GetAppAvailableActions)
				apps.POST("/:appID/action", handlers.AppAction)
				apps.POST("/:appID/action/apply", handlers.ApplyApp)
				apps.GET("/:appID/topology", handlers.GetAppTopology)
				apps.GET("/:appID/topology/nodes/:nodeID/resource-yaml", handlers.GetAppTopologyResourceYaml)
				apps.GET("/:appID/instances", handlers.ListAppInstances)
				apps.DELETE("/:appID/instances/:instanceName", handlers.DeleteAppInstance)
				apps.GET("/:appID/instances/:instanceName/events", handlers.ListAppInstanceEvents)
				apps.GET("/:appID/instances/:instanceName/logs", handlers.StreamAppLogs)
				apps.GET("/:appID/instances/:instanceName/exec", handlers.ExecAppContainerTerminal)

				// File explorer
				apps.GET("/:appID/instances/:instanceName/files", handlers.ListFiles)
				apps.GET("/:appID/instances/:instanceName/files/home", handlers.GetHomeDir)
				apps.GET("/:appID/instances/:instanceName/files/read", handlers.ReadFile)
				apps.POST("/:appID/instances/:instanceName/files/write", handlers.WriteFile)
				apps.POST("/:appID/instances/:instanceName/files/mkdir", handlers.MkdirInContainer)
				apps.POST("/:appID/instances/:instanceName/files/delete", handlers.DeleteFileInContainer)
				apps.POST("/:appID/instances/:instanceName/files/move", handlers.MoveFileInContainer)
				apps.POST("/:appID/instances/:instanceName/files/copy", handlers.CopyFileInContainer)
				apps.GET("/:appID/instances/:instanceName/files/download", handlers.DownloadFile)
				apps.GET("/:appID/instances/:instanceName/files/download-dir", handlers.DownloadFileDir)
				apps.POST("/:appID/instances/:instanceName/files/upload", handlers.UploadFile)
				apps.POST("/:appID/instances/:instanceName/files/compress", handlers.CompressFiles)
				apps.POST("/:appID/instances/:instanceName/files/compress-download", handlers.CompressAndDownloadFiles)

				apps.GET("/:appID/env-vars", handlers.ListAppEnvVars)
				apps.POST("/:appID/env-vars", handlers.CreateAppEnvVar)

				apps.GET("/:appID/volumes", handlers.ListAppVolumes)
				apps.POST("/:appID/volumes", handlers.CreateAppVolume)

				apps.GET("/:appID/config-files", handlers.ListAppConfigFiles)
				apps.POST("/:appID/config-files", handlers.CreateAppConfigFile)

				apps.GET("/:appID/gateways", handlers.ListAppGateways)
				apps.POST("/:appID/gateways", handlers.CreateAppGateway)

				apps.GET("/:appID/plugins", handlers.ListAppPlugins)
				apps.POST("/:appID/plugins", handlers.InstallPluginToApp)
				apps.DELETE("/:appID/plugins/:pluginID", handlers.UninstallPluginFromApp)
				apps.PATCH("/:appID/plugins/:pluginID/toggle", handlers.ToggleAppPlugin)
				apps.PATCH("/:appID/plugins/:pluginID/env", handlers.UpdateAppPluginEnv)

				// Build Config
				apps.GET("/:appID/build-config", handlers.GetBuildConfig)
				apps.POST("/:appID/build-config", handlers.UpsertBuildConfig)
				apps.DELETE("/:appID/build-config", handlers.DeleteBuildConfig)
				apps.POST("/:appID/build-config/test-git", handlers.TestGitConnection)
				apps.GET("/:appID/build-config/container-registries", handlers.ListAvailableContainerRegistries)

				// Builds
				apps.GET("/:appID/builds", handlers.ListBuilds)
				apps.POST("/:appID/builds", handlers.TriggerBuild)
				apps.GET("/:appID/builds/:buildID", handlers.GetBuild)
				apps.GET("/:appID/builds/:buildID/logs", handlers.StreamBuildLogs)
				apps.POST("/:appID/builds/:buildID/cancel", handlers.CancelBuild)
				apps.POST("/:appID/builds/:buildID/deploy", handlers.DeployBuild)
				apps.POST("/:appID/builds/:buildID/rebuild", handlers.RebuildBuild)

				// Deployment History
				apps.GET("/:appID/deployment-history", handlers.ListDeploymentHistory)
				apps.POST("/:appID/deployment-history/rollback", handlers.RollbackDeployment)
			}

			authorized.PUT("/env-vars/:id", handlers.UpdateAppEnvVar)
			authorized.DELETE("/env-vars/:id", handlers.DeleteAppEnvVar)

			authorized.PUT("/volumes/:id", handlers.UpdateAppVolume)
			authorized.DELETE("/volumes/:id", handlers.DeleteAppVolume)

			authorized.PUT("/config-files/:id", handlers.UpdateAppConfigFile)
			authorized.DELETE("/config-files/:id", handlers.DeleteAppConfigFile)

			authorized.PUT("/gateways/:id", handlers.UpdateAppGateway)
			authorized.DELETE("/gateways/:id", handlers.DeleteAppGateway)

			plugins := authorized.Group("/plugins")
			{
				plugins.GET("", handlers.ListPlugins)
				plugins.POST("", handlers.CreatePlugin)
				plugins.GET("/:pluginID", handlers.GetPlugin)
				plugins.PUT("/:pluginID", handlers.UpdatePlugin)
				plugins.DELETE("/:pluginID", handlers.DeletePlugin)
				plugins.GET("/:pluginID/installed-apps", handlers.GetPluginInstalledApps)
			}

			authorized.GET("/clusters/public", handlers.ListPublicClusters)
			authorized.GET("/clusters/:clusterID/public", handlers.GetPublicCluster)
			authorized.GET("/clusters/:clusterID/storage-classes", handlers.ListStorageClasses)

			clusters := authorized.Group("/clusters")
			clusters.Use(middlewares.AdminOnly())
			{
				clusters.GET("", handlers.ListClusters)
				clusters.POST("", handlers.CreateCluster)
				clusters.POST("/ping", handlers.PingCluster)
				clusters.POST("/check-connectivity", handlers.CheckAllClustersConnectivity)
				clusters.GET("/:clusterID", handlers.GetCluster)
				clusters.PUT("/:clusterID", handlers.UpdateCluster)
				clusters.PATCH("/:clusterID/basic", handlers.UpdateClusterBasic)
				clusters.PATCH("/:clusterID/credentials", handlers.UpdateClusterCredentials)
				clusters.DELETE("/:clusterID", handlers.DeleteCluster)
				clusters.POST("/:clusterID/check-connectivity", handlers.CheckClusterConnectivity)
				clusters.GET("/:clusterID/nodes", handlers.ListClusterNodes)
				clusters.GET("/:clusterID/nodes/:nodeName", handlers.GetClusterNode)
				clusters.PATCH("/:clusterID/nodes/:nodeName/cordon", handlers.CordonClusterNode)
				clusters.PATCH("/:clusterID/nodes/:nodeName/labels", handlers.UpdateClusterNodeLabels)
				clusters.PATCH("/:clusterID/nodes/:nodeName/annotations", handlers.UpdateClusterNodeAnnotations)
				clusters.PATCH("/:clusterID/nodes/:nodeName/taints", handlers.UpdateClusterNodeTaints)
				clusters.GET("/:clusterID/nodes/:nodeName/exec", handlers.ExecClusterNodeTerminal)

				clusters.GET("/:clusterID/integrations", handlers.ListClusterIntegrations)
				clusters.POST("/:clusterID/integrations", handlers.CreateClusterIntegration)
				clusters.GET("/:clusterID/integrations/:integrationID", handlers.GetClusterIntegration)
				clusters.PUT("/:clusterID/integrations/:integrationID", handlers.UpdateClusterIntegration)
				clusters.DELETE("/:clusterID/integrations/:integrationID", handlers.DeleteClusterIntegration)

				clusters.GET("/:clusterID/namespaces", handlers.ListClusterNamespaces)
				clusters.GET("/:clusterID/services", handlers.ListClusterServices)

				// Helm Operator
				clusters.GET("/:clusterID/helm-operator/status", handlers.GetHelmOperatorStatus)
				clusters.POST("/:clusterID/helm-operator/install", handlers.InstallHelmOperator)
				clusters.POST("/:clusterID/helm-operator/uninstall", handlers.UninstallHelmOperator)

				// Helm Repositories
				clusters.GET("/:clusterID/helm-repositories", handlers.ListHelmRepositories)
				clusters.POST("/:clusterID/helm-repositories", handlers.CreateHelmRepository)
				clusters.GET("/:clusterID/helm-repositories/:repoName", handlers.GetHelmRepository)
				clusters.GET("/:clusterID/helm-repositories/:repoName/charts/:chartName/values", handlers.GetChartValues)
				clusters.DELETE("/:clusterID/helm-repositories/:repoName", handlers.DeleteHelmRepository)

				// Extensions (HelmReleases)
				clusters.GET("/:clusterID/extensions", handlers.ListExtensions)
				clusters.POST("/:clusterID/extensions", handlers.InstallExtension)
				clusters.GET("/:clusterID/extensions/:extensionName", handlers.GetExtension)
				clusters.PUT("/:clusterID/extensions/:extensionName", handlers.UpdateExtension)
				clusters.DELETE("/:clusterID/extensions/:extensionName", handlers.UninstallExtension)

				// Container Registries (cluster scope)
				clusters.GET("/:clusterID/container-registries", handlers.ListClusterRegistries)
				clusters.POST("/:clusterID/container-registries", handlers.CreateClusterRegistry)
			}

			// Container Registries (common scope)
			containerRegistries := authorized.Group("/container-registries")
			{
				containerRegistries.GET("/:registryID", handlers.GetContainerRegistry)
				containerRegistries.PUT("/:registryID", handlers.UpdateContainerRegistry)
				containerRegistries.DELETE("/:registryID", handlers.DeleteContainerRegistry)
				containerRegistries.POST("/:registryID/test", handlers.TestContainerRegistry)
			}

			// Prometheus metrics - accessible by all authenticated users (not AdminOnly)
			prometheus := authorized.Group("/clusters/:clusterID/prometheus")
			{
				prometheus.GET("/query", handlers.ProxyPrometheusQuery)
				prometheus.GET("/query_range", handlers.ProxyPrometheusQueryRange)
			}

			dashboard := authorized.Group("/dashboard")
			{
				dashboard.GET("/stats", handlers.GetDashboardStats)
				dashboard.GET("/environments", handlers.GetDashboardEnvironments)
			}

			recycleBin := authorized.Group("/recycle-bin")
			{
				recycleBin.GET("/apps", handlers.ListRecycleBinApps)
				recycleBin.POST("/apps/restore", handlers.RestoreApps)
				recycleBin.POST("/apps/permanently-delete", handlers.PermanentlyDeleteApps)
				recycleBin.GET("/envs", handlers.ListRecycleBinEnvs)
				recycleBin.POST("/envs/restore", handlers.RestoreEnvs)
				recycleBin.POST("/envs/permanently-delete", handlers.PermanentlyDeleteEnvs)
				recycleBin.GET("/envs/:envID/deletion-conflicts", handlers.CheckEnvDeletionConflicts)
			}
		}
	}
}
