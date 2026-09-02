import { beginSessionLogout, cancelSessionLogout, completeSessionLogout } from "./auth-session"

export interface LogoutSessionActions {
  requestLogout: () => Promise<void>
  markManualLogout: () => void
  clearQueries: () => void
  clearAuth: () => void
  navigateToLogin: () => void
}

export async function logoutSession(actions: LogoutSessionActions): Promise<void> {
  beginSessionLogout()
  try {
    await actions.requestLogout()
  } catch (error) {
    cancelSessionLogout()
    throw error
  }

  completeSessionLogout()
  actions.markManualLogout()
  actions.clearQueries()
  actions.clearAuth()
  actions.navigateToLogin()
}
