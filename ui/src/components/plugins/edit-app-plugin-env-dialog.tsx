import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type AxiosError } from "axios"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { pluginsApi, type AppPlugin } from "@/api/plugins"
import { KeyValueInput, type KeyValuePair } from "@/components/shared/key-value-input"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"

interface EditAppPluginEnvDialogProps {
  appId: string
  appPlugin: AppPlugin | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function EditAppPluginEnvDialog({
  appId,
  appPlugin,
  open,
  onOpenChange,
}: EditAppPluginEnvDialogProps) {
  const queryClient = useQueryClient()
  const [envVars, setEnvVars] = React.useState<KeyValuePair[]>([])

  React.useEffect(() => {
    if (appPlugin && open) {
      setEnvVars(Array.isArray(appPlugin.env_vars) ? appPlugin.env_vars : [])
    }
  }, [appPlugin, open])

  const updateMutation = useMutation<unknown, AxiosError<{ error: string }>, KeyValuePair[]>({
    mutationFn: (env_vars: KeyValuePair[]) =>
      pluginsApi.updateAppPluginEnv(appId, appPlugin!.plugin_id, env_vars),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-plugins", appId] })
      toast.success("Plugin environment variables updated")
      onOpenChange(false)
    },
    onError: (error) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update environment variables",
      })
    },
  })

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    updateMutation.mutate(envVars)
  }

  if (!appPlugin) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Edit Environment Variables</DialogTitle>
            <DialogDescription>
              Configure environment variables for <span className="font-medium text-foreground">{appPlugin.plugin.name}</span>
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <Field>
              <FieldLabel>Environment Variables</FieldLabel>
              <FieldContent>
                <KeyValueInput
                  value={envVars}
                  onChange={setEnvVars}
                />
              </FieldContent>
            </Field>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? (
                <>
                  <Loader2 className="h-4 w-4 animate-spin mr-2" />
                  Saving...
                </>
              ) : (
                "Save Changes"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
