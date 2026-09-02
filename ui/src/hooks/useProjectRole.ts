import { useQuery } from '@tanstack/react-query'
import { projectsApi, type ProjectCapabilitiesResponse, type ProjectRole } from '@/api/projects'
import { useAuthStore } from '@/stores/auth'
import { useProjectStore } from '@/stores/project'

export function useProjectCapabilities(projectIdOverride?: string | null): ProjectCapabilitiesResponse | null {
  const user = useAuthStore((state) => state.user)
  const activeProjectIdFromStore = useProjectStore((state) => state.activeProjectId)
  const activeProjectId = projectIdOverride === undefined
    ? activeProjectIdFromStore
    : projectIdOverride

  const { data, isError } = useQuery({
    queryKey: ['project-capabilities', activeProjectId, user?.id, user?.role],
    queryFn: () => projectsApi.getCapabilities(activeProjectId!),
    enabled: !!activeProjectId && !!user,
    staleTime: 60 * 1000,
  })

  // React Query keeps the previous value when a background refetch fails.
  // Treat that stale capability as unknown so callers fail closed.
  return isError ? null : data ?? null
}

// Returns null while access is unknown or the request failed. Consumers must
// treat null as read-only so permissions fail closed.
export function useProjectRole(projectIdOverride?: string | null): ProjectRole | null {
  const capabilities = useProjectCapabilities(projectIdOverride)

  return capabilities?.project_role ?? null
}
