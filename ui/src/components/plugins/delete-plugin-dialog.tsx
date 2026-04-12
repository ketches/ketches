import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { pluginsApi, type Plugin } from "@/api/plugins"
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
import type { AxiosError } from "axios"

interface DeletePluginDialogProps {
  plugin: Plugin
  projectId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DeletePluginDialog({ plugin, projectId, open, onOpenChange }: DeletePluginDialogProps) {
  const queryClient = useQueryClient()
  const isInstalled = plugin.install_count > 0

  const deleteMutation = useMutation({
    mutationFn: () => pluginsApi.deletePlugin(projectId, plugin.id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['plugins', projectId] })
      toast.success("Plugin deleted successfully")
      onOpenChange(false)
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to delete plugin", {
        description: err.response?.data?.error || "An unknown error occurred"
      })
    }
  })

  const handleDelete = () => {
    if (isInstalled) {
      return
    }
    deleteMutation.mutate()
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{isInstalled ? "Plugin In Use" : "Delete Plugin"}</AlertDialogTitle>
          <AlertDialogDescription>
            {isInstalled ? (
              <>
                <strong>{plugin.name}</strong> is currently installed in {plugin.install_count} app{plugin.install_count === 1 ? "" : "s"}.
                Uninstall it from those apps before deleting this plugin.
              </>
            ) : (
              <>
                Are you sure you want to delete <strong>{plugin.name}</strong>? This action cannot be undone.
              </>
            )}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel variant="secondary">Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            disabled={deleteMutation.isPending || isInstalled}
            variant="destructive"
          >
            {isInstalled ? "Uninstall First" : deleteMutation.isPending ? "Deleting..." : "Delete Plugin"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
