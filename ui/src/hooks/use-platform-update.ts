import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  platformUpdateApi,
  type CheckPlatformUpdateRequest,
  type PlatformUpdateConfig,
  type TriggerPlatformRolloutRequest,
} from "@/api/platform-update"

export const PLATFORM_UPDATE_CONFIG_QUERY_KEY = ["platform-update", "config"] as const
export const PLATFORM_UPDATE_STATUS_QUERY_KEY = ["platform-update", "status"] as const

export function usePlatformUpdateConfigQuery(enabled: boolean = true) {
  return useQuery({
    queryKey: PLATFORM_UPDATE_CONFIG_QUERY_KEY,
    queryFn: platformUpdateApi.getConfig,
    enabled,
  })
}

export function useUpdatePlatformUpdateConfigMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: PlatformUpdateConfig) => platformUpdateApi.updateConfig(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PLATFORM_UPDATE_CONFIG_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: PLATFORM_UPDATE_STATUS_QUERY_KEY })
    },
  })
}

export function usePlatformUpdateStatusQuery(enabled: boolean = true) {
  return useQuery({
    queryKey: PLATFORM_UPDATE_STATUS_QUERY_KEY,
    queryFn: platformUpdateApi.getStatus,
    enabled,
  })
}

export function useCheckPlatformUpdateMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: CheckPlatformUpdateRequest) => platformUpdateApi.check(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PLATFORM_UPDATE_STATUS_QUERY_KEY })
    },
  })
}

export function useTriggerPlatformRolloutMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: TriggerPlatformRolloutRequest) => platformUpdateApi.triggerRollout(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: PLATFORM_UPDATE_STATUS_QUERY_KEY })
    },
  })
}
