import { codeRepositoriesApi, type BuildSetting, type UpdateBuildSettingRequest } from "@/api/code-repositories"
import { registryProviderLabels } from "@/api/container-registries"
import { GitRefSelect } from "@/components/code-repositories/git-ref-select"
import { Button } from "@/components/ui/button"
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

interface EditBuildSettingDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  repoId: string
  setting: BuildSetting | null
  onSuccess?: () => void
}

export function EditBuildSettingDialog({ open, onOpenChange, repoId, setting: setting, onSuccess }: EditBuildSettingDialogProps) {
  const queryClient = useQueryClient()
  const [form, setForm] = React.useState<UpdateBuildSettingRequest>({})

  React.useEffect(() => {
    if (setting) {
      setForm({
        name: setting.name,
        git_ref: setting.git_ref,
        dockerfile_path: setting.dockerfile_path,
        build_context: setting.build_context,
        image_name: setting.image_name,
        registry_id: setting.registry_id,
        build_args: setting.build_args,
      })
    }
  }, [setting, open])

  const { data: registries } = useQuery({
    queryKey: ['code-repository-registries', repoId],
    queryFn: () => codeRepositoriesApi.listContainerRegistries(repoId),
    enabled: open && !!repoId,
  })

  const updateMutation = useMutation({
    mutationFn: () => codeRepositoriesApi.updateBuildSetting(repoId, setting!.id, form),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['build-settings', repoId] })
      toast.success('Build setting updated')
      onOpenChange(false)
      onSuccess?.()
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to update build setting')
    },
  })

  const handleSubmit = () => {
    if (!form.name?.trim() || !form.image_name?.trim() || !form.registry_id) {
      toast.error('Name, image name, and registry are required')
      return
    }
    updateMutation.mutate()
  }

  if (!setting) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto gap-0">
        <DialogHeader>
          <DialogTitle>Edit Build Setting</DialogTitle>
          <DialogDescription>
            Update build setting for this repository.
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
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button onClick={handleSubmit} disabled={updateMutation.isPending}>
            {updateMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
            Save
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog >
  )
}
