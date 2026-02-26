import {
  Blocks,
  Box,
  FolderGit2,
  LayoutDashboard,
  ShipWheel,
  Trash2,
  User,
  Warehouse
} from "lucide-react"
import * as React from "react"

import { GlobalSearchDialog } from "@/components/global-search/global-search"
import { NavGlobalSearch } from "@/components/sidebar/nav-global-search"
import { NavMain } from "@/components/sidebar/nav-main"
import { NavOthers } from "@/components/sidebar/nav-others"
import { NavUser } from "@/components/sidebar/nav-user"
import { ProjectSwitcher } from "@/components/sidebar/project-switcher"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from "@/components/ui/sidebar"
import { useAuthStore } from "@/stores/auth"

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const [searchOpen, setSearchOpen] = React.useState(false)
  const userRole = useAuthStore((state) => state.user?.role)

  const navMain = [
    {
      title: "Dashboard",
      url: "/",
      icon: LayoutDashboard,
    },
    {
      title: "Applications",
      url: "/applications",
      icon: Box,
    },
    {
      title: "Code Repositories",
      url: "/code-repositories",
      icon: FolderGit2,
    },
    {
      title: "Container Registries",
      url: "/container-registries",
      icon: Warehouse,
    },
    {
      title: "Clusters",
      url: "/clusters",
      icon: ShipWheel,
    },
    {
      title: "Recycle Bin",
      url: "/recycle-bin",
      icon: Trash2,
    },
  ]

  if (userRole === 'admin') {
    navMain.push({
      title: "Extensions",
      url: "/extensions",
      icon: Blocks,
    })
    navMain.push({
      title: "Users",
      url: "/users",
      icon: User,
    })
  }

  const userData = {
    name: "ketches",
    email: "ketches@ketches.cn",
    avatar: "/avatars/shadcn.jpg",
  }

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <ProjectSwitcher />
      </SidebarHeader>
      <SidebarContent>
        <NavGlobalSearch onOpenSearch={() => setSearchOpen(true)} />
        <NavMain items={navMain} />
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
