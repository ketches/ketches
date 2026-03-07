import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface ProjectState {
  hasHydrated: boolean
  activeProjectId: string | null
  activeEnvId: string | null
  setActiveProjectId: (id: string | null) => void
  setActiveEnvId: (id: string | null) => void
  setActiveContext: (projectId: string | null, envId: string | null) => void
  setHasHydrated: (hydrated: boolean) => void
}

export const useProjectStore = create<ProjectState>()(
  persist(
    (set) => ({
      hasHydrated: false,
      activeProjectId: null,
      activeEnvId: null,
      setActiveProjectId: (id) => set((state) => ({
        activeProjectId: id,
        activeEnvId: state.activeProjectId === id ? state.activeEnvId : null,
      })),
      setActiveEnvId: (id) => set({ activeEnvId: id }),
      setActiveContext: (projectId, envId) => set({ activeProjectId: projectId, activeEnvId: envId }),
      setHasHydrated: (hydrated) => set({ hasHydrated: hydrated }),
    }),
    {
      name: 'project-storage',
      partialize: (state) => ({
        activeProjectId: state.activeProjectId,
        activeEnvId: state.activeEnvId,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true)
      },
    }
  )
)
