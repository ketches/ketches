import * as React from "react"
import { toast } from "sonner"

import { usePlatformUpdateConfigQuery, useUpdatePlatformUpdateConfigMutation } from "@/hooks/use-platform-update"
import { Button } from "@/components/ui/button"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"

interface PlatformUpdateConfigDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface PlatformUpdateConfigFormState {
  apiImageRepository: string
  apiNamespace: string
  apiDeploymentName: string
  apiContainerName: string
  uiImageRepository: string
  uiNamespace: string
  uiDeploymentName: string
  uiContainerName: string
}

function emptyFormState(): PlatformUpdateConfigFormState {
  return {
    apiImageRepository: "",
    apiNamespace: "",
    apiDeploymentName: "",
    apiContainerName: "",
    uiImageRepository: "",
    uiNamespace: "",
    uiDeploymentName: "",
    uiContainerName: "",
  }
}

export function PlatformUpdateConfigDialog({
  open,
  onOpenChange,
}: PlatformUpdateConfigDialogProps) {
  const { data, isLoading } = usePlatformUpdateConfigQuery(open)
  const updateMutation = useUpdatePlatformUpdateConfigMutation()
  const [form, setForm] = React.useState<PlatformUpdateConfigFormState>(emptyFormState)

  React.useEffect(() => {
    if (!open || !data) return
    setForm({
      apiImageRepository: data.api.image_repository,
      apiNamespace: data.api.namespace,
      apiDeploymentName: data.api.deployment_name,
      apiContainerName: data.api.container_name,
      uiImageRepository: data.ui.image_repository,
      uiNamespace: data.ui.namespace,
      uiDeploymentName: data.ui.deployment_name,
      uiContainerName: data.ui.container_name,
    })
  }, [data, open])

  const updateField = (field: keyof PlatformUpdateConfigFormState, value: string) => {
    setForm((current) => ({ ...current, [field]: value }))
  }

  const handleSubmit = (event: React.SubmitEvent<HTMLFormElement>) => {
    event.preventDefault()

    updateMutation.mutate({
      api: {
        image_repository: form.apiImageRepository.trim(),
        namespace: form.apiNamespace.trim(),
        deployment_name: form.apiDeploymentName.trim(),
        container_name: form.apiContainerName.trim(),
      },
      ui: {
        image_repository: form.uiImageRepository.trim(),
        namespace: form.uiNamespace.trim(),
        deployment_name: form.uiDeploymentName.trim(),
        container_name: form.uiContainerName.trim(),
      },
    }, {
      onSuccess: () => {
        toast.success("Platform update targets saved")
        onOpenChange(false)
      },
      onError: (error) => {
        toast.error("Failed to save platform update targets", {
          description: error instanceof Error ? error.message : String(error),
        })
      },
    })
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={handleSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>Platform Update Targets</DialogTitle>
            <DialogDescription>
              Configure the image repositories and Kubernetes rollout targets for the
              platform API and UI deployments.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <section className="space-y-3">
              <h3 className="text-sm font-medium">API</h3>
              <Field>
                <FieldLabel htmlFor="api-image-repository">Image Repository</FieldLabel>
                <FieldContent>
                  <Input
                    id="api-image-repository"
                    name="api-image-repository"
                    value={form.apiImageRepository}
                    onChange={(event) => updateField("apiImageRepository", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="api-namespace">Namespace</FieldLabel>
                <FieldContent>
                  <Input
                    id="api-namespace"
                    name="api-namespace"
                    value={form.apiNamespace}
                    onChange={(event) => updateField("apiNamespace", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="api-deployment-name">Deployment</FieldLabel>
                <FieldContent>
                  <Input
                    id="api-deployment-name"
                    name="api-deployment-name"
                    value={form.apiDeploymentName}
                    onChange={(event) => updateField("apiDeploymentName", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="api-container-name">Container</FieldLabel>
                <FieldContent>
                  <Input
                    id="api-container-name"
                    name="api-container-name"
                    value={form.apiContainerName}
                    onChange={(event) => updateField("apiContainerName", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
            </section>

            <section className="space-y-3">
              <h3 className="text-sm font-medium">UI</h3>
              <Field>
                <FieldLabel htmlFor="ui-image-repository">Image Repository</FieldLabel>
                <FieldContent>
                  <Input
                    id="ui-image-repository"
                    name="ui-image-repository"
                    value={form.uiImageRepository}
                    onChange={(event) => updateField("uiImageRepository", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="ui-namespace">Namespace</FieldLabel>
                <FieldContent>
                  <Input
                    id="ui-namespace"
                    name="ui-namespace"
                    value={form.uiNamespace}
                    onChange={(event) => updateField("uiNamespace", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="ui-deployment-name">Deployment</FieldLabel>
                <FieldContent>
                  <Input
                    id="ui-deployment-name"
                    name="ui-deployment-name"
                    value={form.uiDeploymentName}
                    onChange={(event) => updateField("uiDeploymentName", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
              <Field>
                <FieldLabel htmlFor="ui-container-name">Container</FieldLabel>
                <FieldContent>
                  <Input
                    id="ui-container-name"
                    name="ui-container-name"
                    value={form.uiContainerName}
                    onChange={(event) => updateField("uiContainerName", event.target.value)}
                    disabled={isLoading || updateMutation.isPending}
                    required
                  />
                </FieldContent>
              </Field>
            </section>
          </div>

          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={isLoading || updateMutation.isPending}>
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
