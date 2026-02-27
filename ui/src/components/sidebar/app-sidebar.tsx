import {
Blocks,
Box,
  FolderGit2,
  FolderKanban,
LayoutDashboard,
ShipWheel,
Trash2,
User,
  Warehouse,
} from "lucide-react"
import * as React from "react"

import { GlobalSearchDialog } from "@/components/global-search/global-search"
import { NavGlobalSearch } from "@/components/sidebar/nav-global-search"
import { NavMain } from "@/components/sidebar/nav-main"
import { NavOthers } from "@/components/sidebar/nav-others"
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
import { useAuthStore } from "@/stores/auth"
import { useProjectRole } from "@/hooks/useProjectRole"

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const [searchOpen, setSearchOpen] = React.useState(false)
  const userRole = useAuthStore((state) => state.user?.role)
  const projectRole = useProjectRole()

  const isAdmin = userRole === 'admin'
  const isViewer = projectRole === 'viewer'

  // Admin nav: global platform management modules only
  const adminNavItems = [
    { title: "Dashboard", url: "/", icon: LayoutDashboard },
    { title: "Projects", url: "/projects", icon: FolderKanban },
    { title: "Clusters", url: "/clusters", icon: ShipWheel },
    { title: "Extensions", url: "/extensions", icon: Blocks },
    { title: "Users", url: "/users", icon: User },
  ]

  // User nav: project-scoped modules; some hidden for viewers
  const userNavItems = [
    { title: "Dashboard", url: "/", icon: LayoutDashboard },
    { title: "Applications", url: "/applications", icon: Box },
    { title: "Code Repositories", url: "/code-repositories", icon: FolderGit2, hidden: isViewer },
    { title: "Container Registries", url: "/container-registries", icon: Warehouse, hidden: isViewer },
    { title: "Recycle Bin", url: "/recycle-bin", icon: Trash2, hidden: isViewer },
  ]

  const navItems = isAdmin ? adminNavItems : userNavItems

  const userData = {
    name: "ketches",
    email: "ketches@ketches.cn",
    avatar: "/avatars/shadcn.jpg",
  }

  return (
    <Sidebar collapsible="icon" {...props}>
<SidebarHeader>
{isAdmin ? <PlatformHeader /> :
<ProjectSwitcher />}
</SidebarHeader>
      <SidebarContent>
        <NavGlobalSearch onOpenSearch={() => setSearchOpen(true)} />
        <NavMain items={navItems} />
        <NavOthers />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={userData} />
      </SidebarFooter>
      <SidebarRail />
      <GlobalSearchDialog open={searchOpen} onOpenChange={setSearchOpen} />
    </Sidebar>
  )
}
