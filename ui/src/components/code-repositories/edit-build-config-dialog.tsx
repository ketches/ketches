import { codeRepositoriesApi, type CodeRepositoryBuildConfig, type UpdateCodeRepositoryBuildConfigRequest } from "@/api/code-repositories"
import { registryProviderLabels } from "@/api/container-registries"
import { GitRefSelect } from "@/components/code-repositories/git-ref-select"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Item, ItemContent, ItemDescription, ItemTitle } from "@/components/ui/item"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import type { AxiosError } from "axios"
import { Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

interface EditBuildConfigDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  repoId: string
  config: CodeRepositoryBuildConfig | null
  onSuccess?: () => void
}

export function EditBuildConfigDialog({ open, onOpenChange, repoId, config, onSuccess }: EditBuildConfigDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<UpdateCodeRepositoryBuildConfigRequest>({})

  React.useEffect(() => {
    if (config) {
      setForm({
        name: config.name,
        git_ref: config.git_ref,
        dockerfile_path: config.dockerfile_path,
        build_context: config.build_context,
        image_name: config.image_name,
        registry_id: config.registry_id,
        build_args: config.build_args,
        auto_build: config.auto_build,
        auto_deploy: config.auto_deploy,
        webhook_enabled: config.webhook_enabled,
      })
    }
  }, [config, open])

  const { data: registries } = useQuery({
    queryKey: ['code-repository-registries', repoId],
    queryFn: () => codeRepositoriesApi.listContainerRegistries(repoId),
    enabled: open && !!repoId,
  })

  const updateMutation = useMutation({
    mutationFn: () => codeRepositoriesApi.updateBuildConfig(repoId, config!.id, form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['code-repository-build-configs', repoId] })
      toast.success('Build config updated')
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to update build config')
    },
  })

  const handleSubmit = () => {
    if (!form.name?.trim() || !form.image_name?.trim() || !form.registry_id) {
      toast.error('Name, image name, and registry are required')
      return
    }
    updateMutation.mutate()
  }

  if (!config) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Edit Build Config</DialogTitle>
          <DialogDescription>
            Update build configuration for this repository.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <Field>
            <FieldLabel>Name *</FieldLabel>
            <FieldContent>
              <Input
                placeholder="e.g. backend, frontend"
                value={form.name || ''}
                onChange={(e) => setForm({ ...form, name: e.target.value })}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel>Git Branch / Tag</FieldLabel>
            <FieldContent>
              <GitRefSelect
                repoId={repoId}
                value={form.git_ref || ''}
                onValueChange={(v) => setForm({ ...form, git_ref: v ?? undefined })}
                className="w-full"
              />
            </FieldContent>
          </Field>
          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Dockerfile path</FieldLabel>
              <FieldContent>
                <Input
                  value={form.dockerfile_path || ''}
                  onChange={(e) => setForm({ ...form, dockerfile_path: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Build context</FieldLabel>
              <FieldContent>
                <Input
                  value={form.build_context || ''}
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
                value={form.image_name || ''}
                onChange={(e) => setForm({ ...form, image_name: e.target.value })}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel>Container registry *</FieldLabel>
            <FieldContent>
              <Combobox
                value={form.registry_id}
                onValueChange={(v) => setForm({ ...form, registry_id: v ?? undefined })}
                itemToStringLabel={(id) => registries?.find((r) => r.id === id)?.name ?? id ?? ""}
              >
                <ComboboxInput placeholder="Select registry" />
                <ComboboxContent>
                  <ComboboxList>
                    {registries?.map((r) => (
                      <ComboboxItem key={r.id} value={r.id}>
                        {/* {r.name} ({registryProviderLabels[r.provider]}) */}
                        <Item size="xs" className="p-0">
                          <ItemContent>
                            <ItemTitle>{r.name}</ItemTitle>
                            <ItemDescription>{registryProviderLabels[r.provider]}</ItemDescription>
                          </ItemContent>
                        </Item>
                      </ComboboxItem>
                    ))}
                  </ComboboxList>
                </ComboboxContent>
              </Combobox>
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel>Build args (optional)</FieldLabel>
            <FieldContent>
              <Input
                placeholder='KEY1=val1,KEY2=val2 or {"KEY1":"val1"}'
                value={form.build_args || ''}
                onChange={(e) => setForm({ ...form, build_args: e.target.value })}
              />
            </FieldContent>
          </Field>
          <div className="flex items-center gap-2">
            <Checkbox
              id="edit-webhook-enabled"
              checked={form.webhook_enabled ?? false}
              onCheckedChange={(v) => setForm({ ...form, webhook_enabled: v === true })}
            />
            <label htmlFor="edit-webhook-enabled" className="cursor-pointer">
              Auto build on webhook
            </label>
          </div>
          <div className="flex items-center gap-2">
            <Checkbox
              id="edit-auto-deploy"
              checked={form.auto_deploy ?? false}
              onCheckedChange={(v) => setForm({ ...form, auto_deploy: v === true })}
            />
            <label htmlFor="edit-auto-deploy" className="cursor-pointer">
              Auto deploy after build
            </label>
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
