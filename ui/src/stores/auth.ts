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

  setAuth: (user: User) => void
  updateUser: (user: Partial<User>) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      isAuthenticated: false,

      setAuth: (user) =>
        set({
          user,
          isAuthenticated: true,
        }),

      updateUser: (user) =>
        set((state) => ({
          user: state.user ? { ...state.user, ...user } : state.user,
        })),

      logout: () =>
        (clearPersistedAuthState(), set({
          user: null,
          isAuthenticated: false,
        })),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => sessionStorage),
    }
  )
)
