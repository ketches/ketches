import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  ArrowUpFromLine,
  Blocks,
  Loader2,
  Plus,
  Trash2
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ExtensionCatalogItem,
  type InstalledExtension,
} from "@/api/clusters"
import { UpdateExtensionDialog } from "@/components/cluster/update-extension-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { BrowseExtensionsDialog } from "@/components/extensions/browse-extensions-dialog"
import { InstallExtensionToClusterDialog } from "@/components/extensions/install-extension-dialog"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
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
import type { ColumnDef } from "@tanstack/react-table"

interface ClusterExtensionsProps {
  clusterId: string
}

export function ClusterExtensions({ clusterId }: ClusterExtensionsProps) {
  const queryClient = useQueryClient()
  const [browseOpen, setBrowseOpen] = React.useState(false)
  const [deleteTarget, setDeleteTarget] = React.useState<string | null>(null)
  const [updateTarget, setUpdateTarget] =
    React.useState<InstalledExtension | null>(null)
  const [installTarget, setInstallTarget] =
    React.useState<ExtensionCatalogItem | null>(null)
  const [installOpen, setInstallOpen] = React.useState(false)

  const { data: extensions = [], isLoading } = useQuery({
    queryKey: ["extensions", clusterId],
    queryFn: () => clustersApi.listExtensions(clusterId),
    refetchInterval: 10000,
  })

  const uninstallMutation = useMutation({
    mutationFn: (name: string) =>
      clustersApi.uninstallExtension(clusterId, name),
    onSuccess: () => {
      toast.success("Extension uninstalled")
      queryClient.invalidateQueries({ queryKey: ["extensions", clusterId] })
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

  const safeExtensions: InstalledExtension[] = Array.isArray(extensions)
    ? extensions
    : []

  // Derive installed extension names to pass to BrowseExtensionsDialog
  const installedNames = safeExtensions.map((e) => e.name)

  // Fetch the full extension catalog to derive available (not-yet-installed) items
  const { data: catalog = [], isLoading: catalogLoading } = useQuery({
    queryKey: ["extension-catalog"],
    queryFn: () => clustersApi.listExtensionCatalog(),
  })

  const safeCatalog: ExtensionCatalogItem[] = Array.isArray(catalog) ? catalog : []

  // Filter out already-installed catalog items (match by catalog_item_id or by name)
  const installedIds = new Set(safeExtensions.map((e) => e.catalog_item_id).filter(Boolean))
  const installedNameSet = new Set(installedNames)
  const availableItems = safeCatalog.filter(
    (item) => !installedIds.has(item.id) && !installedNameSet.has(item.name),
  )

  const catalogColumns: ColumnDef<ExtensionCatalogItem>[] = [
    {
      accessorKey: "name",
      header: "Extension",
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-primary/10 rounded-md text-primary shrink-0">
              <Blocks />
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
              <Plus />
              Install
            </Button>
          </div>
        )
      },
    },
  ]

  const getStatusBadge = (ext: InstalledExtension) => {
    if (ext.status === "deployed") {
      return (
        <ColorBadge color="green">Completed</ColorBadge>
      )
    }
    if (ext.status === "failed") {
      return (
        <ColorBadge color="red">
          Failed
        </ColorBadge>
      )
    }
    return (
      <ColorBadge color="blue">
        {ext.status || "Pending"}
      </ColorBadge>
    )
  }

  const columns: ColumnDef<InstalledExtension>[] = [
    {
      accessorKey: "name",
      header: "Extension",
      cell: ({ row }) => {
        const ext = row.original
        return (
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-primary/10 rounded-md text-primary shrink-0">
              <Blocks />
            </div>
            <div className="min-w-0">
              <p className="font-medium text-sm truncate">{ext.name}</p>
              <p className="text-xs text-muted-foreground font-mono truncate">
                {ext.oci_url}
                {ext.chart_version ? `:${ext.chart_version}` : ""}
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
      accessorKey: "release_namespace",
      header: "Namespace",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.release_namespace}</span>
      ),
    },
    {
      accessorKey: "revision",
      header: "Revision",
      cell: ({ row }) => (
        <span className="text-xs">{row.original.revision || "-"}</span>
      ),
    },
    {
      accessorKey: "app_version",
      header: "App Version",
      cell: ({ row }) => (
        <span className="font-mono text-xs">{row.original.app_version || "-"}</span>
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
              variant="outline"
              size="sm"
              onClick={() => setUpdateTarget(ext)}
            >
              <ArrowUpFromLine />
              Update
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
              onClick={() => setDeleteTarget(ext.name)}
            >
              <Trash2 />
              Uninstall
            </Button>
          </div>
        )
      },
    },
  ]

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
              actionIcon={Plus}
            />
          ) : (
            <DataTable
              borderless
              columns={columns}
              data={safeExtensions}
              searchKey="name"
              searchPlaceholder="Filter extensions..."
              leftActions={() => (
                <Button onClick={() => setBrowseOpen(true)}>
                  <Plus />
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
        catalogItem={installTarget}
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
              Are you sure you want to uninstall "{deleteTarget}"? This will
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
    </>
  )
}
