import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface ProjectState {
  hasHydrated: boolean
  activeProjectId: string | null
  activeProjectName: string | null
  activeEnvId: string | null
  activeEnvName: string | null
  setActiveProjectId: (id: string | null) => void
  setActiveEnvId: (id: string | null) => void
  setActiveContext: (projectId: string | null, envId: string | null) => void
  setActiveContextWithNames: (projectId: string | null, projectName: string | null, envId: string | null, envName: string | null) => void
  setHasHydrated: (hydrated: boolean) => void
}

export const useProjectStore = create<ProjectState>()(
  persist(
    (set) => ({
      hasHydrated: false,
      activeProjectId: null,
      activeProjectName: null,
      activeEnvId: null,
      activeEnvName: null,
      setActiveProjectId: (id) => set((state) => ({
        activeProjectId: id,
        activeProjectName: state.activeProjectId === id ? state.activeProjectName : null,
        activeEnvId: state.activeProjectId === id ? state.activeEnvId : null,
        activeEnvName: state.activeProjectId === id ? state.activeEnvName : null,
      })),
      setActiveEnvId: (id) => set({ activeEnvId: id }),
      setActiveContext: (projectId, envId) => set({ activeProjectId: projectId, activeEnvId: envId }),
      setActiveContextWithNames: (projectId, projectName, envId, envName) => set({ activeProjectId: projectId, activeProjectName: projectName, activeEnvId: envId, activeEnvName: envName }),
      setHasHydrated: (hydrated) => set({ hasHydrated: hydrated }),
    }),
    {
      name: 'project-storage',
      partialize: (state) => ({
        activeProjectId: state.activeProjectId,
        activeProjectName: state.activeProjectName,
        activeEnvId: state.activeEnvId,
        activeEnvName: state.activeEnvName,
      }),
      onRehydrateStorage: () => (state) => {
        state?.setHasHydrated(true)
      },
    }
  )
)
