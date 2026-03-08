import {
  codeRepositoriesApi,
  type CodeRepositoryBuildConfig,
} from "@/api/code-repositories"
import { envsApi } from "@/api/envs"
import { BuildLogViewer } from "@/components/builds/build-log-viewer"
import { BuildStatusBadge } from "@/components/builds/build-status-badge"
import { CreateBuildConfigDialog } from "@/components/code-repositories/create-build-config-dialog"
import { EditBuildConfigDialog } from "@/components/code-repositories/edit-build-config-dialog"
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
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import type { AxiosError } from "axios"
import {
  CheckCircle,
  ChevronsUpDown,
  Copy,
  ExternalLink,
  FileClock,
  FileText,
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
import { useLocation, useNavigate, useParams, useSearchParams } from "react-router-dom"
import { toast } from "sonner"

export function CodeRepositoryDetailPage() {
  const { repoId } = useParams<{ repoId: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()

  const currentTab = searchParams.get("tab") || "overview"
  const [triggerBuildDialogOpen, setTriggerBuildDialogOpen] = React.useState(false)
  const [selectedBuildConfigId, setSelectedBuildConfigId] = React.useState<string | undefined>(undefined)
  const [selectedBuildId, setSelectedBuildId] = React.useState<string | undefined>(undefined)
  const [logBuildId, setLogBuildId] = React.useState<string | null>(null)
  const [addConfigOpen, setAddConfigOpen] = React.useState(false)
  const [editConfigOpen, setEditConfigOpen] = React.useState(false)
  const [editingConfig, setEditingConfig] = React.useState<CodeRepositoryBuildConfig | null>(null)
  const [deleteConfigDialogOpen, setDeleteConfigDialogOpen] = React.useState(false)
  const [deletingConfig, setDeletingConfig] = React.useState<CodeRepositoryBuildConfig | null>(null)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const { activeProjectId, setActiveProjectId } = useProjectStore()
  const hasSyncedProjectFromRepoRef = React.useRef(false)

  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'

  // Read project context from navigation state (set when navigating from admin project detail page)
  const location = useLocation()
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")
  const fromProject = isAdmin
    ? (location.state as { fromProjectId?: string; fromProjectName?: string } | null)
    : null

  const { data: repo, isLoading } = useQuery({
    queryKey: ["code-repository", repoId],
    queryFn: () => codeRepositoriesApi.get(repoId!),
    enabled: !!repoId,
  })

  React.useEffect(() => {
    hasSyncedProjectFromRepoRef.current = false
  }, [repoId])

  React.useEffect(() => {
    if (!hasSyncedProjectFromRepoRef.current && repo?.project_id && activeProjectId !== repo.project_id) {
      setActiveProjectId(repo.project_id)
    }
    if (repo?.project_id) {
      hasSyncedProjectFromRepoRef.current = true
    }
  }, [repo?.project_id, activeProjectId, setActiveProjectId])

  const { data: repos = [] } = useQuery({
    queryKey: ["code-repositories-simple", repo?.project_id],
    queryFn: () => codeRepositoriesApi.listSimple(repo!.project_id),
    enabled: !!repo?.project_id,
  })

  const safeRepos = Array.isArray(repos) ? repos : []

  const { data: buildConfigs = [] } = useQuery({
    queryKey: ["code-repository-build-configs", repoId],
    queryFn: () => codeRepositoriesApi.listBuildConfigs(repoId!),
    enabled: !!repoId,
  })

  const { data: builds = [] } = useQuery({
    queryKey: ["code-repository-builds", repoId],
    queryFn: () => codeRepositoriesApi.listBuilds(repoId!),
    enabled: !!repoId,
    refetchInterval: 5000,
  })

  const { data: deployments = [] } = useQuery({
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

  const retryBuildMutation = useMutation({
    mutationFn: (b: any) => {
      return codeRepositoriesApi.triggerBuild(repoId!, {
        build_config_id: b.code_repository_build_config_id,
        build_env_id: b.build_env_id,
        git_ref: b.git_ref,
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["code-repository-builds", repoId] })
      toast.success("Build retry triggered")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || "Failed to retry build")
    },
  })

  React.useEffect(() => {
    if (!triggerBuildDialogOpen) {
      setSelectedBuildConfigId(undefined)
      setSelectedBuildId(undefined)
    }
  }, [triggerBuildDialogOpen])

  const deleteConfigMutation = useMutation({
    mutationFn: (configId: string) => codeRepositoriesApi.deleteBuildConfig(repoId!, configId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["code-repository-build-configs", repoId] })
      toast.success("Build config removed")
    },
    onError: (err: unknown) => {
      const msg =
        err && typeof err === "object" && "response" in err
          ? (err as { response?: { data?: { error?: string } } }).response?.data?.error
          : null
      toast.error(msg || "Failed to remove build config")
    },
  })

  const formatDuration = (s: number) =>
    s < 60 ? `${s}s` : `${Math.floor(s / 60)}m ${s % 60}s`
  const configNameById = (id: string) =>
    buildConfigs.find((c) => c.id === id)?.name ?? id

  // ColumnDef for Build Configs table
  const buildConfigColumns: ColumnDef<CodeRepositoryBuildConfig>[] = [
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
      header: "Image",
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
          cell: ({ row }: { row: { original: CodeRepositoryBuildConfig } }) => {
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
                  <TooltipContent>Edit build configuration</TooltipContent>
                </Tooltip>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => {
                    setSelectedBuildConfigId(cfg.id)
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
                  <TooltipContent>Delete build configuration</TooltipContent>
                </Tooltip>
              </div>
            )
          },
        } as ColumnDef<CodeRepositoryBuildConfig>,
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
      id: "config",
      header: "Config",
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">
          {row.original.code_repository_build_config_id
            ? configNameById(row.original.code_repository_build_config_id)
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
            className="opacity-0 group-hover/card:opacity-100 transition-opacity"
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
      header: "Created",
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">
          {new Date(row.original.created_at).toLocaleString()}
        </span>
      ),
    },
    {
      id: "actions",
      header: () => <span className="flex justify-end">Actions</span>,
      cell: ({ row }) => {
        const b = row.original
        return (
          <div className="flex items-center justify-end gap-1">
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={<Button variant="ghost" size="icon-sm" onClick={() => setLogBuildId(b.id)} />}
              >
                <FileText />
              </TooltipTrigger>
              <TooltipContent>View Logs</TooltipContent>
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
                      disabled={retryBuildMutation.isPending}
                    />
                  }
                >
                  <>
                    {retryBuildMutation.isPending ? (
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
                        setSelectedBuildConfigId(b.code_repository_build_config_id || undefined)
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
      cell: ({ row }) => row.original.app?.env?.name ?? "-",
    },
    {
      id: "application",
      header: "Application",
      cell: ({ row }) => {
        const d = row.original
        return d.app ? (
          <Button
            variant="link"
            className="p-0 h-auto text-xs"
            onClick={() => navigate(`/applications/${d.app?.id}`)}
          >
            <ExternalLink />
            {d.app.name}
          </Button>
        ) : (
          "-"
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
            className="opacity-0 group-hover/card:opacity-100 transition-opacity"
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
      accessorKey: "created_at",
      header: "Deployed At",
      cell: ({ row }) => (
        <span className="text-muted-foreground text-xs">
          {new Date(row.original.created_at).toLocaleString()}
        </span>
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
    return (
      <NotFoundPage
        resourceType="Code Repository"
        backHref="/code-repositories"
        backLabel="Back to Code Repositories"
      />
    )
  }

  const breadcrumbs = [
    // Prepend project layer when navigating from admin project detail page
    ...(fromProject?.fromProjectId ? [
      { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
      {
        label: fromProject.fromProjectName ?? "Project",
        icon: GalleryVerticalEnd,
        href: `/projects/${fromProject.fromProjectId}`,
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
  const totalBuildConfigs = buildConfigs.length


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

                {repo.webhook_enabled && (
                  <span className="text-xs text-muted-foreground px-2 py-0.5 rounded-full bg-muted border">
                    Webhook enabled
                  </span>
                )}
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
              <Button variant="outline" onClick={() => setEditDialogOpen(true)}>
                <Pencil />
                Edit
              </Button>
            )}
            {buildConfigs.length > 0 && !isViewer && (
              <Button onClick={() => {
                setSelectedBuildConfigId(undefined)
                setSelectedBuildId(undefined)
                setTriggerBuildDialogOpen(true)
              }}>
                <Hammer />
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

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="bg-linear-to-b/increasing from-blue/5 to-transparent data-[active=true]:bg-transparent">
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
                <div>
                  <p className="text-xs font-medium text-muted-foreground">Webhook</p>
                  <p className="text-sm">{repo.webhook_enabled ? "Enabled" : "Disabled"}</p>
                </div>
              </div>
            </CardContent>
          </Card>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <StatCard
              title="Build Configs"
              value={totalBuildConfigs}
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
                Build Configs
              </CardTitle>
              <CardDescription>
                One repo can have multiple build configs (e.g. frontend, backend). Configure Dockerfile, context, image, and registry per config.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              {buildConfigs.length > 0 && !isViewer ? (<div className="flex items-center justify-end">
                <Button onClick={() => setAddConfigOpen(true)}>
                  <Plus />
                  Create
                </Button>
              </div>) : null}
              {buildConfigs.length === 0 ? (
                <EmptyState
                  title="No build configurations"
                  description="Add a build configuration to start building images from this repository."
                  icon={Hammer}
                  actionText={!isViewer ? "Create build config" : undefined}
                  onAction={!isViewer ? () => setAddConfigOpen(true) : undefined}
                  actionIcon={!isViewer ? Plus : undefined}
                />
              ) : (
                <DataTable
                  columns={buildConfigColumns}
                  data={buildConfigs}
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
        preSelectedConfigId={selectedBuildConfigId}
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

      <EditBuildConfigDialog
        open={editConfigOpen}
        onOpenChange={setEditConfigOpen}
        repoId={repoId}
        config={editingConfig}
        onSuccess={() => {
          setEditingConfig(null)
          queryClient.invalidateQueries({
            queryKey: ["code-repository-build-configs", repoId],
          })
        }
        }
      />

      <CreateBuildConfigDialog
        open={addConfigOpen}
        onOpenChange={setAddConfigOpen}
        repoId={repoId}
        onSuccess={() =>
          queryClient.invalidateQueries({
            queryKey: ["code-repository-build-configs", repoId],
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
            <AlertDialogTitle>Remove Build Config</AlertDialogTitle>
            <AlertDialogDescription>
              {deletingConfig
                ? `Remove build config "${deletingConfig.name}"? This action cannot be undone.`
                : "Are you sure you want to remove this build config?"}
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
