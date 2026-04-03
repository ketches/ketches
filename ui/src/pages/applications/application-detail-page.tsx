import { appFavoritesApi } from "@/api/app-favorite"
import { appsApi, type App, type AppInstance } from "@/api/apps"
import { envsApi } from "@/api/envs"
import { operationLogsApi } from "@/api/operation-logs"
import { TopologyView } from "@/components/applications/app-topology-view"
import { AutoScalingConfig } from "@/components/applications/auto-scaling-config"
import { CommandConfig } from "@/components/applications/command-config"
import { ConfigFilesTable } from "@/components/applications/config-files-table"
import { EditAppDialog } from "@/components/applications/edit-app-dialog"
import { EnvVarTable } from "@/components/applications/env-var-table"
import { NetworkConfig } from "@/components/applications/gateway-table"
import { HealthConfig } from "@/components/applications/health-config"
import { ImageEditor } from "@/components/applications/image-editor"
import { ResourceConfig } from "@/components/applications/resource-config"
import { SchedulingConfig } from "@/components/applications/scheduling-config"
import { VolumesTable } from "@/components/applications/volumes-table"
import { BuildList } from "@/components/builds/build-list"
import { UnifiedBuildDeployDialog } from "@/components/code-repositories/unified-build-deploy-dialog"
import { DeploymentHistoryList } from "@/components/deployments/deployment-history-list"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { useTimeRange } from "@/components/monitoring/use-time-range"
import { AppPlugins } from "@/components/plugins/app-plugins"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { DropdownMenu, DropdownMenuContent, DropdownMenuGroup, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useBottomPanel } from "@/contexts/bottom-panel-context"
import type { BreadcrumbItem } from "@/contexts/breadcrumb-state"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type PaginationState, type RowSelectionState } from "@tanstack/react-table"
import { isAxiosError, type AxiosError } from "axios"
import {
  Box,
  ChevronsUpDown,
  CircleAlert,
  Cog,
  ExternalLink,
  FileCog,
  FolderGit2,
  Footprints,
  GalleryVerticalEnd,
  Hammer,
  HardDrive,
  Key,
  Network,
  Orbit,
  Puzzle,
  Ruler,
  Share2,
  Telescope,
} from "lucide-react"
import * as React from "react"
import { Link, useNavigate, useParams } from "react-router-dom"
import { toast } from "sonner"

import { ApplicationDetailHeader } from "./components/application-detail-header"
import { ApplicationOperationsTab } from "./components/application-operations-tab"
import { ApplicationOverviewTab } from "./components/application-overview-tab"
import { InstanceEventsDialog } from "./components/instance-events-dialog"
import { useApplicationDetailTabs } from "./hooks/use-application-detail-tabs"

function shouldPollAppDetail(status: string | undefined): boolean {
  switch ((status ?? "").toLowerCase()) {
    case "starting":
    case "updating":
    case "stopping":
    case "debugging":
      return true
    default:
      return false
  }
}

export function ApplicationDetailPage() {
  const navigate = useNavigate()
  const { appId } = useParams()
  const queryClient = useQueryClient()

  const { currentTab, setCurrentTab, viewMode, setViewMode } = useApplicationDetailTabs()
  const { openPanel } = useBottomPanel()
  const { activeProjectId, activeEnvId, activeProjectName, setActiveContextWithNames } = useProjectStore()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const { timeRange, setTimeRange, rangeSeconds, step } = useTimeRange()

  const isAdmin = useAuthStore((state) => state.user?.role === "admin")

  const { data: app, isLoading, error } = useQuery<App>({
    queryKey: ['app', appId],
    queryFn: () => appsApi.get(appId!),
    enabled: !!appId,
    refetchInterval: (query) => shouldPollAppDetail((query.state.data as App | undefined)?.status) ? 5000 : false,
  })

  const currentEnv = app?.env

  const projectIdToUse = currentEnv?.project_id || activeProjectId
  const projectNameToUse = currentEnv?.project_name || activeProjectName

  const { data: envs = [] } = useQuery({
    queryKey: ['envs-simple', projectIdToUse],
    queryFn: () => envsApi.listSimpleByProject(projectIdToUse!),
    enabled: !!projectIdToUse,
    staleTime: 5 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  })

  const safeEnvs = Array.isArray(envs) ? envs : []

  const { data: apps = [] } = useQuery({
    queryKey: ['apps-simple', app?.env_id],
    queryFn: () => appsApi.listSimple(app!.env_id!),
    enabled: !!app?.env_id,
    staleTime: 5 * 60 * 1000,
    gcTime: 15 * 60 * 1000,
  })
  const { data: favoriteStatus } = useQuery({
    queryKey: ['app-favorite', appId],
    queryFn: () => appFavoritesApi.getFavoriteStatus(appId!),
    enabled: !!appId,
    staleTime: 60 * 1000,
  })

  const toggleFavMutation = useMutation({
    mutationFn: (): Promise<void> =>
      favoriteStatus?.is_favorite
        ? appFavoritesApi.removeFavorite(appId!)
        : appFavoritesApi.addFavorite(appId!).then(() => undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app-favorite', appId] })
      queryClient.invalidateQueries({ queryKey: ['app-favorites'] })
      toast.success(favoriteStatus?.is_favorite ? 'Removed from favorites' : 'Added to favorites')
    },
  })

  const safeApps = Array.isArray(apps) ? apps : []

  const [rowSelection, setRowSelection] = React.useState<RowSelectionState>({})
  const [instancePagination, setInstancePagination] = React.useState({ pageIndex: 0, pageSize: 10 })
  const [operationLogsPagination, setOperationLogsPagination] = React.useState<PaginationState>({ pageIndex: 0, pageSize: 10 })
  const [isEditImageDialogOpen, setIsEditImageDialogOpen] = React.useState(false)
  const [isEditAppDialogOpen, setIsEditAppDialogOpen] = React.useState(false)
  const [isUnifiedBuildDialogOpen, setIsUnifiedBuildDialogOpen] = React.useState(false)
  const [selectedInstanceForEvents, setSelectedInstanceForEvents] = React.useState<string | null>(null)
  const [deleteInstanceDialogOpen, setDeleteInstanceDialogOpen] = React.useState(false)
  const [deletingInstanceName, setDeletingInstanceName] = React.useState<string | null>(null)
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = React.useState(false)
  const [selectedInstanceNames, setSelectedInstanceNames] = React.useState<string[]>([])
  const [metricsInstance, setMetricsInstance] = React.useState<string | null>(null)
  const hasSyncedContextFromAppRef = React.useRef(false)

  React.useEffect(() => {
    hasSyncedContextFromAppRef.current = false
  }, [appId])

  React.useEffect(() => {
    if (!hasSyncedContextFromAppRef.current && currentEnv && app?.env_id) {
      if (activeProjectId !== currentEnv.project_id || activeEnvId !== app.env_id) {
        setActiveContextWithNames(currentEnv.project_id, currentEnv.project_name, app.env_id, currentEnv.name)
      }
      hasSyncedContextFromAppRef.current = true
    }
  }, [currentEnv, app?.env_id, activeProjectId, activeEnvId, setActiveContextWithNames])

  const deleteInstanceMutation = useMutation({
    mutationFn: (instanceName: string) => appsApi.deleteInstance(appId!, instanceName),
    onSuccess: () => {
      toast.success("Instance deletion initiated")
      queryClient.invalidateQueries({ queryKey: ['app-instances', appId] })
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to delete instance", {
        description: error.response?.data?.error || error.message
      })
    }
  })

  const { data: instances = [] } = useQuery({
    queryKey: ['app-instances', appId],
    queryFn: () => appsApi.listInstances(appId!),
    enabled: !!appId && currentTab === 'overview',
    refetchInterval: currentTab === 'overview' && shouldPollAppDetail(app?.status) ? 5000 : false,
  })

  const safeInstances = React.useMemo(() => (Array.isArray(instances) ? instances : []), [instances])
  const selectedInstanceIndices = React.useMemo(
    () => Object.entries(rowSelection)
      .filter(([, isSelected]) => isSelected)
      .map(([key]) => Number(key))
      .filter((index) => Number.isInteger(index) && index >= 0),
    [rowSelection]
  )
  const selectedInstanceNamesFromRows = React.useMemo(
    () => selectedInstanceIndices
      .map((index) => safeInstances[index]?.instance_name)
      .filter((instanceName): instanceName is string => Boolean(instanceName)),
    [safeInstances, selectedInstanceIndices]
  )

  const { data: operationLogsResponse, isLoading: operationLogsLoading, isFetching: operationLogsFetching } = useQuery({
    queryKey: ['app-operation-logs', appId, operationLogsPagination.pageIndex, operationLogsPagination.pageSize],
    queryFn: () => operationLogsApi.listAppOperationLogs(appId!, {
      page: operationLogsPagination.pageIndex + 1,
      page_size: operationLogsPagination.pageSize,
    }),
    enabled: !!appId && currentTab === 'operations',
  })

  const bulkDeleteMutation = useMutation({
    mutationFn: async (instanceNames: string[]) => {
      return Promise.all(instanceNames.map(name => appsApi.deleteInstance(appId!, name)))
    },
    onSuccess: () => {
      toast.success("Bulk deletion initiated")
      queryClient.invalidateQueries({ queryKey: ['app-instances', appId] })
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to delete instances", {
        description: error.response?.data?.error || error.message
      })
    }
  })

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 gap-6 animate-pulse">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-4 w-32" />
          <div className="flex justify-between items-start">
            <div className="flex items-center gap-4">
              <Skeleton className="h-14 w-14 rounded-lg" />
              <div className="space-y-2">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-4 w-32" />
              </div>
            </div>
            <div className="flex flex-col items-end gap-2">
              <Skeleton className="h-6 w-20" />
              <div className="flex gap-2">
                <Skeleton className="h-9 w-24" />
                <Skeleton className="h-9 w-24" />
              </div>
            </div>
          </div>
        </div>
        <div className="space-y-4">
          <Skeleton className="h-10 w-100" />
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
            <Skeleton className="h-32 w-full" />
          </div>
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  if (error || !app) {
    if (isAxiosError(error) && error.response?.status === 403) {
      return (
        <EmptyState
          title="No permission"
          description="You do not have permission to view this application."
          icon={CircleAlert}
        />
      )
    }

    return (
      <NotFoundPage
        resourceType="Application"
        backHref="/applications"
        backLabel="Back to Applications"
      />
    )
  }

  const breadcrumbs: BreadcrumbItem[] = isAdmin ? [
    { label: "Projects", icon: GalleryVerticalEnd, href: '/projects' },
    {
      label: projectNameToUse || "Project",
      icon: GalleryVerticalEnd,
      href: `/projects/${projectIdToUse}`,
    }] : [{ label: "Applications", icon: Box }
  ]

  breadcrumbs.push({
    label: currentEnv?.name || "Environment",
    icon: Orbit,
    href: '/applications',
    dropdown: safeEnvs.length > 1 ? (
      <DropdownMenu>
        <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
        <DropdownMenuContent align="start" className="w-fit">
          <DropdownMenuGroup>
            {safeEnvs.map(env => (
              <DropdownMenuItem
                key={env.id}
                onClick={() => {
                  setActiveContextWithNames(activeProjectId, activeProjectName, env.id, env.name)
                  navigate('/applications')
                }}
              >
                <Orbit className="h-4 w-4" />
                {env.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    ) : undefined
  },
    {
      label: app.name,
      icon: Box,
      dropdown: safeApps.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-fit">
            <DropdownMenuGroup>
              {safeApps.map(appItem => (
                <DropdownMenuItem
                  key={appItem.id}
                  onClick={() => navigate(`/applications/${appItem.id}`)}
                >
                  <Box className="h-4 w-4" />
                  {appItem.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined
    })

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <ApplicationDetailHeader
        app={app}
        isViewer={isViewer}
        isFavorite={favoriteStatus?.is_favorite ?? false}
        onToggleFavorite={() => toggleFavMutation.mutate()}
        onEdit={() => setIsEditAppDialogOpen(true)}
        onDeleteSuccess={() => navigate("/applications")}
      />

      <Tabs value={currentTab} onValueChange={setCurrentTab}>
        <TabsList>
          <TabsTrigger value="overview"><Telescope />Overview</TabsTrigger>
          <TabsTrigger value="topology"><Share2 />Topology</TabsTrigger>
          <TabsTrigger value="operations"><Footprints />Operations</TabsTrigger>
          <TabsTrigger value="resources"><Ruler />Resources</TabsTrigger>
          {!isViewer && <TabsTrigger value="env-vars"><Key />Env Vars</TabsTrigger>}
          {!isViewer && <TabsTrigger value="config-files"><FileCog />Config Files</TabsTrigger>}
          {!isViewer && <TabsTrigger value="volumes"><HardDrive />Volumes</TabsTrigger>}
          <TabsTrigger value="gateways"><Network />Gateways</TabsTrigger>
          {!isViewer && <TabsTrigger value="plugins"><Puzzle />Plugins</TabsTrigger>}
          {!isViewer && <TabsTrigger value="build"><Hammer />Deploy</TabsTrigger>}
          {!isViewer && <TabsTrigger value="advanced"><Cog />Advanced</TabsTrigger>}
        </TabsList>

        <TabsContent value="overview" className="group/card space-y-4 mt-2">
          <ApplicationOverviewTab
            app={app}
            currentEnv={currentEnv}
            projectIdToUse={projectIdToUse || undefined}
            isViewer={isViewer}
            isLoading={isLoading}
            viewMode={viewMode}
            onViewModeChange={(value) => {
              setViewMode(value)
              setInstancePagination({ pageIndex: 0, pageSize: 10 })
            }}
            rowSelection={rowSelection}
            onRowSelectionChange={setRowSelection}
            instancePagination={instancePagination}
            onInstancePaginationChange={setInstancePagination}
            safeInstances={safeInstances}
            selectedInstanceNamesFromRows={selectedInstanceNamesFromRows}
            selectedInstanceCount={selectedInstanceIndices.length}
            timeRange={timeRange}
            onTimeRangeChange={setTimeRange}
            rangeSeconds={rangeSeconds}
            step={step}
            deleteInstancePending={deleteInstanceMutation.isPending}
            metricsInstance={metricsInstance}
            onMetricsInstanceChange={setMetricsInstance}
            onOpenLogs={(instance, containerName) => {
              openPanel({
                type: "logs",
                appId: app.id,
                appName: app.name,
                instanceName: instance.instance_name,
                containerName,
                containers: instance.containers,
                initContainers: instance.init_containers,
              })
            }}
            onOpenTerminal={(instance, containerName) => {
              openPanel({
                type: "terminal",
                appId: app.id,
                appName: app.name,
                instanceName: instance.instance_name,
                containerName,
                containers: instance.containers,
                initContainers: instance.init_containers,
              })
            }}
            onOpenFiles={(instance, containerName) => {
              openPanel({
                type: "files",
                appId: app.id,
                appName: app.name,
                instanceName: instance.instance_name,
                containerName,
                containers: instance.containers,
                initContainers: instance.init_containers,
              })
            }}
            onOpenInstanceEvents={setSelectedInstanceForEvents}
            onRequestDeleteInstance={(instanceName) => {
              setDeletingInstanceName(instanceName)
              setDeleteInstanceDialogOpen(true)
            }}
            onRequestBulkDelete={(instanceNames) => {
              setSelectedInstanceNames(instanceNames)
              setBulkDeleteDialogOpen(true)
            }}
            onEditImage={() => setIsEditImageDialogOpen(true)}
          />
        </TabsContent>

        <TabsContent value="topology" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Share2 className="h-4 w-4" />
                Topology
              </CardTitle>
              <CardDescription>
                Resource topology of the application
              </CardDescription>
            </CardHeader>
            <CardContent>
              <TopologyView appId={app.id} isViewer={isViewer} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="operations" className="space-y-4 mt-2">
          <ApplicationOperationsTab
            items={operationLogsResponse?.items ?? []}
            isLoading={operationLogsLoading}
            isFetching={operationLogsFetching}
            pagination={operationLogsPagination}
            onPaginationChange={setOperationLogsPagination}
            totalCount={operationLogsResponse?.pagination.total ?? 0}
          />
        </TabsContent>

        <TabsContent value="resources" className="space-y-4 mt-2">
          <ResourceConfig app={app} />
        </TabsContent>

        <TabsContent value="gateways" className="space-y-4 mt-2">
          <NetworkConfig app={app} />
        </TabsContent>

        {!isViewer && (
          <TabsContent value="env-vars" className="space-y-4 mt-2">
            <EnvVarTable app={app} />
          </TabsContent>
        )}

        {!isViewer && (
          <TabsContent value="volumes" className="space-y-4 mt-2">
            <VolumesTable app={app} />
          </TabsContent>
        )}

        {!isViewer && (
          <TabsContent value="config-files" className="space-y-4 mt-2">
            <ConfigFilesTable app={app} />
          </TabsContent>
        )}

        {!isViewer && (
          <TabsContent value="plugins" className="space-y-4 mt-2">
            <AppPlugins appId={app.id} projectId={projectIdToUse!} />
          </TabsContent>
        )}

        {!isViewer && (
          <TabsContent value="build" className="space-y-4 mt-2">
            {app.code_repository_id && (
              <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
                <CardHeader>
                  <CardTitle className="text-sm flex items-center gap-2">
                    <FolderGit2 className="h-4 w-4" />
                    Code Repository
                  </CardTitle>
                  <CardDescription>
                    This application was deployed from a code repository. View build history and trigger new builds.
                  </CardDescription>
                </CardHeader>
                <CardContent className="flex items-center gap-2">
                  <Button variant="outline" render={<Link to={`/code-repositories/${app.code_repository_id}`} className="flex items-center whitespace-nowrap" />}>
                    <ExternalLink className="h-4 w-4" />
                    View in Code Repository
                  </Button>
                  {!isViewer && (
                    <Button onClick={() => setIsUnifiedBuildDialogOpen(true)} className="flex items-center">
                      <Hammer className="h-4 w-4" />
                      Build & Deploy
                    </Button>
                  )}
                </CardContent>
              </Card>
            )}
            <BuildList appId={app.id} />
            <DeploymentHistoryList appId={app.id} />
          </TabsContent>
        )}

        {!isViewer && (
          <TabsContent value="advanced" className="space-y-4 mt-2">
            <CommandConfig app={app} />
            <AutoScalingConfig app={app} />
            <HealthConfig app={app} />
            <SchedulingConfig app={app} />
          </TabsContent>
        )}
      </Tabs>

      <ImageEditor
        app={app}
        open={isEditImageDialogOpen}
        onOpenChange={setIsEditImageDialogOpen}
      />

      <EditAppDialog
        app={app}
        open={isEditAppDialogOpen}
        onOpenChange={setIsEditAppDialogOpen}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ['app', app.id] })}
      />

      {app && app.code_repository_id && projectIdToUse && (
        <UnifiedBuildDeployDialog
          open={isUnifiedBuildDialogOpen}
          onOpenChange={setIsUnifiedBuildDialogOpen}
          repoId={app.code_repository_id}
          projectId={projectIdToUse}
          preSelectedDeployEnvId={app.env_id}
          preSelectedDeployAppId={app.id}
        />
      )}

      {app && selectedInstanceForEvents && (
        <InstanceEventsDialog
          appId={app.id}
          instanceName={selectedInstanceForEvents}
          open={!!selectedInstanceForEvents}
          onOpenChange={(open) => !open && setSelectedInstanceForEvents(null)}
        />
      )}

      <AlertDialog open={deleteInstanceDialogOpen} onOpenChange={setDeleteInstanceDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Instance</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingInstanceName
                ? `Are you sure you want to delete instance "${deletingInstanceName}"?`
                : "Are you sure you want to delete this instance?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingInstanceName) {
                  deleteInstanceMutation.mutate(deletingInstanceName)
                }
                setDeleteInstanceDialogOpen(false)
                setDeletingInstanceName(null)
              }}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={bulkDeleteDialogOpen} onOpenChange={setBulkDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Instances</AlertDialogTitle>
            <AlertDialogDescription>
              {selectedInstanceNames.length > 0
                ? `Are you sure you want to delete ${selectedInstanceNames.length} instances? This action cannot be undone.`
                : "Are you sure you want to delete these instances?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (selectedInstanceNames.length > 0) {
                  bulkDeleteMutation.mutate(selectedInstanceNames)
                  setRowSelection({})
                }
                setBulkDeleteDialogOpen(false)
                setSelectedInstanceNames([])
              }}
              variant="destructive"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default ApplicationDetailPage
