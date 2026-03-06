import { useAuthStore } from "@/stores/auth"
import * as React from "react"
import { useParams } from "react-router-dom"

// ForwardPage acts as a client-side relay for gateway quick access.
// It reads the JWT from the Zustand store and immediately redirects
// the browser to the backend proxy URL, keeping the token out of
// any shareable link the user might copy from the address bar.
export function ForwardPage() {
  const { gatewayID } = useParams<{ gatewayID: string }>()
  const accessToken = useAuthStore((state) => state.accessToken)

  React.useEffect(() => {
    if (!gatewayID || !accessToken) return
    const target = `/api/v1/gateways/${gatewayID}/proxy/?token=${encodeURIComponent(accessToken)}`
    window.location.replace(target)
  }, [gatewayID, accessToken])

  return (
    <div className="flex h-screen items-center justify-center text-sm text-muted-foreground">
      Redirecting…
    </div>
  )
}
