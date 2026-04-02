import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type AxiosError } from "axios"
import {
  ArrowBigUpDash,
  Bug,
  BugOff,
  Download,
  Folder,
  FolderInput,
  FolderOutput,
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
  DropdownMenuSeparator,
  DropdownMenuSub,
  DropdownMenuSubContent,
  DropdownMenuSubTrigger,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { ExportAppsDialog } from "./export-apps-dialog"

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
  envId: string
  actions: ActionMetadata[]
  /** Groups available to move this app into */
  appGroups?: Array<{ id: string; name: string }>
  /** If set, show 'Remove from Group' instead of 'Move to Group' */
  currentGroupId?: string
  onMoveToGroup?: (groupId: string) => void
  onRemoveFromGroup?: () => void
}

export function AppActionIcons({ appId, envId, actions, appGroups, currentGroupId, onMoveToGroup, onRemoveFromGroup }: AppActionIconsProps) {
  const queryClient = useQueryClient()
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [exportDialogOpen, setExportDialogOpen] = React.useState(false)
  const [exportAppIds, setExportAppIds] = React.useState<string[]>([])
  const [_exportAppId, setExportAppId] = React.useState<string | undefined>(undefined)

  const executeMutation = useMutation<unknown, AxiosError<{ error: string }>, string>({
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
    onError: (error, action) => {
      toast.error("Failed", {
        description: error.response?.data?.error || error.message || `Failed to execute ${action}`,
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
    <>
      <div className="flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
        {primaryActions.map((action) => {
          const Icon = iconMap[action.icon] || RefreshCw
          const isLoading = executeMutation.isPending && executeMutation.variables === action.action

          return (
            <Tooltip key={action.action}>
              <TooltipTrigger
                delay={200}
                render={(
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={(e) => handleAction(e, action.action)}
                    disabled={executeMutation.isPending}
                    className={action.variant === "destructive" ? "text-destructive hover:text-destructive hover:bg-destructive/10" : ""}
                  />
                )}
              >
                {isLoading ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Icon className="h-4 w-4" />
                )}
              </TooltipTrigger>
              <TooltipContent>
                <p>{action.label}</p>
              </TooltipContent>
            </Tooltip>
          )
        })}

        {moreActions.length > 0 && (
          <DropdownMenu>
            <DropdownMenuTrigger
              render={<Button variant="ghost" size="icon-sm" onClick={(e) => e.stopPropagation()} />}
            >
              <MoreVertical />
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" onClick={(e) => e.stopPropagation()} className="w-fit">
              {moreActions.map((action) => {
                const Icon = iconMap[action.icon] || RefreshCw
                const isLoading = executeMutation.isPending && executeMutation.variables === action.action
                const isDestructive = action.variant === "destructive"

                return (
                  <DropdownMenuItem
                    key={action.action}
                    variant={isDestructive ? "destructive" : "default"}
                    onClick={(e) => handleAction(e, action.action)}
                    disabled={executeMutation.isPending}
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
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={(e) => {
                e.stopPropagation()
                setExportAppId(appId)
                setExportAppIds([])
                setExportDialogOpen(true)
              }}>
                <Download />
                Export
              </DropdownMenuItem>
              {/* Move to Group / Remove from Group below Export */}
              {currentGroupId && onRemoveFromGroup && (
                <DropdownMenuItem
                  onClick={(e) => { e.stopPropagation(); onRemoveFromGroup() }}
                >
                  <FolderOutput />
                  Remove from Group
                </DropdownMenuItem>
              )}
              {!currentGroupId && appGroups && appGroups.length > 0 && onMoveToGroup && (
                <DropdownMenuSub>
                  <DropdownMenuSubTrigger onClick={(e) => e.stopPropagation()}><FolderInput />Move to Group</DropdownMenuSubTrigger>
                  <DropdownMenuSubContent onClick={(e) => e.stopPropagation()}>
                    {appGroups.map((g) => (
                      <DropdownMenuItem key={g.id} onClick={(e) => { e.stopPropagation(); onMoveToGroup(g.id) }}>
                        <Folder />{g.name}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuSubContent>
                </DropdownMenuSub>
              )}
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
            <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
            <AlertDialogAction
              onClick={handleDelete}
              variant="destructive"
            >
              {executeMutation.isPending ? "Deleting..." : "Delete"}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>


      <ExportAppsDialog
        open={exportDialogOpen}
        onOpenChange={setExportDialogOpen}
        appIds={exportAppIds}
        appId={appId}
        envId={envId}
        onSuccess={() => {
          setExportDialogOpen(false)
        }}
      />
    </>
  )
}
