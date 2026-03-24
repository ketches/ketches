import { Plus, Sparkles } from "lucide-react"

import { projectsApi } from "@/api/projects"
import { EmptyState } from "@/components/shared/empty-state"
import { Button } from "@/components/ui/button"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

interface ProjectAiProvidersPanelProps {
  projectId: string
}

export function ProjectAiProvidersPanel({ projectId }: ProjectAiProvidersPanelProps) {
  const queryClient = useQueryClient()
  const { data: providers = [] } = useQuery({
    queryKey: ["projects", projectId, "ai-providers"],
    queryFn: () => projectsApi.listAiProviders(projectId),
    enabled: !!projectId,
  })
  const createMutation = useMutation({
    mutationFn: (data: Parameters<typeof projectsApi.createAiProvider>[1]) => projectsApi.createAiProvider(projectId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects", projectId, "ai-providers"] })
    },
  })
  const updateMutation = useMutation({
    mutationFn: ({ providerId, data }: { providerId: string; data: Parameters<typeof projectsApi.updateAiProvider>[2] }) =>
      projectsApi.updateAiProvider(projectId, providerId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["projects", projectId, "ai-providers"] })
    },
  })

  const handleAddProvider = async () => {
    await createMutation.mutateAsync({
      provider_key: "openai-project",
      display_name: "OpenAI Shared",
      base_url: "https://api.openai.com",
      api_key: "shared-key",
      default_model_profile_key: "gpt-4.1",
      enabled: true,
    })
  }

  const handleEditProvider = async (providerId: string) => {
    await updateMutation.mutateAsync({
      providerId,
      data: {
        provider_key: "anthropic-project",
        display_name: "Anthropic Shared Updated",
        base_url: "https://api.anthropic.com",
        api_key: "updated-shared-key",
        default_model_profile_key: "claude-sonnet-4",
        enabled: true,
      },
    })
  }

  return (
    <div className="space-y-4">
      <div className="space-y-1">
        <h2 className="text-sm font-medium">Project AI providers</h2>
        <p className="text-sm text-muted-foreground">
          Configure project-level AI providers available to Builder conversations in this project.
        </p>
      </div>

      {providers.length === 0 ? (
        <div className="space-y-4 rounded-lg border border-dashed bg-muted/20 p-4">
          <EmptyState
            title="No project AI providers configured yet"
            description="Add a project-level provider to make shared models available in Builder."
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
