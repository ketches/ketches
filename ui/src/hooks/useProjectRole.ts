// useProjectRole returns the current user's role in the active project.
// Admin system users always get 'owner'. Returns null if no active project or not loaded.
import { useQuery } from '@tanstack/react-query'
import { projectsApi, type ProjectRole } from '@/api/projects'
import { useAuthStore } from '@/stores/auth'
import { useProjectStore } from '@/stores/project'

export function useProjectRole(projectIdOverride?: string | null): ProjectRole | null {
  const user = useAuthStore((state) => state.user)
  const activeProjectIdFromStore = useProjectStore((state) => state.activeProjectId)
  const activeProjectId = projectIdOverride ?? activeProjectIdFromStore

  const isAdmin = user?.role === 'admin'

  // Only fetch members for non-admin users with an active project
  const { data } = useQuery({
    queryKey: ['project-members', activeProjectId],
    queryFn: () => projectsApi.listMembers(activeProjectId!),
    enabled: !isAdmin && !!activeProjectId && !!user,
    staleTime: 5 * 60 * 1000,
  })

  // Admin system role gets full owner-equivalent access
  if (isAdmin) {
    return 'owner'
  }

  if (!data || !user) return null

  const membership = data.items.find((m) => m.user_id === user.id)
  return membership?.project_role ?? null
}
