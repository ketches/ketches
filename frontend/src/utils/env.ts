export function getApiBaseUrl(): string {
    // Prefer runtime-injected window variable, fallback to placeholder for build-time replacement
    return (window as any).VITE_API_BASE_URL || 'VITE_API_BASE_URL_PLACEHOLDER';
}
