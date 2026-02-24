import { useQuery } from "@tanstack/react-query"
import { Box, ExternalLink } from "lucide-react"
import { useNavigate } from "react-router-dom"

import { pluginsApi } from "@/api/plugins"
import { ColorBadge } from "@/components/shared/color-badge"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { EmptyState } from "../shared/empty-state"

interface InstalledAppsDialogProps {
  plugin: any
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function InstalledAppsDialog({ plugin, open, onOpenChange }: InstalledAppsDialogProps) {
  const navigate = useNavigate()

  const { data: apps = [], isLoading } = useQuery({
    queryKey: ['plugin-installed-apps', plugin?.id],
    queryFn: () => pluginsApi.getPluginInstalledApps(plugin.id),
    enabled: !!plugin && open
  })

  const activeApps = apps.filter((app: any) => !app.deleted_at)

  const handleNavigateToApp = (appId: string) => {
    onOpenChange(false)
    navigate(`/applications/${appId}`)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>Applications Using This Plugin</DialogTitle>
          <DialogDescription>
            {plugin?.name} is installed in the following applications
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 max-h-96 overflow-y-auto">
          {isLoading ? (
            <div className="flex items-center justify-center py-8">
              <span className="text-sm text-muted-foreground animate-pulse">Loading applications...</span>
            </div>
          ) : activeApps.length === 0 ? (
            <EmptyState title="No Applications Found" description="This plugin is not currently installed in any applications." icon={Box} />
          ) : (
            activeApps.map((app: any) => (
              <div
                key={app.id}
                className="flex items-center justify-between p-3 rounded-lg border hover:bg-muted/50 transition-colors group"
              >
                <div className="flex flex-col gap-1 min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="font-medium truncate">{app.name}</span>
                    <ColorBadge
                      color={app.status === 'running' ? 'green' : app.status === 'undeployed' ? 'gray' : 'yellow'}
                      className="text-[10px]"
                    >
                      {app.status?.toUpperCase() || 'UNKNOWN'}
                    </ColorBadge>
                  </div>
                  <span className="text-xs text-muted-foreground font-mono truncate">{app.slug}</span>
                </div>
                <Button
                  variant="ghost"
                  onClick={() => handleNavigateToApp(app.id)}
                  className="shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
                >
                  <ExternalLink />
                  View
                </Button>
              </div>
            ))
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
