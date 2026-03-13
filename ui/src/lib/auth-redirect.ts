const LOGIN_PATH = "/login"
const SIGNUP_PATH = "/signup"
const APP_ORIGIN = "https://ketches.local"
const MANUAL_LOGOUT_MARKER_KEY = "manual-logout-at"
export const MANUAL_LOGOUT_MARKER_TTL_MS = 10_000

type StorageLike = Pick<Storage, "getItem" | "setItem" | "removeItem">

function isAuthPage(pathname: string): boolean {
  return pathname === LOGIN_PATH || pathname === SIGNUP_PATH
}

function parseRelativePath(relativePath: string): URL | null {
  try {
    return new URL(relativePath, APP_ORIGIN)
  } catch {
    return null
  }
}

export function getCurrentRelativePath(
  locationLike: Pick<Location, "pathname" | "search" | "hash">
): string {
  return `${locationLike.pathname || "/"}${locationLike.search || ""}${locationLike.hash || ""}`
}

export function isSafeRedirectTarget(target: string | null | undefined): target is string {
  if (!target || !target.startsWith("/") || target.startsWith("//")) {
    return false
  }

  const parsed = parseRelativePath(target)
  return parsed ? !isAuthPage(parsed.pathname) : false
}

export function buildLoginHref(currentPath: string | null | undefined): string {
  if (!isSafeRedirectTarget(currentPath)) {
    return LOGIN_PATH
  }

  return `${LOGIN_PATH}?redirect=${encodeURIComponent(currentPath)}`
}

export function getSafeRedirectTarget(search: string): string | null {
  const redirect = new URLSearchParams(search).get("redirect")

  if (!isSafeRedirectTarget(redirect)) {
    return null
  }

  return redirect
}

export function getPostLoginTarget(search: string): string {
  return getSafeRedirectTarget(search) ?? "/"
}

export function markManualLogout(
  storage: StorageLike = sessionStorage,
  now: number = Date.now()
): void {
  storage.setItem(MANUAL_LOGOUT_MARKER_KEY, String(now))
}

export function clearManualLogoutMarker(storage: StorageLike = sessionStorage): void {
  storage.removeItem(MANUAL_LOGOUT_MARKER_KEY)
}

function hasRecentManualLogoutMarker(
  storage: StorageLike = sessionStorage,
  now: number = Date.now()
): boolean {
  const rawValue = storage.getItem(MANUAL_LOGOUT_MARKER_KEY)

  if (!rawValue) {
    return false
  }

  const marker = Number(rawValue)

  if (!Number.isFinite(marker) || now - marker > MANUAL_LOGOUT_MARKER_TTL_MS) {
    storage.removeItem(MANUAL_LOGOUT_MARKER_KEY)
    return false
  }

  return true
}

export function buildUnauthenticatedLoginHref(
  currentPath: string | null | undefined,
  storage: StorageLike = sessionStorage,
  now: number = Date.now()
): string {
  if (currentPath) {
    const currentUrl = parseRelativePath(currentPath)
    const currentRedirect = currentUrl ? getSafeRedirectTarget(currentUrl.search) : null

    if (currentUrl?.pathname === LOGIN_PATH && currentRedirect) {
      return currentPath
    }
  }

  if (hasRecentManualLogoutMarker(storage, now)) {
    return LOGIN_PATH
  }

  return buildLoginHref(currentPath)
}
