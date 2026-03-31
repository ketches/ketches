import { Plus, Sparkles } from "lucide-react"
import * as React from "react"

import { usersApi, type UpsertMyAiProviderRequest } from "@/api/users"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

const emptyForm: UpsertMyAiProviderRequest = {
  provider_key: "",
  display_name: "",
  base_url: "",
  api_key: "",
  default_model_profile_key: "",
  enabled: true,
  is_default: false,
}

export function AccountAiProvidersPanel() {
  const queryClient = useQueryClient()
  const [editingProviderId, setEditingProviderId] = React.useState<string | null>(null)
  const [formOpen, setFormOpen] = React.useState(false)
  const [formData, setFormData] = React.useState<UpsertMyAiProviderRequest>(emptyForm)
  const { data: providers = [] } = useQuery({
    queryKey: ["users", "me", "ai-providers"],
    queryFn: usersApi.listMyAiProviders,
  })
  const createMutation = useMutation({
    mutationFn: usersApi.createMyAiProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me", "ai-providers"] })
    },
  })
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof usersApi.updateMyAiProvider>[1] }) => usersApi.updateMyAiProvider(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me", "ai-providers"] })
    },
  })
  const deleteMutation = useMutation({
    mutationFn: usersApi.deleteMyAiProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me", "ai-providers"] })
    },
  })

  const handleAddProvider = () => {
    setEditingProviderId(null)
    setFormData(emptyForm)
    setFormOpen(true)
  }

  const handleEditProvider = (providerId: string) => {
    const provider = providers.find((item) => item.id === providerId)
    if (!provider) {
      return
    }

    setEditingProviderId(provider.id)
    setFormData({
      provider_key: provider.provider_key,
      display_name: provider.display_name,
      base_url: provider.base_url,
      api_key: "",
      default_model_profile_key: provider.default_model_profile_key,
      enabled: provider.enabled,
      is_default: provider.is_default,
    })
    setFormOpen(true)
  }

  const handleSaveProvider = async () => {
    if (editingProviderId) {
      await updateMutation.mutateAsync({ id: editingProviderId, data: formData })
    } else {
      await createMutation.mutateAsync(formData)
    }
    setFormOpen(false)
    setEditingProviderId(null)
    setFormData(emptyForm)
  }

  const handleDeleteProvider = async (providerId: string) => {
    await deleteMutation.mutateAsync(providerId)
  }

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-sm font-medium">Personal AI providers</h2>
        <p className="text-sm text-muted-foreground">
          Configure your personal AI providers for Builder sessions and future AI-powered workflows.
        </p>
      </div>

      {providers.length === 0 ? (
        <div className="space-y-4 rounded-lg border border-dashed bg-muted/20 p-4">
          <EmptyState
            title="No personal AI providers configured yet"
            description="Add your first provider to make personal models available in Builder."
            icon={Sparkles}
            border={false}
          />
          <div className="flex justify-start">
            <Button type="button" onClick={handleAddProvider}>
              <Plus className="h-4 w-4" />
              Add provider
            </Button>
          </div>
        </div>
      ) : (
        <div className="space-y-3 rounded-lg border bg-background p-4">
          <div className="flex justify-end">
            <Button type="button" onClick={handleAddProvider}>
              <Plus className="h-4 w-4" />
              Add provider
            </Button>
          </div>
          {providers.map((provider) => (
            <div key={provider.id} className="flex items-center justify-between rounded-md border p-3">
              <div className="space-y-1">
                <div className="text-sm font-medium">{provider.display_name}</div>
                <div className="text-xs text-muted-foreground">
                  {provider.provider_key} · {provider.default_model_profile_key}
                </div>
                <div className="flex items-center gap-2 text-xs text-muted-foreground">
                  <span>{provider.enabled ? "Enabled" : "Disabled"}</span>
                  {provider.is_default ? <span>Default</span> : null}
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button type="button" variant="outline" onClick={() => handleEditProvider(provider.id)}>
                  Edit
                </Button>
                <Button type="button" variant="outline" onClick={() => void handleDeleteProvider(provider.id)}>
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      {formOpen ? (
        <div className="space-y-4 rounded-lg border bg-background p-4">
          <Field>
            <FieldLabel htmlFor="account-provider-key">Provider key</FieldLabel>
            <FieldContent>
              <Input
                id="account-provider-key"
                name="provider_key"
                value={formData.provider_key}
                onInput={(event) => setFormData((prev) => ({ ...prev, provider_key: (event.target as HTMLInputElement).value }))}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel htmlFor="account-provider-display-name">Display name</FieldLabel>
            <FieldContent>
              <Input
                id="account-provider-display-name"
                name="display_name"
                value={formData.display_name}
                onInput={(event) => setFormData((prev) => ({ ...prev, display_name: (event.target as HTMLInputElement).value }))}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel htmlFor="account-provider-base-url">Base URL</FieldLabel>
            <FieldContent>
              <Input
                id="account-provider-base-url"
                name="base_url"
                value={formData.base_url}
                onInput={(event) => setFormData((prev) => ({ ...prev, base_url: (event.target as HTMLInputElement).value }))}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel htmlFor="account-provider-api-key">API key</FieldLabel>
            <FieldContent>
              <Input
                id="account-provider-api-key"
                name="api_key"
                value={formData.api_key}
                onInput={(event) => setFormData((prev) => ({ ...prev, api_key: (event.target as HTMLInputElement).value }))}
              />
            </FieldContent>
          </Field>
          <Field>
            <FieldLabel htmlFor="account-provider-model-profile">Default model profile key</FieldLabel>
            <FieldContent>
              <Input
                id="account-provider-model-profile"
                name="default_model_profile_key"
                value={formData.default_model_profile_key}
                onInput={(event) => setFormData((prev) => ({ ...prev, default_model_profile_key: (event.target as HTMLInputElement).value }))}
              />
            </FieldContent>
          </Field>
          <div className="flex items-center gap-3">
            <Checkbox
              id="account-provider-enabled"
              name="enabled"
              checked={formData.enabled}
              onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, enabled: checked === true }))}
            />
            <label htmlFor="account-provider-enabled" className="cursor-pointer text-sm">
              Enabled
            </label>
          </div>
          <div className="flex items-center gap-3">
            <Checkbox
              id="account-provider-default"
              name="is_default"
              checked={formData.is_default}
              onCheckedChange={(checked) => setFormData((prev) => ({ ...prev, is_default: checked === true }))}
            />
            <label htmlFor="account-provider-default" className="cursor-pointer text-sm">
              Set as default
            </label>
          </div>
          <div className="flex items-center justify-end gap-2">
            <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>
              Cancel
            </Button>
            <Button type="button" onClick={() => void handleSaveProvider()}>
              {editingProviderId ? "Update provider" : "Save provider"}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  )
}
