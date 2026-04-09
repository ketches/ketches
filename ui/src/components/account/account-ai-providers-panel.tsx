import { Copy, Edit2, Plus, Sparkles, Trash2 } from "lucide-react"
import * as React from "react"

import { usersApi, type UpsertMyAiProviderRequest } from "@/api/users"
import { DataTable } from "@/components/data-table/data-table"
import { ColorBadge } from "@/components/shared/color-badge"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { Checkbox } from "@/components/ui/checkbox"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Field, FieldContent, FieldLabel } from "@/components/ui/field"
import { Input } from "@/components/ui/input"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { type ColumnDef } from "@tanstack/react-table"
import { toast } from "sonner"

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
  const { data: providers = [], isLoading, refetch } = useQuery({
    queryKey: ["users", "me", "self", "ai-providers"],
    queryFn: usersApi.listMyAiProviders,
    enabled: true,
  })
  const createMutation = useMutation({
    mutationFn: usersApi.createMyAiProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me", "self", "ai-providers"] })
    },
  })
  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: Parameters<typeof usersApi.updateMyAiProvider>[1] }) => usersApi.updateMyAiProvider(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me", "self", "ai-providers"] })
    },
  })
  const deleteMutation = useMutation({
    mutationFn: usersApi.deleteMyAiProvider,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["users", "me", "self", "ai-providers"] })
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

  const providerColumns: ColumnDef<(typeof providers)[number]>[] = [
    {
      accessorKey: "display_name",
      header: "Provider",
      cell: ({ row }) => (
        <div className="flex flex-col">
          <span className="font-medium">{row.original.display_name}</span>
          <span className="text-xs text-muted-foreground">{row.original.provider_key}</span>
        </div>
      ),
    },
    {
      accessorKey: "base_url",
      header: "Base URL",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="font-mono text-xs text-muted-foreground truncate block max-w-100">
            {row.original.base_url}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/row:opacity-100 transition-opacity"
            onClick={() => {
              navigator.clipboard.writeText(row.original.base_url)
              toast.success("Base URL copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      accessorKey: "default_model_profile_key",
      header: "Default Model",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <span className="text-xs text-muted-foreground">
            {row.original.default_model_profile_key}
          </span>
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            className="opacity-0 group-hover/row:opacity-100 transition-opacity"
            onClick={() => {
              navigator.clipboard.writeText(row.original.default_model_profile_key)
              toast.success("Default model copied to clipboard")
            }}
          >
            <Copy />
          </Button>
        </div>
      ),
    },
    {
      id: "status",
      header: "Status",
      cell: ({ row }) => (
        <div className="flex items-center gap-2">
          <ColorBadge color={row.original.enabled ? "green" : "gray"}>
            {row.original.enabled ? "Enabled" : "Disabled"}
          </ColorBadge>
          {row.original.is_default ? (
            <ColorBadge color="blue">
              Default
            </ColorBadge>
          ) : null}
        </div>
      ),
    },
    {
      id: "actions",
      header: () => <div className="text-right">Actions</div>,
      cell: ({ row }) => (
        <div className="flex items-center justify-end gap-1">
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => handleEditProvider(row.original.id)}
                />
              }
            >
              <div className="flex items-center">
                <Edit2 />
                <span className="sr-only">Edit</span>
              </div>
            </TooltipTrigger>
            <TooltipContent>Edit provider</TooltipContent>
          </Tooltip>
          <Tooltip>
            <TooltipTrigger
              delay={200}
              render={
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  className="text-destructive hover:text-destructive hover:bg-destructive/10"
                  onClick={() => void handleDeleteProvider(row.original.id)}
                />
              }
            >
              <div className="flex items-center">
                <Trash2 />
                <span className="sr-only">Delete</span>
              </div>
            </TooltipTrigger>
            <TooltipContent>Delete provider</TooltipContent>
          </Tooltip>
        </div>
      ),
    },
  ]

  return (
    <div className="space-y-4">
      <DataTable
        columns={providerColumns}
        data={providers}
        sourceDataCount={providers.length}
        isLoading={isLoading}
        searchKey="display_name"
        searchPlaceholder="Filter providers..."
        sourceEmptyContent={(
          <EmptyState
            title="No AI providers yet"
            description="Add your first provider to make personal models available in Builder."
            icon={Sparkles}
            actionText="Add Provider"
            onAction={handleAddProvider}
            actionIcon={Plus}
          />
        )}
        useStandaloneEmptyState
        rightToolbar={() => (
          <Button type="button" onClick={handleAddProvider}>
            <Plus className="h-4 w-4" />
            Add Provider
          </Button>
        )}
        onRefresh={() => refetch()}
      />

      <Dialog open={formOpen} onOpenChange={setFormOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>{editingProviderId ? "Edit AI Provider" : "Add AI Provider"}</DialogTitle>
            <DialogDescription>
              Configure a personal AI provider for Builder sessions and future AI workflows.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4">
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
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setFormOpen(false)}>
              Cancel
            </Button>
            <Button type="button" onClick={() => void handleSaveProvider()}>
              {editingProviderId ? "Update Provider" : "Save Provider"}
            </Button>
          </DialogFooter>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
