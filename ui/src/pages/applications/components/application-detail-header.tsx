import { type App } from "@/api/apps"
import { AppActionButtons } from "@/components/applications/app-action-buttons"
import { ColorBadge } from "@/components/shared/color-badge"
import { Button } from "@/components/ui/button"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { getAppStatusColor } from "@/lib/app-status"
import { Box, Pencil, Star } from "lucide-react"

interface ApplicationDetailHeaderProps {
  app: App
  isViewer: boolean
  isFavorite: boolean
  onToggleFavorite: () => void
  onEdit: () => void
  onDeleteSuccess: () => void
}

export function ApplicationDetailHeader({
  app,
  isViewer,
  isFavorite,
  onToggleFavorite,
  onEdit,
  onDeleteSuccess,
}: ApplicationDetailHeaderProps) {
  return (
    <div className="flex flex-col gap-4">
      <div className="flex justify-between items-start">
        <div className="flex items-center gap-4">
          <div className="p-3 bg-primary/10 rounded-lg text-primary">
            <Box className="h-8 w-8" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h1 className="text-2xl font-bold tracking-tight">{app.name}</h1>
              <ColorBadge color={getAppStatusColor(app.status)}>
                {app.status.toUpperCase()}
              </ColorBadge>
            </div>
            <div className="flex items-center gap-2 text-sm text-muted-foreground">
              <span className="font-mono">{app.slug}</span>
              <span>•</span>
              {app.description ? (
                <span className="truncate">{app.description}</span>
              ) : (
                <span className="italic">No description</span>
              )}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {!isViewer && (
            <Tooltip>
              <TooltipTrigger
                delay={200}
                render={(
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={onToggleFavorite}
                  />
                )}
              >
                <Star className={`h-4 w-4 ${isFavorite ? "fill-yellow-400 text-yellow-400" : "text-muted-foreground"}`} />
                <span className="sr-only">
                  {isFavorite ? "Remove from favorites" : "Add to favorites"}
                </span>
              </TooltipTrigger>
              <TooltipContent>
                {isFavorite ? "Remove from favorites" : "Add to favorites"}
              </TooltipContent>
            </Tooltip>
          )}
          {!isViewer && (
            <Button
              variant="outline"
              size="icon"
              onClick={onEdit}
            >
              <Pencil />
            </Button>
          )}
          {!isViewer && app.available_actions.length > 0 && (
            <AppActionButtons
              appId={app.id}
              actions={app.available_actions}
              onDeleteSuccess={onDeleteSuccess}
            />
          )}
        </div>
      </div>
    </div>
  )
}
