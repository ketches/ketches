const AUTH_STORAGE_KEY = "auth-storage"
const CSRF_COOKIE_NAME = "X-Ketches-CSRF"
const CSRF_HEADER_NAME = "X-CSRF-Token"

type PersistedAuthState = {
  state?: {
    isAuthenticated?: boolean
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

function readCookie(name: string): string | null {
  if (typeof document === "undefined") {
    return null
  }

  const encodedName = encodeURIComponent(name)
  const cookies = document.cookie.split(";")
  for (const cookie of cookies) {
    const trimmedCookie = cookie.trim()
    if (!trimmedCookie.startsWith(`${encodedName}=`)) {
      continue
    }
    return decodeURIComponent(trimmedCookie.slice(encodedName.length + 1))
  }

  return null
}

export function hasPersistedAuthSession(): boolean {
  const sessionState = parsePersistedAuthState(getStorageValue(typeof window !== "undefined" ? window.sessionStorage : null))
  if (sessionState?.state?.isAuthenticated) {
    return true
  }

  const localState = parsePersistedAuthState(getStorageValue(typeof window !== "undefined" ? window.localStorage : null))
  return !!localState?.state?.isAuthenticated
}

export function getCSRFToken(): string {
  return readCookie(CSRF_COOKIE_NAME) ?? ""
}

export function shouldAttachCSRF(method?: string): boolean {
  const normalizedMethod = (method ?? "GET").toUpperCase()
  return !["GET", "HEAD", "OPTIONS", "TRACE"].includes(normalizedMethod)
}

export function applyCSRFHeader(headers: Headers, method?: string): Headers {
  if (!shouldAttachCSRF(method)) {
    return headers
  }

  const csrfToken = getCSRFToken()
  if (csrfToken) {
    headers.set(CSRF_HEADER_NAME, csrfToken)
  }

  return headers
}

export async function authenticatedFetch(input: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers)
  applyCSRFHeader(headers, init.method)

  return fetch(input, {
    ...init,
    headers,
    credentials: "include",
  })
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
