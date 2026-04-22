import { refreshSession } from "@/api/client"
import { getLastSessionRefreshAt } from "@/lib/auth-session"
import { useAuthStore } from "@/stores/auth"
import * as React from "react"

export const SESSION_REFRESH_INTERVAL_MS = 50 * 60 * 1000
function shouldRefreshSession(now: number = Date.now()): boolean {
  return now - getLastSessionRefreshAt() >= SESSION_REFRESH_INTERVAL_MS
}

export function AuthSessionManager() {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated)

  const runRefresh = React.useEffectEvent(async () => {
    if (!isAuthenticated) {
      return
    }

    try {
      await refreshSession()
    } catch {
    }
  })

  React.useEffect(() => {
    if (!isAuthenticated || typeof window === "undefined") {
      return
    }

    if (shouldRefreshSession()) {
      void runRefresh()
    }

    const intervalId = window.setInterval(() => {
      void runRefresh()
    }, SESSION_REFRESH_INTERVAL_MS)

    const handleFocus = () => {
      if (shouldRefreshSession()) {
        void runRefresh()
      }
    }

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible" && shouldRefreshSession()) {
        void runRefresh()
      }
    }

    window.addEventListener("focus", handleFocus)
    document.addEventListener("visibilitychange", handleVisibilityChange)

    return () => {
      window.clearInterval(intervalId)
      window.removeEventListener("focus", handleFocus)
      document.removeEventListener("visibilitychange", handleVisibilityChange)
    }
  }, [isAuthenticated])

  return null
}
