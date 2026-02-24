import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Blocks,
  BookOpen,
  CheckCircle2,
  Clock,
  Download,
  FolderSync,
  Globe,
  Info,
  Library,
  Loader2,
  PauseCircle,
  Pencil,
  PlugZap,
  Plus,
  RefreshCcw,
  ShieldAlert,
  Trash2,
  XCircle
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type Extension,
  type HelmChartInfo,
  type HelmRepository,
} from "@/api/clusters"
import { AddRepositoryDialog } from "@/components/cluster/add-repository-dialog"
import { InstallExtensionDialog } from "@/components/cluster/install-extension-dialog"
import { UpdateExtensionDialog } from "@/components/cluster/update-extension-dialog"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Empty,
  EmptyContent,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/components/ui/empty"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

interface ClusterExtensionsProps {
  clusterId: string
}

export function ClusterExtensions({ clusterId }: ClusterExtensionsProps) {
  const queryClient = useQueryClient()

  // Helm Operator status check
  const {
    data: operatorStatus,
    isLoading: operatorLoading,
    error: operatorError,
  } = useQuery({
    queryKey: ["helm-operator-status", clusterId],
    queryFn: () => clustersApi.getHelmOperatorStatus(clusterId),
    retry: 1,
  })

  const installOperatorMutation = useMutation({
    mutationFn: () => clustersApi.installHelmOperator(clusterId),
    onSuccess: () => {
      toast.success("Helm Operator installed successfully", {
        description: "The cluster is now ready for extension management.",
      })
      queryClient.invalidateQueries({
        queryKey: ["helm-operator-status", clusterId],
      })
    },
    onError: (error: any) => {
      toast.error("Failed to install Helm Operator", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  // Loading state
  if (operatorLoading) {
    return (
      <Card>
        <CardContent className="py-12">
          <div className="flex flex-col items-center justify-center gap-4">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Detecting Helm Operator...
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  // Error state
  if (operatorError) {
    return (
      <Card>
        <CardContent className="py-8">
          <Empty className="border-0 flex-1">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <ShieldAlert />
              </EmptyMedia>
              <EmptyTitle>Unable to Check Helm Operator</EmptyTitle>
              <EmptyDescription>
                Failed to detect Helm Operator status. Please check the cluster
                connectivity and try again.
              </EmptyDescription>
              <EmptyContent>
                <Button
                  variant="outline"
                  onClick={() =>
                    queryClient.invalidateQueries({
                      queryKey: ["helm-operator-status", clusterId],
                    })
                  }
                >
                  <RefreshCcw />
                  Retry
                </Button>
              </EmptyContent>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }

  // Not installed - show install guide
  if (!operatorStatus?.installed) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <PlugZap className="h-4 w-4" />
            Cluster Extensions
          </CardTitle>
          <CardDescription>
            Install and manage cluster extensions like monitoring, logging, and
            service mesh
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Empty className="border-0 flex-1">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Info className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>Helm Operator Required</EmptyTitle>
              <EmptyDescription>
                The Helm Operator is required to manage cluster extensions. It
                enables installing and managing Helm charts as extensions in your
                cluster.
              </EmptyDescription>
              <EmptyContent>
                <Button
                  onClick={() => installOperatorMutation.mutate()}
                  disabled={installOperatorMutation.isPending}
                >
                  {installOperatorMutation.isPending ? (
                    <>
                      <Loader2 className="animate-spin" />
                      Installing...
                    </>
                  ) : (
                    <>
                      <Download />
                      Install Helm Operator
                    </>
                  )}
                </Button>
              </EmptyContent>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }

  // Installed - show the full extension management UI
  return <ExtensionManager clusterId={clusterId} />
}

// ========================
// Extension Manager (shown after helm-operator is installed)
// ========================

function ExtensionManager({ clusterId }: { clusterId: string }) {
  const [activeTab, setActiveTab] = React.useState("installed")

  return (
    <div className="space-y-4">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="installed">
            <Blocks />
            Installed
          </TabsTrigger>
          <TabsTrigger value="catalog">
            <Library />
            Catalog
          </TabsTrigger>
          <TabsTrigger value="repositories">
            <FolderSync />
            Repositories
          </TabsTrigger>
        </TabsList>

        <TabsContent value="installed" className="mt-2">
          <InstalledExtensions clusterId={clusterId} />
        </TabsContent>

        <TabsContent value="catalog" className="mt-2">
          <ExtensionCatalog clusterId={clusterId} />
        </TabsContent>

        <TabsContent value="repositories" className="mt-2">
          <RepositoryList clusterId={clusterId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// ========================
// Installed Extensions Tab
// ========================

function InstalledExtensions({ clusterId }: { clusterId: string }) {
  const queryClient = useQueryClient()

  const { data: extensions = [], isLoading } = useQuery({
    queryKey: ["extensions", clusterId],
    queryFn: () => clustersApi.listExtensions(clusterId),
    refetchInterval: 10000,
  })

  const [deleteTarget, setDeleteTarget] = React.useState<string | null>(null)
  const [updateTarget, setUpdateTarget] = React.useState<Extension | null>(null)

  const uninstallMutation = useMutation({
    mutationFn: (name: string) =>
      clustersApi.uninstallExtension(clusterId, name),
    onSuccess: () => {
      toast.success("Extension uninstalled")
      queryClient.invalidateQueries({ queryKey: ["extensions", clusterId] })
      setDeleteTarget(null)
    },
    onError: (error: any) => {
      toast.error("Failed to uninstall extension", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const safeExtensions: Extension[] = Array.isArray(extensions) ? extensions : []

  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-12">
          <div className="flex flex-col items-center justify-center gap-4">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">Loading extensions...</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (safeExtensions.length === 0) {
    return (
      <Card>
        <CardContent className="py-8">
          <Empty className="border-0 flex-1">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <Blocks className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>No Extensions Installed</EmptyTitle>
              <EmptyDescription>
                Browse the catalog to discover and install extensions for your
                cluster.
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {safeExtensions.map((ext) => (
          <ExtensionCard
            key={ext.name}
            extension={ext}
            onUpdate={() => setUpdateTarget(ext)}
            onUninstall={() => setDeleteTarget(ext.name)}
          />
        ))}
      </div>

      <UpdateExtensionDialog
        open={!!updateTarget}
        onOpenChange={(open) => !open && setUpdateTarget(null)}
        clusterId={clusterId}
        extension={updateTarget}
      />

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Uninstall Extension</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to uninstall "{deleteTarget}"? This will
              remove the Helm release and all associated resources from the
              cluster.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteTarget && uninstallMutation.mutate(deleteTarget)}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {uninstallMutation.isPending ? "Uninstalling..." : "Uninstall"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function ExtensionCard({
  extension,
  onUpdate,
  onUninstall,
}: {
  extension: Extension
  onUpdate: () => void
  onUninstall: () => void
}) {
  const getStatusBadge = () => {
    if (extension.ready) {
      return (
        <Badge variant="outline" className="text-green-600 gap-1">
          <CheckCircle2 className="h-3 w-3" />
          Ready
        </Badge>
      )
    }
    if (extension.suspended) {
      return (
        <Badge variant="outline" className="text-yellow-600 gap-1">
          <PauseCircle className="h-3 w-3" />
          Suspended
        </Badge>
      )
    }
    if (extension.status === "failed") {
      return (
        <Badge variant="destructive" className="gap-1">
          <XCircle className="h-3 w-3" />
          Failed
        </Badge>
      )
    }
    return (
      <Badge variant="outline" className="text-blue-600 gap-1">
        <Clock className="h-3 w-3" />
        {extension.status || "Reconciling"}
      </Badge>
    )
  }

  return (
    <Card className="flex flex-col">
      <CardHeader className="pb-3">
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-2 min-w-0">
            <div className="p-1.5 bg-primary/10 rounded-md text-primary shrink-0">
              <Blocks className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <CardTitle className="text-sm truncate">{extension.name}</CardTitle>
              <p className="text-xs text-muted-foreground font-mono truncate">
                {extension.chart_name}
                {extension.chart_version ? `:${extension.chart_version}` : ""}
              </p>
            </div>
          </div>
          {getStatusBadge()}
        </div>
      </CardHeader>
      <CardContent className="flex-1 pb-3">
        <div className="grid grid-cols-2 gap-y-2 text-xs">
          <div>
            <p className="text-muted-foreground">Namespace</p>
            <p className="font-mono">{extension.release_namespace}</p>
          </div>
          <div>
            <p className="text-muted-foreground">Revision</p>
            <p>{extension.revision || "-"}</p>
          </div>
          {extension.app_version && (
            <div>
              <p className="text-muted-foreground">App Version</p>
              <p className="font-mono">{extension.app_version}</p>
            </div>
          )}
          {extension.repository && (
            <div>
              <p className="text-muted-foreground">Repository</p>
              <p className="truncate">{extension.repository}</p>
            </div>
          )}
        </div>
        {extension.message && !extension.ready && (
          <p className="text-xs text-muted-foreground mt-2 line-clamp-2 bg-muted/50 rounded px-2 py-1">
            {extension.message}
          </p>
        )}
      </CardContent>
      <div className="flex gap-2 px-6 pb-4 pt-0">
        <Button
          variant="outline"
          size="sm"
          className="flex-1"
          onClick={onUpdate}
        >
          <Pencil className="h-3.5 w-3.5" />
          Update
        </Button>
        <Button
          variant="outline"
          size="sm"
          className="flex-1 text-destructive hover:text-destructive"
          onClick={onUninstall}
        >
          <Trash2 className="h-3.5 w-3.5" />
          Uninstall
        </Button>
      </div>
    </Card>
  )
}

// ========================
// Extension Catalog Tab
// ========================

function ExtensionCatalog({ clusterId }: { clusterId: string }) {
  const { data: repos = [], isLoading } = useQuery({
    queryKey: ["helm-repositories", clusterId],
    queryFn: () => clustersApi.listHelmRepositories(clusterId),
  })

  const [selectedChart, setSelectedChart] = React.useState<HelmChartInfo | null>(null)
  const [selectedRepoName, setSelectedRepoName] = React.useState<string>("")
  const [installOpen, setInstallOpen] = React.useState(false)

  const safeRepos: HelmRepository[] = Array.isArray(repos) ? repos : []

  // Collect all charts from all ready repos
  const allCharts: Array<{ chart: HelmChartInfo; repoName: string; repoURL: string }> = []
  for (const repo of safeRepos) {
    if (repo.ready && repo.charts) {
      for (const chart of repo.charts) {
        allCharts.push({ chart, repoName: repo.name, repoURL: repo.url })
      }
    }
  }

  const handleInstall = (chart: HelmChartInfo, repoName: string) => {
    setSelectedChart(chart)
    setSelectedRepoName(repoName)
    setInstallOpen(true)
  }

  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-12">
          <div className="flex flex-col items-center justify-center gap-4">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">Loading catalog...</p>
          </div>
        </CardContent>
      </Card>
    )
  }

  if (allCharts.length === 0) {
    return (
      <Card>
        <CardContent className="py-8">
          <Empty className="border-0 flex-1">
            <EmptyHeader>
              <EmptyMedia variant="icon">
                <BookOpen className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>No Charts Available</EmptyTitle>
              <EmptyDescription>
                {safeRepos.length === 0
                  ? "Add a Helm repository first to browse available charts."
                  : "Repositories are syncing. Charts will appear once sync completes."}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        {allCharts.map(({ chart, repoName }) => (
          <Card key={`${repoName}-${chart.name}`} className="flex flex-col">
            <CardHeader className="pb-3">
              <div className="flex items-center gap-2">
                <div className="p-1.5 bg-blue-500/10 rounded-md text-blue-600 shrink-0">
                  <Blocks className="h-4 w-4" />
                </div>
                <div className="min-w-0">
                  <CardTitle className="text-sm truncate">
                    {chart.name}
                  </CardTitle>
                  <p className="text-xs text-muted-foreground truncate">
                    from {repoName}
                  </p>
                </div>
              </div>
            </CardHeader>
            <CardContent className="flex-1 pb-3">
              {chart.description && (
                <p className="text-xs text-muted-foreground line-clamp-2 mb-2">
                  {chart.description}
                </p>
              )}
              <div className="flex flex-wrap gap-1">
                {chart.versions?.slice(0, 3).map((v) => (
                  <Badge
                    key={v.version}
                    variant="secondary"
                    className="text-[10px]"
                  >
                    {v.version}
                  </Badge>
                ))}
                {(chart.versions?.length || 0) > 3 && (
                  <Badge variant="outline" className="text-[10px]">
                    +{(chart.versions?.length || 0) - 3} more
                  </Badge>
                )}
              </div>
            </CardContent>
            <div className="px-6 pb-4 pt-0">
              <Button
                size="sm"
                className="w-full"
                onClick={() => handleInstall(chart, repoName)}
              >
                <Download className="h-3.5 w-3.5" />
                Install
              </Button>
            </div>
          </Card>
        ))}
      </div>

      <InstallExtensionDialog
        open={installOpen}
        onOpenChange={setInstallOpen}
        clusterId={clusterId}
        chart={selectedChart}
        repositoryName={selectedRepoName}
      />
    </>
  )
}

// ========================
// Repository List Tab
// ========================

function RepositoryList({ clusterId }: { clusterId: string }) {
  const queryClient = useQueryClient()
  const [addOpen, setAddOpen] = React.useState(false)
  const [deleteTarget, setDeleteTarget] = React.useState<string | null>(null)

  const { data: repos = [], isLoading } = useQuery({
    queryKey: ["helm-repositories", clusterId],
    queryFn: () => clustersApi.listHelmRepositories(clusterId),
  })

  const deleteMutation = useMutation({
    mutationFn: (name: string) =>
      clustersApi.deleteHelmRepository(clusterId, name),
    onSuccess: () => {
      toast.success("Repository deleted")
      queryClient.invalidateQueries({
        queryKey: ["helm-repositories", clusterId],
      })
      setDeleteTarget(null)
    },
    onError: (error: any) => {
      toast.error("Failed to delete repository", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  const safeRepos: HelmRepository[] = Array.isArray(repos) ? repos : []

  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-12">
          <div className="flex flex-col items-center justify-center gap-4">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Loading repositories...
            </p>
          </div>
        </CardContent>
      </Card>
    )
  }

  return (
    <>
      <div className="space-y-4">
        <div className="flex justify-end">
          <Button onClick={() => setAddOpen(true)}>
            <Plus />
            Add Repository
          </Button>
        </div>

        {safeRepos.length === 0 ? (
          <Card>
            <CardContent className="py-8">
              <Empty className="border-0 flex-1">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <FolderSync className="h-6 w-6" />
                  </EmptyMedia>
                  <EmptyTitle>No Repositories</EmptyTitle>
                  <EmptyDescription>
                    Add a Helm repository to start browsing and installing
                    extensions.
                  </EmptyDescription>
                  <EmptyContent>
                    <Button onClick={() => setAddOpen(true)}>
                      <Plus />
                      Add Repository
                    </Button>
                  </EmptyContent>
                </EmptyHeader>
              </Empty>
            </CardContent>
          </Card>
        ) : (
          <div className="space-y-3">
            {safeRepos.map((repo) => (
              <Card key={repo.name}>
                <CardContent className="py-4">
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 min-w-0">
                      <div className="p-2 bg-muted rounded-md shrink-0">
                        {repo.type === "oci" ? (
                          <Blocks className="h-4 w-4" />
                        ) : (
                          <Globe className="h-4 w-4" />
                        )}
                      </div>
                      <div className="min-w-0">
                        <div className="flex items-center gap-2">
                          <p className="text-sm font-medium truncate">
                            {repo.name}
                          </p>
                          {repo.system && (
                            <Badge variant="secondary" className="text-[10px] shrink-0">
                              System
                            </Badge>
                          )}
                          <Badge
                            variant="outline"
                            className="text-[10px] uppercase shrink-0"
                          >
                            {repo.type}
                          </Badge>
                        </div>
                        <p className="text-xs text-muted-foreground font-mono truncate">
                          {repo.url}
                        </p>
                      </div>
                    </div>

                    <div className="flex items-center gap-3 shrink-0 ml-4">
                      <div className="text-right text-xs">
                        {repo.ready ? (
                          <span className="flex items-center gap-1 text-green-600">
                            <CheckCircle2 className="h-3 w-3" />
                            Ready
                          </span>
                        ) : (
                          <span className="flex items-center gap-1 text-yellow-600">
                            <Clock className="h-3 w-3" />
                            Syncing
                          </span>
                        )}
                        {repo.total_charts > 0 && (
                          <p className="text-muted-foreground">
                            {repo.total_charts} charts
                          </p>
                        )}
                      </div>

                      {!repo.system && (
                        <Button
                          variant="ghost"
                          size="icon-xs"
                          className="text-destructive hover:text-destructive hover:bg-destructive/10"
                          onClick={() => setDeleteTarget(repo.name)}
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      )}
                    </div>
                  </div>
                  {repo.message && !repo.ready && (
                    <p className="text-xs text-muted-foreground mt-2 bg-muted/50 rounded px-2 py-1">
                      {repo.message}
                    </p>
                  )}
                </CardContent>
              </Card>
            ))}
          </div>
        )}
      </div>

      <AddRepositoryDialog
        open={addOpen}
        onOpenChange={setAddOpen}
        clusterId={clusterId}
      />

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Repository</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to delete the repository "{deleteTarget}"?
              Installed extensions from this repository will not be affected.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget)
              }
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
