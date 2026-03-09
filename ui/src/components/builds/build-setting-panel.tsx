import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { GitBranch, Loader2, Save, TestTube2, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { buildSettingsApi, type UpsertBuildSettingRequest } from "@/api/build-settings"
import { registryProviderLabels, type ContainerRegistry } from "@/api/container-registries"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger
} from "@/components/ui/tooltip"

import {
  Combobox,
  ComboboxContent,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox"
import { Textarea } from "@/components/ui/textarea"
import type { AxiosError } from "axios"

interface BuildSettingPanelProps {
  appId: string
}

export function BuildSettingPanel({ appId }: BuildSettingPanelProps) {
  const queryClient = useQueryClient()

  const { data: setting, isLoading } = useQuery({
    queryKey: ['build-setting', appId],
    queryFn: () => buildSettingsApi.get(appId),
    retry: false,
  })

  const { data: registries } = useQuery({
    queryKey: ['available-registries', appId],
    queryFn: () => buildSettingsApi.listAvailableRegistries(appId),
  })

  const [form, setForm] = React.useState<UpsertBuildSettingRequest>({
    git_repo_url: '',
    git_ref: 'main',
    git_username: '',
    git_password: '',
    dockerfile_path: 'Dockerfile',
    build_context: '.',
    image_name: '',
    registry_id: '',
    build_args: '',
  })

  React.useEffect(() => {
    if (setting) {
      setForm({
        git_repo_url: setting.git_repo_url,
        git_ref: setting.git_ref || 'main',
        git_username: setting.git_username || '',
        git_password: '',
        dockerfile_path: setting.dockerfile_path || 'Dockerfile',
        build_context: setting.build_context || '.',
        image_name: setting.image_name,
        registry_id: setting.registry_id,
        build_args: setting.build_args || '',
      })
    }
  }, [setting])

  const saveMutation = useMutation({
    mutationFn: (data: UpsertBuildSettingRequest) => buildSettingsApi.upsert(appId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['build-setting', appId] })
      toast.success('Build setting saved')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to save build setting')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => buildSettingsApi.delete(appId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['build-setting', appId] })
      setForm({
        git_repo_url: '', git_ref: 'main', git_username: '', git_password: '',
        dockerfile_path: 'Dockerfile', build_context: '.', image_name: '',
        registry_id: '', build_args: '',
      })
      toast.success('Build setting deleted')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to delete build setting')
    },
  })

  const testGitMutation = useMutation({
    mutationFn: () => buildSettingsApi.testGit(appId, {
      git_repo_url: form.git_repo_url,
      git_ref: form.git_ref,
      git_username: form.git_username,
      git_password: form.git_password,
    }),
    onSuccess: (result) => {
      if (result.success) {
        toast.success(result.message)
      } else {
        toast.error(result.message)
      }
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Git connection test failed')
    },
  })

  const handleSave = () => {
    if (!form.git_repo_url || !form.image_name || !form.registry_id) {
      toast.error('Please fill in required fields: Git URL, Image Name, Registry')
      return
    }
    saveMutation.mutate(form)
  }

  if (isLoading) {
    return <div className="flex items-center justify-center p-8"><Loader2 className="h-6 w-6 animate-spin" /></div>
  }

  return (
    <Card>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div>
            <CardTitle className="text-sm flex items-center gap-2">
              <GitBranch className="h-4 w-4" />
              Build Setting
            </CardTitle>
            <CardDescription>Configure Git source and build settings for this application</CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {setting && (
              <Button variant="outline" size="sm" onClick={() => deleteMutation.mutate()} disabled={deleteMutation.isPending}>
                <Trash2 />
                Delete
              </Button>
            )}
            <Button size="sm" onClick={handleSave} disabled={saveMutation.isPending}>
              {saveMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <Save />}
              Save
            </Button>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Git Source */}
        <div className="space-y-4">
          <h4 className="text-sm font-medium">Git Source</h4>
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <Field>
              <FieldLabel>Repository URL *</FieldLabel>
              <FieldContent>
                <div className="flex gap-2">
                  <Input
                    placeholder="https://github.com/user/repo.git"
                    value={form.git_repo_url}
                    onChange={(e) => setForm({ ...form, git_repo_url: e.target.value })}
                  />
                  <Tooltip>
                    <TooltipTrigger
                      delay={200}
                      render={
                        <Button
                          variant="outline"
                          size="icon"
                          onClick={() => testGitMutation.mutate()}
                          disabled={testGitMutation.isPending}
                        />
                      }
                    >
                      {testGitMutation.isPending ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <TestTube2 />
                      )}
                    </TooltipTrigger>
                    <TooltipContent>Test connection</TooltipContent>
                  </Tooltip>
                </div>
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Git Branch / Tag</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="main"
                  value={form.git_ref}
                  onChange={(e) => setForm({ ...form, git_ref: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Git Username</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="Username or token name"
                  value={form.git_username}
                  onChange={(e) => setForm({ ...form, git_username: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Git Password / Token</FieldLabel>
              <FieldContent>
                <Input
                  type="password"
                  autoComplete="new-password"
                  placeholder="Password or personal access token"
                  value={form.git_password}
                  onChange={(e) => setForm({ ...form, git_password: e.target.value })}
                />
              </FieldContent>
            </Field>
          </div>
        </div>

        {/* Build Settings */}
        <div className="space-y-4">
          <h4 className="text-sm font-medium">Build Settings</h4>
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2">
            <Field>
              <FieldLabel>Dockerfile Path</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="Dockerfile"
                  value={form.dockerfile_path}
                  onChange={(e) => setForm({ ...form, dockerfile_path: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Build Context</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="."
                  value={form.build_context}
                  onChange={(e) => setForm({ ...form, build_context: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Image Name *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="my-app"
                  value={form.image_name}
                  onChange={(e) => setForm({ ...form, image_name: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Target Registry *</FieldLabel>
              <FieldContent>
                <Combobox
                  value={form.registry_id}
                  onValueChange={(v) => v && setForm({ ...form, registry_id: v })}
                  itemToStringLabel={(id) => registries?.find((r: ContainerRegistry) => r.id === id)?.name ?? id ?? ""}
                >
                  <ComboboxInput placeholder="Select a registry" />
                  <ComboboxContent>
                    <ComboboxList>
                      {registries?.map((r: ContainerRegistry) => (
                        <ComboboxItem key={r.id} value={r.id}>
                          {`${r.name} (${registryProviderLabels[r.provider]})`}
                        </ComboboxItem>
                      ))}
                    </ComboboxList>
                  </ComboboxContent>
                </Combobox>
              </FieldContent>
            </Field>
          </div>
          <Field>
            <FieldLabel>Build Arguments (JSON)</FieldLabel>
            <FieldContent>
              <Textarea
                placeholder='{"NODE_ENV": "production"}'
                value={form.build_args}
                onChange={(e) => setForm({ ...form, build_args: e.target.value })}
                rows={3}
              />
            </FieldContent>
          </Field>
        </div>
      </CardContent>
    </Card>
  )
}
