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

interface AppActionButtonsProps {
  appId: string
  actions: ActionMetadata[]
  onDeleteSuccess?: () => void
}

export function AppActionButtons({ appId, actions, onDeleteSuccess }: AppActionButtonsProps) {
  const queryClient = useQueryClient()
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)

  const executeMutation = useMutation({
    mutationFn: async (action: string) => {
      return await appsApi.executeAction(appId, action)
    },
    onSuccess: (_, action) => {
      queryClient.invalidateQueries({ queryKey: ['app', appId] })
      queryClient.invalidateQueries({ queryKey: ['app-pods', appId] })

      if (action === "delete") {
        queryClient.invalidateQueries({ queryKey: ['apps'] })
        toast.success("Application deleted", {
          description: "The application has been successfully deleted",
        })
        onDeleteSuccess?.()
      } else {
        toast.success("Action executed", {
          description: `Action "${action}" executed successfully`,
        })
      }
    },
    onError: (error: any, action) => {
      toast.error("Action failed", {
        description: error.response?.data?.error || `Failed to execute "${action}"`,
      })
    },
  })

  const handleAction = (action: string) => {
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

  const primaryActions = actions.filter(a => a.category === "primary")
  const secondaryActions = actions.filter(a => a.category === "secondary")

  return (
    <>
      <div className="flex items-center gap-2">
        {primaryActions.map((action) => {
          const Icon = iconMap[action.icon] || RefreshCw
          const isLoading = executeMutation.isPending && executeMutation.variables === action.action
          const isDestructive = action.variant === "destructive"

          return (
            <Button
              key={action.action}
              variant={isDestructive ? "outline" : action.variant as any}
              onClick={() => handleAction(action.action)}
              disabled={executeMutation.isPending}
              className={isDestructive ? "text-destructive hover:text-destructive hover:bg-destructive/10" : ""}
            >
              {isLoading ? (
                <Loader2 className="animate-spin" />
              ) : (
                <Icon />
              )}
              {action.label}
            </Button>
          )
        })}

        {secondaryActions.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger>
              <Button variant="outline">
                <MoreVertical />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              {secondaryActions.map((action) => {
                const Icon = iconMap[action.icon] || RefreshCw
                const isLoading = executeMutation.isPending && executeMutation.variables === action.action
                const isDestructive = action.variant === "destructive"

                return (
                  <DropdownMenuItem
                    key={action.action}
                    onClick={() => handleAction(action.action)}
                    disabled={executeMutation.isPending}
                    variant={isDestructive ? "destructive" : "default"}
                  >
                    {isLoading ? (
                      <Loader2 className="animate-spin" />
                    ) : (
                      <Icon />
                    )}
                    {action.label}
                  </DropdownMenuItem>
                )
              })}
            </DropdownMenuContent>
          </DropdownMenu>
        )}
      </div >

      <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Delete Application?</AlertDialogTitle>
            <AlertDialogDescription>
              This action cannot be undone. This will permanently delete the application and all its resources from the cluster.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              variant="destructive"
            >
              {executeMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
