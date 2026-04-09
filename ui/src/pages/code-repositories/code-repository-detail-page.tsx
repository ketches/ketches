import { BuildLogViewer } from "@/components/builds/build-log-viewer"
import { CreateBuildSettingDialog } from "@/components/code-repositories/create-build-setting-dialog"
import { EditBuildSettingDialog } from "@/components/code-repositories/edit-build-setting-dialog"
import { EditCodeRepositoryDialog } from "@/components/code-repositories/edit-code-repository-dialog"
import { RepoTopologyView } from "@/components/code-repositories/repo-topology-view"
import { UnifiedBuildDeployDialog } from "@/components/code-repositories/unified-build-deploy-dialog"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { StatCard } from "@/components/shared/stat-card"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useQueryClient } from "@tanstack/react-query"
import { CircleAlert, Copy, Footprints, Info, Play, Rocket, Share2, Telescope } from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

import type { BuildSetting } from "@/api/code-repositories"
import { CodeRepositoryBuildSettingsSection } from "./components/code-repository-build-settings-section"
import { CodeRepositoryBuildsSection } from "./components/code-repository-builds-section"
import { CodeRepositoryDeploymentsSection } from "./components/code-repository-deployments-section"
import { CodeRepositoryDetailHeader } from "./components/code-repository-detail-header"
import { CodeRepositoryOperationLogsSection } from "./components/code-repository-operation-logs-section"
import { useCodeRepositoryDetail } from "./hooks/use-code-repository-detail"

export function CodeRepositoryDetailPage() {
  const { repoId } = useParams<{ repoId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole === "viewer"
  const currentTab = searchParams.get("tab") || "overview"

  const [triggerBuildDialogOpen, setTriggerBuildDialogOpen] = React.useState(false)
  const [selectedBuildSettingId, setSelectedBuildSettingId] = React.useState<string | undefined>(undefined)
  const [selectedBuildId, setSelectedBuildId] = React.useState<string | undefined>(undefined)
  const [logBuildId, setLogBuildId] = React.useState<string | null>(null)
  const [addConfigOpen, setAddConfigOpen] = React.useState(false)
  const [editConfigOpen, setEditConfigOpen] = React.useState(false)
  const [editingConfig, setEditingConfig] = React.useState<BuildSetting | null>(null)
  const [deleteConfigDialogOpen, setDeleteConfigDialogOpen] = React.useState(false)
  const [deletingConfig, setDeletingConfig] = React.useState<BuildSetting | null>(null)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)

  const detail = useCodeRepositoryDetail(repoId)

  React.useEffect(() => {
    if (!triggerBuildDialogOpen) {
      setSelectedBuildSettingId(undefined)
      setSelectedBuildId(undefined)
    }
  }, [triggerBuildDialogOpen])

  if (!repoId) {
    return (
      <NotFoundPage
        resourceType="Code Repository"
        backHref="/code-repositories"
        backLabel="Back to Code Repositories"
      />
    )
  }

  if (detail.isLoading) {
    return (
      <div className="flex flex-1 flex-col gap-6 animate-pulse">
        <div className="flex flex-col gap-2">
          <Skeleton className="h-4 w-32" />
          <div className="flex items-start justify-between">
            <div className="flex items-center gap-4">
              <Skeleton className="h-14 w-14 rounded-lg" />
              <div className="space-y-2">
                <Skeleton className="h-8 w-48" />
                <Skeleton className="h-4 w-64" />
              </div>
            </div>
          </div>
        </div>
        <div className="space-y-4">
          <Skeleton className="h-10 w-full max-w-50" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  if (!detail.repo) {
    if (detail.isRepoForbidden) {
      return (
        <EmptyState
          title="No permission"
          description="You do not have permission to view this code repository."
          icon={CircleAlert}
        />
      )
    }

    return (
      <NotFoundPage
        resourceType="Code Repository"
        backHref="/code-repositories"
        backLabel="Back to Code Repositories"
      />
    )
  }

  return (
    <div className="flex flex-1 flex-col gap-6">
      <PageHeader items={detail.breadcrumbs} />

      <CodeRepositoryDetailHeader
        repo={detail.repo}
        safeRepos={detail.safeRepos}
        isViewer={isViewer}
        hasBuildSettings={detail.buildSettings.length > 0}
        onSelectRepo={(selectedRepoId) => navigate(`/code-repositories/${selectedRepoId}`)}
        onEdit={() => setEditDialogOpen(true)}
        onBuild={() => {
          setSelectedBuildSettingId(undefined)
          setSelectedBuildId(undefined)
          setTriggerBuildDialogOpen(true)
        }}
      />

      <Tabs value={currentTab} onValueChange={(value) => setSearchParams({ tab: value }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview"><Telescope />Overview</TabsTrigger>
          <TabsTrigger value="topology"><Share2 />Topology</TabsTrigger>
          <TabsTrigger value="operations"><Footprints />Operations</TabsTrigger>
          <TabsTrigger value="build"><Play />Build</TabsTrigger>
          <TabsTrigger value="deploy"><Rocket />Deploy</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-2 space-y-4">
          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-sm">
                <Info className="h-4 w-4" />
                Repository Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Git URL</p>
                  <div className="flex items-center gap-2">
                    <p className="break-all font-mono text-sm">{detail.repo.git_repo_url}</p>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="opacity-0 transition-opacity group-hover/card:opacity-100"
                      onClick={(event) => {
                        event.stopPropagation()
                        if (detail.repo) {
                          navigator.clipboard.writeText(detail.repo.git_repo_url)
                          toast.success("Git URL copied to clipboard")
                        }
                      }}
                    >
                      <Copy />
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
            {detail.overviewStats.map((stat) => (
              <StatCard
                key={stat.title}
                title={stat.title}
                value={stat.value}
                icon={stat.icon}
                description={stat.description}
                color={stat.color}
              />
            ))}
          </div>
        </TabsContent>

        <TabsContent value="topology" className="mt-2 space-y-4">
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-sm">
                <Share2 className="h-4 w-4" />
                Topology
              </CardTitle>
            </CardHeader>
            <CardContent>
              <RepoTopologyView repoId={detail.repo.id} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="build" className="mt-2 space-y-4">
          <CodeRepositoryBuildSettingsSection
            buildSettings={detail.buildSettings}
            isLoading={detail.buildSettingsLoading}
            isViewer={isViewer}
            onCreate={() => setAddConfigOpen(true)}
            onEdit={(setting) => {
              setEditingConfig(setting)
              setEditConfigOpen(true)
            }}
            onBuild={(setting) => {
              setSelectedBuildSettingId(setting.id)
              setSelectedBuildId(undefined)
              setTriggerBuildDialogOpen(true)
            }}
            onDelete={(setting) => {
              setDeletingConfig(setting)
              setDeleteConfigDialogOpen(true)
            }}
          />

          <CodeRepositoryBuildsSection
            builds={detail.builds}
            isLoading={detail.buildsLoading}
            isViewer={isViewer}
            retryingBuildId={detail.retryingBuildId}
            isRetryPending={detail.retryBuildMutation.isPending}
            settingNameById={detail.settingNameById}
            onViewLogs={setLogBuildId}
            onRetry={detail.retryBuildMutation.mutate}
            onDeploy={(buildId, buildSettingId) => {
              setSelectedBuildId(buildId)
              setSelectedBuildSettingId(buildSettingId)
              setTriggerBuildDialogOpen(true)
            }}
          />
        </TabsContent>

        <TabsContent value="deploy" className="mt-2 space-y-4">
          <CodeRepositoryDeploymentsSection
            deployments={detail.deployments}
            isLoading={detail.deploymentsLoading}
            onOpenApplication={(appId) => navigate(`/applications/${appId}`)}
          />
        </TabsContent>

        <TabsContent value="operations" className="mt-2 space-y-4">
          <CodeRepositoryOperationLogsSection
            operationLogs={detail.operationLogsResponse?.items ?? []}
            isLoading={detail.operationLogsLoading}
            isFetching={detail.operationLogsFetching}
            pagination={detail.operationLogsPagination}
            onPaginationChange={detail.setOperationLogsPagination}
            totalCount={detail.operationLogsResponse?.pagination.total ?? 0}
          />
        </TabsContent>
      </Tabs>

      <UnifiedBuildDeployDialog
        open={triggerBuildDialogOpen}
        onOpenChange={setTriggerBuildDialogOpen}
        repoId={repoId}
        projectId={detail.repo.project_id}
        preSelectedConfigId={selectedBuildSettingId}
        preSelectedBuildId={selectedBuildId}
      />

      <EditCodeRepositoryDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        repo={detail.repo}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ["code-repository", repoId] })}
      />

      <EditBuildSettingDialog
        open={editConfigOpen}
        onOpenChange={setEditConfigOpen}
        repoId={repoId}
        setting={editingConfig}
        onSuccess={() => {
          setEditingConfig(null)
          queryClient.invalidateQueries({ queryKey: ["build-settings", repoId] })
        }}
      />

      <CreateBuildSettingDialog
        open={addConfigOpen}
        onOpenChange={setAddConfigOpen}
        repoId={repoId}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: ["build-settings", repoId] })}
      />

      {logBuildId && (
        <Dialog open={!!logBuildId} onOpenChange={() => setLogBuildId(null)}>
          <DialogContent className="flex w-full flex-col sm:max-h-[90vh] sm:max-w-[90vw]">
            <DialogHeader>
              <DialogTitle>Build logs</DialogTitle>
            </DialogHeader>
            <div className="min-h-0 flex-1 overflow-hidden">
              <BuildLogViewer buildId={logBuildId} repoId={repoId} />
            </div>
          </DialogContent>
        </Dialog>
      )}

      <AlertDialog open={deleteConfigDialogOpen} onOpenChange={setDeleteConfigDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove Build Setting</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingConfig
                ? `Remove build setting "${deletingConfig.name}"? This action cannot be undone.`
                : "Are you sure you want to remove this build setting?"}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (deletingConfig) {
                  detail.deleteBuildSettingMutation.mutate(deletingConfig.id)
                }
                setDeleteConfigDialogOpen(false)
                setDeletingConfig(null)
              }}
              variant="destructive"
            >
              Remove
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
