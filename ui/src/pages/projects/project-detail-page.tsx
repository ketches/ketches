import * as React from "react"
import { useNavigate, useParams } from "react-router-dom"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import {
  GalleryVerticalEnd,
  LayoutDashboard,
  Orbit,
  Box,
  FolderGit2,
  Warehouse,
  Puzzle,
  Users,
  Loader2,
  Pencil,
  Trash2,
} from "lucide-react"
import { toast } from "sonner"

import { projectsApi } from "@/api/projects"
import { PageHeader } from "@/components/layout/page-header"
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
import { Button } from "@/components/ui/button"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { UserDashboard } from "@/pages/dashboard/dashboard-page"
import { EnvironmentsPage } from "@/pages/environments/environments-page"
import { ApplicationsPage } from "@/pages/applications/applications-page"
import { CodeRepositoriesPage } from "@/pages/code-repositories/code-repositories-page"
import { ContainerRegistriesPage } from "@/pages/container-registries/container-registries-page"
import { PluginsPage } from "@/pages/plugins/plugins-page"
import { MembersPage } from "@/pages/members/members-page"

export function ProjectDetailPage() {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = React.useState("overview")
  const [editOpen, setEditOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)

  const { data: project, isLoading } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => projectsApi.get(projectId!),
    enabled: !!projectId,
  })

  const deleteMutation = useMutation({
    mutationFn: () => projectsApi.delete(projectId!),
    onSuccess: () => {
      toast.success("Project deleted", {
        description: `"${project?.name}" has been permanently deleted.`,
      })
      queryClient.invalidateQueries({ queryKey: ["projects"] })
      navigate("/projects")
    },
    onError: (error: any) => {
      toast.error("Failed to delete project", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const breadcrumbs = [
    { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
    { label: project?.name ?? "...", icon: GalleryVerticalEnd },
  ]

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      {/* Hero section: icon, name, description, action buttons */}
      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary/10 rounded-lg text-primary">
              <GalleryVerticalEnd className="h-8 w-8" />
            </div>
            <div>
              <h1 className="text-2xl font-bold tracking-tight">{project?.name}</h1>
              <p className="text-sm text-muted-foreground mt-1">
                {project?.description || "No description"}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button variant="outline" onClick={() => setEditOpen(true)}>
              <Pencil />
              Edit
            </Button>
            <Button variant="outline" onClick={() => setDeleteOpen(true)}>
              <Trash2 />
              Delete
            </Button>
          </div>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">
            <LayoutDashboard className="h-4 w-4 mr-1.5" />
            Overview
          </TabsTrigger>
          <TabsTrigger value="environments">
            <Orbit className="h-4 w-4 mr-1.5" />
            Environments
          </TabsTrigger>
          <TabsTrigger value="applications">
            <Box className="h-4 w-4 mr-1.5" />
            Applications
          </TabsTrigger>
          <TabsTrigger value="code-repositories">
            <FolderGit2 className="h-4 w-4 mr-1.5" />
            Code Repositories
          </TabsTrigger>
          <TabsTrigger value="container-registries">
            <Warehouse className="h-4 w-4 mr-1.5" />
            Container Registries
          </TabsTrigger>
          <TabsTrigger value="plugins">
            <Puzzle className="h-4 w-4 mr-1.5" />
            Plugins
          </TabsTrigger>
          <TabsTrigger value="members">
            <Users className="h-4 w-4 mr-1.5" />
            Members
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-6">
          <UserDashboard projectId={projectId} />
        </TabsContent>

        <TabsContent value="environments" className="mt-6">
          <EnvironmentsPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="applications" className="mt-6">
          <ApplicationsPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="code-repositories" className="mt-6">
          <CodeRepositoriesPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="container-registries" className="mt-6">
          <ContainerRegistriesPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="plugins" className="mt-6">
          <PluginsPage projectId={projectId} />
        </TabsContent>

        <TabsContent value="members" className="mt-6">
          <MembersPage projectId={projectId} />
        </TabsContent>
      </Tabs>

      {/* Edit project dialog */}
      <EditProjectDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        project={project ?? null}
        onSuccess={() => {
          setEditOpen(false)
          queryClient.invalidateQueries({ queryKey: ["project", projectId] })
        }}
      />

      {/* Delete confirmation dialog */}
      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete project?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete <strong>{project?.name}</strong> and all its associated data. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteMutation.mutate()}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              Delete
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

export default ProjectDetailPage
