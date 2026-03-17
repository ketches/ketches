package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/ketches/ketches/internal/app"
	"github.com/ketches/ketches/internal/handlers"
	"github.com/ketches/ketches/internal/middlewares"
	"github.com/ketches/ketches/internal/openapi"
)

func SetupRoutes(r *gin.Engine) {
	r.Use(middlewares.CORS())

	setupV1Routes(r)
	setupForwardRoutes(r)

	openapi.RegisterRoutes(r, openapi.Config{
		Title:       "Ketches API",
		Description: "Auto-generated from Gin route table.",
		Version:     app.Version,
		ServerURL:   "/",
	})
}

func setupV1Routes(r *gin.Engine) {
	v1 := r.Group("/api/v1")
	v1.Use(middlewares.OperationLog())
	{
		v1.GET("/version", handlers.GetVersion)

		users := v1.Group("/users")
		{
			users.POST("/sign-up", handlers.SignUp)
			users.POST("/sign-in", handlers.SignIn)
		}

		authorized := v1.Group("")
		authorized.Use(middlewares.Auth())
		{
			authorized.GET("/activities", handlers.ListActivities)
			authorized.GET("/operation-logs", middlewares.AdminOnly(), handlers.ListOperationLogs)
			authorized.GET("/operation-logs/settings", middlewares.AdminOnly(), handlers.GetOperationLogSettings)
			authorized.PUT("/operation-logs/settings", middlewares.AdminOnly(), handlers.UpdateOperationLogSettings)
			authorized.GET("/platform-settings/audit-logs", middlewares.AdminOnly(), handlers.ListPlatformAuditLogs)
			authorized.GET("/platform-update/config", middlewares.AdminOnly(), handlers.GetPlatformUpdateConfig)
			authorized.PUT("/platform-update/config", middlewares.AdminOnly(), handlers.UpdatePlatformUpdateConfig)
			authorized.GET("/platform-update/status", middlewares.AdminOnly(), handlers.GetPlatformUpdateStatus)
			authorized.POST("/platform-update/check", middlewares.AdminOnly(), handlers.CheckPlatformUpdate)
			authorized.POST("/platform-update/rollout", middlewares.AdminOnly(), handlers.TriggerPlatformRollout)
			authorized.GET("/platform-settings/branding", middlewares.AdminOnly(), handlers.GetPlatformBranding)
			authorized.PUT("/platform-settings/branding", middlewares.AdminOnly(), handlers.UpdatePlatformBranding)

			// ── Notifications (user-scoped) ────────────────────────────
			notifications := authorized.Group("/notifications")
			{
				notifications.GET("", handlers.ListNotifications)
				notifications.GET("/unread-count", handlers.GetUnreadCount)
				notifications.POST("/read-all", handlers.MarkAllNotificationsRead)
				notifications.POST("/:notificationID/action", handlers.HandleNotificationAction)
			}

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
				projects.GET("", handlers.ListProjects)
				projects.GET("/simple", handlers.ListProjectsSimple)
				projects.POST("", handlers.CreateProject) // create project — no existing role

				projectsRead := projects.Group("", middlewares.RequireProjectRole(app.ProjectRoleViewer))
				projectsRead.GET("/:projectID", handlers.GetProject)
				projectsRead.GET("/:projectID/members", handlers.ListProjectMembers)
				projectsRead.GET("/:projectID/invitable-users", handlers.ListInvitableUsers)
				projectsRead.GET("/:projectID/envs", handlers.ListEnvs)
				projectsRead.GET("/:projectID/envs/simple", handlers.ListEnvsSimple)
				projectsRead.GET("/:projectID/envs/namespace-availability", handlers.CheckEnvNamespaceAvailability)
				projectsRead.GET("/:projectID/container-registries", handlers.ListProjectContainerRegistries)
				projectsRead.GET("/:projectID/container-registries/simple", handlers.ListProjectContainerRegistriesSimple)
				projectsRead.GET("/:projectID/code-repositories", handlers.ListCodeRepositories)
				projectsRead.GET("/:projectID/code-repositories/simple", handlers.ListCodeRepositoriesSimple)
				projectsRead.GET("/:projectID/plugins", handlers.ListPlugins)
				projectsRead.GET("/:projectID/plugins/simple", handlers.ListPluginsSimple)
				projectsRead.GET("/:projectID/plugins/:pluginID", handlers.GetPlugin)
				projectsRead.GET("/:projectID/plugins/:pluginID/installed-apps", handlers.GetPluginInstalledApps)

				// Write (require at least developer role)
				projectsWrite := projects.Group("", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
				projectsWrite.PUT("/:projectID", handlers.UpdateProject)
				projectsWrite.DELETE("/:projectID", handlers.DeleteProject)
				projectsWrite.POST("/:projectID/members", handlers.InviteProjectMembers)
				projectsWrite.DELETE("/:projectID/members", handlers.RemoveProjectMember)
				projectsWrite.POST("/:projectID/envs", handlers.CreateEnv)
				projectsWrite.POST("/:projectID/container-registries", handlers.CreateProjectContainerRegistry)
				projectsWrite.POST("/:projectID/code-repositories", handlers.CreateCodeRepository)
				projectsWrite.POST("/:projectID/plugins", handlers.CreatePlugin)
				projectsWrite.PUT("/:projectID/plugins/:pluginID", handlers.UpdatePlugin)
				projectsWrite.DELETE("/:projectID/plugins/:pluginID", handlers.DeletePlugin)

				// ── Collaboration (Sprints / Requirements / Tasks) ──────
				projectsRead.GET("/:projectID/sprints", handlers.ListSprints)
				projectsRead.GET("/:projectID/sprints/:sprintID", handlers.GetSprint)
				projectsRead.GET("/:projectID/requirements", handlers.ListRequirements)
				projectsRead.GET("/:projectID/backlog", handlers.ListBacklog)
				projectsRead.GET("/:projectID/requirements/:requirementID", handlers.GetRequirement)
				projectsRead.GET("/:projectID/tasks", handlers.ListTasks)
				projectsRead.GET("/:projectID/tasks/:taskID", handlers.GetTask)
				projectsRead.GET("/:projectID/test-cases", handlers.ListTestCases)
				projectsRead.GET("/:projectID/test-cases/:testCaseID", handlers.GetTestCase)
				projectsRead.GET("/:projectID/defects", handlers.ListDefects)
				projectsRead.GET("/:projectID/defects/:defectID", handlers.GetDefect)

				projectsWrite.POST("/:projectID/sprints", handlers.CreateSprint)
				projectsWrite.PUT("/:projectID/sprints/:sprintID", handlers.UpdateSprint)
				projectsWrite.DELETE("/:projectID/sprints/:sprintID", handlers.DeleteSprint)
				projectsWrite.POST("/:projectID/requirements", handlers.CreateRequirement)
				projectsWrite.POST("/:projectID/requirements/:requirementID/children", handlers.CreateRequirementChild)
				projectsWrite.PUT("/:projectID/requirements/:requirementID", handlers.UpdateRequirement)
				projectsWrite.DELETE("/:projectID/requirements/:requirementID", handlers.DeleteRequirement)
				projectsWrite.POST("/:projectID/requirements/:requirementID/transition", handlers.TransitionRequirement)
				projectsWrite.POST("/:projectID/backlog/reorder", handlers.BacklogReorder)
				projectsWrite.POST("/:projectID/backlog/plan-to-sprint", handlers.BacklogPlanToSprint)
				projectsWrite.POST("/:projectID/backlog/return", handlers.BacklogReturn)
				projectsWrite.POST("/:projectID/tasks", handlers.CreateTask)
				projectsWrite.POST("/:projectID/tasks/:taskID/children", handlers.CreateTaskChild)
				projectsWrite.PUT("/:projectID/tasks/:taskID", handlers.UpdateTask)
				projectsWrite.DELETE("/:projectID/tasks/:taskID", handlers.DeleteTask)
				projectsWrite.POST("/:projectID/tasks/:taskID/transition", handlers.TransitionTask)
				projectsWrite.POST("/:projectID/test-cases", handlers.CreateTestCase)
				projectsWrite.PUT("/:projectID/test-cases/:testCaseID", handlers.UpdateTestCase)
				projectsWrite.DELETE("/:projectID/test-cases/:testCaseID", handlers.DeleteTestCase)
				projectsWrite.POST("/:projectID/test-cases/:testCaseID/runs", handlers.CreateTestRun)
				projectsWrite.POST("/:projectID/defects", handlers.CreateDefect)
				projectsWrite.PUT("/:projectID/defects/:defectID", handlers.UpdateDefect)
				projectsWrite.DELETE("/:projectID/defects/:defectID", handlers.DeleteDefect)
				projectsWrite.POST("/:projectID/defects/:defectID/transition", handlers.TransitionDefect)
			}

			// ── Code Repositories ─────────────────────────────────────
			codeRepos := authorized.Group("/code-repositories")
			{
				codeReposRead := codeRepos.Group("", middlewares.RequireProjectRole(app.ProjectRoleViewer))
				codeReposRead.GET("/:repoID", handlers.GetCodeRepository)
				codeReposRead.GET("/:repoID/topology", handlers.GetCodeRepositoryTopology)
				codeReposRead.GET("/:repoID/refs", handlers.ListCodeRepositoryRefs)
				codeReposRead.GET("/:repoID/container-registries", handlers.ListCodeRepositoryContainerRegistries)
				codeReposRead.GET("/:repoID/build-settings", handlers.ListRepoBuildSettings)
				codeReposRead.GET("/:repoID/build-settings/:settingID", handlers.GetRepoBuildSetting)
				codeReposRead.GET("/:repoID/builds", handlers.ListCodeRepositoryBuilds)
				codeReposRead.GET("/:repoID/builds/:buildID", handlers.GetCodeRepositoryBuild)
				codeReposRead.GET("/:repoID/builds/:buildID/logs", handlers.StreamCodeRepositoryBuildLogs)
				codeReposRead.GET("/:repoID/deployments", handlers.ListBuildDeployments)
				codeReposRead.GET("/:repoID/build-settings/:settingID/deployed-apps", handlers.ListDeployedAppsByEnvironment)
				codeReposRead.GET("/:repoID/operation-logs", handlers.ListCodeRepositoryOperationLogs)

				// Write (require at least developer role)
				codeReposWrite := codeRepos.Group("", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
				codeReposWrite.PUT("/:repoID", handlers.UpdateCodeRepository)
				codeReposWrite.DELETE("/:repoID", handlers.DeleteCodeRepository)
				codeReposWrite.POST("/:repoID/test-git", handlers.TestCodeRepositoryGit)
				codeReposWrite.POST("/:repoID/build-settings", handlers.CreateRepoBuildSetting)
				codeReposWrite.PUT("/:repoID/build-settings/:settingID", handlers.UpdateRepoBuildSetting)
				codeReposWrite.DELETE("/:repoID/build-settings/:settingID", handlers.DeleteRepoBuildSetting)
				codeReposWrite.POST("/:repoID/builds", handlers.TriggerCodeRepositoryBuild)
				codeReposWrite.POST("/:repoID/builds/:buildID/cancel", handlers.CancelCodeRepositoryBuild)
				codeReposWrite.POST("/:repoID/builds/:buildID/deploy", handlers.DeployCodeRepositoryBuild)
			}

			// ── Environments ──────────────────────────────────────────
			envs := authorized.Group("/envs")
			{
				envsRead := envs.Group("", middlewares.RequireProjectRole(app.ProjectRoleViewer))
				envsRead.GET("/:envID", handlers.GetEnv)
				envsRead.GET("/:envID/apps", handlers.ListApps)
				envsRead.GET("/:envID/apps/simple", handlers.ListAppsSimple)
				envsRead.GET("/:envID/apps/export", handlers.ExportEnvApps)
				envsRead.GET("/:envID/app-groups", handlers.ListAppGroups)
				envsRead.GET("/:envID/apps/favorites", handlers.ListFavoriteApps)
				envsRead.GET("/:envID/certificates", handlers.ListEnvCertificates)
				envsRead.GET("/:envID/apps/image-metadata", handlers.GetImageMetadata)
				envsRead.GET("/:envID/certificates/:certID", handlers.GetCertificate)
				// Write (require at least developer role)
				envsWrite := envs.Group("", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
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
				appsRead := apps.Group("", middlewares.RequireProjectRole(app.ProjectRoleViewer))
				appsRead.GET("/:appID", handlers.GetApp)
				appsRead.GET("/:appID/simple", handlers.GetAppSimple)
				appsRead.GET("/:appID/export", handlers.ExportApps)
				appsRead.GET("/:appID/available-actions", handlers.GetAppAvailableActions)
				appsRead.GET("/:appID/topology", handlers.GetAppTopology)
				appsRead.GET("/:appID/instances", handlers.ListAppInstances)
				appsRead.GET("/:appID/instances/:instanceName/events", handlers.ListAppInstanceEvents)
				appsRead.GET("/:appID/env-vars", handlers.ListAppEnvVars)
				appsRead.GET("/:appID/volumes", handlers.ListAppVolumes)
				appsRead.GET("/:appID/config-files", handlers.ListAppConfigFiles)
				appsRead.GET("/:appID/gateways", handlers.ListAppGateways)
				appsRead.GET("/:appID/plugins", handlers.ListAppPlugins)
				appsRead.GET("/:appID/builds", handlers.ListAppBuilds)
				appsRead.GET("/:appID/deployment-history", handlers.ListDeploymentHistory)
				appsRead.GET("/:appID/favorite", handlers.GetAppFavoriteStatus)
				appsRead.GET("/:appID/operation-logs", handlers.ListAppOperationLogs)
				appsRead.GET("/:appID/image-tags", handlers.ListAppImageTags)

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
				appsWrite := apps.Group("", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
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
				appsWrite.POST("/:appID/deployment-history/rollback", handlers.RollbackDeployment)
				appsWrite.POST("/:appID/favorite", handlers.AddFavoriteApp)
				appsWrite.DELETE("/:appID/favorite", handlers.RemoveFavoriteApp)
			}

			// Flat resource write routes — RequireProjectRole resolves project via resource→app→env chain.
			// ── App Groups (flat write routes) ───────────────────────────────
			appGroupsWrite := authorized.Group("/app-groups", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
			{
				appGroupsWrite.GET("/:groupID", handlers.GetAppGroup)
				appGroupsWrite.GET("/:groupID/apps", handlers.ListSpecificGroupedApps)
				appGroupsWrite.PUT("/:groupID", handlers.UpdateAppGroup)
				appGroupsWrite.DELETE("/:groupID", handlers.DeleteAppGroup)
				appGroupsWrite.POST("/:groupID/apps/:appID", handlers.AddAppToGroup)
				appGroupsWrite.DELETE("/:groupID/apps/:appID", handlers.RemoveAppFromGroup)
			}

			flatResourcesWrite := authorized.Group("", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
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
			authorized.GET("/clusters/:clusterID/certificates", handlers.ListClusterCertificates)
			authorized.GET("/clusters/:clusterID/integrations", handlers.ListClusterIntegrations)
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
				clusters.POST("/:clusterID/certificates", handlers.CreateClusterCertificate)
				clusters.GET("/:clusterID/certificates/:certID", handlers.GetCertificate)
				clusters.PUT("/:clusterID/certificates/:certID", handlers.UpdateCertificate)
				clusters.DELETE("/:clusterID/certificates/:certID", handlers.DeleteCertificate)
			}

			// Container Registries (project-scoped) — write routes enforce developer role via registryID→project chain.
			containerRegistries := authorized.Group("/container-registries")
			{
				containerRegistries.GET("/:registryID", middlewares.RequireProjectRole(app.ProjectRoleViewer), handlers.GetContainerRegistry)
				containerRegistriesWrite := containerRegistries.Group("", middlewares.RequireProjectRole(app.ProjectRoleDeveloper))
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
				recycleBin.GET("/users", middlewares.AdminOnly(), handlers.ListRecycleBinUsers)
				recycleBin.POST("/users/restore", middlewares.AdminOnly(), handlers.RestoreUsers)
				recycleBin.POST("/users/permanently-delete", middlewares.AdminOnly(), handlers.PermanentlyDeleteUsers)
				recycleBin.GET("/code-repositories", handlers.ListRecycleBinCodeRepositories)
				recycleBin.POST("/code-repositories/restore", handlers.RestoreCodeRepositories)
				recycleBin.POST("/code-repositories/permanently-delete", handlers.PermanentlyDeleteCodeRepositories)
			}

			authorized.GET("/gateways/:gatewayID/proxy/*path", middlewares.RequireProjectRole(app.ProjectRoleViewer), handlers.ProxyGatewayHTTP)
			authorized.HEAD("/gateways/:gatewayID/proxy/*path", middlewares.RequireProjectRole(app.ProjectRoleViewer), handlers.ProxyGatewayHTTP)
		}
	}
}

// setupForwardRoutes registers /forward/:gatewayID/*path at the root level (outside
// /api/v1) so that nginx can proxy /forward/* directly to the backend without a
// path prefix, keeping the browser address bar at /forward/:gatewayID.
func setupForwardRoutes(r *gin.Engine) {
	forward := r.Group("/forward")
	forward.Use(middlewares.ForwardAuth(), middlewares.RequireProjectRole(app.ProjectRoleViewer))
	{
		forward.GET("/:gatewayID/*path", handlers.ProxyGatewayHTTP)
		forward.HEAD("/:gatewayID/*path", handlers.ProxyGatewayHTTP)
	}
}
