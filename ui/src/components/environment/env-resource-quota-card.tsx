import { Edit2, Gauge, Loader2 } from "lucide-react"
import * as React from "react"

import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useEnvResourceQuota, useUpdateEnvResourceQuotaMutation } from "@/hooks/use-env-resource-quota"

interface EnvResourceQuotaCardProps {
  envId: string
  isViewer: boolean
}

export function EnvResourceQuotaCard({ envId, isViewer }: EnvResourceQuotaCardProps) {
  const { data, isLoading } = useEnvResourceQuota(envId)
  const updateMutation = useUpdateEnvResourceQuotaMutation(envId)
  const [dialogOpen, setDialogOpen] = React.useState(false)
  const [cpuRequest, setCpuRequest] = React.useState("2")
  const [cpuLimit, setCpuLimit] = React.useState("4")
  const [memoryRequest, setMemoryRequest] = React.useState("4Gi")
  const [memoryLimit, setMemoryLimit] = React.useState("8Gi")
  const [pods, setPods] = React.useState("50")

  React.useEffect(() => {
    if (dialogOpen && data) {
      setCpuRequest(data.cpu_request)
      setCpuLimit(data.cpu_limit)
      setMemoryRequest(data.memory_request)
      setMemoryLimit(data.memory_limit)
      setPods(data.pods)
    }
  }, [data, dialogOpen])

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    updateMutation.mutate(
      {
        cpu_request: cpuRequest,
        cpu_limit: cpuLimit,
        memory_request: memoryRequest,
        memory_limit: memoryLimit,
        pods: pods,
      },
      {
        onSuccess: () => setDialogOpen(false),
      }
    )
  }

  return (
    <>
      <Card className="bg-linear-to-b/increasing from-blue-500/5 to-transparent group/card">
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Gauge className="h-4 w-4" />
            Resource Quota
          </CardTitle>
          <CardDescription>
            Configure CPU, memory, and pod count limits for this environment namespace.
          </CardDescription>
          {!isViewer && (
            <CardAction className="opacity-0 transition-opacity group-hover/card:opacity-100 group-focus-within/card:opacity-100">
              <Tooltip>
                <TooltipTrigger
                  render={
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      aria-label="Edit resource quota"
                      disabled={isLoading}
                      onClick={() => setDialogOpen(true)}
                    />
                  }
                >
                  <Edit2 />
                </TooltipTrigger>
                <TooltipContent>Edit</TooltipContent>
              </Tooltip>
            </CardAction>
          )}
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
          ) : (
            <div className="grid grid-cols-2 gap-4 sm:grid-cols-3 lg:grid-cols-5">
              <div>
                <p className="text-xs font-medium text-muted-foreground">CPU Request</p>
                <p className="text-sm font-mono">{data?.cpu_request ?? "2"} cores</p>
              </div>
              <div>
                <p className="text-xs font-medium text-muted-foreground">CPU Limit</p>
                <p className="text-sm font-mono">{data?.cpu_limit ?? "4"} cores</p>
              </div>
              <div>
                <p className="text-xs font-medium text-muted-foreground">Memory Request</p>
                <p className="text-sm font-mono">{data?.memory_request ?? "4Gi"}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-muted-foreground">Memory Limit</p>
                <p className="text-sm font-mono">{data?.memory_limit ?? "8Gi"}</p>
              </div>
              <div>
                <p className="text-xs font-medium text-muted-foreground">Max Pods</p>
                <p className="text-sm font-mono">{data?.pods ?? "50"}</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            <DialogHeader>
              <DialogTitle>Edit Resource Quota</DialogTitle>
              <DialogDescription>
                Set CPU, memory, and pod count limits for this environment.
              </DialogDescription>
            </DialogHeader>


            <Field>
              <FieldLabel htmlFor="pods">Max Pods</FieldLabel>
              <FieldContent>
                <Input
                  id="pods"
                  name="pods"
                  type="number"
                  min="1"
                  value={pods}
                  onChange={(e) => setPods(e.target.value)}
                  disabled={updateMutation.isPending}
                  required
                />
              </FieldContent>
            </Field>

            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="cpu-request">CPU Request (cores)</FieldLabel>
                <FieldContent>
                  <Input
                    id="cpu-request"
                    name="cpu-request"
                    type="number"
                    min="0.1"
                    step="0.1"
                    value={cpuRequest}
                    onChange={(e) => setCpuRequest(e.target.value)}
                    disabled={updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="cpu-limit">CPU Limit (cores)</FieldLabel>
                <FieldContent>
                  <Input
                    id="cpu-limit"
                    name="cpu-limit"
                    type="number"
                    min="0.1"
                    step="0.1"
                    value={cpuLimit}
                    onChange={(e) => setCpuLimit(e.target.value)}
                    disabled={updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
            </div>
            <div className="grid grid-cols-2 gap-4">
              <Field>
                <FieldLabel htmlFor="memory-request">Memory Request</FieldLabel>
                <FieldContent>
                  <Input
                    id="memory-request"
                    name="memory-request"
                    type="text"
                    placeholder="e.g., 4Gi"
                    value={memoryRequest}
                    onChange={(e) => setMemoryRequest(e.target.value)}
                    disabled={updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>

              <Field>
                <FieldLabel htmlFor="memory-limit">Memory Limit</FieldLabel>
                <FieldContent>
                  <Input
                    id="memory-limit"
                    name="memory-limit"
                    type="text"
                    placeholder="e.g., 8Gi"
                    value={memoryLimit}
                    onChange={(e) => setMemoryLimit(e.target.value)}
                    disabled={updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
            </div>

            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                Cancel
              </Button>
              <Button type="submit" disabled={updateMutation.isPending}>
                {updateMutation.isPending ? (
                  <>
                    <Loader2 className="h-4 w-4 animate-spin" />
                    Saving...
                  </>
                ) : (
                  "Save"
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </>
  )
}
