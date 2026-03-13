import { getPostLoginTarget } from "@/lib/auth-redirect"
import { useAuthStore } from "@/stores/auth"
import { Navigate, useLocation } from "react-router-dom"

interface AuthRouteProps {
  children: React.ReactNode
}

export function AuthRoute({ children }: AuthRouteProps) {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)
  const location = useLocation()

  if (isAuthenticated) {
    return <Navigate to={getPostLoginTarget(location.search)} replace />
  }

  return <>{children}</>
}
