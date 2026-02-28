import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { codeRepositoriesApi, type CreateCodeRepositoryBuildConfigRequest } from "@/api/code-repositories"
import { registryProviderLabels } from "@/api/container-registries"
import { GitRefSelect } from "@/components/code-repositories/git-ref-select"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { SimpleCombobox } from "@/components/ui/simple-combobox"
import type { AxiosError } from "axios"

interface CreateBuildConfigDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  repoId: string
  onSuccess?: () => void
}

const defaultForm: CreateCodeRepositoryBuildConfigRequest = {
  name: 'Default',
  git_ref: 'main',
  dockerfile_path: 'Dockerfile',
  build_context: '.',
  image_name: '',
  registry_id: '',
  build_args: '',
  auto_build: false,
  auto_deploy: false,
  webhook_enabled: false,
}

export function CreateBuildConfigDialog({ open, onOpenChange, repoId, onSuccess }: CreateBuildConfigDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<CreateCodeRepositoryBuildConfigRequest>(defaultForm)

  const { data: repo } = useQuery({
    queryKey: ['code-repository', repoId],
    queryFn: () => codeRepositoriesApi.get(repoId),
    enabled: open && !!repoId,
  })

  const { data: registries } = useQuery({
    queryKey: ['code-repository-registries', repoId],
    queryFn: () => codeRepositoriesApi.listContainerRegistries(repoId),
    enabled: open && !!repoId,
  })

  React.useEffect(() => {
    if (open && repo?.slug && !form.image_name) {
      setForm((prev) => ({ ...prev, image_name: repo.slug }))
    }
  }, [open, repo?.slug, form.image_name])

  const createMutation = useMutation({
    mutationFn: () => codeRepositoriesApi.createBuildConfig(repoId, form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['code-repository-build-configs', repoId] })
      toast.success('Build config added')
      onOpenChange(false)
      setForm(defaultForm)
      onSuccess?.()
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to add build config')
    },
  })

  const handleSubmit = () => {
    if (!form.name?.trim() || !form.image_name?.trim() || !form.registry_id) {
      toast.error('Name, image name, and registry are required')
      return
    }
    createMutation.mutate()
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Add Build Config</DialogTitle>
          <DialogDescription>
            One repository can have multiple build configs (e.g. frontend, backend). Configure Dockerfile path, context, image name, and registry.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <Field>
            <FieldLabel>Name *</FieldLabel>
            <FieldContent>
              <Input
                placeholder="e.g. backend, frontend"
                value={form.name}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel>Git Branch / Tag</FieldLabel>
            <FieldContent>
              <GitRefSelect
                repoId={repoId}
                value={form.git_ref ?? 'main'}
                onValueChange={(v) => setForm({ ...form, git_ref: v ?? undefined })}
              />
            </FieldContent>
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Dockerfile path</FieldLabel>
              <FieldContent>
                <Input
                  value={form.dockerfile_path ?? ''}
                  onChange={(e) => setForm({ ...form, dockerfile_path: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Build context</FieldLabel>
              <FieldContent>
                <Input
                  value={form.build_context ?? ''}
                  onChange={(e) => setForm({ ...form, build_context: e.target.value })}
                />
              </FieldContent>
            </Field>
          </div>
          <Field>
            <FieldLabel>Image name *</FieldLabel>
            <FieldContent>
              <Input
                placeholder="my-service"
                value={form.image_name}
                onChange={(e) => setForm({ ...form, image_name: e.target.value })}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel>Container registry *</FieldLabel>
            <FieldContent>
              <SimpleCombobox
                value={form.registry_id}
                onValueChange={(v) => setForm({ ...form, registry_id: v ?? "" })}
                options={
                  registries?.map((r) => ({
                    value: r.id,
                    label: `${r.name} (${registryProviderLabels[r.provider]})`,
                  })) ?? []
                }
                placeholder="Select registry"
                className="w-full"
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel>Build args (optional)</FieldLabel>
            <FieldContent>
              <Input
                placeholder="KEY1=val1,KEY2=val2"
                value={form.build_args ?? ''}
                onChange={(e) => setForm({ ...form, build_args: e.target.value })}
              />
            </FieldContent>
          </Field>
          <div className="flex items-center gap-2">
            <Checkbox
              id="webhook-enabled"
              checked={form.webhook_enabled ?? false}
              onCheckedChange={(v) => setForm({ ...form, webhook_enabled: v === true })}
            />
            <label htmlFor="webhook-enabled" className="cursor-pointer">
              Auto build on webhook
            </label>
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              id="auto-deploy"
              checked={form.auto_deploy ?? false}
              onCheckedChange={(v) => setForm({ ...form, auto_deploy: v === true })}
            />
            <label htmlFor="auto-deploy" className="cursor-pointer">
              Auto deploy after build
            </label>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={createMutation.isPending}>
            {createMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Add
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
