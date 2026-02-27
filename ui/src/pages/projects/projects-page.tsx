import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef, type PaginationState } from "@tanstack/react-table"
import { FolderKanban, GalleryVerticalEnd, LayoutGrid, List as ListIcon, LogIn, Pencil, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { projectsApi, type Project } from "@/api/projects"
import { DataTable } from "@/components/data-table/data-table"
import { PageHeader } from "@/components/layout/page-header"
import CreateProjectDialog from "@/components/project/create-project-dialog"
import EditProjectDialog from "@/components/project/edit-project-dialog"
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
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useDebounce } from "@/hooks/use-debounce"
import { useProjectStore } from "@/stores/project"

// LocalStorage key for persisting view mode preference
const PROJECTS_VIEW_MODE_KEY = "projects_view_mode"

export function ProjectsPage() {
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { activeProjectId, setActiveProjectId } = useProjectStore()

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

  // Form states
  const [_formData, setFormData] = React.useState({
    name: "",
    slug: "",
    description: ""
  })
  const [_errors, setErrors] = React.useState<Record<string, string>>({})

  // Fetch projects with server-side pagination and search
  const { data: projectsResponse, refetch } = useQuery({
    queryKey: ['projects', debouncedSearch, pagination.pageIndex, pagination.pageSize],
    queryFn: () => projectsApi.list({
      search: debouncedSearch,
      page: pagination.pageIndex + 1,
      page_size: pagination.pageSize,
    }),
    placeholderData: (prev) => prev,
  })

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
          <div className="flex size-8 items-center justify-center rounded-lg border bg-sidebar-primary text-sidebar-primary-foreground shrink-0">
            <GalleryVerticalEnd className="size-4" />
          </div>
          <div className="flex flex-col min-w-0">
            <span className="font-medium text-foreground truncate">{row.original.name}</span>
            <span className="text-xs text-muted-foreground font-mono truncate">{row.original.slug}</span>
          </div>
        </div>
      ),
    },
    {
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="text-muted-foreground line-clamp-1">
          {row.original.description || "-"}
        </span>
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
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => {
        const isActive = row.original.id === activeProjectId
        return (
          <div className="flex items-center justify-end gap-2">
            {/* Enter button: disabled if this is already the active project */}
            <Button
              variant="outline"
              size="sm"
              onClick={() => handleEnterProject(row.original)}
              disabled={isActive}
              title={isActive ? "Already active project" : "Set as active project and go to dashboard"}
            >
              <LogIn className="mr-1 h-3.5 w-3.5" />
              Enter
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => openEditDialog(row.original)}
              className="text-muted-foreground hover:text-foreground"
            >
              <Pencil />
              <span className="sr-only">Edit</span>
            </Button>
            <Button
              variant="ghost"
              size="icon-sm"
              onClick={() => openDeleteDialog(row.original)}
              className="text-destructive hover:text-destructive hover:bg-destructive/10"
            >
              <Trash2 />
              <span className="sr-only">Delete</span>
            </Button>
          </div>
        )
      },
    },
  ]

  // Toolbar: search on the left, view toggle + new button on the right
  const toolbarLeft = (
    <Input
      className="flex flex-1 max-w-sm min-w-75"
      placeholder="Search projects..."
      value={search}
      onChange={(e) => setSearch(e.target.value)}
    />
  )

  const toolbarRight = (
    <div className="flex items-center gap-2">
      <Tabs
        value={viewMode}
        onValueChange={(v) => setViewMode(v as "list" | "card")}
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

  return (
    <div className="flex flex-col gap-6">
      <PageHeader items={[{ label: "Projects", icon: FolderKanban }]} />

      <div>
        <h1 className="text-2xl font-bold">Projects</h1>
        <p className="text-sm text-muted-foreground mt-1">
          Manage your projects and resources.
        </p>
      </div>

      <DataTable
        columns={columns}
        data={projects}
        viewMode={viewMode}
        onRefresh={refetch}
        manualPagination
        totalCount={paginationInfo?.total ?? 0}
        pagination={pagination}
        onPaginationChange={setPagination}
        leftActions={() => toolbarLeft}
        toolbarActions={() => toolbarRight}
        renderCard={(project) => {
          const isActive = project.id === activeProjectId
          return (
            <Card className="group/card hover:shadow-md transition-shadow h-full">
              <CardHeader className="pb-2">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex items-start gap-3 min-w-0">
                    <div className="flex size-10 items-center justify-center rounded-lg border bg-sidebar-primary text-sidebar-primary-foreground shrink-0">
                      <GalleryVerticalEnd className="size-5" />
                    </div>
                    <div className="flex flex-col min-w-0">
                      <CardTitle className="text-base font-semibold truncate">{project.name}</CardTitle>
                      <span className="text-xs text-muted-foreground font-mono truncate">{project.slug}</span>
                    </div>
                  </div>
                  {/* Card action buttons, visible on hover */}
                  <div className="flex items-center gap-1 shrink-0 opacity-0 group-hover/card:opacity-100 transition-opacity" onClick={(e) => e.stopPropagation()}>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="h-6 w-6"
                      onClick={() => openEditDialog(project)}
                    >
                      <Pencil />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon-sm"
                      className="h-6 w-6 text-destructive hover:text-destructive hover:bg-destructive/10"
                      onClick={() => openDeleteDialog(project)}
                    >
                      <Trash2 />
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="pt-2 space-y-3">
                {project.description && (
                  <p className="text-xs text-muted-foreground line-clamp-2">{project.description}</p>
                )}
                {project.owner_name && (
                  <p className="text-xs text-muted-foreground">Owner: {project.owner_name}</p>
                )}
                {/* Enter button at the bottom of the card */}
                <Button
                  variant={isActive ? "secondary" : "outline"}
                  size="sm"
                  className="w-full"
                  onClick={() => handleEnterProject(project)}
                  disabled={isActive}
                >
                  <LogIn className="mr-1 h-3.5 w-3.5" />
                  {isActive ? "Active" : "Enter"}
                </Button>
              </CardContent>
            </Card>
          )
        }}
      />

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
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {deleteMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
