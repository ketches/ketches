const AUTH_STORAGE_KEY = "auth-storage"
const AUTH_COOKIE_NAME = "X-Ketches-Token"

type PersistedAuthState = {
  state?: {
    accessToken?: string | null
    refreshToken?: string | null
  }
}

function getStorageValue(storage: Storage | null): string | null {
  if (!storage) {
    return null
  }

  try {
    return storage.getItem(AUTH_STORAGE_KEY)
  } catch {
    return null
  }
}

function parsePersistedAuthState(rawValue: string | null): PersistedAuthState | null {
  if (!rawValue) {
    return null
  }

  try {
    return JSON.parse(rawValue) as PersistedAuthState
  } catch {
    return null
  }
}

function secureCookieSuffix(): string {
  if (typeof window !== "undefined" && window.location.protocol === "https:") {
    return "; Secure"
  }
  return ""
}

export function getStoredAccessToken(): string {
  const sessionState = parsePersistedAuthState(getStorageValue(typeof window !== "undefined" ? window.sessionStorage : null))
  if (sessionState?.state?.accessToken) {
    return sessionState.state.accessToken
  }

  const localState = parsePersistedAuthState(getStorageValue(typeof window !== "undefined" ? window.localStorage : null))
  return localState?.state?.accessToken ?? ""
}

export function setAuthCookie(token: string | null | undefined): void {
  if (typeof document === "undefined") {
    return
  }

  if (!token) {
    clearAuthCookie()
    return
  }

  document.cookie = `${AUTH_COOKIE_NAME}=${encodeURIComponent(token)}; path=/; SameSite=Strict${secureCookieSuffix()}`
}

export function syncAuthCookie(): string {
  const token = getStoredAccessToken()
  setAuthCookie(token)
  return token
}

export function clearAuthCookie(): void {
  if (typeof document === "undefined") {
    return
  }

  const suffix = `Max-Age=0; SameSite=Strict${secureCookieSuffix()}`
  document.cookie = `${AUTH_COOKIE_NAME}=; path=/; ${suffix}`
  document.cookie = `${AUTH_COOKIE_NAME}=; path=/api; ${suffix}`
  document.cookie = `${AUTH_COOKIE_NAME}=; path=/forward; ${suffix}`
}

export function clearPersistedAuthState(): void {
  if (typeof window === "undefined") {
    return
  }

  try {
    window.sessionStorage.removeItem(AUTH_STORAGE_KEY)
  } catch {
  }

  try {
    window.localStorage.removeItem(AUTH_STORAGE_KEY)
  } catch {
  }
}
