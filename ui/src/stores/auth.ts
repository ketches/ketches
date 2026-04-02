import { create } from 'zustand'
import { createJSONStorage, persist } from 'zustand/middleware'
import { clearAuthCookie, clearPersistedAuthState, setAuthCookie } from '@/lib/auth-session'

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
  accessToken: string | null
  refreshToken: string | null
  isAuthenticated: boolean

  setAuth: (user: User, accessToken: string, refreshToken: string) => void
  updateUser: (user: Partial<User>) => void
  updateTokens: (accessToken: string, refreshToken: string) => void
  logout: () => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,

      setAuth: (user, accessToken, refreshToken) =>
        (setAuthCookie(accessToken), set({
          user,
          accessToken,
          refreshToken,
          isAuthenticated: true,
        })),

      updateUser: (user) =>
        set((state) => ({
          user: state.user ? { ...state.user, ...user } : state.user,
        })),

      updateTokens: (accessToken, refreshToken) =>
        (setAuthCookie(accessToken), set({ accessToken, refreshToken })),

      logout: () =>
        (clearAuthCookie(), clearPersistedAuthState(), set({
          user: null,
          accessToken: null,
          refreshToken: null,
          isAuthenticated: false,
        })),
    }),
    {
      name: 'auth-storage',
      storage: createJSONStorage(() => sessionStorage),
    }
  )
)
