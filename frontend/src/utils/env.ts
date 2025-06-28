export function getApiBaseUrl(): string {
    // 优先读取 import.meta.env（本地开发），否则读取 window（生产注入），最后兜底
    return (import.meta.env && import.meta.env.VITE_API_BASE_URL) || (window as any).VITE_API_BASE_URL || '/api/v1';
}
