import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Ruler, Save } from "lucide-react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import * as z from "zod"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldContent, FieldError, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useProjectRole } from "@/hooks/useProjectRole"
import type { AxiosError } from "axios"

const resourceSchema = z.object({
  request_cpu: z.number().min(0),
  request_memory: z.number().min(0),
  limit_cpu: z.number().min(0),
  limit_memory: z.number().min(0),
})

interface ResourceConfigProps {
  app: App
}

export function ResourceConfig({ app }: ResourceConfigProps) {
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const { register, handleSubmit, formState: { errors } } = useForm<z.infer<typeof resourceSchema>>({
    resolver: zodResolver(resourceSchema),
    defaultValues: {
      request_cpu: app.request_cpu,
      request_memory: app.request_memory,
      limit_cpu: app.limit_cpu,
      limit_memory: app.limit_memory,
    },
  })

  const updateMutation = useMutation({
    mutationFn: (values: z.infer<typeof resourceSchema>) => {
      return appsApi.updateResources(app.id, {
        request_cpu: values.request_cpu,
        request_memory: values.request_memory,
        limit_cpu: values.limit_cpu,
        limit_memory: values.limit_memory,
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app', app.id] })
      toast.success("Resource configuration updated")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to update resources", {
        description: err.response?.data?.error || "Unknown error"
      })
    }
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-2">
          <Ruler className="h-4 w-4" /> Resources
        </CardTitle>
        <CardDescription>Configure CPU and Memory requests and limits</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit((v) => updateMutation.mutate(v))} className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div className="space-y-4">
              <h4 className="text-sm font-medium">CPU (milliCPUs)</h4>
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel className="text-xs">Request</FieldLabel>
                  <FieldContent>
                    <Input type="number" {...register("request_cpu", { valueAsNumber: true })} />
                  </FieldContent>
                  {errors.request_cpu && <FieldError>{errors.request_cpu.message}</FieldError>}
                </Field>
                <Field>
                  <FieldLabel className="text-xs">Limit</FieldLabel>
                  <FieldContent>
                    <Input type="number" {...register("limit_cpu", { valueAsNumber: true })} />
                  </FieldContent>
                  {errors.limit_cpu && <FieldError>{errors.limit_cpu.message}</FieldError>}
                </Field>
              </div>
            </div>

            <div className="space-y-4">
              <h4 className="text-sm font-medium">Memory (MiB)</h4>
              <div className="grid grid-cols-2 gap-4">
                <Field>
                  <FieldLabel className="text-xs">Request</FieldLabel>
                  <FieldContent>
                    <Input type="number" {...register("request_memory", { valueAsNumber: true })} />
                  </FieldContent>
                  {errors.request_memory && <FieldError>{errors.request_memory.message}</FieldError>}
                </Field>
                <Field>
                  <FieldLabel className="text-xs">Limit</FieldLabel>
                  <FieldContent>
                    <Input type="number" {...register("limit_memory", { valueAsNumber: true })} />
                  </FieldContent>
                  {errors.limit_memory && <FieldError>{errors.limit_memory.message}</FieldError>}
                </Field>
              </div>
            </div>
          </div>

          <div className="flex justify-end">
            {!isViewer && (
              <Button type="submit" disabled={updateMutation.isPending}>
                <Save />
                {updateMutation.isPending ? "Saving..." : "Save"}
              </Button>
            )}
          </div>
        </form>
      </CardContent>
    </Card>
  )
}
