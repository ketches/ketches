// Global module declaration for importing .vue files in TypeScript
declare module '*.vue' {
    import { DefineComponent } from 'vue'
    const component: DefineComponent<{}, {}, any>
    export default component
}

// declare module '@/api/axios.ts' {
//     const api: any
//     export default api
//     export type { ApiResponse } from '@/api/axios.ts'
// }
