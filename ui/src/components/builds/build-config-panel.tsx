import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { GitBranch, Loader2, Save, TestTube2, Trash2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import { buildConfigsApi, type UpsertBuildConfigRequest } from "@/api/build-configs"
import { registryProviderLabels, type ContainerRegistry } from "@/api/container-registries"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldContent, FieldLabel, FieldTitle } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import type { AxiosError } from "axios"

interface BuildConfigPanelProps {
  appId: string
}

export function BuildConfigPanel({ appId }: BuildConfigPanelProps) {
  const queryClient = useQueryClient()

  const { data: config, isLoading } = useQuery({
    queryKey: ['build-config', appId],
    queryFn: () => buildConfigsApi.get(appId),
    retry: false,
  })

  const { data: registries } = useQuery({
    queryKey: ['available-registries', appId],
    queryFn: () => buildConfigsApi.listAvailableRegistries(appId),
  })

  const [form, setForm] = React.useState<UpsertBuildConfigRequest>({
    git_repo_url: '',
    git_ref: 'main',
    git_username: '',
    git_password: '',
    dockerfile_path: 'Dockerfile',
    build_context: '.',
    image_name: '',
    registry_id: '',
    build_args: '',
    auto_build: false,
    auto_deploy: false,
    webhook_enabled: false,
  })

  React.useEffect(() => {
    if (config) {
      setForm({
        git_repo_url: config.git_repo_url,
        git_ref: config.git_ref || 'main',
        git_username: config.git_username || '',
        git_password: '',
        dockerfile_path: config.dockerfile_path || 'Dockerfile',
        build_context: config.build_context || '.',
        image_name: config.image_name,
        registry_id: config.registry_id,
        build_args: config.build_args || '',
        auto_build: config.auto_build,
        auto_deploy: config.auto_deploy,
        webhook_enabled: config.webhook_enabled,
      })
    }
  }, [config])

  const saveMutation = useMutation({
    mutationFn: (data: UpsertBuildConfigRequest) => buildConfigsApi.upsert(appId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['build-config', appId] })
      toast.success('Build configuration saved')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to save build configuration')
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => buildConfigsApi.delete(appId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['build-config', appId] })
      setForm({
        git_repo_url: '', git_ref: 'main', git_username: '', git_password: '',
        dockerfile_path: 'Dockerfile', build_context: '.', image_name: '',
        registry_id: '', build_args: '', auto_build: false, auto_deploy: false, webhook_enabled: false,
      })
      toast.success('Build configuration deleted')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to delete build configuration')
    },
  })

  const testGitMutation = useMutation({
    mutationFn: () => buildConfigsApi.testGit(appId, {
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
              Build Configuration
            </CardTitle>
            <CardDescription>Configure Git source and build settings for this application</CardDescription>
          </div>
          <div className="flex items-center gap-2">
            {config && (
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
                  <Button variant="outline" size="icon" onClick={() => testGitMutation.mutate()} disabled={testGitMutation.isPending} title="Test connection">
                    {testGitMutation.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <TestTube2 />}
                  </Button>
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
                <Select
                  value={form.registry_id || undefined}
                  onValueChange={(v) => setForm({ ...form, registry_id: v ?? "" })}
                  items={
                    registries?.map((r: ContainerRegistry) => ({
                      value: r.id,
                      label: `${r.name} (${registryProviderLabels[r.provider]})`,
                    })) ?? []
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="Select a registry" />
                  </SelectTrigger>
                  <SelectContent>
                    {registries?.map((r: ContainerRegistry) => (
                      <SelectItem key={r.id} value={r.id}>
                        {r.name} ({registryProviderLabels[r.provider]})
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
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

        {/* Behavior */}
        <div className="space-y-4">
          <h4 className="text-sm font-medium">Behavior</h4>
          <div className="flex flex-col gap-4">
            <div className="flex items-center gap-2">
              <Checkbox
                id="auto-build"
                checked={form.auto_build}
                onCheckedChange={(v) => setForm({ ...form, auto_build: v === true })}
              />
              <div className="grid gap-0.5">
                <label htmlFor="auto-build" className="text-sm font-medium leading-none cursor-pointer">Auto Build</label>
                <p className="text-[11px] text-muted-foreground">Automatically build when webhook is triggered</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="auto-deploy"
                checked={form.auto_deploy}
                onCheckedChange={(v) => setForm({ ...form, auto_deploy: v === true })}
              />
              <div className="grid gap-0.5">
                <label htmlFor="auto-deploy" className="text-sm font-medium leading-none cursor-pointer">Auto Deploy</label>
                <p className="text-[11px] text-muted-foreground">Automatically deploy after a successful build</p>
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Checkbox
                id="webhook-enabled-switch"
                checked={form.webhook_enabled}
                onCheckedChange={(v) => setForm({ ...form, webhook_enabled: v === true })}
              />
              <div className="grid gap-0.5">
                <label htmlFor="webhook-enabled-switch" className="text-sm font-medium leading-none cursor-pointer">Webhook</label>
                <p className="text-[11px] text-muted-foreground">Enable webhook trigger for automated builds</p>
              </div>
            </div>
          </div>
        </div>

        {/* Webhook Info */}
        {config?.webhook_enabled && config?.webhook_secret && (
          <div className="space-y-4">
            <h4 className="text-sm font-medium">Webhook Information</h4>
            <div className="rounded-lg border p-4 space-y-3 bg-muted/50">
              <div className="space-y-1">
                <FieldTitle className="text-xs text-muted-foreground uppercase">Webhook URL</FieldTitle>
                <code className="block text-xs bg-background p-2 rounded border break-all">
                  {`${window.location.origin}/api/v1/webhooks/git/${appId}?secret=${config.webhook_secret}`}
                </code>
              </div>
              <div className="space-y-1">
                <FieldTitle className="text-xs text-muted-foreground uppercase">Secret</FieldTitle>
                <code className="block text-xs bg-background p-2 rounded border break-all font-mono">
                  {config.webhook_secret}
                </code>
              </div>
              <p className="text-xs text-muted-foreground">
                Add this URL as a webhook in your Git provider. Supports GitHub (X-Hub-Signature-256) and GitLab (X-Gitlab-Token) signature verification.
              </p>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
