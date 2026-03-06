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
		v1.GET("/version", handlers.GetVersion)

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
				users.POST("/import", middlewares.AdminOnly(), handlers.ImportUsers)
				users.PUT("/:userID", handlers.UpdateUser)
				users.DELETE("/:userID", handlers.DeleteUser)
				users.PUT("/:userID/change-role", middlewares.AdminOnly(), handlers.ChangeUserRole)
				users.PATCH("/:userID/role", middlewares.AdminOnly(), handlers.ChangeUserRole)
			}

			// ── Projects ──────────────────────────────────────────────
			projects := authorized.Group("/projects")
			{
				// Read (no extra middleware)
				projects.GET("", handlers.ListProjects)
				projects.GET("/simple", handlers.ListProjectsSimple)
				projects.POST("", handlers.CreateProject) // create project — no existing role
				projects.GET("/:projectID", handlers.GetProject)
				projects.GET("/:projectID/members", handlers.ListProjectMembers)
				projects.GET("/:projectID/envs", handlers.ListEnvs)
				projects.GET("/:projectID/envs/simple", handlers.ListEnvsSimple)
				projects.GET("/:projectID/container-registries", handlers.ListProjectContainerRegistries)
				projects.GET("/:projectID/container-registries/simple", handlers.ListProjectContainerRegistriesSimple)
				projects.GET("/:projectID/code-repositories", handlers.ListCodeRepositories)
				projects.GET("/:projectID/code-repositories/simple", handlers.ListCodeRepositoriesSimple)
				projects.GET("/:projectID/plugins", handlers.ListPlugins)
				projects.GET("/:projectID/plugins/simple", handlers.ListPluginsSimple)
				projects.GET("/:projectID/plugins/:pluginID", handlers.GetPlugin)
				projects.GET("/:projectID/plugins/:pluginID/installed-apps", handlers.GetPluginInstalledApps)

				// Write (require at least developer role)
				projectsWrite := projects.Group("", middlewares.RequireProjectRole("developer"))
				projectsWrite.PUT("/:projectID", handlers.UpdateProject)
				projectsWrite.DELETE("/:projectID", handlers.DeleteProject)
				projectsWrite.POST("/:projectID/members", handlers.AddProjectMember)
				projectsWrite.DELETE("/:projectID/members", handlers.RemoveProjectMember)
				projectsWrite.POST("/:projectID/envs", handlers.CreateEnv)
				projectsWrite.POST("/:projectID/container-registries", handlers.CreateProjectContainerRegistry)
				projectsWrite.POST("/:projectID/code-repositories", handlers.CreateCodeRepository)
				projectsWrite.POST("/:projectID/plugins", handlers.CreatePlugin)
				projectsWrite.PUT("/:projectID/plugins/:pluginID", handlers.UpdatePlugin)
				projectsWrite.DELETE("/:projectID/plugins/:pluginID", handlers.DeletePlugin)
			}

			// ── Code Repositories ─────────────────────────────────────
			codeRepos := authorized.Group("/code-repositories")
			{
				// Read (no extra middleware)
				codeRepos.GET("/:repoID", handlers.GetCodeRepository)
				codeRepos.GET("/:repoID/topology", handlers.GetCodeRepositoryTopology)
				codeRepos.GET("/:repoID/refs", handlers.ListCodeRepositoryRefs)
				codeRepos.GET("/:repoID/container-registries", handlers.ListCodeRepositoryContainerRegistries)
				codeRepos.GET("/:repoID/build-configs", handlers.ListCodeRepositoryBuildConfigs)
				codeRepos.GET("/:repoID/build-configs/:configID", handlers.GetCodeRepositoryBuildConfig)
				codeRepos.GET("/:repoID/builds", handlers.ListCodeRepositoryBuilds)
				codeRepos.GET("/:repoID/builds/:buildID", handlers.GetCodeRepositoryBuild)
				codeRepos.GET("/:repoID/builds/:buildID/logs", handlers.StreamCodeRepositoryBuildLogs)
				codeRepos.GET("/:repoID/deployments", handlers.ListCodeRepositoryDeployments)

				// Write (require at least developer role)
				codeReposWrite := codeRepos.Group("", middlewares.RequireProjectRole("developer"))
				codeReposWrite.PUT("/:repoID", handlers.UpdateCodeRepository)
				codeReposWrite.DELETE("/:repoID", handlers.DeleteCodeRepository)
				codeReposWrite.POST("/:repoID/test-git", handlers.TestCodeRepositoryGit)
				codeReposWrite.POST("/:repoID/build-configs", handlers.CreateCodeRepositoryBuildConfig)
				codeReposWrite.PUT("/:repoID/build-configs/:configID", handlers.UpdateCodeRepositoryBuildConfig)
				codeReposWrite.DELETE("/:repoID/build-configs/:configID", handlers.DeleteCodeRepositoryBuildConfig)
				codeReposWrite.POST("/:repoID/builds", handlers.TriggerCodeRepositoryBuild)
				codeReposWrite.POST("/:repoID/builds/:buildID/cancel", handlers.CancelCodeRepositoryBuild)
				codeReposWrite.POST("/:repoID/builds/:buildID/deploy", handlers.DeployCodeRepositoryBuild)
			}

			// ── Environments ──────────────────────────────────────────
			envs := authorized.Group("/envs")
			{
				// Read (no extra middleware)
				envs.GET("/:envID", handlers.GetEnv)
				envs.GET("/:envID/apps", handlers.ListApps)
				envs.GET("/:envID/apps/simple", handlers.ListAppsSimple)
				envs.GET("/:envID/apps/export", handlers.ExportEnvApps)
				envs.GET("/:envID/app-groups", handlers.ListAppGroups)
				envs.GET("/:envID/apps/favorites", handlers.ListFavoriteApps)
				envs.GET("/:envID/certificates", handlers.ListEnvCertificates)
				envs.GET("/:envID/apps/image-metadata", handlers.GetImageMetadata)
				envs.GET("/:envID/certificates/:certID", handlers.GetCertificate)
				// Write (require at least developer role)
				envsWrite := envs.Group("", middlewares.RequireProjectRole("developer"))
				envsWrite.PUT("/:envID", handlers.UpdateEnv)
				envsWrite.PATCH("/:envID/basic", handlers.UpdateEnvBasic)
				envsWrite.DELETE("/:envID", handlers.DeleteEnv)
				envsWrite.PATCH("/:envID/set-build-env", handlers.SetBuildEnv)
				envsWrite.PATCH("/:envID/unset-build-env", handlers.UnsetBuildEnv)
				envsWrite.POST("/:envID/apps", handlers.CreateApp)
				envsWrite.POST("/:envID/apps/import", handlers.ImportApps)
				envsWrite.POST("/:envID/app-groups", handlers.CreateAppGroup)
				envsWrite.POST("/:envID/certificates", handlers.CreateEnvCertificate)
				envsWrite.PUT("/:envID/certificates/:certID", handlers.UpdateCertificate)
				envsWrite.DELETE("/:envID/certificates/:certID", handlers.DeleteCertificate)
			}

			// ── Apps ──────────────────────────────────────────────────
			apps := authorized.Group("/apps")
			{
				// Read (no extra middleware)
				apps.GET("/:appID", handlers.GetApp)
				apps.GET("/:appID/export", handlers.ExportApps)
				apps.GET("/:appID/available-actions", handlers.GetAppAvailableActions)
				apps.GET("/:appID/topology", handlers.GetAppTopology)
				apps.GET("/:appID/instances", handlers.ListAppInstances)
				apps.GET("/:appID/instances/:instanceName/events", handlers.ListAppInstanceEvents)
				apps.GET("/:appID/env-vars", handlers.ListAppEnvVars)
				apps.GET("/:appID/volumes", handlers.ListAppVolumes)
				apps.GET("/:appID/config-files", handlers.ListAppConfigFiles)
				apps.GET("/:appID/gateways", handlers.ListAppGateways)
				apps.GET("/:appID/plugins", handlers.ListAppPlugins)
				apps.GET("/:appID/build-config", handlers.GetBuildConfig)
				apps.GET("/:appID/build-config/container-registries", handlers.ListAvailableContainerRegistries)
				apps.GET("/:appID/builds", handlers.ListBuilds)
				apps.GET("/:appID/builds/:buildID", handlers.GetBuild)
				apps.GET("/:appID/builds/:buildID/logs", handlers.StreamBuildLogs)
				apps.GET("/:appID/deployment-history", handlers.ListDeploymentHistory)
				apps.GET("/:appID/favorite", handlers.GetAppFavoriteStatus)

				// Exec / Log / Files (block viewer — require at least developer)
				appsExec := apps.Group("", middlewares.BlockViewer())
				appsExec.GET("/:appID/topology/nodes/:nodeID/resource-yaml", handlers.GetAppTopologyResourceYaml)
				appsExec.GET("/:appID/instances/:instanceName/logs", handlers.StreamAppLogs)
				appsExec.GET("/:appID/instances/:instanceName/exec", handlers.ExecAppContainerTerminal)
				appsExec.GET("/:appID/instances/:instanceName/files", handlers.ListFiles)
				appsExec.GET("/:appID/instances/:instanceName/files/home", handlers.GetHomeDir)
				appsExec.GET("/:appID/instances/:instanceName/files/read", handlers.ReadFile)
				appsExec.POST("/:appID/instances/:instanceName/files/write", handlers.WriteFile)
				appsExec.POST("/:appID/instances/:instanceName/files/mkdir", handlers.MkdirInContainer)
				appsExec.POST("/:appID/instances/:instanceName/files/delete", handlers.DeleteFileInContainer)
				appsExec.POST("/:appID/instances/:instanceName/files/move", handlers.MoveFileInContainer)
				appsExec.POST("/:appID/instances/:instanceName/files/copy", handlers.CopyFileInContainer)
				appsExec.GET("/:appID/instances/:instanceName/files/download", handlers.DownloadFile)
				appsExec.GET("/:appID/instances/:instanceName/files/download-dir", handlers.DownloadFileDir)
				appsExec.POST("/:appID/instances/:instanceName/files/upload", handlers.UploadFile)
				appsExec.POST("/:appID/instances/:instanceName/files/compress", handlers.CompressFiles)
				appsExec.POST("/:appID/instances/:instanceName/files/compress-download", handlers.CompressAndDownloadFiles)

				// Write (require at least developer role)
				appsWrite := apps.Group("", middlewares.RequireProjectRole("developer"))
				appsWrite.PATCH("/:appID/basic", handlers.UpdateAppBasic)
				appsWrite.PATCH("/:appID/image", handlers.UpdateAppImage)
				appsWrite.PATCH("/:appID/replicas", handlers.UpdateAppReplicas)
				appsWrite.PATCH("/:appID/resources", handlers.UpdateAppResources)
				appsWrite.PATCH("/:appID/auto-scaling", handlers.UpdateAppAutoScaling)
				appsWrite.PATCH("/:appID/health", handlers.UpdateAppHealth)
				appsWrite.PATCH("/:appID/scheduling", handlers.UpdateAppScheduling)
				appsWrite.PATCH("/:appID/command", handlers.UpdateAppCommand)
				appsWrite.DELETE("/:appID", handlers.DeleteApp)
				appsWrite.POST("/batch-delete", handlers.BatchDeleteApps)
				appsWrite.DELETE("/:appID/instances/:instanceName", handlers.DeleteAppInstance)
				appsWrite.POST("/:appID/action", handlers.AppAction)
				appsWrite.POST("/:appID/action/apply", handlers.ApplyApp)
				appsWrite.POST("/:appID/env-vars", handlers.CreateAppEnvVar)
				appsWrite.POST("/:appID/volumes", handlers.CreateAppVolume)
				appsWrite.POST("/:appID/config-files", handlers.CreateAppConfigFile)
				appsWrite.POST("/:appID/gateways", handlers.CreateAppGateway)
				appsWrite.POST("/:appID/plugins", handlers.InstallPluginToApp)
				appsWrite.DELETE("/:appID/plugins/:pluginID", handlers.UninstallPluginFromApp)
				appsWrite.PATCH("/:appID/plugins/:pluginID/toggle", handlers.ToggleAppPlugin)
				appsWrite.PATCH("/:appID/plugins/:pluginID/env", handlers.UpdateAppPluginEnv)
				appsWrite.POST("/:appID/build-config", handlers.UpsertBuildConfig)
				appsWrite.DELETE("/:appID/build-config", handlers.DeleteBuildConfig)
				appsWrite.POST("/:appID/build-config/test-git", handlers.TestGitConnection)
				appsWrite.POST("/:appID/builds", handlers.TriggerBuild)
				appsWrite.POST("/:appID/builds/:buildID/cancel", handlers.CancelBuild)
				appsWrite.POST("/:appID/builds/:buildID/deploy", handlers.DeployBuild)
				appsWrite.POST("/:appID/builds/:buildID/rebuild", handlers.RebuildBuild)
				appsWrite.POST("/:appID/deployment-history/rollback", handlers.RollbackDeployment)
				appsWrite.POST("/:appID/favorite", handlers.AddFavoriteApp)
				appsWrite.DELETE("/:appID/favorite", handlers.RemoveFavoriteApp)
			}

			// Flat resource write routes — RequireProjectRole resolves project via resource→app→env chain.
			// ── App Groups (flat write routes) ───────────────────────────────
			appGroupsWrite := authorized.Group("/app-groups", middlewares.RequireProjectRole("developer"))
			{
				appGroupsWrite.GET("/:groupID", handlers.GetAppGroup)
				appGroupsWrite.GET("/:groupID/apps", handlers.ListSpecificGroupedApps)
				appGroupsWrite.PUT("/:groupID", handlers.UpdateAppGroup)
				appGroupsWrite.DELETE("/:groupID", handlers.DeleteAppGroup)
				appGroupsWrite.POST("/:groupID/apps/:appID", handlers.AddAppToGroup)
				appGroupsWrite.DELETE("/:groupID/apps/:appID", handlers.RemoveAppFromGroup)
			}

			flatResourcesWrite := authorized.Group("", middlewares.RequireProjectRole("developer"))
			{
				flatResourcesWrite.PUT("/env-vars/:id", handlers.UpdateAppEnvVar)
				flatResourcesWrite.DELETE("/env-vars/:id", handlers.DeleteAppEnvVar)
				flatResourcesWrite.PUT("/volumes/:id", handlers.UpdateAppVolume)
				flatResourcesWrite.DELETE("/volumes/:id", handlers.DeleteAppVolume)
				flatResourcesWrite.PUT("/config-files/:id", handlers.UpdateAppConfigFile)
				flatResourcesWrite.DELETE("/config-files/:id", handlers.DeleteAppConfigFile)
				flatResourcesWrite.PUT("/gateways/:id", handlers.UpdateAppGateway)
				flatResourcesWrite.DELETE("/gateways/:id", handlers.DeleteAppGateway)
			}

			authorized.GET("/clusters/public", handlers.ListPublicClusters)
			authorized.GET("/clusters/:clusterID/public", handlers.GetPublicCluster)
			authorized.GET("/clusters/:clusterID/storage-classes", handlers.ListStorageClasses)
			authorized.GET("/clusters/:clusterID/gateway-api-status", handlers.GetClusterGatewayAPIStatus)

			// Extensions (platform-level)
			extensions := authorized.Group("/extensions")
			extensions.Use(middlewares.AdminOnly())
			{
				extensions.GET("", handlers.ListExtensions)
				extensions.POST("", middlewares.AdminOnly(), handlers.CreateExtension)
				extensions.DELETE("/:extensionID", middlewares.AdminOnly(), handlers.DeleteExtension)
				extensions.PUT("/:extensionID", middlewares.AdminOnly(), handlers.UpdateExtension)
				extensions.GET("/:extensionID/versions", handlers.ListExtensionVersions)
				extensions.GET("/:extensionID/versions/:version/values", handlers.GetExtensionValues)
				extensions.GET("/:extensionID/installed-clusters", handlers.GetInstalledClustersForExtension)
			}
			clusters := authorized.Group("/clusters")
			clusters.Use(middlewares.AdminOnly())
			{
				clusters.GET("", handlers.ListClusters)
				clusters.GET("/simple", handlers.ListClustersSimple)
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

				// Cluster Extensions
				clusters.GET("/:clusterID/extensions", handlers.ListClusterExtensions)
				clusters.POST("/:clusterID/extensions", handlers.InstallClusterExtension)
				clusters.GET("/:clusterID/extensions/:clusterExtensionID", handlers.GetClusterExtension)
				clusters.PUT("/:clusterID/extensions/:clusterExtensionID", handlers.UpgradeClusterExtension)
				clusters.DELETE("/:clusterID/extensions/:clusterExtensionID", handlers.UninstallClusterExtension)

				// Container Registries (cluster scope)
				clusters.GET("/:clusterID/container-registries", handlers.ListClusterRegistries)
				clusters.POST("/:clusterID/container-registries", handlers.CreateClusterRegistry)

				// Certificates (cluster scope)
				clusters.GET("/:clusterID/certificates", handlers.ListClusterCertificates)
				clusters.POST("/:clusterID/certificates", handlers.CreateClusterCertificate)
				clusters.GET("/:clusterID/certificates/:certID", handlers.GetCertificate)
				clusters.PUT("/:clusterID/certificates/:certID", handlers.UpdateCertificate)
				clusters.DELETE("/:clusterID/certificates/:certID", handlers.DeleteCertificate)
			}

			// Container Registries (project-scoped) — write routes enforce developer role via registryID→project chain.
			containerRegistries := authorized.Group("/container-registries")
			{
				containerRegistries.GET("/:registryID", handlers.GetContainerRegistry)
				containerRegistriesWrite := containerRegistries.Group("", middlewares.RequireProjectRole("developer"))
				{
					containerRegistriesWrite.PUT("/:registryID", handlers.UpdateContainerRegistry)
					containerRegistriesWrite.DELETE("/:registryID", handlers.DeleteContainerRegistry)
					containerRegistriesWrite.POST("/:registryID/test", handlers.TestContainerRegistry)
				}
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
				recycleBin.GET("/projects", handlers.ListRecycleBinProjects)
				recycleBin.POST("/projects/restore", handlers.RestoreProjects)
				recycleBin.POST("/projects/permanently-delete", handlers.PermanentlyDeleteProjects)
			}

			// Gateway HTTP proxy — any authenticated user (read-only access)
			authorized.GET("/gateways/:gatewayID/proxy/*path", handlers.ProxyGatewayHTTP)
			authorized.HEAD("/gateways/:gatewayID/proxy/*path", handlers.ProxyGatewayHTTP)

			// Gateway forward proxy — clean URL variant, same handler
			authorized.GET("/forward/:gatewayID/*path", handlers.ProxyGatewayHTTP)
			authorized.HEAD("/forward/:gatewayID/*path", handlers.ProxyGatewayHTTP)
		}
	}
}
