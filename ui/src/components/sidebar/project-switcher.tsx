import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { ChevronsUpDown, GalleryVerticalEnd, MoreVertical, Pencil, Plus, Trash2 } from "lucide-react"
import * as React from "react"
import { useNavigate } from "react-router-dom"
import { toast } from "sonner"

import { projectsApi, type Project } from "@/api/projects"
import { type SimpleResponse } from "@/api/pagination"
import { CreateProjectDialog } from "@/components/project/create-project-dialog"
import { EditProjectDialog } from "@/components/project/edit-project-dialog"
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar"
import { useProjectStore } from "@/stores/project"
import type { AxiosError } from "axios"

function ProjectItem({
  project,
  isActive: _isActive,
  onSelect,
  onEdit,
  onDelete,
}: {
  project: SimpleResponse
  isActive: boolean
  onSelect: () => void
  onEdit: () => void
  onDelete: () => void
}) {
  const [isHovered, setIsHovered] = React.useState(false)

  return (
    <DropdownMenuItem
      className="gap-2 p-2 justify-start"
      onClick={onSelect}
      onMouseEnter={() => setIsHovered(true)}
      onMouseLeave={() => setIsHovered(false)}
    >
      <div className="flex size-6 items-center justify-center rounded-md border">
        <GalleryVerticalEnd className="size-3.5 shrink-0" />
      </div>
      {project.name}
      <DropdownMenu>
        <DropdownMenuTrigger
          onClick={(e) => e.stopPropagation()}
          className={`${isHovered ? "opacity-100" : "opacity-0"} ml-auto transition-opacity focus:opacity-100 flex size-6 items-center justify-center rounded hover:bg-accent-foreground/10 cursor-pointer`}
        >
          <MoreVertical className="text-muted-foreground" />
        </DropdownMenuTrigger>
        <DropdownMenuContent align="start" className="w-40">
          <DropdownMenuItem onClick={(e) => { e.stopPropagation(); onEdit(); }}>
            <Pencil className="mr-2" />
            Edit
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={(e) => { e.stopPropagation(); onDelete(); }}
            className="text-destructive focus:text-destructive"
          >
            <Trash2 className="mr-2" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </DropdownMenuItem>
  )
}

export function ProjectSwitcher() {
  const { isMobile } = useSidebar()
  const queryClient = useQueryClient()
  const [createDialogOpen, setCreateDialogOpen] = React.useState(false)
  const [editDialogOpen, setEditDialogOpen] = React.useState(false)
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [selectedProject, setSelectedProject] = React.useState<SimpleResponse | null>(null)
  const { activeProjectId, setActiveProjectId } = useProjectStore()
  const navigate = useNavigate()

  const { data: projects = [] } = useQuery({
    queryKey: ['projects-simple'],
    queryFn: projectsApi.listSimple,
  })

  const safeProjects = Array.isArray(projects) ? projects : []
  const activeProject = safeProjects.find(p => p.id === activeProjectId) || safeProjects[0]

  React.useEffect(() => {
    if (safeProjects.length > 0) {
      const currentProjectExists = safeProjects.some(p => p.id === activeProjectId)
      if (!currentProjectExists) {
        setActiveProjectId(safeProjects[0].id)
      }
    }
  }, [safeProjects, activeProjectId, setActiveProjectId])

  const deleteMutation = useMutation({
    mutationFn: (projectId: string) => projectsApi.delete(projectId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['projects'] })
      toast.success("Project deleted successfully")
      setDeleteDialogOpen(false)
      setSelectedProject(null)
    },
    onError: (err: AxiosError<{ error: string }>) => {
      const errMsg = err.response?.data?.error || "Failed to delete project"
      toast.error("Error", { description: errMsg })
    }
  })

  const handleEdit = (project: SimpleResponse) => {
    setSelectedProject(project)
    setEditDialogOpen(true)
  }

  const handleDelete = (project: SimpleResponse) => {
    setSelectedProject(project)
    setDeleteDialogOpen(true)
  }

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger
            render={
              <SidebarMenuButton
                size="lg"
                className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
              >
                <div className="bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg">
                  <GalleryVerticalEnd className="size-4" />
                </div>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium">
                    {activeProject?.name || "Select Project"}
                  </span>
                  <span className="truncate text-xs font-mono">
                    {activeProject?.slug || "no-project"}
                  </span>
                </div>
                <ChevronsUpDown className="ml-auto" />
              </SidebarMenuButton>
            }
          />
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            align="start"
            side={isMobile ? "bottom" : "right"}
            sideOffset={4}
          >
            <DropdownMenuGroup>
              <DropdownMenuLabel className="text-muted-foreground text-xs">
                Projects
              </DropdownMenuLabel>
              {safeProjects.map((project) => (
                <ProjectItem
                  key={project.id}
                  project={project}
                  isActive={project.id === activeProjectId}
                  onSelect={() => { setActiveProjectId(project.id); navigate("/") }}
                  onEdit={() => handleEdit(project)}
                  onDelete={() => handleDelete(project)}
                />
              ))}
            </DropdownMenuGroup>
            <DropdownMenuSeparator />
            <DropdownMenuGroup>
              <DropdownMenuItem onClick={() => setCreateDialogOpen(true)} className="gap-2 p-2">
                <div className="flex size-6 items-center justify-center rounded-md border bg-transparent">
                  <Plus className="size-4" />
                </div>
                <div className="font-medium">Create project</div>
              </DropdownMenuItem>
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>

      <CreateProjectDialog
        open={createDialogOpen}
        onOpenChange={setCreateDialogOpen}
      />

      <EditProjectDialog
        open={editDialogOpen}
        onOpenChange={setEditDialogOpen}
        project={selectedProject as unknown as Project | null}
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
    </SidebarMenu>
  )
}

export default ProjectSwitcher
