import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { ArchiveRestore, Box, GalleryVerticalEnd, Loader2, Orbit, RotateCcw, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { recycleBinApi, type RecycleBinApp, type RecycleBinEnv, type RecycleBinProject } from "@/api/recycle-bin"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import { EmptyState } from "@/components/shared/empty-state"
import { AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle } from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Input } from "@/components/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger
} from "@/components/ui/tooltip"
import { useDebounce } from "@/hooks/use-debounce"

import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { useProjectRole } from "@/hooks/useProjectRole"
const formatDate = (dateString: string) => {
  if (!dateString) return "-"
  const date = new Date(dateString)
  return date.toLocaleString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit'
  })
}

export function RecycleBinPage() {
  const queryClient = useQueryClient()
  const [searchQuery, setSearchQuery] = React.useState("")
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const debouncedSearch = useDebounce(searchQuery, 300)

  const [selectedAppRows, setSelectedAppRows] = React.useState({})
  const [selectedEnvRows, setSelectedEnvRows] = React.useState({})
  const [selectedProjectRows, setSelectedProjectRows] = React.useState({})

  const [restoreDialogOpen, setRestoreDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [conflictDialogOpen, setConflictDialogOpen] = React.useState(false)
  const [conflictApps, setConflictApps] = React.useState<RecycleBinApp[]>([])

  const [activeTab, setActiveTab] = React.useState<"projects" | "apps" | "envs">("projects")
  const [restoringItemId, setRestoringItemId] = React.useState<string | null>(null)
  const [deletingItemId, setDeletingItemId] = React.useState<string | null>(null)

  const [appsPagination, setAppsPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const [envsPagination, setEnvsPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const [projectsPagination, setProjectsPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const { data: appsResponse, isLoading: appsLoading, refetch: refetchApps } = useQuery({
    queryKey: ['recycle-bin-apps', debouncedSearch, appsPagination.pageIndex, appsPagination.pageSize],
    queryFn: () => recycleBinApi.listApps(undefined, {
      search: debouncedSearch,
      page: appsPagination.pageIndex + 1,
      page_size: appsPagination.pageSize
    }),
  })

  const apps = React.useMemo(() => appsResponse?.items ?? [], [appsResponse])
  const appsPaginationInfo = appsResponse?.pagination

  const { data: envsResponse, isLoading: envsLoading, refetch: refetchEnvs } = useQuery({
    queryKey: ['recycle-bin-envs', debouncedSearch, envsPagination.pageIndex, envsPagination.pageSize],
    queryFn: () => recycleBinApi.listEnvs(undefined, {
      search: debouncedSearch,
      page: envsPagination.pageIndex + 1,
      page_size: envsPagination.pageSize
    }),
  })

  const envs = React.useMemo(() => envsResponse?.items ?? [], [envsResponse])
  const envsPaginationInfo = envsResponse?.pagination

  const { data: projectsResponse, isLoading: projectsLoading, refetch: refetchProjects } = useQuery({
    queryKey: ['recycle-bin-projects', debouncedSearch, projectsPagination.pageIndex, projectsPagination.pageSize],
    queryFn: () => recycleBinApi.listProjects({
      search: debouncedSearch,
      page: projectsPagination.pageIndex + 1,
      page_size: projectsPagination.pageSize
    }),
  })

  const projects = React.useMemo(() => projectsResponse?.items ?? [], [projectsResponse])
  const projectsPaginationInfo = projectsResponse?.pagination

  const selectedAppIds = React.useMemo(() => {
    return Object.keys(selectedAppRows).filter(key => (selectedAppRows as Record<string, boolean>)[key]).map(index => apps[parseInt(index)]?.id).filter(Boolean)
  }, [selectedAppRows, apps])

  const selectedEnvIds = React.useMemo(() => {
    return Object.keys(selectedEnvRows).filter(key => (selectedEnvRows as Record<string, boolean>)[key]).map(index => envs[parseInt(index)]?.id).filter(Boolean)
  }, [selectedEnvRows, envs])

  const selectedProjectIds = React.useMemo(() => {
    return Object.keys(selectedProjectRows).filter(key => (selectedProjectRows as Record<string, boolean>)[key]).map(index => projects[parseInt(index)]?.id).filter(Boolean)
  }, [selectedProjectRows, projects])

  const restoreAppsMutation = useMutation({
    mutationFn: (ids: string[]) => recycleBinApi.restoreApps(ids),
    onSuccess: () => {
      toast.success("Applications restored")
      queryClient.invalidateQueries({ queryKey: ['recycle-bin-apps'] })
      setSelectedAppRows({})
      setRestoreDialogOpen(false)
      setRestoringItemId(null)
    },
    onError: (error: any) => {
      toast.error("Failed to restore applications", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setRestoringItemId(null)
    },
  })

  const deleteAppsMutation = useMutation({
    mutationFn: (ids: string[]) => recycleBinApi.permanentlyDeleteApps(ids),
    onSuccess: () => {
      toast.success("Applications permanently deleted")
      queryClient.invalidateQueries({ queryKey: ['recycle-bin-apps'] })
      setSelectedAppRows({})
      setDeleteDialogOpen(false)
      setDeletingItemId(null)
    },
    onError: (error: any) => {
      toast.error("Failed to delete applications", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setDeletingItemId(null)
    },
  })

  const restoreEnvsMutation = useMutation({
    mutationFn: (ids: string[]) => recycleBinApi.restoreEnvs(ids),
    onSuccess: () => {
      toast.success("Environments restored")
      queryClient.invalidateQueries({ queryKey: ['recycle-bin-envs'] })
      setSelectedEnvRows({})
      setRestoreDialogOpen(false)
      setRestoringItemId(null)
    },
    onError: (error: any) => {
      toast.error("Failed to restore environments", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setRestoringItemId(null)
    },
  })

  const deleteEnvsMutation = useMutation({
    mutationFn: async (ids: string[]) => {
      for (const id of ids) {
        const conflicts = await recycleBinApi.checkEnvDeletionConflicts(id)
        if (conflicts.apps && conflicts.apps.length > 0) {
          setConflictApps(conflicts.apps)
          setConflictDialogOpen(true)
          throw new Error("Environment has deleted applications")
        }
      }
      return recycleBinApi.permanentlyDeleteEnvs(ids)
    },
    onSuccess: () => {
      toast.success("Environments permanently deleted")
      queryClient.invalidateQueries({ queryKey: ['recycle-bin-envs'] })
      setSelectedEnvRows({})
      setDeleteDialogOpen(false)
    },
    onError: (error: any) => {
      if (error.message !== "Environment has deleted applications") {
        toast.error("Failed to delete environments", {
          description: error.response?.data?.error || "An unknown error occurred",
        })
      }
    },
  })

  const restoreProjectsMutation = useMutation({
    mutationFn: (ids: string[]) => recycleBinApi.restoreProjects(ids),
    onSuccess: () => {
      toast.success("Projects restored")
      queryClient.invalidateQueries({ queryKey: ['recycle-bin-projects'] })
      setSelectedProjectRows({})
      setRestoreDialogOpen(false)
      setRestoringItemId(null)
    },
    onError: (error: any) => {
      toast.error("Failed to restore projects", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setRestoringItemId(null)
    },
  })

  const deleteProjectsMutation = useMutation({
    mutationFn: (ids: string[]) => recycleBinApi.permanentlyDeleteProjects(ids),
    onSuccess: () => {
      toast.success("Projects permanently deleted")
      queryClient.invalidateQueries({ queryKey: ['recycle-bin-projects'] })
      setSelectedProjectRows({})
      setDeleteDialogOpen(false)
      setDeletingItemId(null)
    },
    onError: (error: any) => {
      toast.error("Failed to delete projects", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
      setDeletingItemId(null)
    },
  })

  const handleRestore = () => {
    if (activeTab === "projects" && selectedProjectIds.length > 0) {
      restoreProjectsMutation.mutate(selectedProjectIds)
    } else if (activeTab === "apps" && selectedAppIds.length > 0) {
      restoreAppsMutation.mutate(selectedAppIds)
    } else if (activeTab === "envs" && selectedEnvIds.length > 0) {
      restoreEnvsMutation.mutate(selectedEnvIds)
    }
  }

  const handleDelete = () => {
    if (activeTab === "projects" && selectedProjectIds.length > 0) {
      deleteProjectsMutation.mutate(selectedProjectIds)
    } else if (activeTab === "apps" && selectedAppIds.length > 0) {
      deleteAppsMutation.mutate(selectedAppIds)
    } else if (activeTab === "envs" && selectedEnvIds.length > 0) {
      deleteEnvsMutation.mutate(selectedEnvIds)
    }
  }

  const handleRestoreSingle = (id: string, type: "project" | "app" | "env") => {
    setRestoringItemId(id)
    if (type === "project") {
      restoreProjectsMutation.mutate([id])
    } else if (type === "app") {
      restoreAppsMutation.mutate([id])
    } else {
      restoreEnvsMutation.mutate([id])
    }
  }

  const handleDeleteSingle = (id: string, type: "project" | "app" | "env") => {
    setDeletingItemId(id)
    if (type === "project") {
      deleteProjectsMutation.mutate([id])
    } else if (type === "app") {
      deleteAppsMutation.mutate([id])
    } else {
      deleteEnvsMutation.mutate([id])
    }
  }

  const projectColumns: ColumnDef<RecycleBinProject>[] = [
    {
      accessorKey: "name",
      header: "Project",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <Avatar className="h-8 w-8 rounded-lg bg-primary/10 text-primary border-none">
            <AvatarFallback className="rounded-lg text-xs font-bold">
              {row.original.name.charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <div className="flex flex-col">
            <span className="font-medium text-foreground">{row.original.name}</span>
            <span className="text-xs text-muted-foreground font-mono">{row.original.slug}</span>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "description",
      header: "Description",
    },
    {
      accessorKey: "deleted_at",
      header: "Deleted At",
      cell: ({ row }) => formatDate(row.original.deleted_at),
    },
  ]

  if (!isViewer) {
    projectColumns.unshift({
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    })
    projectColumns.push({
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-2">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleRestoreSingle(row.original.id, "project")
                  }}
                  disabled={restoringItemId === row.original.id}
                />
              }
            >
              <ArchiveRestore />
            </TooltipTrigger>
            <TooltipContent>Restore project</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleDeleteSingle(row.original.id, "project")
                  }}
                  disabled={deletingItemId === row.original.id}
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>Permanently delete</TooltipContent>
          </Tooltip>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    })
  }

  const appColumns: ColumnDef<RecycleBinApp>[] = [
    {
      accessorKey: "name",
      header: "Application",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-blue-500/10 rounded-md text-blue-600 shrink-0">
            <Box className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="font-medium text-xs truncate">
              {row.original.name}
            </p>
            <p className="text-xs text-muted-foreground font-mono truncate">
              {row.original.slug}
            </p>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "env_name",
      header: "Environment",
    },
    {
      accessorKey: "project_name",
      header: "Project",
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.project_name}</div>
          <div className="text-sm text-muted-foreground">{row.original.project_slug}</div>
        </div>
      ),
    },
    {
      accessorKey: "app_type",
      header: "Type",
    },
    {
      accessorKey: "deleted_at",
      header: "Deleted At",
      cell: ({ row }) => formatDate(row.original.deleted_at),
    },
  ]

  if (!isViewer) {
    appColumns.unshift({
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    })
    appColumns.push({
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-2">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleRestoreSingle(row.original.id, "app")
                  }}
                  disabled={restoringItemId === row.original.id}
                />
              }
            >
              <ArchiveRestore />
            </TooltipTrigger>
            <TooltipContent>Restore application</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleDeleteSingle(row.original.id, "app")
                  }}
                  disabled={deletingItemId === row.original.id}
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>Permanently delete</TooltipContent>
          </Tooltip>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    })
  }

  const envColumns: ColumnDef<RecycleBinEnv>[] = [
    {
      accessorKey: "name",
      header: "Environment",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <div className="p-1.5 bg-green-500/10 rounded-md text-green-600 shrink-0">
            <Box className="h-4 w-4" />
          </div>
          <div className="min-w-0">
            <p className="font-medium text-xs truncate">
              {row.original.name}
            </p>
            <p className="text-xs text-muted-foreground font-mono truncate">
              {row.original.slug}
            </p>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "project_name",
      header: "Project",
      cell: ({ row }) => (
        <div>
          <div className="font-medium">{row.original.project_name}</div>
          <div className="text-sm text-muted-foreground">{row.original.project_slug}</div>
        </div>
      ),
    },
    {
      accessorKey: "cluster_name",
      header: "Cluster",
    },
    {
      accessorKey: "deleted_at",
      header: "Deleted At",
      cell: ({ row }) => formatDate(row.original.deleted_at),
    },
  ]

  if (!isViewer) {
    envColumns.unshift({
      id: "select",
      header: ({ table }) => (
        <Checkbox
          checked={table.getIsAllPageRowsSelected()}
          onCheckedChange={(value) => table.toggleAllPageRowsSelected(!!value)}
          aria-label="Select all"
        />
      ),
      cell: ({ row }) => (
        <Checkbox
          checked={row.getIsSelected()}
          onCheckedChange={(value) => row.toggleSelected(!!value)}
          aria-label="Select row"
        />
      ),
      enableSorting: false,
      enableHiding: false,
    })
    envColumns.push({
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-2">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleRestoreSingle(row.original.id, "env")
                  }}
                  disabled={restoringItemId === row.original.id}
                />
              }
            >
              <ArchiveRestore />
            </TooltipTrigger>
            <TooltipContent>Restore environment</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={(e) => {
                    e.stopPropagation()
                    handleDeleteSingle(row.original.id, "env")
                  }}
                  disabled={deletingItemId === row.original.id}
                />
              }
            >
              <Trash2 />
            </TooltipTrigger>
            <TooltipContent>Permanently delete</TooltipContent>
          </Tooltip>
        </div>
      ),
      enableSorting: false,
      enableHiding: false,
    })
  }
  const selectedCount = activeTab === "projects" ? selectedProjectIds.length : activeTab === "apps" ? selectedAppIds.length : selectedEnvIds.length

  const breadcrumbs = [
    { label: "Recycle Bin", icon: Trash2 }
  ]

  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search deleted resources..."
      value={searchQuery}
      onChange={(e) => setSearchQuery(e.target.value)}
    />
  )

  const batchActions = (table: any) => {
    const count = table.getFilteredSelectedRowModel().rows.length
    if (count === 0) return null

    return (
      <>
        <Button
          variant="outline"
          onClick={() => setRestoreDialogOpen(true)}
        >
          <RotateCcw />
          Restore ({count})
        </Button>
        <Button
          variant="destructive"
          onClick={() => setDeleteDialogOpen(true)}
        >
          <Trash2 />
          Delete ({count})
        </Button>
      </>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">Recycle Bin</h1>
          <p className="text-sm text-muted-foreground mt-1">
            Restore or permanently delete soft-deleted resources
          </p>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={(v) => setActiveTab(v as "projects" | "apps" | "envs")}>
        <TabsList>
          <TabsTrigger value="projects">Projects ({projectsPaginationInfo?.total || 0})</TabsTrigger>
          <TabsTrigger value="apps">Applications ({appsPaginationInfo?.total || 0})</TabsTrigger>
          <TabsTrigger value="envs">Environments ({envsPaginationInfo?.total || 0})</TabsTrigger>
        </TabsList>

        <TabsContent value="projects" className="mt-2">
          {projectsLoading && !projectsResponse ? (
            <div className="flex flex-col items-center justify-center min-h-100">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : projects.length === 0 ? (
            <EmptyState
              title="No deleted projects"
              description="Deleted projects will appear here. You can restore or permanently delete them."
              icon={GalleryVerticalEnd}
            />
          ) : (
            <DataTable
              columns={projectColumns}
              data={projects}
              leftActions={() => toolbarLeft}
              batchActions={!isViewer ? batchActions : undefined}
              rowSelection={selectedProjectRows}
              onRowSelectionChange={setSelectedProjectRows}
              onRefresh={refetchProjects}
              manualPagination
              totalCount={projectsPaginationInfo?.total || 0}
              pagination={projectsPagination}
              onPaginationChange={setProjectsPagination}
            />
          )}
        </TabsContent>

        <TabsContent value="apps" className="mt-2">
          {appsLoading && !appsResponse ? (
            <div className="flex flex-col items-center justify-center min-h-100">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : apps.length === 0 ? (
            <EmptyState
              title="No deleted applications"
              description="Deleted applications will appear here. You can restore or permanently delete them."
              icon={Box}
            />
          ) : (
            <DataTable
              columns={appColumns}
              data={apps}
              leftActions={() => toolbarLeft}
              batchActions={!isViewer ? batchActions : undefined}
              rowSelection={selectedAppRows}
              onRowSelectionChange={setSelectedAppRows}
              onRefresh={refetchApps}
              manualPagination
              totalCount={appsPaginationInfo?.total || 0}
              pagination={appsPagination}
              onPaginationChange={setAppsPagination}
            />
          )}
        </TabsContent>

        <TabsContent value="envs" className="mt-2">
          {envsLoading && !envsResponse ? (
            <div className="flex flex-col items-center justify-center min-h-100">
              <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            </div>
          ) : envs.length === 0 ? (
            <EmptyState
              title="No deleted environments"
              description="Deleted environments will appear here. You can restore or permanently delete them."
              icon={Orbit}
            />
          ) : (
            <DataTable
              columns={envColumns}
              data={envs}
              leftActions={() => toolbarLeft}
              batchActions={!isViewer ? batchActions : undefined}
              rowSelection={selectedEnvRows}
              onRowSelectionChange={setSelectedEnvRows}
              onRefresh={refetchEnvs}
              manualPagination
              totalCount={envsPaginationInfo?.total || 0}
              pagination={envsPagination}
              onPaginationChange={setEnvsPagination}
            />
          )}
        </TabsContent>
      </Tabs>

      <AlertDialog open={restoreDialogOpen} onOpenChange={setRestoreDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Restore Resources</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to restore {selectedCount} {activeTab === "projects" ? "project(s)" : activeTab === "apps" ? "application(s)" : "environment(s)"}?
              This will make them visible and usable again.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleRestore}>Restore</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Permanently Delete Resources</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to permanently delete {selectedCount} {activeTab === "projects" ? "project(s)" : activeTab === "apps" ? "application(s)" : "environment(s)"}?
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete} variant="destructive">
              Permanently Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={conflictDialogOpen} onOpenChange={setConflictDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Cannot Delete Environment</AlertDialogTitle>
            <AlertDialogDescription>
              This environment contains {conflictApps.length} deleted application(s). Please permanently delete or restore these applications first:
            </AlertDialogDescription>
          </AlertDialogHeader>
          <div className="max-h-50 overflow-y-auto">
            <ul className="list-disc pl-6">
              {conflictApps.map(app => (
                <li key={app.id}>{app.name} ({app.slug})</li>
              ))}
            </ul>
          </div>
          <AlertDialogFooter>
            <AlertDialogAction onClick={() => setConflictDialogOpen(false)}>OK</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
