import Editor from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ArrowBigUpDash,
  Blocks,
  Download,
  Loader2,
  RotateCcw,
  Trash2
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ClusterExtension,
  type Extension,
} from "@/api/clusters"
import { RetryInstallExtensionDialog } from "@/components/cluster/retry-install-extension-dialog"
import { UpdateExtensionDialog } from "@/components/cluster/update-extension-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { BrowseExtensionsDialog } from "@/components/extensions/browse-extensions-dialog"
import { InstallExtensionToClusterDialog } from "@/components/extensions/install-extension-dialog"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { useTheme } from "@/components/theme-provider/theme-provider"
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
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type { ColumnDef } from "@tanstack/react-table"

interface ClusterExtensionsProps {
  clusterId: string
}

export function ClusterExtensions({ clusterId }: ClusterExtensionsProps) {
  const queryClient = useQueryClient()
  const [browseOpen, setBrowseOpen] = React.useState(false)
  const [deleteTarget, setDeleteTarget] = React.useState<string | null>(null)
  const [updateTarget, setUpdateTarget] =
    React.useState<ClusterExtension | null>(null)
  const [installTarget, setInstallTarget] =
    React.useState<Extension | null>(null)
  const [retryInstallTarget, setRetryInstallTarget] =
    React.useState<ClusterExtension | null>(null)
  const [valuesTarget, setValuesTarget] =
    React.useState<ClusterExtension | null>(null)
  const [installOpen, setInstallOpen] = React.useState(false)
  const { theme } = useTheme()

  const [monacoTheme, setMonacoTheme] = React.useState<"vs" | "vs-dark">("vs")
  React.useEffect(() => {
    const resolve = () => {
      if (theme === "dark") return "vs-dark" as const
      if (theme === "light") return "vs" as const
      return document.documentElement.classList.contains("dark")
        ? ("vs-dark" as const)
        : ("vs" as const)
    }
    setMonacoTheme(resolve())
    if (theme !== "system") return
    const media = window.matchMedia("(prefers-color-scheme: dark)")
    const handler = () => setMonacoTheme(resolve())
    media.addEventListener("change", handler)
    return () => media.removeEventListener("change", handler)
  }, [theme])

  const { data: extensions = [], isLoading } = useQuery({
    queryKey: ["cluster-extensions", clusterId],
    queryFn: () => clustersApi.listClusterExtensions(clusterId),
    refetchInterval: (query) => {
      const data = query.state.data as ClusterExtension[] | undefined
      const isTransitioning = data?.some(
        (e) =>
          e.status === "pending" ||
          e.status === "installing" ||
          e.status === "uninstalling" ||
          e.status === "upgrading"
      )
      return isTransitioning ? 3000 : 10000
    },
  })

  const uninstallMutation = useMutation({
    mutationFn: (id: string) =>
      clustersApi.uninstallExtension(clusterId, id),
    onSuccess: () => {
      toast.success("Extension uninstalled")
      queryClient.invalidateQueries({ queryKey: ["cluster-extensions", clusterId] })
      setDeleteTarget(null)
    },
    onError: (error: unknown) => {
      const msg =
        error && typeof error === "object" && "response" in error
          ? (
            error as {
              response?: { data?: { error?: string } }
            }
          ).response?.data?.error
          : null
      toast.error("Failed to uninstall extension", {
        description:
          msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const retryMutation = useMutation({
    mutationFn: ({
      extension,
      data,
    }: {
      extension: ClusterExtension
      data?: { name?: string; values?: string }
    }) => clustersApi.retryExtension(clusterId, extension.id, data),
    onSuccess: (_data, payload) => {
      const ext = payload.extension
      const actionLabel = ext.phase === "uninstalling" ? "uninstall" : ext.phase === "upgrading" ? "update" : "install"
      toast.success("Retry started", {
        description: `${ext.name || ext.release_name} ${actionLabel} has been queued again.`,
      })
      queryClient.invalidateQueries({ queryKey: ["cluster-extensions", clusterId] })
    },
    onError: (error: unknown) => {
      const msg =
        error && typeof error === "object" && "response" in error
          ? (
            error as {
              response?: { data?: { error?: string } }
            }
          ).response?.data?.error
          : null
      toast.error("Failed to retry extension operation", {
        description:
          msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const safeClusterExtensions: ClusterExtension[] = Array.isArray(extensions)
    ? extensions
    : []

  // Derive installed extension names to pass to BrowseExtensionsDialog
  const installedExtensionIds = safeClusterExtensions.map((e) => e.extension_id)

  // Fetch the full extension to derive available (not-yet-installed) items
  const { data: extension = [], isLoading: loading } = useQuery({
    queryKey: ["extensions"],
    queryFn: () => clustersApi.listExtensions(),
  })

  const safeExtensions: Extension[] = Array.isArray(extension) ? extension : []

  // Filter out already-installed extensions (match by extension_id)
  const installedIds = new Set(safeClusterExtensions.map((e) => e.extension_id))
  const availableExtensions = safeExtensions.filter(
    (item) => !installedIds.has(item.id)
  )

  const extensionColumns: ColumnDef<Extension>[] = [
    {
      accessorKey: "name",
      header: "Extension",
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-purple-500/10 rounded-md text-purple-600 shrink-0">
              <Blocks className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <p className="font-medium text-sm truncate">
                {item.name || item.slug}
              </p>
              <p className="text-xs text-muted-foreground font-mono truncate">
                {item.oci_url}
              </p>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="text-xs text-muted-foreground line-clamp-2">
          {row.original.description || "-"}
        </span>
      ),
    },
    {
      accessorKey: "builtin",
      header: "Type",
      cell: ({ row }) =>
        row.original.builtin ? (
          <ColorBadge color="purple" className="text-[10px]">
            Built-in
          </ColorBadge>
        ) : (
          <ColorBadge color="blue" className="text-[10px]">
            Custom
          </ColorBadge>
        ),
    },
    {
      id: "actions",
      // header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex justify-end">
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                setInstallTarget(item)
                setInstallOpen(true)
              }}
            >
              <Download />
              Install
            </Button>
          </div>
        )
      }
    },
  ]

  const getStatusBadge = (ext: ClusterExtension) => {
    if (ext.status === "pending") return <ColorBadge color="orange">Pending</ColorBadge>
    if (ext.status === "installing") return <ColorBadge color="blue"><Loader2 className="h-3 w-3 animate-spin mr-1 inline-block" />Installing</ColorBadge>
    if (ext.status === "uninstalling")
      return <ColorBadge color="orange"><Loader2 className="h-3 w-3 animate-spin mr-1 inline-block" />Uninstalling</ColorBadge>
    if (ext.status === "upgrading")
      return <ColorBadge color="blue"><Loader2 className="h-3 w-3 animate-spin mr-1 inline-block" />Upgrading</ColorBadge>
    if (ext.status === "deployed")
      return <ColorBadge color="green">Installed</ColorBadge>
    if (ext.status === "failed" && ext.phase === "uninstalling")
      return <ColorBadge color="red">Uninstall Failed</ColorBadge>
    if (ext.status === "failed" && ext.phase === "upgrading")
      return <ColorBadge color="red">Update Failed</ColorBadge>
    if (ext.status === "failed")
      return <ColorBadge color="red">Install Failed</ColorBadge>
    return <ColorBadge color="gray">{ext.status || "Unknown"}</ColorBadge>
  }

  const getRetryLabel = (ext: ClusterExtension) => {
    if (ext.phase === "uninstalling") return "Retry Uninstall"
    if (ext.phase === "upgrading") return "Retry Update"
    return "Retry Install"
  }

  const isTransitioning = (ext: ClusterExtension) =>
    ext.status === "pending" ||
    ext.status === "installing" ||
    ext.status === "upgrading" ||
    ext.status === "uninstalling"

  const columns: ColumnDef<ClusterExtension>[] = [
    {
      accessorKey: "name",
      header: "Extension",
      cell: ({ row }) => {
        const ext = row.original
        return (
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-purple-500/10 rounded-md text-purple-600 shrink-0">
              <Blocks className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <p className="font-medium text-sm truncate">{ext.name || ext.release_name}</p>
              <p className="text-xs text-muted-foreground font-mono truncate">
                {ext.release_name} · {ext.namespace}
              </p>
            </div>
          </div>
        )
      },
    },
    {
      accessorKey: "status",
      header: "Status",
      cell: ({ row }) => getStatusBadge(row.original),
    },
    {
      accessorKey: "namespace",
      header: "Namespace",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.namespace}</span>
      ),
    },
    {
      accessorKey: "installed_version",
      header: "Installed Version",
      cell: ({ row }) => (
        <Button
          variant="link"
          size="sm"
          onClick={() => setValuesTarget(row.original)}
        >
          {row.original.version}
        </Button>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const ext = row.original
        const busy = isTransitioning(ext)
        const failed = ext.status === "failed"
        const uninstallFailed = failed && ext.phase === "uninstalling"
        const installFailed = failed && (ext.phase === "installing" || !ext.phase)
        const upgradeFailed = failed && ext.phase === "upgrading"
        return (
          <div className="flex items-center gap-2 justify-end">
            {ext.status === "deployed" && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => setUpdateTarget(ext)}
              >
                <ArrowBigUpDash />
                Update
              </Button>
            )}
            {installFailed && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => setRetryInstallTarget(ext)}
              >
                <RotateCcw />
                {getRetryLabel(ext)}
              </Button>
            )}
            {upgradeFailed && (
              <Button
                size="sm"
                variant="outline"
                onClick={() => setUpdateTarget(ext)}
              >
                <RotateCcw />
                {getRetryLabel(ext)}
              </Button>
            )}
            {!busy && (
              <Button
                variant="destructive"
                size="sm"
                className="text-destructive hover:text-destructive hover:bg-destructive/10"
                onClick={() => uninstallFailed ? retryMutation.mutate({ extension: ext }) : setDeleteTarget(ext.id)}
                disabled={retryMutation.isPending && uninstallFailed}
              >
                <Trash2 />
                {uninstallFailed ? "Retry Uninstall" : "Uninstall"}
              </Button>
            )}
          </div>
        )
      },
    },
  ]

  const deleteExtensionName = safeClusterExtensions.find((e) => e.id === deleteTarget)?.name ?? safeClusterExtensions.find((e) => e.id === deleteTarget)?.release_name ?? deleteTarget

  return (
    <>
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Blocks className="h-4 w-4" />
            Installed Extensions
          </CardTitle>
          <CardDescription>
            Extensions currently installed in this cluster
          </CardDescription>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={columns}
            data={safeClusterExtensions}
            sourceDataCount={safeClusterExtensions.length}
            isLoading={isLoading}
            searchKey="name"
            searchPlaceholder="Filter extensions..."
            sourceEmptyContent={(
              <EmptyState
                title="No Extensions Installed"
                description="Browse the extension to discover and install extensions for this cluster."
                icon={Blocks}
                actionText="Install Extension"
                onAction={() => setBrowseOpen(true)}
                actionIcon={Download}
              />
            )}
            useStandaloneEmptyState
            rightToolbar={() => (
              <Button onClick={() => setBrowseOpen(true)}>
                <Download />
                Install Extension
              </Button>
            )}
          />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Blocks className="h-4 w-4" />
            Available Extensions
          </CardTitle>
          <CardDescription>
            Extensions available to install on this cluster
          </CardDescription>
        </CardHeader>
        <CardContent>
          <DataTable
            columns={extensionColumns}
            data={availableExtensions}
            sourceDataCount={availableExtensions.length}
            isLoading={loading}
            searchKey="name"
            searchPlaceholder="Filter available extensions..."
            sourceEmptyContent={(
              <EmptyState
                title="All Extensions Installed"
                description="All available extensions are already installed on this cluster."
                icon={Blocks}
              />
            )}
            useStandaloneEmptyState
          />
        </CardContent>
      </Card>

      <InstallExtensionToClusterDialog
        open={installOpen}
        onOpenChange={setInstallOpen}
        extension={installTarget}
        preselectedClusterId={clusterId}
      />

      <BrowseExtensionsDialog
        clusterId={clusterId}
        open={browseOpen}
        onOpenChange={setBrowseOpen}
        installedExtensionIds={installedExtensionIds}
      />

      <UpdateExtensionDialog
        open={!!updateTarget}
        onOpenChange={(open) => !open && setUpdateTarget(null)}
        clusterId={clusterId}
        extension={updateTarget}
      />

      <RetryInstallExtensionDialog
        open={!!retryInstallTarget}
        onOpenChange={(open) => !open && setRetryInstallTarget(null)}
        clusterId={clusterId}
        extension={retryInstallTarget}
      />

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Uninstall Extension</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to uninstall "{deleteExtensionName}"? This will
              remove the Helm release and all associated resources from the
              cluster.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteTarget && uninstallMutation.mutate(deleteTarget)
              }
              variant="destructive"
            >
              {uninstallMutation.isPending ? "Uninstalling..." : "Uninstall"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog
        open={!!valuesTarget}
        onOpenChange={(open) => !open && setValuesTarget(null)}
      >
        <DialogContent className="flex flex-col h-[90vh] max-h-[90vh] w-[90vw] max-w-[90vw] overflow-hidden sm:h-[90vh] sm:max-h-[90vh] sm:max-w-[90vw]">
          <DialogHeader>
            <DialogTitle>
              {valuesTarget?.version
                ? `Extension Values - ${valuesTarget.version}`
                : "Extension Values"}
            </DialogTitle>
            <DialogDescription>{valuesTarget?.name || valuesTarget?.release_name}</DialogDescription>
          </DialogHeader>
          <div className="flex-1 min-h-0 overflow-hidden rounded-md border">
            {valuesTarget?.values ? (
              <Editor
                height="100%"
                language="yaml"
                theme={monacoTheme}
                value={valuesTarget.values}
                options={{
                  readOnly: true,
                  fontSize: 12,
                  lineNumbers: "on",
                  minimap: { enabled: false },
                  scrollBeyondLastLine: false,
                  wordWrap: "on",
                  automaticLayout: true,
                  padding: { top: 16, bottom: 16 },
                }}
                loading=""
              />
            ) : (
              <div className="flex items-center justify-center h-full text-muted-foreground bg-muted/30">
                No content available
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
