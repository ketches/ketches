import { Search } from "lucide-react"

import { Kbd } from "@/components/ui/kbd"
import { SidebarGroup, SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar"

interface NavGlobalSearchProps {
  onOpenSearch: () => void
}

export function NavGlobalSearch({ onOpenSearch }: NavGlobalSearchProps) {
  const { state } = useSidebar()

  return (
    <SidebarGroup>
      <SidebarMenu>
        <SidebarMenuItem key="global-search">
          <SidebarMenuButton onClick={onOpenSearch}>
            <Search />
            <span>Search</span>
            {state !== "collapsed" && (<Kbd className="absolute right-1.5 rounded">⌘ K</Kbd>)}
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    </SidebarGroup>
  )
}

export default NavGlobalSearch
