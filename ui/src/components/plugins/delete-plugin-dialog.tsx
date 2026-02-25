import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { pluginsApi } from "@/api/plugins"
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
  plugin: any
  projectId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function DeletePluginDialog({ plugin, projectId, open, onOpenChange }: DeletePluginDialogProps) {
  const queryClient = useQueryClient()

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
    deleteMutation.mutate()
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete Plugin</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete <strong>{plugin.name}</strong>? This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={handleDelete}
            disabled={deleteMutation.isPending}
            className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
          >
            {deleteMutation.isPending ? "Deleting..." : "Delete Plugin"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
