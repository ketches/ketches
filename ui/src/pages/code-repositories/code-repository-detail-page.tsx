import { codeRepositoriesApi, type BuildSetting } from "@/api/code-repositories"
import { projectsApi } from "@/api/projects"
import { envsApi } from "@/api/envs"
import { BuildLogViewer } from "@/components/builds/build-log-viewer"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { CreateBuildSettingDialog } from "@/components/code-repositories/create-build-setting-dialog"
import { EditBuildSettingDialog } from "@/components/code-repositories/edit-build-setting-dialog"
import { EditCodeRepositoryDialog } from "@/components/code-repositories/edit-code-repository-dialog"
import { RepoTopologyView } from "@/components/code-repositories/repo-topology-view"
import { UnifiedBuildDeployDialog } from "@/components/code-repositories/unified-build-deploy-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { StatCard } from "@/components/shared/stat-card"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useProjectRole } from "@/hooks/useProjectRole"
import { formatDate } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { isAxiosError, type AxiosError } from "axios"
import {
  CheckCircle,
  ChevronsUpDown,
  CircleAlert,
  Clock,
  Copy,
  ExternalLink,
  FileClock,
  FolderGit2,
  GalleryVerticalEnd,
  Hammer,
  History,
  Info,
  Loader2,
  Pencil,
  Play,
  Plus,
  Rocket,
  RotateCw,
  Share2,
  Telescope,
  Trash2
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

function DeploymentErrorPopover({ errorMessage }: { errorMessage: string }) {
  const [open, setOpen] = React.useState(false)

  return (
    < Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger
        render={<button type="button" className="inline-flex items-center" />}
      >
        <BuildStatusBadge status="failed" />
      </PopoverTrigger>
      <PopoverContent side="top" align="start" className="w-md max-w-[calc(100vw-2rem)] gap-2">
        <p className="text-xs font-medium text-destructive">Deployment failed</p>
        <p className="text-xs text-muted-foreground wrap-break-word whitespace-pre-wrap">{errorMessage}</p>
      </PopoverContent>
    </Popover >
  )
}

export function CodeRepositoryDetailPage() {
  const { repoId } = useParams<{ repoId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()

  const currentTab = searchParams.get("tab") || "overview"
  const [triggerBuildDialogOpen, setTriggerBuildDialogOpen] = React.useState(false)
  const [selectedBuildSettingId, setSelectedBuildSettingId] = React.useState<string | undefined>(undefined)
  const [selectedBuildId, setSelectedBuildId] = React.useState<string | undefined>(undefined)
  const [retryingBuildId, setRetryingBuildId] = React.useState<string | null>(null)
  const [logBuildId, setLogBuildId] = React.useState<string | null>(null)
  const [addConfigOpen, setAddConfigOpen] = React.useState(false)
  const [editConfigOpen, setEditConfigOpen] = React.useState(false)
  const [editingConfig, setEditingConfig] = React.useState<BuildSetting | null>(null)
  const [deleteConfigDialogOpen, setDeleteConfigDialogOpen] = React.useState(false)
  const [deletingConfig, setDeletingConfig] = React.useState<BuildSetting | null>(null)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const { activeProjectId, activeProjectName, setActiveContextWithNames } = useProjectStore()
  const hasSyncedProjectFromRepoRef = React.useRef(false)

  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  const isAdmin = useAuthStore((state) => state.user?.role === "admin")

  const { data: repo, isLoading, error: repoError } = useQuery({
    queryKey: ["code-repository", repoId],
    queryFn: () => codeRepositoriesApi.get(repoId!),
    enabled: !!repoId,
  })

  React.useEffect(() => {
    hasSyncedProjectFromRepoRef.current = false
  }, [repoId])

  React.useEffect(() => {
    if (!hasSyncedProjectFromRepoRef.current && repo?.project_id) {
      if (activeProjectId !== repo.project_id) {
        // Fetch project name and set context with names
        projectsApi.get(repo.project_id).then(project => {
          setActiveContextWithNames(repo.project_id, project.name, null, null)
        }).catch(() => {
          setActiveContextWithNames(repo.project_id, null, null, null)
        })
      }
      hasSyncedProjectFromRepoRef.current = true
    }
  }, [repo?.project_id, activeProjectId, setActiveContextWithNames])

  const { data: repos = [] } = useQuery({
    queryKey: ["code-repositories-simple", repo?.project_id],
    queryFn: () => codeRepositoriesApi.listSimple(repo!.project_id),
    enabled: !!repo?.project_id,
  })

  const safeRepos = Array.isArray(repos) ? repos : []

  const { data: buildSettings = [], isLoading: buildSettingsLoading } = useQuery({
    queryKey: ["build-settings", repoId],
    queryFn: () => codeRepositoriesApi.listBuildSettings(repoId!),
    enabled: !!repoId,
  })

  const { data: builds = [], isLoading: buildsLoading } = useQuery({
    queryKey: ["builds", repoId],
    queryFn: () => codeRepositoriesApi.listBuilds(repoId!),
    enabled: !!repoId,
    refetchInterval: 5000,
  })

  const { data: deployments = [], isLoading: deploymentsLoading } = useQuery({
    queryKey: ["code-repository-deployments", repoId],
    queryFn: () => codeRepositoriesApi.listDeployments(repoId!),
    enabled: !!repoId,
    refetchInterval: 5000,
  })

  const { data: _envs = [] } = useQuery({
    queryKey: ["envs", repo?.project_id],
    queryFn: () => envsApi.list(repo!.project_id),
    enabled: !!repo?.project_id,
  })

  type RetryBuildPayload = {
    id: string
    build_setting_id: string
    build_env_id: string
    git_ref?: string
  }

  const retryBuildMutation = useMutation({
    mutationFn: (b: RetryBuildPayload) => {
      return codeRepositoriesApi.triggerBuild(repoId!, {
        build_setting_id: b.build_setting_id,
        build_env_id: b.build_env_id,
        git_ref: b.git_ref,
      })
    },
    onMutate: (b: RetryBuildPayload) => {
      setRetryingBuildId(b?.id ?? null)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["builds", repoId] })
      toast.success("Build retry triggered")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || "Failed to retry build")
    },
    onSettled: () => {
      setRetryingBuildId(null)
    },
  })

  React.useEffect(() => {
    if (!triggerBuildDialogOpen) {
      setSelectedBuildSettingId(undefined)
      setSelectedBuildId(undefined)
    }
  }, [triggerBuildDialogOpen])

  const deleteConfigMutation = useMutation({
    mutationFn: (settingId: string) => codeRepositoriesApi.deleteBuildSetting(repoId!, settingId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["build-settings", repoId] })
      toast.success("Build setting removed")
    },
    onError: (err: unknown) => {
      const msg =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : null
      toast.error(msg || "Failed to remove build setting")
    },
  })

  const formatDuration = (s: number) =>
    s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`
  const settingNameById = (id: string) =>
    buildSettings.find((c) => c.id === id)?.name ?? id

  // ColumnDef for Build Settings table
  const buildSettingColumns: ColumnDef<BuildSetting>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => <span className="font-medium">{row.original.name}</span>,
    },
    {
      accessorKey: "git_ref",
      header: "Ref",
      cell: ({ row }) => row.original.git_ref || "main",
    },
    {
      accessorKey: "dockerfile_path",
      header: "Dockerfile",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.dockerfile_path}</span>
      ),
    },
    {
      accessorKey: "build_context",
      header: "Context",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.build_context}</span>
      ),
    },
    {
      accessorKey: "image_name",
      header: "Image Name",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.image_name}</span>
      ),
    },
    {
      id: "registry",
      header: "Registry",
      cell: ({ row }) => row.original.registry?.name ?? row.original.registry_id,
    },
    ...(!isViewer
      ? [
        {
          id: "actions",
          header: () => <span className="flex justify-end">Actions</span>,
          cell: ({ row }: { row: { original: BuildSetting } }) => {
            const cfg = row.original
            return (
              <div className="flex items-center justify-end gap-1">
                <Tooltip>
                  <TooltipTrigger
                    delay={200}
                    render={
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => {
                          setEditingConfig(cfg)
                          setEditConfigOpen(true)
                        }}
                      />
                    }
                  >
                    <Pencil />
                  </TooltipTrigger>
                  <TooltipContent>Edit build setting</TooltipContent>
                </Tooltip>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setSelectedBuildSettingId(cfg.id)
                    setSelectedBuildId(undefined)
                    setTriggerBuildDialogOpen(true)
                  }}
                >
                  <Play />
                  Build
                </Button>
                <Tooltip>
                  <TooltipTrigger
                    delay={200}
                    render={
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={() => {
                          setDeletingConfig(cfg)
                          setDeleteConfigDialogOpen(true)
                        }}
                      />
                    }
                  >
                    <Trash2 />
                  </TooltipTrigger>
                  <TooltipContent>Delete build setting</TooltipContent>
                </Tooltip>
              </div>
            )
          },
        } as ColumnDef<BuildSetting>,
      ]
      : []),
  ]

  // ColumnDef for Build History table
  type BuildItem = (typeof builds)[number]
  const buildHistoryColumns: ColumnDef<BuildItem>[] = [
    {
      accessorKey: "build_number",
      header: "#",
      cell: ({ row }) => row.original.build_number,
    },
    {
      id: "setting",
      header: "Build Setting",
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">
          {row.original.build_setting_id
            ? settingNameById(row.original.build_setting_id)
            : "-"}
        </span>
      ),
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => <BuildStatusBadge status={row.original.status} />,
    },
    {
      accessorKey: "git_ref",
      header: "Ref",
    },
    {
      accessorKey: "image_full_name",
      header: "Image",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs truncate block">
            {row.original.image_full_name}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/row:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation()
              navigator.clipboard.writeText(row.original.image_full_name)
              toast.success("Image address copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      accessorKey: "duration",
      header: "Duration",
      cell: ({ row }) => row.original.duration ? formatDuration(row.original.duration) : "-",
    },
    {
      accessorKey: "created_at",
      header: "Created At",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.created_at)}</span>
        </div>
      ),
    },
    {
      id: "actions",
      header: () => <span className="flex justify-end">Actions</span>,
      cell: ({ row }) => {
        const b = row.original
        const isRetryingCurrentBuild = retryBuildMutation.isPending && retryingBuildId === b.id
        return (
          <div className="flex items-center justify-end gap-1">
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={<Button variant="ghost" size="icon-sm" onClick={() => setLogBuildId(b.id)} />}
              >
                <FileClock />
              </TooltipTrigger>
              <TooltipContent>View Build Logs</TooltipContent>
            </Tooltip>
            {!isViewer && (b.status === "failed" || b.status === "cancelled") && (
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => retryBuildMutation.mutate(b)}
                      disabled={isRetryingCurrentBuild}
                    />
                  }
                >
                  <>
                    {isRetryingCurrentBuild ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <RotateCw />
                    )}
                    Retry
                  </>
                </TooltipTrigger>
                <TooltipContent>Retry Build</TooltipContent>
              </Tooltip>
            )}
            {!isViewer && b.status === "succeeded" && (
              <Tooltip>
                <TooltipTrigger
                  delay={200}
                  render={
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => {
                        setSelectedBuildId(b.id)
                        setSelectedBuildSettingId(b.build_setting_id || undefined)
                        setTriggerBuildDialogOpen(true)
                      }}
                    />
                  }
                >
                  <>
                    <Rocket />
                    Deploy
                  </>
                </TooltipTrigger>
              </Tooltip>
            )}
          </div>
        )
      },
    },
  ]

  // ColumnDef for Deployment History table
  type DeploymentItem = (typeof deployments)[number]
  const deploymentColumns: ColumnDef<DeploymentItem>[] = [
    {
      accessorKey: "build_number",
      header: "Build #",
      cell: ({ row }) => <span className="font-medium">{row.original.build_number}</span>,
    },
    {
      id: "environment",
      header: "Environment",
      cell: ({ row }) => row.original.env_name ?? "-",
    },
    {
      id: "application",
      header: "Application",
      cell: ({ row }) => {
        const d = row.original
        return (
          <Button
            variant="link"
            className="p-0 h-auto text-xs"
            onClick={() => navigate(`/applications/${d.app_id}`)}
          >
            <ExternalLink />
            {d.app_name}
          </Button>
        )
      },
    },
    {
      accessorKey: "image_full_name",
      header: "Image",
      cell: ({ row }) => (
        <div className="flex items-center gap-1">
          <span className="font-mono text-xs truncate block" title={row.original.image_full_name}>
            {row.original.image_full_name}
          </span>
          <Button
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/row:opacity-100 transition-opacity"
            onClick={(e) => {
              e.stopPropagation()
              navigator.clipboard.writeText(row.original.image_full_name)
              toast.success("Image address copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      accessorKey: "git_ref",
      header: "Ref",
      cell: ({ row }) => <span className="text-xs">{row.original.git_ref}</span>,
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => {
        const status = row.original.status
        const errorMessage = row.original.error_message

        if (status === "failed" && errorMessage) {
          return <DeploymentErrorPopover errorMessage={errorMessage} />
        }

        return <BuildStatusBadge status={status} className={status === "failed" ? "cursor-pointer" : ""} />
      },
    },
    {
      accessorKey: "created_at",
      header: "Deployed At",
      cell: ({ row }) => (
        <div className="flex items-center gap-1.5 text-muted-foreground">
          <Clock className="h-3 w-3" />
          <span>{formatDate(row.original.created_at)}</span>
        </div>
      ),
    },
  ]

  if (!repoId) {
    return (
      <NotFoundPage
        resourceType="Code Repository"
        backHref="/code-repositories"
        backLabel="Back to Code Repositories"
      />
    )
  }

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

  if (!repo) {
    if (isAxiosError(repoError) && repoError.response?.status === 403) {
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
  const breadcrumbs = [
    // Show project layer for admin users when activeProjectId is set
    ...(isAdmin && activeProjectId ? [
      { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
      {
        label: activeProjectName ?? "Project",
        icon: GalleryVerticalEnd,
        href: `/projects/${activeProjectId}`,
      },
    ] : []),
    {
      label: "Code Repositories",
      href: "/code-repositories",
      icon: FolderGit2,
    },
    {
      label: repo.name,
      icon: FolderGit2,
      dropdown:
        safeRepos.length > 1 ? (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={
                <Button variant="ghost" size="icon-sm">
                  <ChevronsUpDown />
                </Button>
              }
            />
            <DropdownMenuContent align="start" className="w-fit">
              <DropdownMenuGroup>
                {safeRepos.map((r) => (
                  <DropdownMenuItem
                    key={r.id}
                    onClick={() => navigate(`/code-repositories/${r.id}`)}
                  >
                    <FolderGit2 className="h-4 w-4" />
                    {r.name}
                  </DropdownMenuItem>
                ))}
              </DropdownMenuGroup>
            </DropdownMenuContent>
          </DropdownMenu>
        ) : undefined,
    },
  ]


  const totalBuilds = builds.length
  const successfulBuilds = builds.filter((b) => b.status === "succeeded").length
  const successRate = totalBuilds > 0 ? (successfulBuilds / totalBuilds) * 100 : 0
  const totalDeployments = deployments.length
  const totalBuildSettings = buildSettings.length


  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary/10 rounded-lg text-primary shrink-0">
              <FolderGit2 className="h-8 w-8" />
            </div>
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold tracking-tight truncate">
                  {repo.name}
                </h1>
              </div>
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <span className="font-mono">{repo.slug}</span>
                <span>•</span>
                {repo.description ? (
                  <span className="truncate">{repo.description}</span>
                ) : (
                  <span className="italic">No description</span>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {!isViewer && (
              <Button variant="outline" size="icon" onClick={() => setEditDialogOpen(true)}>
                <Pencil />
              </Button>
            )}
            {buildSettings.length > 0 && !isViewer && (
              <Button onClick={() => {
                setSelectedBuildSettingId(undefined)
                setSelectedBuildId(undefined)
                setTriggerBuildDialogOpen(true)
              }}>
                <Play />
                Build
              </Button>
            )}
          </div>
        </div>
      </div>

      <Tabs value={currentTab} onValueChange={(v) => setSearchParams({ tab: v }, { replace: true })}>
        <TabsList>
          <TabsTrigger value="overview"><Telescope />Overview</TabsTrigger>
          <TabsTrigger value="topology"><Share2 />Topology</TabsTrigger>
          <TabsTrigger value="build"><Hammer />Build</TabsTrigger>
          <TabsTrigger value="deploy"><Rocket />Deploy</TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="group/card space-y-4 mt-2">
          <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Info className="h-4 w-4" />
                Repository Information
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-1 lg:grid-cols-3">
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Git URL</p>
                  <div className="flex items-center gap-2">
                    <p className="text-sm font-mono break-all">{repo.git_repo_url}</p>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="opacity-0 group-hover/card:opacity-100 transition-opacity"
                      onClick={(e) => {
                        e.stopPropagation()
                        navigator.clipboard.writeText(repo.git_repo_url)
                        toast.success("Git URL copied to clipboard")
                      }}
                    >
                      <Copy />
                    </Button>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              title="Build Settings"
              value={totalBuildSettings}
              icon={Hammer}
              description="Configurations defined"
              color="purple"
            />
            <StatCard
              title="Total Builds"
              value={totalBuilds}
              icon={Play}
              description="All time builds"
              color="blue"
            />
            <StatCard
              title="Success Rate"
              value={totalBuilds > 0 ? `${successRate.toFixed(0)}%` : "N/A"}
              icon={CheckCircle}
              description="Build success rate"
              color={totalBuilds > 0 ? (successRate >= 90 ? "green" : successRate >= 70 ? "orange" : "red") : "gray"}
            />
            <StatCard
              title="Deployments"
              value={totalDeployments}
              icon={Rocket}
              description="Total deployments"
              color="sky"
            />
          </div>
        </TabsContent>
        <TabsContent value="topology" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Share2 className="h-4 w-4" />
                Topology
              </CardTitle>
              <CardDescription>
                Visualization of the delivery pipeline from code to deployment.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <RepoTopologyView repoId={repo.id} />
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="build" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Hammer className="h-4 w-4" />
                Build Settings
              </CardTitle>
              <CardDescription>
                One repo can have multiple build settings (e.g. frontend, backend).
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {buildSettings.length > 0 && !isViewer ? (<div className="flex items-center justify-end">
                <Button onClick={() => setAddConfigOpen(true)}>
                  <Plus />
                  Create
                </Button>
              </div>) : null}
              {buildSettings.length === 0 ? (
                <EmptyState
                  title="No build settings"
                  description="Add a build setting to start building images from this repository."
                  icon={Hammer}
                  actionText={!isViewer ? "Create build setting" : undefined}
                  onAction={!isViewer ? () => setAddConfigOpen(true) : undefined}
                  actionIcon={!isViewer ? Plus : undefined}
                />
              ) : (
                <DataTable
                  columns={buildSettingColumns}
                  data={buildSettings}
                  isLoading={buildSettingsLoading}
                />
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <FileClock className="h-4 w-4" />
                Build History
              </CardTitle>
              <CardDescription>
                All builds for this repository. Deploy succeeded builds to an environment.
              </CardDescription>
            </CardHeader>
            <CardContent>
              {builds.length === 0 ? (
                <EmptyState
                  title="No builds yet"
                  description="Trigger a build from a configuration above to see the history here."
                  icon={FileClock}
                />
              ) : (
                <DataTable
                  columns={buildHistoryColumns}
                  data={builds}
                  isLoading={buildsLoading}
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="deploy" className="space-y-4 mt-2">
          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Rocket className="h-4 w-4" />
                Deployment History
              </CardTitle>
              <CardDescription>
                Track when and where builds from this repository were deployed.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {deployments.length === 0 ? (
                <EmptyState
                  title="No deployment history"
                  description="Deploy a successful build to an environment to see it here."
                  icon={History}
                />
              ) : (
                <DataTable
                  columns={deploymentColumns}
                  data={deployments as DeploymentItem[]}
                  isLoading={deploymentsLoading}
                />
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>

      <UnifiedBuildDeployDialog
        open={triggerBuildDialogOpen}
        onOpenChange={setTriggerBuildDialogOpen}
        repoId={repoId}
        projectId={repo.project_id}
        preSelectedConfigId={selectedBuildSettingId}
        preSelectedBuildId={selectedBuildId}
      />

      <EditCodeRepositoryDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        repo={repo}
        onSuccess={() =>
          queryClient.invalidateQueries({ queryKey: ["code-repository", repoId] })
        }
      />

      <EditBuildSettingDialog
        open={editConfigOpen}
        onOpenChange={setEditConfigOpen}
        repoId={repoId}
        setting={editingConfig}
        onSuccess={() => {
          setEditingConfig(null)
          queryClient.invalidateQueries({
            queryKey: ["build-settings", repoId],
          })
        }
        }
      />

      <CreateBuildSettingDialog
        open={addConfigOpen}
        onOpenChange={setAddConfigOpen}
        repoId={repoId}
        onSuccess={() =>
          queryClient.invalidateQueries({
            queryKey: ["build-settings", repoId],
          })
        }
      />

      {
        logBuildId && repoId && (
          <Dialog
            open={!!logBuildId}
            onOpenChange={() => setLogBuildId(null)}
          >
            <DialogContent className="sm:max-w-[90vw] w-full sm:max-h-[90vh] flex flex-col">
              <DialogHeader>
                <DialogTitle>Build logs</DialogTitle>
              </DialogHeader>
              <div className="min-h-0 flex-1 overflow-hidden">
                <BuildLogViewer buildId={logBuildId} repoId={repoId} />
              </div>
            </DialogContent>
          </Dialog>
        )
      }
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
                  deleteConfigMutation.mutate(deletingConfig.id)
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
