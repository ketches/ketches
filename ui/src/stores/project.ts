import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface ProjectState {
  activeProjectId: string | null
  activeEnvId: string | null
  setActiveProjectId: (id: string | null) => void
  setActiveEnvId: (id: string | null) => void
}

export const useProjectStore = create<ProjectState>()(
  persist(
    (set) => ({
      activeProjectId: null,
      activeEnvId: null,
      setActiveProjectId: (id) => set({ activeProjectId: id, activeEnvId: null }),
      setActiveEnvId: (id) => set({ activeEnvId: id }),
    }),
    {
      name: 'project-storage',
    }
  )
)
