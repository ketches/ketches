import { useMutation, useQueryClient } from "@tanstack/react-query"
import { Link2, Loader2 } from "lucide-react"
import * as React from "react"
import { toast } from "sonner"

import {
  containerRegistriesApi,
  registryProviderLabels,
  type ContainerRegistry,
  type RegistryProvider as ContainerRegistryProvider,
  type CreateContainerRegistryRequest,
} from "@/api/container-registries"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Textarea } from "@/components/ui/textarea"
import type { AxiosError } from "axios"

interface ContainerRegistryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  scope: 'cluster' | 'project'
  scopeId: string
  registry?: ContainerRegistry | null
}

const providers: ContainerRegistryProvider[] = ['dockerhub', 'harbor', 'ghcr', 'acr', 'ecr', 'custom']

const providerEndpointDefaults: Partial<Record<ContainerRegistryProvider, string>> = {
  dockerhub: 'https://index.docker.io/v1/',
  ghcr: 'https://ghcr.io',
}

export function ContainerRegistryDialog({ open, onOpenChange, scope, scopeId, registry }: ContainerRegistryDialogProps) {
  const queryClient = useQueryClient()
  const isEdit = !!registry

  const [form, setForm] = React.useState<CreateContainerRegistryRequest>({
    name: '',
    provider: 'dockerhub',
    endpoint: providerEndpointDefaults.dockerhub ?? '',
    namespace: '',
    username: '',
    password: '',
    skip_tls_verify: false,
    is_default: false,
    enabled: true,
    description: '',
  })

  React.useEffect(() => {
    if (registry) {
      setForm({
        name: registry.name,
        provider: registry.provider,
        endpoint: registry.endpoint,
        skip_tls_verify: registry.skip_tls_verify ?? false,
        namespace: registry.namespace || '',
        username: registry.username || '',
        password: registry.password || '',
        is_default: registry.is_default,
        enabled: registry.enabled,
        description: registry.description || '',
      })
    } else {
      setForm({
        name: '',
        provider: 'dockerhub',
        endpoint: providerEndpointDefaults.dockerhub ?? '',
        namespace: '',
        username: '',
        password: '',
        skip_tls_verify: false,
        is_default: false,
        enabled: true,
        description: '',
      })
    }
  }, [registry, open])

  const createMutation = useMutation({
    mutationFn: (data: CreateContainerRegistryRequest) => {
      if (scope === 'cluster') {
        return containerRegistriesApi.createForCluster(scopeId, data)
      }
      return containerRegistriesApi.createForProject(scopeId, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['registries', scope, scopeId] })
      onOpenChange(false)
      toast.success('Registry created')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to create registry')
    },
  })

  const updateMutation = useMutation({
    mutationFn: (data: CreateContainerRegistryRequest) => {
      return containerRegistriesApi.update(registry!.id, data)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['registries', scope, scopeId] })
      onOpenChange(false)
      toast.success('Registry updated')
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Failed to update registry')
    },
  })

  const testMutation = useMutation({
    mutationFn: () => {
      const id = registry?.id || 'test'
      return containerRegistriesApi.test(id, {
        provider: form.provider,
        endpoint: form.endpoint,
        skip_tls_verify: form.skip_tls_verify,
        username: form.username,
        password: form.password,
      })
    },
    onSuccess: (result) => {
      if (result.success) {
        toast.success(result.message)
      } else {
        toast.error(result.message)
      }
    },
    onError: (err: AxiosError<{ error: string }>) => {
      toast.error(err?.response?.data?.error || 'Connection test failed')
    },
  })

  const handleSubmit = () => {
    if (!form.name || !form.provider || !form.endpoint) {
      toast.error('Please fill in required fields')
      return
    }
    if (isEdit) {
      updateMutation.mutate(form)
    } else {
      createMutation.mutate(form)
    }
  }

  const handleProviderChange = (provider: ContainerRegistryProvider) => {
    const endpoint = providerEndpointDefaults[provider] ?? ''
    setForm({ ...form, provider, endpoint })
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-140 max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Edit' : 'Add'} Image Registry</DialogTitle>
          <DialogDescription>
            {scope === 'cluster' ? 'Cluster-level registry available to all projects' : 'Project-level registry'}
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Name *</FieldLabel>
              <FieldContent>
                <Input
                  placeholder="My Container Registry"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Provider *</FieldLabel>
              <FieldContent>
                <Select
                  value={form.provider}
                  onValueChange={(v) => handleProviderChange(v as ContainerRegistryProvider)}
                  items={providers.map((p) => ({ value: p, label: registryProviderLabels[p] }))}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {providers.map((p) => (
                      <SelectItem key={p} value={p}>{registryProviderLabels[p]}</SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </FieldContent>
            </Field>
          </div>

          <Field>
            <FieldLabel>Server *</FieldLabel>
            <FieldContent>
              <Input
                placeholder="https://registry.example.com"
                value={form.endpoint}
                onChange={(e) => setForm({ ...form, endpoint: e.target.value })}
              />
            </FieldContent>
          </Field>

          {(form.provider === 'harbor' || form.provider === 'custom') && (
            <>
              <div className="flex items-center gap-2">
                <Checkbox
                  id="skip-tls"
                  checked={form.skip_tls_verify ?? false}
                  onCheckedChange={(v) => setForm({ ...form, skip_tls_verify: v === true })}
                />
                <label htmlFor="skip-tls" className="cursor-pointer">
                  Skip TLS verification
                </label>
              </div>
              <p className="text-[11px] text-muted-foreground -mt-4">
                Use for self-hosted registries (e.g. Harbor) without a valid TLS certificate. Default is TLS verified.
              </p>
            </>
          )}

          <Field>
            <FieldLabel>Namespace / Organization</FieldLabel>
            <FieldContent>
              <Input
                placeholder="my-org"
                value={form.namespace}
                onChange={(e) => setForm({ ...form, namespace: e.target.value })}
              />
            </FieldContent>
          </Field>

          <div className="grid grid-cols-2 gap-4">
            <Field>
              <FieldLabel>Username</FieldLabel>
              <FieldContent>
                <Input
                  value={form.username}
                  onChange={(e) => setForm({ ...form, username: e.target.value })}
                />
              </FieldContent>
            </Field>
            <Field>
              <FieldLabel>Password</FieldLabel>
              <FieldContent>
                <Input
                  type="password"
                  value={form.password}
                  onChange={(e) => setForm({ ...form, password: e.target.value })}
                />
              </FieldContent>
            </Field>
          </div>

          <Field>
            <FieldLabel>Description</FieldLabel>
            <FieldContent>
              <Textarea
                value={form.description}
                onChange={(e) => setForm({ ...form, description: e.target.value })}
                rows={2}
              />
            </FieldContent>
          </Field>

          <div className="flex items-center gap-2">
            <Checkbox
              id="is-default"
              checked={form.is_default}
              onCheckedChange={(v) => setForm({ ...form, is_default: v === true })}
            />
            <label htmlFor="is-default" className="cursor-pointer">
              Default Registry
            </label>
          </div>

          <div className="flex items-center gap-2">
            <Checkbox
              id="is-enabled"
              checked={form.enabled}
              onCheckedChange={(v) => setForm({ ...form, enabled: v === true })}
            />
            <label htmlFor="is-enabled" className="cursor-pointer">
              Enabled
            </label>
          </div>
        </div>

        <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-between">
          <Button variant="outline" onClick={() => testMutation.mutate()} disabled={testMutation.isPending}>
            {testMutation.isPending ? <Loader2 className="animate-spin" /> : null}
            <Link2 />
            Test Connection
          </Button>
          <Button onClick={handleSubmit} disabled={isPending}>
            {isPending ? <Loader2 className="animate-spin" /> : null}
            {isEdit ? 'Update' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
