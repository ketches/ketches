import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Save, Scale, Trash2 } from "lucide-react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import * as z from "zod"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Card, CardAction, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList
} from "@/components/ui/combobox"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useProjectRole } from "@/hooks/useProjectRole"
import type { AxiosError } from "axios"

const METRIC_OPTIONS = [
  { value: "cpu", label: "CPU Utilization" },
  { value: "memory", label: "Memory Utilization" },
]
const autoScalingSchema = z.object({
  enabled: z.boolean(),
  min_replicas: z.number().min(0),
  max_replicas: z.number().min(1),
  cpu_enabled: z.boolean(),
  memory_enabled: z.boolean(),
  target_cpu_utilization: z.number().min(0).max(100),
  target_memory_utilization: z.number().min(0).max(100),
})

interface AutoScalingConfigProps {
  app: App
}

export function AutoScalingConfig({ app }: AutoScalingConfigProps) {
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole !== 'owner' && projectRole !== 'developer'
  const { register, handleSubmit, watch, setValue, formState: { errors } } = useForm<z.infer<typeof autoScalingSchema>>({
    resolver: zodResolver(autoScalingSchema),
    defaultValues: {
      enabled: !!app.auto_scaling,
      min_replicas: app.auto_scaling?.min_replicas ?? 1,
      max_replicas: app.auto_scaling?.max_replicas ?? 10,
      cpu_enabled: (app.auto_scaling?.target_cpu_utilization ?? 0) > 0,
      memory_enabled: (app.auto_scaling?.target_memory_utilization ?? 0) > 0,
      target_cpu_utilization: app.auto_scaling?.target_cpu_utilization || 80,
      target_memory_utilization: app.auto_scaling?.target_memory_utilization || 80,
    },
  })

  const enabled = watch("enabled")
  const cpuEnabled = watch("cpu_enabled")
  const memoryEnabled = watch("memory_enabled")

  const updateMutation = useMutation({
    mutationFn: (values: z.infer<typeof autoScalingSchema>) => {
      const auto_scaling = values.enabled ? {
        min_replicas: values.min_replicas,
        max_replicas: values.max_replicas,
        target_cpu_utilization: values.cpu_enabled ? values.target_cpu_utilization : 0,
        target_memory_utilization: values.memory_enabled ? values.target_memory_utilization : 0,
      } : null
      return appsApi.updateAutoScaling(app.id, auto_scaling)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app', app.id] })
      toast.success("AutoScaling configuration updated")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to update AutoScaling configuration", {
        description: err.response?.data?.error || "Unknown error"
      })
    }
  })

  return (
    <Card>
      <form onSubmit={handleSubmit((v) => updateMutation.mutate(v))}>
        <CardHeader>
          <CardTitle className="text-sm flex items-center gap-2">
            <Scale className="h-4 w-4" /> AutoScaling
          </CardTitle>
          <CardDescription>Automatically scale replicas based on CPU and Memory usage</CardDescription>
          <CardAction>
            <div className="flex items-center gap-2">
              <Checkbox
                checked={enabled}
                onCheckedChange={(checked) => setValue("enabled", checked === true)}
              />
              <span className="text-xs text-muted-foreground">{enabled ? "Enabled" : "Disabled"}</span>
              {!isViewer && (
                <Button type="submit" onClick={handleSubmit((v) => updateMutation.mutate(v))} disabled={updateMutation.isPending}>
                  <Save />
                  {updateMutation.isPending ? "Saving..." : "Save"}
                </Button>
              )}
            </div>
          </CardAction>
        </CardHeader>
        <CardContent>
          <div className="space-y-4 mt-4">
            <div className={`space-y-4 transition-opacity duration-200 ${enabled ? 'opacity-100' : 'opacity-50 pointer-events-none'}`}>
              <div className="space-y-2">
                <h4 className="text-sm font-medium">Replica Range</h4>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <Field>
                    <FieldLabel className="text-xs text-muted-foreground">Min Replicas</FieldLabel>
                    <FieldContent>
                      <Input type="number" {...register("min_replicas", { valueAsNumber: true })} disabled={!enabled} />
                    </FieldContent>
                    {errors.min_replicas && <FieldError>{errors.min_replicas.message}</FieldError>}
                  </Field>
                  <Field>
                    <FieldLabel className="text-xs text-muted-foreground">Max Replicas</FieldLabel>
                    <FieldContent>
                      <Input type="number" {...register("max_replicas", { valueAsNumber: true })} disabled={!enabled} />
                    </FieldContent>
                    {errors.max_replicas && <FieldError>{errors.max_replicas.message}</FieldError>}
                  </Field>
                </div>
              </div>

              <div className="space-y-2">
                <div className="flex items-center justify-between">
                  <h4 className="text-sm font-medium">Metrics</h4>
                  {(!cpuEnabled || !memoryEnabled) && (
                    <Combobox
                      value={null}
                      onValueChange={(value: string | null) => {
                        if (value === "cpu") setValue("cpu_enabled", true)
                        if (value === "memory") setValue("memory_enabled", true)
                      }}
                      itemToStringLabel={(v) => METRIC_OPTIONS.find((o) => o.value === v)?.label ?? v ?? ""}
                      disabled={!enabled}
                    >
                      <ComboboxInput placeholder="Add Metric" />
                      <ComboboxContent>
                        <ComboboxList>
                          {[
                            ...(!cpuEnabled ? [{ value: "cpu", label: "CPU Utilization" }] : []),
                            ...(!memoryEnabled ? [{ value: "memory", label: "Memory Utilization" }] : []),
                          ].map((option) => (
                            <ComboboxItem key={option.value} value={option.value}>
                              {option.label}
                            </ComboboxItem>
                          ))}
                        </ComboboxList>
                      </ComboboxContent>
                    </Combobox>
                  )}
                </div>

                <div className="space-y-4">
                  {cpuEnabled && (
                    <div className="flex items-end gap-2 p-3 border rounded-lg bg-muted/30">
                      <div className="flex-1 grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="flex flex-col gap-2">
                          <span className="text-xs font-medium text-muted-foreground">Metric Type</span>
                          <div className="h-7 flex items-center px-3 bg-muted rounded border text-xs">
                            CPU Utilization
                          </div>
                        </div>
                        <Field>
                          <FieldLabel className="text-xs text-muted-foreground">Target Utilization (%)</FieldLabel>
                          <FieldContent>
                            <Input type="number" {...register("target_cpu_utilization", { valueAsNumber: true })} disabled={!enabled} />
                          </FieldContent>
                          {errors.target_cpu_utilization && <FieldError>{errors.target_cpu_utilization.message}</FieldError>}
                        </Field>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={() => setValue("cpu_enabled", false)}
                        disabled={!enabled}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  )}

                  {memoryEnabled && (
                    <div className="flex items-end gap-2 p-3 border rounded-lg bg-muted/30">
                      <div className="flex-1 grid grid-cols-1 md:grid-cols-2 gap-4">
                        <div className="flex flex-col gap-2">
                          <span className="text-xs font-medium text-muted-foreground">Metric Type</span>
                          <div className="h-7 flex items-center px-3 bg-muted rounded border text-xs">
                            Memory Utilization
                          </div>
                        </div>
                        <Field>
                          <FieldLabel className="text-xs text-muted-foreground">Target Utilization (%)</FieldLabel>
                          <FieldContent>
                            <Input type="number" {...register("target_memory_utilization", { valueAsNumber: true })} disabled={!enabled} />
                          </FieldContent>
                          {errors.target_memory_utilization && <FieldError>{errors.target_memory_utilization.message}</FieldError>}
                        </Field>
                      </div>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                        onClick={() => setValue("memory_enabled", false)}
                        disabled={!enabled}
                      >
                        <Trash2 />
                      </Button>
                    </div>
                  )}

                  {!cpuEnabled && !memoryEnabled && (
                    <EmptyState title="" description="No metrics configured. Click 'Add Metric' to begin." />
                  )}
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </form>
    </Card>
  )
}
