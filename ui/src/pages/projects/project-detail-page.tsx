import * as React from "react"
import { useParams } from "react-router-dom"
import { useQuery } from "@tanstack/react-query"
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
} from "lucide-react"

import { projectsApi } from "@/api/projects"
import { PageHeader } from "@/components/layout/page-header"
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
  const [activeTab, setActiveTab] = React.useState("overview")

  const { data: project, isLoading } = useQuery({
    queryKey: ["project", projectId],
    queryFn: () => projectsApi.get(projectId!),
    enabled: !!projectId,
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

      <div>
        <h1 className="text-2xl font-bold">{project?.name}</h1>
        <p className="text-sm text-muted-foreground mt-1">
          {project?.description || "No description"}
        </p>
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
    </div>
  )
}

export default ProjectDetailPage
