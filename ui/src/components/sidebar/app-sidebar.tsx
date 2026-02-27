import {
  Activity,
  Blocks,
  Box,
  FolderGit2,
  FolderKanban,
  LayoutDashboard,
  Orbit,
  Puzzle,
  ShipWheel,
  Trash2,
  User,
  Users,
  Warehouse,
} from "lucide-react"
import * as React from "react"

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

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const [searchOpen, setSearchOpen] = React.useState(false)
  const userRole = useAuthStore((state) => state.user?.role)
  const projectRole = useProjectRole()

  const isAdmin = userRole === 'admin'
  const isViewer = projectRole === 'viewer'

  // Admin nav: platform management modules (Dashboard rendered separately)
  const adminNavItems = [
    { title: "Clusters", url: "/clusters", icon: ShipWheel },
    { title: "Extensions", url: "/extensions", icon: Blocks },
    { title: "Projects", url: "/projects", icon: FolderKanban },
    { title: "Users", url: "/users", icon: User },
  ]

  // Project group: project-scoped modules (Dashboard rendered separately); some hidden for viewers
  const projectItems = isAdmin ? [] : [
    { title: "Applications", url: "/applications", icon: Box },
    { title: "Environments", url: "/environments", icon: Orbit, hidden: isViewer },
    { title: "Code Repositories", url: "/code-repositories", icon: FolderGit2, hidden: isViewer },
    { title: "Container Registries", url: "/container-registries", icon: Warehouse, hidden: isViewer },
    { title: "Plugins", url: "/plugins", icon: Puzzle, hidden: isViewer },
    { title: "Members", url: "/members", icon: Users, hidden: isViewer },
  ]

  // Global group: cross-project modules
  const globalItems = isAdmin ? [] : [
    { title: "Projects", url: "/projects", icon: FolderKanban },
    { title: "Recycle Bin", url: "/recycle-bin", icon: Trash2, hidden: isViewer },
    { title: "Activity", url: "/activity", icon: Activity },
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
          projectItems={isAdmin ? adminNavItems : projectItems}
          projectGroupLabel={isAdmin ? "Admin" : "Project"}
          globalItems={isAdmin ? [] : globalItems}
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
