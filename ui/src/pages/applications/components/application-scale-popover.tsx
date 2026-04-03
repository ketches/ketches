import { appsApi, type App } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { type AxiosError } from "axios"
import { Diff, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

export function ApplicationScalePopover({ app }: { app: App }) {
  const [replicas, setReplicas] = React.useState(app.replicas)
  const [open, setOpen] = React.useState(false)
  const queryClient = useQueryClient()

  React.useEffect(() => {
    setReplicas(app.replicas)
  }, [app.replicas])

  const scaleMutation = useMutation({
    mutationFn: (count: number) => appsApi.updateReplicas(app.id, count),
    onSuccess: () => {
      toast.success(`Application scaling to ${replicas} replicas initiated`)
      queryClient.invalidateQueries({ queryKey: ["app", app.id] })
      setOpen(false)
    },
    onError: (error: AxiosError<{ error: string }>) => {
      toast.error("Failed to scale application", {
        description: error.response?.data?.error || error.message,
      })
    },
  })

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger>
        <Button>
          <Diff />
          Scale: {app.replicas}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-80">
        <div className="space-y-4">
          <div className="space-y-2">
            <h4 className="font-medium text-sm">Scale Application</h4>
            <p className="text-xs text-muted-foreground">
              Change the number of desired replicas.
            </p>
          </div>
          <div className="flex items-center gap-4">
            <Button
              variant="outline"
              size="icon"
              onClick={() => setReplicas(Math.max(0, replicas - 1))}
              disabled={replicas <= 0}
            >
              -
            </Button>
            <Input
              type="number"
              value={replicas}
              onChange={(event) => setReplicas(parseInt(event.target.value) || 0)}
              className="text-center font-bold text-lg"
            />
            <Button
              variant="outline"
              size="icon"
              onClick={() => setReplicas(replicas + 1)}
            >
              +
            </Button>
          </div>
          {app.auto_scaling && (
            <p className="text-[10px] text-destructive font-medium">
              Warning: AutoScaling is enabled. Manual scaling might be overridden by the autoscaler.
            </p>
          )}
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={() => setOpen(false)} className="flex-1">
              Cancel
            </Button>
            <Button
              size="sm"
              onClick={() => scaleMutation.mutate(replicas)}
              disabled={scaleMutation.isPending}
              className="flex-1"
            >
              {scaleMutation.isPending && <Loader2 className="h-4 w-4 animate-spin" />}
              Scale
            </Button>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  )
}
