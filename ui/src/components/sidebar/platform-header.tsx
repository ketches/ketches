import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from '@/components/ui/sidebar'
import { usePlatformBranding } from '@/hooks/use-platform-settings'
import { useVersion } from '@/hooks/useVersion'

export function PlatformHeader() {
  const version = useVersion()
  const { data: branding, isLoading } = usePlatformBranding()
  const brandName = branding?.name || (isLoading ? '' : 'Ketches Admin')

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <SidebarMenuButton size="lg" className="cursor-default hover:bg-transparent active:bg-transparent">
          <div className="flex aspect-square size-8 items-center justify-center rounded-lg">
            {!isLoading ? (
              <img src="/ketches.svg" className="size-8 object-cover rounded-lg" alt={brandName || 'Platform'} />
            ) : (
              <div className="size-8 rounded-lg bg-sidebar-accent animate-pulse" />
            )}
          </div>
          <div className="grid flex-1 text-left text-sm leading-tight">
            <span className="truncate font-medium">{brandName}</span>
            <span className="truncate font-mono text-xs text-muted-foreground">{version ?? 'loading...'}</span>
          </div>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
