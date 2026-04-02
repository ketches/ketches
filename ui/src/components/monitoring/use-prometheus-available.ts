import { useQuery } from "@tanstack/react-query"
import { clustersApi } from "@/api/clusters"

export function usePrometheusAvailable(clusterId: string, projectId?: string) {
  const { data, isLoading } = useQuery({
    queryKey: ["cluster-prometheus-available", clusterId, projectId],
    queryFn: async () => {
      const integrations = await clustersApi.listIntegrations(clusterId, projectId)
      return integrations.some(i => i.integration_type === "prometheus")
    },
    enabled: !!clusterId,
    staleTime: 5 * 60 * 1000, // cache for 5 minutes
  })

  return {
    available: data,
    isLoading,
  }
}
