// useProjectRole returns the current user's role in the active project.
// Admin system users always get 'owner'. Returns null if no active project or not loaded.
import { useQuery } from '@tanstack/react-query'
import { projectsApi, type ProjectRole } from '@/api/projects'
import { useAuthStore } from '@/stores/auth'
import { useProjectStore } from '@/stores/project'

export function useProjectRole(): ProjectRole | null {
  const user = useAuthStore((state) => state.user)
  const activeProjectId = useProjectStore((state) => state.activeProjectId)

  // Admin system role gets full owner-equivalent access
  if (user?.role === 'admin') {
    return 'owner'
  }

  const { data } = useQuery({
    queryKey: ['project-members', activeProjectId],
    queryFn: () => projectsApi.listMembers(activeProjectId!),
    enabled: !!activeProjectId && !!user,
    staleTime: 5 * 60 * 1000,
  })

  if (!data || !user) return null

  const membership = data.items.find((m) => m.user_id === user.id)
  return membership?.project_role ?? null
}
