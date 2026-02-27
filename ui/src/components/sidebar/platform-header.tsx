import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar'
import { useVersion } from '@/hooks/useVersion'

export function PlatformHeader() {
  const version = useVersion()

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size="lg" className="cursor-default hover:bg-transparent active:bg-transparent">
          <div className="flex aspect-square size-8 items-center justify-center rounded-lg">
            <img src="/ketches.svg" className="size-8" alt="Ketches" />
          </div>
          <div className="grid flex-1 text-left text-sm leading-tight">
            <span className="truncate font-medium">Ketches Admin</span>
            <span className="truncate font-mono text-xs text-muted-foreground">{version ?? 'loading...'}</span>
          </div>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
