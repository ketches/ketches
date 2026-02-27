import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Play, Save } from "lucide-react"
import { useForm } from "react-hook-form"
import { toast } from "sonner"
import * as z from "zod"

import type { App } from "@/api/apps"
import { appsApi } from "@/api/apps"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldContent, FieldDescription, FieldError, FieldLabel } from "@/components/ui/field"
import { Textarea } from "@/components/ui/textarea"
import type { AxiosError } from "axios"
import { useProjectRole } from "@/hooks/useProjectRole"

const commandSchema = z.object({
  container_command: z.string().optional(),
})

interface CommandConfigProps {
  app: App
}

export function CommandConfig({ app }: CommandConfigProps) {
  const queryClient = useQueryClient()
  const projectRole = useProjectRole()
  const isViewer = projectRole === 'viewer'
  const { register, handleSubmit, formState: { errors } } = useForm<z.infer<typeof commandSchema>>({
    resolver: zodResolver(commandSchema),
    defaultValues: {
      container_command: app.container_command || "",
    },
  })

  const updateMutation = useMutation({
    mutationFn: (values: z.infer<typeof commandSchema>) => {
      const data: Partial<App> = {
        ...app,
        container_command: values.container_command,
      }
      return appsApi.update(app.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['app', app.id] })
      toast.success("Command configuration updated")
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error("Failed to update command", {
        description: err.response?.data?.error || "Unknown error"
      })
    }
  })

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-sm flex items-center gap-2"><Play className="h-4 w-4" /> Startup Command</CardTitle>
        <CardDescription>Override the default container startup command</CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit((v) => updateMutation.mutate(v))} className="space-y-2">
          <Field>
            <FieldLabel>Command</FieldLabel>
            <FieldContent>
              <Textarea
                placeholder="echo Hello World"
                className="font-mono min-h-32"
                {...register("container_command")}
              />
            </FieldContent>
            <FieldDescription>
              <i>The command will be executed via <code>sh -c</code>.</i>
            </FieldDescription>
            {errors.container_command && <FieldError>{errors.container_command.message}</FieldError>}
          </Field>

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
