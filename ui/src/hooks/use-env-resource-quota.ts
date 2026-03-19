import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"

import { envsApi, type UpdateResourceQuotaRequest } from "@/api/envs"

const RESOURCE_QUOTA_QUERY_KEY = (envId: string) => ["env-resource-quota", envId] as const

export function useEnvResourceQuota(envId: string) {
  return useQuery({
    queryKey: RESOURCE_QUOTA_QUERY_KEY(envId),
    queryFn: () => envsApi.getResourceQuota(envId),
    enabled: !!envId,
  })
}

export function useUpdateEnvResourceQuotaMutation(envId: string) {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: UpdateResourceQuotaRequest) => envsApi.updateResourceQuota(envId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: RESOURCE_QUOTA_QUERY_KEY(envId) })
      toast.success("Resource quota updated")
    },
    onError: (error: any) => {
      toast.error("Failed to update resource quota", {
        description: error.response?.data?.error || "An unknown error occurred",
      })
    },
  })
}
