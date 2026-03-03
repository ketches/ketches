import Editor from "@monaco-editor/react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ArrowBigUpDash,
  Blocks,
  Download,
  Loader2,
  Trash2
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ClusterExtension,
  type Extension,
} from "@/api/clusters"
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
      const isInstalling = data?.some(
        (e) => e.status === "pending" || e.status === "installing"
      )
      return isInstalling ? 3000 : 10000
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

  const safeExtensions: ClusterExtension[] = Array.isArray(extensions)
    ? extensions
    : []

  // Derive installed extension names to pass to BrowseExtensionsDialog
  const installedNames = safeExtensions.map((e) => e.release_name)

  // Fetch the full extension catalog to derive available (not-yet-installed) items
  const { data: catalog = [], isLoading: catalogLoading } = useQuery({
    queryKey: ["extensions"],
    queryFn: () => clustersApi.listExtensions(),
  })

  const safeCatalog: Extension[] = Array.isArray(catalog) ? catalog : []

  // Filter out already-installed catalog items (match by extension_id)
  const installedIds = new Set(safeExtensions.map((e) => e.extension_id))
  const availableItems = safeCatalog.filter(
    (item) => !installedIds.has(item.id)
  )

  const catalogColumns: ColumnDef<Extension>[] = [
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
                {item.display_name || item.name}
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
      id: "catalog-actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex justify-end">
            <Button
              size="sm"
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
    if (ext.status === "deployed") return <ColorBadge color="green">Completed</ColorBadge>
    if (ext.status === "failed") return <ColorBadge color="red">Failed</ColorBadge>

    return <ColorBadge color="gray">{ext.status || "Unknown"}</ColorBadge>
  }

  const columns: ColumnDef<ClusterExtension>[] = [
    {
      accessorKey: "release_name",
      header: "Extension",
      cell: ({ row }) => {
        const ext = row.original
        return (
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-purple-500/10 rounded-md text-purple-600 shrink-0">
              <Blocks className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <p className="font-medium text-sm truncate">{ext.release_name}</p>
              <p className="text-xs text-muted-foreground font-mono truncate">
                {ext.namespace}
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
        return (
          <div className="flex items-center gap-2 justify-end">
            <Button
              size="sm"
              onClick={() => setUpdateTarget(ext)}
            >
              <ArrowBigUpDash />
              Update
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => setDeleteTarget(ext.id)}
            >
              <Trash2 />
              Uninstall
            </Button>
          </div>
        )
      },
    },
  ]

  const deleteReleaseName = safeExtensions.find(e => e.id === deleteTarget)?.release_name ?? deleteTarget

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
          {isLoading ? (
            <div className="flex flex-col items-center justify-center gap-4 py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                Loading extensions...
              </p>
            </div>
          ) : safeExtensions.length === 0 ? (
            <EmptyState
              title="No Extensions Installed"
              description="Browse the extension catalog to discover and install extensions for this cluster."
              icon={Blocks}
              actionText="Install Extension"
              onAction={() => setBrowseOpen(true)}
              actionIcon={Download}
            />
          ) : (
            <DataTable
              borderless
              columns={columns}
              data={safeExtensions}
              searchKey="release_name"
              searchPlaceholder="Filter extensions..."
              leftActions={() => (
                <Button onClick={() => setBrowseOpen(true)}>
                  <Download />
                  Install Extension
                </Button>
              )}
            />
          )}
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
          {catalogLoading ? (
            <div className="flex flex-col items-center justify-center gap-4 py-12">
              <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
              <p className="text-sm text-muted-foreground">Loading catalog...</p>
            </div>
          ) : availableItems.length === 0 ? (
            <EmptyState
              title="All Extensions Installed"
              description="All available extensions from the catalog are already installed on this cluster."
              icon={Blocks}
            />
          ) : (
            <DataTable
              borderless
              columns={catalogColumns}
              data={availableItems}
              searchKey="name"
              searchPlaceholder="Filter available extensions..."
            />
          )}
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
        installedExtensionNames={installedNames}
      />

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
              Are you sure you want to uninstall "{deleteReleaseName}"? This will
              remove the Helm release and all associated resources from the
              cluster.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteTarget && uninstallMutation.mutate(deleteTarget)
              }
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
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
            <DialogDescription>{valuesTarget?.release_name}</DialogDescription>
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

