const CACHE_KEY = 'resource-store'

export function saveResourceRefCache(data: {
    lastActiveProjectID?: string
    lastActiveEnvID?: string
    lastActiveAppID?: string
}) {
    localStorage.setItem(CACHE_KEY, JSON.stringify(data))
}

export function loadResourceRefCache(): {
    lastActiveProjectID?: string
    lastActiveEnvID?: string
    lastActiveAppID?: string
} {
    try {
        return JSON.parse(localStorage.getItem(CACHE_KEY) || '{}')
    } catch {
        return {}
    }
}
