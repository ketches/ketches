import * as React from "react"
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import {
  platformSettingsApi,
  type PlatformBranding,
  type UpdatePlatformBrandingRequest,
} from "@/api/platform-settings"

export const PLATFORM_BRANDING_QUERY_KEY = ["platform-settings", "branding"] as const
const PLATFORM_BRANDING_STORAGE_KEY = "platform-branding"

interface CachedPlatformBranding {
  name: string
}

function readCachedPlatformBranding(): CachedPlatformBranding | null {
  if (typeof window === "undefined") {
    return null
  }

  try {
    const rawValue = window.localStorage.getItem(PLATFORM_BRANDING_STORAGE_KEY)
    if (!rawValue) {
      return null
    }

    const parsed = JSON.parse(rawValue) as CachedPlatformBranding
    if (typeof parsed.name !== "string" || !parsed.name.trim()) {
      return null
    }

    return {
      name: parsed.name.trim(),
    }
  } catch {
    return null
  }
}

function writeCachedPlatformBranding(cache: CachedPlatformBranding) {
  if (typeof window === "undefined") {
    return
  }

  window.localStorage.setItem(PLATFORM_BRANDING_STORAGE_KEY, JSON.stringify(cache))
}

function toCachedBrandingData(cache: CachedPlatformBranding): PlatformBranding {
  return {
    name: cache.name,
  }
}

export function usePlatformBranding() {
  const [cachedBranding, setCachedBranding] = React.useState<CachedPlatformBranding | null>(() =>
    readCachedPlatformBranding()
  )

  const brandingQuery = useQuery({
    queryKey: PLATFORM_BRANDING_QUERY_KEY,
    queryFn: platformSettingsApi.getBranding,
    initialData: cachedBranding ? toCachedBrandingData(cachedBranding) : undefined,
  })

  React.useEffect(() => {
    if (!brandingQuery.data) {
      return
    }

    setCachedBranding((current) => {
      const nextCache = {
        name: brandingQuery.data.name,
      }

      if (current?.name === nextCache.name) {
        return current
      }

      writeCachedPlatformBranding(nextCache)
      return nextCache
    })
  }, [brandingQuery.data])

  return {
    ...brandingQuery,
    data: brandingQuery.data ?? (cachedBranding ? toCachedBrandingData(cachedBranding) : brandingQuery.data),
  }
}

export function useUpdatePlatformBrandingMutation() {
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (data: UpdatePlatformBrandingRequest) => platformSettingsApi.updateBranding(data),
    onSuccess: async (branding) => {
      writeCachedPlatformBranding({
        name: branding.name,
      })

      queryClient.setQueryData(PLATFORM_BRANDING_QUERY_KEY, branding)
      queryClient.invalidateQueries({ queryKey: PLATFORM_BRANDING_QUERY_KEY })
    },
  })
}
