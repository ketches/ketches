export function getApiBaseUrl(): string {
    return (window as any).VITE_API_BASE_URL || '/api/v1';
}
