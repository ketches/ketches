const AUTH_STORAGE_KEY = "auth-storage"
const AUTH_SESSION_REFRESHED_AT_KEY = "auth-session-refreshed-at"
const CSRF_COOKIE_NAME = "X-Ketches-CSRF"
const CSRF_HEADER_NAME = "X-CSRF-Token"

// Session generations invalidate refreshes that were started before a logout.
let sessionGeneration = 0
let sessionLogoutInProgress = false

export function getSessionGeneration(): number {
  return sessionGeneration
}

export function isSessionLogoutInProgress(): boolean {
  return sessionLogoutInProgress
}

export function beginSessionLogout(): number {
  sessionGeneration += 1
  sessionLogoutInProgress = true
  return sessionGeneration
}

export function completeSessionLogout(): void {
  sessionLogoutInProgress = false
}

export function cancelSessionLogout(): void {
  sessionGeneration += 1
  sessionLogoutInProgress = false
}

type PersistedAuthState = {
  state?: {
    isAuthenticated?: boolean
  }
}

function getWindowStorage(kind: "sessionStorage" | "localStorage"): Storage | null {
  if (typeof window === "undefined") {
    return null
  }

  return window[kind]
}

function getStorageValue(storage: Storage | null, key: string): string | null {
  if (!storage) {
    return null
  }

  try {
    return storage.getItem(key)
  } catch {
    return null
  }
}

function clearStorageValue(storage: Storage | null, key: string): void {
  if (!storage) {
    return
  }

  try {
    storage.removeItem(key)
  } catch {
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
  const sessionState = parsePersistedAuthState(getStorageValue(getWindowStorage("sessionStorage"), AUTH_STORAGE_KEY))
  if (sessionState?.state?.isAuthenticated) {
    return true
  }

  const localState = parsePersistedAuthState(getStorageValue(getWindowStorage("localStorage"), AUTH_STORAGE_KEY))
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

export function getLastSessionRefreshAt(storage: Storage | null = getWindowStorage("sessionStorage")): number {
  const rawValue = getStorageValue(storage, AUTH_SESSION_REFRESHED_AT_KEY)
  const parsedValue = Number(rawValue)

  return Number.isFinite(parsedValue) && parsedValue > 0 ? parsedValue : 0
}

export function markSessionRefreshed(
  storage: Storage | null = getWindowStorage("sessionStorage"),
  now: number = Date.now()
): void {
  if (!storage) {
    return
  }

  try {
    storage.setItem(AUTH_SESSION_REFRESHED_AT_KEY, String(now))
  } catch {
  }
}

export function clearPersistedAuthState(): void {
  clearStorageValue(getWindowStorage("sessionStorage"), AUTH_STORAGE_KEY)
  clearStorageValue(getWindowStorage("localStorage"), AUTH_STORAGE_KEY)
  clearStorageValue(getWindowStorage("sessionStorage"), AUTH_SESSION_REFRESHED_AT_KEY)
}
