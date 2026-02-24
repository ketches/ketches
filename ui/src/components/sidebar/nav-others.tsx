import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
} from "@/components/ui/sidebar"
import { Orbit, Puzzle, Users } from "lucide-react"
import { NavMenuItem } from "./nav-main"

export function NavOthers() {
  return (
    <SidebarGroup>
      <SidebarGroupLabel className="group-data-[collapsible=icon]:hidden">Others</SidebarGroupLabel>
      <SidebarMenu>
        <NavMenuItem item={{ title: "Environments", url: "/environments", icon: Orbit }} />
        <NavMenuItem item={{ title: "Plugins", url: "/plugins", icon: Puzzle }} />
        <NavMenuItem item={{ title: "Members", url: "/members", icon: Users }} />
      </SidebarMenu>
    </SidebarGroup>
  )
}

