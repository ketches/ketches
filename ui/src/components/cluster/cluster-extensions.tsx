import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  Blocks,
  CheckCircle2,
  Clock,
  Library,
  Loader2,
  Plus,
  Trash2,
  XCircle,
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  clustersApi,
  type ExtensionCatalogItem,
  type InstalledExtension,
} from "@/api/clusters"
import { AddExtensionCatalogDialog } from "@/components/cluster/add-extension-catalog-dialog"
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
  return <ExtensionManager clusterId={clusterId} />
}

// ========================
// Extension Manager
// ========================

function ExtensionManager({ clusterId }: { clusterId: string }) {
  const [activeTab, setActiveTab] = React.useState("catalog")

  return (
    <div className="space-y-4">
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="catalog">
            <Library />
            Catalog
          </TabsTrigger>
          <TabsTrigger value="installed">
            <Blocks />
            Installed
          </TabsTrigger>
        </TabsList>

        <TabsContent value="catalog" className="mt-2">
          <ExtensionCatalog clusterId={clusterId} />
        </TabsContent>

        <TabsContent value="installed" className="mt-2">
          <InstalledExtensions clusterId={clusterId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

// ========================
// Extension Catalog Tab
// ========================

function ExtensionCatalog({ clusterId }: { clusterId: string }) {
  const queryClient = useQueryClient()
  const [addOpen, setAddOpen] = React.useState(false)
  const [installTarget, setInstallTarget] =
    React.useState<ExtensionCatalogItem | null>(null)
  const [installOpen, setInstallOpen] = React.useState(false)
  const [deleteTarget, setDeleteTarget] =
    React.useState<ExtensionCatalogItem | null>(null)

  const { data: catalog = [], isLoading } = useQuery({
    queryKey: ["extension-catalog"],
    queryFn: () => clustersApi.listExtensionCatalog(),
  })

  const deleteMutation = useMutation({
    mutationFn: (itemId: string) =>
      clustersApi.deleteExtensionCatalogItem(itemId),
    onSuccess: () => {
      toast.success("Extension removed from catalog")
      queryClient.invalidateQueries({ queryKey: ["extension-catalog"] })
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
      toast.error("Failed to remove extension", {
        description:
          msg ?? (error instanceof Error ? error.message : String(error)),
      })
    },
  })

  const safeItems: ExtensionCatalogItem[] = Array.isArray(catalog)
    ? catalog
    : []

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

  return (
    <>
      <div className="space-y-4">
        <div className="flex justify-end">
          <Button onClick={() => setAddOpen(true)}>
            <Plus />
            Add Extension
          </Button>
        </div>

        {safeItems.length === 0 ? (
          <Card>
            <CardContent className="py-8">
              <Empty className="border-0 flex-1">
                <EmptyHeader>
                  <EmptyMedia variant="icon">
                    <Library className="h-6 w-6" />
                  </EmptyMedia>
                  <EmptyTitle>No Extensions in Catalog</EmptyTitle>
                  <EmptyDescription>
                    Add OCI-based Helm chart extensions to make them available
                    for installation.
                  </EmptyDescription>
                  <EmptyContent>
                    <Button onClick={() => setAddOpen(true)}>
                      <Plus />
                      Add Extension
                    </Button>
                  </EmptyContent>
                </EmptyHeader>
              </Empty>
            </CardContent>
          </Card>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {safeItems.map((item) => (
              <Card key={item.id} className="flex flex-col">
                <CardHeader className="pb-3">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-2 min-w-0">
                      <div className="p-1.5 bg-blue-500/10 rounded-md text-blue-600 shrink-0">
                        <Blocks className="h-4 w-4" />
                      </div>
                      <div className="min-w-0">
                        <CardTitle className="text-sm truncate">
                          {item.display_name || item.name}
                        </CardTitle>
                        <p className="text-xs text-muted-foreground font-mono truncate">
                          {item.oci_url}
                        </p>
                      </div>
                    </div>
                    {item.builtin && (
                      <Badge variant="secondary" className="text-[10px] shrink-0">
                        Built-in
                      </Badge>
                    )}
                  </div>
                </CardHeader>
                <CardContent className="flex-1 pb-3">
                  {item.description && (
                    <p className="text-xs text-muted-foreground line-clamp-2">
                      {item.description}
                    </p>
                  )}
                </CardContent>
                <div className="flex gap-2 px-6 pb-4 pt-0">
                  <Button
                    size="sm"
                    className="flex-1"
                    onClick={() => {
                      setInstallTarget(item)
                      setInstallOpen(true)
                    }}
                  >
                    Install
                  </Button>
                  {!item.builtin && (
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => setDeleteTarget(item)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </Button>
                  )}
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>

      <AddExtensionCatalogDialog open={addOpen} onOpenChange={setAddOpen} />

      <InstallExtensionDialog
        open={installOpen}
        onOpenChange={setInstallOpen}
        clusterId={clusterId}
        catalogItem={installTarget}
      />

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove Extension from Catalog</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove "
              {deleteTarget?.display_name || deleteTarget?.name}" from the
              catalog? Installed extensions will not be affected.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.id)
              }
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Removing..." : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
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
  const [updateTarget, setUpdateTarget] =
    React.useState<InstalledExtension | null>(null)

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

  if (isLoading) {
    return (
      <Card>
        <CardContent className="py-12">
          <div className="flex flex-col items-center justify-center gap-4">
            <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
            <p className="text-sm text-muted-foreground">
              Loading extensions...
            </p>
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
          <InstalledExtensionCard
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

function InstalledExtensionCard({
  extension,
  onUpdate,
  onUninstall,
}: {
  extension: InstalledExtension
  onUpdate: () => void
  onUninstall: () => void
}) {
  const getStatusBadge = () => {
    if (extension.status === "deployed") {
      return (
        <Badge variant="outline" className="text-green-600 gap-1">
          <CheckCircle2 className="h-3 w-3" />
          Deployed
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
        {extension.status || "Pending"}
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
                {extension.oci_url}
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
        </div>
      </CardContent>
      <div className="flex gap-2 px-6 pb-4 pt-0">
        <Button
          variant="outline"
          size="sm"
          className="flex-1"
          onClick={onUpdate}
        >
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
