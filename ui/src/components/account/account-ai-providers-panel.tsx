import { Plus, Sparkles } from "lucide-react"

import { usersApi } from "@/api/users"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

export function AccountAiProvidersPanel() {
  const queryClient = useQueryClient()
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

  const handleAddProvider = async () => {
    await createMutation.mutateAsync({
      provider_key: "anthropic-user",
      display_name: "Anthropic Personal",
      base_url: "https://api.anthropic.com",
      api_key: "test-key",
      default_model_profile_key: "claude-sonnet-4",
      enabled: true,
    })
  }

  const handleEditProvider = async (providerId: string) => {
    await updateMutation.mutateAsync({
      id: providerId,
      data: {
        provider_key: "openai-user",
        display_name: "OpenAI Personal Updated",
        base_url: "https://api.openai.com",
        api_key: "updated-secret-key",
        default_model_profile_key: "gpt-4.1",
        enabled: true,
      },
    })
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
            <Button type="button" onClick={() => void handleAddProvider()}>
              <Plus className="h-4 w-4" />
              Add provider
            </Button>
          </div>
        </div>
      ) : (
        <div className="space-y-3 rounded-lg border bg-background p-4">
          <div className="flex justify-end">
            <Button type="button" onClick={() => void handleAddProvider()}>
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
              </div>
              <Button type="button" variant="outline" onClick={() => void handleEditProvider(provider.id)}>
                Edit
              </Button>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
