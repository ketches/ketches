import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type AxiosError } from "axios"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { pluginsApi, type AppPlugin } from "@/api/plugins"
import { Button } from "@/components/ui/button"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"

interface PluginResourcePopoverProps {
  appId: string
  appPlugin: AppPlugin
  children: React.ReactNode
}

export function PluginResourcePopover({
  appId,
  appPlugin,
  children,
}: PluginResourcePopoverProps) {
  const queryClient = useQueryClient()
  const [open, setOpen] = React.useState(false)
  const [formData, setFormData] = React.useState({
    request_cpu: appPlugin.request_cpu || 0,
    limit_cpu: appPlugin.limit_cpu || 0,
    request_memory: appPlugin.request_memory || 0,
    limit_memory: appPlugin.limit_memory || 0,
  })

  React.useEffect(() => {
    if (open) {
      setFormData({
        request_cpu: appPlugin.request_cpu || 0,
        limit_cpu: appPlugin.limit_cpu || 0,
        request_memory: appPlugin.request_memory || 0,
        limit_memory: appPlugin.limit_memory || 0,
      })
    }
  }, [open, appPlugin])

  const updateMutation = useMutation<unknown, AxiosError<{ error: string }>, {
    request_cpu?: number
    limit_cpu?: number
    request_memory?: number
    limit_memory?: number
  }>({
    mutationFn: (resources: {
      request_cpu?: number
      limit_cpu?: number
      request_memory?: number
      limit_memory?: number
    }) => pluginsApi.updateAppPluginResources(appId, appPlugin.plugin_id, resources),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["app-plugins", appId] })
      toast.success("Plugin resources updated")
      setOpen(false)
    },
    onError: (error) => {
      toast.error("Error", {
        description: error.response?.data?.error || "Failed to update resources",
      })
    },
  })

  const handleSubmit = (e: React.SubmitEvent<HTMLFormElement>) => {
    e.preventDefault()
    updateMutation.mutate(formData)
  }

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>{children}</PopoverTrigger>
      <PopoverContent className="w-80">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <h4 className="font-medium text-sm">Resource Limits</h4>
            <p className="text-xs text-muted-foreground">
              Configure CPU and memory for {appPlugin.plugin.name}
            </p>
          </div>

          <div className="grid gap-4">
            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="request_cpu" className="text-xs">
                  CPU Request (m)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="request_cpu"
                    type="number"
                    min="0"
                    value={formData.request_cpu}
                    onChange={(e) =>
                      setFormData({ ...formData, request_cpu: parseInt(e.target.value) || 0 })
                    }
                    className="h-8"
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="limit_cpu" className="text-xs">
                  CPU Limit (m)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="limit_cpu"
                    type="number"
                    min="0"
                    value={formData.limit_cpu}
                    onChange={(e) =>
                      setFormData({ ...formData, limit_cpu: parseInt(e.target.value) || 0 })
                    }
                    className="h-8"
                  />
                </FieldContent>
              </Field>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <Field>
                <FieldLabel htmlFor="request_memory" className="text-xs">
                  Memory Request (Mi)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="request_memory"
                    type="number"
                    min="0"
                    value={formData.request_memory}
                    onChange={(e) =>
                      setFormData({ ...formData, request_memory: parseInt(e.target.value) || 0 })
                    }
                    className="h-8"
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="limit_memory" className="text-xs">
                  Memory Limit (Mi)
                </FieldLabel>
                <FieldContent>
                  <Input
                    id="limit_memory"
                    type="number"
                    min="0"
                    value={formData.limit_memory}
                    onChange={(e) =>
                      setFormData({ ...formData, limit_memory: parseInt(e.target.value) || 0 })
                    }
                    className="h-8"
                  />
                </FieldContent>
              </Field>
            </div>
          </div>

          <div className="flex justify-end gap-2 pt-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => setOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin mr-1.5" />
                  Saving...
                </>
              ) : (
                "Save"
              )}
            </Button>
          </div>
        </form>
      </PopoverContent>
    </Popover>
  )
}
