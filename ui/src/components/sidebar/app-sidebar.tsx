import {
  Activity,
  Blocks,
  Box,
  FolderGit2,
  GalleryVerticalEnd,
  LayoutDashboard,
  Orbit,
  Settings2,
  Share2,
  ShipWheel,
  Sparkles,
  Trash2,
  User
} from "lucide-react"
import * as React from "react"

import { projectsApi } from "@/api/projects"
import { GlobalSearchDialog } from "@/components/global-search/global-search"
import { NavGlobalSearch } from "@/components/sidebar/nav-global-search"
import { NavMain } from "@/components/sidebar/nav-main"
import { NavUser } from "@/components/sidebar/nav-user"
import { PlatformHeader } from "@/components/sidebar/platform-header"
import { ProjectSwitcher } from "@/components/sidebar/project-switcher"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar"
import { useProjectRole } from "@/hooks/useProjectRole"
import { useAuthStore } from "@/stores/auth"
import { useProjectStore } from "@/stores/project"
import { useQuery } from "@tanstack/react-query"

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const [searchOpen, setSearchOpen] = React.useState(false)
  const userRole = useAuthStore((state) => state.user?.role)
  const projectRole = useProjectRole()
  const activeProjectId = useProjectStore((state) => state.activeProjectId)

  const isAdmin = userRole === "admin"
  const isViewer = projectRole === 'viewer'

  const { data: activeProject } = useQuery({
    queryKey: ['project', activeProjectId],
    queryFn: () => projectsApi.get(activeProjectId!),
    enabled: !isAdmin && !!activeProjectId,
    staleTime: 5 * 60 * 1000,
  })

  const canShowCollaboration = !isAdmin && !!activeProjectId && !!activeProject?.collaboration_enabled

  const infrastructureItems = [
    { title: "Clusters", url: "/clusters", icon: ShipWheel },
    { title: "Extensions", url: "/extensions", icon: Blocks },
  ]

  const userScopeItems = [
    { title: "Projects", url: "/projects", icon: GalleryVerticalEnd },
    { title: "User Activities", url: "/activities", icon: Activity },
    { title: "Recycle Bin", url: "/recycle-bin", icon: Trash2 },

  ]

  const platformItems = [
    { title: "Users", url: "/users", icon: User },
    { title: "Settings", url: "/platform-settings", icon: Settings2 },
  ]
  // Project group: project-scoped modules (Dashboard rendered separately); some hidden for viewers
  const projectItems = isAdmin ? [] : [
    { title: "Collaborations", url: "/collaborations", icon: Share2, hidden: !canShowCollaboration },
    { title: "Applications", url: "/applications", icon: Box },
    { title: "Environments", url: "/environments", icon: Orbit, hidden: isViewer },
    { title: "Code Repositories", url: "/code-repositories", icon: FolderGit2, hidden: isViewer },
  ]

  // Global group: cross-project modules
  const globalItems = isAdmin ? [] : [
    { title: "Projects", url: "/projects", icon: GalleryVerticalEnd },
    { title: "Recycle Bin", url: "/recycle-bin", icon: Trash2, hidden: isViewer },
  ]

  const userData = {
    name: "ketches",
    email: "ketches@ketches.cn",
    avatar: "/avatars/shadcn.jpg",
  }

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        {isAdmin ? <PlatformHeader /> : <ProjectSwitcher />}
      </SidebarHeader>
      <SidebarContent>
        <NavGlobalSearch onOpenSearch={() => setSearchOpen(true)} />
        <NavMain
          dashboardItem={{ title: "Dashboard", url: "/", icon: LayoutDashboard }}
          topItems={isAdmin ? [] : [{ title: "Builder", url: "/builder-sessions", icon: Sparkles, hidden: !activeProjectId }]}
          projectItems={isAdmin ? [] : projectItems}
          globalItems={isAdmin ? [] : globalItems}
          infrastructureItems={isAdmin ? infrastructureItems : []}
          userScopeItems={isAdmin ? userScopeItems : []}
          platformItems={isAdmin ? platformItems : []}
        />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
      <SidebarRail />
      <GlobalSearchDialog open={searchOpen} onOpenChange={setSearchOpen} />
    </Sidebar>
  )
}
