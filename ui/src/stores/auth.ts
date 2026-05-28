import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { clearPersistedAuthState } from '@/lib/auth-session'

export interface User {
  id: string
  username: string
  email: string
  role: string
  fullname?: string
  bio?: string
}

interface AuthState {
  user: User | null
  isAuthenticated: boolean
  hasCheckedSession: boolean
  isRestoringSession: boolean

  setAuth: (user: User) => void
  updateUser: (user: Partial<User>) => void
  logout: () => void
  markSessionRestoreStarted: () => void
  markSessionRestoreFinished: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,
      hasCheckedSession: false,
      isRestoringSession: false,

      setAuth: (user) =>
        set({
          user,
          hasCheckedSession: true,
          isAuthenticated: true,
          isRestoringSession: false,
        }),

      updateUser: (user) =>
        set((state) => ({
          user: state.user ? { ...state.user, ...user } : state.user,
        })),

      logout: () =>
        (clearPersistedAuthState(), set({
          hasCheckedSession: true,
          isRestoringSession: false,
          user: null,
          isAuthenticated: false,
        })),

      markSessionRestoreStarted: () =>
        set({
          hasCheckedSession: false,
          isRestoringSession: true,
        }),

      markSessionRestoreFinished: () =>
        set({
          hasCheckedSession: true,
          isRestoringSession: false,
        }),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => sessionStorage),
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)
