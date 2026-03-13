import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'
import { buildLoginHref, getCurrentRelativePath } from '@/lib/auth-redirect'

interface ProtectedRouteProps {
  children: React.ReactNode
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const location = useLocation()

  if (!isAuthenticated) {
    return <Navigate to={buildLoginHref(getCurrentRelativePath(location))} replace />
  }

  return <>{children}</>
}
