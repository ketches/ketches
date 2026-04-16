import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { isAxiosError, type AxiosError } from "axios"
import {
  Box,
  Brain,
  ChevronsUpDown,
  CircleAlert,
  Clock,
  FolderGit2,
  GalleryVerticalEnd,
  Info,
  LayoutDashboard,
  Loader2,
  Orbit,
  Pencil,
  Puzzle,
  Share2,
  Telescope,
  Trash2,
  Users,
  Warehouse
} from "lucide-react"
import * as React from "react"
import { useNavigate, useParams } from "react-router-dom"
import { toast } from "sonner"

import { ProjectRoleLabels, projectsApi } from "@/api/projects"
import { NotFoundPage } from "@/components/layout/not-found-page"
import { PageHeader } from "@/components/layout/page-header"
import { EditProjectDialog } from "@/components/project/edit-project-dialog"
import { ProjectAiProvidersPanel } from "@/components/project/project-ai-providers-panel"
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
import { Card, CardAction, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Switch } from "@/components/ui/switch"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { useProjectRole } from "@/hooks/useProjectRole"
import { formatDate } from "@/lib/utils"
import { ApplicationsPage } from "@/pages/applications/applications-page"
import { CodeRepositoriesPage } from "@/pages/code-repositories/code-repositories-page"
import { CollaborationsPage } from "@/pages/collaborations/collaborations-page"
import { ContainerRegistriesPage } from "@/pages/container-registries/container-registries-page"
import { UserDashboard } from "@/pages/dashboard/dashboard-page"
import { EnvironmentsPage } from "@/pages/environments/environments-page"
import { MembersPage } from "@/pages/members/members-page"
import { PluginsPage } from "@/pages/plugins/plugins-page"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"

interface ProjectDetailPageProps {
  initialTab?: string
}
export function ProjectDetailPage({ initialTab = "overview" }: ProjectDetailPageProps) {
  const { projectId } = useParams<{ projectId: string }>()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [activeTab, setActiveTab] = React.useState<string>(initialTab)
  const authUser = useAuthStore((state) => state.user)
  const isAdmin = authUser?.role === "admin"
  const { activeProjectId, setActiveContextWithNames } = useProjectStore()
  const projectRole = useProjectRole(projectId)
  const isViewer = projectRole === "viewer"
  const canManageProject = isAdmin || projectRole === "owner"
  const adminTabs = ["overview", "dashboard", "environments", "applications", "code-repositories", "container-registries", "plugins", "members", "collaboration", "ai-providers"]
  const memberTabs = ["overview", "container-registries", "plugins", "members", "ai-providers"]
  const viewerTabs = ["overview", "members"]
  const allowedTabs = isAdmin ? adminTabs : isViewer ? viewerTabs : memberTabs

  React.useEffect(() => {
    setActiveTab(initialTab)
  }, [initialTab])

  React.useEffect(() => {
    if (!allowedTabs.includes(activeTab)) {
      setActiveTab("overview")
    }
  }, [activeTab, allowedTabs])
  const [editOpen, setEditOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)

  const { data: project, isLoading, error: projectError } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => projectsApi.get(projectId!),
    enabled: !!projectId,
  })
  const { data: membersResponse } = useQuery({
    queryKey: ["project-members", projectId, "owner-summary"],
    queryFn: () => projectsApi.listMembers(projectId!, { page: 1, page_size: 200 }),
    enabled: !!projectId,
  })
  const { data: projectsResponse } = useQuery({
    queryKey: ["projects", "breadcrumb-switcher"],
    queryFn: () => projectsApi.list({ page: 1, page_size: 100 }),
    enabled: !!projectId,
  })
  const ownerMember = React.useMemo(
    () => membersResponse?.items?.find((member) => member.project_role === "owner"),
    [membersResponse?.items]
  )
  const ownerDisplayName = ownerMember?.fullname || ownerMember?.username || project?.owner_name || "-"
  const safeProjects = React.useMemo(() => (
    Array.isArray(projectsResponse?.items) ? projectsResponse.items : []
  ), [projectsResponse?.items])

  React.useEffect(() => {
    if (project && activeProjectId !== project.id) {
      setActiveContextWithNames(project.id, project.name, null, null)
    }
  }, [project, activeProjectId, setActiveContextWithNames])

  const deleteMutation = useMutation<unknown, AxiosError<{ error: string }>, void>({
    mutationFn: () => projectsApi.delete(projectId!),
    onSuccess: () => {
      toast.success("Project deleted", {
        description: `"${project?.name}" has been permanently deleted.`,
      })
      queryClient.invalidateQueries({ queryKey: ["projects"] })
      navigate("/projects")
    },
    onError: (error) => {
      toast.error("Failed to delete project", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })
  const updateCollaborationMutation = useMutation({
    mutationFn: (enabled: boolean) => {
      if (!project) {
        throw new Error("Project not loaded")
      }
      return projectsApi.update(projectId!, {
        name: project.name,
        description: project.description,
        collaboration_enabled: enabled,
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["project", projectId] })
      queryClient.invalidateQueries({ queryKey: ["project", activeProjectId] })
      queryClient.invalidateQueries({ queryKey: ["projects"] })
      toast.success("Collaboration setting updated")
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to update collaboration setting", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })

  const breadcrumbs = [
    { label: "Projects", icon: GalleryVerticalEnd, href: "/projects" },
    {
      label: project?.name ?? "...",
      icon: GalleryVerticalEnd,
      dropdown: safeProjects.length > 1 ? (
        <DropdownMenu>
          <DropdownMenuTrigger render={<Button variant="ghost" size="icon-sm"><ChevronsUpDown /></Button>} />
          <DropdownMenuContent align="start" className="w-fit">
            <DropdownMenuGroup>
              {safeProjects.map((projectOption) => (
                <DropdownMenuItem
                  key={projectOption.id}
                  onClick={() => {
                    setActiveContextWithNames(projectOption.id, projectOption.name, null, null)
                    navigate(`/projects/${projectOption.id}`)
                  }}
                >
                  <GalleryVerticalEnd className="h-4 w-4" />
                  {projectOption.name}
                </DropdownMenuItem>
              ))}
            </DropdownMenuGroup>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : undefined,
    },
  ]

  if (isLoading) {
    return (
      <div className="flex flex-col flex-1 items-center justify-center">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (!project) {
    if (isAxiosError(projectError) && projectError.response?.status === 403) {
      return (
        <EmptyState
          title="No permission"
          description="You do not have permission to view this project."
          icon={CircleAlert}
        />
      )
    }

    return (
      <NotFoundPage
        resourceType="Project"
        backHref="/projects"
        backLabel="Back to Projects"
      />
    )
  }

  return (
    <div className="flex flex-col flex-1 gap-6">
      <PageHeader items={breadcrumbs} />

      <div className="flex flex-col gap-4">
        <div className="flex justify-between items-start">
          <div className="flex items-center gap-4">
            <div className="p-3 bg-primary/10 rounded-lg text-primary">
              <GalleryVerticalEnd className="h-8 w-8" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold tracking-tight">{project?.name}</h1>
                <ColorBadge color={project.collaboration_enabled ? "blue" : "gray"}>
                  {project.collaboration_enabled ? "Collaboration On" : "Collaboration Off"}
                </ColorBadge>
              </div>
              <p className="text-sm text-muted-foreground mt-1">
                {project?.description || "No description"}
              </p>
            </div>
          </div>
          {canManageProject ? (
            <div className="flex items-center gap-2">
              <Button variant="outline" size="icon" onClick={() => setEditOpen(true)}>
                <Pencil />
              </Button>
              <Button variant="outline" onClick={() => setDeleteOpen(true)} className="text-destructive hover:text-destructive hover:bg-destructive/10">
                <Trash2 />
                Delete
              </Button>
            </div>
          ) : null}
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">
            <Telescope />
            Overview
          </TabsTrigger>
          {isAdmin ? (
            <TabsTrigger value="dashboard">
              <LayoutDashboard />
              Dashboard
            </TabsTrigger>
          ) : null}
          {isAdmin ? (
            <TabsTrigger value="environments">
              <Orbit />
              Environments
            </TabsTrigger>
          ) : null}
          {isAdmin ? (
            <TabsTrigger value="applications">
              <Box />
              Applications
            </TabsTrigger>
          ) : null}
          {isAdmin ? (
            <TabsTrigger value="code-repositories">
              <FolderGit2 />
              Code Repositories
            </TabsTrigger>
          ) : null}
          {!isViewer ? (
            <TabsTrigger value="container-registries">
              <Warehouse />
              Container Registries
            </TabsTrigger>
          ) : null}
          {!isViewer ? (
            <TabsTrigger value="plugins">
              <Puzzle />
              Plugins
            </TabsTrigger>
          ) : null}
          <TabsTrigger value="members">
            <Users />
            Members
          </TabsTrigger>
          {isAdmin ? (
            <TabsTrigger value="collaboration">
              <Share2 />
              Collaborations
            </TabsTrigger>
          ) : null}
          {!isViewer ? (
            <TabsTrigger value="ai-providers">
              <Brain />
              AI Providers
            </TabsTrigger>
          ) : null}
        </TabsList>

        <TabsContent value="overview" className="space-y-4 mt-2">
          <Card className="group/card bg-linear-to-b/increasing from-blue-500/5 to-transparent data-[active=true]:bg-transparent">
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Info className="h-4 w-4" />
                Project Information
              </CardTitle>
              {canManageProject ? (
                <CardAction className="opacity-0 transition-opacity group-hover/card:opacity-100 group-focus-within/card:opacity-100">
                  <Button variant="ghost" size="icon-sm" onClick={() => setEditOpen(true)} aria-label="Edit project">
                    <Pencil />
                  </Button>
                </CardAction>
              ) : null}
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Name</p>
                  <p className="text-sm">{project.name}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Slug</p>
                  <p className="text-sm font-mono">{project.slug}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Owner</p>
                  <p className="text-sm">{ownerDisplayName}</p>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">My Role</p>
                  <div className="flex items-center">
                    <ColorBadge color={isAdmin ? "orange" : projectRole === "owner" ? "blue" : projectRole === "developer" ? "green" : "gray"}>
                      {isAdmin ? "Admin" : projectRole ? ProjectRoleLabels[projectRole] : "Unknown"}
                    </ColorBadge>
                  </div>
                </div>
                <div className="space-y-1">
                  <p className="text-xs font-medium text-muted-foreground">Created At</p>
                  <div className="flex items-center gap-1.5 text-sm">
                    <Clock className="h-3.5 w-3.5 text-muted-foreground" />
                    <span>{formatDate(project.created_at)}</span>
                  </div>
                </div>
              </div>
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle className="text-sm flex items-center gap-2">
                <Share2 className="h-4 w-4" />
                Enable Collaboration
              </CardTitle>
              <CardAction>
                <Switch
                  checked={project.collaboration_enabled}
                  onCheckedChange={(checked) => updateCollaborationMutation.mutate(checked)}
                  disabled={!canManageProject || updateCollaborationMutation.isPending}
                />
              </CardAction>
            </CardHeader>
            <CardContent>
              <div className="flex items-center">
                Enable or disable collaboration features for this project. When enabled, Collaborations menu will be displayed, allowing you to collaborate with other users on this project.
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        {isAdmin ? (
          <TabsContent value="dashboard" className="space-y-4 mt-2">
            <UserDashboard projectId={projectId} />
          </TabsContent>
        ) : null}

        {isAdmin ? (
          <TabsContent value="environments" className="space-y-4 mt-2">
            <EnvironmentsPage projectId={projectId} />
          </TabsContent>
        ) : null}

        {isAdmin ? (
          <TabsContent value="applications" className="space-y-4 mt-2">
            <ApplicationsPage projectId={projectId} />
          </TabsContent>
        ) : null}

        {isAdmin ? (
          <TabsContent value="code-repositories" className="space-y-4 mt-2">
            <CodeRepositoriesPage projectId={projectId} />
          </TabsContent>
        ) : null}

        {!isViewer ? (
          <TabsContent value="container-registries" className="space-y-4 mt-2">
            <ContainerRegistriesPage projectId={projectId} />
          </TabsContent>
        ) : null}

        {!isViewer ? (
          <TabsContent value="plugins" className="space-y-4 mt-2">
            <PluginsPage projectId={projectId} />
          </TabsContent>
        ) : null}

        <TabsContent value="members" className="space-y-4 mt-2">
          <MembersPage projectId={projectId} />
        </TabsContent>

        {isAdmin ? (
          <TabsContent value="collaboration" className="space-y-4 mt-2">
            <CollaborationsPage projectId={projectId!} />
          </TabsContent>
        ) : null}

        {!isViewer ? (
          <TabsContent value="ai-providers" className="space-y-4 mt-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-sm flex items-center gap-2">
                  <Brain className="h-4 w-4" />
                  AI Providers
                </CardTitle>
              </CardHeader>
              <CardContent>
                <ProjectAiProvidersPanel projectId={projectId!} isViewer={isViewer} />
              </CardContent>
            </Card>
          </TabsContent>
        ) : null}
      </Tabs>

      <EditProjectDialog
        open={editOpen}
        onOpenChange={setEditOpen}
        project={project ?? null}
        onSuccess={() => {
          setEditOpen(false)
          queryClient.invalidateQueries({ queryKey: ["project", projectId] })
        }}
      />

      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete project?</AlertDialogTitle>
            <AlertDialogDescription>
              This will permanently delete <strong>{project?.name}</strong> and all its associated data. This action cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => deleteMutation.mutate()}
              variant="destructive"
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
