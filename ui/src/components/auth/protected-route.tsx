import { Navigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'
import { buildLoginHref, getCurrentRelativePath } from '@/lib/auth-redirect'

interface ProtectedRouteProps {
  children: React.ReactNode
}

export function ProtectedRoute({ children }: ProtectedRouteProps) {
  const hasCheckedSession = useAuthStore((state) => state.hasCheckedSession)
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const isRestoringSession = useAuthStore((state) => state.isRestoringSession)
  const location = useLocation()

  if (!isAuthenticated) {
    if (!hasCheckedSession || isRestoringSession) {
      return (
        <div className="flex min-h-0 flex-1 items-center justify-center text-sm text-muted-foreground">
          Loading page...
        </div>
      )
    }

    return <Navigate to={buildLoginHref(getCurrentRelativePath(location))} replace />
  }

  return <>{children}</>
}
