import { type LucideIcon } from "lucide-react"
import { Link, useLocation } from "react-router-dom"

import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

interface NavItem {
  title: string
  url: string
  icon: LucideIcon
  hidden?: boolean
}


export function NavMenuItem({ item }: { item: NavItem }) {
  const location = useLocation()
  const isActive = item.url === "/"
    ? location.pathname === "/"
    : location.pathname.startsWith(item.url)

  return (
    <SidebarMenuItem>
      <SidebarMenuButton
        render={<Link to={item.url} />}
        isActive={isActive}
        className={isActive ? "bg-linear-to-r/increasing from-primary/25 to-transparent text-primary font-medium data-[active=true]:bg-transparent" : ""}
      >
        <item.icon />
        <span>{item.title}</span>
      </SidebarMenuButton>
    </SidebarMenuItem>
  )
}

export function NavMain({
  dashboardItem,
  projectItems,
  projectGroupLabel = "Project",
  globalItems,
}: {
  dashboardItem?: NavItem
  projectItems: NavItem[]
  projectGroupLabel?: string
  globalItems: NavItem[]
}) {
  const visibleProjectItems = projectItems.filter((item) => !item.hidden)
  const visibleGlobalItems = globalItems.filter((item) => !item.hidden)

  return (
    <>
      {/* Dashboard rendered outside any group */}
      {dashboardItem && (
        <SidebarGroup>
          <SidebarMenu>
            <NavMenuItem item={dashboardItem} />
          </SidebarMenu>
        </SidebarGroup>
      )}
      {visibleProjectItems.length > 0 && (
        <SidebarGroup>
          <SidebarGroupLabel className="group-data-[collapsible=icon]:hidden">{projectGroupLabel}</SidebarGroupLabel>
          <SidebarMenu>
            {visibleProjectItems.map((item) => (
              <NavMenuItem key={item.title} item={item} />
            ))}
          </SidebarMenu>
        </SidebarGroup>
      )}
      {visibleGlobalItems.length > 0 && (
        <SidebarGroup>
          <SidebarGroupLabel className="group-data-[collapsible=icon]:hidden">Global</SidebarGroupLabel>
          <SidebarMenu>
            {visibleGlobalItems.map((item) => (
              <NavMenuItem key={item.title} item={item} />
            ))}
          </SidebarMenu>
        </SidebarGroup>
      )}
    </>
  )
}
