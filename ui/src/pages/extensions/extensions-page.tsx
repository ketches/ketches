import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import {
  Blocks,
  Download,
  LayoutGrid,
  List as ListIcon,
  Pencil,
  Plus,
  Trash2,
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { clustersApi, type Extension } from "@/api/clusters"
import { AddExtensionDialog } from "@/components/cluster/add-extension-dialog"
import { EditExtensionDialog } from "@/components/cluster/edit-extension-dialog"
import { DataTable } from "@/components/data-table/data-table"
import { InstallExtensionToClusterDialog } from "@/components/extensions/install-extension-dialog"
import { InstalledClustersDialog } from "@/components/extensions/installed-clusters-dialog"
import { PageHeader } from "@/components/layout/page-header"
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
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDebounce } from "@/hooks/use-debounce"
import type { AxiosError } from "axios"

const EXTENSIONS_VIEW_MODE_KEY = "extensions_view_mode"

export function ExtensionsPage() {
  const queryClient = useQueryClient()
  const [addOpen, setAddOpen] = React.useState(false)
  const [installTarget, setInstallTarget] =
    React.useState<Extension | null>(null)
  const [installOpen, setInstallOpen] = React.useState(false)
  const [installedClustersTarget, setInstalledClustersTarget] =
    React.useState<Extension | null>(null)
  const [installedClustersOpen, setInstalledClustersOpen] =
    React.useState(false)
  const [deleteTarget, setDeleteTarget] =
    React.useState<Extension | null>(null)
  const [editTarget, setEditTarget] =
    React.useState<Extension | null>(null)
  const [editOpen, setEditOpen] = React.useState(false)

  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(EXTENSIONS_VIEW_MODE_KEY)
    return saved === "list" || saved === "card" ? saved : "list"
  })
  const [searchQuery, setSearchQuery] = React.useState("")
  const debouncedSearch = useDebounce(searchQuery, 300)
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  React.useEffect(() => {
    localStorage.setItem(EXTENSIONS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  const { data: extension = [], isLoading, refetch } = useQuery({
    queryKey: ["extensions"],
    queryFn: () => clustersApi.listExtensions(),
  })

  const safeItems: Extension[] = Array.isArray(extension)
    ? extension
    : []

  // Client-side search filter
  const filteredItems = safeItems.filter((item) => {
    if (!debouncedSearch) return true
    const q = debouncedSearch.toLowerCase()
    return (
      (item.display_name || item.name).toLowerCase().includes(q) ||
      item.oci_url.toLowerCase().includes(q) ||
      (item.description ?? "").toLowerCase().includes(q)
    )
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => clustersApi.deleteExtension(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["extensions"] })
      toast.success("Extension removed")
      setDeleteTarget(null)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to remove extension", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const columns: ColumnDef<Extension>[] = [
    {
      accessorKey: "name",
      header: "Extension",
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex items-center gap-2">
            <div className="p-1.5 bg-blue-500/10 rounded-md text-blue-600 shrink-0">
              <Blocks className="h-4 w-4" />
            </div>
            <div className="min-w-0">
              <p className="font-medium text-xs truncate">
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
      id: "install_count",
      header: "Installs",
      cell: ({ row }) => {
        const item = row.original
        return (
          <Button
            variant="link"
            className="p-0 h-auto font-normal"
            onClick={() => {
              setInstalledClustersTarget(item)
              setInstalledClustersOpen(true)
            }}
          >
            {row.original.install_count}
          </Button>
        )
      },
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const item = row.original
        return (
          <div className="flex items-center gap-2 justify-end">
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => {
                setInstallTarget(item)
                setInstallOpen(true)
              }}
            >
              <Download />
            </Button>
            {!item.builtin && (
              <>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => {
                    setEditTarget(item)
                    setEditOpen(true)
                  }}
                >
                  <Pencil />
                </Button>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={() => setDeleteTarget(item)}
                >
                  <Trash2 />
                </Button>
              </>
            )}
          </div>
        )
      },
    },
  ]

  const breadcrumbs = [{ label: "Extensions", icon: Blocks }]

  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search extensions..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const toolbarRight = (
    <div className="flex items-center gap-2">
      <Tabs
        value={viewMode}
        onValueChange={(v) => {
          const newMode = v as "list" | "card"
          setViewMode(newMode)
          setPagination((prev) => ({
            ...prev,
            pageIndex: 0,
            pageSize: newMode === "card" ? 9 : 10,
          }))
        }}
        className="w-auto h-7"
      >
        <TabsList>
          <TabsTrigger value="list">
            <ListIcon />
          </TabsTrigger>
          <TabsTrigger value="card">
            <LayoutGrid />
          </TabsTrigger>
        </TabsList>
      </Tabs>
      <Button onClick={() => setAddOpen(true)}>
        <Plus />
        Add Extension
      </Button>
    </div>
  )

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 gap-6">
        <PageHeader items={breadcrumbs} />
        <div className="flex items-center justify-center flex-1">
          <div className="text-muted-foreground animate-pulse">
            Loading extensions...
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      {safeItems.length === 0 && !searchQuery ? (
        <EmptyState
          title="No extensions found"
          description="Add OCI-based Helm chart extensions to make them available for installation on clusters."
          icon={Blocks}
          actionText="Add Extension"
          onAction={() => setAddOpen(true)}
          actionIcon={Plus}
        />
      ) : (
        <>
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-bold">Extensions</h1>
              <p className="text-sm text-muted-foreground mt-1">
                Manage OCI-based Helm chart extensions
              </p>
            </div>
          </div>
          <DataTable
            columns={columns}
            data={filteredItems}
            viewMode={viewMode}
            onRefresh={refetch}
            manualPagination
            totalCount={filteredItems.length}
            pagination={pagination}
            onPaginationChange={setPagination}
            leftActions={() => toolbarLeft}
            toolbarActions={() => toolbarRight}
            renderCard={(item) => (
              <Card
                key={item.id}
                className="group/card hover:shadow-md transition-shadow h-full"
              >
                <CardHeader className="pb-2">
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex items-start gap-3 min-w-0">
                      <div className="p-2 bg-blue-500/10 rounded-lg text-blue-600 shrink-0">
                        <Blocks className="h-5 w-5" />
                      </div>
                      <div className="flex flex-col min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <CardTitle className="text-base font-semibold truncate">
                            {item.display_name || item.name}
                          </CardTitle>
                          {item.builtin ? (
                            <ColorBadge color="purple" className="text-[10px]">
                              Built-in
                            </ColorBadge>
                          ) : (
                            <ColorBadge color="gray" className="text-[10px]">
                              Custom
                            </ColorBadge>
                          )}
                        </div>
                        <p className="text-[10px] text-muted-foreground font-mono truncate">
                          {item.oci_url}
                        </p>
                      </div>
                    </div>
                    <div
                      className="flex items-center gap-1 shrink-0"
                      onClick={(e) => e.stopPropagation()}
                    >
                      <Button
                        variant="ghost"
                        size="icon-sm"
                        onClick={() => {
                          setInstallTarget(item)
                          setInstallOpen(true)
                        }}
                      >
                        <Download />
                      </Button>
                      {!item.builtin && (
                        <>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            onClick={() => {
                              setEditTarget(item)
                              setEditOpen(true)
                            }}
                          >
                            <Pencil />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon-sm"
                            className="text-destructive hover:text-destructive hover:bg-destructive/10"
                            onClick={() => setDeleteTarget(item)}
                          >
                            <Trash2 />
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="pt-2">
                  {item.description && (
                    <p className="text-xs text-muted-foreground line-clamp-2">
                      {item.description}
                    </p>
                  )}
                  <div className="mt-3">
                    <Button
                      variant="link"
                      className="p-0 h-auto text-xs font-normal text-muted-foreground hover:text-foreground"
                      onClick={() => {
                        setInstalledClustersTarget(item)
                        setInstalledClustersOpen(true)
                      }}
                    >
                      View installed clusters →
                    </Button>
                  </div>
                </CardContent>
              </Card>
            )}
          />
        </>
      )}

      <AddExtensionDialog open={addOpen} onOpenChange={setAddOpen} />

      <EditExtensionDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        item={editTarget}
      />

      <InstallExtensionToClusterDialog
        open={installOpen}
        onOpenChange={setInstallOpen}
        extension={installTarget}
      />

      <InstalledClustersDialog
        open={installedClustersOpen}
        onOpenChange={setInstalledClustersOpen}
        extension={installedClustersTarget}
      />

      <AlertDialog
        open={!!deleteTarget}
        onOpenChange={() => setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Remove Extension</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to remove "
              {deleteTarget?.display_name || deleteTarget?.name}"? Installed extensions will not be affected.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setDeleteTarget(null)}>
              Cancel
            </AlertDialogCancel>
            <AlertDialogAction
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.id)
              }
              disabled={deleteMutation.isPending}
              variant="destructive"
            >
              {deleteMutation.isPending ? "Removing..." : "Remove"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default ExtensionsPage
