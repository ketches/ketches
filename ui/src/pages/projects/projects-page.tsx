import { projectsApi, type Project } from "@/api/projects"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import CreateProjectDialog from "@/components/project/create-project-dialog"
import EditProjectDialog from "@/components/project/edit-project-dialog"
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
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useDebounce } from "@/hooks/use-debounce"
import { formatDate } from "@/lib/utils"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { Clock, GalleryVerticalEnd, LayoutGrid, List as ListIcon, Loader2, LogIn, Pencil, Plus, Trash2, UserCog } from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

// LocalStorage key for persisting view mode preference
const PROJECTS_VIEW_MODE_KEY = "projects_view_mode"

export function ProjectsPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { activeProjectId, setActiveProjectId } = useProjectStore()
  const isAdmin = useAuthStore((state) => state.user?.role === "admin")

  // View mode persisted in localStorage, defaulting to "list"
  const [viewMode, setViewMode] = React.useState<"list" | "card">(() => {
    const saved = localStorage.getItem(PROJECTS_VIEW_MODE_KEY)
    return saved === "list" || saved === "card" ? saved : "list"
  })

  // Persist view mode changes
  React.useEffect(() => {
    localStorage.setItem(PROJECTS_VIEW_MODE_KEY, viewMode)
  }, [viewMode])

  // Server-side pagination state
  const [pagination, setPagination] = React.useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const [search, setSearch] = React.useState("")
  const debouncedSearch = useDebounce(search, 300)

  // Reset to first page when search changes
  React.useEffect(() => {
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }, [debouncedSearch])

  // Dialog states
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [selectedProject, setSelectedProject] = React.useState<Project | null>(null)

  // Fetch projects with server-side pagination and search
  const { data: projectsResponse, refetch, isLoading } = useQuery({
    queryKey: ['projects', debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => projectsApi.list({
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
    }),
    placeholderData: (prev) => prev,
  })

  // Form states
  const [_formData, setFormData] = React.useState({
    name: "",
    slug: "",
    description: ""
  })
  const [_errors, setErrors] = React.useState<Record<string, string>>({})

  const projects = projectsResponse?.items ?? []
  const paginationInfo = projectsResponse?.pagination

  const deleteMutation = useMutation({
    mutationFn: projectsApi.delete,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      toast.success("Project deleted successfully")
      setDeleteDialogOpen(false)
      setSelectedProject(null)
    },
    onError: (error: any) => {
      toast.error("Failed to delete project", {
        description: error.response?.data?.error || "Unknown error occurred"
      })
    }
  })

  // Helpers
  const resetForm = () => {
    setFormData({ name: "", slug: "", description: "" })
    setErrors({})
    setSelectedProject(null)
  }

  const openEditDialog = (project: Project) => {
    setSelectedProject(project)
    setFormData({
      name: project.name,
      slug: project.slug,
      description: project.description || ""
    })
    setEditDialogOpen(true)
  }

  const openDeleteDialog = (project: Project) => {
    setSelectedProject(project)
    setDeleteDialogOpen(true)
  }

  // Activate a project and navigate to dashboard
  const handleEnterProject = (project: Project) => {
    setActiveProjectId(project.id)
    navigate("/")
  }

  const columns: ColumnDef<Project>[] = [
    {
      accessorKey: "name",
      header: "Name",
      cell: ({ row }) => (
        <div className="flex items-center gap-3">
          <Avatar className="h-8 w-8 rounded-lg bg-primary/10 text-primary border-none">
            <AvatarFallback className="rounded-lg text-xs font-bold">
              {row.original.name.charAt(0).toUpperCase()}
            </AvatarFallback>
          </Avatar>
          <div className="flex flex-col">
            <div className="flex items-center gap-2 flex-wrap">
              {isAdmin ? (
                <span className="font-medium text-foreground cursor-pointer hover:text-primary transition-colors" onClick={() => navigate(`/projects/${row.original.id}`)}>
                  {row.original.name}
                </span>
              ) : (
                <span className="font-medium text-foreground">
                  {row.original.name}
                </span>
              )}
              {row.original.id === activeProjectId && (
                <ColorBadge color="green">
                  Active
                </ColorBadge>
              )}
            </div>
            <span className="text-xs text-muted-foreground font-mono">{row.original.slug}</span>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "owner_name",
      header: "Owner",
      cell: ({ row }) => (
        <span className="text-muted-foreground">{row.original.owner_name || "-"}</span>
      ),
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
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const isActive = row.original.id === activeProjectId
        return (
          <div className="flex items-center justify-end gap-2">
            {/* Enter button: disabled if this is already the active project */}
            {!isAdmin && <Button
              variant="outline"
              size="sm"
              onClick={() => handleEnterProject(row.original)}
              disabled={isActive}
              title={isActive ? "Already active project" : "Set as active project and go to dashboard"}
            >
              <LogIn className="mr-1 h-3.5 w-3.5" />
              Enter
            </Button>}
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => openEditDialog(row.original)}
                  />
                }
              >
                <div className="flex items-center">
                  <Pencil />
                  <span className="sr-only">Edit</span>
                </div>
              </TooltipTrigger>
              <TooltipContent>Edit project</TooltipContent>
            </Tooltip>
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => openDeleteDialog(row.original)}
                    className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  />
                }
              >
                <div className="flex items-center">
                  <Trash2 />
                  <span className="sr-only">Delete</span>
                </div>
              </TooltipTrigger>
              <TooltipContent>Delete project</TooltipContent>
            </Tooltip>
          </div>
        )
      },
    },
  ]

  // Toolbar: search on the left, view toggle + new button on the right
  const leftToolbar = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search projects..."
      value={search}
      onChange={(e) => setSearch(e.target.value)}
    />
  )

  const rightToolbar = (
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
      <Button onClick={() => { resetForm(); setCreateDialogOpen(true) }}>
        <Plus className="h-4 w-4" />
        Create Project
      </Button>
    </div>
  )

  const isEmptyProjects = !isLoading && projects.length === 0 && !search.trim()

  const renderProjectsTable = (loading: boolean) => (
    <DataTable
      columns={columns}
      data={projects}
      isLoading={loading}
      viewMode={viewMode}
      onRefresh={refetch}
      manualPagination
      totalCount={paginationInfo?.total ?? 0}
      pagination={pagination}
      onPaginationChange={setPagination}
      leftToolbar={() => leftToolbar}
      rightToolbar={() => rightToolbar}
      renderCard={(project) => {
        const isActive = project.id === activeProjectId
        return (
          <Card className="group/card hover:shadow-md transition-shadow h-full">
            <CardHeader className="pb-2">
              <div className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-3 min-w-0">
                  <Avatar className="h-10 w-10 rounded-lg bg-primary/10 text-primary border-none">
                    <AvatarFallback className="rounded-lg text-lg font-bold">
                      {project.name.charAt(0).toUpperCase()}
                    </AvatarFallback>
                  </Avatar>
                  <div className="flex flex-col min-w-0">
                    <div className="flex items-center gap-2 flex-wrap">
                      {isAdmin ? (
                        <CardTitle className="text-base font-semibold truncate cursor-pointer hover:text-primary transition-colors" onClick={() => navigate(`/projects/${project.id}`)}>
                          {project.name}
                        </CardTitle>
                      ) : (
                        <CardTitle className="text-base font-semibold truncate">
                          {project.name}
                        </CardTitle>
                      )}
                      {project.id === activeProjectId && (
                        <ColorBadge color="green">
                          Active
                        </ColorBadge>
                      )}
                    </div>
                    <div className="flex items-center gap-2 text-[10px] text-muted-foreground truncate font-mono">
                      <span>{project.slug}</span>
                      <span>•</span>
                      {project.description ? (
                        <span className="truncate">{project.description}</span>
                      ) : (
                        <span className="italic">No description</span>
                      )}
                    </div>
                  </div>
                </div>
                {/* Card action buttons, visible on hover */}
                <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                  {!isAdmin && (
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => handleEnterProject(project)}
                      disabled={isActive}
                    >
                      <LogIn />
                      Enter
                    </Button>
                  )}
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          onClick={() => openEditDialog(project)}
                        />
                      }
                    >
                      <div className="flex items-center">
                        <Pencil />
                        <span className="sr-only">Edit</span>
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>Edit project</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={
                        <Button
                          variant="ghost"
                          size="icon-sm"
                          className="text-destructive hover:text-destructive hover:bg-destructive/10"
                          onClick={() => openDeleteDialog(project)}
                        />
                      }
                    >
                      <div className="flex items-center">
                        <Trash2 />
                        <span className="sr-only">Delete</span>
                      </div>
                    </TooltipTrigger>
                    <TooltipContent>Delete project</TooltipContent>
                  </Tooltip>
                </div>
              </div>
            </CardHeader>
            <CardContent className="pt-2 space-y-3">
              <div className="space-y-2">
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <UserCog className="h-3.5 w-3.5" />
                  <span className="font-mono">
                    {project.owner_name}
                  </span>
                </div>

              </div>
              <div className="flex items-center justify-between gap-2 text-[10px] text-muted-foreground/60 border-t pt-2">
                <div className="flex items-center gap-1.5">
                  <Clock className="h-3 w-3" />
                  <span>Created at {formatDate(project.created_at)}</span>
                </div>
              </div>
            </CardContent>
          </Card>
        )
      }}
    />
  )

  return (
    <div className="flex flex-col flex-1  gap-6">
      <PageHeader items={[{ label: "Projects", icon: GalleryVerticalEnd }]} />

      <div>
        <h1 className="text-2xl font-bold">Projects</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Manage your projects and resources.
        </p>
      </div>

      {isLoading && projects.length === 0 ? (
        <div className="flex flex-col flex-1 items-center justify-center min-h-100">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      ) : isEmptyProjects ? (
        <EmptyState
          title="No projects yet"
          description="Create your first project to organize environments and applications."
          icon={GalleryVerticalEnd}
          actionText="Create Project"
          onAction={() => { resetForm(); setCreateDialogOpen(true) }}
          actionIcon={Plus}
        />
      ) : renderProjectsTable(false)}

      <CreateProjectDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />

      <EditProjectDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        project={selectedProject}
        onSuccess={() => {
          setSelectedProject(null)
        }}
      />

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Are you sure?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete the project "{selectedProject?.name}" and all its environments and applications.
              This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel onClick={() => setSelectedProject(null)}>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => selectedProject && deleteMutation.mutate(selectedProject.id)}
              disabled={deleteMutation.isPending}
              variant="destructive"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
