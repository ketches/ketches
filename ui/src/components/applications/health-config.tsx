import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { HeartPulse, Plus, Save, Trash2 } from "lucide-react"
import { Controller, useFieldArray, useForm, type Resolver } from "react-hook-form"
import { toast } from "sonner"
import * as z from "zod"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import type { AxiosError } from "axios"

const probeSchema = z.object({
  probes: z.array(z.object({
    type: z.enum(['liveness', 'readiness', 'startup']),
    probe_mode: z.enum(['httpGet', 'tcpSocket', 'exec']),
    enabled: z.boolean(),
    http_get_path: z.string().optional(),
    http_get_port: z.coerce.number().optional(),
    tcp_socket_port: z.coerce.number().optional(),
    exec_command: z.string().optional(),
    initial_delay_seconds: z.coerce.number().min(0),
    period_seconds: z.coerce.number().min(1),
    timeout_seconds: z.coerce.number().min(1),
    success_threshold: z.coerce.number().min(1),
    failure_threshold: z.coerce.number().min(1),
  }))
})

type ProbeFormValues = {
  probes: {
    type: 'liveness' | 'readiness' | 'startup'
    probe_mode: 'httpGet' | 'tcpSocket' | 'exec'
    enabled: boolean
    http_get_path?: string
    http_get_port?: number
    tcp_socket_port?: number
    exec_command?: string
    initial_delay_seconds: number
    period_seconds: number
    timeout_seconds: number
    success_threshold: number
    failure_threshold: number
  }[]
}

interface ProbeConfigProps {
  app: App
}

export function HealthConfig({ app }: ProbeConfigProps) {
  const queryClient = useQueryClient()
  const { control, register, handleSubmit, watch } = useForm<ProbeFormValues>({
    resolver: zodResolver(probeSchema) as Resolver<ProbeFormValues>,
    defaultValues: {
      probes: app.probes?.length ? app.probes : [
        { type: 'readiness', probe_mode: 'httpGet', enabled: false, http_get_path: '/', http_get_port: 80, initial_delay_seconds: 0, period_seconds: 10, timeout_seconds: 1, success_threshold: 1, failure_threshold: 3 },
        { type: 'liveness', probe_mode: 'httpGet', enabled: false, http_get_path: '/', http_get_port: 80, initial_delay_seconds: 0, period_seconds: 10, timeout_seconds: 1, success_threshold: 1, failure_threshold: 3 },
        { type: 'startup', probe_mode: 'httpGet', enabled: false, http_get_path: '/', http_get_port: 80, initial_delay_seconds: 0, period_seconds: 10, timeout_seconds: 1, success_threshold: 1, failure_threshold: 3 },
      ],
    },
  })

  const { fields, append, remove } = useFieldArray({
    control,
    name: "probes",
  })

  const updateMutation = useMutation({
    mutationFn: (values: z.infer<typeof probeSchema>) => {
      const data: Partial<App> = {
        ...app,
        probes: values.probes,
      }
      return appsApi.update(app.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app', app.id] })
      toast.success("Probe configuration updated")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to update probes", {
        description: err.response?.data?.error || "Unknown error"
      })
    }
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-2">
          <HeartPulse className="h-4 w-4" /> Health Checks
        </CardTitle>
        <CardDescription>Configure Readiness, Liveness, and Startup probes</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit((v) => updateMutation.mutate(v))} className="space-y-8">
          {fields.map((field, index) => {
            const isEnabled = watch(`probes.${index}.enabled`)
            const probeMode = watch(`probes.${index}.probe_mode`)

            return (
              <div key={field.id} className="p-4 border rounded-lg space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-4">
                    <Field orientation="horizontal" className="w-auto">
                      <FieldContent>
                        <Controller
                          control={control}
                          name={`probes.${index}.type`}
                          render={({ field }) => (
                            <Select
                              onValueChange={field.onChange}
                              defaultValue={field.value}
                              items={[
                                { value: "readiness", label: "READINESS" },
                                { value: "liveness", label: "LIVENESS" },
                                { value: "startup", label: "STARTUP" },
                              ]}
                            >
                              <SelectTrigger className="w-40 font-bold uppercase">
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="readiness">READINESS</SelectItem>
                                <SelectItem value="liveness">LIVENESS</SelectItem>
                                <SelectItem value="startup">STARTUP</SelectItem>
                              </SelectContent>
                            </Select>
                          )}
                        />
                      </FieldContent>
                    </Field>

                    <Field orientation="horizontal" className="w-auto gap-2">
                      <FieldContent>
                        <Controller
                          control={control}
                          name={`probes.${index}.enabled`}
                          render={({ field }) => (
                            <Checkbox checked={field.value} onCheckedChange={field.onChange} />
                          )}
                        />
                      </FieldContent>
                      <FieldLabel className="text-xs font-normal">Enabled</FieldLabel>
                    </Field>
                  </div>
                  <Button variant="ghost" size="icon" onClick={() => remove(index)}>
                    <Trash2 className="h-4 w-4 text-destructive" />
                  </Button>
                </div>

                {isEnabled && (
                  <div className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                      <Field>
                        <FieldLabel className="text-xs">Mode</FieldLabel>
                        <FieldContent>
                          <Controller
                            control={control}
                            name={`probes.${index}.probe_mode`}
                            render={({ field }) => (
                              <Select
                                onValueChange={field.onChange}
                                defaultValue={field.value}
                                items={[
                                  { value: "httpGet", label: "HTTP GET" },
                                  { value: "tcpSocket", label: "TCP Socket" },
                                  { value: "exec", label: "Exec Command" },
                                ]}
                              >
                                <SelectTrigger className="w-full">
                                  <SelectValue />
                                </SelectTrigger>
                                <SelectContent>
                                  <SelectItem value="httpGet">HTTP GET</SelectItem>
                                  <SelectItem value="tcpSocket">TCP Socket</SelectItem>
                                  <SelectItem value="exec">Exec Command</SelectItem>
                                </SelectContent>
                              </Select>
                            )}
                          />
                        </FieldContent>
                      </Field>
                    </div>

                    {probeMode === 'httpGet' && (
                      <div className="grid grid-cols-2 gap-4">
                        <Field>
                          <FieldLabel className="text-xs">Path</FieldLabel>
                          <FieldContent>
                            <Input {...register(`probes.${index}.http_get_path`)} />
                          </FieldContent>
                        </Field>
                        <Field>
                          <FieldLabel className="text-xs">Port</FieldLabel>
                          <FieldContent>
                            <Input type="number" {...register(`probes.${index}.http_get_port`)} />
                          </FieldContent>
                        </Field>
                      </div>
                    )}

                    {probeMode === 'tcpSocket' && (
                      <Field>
                        <FieldLabel className="text-xs">Port</FieldLabel>
                        <FieldContent>
                          <Input type="number" {...register(`probes.${index}.tcp_socket_port`)} />
                        </FieldContent>
                      </Field>
                    )}

                    {probeMode === 'exec' && (
                      <Field>
                        <FieldLabel className="text-xs">Command</FieldLabel>
                        <FieldContent>
                          <Input placeholder="cat /tmp/healthy" {...register(`probes.${index}.exec_command`)} />
                        </FieldContent>
                      </Field>
                    )}

                    <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                      <Field>
                        <FieldLabel className="text-xs">Delay (s)</FieldLabel>
                        <FieldContent>
                          <Input type="number" {...register(`probes.${index}.initial_delay_seconds`)} />
                        </FieldContent>
                      </Field>
                      <Field>
                        <FieldLabel className="text-xs">Period (s)</FieldLabel>
                        <FieldContent>
                          <Input type="number" {...register(`probes.${index}.period_seconds`)} />
                        </FieldContent>
                      </Field>
                      <Field>
                        <FieldLabel className="text-xs">Timeout (s)</FieldLabel>
                        <FieldContent>
                          <Input type="number" {...register(`probes.${index}.timeout_seconds`)} />
                        </FieldContent>
                      </Field>
                      <Field>
                        <FieldLabel className="text-xs">Success Thres.</FieldLabel>
                        <FieldContent>
                          <Input type="number" {...register(`probes.${index}.success_threshold`)} />
                        </FieldContent>
                      </Field>
                      <Field>
                        <FieldLabel className="text-xs">Failure Thres.</FieldLabel>
                        <FieldContent>
                          <Input type="number" {...register(`probes.${index}.failure_threshold`)} />
                        </FieldContent>
                      </Field>
                    </div>
                  </div>
                )}
              </div>
            )
          })}

          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={() => append({ type: 'readiness', probe_mode: 'httpGet', enabled: false, http_get_path: '/', http_get_port: 80, initial_delay_seconds: 0, period_seconds: 10, timeout_seconds: 1, success_threshold: 1, failure_threshold: 3 })}
          >
            <Plus /> Add Probe
          </Button>

          <div className="flex justify-end pt-4">
            <Button type="submit" disabled={updateMutation.isPending}>
              <Save />
              {updateMutation.isPending ? "Saving..." : "Save"}
            </Button>
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
