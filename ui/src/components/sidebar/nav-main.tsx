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
  items,
}: {
  items: NavItem[]
}) {
  const visibleItems = items.filter((item) => !item.hidden)
  const dashboardItem = visibleItems.find((item) => item.url === "/")
  const coreItems = visibleItems.filter((item) => item.url !== "/")

  return (
    <>
      {dashboardItem && (
        <SidebarGroup>
          <SidebarMenu>
            <NavMenuItem item={dashboardItem} />
          </SidebarMenu>
        </SidebarGroup>
      )}
      <SidebarGroup>
        <SidebarGroupLabel className="group-data-[collapsible=icon]:hidden">Core</SidebarGroupLabel>
        <SidebarMenu>
          {coreItems.map((item) => (
            <NavMenuItem key={item.title} item={item} />
          ))}
        </SidebarMenu>
      </SidebarGroup>
    </>
  )
}
