import { useMutation, useQueryClient } from "@tanstack/react-query"
import {
  ArrowBigUpDash,
  Bug,
  BugOff,
  Loader2,
  MoreVertical,
  Pause,
  Play,
  RefreshCw,
  Rocket,
  RotateCw,
  Trash2,
  Undo
} from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { appsApi, type ActionMetadata } from "@/api/apps"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"

const iconMap: Record<string, React.ComponentType<{ className?: string }>> = {
  "rocket": Rocket,
  "play": Play,
  "pause": Pause,
  "update": ArrowBigUpDash,
  "cloud-sync": RefreshCw,
  "cloud-backup": Undo,
  "rotate-cw": RotateCw,
  "bug": Bug,
  "bug-off": BugOff,
  "trash-2": Trash2,
}

interface AppActionIconsProps {
  appId: string
  actions: ActionMetadata[]
}

export function AppActionIcons({ appId, actions }: AppActionIconsProps) {
  const queryClient = useQueryClient()
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)

  const executeMutation = useMutation({
    mutationFn: async (action: string) => {
      return await appsApi.executeAction(appId, action)
    },
    onSuccess: (_, action) => {
      queryClient.invalidateQueries({ queryKey: ['apps'] })
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      toast.success("Success", {
        description: `${action} executed`,
      })
    },
    onError: (error: any, action) => {
      toast.error("Failed", {
        description: error.response?.data?.error || `Failed to execute ${action}`,
      })
    },
  })

  const handleAction = (e: React.MouseEvent, action: string) => {
    e.stopPropagation()
    if (action === "delete") {
      setDeleteDialogOpen(true)
      return
    }
    executeMutation.mutate(action)
  }

  const handleDelete = () => {
    executeMutation.mutate("delete")
    setDeleteDialogOpen(false)
  }

  const primaryActions = actions.filter(a => a.category === "primary").slice(0, 2)
  const moreActions = [
    ...actions.filter(a => a.category === "primary").slice(2),
    ...actions.filter(a => a.category === "secondary")
  ]

  return (
    <TooltipProvider>
      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
        {primaryActions.map((action) => {
          const Icon = iconMap[action.icon] || RefreshCw
          const isLoading = executeMutation.isPending && executeMutation.variables === action.action

          return (
            <Tooltip key={action.action}>
              <TooltipTrigger>
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={(e) => handleAction(e, action.action)}
                  disabled={executeMutation.isPending}
                >
                  {isLoading ? (
                    <Loader2 className="animate-spin" />
                  ) : (
                    <Icon />
                  )}
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>{action.label}</p>
              </TooltipContent>
            </Tooltip>
          )
        })}

        {moreActions.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={(e) => e.stopPropagation()}
              >
                <MoreVertical />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()}>
              {moreActions.map((action) => {
                const Icon = iconMap[action.icon] || RefreshCw
                const isLoading = executeMutation.isPending && executeMutation.variables === action.action
                const isDestructive = action.variant === "destructive"

                return (
                  <DropdownMenuItem
                    key={action.action}
                    onClick={(e) => handleAction(e, action.action)}
                    disabled={executeMutation.isPending}
                    className={isDestructive ? "text-destructive" : ""}
                  >
                    {isLoading ? (
                      <Loader2 className="animate-spin mr-2" />
                    ) : (
                      <Icon className="mr-2" />
                    )}
                    {action.label}
                  </DropdownMenuItem>
                )
              })}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div>

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent onClick={(e) => e.stopPropagation()}>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Application?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the application.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {executeMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </TooltipProvider>
  )
}
