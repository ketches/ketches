import { BreadcrumbContext } from "@/contexts/breadcrumb-state"
import { useContext } from "react"

export function useBreadcrumbs() {
    const context = useContext(BreadcrumbContext)
    if (!context) {
        throw new Error("useBreadcrumbs must be used within a BreadcrumbProvider")
    }
    return context
}
